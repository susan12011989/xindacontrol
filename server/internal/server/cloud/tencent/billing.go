package tencent

import (
	"errors"
	"fmt"

	billing "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/billing/v20180709"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	"github.com/zeromicro/go-zero/core/logx"
)

// BalanceInfo 腾讯云账户余额信息
type BalanceInfo struct {
	Balance          int64  `json:"balance"`           // 可用余额，单位：分
	BalanceYuan      string `json:"balance_yuan"`      // 可用余额，单位：元
	CashBalance      int64  `json:"cash_balance"`      // 现金余额，单位：分
	CashBalanceYuan  string `json:"cash_balance_yuan"` // 现金余额，单位：元
	IncomeBalance    int64  `json:"income_balance"`    // 收入余额，单位：分（代金券等）
	PresentBalance   int64  `json:"present_balance"`   // 赠送余额，单位：分
	FreezeBalance    int64  `json:"freeze_balance"`    // 冻结金额，单位：分
	OweBalance       int64  `json:"owe_balance"`       // 欠费金额，单位：分
	IsOverdue        bool   `json:"is_overdue"`        // 是否欠费
	IsOverdueBalance bool   `json:"is_overdue_balance"` // 余额是否小于0
}

// newBillingClient 创建腾讯云 Billing 客户端
func newBillingClient(accessKey, accessSecret string) (*billing.Client, error) {
	credential := common.NewCredential(accessKey, accessSecret)
	cpf := profile.NewClientProfile()
	cpf.HttpProfile.Endpoint = "billing.tencentcloudapi.com"
	return billing.NewClient(credential, "", cpf)
}

// GetAccountBalance 查询腾讯云账户余额
func GetAccountBalance(merchantId int, cloudAccountId int64) (*BalanceInfo, error) {
	acc, err := GetCloudCredentials(merchantId, cloudAccountId)
	if err != nil {
		return nil, err
	}

	client, err := newBillingClient(acc.AccessKey, acc.AccessSecret)
	if err != nil {
		logx.Errorf("创建腾讯云Billing客户端失败: %v", err)
		return nil, err
	}

	request := billing.NewDescribeAccountBalanceRequest()
	response, err := client.DescribeAccountBalance(request)
	if err != nil {
		logx.Errorf("查询腾讯云账户余额失败: %v", err)
		return nil, err
	}

	if response.Response == nil {
		return nil, errors.New("腾讯云返回空响应")
	}

	info := &BalanceInfo{}

	// 可用余额（单位：分）
	if response.Response.Balance != nil {
		info.Balance = *response.Response.Balance
		info.BalanceYuan = fmt.Sprintf("%.2f", float64(info.Balance)/100)
	}

	// 现金余额
	if response.Response.RealBalance != nil {
		info.CashBalance = int64(*response.Response.RealBalance)
		info.CashBalanceYuan = fmt.Sprintf("%.2f", float64(info.CashBalance)/100)
	}

	// 收入余额（代金券等）
	if response.Response.CashAccountBalance != nil {
		info.IncomeBalance = int64(*response.Response.CashAccountBalance)
	}

	// 赠送余额
	if response.Response.IncomeIntoAccountBalance != nil {
		info.PresentBalance = int64(*response.Response.IncomeIntoAccountBalance)
	}

	// 冻结金额
	if response.Response.FreezeAmount != nil {
		info.FreezeBalance = int64(*response.Response.FreezeAmount)
	}

	// 欠费金额
	if response.Response.OweAmount != nil {
		info.OweBalance = int64(*response.Response.OweAmount)
	}

	// 是否欠费
	if response.Response.IsAllowArrears != nil {
		info.IsOverdue = !*response.Response.IsAllowArrears
	}

	// 余额是否小于0
	info.IsOverdueBalance = info.Balance < 0

	logx.Infof("查询腾讯云账户余额成功: 可用余额=%s元, 现金余额=%s元",
		info.BalanceYuan, info.CashBalanceYuan)

	return info, nil
}

// GetAccountBalanceSimple 查询腾讯云账户余额（简化版，返回字符串）
func GetAccountBalanceSimple(merchantId int, cloudAccountId int64) (string, error) {
	info, err := GetAccountBalance(merchantId, cloudAccountId)
	if err != nil {
		return "", err
	}
	return info.BalanceYuan, nil
}
