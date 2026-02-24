package model

// ========== 资源标签 ==========

type ResourceTagReq struct {
	Name        string `json:"name" binding:"required"`
	Color       string `json:"color"`
	Description string `json:"description"`
}

type ResourceTagResp struct {
	Id          int    `json:"id"`
	Name        string `json:"name"`
	Color       string `json:"color"`
	Description string `json:"description"`
	CreatedAt   string `json:"created_at"`
}

// ========== 标签分配 ==========

type AssignTagsReq struct {
	ResourceType string `json:"resource_type" binding:"required"` // oss_config / gost_server / storage_config
	ResourceIds  []int  `json:"resource_ids" binding:"required"`
	TagIds       []int  `json:"tag_ids" binding:"required"`
}

type RemoveTagsReq struct {
	ResourceType string `json:"resource_type" binding:"required"`
	ResourceIds  []int  `json:"resource_ids" binding:"required"`
	TagIds       []int  `json:"tag_ids" binding:"required"`
}

// ========== 全局 OSS 配置列表 ==========

type QueryGlobalOssConfigsReq struct {
	Pagination
	MerchantId int    `json:"merchant_id" form:"merchant_id"`
	CloudType  string `json:"cloud_type" form:"cloud_type"` // aliyun / aws / tencent
	Region     string `json:"region" form:"region"`
	TagId      int    `json:"tag_id" form:"tag_id"`
	Status     *int   `json:"status" form:"status"`
}

type GlobalOssConfigResp struct {
	Id               int               `json:"id"`
	MerchantId       int               `json:"merchant_id"`
	MerchantName     string            `json:"merchant_name"`
	MerchantNo       string            `json:"merchant_no"`
	CloudAccountId   int64             `json:"cloud_account_id"`
	CloudAccountName string            `json:"cloud_account_name"`
	CloudType        string            `json:"cloud_type"`
	Name             string            `json:"name"`
	Bucket           string            `json:"bucket"`
	Region           string            `json:"region"`
	Endpoint         string            `json:"endpoint"`
	CustomDomain     string            `json:"custom_domain"`
	DownloadUrl      string            `json:"download_url"`
	IsDefault        int               `json:"is_default"`
	Status           int               `json:"status"`
	Tags             []ResourceTagResp `json:"tags"`
	CreatedAt        string            `json:"created_at"`
	UpdatedAt        string            `json:"updated_at"`
}

type QueryGlobalOssConfigsResponse struct {
	List  []GlobalOssConfigResp `json:"list"`
	Total int                   `json:"total"`
}

// ========== 全局 GOST 服务器列表 ==========

type QueryGlobalGostServersReq struct {
	Pagination
	MerchantId int    `json:"merchant_id" form:"merchant_id"`
	CloudType  string `json:"cloud_type" form:"cloud_type"`
	Region     string `json:"region" form:"region"`
	TagId      int    `json:"tag_id" form:"tag_id"`
	Status     *int   `json:"status" form:"status"`
}

type GlobalGostServerResp struct {
	Id           int               `json:"id"`
	MerchantId   int               `json:"merchant_id"`
	MerchantName string            `json:"merchant_name"`
	MerchantNo   string            `json:"merchant_no"`
	ServerId     int               `json:"server_id"`
	ServerName   string            `json:"server_name"`
	ServerHost   string            `json:"server_host"`
	CloudType    string            `json:"cloud_type"`
	Region       string            `json:"region"`
	ListenPort   int               `json:"listen_port"`
	IsPrimary    int               `json:"is_primary"`
	Priority     int               `json:"priority"`
	Status       int               `json:"status"`
	Remark       string            `json:"remark"`
	Tags         []ResourceTagResp `json:"tags"`
	CreatedAt    string            `json:"created_at"`
	UpdatedAt    string            `json:"updated_at"`
}

type QueryGlobalGostServersResponse struct {
	List  []GlobalGostServerResp `json:"list"`
	Total int                    `json:"total"`
}

// ========== 批量操作 ==========

type BatchSyncGostIPByFilterReq struct {
	MerchantIds []int  `json:"merchant_ids"`
	CloudType   string `json:"cloud_type"`
	TagId       int    `json:"tag_id"`
}

// ========== OSS 健康检测 ==========

type CheckOssHealthReq struct {
	OssConfigIds []int `json:"oss_config_ids" binding:"required"`
}

type OssHealthCheckResult struct {
	OssConfigId   int    `json:"oss_config_id"`
	OssConfigName string `json:"oss_config_name"`
	MerchantName  string `json:"merchant_name"`
	CloudType     string `json:"cloud_type"`
	Bucket        string `json:"bucket"`
	DownloadUrl   string `json:"download_url"`
	CdnUrl        string `json:"cdn_url,omitempty"`
	Healthy       bool   `json:"healthy"`
	CdnHealthy    *bool  `json:"cdn_healthy,omitempty"`
	StatusCode    int    `json:"status_code"`
	CdnStatusCode *int   `json:"cdn_status_code,omitempty"`
	Message       string `json:"message"`
	Latency       string `json:"latency"`
}
