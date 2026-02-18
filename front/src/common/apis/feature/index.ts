import type * as Feature from "./type"
import { request } from "@/http/axios"

// ========== 功能开关管理 ==========

/** 获取功能定义列表 */
export function getFeatureDefinitions() {
  return request<Feature.FeatureDefinitionsResponseData>({
    url: "feature/definitions",
    method: "get"
  })
}

/** 获取商户功能开关列表 */
export function getFeatureFlags(merchant_id: number) {
  return request<Feature.QueryFeatureFlagsResponseData>({
    url: "feature/flags",
    method: "get",
    params: { merchant_id }
  })
}

/** 更新单个功能开关 */
export function updateFeatureFlag(data: Feature.UpdateFeatureFlagReq) {
  return request<Feature.FeatureFlagOperationResponseData>({
    url: "feature/flags",
    method: "put",
    data
  })
}

/** 批量更新功能开关 */
export function batchUpdateFeatureFlags(data: Feature.BatchUpdateFeatureFlagsReq) {
  return request<Feature.FeatureFlagOperationResponseData>({
    url: "feature/flags/batch",
    method: "put",
    data
  })
}

/** 初始化商户功能开关 */
export function initFeatureFlags(data: Feature.InitFeatureFlagsReq) {
  return request<Feature.FeatureFlagOperationResponseData>({
    url: "feature/flags/init",
    method: "post",
    data
  })
}

/** 检查功能是否启用 */
export function checkFeature(merchant_id: number, feature_name: string) {
  return request<Feature.CheckFeatureResponseData>({
    url: "feature/check",
    method: "get",
    params: { merchant_id, feature_name }
  })
}
