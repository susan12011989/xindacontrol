package aliyun

import (
	"fmt"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
)

// NewOssClient 根据访问密钥创建 OSS 客户端
func NewOssClient(cloud *CloudAccountInfo, regionId string) (*oss.Client, error) {
	// 典型 endpoint: oss-cn-hangzhou.aliyuncs.com
	// 如果 regionId 已经包含 oss- 前缀（如从 Bucket.Location 获取），需要去掉
	region := regionId
	if len(region) > 4 && region[:4] == "oss-" {
		region = region[4:]
	}
	endpoint := fmt.Sprintf("oss-%s.aliyuncs.com", region)
	return oss.New(endpoint, cloud.AccessKey, cloud.AccessSecret)
}

// NewOssClientWithEndpoint 自定义Endpoint（支持CNAME）
func NewOssClientWithEndpoint(cloud *CloudAccountInfo, regionId, endpoint string, useCname bool) (*oss.Client, error) {
	ep := endpoint
	if ep == "" {
		ep = fmt.Sprintf("oss-%s.aliyuncs.com", regionId)
	}
	opts := []oss.ClientOption{}
	if useCname {
		opts = append(opts, oss.UseCname(true))
	}
	return oss.New(ep, cloud.AccessKey, cloud.AccessSecret, opts...)
}

// GetOssBucket 通过商户云账号或系统云账号获取 OSS Bucket 句柄
// 优先使用 cloudAccountId，其次 merchantId
func GetOssBucket(merchantId int, cloudAccountId int64, regionId, bucket string) (*oss.Bucket, error) {
	var (
		cred *CloudAccountInfo
		err  error
	)
	if cloudAccountId > 0 {
		cred, err = GetSystemCloudAccount(cloudAccountId)
		if err != nil {
			return nil, err
		}
	} else {
		cred, err = GetMerchantCloud(merchantId)
		if err != nil {
			return nil, err
		}
	}
	client, err := NewOssClient(cred, regionId)
	if err != nil {
		return nil, err
	}
	return client.Bucket(bucket)
}

// GetOssBucketWithEndpoint 获取带自定义Endpoint（如CNAME）的Bucket句柄
func GetOssBucketWithEndpoint(merchantId int, cloudAccountId int64, regionId, bucket, endpoint string, useCname bool) (*oss.Bucket, error) {
	var (
		cred *CloudAccountInfo
		err  error
	)
	if cloudAccountId > 0 {
		cred, err = GetSystemCloudAccount(cloudAccountId)
		if err != nil {
			return nil, err
		}
	} else {
		cred, err = GetMerchantCloud(merchantId)
		if err != nil {
			return nil, err
		}
	}
	client, err := NewOssClientWithEndpoint(cred, regionId, endpoint, useCname)
	if err != nil {
		return nil, err
	}
	return client.Bucket(bucket)
}
