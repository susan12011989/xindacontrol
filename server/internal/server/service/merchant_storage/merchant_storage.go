package merchant_storage

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"server/internal/dbhelper"
	"server/internal/server/model"
	"server/internal/server/service/auth"
	"server/pkg/dbs"
	"server/pkg/entity"
	"server/pkg/gostapi"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

// QueryMerchantStorageConfigs 查询商户存储配置列表
func QueryMerchantStorageConfigs(req model.QueryMerchantStorageReq) (model.QueryMerchantStorageResponse, error) {
	var resp model.QueryMerchantStorageResponse

	session := dbs.DBAdmin.Table("merchant_storage_configs").Alias("msc")

	// 条件过滤
	if req.MerchantId > 0 {
		session = session.Where("msc.merchant_id = ?", req.MerchantId)
	}
	if req.StorageType != "" {
		session = session.Where("msc.storage_type = ?", req.StorageType)
	}
	if req.Status != nil {
		session = session.Where("msc.status = ?", *req.Status)
	}

	// 分页
	offset := (req.Page - 1) * req.Size
	var configs []entity.MerchantStorageConfigs
	total, err := session.Desc("msc.id").Limit(req.Size, offset).FindAndCount(&configs)
	if err != nil {
		logx.Errorf("query merchant storage configs err: %+v", err)
		return resp, err
	}
	resp.Total = int(total)

	// 获取商户信息
	merchantIds := make([]int, 0)
	for _, c := range configs {
		merchantIds = append(merchantIds, c.MerchantId)
	}
	merchantMap := getMerchantMap(merchantIds)

	// 转换为响应格式
	for _, c := range configs {
		merchant := merchantMap[c.MerchantId]
		lastPushAt := ""
		if !c.LastPushAt.IsZero() {
			lastPushAt = c.LastPushAt.Format("2006-01-02 15:04:05")
		}
		resp.List = append(resp.List, model.MerchantStorageResp{
			Id:              c.Id,
			MerchantId:      c.MerchantId,
			MerchantName:    merchant.Name,
			MerchantNo:      merchant.No,
			StorageType:     c.StorageType,
			Name:            c.Name,
			Endpoint:        c.Endpoint,
			Bucket:          c.Bucket,
			Region:          c.Region,
			AccessKeyId:     c.AccessKeyId,
			AccessKeySecret: maskSecret(c.AccessKeySecret),
			UploadUrl:       c.UploadUrl,
			DownloadUrl:     c.DownloadUrl,
			FileBaseUrl:     c.FileBaseUrl,
			BucketUrl:       c.BucketUrl,
			CustomDomain:    c.CustomDomain,
			IsDefault:       c.IsDefault,
			Status:          c.Status,
			LastPushAt:      lastPushAt,
			LastPushResult:  c.LastPushResult,
			CreatedAt:       c.CreatedAt.Format("2006-01-02 15:04:05"),
			UpdatedAt:       c.UpdatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	return resp, nil
}

// GetMerchantStorageDetail 获取存储配置详情
func GetMerchantStorageDetail(id int) (model.MerchantStorageResp, error) {
	var resp model.MerchantStorageResp
	var config entity.MerchantStorageConfigs

	has, err := dbs.DBAdmin.Where("id = ?", id).Get(&config)
	if err != nil {
		logx.Errorf("get merchant storage config err: %+v", err)
		return resp, err
	}
	if !has {
		return resp, errors.New("存储配置不存在")
	}

	merchant, _ := dbhelper.GetMerchantByID(config.MerchantId)
	lastPushAt := ""
	if !config.LastPushAt.IsZero() {
		lastPushAt = config.LastPushAt.Format("2006-01-02 15:04:05")
	}

	resp = model.MerchantStorageResp{
		Id:              config.Id,
		MerchantId:      config.MerchantId,
		MerchantName:    merchant.Name,
		MerchantNo:      merchant.No,
		StorageType:     config.StorageType,
		Name:            config.Name,
		Endpoint:        config.Endpoint,
		Bucket:          config.Bucket,
		Region:          config.Region,
		AccessKeyId:     config.AccessKeyId,
		AccessKeySecret: config.AccessKeySecret, // 详情页返回完整密钥
		UploadUrl:       config.UploadUrl,
		DownloadUrl:     config.DownloadUrl,
		FileBaseUrl:     config.FileBaseUrl,
		BucketUrl:       config.BucketUrl,
		CustomDomain:    config.CustomDomain,
		IsDefault:       config.IsDefault,
		Status:          config.Status,
		LastPushAt:      lastPushAt,
		LastPushResult:  config.LastPushResult,
		CreatedAt:       config.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:       config.UpdatedAt.Format("2006-01-02 15:04:05"),
	}

	return resp, nil
}

// CreateMerchantStorageConfig 创建存储配置
func CreateMerchantStorageConfig(req model.MerchantStorageReq) (int, error) {
	// 验证商户是否存在
	merchant, err := dbhelper.GetMerchantByID(req.MerchantId)
	if err != nil {
		return 0, errors.New("商户不存在")
	}
	if merchant.Id == 0 {
		return 0, errors.New("商户不存在")
	}

	// 验证存储类型
	if !isValidStorageType(req.StorageType) {
		return 0, errors.New("不支持的存储类型")
	}

	now := time.Now()
	config := entity.MerchantStorageConfigs{
		MerchantId:      req.MerchantId,
		StorageType:     req.StorageType,
		Name:            req.Name,
		Endpoint:        req.Endpoint,
		Bucket:          req.Bucket,
		Region:          req.Region,
		AccessKeyId:     req.AccessKeyId,
		AccessKeySecret: req.AccessKeySecret,
		UploadUrl:       req.UploadUrl,
		DownloadUrl:     req.DownloadUrl,
		FileBaseUrl:     req.FileBaseUrl,
		BucketUrl:       req.BucketUrl,
		CustomDomain:    req.CustomDomain,
		IsDefault:       req.IsDefault,
		Status:          1,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	// 如果设置为默认，先取消其他默认配置
	if req.IsDefault == 1 {
		_, err := dbs.DBAdmin.Table("merchant_storage_configs").
			Where("merchant_id = ?", req.MerchantId).
			Update(map[string]interface{}{"is_default": 0})
		if err != nil {
			logx.Errorf("clear default storage config err: %+v", err)
		}
	}

	affected, err := dbs.DBAdmin.Insert(&config)
	if err != nil {
		logx.Errorf("create merchant storage config err: %+v", err)
		return 0, err
	}
	if affected == 0 {
		return 0, errors.New("创建失败")
	}

	return config.Id, nil
}

// UpdateMerchantStorageConfig 更新存储配置
func UpdateMerchantStorageConfig(id int, req model.MerchantStorageReq) error {
	// 检查配置是否存在
	var config entity.MerchantStorageConfigs
	has, err := dbs.DBAdmin.Where("id = ?", id).Get(&config)
	if err != nil {
		logx.Errorf("check storage config exist err: %+v", err)
		return err
	}
	if !has {
		return errors.New("存储配置不存在")
	}

	updates := map[string]interface{}{
		"storage_type":  req.StorageType,
		"name":          req.Name,
		"endpoint":      req.Endpoint,
		"bucket":        req.Bucket,
		"region":        req.Region,
		"access_key_id": req.AccessKeyId,
		"upload_url":    req.UploadUrl,
		"download_url":  req.DownloadUrl,
		"file_base_url": req.FileBaseUrl,
		"bucket_url":    req.BucketUrl,
		"custom_domain": req.CustomDomain,
		"is_default":    req.IsDefault,
		"status":        req.Status,
		"updated_at":    time.Now(),
	}

	// 如果提供了新密钥，则更新
	if req.AccessKeySecret != "" {
		updates["access_key_secret"] = req.AccessKeySecret
	}

	// 如果设置为默认，先取消其他默认配置
	if req.IsDefault == 1 {
		_, err := dbs.DBAdmin.Table("merchant_storage_configs").
			Where("merchant_id = ? AND id != ?", config.MerchantId, id).
			Update(map[string]interface{}{"is_default": 0})
		if err != nil {
			logx.Errorf("clear default storage config err: %+v", err)
		}
	}

	affected, err := dbs.DBAdmin.Table("merchant_storage_configs").
		Where("id = ?", id).
		Update(updates)
	if err != nil {
		logx.Errorf("update merchant storage config err: %+v", err)
		return err
	}
	if affected == 0 {
		return errors.New("更新失败")
	}

	return nil
}

// DeleteMerchantStorageConfig 删除存储配置
func DeleteMerchantStorageConfig(id int) error {
	affected, err := dbs.DBAdmin.Where("id = ?", id).Delete(&entity.MerchantStorageConfigs{})
	if err != nil {
		logx.Errorf("delete merchant storage config err: %+v", err)
		return err
	}
	if affected == 0 {
		return errors.New("存储配置不存在")
	}

	return nil
}

// PushStorageConfig 推送存储配置到商户服务器
func PushStorageConfig(req model.PushStorageConfigReq, username string) (model.PushStorageResult, error) {
	var result model.PushStorageResult

	// 1. 验证 2FA
	user, err := dbhelper.GetSysUserByUsername(username)
	if err != nil {
		return result, errors.New("用户不存在")
	}
	if user.TwoFactorEnabled != 1 {
		return result, errors.New("请先启用 2FA")
	}
	if !auth.VerifyTwoFACode(user.TwoFactorSecret, req.TwoFACode) {
		return result, errors.New("2FA 验证码错误")
	}

	// 2. 获取存储配置
	var config entity.MerchantStorageConfigs
	has, err := dbs.DBAdmin.Where("id = ?", req.ConfigId).Get(&config)
	if err != nil || !has {
		return result, errors.New("存储配置不存在")
	}

	// 验证配置是否属于指定商户
	if config.MerchantId != req.MerchantId {
		return result, errors.New("配置不属于该商户")
	}

	// 3. 获取商户信息
	merchant, err := dbhelper.GetMerchantByID(req.MerchantId)
	if err != nil || merchant.Id == 0 {
		return result, errors.New("商户不存在")
	}

	// 4. 构建推送 payload
	payload := buildStoragePushPayload(&config)

	// 5. 调用商户服务器 API
	err = callMerchantStorageAPI(merchant, payload)
	pushResult := "成功"
	if err != nil {
		pushResult = fmt.Sprintf("失败: %s", err.Error())
		result.Success = false
		result.Message = err.Error()
	} else {
		result.Success = true
		result.Message = "配置已成功推送到商户服务器"
	}

	// 6. 更新推送结果
	dbs.DBAdmin.Table("merchant_storage_configs").
		Where("id = ?", req.ConfigId).
		Update(map[string]interface{}{
			"last_push_at":     time.Now(),
			"last_push_result": pushResult,
		})

	return result, err
}

// 构建推送 payload
func buildStoragePushPayload(config *entity.MerchantStorageConfigs) map[string]interface{} {
	return map[string]interface{}{
		"storage_type":      config.StorageType,
		"endpoint":          config.Endpoint,
		"bucket":            config.Bucket,
		"region":            config.Region,
		"access_key_id":     config.AccessKeyId,
		"access_key_secret": config.AccessKeySecret,
		"upload_url":        config.UploadUrl,
		"download_url":      config.DownloadUrl,
		"file_base_url":     config.FileBaseUrl,
		"bucket_url":        config.BucketUrl,
		"custom_domain":     config.CustomDomain,
	}
}

// 调用商户服务器存储配置 API
func callMerchantStorageAPI(merchant *entity.Merchants, payload map[string]interface{}) error {
	// 构建 URL（统一使用 API 端口 10002）
	url := fmt.Sprintf("http://%s:%d/v1/control/storage", merchant.ServerIP, gostapi.MerchantAppPortHTTP)

	// 序列化 payload
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("序列化配置失败: %w", err)
	}

	// 创建请求
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// 添加 Basic Auth (使用商户 no 作为用户名，control 作为密码)
	req.SetBasicAuth("control", merchant.No)

	// 发送请求
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("请求商户服务器失败: %w", err)
	}
	defer resp.Body.Close()

	// 检查响应
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("商户服务器返回错误 %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// 验证存储类型
func isValidStorageType(storageType string) bool {
	validTypes := []string{
		entity.StorageTypeMinio,
		entity.StorageTypeAliyunOSS,
		entity.StorageTypeAwsS3,
		entity.StorageTypeTencentCOS,
	}
	for _, t := range validTypes {
		if t == storageType {
			return true
		}
	}
	return false
}

// 脱敏处理密钥
func maskSecret(secret string) string {
	if len(secret) <= 8 {
		return "****"
	}
	return secret[:4] + "****" + secret[len(secret)-4:]
}

// 获取商户信息 Map
func getMerchantMap(merchantIds []int) map[int]*entity.Merchants {
	result := make(map[int]*entity.Merchants)
	if len(merchantIds) == 0 {
		return result
	}

	var merchants []entity.Merchants
	err := dbs.DBAdmin.In("id", merchantIds).Find(&merchants)
	if err != nil {
		logx.Errorf("get merchants err: %+v", err)
		return result
	}

	for i := range merchants {
		result[merchants[i].Id] = &merchants[i]
	}
	return result
}

// GetStorageTypeOptions 获取存储类型选项
func GetStorageTypeOptions() []model.StorageTypeOption {
	return model.GetStorageTypeOptions()
}
