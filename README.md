# ZTE Telemetry Data Collector

高性能的ZTE设备遥测数据采集系统，支持gRPC数据采集、PostgreSQL批量写入和Prometheus监控。

## 🚀 主要特性

### 核心功能
- **高性能数据采集**: 支持gRPC流式数据接收
- **批量数据写入**: 使用pgx + COPY FROM STDIN，解决PostgreSQL 65535参数限制
- **实时监控**: 完整的Prometheus指标体系
- **多设备支持**: 同时处理多台ZTE设备的遥测数据
- **数据类型支持**: 平台指标、接口指标、子接口指标

### 性能优化
- **连接池优化**: 200最大连接，25最小连接
- **并行写入**: 支持多表并行写入
- **内存管理**: 智能缓冲区管理，可配置刷新策略
- **错误重试**: 自动重试机制，提高数据可靠性

### 监控体系
- **Prometheus指标**: 20+个业务指标
- **实时监控**: 缓冲区状态、数据库性能、系统资源
- **告警支持**: 可配置的阈值告警
- **Web界面**: 指标说明和健康检查

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

-- 平台指标表
CREATE TABLE platform_metrics (
    timestamp TIMESTAMPTZ NOT NULL,
    system_id TEXT NOT NULL,
    component_name TEXT,
    oper_status TEXT,
    admin_status TEXT,
    alarm_status TEXT,
    temperature NUMERIC,
    cpu_usage NUMERIC,
    memory_usage NUMERIC,
    power_consumption NUMERIC
);

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

-- 创建索引优化查询性能
CREATE INDEX idx_platform_timestamp ON platform_metrics(timestamp);
CREATE INDEX idx_platform_system_id ON platform_metrics(system_id);
CREATE INDEX idx_interface_timestamp ON interface_metrics(timestamp);
CREATE INDEX idx_interface_system_id ON interface_metrics(system_id);
CREATE INDEX idx_subinterface_timestamp ON subinterface_metrics(timestamp);
CREATE INDEX idx_subinterface_system_id ON subinterface_metrics(system_id);
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

# 监控配置
monitoring:
  enabled: true
  prometheus_enabled: true
  prometheus_port: 12112
  health_check_port: 8080
  metrics_interval: "15s"
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

## 🔄 更新日志

### v2.0.0 (2025-09-16)
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