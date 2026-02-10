package model

// ========== 端口转换 ==========

type Port2EnterpriseReq struct {
	Port uint16 `json:"port" binding:"required"`
}
type Port2EnterpriseResp struct {
	Enterprise string `json:"enterprise" binding:"required"`
}
type Enterprise2PortReq struct {
	Enterprise string `json:"enterprise" binding:"required"`
}
type Enterprise2PortResp struct {
	Port uint16 `json:"port" binding:"required"`
}

// ========== IP工具 ==========

// IP嵌入请求
type EmbedIPsReq struct {
	IPs []string `json:"ips" binding:"required,dive,ip"`
}

// IP提取响应
type ExtractIPsResp struct {
	IPs []string `json:"ips"`
}

// ========== URL工具 ==========

// URL提取响应
type ExtractURLsResp struct {
	URLs []string `json:"urls"`
}

// ========== 版本管理 ==========

// 版本条目
type VersionEntry struct {
	Channel string `json:"channel" binding:"required"`
	Version string `json:"version" binding:"required"`
}

// 生成版本配置请求
type GenerateVersionReq struct {
	Package  string         `json:"package" binding:"required"`
	Versions []VersionEntry `json:"versions" binding:"required,dive"`
}
