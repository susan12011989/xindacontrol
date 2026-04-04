import type { ApiResponseData } from "@/common/apis/type"

// 分页基础类型
export interface Pagination {
  page: number
  size: number
}

// 查询云账号请求
export interface QueryCloudAccountsReq extends Pagination {
  name?: string
  cloud_type?: string
  status?: number
  account_type?: string
  merchant_id?: number
}

// 创建云账号请求
export interface CreateCloudAccountReq {
  name: string
  cloud_type: string
  site_type?: string // cn-国内站, intl-国际站
  access_key_id: string
  access_key_secret: string
  region?: string
  description?: string
  merchant_id?: number // 商户ID，0或空表示系统账号
}

// 更新云账号请求
export interface UpdateCloudAccountReq {
  name?: string
  site_type?: string // cn-国内站, intl-国际站
  access_key_id?: string
  access_key_secret?: string
  description?: string
  status?: number
  merchant_id?: number // 商户ID，0表示系统账号
}

// 云账号响应
export interface CloudAccountResp {
  id: number
  name: string
  cloud_type: string
  site_type: string // cn-国内站, intl-国际站
  access_key_id: string
  access_key_secret: string
  region: string
  description: string
  status: number
  account_type: string
  merchant_id: number
  merchant_name: string
  created_at: string
  updated_at: string
}

// 阿里云账户余额响应
export interface AliyunBalanceResp {
  available_amount: string // 可用额度（含信用额度）
  available_cash_amount: string // 现金余额（真实余额）
  credit_amount: string // 信用额度
  currency: string // 币种
}

// 腾讯云账户余额响应
export interface TencentBalanceResp {
  balance: number // 可用余额（分）
  balance_yuan: string // 可用余额（元）
  cash_balance: number // 现金余额（分）
  cash_balance_yuan: string // 现金余额（元）
  income_balance: number // 收入余额（分，代金券等）
  present_balance: number // 赠送余额（分）
  freeze_balance: number // 冻结金额（分）
  owe_balance: number // 欠费金额（分）
  is_overdue: boolean // 是否欠费
  is_overdue_balance: boolean // 余额是否小于0
}

// 云账号列表响应
export interface QueryCloudAccountsResponse {
  list: CloudAccountResp[]
  total: number
}

// 云账号选项（用于下拉框）
export interface CloudAccountOption {
  value: number
  label: string
  type: string
}

// API 响应类型
export type QueryCloudAccountsResponseData = ApiResponseData<QueryCloudAccountsResponse>
export type CloudAccountDetailResponseData = ApiResponseData<CloudAccountResp>
export type CloudAccountOptionsResponseData = ApiResponseData<CloudAccountOption[]>
