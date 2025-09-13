package main

import (
	"fmt"
)

// 测试float64Ptr函数
func float64Ptr(v float64) *float64 {
	return &v
}

func main() {
	// 测试不同的utilization值
	testValues := []float64{0.0, 12.34, 56.78, 100.0}
	
	for _, val := range testValues {
		ptr := float64Ptr(val)
		if ptr != nil {
			fmt.Printf("Input: %.2f, Pointer: %.2f\n", val, *ptr)
		} else {
			fmt.Printf("Input: %.2f, Pointer: nil\n", val)
		}
	}
}