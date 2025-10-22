# 僵尸连接检测优化总结

## 问题描述

**现象**：
- 程序日志显示：`僵尸连接比例过高 (311/311 = 100.0%)`
- Prometheus指标显示：`telemetry_zombie_ratio 0`（僵尸连接为0）
- 自动重启机制未触发

**根本原因**：
- 2025-10-20 18:59:00 后数据库无新数据写入，实际业务数据已停止
- 但gRPC连接的心跳/keepalive机制持续更新 `LastDataTime`
- 导致Prometheus指标判定所有连接为"活跃"，与实际情况不符

## 解决方案

### 优化策略

从**基于Prometheus指标**的检测改为**基于程序日志**的检测：

| 对比项 | 原方案 | 新方案 |
|--------|--------|--------|
| 数据源 | Prometheus metrics | 程序日志 |
| 判定依据 | `LastDataTime` (受心跳影响) | 程序内部业务逻辑 |
| 准确性 | ❌ 不准确 | ✅ 准确 |
| 可靠性 | 低 | 高 |

### 实施步骤

1. **创建新检测脚本** `telemetry-zombie-check-v2.sh`
   - 解析日志文件 `/var/log/telemetry/telemetry.log`
   - 提取"连接健康检查"或"僵尸连接警告"信息
   - 计算僵尸连接比例
   - 超过阈值(10%)时触发重启

2. **更新systemd配置**
   - 修改 `telemetry-zombie-watch.service` 使用新脚本
   - 保持timer配置不变（每分钟检测一次）

3. **部署验证**
   - 备份旧脚本和配置
   - 部署新脚本
   - 重新加载systemd配置
   - 测试验证

## 验证结果

### 测试1：检测到僵尸连接时

```bash
# 日志显示：僵尸连接 311/311 = 100%
# 检测结果：
检测到僵尸连接警告: stale=311 total=311 ratio=100%
僵尸连接比例 100% 超过阈值 10%，重启服务
# ✅ 成功触发重启
```

### 测试2：服务恢复后

```bash
# 日志显示：活跃连接=311, 僵尸连接=0
# 检测结果：
从健康检查日志提取: total=311 active=311 stale=0 ratio=0%
僵尸连接比例 0% 正常（阈值: 10%）
# ✅ 正常，不触发重启
```

## 关键文件

### 1. 检测脚本
```
/home/telemetry/scripts/telemetry-zombie-check.sh (新版本)
/home/telemetry/scripts/telemetry-zombie-check.sh.bak (旧版本备份)
```

### 2. Systemd配置
```
/etc/systemd/system/telemetry-zombie-watch.service
/etc/systemd/system/telemetry-zombie-watch.timer
```

### 3. 文档
```
docs/zombie-detection-optimization.md (详细说明)
scripts/test-zombie-detection.sh (测试脚本)
```

## 使用说明

### 查看检测状态

```bash
# 查看timer状态
systemctl status telemetry-zombie-watch.timer

# 查看最近的检测日志
journalctl -t telemetry-zombie-v2 --since "10 minutes ago"

# 手动运行检测
systemctl start telemetry-zombie-watch.service
```

### 运行测试

```bash
# 完整测试
bash /home/telemetry/scripts/test-zombie-detection.sh

# 手动测试检测脚本
bash -x /home/telemetry/scripts/telemetry-zombie-check.sh
```

### 调整阈值

编辑脚本修改阈值（默认10%）：

```bash
vim /home/telemetry/scripts/telemetry-zombie-check.sh

# 修改这一行
THRESHOLD=10  # 改为你需要的值，如 20
```

## 监控建议

### 1. 日常监控

```bash
# 查看服务状态
systemctl status telemetry.service

# 查看最近的重启记录
journalctl -u telemetry.service | grep -E "Started|Stopped" | tail -10

# 查看检测日志
journalctl -t telemetry-zombie-v2 -f
```

### 2. 告警配置

建议配置以下告警：
- ⚠️ 僵尸连接比例 > 10% 持续5分钟
- 🚨 服务1小时内重启超过3次
- ❌ 检测脚本执行失败

### 3. 日志轮转

确保日志文件不会无限增长：

```bash
# 检查logrotate配置
cat /etc/logrotate.d/telemetry

# 手动轮转测试
logrotate -f /etc/logrotate.d/telemetry
```

## 后续优化建议

### 短期（已完成）
- ✅ 基于日志的检测机制
- ✅ 自动重启功能
- ✅ 详细的检测日志

### 中期（建议）
1. **改进程序逻辑**
   - 区分心跳数据和业务数据
   - 新增 `LastBusinessDataTime` 字段
   - 基于业务数据判定连接状态

2. **双重检测机制**
   - 主检测：基于日志（当前方案）
   - 辅助检测：基于改进后的Prometheus指标
   - 两者结合提高可靠性

### 长期（建议）
1. **渐进式重启策略**
   - 第一次检测：记录警告
   - 连续3次检测：触发重启
   - 频繁重启：暂停自动重启并告警

2. **连接管理优化**
   - 主动清理僵尸连接
   - 连接池管理
   - 优雅关闭机制

3. **可观测性增强**
   - 添加更多业务指标
   - 区分心跳和业务数据的指标
   - 连接生命周期追踪

## 故障排查

### Q1: 检测脚本未触发重启

**检查清单**：
- [ ] Timer是否运行：`systemctl status telemetry-zombie-watch.timer`
- [ ] 日志文件是否存在：`ls -la /var/log/telemetry/telemetry.log`
- [ ] 日志中是否有僵尸连接信息：`tail -100 /var/log/telemetry/telemetry.log | grep 僵尸`
- [ ] 手动运行脚本：`bash -x /home/telemetry/scripts/telemetry-zombie-check.sh`

### Q2: 服务重启失败

**检查清单**：
- [ ] Systemd权限：`systemctl restart telemetry.service`
- [ ] 服务配置：`systemctl cat telemetry.service`
- [ ] 服务日志：`journalctl -u telemetry.service -n 50`
- [ ] 二进制文件：`ls -la /usr/local/bin/telemetry`

### Q3: 日志格式变化导致解析失败

**解决方法**：
1. 查看当前日志格式：`tail -20 /var/log/telemetry/telemetry.log`
2. 更新脚本中的正则表达式
3. 测试验证：`bash /home/telemetry/scripts/test-zombie-detection.sh`

## 总结

### 成果
✅ **问题已解决**：僵尸连接检测现在能够准确识别并自动触发重启

✅ **可靠性提升**：基于程序内部逻辑，不受心跳包干扰

✅ **可维护性好**：详细的日志记录和测试脚本

### 关键改进
1. 从Prometheus指标改为日志解析
2. 使用程序内部的业务逻辑判定
3. 完善的测试和文档

### 验证通过
- ✅ 能够检测到100%僵尸连接
- ✅ 成功触发服务重启
- ✅ 服务恢复后正常运行
- ✅ 不会误触发重启

---

**更新时间**: 2025-10-20  
**版本**: v2.0  
**状态**: ✅ 已部署并验证
