import type * as Types from "./type"
import { request } from "@/http/axios"

export function listObjects(params: Types.OssListRequestData) {
  return request<Types.OssListResponse>({
    url: "/cloud/oss/objects",
    method: "get",
    params
  })
}

export function listBuckets(params: { merchant_id?: number, cloud_account_id?: number, region_id?: string, prefix?: string, marker?: string, max_keys?: number }) {
  return request<ApiResponseData<{ list: { name: string, location: string, creation_date: string, storage_class: string }[], is_truncated: boolean, next_marker: string, total: number }>>({
    url: "/cloud/oss/buckets",
    method: "get",
    params
  })
}

export function uploadObject(form: FormData, onUploadProgress?: (progressEvent: any) => void) {
  return request({
    url: "/cloud/oss/object",
    method: "post",
    headers: { "Content-Type": "multipart/form-data" },
    data: form,
    timeout: 300000,
    onUploadProgress
  })
}

export function downloadObject(params: { merchant_id?: number, cloud_account_id?: number, region_id: string, bucket: string, object_key: string, filename?: string, attachment?: number }) {
  return request<Blob>({
    url: "/cloud/oss/object",
    method: "get",
    params,
    responseType: "blob",
    timeout: 300000
  })
}
