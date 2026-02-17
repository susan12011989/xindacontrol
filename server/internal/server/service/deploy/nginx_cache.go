package deploy

import (
	"fmt"
	"server/internal/server/model"
	"server/pkg/gostapi"
	"strings"

	"github.com/zeromicro/go-zero/core/logx"
	"golang.org/x/crypto/ssh"
)

// Nginx 缓存相关常量
const (
	NginxCacheDir     = "/var/cache/nginx/gost_cache"
	NginxBaseConfPath = "/etc/nginx/conf.d/gost-cache.conf"
)

// installNginxViaSSH 通过 SSH 客户端安装 Nginx（用于 GOST 部署时同步安装）
func installNginxViaSSH(client *ssh.Client) error {
	installScript := fmt.Sprintf(`#!/bin/bash
set -e

if [ "$(id -u)" -eq 0 ]; then SUDO=""; else SUDO="sudo"; fi

echo ">>> 安装 Nginx..."
$SUDO apt-get update -qq
$SUDO apt-get install -y -qq nginx

echo ">>> 配置 Nginx 缓存目录..."
$SUDO mkdir -p %s
$SUDO chown www-data:www-data %s

echo ">>> 创建基础缓存配置..."
$SUDO tee %s > /dev/null << 'NGINXEOF'
# GOST HTTP 缓存代理 - 基础配置
# 端口配置在 /etc/nginx/conf.d/gost-port-*.conf
proxy_cache_path %s
    levels=1:2
    keys_zone=gost_media:10m
    max_size=2g
    inactive=7d
    use_temp_path=off;
NGINXEOF

# 移除默认站点，避免端口 80 冲突
$SUDO rm -f /etc/nginx/sites-enabled/default

echo ">>> 启动 Nginx..."
$SUDO systemctl enable nginx
$SUDO nginx -t && $SUDO systemctl restart nginx

echo ">>> Nginx 安装完成!"
`, NginxCacheDir, NginxCacheDir, NginxBaseConfPath, NginxCacheDir)

	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("创建 SSH 会话失败: %w", err)
	}
	defer session.Close()

	output, err := session.CombinedOutput(installScript)
	if err != nil {
		return fmt.Errorf("安装 Nginx 失败: %s, output: %s", err, string(output))
	}

	return nil
}

// InstallNginxToServer 在已有服务器上安装 Nginx（供 API 调用）
func InstallNginxToServer(serverId int, progressCallback func(string)) error {
	client, err := GetSSHClient(serverId)
	if err != nil {
		return fmt.Errorf("获取 SSH 连接失败: %w", err)
	}
	defer client.Close()

	progressCallback("连接成功，开始安装 Nginx...")

	// 检查是否已安装
	output, _ := client.ExecuteCommand("which nginx")
	if strings.TrimSpace(output) != "" {
		progressCallback("Nginx 已安装，检查配置...")
		// 确保缓存目录和基础配置存在
		_, _ = client.ExecuteCommand(fmt.Sprintf("sudo mkdir -p %s && sudo chown www-data:www-data %s", NginxCacheDir, NginxCacheDir))

		// 检查基础配置
		checkOutput, _ := client.ExecuteCommand(fmt.Sprintf("cat %s 2>/dev/null", NginxBaseConfPath))
		if !strings.Contains(checkOutput, "gost_media") {
			progressCallback("更新基础缓存配置...")
			baseConf := fmt.Sprintf(`proxy_cache_path %s levels=1:2 keys_zone=gost_media:10m max_size=2g inactive=7d use_temp_path=off;`, NginxCacheDir)
			_, err = client.ExecuteCommand(fmt.Sprintf("echo '%s' | sudo tee %s > /dev/null", baseConf, NginxBaseConfPath))
			if err != nil {
				progressCallback(fmt.Sprintf("警告: 写入基础配置失败: %s", err))
			}
		}

		_, _ = client.ExecuteCommand("sudo rm -f /etc/nginx/sites-enabled/default")
		_, _ = client.ExecuteCommand("sudo nginx -t && sudo systemctl reload nginx")
		progressCallback("Nginx 配置更新完成")
		return nil
	}

	// 安装 Nginx
	progressCallback("安装 Nginx 包...")
	_, err = client.ExecuteCommand("sudo apt-get update -qq && sudo apt-get install -y -qq nginx")
	if err != nil {
		return fmt.Errorf("安装 Nginx 失败: %w", err)
	}

	progressCallback("配置缓存目录...")
	_, _ = client.ExecuteCommand(fmt.Sprintf("sudo mkdir -p %s && sudo chown www-data:www-data %s", NginxCacheDir, NginxCacheDir))

	progressCallback("写入基础配置...")
	baseConf := fmt.Sprintf(`proxy_cache_path %s levels=1:2 keys_zone=gost_media:10m max_size=2g inactive=7d use_temp_path=off;`, NginxCacheDir)
	_, err = client.ExecuteCommand(fmt.Sprintf("echo '%s' | sudo tee %s > /dev/null", baseConf, NginxBaseConfPath))
	if err != nil {
		return fmt.Errorf("写入基础配置失败: %w", err)
	}

	// 移除默认站点
	_, _ = client.ExecuteCommand("sudo rm -f /etc/nginx/sites-enabled/default")

	progressCallback("启动 Nginx...")
	_, _ = client.ExecuteCommand("sudo systemctl enable nginx")
	_, err = client.ExecuteCommand("sudo nginx -t && sudo systemctl restart nginx")
	if err != nil {
		return fmt.Errorf("启动 Nginx 失败: %w", err)
	}

	progressCallback("✓ Nginx 安装完成!")
	return nil
}

// generateNginxPortConfig 生成指定端口的 Nginx 配置内容
func generateNginxPortConfig(httpPort int) string {
	return fmt.Sprintf(`# GOST 缓存代理 - 端口 %d（自动生成，请勿手动修改）
server {
    listen %d;
    listen [::]:%d;

    # 媒体文件 → 缓存（图片/视频/音频/文档）
    location ~* \.(jpg|jpeg|png|gif|webp|bmp|ico|svg|mp4|avi|mkv|mov|webm|mp3|wav|ogg|aac|flac|amr|pdf|doc|docx|xls|xlsx|ppt|pptx|zip|rar|7z)$ {
        proxy_pass http://127.0.0.1:%d;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;

        proxy_cache gost_media;
        proxy_cache_valid 200 7d;
        proxy_cache_valid 304 7d;
        proxy_cache_key $uri$is_args$args;
        proxy_cache_use_stale error timeout updating http_500 http_502 http_503 http_504;
        proxy_ignore_headers Cache-Control Expires Set-Cookie;

        add_header X-Cache-Status $upstream_cache_status;
        proxy_max_temp_file_size 100m;
    }

    # 其他请求（API等）→ 直接透传，不缓存
    location / {
        proxy_pass http://127.0.0.1:%d;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    }
}
`, httpPort, httpPort, httpPort, httpPort, httpPort)
}

// ConfigureNginxCacheForPort 为指定 HTTP 端口配置 Nginx 缓存
func ConfigureNginxCacheForPort(serverId int, httpPort int) error {
	client, err := GetSSHClient(serverId)
	if err != nil {
		return fmt.Errorf("获取 SSH 连接失败: %w", err)
	}
	defer client.Close()

	confPath := fmt.Sprintf("/etc/nginx/conf.d/gost-port-%d.conf", httpPort)
	confContent := generateNginxPortConfig(httpPort)

	// 写入 Nginx 配置（用 heredoc 避免特殊字符问题）
	cmd := fmt.Sprintf("cat << 'CONFEOF' | sudo tee %s > /dev/null\n%sCONFEOF", confPath, confContent)
	_, err = client.ExecuteCommand(cmd)
	if err != nil {
		return fmt.Errorf("写入 Nginx 配置失败: %w", err)
	}

	// 测试配置并 reload
	_, err = client.ExecuteCommand("sudo nginx -t && sudo systemctl reload nginx")
	if err != nil {
		// 配置有问题，删除并回滚
		_, _ = client.ExecuteCommand(fmt.Sprintf("sudo rm -f %s && sudo systemctl reload nginx", confPath))
		return fmt.Errorf("Nginx 配置检查失败: %w", err)
	}

	logx.Infof("Nginx 缓存配置成功: 服务器 %d, 端口 %d", serverId, httpPort)
	return nil
}

// UpdateGostServiceToLoopback 修改 GOST HTTP 服务地址为 127.0.0.1（仅本地监听）
func UpdateGostServiceToLoopback(serverId int, httpPort int) error {
	host, err := getServerHostById(serverId)
	if err != nil {
		return err
	}

	// 查找 HTTP 端口对应的 GOST 服务名
	// 服务名格式: http-relay-{port} 或 http-direct-{port} 或 tcp-relay-{port}（默认端口）
	serviceNames := []string{
		fmt.Sprintf("http-relay-%d", httpPort),
		fmt.Sprintf("http-direct-%d", httpPort),
		fmt.Sprintf("tcp-relay-%d", httpPort),
	}

	for _, name := range serviceNames {
		svc, err := gostapi.GetService(host, name)
		if err != nil {
			continue // 服务不存在，尝试下一个名称
		}

		// 已经是 loopback 地址，跳过
		newAddr := fmt.Sprintf("127.0.0.1:%d", httpPort)
		if svc.Addr == newAddr {
			return nil
		}

		// 更新为 loopback 地址
		svc.Addr = newAddr
		_, err = gostapi.UpdateService(host, name, svc)
		if err != nil {
			return fmt.Errorf("更新 GOST 服务地址失败: %w", err)
		}

		// 保存配置到文件
		_, err = gostapi.SaveConfig(host, "yaml", "")
		if err != nil {
			logx.Errorf("GOST 配置保存失败: %v", err)
		}

		logx.Infof("GOST HTTP 服务已切换为 loopback: %s → %s", name, newAddr)
		return nil
	}

	return fmt.Errorf("未找到端口 %d 对应的 GOST HTTP 服务", httpPort)
}

// RestoreGostServiceToPublic 恢复 GOST HTTP 服务地址为公网监听
func RestoreGostServiceToPublic(serverId int, httpPort int) error {
	host, err := getServerHostById(serverId)
	if err != nil {
		return err
	}

	serviceNames := []string{
		fmt.Sprintf("http-relay-%d", httpPort),
		fmt.Sprintf("http-direct-%d", httpPort),
		fmt.Sprintf("tcp-relay-%d", httpPort),
	}

	for _, name := range serviceNames {
		svc, err := gostapi.GetService(host, name)
		if err != nil {
			continue
		}

		publicAddr := fmt.Sprintf(":%d", httpPort)
		if svc.Addr == publicAddr {
			return nil // 已经是公网地址
		}

		svc.Addr = publicAddr
		_, err = gostapi.UpdateService(host, name, svc)
		if err != nil {
			return fmt.Errorf("恢复 GOST 服务地址失败: %w", err)
		}

		_, _ = gostapi.SaveConfig(host, "yaml", "")
		logx.Infof("GOST HTTP 服务已恢复为公网: %s → %s", name, publicAddr)
		return nil
	}

	return nil // 找不到服务不报错（可能已被删除）
}

// RemoveNginxCacheForPort 移除指定端口的 Nginx 缓存配置
func RemoveNginxCacheForPort(serverId int, httpPort int) error {
	client, err := GetSSHClient(serverId)
	if err != nil {
		return fmt.Errorf("获取 SSH 连接失败: %w", err)
	}
	defer client.Close()

	confPath := fmt.Sprintf("/etc/nginx/conf.d/gost-port-%d.conf", httpPort)
	_, _ = client.ExecuteCommand(fmt.Sprintf("sudo rm -f %s", confPath))
	_, _ = client.ExecuteCommand("sudo nginx -t && sudo systemctl reload nginx 2>/dev/null")

	// 恢复 GOST 服务为公网监听
	_ = RestoreGostServiceToPublic(serverId, httpPort)

	return nil
}

// RemoveAllNginxCacheConfigs 移除服务器上所有 GOST 端口的 Nginx 缓存配置
func RemoveAllNginxCacheConfigs(serverId int) error {
	client, err := GetSSHClient(serverId)
	if err != nil {
		return fmt.Errorf("获取 SSH 连接失败: %w", err)
	}
	defer client.Close()

	// 删除所有 gost-port-*.conf 文件
	_, _ = client.ExecuteCommand("sudo rm -f /etc/nginx/conf.d/gost-port-*.conf")
	_, _ = client.ExecuteCommand("sudo nginx -t && sudo systemctl reload nginx 2>/dev/null")

	return nil
}

// ClearNginxCache 清除 Nginx 缓存
func ClearNginxCache(serverId int) error {
	client, err := GetSSHClient(serverId)
	if err != nil {
		return fmt.Errorf("获取 SSH 连接失败: %w", err)
	}
	defer client.Close()

	// 清空缓存目录
	_, err = client.ExecuteCommand(fmt.Sprintf("sudo rm -rf %s/* && sudo nginx -s reload", NginxCacheDir))
	if err != nil {
		return fmt.Errorf("清除缓存失败: %w", err)
	}

	logx.Infof("Nginx 缓存已清除: 服务器 %d", serverId)
	return nil
}

// GetNginxCacheStatus 获取 Nginx 缓存状态
func GetNginxCacheStatus(serverId int) (*model.NginxCacheStatusResp, error) {
	client, err := GetSSHClient(serverId)
	if err != nil {
		return nil, fmt.Errorf("获取 SSH 连接失败: %w", err)
	}
	defer client.Close()

	resp := &model.NginxCacheStatusResp{}

	// 检查是否安装
	output, _ := client.ExecuteCommand("which nginx")
	resp.Installed = strings.TrimSpace(output) != ""

	if !resp.Installed {
		return resp, nil
	}

	// 检查是否运行
	output, _ = client.ExecuteCommand("systemctl is-active nginx 2>/dev/null")
	resp.Running = strings.TrimSpace(output) == "active"

	// 获取缓存目录大小
	output, _ = client.ExecuteCommand(fmt.Sprintf("du -sh %s 2>/dev/null | awk '{print $1}'", NginxCacheDir))
	resp.CacheSize = strings.TrimSpace(output)
	if resp.CacheSize == "" {
		resp.CacheSize = "0"
	}

	return resp, nil
}

// isNginxInstalled 检查服务器是否安装了 Nginx
func isNginxInstalled(serverId int) bool {
	client, err := GetSSHClient(serverId)
	if err != nil {
		return false
	}
	defer client.Close()

	output, _ := client.ExecuteCommand("which nginx")
	return strings.TrimSpace(output) != ""
}

// identifyHttpPort 从端口列表中识别 HTTP 端口
// 如果提供了自定义端口，取第 3 个（offset+2 是 HTTP）
// 如果没有提供自定义端口，使用默认的 ForwardPorts 中的第 3 个（10012）
func identifyHttpPort(ports []int) int {
	if len(ports) >= 3 {
		return ports[2] // 第 3 个端口是 HTTP（offset+2）
	}
	if len(ports) == 0 {
		// 使用默认端口
		if len(gostapi.ForwardPorts) >= 3 {
			return gostapi.ForwardPorts[2] // 10012
		}
	}
	return 0
}
