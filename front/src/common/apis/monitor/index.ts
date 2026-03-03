import type * as Monitor from "./type"
import type { ApiResponseData } from "@/common/apis/type"
import { createStreamRequest, request } from "@/http/axios"

/** 触发 GOST 健康检查 */
export function checkGostServers() {
  return request<{ list: Monitor.GostCheckResult[] }>({
    url: "monitor/gost/check",
    method: "post"
  })
}

/** 查询 GOST 监控历史日志 */
export function getGostMonitorLogs(params: Monitor.QueryMonitorLogsReq) {
  return request<{ list: Monitor.GostMonitorLog[]; total: number }>({
    url: "monitor/gost/logs",
    method: "get",
    params
  })
}

/** 带宽测速（流式API） */
export function runBandwidthTest(data: { server_id: number }, onData: (chunk: any, isComplete?: boolean) => void, onError?: (err: any) => void) {
  return createStreamRequest({
    url: "monitor/gost/bandwidth-test",
    method: "post",
    data,
    timeout: 600000
  }, onData, onError)
}

/** 查询服务器 ECS 实例带宽 */
export function getServerBandwidth(params: { server_id: number }) {
  return request<ApiResponseData<Monitor.BandwidthInfoResp>>({
    url: "cloud/ecs/instance/bandwidth",
    method: "get",
    params
  })
}

/** 修改服务器 ECS 实例带宽 */
export function modifyServerBandwidth(data: { server_id: number; internet_max_bandwidth_out: number }) {
  return request({
    url: "cloud/ecs/instance/bandwidth",
    method: "post",
    data
  })
}
