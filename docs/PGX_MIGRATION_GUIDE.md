# pgx + COPY FROM STDIN 高性能优化迁移指南

## 🎯 优化效果预期

### 性能提升
- **写入速度**: 5-10倍提升 (COPY vs INSERT)
- **CPU使用率**: 降低60-80% (无占位符解析)
- **内存使用**: 降低40-60% (流式处理)
- **并发能力**: 提升3-5倍 (真正的并行写入)

### 监控能力
- **实时指标**: Prometheus标准格式
- **性能分析**: 延迟、吞吐量、错误率
- **容量规划**: 连接池、队列深度监控

## 📋 迁移步骤

### 1. 安装依赖
```bash
cd /home/telemetry

# 添加pgx和prometheus依赖
go get github.com/jackc/pgx/v5@v5.4.3
go get github.com/prometheus/client_golang@v1.17.0

# 更新依赖
go mod tidy
```

### 2. 备份现有文件
```bash
# 备份数据库层
cp internal/database/database.go internal/database/database_backup.go

# 备份缓冲区管理
cp internal/buffer/buffer_manager.go internal/buffer/buffer_manager_backup.go

# 备份主程序（如果需要修改）
cp main.go main_backup.go
```

### 3. 集成新的数据库层

#### 3.1 修改主程序 (main.go)
```go
// 在import部分添加
import (
    "github.com/wwswwsuns/ztelem/internal/monitoring"
    // ... 其他导入
)

// 在main函数中添加Prometheus服务器
func main() {
    // ... 现有代码 ...
    
    // 启动Prometheus指标服务器
    prometheusServer := monitoring.NewPrometheusServer(12112, logger)
    if err := prometheusServer.Start(); err != nil {
        logger.WithError(err).Fatal("启动Prometheus服务器失败")
    }
    defer prometheusServer.Stop()
    
    // 创建pgx数据库连接（替换现有数据库连接）
    pgxDB, err := database.NewPgxDatabase(cfg.Database, logger)
    if err != nil {
        logger.WithError(err).Fatal("创建pgx数据库连接失败")
    }
    defer pgxDB.Close()
    
    // 测试连接
    if err := pgxDB.TestConnection(); err != nil {
        logger.WithError(err).Fatal("数据库连接测试失败")
    }
    
    // 启动连接池指标更新
    go func() {
        ticker := time.NewTicker(30 * time.Second)
        defer ticker.Stop()
        for {
            select {
            case <-ticker.C:
                pgxDB.UpdatePoolMetrics()
            case <-ctx.Done():
                return
            }
        }
    }()
    
    // ... 其余代码保持不变，但将数据库实例传递给缓冲区管理器 ...
}
```

#### 3.2 修改缓冲区管理器
```go
// 在buffer_manager.go中，将数据库接口改为支持pgx
// 或者创建一个适配器接口

type DatabaseWriter interface {
    BatchInsertPlatformMetrics([]models.PlatformMetric) error
    BatchInsertInterfaceMetrics([]models.InterfaceMetric) error
    BatchInsertSubinterfaceMetrics([]models.SubinterfaceMetric) error
}

// 确保PgxDatabase实现了这个接口（已经实现）
```

### 4. 配置优化

#### 4.1 更新配置文件
```yaml
# 在production-config-optimized.yaml中添加
monitoring:
  prometheus_port: 12112
  metrics_update_interval: "30s"

database_writer:
  # pgx优化配置
  max_batch_size: 2000        # pgx可以处理更大的批次
  batch_timeout: "5s"
  max_retries: 3
  retry_delay: "1s"
  
  # 并发写入配置
  parallel_writers: 10        # 增加并行写入数量
  worker_pool_size: 50        # 工作池大小
```

#### 4.2 数据库连接池优化
```yaml
database:
  max_open_conns: 200         # 增加连接数
  max_idle_conns: 50
  conn_max_lifetime: "1h"
  conn_max_idle_time: "30m"
```

### 5. 测试验证

#### 5.1 编译测试
```bash
# 编译新版本
go build -o bin/telemetry-pgx main.go

# 检查依赖
go mod verify
```

#### 5.2 功能测试
```bash
# 启动新版本（测试模式）
./bin/telemetry-pgx -config production-config-optimized.yaml

# 检查Prometheus指标
curl http://localhost:12112/metrics

# 检查健康状态
curl http://localhost:12112/health
```

#### 5.3 性能对比测试
```bash
# 监控关键指标
# 1. 写入延迟
curl -s http://localhost:12112/metrics | grep telemetry_db_write_duration_seconds

# 2. 吞吐量
curl -s http://localhost:12112/metrics | grep telemetry_db_records_written_total

# 3. 错误率
curl -s http://localhost:12112/metrics | grep telemetry_db_write_errors_total

# 4. 连接池状态
curl -s http://localhost:12112/metrics | grep telemetry_db_pool_connections
```

## 📊 监控仪表板

### Grafana查询示例

#### 1. 写入吞吐量
```promql
# 每秒写入记录数
rate(telemetry_db_records_written_total[5m])

# 按表分组的吞吐量
sum(rate(telemetry_db_records_written_total[5m])) by (table)
```

#### 2. 写入延迟
```promql
# 平均写入延迟
rate(telemetry_db_write_duration_seconds_sum[5m]) / rate(telemetry_db_write_duration_seconds_count[5m])

# 95%分位延迟
histogram_quantile(0.95, rate(telemetry_db_write_duration_seconds_bucket[5m]))
```

#### 3. 错误率
```promql
# 写入错误率
rate(telemetry_db_write_errors_total[5m]) / rate(telemetry_db_records_written_total[5m]) * 100
```

#### 4. 连接池监控
```promql
# 连接池使用率
telemetry_db_pool_connections{state="acquired"} / telemetry_db_pool_connections{state="total"} * 100

# 连接等待时间
rate(telemetry_db_pool_wait_duration_seconds_sum[5m]) / rate(telemetry_db_pool_wait_duration_seconds_count[5m])
```

## 🚨 告警规则

### Prometheus告警配置
```yaml
groups:
- name: telemetry_database
  rules:
  # 写入延迟告警
  - alert: HighDatabaseWriteLatency
    expr: rate(telemetry_db_write_duration_seconds_sum[5m]) / rate(telemetry_db_write_duration_seconds_count[5m]) > 1
    for: 2m
    labels:
      severity: warning
    annotations:
      summary: "数据库写入延迟过高"
      description: "数据库写入平均延迟超过1秒，当前值: {{ $value }}秒"

  # 写入错误率告警
  - alert: HighDatabaseWriteErrorRate
    expr: rate(telemetry_db_write_errors_total[5m]) / rate(telemetry_db_records_written_total[5m]) > 0.01
    for: 1m
    labels:
      severity: critical
    annotations:
      summary: "数据库写入错误率过高"
      description: "数据库写入错误率超过1%，当前值: {{ $value | humanizePercentage }}"

  # 连接池耗尽告警
  - alert: DatabaseConnectionPoolExhausted
    expr: telemetry_db_pool_connections{state="acquired"} / telemetry_db_pool_connections{state="total"} > 0.9
    for: 30s
    labels:
      severity: warning
    annotations:
      summary: "数据库连接池使用率过高"
      description: "数据库连接池使用率超过90%，当前值: {{ $value | humanizePercentage }}"
```

## 🔄 回滚方案

如果遇到问题需要回滚：

```bash
# 1. 停止新版本
pkill telemetry-pgx

# 2. 恢复原版本
cp internal/database/database_backup.go internal/database/database.go
cp internal/buffer/buffer_manager_backup.go internal/buffer/buffer_manager.go
cp main_backup.go main.go

# 3. 重新编译
go build -o bin/telemetry main.go

# 4. 启动原版本
./bin/telemetry -config production-config-optimized.yaml
```

## 📈 预期性能提升

### 1秒采集周期场景
- **原版本**: 133,110参数错误，无法正常工作
- **优化版本**: 无参数限制，COPY FROM STDIN流式处理

### 多设备并发场景
- **原版本**: 单线程写入，数据丢失
- **优化版本**: 真正并行写入，数据完整性保证

### 监控可见性
- **原版本**: 无性能指标，问题难以定位
- **优化版本**: 全面的Prometheus指标，实时监控

## 🎯 成功标准

迁移成功的标志：
1. ✅ 1秒采集周期无参数限制错误
2. ✅ oper_status等字段数据完整性提升
3. ✅ 写入吞吐量提升5倍以上
4. ✅ Prometheus指标正常暴露
5. ✅ 系统稳定运行24小时以上

按照这个指南，你可以安全地将系统升级到高性能的pgx + COPY FROM STDIN架构！