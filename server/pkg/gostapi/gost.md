# GOST API 客户端

## GOST 服务启动

启动全局web管理服务：
```bash
gost -api mida:midaAOIhdw92@:22222
```

## API 使用说明

### 1. 查看当前的配置
```bash
curl -u mida:midaAOIhdw92 http://127.0.0.1:22222/config?format=json
```

Go 代码：
```go
config, err := gostapi.GetConfig("127.0.0.1", "json")
```

### 2. 保存当前的配置到gost.json或gost.yaml文件
```bash
curl -u mida:midaAOIhdw92 -X POST http://127.0.0.1:22222/config?format=yaml
```

Go 代码：
```go
resp, err := gostapi.SaveConfig("127.0.0.1", "yaml", "")
```

### 3. 获取服务列表
```bash
curl -u mida:midaAOIhdw92 http://127.0.0.1:22222/config/services
```

Go 代码：
```go
serviceList, err := gostapi.GetServiceList("127.0.0.1")
```

### 4. 查看当前服务详情
```bash
curl -u mida:midaAOIhdw92 http://127.0.0.1:22222/config/services/service-0
```

Go 代码：
```go
service, err := gostapi.GetService("127.0.0.1", "service-0")
```

### 5. 添加一个新的服务
添加一个新的服务不会对现有服务造成影响，如果配置成功则服务立即生效。

```bash
curl -u mida:midaAOIhdw92 http://127.0.0.1:22222/config/services -d \
'{"name":"service-0","addr":":8080","handler":{"type":"http"},"listener":{"type":"tcp"}}'
```

Go 代码：
```go
newService := &gostapi.ServiceConfig{
    Name: "service-0",
    Addr: ":8080",
    Handler: &gostapi.HandlerConfig{
        Type: "http",
    },
    Listener: &gostapi.ListenerConfig{
        Type: "tcp",
    },
}
resp, err := gostapi.CreateService("127.0.0.1", newService)
```

### 6. 修改一个现有的服务
修改一个现有的服务会导致此服务重启。

```bash
curl -u mida:midaAOIhdw92 -X PUT http://127.0.0.1:22222/config/services/service-0 -d \
'{"name":"service-0","addr":":8080","handler":{"type":"socks5"},"listener":{"type":"tcp"}}'
```

Go 代码：
```go
updatedService := &gostapi.ServiceConfig{
    Name: "service-0",
    Addr: ":8080",
    Handler: &gostapi.HandlerConfig{
        Type: "socks5",
    },
    Listener: &gostapi.ListenerConfig{
        Type: "tcp",
    },
}
resp, err := gostapi.UpdateService("127.0.0.1", "service-0", updatedService)
```

### 7. 删除一个现有服务
删除一个现有服务会立即关闭并删除此服务。

```bash
curl -u mida:midaAOIhdw92 -X DELETE http://127.0.0.1:22222/config/services/service-0
```

Go 代码：
```go
resp, err := gostapi.DeleteService("127.0.0.1", "service-0")
```

## 完整示例

```go
package main

import (
    "fmt"
    "log"
    
    "server/pkg/gostapi"
)

func main() {
    // GOST 服务器 IP
    ip := "127.0.0.1"
    
    // 直接调用，无需创建客户端
    serviceList, err := gostapi.GetServiceList(ip)
    if err != nil {
        log.Fatalf("获取服务列表失败: %v", err)
    }
    
    fmt.Printf("共有 %d 个服务\n", serviceList.Count)
    for _, svc := range serviceList.List {
        fmt.Printf("- %s: %s\n", svc.Name, svc.Addr)
    }
}
```