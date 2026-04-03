import { request, createStreamRequest } from "@/http/axios"
import type * as Control from "./type"

// 获取控制模式
export function getControlModeApi() {
  return request<Control.ModeResp>({ url: "control/mode", method: "get" })
}

// 健康检查
export function controlHealthCheckApi(serverId?: number) {
  return request<Control.HealthCheckResp>({
    url: "control/health",
    method: "get",
    params: serverId ? { server_id: serverId } : {}
  })
}

// 服务操作
export function controlServiceActionApi(data: Control.ServiceActionReq) {
  return request({ url: "control/service/action", method: "post", data })
}

// 获取服务状态
export function controlServiceStatusApi(params: { server_id?: number; service_name?: string }) {
  return request<Control.ServiceStatusResp>({ url: "control/service/status", method: "get", params })
}

// 获取服务日志
export function controlServiceLogsApi(params: { server_id?: number; service_name: string; lines?: number }) {
  return request<Control.ServiceLogsResp>({ url: "control/service/logs", method: "get", params })
}

// 获取服务器资源
export function controlServerStatsApi(serverId?: number) {
  return request<Control.ServerStatsResp>({
    url: "control/stats",
    method: "get",
    params: serverId ? { server_id: serverId } : {}
  })
}

// GOST 一键部署（流式）
export function controlGostOneClickDeployApi(data?: any, onData?: (data: any, isComplete?: boolean) => void, onError?: (error: any) => void) {
  return createStreamRequest({ url: "control/gost/one-click-deploy", method: "post", data: data || {} }, onData || (() => {}), onError)
}

// 获取服务端点
export function controlEndpointsApi(serviceName: string) {
  return request<Control.EndpointsResp>({
    url: "control/endpoints",
    method: "get",
    params: { service_name: serviceName }
  })
}
