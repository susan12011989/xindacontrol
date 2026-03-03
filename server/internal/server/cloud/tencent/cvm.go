package tencent

import (
	"fmt"
	"strings"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	cvm "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cvm/v20170312"
)

// NewCvmClient 创建腾讯云 CVM 客户端
func NewCvmClient(accessKey, accessSecret, regionId string) (*cvm.Client, error) {
	credential := common.NewCredential(accessKey, accessSecret)
	cpf := profile.NewClientProfile()
	cpf.HttpProfile.Endpoint = "cvm.tencentcloudapi.com"
	return cvm.NewClient(credential, regionId, cpf)
}

// DescribeInstances 查询实例列表（单 region）
func DescribeInstances(cred *CloudAccountInfo, regionId string) ([]*cvm.Instance, int64, error) {
	client, err := NewCvmClient(cred.AccessKey, cred.AccessSecret, regionId)
	if err != nil {
		return nil, 0, fmt.Errorf("创建 CVM 客户端失败: %v", err)
	}

	request := cvm.NewDescribeInstancesRequest()
	request.Limit = common.Int64Ptr(100)

	response, err := client.DescribeInstances(request)
	if err != nil {
		return nil, 0, fmt.Errorf("查询实例失败: %v", err)
	}

	return response.Response.InstanceSet, *response.Response.TotalCount, nil
}

// StartInstances 启动实例
func StartInstances(cred *CloudAccountInfo, regionId string, instanceIds []string) error {
	client, err := NewCvmClient(cred.AccessKey, cred.AccessSecret, regionId)
	if err != nil {
		return fmt.Errorf("创建 CVM 客户端失败: %v", err)
	}

	request := cvm.NewStartInstancesRequest()
	request.InstanceIds = common.StringPtrs(instanceIds)

	_, err = client.StartInstances(request)
	if err != nil {
		return fmt.Errorf("启动实例失败: %v", err)
	}
	return nil
}

// StopInstances 关闭实例
func StopInstances(cred *CloudAccountInfo, regionId string, instanceIds []string) error {
	client, err := NewCvmClient(cred.AccessKey, cred.AccessSecret, regionId)
	if err != nil {
		return fmt.Errorf("创建 CVM 客户端失败: %v", err)
	}

	request := cvm.NewStopInstancesRequest()
	request.InstanceIds = common.StringPtrs(instanceIds)
	request.StopType = common.StringPtr("SOFT_FIRST") // 优先软关机

	_, err = client.StopInstances(request)
	if err != nil {
		return fmt.Errorf("关闭实例失败: %v", err)
	}
	return nil
}

// RebootInstances 重启实例
func RebootInstances(cred *CloudAccountInfo, regionId string, instanceIds []string) error {
	client, err := NewCvmClient(cred.AccessKey, cred.AccessSecret, regionId)
	if err != nil {
		return fmt.Errorf("创建 CVM 客户端失败: %v", err)
	}

	request := cvm.NewRebootInstancesRequest()
	request.InstanceIds = common.StringPtrs(instanceIds)

	_, err = client.RebootInstances(request)
	if err != nil {
		return fmt.Errorf("重启实例失败: %v", err)
	}
	return nil
}

// TerminateInstances 释放/退还实例
func TerminateInstances(cred *CloudAccountInfo, regionId string, instanceIds []string) error {
	client, err := NewCvmClient(cred.AccessKey, cred.AccessSecret, regionId)
	if err != nil {
		return fmt.Errorf("创建 CVM 客户端失败: %v", err)
	}

	request := cvm.NewTerminateInstancesRequest()
	request.InstanceIds = common.StringPtrs(instanceIds)

	_, err = client.TerminateInstances(request)
	if err != nil {
		return fmt.Errorf("释放实例失败: %v", err)
	}
	return nil
}

// ModifyInstancesAttribute 修改实例名称
func ModifyInstancesAttribute(cred *CloudAccountInfo, regionId, instanceId, instanceName string) error {
	client, err := NewCvmClient(cred.AccessKey, cred.AccessSecret, regionId)
	if err != nil {
		return fmt.Errorf("创建 CVM 客户端失败: %v", err)
	}

	request := cvm.NewModifyInstancesAttributeRequest()
	request.InstanceIds = common.StringPtrs([]string{instanceId})
	request.InstanceName = common.StringPtr(instanceName)

	_, err = client.ModifyInstancesAttribute(request)
	if err != nil {
		return fmt.Errorf("修改实例属性失败: %v", err)
	}
	return nil
}

// ModifyInstancesSecurityGroups 修改实例安全组绑定
func ModifyInstancesSecurityGroups(cred *CloudAccountInfo, regionId string, instanceIds, sgIds []string) error {
	client, err := NewCvmClient(cred.AccessKey, cred.AccessSecret, regionId)
	if err != nil {
		return fmt.Errorf("创建 CVM 客户端失败: %v", err)
	}

	request := cvm.NewModifyInstancesAttributeRequest()
	request.InstanceIds = common.StringPtrs(instanceIds)
	request.SecurityGroups = common.StringPtrs(sgIds)

	_, err = client.ModifyInstancesAttribute(request)
	if err != nil {
		return fmt.Errorf("修改安全组绑定失败: %v", err)
	}
	return nil
}

// ResetInstancesPassword 重置实例密码
func ResetInstancesPassword(cred *CloudAccountInfo, regionId, instanceId, password string) error {
	client, err := NewCvmClient(cred.AccessKey, cred.AccessSecret, regionId)
	if err != nil {
		return fmt.Errorf("创建 CVM 客户端失败: %v", err)
	}

	request := cvm.NewResetInstancesPasswordRequest()
	request.InstanceIds = common.StringPtrs([]string{instanceId})
	request.Password = common.StringPtr(password)
	request.ForceStop = common.BoolPtr(true) // 强制关机后重置

	_, err = client.ResetInstancesPassword(request)
	if err != nil {
		return fmt.Errorf("重置密码失败: %v", err)
	}
	return nil
}

// DescribeImages 查询镜像列表
func DescribeImages(cred *CloudAccountInfo, regionId string) ([]*cvm.Image, int64, error) {
	client, err := NewCvmClient(cred.AccessKey, cred.AccessSecret, regionId)
	if err != nil {
		return nil, 0, fmt.Errorf("创建 CVM 客户端失败: %v", err)
	}

	request := cvm.NewDescribeImagesRequest()
	request.Limit = common.Uint64Ptr(100)

	response, err := client.DescribeImages(request)
	if err != nil {
		return nil, 0, fmt.Errorf("查询镜像失败: %v", err)
	}

	return response.Response.ImageSet, int64(*response.Response.TotalCount), nil
}

// GetDefaultUbuntuImageId 获取默认 Ubuntu 公共镜像 ID
func GetDefaultUbuntuImageId(cred *CloudAccountInfo, regionId string) (string, error) {
	client, err := NewCvmClient(cred.AccessKey, cred.AccessSecret, regionId)
	if err != nil {
		return "", fmt.Errorf("创建 CVM 客户端失败: %v", err)
	}

	request := cvm.NewDescribeImagesRequest()
	request.Filters = []*cvm.Filter{
		{
			Name:   common.StringPtr("image-type"),
			Values: common.StringPtrs([]string{"PUBLIC_IMAGE"}),
		},
		{
			Name:   common.StringPtr("platform"),
			Values: common.StringPtrs([]string{"Ubuntu"}),
		},
	}
	request.Limit = common.Uint64Ptr(50)

	response, err := client.DescribeImages(request)
	if err != nil {
		return "", fmt.Errorf("查询公共镜像失败: %v", err)
	}

	if len(response.Response.ImageSet) == 0 {
		return "", fmt.Errorf("未找到 Ubuntu 公共镜像")
	}

	// 优先选择 Ubuntu 22.04，否则取第一个
	for _, img := range response.Response.ImageSet {
		if img.ImageName != nil && (strings.Contains(*img.ImageName, "22.04") || strings.Contains(*img.ImageName, "2204")) {
			return *img.ImageId, nil
		}
	}

	return *response.Response.ImageSet[0].ImageId, nil
}

// RunInstancesInput 创建实例输入参数
type RunInstancesInput struct {
	RegionId                string
	Zone                    string
	ImageId                 string
	InstanceType            string
	InstanceChargeType      string // PREPAID / POSTPAID_BY_HOUR
	SystemDiskType          string
	SystemDiskSize          int64
	VpcId                   string
	SubnetId                string
	SecurityGroupIds        []string
	InstanceName            string
	Password                string
	Period                  int64  // 预付费月数
	RenewFlag               string // NOTIFY_AND_AUTO_RENEW / NOTIFY_AND_MANUAL_RENEW
	InternetMaxBandwidthOut int64  // 公网出带宽 Mbps
}

// RunInstances 创建 CVM 实例
func RunInstances(cred *CloudAccountInfo, input *RunInstancesInput) ([]string, error) {
	client, err := NewCvmClient(cred.AccessKey, cred.AccessSecret, input.RegionId)
	if err != nil {
		return nil, fmt.Errorf("创建 CVM 客户端失败: %v", err)
	}

	request := cvm.NewRunInstancesRequest()
	request.ImageId = common.StringPtr(input.ImageId)
	request.InstanceType = common.StringPtr(input.InstanceType)
	request.Placement = &cvm.Placement{
		Zone: common.StringPtr(input.Zone),
	}

	// 付费类型
	if input.InstanceChargeType != "" {
		request.InstanceChargeType = common.StringPtr(input.InstanceChargeType)
	}
	if input.InstanceChargeType == "PREPAID" && input.Period > 0 {
		request.InstanceChargePrepaid = &cvm.InstanceChargePrepaid{
			Period: common.Int64Ptr(input.Period),
		}
		if input.RenewFlag != "" {
			request.InstanceChargePrepaid.RenewFlag = common.StringPtr(input.RenewFlag)
		}
	}

	// 系统盘
	if input.SystemDiskType != "" || input.SystemDiskSize > 0 {
		request.SystemDisk = &cvm.SystemDisk{}
		if input.SystemDiskType != "" {
			request.SystemDisk.DiskType = common.StringPtr(input.SystemDiskType)
		}
		if input.SystemDiskSize > 0 {
			request.SystemDisk.DiskSize = common.Int64Ptr(input.SystemDiskSize)
		}
	}

	// VPC/子网
	if input.VpcId != "" || input.SubnetId != "" {
		request.VirtualPrivateCloud = &cvm.VirtualPrivateCloud{}
		if input.VpcId != "" {
			request.VirtualPrivateCloud.VpcId = common.StringPtr(input.VpcId)
		}
		if input.SubnetId != "" {
			request.VirtualPrivateCloud.SubnetId = common.StringPtr(input.SubnetId)
		}
	}

	// 安全组
	if len(input.SecurityGroupIds) > 0 {
		request.SecurityGroupIds = common.StringPtrs(input.SecurityGroupIds)
	}

	// 实例名称
	if input.InstanceName != "" {
		request.InstanceName = common.StringPtr(input.InstanceName)
	}

	// 登录密码
	if input.Password != "" {
		request.LoginSettings = &cvm.LoginSettings{
			Password: common.StringPtr(input.Password),
		}
	}

	// 公网带宽
	if input.InternetMaxBandwidthOut > 0 {
		request.InternetAccessible = &cvm.InternetAccessible{
			InternetMaxBandwidthOut:  common.Int64Ptr(input.InternetMaxBandwidthOut),
			PublicIpAssigned:         common.BoolPtr(true),
			InternetChargeType:       common.StringPtr("TRAFFIC_POSTPAID_BY_HOUR"),
		}
	}

	response, err := client.RunInstances(request)
	if err != nil {
		return nil, fmt.Errorf("创建实例失败: %v", err)
	}

	var instanceIds []string
	for _, id := range response.Response.InstanceIdSet {
		instanceIds = append(instanceIds, *id)
	}
	return instanceIds, nil
}

// DescribeInstanceTypeConfigs 查询实例规格
func DescribeInstanceTypeConfigs(cred *CloudAccountInfo, regionId string) ([]*cvm.InstanceTypeConfig, error) {
	client, err := NewCvmClient(cred.AccessKey, cred.AccessSecret, regionId)
	if err != nil {
		return nil, fmt.Errorf("创建 CVM 客户端失败: %v", err)
	}

	request := cvm.NewDescribeInstanceTypeConfigsRequest()

	response, err := client.DescribeInstanceTypeConfigs(request)
	if err != nil {
		return nil, fmt.Errorf("查询实例规格失败: %v", err)
	}

	return response.Response.InstanceTypeConfigSet, nil
}

// ModifyInstancesRenewFlag 修改实例续费标识
// renewFlag: NOTIFY_AND_AUTO_RENEW / NOTIFY_AND_MANUAL_RENEW / DISABLE_NOTIFY_AND_MANUAL_RENEW
func ModifyInstancesRenewFlag(cred *CloudAccountInfo, regionId string, instanceIds []string, renewFlag string) error {
	client, err := NewCvmClient(cred.AccessKey, cred.AccessSecret, regionId)
	if err != nil {
		return fmt.Errorf("创建CVM客户端失败: %v", err)
	}

	request := cvm.NewModifyInstancesRenewFlagRequest()
	for _, id := range instanceIds {
		request.InstanceIds = append(request.InstanceIds, common.StringPtr(id))
	}
	request.RenewFlag = common.StringPtr(renewFlag)

	_, err = client.ModifyInstancesRenewFlag(request)
	if err != nil {
		return fmt.Errorf("修改续费标识失败: %v", err)
	}
	return nil
}
