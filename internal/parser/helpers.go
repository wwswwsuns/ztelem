package parser

import (
	"fmt"
	"time"
)

// formatUptime 将秒数转换为dd:hh:mm:ss格式
func formatUptime(seconds uint32) string {
	if seconds == 0 {
		return "00:00:00:00"
	}
	
	days := seconds / 86400
	hours := (seconds % 86400) / 3600
	minutes := (seconds % 3600) / 60
	secs := seconds % 60
	
	return fmt.Sprintf("%02d:%02d:%02d:%02d", days, hours, minutes, secs)
}

// bytesToMB 将字节转换为MB
func bytesToMB(bytes uint64) uint64 {
	return bytes / (1024 * 1024)
}

// percentageFormat 将数值转换为百分比格式，保留两位小数
func percentageFormat(value float64) string {
	return fmt.Sprintf("%.2f%%", value)
}

// utilizationToPercentage 将利用率(1/10000单位)转换为百分比
func utilizationToPercentage(utilization float32) string {
	percentage := float64(utilization) / 100.0
	return fmt.Sprintf("%.2f%%", percentage)
}

// nanosToSeconds 将纳秒转换为秒
func nanosToSeconds(nanos uint64) uint64 {
	return nanos / 1000000000
}

// nanosToTimestamp 将纳秒时间戳转换为time.Time
func nanosToTimestamp(nanos uint64) time.Time {
	seconds := int64(nanos / 1000000000)
	nanosRemainder := int64(nanos % 1000000000)
	return time.Unix(seconds, nanosRemainder)
}

// addUnitMbps 添加Mbps单位
func addUnitMbps(value float32) string {
	return fmt.Sprintf("%.2f Mbps", value)
}

// addUnitKfps 添加Kfps单位
func addUnitKfps(value float32) string {
	return fmt.Sprintf("%.2f Kfps", value)
}

// alarmStatusToString 将AlarmStatus枚举转换为字符串
func alarmStatusToString(status int32) string {
	switch status {
	case 0:
		return "INVALID"
	case 1:
		return "NORMAL"
	case 2:
		return "ALARM"
	default:
		return "UNKNOWN"
	}
}

// adminStatusToString 将AdminStatus枚举转换为字符串
func adminStatusToString(status int32) string {
	switch status {
	case 0:
		return "ADMIN_STATUS_INVALID"
	case 1:
		return "ADMIN_STATUS_UP"
	case 2:
		return "ADMIN_STATUS_DOWN"
	case 3:
		return "ADMIN_STATUS_TESTING"
	default:
		return "UNKNOWN"
	}
}

// operStatusToString 将OperStatus枚举转换为字符串
func operStatusToString(status int32) string {
	switch status {
	case 0:
		return "OPER_STATUS_INVALID"
	case 1:
		return "OPER_STATUS_UP"
	case 2:
		return "OPER_STATUS_DOWN"
	case 3:
		return "OPER_STATUS_TESTING"
	case 4:
		return "OPER_STATUS_UNKNOWN"
	case 5:
		return "OPER_STATUS_DORMANT"
	case 6:
		return "OPER_STATUS_NOT_PRESENT"
	case 7:
		return "OPER_STATUS_LOWER_LAYER_DOWN"
	default:
		return "UNKNOWN"
	}
}

// phyStatusToString 将PhyStatus枚举转换为字符串
func phyStatusToString(status int32) string {
	switch status {
	case 0:
		return "PHY_STATUS_INVALID"
	case 1:
		return "PHY_STATUS_UP"
	case 2:
		return "PHY_STATUS_DOWN"
	default:
		return "UNKNOWN"
	}
}

// ipv4OperStatusToString 将IPv4OperStatus枚举转换为字符串
func ipv4OperStatusToString(status int32) string {
	switch status {
	case 0:
		return "IPV4OPERSTATUS_STATUS_INVALID"
	case 1:
		return "IPV4OPERSTATUS_STATUS_UP"
	case 2:
		return "IPV4OPERSTATUS_STATUS_DOWN"
	default:
		return "UNKNOWN"
	}
}

// ipv6OperStatusToString 将IPv6OperStatus枚举转换为字符串
func ipv6OperStatusToString(status int32) string {
	switch status {
	case 0:
		return "IPV6OPERSTATUS_STATUS_INVALID"
	case 1:
		return "IPV6OPERSTATUS_STATUS_UP"
	case 2:
		return "IPV6OPERSTATUS_STATUS_DOWN"
	default:
		return "UNKNOWN"
	}
}

// safeStringValue 安全获取字符串值，避免空指针
func safeStringValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// safeUint32Value 安全获取uint32值，避免空指针
func safeUint32Value(v *uint32) uint32 {
	if v == nil {
		return 0
	}
	return *v
}

// safeUint64Value 安全获取uint64值，避免空指针
func safeUint64Value(v *uint64) uint64 {
	if v == nil {
		return 0
	}
	return *v
}

// safeFloat32Value 安全获取float32值，避免空指针
func safeFloat32Value(v *float32) float32 {
	if v == nil {
		return 0.0
	}
	return *v
}

// safeBoolValue 安全获取bool值，避免空指针
func safeBoolValue(v *bool) bool {
	if v == nil {
		return false
	}
	return *v
}

// 指针赋值辅助函数
func stringPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func uint32Ptr(v uint32) *uint32 {
	if v == 0 {
		return nil
	}
	return &v
}

func uint64Ptr(v uint64) *uint64 {
	if v == 0 {
		return nil
	}
	return &v
}

func float64Ptr(v float64) *float64 {
	if v == 0.0 {
		return nil
	}
	return &v
}

func boolPtr(v bool) *bool {
	return &v
}

func timePtr(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}

// formatPercentage 格式化百分比，将1/10000单位转换为百分比字符串
func formatPercentage(value float32) string {
	return utilizationToPercentage(value)
}

// utilizationToNumeric 将利用率(1/10000单位)转换为数字百分比(不带%符号)
func utilizationToNumeric(utilization float32) float64 {
	return float64(utilization) / 100.0
}

// percentageToNumeric 将uint32百分比转换为数字百分比(不带%符号)
func percentageToNumeric(percentage uint32) float64 {
	return float64(percentage)
}

// storageAvailabilityToNumeric 将存储可用性转换为数字百分比
func storageAvailabilityToNumeric(availability uint32) float64 {
	return float64(availability)
}