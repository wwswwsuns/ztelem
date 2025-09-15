# 部署指南

本文档提供ZTE Telemetry Data Collector的详细部署说明，包括生产环境部署、Docker部署和集群部署方案。

## 🏗️ 生产环境部署

### 系统准备

#### 1. 服务器配置推荐
```bash
# 最小配置
CPU: 4核
内存: 8GB
存储: 100GB SSD
网络: 1Gbps

# 推荐配置
CPU: 8核+
内存: 16GB+
存储: 500GB+ NVMe SSD
网络: 10Gbps
```

#### 2. 操作系统优化
```bash
# 增加文件描述符限制
echo "* soft nofile 65536" >> /etc/security/limits.conf
echo "* hard nofile 65536" >> /etc/security/limits.conf

# 优化网络参数
echo "net.core.rmem_max = 134217728" >> /etc/sysctl.conf
echo "net.core.wmem_max = 134217728" >> /etc/sysctl.conf
echo "net.ipv4.tcp_rmem = 4096 65536 134217728" >> /etc/sysctl.conf
echo "net.ipv4.tcp_wmem = 4096 65536 134217728" >> /etc/sysctl.conf
sysctl -p
```

### 数据库部署

#### 1. PostgreSQL + TimescaleDB 安装
```bash
# 添加PostgreSQL官方源
wget --quiet -O - https://www.postgresql.org/media/keys/ACCC4CF8.asc | sudo apt-key add -
echo "deb http://apt.postgresql.org/pub/repos/apt/ $(lsb_release -cs)-pgdg main" | sudo tee /etc/apt/sources.list.d/pgdg.list

# 安装PostgreSQL 14
sudo apt update
sudo apt install postgresql-14 postgresql-client-14 postgresql-contrib-14

# 安装TimescaleDB (推荐用于时序数据)
sudo add-apt-repository ppa:timescale/timescaledb-ppa
sudo apt update
sudo apt install timescaledb-2-postgresql-14
sudo timescaledb-tune --quiet --yes
```

#### 2. 数据库配置优化
```bash
# 编辑PostgreSQL配置
sudo vim /etc/postgresql/14/main/postgresql.conf
```

关键配置参数：
```ini
# 内存配置
shared_buffers = 4GB                    # 25% of RAM
effective_cache_size = 12GB             # 75% of RAM
work_mem = 256MB                        # 根据并发连接数调整
maintenance_work_mem = 1GB

# 连接配置
max_connections = 300                   # 支持更多连接
max_prepared_transactions = 300

# WAL配置
wal_buffers = 64MB
checkpoint_completion_target = 0.9
wal_compression = on

# 查询优化
random_page_cost = 1.1                 # SSD优化
effective_io_concurrency = 200

# 日志配置
log_min_duration_statement = 1000      # 记录慢查询
log_checkpoints = on
log_connections = on
log_disconnections = on
```

#### 3. 创建优化的数据表
```sql
-- 连接数据库
sudo -u postgres psql

-- 创建数据库
CREATE DATABASE telemetrydb;
CREATE USER telemetry_app WITH PASSWORD 'SecurePassword123!';
GRANT ALL PRIVILEGES ON DATABASE telemetrydb TO telemetry_app;

-- 连接到telemetrydb
\c telemetrydb

-- 启用TimescaleDB扩展
CREATE EXTENSION IF NOT EXISTS timescaledb;

-- 创建schema
CREATE SCHEMA telemetry;
GRANT ALL ON SCHEMA telemetry TO telemetry_app;
ALTER USER telemetry_app SET search_path TO telemetry,public;

-- 切换到telemetry schema
SET search_path TO telemetry;

-- 创建平台指标表（时序表）
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

-- 转换为TimescaleDB超表
SELECT create_hypertable('platform_metrics', 'timestamp', chunk_time_interval => INTERVAL '1 day');

-- 创建接口指标表
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

SELECT create_hypertable('interface_metrics', 'timestamp', chunk_time_interval => INTERVAL '1 day');

-- 创建子接口指标表
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

SELECT create_hypertable('subinterface_metrics', 'timestamp', chunk_time_interval => INTERVAL '1 day');

-- 创建复合索引
CREATE INDEX idx_platform_system_time ON platform_metrics(system_id, timestamp DESC);
CREATE INDEX idx_interface_system_time ON interface_metrics(system_id, timestamp DESC);
CREATE INDEX idx_subinterface_system_time ON subinterface_metrics(system_id, timestamp DESC);

-- 设置数据保留策略（保留90天）
SELECT add_retention_policy('platform_metrics', INTERVAL '90 days');
SELECT add_retention_policy('interface_metrics', INTERVAL '90 days');
SELECT add_retention_policy('subinterface_metrics', INTERVAL '90 days');

-- 启用压缩（7天后压缩数据）
ALTER TABLE platform_metrics SET (timescaledb.compress, timescaledb.compress_segmentby = 'system_id');
ALTER TABLE interface_metrics SET (timescaledb.compress, timescaledb.compress_segmentby = 'system_id');
ALTER TABLE subinterface_metrics SET (timescaledb.compress, timescaledb.compress_segmentby = 'system_id');

SELECT add_compression_policy('platform_metrics', INTERVAL '7 days');
SELECT add_compression_policy('interface_metrics', INTERVAL '7 days');
SELECT add_compression_policy('subinterface_metrics', INTERVAL '7 days');
```

### 应用部署

#### 1. 创建部署目录
```bash
# 创建应用用户
sudo useradd -r -s /bin/false telemetry

# 创建目录结构
sudo mkdir -p /opt/telemetry/{bin,config,logs,data}
sudo chown -R telemetry:telemetry /opt/telemetry
```

#### 2. 部署应用文件
```bash
# 复制编译好的二进制文件
sudo cp bin/telemetry /opt/telemetry/bin/
sudo chmod +x /opt/telemetry/bin/telemetry

# 复制配置文件
sudo cp production-config-optimized.yaml /opt/telemetry/config/config.yaml
sudo chown telemetry:telemetry /opt/telemetry/config/config.yaml
sudo chmod 600 /opt/telemetry/config/config.yaml
```

#### 3. 生产环境配置
```yaml
# /opt/telemetry/config/config.yaml
database:
  host: "localhost"
  port: 5432
  user: "telemetry_app"
  password: "SecurePassword123!"
  database: "telemetrydb"
  max_open_conns: 200
  max_idle_conns: 50
  conn_max_lifetime: "1h"
  conn_max_idle_time: "30m"

server:
  port: 50051
  max_recv_msg_size: 104857600
  max_concurrent_streams: 2000
  keepalive_time: "30s"
  keepalive_timeout: "5s"

buffer:
  max_size: 100000
  flush_threshold: 1000
  flush_interval: "30s"
  batch_size: 50

database_writer:
  parallel_writers: 10
  max_batch_size: 50
  retry_attempts: 5
  retry_delay: "1s"
  enable_parallel_table_writes: true
  platform_writer_count: 2
  interface_writer_count: 2
  subinterface_writer_count: 1

performance:
  max_procs: 8
  gc_percent: 75

memory:
  max_memory_usage: "8GB"
  gc_target_percent: 75
  buffer_pool_size: 1000

logging:
  level: "info"
  format: "json"
  output: "file"
  file_path: "/opt/telemetry/logs/telemetry.log"
  max_size: 1000
  max_age: 30
  max_backups: 10
  compress: true

monitoring:
  enabled: true
  metrics_interval: "15s"
  health_check_port: 8080
  prometheus_enabled: true
  prometheus_port: 12112
  
  alert_thresholds:
    buffer_usage_percent: 80
    db_connection_usage_percent: 85
    memory_usage_percent: 90
    error_rate_percent: 5
```

#### 4. 创建systemd服务
```bash
sudo tee /etc/systemd/system/telemetry.service > /dev/null <<EOF
[Unit]
Description=ZTE Telemetry Data Collector
Documentation=https://github.com/wwswwsuns/ztelem
After=network.target postgresql.service
Wants=postgresql.service

[Service]
Type=simple
User=telemetry
Group=telemetry
WorkingDirectory=/opt/telemetry
ExecStart=/opt/telemetry/bin/telemetry -config /opt/telemetry/config/config.yaml
ExecReload=/bin/kill -HUP \$MAINPID
KillMode=mixed
KillSignal=SIGTERM
TimeoutStopSec=30
Restart=always
RestartSec=5
StartLimitInterval=0

# 安全配置
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/opt/telemetry/logs /opt/telemetry/data

# 资源限制
LimitNOFILE=65536
LimitNPROC=32768

[Install]
WantedBy=multi-user.target
EOF

# 启动服务
sudo systemctl daemon-reload
sudo systemctl enable telemetry
sudo systemctl start telemetry

# 检查状态
sudo systemctl status telemetry
```

## 🐳 Docker部署

### 1. 创建Dockerfile
```dockerfile
# Dockerfile
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o telemetry main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /root/

COPY --from=builder /app/telemetry .
COPY --from=builder /app/production-config-optimized.yaml ./config.yaml

# 创建日志目录
RUN mkdir -p /var/log/telemetry

EXPOSE 50051 12112 8080

CMD ["./telemetry", "-config", "config.yaml"]
```

### 2. 创建docker-compose.yml
```yaml
version: '3.8'

services:
  postgres:
    image: timescale/timescaledb:latest-pg14
    environment:
      POSTGRES_DB: telemetrydb
      POSTGRES_USER: telemetry_app
      POSTGRES_PASSWORD: SecurePassword123!
      TIMESCALEDB_TELEMETRY: off
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./init.sql:/docker-entrypoint-initdb.d/init.sql
    ports:
      - "5432:5432"
    command: >
      postgres
      -c shared_buffers=256MB
      -c max_connections=300
      -c effective_cache_size=1GB
      -c work_mem=16MB
      -c maintenance_work_mem=256MB
      -c wal_buffers=16MB
      -c checkpoint_completion_target=0.9
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U telemetry_app -d telemetrydb"]
      interval: 30s
      timeout: 10s
      retries: 3

  telemetry:
    build: .
    depends_on:
      postgres:
        condition: service_healthy
    ports:
      - "50051:50051"
      - "12112:12112"
      - "8080:8080"
    volumes:
      - ./logs:/var/log/telemetry
      - ./config/docker-config.yaml:/root/config.yaml
    environment:
      - DB_HOST=postgres
      - DB_PORT=5432
      - DB_USER=telemetry_app
      - DB_PASSWORD=SecurePassword123!
      - DB_NAME=telemetrydb
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "wget", "--quiet", "--tries=1", "--spider", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3

  prometheus:
    image: prom/prometheus:latest
    ports:
      - "9090:9090"
    volumes:
      - ./monitoring/prometheus.yml:/etc/prometheus/prometheus.yml
      - prometheus_data:/prometheus
    command:
      - '--config.file=/etc/prometheus/prometheus.yml'
      - '--storage.tsdb.path=/prometheus'
      - '--web.console.libraries=/etc/prometheus/console_libraries'
      - '--web.console.templates=/etc/prometheus/consoles'
      - '--storage.tsdb.retention.time=30d'
      - '--web.enable-lifecycle'

  grafana:
    image: grafana/grafana:latest
    ports:
      - "3000:3000"
    volumes:
      - grafana_data:/var/lib/grafana
      - ./monitoring/grafana/dashboards:/etc/grafana/provisioning/dashboards
      - ./monitoring/grafana/datasources:/etc/grafana/provisioning/datasources
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=admin123
      - GF_USERS_ALLOW_SIGN_UP=false

volumes:
  postgres_data:
  prometheus_data:
  grafana_data:
```

### 3. 部署命令
```bash
# 构建和启动
docker-compose up -d

# 查看日志
docker-compose logs -f telemetry

# 停止服务
docker-compose down

# 更新服务
docker-compose pull
docker-compose up -d --force-recreate
```

## ☸️ Kubernetes部署

### 1. 创建命名空间和配置
```yaml
# namespace.yaml
apiVersion: v1
kind: Namespace
metadata:
  name: telemetry

---
# configmap.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: telemetry-config
  namespace: telemetry
data:
  config.yaml: |
    database:
      host: "postgres-service"
      port: 5432
      user: "telemetry_app"
      password: "SecurePassword123!"
      database: "telemetrydb"
      max_open_conns: 200
      max_idle_conns: 50
      conn_max_lifetime: "1h"
    
    server:
      port: 50051
      max_recv_msg_size: 104857600
      max_concurrent_streams: 2000
    
    monitoring:
      enabled: true
      prometheus_enabled: true
      prometheus_port: 12112
      health_check_port: 8080

---
# secret.yaml
apiVersion: v1
kind: Secret
metadata:
  name: postgres-secret
  namespace: telemetry
type: Opaque
data:
  password: U2VjdXJlUGFzc3dvcmQxMjMh  # base64 encoded
```

### 2. PostgreSQL部署
```yaml
# postgres-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: postgres
  namespace: telemetry
spec:
  replicas: 1
  selector:
    matchLabels:
      app: postgres
  template:
    metadata:
      labels:
        app: postgres
    spec:
      containers:
      - name: postgres
        image: timescale/timescaledb:latest-pg14
        env:
        - name: POSTGRES_DB
          value: "telemetrydb"
        - name: POSTGRES_USER
          value: "telemetry_app"
        - name: POSTGRES_PASSWORD
          valueFrom:
            secretKeyRef:
              name: postgres-secret
              key: password
        - name: TIMESCALEDB_TELEMETRY
          value: "off"
        ports:
        - containerPort: 5432
        volumeMounts:
        - name: postgres-storage
          mountPath: /var/lib/postgresql/data
        resources:
          requests:
            memory: "2Gi"
            cpu: "1000m"
          limits:
            memory: "4Gi"
            cpu: "2000m"
      volumes:
      - name: postgres-storage
        persistentVolumeClaim:
          claimName: postgres-pvc

---
apiVersion: v1
kind: Service
metadata:
  name: postgres-service
  namespace: telemetry
spec:
  selector:
    app: postgres
  ports:
  - port: 5432
    targetPort: 5432
  type: ClusterIP

---
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: postgres-pvc
  namespace: telemetry
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 100Gi
  storageClassName: fast-ssd
```

### 3. Telemetry应用部署
```yaml
# telemetry-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: telemetry
  namespace: telemetry
spec:
  replicas: 3
  selector:
    matchLabels:
      app: telemetry
  template:
    metadata:
      labels:
        app: telemetry
    spec:
      containers:
      - name: telemetry
        image: your-registry/zte-telemetry:latest
        ports:
        - containerPort: 50051
          name: grpc
        - containerPort: 12112
          name: metrics
        - containerPort: 8080
          name: health
        volumeMounts:
        - name: config
          mountPath: /root/config.yaml
          subPath: config.yaml
        - name: logs
          mountPath: /var/log/telemetry
        resources:
          requests:
            memory: "2Gi"
            cpu: "1000m"
          limits:
            memory: "4Gi"
            cpu: "2000m"
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
      volumes:
      - name: config
        configMap:
          name: telemetry-config
      - name: logs
        emptyDir: {}

---
apiVersion: v1
kind: Service
metadata:
  name: telemetry-service
  namespace: telemetry
  annotations:
    prometheus.io/scrape: "true"
    prometheus.io/port: "12112"
    prometheus.io/path: "/metrics"
spec:
  selector:
    app: telemetry
  ports:
  - name: grpc
    port: 50051
    targetPort: 50051
  - name: metrics
    port: 12112
    targetPort: 12112
  - name: health
    port: 8080
    targetPort: 8080
  type: LoadBalancer

---
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: telemetry-hpa
  namespace: telemetry
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: telemetry
  minReplicas: 3
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 80
```

### 4. 部署命令
```bash
# 应用所有配置
kubectl apply -f namespace.yaml
kubectl apply -f configmap.yaml
kubectl apply -f postgres-deployment.yaml
kubectl apply -f telemetry-deployment.yaml

# 检查部署状态
kubectl get pods -n telemetry
kubectl get services -n telemetry

# 查看日志
kubectl logs -f deployment/telemetry -n telemetry

# 扩缩容
kubectl scale deployment telemetry --replicas=5 -n telemetry
```

## 🔍 监控和告警

### Prometheus配置
```yaml
# monitoring/prometheus.yml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

rule_files:
  - "telemetry_rules.yml"

scrape_configs:
  - job_name: 'telemetry'
    static_configs:
      - targets: ['telemetry-service:12112']
    scrape_interval: 15s
    metrics_path: /metrics

alerting:
  alertmanagers:
    - static_configs:
        - targets:
          - alertmanager:9093
```

### 告警规则
```yaml
# monitoring/telemetry_rules.yml
groups:
- name: telemetry_alerts
  rules:
  - alert: TelemetryHighErrorRate
    expr: rate(telemetry_db_write_errors_total[5m]) / rate(telemetry_db_records_written_total[5m]) > 0.05
    for: 2m
    labels:
      severity: warning
    annotations:
      summary: "Telemetry high error rate"
      description: "Error rate is {{ $value | humanizePercentage }} for the last 5 minutes"

  - alert: TelemetryHighMemoryUsage
    expr: telemetry_system_memory_bytes{type="alloc"} / (8 * 1024 * 1024 * 1024) > 0.9
    for: 5m
    labels:
      severity: critical
    annotations:
      summary: "Telemetry high memory usage"
      description: "Memory usage is {{ $value | humanizePercentage }}"

  - alert: TelemetryDatabaseConnectionPoolExhausted
    expr: telemetry_db_pool_connections{state="in_use"} / telemetry_db_pool_connections{state="open"} > 0.9
    for: 2m
    labels:
      severity: warning
    annotations:
      summary: "Database connection pool nearly exhausted"
      description: "Connection pool usage is {{ $value | humanizePercentage }}"
```

## 🚀 性能优化建议

### 1. 数据库优化
- 使用TimescaleDB进行时序数据优化
- 定期清理历史数据
- 合理设置压缩策略
- 监控慢查询并优化索引

### 2. 应用优化
- 根据负载调整缓冲区大小
- 优化并行写入器数量
- 监控内存使用情况
- 定期重启释放内存

### 3. 网络优化
- 使用专用网络连接设备
- 配置合适的TCP参数
- 监控网络延迟和丢包

### 4. 监控优化
- 设置合理的告警阈值
- 定期检查监控指标
- 建立性能基线
- 制定容量规划

这个部署指南涵盖了从单机部署到Kubernetes集群部署的完整方案，可以根据实际需求选择合适的部署方式。