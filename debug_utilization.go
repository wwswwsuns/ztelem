package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/wwswwsuns/ztelem/internal/config"
	"github.com/wwswwsuns/ztelem/internal/database"
	"github.com/wwswwsuns/ztelem/internal/models"
)

func main() {
	// 加载配置
	cfg, err := config.LoadConfig("production-config.yaml")
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 连接数据库
	db, err := database.NewConnection(cfg.Database)
	if err != nil {
		log.Fatalf("数据库连接失败: %v", err)
	}
	defer db.Close()

	// 创建测试数据
	testMetric := &models.InterfaceMetric{
		Timestamp:         time.Now(),
		SystemID:          "test-system",
		InterfaceName:     "test-interface",
		InputUtilization:  floatPtr(12.34),
		OutputUtilization: floatPtr(56.78),
	}

	fmt.Printf("测试数据: InputUtilization=%.2f, OutputUtilization=%.2f\n", 
		*testMetric.InputUtilization, *testMetric.OutputUtilization)

	// 写入数据库
	writer := database.NewWriter(db, cfg.Writer)
	err = writer.WriteInterfaceMetrics([]*models.InterfaceMetric{testMetric})
	if err != nil {
		log.Fatalf("写入数据失败: %v", err)
	}

	fmt.Println("测试数据写入成功")

	// 查询验证
	query := `SELECT interface_name, input_utilization, output_utilization 
			  FROM telemetry.interface_metrics 
			  WHERE interface_name = 'test-interface' 
			  ORDER BY time DESC LIMIT 1`
	
	rows, err := db.Query(query)
	if err != nil {
		log.Fatalf("查询失败: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var name string
		var inputUtil, outputUtil float64
		err := rows.Scan(&name, &inputUtil, &outputUtil)
		if err != nil {
			log.Fatalf("扫描失败: %v", err)
		}
		fmt.Printf("数据库中的数据: %s, InputUtilization=%.2f, OutputUtilization=%.2f\n", 
			name, inputUtil, outputUtil)
	}
}

func floatPtr(f float64) *float64 {
	return &f
}