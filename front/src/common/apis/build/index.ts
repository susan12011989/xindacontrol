import { request } from "@/http/axios"
import type { ApiResponseData } from "@/common/apis/type"
import type * as Build from "./type"

// ========== 商户配置 ==========

/** 获取商户配置列表 */
export function getBuildMerchants(params: Build.MerchantListReq) {
  return request<ApiResponseData<Build.ListResp<Build.BuildMerchant>>>({
    url: "/build/merchants",
    method: "get",
    params,
  })
}

/** 获取商户配置详情 */
export function getBuildMerchant(id: number) {
  return request<Build.BuildMerchant>({
    url: `/build/merchants/${id}`,
    method: "get",
  })
}

/** 创建商户配置 */
export function createBuildMerchant(data: Build.BuildMerchantReq) {
  return request<Build.BuildMerchant>({
    url: "/build/merchants",
    method: "post",
    data,
  })
}

/** 更新商户配置 */
export function updateBuildMerchant(id: number, data: Build.BuildMerchantReq) {
  return request({
    url: `/build/merchants/${id}`,
    method: "put",
    data,
  })
}

/** 删除商户配置 */
export function deleteBuildMerchant(id: number) {
  return request({
    url: `/build/merchants/${id}`,
    method: "delete",
  })
}

/** 上传图标 */
export function uploadBuildIcon(id: number, file: File) {
  const formData = new FormData()
  formData.append("file", file)
  return request<{ url: string }>({
    url: `/build/merchants/${id}/icon`,
    method: "post",
    data: formData,
    headers: { "Content-Type": "multipart/form-data" },
  })
}

// ========== 构建任务 ==========

/** 获取构建任务列表 */
export function getBuildTasks(params: Build.TaskListReq) {
  return request<ApiResponseData<Build.ListResp<Build.BuildTask>>>({
    url: "/build/tasks",
    method: "get",
    params,
  })
}

/** 获取构建任务详情 */
export function getBuildTask(id: number) {
  return request<{ task: Build.BuildTask; artifacts: Build.BuildArtifact[] }>({
    url: `/build/tasks/${id}`,
    method: "get",
  })
}

/** 创建构建任务 */
export function createBuildTask(data: Build.CreateTaskReq) {
  return request<Build.BuildTask>({
    url: "/build/tasks",
    method: "post",
    data,
  })
}

/** 取消构建任务 */
export function cancelBuildTask(id: number) {
  return request({
    url: `/build/tasks/${id}/cancel`,
    method: "post",
  })
}

/** 重试构建任务 */
export function retryBuildTask(id: number) {
  return request<Build.BuildTask>({
    url: `/build/tasks/${id}/retry`,
    method: "post",
  })
}

/** 获取构建任务实时进度 */
export function getTaskProgress(id: number) {
  return request<Build.BuildTaskProgress>({
    url: `/build/tasks/${id}/progress`,
    method: "get",
  })
}

// ========== 产物管理 ==========

/** 获取构建产物列表 */
export function getBuildArtifacts(params: Build.ArtifactListReq) {
  return request<Build.ListResp<Build.BuildArtifact>>({
    url: "/build/artifacts",
    method: "get",
    params,
  })
}

/** 下载产物 */
export function downloadBuildArtifact(id: number) {
  return `/server/v1/build/artifacts/${id}/download`
}

/** 清理过期产物 */
export function cleanExpiredArtifacts() {
  return request<{ deleted_count: number }>({
    url: "/build/artifacts/expired",
    method: "delete",
  })
}

// ========== 统计 ==========

/** 获取构建统计 */
export function getBuildStats() {
  return request<ApiResponseData<Build.BuildStats>>({
    url: "/build/stats",
    method: "get",
  })
}

// ========== 构建服务器 ==========

/** 获取构建服务器列表 */
export function getBuildServers() {
  return request<Build.BuildServer[]>({
    url: "/build/servers",
    method: "get",
  })
}

/** 创建构建服务器 */
export function createBuildServer(data: Build.BuildServerReq) {
  return request<Build.BuildServer>({
    url: "/build/servers",
    method: "post",
    data,
  })
}

/** 更新构建服务器 */
export function updateBuildServer(id: number, data: Build.BuildServerReq) {
  return request({
    url: `/build/servers/${id}`,
    method: "put",
    data,
  })
}

/** 删除构建服务器 */
export function deleteBuildServer(id: number) {
  return request({
    url: `/build/servers/${id}`,
    method: "delete",
  })
}
