package main

import (
	"database/sql"
	"flag"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"syscall"
	"time"

	"github.com/wwswwsuns/ztelem/internal/buffer"
	"github.com/wwswwsuns/ztelem/internal/collector"
	"github.com/wwswwsuns/ztelem/internal/config"
	"github.com/wwswwsuns/ztelem/internal/database"

	"github.com/sirupsen/logrus"
)

var (
	configFile = flag.String("config", "production-config-optimized.yaml", "配置文件路径")
	debugMode  = flag.Bool("debug", false, "启用调试模式")
	port       = flag.Int("port", 0, "gRPC服务端口（覆盖配置文件）")
)

func main() {
	flag.Parse()

	// 加载扩展配置
	cfg, err := config.LoadConfig(*configFile)
	if err != nil {
		logrus.WithError(err).Fatal("加载配置失败")
	}

	// 应用性能配置
	applyPerformanceConfig(cfg.Performance)

	// 初始化扩展日志
	log := initializeLogger(cfg.Logging, *debugMode)
	log.Info("启动Telemetry数据采集器 - 扩展版本")

	// 打印配置信息
	printConfigSummary(log, cfg)

	// 初始化扩展数据库连接
	db, err := database.NewExtendedConnection(cfg.Database)
	if err != nil {
		log.WithError(err).Fatal("数据库连接失败")
	}
	defer db.Close()

	// 测试数据库连接
	if err := db.TestConnection(); err != nil {
		log.WithError(err).Fatal("数据库连接测试失败")
	}

	// 打印数据库连接池状态
	stats := db.GetStats()
	log.Infof("数据库连接池状态: OpenConnections=%d, InUse=%d, Idle=%d", 
		stats.OpenConnections, stats.InUse, stats.Idle)

	// 创建扩展缓冲区管理器
	bufferManager := buffer.NewExtendedBufferManager(
		cfg.Buffer, 
		cfg.DatabaseWriter, 
		log, 
		db,
	)

	// 创建采集器
	telemetryCollector := collector.NewSimpleCollector(log, db)

	// 启动监控服务（如果启用）
	if cfg.Monitoring.Enabled {
		go startMonitoringService(cfg.Monitoring, log, bufferManager, db)
	}

	// 优雅关闭处理
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 确定服务端口
	serverPort := cfg.Server.Port
	if *port != 0 {
		serverPort = *port
	}

	go func() {
		log.Infof("启动扩展采集器，监听端口: %d", serverPort)
		log.Infof("服务器配置: MaxRecvMsgSize=%dMB, MaxConcurrentStreams=%d", 
			cfg.Server.MaxRecvMsgSize/(1024*1024), cfg.Server.MaxConcurrentStreams)
		
		if err := telemetryCollector.Start(serverPort); err != nil {
			log.WithError(err).Error("采集器启动错误")
		}
	}()

	// 启动定期状态报告
	go startStatusReporter(log, bufferManager, db, cfg.Monitoring.MetricsInterval)

	// 等待关闭信号
	<-sigChan
	log.Info("收到关闭信号，正在优雅关闭...")

	// 停止采集器
	telemetryCollector.Stop()

	// 停止缓冲区管理器
	bufferManager.Stop()

	// 等待一段时间让服务完全停止
	time.Sleep(3 * time.Second)

	log.Info("程序已关闭")
}

// applyPerformanceConfig 应用性能配置
func applyPerformanceConfig(perfConfig config.PerformanceConfig) {
	// 设置最大CPU核数
	if perfConfig.MaxProcs > 0 {
		runtime.GOMAXPROCS(perfConfig.MaxProcs)
		logrus.Infof("设置GOMAXPROCS为: %d", perfConfig.MaxProcs)
	}

	// 设置GC目标百分比
	if perfConfig.GCPercent > 0 {
		debug.SetGCPercent(perfConfig.GCPercent)
		logrus.Infof("设置GC目标百分比为: %d", perfConfig.GCPercent)
	}
}

// initializeLogger 初始化扩展日志
func initializeLogger(logConfig config.LoggingConfig, debugMode bool) *logrus.Logger {
	log := logrus.New()

	// 设置日志级别
	if debugMode {
		log.SetLevel(logrus.DebugLevel)
	} else {
		level, err := logrus.ParseLevel(logConfig.Level)
		if err != nil {
			log.SetLevel(logrus.InfoLevel)
		} else {
			log.SetLevel(level)
		}
	}

	// 设置日志格式
	if logConfig.Format == "json" {
		log.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: "2006-01-02 15:04:05",
		})
	} else {
		log.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: "2006-01-02 15:04:05",
		})
	}

	// 设置输出
	if logConfig.Output == "file" && logConfig.FilePath != "" {
		// 创建日志目录
		logDir := filepath.Dir(logConfig.FilePath)
		if err := os.MkdirAll(logDir, 0755); err != nil {
			logrus.WithError(err).Warnf("创建日志目录失败: %s", logDir)
			log.SetOutput(os.Stdout)
		} else {
			// 打开日志文件
			logFile, err := os.OpenFile(logConfig.FilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
			if err != nil {
				logrus.WithError(err).Warnf("打开日志文件失败: %s", logConfig.FilePath)
				log.SetOutput(os.Stdout)
			} else {
				log.SetOutput(logFile)
				logrus.Infof("日志将写入文件: %s", logConfig.FilePath)
			}
		}
	} else {
		log.SetOutput(os.Stdout)
	}

	return log
}

// printConfigSummary 打印配置摘要
func printConfigSummary(log *logrus.Logger, cfg *config.Config) {
	log.Infof("=== 配置摘要 ===")
	log.Infof("数据库: %s:%d/%s", cfg.Database.Host, cfg.Database.Port, cfg.Database.Database)
	log.Infof("连接池: MaxOpen=%d, MaxIdle=%d, MaxLifetime=%v", 
		cfg.Database.MaxOpenConns, cfg.Database.MaxIdleConns, cfg.Database.ConnMaxLifetime)
	log.Infof("缓冲区: MaxSize=%d, FlushInterval=%v, BatchSize=%d", 
		cfg.Buffer.MaxSize, cfg.Buffer.FlushInterval, cfg.Buffer.BatchSize)
	log.Infof("写入器: ParallelWriters=%d, MaxBatchSize=%d, RetryAttempts=%d", 
		cfg.DatabaseWriter.ParallelWriters, cfg.DatabaseWriter.MaxBatchSize, cfg.DatabaseWriter.RetryAttempts)
	log.Infof("性能: MaxProcs=%d, GCPercent=%d", 
		cfg.Performance.MaxProcs, cfg.Performance.GCPercent)
	log.Infof("================")
}

// startMonitoringService 启动监控服务
func startMonitoringService(monConfig config.MonitoringConfig, log *logrus.Logger, bufferManager *buffer.ExtendedBufferManager, db *database.ExtendedDB) {
	log.Infof("启动监控服务，健康检查端口: %d", monConfig.HealthCheckPort)
	
	// 这里可以实现 HTTP 健康检查端点
	// 以及 Prometheus 指标暴露
	// 暂时只做日志记录
	
	ticker := time.NewTicker(monConfig.MetricsInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			// 获取缓冲区统计信息
			bufferStats := bufferManager.GetStats()
			dbStats := db.GetStats()
			
			log.Infof("监控指标 - 缓冲区: Platform=%d, Interface=%d, Subinterface=%d, 已处理=%d, 错误=%d", 
				bufferStats.PlatformBufferSize, 
				bufferStats.InterfaceBufferSize, 
				bufferStats.SubinterfaceBufferSize,
				bufferStats.TotalProcessed,
				bufferStats.TotalErrors)
			
			log.Infof("监控指标 - 数据库连接: Open=%d, InUse=%d, Idle=%d", 
				dbStats.OpenConnections, dbStats.InUse, dbStats.Idle)
			
			// 检查告警阈值
			checkAlertThresholds(log, monConfig.AlertThresholds, bufferStats, dbStats)
		}
	}
}

// checkAlertThresholds 检查告警阈值
func checkAlertThresholds(log *logrus.Logger, thresholds config.AlertThresholdsConfig, bufferStats buffer.ExtendedBufferStats, dbStats sql.DBStats) {
	// 检查缓冲区使用率
	totalBufferSize := bufferStats.PlatformBufferSize + bufferStats.InterfaceBufferSize + bufferStats.SubinterfaceBufferSize
	if totalBufferSize > 0 {
		// 这里需要知道最大缓冲区大小来计算百分比
		// 暂时跳过具体实现
	}
	
	// 检查数据库连接使用率
	if dbStats.MaxOpenConnections > 0 {
		connectionUsagePercent := (dbStats.InUse * 100) / dbStats.MaxOpenConnections
		if connectionUsagePercent >= thresholds.DBConnectionUsagePercent {
			log.Warnf("数据库连接使用率告警: %d%% (阈值: %d%%)", 
				connectionUsagePercent, thresholds.DBConnectionUsagePercent)
		}
	}
}

// startStatusReporter 启动状态报告器
func startStatusReporter(log *logrus.Logger, bufferManager *buffer.ExtendedBufferManager, db *database.ExtendedDB, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			// 获取并报告系统状态
			var m runtime.MemStats
			runtime.ReadMemStats(&m)
			
			bufferStats := bufferManager.GetStats()
			dbStats := db.GetStats()
			
			log.Infof("系统状态 - 内存: Alloc=%dMB, Sys=%dMB, NumGC=%d", 
				m.Alloc/(1024*1024), m.Sys/(1024*1024), m.NumGC)
			
			log.Infof("系统状态 - Goroutines=%d, 缓冲区总大小=%d, 数据库连接=%d/%d", 
				runtime.NumGoroutine(),
				bufferStats.PlatformBufferSize + bufferStats.InterfaceBufferSize + bufferStats.SubinterfaceBufferSize,
				dbStats.InUse, dbStats.OpenConnections)
		}
	}
}