package deploy

import (
	"fmt"
	"server/pkg/dbs"
	"server/pkg/entity"
	"server/pkg/gostapi"
	"strings"
)

// getServerHostById 查询服务器 Host
func getServerHostById(serverId int) (string, error) {
	var server entity.Servers
	has, err := dbs.DBAdmin.ID(serverId).Get(&server)
	if err != nil {
		return "", fmt.Errorf("查询服务器失败: %v", err)
	}
	if !has {
		return "", fmt.Errorf("服务器不存在: %d", serverId)
	}
	return server.Host, nil
}

// ListGostServices 通过 GOST Web API 列出服务（内部分页）
func ListGostServices(serverId int, page int, size int, port int) (*gostapi.ServiceList, error) {
	host, err := getServerHostById(serverId)
	if err != nil {
		return nil, err
	}
	data, err := gostapi.GetServiceList(host)
	if err != nil {
		return nil, err
	}
	// 内部分页：默认每页 20
	if size <= 0 {
		size = 20
	}
	if page <= 0 {
		page = 1
	}
	// 端口过滤（匹配 tcp-relay-{port} 名称）
	if port > 0 {
		name := fmt.Sprintf("tcp-relay-%d", port)
		filtered := make([]gostapi.ServiceConfig, 0, len(data.List))
		for _, s := range data.List {
			if s.Name == name {
				filtered = append(filtered, s)
			}
		}
		data.List = filtered
	}
	total := len(data.List)
	start := (page - 1) * size
	if start >= total {
		// 超出范围返回空列表，但保留总数
		return &gostapi.ServiceList{Count: total, List: []gostapi.ServiceConfig{}}, nil
	}
	end := start + size
	if end > total {
		end = total
	}
	// 保留 Count 为总数，List 为分页数据
	data.Count = total
	data.List = data.List[start:end]
	return data, nil
}

// ListGostChains 通过 GOST Web API 列出链
func ListGostChains(serverId int) (*gostapi.ChainList, error) {
	host, err := getServerHostById(serverId)
	if err != nil {
		return nil, err
	}
	return gostapi.GetChainList(host)
}

// GetGostServiceDetail 获取指定 Service 详情
func GetGostServiceDetail(serverId int, name string) (*gostapi.ServiceConfig, error) {
	host, err := getServerHostById(serverId)
	if err != nil {
		return nil, err
	}
	return gostapi.GetService(host, name)
}

// UpdateGostServiceDetail 更新指定 Service 配置
func UpdateGostServiceDetail(serverId int, name string, cfg *gostapi.ServiceConfig) (*gostapi.Response, error) {
	host, err := getServerHostById(serverId)
	if err != nil {
		return nil, err
	}
	return gostapi.UpdateService(host, name, cfg)
}

// GetGostChainDetail 获取指定 Chain 详情
func GetGostChainDetail(serverId int, name string) (*gostapi.ChainConfig, error) {
	host, err := getServerHostById(serverId)
	if err != nil {
		return nil, err
	}
	return gostapi.GetChain(host, name)
}

// UpdateGostChainDetail 更新指定 Chain 配置
func UpdateGostChainDetail(serverId int, name string, cfg *gostapi.ChainConfig) (*gostapi.Response, error) {
	host, err := getServerHostById(serverId)
	if err != nil {
		return nil, err
	}
	return gostapi.UpdateChain(host, name, cfg)
}

// CreateGostServiceAPI 通过 CreateRelayTLSForward 统一创建
func CreateGostServiceAPI(serverId int, listenPort int, forwardHost string, forwardPort int) (string, error) {
	host, err := getServerHostById(serverId)
	if err != nil {
		return "", err
	}
	return gostapi.CreateRelayTLSForward(host, listenPort, forwardHost, forwardPort)
}

// DeleteGostServiceAPI 直接按服务名删除服务及其对应的链
func DeleteGostServiceAPI(serverId int, name string) (*gostapi.Response, error) {
	host, err := getServerHostById(serverId)
	if err != nil {
		return nil, err
	}

	// 直接删除服务
	_, err = gostapi.DeleteService(host, name)
	if err != nil {
		return nil, fmt.Errorf("删除 Service 失败: %w", err)
	}

	// 根据服务名推导对应的链名并删除（如果存在）
	chainName := deriveChainNameFromService(name)
	if chainName != "" {
		// 链不存在不报错
		_, _ = gostapi.DeleteChain(host, chainName)
	}

	// 保存配置
	if _, err := gostapi.SaveConfig(host, "yaml", ""); err != nil {
		return nil, fmt.Errorf("服务删除成功，但保存配置失败: %w", err)
	}

	return &gostapi.Response{Code: 200, Msg: "ok"}, nil
}

// deriveChainNameFromService 根据服务名推导对应的链名
// tcp-relay-{port} → chain-relay-tls-{port} (兼容旧格式)
// ws-relay-{port} → chain-ws-relay-{port}
// http-relay-{port} → chain-http-relay-{port}
// local-* 类服务没有对应的链
func deriveChainNameFromService(serviceName string) string {
	// local-* 类服务没有链
	if strings.HasPrefix(serviceName, "local-") {
		return ""
	}

	// tcp-relay-{port} → chain-relay-tls-{port} (兼容旧命名)
	if strings.HasPrefix(serviceName, "tcp-relay-") {
		port := strings.TrimPrefix(serviceName, "tcp-relay-")
		return "chain-relay-tls-" + port
	}

	// ws-relay-{port} → chain-ws-relay-{port}
	if strings.HasPrefix(serviceName, "ws-relay-") {
		port := strings.TrimPrefix(serviceName, "ws-relay-")
		return "chain-ws-relay-" + port
	}

	// http-relay-{port} → chain-http-relay-{port}
	if strings.HasPrefix(serviceName, "http-relay-") {
		port := strings.TrimPrefix(serviceName, "http-relay-")
		return "chain-http-relay-" + port
	}

	return ""
}

// CreateGostChainAPI 也统一走 CreateRelayTLSForward（创建 chain+service 配套）
func CreateGostChainAPI(serverId int, listenPort int, forwardHost string, forwardPort int) (string, error) {
	host, err := getServerHostById(serverId)
	if err != nil {
		return "", err
	}
	return gostapi.CreateRelayTLSForward(host, listenPort, forwardHost, forwardPort)
}

// DeleteGostChainAPI 直接按链名删除链及其对应的服务
func DeleteGostChainAPI(serverId int, name string) (*gostapi.Response, error) {
	host, err := getServerHostById(serverId)
	if err != nil {
		return nil, err
	}

	// 直接删除链
	_, err = gostapi.DeleteChain(host, name)
	if err != nil {
		return nil, fmt.Errorf("删除 Chain 失败: %w", err)
	}

	// 根据链名推导对应的服务名并删除（如果存在）
	serviceName := deriveServiceNameFromChain(name)
	if serviceName != "" {
		// 服务不存在不报错
		_, _ = gostapi.DeleteService(host, serviceName)
	}

	// 保存配置
	if _, err := gostapi.SaveConfig(host, "yaml", ""); err != nil {
		return nil, fmt.Errorf("链删除成功，但保存配置失败: %w", err)
	}

	return &gostapi.Response{Code: 200, Msg: "ok"}, nil
}

// deriveServiceNameFromChain 根据链名推导对应的服务名
// chain-relay-tls-{port} → tcp-relay-{port}
// chain-ws-relay-{port} → ws-relay-{port}
// chain-http-relay-{port} → http-relay-{port}
func deriveServiceNameFromChain(chainName string) string {
	// chain-relay-tls-{port} → tcp-relay-{port}
	if strings.HasPrefix(chainName, "chain-relay-tls-") {
		port := strings.TrimPrefix(chainName, "chain-relay-tls-")
		return "tcp-relay-" + port
	}

	// chain-tcp-relay-{port} → tcp-relay-{port}
	if strings.HasPrefix(chainName, "chain-tcp-relay-") {
		port := strings.TrimPrefix(chainName, "chain-tcp-relay-")
		return "tcp-relay-" + port
	}

	// chain-ws-relay-{port} → ws-relay-{port}
	if strings.HasPrefix(chainName, "chain-ws-relay-") {
		port := strings.TrimPrefix(chainName, "chain-ws-relay-")
		return "ws-relay-" + port
	}

	// chain-http-relay-{port} → http-relay-{port}
	if strings.HasPrefix(chainName, "chain-http-relay-") {
		port := strings.TrimPrefix(chainName, "chain-http-relay-")
		return "http-relay-" + port
	}

	return ""
}
