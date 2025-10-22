# 僵尸连接检测 - 快速参考

## 🚀 快速命令

### 查看状态
```bash
# 查看服务状态
systemctl status telemetry.service

# 查看检测timer状态
systemctl status telemetry-zombie-watch.timer

# 查看最近的检测日志
journalctl -t telemetry-zombie-v2 --since "10 minutes ago"

# 查看程序日志中的连接状态
tail -20 /var/log/telemetry/telemetry.log | grep "连接健康检查"
```

### 手动操作
```bash
# 手动触发一次检测（不等timer）
systemctl start telemetry-zombie-watch.service

# 手动重启telemetry服务
systemctl restart telemetry.service

# 运行完整测试
bash /home/telemetry/scripts/test-zombie-detection.sh
```

### 实时监控
```bash
# 监控检测日志
journalctl -t telemetry-zombie-v2 -f

# 监控程序日志
tail -f /var/log/telemetry/telemetry.log | grep -E "僵尸|健康检查"
```

## 📊 关键指标

### 正常状态
```
连接健康检查: 总连接=311, 活跃连接=311, 僵尸连接=0
僵尸连接比例: 0%
```

### 异常状态（会触发重启）
```
警告: 僵尸连接比例过高 (311/311 = 100.0%)
僵尸连接比例: 100% > 阈值 10%
→ 自动重启服务
```

## ⚙️ 配置

### 当前配置
- **检测频率**: 每1分钟
- **触发阈值**: 僵尸连接比例 > 10%
- **检测方法**: 解析程序日志
- **日志标签**: telemetry-zombie-v2

### 修改阈值
```bash
vim /home/telemetry/scripts/telemetry-zombie-check.sh
# 修改: THRESHOLD=10
```

### 修改检测频率
```bash
vim /etc/systemd/system/telemetry-zombie-watch.timer
# 修改: OnUnitActiveSec=60s
systemctl daemon-reload
systemctl restart telemetry-zombie-watch.timer
```

## 🔍 故障排查

### 问题：检测未触发重启
```bash
# 1. 检查timer是否运行
systemctl status telemetry-zombie-watch.timer

# 2. 查看最近的检测日志
journalctl -t telemetry-zombie-v2 --since "5 minutes ago"

# 3. 手动运行检测脚本
bash -x /home/telemetry/scripts/telemetry-zombie-check.sh

# 4. 检查日志文件
tail -100 /var/log/telemetry/telemetry.log | grep -E "僵尸|健康检查"
```

### 问题：服务频繁重启
```bash
# 1. 查看重启历史
journalctl -u telemetry.service --since "1 hour ago" | grep "Started"

# 2. 查看检测日志
journalctl -t telemetry-zombie-v2 --since "1 hour ago"

# 3. 临时停止自动检测
systemctl stop telemetry-zombie-watch.timer

# 4. 调高阈值或延长检测间隔
```

## 📁 重要文件

### 脚本
- `/home/telemetry/scripts/telemetry-zombie-check.sh` - 检测脚本
- `/home/telemetry/scripts/test-zombie-detection.sh` - 测试脚本

### 配置
- `/etc/systemd/system/telemetry-zombie-watch.service` - 检测服务
- `/etc/systemd/system/telemetry-zombie-watch.timer` - 定时器

### 日志
- `/var/log/telemetry/telemetry.log` - 程序日志
- `journalctl -t telemetry-zombie-v2` - 检测日志

### 文档
- `docs/zombie-detection-optimization.md` - 详细文档
- `ZOMBIE_DETECTION_SUMMARY.md` - 优化总结
- `DEPLOYMENT_CHECKLIST.md` - 部署清单

## 🎯 常见场景

### 场景1：查看当前连接状态
```bash
tail -20 /var/log/telemetry/telemetry.log | grep "连接健康检查" | tail -1
```

### 场景2：查看最近是否有重启
```bash
journalctl -u telemetry.service --since "1 hour ago" | grep -E "Started|Stopped"
```

### 场景3：验证检测功能
```bash
bash /home/telemetry/scripts/test-zombie-detection.sh
```

### 场景4：临时禁用自动重启
```bash
systemctl stop telemetry-zombie-watch.timer
# 完成维护后记得启动：
systemctl start telemetry-zombie-watch.timer
```

## 📞 获取帮助

1. 运行测试脚本查看详细状态
2. 查看详细文档：`docs/zombie-detection-optimization.md`
3. 查看部署清单：`DEPLOYMENT_CHECKLIST.md`
