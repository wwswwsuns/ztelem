package collector

import (
	"fmt"
	"net"
	"time"

	"telemetry-collector/internal/buffer"
	"telemetry-collector/internal/config"
	"telemetry-collector/internal/database"
	"telemetry-collector/internal/parser"
	"telemetry-collector/proto/zte_dialout"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

// SimpleCollector 简化的采集器实现
type SimpleCollector struct {
	proto.UnimplementedZtedialoutServiceServer
	logger        *logrus.Logger
	db            *database.ExtendedDB
	parser        *parser.TelemetryParser
	bufferManager *buffer.ExtendedBufferManager
	server        *grpc.Server
	listener      net.Listener
}

// NewSimpleCollector 创建简化的采集器
func NewSimpleCollector(logger *logrus.Logger, db *database.ExtendedDB) *SimpleCollector {
	// 使用默认配置
	bufferConfig := config.BufferConfig{
		MaxSize:        100000,
		FlushThreshold: 15000,
		FlushInterval:  15 * time.Second,
		BatchSize:      2000,
		PlatformBufferSize:     30000,
		InterfaceBufferSize:    40000,
		SubinterfaceBufferSize: 30000,
	}
	
	writerConfig := config.DatabaseWriterConfig{
		ParallelWriters:         5,
		PlatformWriterCount:     2,
		InterfaceWriterCount:    2,
		SubinterfaceWriterCount: 1,
		MaxBatchSize:           2000,
		BatchTimeout:           30 * time.Second,
		RetryAttempts:          3,
		RetryDelay:             time.Second,
	}
	
	bufferManager := buffer.NewExtendedBufferManager(bufferConfig, writerConfig, logger, db)
	
	return &SimpleCollector{
		logger:        logger,
		db:            db,
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

		c.logger.Debugf("解析成功: system_id=%s, sensor_path=%s, platform_metrics=%d, interface_metrics=%d, subinterface_metrics=%d",
			result.SystemID, result.SensorPath, len(result.PlatformMetrics), len(result.InterfaceMetrics), len(result.SubinterfaceMetrics))

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
	}

	// 处理JSON数据（如果有）
	jsonData := req.GetJsonData()
	if jsonData != "" {
		c.logger.Debugf("收到JSON数据: %s", jsonData)
		// TODO: 实现JSON数据解析（GPBKV格式）
	}

	return nil
}