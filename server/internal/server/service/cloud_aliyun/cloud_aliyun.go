package cloud_aliyun

import (
	"errors"
	"fmt"
	"server/internal/server/cloud/aliyun"
	"server/internal/server/model"
	"server/pkg/dbs"
	"server/pkg/entity"
	"time"

	ecs20140526 "github.com/alibabacloud-go/ecs-20140526/v6/client"
	"github.com/alibabacloud-go/tea/tea"
	vpc20160428 "github.com/alibabacloud-go/vpc-20160428/v6/client"
	"github.com/zeromicro/go-zero/core/logx"
)

// GetAccountBalance 查询阿里云账户余额（支持：merchant_id 或 cloud_account_id）
func GetAccountBalance(merchantId int, cloudAccountId int64) (string, error) {
	if cloudAccountId > 0 {
		return aliyun.BalanceByCloudAccount(cloudAccountId)
	}
	if merchantId > 0 {
		return aliyun.Balance(merchantId)
	}
	return "", errors.New("merchant_id或cloud_account_id必须提供一个")
}

// GetAccountBalanceDetail 查询阿里云账户余额详情
func GetAccountBalanceDetail(merchantId int, cloudAccountId int64) (*aliyun.BalanceInfo, error) {
	if cloudAccountId > 0 {
		return aliyun.BalanceDetailByCloudAccount(cloudAccountId)
	}
	if merchantId > 0 {
		return aliyun.BalanceDetail(merchantId)
	}
	return nil, errors.New("merchant_id或cloud_account_id必须提供一个")
}

// ECS实例相关服务

// CreateEcsInstance 创建ECS实例
// 创建成功后会自动：1. 启动实例 2. 等待获取公网IP 3. 注册到服务器列表
func CreateEcsInstance(req model.CreateInstancesReq, progressCallback func(message string)) error {
	for _, r := range req.List {
		progressCallback(fmt.Sprintf("创建ECS实例中 %s %s %s", r.Region, r.InstanceType, r.ImageId))
		result, err := aliyun.CreateInstance(r)
		if err != nil {
			logx.Errorf("创建ECS实例失败: %s", err)
			progressCallback(fmt.Sprintf("创建ECS实例失败: %s", err))
			continue
		}
		// 创建成功，后台会自动：启动实例 -> 获取公网IP -> 注册服务器到数据库
		progressCallback(fmt.Sprintf("创建ECS实例成功: %s (名称: %s)", result.InstanceId, result.InstanceName))
		if result.KeyPairName != "" {
			progressCallback(fmt.Sprintf("SSH密钥对: %s", result.KeyPairName))
		} else if result.Password != "" {
			progressCallback(fmt.Sprintf("SSH密码: %s (请妥善保管)", result.Password))
		}
		progressCallback("后台正在自动: 启动实例 -> 获取公网IP -> 注册到服务器列表...")
		time.Sleep(2 * time.Second)
	}
	return nil
}

// GetEcsInstanceList 获取ECS实例列表 - 使用商户ID
func GetEcsInstanceList(cloudAccountId int64, merchantId int, regionId string) ([]*ecs20140526.DescribeInstancesResponseBodyInstancesInstance, error) {
	if cloudAccountId > 0 {
		return aliyun.DescribeInstancesByCloudAccount(cloudAccountId, regionId)
	}
	return aliyun.DescribeInstances(merchantId, regionId)
}

// GetEcsInstanceListByCloudAccount 获取ECS实例列表 - 使用系统云账号ID
func GetEcsInstanceListByCloudAccount(cloudAccountId int64, regionId string) ([]*ecs20140526.DescribeInstancesResponseBodyInstancesInstance, error) {
	return aliyun.DescribeInstancesByCloudAccount(cloudAccountId, regionId)
}

// GetNetworkInterfaceEipMap 获取网卡ID到公网IP的映射
func GetNetworkInterfaceEipMap(cloudAccountId int64, merchantId int, regionId string) (map[string][]map[string]interface{}, error) {
	// 查询该区域的所有网卡信息（包含AssociatedPublicIp字段）
	networkInterfaces, err := aliyun.DescribeNetworkInterfaces(cloudAccountId, merchantId, regionId)
	if err != nil {
		logx.Errorf("查询区域 %s 的网卡列表失败: %v", regionId, err)
		return nil, err
	}

	// 创建网卡ID到公网IP列表的映射
	nicEipMap := make(map[string][]map[string]interface{})
	for _, nic := range networkInterfaces {
		if nic.NetworkInterfaceId == nil {
			continue
		}

		nicId := *nic.NetworkInterfaceId
		var publicIps []map[string]interface{}

		// 遍历私有IP，收集绑定的公网IP
		if nic.PrivateIpSets != nil && nic.PrivateIpSets.PrivateIpSet != nil {
			for _, privateIp := range nic.PrivateIpSets.PrivateIpSet {
				if privateIp.AssociatedPublicIp != nil && privateIp.AssociatedPublicIp.PublicIpAddress != nil && *privateIp.AssociatedPublicIp.PublicIpAddress != "" {
					publicIpInfo := map[string]interface{}{
						"PrivateIpAddress": tea.StringValue(privateIp.PrivateIpAddress),
						"PublicIpAddress":  tea.StringValue(privateIp.AssociatedPublicIp.PublicIpAddress),
						"AllocationId":     tea.StringValue(privateIp.AssociatedPublicIp.AllocationId),
						"Primary":          tea.BoolValue(privateIp.Primary),
					}
					publicIps = append(publicIps, publicIpInfo)
				}
			}
		}

		if len(publicIps) > 0 {
			nicEipMap[nicId] = publicIps
		}
	}

	return nicEipMap, nil
}

// OperateEcsInstance 操作ECS实例
func OperateEcsInstance(merchantId int, cloudAccountId int64, regionId, instanceId, operation string) error {
	// 直接调用云API操作ECS实例
	switch operation {
	case "start":
		return aliyun.StartInstance(merchantId, cloudAccountId, regionId, instanceId)
	case "stop":
		return aliyun.StopInstance(merchantId, cloudAccountId, regionId, instanceId)
	case "restart":
		return aliyun.RebootInstance(merchantId, cloudAccountId, regionId, instanceId)
	case "delete":
		return aliyun.DeleteInstance(merchantId, cloudAccountId, regionId, instanceId)
	default:
		return errors.New("不支持的操作")
	}
}

// ModifyInstanceAttribute 修改实例属性
func ModifyInstanceAttribute(req *aliyun.ModifyInstanceAttributeRequest) error {
	return aliyun.ModifyInstanceAttribute(req)
}

// 安全组相关服务
var defPermissions = []*ecs20140526.AuthorizeSecurityGroupRequestPermissions{
	{
		Policy:       tea.String("Accept"),
		IpProtocol:   tea.String("TCP"),
		PortRange:    tea.String("80/80"),
		SourceCidrIp: tea.String("0.0.0.0/0"),
	},
	{
		Policy:       tea.String("Accept"),
		IpProtocol:   tea.String("TCP"),
		PortRange:    tea.String("443/443"),
		SourceCidrIp: tea.String("0.0.0.0/0"),
	},
	{
		Policy:       tea.String("Accept"),
		IpProtocol:   tea.String("TCP"),
		PortRange:    tea.String("22/22"),
		SourceCidrIp: tea.String("0.0.0.0/0"),
	},
	{
		Policy:       tea.String("Accept"),
		IpProtocol:   tea.String("TCP"),
		PortRange:    tea.String("58182/58182"),
		SourceCidrIp: tea.String("0.0.0.0/0"),
	},
	{
		Policy:       tea.String("Accept"),
		IpProtocol:   tea.String("TCP"),
		PortRange:    tea.String("32798/32804"),
		SourceCidrIp: tea.String("0.0.0.0/0"),
	},
}

// CreateSecurityGroup 创建安全组
func CreateSecurityGroup(req *model.CreateSecurityGroupReq, progressCallback func(message string)) {
	for _, req := range req.List {
		progressCallback(fmt.Sprintf("创建安全组中 %s %s", req.RegionId, req.Name))
		sgId, err := aliyun.CreateSecurityGroup(req)
		if err != nil {
			logx.Errorf("创建安全组失败: %s", err)
			progressCallback(fmt.Sprintf("创建安全组失败: %s", err))
			continue
		}
		progressCallback(fmt.Sprintf("创建安全组成功 %s", sgId))
		progressCallback("添加默认规则(22,80,443,58182,32798-32804)")
		err = aliyun.AuthorizeSecurityGroup(&aliyun.AuthorizeSecurityGroupRequest{
			MerchantId:      req.MerchantId,
			RegionId:        req.RegionId,
			SecurityGroupId: sgId,
			Permissions:     defPermissions,
		})
		if err != nil {
			logx.Errorf("添加默认规则失败: %s", err)
			progressCallback(fmt.Sprintf("添加默认规则失败: %s", err))
			continue
		}
		progressCallback("添加默认规则成功")
	}
}

// GetSecurityGroupList 获取安全组列表
func GetSecurityGroupList(merchantId int, regionId string) ([]*ecs20140526.DescribeSecurityGroupsResponseBodySecurityGroupsSecurityGroup, error) {
	return aliyun.DescribeSecurityGroups(merchantId, regionId)
}

// GetSecurityGroupListByCloudAccount 获取安全组列表 - 使用系统云账号ID
func GetSecurityGroupListByCloudAccount(cloudAccountId int64, regionId string) ([]*ecs20140526.DescribeSecurityGroupsResponseBodySecurityGroupsSecurityGroup, error) {
	return aliyun.DescribeSecurityGroupsByCloudAccount(cloudAccountId, regionId)
}

// DescribeSecurityGroupAttribute 查询安全组详情
func DescribeSecurityGroupAttribute(merchantId int, cloudAccountId int64, regionId, securityGroupId string) (*ecs20140526.DescribeSecurityGroupAttributeResponseBody, error) {
	if cloudAccountId > 0 {
		return aliyun.DescribeSecurityGroupAttributeByCloudAccount(cloudAccountId, regionId, securityGroupId)
	}
	return aliyun.DescribeSecurityGroupAttribute(merchantId, regionId, securityGroupId)
}

// DeleteSecurityGroup 删除安全组
func DeleteSecurityGroup(req *aliyun.DeleteSecurityGroupRequest) error {
	return aliyun.DeleteSecurityGroup(req)
}

// AuthorizeSecurityGroup 授权安全组规则
func AuthorizeSecurityGroup(req *aliyun.AuthorizeSecurityGroupRequest) error {
	return aliyun.AuthorizeSecurityGroup(req)
}

// RevokeSecurityGroup 撤销安全组规则
func RevokeSecurityGroup(req *aliyun.RevokeSecurityGroupRequest) error {
	return aliyun.RevokeSecurityGroup(req)
}

// 网络接口相关服务

// CreateNetworkInterface 创建弹性网卡
func CreateNetworkInterface(req *aliyun.CreateNetworkInterfaceRequest) (string, error) {
	return aliyun.CreateNetworkInterface(req)
}

// GetNetworkInterfaceList 获取弹性网卡列表
func GetNetworkInterfaceList(cloudAccountId int64, merchantId int, region string) ([]*ecs20140526.DescribeNetworkInterfacesResponseBodyNetworkInterfaceSetsNetworkInterfaceSet, error) {
	return aliyun.DescribeNetworkInterfaces(cloudAccountId, merchantId, region)
}

// DeleteNetworkInterface 删除弹性网卡
func DeleteNetworkInterface(req *aliyun.DeleteNetworkInterfaceRequest) error {
	return aliyun.DeleteNetworkInterface(req)
}

// AttachNetworkInterface 附加弹性网卡到ECS实例
func AttachNetworkInterface(req *aliyun.AttachNetworkInterfaceRequest) error {
	return aliyun.AttachNetworkInterface(req)
}

// DetachNetworkInterface 分离弹性网卡
func DetachNetworkInterface(req *aliyun.DetachNetworkInterfaceRequest) error {
	return aliyun.DetachNetworkInterface(req)
}

// ModifyNetworkInterfaceAttribute 修改弹性网卡属性
func ModifyNetworkInterfaceAttribute(req *aliyun.ModifyNetworkInterfaceAttributeRequest) error {
	return aliyun.ModifyNetworkInterfaceAttribute(req)
}

// 弹性IP相关服务

// AllocateEipAddress 申请弹性IP
func AllocateEipAddress(req *model.CreateEipReq, progressCallback func(message string)) {
	for _, req := range req.List {
		num := req.Num
		if num == 0 {
			num = 1
		}
		for i := 0; i < num; i++ {
			progressCallback(fmt.Sprintf("申请弹性IP中 %s %s %s %s",
				req.RegionId, req.InstanceChargeType, req.InternetChargeType, req.Bandwidth))
			instanceId, ip, err := aliyun.AllocateEipAddress(req)
			if err != nil {
				logx.Errorf("申请弹性IP失败: %s", err)
				progressCallback(fmt.Sprintf("申请弹性IP失败: %s", err))
				continue
			}
			progressCallback(fmt.Sprintf("申请弹性IP成功 %s %s", instanceId, ip))
		}
	}
}

// GetEipList 获取弹性IP列表
func GetEipList(merchantId int, region string) ([]*vpc20160428.DescribeEipAddressesResponseBodyEipAddressesEipAddress, error) {
	return aliyun.DescribeEipAddresses(&aliyun.DescribeEipAddressesRequest{
		MerchantId: merchantId,
		Region:     region,
	})
}

// GetEipListByCloudAccount 获取EIP列表 - 使用系统云账号ID
func GetEipListByCloudAccount(cloudAccountId int64, region string) ([]*vpc20160428.DescribeEipAddressesResponseBodyEipAddressesEipAddress, error) {
	return aliyun.DescribeEipAddressesByCloudAccount(&aliyun.DescribeEipAddressesByCloudAccountRequest{
		CloudAccountId: cloudAccountId,
		Region:         region,
	})
}

// ReleaseEipAddress 释放弹性IP
func ReleaseEipAddress(req *aliyun.ReleaseEipAddressRequest) error {
	return aliyun.ReleaseEipAddress(req)
}

// ModifyEipAddressAttribute 修改弹性IP属性
func ModifyEipAddressAttribute(req *aliyun.ModifyEipAddressAttributeRequest) error {
	return aliyun.ModifyEipAddressAttribute(req)
}

// AssociateEipAddress 绑定弹性IP到实例
func AssociateEipAddress(req *aliyun.AssociateEipAddressRequest) error {
	return aliyun.AssociateEipAddress(req)
}

// UnassociateEipAddress 解绑弹性IP
func UnassociateEipAddress(req *aliyun.UnassociateEipAddressRequest) error {
	return aliyun.UnassociateEipAddress(req)
}

// 带宽包相关服务

// CreateBandwidthPackage 创建共享带宽
func CreateBandwidthPackage(req *aliyun.CreateCommonBandwidthPackageRequest) (string, error) {
	return aliyun.CreateCommonBandwidthPackage(req)
}

// GetBandwidthPackageList 获取共享带宽列表
func GetBandwidthPackageList(merchantId int, region string) ([]*vpc20160428.DescribeCommonBandwidthPackagesResponseBodyCommonBandwidthPackagesCommonBandwidthPackage, error) {
	return aliyun.DescribeCommonBandwidthPackages(merchantId, region)
}

// GetBandwidthPackageListByCloudAccount 获取带宽包列表 - 使用系统云账号ID
func GetBandwidthPackageListByCloudAccount(cloudAccountId int64, region string) ([]*vpc20160428.DescribeCommonBandwidthPackagesResponseBodyCommonBandwidthPackagesCommonBandwidthPackage, error) {
	return aliyun.DescribeCommonBandwidthPackagesByCloudAccount(cloudAccountId, region)
}

// DeleteBandwidthPackage 删除共享带宽
func DeleteBandwidthPackage(req *aliyun.DeleteCommonBandwidthPackageRequest) error {
	return aliyun.DeleteCommonBandwidthPackage(req)
}

// ModifyBandwidthPackageAttribute 修改共享带宽属性
func ModifyBandwidthPackageAttribute(req *aliyun.ModifyCommonBandwidthPackageAttributeRequest) error {
	return aliyun.ModifyCommonBandwidthPackageAttribute(req)
}

// ModifyBandwidthPackageSpec 修改共享带宽规格
func ModifyBandwidthPackageSpec(req *aliyun.ModifyCommonBandwidthPackageSpecRequest) error {
	return aliyun.ModifyCommonBandwidthPackageSpec(req)
}

// AddEipToBandwidthPackage 添加EIP到共享带宽
func AddEipToBandwidthPackage(req *aliyun.AddCommonBandwidthPackageIpsRequest) error {
	return aliyun.AddCommonBandwidthPackageIps(req)
}

// RemoveEipFromBandwidthPackage 从共享带宽移除EIP
func RemoveEipFromBandwidthPackage(req *aliyun.RemoveCommonBandwidthPackageIpRequest) error {
	return aliyun.RemoveCommonBandwidthPackageIp(req)
}

// 镜像相关服务

// DescribeImages 查看可用的镜像列表
func DescribeImages(cloudAccountId int64, merchantId int, region string) ([]*ecs20140526.DescribeImagesResponseBodyImagesImage, error) {
	return aliyun.DescribeImages(cloudAccountId, merchantId, region)
}

// CreateSecondaryNetworkInterface 为指定实例创建并绑定辅助网卡
func CreateSecondaryNetworkInterface(cloudAccountId int64, merchantId int, instances []model.InstanceRegionPair, progressCallback func(message string)) error {
	logx.Infof("CreateSecondaryNetworkInterface: %v", instances)
	// 按区域对实例进行分组，以便批量获取每个区域的网卡信息
	regionInstanceMap := make(map[string][]string)
	for _, instance := range instances {
		regionInstanceMap[instance.RegionId] = append(regionInstanceMap[instance.RegionId], instance.InstanceId)
	}

	// 处理每个区域的实例
	for regionId, instanceIds := range regionInstanceMap {
		progressCallback(fmt.Sprintf("开始处理区域 %s 的 %d 个实例", regionId, len(instanceIds)))

		// 获取区域的所有网卡信息，避免重复调用 API
		networkInterfaces, err := aliyun.DescribeNetworkInterfaces(cloudAccountId, merchantId, regionId)
		if err != nil {
			errMsg := fmt.Sprintf("获取区域 %s 的网卡列表失败: %s", regionId, err)
			progressCallback(errMsg)
			logx.Error(errMsg)
			continue
		}

		// 创建实例ID到网卡的映射，以便快速查找实例是否已有辅助网卡
		instanceNicMap := make(map[string][]*ecs20140526.DescribeNetworkInterfacesResponseBodyNetworkInterfaceSetsNetworkInterfaceSet)
		for _, nic := range networkInterfaces {
			if nic.InstanceId != nil && *nic.InstanceId != "" {
				instanceNicMap[*nic.InstanceId] = append(instanceNicMap[*nic.InstanceId], nic)
			}
		}

		// 获取区域中的所有实例，以获取其VPC和安全组信息
		allInstances, err := GetEcsInstanceList(cloudAccountId, merchantId, regionId)
		if err != nil {
			errMsg := fmt.Sprintf("获取区域 %s 的实例列表失败: %s", regionId, err)
			progressCallback(errMsg)
			logx.Error(errMsg)
			continue
		}

		// 创建实例ID到实例详情的映射，以便快速查找实例详情
		instanceMap := make(map[string]*ecs20140526.DescribeInstancesResponseBodyInstancesInstance)
		for _, instance := range allInstances {
			if instance.InstanceId != nil {
				instanceMap[*instance.InstanceId] = instance
			}
		}

		// 处理每个指定的实例
		for _, instanceId := range instanceIds {
			progressCallback(fmt.Sprintf("开始检查实例 %s", instanceId))

			// 获取实例详情
			instance, exists := instanceMap[instanceId]
			if !exists {
				progressCallback(fmt.Sprintf("实例 %s 不存在，跳过", instanceId))
				continue
			}

			// 检查实例是否已有辅助网卡
			// hasSecondaryNIC := false
			// if nics, ok := instanceNicMap[instanceId]; ok {
			// 	for _, nic := range nics {
			// 		if nic.Type != nil && *nic.Type == "Secondary" {
			// 			hasSecondaryNIC = true
			// 			break
			// 		}
			// 	}
			// }

			// if hasSecondaryNIC {
			// 	progressCallback(fmt.Sprintf("实例 %s 已有辅助网卡，跳过", instanceId))
			// 	continue
			// }

			// 获取实例的交换机ID和安全组ID
			vSwitchId := *instance.VpcAttributes.VSwitchId
			var securityGroupId string
			if len(instance.SecurityGroupIds.SecurityGroupId) > 0 {
				securityGroupId = *instance.SecurityGroupIds.SecurityGroupId[0]
			}

			// 创建辅助网卡
			nicName := fmt.Sprintf("nic-secondary-%s", instanceId)
			createReq := &aliyun.CreateNetworkInterfaceRequest{
				CloudAccountId:       cloudAccountId,
				MerchantId:           merchantId,
				RegionId:             regionId,
				VSwitchId:            vSwitchId,
				SecurityGroupId:      securityGroupId,
				NetworkInterfaceName: nicName,
			}

			progressCallback(fmt.Sprintf("正在为实例 %s 创建辅助网卡", instanceId))
			nicId, err := aliyun.CreateNetworkInterface(createReq)
			if err != nil {
				errMsg := fmt.Sprintf("为实例 %s 创建辅助网卡失败: %s", instanceId, err)
				progressCallback(errMsg)
				logx.Error(errMsg)
				continue
			}

			progressCallback(fmt.Sprintf("为实例 %s 创建辅助网卡成功，网卡ID: %s", instanceId, nicId))

			// 将辅助网卡绑定到实例
			attachReq := &aliyun.AttachNetworkInterfaceRequest{
				CloudAccountId:     cloudAccountId,
				MerchantId:         merchantId,
				RegionId:           regionId,
				NetworkInterfaceId: nicId,
				InstanceId:         instanceId,
			}

			progressCallback(fmt.Sprintf("正在将辅助网卡 %s 绑定到实例 %s", nicId, instanceId))
			err = aliyun.AttachNetworkInterface(attachReq)
			if err != nil {
				errMsg := fmt.Sprintf("将辅助网卡 %s 绑定到实例 %s 失败: %s", nicId, instanceId, err)
				progressCallback(errMsg)
				logx.Error(errMsg)

				// 绑定失败，删除创建的网卡
				/*
					deleteReq := &cloud.DeleteNetworkInterfaceRequest{
						MerchantId:         merchantId,
						RegionId:           regionId,
						NetworkInterfaceId: nicId,
					}
					deleteErr := cloud.DeleteNetworkInterface(deleteReq)
					if deleteErr != nil {
						progressCallback(fmt.Sprintf("删除未绑定的网卡 %s 失败: %s", nicId, deleteErr))
					} else {
						progressCallback(fmt.Sprintf("已删除未绑定的网卡 %s", nicId))
					}
				*/
				continue
			}

			progressCallback(fmt.Sprintf("已成功将辅助网卡 %s 绑定到实例 %s", nicId, instanceId))

			// 等待一段时间，避免API调用过快
			time.Sleep(500 * time.Millisecond)
		}
	}

	return nil
}

// BatchAssociateEip 批量绑定弹性IP到实例或辅助网卡
func BatchAssociateEip(merchantId int, cloudAccountId int64, eipList []model.EipBindConfig, progressCallback func(message string)) error {
	progressCallback("开始批量绑定弹性IP")
	// 按区域对EIP进行分组，提高处理效率
	regionEipsMap := make(map[string][]model.EipBindConfig)
	for _, eip := range eipList {
		regionEipsMap[eip.RegionId] = append(regionEipsMap[eip.RegionId], eip)
	}

	// 处理每个区域
	for regionId, eips := range regionEipsMap {
		progressCallback(fmt.Sprintf("开始处理区域 %s 的 %d 个弹性IP", regionId, len(eips)))
		networkInterfaces, err := aliyun.DescribeNetworkInterfaces(cloudAccountId, merchantId, regionId)
		if err != nil {
			errMsg := fmt.Sprintf("获取区域 %s 的网卡列表失败: %s", regionId, err)
			progressCallback(errMsg)
			logx.Error(errMsg)
			continue
		}

		if len(networkInterfaces) == 0 {
			progressCallback(fmt.Sprintf("区域 %s 没有可用的网卡，跳过处理", regionId))
			continue
		}

		// 将网卡按类型分组
		var primaryNICs []*ecs20140526.DescribeNetworkInterfacesResponseBodyNetworkInterfaceSetsNetworkInterfaceSet
		var secondaryNICs []*ecs20140526.DescribeNetworkInterfacesResponseBodyNetworkInterfaceSetsNetworkInterfaceSet

		// 检查网卡是否已绑定EIP
		nicWithEIPMap := make(map[string]bool)

		for _, nic := range networkInterfaces {
			// 跳过未绑定实例的网卡
			if nic.InstanceId == nil || *nic.InstanceId == "" {
				progressCallback(fmt.Sprintf("网卡 %s 未绑定实例，跳过", *nic.NetworkInterfaceId))
				continue
			}

			// 检查网卡是否已有公网IP或绑定了EIP
			hasPublicIP := false
			if nic.PrivateIpSets != nil && nic.PrivateIpSets.PrivateIpSet != nil {
				for _, privateIP := range nic.PrivateIpSets.PrivateIpSet {
					if privateIP.AssociatedPublicIp != nil && privateIP.AssociatedPublicIp.PublicIpAddress != nil && *privateIP.AssociatedPublicIp.PublicIpAddress != "" {
						hasPublicIP = true
						break
					}
				}
			}

			if hasPublicIP {
				if nic.NetworkInterfaceId != nil {
					nicWithEIPMap[*nic.NetworkInterfaceId] = true
				}
				continue
			}

			// 区分主网卡和辅助网卡
			if nic.Type != nil {
				if *nic.Type == "Primary" {
					primaryNICs = append(primaryNICs, nic)
				} else if *nic.Type == "Secondary" {
					secondaryNICs = append(secondaryNICs, nic)
				}
			}
		}

		// 获取该区域所有的EIP信息
		allEips, err := aliyun.DescribeEipAddresses(&aliyun.DescribeEipAddressesRequest{
			CloudAccountId: cloudAccountId,
			MerchantId:     merchantId,
			Region:         regionId,
		})
		if err != nil {
			errMsg := fmt.Sprintf("获取区域 %s 的弹性IP列表失败: %s", regionId, err)
			progressCallback(errMsg)
			logx.Error(errMsg)
			continue
		}

		// 创建EIP ID到EIP详情的映射
		eipMap := make(map[string]*vpc20160428.DescribeEipAddressesResponseBodyEipAddressesEipAddress)
		for _, eip := range allEips {
			if eip.AllocationId != nil {
				eipMap[*eip.AllocationId] = eip
			}
		}

		// 合并主网卡和辅助网卡，主网卡优先
		availableNICs := append(primaryNICs, secondaryNICs...)

		if len(availableNICs) == 0 {
			progressCallback(fmt.Sprintf("区域 %s 没有可用的网卡，无法绑定弹性IP", regionId))
			continue
		}

		// 当前网卡索引，用于循环分配
		nicIndex := 0

		// 处理每个待绑定的EIP
		for _, eip := range eips {
			// 检查EIP是否存在
			eipInfo, exists := eipMap[eip.AllocationId]
			if !exists {
				progressCallback(fmt.Sprintf("弹性IP %s 不存在，跳过", eip.AllocationId))
				continue
			}

			// 检查EIP当前状态
			if eipInfo.Status != nil && *eipInfo.Status == "InUse" {
				// 如果EIP已经绑定，跳过
				instanceIdStr := "未知"
				instanceTypeStr := "未知"
				if eipInfo.InstanceId != nil {
					instanceIdStr = *eipInfo.InstanceId
				}
				if eipInfo.InstanceType != nil {
					instanceTypeStr = *eipInfo.InstanceType
				}
				progressCallback(fmt.Sprintf("弹性IP %s 已绑定到 %s(%s)，跳过",
					eip.AllocationId, instanceIdStr, instanceTypeStr))
				continue
			}

			// 循环找到一个未绑定EIP的网卡
			nicFound := false
			startIndex := nicIndex

			// 尝试一轮查找有效网卡
			for i := 0; i < len(availableNICs); i++ {
				currentIndex := (startIndex + i) % len(availableNICs)
				nic := availableNICs[currentIndex]

				if nic.NetworkInterfaceId == nil {
					continue
				}

				// 检查网卡是否已绑定EIP
				if nicWithEIPMap[*nic.NetworkInterfaceId] {
					continue
				}

				nicFound = true
				nicIndex = (currentIndex + 1) % len(availableNICs) // 更新索引为下一个网卡

				// 确定网卡类型和实例ID
				nicType := "未知"
				if nic.Type != nil {
					nicType = *nic.Type
				}

				instanceId := "未知"
				if nic.InstanceId != nil {
					instanceId = *nic.InstanceId
				}

				// 绑定EIP到网卡
				progressCallback(fmt.Sprintf("尝试将弹性IP %s 绑定到%s网卡 %s (实例: %s)",
					eip.AllocationId, nicType, *nic.NetworkInterfaceId, instanceId))

				// 确定绑定类型
				instanceType := "NetworkInterface"
				if nicType == "Primary" {
					instanceType = "EcsInstance"
				}

				// 确定绑定目标ID
				targetId := *nic.NetworkInterfaceId
				if nicType == "Primary" && nic.InstanceId != nil {
					targetId = *nic.InstanceId
				}

				// 执行绑定
				associateReq := &aliyun.AssociateEipAddressRequest{
					MerchantId:     merchantId,
					CloudAccountId: cloudAccountId,
					Region:         regionId,
					AllocationId:   eip.AllocationId,
					InstanceId:     targetId,
					InstanceType:   instanceType,
				}

				err := aliyun.AssociateEipAddress(associateReq)
				if err != nil {
					errMsg := fmt.Sprintf("绑定弹性IP %s 到%s网卡 %s 失败: %s",
						eip.AllocationId, nicType, *nic.NetworkInterfaceId, err)
					progressCallback(errMsg)
					logx.Error(errMsg)
					continue
				}

				// 更新网卡EIP状态
				nicWithEIPMap[*nic.NetworkInterfaceId] = true

				progressCallback(fmt.Sprintf("成功将弹性IP %s 绑定到%s网卡 %s (实例: %s)",
					eip.AllocationId, nicType, *nic.NetworkInterfaceId, instanceId))

				break
			}

			if !nicFound {
				progressCallback(fmt.Sprintf("区域 %s 没有找到可用的未绑定EIP的网卡，无法绑定弹性IP %s",
					regionId, eip.AllocationId))
			}

			// 等待一段时间，避免API调用过快
			time.Sleep(500 * time.Millisecond)
		}
	}

	progressCallback("批量绑定弹性IP完成")
	return nil
}

// ReplaceEipAddress 更换弹性IP
// 执行流程：查询旧IP -> 创建新EIP -> 解绑旧EIP -> 移除旧EIP共享带宽 -> 新EIP加入共享带宽 -> 绑定新EIP -> 释放旧EIP -> 更新服务器记录
// 注意：先创建新EIP再释放旧EIP，避免中间步骤失败导致实例失去公网访问能力
func ReplaceEipAddress(req *model.ReplaceEipReq, progressCallback func(message string)) error {
	progressCallback(fmt.Sprintf("开始更换弹性IP %s", req.AllocationId))

	// 获取旧EIP的IP地址（用于后续更新服务器记录）
	// 优先使用前端传入的，未传入则调用API查询
	oldIpAddress := req.OldIpAddress
	if oldIpAddress != "" {
		progressCallback(fmt.Sprintf("使用前端传入的旧IP地址: %s", oldIpAddress))
	} else {
		progressCallback("步骤0/7: 查询旧弹性IP详情")
		oldEipDetail, err := aliyun.DescribeEipAddressByAllocationId(&aliyun.DescribeEipAddressByAllocationIdRequest{
			MerchantId:     req.MerchantId,
			CloudAccountId: req.CloudAccountId,
			Region:         req.RegionId,
			AllocationId:   req.AllocationId,
		})
		if err != nil {
			errMsg := fmt.Sprintf("查询旧弹性IP详情失败: %s", err)
			progressCallback(errMsg)
			return errors.New(errMsg)
		}
		if oldEipDetail.IpAddress != nil {
			oldIpAddress = *oldEipDetail.IpAddress
		}
		progressCallback(fmt.Sprintf("旧弹性IP地址: %s", oldIpAddress))
	}

	// 步骤1: 先创建新EIP（确保新资源可用后再进行后续操作）
	progressCallback("步骤1/7: 创建新弹性IP")

	// 处理带宽参数
	// 如果 EIP 会加入共享带宽，创建时使用较小带宽值（共享带宽会接管带宽控制）
	// 如果不加入共享带宽，使用用户指定的带宽，但需要限制在合理范围内
	bandwidth := req.Bandwidth
	if req.BandwidthPackageId != "" {
		// 有共享带宽时，使用最小带宽即可（加入共享带宽后会使用共享带宽的带宽）
		bandwidth = "1"
	} else if bandwidth == "" {
		bandwidth = "1" // 默认带宽
	}

	internetChargeType := req.InternetChargeType
	if internetChargeType == "" {
		internetChargeType = "PayByTraffic" // 默认按流量计费
	}

	createReq := &aliyun.AllocateEipAddressRequest{
		MerchantId:         req.MerchantId,
		CloudAccountId:     req.CloudAccountId,
		RegionId:           req.RegionId,
		InstanceChargeType: "PostPaid", // 按量付费
		Bandwidth:          bandwidth,
		InternetChargeType: internetChargeType,
	}

	newAllocationId, newIpAddress, err := aliyun.AllocateEipAddress(createReq)
	if err != nil {
		errMsg := fmt.Sprintf("创建新弹性IP失败: %s", err)
		progressCallback(errMsg)
		return errors.New(errMsg)
	}
	progressCallback(fmt.Sprintf("创建新弹性IP成功: %s (%s)", newAllocationId, newIpAddress))
	time.Sleep(1 * time.Second) // 等待创建完成

	// 步骤2: 解绑旧EIP与实例（如果已绑定）
	if req.InstanceId != "" {
		progressCallback(fmt.Sprintf("步骤2/7: 解绑旧弹性IP %s 与实例 %s", req.AllocationId, req.InstanceId))
		err := aliyun.UnassociateEipAddress(&aliyun.UnassociateEipAddressRequest{
			MerchantId:     req.MerchantId,
			CloudAccountId: req.CloudAccountId,
			Region:         req.RegionId,
			AllocationId:   req.AllocationId,
			InstanceId:     req.InstanceId,
			InstanceType:   req.InstanceType,
		})
		if err != nil {
			errMsg := fmt.Sprintf("解绑旧弹性IP失败: %s（新EIP %s 已创建，请手动处理）", err, newAllocationId)
			progressCallback(errMsg)
			return errors.New(errMsg)
		}
		progressCallback("解绑旧弹性IP成功")
		time.Sleep(1 * time.Second) // 等待解绑完成
	} else {
		progressCallback("步骤2/7: 旧弹性IP未绑定实例，跳过解绑")
	}

	// 步骤3: 从共享带宽移除旧EIP（如果在共享带宽中）
	if req.BandwidthPackageId != "" {
		progressCallback(fmt.Sprintf("步骤3/7: 从共享带宽 %s 移除旧弹性IP %s", req.BandwidthPackageId, req.AllocationId))
		err := aliyun.RemoveCommonBandwidthPackageIp(&aliyun.RemoveCommonBandwidthPackageIpRequest{
			MerchantId:         req.MerchantId,
			CloudAccountId:     req.CloudAccountId,
			Region:             req.RegionId,
			BandwidthPackageId: req.BandwidthPackageId,
			IpInstanceId:       req.AllocationId,
		})
		if err != nil {
			errMsg := fmt.Sprintf("从共享带宽移除旧弹性IP失败: %s（新EIP %s 已创建，请手动处理）", err, newAllocationId)
			progressCallback(errMsg)
			return errors.New(errMsg)
		}
		progressCallback("从共享带宽移除旧弹性IP成功")
		time.Sleep(1 * time.Second) // 等待移除完成
	} else {
		progressCallback("步骤3/7: 旧弹性IP未加入共享带宽，跳过移除")
	}

	// 步骤4: 将新EIP加入共享带宽（如果原来在共享带宽中）
	if req.BandwidthPackageId != "" {
		progressCallback(fmt.Sprintf("步骤4/7: 将新弹性IP %s 加入共享带宽 %s", newAllocationId, req.BandwidthPackageId))
		err := aliyun.AddCommonBandwidthPackageIps(&aliyun.AddCommonBandwidthPackageIpsRequest{
			MerchantId:         req.MerchantId,
			CloudAccountId:     req.CloudAccountId,
			Region:             req.RegionId,
			BandwidthPackageId: req.BandwidthPackageId,
			IpInstanceIds:      []string{newAllocationId},
		})
		if err != nil {
			errMsg := fmt.Sprintf("将新弹性IP加入共享带宽失败: %s（新EIP %s 已创建，旧EIP %s 未释放，请手动处理）", err, newAllocationId, req.AllocationId)
			progressCallback(errMsg)
			return errors.New(errMsg)
		}
		progressCallback("将新弹性IP加入共享带宽成功")
		time.Sleep(1 * time.Second) // 等待加入完成
	} else {
		progressCallback("步骤4/7: 原弹性IP未加入共享带宽，跳过加入")
	}

	// 步骤5: 绑定新EIP到实例（如果原来绑定了实例）
	if req.InstanceId != "" {
		progressCallback(fmt.Sprintf("步骤5/7: 将新弹性IP %s 绑定到实例 %s", newAllocationId, req.InstanceId))
		err := aliyun.AssociateEipAddress(&aliyun.AssociateEipAddressRequest{
			MerchantId:     req.MerchantId,
			CloudAccountId: req.CloudAccountId,
			Region:         req.RegionId,
			AllocationId:   newAllocationId,
			InstanceId:     req.InstanceId,
			InstanceType:   req.InstanceType,
		})
		if err != nil {
			errMsg := fmt.Sprintf("将新弹性IP绑定到实例失败: %s（新EIP %s 已创建，旧EIP %s 未释放，请手动处理）", err, newAllocationId, req.AllocationId)
			progressCallback(errMsg)
			return errors.New(errMsg)
		}
		progressCallback("将新弹性IP绑定到实例成功")
		time.Sleep(1 * time.Second) // 等待绑定完成
	} else {
		progressCallback("步骤5/7: 原弹性IP未绑定实例，跳过绑定")
	}

	// 步骤6: 最后释放旧EIP（确保所有操作成功后再释放）
	progressCallback(fmt.Sprintf("步骤6/7: 释放旧弹性IP %s", req.AllocationId))
	err = aliyun.ReleaseEipAddress(&aliyun.ReleaseEipAddressRequest{
		MerchantId:     req.MerchantId,
		CloudAccountId: req.CloudAccountId,
		Region:         req.RegionId,
		AllocationId:   req.AllocationId,
	})
	if err != nil {
		// 旧EIP释放失败不影响主流程，只是会多一个未使用的EIP需要手动清理
		warnMsg := fmt.Sprintf("释放旧弹性IP失败: %s（新EIP已绑定成功，旧EIP需手动释放）", err)
		progressCallback(warnMsg)
		// 不返回错误，继续完成流程
	} else {
		progressCallback("释放旧弹性IP成功")
	}

	// 步骤7: 更新系统服务器记录（如果 Host 或 AuxiliaryIP 包含旧IP）
	progressCallback("步骤7/7: 检查并更新系统服务器记录")
	if oldIpAddress != "" {
		// 查询系统服务器（server_type=2）中 Host 或 AuxiliaryIP 包含旧IP的记录
		var servers []entity.Servers
		err = dbs.DBAdmin.Where("server_type = ?", 2).
			And("(host = ? OR auxiliary_ip = ?)", oldIpAddress, oldIpAddress).
			Find(&servers)
		if err != nil {
			warnMsg := fmt.Sprintf("查询系统服务器失败: %s（IP更换已完成，服务器记录需手动更新）", err)
			progressCallback(warnMsg)
			logx.Error(warnMsg)
		} else if len(servers) > 0 {
			updatedCount := 0
			for _, server := range servers {
				updateFields := make(map[string]interface{})
				if server.Host == oldIpAddress {
					updateFields["host"] = newIpAddress
				}
				if server.AuxiliaryIP == oldIpAddress {
					updateFields["auxiliary_ip"] = newIpAddress
				}
				if len(updateFields) > 0 {
					_, err := dbs.DBAdmin.Table("servers").Where("id = ?", server.Id).Update(updateFields)
					if err != nil {
						warnMsg := fmt.Sprintf("更新服务器 %s (ID:%d) 失败: %s", server.Name, server.Id, err)
						progressCallback(warnMsg)
						logx.Error(warnMsg)
					} else {
						updatedCount++
						progressCallback(fmt.Sprintf("已更新服务器 %s (ID:%d) 的IP: %s -> %s", server.Name, server.Id, oldIpAddress, newIpAddress))
					}
				}
			}
			progressCallback(fmt.Sprintf("共更新 %d 个系统服务器记录", updatedCount))
		} else {
			progressCallback("未找到需要更新的系统服务器记录")
		}
	} else {
		progressCallback("旧IP地址为空，跳过更新服务器记录")
	}

	progressCallback(fmt.Sprintf("更换弹性IP完成! 新IP: %s (%s)", newAllocationId, newIpAddress))
	return nil
}

// BatchReplaceEipAddress 批量更换弹性IP
func BatchReplaceEipAddress(req *model.BatchReplaceEipReq, progressCallback func(message string)) error {
	total := len(req.EipList)
	progressCallback(fmt.Sprintf("开始批量更换弹性IP，共 %d 个", total))

	successCount := 0
	failCount := 0

	for i, eipConfig := range req.EipList {
		progressCallback(fmt.Sprintf("\n========== 处理第 %d/%d 个EIP: %s ==========", i+1, total, eipConfig.AllocationId))

		// 构建单个更换请求
		replaceReq := &model.ReplaceEipReq{
			MerchantId:         req.MerchantId,
			CloudAccountId:     req.CloudAccountId,
			RegionId:           eipConfig.RegionId,
			AllocationId:       eipConfig.AllocationId,
			OldIpAddress:       eipConfig.OldIpAddress,
			InstanceId:         eipConfig.InstanceId,
			InstanceType:       eipConfig.InstanceType,
			BandwidthPackageId: eipConfig.BandwidthPackageId,
			Bandwidth:          eipConfig.Bandwidth,
			InternetChargeType: eipConfig.InternetChargeType,
		}

		// 调用单个更换函数
		err := ReplaceEipAddress(replaceReq, progressCallback)
		if err != nil {
			failCount++
			progressCallback(fmt.Sprintf("EIP %s 更换失败: %s", eipConfig.AllocationId, err.Error()))
		} else {
			successCount++
		}

		// 在每个EIP处理完后稍作等待，避免API频率限制
		if i < total-1 {
			time.Sleep(2 * time.Second)
		}
	}

	progressCallback(fmt.Sprintf("\n========== 批量更换完成 =========="))
	progressCallback(fmt.Sprintf("总数: %d, 成功: %d, 失败: %d", total, successCount, failCount))

	if failCount > 0 {
		return fmt.Errorf("部分EIP更换失败，成功: %d, 失败: %d", successCount, failCount)
	}
	return nil
}
