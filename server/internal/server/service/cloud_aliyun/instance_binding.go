package cloud_aliyun

import (
	"server/internal/server/model"
	"server/pkg/dbs"
	"server/pkg/entity"
	"time"
)

// BindInstanceMerchant 绑定云实例到商户（upsert）
func BindInstanceMerchant(req model.BindInstanceMerchantReq) error {
	cloudType := req.CloudType
	if cloudType == "" {
		cloudType = "aliyun"
	}

	// 查找已有绑定
	var existing entity.CloudInstanceBindings
	has, err := dbs.DBAdmin.Where("instance_id = ? AND cloud_type = ?", req.InstanceId, cloudType).Get(&existing)
	if err != nil {
		return err
	}

	if has {
		// 更新
		existing.MerchantId = req.MerchantId
		existing.RegionId = req.RegionId
		existing.UpdatedAt = time.Now()
		_, err = dbs.DBAdmin.Where("id = ?", existing.Id).Cols("merchant_id", "region_id", "updated_at").Update(&existing)
		return err
	}

	// 新建
	binding := entity.CloudInstanceBindings{
		InstanceId: req.InstanceId,
		RegionId:   req.RegionId,
		CloudType:  cloudType,
		MerchantId: req.MerchantId,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	_, err = dbs.DBAdmin.Insert(&binding)
	return err
}

// UnbindInstanceMerchant 解绑云实例的商户
func UnbindInstanceMerchant(instanceId string, cloudType string) error {
	if cloudType == "" {
		cloudType = "aliyun"
	}
	_, err := dbs.DBAdmin.Where("instance_id = ? AND cloud_type = ?", instanceId, cloudType).Delete(&entity.CloudInstanceBindings{})
	return err
}

// GetInstanceBindings 批量查询云实例的商户绑定信息
func GetInstanceBindings(instanceIds []string, cloudType string) (map[string]model.InstanceBindingResp, error) {
	if cloudType == "" {
		cloudType = "aliyun"
	}

	result := make(map[string]model.InstanceBindingResp)
	if len(instanceIds) == 0 {
		return result, nil
	}

	// 查询绑定关系
	var bindings []entity.CloudInstanceBindings
	err := dbs.DBAdmin.In("instance_id", instanceIds).Where("cloud_type = ?", cloudType).Find(&bindings)
	if err != nil {
		return nil, err
	}

	if len(bindings) == 0 {
		return result, nil
	}

	// 收集商户ID
	merchantIds := make([]int, 0, len(bindings))
	for _, b := range bindings {
		merchantIds = append(merchantIds, b.MerchantId)
	}

	// 批量查询商户信息
	var merchants []entity.Merchants
	err = dbs.DBAdmin.In("id", merchantIds).Find(&merchants)
	if err != nil {
		return nil, err
	}

	merchantMap := make(map[int]*entity.Merchants)
	for i := range merchants {
		merchantMap[merchants[i].Id] = &merchants[i]
	}

	// 组装结果
	for _, b := range bindings {
		resp := model.InstanceBindingResp{
			InstanceId: b.InstanceId,
			MerchantId: b.MerchantId,
		}
		if m, ok := merchantMap[b.MerchantId]; ok {
			resp.MerchantName = m.Name
			resp.MerchantNo = m.No
		}
		result[b.InstanceId] = resp
	}

	return result, nil
}
