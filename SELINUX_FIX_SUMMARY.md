# SELinux权限问题修复总结

## 🔍 问题分析

### 现象
- **时间**: 2025-10-22 21:31:25
- **日志**: 僵尸连接比例过高 (311/311 = 100.0%)
- **结果**: 服务**未自动重启** ❌

### 根本原因

**SELinux阻止systemd执行检测脚本**

```
错误信息：
telemetry-zombie-watch.service: Unable to locate executable 
'/home/telemetry/scripts/telemetry-zombie-check-v2.sh': Permission denied
```

**SELinux审计日志**：
```
type=AVC avc: denied { execute } 
scontext=system_u:system_r:init_t:s0 (systemd)
tcontext=unconfined_u:object_r:user_home_t:s0 (脚本)
```

**问题**：
- 脚本的SELinux类型是 `user_home_t`（用户家目录文件）
- systemd无法执行 `user_home_t` 类型的文件
- 需要 `bin_t` 类型才能被systemd执行

## ✅ 解决方案

### 修复命令

```bash
# 1. 添加SELinux文件上下文规则
semanage fcontext -a -t bin_t "/home/telemetry/scripts/telemetry-zombie-check.*\.sh"

# 2. 应用上下文
restorecon -Rv /home/telemetry/scripts/

# 3. 验证
ls -Z /home/telemetry/scripts/telemetry-zombie-check*.sh
# 应该显示: unconfined_u:object_r:bin_t:s0
```

### 修复结果

**修复前**：
```
ls -Z /home/telemetry/scripts/telemetry-zombie-check-v2.sh
unconfined_u:object_r:user_home_t:s0  ❌
```

**修复后**：
```
ls -Z /home/telemetry/scripts/telemetry-zombie-check-v2.sh
unconfined_u:object_r:bin_t:s0  ✅
```

## 📊 验证结果

### 1. 脚本成功执行
```
10月 22 22:20:52 qyyz-telemetry telemetry-zombie-v2[733313]: 
    检测到僵尸连接警告: stale=312 total=312 ratio=100%
10月 22 22:20:52 qyyz-telemetry telemetry-zombie-v2[733314]: 
    僵尸连接比例 100% 超过阈值 10%，重启服务
```

### 2. 服务成功重启
```
10月 22 22:20:55 qyyz-telemetry systemd[1]: 
    Started telemetry.service - Telemetry Service.
```

### 3. 连接恢复正常
```
{"level":"info","msg":"监控指标 - gRPC连接: 总连接=311, 活跃连接=311, 僵尸连接=0","time":"2025-10-22 22:21:40"}
```

## 🚀 快速检查

### 检查SELinux状态
```bash
getenforce
# 如果显示 "Enforcing"，需要配置SELinux上下文
```

### 检查脚本上下文
```bash
ls -Z /home/telemetry/scripts/telemetry-zombie-check-v2.sh
# 应该显示 bin_t，不是 user_home_t
```

### 检查服务状态
```bash
systemctl status telemetry-zombie-watch.service
# 不应该有 "Permission denied" 错误
```

### 查看检测日志
```bash
journalctl -t telemetry-zombie-v2 --since "10 minutes ago"
# 应该能看到检测日志
```

## 📝 部署注意事项

### 新系统部署时必须执行

如果系统启用了SELinux（大多数RHEL/CentOS/Rocky Linux默认启用），在部署时**必须**配置SELinux上下文：

```bash
# 部署脚本后立即执行
semanage fcontext -a -t bin_t "/home/telemetry/scripts/telemetry-zombie-check.*\.sh"
restorecon -Rv /home/telemetry/scripts/
```

### 添加到部署脚本

建议在部署脚本中添加自动检测和修复：

```bash
#!/bin/bash
# deploy.sh

# ... 其他部署步骤 ...

# SELinux配置
if [[ $(getenforce) == "Enforcing" ]]; then
    echo "配置SELinux上下文..."
    semanage fcontext -a -t bin_t "/home/telemetry/scripts/telemetry-zombie-check.*\.sh"
    restorecon -Rv /home/telemetry/scripts/
    echo "SELinux配置完成"
fi

# ... 继续其他步骤 ...
```

## 🔧 故障排查

### 如果仍然失败

1. **检查SELinux审计日志**
   ```bash
   ausearch -m AVC -ts recent | grep telemetry
   ```

2. **检查脚本权限**
   ```bash
   ls -la /home/telemetry/scripts/telemetry-zombie-check-v2.sh
   # 应该是 -rwxr-xr-x
   ```

3. **手动运行脚本**
   ```bash
   bash /home/telemetry/scripts/telemetry-zombie-check-v2.sh
   # 应该能正常执行
   ```

4. **查看详细日志**
   ```bash
   journalctl -u telemetry-zombie-watch.service -n 50
   ```

## 📚 相关文档

- `docs/selinux-permission-fix.md` - 详细的SELinux问题分析和修复
- `docs/zombie-detection-optimization.md` - 僵尸连接检测优化
- `ZOMBIE_DETECTION_SUMMARY.md` - 检测机制总结

## ⏱️ 时间线

| 时间 | 事件 |
|------|------|
| 2025-10-20 22:47 | 部署新的检测脚本 |
| 2025-10-20 22:47 - 2025-10-22 22:20 | SELinux阻止脚本执行（约2天） |
| 2025-10-22 21:31 | 检测到100%僵尸连接，但未触发重启 |
| 2025-10-22 22:20 | 发现并修复SELinux问题 |
| 2025-10-22 22:20 | 成功触发重启，连接恢复正常 ✅ |

## ✅ 总结

### 问题
- SELinux阻止systemd执行位于用户家目录的脚本
- 脚本SELinux类型错误（`user_home_t` 应该是 `bin_t`）

### 解决
- 使用 `semanage` 和 `restorecon` 修改SELinux上下文
- 将脚本类型从 `user_home_t` 改为 `bin_t`

### 结果
- ✅ 脚本可以正常执行
- ✅ 检测到僵尸连接并触发重启
- ✅ 服务成功重启，连接恢复正常
- ✅ 自动重启机制正常工作

---

**修复时间**: 2025-10-22 22:20  
**影响时间**: 约2天  
**状态**: ✅ 已修复并验证
