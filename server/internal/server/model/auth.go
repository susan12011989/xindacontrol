package model

type LoginReq struct {
	Username string `json:"username" form:"username"`
	Password string `json:"password" form:"password"`
}

// EncryptedLoginReq 前端使用服务端公钥对 JSON 明文进行 RSA-OAEP 加密后的密文
// 密文为 Base64 字符串
type EncryptedLoginReq struct {
	Cipher string `json:"cipher" binding:"required"`
}

// EncryptedLoginPayload 明文负载
type EncryptedLoginPayload struct {
	Username  string `json:"username"`
	Password  string `json:"password"`
	TwoFACode string `json:"two_fa_code"` // 2FA验证码，可选
	Nonce     string `json:"nonce"`
	Ts        int64  `json:"ts"` // unix seconds
}

type LoginResp struct {
	Token            string `json:"token"`
	TwoFactorEnabled bool   `json:"two_factor_enabled"` // 是否已启用2FA
}

type MeResp struct {
	Username         string   `json:"username"`
	Roles            []string `json:"roles"`
	TwoFactorEnabled bool     `json:"two_factor_enabled"` // 是否已启用2FA
	Ip               string   `json:"ip"`
}

// ChallengeResp 返回一次性 nonce 与 RSA 公钥（PEM）
type ChallengeResp struct {
	Nonce   string `json:"nonce"`
	PubPEM  string `json:"pub_pem"`
	Expires int64  `json:"expires"` // unix seconds
}
