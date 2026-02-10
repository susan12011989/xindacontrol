package entity

// AdminmUserPayload 下发给 server 的账号管理载荷
// 匹配商户端 TangSengDaoDaoServer/modules/control/model.go 的 AdminmUserPayload
type AdminmUserPayload struct {
	LoginName string `json:"login_name"`         // 登录用户名（对应商户端 user.username）
	Name      string `json:"name"`               // 显示名称（对应商户端 user.name）
	Phone     string `json:"phone"`              // 手机号（对应商户端 user.phone）
	Password  string `json:"password,omitempty"` // 密码
}

// AdminmUserUpdatePayload 更新时的载荷（以 login_name 为定位键）
// 匹配商户端 TangSengDaoDaoServer/modules/control/model.go 的 AdminmUserUpdatePayload
type AdminmUserUpdatePayload struct {
	LoginName  string  `json:"login_name"`           // 目标登录用户名（用于定位用户）
	Name       *string `json:"name,omitempty"`       // 显示名称
	Password   *string `json:"password,omitempty"`   // 密码
	AllowedIPs *string `json:"allowed_ips,omitempty"` // IP白名单（逗号分隔）
}

// AdminmUserDeletePayload 删除时的载荷
// 匹配商户端 TangSengDaoDaoServer/modules/control/model.go 的 AdminmUserDeletePayload
type AdminmUserDeletePayload struct {
	LoginName string `json:"login_name"` // 登录用户名
}

// ===== 列表查询 =====

type AdminmUserQueryReq struct {
	RequestId string `json:"request_id"`
	Page      int    `json:"page,omitempty"`
	Size      int    `json:"size,omitempty"`
	Username  string `json:"username,omitempty"`
}

// AdminmUserListItem 管理用户列表项
// 匹配商户端 TangSengDaoDaoServer/modules/control/model.go 的 AdminmUserListItem
type AdminmUserListItem struct {
	UID          string `json:"uid"`           // 用户 UID
	Name         string `json:"name"`          // 显示名称
	Username     string `json:"username"`      // 登录用户名
	RegisterTime string `json:"register_time"` // 注册时间
	AllowedIPs   string `json:"allowed_ips"`   // IP白名单
}

// AdminmUserQueryResp 管理用户查询响应
// 匹配商户端 TangSengDaoDaoServer/modules/control/model.go 的 AdminmUserQueryResp
type AdminmUserQueryResp struct {
	List  []AdminmUserListItem `json:"list"`
	Total int64                `json:"total"`
	Err   string               `json:"err,omitempty"` // 用于内部错误传递
}

// ===== 活跃数据 =====
// AdminmActiveResp 活跃数据响应
// 匹配商户端 TangSengDaoDaoServer/modules/control/model.go 的 ActiveDataResp
type AdminmActiveResp struct {
	TotalUsers  int64  `json:"total_users"`
	OnlineUsers int64  `json:"online_users"`
	Dau         int    `json:"dau"`
	Err         string `json:"err,omitempty"` // 用于内部错误传递
}

// ===== 短信配置查询 =====
type AdminmSmsConfigResp struct {
	RequestId string    `json:"request_id"`
	Config    SmsConfig `json:"config"`
	Err       string    `json:"err,omitempty"`
}

// ===== 敏感词内容 =====
type SensitiveContent struct {
	Word string `json:"word"`
	Tip  string `json:"tip"`
}
