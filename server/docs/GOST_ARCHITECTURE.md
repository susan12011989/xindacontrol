# GOST 转发架构文档

## 1. 整体架构

```
┌─────────┐     ┌──────────────────────┐     ┌──────────────────────┐     ┌──────────────┐
│   App   │────▶│   系统服务器 GOST     │────▶│   商户服务器 GOST     │────▶│   业务程序    │
│ (客户端) │     │  (relay+tls 转发)    │     │  (relay+tls 解密)    │     │  (实际服务)   │
└─────────┘     └──────────────────────┘     └──────────────────────┘     └──────────────┘
                 监听: m.Port/+1/+2          监听: 10010/10011/10012       监听: 10000/10001/10002
                 转发: 商户IP:10010/+1/+2    转发: 127.0.0.1:10000/+1/+2
```

### 1.1 端口规划

| 协议类型 | 系统服务器监听 | 商户服务器 GOST 监听 | 业务程序端口 |
|---------|---------------|---------------------|-------------|
| TCP     | m.Port        | 10010               | 10000       |
| WS      | m.Port + 1    | 10011               | 10001       |
| HTTP    | m.Port + 2    | 10012               | 10002       |

> `m.Port` 为商户的对外端口（merchants.port 字段）

### 1.2 GOST API 配置

| 配置项 | 值 |
|-------|-----|
| API 端口 | 9394 |
| 认证用户名 | tsdd |
| 认证密码 | Oa21isSdaiuwhq |

---

## 2. 服务命名规范

### 2.1 系统服务器上的服务

用于将流量从系统服务器转发到商户服务器。

| 协议 | Service 名称 | Chain 名称 | 说明 |
|------|-------------|-----------|------|
| TCP  | `tcp-relay-{port}` | `chain-tcp-relay-{port}` | 监听 port，转发到商户IP:10000 |
| WS   | `ws-relay-{port+1}` | `chain-ws-relay-{port+1}` | 监听 port+1，转发到商户IP:10001 |
| HTTP | `http-relay-{port+2}` | `chain-http-relay-{port+2}` | 监听 port+2，转发到商户IP:10002 |

**配置结构**：
```yaml
# Service 配置
services:
  - name: tcp-relay-20000
    addr: ":20000"
    handler:
      type: tcp
      chain: chain-tcp-relay-20000
    listener:
      type: tcp

# Chain 配置
chains:
  - name: chain-tcp-relay-20000
    hops:
      - name: hop-20000
        nodes:
          - name: node-20000
            addr: "商户IP:10000"
            connector:
              type: relay
            dialer:
              type: tls
```

### 2.2 商户服务器上的服务

用于接收系统服务器的转发流量，解密后转发到本地业务程序。

| 协议 | Service 名称 | 监听端口 | 转发目标 |
|------|-------------|---------|---------|
| TCP  | `local-tcp-10010` | 10010 | 127.0.0.1:10000 |
| WS   | `local-ws-10011` | 10011 | 127.0.0.1:10001 |
| HTTP | `local-http-10012` | 10012 | 127.0.0.1:10002 |

**配置结构**：
```yaml
services:
  - name: local-tcp-10010
    addr: ":10010"
    handler:
      type: relay
    listener:
      type: tls
    forwarder:
      nodes:
        - name: target-10000
          addr: "127.0.0.1:10000"
```

---

## 3. 业务流程

### 3.1 创建商户

**触发时机**：调用 `CreateMerchant` 创建新商户

**代码位置**：`server/internal/server/service/merchant/merchant.go`

**执行流程**：

```
1. 验证端口范围（必须 > 10003，且 port/port+1/port+2 均未被占用）
2. 数据库创建商户记录
3. 在商户服务器上创建 GOST 本地转发服务
   └── 调用 gostapi.CreateMerchantLocalForwards(merchantServerIP)
       ├── 创建 local-tcp-10010 → 127.0.0.1:10000
       ├── 创建 local-ws-10011 → 127.0.0.1:10001
       └── 创建 local-http-10012 → 127.0.0.1:10002
4. 在所有系统服务器上创建 GOST 转发服务
   └── 调用 createGostServicesForServers(port, serverIP)
       └── 对每个系统服务器调用 gostapi.CreateMerchantForwards(serverHost, port, merchantServerIP)
           ├── 创建 tcp-relay-{port} → 商户IP:10010
           ├── 创建 ws-relay-{port+1} → 商户IP:10011
           └── 创建 http-relay-{port+2} → 商户IP:10012
```

> 注：安全组规则（全端口放行）在创建系统服务器时已开放，无需重复操作。

**关键函数调用链**：
```go
CreateMerchant()
  ├── gostapi.CreateMerchantLocalForwards(merchantServerIP)
  │   └── createMerchantLocalForward(merchantServerIP, gostPort, appPort, protocolName)
  │       ├── CreateService() - 创建本地转发服务
  │       └── SaveConfig() - 持久化配置
  └── createGostServicesForServers(listenPort, forwardHost)
      └── gostapi.CreateMerchantForwards(serverHost, basePort, targetIP)
          └── createRelayTLSForwardWithProtocol(serverIP, port, targetIP, targetPort, protocolName)
              ├── CreateChain() - 创建转发链
              ├── CreateService() - 创建服务
              └── SaveConfig() - 持久化配置
```

---

### 3.2 创建系统服务器

**触发时机**：调用 `CreateServer` 创建 server_type=2 的系统服务器

**代码位置**：`server/internal/server/service/deploy/servers.go`

**执行流程**：

```
1. 创建服务器记录
2. 部署 GOST 服务
3. 为所有有效商户创建转发服务
   └── 调用 createGostServicesForMerchants(serverId)
       └── 对每个商户调用 gostapi.CreateMerchantForwards(serverHost, merchant.Port, merchant.ServerIP)
           ├── 创建 tcp-relay-{port} → 商户IP:10000
           ├── 创建 ws-relay-{port+1} → 商户IP:10001
           └── 创建 http-relay-{port+2} → 商户IP:10002
```

**关键函数调用链**：
```go
CreateServer() // server_type=2 时
  └── createGostServicesForMerchants(serverId)
      └── gostapi.CreateMerchantForwards(serverHost, merchant.Port, merchant.ServerIP)
```

---

### 3.3 删除商户

**触发时机**：调用 `DeleteMerchant` 删除商户

**代码位置**：`server/internal/server/service/merchant/merchant.go`

**执行流程**：

```
1. 数据库删除商户记录
2. 在所有系统服务器上删除 GOST 转发服务
   └── 异步调用 stopGostServicesOnAllSystemServers(port)
       └── 对每个系统服务器调用 gostapi.DeleteMerchantForwards(serverHost, port)
           ├── 删除 tcp-relay-{port}
           ├── 删除 ws-relay-{port+1}
           └── 删除 http-relay-{port+2}
```

**关键函数调用链**：
```go
DeleteMerchant()
  └── stopGostServicesOnAllSystemServers(port) // 异步执行
      └── gostapi.DeleteMerchantForwards(serverHost, port)
          └── deleteRelayTLSForwardWithProtocol(serverIP, port, protocolName)
              ├── DeleteService() - 删除服务
              ├── DeleteChain() - 删除转发链
              └── SaveConfig() - 持久化配置
```

---

### 3.4 商户 IP 变更

**触发时机**：调用 `ChangeMerchantIP` 更换商户服务器公网 IP

**代码位置**：`server/internal/server/service/merchant/change_ip.go`

**执行流程**：

```
1. 通过 AWS API 更换 EIP
2. 更新数据库（merchants.server_ip, servers.host）
3. 更新所有系统服务器上的 GOST 转发目标
   └── 调用 onMerchantServerIPChanged(merchantId, port, newServerIP)
       └── updateGostServicesOnSystemServers(merchantId, port, newServerIP, 0)
           └── 对每个系统服务器调用 gostapi.UpdateMerchantForwards(serverHost, port, newServerIP)
               ├── 删除旧服务
               └── 创建新服务（转发到新 IP）
```

**关键函数调用链**：
```go
ChangeMerchantIP()
  └── onMerchantServerIPChanged(merchantId, port, newServerIP)
      └── updateGostServicesOnSystemServers(merchantId, port, newServerIP, 0)
          └── gostapi.UpdateMerchantForwards(serverHost, port, newServerIP)
              ├── DeleteMerchantForwards() - 删除旧配置
              └── CreateMerchantForwards() - 创建新配置
```

---

### 3.5 修改商户 GOST 端口

**触发时机**：调用 `ChangeMerchantGostPort` 修改商户服务器的 GOST 监听端口

**代码位置**：`server/internal/server/service/merchant/change_ip.go`

**执行流程**：

```
1. 验证新端口范围
2. AWS 安全组开放新端口
3. 更新商户服务器上的 GOST 本地转发服务（通过 API）
   └── gostapi.UpdateMerchantLocalForwardsWithCustomPorts(merchantIP, newPort, appBasePort)
       ├── 删除旧的 local-tcp-10000, local-ws-10001, local-http-10002
       └── 创建新的 local-tcp-{newPort}, local-ws-{newPort+1}, local-http-{newPort+2}
           └── 转发目标仍为 127.0.0.1:10010/10011/10012
4. 更新所有系统服务器上的 GOST 转发目标
   └── updateGostServicesOnSystemServers(merchantId, m.Port, m.ServerIP, newPort)
       └── gostapi.UpdateMerchantForwardsWithTargetPort(serverHost, m.Port, m.ServerIP, newPort)
           └── 系统服务器转发目标改为 商户IP:newPort/+1/+2
```

**关键函数调用链**：
```go
ChangeMerchantGostPort(merchantId, newPort)
  ├── gostapi.UpdateMerchantLocalForwardsWithCustomPorts(merchantIP, newPort, MerchantAppPortTCP)
  │   └── createMerchantLocalForward(merchantIP, gostPort, appPort, protocolName)
  │       └── CreateService() with Forwarder
  └── updateGostServicesOnSystemServers(merchantId, m.Port, m.ServerIP, newPort)
      └── gostapi.UpdateMerchantForwardsWithTargetPort(serverHost, m.Port, m.ServerIP, newPort)
```

**端口变化示例**（假设 newPort = 20000）：

| 位置 | 修改前 | 修改后 |
|------|--------|--------|
| 商户服务器 GOST 监听 | 10000/10001/10002 | 20000/20001/20002 |
| 商户服务器转发目标 | 127.0.0.1:10010/11/12 | 127.0.0.1:10010/11/12（不变）|
| 系统服务器监听 | m.Port/+1/+2（不变）| m.Port/+1/+2（不变）|
| 系统服务器转发目标 | 商户IP:10000/+1/+2 | 商户IP:20000/+1/+2 |

---

### 3.6 隧道连通性检测

**触发时机**：调用 `TunnelCheck` 检测系统服务器到商户服务器的隧道连通性

**代码位置**：`server/internal/server/service/merchant/tunnel.go`

**执行流程**：

```
1. 获取商户服务器 IP
2. 使用固定端口 10000（gostapi.TargetPortTCP）
3. 在每个系统服务器上通过 SSH 执行 TCP 探测
   └── nc -z -w 5 商户IP 10000
4. 返回各系统服务器的检测结果
```

---

## 4. API 函数清单

### 4.1 系统服务器转发（gostapi 包）

| 函数 | 说明 |
|------|------|
| `CreateMerchantForwards(serverIP, basePort, targetIP)` | 批量创建商户的 3 个转发服务 |
| `DeleteMerchantForwards(serverIP, basePort)` | 批量删除商户的 3 个转发服务 |
| `UpdateMerchantForwards(serverIP, basePort, targetIP)` | 批量更新商户的 3 个转发服务（使用默认目标端口）|
| `UpdateMerchantForwardsWithTargetPort(serverIP, basePort, targetIP, targetBasePort)` | 批量更新商户的 3 个转发服务（自定义目标端口）|

### 4.2 商户服务器本地转发（gostapi 包）

| 函数 | 说明 |
|------|------|
| `CreateMerchantLocalForwards(merchantIP)` | 创建商户服务器本地转发（使用默认端口）|
| `DeleteMerchantLocalForwards(merchantIP)` | 删除商户服务器本地转发 |
| `UpdateMerchantLocalForwards(merchantIP)` | 更新商户服务器本地转发（使用默认端口）|
| `UpdateMerchantLocalForwardsWithCustomPorts(merchantIP, gostBasePort, appBasePort)` | 更新商户服务器本地转发（自定义端口）|

### 4.3 底层 API（gostapi 包）

| 函数 | 说明 |
|------|------|
| `CreateRelayTLSForward(serverIP, listenPort, targetIP, targetPort)` | 创建单个 relay+tls 转发 |
| `DeleteRelayTLSForward(serverIP, listenPort)` | 删除单个 relay+tls 转发 |
| `CreateService(serverIP, config)` | 创建 GOST 服务 |
| `DeleteService(serverIP, name)` | 删除 GOST 服务 |
| `CreateChain(serverIP, config)` | 创建 GOST 转发链 |
| `DeleteChain(serverIP, name)` | 删除 GOST 转发链 |
| `SaveConfig(serverIP, format, file)` | 保存配置到文件 |
| `GetServiceList(serverIP)` | 获取服务列表 |
| `GetChainList(serverIP)` | 获取转发链列表 |

### 4.4 任务队列 API（gostapi 包）

所有 GOST 服务创建/删除/更新操作都通过 Redis 任务队列执行，支持失败自动重试（最多 5 次）。

| 函数 | 说明 |
|------|------|
| `EnqueueCreateMerchantLocalForwards(merchantIP)` | 入队：创建商户本地转发 |
| `EnqueueDeleteMerchantLocalForwards(merchantIP)` | 入队：删除商户本地转发 |
| `EnqueueCreateMerchantForwards(serverIP, basePort, targetIP)` | 入队：创建系统服务器转发 |
| `EnqueueDeleteMerchantForwards(serverIP, basePort)` | 入队：删除系统服务器转发 |
| `EnqueueUpdateMerchantForwards(serverIP, basePort, targetIP)` | 入队：更新系统服务器转发 |
| `EnqueueUpdateMerchantForwardsWithTargetPort(...)` | 入队：更新系统服务器转发（自定义目标端口）|

**任务队列配置**：
- Redis Key: `gost:task:queue`（任务队列）
- Redis Key: `gost:task:version`（版本号 Hash）
- 最大重试次数: 5
- 重试间隔: 10 秒

**版本控制机制**：

每个目标（`{serverIP}:{port}` 或 `{serverIP}:local`）维护一个版本号。新任务入队时版本号递增，任务执行时检查版本号：
- 如果任务版本 < 当前版本，说明有更新的任务，跳过执行
- 这样可以确保删除/更新操作不会被旧的创建任务覆盖

```
场景：创建商户后立即删除
1. 创建任务入队，version=1
2. 删除任务入队，version=2
3. 创建任务执行时检查 version=1 < 当前版本=2，跳过
4. 删除任务执行，完成
```

---

## 5. 常量定义

**文件位置**：`server/pkg/gostapi/client.go`

```go
const (
    // GOST API 配置
    GostAPIUsername = "mida"
    GostAPIPassword = "midaAOIhdw92"
    GostAPIPort     = "9394"

    // 端口偏移
    PortOffsetTCP  = 0
    PortOffsetWS   = 1
    PortOffsetHTTP = 2

    // 商户服务器 GOST 监听端口
    MerchantGostPortTCP  = 10000
    MerchantGostPortWS   = 10001
    MerchantGostPortHTTP = 10002

    // 商户业务程序端口
    MerchantAppPortTCP  = 10010
    MerchantAppPortWS   = 10011
    MerchantAppPortHTTP = 10012
)
```

---

## 6. 验证方法

### 6.1 查看系统服务器上的 GOST 服务

```bash
# 查看所有服务
curl -u mida:midaAOIhdw92 http://{系统服务器IP}:9394/config/services

# 查看所有转发链
curl -u mida:midaAOIhdw92 http://{系统服务器IP}:9394/config/chains
```

### 6.2 查看商户服务器上的 GOST 服务

```bash
# 查看所有服务
curl -u mida:midaAOIhdw92 http://{商户服务器IP}:9394/config/services
```

### 6.3 预期结果示例

**系统服务器**（假设商户端口为 20000，商户 IP 为 1.2.3.4）：
```json
{
  "services": [
    {"name": "tcp-relay-20000", "addr": ":20000"},
    {"name": "ws-relay-20001", "addr": ":20001"},
    {"name": "http-relay-20002", "addr": ":20002"}
  ],
  "chains": [
    {"name": "chain-tcp-relay-20000", "hops": [{"nodes": [{"addr": "1.2.3.4:10010"}]}]},
    {"name": "chain-ws-relay-20001", "hops": [{"nodes": [{"addr": "1.2.3.4:10011"}]}]},
    {"name": "chain-http-relay-20002", "hops": [{"nodes": [{"addr": "1.2.3.4:10012"}]}]}
  ]
}
```

**商户服务器**：
```json
{
  "services": [
    {"name": "local-tcp-10010", "addr": ":10010", "forwarder": {"nodes": [{"addr": "127.0.0.1:10000"}]}},
    {"name": "local-ws-10011", "addr": ":10011", "forwarder": {"nodes": [{"addr": "127.0.0.1:10001"}]}},
    {"name": "local-http-10012", "addr": ":10012", "forwarder": {"nodes": [{"addr": "127.0.0.1:10002"}]}}
  ]
}
```

---

## 7. 故障排查

### 7.1 常见问题

| 问题 | 可能原因 | 解决方法 |
|------|---------|---------|
| 隧道检测失败 | 商户服务器 GOST 未启动 | 检查商户服务器 GOST 服务状态 |
| 隧道检测失败 | 安全组未开放端口 | 检查 AWS/阿里云安全组配置 |
| 服务创建失败 | GOST API 不可达 | 检查 GOST 服务是否正常运行 |
| 转发不通 | Chain 配置错误 | 检查转发链的目标地址是否正确 |

### 7.2 日志关键字

```bash
# 搜索 GOST 相关日志
grep -E "gost|merchant forwards|relay" server.log
```

---

## 8. 文件索引

| 文件路径 | 说明 |
|---------|------|
| `server/pkg/gostapi/client.go` | GOST API 客户端和批量操作函数 |
| `server/pkg/gostapi/types.go` | GOST 配置结构体定义 |
| `server/pkg/gostapi/task_queue.go` | GOST 任务队列（Redis 持久化，自动重试）|
| `server/internal/server/service/merchant/merchant.go` | 商户 CRUD 和 GOST 服务创建/删除 |
| `server/internal/server/service/merchant/gost_port.go` | 系统服务器 GOST 服务更新 |
| `server/internal/server/service/merchant/change_ip.go` | 商户 IP/端口变更 |
| `server/internal/server/service/merchant/tunnel.go` | 隧道连通性检测 |
| `server/internal/server/service/deploy/servers.go` | 系统服务器创建和 GOST 初始化 |
| `server/internal/server/service/deploy/gost_api_proxy.go` | GOST API 代理函数 |
