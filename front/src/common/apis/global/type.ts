import type { ApiResponseData } from "@/common/apis/type"

// 基础分页类型
export interface Pagination {
  page: number
  size: number
}

// ========== OSS URL 管理 ==========

// 查询OSS URL请求
export interface QueryOssUrlReq extends Pagination {
  url?: string
}

// 创建OSS URL请求
export interface CreateOssUrlReq {
  url: string
}

// 更新OSS URL请求
export interface UpdateOssUrlReq {
  url: string
}

// OSS URL响应
export interface OssUrlResp {
  id: number
  url: string
  updated_at: string
}

// OSS URL列表响应
export interface QueryOssUrlResponse {
  list: OssUrlResp[]
  total: number
}

// API响应类型
export type QueryOssUrlResponseData = ApiResponseData<QueryOssUrlResponse>
export type OssUrlDetailResponseData = ApiResponseData<OssUrlResp>
