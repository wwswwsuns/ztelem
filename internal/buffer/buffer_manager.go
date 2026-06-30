package buffer

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/wwswwsuns/ztelem/internal/config"
	"github.com/wwswwsuns/ztelem/internal/models"
	"github.com/sirupsen/logrus"
)

type DatabaseWriter interface {
	BatchInsertPlatformMetrics([]models.PlatformMetric) error
	BatchInsertInterfaceMetrics([]models.InterfaceMetric) error
	BatchInsertSubinterfaceMetrics([]models.SubinterfaceMetric) error
}

type FixedBufferStats struct {
	PlatformBufferSize           int
	InterfaceBufferSize          int
	SubinterfaceBufferSize       int
	AlarmReportBufferSize        int
	NotificationReportBufferSize int
	TotalRecordsProcessed        int64
	TotalRecordsWritten          int64
	TotalErrors                  int64
	LastFlushTime                time.Time
	FlushDuration                time.Duration
	KeyCollisions                int64
}

type DatabaseInterface interface {
	BatchInsertPlatformMetrics(data []models.PlatformMetric) error
	BatchInsertInterfaceMetrics(data []models.InterfaceMetric) error
	BatchInsertSubinterfaceMetrics(data []models.SubinterfaceMetric) error
	BatchInsertAlarmReportMetrics(data []models.AlarmReportMetric) error
	BatchInsertNotificationReportMetrics(data []models.NotificationReportMetric) error
}

// FixedBufferManager 缓冲区管理器（分片锁 + 零分配聚合键）
type FixedBufferManager struct {
	db           DatabaseInterface
	config       config.BufferConfig
	writerConfig config.DatabaseWriterConfig
	logger       *logrus.Logger

	// 分片缓冲区 — 16 分片，每片独立锁
	platformBuffer    *ShardedPlatformMap
	interfaceBuffer   *ShardedInterfaceMap
	subinterfaceBuffer *ShardedSubinterfaceMap
	alarmReportBuffer *ShardedAlarmMap
	notificationReportBuffer *ShardedNotificationMap

	// 统计信息
	stats      FixedBufferStats
	statsMutex sync.RWMutex

	// 聚合键构建 buffer（每 goroutine 本地复用）
	keyBuf sync.Pool

	// 定时器和控制
	flushTimer *time.Timer
	stopChan   chan struct{}

	// 写入通道
	platformWriteChan           chan []models.PlatformMetric
	interfaceWriteChan          chan []models.InterfaceMetric
	subinterfaceWriteChan       chan []models.SubinterfaceMetric
	alarmReportWriteChan        chan []models.AlarmReportMetric
	notificationReportWriteChan chan []models.NotificationReportMetric
}

// keyBuffer 聚合键字节构建器，复用避免分配
type keyBuffer struct {
	buf []byte
}

func (kb *keyBuffer) reset() {
	kb.buf = kb.buf[:0]
}

func (kb *keyBuffer) writeInt(v int64) {
	kb.buf = strconv.AppendInt(kb.buf, v, 10)
}

func (kb *keyBuffer) writeByte(b byte) {
	kb.buf = append(kb.buf, b)
}

func (kb *keyBuffer) writeString(s string) {
	kb.buf = append(kb.buf, s...)
}

func (kb *keyBuffer) string() string {
	return string(kb.buf)
}

func NewFixedBufferManager(db DatabaseInterface, cfg config.BufferConfig, writerConfig config.DatabaseWriterConfig, logger *logrus.Logger) *FixedBufferManager {
	bm := &FixedBufferManager{
		db:                       db,
		config:                   cfg,
		writerConfig:             writerConfig,
		logger:                   logger,
		platformBuffer:           newShardedPlatformMap(),
		interfaceBuffer:          newShardedInterfaceMap(),
		subinterfaceBuffer:       newShardedSubinterfaceMap(),
		alarmReportBuffer:        newShardedAlarmMap(),
		notificationReportBuffer: newShardedNotificationMap(),
		stopChan:                 make(chan struct{}),
		keyBuf: sync.Pool{
			New: func() interface{} { return &keyBuffer{buf: make([]byte, 0, 128)} },
		},
		platformWriteChan:        make(chan []models.PlatformMetric, 100),
		interfaceWriteChan:       make(chan []models.InterfaceMetric, 100),
		subinterfaceWriteChan:    make(chan []models.SubinterfaceMetric, 100),
		alarmReportWriteChan:     make(chan []models.AlarmReportMetric, 100),
		notificationReportWriteChan: make(chan []models.NotificationReportMetric, 100),
	}

	bm.startFlushTimer()
	bm.startParallelWriters()

	return bm
}

// acquireKeyBuf 从 pool 获取 keyBuffer
func (bm *FixedBufferManager) acquireKeyBuf() *keyBuffer {
	return bm.keyBuf.Get().(*keyBuffer)
}

func (bm *FixedBufferManager) releaseKeyBuf(kb *keyBuffer) {
	kb.reset()
	bm.keyBuf.Put(kb)
}

// generatePlatformKey 零分配聚合键：秒级时间戳_系统ID_组件名
func (bm *FixedBufferManager) generatePlatformKey(metric *models.PlatformMetric) string {
	kb := bm.acquireKeyBuf()
	defer bm.releaseKeyBuf(kb)
	kb.writeInt(metric.Timestamp.Truncate(time.Second).Unix())
	kb.writeByte('_')
	kb.writeString(metric.SystemID)
	kb.writeByte('_')
	kb.writeString(metric.ComponentName)
	return kb.string()
}

func (bm *FixedBufferManager) generateInterfaceKey(metric *models.InterfaceMetric) string {
	kb := bm.acquireKeyBuf()
	defer bm.releaseKeyBuf(kb)
	kb.writeInt(metric.Timestamp.Truncate(time.Second).Unix())
	kb.writeByte('_')
	kb.writeString(metric.SystemID)
	kb.writeByte('_')
	kb.writeString(metric.InterfaceName)
	return kb.string()
}

func (bm *FixedBufferManager) generateSubinterfaceKey(metric *models.SubinterfaceMetric) string {
	kb := bm.acquireKeyBuf()
	defer bm.releaseKeyBuf(kb)
	kb.writeInt(metric.Timestamp.Truncate(time.Second).Unix())
	kb.writeByte('_')
	kb.writeString(metric.SystemID)
	kb.writeByte('_')
	kb.writeString(metric.InterfaceName)
	kb.writeByte('_')
	kb.writeString(metric.SubinterfaceName)
	return kb.string()
}

func (bm *FixedBufferManager) generateAlarmKey(metric *models.AlarmReportMetric) string {
	kb := bm.acquireKeyBuf()
	defer bm.releaseKeyBuf(kb)
	kb.writeString(metric.SystemID)
	kb.writeByte(':')
	kb.writeInt(int64(metric.FlowID))
	kb.writeByte(':')
	kb.writeInt(int64(metric.AlarmTimestamp))
	return kb.string()
}

func (bm *FixedBufferManager) generateNotificationKey(metric *models.NotificationReportMetric) string {
	kb := bm.acquireKeyBuf()
	defer bm.releaseKeyBuf(kb)
	kb.writeString(metric.SystemID)
	kb.writeByte(':')
	kb.writeInt(int64(metric.FlowID))
	kb.writeByte(':')
	kb.writeInt(int64(metric.NotificationTimestamp))
	return kb.string()
}

// AddPlatformMetrics 添加平台指标数据（分片锁，无全局互斥）
func (bm *FixedBufferManager) AddPlatformMetrics(metrics []models.PlatformMetric) error {
	for i := range metrics {
		key := bm.generatePlatformKey(&metrics[i])
		shard := bm.platformBuffer.getShard(key)
		shard.mu.Lock()

		if existing, exists := shard.items[key]; exists {
			bm.mergePlatformMetric(existing, &metrics[i])
			bm.statsMutex.Lock()
			bm.stats.KeyCollisions++
			bm.statsMutex.Unlock()
		} else {
			metricCopy := metrics[i]
			shard.items[key] = &metricCopy
		}
		shard.mu.Unlock()
	}

	atomic.AddInt64(&bm.stats.TotalRecordsProcessed, int64(len(metrics)))

	if bm.platformBuffer.Len() >= bm.config.FlushThreshold {
		go bm.FlushPlatformMetrics()
	}

	return nil
}

func (bm *FixedBufferManager) AddInterfaceMetrics(metrics []models.InterfaceMetric) error {
	for i := range metrics {
		key := bm.generateInterfaceKey(&metrics[i])
		shard := bm.interfaceBuffer.getShard(key)
		shard.mu.Lock()

		if existing, exists := shard.items[key]; exists {
			bm.mergeInterfaceMetric(existing, &metrics[i])
			bm.statsMutex.Lock()
			bm.stats.KeyCollisions++
			bm.statsMutex.Unlock()
		} else {
			metricCopy := metrics[i]
			shard.items[key] = &metricCopy
		}
		shard.mu.Unlock()
	}

	atomic.AddInt64(&bm.stats.TotalRecordsProcessed, int64(len(metrics)))

	if bm.interfaceBuffer.Len() >= bm.config.FlushThreshold {
		go bm.FlushInterfaceMetrics()
	}

	return nil
}

func (bm *FixedBufferManager) AddSubinterfaceMetrics(metrics []models.SubinterfaceMetric) error {
	for i := range metrics {
		key := bm.generateSubinterfaceKey(&metrics[i])
		shard := bm.subinterfaceBuffer.getShard(key)
		shard.mu.Lock()

		if existing, exists := shard.items[key]; exists {
			bm.mergeSubinterfaceMetric(existing, &metrics[i])
			bm.statsMutex.Lock()
			bm.stats.KeyCollisions++
			bm.statsMutex.Unlock()
		} else {
			metricCopy := metrics[i]
			shard.items[key] = &metricCopy
		}
		shard.mu.Unlock()
	}

	atomic.AddInt64(&bm.stats.TotalRecordsProcessed, int64(len(metrics)))

	if bm.subinterfaceBuffer.Len() >= bm.config.FlushThreshold {
		go bm.FlushSubinterfaceMetrics()
	}

	return nil
}

func (bm *FixedBufferManager) AddAlarmReportMetrics(metrics []models.AlarmReportMetric) error {
	for i := range metrics {
		key := bm.generateAlarmKey(&metrics[i])
		metricCopy := metrics[i]
		bm.alarmReportBuffer.Set(key, &metricCopy)
	}

	atomic.AddInt64(&bm.stats.TotalRecordsProcessed, int64(len(metrics)))

	if bm.alarmReportBuffer.Len() >= bm.config.FlushThreshold {
		go bm.FlushAlarmReportMetrics()
	}

	return nil
}

func (bm *FixedBufferManager) AddNotificationReportMetrics(metrics []models.NotificationReportMetric) error {
	for i := range metrics {
		key := bm.generateNotificationKey(&metrics[i])
		metricCopy := metrics[i]
		bm.notificationReportBuffer.Set(key, &metricCopy)
	}

	atomic.AddInt64(&bm.stats.TotalRecordsProcessed, int64(len(metrics)))

	if bm.notificationReportBuffer.Len() >= bm.config.FlushThreshold {
		go bm.FlushNotificationReportMetrics()
	}

	return nil
}

func (bm *FixedBufferManager) mergePlatformMetric(existing, new *models.PlatformMetric) {
	if new.CommonState != nil {
		if existing.CommonState == nil {
			existing.CommonState = &models.CommonState{}
		}
		if new.OperStatus != nil {
			existing.OperStatus = new.OperStatus
		}
		if new.Uptime != nil {
			existing.Uptime = new.Uptime
		}
		if new.UsedPower != nil {
			existing.UsedPower = new.UsedPower
		}
		if new.AllocatedPower != nil {
			existing.AllocatedPower = new.AllocatedPower
		}
	}
	if new.MemData != nil && new.MemAlarmStatus != nil {
		if existing.MemData == nil {
			existing.MemData = &models.MemData{}
		}
		existing.MemAlarmStatus = new.MemAlarmStatus
	}
	if new.CPUData != nil && new.CPUAlarmStatus != nil {
		if existing.CPUData == nil {
			existing.CPUData = &models.CPUData{}
		}
		existing.CPUAlarmStatus = new.CPUAlarmStatus
	}
	if new.Timestamp.After(existing.Timestamp) {
		existing.Timestamp = new.Timestamp
	}
}

func (bm *FixedBufferManager) mergeInterfaceMetric(existing, new *models.InterfaceMetric) {
	if new.AdminStatusStr != nil {
		existing.AdminStatusStr = new.AdminStatusStr
	}
	if new.OperStatusStr != nil {
		existing.OperStatusStr = new.OperStatusStr
	}
	if new.PhyStatusStr != nil {
		existing.PhyStatusStr = new.PhyStatusStr
	}
	if new.InOctets != nil {
		existing.InOctets = new.InOctets
	}
	if new.OutOctets != nil {
		existing.OutOctets = new.OutOctets
	}
	if new.Timestamp.After(existing.Timestamp) {
		existing.Timestamp = new.Timestamp
	}
}

func (bm *FixedBufferManager) mergeSubinterfaceMetric(existing, new *models.SubinterfaceMetric) {
	if new.AdminStatusStr != nil {
		existing.AdminStatusStr = new.AdminStatusStr
	}
	if new.OperStatusStr != nil {
		existing.OperStatusStr = new.OperStatusStr
	}
	if new.InOctets != nil {
		existing.InOctets = new.InOctets
	}
	if new.OutOctets != nil {
		existing.OutOctets = new.OutOctets
	}
	if new.Timestamp.After(existing.Timestamp) {
		existing.Timestamp = new.Timestamp
	}
}

func (bm *FixedBufferManager) startParallelWriters() {
	for i := 0; i < bm.writerConfig.ParallelWriters; i++ {
		go bm.platformWriter()
	}
	for i := 0; i < bm.writerConfig.ParallelWriters; i++ {
		go bm.interfaceWriter()
	}
	for i := 0; i < bm.writerConfig.ParallelWriters; i++ {
		go bm.subinterfaceWriter()
	}
	for i := 0; i < bm.writerConfig.ParallelWriters; i++ {
		go bm.alarmReportWriter()
	}
	for i := 0; i < bm.writerConfig.ParallelWriters; i++ {
		go bm.notificationReportWriter()
	}
}

func (bm *FixedBufferManager) platformWriter() {
	for {
		select {
		case batch := <-bm.platformWriteChan:
			if err := bm.writeWithRetry(func() error {
				return bm.db.BatchInsertPlatformMetrics(batch)
			}); err != nil {
				bm.logger.Errorf("平台指标写入失败: %v", err)
				atomic.AddInt64(&bm.stats.TotalErrors, 1)
			} else {
				atomic.AddInt64(&bm.stats.TotalRecordsWritten, int64(len(batch)))
			}
		case <-bm.stopChan:
			return
		}
	}
}

func (bm *FixedBufferManager) interfaceWriter() {
	for {
		select {
		case batch := <-bm.interfaceWriteChan:
			if err := bm.writeWithRetry(func() error {
				return bm.db.BatchInsertInterfaceMetrics(batch)
			}); err != nil {
				bm.logger.Errorf("接口指标写入失败: %v", err)
				atomic.AddInt64(&bm.stats.TotalErrors, 1)
			} else {
				atomic.AddInt64(&bm.stats.TotalRecordsWritten, int64(len(batch)))
			}
		case <-bm.stopChan:
			return
		}
	}
}

func (bm *FixedBufferManager) subinterfaceWriter() {
	for {
		select {
		case batch := <-bm.subinterfaceWriteChan:
			if err := bm.writeWithRetry(func() error {
				return bm.db.BatchInsertSubinterfaceMetrics(batch)
			}); err != nil {
				bm.logger.Errorf("子接口指标写入失败: %v", err)
				atomic.AddInt64(&bm.stats.TotalErrors, 1)
			} else {
				atomic.AddInt64(&bm.stats.TotalRecordsWritten, int64(len(batch)))
			}
		case <-bm.stopChan:
			return
		}
	}
}

func (bm *FixedBufferManager) alarmReportWriter() {
	for {
		select {
		case batch := <-bm.alarmReportWriteChan:
			if err := bm.writeWithRetry(func() error {
				return bm.db.BatchInsertAlarmReportMetrics(batch)
			}); err != nil {
				bm.logger.Errorf("告警上报写入失败: %v", err)
				atomic.AddInt64(&bm.stats.TotalErrors, 1)
			} else {
				atomic.AddInt64(&bm.stats.TotalRecordsWritten, int64(len(batch)))
			}
		case <-bm.stopChan:
			return
		}
	}
}

func (bm *FixedBufferManager) notificationReportWriter() {
	for {
		select {
		case batch := <-bm.notificationReportWriteChan:
			if err := bm.writeWithRetry(func() error {
				return bm.db.BatchInsertNotificationReportMetrics(batch)
			}); err != nil {
				bm.logger.Errorf("通知上报写入失败: %v", err)
				atomic.AddInt64(&bm.stats.TotalErrors, 1)
			} else {
				atomic.AddInt64(&bm.stats.TotalRecordsWritten, int64(len(batch)))
			}
		case <-bm.stopChan:
			return
		}
	}
}

func (bm *FixedBufferManager) writeWithRetry(writeFunc func() error) error {
	var lastErr error

	for attempt := 0; attempt < bm.writerConfig.RetryAttempts; attempt++ {
		if attempt > 0 {
			time.Sleep(bm.writerConfig.RetryDelay)
			bm.logger.Debugf("重试写入，第 %d 次尝试", attempt+1)
		}

		ctx, cancel := context.WithTimeout(context.Background(), bm.writerConfig.BatchTimeout)

		done := make(chan error, 1)
		go func() {
			done <- writeFunc()
		}()

		select {
		case err := <-done:
			cancel()
			if err == nil {
				return nil
			}
			lastErr = err
			bm.logger.Warnf("写入失败 (尝试 %d/%d): %v", attempt+1, bm.writerConfig.RetryAttempts, err)
		case <-ctx.Done():
			cancel()
			lastErr = fmt.Errorf("写入超时")
			bm.logger.Warnf("写入超时 (尝试 %d/%d)", attempt+1, bm.writerConfig.RetryAttempts)
		}
	}

	return fmt.Errorf("写入失败，已重试 %d 次: %v", bm.writerConfig.RetryAttempts, lastErr)
}

func (bm *FixedBufferManager) FlushAll() error {
	start := time.Now()
	var errs []error

	var wg sync.WaitGroup
	errChan := make(chan error, 5)

	wg.Add(5)

	go func() {
		defer wg.Done()
		if err := bm.FlushPlatformMetrics(); err != nil {
			errChan <- fmt.Errorf("平台指标刷新失败: %v", err)
		}
	}()

	go func() {
		defer wg.Done()
		if err := bm.FlushInterfaceMetrics(); err != nil {
			errChan <- fmt.Errorf("接口指标刷新失败: %v", err)
		}
	}()

	go func() {
		defer wg.Done()
		if err := bm.FlushSubinterfaceMetrics(); err != nil {
			errChan <- fmt.Errorf("子接口指标刷新失败: %v", err)
		}
	}()

	go func() {
		defer wg.Done()
		if err := bm.FlushAlarmReportMetrics(); err != nil {
			errChan <- fmt.Errorf("告警上报刷新失败: %v", err)
		}
	}()

	go func() {
		defer wg.Done()
		if err := bm.FlushNotificationReportMetrics(); err != nil {
			errChan <- fmt.Errorf("通知上报刷新失败: %v", err)
		}
	}()

	wg.Wait()
	close(errChan)

	for err := range errChan {
		errs = append(errs, err)
	}

	bm.statsMutex.Lock()
	bm.stats.LastFlushTime = time.Now()
	bm.stats.FlushDuration = time.Since(start)
	bm.statsMutex.Unlock()

	if len(errs) > 0 {
		return errs[0]
	}

	return nil
}

func (bm *FixedBufferManager) FlushPlatformMetrics() error {
	metrics := bm.platformBuffer.SwapAll()
	if len(metrics) == 0 {
		return nil
	}

	batchSize := bm.writerConfig.MaxBatchSize
	for i := 0; i < len(metrics); i += batchSize {
		end := i + batchSize
		if end > len(metrics) {
			end = len(metrics)
		}
		batch := metrics[i:end]
		select {
		case bm.platformWriteChan <- batch:
		default:
			if err := bm.writeWithRetry(func() error {
				return bm.db.BatchInsertPlatformMetrics(batch)
			}); err != nil {
				return err
			}
		}
	}
	return nil
}

func (bm *FixedBufferManager) FlushInterfaceMetrics() error {
	metrics := bm.interfaceBuffer.SwapAll()
	if len(metrics) == 0 {
		return nil
	}

	batchSize := bm.writerConfig.MaxBatchSize
	for i := 0; i < len(metrics); i += batchSize {
		end := i + batchSize
		if end > len(metrics) {
			end = len(metrics)
		}
		batch := metrics[i:end]
		select {
		case bm.interfaceWriteChan <- batch:
		default:
			if err := bm.writeWithRetry(func() error {
				return bm.db.BatchInsertInterfaceMetrics(batch)
			}); err != nil {
				return err
			}
		}
	}
	return nil
}

func (bm *FixedBufferManager) FlushSubinterfaceMetrics() error {
	metrics := bm.subinterfaceBuffer.SwapAll()
	if len(metrics) == 0 {
		return nil
	}

	batchSize := bm.writerConfig.MaxBatchSize
	for i := 0; i < len(metrics); i += batchSize {
		end := i + batchSize
		if end > len(metrics) {
			end = len(metrics)
		}
		batch := metrics[i:end]
		select {
		case bm.subinterfaceWriteChan <- batch:
		default:
			if err := bm.writeWithRetry(func() error {
				return bm.db.BatchInsertSubinterfaceMetrics(batch)
			}); err != nil {
				return err
			}
		}
	}
	return nil
}

func (bm *FixedBufferManager) FlushAlarmReportMetrics() error {
	metrics := bm.alarmReportBuffer.SwapAll()
	if len(metrics) == 0 {
		return nil
	}

	batchSize := bm.writerConfig.MaxBatchSize
	for i := 0; i < len(metrics); i += batchSize {
		end := i + batchSize
		if end > len(metrics) {
			end = len(metrics)
		}
		batch := metrics[i:end]
		select {
		case bm.alarmReportWriteChan <- batch:
		default:
			if err := bm.writeWithRetry(func() error {
				return bm.db.BatchInsertAlarmReportMetrics(batch)
			}); err != nil {
				return err
			}
		}
	}
	return nil
}

func (bm *FixedBufferManager) FlushNotificationReportMetrics() error {
	metrics := bm.notificationReportBuffer.SwapAll()
	if len(metrics) == 0 {
		return nil
	}

	batchSize := bm.writerConfig.MaxBatchSize
	for i := 0; i < len(metrics); i += batchSize {
		end := i + batchSize
		if end > len(metrics) {
			end = len(metrics)
		}
		batch := metrics[i:end]
		select {
		case bm.notificationReportWriteChan <- batch:
		default:
			if err := bm.writeWithRetry(func() error {
				return bm.db.BatchInsertNotificationReportMetrics(batch)
			}); err != nil {
				return err
			}
		}
	}
	return nil
}

func (bm *FixedBufferManager) startFlushTimer() {
	bm.flushTimer = time.NewTimer(bm.config.FlushInterval)

	go func() {
		for {
			select {
			case <-bm.flushTimer.C:
				bm.logger.Debug("定时刷新缓冲区")
				bm.FlushAll()
				bm.flushTimer.Reset(bm.config.FlushInterval)
			case <-bm.stopChan:
				bm.flushTimer.Stop()
				return
			}
		}
	}()
}

func (bm *FixedBufferManager) Stop() error {
	select {
	case <-bm.stopChan:
		return nil
	default:
		close(bm.stopChan)
	}

	return bm.FlushAll()
}

func (bm *FixedBufferManager) GetStats() FixedBufferStats {
	bm.statsMutex.RLock()
	stats := bm.stats
	bm.statsMutex.RUnlock()

	stats.PlatformBufferSize = bm.platformBuffer.Len()
	stats.InterfaceBufferSize = bm.interfaceBuffer.Len()
	stats.SubinterfaceBufferSize = bm.subinterfaceBuffer.Len()
	stats.AlarmReportBufferSize = bm.alarmReportBuffer.Len()
	stats.NotificationReportBufferSize = bm.notificationReportBuffer.Len()

	return stats
}
