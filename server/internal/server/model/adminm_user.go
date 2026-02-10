package model

// CreateAdminmUserReq 创建 adminm 登录账号
type CreateAdminmUserReq struct {
	MerchantNo       string `json:"merchant_no" binding:"required"` // 目标商户编号
	Username         string `json:"username" binding:"required"`
	Phone            string `json:"phone" binding:"required"` // 手机号
	Password         string `json:"password" binding:"required"`
	TwoFactorSecret  string `json:"two_factor_secret"`  // 可选
	TwoFactorEnabled *int   `json:"two_factor_enabled"` // 可选 0/1
}

// UpdateAdminmUserReq 更新 adminm 登录账号
type UpdateAdminmUserReq struct {
	MerchantNo       string  `json:"merchant_no" binding:"required"`
	TargetUsername   string  `json:"target_username" binding:"required"`
	Username         *string `json:"username"`           // 可修改
	Password         *string `json:"password"`           // 可修改
	TwoFactorSecret  *string `json:"two_factor_secret"`  // 可修改
	TwoFactorEnabled *int    `json:"two_factor_enabled"` // 可修改
	AllowedIPs       *string `json:"allowed_ips"`        // IP白名单（逗号分隔）
}

// DeleteAdminmUserReq 删除 adminm 登录账号
type DeleteAdminmUserReq struct {
	MerchantNo string `json:"merchant_no" binding:"required"`
	Username   string `json:"username" binding:"required"`
}
