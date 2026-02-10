package model

// 查询商户存储配置请求
type QueryMerchantStorageReq struct {
	Pagination
	MerchantId  int    `json:"merchant_id" form:"merchant_id"`
	StorageType string `json:"storage_type" form:"storage_type"`
	Status      *int   `json:"status" form:"status"`
}

// 创建/更新商户存储配置请求
type MerchantStorageReq struct {
	MerchantId      int    `json:"merchant_id" binding:"required"`
	StorageType     string `json:"storage_type" binding:"required"`
	Name            string `json:"name" binding:"required"`
	Endpoint        string `json:"endpoint"`
	Bucket          string `json:"bucket" binding:"required"`
	Region          string `json:"region"`
	AccessKeyId     string `json:"access_key_id" binding:"required"`
	AccessKeySecret string `json:"access_key_secret"` // 更新时可选
	UploadUrl       string `json:"upload_url"`
	DownloadUrl     string `json:"download_url"`
	FileBaseUrl     string `json:"file_base_url"`
	BucketUrl       string `json:"bucket_url"`
	CustomDomain    string `json:"custom_domain"`
	IsDefault       int    `json:"is_default"`
	Status          int    `json:"status"`
}

// 推送存储配置请求
type PushStorageConfigReq struct {
	MerchantId int    `json:"merchant_id" binding:"required"`
	ConfigId   int    `json:"config_id" binding:"required"`
	TwoFACode  string `json:"twofa_code" binding:"required"`
}

// 测试存储连接请求
type TestStorageConnectionReq struct {
	StorageType     string `json:"storage_type" binding:"required"`
	Endpoint        string `json:"endpoint"`
	Bucket          string `json:"bucket" binding:"required"`
	Region          string `json:"region"`
	AccessKeyId     string `json:"access_key_id" binding:"required"`
	AccessKeySecret string `json:"access_key_secret" binding:"required"`
}

// 商户存储配置响应
type MerchantStorageResp struct {
	Id              int    `json:"id"`
	MerchantId      int    `json:"merchant_id"`
	MerchantName    string `json:"merchant_name"`
	MerchantNo      string `json:"merchant_no"`
	StorageType     string `json:"storage_type"`
	Name            string `json:"name"`
	Endpoint        string `json:"endpoint"`
	Bucket          string `json:"bucket"`
	Region          string `json:"region"`
	AccessKeyId     string `json:"access_key_id"`
	AccessKeySecret string `json:"access_key_secret"` // 脱敏显示
	UploadUrl       string `json:"upload_url"`
	DownloadUrl     string `json:"download_url"`
	FileBaseUrl     string `json:"file_base_url"`
	BucketUrl       string `json:"bucket_url"`
	CustomDomain    string `json:"custom_domain"`
	IsDefault       int    `json:"is_default"`
	Status          int    `json:"status"`
	LastPushAt      string `json:"last_push_at"`
	LastPushResult  string `json:"last_push_result"`
	CreatedAt       string `json:"created_at"`
	UpdatedAt       string `json:"updated_at"`
}

// 推送结果
type PushStorageResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// 查询响应
type QueryMerchantStorageResponse struct {
	List  []MerchantStorageResp `json:"list"`
	Total int                   `json:"total"`
}

// 存储类型选项
type StorageTypeOption struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

// GetStorageTypeOptions 获取存储类型选项列表
func GetStorageTypeOptions() []StorageTypeOption {
	return []StorageTypeOption{
		{Value: "minio", Label: "MinIO"},
		{Value: "aliyunOSS", Label: "阿里云 OSS"},
		{Value: "aws_s3", Label: "AWS S3"},
		{Value: "tencent_cos", Label: "腾讯云 COS"},
	}
}
