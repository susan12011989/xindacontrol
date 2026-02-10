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
	}

	instanceResult, err := aliyun.CreateInstance(createReq)
	if err != nil {
		return nil, fmt.Errorf("创建 ECS 实例失败: %w", err)
	}
	result.InstanceId = instanceResult.InstanceId
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

	// 注册到数据库
	server := &entity.Servers{
		Name:        config.ServerName,
		Host:        publicIP,
		Port:        22,
		Username:    "root",
		AuthType:    1, // 密码认证
		Password:    config.Password,
		ServerType:  2, // 系统服务器
		ForwardType: 2, // 直连转发
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

	progressCallback("========================================")
	progressCallback(fmt.Sprintf("✓ GOST 服务器部署完成!"))
	progressCallback(fmt.Sprintf("  公网IP: %s", publicIP))
	progressCallback(fmt.Sprintf("  GOST API: http://%s:%d", publicIP, DefaultGostAPIPort))
	progressCallback(fmt.Sprintf("  服务器ID: %d", result.ServerId))
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
wget -q "https://github.com/go-gost/gost/releases/download/v${GOST_VERSION}/gost_${GOST_VERSION}_linux_amd64.tar.gz" -O gost.tar.gz
tar -xzf gost.tar.gz
$SUDO mv gost /usr/local/bin/
$SUDO chmod +x /usr/local/bin/gost
rm -f gost.tar.gz

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

echo ">>> 启动服务..."
$SUDO systemctl daemon-reload
$SUDO systemctl enable gost --now

echo ">>> 优化网络..."
$SUDO tee -a /etc/sysctl.conf > /dev/null << 'SYSCTL'
net.core.somaxconn=65535
net.ipv4.tcp_max_syn_backlog=65535
net.ipv4.ip_local_port_range=1024 65535
net.ipv4.tcp_tw_reuse=1
fs.file-max=1048576
SYSCTL
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
	progressCallback(fmt.Sprintf("  10010(TCP) → 127.0.0.1:10000"))
	progressCallback(fmt.Sprintf("  10011(WS)  → 127.0.0.1:10001"))
	progressCallback(fmt.Sprintf("  10012(HTTP) → 127.0.0.1:10002"))

	err = gostapi.CreateMerchantLocalForwards(host)
	if err != nil {
		progressCallback(fmt.Sprintf("警告: 配置本地转发失败: %s", err))
		progressCallback("请稍后手动配置或重试安装")
	} else {
		progressCallback("商户本地转发配置成功")
	}

	progressCallback(fmt.Sprintf("✓ GOST 安装完成! API: http://%s:%d", host, DefaultGostAPIPort))
	progressCallback("  监听端口: 10010(TCP), 10011(WS), 10012(HTTP) - relay+tls")
	progressCallback("  转发到: 127.0.0.1:10000, 10001, 10002")

	return nil
}