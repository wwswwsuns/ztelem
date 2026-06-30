# ZTE Telemetry Data Collector

高性能的ZTE设备遥测数据采集系统，支持gRPC数据采集、PostgreSQL/TimescaleDB批量写入和Prometheus监控。

## 🚀 主要特性

### 核心功能
- **高性能数据采集**: 支持gRPC流式数据接收，配置keepalive机制确保连接稳定性
- **批量数据写入**: 使用pgx + COPY FROM STDIN，解决PostgreSQL 65535参数限制
- **实时监控**: 完整的Prometheus指标体系
- **多设备支持**: 同时处理多台ZTE设备的遥测数据（生产环境已验证300+设备）
- **全面数据类型支持**: 平台指标、接口指标、子接口指标、**告警数据、通知消息**
- **智能告警解析**: 支持ZTE设备告警上报和通知消息的实时解析与存储
- **光功率数据优化**: 准确区分0.0 dBm有效值和无光信号状态(-60 dBm)

### 性能优化
- **分片锁**: 16分片 × 5种缓冲区类型，消除高并发写入锁竞争
- **零分配聚合键**: sync.Pool + byte buffer 拼接，避免 Sprintf GC 压力
- **protobuf 对象池**: sync.Pool 复用 Telemetry/ComponentInfo/InterfaceInfo
- **PlatformMetric 子结构体**: 拆分为 CommonState/CPU/Mem/Temp/Fan/Power/Optical，按需分配
- **连接池优化**: 200最大连接，50最小连接，参数可配置
- **并行写入**: 10个并行写入器，支持多表并行
- **智能缓冲区管理**: 分片锁 + SwapAll flush，可配置刷新策略
- **错误重试**: 自动重试机制，提高数据可靠性
- **gRPC优化**: Keepalive配置、连接监控、自动重连机制
- **TimescaleDB支持**: 超表、压缩策略、数据保留策略

### 监控体系
- **Prometheus指标**: 7个精简业务指标（已清理高基数和死代码指标）
- **实时监控**: 缓冲区状态、数据库连接池、系统资源
- **告警支持**: 可配置的阈值告警
- **Web界面**: 指标说明和健康检查
- **连接监控**: gRPC连接状态实时监控

## 📋 系统要求

### 运行环境
- **操作系统**: Linux (推荐 Ubuntu 18.04+)
- **Go版本**: 1.19+
- **数据库**: PostgreSQL 12+ (推荐使用TimescaleDB)
- **内存**: 最小4GB，推荐8GB+
- **CPU**: 最小4核，推荐8核+

### 网络要求
- **gRPC端口**: 50051 (设备连接)
- **Prometheus端口**: 12112 (监控指标)
- **健康检查端口**: 8080 (可选)

## 🛠️ 安装部署

### 1. 克隆代码
```bash
git clone https://github.com/wwswwsuns/ztelem.git
cd ztelem
```

### 2. 安装依赖
```bash
go mod download
```

### 3. 数据库准备

#### PostgreSQL安装
```bash
# Ubuntu/Debian
sudo apt update
sudo apt install postgresql postgresql-contrib

# 启动服务
sudo systemctl start postgresql
sudo systemctl enable postgresql
```

#### 创建数据库和用户
```sql
-- 连接到PostgreSQL
sudo -u postgres psql

-- 创建数据库和用户
CREATE DATABASE telemetrydb;
CREATE USER telemetry_app WITH PASSWORD 'your_password';
GRANT ALL PRIVILEGES ON DATABASE telemetrydb TO telemetry_app;

-- 创建schema
\c telemetrydb
CREATE SCHEMA telemetry;
GRANT ALL ON SCHEMA telemetry TO telemetry_app;
```

#### 创建数据表
```sql
-- 切换到telemetry schema
SET search_path TO telemetry;

-- 平台指标表（包含光功率等完整字段）
CREATE TABLE platform_metrics (
    id BIGSERIAL PRIMARY KEY,
    timestamp TIMESTAMPTZ NOT NULL,
    system_id TEXT NOT NULL,
    component_name TEXT,
    component_type TEXT,
    oper_status TEXT,
    admin_status TEXT,
    alarm_status BOOLEAN,
    
    -- CPU相关字段
    cpu_instant DOUBLE PRECISION,
    cpu_avg DOUBLE PRECISION,
    cpu_min DOUBLE PRECISION,
    cpu_max DOUBLE PRECISION,
    cpu_interval BIGINT,
    cpu_min_time TIMESTAMPTZ,
    cpu_max_time TIMESTAMPTZ,
    cpu_alarm_status TEXT,
    
    -- 内存相关字段
    mem_free BIGINT,
    mem_usage DOUBLE PRECISION,
    
    -- 温度相关字段
    temp_instant DOUBLE PRECISION,
    temp_avg DOUBLE PRECISION,
    temp_min DOUBLE PRECISION,
    temp_max DOUBLE PRECISION,
    temp_interval BIGINT,
    temp_alarm_threshold DOUBLE PRECISION,
    temp_alarm_severity TEXT,
    temp_minor_threshold DOUBLE PRECISION,
    temp_major_threshold DOUBLE PRECISION,
    temp_fatal_threshold DOUBLE PRECISION,
    temp_instant_string TEXT,
    
    -- 电源相关字段
    power_enable BOOLEAN,
    power_capacity DOUBLE PRECISION,
    power_input_current DOUBLE PRECISION,
    power_input_voltage DOUBLE PRECISION,
    power_output_current DOUBLE PRECISION,
    power_output_voltage DOUBLE PRECISION,
    power_output_power DOUBLE PRECISION,
    power_work_state TEXT,
    power_input_power TEXT,
    power_input2_current DOUBLE PRECISION,
    power_input2_voltage DOUBLE PRECISION,
    power_output2_current DOUBLE PRECISION,
    power_output2_voltage DOUBLE PRECISION,
    
    -- 存储相关字段
    storage_availability DOUBLE PRECISION,
    
    -- 光模块数据（优化后的光功率处理）
    optical_in_power DOUBLE PRECISION,           -- 输入光功率：0.0=有效零值，-60.0=无光信号
    optical_out_power DOUBLE PRECISION,          -- 输出光功率：0.0=有效零值，-60.0=无光信号
    optical_bias_current DOUBLE PRECISION,
    optical_temperature DOUBLE PRECISION,
    optical_voltage_vol33 DOUBLE PRECISION,
    optical_voltage_vol5 DOUBLE PRECISION,
    optical_alarm_los_status TEXT,
    optical_alarm_los_info_event_id INTEGER,
    optical_alarm_los_info_event_interval INTEGER,
    optical_alarm_los_info_in_power DOUBLE PRECISION,   -- 告警输入光功率：0.0=有效零值，-60.0=无光信号
    optical_alarm_los_info_out_power DOUBLE PRECISION,  -- 告警输出光功率：0.0=有效零值，-60.0=无光信号
    optical_online_status TEXT,
    optical_rx_threshold_high_alarm DOUBLE PRECISION,
    optical_rx_threshold_pre_high_alarm DOUBLE PRECISION,
    optical_rx_threshold_low_alarm DOUBLE PRECISION,
    optical_rx_threshold_pre_low_alarm DOUBLE PRECISION,
    optical_tx_threshold_high_alarm DOUBLE PRECISION,
    optical_tx_threshold_pre_high_alarm DOUBLE PRECISION,
    optical_tx_threshold_low_alarm DOUBLE PRECISION,
    optical_tx_threshold_pre_low_alarm DOUBLE PRECISION,
    
    -- 板卡相关字段
    linecard_power_admin_state TEXT
);

-- 光功率数据说明：
-- • 0.0 dBm = 有效的零光功率值
-- • -60.0 dBm = 无光信号或无效数据
-- • 其他值 = 实际测量的光功率值

-- 接口指标表  
CREATE TABLE interface_metrics (
    timestamp TIMESTAMPTZ NOT NULL,
    system_id TEXT NOT NULL,
    interface_name TEXT NOT NULL,
    admin_status TEXT,
    oper_status TEXT,
    in_octets BIGINT,
    out_octets BIGINT,
    in_pkts BIGINT,
    out_pkts BIGINT,
    in_errors BIGINT,
    out_errors BIGINT,
    in_discards BIGINT,
    out_discards BIGINT,
    speed BIGINT,
    mtu INTEGER,
    duplex TEXT,
    description TEXT
);

-- 子接口指标表
CREATE TABLE subinterface_metrics (
    timestamp TIMESTAMPTZ NOT NULL,
    system_id TEXT NOT NULL,
    interface_name TEXT NOT NULL,
    subinterface_name TEXT NOT NULL,
    admin_status TEXT,
    oper_status TEXT,
    in_octets BIGINT,
    out_octets BIGINT,
    in_pkts BIGINT,
    out_pkts BIGINT,
    vlan_id INTEGER,
    description TEXT
);

-- 告警上报表（优化后的时间字段处理）
CREATE TABLE alarm_report (
    id BIGSERIAL PRIMARY KEY,
    timestamp TIMESTAMPTZ NOT NULL,
    system_id TEXT NOT NULL,
    flow_id BIGINT NOT NULL,
    alarm_type TEXT,
    severity TEXT,
    description TEXT,
    resource TEXT,
    probable_cause TEXT,
    occurrence_time TIMESTAMPTZ,      -- 告警发生时间（从uint32 Unix时间戳自动转换）
    update_time TIMESTAMPTZ,          -- 告警更新时间（从uint32 Unix时间戳自动转换）
    disappeared_time TIMESTAMPTZ,     -- 告警消失时间（从uint32 Unix时间戳自动转换）
    sequence_number BIGINT,
    additional_info JSONB
);

-- 通知消息表（优化后的时间字段处理）
CREATE TABLE notification_report (
    id BIGSERIAL PRIMARY KEY,
    timestamp TIMESTAMPTZ NOT NULL,
    system_id TEXT NOT NULL,
    flow_id BIGINT NOT NULL,
    classification TEXT,
    severity TEXT,
    description TEXT,
    resource TEXT,
    occur_time TIMESTAMPTZ,           -- 通知发生时间（从uint32 Unix时间戳自动转换）
    sequence_number BIGINT,
    additional_info JSONB
);

-- 时间字段说明：
-- • 所有时间字段自动从Proto的uint32 Unix时间戳转换为TIMESTAMPTZ格式
-- • 存储格式：'2025-09-28 14:30:00+00' (UTC时区)
-- • 支持时区查询：AT TIME ZONE 'Asia/Shanghai' 转换为本地时间
-- • 0或4294967295(uint32最大值)表示无效时间，存储为NULL

-- 创建索引优化查询性能
CREATE INDEX idx_platform_timestamp ON platform_metrics(timestamp);
CREATE INDEX idx_platform_system_id ON platform_metrics(system_id);
CREATE INDEX idx_interface_timestamp ON interface_metrics(timestamp);
CREATE INDEX idx_interface_system_id ON interface_metrics(system_id);
CREATE INDEX idx_subinterface_timestamp ON subinterface_metrics(timestamp);
CREATE INDEX idx_subinterface_system_id ON subinterface_metrics(system_id);
CREATE INDEX idx_alarm_timestamp ON alarm_report(timestamp);
CREATE INDEX idx_alarm_system_id ON alarm_report(system_id);
CREATE INDEX idx_alarm_flow_id ON alarm_report(flow_id);
CREATE INDEX idx_alarm_occurrence_time ON alarm_report(occurrence_time);
CREATE INDEX idx_notification_timestamp ON notification_report(timestamp);
CREATE INDEX idx_notification_system_id ON notification_report(system_id);
CREATE INDEX idx_notification_flow_id ON notification_report(flow_id);
CREATE INDEX idx_notification_occur_time ON notification_report(occur_time);
```

### 5. TimescaleDB优化（推荐用于生产环境）

#### 安装TimescaleDB扩展
```bash
# Ubuntu/Debian
sudo apt install timescaledb-2-postgresql-14

# 启用扩展
sudo -u postgres psql -d telemetrydb -c "CREATE EXTENSION IF NOT EXISTS timescaledb;"
```

#### 转换为超表并配置压缩和保留策略
```sql
-- 连接到数据库
\c telemetrydb
SET search_path TO telemetry;

-- 转换告警表为超表
SELECT create_hypertable('alarm_report', 'timestamp', 
    chunk_time_interval => INTERVAL '1 day',
    if_not_exists => TRUE);

-- 配置压缩策略（7天后压缩）
ALTER TABLE alarm_report SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'system_id',
    timescaledb.compress_orderby = 'timestamp DESC'
);

SELECT add_compression_policy('alarm_report', INTERVAL '7 days');

-- 配置数据保留策略（1年后删除）
SELECT add_retention_policy('alarm_report', INTERVAL '1 year');

-- 同样配置通知表
SELECT create_hypertable('notification_report', 'timestamp', 
    chunk_time_interval => INTERVAL '1 day',
    if_not_exists => TRUE);

ALTER TABLE notification_report SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'system_id',
    timescaledb.compress_orderby = 'timestamp DESC'
);

SELECT add_compression_policy('notification_report', INTERVAL '7 days');
SELECT add_retention_policy('notification_report', INTERVAL '1 year');

-- 查看超表状态
SELECT * FROM timescaledb_information.hypertables;
SELECT * FROM timescaledb_information.compression_settings;
```

### 4. 配置文件

复制并修改配置文件：
```bash
cp production-config-optimized.yaml config.yaml
```

编辑 `config.yaml`：
```yaml
# 数据库配置
database:
  host: "localhost"
  port: 5432
  user: "telemetry_app"
  password: "your_password"  # 修改为实际密码
  database: "telemetrydb"
  max_open_conns: 200
  max_idle_conns: 50
  conn_max_lifetime: "1h"

# 服务器配置
server:
  port: 50051
  max_recv_msg_size: 104857600  # 100MB
  max_concurrent_streams: 2000
  keepalive_time: "30s"         # gRPC keepalive时间
  keepalive_timeout: "5s"       # gRPC keepalive超时
  tcp_keepalive: true           # TCP keepalive
  tcp_no_delay: true            # TCP无延迟

# 监控配置
monitoring:
  enabled: true
  prometheus_enabled: true
  prometheus_port: 12112
  health_check_port: 8080
  metrics_interval: "15s"

# 日志配置
logging:
  level: "info"                 # 生产环境推荐info级别
  file: "logs/telemetry.log"
  max_size: 100                 # MB
  max_backups: 5
  max_age: 30                   # 天
  compress: true
```

### 5. 编译运行

```bash
# 编译
go build -o bin/telemetry main.go

# 运行
./bin/telemetry -config config.yaml
```

### 6. 后台运行 (生产环境)

```bash
# 使用systemd服务
sudo tee /etc/systemd/system/telemetry.service > /dev/null <<EOF
[Unit]
Description=ZTE Telemetry Data Collector
After=network.target postgresql.service

[Service]
Type=simple
User=telemetry
WorkingDirectory=/opt/telemetry
ExecStart=/opt/telemetry/bin/telemetry -config /opt/telemetry/config.yaml
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

# 启动服务
sudo systemctl daemon-reload
sudo systemctl enable telemetry
sudo systemctl start telemetry
```

## 📊 监控配置

### Prometheus配置

在Prometheus配置文件中添加：
```yaml
scrape_configs:
  - job_name: 'zte-telemetry'
    static_configs:
      - targets: ['your-server-ip:12112']
    scrape_interval: 15s
    metrics_path: /metrics
```

### 关键指标说明

| 指标名称 | 类型 | 说明 |
|---------|------|------|
| `telemetry_buffer_size` | Gauge | 缓冲区当前大小 |
| `telemetry_db_pool_connections` | Gauge | 数据库连接池状态 |
| `telemetry_db_records_written_total` | Counter | 写入记录总数 |
| `telemetry_db_write_errors_total` | Counter | 写入错误总数 |
| `telemetry_db_write_duration_seconds` | Histogram | 写入耗时分布 |
| `telemetry_system_goroutines` | Gauge | Goroutine数量 |
| `telemetry_system_memory_bytes` | Gauge | 内存使用情况 |

### Grafana仪表板

推荐监控查询：
```promql
# 数据写入速率 (records/second)
rate(telemetry_db_records_written_total[5m])

# 平均写入延迟
rate(telemetry_db_write_duration_seconds_sum[5m]) / rate(telemetry_db_write_duration_seconds_count[5m])

# 错误率
rate(telemetry_db_write_errors_total[5m]) / rate(telemetry_db_records_written_total[5m]) * 100

# 连接池使用率
telemetry_db_pool_connections{state="in_use"} / telemetry_db_pool_connections{state="open"} * 100

# 缓冲区总大小
sum(telemetry_buffer_size)
```

## 🔧 配置参数详解

### 缓冲区配置
```yaml
buffer:
  max_size: 100000          # 最大缓冲区大小
  flush_threshold: 1000     # 刷新阈值
  flush_interval: "30s"     # 刷新间隔
  batch_size: 50           # 批处理大小
```

### 数据库写入配置
```yaml
database_writer:
  parallel_writers: 10      # 并行写入器数量
  max_batch_size: 50       # 最大批次大小
  retry_attempts: 5        # 重试次数
  retry_delay: "1s"        # 重试延迟
```

### 性能调优配置
```yaml
performance:
  max_procs: 8             # 最大CPU核数
  gc_percent: 75           # GC目标百分比

memory:
  max_memory_usage: "4GB"  # 最大内存限制
  gc_target_percent: 75    # GC目标百分比
```

## 🚨 故障排查

### 常见问题

#### 1. 数据库连接失败
```bash
# 检查数据库状态
sudo systemctl status postgresql

# 检查连接
psql -h localhost -U telemetry_app -d telemetrydb

# 查看日志
tail -f logs/telemetry.log | grep -i "database\|connection"
```

#### 2. 端口占用
```bash
# 检查端口占用
netstat -tlnp | grep 50051
netstat -tlnp | grep 12112

# 杀死占用进程
sudo lsof -ti:50051 | xargs sudo kill -9
```

#### 3. 内存不足
```bash
# 监控内存使用
free -h
top -p $(pgrep telemetry)

# 调整配置
# 减少 max_open_conns, buffer.max_size
```

#### 4. 数据写入失败
```bash
# 检查数据库权限
\c telemetrydb
\dn+  -- 查看schema权限

# 检查表结构
\d telemetry.platform_metrics
\d telemetry.interface_metrics
\d telemetry.subinterface_metrics
```

### 日志分析

```bash
# 实时查看日志
tail -f logs/telemetry.log

# 查看错误日志
grep -i "error\|failed" logs/telemetry.log

# 查看性能指标
grep "监控指标\|系统状态" logs/telemetry.log | tail -10

# 查看数据写入统计
grep "成功写入\|插入.*成功" logs/telemetry.log | tail -10

# 查看告警相关日志
grep -i "告警\|alarm\|notification" logs/telemetry.log | tail -10

# 调试模式查看详细解析过程
./bin/telemetry -config config.yaml -debug
```

## 🚨 告警与通知功能

### 支持的告警类型
- **设备告警**: 硬件故障、温度异常、电源问题等
- **接口告警**: 链路状态变化、流量异常等  
- **系统告警**: CPU/内存使用率、存储空间等
- **通知消息**: 配置变更、状态更新等

### 告警数据结构
```json
{
  "timestamp": "2025-09-20T16:30:00Z",
  "system_id": "GDQY-QYYYJRJ-6180H-SM-A5134",
  "flow_id": 779854,
  "alarm_type": "device-alarm",
  "severity": "emergencies",
  "alarm_text": "设备温度过高",
  "resource": "/components/component[name=PSU-1]",
  "probable_cause": "cooling-system-failure",
  "occurrence_time": "2025-09-20T16:29:45Z",  // 告警产生时间(UTC)
  "update_time": "2025-09-20T16:29:50Z",      // 告警更新时间(UTC)
  "disappeared_time": null,                    // 告警消失时间(UTC)
  "sequence_number": 12345,
  "additional_info": {
    "temperature": 85.5,
    "threshold": 75.0
  }
}
```

### 时间字段说明
- **自动转换**: Proto中的uint32 Unix时间戳自动转换为PostgreSQL TIMESTAMPTZ格式
- **UTC时间**: 所有时间字段统一使用UTC时区存储
- **可读格式**: 数据库中存储为 `2025-09-20 16:29:45+00` 格式
- **查询友好**: 支持时区转换和时间范围查询

### 告警监控查询
```sql
-- 查看最近1小时的告警（包含可读时间格式）
SELECT 
    system_id,
    flow_id,
    alarm_type,
    severity,
    occurrence_time AT TIME ZONE 'Asia/Shanghai' as occurrence_time_local,
    update_time AT TIME ZONE 'Asia/Shanghai' as update_time_local,
    disappeared_time AT TIME ZONE 'Asia/Shanghai' as disappeared_time_local,
    description
FROM telemetry.alarm_report 
WHERE timestamp >= NOW() - INTERVAL '1 hour'
ORDER BY occurrence_time DESC;

-- 按严重性和时间范围统计告警数量
SELECT 
    severity, 
    COUNT(*) as count,
    MIN(occurrence_time) as first_occurrence,
    MAX(occurrence_time) as last_occurrence
FROM telemetry.alarm_report 
WHERE occurrence_time >= NOW() - INTERVAL '24 hours'
GROUP BY severity
ORDER BY count DESC;

-- 查看特定设备的告警趋势（按小时统计）
SELECT 
    DATE_TRUNC('hour', occurrence_time) as hour, 
    COUNT(*) as alarm_count,
    COUNT(DISTINCT alarm_type) as unique_alarm_types,
    array_agg(DISTINCT severity) as severities
FROM telemetry.alarm_report 
WHERE system_id = 'GDQY-QYYYJRJ-6180H-SM-A5134'
  AND occurrence_time >= NOW() - INTERVAL '7 days'
GROUP BY hour
ORDER BY hour;

-- 查看活跃告警（未消失的告警）
SELECT 
    system_id,
    flow_id,
    alarm_type,
    severity,
    occurrence_time,
    update_time,
    EXTRACT(EPOCH FROM (NOW() - occurrence_time))/3600 as hours_since_occurrence
FROM telemetry.alarm_report 
WHERE disappeared_time IS NULL
ORDER BY occurrence_time DESC;

-- 通知消息查询
SELECT 
    system_id,
    flow_id,
    classification,
    severity,
    occur_time AT TIME ZONE 'Asia/Shanghai' as occur_time_local,
    description
FROM telemetry.notification_report 
WHERE occur_time >= NOW() - INTERVAL '1 hour'
ORDER BY occur_time DESC;
```

## 📈 性能基准

### 测试环境
- **CPU**: 8核 Intel Xeon
- **内存**: 16GB RAM
- **存储**: SSD
- **网络**: 1Gbps

### 性能指标
- **数据写入**: 10,000+ records/second
- **并发连接**: 支持100+设备同时连接
- **内存使用**: 稳定在2-4GB
- **CPU使用**: 平均30-50%

## 🤝 贡献指南

1. Fork 项目
2. 创建特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 打开 Pull Request

## 📄 许可证

本项目采用 MIT 许可证 - 查看 [LICENSE](LICENSE) 文件了解详情。

## 📞 支持

如有问题或建议，请：
1. 查看 [Issues](https://github.com/wwswwsuns/ztelem/issues)
2. 创建新的 Issue
3. 联系维护者

## 🛠 运维与排障更新（2025-10-14）

本次版本在“连接健康检查与 Prometheus 指标”的一致性方面进行了修正，并补充了 SELinux 与 systemd 的运维指引。

- 连接健康检查与指标对齐
  - 日志与 Prometheus 指标统一复用同一份原子快照（一次加锁内计算）：total/active/stale 由同一来源生成，避免时序不一致。
  - 暴露指标：
    - telemetry_grpc_connections{state="total"}
    - telemetry_grpc_connections{state="active"}
    - telemetry_grpc_connections{state="stale"}
  - 僵尸判定保持不变：最后一次收到数据时间超过 15 分钟即视为僵尸连接。

- Watchdog（僵尸连接）说明
  - 周期性检查由 telemetry-zombie-watch.timer 触发，不需要手动常驻 telemetry-zombie-watch.service。
  - 如需立即执行一次检查（不等周期），可手动运行：systemctl start telemetry-zombie-watch.service。
  - **检测方法（已优化）**：
    - 原方案：从 Prometheus /metrics 读取指标计算僵尸比例
    - **新方案（当前使用）**：直接解析程序日志 `/var/log/telemetry/telemetry.log`
    - 原因：Prometheus指标可能受心跳包影响，无法准确反映业务数据停止的情况
    - 新方案直接使用程序内部的僵尸连接判定逻辑，更加准确可靠
  - **触发条件**：僵尸连接比例 > 10%（可在脚本中调整 THRESHOLD 变量）
  - **查看检测日志**：`journalctl -t telemetry-zombie-v2 --since "10 minutes ago"`
  - 详细说明见：`docs/zombie-detection-optimization.md`

- 部署与升级（软链方式，便于回滚）
  1) 编译
     - go build -o bin/telemetry main.go
  2) 暂停定时器与服务（避免抢启）
     - systemctl stop telemetry-zombie-watch.timer || true
     - systemctl stop telemetry.service || true
  3) 备份与替换
     - ts=$(date +%Y%m%d%H%M%S)
     - if [ -x /usr/local/bin/telemetry ]; then cp -f /usr/local/bin/telemetry /usr/local/bin/telemetry.bak-$ts; fi
     - ln -sfn /home/telemetry/bin/telemetry /usr/local/bin/telemetry
  4) 启动与验证
     - systemctl start telemetry.service
     - systemctl start telemetry-zombie-watch.timer
     - 验证：curl -fsS http://127.0.0.1:12112/metrics | grep '^telemetry_grpc_connections'
     - 验证：journalctl -u telemetry.service -n 100 --no-pager | grep '连接健康检查'

- SELinux（Enforcing）下的处理
  - 查看状态：
    - getenforce（Enforcing/Permissive/Disabled）
    - sestatus
  - 若服务报数据库连接 permission denied（但本机到 127.0.0.1:5432 可通），通常是 SELinux 拒绝所致。
  - 临时验证（不建议长期）：
    - setenforce 0
    - systemctl restart telemetry.service
  - 生成并安装最小放行策略（推荐做法）：
    - 查看拒绝记录：ausearch -m avc -ts recent | tail -n 100
    - 生成策略模块：grep denied /var/log/audit/audit.log | audit2allow -M telemetry_local
    - 安装策略模块：semodule -i telemetry_local.pp
    - 恢复 Enforcing：setenforce 1；如仍有拒绝，重复上述“生成+安装”过程。
  - 纠正二进制标签（可选）：
    - restorecon -v /usr/local/bin/telemetry
  - 备选方案：将数据库连接切换为 Unix Socket（/var/run/postgresql）以绕开 TCP 策略限制，并在 pg_hba.conf 中放行 local 规则。

- 常见问题速查
  - systemctl 重启风暴：
    - systemctl reset-failed telemetry.service 后再启动；必要时临时设置 Restart=no（drop-in 覆盖）排查。
  - timer 与 service 关系：
    - 周期性执行只需启用 timer；手动立即检查才启动 .service 一次。
  - 指标与日志不一致：
    - 已统一为同一快照口径，若仍不一致，请检查 Prometheus 端点是否可达、metrics_interval 周期是否按预期运行。

## 🔐 SELinux 一键策略安装脚本

为了在 SELinux Enforcing 下稳定运行，可使用一键脚本基于审计日志生成并安装最小放行策略：

- 前置条件
  - 已复现过一次拒绝（AVC），或准备在 Permissive/Enforcing 下重启服务以产生最新拒绝日志
  - 系统具备 audit2allow 与 semodule（通常来自 policycoreutils-python-utils 与 selinux-policy-devel）

- 使用步骤
  ```bash
  # 如需复现拒绝以抓取最新 AVC（可选）
  sudo setenforce 0
  sudo systemctl restart telemetry.service

  # 生成并安装策略（脚本会优先使用 ausearch recent，回退 audit.log）
  sudo bash scripts/selinux-allow-telemetry.sh

  # 恢复 Enforcing 并验证
  sudo setenforce 1
  sudo systemctl restart telemetry.service
  ```

- 脚本说明
  - 位置：scripts/selinux-allow-telemetry.sh
  - 变量（可选）：
    - MOD_NAME：策略模块名，默认 telemetry_local
    - AUDIT_LOG：审计日志路径，默认 /var/log/audit/audit.log
    - BIN_PATH：二进制路径，默认 /usr/local/bin/telemetry
  - 脚本会：
    1) 收集最近 AVC（ausearch -m avc -ts recent），无 ausearch 时回退 grep audit.log
    2) 使用 audit2allow -M 生成 ${MOD_NAME}.pp
    3) 使用 semodule -i 安装策略模块
    4) 可选 restorecon 纠正二进制标签

- 常见问题
  - 生成空策略：说明当前无“denied”记录。请在 Enforcing 下复现一次拒绝，再运行脚本。
  - 安装后仍拒绝：再次运行服务收集新的 AVC，再次执行脚本迭代放行。
  - 不想依赖 TCP：可将数据库连接切换为 Unix Socket（/var/run/postgresql），并在 pg_hba.conf 放行 local。

## 📖 操作指引

### 增加新的 sensor_path 数据类型

当 ZTE 设备新增遥测数据类型时，按以下步骤添加支持：

#### 1. 定义 Protobuf 消息（如有新 proto）

如果新数据类型使用新的 proto 定义：
```bash
# 在 proto/ 目录下添加 .proto 文件
protoc --go_out=. --go-grpc_out=. proto/new_type/new_type.proto
```

#### 2. 添加数据模型

在 `internal/models/models.go` 中添加新结构体（或扩展现有结构体）：

```go
// 新数据类型的结构体
type NewMetricType struct {
    Timestamp     time.Time `json:"timestamp" db:"timestamp"`
    SystemID      string    `json:"system_id" db:"system_id"`
    ComponentName string    `json:"component_name" db:"component_name"`
    // ... 字段定义
}
```

#### 3. 添加解析函数

在 `internal/parser/telemetry_parser.go` 中：

```go
// 在路由表中添加新的 sensor_path
routes := []routeEntry{
    // ... 已有路由
    {"oc-platform:components/component/new-state", func(m *zteTelemetry.Telemetry) (interface{}, error) {
        return p.parseNewComponentState(m)
    }, false},
}

// 添加解析函数
func (p *TelemetryParser) parseNewComponentState(msg *zteTelemetry.Telemetry) ([]models.NewMetricType, error) {
    if len(msg.DataGpb) == 0 {
        return nil, fmt.Errorf("DataGpb为空")
    }
    
    var metrics []models.NewMetricType
    for _, dataGpb := range msg.DataGpb {
        componentInfo := p.componentPool.Get().(*platformProto.ComponentInfo)
        if err := proto.Unmarshal(dataGpb.GetContent(), componentInfo); err != nil {
            p.componentPool.Put(componentInfo)
            continue
        }
        
        // 解析逻辑...
        
        proto.Reset(componentInfo)
        p.componentPool.Put(componentInfo)
    }
    return metrics, nil
}
```

#### 4. 更新 ParseResult

在 `internal/parser/telemetry_parser.go` 的 `ParseResult` 结构体中添加新字段：

```go
type ParseResult struct {
    // ... 已有字段
    NewMetrics []models.NewMetricType
}
```

#### 5. 添加数据库写入

在 `internal/database/database.go` 中：

```go
func (db *Database) BatchInsertNewMetricsWithContext(ctx context.Context, metrics []models.NewMetricType) error {
    // 使用 COPY FROM STDIN 批量写入
    // 参考 BatchInsertPlatformMetricsWithContext 的实现
}
```

#### 6. 更新缓冲区管理器

在 `internal/buffer/buffer_manager.go` 和 `internal/buffer/sharded_map.go` 中添加对应的分片 map 和写入逻辑。

#### 7. 更新 collector

在 `internal/collector/simple_collector.go` 的 `processPublishArgs` 中添加新数据类型的处理分支。

#### 8. 添加数据库表

参考 `create_tables.sql`，为新数据类型创建表和索引。

---

### 调整数据上报频率

如需提高设备遥测数据的上报频率，需同步调整缓冲区配置：

| 上报间隔 | 建议 `flush_interval` | 建议 `flush_threshold` | 建议 `parallel_writers` |
|---|---|---|---|
| 10 分钟 | 30s | 1000 | 10 |
| 2 分钟 | 30s | 1000 | 10 |
| 1 分钟 | 15s | 500 | 10 |
| 30 秒 | 10s | 500 | 15 |
| 10 秒 | 5s | 300 | 20 |
| 5 秒 | 3s | 200 | 30 |

配置示例（10秒上报间隔）：
```yaml
buffer:
  flush_interval: "10s"
  flush_threshold: 500
database_writer:
  parallel_writers: 15
```

---

### 编译和部署

```bash
# 编译
go build -o /usr/local/bin/telemetry ./main.go

# 修复 SELinux 上下文（SELinux Enforcing 环境必须）
restorecon -v /usr/local/bin/telemetry

# 重启服务
systemctl restart telemetry.service

# 验证
curl -s http://localhost:12112/health
curl -s http://localhost:12112/metrics | grep "telemetry_"
```

### 运行测试

```bash
# 运行所有单元测试
go test -v ./internal/buffer/... ./internal/parser/... ./internal/models/...

# 静态检查
go vet ./...
```

### 查看实时日志

```bash
# 查看最近日志
tail -f /var/log/telemetry/telemetry.log

# 过滤错误
grep '"level":"error"' /var/log/telemetry/telemetry.log | tail -20

# 查看监控指标
journalctl -u telemetry.service --since "5 min ago" --no-pager
```

## 🔄 更新日志

### v2.3.0 (2026-06-30)
- ⚡ **高并发性能优化**
- 🔧 缓冲区 16 分片锁，消除 300+ 设备并发写入的全局锁竞争
- 🔧 聚合键生成改用 sync.Pool + byte buffer，零 Sprintf 分配
- 🔧 protobuf sync.Pool 复用 Telemetry/ComponentInfo/InterfaceInfo 对象
- 🔧 PlatformMetric 拆分为 7 个子结构体（CommonState/CPU/Mem/Temp/Fan/Power/Optical）
- 🔧 sensor_path 路由改用有序前缀表 + handler 映射
- 🔧 枚举转换 Sprintf 改为 strconv.Itoa + 拼接
- 🔧 连接池参数从 DatabaseConfig 读取（NewDatabaseWithConfig）
- 📊 **Prometheus 指标清理**
- 🔧 删除 grpc_connection_info（高基数，326 个独立 series）
- 🔧 删除 6 个未使用的死代码指标
- 🐛 修复 Records Processed Rate 累积计数 bug（只添加增量）
- ✅ 新增 56 个单元测试（buffer/parser/models）
- 📝 新增操作指引：增加数据类型、调整上报频率、编译部署

### v2.2.0 (2025-09-28)
- ✨ **光功率数据处理优化**
- 🔧 修复0.0 dBm光功率被写入为NULL的问题
- 🔧 新增opticalPowerPtr函数，区分有效0.0值和无光信号(-60.0)
- 🎯 优化OpticalInPower、OpticalOutPower等光功率字段处理
- 📊 提高光功率数据分析的准确性
- ✅ 生产环境300+设备验证通过

### v2.1.2 (2025-09-23)
- ✨ **gRPC连接稳定性优化**
- 🔧 新增keepalive配置：keepalive_time=30s, keepalive_timeout=5s
- 🔧 启用TCP keepalive和TCP_NODELAY优化
- 🔧 增强gRPC连接监控和自动重连机制
- 📊 新增连接状态监控指标
- 🐛 修复长时间运行后连接中断问题
- ⚡ 提升网络连接的稳定性和可靠性

### v2.1.1 (2025-09-22)
- ✨ **TimescaleDB支持**
- 🔧 支持告警表转换为TimescaleDB超表
- 🔧 配置自动压缩策略（7天后压缩）
- 🔧 配置数据保留策略（1年后删除）
- 📊 显著提升大数据量查询性能
- 💾 优化存储空间使用效率
- 📝 添加超表迁移脚本和操作指南

### v2.1.0 (2025-09-20)
- ✨ **告警时间字段优化**
- 🔧 将Unix时间戳(uint32)转换为可读的TIMESTAMPTZ格式存储
- 🔧 改进occurrence_time、update_time、disappeared_time和occur_time字段处理
- 📝 添加数据库时间字段迁移脚本(migrate_alarm_time_fields.sql)
- 🎯 增强时间数据的可读性和查询便利性
- ✅ 时间字段现在以UTC格式存储，支持时区查询

### v2.0.0 (2025-09-20)
- ✨ **新增告警与通知消息采集功能**
- ✨ 支持ZTE设备告警上报数据解析与存储
- ✨ 支持通知消息的实时采集与处理
- ✨ 智能Proto解析：支持AlarmInfo和CurrentAlarm双重解析策略
- 🔧 优化日志级别控制：普通模式只记录info日志，debug模式记录详细日志
- 📊 新增告警相关监控指标和统计功能
- 🐛 修复告警数据解析失败问题
- 📝 完善告警数据库表结构和索引

### v1.9.0 (2025-09-16)
- ✨ 实现pgx + COPY FROM STDIN高性能写入
- ✨ 添加完整Prometheus监控体系
- 🐛 修复PostgreSQL 65535参数限制问题
- 🐛 修复IPv4/IPv6网络绑定问题
- ⚡ 优化数据库连接池配置
- 📊 新增20+个业务监控指标

### v1.0.0
- 🎉 初始版本发布
- ✨ 基础gRPC数据采集功能
- ✨ PostgreSQL数据存储
- ✨ 基础监控功能