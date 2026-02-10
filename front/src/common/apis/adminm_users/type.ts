import type { ApiResponseData } from "@@/apis/type"

export interface AdminmUserListItem {
  uid: string
  name: string
  username: string
  register_time: string
  allowed_ips: string // IP白名单
}

export interface AdminmUsersQueryReq {
  merchant_no: string
  page: number
  size: number
  username?: string
}

export type AdminmUsersQueryResp = ApiResponseData<{
  list: AdminmUserListItem[]
  total: number
}>

export interface CreateAdminmUserReq {
  merchant_no: string
  username: string
  phone: string
  password: string
  two_factor_secret?: string
  two_factor_enabled?: number
}

export interface UpdateAdminmUserReq {
  merchant_no: string
  target_username: string
  username?: string
  password?: string
  two_factor_secret?: string
  two_factor_enabled?: number
  allowed_ips?: string // IP白名单，逗号分隔
}

export interface DeleteAdminmUserReq {
  merchant_no: string
  username: string
}

// 活跃数据查询
export interface AdminmActiveQueryReq {
  merchant_no: string
}

export interface AdminmActiveResp {
  total_users: number
  online_users: number
  dau: number
}

export type AdminmActiveRespData = ApiResponseData<AdminmActiveResp>
