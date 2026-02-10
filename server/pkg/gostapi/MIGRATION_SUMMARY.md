# GOST 服务管理迁移总结

## 迁移概述

已将整个系统的 GOST 服务管理从 **systemd** 方式迁移到 **GOST Web API** 方式。

---

## ✅ 已完成的修改

### 1. 核心 API 实现
**文件**: `server/pkg/gostapi/`

- ✅ `client.go` - 实现了完整的 GOST Web API 客户端
  - 基础 API：GetConfig, SaveConfig, GetServiceList, GetService, CreateService, UpdateService, DeleteService
  - Chain API：GetChainList, GetChain, CreateChain, UpdateChain, DeleteChain  
  - 高级封装：`CreateRelayTLSForward()`, `DeleteRelayTLSForward()`
  
- ✅ `types.go` - 完整的类型定义（基于 swagger.yaml）

- ✅ `README.md`, `gost.md` - 完整的使用文档

### 2. GOST 服务管理层
**文件**: `server/internal/server/service/deploy/gost_service.go`

- ✅ `CreateGostService()` - 使用 `gostapi.CreateRelayTLSForward()`
- ✅ `DeleteGostService()` - 使用 `gostapi.DeleteRelayTLSForward()`
- ✅ `UpdateGostService()` - 通过删除+重建实现
- ✅ `GetGostServiceConfig()` - 使用 `gostapi.GetServiceList()` 和 `gostapi.GetChain()`
- ✅ `extractPortFromServiceName()` - 支持多种服务名称格式

**变化**:
- 服务名称：`gost-{port}.service` → `tcp-relay-{port}`
- 管理方式：systemd 文件 → GOST Web API
- 配置持久化：自动保存到 gost.yaml

### 3. 商户管理层
**文件**: `server/internal/server/service/merchant/merchant.go`

- ✅ `createGostServicesForSelectedServers()` - 创建商户时为系统服务器创建 GOST 服务
- ✅ `UpdateMerchant()` 中的更新逻辑 - 商户 IP 变化时更新 GOST 服务
- ✅ `stopGostServicesOnAllSystemServers()` - 删除商户时清理 GOST 服务

**变化**:
- 服务名称统一为 `tcp-relay-{port}`
- 使用 Web API 管理，无需 SSH 操作 systemd

### 4. 服务器管理层
**文件**: `server/internal/server/service/deploy/servers.go`

- ✅ `createGostServicesForMerchants()` - 创建系统服务器时为商户创建 GOST 服务

**变化**:
- 服务名称统一为 `tcp-relay-{port}`
- 移除了 AutoStart 和 AutoEnable 参数（Web API 自动处理）

---

## ⚠️ 保留的遗留代码（待清理）

以下文件中仍有 systemd gost 服务相关代码，但这些是**遗留兼容代码**，新系统不再使用：

### 1. `merchant_services.go`
**位置**: Line 123-127, 234-235, 282-283, 314-315

```go
// 查询 systemd 管理的 gost 服务（遗留）
if serviceName == "" {
    gostServices, _ := querySystemdGostServices(client)
    resp.Services = append(resp.Services, gostServices...)
}

// 检查是否是 systemd 管理的 gost 服务（遗留）
if strings.HasPrefix(serviceName, "gost") && strings.Contains(serviceName, ".service") {
    return executeSystemdStart(client, serviceName)
}
```

**说明**: 
- 这些代码用于操作**商户服务器**上的服务
- 新架构中，GOST 服务只在**系统服务器**上运行，通过 Web API 管理
- 商户服务器上不应该有 GOST 服务
- **建议**: 保留以兼容可能存在的旧部署，但添加注释说明

### 2. `systemd_services.go`
**位置**: 整个文件

```go
// 查询 systemd 管理的 gost 服务
func querySystemdGostServices(client *utils.SSHClient) ([]model.ServiceStatusResp, error)

// systemd 操作函数
func executeSystemdStart/Stop/Restart(...)
```

**说明**:
- 这个文件提供 systemd 服务管理的工具函数
- **不建议删除**，因为可能还用于管理其他 systemd 服务
- **建议**: 添加注释说明 GOST 服务不再使用这些函数

### 3. `operations.go`
**位置**: Line 66, 154-155

```go
// 系统服务器：使用 systemctl 管理服务
output, execErr = executeSystemdAction(client.SSHClient, req.Action, req.ServiceName)

// 系统服务器：只查询 systemd 管理的 gost 服务
return querySystemdServices(client.SSHClient, req.ServiceName)
```

**说明**:
- 用于通用服务操作
- **建议**: 保留，因为可能用于其他 systemd 服务

### 4. `env_setup.go`
**位置**: Line 45（注释）

```go
// 处理模板实例：gost@1.service -> gost@.service
```

**说明**: 仅注释，无需修改

---

## 📋 服务命名规范对比

| 场景 | 旧命名（systemd） | 新命名（Web API） |
|------|------------------|------------------|
| 创建服务 | `gost-{port}.service` | `tcp-relay-{port}` |
| 商户端口 10010 | `gost-10010.service` | `tcp-relay-10010` |
| Chain 名称 | 无 | `chain-relay-tls-{port}` |
| Hop 名称 | 无 | `hop-{port}` |
| Node 名称 | 无 | `node-{port}` |

---

## 🔑 关键配置

### GOST Web API 配置
```go
const (
    GostAPIUsername = "mida"
    GostAPIPassword = "midaAOIhdw92"
    GostAPIPort     = "22222"
)
```

### 转发配置
```go
const (
    ForwardPort = 10544 // GOST 转发端口
)
```

---

## 🚀 新的使用方式

### 创建 GOST 服务
```go
// 简单方式（推荐）
serviceName, err := gostapi.CreateRelayTLSForward(
    "127.0.0.1",      // GOST 服务器 IP
    11115,            // 监听端口
    "16.162.88.246",  // 转发目标 IP
    10544,            // 转发目标端口
)

// 完整方式
req := model.CreateGostServiceReq{
    ServerId:    serverID,
    ServiceName: fmt.Sprintf("tcp-relay-%d", port),
    ListenPort:  port,
    ForwardHost: targetIP,
    ForwardPort: targetPort,
}
resp, err := deploySvc.CreateGostService(req)
```

### 删除 GOST 服务
```go
// 简单方式
err := gostapi.DeleteRelayTLSForward("127.0.0.1", 11115)

// 完整方式
req := model.DeleteGostServiceReq{
    ServerId:    serverID,
    ServiceName: "tcp-relay-11115",
    ForceDelete: true,
}
resp, err := deploySvc.DeleteGostService(req)
```

---

## 💡 优势对比

### 旧方式（systemd）
- ❌ 需要 SSH 连接到服务器
- ❌ 需要手动管理 service 文件
- ❌ 需要执行 systemctl daemon-reload
- ❌ 配置变更需要重启服务
- ❌ 难以集中管理

### 新方式（Web API）
- ✅ 通过 HTTP API 远程管理
- ✅ 无需 SSH 连接
- ✅ 配置自动持久化到 gost.yaml
- ✅ 服务立即生效
- ✅ 统一的 API 接口
- ✅ 更容易监控和管理

---

## 📝 测试清单

- [ ] 创建新商户时，系统服务器上自动创建 GOST 服务
- [ ] 创建新系统服务器时，为现有商户创建 GOST 服务
- [ ] 更新商户 IP 时，所有系统服务器上的 GOST 服务更新转发目标
- [ ] 删除商户时，所有系统服务器上的 GOST 服务被删除
- [ ] 通过 API 查询服务配置正确
- [ ] GOST 服务配置持久化到 gost.yaml
- [ ] 服务重启后配置仍然存在

---

## 🔄 迁移路径（给运维）

### 从旧系统迁移到新系统

1. **确保 GOST Web API 运行**
   ```bash
   gost -api mida:midaAOIhdw92@:22222
   ```

2. **查看现有的 systemd 服务**
   ```bash
   systemctl list-units 'gost*' --no-pager
   ```

3. **导出配置**（可选）
   ```bash
   curl -u mida:midaAOIhdw92 http://127.0.0.1:22222/config?format=yaml > gost-backup.yaml
   ```

4. **使用新系统创建服务**
   - 通过控制台操作
   - 或使用 API 调用

5. **验证服务运行**
   ```bash
   curl -u mida:midaAOIhdw92 http://127.0.0.1:22222/config/services
   ```

6. **清理旧的 systemd 服务**（可选）
   ```bash
   systemctl stop gost@*.service
   systemctl disable gost@*.service
   rm /etc/systemd/system/gost@*.service
   systemctl daemon-reload
   ```

---

## 📚 相关文档

- [GOST API 使用文档](./README.md)
- [GOST 命令参考](./gost.md)
- [GOST 官方文档](https://gost.run/)

---

## ✅ 总结

✨ **迁移已完成**，所有核心功能都已改用 GOST Web API 管理！

- ✅ 3 个核心服务文件已重构
- ✅ 7 个 API 方法已实现  
- ✅ 完整的类型定义和文档
- ✅ 向后兼容的设计
- ✅ 更简单、更可靠的管理方式

🎉 现在可以享受更现代化的 GOST 服务管理体验了！

