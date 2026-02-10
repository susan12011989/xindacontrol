import type * as Docker from "./type"
import { request } from "@/http/axios"

// ========== Docker 容器管理 ==========

/** 获取容器列表 */
export function getContainerList(params: Docker.QueryDockerContainersReq) {
  return request<Docker.QueryDockerContainersResponseData>({
    url: "docker/containers",
    method: "get",
    params
  })
}

/** 获取容器资源使用情况 */
export function getContainerStats(server_id: number) {
  return request<{ list: Docker.DockerContainerStatsResp[] }>({
    url: "docker/containers/stats",
    method: "get",
    params: { server_id }
  })
}

/** 获取容器日志 */
export function getContainerLogs(params: Docker.GetDockerLogsReq) {
  return request<Docker.GetDockerLogsResponseData>({
    url: "docker/logs",
    method: "get",
    params
  })
}

/** 容器操作 */
export function operateContainer(data: Docker.DockerContainerOperationReq) {
  return request<Docker.DockerOperationResponseData>({
    url: "docker/containers/operate",
    method: "post",
    data
  })
}

/** 批量操作容器 */
export function batchOperateContainers(data: Docker.DockerBatchOperationReq) {
  return request<Docker.DockerOperationResponseData>({
    url: "docker/containers/batch-operate",
    method: "post",
    data
  })
}

/** 查询Docker操作历史 */
export function getDockerHistory(params: Docker.QueryDockerHistoryReq) {
  return request<Docker.QueryDockerHistoryResponseData>({
    url: "docker/history",
    method: "get",
    params
  })
}

// ========== 服务器健康检查 ==========

/** 单个服务器健康检查 */
export function checkServerHealth(server_id: number) {
  return request<Docker.HealthCheckResponseData>({
    url: "docker/health",
    method: "get",
    params: { server_id }
  })
}

/** 批量服务器健康检查 */
export function batchCheckServerHealth(data: Docker.BatchHealthCheckReq) {
  return request<Docker.BatchHealthCheckResponseData>({
    url: "docker/health/batch",
    method: "post",
    data
  })
}
