# Telemetry数据采集器

这是一个用Go语言开发的网络设备telemetry数据采集程序，用于采集路由器/交换机的telemetry数据并写入TimescaleDB时序数据库。

## 功能特性

- 支持gRPC dialout模式数据采集
- 支持GPB和GPBKV编码格式
- 多种sensor_path数据解析
- 智能缓冲区管理和批量写入
- 完整的日志记录和错误处理
- 支持大规模设备采集（500-1000台设备）

## 系统架构

```
设备 -> gRPC(50051) -> 采集器 -> 解析器 -> 缓冲区 -> TimescaleDB
```

## 支持的数据类型

### 平台组件数据 (platform_metrics表)
- 组件通用状态 (`oc-platform:components/component/state`)
- 风扇状态 (`oc-platform:components/component/fan/state`)
- 内存状态 (`oc-platform:components/component/state/memory`)
- 存储状态 (`oc-platform:components/component/state/storage`)
- 温度状态 (`oc-platform:components/component/state/temperature`)
- 电源状态 (`oc-platform:components/component/power-supply/state`)
- 线卡状态 (`oc-platform:components/component/oc-linecard:linecard/state`)
- CPU状态 (`oc-platform:components/component/cpu/oc-cpu:utilization/state`)
- 光模块状态 (`oc-platform:components/component/oc-transceiver:transceiver/state`)

### 接口数据 (interface_metrics表)
- 接口状态 (`oc-if:interfaces/interface/state`)
- ZTE接口扩展 (`oc-if:interfaces/interface/zte-if:state-period`)
- 接口计数器 (`oc-if:interfaces/interface/state/counters`)

### 子接口数据 (subinterface_metrics表)
- 子接口状态 (`oc-if:interfaces/interface/subinterfaces/subinterface/state`)
- ZTE子接口扩展 (`oc-if:interfaces/interface/subinterfaces/subinterface/zte-if:state-period`)
- 子接口计数器 (`oc-if:interfaces/interface/subinterfaces/subinterface/state/counters`)

## 安装和配置

### 1. 环境要求
- Go 1.21+
- TimescaleDB数据库
- protoc编译器

### 2. 编译程序
```bash
# 克隆代码
git clone <repository>
cd telemetry

# 生成proto文件
make proto

# 编译程序
make build
```

### 3. 配置文件
复制并修改配置文件：
```bash
cp config.yaml.example config.yaml
```

配置示例：
```yaml
# 数据库配置
database:
  host: "localhost"
  port: 5432
  user: "telemetry_app"
  password: "your_password"
  dbname: "telemetrydb"
  schema: "telemetry"
  
# gRPC服务配置
grpc:
  port: 50051
  
# 缓冲区配置
buffer:
  max_size: 1000
  flush_interval: "30s"
  
# 日志配置
logging:
  level: "info"
  file: "logs/telemetry.log"
```

### 4. 数据库准备
确保TimescaleDB中已创建相应的表结构：
- `telemetry.platform_metrics`
- `telemetry.interface_metrics`
- `telemetry.subinterface_metrics`

## 运行程序

### 启动服务
```bash
# 前台运行
./bin/telemetry -config config.yaml

# 后台运行
nohup ./bin/telemetry -config config.yaml > /dev/null 2>&1 &

# 使用systemd管理
sudo systemctl start telemetry
```

### 调试模式
```bash
./bin/telemetry -config config.yaml -debug
```

## 监控和维护

### 日志文件
- 应用日志：`logs/telemetry.log`
- 错误日志：`logs/error.log`

### 性能监控
程序提供以下监控指标：
- 连接设备数量
- 数据处理速率
- 缓冲区使用情况
- 数据库写入性能

### 故障排除

1. **连接问题**
   - 检查端口50051是否被占用
   - 验证防火墙设置
   - 确认设备配置正确

2. **数据库问题**
   - 检查数据库连接配置
   - 验证表结构是否正确
   - 监控数据库性能

3. **内存问题**
   - 调整缓冲区大小
   - 监控内存使用情况
   - 优化批量写入策略

## 开发说明

### 项目结构
```
telemetry/
├── main.go                    # 主程序入口
├── internal/
│   ├── config/               # 配置管理
│   ├── database/             # 数据库操作
│   ├── logger/               # 日志管理
│   ├── models/               # 数据模型
│   └── collector/            # 采集器核心
├── proto/                    # Proto文件和生成代码
├── logs/                     # 日志目录
├── scripts/                  # 脚本文件
└── config.yaml              # 配置文件
```

### 添加新的sensor_path支持
1. 在`unified_parsers.go`中添加解析函数
2. 在`collector.go`中注册路由
3. 更新数据模型（如需要）
4. 添加相应的数据库字段

### 性能优化建议
- 合理设置缓冲区大小
- 优化数据库批量插入
- 使用连接池管理数据库连接
- 监控内存使用情况

## 许可证

本项目采用MIT许可证。

