# ZTE遥测数据收集系统 (ZTelem)

一个专为ZTE网络设备设计的高性能遥测数据收集和存储系统，支持大规模网络设备的实时数据采集，使用PostgreSQL + TimescaleDB进行时序数据存储和分析。

[![GitHub Stars](https://img.shields.io/github/stars/wwswwsuns/ztelem?style=flat-square)](https://github.com/wwswwsuns/ztelem/stargazers)
[![GitHub Issues](https://img.shields.io/github/issues/wwswwsuns/ztelem?style=flat-square)](https://github.com/wwswwsuns/ztelem/issues)
[![GitHub License](https://img.shields.io/github/license/wwswwsuns/ztelem?style=flat-square)](https://github.com/wwswwsuns/ztelem/blob/main/LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.19+-blue?style=flat-square)](https://golang.org/)

## 🚀 功能特性

- **🔥 高性能数据收集**：支持500+网络设备并发连接，每分钟处理20K-30K数据记录
- **📊 时序数据存储**：基于TimescaleDB的高效时序数据管理，支持数据压缩和分区
- **📈 多数据类型支持**：
  - 平台指标：CPU、内存、温度、风扇状态
  - 接口指标：流量统计、错误包、接口状态
  - 子接口指标：VLAN数据、子接口流量
- **⚡ 缓冲批量写入**：智能缓冲机制，优化数据库写入性能
- **🔧 灵活配置**：支持生产、测试、开发多环境配置管理
- **🗄️ 数据保留策略**：自动数据清理和存储优化，支持自定义保留期
- **🛡️ 安全设计**：数据库连接池、错误重试、优雅关闭
- **📝 完整日志**：结构化日志记录，支持文件和控制台输出

## 📋 系统要求

### 基础环境
- **Go**: 1.19+ (推荐 1.21+)
- **PostgreSQL**: 12+ (推荐 14+)
- **TimescaleDB**: 2.0+ (推荐 2.11+)
- **操作系统**: Linux/macOS/Windows

### 硬件建议
- **CPU**: 4核心+ (生产环境推荐8核心)
- **内存**: 8GB+ (生产环境推荐16GB+)
- **存储**: SSD硬盘，至少50GB可用空间
- **网络**: 千兆网卡，稳定的网络连接

### 生产环境规模支持
- **设备数量**: 500+ 台网络设备
- **数据吞吐**: 20K-30K 记录/分钟
- **存储需求**: ~15GB/月 (30天保留策略)
- **并发连接**: 1000+ 并发TCP连接

## 🛠️ 安装部署

### 1. 克隆项目
```bash
git clone https://github.com/wwswwsuns/ztelem.git
cd ztelem
```

### 2. 安装依赖
```bash
go mod download
```

### 3. 数据库初始化
```bash
# 创建数据库和表结构
PGPASSWORD=your_password psql -h localhost -U postgres -f create_tables.sql

# 或者使用项目提供的部署脚本
chmod +x deploy.sh
./deploy.sh
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

# 运行 (开发环境)
./bin/telemetry -config config.yaml

# 运行 (生产环境)
./bin/telemetry -config production-config-optimized.yaml

# 后台运行
nohup ./bin/telemetry -config production-config-optimized.yaml > /dev/null 2>&1 &
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

欢迎贡献代码！请遵循以下步骤：

1. **Fork 项目** - 点击右上角的 Fork 按钮
2. **创建功能分支** 
   ```bash
   git checkout -b feature/AmazingFeature
   ```
3. **提交更改**
   ```bash
   git commit -m 'Add some AmazingFeature'
   ```
4. **推送到分支**
   ```bash
   git push origin feature/AmazingFeature
   ```
5. **创建 Pull Request** - 在GitHub上创建PR

### 代码规范
- 遵循Go语言官方代码规范
- 添加必要的注释和文档
- 确保所有测试通过
- 更新相关文档

## 📄 许可证

本项目采用 MIT 许可证 - 查看 [LICENSE](LICENSE) 文件了解详情。

## ⭐ Star History

如果这个项目对你有帮助，请给个Star支持一下！

[![Star History Chart](https://api.star-history.com/svg?repos=wwswwsuns/ztelem&type=Date)](https://star-history.com/#wwswwsuns/ztelem&Date)

## 📞 联系方式

- 项目链接：[https://github.com/wwswwsuns/ztelem](https://github.com/wwswwsuns/ztelem)
- 问题反馈：[Issues](https://github.com/wwswwsuns/ztelem/issues)

## 🙏 致谢

感谢所有为这个项目做出贡献的开发者和以下开源项目：

- [PostgreSQL](https://www.postgresql.org/) - 强大的开源关系数据库
- [TimescaleDB](https://www.timescale.com/) - 时序数据库扩展
- [Go](https://golang.org/) - 高效的编程语言
- [Protocol Buffers](https://developers.google.com/protocol-buffers) - 数据序列化协议

## 📈 项目统计

![GitHub repo size](https://img.shields.io/github/repo-size/wwswwsuns/ztelem?style=flat-square)
![GitHub code size](https://img.shields.io/github/languages/code-size/wwswwsuns/ztelem?style=flat-square)
![GitHub last commit](https://img.shields.io/github/last-commit/wwswwsuns/ztelem?style=flat-square)

---

**如果这个项目对你有帮助，请考虑给个 ⭐ Star！**