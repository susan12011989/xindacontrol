package aliyun

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"server/internal/dbhelper"
	"server/pkg/dbs"
	"server/pkg/entity"
	"strings"
	"time"

	ecs20140526 "github.com/alibabacloud-go/ecs-20140526/v6/client"
	util "github.com/alibabacloud-go/tea-utils/v2/service"
	"github.com/alibabacloud-go/tea/tea"
	"github.com/zeromicro/go-zero/core/logx"
	"golang.org/x/crypto/ssh"
)

type CreateInstanceRequest struct {
	MerchantId         int    `json:"merchant_id"`
	CloudAccountId     int64  `json:"cloud_account_id"`
	Region             string `json:"region"`
	ImageId            string `json:"image_id"`             // 镜像id
	InstanceType       string `json:"instance_type"`        // 规格
	InstanceChargeType string `json:"instance_charge_type"` // 付费类型 PrePaid：包年包月 PostPaid：按量付费
	PeriodUnit         string `json:"period_unit"`          // 时长单位 Month：月 Week：周
	// PeriodUnit=Week 时，Period 取值：1、2、3、4
	// PeriodUnit=Month 时，Period 取值：1、2、3、4、5、6、7、8、9、12、24、36、48、60。
	Period int32 `json:"period"`
	/*
		系统盘的云盘种类。取值范围：
		cloud_efficiency：高效云盘。
		cloud_ssd：SSD 云盘。
		cloud_essd：ESSD 云盘。
		cloud：普通云盘。
		cloud_auto：ESSD AutoPL 云盘。
		cloud_essd_entry：ESSD Entry 云盘。
	*/
	DiskCategory  string `json:"disk_category"`
	DiskSize      int32  `json:"disk_size"`       // 系统盘大小 40G
	Password      string `json:"password"`        // SSH登录密码，8-30个字符，必须包含大小写字母、数字
	UsePassword   bool   `json:"use_password"`    // 是否使用密码认证（true=密码认证，false=自动创建密钥对）
	KeyPairName   string `json:"key_pair_name"`   // SSH密钥对名称（阿里云密钥对名称，可选）
	SshPrivateKey string `json:"ssh_private_key"` // SSH私钥内容（PEM格式，用于保存到数据库）
	AutoRenew     bool   `json:"auto_renew"`      // 是否自动续费（仅包年包月生效）
	AutoRenewPeriod int32 `json:"auto_renew_period"` // 自动续费周期（月），默认与购买周期一致
}

// CreateInstanceResult 创建实例的结果
type CreateInstanceResult struct {
	InstanceId   string `json:"instance_id"`
	InstanceName string `json:"instance_name"`
	PublicIp     string `json:"public_ip"`
	Password     string `json:"password"`      // 如果使用密码认证
	KeyPairName  string `json:"key_pair_name"` // 如果使用密钥认证
}

// getDefaultImageId 获取默认Ubuntu镜像ID（根据地区）
func getDefaultImageId(region string) string {
	// 阿里云各地区Ubuntu 22.04 LTS镜像ID
	defaultImages := map[string]string{
		"cn-hangzhou":      "ubuntu_22_04_x64_20G_alibase_20240926.vhd",
		"cn-shanghai":      "ubuntu_22_04_x64_20G_alibase_20240926.vhd",
		"cn-shenzhen":      "ubuntu_22_04_x64_20G_alibase_20240926.vhd",
		"cn-beijing":       "ubuntu_22_04_x64_20G_alibase_20240926.vhd",
		"cn-hongkong":      "ubuntu_22_04_x64_20G_alibase_20240926.vhd",
		"ap-southeast-1":   "ubuntu_22_04_x64_20G_alibase_20240926.vhd", // 新加坡
		"ap-northeast-1":   "ubuntu_22_04_x64_20G_alibase_20240926.vhd", // 东京
		"us-west-1":        "ubuntu_22_04_x64_20G_alibase_20240926.vhd", // 美西
		"eu-central-1":     "ubuntu_22_04_x64_20G_alibase_20240926.vhd", // 法兰克福
	}
	if imageId, ok := defaultImages[region]; ok {
		return imageId
	}
	// 默认返回通用镜像名称
	return "ubuntu_22_04_x64_20G_alibase_20240926.vhd"
}

// CreateKeyPair 创建SSH密钥对（私钥只返回一次）
func CreateKeyPair(client *ecs20140526.Client, region, keyPairName string) (privateKey string, err error) {
	request := &ecs20140526.CreateKeyPairRequest{
		RegionId:    tea.String(region),
		KeyPairName: tea.String(keyPairName),
	}
	response, err := client.CreateKeyPair(request)
	if err != nil {
		return "", fmt.Errorf("create key pair failed: %v", err)
	}
	logx.Infof("created key pair: %s", keyPairName)
	return *response.Body.PrivateKeyBody, nil
}

// CreateInstance 创建实例
func CreateInstance(req *CreateInstanceRequest) (*CreateInstanceResult, error) {
	if req.Region == "" {
		return nil, errors.New("region is required")
	}
	// 如果未指定镜像ID，使用默认Ubuntu 22.04镜像
	if req.ImageId == "" {
		req.ImageId = getDefaultImageId(req.Region)
	}
	if req.InstanceType == "" {
		return nil, errors.New("instance_type is required")
	}

	var cloud *CloudAccountInfo
	var err error

	if req.CloudAccountId > 0 {
		cloud, err = GetSystemCloudAccount(req.CloudAccountId)
	} else {
		cloud, err = GetMerchantCloud(req.MerchantId)
	}
	if err != nil {
		return nil, err
	}

	client, err := NewEcsClient(cloud.AccessKey, cloud.AccessSecret, req.Region)
	if err != nil {
		return nil, err
	}

	var instanceName string
	if req.MerchantId > 0 {
		merchant, err := dbhelper.GetMerchantByID(req.MerchantId)
		if err != nil {
			return nil, err
		}
		instanceName = fmt.Sprintf("traffic-%s-%s", merchant.Name, req.Region)
	} else {
		instanceName = fmt.Sprintf("traffic-system-%s", req.Region)
	}

	// 创建实例(使用默认安全组)
	request := &ecs20140526.CreateInstanceRequest{
		RegionId:           tea.String(req.Region),
		ImageId:            tea.String(req.ImageId),
		InstanceType:       tea.String(req.InstanceType),
		InstanceName:       tea.String(instanceName),
		InstanceChargeType: tea.String(req.InstanceChargeType),
		SystemDisk: &ecs20140526.CreateInstanceRequestSystemDisk{
			Category: tea.String(req.DiskCategory),
			Size:     tea.Int32(req.DiskSize),
		},
	}

	// 认证信息 - 根据用户选择决定使用密码还是密钥
	var authInfo *ServerAuthInfo

	if req.UsePassword {
		// 用户选择密码认证
		password := req.Password
		if password == "" {
			password = generateSecurePassword()
		}
		request.Password = tea.String(password)
		authInfo = &ServerAuthInfo{
			AuthType: 1, // 密码认证
			Password: password,
		}
		logx.Infof("creating instance with password authentication (user selected)")
	} else {
		// 默认使用SSH密钥对
		keyPairName := fmt.Sprintf("control-auto-%s-%d", req.Region, time.Now().Unix())
		privateKey, err := CreateKeyPair(client, req.Region, keyPairName)
		if err != nil {
			logx.Errorf("create key pair failed, fallback to password: %v", err)
			// 如果创建密钥对失败，回退到密码认证
			password := req.Password
			if password == "" {
				password = generateSecurePassword()
			}
			request.Password = tea.String(password)
			authInfo = &ServerAuthInfo{
				AuthType: 1, // 密码认证
				Password: password,
			}
			logx.Infof("creating instance with password authentication (fallback)")
		} else {
			// 使用自动创建的密钥对
			request.KeyPairName = tea.String(keyPairName)
			authInfo = &ServerAuthInfo{
				AuthType:   2, // 密钥认证
				KeyName:    keyPairName,
				PrivateKey: privateKey,
			}
			logx.Infof("creating instance with auto-created SSH key pair: %s", keyPairName)
		}
	}

	if req.InstanceChargeType == "PrePaid" {
		request.PeriodUnit = tea.String(req.PeriodUnit)
		request.Period = tea.Int32(req.Period)
		if req.AutoRenew {
			request.AutoRenew = tea.Bool(true)
			if req.AutoRenewPeriod > 0 {
				request.AutoRenewPeriod = tea.Int32(req.AutoRenewPeriod)
			} else {
				request.AutoRenewPeriod = tea.Int32(req.Period)
			}
		}
	}

	response, err := client.CreateInstance(request)
	if err != nil {
		return nil, err
	}
	instanceId := *response.Body.InstanceId
	logx.Infof("create instance success: %s", instanceId)

	result := &CreateInstanceResult{
		InstanceId:   instanceId,
		InstanceName: instanceName,
		Password:     authInfo.Password,
		KeyPairName:  authInfo.KeyName,
	}

	// 异步处理：授权安全组 + 注册服务器到数据库
	go createAfterAuthorizeSecurityGroupAndRegister(req.MerchantId, req.CloudAccountId, req.Region, instanceId, instanceName, authInfo, cloud)
	return result, nil
}

// ServerAuthInfo 服务器认证信息
type ServerAuthInfo struct {
	AuthType   int    // 1=密码 2=密钥
	Password   string // 密码（AuthType=1时使用）
	KeyName    string // 密钥对名称（AuthType=2时使用）
	PrivateKey string // 私钥内容（AuthType=2时使用）
}

// generateSecurePassword 生成安全密码（8-30位，包含大小写字母和数字）
func generateSecurePassword() string {
	const (
		lowerChars = "abcdefghijklmnopqrstuvwxyz"
		upperChars = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
		digitChars = "0123456789"
		allChars   = lowerChars + upperChars + digitChars
	)
	// 生成16位密码
	password := make([]byte, 16)
	// 确保至少有一个小写字母、一个大写字母、一个数字
	password[0] = lowerChars[time.Now().UnixNano()%int64(len(lowerChars))]
	password[1] = upperChars[time.Now().UnixNano()%int64(len(upperChars))]
	password[2] = digitChars[time.Now().UnixNano()%int64(len(digitChars))]
	// 填充剩余位置
	for i := 3; i < 16; i++ {
		password[i] = allChars[(time.Now().UnixNano()+int64(i*1000))%int64(len(allChars))]
		time.Sleep(time.Nanosecond) // 增加随机性
	}
	return string(password)
}

// createAfterAuthorizeSecurityGroupAndRegister 创建实例后的后续处理：
// 1. 等待实例初始化
// 2. 授权安全组规则
// 3. 启动实例
// 4. 等待获取公网IP
// 5. 注册服务器到数据库
func createAfterAuthorizeSecurityGroupAndRegister(merchantId int, cloudAccountId int64, region string, instanceId string, instanceName string, authInfo *ServerAuthInfo, cloud *CloudAccountInfo) {
	// 等待30秒让阿里云为实例分配安全组
	logx.Infof("waiting 30 seconds for instance %s to initialize security group", instanceId)
	time.Sleep(30 * time.Second)

	client, err := NewEcsClient(cloud.AccessKey, cloud.AccessSecret, region)
	if err != nil {
		logx.Errorf("new ecs client failed: %v", err)
		return
	}

	// 获取实例详情以获取默认安全组ID
	describeReq := &ecs20140526.DescribeInstancesRequest{
		RegionId:    tea.String(region),
		InstanceIds: tea.String(fmt.Sprintf("[\"%s\"]", instanceId)),
	}
	describeResp, err := client.DescribeInstances(describeReq)
	if err != nil {
		logx.Errorf("describe instance failed: %v", err)
		return
	}

	if describeResp.Body.Instances == nil || len(describeResp.Body.Instances.Instance) == 0 {
		logx.Errorf("instance not found: %s", instanceId)
		return
	}

	instance := describeResp.Body.Instances.Instance[0]
	if instance.SecurityGroupIds == nil || len(instance.SecurityGroupIds.SecurityGroupId) == 0 {
		logx.Errorf("instance has no security group: %s", instanceId)
		return
	}

	// 获取默认安全组ID
	securityGroupId := *instance.SecurityGroupIds.SecurityGroupId[0]
	logx.Infof("instance %s default security group: %s", instanceId, securityGroupId)

	// 在默认安全组上添加TCP 1-65535全网开放规则
	err = AuthorizeSecurityGroup(&AuthorizeSecurityGroupRequest{
		MerchantId:      merchantId,
		CloudAccountId:  cloudAccountId,
		RegionId:        region,
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
		// 规则可能已存在,记录日志但不返回错误
		logx.Errorf("authorize security group failed (may already exist): %v", err)
	} else {
		logx.Infof("authorize security group success for TCP 1-65535 on group %s", securityGroupId)
	}

	// 启动实例
	logx.Infof("starting instance %s", instanceId)
	startReq := &ecs20140526.StartInstanceRequest{
		InstanceId: tea.String(instanceId),
	}
	_, err = client.StartInstance(startReq)
	if err != nil {
		logx.Errorf("start instance failed: %v", err)
		return
	}
	logx.Infof("instance %s started", instanceId)

	// 等待实例运行并获取公网IP（最多等待5分钟）
	var publicIp string
	for i := 0; i < 30; i++ {
		time.Sleep(10 * time.Second)
		describeResp, err := client.DescribeInstances(describeReq)
		if err != nil {
			logx.Errorf("describe instance failed: %v", err)
			continue
		}
		if describeResp.Body.Instances == nil || len(describeResp.Body.Instances.Instance) == 0 {
			continue
		}
		inst := describeResp.Body.Instances.Instance[0]

		// 检查公网IP（可能来自多个来源）
		if inst.PublicIpAddress != nil && inst.PublicIpAddress.IpAddress != nil && len(inst.PublicIpAddress.IpAddress) > 0 {
			publicIp = *inst.PublicIpAddress.IpAddress[0]
			break
		}
		// 检查EIP
		if inst.EipAddress != nil && inst.EipAddress.IpAddress != nil && *inst.EipAddress.IpAddress != "" {
			publicIp = *inst.EipAddress.IpAddress
			break
		}
		logx.Infof("waiting for public IP... attempt %d/30", i+1)
	}

	if publicIp == "" {
		logx.Errorf("failed to get public IP for instance %s after 5 minutes", instanceId)
		return
	}

	logx.Infof("instance %s got public IP: %s", instanceId, publicIp)

	// 注册服务器到数据库
	err = registerServerToDatabase(instanceId, instanceName, publicIp, authInfo, merchantId)
	if err != nil {
		logx.Errorf("register server to database failed: %v", err)
		return
	}
	logx.Infof("server registered to database: %s (%s)", instanceName, publicIp)

	// 自动安装 GOST 服务
	logx.Infof("starting GOST installation on %s", publicIp)
	err = autoInstallGost(publicIp, authInfo)
	if err != nil {
		logx.Errorf("auto install GOST failed: %v (server still registered, GOST can be installed manually)", err)
		return
	}
	logx.Infof("GOST installed successfully on %s", publicIp)
}

// registerServerToDatabase 将新创建的实例注册到服务器表
func registerServerToDatabase(instanceId, instanceName, publicIp string, authInfo *ServerAuthInfo, merchantId int) error {
	now := time.Now()

	// 确定服务器类型：如果有 merchantId 则是商户服务器(1)，否则是系统服务器(2)
	serverType := 2 // 默认系统服务器
	if merchantId > 0 {
		serverType = 1 // 商户服务器
	}

	server := &entity.Servers{
		ServerType:  serverType,
		MerchantId:  merchantId,
		Name:        instanceName,
		Host:        publicIp,
		Port:        22,
		Username:    "root",
		AuthType:    authInfo.AuthType,
		Password:    authInfo.Password,
		PrivateKey:  authInfo.PrivateKey,
		DeployPath:  "/opt/teamgram/bin",
		Status:      1, // 启用
		Description: fmt.Sprintf("Auto-created from ECS instance %s", instanceId),
		ForwardType: 1, // 加密转发
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	_, err := dbs.DBAdmin.Insert(server)
	return err
}

// StartInstance 启动实例
func StartInstance(merchantId int, cloudAccountId int64, region, instanceId string) error {
	var cloud *CloudAccountInfo
	var err error

	if cloudAccountId > 0 {
		cloud, err = GetSystemCloudAccount(cloudAccountId)
	} else {
		cloud, err = GetMerchantCloud(merchantId)
	}
	if err != nil {
		return err
	}
	client, err := NewEcsClient(cloud.AccessKey, cloud.AccessSecret, region)
	if err != nil {
		return err
	}
	request := &ecs20140526.StartInstanceRequest{
		InstanceId: tea.String(instanceId),
	}
	response, err := client.StartInstance(request)
	if err != nil {
		return err
	}
	if *response.StatusCode != 200 {
		return errors.New(response.String())
	}
	return nil
}

// StopInstance 停止实例
func StopInstance(merchantId int, cloudAccountId int64, region, instanceId string) error {
	var cloud *CloudAccountInfo
	var err error

	if cloudAccountId > 0 {
		cloud, err = GetSystemCloudAccount(cloudAccountId)
	} else {
		cloud, err = GetMerchantCloud(merchantId)
	}
	if err != nil {
		return err
	}
	client, err := NewEcsClient(cloud.AccessKey, cloud.AccessSecret, region)
	if err != nil {
		return err
	}
	request := &ecs20140526.StopInstanceRequest{
		InstanceId: tea.String(instanceId),
	}
	response, err := client.StopInstance(request)
	if err != nil {
		return err
	}
	if *response.StatusCode != 200 {
		return errors.New(response.String())
	}
	return nil
}

// RebootInstance 重启实例
func RebootInstance(merchantId int, cloudAccountId int64, region, instanceId string) error {
	var cloud *CloudAccountInfo
	var err error

	if cloudAccountId > 0 {
		cloud, err = GetSystemCloudAccount(cloudAccountId)
	} else {
		cloud, err = GetMerchantCloud(merchantId)
	}
	if err != nil {
		return err
	}
	client, err := NewEcsClient(cloud.AccessKey, cloud.AccessSecret, region)
	if err != nil {
		return err
	}
	request := &ecs20140526.RebootInstanceRequest{
		InstanceId: tea.String(instanceId),
	}
	response, err := client.RebootInstance(request)
	if err != nil {
		return err
	}
	if *response.StatusCode != 200 {
		return errors.New(response.String())
	}
	return nil
}

// DescribeInstanceStatus 获取实例状态
func DescribeInstanceStatus(merchantId int, region string) (map[string]string, error) {
	cloud, err := GetMerchantCloud(merchantId)
	if err != nil {
		return nil, err
	}
	client, err := NewEcsClient(cloud.AccessKey, cloud.AccessSecret, region)
	if err != nil {
		return nil, err
	}
	request := &ecs20140526.DescribeInstanceStatusRequest{
		RegionId:   tea.String(region),
		PageNumber: tea.Int32(1),
		PageSize:   tea.Int32(50),
	}
	response, err := client.DescribeInstanceStatusWithOptions(request, &util.RuntimeOptions{})
	if err != nil {
		return nil, err
	}
	statusMap := make(map[string]string)
	for _, ss := range response.Body.InstanceStatuses.InstanceStatus {
		statusMap[*ss.InstanceId] = *ss.Status
	}
	return statusMap, nil
}

// DescribeInstances 查看实例详情(列表) - 使用商户ID
func DescribeInstances(merchantId int, region string) ([]*ecs20140526.DescribeInstancesResponseBodyInstancesInstance, error) {
	cloud, err := GetMerchantCloud(merchantId)
	if err != nil {
		logx.Errorf("获取商户 %d 的云账号配置失败: %v", merchantId, err)
		return nil, err
	}
	logx.Infof("商户 %d 云账号: AccessKey=%s", merchantId, cloud.AccessKey)

	client, err := NewEcsClient(cloud.AccessKey, cloud.AccessSecret, region)
	if err != nil {
		logx.Errorf("创建ECS客户端失败(region=%s): %v", region, err)
		return nil, err
	}

	request := &ecs20140526.DescribeInstancesRequest{
		RegionId:   tea.String(region),
		PageNumber: tea.Int32(1),
		PageSize:   tea.Int32(100),
	}

	logx.Infof("调用阿里云API查询实例列表: MerchantId=%d, Region=%s", merchantId, region)
	response, err := client.DescribeInstances(request)
	if err != nil {
		logx.Errorf("调用阿里云API失败: %v", err)
		// 打印更详细的错误信息
		if sdkErr, ok := err.(*tea.SDKError); ok {
			logx.Errorf("SDK错误详情: Code=%s, Message=%s, RequestId=%s",
				tea.StringValue(sdkErr.Code),
				tea.StringValue(sdkErr.Message),
				tea.StringValue(sdkErr.Data))
		}
		return nil, err
	}

	instanceCount := 0
	if response.Body != nil && response.Body.Instances != nil {
		instanceCount = len(response.Body.Instances.Instance)
	}
	logx.Infof("查询成功, 区域 %s 返回 %d 个实例", region, instanceCount)

	return response.Body.Instances.Instance, nil
}

// DescribeInstancesByCloudAccount 查看实例详情(列表) - 使用系统云账号ID
func DescribeInstancesByCloudAccount(cloudAccountId int64, region string) ([]*ecs20140526.DescribeInstancesResponseBodyInstancesInstance, error) {
	cloud, err := GetSystemCloudAccount(cloudAccountId)
	if err != nil {
		logx.Errorf("获取云账号 %d 配置失败: %v", cloudAccountId, err)
		return nil, err
	}
	logx.Infof("云账号 %d AccessKey=%s", cloudAccountId, cloud.AccessKey)

	client, err := NewEcsClient(cloud.AccessKey, cloud.AccessSecret, region)
	if err != nil {
		logx.Errorf("创建ECS客户端失败(region=%s): %v", region, err)
		return nil, err
	}

	request := &ecs20140526.DescribeInstancesRequest{
		RegionId:   tea.String(region),
		PageNumber: tea.Int32(1),
		PageSize:   tea.Int32(100),
	}

	logx.Infof("调用阿里云API查询实例列表: CloudAccountId=%d, Region=%s", cloudAccountId, region)
	response, err := client.DescribeInstances(request)
	if err != nil {
		logx.Errorf("调用阿里云API失败: %v", err)
		if sdkErr, ok := err.(*tea.SDKError); ok {
			logx.Errorf("SDK错误详情: Code=%s, Message=%s, RequestId=%s",
				tea.StringValue(sdkErr.Code),
				tea.StringValue(sdkErr.Message),
				tea.StringValue(sdkErr.Data))
		}
		return nil, err
	}

	instanceCount := 0
	if response.Body != nil && response.Body.Instances != nil {
		instanceCount = len(response.Body.Instances.Instance)
	}
	logx.Infof("查询成功, 区域 %s 返回 %d 个实例", region, instanceCount)

	return response.Body.Instances.Instance, nil
}

type ModifyInstanceAttributeRequest struct {
	MerchantId      int       `json:"merchant_id"`
	CloudAccountId  int64     `json:"cloud_account_id"`
	RegionId        string    `json:"region_id"`
	InstanceId      string    `json:"instance_id"`
	InstanceName    string    `json:"instance_name"`
	Description     string    `json:"description"`
	Password        string    `json:"password"`          // 密码
	SecurityGroupId []*string `json:"security_group_id"` // 安全组
}

func ModifyInstanceAttribute(req *ModifyInstanceAttributeRequest) error {
	var cloud *CloudAccountInfo
	var err error

	if req.CloudAccountId > 0 {
		cloud, err = GetSystemCloudAccount(req.CloudAccountId)
	} else {
		cloud, err = GetMerchantCloud(req.MerchantId)
	}
	if err != nil {
		return err
	}
	client, err := NewEcsClient(cloud.AccessKey, cloud.AccessSecret, req.RegionId)
	if err != nil {
		return err
	}
	request := &ecs20140526.ModifyInstanceAttributeRequest{
		InstanceId: tea.String(req.InstanceId),
	}
	if req.InstanceName != "" {
		request.InstanceName = tea.String(req.InstanceName)
	}
	if req.Description != "" {
		request.Description = tea.String(req.Description)
	}
	if req.Password != "" {
		request.Password = tea.String(req.Password)
	}
	if len(req.SecurityGroupId) > 0 {
		request.SecurityGroupIds = req.SecurityGroupId
	}
	response, err := client.ModifyInstanceAttribute(request)
	if err != nil {
		return err
	}
	if *response.StatusCode != 200 {
		return errors.New(response.String())
	}
	return nil
}

// DescribeImages 查看可用的镜像列表（包括自己创建的和他人共享的镜像）
func DescribeImages(cloudAccountId int64, merchantId int, region string) ([]*ecs20140526.DescribeImagesResponseBodyImagesImage, error) {
	var cloud *CloudAccountInfo
	var err error

	if cloudAccountId > 0 {
		cloud, err = GetSystemCloudAccount(cloudAccountId)
	} else {
		cloud, err = GetMerchantCloud(merchantId)
	}
	client, err := NewEcsClient(cloud.AccessKey, cloud.AccessSecret, region)
	if err != nil {
		return nil, err
	}

	// 合并结果
	var allImages []*ecs20140526.DescribeImagesResponseBodyImagesImage

	// 1. 查询自己的镜像
	requestSelf := &ecs20140526.DescribeImagesRequest{
		RegionId:        tea.String(region),
		OSType:          tea.String("linux"),
		ImageOwnerAlias: tea.String("self"),
		PageSize:        tea.Int32(100),
	}
	responseSelf, err := client.DescribeImages(requestSelf)
	if err != nil {
		return nil, err
	}
	if responseSelf.Body.Images != nil && responseSelf.Body.Images.Image != nil {
		allImages = append(allImages, responseSelf.Body.Images.Image...)
	}

	// 2. 查询他人共享的镜像
	requestOthers := &ecs20140526.DescribeImagesRequest{
		RegionId:        tea.String(region),
		OSType:          tea.String("linux"),
		ImageOwnerAlias: tea.String("others"),
		PageSize:        tea.Int32(100),
	}
	responseOthers, err := client.DescribeImages(requestOthers)
	if err != nil {
		// 如果查询共享镜像失败，不影响返回自己的镜像
		return allImages, nil
	}
	if responseOthers.Body.Images != nil && responseOthers.Body.Images.Image != nil {
		allImages = append(allImages, responseOthers.Body.Images.Image...)
	}

	return allImages, nil
}

// ModifyInstanceChargeType 修改实例付费类型
func ModifyInstanceChargeType(cloudAccountId int64, merchantId int, region, instanceId string) error {
	var cloud *CloudAccountInfo
	var err error

	if cloudAccountId > 0 {
		cloud, err = GetSystemCloudAccount(cloudAccountId)
	} else {
		cloud, err = GetMerchantCloud(merchantId)
	}
	client, err := NewEcsClient(cloud.AccessKey, cloud.AccessSecret, region)
	if err != nil {
		return err
	}
	ss := []string{instanceId}
	jsonData, _ := json.Marshal(ss)
	request := &ecs20140526.ModifyInstanceChargeTypeRequest{
		RegionId:           tea.String(region),
		IsDetailFee:        tea.Bool(true),               // 是否返回详细的费用信息
		InstanceIds:        tea.String(string(jsonData)), // ["id1","id2"]
		InstanceChargeType: tea.String("PostPaid"),       // 将包年包月实例改为按量付费
	}
	response, err := client.ModifyInstanceChargeType(request)
	if err != nil {
		return err
	}
	if *response.StatusCode != 200 {
		return errors.New(response.String())
	}
	logx.Infof("修改实例付费类型成功: %s %+v", instanceId, response.Body)
	return nil
}

// DeleteInstance 删除实例
func DeleteInstance(merchantId int, cloudAccountId int64, region, instanceId string) error {
	var cloud *CloudAccountInfo
	var err error

	if cloudAccountId > 0 {
		cloud, err = GetSystemCloudAccount(cloudAccountId)
	} else {
		cloud, err = GetMerchantCloud(merchantId)
	}
	if err != nil {
		return err
	}
	client, err := NewEcsClient(cloud.AccessKey, cloud.AccessSecret, region)
	if err != nil {
		return err
	}
	request := &ecs20140526.DeleteInstanceRequest{
		InstanceId: tea.String(instanceId),
		Force:      tea.Bool(true), // 强制释放
	}
	response, err := client.DeleteInstance(request)
	if err != nil {
		return err
	}
	if *response.StatusCode != 200 {
		return errors.New(response.String())
	}
	return nil
}

// ========== 镜像共享相关 ==========

// ModifyImageSharePermissionRequest 修改镜像共享权限请求
type ModifyImageSharePermissionRequest struct {
	CloudAccountId int64    `json:"cloud_account_id"`
	MerchantId     int      `json:"merchant_id"`
	RegionId       string   `json:"region_id" binding:"required"`
	ImageId        string   `json:"image_id" binding:"required"`
	AddAccounts    []string `json:"add_accounts"`    // 要添加共享的阿里云账号ID列表
	RemoveAccounts []string `json:"remove_accounts"` // 要取消共享的阿里云账号ID列表
}

// ModifyImageSharePermission 修改镜像共享权限
func ModifyImageSharePermission(req *ModifyImageSharePermissionRequest) error {
	var cloud *CloudAccountInfo
	var err error

	if req.CloudAccountId > 0 {
		cloud, err = GetSystemCloudAccount(req.CloudAccountId)
	} else {
		cloud, err = GetMerchantCloud(req.MerchantId)
	}
	if err != nil {
		return err
	}

	client, err := NewEcsClient(cloud.AccessKey, cloud.AccessSecret, req.RegionId)
	if err != nil {
		return err
	}

	request := &ecs20140526.ModifyImageSharePermissionRequest{
		RegionId: tea.String(req.RegionId),
		ImageId:  tea.String(req.ImageId),
	}

	// 添加共享账号
	if len(req.AddAccounts) > 0 {
		addAccountIds := make([]*string, len(req.AddAccounts))
		for i, acc := range req.AddAccounts {
			addAccountIds[i] = tea.String(acc)
		}
		request.AddAccount = addAccountIds
	}

	// 移除共享账号
	if len(req.RemoveAccounts) > 0 {
		removeAccountIds := make([]*string, len(req.RemoveAccounts))
		for i, acc := range req.RemoveAccounts {
			removeAccountIds[i] = tea.String(acc)
		}
		request.RemoveAccount = removeAccountIds
	}

	response, err := client.ModifyImageSharePermission(request)
	if err != nil {
		return err
	}
	if *response.StatusCode != 200 {
		return errors.New(response.String())
	}

	logx.Infof("修改镜像共享权限成功: ImageId=%s, AddAccounts=%v, RemoveAccounts=%v",
		req.ImageId, req.AddAccounts, req.RemoveAccounts)
	return nil
}

// ImageShareAccount 镜像共享账号信息
type ImageShareAccount struct {
	AliyunId string `json:"aliyun_id"` // 阿里云账号ID
}

// DescribeImageSharePermissionResponse 查询镜像共享权限响应
type DescribeImageSharePermissionResponse struct {
	ImageId       string              `json:"image_id"`
	RegionId      string              `json:"region_id"`
	TotalCount    int32               `json:"total_count"`
	ShareAccounts []ImageShareAccount `json:"share_accounts"`
}

// DescribeImageSharePermission 查询镜像共享权限
func DescribeImageSharePermission(cloudAccountId int64, merchantId int, regionId, imageId string) (*DescribeImageSharePermissionResponse, error) {
	var cloud *CloudAccountInfo
	var err error

	if cloudAccountId > 0 {
		cloud, err = GetSystemCloudAccount(cloudAccountId)
	} else {
		cloud, err = GetMerchantCloud(merchantId)
	}
	if err != nil {
		return nil, err
	}

	client, err := NewEcsClient(cloud.AccessKey, cloud.AccessSecret, regionId)
	if err != nil {
		return nil, err
	}

	request := &ecs20140526.DescribeImageSharePermissionRequest{
		RegionId: tea.String(regionId),
		ImageId:  tea.String(imageId),
	}

	response, err := client.DescribeImageSharePermission(request)
	if err != nil {
		return nil, err
	}

	result := &DescribeImageSharePermissionResponse{
		ImageId:       imageId,
		RegionId:      regionId,
		TotalCount:    *response.Body.TotalCount,
		ShareAccounts: make([]ImageShareAccount, 0),
	}

	if response.Body.Accounts != nil && response.Body.Accounts.Account != nil {
		for _, acc := range response.Body.Accounts.Account {
			result.ShareAccounts = append(result.ShareAccounts, ImageShareAccount{
				AliyunId: *acc.AliyunId,
			})
		}
	}

	return result, nil
}

// CreateImageRequest 创建镜像请求
type CreateImageRequest struct {
	CloudAccountId int64  `json:"cloud_account_id"`
	MerchantId     int    `json:"merchant_id"`
	RegionId       string `json:"region_id" binding:"required"`
	InstanceId     string `json:"instance_id" binding:"required"` // 从实例创建镜像
	ImageName      string `json:"image_name" binding:"required"`
	Description    string `json:"description"`
}

// CreateImage 从实例创建自定义镜像
func CreateImage(req *CreateImageRequest) (string, error) {
	var cloud *CloudAccountInfo
	var err error

	if req.CloudAccountId > 0 {
		cloud, err = GetSystemCloudAccount(req.CloudAccountId)
	} else {
		cloud, err = GetMerchantCloud(req.MerchantId)
	}
	if err != nil {
		return "", err
	}

	client, err := NewEcsClient(cloud.AccessKey, cloud.AccessSecret, req.RegionId)
	if err != nil {
		return "", err
	}

	request := &ecs20140526.CreateImageRequest{
		RegionId:    tea.String(req.RegionId),
		InstanceId:  tea.String(req.InstanceId),
		ImageName:   tea.String(req.ImageName),
		Description: tea.String(req.Description),
	}

	response, err := client.CreateImage(request)
	if err != nil {
		return "", err
	}

	imageId := *response.Body.ImageId
	logx.Infof("创建镜像成功: ImageId=%s, ImageName=%s, InstanceId=%s", imageId, req.ImageName, req.InstanceId)
	return imageId, nil
}

// AttachKeyPair 将密钥对绑定到实例（实例必须处于停止状态）
func AttachKeyPair(client *ecs20140526.Client, region, keyPairName, instanceId string) error {
	request := &ecs20140526.AttachKeyPairRequest{
		RegionId:    tea.String(region),
		KeyPairName: tea.String(keyPairName),
		InstanceIds: tea.String(fmt.Sprintf("[\"%s\"]", instanceId)),
	}
	_, err := client.AttachKeyPair(request)
	if err != nil {
		return fmt.Errorf("attach key pair failed: %v", err)
	}
	logx.Infof("attached key pair %s to instance %s", keyPairName, instanceId)
	return nil
}

// DetachKeyPair 解绑密钥对
func DetachKeyPair(client *ecs20140526.Client, region, keyPairName, instanceId string) error {
	request := &ecs20140526.DetachKeyPairRequest{
		RegionId:    tea.String(region),
		KeyPairName: tea.String(keyPairName),
		InstanceIds: tea.String(fmt.Sprintf("[\"%s\"]", instanceId)),
	}
	_, err := client.DetachKeyPair(request)
	if err != nil {
		return fmt.Errorf("detach key pair failed: %v", err)
	}
	logx.Infof("detached key pair %s from instance %s", keyPairName, instanceId)
	return nil
}

// RegisterInstanceWithSSHKeyRequest 注册实例请求（自动创建SSH密钥）
type RegisterInstanceWithSSHKeyRequest struct {
	CloudAccountId int64  `json:"cloud_account_id"` // 云账号ID
	RegionId       string `json:"region_id"`        // 区域
	InstanceId     string `json:"instance_id"`      // 实例ID
	ServerName     string `json:"server_name"`      // 服务器名称
	ServerType     int    `json:"server_type"`      // 服务器类型：1-商户服务器 2-系统服务器
	PublicIp       string `json:"public_ip"`        // 公网IP
}

// RegisterInstanceWithSSHKeyResult 注册实例结果
type RegisterInstanceWithSSHKeyResult struct {
	ServerId    int    `json:"server_id"`
	ServerName  string `json:"server_name"`
	Host        string `json:"host"`
	KeyPairName string `json:"key_pair_name"`
	PrivateKey  string `json:"private_key"` // 私钥内容，用户需要下载保存
}

// RegisterInstanceWithSSHKey 注册实例到服务器管理（自动创建并绑定SSH密钥对）
// 流程：创建密钥对 -> 停止实例 -> 绑定密钥 -> 启动实例 -> 注册服务器
func RegisterInstanceWithSSHKey(req *RegisterInstanceWithSSHKeyRequest) (*RegisterInstanceWithSSHKeyResult, error) {
	if req.CloudAccountId == 0 {
		return nil, errors.New("cloud_account_id is required")
	}
	if req.RegionId == "" {
		return nil, errors.New("region_id is required")
	}
	if req.InstanceId == "" {
		return nil, errors.New("instance_id is required")
	}
	if req.ServerName == "" {
		return nil, errors.New("server_name is required")
	}
	if req.PublicIp == "" {
		return nil, errors.New("public_ip is required")
	}

	cloud, err := GetSystemCloudAccount(req.CloudAccountId)
	if err != nil {
		return nil, fmt.Errorf("get cloud account failed: %v", err)
	}

	client, err := NewEcsClient(cloud.AccessKey, cloud.AccessSecret, req.RegionId)
	if err != nil {
		return nil, fmt.Errorf("create ecs client failed: %v", err)
	}

	// 1. 创建SSH密钥对
	keyPairName := fmt.Sprintf("control-auto-%s-%d", req.RegionId, time.Now().Unix())
	privateKey, err := CreateKeyPair(client, req.RegionId, keyPairName)
	if err != nil {
		return nil, fmt.Errorf("create key pair failed: %v", err)
	}
	logx.Infof("created key pair: %s", keyPairName)

	// 2. 获取实例当前状态
	describeReq := &ecs20140526.DescribeInstancesRequest{
		RegionId:    tea.String(req.RegionId),
		InstanceIds: tea.String(fmt.Sprintf("[\"%s\"]", req.InstanceId)),
	}
	describeResp, err := client.DescribeInstances(describeReq)
	if err != nil {
		return nil, fmt.Errorf("describe instance failed: %v", err)
	}
	if describeResp.Body.Instances == nil || len(describeResp.Body.Instances.Instance) == 0 {
		return nil, fmt.Errorf("instance not found: %s", req.InstanceId)
	}
	instance := describeResp.Body.Instances.Instance[0]
	instanceStatus := *instance.Status

	// 3. 如果实例正在运行，需要先停止
	needRestart := false
	if instanceStatus == "Running" {
		logx.Infof("instance %s is running, stopping...", req.InstanceId)
		stopReq := &ecs20140526.StopInstanceRequest{
			InstanceId: tea.String(req.InstanceId),
			ForceStop:  tea.Bool(true),
		}
		_, err = client.StopInstance(stopReq)
		if err != nil {
			return nil, fmt.Errorf("stop instance failed: %v", err)
		}
		needRestart = true

		// 等待实例停止（最多2分钟）
		for i := 0; i < 24; i++ {
			time.Sleep(5 * time.Second)
			describeResp, err := client.DescribeInstances(describeReq)
			if err != nil {
				continue
			}
			if len(describeResp.Body.Instances.Instance) > 0 {
				status := *describeResp.Body.Instances.Instance[0].Status
				if status == "Stopped" {
					logx.Infof("instance %s stopped", req.InstanceId)
					break
				}
				logx.Infof("waiting for instance to stop, current status: %s", status)
			}
		}
	}

	// 4. 绑定密钥对
	err = AttachKeyPair(client, req.RegionId, keyPairName, req.InstanceId)
	if err != nil {
		return nil, fmt.Errorf("attach key pair failed: %v", err)
	}

	// 5. 如果之前是运行状态，重新启动实例
	if needRestart {
		logx.Infof("restarting instance %s", req.InstanceId)
		startReq := &ecs20140526.StartInstanceRequest{
			InstanceId: tea.String(req.InstanceId),
		}
		_, err = client.StartInstance(startReq)
		if err != nil {
			logx.Errorf("start instance failed: %v", err)
			// 不返回错误，继续保存服务器信息
		}

		// 等待实例启动（最多2分钟）
		for i := 0; i < 24; i++ {
			time.Sleep(5 * time.Second)
			describeResp, err := client.DescribeInstances(describeReq)
			if err != nil {
				continue
			}
			if len(describeResp.Body.Instances.Instance) > 0 {
				status := *describeResp.Body.Instances.Instance[0].Status
				if status == "Running" {
					logx.Infof("instance %s is running", req.InstanceId)
					break
				}
				logx.Infof("waiting for instance to start, current status: %s", status)
			}
		}
	}

	// 6. 获取实例的所有公网IP（包括主网卡和辅助网卡绑定的EIP）
	allPublicIPs := getInstanceAllPublicIPs(client, req.RegionId, req.InstanceId, instance)
	logx.Infof("instance %s has public IPs: %v", req.InstanceId, allPublicIPs)

	// 7. 检查是否已有服务器记录存在（通过任意一个公网IP匹配）
	var existingServer entity.Servers
	hasExisting := false
	for _, ip := range allPublicIPs {
		found, err := dbs.DBAdmin.Where("host = ?", ip).Get(&existingServer)
		if err != nil {
			logx.Errorf("check existing server by host %s failed: %v", ip, err)
			continue
		}
		if found {
			hasExisting = true
			logx.Infof("found existing server record: id=%d, host=%s", existingServer.Id, existingServer.Host)
			break
		}
	}

	now := time.Now()

	if hasExisting {
		// 服务器记录已存在，更新辅助IP
		auxiliaryIPs := []string{}
		// 收集除主IP外的其他IP作为辅助IP
		for _, ip := range allPublicIPs {
			if ip != existingServer.Host {
				auxiliaryIPs = append(auxiliaryIPs, ip)
			}
		}
		// 如果当前请求的IP不是主IP，也添加到辅助IP列表
		if req.PublicIp != existingServer.Host {
			found := false
			for _, ip := range auxiliaryIPs {
				if ip == req.PublicIp {
					found = true
					break
				}
			}
			if !found {
				auxiliaryIPs = append(auxiliaryIPs, req.PublicIp)
			}
		}

		// 合并现有的辅助IP
		if existingServer.AuxiliaryIP != "" {
			existingAux := strings.Split(existingServer.AuxiliaryIP, ",")
			for _, ip := range existingAux {
				ip = strings.TrimSpace(ip)
				if ip != "" && ip != existingServer.Host {
					found := false
					for _, aux := range auxiliaryIPs {
						if aux == ip {
							found = true
							break
						}
					}
					if !found {
						auxiliaryIPs = append(auxiliaryIPs, ip)
					}
				}
			}
		}

		auxiliaryIPStr := strings.Join(auxiliaryIPs, ",")

		// 更新辅助IP和私钥（如果需要）
		updates := map[string]interface{}{
			"auxiliary_ip": auxiliaryIPStr,
			"updated_at":   now,
		}
		// 如果原服务器没有私钥，则更新私钥
		if existingServer.PrivateKey == "" {
			updates["private_key"] = privateKey
			updates["auth_type"] = 2
		}

		_, err = dbs.DBAdmin.Table("servers").Where("id = ?", existingServer.Id).Update(updates)
		if err != nil {
			return nil, fmt.Errorf("update server auxiliary_ip failed: %v", err)
		}

		logx.Infof("updated existing server: id=%d, host=%s, auxiliary_ip=%s", existingServer.Id, existingServer.Host, auxiliaryIPStr)

		return &RegisterInstanceWithSSHKeyResult{
			ServerId:    existingServer.Id,
			ServerName:  existingServer.Name,
			Host:        existingServer.Host,
			KeyPairName: keyPairName,
			PrivateKey:  privateKey, // 返回私钥供用户下载备份
		}, nil
	}

	// 8. 创建新的服务器记录
	// 计算辅助IP（除请求的主IP外的其他IP）
	auxiliaryIPs := []string{}
	for _, ip := range allPublicIPs {
		if ip != req.PublicIp {
			auxiliaryIPs = append(auxiliaryIPs, ip)
		}
	}
	auxiliaryIPStr := strings.Join(auxiliaryIPs, ",")

	server := &entity.Servers{
		ServerType:  req.ServerType,
		Name:        req.ServerName,
		Host:        req.PublicIp,
		AuxiliaryIP: auxiliaryIPStr,
		Port:        22,
		Username:    "root",
		AuthType:    2, // 密钥认证
		PrivateKey:  privateKey,
		DeployPath:  "/opt/teamgram/bin",
		Status:      1,
		Description: fmt.Sprintf("Auto-registered from ECS instance %s with SSH key %s", req.InstanceId, keyPairName),
		ForwardType: 1,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	affected, err := dbs.DBAdmin.Insert(server)
	if err != nil {
		return nil, fmt.Errorf("insert server to database failed: %v", err)
	}
	if affected == 0 {
		return nil, fmt.Errorf("insert server failed: no rows affected")
	}

	serverId := server.Id
	logx.Infof("registered server: id=%d, name=%s, host=%s, auxiliary_ip=%s, key=%s", serverId, req.ServerName, req.PublicIp, auxiliaryIPStr, keyPairName)

	return &RegisterInstanceWithSSHKeyResult{
		ServerId:    int(serverId),
		ServerName:  req.ServerName,
		Host:        req.PublicIp,
		KeyPairName: keyPairName,
		PrivateKey:  privateKey, // 返回私钥供用户下载备份
	}, nil
}

// getInstanceAllPublicIPs 获取ECS实例的所有公网IP（包括主网卡和辅助网卡绑定的EIP）
// 需要调用DescribeNetworkInterfaces API获取完整的网卡信息
func getInstanceAllPublicIPs(client *ecs20140526.Client, regionId, instanceId string, instance *ecs20140526.DescribeInstancesResponseBodyInstancesInstance) []string {
	publicIPs := []string{}
	ipSet := make(map[string]bool)

	// 1. 从实例的PublicIpAddress获取
	if instance.PublicIpAddress != nil && instance.PublicIpAddress.IpAddress != nil {
		for _, ip := range instance.PublicIpAddress.IpAddress {
			if ip != nil && *ip != "" && !ipSet[*ip] {
				publicIPs = append(publicIPs, *ip)
				ipSet[*ip] = true
			}
		}
	}

	// 2. 从实例的EipAddress获取（主网卡绑定的EIP）
	if instance.EipAddress != nil && instance.EipAddress.IpAddress != nil && *instance.EipAddress.IpAddress != "" {
		ip := *instance.EipAddress.IpAddress
		if !ipSet[ip] {
			publicIPs = append(publicIPs, ip)
			ipSet[ip] = true
		}
	}

	// 3. 调用DescribeNetworkInterfaces获取所有网卡的EIP信息
	nicReq := &ecs20140526.DescribeNetworkInterfacesRequest{
		RegionId:   tea.String(regionId),
		InstanceId: tea.String(instanceId),
	}
	nicResp, err := client.DescribeNetworkInterfaces(nicReq)
	if err != nil {
		logx.Errorf("DescribeNetworkInterfaces for instance %s failed: %v", instanceId, err)
		return publicIPs
	}

	if nicResp.Body != nil && nicResp.Body.NetworkInterfaceSets != nil && nicResp.Body.NetworkInterfaceSets.NetworkInterfaceSet != nil {
		for _, nic := range nicResp.Body.NetworkInterfaceSets.NetworkInterfaceSet {
			if nic.PrivateIpSets != nil && nic.PrivateIpSets.PrivateIpSet != nil {
				for _, pip := range nic.PrivateIpSets.PrivateIpSet {
					if pip.AssociatedPublicIp != nil && pip.AssociatedPublicIp.PublicIpAddress != nil {
						ip := *pip.AssociatedPublicIp.PublicIpAddress
						if ip != "" && !ipSet[ip] {
							publicIPs = append(publicIPs, ip)
							ipSet[ip] = true
						}
					}
				}
			}
		}
	}

	return publicIPs
}

// GOST 配置常量
const (
	GostAPIPort  = 9394
	GostAPIUser  = "tsdd"
	GostAPIPass  = "Oa21isSdaiuwhq"
)

// autoInstallGost 自动安装 GOST 服务
func autoInstallGost(host string, authInfo *ServerAuthInfo) error {
	// 等待 SSH 服务可用
	logx.Infof("waiting for SSH service on %s...", host)
	if err := waitForSSHReady(host, 30); err != nil {
		return fmt.Errorf("SSH service not ready: %w", err)
	}
	logx.Infof("SSH service ready on %s", host)

	// 建立 SSH 连接
	var sshConfig *ssh.ClientConfig
	if authInfo.AuthType == 2 && authInfo.PrivateKey != "" {
		// 密钥认证
		signer, err := ssh.ParsePrivateKey([]byte(authInfo.PrivateKey))
		if err != nil {
			return fmt.Errorf("parse private key failed: %w", err)
		}
		sshConfig = &ssh.ClientConfig{
			User: "root",
			Auth: []ssh.AuthMethod{
				ssh.PublicKeys(signer),
			},
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
			Timeout:         30 * time.Second,
		}
	} else {
		// 密码认证
		sshConfig = &ssh.ClientConfig{
			User: "root",
			Auth: []ssh.AuthMethod{
				ssh.Password(authInfo.Password),
			},
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
			Timeout:         30 * time.Second,
		}
	}

	client, err := ssh.Dial("tcp", host+":22", sshConfig)
	if err != nil {
		return fmt.Errorf("SSH dial failed: %w", err)
	}
	defer client.Close()

	// GOST 安装脚本
	installScript := fmt.Sprintf(`#!/bin/bash
set -e

GOST_VERSION="3.0.0-rc10"
API_USER="%s"
API_PASS="%s"
API_PORT="%d"

echo ">>> Downloading GOST..."
cd /tmp
wget -q "https://github.com/go-gost/gost/releases/download/v${GOST_VERSION}/gost_${GOST_VERSION}_linux_amd64.tar.gz" -O gost.tar.gz || {
    echo ">>> wget failed, trying curl..."
    curl -sL "https://github.com/go-gost/gost/releases/download/v${GOST_VERSION}/gost_${GOST_VERSION}_linux_amd64.tar.gz" -o gost.tar.gz
}
tar -xzf gost.tar.gz && mv gost /usr/local/bin/ && chmod +x /usr/local/bin/gost
rm -f gost.tar.gz

echo ">>> Creating config..."
mkdir -p /etc/gost /var/log/gost
cat > /etc/gost/config.yaml << EOF
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

echo ">>> Creating systemd service..."
cat > /etc/systemd/system/gost.service << EOF
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

echo ">>> Starting GOST service..."
systemctl daemon-reload
systemctl enable gost --now

echo ">>> Optimizing network..."
cat >> /etc/sysctl.conf << 'SYSCTL'
net.core.somaxconn=65535
net.ipv4.tcp_max_syn_backlog=65535
net.ipv4.ip_local_port_range=1024 65535
net.ipv4.tcp_tw_reuse=1
fs.file-max=1048576
SYSCTL
sysctl -p > /dev/null 2>&1 || true

echo ">>> GOST installation completed!"
`, GostAPIUser, GostAPIPass, GostAPIPort)

	// 执行安装脚本
	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("create SSH session failed: %w", err)
	}
	defer session.Close()

	logx.Infof("executing GOST installation script on %s...", host)
	output, err := session.CombinedOutput(installScript)
	if err != nil {
		logx.Errorf("GOST installation output: %s", string(output))
		return fmt.Errorf("execute install script failed: %w", err)
	}
	logx.Infof("GOST installation output: %s", string(output))

	return nil
}

// waitForSSHReady 等待 SSH 服务就绪
func waitForSSHReady(host string, maxRetries int) error {
	for i := 0; i < maxRetries; i++ {
		conn, err := net.DialTimeout("tcp", host+":22", 5*time.Second)
		if err == nil {
			conn.Close()
			// 再等待几秒确保 SSH 服务完全就绪
			time.Sleep(5 * time.Second)
			return nil
		}
		logx.Infof("waiting for SSH... attempt %d/%d", i+1, maxRetries)
		time.Sleep(10 * time.Second)
	}
	return fmt.Errorf("SSH connection timeout after %d attempts", maxRetries)
}

// BandwidthInfo 实例带宽信息
type BandwidthInfo struct {
	InternetMaxBandwidthIn  int32  `json:"internet_max_bandwidth_in"`
	InternetMaxBandwidthOut int32  `json:"internet_max_bandwidth_out"`
	InternetChargeType      string `json:"internet_charge_type"`
	// EIP 相关 (实例绑定了 EIP 时填充)
	EipAllocationId string `json:"eip_allocation_id,omitempty"`
	EipBandwidth    int32  `json:"eip_bandwidth,omitempty"`
	EipChargeType   string `json:"eip_charge_type,omitempty"`
	HasEip          bool   `json:"has_eip"`
}

// GetInstanceBandwidth 查询实例的公网带宽信息
func GetInstanceBandwidth(cloudAccountId int64, regionId, instanceId string) (*BandwidthInfo, error) {
	cloud, err := GetSystemCloudAccount(cloudAccountId)
	if err != nil {
		return nil, err
	}
	client, err := NewEcsClient(cloud.AccessKey, cloud.AccessSecret, regionId)
	if err != nil {
		return nil, err
	}

	request := &ecs20140526.DescribeInstancesRequest{
		RegionId:    tea.String(regionId),
		InstanceIds: tea.String(fmt.Sprintf("[\"%s\"]", instanceId)),
	}
	response, err := client.DescribeInstances(request)
	if err != nil {
		return nil, fmt.Errorf("查询实例失败: %v", err)
	}
	if response.Body.Instances == nil || len(response.Body.Instances.Instance) == 0 {
		return nil, fmt.Errorf("实例不存在: %s", instanceId)
	}

	inst := response.Body.Instances.Instance[0]
	info := &BandwidthInfo{}
	if inst.InternetMaxBandwidthIn != nil {
		info.InternetMaxBandwidthIn = *inst.InternetMaxBandwidthIn
	}
	if inst.InternetMaxBandwidthOut != nil {
		info.InternetMaxBandwidthOut = *inst.InternetMaxBandwidthOut
	}
	if inst.InternetChargeType != nil {
		info.InternetChargeType = *inst.InternetChargeType
	}
	// 检测 EIP
	if inst.EipAddress != nil && inst.EipAddress.AllocationId != nil && *inst.EipAddress.AllocationId != "" {
		info.HasEip = true
		info.EipAllocationId = *inst.EipAddress.AllocationId
		if inst.EipAddress.Bandwidth != nil {
			info.EipBandwidth = int32(*inst.EipAddress.Bandwidth)
		}
		if inst.EipAddress.InternetChargeType != nil {
			info.EipChargeType = *inst.EipAddress.InternetChargeType
		}
		// EIP 场景下，实际带宽以 EIP 为准
		if info.EipBandwidth > 0 {
			info.InternetMaxBandwidthOut = info.EipBandwidth
		}
	}
	return info, nil
}

// ModifyInstanceNetworkSpecRequest 修改实例公网带宽请求
type ModifyInstanceNetworkSpecRequest struct {
	CloudAccountId          int64  `json:"cloud_account_id"`
	RegionId                string `json:"region_id"`
	InstanceId              string `json:"instance_id"`
	InternetMaxBandwidthOut int32  `json:"internet_max_bandwidth_out"`
}

// ModifyInstanceNetworkSpec 修改实例公网带宽
func ModifyInstanceNetworkSpec(req *ModifyInstanceNetworkSpecRequest) error {
	cloud, err := GetSystemCloudAccount(req.CloudAccountId)
	if err != nil {
		return err
	}
	client, err := NewEcsClient(cloud.AccessKey, cloud.AccessSecret, req.RegionId)
	if err != nil {
		return err
	}

	request := &ecs20140526.ModifyInstanceNetworkSpecRequest{
		InstanceId:              tea.String(req.InstanceId),
		InternetMaxBandwidthOut: tea.Int32(req.InternetMaxBandwidthOut),
	}
	response, err := client.ModifyInstanceNetworkSpec(request)
	if err != nil {
		return fmt.Errorf("修改实例带宽失败: %v", err)
	}
	if *response.StatusCode != 200 {
		return errors.New(response.String())
	}

	logx.Infof("修改实例带宽成功: InstanceId=%s, BandwidthOut=%d Mbps", req.InstanceId, req.InternetMaxBandwidthOut)
	return nil
}

// ModifyAutoRenewAttribute 修改实例自动续费属性
func ModifyAutoRenewAttribute(merchantId int, cloudAccountId int64, regionId, instanceId string, autoRenew bool, duration int32) error {
	var cloud *CloudAccountInfo
	var credErr error
	if cloudAccountId > 0 {
		cloud, credErr = GetSystemCloudAccount(cloudAccountId)
	} else {
		cloud, credErr = GetMerchantCloud(merchantId)
	}
	if credErr != nil {
		return credErr
	}
	client, err := NewEcsClient(cloud.AccessKey, cloud.AccessSecret, regionId)
	if err != nil {
		return fmt.Errorf("创建ECS客户端失败: %v", err)
	}

	request := &ecs20140526.ModifyInstanceAutoRenewAttributeRequest{
		RegionId:   tea.String(regionId),
		InstanceId: tea.String(instanceId),
	}

	if autoRenew {
		request.RenewalStatus = tea.String("AutoRenewal")
		if duration > 0 {
			request.Duration = tea.Int32(duration)
		} else {
			request.Duration = tea.Int32(1)
		}
		request.PeriodUnit = tea.String("Month")
	} else {
		request.RenewalStatus = tea.String("Normal")
	}

	runtime := &util.RuntimeOptions{}
	_, err = client.ModifyInstanceAutoRenewAttributeWithOptions(request, runtime)
	if err != nil {
		return fmt.Errorf("修改自动续费失败: %v", err)
	}
	return nil
}
