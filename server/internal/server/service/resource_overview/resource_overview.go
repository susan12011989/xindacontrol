package resource_overview

import (
	"fmt"
	"net/http"
	"server/internal/dbhelper"
	"server/internal/server/model"
	merchantService "server/internal/server/service/merchant"
	"server/pkg/dbs"
	"server/pkg/entity"
	"strings"
	"sync"
	"time"

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
			DownloadUrl:      GetOssDownloadUrl(c.CloudType, c.MerchantOssConfigs.Bucket, c.MerchantOssConfigs.Region, c.MerchantOssConfigs.Endpoint, c.MerchantOssConfigs.CustomDomain),
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

// ========== OSS 下载地址生成 ==========

// GetOssDownloadUrl 根据云类型、Bucket、Region 等信息生成 OSS 下载基础地址
func GetOssDownloadUrl(cloudType, bucket, region, endpoint, customDomain string) string {
	if customDomain != "" {
		if strings.HasPrefix(customDomain, "http") {
			return customDomain
		}
		return "https://" + customDomain
	}
	if endpoint != "" {
		if strings.HasPrefix(endpoint, "http") {
			return endpoint
		}
		return "https://" + bucket + "." + endpoint
	}
	switch cloudType {
	case "aws":
		return fmt.Sprintf("https://%s.s3.%s.amazonaws.com", bucket, region)
	case "aliyun":
		return fmt.Sprintf("https://%s.oss-%s.aliyuncs.com", bucket, region)
	case "tencent":
		return fmt.Sprintf("https://%s.cos.%s.myqcloud.com", bucket, region)
	default:
		return ""
	}
}

// ========== OSS 健康检测（URL 可达性） ==========

const maxHealthCheckConcurrent = 10

// CheckOssHealth 批量检测 OSS 下载地址可达性（不需要 KEY，只检测 URL 是否可访问）
func CheckOssHealth(ossConfigIds []int) []model.OssHealthCheckResult {
	results := make([]model.OssHealthCheckResult, len(ossConfigIds))

	sem := make(chan struct{}, maxHealthCheckConcurrent)
	var wg sync.WaitGroup

	for i, configId := range ossConfigIds {
		wg.Add(1)
		go func(idx, id int) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			results[idx] = checkSingleOssUrl(id)
		}(i, configId)
	}

	wg.Wait()
	return results
}

// checkSingleOssUrl 检测单个 OSS 配置的下载地址可达性
func checkSingleOssUrl(ossConfigId int) model.OssHealthCheckResult {
	start := time.Now()
	result := model.OssHealthCheckResult{OssConfigId: ossConfigId}

	// 获取 OSS 配置
	var ossConfig entity.MerchantOssConfigs
	has, err := dbs.DBAdmin.ID(ossConfigId).Get(&ossConfig)
	if err != nil || !has {
		result.Message = fmt.Sprintf("OSS配置不存在: %d", ossConfigId)
		result.Latency = time.Since(start).Round(time.Millisecond).String()
		return result
	}
	result.OssConfigName = ossConfig.Name
	result.Bucket = ossConfig.Bucket

	// 获取商户名
	var merchant entity.Merchants
	if ok, _ := dbs.DBAdmin.ID(ossConfig.MerchantId).Get(&merchant); ok {
		result.MerchantName = merchant.Name
	}

	// 获取云类型
	acc, _ := dbhelper.GetCloudAccountByID(ossConfig.CloudAccountId)
	if acc != nil {
		result.CloudType = acc.CloudType
	}

	// 生成下载地址
	result.DownloadUrl = GetOssDownloadUrl(result.CloudType, ossConfig.Bucket, ossConfig.Region, ossConfig.Endpoint, ossConfig.CustomDomain)
	if result.DownloadUrl == "" {
		result.Message = "无法生成下载地址"
		result.Latency = time.Since(start).Round(time.Millisecond).String()
		return result
	}

	// 检测下载地址可达性
	result.Healthy, result.StatusCode, result.Message = checkUrlReachable(result.DownloadUrl)

	// 如果有 CDN 域名且与下载地址不同，额外检测
	if ossConfig.CustomDomain != "" {
		cdnUrl := "https://" + ossConfig.CustomDomain
		if !strings.HasPrefix(ossConfig.CustomDomain, "http") {
			cdnUrl = "https://" + ossConfig.CustomDomain
		} else {
			cdnUrl = ossConfig.CustomDomain
		}
		if cdnUrl != result.DownloadUrl {
			result.CdnUrl = cdnUrl
			healthy, code, _ := checkUrlReachable(cdnUrl)
			result.CdnHealthy = &healthy
			result.CdnStatusCode = &code
		}
	}

	result.Latency = time.Since(start).Round(time.Millisecond).String()
	return result
}

// checkUrlReachable HTTP 请求检测 URL 是否可达（任何 HTTP 响应都视为可达）
func checkUrlReachable(url string) (bool, int, string) {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Head(url)
	if err != nil {
		// HEAD 不支持时尝试 GET
		resp, err = client.Get(url)
		if err != nil {
			return false, 0, fmt.Sprintf("不可达: %v", err)
		}
	}
	defer resp.Body.Close()
	// 任何 HTTP 响应都表示端点可达（包括 403/404）
	return true, resp.StatusCode, fmt.Sprintf("HTTP %d", resp.StatusCode)
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
