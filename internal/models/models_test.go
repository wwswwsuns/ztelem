package models

import (
	"testing"
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
	}

	for _, tt := range tests {
		got := FormatUptime(tt.input)
		if got != tt.expected {
			t.Errorf("FormatUptime(%d) = %s, want %s", tt.input, got, tt.expected)
		}
	}
}

func TestBytesToMB(t *testing.T) {
	tests := []struct {
		input    uint64
		expected float64
	}{
		{0, 0},
		{1024 * 1024, 1.0},
		{1024 * 1024 * 100, 100.0},
	}

	for _, tt := range tests {
		got := BytesToMB(tt.input)
		if got != tt.expected {
			t.Errorf("BytesToMB(%d) = %f, want %f", tt.input, got, tt.expected)
		}
	}
}

func TestNanosecondsToTime(t *testing.T) {
	got := NanosecondsToTime(1000000000)
	if got.Unix() != 1 {
		t.Errorf("NanosecondsToTime(1000000000).Unix() = %d, want 1", got.Unix())
	}
}

func TestNanosecondsToSeconds(t *testing.T) {
	tests := []struct {
		input    uint64
		expected uint64
	}{
		{0, 0},
		{1000000000, 1},
		{1999999999, 1},
		{2000000000, 2},
	}

	for _, tt := range tests {
		got := NanosecondsToSeconds(tt.input)
		if got != tt.expected {
			t.Errorf("NanosecondsToSeconds(%d) = %d, want %d", tt.input, got, tt.expected)
		}
	}
}

func TestFormatPercentage(t *testing.T) {
	// FormatPercentage just formats the value as-is with %%
	got := FormatPercentage(0.1234)
	if got != "0.12%" {
		t.Errorf("FormatPercentage(0.1234) = %s, want 0.12%%", got)
	}
	got = FormatPercentage(50.0)
	if got != "50.00%" {
		t.Errorf("FormatPercentage(50.0) = %s, want 50.00%%", got)
	}
}

func TestFormatUtilization(t *testing.T) {
	got := FormatUtilization(0.5)
	if got != "50.00%" {
		t.Errorf("FormatUtilization(0.5) = %s, want 50.00%%", got)
	}
}

func TestFormatTrafficRate(t *testing.T) {
	got := FormatTrafficRate(100.5)
	if got != "100.50 Mbps" {
		t.Errorf("FormatTrafficRate(100.5) = %s, want 100.50 Mbps", got)
	}
}

func TestFormatPacketRate(t *testing.T) {
	got := FormatPacketRate(50.25)
	if got != "50.25 Kfps" {
		t.Errorf("FormatPacketRate(50.25) = %s, want 50.25 Kfps", got)
	}
}

func TestAlarmStatusString(t *testing.T) {
	tests := []struct {
		input    AlarmStatus
		expected string
	}{
		{AlarmStatusInvalid, "INVALID"},
		{AlarmStatusNormal, "NORMAL"},
		{AlarmStatusAlarm, "ALARM"},
		{AlarmStatus(99), "INVALID"},
	}

	for _, tt := range tests {
		got := tt.input.String()
		if got != tt.expected {
			t.Errorf("AlarmStatus(%d).String() = %s, want %s", tt.input, got, tt.expected)
		}
	}
}

func TestAdminStatusString(t *testing.T) {
	tests := []struct {
		input    AdminStatus
		expected string
	}{
		{AdminStatusInvalid, "INVALID"},
		{AdminStatusUp, "UP"},
		{AdminStatusDown, "DOWN"},
		{AdminStatusTesting, "TESTING"},
		{AdminStatus(99), "INVALID"},
	}

	for _, tt := range tests {
		got := tt.input.String()
		if got != tt.expected {
			t.Errorf("AdminStatus(%d).String() = %s, want %s", tt.input, got, tt.expected)
		}
	}
}

func TestOperStatusString(t *testing.T) {
	tests := []struct {
		input    OperStatus
		expected string
	}{
		{OperStatusInvalid, "INVALID"},
		{OperStatusUp, "UP"},
		{OperStatusDown, "DOWN"},
		{OperStatusTesting, "TESTING"},
		{OperStatusUnknown, "UNKNOWN"},
		{OperStatusDormant, "DORMANT"},
		{OperStatusNotPresent, "NOT_PRESENT"},
		{OperStatusLowerLayerDown, "LOWER_LAYER_DOWN"},
		{OperStatus(99), "INVALID"},
	}

	for _, tt := range tests {
		got := tt.input.String()
		if got != tt.expected {
			t.Errorf("OperStatus(%d).String() = %s, want %s", tt.input, got, tt.expected)
		}
	}
}

func TestPhyStatusString(t *testing.T) {
	tests := []struct {
		input    PhyStatus
		expected string
	}{
		{PhyStatusInvalid, "INVALID"},
		{PhyStatusUp, "UP"},
		{PhyStatusDown, "DOWN"},
		{PhyStatus(99), "INVALID"},
	}

	for _, tt := range tests {
		got := tt.input.String()
		if got != tt.expected {
			t.Errorf("PhyStatus(%d).String() = %s, want %s", tt.input, got, tt.expected)
		}
	}
}

func TestSubinterfaceMetric_Getters(t *testing.T) {
	inPkts := uint64(100)
	outPkts := uint64(200)
	m := &SubinterfaceMetric{
		InPkts:  &inPkts,
		OutPkts: &outPkts,
	}

	if m.GetInPkts() != 100 {
		t.Errorf("GetInPkts() = %d, want 100", m.GetInPkts())
	}
	if m.GetOutPkts() != 200 {
		t.Errorf("GetOutPkts() = %d, want 200", m.GetOutPkts())
	}

	// Nil case
	m2 := &SubinterfaceMetric{}
	if m2.GetInPkts() != 0 {
		t.Errorf("GetInPkts() on nil = %d, want 0", m2.GetInPkts())
	}
}

func TestPlatformMetric_SubStructs(t *testing.T) {
	// Verify sub-structs work correctly
	m := PlatformMetric{
		SystemID:      "dev-1",
		ComponentName: "CPU0",
		CommonState: &CommonState{
			OperStatus: strPtr("UP"),
		},
		CPUData: &CPUData{
			CPUInstant: floatPtr(75.5),
		},
	}

	if *m.OperStatus != "UP" {
		t.Errorf("OperStatus = %s, want UP", *m.OperStatus)
	}
	if *m.CPUInstant != 75.5 {
		t.Errorf("CPUInstant = %f, want 75.5", *m.CPUInstant)
	}

	// Nil sub-struct should not panic
	m2 := PlatformMetric{SystemID: "dev-2"}
	if m2.CommonState != nil {
		t.Error("expected nil CommonState")
	}
}

func strPtr(s string) *string { return &s }
func floatPtr(f float64) *float64 { return &f }
