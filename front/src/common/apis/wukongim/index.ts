import type { ApiResponseData } from "@/common/apis/type"
import type * as WuKongIM from "./type"
import { request } from "@/http/axios"

// ========== WuKongIM 监控 ==========

/** 获取可用的 WuKongIM 节点列表 */
export function getWuKongIMNodes() {
  return request<ApiResponseData<WuKongIM.WuKongIMNode[]>>({
    url: "wukongim/nodes",
    method: "get"
  })
}

/** 获取系统变量（连接数/CPU/内存/消息量） */
export function getWuKongIMVarz(server_id: number) {
  return request<WuKongIM.WuKongIMVarzResponseData>({
    url: "wukongim/varz",
    method: "get",
    params: { server_id }
  })
}

/** 获取连接详情（支持过滤/排序/分页） */
export function getWuKongIMConnz(params: WuKongIM.WuKongIMConnzReq) {
  return request<WuKongIM.WuKongIMConnzResponseData>({
    url: "wukongim/connz",
    method: "get",
    params
  })
}

/** 查询用户在线状态 */
export function getWuKongIMOnlineStatus(data: WuKongIM.WuKongIMOnlineStatusReq) {
  return request<WuKongIM.WuKongIMOnlineStatusResponseData>({
    url: "wukongim/user/onlinestatus",
    method: "post",
    data
  })
}

/** 强制设备下线 */
export function wukongimDeviceQuit(data: WuKongIM.WuKongIMDeviceQuitReq) {
  return request<ApiResponseData<null>>({
    url: "wukongim/user/device_quit",
    method: "post",
    data
  })
}
