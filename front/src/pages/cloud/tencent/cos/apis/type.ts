/**
 * 腾讯云 COS 类型定义
 */

/** 列出对象请求参数 */
export interface CosListRequestData {
  merchant_id?: number
  cloud_account_id?: number
  region_id: string
  bucket: string
  prefix?: string
  marker?: string
  max_keys?: number
}

/** COS 对象项 */
export interface CosObjectItem {
  key: string
  size: number
  etag: string
  last_modified: string
  storage_class: string
}

/** COS 列表响应 */
export type CosListResponse = ApiResponseData<{
  list: CosObjectItem[]
  is_truncated: boolean
  next_marker: string
  total: number
}>

/** COS Bucket 项 */
export interface CosBucketItem {
  name: string
  location: string
  creation_date: string
}

/** 列出 Bucket 响应 */
export type CosBucketListResponse = ApiResponseData<{
  list: CosBucketItem[]
  total: number
}>
