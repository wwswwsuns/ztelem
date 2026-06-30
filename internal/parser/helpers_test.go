package parser

import (
	"testing"
	"time"
)

func TestFormatUptime(t *testing.T) {
	tests := []struct {
		input    uint32
		expected string
	}{
		{0, "00:00:00:00"},
		{61, "00:00:01:01"},
		{3661, "00:01:01:01"},
		{90061, "01:01:01:01"},
		{86400, "01:00:00:00"},
		{86400*365 + 3600*12 + 60*30 + 15, "365:12:30:15"},
	}

	for _, tt := range tests {
		got := formatUptime(tt.input)
		if got != tt.expected {
			t.Errorf("formatUptime(%d) = %s, want %s", tt.input, got, tt.expected)
		}
	}
}

func TestBytesToMB(t *testing.T) {
	tests := []struct {
		input    uint64
		expected uint64
	}{
		{0, 0},
		{1024 * 1024, 1},
		{1024*1024*100 + 500*1024, 100},
		{1024 * 1024 * 1024, 1024},
	}

	for _, tt := range tests {
		got := bytesToMB(tt.input)
		if got != tt.expected {
			t.Errorf("bytesToMB(%d) = %d, want %d", tt.input, got, tt.expected)
		}
	}
}

func TestNanosToSeconds(t *testing.T) {
	tests := []struct {
		input    uint64
		expected uint64
	}{
		{0, 0},
		{1000000000, 1},
		{1500000000, 1},
		{2000000000, 2},
	}

	for _, tt := range tests {
		got := nanosToSeconds(tt.input)
		if got != tt.expected {
			t.Errorf("nanosToSeconds(%d) = %d, want %d", tt.input, got, tt.expected)
		}
	}
}

func TestNanosToTimestamp(t *testing.T) {
	// 1000000000 ns = 1 second = 1970-01-01 00:00:01 UTC
	got := nanosToTimestamp(1000000000)
	expected := time.Unix(1, 0)
	if !got.Equal(expected) {
		t.Errorf("nanosToTimestamp(1000000000) = %v, want %v", got, expected)
	}

	// With sub-second nanoseconds
	got = nanosToTimestamp(1500000000)
	expected = time.Unix(1, 500000000)
	if !got.Equal(expected) {
		t.Errorf("nanosToTimestamp(1500000000) = %v, want %v", got, expected)
	}
}

func TestConvertAlarmStatus(t *testing.T) {
	tests := []struct {
		input    int32
		expected string
	}{
		{0, "INVALID"},
		{1, "NORMAL"},
		{2, "ALARM"},
		{99, "UNKNOWN_99"},
	}

	for _, tt := range tests {
		got := convertAlarmStatus(tt.input)
		if got != tt.expected {
			t.Errorf("convertAlarmStatus(%d) = %s, want %s", tt.input, got, tt.expected)
		}
	}
}

func TestConvertAdminStatus(t *testing.T) {
	tests := []struct {
		input    int32
		expected string
	}{
		{0, "ADMIN_STATUS_INVALID"},
		{1, "ADMIN_STATUS_UP"},
		{2, "ADMIN_STATUS_DOWN"},
		{3, "ADMIN_STATUS_TESTING"},
		{99, "ADMIN_STATUS_UNKNOWN_99"},
	}

	for _, tt := range tests {
		got := convertAdminStatus(tt.input)
		if got != tt.expected {
			t.Errorf("convertAdminStatus(%d) = %s, want %s", tt.input, got, tt.expected)
		}
	}
}

func TestConvertOperStatus(t *testing.T) {
	tests := []struct {
		input    int32
		expected string
	}{
		{0, "OPER_STATUS_INVALID"},
		{1, "OPER_STATUS_UP"},
		{2, "OPER_STATUS_DOWN"},
		{3, "OPER_STATUS_TESTING"},
		{4, "OPER_STATUS_UNKNOWN"},
		{5, "OPER_STATUS_DORMANT"},
		{6, "OPER_STATUS_NOT_PRESENT"},
		{7, "OPER_STATUS_LOWER_LAYER_DOWN"},
		{99, "OPER_STATUS_UNKNOWN_99"},
	}

	for _, tt := range tests {
		got := convertOperStatus(tt.input)
		if got != tt.expected {
			t.Errorf("convertOperStatus(%d) = %s, want %s", tt.input, got, tt.expected)
		}
	}
}

func TestConvertIPv4OperStatus(t *testing.T) {
	tests := []struct {
		input    int32
		expected string
	}{
		{0, "IPV4OPERSTATUS_STATUS_INVALID"},
		{1, "IPV4OPERSTATUS_STATUS_UP"},
		{2, "IPV4OPERSTATUS_STATUS_DOWN"},
		{99, "IPV4OPERSTATUS_STATUS_UNKNOWN_99"},
	}

	for _, tt := range tests {
		got := convertIPv4OperStatus(tt.input)
		if got != tt.expected {
			t.Errorf("convertIPv4OperStatus(%d) = %s, want %s", tt.input, got, tt.expected)
		}
	}
}

func TestConvertIPv6OperStatus(t *testing.T) {
	tests := []struct {
		input    int32
		expected string
	}{
		{0, "IPV6OPERSTATUS_STATUS_INVALID"},
		{1, "IPV6OPERSTATUS_STATUS_UP"},
		{2, "IPV6OPERSTATUS_STATUS_DOWN"},
		{99, "IPV6OPERSTATUS_STATUS_UNKNOWN_99"},
	}

	for _, tt := range tests {
		got := convertIPv6OperStatus(tt.input)
		if got != tt.expected {
			t.Errorf("convertIPv6OperStatus(%d) = %s, want %s", tt.input, got, tt.expected)
		}
	}
}

func TestConvertPhyStatus(t *testing.T) {
	tests := []struct {
		input    int32
		expected string
	}{
		{0, "PHY_STATUS_INVALID"},
		{1, "PHY_STATUS_UP"},
		{2, "PHY_STATUS_DOWN"},
		{99, "PHY_STATUS_UNKNOWN_99"},
	}

	for _, tt := range tests {
		got := convertPhyStatus(tt.input)
		if got != tt.expected {
			t.Errorf("convertPhyStatus(%d) = %s, want %s", tt.input, got, tt.expected)
		}
	}
}

func TestStringPtr(t *testing.T) {
	if stringPtr("") != nil {
		t.Error("stringPtr(\"\") should return nil")
	}
	s := stringPtr("hello")
	if s == nil || *s != "hello" {
		t.Errorf("stringPtr(\"hello\") = %v, want \"hello\"", s)
	}
}

func TestUint32Ptr(t *testing.T) {
	if uint32Ptr(0) != nil {
		t.Error("uint32Ptr(0) should return nil")
	}
	v := uint32Ptr(42)
	if v == nil || *v != 42 {
		t.Errorf("uint32Ptr(42) = %v, want 42", v)
	}
}

func TestFloat64Ptr(t *testing.T) {
	if float64Ptr(0) != nil {
		t.Error("float64Ptr(0) should return nil")
	}
	v := float64Ptr(3.14)
	if v == nil || *v != 3.14 {
		t.Errorf("float64Ptr(3.14) = %v, want 3.14", v)
	}
}

func TestOpticalPowerPtr(t *testing.T) {
	// 0.0 with isValid=true should still return -60 (proto3 default = no signal)
	v := opticalPowerPtr(0, true)
	if v == nil || *v != -60 {
		t.Errorf("opticalPowerPtr(0, true) = %v, want -60 (proto3 default)", v)
	}

	// Valid negative value (normal optical power)
	v = opticalPowerPtr(-5.5, true)
	if v == nil || *v != -5.5 {
		t.Errorf("opticalPowerPtr(-5.5, true) = %v, want -5.5", v)
	}

	// Valid value around -20 dBm (typical receive power)
	v = opticalPowerPtr(-20.0, true)
	if v == nil || *v != -20.0 {
		t.Errorf("opticalPowerPtr(-20.0, true) = %v, want -20.0", v)
	}

	// Invalid (isValid=false) should return -60
	v = opticalPowerPtr(0, false)
	if v == nil || *v != -60 {
		t.Errorf("opticalPowerPtr(0, false) = %v, want -60", v)
	}

	// Invalid with non-zero value should still return -60
	v = opticalPowerPtr(-5.0, false)
	if v == nil || *v != -60 {
		t.Errorf("opticalPowerPtr(-5.0, false) = %v, want -60", v)
	}
}

func TestBoolPtr(t *testing.T) {
	v := boolPtr(true)
	if v == nil || !*v {
		t.Errorf("boolPtr(true) = %v, want true", v)
	}
	v = boolPtr(false)
	if v == nil || *v {
		t.Errorf("boolPtr(false) = %v, want false", v)
	}
}

func TestTimePtr(t *testing.T) {
	if timePtr(time.Time{}) != nil {
		t.Error("timePtr(zero) should return nil")
	}
	now := time.Now()
	v := timePtr(now)
	if v == nil || !v.Equal(now) {
		t.Errorf("timePtr(now) = %v, want %v", v, now)
	}
}

func TestUtilizationToNumeric(t *testing.T) {
	got := utilizationToNumeric(0.5)
	if got != 50.0 {
		t.Errorf("utilizationToNumeric(0.5) = %f, want 50.0", got)
	}
	got = utilizationToNumeric(0)
	if got != 0.0 {
		t.Errorf("utilizationToNumeric(0) = %f, want 0.0", got)
	}
	got = utilizationToNumeric(1.0)
	if got != 100.0 {
		t.Errorf("utilizationToNumeric(1.0) = %f, want 100.0", got)
	}
}

func TestSafeStringValue(t *testing.T) {
	if safeStringValue(nil) != "" {
		t.Error("safeStringValue(nil) should return empty string")
	}
	s := "hello"
	if safeStringValue(&s) != "hello" {
		t.Errorf("safeStringValue(\"hello\") = %s", safeStringValue(&s))
	}
}

func TestSafeUint32Value(t *testing.T) {
	if safeUint32Value(nil) != 0 {
		t.Error("safeUint32Value(nil) should return 0")
	}
	v := uint32(42)
	if safeUint32Value(&v) != 42 {
		t.Errorf("safeUint32Value(42) = %d", safeUint32Value(&v))
	}
}
