package deploy

import (
	"fmt"
	awscloud "server/internal/server/cloud/aws"
	"server/internal/server/model"
	cloudaws "server/internal/server/service/cloud_aws"
	"server/internal/server/utils"
	"server/pkg/consts"
	"server/pkg/dbs"
	"server/pkg/entity"
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

// TSDD 默认 AMI ID（按区域）
var DefaultTSDDAMI = map[string]string{
	"ap-east-1": "ami-0e26671de387eded1", // 香港
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
			// 使用默认 AMI
			amiId = DefaultTSDDAMI[req.RegionId]
			if amiId == "" {
				return resp, fmt.Errorf("该区域 %s 没有预置的 TSDD AMI，请指定 ami_id 或 source_server_id", req.RegionId)
			}
		}
	}

	// 设置默认值
	instanceType := req.InstanceType
	if instanceType == "" {
		instanceType = "t3.medium"
	}
	volumeSize := req.VolumeSizeGiB
	if volumeSize == 0 {
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

	// SSH 连接并更新配置
	logx.Info("SSH 连接更新配置...")
	err = updateTSDDConfig(publicIP, publicIP)
	if err != nil {
		logx.Errorf("更新配置失败: %v", err)
		resp.Message = fmt.Sprintf("实例已创建，但配置更新失败: %v", err)
	}

	// 注册服务器到数据库
	newServer := &entity.Servers{
		MerchantId: req.MerchantId,
		Name:       serverName,
		Host:       publicIP,
		Port:       22,
		Username:   "root",
		Password:   consts.DefaultPassword,
		Status:     1,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	if _, err := dbs.DBAdmin.Insert(newServer); err != nil {
		logx.Errorf("注册服务器失败: %v", err)
	} else {
		resp.ServerId = newServer.Id
	}

	// 设置响应
	config := model.DefaultDeployConfig
	resp.Success = true
	resp.Message = "部署成功"
	resp.APIUrl = fmt.Sprintf("http://%s:%d", publicIP, config.APIPort)
	resp.WebUrl = fmt.Sprintf("http://%s:%d", publicIP, config.WebPort)
	resp.AdminUrl = fmt.Sprintf("http://%s:%d", publicIP, config.ManagerPort)

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

// updateTSDDConfig 通过 SSH 更新 TSDD 配置中的外网 IP
func updateTSDDConfig(host, newExternalIP string) error {
	client := &utils.SSHClient{
		Host:     host,
		Port:     22,
		Username: "root",
		Password: consts.DefaultPassword,
	}

	// 等待 SSH 可用
	var lastErr error
	for i := 0; i < 12; i++ { // 最多等待 2 分钟
		_, err := client.ExecuteCommand("echo 'SSH ready'")
		if err == nil {
			break
		}
		lastErr = err
		time.Sleep(10 * time.Second)
	}
	if lastErr != nil {
		return fmt.Errorf("SSH 连接失败: %v", lastErr)
	}

	// 更新 .env 文件中的 EXTERNAL_IP
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

# 重启服务
docker compose down 2>/dev/null || docker-compose down 2>/dev/null || true
docker compose up -d 2>/dev/null || docker-compose up -d 2>/dev/null

# 等待服务启动
sleep 10
docker ps --format "table {{.Names}}\t{{.Status}}" | grep tsdd
`, newExternalIP, newExternalIP)

	output, err := client.ExecuteCommandWithTimeout(updateCmd, 3*time.Minute)
	if err != nil {
		return fmt.Errorf("更新配置失败: %v, output: %s", err, output)
	}

	logx.Infof("配置更新完成: %s", output)
	return nil
}
