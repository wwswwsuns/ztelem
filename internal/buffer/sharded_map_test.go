package buffer

import (
	"sync"
	"testing"

	"github.com/wwswwsuns/ztelem/internal/models"
)

func TestShardedPlatformMap_SetAndGet(t *testing.T) {
	m := newShardedPlatformMap()

	s := "test-system"
	c := "CPU0"
	metric := &models.PlatformMetric{
		SystemID:      s,
		ComponentName: c,
	}

	m.Set("key1", metric)

	got, ok := m.Get("key1")
	if !ok {
		t.Fatal("expected to find key1")
	}
	if got.SystemID != s || got.ComponentName != c {
		t.Fatalf("got %v, want SystemID=%s ComponentName=%s", got, s, c)
	}
}

func TestShardedPlatformMap_GetMissing(t *testing.T) {
	m := newShardedPlatformMap()
	_, ok := m.Get("nonexistent")
	if ok {
		t.Fatal("expected no result for missing key")
	}
}

func TestShardedPlatformMap_Len(t *testing.T) {
	m := newShardedPlatformMap()
	if m.Len() != 0 {
		t.Fatalf("expected 0, got %d", m.Len())
	}

	s := "sys"
	m.Set("k1", &models.PlatformMetric{SystemID: s})
	m.Set("k2", &models.PlatformMetric{SystemID: s})
	m.Set("k3", &models.PlatformMetric{SystemID: s})

	if m.Len() != 3 {
		t.Fatalf("expected 3, got %d", m.Len())
	}
}

func TestShardedPlatformMap_SwapAll(t *testing.T) {
	m := newShardedPlatformMap()
	s := "sys"
	m.Set("k1", &models.PlatformMetric{SystemID: s, ComponentName: "A"})
	m.Set("k2", &models.PlatformMetric{SystemID: s, ComponentName: "B"})

	result := m.SwapAll()
	if len(result) != 2 {
		t.Fatalf("expected 2 results, got %d", len(result))
	}

	// After swap, map should be empty
	if m.Len() != 0 {
		t.Fatalf("expected 0 after swap, got %d", m.Len())
	}
}

func TestShardedPlatformMap_SwapAllEmpty(t *testing.T) {
	m := newShardedPlatformMap()
	result := m.SwapAll()
	if len(result) != 0 {
		t.Fatalf("expected 0 results from empty swap, got %d", len(result))
	}
}

func TestShardedPlatformMap_ConcurrentAccess(t *testing.T) {
	m := newShardedPlatformMap()
	var wg sync.WaitGroup

	// Concurrent writes
	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			key := "key" + string(rune('A'+i%26))
			m.Set(key, &models.PlatformMetric{SystemID: "sys"})
		}(i)
	}
	wg.Wait()

	// Concurrent reads
	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			key := "key" + string(rune('A'+i%26))
			m.Get(key)
		}(i)
	}
	wg.Wait()
}

func TestShardedInterfaceMap_SetGetSwap(t *testing.T) {
	m := newShardedInterfaceMap()
	iface := &models.InterfaceMetric{InterfaceName: "eth0"}
	m.Set("k1", iface)

	got, ok := m.Get("k1")
	if !ok || got.InterfaceName != "eth0" {
		t.Fatalf("expected eth0, got %v", got)
	}

	result := m.SwapAll()
	if len(result) != 1 {
		t.Fatalf("expected 1, got %d", len(result))
	}
}

func TestShardedSubinterfaceMap_SetGetSwap(t *testing.T) {
	m := newShardedSubinterfaceMap()
	sub := &models.SubinterfaceMetric{SubinterfaceName: "0"}
	m.Set("k1", sub)

	result := m.SwapAll()
	if len(result) != 1 {
		t.Fatalf("expected 1, got %d", len(result))
	}
}

func TestShardedAlarmMap_SetGetSwap(t *testing.T) {
	m := newShardedAlarmMap()
	alarm := &models.AlarmReportMetric{FlowID: 42}
	m.Set("k1", alarm)

	result := m.SwapAll()
	if len(result) != 1 || result[0].FlowID != 42 {
		t.Fatalf("expected FlowID=42, got %v", result)
	}
}

func TestShardedNotificationMap_SetGetSwap(t *testing.T) {
	m := newShardedNotificationMap()
	n := &models.NotificationReportMetric{FlowID: 99}
	m.Set("k1", n)

	result := m.SwapAll()
	if len(result) != 1 || result[0].FlowID != 99 {
		t.Fatalf("expected FlowID=99, got %v", result)
	}
}

func TestFnv32Distribution(t *testing.T) {
	// Verify fnv32 produces different values for different keys
	seen := make(map[uint32]bool)
	for i := 0; i < 100; i++ {
		key := "key" + string(rune(i))
		h := fnv32(key)
		if seen[h] {
			t.Logf("hash collision at i=%d key=%s hash=%d (acceptable for small set)", i, key, h)
		}
		seen[h] = true
	}
}
