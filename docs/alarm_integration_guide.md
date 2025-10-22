# 告警数据采集功能集成指南

## 功能概述

已成功为遥测数据采集系统添加了告警上报和通知上报功能，支持以下两个新的sensor-path：
- `alm:current-alarm-report` - 当前告警上报
- `alm:notification-report` - 通知上报

## 新增组件

### 1. 数据模型 (internal/models/models.go)
- `AlarmReportMetric` - 告警上报数据结构
- `NotificationReportMetric` - 通知上报数据结构

### 2. 解析器 (internal/parser/)
- `alarm_parser.go` - 专门处理告警数据的解析器
- 更新了 `telemetry_parser.go` 以支持告警sensor-path

### 3. 缓冲区管理 (internal/buffer/buffer_manager.go)
- 添加了告警数据的缓冲区管理
- 支持告警数据的批量写入和并行处理

### 4. 数据库层 (internal/database/database.go)
- 添加了告警数据的批量插入方法
- 支持高性能的COPY FROM STDIN操作

### 5. 数据库表结构 (scripts/create_alarm_tables.sql)
- `alarm_report` - 告警上报表
- `notification_report` - 通知上报表

## 数据库表结构

### alarm_report 表
```sql
CREATE TABLE telemetry.alarm_report (
    id BIGSERIAL PRIMARY KEY,
    timestamp TIMESTAMPTZ NOT NULL,
    system_id VARCHAR(255) NOT NULL,
    flow_id BIGINT NOT NULL,
    code BIGINT NOT NULL,
    occurrence_time BIGINT,
    update_time BIGINT,
    disappeared_time BIGINT,
    -- ... 更多字段
    description TEXT,
    caption TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
```

### notification_report 表
```sql
CREATE TABLE telemetry.notification_report (
    id BIGSERIAL PRIMARY KEY,
    timestamp TIMESTAMPTZ NOT NULL,
    system_id VARCHAR(255) NOT NULL,
    flow_id BIGINT NOT NULL,
    code BIGINT NOT NULL,
    occur_time BIGINT,
    occur_ms INTEGER,
    -- ... 更多字段
    description TEXT,
    caption TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
```

## 部署步骤

### 1. 创建数据库表
```bash
# 连接到PostgreSQL数据库
psql -h your_host -U your_user -d your_database

# 执行建表脚本
\i scripts/create_alarm_tables.sql
```

### 2. 编译程序
```bash
cd /home/telemetry
go build -o bin/telemetry main.go
```

### 3. 配置文件更新
确保配置文件中包含告警相关的sensor-path：
```yaml
telemetry:
  subscriptions:
    - sensor_path: "alm:current-alarm-report"
      sample_interval: 10000  # 10秒
    - sensor_path: "alm:notification-report"
      sample_interval: 10000  # 10秒
```

## 数据流程

1. **数据接收**: gRPC服务接收告警数据
2. **数据解析**: `alarm_parser.go` 解析protobuf数据
3. **数据缓冲**: `buffer_manager.go` 管理告警数据缓冲
4. **数据写入**: `database.go` 批量写入数据库
5. **数据存储**: PostgreSQL存储告警和通知数据

## 性能特性

- **并行处理**: 支持多个并行写入器
- **批量写入**: 使用COPY FROM STDIN提高写入性能
- **缓冲管理**: 智能缓冲区管理，避免数据丢失
- **错误重试**: 内置重试机制，提高可靠性

## 监控和统计

缓冲区统计信息现在包含告警数据：
- `AlarmReportBufferSize` - 告警上报缓冲区大小
- `NotificationReportBufferSize` - 通知上报缓冲区大小

## 注意事项

1. **时间戳处理**: 告警数据使用GPB中的时间戳，确保时序准确性
2. **唯一性**: 告警数据使用flow_id + timestamp作为唯一键
3. **数据完整性**: 告警数据不进行聚合，保持原始数据完整性
4. **索引优化**: 已为关键字段创建索引，提高查询性能

## 验证方法

### 1. 检查数据库表
```sql
-- 检查告警上报数据
SELECT COUNT(*) FROM telemetry.alarm_report;
SELECT * FROM telemetry.alarm_report ORDER BY timestamp DESC LIMIT 10;

-- 检查通知上报数据
SELECT COUNT(*) FROM telemetry.notification_report;
SELECT * FROM telemetry.notification_report ORDER BY timestamp DESC LIMIT 10;
```

### 2. 检查程序日志
```bash
tail -f logs/telemetry.log | grep -i alarm
```

### 3. 监控缓冲区状态
程序运行时会输出缓冲区统计信息，包含告警数据的缓冲区大小。

## 故障排除

1. **编译错误**: 确保所有依赖包已正确安装
2. **数据库连接**: 检查数据库连接配置和权限
3. **表不存在**: 执行建表脚本创建告警相关表
4. **数据解析错误**: 检查protobuf定义和解析逻辑

## 扩展性

该实现具有良好的扩展性：
- 可以轻松添加新的告警字段
- 支持自定义告警处理逻辑
- 可以集成告警通知系统
- 支持告警数据的实时分析

## 总结

告警数据采集功能已成功集成到现有的遥测系统中，提供了完整的告警数据收集、处理和存储能力。系统现在可以同时处理平台指标、接口指标、子接口指标以及告警数据，为网络监控提供了全面的数据支持。