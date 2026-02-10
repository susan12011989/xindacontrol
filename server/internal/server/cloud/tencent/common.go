package tencent

import (
	"errors"
	"server/internal/dbhelper"
)

// CloudAccountInfo 腾讯云账号信息
type CloudAccountInfo struct {
	AccessKey    string
	AccessSecret string
}

// GetMerchantCloud 获取商户级腾讯云账号
func GetMerchantCloud(merchantId int) (*CloudAccountInfo, error) {
	acc, err := dbhelper.GetCloudAccountByMerchantType(merchantId, "tencent")
	if err != nil {
		return nil, err
	}
	return &CloudAccountInfo{
		AccessKey:    acc.AccessKeyId,
		AccessSecret: acc.AccessKeySecret,
	}, nil
}

// GetSystemCloudAccount 获取系统云账号信息
func GetSystemCloudAccount(cloudAccountId int64) (*CloudAccountInfo, error) {
	account, err := dbhelper.GetCloudAccountByID(cloudAccountId)
	if err != nil {
		return nil, err
	}
	if account.Status != 1 {
		return nil, errors.New("cloud account is disabled")
	}
	return &CloudAccountInfo{
		AccessKey:    account.AccessKeyId,
		AccessSecret: account.AccessKeySecret,
	}, nil
}

// GetCloudCredentials 根据参数获取云账号凭证
func GetCloudCredentials(merchantId int, cloudAccountId int64) (*CloudAccountInfo, error) {
	if cloudAccountId > 0 {
		return GetSystemCloudAccount(cloudAccountId)
	}
	return GetMerchantCloud(merchantId)
}
