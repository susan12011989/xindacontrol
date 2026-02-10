import type * as Types from "./type"
import { request } from "@/http/axios"

/** 获取通知列表 */
export function getNotifyListApi() {
  return request<Types.NotifyResponseData>({
    url: "notify",
    method: "get"
  })
}
