import type * as Types from "./type"
import { request } from "@/http/axios"

/** 获取系统服务器IP列表 */
export function getSystemIPs() {
  return request<ApiResponseData<Types.GetSystemIPsResp>>({
    url: "ip-embed/system-ips",
    method: "get"
  })
}

/** 获取上传目标配置 */
export function getTargets() {
  return request<ApiResponseData<Types.GetTargetsResp>>({
    url: "ip-embed/targets",
    method: "get"
  })
}

/** 获取源文件列表 */
export function getSourceFiles() {
  return request<ApiResponseData<Types.GetSourceFilesResp>>({
    url: "ip-embed/source-files",
    method: "get"
  })
}

/** 执行批量嵌入并上传 */
export function executeEmbedAndUpload(data: Types.ExecuteEmbedReq) {
  return request<ApiResponseData<Types.ExecuteEmbedResp>>({
    url: "ip-embed/execute",
    method: "post",
    data,
    timeout: 300000 // 5分钟超时
  })
}

/** 获取上次选中的IP列表 */
export function getSelectedIPs() {
  return request<ApiResponseData<Types.GetSelectedIPsResp>>({
    url: "ip-embed/selected-ips",
    method: "get"
  })
}

/** 保存选中的IP列表 */
export function saveSelectedIPs(data: Types.SaveSelectedIPsReq) {
  return request<ApiResponseData<null>>({
    url: "ip-embed/selected-ips",
    method: "post",
    data
  })
}

/** 创建上传目标 */
export function createTarget(data: Types.CreateTargetReq) {
  return request<ApiResponseData<{ id: number }>>({
    url: "ip-embed/targets",
    method: "post",
    data
  })
}

/** 更新上传目标 */
export function updateTarget(id: number, data: Types.UpdateTargetReq) {
  return request<ApiResponseData<null>>({
    url: `ip-embed/targets/${id}`,
    method: "put",
    data
  })
}

/** 删除上传目标 */
export function deleteTarget(id: number) {
  return request<ApiResponseData<null>>({
    url: `ip-embed/targets/${id}`,
    method: "delete"
  })
}

/** 切换目标启用状态 */
export function toggleTarget(id: number) {
  return request<ApiResponseData<null>>({
    url: `ip-embed/targets/${id}/toggle`,
    method: "put"
  })
}

// ============ 资源分组 ============

/** 获取资源分组列表 */
export function getResourceGroups(type = "ip_embed_target") {
  return request<ApiResponseData<Types.ResourceGroupItem[]>>({
    url: "resource-groups",
    method: "get",
    params: { type }
  })
}

/** 创建资源分组 */
export function createResourceGroup(data: Types.ResourceGroupReq, type = "ip_embed_target") {
  return request<ApiResponseData<{ id: number }>>({
    url: "resource-groups",
    method: "post",
    params: { type },
    data
  })
}

/** 更新资源分组 */
export function updateResourceGroup(id: number, data: Types.ResourceGroupReq) {
  return request<ApiResponseData<null>>({
    url: `resource-groups/${id}`,
    method: "put",
    data
  })
}

/** 删除资源分组 */
export function deleteResourceGroup(id: number, type = "ip_embed_target") {
  return request<ApiResponseData<null>>({
    url: `resource-groups/${id}`,
    method: "delete",
    params: { type }
  })
}
