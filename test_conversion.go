package main

import (
	"fmt"
)

// utilizationToNumeric 将利用率(1/10000单位)转换为数字百分比(不带%符号)
func utilizationToNumeric(utilization float32) float64 {
	return float64(utilization) / 10000.0 * 100.0
}

func main() {
	fmt.Println("测试utilization转换函数:")
	
	// 测试不同的原始值
	testCases := []struct {
		input    float32
		expected string
	}{
		{0, "0.00% (无流量)"},
		{1234, "12.34% (中等流量)"},
		{5000, "50.00% (高流量)"},
		{10000, "100.00% (满载)"},
		{500, "5.00% (低流量)"},
	}
	
	for _, tc := range testCases {
		result := utilizationToNumeric(tc.input)
		fmt.Printf("原始值: %.0f -> 百分比: %.2f%%\n", tc.input, result)
	}
}