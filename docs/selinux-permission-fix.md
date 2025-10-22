# SELinux权限问题修复

## 问题描述

### 现象
- 程序日志显示：`僵尸连接比例过高 (311/311 = 100.0%)`（21:31:25）
- 但telemetry服务没有自动重启
- 服务持续运行：`Active: active (running) since Mon 2025-10-20 22:47:25 CST; 1 day 23h ago`

### 根本原因

**SELinux阻止了systemd执行检测脚本**

#### 错误日志
```
telemetry-zombie-watch.service: Unable to locate executable 
'/home/telemetry/scripts/telemetry-zombie-check-v2.sh': Permission denied

telemetry-zombie-watch.service: Failed at step EXEC spawning 
/home/telemetry/scripts/telemetry-zombie-check-v2.sh: Permission denied
```

#### SELinux审计日志
```bash
ausearch -m AVC -ts recent | grep telemetry-zombie

type=AVC msg=audit(...): avc: denied { execute } for pid=... 
comm="(ck-v2.sh)" name="telemetry-zombie-check-v2.sh" 
dev="dm-2" ino=2859 
scontext=system_u:system_r:init_t:s0 
tcontext=unconfined_u:object_r:user_home_t:s0 
tclass=file permissive=0
```

**关键信息**：
- `denied { execute }` - 拒绝执行权限
- `scontext=system_u:system_r:init_t:s0` - systemd进程上下文
- `tcontext=unconfined_u:object_r:user_home_t:s0` - 脚本文件上下文（错误）
- `user_home_t` - 用户家目录类型，systemd无法执行

## 问题分析

### SELinux上下文

#### 错误的上下文
```bash
ls -Z /home/telemetry/scripts/telemetry-zombie-check-v2.sh
# 输出：
unconfined_u:object_r:user_home_t:s0 /home/telemetry/scripts/telemetry-zombie-check-v2.sh
```

**问题**：
- `user_home_t` 类型用于用户家目录中的普通文件
- systemd服务（`init_t`上下文）无法执行 `user_home_t` 类型的文件
- 需要 `bin_t` 类型才能被systemd执行

#### 正确的上下文
```bash
# 应该是：
unconfined_u:object_r:bin_t:s0 /home/telemetry/scripts/telemetry-zombie-check-v2.sh
```

### 为什么会出现这个问题？

1. **脚本创建在用户家目录**
   - `/home/telemetry/scripts/` 位于用户家目录
   - 新创建的文件默认继承父目录的SELinux上下文
   - 父目录是 `user_home_t`，所以新文件也是 `user_home_t`

2. **systemd的安全策略**
   - systemd运行在 `init_t` 上下文
   - SELinux策略不允许 `init_t` 执行 `user_home_t` 类型的文件
   - 这是一个安全特性，防止systemd执行不受信任的脚本

## 解决方案

### 方法1：修改SELinux上下文（推荐）

#### 步骤1：添加文件上下文规则
```bash
semanage fcontext -a -t bin_t "/home/telemetry/scripts/telemetry-zombie-check.*\.sh"
```

**说明**：
- `-a` - 添加新规则
- `-t bin_t` - 设置类型为 `bin_t`（可执行文件类型）
- 使用正则表达式匹配所有相关脚本

#### 步骤2：应用上下文
```bash
restorecon -Rv /home/telemetry/scripts/
```

**输出**：
```
Relabeled /home/telemetry/scripts/telemetry-zombie-check.sh 
from unconfined_u:object_r:user_home_t:s0 
to unconfined_u:object_r:bin_t:s0
```

#### 步骤3：验证
```bash
ls -Z /home/telemetry/scripts/telemetry-zombie-check*.sh
# 应该显示：
unconfined_u:object_r:bin_t:s0 /home/telemetry/scripts/telemetry-zombie-check.sh
unconfined_u:object_r:bin_t:s0 /home/telemetry/scripts/telemetry-zombie-check-v2.sh
```

#### 步骤4：测试
```bash
systemctl start telemetry-zombie-watch.service
systemctl status telemetry-zombie-watch.service
```

### 方法2：移动脚本到标准位置（备选）

```bash
# 移动到 /usr/local/bin/
mv /home/telemetry/scripts/telemetry-zombie-check-v2.sh /usr/local/bin/

# 更新systemd配置
vim /etc/systemd/system/telemetry-zombie-watch.service
# 修改：ExecStart=/usr/local/bin/telemetry-zombie-check-v2.sh

# 重新加载
systemctl daemon-reload
```

**优点**：
- `/usr/local/bin/` 默认就是 `bin_t` 类型
- 不需要手动设置SELinux上下文

**缺点**：
- 需要修改systemd配置
- 脚本不在项目目录中

### 方法3：临时禁用SELinux（不推荐）

```bash
# 临时禁用（重启后恢复）
setenforce 0

# 永久禁用（不推荐）
vim /etc/selinux/config
# 修改：SELINUX=disabled
```

**警告**：
- ⚠️ 降低系统安全性
- ⚠️ 不推荐在生产环境使用
- ⚠️ 只用于临时测试

## 验证修复

### 1. 检查SELinux上下文
```bash
ls -Z /home/telemetry/scripts/telemetry-zombie-check-v2.sh
# 应该显示 bin_t
```

### 2. 检查服务状态
```bash
systemctl status telemetry-zombie-watch.service
# 应该显示 SUCCESS，不再有 Permission denied
```

### 3. 查看检测日志
```bash
journalctl -t telemetry-zombie-v2 --since "5 minutes ago"
# 应该能看到检测日志
```

### 4. 手动触发测试
```bash
systemctl start telemetry-zombie-watch.service
journalctl -u telemetry-zombie-watch.service -n 20
```

## 修复结果

### 修复前
```
10月 22 21:31:23 qyyz-telemetry systemd[1]: Starting telemetry-zombie-watch.service...
10月 22 21:31:23 qyyz-telemetry (ck-v2.sh)[730707]: 
    telemetry-zombie-watch.service: Unable to locate executable 
    '/home/telemetry/scripts/telemetry-zombie-check-v2.sh': Permission denied
10月 22 21:31:23 qyyz-telemetry systemd[1]: 
    telemetry-zombie-watch.service: Failed with result 'exit-code'.
```

### 修复后
```
10月 22 22:20:52 qyyz-telemetry systemd[1]: Starting telemetry-zombie-watch.service...
10月 22 22:20:52 qyyz-telemetry telemetry-zombie-v2[733313]: 
    检测到僵尸连接警告: stale=312 total=312 ratio=100%
10月 22 22:20:52 qyyz-telemetry telemetry-zombie-v2[733314]: 
    僵尸连接比例 100% 超过阈值 10%，重启服务
10月 22 22:20:52 qyyz-telemetry systemd[1]: 
    telemetry-zombie-watch.service: Deactivated successfully.
10月 22 22:20:52 qyyz-telemetry systemd[1]: 
    Finished telemetry-zombie-watch.service.
```

### 服务重启成功
```
10月 22 22:20:55 qyyz-telemetry systemd[1]: Started telemetry.service - Telemetry Service.
10月 22 22:20:55 qyyz-telemetry telemetry[733332]: 
    time="2025-10-22T22:20:55+08:00" level=info msg="设置GOMAXPROCS为: 8"
```

### 连接恢复正常
```
{"level":"info","msg":"监控指标 - gRPC连接: 总连接=311, 活跃连接=311, 僵尸连接=0, 总数据=1","time":"2025-10-22 22:21:40"}
```

## 预防措施

### 1. 在部署文档中添加SELinux配置

更新 `DEPLOYMENT_CHECKLIST.md`：

```markdown
## SELinux配置（必须）

如果系统启用了SELinux（`getenforce` 显示 Enforcing），需要配置脚本的SELinux上下文：

```bash
# 添加文件上下文规则
semanage fcontext -a -t bin_t "/home/telemetry/scripts/telemetry-zombie-check.*\.sh"

# 应用上下文
restorecon -Rv /home/telemetry/scripts/

# 验证
ls -Z /home/telemetry/scripts/telemetry-zombie-check*.sh
```
```

### 2. 添加自动检测脚本

创建 `scripts/check-selinux.sh`：

```bash
#!/bin/bash
# 检查并修复SELinux上下文

if [[ $(getenforce) == "Enforcing" ]]; then
    echo "SELinux is Enforcing, checking contexts..."
    
    # 检查脚本上下文
    context=$(ls -Z /home/telemetry/scripts/telemetry-zombie-check-v2.sh | awk '{print $1}')
    
    if [[ $context != *"bin_t"* ]]; then
        echo "Incorrect SELinux context detected, fixing..."
        semanage fcontext -a -t bin_t "/home/telemetry/scripts/telemetry-zombie-check.*\.sh"
        restorecon -Rv /home/telemetry/scripts/
        echo "SELinux context fixed"
    else
        echo "SELinux context is correct"
    fi
else
    echo "SELinux is not Enforcing, no action needed"
fi
```

### 3. 在systemd服务中添加检查

可以在服务启动前检查SELinux上下文：

```ini
[Service]
Type=oneshot
ExecStartPre=/home/telemetry/scripts/check-selinux.sh
ExecStart=/home/telemetry/scripts/telemetry-zombie-check-v2.sh
```

## 常见问题

### Q1: 为什么之前（10月20日）可以工作？

**A**: 可能的原因：
1. 之前手动运行过脚本，临时设置了正确的上下文
2. 之前SELinux处于Permissive模式
3. 脚本位置不同（可能在 `/usr/local/bin/`）

### Q2: 如何检查SELinux是否是问题原因？

**A**: 
```bash
# 1. 检查SELinux状态
getenforce

# 2. 查看审计日志
ausearch -m AVC -ts recent | grep telemetry

# 3. 临时禁用SELinux测试
setenforce 0
systemctl start telemetry-zombie-watch.service
# 如果成功，说明是SELinux问题
setenforce 1  # 恢复
```

### Q3: 修复后还是失败怎么办？

**A**: 检查以下几点：
1. 脚本权限：`ls -la /home/telemetry/scripts/telemetry-zombie-check-v2.sh`
2. 脚本语法：`bash -n /home/telemetry/scripts/telemetry-zombie-check-v2.sh`
3. 手动运行：`bash /home/telemetry/scripts/telemetry-zombie-check-v2.sh`
4. 查看详细日志：`journalctl -u telemetry-zombie-watch.service -n 50`

## 总结

### 问题根源
- SELinux阻止systemd执行位于用户家目录的脚本
- 脚本的SELinux上下文是 `user_home_t`，应该是 `bin_t`

### 解决方案
- 使用 `semanage` 和 `restorecon` 修改SELinux上下文
- 将脚本类型从 `user_home_t` 改为 `bin_t`

### 验证结果
- ✅ 脚本可以正常执行
- ✅ 检测到僵尸连接并触发重启
- ✅ 服务成功重启，连接恢复正常

---

**修复时间**: 2025-10-22 22:20  
**影响时间**: 2025-10-20 22:47 - 2025-10-22 22:20 (约2天)  
**状态**: ✅ 已修复并验证
