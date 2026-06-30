package buffer

import (
	"hash/fnv"
	"sync"

	"github.com/wwswwsuns/ztelem/internal/models"
)

const defaultShardCount = 16

// ShardedPlatformMap 分片的平台指标缓冲区
type ShardedPlatformMap struct {
	shards    []*platformShard
	shardMask uint32
}

type platformShard struct {
	mu    sync.RWMutex
	items map[string]*models.PlatformMetric
}

// ShardedInterfaceMap 分片的接口指标缓冲区
type ShardedInterfaceMap struct {
	shards    []*interfaceShard
	shardMask uint32
}

type interfaceShard struct {
	mu    sync.RWMutex
	items map[string]*models.InterfaceMetric
}

// ShardedSubinterfaceMap 分片的子接口指标缓冲区
type ShardedSubinterfaceMap struct {
	shards    []*subinterfaceShard
	shardMask uint32
}

type subinterfaceShard struct {
	mu    sync.RWMutex
	items map[string]*models.SubinterfaceMetric
}

// ShardedAlarmMap 分片的告警缓冲区
type ShardedAlarmMap struct {
	shards    []*alarmShard
	shardMask uint32
}

type alarmShard struct {
	mu    sync.RWMutex
	items map[string]*models.AlarmReportMetric
}

// ShardedNotificationMap 分片的通知缓冲区
type ShardedNotificationMap struct {
	shards    []*notificationShard
	shardMask uint32
}

type notificationShard struct {
	mu    sync.RWMutex
	items map[string]*models.NotificationReportMetric
}

func fnv32(s string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(s))
	return h.Sum32()
}

func newShardedPlatformMap() *ShardedPlatformMap {
	m := &ShardedPlatformMap{shardMask: defaultShardCount - 1}
	m.shards = make([]*platformShard, defaultShardCount)
	for i := range m.shards {
		m.shards[i] = &platformShard{items: make(map[string]*models.PlatformMetric)}
	}
	return m
}

func (m *ShardedPlatformMap) getShard(key string) *platformShard {
	return m.shards[fnv32(key)&m.shardMask]
}

func (m *ShardedPlatformMap) Get(key string) (*models.PlatformMetric, bool) {
	shard := m.getShard(key)
	shard.mu.RLock()
	v, ok := shard.items[key]
	shard.mu.RUnlock()
	return v, ok
}

func (m *ShardedPlatformMap) Set(key string, val *models.PlatformMetric) {
	shard := m.getShard(key)
	shard.mu.Lock()
	shard.items[key] = val
	shard.mu.Unlock()
}

func (m *ShardedPlatformMap) Swap(key string) *models.PlatformMetric {
	shard := m.getShard(key)
	shard.mu.Lock()
	old := shard.items[key]
	delete(shard.items, key)
	shard.mu.Unlock()
	return old
}

func (m *ShardedPlatformMap) Len() int {
	total := 0
	for _, shard := range m.shards {
		shard.mu.RLock()
		total += len(shard.items)
		shard.mu.RUnlock()
	}
	return total
}

// SwapAll 清空所有分片并返回旧数据（持有各分片锁的时间最短）
func (m *ShardedPlatformMap) SwapAll() []models.PlatformMetric {
	var result []models.PlatformMetric
	for _, shard := range m.shards {
		shard.mu.Lock()
		if len(shard.items) > 0 {
			for _, v := range shard.items {
				result = append(result, *v)
			}
			shard.items = make(map[string]*models.PlatformMetric)
		}
		shard.mu.Unlock()
	}
	return result
}

// --- Interface ---

func newShardedInterfaceMap() *ShardedInterfaceMap {
	m := &ShardedInterfaceMap{shardMask: defaultShardCount - 1}
	m.shards = make([]*interfaceShard, defaultShardCount)
	for i := range m.shards {
		m.shards[i] = &interfaceShard{items: make(map[string]*models.InterfaceMetric)}
	}
	return m
}

func (m *ShardedInterfaceMap) getShard(key string) *interfaceShard {
	return m.shards[fnv32(key)&m.shardMask]
}

func (m *ShardedInterfaceMap) Get(key string) (*models.InterfaceMetric, bool) {
	shard := m.getShard(key)
	shard.mu.RLock()
	v, ok := shard.items[key]
	shard.mu.RUnlock()
	return v, ok
}

func (m *ShardedInterfaceMap) Set(key string, val *models.InterfaceMetric) {
	shard := m.getShard(key)
	shard.mu.Lock()
	shard.items[key] = val
	shard.mu.Unlock()
}

func (m *ShardedInterfaceMap) Len() int {
	total := 0
	for _, shard := range m.shards {
		shard.mu.RLock()
		total += len(shard.items)
		shard.mu.RUnlock()
	}
	return total
}

func (m *ShardedInterfaceMap) SwapAll() []models.InterfaceMetric {
	var result []models.InterfaceMetric
	for _, shard := range m.shards {
		shard.mu.Lock()
		if len(shard.items) > 0 {
			for _, v := range shard.items {
				result = append(result, *v)
			}
			shard.items = make(map[string]*models.InterfaceMetric)
		}
		shard.mu.Unlock()
	}
	return result
}

// --- Subinterface ---

func newShardedSubinterfaceMap() *ShardedSubinterfaceMap {
	m := &ShardedSubinterfaceMap{shardMask: defaultShardCount - 1}
	m.shards = make([]*subinterfaceShard, defaultShardCount)
	for i := range m.shards {
		m.shards[i] = &subinterfaceShard{items: make(map[string]*models.SubinterfaceMetric)}
	}
	return m
}

func (m *ShardedSubinterfaceMap) getShard(key string) *subinterfaceShard {
	return m.shards[fnv32(key)&m.shardMask]
}

func (m *ShardedSubinterfaceMap) Get(key string) (*models.SubinterfaceMetric, bool) {
	shard := m.getShard(key)
	shard.mu.RLock()
	v, ok := shard.items[key]
	shard.mu.RUnlock()
	return v, ok
}

func (m *ShardedSubinterfaceMap) Set(key string, val *models.SubinterfaceMetric) {
	shard := m.getShard(key)
	shard.mu.Lock()
	shard.items[key] = val
	shard.mu.Unlock()
}

func (m *ShardedSubinterfaceMap) Len() int {
	total := 0
	for _, shard := range m.shards {
		shard.mu.RLock()
		total += len(shard.items)
		shard.mu.RUnlock()
	}
	return total
}

func (m *ShardedSubinterfaceMap) SwapAll() []models.SubinterfaceMetric {
	var result []models.SubinterfaceMetric
	for _, shard := range m.shards {
		shard.mu.Lock()
		if len(shard.items) > 0 {
			for _, v := range shard.items {
				result = append(result, *v)
			}
			shard.items = make(map[string]*models.SubinterfaceMetric)
		}
		shard.mu.Unlock()
	}
	return result
}

// --- Alarm ---

func newShardedAlarmMap() *ShardedAlarmMap {
	m := &ShardedAlarmMap{shardMask: defaultShardCount - 1}
	m.shards = make([]*alarmShard, defaultShardCount)
	for i := range m.shards {
		m.shards[i] = &alarmShard{items: make(map[string]*models.AlarmReportMetric)}
	}
	return m
}

func (m *ShardedAlarmMap) getShard(key string) *alarmShard {
	return m.shards[fnv32(key)&m.shardMask]
}

func (m *ShardedAlarmMap) Set(key string, val *models.AlarmReportMetric) {
	shard := m.getShard(key)
	shard.mu.Lock()
	shard.items[key] = val
	shard.mu.Unlock()
}

func (m *ShardedAlarmMap) Len() int {
	total := 0
	for _, shard := range m.shards {
		shard.mu.RLock()
		total += len(shard.items)
		shard.mu.RUnlock()
	}
	return total
}

func (m *ShardedAlarmMap) SwapAll() []models.AlarmReportMetric {
	var result []models.AlarmReportMetric
	for _, shard := range m.shards {
		shard.mu.Lock()
		if len(shard.items) > 0 {
			for _, v := range shard.items {
				result = append(result, *v)
			}
			shard.items = make(map[string]*models.AlarmReportMetric)
		}
		shard.mu.Unlock()
	}
	return result
}

// --- Notification ---

func newShardedNotificationMap() *ShardedNotificationMap {
	m := &ShardedNotificationMap{shardMask: defaultShardCount - 1}
	m.shards = make([]*notificationShard, defaultShardCount)
	for i := range m.shards {
		m.shards[i] = &notificationShard{items: make(map[string]*models.NotificationReportMetric)}
	}
	return m
}

func (m *ShardedNotificationMap) getShard(key string) *notificationShard {
	return m.shards[fnv32(key)&m.shardMask]
}

func (m *ShardedNotificationMap) Set(key string, val *models.NotificationReportMetric) {
	shard := m.getShard(key)
	shard.mu.Lock()
	shard.items[key] = val
	shard.mu.Unlock()
}

func (m *ShardedNotificationMap) Len() int {
	total := 0
	for _, shard := range m.shards {
		shard.mu.RLock()
		total += len(shard.items)
		shard.mu.RUnlock()
	}
	return total
}

func (m *ShardedNotificationMap) SwapAll() []models.NotificationReportMetric {
	var result []models.NotificationReportMetric
	for _, shard := range m.shards {
		shard.mu.Lock()
		if len(shard.items) > 0 {
			for _, v := range shard.items {
				result = append(result, *v)
			}
			shard.items = make(map[string]*models.NotificationReportMetric)
		}
		shard.mu.Unlock()
	}
	return result
}
