package deploy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"server/internal/server/model"
	"server/pkg/gostapi"
	"strings"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
	"gopkg.in/yaml.v3"
)

const gostConfigPath = "/etc/gost/config.yaml"

// PersistGostConfig 从 GOST API 获取运行配置，通过 SSH 写入 config.yaml
// 替代不可靠的 GOST API SaveConfig (POST /config)
func PersistGostConfig(serverId int) error {
	host, err := getServerHostById(serverId)
	if err != nil {
		return err
	}

	// 1. 获取运行中的完整配置
	config, err := gostapi.GetConfig(host, "")
	if err != nil {
		return fmt.Errorf("获取 GOST 运行配置失败: %w", err)
	}

	// 2. 转换为 YAML（Config 只有 json tag，需要 JSON→generic→YAML）
	yamlContent, err := configToYAML(config)
	if err != nil {
		return fmt.Errorf("配置转换 YAML 失败: %w", err)
	}

	// 3. 通过 SSH 写入文件
	client, err := GetSSHClient(serverId)
	if err != nil {
		return fmt.Errorf("获取 SSH 连接失败: %w", err)
	}

	// 备份旧配置（保留最近 5 份）
	ts := time.Now().Format("20060102_150405")
	backupPath := fmt.Sprintf("%s.%s.bak", gostConfigPath, ts)
	_, _ = client.ExecuteCommand(fmt.Sprintf("sudo cp -f '%s' '%s' 2>/dev/null", gostConfigPath, backupPath))
	// 清理旧备份，只保留最近 5 份
	_, _ = client.ExecuteCommand(fmt.Sprintf("ls -t %s.*.bak 2>/dev/null | tail -n +6 | xargs -r sudo rm -f", gostConfigPath))

	// 确保目录存在
	_, _ = client.ExecuteCommand("sudo mkdir -p /etc/gost")

	// 写入新配置（使用 UploadFile 避免 shell 转义问题）
	reader := bytes.NewReader([]byte(yamlContent))
	tmpPath := fmt.Sprintf("/tmp/gost-config-%d.yaml", time.Now().UnixNano())
	if err = client.UploadFile(tmpPath, reader); err != nil {
		return fmt.Errorf("上传配置文件失败: %w", err)
	}

	// 移动到目标位置
	if _, err = client.ExecuteCommand(fmt.Sprintf("sudo mv -f '%s' '%s'", tmpPath, gostConfigPath)); err != nil {
		return fmt.Errorf("移动配置文件失败: %w", err)
	}

	return nil
}

// GetGostConfigSyncStatus 比较运行配置与文件配置的同步状态
func GetGostConfigSyncStatus(serverId int) (*model.GostConfigSyncStatusResp, error) {
	host, err := getServerHostById(serverId)
	if err != nil {
		return nil, err
	}

	// 1. 获取运行配置
	runningConfig, err := gostapi.GetConfig(host, "")
	if err != nil {
		return nil, fmt.Errorf("获取运行配置失败: %w", err)
	}

	// 2. 通过 SSH 读取文件配置
	client, err := GetSSHClient(serverId)
	if err != nil {
		return nil, fmt.Errorf("获取 SSH 连接失败: %w", err)
	}

	fileContent, err := client.ExecuteCommand(fmt.Sprintf("sudo cat '%s' 2>/dev/null", gostConfigPath))
	if err != nil || strings.TrimSpace(fileContent) == "" {
		return &model.GostConfigSyncStatusResp{
			Synced:              false,
			RunningServiceCount: len(runningConfig.Services),
			RunningChainCount:   len(runningConfig.Chains),
			FileServiceCount:    0,
			FileChainCount:      0,
			Message:             "配置文件不存在或为空",
		}, nil
	}

	// 清理 SSH 输出中的 STDERR 行
	fileContent = cleanSSHOutput(fileContent)

	// 3. 解析文件配置
	var fileGeneric interface{}
	if err := yaml.Unmarshal([]byte(fileContent), &fileGeneric); err != nil {
		return &model.GostConfigSyncStatusResp{
			Synced:              false,
			RunningServiceCount: len(runningConfig.Services),
			RunningChainCount:   len(runningConfig.Chains),
			FileServiceCount:    0,
			FileChainCount:      0,
			Message:             "配置文件 YAML 格式错误",
		}, nil
	}

	// 提取文件中的服务数和链数
	fileServiceCount, fileChainCount := countServicesAndChains(fileGeneric)

	// 4. 比较：将运行配置也转为 generic 后比较核心部分
	runningGeneric := configToGeneric(runningConfig)
	synced := compareConfigs(runningGeneric, fileGeneric)

	message := "已同步"
	if !synced {
		message = fmt.Sprintf("不同步: 运行中 %d 服务 / %d 链, 文件中 %d 服务 / %d 链",
			len(runningConfig.Services), len(runningConfig.Chains),
			fileServiceCount, fileChainCount)
	}

	return &model.GostConfigSyncStatusResp{
		Synced:              synced,
		RunningServiceCount: len(runningConfig.Services),
		RunningChainCount:   len(runningConfig.Chains),
		FileServiceCount:    fileServiceCount,
		FileChainCount:      fileChainCount,
		Message:             message,
	}, nil
}

// configToYAML 将 GOST Config 转换为 YAML 字符串
// 由于 Config 只有 json tag，先序列化为 JSON 再转为 YAML
func configToYAML(config *gostapi.Config) (string, error) {
	// 清除运行时字段
	cleanConfig := *config
	for i := range cleanConfig.Services {
		cleanConfig.Services[i].Status = nil
	}

	// JSON → generic → YAML（保留 json tag 的字段名）
	jsonData, err := json.Marshal(cleanConfig)
	if err != nil {
		return "", fmt.Errorf("JSON 序列化失败: %w", err)
	}

	var generic interface{}
	if err := json.Unmarshal(jsonData, &generic); err != nil {
		return "", fmt.Errorf("JSON 反序列化失败: %w", err)
	}

	yamlData, err := yaml.Marshal(generic)
	if err != nil {
		return "", fmt.Errorf("YAML 序列化失败: %w", err)
	}

	return string(yamlData), nil
}

// configToGeneric 将 Config 结构体转为 generic interface（用于比较）
func configToGeneric(config *gostapi.Config) interface{} {
	// 清除运行时字段
	cleanConfig := *config
	for i := range cleanConfig.Services {
		cleanConfig.Services[i].Status = nil
	}

	jsonData, err := json.Marshal(cleanConfig)
	if err != nil {
		return nil
	}
	var generic interface{}
	_ = json.Unmarshal(jsonData, &generic)
	return generic
}

// compareConfigs 比较两个配置的 services 和 chains 是否一致
func compareConfigs(running, file interface{}) bool {
	runningMap, ok1 := running.(map[string]interface{})
	fileMap, ok2 := file.(map[string]interface{})
	if !ok1 || !ok2 {
		return false
	}

	// 只比较 services 和 chains（忽略 api、log 等静态配置）
	for _, key := range []string{"services", "chains"} {
		rVal, _ := json.Marshal(runningMap[key])
		fVal, _ := json.Marshal(fileMap[key])

		// 归一化：nil 和 [] 视为等价
		rStr := normalizeEmptyArray(string(rVal))
		fStr := normalizeEmptyArray(string(fVal))

		if rStr != fStr {
			logx.Infof("GOST config diff on key=%s: running=%s, file=%s", key, rStr, fStr)
			return false
		}
	}

	return true
}

// normalizeEmptyArray 将 "null" 和 "[]" 统一为 "[]"
func normalizeEmptyArray(s string) string {
	if s == "null" || s == "" {
		return "[]"
	}
	return s
}

// countServicesAndChains 从 generic 配置中提取服务和链的数量
func countServicesAndChains(config interface{}) (services int, chains int) {
	m, ok := config.(map[string]interface{})
	if !ok {
		return 0, 0
	}
	if svcs, ok := m["services"].([]interface{}); ok {
		services = len(svcs)
	}
	if chs, ok := m["chains"].([]interface{}); ok {
		chains = len(chs)
	}
	return
}

// cleanSSHOutput 清理 SSH 命令输出中的 STDERR 行
func cleanSSHOutput(output string) string {
	idx := strings.Index(output, "\nSTDERR:\n")
	if idx >= 0 {
		return output[:idx]
	}
	return output
}
