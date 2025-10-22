# 遥测数据采集系统优化指南

## 问题分析与解决方案

### 1. 时间戳问题 ✅
**问题**：担心使用服务器时间而非GPB数据时间戳
**分析**：代码实际已正确使用GPB数据中的`MsgTimestamp`
**解决方案**：
- 添加了时间戳验证机制
- 检测异常时间戳（过于久远或来自未来）
- 记录时间戳验证错误统计

### 2. PostgreSQL参数限制问题 ❌ → ✅
**问题**：`pq: got 133110 parameters but PostgreSQL only supports 65535 parameters`
**根本原因**：
- 平台指标90个字段 × 1000条记录 = 90,000参数 > 65,535限制
- 1秒采集周期产生大量数据，批量插入超限

**解决方案**：
```go
// 动态计算最优批次大小
func calculateOptimalBatchSize(fieldsPerRecord int) int {
    const maxParams = 65535
    maxBatchSize := maxParams / fieldsPerRecord
    return maxBatchSize - 10  // 保留余量
}
```

**配置优化**：
- `max_batch_size: 500` （平台指标：90×500=45,000 < 65,535）
- `flush_threshold: 5000` （降低阈值，避免积累过多）
- `flush_interval: "10s"` （缩短间隔，提高实时性）

### 3. 数据丢失问题 ❌ → ✅
**问题**：多设备并发时部分数据丢失，如`oper_status`字段缺失
**根本原因**：
- 聚合键不够精确，导致不同组件数据被错误合并
- 并发写入时存在竞态条件
- 缺乏冲突检测机制

**解决方案**：
1. **优化聚合键**：
```go
// 原来：timestamp+system_id+component_name
// 优化：精确到秒的时间戳+system_id+component_name
func generateOptimizedPlatformKey(metric *models.PlatformMetric) string {
    return fmt.Sprintf("%d_%s_%s",
        metric.Timestamp.Truncate(time.Second).Unix(),
        metric.SystemID,
        metric.ComponentName,
    )
}
```

2. **添加冲突检测**：
```go
func detectPlatformConflict(existing, new *models.PlatformMetric) bool {
    // 检查关键字段是否冲突
    if existing.ComponentName != new.ComponentName {
        return true
    }
    if existing.OperStatus != nil && new.OperStatus != nil && 
       *existing.OperStatus != *new.OperStatus {
        return true
    }
    return false
}
```

3. **并发安全优化**：
- 使用读写锁保护缓冲区
- 添加并行写入器
- 实现重试机制

## 使用方法

### 1. 替换优化组件

```bash
# 备份原文件
cp internal/database/database.go internal/database/database_backup.go
cp internal/buffer/buffer_manager.go internal/buffer/buffer_manager_backup.go

# 使用优化版本
mv internal/database/database_optimized.go internal/database/database.go
mv internal/buffer/buffer_manager_optimized.go internal/buffer/buffer_manager.go
```

### 2. 更新配置文件

```bash
# 备份原配置
cp config.yaml config_backup.yaml

# 使用优化配置
cp config_optimized.yaml config.yaml
```

### 3. 修改主程序

在主程序中使用优化的组件：

```go
// 使用优化的数据库连接
db, err := database.NewOptimizedConnection(config.Database)
if err != nil {
    log.Fatal("数据库连接失败:", err)
}

// 使用优化的缓冲区管理器
bufferManager := buffer.NewOptimizedBufferManager(
    db, 
    config.Buffer, 
    config.DatabaseWriter, 
    logger,
)
```

### 4. 监控关键指标

```go
// 定期检查统计信息
stats := bufferManager.GetStats()
log.Printf("缓冲区统计: 处理=%d, 写入=%d, 错误=%d, 冲突=%d, 时间戳错误=%d",
    stats.TotalRecordsProcessed,
    stats.TotalRecordsWritten,
    stats.TotalErrors,
    stats.AggregationConflicts,
    stats.TimestampValidationErrors,
)
```

## 性能优化建议

### 1. 数据库层面
```sql
-- 创建合适的索引
CREATE INDEX CONCURRENTLY idx_platform_metrics_time_system 
ON platform_metrics (time, system_id);

CREATE INDEX CONCURRENTLY idx_interface_metrics_time_system 
ON interface_metrics (time, system_id);

-- 启用并行查询
SET max_parallel_workers_per_gather = 4;
```

### 2. 系统层面
```bash
# 调整PostgreSQL配置
echo "max_connections = 200" >> /etc/postgresql/postgresql.conf
echo "shared_buffers = 256MB" >> /etc/postgresql/postgresql.conf
echo "work_mem = 4MB" >> /etc/postgresql/postgresql.conf
echo "maintenance_work_mem = 64MB" >> /etc/postgresql/postgresql.conf

# 重启PostgreSQL
systemctl restart postgresql
```

### 3. 应用层面
- 使用连接池管理数据库连接
- 实现熔断器模式防止雪崩
- 添加监控和告警机制

## 测试验证

### 1. 参数限制测试
```bash
# 测试1秒采集周期是否还会报错
# 观察日志中是否还有"got XXX parameters"错误
tail -f logs/telemetry.log | grep "parameters"
```

### 2. 数据完整性测试
```sql
-- 检查oper_status字段完整性
SELECT 
    system_id,
    component_name,
    COUNT(*) as total_records,
    COUNT(oper_status) as records_with_status,
    (COUNT(oper_status) * 100.0 / COUNT(*)) as completeness_rate
FROM platform_metrics 
WHERE time >= NOW() - INTERVAL '1 hour'
GROUP BY system_id, component_name
ORDER BY completeness_rate;
```

### 3. 性能测试
```bash
# 监控系统资源使用
top -p $(pgrep telemetry)
iostat -x 1
```

## 预期效果

1. **解决参数限制**：批次大小动态调整，不再超过65535参数限制
2. **减少数据丢失**：优化聚合逻辑，添加冲突检测，提高数据完整性
3. **提升性能**：并行写入，连接池优化，减少延迟
4. **增强监控**：详细统计信息，便于问题诊断

## 故障排除

### 常见问题

1. **仍然出现参数限制错误**
   - 检查`max_batch_size`配置是否正确
   - 确认字段数量计算是否准确

2. **数据仍有丢失**
   - 检查聚合键生成逻辑
   - 查看冲突检测统计信息

3. **性能下降**
   - 调整并行写入器数量
   - 优化数据库连接池配置

### 日志分析
```bash
# 查看优化效果
grep "成功插入.*批次" logs/telemetry.log
grep "聚合.*冲突" logs/telemetry.log
grep "时间戳验证失败" logs/telemetry.log