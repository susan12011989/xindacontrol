import type * as MerchantStorage from "./type"
import { request } from "@/http/axios"

// 查询商户存储配置列表
export function getMerchantStorageList(params: MerchantStorage.QueryMerchantStorageReq) {
  return request<MerchantStorage.QueryMerchantStorageResponseData>({
    url: "merchant-storage",
    method: "get",
    params
  })
}

// 获取商户存储配置详情
export function getMerchantStorageDetail(id: number) {
  return request<MerchantStorage.MerchantStorageDetailResponseData>({
    url: `merchant-storage/${id}`,
    method: "get"
  })
}

// 创建商户存储配置
export function createMerchantStorage(data: MerchantStorage.MerchantStorageReq) {
  return request({
    url: "merchant-storage",
    method: "post",
    data
  })
}

// 更新商户存储配置
export function updateMerchantStorage(id: number, data: MerchantStorage.MerchantStorageReq) {
  return request({
    url: `merchant-storage/${id}`,
    method: "put",
    data
  })
}

// 删除商户存储配置
export function deleteMerchantStorage(id: number) {
  return request({
    url: `merchant-storage/${id}`,
    method: "delete"
  })
}

// 推送存储配置到商户服务器
export function pushMerchantStorage(data: MerchantStorage.PushStorageConfigReq) {
  return request<MerchantStorage.PushStorageResultResponseData>({
    url: "merchant-storage/push",
    method: "post",
    data
  })
}

// 获取存储类型选项
export function getStorageTypes() {
  return request<MerchantStorage.StorageTypesResponseData>({
    url: "merchant-storage/types",
    method: "get"
  })
}
