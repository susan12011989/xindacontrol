package consts

import (
	"os"
	"strconv"
)

// 默认值常量
const (
	defaultDeployPath      = "/root/teamgram/bin"
	defaultUsername        = "root"
	defaultPort            = 22
	defaultPassword        = "6oecMmbAo0Xq3okM1"
	defaultAliyunPassword  = "ch&Y2xTxC1-WXWDzxB&X"
	defaultDirectPort      = 10543
	defaultVersionsDir     = "/opt/control/versions"
	defaultAssetsDir       = "./assets"
)

// 可配置变量（支持环境变量覆盖）
var (
	DeployPath      = getEnvOrDefault("DEPLOY_PATH", defaultDeployPath)
	DefaultUsername = getEnvOrDefault("SSH_USERNAME", defaultUsername)
	DefaultPort     = getEnvOrDefaultInt("SSH_PORT", defaultPort)
	DefaultPassword = getEnvOrDefault("SSH_PASSWORD", defaultPassword)

	DefaultAliyunPassword = getEnvOrDefault("ALIYUN_PASSWORD", defaultAliyunPassword)

	DefalutDirectPort = getEnvOrDefaultInt("GOST_DIRECT_PORT", defaultDirectPort)

	// 新增配置项
	VersionsDir = getEnvOrDefault("VERSIONS_DIR", defaultVersionsDir)
	AssetsDir   = getEnvOrDefault("ASSETS_DIR", defaultAssetsDir)
)

// getEnvOrDefault 获取环境变量，如果不存在则返回默认值
func getEnvOrDefault(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

// getEnvOrDefaultInt 获取环境变量并转换为整数，如果不存在则返回默认值
func getEnvOrDefaultInt(key string, defaultValue int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return defaultValue
}
