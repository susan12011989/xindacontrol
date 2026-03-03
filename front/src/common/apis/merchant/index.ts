import type * as Merchant from "./type"
import type { ApiResponseData } from "@@/apis/type"
import { request } from "@/http/axios"

export function getMerchantList(params: Merchant.QueryMerchantsReq) {
  return request<Merchant.QueryMerchantsResponseData>({
    url: "merchant",
    method: "get",
    params
  })
}

/** 获取商户关联的 GOST 服务器列表 */
export function listMerchantGostServers(merchantId: number) {
  return request<Merchant.MerchantGostServersResponseData>({
    url: `merchant/${merchantId}/gost-servers`,
    method: "get"
  })
}

/** 创建商户 GOST 服务器关联 */
export function createMerchantGostServer(merchantId: number, data: Merchant.CreateMerchantGostServerReq) {
  return request({
    url: `merchant/${merchantId}/gost-servers`,
    method: "post",
    data: { ...data, merchant_id: merchantId }
  })
}

/** 删除商户 GOST 服务器关联 */
export function deleteMerchantGostServer(relationId: number) {
  return request({
    url: `merchant/gost-servers/${relationId}`,
    method: "delete"
  })
}

// ========== 商户 OSS 配置 ==========

/** 获取商户 OSS 配置列表 */
export function listMerchantOssConfigs(merchantId: number) {
  return request<Merchant.MerchantOssConfigsResponseData>({
    url: `merchant/${merchantId}/oss-configs`,
    method: "get"
  })
}

/** 创建商户 OSS 配置 */
export function createMerchantOssConfig(merchantId: number, data: Merchant.CreateMerchantOssConfigReq) {
  return request({
    url: `merchant/${merchantId}/oss-configs`,
    method: "post",
    data: { ...data, merchant_id: merchantId }
  })
}

/** 更新商户 OSS 配置 */
export function updateMerchantOssConfig(configId: number, data: Merchant.UpdateMerchantOssConfigReq) {
  return request({
    url: `merchant/oss-configs/${configId}`,
    method: "put",
    data
  })
}

/** 删除商户 OSS 配置 */
export function deleteMerchantOssConfig(configId: number) {
  return request({
    url: `merchant/oss-configs/${configId}`,
    method: "delete"
  })
}

// ========== TURN 服务器配置管理 ==========

/** 获取所有商户 TURN 配置列表 */
export function listMerchantTurnConfigs(params?: { name?: string }) {
  return request<Merchant.MerchantTurnConfigsResponseData>({
    url: "merchant/turn-configs",
    method: "get",
    params
  })
}

/** 更新单个商户 TURN 服务器 */
export function updateMerchantTurnServer(
  merchantId: number,
  data: { turn_server: string; turn_username?: string; turn_credential?: string }
) {
  return request<ApiResponseData<Merchant.BatchTurnUpdateResult>>({
    url: `merchant/${merchantId}/turn-server`,
    method: "put",
    data
  })
}

/** 批量更新商户 TURN 服务器 */
export function batchUpdateTurnServer(data: Merchant.BatchUpdateTurnServerReq) {
  return request<Merchant.BatchUpdateTurnServerResponseData>({
    url: "merchant/batch-turn-update",
    method: "post",
    data,
    timeout: 120000
  })
}
