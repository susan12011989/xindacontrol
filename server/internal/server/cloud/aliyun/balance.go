package aliyun

import (
	"errors"

	"github.com/zeromicro/go-zero/core/logx"
)

// BalanceInfo 阿里云账户余额详情
type BalanceInfo struct {
	AvailableAmount     string `json:"available_amount"`      // 可用额度（含信用额度）
	AvailableCashAmount string `json:"available_cash_amount"` // 现金余额（真实余额）
	CreditAmount        string `json:"credit_amount"`         // 信用额度
	Currency            string `json:"currency"`              // 币种
}

func Balance(merchantId int) (string, error) {
	info, err := BalanceDetail(merchantId)
	if err != nil {
		return "", err
	}
	return info.AvailableCashAmount, nil
}

// BalanceDetail 获取商户余额详情
func BalanceDetail(merchantId int) (*BalanceInfo, error) {
	cloud, err := GetMerchantCloud(merchantId)
	if err != nil {
		return nil, err
	}
	return queryBalanceDetail(cloud, "商户", merchantId)
}

// BalanceByCloudAccount 使用系统云账号ID查询阿里云账户余额
func BalanceByCloudAccount(cloudAccountId int64) (string, error) {
	info, err := BalanceDetailByCloudAccount(cloudAccountId)
	if err != nil {
		return "", err
	}
	return info.AvailableCashAmount, nil
}

// BalanceDetailByCloudAccount 使用系统云账号ID查询阿里云账户余额详情
func BalanceDetailByCloudAccount(cloudAccountId int64) (*BalanceInfo, error) {
	cloud, err := GetSystemCloudAccount(cloudAccountId)
	if err != nil {
		return nil, err
	}
	return queryBalanceDetail(cloud, "系统云账号", int(cloudAccountId))
}

func queryBalanceDetail(cloud *CloudAccountInfo, label string, id int) (*BalanceInfo, error) {
	client, err := NewOpenApiClient(cloud.AccessKey, cloud.AccessSecret, cloud.SiteType)
	if err != nil {
		return nil, err
	}
	res, err := client.QueryAccountBalance()
	if err != nil {
		logx.Errorf("获取%s%d余额失败1: %v", label, id, err)
		return nil, err
	}
	if *res.StatusCode != 200 {
		logx.Errorf("获取%s%d余额失败2: %v", label, id, res.String())
		return nil, errors.New(res.String())
	}
	info := &BalanceInfo{}
	data := res.Body.Data
	if data.AvailableAmount != nil {
		info.AvailableAmount = *data.AvailableAmount
	}
	if data.AvailableCashAmount != nil {
		info.AvailableCashAmount = *data.AvailableCashAmount
	}
	if data.CreditAmount != nil {
		info.CreditAmount = *data.CreditAmount
	}
	if data.Currency != nil {
		info.Currency = *data.Currency
	}
	logx.Infof("获取%s%d余额成功: 现金=%s, 可用额度=%s, 信用额度=%s",
		label, id, info.AvailableCashAmount, info.AvailableAmount, info.CreditAmount)
	return info, nil
}
