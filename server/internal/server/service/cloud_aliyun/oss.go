package cloud_aliyun

import (
	"bytes"
	"io"
	"net/http"
	"path"
	"server/internal/server/cloud/aliyun"
	"server/internal/server/model"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
)

// ListOssObjects 列举 OSS 对象
func ListOssObjects(req model.OssListObjectsReq) (model.OssListObjectsResponse, error) {
	var resp model.OssListObjectsResponse
	bucket, err := aliyun.GetOssBucket(req.MerchantId, req.CloudAccountId, req.RegionId, req.Bucket)
	if err != nil {
		return resp, err
	}

	lsOpt := []oss.Option{}
	if req.Prefix != "" {
		lsOpt = append(lsOpt, oss.Prefix(req.Prefix))
	}
	if req.Marker != "" {
		lsOpt = append(lsOpt, oss.Marker(req.Marker))
	}
	if req.MaxKeys > 0 {
		lsOpt = append(lsOpt, oss.MaxKeys(req.MaxKeys))
	}

	res, err := bucket.ListObjects(lsOpt...)
	if err != nil {
		return resp, err
	}

	for _, obj := range res.Objects {
		resp.List = append(resp.List, model.OssObjectItem{
			Key:          obj.Key,
			Size:         obj.Size,
			ETag:         obj.ETag,
			LastModified: obj.LastModified.Format("2006-01-02 15:04:05"),
			StorageClass: obj.StorageClass,
		})
	}
	resp.IsTruncated = res.IsTruncated
	resp.NextMarker = res.NextMarker
	resp.Total = len(resp.List)
	return resp, nil
}

// UploadOssObject 上传对象（从 Reader）
func UploadOssObject(merchantId int, cloudAccountId int64, regionId, bucketName, objectKey string, r io.Reader) error {
	bucket, err := aliyun.GetOssBucket(merchantId, cloudAccountId, regionId, bucketName)
	if err != nil {
		return err
	}
	return bucket.PutObject(objectKey, r)
}

// DownloadOssObject 下载对象，返回内容与 ContentType、文件名
func DownloadOssObject(merchantId int, cloudAccountId int64, regionId, bucketName, objectKey string) (body []byte, contentType string, filename string, err error) {
	bucket, err := aliyun.GetOssBucket(merchantId, cloudAccountId, regionId, bucketName)
	if err != nil {
		return nil, "", "", err
	}
	reader, err := bucket.GetObject(objectKey)
	if err != nil {
		return nil, "", "", err
	}
	defer reader.Close()
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, "", "", err
	}

	// 尝试获取对象元信息确定 Content-Type
	head, err := bucket.GetObjectMeta(objectKey)
	if err == nil {
		contentType = head.Get("Content-Type")
	}
	if contentType == "" {
		contentType = http.DetectContentType(data)
	}
	filename = path.Base(objectKey)
	return data, contentType, filename, nil
}

// UploadOssObjectBytes 简便方法
func UploadOssObjectBytes(merchantId int, cloudAccountId int64, regionId, bucketName, objectKey string, content []byte) error {
	return UploadOssObject(merchantId, cloudAccountId, regionId, bucketName, objectKey, bytes.NewReader(content))
}

// DeleteOssObject 删除 OSS 对象
func DeleteOssObject(merchantId int, cloudAccountId int64, regionId, bucketName, objectKey string) error {
	bucket, err := aliyun.GetOssBucket(merchantId, cloudAccountId, regionId, bucketName)
	if err != nil {
		return err
	}
	return bucket.DeleteObject(objectKey)
}

// CreateBucket 创建 OSS Bucket
func CreateBucket(req model.OssCreateBucketReq) error {
	var (
		cloudCred *aliyun.CloudAccountInfo
		err       error
	)
	if req.CloudAccountId > 0 {
		cloudCred, err = aliyun.GetSystemCloudAccount(req.CloudAccountId)
	} else {
		cloudCred, err = aliyun.GetMerchantCloud(req.MerchantId)
	}
	if err != nil {
		return err
	}

	// 构建 endpoint
	region := req.RegionId
	if len(region) > 4 && region[:4] == "oss-" {
		region = region[4:]
	}
	endpoint := "oss-" + region + ".aliyuncs.com"
	client, err := oss.New(endpoint, cloudCred.AccessKey, cloudCred.AccessSecret)
	if err != nil {
		return err
	}

	// 设置存储类型
	storageClass := oss.StorageStandard
	if req.StorageClass != "" {
		switch req.StorageClass {
		case "IA":
			storageClass = oss.StorageIA
		case "Archive":
			storageClass = oss.StorageArchive
		}
	}

	return client.CreateBucket(req.Bucket, oss.StorageClass(storageClass))
}

// DeleteBucket 删除 OSS Bucket（Bucket必须为空）
func DeleteBucket(req model.OssDeleteBucketReq) error {
	var (
		cloudCred *aliyun.CloudAccountInfo
		err       error
	)
	if req.CloudAccountId > 0 {
		cloudCred, err = aliyun.GetSystemCloudAccount(req.CloudAccountId)
	} else {
		cloudCred, err = aliyun.GetMerchantCloud(req.MerchantId)
	}
	if err != nil {
		return err
	}

	// 构建 endpoint
	region := req.RegionId
	if len(region) > 4 && region[:4] == "oss-" {
		region = region[4:]
	}
	endpoint := "oss-" + region + ".aliyuncs.com"
	client, err := oss.New(endpoint, cloudCred.AccessKey, cloudCred.AccessSecret)
	if err != nil {
		return err
	}

	return client.DeleteBucket(req.Bucket)
}

// SetBucketPublicAccess 设置 OSS Bucket 公开访问权限
func SetBucketPublicAccess(req model.OssSetBucketPublicReq) error {
	var (
		cloudCred *aliyun.CloudAccountInfo
		err       error
	)
	if req.CloudAccountId > 0 {
		cloudCred, err = aliyun.GetSystemCloudAccount(req.CloudAccountId)
	} else {
		cloudCred, err = aliyun.GetMerchantCloud(req.MerchantId)
	}
	if err != nil {
		return err
	}

	// 构建 endpoint
	region := req.RegionId
	if len(region) > 4 && region[:4] == "oss-" {
		region = region[4:]
	}
	endpoint := "oss-" + region + ".aliyuncs.com"
	client, err := oss.New(endpoint, cloudCred.AccessKey, cloudCred.AccessSecret)
	if err != nil {
		return err
	}

	if req.Public {
		// 1. 先关闭 "阻止公共访问" 设置（使用原生 API）
		err = deleteBucketPublicAccessBlock(client, req.Bucket)
		if err != nil {
			// 如果没有设置过，删除会报错，忽略 404 错误
			if !isOssNotFoundError(err) {
				return err
			}
		}

		// 2. 设置 Bucket ACL 为公开读
		return client.SetBucketACL(req.Bucket, oss.ACLPublicRead)
	} else {
		// 1. 设置 Bucket ACL 为私有
		err = client.SetBucketACL(req.Bucket, oss.ACLPrivate)
		if err != nil {
			return err
		}

		// 2. 开启 "阻止公共访问"（使用原生 API）
		return putBucketPublicAccessBlock(client, req.Bucket, true)
	}
}

// deleteBucketPublicAccessBlock 删除 Bucket 的公共访问阻止配置（关闭阻止公共访问）
func deleteBucketPublicAccessBlock(client *oss.Client, bucketName string) error {
	params := map[string]interface{}{
		"publicAccessBlock": nil,
	}
	resp, err := client.Conn.Do("DELETE", bucketName, "", params, nil, nil, 0, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return oss.CheckRespCode(resp.StatusCode, []int{http.StatusNoContent, http.StatusOK})
}

// putBucketPublicAccessBlock 设置 Bucket 的公共访问阻止配置（开启阻止公共访问）
func putBucketPublicAccessBlock(client *oss.Client, bucketName string, blockPublicAccess bool) error {
	params := map[string]interface{}{
		"publicAccessBlock": nil,
	}
	xmlBody := `<?xml version="1.0" encoding="UTF-8"?>
<PublicAccessBlockConfiguration>
  <BlockPublicAccess>` + boolToString(blockPublicAccess) + `</BlockPublicAccess>
</PublicAccessBlockConfiguration>`

	buffer := bytes.NewBufferString(xmlBody)
	headers := map[string]string{
		"Content-Type": "application/xml",
	}
	resp, err := client.Conn.Do("PUT", bucketName, "", params, headers, buffer, 0, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return oss.CheckRespCode(resp.StatusCode, []int{http.StatusOK})
}

// boolToString 布尔值转字符串
func boolToString(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

// isOssNotFoundError 检查是否是 OSS 404 错误
func isOssNotFoundError(err error) bool {
	if ossErr, ok := err.(oss.ServiceError); ok {
		return ossErr.StatusCode == 404 || ossErr.Code == "NoSuchPublicAccessBlockConfiguration"
	}
	return false
}

// ListBuckets 列举所有 Buckets
func ListBuckets(req model.OssListBucketsReq) (model.OssListBucketsResponse, error) {
	var resp model.OssListBucketsResponse

	var (
		cloudCred *aliyun.CloudAccountInfo
		err       error
	)
	if req.CloudAccountId > 0 {
		cloudCred, err = aliyun.GetSystemCloudAccount(req.CloudAccountId)
	} else {
		cloudCred, err = aliyun.GetMerchantCloud(req.MerchantId)
	}
	if err != nil {
		return resp, err
	}

	// endpoint 若未给 region，采用公共域名；否则区域域名
	var endpoint string
	if req.RegionId == "" {
		endpoint = "oss.aliyuncs.com"
	} else {
		// 去掉可能存在的 oss- 前缀
		region := req.RegionId
		if len(region) > 4 && region[:4] == "oss-" {
			region = region[4:]
		}
		endpoint = "oss-" + region + ".aliyuncs.com"
	}
	client, err := oss.New(endpoint, cloudCred.AccessKey, cloudCred.AccessSecret)
	if err != nil {
		return resp, err
	}

	opts := []oss.Option{}
	if req.Prefix != "" {
		opts = append(opts, oss.Prefix(req.Prefix))
	}
	if req.Marker != "" {
		opts = append(opts, oss.Marker(req.Marker))
	}
	if req.MaxKeys > 0 {
		opts = append(opts, oss.MaxKeys(req.MaxKeys))
	}
	res, err := client.ListBuckets(opts...)
	if err != nil {
		return resp, err
	}
	for _, b := range res.Buckets {
		resp.List = append(resp.List, model.OssBucketItem{
			Name:         b.Name,
			Location:     b.Location,
			CreationDate: b.CreationDate.Format("2006-01-02 15:04:05"),
			StorageClass: b.StorageClass,
		})
	}
	resp.IsTruncated = res.IsTruncated
	resp.NextMarker = res.NextMarker
	resp.Total = len(resp.List)
	return resp, nil
}
