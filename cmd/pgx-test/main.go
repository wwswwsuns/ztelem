package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/wwswwsuns/ztelem/internal/config"
	"github.com/wwswwsuns/ztelem/internal/database"
	"github.com/wwswwsuns/ztelem/internal/monitoring"
	"github.com/sirupsen/logrus"
)

var (
	configFile = flag.String("config", "production-config-optimized.yaml", "配置文件路径")
	debugMode  = flag.Bool("debug", false, "启用调试模式")
)

func main() {
	flag.Parse()

	// 初始化日志
	logger := logrus.New()
	if *debugMode {
		logger.SetLevel(logrus.DebugLevel)
	}
	logger.Info("启动pgx性能测试版本")

	// 加载配置
	cfg, err := config.LoadConfig(*configFile)
	if err != nil {
		logger.WithError(err).Fatal("加载配置失败")
	}

	// 启动Prometheus指标服务器
	prometheusServer := monitoring.NewPrometheusServer(12112, logger)
	if err := prometheusServer.Start(); err != nil {
		logger.WithError(err).Fatal("启动Prometheus服务器失败")
	}
	defer prometheusServer.Stop()

	// 创建pgx数据库连接
	pgxDB, err := database.NewPgxDatabase(cfg.Database, logger)
	if err != nil {
		logger.WithError(err).Fatal("创建pgx数据库连接失败")
	}
	defer pgxDB.Close()

	// 测试连接
	if err := pgxDB.TestConnection(); err != nil {
		logger.WithError(err).Fatal("数据库连接测试失败")
	}

	logger.Info("pgx数据库连接成功")
	logger.Info("Prometheus指标服务器运行在端口 12112")
	logger.Info("访问 http://localhost:12112/metrics 查看指标")

	// 启动连接池指标更新
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				pgxDB.UpdatePoolMetrics()
			case <-ctx.Done():
				return
			}
		}
	}()

	// 性能测试：创建一些测试数据
	go performanceTest(pgxDB, logger)

	// 优雅关闭处理
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	logger.Info("pgx测试服务器已启动，按 Ctrl+C 退出")
	<-sigChan

	logger.Info("正在关闭服务器...")
	cancel()
}

func performanceTest(db *database.PgxDatabase, logger *logrus.Logger) {
	// 这里可以添加性能测试代码
	// 例如：批量插入测试数据，测量性能指标
	
	logger.Info("性能测试功能已准备就绪")
	logger.Info("可以通过现有的采集器向数据库写入数据来测试性能")
}