package control

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"server/pkg/gostapi"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// --- GOST 管理相关类型 ---

// GostServiceList GOST 服务列表
type GostServiceList = gostapi.ServiceList

// GostServiceConfig GOST 服务配置
type GostServiceConfig = gostapi.ServiceConfig

// GostChainList GOST 链列表
type GostChainList = gostapi.ChainList

// GostConfigSyncStatus GOST 配置同步状态
type GostConfigSyncStatus struct {
	Synced              bool   `json:"synced"`
	RunningServiceCount int    `json:"running_service_count"`
	RunningChainCount   int    `json:"running_chain_count"`
	FileServiceCount    int    `json:"file_service_count"`
	FileChainCount      int    `json:"file_chain_count"`
	Message             string `json:"message"`
}

// IGostController GOST 管理接口
// 同时兼容单机和多机模式
type IGostController interface {
	// ListGostServices 列出 GOST 服务
	ListGostServices(ctx context.Context, page, size, port int) (*GostServiceList, error)

	// GetGostService 获取单个 GOST 服务
	GetGostService(ctx context.Context, name string) (*GostServiceConfig, error)

	// CreateGostService 创建 GOST 服务
	CreateGostService(ctx context.Context, config *GostServiceConfig) error

	// UpdateGostService 更新 GOST 服务
	UpdateGostService(ctx context.Context, name string, config *GostServiceConfig) error

	// DeleteGostService 删除 GOST 服务
	DeleteGostService(ctx context.Context, name string) error

	// ListGostChains 列出 GOST 链
	ListGostChains(ctx context.Context) (*GostChainList, error)

	// PersistGostConfig 持久化 GOST 运行配置到文件
	PersistGostConfig(ctx context.Context) error

	// GetGostConfigSyncStatus 获取配置同步状态
	GetGostConfigSyncStatus(ctx context.Context) (*GostConfigSyncStatus, error)
}

// --- 单机 GOST 控制器 ---

// localGostController 单机模式 GOST 控制器
// GOST 运行在本机，API 调用 127.0.0.1
type localGostController struct {
	executor *LocalExecutor
	host     string // 固定 127.0.0.1
}

func newLocalGostController(executor *LocalExecutor) *localGostController {
	return &localGostController{
		executor: executor,
		host:     "127.0.0.1",
	}
}

func (g *localGostController) ListGostServices(_ context.Context, page, size, port int) (*GostServiceList, error) {
	data, err := gostapi.GetServiceList(g.host)
	if err != nil {
		return nil, err
	}
	return filterAndPaginateServices(data, page, size, port), nil
}

func (g *localGostController) GetGostService(_ context.Context, name string) (*GostServiceConfig, error) {
	return gostapi.GetService(g.host, name)
}

func (g *localGostController) CreateGostService(ctx context.Context, config *GostServiceConfig) error {
	_, err := gostapi.CreateService(g.host, config)
	if err != nil {
		return err
	}
	g.autoPersist(ctx)
	return nil
}

func (g *localGostController) UpdateGostService(ctx context.Context, name string, config *GostServiceConfig) error {
	_, err := gostapi.UpdateService(g.host, name, config)
	if err != nil {
		return err
	}
	g.autoPersist(ctx)
	return nil
}

func (g *localGostController) DeleteGostService(ctx context.Context, name string) error {
	_, err := gostapi.DeleteService(g.host, name)
	if err != nil {
		return err
	}
	g.autoPersist(ctx)
	return nil
}

// autoPersist 自动持久化配置（异步，失败只打日志不影响主流程）
func (g *localGostController) autoPersist(ctx context.Context) {
	go func() {
		if err := g.PersistGostConfig(ctx); err != nil {
			fmt.Printf("[GOST] 自动持久化失败: %v\n", err)
		}
	}()
}

func (g *localGostController) ListGostChains(_ context.Context) (*GostChainList, error) {
	return gostapi.GetChainList(g.host)
}

func (g *localGostController) PersistGostConfig(ctx context.Context) error {
	config, err := gostapi.GetConfig(g.host, "")
	if err != nil {
		return fmt.Errorf("获取 GOST 运行配置失败: %w", err)
	}

	yamlContent, err := gostConfigToYAML(config)
	if err != nil {
		return fmt.Errorf("配置转换 YAML 失败: %w", err)
	}

	// 单机模式：直接本地文件操作
	ts := time.Now().Format("20060102_150405")
	backupPath := fmt.Sprintf("%s.%s.bak", gostConfigFilePath, ts)

	g.executor.Execute(ctx, fmt.Sprintf("sudo cp -f '%s' '%s' 2>/dev/null", gostConfigFilePath, backupPath))
	g.executor.Execute(ctx, "sudo mkdir -p /etc/gost")

	tmpPath := fmt.Sprintf("/tmp/gost-config-%d.yaml", time.Now().UnixNano())
	if err := g.executor.UploadFile(ctx, tmpPath, bytes.NewReader([]byte(yamlContent))); err != nil {
		return fmt.Errorf("写入配置文件失败: %w", err)
	}

	result := g.executor.Execute(ctx, fmt.Sprintf("sudo mv -f '%s' '%s'", tmpPath, gostConfigFilePath))
	if result.Err != nil {
		return fmt.Errorf("移动配置文件失败: %w", result.Err)
	}
	return nil
}

func (g *localGostController) GetGostConfigSyncStatus(ctx context.Context) (*GostConfigSyncStatus, error) {
	runningConfig, err := gostapi.GetConfig(g.host, "")
	if err != nil {
		return nil, fmt.Errorf("获取运行配置失败: %w", err)
	}

	result := g.executor.Execute(ctx, fmt.Sprintf("sudo cat '%s' 2>/dev/null", gostConfigFilePath))
	fileContent := strings.TrimSpace(result.Output)

	if result.Err != nil || fileContent == "" {
		return &GostConfigSyncStatus{
			Synced:              false,
			RunningServiceCount: len(runningConfig.Services),
			RunningChainCount:   len(runningConfig.Chains),
			Message:             "配置文件不存在或为空",
		}, nil
	}

	return buildSyncStatus(runningConfig, fileContent), nil
}

// --- 多机 GOST 控制器 ---

// clusterGostController 多机模式 GOST 控制器
// 通过 serverId 获取目标服务器 IP，SSH 管理配置持久化
type clusterGostController struct {
	cluster *ClusterController
}

func newClusterGostController(cluster *ClusterController) *clusterGostController {
	return &clusterGostController{cluster: cluster}
}

// resolveHost 根据 serverId 获取服务器 Host
func resolveHost(serverId int) (string, error) {
	server, err := getServerEntity(serverId)
	if err != nil {
		return "", err
	}
	return server.Host, nil
}

// ListGostServicesOnServer 在指定服务器上列出 GOST 服务
func (g *clusterGostController) ListGostServicesOnServer(_ context.Context, serverId int, page, size, port int) (*GostServiceList, error) {
	host, err := resolveHost(serverId)
	if err != nil {
		return nil, err
	}
	data, err := gostapi.GetServiceList(host)
	if err != nil {
		return nil, err
	}
	return filterAndPaginateServices(data, page, size, port), nil
}

// GetGostServiceOnServer 获取指定服务器上的 GOST 服务
func (g *clusterGostController) GetGostServiceOnServer(_ context.Context, serverId int, name string) (*GostServiceConfig, error) {
	host, err := resolveHost(serverId)
	if err != nil {
		return nil, err
	}
	return gostapi.GetService(host, name)
}

// CreateGostServiceOnServer 在指定服务器上创建 GOST 服务
func (g *clusterGostController) CreateGostServiceOnServer(ctx context.Context, serverId int, config *GostServiceConfig) error {
	host, err := resolveHost(serverId)
	if err != nil {
		return err
	}
	_, err = gostapi.CreateService(host, config)
	if err != nil {
		return err
	}
	g.autoPersist(ctx, serverId)
	return nil
}

// UpdateGostServiceOnServer 更新指定服务器上的 GOST 服务
func (g *clusterGostController) UpdateGostServiceOnServer(ctx context.Context, serverId int, name string, config *GostServiceConfig) error {
	host, err := resolveHost(serverId)
	if err != nil {
		return err
	}
	_, err = gostapi.UpdateService(host, name, config)
	if err != nil {
		return err
	}
	g.autoPersist(ctx, serverId)
	return nil
}

// DeleteGostServiceOnServer 删除指定服务器上的 GOST 服务
func (g *clusterGostController) DeleteGostServiceOnServer(ctx context.Context, serverId int, name string) error {
	host, err := resolveHost(serverId)
	if err != nil {
		return err
	}
	_, err = gostapi.DeleteService(host, name)
	if err != nil {
		return err
	}
	g.autoPersist(ctx, serverId)
	return nil
}

// autoPersist 自动持久化配置（异步，失败只打日志不影响主流程）
func (g *clusterGostController) autoPersist(ctx context.Context, serverId int) {
	go func() {
		if err := g.PersistGostConfigOnServer(ctx, serverId); err != nil {
			fmt.Printf("[GOST] 自动持久化失败(serverId=%d): %v\n", serverId, err)
		}
	}()
}

// ListGostChainsOnServer 列出指定服务器上的 GOST 链
func (g *clusterGostController) ListGostChainsOnServer(_ context.Context, serverId int) (*GostChainList, error) {
	host, err := resolveHost(serverId)
	if err != nil {
		return nil, err
	}
	return gostapi.GetChainList(host)
}

// PersistGostConfigOnServer 持久化指定服务器的 GOST 配置
func (g *clusterGostController) PersistGostConfigOnServer(ctx context.Context, serverId int) error {
	host, err := resolveHost(serverId)
	if err != nil {
		return err
	}

	config, err := gostapi.GetConfig(host, "")
	if err != nil {
		return fmt.Errorf("获取 GOST 运行配置失败: %w", err)
	}

	yamlContent, err := gostConfigToYAML(config)
	if err != nil {
		return fmt.Errorf("配置转换 YAML 失败: %w", err)
	}

	executor, err := g.cluster.getExecutor(serverId)
	if err != nil {
		return fmt.Errorf("获取执行器失败: %w", err)
	}

	ts := time.Now().Format("20060102_150405")
	backupPath := fmt.Sprintf("%s.%s.bak", gostConfigFilePath, ts)
	executor.Execute(ctx, fmt.Sprintf("sudo cp -f '%s' '%s' 2>/dev/null", gostConfigFilePath, backupPath))
	executor.Execute(ctx, "sudo mkdir -p /etc/gost")

	tmpPath := fmt.Sprintf("/tmp/gost-config-%d.yaml", time.Now().UnixNano())
	if err := executor.UploadFile(ctx, tmpPath, bytes.NewReader([]byte(yamlContent))); err != nil {
		return fmt.Errorf("上传配置文件失败: %w", err)
	}

	result := executor.Execute(ctx, fmt.Sprintf("sudo mv -f '%s' '%s'", tmpPath, gostConfigFilePath))
	if result.Err != nil {
		return fmt.Errorf("移动配置文件失败: %w", result.Err)
	}
	return nil
}

// GetGostConfigSyncStatusOnServer 获取指定服务器配置同步状态
func (g *clusterGostController) GetGostConfigSyncStatusOnServer(ctx context.Context, serverId int) (*GostConfigSyncStatus, error) {
	host, err := resolveHost(serverId)
	if err != nil {
		return nil, err
	}

	runningConfig, err := gostapi.GetConfig(host, "")
	if err != nil {
		return nil, fmt.Errorf("获取运行配置失败: %w", err)
	}

	executor, err := g.cluster.getExecutor(serverId)
	if err != nil {
		return nil, err
	}

	result := executor.Execute(ctx, fmt.Sprintf("sudo cat '%s' 2>/dev/null", gostConfigFilePath))
	fileContent := cleanSSHOutput(strings.TrimSpace(result.Output))

	if result.Err != nil || fileContent == "" {
		return &GostConfigSyncStatus{
			Synced:              false,
			RunningServiceCount: len(runningConfig.Services),
			RunningChainCount:   len(runningConfig.Chains),
			Message:             "配置文件不存在或为空",
		}, nil
	}

	return buildSyncStatus(runningConfig, fileContent), nil
}

// --- IController 兼容（代理到默认服务器） ---

func (g *clusterGostController) ListGostServices(ctx context.Context, page, size, port int) (*GostServiceList, error) {
	serverId, err := g.cluster.firstActiveServerId()
	if err != nil {
		return nil, err
	}
	return g.ListGostServicesOnServer(ctx, serverId, page, size, port)
}

func (g *clusterGostController) GetGostService(ctx context.Context, name string) (*GostServiceConfig, error) {
	serverId, err := g.cluster.firstActiveServerId()
	if err != nil {
		return nil, err
	}
	return g.GetGostServiceOnServer(ctx, serverId, name)
}

func (g *clusterGostController) CreateGostService(ctx context.Context, config *GostServiceConfig) error {
	serverId, err := g.cluster.firstActiveServerId()
	if err != nil {
		return err
	}
	return g.CreateGostServiceOnServer(ctx, serverId, config)
}

func (g *clusterGostController) UpdateGostService(ctx context.Context, name string, config *GostServiceConfig) error {
	serverId, err := g.cluster.firstActiveServerId()
	if err != nil {
		return err
	}
	return g.UpdateGostServiceOnServer(ctx, serverId, name, config)
}

func (g *clusterGostController) DeleteGostService(ctx context.Context, name string) error {
	serverId, err := g.cluster.firstActiveServerId()
	if err != nil {
		return err
	}
	return g.DeleteGostServiceOnServer(ctx, serverId, name)
}

func (g *clusterGostController) ListGostChains(ctx context.Context) (*GostChainList, error) {
	serverId, err := g.cluster.firstActiveServerId()
	if err != nil {
		return nil, err
	}
	return g.ListGostChainsOnServer(ctx, serverId)
}

func (g *clusterGostController) PersistGostConfig(ctx context.Context) error {
	serverId, err := g.cluster.firstActiveServerId()
	if err != nil {
		return err
	}
	return g.PersistGostConfigOnServer(ctx, serverId)
}

func (g *clusterGostController) GetGostConfigSyncStatus(ctx context.Context) (*GostConfigSyncStatus, error) {
	serverId, err := g.cluster.firstActiveServerId()
	if err != nil {
		return nil, err
	}
	return g.GetGostConfigSyncStatusOnServer(ctx, serverId)
}

// --- 共享工具函数 ---

const gostConfigFilePath = "/etc/gost/config.yaml"

func filterAndPaginateServices(data *GostServiceList, page, size, port int) *GostServiceList {
	if size <= 0 {
		size = 20
	}
	if page <= 0 {
		page = 1
	}

	list := data.List

	// 端口过滤
	if port > 0 {
		name := fmt.Sprintf("tcp-relay-%d", port)
		filtered := make([]gostapi.ServiceConfig, 0)
		for _, svc := range list {
			if strings.Contains(svc.Name, name) {
				filtered = append(filtered, svc)
			}
		}
		list = filtered
	}

	total := len(list)
	start := (page - 1) * size
	if start >= total {
		data.List = []gostapi.ServiceConfig{}
		return data
	}
	end := start + size
	if end > total {
		end = total
	}
	data.List = list[start:end]
	return data
}

func gostConfigToYAML(config *gostapi.Config) (string, error) {
	cleanConfig := *config
	for i := range cleanConfig.Services {
		cleanConfig.Services[i].Status = nil
	}

	jsonData, err := json.Marshal(cleanConfig)
	if err != nil {
		return "", err
	}

	var generic interface{}
	if err := json.Unmarshal(jsonData, &generic); err != nil {
		return "", err
	}

	yamlData, err := yaml.Marshal(generic)
	if err != nil {
		return "", err
	}
	return string(yamlData), nil
}

func buildSyncStatus(runningConfig *gostapi.Config, fileContent string) *GostConfigSyncStatus {
	var fileGeneric interface{}
	if err := yaml.Unmarshal([]byte(fileContent), &fileGeneric); err != nil {
		return &GostConfigSyncStatus{
			Synced:              false,
			RunningServiceCount: len(runningConfig.Services),
			RunningChainCount:   len(runningConfig.Chains),
			Message:             "配置文件 YAML 格式错误",
		}
	}

	fileServiceCount, fileChainCount := countServicesAndChains(fileGeneric)

	runningGeneric := configToGeneric(runningConfig)
	synced := compareConfigs(runningGeneric, fileGeneric)

	message := "已同步"
	if !synced {
		message = fmt.Sprintf("不同步: 运行中 %d 服务 / %d 链, 文件中 %d 服务 / %d 链",
			len(runningConfig.Services), len(runningConfig.Chains),
			fileServiceCount, fileChainCount)
	}

	return &GostConfigSyncStatus{
		Synced:              synced,
		RunningServiceCount: len(runningConfig.Services),
		RunningChainCount:   len(runningConfig.Chains),
		FileServiceCount:    fileServiceCount,
		FileChainCount:      fileChainCount,
		Message:             message,
	}
}

func configToGeneric(config *gostapi.Config) interface{} {
	cleanConfig := *config
	for i := range cleanConfig.Services {
		cleanConfig.Services[i].Status = nil
	}
	jsonData, _ := json.Marshal(cleanConfig)
	var generic interface{}
	json.Unmarshal(jsonData, &generic)
	return generic
}

func compareConfigs(running, file interface{}) bool {
	runningMap, ok1 := running.(map[string]interface{})
	fileMap, ok2 := file.(map[string]interface{})
	if !ok1 || !ok2 {
		return false
	}
	for _, key := range []string{"services", "chains"} {
		rVal, _ := json.Marshal(runningMap[key])
		fVal, _ := json.Marshal(fileMap[key])
		rStr := normalizeEmpty(string(rVal))
		fStr := normalizeEmpty(string(fVal))
		if rStr != fStr {
			return false
		}
	}
	return true
}

func normalizeEmpty(s string) string {
	if s == "null" || s == "" {
		return "[]"
	}
	return s
}

func countServicesAndChains(config interface{}) (int, int) {
	m, ok := config.(map[string]interface{})
	if !ok {
		return 0, 0
	}
	services, chains := 0, 0
	if svcs, ok := m["services"].([]interface{}); ok {
		services = len(svcs)
	}
	if chs, ok := m["chains"].([]interface{}); ok {
		chains = len(chs)
	}
	return services, chains
}

func cleanSSHOutput(output string) string {
	idx := strings.Index(output, "\nSTDERR:\n")
	if idx >= 0 {
		return output[:idx]
	}
	return output
}
