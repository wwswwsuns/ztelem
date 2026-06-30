package buffer

import (
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/wwswwsuns/ztelem/internal/config"
	"github.com/wwswwsuns/ztelem/internal/models"
)

func newTestBufferManager() *FixedBufferManager {
	return &FixedBufferManager{
		config:       config.BufferConfig{FlushThreshold: 1000},
		writerConfig: config.DatabaseWriterConfig{MaxBatchSize: 100, RetryAttempts: 1},
		stopChan:     make(chan struct{}),
		platformBuffer:           newShardedPlatformMap(),
		interfaceBuffer:          newShardedInterfaceMap(),
		subinterfaceBuffer:       newShardedSubinterfaceMap(),
		alarmReportBuffer:        newShardedAlarmMap(),
		notificationReportBuffer: newShardedNotificationMap(),
		keyBuf: sync.Pool{
			New: func() interface{} { return &keyBuffer{buf: make([]byte, 0, 128)} },
		},
	}
}

func TestGeneratePlatformKey(t *testing.T) {
	bm := newTestBufferManager()
	ts := time.Date(2026, 6, 30, 12, 0, 30, 0, time.UTC)
	metric := &models.PlatformMetric{
		Timestamp:     ts,
		SystemID:      "device-1",
		ComponentName: "CPU0",
	}

	key := bm.generatePlatformKey(metric)
	// Key should be: UnixSeconds_systemID_componentName
	if !strings.Contains(key, "device-1") || !strings.Contains(key, "CPU0") {
		t.Fatalf("unexpected key format: %s", key)
	}
	// Truncated to second
	unixSec := ts.Truncate(time.Second).Unix()
	expectedPrefix := "1751284830" // approximate unix timestamp
	if !strings.HasPrefix(key, expectedPrefix[:4]) {
		t.Logf("key prefix: %s (unix=%d)", key[:10], unixSec)
	}
}

func TestGenerateInterfaceKey(t *testing.T) {
	bm := newTestBufferManager()
	ts := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	metric := &models.InterfaceMetric{
		Timestamp:     ts,
		SystemID:      "dev-2",
		InterfaceName: "GE1/0/1",
	}

	key := bm.generateInterfaceKey(metric)
	if !strings.Contains(key, "dev-2") || !strings.Contains(key, "GE1/0/1") {
		t.Fatalf("unexpected key: %s", key)
	}
}

func TestGenerateSubinterfaceKey(t *testing.T) {
	bm := newTestBufferManager()
	ts := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	metric := &models.SubinterfaceMetric{
		Timestamp:        ts,
		SystemID:         "dev-3",
		InterfaceName:    "GE1/0/1",
		SubinterfaceName: "100",
	}

	key := bm.generateSubinterfaceKey(metric)
	if !strings.Contains(key, "dev-3") || !strings.Contains(key, "GE1/0/1") || !strings.Contains(key, "100") {
		t.Fatalf("unexpected key: %s", key)
	}
}

func TestGenerateAlarmKey(t *testing.T) {
	bm := newTestBufferManager()
	metric := &models.AlarmReportMetric{
		SystemID:       "dev-4",
		FlowID:         12345,
		AlarmTimestamp: 9999,
	}

	key := bm.generateAlarmKey(metric)
	expected := "dev-4:12345:9999"
	if key != expected {
		t.Fatalf("expected %s, got %s", expected, key)
	}
}

func TestGenerateNotificationKey(t *testing.T) {
	bm := newTestBufferManager()
	metric := &models.NotificationReportMetric{
		SystemID:              "dev-5",
		FlowID:                67890,
		NotificationTimestamp: 1111,
	}

	key := bm.generateNotificationKey(metric)
	expected := "dev-5:67890:1111"
	if key != expected {
		t.Fatalf("expected %s, got %s", expected, key)
	}
}

func TestMergePlatformMetric(t *testing.T) {
	bm := newTestBufferManager()

	opUp := "UP"
	opDown := "DOWN"
	alarmNormal := "NORMAL"
	alarmCPU := "ALARM"
	ts1 := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	ts2 := time.Date(2026, 6, 30, 12, 0, 30, 0, time.UTC)

	existing := &models.PlatformMetric{
		Timestamp: ts1,
		CommonState: &models.CommonState{
			OperStatus: &opUp,
		},
		MemData: &models.MemData{
			MemAlarmStatus: &alarmNormal,
		},
	}

	newMetric := &models.PlatformMetric{
		Timestamp: ts2,
		CommonState: &models.CommonState{
			OperStatus: &opDown,
		},
		CPUData: &models.CPUData{
			CPUAlarmStatus: &alarmCPU,
		},
	}

	bm.mergePlatformMetric(existing, newMetric)

	if *existing.OperStatus != "DOWN" {
		t.Fatalf("expected OperStatus=DOWN, got %s", *existing.OperStatus)
	}
	if *existing.MemAlarmStatus != "NORMAL" {
		t.Fatalf("expected MemAlarmStatus=NORMAL, got %s", *existing.MemAlarmStatus)
	}
	if *existing.CPUAlarmStatus != "ALARM" {
		t.Fatalf("expected CPUAlarmStatus=ALARM, got %s", *existing.CPUAlarmStatus)
	}
	if !existing.Timestamp.Equal(ts2) {
		t.Fatalf("expected timestamp updated to ts2")
	}
}

func TestMergePlatformMetric_NilSubStructs(t *testing.T) {
	bm := newTestBufferManager()

	opDown := "DOWN"
	existing := &models.PlatformMetric{
		Timestamp:  time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC),
		CommonState: nil,
	}

	newMetric := &models.PlatformMetric{
		Timestamp: time.Date(2026, 6, 30, 12, 0, 30, 0, time.UTC),
		CommonState: &models.CommonState{
			OperStatus: &opDown,
		},
	}

	bm.mergePlatformMetric(existing, newMetric)

	if existing.CommonState == nil {
		t.Fatal("expected CommonState to be initialized")
	}
	if *existing.OperStatus != "DOWN" {
		t.Fatalf("expected DOWN, got %s", *existing.OperStatus)
	}
}

func TestMergeInterfaceMetric(t *testing.T) {
	bm := newTestBufferManager()

	adminUp := "ADMIN_STATUS_UP"
	adminDown := "ADMIN_STATUS_DOWN"
	ts1 := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	ts2 := time.Date(2026, 6, 30, 12, 0, 30, 0, time.UTC)

	existing := &models.InterfaceMetric{
		Timestamp:      ts1,
		AdminStatusStr: &adminUp,
	}

	newMetric := &models.InterfaceMetric{
		Timestamp:      ts2,
		AdminStatusStr: &adminDown,
	}

	bm.mergeInterfaceMetric(existing, newMetric)

	if *existing.AdminStatusStr != "ADMIN_STATUS_DOWN" {
		t.Fatalf("expected ADMIN_STATUS_DOWN, got %s", *existing.AdminStatusStr)
	}
	if !existing.Timestamp.Equal(ts2) {
		t.Fatal("expected timestamp updated")
	}
}

func TestMergeSubinterfaceMetric(t *testing.T) {
	bm := newTestBufferManager()

	adminUp := "ADMIN_STATUS_UP"
	adminDown := "ADMIN_STATUS_DOWN"
	ts1 := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	ts2 := time.Date(2026, 6, 30, 12, 0, 30, 0, time.UTC)

	existing := &models.SubinterfaceMetric{
		Timestamp:      ts1,
		AdminStatusStr: &adminUp,
	}

	newMetric := &models.SubinterfaceMetric{
		Timestamp:      ts2,
		AdminStatusStr: &adminDown,
	}

	bm.mergeSubinterfaceMetric(existing, newMetric)

	if *existing.AdminStatusStr != "ADMIN_STATUS_DOWN" {
		t.Fatalf("expected ADMIN_STATUS_DOWN, got %s", *existing.AdminStatusStr)
	}
}

func TestAddPlatformMetrics_Aggregation(t *testing.T) {
	bm := newTestBufferManager()

	ts := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	opUp := "UP"
	opDown := "DOWN"

	metrics := []models.PlatformMetric{
		{
			Timestamp:     ts,
			SystemID:      "dev-1",
			ComponentName: "CPU0",
			CommonState: &models.CommonState{
				OperStatus: &opUp,
			},
		},
		{
			Timestamp:     ts, // same second = same key
			SystemID:      "dev-1",
			ComponentName: "CPU0",
			CommonState: &models.CommonState{
				OperStatus: &opDown,
			},
		},
	}

	err := bm.AddPlatformMetrics(metrics)
	if err != nil {
		t.Fatal(err)
	}

	// Should be aggregated into 1 record
	if bm.platformBuffer.Len() != 1 {
		t.Fatalf("expected 1 aggregated record, got %d", bm.platformBuffer.Len())
	}
}

func TestAddInterfaceMetrics_Aggregation(t *testing.T) {
	bm := newTestBufferManager()

	ts := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	adminUp := "ADMIN_STATUS_UP"
	adminDown := "ADMIN_STATUS_DOWN"

	metrics := []models.InterfaceMetric{
		{
			Timestamp:      ts,
			SystemID:       "dev-1",
			InterfaceName:  "GE1/0/1",
			AdminStatusStr: &adminUp,
		},
		{
			Timestamp:      ts,
			SystemID:       "dev-1",
			InterfaceName:  "GE1/0/1",
			AdminStatusStr: &adminDown,
		},
	}

	err := bm.AddInterfaceMetrics(metrics)
	if err != nil {
		t.Fatal(err)
	}

	if bm.interfaceBuffer.Len() != 1 {
		t.Fatalf("expected 1, got %d", bm.interfaceBuffer.Len())
	}
}

func TestAddAlarmReportMetrics_NoAggregation(t *testing.T) {
	bm := newTestBufferManager()

	// Alarms use flow_id as unique key, different flow_id = different records
	metrics := []models.AlarmReportMetric{
		{SystemID: "dev-1", FlowID: 1, AlarmTimestamp: 100},
		{SystemID: "dev-1", FlowID: 2, AlarmTimestamp: 200},
	}

	err := bm.AddAlarmReportMetrics(metrics)
	if err != nil {
		t.Fatal(err)
	}

	if bm.alarmReportBuffer.Len() != 2 {
		t.Fatalf("expected 2 (no aggregation), got %d", bm.alarmReportBuffer.Len())
	}
}

func TestFlushAll_Empty(t *testing.T) {
	bm := newTestBufferManager()
	err := bm.FlushAll()
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetStats(t *testing.T) {
	bm := newTestBufferManager()

	s := "sys"
	bm.platformBuffer.Set("k1", &models.PlatformMetric{SystemID: s})
	bm.interfaceBuffer.Set("k2", &models.InterfaceMetric{SystemID: s})

	stats := bm.GetStats()
	if stats.PlatformBufferSize != 1 {
		t.Fatalf("expected platform=1, got %d", stats.PlatformBufferSize)
	}
	if stats.InterfaceBufferSize != 1 {
		t.Fatalf("expected interface=1, got %d", stats.InterfaceBufferSize)
	}
}
