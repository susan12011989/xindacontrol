package aliyun

import (
	"errors"
	"fmt"
	"server/internal/dbhelper"

	bssopenapi20171214 "github.com/alibabacloud-go/bssopenapi-20171214/v6/client"
	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	ecs20140526 "github.com/alibabacloud-go/ecs-20140526/v6/client"
	"github.com/alibabacloud-go/tea/tea"
	vpc20160428 "github.com/alibabacloud-go/vpc-20160428/v6/client"
)

func NewEcsClient(accessKey, accessSecret, regionId string) (*ecs20140526.Client, error) {
	config := &openapi.Config{
		AccessKeyId:     tea.String(accessKey),
		AccessKeySecret: tea.String(accessSecret),
	}

	// endpoint 请参考 https://api.aliyun.com/product/Ecs
	config.Endpoint = tea.String(fmt.Sprintf("ecs.%s.aliyuncs.com", regionId))

	return ecs20140526.NewClient(config)
}

func NewVpcEcsClient(accessKey, accessSecret, regionId string) (*vpc20160428.Client, error) {
	config := &openapi.Config{
		AccessKeyId:     tea.String(accessKey),
		AccessKeySecret: tea.String(accessSecret),
	}
	// Endpoint 请参考 https://api.aliyun.com/product/Vpc
	config.Endpoint = tea.String(fmt.Sprintf("vpc.%s.aliyuncs.com", regionId))
	return vpc20160428.NewClient(config)
}

// NewOpenApiClient 创建账单查询客户端
// siteType: cn-国内站, intl-国际站
func NewOpenApiClient(accessKey, accessSecret, siteType string) (*bssopenapi20171214.Client, error) {
	config := &openapi.Config{
		AccessKeyId:     tea.String(accessKey),
		AccessKeySecret: tea.String(accessSecret),
	}
	// Endpoint 请参考 https://api.aliyun.com/product/BssOpenApi
	// 国内站使用 business.aliyuncs.com，国际站使用 business.ap-southeast-1.aliyuncs.com
	if siteType == "intl" {
		config.Endpoint = tea.String("business.ap-southeast-1.aliyuncs.com")
	} else {
		config.Endpoint = tea.String("business.aliyuncs.com")
	}
	return bssopenapi20171214.NewClient(config)
}

// CloudAccountInfo 云账号信息（包含站点类型）
type CloudAccountInfo struct {
	AccessKey    string
	AccessSecret string
	SiteType     string // cn-国内站, intl-国际站
}

// 废弃 Merchants.CloudAccount，改为从 CloudAccounts 表读取商户级阿里云账号
func GetMerchantCloud(merchantId int) (*CloudAccountInfo, error) {
	acc, err := dbhelper.GetCloudAccountByMerchantType(merchantId, "aliyun")
	if err != nil {
		return nil, err
	}
	siteType := acc.SiteType
	if siteType == "" {
		siteType = "cn" // 默认国内站
	}
	return &CloudAccountInfo{
		AccessKey:    acc.AccessKeyId,
		AccessSecret: acc.AccessKeySecret,
		SiteType:     siteType,
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
	siteType := account.SiteType
	if siteType == "" {
		siteType = "cn" // 默认国内站
	}
	return &CloudAccountInfo{
		AccessKey:    account.AccessKeyId,
		AccessSecret: account.AccessKeySecret,
		SiteType:     siteType,
	}, nil
}
