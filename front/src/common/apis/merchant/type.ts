import type { ApiResponseData } from "@@/apis/type"

export interface Pagination {
  page: number
  size: number
}

export interface QueryMerchantsReq extends Pagination {
  name?: string
  order?: string
}

export interface MerchantResp {
  id: number
  no: string
  name: string
  port?: number
  server_ip?: string
  status?: number
  expired_at?: string
}

export interface QueryMerchantsResponse {
  list: MerchantResp[]
  total: number
}

export type QueryMerchantsResponseData = ApiResponseData<QueryMerchantsResponse>

export interface MerchantGostServerResp {
  id: number
  merchant_id: number
  server_id: number
  server_name: string
  server_host: string
  cloud_type: string
  region: string
  listen_port: number
  is_primary: number
  priority: number
  status: number
  server_status: number
  created_at: string
  updated_at: string
}

export type MerchantGostServersResponseData = ApiResponseData<MerchantGostServerResp[]>

/** 创建商户 GOST 服务器关联请求 */
export interface CreateMerchantGostServerReq {
  server_id: number
  is_primary?: number
  priority?: number
  status?: number
}

// ========== 商户 OSS 配置 ==========

export interface MerchantOssConfigResp {
  id: number
  merchant_id: number
  cloud_account_id: number
  cloud_account_name: string
  cloud_type: string
  name: string
  bucket: string
  region: string
  endpoint: string
  custom_domain: string
  is_default: number
  status: number
  created_at: string
  updated_at: string
}

export type MerchantOssConfigsResponseData = ApiResponseData<MerchantOssConfigResp[]>

export interface CreateMerchantOssConfigReq {
  cloud_account_id: number
  name: string
  bucket: string
  region?: string
  endpoint?: string
  custom_domain?: string
  is_default?: number
}

export interface UpdateMerchantOssConfigReq extends CreateMerchantOssConfigReq {
  status?: number
}
