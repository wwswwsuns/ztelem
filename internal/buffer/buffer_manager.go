package buffer

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/wwswwsuns/ztelem/internal/config"
	"github.com/wwswwsuns/ztelem/internal/models"
	"github.com/sirupsen/logrus"
)

// FixedBufferManager 修复数据丢失问题的缓冲区管理器
type FixedBufferManager struct {
	// 数据库接口
	db DatabaseInterface

	// 配置
	config       config.BufferConfig
	writerConfig config.DatabaseWriterConfig
	logger       *logrus.Logger

	// 分类缓冲区 - 使用精确的聚合键
	platformBuffer           map[string]*models.PlatformMetric
	interfaceBuffer          map[string]*models.InterfaceMetric
	subinterfaceBuffer       map[string]*models.SubinterfaceMetric
	alarmReportBuffer        map[string]*models.AlarmReportMetric
	notificationReportBuffer map[string]*models.NotificationReportMetric

	// 互斥锁
	platformMutex           sync.RWMutex
	interfaceMutex          sync.RWMutex
	subinterfaceMutex       sync.RWMutex
	alarmReportMutex        sync.RWMutex
	notificationReportMutex sync.RWMutex

	// 统计信息
	stats      FixedBufferStats
	statsMutex sync.RWMutex

	// 定时器和控制
	flushTimer *time.Timer
	stopChan   chan struct{}

	// 写入通道 - 用于并行写入
	platformWriteChan           chan []models.PlatformMetric
	interfaceWriteChan          chan []models.InterfaceMetric
	subinterfaceWriteChan       chan []models.SubinterfaceMetric
	alarmReportWriteChan        chan []models.AlarmReportMetric
	notificationReportWriteChan chan []models.NotificationReportMetric
}

// 在buffer_manager.go中，将数据库接口改为支持pgx
// 或者创建一个适配器接口
type DatabaseWriter interface {
    BatchInsertPlatformMetrics([]models.PlatformMetric) error
    BatchInsertInterfaceMetrics([]models.InterfaceMetric) error
    BatchInsertSubinterfaceMetrics([]models.SubinterfaceMetric) error
}
// FixedBufferStats 修复版缓冲区统计信息
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
	KeyCollisions                int64  // 聚合键冲突计数
}

// DatabaseInterface 数据库接口
type DatabaseInterface interface {
	BatchInsertPlatformMetrics(data []models.PlatformMetric) error
	BatchInsertInterfaceMetrics(data []models.InterfaceMetric) error
	BatchInsertSubinterfaceMetrics(data []models.SubinterfaceMetric) error
	BatchInsertAlarmReportMetrics(data []models.AlarmReportMetric) error
	BatchInsertNotificationReportMetrics(data []models.NotificationReportMetric) error
}

// NewFixedBufferManager 创建修复版缓冲区管理器
func NewFixedBufferManager(db DatabaseInterface, config config.BufferConfig, writerConfig config.DatabaseWriterConfig, logger *logrus.Logger) *FixedBufferManager {
	bm := &FixedBufferManager{
		db:                       db,
		config:                   config,
		writerConfig:             writerConfig,
		logger:                   logger,
		platformBuffer:           make(map[string]*models.PlatformMetric),
		interfaceBuffer:          make(map[string]*models.InterfaceMetric),
		subinterfaceBuffer:       make(map[string]*models.SubinterfaceMetric),
		alarmReportBuffer:        make(map[string]*models.AlarmReportMetric),
		notificationReportBuffer: make(map[string]*models.NotificationReportMetric),
		stopChan:                 make(chan struct{}),
		platformWriteChan:        make(chan []models.PlatformMetric, 100),
		interfaceWriteChan:       make(chan []models.InterfaceMetric, 100),
		subinterfaceWriteChan:    make(chan []models.SubinterfaceMetric, 100),
		alarmReportWriteChan:     make(chan []models.AlarmReportMetric, 100),
		notificationReportWriteChan: make(chan []models.NotificationReportMetric, 100),
	}

	// 启动定时刷新
	bm.startFlushTimer()

	// 启动并行写入器
	bm.startParallelWriters()

	return bm
}

// AddPlatformMetrics 添加平台指标数据 - 修复版本
func (bm *FixedBufferManager) AddPlatformMetrics(metrics []models.PlatformMetric) error {
	bm.platformMutex.Lock()
	defer bm.platformMutex.Unlock()

	// 使用精确聚合键进行聚合
	for _, metric := range metrics {
		// 生成精确的聚合键
		key := bm.generatePrecisePlatformKey(&metric)
		
		if existing, exists := bm.platformBuffer[key]; exists {
			// 合并数据，保持数据完整性
			bm.mergePlatformMetric(existing, &metric)
			bm.logger.Debugf("合并平台指标数据: %s", key)
			
			bm.statsMutex.Lock()
			bm.stats.KeyCollisions++
			bm.statsMutex.Unlock()
		} else {
			// 创建新记录的副本
			metricCopy := metric
			bm.platformBuffer[key] = &metricCopy
		}
	}

	bm.statsMutex.Lock()
	bm.stats.TotalRecordsProcessed += int64(len(metrics))
	bm.statsMutex.Unlock()

	// 检查是否需要刷新
	if len(bm.platformBuffer) >= bm.config.FlushThreshold {
		go bm.FlushPlatformMetrics()
	}

	return nil
}

// AddInterfaceMetrics 添加接口指标数据 - 修复版本
func (bm *FixedBufferManager) AddInterfaceMetrics(metrics []models.InterfaceMetric) error {
	bm.interfaceMutex.Lock()
	defer bm.interfaceMutex.Unlock()

	// 使用精确聚合键进行聚合
	for _, metric := range metrics {
		// 生成精确的聚合键
		key := bm.generatePreciseInterfaceKey(&metric)
		
		if existing, exists := bm.interfaceBuffer[key]; exists {
			// 合并数据，保持数据完整性
			bm.mergeInterfaceMetric(existing, &metric)
			bm.logger.Debugf("合并接口指标数据: %s", key)
			
			bm.statsMutex.Lock()
			bm.stats.KeyCollisions++
			bm.statsMutex.Unlock()
		} else {
			// 创建新记录的副本
			metricCopy := metric
			bm.interfaceBuffer[key] = &metricCopy
		}
	}

	bm.statsMutex.Lock()
	bm.stats.TotalRecordsProcessed += int64(len(metrics))
	bm.statsMutex.Unlock()

	// 检查是否需要刷新
	if len(bm.interfaceBuffer) >= bm.config.FlushThreshold {
		go bm.FlushInterfaceMetrics()
	}

	return nil
}

// AddSubinterfaceMetrics 添加子接口指标数据 - 修复版本
func (bm *FixedBufferManager) AddSubinterfaceMetrics(metrics []models.SubinterfaceMetric) error {
	bm.subinterfaceMutex.Lock()
	defer bm.subinterfaceMutex.Unlock()

	// 使用精确聚合键进行聚合
	for _, metric := range metrics {
		// 生成精确的聚合键
		key := bm.generatePreciseSubinterfaceKey(&metric)
		
		if existing, exists := bm.subinterfaceBuffer[key]; exists {
			// 合并数据，保持数据完整性
			bm.mergeSubinterfaceMetric(existing, &metric)
			bm.logger.Debugf("合并子接口指标数据: %s", key)
			
			bm.statsMutex.Lock()
			bm.stats.KeyCollisions++
			bm.statsMutex.Unlock()
		} else {
			// 创建新记录的副本
			metricCopy := metric
			bm.subinterfaceBuffer[key] = &metricCopy
		}
	}

	bm.statsMutex.Lock()
	bm.stats.TotalRecordsProcessed += int64(len(metrics))
	bm.statsMutex.Unlock()

	// 检查是否需要刷新
	if len(bm.subinterfaceBuffer) >= bm.config.FlushThreshold {
		go bm.FlushSubinterfaceMetrics()
	}

	return nil
}

// AddAlarmReportMetrics 添加告警上报数据到缓冲区
func (bm *FixedBufferManager) AddAlarmReportMetrics(metrics []models.AlarmReportMetric) error {
	bm.alarmReportMutex.Lock()
	defer bm.alarmReportMutex.Unlock()

	for _, metric := range metrics {
		// 告警数据使用流水号作为唯一键，不进行聚合
		key := fmt.Sprintf("%s:%d:%d", metric.SystemID, metric.FlowID, metric.AlarmTimestamp)
		
		// 告警数据不聚合，直接存储
		metricCopy := metric
		bm.alarmReportBuffer[key] = &metricCopy
	}

	bm.statsMutex.Lock()
	bm.stats.TotalRecordsProcessed += int64(len(metrics))
	bm.statsMutex.Unlock()

	// 检查是否需要刷新
	if len(bm.alarmReportBuffer) >= bm.config.FlushThreshold {
		go bm.FlushAlarmReportMetrics()
	}

	return nil
}

// AddNotificationReportMetrics 添加通知上报数据到缓冲区
func (bm *FixedBufferManager) AddNotificationReportMetrics(metrics []models.NotificationReportMetric) error {
	bm.notificationReportMutex.Lock()
	defer bm.notificationReportMutex.Unlock()

	for _, metric := range metrics {
		// 通知数据使用流水号作为唯一键，不进行聚合
		key := fmt.Sprintf("%s:%d:%d", metric.SystemID, metric.FlowID, metric.NotificationTimestamp)
		
		// 通知数据不聚合，直接存储
		metricCopy := metric
		bm.notificationReportBuffer[key] = &metricCopy
	}

	bm.statsMutex.Lock()
	bm.stats.TotalRecordsProcessed += int64(len(metrics))
	bm.statsMutex.Unlock()

	// 检查是否需要刷新
	if len(bm.notificationReportBuffer) >= bm.config.FlushThreshold {
		go bm.FlushNotificationReportMetrics()
	}

	return nil
}

// generatePrecisePlatformKey 生成精确的平台指标聚合键
func (bm *FixedBufferManager) generatePrecisePlatformKey(metric *models.PlatformMetric) string {
	// 使用精确到秒的时间戳 + 系统ID + 组件名称
	// 这样可以避免不同组件的数据被错误聚合
	return fmt.Sprintf("%d_%s_%s",
		metric.Timestamp.Truncate(time.Second).Unix(),
		metric.SystemID,
		metric.ComponentName,
	)
}

// generatePreciseInterfaceKey 生成精确的接口指标聚合键
func (bm *FixedBufferManager) generatePreciseInterfaceKey(metric *models.InterfaceMetric) string {
	// 使用精确到秒的时间戳 + 系统ID + 接口名称
	return fmt.Sprintf("%d_%s_%s",
		metric.Timestamp.Truncate(time.Second).Unix(),
		metric.SystemID,
		metric.InterfaceName,
	)
}

// generatePreciseSubinterfaceKey 生成精确的子接口指标聚合键
func (bm *FixedBufferManager) generatePreciseSubinterfaceKey(metric *models.SubinterfaceMetric) string {
	// 使用精确到秒的时间戳 + 系统ID + 接口名称 + 子接口名称
	return fmt.Sprintf("%d_%s_%s_%s",
		metric.Timestamp.Truncate(time.Second).Unix(),
		metric.SystemID,
		metric.InterfaceName,
		metric.SubinterfaceName,
	)
}

// mergePlatformMetric 合并平台指标数据 - 修复数据丢失问题
func (bm *FixedBufferManager) mergePlatformMetric(existing, new *models.PlatformMetric) {
	// 总是使用最新的非空数据，避免数据丢失
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
	// 总是更新告警状态字段
	if new.MemAlarmStatus != nil {
		existing.MemAlarmStatus = new.MemAlarmStatus
	}
	if new.CPUAlarmStatus != nil {
		existing.CPUAlarmStatus = new.CPUAlarmStatus
	}
	// 更新时间戳为最新
	if new.Timestamp.After(existing.Timestamp) {
		existing.Timestamp = new.Timestamp
	}
}

// mergeInterfaceMetric 合并接口指标数据 - 修复数据丢失问题
func (bm *FixedBufferManager) mergeInterfaceMetric(existing, new *models.InterfaceMetric) {
	// 总是使用最新的非空数据，避免数据丢失
	if new.AdminStatusStr != nil {
		existing.AdminStatusStr = new.AdminStatusStr
	}
	if new.OperStatusStr != nil {
		existing.OperStatusStr = new.OperStatusStr
	}
	if new.PhyStatusStr != nil {
		existing.PhyStatusStr = new.PhyStatusStr
	}
	// 合并统计数据 - 使用最新值
	if new.InOctets != nil {
		existing.InOctets = new.InOctets
	}
	if new.OutOctets != nil {
		existing.OutOctets = new.OutOctets
	}
	// 更新时间戳为最新
	if new.Timestamp.After(existing.Timestamp) {
		existing.Timestamp = new.Timestamp
	}
}

// mergeSubinterfaceMetric 合并子接口指标数据 - 修复数据丢失问题
func (bm *FixedBufferManager) mergeSubinterfaceMetric(existing, new *models.SubinterfaceMetric) {
	// 总是使用最新的非空数据，避免数据丢失
	if new.AdminStatusStr != nil {
		existing.AdminStatusStr = new.AdminStatusStr
	}
	if new.OperStatusStr != nil {
		existing.OperStatusStr = new.OperStatusStr
	}
	// 合并统计数据 - 使用最新值
	if new.InOctets != nil {
		existing.InOctets = new.InOctets
	}
	if new.OutOctets != nil {
		existing.OutOctets = new.OutOctets
	}
	// 更新时间戳为最新
	if new.Timestamp.After(existing.Timestamp) {
		existing.Timestamp = new.Timestamp
	}
}

// startParallelWriters 启动并行写入器
func (bm *FixedBufferManager) startParallelWriters() {
	// 启动平台指标写入器
	for i := 0; i < bm.writerConfig.ParallelWriters; i++ {
		go bm.platformWriter()
	}
	
	// 启动接口指标写入器
	for i := 0; i < bm.writerConfig.ParallelWriters; i++ {
		go bm.interfaceWriter()
	}
	
	// 启动子接口指标写入器
	for i := 0; i < bm.writerConfig.ParallelWriters; i++ {
		go bm.subinterfaceWriter()
	}
	
	// 启动告警上报写入器
	for i := 0; i < bm.writerConfig.ParallelWriters; i++ {
		go bm.alarmReportWriter()
	}
	
	// 启动通知上报写入器
	for i := 0; i < bm.writerConfig.ParallelWriters; i++ {
		go bm.notificationReportWriter()
	}
}

// platformWriter 平台指标写入器
func (bm *FixedBufferManager) platformWriter() {
	for {
		select {
		case batch := <-bm.platformWriteChan:
			if err := bm.writeWithRetry(func() error {
				return bm.db.BatchInsertPlatformMetrics(batch)
			}); err != nil {
				bm.logger.Errorf("平台指标写入失败: %v", err)
				bm.statsMutex.Lock()
				bm.stats.TotalErrors++
				bm.statsMutex.Unlock()
			} else {
				bm.statsMutex.Lock()
				bm.stats.TotalRecordsWritten += int64(len(batch))
				bm.statsMutex.Unlock()
			}
		case <-bm.stopChan:
			return
		}
	}
}

// interfaceWriter 接口指标写入器
func (bm *FixedBufferManager) interfaceWriter() {
	for {
		select {
		case batch := <-bm.interfaceWriteChan:
			if err := bm.writeWithRetry(func() error {
				return bm.db.BatchInsertInterfaceMetrics(batch)
			}); err != nil {
				bm.logger.Errorf("接口指标写入失败: %v", err)
				bm.statsMutex.Lock()
				bm.stats.TotalErrors++
				bm.statsMutex.Unlock()
			} else {
				bm.statsMutex.Lock()
				bm.stats.TotalRecordsWritten += int64(len(batch))
				bm.statsMutex.Unlock()
			}
		case <-bm.stopChan:
			return
		}
	}
}

// subinterfaceWriter 子接口指标写入器
func (bm *FixedBufferManager) subinterfaceWriter() {
	for {
		select {
		case batch := <-bm.subinterfaceWriteChan:
			if err := bm.writeWithRetry(func() error {
				return bm.db.BatchInsertSubinterfaceMetrics(batch)
			}); err != nil {
				bm.logger.Errorf("子接口指标写入失败: %v", err)
				bm.statsMutex.Lock()
				bm.stats.TotalErrors++
				bm.statsMutex.Unlock()
			} else {
				bm.statsMutex.Lock()
				bm.stats.TotalRecordsWritten += int64(len(batch))
				bm.statsMutex.Unlock()
			}
		case <-bm.stopChan:
			return
		}
	}
}

// alarmReportWriter 告警上报写入器
func (bm *FixedBufferManager) alarmReportWriter() {
	for {
		select {
		case batch := <-bm.alarmReportWriteChan:
			if err := bm.writeWithRetry(func() error {
				return bm.db.BatchInsertAlarmReportMetrics(batch)
			}); err != nil {
				bm.logger.Errorf("告警上报写入失败: %v", err)
				bm.statsMutex.Lock()
				bm.stats.TotalErrors++
				bm.statsMutex.Unlock()
			} else {
				bm.statsMutex.Lock()
				bm.stats.TotalRecordsWritten += int64(len(batch))
				bm.statsMutex.Unlock()
			}
		case <-bm.stopChan:
			return
		}
	}
}

// notificationReportWriter 通知上报写入器
func (bm *FixedBufferManager) notificationReportWriter() {
	for {
		select {
		case batch := <-bm.notificationReportWriteChan:
			if err := bm.writeWithRetry(func() error {
				return bm.db.BatchInsertNotificationReportMetrics(batch)
			}); err != nil {
				bm.logger.Errorf("通知上报写入失败: %v", err)
				bm.statsMutex.Lock()
				bm.stats.TotalErrors++
				bm.statsMutex.Unlock()
			} else {
				bm.statsMutex.Lock()
				bm.stats.TotalRecordsWritten += int64(len(batch))
				bm.statsMutex.Unlock()
			}
		case <-bm.stopChan:
			return
		}
	}
}

// writeWithRetry 带重试的写入
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

// FlushAll 刷新所有缓冲区
func (bm *FixedBufferManager) FlushAll() error {
	start := time.Now()
	var errs []error
	
	// 并行刷新所有缓冲区
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
	
	// 收集错误
	for err := range errChan {
		errs = append(errs, err)
	}
	
	// 更新统计信息
	bm.statsMutex.Lock()
	bm.stats.LastFlushTime = time.Now()
	bm.stats.FlushDuration = time.Since(start)
	bm.statsMutex.Unlock()
	
	if len(errs) > 0 {
		return errs[0] // 返回第一个错误
	}
	
	return nil
}

// FlushPlatformMetrics 刷新平台指标缓冲区
func (bm *FixedBufferManager) FlushPlatformMetrics() error {
	bm.platformMutex.Lock()
	defer bm.platformMutex.Unlock()
	
	return bm.flushPlatformMetrics()
}

// FlushInterfaceMetrics 刷新接口指标缓冲区
func (bm *FixedBufferManager) FlushInterfaceMetrics() error {
	bm.interfaceMutex.Lock()
	defer bm.interfaceMutex.Unlock()
	
	return bm.flushInterfaceMetrics()
}

// FlushSubinterfaceMetrics 刷新子接口指标缓冲区
func (bm *FixedBufferManager) FlushSubinterfaceMetrics() error {
	bm.subinterfaceMutex.Lock()
	defer bm.subinterfaceMutex.Unlock()
	
	return bm.flushSubinterfaceMetrics()
}

// FlushAlarmReportMetrics 刷新告警上报缓冲区
func (bm *FixedBufferManager) FlushAlarmReportMetrics() error {
	bm.alarmReportMutex.Lock()
	defer bm.alarmReportMutex.Unlock()
	
	return bm.flushAlarmReportMetrics()
}

// FlushNotificationReportMetrics 刷新通知上报缓冲区
func (bm *FixedBufferManager) FlushNotificationReportMetrics() error {
	bm.notificationReportMutex.Lock()
	defer bm.notificationReportMutex.Unlock()
	
	return bm.flushNotificationReportMetrics()
}

// flushPlatformMetrics 内部平台指标刷新方法
func (bm *FixedBufferManager) flushPlatformMetrics() error {
	if len(bm.platformBuffer) == 0 {
		return nil
	}

	// 转换为切片
	metrics := make([]models.PlatformMetric, 0, len(bm.platformBuffer))
	for _, metric := range bm.platformBuffer {
		metrics = append(metrics, *metric)
	}

	// 清空缓冲区
	bm.platformBuffer = make(map[string]*models.PlatformMetric)

	// 分批发送到写入器
	batchSize := bm.writerConfig.MaxBatchSize
	for i := 0; i < len(metrics); i += batchSize {
		end := i + batchSize
		if end > len(metrics) {
			end = len(metrics)
		}
		
		batch := metrics[i:end]
		select {
		case bm.platformWriteChan <- batch:
			// 成功发送到写入器
		default:
			// 写入通道满，直接写入
			if err := bm.writeWithRetry(func() error {
				return bm.db.BatchInsertPlatformMetrics(batch)
			}); err != nil {
				return err
			}
		}
	}

	return nil
}

// flushInterfaceMetrics 内部接口指标刷新方法
func (bm *FixedBufferManager) flushInterfaceMetrics() error {
	if len(bm.interfaceBuffer) == 0 {
		return nil
	}

	// 转换为切片
	metrics := make([]models.InterfaceMetric, 0, len(bm.interfaceBuffer))
	for _, metric := range bm.interfaceBuffer {
		metrics = append(metrics, *metric)
	}

	// 清空缓冲区
	bm.interfaceBuffer = make(map[string]*models.InterfaceMetric)

	// 分批发送到写入器
	batchSize := bm.writerConfig.MaxBatchSize
	for i := 0; i < len(metrics); i += batchSize {
		end := i + batchSize
		if end > len(metrics) {
			end = len(metrics)
		}
		
		batch := metrics[i:end]
		select {
		case bm.interfaceWriteChan <- batch:
			// 成功发送到写入器
		default:
			// 写入通道满，直接写入
			if err := bm.writeWithRetry(func() error {
				return bm.db.BatchInsertInterfaceMetrics(batch)
			}); err != nil {
				return err
			}
		}
	}

	return nil
}

// flushSubinterfaceMetrics 内部子接口指标刷新方法
func (bm *FixedBufferManager) flushSubinterfaceMetrics() error {
	if len(bm.subinterfaceBuffer) == 0 {
		return nil
	}

	// 转换为切片
	metrics := make([]models.SubinterfaceMetric, 0, len(bm.subinterfaceBuffer))
	for _, metric := range bm.subinterfaceBuffer {
		metrics = append(metrics, *metric)
	}

	// 清空缓冲区
	bm.subinterfaceBuffer = make(map[string]*models.SubinterfaceMetric)

	// 分批发送到写入器
	batchSize := bm.writerConfig.MaxBatchSize
	for i := 0; i < len(metrics); i += batchSize {
		end := i + batchSize
		if end > len(metrics) {
			end = len(metrics)
		}
		
		batch := metrics[i:end]
		select {
		case bm.subinterfaceWriteChan <- batch:
			// 成功发送到写入器
		default:
			// 写入通道满，直接写入
			if err := bm.writeWithRetry(func() error {
				return bm.db.BatchInsertSubinterfaceMetrics(batch)
			}); err != nil {
				return err
			}
		}
	}

	return nil
}

// flushAlarmReportMetrics 内部告警上报刷新方法
func (bm *FixedBufferManager) flushAlarmReportMetrics() error {
	if len(bm.alarmReportBuffer) == 0 {
		return nil
	}

	// 转换为切片
	metrics := make([]models.AlarmReportMetric, 0, len(bm.alarmReportBuffer))
	for _, metric := range bm.alarmReportBuffer {
		metrics = append(metrics, *metric)
	}

	// 清空缓冲区
	bm.alarmReportBuffer = make(map[string]*models.AlarmReportMetric)

	// 分批发送到写入器
	batchSize := bm.writerConfig.MaxBatchSize
	for i := 0; i < len(metrics); i += batchSize {
		end := i + batchSize
		if end > len(metrics) {
			end = len(metrics)
		}
		
		batch := metrics[i:end]
		select {
		case bm.alarmReportWriteChan <- batch:
			// 成功发送到写入器
		default:
			// 写入通道满，直接写入
			if err := bm.writeWithRetry(func() error {
				return bm.db.BatchInsertAlarmReportMetrics(batch)
			}); err != nil {
				return err
			}
		}
	}

	return nil
}

// flushNotificationReportMetrics 内部通知上报刷新方法
func (bm *FixedBufferManager) flushNotificationReportMetrics() error {
	if len(bm.notificationReportBuffer) == 0 {
		return nil
	}

	// 转换为切片
	metrics := make([]models.NotificationReportMetric, 0, len(bm.notificationReportBuffer))
	for _, metric := range bm.notificationReportBuffer {
		metrics = append(metrics, *metric)
	}

	// 清空缓冲区
	bm.notificationReportBuffer = make(map[string]*models.NotificationReportMetric)

	// 分批发送到写入器
	batchSize := bm.writerConfig.MaxBatchSize
	for i := 0; i < len(metrics); i += batchSize {
		end := i + batchSize
		if end > len(metrics) {
			end = len(metrics)
		}
		
		batch := metrics[i:end]
		select {
		case bm.notificationReportWriteChan <- batch:
			// 成功发送到写入器
		default:
			// 写入通道满，直接写入
			if err := bm.writeWithRetry(func() error {
				return bm.db.BatchInsertNotificationReportMetrics(batch)
			}); err != nil {
				return err
			}
		}
	}

	return nil
}

// startFlushTimer 启动定时刷新
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

// Stop 停止缓冲区管理器
func (bm *FixedBufferManager) Stop() error {
	// 使用select防止重复关闭channel
	select {
	case <-bm.stopChan:
		// channel已经关闭，直接返回
		return nil
	default:
		close(bm.stopChan)
	}
	
	// 最后刷新一次
	return bm.FlushAll()
}

// GetStats 获取缓冲区统计信息
func (bm *FixedBufferManager) GetStats() FixedBufferStats {
	bm.statsMutex.RLock()
	defer bm.statsMutex.RUnlock()
	
	bm.platformMutex.RLock()
	bm.interfaceMutex.RLock()
	bm.subinterfaceMutex.RLock()
	bm.alarmReportMutex.RLock()
	bm.notificationReportMutex.RLock()
	
	stats := bm.stats
	stats.PlatformBufferSize = len(bm.platformBuffer)
	stats.InterfaceBufferSize = len(bm.interfaceBuffer)
	stats.SubinterfaceBufferSize = len(bm.subinterfaceBuffer)
	stats.AlarmReportBufferSize = len(bm.alarmReportBuffer)
	stats.NotificationReportBufferSize = len(bm.notificationReportBuffer)
	
	bm.notificationReportMutex.RUnlock()
	bm.alarmReportMutex.RUnlock()
	bm.subinterfaceMutex.RUnlock()
	bm.interfaceMutex.RUnlock()
	bm.platformMutex.RUnlock()
	
	return stats
}