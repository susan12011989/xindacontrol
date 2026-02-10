import type { ApiResponseData } from "@/common/apis/type"

// ========== 功能开关管理 ==========

// 功能定义
export interface FeatureDefinition {
  name: string
  label: string
  description: string
  category: string
}

// 功能开关响应
export interface FeatureFlagResp {
  id: number
  merchant_id: number
  feature_name: string
  label: string
  description: string
  category: string
  enabled: boolean
  created_at: string
  updated_at: string
}

// 查询功能开关请求
export interface QueryFeatureFlagsReq {
  merchant_id: number
}

// 功能开关列表响应
export interface QueryFeatureFlagsResponse {
  list: FeatureFlagResp[]
  merchant_id: number
}

// 更新功能开关请求
export interface UpdateFeatureFlagReq {
  merchant_id: number
  feature_name: string
  enabled: boolean
}

// 批量更新功能开关请求
export interface BatchUpdateFeatureFlagsReq {
  merchant_id: number
  features: FeatureFlagUpdateItem[]
}

export interface FeatureFlagUpdateItem {
  feature_name: string
  enabled: boolean
}

// 初始化功能开关请求
export interface InitFeatureFlagsReq {
  merchant_id: number
}

// 操作响应
export interface FeatureFlagOperationResponse {
  success: boolean
  message: string
}

// API 响应类型
export type FeatureDefinitionsResponseData = ApiResponseData<{ list: FeatureDefinition[] }>
export type QueryFeatureFlagsResponseData = ApiResponseData<QueryFeatureFlagsResponse>
export type FeatureFlagOperationResponseData = ApiResponseData<FeatureFlagOperationResponse>
