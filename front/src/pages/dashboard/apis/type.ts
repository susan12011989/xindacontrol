// 套餐配置
export interface PackageConfiguration {
  dau_limit: number // 日活限制
  register_limit: number // 注册人数限制
  group_member_limit: number // 群人数限制
  app_packages?: string[] // 套餐内可用应用列表（客户端包名）
  turn_server?: string // TURN服务器地址 (格式: ip:port)
}

// 商户区域信息
export interface MerchantRegions {
  region_name: string
}

// 商户信息
export interface Merchant {
  id: number // 商户ID
  uuid?: string // 兼容前端显示字段
  no: string // 商户编号
  port: number // 商户端口
  server_ip: string // 服务器IP
  name: string // 商户名称
  app_name?: string // 应用名称（用于打包显示）
  logo_url?: string // Logo 地址
  icon_url?: string // 应用图标地址
  status: number // 1:正常,-1:禁用
  expired_at: string // 服务过期时间 yyyy-MM-dd HH:mm:ss
  created_at: string // 创建时间 yyyy-MM-dd HH:mm:ss
  updated_at: string // 更新时间 yyyy-MM-dd HH:mm:ss
  package_configuration?: PackageConfiguration // 套餐配置
  expiring_soon: number // 2:已过期 1:即将过期 0:正常
  regions?: MerchantRegions[] // 商户区域信息
}

export interface MerchantQueryRequestData {
  page: number
  size: number
  name?: string
  order?: string // 排序字段 创建时间升序或倒序: id desc / id asc , 过期时间升序或倒序: expired_at desc / expired_at asc
  expiring_soon?: number // 2:已过期 1:即将过期 0:正常
  merchant_no?: string // 商户编号
}

export type MerchantQueryResponseData = ApiResponseData<{
  total: number
  list: Merchant[]
}>

// 创建或编辑商户请求
export interface CreateOrEditMerchantRequestData {
  id?: number
  name: string
  app_name?: string // 应用名称（用于打包显示）
  logo_url?: string // Logo 地址
  icon_url?: string // 应用图标地址
  no?: string
  port?: number
  server_ip?: string
  status?: number
  expired_at: string
  package_configuration?: PackageConfiguration // 套餐配置
  // 创建商户时使用的AWS账号（写入 CloudAccounts）
  aws_access_key_id?: string
  aws_access_key_secret?: string
  // 选择现有系统AWS账号的ID（优先使用此选项）
  selected_aws_account_id?: number
  // 是否将选中的系统账号移除（转为商户账号）
  remove_from_system?: boolean
}
export type CreateOrEditMerchantResponseData = ApiResponseData<null>

// 云账号余额查询请求
export interface BalanceReq {
  merchant_id: number[]
}

// 云账号余额数据
export interface BalanceData {
  merchant_id: number
  balance: string
}

export type BalanceResponseData = ApiResponseData<BalanceData[]>

// 隧道连接检测
export interface TunnelCheckReq {
  merchant_id?: number
  server_ip?: string
}

export interface TunnelCheckItem {
  server_name: string
  server_ip: string
  success: boolean
  message: string
}

export type TunnelCheckResponseData = ApiResponseData<TunnelCheckItem[]>

// 更换IP
export interface ChangeIPResp {
  old_ip: string
  new_ip: string
  region: string
  instance_id: string
  old_allocation_id?: string
  new_allocation_id: string
}
export type ChangeIPResponseData = ApiResponseData<ChangeIPResp>

// 更换 GOST 隧道端口
export interface ChangeGostPortResp {
  merchant_id: number
  old_port: number
  new_port: number
}
export type ChangeGostPortResponseData = ApiResponseData<ChangeGostPortResp>

// ========== Adminm 配置（短信） ==========
export interface SmsConfig {
  region_id: string
  access_key: string
  secret_key: string
  sign_name: string
  template_code: string
}

export type AdminmSmsConfigResponseData = ApiResponseData<SmsConfig | null>

// 保存请求
export interface AdminmSmsSaveReq {
  merchant_no?: string
  merchant_nos?: string[]
  broadcast?: boolean
  config: SmsConfig
}

// ===== 敏感词（从txt文本解析 word,tip） =====
export interface SensitiveContent {
  word: string
  tip: string
}
export interface AdminmSensitiveSaveReq {
  merchant_no?: string
  merchant_nos?: string[]
  broadcast?: boolean
  txt: string
  contents?: SensitiveContent[]
}

export type AdminmSaveResponseData = ApiResponseData<any>

// 系统用户昵称保存
export interface AdminmNicknameSaveReq {
  merchant_no?: string
  merchant_nos?: string[]
  broadcast?: boolean
  first_name: string
}

// ========== 隧道统计 ==========
export interface TunnelStats {
  total_merchants: number // 商户总数
  total_gost_servers: number // 系统服务器总数
  total_merchant_servers: number // 商户服务器总数
}

export type TunnelStatsResponseData = ApiResponseData<TunnelStats>
