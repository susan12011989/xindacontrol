import type { ApiResponseData } from "@/common/apis/type"

// 分页
export interface Pagination {
  page: number
  size: number
}

// 查询客户端请求
export interface QueryClientsReq extends Pagination {
  app_package_name?: string // 安卓包名
  app_name?: string // app名称
}

// 短信配置
export interface SmsConfig {
  region_id: string // 区域ID
  access_key: string // AccessKey
  secret_key: string // SecretKey
  sign_name: string // 签名
  template_code: string // 模板代码
}

// TRTC 配置
export interface TrtcConfig {
  app_id: number
  app_key: string
}

// 华为推送配置
export interface PushHMS {
  app_id: string
  app_secret: string
}

// 小米推送配置
export interface PushXiaomi {
  app_secret: string
  package: string
  channel_id: string
  time_to_live: number
}

// OPPO推送配置
export interface PushOppo {
  app_key: string
  master_secret: string
  channel_id: string
  time_to_live: number
}

// Vivo推送配置
export interface PushVivo {
  app_id: number
  app_key: string
  app_secret: string
  time_to_live: number
}

// 荣耀推送配置
export interface PushHonor {
  app_id: string
  client_id: string
  client_secret: string
  time_to_live: number
}

// 所有推送配置
export interface AllPushConfig {
  push_hms?: PushHMS // 华为
  push_xiaomi?: PushXiaomi // 小米
  push_oppo?: PushOppo // OPPO
  push_vivo?: PushVivo // Vivo
  push_honor?: PushHonor // 荣耀
}

// 创建客户端请求
export interface CreateClientReq {
  app_package_name: string // 安卓包名
  app_name: string // app名称
  sms_config?: SmsConfig // 短信配置
  push_config?: AllPushConfig // 推送配置
  trtc_config?: TrtcConfig // TRTC配置
}

// 更新客户端请求
export interface UpdateClientReq {
  app_package_name?: string // 安卓包名
  app_name?: string // app名称
  sms_config?: SmsConfig // 短信配置
  push_config?: AllPushConfig // 推送配置
  trtc_config?: TrtcConfig // TRTC配置
}

// 客户端响应
export interface ClientResp {
  id: number // ID
  app_package_name: string // 安卓包名
  app_name: string // app名称
  sms_config?: SmsConfig // 短信配置
  push_config?: AllPushConfig // 推送配置
  trtc_config?: TrtcConfig // TRTC配置
}

// 客户端列表响应
export interface QueryClientsResponse {
  list: ClientResp[]
  total: number
}

// API 响应类型
export type QueryClientsResponseData = ApiResponseData<QueryClientsResponse>
export type ClientDetailResponseData = ApiResponseData<ClientResp>
