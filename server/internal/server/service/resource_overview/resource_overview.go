package resource_overview

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"server/internal/dbhelper"
	"server/internal/server/cloud/aliyun"
	"server/internal/server/cloud/tencent"
	"server/internal/server/model"
	merchantService "server/internal/server/service/merchant"
	cloud_aliyun "server/internal/server/service/cloud_aliyun"
	cloud_aws "server/internal/server/service/cloud_aws"
	"server/pkg/dbs"
	"server/pkg/entity"
	"sync"
	"time"

	awscloud "server/internal/server/cloud/aws"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/zeromicro/go-zero/core/logx"
)

// ========== 标签 CRUD ==========

func ListTags() ([]model.ResourceTagResp, error) {
	var tags []entity.ResourceTags
	err := dbs.DBAdmin.OrderBy("id ASC").Find(&tags)
	if err != nil {
		return nil, err
	}

	result := make([]model.ResourceTagResp, len(tags))
	for i, t := range tags {
		result[i] = model.ResourceTagResp{
			Id:          t.Id,
			Name:        t.Name,
			Color:       t.Color,
			Description: t.Description,
			CreatedAt:   t.CreatedAt.Format(time.DateTime),
		}
	}
	return result, nil
}

func CreateTag(req model.ResourceTagReq) (int, error) {
	tag := &entity.ResourceTags{
		Name:        req.Name,
		Color:       req.Color,
		Description: req.Description,
		CreatedAt:   time.Now(),
	}
	_, err := dbs.DBAdmin.Insert(tag)
	if err != nil {
		return 0, err
	}
	return tag.Id, nil
}

func UpdateTag(id int, req model.ResourceTagReq) error {
	_, err := dbs.DBAdmin.Table("resource_tags").Where("id = ?", id).Update(map[string]interface{}{
		"name":        req.Name,
		"color":       req.Color,
		"description": req.Description,
	})
	return err
}

func DeleteTag(id int) error {
	// 同时清理关联关系
	_, _ = dbs.DBAdmin.Where("tag_id = ?", id).Delete(&entity.ResourceTagRelations{})
	_, err := dbs.DBAdmin.ID(id).Delete(&entity.ResourceTags{})
	return err
}

// ========== 标签分配 ==========

func AssignTags(req model.AssignTagsReq) error {
	for _, resourceId := range req.ResourceIds {
		for _, tagId := range req.TagIds {
			// 忽略已存在的（UNIQUE INDEX 会去重）
			relation := &entity.ResourceTagRelations{
				TagId:        tagId,
				ResourceType: req.ResourceType,
				ResourceId:   resourceId,
				CreatedAt:    time.Now(),
			}
			_, _ = dbs.DBAdmin.Insert(relation)
		}
	}
	return nil
}

func RemoveTags(req model.RemoveTagsReq) error {
	for _, resourceId := range req.ResourceIds {
		_, err := dbs.DBAdmin.Where("resource_type = ? AND resource_id = ? AND tag_id IN (?)",
			req.ResourceType, resourceId, req.TagIds).Delete(&entity.ResourceTagRelations{})
		if err != nil {
			return err
		}
	}
	return nil
}

// CleanResourceTags 删除资源时清理标签关联
func CleanResourceTags(resourceType string, resourceId int) {
	_, err := dbs.DBAdmin.Where("resource_type = ? AND resource_id = ?", resourceType, resourceId).Delete(&entity.ResourceTagRelations{})
	if err != nil {
		logx.Errorf("clean resource tags err: resource_type=%s resource_id=%d err=%v", resourceType, resourceId, err)
	}
}

// GetResourceTagsMap 批量获取资源的标签（返回 map[resourceId][]TagResp）
func GetResourceTagsMap(resourceType string, resourceIds []int) map[int][]model.ResourceTagResp {
	result := make(map[int][]model.ResourceTagResp)
	if len(resourceIds) == 0 {
		return result
	}

	// 查询关联关系
	var relations []entity.ResourceTagRelations
	err := dbs.DBAdmin.Where("resource_type = ?", resourceType).In("resource_id", resourceIds).Find(&relations)
	if err != nil {
		logx.Errorf("get resource tag relations err: %v", err)
		return result
	}
	if len(relations) == 0 {
		return result
	}

	// 获取标签详情
	tagIds := make([]int, 0)
	for _, r := range relations {
		tagIds = append(tagIds, r.TagId)
	}
	var tags []entity.ResourceTags
	err = dbs.DBAdmin.In("id", tagIds).Find(&tags)
	if err != nil {
		logx.Errorf("get resource tags err: %v", err)
		return result
	}
	tagMap := make(map[int]entity.ResourceTags)
	for _, t := range tags {
		tagMap[t.Id] = t
	}

	// 组装结果
	for _, r := range relations {
		tag, ok := tagMap[r.TagId]
		if !ok {
			continue
		}
		result[r.ResourceId] = append(result[r.ResourceId], model.ResourceTagResp{
			Id:    tag.Id,
			Name:  tag.Name,
			Color: tag.Color,
		})
	}
	return result
}

// ========== 全局 OSS 配置列表 ==========

func QueryGlobalOssConfigs(req model.QueryGlobalOssConfigsReq) (model.QueryGlobalOssConfigsResponse, error) {
	var resp model.QueryGlobalOssConfigsResponse

	// 构建查询
	type OssWithInfo struct {
		entity.MerchantOssConfigs `xorm:"extends"`
		MerchantName              string `xorm:"'merchant_name'"`
		MerchantNo                string `xorm:"'merchant_no'"`
		CloudAccountName          string `xorm:"'cloud_account_name'"`
		CloudType                 string `xorm:"'cloud_type'"`
	}

	session := dbs.DBAdmin.Table("merchant_oss_configs").Alias("moc").
		Join("LEFT", []string{"merchants", "m"}, "m.id = moc.merchant_id").
		Join("LEFT", []string{"cloud_accounts", "ca"}, "ca.id = moc.cloud_account_id").
		Select("moc.*, m.name AS merchant_name, m.no AS merchant_no, ca.name AS cloud_account_name, ca.cloud_type AS cloud_type")

	// 条件过滤
	if req.MerchantId > 0 {
		session = session.Where("moc.merchant_id = ?", req.MerchantId)
	}
	if req.CloudType != "" {
		session = session.Where("ca.cloud_type = ?", req.CloudType)
	}
	if req.Region != "" {
		session = session.Where("moc.region = ?", req.Region)
	}
	if req.Status != nil {
		session = session.Where("moc.status = ?", *req.Status)
	}
	if req.TagId > 0 {
		session = session.Where("moc.id IN (SELECT resource_id FROM resource_tag_relations WHERE resource_type = ? AND tag_id = ?)",
			entity.ResourceTypeOssConfig, req.TagId)
	}

	// 分页
	if req.Page == 0 {
		req.Page = 1
	}
	if req.Size == 0 {
		req.Size = 20
	}
	offset := (req.Page - 1) * req.Size

	var configs []OssWithInfo
	total, err := session.Desc("moc.id").Limit(req.Size, offset).FindAndCount(&configs)
	if err != nil {
		logx.Errorf("query global oss configs err: %+v", err)
		return resp, err
	}
	resp.Total = int(total)

	// 批量获取标签
	resourceIds := make([]int, len(configs))
	for i, c := range configs {
		resourceIds[i] = c.Id
	}
	tagsMap := GetResourceTagsMap(entity.ResourceTypeOssConfig, resourceIds)

	// 组装响应
	for _, c := range configs {
		tags := tagsMap[c.Id]
		if tags == nil {
			tags = []model.ResourceTagResp{}
		}
		resp.List = append(resp.List, model.GlobalOssConfigResp{
			Id:               c.MerchantOssConfigs.Id,
			MerchantId:       c.MerchantOssConfigs.MerchantId,
			MerchantName:     c.MerchantName,
			MerchantNo:       c.MerchantNo,
			CloudAccountId:   c.MerchantOssConfigs.CloudAccountId,
			CloudAccountName: c.CloudAccountName,
			CloudType:        c.CloudType,
			Name:             c.MerchantOssConfigs.Name,
			Bucket:           c.MerchantOssConfigs.Bucket,
			Region:           c.MerchantOssConfigs.Region,
			Endpoint:         c.MerchantOssConfigs.Endpoint,
			CustomDomain:     c.MerchantOssConfigs.CustomDomain,
			IsDefault:        c.MerchantOssConfigs.IsDefault,
			Status:           c.MerchantOssConfigs.Status,
			Tags:             tags,
			CreatedAt:        c.MerchantOssConfigs.CreatedAt.Format(time.DateTime),
			UpdatedAt:        c.MerchantOssConfigs.UpdatedAt.Format(time.DateTime),
		})
	}

	return resp, nil
}

// ========== 全局 GOST 服务器列表 ==========

func QueryGlobalGostServers(req model.QueryGlobalGostServersReq) (model.QueryGlobalGostServersResponse, error) {
	var resp model.QueryGlobalGostServersResponse

	type GostWithInfo struct {
		entity.MerchantGostServers `xorm:"extends"`
		MerchantName               string `xorm:"'merchant_name'"`
		MerchantNo                 string `xorm:"'merchant_no'"`
		ServerName                 string `xorm:"'server_name'"`
		ServerHost                 string `xorm:"'server_host'"`
	}

	session := dbs.DBAdmin.Table("merchant_gost_servers").Alias("mgs").
		Join("LEFT", []string{"merchants", "m"}, "m.id = mgs.merchant_id").
		Join("LEFT", []string{"servers", "s"}, "s.id = mgs.server_id").
		Select("mgs.*, m.name AS merchant_name, m.no AS merchant_no, s.name AS server_name, s.host AS server_host")

	// 条件过滤
	if req.MerchantId > 0 {
		session = session.Where("mgs.merchant_id = ?", req.MerchantId)
	}
	if req.CloudType != "" {
		session = session.Where("mgs.cloud_type = ?", req.CloudType)
	}
	if req.Region != "" {
		session = session.Where("mgs.region = ?", req.Region)
	}
	if req.Status != nil {
		session = session.Where("mgs.status = ?", *req.Status)
	}
	if req.TagId > 0 {
		session = session.Where("mgs.id IN (SELECT resource_id FROM resource_tag_relations WHERE resource_type = ? AND tag_id = ?)",
			entity.ResourceTypeGostServer, req.TagId)
	}

	// 分页
	if req.Page == 0 {
		req.Page = 1
	}
	if req.Size == 0 {
		req.Size = 20
	}
	offset := (req.Page - 1) * req.Size

	var servers []GostWithInfo
	total, err := session.Desc("mgs.id").Limit(req.Size, offset).FindAndCount(&servers)
	if err != nil {
		logx.Errorf("query global gost servers err: %+v", err)
		return resp, err
	}
	resp.Total = int(total)

	// 批量获取标签
	resourceIds := make([]int, len(servers))
	for i, s := range servers {
		resourceIds[i] = s.Id
	}
	tagsMap := GetResourceTagsMap(entity.ResourceTypeGostServer, resourceIds)

	// 组装响应
	for _, s := range servers {
		tags := tagsMap[s.Id]
		if tags == nil {
			tags = []model.ResourceTagResp{}
		}
		resp.List = append(resp.List, model.GlobalGostServerResp{
			Id:           s.MerchantGostServers.Id,
			MerchantId:   s.MerchantGostServers.MerchantId,
			MerchantName: s.MerchantName,
			MerchantNo:   s.MerchantNo,
			ServerId:     s.MerchantGostServers.ServerId,
			ServerName:   s.ServerName,
			ServerHost:   s.ServerHost,
			CloudType:    s.MerchantGostServers.CloudType,
			Region:       s.MerchantGostServers.Region,
			ListenPort:   s.MerchantGostServers.ListenPort,
			IsPrimary:    s.MerchantGostServers.IsPrimary,
			Priority:     s.MerchantGostServers.Priority,
			Status:       s.MerchantGostServers.Status,
			Remark:       s.MerchantGostServers.Remark,
			Tags:         tags,
			CreatedAt:    s.MerchantGostServers.CreatedAt.Format(time.DateTime),
			UpdatedAt:    s.MerchantGostServers.UpdatedAt.Format(time.DateTime),
		})
	}

	return resp, nil
}

// ========== OSS 健康检测 ==========

const healthCheckObjectKey = "_health_check.txt"
const healthCheckContent = "health_check_ok"
const maxHealthCheckConcurrent = 5

// CheckOssHealth 批量检测 OSS 健康状态
func CheckOssHealth(ossConfigIds []int) []model.OssHealthCheckResult {
	results := make([]model.OssHealthCheckResult, len(ossConfigIds))

	// 并发控制
	sem := make(chan struct{}, maxHealthCheckConcurrent)
	var wg sync.WaitGroup

	for i, configId := range ossConfigIds {
		wg.Add(1)
		go func(idx, id int) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			results[idx] = checkSingleOssHealth(id)
		}(i, configId)
	}

	wg.Wait()
	return results
}

// checkSingleOssHealth 检测单个 OSS 配置的健康状态
func checkSingleOssHealth(ossConfigId int) model.OssHealthCheckResult {
	startTime := time.Now()
	result := model.OssHealthCheckResult{
		OssConfigId: ossConfigId,
		Healthy:     true,
	}

	// 1. 获取 OSS 配置
	var ossConfig entity.MerchantOssConfigs
	has, err := dbs.DBAdmin.ID(ossConfigId).Get(&ossConfig)
	if err != nil || !has {
		result.Healthy = false
		result.Steps = []model.OssHealthStepResult{{Step: "sdk_connect", Ok: false, Message: fmt.Sprintf("OSS配置不存在: %d", ossConfigId)}}
		result.Duration = time.Since(startTime).Round(time.Millisecond).String()
		return result
	}
	result.OssConfigName = ossConfig.Name
	result.Bucket = ossConfig.Bucket
	result.Region = ossConfig.Region

	// 获取商户名
	var merchant entity.Merchants
	if ok, _ := dbs.DBAdmin.ID(ossConfig.MerchantId).Get(&merchant); ok {
		result.MerchantName = merchant.Name
	}

	// 获取云类型
	acc, err := dbhelper.GetCloudAccountByID(ossConfig.CloudAccountId)
	if err != nil {
		result.Healthy = false
		result.Steps = []model.OssHealthStepResult{{Step: "sdk_connect", Ok: false, Message: fmt.Sprintf("获取云账号失败: %v", err)}}
		result.Duration = time.Since(startTime).Round(time.Millisecond).String()
		return result
	}
	result.CloudType = acc.CloudType

	// 2. SDK 连接检测（尝试 ListObjects）
	sdkStep := checkSdkConnect(acc, &ossConfig)
	result.Steps = append(result.Steps, sdkStep)
	if !sdkStep.Ok {
		result.Healthy = false
		result.Duration = time.Since(startTime).Round(time.Millisecond).String()
		return result
	}

	// 3. 上传测试文件
	uploadStep, publicUrl := checkUpload(acc, &ossConfig)
	result.Steps = append(result.Steps, uploadStep)
	if !uploadStep.Ok {
		result.Healthy = false
		result.Duration = time.Since(startTime).Round(time.Millisecond).String()
		return result
	}
	result.PublicUrl = publicUrl

	// 4. SDK 下载检测
	downloadSdkStep := checkDownloadSdk(acc, &ossConfig)
	result.Steps = append(result.Steps, downloadSdkStep)
	if !downloadSdkStep.Ok {
		result.Healthy = false
	}

	// 5. 公网 URL 下载检测
	downloadUrlStep := checkDownloadUrl(publicUrl)
	result.Steps = append(result.Steps, downloadUrlStep)
	if !downloadUrlStep.Ok {
		result.Healthy = false
	}

	// 6. 自定义域名检测（如果配置了）
	if ossConfig.CustomDomain != "" {
		cdnUrl := fmt.Sprintf("https://%s/%s", ossConfig.CustomDomain, healthCheckObjectKey)
		cdnStep := checkDownloadUrl(cdnUrl)
		cdnStep.Step = "download_cdn"
		result.Steps = append(result.Steps, cdnStep)
		ok := cdnStep.Ok
		result.CustomDomainOk = &ok
	}

	// 7. 清理测试文件
	cleanupStep := checkCleanup(acc, &ossConfig)
	result.Steps = append(result.Steps, cleanupStep)

	result.Duration = time.Since(startTime).Round(time.Millisecond).String()
	return result
}

// checkSdkConnect SDK 连接检测（ListObjects MaxKeys=1）
func checkSdkConnect(acc *entity.CloudAccounts, ossConfig *entity.MerchantOssConfigs) model.OssHealthStepResult {
	step := model.OssHealthStepResult{Step: "sdk_connect"}
	start := time.Now()

	switch acc.CloudType {
	case "aws":
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		cli, err := awscloud.NewS3Client(ctx, acc, ossConfig.Region)
		if err != nil {
			step.Message = fmt.Sprintf("创建S3客户端失败: %v", err)
			step.Latency = time.Since(start).Round(time.Millisecond).String()
			return step
		}
		maxKeys := int32(1)
		_, err = cli.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
			Bucket:  &ossConfig.Bucket,
			MaxKeys: &maxKeys,
		})
		if err != nil {
			step.Message = fmt.Sprintf("Bucket不可访问: %v", err)
			step.Latency = time.Since(start).Round(time.Millisecond).String()
			return step
		}

	case "aliyun":
		bucket, err := aliyun.GetOssBucket(0, ossConfig.CloudAccountId, ossConfig.Region, ossConfig.Bucket)
		if err != nil {
			step.Message = fmt.Sprintf("获取Bucket失败: %v", err)
			step.Latency = time.Since(start).Round(time.Millisecond).String()
			return step
		}
		_, err = bucket.ListObjects()
		if err != nil {
			step.Message = fmt.Sprintf("Bucket不可访问: %v", err)
			step.Latency = time.Since(start).Round(time.Millisecond).String()
			return step
		}

	case "tencent":
		_, err := tencent.ListObjects(model.CosListObjectsReq{
			MerchantId:     0,
			CloudAccountId: ossConfig.CloudAccountId,
			RegionId:       ossConfig.Region,
			Bucket:         ossConfig.Bucket,
			MaxKeys:        1,
		})
		if err != nil {
			step.Message = fmt.Sprintf("Bucket不可访问: %v", err)
			step.Latency = time.Since(start).Round(time.Millisecond).String()
			return step
		}

	default:
		step.Message = fmt.Sprintf("不支持的云类型: %s", acc.CloudType)
		step.Latency = time.Since(start).Round(time.Millisecond).String()
		return step
	}

	step.Ok = true
	step.Message = "SDK连接正常"
	step.Latency = time.Since(start).Round(time.Millisecond).String()
	return step
}

// checkUpload 上传测试文件
func checkUpload(acc *entity.CloudAccounts, ossConfig *entity.MerchantOssConfigs) (model.OssHealthStepResult, string) {
	step := model.OssHealthStepResult{Step: "upload"}
	start := time.Now()
	var publicUrl string

	data := []byte(healthCheckContent)
	reader := bytes.NewReader(data)

	switch acc.CloudType {
	case "aws":
		err := cloud_aws.UploadObject(acc, ossConfig.Region, ossConfig.Bucket, healthCheckObjectKey, reader)
		if err != nil {
			step.Message = fmt.Sprintf("上传失败: %v", err)
			step.Latency = time.Since(start).Round(time.Millisecond).String()
			return step, ""
		}
		publicUrl = fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", ossConfig.Bucket, ossConfig.Region, healthCheckObjectKey)

	case "aliyun":
		err := cloud_aliyun.UploadOssObject(0, ossConfig.CloudAccountId, ossConfig.Region, ossConfig.Bucket, healthCheckObjectKey, reader)
		if err != nil {
			step.Message = fmt.Sprintf("上传失败: %v", err)
			step.Latency = time.Since(start).Round(time.Millisecond).String()
			return step, ""
		}
		publicUrl = fmt.Sprintf("https://%s.oss-%s.aliyuncs.com/%s", ossConfig.Bucket, ossConfig.Region, healthCheckObjectKey)

	case "tencent":
		err := tencent.UploadObject(0, ossConfig.CloudAccountId, ossConfig.Region, ossConfig.Bucket, healthCheckObjectKey, reader)
		if err != nil {
			step.Message = fmt.Sprintf("上传失败: %v", err)
			step.Latency = time.Since(start).Round(time.Millisecond).String()
			return step, ""
		}
		publicUrl = fmt.Sprintf("https://%s.cos.%s.myqcloud.com/%s", ossConfig.Bucket, ossConfig.Region, healthCheckObjectKey)
	}

	step.Ok = true
	step.Message = "上传成功"
	step.Latency = time.Since(start).Round(time.Millisecond).String()
	return step, publicUrl
}

// checkDownloadSdk SDK 下载检测
func checkDownloadSdk(acc *entity.CloudAccounts, ossConfig *entity.MerchantOssConfigs) model.OssHealthStepResult {
	step := model.OssHealthStepResult{Step: "download_sdk"}
	start := time.Now()

	var content []byte
	var err error

	switch acc.CloudType {
	case "aws":
		content, _, _, err = cloud_aws.DownloadObject(acc, ossConfig.Region, ossConfig.Bucket, healthCheckObjectKey)
	case "aliyun":
		content, _, _, err = cloud_aliyun.DownloadOssObject(0, ossConfig.CloudAccountId, ossConfig.Region, ossConfig.Bucket, healthCheckObjectKey)
	case "tencent":
		content, _, _, err = tencent.DownloadObject(0, ossConfig.CloudAccountId, ossConfig.Region, ossConfig.Bucket, healthCheckObjectKey)
	}

	if err != nil {
		step.Message = fmt.Sprintf("SDK下载失败: %v", err)
		step.Latency = time.Since(start).Round(time.Millisecond).String()
		return step
	}
	if string(content) != healthCheckContent {
		step.Message = "下载内容不匹配"
		step.Latency = time.Since(start).Round(time.Millisecond).String()
		return step
	}

	step.Ok = true
	step.Message = "SDK下载正常"
	step.Latency = time.Since(start).Round(time.Millisecond).String()
	return step
}

// checkDownloadUrl 公网 URL 下载检测
func checkDownloadUrl(url string) model.OssHealthStepResult {
	step := model.OssHealthStepResult{Step: "download_url"}
	start := time.Now()

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		step.Message = fmt.Sprintf("URL不可访问: %v", err)
		step.Latency = time.Since(start).Round(time.Millisecond).String()
		return step
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		step.Message = fmt.Sprintf("HTTP状态码: %d", resp.StatusCode)
		step.Latency = time.Since(start).Round(time.Millisecond).String()
		return step
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		step.Message = fmt.Sprintf("读取响应失败: %v", err)
		step.Latency = time.Since(start).Round(time.Millisecond).String()
		return step
	}

	if string(body) != healthCheckContent {
		step.Message = "URL下载内容不匹配"
		step.Latency = time.Since(start).Round(time.Millisecond).String()
		return step
	}

	step.Ok = true
	step.Message = "URL访问正常"
	step.Latency = time.Since(start).Round(time.Millisecond).String()
	return step
}

// checkCleanup 清理测试文件
func checkCleanup(acc *entity.CloudAccounts, ossConfig *entity.MerchantOssConfigs) model.OssHealthStepResult {
	step := model.OssHealthStepResult{Step: "cleanup"}
	start := time.Now()

	var err error
	switch acc.CloudType {
	case "aws":
		err = cloud_aws.DeleteObject(acc, ossConfig.Region, ossConfig.Bucket, healthCheckObjectKey)
	case "aliyun":
		err = cloud_aliyun.DeleteOssObject(0, ossConfig.CloudAccountId, ossConfig.Region, ossConfig.Bucket, healthCheckObjectKey)
	case "tencent":
		err = tencent.DeleteObject(0, ossConfig.CloudAccountId, ossConfig.Region, ossConfig.Bucket, healthCheckObjectKey)
	}

	if err != nil {
		step.Message = fmt.Sprintf("清理失败: %v", err)
		step.Latency = time.Since(start).Round(time.Millisecond).String()
		return step
	}

	step.Ok = true
	step.Message = "清理完成"
	step.Latency = time.Since(start).Round(time.Millisecond).String()
	return step
}

// ========== 批量操作 ==========

// BatchSyncGostIPByFilter 按筛选条件批量同步 GOST IP
func BatchSyncGostIPByFilter(req model.BatchSyncGostIPByFilterReq) ([]merchantService.SyncGostIPResp, error) {
	merchantIds := req.MerchantIds

	// 如果没有指定商户，按筛选条件查找
	if len(merchantIds) == 0 {
		session := dbs.DBAdmin.Table("merchants").Select("id")
		if req.CloudType != "" {
			// 找到拥有该云类型 OSS 的商户
			session = session.Where("id IN (SELECT moc.merchant_id FROM merchant_oss_configs moc LEFT JOIN cloud_accounts ca ON ca.id = moc.cloud_account_id WHERE ca.cloud_type = ?)", req.CloudType)
		}
		if req.TagId > 0 {
			session = session.Where("id IN (SELECT DISTINCT mgs.merchant_id FROM merchant_gost_servers mgs INNER JOIN resource_tag_relations rtr ON rtr.resource_type = ? AND rtr.resource_id = mgs.id AND rtr.tag_id = ?)",
				entity.ResourceTypeGostServer, req.TagId)
		}

		var merchants []entity.Merchants
		err := session.Find(&merchants)
		if err != nil {
			return nil, fmt.Errorf("查询商户列表失败: %v", err)
		}
		for _, m := range merchants {
			merchantIds = append(merchantIds, m.Id)
		}
	}

	if len(merchantIds) == 0 {
		return nil, fmt.Errorf("未找到符合条件的商户")
	}

	// 复用现有的批量同步逻辑
	return merchantService.BatchSyncMerchantGostIP(merchantIds)
}
