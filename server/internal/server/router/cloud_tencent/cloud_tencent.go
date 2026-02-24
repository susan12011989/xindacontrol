package cloud_tencent

import (
	"fmt"
	"net/http"
	"server/internal/server/cloud/tencent"
	"server/internal/server/middleware"
	"server/internal/server/model"
	"server/pkg/result"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// Routes 腾讯云服务API路由
func Routes(ge gin.IRouter) {
	cloudGroup := ge.Group("/cloud/tencent", middleware.Authorization)

	// 账户余额
	cloudGroup.GET("/balance", getAccountBalance)

	// CVM 实例管理
	cloudGroup.GET("/cvm/instances", listCvmInstances)
	cloudGroup.POST("/cvm/instances/create", createCvmInstances)
	cloudGroup.POST("/cvm/instance/operate", operateCvmInstance)
	cloudGroup.POST("/cvm/instances/operate", batchOperateCvmInstances)
	cloudGroup.POST("/cvm/instance/modify", modifyCvmInstance)
	cloudGroup.POST("/cvm/instance/reset-password", resetCvmPassword)
	cloudGroup.GET("/cvm/images", listCvmImages)
	cloudGroup.GET("/cvm/instance-types", listCvmInstanceTypes)

	// VPC/子网
	cloudGroup.GET("/vpcs", listVpcs)
	cloudGroup.GET("/subnets", listSubnets)

	// 安全组管理
	cloudGroup.GET("/security-groups", listSecurityGroups)
	cloudGroup.POST("/security-groups/create", createSecurityGroups)
	cloudGroup.GET("/security-group/policies", describeSecurityGroupPolicies)
	cloudGroup.POST("/security-group/authorize", authorizeSecurityGroup)
	cloudGroup.POST("/security-group/revoke", revokeSecurityGroup)

	// COS 对象存储
	cloudGroup.GET("/cos/buckets", listCosBuckets)
	cloudGroup.GET("/cos/objects", listCosObjects)
	cloudGroup.POST("/cos/object", uploadCosObject)
	cloudGroup.GET("/cos/object", downloadCosObject)
	cloudGroup.DELETE("/cos/object", deleteCosObject)
	cloudGroup.POST("/cos/bucket", createCosBucket)
	cloudGroup.DELETE("/cos/bucket", deleteCosBucket)
	cloudGroup.POST("/cos/bucket/set-public", setCosBucketPublic)
}

// ========== COS Handlers ==========

// listCosBuckets 列举 Bucket
func listCosBuckets(c *gin.Context) {
	var req model.CosListBucketsReq
	if err := c.ShouldBindQuery(&req); err != nil {
		result.GErr(c, err)
		return
	}
	if req.MerchantId == 0 && req.CloudAccountId == 0 {
		result.GErr(c, fmt.Errorf("merchant_id或cloud_account_id必须提供一个"))
		return
	}
	data, err := tencent.ListBuckets(req)
	if err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, data)
}

// listCosObjects 列举对象
func listCosObjects(c *gin.Context) {
	var req model.CosListObjectsReq
	if err := c.ShouldBindQuery(&req); err != nil {
		result.GErr(c, err)
		return
	}
	if req.MerchantId == 0 && req.CloudAccountId == 0 {
		result.GErr(c, fmt.Errorf("merchant_id或cloud_account_id必须提供一个"))
		return
	}
	data, err := tencent.ListObjects(req)
	if err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, data)
}

// uploadCosObject 上传对象（multipart/form-data）
func uploadCosObject(c *gin.Context) {
	var form model.CosUploadForm
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

	if err := tencent.UploadObject(form.MerchantId, form.CloudAccountId, form.RegionId, form.Bucket, form.ObjectKey, file); err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, nil)
}

// downloadCosObject 下载对象
func downloadCosObject(c *gin.Context) {
	var req model.CosDownloadReq
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

	data, contentType, filename, err := tencent.DownloadObject(req.MerchantId, req.CloudAccountId, req.RegionId, req.Bucket, req.ObjectKey)
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

// deleteCosObject 删除对象
func deleteCosObject(c *gin.Context) {
	var req model.CosDownloadReq
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

	if err := tencent.DeleteObject(req.MerchantId, req.CloudAccountId, req.RegionId, req.Bucket, req.ObjectKey); err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, nil)
}

// createCosBucket 创建 Bucket
func createCosBucket(c *gin.Context) {
	var req model.CosCreateBucketReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.GErr(c, err)
		return
	}
	if req.MerchantId == 0 && req.CloudAccountId == 0 {
		result.GErr(c, fmt.Errorf("merchant_id或cloud_account_id必须提供一个"))
		return
	}
	if err := tencent.CreateBucket(req.MerchantId, req.CloudAccountId, req.RegionId, req.Bucket); err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, nil)
}

// deleteCosBucket 删除 Bucket
func deleteCosBucket(c *gin.Context) {
	var req model.CosDeleteBucketReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.GErr(c, err)
		return
	}
	if req.MerchantId == 0 && req.CloudAccountId == 0 {
		result.GErr(c, fmt.Errorf("merchant_id或cloud_account_id必须提供一个"))
		return
	}
	if err := tencent.DeleteBucket(req.MerchantId, req.CloudAccountId, req.RegionId, req.Bucket); err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, nil)
}

// setCosBucketPublic 设置 Bucket 公开访问
func setCosBucketPublic(c *gin.Context) {
	var req model.CosSetBucketPublicReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.GErr(c, err)
		return
	}
	if req.MerchantId == 0 && req.CloudAccountId == 0 {
		result.GErr(c, fmt.Errorf("merchant_id或cloud_account_id必须提供一个"))
		return
	}
	if err := tencent.SetBucketPublicAccess(req.MerchantId, req.CloudAccountId, req.RegionId, req.Bucket, req.Public); err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, nil)
}

// ========== CVM Handlers ==========

// listCvmInstances 查询实例列表（多 region 并发）
func listCvmInstances(c *gin.Context) {
	var req model.TencentListReq
	if err := c.ShouldBindQuery(&req); err != nil {
		result.GParamErr(c, err)
		return
	}
	if req.MerchantId == 0 && req.CloudAccountId == 0 {
		result.GErr(c, fmt.Errorf("merchant_id或cloud_account_id必须提供一个"))
		return
	}

	cred, err := tencent.GetCloudCredentials(req.MerchantId, req.CloudAccountId)
	if err != nil {
		result.GErr(c, err)
		return
	}

	var allList []interface{}
	var wg sync.WaitGroup
	var mu sync.Mutex
	for _, regionId := range req.RegionId {
		wg.Add(1)
		go func(region string) {
			defer wg.Done()
			instances, _, err := tencent.DescribeInstances(cred, region)
			if err != nil || instances == nil {
				return
			}
			mu.Lock()
			for _, inst := range instances {
				allList = append(allList, inst)
			}
			mu.Unlock()
		}(regionId)
	}
	wg.Wait()

	result.GOK(c, map[string]interface{}{
		"list":  allList,
		"total": len(allList),
	})
}

// operateCvmInstance 单实例操作
func operateCvmInstance(c *gin.Context) {
	var req model.TencentOperateInstanceReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.GParamErr(c, err)
		return
	}
	if req.MerchantId == 0 && req.CloudAccountId == 0 {
		result.GErr(c, fmt.Errorf("merchant_id或cloud_account_id必须提供一个"))
		return
	}

	cred, err := tencent.GetCloudCredentials(req.MerchantId, req.CloudAccountId)
	if err != nil {
		result.GErr(c, err)
		return
	}

	ids := []string{req.InstanceId}
	switch req.Operation {
	case "start":
		err = tencent.StartInstances(cred, req.RegionId, ids)
	case "stop":
		err = tencent.StopInstances(cred, req.RegionId, ids)
	case "restart":
		err = tencent.RebootInstances(cred, req.RegionId, ids)
	case "delete":
		err = tencent.TerminateInstances(cred, req.RegionId, ids)
	default:
		result.GErr(c, fmt.Errorf("不支持的操作: %s", req.Operation))
		return
	}

	if err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, nil)
}

// batchOperateCvmInstances 批量实例操作
func batchOperateCvmInstances(c *gin.Context) {
	var req model.TencentBatchOperateInstanceReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.GParamErr(c, err)
		return
	}
	if req.MerchantId == 0 && req.CloudAccountId == 0 {
		result.GErr(c, fmt.Errorf("merchant_id或cloud_account_id必须提供一个"))
		return
	}

	cred, err := tencent.GetCloudCredentials(req.MerchantId, req.CloudAccountId)
	if err != nil {
		result.GErr(c, err)
		return
	}

	switch req.Operation {
	case "start":
		err = tencent.StartInstances(cred, req.RegionId, req.InstanceIds)
	case "stop":
		err = tencent.StopInstances(cred, req.RegionId, req.InstanceIds)
	case "restart":
		err = tencent.RebootInstances(cred, req.RegionId, req.InstanceIds)
	case "delete":
		err = tencent.TerminateInstances(cred, req.RegionId, req.InstanceIds)
	default:
		result.GErr(c, fmt.Errorf("不支持的操作: %s", req.Operation))
		return
	}

	if err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, nil)
}

// modifyCvmInstance 修改实例属性
func modifyCvmInstance(c *gin.Context) {
	var req model.TencentModifyInstanceReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.GParamErr(c, err)
		return
	}
	if req.MerchantId == 0 && req.CloudAccountId == 0 {
		result.GErr(c, fmt.Errorf("merchant_id或cloud_account_id必须提供一个"))
		return
	}

	cred, err := tencent.GetCloudCredentials(req.MerchantId, req.CloudAccountId)
	if err != nil {
		result.GErr(c, err)
		return
	}

	// 修改名称
	if req.InstanceName != "" {
		if err := tencent.ModifyInstancesAttribute(cred, req.RegionId, req.InstanceId, req.InstanceName); err != nil {
			result.GErr(c, err)
			return
		}
	}

	// 修改安全组绑定
	if len(req.SecurityGroupIds) > 0 {
		if err := tencent.ModifyInstancesSecurityGroups(cred, req.RegionId, []string{req.InstanceId}, req.SecurityGroupIds); err != nil {
			result.GErr(c, err)
			return
		}
	}

	result.GOK(c, nil)
}

// resetCvmPassword 重置实例密码
func resetCvmPassword(c *gin.Context) {
	var req model.TencentResetPasswordReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.GParamErr(c, err)
		return
	}
	if req.MerchantId == 0 && req.CloudAccountId == 0 {
		result.GErr(c, fmt.Errorf("merchant_id或cloud_account_id必须提供一个"))
		return
	}

	cred, err := tencent.GetCloudCredentials(req.MerchantId, req.CloudAccountId)
	if err != nil {
		result.GErr(c, err)
		return
	}

	if err := tencent.ResetInstancesPassword(cred, req.RegionId, req.InstanceId, req.Password); err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, nil)
}

// listCvmImages 查询镜像列表
func listCvmImages(c *gin.Context) {
	var req model.TencentListReq
	if err := c.ShouldBindQuery(&req); err != nil {
		result.GParamErr(c, err)
		return
	}
	if req.MerchantId == 0 && req.CloudAccountId == 0 {
		result.GErr(c, fmt.Errorf("merchant_id或cloud_account_id必须提供一个"))
		return
	}

	cred, err := tencent.GetCloudCredentials(req.MerchantId, req.CloudAccountId)
	if err != nil {
		result.GErr(c, err)
		return
	}

	// 取第一个 region 查询镜像
	regionId := req.RegionId[0]
	images, total, err := tencent.DescribeImages(cred, regionId)
	if err != nil {
		result.GErr(c, err)
		return
	}

	result.GOK(c, map[string]interface{}{
		"list":  images,
		"total": total,
	})
}

// listCvmInstanceTypes 查询实例规格
func listCvmInstanceTypes(c *gin.Context) {
	var req model.TencentListReq
	if err := c.ShouldBindQuery(&req); err != nil {
		result.GParamErr(c, err)
		return
	}
	if req.MerchantId == 0 && req.CloudAccountId == 0 {
		result.GErr(c, fmt.Errorf("merchant_id或cloud_account_id必须提供一个"))
		return
	}

	cred, err := tencent.GetCloudCredentials(req.MerchantId, req.CloudAccountId)
	if err != nil {
		result.GErr(c, err)
		return
	}

	regionId := req.RegionId[0]
	types, err := tencent.DescribeInstanceTypeConfigs(cred, regionId)
	if err != nil {
		result.GErr(c, err)
		return
	}

	result.GOK(c, map[string]interface{}{
		"list":  types,
		"total": len(types),
	})
}

// ========== Security Group Handlers ==========

// listSecurityGroups 查询安全组列表
func listSecurityGroups(c *gin.Context) {
	var req model.TencentListSecurityGroupsReq
	if err := c.ShouldBindQuery(&req); err != nil {
		result.GParamErr(c, err)
		return
	}
	if req.MerchantId == 0 && req.CloudAccountId == 0 {
		result.GErr(c, fmt.Errorf("merchant_id或cloud_account_id必须提供一个"))
		return
	}

	cred, err := tencent.GetCloudCredentials(req.MerchantId, req.CloudAccountId)
	if err != nil {
		result.GErr(c, err)
		return
	}

	groups, err := tencent.DescribeSecurityGroups(cred, req.RegionId)
	if err != nil {
		result.GErr(c, err)
		return
	}

	result.GOK(c, map[string]interface{}{
		"list":  groups,
		"total": len(groups),
	})
}

// describeSecurityGroupPolicies 查询安全组规则
func describeSecurityGroupPolicies(c *gin.Context) {
	var req model.TencentDescribeSecurityGroupReq
	if err := c.ShouldBindQuery(&req); err != nil {
		result.GParamErr(c, err)
		return
	}
	if req.MerchantId == 0 && req.CloudAccountId == 0 {
		result.GErr(c, fmt.Errorf("merchant_id或cloud_account_id必须提供一个"))
		return
	}

	cred, err := tencent.GetCloudCredentials(req.MerchantId, req.CloudAccountId)
	if err != nil {
		result.GErr(c, err)
		return
	}

	policies, err := tencent.DescribeSecurityGroupPolicies(cred, req.RegionId, req.SecurityGroupId)
	if err != nil {
		result.GErr(c, err)
		return
	}

	result.GOK(c, policies)
}

// authorizeSecurityGroup 添加入站规则
func authorizeSecurityGroup(c *gin.Context) {
	var req model.TencentSecurityGroupPolicyReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.GParamErr(c, err)
		return
	}
	if req.MerchantId == 0 && req.CloudAccountId == 0 {
		result.GErr(c, fmt.Errorf("merchant_id或cloud_account_id必须提供一个"))
		return
	}

	cred, err := tencent.GetCloudCredentials(req.MerchantId, req.CloudAccountId)
	if err != nil {
		result.GErr(c, err)
		return
	}

	var rules []tencent.SecurityGroupRuleInput
	for _, p := range req.Policies {
		rules = append(rules, tencent.SecurityGroupRuleInput{
			Protocol:    p.Protocol,
			Port:        p.Port,
			CidrBlock:   p.CidrBlock,
			Action:      p.Action,
			Description: p.Description,
		})
	}

	if err := tencent.CreateSecurityGroupIngress(cred, req.RegionId, req.SecurityGroupId, rules); err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, nil)
}

// revokeSecurityGroup 删除入站规则
func revokeSecurityGroup(c *gin.Context) {
	var req model.TencentDeleteSecurityGroupPoliciesReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.GParamErr(c, err)
		return
	}
	if req.MerchantId == 0 && req.CloudAccountId == 0 {
		result.GErr(c, fmt.Errorf("merchant_id或cloud_account_id必须提供一个"))
		return
	}

	cred, err := tencent.GetCloudCredentials(req.MerchantId, req.CloudAccountId)
	if err != nil {
		result.GErr(c, err)
		return
	}

	if err := tencent.DeleteSecurityGroupIngress(cred, req.RegionId, req.SecurityGroupId, req.PolicyIndexes); err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, nil)
}

// ========== Create Handlers (Streaming) ==========

// createCvmInstances 流式创建 CVM 实例
func createCvmInstances(c *gin.Context) {
	var req model.TencentCreateInstancesReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.GStreamEnd(c, false, err.Error())
		return
	}

	result.GStream(c)

	for i, item := range req.List {
		if item.MerchantId == 0 && item.CloudAccountId == 0 {
			result.GStreamData(c, gin.H{"message": fmt.Sprintf("%s [%d] 缺少 merchant_id 或 cloud_account_id", time.Now().Format(time.DateTime), i+1)})
			continue
		}

		cred, err := tencent.GetCloudCredentials(item.MerchantId, item.CloudAccountId)
		if err != nil {
			result.GStreamData(c, gin.H{"message": fmt.Sprintf("%s [%d] 获取凭证失败: %v", time.Now().Format(time.DateTime), i+1, err)})
			continue
		}

		result.GStreamData(c, gin.H{"message": fmt.Sprintf("%s [%d] 创建实例中 %s %s %s...", time.Now().Format(time.DateTime), i+1, item.RegionId, item.Zone, item.InstanceType)})

		input := &tencent.RunInstancesInput{
			RegionId:                item.RegionId,
			Zone:                    item.Zone,
			ImageId:                 item.ImageId,
			InstanceType:            item.InstanceType,
			InstanceChargeType:      item.InstanceChargeType,
			SystemDiskType:          item.SystemDiskType,
			SystemDiskSize:          item.SystemDiskSize,
			VpcId:                   item.VpcId,
			SubnetId:                item.SubnetId,
			SecurityGroupIds:        item.SecurityGroupIds,
			InstanceName:            item.InstanceName,
			Password:                item.Password,
			Period:                  item.Period,
			RenewFlag:               item.RenewFlag,
			InternetMaxBandwidthOut: item.InternetMaxBandwidthOut,
		}

		instanceIds, err := tencent.RunInstances(cred, input)
		if err != nil {
			result.GStreamData(c, gin.H{"message": fmt.Sprintf("%s [%d] 创建实例失败: %v", time.Now().Format(time.DateTime), i+1, err)})
			continue
		}

		for _, id := range instanceIds {
			result.GStreamData(c, gin.H{"message": fmt.Sprintf("%s [%d] 创建实例成功: %s", time.Now().Format(time.DateTime), i+1, id)})
		}
	}

	result.GStreamEnd(c, true, "创建实例执行完毕")
}

// createSecurityGroups 流式创建安全组
func createSecurityGroups(c *gin.Context) {
	var req model.TencentCreateSecurityGroupsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.GStreamEnd(c, false, err.Error())
		return
	}

	// GOST 相关默认入站规则
	defaultRules := []tencent.SecurityGroupRuleInput{
		{Protocol: "TCP", Port: "22", CidrBlock: "0.0.0.0/0", Action: "ACCEPT", Description: "SSH"},
		{Protocol: "TCP", Port: "80", CidrBlock: "0.0.0.0/0", Action: "ACCEPT", Description: "HTTP"},
		{Protocol: "TCP", Port: "443", CidrBlock: "0.0.0.0/0", Action: "ACCEPT", Description: "HTTPS"},
		{Protocol: "TCP", Port: "58182", CidrBlock: "0.0.0.0/0", Action: "ACCEPT", Description: "Control"},
		{Protocol: "TCP", Port: "10000-10012", CidrBlock: "0.0.0.0/0", Action: "ACCEPT", Description: "GOST TCP"},
		{Protocol: "TCP", Port: "32798-32804", CidrBlock: "0.0.0.0/0", Action: "ACCEPT", Description: "GOST Tunnel"},
		{Protocol: "TCP", Port: "9394", CidrBlock: "0.0.0.0/0", Action: "ACCEPT", Description: "GOST API"},
	}

	result.GStream(c)

	for i, item := range req.List {
		if item.MerchantId == 0 && item.CloudAccountId == 0 {
			result.GStreamData(c, gin.H{"message": fmt.Sprintf("%s [%d] 缺少 merchant_id 或 cloud_account_id", time.Now().Format(time.DateTime), i+1)})
			continue
		}

		cred, err := tencent.GetCloudCredentials(item.MerchantId, item.CloudAccountId)
		if err != nil {
			result.GStreamData(c, gin.H{"message": fmt.Sprintf("%s [%d] 获取凭证失败: %v", time.Now().Format(time.DateTime), i+1, err)})
			continue
		}

		name := item.Name
		if name == "" {
			name = fmt.Sprintf("sg-%s", time.Now().Format("20060102150405"))
		}

		result.GStreamData(c, gin.H{"message": fmt.Sprintf("%s [%d] 创建安全组中 %s %s...", time.Now().Format(time.DateTime), i+1, item.RegionId, name)})

		sgId, err := tencent.CreateSecurityGroupNew(cred, item.RegionId, name, item.Description)
		if err != nil {
			result.GStreamData(c, gin.H{"message": fmt.Sprintf("%s [%d] 创建安全组失败: %v", time.Now().Format(time.DateTime), i+1, err)})
			continue
		}

		result.GStreamData(c, gin.H{"message": fmt.Sprintf("%s [%d] 创建安全组成功: %s", time.Now().Format(time.DateTime), i+1, sgId)})

		// 添加默认规则
		result.GStreamData(c, gin.H{"message": fmt.Sprintf("%s [%d] 添加默认规则(22,80,443,58182,10000-10012,32798-32804,9394)...", time.Now().Format(time.DateTime), i+1)})

		if err := tencent.CreateSecurityGroupIngress(cred, item.RegionId, sgId, defaultRules); err != nil {
			result.GStreamData(c, gin.H{"message": fmt.Sprintf("%s [%d] 添加默认规则失败: %v", time.Now().Format(time.DateTime), i+1, err)})
			continue
		}

		result.GStreamData(c, gin.H{"message": fmt.Sprintf("%s [%d] 添加默认规则成功", time.Now().Format(time.DateTime), i+1)})
	}

	result.GStreamEnd(c, true, "创建安全组执行完毕")
}

// ========== VPC/Subnet Handlers ==========

// listVpcs 查询 VPC 列表
func listVpcs(c *gin.Context) {
	var req model.TencentVpcReq
	if err := c.ShouldBindQuery(&req); err != nil {
		result.GParamErr(c, err)
		return
	}
	if req.MerchantId == 0 && req.CloudAccountId == 0 {
		result.GErr(c, fmt.Errorf("merchant_id或cloud_account_id必须提供一个"))
		return
	}

	cred, err := tencent.GetCloudCredentials(req.MerchantId, req.CloudAccountId)
	if err != nil {
		result.GErr(c, err)
		return
	}

	vpcs, err := tencent.DescribeVpcs(cred, req.RegionId)
	if err != nil {
		result.GErr(c, err)
		return
	}

	result.GOK(c, map[string]interface{}{
		"list":  vpcs,
		"total": len(vpcs),
	})
}

// listSubnets 查询子网列表
func listSubnets(c *gin.Context) {
	var req model.TencentVpcReq
	if err := c.ShouldBindQuery(&req); err != nil {
		result.GParamErr(c, err)
		return
	}
	if req.MerchantId == 0 && req.CloudAccountId == 0 {
		result.GErr(c, fmt.Errorf("merchant_id或cloud_account_id必须提供一个"))
		return
	}

	cred, err := tencent.GetCloudCredentials(req.MerchantId, req.CloudAccountId)
	if err != nil {
		result.GErr(c, err)
		return
	}

	subnets, err := tencent.DescribeSubnets(cred, req.RegionId, req.VpcId)
	if err != nil {
		result.GErr(c, err)
		return
	}

	result.GOK(c, map[string]interface{}{
		"list":  subnets,
		"total": len(subnets),
	})
}

// ========== Billing Handlers ==========

// getAccountBalance 查询腾讯云账户余额
func getAccountBalance(c *gin.Context) {
	merchantIdStr := c.Query("merchant_id")
	cloudAccountIdStr := c.Query("cloud_account_id")

	var merchantId int
	var cloudAccountId int64
	if merchantIdStr != "" {
		merchantId, _ = strconv.Atoi(merchantIdStr)
	}
	if cloudAccountIdStr != "" {
		cloudAccountId, _ = strconv.ParseInt(cloudAccountIdStr, 10, 64)
	}

	if merchantId == 0 && cloudAccountId == 0 {
		result.GErr(c, fmt.Errorf("merchant_id或cloud_account_id必须提供一个"))
		return
	}

	data, err := tencent.GetAccountBalance(merchantId, cloudAccountId)
	if err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, data)
}
