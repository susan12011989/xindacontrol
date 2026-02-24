package deploy

import (
	"fmt"
	"server/internal/server/model"
	"server/internal/server/utils"
	"strings"
	"time"
)

// DetectPrivateIP 通过 SSH 检测节点自身内网 IP（优先 AWS EC2 metadata，回退 hostname）
func DetectPrivateIP(client *utils.SSHClient) string {
	output, err := client.ExecuteCommandWithTimeout(
		`curl -sf --connect-timeout 2 http://169.254.169.254/latest/meta-data/local-ipv4 2>/dev/null || hostname -I | awk '{print $1}'`,
		10*time.Second,
	)
	if err != nil || output == "" {
		return ""
	}
	return strings.TrimSpace(output)
}

// SetupNode 初始化节点：安装 Docker、挂载磁盘、创建目录
func SetupNode(client *utils.SSHClient, config model.DeployConfig) []model.DeployStep {
	steps := make([]model.DeployStep, 0)

	// Step 1: 安装 Docker（AMI 部署时已预装，installDocker 内部会检测并跳过）
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

# DB 节点需要 1 个数据盘：/data/db (MySQL+Redis)
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

# Allinone 节点需要 2 个数据盘：/data/db 和 /data/minio
MOUNTS=("/data/db" "/data/minio")

for i in "${!UNATTACHED[@]}"; do
    disk="${UNATTACHED[$i]}"
    mount_point="${MOUNTS[$i]:-}"
    [ -z "$mount_point" ] && break

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
done

echo "=== 磁盘状态 ==="
df -h /data/db /data/minio 2>/dev/null || echo "部分挂载点不存在"
`
}

// createDirectories 创建工作目录
func createDirectories(client *utils.SSHClient, config model.DeployConfig) model.DeployStep {
	step := model.DeployStep{
		Name:   "创建工作目录",
		Status: "running",
	}

	var dirs string
	switch config.NodeRole {
	case "db":
		dirs = "mkdir -p /opt/tsdd /data/db/mysql /data/db/redis"
	case "minio":
		dirs = "mkdir -p /opt/tsdd /data/minio"
	case "app":
		dirs = "mkdir -p /opt/tsdd /opt/tsdd/assets /opt/tsdd/ssl /opt/tsdd/web /data/db/wukongim"
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

	// Step 6: 拉取镜像并启动（AMI 部署跳过镜像拉取）
	startStep := pullAndStartServices(client, config.FromAMI)
	resp.Steps = append(resp.Steps, startStep)
	if startStep.Status == "failed" {
		resp.Success = false
		resp.Message = "启动服务失败"
		return resp
	}

	// Step 7: 健康检查
	healthStep := checkNodeHealth(client, config)
	resp.Steps = append(resp.Steps, healthStep)

	resp.Success = true
	resp.Message = fmt.Sprintf("%s 节点部署完成", config.NodeRole)

	if config.NodeRole != "db" && config.NodeRole != "minio" {
		resp.APIUrl = fmt.Sprintf("http://%s:%d", config.ExternalIP, config.APIPort)
		resp.WebUrl = fmt.Sprintf("http://%s:%d", config.ExternalIP, config.WebPort)
		resp.AdminUrl = fmt.Sprintf("http://%s:%d", config.ExternalIP, config.ManagerPort)
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
docker ps --format "table {{.Names}}\t{{.Status}}" | grep tsdd
echo ""
echo "=== MySQL ==="
docker exec tsdd-mysql mysqladmin ping -h localhost 2>/dev/null && echo "MySQL: OK" || echo "MySQL: FAIL"
echo "=== Redis ==="
docker exec tsdd-redis redis-cli ping 2>/dev/null || echo "Redis: FAIL"
`
	case "minio":
		checkCmd = `
echo "=== 容器状态 ==="
docker ps --format "table {{.Names}}\t{{.Status}}" | grep tsdd
echo ""
echo "=== MinIO ==="
curl -sf http://localhost:9000/minio/health/live >/dev/null && echo "MinIO: OK" || echo "MinIO: FAIL"
`
	case "app":
		checkCmd = fmt.Sprintf(`
echo "=== 容器状态 ==="
docker ps --format "table {{.Names}}\t{{.Status}}" | grep tsdd
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
docker ps --format "table {{.Names}}\t{{.Status}}" | grep tsdd
echo ""
echo "=== 各服务健康检查 ==="
docker exec tsdd-mysql mysqladmin ping -h localhost 2>/dev/null && echo "MySQL: OK" || echo "MySQL: FAIL"
docker exec tsdd-redis redis-cli ping 2>/dev/null || echo "Redis: FAIL"
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
