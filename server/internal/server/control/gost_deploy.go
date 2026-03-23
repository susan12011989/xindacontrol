package control

import (
	"context"
	"fmt"
	"os"
	"server/pkg/gostapi"
	"strings"
	"time"
)

// GostDeployRequest 一键部署请求
// 单机模式: 无需任何参数，全自动
// 多机模式: 需要 ServerId + MerchantIds + ForwardType
type GostDeployRequest struct {
	ServerId    int   `json:"server_id"`    // 多机模式必填
	MerchantIds []int `json:"merchant_ids"` // 多机模式必填
	ForwardType int   `json:"forward_type"` // 多机模式：1-加密 2-直连，单机自动选直连
}

// GostOneClickDeploy 统一一键部署入口
// 单机模式: 自动安装本地 GOST + 配置直连端口映射，零参数
// 多机模式: 委托给现有 SetupGostDeploy 流程
func GostOneClickDeploy(ctx context.Context, req GostDeployRequest, progress func(string)) error {
	if IsLocalMode() {
		return localGostDeploy(ctx, progress)
	}
	// 多机模式保持原有逻辑（由 deploy service 处理）
	return fmt.Errorf("多机模式请使用 deploy/gost/setup 接口")
}

// localGostDeploy 单机一键部署 GOST
// 全自动：检测安装 → 配置端口映射 → 持久化
func localGostDeploy(ctx context.Context, progress func(string)) error {
	executor := NewLocalExecutor()

	progress("========== 单机模式 GOST 一键部署 ==========")

	// Step 1: 检测 GOST 是否已安装
	progress("步骤 1/5: 检测 GOST 安装状态...")
	result := executor.Execute(ctx, "which gost 2>/dev/null || echo ''")
	gostPath := strings.TrimSpace(result.Output)

	if gostPath != "" {
		progress(fmt.Sprintf("GOST 已安装: %s", gostPath))
	} else {
		// 尝试使用预存二进制
		progress("GOST 未安装，开始安装...")
		if err := installGostLocally(ctx, executor, progress); err != nil {
			return fmt.Errorf("安装 GOST 失败: %w", err)
		}
		progress("GOST 安装完成")
	}

	// Step 2: 确保 GOST 服务运行
	progress("步骤 2/5: 确保 GOST 服务运行...")
	result = executor.Execute(ctx, "systemctl is-active gost 2>/dev/null || echo 'inactive'")
	if strings.TrimSpace(result.Output) != "active" {
		executor.Execute(ctx, "sudo systemctl start gost 2>/dev/null || true")
		time.Sleep(2 * time.Second)
	}

	// 验证 API
	_, err := gostapi.GetConfig("127.0.0.1", "")
	if err != nil {
		progress(fmt.Sprintf("警告: GOST API 未响应: %s，尝试重启...", err))
		executor.Execute(ctx, "sudo systemctl restart gost")
		time.Sleep(3 * time.Second)
		if _, err := gostapi.GetConfig("127.0.0.1", ""); err != nil {
			return fmt.Errorf("GOST API 不可用: %w", err)
		}
	}
	progress("GOST API 正常")

	// Step 3: 配置 GOST 本地转发（V2 精简架构）
	progress("步骤 3/5: 配置 GOST 本地转发...")
	progress(fmt.Sprintf("  %d(统一入口) → nginx:%d (路径分发 WS/HTTP/S3)", gostapi.MerchantUnifiedPort, gostapi.MerchantNginxPort))
	progress(fmt.Sprintf("  %d(TCP)      → 127.0.0.1:%d (WuKongIM TCP)", gostapi.MerchantTCPPort, gostapi.MerchantAppPortTCP))

	if err := gostapi.CreateMerchantLocalForwards("127.0.0.1"); err != nil {
		progress(fmt.Sprintf("警告: 本地转发配置失败: %s (可能已存在)", err))
	} else {
		progress("GOST 本地转发配置成功")
	}

	// Step 4: 部署 nginx 路径分发
	progress("步骤 4/5: 配置 nginx 路径分发...")
	nginxConf := gostapi.MerchantNginxConfigTemplate()
	for _, nc := range gostapi.MerchantNginxConfigs {
		progress(fmt.Sprintf("  %s → 127.0.0.1:%d (%s)", nc.Path, nc.AppPort, nc.Name))
	}

	// 写入 nginx 配置并启动
	nginxSetupCmd := fmt.Sprintf(`
sudo mkdir -p /opt/tsdd/nginx
cat > /tmp/tsdd-nginx.conf << 'NGINX_EOF'
%s
NGINX_EOF
sudo mv /tmp/tsdd-nginx.conf /opt/tsdd/nginx/default.conf

# 启动 nginx 容器
sudo docker rm -f tsdd-nginx 2>/dev/null || true
sudo docker run -d \
    --name tsdd-nginx \
    --restart always \
    --network host \
    -v /opt/tsdd/nginx/default.conf:/etc/nginx/conf.d/default.conf:ro \
    nginx:alpine
`, nginxConf)

	result = executor.Execute(ctx, nginxSetupCmd)
	if result.Err != nil {
		progress(fmt.Sprintf("警告: nginx 部署失败: %s", result.Err))
		progress("  请确保 Docker 已安装")
	} else {
		progress("nginx 路径分发配置成功")
	}

	// Step 5: 持久化配置
	progress("步骤 5/5: 保存配置...")
	gostCtrl := newLocalGostController(executor)
	if err := gostCtrl.PersistGostConfig(ctx); err != nil {
		progress(fmt.Sprintf("警告: 配置持久化失败: %s", err))
	} else {
		progress("配置已保存到 /etc/gost/config.yaml")
	}

	progress("========================================")
	progress("GOST 一键部署完成! (V2 精简端口架构)")
	progress(fmt.Sprintf("  GOST API:  http://127.0.0.1:%d", gostapi.GostAPIPortInt))
	progress(fmt.Sprintf("  统一入口:  :%d (relay+tls → nginx 路径分发)", gostapi.MerchantUnifiedPort))
	progress(fmt.Sprintf("  TCP 长连接: :%d (relay+tls → WuKongIM)", gostapi.MerchantTCPPort))
	progress(fmt.Sprintf("  nginx:     :%d (/ws→%d, /api/→%d, /s3/→%d)",
		gostapi.MerchantNginxPort, gostapi.MerchantAppPortWS, gostapi.MerchantAppPortHTTP, gostapi.MerchantAppPortMinIO))
	progress("========================================")

	return nil
}

// installGostLocally 在本机安装 GOST
func installGostLocally(ctx context.Context, executor *LocalExecutor, progress func(string)) error {
	// 优先使用预存二进制
	gostBinaryPath := "/opt/control/assets/gost"
	if _, err := os.Stat(gostBinaryPath); err == nil {
		progress("使用预存的 GOST 二进制...")
		result := executor.Execute(ctx, fmt.Sprintf("sudo cp '%s' /usr/local/bin/gost && sudo chmod +x /usr/local/bin/gost", gostBinaryPath))
		if result.Err != nil {
			return fmt.Errorf("复制二进制失败: %v", result.Err)
		}
	} else {
		// 从 GitHub 下载
		progress("从 GitHub 下载 GOST...")
		downloadCmd := `
GOST_VERSION="3.0.0-rc10"
cd /tmp
wget -q --timeout=60 "https://github.com/go-gost/gost/releases/download/v${GOST_VERSION}/gost_${GOST_VERSION}_linux_amd64.tar.gz" -O gost.tar.gz
tar -xzf gost.tar.gz
sudo mv gost /usr/local/bin/ && sudo chmod +x /usr/local/bin/gost
rm -f gost.tar.gz
`
		result := executor.Execute(ctx, downloadCmd)
		if result.Err != nil {
			return fmt.Errorf("下载安装失败: %v\n%s", result.Err, result.Output)
		}
	}

	// 创建配置和 systemd 服务
	progress("创建 GOST 配置和服务...")
	setupCmd := fmt.Sprintf(`
sudo mkdir -p /etc/gost /var/log/gost

sudo tee /etc/gost/config.yaml > /dev/null << EOF
api:
  addr: ":%d"
  auth:
    username: %s
    password: %s
  pathPrefix: ""
  accesslog: false
log:
  level: info
  format: json
  output: /var/log/gost/gost.log
services: []
chains: []
EOF

sudo tee /etc/systemd/system/gost.service > /dev/null << EOF
[Unit]
Description=GOST
After=network.target
[Service]
Type=simple
ExecStart=/usr/local/bin/gost -C /etc/gost/config.yaml
Restart=always
LimitNOFILE=1048576
[Install]
WantedBy=multi-user.target
EOF

sudo systemctl daemon-reload
sudo systemctl enable gost --now

# 配置 logrotate 防止日志撑满磁盘
sudo tee /etc/logrotate.d/gost > /dev/null << EOF
/var/log/gost/gost.log {
    daily
    rotate 7
    compress
    delaycompress
    missingok
    notifempty
    size 50M
    copytruncate
}
EOF
`, 9394, gostapi.GostAPIUsername, gostapi.GostAPIPassword)

	result := executor.Execute(ctx, setupCmd)
	if result.Err != nil {
		return fmt.Errorf("配置服务失败: %v", result.Err)
	}

	// 等待服务启动
	time.Sleep(2 * time.Second)
	return nil
}
