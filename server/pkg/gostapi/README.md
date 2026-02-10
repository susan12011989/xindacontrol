# GOST API 客户端

这个包提供了一个简单易用的 Go 客户端，用于与 GOST (GO Simple Tunnel) Web API 进行交互。

## 功能特性

- ✅ 获取 GOST 配置
- ✅ 保存配置到文件
- ✅ 获取服务列表
- ✅ 查看服务详情
- ✅ 创建新服务
- ✅ 更新现有服务
- ✅ 删除服务
- ✅ 自动处理 HTTP Basic Auth
- ✅ 完整的类型定义（基于 GOST Swagger 规范）

## 快速开始

### 导入

```go
import "server/pkg/gostapi"
```

### 基本使用

**推荐方式：直接使用包级别函数**

```go
package main

import (
    "fmt"
    "log"
    
    "server/pkg/gostapi"
)

func main() {
    // GOST 服务器 IP 地址
    ip := "127.0.0.1"
    
    // 直接调用，无需创建客户端
    serviceList, err := gostapi.GetServiceList(ip)
    if err != nil {
        log.Fatalf("获取服务列表失败: %v", err)
    }
    
    fmt.Printf("共有 %d 个服务\n", serviceList.Count)
}
```

**可选方式：使用自定义客户端（如需自定义配置）**

```go
// 如果需要自定义超时等配置
client := gostapi.NewClient()
serviceList, err := client.GetServiceList(ip)
```

## API 参考

所有 API 都可以直接通过包名调用，无需创建客户端实例。

### 1. 获取配置

```go
config, err := gostapi.GetConfig(ip, "json") // format 可以是 "json" 或 "yaml"
if err != nil {
    log.Fatal(err)
}
```

### 2. 保存配置

```go
resp, err := gostapi.SaveConfig(ip, "yaml", "") // format: "yaml" 或 "json", path 可选
if err != nil {
    log.Fatal(err)
}
```

### 3. 获取服务列表

```go
serviceList, err := gostapi.GetServiceList(ip)
if err != nil {
    log.Fatal(err)
}

for _, svc := range serviceList.List {
    fmt.Printf("服务: %s, 地址: %s\n", svc.Name, svc.Addr)
}
```

### 4. 获取服务详情

```go
service, err := gostapi.GetService(ip, "service-0")
if err != nil {
    log.Fatal(err)
}
```

### 5. 创建服务

```go
newService := &gostapi.ServiceConfig{
    Name: "my-http-proxy",
    Addr: ":8080",
    Handler: &gostapi.HandlerConfig{
        Type: "http",
    },
    Listener: &gostapi.ListenerConfig{
        Type: "tcp",
    },
}

resp, err := gostapi.CreateService(ip, newService)
if err != nil {
    log.Fatal(err)
}
```

### 6. 更新服务

```go
updatedService := &gostapi.ServiceConfig{
    Name: "my-http-proxy",
    Addr: ":8080",
    Handler: &gostapi.HandlerConfig{
        Type: "socks5", // 从 http 改为 socks5
    },
    Listener: &gostapi.ListenerConfig{
        Type: "tcp",
    },
}

resp, err := gostapi.UpdateService(ip, "my-http-proxy", updatedService)
if err != nil {
    log.Fatal(err)
}
```

### 7. 删除服务

```go
resp, err := gostapi.DeleteService(ip, "my-http-proxy")
if err != nil {
    log.Fatal(err)
}
```

## 配置

默认配置常量定义在 `client.go` 中：

```go
const (
    GostAPIUsername = "mida"           // HTTP Basic Auth 用户名
    GostAPIPassword = "midaAOIhdw92"   // HTTP Basic Auth 密码
    GostAPIPort     = "22222"          // GOST API 端口
)
```

如需修改这些值，请直接编辑 `client.go` 文件中的常量定义。

## 完整示例

### 创建 SOCKS5 代理服务

```go
package main

import (
    "log"
    
    "server/pkg/gostapi"
)

func main() {
    ip := "192.168.1.100" // GOST 服务器 IP
    
    // 创建 SOCKS5 代理
    service := &gostapi.ServiceConfig{
        Name: "socks5-proxy",
        Addr: ":1080",
        Handler: &gostapi.HandlerConfig{
            Type: "socks5",
            Auth: &gostapi.AuthConfig{
                Username: "user",
                Password: "pass",
            },
        },
        Listener: &gostapi.ListenerConfig{
            Type: "tcp",
        },
    }
    
    resp, err := gostapi.CreateService(ip, service)
    if err != nil {
        log.Fatalf("创建服务失败: %v", err)
    }
    
    log.Printf("服务创建成功: %+v", resp)
}
```

### 批量管理服务

```go
package main

import (
    "fmt"
    "log"
    
    "server/pkg/gostapi"
)

func main() {
    ip := "127.0.0.1"
    
    // 获取所有服务
    serviceList, err := gostapi.GetServiceList(ip)
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("共有 %d 个服务:\n", serviceList.Count)
    
    // 遍历并显示每个服务的详细信息
    for _, svc := range serviceList.List {
        detail, err := gostapi.GetService(ip, svc.Name)
        if err != nil {
            log.Printf("获取服务 %s 详情失败: %v", svc.Name, err)
            continue
        }
        
        fmt.Printf("\n服务名称: %s\n", detail.Name)
        fmt.Printf("监听地址: %s\n", detail.Addr)
        if detail.Handler != nil {
            fmt.Printf("处理器类型: %s\n", detail.Handler.Type)
        }
        if detail.Listener != nil {
            fmt.Printf("监听器类型: %s\n", detail.Listener.Type)
        }
        
        // 可选：显示服务状态
        if detail.Status != nil {
            fmt.Printf("状态: %s\n", detail.Status.State)
            if detail.Status.Stats != nil {
                fmt.Printf("总连接数: %d\n", detail.Status.Stats.TotalConns)
                fmt.Printf("当前连接数: %d\n", detail.Status.Stats.CurrentConns)
            }
        }
    }
}
```

## 注意事项

1. **服务重启**: 更新现有服务会导致该服务重启
2. **即时生效**: 创建新服务会立即生效，不影响其他服务
3. **立即删除**: 删除服务会立即关闭并移除该服务
4. **网络超时**: 默认 HTTP 客户端超时为 30 秒
5. **错误处理**: 所有 API 方法都会返回错误，请务必检查错误

## 类型系统

包中包含完整的类型定义（在 `types.go` 中），基于 GOST 官方 Swagger 规范。主要类型包括：

- `ServiceConfig` - 服务配置
- `HandlerConfig` - 处理器配置
- `ListenerConfig` - 监听器配置
- `ChainConfig` - 链配置
- `HopConfig` - 跃点配置
- `NodeConfig` - 节点配置
- `AuthConfig` - 认证配置
- `TLSConfig` - TLS 配置
- 以及更多...

## 测试

查看 `example_test.go` 文件获取更多使用示例。

## 相关链接

- [GOST 官方文档](https://gost.run/)
- [GOST GitHub](https://github.com/go-gost/gost)

## 许可证

与主项目保持一致。

