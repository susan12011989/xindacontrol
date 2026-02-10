import type { ApiResponseData } from "@/common/apis/type"

// ========== Docker 容器管理 ==========

// 分页
export interface Pagination {
  page: number
  size: number
}

// 查询容器列表请求
export interface QueryDockerContainersReq {
  server_id: number
  status?: "all" | "running" | "exited"
  name?: string
}

// 容器信息响应
export interface DockerContainerResp {
  container_id: string
  name: string
  image: string
  status: string
  state: string
  ports: string
  created_at: string
}

// 服务器信息
export interface ServerInfo {
  id: number
  merchant_id: number
  name: string
  host: string
  merchant_name: string
}

// 容器列表响应
export interface QueryDockerContainersResponse {
  list: DockerContainerResp[]
  total: number
  server_info: ServerInfo
}

// 容器资源使用情况
export interface DockerContainerStatsResp {
  container_id: string
  name: string
  cpu_perc: string
  mem_usage: string
  mem_perc: string
  net_io: string
  block_io: string
  pids: string
}

// 容器日志请求
export interface GetDockerLogsReq {
  server_id: number
  container_id: string
  lines?: number
  since?: string
  until?: string
  follow?: boolean
  timestamps?: boolean
}

// 容器日志响应
export interface GetDockerLogsResponse {
  logs: string
  total_lines: number
  container_id: string
  container_name: string
}

// 容器操作请求
export interface DockerContainerOperationReq {
  server_id: number
  container_id: string
  action: "start" | "stop" | "restart" | "remove"
  force?: boolean
}

// 批量操作请求
export interface DockerBatchOperationReq {
  server_id: number
  container_ids: string[]
  action: "start" | "stop" | "restart" | "remove"
  force?: boolean
}

// 操作结果响应
export interface DockerOperationResponse {
  success: boolean
  message: string
  results?: DockerOperationResult[]
}

// 单个操作结果
export interface DockerOperationResult {
  container_id: string
  name: string
  success: boolean
  message: string
}

// 查询 Docker 操作历史
export interface QueryDockerHistoryReq extends Pagination {
  server_id?: number
  merchant_id?: number
  container_id?: string
  action?: string
}

// Docker 操作历史响应
export interface DockerHistoryResp {
  id: number
  server_name: string
  merchant_name: string
  container_id: string
  container_name: string
  action: string
  operator: string
  status: number
  output: string
  error_msg: string
  created_at: string
}

// Docker 操作历史列表响应
export interface QueryDockerHistoryResponse {
  list: DockerHistoryResp[]
  total: number
}

// ========== 服务器健康检查 ==========

// 健康检查请求
export interface HealthCheckReq {
  server_id: number
}

// 单项健康检查结果
export interface HealthCheckItem {
  name: string
  status: "ok" | "error" | "warning"
  message: string
  latency: number
  action?: "restart" | "start" | "deploy" | "none"  // 建议操作
  action_label?: string                              // 操作按钮文案
  container_name?: string                            // 关联的容器名称
}

// 健康检查响应
export interface HealthCheckResponse {
  server_id: number
  server_name: string
  server_host: string
  check_time: string
  overall: "healthy" | "unhealthy" | "partial"
  services: HealthCheckItem[]
  apis: HealthCheckItem[]
}

// 批量健康检查请求
export interface BatchHealthCheckReq {
  server_ids: number[]
}

// 批量健康检查响应
export interface BatchHealthCheckResponse {
  results: HealthCheckResponse[]
  summary: {
    total: number
    healthy: number
    unhealthy: number
    partial: number
  }
}

// API 响应类型
export type QueryDockerContainersResponseData = ApiResponseData<QueryDockerContainersResponse>
export type GetDockerLogsResponseData = ApiResponseData<GetDockerLogsResponse>
export type DockerOperationResponseData = ApiResponseData<DockerOperationResponse>
export type QueryDockerHistoryResponseData = ApiResponseData<QueryDockerHistoryResponse>
export type HealthCheckResponseData = ApiResponseData<HealthCheckResponse>
export type BatchHealthCheckResponseData = ApiResponseData<BatchHealthCheckResponse>
