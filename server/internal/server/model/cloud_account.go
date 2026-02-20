package model

// ========== 系统云账号管理 ==========

// 查询云账号请求
type QueryCloudAccountsReq struct {
	Pagination
	Name        string `json:"name" form:"name"`                 // 账号名称（模糊查询）
	CloudType   string `json:"cloud_type" form:"cloud_type"`     // 云类型
	Status      *int   `json:"status" form:"status"`             // 状态
	AccountType string `json:"account_type" form:"account_type"` // 账号类型: system, merchant
	MerchantId  int    `json:"merchant_id" form:"merchant_id"`   // 商户ID
}

// 创建云账号请求
type CreateCloudAccountReq struct {
	Name            string `json:"name" binding:"required"`
	CloudType       string `json:"cloud_type" binding:"required"`
	SiteType        string `json:"site_type"`                       // cn-国内站, intl-国际站，默认cn
	AccessKeyId     string `json:"access_key_id" binding:"required"`
	AccessKeySecret string `json:"access_key_secret" binding:"required"`
	Region          string `json:"region"`
	Description     string `json:"description"`
	MerchantId      int    `json:"merchant_id"`                     // 商户ID，0或空表示系统账号
}

// 更新云账号请求
type UpdateCloudAccountReq struct {
	Name            string `json:"name"`
	SiteType        string `json:"site_type"` // cn-国内站, intl-国际站
	AccessKeyId     string `json:"access_key_id"`
	AccessKeySecret string `json:"access_key_secret"`
	Description     string `json:"description"`
	Status          *int   `json:"status"`
	MerchantId      *int   `json:"merchant_id"` // 商户ID，0表示系统账号
}

// 云账号响应
type CloudAccountResp struct {
	Id              int64  `json:"id"`
	Name            string `json:"name"`
	CloudType       string `json:"cloud_type"`
	SiteType        string `json:"site_type"` // cn-国内站, intl-国际站
	AccessKeyId     string `json:"access_key_id"`
	AccessKeySecret string `json:"access_key_secret"`
	Description     string `json:"description"`
	Status          int    `json:"status"`
	AccountType     string `json:"account_type"`
	MerchantId      int    `json:"merchant_id"`
	MerchantName    string `json:"merchant_name"` // 商户名称
	CreatedAt       string `json:"created_at"`
	UpdatedAt       string `json:"updated_at"`
}

// 云账号列表响应
type QueryCloudAccountsResponse struct {
	List  []CloudAccountResp `json:"list"`
	Total int                `json:"total"`
}

// 云账号选项（用于下拉框）
type CloudAccountOption struct {
	Value int64  `json:"value"` // id
	Label string `json:"label"` // name
	Type  string `json:"type"`  // cloud_type
}
