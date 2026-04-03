package merchant

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"server/internal/dbhelper"
	"server/internal/server/cfg"
	"server/internal/server/utils"
	"server/pkg/consts"
	"server/pkg/dbs"
	"server/pkg/entity"
	"strings"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

// getMerchantAPIURL 获取商户API地址
// 多机模式下使用 API 节点地址，单机模式回退到 merchants.server_ip
// 端口从商户配置读取，默认 80
func getMerchantAPIURL(merchantNo string, path string) (string, error) {
	merchant, err := dbhelper.GetMerchantByNo(merchantNo)
	if err != nil {
		return "", err
	}

	port := merchant.Port
	if port == 0 {
		port = 80
	}

	// 优先从 service_nodes 获取 API 节点地址
	hosts, hostsErr := GetMerchantServiceHosts(merchant.Id)
	if hostsErr == nil && hosts.APIHost != "" {
		return fmt.Sprintf("http://%s:%d%s", hosts.APIHost, port, path), nil
	}

	// 回退到 merchants.server_ip
	if merchant.ServerIP == "" {
		return "", fmt.Errorf("商户服务器IP为空")
	}
	return fmt.Sprintf("http://%s:%d%s", merchant.ServerIP, port, path), nil
}

// doMerchantRequest 向商户发起HTTP请求
func doMerchantRequest(method, url string, body interface{}) (*http.Response, error) {
	return doMerchantRequestWithTimeout(method, url, body, 15*time.Second)
}

// doMerchantRequestWithTimeout 向商户发起HTTP请求（自定义超时）
func doMerchantRequestWithTimeout(method, url string, body interface{}, timeout time.Duration) (*http.Response, error) {
	if cfg.C.MerchantAPI == nil {
		return nil, fmt.Errorf("MerchantAPI配置未设置")
	}

	var reqBody *bytes.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reqBody = bytes.NewReader(jsonData)
	} else {
		reqBody = bytes.NewReader(nil)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth(cfg.C.MerchantAPI.Username, cfg.C.MerchantAPI.Password)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: timeout}
	return client.Do(req)
}

// ============ Sensitive Contents ============

// SaveAdminmSensitiveContents 保存敏感词配置到商户
func SaveAdminmSensitiveContents(merchantNo string, contents []*entity.SensitiveContent) error {
	url, err := getMerchantAPIURL(merchantNo, "/v1/control/config/sensitive_contents")
	if err != nil {
		return err
	}

	payload := map[string]interface{}{
		"contents": contents,
	}

	resp, err := doMerchantRequest("POST", url, payload)
	if err != nil {
		logx.Errorf("保存商户敏感词配置失败: merchant=%s, err=%v", merchantNo, err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("保存失败，状态码: %d", resp.StatusCode)
	}

	logx.Infof("敏感词配置已保存: merchant=%s, count=%d", merchantNo, len(contents))
	return nil
}

// ============ SMS Config ============

// GetAdminmSmsConfig 从商户服务器读取短信配置
func GetAdminmSmsConfig(merchantNo string) (*entity.SmsConfig, error) {
	url, err := getMerchantAPIURL(merchantNo, "/v1/control/config/sms")
	if err != nil {
		return nil, err
	}

	resp, err := doMerchantRequest("GET", url, nil)
	if err != nil {
		logx.Errorf("获取商户短信配置失败: merchant=%s, err=%v", merchantNo, err)
		return nil, fmt.Errorf("请求商户服务器失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("获取失败，状态码: %d", resp.StatusCode)
	}

	var result struct {
		Config *entity.SmsConfig `json:"config"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v", err)
	}

	return result.Config, nil
}

// SaveAdminmSmsConfig 保存短信配置到商户
func SaveAdminmSmsConfig(merchantNo string, config *entity.SmsConfig) error {
	url, err := getMerchantAPIURL(merchantNo, "/v1/control/config/sms")
	if err != nil {
		return err
	}

	payload := map[string]interface{}{
		"config": config,
	}

	resp, err := doMerchantRequest("POST", url, payload)
	if err != nil {
		logx.Errorf("保存商户短信配置失败: merchant=%s, err=%v", merchantNo, err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("保存失败，状态码: %d", resp.StatusCode)
	}

	logx.Infof("短信配置已保存: merchant=%s", merchantNo)
	return nil
}

// ============ System User Nickname ============

// SaveAdminmSystemNickname 保存系统用户昵称到商户
func SaveAdminmSystemNickname(merchantNo string, firstName string) error {
	url, err := getMerchantAPIURL(merchantNo, "/v1/control/config/system_user_nickname")
	if err != nil {
		return err
	}

	payload := map[string]interface{}{
		"first_name": firstName,
	}

	resp, err := doMerchantRequest("POST", url, payload)
	if err != nil {
		logx.Errorf("保存商户系统昵称失败: merchant=%s, err=%v", merchantNo, err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("保存失败，状态码: %d", resp.StatusCode)
	}

	logx.Infof("系统昵称已保存: merchant=%s, firstName=%s", merchantNo, firstName)
	return nil
}

// SaveAdminmSystemProfile 保存系统用户昵称和头像到商户
func SaveAdminmSystemProfile(merchantNo string, firstName string, avatarUrl string) error {
	// 1. 更新昵称（通过已有的 system_user_nickname 接口）
	if firstName != "" {
		if err := SaveAdminmSystemNickname(merchantNo, firstName); err != nil {
			return fmt.Errorf("更新昵称失败: %v", err)
		}
	}

	// 2. 更新头像（通过已有的 config/update 接口推送 logo_url，商户端会自动写入 app.app_logo）
	if avatarUrl != "" {
		// 如果是相对路径（/assets/...），拼上 control 外网地址
		if strings.HasPrefix(avatarUrl, "/") {
			avatarUrl = resolveControlAssetURL(avatarUrl)
		}

		url, err := getMerchantAPIURL(merchantNo, "/v1/control/config/update")
		if err != nil {
			return err
		}

		payload := map[string]interface{}{
			"logo_url": avatarUrl,
		}

		resp, err := doMerchantRequest("POST", url, payload)
		if err != nil {
			logx.Errorf("更新商户系统头像失败: merchant=%s, err=%v", merchantNo, err)
			return fmt.Errorf("更新头像失败: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			return fmt.Errorf("更新头像失败，状态码: %d", resp.StatusCode)
		}
		logx.Infof("系统头像已更新: merchant=%s, avatarUrl=%s", merchantNo, avatarUrl)
	}

	return nil
}

// resolveControlAssetURL 将 control 本地相对路径转为完整的外网可访问 URL
// 例如 /assets/logo/xxx.png -> http://16.163.223.214:58181/assets/logo/xxx.png
func resolveControlAssetURL(path string) string {
	externalURL := cfg.C.ExternalURL
	if externalURL == "" {
		logx.Errorf("resolveControlAssetURL: ExternalURL 未配置，返回原路径: %s", path)
		return path
	}
	return strings.TrimRight(externalURL, "/") + path
}

// ============ Export Database ============

// GetMerchantSSHClient 获取商户服务器的 SSH 客户端
func GetMerchantSSHClient(merchantNo string) (*utils.SSHClient, error) {
	merchant, err := dbhelper.GetMerchantByNo(merchantNo)
	if err != nil {
		return nil, fmt.Errorf("获取商户信息失败: %v", err)
	}

	var server entity.Servers
	has, err := dbs.DBAdmin.Where("merchant_id = ? AND server_type = 1 AND status = 1", merchant.Id).Get(&server)
	if err != nil {
		return nil, fmt.Errorf("查询商户服务器失败: %v", err)
	}
	if !has {
		return nil, fmt.Errorf("未找到商户服务器")
	}

	return &utils.SSHClient{
		Host:       server.Host,
		Port:       server.Port,
		Username:   server.Username,
		Password:   server.Password,
		PrivateKey: server.PrivateKey,
	}, nil
}

// GetMerchantWebSSHClient 获取运行 Web(nginx) 的商户服务器 SSH 客户端（返回第一个匹配的）
// 多机部署时 web 可能在独立的 app 节点上，不能用 GetMerchantSSHClient（可能连到 DB/MinIO 节点）
func GetMerchantWebSSHClient(merchantNo string) (*utils.SSHClient, error) {
	clients, err := GetAllMerchantWebSSHClients(merchantNo)
	if err != nil {
		return nil, err
	}
	if len(clients) == 0 {
		return nil, fmt.Errorf("未找到商户Web服务器")
	}
	return clients[0], nil
}

// GetAllMerchantWebSSHClients 获取商户所有运行 Web(nginx) 的服务器 SSH 客户端
// 多节点部署时返回所有 app/web/all 节点，用于批量推送资源
func GetAllMerchantWebSSHClients(merchantNo string) ([]*utils.SSHClient, error) {
	merchant, err := dbhelper.GetMerchantByNo(merchantNo)
	if err != nil {
		return nil, fmt.Errorf("获取商户信息失败: %v", err)
	}

	// 从 MerchantServiceNodes 查找所有 web/app/all 角色的节点
	var nodes []entity.MerchantServiceNodes
	if err := dbs.DBAdmin.Where("merchant_id = ? AND status = 1", merchant.Id).Find(&nodes); err != nil {
		return nil, fmt.Errorf("查询服务节点失败: %v", err)
	}

	webRoles := map[string]bool{entity.ServiceNodeRoleWeb: true, "app": true, entity.ServiceNodeRoleAll: true, entity.ServiceNodeRoleAPI: true}
	var serverIds []int
	for _, n := range nodes {
		if webRoles[n.Role] && n.ServerId > 0 {
			serverIds = append(serverIds, n.ServerId)
		}
	}

	if len(serverIds) == 0 {
		// 没有 service_nodes 记录，回退到旧逻辑
		var server entity.Servers
		has, err := dbs.DBAdmin.Where("merchant_id = ? AND server_type = 1 AND status = 1", merchant.Id).Get(&server)
		if err != nil {
			return nil, fmt.Errorf("查询服务器失败: %v", err)
		}
		if !has {
			return nil, fmt.Errorf("未找到商户Web服务器")
		}
		return []*utils.SSHClient{{
			Host: server.Host, Port: server.Port,
			Username: server.Username, Password: server.Password, PrivateKey: server.PrivateKey,
		}}, nil
	}

	var servers []entity.Servers
	if err := dbs.DBAdmin.In("id", serverIds).Where("status = 1").Find(&servers); err != nil {
		return nil, fmt.Errorf("查询服务器失败: %v", err)
	}
	if len(servers) == 0 {
		return nil, fmt.Errorf("未找到商户Web服务器")
	}

	clients := make([]*utils.SSHClient, 0, len(servers))
	for _, s := range servers {
		clients = append(clients, &utils.SSHClient{
			Host: s.Host, Port: s.Port,
			Username: s.Username, Password: s.Password, PrivateKey: s.PrivateKey,
		})
	}
	return clients, nil
}

// ============ Clear Data ============

// ClearMerchantData 清除商户所有用户数据（保留系统账号和配置）
// 清除范围：MySQL 用户数据 + Redis 缓存 + WuKongIM 消息 + MinIO 文件
func ClearMerchantData(merchantNo string) error {
	// 1. 调用 TSDD API 清除 MySQL 数据（保留系统账号、文件助手、管理员）
	url, err := getMerchantAPIURL(merchantNo, "/v1/control/data/clear")
	if err != nil {
		return err
	}

	resp, err := doMerchantRequest("POST", url, nil)
	if err != nil {
		logx.Errorf("清除商户MySQL数据失败: merchant=%s, err=%v", merchantNo, err)
		return fmt.Errorf("请求商户服务器失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("清除MySQL数据失败，状态码: %d", resp.StatusCode)
	}
	logx.Infof("商户MySQL数据已清除: merchant=%s", merchantNo)

	// 2. SSH 到商户服务器清除 Redis/WuKongIM/MinIO
	merchant, err := dbhelper.GetMerchantByNo(merchantNo)
	if err != nil {
		logx.Errorf("获取商户信息失败（MySQL已清除）: %v", err)
		return nil // MySQL 已清除成功，SSH 清理失败不阻塞
	}

	// 查找商户服务器
	var server entity.Servers
	has, _ := dbs.DBAdmin.Where("merchant_id = ? AND server_type = 1 AND status = 1", merchant.Id).Get(&server)
	if !has {
		logx.Infof("未找到商户服务器，跳过 SSH 清理: merchant=%s", merchantNo)
		return nil
	}

	// 构建 SSH 客户端
	sshClient := &utils.SSHClient{
		Host:       server.Host,
		Port:       server.Port,
		Username:   server.Username,
		Password:   server.Password,
		PrivateKey: server.PrivateKey,
	}

	// 清理脚本：Redis + WuKongIM + MinIO + 重启服务
	// 注意：用 docker stop/start（container name）而非 docker compose stop/up（service name）
	// 因为 service name 和 container name 可能不同（如 wukongim vs tsdd-wukongim）
	cleanScript := `
set -e
cd /opt/tsdd

echo "=== Flushing Redis ==="
docker exec tsdd-redis redis-cli FLUSHALL 2>/dev/null || echo "Redis flush skipped (container not found)"

echo "=== Clearing WuKongIM data ==="
WKIM_VOLUME=$(docker inspect tsdd-wukongim --format='{{range .Mounts}}{{if eq .Destination "/root/wukongim"}}{{.Source}}{{end}}{{end}}' 2>/dev/null || echo "")
if [ -n "$WKIM_VOLUME" ] && [ -d "$WKIM_VOLUME" ]; then
    docker stop tsdd-wukongim 2>/dev/null || true
    sleep 2
    sudo rm -rf "$WKIM_VOLUME"/* 2>/dev/null || true
    docker start tsdd-wukongim 2>/dev/null || true
    # 等待 WuKongIM 启动就绪
    for i in $(seq 1 15); do
        if curl -s -o /dev/null -w '%{http_code}' http://127.0.0.1:5002/health 2>/dev/null | grep -q '200'; then
            echo "WuKongIM started (${i}s)"
            break
        fi
        sleep 1
    done
    echo "WuKongIM data cleared"
else
    echo "WuKongIM volume not found, skipped"
fi

echo "=== Clearing MinIO data ==="
MINIO_VOLUME=$(docker inspect tsdd-minio --format='{{range .Mounts}}{{if eq .Destination "/data"}}{{.Source}}{{end}}{{end}}' 2>/dev/null || echo "")
if [ -n "$MINIO_VOLUME" ] && [ -d "$MINIO_VOLUME" ]; then
    docker stop tsdd-minio 2>/dev/null || true
    sleep 2
    sudo rm -rf "$MINIO_VOLUME"/* 2>/dev/null || true
    docker start tsdd-minio 2>/dev/null || true
    sleep 3
    echo "MinIO data cleared"
else
    echo "MinIO volume not found, skipped"
fi

echo "=== Restarting TSDD server ==="
docker restart tsdd-server 2>/dev/null || true

echo "=== All clear done ==="
`

	output, sshErr := sshClient.ExecuteCommandWithTimeout(cleanScript, 120*time.Second)
	if sshErr != nil {
		logx.Errorf("SSH清理失败（MySQL已清除）: merchant=%s, err=%v, output=%s", merchantNo, sshErr, output)
		// MySQL 已成功清除，SSH 清理失败记录日志但不返回错误
	} else {
		logx.Infof("商户全部数据已清除: merchant=%s, output=%s", merchantNo, output)
	}

	return nil
}

// PushWebLogo 推送 Logo + 应用名称到商户所有 Web 节点
// logoPath 为 Control 服务器上 logo 原图的本地路径（空则不推 logo），appName 为应用显示名称（空则不改名称）
func PushWebLogo(merchantNo string, logoPath string, appName string) error {
	// 生成 logo 文件（如果有 logoPath）
	var files map[string][]byte
	if logoPath != "" {
		var err error
		files, err = utils.GenerateWebLogoFiles(logoPath)
		if err != nil {
			return fmt.Errorf("生成Logo文件失败: %v", err)
		}
	}

	if len(files) == 0 && appName == "" {
		return nil
	}

	clients, err := GetAllMerchantWebSSHClients(merchantNo)
	if err != nil {
		return fmt.Errorf("获取Web服务器SSH连接失败: %v", err)
	}

	var lastErr error
	for _, sshClient := range clients {
		if err := pushWebLogoToNode(sshClient, merchantNo, files, appName); err != nil {
			logx.Errorf("PushWebLogo: 推送到 %s 失败: %v", sshClient.Host, err)
			lastErr = err
		}
	}
	return lastErr
}

// pushWebLogoToNode 推送 Logo 到单个节点
func pushWebLogoToNode(sshClient *utils.SSHClient, merchantNo string, files map[string][]byte, appName string) error {
	defer sshClient.Close()

	if err := sshClient.Connect(); err != nil {
		return fmt.Errorf("SSH连接失败: %v", err)
	}
	logx.Infof("PushWebLogo: 连接到Web服务器 %s, merchant=%s", sshClient.Host, merchantNo)

	tmpDir := "/tmp/tsdd-logo"
	hasLogoFiles := len(files) > 0
	if hasLogoFiles {
		if _, err := sshClient.ExecuteCommandWithTimeout(fmt.Sprintf("mkdir -p %s", tmpDir), 10*time.Second); err != nil {
			return fmt.Errorf("创建临时目录失败: %v", err)
		}
		for filename, data := range files {
			remotePath := fmt.Sprintf("%s/%s", tmpDir, filename)
			if err := sshClient.UploadFile(remotePath, bytes.NewReader(data)); err != nil {
				return fmt.Errorf("上传 %s 失败: %v", filename, err)
			}
		}
	}

	escapedName := ""
	if appName != "" {
		escapedName = strings.ReplaceAll(appName, `/`, `\/`)
		escapedName = strings.ReplaceAll(escapedName, `&`, `\&`)
	}

	// 构建统一脚本：先探测 web 文件位置，再执行所有操作
	script := fmt.Sprintf(`
set -e
WEB_DIR="/opt/tsdd/web"
FOUND_DIR=""

# 第一步：探测 index.html 实际所在位置
# 优先：宿主机 /opt/tsdd/web
if [ -f "$WEB_DIR/index.html" ]; then
    FOUND_DIR="$WEB_DIR"
    MODE="host"
    echo "MODE=host, dir=$FOUND_DIR"
fi

# 如果宿主机没有 index.html，尝试 docker 容器
if [ -z "$FOUND_DIR" ]; then
    NGINX_CONTAINER=""
    if docker ps --format '{{.Names}}' | grep -q '^tsdd-web$'; then
        NGINX_CONTAINER="tsdd-web"
    elif docker ps --format '{{.Names}}' | grep -q '^tsdd-nginx$'; then
        NGINX_CONTAINER="tsdd-nginx"
    fi
    if [ -n "$NGINX_CONTAINER" ]; then
        # tsdd-nginx 的 web root 是 /usr/share/nginx/html/web
        # tsdd-web 的 web root 是 /usr/share/nginx/html
        if docker exec "$NGINX_CONTAINER" test -f /usr/share/nginx/html/web/index.html 2>/dev/null; then
            CONTAINER_WEB_ROOT="/usr/share/nginx/html/web"
        elif docker exec "$NGINX_CONTAINER" test -f /usr/share/nginx/html/index.html 2>/dev/null; then
            CONTAINER_WEB_ROOT="/usr/share/nginx/html"
        fi
        if [ -n "$CONTAINER_WEB_ROOT" ]; then
            MODE="docker"
            echo "MODE=docker, container=$NGINX_CONTAINER, root=$CONTAINER_WEB_ROOT"
        fi
    fi
fi

if [ -z "$MODE" ]; then
    echo "ERROR: 未找到 web index.html（宿主机 $WEB_DIR 和 docker 容器均无）"
    exit 1
fi
`)

	// Logo 文件复制
	if hasLogoFiles {
		script += fmt.Sprintf(`
# 第二步：复制 Logo 文件
if [ "$MODE" = "host" ]; then
    sudo cp %s/* "$FOUND_DIR/"
    echo "Logo files copied to host: $FOUND_DIR"
else
    for f in %s/*; do
        fname=$(basename "$f")
        docker cp "$f" "${NGINX_CONTAINER}:${CONTAINER_WEB_ROOT}/${fname}"
    done
    echo "Logo files copied to container: $NGINX_CONTAINER:$CONTAINER_WEB_ROOT"
fi
`, tmpDir, tmpDir)
	}

	// 应用名称修改
	if appName != "" {
		script += fmt.Sprintf(`
# 第三步：修改应用名称
if [ "$MODE" = "host" ]; then
    echo "Before: $(grep '<title>' "$FOUND_DIR/index.html" 2>/dev/null || echo 'no title found')"
    sudo sed -i 's|<title>[^<]*</title>|<title>%s</title>|' "$FOUND_DIR/index.html"
    sudo sed -i 's|"name": "[^"]*"|"name": "%s"|' "$FOUND_DIR/manifest.json"
    sudo sed -i 's|"short_name": "[^"]*"|"short_name": "%s"|' "$FOUND_DIR/manifest.json"
    echo "After:  $(grep '<title>' "$FOUND_DIR/index.html" 2>/dev/null || echo 'no title found')"
else
    echo "Before: $(docker exec "$NGINX_CONTAINER" grep '<title>' "${CONTAINER_WEB_ROOT}/index.html" 2>/dev/null || echo 'no title found')"
    docker exec "$NGINX_CONTAINER" sed -i 's|<title>[^<]*</title>|<title>%s</title>|' "${CONTAINER_WEB_ROOT}/index.html"
    docker exec "$NGINX_CONTAINER" sed -i 's|"name": "[^"]*"|"name": "%s"|' "${CONTAINER_WEB_ROOT}/manifest.json"
    docker exec "$NGINX_CONTAINER" sed -i 's|"short_name": "[^"]*"|"short_name": "%s"|' "${CONTAINER_WEB_ROOT}/manifest.json"
    echo "After:  $(docker exec "$NGINX_CONTAINER" grep '<title>' "${CONTAINER_WEB_ROOT}/index.html" 2>/dev/null || echo 'no title found')"
fi
`, escapedName, escapedName, escapedName, escapedName, escapedName, escapedName)
	}

	// Nginx reload + 清理
	script += `
# 第四步：Reload nginx
if [ -n "$NGINX_CONTAINER" ]; then
    docker exec "$NGINX_CONTAINER" nginx -s reload 2>/dev/null && echo "nginx reloaded: $NGINX_CONTAINER" || echo "WARN: nginx reload failed"
elif docker ps --format '{{.Names}}' | grep -q '^tsdd-web$'; then
    docker exec tsdd-web nginx -s reload 2>/dev/null && echo "nginx reloaded: tsdd-web" || echo "WARN: nginx reload failed"
elif docker ps --format '{{.Names}}' | grep -q '^tsdd-nginx$'; then
    docker exec tsdd-nginx nginx -s reload 2>/dev/null && echo "nginx reloaded: tsdd-nginx" || echo "WARN: nginx reload failed"
else
    echo "WARN: no nginx container found, skip reload"
fi
`
	if hasLogoFiles {
		script += fmt.Sprintf("rm -rf %s\n", tmpDir)
	}
	script += `echo "OK"`

	out, err := sshClient.ExecuteCommandWithTimeout(script, 60*time.Second)
	if err != nil {
		logx.Errorf("推送Logo失败: merchant=%s, err=%v, output=%s", merchantNo, err, out)
		if hasLogoFiles {
			sshClient.ExecuteCommandSilent(fmt.Sprintf("rm -rf %s", tmpDir))
		}
		return fmt.Errorf("推送Logo失败: %v", err)
	}

	logx.Infof("Logo已推送: merchant=%s, files=%d, appName=%s, output=%s", merchantNo, len(files), appName, out)
	return nil
}

// ReapplyWebBranding 根据商户 ID 重新应用 Web 品牌配置（logo/应用名称）
func ReapplyWebBranding(merchantId int) {
	var m entity.Merchants
	has, err := dbs.DBAdmin.Where("id = ?", merchantId).Get(&m)
	if err != nil || !has {
		logx.Infof("[ReapplyWebBranding] 商户不存在: merchantId=%d", merchantId)
		return
	}

	appName := m.AppName
	if appName == "" {
		appName = m.Name
	}
	if m.LogoUrl == "" && appName == "" {
		return
	}

	var logoPath string
	if m.LogoUrl != "" {
		logoRelPath := strings.TrimPrefix(m.LogoUrl, "/assets/")
		logoPath = filepath.Join(consts.AssetsDir, logoRelPath)
		if _, err := os.Stat(logoPath); os.IsNotExist(err) {
			logx.Errorf("[ReapplyWebBranding] Logo文件不存在: path=%s, merchant=%s", logoPath, m.No)
			logoPath = ""
		}
	}
	if logoPath == "" && appName == "" {
		return
	}

	time.Sleep(15 * time.Second)

	if err := PushWebLogo(m.No, logoPath, appName); err != nil {
		logx.Errorf("[ReapplyWebBranding] 推送失败: merchant=%s, err=%v", m.No, err)
	} else {
		logx.Infof("[ReapplyWebBranding] 推送成功: merchant=%s", m.No)
	}
}

