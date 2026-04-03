import type * as Merchant from "./type"
import { request } from "@/http/axios"

export function getMerchantList(params: Merchant.QueryMerchantsReq) {
  return request<Merchant.QueryMerchantsResponseData>({
    url: "merchant",
    method: "get",
    params
  })
}

// ========== GOST 服务器管理 ==========

export function getMerchantGostServers(merchantId: number) {
  return request<Merchant.MerchantGostServersResponseData>({
    url: `merchant/${merchantId}/gost-servers`,
    method: "get"
  })
}

export function createMerchantGostServer(merchantId: number, data: Merchant.MerchantGostServerReq) {
  return request<Merchant.CreateResponseData>({
    url: `merchant/${merchantId}/gost-servers`,
    method: "post",
    data
  })
}

export function deleteMerchantGostServer(relationId: number) {
  return request({
    url: `merchant/gost-servers/${relationId}`,
    method: "delete"
  })
}

export function importOssFromTargets(merchantId: number) {
  return request<any>({
    url: `merchant/${merchantId}/oss-configs/import-from-targets`,
    method: "post"
  })
}

export function reorderMerchantGostServers(merchantId: number, ids: number[]) {
  return request({
    url: `merchant/${merchantId}/gost-servers/reorder`,
    method: "post",
    data: { ids }
  })
}

// ========== OSS 配置管理 ==========

export function getMerchantOssConfigs(merchantId: number) {
  return request<Merchant.MerchantOssConfigsResponseData>({
    url: `merchant/${merchantId}/oss-configs`,
    method: "get"
  })
}

export function createMerchantOssConfig(merchantId: number, data: Merchant.MerchantOssConfigReq) {
  return request<Merchant.CreateResponseData>({
    url: `merchant/${merchantId}/oss-configs`,
    method: "post",
    data
  })
}

export function deleteMerchantOssConfig(configId: number) {
  return request({
    url: `merchant/oss-configs/${configId}`,
    method: "delete"
  })
}

// ========== GOST IP 同步 ==========

export function syncMerchantGostIP(merchantId: number, data?: any) {
  return request<Merchant.SyncGostIPResponseData>({
    url: `merchant/${merchantId}/sync-gost-ip`,
    method: "post",
    data
  })
}
