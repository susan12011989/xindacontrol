import type * as Auth from "./type"
import { request } from "@/http/axios"

/** 获取登录验证码 */
// export function getCaptchaApi() {
//   return request<Auth.CaptchaResponseData>({
//     url: "auth/captcha",
//     method: "get"
//   })
// }

/** 登录并返回 Token */
export function loginApi(data: Auth.LoginRequestData) {
  return request<Auth.LoginResponseData>({
    url: "auth/login",
    method: "post",
    data
  })
}

/** 获取一次性挑战 nonce 与公钥 */
export function getChallengeApi() {
  return request<ApiResponseData<Auth.ChallengeResponseData>>({
    url: "auth/challenge",
    method: "get"
  })
}

/** 加密登录 */
export function loginEncryptedApi(data: Auth.EncryptedLoginRequestData) {
  return request<Auth.LoginResponseData>({
    url: "auth/login",
    method: "post",
    data
  })
}
