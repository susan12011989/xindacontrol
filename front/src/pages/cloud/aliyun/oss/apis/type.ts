export interface OssListRequestData {
  merchant_id?: number
  cloud_account_id?: number
  region_id: string
  bucket: string
  prefix?: string
  marker?: string
  max_keys?: number
}

export interface OssObjectItem {
  key: string
  size: number
  etag: string
  last_modified: string
  storage_class: string
}

export type OssListResponse = ApiResponseData<{
  list: OssObjectItem[]
  is_truncated: boolean
  next_marker: string
  total: number
}>
