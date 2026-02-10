// 云存储公共类型定义 - 支持 AWS S3、阿里云 OSS 和腾讯云 COS

/** 云类型 */
export type CloudType = "aws" | "aliyun" | "tencent"

/** 通用请求基础参数 */
export interface CloudStorageBaseReq {
  merchant_id?: number
  cloud_account_id?: number
  region_id?: string
}

/** Bucket 项 */
export interface CloudBucketItem {
  name: string
  location: string
  creation_date: string
  storage_class: string
}

/** 列出 Bucket 请求参数 */
export interface ListBucketsReq extends CloudStorageBaseReq {
  prefix?: string
}

/** 列出 Bucket 响应 */
export interface ListBucketsResponse {
  list: CloudBucketItem[]
  is_truncated?: boolean
  next_marker?: string
  total?: number
}

/** 对象项 */
export interface CloudObjectItem {
  key: string
  size: number
  etag?: string
  last_modified: string
  storage_class: string
}

/** 列出对象请求参数 */
export interface ListObjectsReq extends CloudStorageBaseReq {
  bucket: string
  prefix?: string
  max_keys?: number
  // AWS 使用 continuation_token
  continuation_token?: string
  // 阿里云使用 marker
  marker?: string
}

/** 列出对象响应 */
export interface ListObjectsResponse {
  list: CloudObjectItem[]
  is_truncated: boolean
  // AWS 返回 next_continuation_token
  next_continuation_token?: string
  // 阿里云返回 next_marker
  next_marker?: string
  total?: number
}

/** 上传对象参数 (FormData fields) */
export interface UploadObjectParams {
  file: File
  cloud_account_id?: number
  merchant_id?: number
  region_id: string
  bucket: string
  object_key: string
}

/** 下载对象请求参数 */
export interface DownloadObjectReq extends CloudStorageBaseReq {
  region_id: string
  bucket: string
  object_key: string
  filename?: string
  attachment?: number
}

/** 删除对象请求参数 */
export interface DeleteObjectReq extends CloudStorageBaseReq {
  region_id: string
  bucket: string
  object_key: string
}

/** 设置 Bucket 公开访问请求参数 */
export interface SetBucketPublicReq extends CloudStorageBaseReq {
  region_id: string
  bucket: string
  public: boolean
}

/** 创建 Bucket 请求参数 */
export interface CreateBucketReq extends CloudStorageBaseReq {
  region_id: string
  bucket: string
  storage_class?: string // 仅阿里云支持: Standard, IA, Archive
}

/** 删除 Bucket 请求参数 */
export interface DeleteBucketReq extends CloudStorageBaseReq {
  region_id: string
  bucket: string
}

/** 上传队列项 */
export interface UploadQueueItem {
  name: string
  status: "waiting" | "uploading" | "success" | "failed"
  progress: number
  size: number
}

/** 上传统计 */
export interface UploadStats {
  total: number
  success: number
  failed: number
}

/** 区域选项 */
export interface RegionOption {
  id: string
  name: string
}
