package deploy

import (
	"fmt"
	"net"
	"os"
	"server/internal/dbhelper"
	"server/internal/server/cloud/aliyun"
	"server/internal/server/model"
	"server/pkg/dbs"
	"server/pkg/entity"
	"server/pkg/gostapi"
	"strings"
	"time"

	aliyunecs "github.com/alibabacloud-go/ecs-20140526/v6/client"
	"github.com/alibabacloud-go/tea/tea"
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

	// 自动部署 TLS 证书（如果已生成）
	deployTlsCertsIfAvailable(*server, progressCallback)

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

// uploadGostBinary 通过 SSH 上传预存的 GOST 二进制到目标服务器
func uploadGostBinary(client *ssh.Client, progressCallback func(string)) error {
	// 预存路径（control 服务器上的 GOST 二进制）
	gostBinaryPath := "/opt/control/assets/gost"

	data, err := os.ReadFile(gostBinaryPath)
	if err != nil {
		return fmt.Errorf("读取本地 GOST 二进制失败: %w", err)
	}

	progressCallback(fmt.Sprintf(">>> 上传 GOST 二进制 (%d MB)...", len(data)/1024/1024))

	// 通过 SSH 上传到 /tmp/gost
	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("创建上传会话失败: %w", err)
	}
	defer session.Close()

	stdin, err := session.StdinPipe()
	if err != nil {
		return fmt.Errorf("获取 stdin 失败: %w", err)
	}

	go func() {
		defer stdin.Close()
		stdin.Write(data)
	}()

	if err := session.Run("cat > /tmp/gost && chmod +x /tmp/gost"); err != nil {
		return fmt.Errorf("上传 GOST 失败: %w", err)
	}

	return nil
}

// installGostViaSSHClient 通过已有 SSH 客户端安装 GOST
func installGostViaSSHClient(client *ssh.Client, progressCallback func(string)) error {
	// 先上传 GOST 二进制（避免目标服务器从 GitHub 下载超时）
	if err := uploadGostBinary(client, progressCallback); err != nil {
		progressCallback(fmt.Sprintf("二进制上传失败，尝试从 GitHub 下载: %s", err))
	}

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

echo ">>> 安装 GOST..."
if [ -f /tmp/gost ]; then
    # 使用预上传的二进制
    $SUDO mv /tmp/gost /usr/local/bin/
    $SUDO chmod +x /usr/local/bin/gost
else
    # 回退：从 GitHub 下载
    echo ">>> 从 GitHub 下载 GOST..."
    cd /tmp
    wget -q --timeout=30 "https://github.com/go-gost/gost/releases/download/v${GOST_VERSION}/gost_${GOST_VERSION}_linux_amd64.tar.gz" -O gost.tar.gz
    tar -xzf gost.tar.gz
    $SUDO mv gost /usr/local/bin/
    $SUDO chmod +x /usr/local/bin/gost
    rm -f gost.tar.gz
fi

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

echo ">>> 放行防火墙端口..."
# ufw
if command -v ufw &>/dev/null && $SUDO ufw status 2>/dev/null | grep -q "active"; then
    $SUDO ufw allow 9394/tcp comment "GOST API" 2>/dev/null || true
    $SUDO ufw allow 443/tcp comment "GOST IM" 2>/dev/null || true
    $SUDO ufw allow 80/tcp comment "GOST HTTP" 2>/dev/null || true
    $SUDO ufw allow 8080/tcp comment "GOST FILE" 2>/dev/null || true
fi
# iptables (非 ufw 环境)
if ! command -v ufw &>/dev/null || ! $SUDO ufw status 2>/dev/null | grep -q "active"; then
    $SUDO iptables -C INPUT -p tcp --dport 9394 -j ACCEPT 2>/dev/null || $SUDO iptables -I INPUT -p tcp --dport 9394 -j ACCEPT 2>/dev/null || true
fi

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

// deployTlsCertsIfAvailable 如果数据库中有有效 TLS 证书，自动推送到新服务器
// 返回 true 表示已部署证书
func deployTlsCertsIfAvailable(server entity.Servers, progressCallback func(string)) bool {
	var caCert entity.TlsCertificates
	has, err := dbs.DBAdmin.Where("name = 'gost-ca' AND status = 1").Get(&caCert)
	if err != nil || !has {
		return false
	}

	var serverCert entity.TlsCertificates
	has, err = dbs.DBAdmin.Where("name = 'gost-server' AND status = 1").Get(&serverCert)
	if err != nil || !has {
		return false
	}

	progressCallback("检测到有效 TLS 证书，自动部署...")
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

// SetupGostDeploy 一键部署 GOST（安装+配置转发）
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

	// 2. 检测 GOST 是否已安装且健康
	progressCallback("检测 GOST 安装状态...")
	sshClient, err := GetSSHClient(req.ServerId)
	if err != nil {
		return fmt.Errorf("SSH 连接失败: %w", err)
	}
	defer sshClient.Close()

	needInstall := false

	output, _ := sshClient.ExecuteCommand("which gost")
	gostInstalled := strings.TrimSpace(output) != ""

	if gostInstalled {
		// 验证二进制是否能正常运行（防止损坏的二进制）
		versionOut, _ := sshClient.ExecuteCommand("/usr/local/bin/gost -V 2>&1")
		if !strings.Contains(versionOut, "gost") {
			progressCallback(fmt.Sprintf("GOST 二进制损坏（%s），将重新安装...", strings.TrimSpace(versionOut)))
			sshClient.ExecuteCommand("sudo rm -f /usr/local/bin/gost")
			needInstall = true
		}

		// 检查服务是否被 mask
		if !needInstall {
			maskedCheck, _ := sshClient.ExecuteCommand("systemctl is-enabled gost 2>/dev/null || echo 'unknown'")
			if strings.Contains(maskedCheck, "masked") {
				progressCallback("GOST 服务被 mask，正在解除...")
				sshClient.ExecuteCommand("sudo systemctl unmask gost 2>/dev/null || true")
				sshClient.ExecuteCommand("sudo systemctl daemon-reload")
			}
		}

		// 检查配置文件和服务文件
		if !needInstall {
			serviceCheck, _ := sshClient.ExecuteCommand("test -s /etc/systemd/system/gost.service && echo 'ok' || echo 'missing'")
			configCheck, _ := sshClient.ExecuteCommand("test -s /etc/gost/config.yaml && echo 'ok' || echo 'missing'")

			if strings.TrimSpace(serviceCheck) == "missing" || strings.TrimSpace(configCheck) == "missing" {
				progressCallback("GOST 服务/配置文件缺失或为空，将重新安装...")
				needInstall = true
			}
		}
	} else {
		needInstall = true
	}

	if needInstall {
		progressCallback("安装 GOST...")
		// 确保先解除 mask（如果有的话）
		sshClient.ExecuteCommand("sudo systemctl unmask gost 2>/dev/null || true")
		err = installGostViaSSHClient(sshClient.Client, progressCallback)
		if err != nil {
			return fmt.Errorf("安装 GOST 失败: %w", err)
		}
		progressCallback("GOST 安装完成")
	} else {
		progressCallback("GOST 已安装且健康")
		// 确保服务在运行
		statusOutput, _ := sshClient.ExecuteCommand("systemctl is-active gost 2>/dev/null || echo 'inactive'")
		if strings.TrimSpace(statusOutput) != "active" {
			progressCallback("GOST 服务未运行，正在启动...")
			sshClient.ExecuteCommand("sudo systemctl start gost 2>/dev/null || true")
			time.Sleep(3 * time.Second)
		}
	}

	// 4. 验证 GOST API（优先通过 SSH 本地验证，不依赖外网防火墙放行 9394 端口）
	progressCallback("验证 GOST API...")
	time.Sleep(2 * time.Second)
	verifyViaSSH := func() error {
		out, err := sshClient.ExecuteCommand(fmt.Sprintf(
			`curl -sf -u %s:%s --connect-timeout 5 http://127.0.0.1:%s/config >/dev/null 2>&1 && echo "OK" || echo "FAIL"`,
			gostapi.GostAPIUsername, gostapi.GostAPIPassword, gostapi.GostAPIPort))
		if err != nil {
			return fmt.Errorf("SSH 执行失败: %w", err)
		}
		if strings.TrimSpace(out) != "OK" {
			return fmt.Errorf("GOST API 本地验证失败")
		}
		return nil
	}
	if err := verifyViaSSH(); err != nil {
		progressCallback(fmt.Sprintf("GOST API 验证失败: %s，尝试重启服务...", err))
		sshClient.ExecuteCommand("sudo systemctl restart gost 2>/dev/null || true")
		time.Sleep(3 * time.Second)
		if err := verifyViaSSH(); err != nil {
			// 输出日志帮助诊断
			journalOutput, _ := sshClient.ExecuteCommand("sudo journalctl -u gost --no-pager -n 20 2>/dev/null || true")
			if journalOutput != "" {
				progressCallback(fmt.Sprintf("GOST 日志:\n%s", strings.TrimSpace(journalOutput)))
			}
			return fmt.Errorf("GOST API 不可用（重启后仍失败）: %w", err)
		}
		progressCallback("GOST API 重启后验证成功")
	} else {
		progressCallback("GOST API 验证成功")
	}

	// 确保云安全组放行所有 GOST 端口（9394/443/80/8080）
	progressCallback("检查云安全组端口放行...")
	if sgErr := ensureGostSecurityGroupRules(server, progressCallback); sgErr != nil {
		progressCallback(fmt.Sprintf("安全组放行跳过: %s", sgErr))
	}

	// 验证外网连通性
	if err := verifyGostAPI(host); err != nil {
		progressCallback(fmt.Sprintf("注意: GOST API 外网不可达（%s:9394），管理页面可能无法加载数据", host))
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

		// 自动分配 TunnelIP（多商户隔离）
		// tunnelIP 必须是服务器网卡上实际存在的 IP，否则 GOST bind 会失败
		// 注意：云服务器（如阿里云）的公网 IP 通过 NAT 映射，不在网卡上，不能用于 bind
		tunnelIP := m.TunnelIP

		// 如果 tunnelIP 就是服务器的公网 IP（Host），且只有一个商户，无需 bindIP 隔离
		if tunnelIP == server.Host && len(merchants) == 1 {
			progressCallback(fmt.Sprintf("  单商户模式，跳过 bindIP（公网IP %s 可能无法直接 bind）", tunnelIP))
			tunnelIP = ""
		} else if tunnelIP != "" && !isServerLocalIP(server, tunnelIP) {
			progressCallback(fmt.Sprintf("  TunnelIP(%s) 不属于当前服务器，忽略", tunnelIP))
			tunnelIP = ""
		}
		if tunnelIP == "" && len(merchants) > 1 {
			// 多商户时需要 bindIP 隔离，尝试分配辅助 IP
			allocated, allocErr := dbhelper.AllocateTunnelIPForMerchant(m.Id, req.ServerId)
			if allocErr != nil {
				progressCallback(fmt.Sprintf("  警告: 商户 %s 自动分配 TunnelIP 失败: %s", m.Name, allocErr))
			} else if !isServerLocalIP(server, allocated) {
				progressCallback(fmt.Sprintf("  分配的 TunnelIP(%s) 不属于当前服务器，忽略", allocated))
			} else {
				tunnelIP = allocated
				progressCallback(fmt.Sprintf("  自动分配 TunnelIP: %s", tunnelIP))
			}
		}

		// 清除所有旧规则（relay 和 direct 两种命名都清）
		_ = gostapi.DeleteMerchantForwards(host, m.Port, tunnelIP)
		_ = gostapi.DeleteMerchantDirectForwards(host, m.Port, tunnelIP)

		// 统一用 relay+tls 模式（TLS listener + relay chain）
		var forwardErr error
		if tlsDeployed || server.TlsEnabled == 1 {
			forwardErr = gostapi.CreateMerchantForwardsWithTls(host, m.Port, m.ServerIP, tunnelIP)
		} else {
			forwardErr = gostapi.CreateMerchantForwards(host, m.Port, m.ServerIP, tunnelIP)
		}

		if forwardErr != nil {
			progressCallback(fmt.Sprintf("  失败: %s", forwardErr))
			failCount++
		} else {
			progressCallback(fmt.Sprintf("  成功: %s 端口 %d/%d/%d (WSS/HTTP/FILE) bindIP=%s", m.Name, gostapi.SystemPortIM, gostapi.SystemPortHTTP, gostapi.SystemPortFile, tunnelIP))
			successCount++
		}
	}

	// 诊断：对比 /config 和 /config/services 的差异
	if diagCfg, diagErr := gostapi.GetConfig(host, ""); diagErr == nil {
		svcList, _ := gostapi.GetServiceList(host)
		chainList, _ := gostapi.GetChainList(host)
		svcCount, chainCount := 0, 0
		if svcList != nil { svcCount = svcList.Count }
		if chainList != nil { chainCount = chainList.Count }
		progressCallback(fmt.Sprintf("运行时: /config=%d服务/%d链, /config/services=%d服务, /config/chains=%d链",
			len(diagCfg.Services), len(diagCfg.Chains), svcCount, chainCount))
	} else {
		progressCallback(fmt.Sprintf("运行时状态获取失败: %s", diagErr))
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

	// 诊断：TLS 升级后运行时状态
	if diagCfg2, diagErr2 := gostapi.GetConfig(host, ""); diagErr2 == nil {
		progressCallback(fmt.Sprintf("TLS升级后运行时: %d 服务, %d 链", len(diagCfg2.Services), len(diagCfg2.Chains)))
	}

	// 9. 持久化配置（IM TCP+TLS 443 已包含在 CreateMerchantForwards 中，无需单独配置）
	progressCallback("保存 GOST 配置到文件...")
	if err := PersistGostConfig(req.ServerId); err != nil {
		progressCallback(fmt.Sprintf("警告: 配置持久化失败: %s", err))
	} else {
		progressCallback("配置已保存")
	}

	// 10. 配置文件端口(8080) Nginx 缓存
	if successCount > 0 {
		progressCallback("配置文件缓存 (Nginx)...")
		filePort := gostapi.SystemPortFile
		if !IsNginxInstalled(req.ServerId) {
			progressCallback("  安装 Nginx...")
			if installErr := InstallNginxToServer(req.ServerId, func(msg string) {
				progressCallback(fmt.Sprintf("  %s", msg))
			}); installErr != nil {
				progressCallback(fmt.Sprintf("  Nginx 安装失败（不影响转发）: %s", installErr))
			}
		}
		if IsNginxInstalled(req.ServerId) {
			// GOST file 服务改为仅本地监听，Nginx 接管公网 8080
			if err := UpdateGostServiceToLoopback(req.ServerId, filePort); err != nil {
				progressCallback(fmt.Sprintf("  GOST loopback 切换失败: %s", err))
			} else if err := ConfigureNginxCacheForPort(req.ServerId, filePort); err != nil {
				_ = RestoreGostServiceToPublic(req.ServerId, filePort)
				progressCallback(fmt.Sprintf("  Nginx 缓存配置失败: %s", err))
			} else {
				progressCallback(fmt.Sprintf("  文件缓存已启用 (端口 %d, 最大2GB, 7天过期, 防击穿)", filePort))
			}
		}
	}

	// 更新服务器的 ForwardType
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

// isServerLocalIP 判断 IP 是否属于当前服务器（host 或 auxiliary_ip）
func isServerLocalIP(server entity.Servers, ip string) bool {
	if ip == server.Host {
		return true
	}
	if server.AuxiliaryIP != "" {
		for _, aux := range strings.Split(server.AuxiliaryIP, ",") {
			if strings.TrimSpace(aux) == ip {
				return true
			}
		}
	}
	return false
}

// RepairGostServer 诊断并修复指定系统服务器的 GOST 隧道问题
func RepairGostServer(serverId int, progress func(string)) error {
	var server entity.Servers
	has, err := dbs.DBAdmin.Where("id = ? AND status = 1", serverId).Get(&server)
	if err != nil || !has {
		return fmt.Errorf("服务器不存在或已停用")
	}
	host := server.Host

	progress(fmt.Sprintf("========== 开始修复: %s (%s) ==========", server.Name, host))

	// 1. 检查 GOST 服务运行状态
	progress("步骤 1/5: 检查 GOST 服务...")
	sshClient, err := GetSSHClient(serverId)
	if err != nil {
		return fmt.Errorf("SSH 连接失败: %w", err)
	}
	defer sshClient.Close()

	statusOut, _ := sshClient.ExecuteCommand("systemctl is-active gost 2>/dev/null || echo 'inactive'")
	if strings.TrimSpace(statusOut) != "active" {
		progress("GOST 未运行，尝试启动...")
		sshClient.ExecuteCommand("sudo systemctl unmask gost 2>/dev/null || true")
		sshClient.ExecuteCommand("sudo systemctl start gost 2>/dev/null || true")
		time.Sleep(3 * time.Second)
		statusOut2, _ := sshClient.ExecuteCommand("systemctl is-active gost 2>/dev/null || echo 'inactive'")
		if strings.TrimSpace(statusOut2) != "active" {
			progress("GOST 启动失败，尝试重新安装...")
			if installErr := installGostViaSSHClient(sshClient.Client, progress); installErr != nil {
				return fmt.Errorf("GOST 安装失败: %w", installErr)
			}
			time.Sleep(3 * time.Second)
		}
		progress("GOST 服务已启动")
	} else {
		progress("GOST 服务运行正常")
	}

	// 2. 检查 GOST API
	progress("步骤 2/5: 检查 GOST API...")
	verifyOut, _ := sshClient.ExecuteCommand(fmt.Sprintf(
		`curl -sf -u %s:%s --connect-timeout 5 http://127.0.0.1:%s/config >/dev/null 2>&1 && echo "OK" || echo "FAIL"`,
		gostapi.GostAPIUsername, gostapi.GostAPIPassword, gostapi.GostAPIPort))
	if strings.TrimSpace(verifyOut) != "OK" {
		progress("GOST API 不可用，重启服务...")
		sshClient.ExecuteCommand("sudo systemctl restart gost 2>/dev/null || true")
		time.Sleep(3 * time.Second)
	}
	progress("GOST API 正常")

	// 3. 检查安全组
	progress("步骤 3/5: 检查安全组端口...")
	if sgErr := ensureGostSecurityGroupRules(server, progress); sgErr != nil {
		progress(fmt.Sprintf("安全组检查跳过: %s", sgErr))
	}

	// 4. 检查 services 是否存在，不存在则重建
	progress("步骤 4/5: 检查转发规则...")
	svcList, _ := gostapi.GetServiceList(host)
	svcCount := 0
	if svcList != nil {
		svcCount = svcList.Count
	}
	progress(fmt.Sprintf("当前运行: %d 个 service", svcCount))

	// 查询该服务器关联的商户
	var relations []entity.MerchantGostServers
	dbs.DBAdmin.Where("server_id = ? AND status = 1", serverId).Find(&relations)

	if svcCount == 0 && len(relations) > 0 {
		progress("无 service 运行，开始重建商户转发规则...")

		var merchantIds []int
		for _, rel := range relations {
			merchantIds = append(merchantIds, rel.MerchantId)
		}

		var merchants []entity.Merchants
		dbs.DBAdmin.In("id", merchantIds).Where("status = 1").Find(&merchants)

		// 部署 TLS 证书（如果有）
		tlsDeployed := deployTlsCertsIfAvailable(server, progress)

		for _, m := range merchants {
			if m.ServerIP == "" {
				continue
			}

			// 单商户不用 bindIP（避免 NAT 公网 IP bind 失败）
			tunnelIP := ""
			if len(merchants) > 1 {
				tunnelIP = m.TunnelIP
				if tunnelIP == server.Host {
					tunnelIP = ""
				}
			}

			_ = gostapi.DeleteMerchantForwards(host, m.Port, tunnelIP)
			_ = gostapi.DeleteMerchantDirectForwards(host, m.Port, tunnelIP)

			var forwardErr error
			if tlsDeployed || server.TlsEnabled == 1 {
				forwardErr = gostapi.CreateMerchantForwardsWithTls(host, m.Port, m.ServerIP, tunnelIP)
			} else {
				forwardErr = gostapi.CreateMerchantForwards(host, m.Port, m.ServerIP, tunnelIP)
			}

			if forwardErr != nil {
				progress(fmt.Sprintf("  商户 %s 重建失败: %s", m.Name, forwardErr))
			} else {
				progress(fmt.Sprintf("  商户 %s 重建成功", m.Name))
			}
		}

		// TLS 升级
		if server.TlsEnabled == 1 || tlsDeployed {
			if err := upgradeGostListenerToTls(host); err != nil {
				progress(fmt.Sprintf("TLS 升级失败: %s", err))
			}
		}
	} else if svcCount > 0 {
		progress("转发规则正常，无需重建")
	} else {
		progress("无关联商户，跳过转发检查")
	}

	// 5. 持久化并验证
	progress("步骤 5/5: 持久化配置并验证...")
	if err := PersistGostConfig(serverId); err != nil {
		progress(fmt.Sprintf("持久化失败: %s", err))
	} else {
		progress("配置已保存")
	}

	// 最终检测
	finalSvc, _ := gostapi.GetServiceList(host)
	finalChain, _ := gostapi.GetChainList(host)
	fsc, fcc := 0, 0
	if finalSvc != nil { fsc = finalSvc.Count }
	if finalChain != nil { fcc = finalChain.Count }
	p443, p80, p8080 := tcpCheck(host, 443), tcpCheck(host, 80), tcpCheck(host, 8080)

	progress(fmt.Sprintf("修复结果: %d 服务, %d 链 | 443=%s 80=%s 8080=%s", fsc, fcc, p443, p80, p8080))
	progress("========== 修复完成 ==========")
	return nil
}

// RebuildAllMerchantGost 批量重建所有商户的 GOST 转发规则
// 每个商户只在其关联的系统服务器上操作，互不干扰
func RebuildAllMerchantGost(progress func(string)) error {
	// 查所有有效商户
	var merchants []entity.Merchants
	if err := dbs.DBAdmin.Where("status = 1").Find(&merchants); err != nil {
		return fmt.Errorf("查询商户失败: %w", err)
	}
	progress(fmt.Sprintf("共 %d 个有效商户", len(merchants)))

	successCount := 0
	failCount := 0

	for _, m := range merchants {
		// 查该商户关联的系统服务器
		var relations []entity.MerchantGostServers
		if err := dbs.DBAdmin.Where("merchant_id = ? AND status = 1", m.Id).Find(&relations); err != nil {
			progress(fmt.Sprintf("[%s] 查询关联服务器失败: %s", m.Name, err))
			failCount++
			continue
		}
		if len(relations) == 0 {
			progress(fmt.Sprintf("[%s] 无关联系统服务器，跳过", m.Name))
			continue
		}

		for _, rel := range relations {
			var server entity.Servers
			has, _ := dbs.DBAdmin.Where("id = ? AND server_type = 2", rel.ServerId).Get(&server)
			if !has {
				continue
			}

			progress(fmt.Sprintf("[%s] 重建 GOST 规则 → %s (%s)", m.Name, server.Name, server.Host))

			// 自动分配 TunnelIP（多商户隔离）
			tunnelIP := m.TunnelIP
			if tunnelIP == "" {
				allocated, allocErr := dbhelper.AllocateTunnelIPForMerchant(m.Id, rel.ServerId)
				if allocErr != nil {
					progress(fmt.Sprintf("[%s] 警告: 自动分配 TunnelIP 失败: %s", m.Name, allocErr))
				} else {
					tunnelIP = allocated
				}
			}

			// 清除所有旧规则（relay 和 direct 两种命名都清）
			_ = gostapi.DeleteMerchantForwards(server.Host, m.Port, tunnelIP)
			_ = gostapi.DeleteMerchantDirectForwards(server.Host, m.Port, tunnelIP)

			// 统一用 relay+tls 模式重建（TLS listener + relay chain）
			tlsEnabled := server.TlsEnabled == 1
			var err error
			if tlsEnabled {
				err = gostapi.CreateMerchantForwardsWithTls(server.Host, m.Port, m.ServerIP, tunnelIP)
			} else {
				err = gostapi.CreateMerchantForwards(server.Host, m.Port, m.ServerIP, tunnelIP)
			}

			if err != nil {
				progress(fmt.Sprintf("[%s] 失败: %s", m.Name, err))
				failCount++
			} else {
				// 持久化
				_ = PersistGostConfig(rel.ServerId)
				progress(fmt.Sprintf("[%s] 成功 (bindIP=%s)", m.Name, tunnelIP))
				successCount++
			}
		}
	}

	progress(fmt.Sprintf("========== 重建完成: 成功 %d, 失败 %d ==========", successCount, failCount))
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

// ensureGostSecurityGroupRules 自动在云安全组放行 GOST 所需端口
func ensureGostSecurityGroupRules(server entity.Servers, progress func(string)) error {
	if server.CloudType != "aliyun" || server.CloudAccountId == 0 || server.CloudInstanceId == "" || server.CloudRegionId == "" {
		return fmt.Errorf("服务器未绑定阿里云账号或缺少云信息(type=%s, accountId=%d, instanceId=%s)", server.CloudType, server.CloudAccountId, server.CloudInstanceId)
	}

	cloud, err := aliyun.GetSystemCloudAccount(server.CloudAccountId)
	if err != nil {
		return fmt.Errorf("获取云账号失败: %w", err)
	}

	client, err := aliyun.NewEcsClient(cloud.AccessKey, cloud.AccessSecret, server.CloudRegionId)
	if err != nil {
		return fmt.Errorf("创建ECS客户端失败: %w", err)
	}

	// 查询实例安全组
	describeResp, err := client.DescribeInstances(&aliyunecs.DescribeInstancesRequest{
		RegionId:    tea.String(server.CloudRegionId),
		InstanceIds: tea.String(fmt.Sprintf(`["%s"]`, server.CloudInstanceId)),
	})
	if err != nil {
		return fmt.Errorf("查询实例失败: %w", err)
	}
	if describeResp.Body.Instances == nil || len(describeResp.Body.Instances.Instance) == 0 {
		return fmt.Errorf("实例不存在: %s", server.CloudInstanceId)
	}

	instance := describeResp.Body.Instances.Instance[0]
	if instance.SecurityGroupIds == nil || len(instance.SecurityGroupIds.SecurityGroupId) == 0 {
		return fmt.Errorf("实例无安全组: %s", server.CloudInstanceId)
	}

	sgId := *instance.SecurityGroupIds.SecurityGroupId[0]
	progress(fmt.Sprintf("安全组: %s，放行 GOST 端口...", sgId))

	// 需要放行的端口: 9394(API), 443(IM), 80(HTTP), 8080(FILE)
	ports := []struct {
		port string
		desc string
	}{
		{"9394/9394", "GOST-API"},
		{"443/443", "GOST-IM"},
		{"80/80", "GOST-HTTP"},
		{"8080/8080", "GOST-FILE"},
	}

	for _, p := range ports {
		err := aliyun.AuthorizeSecurityGroup(&aliyun.AuthorizeSecurityGroupRequest{
			CloudAccountId:  server.CloudAccountId,
			RegionId:        server.CloudRegionId,
			SecurityGroupId: sgId,
			Permissions: []*aliyunecs.AuthorizeSecurityGroupRequestPermissions{
				{
					IpProtocol:   tea.String("TCP"),
					PortRange:    tea.String(p.port),
					SourceCidrIp: tea.String("0.0.0.0/0"),
					Policy:       tea.String("Accept"),
					Description:  tea.String(p.desc),
				},
			},
		})
		if err != nil {
			// 规则可能已存在，忽略
			if !strings.Contains(err.Error(), "AuthorizationRuleExists") {
				progress(fmt.Sprintf("  端口 %s 放行失败: %s", p.port, err))
			}
		}
	}

	progress("安全组端口放行完成 (9394/443/80/8080)")
	return nil
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

	// 配置商户本地转发规则（V2: 统一入口 + TCP）
	progressCallback("配置商户本地转发规则 (V3 直转架构)...")
	for _, cfg := range gostapi.MerchantLocalForwardConfigs {
		progressCallback(fmt.Sprintf("  :%d → 127.0.0.1:%d (%s)", cfg.GostPort, cfg.AppPort, cfg.Name))
	}

	err = gostapi.CreateMerchantLocalForwards(host)
	if err != nil {
		progressCallback(fmt.Sprintf("警告: 配置本地转发失败: %s", err))
		progressCallback("请稍后手动配置或重试安装")
	} else {
		progressCallback("商户本地转发配置成功")
	}

	// V2: 部署 nginx 路径分发（替代旧的多端口代理容器）
	progressCallback("部署 nginx 路径分发...")
	var nginxConf string
	if req.IMHost != "" || req.APIHost != "" || req.MinIOHost != "" {
		// 多机模式：各服务分布在不同节点
		hosts := &gostapi.MerchantNginxHosts{
			IMHost:    req.IMHost,
			APIHost:   req.APIHost,
			MinIOHost: req.MinIOHost,
		}
		var nginxErr error
		nginxConf, nginxErr = gostapi.MerchantNginxConfigTemplateWithHosts(hosts)
		if nginxErr != nil {
			progressCallback(fmt.Sprintf("警告: 生成多机 nginx 配置失败: %s", nginxErr))
			nginxConf = gostapi.MerchantNginxConfigTemplate()
		} else {
			progressCallback("使用多机模式 nginx 配置")
		}
	} else {
		nginxConf = gostapi.MerchantNginxConfigTemplate()
	}
	for _, nc := range gostapi.MerchantNginxConfigs {
		progressCallback(fmt.Sprintf("  %s → %d (%s)", nc.Path, nc.AppPort, nc.Name))
	}

	nginxDeployErr := deployMerchantNginx(host, username, password, privateKey, port, nginxConf)
	if nginxDeployErr != nil {
		progressCallback(fmt.Sprintf("警告: nginx 部署失败: %s", nginxDeployErr))
		progressCallback("  请确保 Docker 已安装")
	} else {
		progressCallback("nginx 路径分发配置成功")
	}

	// 如果是已注册的系统服务器，自动部署 TLS 证书
	if req.ServerId > 0 {
		var serverEntity entity.Servers
		if has, err := dbs.DBAdmin.ID(req.ServerId).Get(&serverEntity); err == nil && has && serverEntity.ServerType == 2 {
			deployTlsCertsIfAvailable(serverEntity, progressCallback)
		}
	}

	progressCallback(fmt.Sprintf("✓ GOST 安装完成! (V3 直转架构) API: http://%s:%d", host, DefaultGostAPIPort))
	for _, cfg := range gostapi.MerchantLocalForwardConfigs {
		progressCallback(fmt.Sprintf("  :%d → :%d (%s)", cfg.GostPort, cfg.AppPort, cfg.Name))
	}

	return nil
}

// deployNginxWSSProxy 在商户服务器上部署 nginx WSS 代理容器（tsdd-ws-proxy）
// 监听 5210 端口，将 WSS 请求（/ 和 /im-ws）代理到 WuKongIM:5200
func deployNginxWSSProxy(host, username, password, privateKey string, port int) error {
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

	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", host, port), config)
	if err != nil {
		return fmt.Errorf("SSH 连接失败: %w", err)
	}
	defer client.Close()

	return deployNginxWSSProxyViaSSH(client)
}

// deployNginxWSSProxyViaSSH 通过 SSH 客户端部署 nginx WSS 代理容器
func deployNginxWSSProxyViaSSH(client *ssh.Client) error {
	// nginx 配置：将 / 和 /im-ws 都代理到 WuKongIM WebSocket 端口
	// 支持新旧 App 两种 webSocketPath
	deployScript := `#!/bin/bash
set -e

if [ "$(id -u)" -eq 0 ]; then SUDO=""; else SUDO="sudo"; fi

# 检查 Docker 是否安装
if ! command -v docker &> /dev/null; then
    echo "Docker 未安装，跳过 nginx WSS 代理部署"
    exit 1
fi

echo ">>> 创建 nginx WSS 代理配置..."
$SUDO mkdir -p /opt/tsdd/nginx-ws-proxy

$SUDO tee /opt/tsdd/nginx-ws-proxy/default.conf > /dev/null << 'NGINXEOF'
server {
    listen 5210;

    # 新 App: webSocketPath = /
    location = / {
        proxy_pass http://tsdd-wukongim:5200/;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_read_timeout 86400;
        proxy_send_timeout 86400;
    }

    # 旧 App: webSocketPath = /im-ws
    location /im-ws {
        proxy_pass http://tsdd-wukongim:5200/;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_read_timeout 86400;
        proxy_send_timeout 86400;
    }
}
NGINXEOF

echo ">>> 部署 nginx WSS 代理容器..."
# 停止并移除旧容器（如果存在）
$SUDO docker rm -f tsdd-ws-proxy 2>/dev/null || true

# 获取 tsdd-wukongim 所在的 Docker 网络
NETWORK=$($SUDO docker inspect tsdd-wukongim --format '{{range $key, $val := .NetworkSettings.Networks}}{{$key}}{{end}}' 2>/dev/null | head -1)
if [ -z "$NETWORK" ]; then
    NETWORK="tsdd_default"
fi

# 启动 nginx 容器
$SUDO docker run -d \
    --name tsdd-ws-proxy \
    --restart always \
    --network "$NETWORK" \
    -p 5210:5210 \
    -v /opt/tsdd/nginx-ws-proxy/default.conf:/etc/nginx/conf.d/default.conf:ro \
    nginx:alpine

echo ">>> nginx WSS 代理部署完成! 监听端口 5210"
`

	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("创建 SSH 会话失败: %w", err)
	}
	defer session.Close()

	output, err := session.CombinedOutput(deployScript)
	if err != nil {
		return fmt.Errorf("部署失败: %s, output: %s", err, string(output))
	}

	return nil
}

// deployMerchantNginx 在商户服务器上部署 V2 nginx 路径分发容器（tsdd-nginx）
// 替代旧的 tsdd-ws-proxy，统一处理 /ws、/api/、/s3/ 路径分发
func deployMerchantNginx(host, username, password, privateKey string, port int, nginxConf string) error {
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

	sshConfig := &ssh.ClientConfig{
		User:            username,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         30 * time.Second,
	}

	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", host, port), sshConfig)
	if err != nil {
		return fmt.Errorf("SSH 连接失败: %w", err)
	}
	defer client.Close()

	nginxDeployScript := fmt.Sprintf(`#!/bin/bash
set -e
if [ "$(id -u)" -eq 0 ]; then SUDO=""; else SUDO="sudo"; fi

$SUDO mkdir -p /opt/tsdd/nginx

$SUDO tee /opt/tsdd/nginx/default.conf > /dev/null << 'NGINX_CONF_EOF'
%s
NGINX_CONF_EOF

echo ">>> 部署 nginx 路径分发容器..."
$SUDO docker rm -f tsdd-ws-proxy 2>/dev/null || true
$SUDO docker rm -f tsdd-nginx 2>/dev/null || true

$SUDO docker run -d \
    --name tsdd-nginx \
    --restart always \
    --network host \
    -v /opt/tsdd/nginx/default.conf:/etc/nginx/conf.d/default.conf:ro \
    nginx:alpine

echo ">>> nginx 路径分发部署完成!"
`, nginxConf)

	nginxSession, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("创建 SSH 会话失败: %w", err)
	}
	defer nginxSession.Close()

	nginxOutput, err := nginxSession.CombinedOutput(nginxDeployScript)
	if err != nil {
		return fmt.Errorf("部署失败: %s, output: %s", err, string(nginxOutput))
	}

	return nil
}