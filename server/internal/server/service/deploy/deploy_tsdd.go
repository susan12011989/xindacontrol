package deploy

import (
	"fmt"
	"os"
	awscloud "server/internal/server/cloud/aws"
	"server/internal/server/model"
	cloudaws "server/internal/server/service/cloud_aws"
	"server/internal/server/utils"
	"server/pkg/consts"
	"server/pkg/dbs"
	"server/pkg/entity"
	"server/pkg/gostapi"
	"strings"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

// DeployTSDD 一键部署TSDD服务到已注册的服务器
func DeployTSDD(req model.DeployTSDDReq, operator string) (model.DeployTSDDResp, error) {
	var resp model.DeployTSDDResp

	// 获取服务器信息
	var server entity.Servers
	has, err := dbs.DBAdmin.Where("id = ?", req.ServerId).Get(&server)
	if err != nil {
		return resp, fmt.Errorf("查询服务器失败: %v", err)
	}
	if !has {
		return resp, fmt.Errorf("服务器不存在")
	}

	// 获取SSH客户端
	client, err := GetSSHClient(req.ServerId)
	if err != nil {
		return resp, fmt.Errorf("连接服务器失败: %v", err)
	}

	// 获取商户信息（用于配置）
	var merchant entity.Merchants
	if req.MerchantId > 0 {
		dbs.DBAdmin.Where("id = ?", req.MerchantId).Get(&merchant)
	}

	// 构建部署配置
	config := model.DefaultDeployConfig
	config.ExternalIP = server.Host

	// 执行部署
	resp = executeDeployment(client.SSHClient, config, req.ForceReset)
	resp.ServerId = req.ServerId

	if resp.Success {
		resp.APIUrl = fmt.Sprintf("http://%s:%d", server.Host, config.APIPort)
		resp.WebUrl = fmt.Sprintf("http://%s:%d", server.Host, config.WebPort)
		resp.AdminUrl = fmt.Sprintf("http://%s:%d", server.Host, config.ManagerPort)

		// 更新服务器端口信息
		dbs.DBAdmin.Where("id = ?", req.ServerId).Cols("port", "updated_at").Update(&entity.Servers{
			Port:      config.APIPort,
			UpdatedAt: time.Now(),
		})
	}

	// 记录部署历史
	logDeployHistory(req.ServerId, "deploy_tsdd", operator, resp)

	return resp, nil
}

// DeployTSDDByIP 通过IP直接部署（新服务器）
func DeployTSDDByIP(req model.DeployTSDDByIPReq, operator string) (model.DeployTSDDResp, error) {
	var resp model.DeployTSDDResp

	// 设置默认值
	if req.Port == 0 {
		req.Port = 22
	}
	if req.Username == "" {
		req.Username = "root"
	}

	// 创建SSH客户端
	client := &utils.SSHClient{
		Host:     req.Host,
		Port:     req.Port,
		Username: req.Username,
		Password: req.Password,
	}

	// 测试连接
	if _, err := client.ExecuteCommand("echo 'connection test'"); err != nil {
		return resp, fmt.Errorf("SSH连接失败: %v", err)
	}

	// 构建部署配置
	config := model.DefaultDeployConfig
	config.ExternalIP = req.Host

	// 执行部署
	resp = executeDeployment(client, config, req.ForceReset)

	if resp.Success {
		// 注册服务器到数据库
		serverName := req.ServerName
		if serverName == "" {
			serverName = fmt.Sprintf("TSDD-%s", req.Host)
		}

		newServer := &entity.Servers{
			MerchantId: req.MerchantId,
			Name:       serverName,
			Host:       req.Host,
			Port:       req.Port, // SSH端口
			Username:   req.Username,
			Password:   req.Password,
			Status:     1,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}
		if _, err := dbs.DBAdmin.Insert(newServer); err != nil {
			logx.Errorf("注册服务器失败: %v", err)
		} else {
			resp.ServerId = newServer.Id
		}

		resp.APIUrl = fmt.Sprintf("http://%s:%d", req.Host, config.APIPort)
		resp.WebUrl = fmt.Sprintf("http://%s:%d", req.Host, config.WebPort)
		resp.AdminUrl = fmt.Sprintf("http://%s:%d", req.Host, config.ManagerPort)
	}

	// 记录部署历史
	if resp.ServerId > 0 {
		logDeployHistory(resp.ServerId, "deploy_tsdd_new", operator, resp)
	}

	return resp, nil
}

// executeDeployment 执行实际的部署步骤
func executeDeployment(client *utils.SSHClient, config model.DeployConfig, forceReset bool) model.DeployTSDDResp {
	var resp model.DeployTSDDResp
	resp.Steps = make([]model.DeployStep, 0)

	// Step 1: 检查并安装Docker
	step1 := installDocker(client)
	resp.Steps = append(resp.Steps, step1)
	if step1.Status == "failed" {
		resp.Success = false
		resp.Message = "Docker安装失败"
		return resp
	}

	// Step 2: 如果强制重置，清理现有容器
	if forceReset {
		step2 := cleanupExisting(client)
		resp.Steps = append(resp.Steps, step2)
	}

	// Step 3: 创建工作目录和配置文件
	step3 := createWorkspace(client, config)
	resp.Steps = append(resp.Steps, step3)
	if step3.Status == "failed" {
		resp.Success = false
		resp.Message = "创建工作目录失败"
		return resp
	}

	// Step 4: 生成docker-compose.yml
	step4 := generateDockerCompose(client, config)
	resp.Steps = append(resp.Steps, step4)
	if step4.Status == "failed" {
		resp.Success = false
		resp.Message = "生成docker-compose配置失败"
		return resp
	}

	// Step 5: 拉取镜像并启动服务
	step5 := pullAndStartServices(client)
	resp.Steps = append(resp.Steps, step5)
	if step5.Status == "failed" {
		resp.Success = false
		resp.Message = "启动服务失败"
		return resp
	}

	// Step 6: 等待服务就绪并检查健康状态
	step6 := waitForServices(client, config)
	resp.Steps = append(resp.Steps, step6)

	resp.Success = true
	resp.Message = "部署完成"
	return resp
}

// installDocker 安装Docker（如果未安装）
func installDocker(client *utils.SSHClient) model.DeployStep {
	step := model.DeployStep{
		Name:   "安装Docker",
		Status: "running",
	}

	// 检查Docker是否已安装
	output, err := client.ExecuteCommand("docker --version 2>/dev/null")
	if err == nil && strings.Contains(output, "Docker version") {
		step.Status = "success"
		step.Message = "Docker已安装"
		step.Output = strings.TrimSpace(output)
		return step
	}

	// 安装Docker
	installScript := `
set -e
# 更新包列表
apt-get update -qq

# 安装依赖
apt-get install -y -qq ca-certificates curl gnupg lsb-release

# 添加Docker官方GPG密钥
install -m 0755 -d /etc/apt/keyrings
curl -fsSL https://download.docker.com/linux/debian/gpg | gpg --dearmor -o /etc/apt/keyrings/docker.gpg 2>/dev/null || true
chmod a+r /etc/apt/keyrings/docker.gpg

# 添加Docker仓库
echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/debian $(lsb_release -cs) stable" | tee /etc/apt/sources.list.d/docker.list > /dev/null

# 安装Docker
apt-get update -qq
apt-get install -y -qq docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin

# 启动Docker
systemctl enable docker
systemctl start docker

# 安装docker-compose（兼容旧版命令）
if ! command -v docker-compose &> /dev/null; then
    curl -SL "https://github.com/docker/compose/releases/download/v2.24.0/docker-compose-linux-$(uname -m)" -o /usr/local/bin/docker-compose
    chmod +x /usr/local/bin/docker-compose
fi

docker --version
`
	output, err = client.ExecuteCommandWithTimeout(installScript, 10*time.Minute)
	if err != nil {
		step.Status = "failed"
		step.Message = fmt.Sprintf("安装Docker失败: %v", err)
		step.Output = output
		return step
	}

	step.Status = "success"
	step.Message = "Docker安装成功"
	step.Output = output
	return step
}

// cleanupExisting 清理现有容器和数据
func cleanupExisting(client *utils.SSHClient) model.DeployStep {
	step := model.DeployStep{
		Name:   "清理现有部署",
		Status: "running",
	}

	cleanupScript := `
cd /opt/tsdd 2>/dev/null || true
docker-compose down -v 2>/dev/null || true
docker stop $(docker ps -aq --filter "name=tsdd") 2>/dev/null || true
docker rm $(docker ps -aq --filter "name=tsdd") 2>/dev/null || true
echo "cleanup done"
`
	output, _ := client.ExecuteCommand(cleanupScript)
	step.Status = "success"
	step.Message = "清理完成"
	step.Output = output
	return step
}

// createWorkspace 创建工作目录
func createWorkspace(client *utils.SSHClient, config model.DeployConfig) model.DeployStep {
	step := model.DeployStep{
		Name:   "创建工作目录",
		Status: "running",
	}

	cmd := `
mkdir -p /opt/tsdd/configs
mkdir -p /opt/tsdd/data
cd /opt/tsdd && pwd
`
	output, err := client.ExecuteCommand(cmd)
	if err != nil {
		step.Status = "failed"
		step.Message = fmt.Sprintf("创建目录失败: %v", err)
		step.Output = output
		return step
	}

	step.Status = "success"
	step.Message = "工作目录创建成功"
	step.Output = strings.TrimSpace(output)
	return step
}

// generateDockerCompose 生成docker-compose.yml
func generateDockerCompose(client *utils.SSHClient, config model.DeployConfig) model.DeployStep {
	step := model.DeployStep{
		Name:   "生成Docker Compose配置",
		Status: "running",
	}

	// 生成docker-compose.yml内容
	composeContent := generateComposeYAML(config)

	// 生成.env文件内容
	envContent := generateEnvFile(config)

	// 写入文件
	writeCmd := fmt.Sprintf(`cat > /opt/tsdd/docker-compose.yml << 'COMPOSE_EOF'
%s
COMPOSE_EOF

cat > /opt/tsdd/.env << 'ENV_EOF'
%s
ENV_EOF

cat /opt/tsdd/docker-compose.yml | head -20
`, composeContent, envContent)

	output, err := client.ExecuteCommand(writeCmd)
	if err != nil {
		step.Status = "failed"
		step.Message = fmt.Sprintf("写入配置文件失败: %v", err)
		step.Output = output
		return step
	}

	step.Status = "success"
	step.Message = "配置文件生成成功"
	step.Output = output
	return step
}

// generateComposeYAML 生成docker-compose.yml内容
func generateComposeYAML(config model.DeployConfig) string {
	return fmt.Sprintf(`version: '3.8'

services:
  # MySQL 数据库
  mysql:
    image: mysql:8.0
    container_name: tsdd-mysql
    restart: always
    environment:
      MYSQL_ROOT_PASSWORD: ${MYSQL_ROOT_PASSWORD}
      MYSQL_DATABASE: tsdd
    volumes:
      - mysql_data:/var/lib/mysql
    ports:
      - "3306:3306"
    command: --character-set-server=utf8mb4 --collation-server=utf8mb4_unicode_ci
    healthcheck:
      test: ["CMD", "mysqladmin", "ping", "-h", "localhost"]
      interval: 10s
      timeout: 5s
      retries: 5

  # Redis 缓存
  redis:
    image: redis:7-alpine
    container_name: tsdd-redis
    restart: always
    volumes:
      - redis_data:/data
    ports:
      - "6379:6379"
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 5s
      retries: 5

  # MinIO 文件存储
  minio:
    image: minio/minio:latest
    container_name: tsdd-minio
    restart: always
    environment:
      MINIO_ROOT_USER: ${MINIO_ROOT_USER}
      MINIO_ROOT_PASSWORD: ${MINIO_ROOT_PASSWORD}
    volumes:
      - minio_data:/data
    ports:
      - "9000:9000"
      - "9001:9001"
    command: server /data --console-address ":9001"
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:9000/minio/health/live"]
      interval: 30s
      timeout: 20s
      retries: 3

  # WuKongIM 通讯服务
  wukongim:
    image: registry.cn-shanghai.aliyuncs.com/wukongim/wukongim:latest
    container_name: tsdd-wukongim
    restart: always
    environment:
      WK_MODE: release
      WK_EXTERNAL_IP: ${EXTERNAL_IP}
      WK_EXTERNAL_TCP_ADDR: ${EXTERNAL_IP}:5100
      WK_EXTERNAL_WS_ADDR: ws://${EXTERNAL_IP}:5200
      WK_WEBHOOK_GRPCADDR: tsdd-server:6979
      WK_DATASOURCE_ADDR: http://tsdd-server:8090/v1/datasource
    volumes:
      - wukongim_data:/root/wukongim
    ports:
      - "5001:5001"
      - "5100:5100"
      - "5200:5200"
      - "5300:5300"
    depends_on:
      mysql:
        condition: service_healthy
      redis:
        condition: service_healthy

  # 唐僧叨叨后端服务
  tsdd-server:
    image: registry.cn-shanghai.aliyuncs.com/wukongim/tangsengdaodaoserver:latest
    container_name: tsdd-server
    restart: always
    environment:
      TS_MODE: release
      TS_EXTERNAL_IP: ${EXTERNAL_IP}
      TS_EXTERNAL_BASEURL: http://${EXTERNAL_IP}:8090
      TS_MYSQL_ADDR: root:${MYSQL_ROOT_PASSWORD}@tcp(tsdd-mysql:3306)/tsdd?charset=utf8mb4&parseTime=true
      TS_REDIS_ADDR: tsdd-redis:6379
      TS_WUKONGIM_APIURL: http://tsdd-wukongim:5001
      TS_MINIO_URL: http://tsdd-minio:9000
      TS_MINIO_ACCESSKEYID: ${MINIO_ROOT_USER}
      TS_MINIO_SECRETACCESSKEY: ${MINIO_ROOT_PASSWORD}
      TS_MINIO_UPLOADURL: http://${EXTERNAL_IP}:9000
      TS_SMSCODE: "${SMS_CODE}"
      TS_ADMINPWD: "${ADMIN_PASSWORD}"
    ports:
      - "8090:8090"
      - "6979:6979"
    depends_on:
      mysql:
        condition: service_healthy
      redis:
        condition: service_healthy
      minio:
        condition: service_healthy
      wukongim:
        condition: service_started

  # 管理后台
  manager:
    image: registry.cn-shanghai.aliyuncs.com/wukongim/tangsengdaodaomanager:latest
    container_name: tsdd-manager
    restart: always
    environment:
      API_URL: http://tsdd-server:8090/v1/
    ports:
      - "%d:80"
    depends_on:
      - tsdd-server

  # Web IM 前端
  web:
    image: registry.cn-shanghai.aliyuncs.com/wukongim/tangsengdaodaoweb:latest
    container_name: tsdd-web
    restart: always
    environment:
      API_URL: http://${EXTERNAL_IP}:8090
      WS_URL: ws://${EXTERNAL_IP}:5200
    ports:
      - "%d:80"
    depends_on:
      - tsdd-server

volumes:
  mysql_data:
  redis_data:
  minio_data:
  wukongim_data:
`, config.ManagerPort, config.WebPort)
}

// generateEnvFile 生成.env文件内容
func generateEnvFile(config model.DeployConfig) string {
	return fmt.Sprintf(`EXTERNAL_IP=%s
MYSQL_ROOT_PASSWORD=%s
MINIO_ROOT_USER=%s
MINIO_ROOT_PASSWORD=%s
ADMIN_PASSWORD=%s
SMS_CODE=%s
`, config.ExternalIP, config.MySQLPassword, config.MinioUser, config.MinioPassword, config.AdminPassword, config.SMSCode)
}

// pullAndStartServices 拉取镜像并启动服务
func pullAndStartServices(client *utils.SSHClient) model.DeployStep {
	step := model.DeployStep{
		Name:   "拉取镜像并启动服务",
		Status: "running",
	}

	// 使用docker compose启动（新版命令）
	startCmd := `
cd /opt/tsdd

# 拉取镜像
echo ">>> 拉取镜像..."
docker compose pull 2>&1 || docker-compose pull 2>&1

# 启动服务
echo ">>> 启动服务..."
docker compose up -d 2>&1 || docker-compose up -d 2>&1

# 等待几秒后检查状态
sleep 5
echo ">>> 容器状态:"
docker ps --format "table {{.Names}}\t{{.Status}}" | grep tsdd || true
`
	output, err := client.ExecuteCommandWithTimeout(startCmd, 15*time.Minute)
	if err != nil {
		step.Status = "failed"
		step.Message = fmt.Sprintf("启动服务失败: %v", err)
		step.Output = output
		return step
	}

	// 检查是否有容器在运行
	checkCmd := "docker ps --format '{{.Names}}' | grep -c tsdd || echo '0'"
	countOutput, _ := client.ExecuteCommand(checkCmd)
	count := strings.TrimSpace(countOutput)
	if count == "0" {
		step.Status = "failed"
		step.Message = "没有容器成功启动"
		step.Output = output
		return step
	}

	step.Status = "success"
	step.Message = fmt.Sprintf("服务启动成功，%s个容器运行中", count)
	step.Output = output
	return step
}

// waitForServices 等待服务就绪
func waitForServices(client *utils.SSHClient, config model.DeployConfig) model.DeployStep {
	step := model.DeployStep{
		Name:   "检查服务健康状态",
		Status: "running",
	}

	// 等待服务启动
	time.Sleep(10 * time.Second)

	checkCmd := fmt.Sprintf(`
echo "=== 容器状态 ==="
docker ps --format "table {{.Names}}\t{{.Status}}" | grep tsdd

echo ""
echo "=== API健康检查 ==="
curl -s -o /dev/null -w "API (8090): %%{http_code}\n" --connect-timeout 5 http://127.0.0.1:8090/v1/health 2>/dev/null || echo "API (8090): 连接失败"
curl -s -o /dev/null -w "Manager (%d): %%{http_code}\n" --connect-timeout 5 http://127.0.0.1:%d/v1/health 2>/dev/null || echo "Manager (%d): 连接失败"
curl -s -o /dev/null -w "WuKongIM (5001): %%{http_code}\n" --connect-timeout 5 http://127.0.0.1:5001/health 2>/dev/null || echo "WuKongIM (5001): 连接失败"
`, config.ManagerPort, config.ManagerPort, config.ManagerPort)

	output, _ := client.ExecuteCommand(checkCmd)

	// 检查是否有服务不健康
	if strings.Contains(output, "连接失败") || strings.Contains(output, "502") {
		step.Status = "warning"
		step.Message = "部分服务可能还在启动中"
	} else {
		step.Status = "success"
		step.Message = "所有服务运行正常"
	}
	step.Output = output
	return step
}

// GetDeployStatus 获取部署状态
func GetDeployStatus(req model.GetDeployStatusReq) (model.GetDeployStatusResp, error) {
	var resp model.GetDeployStatusResp

	client, err := GetSSHClient(req.ServerId)
	if err != nil {
		return resp, err
	}

	// 检查是否有tsdd容器
	cmd := "docker ps --format '{{.Names}}' 2>/dev/null | grep tsdd || echo ''"
	output, _ := client.ExecuteCommand(cmd)

	services := strings.Split(strings.TrimSpace(output), "\n")
	if len(services) == 1 && services[0] == "" {
		resp.Deployed = false
		return resp, nil
	}

	resp.Deployed = true
	resp.Services = services

	// 检查健康状态
	healthCmd := "curl -s -o /dev/null -w '%{http_code}' --connect-timeout 3 http://127.0.0.1:8090/v1/health 2>/dev/null || echo '000'"
	healthOutput, _ := client.ExecuteCommand(healthCmd)
	resp.Healthy = strings.TrimSpace(healthOutput) == "200"

	return resp, nil
}

// logDeployHistory 记录部署历史
func logDeployHistory(serverId int, action string, operator string, resp model.DeployTSDDResp) {
	status := 1
	if !resp.Success {
		status = 2
	}

	// 构建输出摘要
	var outputParts []string
	for _, step := range resp.Steps {
		outputParts = append(outputParts, fmt.Sprintf("[%s] %s: %s", step.Status, step.Name, step.Message))
	}
	output := strings.Join(outputParts, "\n")

	history := entity.DeployHistory{
		ServerId:    serverId,
		Action:      action,
		ServiceName: "tsdd",
		Operator:    operator,
		Status:      status,
		Output:      output,
		CreatedAt:   time.Now(),
	}
	if !resp.Success {
		history.ErrorMsg = resp.Message
	}
	dbs.DBAdmin.Insert(&history)
}

// 默认密码常量（与consts包一致）
func init() {
	if consts.DefaultPassword == "" {
		// 设置默认值
	}
}

// ========== AMI 方案部署 ==========

// TSDD 默认 AMI ID（按区域）— 使用 Ubuntu 22.04 LTS 官方公共镜像
var DefaultTSDDAMI = map[string]string{
	"us-east-1":      "ami-0c7217cdde317cfec", // 美东
	"us-west-2":      "ami-0efcece6bed30fd98", // 美西(俄勒冈)
	"eu-west-1":      "ami-0905a3c97561e0b69", // 欧洲(爱尔兰)
	"ap-southeast-1": "ami-078c1149d8ad719a7", // 新加坡
	"ap-northeast-1": "ami-0d52744d6551d851e", // 东京
	"ap-east-1":      "ami-0d96ec8a788679eb2", // 香港
}

// trySSHConnect 尝试用 root 密码连接 SSH（cloud-init 设置的）
// 创建 AMI 前已执行 cloud-init clean，确保新实例会重新设置 root 密码登录
func trySSHConnect(host, password string) (*utils.SSHClient, string) {
	client := &utils.SSHClient{
		Host:     host,
		Port:     22,
		Username: "root",
		Password: password,
	}
	// 最多等待 3 分钟（cloud-init 需要时间执行）
	for i := 0; i < 18; i++ {
		if _, err := client.ExecuteCommand("echo 'SSH ready'"); err == nil {
			return client, "root"
		}
		time.Sleep(10 * time.Second)
	}
	logx.Errorf("SSH root@%s 连接失败（3分钟超时）", host)
	return nil, ""
}

// DeployTSDDWithAMI 使用 AMI 方式部署 TSDD
func DeployTSDDWithAMI(req model.DeployTSDDWithAMIReq, operator string) (model.DeployTSDDWithAMIResp, error) {
	var resp model.DeployTSDDWithAMIResp

	// 获取云账号
	acc, err := awscloud.ResolveAwsAccount(nil, req.MerchantId, req.CloudAccountId)
	if err != nil {
		return resp, fmt.Errorf("获取云账号失败: %v", err)
	}

	// 确定使用的 AMI
	amiId := req.AMIId
	if amiId == "" {
		// 如果指定了源服务器，先从该服务器创建 AMI
		if req.SourceServerId > 0 {
			logx.Infof("从服务器 %d 创建 AMI...", req.SourceServerId)
			createdAMI, err := createAMIFromServer(acc, req.SourceServerId, req.RegionId)
			if err != nil {
				return resp, fmt.Errorf("从服务器创建 AMI 失败: %v", err)
			}
			amiId = createdAMI
		} else {
			// 动态查找最新 Ubuntu 22.04 AMI（通过 AWS API）
			logx.Infof("动态查找 Ubuntu AMI (region: %s)...", req.RegionId)
			foundAMI, findErr := cloudaws.FindLatestUbuntuAMI(acc, req.RegionId)
			if findErr != nil {
				logx.Errorf("动态查找 AMI 失败，使用硬编码回退: %v", findErr)
				amiId = DefaultTSDDAMI[req.RegionId]
			} else {
				amiId = foundAMI
			}
			if amiId == "" {
				return resp, fmt.Errorf("该区域 %s 无法获取可用的 Ubuntu AMI，请手动指定 ami_id", req.RegionId)
			}
		}
	}

	// 设置默认值
	instanceType := req.InstanceType
	if instanceType == "" {
		instanceType = "t3.medium"
	}
	volumeSize := req.VolumeSizeGiB
	if volumeSize == 0 && req.AMIId == "" && req.SourceServerId == 0 {
		// 仅在使用自动查找的 Ubuntu AMI 时设置默认 30GB
		// 用户指定的自定义 AMI 使用 AMI 自身的卷配置（volumeSize=0 不覆盖）
		volumeSize = 30
	}
	serverName := req.ServerName
	if serverName == "" {
		serverName = fmt.Sprintf("TSDD-%s-%d", req.RegionId, time.Now().Unix())
	}

	// 创建 EC2 实例
	logx.Infof("使用 AMI %s 创建 EC2 实例...", amiId)
	createReq := model.AwsCreateEc2InstanceReq{
		MerchantId:     req.MerchantId,
		CloudAccountId: req.CloudAccountId,
		RegionId:       req.RegionId,
		ImageId:        amiId,
		InstanceType:   instanceType,
		SubnetId:       req.SubnetId,
		KeyName:        req.KeyName,
		VolumeSizeGiB:  volumeSize,
		InstanceName:   serverName,
	}

	instanceId, err := cloudaws.CreateEc2Instance(acc, createReq)
	if err != nil {
		return resp, fmt.Errorf("创建 EC2 实例失败: %v", err)
	}
	resp.InstanceId = instanceId
	logx.Infof("EC2 实例创建成功: %s", instanceId)

	// 等待实例运行
	logx.Info("等待实例启动...")
	err = cloudaws.WaitForInstanceRunning(acc, req.RegionId, instanceId, 5*time.Minute)
	if err != nil {
		resp.Message = fmt.Sprintf("实例创建成功但等待启动超时: %v", err)
		return resp, nil
	}

	// 等待一段时间让系统完全初始化
	time.Sleep(30 * time.Second)

	// 获取公网 IP
	publicIP, err := cloudaws.GetInstancePublicIP(acc, req.RegionId, instanceId)
	if err != nil {
		resp.Message = fmt.Sprintf("获取公网 IP 失败: %v", err)
		return resp, nil
	}
	resp.PublicIP = publicIP
	logx.Infof("实例公网 IP: %s", publicIP)

	// EBS 数据卷设置（如果启用）
	if req.EnableExtraEBS {
		logx.Info("创建并挂载额外 EBS 数据卷...")
		dbVolId, minioVolId, ebsErr := setupExtraEBSVolumes(acc, req, instanceId, publicIP)
		resp.DBVolumeId = dbVolId
		resp.MinioVolumeId = minioVolId
		if ebsErr != nil {
			logx.Errorf("EBS 卷设置失败: %v", ebsErr)
			resp.Message = fmt.Sprintf("实例已创建，但EBS卷设置失败: %v", ebsErr)
			// 不中断流程，实例仍可使用
		}
	}

	// 判断是自定义 AMI 还是纯 Ubuntu AMI
	isCustomAMI := req.AMIId != "" || req.SourceServerId > 0

	// 尝试 SSH 连接：先用 root（cloud-init 设置的），再 fallback 到 ubuntu（自定义 AMI 可能保留原用户）
	sshClient, sshUser := trySSHConnect(publicIP, consts.DefaultPassword)
	if sshClient == nil {
		resp.Message = fmt.Sprintf("实例已创建(IP: %s)，但 SSH 连接失败（已尝试 root 和 ubuntu 用户）", publicIP)
		// 仍然更新服务器记录，方便后续手动修复
		goto updateServerRecord
	}
	logx.Infof("SSH 连接成功: %s@%s", sshUser, publicIP)

	{
		deployOK := true // 跟踪部署是否真正成功

		if isCustomAMI {
			// 自定义 AMI：Docker 和服务已预装，只需把旧 IP 全局替换为新 IP，然后重启
			logx.Infof("自定义 AMI 部署：全局替换 IP 为 %s (SSH用户: %s)...", publicIP, sshUser)

			// 如果启用了独立数据磁盘(fresh EBS)且是克隆部署，预先 dump 源数据库 schema
			// 避免 tsdd-server migration 在全新数据库上因冲突 panic
			var schemaDump string
			if req.EnableExtraEBS && req.SourceServerId > 0 {
				logx.Infof("Fresh EBS + 克隆部署：从源服务器 %d dump 数据库 schema...", req.SourceServerId)
				if dump, dumpErr := dumpSourceDBSchema(req.SourceServerId); dumpErr != nil {
					logx.Errorf("dump 源数据库 schema 失败（非致命）: %v", dumpErr)
				} else {
					schemaDump = dump
				}
			}

			if err := updateCustomAMIConfig(sshClient, publicIP, schemaDump); err != nil {
				logx.Errorf("更新自定义 AMI 配置失败: %v", err)
				resp.Message = fmt.Sprintf("实例已创建(IP: %s)，但配置更新失败: %v", publicIP, err)
				deployOK = false
			} else {
				logx.Infof("自定义 AMI 配置更新成功: %s", publicIP)
			}
		} else {
			// 纯 Ubuntu AMI：需要完整安装 TSDD（Docker + docker-compose + 拉镜像）
			logx.Info("开始完整 TSDD 部署（Docker 安装 + 服务启动）...")

			deployConfig := model.DefaultDeployConfig
			deployConfig.ExternalIP = publicIP
			deployResult := executeDeployment(sshClient, deployConfig, false)
			if !deployResult.Success {
				logx.Errorf("TSDD 部署失败: %s", deployResult.Message)
				resp.Message = fmt.Sprintf("实例已创建(IP: %s)，但 TSDD 部署失败: %s", publicIP, deployResult.Message)
				deployOK = false
			} else {
				logx.Infof("TSDD 部署成功: %s", publicIP)
			}

			// 上传 TSDD 服务器资源（修复后的二进制 + assets 默认头像，非致命）
			logx.Info("上传TSDD服务器资源...")
			if resErr := sshUploadAndSetupResources(sshClient); resErr != nil {
				logx.Errorf("资源上传失败（非致命）: %v", resErr)
			}

			// 最终重启确保使用最新二进制
			sshClient.ExecuteCommandWithTimeout("cd /opt/tsdd && docker compose up -d tsdd-server 2>/dev/null || docker-compose up -d tsdd-server 2>/dev/null", 2*time.Minute)
		}

		resp.Success = deployOK
	}

updateServerRecord:
	// 更新商户服务器记录（创建商户时已自动创建，这里更新 IP 等信息）
	// 使用实际连接成功的 SSH 用户名（而非硬编码 root）
	actualUser := sshUser
	if actualUser == "" {
		actualUser = "root" // SSH 失败时的默认值
	}
	if req.MerchantId > 0 {
		var existingServer entity.Servers
		has, _ := dbs.DBAdmin.Where("merchant_id = ? AND server_type = 1", req.MerchantId).Get(&existingServer)
		if has {
			existingServer.Host = publicIP
			existingServer.Name = serverName
			existingServer.Username = actualUser
			existingServer.Password = consts.DefaultPassword
			existingServer.Port = 22
			existingServer.AwsInstanceId = instanceId
			existingServer.AwsRegionId = req.RegionId
			existingServer.UpdatedAt = time.Now()
			if _, err := dbs.DBAdmin.ID(existingServer.Id).Cols("host", "name", "username", "password", "port", "aws_instance_id", "aws_region_id", "updated_at").Update(&existingServer); err != nil {
				logx.Errorf("更新商户服务器失败: %v", err)
			}
			resp.ServerId = existingServer.Id
		} else {
			newServer := &entity.Servers{
				ServerType:    1,
				MerchantId:    req.MerchantId,
				Name:          serverName,
				Host:          publicIP,
				Port:          22,
				Username:      actualUser,
				Password:      consts.DefaultPassword,
				AwsInstanceId: instanceId,
				AwsRegionId:   req.RegionId,
				Status:        1,
				CreatedAt:     time.Now(),
				UpdatedAt:     time.Now(),
			}
			if _, err := dbs.DBAdmin.Insert(newServer); err != nil {
				logx.Errorf("注册服务器失败: %v", err)
			} else {
				resp.ServerId = newServer.Id
			}
		}
	}

	// 同步更新商户的 server_ip，并刷新系统服务器 GOST 转发
	if req.MerchantId > 0 && publicIP != "" {
		if _, err := dbs.DBAdmin.Where("id = ?", req.MerchantId).Cols("server_ip").Update(&entity.Merchants{ServerIP: publicIP}); err != nil {
			logx.Errorf("更新商户 server_ip 失败: %v", err)
		} else {
			// 同步 servers.host
			dbs.DBAdmin.Table("servers").
				Where("server_type = ? AND merchant_id = ?", 1, req.MerchantId).
				Update(map[string]any{"host": publicIP, "updated_at": time.Now()})

			// 触发 GOST 转发更新
			var m entity.Merchants
			if has, _ := dbs.DBAdmin.Where("id = ?", req.MerchantId).Get(&m); has && m.Port > 0 {
				var sysServers []entity.Servers
				if err := dbs.DBAdmin.Where("server_type = ?", 2).Find(&sysServers); err == nil {
					for _, s := range sysServers {
						tlsEnabled := s.TlsEnabled == 1
						if s.ForwardType == entity.ForwardTypeDirect {
							gostapi.EnqueueUpdateMerchantDirectForwards(s.Host, m.Port, publicIP)
						} else {
							gostapi.EnqueueUpdateMerchantForwards(s.Host, m.Port, publicIP, tlsEnabled)
						}
					}
					logx.Infof("AMI 部署完成，已触发 GOST 转发更新: merchant=%d, ip=%s, port=%d, servers=%d", m.Id, publicIP, m.Port, len(sysServers))
				}
			}
		}
	}

	// 设置响应 URL（自定义 AMI 检测实际端口，全新部署用默认端口）
	if resp.Message == "" {
		resp.Message = "部署成功"
	}
	if isCustomAMI && sshClient != nil {
		detectedAPI, detectedWeb, detectedManager := detectCustomAMIPorts(sshClient)
		resp.APIUrl = fmt.Sprintf("http://%s:%d", publicIP, detectedAPI)
		resp.WebUrl = fmt.Sprintf("http://%s:%d", publicIP, detectedWeb)
		resp.AdminUrl = fmt.Sprintf("http://%s:%d", publicIP, detectedManager)
	} else {
		resp.APIUrl = fmt.Sprintf("http://%s:%d", publicIP, model.DefaultDeployConfig.APIPort)
		resp.WebUrl = fmt.Sprintf("http://%s:%d", publicIP, model.DefaultDeployConfig.WebPort)
		resp.AdminUrl = fmt.Sprintf("http://%s:%d", publicIP, model.DefaultDeployConfig.ManagerPort)
	}

	// 记录部署历史
	if resp.ServerId > 0 {
		logDeployHistory(resp.ServerId, "deploy_tsdd_ami", operator, model.DeployTSDDResp{
			Success:  resp.Success,
			Message:  resp.Message,
			ServerId: resp.ServerId,
			APIUrl:   resp.APIUrl,
			WebUrl:   resp.WebUrl,
			AdminUrl: resp.AdminUrl,
		})
	}

	return resp, nil
}

// createAMIFromServer 从已有服务器创建 AMI
func createAMIFromServer(acc *entity.CloudAccounts, serverId int, region string) (string, error) {
	// 获取服务器信息
	var server entity.Servers
	has, err := dbs.DBAdmin.Where("id = ?", serverId).Get(&server)
	if err != nil {
		return "", err
	}
	if !has {
		return "", fmt.Errorf("服务器不存在")
	}

	// 通过 SSH 获取实例 ID
	client := &utils.SSHClient{
		Host:     server.Host,
		Port:     server.Port,
		Username: server.Username,
		Password: server.Password,
	}

	instanceId, err := client.ExecuteCommand("curl -s http://169.254.169.254/latest/meta-data/instance-id")
	if err != nil {
		return "", fmt.Errorf("获取实例 ID 失败: %v", err)
	}
	instanceId = strings.TrimSpace(instanceId)
	if instanceId == "" {
		return "", fmt.Errorf("无法获取实例 ID")
	}

	// 清理 cloud-init 状态，确保从此 AMI 启动的新实例会重新执行 user-data（设置 root 密码登录）
	logx.Infof("清理 cloud-init 状态: %s", server.Host)
	if output, cleanErr := client.ExecuteCommand("sudo cloud-init clean 2>&1 || true"); cleanErr != nil {
		logx.Errorf("cloud-init clean 失败（非致命）: %v, output: %s", cleanErr, output)
	}

	// 创建 AMI
	amiName := fmt.Sprintf("TSDD-Clone-%s-%d", server.Name, time.Now().Unix())
	createReq := model.AwsCreateAMIReq{
		RegionId:    region,
		InstanceId:  instanceId,
		Name:        amiName,
		Description: fmt.Sprintf("TSDD 克隆自 %s (%s)", server.Name, server.Host),
		NoReboot:    true, // 不重启以避免服务中断
	}

	resp, err := cloudaws.CreateAMI(acc, createReq)
	if err != nil {
		return "", fmt.Errorf("创建 AMI 失败: %v", err)
	}

	// 等待 AMI 可用
	logx.Infof("等待 AMI %s 创建完成...", resp.ImageId)
	err = cloudaws.WaitForAMIAvailable(acc, region, resp.ImageId, 15*time.Minute)
	if err != nil {
		return "", fmt.Errorf("等待 AMI 可用失败: %v", err)
	}

	logx.Infof("AMI 创建成功: %s", resp.ImageId)
	return resp.ImageId, nil
}

// dumpSourceDBSchema 从源服务器 dump 数据库 schema（无数据）+ gorp_migrations 记录
// 用于克隆部署时初始化全新数据库，避免 migration 冲突
func dumpSourceDBSchema(sourceServerId int) (string, error) {
	var server entity.Servers
	has, err := dbs.DBAdmin.Where("id = ?", sourceServerId).Get(&server)
	if err != nil || !has {
		return "", fmt.Errorf("源服务器不存在: id=%d", sourceServerId)
	}

	client := &utils.SSHClient{
		Host:     server.Host,
		Port:     server.Port,
		Username: server.Username,
		Password: server.Password,
	}

	// Dump schema (no data) + gorp_migrations data
	dumpCmd := `docker exec tsdd-mysql bash -c 'MYSQL_PWD="$MYSQL_ROOT_PASSWORD" mysqldump --no-data tsdd 2>/dev/null && MYSQL_PWD="$MYSQL_ROOT_PASSWORD" mysqldump tsdd gorp_migrations --no-create-info --complete-insert 2>/dev/null'`
	output, err := client.ExecuteCommandWithTimeout(dumpCmd, 2*time.Minute)
	if err != nil {
		return "", fmt.Errorf("dump 源数据库失败: %v", err)
	}

	logx.Infof("[CloneDeploy] 从源服务器 %s dump schema 完成, 大小: %d bytes", server.Host, len(output))
	return output, nil
}

// detectCustomAMIPorts 从远程 docker-compose.yml 中检测实际端口映射
func detectCustomAMIPorts(client *utils.SSHClient) (apiPort, webPort, managerPort int) {
	apiPort = model.DefaultDeployConfig.APIPort       // 8090
	webPort = model.DefaultDeployConfig.WebPort        // 82
	managerPort = model.DefaultDeployConfig.ManagerPort // 8084

	// 检测 tsdd-server 的 API 端口（映射到容器内 5002 或 8090）
	output, err := client.ExecuteCommand(`cd /opt/tsdd && grep -A 30 'tsdd-server:' docker-compose.yml | grep -E '^\s+-\s+.+:(5002|8090)' | head -1 | grep -oP '(\d+):' | tr -d ':' || true`)
	if err == nil {
		output = strings.TrimSpace(output)
		if p := parsePort(output); p > 0 {
			apiPort = p
		}
	}

	// 检测 web 端口（映射到容器内 80）
	output, err = client.ExecuteCommand(`cd /opt/tsdd && grep -A 20 'tsdd-web:\|web:' docker-compose.yml | grep -E '^\s+-\s+\d+:80$' | head -1 | grep -oP '\d+:' | tr -d ':' || true`)
	if err == nil {
		output = strings.TrimSpace(output)
		if p := parsePort(output); p > 0 {
			webPort = p
		}
	}

	// 检测 manager 端口（映射到容器内 80，容器名含 manager）
	output, err = client.ExecuteCommand(`cd /opt/tsdd && grep -A 20 'manager:' docker-compose.yml | grep -E '^\s+-\s+\d+:80$' | head -1 | grep -oP '\d+:' | tr -d ':' || true`)
	if err == nil {
		output = strings.TrimSpace(output)
		if p := parsePort(output); p > 0 {
			managerPort = p
		}
	}

	logx.Infof("[CustomAMI] 检测到端口: API=%d, Web=%d, Manager=%d", apiPort, webPort, managerPort)
	return
}

// parsePort 安全地将字符串解析为端口号
func parsePort(s string) int {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	var port int
	for _, c := range s {
		if c >= '0' && c <= '9' {
			port = port*10 + int(c-'0')
		} else {
			break
		}
	}
	if port > 0 && port <= 65535 {
		return port
	}
	return 0
}

// updateCustomAMIConfig 自定义 AMI 部署：只做 IP 全局替换，完全保留原有配置（端口、镜像、挂载等）
// schemaDump: 可选的源数据库 schema dump（用于 fresh EBS 场景，避免 migration 冲突）
func updateCustomAMIConfig(client *utils.SSHClient, newIP string, schemaDump string) error {
	// ==================== Phase 1: IP 替换 + SSL 证书 ====================
	phase1Script := fmt.Sprintf(`
set -euo pipefail
cd /opt/tsdd

# 1. 从 .env 中提取旧 IP
OLD_IP=""
if [ -f .env ]; then
    OLD_IP=$(grep '^EXTERNAL_IP=' .env | cut -d= -f2 | tr -d '[:space:]')
fi
if [ -z "$OLD_IP" ]; then
    echo "WARNING: 无法从 .env 提取旧 IP，尝试从 docker-compose.yml 提取"
    OLD_IP=$(grep -oP '\d+\.\d+\.\d+\.\d+' docker-compose.yml | head -1 || true)
fi

NEW_IP="%s"
echo "IP 替换: $OLD_IP -> $NEW_IP"

if [ -n "$OLD_IP" ] && [ "$OLD_IP" != "$NEW_IP" ]; then
    for f in .env docker-compose.yml configs/hxd.yaml; do
        if [ -f "$f" ]; then
            sed -i "s|$OLD_IP|$NEW_IP|g" "$f"
            echo "Updated: $f"
        fi
    done
else
    if [ -f .env ]; then
        sed -i "s/^EXTERNAL_IP=.*/EXTERNAL_IP=$NEW_IP/" .env
    else
        echo "EXTERNAL_IP=$NEW_IP" > .env
    fi
    echo "EXTERNAL_IP set to $NEW_IP"
fi

# 2. 重新生成自签名 SSL 证书
if [ -d /opt/tsdd/ssl ]; then
    openssl req -x509 -nodes -days 3650 -newkey rsa:2048 \
        -keyout /opt/tsdd/ssl/server.key \
        -out /opt/tsdd/ssl/server.crt \
        -subj "/CN=$NEW_IP" \
        -addext "subjectAltName=IP:$NEW_IP" 2>/dev/null && \
    echo "SSL cert regenerated for $NEW_IP" || \
    echo "SSL cert generation failed (non-critical)"
fi

# 3. 清理 docker-compose.yml 中废弃的 version 字段
sed -i '/^version:/d' docker-compose.yml 2>/dev/null || true
echo "Phase 1 done"
`, newIP)

	output, err := client.ExecuteCommandWithTimeout(phase1Script, 2*time.Minute)
	if err != nil {
		return fmt.Errorf("Phase 1 (IP替换) 失败: %v, output: %s", err, output)
	}
	logx.Infof("[CustomAMI] Phase 1 完成: %s", output)

	// ==================== Phase 2: 启动基础设施 + 导入 schema ====================
	if schemaDump != "" {
		// Fresh EBS 场景：先启动 MySQL，导入 schema，再启动 tsdd-server
		logx.Info("[CustomAMI] Fresh EBS 模式：分阶段启动，先导入源数据库 schema...")

		// 2a. 启动基础设施（不启动 tsdd-server, web, manager）
		phase2a := `
cd /opt/tsdd
docker compose down 2>/dev/null || docker-compose down 2>/dev/null || true
docker compose up -d mysql redis minio wukongim 2>/dev/null || docker-compose up -d mysql redis minio wukongim 2>/dev/null

echo "Waiting for MySQL..."
for i in $(seq 1 60); do
    if docker exec tsdd-mysql mysqladmin ping -h localhost --silent 2>/dev/null; then
        echo "MySQL ready (attempt $i)"
        break
    fi
    [ "$i" -eq 60 ] && echo "WARNING: MySQL timeout"
    sleep 2
done
echo "Phase 2a done"
`
		output, err = client.ExecuteCommandWithTimeout(phase2a, 3*time.Minute)
		if err != nil {
			return fmt.Errorf("Phase 2a (启动MySQL) 失败: %v, output: %s", err, output)
		}
		logx.Infof("[CustomAMI] Phase 2a 完成: %s", output)

		// 2b. 上传 schema dump 并导入
		logx.Infof("[CustomAMI] 导入源数据库 schema (%d bytes)...", len(schemaDump))

		// 先写入文件到远程服务器
		writeCmd := fmt.Sprintf(`cat > /tmp/source_schema.sql << 'SCHEMA_DUMP_EOF'
%s
SCHEMA_DUMP_EOF
echo "Schema file written: $(wc -l < /tmp/source_schema.sql) lines"`, schemaDump)
		output, err = client.ExecuteCommandWithTimeout(writeCmd, 1*time.Minute)
		if err != nil {
			logx.Errorf("[CustomAMI] 写入 schema 文件失败: %v", err)
			// 不中断，继续尝试正常启动
		} else {
			logx.Infof("[CustomAMI] Schema 文件已写入: %s", output)

			// 导入 schema
			importCmd := `
cd /opt/tsdd
MYSQL_PASS=$(grep '^MYSQL_ROOT_PASSWORD=' .env | cut -d= -f2)

# 先清空数据库（可能有部分表）再导入完整 schema
docker exec tsdd-mysql bash -c "mysql -uroot -p\"$MYSQL_PASS\" -e 'DROP DATABASE IF EXISTS tsdd; CREATE DATABASE tsdd CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;'" 2>/dev/null

# 导入 schema + gorp_migrations
docker cp /tmp/source_schema.sql tsdd-mysql:/tmp/source_schema.sql
docker exec tsdd-mysql bash -c "mysql -uroot -p\"$MYSQL_PASS\" tsdd < /tmp/source_schema.sql" 2>/dev/null

# 验证
TABLES=$(docker exec tsdd-mysql bash -c "mysql -uroot -p\"$MYSQL_PASS\" tsdd -N -e 'SELECT COUNT(*) FROM information_schema.tables WHERE table_schema=\"tsdd\"'" 2>/dev/null)
MIGRATIONS=$(docker exec tsdd-mysql bash -c "mysql -uroot -p\"$MYSQL_PASS\" tsdd -N -e 'SELECT COUNT(*) FROM gorp_migrations'" 2>/dev/null)
echo "Schema imported: ${TABLES} tables, ${MIGRATIONS} migration records"
`
			output, err = client.ExecuteCommandWithTimeout(importCmd, 2*time.Minute)
			if err != nil {
				logx.Errorf("[CustomAMI] Schema 导入失败（非致命）: %v, output: %s", err, output)
			} else {
				logx.Infof("[CustomAMI] Schema 导入完成: %s", output)
			}
		}

		// 2c. 启动剩余服务
		phase2c := `
cd /opt/tsdd
docker compose up -d 2>/dev/null || docker-compose up -d 2>/dev/null
sleep 10
echo "=== Container Status ==="
docker ps --format "table {{.Names}}\t{{.Status}}\t{{.Image}}" | grep -E "tsdd|NAME"
`
		output, err = client.ExecuteCommandWithTimeout(phase2c, 3*time.Minute)
		if err != nil {
			return fmt.Errorf("Phase 2c (启动全部服务) 失败: %v, output: %s", err, output)
		}
		logx.Infof("[CustomAMI] Phase 2c 完成: %s", output)
	} else {
		// AMI 自带数据（无 fresh EBS）：直接重启所有服务
		phase2Script := `
cd /opt/tsdd
docker compose down 2>/dev/null || docker-compose down 2>/dev/null || true
docker compose up -d 2>/dev/null || docker-compose up -d 2>/dev/null
sleep 10
echo "=== Container Status ==="
docker ps --format "table {{.Names}}\t{{.Status}}\t{{.Image}}" | grep -E "tsdd|NAME"
`
		output, err = client.ExecuteCommandWithTimeout(phase2Script, 3*time.Minute)
		if err != nil {
			return fmt.Errorf("Phase 2 (重启服务) 失败: %v, output: %s", err, output)
		}
		logx.Infof("[CustomAMI] Phase 2 完成: %s", output)
	}

	// ==================== Phase 3: 统一 MySQL collation ====================
	// MySQL 8 默认 utf8mb4_0900_ai_ci，但原有表用 utf8mb4_unicode_ci
	// tsdd-server auto-migration 创建的新表会用 MySQL 默认 collation，导致 JOIN 冲突
	logx.Info("[CustomAMI] Phase 3: 统一 MySQL collation...")
	collationScript := "cd /opt/tsdd\n" +
		"MYSQL_PASS=$(grep '^MYSQL_ROOT_PASSWORD=' .env | cut -d= -f2)\n" +
		"for i in $(seq 1 30); do\n" +
		"    if docker exec tsdd-mysql mysqladmin ping -h localhost --silent 2>/dev/null; then break; fi\n" +
		"    sleep 2\n" +
		"done\n" +
		"docker exec tsdd-mysql bash -c \"mysql -uroot -p\\\"$MYSQL_PASS\\\" -e \\\"ALTER DATABASE tsdd CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;\\\"\" 2>/dev/null\n" +
		"docker exec tsdd-mysql bash -c \"mysql -uroot -p\\\"$MYSQL_PASS\\\" -e \\\"SET GLOBAL collation_server = 'utf8mb4_unicode_ci'; SET GLOBAL character_set_server = 'utf8mb4';\\\"\" 2>/dev/null\n" +
		"TABLES=$(docker exec tsdd-mysql bash -c \"mysql -uroot -p\\\"$MYSQL_PASS\\\" -N -e \\\"SELECT table_name FROM information_schema.tables WHERE table_schema='tsdd' AND table_collation != 'utf8mb4_unicode_ci';\\\"\" 2>/dev/null)\n" +
		"FIXED=0\n" +
		"for t in $TABLES; do\n" +
		"    docker exec tsdd-mysql bash -c \"mysql -uroot -p\\\"$MYSQL_PASS\\\" -e \\\"ALTER TABLE tsdd.\\`$t\\` CONVERT TO CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;\\\"\" 2>/dev/null && FIXED=$((FIXED+1))\n" +
		"done\n" +
		"echo \"Collation fixed: $FIXED tables converted to utf8mb4_unicode_ci\"\n"
	output, err = client.ExecuteCommandWithTimeout(collationScript, 2*time.Minute)
	if err != nil {
		logx.Errorf("[CustomAMI] Collation 统一失败（非致命）: %v, output: %s", err, output)
	} else {
		logx.Infof("[CustomAMI] Phase 3 完成: %s", output)
	}

	return nil
}

// updateTSDDConfig 通过已建立的 SSH 连接更新 TSDD 配置中的外网 IP（用于非自定义 AMI 场景）
func updateTSDDConfig(client *utils.SSHClient, newExternalIP string) error {
	// 更新 .env + configs/hxd.yaml + SSL 证书 + DB 迁移
	updateCmd := fmt.Sprintf(`
cd /opt/tsdd
# 备份原配置
cp .env .env.bak 2>/dev/null || true

# 更新 EXTERNAL_IP
if [ -f .env ]; then
    sed -i 's/^EXTERNAL_IP=.*/EXTERNAL_IP=%s/' .env
else
    echo "EXTERNAL_IP=%s" > .env
fi

# 更新 configs/hxd.yaml 中的外网 IP（替换所有旧 IP 为新 IP）
if [ -f configs/hxd.yaml ]; then
    # 提取当前配置中的 IP（从 external.ip 行）
    OLD_IP=$(grep '^\s*ip:' configs/hxd.yaml | head -1 | sed 's/.*"\(.*\)".*/\1/')
    if [ -n "$OLD_IP" ] && [ "$OLD_IP" != "%s" ]; then
        sed -i "s|$OLD_IP|%s|g" configs/hxd.yaml
        echo "Updated hxd.yaml IP: $OLD_IP -> %s"
    fi
    # 确保 baseURL 端口正确
    sed -i 's|baseURL: "http://%s:[0-9]*"|baseURL: "http://%s:8090"|' configs/hxd.yaml
    sed -i 's|webLoginURL: "http://%s:[0-9]*"|webLoginURL: "http://%s:82"|' configs/hxd.yaml
fi

# 重新生成自签名 SSL 证书（绑定新 IP）
if [ -d /opt/tsdd/ssl ]; then
    openssl req -x509 -nodes -days 3650 -newkey rsa:2048 \
        -keyout /opt/tsdd/ssl/server.key \
        -out /opt/tsdd/ssl/server.crt \
        -subj "/CN=%s" \
        -addext "subjectAltName=IP:%s" 2>/dev/null && \
    echo "SSL cert regenerated for %s" || \
    echo "SSL cert generation failed (non-critical)"
fi

# DB 迁移：添加自定义 HXD 表字段（如果不存在）
echo "Running DB migration..."
MYSQL_PASS=$(grep '^MYSQL_ROOT_PASSWORD=' .env | cut -d= -f2)
docker exec tsdd-mysql mysql -uroot -p"$MYSQL_PASS" tsdd -e "
ALTER TABLE app_config ADD COLUMN redpacket_on smallint NOT NULL DEFAULT 0 COMMENT '红包功能开关';
ALTER TABLE app_config ADD COLUMN checkin_on smallint NOT NULL DEFAULT 0 COMMENT '签到功能开关';
ALTER TABLE app_config ADD COLUMN checkin_reward_on smallint NOT NULL DEFAULT 0 COMMENT '签到奖金开关';
ALTER TABLE app_config ADD COLUMN checkin_base_reward decimal(10,2) NOT NULL DEFAULT 0 COMMENT '签到基础奖金';
ALTER TABLE app_config ADD COLUMN checkin_max_reward decimal(10,2) NOT NULL DEFAULT 0 COMMENT '签到最大连续奖金';
ALTER TABLE app_config ADD COLUMN checkin_increment decimal(10,2) NOT NULL DEFAULT 0 COMMENT '签到连续递增奖金';
ALTER TABLE app_config ADD COLUMN receipt_on smallint NOT NULL DEFAULT 0 COMMENT '已读回执全局开关';
ALTER TABLE app_config ADD COLUMN recommended_sites_on smallint NOT NULL DEFAULT 0 COMMENT '推荐网址开关';
" 2>&1 || echo "DB migration columns may already exist (OK)"

# 重启服务
docker compose down 2>/dev/null || docker-compose down 2>/dev/null || true
docker compose up -d 2>/dev/null || docker-compose up -d 2>/dev/null

# 等待服务启动
sleep 10
docker ps --format "table {{.Names}}\t{{.Status}}" | grep tsdd
`, newExternalIP, newExternalIP,
		newExternalIP, newExternalIP, newExternalIP,
		newExternalIP, newExternalIP,
		newExternalIP, newExternalIP,
		newExternalIP, newExternalIP, newExternalIP)

	output, err := client.ExecuteCommandWithTimeout(updateCmd, 3*time.Minute)
	if err != nil {
		return fmt.Errorf("更新配置失败: %v, output: %s", err, output)
	}

	logx.Infof("配置更新完成: %s", output)
	return nil
}

// ========== EBS 数据卷部署 ==========

// setupExtraEBSVolumes 创建、挂载、格式化额外的 EBS 数据卷
// 返回 (dbVolumeId, minioVolumeId, error)
func setupExtraEBSVolumes(acc *entity.CloudAccounts, req model.DeployTSDDWithAMIReq,
	instanceId, publicIP string) (string, string, error) {

	region := req.RegionId

	// 1. 获取实例可用区
	az, err := cloudaws.GetInstanceAZ(acc, region, instanceId)
	if err != nil {
		return "", "", fmt.Errorf("获取实例可用区失败: %v", err)
	}
	logx.Infof("[EBS] 实例 %s 位于可用区: %s", instanceId, az)

	// 2. 设置默认值
	dbSize := req.DBVolumeSizeGiB
	if dbSize == 0 {
		dbSize = 20
	}
	dbIOPS := req.DBVolumeIOPS
	if dbIOPS == 0 {
		dbIOPS = 3000
	}
	minioSize := req.MinioVolumeSizeGiB
	if minioSize == 0 {
		minioSize = 50
	}
	minioIOPS := req.MinioVolumeIOPS
	if minioIOPS == 0 {
		minioIOPS = 3000
	}
	serverName := req.ServerName
	if serverName == "" {
		serverName = "TSDD"
	}

	// 3. 创建两块数据卷
	logx.Infof("[EBS] 创建DB数据卷: %dGiB, %d IOPS", dbSize, dbIOPS)
	dbVolId, err := cloudaws.CreateEBSVolume(acc, region, az, dbSize, dbIOPS,
		fmt.Sprintf("%s-db-data", serverName))
	if err != nil {
		return "", "", fmt.Errorf("创建DB数据卷失败: %v", err)
	}
	logx.Infof("[EBS] DB数据卷已创建: %s", dbVolId)

	logx.Infof("[EBS] 创建MinIO数据卷: %dGiB, %d IOPS", minioSize, minioIOPS)
	minioVolId, err := cloudaws.CreateEBSVolume(acc, region, az, minioSize, minioIOPS,
		fmt.Sprintf("%s-minio-data", serverName))
	if err != nil {
		return dbVolId, "", fmt.Errorf("创建MinIO数据卷失败: %v", err)
	}
	logx.Infof("[EBS] MinIO数据卷已创建: %s", minioVolId)

	// 4. 等待卷可用
	if err := cloudaws.WaitForVolumeAvailable(acc, region, dbVolId, 2*time.Minute); err != nil {
		return dbVolId, minioVolId, fmt.Errorf("等待DB数据卷可用超时: %v", err)
	}
	if err := cloudaws.WaitForVolumeAvailable(acc, region, minioVolId, 2*time.Minute); err != nil {
		return dbVolId, minioVolId, fmt.Errorf("等待MinIO数据卷可用超时: %v", err)
	}

	// 5. 挂载卷到实例
	logx.Info("[EBS] 挂载DB数据卷到 /dev/xvdb")
	if err := cloudaws.AttachEBSVolume(acc, region, dbVolId, instanceId, "/dev/xvdb"); err != nil {
		return dbVolId, minioVolId, fmt.Errorf("挂载DB数据卷失败: %v", err)
	}
	logx.Info("[EBS] 挂载MinIO数据卷到 /dev/xvdc")
	if err := cloudaws.AttachEBSVolume(acc, region, minioVolId, instanceId, "/dev/xvdc"); err != nil {
		return dbVolId, minioVolId, fmt.Errorf("挂载MinIO数据卷失败: %v", err)
	}

	// 6. 等待挂载完成
	if err := cloudaws.WaitForVolumeAttached(acc, region, dbVolId, 2*time.Minute); err != nil {
		return dbVolId, minioVolId, fmt.Errorf("等待DB数据卷挂载完成超时: %v", err)
	}
	if err := cloudaws.WaitForVolumeAttached(acc, region, minioVolId, 2*time.Minute); err != nil {
		return dbVolId, minioVolId, fmt.Errorf("等待MinIO数据卷挂载完成超时: %v", err)
	}

	// 7. SSH 格式化和挂载
	logx.Info("[EBS] SSH 格式化和挂载磁盘...")
	if err := sshFormatAndMountEBS(publicIP); err != nil {
		return dbVolId, minioVolId, fmt.Errorf("SSH格式化挂载失败: %v", err)
	}

	// 8. 更新 docker-compose.yml
	logx.Info("[EBS] 更新 docker-compose.yml 使用宿主机路径...")
	if err := sshUpdateDockerComposeVolumes(publicIP); err != nil {
		return dbVolId, minioVolId, fmt.Errorf("更新docker-compose失败: %v", err)
	}

	return dbVolId, minioVolId, nil
}

// sshFormatAndMountEBS SSH 到实例执行格式化、挂载、写 fstab
func sshFormatAndMountEBS(host string) error {
	client := &utils.SSHClient{
		Host:     host,
		Port:     22,
		Username: "root",
		Password: consts.DefaultPassword,
	}

	// Nitro 实例上 /dev/xvdb 可能显示为 /dev/nvme1n1，脚本自动处理
	script := `
set -euo pipefail

# 解析设备名（兼容 NVMe）
resolve_dev() {
    local DEV="$1"
    if [ -b "$DEV" ]; then echo "$DEV"; return; fi
    local IDX
    case "$DEV" in
        /dev/xvdb) IDX=1 ;;
        /dev/xvdc) IDX=2 ;;
        *) echo "$DEV"; return ;;
    esac
    local NVME="/dev/nvme${IDX}n1"
    if [ -b "$NVME" ]; then echo "$NVME"; return; fi
    sleep 5
    if [ -b "$DEV" ]; then echo "$DEV"; return; fi
    if [ -b "$NVME" ]; then echo "$NVME"; return; fi
    echo "ERROR: device not found for $DEV" >&2
    return 1
}

DB_DEV=$(resolve_dev /dev/xvdb)
MINIO_DEV=$(resolve_dev /dev/xvdc)
echo "DB device: $DB_DEV"
echo "MinIO device: $MINIO_DEV"

mkdir -p /data/db /data/minio

# 格式化（仅无文件系统时）
if ! blkid "$DB_DEV" | grep -q TYPE; then
    mkfs.ext4 -F "$DB_DEV"
    echo "Formatted $DB_DEV as ext4"
fi
if ! blkid "$MINIO_DEV" | grep -q TYPE; then
    mkfs.ext4 -F "$MINIO_DEV"
    echo "Formatted $MINIO_DEV as ext4"
fi

mount "$DB_DEV" /data/db
mount "$MINIO_DEV" /data/minio

# 用 UUID 写 fstab（更可靠）
DB_UUID=$(blkid -s UUID -o value "$DB_DEV")
MINIO_UUID=$(blkid -s UUID -o value "$MINIO_DEV")

grep -q "$DB_UUID" /etc/fstab 2>/dev/null || echo "UUID=$DB_UUID /data/db ext4 defaults,nofail 0 2" >> /etc/fstab
grep -q "$MINIO_UUID" /etc/fstab 2>/dev/null || echo "UUID=$MINIO_UUID /data/minio ext4 defaults,nofail 0 2" >> /etc/fstab

echo "EBS volumes mounted successfully"
df -h /data/db /data/minio
`

	output, err := client.ExecuteCommandWithTimeout(script, 3*time.Minute)
	if err != nil {
		return fmt.Errorf("格式化/挂载EBS失败: %v, output: %s", err, output)
	}
	logx.Infof("[EBS] 格式化/挂载完成: %s", output)
	return nil
}

// sshUpdateDockerComposeVolumes 更新 docker-compose.yml 把 named volumes 替换为宿主机路径
func sshUpdateDockerComposeVolumes(host string) error {
	client := &utils.SSHClient{
		Host:     host,
		Port:     22,
		Username: "root",
		Password: consts.DefaultPassword,
	}

	script := `
set -euo pipefail
cd /opt/tsdd

# 先停止服务
docker compose down 2>/dev/null || docker-compose down 2>/dev/null || true

# 创建数据目录
mkdir -p /data/db/mysql /data/db/redis /data/db/wukongim /data/minio/data

# 替换 named volumes 为宿主机路径
sed -i 's|^\(\s*-\s*\)mysql_data:/var/lib/mysql|\1/data/db/mysql:/var/lib/mysql|' docker-compose.yml
sed -i 's|^\(\s*-\s*\)redis_data:/data|\1/data/db/redis:/data|' docker-compose.yml
sed -i 's|^\(\s*-\s*\)minio_data:/data|\1/data/minio/data:/data|' docker-compose.yml
sed -i 's|^\(\s*-\s*\)wukongim_data:/root/wukongim|\1/data/db/wukongim:/root/wukongim|' docker-compose.yml

# 删除底部的 named volumes 声明
# 匹配 "^volumes:" 到文件末尾的 volume 声明行
sed -i '/^volumes:$/,$ { /^volumes:$/d; /^  [a-z_]*:$/d; }' docker-compose.yml

echo "=== docker-compose.yml volume mappings ==="
grep -n "/data/" docker-compose.yml || echo "No host path volumes found"
`

	output, err := client.ExecuteCommandWithTimeout(script, 3*time.Minute)
	if err != nil {
		return fmt.Errorf("更新docker-compose失败: %v, output: %s", err, output)
	}
	logx.Infof("[EBS] docker-compose更新完成: %s", output)
	return nil
}

// sshUploadAndSetupResources 上传 TSDD 服务器资源（修复后的二进制 + assets 默认头像）
// 资源包存放在 /opt/control/tsdd-resources.tar.gz，包含:
//   - assets/assets/*.png  (默认头像文件)
//   - TangSengDaoDaoServer (修复后的服务器二进制)
//
// 如果资源包不存在则跳过（非致命），AMI 可能已经包含这些资源
func sshUploadAndSetupResources(client *utils.SSHClient) error {
	const resourcesPath = "/opt/control/tsdd-resources.tar.gz"

	f, err := os.Open(resourcesPath)
	if err != nil {
		logx.Infof("[Deploy] 资源包 %s 未找到，跳过资源上传", resourcesPath)
		return nil
	}
	defer f.Close()

	// 上传资源包
	logx.Info("[Deploy] 上传TSDD资源包...")
	if err := client.UploadFile("/tmp/tsdd-resources.tar.gz", f); err != nil {
		return fmt.Errorf("上传资源包失败: %v", err)
	}

	// 解压并配置 docker-compose 挂载
	script := `
set -euo pipefail
cd /opt/tsdd

# 解压资源包
tar xzf /tmp/tsdd-resources.tar.gz
chmod +x /opt/tsdd/TangSengDaoDaoServer 2>/dev/null || true
rm -f /tmp/tsdd-resources.tar.gz

# 确保 configs 目录存在
mkdir -p /opt/tsdd/configs

# 如果 configs/hxd.yaml 不存在，生成默认配置
if [ ! -f /opt/tsdd/configs/hxd.yaml ]; then
    EXTERNAL_IP=$(grep '^EXTERNAL_IP=' .env 2>/dev/null | cut -d= -f2 || echo "0.0.0.0")
    MYSQL_PASS=$(grep '^MYSQL_ROOT_PASSWORD=' .env 2>/dev/null | cut -d= -f2 || echo "TsddSecure2024!")
    MINIO_USER=$(grep '^MINIO_ROOT_USER=' .env 2>/dev/null | cut -d= -f2 || echo "admin")
    MINIO_PASS=$(grep '^MINIO_ROOT_PASSWORD=' .env 2>/dev/null | cut -d= -f2 || echo "TsddMinio2024!")
    cat > /opt/tsdd/configs/hxd.yaml << CFGEOF
mode: "release"
addr: ":8090"
grpcAddr: "0.0.0.0:6979"
rootDir: "tsdddata"
appName: "唐僧叨叨"
messageSaveAcrossDevice: true
onlineStatusOn: true
onlineStatusOnForMember: true
onlineStatusOnForRegular: true
groupUpgradeWhenMemberCount: 1000
eventPoolSize: 100
wukongIM:
  apiURL: "http://wukongim:5001"
db:
  mysqlAddr: "root:${MYSQL_PASS}@tcp(mysql:3306)/tsdd?charset=utf8mb4&parseTime=true&loc=Local"
  redisAddr: "redis:6379"
  redisPass: ""
external:
  ip: "${EXTERNAL_IP}"
  baseURL: "http://${EXTERNAL_IP}:8090"
  webLoginURL: "http://${EXTERNAL_IP}:82"
smsCode: "123456"
logger:
  level: 2
  dir: "./logs"
  lineNum: false
fileService: "minio"
minio:
  url: "http://minio:9000"
  accessKeyID: "${MINIO_USER}"
  secretAccessKey: "${MINIO_PASS}"
CFGEOF
    echo "Generated configs/hxd.yaml"
fi

# 添加 volume 挂载到 tsdd-server（如果还没有）
if ! grep -q '/home/configs/hxd.yaml' docker-compose.yml && ! grep -q '/home/assets' docker-compose.yml; then
    python3 -c "
path = 'docker-compose.yml'
with open(path) as f:
    content = f.read()

# 在 condition: service_started 后面插入 volumes
old = 'condition: service_started\n'
new = old + '    volumes:\n      - ./configs/hxd.yaml:/home/configs/hxd.yaml:ro\n      - ./assets:/home/assets:ro\n      - ./TangSengDaoDaoServer:/home/app\n'

if old in content:
    content = content.replace(old, new, 1)
    with open(path, 'w') as f:
        f.write(content)
    print('Volume mounts added')
else:
    print('WARNING: insertion point not found')
"
fi

echo "=== Resources setup ==="
ls -la /opt/tsdd/assets/assets/ 2>/dev/null && echo "Assets OK" || echo "No assets"
ls -lh /opt/tsdd/TangSengDaoDaoServer 2>/dev/null && echo "Binary OK" || echo "No binary"
ls -la /opt/tsdd/configs/hxd.yaml 2>/dev/null && echo "Config OK" || echo "No config"
grep '/home/configs/hxd.yaml\|/home/assets\|/home/app' docker-compose.yml && echo "Mount OK" || echo "No mount"
`
	output, err := client.ExecuteCommandWithTimeout(script, 5*time.Minute)
	if err != nil {
		return fmt.Errorf("资源部署失败: %v, output: %s", err, output)
	}
	logx.Infof("[Deploy] 资源部署完成: %s", output)
	return nil
}
