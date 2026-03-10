package gostapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	// GostAPIUsername GOST API 认证用户名
	GostAPIUsername = "tsdd"
	// GostAPIPassword GOST API 认证密码
	GostAPIPassword = "Oa21isSdaiuwhq"
	// GostAPIPort GOST API 端口
	GostAPIPort = "9394"

	// 商户端口偏移
	PortOffsetTCP   = 0
	PortOffsetWS    = 1
	PortOffsetHTTP  = 2
	PortOffsetMinIO = 3

	// 商户服务器 GOST 监听端口（接收系统服务器转发）
	MerchantGostPortTCP   = 10010
	MerchantGostPortWS    = 10011
	MerchantGostPortHTTP  = 10012
	MerchantGostPortMinIO = 10013

	// 商户服务器业务程序端口（GOST relay 转发到本地业务）
	MerchantAppPortTCP   = 5002 // WuKongIM TCP
	MerchantAppPortWS    = 5200 // WuKongIM WebSocket
	MerchantAppPortHTTP  = 5003 // tsdd-server HTTP API
	MerchantAppPortMinIO = 9000 // MinIO S3 API

	// 商户服务器 PC 直连端口（普通 TCP 转发，非 relay）
	MerchantDirectPortTCP  = 10000 // PC 直连 TCP → 5002
	MerchantDirectPortHTTP = 10002 // PC 直连 HTTP → 5003

	// 商户服务器 WSS 代理端口
	MerchantWSSProxyPort    = 10014 // relay+tls → WuKongIM WS
	MerchantWSSProxyAppPort = 5200  // 直连 WuKongIM WebSocket 端口

	// 系统服务器 WSS 监听端口（手机 App WSS 连接）
	SystemWSSListenPort = 443 // :443 (TLS) → relay+tls → 商户:10014

	// 兼容旧代码的别名
	TargetPortTCP  = MerchantGostPortTCP
	TargetPortWS   = MerchantGostPortWS
	TargetPortHTTP = MerchantGostPortHTTP
)

// ========== 简化转发配置 ==========

// ForwardPorts 固定转发端口列表（监听端口=目标端口，对称转发）
// 默认使用商户 GOST 监听端口: TCP/WS/HTTP
var ForwardPorts = []int{10010, 10011, 10012}

// MerchantPortConfig 商户端口配置（用于系统服务器转发）
type MerchantPortConfig struct {
	Offset     int
	TargetPort int
	Name       string
}

// MerchantPortConfigs 商户三端口配置列表（系统服务器 → 商户服务器）
var MerchantPortConfigs = []MerchantPortConfig{
	{PortOffsetTCP, MerchantGostPortTCP, "tcp"},
	{PortOffsetWS, MerchantGostPortWS, "ws"},
	{PortOffsetHTTP, MerchantGostPortHTTP, "http"},
}

// MerchantLocalForwardConfig 商户服务器本地转发配置（GOST → 业务程序）
type MerchantLocalForwardConfig struct {
	GostPort int    // GOST 监听端口
	AppPort  int    // 业务程序端口
	Name     string // 协议名称
}

// MerchantLocalForwardConfigs 商户服务器本地转发配置列表（relay+tls → 本地业务）
var MerchantLocalForwardConfigs = []MerchantLocalForwardConfig{
	{MerchantGostPortTCP, MerchantAppPortTCP, "tcp"},           // 10010 → 5002
	{MerchantGostPortWS, MerchantAppPortWS, "ws"},              // 10011 → 5200
	{MerchantGostPortHTTP, MerchantAppPortHTTP, "http"},         // 10012 → 5003
	{MerchantWSSProxyPort, MerchantWSSProxyAppPort, "wss"},       // 10014 → 5200 (WuKongIM WS)
}

// MerchantPCDirectConfigs PC 直连端口配置（普通 TCP 转发，非 relay）
var MerchantPCDirectConfigs = []MerchantLocalForwardConfig{
	{MerchantDirectPortTCP, MerchantAppPortTCP, "tcp"},   // 10000 → 5002
	{MerchantDirectPortHTTP, MerchantAppPortHTTP, "http"}, // 10002 → 5003
}

// Client GOST API 客户端
type Client struct {
	httpClient *http.Client
}

// 默认客户端实例
var defaultClient = &Client{
	httpClient: &http.Client{
		Timeout: 30 * time.Second,
	},
}

// NewClient 创建新的 GOST API 客户端
// 如果不需要自定义配置，可以直接使用包级别的函数
func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// doRequest 执行 HTTP 请求
func (c *Client) doRequest(method, url string, body interface{}) ([]byte, error) {
	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("序列化请求体失败: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	req.SetBasicAuth(GostAPIUsername, GostAPIPassword)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("执行请求失败: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("请求失败，状态码: %d, 响应: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// buildURL 构建请求 URL
func buildURL(ip, path string) string {
	return fmt.Sprintf("http://%s:%s%s", ip, GostAPIPort, path)
}

// GetConfig 获取当前配置
func (c *Client) GetConfig(ip string, format string) (*Config, error) {
	url := buildURL(ip, "/config")
	if format != "" {
		url += "?format=" + format
	}

	respBody, err := c.doRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := json.Unmarshal(respBody, &config); err != nil {
		return nil, fmt.Errorf("解析配置失败: %w", err)
	}

	return &config, nil
}

// SaveConfig 保存当前配置到文件
func (c *Client) SaveConfig(ip string, format string, path string) (*Response, error) {
	url := buildURL(ip, "/config")
	params := ""
	if format != "" {
		params += "format=" + format
	}
	if path != "" {
		if params != "" {
			params += "&"
		}
		params += "path=" + path
	}
	if params != "" {
		url += "?" + params
	}

	respBody, err := c.doRequest(http.MethodPost, url, nil)
	if err != nil {
		return nil, err
	}

	var resp Response
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	return &resp, nil
}

// GetServiceList 获取服务列表
// GOST v3 不支持 GET /config/services，改为从 /config 获取并提取 services
func (c *Client) GetServiceList(ip string) (*ServiceList, error) {
	// 从 /config 获取完整配置
	config, err := c.GetConfig(ip, "")
	if err != nil {
		return nil, err
	}

	// 从配置中提取服务列表
	return &ServiceList{
		Count: len(config.Services),
		List:  config.Services,
	}, nil
}

// GetService 获取服务详情
func (c *Client) GetService(ip string, serviceName string) (*ServiceConfig, error) {
	url := buildURL(ip, fmt.Sprintf("/config/services/%s", serviceName))

	respBody, err := c.doRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	var service ServiceConfig
	if err := json.Unmarshal(respBody, &service); err != nil {
		return nil, fmt.Errorf("解析服务详情失败: %w", err)
	}

	return &service, nil
}

// CreateService 创建新服务
func (c *Client) CreateService(ip string, service *ServiceConfig) (*Response, error) {
	url := buildURL(ip, "/config/services")

	respBody, err := c.doRequest(http.MethodPost, url, service)
	if err != nil {
		return nil, err
	}

	var resp Response
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	return &resp, nil
}

// UpdateService 更新服务
func (c *Client) UpdateService(ip string, serviceName string, service *ServiceConfig) (*Response, error) {
	url := buildURL(ip, fmt.Sprintf("/config/services/%s", serviceName))

	respBody, err := c.doRequest(http.MethodPut, url, service)
	if err != nil {
		return nil, err
	}

	var resp Response
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	return &resp, nil
}

// DeleteService 删除服务
func (c *Client) DeleteService(ip string, serviceName string) (*Response, error) {
	url := buildURL(ip, fmt.Sprintf("/config/services/%s", serviceName))

	respBody, err := c.doRequest(http.MethodDelete, url, nil)
	if err != nil {
		return nil, err
	}

	var resp Response
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	return &resp, nil
}

// GetChainList 获取链列表
// GOST v3 不支持 GET /config/chains，改为从 /config 获取并提取 chains
func (c *Client) GetChainList(ip string) (*ChainList, error) {
	// 从 /config 获取完整配置
	config, err := c.GetConfig(ip, "")
	if err != nil {
		return nil, err
	}

	// 从配置中提取链列表
	return &ChainList{
		Count: len(config.Chains),
		List:  config.Chains,
	}, nil
}

// GetChain 获取链详情
func (c *Client) GetChain(ip string, chainName string) (*ChainConfig, error) {
	url := buildURL(ip, fmt.Sprintf("/config/chains/%s", chainName))

	respBody, err := c.doRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	var chain ChainConfig
	if err := json.Unmarshal(respBody, &chain); err != nil {
		return nil, fmt.Errorf("解析链详情失败: %w", err)
	}

	return &chain, nil
}

// CreateChain 创建新链
func (c *Client) CreateChain(ip string, chain *ChainConfig) (*Response, error) {
	url := buildURL(ip, "/config/chains")

	respBody, err := c.doRequest(http.MethodPost, url, chain)
	if err != nil {
		return nil, err
	}

	var resp Response
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	return &resp, nil
}

// UpdateChain 更新链
func (c *Client) UpdateChain(ip string, chainName string, chain *ChainConfig) (*Response, error) {
	url := buildURL(ip, fmt.Sprintf("/config/chains/%s", chainName))

	respBody, err := c.doRequest(http.MethodPut, url, chain)
	if err != nil {
		return nil, err
	}

	var resp Response
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	return &resp, nil
}

// DeleteChain 删除链
func (c *Client) DeleteChain(ip string, chainName string) (*Response, error) {
	url := buildURL(ip, fmt.Sprintf("/config/chains/%s", chainName))

	respBody, err := c.doRequest(http.MethodDelete, url, nil)
	if err != nil {
		return nil, err
	}

	var resp Response
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	return &resp, nil
}

// ========== 包级别函数，使用默认客户端 ==========

// GetConfig 获取当前配置
func GetConfig(ip string, format string) (*Config, error) {
	return defaultClient.GetConfig(ip, format)
}

// SaveConfig 保存当前配置到文件
func SaveConfig(ip string, format string, path string) (*Response, error) {
	return defaultClient.SaveConfig(ip, format, path)
}

// GetServiceList 获取服务列表
func GetServiceList(ip string) (*ServiceList, error) {
	return defaultClient.GetServiceList(ip)
}

// GetService 获取服务详情
func GetService(ip string, serviceName string) (*ServiceConfig, error) {
	return defaultClient.GetService(ip, serviceName)
}

// CreateService 创建新服务
func CreateService(ip string, service *ServiceConfig) (*Response, error) {
	return defaultClient.CreateService(ip, service)
}

// UpdateService 更新服务
func UpdateService(ip string, serviceName string, service *ServiceConfig) (*Response, error) {
	return defaultClient.UpdateService(ip, serviceName, service)
}

// DeleteService 删除服务
func DeleteService(ip string, serviceName string) (*Response, error) {
	return defaultClient.DeleteService(ip, serviceName)
}

// GetChainList 获取链列表
func GetChainList(ip string) (*ChainList, error) {
	return defaultClient.GetChainList(ip)
}

// GetChain 获取链详情
func GetChain(ip string, chainName string) (*ChainConfig, error) {
	return defaultClient.GetChain(ip, chainName)
}

// CreateChain 创建新链
func CreateChain(ip string, chain *ChainConfig) (*Response, error) {
	return defaultClient.CreateChain(ip, chain)
}

// UpdateChain 更新链
func UpdateChain(ip string, chainName string, chain *ChainConfig) (*Response, error) {
	return defaultClient.UpdateChain(ip, chainName, chain)
}

// DeleteChain 删除链
func DeleteChain(ip string, chainName string) (*Response, error) {
	return defaultClient.DeleteChain(ip, chainName)
}

// ========== 高级封装函数 ==========

// CreateRelayTLSForward 创建 TCP Relay+TLS 转发服务
// 对应命令: gost -L tcp://:listenPort -F "relay+tls://targetIP:targetPort"
//
// 参数:
//   - gostServerIP: GOST 服务器的 IP 地址
//   - listenPort: 本地监听端口
//   - targetIP: 转发目标 IP
//   - targetPort: 转发目标端口
//
// 返回:
//   - serviceName: 创建的服务名称
//   - error: 错误信息
func CreateRelayTLSForward(gostServerIP string, listenPort int, targetIP string, targetPort int) (serviceName string, err error) {
	return defaultClient.CreateRelayTLSForward(gostServerIP, listenPort, targetIP, targetPort)
}

// CreateRelayTLSForward 创建 TCP Relay+TLS 转发服务
func (c *Client) CreateRelayTLSForward(gostServerIP string, listenPort int, targetIP string, targetPort int) (serviceName string, err error) {
	// 生成唯一的名称
	chainName := fmt.Sprintf("chain-relay-tls-%d", listenPort)
	serviceName = fmt.Sprintf("tcp-relay-%d", listenPort)
	targetAddr := fmt.Sprintf("%s:%d", targetIP, targetPort)
	listenAddr := fmt.Sprintf(":%d", listenPort) // 可以再传入监听的网卡地址

	// 1. 创建 Chain
	chain := &ChainConfig{
		Name: chainName,
		Hops: []HopConfig{
			{
				Name: fmt.Sprintf("hop-%d", listenPort),
				Nodes: []NodeConfig{
					{
						Name: fmt.Sprintf("node-%d", listenPort),
						Addr: targetAddr,
						Connector: &ConnectorConfig{
							Type: "relay",
						},
						Dialer: &DialerConfig{
							Type: "tls",
						},
					},
				},
			},
		},
	}

	// 创建 Chain
	_, err = c.CreateChain(gostServerIP, chain)
	if err != nil {
		return "", fmt.Errorf("创建 Chain 失败: %w", err)
	}

	// 2. 创建 Service（必须包含 forwarder 指定目标地址，否则 dst 为 :0）
	service := &ServiceConfig{
		Name: serviceName,
		Addr: listenAddr,
		Handler: &HandlerConfig{
			Type:  "tcp",
			Chain: chainName,
		},
		Listener: &ListenerConfig{
			Type: "tcp",
		},
		Forwarder: &ForwarderConfig{
			Nodes: []ForwardNodeConfig{
				{
					Name: fmt.Sprintf("target-%d", targetPort),
					Addr: targetAddr,
				},
			},
		},
	}

	_, err = c.CreateService(gostServerIP, service)
	if err != nil {
		// 如果创建 Service 失败，尝试清理 Chain
		_, _ = c.DeleteChain(gostServerIP, chainName)
		return "", fmt.Errorf("创建 Service 失败: %w", err)
	}

	// 3. 保存配置到文件（持久化）
	_, err = c.SaveConfig(gostServerIP, "yaml", "")
	if err != nil {
		// 保存配置失败不影响服务运行，只记录错误
		return serviceName, fmt.Errorf("服务创建成功，但保存配置失败: %w", err)
	}

	return serviceName, nil
}

// DeleteRelayTLSForward 删除 TCP Relay+TLS 转发服务
// 会同时删除关联的 Chain 和 Service
//
// 参数:
//   - gostServerIP: GOST 服务器的 IP 地址
//   - listenPort: 监听端口（用于定位服务）
func DeleteRelayTLSForward(gostServerIP string, listenPort int) error {
	return defaultClient.DeleteRelayTLSForward(gostServerIP, listenPort)
}

// DeleteRelayTLSForward 删除 TCP Relay+TLS 转发服务
func (c *Client) DeleteRelayTLSForward(gostServerIP string, listenPort int) error {
	chainName := fmt.Sprintf("chain-relay-tls-%d", listenPort)
	serviceName := fmt.Sprintf("tcp-relay-%d", listenPort)

	// 删除 Service
	_, err := c.DeleteService(gostServerIP, serviceName)
	if err != nil {
		return fmt.Errorf("删除 Service 失败: %w", err)
	}

	// 删除 Chain
	_, err = c.DeleteChain(gostServerIP, chainName)
	if err != nil {
		return fmt.Errorf("删除 Chain 失败: %w", err)
	}

	// 保存配置到文件（持久化）
	_, err = c.SaveConfig(gostServerIP, "yaml", "")
	if err != nil {
		return fmt.Errorf("服务删除成功，但保存配置失败: %w", err)
	}

	return nil
}

// ========== TLS Listener 支持 ==========

// 系统服务器 TLS 证书默认路径
const (
	TlsCertPath = "/etc/gost/certs/server.crt"
	TlsKeyPath  = "/etc/gost/certs/server.key"
)

// ========== 幂等性辅助函数 ==========

// isAlreadyExistsError 检查是否是 "already exists" 错误（创建时已存在）
func isAlreadyExistsError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "already exists") || strings.Contains(errStr, "40002")
}

// isNotFoundError 检查是否是 "not found" 错误（删除时不存在）
func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "not found") || strings.Contains(errStr, "40004")
}

// ========== 多 IP 绑定支持 ==========

// sanitizeIP 将 IP 中的点号替换为横线，用于 GOST 服务命名
func sanitizeIP(ip string) string {
	return strings.ReplaceAll(ip, ".", "-")
}

// buildServiceName 根据是否有 bindIP 生成服务名
// 有 bindIP: "tcp-relay-47-242-71-251-10000"（多商户隔离）
// 无 bindIP: "tcp-relay-10000"（兼容旧格式）
func buildServiceName(protocolName string, suffix string, listenPort int, bindIP string) string {
	if bindIP != "" {
		return fmt.Sprintf("%s-%s-%s-%d", protocolName, suffix, sanitizeIP(bindIP), listenPort)
	}
	return fmt.Sprintf("%s-%s-%d", protocolName, suffix, listenPort)
}

// buildListenAddr 根据是否有 bindIP 生成监听地址
// 有 bindIP: "47.242.71.251:10000"（只监听指定 IP）
// 无 bindIP: ":10000"（监听所有接口）
func buildListenAddr(listenPort int, bindIP string) string {
	if bindIP != "" {
		return fmt.Sprintf("%s:%d", bindIP, listenPort)
	}
	return fmt.Sprintf(":%d", listenPort)
}

// ========== 商户批量操作函数 ==========

// createRelayTLSForwardWithProtocol 创建带协议名的 Relay+TLS 转发服务（内部方法）
// protocolName 用于区分服务名，如 "tcp", "ws", "http"
// tlsListener: 是否在监听端启用 TLS（客户端加密）
// bindIP: 监听绑定 IP（为空则监听所有接口）
func (c *Client) createRelayTLSForwardWithProtocol(gostServerIP string, listenPort int, targetIP string, targetPort int, protocolName string, tlsListener bool, bindIP ...string) (serviceName string, err error) {
	// 提取 bindIP
	ip := ""
	if len(bindIP) > 0 {
		ip = bindIP[0]
	}

	// 生成唯一的名称（根据协议类型和绑定 IP 区分）
	chainName := buildServiceName(protocolName, "relay", listenPort, ip)
	chainName = "chain-" + chainName
	serviceName = buildServiceName(protocolName, "relay", listenPort, ip)
	targetAddr := fmt.Sprintf("%s:%d", targetIP, targetPort)
	listenAddr := buildListenAddr(listenPort, ip)

	// 1. 创建 Chain
	chain := &ChainConfig{
		Name: chainName,
		Hops: []HopConfig{
			{
				Name: fmt.Sprintf("hop-%d", listenPort),
				Nodes: []NodeConfig{
					{
						Name: fmt.Sprintf("node-%d", listenPort),
						Addr: targetAddr,
						Connector: &ConnectorConfig{
							Type: "relay",
						},
						Dialer: &DialerConfig{
							Type: "tls",
						},
					},
				},
			},
		},
	}

	_, err = c.CreateChain(gostServerIP, chain)
	if err != nil && !isAlreadyExistsError(err) {
		return "", fmt.Errorf("创建 Chain 失败: %w", err)
	}

	// 2. 创建 Service
	listener := &ListenerConfig{Type: "tcp"}
	if tlsListener {
		listener = &ListenerConfig{
			Type: "tls",
			TLS: &TLSConfig{
				CertFile: TlsCertPath,
				KeyFile:  TlsKeyPath,
			},
		}
	}

	service := &ServiceConfig{
		Name: serviceName,
		Addr: listenAddr,
		Handler: &HandlerConfig{
			Type:  "tcp",
			Chain: chainName,
		},
		Listener: listener,
		Forwarder: &ForwarderConfig{
			Nodes: []ForwardNodeConfig{
				{
					Name: fmt.Sprintf("target-%d", targetPort),
					Addr: targetAddr,
				},
			},
		},
	}

	_, err = c.CreateService(gostServerIP, service)
	if err != nil && !isAlreadyExistsError(err) {
		_, _ = c.DeleteChain(gostServerIP, chainName)
		return "", fmt.Errorf("创建 Service 失败: %w", err)
	}

	// 3. 保存配置到文件（持久化）
	_, err = c.SaveConfig(gostServerIP, "yaml", "")
	if err != nil {
		return serviceName, fmt.Errorf("服务创建成功，但保存配置失败: %w", err)
	}

	return serviceName, nil
}

// deleteRelayTLSForwardWithProtocol 删除带协议名的 Relay+TLS 转发服务（内部方法）
// 幂等操作：如果服务/链不存在，视为删除成功
// bindIP: 监听绑定 IP（为空则使用旧命名格式）
func (c *Client) deleteRelayTLSForwardWithProtocol(gostServerIP string, listenPort int, protocolName string, bindIP ...string) error {
	ip := ""
	if len(bindIP) > 0 {
		ip = bindIP[0]
	}

	chainName := "chain-" + buildServiceName(protocolName, "relay", listenPort, ip)
	serviceName := buildServiceName(protocolName, "relay", listenPort, ip)

	// 如果使用了新命名（带 IP），同时清理旧命名的服务（迁移兼容）
	if ip != "" {
		oldChainName := fmt.Sprintf("chain-%s-relay-%d", protocolName, listenPort)
		oldServiceName := fmt.Sprintf("%s-relay-%d", protocolName, listenPort)
		_, _ = c.DeleteService(gostServerIP, oldServiceName)
		_, _ = c.DeleteChain(gostServerIP, oldChainName)
	}

	// 删除 Service（不存在视为成功）
	_, err := c.DeleteService(gostServerIP, serviceName)
	if err != nil && !isNotFoundError(err) {
		return fmt.Errorf("删除 Service 失败: %w", err)
	}

	// 删除 Chain（不存在视为成功）
	_, err = c.DeleteChain(gostServerIP, chainName)
	if err != nil && !isNotFoundError(err) {
		return fmt.Errorf("删除 Chain 失败: %w", err)
	}

	// 保存配置到文件（持久化）
	_, err = c.SaveConfig(gostServerIP, "yaml", "")
	if err != nil {
		return fmt.Errorf("服务删除成功，但保存配置失败: %w", err)
	}

	return nil
}

// CreateMerchantForwards 批量创建商户的 3 个转发服务 (TCP/WS/HTTP)
// bindIP: 可选，系统服务器上为此商户分配的 IP（多商户隔离）
func CreateMerchantForwards(gostServerIP string, basePort int, targetIP string, bindIP ...string) error {
	return defaultClient.CreateMerchantForwards(gostServerIP, basePort, targetIP, bindIP...)
}

// CreateMerchantForwards 批量创建商户的 3 个转发服务
func (c *Client) CreateMerchantForwards(gostServerIP string, basePort int, targetIP string, bindIP ...string) error {
	ip := ""
	if len(bindIP) > 0 {
		ip = bindIP[0]
	}

	var createdConfigs []MerchantPortConfig
	for _, cfg := range MerchantPortConfigs {
		listenPort := basePort + cfg.Offset
		_, err := c.createRelayTLSForwardWithProtocol(gostServerIP, listenPort, targetIP, cfg.TargetPort, cfg.Name, false, ip)
		if err != nil {
			for _, created := range createdConfigs {
				_ = c.deleteRelayTLSForwardWithProtocol(gostServerIP, basePort+created.Offset, created.Name, ip)
			}
			return fmt.Errorf("创建 %s 端口(%d)失败: %w", cfg.Name, listenPort, err)
		}
		createdConfigs = append(createdConfigs, cfg)
	}

	return nil
}

// CreateMerchantForwardsWithTls 批量创建商户的 3 个转发服务，监听端启用 TLS
func CreateMerchantForwardsWithTls(gostServerIP string, basePort int, targetIP string, bindIP ...string) error {
	return defaultClient.CreateMerchantForwardsWithTls(gostServerIP, basePort, targetIP, bindIP...)
}

// CreateMerchantForwardsWithTls 批量创建商户的 3 个转发服务，监听端启用 TLS
func (c *Client) CreateMerchantForwardsWithTls(gostServerIP string, basePort int, targetIP string, bindIP ...string) error {
	ip := ""
	if len(bindIP) > 0 {
		ip = bindIP[0]
	}

	var createdConfigs []MerchantPortConfig
	for _, cfg := range MerchantPortConfigs {
		listenPort := basePort + cfg.Offset
		_, err := c.createRelayTLSForwardWithProtocol(gostServerIP, listenPort, targetIP, cfg.TargetPort, cfg.Name, true, ip)
		if err != nil {
			for _, created := range createdConfigs {
				_ = c.deleteRelayTLSForwardWithProtocol(gostServerIP, basePort+created.Offset, created.Name, ip)
			}
			return fmt.Errorf("创建 TLS %s 端口(%d)失败: %w", cfg.Name, listenPort, err)
		}
		createdConfigs = append(createdConfigs, cfg)
	}

	return nil
}

// DeleteMerchantForwards 批量删除商户的 3 个转发服务
func DeleteMerchantForwards(gostServerIP string, basePort int, bindIP ...string) error {
	return defaultClient.DeleteMerchantForwards(gostServerIP, basePort, bindIP...)
}

// DeleteMerchantForwards 批量删除商户的 3 个转发服务
func (c *Client) DeleteMerchantForwards(gostServerIP string, basePort int, bindIP ...string) error {
	ip := ""
	if len(bindIP) > 0 {
		ip = bindIP[0]
	}

	var lastErr error
	for _, cfg := range MerchantPortConfigs {
		listenPort := basePort + cfg.Offset
		if err := c.deleteRelayTLSForwardWithProtocol(gostServerIP, listenPort, cfg.Name, ip); err != nil {
			lastErr = fmt.Errorf("删除 %s 端口(%d)失败: %w", cfg.Name, listenPort, err)
		}
	}

	return lastErr
}

// UpdateMerchantForwards 批量更新商户的 3 个转发服务（删除+创建）
func UpdateMerchantForwards(gostServerIP string, basePort int, targetIP string, bindIP ...string) error {
	return defaultClient.UpdateMerchantForwards(gostServerIP, basePort, targetIP, bindIP...)
}

// UpdateMerchantForwards 批量更新商户的 3 个转发服务
func (c *Client) UpdateMerchantForwards(gostServerIP string, basePort int, targetIP string, bindIP ...string) error {
	_ = c.DeleteMerchantForwards(gostServerIP, basePort, bindIP...)
	return c.CreateMerchantForwards(gostServerIP, basePort, targetIP, bindIP...)
}

// UpdateMerchantForwardsWithTls 批量更新商户的 3 个转发服务，监听端启用 TLS
func UpdateMerchantForwardsWithTls(gostServerIP string, basePort int, targetIP string, bindIP ...string) error {
	return defaultClient.UpdateMerchantForwardsWithTls(gostServerIP, basePort, targetIP, bindIP...)
}

// UpdateMerchantForwardsWithTls 批量更新商户的 3 个转发服务，监听端启用 TLS
func (c *Client) UpdateMerchantForwardsWithTls(gostServerIP string, basePort int, targetIP string, bindIP ...string) error {
	_ = c.DeleteMerchantForwards(gostServerIP, basePort, bindIP...)
	return c.CreateMerchantForwardsWithTls(gostServerIP, basePort, targetIP, bindIP...)
}

// UpdateMerchantForwardsWithTargetPort 批量更新商户的 3 个转发服务，支持自定义目标基础端口
func UpdateMerchantForwardsWithTargetPort(gostServerIP string, basePort int, targetIP string, targetBasePort int, bindIP ...string) error {
	return defaultClient.updateMerchantForwardsWithTargetPort(gostServerIP, basePort, targetIP, targetBasePort, false, bindIP...)
}

// UpdateMerchantForwardsWithTargetPortWithTls 批量更新商户的 3 个转发服务，支持自定义目标端口，监听端启用 TLS
func UpdateMerchantForwardsWithTargetPortWithTls(gostServerIP string, basePort int, targetIP string, targetBasePort int, bindIP ...string) error {
	return defaultClient.updateMerchantForwardsWithTargetPort(gostServerIP, basePort, targetIP, targetBasePort, true, bindIP...)
}

// updateMerchantForwardsWithTargetPort 内部实现，支持 TLS 开关和 bindIP
func (c *Client) updateMerchantForwardsWithTargetPort(gostServerIP string, basePort int, targetIP string, targetBasePort int, tlsListener bool, bindIP ...string) error {
	ip := ""
	if len(bindIP) > 0 {
		ip = bindIP[0]
	}

	_ = c.DeleteMerchantForwards(gostServerIP, basePort, ip)

	var createdConfigs []MerchantPortConfig
	for i, cfg := range MerchantPortConfigs {
		listenPort := basePort + cfg.Offset
		targetPort := targetBasePort + i
		_, err := c.createRelayTLSForwardWithProtocol(gostServerIP, listenPort, targetIP, targetPort, cfg.Name, tlsListener, ip)
		if err != nil {
			for _, created := range createdConfigs {
				_ = c.deleteRelayTLSForwardWithProtocol(gostServerIP, basePort+created.Offset, created.Name, ip)
			}
			return fmt.Errorf("创建 %s 端口(%d)失败: %w", cfg.Name, listenPort, err)
		}
		createdConfigs = append(createdConfigs, cfg)
	}
	return nil
}

// ========== 商户服务器本地转发函数（GOST → 业务程序） ==========

// CreateMerchantLocalForwards 在商户服务器上创建本地转发服务
// 监听 relay+tls 端口，转发到本地业务程序端口
// merchantServerIP: 商户服务器 IP（用于调用其 GOST API）
func CreateMerchantLocalForwards(merchantServerIP string) error {
	return defaultClient.CreateMerchantLocalForwards(merchantServerIP)
}

// CreateMerchantLocalForwards 在商户服务器上创建本地转发服务
func (c *Client) CreateMerchantLocalForwards(merchantServerIP string) error {
	var createdConfigs []MerchantLocalForwardConfig

	for _, cfg := range MerchantLocalForwardConfigs {
		_, err := c.createMerchantLocalForward(merchantServerIP, cfg.GostPort, cfg.AppPort, cfg.Name)
		if err != nil {
			// 回滚已创建的服务
			for _, created := range createdConfigs {
				_ = c.deleteMerchantLocalForward(merchantServerIP, created.GostPort, created.Name)
			}
			return fmt.Errorf("创建商户本地转发 %s 端口(%d→%d)失败: %w", cfg.Name, cfg.GostPort, cfg.AppPort, err)
		}
		createdConfigs = append(createdConfigs, cfg)
	}

	return nil
}

// DeleteMerchantLocalForwards 删除商户服务器上的本地转发服务
func DeleteMerchantLocalForwards(merchantServerIP string) error {
	return defaultClient.DeleteMerchantLocalForwards(merchantServerIP)
}

// DeleteMerchantLocalForwards 删除商户服务器上的本地转发服务
func (c *Client) DeleteMerchantLocalForwards(merchantServerIP string) error {
	var lastErr error

	for _, cfg := range MerchantLocalForwardConfigs {
		if err := c.deleteMerchantLocalForward(merchantServerIP, cfg.GostPort, cfg.Name); err != nil {
			lastErr = fmt.Errorf("删除商户本地转发 %s 端口(%d)失败: %w", cfg.Name, cfg.GostPort, err)
		}
	}

	return lastErr
}

// UpdateMerchantLocalForwards 更新商户服务器上的本地转发服务（删除+创建）
func UpdateMerchantLocalForwards(merchantServerIP string) error {
	return defaultClient.UpdateMerchantLocalForwards(merchantServerIP)
}

// UpdateMerchantLocalForwards 更新商户服务器上的本地转发服务
func (c *Client) UpdateMerchantLocalForwards(merchantServerIP string) error {
	_ = c.DeleteMerchantLocalForwards(merchantServerIP)
	return c.CreateMerchantLocalForwards(merchantServerIP)
}

// createMerchantLocalForward 创建单个商户本地转发服务（内部方法）
// 监听 relay+tls 端口，通过 forwarder 直接 TCP 转发到本地业务端口
func (c *Client) createMerchantLocalForward(merchantServerIP string, listenPort int, appPort int, protocolName string) (serviceName string, err error) {
	serviceName = fmt.Sprintf("local-%s-%d", protocolName, listenPort)
	targetAddr := fmt.Sprintf("127.0.0.1:%d", appPort)
	listenAddr := fmt.Sprintf(":%d", listenPort)

	// 创建 Service（监听 relay+tls，使用 forwarder 直接转发到本地业务端口）
	service := &ServiceConfig{
		Name: serviceName,
		Addr: listenAddr,
		Handler: &HandlerConfig{
			Type: "relay",
		},
		Listener: &ListenerConfig{
			Type: "tls",
		},
		Forwarder: &ForwarderConfig{
			Nodes: []ForwardNodeConfig{
				{
					Name: fmt.Sprintf("target-%d", appPort),
					Addr: targetAddr,
				},
			},
		},
	}

	_, err = c.CreateService(merchantServerIP, service)
	if err != nil && !isAlreadyExistsError(err) {
		return "", fmt.Errorf("创建 Service 失败: %w", err)
	}

	// 保存配置
	_, err = c.SaveConfig(merchantServerIP, "yaml", "")
	if err != nil {
		return serviceName, fmt.Errorf("服务创建成功，但保存配置失败: %w", err)
	}

	return serviceName, nil
}

// deleteMerchantLocalForward 删除单个商户本地转发服务（内部方法）
// 幂等操作：如果服务不存在，视为删除成功
func (c *Client) deleteMerchantLocalForward(merchantServerIP string, listenPort int, protocolName string) error {
	serviceName := fmt.Sprintf("local-%s-%d", protocolName, listenPort)

	// 删除 Service（不存在视为成功）
	_, err := c.DeleteService(merchantServerIP, serviceName)
	if err != nil && !isNotFoundError(err) {
		return fmt.Errorf("删除 Service 失败: %w", err)
	}

	// 保存配置
	_, err = c.SaveConfig(merchantServerIP, "yaml", "")
	if err != nil {
		return fmt.Errorf("服务删除成功，但保存配置失败: %w", err)
	}

	return nil
}

// UpdateMerchantLocalForwardsWithCustomPorts 更新商户服务器上的本地转发服务，支持自定义端口
// gostBasePort: GOST 监听基础端口
// appBasePort: 业务程序基础端口
func UpdateMerchantLocalForwardsWithCustomPorts(merchantServerIP string, gostBasePort int, appBasePort int) error {
	return defaultClient.UpdateMerchantLocalForwardsWithCustomPorts(merchantServerIP, gostBasePort, appBasePort)
}

// UpdateMerchantLocalForwardsWithCustomPorts 更新商户服务器上的本地转发服务，支持自定义端口
func (c *Client) UpdateMerchantLocalForwardsWithCustomPorts(merchantServerIP string, gostBasePort int, appBasePort int) error {
	// 先删除旧的（使用默认端口）
	_ = c.DeleteMerchantLocalForwards(merchantServerIP)

	// 创建新的（使用自定义端口）
	var createdPorts []int
	for i, cfg := range MerchantLocalForwardConfigs {
		gostPort := gostBasePort + i
		appPort := appBasePort + i
		_, err := c.createMerchantLocalForward(merchantServerIP, gostPort, appPort, cfg.Name)
		if err != nil {
			// 回滚
			for j, created := range createdPorts {
				_ = c.deleteMerchantLocalForward(merchantServerIP, created, MerchantLocalForwardConfigs[j].Name)
			}
			return fmt.Errorf("创建商户本地转发 %s 端口(%d→%d)失败: %w", cfg.Name, gostPort, appPort, err)
		}
		createdPorts = append(createdPorts, gostPort)
	}

	return nil
}

// ========== PC 直连端口转发（商户服务器本地，普通 TCP） ==========

// CreateMerchantPCDirectForwards 在商户服务器上创建 PC 直连端口（普通 TCP 转发）
// 10000 → 5002 (WuKongIM TCP), 10002 → 5003 (tsdd-server HTTP)
func CreateMerchantPCDirectForwards(merchantServerIP string) error {
	return defaultClient.CreateMerchantPCDirectForwards(merchantServerIP)
}

// CreateMerchantPCDirectForwards 在商户服务器上创建 PC 直连端口
func (c *Client) CreateMerchantPCDirectForwards(merchantServerIP string) error {
	for _, cfg := range MerchantPCDirectConfigs {
		serviceName := fmt.Sprintf("direct-%s-%d", cfg.Name, cfg.GostPort)
		targetAddr := fmt.Sprintf("127.0.0.1:%d", cfg.AppPort)
		listenAddr := fmt.Sprintf(":%d", cfg.GostPort)

		service := &ServiceConfig{
			Name: serviceName,
			Addr: listenAddr,
			Handler: &HandlerConfig{
				Type: "tcp",
			},
			Listener: &ListenerConfig{
				Type: "tcp",
			},
			Forwarder: &ForwarderConfig{
				Nodes: []ForwardNodeConfig{
					{
						Name: fmt.Sprintf("target-%d", cfg.AppPort),
						Addr: targetAddr,
					},
				},
			},
		}

		_, err := c.CreateService(merchantServerIP, service)
		if err != nil && !isAlreadyExistsError(err) {
			return fmt.Errorf("创建 PC 直连 %s 端口(%d→%d)失败: %w", cfg.Name, cfg.GostPort, cfg.AppPort, err)
		}
	}

	// 保存配置
	_, err := c.SaveConfig(merchantServerIP, "yaml", "")
	if err != nil {
		return fmt.Errorf("PC 直连服务创建成功，但保存配置失败: %w", err)
	}

	return nil
}

// ========== 系统服务器 → 商户的直连转发函数（TCP 直转，不加密） ==========

// MerchantDirectPortConfigs 商户直连端口配置列表（系统服务器 → 商户业务程序）
// 直连模式跳过商户 GOST 层，直接转发到业务程序端口 5002/5200/5003
var MerchantDirectPortConfigs = []MerchantPortConfig{
	{PortOffsetTCP, MerchantAppPortTCP, "tcp"},
	{PortOffsetWS, MerchantAppPortWS, "ws"},
	{PortOffsetHTTP, MerchantAppPortHTTP, "http"},
}

// createDirectForwardWithProtocol 创建 TCP 直连转发服务（内部方法）
// 不使用 relay+tls，直接 TCP 转发
// tlsListener: 是否在监听端启用 TLS（客户端加密）
// bindIP: 监听绑定 IP（为空则监听所有接口）
func (c *Client) createDirectForwardWithProtocol(gostServerIP string, listenPort int, targetIP string, targetPort int, protocolName string, tlsListener bool, bindIP ...string) (serviceName string, err error) {
	ip := ""
	if len(bindIP) > 0 {
		ip = bindIP[0]
	}
	serviceName = buildServiceName(protocolName, "direct", listenPort, ip)
	targetAddr := fmt.Sprintf("%s:%d", targetIP, targetPort)
	listenAddr := buildListenAddr(listenPort, ip)

	// 构建 listener
	listener := &ListenerConfig{Type: "tcp"}
	if tlsListener {
		listener = &ListenerConfig{
			Type: "tls",
			TLS: &TLSConfig{
				CertFile: TlsCertPath,
				KeyFile:  TlsKeyPath,
			},
		}
	}

	// 创建 Service（直接 TCP 转发，使用 forwarder）
	service := &ServiceConfig{
		Name: serviceName,
		Addr: listenAddr,
		Handler: &HandlerConfig{
			Type: "tcp",
		},
		Listener: listener,
		Forwarder: &ForwarderConfig{
			Nodes: []ForwardNodeConfig{
				{
					Name: fmt.Sprintf("target-%d", targetPort),
					Addr: targetAddr,
				},
			},
		},
	}

	_, err = c.CreateService(gostServerIP, service)
	if err != nil && !isAlreadyExistsError(err) {
		return "", fmt.Errorf("创建 Service 失败: %w", err)
	}

	// 保存配置到文件（持久化）
	_, err = c.SaveConfig(gostServerIP, "yaml", "")
	if err != nil {
		return serviceName, fmt.Errorf("服务创建成功，但保存配置失败: %w", err)
	}

	return serviceName, nil
}

// deleteDirectForwardWithProtocol 删除 TCP 直连转发服务（内部方法）
// 幂等操作：如果服务不存在，视为删除成功
func (c *Client) deleteDirectForwardWithProtocol(gostServerIP string, listenPort int, protocolName string, bindIP ...string) error {
	ip := ""
	if len(bindIP) > 0 {
		ip = bindIP[0]
	}
	serviceName := buildServiceName(protocolName, "direct", listenPort, ip)

	// 迁移兼容：清理旧命名
	if ip != "" {
		oldServiceName := fmt.Sprintf("%s-direct-%d", protocolName, listenPort)
		_, _ = c.DeleteService(gostServerIP, oldServiceName)
	}

	// 删除 Service（不存在视为成功）
	_, err := c.DeleteService(gostServerIP, serviceName)
	if err != nil && !isNotFoundError(err) {
		return fmt.Errorf("删除 Service 失败: %w", err)
	}

	// 保存配置到文件（持久化）
	_, err = c.SaveConfig(gostServerIP, "yaml", "")
	if err != nil {
		return fmt.Errorf("服务删除成功，但保存配置失败: %w", err)
	}

	return nil
}

// CreateMerchantDirectForwards 批量创建商户的直连转发服务
func CreateMerchantDirectForwards(gostServerIP string, basePort int, targetIP string, bindIP ...string) error {
	return defaultClient.CreateMerchantDirectForwards(gostServerIP, basePort, targetIP, bindIP...)
}

func (c *Client) CreateMerchantDirectForwards(gostServerIP string, basePort int, targetIP string, bindIP ...string) error {
	ip := ""
	if len(bindIP) > 0 {
		ip = bindIP[0]
	}

	var createdConfigs []MerchantPortConfig
	for _, cfg := range MerchantDirectPortConfigs {
		listenPort := basePort + cfg.Offset
		_, err := c.createDirectForwardWithProtocol(gostServerIP, listenPort, targetIP, cfg.TargetPort, cfg.Name, false, ip)
		if err != nil {
			for _, created := range createdConfigs {
				_ = c.deleteDirectForwardWithProtocol(gostServerIP, basePort+created.Offset, created.Name, ip)
			}
			return fmt.Errorf("创建直连 %s 端口(%d)失败: %w", cfg.Name, listenPort, err)
		}
		createdConfigs = append(createdConfigs, cfg)
	}
	return nil
}

// CreateMerchantDirectForwardsWithTls 批量创建商户的直连转发服务，监听端启用 TLS
func CreateMerchantDirectForwardsWithTls(gostServerIP string, basePort int, targetIP string, bindIP ...string) error {
	return defaultClient.CreateMerchantDirectForwardsWithTls(gostServerIP, basePort, targetIP, bindIP...)
}

func (c *Client) CreateMerchantDirectForwardsWithTls(gostServerIP string, basePort int, targetIP string, bindIP ...string) error {
	ip := ""
	if len(bindIP) > 0 {
		ip = bindIP[0]
	}

	var createdConfigs []MerchantPortConfig
	for _, cfg := range MerchantDirectPortConfigs {
		listenPort := basePort + cfg.Offset
		_, err := c.createDirectForwardWithProtocol(gostServerIP, listenPort, targetIP, cfg.TargetPort, cfg.Name, true, ip)
		if err != nil {
			for _, created := range createdConfigs {
				_ = c.deleteDirectForwardWithProtocol(gostServerIP, basePort+created.Offset, created.Name, ip)
			}
			return fmt.Errorf("创建 TLS 直连 %s 端口(%d)失败: %w", cfg.Name, listenPort, err)
		}
		createdConfigs = append(createdConfigs, cfg)
	}
	return nil
}

// DeleteMerchantDirectForwards 批量删除商户的直连转发服务
func DeleteMerchantDirectForwards(gostServerIP string, basePort int, bindIP ...string) error {
	return defaultClient.DeleteMerchantDirectForwards(gostServerIP, basePort, bindIP...)
}

func (c *Client) DeleteMerchantDirectForwards(gostServerIP string, basePort int, bindIP ...string) error {
	ip := ""
	if len(bindIP) > 0 {
		ip = bindIP[0]
	}

	var lastErr error
	for _, cfg := range MerchantDirectPortConfigs {
		listenPort := basePort + cfg.Offset
		if err := c.deleteDirectForwardWithProtocol(gostServerIP, listenPort, cfg.Name, ip); err != nil {
			lastErr = fmt.Errorf("删除直连 %s 端口(%d)失败: %w", cfg.Name, listenPort, err)
		}
	}
	return lastErr
}

// UpdateMerchantDirectForwards 批量更新商户的直连转发服务
func UpdateMerchantDirectForwards(gostServerIP string, basePort int, targetIP string, bindIP ...string) error {
	return defaultClient.UpdateMerchantDirectForwards(gostServerIP, basePort, targetIP, bindIP...)
}

func (c *Client) UpdateMerchantDirectForwards(gostServerIP string, basePort int, targetIP string, bindIP ...string) error {
	_ = c.DeleteMerchantDirectForwards(gostServerIP, basePort, bindIP...)
	return c.CreateMerchantDirectForwards(gostServerIP, basePort, targetIP, bindIP...)
}

// ========== 系统服务器 WSS 443 隧道（手机 App WSS 连接） ==========

// CreateWSSRelayForward 在系统服务器上创建 WSS 443 隧道
// bindIP:443 (TLS) → relay+tls → merchantIP:10014 → WuKongIM:5200
func CreateWSSRelayForward(gostServerIP string, merchantIP string, bindIP ...string) error {
	ip := ""
	if len(bindIP) > 0 {
		ip = bindIP[0]
	}
	_, err := defaultClient.createRelayTLSForwardWithProtocol(
		gostServerIP, SystemWSSListenPort, merchantIP, MerchantWSSProxyPort, "wss", true, ip)
	return err
}

// DeleteWSSRelayForward 删除系统服务器上的 WSS 443 隧道
func DeleteWSSRelayForward(gostServerIP string, bindIP ...string) error {
	ip := ""
	if len(bindIP) > 0 {
		ip = bindIP[0]
	}
	return defaultClient.deleteRelayTLSForwardWithProtocol(gostServerIP, SystemWSSListenPort, "wss", ip)
}

// UpdateWSSRelayForward 更新系统服务器上的 WSS 443 隧道（商户 IP 变更时）
func UpdateWSSRelayForward(gostServerIP string, merchantIP string, bindIP ...string) error {
	_ = DeleteWSSRelayForward(gostServerIP, bindIP...)
	return CreateWSSRelayForward(gostServerIP, merchantIP, bindIP...)
}

// ========== 一键部署：简化转发配置 ==========

// SetupForwardTarget 一键配置 GOST 转发到目标服务器（使用 relay+tls 加密）
// 监听 ForwardPorts 中的端口，通过 relay+tls 转发到 targetIP 的相同端口（对称转发）
//
// 流量路径：客户端 → 系统GOST:10010 → [relay+tls加密] → 商户GOST:10010
//
// 示例：
//
//	SetupForwardTarget("1.2.3.4", "5.6.7.8")
//	结果：1.2.3.4:10010 → relay+tls → 5.6.7.8:10010
//	      1.2.3.4:10011 → relay+tls → 5.6.7.8:10011
//	      1.2.3.4:10012 → relay+tls → 5.6.7.8:10012
func SetupForwardTarget(gostServerIP, targetIP string) error {
	return defaultClient.SetupForwardTarget(gostServerIP, targetIP)
}

// SetupForwardTarget 一键配置转发（使用 relay+tls 加密）
func (c *Client) SetupForwardTarget(gostServerIP, targetIP string) error {
	var createdPorts []int

	for _, port := range ForwardPorts {
		// 使用 relay+tls 加密转发，而不是普通 TCP
		_, err := c.CreateRelayTLSForward(gostServerIP, port, targetIP, port)
		if err != nil {
			// 回滚已创建的
			for _, p := range createdPorts {
				_ = c.DeleteRelayTLSForward(gostServerIP, p)
			}
			return fmt.Errorf("创建端口 %d 转发失败: %w", port, err)
		}
		createdPorts = append(createdPorts, port)
	}

	return nil
}

// SetupForwardTargetWithPorts 一键配置转发（自定义端口列表，使用 relay+tls 加密）
func SetupForwardTargetWithPorts(gostServerIP, targetIP string, ports []int) error {
	return defaultClient.SetupForwardTargetWithPorts(gostServerIP, targetIP, ports)
}

// SetupForwardTargetWithPorts 一键配置转发（自定义端口列表，使用 relay+tls 加密）
func (c *Client) SetupForwardTargetWithPorts(gostServerIP, targetIP string, ports []int) error {
	var createdPorts []int

	for _, port := range ports {
		// 使用 relay+tls 加密转发
		_, err := c.CreateRelayTLSForward(gostServerIP, port, targetIP, port)
		if err != nil {
			// 回滚已创建的
			for _, p := range createdPorts {
				_ = c.DeleteRelayTLSForward(gostServerIP, p)
			}
			return fmt.Errorf("创建端口 %d 转发失败: %w", port, err)
		}
		createdPorts = append(createdPorts, port)
	}

	return nil
}

// ClearForwardTarget 清除所有转发规则
func ClearForwardTarget(gostServerIP string) error {
	return defaultClient.ClearForwardTarget(gostServerIP)
}

// ClearForwardTarget 清除所有转发规则
func (c *Client) ClearForwardTarget(gostServerIP string) error {
	for _, port := range ForwardPorts {
		_ = c.DeleteRelayTLSForward(gostServerIP, port)
	}
	return nil
}

// ClearForwardTargetWithPorts 清除指定端口的转发规则
func ClearForwardTargetWithPorts(gostServerIP string, ports []int) error {
	return defaultClient.ClearForwardTargetWithPorts(gostServerIP, ports)
}

// ClearForwardTargetWithPorts 清除指定端口的转发规则
func (c *Client) ClearForwardTargetWithPorts(gostServerIP string, ports []int) error {
	for _, port := range ports {
		_ = c.DeleteRelayTLSForward(gostServerIP, port)
	}
	return nil
}

// UpdateForwardTarget 更新转发目标（先清除再创建）
func UpdateForwardTarget(gostServerIP, targetIP string) error {
	return defaultClient.UpdateForwardTarget(gostServerIP, targetIP)
}

// UpdateForwardTarget 更新转发目标
func (c *Client) UpdateForwardTarget(gostServerIP, targetIP string) error {
	_ = c.ClearForwardTarget(gostServerIP)
	return c.SetupForwardTarget(gostServerIP, targetIP)
}

// ========== TCP 直连转发（不加密） ==========

// SetupDirectForwardTarget 一键配置 TCP 直连转发（不加密）
// 适用于内网或不需要加密的场景
func SetupDirectForwardTarget(gostServerIP, targetIP string) error {
	return defaultClient.SetupDirectForwardTarget(gostServerIP, targetIP)
}

// SetupDirectForwardTarget 一键配置 TCP 直连转发
func (c *Client) SetupDirectForwardTarget(gostServerIP, targetIP string) error {
	var createdPorts []int

	for _, port := range ForwardPorts {
		_, err := c.createSimpleForward(gostServerIP, port, targetIP, port)
		if err != nil {
			// 回滚已创建的
			for _, p := range createdPorts {
				_ = c.deleteSimpleForward(gostServerIP, p)
			}
			return fmt.Errorf("创建端口 %d 转发失败: %w", port, err)
		}
		createdPorts = append(createdPorts, port)
	}

	return nil
}

// SetupDirectForwardTargetWithPorts 一键配置 TCP 直连转发（自定义端口列表）
func SetupDirectForwardTargetWithPorts(gostServerIP, targetIP string, ports []int) error {
	return defaultClient.SetupDirectForwardTargetWithPorts(gostServerIP, targetIP, ports)
}

// SetupDirectForwardTargetWithPorts 一键配置 TCP 直连转发（自定义端口列表）
func (c *Client) SetupDirectForwardTargetWithPorts(gostServerIP, targetIP string, ports []int) error {
	var createdPorts []int

	for _, port := range ports {
		_, err := c.createSimpleForward(gostServerIP, port, targetIP, port)
		if err != nil {
			// 回滚已创建的
			for _, p := range createdPorts {
				_ = c.deleteSimpleForward(gostServerIP, p)
			}
			return fmt.Errorf("创建端口 %d 转发失败: %w", port, err)
		}
		createdPorts = append(createdPorts, port)
	}

	return nil
}

// ClearDirectForwardTarget 清除 TCP 直连转发规则
func ClearDirectForwardTarget(gostServerIP string) error {
	return defaultClient.ClearDirectForwardTarget(gostServerIP)
}

// ClearDirectForwardTarget 清除 TCP 直连转发规则
func (c *Client) ClearDirectForwardTarget(gostServerIP string) error {
	for _, port := range ForwardPorts {
		_ = c.deleteSimpleForward(gostServerIP, port)
	}
	return nil
}

// ClearDirectForwardTargetWithPorts 清除指定端口的 TCP 直连转发规则
func ClearDirectForwardTargetWithPorts(gostServerIP string, ports []int) error {
	return defaultClient.ClearDirectForwardTargetWithPorts(gostServerIP, ports)
}

// ClearDirectForwardTargetWithPorts 清除指定端口的 TCP 直连转发规则
func (c *Client) ClearDirectForwardTargetWithPorts(gostServerIP string, ports []int) error {
	for _, port := range ports {
		_ = c.deleteSimpleForward(gostServerIP, port)
	}
	return nil
}

// createSimpleForward 创建简单的 TCP 直连转发（内部方法）
func (c *Client) createSimpleForward(gostServerIP string, listenPort int, targetIP string, targetPort int) (string, error) {
	serviceName := fmt.Sprintf("fwd-%d", listenPort)
	targetAddr := fmt.Sprintf("%s:%d", targetIP, targetPort)
	listenAddr := fmt.Sprintf(":%d", listenPort)

	service := &ServiceConfig{
		Name: serviceName,
		Addr: listenAddr,
		Handler: &HandlerConfig{
			Type: "tcp",
		},
		Listener: &ListenerConfig{
			Type: "tcp",
		},
		Forwarder: &ForwarderConfig{
			Nodes: []ForwardNodeConfig{
				{
					Name: fmt.Sprintf("target-%d", targetPort),
					Addr: targetAddr,
				},
			},
		},
	}

	_, err := c.CreateService(gostServerIP, service)
	if err != nil && !isAlreadyExistsError(err) {
		return "", fmt.Errorf("创建 Service 失败: %w", err)
	}

	// 保存配置
	_, err = c.SaveConfig(gostServerIP, "yaml", "")
	if err != nil {
		return serviceName, fmt.Errorf("服务创建成功，但保存配置失败: %w", err)
	}

	return serviceName, nil
}

// deleteSimpleForward 删除简单转发（内部方法）
func (c *Client) deleteSimpleForward(gostServerIP string, listenPort int) error {
	serviceName := fmt.Sprintf("fwd-%d", listenPort)

	_, err := c.DeleteService(gostServerIP, serviceName)
	if err != nil && !isNotFoundError(err) {
		return fmt.Errorf("删除 Service 失败: %w", err)
	}

	_, err = c.SaveConfig(gostServerIP, "yaml", "")
	if err != nil {
		return fmt.Errorf("删除成功，但保存配置失败: %w", err)
	}

	return nil
}

// GetForwardStatus 获取转发状态
func GetForwardStatus(gostServerIP string) (map[int]string, error) {
	return defaultClient.GetForwardStatus(gostServerIP)
}

// GetForwardStatus 获取转发状态
// 返回：端口 → 目标地址 的映射
func (c *Client) GetForwardStatus(gostServerIP string) (map[int]string, error) {
	result := make(map[int]string)

	config, err := c.GetConfig(gostServerIP, "")
	if err != nil {
		return nil, err
	}

	for _, svc := range config.Services {
		// 匹配 fwd-XXXX 格式的服务名
		if strings.HasPrefix(svc.Name, "fwd-") {
			var port int
			if _, err := fmt.Sscanf(svc.Name, "fwd-%d", &port); err == nil {
				if svc.Forwarder != nil && len(svc.Forwarder.Nodes) > 0 {
					result[port] = svc.Forwarder.Nodes[0].Addr
				}
			}
		}
	}

	return result, nil
}
