# 僵尸连接检测优化说明

## 问题背景

原有的僵尸连接检测机制基于Prometheus指标（`telemetry_zombie_ratio`），但发现该指标存在以下问题：

1. **指标不准确**：Prometheus指标显示僵尸连接为0%，但实际上所有连接都已经停止接收业务数据
2. **判定标准问题**：原有的 `dataTimeout` 设置为15分钟，但可能存在心跳或keepalive机制持续更新 `LastDataTime`，导致即使没有业务数据，连接仍被判定为活跃
3. **无法触发重启**：由于Prometheus指标显示正常，自动重启机制无法被触发

## 根本原因

通过分析发现，`LastDataTime` 在每次收到任何数据（包括心跳包）时都会更新，这导致：
- 即使业务数据已经停止（如2025-10-20 18:59:00后无数据写入数据库）
- 但由于gRPC连接的心跳机制，`LastDataTime` 仍在持续更新
- 因此Prometheus指标显示所有连接都是"活跃"的

## 解决方案

### 1. 新的检测方法

创建了新的检测脚本 `telemetry-zombie-check-v2.sh`，采用**基于日志的检测方法**：

- **数据源**：直接解析程序日志 `/var/log/telemetry/telemetry.log`
- **检测逻辑**：
  1. 查找最近的"僵尸连接比例过高"警告日志
  2. 或者查找"连接健康检查"日志，提取僵尸连接统计
  3. 从日志中解析出：总连接数、活跃连接数、僵尸连接数
  4. 计算僵尸连接比例
  5. 如果比例超过阈值（默认10%），触发服务重启

### 2. 检测脚本特点

```bash
#!/usr/bin/env bash
# 关键特性：
# 1. 直接从程序日志提取连接状态
# 2. 支持两种日志格式：
#    - "警告: 僵尸连接比例过高 (311/311 = 100.0%)"
#    - "连接健康检查: 总连接=311, 活跃连接=0, 僵尸连接=311"
# 3. 可配置阈值（默认10%）
# 4. 记录详细的检测日志到syslog
```

### 3. 部署步骤

```bash
# 1. 备份旧脚本
cp /home/telemetry/scripts/telemetry-zombie-check.sh \
   /home/telemetry/scripts/telemetry-zombie-check.sh.bak

# 2. 部署新脚本
cp /home/telemetry/scripts/telemetry-zombie-check-v2.sh \
   /home/telemetry/scripts/telemetry-zombie-check.sh

# 3. 更新systemd服务配置
cp systemd/telemetry-zombie-watch.service \
   /etc/systemd/system/telemetry-zombie-watch.service

# 4. 重新加载systemd配置
systemctl daemon-reload

# 5. 验证配置
systemctl status telemetry-zombie-watch.timer
```

## 验证测试

### 测试1：手动运行检测脚本

```bash
bash -x /home/telemetry/scripts/telemetry-zombie-check.sh
```

**预期结果**：
- 如果检测到僵尸连接比例>10%，会触发服务重启
- 日志记录到syslog：`journalctl -t telemetry-zombie-v2`

### 测试2：查看检测日志

```bash
# 查看最近的检测日志
journalctl -t telemetry-zombie-v2 --since "10 minutes ago"

# 查看timer触发情况
systemctl status telemetry-zombie-watch.timer
```

## 配置说明

### 阈值调整

如果需要调整僵尸连接比例阈值，编辑脚本：

```bash
vim /home/telemetry/scripts/telemetry-zombie-check.sh

# 修改这一行（默认10%）
THRESHOLD=10
```

### 检测频率

检测频率由timer控制，当前配置为每分钟检测一次：

```bash
# 查看timer配置
cat /etc/systemd/system/telemetry-zombie-watch.timer

# 修改检测频率（如改为每2分钟）
# OnUnitActiveSec=120s
```

## 监控建议

### 1. 日志监控

```bash
# 实时监控检测日志
journalctl -t telemetry-zombie-v2 -f

# 查看重启历史
journalctl -u telemetry.service | grep -E "Started|Stopped"
```

### 2. 告警配置

建议配置以下告警：
- 僵尸连接比例持续>10%超过5分钟
- 服务频繁重启（1小时内重启超过3次）
- 检测脚本执行失败

### 3. 性能影响

新的检测方法性能影响极小：
- 只读取日志文件最后100行
- 使用grep和awk进行文本处理
- 每次执行耗时<100ms

## 与原方案对比

| 特性 | 原方案（Prometheus指标） | 新方案（日志解析） |
|------|------------------------|------------------|
| 数据源 | Prometheus metrics | 程序日志 |
| 准确性 | ❌ 不准确（受心跳影响） | ✅ 准确（基于业务逻辑） |
| 实时性 | ✅ 实时 | ✅ 实时（1分钟延迟） |
| 可靠性 | ❌ 低（指标可能失效） | ✅ 高（日志更可靠） |
| 复杂度 | 简单 | 简单 |
| 依赖 | Prometheus服务 | 日志文件 |

## 后续优化建议

### 1. 改进程序逻辑（推荐）

在程序中区分心跳数据和业务数据：

```go
type ConnectionInfo struct {
    LastDataTime     time.Time  // 最后收到任何数据的时间
    LastBusinessData time.Time  // 最后收到业务数据的时间（新增）
    // ...
}

// 判定僵尸连接时使用 LastBusinessData
if now.Sub(conn.LastBusinessData) > c.dataTimeout {
    status = "stale"
}
```

### 2. 双重检测机制

结合Prometheus指标和日志检测，提高可靠性：
- 主检测：基于日志
- 辅助检测：基于Prometheus指标
- 只有两者都确认时才触发重启

### 3. 渐进式重启

避免频繁重启影响服务：
- 第一次检测到问题：记录警告
- 连续3次检测到问题：触发重启
- 1小时内重启超过3次：发送告警，暂停自动重启

## 故障排查

### 问题1：检测脚本未触发重启

**排查步骤**：
```bash
# 1. 检查timer是否运行
systemctl status telemetry-zombie-watch.timer

# 2. 手动运行脚本查看输出
bash -x /home/telemetry/scripts/telemetry-zombie-check.sh

# 3. 检查日志文件权限
ls -la /var/log/telemetry/telemetry.log

# 4. 查看最近的日志内容
tail -100 /var/log/telemetry/telemetry.log | grep -E "僵尸|健康检查"
```

### 问题2：服务重启失败

**排查步骤**：
```bash
# 1. 检查systemctl权限
systemctl restart telemetry.service

# 2. 查看服务状态
systemctl status telemetry.service

# 3. 查看服务日志
journalctl -u telemetry.service -n 50
```

### 问题3：日志格式变化

如果程序日志格式发生变化，需要更新脚本中的正则表达式：

```bash
# 编辑脚本
vim /home/telemetry/scripts/telemetry-zombie-check.sh

# 更新grep模式和正则表达式
```

## 更新历史

- **2025-10-20**: 初始版本，基于日志的僵尸连接检测
- 优化原因：Prometheus指标不准确，无法正确反映僵尸连接状态
- 解决方案：直接解析程序日志，使用程序内部的判定逻辑
