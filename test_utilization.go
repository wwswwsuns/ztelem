package main

import (
	"fmt"
)

// utilizationToNumeric 将利用率(1/10000单位)转换为数字百分比(不带%符号)
func utilizationToNumeric(utilization float32) float64 {
	return float64(utilization) / 100.0
}

func main() {
	fmt.Printf("测试utilizationToNumeric函数:\n")
	fmt.Printf("Input: 1234.5, Output: %.2f\n", utilizationToNumeric(1234.5))
	fmt.Printf("Input: 0.0, Output: %.2f\n", utilizationToNumeric(0.0))
	fmt.Printf("Input: 50.0, Output: %.2f\n", utilizationToNumeric(50.0))
	fmt.Printf("Input: 10000.0, Output: %.2f\n", utilizationToNumeric(10000.0))
}