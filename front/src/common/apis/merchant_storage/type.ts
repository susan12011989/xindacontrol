import type { ApiResponseData } from "@/common/apis/type"

// 分页基础接口
export interface Pagination {
  page: number
  size: number
}

// 查询商户存储配置请求
export interface QueryMerchantStorageReq extends Pagination {
  merchant_id?: number
  storage_type?: string
  status?: number
}

// 创建/更新商户存储配置请求
export interface MerchantStorageReq {
  merchant_id: number
  storage_type: string
  name: string
  endpoint?: string
  bucket: string
  region?: string
  access_key_id: string
  access_key_secret?: string
  upload_url?: string
  download_url?: string
  file_base_url?: string
  bucket_url?: string
  custom_domain?: string
  is_default?: number
  status?: number
}

// 推送存储配置请求
export interface PushStorageConfigReq {
  merchant_id: number
  config_id: number
  twofa_code: string
}

// 商户存储配置响应
export interface MerchantStorageResp {
  id: number
  merchant_id: number
  merchant_name: string
  merchant_no: string
  storage_type: string
  name: string
  endpoint: string
  bucket: string
  region: string
  access_key_id: string
  access_key_secret: string
  upload_url: string
  download_url: string
  file_base_url: string
  bucket_url: string
  custom_domain: string
  is_default: number
  status: number
  last_push_at: string
  last_push_result: string
  created_at: string
  updated_at: string
}

// 推送结果
export interface PushStorageResult {
  success: boolean
  message: string
}

// 存储类型选项
export interface StorageTypeOption {
  value: string
  label: string
}

// 列表响应
export interface QueryMerchantStorageResponse {
  list: MerchantStorageResp[]
  total: number
}

// API 响应类型
export type QueryMerchantStorageResponseData = ApiResponseData<QueryMerchantStorageResponse>
export type MerchantStorageDetailResponseData = ApiResponseData<MerchantStorageResp>
export type PushStorageResultResponseData = ApiResponseData<PushStorageResult>
export type StorageTypesResponseData = ApiResponseData<StorageTypeOption[]>
