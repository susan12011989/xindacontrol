package model

// TwoFASetupReq 设置2FA请求
type TwoFASetupReq struct {
	Code string `json:"code" binding:"required"` // 6位TOTP验证码，用于验证绑定
}

// TwoFASetupResp 设置2FA响应
type TwoFASetupResp struct {
	Secret  string `json:"secret"`  // Base32编码的密钥
	QRCode  string `json:"qr_code"` // 二维码URL (otpauth://totp/...)
	Enabled bool   `json:"enabled"` // 是否启用
}

// TwoFAVerifyReq 验证2FA请求
type TwoFAVerifyReq struct {
	Code string `json:"code" binding:"required"` // 6位TOTP验证码
}

// TwoFAStatusResp 2FA状态响应
type TwoFAStatusResp struct {
	Enabled bool `json:"enabled"` // 是否已启用2FA
}

// TwoFADisableReq 禁用2FA请求
type TwoFADisableReq struct {
	Password string `json:"password" binding:"required"` // 需要密码验证
}
