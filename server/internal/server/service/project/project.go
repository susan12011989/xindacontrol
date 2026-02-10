package project

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"server/internal/dbhelper"
	"server/internal/server/cloud/tencent"
	cloud_aliyun "server/internal/server/service/cloud_aliyun"
	cloud_aws "server/internal/server/service/cloud_aws"
	"server/pkg/dbs"
	"server/pkg/entity"

	"github.com/zeromicro/go-zero/core/logx"
)

// ========== 项目 CRUD ==========

// ProjectReq 项目请求
type ProjectReq struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	Status      int    `json:"status"`
}

// ProjectResp 项目响应
type ProjectResp struct {
	Id           int    `json:"id"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	Status       int    `json:"status"`
	MerchantCount int   `json:"merchant_count"` // 商户数量
	GostServerCount int `json:"gost_server_count"` // GOST服务器数量
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

// ListProjects 获取项目列表
func ListProjects(page, size int, name string) ([]ProjectResp, int, error) {
	// 计数
	countSession := dbs.DBAdmin.Table("projects")
	if name != "" {
		countSession = countSession.Where("name LIKE ?", "%"+name+"%")
	}
	total, err := countSession.Count(&entity.Projects{})
	if err != nil {
		return nil, 0, err
	}

	// 分页查询
	querySession := dbs.DBAdmin.Table("projects")
	if name != "" {
		querySession = querySession.Where("name LIKE ?", "%"+name+"%")
	}
	var projects []entity.Projects
	offset := (page - 1) * size
	err = querySession.Desc("id").Limit(size, offset).Find(&projects)
	if err != nil {
		return nil, 0, err
	}

	// 获取每个项目的商户数量和GOST服务器数量
	result := make([]ProjectResp, len(projects))
	for i, p := range projects {
		merchantCount, _ := dbs.DBAdmin.Where("project_id = ?", p.Id).Count(&entity.Merchants{})
		gostCount, _ := dbs.DBAdmin.Where("project_id = ?", p.Id).Count(&entity.ProjectGostServers{})

		result[i] = ProjectResp{
			Id:              p.Id,
			Name:            p.Name,
			Description:     p.Description,
			Status:          p.Status,
			MerchantCount:   int(merchantCount),
			GostServerCount: int(gostCount),
			CreatedAt:       p.CreatedAt.Format(time.DateTime),
			UpdatedAt:       p.UpdatedAt.Format(time.DateTime),
		}
	}

	return result, int(total), nil
}

// GetProject 获取项目详情
func GetProject(id int) (*ProjectResp, error) {
	var project entity.Projects
	has, err := dbs.DBAdmin.ID(id).Get(&project)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, fmt.Errorf("项目不存在")
	}

	merchantCount, _ := dbs.DBAdmin.Where("project_id = ?", id).Count(&entity.Merchants{})
	gostCount, _ := dbs.DBAdmin.Where("project_id = ?", id).Count(&entity.ProjectGostServers{})

	return &ProjectResp{
		Id:              project.Id,
		Name:            project.Name,
		Description:     project.Description,
		Status:          project.Status,
		MerchantCount:   int(merchantCount),
		GostServerCount: int(gostCount),
		CreatedAt:       project.CreatedAt.Format(time.DateTime),
		UpdatedAt:       project.UpdatedAt.Format(time.DateTime),
	}, nil
}

// CreateProject 创建项目
func CreateProject(req ProjectReq) (int, error) {
	// 检查名称是否已存在
	count, err := dbs.DBAdmin.Where("name = ?", req.Name).Count(&entity.Projects{})
	if err != nil {
		return 0, err
	}
	if count > 0 {
		return 0, fmt.Errorf("项目名称已存在")
	}

	project := &entity.Projects{
		Name:        req.Name,
		Description: req.Description,
		Status:      1,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	_, err = dbs.DBAdmin.Insert(project)
	if err != nil {
		return 0, err
	}

	return project.Id, nil
}

// UpdateProject 更新项目
func UpdateProject(id int, req ProjectReq) error {
	var project entity.Projects
	has, err := dbs.DBAdmin.ID(id).Get(&project)
	if err != nil {
		return err
	}
	if !has {
		return fmt.Errorf("项目不存在")
	}

	// 检查名称是否与其他项目重复
	if req.Name != project.Name {
		count, err := dbs.DBAdmin.Where("name = ? AND id != ?", req.Name, id).Count(&entity.Projects{})
		if err != nil {
			return err
		}
		if count > 0 {
			return fmt.Errorf("项目名称已存在")
		}
	}

	updates := map[string]interface{}{
		"name":        req.Name,
		"description": req.Description,
		"updated_at":  time.Now(),
	}
	if req.Status != 0 {
		updates["status"] = req.Status
	}

	_, err = dbs.DBAdmin.Table("projects").Where("id = ?", id).Update(updates)
	return err
}

// DeleteProject 删除项目
func DeleteProject(id int) error {
	// 检查是否有商户关联
	count, _ := dbs.DBAdmin.Where("project_id = ?", id).Count(&entity.Merchants{})
	if count > 0 {
		return fmt.Errorf("项目下有 %d 个商户，无法删除", count)
	}

	// 删除项目关联的 GOST 服务器
	_, _ = dbs.DBAdmin.Where("project_id = ?", id).Delete(&entity.ProjectGostServers{})

	// 删除项目
	_, err := dbs.DBAdmin.ID(id).Delete(&entity.Projects{})
	return err
}

// ========== 项目 GOST 服务器管理 ==========

// ProjectGostServerReq 项目 GOST 服务器请求
type ProjectGostServerReq struct {
	ProjectId int    `json:"project_id"`
	ServerId  int    `json:"server_id" binding:"required"`
	IsPrimary int    `json:"is_primary"`
	Priority  int    `json:"priority"`
	Status    int    `json:"status"`
	Remark    string `json:"remark"`
}

// ProjectGostServerResp 项目 GOST 服务器响应
type ProjectGostServerResp struct {
	Id           int    `json:"id"`
	ProjectId    int    `json:"project_id"`
	ServerId     int    `json:"server_id"`
	ServerName   string `json:"server_name"`
	ServerHost   string `json:"server_host"`
	AuxiliaryIP  string `json:"auxiliary_ip"`
	IsPrimary    int    `json:"is_primary"`
	Priority     int    `json:"priority"`
	Status       int    `json:"status"`
	Remark       string `json:"remark"`
	ServerStatus int    `json:"server_status"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

// ListProjectGostServers 获取项目的 GOST 服务器列表
func ListProjectGostServers(projectId int) ([]ProjectGostServerResp, error) {
	var relations []entity.ProjectGostServers
	err := dbs.DBAdmin.Where("project_id = ?", projectId).
		OrderBy("is_primary DESC, priority ASC, id ASC").Find(&relations)
	if err != nil {
		return nil, err
	}

	// 获取服务器信息
	serverIds := make([]int, len(relations))
	for i, r := range relations {
		serverIds[i] = r.ServerId
	}

	serverMap := make(map[int]entity.Servers)
	if len(serverIds) > 0 {
		var servers []entity.Servers
		err = dbs.DBAdmin.In("id", serverIds).Find(&servers)
		if err != nil {
			return nil, err
		}
		for _, s := range servers {
			serverMap[s.Id] = s
		}
	}

	result := make([]ProjectGostServerResp, len(relations))
	for i, r := range relations {
		server := serverMap[r.ServerId]
		result[i] = ProjectGostServerResp{
			Id:           r.Id,
			ProjectId:    r.ProjectId,
			ServerId:     r.ServerId,
			ServerName:   server.Name,
			ServerHost:   server.Host,
			AuxiliaryIP:  server.AuxiliaryIP,
			IsPrimary:    r.IsPrimary,
			Priority:     r.Priority,
			Status:       r.Status,
			Remark:       r.Remark,
			ServerStatus: server.Status,
			CreatedAt:    r.CreatedAt.Format(time.DateTime),
			UpdatedAt:    r.UpdatedAt.Format(time.DateTime),
		}
	}

	return result, nil
}

// CreateProjectGostServer 为项目添加 GOST 服务器
func CreateProjectGostServer(projectId int, req ProjectGostServerReq) (int, error) {
	// 检查项目是否存在
	has, err := dbs.DBAdmin.ID(projectId).Exist(&entity.Projects{})
	if err != nil {
		return 0, err
	}
	if !has {
		return 0, fmt.Errorf("项目不存在")
	}

	// 检查服务器是否存在
	has, err = dbs.DBAdmin.ID(req.ServerId).Exist(&entity.Servers{})
	if err != nil {
		return 0, err
	}
	if !has {
		return 0, fmt.Errorf("服务器不存在")
	}

	// 检查是否已关联
	has, err = dbs.DBAdmin.Where("project_id = ? AND server_id = ?", projectId, req.ServerId).
		Exist(&entity.ProjectGostServers{})
	if err != nil {
		return 0, err
	}
	if has {
		return 0, fmt.Errorf("该服务器已关联此项目")
	}

	// 如果设置为主服务器，先清除其他主服务器
	if req.IsPrimary == 1 {
		_, _ = dbs.DBAdmin.Exec("UPDATE project_gost_servers SET is_primary = 0 WHERE project_id = ?", projectId)
	}

	relation := &entity.ProjectGostServers{
		ProjectId: projectId,
		ServerId:  req.ServerId,
		IsPrimary: req.IsPrimary,
		Priority:  req.Priority,
		Status:    1,
		Remark:    req.Remark,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	_, err = dbs.DBAdmin.Insert(relation)
	if err != nil {
		return 0, err
	}

	return relation.Id, nil
}

// UpdateProjectGostServer 更新项目 GOST 服务器关联
func UpdateProjectGostServer(id int, req ProjectGostServerReq) error {
	var relation entity.ProjectGostServers
	has, err := dbs.DBAdmin.ID(id).Get(&relation)
	if err != nil {
		return err
	}
	if !has {
		return fmt.Errorf("关联记录不存在")
	}

	// 如果设置为主服务器，先清除其他主服务器
	if req.IsPrimary == 1 {
		_, _ = dbs.DBAdmin.Exec("UPDATE project_gost_servers SET is_primary = 0 WHERE project_id = ? AND id != ?",
			relation.ProjectId, id)
	}

	updates := map[string]interface{}{
		"is_primary": req.IsPrimary,
		"priority":   req.Priority,
		"remark":     req.Remark,
		"updated_at": time.Now(),
	}
	if req.Status != 0 {
		updates["status"] = req.Status
	}

	_, err = dbs.DBAdmin.Table("project_gost_servers").Where("id = ?", id).Update(updates)
	return err
}

// DeleteProjectGostServer 删除项目 GOST 服务器关联
func DeleteProjectGostServer(id int) error {
	_, err := dbs.DBAdmin.ID(id).Delete(&entity.ProjectGostServers{})
	return err
}

// ========== 项目商户管理 ==========

// ProjectMerchantResp 项目商户响应
type ProjectMerchantResp struct {
	Id        int    `json:"id"`
	No        string `json:"no"`
	Name      string `json:"name"`
	ServerIP  string `json:"server_ip"`
	Status    int    `json:"status"`
	CreatedAt string `json:"created_at"`
}

// ListProjectMerchants 获取项目下的商户列表
func ListProjectMerchants(projectId int) ([]ProjectMerchantResp, error) {
	var merchants []entity.Merchants
	err := dbs.DBAdmin.Where("project_id = ?", projectId).OrderBy("id ASC").Find(&merchants)
	if err != nil {
		return nil, err
	}

	result := make([]ProjectMerchantResp, len(merchants))
	for i, m := range merchants {
		result[i] = ProjectMerchantResp{
			Id:        m.Id,
			No:        m.No,
			Name:      m.Name,
			ServerIP:  m.ServerIP,
			Status:    m.Status,
			CreatedAt: m.CreatedAt.Format(time.DateTime),
		}
	}

	return result, nil
}

// AddMerchantToProject 将商户添加到项目
func AddMerchantToProject(projectId, merchantId int) error {
	// 检查项目是否存在
	has, err := dbs.DBAdmin.ID(projectId).Exist(&entity.Projects{})
	if err != nil {
		return err
	}
	if !has {
		return fmt.Errorf("项目不存在")
	}

	// 更新商户的 project_id
	_, err = dbs.DBAdmin.Exec("UPDATE merchants SET project_id = ? WHERE id = ?", projectId, merchantId)
	return err
}

// RemoveMerchantFromProject 将商户从项目移除
func RemoveMerchantFromProject(merchantId int) error {
	_, err := dbs.DBAdmin.Exec("UPDATE merchants SET project_id = 0 WHERE id = ?", merchantId)
	return err
}

// BatchAddMerchantsToProject 批量将商户添加到项目
func BatchAddMerchantsToProject(projectId int, merchantIds []int) error {
	if len(merchantIds) == 0 {
		return nil
	}

	// 检查项目是否存在
	has, err := dbs.DBAdmin.ID(projectId).Exist(&entity.Projects{})
	if err != nil {
		return err
	}
	if !has {
		return fmt.Errorf("项目不存在")
	}

	// 批量更新
	_, err = dbs.DBAdmin.In("id", merchantIds).
		Update(&entity.Merchants{ProjectId: projectId})
	return err
}

// ========== 项目 IP 同步 ==========

// SyncProjectGostIPReq 同步项目 GOST IP 请求
type SyncProjectGostIPReq struct {
	ProjectId int      `json:"project_id" binding:"required"`
	IPs       []string `json:"ips"`        // 指定 IP 列表，留空则自动从项目 GOST 服务器获取
	ObjectKey string   `json:"object_key"` // 自定义 OSS 对象名，留空使用默认
}

// SyncProjectGostIPResp 同步项目 GOST IP 响应
type SyncProjectGostIPResp struct {
	ProjectId     int                      `json:"project_id"`
	ProjectName   string                   `json:"project_name"`
	IPs           []string                 `json:"ips"`
	MerchantCount int                      `json:"merchant_count"`
	Results       []MerchantSyncResult     `json:"results"`
	Summary       ProjectSyncSummary       `json:"summary"`
}

// MerchantSyncResult 商户同步结果
type MerchantSyncResult struct {
	MerchantId   int              `json:"merchant_id"`
	MerchantName string           `json:"merchant_name"`
	OssResults   []OssSyncResult  `json:"oss_results"`
	Success      bool             `json:"success"`
	Error        string           `json:"error,omitempty"`
}

// OssSyncResult OSS 同步结果
type OssSyncResult struct {
	OssConfigId   int    `json:"oss_config_id"`
	OssConfigName string `json:"oss_config_name"`
	CloudType     string `json:"cloud_type"`
	Bucket        string `json:"bucket"`
	ObjectKey     string `json:"object_key"`
	ObjectUrl     string `json:"object_url"`
	Success       bool   `json:"success"`
	Error         string `json:"error,omitempty"`
}

// ProjectSyncSummary 项目同步摘要
type ProjectSyncSummary struct {
	TotalMerchants   int    `json:"total_merchants"`
	SuccessMerchants int    `json:"success_merchants"`
	FailMerchants    int    `json:"fail_merchants"`
	TotalOss         int    `json:"total_oss"`
	SuccessOss       int    `json:"success_oss"`
	FailOss          int    `json:"fail_oss"`
	Duration         string `json:"duration"`
}

// SyncProjectGostIP 同步项目的 GOST IP 到所有商户的 OSS
func SyncProjectGostIP(req SyncProjectGostIPReq) (*SyncProjectGostIPResp, error) {
	startTime := time.Now()

	// 1. 获取项目信息
	var project entity.Projects
	has, err := dbs.DBAdmin.ID(req.ProjectId).Get(&project)
	if err != nil {
		return nil, fmt.Errorf("查询项目失败: %v", err)
	}
	if !has {
		return nil, fmt.Errorf("项目不存在: %d", req.ProjectId)
	}

	// 2. 获取 IP 列表
	ips := req.IPs
	if len(ips) == 0 {
		ips, err = getProjectGostIPs(req.ProjectId)
		if err != nil {
			return nil, fmt.Errorf("获取项目 GOST IP 失败: %v", err)
		}
	}
	if len(ips) == 0 {
		return nil, fmt.Errorf("项目没有配置 GOST 服务器")
	}

	// 3. 获取项目下的所有商户
	var merchants []entity.Merchants
	err = dbs.DBAdmin.Where("project_id = ?", req.ProjectId).Find(&merchants)
	if err != nil {
		return nil, fmt.Errorf("获取项目商户失败: %v", err)
	}
	if len(merchants) == 0 {
		return nil, fmt.Errorf("项目下没有商户")
	}

	// 4. 准备上传内容
	objectKey := req.ObjectKey
	if objectKey == "" {
		objectKey = "ip.txt"
	}
	uploadData := []byte(strings.Join(ips, "\n"))

	// 5. 同步到每个商户的 OSS
	results := make([]MerchantSyncResult, 0, len(merchants))
	totalOss, successOss, failOss := 0, 0, 0
	successMerchants, failMerchants := 0, 0

	for _, merchant := range merchants {
		merchantResult := MerchantSyncResult{
			MerchantId:   merchant.Id,
			MerchantName: merchant.Name,
			OssResults:   []OssSyncResult{},
			Success:      true,
		}

		// 获取商户的 OSS 配置
		ossConfigs, err := getMerchantOssConfigs(merchant.Id)
		if err != nil {
			merchantResult.Success = false
			merchantResult.Error = fmt.Sprintf("获取 OSS 配置失败: %v", err)
			failMerchants++
			results = append(results, merchantResult)
			continue
		}

		if len(ossConfigs) == 0 {
			merchantResult.Success = false
			merchantResult.Error = "商户未配置 OSS"
			failMerchants++
			results = append(results, merchantResult)
			continue
		}

		merchantSuccess := true
		for _, ossConfig := range ossConfigs {
			totalOss++
			ossResult := OssSyncResult{
				OssConfigId:   ossConfig.Id,
				OssConfigName: ossConfig.Name,
				CloudType:     ossConfig.CloudType,
				Bucket:        ossConfig.Bucket,
				ObjectKey:     objectKey,
			}

			objectUrl, uploadErr := uploadToOss(ossConfig, objectKey, uploadData)
			if uploadErr != nil {
				ossResult.Success = false
				ossResult.Error = uploadErr.Error()
				failOss++
				merchantSuccess = false
				logx.Errorf("同步项目%d商户%d OSS[%s]失败: %v",
					req.ProjectId, merchant.Id, ossConfig.Name, uploadErr)
			} else {
				ossResult.Success = true
				ossResult.ObjectUrl = objectUrl
				successOss++
				logx.Infof("同步项目%d商户%d OSS[%s]成功: %s",
					req.ProjectId, merchant.Id, ossConfig.Name, objectUrl)
			}
			merchantResult.OssResults = append(merchantResult.OssResults, ossResult)
		}

		merchantResult.Success = merchantSuccess
		if merchantSuccess {
			successMerchants++
		} else {
			failMerchants++
		}
		results = append(results, merchantResult)
	}

	duration := time.Since(startTime)

	return &SyncProjectGostIPResp{
		ProjectId:     req.ProjectId,
		ProjectName:   project.Name,
		IPs:           ips,
		MerchantCount: len(merchants),
		Results:       results,
		Summary: ProjectSyncSummary{
			TotalMerchants:   len(merchants),
			SuccessMerchants: successMerchants,
			FailMerchants:    failMerchants,
			TotalOss:         totalOss,
			SuccessOss:       successOss,
			FailOss:          failOss,
			Duration:         duration.String(),
		},
	}, nil
}

// getProjectGostIPs 获取项目的 GOST 服务器 IP 列表
func getProjectGostIPs(projectId int) ([]string, error) {
	var relations []entity.ProjectGostServers
	err := dbs.DBAdmin.Where("project_id = ? AND status = 1", projectId).
		OrderBy("is_primary DESC, priority ASC").Find(&relations)
	if err != nil {
		return nil, err
	}

	if len(relations) == 0 {
		return nil, nil
	}

	serverIds := make([]int, len(relations))
	for i, r := range relations {
		serverIds[i] = r.ServerId
	}

	var servers []entity.Servers
	err = dbs.DBAdmin.In("id", serverIds).Where("status = 1").Find(&servers)
	if err != nil {
		return nil, err
	}

	serverMap := make(map[int]entity.Servers)
	for _, s := range servers {
		serverMap[s.Id] = s
	}

	// 按优先级收集 IP
	ips := make([]string, 0)
	for _, r := range relations {
		if server, ok := serverMap[r.ServerId]; ok {
			if server.Host != "" {
				ips = append(ips, server.Host)
			}
			if server.AuxiliaryIP != "" {
				ips = append(ips, server.AuxiliaryIP)
			}
		}
	}

	return ips, nil
}

// MerchantOssConfig 商户 OSS 配置（简化版）
type MerchantOssConfig struct {
	Id             int
	Name           string
	CloudType      string
	CloudAccountId int64
	Bucket         string
	Region         string
}

// getMerchantOssConfigs 获取商户的 OSS 配置列表
func getMerchantOssConfigs(merchantId int) ([]MerchantOssConfig, error) {
	var configs []entity.MerchantOssConfigs
	err := dbs.DBAdmin.Where("merchant_id = ? AND status = 1", merchantId).Find(&configs)
	if err != nil {
		return nil, err
	}

	// 获取云账号信息
	accountIds := make([]int64, 0, len(configs))
	for _, c := range configs {
		accountIds = append(accountIds, c.CloudAccountId)
	}

	accountMap := make(map[int64]entity.CloudAccounts)
	if len(accountIds) > 0 {
		var accounts []entity.CloudAccounts
		err = dbs.DBAdmin.In("id", accountIds).Find(&accounts)
		if err != nil {
			return nil, err
		}
		for _, a := range accounts {
			accountMap[a.Id] = a
		}
	}

	result := make([]MerchantOssConfig, len(configs))
	for i, c := range configs {
		account := accountMap[c.CloudAccountId]
		result[i] = MerchantOssConfig{
			Id:             c.Id,
			Name:           c.Name,
			CloudType:      account.CloudType,
			CloudAccountId: c.CloudAccountId,
			Bucket:         c.Bucket,
			Region:         c.Region,
		}
	}

	return result, nil
}

// uploadToOss 上传到 OSS
func uploadToOss(ossConfig MerchantOssConfig, objectKey string, data []byte) (string, error) {
	reader := bytes.NewReader(data)

	switch ossConfig.CloudType {
	case "aws":
		acc, err := dbhelper.GetCloudAccountByID(ossConfig.CloudAccountId)
		if err != nil {
			return "", fmt.Errorf("获取 AWS 账号失败: %v", err)
		}
		err = cloud_aws.UploadObject(acc, ossConfig.Region, ossConfig.Bucket, objectKey, reader)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", ossConfig.Bucket, ossConfig.Region, objectKey), nil

	case "aliyun":
		err := cloud_aliyun.UploadOssObject(0, ossConfig.CloudAccountId, ossConfig.Region, ossConfig.Bucket, objectKey, reader)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("https://%s.oss-%s.aliyuncs.com/%s", ossConfig.Bucket, ossConfig.Region, objectKey), nil

	case "tencent":
		err := tencent.UploadObject(0, ossConfig.CloudAccountId, ossConfig.Region, ossConfig.Bucket, objectKey, reader)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("https://%s.cos.%s.myqcloud.com/%s", ossConfig.Bucket, ossConfig.Region, objectKey), nil

	default:
		return "", fmt.Errorf("不支持的云类型: %s", ossConfig.CloudType)
	}
}

// GetProjectSyncStatus 获取项目 GOST IP 同步状态
func GetProjectSyncStatus(projectId int) (*ProjectSyncStatusResp, error) {
	// 获取项目信息
	var project entity.Projects
	has, err := dbs.DBAdmin.ID(projectId).Get(&project)
	if err != nil {
		return nil, fmt.Errorf("查询项目失败: %v", err)
	}
	if !has {
		return nil, fmt.Errorf("项目不存在: %d", projectId)
	}

	// 获取项目 GOST 服务器
	gostServers, err := ListProjectGostServers(projectId)
	if err != nil {
		return nil, fmt.Errorf("获取 GOST 服务器失败: %v", err)
	}

	// 获取项目商户
	merchants, err := ListProjectMerchants(projectId)
	if err != nil {
		return nil, fmt.Errorf("获取商户列表失败: %v", err)
	}

	// 收集 IP
	ips := make([]string, 0)
	for _, g := range gostServers {
		if g.ServerHost != "" {
			ips = append(ips, g.ServerHost)
		}
		if g.AuxiliaryIP != "" {
			ips = append(ips, g.AuxiliaryIP)
		}
	}

	return &ProjectSyncStatusResp{
		ProjectId:   projectId,
		ProjectName: project.Name,
		GostServers: gostServers,
		Merchants:   merchants,
		CurrentIPs:  ips,
	}, nil
}

// ProjectSyncStatusResp 项目同步状态响应
type ProjectSyncStatusResp struct {
	ProjectId   int                     `json:"project_id"`
	ProjectName string                  `json:"project_name"`
	GostServers []ProjectGostServerResp `json:"gost_servers"`
	Merchants   []ProjectMerchantResp   `json:"merchants"`
	CurrentIPs  []string                `json:"current_ips"`
}

// GetProjectOptions 获取项目选项列表（用于下拉框）
func GetProjectOptions() ([]ProjectOptionResp, error) {
	var projects []entity.Projects
	err := dbs.DBAdmin.Where("status = 1").OrderBy("name ASC").Find(&projects)
	if err != nil {
		return nil, err
	}

	result := make([]ProjectOptionResp, len(projects))
	for i, p := range projects {
		result[i] = ProjectOptionResp{
			Value: p.Id,
			Label: p.Name,
		}
	}

	return result, nil
}

// ProjectOptionResp 项目选项响应
type ProjectOptionResp struct {
	Value int    `json:"value"`
	Label string `json:"label"`
}
