package model

// ========== TLS 证书管理 ==========

// 证书详情响应
type TlsCertificateResp struct {
	Id          int    `json:"id"`
	MerchantId  int    `json:"merchant_id"`
	Name        string `json:"name"`
	CertType    int    `json:"cert_type"`   // 1-CA根证书 2-服务器证书
	Fingerprint string `json:"fingerprint"` // SHA-256 指纹
	ExpiresAt   string `json:"expires_at"`
	Status      int    `json:"status"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// 生成证书请求
type GenerateTlsCertReq struct {
	MerchantId   int `json:"merchant_id" binding:"required"` // 商户ID
	ValidityDays int `json:"validity_days"`                  // 有效期天数，默认 3650(10年)
}

// 生成证书响应
type GenerateTlsCertResp struct {
	CA     TlsCertificateResp `json:"ca"`
	Server TlsCertificateResp `json:"server"`
}

// ========== TLS 批量操作 ==========

// 批量升级 TLS 请求
type BatchUpgradeTlsReq struct {
	MerchantId int   `json:"merchant_id" binding:"required"` // 商户ID
	ServerIds  []int `json:"server_ids"`                     // 为空则升级该商户所有 GOST 服务器
}

// 批量回滚 TLS 请求
type BatchRollbackTlsReq struct {
	MerchantId int   `json:"merchant_id" binding:"required"` // 商户ID
	ServerIds  []int `json:"server_ids"`                     // 为空则回滚该商户所有 GOST 服务器
}

// 停用证书请求
type DisableTlsCertReq struct {
	MerchantId int `json:"merchant_id" binding:"required"` // 商户ID
}

// 单台服务器 TLS 操作结果
type TlsServerResult struct {
	ServerId   int    `json:"server_id"`
	ServerName string `json:"server_name"`
	Host       string `json:"host"`
	Success    bool   `json:"success"`
	Error      string `json:"error,omitempty"`
}

// 批量 TLS 操作响应
type BatchTlsResp struct {
	Total   int               `json:"total"`
	Success int               `json:"success"`
	Failed  int               `json:"failed"`
	Results []TlsServerResult `json:"results"`
}

// TLS 状态查询响应
type TlsStatusResp struct {
	Total    int               `json:"total"`     // 系统服务器总数
	TlsCount int              `json:"tls_count"` // 已启用 TLS 数量
	TcpCount int              `json:"tcp_count"` // 未启用 TLS 数量
	Servers  []TlsServerStatus `json:"servers"`
}

// 单台服务器 TLS 状态
type TlsServerStatus struct {
	ServerId      int    `json:"server_id"`
	ServerName    string `json:"server_name"`
	Host          string `json:"host"`
	TlsEnabled    int    `json:"tls_enabled"`     // 0-未启用 1-已启用
	TlsDeployedAt string `json:"tls_deployed_at"` // 证书部署时间
	TlsVerified   bool   `json:"tls_verified"`    // TLS 连接验证是否通过
	VerifyError   string `json:"verify_error,omitempty"`
}
