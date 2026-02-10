/**
 * 腾讯云 COS API 定义
 */
import type * as Types from "./type"
import { request } from "@/http/axios"

/** 列出对象 */
export function listObjects(params: Types.CosListRequestData) {
  return request<Types.CosListResponse>({
    url: "/cloud/tencent/cos/objects",
    method: "get",
    params
  })
}

/** 列出 Bucket */
export function listBuckets(params: {
  merchant_id?: number
  cloud_account_id?: number
  region_id?: string
}) {
  return request<Types.CosBucketListResponse>({
    url: "/cloud/tencent/cos/buckets",
    method: "get",
    params
  })
}

/** 上传对象 */
export function uploadObject(form: FormData, onUploadProgress?: (progressEvent: any) => void) {
  return request({
    url: "/cloud/tencent/cos/object",
    method: "post",
    headers: { "Content-Type": "multipart/form-data" },
    data: form,
    timeout: 300000,
    onUploadProgress
  })
}

/** 下载对象 */
export function downloadObject(params: {
  merchant_id?: number
  cloud_account_id?: number
  region_id: string
  bucket: string
  object_key: string
  filename?: string
  attachment?: number
}) {
  return request<Blob>({
    url: "/cloud/tencent/cos/object",
    method: "get",
    params,
    responseType: "blob",
    timeout: 300000
  })
}
