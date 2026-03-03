package ip_embed

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"server/internal/dbhelper"
	"server/internal/server/cfg"
	"server/internal/server/cloud/tencent"
	"server/internal/server/model"
	cloud_aliyun "server/internal/server/service/cloud_aliyun"
	cloud_aws "server/internal/server/service/cloud_aws"
	"server/internal/server/service/utils"
	"server/pkg/dbs"
	"server/pkg/entity"

	"github.com/google/uuid"
)

// IP选择记录的键名
const selectedIPsKey = "ip_embed_selected_ips"

// IP隐写使用的seed（与现有代码一致）
const ipEmbedSeed uint32 = 444013

// GetSystemIPs 获取系统服务器IP列表
func GetSystemIPs() (*model.GetSystemIPsResp, error) {
	var servers []entity.Servers
	err := dbs.DBAdmin.Where("server_type = ? AND status = 1", 2).Find(&servers)
	if err != nil {
		return nil, err
	}

	// 查询 merchant_gost_servers 关联表，补充 servers.merchant_id 为 0 的情况
	serverMerchantMap := make(map[int]int) // server_id -> merchant_id
	var relations []entity.MerchantGostServers
	_ = dbs.DBAdmin.Where("status = 1").Find(&relations)
	for _, r := range relations {
		serverMerchantMap[r.ServerId] = r.MerchantId
	}

	// 确定每台服务器关联的商户ID：优先 servers.merchant_id，其次关联表
	merchantIds := make([]int, 0)
	serverFinalMerchant := make(map[int]int) // server_id -> final merchant_id
	for _, s := range servers {
		mid := s.MerchantId
		if mid == 0 {
			mid = serverMerchantMap[s.Id]
		}
		serverFinalMerchant[s.Id] = mid
		if mid > 0 {
			merchantIds = append(merchantIds, mid)
		}
	}

	// 批量查询商户名称
	merchantMap := make(map[int]string)
	if len(merchantIds) > 0 {
		var merchants []entity.Merchants
		_ = dbs.DBAdmin.In("id", merchantIds).Cols("id", "name").Find(&merchants)
		for _, m := range merchants {
			merchantMap[m.Id] = m.Name
		}
	}

	items := make([]model.SystemIPItem, 0, len(servers))
	for _, s := range servers {
		mid := serverFinalMerchant[s.Id]
		items = append(items, model.SystemIPItem{
			ServerId:     s.Id,
			ServerName:   s.Name,
			IP:           s.Host,
			AuxiliaryIP:  s.AuxiliaryIP,
			Status:       s.Status,
			MerchantId:   mid,
			MerchantName: merchantMap[mid],
		})
	}

	return &model.GetSystemIPsResp{
		IPs:   items,
		Total: len(items),
	}, nil
}

// ========== 上传目标 CRUD ==========

// GetTargets 获取上传目标配置列表（从数据库读取）
func GetTargets() (*model.GetTargetsResp, error) {
	var targets []entity.IpEmbedTargets
	err := dbs.DBAdmin.OrderBy("sort_order ASC, id ASC").Find(&targets)
	if err != nil {
		return nil, err
	}

	// 批量查询分组名称
	groupIds := make([]int, 0)
	for _, t := range targets {
		if t.GroupId > 0 {
			groupIds = append(groupIds, t.GroupId)
		}
	}
	groupMap := make(map[int]string)
	if len(groupIds) > 0 {
		var groups []entity.ResourceGroups
		_ = dbs.DBAdmin.In("id", groupIds).Find(&groups)
		for _, g := range groups {
			groupMap[g.Id] = g.Name
		}
	}

	// 批量查询云账号信息（名称、商户ID）
	accountIds := make([]int64, 0)
	for _, t := range targets {
		if t.CloudAccountId > 0 {
			accountIds = append(accountIds, t.CloudAccountId)
		}
	}
	type accInfo struct {
		Name       string
		MerchantId int
	}
	accMap := make(map[int64]accInfo)
	if len(accountIds) > 0 {
		var accounts []entity.CloudAccounts
		_ = dbs.DBAdmin.In("id", accountIds).Cols("id", "name", "merchant_id").Find(&accounts)
		for _, a := range accounts {
			accMap[a.Id] = accInfo{Name: a.Name, MerchantId: a.MerchantId}
		}
	}

	// 批量查询商户名称
	merchantIds := make([]int, 0)
	for _, ai := range accMap {
		if ai.MerchantId > 0 {
			merchantIds = append(merchantIds, ai.MerchantId)
		}
	}
	merchantMap := make(map[int]string)
	if len(merchantIds) > 0 {
		var merchants []entity.Merchants
		_ = dbs.DBAdmin.In("id", merchantIds).Cols("id", "name").Find(&merchants)
		for _, m := range merchants {
			merchantMap[m.Id] = m.Name
		}
	}

	items := make([]model.TargetItem, 0, len(targets))
	for i, t := range targets {
		ai := accMap[t.CloudAccountId]

		items = append(items, model.TargetItem{
			Id:             t.Id,
			Index:          i, // 兼容旧逻辑
			Name:           t.Name,
			CloudType:      t.CloudType,
			CloudAccountId: t.CloudAccountId,
			AccountName:    ai.Name,
			RegionId:       t.RegionId,
			Bucket:         t.Bucket,
			ObjectPrefix:   t.ObjectPrefix,
			Enabled:        t.Enabled == 1,
			SortOrder:      t.SortOrder,
			GroupId:        t.GroupId,
			GroupName:      groupMap[t.GroupId],
			MerchantId:     ai.MerchantId,
			MerchantName:   merchantMap[ai.MerchantId],
		})
	}

	return &model.GetTargetsResp{Targets: items}, nil
}

// CreateTarget 创建上传目标
func CreateTarget(req model.CreateTargetReq) (int, error) {
	// 验证云账号存在
	acc, err := dbhelper.GetCloudAccountByID(req.CloudAccountId)
	if err != nil || acc == nil {
		return 0, fmt.Errorf("云账号不存在: %d", req.CloudAccountId)
	}

	// 验证云类型匹配
	if acc.CloudType != req.CloudType {
		return 0, fmt.Errorf("云账号类型不匹配: 账号类型=%s, 请求类型=%s", acc.CloudType, req.CloudType)
	}

	enabled := 0
	if req.Enabled {
		enabled = 1
	}

	target := &entity.IpEmbedTargets{
		Name:           req.Name,
		CloudType:      req.CloudType,
		CloudAccountId: req.CloudAccountId,
		RegionId:       req.RegionId,
		Bucket:         req.Bucket,
		ObjectPrefix:   req.ObjectPrefix,
		Enabled:        enabled,
		SortOrder:      req.SortOrder,
		GroupId:        req.GroupId,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	_, err = dbs.DBAdmin.Insert(target)
	if err != nil {
		return 0, fmt.Errorf("创建目标失败: %v", err)
	}

	return target.Id, nil
}

// UpdateTarget 更新上传目标
func UpdateTarget(id int, req model.UpdateTargetReq) error {
	// 检查目标是否存在
	var target entity.IpEmbedTargets
	has, err := dbs.DBAdmin.ID(id).Get(&target)
	if err != nil {
		return fmt.Errorf("查询目标失败: %v", err)
	}
	if !has {
		return fmt.Errorf("目标不存在: %d", id)
	}

	updates := make(map[string]interface{})
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.CloudType != nil {
		updates["cloud_type"] = *req.CloudType
	}
	if req.CloudAccountId != nil {
		// 验证云账号存在
		acc, err := dbhelper.GetCloudAccountByID(*req.CloudAccountId)
		if err != nil || acc == nil {
			return fmt.Errorf("云账号不存在: %d", *req.CloudAccountId)
		}
		updates["cloud_account_id"] = *req.CloudAccountId
	}
	if req.RegionId != nil {
		updates["region_id"] = *req.RegionId
	}
	if req.Bucket != nil {
		updates["bucket"] = *req.Bucket
	}
	if req.ObjectPrefix != nil {
		updates["object_prefix"] = *req.ObjectPrefix
	}
	if req.Enabled != nil {
		if *req.Enabled {
			updates["enabled"] = 1
		} else {
			updates["enabled"] = 0
		}
	}
	if req.SortOrder != nil {
		updates["sort_order"] = *req.SortOrder
	}
	if req.GroupId != nil {
		updates["group_id"] = *req.GroupId
	}

	if len(updates) == 0 {
		return nil
	}

	updates["updated_at"] = time.Now()
	_, err = dbs.DBAdmin.Table("ip_embed_targets").Where("id = ?", id).Update(updates)
	if err != nil {
		return fmt.Errorf("更新目标失败: %v", err)
	}

	return nil
}

// DeleteTarget 删除上传目标
func DeleteTarget(id int) error {
	_, err := dbs.DBAdmin.ID(id).Delete(&entity.IpEmbedTargets{})
	if err != nil {
		return fmt.Errorf("删除目标失败: %v", err)
	}
	return nil
}

// ToggleTarget 切换目标启用状态
func ToggleTarget(id int) error {
	var target entity.IpEmbedTargets
	has, err := dbs.DBAdmin.ID(id).Get(&target)
	if err != nil {
		return fmt.Errorf("查询目标失败: %v", err)
	}
	if !has {
		return fmt.Errorf("目标不存在: %d", id)
	}

	newEnabled := 1
	if target.Enabled == 1 {
		newEnabled = 0
	}

	_, err = dbs.DBAdmin.ID(id).Update(&entity.IpEmbedTargets{
		Enabled:   newEnabled,
		UpdatedAt: time.Now(),
	})
	if err != nil {
		return fmt.Errorf("更新状态失败: %v", err)
	}

	return nil
}

// ========== 源文件 ==========

// GetSourceFiles 获取源文件列表
func GetSourceFiles() (*model.GetSourceFilesResp, error) {
	if cfg.C.IpEmbed == nil || cfg.C.IpEmbed.SourceDir == "" {
		return &model.GetSourceFilesResp{
			Files:     []model.SourceFileItem{},
			Total:     0,
			SourceDir: "",
		}, nil
	}

	sourceDir := cfg.C.IpEmbed.SourceDir
	files := make([]model.SourceFileItem, 0)

	// 确保目录存在
	if _, err := os.Stat(sourceDir); os.IsNotExist(err) {
		return &model.GetSourceFilesResp{
			Files:     files,
			Total:     0,
			SourceDir: sourceDir,
		}, nil
	}

	err := filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// 跳过目录和隐藏文件
		if info.IsDir() || len(info.Name()) > 0 && info.Name()[0] == '.' {
			return nil
		}

		files = append(files, model.SourceFileItem{
			Name:    info.Name(),
			Size:    info.Size(),
			ModTime: info.ModTime().Format("2006-01-02 15:04:05"),
		})
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("读取源文件目录失败: %v", err)
	}

	return &model.GetSourceFilesResp{
		Files:     files,
		Total:     len(files),
		SourceDir: sourceDir,
	}, nil
}

// ========== 执行操作 ==========

// ExecuteEmbedAndUpload 执行嵌入并上传
func ExecuteEmbedAndUpload(req model.ExecuteEmbedReq) (*model.ExecuteEmbedResp, error) {
	startTime := time.Now()
	executionId := uuid.New().String()[:8]

	if cfg.C.IpEmbed == nil {
		return nil, fmt.Errorf("IP嵌入配置未设置")
	}

	// 1. 使用请求中直接传递的IP列表
	ips := req.IPs
	if len(ips) == 0 {
		return nil, fmt.Errorf("没有选中有效的IP")
	}

	// 保存使用的IP列表
	_ = SaveSelectedIPs(ips)

	// 2. 从数据库获取选中的目标
	var allTargets []entity.IpEmbedTargets
	err := dbs.DBAdmin.OrderBy("sort_order ASC, id ASC").Find(&allTargets)
	if err != nil {
		return nil, fmt.Errorf("获取目标列表失败: %v", err)
	}

	// 根据索引选择目标
	selectedTargets := make([]entity.IpEmbedTargets, 0)
	for _, idx := range req.TargetIndexes {
		if idx >= 0 && idx < len(allTargets) {
			selectedTargets = append(selectedTargets, allTargets[idx])
		}
	}
	if len(selectedTargets) == 0 {
		return nil, fmt.Errorf("没有选中有效的上传目标")
	}

	// 3. 获取源文件
	filesResp, err := GetSourceFiles()
	if err != nil {
		return nil, fmt.Errorf("获取源文件失败: %v", err)
	}

	// 过滤文件（如果指定了文件名）
	filesToProcess := filesResp.Files
	if len(req.FileNames) > 0 {
		fileNameSet := make(map[string]bool)
		for _, name := range req.FileNames {
			fileNameSet[name] = true
		}
		filtered := make([]model.SourceFileItem, 0)
		for _, f := range filesToProcess {
			if fileNameSet[f.Name] {
				filtered = append(filtered, f)
			}
		}
		filesToProcess = filtered
	}

	if len(filesToProcess) == 0 {
		return nil, fmt.Errorf("没有找到要处理的源文件")
	}

	// 4. 执行嵌入和上传
	results := make([]model.UploadResultItem, 0)
	successCount, failCount := 0, 0

	for _, file := range filesToProcess {
		srcPath := filepath.Join(cfg.C.IpEmbed.SourceDir, file.Name)

		// 读取原文件并嵌入IP（内存中处理）
		embeddedData, err := embedIPsToBytes(srcPath, ips)
		if err != nil {
			// 记录嵌入失败
			for _, target := range selectedTargets {
				results = append(results, model.UploadResultItem{
					FileName:   file.Name,
					TargetName: target.Name,
					CloudType:  target.CloudType,
					Bucket:     target.Bucket,
					Success:    false,
					Error:      fmt.Sprintf("嵌入IP失败: %v", err),
				})
				failCount++
			}
			continue
		}

		// 上传到各个目标
		for _, target := range selectedTargets {
			objectKey := target.ObjectPrefix + file.Name
			objectUrl, uploadErr := uploadToCloudByEntity(target, objectKey, embeddedData)

			result := model.UploadResultItem{
				FileName:   file.Name,
				TargetName: target.Name,
				CloudType:  target.CloudType,
				Bucket:     target.Bucket,
				ObjectKey:  objectKey,
			}

			if uploadErr != nil {
				result.Success = false
				result.Error = uploadErr.Error()
				failCount++
			} else {
				result.Success = true
				result.ObjectUrl = objectUrl
				successCount++
			}
			results = append(results, result)
		}
	}

	duration := time.Since(startTime)

	return &model.ExecuteEmbedResp{
		ExecutionId:  executionId,
		TotalFiles:   len(filesToProcess),
		TotalTargets: len(selectedTargets),
		Results:      results,
		Summary: model.ExecutionSummary{
			SuccessCount: successCount,
			FailCount:    failCount,
			TotalCount:   successCount + failCount,
			Duration:     duration.String(),
		},
	}, nil
}

// embedIPsToBytes 将IP嵌入到文件并返回字节数组（内存中处理）
func embedIPsToBytes(srcPath string, ips []string) ([]byte, error) {
	// 读取原文件
	srcBytes, err := os.ReadFile(srcPath)
	if err != nil {
		return nil, err
	}

	// 使用现有的嵌入逻辑
	return utils.EmbedIPsToBytes(srcBytes, ips, ipEmbedSeed)
}

// uploadToCloudByEntity 上传到云存储（使用数据库实体）
func uploadToCloudByEntity(target entity.IpEmbedTargets, objectKey string, data []byte) (string, error) {
	reader := bytes.NewReader(data)

	switch target.CloudType {
	case "aws":
		acc, err := dbhelper.GetCloudAccountByID(target.CloudAccountId)
		if err != nil {
			return "", fmt.Errorf("获取AWS账号失败: %v", err)
		}
		err = cloud_aws.UploadObject(acc, target.RegionId, target.Bucket, objectKey, reader)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", target.Bucket, target.RegionId, objectKey), nil

	case "aliyun":
		err := cloud_aliyun.UploadOssObject(0, target.CloudAccountId, target.RegionId, target.Bucket, objectKey, reader)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("https://%s.oss-%s.aliyuncs.com/%s", target.Bucket, target.RegionId, objectKey), nil

	case "tencent":
		err := tencent.UploadObject(0, target.CloudAccountId, target.RegionId, target.Bucket, objectKey, reader)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("https://%s.cos.%s.myqcloud.com/%s", target.Bucket, target.RegionId, objectKey), nil

	default:
		return "", fmt.Errorf("不支持的云类型: %s", target.CloudType)
	}
}

// ========== IP选择记录 ==========

// SaveSelectedIPs 保存选中的IP列表（覆盖保存）
func SaveSelectedIPs(ips []string) error {
	ipsJson, err := json.Marshal(ips)
	if err != nil {
		return fmt.Errorf("序列化IP列表失败: %v", err)
	}

	// 查询是否已存在
	var existing entity.IpEmbedSelections
	has, err := dbs.DBAdmin.Where("key_name = ?", selectedIPsKey).Get(&existing)
	if err != nil {
		return fmt.Errorf("查询记录失败: %v", err)
	}

	if has {
		// 更新
		existing.SelectedIPs = string(ipsJson)
		existing.UpdatedAt = time.Now()
		_, err = dbs.DBAdmin.ID(existing.Id).Cols("selected_ips", "updated_at").Update(&existing)
	} else {
		// 插入
		record := &entity.IpEmbedSelections{
			KeyName:     selectedIPsKey,
			SelectedIPs: string(ipsJson),
			UpdatedAt:   time.Now(),
		}
		_, err = dbs.DBAdmin.Insert(record)
	}

	if err != nil {
		return fmt.Errorf("保存记录失败: %v", err)
	}
	return nil
}

// GetSelectedIPs 获取选中的IP列表
func GetSelectedIPs() (*model.GetSelectedIPsResp, error) {
	var record entity.IpEmbedSelections
	has, err := dbs.DBAdmin.Where("key_name = ?", selectedIPsKey).Get(&record)
	if err != nil {
		return nil, fmt.Errorf("查询记录失败: %v", err)
	}

	resp := &model.GetSelectedIPsResp{
		IPs: []string{},
	}

	if has && record.SelectedIPs != "" {
		if err := json.Unmarshal([]byte(record.SelectedIPs), &resp.IPs); err != nil {
			return nil, fmt.Errorf("解析IP列表失败: %v", err)
		}
	}

	return resp, nil
}
