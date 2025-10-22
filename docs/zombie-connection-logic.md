# 僵尸连接判定逻辑详解

## 概述

本文档详细说明程序内部如何判定僵尸连接，以及为什么Prometheus指标会与实际情况不符。

## 核心数据结构

### ConnectionInfo 结构
```go
type ConnectionInfo struct {
    RemoteAddr    string      // 远程地址
    ConnectedAt   time.Time   // 连接建立时间
    LastDataTime  time.Time   // 最后收到数据的时间 ⚠️ 关键字段
    DataCount     int64       // 收到的数据总数
    IsActive      bool        // 连接是否活跃
}
```

### SimpleCollector 配置
```go
type SimpleCollector struct {
    connections     map[string]*ConnectionInfo  // 连接映射表
    connectionsMux  sync.RWMutex               // 读写锁
    dataTimeout     time.Duration              // 数据超时时间：15分钟
    // ...
}
```

## 判定逻辑

### 1. 僵尸连接判定标准

**核心函数**: `computeConnectionSnapshot()`

```go
func (c *SimpleCollector) computeConnectionSnapshot() (total, active, stale int, totalDataCount int64) {
    c.connectionsMux.RLock()
    defer c.connectionsMux.RUnlock()
    
    now := time.Now()
    total = len(c.connections)
    
    for _, conn := range c.connections {
        totalDataCount += conn.DataCount
        
        // 判定逻辑：最后收到数据的时间距离现在是否超过15分钟
        if now.Sub(conn.LastDataTime) <= c.dataTimeout {
            active++    // 活跃连接
        } else {
            stale++     // 僵尸连接
        }
    }
    return
}
```

**判定标准**：
- ✅ **活跃连接**: `now - LastDataTime <= 15分钟`
- ❌ **僵尸连接**: `now - LastDataTime > 15分钟`

### 2. LastDataTime 更新机制

**关键问题**: `LastDataTime` 在什么时候更新？

#### 数据接收流程

```go
func (c *SimpleCollector) Publish(stream proto.ZtedialoutService_PublishServer) error {
    // 1. 建立连接
    connID := c.registerConnection(remoteAddr)
    
    // 2. 循环接收数据
    for {
        req, err := stream.Recv()  // 接收数据
        if err != nil {
            return err
        }

        // 3. 更新连接活动时间 ⚠️ 关键点
        c.updateConnectionActivity(connID)
        
        // 4. 处理业务数据
        c.processPublishArgs(req)
        
        // 5. 发送响应
        stream.Send(response)
    }
}
```

#### 更新函数

```go
func (c *SimpleCollector) updateConnectionActivity(connID string) {
    c.connectionsMux.Lock()
    defer c.connectionsMux.Unlock()
    
    if conn, exists := c.connections[connID]; exists {
        conn.LastDataTime = time.Now()  // ⚠️ 每次收到数据都更新
        atomic.AddInt64(&conn.DataCount, 1)
    }
}
```

**关键发现**: 
- `stream.Recv()` 接收到**任何数据**都会更新 `LastDataTime`
- 包括：业务数据、心跳包、keepalive消息等

### 3. gRPC Keepalive 机制

#### Keepalive 配置

```go
opts := []grpc.ServerOption{
    grpc.KeepaliveParams(keepalive.ServerParameters{
        Time:    c.serverConfig.KeepaliveTime,      // 例如: 30秒
        Timeout: c.serverConfig.KeepaliveTimeout,   // 例如: 10秒
    }),
    grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
        MinTime:             10 * time.Second,
        PermitWithoutStream: true,  // ⚠️ 允许没有活跃流的keepalive
    }),
}
```

#### Keepalive 工作原理

```
客户端                          服务端
  |                               |
  |------ PING (keepalive) ------>|  ← stream.Recv() 收到
  |<----- PONG (keepalive) -------|
  |                               |
  |                               | ← updateConnectionActivity()
  |                               | ← LastDataTime = now
  |                               |
  |  (每30秒重复一次)              |
```

**问题所在**:
- gRPC keepalive 每30秒发送一次心跳
- `stream.Recv()` 会接收到这些心跳包
- 每次接收都会更新 `LastDataTime`
- 即使没有业务数据，`LastDataTime` 也会持续更新
- 因此 `now - LastDataTime` 永远 < 15分钟
- 所以所有连接都被判定为"活跃"

## 问题分析

### 为什么日志显示100%僵尸连接，但Prometheus显示0%？

这是一个**时间窗口问题**：

#### 场景重现

1. **2025-10-20 18:59:00** - 设备停止发送业务数据
2. **18:59:00 - 22:45:00** (约4小时) - 只有keepalive心跳，无业务数据
3. **某个时刻** - `checkConnectionHealth()` 被调用

#### 日志记录逻辑

```go
func (c *SimpleCollector) checkConnectionHealth() {
    // 调用相同的判定函数
    totalConnections, activeConnections, staleConnections, _ := c.computeConnectionSnapshot()

    if totalConnections > 0 {
        c.logger.Infof("连接健康检查: 总连接=%d, 活跃连接=%d, 僵尸连接=%d", 
            totalConnections, activeConnections, staleConnections)
    }

    // 如果僵尸连接过多，记录警告
    if totalConnections > 0 && float64(staleConnections)/float64(totalConnections) > 0.5 {
        c.logger.Errorf("警告: 僵尸连接比例过高 (%d/%d = %.1f%%)，可能需要重启服务", 
            staleConnections, totalConnections, float64(staleConnections)/float64(totalConnections)*100)
    }
}
```

#### 为什么会出现100%僵尸连接的日志？

**可能的原因**：

1. **Keepalive 暂时中断**
   - 网络抖动导致keepalive包丢失
   - 在某个时刻，所有连接的 `LastDataTime` 都超过15分钟
   - `checkConnectionHealth()` 此时被调用，记录了100%僵尸连接
   - 之后keepalive恢复，`LastDataTime` 又被更新

2. **Keepalive 配置问题**
   - 如果 `KeepaliveTime` 配置 > 15分钟
   - 那么在两次keepalive之间，连接会被判定为僵尸

3. **实际情况（最可能）**
   - 日志显示的是**真实的业务数据停止**
   - 但由于keepalive机制，Prometheus指标显示"活跃"
   - 这就是为什么需要改用基于日志的检测方法

### 当前的矛盾

| 时间点 | LastDataTime | 距离现在 | 判定结果 | Prometheus显示 |
|--------|-------------|---------|---------|---------------|
| 18:59:00 | 18:59:00 | 0分钟 | 活跃 | active |
| 19:00:00 | 19:00:00 (keepalive) | 0分钟 | 活跃 | active |
| 19:00:30 | 19:00:30 (keepalive) | 0分钟 | 活跃 | active |
| ... | ... | ... | ... | ... |
| 22:45:00 | 22:44:30 (keepalive) | 30秒 | 活跃 | active |

**实际情况**：
- 业务数据在 18:59:00 停止
- 但 keepalive 持续到 22:45:00
- 所以 Prometheus 一直显示"活跃"

## 解决方案对比

### 方案1：基于 LastDataTime（当前实现）

```go
// 判定逻辑
if now.Sub(conn.LastDataTime) <= 15分钟 {
    active++
} else {
    stale++
}
```

**问题**：
- ❌ 无法区分业务数据和keepalive
- ❌ Prometheus指标不准确
- ❌ 无法触发自动重启

### 方案2：基于日志解析（已实施）

```bash
# 从日志中提取真实的连接状态
tail -100 /var/log/telemetry/telemetry.log | grep "连接健康检查"
# 输出: 总连接=311, 活跃连接=0, 僵尸连接=311
```

**优点**：
- ✅ 使用程序内部的判定逻辑
- ✅ 不受Prometheus指标影响
- ✅ 能够正确触发重启

### 方案3：改进程序逻辑（推荐长期方案）

#### 新增字段

```go
type ConnectionInfo struct {
    RemoteAddr           string
    ConnectedAt          time.Time
    LastDataTime         time.Time      // 最后收到任何数据（包括keepalive）
    LastBusinessDataTime time.Time      // 最后收到业务数据 ⚠️ 新增
    DataCount            int64
    BusinessDataCount    int64           // 业务数据计数 ⚠️ 新增
    IsActive             bool
}
```

#### 改进判定逻辑

```go
func (c *SimpleCollector) computeConnectionSnapshot() (total, active, stale int, totalDataCount int64) {
    c.connectionsMux.RLock()
    defer c.connectionsMux.RUnlock()
    
    now := time.Now()
    total = len(c.connections)
    
    for _, conn := range c.connections {
        totalDataCount += conn.BusinessDataCount  // 只统计业务数据
        
        // 基于业务数据判定
        if now.Sub(conn.LastBusinessDataTime) <= c.dataTimeout {
            active++
        } else {
            stale++
        }
    }
    return
}
```

#### 区分数据类型

```go
func (c *SimpleCollector) Publish(stream proto.ZtedialoutService_PublishServer) error {
    for {
        req, err := stream.Recv()
        if err != nil {
            return err
        }

        // 总是更新 LastDataTime（包括keepalive）
        c.updateLastDataTime(connID)
        
        // 判断是否为业务数据
        if c.isBusinessData(req) {
            // 只有业务数据才更新 LastBusinessDataTime
            c.updateBusinessDataTime(connID)
            c.processPublishArgs(req)
        }
        
        stream.Send(response)
    }
}
```

## 配置查询

### 查看当前 Keepalive 配置

```bash
# 查看配置文件
cat /etc/telemetry/default.yaml | grep -i keepalive

# 查看程序日志中的配置信息
grep "gRPC服务启动" /var/log/telemetry/telemetry.log | tail -1
```

### 查看实际连接状态

```bash
# 查看Prometheus指标
curl -s http://127.0.0.1:12112/metrics | grep telemetry_grpc_connection_info | head -5

# 查看程序日志
tail -20 /var/log/telemetry/telemetry.log | grep "连接健康检查"
```

## 总结

### 当前判定逻辑

1. **标准**: `now - LastDataTime > 15分钟` = 僵尸连接
2. **更新**: 每次 `stream.Recv()` 都更新 `LastDataTime`
3. **问题**: keepalive 会持续更新 `LastDataTime`，导致无法检测业务数据停止

### 为什么需要基于日志的检测

- Prometheus指标基于 `LastDataTime`，受keepalive影响
- 程序日志记录的是相同的判定逻辑，但可能在keepalive中断时捕获到真实状态
- 基于日志的检测更可靠，因为它使用程序内部的判定逻辑

### 长期改进建议

1. **区分业务数据和keepalive**
   - 新增 `LastBusinessDataTime` 字段
   - 只有业务数据才更新此字段
   - 基于此字段判定僵尸连接

2. **改进Prometheus指标**
   - 导出 `last_business_data_age` 指标
   - 区分 `business_data_count` 和 `total_data_count`
   - 提供更准确的监控数据

3. **增强可观测性**
   - 记录keepalive和业务数据的比例
   - 追踪连接的完整生命周期
   - 提供更详细的诊断信息

---

**文档版本**: v1.0  
**更新时间**: 2025-10-20  
**相关文档**: 
- `docs/zombie-detection-optimization.md` - 检测优化说明
- `ZOMBIE_DETECTION_SUMMARY.md` - 优化总结
