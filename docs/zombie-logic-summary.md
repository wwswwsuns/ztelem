# 僵尸连接判定逻辑 - 快速总结

## 核心判定逻辑

```
僵尸连接 = (当前时间 - LastDataTime) > 15分钟
```

## 问题根源

### LastDataTime 更新时机

```
stream.Recv() 接收到任何数据 → updateConnectionActivity() → LastDataTime = now
                    ↓
            包括以下所有类型：
            • 业务数据（telemetry数据）
            • gRPC keepalive PING
            • gRPC keepalive PONG
            • 其他控制消息
```

### Keepalive 机制影响

```
时间轴：
18:59:00  业务数据停止 ❌
19:00:00  keepalive ✓ → LastDataTime更新
19:00:30  keepalive ✓ → LastDataTime更新
19:01:00  keepalive ✓ → LastDataTime更新
...
22:45:00  keepalive ✓ → LastDataTime更新

结果：LastDataTime 持续更新，连接被判定为"活跃"
实际：业务数据已停止4小时，应该是"僵尸"
```

## 数据流图

```
┌─────────────┐
│   设备端    │
└──────┬──────┘
       │
       │ ① 业务数据 (18:59前)
       │ ② Keepalive (持续)
       ↓
┌─────────────────────────┐
│  gRPC Server            │
│  stream.Recv()          │
└──────┬──────────────────┘
       │
       │ 接收到任何数据
       ↓
┌─────────────────────────┐
│ updateConnectionActivity│
│ LastDataTime = now      │
└──────┬──────────────────┘
       │
       ↓
┌─────────────────────────┐
│ computeConnectionSnapshot│
│ if (now - LastDataTime) │
│    > 15分钟             │
│    → 僵尸连接           │
│ else                    │
│    → 活跃连接           │
└──────┬──────────────────┘
       │
       ├─→ Prometheus指标 (不准确)
       │   telemetry_zombie_ratio = 0
       │
       └─→ 程序日志 (准确)
           "僵尸连接比例过高 100%"
```

## 为什么日志准确而指标不准确？

### 日志记录时机

```go
// 每分钟执行一次
func (c *SimpleCollector) startConnectionMonitor() {
    ticker := time.NewTicker(1 * time.Minute)
    for {
        case <-ticker.C:
            c.checkConnectionHealth()  // 记录日志
    }
}
```

### 可能的情况

**情况1：Keepalive 暂时中断**
```
22:44:00  keepalive ✓ → LastDataTime = 22:44:00
22:44:30  keepalive ✓ → LastDataTime = 22:44:30
22:45:00  网络抖动，keepalive丢失 ❌
22:45:10  checkConnectionHealth() 被调用
          now = 22:45:10
          LastDataTime = 22:44:30
          差值 = 40秒 < 15分钟
          → 判定为活跃 ✓

但如果：
22:29:00  最后一次keepalive
22:29:00 - 22:45:00  keepalive全部丢失
22:45:10  checkConnectionHealth() 被调用
          now = 22:45:10
          LastDataTime = 22:29:00
          差值 = 16分钟 > 15分钟
          → 判定为僵尸 ❌
          → 记录日志 "僵尸连接比例过高 100%"
```

**情况2：Keepalive 配置过长**
```
如果 KeepaliveTime = 20分钟
那么两次keepalive之间，连接会被判定为僵尸
```

## 三种数据源对比

| 数据源 | 更新频率 | 判定依据 | 受keepalive影响 | 准确性 |
|--------|---------|---------|----------------|--------|
| Prometheus指标 | 实时 | LastDataTime | ✓ 是 | ❌ 不准确 |
| 程序日志 | 每分钟 | LastDataTime | ✓ 是 | ⚠️ 部分准确 |
| 数据库写入 | 实时 | 实际业务数据 | ✗ 否 | ✅ 最准确 |

## 解决方案

### 当前方案：基于日志检测

```bash
# 从日志中提取连接状态
tail -100 /var/log/telemetry/telemetry.log | grep "连接健康检查"

# 如果显示：僵尸连接 > 10%
# → 触发重启
```

**优点**：
- 使用程序内部逻辑
- 可能捕获到keepalive中断时的真实状态
- 实施简单

**缺点**：
- 仍然依赖 LastDataTime
- 如果keepalive正常，仍然无法检测

### 推荐方案：区分业务数据

```go
type ConnectionInfo struct {
    LastDataTime         time.Time  // 任何数据（包括keepalive）
    LastBusinessDataTime time.Time  // 只有业务数据 ⚠️ 新增
}

// 判定逻辑改为：
if now.Sub(conn.LastBusinessDataTime) > 15分钟 {
    stale++  // 僵尸连接
}
```

## 验证方法

### 1. 查看数据库最后写入时间

```sql
-- 查看最后一条数据的时间
SELECT MAX(timestamp) FROM telemetry_data;
-- 结果：2025-10-20 18:59:00

-- 当前时间：2025-10-20 22:45:00
-- 差值：约4小时，业务数据确实停止
```

### 2. 查看连接的 last_data_age

```bash
curl -s http://127.0.0.1:12112/metrics | grep last_data_age | head -5
# 结果：所有连接的 last_data_age 都是 1分钟左右
# 说明：keepalive 正常工作
```

### 3. 查看 DataCount 变化

```bash
# 记录当前 DataCount
curl -s http://127.0.0.1:12112/metrics | grep data_count

# 等待1分钟后再次查询
# 如果 DataCount 增加，说明有数据接收（可能是keepalive）
# 如果 DataCount 不变，说明连接真的断了
```

## 关键配置

### 查看 Keepalive 配置

```bash
# 方法1：查看配置文件
cat /etc/telemetry/default.yaml | grep -A 5 keepalive

# 方法2：查看程序日志
grep "gRPC服务启动" /var/log/telemetry/telemetry.log | tail -1
# 输出示例：
# gRPC服务启动在端口 50051，配置: KeepAlive=30s, Timeout=10s
```

### 典型配置

```yaml
server:
  keepalive_time: 30s      # 每30秒发送一次keepalive
  keepalive_timeout: 10s   # keepalive超时时间
  max_concurrent_streams: 100
```

## 总结

### 当前状态
- ✅ 判定逻辑：基于 `LastDataTime`
- ❌ 问题：keepalive 会更新 `LastDataTime`
- ⚠️ 结果：无法准确检测业务数据停止

### 临时方案（已实施）
- 基于日志解析检测
- 可能在keepalive中断时捕获真实状态
- 已验证可以触发重启

### 长期方案（推荐）
- 区分 `LastDataTime` 和 `LastBusinessDataTime`
- 基于业务数据判定僵尸连接
- 提供准确的监控指标

---

**相关文档**：
- `docs/zombie-connection-logic.md` - 详细逻辑说明
- `docs/zombie-detection-optimization.md` - 检测优化方案
