package cloud_aliyun

import (
	"fmt"
	"net/http"
	"server/internal/server/cloud/aliyun"
	"server/internal/server/middleware"
	"server/internal/server/model"
	"server/internal/server/service/cloud_aliyun"
	"server/pkg/result"
	"strconv"

	"strings"
	"sync"
	"time"

	ecs20140526 "github.com/alibabacloud-go/ecs-20140526/v6/client"
	vpc20160428 "github.com/alibabacloud-go/vpc-20160428/v6/client"
	"github.com/gin-gonic/gin"
	"github.com/zeromicro/go-zero/core/logx"
)

// 阿里云服务API路由
func Routes(ge gin.IRouter) {
	cloudGroup := ge.Group("/cloud", middleware.Authorization)

	// ECS实例管理
	cloudGroup.POST("/ecs/instance", createEcsInstance) // 创建ECS实例 流式api
	cloudGroup.GET("/ecs/instance", listEcsInstance)
	cloudGroup.POST("/ecs/instance/operate", operateEcsInstance)
	cloudGroup.POST("/ecs/instance/modify", modifyInstanceAttribute)
	cloudGroup.POST("/ecs/instance/modify-charge", modifyInstanceChargeType)               // 修改实例付费类型
	cloudGroup.POST("/ecs/instance/create-secondary-nic", createSecondaryNetworkInterface) // 创建辅助网卡 流式api
	cloudGroup.POST("/ecs/instance/register-with-ssh-key", registerInstanceWithSSHKey)     // 注册实例并自动创建SSH密钥
	cloudGroup.POST("/ecs/instance/bind-merchant", bindInstanceMerchant)                   // 绑定商户
	cloudGroup.POST("/ecs/instance/unbind-merchant", unbindInstanceMerchant)               // 解绑商户
	cloudGroup.POST("/ecs/instance/bindings", getInstanceBindings)                         // 批量查询绑定

	// 镜像管理
	cloudGroup.GET("/ecs/image", listImage)
	cloudGroup.POST("/ecs/image", createImage)                            // 创建镜像
	cloudGroup.GET("/ecs/image/share", describeImageSharePermission)      // 查询镜像共享权限
	cloudGroup.POST("/ecs/image/share", modifyImageSharePermission)       // 修改镜像共享权限

	// 安全组管理
	cloudGroup.POST("/ecs/security-group", createSecurityGroup) // 创建安全组 流式api
	cloudGroup.GET("/ecs/security-group", listSecurityGroup)
	cloudGroup.GET("/ecs/security-group/attribute", describeSecurityGroupAttribute)
	cloudGroup.DELETE("/ecs/security-group", deleteSecurityGroup)
	cloudGroup.POST("/ecs/security-group/authorize", authorizeSecurityGroup)
	cloudGroup.POST("/ecs/security-group/revoke", revokeSecurityGroup)
	cloudGroup.POST("/ecs/security-group/authorize/batch", authorizeSecurityBatch) // 批量授权安全组

	// 弹性网卡管理
	cloudGroup.GET("/ecs/network-interface", listNetworkInterface)
	cloudGroup.POST("/ecs/network-interface", createNetworkInterface)
	cloudGroup.DELETE("/ecs/network-interface", deleteNetworkInterface)
	cloudGroup.POST("/ecs/network-interface/attach", attachNetworkInterface)
	cloudGroup.POST("/ecs/network-interface/detach", detachNetworkInterface)
	cloudGroup.POST("/ecs/network-interface/modify", modifyNetworkInterface)

	// 弹性IP管理
	cloudGroup.POST("/vpc/eip", allocateEipAddress) // 申请弹性IP 流式api
	cloudGroup.GET("/vpc/eip", listEip)
	cloudGroup.POST("/vpc/eip/operate", operateEip)
	cloudGroup.POST("/vpc/eip/batch-associate", batchAssociateEip) // 批量绑定弹性IP 流式api
	cloudGroup.POST("/vpc/eip/replace", replaceEip)                // 更换弹性IP 流式api
	cloudGroup.POST("/vpc/eip/batch-replace", batchReplaceEip)     // 批量更换弹性IP 流式api

	// 共享带宽管理
	cloudGroup.GET("/vpc/bandwidth", listBandwidthPackage)
	cloudGroup.POST("/vpc/bandwidth", createBandwidthPackage)
	cloudGroup.POST("/vpc/bandwidth/operate", operateBandwidthPackage)

	// OSS 对象存储
	cloudGroup.GET("/oss/objects", listOssObjects)
	cloudGroup.GET("/oss/buckets", listOssBuckets)
	cloudGroup.POST("/oss/object", uploadOssObject)
	cloudGroup.GET("/oss/object", downloadOssObject)
	cloudGroup.DELETE("/oss/object", deleteOssObject)
	cloudGroup.POST("/oss/bucket", createOssBucket)
	cloudGroup.DELETE("/oss/bucket", deleteOssBucket)
	cloudGroup.POST("/oss/bucket/set-public", setOssBucketPublic)

	// 账户余额
	cloudGroup.GET("/account/balance", getAliyunAccountBalance)
}

// ECS实例相关接口
func createEcsInstance(c *gin.Context) {

	var req model.CreateInstancesReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.GStreamEnd(c, true, err.Error())
		return
	}
	// 流式响应
	result.GStream(c)
	err := cloud_aliyun.CreateEcsInstance(req, func(message string) {
		result.GStreamData(c, gin.H{
			"message": fmt.Sprintf("%s %s", time.Now().Format(time.DateTime), message),
		})
	})
	if err != nil {
		result.GStreamEnd(c, true, err.Error())
	} else {
		result.GStreamEnd(c, true, "创建所有ECS实例成功")
	}
}

// 为指定实例创建并绑定辅助网卡
func createSecondaryNetworkInterface(c *gin.Context) {
	var req model.CreateSecondaryNetworkInterfaceReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.GStreamEnd(c, true, err.Error())
		return
	}

	// 检查请求中是否有实例
	if len(req.Instances) == 0 {
		result.GStreamEnd(c, true, "未指定任何实例")
		return
	}

	// 流式响应
	result.GStream(c)

	// 调用服务逻辑创建并绑定辅助网卡
	err := cloud_aliyun.CreateSecondaryNetworkInterface(req.CloudAccountId, req.MerchantId, req.Instances, func(message string) {
		result.GStreamData(c, gin.H{
			"message": fmt.Sprintf("%s %s", time.Now().Format(time.DateTime), message),
		})
	})

	if err != nil {
		result.GStreamEnd(c, true, err.Error())
	} else {
		result.GStreamEnd(c, true, "辅助网卡创建和绑定任务完成")
	}
}

func listEcsInstance(c *gin.Context) {
	var req model.ListReq
	if err := c.ShouldBindQuery(&req); err != nil {
		logx.Errorf("参数绑定失败: %v", err)
		result.GErr(c, err)
		return
	}

	// 验证必须有 MerchantId 或 CloudAccountId 之一
	if req.MerchantId == 0 && req.CloudAccountId == 0 {
		result.GErr(c, fmt.Errorf("merchant_id 或 cloud_account_id 必须提供一个"))
		return
	}

	logx.Infof("listEcsInstance请求参数: MerchantId=%d, CloudAccountId=%d, RegionId=%v", req.MerchantId, req.CloudAccountId, req.RegionId)

	// 确保RegionId不为空
	if len(req.RegionId) == 0 {
		result.GErr(c, fmt.Errorf("region_id不能为空"))
		return
	}

	// 使用 goroutine 并发请求多个地区的实例
	var wg sync.WaitGroup
	type regionResult struct {
		Region    string
		Instances []*ecs20140526.DescribeInstancesResponseBodyInstancesInstance
		NicEipMap map[string][]map[string]interface{} // 网卡ID到公网IP的映射
		Err       error
	}
	resultChan := make(chan regionResult, len(req.RegionId))

	for _, regionId := range req.RegionId {
		wg.Add(1)
		go func(region string) {
			defer wg.Done()
			logx.Infof("开始查询区域 %s 的实例列表", region)
			var instances []*ecs20140526.DescribeInstancesResponseBodyInstancesInstance
			var nicEipMap map[string][]map[string]interface{}
			var err error
			if req.CloudAccountId > 0 {
				instances, err = cloud_aliyun.GetEcsInstanceListByCloudAccount(req.CloudAccountId, region)
			} else {
				instances, err = cloud_aliyun.GetEcsInstanceList(req.CloudAccountId, req.MerchantId, region)
			}
			if err != nil {
				logx.Errorf("查询区域 %s 失败: %v", region, err)
			} else {
				logx.Infof("查询区域 %s 成功, 实例数量: %d", region, len(instances))
				// 查询网卡EIP映射
				nicEipMap, err = cloud_aliyun.GetNetworkInterfaceEipMap(req.CloudAccountId, req.MerchantId, region)
				if err != nil {
					logx.Errorf("查询区域 %s 网卡EIP映射失败: %v", region, err)
					// 不影响主流程，继续执行
					nicEipMap = make(map[string][]map[string]interface{})
				}
			}
			resultChan <- regionResult{
				Region:    region,
				Instances: instances,
				NicEipMap: nicEipMap,
				Err:       err,
			}
		}(regionId)
	}

	// 等待所有请求完成
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// 合并结果
	var allInstances []*ecs20140526.DescribeInstancesResponseBodyInstancesInstance
	allNicEipMap := make(map[string][]map[string]interface{})
	var errors []string

	for res := range resultChan {
		if res.Err != nil {
			errors = append(errors, fmt.Sprintf("region %s: %s", res.Region, res.Err.Error()))
			continue
		}
		allInstances = append(allInstances, res.Instances...)
		// 合并网卡EIP映射
		for nicId, eipList := range res.NicEipMap {
			allNicEipMap[nicId] = eipList
		}
	}

	if len(errors) > 0 {
		result.GErr(c, fmt.Errorf("部分区域请求失败: %s", strings.Join(errors, "; ")))
		return
	}

	// 查询实例的商户绑定信息
	instanceIds := make([]string, 0, len(allInstances))
	for _, inst := range allInstances {
		if inst.InstanceId != nil {
			instanceIds = append(instanceIds, *inst.InstanceId)
		}
	}
	bindings, _ := cloud_aliyun.GetInstanceBindings(instanceIds, "aliyun")

	result.GOK(c, gin.H{
		"list":        allInstances,
		"total":       len(allInstances),
		"nic_eip_map": allNicEipMap,
		"bindings":    bindings,
	})
}

func bindInstanceMerchant(c *gin.Context) {
	var req model.BindInstanceMerchantReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.GErr(c, err)
		return
	}
	if err := cloud_aliyun.BindInstanceMerchant(req); err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, nil)
}

func unbindInstanceMerchant(c *gin.Context) {
	var req model.UnbindInstanceMerchantReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.GErr(c, err)
		return
	}
	if err := cloud_aliyun.UnbindInstanceMerchant(req.InstanceId, req.CloudType); err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, nil)
}

func getInstanceBindings(c *gin.Context) {
	var req model.GetInstanceBindingsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.GErr(c, err)
		return
	}
	bindings, err := cloud_aliyun.GetInstanceBindings(req.InstanceIds, req.CloudType)
	if err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, bindings)
}

func operateEcsInstance(c *gin.Context) {
	var req model.OperateEcsInstanceReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.GErr(c, err)
		return
	}

	// 验证账号参数
	if req.MerchantId == 0 && req.CloudAccountId == 0 {
		result.GErr(c, fmt.Errorf("merchant_id或cloud_account_id必须提供一个"))
		return
	}

	err := cloud_aliyun.OperateEcsInstance(req.MerchantId, req.CloudAccountId, req.RegionId, req.InstanceId, req.Operation)
	if err != nil {
		result.GErr(c, err)
		return
	}

	result.GOK(c, nil)
}

func modifyInstanceAttribute(c *gin.Context) {
	var req aliyun.ModifyInstanceAttributeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		result.GErr(c, err)
		return
	}
	err := cloud_aliyun.ModifyInstanceAttribute(&req)
	if err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, nil)
}

// 修改实例付费类型
func modifyInstanceChargeType(c *gin.Context) {
	var req struct {
		CloudAccountId int64  `json:"cloud_account_id"`
		MerchantId     int    `json:"merchant_id"`
		RegionId       string `json:"region_id"`
		InstanceId     string `json:"instance_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		result.GErr(c, err)
		return
	}
	err := aliyun.ModifyInstanceChargeType(req.CloudAccountId, req.MerchantId, req.RegionId, req.InstanceId)
	if err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, nil)
}

// registerInstanceWithSSHKey 注册实例到服务器管理（自动创建并绑定SSH密钥）
func registerInstanceWithSSHKey(c *gin.Context) {
	var req aliyun.RegisterInstanceWithSSHKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		result.GErr(c, err)
		return
	}

	res, err := aliyun.RegisterInstanceWithSSHKey(&req)
	if err != nil {
		result.GErr(c, err)
		return
	}

	result.GOK(c, res)
}

func listImage(c *gin.Context) {
	var req model.ListReq
	if err := c.ShouldBindQuery(&req); err != nil {
		result.GErr(c, err)
		return
	}
	// 确保RegionId不为空
	if len(req.RegionId) == 0 {
		result.GErr(c, fmt.Errorf("region_id不能为空"))
		return
	}
	// 使用 goroutine 并发请求多个地区的安全组
	var wg sync.WaitGroup
	type regionResult struct {
		Region string
		Images []*ecs20140526.DescribeImagesResponseBodyImagesImage
		Err    error
	}
	resultChan := make(chan regionResult, len(req.RegionId))

	for _, regionId := range req.RegionId {
		wg.Add(1)
		go func(region string) {
			defer wg.Done()
			images, err := cloud_aliyun.DescribeImages(req.CloudAccountId, req.MerchantId, region)
			resultChan <- regionResult{
				Region: region,
				Images: images,
				Err:    err,
			}
		}(regionId)
	}

	// 等待所有请求完成
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// 合并结果
	var allImages []*ecs20140526.DescribeImagesResponseBodyImagesImage
	var errors []string

	for res := range resultChan {
		if res.Err != nil {
			errors = append(errors, fmt.Sprintf("region %s: %s", res.Region, res.Err.Error()))
			continue
		}
		allImages = append(allImages, res.Images...)
	}

	if len(errors) > 0 {
		result.GErr(c, fmt.Errorf("部分区域请求失败: %s", strings.Join(errors, "; ")))
		return
	}

	result.GOK(c, gin.H{
		"list":  allImages,
		"total": len(allImages),
	})
}

// 安全组相关接口
func createSecurityGroup(c *gin.Context) {
	var req model.CreateSecurityGroupReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.GStreamEnd(c, true, err.Error())
		return
	}
	cloud_aliyun.CreateSecurityGroup(&req, func(message string) {
		result.GStreamData(c, gin.H{
			"message": fmt.Sprintf("%s %s", time.Now().Format(time.DateTime), message),
		})
	})
	result.GStreamEnd(c, true, "创建安全组执行完毕")
}

func listSecurityGroup(c *gin.Context) {
	var req model.ListReq
	if err := c.ShouldBindQuery(&req); err != nil {
		result.GErr(c, err)
		return
	}

	// 确保RegionId不为空
	if len(req.RegionId) == 0 {
		result.GErr(c, fmt.Errorf("region_id不能为空"))
		return
	}

	// 确保merchant_id或cloud_account_id至少提供一个
	if req.MerchantId == 0 && req.CloudAccountId == 0 {
		result.GErr(c, fmt.Errorf("merchant_id或cloud_account_id必须提供一个"))
		return
	}

	// 使用 goroutine 并发请求多个地区的安全组
	var wg sync.WaitGroup
	type regionResult struct {
		Region         string
		SecurityGroups []*ecs20140526.DescribeSecurityGroupsResponseBodySecurityGroupsSecurityGroup
		Err            error
	}
	resultChan := make(chan regionResult, len(req.RegionId))

	for _, regionId := range req.RegionId {
		wg.Add(1)
		go func(region string) {
			defer wg.Done()
			var securityGroups []*ecs20140526.DescribeSecurityGroupsResponseBodySecurityGroupsSecurityGroup
			var err error

			// 优先使用cloud_account_id
			if req.CloudAccountId > 0 {
				securityGroups, err = cloud_aliyun.GetSecurityGroupListByCloudAccount(req.CloudAccountId, region)
			} else {
				securityGroups, err = cloud_aliyun.GetSecurityGroupList(req.MerchantId, region)
			}

			resultChan <- regionResult{
				Region:         region,
				SecurityGroups: securityGroups,
				Err:            err,
			}
		}(regionId)
	}

	// 等待所有请求完成
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// 合并结果
	var allSecurityGroups []*model.SecurityGroupData
	var errors []string

	for res := range resultChan {
		if res.Err != nil {
			errors = append(errors, fmt.Sprintf("region %s: %s", res.Region, res.Err.Error()))
			continue
		}
		for _, sg := range res.SecurityGroups {
			allSecurityGroups = append(allSecurityGroups, &model.SecurityGroupData{
				RegionId:      res.Region,
				SecurityGroup: sg,
			})
		}
	}

	if len(errors) > 0 {
		result.GErr(c, fmt.Errorf("部分区域请求失败: %s", strings.Join(errors, "; ")))
		return
	}

	result.GOK(c, gin.H{
		"list":  allSecurityGroups,
		"total": len(allSecurityGroups),
	})
}

func describeSecurityGroupAttribute(c *gin.Context) {
	var req model.DescribeSecurityGroupAttributeReq
	if err := c.ShouldBindQuery(&req); err != nil {
		result.GErr(c, err)
		return
	}

	if req.MerchantId == 0 && req.CloudAccountId == 0 {
		result.GErr(c, fmt.Errorf("merchant_id或cloud_account_id必须提供一个"))
		return
	}

	sg, err := cloud_aliyun.DescribeSecurityGroupAttribute(req.MerchantId, req.CloudAccountId, req.RegionId, req.SecurityGroupId)
	if err != nil {
		result.GErr(c, err)
		return
	}

	result.GOK(c, sg)
}

func deleteSecurityGroup(c *gin.Context) {
	var req aliyun.DeleteSecurityGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		result.GErr(c, err)
		return
	}

	// 验证账号参数
	if req.MerchantId == 0 && req.CloudAccountId == 0 {
		result.GErr(c, fmt.Errorf("merchant_id或cloud_account_id必须提供一个"))
		return
	}

	err := cloud_aliyun.DeleteSecurityGroup(&req)
	if err != nil {
		result.GErr(c, err)
		return
	}

	result.GOK(c, nil)
}

func authorizeSecurityGroup(c *gin.Context) {
	var req aliyun.AuthorizeSecurityGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		result.GErr(c, err)
		return
	}

	// 验证账号参数
	if req.MerchantId == 0 && req.CloudAccountId == 0 {
		result.GErr(c, fmt.Errorf("merchant_id或cloud_account_id必须提供一个"))
		return
	}

	err := cloud_aliyun.AuthorizeSecurityGroup(&req)
	if err != nil {
		result.GErr(c, err)
		return
	}

	result.GOK(c, nil)
}

func authorizeSecurityBatch(c *gin.Context) {
	var req struct {
		MerchantId  int                                                     `json:"merchant_id"`
		RegionIds   []string                                                `json:"region_ids"`
		Permissions []*ecs20140526.AuthorizeSecurityGroupRequestPermissions `json:"permissions"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		result.GErr(c, err)
		return
	}
	if len(req.RegionIds) == 0 {
		result.GErr(c, fmt.Errorf("region_id不能为空"))
		return
	}
	logx.Infof("授权安全组请求: %d %v", req.MerchantId, req.RegionIds)
	var wg sync.WaitGroup
	for _, regionId := range req.RegionIds {
		wg.Add(1)
		go func(region string) {
			defer wg.Done()
			securityGroups, err := cloud_aliyun.GetSecurityGroupList(req.MerchantId, region)
			if err != nil {
				logx.Errorf("获取安全组列表失败: %d %s %v", req.MerchantId, region, err)
				return
			}
			for _, sg := range securityGroups {
				err = cloud_aliyun.AuthorizeSecurityGroup(&aliyun.AuthorizeSecurityGroupRequest{
					MerchantId:      req.MerchantId,
					RegionId:        region,
					SecurityGroupId: *sg.SecurityGroupId,
					Permissions:     req.Permissions,
				})
				if err != nil {
					logx.Errorf("授权安全组失败: %d %s %s %v", req.MerchantId, region, *sg.SecurityGroupId, err)
					continue
				}
				logx.Infof("授权安全组成功: %d %s %v", req.MerchantId, region, *sg.SecurityGroupId)
			}
		}(regionId)
	}
	wg.Wait()
	result.GOK(c, nil)
}
func revokeSecurityGroup(c *gin.Context) {
	var req aliyun.RevokeSecurityGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		result.GErr(c, err)
		return
	}

	// 验证账号参数
	if req.MerchantId == 0 && req.CloudAccountId == 0 {
		result.GErr(c, fmt.Errorf("merchant_id或cloud_account_id必须提供一个"))
		return
	}

	err := cloud_aliyun.RevokeSecurityGroup(&req)
	if err != nil {
		result.GErr(c, err)
		return
	}

	result.GOK(c, nil)
}

// 弹性网卡相关接口
func createNetworkInterface(c *gin.Context) {
	var req aliyun.CreateNetworkInterfaceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		result.GErr(c, err)
		return
	}

	// 验证账号参数
	if req.MerchantId == 0 && req.CloudAccountId == 0 {
		result.GErr(c, fmt.Errorf("merchant_id或cloud_account_id必须提供一个"))
		return
	}

	networkInterfaceId, err := cloud_aliyun.CreateNetworkInterface(&req)
	if err != nil {
		result.GErr(c, err)
		return
	}

	result.GOK(c, networkInterfaceId)
}

func listNetworkInterface(c *gin.Context) {
	var req model.ListReq
	if err := c.ShouldBindQuery(&req); err != nil {
		result.GErr(c, err)
		return
	}

	// 确保RegionId不为空
	if len(req.RegionId) == 0 {
		result.GErr(c, fmt.Errorf("region_id不能为空"))
		return
	}

	// 确保merchant_id或cloud_account_id至少提供一个
	if req.MerchantId == 0 && req.CloudAccountId == 0 {
		result.GErr(c, fmt.Errorf("merchant_id或cloud_account_id必须提供一个"))
		return
	}

	// 使用 goroutine 并发请求多个地区的网络接口
	var wg sync.WaitGroup
	type regionResult struct {
		Region     string
		Interfaces []*ecs20140526.DescribeNetworkInterfacesResponseBodyNetworkInterfaceSetsNetworkInterfaceSet
		Err        error
	}
	resultChan := make(chan regionResult, len(req.RegionId))

	for _, regionId := range req.RegionId {
		wg.Add(1)
		go func(region string) {
			defer wg.Done()
			var interfaces []*ecs20140526.DescribeNetworkInterfacesResponseBodyNetworkInterfaceSetsNetworkInterfaceSet
			var err error
			interfaces, err = cloud_aliyun.GetNetworkInterfaceList(req.CloudAccountId, req.MerchantId, region)
			resultChan <- regionResult{
				Region:     region,
				Interfaces: interfaces,
				Err:        err,
			}
		}(regionId)
	}

	// 等待所有请求完成
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// 合并结果
	var allInterfaces []*model.NetworkInterfaceData
	var errors []string

	for res := range resultChan {
		if res.Err != nil {
			errors = append(errors, fmt.Sprintf("region %s: %s", res.Region, res.Err.Error()))
			continue
		}
		for _, iface := range res.Interfaces {
			allInterfaces = append(allInterfaces, &model.NetworkInterfaceData{
				RegionId:         res.Region,
				NetworkInterface: iface,
			})
		}
	}

	if len(errors) > 0 {
		result.GErr(c, fmt.Errorf("部分区域请求失败: %s", strings.Join(errors, "; ")))
		return
	}

	result.GOK(c, gin.H{
		"list":  allInterfaces,
		"total": len(allInterfaces),
	})
}

func deleteNetworkInterface(c *gin.Context) {
	var req aliyun.DeleteNetworkInterfaceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		result.GErr(c, err)
		return
	}

	// 验证账号参数
	if req.MerchantId == 0 && req.CloudAccountId == 0 {
		result.GErr(c, fmt.Errorf("merchant_id或cloud_account_id必须提供一个"))
		return
	}

	err := cloud_aliyun.DeleteNetworkInterface(&req)
	if err != nil {
		result.GErr(c, err)
		return
	}

	result.GOK(c, nil)
}

func attachNetworkInterface(c *gin.Context) {
	var req aliyun.AttachNetworkInterfaceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		result.GErr(c, err)
		return
	}

	// 验证账号参数
	if req.MerchantId == 0 && req.CloudAccountId == 0 {
		result.GErr(c, fmt.Errorf("merchant_id或cloud_account_id必须提供一个"))
		return
	}

	err := cloud_aliyun.AttachNetworkInterface(&req)
	if err != nil {
		result.GErr(c, err)
		return
	}

	result.GOK(c, nil)
}

func detachNetworkInterface(c *gin.Context) {
	var req aliyun.DetachNetworkInterfaceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		result.GErr(c, err)
		return
	}

	// 验证账号参数
	if req.MerchantId == 0 && req.CloudAccountId == 0 {
		result.GErr(c, fmt.Errorf("merchant_id或cloud_account_id必须提供一个"))
		return
	}

	err := cloud_aliyun.DetachNetworkInterface(&req)
	if err != nil {
		result.GErr(c, err)
		return
	}

	result.GOK(c, nil)
}

func modifyNetworkInterface(c *gin.Context) {
	var req aliyun.ModifyNetworkInterfaceAttributeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		result.GErr(c, err)
		return
	}

	// 验证账号参数
	if req.MerchantId == 0 && req.CloudAccountId == 0 {
		result.GErr(c, fmt.Errorf("merchant_id或cloud_account_id必须提供一个"))
		return
	}

	err := cloud_aliyun.ModifyNetworkInterfaceAttribute(&req)
	if err != nil {
		result.GErr(c, err)
		return
	}

	result.GOK(c, nil)
}

// 弹性IP相关接口
func allocateEipAddress(c *gin.Context) {
	var req model.CreateEipReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.GStreamEnd(c, true, err.Error())
		return
	}
	cloud_aliyun.AllocateEipAddress(&req, func(message string) {
		result.GStreamData(c, gin.H{
			"message": fmt.Sprintf("%s %s", time.Now().Format(time.DateTime), message),
		})
	})
	result.GStreamEnd(c, true, "申请弹性IP执行完毕")
}

func listEip(c *gin.Context) {
	var req model.ListReq
	if err := c.ShouldBindQuery(&req); err != nil {
		result.GErr(c, err)
		return
	}

	// 确保RegionId不为空
	if len(req.RegionId) == 0 {
		result.GErr(c, fmt.Errorf("region_id不能为空"))
		return
	}

	// 确保merchant_id或cloud_account_id至少提供一个
	if req.MerchantId == 0 && req.CloudAccountId == 0 {
		result.GErr(c, fmt.Errorf("merchant_id或cloud_account_id必须提供一个"))
		return
	}

	// 使用 goroutine 并发请求多个地区的EIP
	var wg sync.WaitGroup
	type regionResult struct {
		Region string
		Eips   []*vpc20160428.DescribeEipAddressesResponseBodyEipAddressesEipAddress
		Err    error
	}
	resultChan := make(chan regionResult, len(req.RegionId))

	for _, regionId := range req.RegionId {
		wg.Add(1)
		go func(region string) {
			defer wg.Done()
			var eips []*vpc20160428.DescribeEipAddressesResponseBodyEipAddressesEipAddress
			var err error

			// 优先使用cloud_account_id
			if req.CloudAccountId > 0 {
				eips, err = cloud_aliyun.GetEipListByCloudAccount(req.CloudAccountId, region)
			} else {
				eips, err = cloud_aliyun.GetEipList(req.MerchantId, region)
			}

			resultChan <- regionResult{
				Region: region,
				Eips:   eips,
				Err:    err,
			}
		}(regionId)
	}

	// 等待所有请求完成
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// 合并结果
	var allEips []*vpc20160428.DescribeEipAddressesResponseBodyEipAddressesEipAddress
	var errors []string

	for res := range resultChan {
		if res.Err != nil {
			errors = append(errors, fmt.Sprintf("region %s: %s", res.Region, res.Err.Error()))
			continue
		}
		allEips = append(allEips, res.Eips...)
	}

	if len(errors) > 0 {
		result.GErr(c, fmt.Errorf("部分区域请求失败: %s", strings.Join(errors, "; ")))
		return
	}

	result.GOK(c, gin.H{
		"list":  allEips,
		"total": len(allEips),
	})
}

func operateEip(c *gin.Context) {
	var req model.OperateEipReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.GErr(c, err)
		return
	}

	// 验证账号参数
	if req.MerchantId == 0 && req.CloudAccountId == 0 {
		result.GErr(c, fmt.Errorf("merchant_id或cloud_account_id必须提供一个"))
		return
	}

	var err error
	switch req.Operation {
	case "modify":
		modifyReq := aliyun.ModifyEipAddressAttributeRequest{
			MerchantId:     req.MerchantId,
			CloudAccountId: req.CloudAccountId,
			Region:         req.RegionId,
			AllocationId:   req.AllocationId,
			Name:           req.Name,
			Bandwidth:      req.Bandwidth,
		}
		err = cloud_aliyun.ModifyEipAddressAttribute(&modifyReq)
	case "associate":
		associateReq := aliyun.AssociateEipAddressRequest{
			MerchantId:     req.MerchantId,
			CloudAccountId: req.CloudAccountId,
			Region:         req.RegionId,
			AllocationId:   req.AllocationId,
			InstanceId:     req.InstanceId,
			InstanceType:   req.InstanceType,
		}
		err = cloud_aliyun.AssociateEipAddress(&associateReq)
	case "unassociate":
		unassociateReq := aliyun.UnassociateEipAddressRequest{
			MerchantId:     req.MerchantId,
			CloudAccountId: req.CloudAccountId,
			Region:         req.RegionId,
			AllocationId:   req.AllocationId,
			InstanceId:     req.InstanceId,
			InstanceType:   req.InstanceType,
		}
		err = cloud_aliyun.UnassociateEipAddress(&unassociateReq)
	case "delete":
		releaseReq := aliyun.ReleaseEipAddressRequest{
			MerchantId:     req.MerchantId,
			CloudAccountId: req.CloudAccountId,
			Region:         req.RegionId,
			AllocationId:   req.AllocationId,
		}
		err = cloud_aliyun.ReleaseEipAddress(&releaseReq)
	}

	if err != nil {
		result.GErr(c, err)
		return
	}

	result.GOK(c, nil)
}

// 批量绑定弹性IP到实例或辅助网卡
func batchAssociateEip(c *gin.Context) {
	var req model.BatchAssociateEipReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.GStreamEnd(c, true, err.Error())
		return
	}

	// 验证账号参数
	if req.MerchantId == 0 && req.CloudAccountId == 0 {
		result.GStreamEnd(c, true, "merchant_id或cloud_account_id必须提供一个")
		return
	}

	// 检查请求中是否有弹性IP
	if len(req.EipList) == 0 {
		result.GStreamEnd(c, true, "未指定任何弹性IP")
		return
	}

	// 流式响应
	result.GStream(c)

	// 调用服务逻辑绑定弹性IP
	err := cloud_aliyun.BatchAssociateEip(req.MerchantId, req.CloudAccountId, req.EipList, func(message string) {
		result.GStreamData(c, gin.H{
			"message": fmt.Sprintf("%s %s", time.Now().Format(time.DateTime), message),
		})
	})

	if err != nil {
		result.GStreamEnd(c, true, err.Error())
	} else {
		result.GStreamEnd(c, true, "弹性IP绑定任务完成")
	}
}

// 更换弹性IP（流式API）
func replaceEip(c *gin.Context) {
	var req model.ReplaceEipReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.GStreamEnd(c, true, err.Error())
		return
	}

	// 验证账号参数
	if req.MerchantId == 0 && req.CloudAccountId == 0 {
		result.GStreamEnd(c, true, "merchant_id或cloud_account_id必须提供一个")
		return
	}

	// 流式响应
	result.GStream(c)

	// 调用服务逻辑更换弹性IP
	err := cloud_aliyun.ReplaceEipAddress(&req, func(message string) {
		result.GStreamData(c, gin.H{
			"message": fmt.Sprintf("%s %s", time.Now().Format(time.DateTime), message),
		})
	})

	if err != nil {
		result.GStreamEnd(c, true, err.Error())
	} else {
		result.GStreamEnd(c, true, "弹性IP更换完成")
	}
}

// 批量更换弹性IP
func batchReplaceEip(c *gin.Context) {
	var req model.BatchReplaceEipReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.GStreamEnd(c, true, err.Error())
		return
	}

	// 验证账号参数
	if req.MerchantId == 0 && req.CloudAccountId == 0 {
		result.GStreamEnd(c, true, "merchant_id或cloud_account_id必须提供一个")
		return
	}

	// 验证EIP列表
	if len(req.EipList) == 0 {
		result.GStreamEnd(c, true, "eip_list不能为空")
		return
	}

	// 流式响应
	result.GStream(c)

	// 调用服务逻辑批量更换弹性IP
	err := cloud_aliyun.BatchReplaceEipAddress(&req, func(message string) {
		result.GStreamData(c, gin.H{
			"message": fmt.Sprintf("%s %s", time.Now().Format(time.DateTime), message),
		})
	})

	if err != nil {
		result.GStreamEnd(c, true, err.Error())
	} else {
		result.GStreamEnd(c, true, "批量更换弹性IP完成")
	}
}

// 共享带宽相关接口
func createBandwidthPackage(c *gin.Context) {
	var req aliyun.CreateCommonBandwidthPackageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		result.GErr(c, err)
		return
	}

	bandwidthPackageId, err := cloud_aliyun.CreateBandwidthPackage(&req)
	if err != nil {
		result.GErr(c, err)
		return
	}

	result.GOK(c, bandwidthPackageId)
}

func listBandwidthPackage(c *gin.Context) {
	var req model.ListReq
	if err := c.ShouldBindQuery(&req); err != nil {
		result.GErr(c, err)
		return
	}

	// 确保RegionId不为空
	if len(req.RegionId) == 0 {
		result.GErr(c, fmt.Errorf("region_id不能为空"))
		return
	}

	// 确保merchant_id或cloud_account_id至少提供一个
	if req.MerchantId == 0 && req.CloudAccountId == 0 {
		result.GErr(c, fmt.Errorf("merchant_id或cloud_account_id必须提供一个"))
		return
	}

	// 使用 goroutine 并发请求多个地区的带宽包
	var wg sync.WaitGroup
	type regionResult struct {
		Region            string
		BandwidthPackages []*vpc20160428.DescribeCommonBandwidthPackagesResponseBodyCommonBandwidthPackagesCommonBandwidthPackage
		Err               error
	}
	resultChan := make(chan regionResult, len(req.RegionId))

	for _, regionId := range req.RegionId {
		wg.Add(1)
		go func(region string) {
			defer wg.Done()
			var packages []*vpc20160428.DescribeCommonBandwidthPackagesResponseBodyCommonBandwidthPackagesCommonBandwidthPackage
			var err error

			// 优先使用cloud_account_id
			if req.CloudAccountId > 0 {
				packages, err = cloud_aliyun.GetBandwidthPackageListByCloudAccount(req.CloudAccountId, region)
			} else {
				packages, err = cloud_aliyun.GetBandwidthPackageList(req.MerchantId, region)
			}

			resultChan <- regionResult{
				Region:            region,
				BandwidthPackages: packages,
				Err:               err,
			}
		}(regionId)
	}

	// 等待所有请求完成
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// 合并结果
	var allPackages []*vpc20160428.DescribeCommonBandwidthPackagesResponseBodyCommonBandwidthPackagesCommonBandwidthPackage
	var errors []string

	for res := range resultChan {
		if res.Err != nil {
			errors = append(errors, fmt.Sprintf("region %s: %s", res.Region, res.Err.Error()))
			continue
		}
		allPackages = append(allPackages, res.BandwidthPackages...)
	}

	if len(errors) > 0 {
		result.GErr(c, fmt.Errorf("部分区域请求失败: %s", strings.Join(errors, "; ")))
		return
	}

	result.GOK(c, gin.H{
		"list":  allPackages,
		"total": len(allPackages),
	})
}

func operateBandwidthPackage(c *gin.Context) {
	var req model.OperateBandwidthPackageReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.GErr(c, err)
		return
	}

	// 验证账号参数
	if req.MerchantId == 0 && req.CloudAccountId == 0 {
		result.GErr(c, fmt.Errorf("merchant_id或cloud_account_id必须提供一个"))
		return
	}

	var err error
	switch req.Operation {
	case "modify":
		modifyReq := aliyun.ModifyCommonBandwidthPackageAttributeRequest{
			MerchantId:         req.MerchantId,
			CloudAccountId:     req.CloudAccountId,
			Region:             req.RegionId,
			BandwidthPackageId: req.BandwidthPackageId,
			Name:               req.Name,
			Description:        req.Description,
		}
		err = cloud_aliyun.ModifyBandwidthPackageAttribute(&modifyReq)
	case "spec":
		specReq := aliyun.ModifyCommonBandwidthPackageSpecRequest{
			MerchantId:         req.MerchantId,
			CloudAccountId:     req.CloudAccountId,
			Region:             req.RegionId,
			BandwidthPackageId: req.BandwidthPackageId,
			Bandwidth:          req.Bandwidth,
		}
		err = cloud_aliyun.ModifyBandwidthPackageSpec(&specReq)
	case "addEip":
		logx.Infof("bandwidth addEip req: %+v", req)
		addEipReq := aliyun.AddCommonBandwidthPackageIpsRequest{
			MerchantId:         req.MerchantId,
			CloudAccountId:     req.CloudAccountId,
			Region:             req.RegionId,
			BandwidthPackageId: req.BandwidthPackageId,
			IpInstanceIds:      req.IpInstanceIds,
		}
		err = cloud_aliyun.AddEipToBandwidthPackage(&addEipReq)
	case "removeEip":
		logx.Infof("bandwidth removeEip req: %+v", req)
		removeEipReq := aliyun.RemoveCommonBandwidthPackageIpRequest{
			MerchantId:         req.MerchantId,
			CloudAccountId:     req.CloudAccountId,
			Region:             req.RegionId,
			BandwidthPackageId: req.BandwidthPackageId,
			IpInstanceId:       req.IpInstanceId,
		}
		err = cloud_aliyun.RemoveEipFromBandwidthPackage(&removeEipReq)
	case "delete":
		deleteReq := aliyun.DeleteCommonBandwidthPackageRequest{
			MerchantId:         req.MerchantId,
			CloudAccountId:     req.CloudAccountId,
			Region:             req.RegionId,
			BandwidthPackageId: req.BandwidthPackageId,
		}
		err = cloud_aliyun.DeleteBandwidthPackage(&deleteReq)
	}

	if err != nil {
		result.GErr(c, err)
		return
	}

	result.GOK(c, nil)
}

// ========== OSS Handlers ==========
// 列表
func listOssObjects(c *gin.Context) {
	var req model.OssListObjectsReq
	if err := c.ShouldBindQuery(&req); err != nil {
		result.GErr(c, err)
		return
	}
	if req.MerchantId == 0 && req.CloudAccountId == 0 {
		result.GErr(c, fmt.Errorf("merchant_id或cloud_account_id必须提供一个"))
		return
	}
	data, err := cloud_aliyun.ListOssObjects(req)
	if err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, data)
}

// 上传（multipart/form-data，字段：file, merchant_id, cloud_account_id, region_id, bucket, object_key）
func uploadOssObject(c *gin.Context) {
	var form model.OssUploadForm
	// 附带非文件字段
	form.MerchantId, _ = strconv.Atoi(c.PostForm("merchant_id"))
	if v := c.PostForm("cloud_account_id"); v != "" {
		if id, err := strconv.ParseInt(v, 10, 64); err == nil {
			form.CloudAccountId = id
		}
	}
	form.RegionId = c.PostForm("region_id")
	form.Bucket = c.PostForm("bucket")
	form.ObjectKey = c.PostForm("object_key")
	if form.RegionId == "" || form.Bucket == "" || form.ObjectKey == "" {
		result.GParamErr(c, fmt.Errorf("region_id/bucket/object_key 必填"))
		return
	}
	if form.MerchantId == 0 && form.CloudAccountId == 0 {
		result.GErr(c, fmt.Errorf("merchant_id或cloud_account_id必须提供一个"))
		return
	}
	file, _, err := c.Request.FormFile("file")
	if err != nil {
		result.GParamErr(c, err)
		return
	}
	defer file.Close()
	if err := cloud_aliyun.UploadOssObject(form.MerchantId, form.CloudAccountId, form.RegionId, form.Bucket, form.ObjectKey, file); err != nil {
		result.GErr(c, err)
		return
	}
	// 构建并返回访问URL
	region := form.RegionId
	if len(region) > 4 && region[:4] == "oss-" {
		region = region[4:]
	}
	url := fmt.Sprintf("https://%s.oss-%s.aliyuncs.com/%s", form.Bucket, region, form.ObjectKey)
	result.GOK(c, gin.H{"url": url, "object_key": form.ObjectKey})
}

// 下载：以二进制流响应
func downloadOssObject(c *gin.Context) {
	var req model.OssDownloadReq
	if err := c.ShouldBindQuery(&req); err != nil {
		result.GErr(c, err)
		return
	}
	if req.MerchantId == 0 && req.CloudAccountId == 0 {
		result.GErr(c, fmt.Errorf("merchant_id或cloud_account_id必须提供一个"))
		return
	}
	if req.RegionId == "" || req.Bucket == "" || req.ObjectKey == "" {
		result.GParamErr(c, fmt.Errorf("region_id/bucket/object_key 必填"))
		return
	}
	data, contentType, filename, err := cloud_aliyun.DownloadOssObject(req.MerchantId, req.CloudAccountId, req.RegionId, req.Bucket, req.ObjectKey)
	if err != nil {
		result.GErr(c, err)
		return
	}
	if req.Filename != "" {
		filename = req.Filename
	}
	if req.Attachment == 1 {
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	} else {
		c.Header("Content-Disposition", fmt.Sprintf("inline; filename=%q", filename))
	}
	c.Data(http.StatusOK, contentType, data)
}

// 列举 Buckets
func listOssBuckets(c *gin.Context) {
	var req model.OssListBucketsReq
	if err := c.ShouldBindQuery(&req); err != nil {
		result.GErr(c, err)
		return
	}
	if req.MerchantId == 0 && req.CloudAccountId == 0 {
		result.GErr(c, fmt.Errorf("merchant_id或cloud_account_id必须提供一个"))
		return
	}
	data, err := cloud_aliyun.ListBuckets(req)
	if err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, data)
}

// 删除对象
func deleteOssObject(c *gin.Context) {
	var req model.OssDownloadReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.GErr(c, err)
		return
	}
	if req.MerchantId == 0 && req.CloudAccountId == 0 {
		result.GErr(c, fmt.Errorf("merchant_id或cloud_account_id必须提供一个"))
		return
	}
	if req.RegionId == "" || req.Bucket == "" || req.ObjectKey == "" {
		result.GParamErr(c, fmt.Errorf("region_id/bucket/object_key 必填"))
		return
	}
	if err := cloud_aliyun.DeleteOssObject(req.MerchantId, req.CloudAccountId, req.RegionId, req.Bucket, req.ObjectKey); err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, nil)
}

// 创建 Bucket
func createOssBucket(c *gin.Context) {
	var req model.OssCreateBucketReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.GErr(c, err)
		return
	}
	if req.MerchantId == 0 && req.CloudAccountId == 0 {
		result.GErr(c, fmt.Errorf("merchant_id或cloud_account_id必须提供一个"))
		return
	}
	if err := cloud_aliyun.CreateBucket(req); err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, nil)
}

// 删除 Bucket
func deleteOssBucket(c *gin.Context) {
	var req model.OssDeleteBucketReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.GErr(c, err)
		return
	}
	if req.MerchantId == 0 && req.CloudAccountId == 0 {
		result.GErr(c, fmt.Errorf("merchant_id或cloud_account_id必须提供一个"))
		return
	}
	if err := cloud_aliyun.DeleteBucket(req); err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, nil)
}

// 设置 Bucket 公开访问
func setOssBucketPublic(c *gin.Context) {
	var req model.OssSetBucketPublicReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.GErr(c, err)
		return
	}
	if req.MerchantId == 0 && req.CloudAccountId == 0 {
		result.GErr(c, fmt.Errorf("merchant_id或cloud_account_id必须提供一个"))
		return
	}
	if err := cloud_aliyun.SetBucketPublicAccess(req); err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, nil)
}

// getAliyunAccountBalance 获取阿里云账号余额
func getAliyunAccountBalance(c *gin.Context) {
	var req struct {
		MerchantId     int   `form:"merchant_id"`
		CloudAccountId int64 `form:"cloud_account_id"`
	}
	if err := c.ShouldBindQuery(&req); err != nil {
		result.GParamErr(c, err)
		return
	}
	if req.MerchantId == 0 && req.CloudAccountId == 0 {
		result.GErr(c, fmt.Errorf("merchant_id或cloud_account_id必须提供一个"))
		return
	}
	bal, err := cloud_aliyun.GetAccountBalance(req.MerchantId, req.CloudAccountId)
	if err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, gin.H{"balance": bal})
}

// ========== 镜像管理相关 ==========

// createImage 创建自定义镜像
func createImage(c *gin.Context) {
	var req aliyun.CreateImageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		result.GErr(c, err)
		return
	}

	if req.MerchantId == 0 && req.CloudAccountId == 0 {
		result.GErr(c, fmt.Errorf("merchant_id或cloud_account_id必须提供一个"))
		return
	}

	imageId, err := aliyun.CreateImage(&req)
	if err != nil {
		result.GErr(c, err)
		return
	}

	result.GOK(c, gin.H{"image_id": imageId})
}

// describeImageSharePermission 查询镜像共享权限
func describeImageSharePermission(c *gin.Context) {
	var req struct {
		CloudAccountId int64  `form:"cloud_account_id"`
		MerchantId     int    `form:"merchant_id"`
		RegionId       string `form:"region_id" binding:"required"`
		ImageId        string `form:"image_id" binding:"required"`
	}
	if err := c.ShouldBindQuery(&req); err != nil {
		result.GErr(c, err)
		return
	}

	if req.MerchantId == 0 && req.CloudAccountId == 0 {
		result.GErr(c, fmt.Errorf("merchant_id或cloud_account_id必须提供一个"))
		return
	}

	shareInfo, err := aliyun.DescribeImageSharePermission(req.CloudAccountId, req.MerchantId, req.RegionId, req.ImageId)
	if err != nil {
		result.GErr(c, err)
		return
	}

	result.GOK(c, shareInfo)
}

// modifyImageSharePermission 修改镜像共享权限
func modifyImageSharePermission(c *gin.Context) {
	var req aliyun.ModifyImageSharePermissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		result.GErr(c, err)
		return
	}

	if req.MerchantId == 0 && req.CloudAccountId == 0 {
		result.GErr(c, fmt.Errorf("merchant_id或cloud_account_id必须提供一个"))
		return
	}

	if len(req.AddAccounts) == 0 && len(req.RemoveAccounts) == 0 {
		result.GErr(c, fmt.Errorf("add_accounts或remove_accounts至少提供一个"))
		return
	}

	err := aliyun.ModifyImageSharePermission(&req)
	if err != nil {
		result.GErr(c, err)
		return
	}

	result.GOK(c, nil)
}
