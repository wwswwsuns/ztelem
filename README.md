# 网络设备遥测数据收集系统

一个高性能的网络设备遥测数据收集和存储系统，支持多种网络设备的实时数据采集，使用PostgreSQL + TimescaleDB进行时序数据存储。

## 🚀 功能特性

- **高性能数据收集**：支持大规模网络设备并发连接
- **时序数据存储**：基于TimescaleDB的高效时序数据管理
- **多数据类型支持**：平台指标、接口指标、子接口指标
- **缓冲批量写入**：优化数据库写入性能
- **灵活配置**：支持多环境配置管理
- **数据保留策略**：自动数据清理和存储优化

## 📋 系统要求

- Go 1.19+
- PostgreSQL 12+
- TimescaleDB 2.0+
- Linux/macOS/Windows

## 🛠️ 安装部署

### 1. 克隆项目
```bash
git clone https://github.com/your-username/telemetry-system.git
cd telemetry-system
```

### 2. 安装依赖
```bash
go mod download
```

### 3. 数据库初始化
```bash
# 创建数据库和表结构
psql -h localhost -U postgres -f create_tables.sql
```

### 4. 配置文件
```bash
# 复制示例配置文件
cp example-config.yaml config.yaml
# 编辑配置文件，修改数据库连接等参数
vim config.yaml
```

### 5. 编译运行
```bash
# 编译
make build

# 运行
./bin/telemetry -config config.yaml
```

## 📊 数据模型

### 平台指标 (Platform Metrics)
- CPU使用率、内存使用率
- 系统负载、运行时间
- 温度、风扇状态等

### 接口指标 (Interface Metrics)
- 接口流量统计
- 错误包统计
- 接口状态信息

### 子接口指标 (Subinterface Metrics)
- 子接口流量数据
- VLAN相关指标
- 子接口状态信息

## ⚙️ 配置说明

### 数据库配置
```yaml
database:
  host: "localhost"
  port: 5432
  user: "telemetry_app"
  password: "your_password"
  dbname: "telemetrydb"
  schema: "telemetry"
  max_open_conns: 50
```

### 缓冲配置
```yaml
buffer:
  size: 50000          # 缓冲区大小
  flush_interval: "30s" # 刷新间隔
  batch_size: 1000     # 批量写入大小
```

### 收集器配置
```yaml
collector:
  bind_address: "0.0.0.0:57400"
  max_connections: 1000
  read_timeout: "30s"
```

## 🔧 运维管理

### 数据保留策略
```sql
-- 设置30天数据保留
SELECT add_retention_policy('platform_metrics', INTERVAL '30 days');
SELECT add_retention_policy('interface_metrics', INTERVAL '30 days');
SELECT add_retention_policy('subinterface_metrics', INTERVAL '30 days');
```

### 性能监控
```bash
# 查看系统状态
curl http://localhost:6060/debug/pprof/

# 查看数据库连接
SELECT * FROM pg_stat_activity WHERE datname = 'telemetrydb';
```

## 📈 性能优化

### 生产环境建议
- **数据库连接池**：根据设备数量调整连接数
- **缓冲区大小**：平衡内存使用和写入性能
- **CPU核心数**：充分利用多核处理能力
- **数据保留**：定期清理历史数据

### 容量规划
- 500台设备：建议100个数据库连接，100K缓冲区
- 预期数据量：每分钟20K-30K记录
- 存储需求：约15GB/月（30天保留）

## 🐛 故障排查

### 常见问题
1. **数据写入失败**：检查数据库连接和字段映射
2. **内存使用过高**：调整缓冲区大小和刷新频率
3. **连接超时**：增加数据库连接池大小

### 日志分析
```bash
# 查看实时日志
tail -f logs/telemetry.log

# 搜索错误信息
grep "ERROR" logs/telemetry.log
```

## 🤝 贡献指南

1. Fork 项目
2. 创建功能分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 创建 Pull Request

## 📄 许可证

本项目采用 MIT 许可证 - 查看 [LICENSE](LICENSE) 文件了解详情。

## 📞 联系方式

- 项目链接：[https://github.com/your-username/telemetry-system](https://github.com/your-username/telemetry-system)
- 问题反馈：[Issues](https://github.com/your-username/telemetry-system/issues)

## 🙏 致谢

感谢所有为这个项目做出贡献的开发者！