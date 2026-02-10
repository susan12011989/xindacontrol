import type * as T from "./type"
import { request } from "@/http/axios"

// 列表查询
export function queryAppLogs(params: T.AppLogsQueryReq) {
  return request<T.AppLogsQueryResp>({
    url: "merchant/app_logs",
    method: "get",
    params
  })
}
