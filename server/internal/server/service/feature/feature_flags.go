package feature

import (
	"fmt"
	"server/internal/server/model"
	"server/pkg/dbs"
	"server/pkg/entity"
	"time"
)

// GetFeatureFlags 获取商户的功能开关列表
func GetFeatureFlags(req model.QueryFeatureFlagsReq) (model.QueryFeatureFlagsResponse, error) {
	var resp model.QueryFeatureFlagsResponse
	resp.MerchantId = req.MerchantId

	// 验证商户是否存在
	var merchant entity.Merchants
	has, err := dbs.DBAdmin.Where("id = ?", req.MerchantId).Get(&merchant)
	if err != nil {
		return resp, fmt.Errorf("查询商户失败: %v", err)
	}
	if !has {
		return resp, fmt.Errorf("商户不存在")
	}

	// 查询已有的功能开关配置
	var existingFlags []entity.FeatureFlags
	err = dbs.DBAdmin.Where("merchant_id = ?", req.MerchantId).Find(&existingFlags)
	if err != nil {
		return resp, fmt.Errorf("查询功能开关失败: %v", err)
	}

	// 构建已存在功能的映射
	existingMap := make(map[string]entity.FeatureFlags)
	for _, flag := range existingFlags {
		existingMap[flag.FeatureName] = flag
	}

	// 遍历所有可用功能，合并已有配置
	for _, feature := range model.AvailableFeatures {
		flagResp := model.FeatureFlagResp{
			MerchantId:  req.MerchantId,
			FeatureName: feature.Name,
			Label:       feature.Label,
			Description: feature.Description,
			Category:    feature.Category,
			Enabled:     true, // 默认启用
		}

		if existing, ok := existingMap[feature.Name]; ok {
			flagResp.Id = existing.Id
			flagResp.Enabled = existing.Enabled == 1
			flagResp.CreatedAt = existing.CreatedAt.Format("2006-01-02 15:04:05")
			flagResp.UpdatedAt = existing.UpdatedAt.Format("2006-01-02 15:04:05")
		}

		resp.List = append(resp.List, flagResp)
	}

	return resp, nil
}

// UpdateFeatureFlag 更新单个功能开关
func UpdateFeatureFlag(req model.UpdateFeatureFlagReq) error {
	// 验证功能名称是否有效
	valid := false
	for _, f := range model.AvailableFeatures {
		if f.Name == req.FeatureName {
			valid = true
			break
		}
	}
	if !valid {
		return fmt.Errorf("无效的功能名称: %s", req.FeatureName)
	}

	// 验证商户是否存在
	var merchant entity.Merchants
	has, err := dbs.DBAdmin.Where("id = ?", req.MerchantId).Get(&merchant)
	if err != nil {
		return fmt.Errorf("查询商户失败: %v", err)
	}
	if !has {
		return fmt.Errorf("商户不存在")
	}

	// 查询是否已有记录
	var existingFlag entity.FeatureFlags
	has, err = dbs.DBAdmin.Where("merchant_id = ? AND feature_name = ?", req.MerchantId, req.FeatureName).Get(&existingFlag)
	if err != nil {
		return fmt.Errorf("查询功能开关失败: %v", err)
	}

	enabled := 0
	if req.Enabled {
		enabled = 1
	}

	if has {
		// 更新已有记录
		existingFlag.Enabled = enabled
		existingFlag.UpdatedAt = time.Now()
		_, err = dbs.DBAdmin.Where("id = ?", existingFlag.Id).Cols("enabled", "updated_at").Update(&existingFlag)
		if err != nil {
			return fmt.Errorf("更新功能开关失败: %v", err)
		}
	} else {
		// 插入新记录
		newFlag := entity.FeatureFlags{
			MerchantId:  req.MerchantId,
			FeatureName: req.FeatureName,
			Enabled:     enabled,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		_, err = dbs.DBAdmin.Insert(&newFlag)
		if err != nil {
			return fmt.Errorf("创建功能开关失败: %v", err)
		}
	}

	return nil
}

// BatchUpdateFeatureFlags 批量更新功能开关
func BatchUpdateFeatureFlags(req model.BatchUpdateFeatureFlagsReq) error {
	// 验证商户是否存在
	var merchant entity.Merchants
	has, err := dbs.DBAdmin.Where("id = ?", req.MerchantId).Get(&merchant)
	if err != nil {
		return fmt.Errorf("查询商户失败: %v", err)
	}
	if !has {
		return fmt.Errorf("商户不存在")
	}

	for _, item := range req.Features {
		updateReq := model.UpdateFeatureFlagReq{
			MerchantId:  req.MerchantId,
			FeatureName: item.FeatureName,
			Enabled:     item.Enabled,
		}
		if err := UpdateFeatureFlag(updateReq); err != nil {
			return err
		}
	}

	return nil
}

// InitFeatureFlags 初始化商户的所有功能开关（全部启用）
func InitFeatureFlags(req model.InitFeatureFlagsReq) error {
	// 验证商户是否存在
	var merchant entity.Merchants
	has, err := dbs.DBAdmin.Where("id = ?", req.MerchantId).Get(&merchant)
	if err != nil {
		return fmt.Errorf("查询商户失败: %v", err)
	}
	if !has {
		return fmt.Errorf("商户不存在")
	}

	for _, feature := range model.AvailableFeatures {
		// 检查是否已存在
		var existing entity.FeatureFlags
		has, err := dbs.DBAdmin.Where("merchant_id = ? AND feature_name = ?", req.MerchantId, feature.Name).Get(&existing)
		if err != nil {
			return fmt.Errorf("查询功能开关失败: %v", err)
		}

		if !has {
			// 不存在则创建
			newFlag := entity.FeatureFlags{
				MerchantId:  req.MerchantId,
				FeatureName: feature.Name,
				Enabled:     1, // 默认启用
				Description: feature.Description,
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			}
			_, err = dbs.DBAdmin.Insert(&newFlag)
			if err != nil {
				return fmt.Errorf("创建功能开关失败: %v", err)
			}
		}
	}

	return nil
}

// CheckFeatureEnabled 检查商户的某个功能是否启用（供主服务调用）
func CheckFeatureEnabled(merchantId int, featureName string) (bool, error) {
	var flag entity.FeatureFlags
	has, err := dbs.DBAdmin.Where("merchant_id = ? AND feature_name = ?", merchantId, featureName).Get(&flag)
	if err != nil {
		return false, fmt.Errorf("查询功能开关失败: %v", err)
	}

	// 如果没有记录，默认启用
	if !has {
		return true, nil
	}

	return flag.Enabled == 1, nil
}

// GetAllFeatureDefinitions 获取所有可用功能定义
func GetAllFeatureDefinitions() []model.FeatureDefinition {
	return model.AvailableFeatures
}

// GetFeatureFlagsMap 获取商户的功能开关映射（供主服务调用）
func GetFeatureFlagsMap(merchantId int) (map[string]bool, error) {
	result := make(map[string]bool)

	// 默认所有功能都启用
	for _, f := range model.AvailableFeatures {
		result[f.Name] = true
	}

	// 查询已有的功能开关配置
	var existingFlags []entity.FeatureFlags
	err := dbs.DBAdmin.Where("merchant_id = ?", merchantId).Find(&existingFlags)
	if err != nil {
		return nil, fmt.Errorf("查询功能开关失败: %v", err)
	}

	// 覆盖已配置的功能状态
	for _, flag := range existingFlags {
		result[flag.FeatureName] = flag.Enabled == 1
	}

	return result, nil
}
