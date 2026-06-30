package collector

import (
	"context"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/wwswwsuns/ztelem/internal/buffer"
	"github.com/wwswwsuns/ztelem/internal/config"
	"github.com/wwswwsuns/ztelem/internal/database"
	"github.com/wwswwsuns/ztelem/internal/parser"
	"github.com/wwswwsuns/ztelem/proto/zte_dialout"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/peer"
)

// ConnectionInfo 连接信息
type ConnectionInfo struct {
	RemoteAddr    string
	ConnectedAt   time.Time
	LastDataTime  time.Time
	DataCount     int64
	IsActive      bool
}

// SimpleCollector 简化的采集器实现
type SimpleCollector struct {
	proto.UnimplementedZtedialoutServiceServer
	logger        *logrus.Logger
	db            *database.Database
	parser        *parser.TelemetryParser
	bufferManager *buffer.FixedBufferManager
	server        *grpc.Server
	listener      net.Listener
	serverConfig  config.ServerConfig
	
	// 连接监控
	connections     map[string]*ConnectionInfo
	connectionsMux  sync.RWMutex
	activeConnCount int64
	
	// 数据流超时监控
	dataTimeout     time.Duration
	shutdownChan    chan struct{}
	monitoringDone  chan struct{}
}

// NewSimpleCollector 创建简化的采集器
func NewSimpleCollector(logger *logrus.Logger, bufferManager *buffer.FixedBufferManager, serverConfig config.ServerConfig) *SimpleCollector {
	return &SimpleCollector{
		logger:         logger,
		parser:         parser.NewTelemetryParser(logger),
		bufferManager:  bufferManager,
		serverConfig:   serverConfig,
		connections:    make(map[string]*ConnectionInfo),
		dataTimeout:    15 * time.Minute, // 15分钟数据超时
		shutdownChan:   make(chan struct{}),
		monitoringDone: make(chan struct{}),
	}
}

// Start 启动采集服务
func (c *SimpleCollector) Start(port int) error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return fmt.Errorf("监听端口失败: %v", err)
	}
	c.listener = lis

	// 配置gRPC服务器选项
	opts := []grpc.ServerOption{
		// Keepalive配置
		grpc.KeepaliveParams(keepalive.ServerParameters{
			Time:    c.serverConfig.KeepaliveTime,
			Timeout: c.serverConfig.KeepaliveTimeout,
		}),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             10 * time.Second, // 最小keepalive时间
			PermitWithoutStream: true,             // 允许没有活跃流的keepalive
		}),
		// 消息大小限制
		grpc.MaxRecvMsgSize(c.serverConfig.MaxRecvMsgSize),
		grpc.MaxSendMsgSize(c.serverConfig.MaxSendMsgSize),
		grpc.MaxConcurrentStreams(c.serverConfig.MaxConcurrentStreams),
	}

	c.server = grpc.NewServer(opts...)
	proto.RegisterZtedialoutServiceServer(c.server, c)

	// 启动连接监控
	go c.startConnectionMonitor()

	c.logger.Infof("gRPC服务启动在端口 %d，配置: KeepAlive=%v, Timeout=%v, MaxStreams=%d", 
		port, c.serverConfig.KeepaliveTime, c.serverConfig.KeepaliveTimeout, c.serverConfig.MaxConcurrentStreams)
	
	return c.server.Serve(lis)
}

// Stop 停止采集服务
func (c *SimpleCollector) Stop() {
	c.logger.Info("开始关闭采集服务...")
	
	// 停止监控协程
	close(c.shutdownChan)
	
	// 等待监控协程结束
	select {
	case <-c.monitoringDone:
		c.logger.Info("连接监控已停止")
	case <-time.After(5 * time.Second):
		c.logger.Warn("等待连接监控停止超时")
	}
	
	if c.bufferManager != nil {
		c.bufferManager.Stop()
	}
	if c.server != nil {
		c.server.GracefulStop()
	}
	if c.listener != nil {
		c.listener.Close()
	}
	
	c.logger.Info("采集服务已停止")
}

// Publish 实现gRPC服务接口
func (c *SimpleCollector) Publish(stream grpc.BidiStreamingServer[proto.PublishArgs, proto.PublishArgs]) error {
	// 获取客户端地址
	var remoteAddr string
	if p, ok := peer.FromContext(stream.Context()); ok {
		remoteAddr = p.Addr.String()
	} else {
		remoteAddr = "unknown"
	}
	
	// 注册连接
	connID := c.registerConnection(remoteAddr)
	defer c.unregisterConnection(connID)
	
	c.logger.Infof("新的设备连接建立: %s (总连接数: %d)", remoteAddr, atomic.LoadInt64(&c.activeConnCount))

	// 设置数据接收超时检测
	ctx, cancel := context.WithTimeout(stream.Context(), c.dataTimeout)
	defer cancel()
	
	// 在单独的goroutine中处理超时
	go func(connectionID string) {
		select {
		case <-ctx.Done():
			if ctx.Err() == context.DeadlineExceeded {
				c.logger.Errorf("连接 %s 数据接收超时 (%v)，将断开连接", remoteAddr, c.dataTimeout)
				// 这里可以添加重启逻辑或其他处理
				c.handleConnectionTimeout(remoteAddr)
			}
		case <-stream.Context().Done():
			// 正常连接关闭
		}
	}(connID)

	for {
		req, err := stream.Recv()
		if err != nil {
			c.logger.WithError(err).Errorf("连接 %s 接收数据流错误", remoteAddr)
			return err
		}

		// 更新连接的最后数据时间
		c.updateConnectionActivity(connID)

		// 处理接收到的数据
		if err := c.processPublishArgs(req); err != nil {
			c.logger.WithError(err).Error("处理数据失败")
			continue
		}

		// 发送响应
		response := &proto.PublishArgs{
			ReqId:  req.ReqId,
			Errors: "",
		}

		if err := stream.Send(response); err != nil {
			c.logger.WithError(err).Errorf("连接 %s 发送响应失败", remoteAddr)
			return err
		}
		
		// 重置超时计时器
		cancel()
		ctx, cancel = context.WithTimeout(stream.Context(), c.dataTimeout)
	}
}

// processPublishArgs 处理发布参数
func (c *SimpleCollector) processPublishArgs(req *proto.PublishArgs) error {
	c.logger.Debugf("处理请求ID: %d", req.ReqId)

	// 解析GPB数据
	data := req.GetData()
	if len(data) > 0 {
		c.logger.Debugf("收到GPB数据，长度: %d bytes", len(data))
		
		// 使用解析器解析telemetry数据
		result, err := c.parser.ParseTelemetryData(data)
		if err != nil {
			c.logger.WithError(err).Error("解析telemetry数据失败")
			return err
		}

		c.logger.Debugf("解析成功: system_id=%s, sensor_path=%s, platform_metrics=%d, interface_metrics=%d, subinterface_metrics=%d, alarm_reports=%d, notifications=%d",
			result.SystemID, result.SensorPath, len(result.PlatformMetrics), len(result.InterfaceMetrics), len(result.SubinterfaceMetrics), len(result.AlarmReportMetrics), len(result.NotificationReportMetrics))

		// 特别记录告警相关的sensor_path
		if result.SensorPath == "alm:current-alarm-report" || result.SensorPath == "alm:notification-report" {
			c.logger.Infof("🚨 检测到告警相关数据: sensor_path=%s, system_id=%s, alarm_reports=%d, notifications=%d", 
				result.SensorPath, result.SystemID, len(result.AlarmReportMetrics), len(result.NotificationReportMetrics))
		}

		// 添加到缓冲区
		if len(result.PlatformMetrics) > 0 {
			if err := c.bufferManager.AddPlatformMetrics(result.PlatformMetrics); err != nil {
				c.logger.WithError(err).Error("添加平台指标数据到缓冲区失败")
				return fmt.Errorf("添加平台指标数据到缓冲区失败: %v", err)
			}
		}

		if len(result.InterfaceMetrics) > 0 {
			if err := c.bufferManager.AddInterfaceMetrics(result.InterfaceMetrics); err != nil {
				c.logger.WithError(err).Error("添加接口指标数据到缓冲区失败")
				return fmt.Errorf("添加接口指标数据到缓冲区失败: %v", err)
			}
		}

		if len(result.SubinterfaceMetrics) > 0 {
			if err := c.bufferManager.AddSubinterfaceMetrics(result.SubinterfaceMetrics); err != nil {
				c.logger.WithError(err).Error("添加子接口指标数据到缓冲区失败")
				return fmt.Errorf("添加子接口指标数据到缓冲区失败: %v", err)
			}
		}

		if len(result.AlarmReportMetrics) > 0 {
			c.logger.Infof("🔥 添加 %d 条告警上报数据到缓冲区", len(result.AlarmReportMetrics))
			if err := c.bufferManager.AddAlarmReportMetrics(result.AlarmReportMetrics); err != nil {
				c.logger.WithError(err).Error("添加告警上报数据到缓冲区失败")
				return fmt.Errorf("添加告警上报数据到缓冲区失败: %v", err)
			}
			c.logger.Infof("✅ 成功添加告警上报数据到缓冲区")
		}

		if len(result.NotificationReportMetrics) > 0 {
			c.logger.Infof("🔔 添加 %d 条通知上报数据到缓冲区", len(result.NotificationReportMetrics))
			if err := c.bufferManager.AddNotificationReportMetrics(result.NotificationReportMetrics); err != nil {
				c.logger.WithError(err).Error("添加通知上报数据到缓冲区失败")
				return fmt.Errorf("添加通知上报数据到缓冲区失败: %v", err)
			}
			c.logger.Infof("✅ 成功添加通知上报数据到缓冲区")
		}
	}

	// 处理JSON数据（如果有）
	jsonData := req.GetJsonData()
	if jsonData != "" {
		c.logger.Debugf("收到JSON数据: %s", jsonData)
		// TODO: 实现JSON数据解析（GPBKV格式）
	}

	return nil
}

// registerConnection 注册新连接
func (c *SimpleCollector) registerConnection(remoteAddr string) string {
	c.connectionsMux.Lock()
	defer c.connectionsMux.Unlock()
	
	connID := fmt.Sprintf("%s-%d", remoteAddr, time.Now().UnixNano())
	c.connections[connID] = &ConnectionInfo{
		RemoteAddr:   remoteAddr,
		ConnectedAt:  time.Now(),
		LastDataTime: time.Now(),
		DataCount:    0,
		IsActive:     true,
	}
	
	atomic.AddInt64(&c.activeConnCount, 1)
	return connID
}

// unregisterConnection 注销连接
func (c *SimpleCollector) unregisterConnection(connID string) {
	c.connectionsMux.Lock()
	defer c.connectionsMux.Unlock()
	
	if conn, exists := c.connections[connID]; exists {
		conn.IsActive = false
		duration := time.Since(conn.ConnectedAt)
		c.logger.Infof("连接 %s 断开，持续时间: %v，处理数据: %d 条", 
			conn.RemoteAddr, duration, conn.DataCount)
		delete(c.connections, connID)
		atomic.AddInt64(&c.activeConnCount, -1)
	}
}

// updateConnectionActivity 更新连接活动
func (c *SimpleCollector) updateConnectionActivity(connID string) {
	c.connectionsMux.Lock()
	defer c.connectionsMux.Unlock()
	
	if conn, exists := c.connections[connID]; exists {
		conn.LastDataTime = time.Now()
		atomic.AddInt64(&conn.DataCount, 1)
	}
}

// startConnectionMonitor 启动连接监控
func (c *SimpleCollector) startConnectionMonitor() {
	ticker := time.NewTicker(1 * time.Minute) // 每分钟检查一次
	defer ticker.Stop()
	defer close(c.monitoringDone)
	
	c.logger.Info("连接监控已启动")
	
	for {
		select {
		case <-c.shutdownChan:
			c.logger.Info("收到关闭信号，停止连接监控")
			return
		case <-ticker.C:
			c.checkConnectionHealth()
		}
	}
}

// checkConnectionHealth 检查连接健康状态
func (c *SimpleCollector) checkConnectionHealth() {
	totalConnections, activeConnections, staleConnections, _ := c.computeConnectionSnapshot()

	if totalConnections > 0 {
		c.logger.Infof("连接健康检查: 总连接=%d, 活跃连接=%d, 僵尸连接=%d", 
			totalConnections, activeConnections, staleConnections)
	}

	// 如果僵尸连接过多，记录警告
	if totalConnections > 0 && float64(staleConnections)/float64(totalConnections) > 0.5 {
		c.logger.Errorf("警告: 僵尸连接比例过高 (%d/%d = %.1f%%)，可能需要重启服务", 
			staleConnections, totalConnections, float64(staleConnections)/float64(totalConnections)*100)
	}
}

// computeConnectionSnapshot 计算连接快照（原子）
func (c *SimpleCollector) computeConnectionSnapshot() (total, active, stale int, totalDataCount int64) {
	c.connectionsMux.RLock()
	defer c.connectionsMux.RUnlock()
	
	now := time.Now()
	total = len(c.connections)
	for _, conn := range c.connections {
		totalDataCount += conn.DataCount
		if now.Sub(conn.LastDataTime) <= c.dataTimeout {
			active++
		} else {
			stale++
		}
	}
	return
}

// handleConnectionTimeout 处理连接超时
func (c *SimpleCollector) handleConnectionTimeout(remoteAddr string) {
	c.logger.Errorf("连接 %s 超时，建议检查网络连接和设备状态", remoteAddr)
	
	// 这里可以添加更多的处理逻辑，比如：
	// 1. 发送告警通知
	// 2. 记录超时事件到数据库
	// 3. 如果超时连接过多，可以考虑重启服务
	
	c.connectionsMux.RLock()
	timeoutCount := 0
	totalCount := len(c.connections)
	for _, conn := range c.connections {
		if time.Since(conn.LastDataTime) > c.dataTimeout {
			timeoutCount++
		}
	}
	c.connectionsMux.RUnlock()
	
	// 如果超过一半的连接都超时，建议重启
	if totalCount > 0 && float64(timeoutCount)/float64(totalCount) > 0.5 {
		c.logger.Errorf("严重警告: 超过50%%的连接超时 (%d/%d)，强烈建议重启服务！", timeoutCount, totalCount)
	}
}

// GetConnectionStats 获取连接统计信息
func (c *SimpleCollector) GetConnectionStats() map[string]interface{} {
	total, active, stale, totalDataCount := c.computeConnectionSnapshot()

	return map[string]interface{}{
		"total_connections":   total,
		"active_connections":  active,
		"stale_connections":   stale,
		"total_data_count":    totalDataCount,
		"data_timeout":        c.dataTimeout.String(),
	}
}
