package deploy

import (
	"fmt"
	"server/internal/server/model"
	"server/internal/server/utils"
	"server/pkg/gostapi"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

// SetupNode 初始化节点：安装 Docker、挂载磁盘、创建目录
func SetupNode(client *utils.SSHClient, config model.DeployConfig) []model.DeployStep {
	steps := make([]model.DeployStep, 0)

	// Step 1: 安装 Docker
	step1 := installDocker(client)
	steps = append(steps, step1)
	if step1.Status == "failed" {
		return steps
	}

	// Step 2: 挂载数据盘（检测并自动格式化/挂载未使用的磁盘）
	step2 := setupDataDisks(client, config)
	steps = append(steps, step2)

	// Step 3: 创建工作目录
	step3 := createDirectories(client, config)
	steps = append(steps, step3)

	return steps
}

// setupDataDisks 自动检测和挂载数据盘
func setupDataDisks(client *utils.SSHClient, config model.DeployConfig) model.DeployStep {
	step := model.DeployStep{
		Name:   "挂载数据盘",
		Status: "running",
	}

	// 根据角色决定挂载点
	var mountScript string
	switch config.NodeRole {
	case "db":
		mountScript = setupDBDisks()
	case "minio":
		mountScript = setupMinioDisks()
	case "app":
		mountScript = setupAppDisks()
	default: // allinone
		mountScript = setupAllinoneDisks()
	}

	output, err := client.ExecuteCommandWithTimeout(mountScript, 2*time.Minute)
	if err != nil {
		step.Status = "warning"
		step.Message = fmt.Sprintf("磁盘挂载部分失败: %v", err)
		step.Output = output
		return step
	}

	step.Status = "success"
	step.Message = "数据盘挂载完成"
	step.Output = output
	return step
}

// setupDBDisks DB 节点磁盘：/data/db (MySQL+Redis)
func setupDBDisks() string {
	return `#!/bin/bash
set -e
echo "=== 检测未挂载的磁盘 ==="

# 收集未挂载的 nvme/xvd 磁盘（排除系统盘和已挂载的）
UNATTACHED=()
for disk in /dev/nvme?n1 /dev/xvd?; do
    [ -b "$disk" ] || continue
    # 跳过系统盘（有分区的通常是系统盘）
    if lsblk -n "$disk" | grep -q "part"; then
        continue
    fi
    # 跳过已挂载的
    if mount | grep -q "^$disk "; then
        echo "$disk 已挂载，跳过"
        continue
    fi
    UNATTACHED+=("$disk")
done

echo "发现 ${#UNATTACHED[@]} 个未挂载磁盘: ${UNATTACHED[*]}"

# DB 节点需要 1 个数据盘：/data/db
if [ ${#UNATTACHED[@]} -ge 1 ]; then
    disk="${UNATTACHED[0]}"
    mount_point="/data/db"

    if ! blkid "$disk" | grep -q "TYPE"; then
        echo "格式化 $disk 为 ext4..."
        mkfs.ext4 -F "$disk"
    fi

    mkdir -p "$mount_point"
    if ! mount | grep -q "$mount_point"; then
        mount "$disk" "$mount_point"
        echo "$disk 已挂载到 $mount_point"
    fi

    UUID=$(blkid -s UUID -o value "$disk")
    if ! grep -q "$UUID" /etc/fstab 2>/dev/null; then
        echo "UUID=$UUID $mount_point ext4 defaults,nofail 0 2" >> /etc/fstab
        echo "已添加到 fstab: $UUID -> $mount_point"
    fi
fi

echo "=== 磁盘状态 ==="
df -h /data/db 2>/dev/null || echo "/data/db 挂载点不存在"
`
}

// setupMinioDisks MinIO 节点磁盘：/data/minio
func setupMinioDisks() string {
	return `#!/bin/bash
set -e
echo "=== 检测未挂载的磁盘 ==="

UNATTACHED=()
for disk in /dev/nvme?n1 /dev/xvd?; do
    [ -b "$disk" ] || continue
    if lsblk -n "$disk" | grep -q "part"; then
        continue
    fi
    if mount | grep -q "^$disk "; then
        echo "$disk 已挂载，跳过"
        continue
    fi
    UNATTACHED+=("$disk")
done

echo "发现 ${#UNATTACHED[@]} 个未挂载磁盘: ${UNATTACHED[*]}"

# MinIO 节点需要 1 个数据盘：/data/minio
if [ ${#UNATTACHED[@]} -ge 1 ]; then
    disk="${UNATTACHED[0]}"
    mount_point="/data/minio"

    if ! blkid "$disk" | grep -q "TYPE"; then
        echo "格式化 $disk 为 ext4..."
        mkfs.ext4 -F "$disk"
    fi

    mkdir -p "$mount_point"
    if ! mount | grep -q "$mount_point"; then
        mount "$disk" "$mount_point"
        echo "$disk 已挂载到 $mount_point"
    fi

    UUID=$(blkid -s UUID -o value "$disk")
    if ! grep -q "$UUID" /etc/fstab 2>/dev/null; then
        echo "UUID=$UUID $mount_point ext4 defaults,nofail 0 2" >> /etc/fstab
        echo "已添加到 fstab: $UUID -> $mount_point"
    fi
fi

echo "=== 磁盘状态 ==="
df -h /data/minio 2>/dev/null || echo "/data/minio 挂载点不存在"
`
}

// setupAppDisks App 节点磁盘：/data/db/wukongim (PebbleDB)
func setupAppDisks() string {
	return `#!/bin/bash
set -e
echo "=== 检测未挂载的磁盘 ==="

UNATTACHED=()
for disk in /dev/nvme?n1 /dev/xvd?; do
    [ -b "$disk" ] || continue
    if lsblk -n "$disk" | grep -q "part"; then
        continue
    fi
    if mount | grep -q "^$disk "; then
        echo "$disk 已挂载，跳过"
        continue
    fi
    UNATTACHED+=("$disk")
done

echo "发现 ${#UNATTACHED[@]} 个未挂载磁盘: ${UNATTACHED[*]}"

# App 节点只需要 1 个数据盘：/data/db (WuKongIM PebbleDB)
if [ ${#UNATTACHED[@]} -ge 1 ]; then
    disk="${UNATTACHED[0]}"
    mount_point="/data/db"

    if ! blkid "$disk" | grep -q "TYPE"; then
        echo "格式化 $disk 为 ext4..."
        mkfs.ext4 -F "$disk"
    fi

    mkdir -p "$mount_point"
    if ! mount | grep -q "$mount_point"; then
        mount "$disk" "$mount_point"
        echo "$disk 已挂载到 $mount_point"
    fi

    UUID=$(blkid -s UUID -o value "$disk")
    if ! grep -q "$UUID" /etc/fstab 2>/dev/null; then
        echo "UUID=$UUID $mount_point ext4 defaults,nofail 0 2" >> /etc/fstab
    fi
fi

echo "=== 磁盘状态 ==="
df -h /data/db 2>/dev/null || echo "/data/db 挂载点不存在"
`
}

// setupAllinoneDisks 全量节点磁盘：/data/db 和 /data/minio
func setupAllinoneDisks() string {
	return setupDBDisks() // 与 DB 节点相同
}

// createDirectories 创建工作目录
func createDirectories(client *utils.SSHClient, config model.DeployConfig) model.DeployStep {
	step := model.DeployStep{
		Name:   "创建工作目录",
		Status: "running",
	}

	dirs := "mkdir -p /opt/tsdd /data/db"
	switch config.NodeRole {
	case "db":
		dirs = "mkdir -p /opt/tsdd /data/db/mysql /data/db/redis"
	case "minio":
		dirs = "mkdir -p /opt/tsdd /data/minio"
	case "app":
		dirs = "mkdir -p /opt/tsdd /opt/tsdd/assets /opt/tsdd/ssl /opt/tsdd/web /opt/tsdd/tsdddata /opt/tsdd/configs /opt/tsdd/nginx /opt/tsdd/manager /data/db/wukongim && chown -R 1000:1000 /opt/tsdd/tsdddata"
	default:
		dirs = "mkdir -p /opt/tsdd /opt/tsdd/assets /opt/tsdd/ssl /opt/tsdd/web /data/db/mysql /data/db/redis /data/db/wukongim /data/minio"
	}

	output, err := client.ExecuteCommand(dirs)
	if err != nil {
		step.Status = "failed"
		step.Message = fmt.Sprintf("创建目录失败: %v", err)
		step.Output = output
		return step
	}

	step.Status = "success"
	step.Message = "目录创建完成"
	step.Output = output
	return step
}

// DeployNode 完整的节点部署流程
func DeployNode(client *utils.SSHClient, config model.DeployConfig, forceReset bool) model.DeployTSDDResp {
	var resp model.DeployTSDDResp
	resp.Steps = make([]model.DeployStep, 0)

	// Step 1-3: 初始化节点
	setupSteps := SetupNode(client, config)
	resp.Steps = append(resp.Steps, setupSteps...)
	for _, s := range setupSteps {
		if s.Status == "failed" {
			resp.Success = false
			resp.Message = fmt.Sprintf("%s失败", s.Name)
			return resp
		}
	}

	// Step 4: 如果强制重置，清理现有容器
	if forceReset {
		step := cleanupExisting(client)
		resp.Steps = append(resp.Steps, step)
	}

	// Step 5: 生成配置文件
	composeContent := GenerateComposeByRole(config)
	envContent := GenerateEnvByRole(config)

	writeCmd := fmt.Sprintf(`cat > /opt/tsdd/docker-compose.yml << 'COMPOSE_EOF'
%s
COMPOSE_EOF

cat > /opt/tsdd/.env << 'ENV_EOF'
%s
ENV_EOF

echo "配置文件已写入"
`, composeContent, envContent)

	// App 节点额外写入 nginx.conf + tsdd-config.js + hxd-config.js
	if config.NodeRole == "app" || config.NodeRole == "allinone" {
		nginxConf := generateNginxConf(config)
		webConfig := generateWebConfig(config)
		managerConfig := generateManagerConfig()
		writeCmd += fmt.Sprintf(`
mkdir -p /opt/tsdd/manager
cat > /opt/tsdd/nginx.conf << 'NGINX_EOF'
%s
NGINX_EOF

cat > /opt/tsdd/web/tsdd-config.js << 'WEBCONF_EOF'
%s
WEBCONF_EOF

cat > /opt/tsdd/manager/hxd-config.js << 'MGRCONF_EOF'
%s
MGRCONF_EOF
echo "nginx.conf + tsdd-config.js + hxd-config.js 已写入"
`, nginxConf, webConfig, managerConfig)
	}

	output, err := client.ExecuteCommand(writeCmd)
	configStep := model.DeployStep{
		Name:   "生成配置文件",
		Output: output,
	}
	if err != nil {
		configStep.Status = "failed"
		configStep.Message = fmt.Sprintf("写入配置失败: %v", err)
		resp.Steps = append(resp.Steps, configStep)
		resp.Success = false
		resp.Message = "生成配置文件失败"
		return resp
	}
	configStep.Status = "success"
	configStep.Message = fmt.Sprintf("已生成 %s 模式配置", config.NodeRole)
	resp.Steps = append(resp.Steps, configStep)

	// Step 6: 拉取镜像并启动
	startStep := pullAndStartServices(client)
	resp.Steps = append(resp.Steps, startStep)
	if startStep.Status == "failed" {
		resp.Success = false
		resp.Message = "启动服务失败"
		return resp
	}

	// Step 7: 健康检查
	healthStep := checkNodeHealth(client, config)
	resp.Steps = append(resp.Steps, healthStep)

	// Step 8: App 节点自动安装商户端 GOST（与单机模式统一）
	if config.NodeRole == "app" {
		gostStep := installMerchantGost(client, config)
		resp.Steps = append(resp.Steps, gostStep)
		if gostStep.Status == "failed" {
			// GOST 安装失败不阻塞整体部署，只记录警告
			logx.Errorf("[DeployNode] 商户端 GOST 安装失败: %s", gostStep.Message)
		}
	}

	resp.Success = true
	resp.Message = fmt.Sprintf("%s 节点部署完成", config.NodeRole)

	if config.NodeRole != "db" && config.NodeRole != "minio" {
		resp.APIUrl = fmt.Sprintf("http://%s/api/", config.ExternalIP)
		resp.WebUrl = fmt.Sprintf("http://%s", config.ExternalIP)
		resp.AdminUrl = fmt.Sprintf("http://%s/hxdadmin/", config.ExternalIP)
	}

	return resp
}

// checkNodeHealth 根据角色检查健康状态
func checkNodeHealth(client *utils.SSHClient, config model.DeployConfig) model.DeployStep {
	step := model.DeployStep{
		Name:   "检查服务健康状态",
		Status: "running",
	}

	time.Sleep(8 * time.Second)

	var checkCmd string
	switch config.NodeRole {
	case "db":
		checkCmd = `
echo "=== 容器状态 ==="
docker ps --format "table {{.Names}}\t{{.Status}}" | grep -E "imchat|tsdd"
echo ""
echo "=== MySQL ==="
docker exec imchat-mysql mysqladmin ping -h localhost 2>/dev/null && echo "MySQL: OK" || docker exec tsdd-mysql mysqladmin ping -h localhost 2>/dev/null && echo "MySQL: OK" || echo "MySQL: FAIL"
echo "=== Redis ==="
docker exec imchat-redis redis-cli ping 2>/dev/null || docker exec tsdd-redis redis-cli ping 2>/dev/null || echo "Redis: FAIL"
`
	case "minio":
		checkCmd = `
echo "=== 容器状态 ==="
docker ps --format "table {{.Names}}\t{{.Status}}" | grep -E "imchat|tsdd"
echo ""
echo "=== MinIO ==="
curl -sf http://localhost:9000/minio/health/live >/dev/null && echo "MinIO: OK" || echo "MinIO: FAIL"
`
	case "app":
		checkCmd = fmt.Sprintf(`
echo "=== 容器状态 ==="
docker ps --format "table {{.Names}}\t{{.Status}}" | grep -E "imchat|tsdd"
echo ""
echo "=== WuKongIM ==="
curl -sf http://127.0.0.1:5002/health >/dev/null && echo "WuKongIM: OK" || echo "WuKongIM: 启动中..."
echo "=== tsdd-server ==="
curl -sf http://127.0.0.1:%d/v1/health >/dev/null && echo "tsdd-server: OK" || echo "tsdd-server: 启动中..."
`, config.APIPort)
		if config.WKNodeId > 0 {
			checkCmd += `
echo "=== WuKongIM 集群 ==="
curl -sf http://127.0.0.1:5002/cluster/nodes 2>/dev/null | python3 -c "import json,sys; d=json.load(sys.stdin); print(f'集群节点数: {len(d.get(\"nodes\",d.get(\"data\",{}).get(\"nodes\",[])))}') if isinstance(d,dict) else print('解析失败')" 2>/dev/null || echo "集群信息获取中..."
`
		}
	default: // allinone
		checkCmd = fmt.Sprintf(`
echo "=== 容器状态 ==="
docker ps --format "table {{.Names}}\t{{.Status}}" | grep -E "imchat|tsdd"
echo ""
echo "=== 各服务健康检查 ==="
docker exec imchat-mysql mysqladmin ping -h localhost 2>/dev/null && echo "MySQL: OK" || docker exec tsdd-mysql mysqladmin ping -h localhost 2>/dev/null && echo "MySQL: OK" || echo "MySQL: FAIL"
docker exec imchat-redis redis-cli ping 2>/dev/null || docker exec tsdd-redis redis-cli ping 2>/dev/null || echo "Redis: FAIL"
curl -sf http://localhost:9000/minio/health/live >/dev/null && echo "MinIO: OK" || echo "MinIO: FAIL"
curl -sf http://127.0.0.1:5002/health >/dev/null && echo "WuKongIM: OK" || echo "WuKongIM: 启动中..."
curl -sf http://127.0.0.1:%d/v1/health >/dev/null && echo "tsdd-server: OK" || echo "tsdd-server: 启动中..."
`, config.APIPort)
	}

	output, _ := client.ExecuteCommand(checkCmd)

	if output == "" || containsAny(output, "FAIL") {
		step.Status = "warning"
		step.Message = "部分服务可能还在启动中"
	} else {
		step.Status = "success"
		step.Message = "所有服务运行正常"
	}
	step.Output = output
	return step
}

func containsAny(s string, substrs ...string) bool {
	for _, sub := range substrs {
		if len(sub) > 0 {
			for i := 0; i <= len(s)-len(sub); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
		}
	}
	return false
}

// installMerchantGost 在 App 节点安装商户端 GOST 并配置本地转发
// 与单机模式统一：GOST 监听 10443/10080/10800，转发到本地业务端口
func installMerchantGost(client *utils.SSHClient, config model.DeployConfig) model.DeployStep {
	step := model.DeployStep{
		Name:   "安装商户端 GOST",
		Status: "running",
	}

	// 1. 检查是否已安装
	output, _ := client.ExecuteCommand("which gost 2>/dev/null || echo ''")
	gostInstalled := len(output) > 0 && output != "\n"

	if !gostInstalled {
		// 尝试使用预存二进制上传
		if err := uploadGostBinary(client.Client, func(msg string) {}); err != nil {
			// 回退：从 GitHub 下载
			downloadCmd := `
GOST_VERSION="3.0.0-rc10"
cd /tmp
wget -q --timeout=60 "https://github.com/go-gost/gost/releases/download/v${GOST_VERSION}/gost_${GOST_VERSION}_linux_amd64.tar.gz" -O gost.tar.gz
tar -xzf gost.tar.gz
sudo mv gost /usr/local/bin/ && sudo chmod +x /usr/local/bin/gost
rm -f gost.tar.gz
`
			if out, err := client.ExecuteCommand(downloadCmd); err != nil {
				step.Status = "failed"
				step.Message = fmt.Sprintf("GOST 安装失败: %v, %s", err, out)
				return step
			}
		} else {
			// 上传成功，移动到 /usr/local/bin
			if out, err := client.ExecuteCommand("sudo mv /tmp/gost /usr/local/bin/ && sudo chmod +x /usr/local/bin/gost"); err != nil {
				step.Status = "failed"
				step.Message = fmt.Sprintf("GOST 移动失败: %v, %s", err, out)
				return step
			}
		}
	}

	// 2. 创建配置和 systemd 服务（与单机模式统一）
	// MinIO 地址：多机模式从 config.MinioHost 获取，单机默认 127.0.0.1
	minioAddr := fmt.Sprintf("127.0.0.1:%d", gostapi.MerchantAppPortMinIO)
	if config.MinioHost != "" && config.MinioHost != "127.0.0.1" {
		minioAddr = fmt.Sprintf("%s:%d", config.MinioHost, gostapi.MerchantAppPortMinIO)
	}

	setupCmd := fmt.Sprintf(`
sudo mkdir -p /etc/gost /var/log/gost

sudo tee /etc/gost/config.yaml > /dev/null << 'EOF'
api:
  addr: ":%d"
  auth:
    username: %s
    password: %s
log:
  level: info
  format: json
  output: /var/log/gost/gost.log
services:
  - name: local-tcp-%d
    addr: ":%d"
    handler:
      type: tcp
    listener:
      type: tcp
    forwarder:
      nodes:
        - name: wukongim
          addr: 127.0.0.1:%d
  - name: local-http-%d
    addr: ":%d"
    handler:
      type: tcp
    listener:
      type: tcp
    forwarder:
      nodes:
        - name: tsdd-server
          addr: 127.0.0.1:%d
  - name: local-file-%d
    addr: ":%d"
    handler:
      type: tcp
    listener:
      type: tcp
    forwarder:
      nodes:
        - name: minio
          addr: %s
chains: []
EOF

sudo tee /etc/systemd/system/gost.service > /dev/null << 'EOF'
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
`,
		gostapi.GostAPIPortInt,
		gostapi.GostAPIUsername,
		gostapi.GostAPIPassword,
		gostapi.MerchantGostPortIM, gostapi.MerchantGostPortIM, gostapi.MerchantAppPortIM,
		gostapi.MerchantGostPortHTTP, gostapi.MerchantGostPortHTTP, gostapi.MerchantAppPortHTTP,
		gostapi.MerchantGostPortFile, gostapi.MerchantGostPortFile, minioAddr,
	)

	out, err := client.ExecuteCommand(setupCmd)
	if err != nil {
		step.Status = "failed"
		step.Message = fmt.Sprintf("GOST 配置失败: %v, %s", err, out)
		return step
	}

	// 3. 等待并验证
	time.Sleep(2 * time.Second)
	verifyOut, _ := client.ExecuteCommand("systemctl is-active gost 2>/dev/null && ss -tlnp | grep -c gost")
	step.Status = "success"
	step.Message = fmt.Sprintf("商户端 GOST 已安装 (10443→%d, 10080→%d, 10800→%s)",
		gostapi.MerchantAppPortIM, gostapi.MerchantAppPortHTTP, minioAddr)
	step.Output = verifyOut

	return step
}
