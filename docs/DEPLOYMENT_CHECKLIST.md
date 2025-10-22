# 僵尸连接检测优化 - 部署清单

## ✅ 已完成的工作

### 1. 脚本文件
- ✅ `/home/telemetry/scripts/telemetry-zombie-check.sh` - 新版检测脚本（基于日志）
- ✅ `/home/telemetry/scripts/telemetry-zombie-check.sh.bak` - 旧版备份（基于Prometheus）
- ✅ `/home/telemetry/scripts/telemetry-zombie-check-v2.sh` - 新版源文件
- ✅ `/home/telemetry/scripts/test-zombie-detection.sh` - 测试脚本

### 2. Systemd配置
- ✅ `/etc/systemd/system/telemetry-zombie-watch.service` - 更新为使用新脚本
- ✅ `/etc/systemd/system/telemetry-zombie-watch.service.bak` - 旧版备份
- ✅ `/etc/systemd/system/telemetry-zombie-watch.timer` - 保持不变（每分钟检测）
- ✅ `systemctl daemon-reload` - 已重新加载配置

### 3. 文档
- ✅ `docs/zombie-detection-optimization.md` - 详细技术文档
- ✅ `ZOMBIE_DETECTION_SUMMARY.md` - 优化总结
- ✅ `DEPLOYMENT_CHECKLIST.md` - 本文件
- ✅ `README.md` - 已更新相关说明

### 4. 服务状态
- ✅ `telemetry.service` - Active (已重启，连接恢复正常)
- ✅ `telemetry-zombie-watch.timer` - Active (正在运行)
- ✅ 检测功能已验证工作正常

## 📋 验证步骤

### 快速验证
```bash
# 1. 运行完整测试
bash /home/telemetry/scripts/test-zombie-detection.sh

# 2. 查看服务状态
systemctl status telemetry.service
systemctl status telemetry-zombie-watch.timer

# 3. 查看最近的检测日志
journalctl -t telemetry-zombie-v2 --since "1 hour ago"
```

### 详细验证
```bash
# 1. 检查脚本文件
ls -lh /home/telemetry/scripts/telemetry-zombie-check*

# 2. 检查systemd配置
cat /etc/systemd/system/telemetry-zombie-watch.service

# 3. 手动运行检测
bash -x /home/telemetry/scripts/telemetry-zombie-check.sh

# 4. 查看程序日志
tail -50 /var/log/telemetry/telemetry.log | grep -E "僵尸|健康检查"

# 5. 查看Prometheus指标
curl -s http://127.0.0.1:12112/metrics | grep -E "telemetry_grpc_connections|telemetry_zombie_ratio"
```

## 🔧 配置说明

### 检测阈值
当前配置：僵尸连接比例 > **10%** 触发重启

修改方法：
```bash
vim /home/telemetry/scripts/telemetry-zombie-check.sh
# 修改 THRESHOLD=10 这一行
```

### 检测频率
当前配置：每 **1分钟** 检测一次

修改方法：
```bash
vim /etc/systemd/system/telemetry-zombie-watch.timer
# 修改 OnUnitActiveSec=60s 这一行
systemctl daemon-reload
systemctl restart telemetry-zombie-watch.timer
```

## 📊 监控命令

### 实时监控
```bash
# 监控检测日志
journalctl -t telemetry-zombie-v2 -f

# 监控服务日志
journalctl -u telemetry.service -f

# 监控程序日志
tail -f /var/log/telemetry/telemetry.log | grep -E "僵尸|健康检查"
```

### 历史查询
```bash
# 查看最近的重启记录
journalctl -u telemetry.service | grep -E "Started|Stopped" | tail -20

# 查看最近的检测记录
journalctl -t telemetry-zombie-v2 --since "1 day ago"

# 查看最近的僵尸连接警告
grep "僵尸连接比例过高" /var/log/telemetry/telemetry.log | tail -20
```

## 🚨 告警建议

建议配置以下监控告警：

### 1. 僵尸连接告警
```bash
# 条件：僵尸连接比例 > 10% 持续 5 分钟
# 命令：
journalctl -t telemetry-zombie-v2 --since "5 minutes ago" | grep "超过阈值"
```

### 2. 频繁重启告警
```bash
# 条件：1小时内重启超过 3 次
# 命令：
journalctl -u telemetry.service --since "1 hour ago" | grep "Started" | wc -l
```

### 3. 检测失败告警
```bash
# 条件：检测脚本执行失败
# 命令：
systemctl status telemetry-zombie-watch.service | grep "failed"
```

## 🔄 回滚方案

如果需要回滚到旧版本：

```bash
# 1. 停止timer
systemctl stop telemetry-zombie-watch.timer

# 2. 恢复旧脚本
cp /home/telemetry/scripts/telemetry-zombie-check.sh.bak \
   /home/telemetry/scripts/telemetry-zombie-check.sh

# 3. 恢复旧配置
cp /etc/systemd/system/telemetry-zombie-watch.service.bak \
   /etc/systemd/system/telemetry-zombie-watch.service

# 4. 重新加载并启动
systemctl daemon-reload
systemctl start telemetry-zombie-watch.timer

# 5. 验证
systemctl status telemetry-zombie-watch.timer
```

## 📝 变更记录

### 2025-10-20 - v2.0
- **变更内容**：从Prometheus指标检测改为日志解析检测
- **变更原因**：Prometheus指标受心跳包影响，无法准确反映业务数据停止
- **影响范围**：僵尸连接检测机制
- **测试结果**：✅ 已验证，能够正确检测并触发重启
- **回滚方案**：已准备备份文件，可快速回滚

### 关键改进
1. ✅ 检测准确性：从不准确 → 准确
2. ✅ 自动重启：从不触发 → 正常触发
3. ✅ 可维护性：添加详细文档和测试脚本
4. ✅ 可观测性：增强日志记录

## 🎯 后续计划

### 短期（1-2周）
- [ ] 观察新检测机制的稳定性
- [ ] 收集检测日志，分析误报率
- [ ] 根据实际情况调整阈值

### 中期（1-2月）
- [ ] 改进程序逻辑，区分心跳和业务数据
- [ ] 实现双重检测机制（日志+改进的指标）
- [ ] 添加更多业务指标

### 长期（3-6月）
- [ ] 实现渐进式重启策略
- [ ] 优化连接管理机制
- [ ] 增强可观测性

## ✅ 部署确认

- [x] 所有脚本文件已部署
- [x] Systemd配置已更新
- [x] 服务已重启并恢复正常
- [x] 检测功能已验证
- [x] 文档已完善
- [x] 备份文件已保存
- [x] 回滚方案已准备

**部署状态**: ✅ 完成  
**部署时间**: 2025-10-20 22:45-22:55  
**部署人员**: Kiro AI Assistant  
**验证状态**: ✅ 通过  

---

## 📞 支持

如有问题，请参考：
1. `docs/zombie-detection-optimization.md` - 详细技术文档
2. `ZOMBIE_DETECTION_SUMMARY.md` - 优化总结
3. 运行测试脚本：`bash /home/telemetry/scripts/test-zombie-detection.sh`
