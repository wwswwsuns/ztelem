package collector

import (
	"fmt"
	"net"

	"github.com/wwswwsuns/ztelem/internal/buffer"
	"github.com/wwswwsuns/ztelem/internal/database"
	"github.com/wwswwsuns/ztelem/internal/parser"
	"github.com/wwswwsuns/ztelem/proto/zte_dialout"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

// SimpleCollector 简化的采集器实现
type SimpleCollector struct {
	proto.UnimplementedZtedialoutServiceServer
	logger        *logrus.Logger
	db            *database.Database
	parser        *parser.TelemetryParser
	bufferManager *buffer.FixedBufferManager
	server        *grpc.Server
	listener      net.Listener
}

// NewSimpleCollector 创建简化的采集器
func NewSimpleCollector(logger *logrus.Logger, bufferManager *buffer.FixedBufferManager) *SimpleCollector {
	return &SimpleCollector{
		logger:        logger,
		parser:        parser.NewTelemetryParser(logger),
		bufferManager: bufferManager,
	}
}

// Start 启动采集服务
func (c *SimpleCollector) Start(port int) error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return fmt.Errorf("监听端口失败: %v", err)
	}
	c.listener = lis

	c.server = grpc.NewServer()
	proto.RegisterZtedialoutServiceServer(c.server, c)

	c.logger.Infof("gRPC服务启动在端口 %d", port)
	return c.server.Serve(lis)
}

// Stop 停止采集服务
func (c *SimpleCollector) Stop() {
	if c.bufferManager != nil {
		c.bufferManager.Stop()
	}
	if c.server != nil {
		c.server.GracefulStop()
	}
	if c.listener != nil {
		c.listener.Close()
	}
}

// Publish 实现gRPC服务接口
func (c *SimpleCollector) Publish(stream grpc.BidiStreamingServer[proto.PublishArgs, proto.PublishArgs]) error {
	c.logger.Info("新的设备连接建立")

	for {
		req, err := stream.Recv()
		if err != nil {
			c.logger.WithError(err).Error("接收数据流错误")
			return err
		}

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
			c.logger.WithError(err).Error("发送响应失败")
			return err
		}
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