package deploy

import (
	"fmt"
	"net"
	"server/internal/server/cloud/aliyun"
	"server/internal/server/model"
	"server/pkg/dbs"
	"server/pkg/entity"
	"server/pkg/gostapi"
	"strings"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
	"golang.org/x/crypto/ssh"
)

// GostDeployConfig GOST 部署配置
type GostDeployConfig struct {
	CloudAccountId int64  `json:"cloud_account_id"` // 云账号ID
	RegionId       string `json:"region_id"`        // 地区ID，如 ap-southeast-1
	InstanceType   string `json:"instance_type"`    // 实例类型，如 ecs.t6-c1m1.large
	ImageId        string `json:"image_id"`         // 镜像ID，为空则使用默认Ubuntu镜像
	ServerName     string `json:"server_name"`      // 服务器名称
	GroupId        int    `json:"group_id"`         // 服务器分组ID
	Password       string `json:"password"`         // SSH 密码（可选，不填自动生成密钥）
	Bandwidth      string `json:"bandwidth"`        // EIP 带宽，默认 5 Mbps
	ForwardType    int    `json:"forward_type"`     // 转发类型: 1-加密(默认) 2-直连
}

// GostDeployResult 部署结果
type GostDeployResult struct {
	InstanceId string `json:"instance_id"`
	PublicIP   string `json:"public_ip"`
	ServerId   int    `json:"server_id"`
	GostPort   int    `json:"gost_port"`
}

// 默认配置
const (
	DefaultInstanceType = "ecs.t6-c1m1.large" // 2核2G，适合转发
	DefaultBandwidth    = "5"
	DefaultGostAPIPort  = 9394
	GostAPIUser         = "tsdd"
	GostAPIPass         = "Oa21isSdaiuwhq"
)

// 默认 Ubuntu 镜像（按地区）
var defaultUbuntuImages = map[string]string{
	"cn-hangzhou":     "ubuntu_22_04_x64_20G_alibase_20240101.vhd",
	"cn-shanghai":     "ubuntu_22_04_x64_20G_alibase_20240101.vhd",
	"cn-shenzhen":     "ubuntu_22_04_x64_20G_alibase_20240101.vhd",
	"cn-hongkong":     "ubuntu_22_04_x64_20G_alibase_20240101.vhd",
	"ap-southeast-1":  "ubuntu_22_04_x64_20G_alibase_20240101.vhd", // 新加坡
	"ap-southeast-3":  "ubuntu_22_04_x64_20G_alibase_20240101.vhd", // 吉隆坡
	"ap-southeast-5":  "ubuntu_22_04_x64_20G_alibase_20240101.vhd", // 雅加达
	"us-west-1":       "ubuntu_22_04_x64_20G_alibase_20240101.vhd", // 硅谷
	"eu-central-1":    "ubuntu_22_04_x64_20G_alibase_20240101.vhd", // 法兰克福
}

// DeployGostServer 一键部署 GOST 转发服务器
func DeployGostServer(config *GostDeployConfig, progressCallback func(message string)) (*GostDeployResult, error) {
	result := &GostDeployResult{
		GostPort: DefaultGostAPIPort,
	}

	// 填充默认值
	if config.InstanceType == "" {
		config.InstanceType = DefaultInstanceType
	}
	if config.Bandwidth == "" {
		config.Bandwidth = DefaultBandwidth
	}
	if config.ImageId == "" {
		if img, ok := defaultUbuntuImages[config.RegionId]; ok {
			config.ImageId = img
		} else {
			// 使用通用的 Ubuntu 22.04 镜像别名
			config.ImageId = "ubuntu_22_04_x64_20G_alibase_20240101.vhd"
		}
	}
	if config.ServerName == "" {
		config.ServerName = fmt.Sprintf("gost-%s-%d", config.RegionId, time.Now().Unix())
	}

	progressCallback(fmt.Sprintf("开始部署 GOST 服务器: %s", config.ServerName))
	progressCallback(fmt.Sprintf("地区: %s, 规格: %s", config.RegionId, config.InstanceType))

	// Step 1: 创建 ECS 实例
	progressCallback("步骤 1/7: 创建 ECS 实例...")
	createReq := &aliyun.CreateInstanceRequest{
		CloudAccountId:     config.CloudAccountId,
		Region:             config.RegionId,
		InstanceType:       config.InstanceType,
		ImageId:            config.ImageId,
		InstanceChargeType: "PostPaid", // 按量付费
		DiskCategory:       "cloud_efficiency",
		DiskSize:           40, // 40GB 系统盘
		Password:           config.Password,
		UsePassword:        true, // GOST 部署必须使用密码认证，否则密钥对私钥不会保存
	}

	instanceResult, err := aliyun.CreateInstance(createReq)
	if err != nil {
		return nil, fmt.Errorf("创建 ECS 实例失败: %w", err)
	}
	result.InstanceId = instanceResult.InstanceId
	// 使用 CreateInstance 返回的密码（用户未提供时会自动生成）
	if instanceResult.Password != "" {
		config.Password = instanceResult.Password
	}
	progressCallback(fmt.Sprintf("ECS 实例创建成功: %s", result.InstanceId))

	// Step 2: 等待实例运行
	progressCallback("步骤 2/7: 等待实例启动...")
	err = waitForInstanceRunning(config.CloudAccountId, config.RegionId, result.InstanceId, progressCallback)
	if err != nil {
		return nil, fmt.Errorf("等待实例启动失败: %w", err)
	}
	progressCallback("实例已启动运行")

	// Step 3: 创建并绑定 EIP
	progressCallback("步骤 3/7: 创建弹性公网IP...")
	allocateReq := &aliyun.AllocateEipAddressRequest{
		CloudAccountId:     config.CloudAccountId,
		RegionId:           config.RegionId,
		Bandwidth:          config.Bandwidth,
		InternetChargeType: "PayByTraffic",
		InstanceChargeType: "PostPaid",
	}

	eipId, publicIP, err := aliyun.AllocateEipAddress(allocateReq)
	if err != nil {
		return nil, fmt.Errorf("创建 EIP 失败: %w", err)
	}
	result.PublicIP = publicIP
	progressCallback(fmt.Sprintf("EIP 创建成功: %s (%s)", eipId, publicIP))

	// Step 4: 绑定 EIP 到实例
	progressCallback("步骤 4/7: 绑定 EIP 到实例...")
	time.Sleep(3 * time.Second) // 等待 EIP 就绪

	associateReq := &aliyun.AssociateEipAddressRequest{
		CloudAccountId: config.CloudAccountId,
		Region:         config.RegionId,
		AllocationId:   eipId,
		InstanceId:     result.InstanceId,
		InstanceType:   "EcsInstance",
	}

	err = aliyun.AssociateEipAddress(associateReq)
	if err != nil {
		return nil, fmt.Errorf("绑定 EIP 失败: %w", err)
	}
	progressCallback("EIP 绑定成功")

	// Step 5: 等待 SSH 可连接
	progressCallback("步骤 5/7: 等待 SSH 服务就绪...")
	err = waitForSSH(publicIP, config.Password, progressCallback)
	if err != nil {
		return nil, fmt.Errorf("SSH 连接失败: %w", err)
	}
	progressCallback("SSH 连接成功")

	// Step 6: 执行 GOST 安装脚本
	progressCallback("步骤 6/7: 安装 GOST 服务...")
	err = installGostViaSSH(publicIP, config.Password, progressCallback)
	if err != nil {
		return nil, fmt.Errorf("安装 GOST 失败: %w", err)
	}
	progressCallback("GOST 安装成功")

	// Step 7: 验证 GOST API 并注册到数据库
	progressCallback("步骤 7/7: 验证并注册服务器...")
	time.Sleep(3 * time.Second) // 等待 GOST 服务启动

	// 验证 GOST API
	err = verifyGostAPI(publicIP)
	if err != nil {
		progressCallback(fmt.Sprintf("警告: GOST API 验证失败: %s，但继续注册", err))
	} else {
		progressCallback("GOST API 验证成功")
	}

	// 转发类型默认为加密
	forwardType := config.ForwardType
	if forwardType == 0 {
		forwardType = entity.ForwardTypeEncrypted
	}

	// 注册到数据库
	server := &entity.Servers{
		Name:        config.ServerName,
		Host:        publicIP,
		Port:        22,
		Username:    "root",
		AuthType:    1, // 密码认证
		Password:    config.Password,
		ServerType:  2, // 系统服务器
		ForwardType: forwardType,
		GroupId:     config.GroupId,
		Status:      1,
		Description: fmt.Sprintf("GOST转发服务器 %s (实例:%s)", config.RegionId, result.InstanceId),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	_, err = dbs.DBAdmin.Insert(server)
	if err != nil {
		return nil, fmt.Errorf("注册服务器失败: %w", err)
	}
	result.ServerId = server.Id
	progressCallback(fmt.Sprintf("服务器注册成功，ID: %d", result.ServerId))

	// 自动部署 TLS 证书（如果已生成）
	tlsDeployed := deployTlsCertsIfAvailable(*server, progressCallback)

	// 为所有商户创建 GOST 转发规则
	forwardTypeName := "加密(relay+tls)"
	if forwardType == entity.ForwardTypeDirect {
		forwardTypeName = "直连(tcp)"
	}
	if tlsDeployed {
		forwardTypeName += "+tls-listener"
	}
	progressCallback(fmt.Sprintf("配置商户转发规则 (模式: %s)...", forwardTypeName))
	go enqueueGostServicesForMerchants(publicIP, forwardType)
	progressCallback("商户转发规则已入队，后台异步创建中")

	progressCallback("========================================")
	progressCallback(fmt.Sprintf("✓ GOST 服务器部署完成!"))
	progressCallback(fmt.Sprintf("  公网IP: %s", publicIP))
	progressCallback(fmt.Sprintf("  GOST API: http://%s:%d", publicIP, DefaultGostAPIPort))
	progressCallback(fmt.Sprintf("  服务器ID: %d", result.ServerId))
	progressCallback(fmt.Sprintf("  转发类型: %s", forwardTypeName))
	progressCallback("========================================")

	return result, nil
}

// waitForInstanceRunning 等待实例运行
func waitForInstanceRunning(cloudAccountId int64, regionId, instanceId string, progressCallback func(string)) error {
	maxRetries := 60
	for i := 0; i < maxRetries; i++ {
		instances, err := aliyun.DescribeInstancesByCloudAccount(cloudAccountId, regionId)
		if err != nil {
			logx.Errorf("查询实例状态失败: %v", err)
			time.Sleep(5 * time.Second)
			continue
		}

		for _, inst := range instances {
			if inst.InstanceId != nil && *inst.InstanceId == instanceId {
				status := ""
				if inst.Status != nil {
					status = *inst.Status
				}
				progressCallback(fmt.Sprintf("实例状态: %s", status))
				if status == "Running" {
					return nil
				}
			}
		}

		time.Sleep(5 * time.Second)
	}
	return fmt.Errorf("等待实例运行超时")
}

// waitForSSH 等待 SSH 可连接
func waitForSSH(host, password string, progressCallback func(string)) error {
	maxRetries := 30
	for i := 0; i < maxRetries; i++ {
		progressCallback(fmt.Sprintf("尝试 SSH 连接... (%d/%d)", i+1, maxRetries))

		conn, err := net.DialTimeout("tcp", host+":22", 5*time.Second)
		if err == nil {
			conn.Close()
			// 再等待几秒确保 SSH 服务完全就绪
			time.Sleep(5 * time.Second)
			return nil
		}

		time.Sleep(10 * time.Second)
	}
	return fmt.Errorf("SSH 连接超时")
}

// installGostViaSSHClient 通过已有 SSH 客户端安装 GOST
func installGostViaSSHClient(client *ssh.Client, progressCallback func(string)) error {
	// GOST 安装脚本 - 支持 root 和非 root 用户（自动使用 sudo）
	installScript := fmt.Sprintf(`#!/bin/bash
set -e

GOST_VERSION="3.0.0-rc10"
API_USER="%s"
API_PASS="%s"
API_PORT="%d"

# 检测是否需要 sudo
if [ "$(id -u)" -eq 0 ]; then
    SUDO=""
else
    SUDO="sudo"
fi

echo ">>> 下载 GOST..."
cd /tmp
GOST_FILE="gost_${GOST_VERSION}_linux_amd64.tar.gz"
# 多源下载：镜像优先，GitHub 备用
URLS=(
    "https://ghfast.top/https://github.com/go-gost/gost/releases/download/v${GOST_VERSION}/${GOST_FILE}"
    "https://gh-proxy.com/https://github.com/go-gost/gost/releases/download/v${GOST_VERSION}/${GOST_FILE}"
    "https://github.com/go-gost/gost/releases/download/v${GOST_VERSION}/${GOST_FILE}"
)
DOWNLOADED=0
for URL in "${URLS[@]}"; do
    echo ">>> 尝试下载: ${URL}"
    if command -v wget >/dev/null 2>&1; then
        wget -q --timeout=30 --tries=2 "$URL" -O gost.tar.gz && DOWNLOADED=1 && break
    elif command -v curl >/dev/null 2>&1; then
        curl -sL --connect-timeout 30 --max-time 120 --retry 2 "$URL" -o gost.tar.gz && DOWNLOADED=1 && break
    fi
    echo ">>> 下载失败，尝试下一个源..."
    rm -f gost.tar.gz
done
if [ "$DOWNLOADED" -ne 1 ]; then
    echo "ERROR: 所有下载源均失败" >&2; exit 1
fi
# 验证下载文件
if [ ! -s gost.tar.gz ]; then
    echo "ERROR: 下载文件为空" >&2; exit 1
fi
tar -xzf gost.tar.gz
$SUDO mv gost /usr/local/bin/
$SUDO chmod +x /usr/local/bin/gost
rm -f gost.tar.gz

echo ">>> 验证 GOST 版本..."
INSTALLED_VERSION=$(/usr/local/bin/gost -V 2>&1 | grep -oP '\d+\.\d+\.\d+-rc\d+' || echo "unknown")
echo ">>> GOST 版本: ${INSTALLED_VERSION} (期望: ${GOST_VERSION})"

echo ">>> 创建配置..."
$SUDO mkdir -p /etc/gost /var/log/gost
$SUDO tee /etc/gost/config.yaml > /dev/null << EOF
api:
  addr: ":${API_PORT}"
  auth:
    username: ${API_USER}
    password: ${API_PASS}
  pathPrefix: ""
  accesslog: false

log:
  level: info
  format: json
  output: /var/log/gost/gost.log

services: []
chains: []
EOF

echo ">>> 创建服务..."
$SUDO tee /etc/systemd/system/gost.service > /dev/null << EOF
[Unit]
Description=GOST Proxy
After=network.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=/usr/local/bin/gost -C /etc/gost/config.yaml
Restart=always
RestartSec=5
LimitNOFILE=1048576
StandardOutput=append:/var/log/gost/gost-stdout.log
StandardError=append:/var/log/gost/gost-stderr.log

[Install]
WantedBy=multi-user.target
EOF

echo ">>> 配置日志轮转..."
$SUDO tee /etc/logrotate.d/gost > /dev/null << 'LOGROTATE'
/var/log/gost/*.log {
    daily
    rotate 7
    compress
    delaycompress
    missingok
    notifempty
    copytruncate
}
LOGROTATE

echo ">>> 启动服务..."
$SUDO systemctl daemon-reload
$SUDO systemctl enable gost --now

echo ">>> 优化网络..."
SYSCTL_CONF="/etc/sysctl.conf"
SYSCTL_MARKER="# GOST network optimization"
if ! grep -q "$SYSCTL_MARKER" "$SYSCTL_CONF" 2>/dev/null; then
    $SUDO tee -a "$SYSCTL_CONF" > /dev/null << SYSCTL

$SYSCTL_MARKER
net.core.somaxconn=65535
net.ipv4.tcp_max_syn_backlog=65535
net.ipv4.ip_local_port_range=1024 65535
net.ipv4.tcp_tw_reuse=1
fs.file-max=1048576
SYSCTL
    echo ">>> 网络参数已写入"
else
    echo ">>> 网络参数已存在，跳过"
fi
$SUDO sysctl -p > /dev/null 2>&1 || true

echo ">>> GOST 安装完成!"
`, GostAPIUser, GostAPIPass, DefaultGostAPIPort)

	// 创建会话并执行
	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("创建 SSH 会话失败: %w", err)
	}
	defer session.Close()

	// 执行脚本
	output, err := session.CombinedOutput(installScript)
	if err != nil {
		progressCallback(fmt.Sprintf("安装输出: %s", string(output)))
		return fmt.Errorf("执行安装脚本失败: %w", err)
	}

	// 解析输出
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, ">>>") {
			progressCallback(line)
		}
	}

	// GOST 安装完成后，安装 Nginx 缓存服务（失败不影响 GOST）
	progressCallback(">>> 安装 Nginx 缓存服务...")
	if err := installNginxViaSSH(client); err != nil {
		progressCallback(fmt.Sprintf("警告: Nginx 安装失败: %s (缓存功能不可用)", err))
	} else {
		progressCallback(">>> Nginx 缓存服务安装成功")
	}

	return nil
}

// installGostViaSSH 通过 SSH 安装 GOST（兼容旧接口，使用 root 用户）
func installGostViaSSH(host, password string, progressCallback func(string)) error {
	// SSH 配置
	config := &ssh.ClientConfig{
		User: "root",
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         30 * time.Second,
	}

	// 连接 SSH
	client, err := ssh.Dial("tcp", host+":22", config)
	if err != nil {
		return fmt.Errorf("SSH 连接失败: %w", err)
	}
	defer client.Close()

	return installGostViaSSHClient(client, progressCallback)
}

// deployTlsCertsIfAvailable 如果数据库中有该服务器关联商户的有效 TLS 证书，自动推送到服务器
// 先查 merchant_gost_servers 关联表，再查 servers.merchant_id，找到商户后按 merchant_id 查证书
// 返回 true 表示已部署证书
func deployTlsCertsIfAvailable(server entity.Servers, progressCallback func(string)) bool {
	// 查找关联的商户 ID：优先关联表，其次 servers.merchant_id
	var merchantId int
	var relation entity.MerchantGostServers
	has, err := dbs.DBAdmin.Where("server_id = ? AND status = 1", server.Id).Get(&relation)
	if err == nil && has {
		merchantId = relation.MerchantId
	} else if server.MerchantId > 0 {
		merchantId = server.MerchantId
	} else {
		return false
	}

	var caCert entity.TlsCertificates
	has, err = dbs.DBAdmin.Where("name = 'gost-ca' AND merchant_id = ? AND status = 1", merchantId).Get(&caCert)
	if err != nil || !has {
		return false
	}

	var serverCert entity.TlsCertificates
	has, err = dbs.DBAdmin.Where("name = 'gost-server' AND merchant_id = ? AND status = 1", merchantId).Get(&serverCert)
	if err != nil || !has {
		return false
	}

	progressCallback(fmt.Sprintf("检测到商户(ID:%d)的有效 TLS 证书，自动部署...", merchantId))
	err = pushCertsToServer(server, caCert, serverCert)
	if err != nil {
		progressCallback(fmt.Sprintf("警告: TLS 证书部署失败: %s", err))
		return false
	}

	// 更新数据库 TLS 状态
	now := time.Now()
	_, err = dbs.DBAdmin.Where("id = ?", server.Id).Cols("tls_enabled", "tls_deployed_at", "updated_at").Update(&entity.Servers{
		TlsEnabled:    1,
		TlsDeployedAt: &now,
		UpdatedAt:     now,
	})
	if err != nil {
		progressCallback(fmt.Sprintf("警告: 更新 TLS 状态失败: %s", err))
	}

	progressCallback("TLS 证书部署成功")
	return true
}

// SetupGostDeploy 一键部署 GOST：安装（如需）+ TLS 证书 + 商户转发配置
func SetupGostDeploy(req *model.SetupGostDeployReq, progressCallback func(string)) error {
	// 1. 获取服务器信息
	var server entity.Servers
	has, err := dbs.DBAdmin.Where("id = ?", req.ServerId).Get(&server)
	if err != nil {
		return fmt.Errorf("查询服务器失败: %w", err)
	}
	if !has {
		return fmt.Errorf("服务器不存在: %d", req.ServerId)
	}
	if server.ServerType != 2 {
		return fmt.Errorf("只能对系统服务器执行此操作")
	}

	host := server.Host
	forwardType := req.ForwardType
	if forwardType == 0 {
		forwardType = entity.ForwardTypeEncrypted
	}

	progressCallback(fmt.Sprintf("目标服务器: %s (%s)", server.Name, host))

	// 2. 检测 GOST 是否已安装
	progressCallback("检测 GOST 安装状态...")
	sshClient, err := GetSSHClient(req.ServerId)
	if err != nil {
		return fmt.Errorf("SSH 连接失败: %w", err)
	}
	defer sshClient.Close()

	output, _ := sshClient.ExecuteCommand("which gost")
	gostInstalled := strings.TrimSpace(output) != ""

	if gostInstalled {
		progressCallback("GOST 已安装，跳过安装步骤")
	} else {
		// 3. 安装 GOST
		progressCallback("GOST 未安装，开始安装...")
		err = installGostViaSSHClient(sshClient.Client, progressCallback)
		if err != nil {
			return fmt.Errorf("安装 GOST 失败: %w", err)
		}
		progressCallback("GOST 安装完成")
	}

	// 4. 验证 GOST API
	progressCallback("验证 GOST API...")
	time.Sleep(2 * time.Second)
	if err := verifyGostAPI(host); err != nil {
		progressCallback(fmt.Sprintf("警告: GOST API 验证失败: %s", err))
	} else {
		progressCallback("GOST API 验证成功")
	}

	// 5. 部署 TLS 证书
	tlsDeployed := deployTlsCertsIfAvailable(server, progressCallback)

	// 6. 查询要配置的商户
	var merchants []entity.Merchants
	err = dbs.DBAdmin.In("id", req.MerchantIds).Where("status = 1").Find(&merchants)
	if err != nil {
		return fmt.Errorf("查询商户失败: %w", err)
	}
	if len(merchants) == 0 {
		return fmt.Errorf("未找到有效商户")
	}

	// 7. 逐个商户配置转发
	forwardTypeName := "加密(relay+tls)"
	if forwardType == entity.ForwardTypeDirect {
		forwardTypeName = "直连(tcp)"
	}

	successCount := 0
	failCount := 0

	for _, m := range merchants {
		progressCallback(fmt.Sprintf("配置商户 [%s] 转发 (端口: %d, 目标: %s, 模式: %s)...",
			m.Name, m.Port, m.ServerIP, forwardTypeName))

		if m.ServerIP == "" {
			progressCallback(fmt.Sprintf("  跳过: 商户 %s 未配置服务器IP", m.Name))
			failCount++
			continue
		}
		if m.Port == 0 {
			progressCallback(fmt.Sprintf("  跳过: 商户 %s 未配置端口", m.Name))
			failCount++
			continue
		}

		var forwardErr error
		if forwardType == entity.ForwardTypeDirect {
			if tlsDeployed || server.TlsEnabled == 1 {
				forwardErr = gostapi.CreateMerchantDirectForwardsWithTls(host, m.Port, m.ServerIP)
			} else {
				forwardErr = gostapi.CreateMerchantDirectForwards(host, m.Port, m.ServerIP)
			}
		} else {
			if tlsDeployed || server.TlsEnabled == 1 {
				forwardErr = gostapi.CreateMerchantForwardsWithTls(host, m.Port, m.ServerIP)
			} else {
				forwardErr = gostapi.CreateMerchantForwards(host, m.Port, m.ServerIP)
			}
		}

		if forwardErr != nil {
			progressCallback(fmt.Sprintf("  失败: %s", forwardErr))
			failCount++
		} else {
			progressCallback(fmt.Sprintf("  成功: %s 端口 %d/%d/%d/%d (TCP/WS/HTTP/MinIO)", m.Name, m.Port, m.Port+1, m.Port+2, m.Port+3))
			successCount++

			// 配置 Nginx 缓存（HTTP 端口 = basePort+2）
			httpPort := m.Port + 2
			sslEnabled := tlsDeployed || server.TlsEnabled == 1
			if isNginxInstalled(req.ServerId) {
				if err := UpdateGostServiceToLoopback(req.ServerId, httpPort, sslEnabled); err == nil {
					if err := ConfigureNginxCacheForPort(req.ServerId, httpPort, sslEnabled); err != nil {
						_ = RestoreGostServiceToPublic(req.ServerId, httpPort)
						progressCallback(fmt.Sprintf("  Nginx 缓存配置失败(端口 %d): %s", httpPort, err))
					} else {
						sslLabel := ""
						if sslEnabled {
							sslLabel = "+SSL"
						}
						progressCallback(fmt.Sprintf("  Nginx 缓存已配置(端口 %d%s)", httpPort, sslLabel))
					}
				}
			}
		}
	}

	// 8. 如果启用了 TLS，升级 listener
	if server.TlsEnabled == 1 || tlsDeployed {
		progressCallback("升级 GOST listener 为 TLS...")
		if err := upgradeGostListenerToTls(host); err != nil {
			progressCallback(fmt.Sprintf("警告: TLS listener 升级失败: %s", err))
		} else {
			progressCallback("TLS listener 升级成功")
		}
	}

	// 9. 持久化配置
	progressCallback("保存 GOST 配置到文件...")
	if err := PersistGostConfig(req.ServerId); err != nil {
		progressCallback(fmt.Sprintf("警告: 配置持久化失败: %s", err))
	} else {
		progressCallback("配置已保存")
	}

	// 更新服务器的 ForwardType，并关联第一个成功的商户
	updateServer := entity.Servers{ForwardType: forwardType, UpdatedAt: time.Now()}
	updateCols := []string{"forward_type", "updated_at"}
	if successCount > 0 && len(merchants) > 0 && server.MerchantId == 0 {
		updateServer.MerchantId = merchants[0].Id
		updateCols = append(updateCols, "merchant_id")
	}
	_, _ = dbs.DBAdmin.Where("id = ?", req.ServerId).Cols(updateCols...).Update(&updateServer)

	progressCallback("========================================")
	progressCallback(fmt.Sprintf("部署完成! 成功: %d, 失败: %d", successCount, failCount))
	progressCallback("========================================")

	return nil
}

// verifyGostAPI 验证 GOST API 是否可用
func verifyGostAPI(host string) error {
	// 简单验证：尝试获取配置
	_, err := gostapi.GetConfig(host, "")
	return err
}

// GetGostDefaultConfig 获取 GOST 部署默认配置
func GetGostDefaultConfig(regionId string) map[string]interface{} {
	imageId := ""
	if img, ok := defaultUbuntuImages[regionId]; ok {
		imageId = img
	}

	return map[string]interface{}{
		"instance_type": DefaultInstanceType,
		"image_id":      imageId,
		"bandwidth":     DefaultBandwidth,
		"gost_port":     DefaultGostAPIPort,
	}
}

// InstallGostToExistingServer 在已有服务器上安装 GOST
func InstallGostToExistingServer(req *model.InstallGostReq, progressCallback func(string)) error {
	var host, password, privateKey, username string
	var port int

	// 如果提供了 ServerId，从数据库获取服务器信息
	if req.ServerId > 0 {
		progressCallback(fmt.Sprintf("从数据库获取服务器信息 (ID: %d)...", req.ServerId))
		var server entity.Servers
		has, err := dbs.DBAdmin.ID(req.ServerId).Get(&server)
		if err != nil {
			return fmt.Errorf("查询服务器失败: %w", err)
		}
		if !has {
			return fmt.Errorf("服务器不存在: %d", req.ServerId)
		}
		host = server.Host
		port = server.Port
		username = server.Username
		password = server.Password
		privateKey = server.PrivateKey
		progressCallback(fmt.Sprintf("服务器: %s (%s)", server.Name, host))
	} else {
		host = req.Host
		port = req.Port
		username = req.Username
		password = req.Password
		privateKey = req.PrivateKey
	}

	// 填充默认值
	if port == 0 {
		port = 22
	}
	if username == "" {
		username = "root"
	}

	progressCallback(fmt.Sprintf("连接服务器: %s:%d", host, port))

	// 配置 SSH 认证
	var authMethods []ssh.AuthMethod
	if password != "" {
		authMethods = append(authMethods, ssh.Password(password))
	}
	if privateKey != "" {
		signer, err := ssh.ParsePrivateKey([]byte(privateKey))
		if err != nil {
			return fmt.Errorf("解析私钥失败: %w", err)
		}
		authMethods = append(authMethods, ssh.PublicKeys(signer))
	}

	if len(authMethods) == 0 {
		return fmt.Errorf("必须提供密码或私钥")
	}

	config := &ssh.ClientConfig{
		User:            username,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         30 * time.Second,
	}

	// 连接 SSH
	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", host, port), config)
	if err != nil {
		return fmt.Errorf("SSH 连接失败: %w", err)
	}
	defer client.Close()

	progressCallback("SSH 连接成功，开始安装 GOST...")

	// 执行安装 - 使用已有的 SSH 客户端，支持非 root 用户
	err = installGostViaSSHClient(client, progressCallback)
	if err != nil {
		return err
	}

	// 验证 GOST API
	progressCallback("验证 GOST API...")
	time.Sleep(3 * time.Second)

	err = verifyGostAPI(host)
	if err != nil {
		progressCallback(fmt.Sprintf("警告: GOST API 验证失败: %s", err))
	} else {
		progressCallback("GOST API 验证成功")
	}

	// 配置商户本地转发规则（relay+tls 监听 → 本地业务端口）
	progressCallback("配置商户本地转发规则...")
	progressCallback(fmt.Sprintf("  10010(TCP)  → 127.0.0.1:5001"))
	progressCallback(fmt.Sprintf("  10011(WS)   → 127.0.0.1:5200"))
	progressCallback(fmt.Sprintf("  10012(HTTP) → 127.0.0.1:10002"))

	err = gostapi.CreateMerchantLocalForwards(host)
	if err != nil {
		progressCallback(fmt.Sprintf("警告: 配置本地转发失败: %s", err))
		progressCallback("请稍后手动配置或重试安装")
	} else {
		progressCallback("商户本地转发配置成功")
	}

	// 配置 MinIO 本地转发（10013 → MinIO:9000）
	// MinIO 目标地址因部署模式不同：allinone=127.0.0.1:9000, cluster=MinIO节点内网IP:9000
	minioAddr := "127.0.0.1:9000" // 默认 allinone 模式
	if req.ServerId > 0 {
		// 尝试从 cluster_nodes 查找 MinIO 节点内网 IP
		var serverEntity entity.Servers
		if has, err := dbs.DBAdmin.ID(req.ServerId).Get(&serverEntity); err == nil && has && serverEntity.MerchantId > 0 {
			var minioNode entity.ClusterNodes
			if has, err := dbs.DBAdmin.Where("merchant_id = ? AND node_role = 'minio'", serverEntity.MerchantId).Get(&minioNode); err == nil && has && minioNode.PrivateIP != "" {
				minioAddr = fmt.Sprintf("%s:9000", minioNode.PrivateIP)
				progressCallback(fmt.Sprintf("检测到集群模式，MinIO 节点: %s", minioNode.PrivateIP))
			}
		}
	}
	progressCallback(fmt.Sprintf("  10013(MinIO) → %s", minioAddr))
	if err := gostapi.CreateMinioLocalForward(host, minioAddr); err != nil {
		progressCallback(fmt.Sprintf("警告: MinIO 本地转发配置失败: %s", err))
	} else {
		progressCallback("MinIO 本地转发配置成功")
	}

	// 如果是已注册的系统服务器，自动部署 TLS 证书
	if req.ServerId > 0 {
		var serverEntity entity.Servers
		if has, err := dbs.DBAdmin.ID(req.ServerId).Get(&serverEntity); err == nil && has && serverEntity.ServerType == 2 {
			deployTlsCertsIfAvailable(serverEntity, progressCallback)
		}
	}

	progressCallback(fmt.Sprintf("✓ GOST 安装完成! API: http://%s:%d", host, DefaultGostAPIPort))
	progressCallback("  监听端口: 10010(TCP), 10011(WS), 10012(HTTP), 10013(MinIO) - relay+tls")
	progressCallback(fmt.Sprintf("  转发到: 127.0.0.1:5001, 5200, 10002, %s", minioAddr))

	return nil
}

// DiagnoseAndRepairGost 诊断并修复 GOST 服务
func DiagnoseAndRepairGost(serverId int, progressCallback func(string)) error {
	// 1. 获取服务器信息
	var server entity.Servers
	has, err := dbs.DBAdmin.Where("id = ?", serverId).Get(&server)
	if err != nil {
		return fmt.Errorf("查询服务器失败: %w", err)
	}
	if !has {
		return fmt.Errorf("服务器不存在: %d", serverId)
	}

	host := server.Host
	progressCallback(fmt.Sprintf("诊断服务器: %s (%s)", server.Name, host))

	// 2. SSH 连接
	progressCallback("步骤 1/6: 建立 SSH 连接...")
	sshClient, err := GetSSHClient(serverId)
	if err != nil {
		return fmt.Errorf("SSH 连接失败: %w", err)
	}
	defer sshClient.Close()
	progressCallback("SSH 连接成功")

	// 3. 检查 GOST 二进制
	progressCallback("步骤 2/6: 检查 GOST 二进制...")
	output, _ := sshClient.ExecuteCommand("which gost 2>/dev/null || echo NOT_FOUND")
	gostPath := strings.TrimSpace(output)
	gostInstalled := gostPath != "NOT_FOUND" && gostPath != ""

	if gostInstalled {
		progressCallback(fmt.Sprintf("GOST 二进制已安装: %s", gostPath))
		// 检查版本
		verOutput, _ := sshClient.ExecuteCommand("gost -V 2>&1 || echo unknown")
		progressCallback(fmt.Sprintf("GOST 版本: %s", strings.TrimSpace(verOutput)))
	} else {
		progressCallback("GOST 二进制未找到，需要安装")
	}

	// 4. 检查 systemd 服务
	progressCallback("步骤 3/6: 检查 systemd 服务...")
	statusOutput, _ := sshClient.ExecuteCommand("systemctl is-active gost 2>/dev/null || echo inactive")
	serviceStatus := strings.TrimSpace(statusOutput)
	progressCallback(fmt.Sprintf("GOST 服务状态: %s", serviceStatus))

	// 检查服务文件是否存在
	svcFileOutput, _ := sshClient.ExecuteCommand("test -f /etc/systemd/system/gost.service && echo EXISTS || echo NOT_FOUND")
	svcFileExists := strings.TrimSpace(svcFileOutput) == "EXISTS"
	if svcFileExists {
		progressCallback("systemd 服务文件存在")
	} else {
		progressCallback("systemd 服务文件不存在")
	}

	if serviceStatus != "active" {
		// 检查错误日志
		journalOutput, _ := sshClient.ExecuteCommand("journalctl -u gost -n 10 --no-pager 2>/dev/null || echo '无日志'")
		progressCallback(fmt.Sprintf("最近日志:\n%s", strings.TrimSpace(journalOutput)))
	}

	// 5. 检查配置文件
	progressCallback("步骤 4/6: 检查配置文件...")
	configOutput, _ := sshClient.ExecuteCommand("test -f /etc/gost/config.yaml && echo EXISTS || echo NOT_FOUND")
	configExists := strings.TrimSpace(configOutput) == "EXISTS"
	if configExists {
		progressCallback("配置文件存在: /etc/gost/config.yaml")
	} else {
		progressCallback("配置文件不存在")
	}

	// 6. 检查端口
	progressCallback("步骤 5/6: 检查端口监听...")
	portOutput, _ := sshClient.ExecuteCommand(fmt.Sprintf("ss -tlnp | grep :%d || echo 'PORT_NOT_LISTENING'", DefaultGostAPIPort))
	portListening := !strings.Contains(portOutput, "PORT_NOT_LISTENING")
	if portListening {
		progressCallback(fmt.Sprintf("端口 %d 正在监听", DefaultGostAPIPort))
	} else {
		progressCallback(fmt.Sprintf("端口 %d 未监听", DefaultGostAPIPort))
	}

	// 7. 修复逻辑
	progressCallback("步骤 6/6: 执行修复...")
	needReinstall := !gostInstalled
	needReconfigure := !configExists || !svcFileExists

	if needReinstall {
		// 二进制不存在，完整重装
		progressCallback("开始重新安装 GOST...")
		err = installGostViaSSHClient(sshClient.Client, progressCallback)
		if err != nil {
			return fmt.Errorf("重新安装 GOST 失败: %w", err)
		}
		progressCallback("GOST 重新安装完成")
	} else if needReconfigure {
		// 二进制存在但配置或服务文件缺失，重建配置和服务
		progressCallback("重建配置文件和 systemd 服务...")
		repairScript := fmt.Sprintf(`#!/bin/bash
set -e
if [ "$(id -u)" -eq 0 ]; then SUDO=""; else SUDO="sudo"; fi

# 重建配置文件
$SUDO mkdir -p /etc/gost /var/log/gost
if [ ! -f /etc/gost/config.yaml ]; then
    echo ">>> 创建配置文件..."
    $SUDO tee /etc/gost/config.yaml > /dev/null << EOF
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
    echo ">>> 配置文件已创建"
else
    echo ">>> 配置文件已存在，跳过"
fi

# 重建 systemd 服务文件
if [ ! -f /etc/systemd/system/gost.service ]; then
    echo ">>> 创建 systemd 服务文件..."
    $SUDO tee /etc/systemd/system/gost.service > /dev/null << EOF
[Unit]
Description=GOST Proxy
After=network.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=/usr/local/bin/gost -C /etc/gost/config.yaml
Restart=always
RestartSec=5
LimitNOFILE=1048576
StandardOutput=append:/var/log/gost/gost-stdout.log
StandardError=append:/var/log/gost/gost-stderr.log

[Install]
WantedBy=multi-user.target
EOF
    $SUDO systemctl daemon-reload
    $SUDO systemctl enable gost
    echo ">>> systemd 服务文件已创建"
else
    echo ">>> systemd 服务文件已存在，跳过"
fi

echo "REPAIR_DONE"
`, DefaultGostAPIPort, GostAPIUser, GostAPIPass)
		session, err := sshClient.Client.NewSession()
		if err != nil {
			return fmt.Errorf("创建 SSH 会话失败: %w", err)
		}
		repairOutput, _ := session.CombinedOutput(repairScript)
		session.Close()

		outputStr := string(repairOutput)
		// 输出每行日志
		for _, line := range strings.Split(outputStr, "\n") {
			if strings.HasPrefix(line, ">>>") {
				progressCallback(line)
			}
		}
		if !strings.Contains(outputStr, "REPAIR_DONE") {
			progressCallback(fmt.Sprintf("修复可能未完成: %s", outputStr))
		}
	}

	if serviceStatus != "active" || needReinstall || needReconfigure {
		// 重启服务
		progressCallback("重启 GOST 服务...")
		restartScript := `#!/bin/bash
if [ "$(id -u)" -eq 0 ]; then SUDO=""; else SUDO="sudo"; fi
$SUDO systemctl daemon-reload
$SUDO systemctl restart gost
sleep 2
$SUDO systemctl is-active gost 2>/dev/null || echo STILL_FAILED
`
		session2, err := sshClient.Client.NewSession()
		if err != nil {
			return fmt.Errorf("创建 SSH 会话失败: %w", err)
		}
		restartOutput, _ := session2.CombinedOutput(restartScript)
		session2.Close()

		if strings.Contains(string(restartOutput), "STILL_FAILED") {
			progressCallback("GOST 服务重启失败，检查错误日志...")
			errLogOutput, _ := sshClient.ExecuteCommand("journalctl -u gost -n 20 --no-pager 2>/dev/null")
			progressCallback(fmt.Sprintf("错误日志:\n%s", strings.TrimSpace(errLogOutput)))
			return fmt.Errorf("GOST 服务重启失败，请检查日志")
		}
		progressCallback("GOST 服务已重启")
	} else {
		progressCallback("GOST 服务运行正常，无需修复")
	}

	// 8. 验证 GOST API
	progressCallback("验证 GOST API...")
	time.Sleep(2 * time.Second)
	if err := verifyGostAPI(host); err != nil {
		progressCallback(fmt.Sprintf("GOST API 验证失败: %s", err))
		return fmt.Errorf("GOST API 验证失败: %w", err)
	}
	progressCallback("GOST API 验证成功")

	// 9. 检查本地转发服务是否齐全（tcp/ws/http/minio）
	progressCallback("步骤 7/8: 检查本地转发服务...")
	repairLocalForwards(host, server, serverId, progressCallback)

	// 10. 如果是系统服务器，检查 TLS 证书
	if server.ServerType == 2 {
		deployTlsCertsIfAvailable(server, progressCallback)
	}

	progressCallback("========================================")
	progressCallback(fmt.Sprintf("诊断修复完成! GOST API: http://%s:%d", host, DefaultGostAPIPort))
	progressCallback("========================================")

	return nil
}

// repairLocalForwards 检查并修复商户服务器上的本地转发服务
// 确保 tcp(10010), ws(10011), http(10012), minio(10013) 四个本地转发都存在
func repairLocalForwards(host string, server entity.Servers, serverId int, progressCallback func(string)) {
	// 获取当前 GOST 上已有的服务列表
	config, err := gostapi.GetConfig(host, "json")
	if err != nil {
		progressCallback(fmt.Sprintf("获取 GOST 配置失败，跳过转发检查: %s", err))
		return
	}

	existingServices := make(map[string]bool)
	for _, svc := range config.Services {
		existingServices[svc.Name] = true
	}

	// 检查基本转发 (tcp/ws/http)
	expectedServices := []struct {
		name string
		port int
	}{
		{fmt.Sprintf("local-tcp-%d", gostapi.MerchantGostPortTCP), gostapi.MerchantGostPortTCP},
		{fmt.Sprintf("local-ws-%d", gostapi.MerchantGostPortWS), gostapi.MerchantGostPortWS},
		{fmt.Sprintf("local-http-%d", gostapi.MerchantGostPortHTTP), gostapi.MerchantGostPortHTTP},
	}

	missingBasic := false
	for _, svc := range expectedServices {
		if !existingServices[svc.name] {
			progressCallback(fmt.Sprintf("缺少本地转发: %s (端口 %d)", svc.name, svc.port))
			missingBasic = true
		}
	}

	// 检查是否需要 mtls 升级（本地转发的 listener 从 tls 升级到 mtls 多路复用）
	needMtlsUpgrade := false
	if !missingBasic {
		for _, svc := range config.Services {
			if strings.HasPrefix(svc.Name, "local-") && svc.Listener != nil && svc.Listener.Type == "tls" {
				needMtlsUpgrade = true
				break
			}
		}
	}

	if missingBasic {
		progressCallback("补建基本本地转发 (tcp/ws/http)...")
		if err := gostapi.CreateMerchantLocalForwards(host); err != nil {
			progressCallback(fmt.Sprintf("补建基本本地转发失败: %s", err))
		} else {
			progressCallback("基本本地转发已补建")
		}
	} else if needMtlsUpgrade {
		progressCallback("升级本地转发: tls → mtls (多路复用)...")
		if err := gostapi.UpdateMerchantLocalForwards(host); err != nil {
			progressCallback(fmt.Sprintf("升级基本本地转发失败: %s", err))
		} else {
			progressCallback("基本本地转发已升级到 mtls")
		}
	} else {
		progressCallback("基本本地转发完整 (tcp/ws/http)")
	}

	// 确定 MinIO 地址
	minioAddr := "127.0.0.1:9000" // 默认 allinone
	if server.MerchantId > 0 {
		var minioNode entity.ClusterNodes
		if has, err := dbs.DBAdmin.Where("merchant_id = ? AND node_role = 'minio'", server.MerchantId).Get(&minioNode); err == nil && has && minioNode.PrivateIP != "" {
			minioAddr = fmt.Sprintf("%s:9000", minioNode.PrivateIP)
			progressCallback(fmt.Sprintf("检测到集群模式，MinIO 节点: %s", minioNode.PrivateIP))
		}
	}

	// 检查 MinIO 转发
	minioServiceName := fmt.Sprintf("local-minio-%d", gostapi.MerchantGostPortMinIO)
	if !existingServices[minioServiceName] {
		progressCallback(fmt.Sprintf("缺少 MinIO 本地转发: %s", minioServiceName))
		progressCallback(fmt.Sprintf("补建 MinIO 本地转发: 10013 → %s", minioAddr))
		if err := gostapi.CreateMinioLocalForward(host, minioAddr); err != nil {
			progressCallback(fmt.Sprintf("补建 MinIO 本地转发失败: %s", err))
		} else {
			progressCallback("MinIO 本地转发已补建")
		}
	} else if needMtlsUpgrade {
		progressCallback("升级 MinIO 本地转发: tls → mtls...")
		if err := gostapi.UpdateMinioLocalForward(host, minioAddr); err != nil {
			progressCallback(fmt.Sprintf("升级 MinIO 本地转发失败: %s", err))
		} else {
			progressCallback("MinIO 本地转发已升级到 mtls")
		}
	} else {
		progressCallback("MinIO 本地转发完整")
	}

	// 检查 relay chains 的 dialer 是否需要 mtls 升级
	repairRelayChains(host, server, serverId, config, progressCallback)
}

// repairRelayChains 检查并升级 relay chain 的 dialer 从 tls 到 mtls
// 升级顺序: 删除中继链 → 重启 GOST → 升级商户 listener → 重建中继链
func repairRelayChains(host string, server entity.Servers, serverId int, config *gostapi.Config, progressCallback func(string)) {
	needUpgrade := false
	for _, chain := range config.Chains {
		for _, hop := range chain.Hops {
			for _, node := range hop.Nodes {
				if node.Dialer != nil && node.Dialer.Type == "tls" && node.Connector != nil && node.Connector.Type == "relay" {
					needUpgrade = true
					break
				}
			}
			if needUpgrade {
				break
			}
		}
		if needUpgrade {
			break
		}
	}

	if !needUpgrade {
		progressCallback("中继 chain 配置正常")
		return
	}

	progressCallback("升级中继 chain dialer: tls → mtls (多路复用)...")

	// 1. 提取商户 IP（在删除配置之前）
	merchantIPs := make(map[string]bool)
	for _, chain := range config.Chains {
		for _, hop := range chain.Hops {
			for _, node := range hop.Nodes {
				if node.Connector != nil && node.Connector.Type == "relay" {
					if idx := strings.LastIndex(node.Addr, ":"); idx > 0 {
						ip := node.Addr[:idx]
						if ip != host {
							merchantIPs[ip] = true
						}
					}
				}
			}
		}
	}

	// 2. 查询关联的商户转发规则
	var gostServers []entity.MerchantGostServers
	_ = dbs.DBAdmin.Where("server_id = ? AND status = 1", server.Id).Find(&gostServers)
	if len(gostServers) == 0 {
		progressCallback("未找到关联的商户转发规则，跳过 chain 升级")
		return
	}

	// 3. 获取最新配置并删除所有中继服务和链
	freshConfig, _ := gostapi.GetConfig(host, "json")
	if freshConfig != nil {
		config = freshConfig
	}
	progressCallback("  清理中继转发服务...")
	for _, svc := range config.Services {
		if !strings.HasPrefix(svc.Name, "local-") {
			gostapi.DeleteService(host, svc.Name)
			if svc.Handler != nil && svc.Handler.Chain != "" {
				gostapi.DeleteChain(host, svc.Handler.Chain)
			}
		}
	}
	for _, chain := range config.Chains {
		gostapi.DeleteChain(host, chain.Name)
	}
	_, _ = gostapi.SaveConfig(host, "yaml", "")

	// 4. 重启 GOST 释放端口
	progressCallback("  重启 GOST 服务释放端口...")
	restartSSH, err := GetSSHClient(serverId)
	if err != nil {
		progressCallback(fmt.Sprintf("  获取 SSH 连接失败: %s，尝试不重启继续...", err))
	} else {
		defer restartSSH.Close()
		restartSSH.ExecuteCommand("sudo systemctl restart gost")
		for i := 0; i < 10; i++ {
			time.Sleep(time.Second)
			if _, err := gostapi.GetConfig(host, "json"); err == nil {
				progressCallback("  GOST 服务已重启")
				break
			}
			if i == 9 {
				progressCallback("  等待 GOST 重启超时")
				return
			}
		}
	}

	// 5. 在无活跃中继连接时，升级商户端 GOST listener (tls → mtls)
	for ip := range merchantIPs {
		merchantConfig, configErr := gostapi.GetConfig(ip, "json")
		if configErr != nil {
			progressCallback(fmt.Sprintf("  获取商户 %s 配置失败: %s", ip, configErr))
			continue
		}
		needMerchantUpgrade := false
		minioTargetAddr := ""
		for _, svc := range merchantConfig.Services {
			if strings.HasPrefix(svc.Name, "local-") && svc.Listener != nil && svc.Listener.Type == "tls" {
				needMerchantUpgrade = true
			}
			if strings.HasPrefix(svc.Name, "local-minio-") && svc.Forwarder != nil && len(svc.Forwarder.Nodes) > 0 {
				minioTargetAddr = svc.Forwarder.Nodes[0].Addr
			}
		}
		if !needMerchantUpgrade {
			progressCallback(fmt.Sprintf("  商户 %s 本地转发已是 mtls", ip))
			continue
		}
		progressCallback(fmt.Sprintf("  升级商户 %s 本地转发: tls → mtls...", ip))
		if err := gostapi.UpdateMerchantLocalForwards(ip); err != nil {
			progressCallback(fmt.Sprintf("  商户 %s 基本转发升级失败: %s", ip, err))
		}
		if minioTargetAddr == "" {
			minioTargetAddr = "127.0.0.1:9000"
		}
		if err := gostapi.UpdateMinioLocalForward(ip, minioTargetAddr); err != nil {
			progressCallback(fmt.Sprintf("  商户 %s MinIO 转发升级失败: %s", ip, err))
		}
		_, _ = gostapi.SaveConfig(ip, "yaml", "")
		progressCallback(fmt.Sprintf("  商户 %s 本地转发已升级到 mtls", ip))
	}

	// 6. 重建中继转发（使用 mtls dialer）
	progressCallback("  重建中继转发...")
	successCount := 0
	for _, gs := range gostServers {
		var merchant entity.Merchants
		has, err := dbs.DBAdmin.ID(gs.MerchantId).Get(&merchant)
		if err != nil || !has || merchant.ServerIP == "" || merchant.Port == 0 {
			continue
		}

		var forwardErr error
		if server.TlsEnabled == 1 {
			forwardErr = gostapi.CreateMerchantForwardsWithTls(host, merchant.Port, merchant.ServerIP)
		} else {
			forwardErr = gostapi.CreateMerchantForwards(host, merchant.Port, merchant.ServerIP)
		}

		if forwardErr != nil {
			progressCallback(fmt.Sprintf("  商户 %s 转发创建失败: %s", merchant.Name, forwardErr))
		} else {
			successCount++
			progressCallback(fmt.Sprintf("  商户 %s 转发已创建 (mtls)", merchant.Name))
		}
	}

	if successCount > 0 {
		_, _ = gostapi.SaveConfig(host, "yaml", "")
		progressCallback(fmt.Sprintf("中继 chain 升级完成，%d 个商户转发已升级", successCount))
	}
}