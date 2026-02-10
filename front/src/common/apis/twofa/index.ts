import type { ApiResponseData } from "../type"
import type { TwoFADisableReq, TwoFAEnableReq, TwoFASetupResp, TwoFAStatusResp } from "./type"
import { request } from "@/http/axios"

// 获取2FA状态
export function getTwoFAStatusApi() {
  return request<ApiResponseData<TwoFAStatusResp>>({
    url: "/2fa/status",
    method: "GET"
  })
}

// 获取2FA设置信息（二维码等）
export function getTwoFASetupApi() {
  return request<ApiResponseData<TwoFASetupResp>>({
    url: "/2fa/setup",
    method: "GET"
  })
}

// 启用2FA
export function enableTwoFAApi(data: TwoFAEnableReq) {
  return request<ApiResponseData<{ message: string }>>({
    url: "/2fa/enable",
    method: "POST",
    data
  })
}

// 禁用2FA
export function disableTwoFAApi(data: TwoFADisableReq) {
  return request<ApiResponseData<{ message: string }>>({
    url: "/2fa/disable",
    method: "POST",
    data
  })
}
