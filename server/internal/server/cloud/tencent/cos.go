package tencent

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"server/internal/server/model"

	"github.com/tencentyun/cos-go-sdk-v5"
)

// newCosClient 创建 COS 客户端
func newCosClient(cred *CloudAccountInfo, bucket, regionId string) *cos.Client {
	// 腾讯云 COS URL 格式: https://{bucket}.cos.{region}.myqcloud.com
	bucketURL, _ := url.Parse(fmt.Sprintf("https://%s.cos.%s.myqcloud.com", bucket, regionId))
	// Service URL（用于 ListBuckets 等操作）
	serviceURL, _ := url.Parse("https://service.cos.myqcloud.com")

	baseURL := &cos.BaseURL{
		BucketURL:  bucketURL,
		ServiceURL: serviceURL,
	}

	return cos.NewClient(baseURL, &http.Client{
		Transport: &cos.AuthorizationTransport{
			SecretID:  cred.AccessKey,
			SecretKey: cred.AccessSecret,
		},
	})
}

// newCosServiceClient 创建用于 Service 级别操作（如 ListBuckets）的客户端
func newCosServiceClient(cred *CloudAccountInfo) *cos.Client {
	serviceURL, _ := url.Parse("https://service.cos.myqcloud.com")
	baseURL := &cos.BaseURL{
		ServiceURL: serviceURL,
	}

	return cos.NewClient(baseURL, &http.Client{
		Transport: &cos.AuthorizationTransport{
			SecretID:  cred.AccessKey,
			SecretKey: cred.AccessSecret,
		},
	})
}

// ListBuckets 列举所有 Buckets
func ListBuckets(req model.CosListBucketsReq) (model.CosListBucketsResponse, error) {
	var resp model.CosListBucketsResponse

	cred, err := GetCloudCredentials(req.MerchantId, req.CloudAccountId)
	if err != nil {
		return resp, err
	}

	client := newCosServiceClient(cred)
	ctx := context.Background()

	result, _, err := client.Service.Get(ctx)
	if err != nil {
		return resp, err
	}

	for _, b := range result.Buckets {
		// 如果指定了 region，只返回该 region 的 bucket
		if req.RegionId != "" && b.Region != req.RegionId {
			continue
		}
		resp.List = append(resp.List, model.CosBucketItem{
			Name:         b.Name,
			Location:     b.Region,
			CreationDate: b.CreationDate,
		})
	}
	resp.Total = len(resp.List)
	return resp, nil
}

// ListObjects 列举对象
func ListObjects(req model.CosListObjectsReq) (model.CosListObjectsResponse, error) {
	var resp model.CosListObjectsResponse

	cred, err := GetCloudCredentials(req.MerchantId, req.CloudAccountId)
	if err != nil {
		return resp, err
	}

	client := newCosClient(cred, req.Bucket, req.RegionId)
	ctx := context.Background()

	opt := &cos.BucketGetOptions{
		Prefix:  req.Prefix,
		Marker:  req.Marker,
		MaxKeys: req.MaxKeys,
	}
	if opt.MaxKeys == 0 {
		opt.MaxKeys = 100
	}

	result, _, err := client.Bucket.Get(ctx, opt)
	if err != nil {
		return resp, err
	}

	for _, obj := range result.Contents {
		resp.List = append(resp.List, model.CosObjectItem{
			Key:          obj.Key,
			Size:         int64(obj.Size),
			ETag:         obj.ETag,
			LastModified: obj.LastModified,
			StorageClass: obj.StorageClass,
		})
	}
	resp.IsTruncated = result.IsTruncated
	resp.NextMarker = result.NextMarker
	resp.Total = len(resp.List)
	return resp, nil
}

// UploadObject 上传对象
func UploadObject(merchantId int, cloudAccountId int64, regionId, bucket, objectKey string, r io.Reader) error {
	cred, err := GetCloudCredentials(merchantId, cloudAccountId)
	if err != nil {
		return err
	}

	client := newCosClient(cred, bucket, regionId)
	ctx := context.Background()

	_, err = client.Object.Put(ctx, objectKey, r, nil)
	return err
}

// DownloadObject 下载对象
func DownloadObject(merchantId int, cloudAccountId int64, regionId, bucket, objectKey string) (body []byte, contentType string, filename string, err error) {
	cred, err := GetCloudCredentials(merchantId, cloudAccountId)
	if err != nil {
		return nil, "", "", err
	}

	client := newCosClient(cred, bucket, regionId)
	ctx := context.Background()

	resp, err := client.Object.Get(ctx, objectKey, nil)
	if err != nil {
		return nil, "", "", err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", "", err
	}

	// 获取 Content-Type
	contentType = resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = http.DetectContentType(data)
	}
	filename = path.Base(objectKey)
	return data, contentType, filename, nil
}

// DeleteObject 删除对象
func DeleteObject(merchantId int, cloudAccountId int64, regionId, bucket, objectKey string) error {
	cred, err := GetCloudCredentials(merchantId, cloudAccountId)
	if err != nil {
		return err
	}

	client := newCosClient(cred, bucket, regionId)
	ctx := context.Background()

	_, err = client.Object.Delete(ctx, objectKey)
	return err
}

// CreateBucket 创建 COS Bucket
func CreateBucket(merchantId int, cloudAccountId int64, regionId, bucket string) error {
	cred, err := GetCloudCredentials(merchantId, cloudAccountId)
	if err != nil {
		return err
	}

	client := newCosClient(cred, bucket, regionId)
	ctx := context.Background()

	_, err = client.Bucket.Put(ctx, nil)
	return err
}

// DeleteBucket 删除 COS Bucket（Bucket必须为空）
func DeleteBucket(merchantId int, cloudAccountId int64, regionId, bucket string) error {
	cred, err := GetCloudCredentials(merchantId, cloudAccountId)
	if err != nil {
		return err
	}

	client := newCosClient(cred, bucket, regionId)
	ctx := context.Background()

	_, err = client.Bucket.Delete(ctx)
	return err
}

// SetBucketPublicAccess 设置 COS Bucket 公开访问权限
func SetBucketPublicAccess(merchantId int, cloudAccountId int64, regionId, bucket string, public bool) error {
	cred, err := GetCloudCredentials(merchantId, cloudAccountId)
	if err != nil {
		return err
	}

	client := newCosClient(cred, bucket, regionId)
	ctx := context.Background()

	var acl string
	if public {
		acl = "public-read"
	} else {
		acl = "private"
	}

	opt := &cos.BucketPutACLOptions{
		Header: &cos.ACLHeaderOptions{
			XCosACL: acl,
		},
	}
	_, err = client.Bucket.PutACL(ctx, opt)
	return err
}
