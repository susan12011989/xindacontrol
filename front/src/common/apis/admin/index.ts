import type * as Types from "./type"
import { request } from "@/http/axios"

/** 修改当前管理员密码 */
export function updatePassword(data: Types.UpdatePasswordRequestData) {
  return request<Types.UpdatePasswordResponseData>({
    url: "server/update-password",
    method: "post",
    data
  })
}

/** 测试接口连接 */
export function ping() {
  return request<Types.Pong>({
    url: "ping",
    method: "get"
  })
}
