package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/wwswwsuns/ztelem/internal/config"
	"github.com/wwswwsuns/ztelem/internal/collector"
)

func main() {
	// 加载配置
	cfg, err := config.LoadConfig("production-config.yaml")
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 创建采集器
	c, err := collector.NewSimpleCollector(cfg)
	if err != nil {
		log.Fatalf("创建采集器失败: %v", err)
	}

	fmt.Println("开始监听遥测数据，查看原始utilization值...")
	fmt.Println("按Ctrl+C停止")

	// 启动采集器，但添加调试输出
	go func() {
		time.Sleep(30 * time.Second)
		fmt.Println("30秒后自动退出")
		os.Exit(0)
	}()

	err = c.Start()
	if err != nil {
		log.Fatalf("启动采集器失败: %v", err)
	}
}