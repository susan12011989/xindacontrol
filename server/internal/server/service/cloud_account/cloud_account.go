package cloud_account

import (
	"errors"
	"server/internal/server/model"
	"server/pkg/dbs"
	"server/pkg/entity"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

// QueryCloudAccounts 查询云账号列表
func QueryCloudAccounts(req model.QueryCloudAccountsReq) (model.QueryCloudAccountsResponse, error) {
	var resp model.QueryCloudAccountsResponse

	session := dbs.DBAdmin.Table("cloud_accounts")

	// 条件过滤
	if req.Name != "" {
		session = session.Where("name LIKE ?", "%"+req.Name+"%")
	}
	if req.CloudType != "" {
		session = session.Where("cloud_type = ?", req.CloudType)
	}
	if req.Status != nil {
		session = session.Where("status = ?", *req.Status)
	}
	if req.AccountType != "" {
		session = session.Where("account_type = ?", req.AccountType)
	}
	if req.MerchantId > 0 {
		session = session.Where("merchant_id = ?", req.MerchantId)
	}

	// 使用 FindAndCount 一次性完成计数和查询
	offset := (req.Page - 1) * req.Size
	var accounts []entity.CloudAccounts
	total, err := session.Desc("id").Limit(req.Size, offset).FindAndCount(&accounts)
	if err != nil {
		logx.Errorf("query cloud accounts err: %+v", err)
		return resp, err
	}
	resp.Total = int(total)

	// 收集商户ID，批量查询商户名称
	merchantIds := make([]int, 0)
	for _, a := range accounts {
		if a.MerchantId > 0 {
			merchantIds = append(merchantIds, a.MerchantId)
		}
	}
	merchantNameMap := make(map[int]string)
	if len(merchantIds) > 0 {
		var merchants []entity.Merchants
		_ = dbs.DBAdmin.In("id", merchantIds).Cols("id", "name").Find(&merchants)
		for _, m := range merchants {
			merchantNameMap[m.Id] = m.Name
		}
	}

	// 转换为响应格式
	for _, a := range accounts {
		siteType := a.SiteType
		if siteType == "" {
			siteType = "cn" // 默认国内站
		}
		resp.List = append(resp.List, model.CloudAccountResp{
			Id:              a.Id,
			Name:            a.Name,
			CloudType:       a.CloudType,
			SiteType:        siteType,
			AccessKeyId:     a.AccessKeyId,
			AccessKeySecret: a.AccessKeySecret,
			Description:     a.Description,
			Status:          a.Status,
			AccountType:     a.AccountType,
			MerchantId:      a.MerchantId,
			MerchantName:    merchantNameMap[a.MerchantId],
			CreatedAt:       a.CreatedAt.Format("2006-01-02 15:04:05"),
			UpdatedAt:       a.UpdatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	return resp, nil
}

// GetCloudAccountDetail 获取云账号详情
func GetCloudAccountDetail(id int64) (model.CloudAccountResp, error) {
	var resp model.CloudAccountResp
	var account entity.CloudAccounts

	has, err := dbs.DBAdmin.Where("id = ?", id).Get(&account)
	if err != nil {
		logx.Errorf("get cloud account err: %+v", err)
		return resp, err
	}
	if !has {
		return resp, errors.New("云账号不存在")
	}

	siteType := account.SiteType
	if siteType == "" {
		siteType = "cn" // 默认国内站
	}
	resp = model.CloudAccountResp{
		Id:              account.Id,
		Name:            account.Name,
		CloudType:       account.CloudType,
		SiteType:        siteType,
		AccessKeyId:     account.AccessKeyId,
		AccessKeySecret: account.AccessKeySecret,
		Description:     account.Description,
		Status:          account.Status,
		CreatedAt:       account.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:       account.UpdatedAt.Format("2006-01-02 15:04:05"),
	}

	return resp, nil
}

// CreateCloudAccount 创建云账号
func CreateCloudAccount(req model.CreateCloudAccountReq) (int64, error) {
	// 检查名称是否已存在
	count, err := dbs.DBAdmin.Where("name = ?", req.Name).Count(&entity.CloudAccounts{})
	if err != nil {
		logx.Errorf("check cloud account name err: %+v", err)
		return 0, err
	}
	if count > 0 {
		return 0, errors.New("账号名称已存在")
	}
	now := time.Now()
	siteType := req.SiteType
	if siteType == "" {
		siteType = "cn" // 默认国内站
	}
	accountType := "system"
	if req.MerchantId > 0 {
		accountType = "merchant"
	}
	account := entity.CloudAccounts{
		Name:            req.Name,
		AccountType:     accountType,
		MerchantId:      req.MerchantId,
		CloudType:       req.CloudType,
		SiteType:        siteType,
		AccessKeyId:     req.AccessKeyId,
		AccessKeySecret: req.AccessKeySecret,
		Description:     req.Description,
		Status:          1, // 默认启用
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	affected, err := dbs.DBAdmin.Insert(&account)
	if err != nil {
		logx.Errorf("create cloud account err: %+v", err)
		return 0, err
	}
	if affected == 0 {
		return 0, errors.New("创建失败")
	}

	return account.Id, nil
}

// UpdateCloudAccount 更新云账号
func UpdateCloudAccount(id int64, req model.UpdateCloudAccountReq) error {
	// 检查云账号是否存在
	has, err := dbs.DBAdmin.Where("id = ?", id).Exist(&entity.CloudAccounts{})
	if err != nil {
		logx.Errorf("check cloud account exist err: %+v", err)
		return err
	}
	if !has {
		return errors.New("云账号不存在")
	}

	updates := make(map[string]interface{})

	if req.Name != "" {
		// 检查名称是否与其他账号重复
		count, err := dbs.DBAdmin.Where("name = ? AND id != ?", req.Name, id).Count(&entity.CloudAccounts{})
		if err != nil {
			logx.Errorf("check cloud account name err: %+v", err)
			return err
		}
		if count > 0 {
			return errors.New("账号名称已存在")
		}
		updates["name"] = req.Name
	}
	if req.SiteType != "" {
		updates["site_type"] = req.SiteType
	}
	if req.AccessKeyId != "" {
		updates["access_key_id"] = req.AccessKeyId
	}
	if req.AccessKeySecret != "" {
		updates["access_key_secret"] = req.AccessKeySecret
	}
	if req.Description != "" {
		updates["description"] = req.Description
	}
	if req.Status != nil {
		updates["status"] = *req.Status
	}
	if req.MerchantId != nil {
		updates["merchant_id"] = *req.MerchantId
		if *req.MerchantId > 0 {
			updates["account_type"] = "merchant"
		} else {
			updates["account_type"] = "system"
		}
	}

	if len(updates) == 0 {
		return errors.New("没有需要更新的字段")
	}

	affected, err := dbs.DBAdmin.Table("cloud_accounts").
		Where("id = ?", id).
		Update(updates)
	if err != nil {
		logx.Errorf("update cloud account err: %+v", err)
		return err
	}
	if affected == 0 {
		return errors.New("更新失败")
	}

	return nil
}

// DeleteCloudAccount 删除云账号
func DeleteCloudAccount(id int64) error {
	affected, err := dbs.DBAdmin.Where("id = ?", id).Delete(&entity.CloudAccounts{})
	if err != nil {
		logx.Errorf("delete cloud account err: %+v", err)
		return err
	}
	if affected == 0 {
		return errors.New("云账号不存在")
	}

	return nil
}

// GetCloudAccountOptions 获取云账号选项列表（用于下拉框）
func GetCloudAccountOptions(cloudType string, merchantId int) ([]model.CloudAccountOption, error) {
	var accounts []entity.CloudAccounts
	session := dbs.DBAdmin.Where("status = ?", 1) // 只返回启用的账号

	if cloudType != "" {
		session = session.Where("cloud_type = ?", cloudType)
	}
	if merchantId > 0 {
		session = session.Where("merchant_id = ?", merchantId)
	}

	err := session.Cols("id", "name", "cloud_type").Find(&accounts)
	if err != nil {
		logx.Errorf("get cloud account options err: %+v", err)
		return nil, err
	}

	var options []model.CloudAccountOption
	for _, a := range accounts {
		options = append(options, model.CloudAccountOption{
			Value: a.Id,
			Label: a.Name,
			Type:  a.CloudType,
		})
	}

	return options, nil
}
