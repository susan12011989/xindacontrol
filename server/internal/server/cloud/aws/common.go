package awscloud

import (
	"context"
	"errors"
	"fmt"
	"server/internal/dbhelper"
	"server/pkg/entity"

	awsv2 "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/costexplorer"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
)

// getCredsByMerchant 获取商户的 AWS 凭证
// 废弃旧字段读取，改为从 CloudAccounts 表读取商户级账号
func getCredsByMerchant(merchantId int) (*entity.CloudAccounts, error) {
	acc, err := dbhelper.GetCloudAccountByMerchantType(merchantId, "aws")
	if err != nil {
		return nil, err
	}
	if acc.AccessKeyId == "" || acc.AccessKeySecret == "" {
		return nil, errors.New("云账号凭证有问题,请先到云账号管理处修改")
	}
	return acc, nil
}

// getCredsBySystem 获取系统云账号的 AWS 凭证
func getCredsBySystem(cloudAccountId int64) (*entity.CloudAccounts, error) {
	acc, err := dbhelper.GetCloudAccountByID(cloudAccountId)
	if err != nil {
		return nil, err
	}
	if acc.Status != 1 {
		return nil, errors.New("cloud account is disabled")
	}
	if acc.CloudType != "aws" {
		return nil, fmt.Errorf("cloud account %d not aws", cloudAccountId)
	}
	if acc.AccessKeyId == "" || acc.AccessKeySecret == "" {
		return nil, errors.New("云账号凭证有问题,请先到云账号管理处修改")
	}
	return acc, nil
}

// loadAwsConfig 加载 AWS 配置
func loadAwsConfig(ctx context.Context, accessKey, secretKey, region string) (awsv2.Config, error) {
	if accessKey != "" && secretKey != "" {
		// 使用静态凭证
		return config.LoadDefaultConfig(
			ctx,
			config.WithRegion(region),
			config.WithCredentialsProvider(awsv2.NewCredentialsCache(credentials.NewStaticCredentialsProvider(accessKey, secretKey, ""))),
		)
	}
	// 回退到默认链路（环境变量/配置文件/IMDS等）
	return config.LoadDefaultConfig(ctx, config.WithRegion(region))
}

func NewEc2ClientBySystem(ctx context.Context, cloudAccountId int64, region string) (*ec2.Client, error) {
	// 向后兼容：委托统一方法
	acc, err := getCredsBySystem(cloudAccountId)
	if err != nil {
		return nil, err
	}
	return NewEc2Client(ctx, acc, region)
}

func NewEc2ClientByMerchant(ctx context.Context, merchantId int, region string) (*ec2.Client, error) {
	// 直接读取商户 CloudAccounts
	acc, err := getCredsByMerchant(merchantId)
	if err != nil {
		return nil, err
	}
	return NewEc2Client(ctx, acc, region)
}

func NewS3ClientBySystem(ctx context.Context, cloudAccountId int64, region string) (*s3.Client, error) {
	// 向后兼容：委托统一方法
	acc, err := getCredsBySystem(cloudAccountId)
	if err != nil {
		return nil, err
	}
	return NewS3Client(ctx, acc, region)
}

func NewS3ClientByMerchant(ctx context.Context, merchantId int, region string) (*s3.Client, error) {
	// 直接读取商户 CloudAccounts
	acc, err := getCredsByMerchant(merchantId)
	if err != nil {
		return nil, err
	}
	return NewS3Client(ctx, acc, region)
}

// ResolveAwsAccount 统一根据 cloud_account_id 或 merchant_id 获取可用的 AWS 账号信息
func ResolveAwsAccount(ctx context.Context, merchantId int, cloudAccountId int64) (*entity.CloudAccounts, error) {
	if cloudAccountId > 0 && merchantId > 0 {
		return nil, errors.New("cloud_account_id 与 merchant_id 不能同时传递")
	}
	if cloudAccountId > 0 {
		acc, err := dbhelper.GetCloudAccountByID(cloudAccountId)
		if err != nil {
			return nil, err
		}
		if acc.AccessKeyId == "" || acc.AccessKeySecret == "" {
			return nil, errors.New("云账号凭证有问题,请先到云账号管理处修改")
		}
		return acc, nil
	} else if merchantId > 0 {
		acc, err := getCredsByMerchant(merchantId)
		if err != nil {
			return nil, err
		}
		if acc.AccessKeyId == "" || acc.AccessKeySecret == "" {
			return nil, errors.New("云账号凭证有问题,请先到云账号管理处修改")
		}
		return acc, nil
	}
	return nil, errors.New("merchant_id 或 cloud_account_id 必须提供一个")
}

// NewEc2Client 使用统一账号信息创建 EC2 客户端
func NewEc2Client(ctx context.Context, acc *entity.CloudAccounts, region string) (*ec2.Client, error) {
	if acc == nil || acc.CloudType != "aws" {
		return nil, errors.New("invalid aws account")
	}
	cfg, err := loadAwsConfig(ctx, acc.AccessKeyId, acc.AccessKeySecret, region)
	if err != nil {
		return nil, err
	}
	return ec2.NewFromConfig(cfg), nil
}

// NewS3Client 使用统一账号信息创建 S3 客户端
func NewS3Client(ctx context.Context, acc *entity.CloudAccounts, region string) (*s3.Client, error) {
	if acc == nil || acc.CloudType != "aws" {
		return nil, errors.New("invalid aws account")
	}
	cfg, err := loadAwsConfig(ctx, acc.AccessKeyId, acc.AccessKeySecret, region)
	if err != nil {
		return nil, err
	}
	return s3.NewFromConfig(cfg), nil
}

// NewSsmClient 使用统一账号信息创建 SSM 客户端
func NewSsmClient(ctx context.Context, acc *entity.CloudAccounts, region string) (*ssm.Client, error) {
	if acc == nil || acc.CloudType != "aws" {
		return nil, errors.New("invalid aws account")
	}
	cfg, err := loadAwsConfig(ctx, acc.AccessKeyId, acc.AccessKeySecret, region)
	if err != nil {
		return nil, err
	}
	return ssm.NewFromConfig(cfg), nil
}

// NewCeClient 使用统一账号信息创建 Cost Explorer 客户端
// Cost Explorer 服务区域固定为 us-east-1（官方要求），如未指定区域则默认使用该区域
func NewCeClient(ctx context.Context, acc *entity.CloudAccounts, region string) (*costexplorer.Client, error) {
	if acc == nil || acc.CloudType != "aws" {
		return nil, errors.New("invalid aws account")
	}
	if region == "" {
		region = "us-east-1"
	}
	cfg, err := loadAwsConfig(ctx, acc.AccessKeyId, acc.AccessKeySecret, region)
	if err != nil {
		return nil, err
	}
	return costexplorer.NewFromConfig(cfg), nil
}
