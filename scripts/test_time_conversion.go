package main

import (
	"fmt"
	"time"
)

// convertUnixTimestamp 将uint32的Unix时间戳转换为time.Time指针
func convertUnixTimestamp(timestamp uint32) *time.Time {
	if timestamp == 0 {
		return nil
	}
	t := time.Unix(int64(timestamp), 0).UTC()
	return &t
}

func main() {
	// 测试时间转换
	fmt.Println("=== 时间转换测试 ===")
	
	// 测试用例1: 正常时间戳
	timestamp1 := uint32(1726833600) // 2024-09-20 14:00:00 UTC
	converted1 := convertUnixTimestamp(timestamp1)
	if converted1 != nil {
		fmt.Printf("时间戳 %d 转换为: %s\n", timestamp1, converted1.Format("2006-01-02 15:04:05 UTC"))
	}
	
	// 测试用例2: 当前时间戳
	now := uint32(time.Now().Unix())
	converted2 := convertUnixTimestamp(now)
	if converted2 != nil {
		fmt.Printf("当前时间戳 %d 转换为: %s\n", now, converted2.Format("2006-01-02 15:04:05 UTC"))
	}
	
	// 测试用例3: 零时间戳
	timestamp3 := uint32(0)
	converted3 := convertUnixTimestamp(timestamp3)
	if converted3 == nil {
		fmt.Printf("时间戳 %d 转换为: nil (正确处理)\n", timestamp3)
	}
	
	// 测试用例4: 历史时间戳
	timestamp4 := uint32(946684800) // 2000-01-01 00:00:00 UTC
	converted4 := convertUnixTimestamp(timestamp4)
	if converted4 != nil {
		fmt.Printf("历史时间戳 %d 转换为: %s\n", timestamp4, converted4.Format("2006-01-02 15:04:05 UTC"))
	}
	
	fmt.Println("\n=== 数据库格式测试 ===")
	
	// 模拟数据库插入格式
	testTime := convertUnixTimestamp(uint32(time.Now().Unix()))
	if testTime != nil {
		fmt.Printf("数据库插入格式: %s\n", testTime.Format("2006-01-02 15:04:05.000000-07:00"))
		fmt.Printf("ISO 8601格式: %s\n", testTime.Format(time.RFC3339))
	}
}