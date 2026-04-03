import type { ApiResponseData } from "@@/apis/type"

export interface Pagination {
  page: number
  size: number
}

export interface QueryMerchantsReq extends Pagination {
  name?: string
  order?: string
}

export interface PackageConfiguration {
  direct_ip?: string
  direct_port?: number
  port?: number
  dau_limit?: number
  register_limit?: number
  group_member_limit?: number
  expired_at?: number
  app_packages?: string[]
  turn_server?: string
  turn_username?: string
  turn_credential?: string
}

export interface AppConfigs {
  oss_url?: string[]
}

export interface MerchantResp {
  id: number
  no: string
  name: string
  port?: number
  server_ip?: string
  status?: number
  expired_at?: string
  package_configuration?: PackageConfiguration
  app_configs?: AppConfigs
  oss_config_count?: number
  gost_server_count?: number
  deploy_mode?: string
}

export interface QueryMerchantsResponse {
  list: MerchantResp[]
  total: number
}

export type QueryMerchantsResponseData = ApiResponseData<QueryMerchantsResponse>

// ========== GOST 服务器 ==========

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
  forward_type: number  // 1=TLS加密 2=TCP直连
  tls_enabled: number   // 0/1
  remark: string
  created_at: string
  updated_at: string
}

export interface MerchantGostServerReq {
  server_id: number
  is_primary?: number
  priority?: number
  status?: number
  remark?: string
}

export type MerchantGostServersResponseData = ApiResponseData<MerchantGostServerResp[]>

// ========== OSS 配置 ==========

export interface MerchantOssConfigResp {
  id: number
  merchant_id: number
  cloud_account_id: number
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

export interface MerchantOssConfigReq {
  cloud_account_id: number
  name: string
  bucket: string
  region?: string
  endpoint?: string
  custom_domain?: string
  is_default?: number
}

export type MerchantOssConfigsResponseData = ApiResponseData<MerchantOssConfigResp[]>

// ========== GOST IP 同步 ==========

export interface SyncGostIPResult {
  success: boolean
  server_name: string
  server_host: string
  error?: string
}

export interface SyncGostIPResp {
  results: SyncGostIPResult[]
  total: number
  success_count: number
  fail_count: number
}

export type SyncGostIPResponseData = ApiResponseData<SyncGostIPResp>

// ========== 通用 ==========

export type CreateResponseData = ApiResponseData<{ id: number }>
