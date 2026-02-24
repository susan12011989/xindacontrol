import type { ApiResponseData } from "@/common/apis/type"

// 分页
export interface Pagination {
  page: number
  size: number
}

// ========== 资源标签 ==========

export interface ResourceTagReq {
  name: string
  color?: string
  description?: string
}

export interface ResourceTagResp {
  id: number
  name: string
  color: string
  description: string
  created_at: string
}

// ========== 标签分配 ==========

export interface AssignTagsReq {
  resource_type: string // oss_config / gost_server / storage_config
  resource_ids: number[]
  tag_ids: number[]
}

export interface RemoveTagsReq {
  resource_type: string
  resource_ids: number[]
  tag_ids: number[]
}

// ========== 全局 OSS 配置 ==========

export interface QueryGlobalOssConfigsReq extends Pagination {
  merchant_id?: number
  cloud_type?: string
  region?: string
  tag_id?: number
  status?: number
}

export interface GlobalOssConfigResp {
  id: number
  merchant_id: number
  merchant_name: string
  merchant_no: string
  cloud_account_id: number
  cloud_account_name: string
  cloud_type: string
  name: string
  bucket: string
  region: string
  endpoint: string
  custom_domain: string
  download_url: string
  is_default: number
  status: number
  tags: ResourceTagResp[]
  created_at: string
  updated_at: string
}

export interface QueryGlobalOssConfigsResponse {
  list: GlobalOssConfigResp[]
  total: number
}

// ========== 全局 GOST 服务器 ==========

export interface QueryGlobalGostServersReq extends Pagination {
  merchant_id?: number
  cloud_type?: string
  region?: string
  tag_id?: number
  status?: number
}

export interface GlobalGostServerResp {
  id: number
  merchant_id: number
  merchant_name: string
  merchant_no: string
  server_id: number
  server_name: string
  server_host: string
  cloud_type: string
  region: string
  listen_port: number
  is_primary: number
  priority: number
  status: number
  remark: string
  tags: ResourceTagResp[]
  created_at: string
  updated_at: string
}

export interface QueryGlobalGostServersResponse {
  list: GlobalGostServerResp[]
  total: number
}

// ========== 批量操作 ==========

export interface BatchSyncGostIPByFilterReq {
  merchant_ids?: number[]
  cloud_type?: string
  tag_id?: number
}

export interface SyncResultItem {
  oss_config_id: number
  oss_config_name: string
  cloud_type: string
  bucket: string
  object_key: string
  object_url: string
  success: boolean
  error?: string
}

export interface SyncGostIPResp {
  merchant_id: number
  merchant_name: string
  ips: string[]
  results: SyncResultItem[]
  summary: {
    total_oss: number
    success_count: number
    fail_count: number
    duration: string
  }
}

// ========== OSS 健康检测 ==========

export interface CheckOssHealthReq {
  oss_config_ids: number[]
}

export interface OssHealthCheckResult {
  oss_config_id: number
  oss_config_name: string
  merchant_name: string
  cloud_type: string
  bucket: string
  download_url: string
  cdn_url?: string
  healthy: boolean
  cdn_healthy?: boolean
  status_code: number
  cdn_status_code?: number
  message: string
  latency: string
}

// API 响应类型
export type TagListResponseData = ApiResponseData<ResourceTagResp[]>
export type GlobalOssConfigsResponseData = ApiResponseData<QueryGlobalOssConfigsResponse>
export type GlobalGostServersResponseData = ApiResponseData<QueryGlobalGostServersResponse>
export type BatchSyncResponseData = ApiResponseData<SyncGostIPResp[]>
export type OssHealthCheckResponseData = ApiResponseData<OssHealthCheckResult[]>
