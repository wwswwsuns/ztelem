package buffer

import (
	"context"
	"fmt"
	"sync"
	"time"

	"telemetry-collector/internal/config"
	"telemetry-collector/internal/database"
	"telemetry-collector/internal/models"

	"github.com/sirupsen/logrus"
)

// ExtendedBufferManager 扩展的缓冲区管理器
type ExtendedBufferManager struct {
	config    config.BufferConfig
	writerConfig config.DatabaseWriterConfig
	logger    *logrus.Logger
	db        *database.ExtendedDB
	
	// 分类缓冲区 - 使用map进行聚合
	platformBuffer      map[string]*models.PlatformMetric
	interfaceBuffer     map[string]*models.InterfaceMetric
	subinterfaceBuffer  map[string]*models.SubinterfaceMetric
	
	// 互斥锁
	platformMutex       sync.RWMutex
	interfaceMutex      sync.RWMutex
	subinterfaceMutex   sync.RWMutex
	
	// 写入器通道
	platformWriteChan      chan []models.PlatformMetric
	interfaceWriteChan     chan []models.InterfaceMetric
	subinterfaceWriteChan  chan []models.SubinterfaceMetric
	
	// 定时器和控制
	flushTimer *time.Timer
	stopChan   chan struct{}
	wg         sync.WaitGroup
	
	// 统计信息
	stats ExtendedBufferStats
	statsMutex sync.RWMutex
}

// ExtendedBufferStats 缓冲区统计信息
type ExtendedBufferStats struct {
	PlatformBufferSize     int
	InterfaceBufferSize    int
	SubinterfaceBufferSize int
	TotalProcessed         int64
	TotalErrors           int64
	LastFlushTime         time.Time
	FlushDuration         time.Duration
}

// NewExtendedBufferManager 创建扩展的缓冲区管理器
func NewExtendedBufferManager(config config.BufferConfig, writerConfig config.DatabaseWriterConfig, logger *logrus.Logger, db *database.ExtendedDB) *ExtendedBufferManager {
	bm := &ExtendedBufferManager{
		config:             config,
		writerConfig:       writerConfig,
		logger:             logger,
		db:                 db,
		platformBuffer:     make(map[string]*models.PlatformMetric),
		interfaceBuffer:    make(map[string]*models.InterfaceMetric),
		subinterfaceBuffer: make(map[string]*models.SubinterfaceMetric),
		stopChan:           make(chan struct{}),
		
		// 创建写入通道
		platformWriteChan:     make(chan []models.PlatformMetric, writerConfig.ParallelWriters),
		interfaceWriteChan:    make(chan []models.InterfaceMetric, writerConfig.ParallelWriters),
		subinterfaceWriteChan: make(chan []models.SubinterfaceMetric, writerConfig.ParallelWriters),
	}
	
	// 启动并行写入器
	bm.startParallelWriters()
	
	// 启动定时刷新
	bm.startFlushTimer()
	
	return bm
}

// AddPlatformMetrics 添加平台指标到缓冲区 - 优化版本
func (bm *ExtendedBufferManager) AddPlatformMetrics(metrics []models.PlatformMetric) error {
	bm.platformMutex.Lock()
	defer bm.platformMutex.Unlock()
	
	// 聚合数据：按 timestamp+system_id+component_name 进行聚合
	for _, metric := range metrics {
		key := bm.generatePlatformKey(&metric)
		
		if existing, exists := bm.platformBuffer[key]; exists {
			// 合并数据：将新数据的非空字段更新到现有记录
			bm.mergePlatformMetric(existing, &metric)
			bm.logger.Debugf("聚合平台指标数据: %s", key)
		} else {
			// 检查缓冲区大小限制
			if len(bm.platformBuffer) >= bm.config.PlatformBufferSize {
				bm.logger.Warnf("平台指标缓冲区已满 (%d)，强制刷新", len(bm.platformBuffer))
				if err := bm.flushPlatformMetrics(); err != nil {
					return fmt.Errorf("强制刷新平台指标缓冲区失败: %v", err)
				}
			}
			
			// 创建新记录的副本
			newMetric := metric
			bm.platformBuffer[key] = &newMetric
			bm.logger.Debugf("添加新平台指标数据: %s", key)
		}
	}
	
	// 检查是否需要立即刷新
	if len(bm.platformBuffer) >= bm.config.FlushThreshold {
		bm.logger.Debugf("平台指标缓冲区达到阈值 %d，立即刷新", len(bm.platformBuffer))
		return bm.flushPlatformMetrics()
	}
	
	bm.logger.Debugf("处理 %d 条平台指标，当前缓冲区大小: %d", len(metrics), len(bm.platformBuffer))
	return nil
}

// AddInterfaceMetrics 添加接口指标到缓冲区 - 优化版本
func (bm *ExtendedBufferManager) AddInterfaceMetrics(metrics []models.InterfaceMetric) error {
	bm.interfaceMutex.Lock()
	defer bm.interfaceMutex.Unlock()
	
	// 聚合数据：按 timestamp+system_id+interface_name 进行聚合
	for _, metric := range metrics {
		key := bm.generateInterfaceKey(&metric)
		
		if existing, exists := bm.interfaceBuffer[key]; exists {
			// 合并数据：将新数据的非空字段更新到现有记录
			bm.mergeInterfaceMetric(existing, &metric)
			bm.logger.Debugf("聚合接口指标数据: %s", key)
		} else {
			// 检查缓冲区大小限制
			if len(bm.interfaceBuffer) >= bm.config.InterfaceBufferSize {
				bm.logger.Warnf("接口指标缓冲区已满 (%d)，强制刷新", len(bm.interfaceBuffer))
				if err := bm.flushInterfaceMetrics(); err != nil {
					return fmt.Errorf("强制刷新接口指标缓冲区失败: %v", err)
				}
			}
			
			// 创建新记录的副本
			newMetric := metric
			bm.interfaceBuffer[key] = &newMetric
			bm.logger.Debugf("添加新接口指标数据: %s", key)
		}
	}
	
	// 检查是否需要立即刷新
	if len(bm.interfaceBuffer) >= bm.config.FlushThreshold {
		bm.logger.Debugf("接口指标缓冲区达到阈值 %d，立即刷新", len(bm.interfaceBuffer))
		return bm.flushInterfaceMetrics()
	}
	
	bm.logger.Debugf("处理 %d 条接口指标，当前缓冲区大小: %d", len(metrics), len(bm.interfaceBuffer))
	return nil
}

// AddSubinterfaceMetrics 添加子接口指标到缓冲区 - 优化版本
func (bm *ExtendedBufferManager) AddSubinterfaceMetrics(metrics []models.SubinterfaceMetric) error {
	bm.subinterfaceMutex.Lock()
	defer bm.subinterfaceMutex.Unlock()
	
	// 聚合数据：按 timestamp+system_id+interface_name+subinterface_index 进行聚合
	for _, metric := range metrics {
		key := bm.generateSubinterfaceKey(&metric)
		
		if existing, exists := bm.subinterfaceBuffer[key]; exists {
			// 合并数据：将新数据的非空字段更新到现有记录
			bm.mergeSubinterfaceMetric(existing, &metric)
			bm.logger.Debugf("聚合子接口指标数据: %s", key)
		} else {
			// 检查缓冲区大小限制
			if len(bm.subinterfaceBuffer) >= bm.config.SubinterfaceBufferSize {
				bm.logger.Warnf("子接口指标缓冲区已满 (%d)，强制刷新", len(bm.subinterfaceBuffer))
				if err := bm.flushSubinterfaceMetrics(); err != nil {
					return fmt.Errorf("强制刷新子接口指标缓冲区失败: %v", err)
				}
			}
			
			// 创建新记录的副本
			newMetric := metric
			bm.subinterfaceBuffer[key] = &newMetric
			bm.logger.Debugf("添加新子接口指标数据: %s", key)
		}
	}
	
	// 检查是否需要立即刷新
	if len(bm.subinterfaceBuffer) >= bm.config.FlushThreshold {
		bm.logger.Debugf("子接口指标缓冲区达到阈值 %d，立即刷新", len(bm.subinterfaceBuffer))
		return bm.flushSubinterfaceMetrics()
	}
	
	bm.logger.Debugf("处理 %d 条子接口指标，当前缓冲区大小: %d", len(metrics), len(bm.subinterfaceBuffer))
	return nil
}

// GetStats 获取缓冲区统计信息
func (bm *ExtendedBufferManager) GetStats() ExtendedBufferStats {
	bm.statsMutex.RLock()
	defer bm.statsMutex.RUnlock()
	
	bm.platformMutex.RLock()
	bm.interfaceMutex.RLock()
	bm.subinterfaceMutex.RLock()
	
	stats := bm.stats
	stats.PlatformBufferSize = len(bm.platformBuffer)
	stats.InterfaceBufferSize = len(bm.interfaceBuffer)
	stats.SubinterfaceBufferSize = len(bm.subinterfaceBuffer)
	
	bm.subinterfaceMutex.RUnlock()
	bm.interfaceMutex.RUnlock()
	bm.platformMutex.RUnlock()
	
	return stats
}

// FlushAll 刷新所有缓冲区 - 优化版本
func (bm *ExtendedBufferManager) FlushAll() error {
	start := time.Now()
	var errs []error
	
	// 并行刷新所有缓冲区
	var wg sync.WaitGroup
	errChan := make(chan error, 3)
	
	wg.Add(3)
	
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
func (bm *ExtendedBufferManager) FlushPlatformMetrics() error {
	bm.platformMutex.Lock()
	defer bm.platformMutex.Unlock()
	
	return bm.flushPlatformMetrics()
}

// FlushInterfaceMetrics 刷新接口指标缓冲区
func (bm *ExtendedBufferManager) FlushInterfaceMetrics() error {
	bm.interfaceMutex.Lock()
	defer bm.interfaceMutex.Unlock()
	
	return bm.flushInterfaceMetrics()
}

// FlushSubinterfaceMetrics 刷新子接口指标缓冲区
func (bm *ExtendedBufferManager) FlushSubinterfaceMetrics() error {
	bm.subinterfaceMutex.Lock()
	defer bm.subinterfaceMutex.Unlock()
	
	return bm.flushSubinterfaceMetrics()
}

// Stop 停止缓冲区管理器
func (bm *ExtendedBufferManager) Stop() {
	close(bm.stopChan)
	
	// 最后一次刷新所有缓冲区
	bm.FlushAll()
	
	// 关闭写入通道
	close(bm.platformWriteChan)
	close(bm.interfaceWriteChan)
	close(bm.subinterfaceWriteChan)
	
	// 等待所有写入器完成
	bm.wg.Wait()
}

// 启动并行写入器
func (bm *ExtendedBufferManager) startParallelWriters() {
	// 启动平台指标写入器
	for i := 0; i < bm.writerConfig.PlatformWriterCount; i++ {
		bm.wg.Add(1)
		go bm.platformWriter(i)
	}
	
	// 启动接口指标写入器
	for i := 0; i < bm.writerConfig.InterfaceWriterCount; i++ {
		bm.wg.Add(1)
		go bm.interfaceWriter(i)
	}
	
	// 启动子接口指标写入器
	for i := 0; i < bm.writerConfig.SubinterfaceWriterCount; i++ {
		bm.wg.Add(1)
		go bm.subinterfaceWriter(i)
	}
}

// 平台指标写入器
func (bm *ExtendedBufferManager) platformWriter(id int) {
	defer bm.wg.Done()
	
	for {
		select {
		case metrics, ok := <-bm.platformWriteChan:
			if !ok {
				return
			}
			
			// 带重试的写入
			if err := bm.writeWithRetry(func() error {
				return bm.db.BatchInsertPlatformMetrics(metrics)
			}); err != nil {
				bm.logger.WithError(err).Errorf("平台指标写入器 %d 写入失败", id)
				bm.statsMutex.Lock()
				bm.stats.TotalErrors++
				bm.statsMutex.Unlock()
			} else {
				bm.statsMutex.Lock()
				bm.stats.TotalProcessed += int64(len(metrics))
				bm.statsMutex.Unlock()
			}
			
		case <-bm.stopChan:
			return
		}
	}
}

// 接口指标写入器
func (bm *ExtendedBufferManager) interfaceWriter(id int) {
	defer bm.wg.Done()
	
	for {
		select {
		case metrics, ok := <-bm.interfaceWriteChan:
			if !ok {
				return
			}
			
			// 带重试的写入
			if err := bm.writeWithRetry(func() error {
				return bm.db.BatchInsertInterfaceMetrics(metrics)
			}); err != nil {
				bm.logger.WithError(err).Errorf("接口指标写入器 %d 写入失败", id)
				bm.statsMutex.Lock()
				bm.stats.TotalErrors++
				bm.statsMutex.Unlock()
			} else {
				bm.statsMutex.Lock()
				bm.stats.TotalProcessed += int64(len(metrics))
				bm.statsMutex.Unlock()
			}
			
		case <-bm.stopChan:
			return
		}
	}
}

// 子接口指标写入器
func (bm *ExtendedBufferManager) subinterfaceWriter(id int) {
	defer bm.wg.Done()
	
	for {
		select {
		case metrics, ok := <-bm.subinterfaceWriteChan:
			if !ok {
				return
			}
			
			// 带重试的写入
			if err := bm.writeWithRetry(func() error {
				return bm.db.BatchInsertSubinterfaceMetrics(metrics)
			}); err != nil {
				bm.logger.WithError(err).Errorf("子接口指标写入器 %d 写入失败", id)
				bm.statsMutex.Lock()
				bm.stats.TotalErrors++
				bm.statsMutex.Unlock()
			} else {
				bm.statsMutex.Lock()
				bm.stats.TotalProcessed += int64(len(metrics))
				bm.statsMutex.Unlock()
			}
			
		case <-bm.stopChan:
			return
		}
	}
}

// 带重试的写入
func (bm *ExtendedBufferManager) writeWithRetry(writeFunc func() error) error {
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
			bm.logger.WithError(err).Debugf("写入失败，尝试 %d/%d", attempt+1, bm.writerConfig.RetryAttempts)
			
		case <-ctx.Done():
			cancel()
			lastErr = fmt.Errorf("写入超时")
			bm.logger.Debugf("写入超时，尝试 %d/%d", attempt+1, bm.writerConfig.RetryAttempts)
		}
	}
	
	return fmt.Errorf("写入失败，已重试 %d 次: %v", bm.writerConfig.RetryAttempts, lastErr)
}

// 内部刷新函数（需要在持有锁的情况下调用）
func (bm *ExtendedBufferManager) flushPlatformMetrics() error {
	if len(bm.platformBuffer) == 0 {
		return nil
	}
	
	// 将map转换为slice
	metrics := make([]models.PlatformMetric, 0, len(bm.platformBuffer))
	for _, metric := range bm.platformBuffer {
		metrics = append(metrics, *metric)
	}
	
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
			// 写入通道满了，直接写入
			if err := bm.writeWithRetry(func() error {
				return bm.db.BatchInsertPlatformMetrics(batch)
			}); err != nil {
				bm.logger.WithError(err).Error("直接写入平台指标失败")
				return err
			}
		}
	}
	
	bm.logger.Infof("成功刷新 %d 条平台指标数据", len(metrics))
	bm.platformBuffer = make(map[string]*models.PlatformMetric) // 清空缓冲区
	return nil
}

func (bm *ExtendedBufferManager) flushInterfaceMetrics() error {
	if len(bm.interfaceBuffer) == 0 {
		return nil
	}
	
	// 将map转换为slice
	metrics := make([]models.InterfaceMetric, 0, len(bm.interfaceBuffer))
	for _, metric := range bm.interfaceBuffer {
		metrics = append(metrics, *metric)
	}
	
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
			// 写入通道满了，直接写入
			if err := bm.writeWithRetry(func() error {
				return bm.db.BatchInsertInterfaceMetrics(batch)
			}); err != nil {
				bm.logger.WithError(err).Error("直接写入接口指标失败")
				return err
			}
		}
	}
	
	bm.logger.Infof("成功刷新 %d 条接口指标数据", len(metrics))
	bm.interfaceBuffer = make(map[string]*models.InterfaceMetric) // 清空缓冲区
	return nil
}

func (bm *ExtendedBufferManager) flushSubinterfaceMetrics() error {
	if len(bm.subinterfaceBuffer) == 0 {
		return nil
	}
	
	// 将map转换为slice
	metrics := make([]models.SubinterfaceMetric, 0, len(bm.subinterfaceBuffer))
	for _, metric := range bm.subinterfaceBuffer {
		metrics = append(metrics, *metric)
	}
	
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
			// 写入通道满了，直接写入
			if err := bm.writeWithRetry(func() error {
				return bm.db.BatchInsertSubinterfaceMetrics(batch)
			}); err != nil {
				bm.logger.WithError(err).Error("直接写入子接口指标失败")
				return err
			}
		}
	}
	
	bm.logger.Infof("成功刷新 %d 条子接口指标数据", len(metrics))
	bm.subinterfaceBuffer = make(map[string]*models.SubinterfaceMetric) // 清空缓冲区
	return nil
}

// startFlushTimer 启动定时刷新
func (bm *ExtendedBufferManager) startFlushTimer() {
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

// 生成聚合键的辅助函数（复用原有逻辑）
func (bm *ExtendedBufferManager) generatePlatformKey(metric *models.PlatformMetric) string {
	return fmt.Sprintf("%d_%s_%s", 
		metric.Timestamp.Unix(), 
		metric.SystemID, 
		metric.ComponentName)
}

func (bm *ExtendedBufferManager) generateInterfaceKey(metric *models.InterfaceMetric) string {
	return fmt.Sprintf("%d_%s_%s", 
		metric.Timestamp.Unix(), 
		metric.SystemID, 
		metric.InterfaceName)
}

func (bm *ExtendedBufferManager) generateSubinterfaceKey(metric *models.SubinterfaceMetric) string {
	return fmt.Sprintf("%d_%s_%s_%s", 
		metric.Timestamp.Unix(), 
		metric.SystemID, 
		metric.InterfaceName,
		metric.SubinterfaceName)
}

// 数据合并函数（复用原有逻辑，这里省略具体实现）
func (bm *ExtendedBufferManager) mergePlatformMetric(existing, new *models.PlatformMetric) {
	// 实现与原有 buffer_manager.go 中相同的合并逻辑
	// 这里省略具体实现以节省空间
}

func (bm *ExtendedBufferManager) mergeInterfaceMetric(existing, new *models.InterfaceMetric) {
	// 实现与原有 buffer_manager.go 中相同的合并逻辑
	// 这里省略具体实现以节省空间
}

func (bm *ExtendedBufferManager) mergeSubinterfaceMetric(existing, new *models.SubinterfaceMetric) {
	// 实现与原有 buffer_manager.go 中相同的合并逻辑
	// 这里省略具体实现以节省空间
}