package model

// ========== 功能开关管理 ==========

// 功能定义
type FeatureDefinition struct {
	Name        string `json:"name"`
	Label       string `json:"label"`
	Description string `json:"description"`
	Category    string `json:"category"`
}

// 预定义的功能列表（基于代码库实际实现的功能）
var AvailableFeatures = []FeatureDefinition{
	// 支付类
	{Name: "redpacket", Label: "红包", Description: "红包发送和领取功能", Category: "支付"},
	{Name: "wallet", Label: "钱包", Description: "钱包充值提现功能", Category: "支付"},

	// 通讯类
	{Name: "rtc", Label: "音视频通话", Description: "实时音视频通话功能", Category: "通讯"},
	{Name: "group", Label: "群组", Description: "群组创建和管理功能", Category: "通讯"},
	{Name: "forward", Label: "消息转发", Description: "转发消息到其他会话", Category: "通讯"},
	{Name: "recall", Label: "消息撤回", Description: "撤回已发送的消息", Category: "通讯"},
	{Name: "read_receipt", Label: "已读回执", Description: "显示消息是否已读", Category: "通讯"},
	{Name: "voice_msg", Label: "语音消息", Description: "发送语音消息功能", Category: "通讯"},
	{Name: "video_msg", Label: "视频消息", Description: "发送视频消息功能", Category: "通讯"},
	{Name: "file_transfer", Label: "文件传输", Description: "发送文件和附件", Category: "通讯"},

	// 社交类
	{Name: "moments", Label: "朋友圈", Description: "朋友圈动态发布和查看", Category: "社交"},
	{Name: "namecard", Label: "名片分享", Description: "分享个人或群名片", Category: "社交"},
	{Name: "add_friend", Label: "添加好友", Description: "搜索和添加好友功能", Category: "社交"},

	// 安全类
	{Name: "burn_after_read", Label: "阅后即焚", Description: "消息阅读后自动销毁", Category: "安全"},
	{Name: "screenshot_notify", Label: "截屏通知", Description: "检测并通知对方截屏", Category: "安全"},

	// 工具类
	{Name: "location", Label: "位置分享", Description: "实时位置分享功能", Category: "工具"},
	{Name: "online_status", Label: "在线状态", Description: "显示用户在线/离线状态", Category: "工具"},
	{Name: "scan_qr", Label: "扫一扫", Description: "扫描二维码功能", Category: "工具"},
	{Name: "favorites", Label: "收藏", Description: "收藏消息和内容", Category: "工具"},

	// 娱乐类
	{Name: "sticker", Label: "表情包", Description: "表情包商店和发送", Category: "娱乐"},

	// 服务类
	{Name: "customer_service", Label: "客服", Description: "在线客服支持功能（hotline模块）", Category: "服务"},
}

// 查询商户功能开关请求
type QueryFeatureFlagsReq struct {
	MerchantId int `json:"merchant_id" form:"merchant_id" binding:"required"`
}

// 功能开关响应
type FeatureFlagResp struct {
	Id          int    `json:"id"`
	MerchantId  int    `json:"merchant_id"`
	FeatureName string `json:"feature_name"`
	Label       string `json:"label"`
	Description string `json:"description"`
	Category    string `json:"category"`
	Enabled     bool   `json:"enabled"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// 商户功能开关列表响应
type QueryFeatureFlagsResponse struct {
	List       []FeatureFlagResp `json:"list"`
	MerchantId int               `json:"merchant_id"`
}

// 更新功能开关请求
type UpdateFeatureFlagReq struct {
	MerchantId  int    `json:"merchant_id" binding:"required"`
	FeatureName string `json:"feature_name" binding:"required"`
	Enabled     bool   `json:"enabled"`
}

// 批量更新功能开关请求
type BatchUpdateFeatureFlagsReq struct {
	MerchantId int                       `json:"merchant_id" binding:"required"`
	Features   []FeatureFlagUpdateItem   `json:"features" binding:"required,min=1"`
}

type FeatureFlagUpdateItem struct {
	FeatureName string `json:"feature_name" binding:"required"`
	Enabled     bool   `json:"enabled"`
}

// 初始化商户功能开关请求
type InitFeatureFlagsReq struct {
	MerchantId int `json:"merchant_id" binding:"required"`
}

// 通用操作响应
type FeatureFlagOperationResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}
