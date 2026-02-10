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
}

// 更新云账号请求
export interface UpdateCloudAccountReq {
  name?: string
  site_type?: string // cn-国内站, intl-国际站
  access_key_id?: string
  access_key_secret?: string
  description?: string
  status?: number
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
  created_at: string
  updated_at: string
}

// 阿里云账户余额响应
export interface AliyunBalanceResp {
  balance: string
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
