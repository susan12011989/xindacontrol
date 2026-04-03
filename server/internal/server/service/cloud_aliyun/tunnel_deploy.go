package cloud_aliyun

import (
	"fmt"
	"server/internal/server/cloud/aliyun"
	"server/internal/server/model"
	"server/internal/server/service/deploy"
	"server/pkg/entity"
	"server/pkg/gostapi"
	"time"

	ecs20140526 "github.com/alibabacloud-go/ecs-20140526/v6/client"
	"github.com/alibabacloud-go/tea/tea"
	"github.com/zeromicro/go-zero/core/logx"
)

// DeployTunnelServerResult 单台服务器部署结果
type DeployTunnelServerResult struct {
	InstanceId  string   `json:"instance_id"`
	ServerName  string   `json:"server_name"`
	PublicIP    string   `json:"public_ip"`
	ServerId    int      `json:"server_id"`
	PrivateKey  string   `json:"private_key"`
	EIPs        []string `json:"eips"`
	KeyPairName string   `json:"key_pair_name"`
}

// DeployTunnelServers 一键部署隧道服务器（批量）
func DeployTunnelServers(req *model.DeployTunnelServerReq, progress func(string)) ([]DeployTunnelServerResult, error) {
	results := make([]DeployTunnelServerResult, 0, req.ServerCount)

	for i := 0; i < req.ServerCount; i++ {
		serverName := req.ServerName
		if req.ServerCount > 1 {
			serverName = fmt.Sprintf("%s-%d", req.ServerName, i+1)
		}

		progress(fmt.Sprintf("===== 开始部署第 %d/%d 台服务器: %s =====", i+1, req.ServerCount, serverName))

		result, err := deploySingleTunnelServer(req, serverName, progress)
		if err != nil {
			progress(fmt.Sprintf("第 %d 台服务器部署失败: %v", i+1, err))
			return results, fmt.Errorf("第 %d 台服务器部署失败: %w", i+1, err)
		}

		results = append(results, *result)
		progress(fmt.Sprintf("第 %d/%d 台服务器部署完成: %s", i+1, req.ServerCount, serverName))
	}

	return results, nil
}

// findInstanceByID 从实例列表中按 ID 查找
func findInstanceByID(instances []*ecs20140526.DescribeInstancesResponseBodyInstancesInstance, instanceId string) *ecs20140526.DescribeInstancesResponseBodyInstancesInstance {
	for _, inst := range instances {
		if inst.InstanceId != nil && *inst.InstanceId == instanceId {
			return inst
		}
	}
	return nil
}

// log 同时输出到 progress 和 logx
func logProgress(progress func(string), format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	progress(msg)
	logx.Infof("[TunnelDeploy] %s", msg)
}

func deploySingleTunnelServer(req *model.DeployTunnelServerReq, serverName string, progress func(string)) (*DeployTunnelServerResult, error) {
	result := &DeployTunnelServerResult{ServerName: serverName}

	// Step 1: 创建 ECS 实例（跳过异步后续处理，由本函数控制全流程）
	logProgress(progress, "Step 1/6: 创建 ECS 实例...")
	instanceResult, err := aliyun.CreateInstance(&aliyun.CreateInstanceRequest{
		CloudAccountId:     req.CloudAccountId,
		Region:             req.RegionId,
		InstanceType:       req.InstanceType,
		InstanceChargeType: "PostPaid",
		DiskCategory:       "cloud_essd",
		DiskSize:           40,
		SkipPostActions:    true,
	})
	if err != nil {
		return nil, fmt.Errorf("创建 ECS 失败: %w", err)
	}
	result.InstanceId = instanceResult.InstanceId
	result.KeyPairName = instanceResult.KeyPairName
	logProgress(progress, "  实例已创建: %s", instanceResult.InstanceId)

	// Step 2: 等待初始化 → 授权安全组 → 启动实例
	logProgress(progress, "Step 2/6: 等待初始化并启动实例...")
	logProgress(progress, "  等待实例初始化 (15s)...")
	time.Sleep(15 * time.Second)

	logProgress(progress, "  获取实例详情...")
	instances, err := aliyun.DescribeInstancesByCloudAccount(req.CloudAccountId, req.RegionId)
	if err != nil {
		return nil, fmt.Errorf("获取实例详情失败: %w", err)
	}
	inst := findInstanceByID(instances, instanceResult.InstanceId)
	if inst == nil {
		return nil, fmt.Errorf("未找到刚创建的实例: %s", instanceResult.InstanceId)
	}

	var securityGroupId string
	if inst.SecurityGroupIds != nil && inst.SecurityGroupIds.SecurityGroupId != nil && len(inst.SecurityGroupIds.SecurityGroupId) > 0 {
		securityGroupId = *inst.SecurityGroupIds.SecurityGroupId[0]
	}
	if securityGroupId != "" {
		logProgress(progress, "  授权安全组 %s (TCP 1-65535)...", securityGroupId)
		err = aliyun.AuthorizeSecurityGroup(&aliyun.AuthorizeSecurityGroupRequest{
			CloudAccountId:  req.CloudAccountId,
			RegionId:        req.RegionId,
			SecurityGroupId: securityGroupId,
			Permissions: []*ecs20140526.AuthorizeSecurityGroupRequestPermissions{
				{
					IpProtocol:   tea.String("TCP"),
					PortRange:    tea.String("1/65535"),
					SourceCidrIp: tea.String("0.0.0.0/0"),
					Policy:       tea.String("Accept"),
				},
			},
		})
		if err != nil {
			logProgress(progress, "  安全组授权警告（可能已存在）: %v", err)
		} else {
			logProgress(progress, "  安全组授权成功")
		}
	}

	// 不先启动 — 直接在 Stopped 状态下绑网卡，绑完再启动（避免热插拔问题）
	logProgress(progress, "  实例保持 Stopped 状态，先绑定网卡再启动")

	// 获取 VSwitch
	var vswitchId string
	if inst.VpcAttributes != nil && inst.VpcAttributes.VSwitchId != nil {
		vswitchId = *inst.VpcAttributes.VSwitchId
	}
	if vswitchId == "" {
		return nil, fmt.Errorf("实例未关联 VSwitch")
	}

	// Step 3: 创建辅助网卡并绑定（实例在 Stopped 状态，无需热插拔）
	logProgress(progress, "Step 3/6: 创建 %d 个辅助网卡（实例 Stopped 状态）...", req.EipCount)
	for j := 0; j < req.EipCount; j++ {
		nicName := fmt.Sprintf("%s-nic-%d", serverName, j+1)
		logProgress(progress, "  创建辅助网卡 %d/%d: %s", j+1, req.EipCount, nicName)

		nicId, err := aliyun.CreateNetworkInterface(&aliyun.CreateNetworkInterfaceRequest{
			CloudAccountId:       req.CloudAccountId,
			RegionId:             req.RegionId,
			VSwitchId:            vswitchId,
			SecurityGroupId:      securityGroupId,
			NetworkInterfaceName: nicName,
		})
		if err != nil {
			return nil, fmt.Errorf("创建辅助网卡 %d 失败: %w", j+1, err)
		}

		time.Sleep(2 * time.Second)
		err = aliyun.AttachNetworkInterface(&aliyun.AttachNetworkInterfaceRequest{
			CloudAccountId:     req.CloudAccountId,
			RegionId:           req.RegionId,
			NetworkInterfaceId: nicId,
			InstanceId:         instanceResult.InstanceId,
		})
		if err != nil {
			return nil, fmt.Errorf("绑定辅助网卡 %d 失败: %w", j+1, err)
		}
		logProgress(progress, "  辅助网卡 %d 已绑定: %s", j+1, nicId)
	}

	// Step 4: 启动实例
	logProgress(progress, "Step 4/6: 启动实例...")
	err = aliyun.StartInstance(0, req.CloudAccountId, req.RegionId, instanceResult.InstanceId)
	if err != nil {
		return nil, fmt.Errorf("启动实例失败: %w", err)
	}
	var publicIP string
	for retry := 0; retry < 60; retry++ {
		time.Sleep(5 * time.Second)
		logProgress(progress, "  等待运行中... (%ds)", (retry+1)*5)
		instances, err := aliyun.DescribeInstancesByCloudAccount(req.CloudAccountId, req.RegionId)
		if err != nil {
			logProgress(progress, "  查询失败: %v，继续...", err)
			continue
		}
		inst = findInstanceByID(instances, instanceResult.InstanceId)
		if inst == nil {
			continue
		}
		if inst.Status != nil && *inst.Status == "Running" {
			if inst.PublicIpAddress != nil && inst.PublicIpAddress.IpAddress != nil {
				for _, ip := range inst.PublicIpAddress.IpAddress {
					if ip != nil && *ip != "" {
						publicIP = *ip
						break
					}
				}
			}
			if publicIP == "" && inst.EipAddress != nil && inst.EipAddress.IpAddress != nil && *inst.EipAddress.IpAddress != "" {
				publicIP = *inst.EipAddress.IpAddress
			}
			logProgress(progress, "  实例已运行, IP: %s", publicIP)
			break
		}
	}
	if inst == nil || (inst.Status != nil && *inst.Status != "Running") {
		return nil, fmt.Errorf("等待实例启动超时")
	}
	if publicIP == "" {
		logProgress(progress, "  实例已运行（无公网 IP，将通过 EIP 分配）")
	}

	// Step 5: 创建 EIP 并绑定
	logProgress(progress, "Step 5/6: 创建 %d 个 EIP (带宽: %sMbps)...", req.EipCount, req.Bandwidth)

	type eipInfo struct {
		AllocationId string
		IP           string
	}
	eipInfos := make([]eipInfo, 0, req.EipCount)

	for j := 0; j < req.EipCount; j++ {
		allocationId, ip, err := aliyun.AllocateEipAddress(&aliyun.AllocateEipAddressRequest{
			CloudAccountId:     req.CloudAccountId,
			RegionId:           req.RegionId,
			InstanceChargeType: "PostPaid",
			InternetChargeType: "PayByTraffic",
			Bandwidth:          req.Bandwidth,
		})
		if err != nil {
			return nil, fmt.Errorf("创建 EIP %d 失败: %w", j+1, err)
		}
		eipInfos = append(eipInfos, eipInfo{allocationId, ip})
		result.EIPs = append(result.EIPs, ip)
		logProgress(progress, "  EIP %d 已创建: %s (%s)", j+1, ip, allocationId)
	}

	// 批量绑定 EIP 到网卡
	time.Sleep(3 * time.Second)

	networkInterfaces, err := aliyun.DescribeNetworkInterfaces(req.CloudAccountId, 0, req.RegionId)
	if err != nil {
		return nil, fmt.Errorf("获取网卡列表失败: %w", err)
	}

	var secondaryNICs []string
	var primaryNIC string
	for _, nic := range networkInterfaces {
		if nic.InstanceId != nil && *nic.InstanceId == instanceResult.InstanceId {
			if nic.AssociatedPublicIp != nil && nic.AssociatedPublicIp.PublicIpAddress != nil && *nic.AssociatedPublicIp.PublicIpAddress != "" {
				continue
			}
			if nic.Type != nil && *nic.Type == "Primary" {
				primaryNIC = *nic.NetworkInterfaceId
			} else {
				secondaryNICs = append(secondaryNICs, *nic.NetworkInterfaceId)
			}
		}
	}
	availableNICs := secondaryNICs
	if primaryNIC != "" {
		availableNICs = append(availableNICs, primaryNIC)
	}

	for j, eip := range eipInfos {
		if j >= len(availableNICs) {
			logProgress(progress, "  EIP %s 无可用网卡绑定", eip.IP)
			continue
		}
		nicId := availableNICs[j]
		err := aliyun.AssociateEipAddress(&aliyun.AssociateEipAddressRequest{
			CloudAccountId: req.CloudAccountId,
			Region:         req.RegionId,
			AllocationId:   eip.AllocationId,
			InstanceType:   "NetworkInterface",
			InstanceId:     nicId,
		})
		if err != nil {
			logx.Errorf("[TunnelDeploy] 绑定 EIP %s 到网卡 %s 失败: %v", eip.IP, nicId, err)
			logProgress(progress, "  EIP %s 绑定失败: %v", eip.IP, err)
		} else {
			logProgress(progress, "  EIP %s 已绑定到网卡 %s", eip.IP, nicId)
		}
	}

	// Step 6: 注册服务器（SSH 密钥）
	serverIP := publicIP
	if len(result.EIPs) > 0 {
		serverIP = result.EIPs[0]
	}
	if serverIP == "" {
		return nil, fmt.Errorf("服务器无可用公网 IP")
	}
	result.PublicIP = serverIP

	logProgress(progress, "Step 6/6: 注册服务器 (SSH 密钥), IP: %s ...", serverIP)
	regResult, err := aliyun.RegisterInstanceWithSSHKey(&aliyun.RegisterInstanceWithSSHKeyRequest{
		CloudAccountId: req.CloudAccountId,
		RegionId:       req.RegionId,
		InstanceId:     instanceResult.InstanceId,
		ServerName:     serverName,
		ServerType:     2,
		PublicIp:       serverIP,
	})
	if err != nil {
		return nil, fmt.Errorf("注册服务器失败: %w", err)
	}
	result.ServerId = regResult.ServerId
	result.PrivateKey = regResult.PrivateKey
	result.KeyPairName = regResult.KeyPairName
	logProgress(progress, "  服务器已注册: ID=%d, 请保存 SSH 私钥", regResult.ServerId)

	// Step 7: 安装 GOST
	logProgress(progress, "Step 7/8: 安装 GOST...")
	gostErr := aliyun.AutoInstallGost(serverIP, &aliyun.ServerAuthInfo{
		AuthType:   2,
		PrivateKey: regResult.PrivateKey,
	})
	if gostErr != nil {
		logProgress(progress, "  GOST 安装失败（可手动安装）: %v", gostErr)
	} else {
		logProgress(progress, "  GOST 安装成功")

		// 验证 GOST API
		_, verifyErr := gostapi.GetConfig(serverIP, "")
		if verifyErr != nil {
			logProgress(progress, "  GOST API 验证失败: %v", verifyErr)
		} else {
			logProgress(progress, "  GOST API 验证成功")
		}
	}

	// Step 8: 为所有商户配置转发规则
	logProgress(progress, "Step 8/9: 配置商户转发规则...")
	deploy.EnqueueGostServicesForMerchants(serverIP, entity.ForwardTypeEncrypted)
	logProgress(progress, "  商户转发规则已入队（异步执行）")

	// Step 9: 安装 Nginx 文件缓存
	logProgress(progress, "Step 9/9: 配置文件缓存 (Nginx)...")
	nginxErr := deploy.InstallNginxToServer(result.ServerId, func(msg string) {
		logProgress(progress, "  %s", msg)
	})
	if nginxErr != nil {
		logProgress(progress, "  Nginx 安装失败（不影响转发）: %v", nginxErr)
	} else {
		filePort := gostapi.SystemPortFile
		if err := deploy.UpdateGostServiceToLoopback(result.ServerId, filePort); err != nil {
			logProgress(progress, "  GOST loopback 切换失败: %v", err)
		} else if err := deploy.ConfigureNginxCacheForPort(result.ServerId, filePort); err != nil {
			_ = deploy.RestoreGostServiceToPublic(result.ServerId, filePort)
			logProgress(progress, "  Nginx 缓存配置失败: %v", err)
		} else {
			logProgress(progress, "  文件缓存已启用 (端口 %d, 最大2GB, 7天过期, 防击穿)", filePort)
		}
	}

	return result, nil
}
