package model

import "server/pkg/entity"

// ========== 客户端管理 ==========

// 查询客户端请求
type QueryClientsReq struct {
	Pagination
	AppPackageName string `json:"app_package_name" form:"app_package_name"` // 安卓包名（模糊查询）
	AppName        string `json:"app_name" form:"app_name"`                 // app名称（模糊查询）
}

// 创建客户端请求
type CreateClientReq struct {
	AppPackageName string                `json:"app_package_name" binding:"required"` // 安卓包名
	AppName        string                `json:"app_name" binding:"required"`         // app名称
	SmsConfig      *entity.SmsConfig     `json:"sms_config"`                          // 短信配置
	PushConfig     *entity.AllPushConfig `json:"push_config"`                         // 推送配置
	TrtcConfig     *entity.TrtcConfig    `json:"trtc_config"`                         // TRTC配置
}

// 更新客户端请求
type UpdateClientReq struct {
	AppPackageName string                `json:"app_package_name"` // 安卓包名
	AppName        string                `json:"app_name"`         // app名称
	SmsConfig      *entity.SmsConfig     `json:"sms_config"`       // 短信配置
	PushConfig     *entity.AllPushConfig `json:"push_config"`      // 推送配置
	TrtcConfig     *entity.TrtcConfig    `json:"trtc_config"`      // TRTC配置
}

// 客户端响应
type ClientResp struct {
	Id             int                   `json:"id"`               // ID
	AppPackageName string                `json:"app_package_name"` // 安卓包名
	AppName        string                `json:"app_name"`         // app名称
	SmsConfig      *entity.SmsConfig     `json:"sms_config"`       // 短信配置
	PushConfig     *entity.AllPushConfig `json:"push_config"`      // 推送配置
	TrtcConfig     *entity.TrtcConfig    `json:"trtc_config"`      // TRTC配置
}

// 客户端列表响应
type QueryClientsResponse struct {
	List  []ClientResp `json:"list"`
	Total int          `json:"total"`
}
