/**
 * 云存储公共 API 适配器
 * 统一封装 AWS S3、阿里云 OSS 和腾讯云 COS 的 API 调用
 */
import type * as Types from "./type"
import { request } from "@/http/axios"

// ========== AWS S3 API ==========

export function awsListBuckets(params: Types.ListBucketsReq) {
  return request<ApiResponseData<{ list: string[] }>>({
    url: "aws/s3/buckets",
    method: "get",
    params
  })
}

export function awsListObjects(params: Types.ListObjectsReq) {
  return request<ApiResponseData<Types.ListObjectsResponse>>({
    url: "aws/s3/objects",
    method: "get",
    params
  })
}

export function awsUploadObject(form: FormData, onUploadProgress?: (progressEvent: any) => void) {
  return request<any>({
    url: "aws/s3/object/upload",
    method: "post",
    headers: { "Content-Type": "multipart/form-data" },
    data: form,
    timeout: 300000,
    onUploadProgress
  })
}

export function awsDownloadObject(params: Types.DownloadObjectReq) {
  return request<Blob>({
    url: "aws/s3/object/download",
    method: "get",
    responseType: "blob",
    params,
    timeout: 300000
  })
}

export function awsSetBucketPublic(data: Types.SetBucketPublicReq) {
  return request<any>({
    url: "aws/s3/bucket/set-public",
    method: "post",
    data
  })
}

export function awsDeleteObject(data: Types.DeleteObjectReq) {
  return request<any>({
    url: "aws/s3/object",
    method: "delete",
    data
  })
}

export function awsCreateBucket(data: Types.CreateBucketReq) {
  return request<any>({
    url: "aws/s3/bucket",
    method: "post",
    data
  })
}

export function awsDeleteBucket(data: Types.DeleteBucketReq) {
  return request<any>({
    url: "aws/s3/bucket",
    method: "delete",
    data
  })
}

// ========== 阿里云 OSS API ==========

export function aliyunListBuckets(params: Types.ListBucketsReq) {
  return request<ApiResponseData<Types.ListBucketsResponse>>({
    url: "/cloud/oss/buckets",
    method: "get",
    params
  })
}

export function aliyunListObjects(params: Types.ListObjectsReq) {
  return request<ApiResponseData<Types.ListObjectsResponse>>({
    url: "/cloud/oss/objects",
    method: "get",
    params
  })
}

export function aliyunUploadObject(form: FormData, onUploadProgress?: (progressEvent: any) => void) {
  return request<any>({
    url: "/cloud/oss/object",
    method: "post",
    headers: { "Content-Type": "multipart/form-data" },
    data: form,
    timeout: 300000,
    onUploadProgress
  })
}

export function aliyunDownloadObject(params: Types.DownloadObjectReq) {
  return request<Blob>({
    url: "/cloud/oss/object",
    method: "get",
    responseType: "blob",
    params,
    timeout: 300000
  })
}

export function aliyunDeleteObject(data: Types.DeleteObjectReq) {
  return request<any>({
    url: "/cloud/oss/object",
    method: "delete",
    data
  })
}

export function aliyunCreateBucket(data: Types.CreateBucketReq) {
  return request<any>({
    url: "/cloud/oss/bucket",
    method: "post",
    data
  })
}

export function aliyunDeleteBucket(data: Types.DeleteBucketReq) {
  return request<any>({
    url: "/cloud/oss/bucket",
    method: "delete",
    data
  })
}

export function aliyunSetBucketPublic(data: Types.SetBucketPublicReq) {
  return request<any>({
    url: "/cloud/oss/bucket/set-public",
    method: "post",
    data
  })
}

// ========== 腾讯云 COS API ==========

export function tencentListBuckets(params: Types.ListBucketsReq) {
  return request<ApiResponseData<Types.ListBucketsResponse>>({
    url: "/cloud/tencent/cos/buckets",
    method: "get",
    params
  })
}

export function tencentListObjects(params: Types.ListObjectsReq) {
  return request<ApiResponseData<Types.ListObjectsResponse>>({
    url: "/cloud/tencent/cos/objects",
    method: "get",
    params
  })
}

export function tencentUploadObject(form: FormData, onUploadProgress?: (progressEvent: any) => void) {
  return request<any>({
    url: "/cloud/tencent/cos/object",
    method: "post",
    headers: { "Content-Type": "multipart/form-data" },
    data: form,
    timeout: 300000,
    onUploadProgress
  })
}

export function tencentDownloadObject(params: Types.DownloadObjectReq) {
  return request<Blob>({
    url: "/cloud/tencent/cos/object",
    method: "get",
    responseType: "blob",
    params,
    timeout: 300000
  })
}

export function tencentDeleteObject(data: Types.DeleteObjectReq) {
  return request<any>({
    url: "/cloud/tencent/cos/object",
    method: "delete",
    data
  })
}

export function tencentCreateBucket(data: Types.CreateBucketReq) {
  return request<any>({
    url: "/cloud/tencent/cos/bucket",
    method: "post",
    data
  })
}

export function tencentDeleteBucket(data: Types.DeleteBucketReq) {
  return request<any>({
    url: "/cloud/tencent/cos/bucket",
    method: "delete",
    data
  })
}

export function tencentSetBucketPublic(data: Types.SetBucketPublicReq) {
  return request<any>({
    url: "/cloud/tencent/cos/bucket/set-public",
    method: "post",
    data
  })
}

// ========== 统一适配器 ==========

export function createCloudStorageAdapter(cloudType: Types.CloudType) {
  if (cloudType === "aws") {
    return {
      listBuckets: awsListBuckets,
      listObjects: awsListObjects,
      uploadObject: awsUploadObject,
      downloadObject: awsDownloadObject,
      deleteObject: awsDeleteObject,
      setBucketPublic: awsSetBucketPublic,
      createBucket: awsCreateBucket,
      deleteBucket: awsDeleteBucket
    }
  } else if (cloudType === "tencent") {
    return {
      listBuckets: tencentListBuckets,
      listObjects: tencentListObjects,
      uploadObject: tencentUploadObject,
      downloadObject: tencentDownloadObject,
      deleteObject: tencentDeleteObject,
      setBucketPublic: tencentSetBucketPublic,
      createBucket: tencentCreateBucket,
      deleteBucket: tencentDeleteBucket
    }
  } else {
    return {
      listBuckets: aliyunListBuckets,
      listObjects: aliyunListObjects,
      uploadObject: aliyunUploadObject,
      downloadObject: aliyunDownloadObject,
      deleteObject: aliyunDeleteObject,
      setBucketPublic: aliyunSetBucketPublic,
      createBucket: aliyunCreateBucket,
      deleteBucket: aliyunDeleteBucket
    }
  }
}

// ========== 工具函数 ==========

/** 格式化文件大小 */
export function formatFileSize(bytes: number): string {
  if (!bytes || bytes === 0) return "0 B"
  const k = 1024
  const sizes = ["B", "KB", "MB", "GB", "TB"]
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return `${Math.round(bytes / k ** i * 100) / 100} ${sizes[i]}`
}

/** 生成对象访问 URL */
export function generateObjectUrl(
  cloudType: Types.CloudType,
  bucket: string,
  key: string,
  regionId?: string
): string {
  if (cloudType === "aws") {
    const region = regionId || "us-east-1"
    return `https://${bucket}.s3.${region}.amazonaws.com/${key}`
  } else if (cloudType === "tencent") {
    // 腾讯云 COS URL 格式: https://{bucket}-{appid}.cos.{region}.myqcloud.com/{key}
    // 由于 bucket 名称已包含 appid，直接使用
    const region = regionId || "ap-guangzhou"
    return `https://${bucket}.cos.${region}.myqcloud.com/${key}`
  } else {
    const region = regionId || "cn-hangzhou"
    return `https://${bucket}.oss-${region}.aliyuncs.com/${key}`
  }
}
