package model

// ========== 腾讯云 COS ==========

// CosListBucketsReq 列举 Bucket 请求
type CosListBucketsReq struct {
	MerchantId     int    `form:"merchant_id"`
	CloudAccountId int64  `form:"cloud_account_id"`
	RegionId       string `form:"region_id"` // 可选；为空则列出所有区域
}

// CosBucketItem Bucket 条目
type CosBucketItem struct {
	Name         string `json:"name"`
	Location     string `json:"location"`
	CreationDate string `json:"creation_date"`
}

// CosListBucketsResponse 列举 Bucket 响应
type CosListBucketsResponse struct {
	List  []CosBucketItem `json:"list"`
	Total int             `json:"total"`
}

// CosListObjectsReq 列举对象请求
type CosListObjectsReq struct {
	MerchantId     int    `form:"merchant_id"`
	CloudAccountId int64  `form:"cloud_account_id"`
	RegionId       string `form:"region_id" binding:"required"`
	Bucket         string `form:"bucket" binding:"required"`
	Prefix         string `form:"prefix"`
	Marker         string `form:"marker"`
	MaxKeys        int    `form:"max_keys"`
}

// CosObjectItem 对象条目
type CosObjectItem struct {
	Key          string `json:"key"`
	Size         int64  `json:"size"`
	ETag         string `json:"etag"`
	LastModified string `json:"last_modified"`
	StorageClass string `json:"storage_class"`
}

// CosListObjectsResponse 列举对象响应
type CosListObjectsResponse struct {
	List        []CosObjectItem `json:"list"`
	IsTruncated bool            `json:"is_truncated"`
	NextMarker  string          `json:"next_marker"`
	Total       int             `json:"total"`
}

// CosUploadForm 上传表单（multipart）
type CosUploadForm struct {
	MerchantId     int    `form:"merchant_id"`
	CloudAccountId int64  `form:"cloud_account_id"`
	RegionId       string `form:"region_id" binding:"required"`
	Bucket         string `form:"bucket" binding:"required"`
	ObjectKey      string `form:"object_key" binding:"required"`
}

// CosDownloadReq 下载请求
type CosDownloadReq struct {
	MerchantId     int    `form:"merchant_id" json:"merchant_id"`
	CloudAccountId int64  `form:"cloud_account_id" json:"cloud_account_id"`
	RegionId       string `form:"region_id" json:"region_id" binding:"required"`
	Bucket         string `form:"bucket" json:"bucket" binding:"required"`
	ObjectKey      string `form:"object_key" json:"object_key" binding:"required"`
	Filename       string `form:"filename" json:"filename"`
	Attachment     int    `form:"attachment" json:"attachment"`
}

// COS 创建 Bucket 请求
type CosCreateBucketReq struct {
	MerchantId     int    `json:"merchant_id"`
	CloudAccountId int64  `json:"cloud_account_id"`
	RegionId       string `json:"region_id" binding:"required"`
	Bucket         string `json:"bucket" binding:"required"` // 格式：bucketname-appid
}

// COS 删除 Bucket 请求
type CosDeleteBucketReq struct {
	MerchantId     int    `json:"merchant_id"`
	CloudAccountId int64  `json:"cloud_account_id"`
	RegionId       string `json:"region_id" binding:"required"`
	Bucket         string `json:"bucket" binding:"required"`
}

// COS 设置 Bucket 公开访问请求
type CosSetBucketPublicReq struct {
	MerchantId     int    `json:"merchant_id"`
	CloudAccountId int64  `json:"cloud_account_id"`
	RegionId       string `json:"region_id" binding:"required"`
	Bucket         string `json:"bucket" binding:"required"`
	Public         bool   `json:"public"` // true=公开读，false=私有
}

// COS 删除对象请求
type CosDeleteObjectReq struct {
	MerchantId     int    `json:"merchant_id"`
	CloudAccountId int64  `json:"cloud_account_id"`
	RegionId       string `json:"region_id" binding:"required"`
	Bucket         string `json:"bucket" binding:"required"`
	ObjectKey      string `json:"object_key" binding:"required"`
}
