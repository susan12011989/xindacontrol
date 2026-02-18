import type * as RO from "./type"
import { request } from "@/http/axios"

// ========== 标签 ==========

export function getTagList() {
  return request<RO.TagListResponseData>({
    url: "resource-overview/tags",
    method: "get"
  })
}

export function createTag(data: RO.ResourceTagReq) {
  return request({
    url: "resource-overview/tags",
    method: "post",
    data
  })
}

export function updateTag(id: number, data: RO.ResourceTagReq) {
  return request({
    url: `resource-overview/tags/${id}`,
    method: "put",
    data
  })
}

export function deleteTag(id: number) {
  return request({
    url: `resource-overview/tags/${id}`,
    method: "delete"
  })
}

// ========== 标签分配 ==========

export function assignTags(data: RO.AssignTagsReq) {
  return request({
    url: "resource-overview/tags/assign",
    method: "post",
    data
  })
}

export function removeTags(data: RO.RemoveTagsReq) {
  return request({
    url: "resource-overview/tags/remove",
    method: "post",
    data
  })
}

// ========== 全局列表 ==========

export function getGlobalOssConfigs(params: RO.QueryGlobalOssConfigsReq) {
  return request<RO.GlobalOssConfigsResponseData>({
    url: "resource-overview/oss-configs",
    method: "get",
    params
  })
}

export function getGlobalGostServers(params: RO.QueryGlobalGostServersReq) {
  return request<RO.GlobalGostServersResponseData>({
    url: "resource-overview/gost-servers",
    method: "get",
    params
  })
}

// ========== 批量操作 ==========

export function batchSyncGostIP(data: RO.BatchSyncGostIPByFilterReq) {
  return request<RO.BatchSyncResponseData>({
    url: "resource-overview/batch-sync-gost-ip",
    method: "post",
    data
  })
}

// ========== OSS 健康检测 ==========

export function checkOssHealth(data: RO.CheckOssHealthReq) {
  return request<RO.OssHealthCheckResponseData>({
    url: "resource-overview/check-oss-health",
    method: "post",
    data
  })
}

export function checkMerchantOssHealth(merchantId: number) {
  return request<RO.OssHealthCheckResponseData>({
    url: `merchant/${merchantId}/check-oss-health`,
    method: "post"
  })
}
