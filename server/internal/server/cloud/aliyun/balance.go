package aliyun

import (
	"errors"

	"github.com/zeromicro/go-zero/core/logx"
)

func Balance(merchantId int) (string, error) {
	cloud, err := GetMerchantCloud(merchantId)
	if err != nil {
		return "", err
	}
	client, err := NewOpenApiClient(cloud.AccessKey, cloud.AccessSecret, cloud.SiteType)
	if err != nil {
		return "", err
	}
	res, err := client.QueryAccountBalance()
	if err != nil {
		logx.Errorf("获取商户%d余额失败1: %v", merchantId, err)
		return "", err
	}
	if *res.StatusCode != 200 {
		logx.Errorf("获取商户%d余额失败2: %v", merchantId, res.String())
		return "", errors.New(res.String())
	}
	logx.Infof("获取商户%d余额成功: %v", merchantId, *res.Body.Data.AvailableAmount)
	return *res.Body.Data.AvailableAmount, nil
}

// BalanceByCloudAccount 使用系统云账号ID查询阿里云账户余额
func BalanceByCloudAccount(cloudAccountId int64) (string, error) {
	cloud, err := GetSystemCloudAccount(cloudAccountId)
	if err != nil {
		return "", err
	}
	client, err := NewOpenApiClient(cloud.AccessKey, cloud.AccessSecret, cloud.SiteType)
	if err != nil {
		return "", err
	}
	res, err := client.QueryAccountBalance()
	if err != nil {
		logx.Errorf("获取系统云账号%d余额失败1: %v", cloudAccountId, err)
		return "", err
	}
	if *res.StatusCode != 200 {
		logx.Errorf("获取系统云账号%d余额失败2: %v", cloudAccountId, res.String())
		return "", errors.New(res.String())
	}
	logx.Infof("获取系统云账号%d余额成功: %v", cloudAccountId, *res.Body.Data.AvailableAmount)
	return *res.Body.Data.AvailableAmount, nil
}
