import type * as Monitor from "./type"
import { request } from "@/http/axios"

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
