import type * as Types from "./type"
import { request } from "@/http/axios"

/** 查询商户列表 */
export function merchantQueryApi(data: Types.MerchantQueryRequestData) {
  return request<Types.MerchantQueryResponseData>({
    url: "merchant",
    method: "get",
    params: data
  })
}

/** 创建商户 */
export function createMerchantApi(data: Types.CreateOrEditMerchantRequestData) {
  return request<Types.CreateOrEditMerchantResponseData>({
    url: "merchant",
    method: "post",
    data
  })
}

/** 更新商户 */
export function updateMerchantApi(data: Types.CreateOrEditMerchantRequestData) {
  return request<Types.CreateOrEditMerchantResponseData>({
    url: `merchant/${data.id}`,
    method: "put",
    data
  })
}

/** 删除商户 */
export function deleteMerchantApi(id: number) {
  return request({
    url: `merchant/${id}`,
    method: "delete"
  })
}

/** 获取云账号余额 */
export function getBalanceApi(data: Types.BalanceReq) {
  return request<Types.BalanceResponseData>({
    url: "merchant/balance-cloud",
    method: "get",
    params: data
  })
}

/** 隧道连接检测 */
export function tunnelCheckApi(params: Types.TunnelCheckReq) {
  return request<Types.TunnelCheckResponseData>({
    url: "merchant/tunnel-check",
    method: "get",
    params
  })
}

/** 更换商户IP */
export function changeMerchantIPApi(id: number) {
  return request<Types.ChangeIPResponseData>({
    url: `merchant/${id}/change-ip`,
    method: "post"
  })
}

/** 更换商户 GOST 隧道端口 */
export function changeMerchantGostPortApi(id: number, gostPort: number) {
  return request<Types.ChangeGostPortResponseData>({
    url: `merchant/${id}/change-gost-port`,
    method: "post",
    data: { gost_port: gostPort }
  })
}

/** 获取商户短信配置（返回原始JSON，需自行解析） */
export function getAdminmSmsConfigApi(merchant_no: string) {
  return request<Types.AdminmSmsConfigResponseData>({
    url: "merchant/adminm_config/sms",
    method: "get",
    params: { merchant_no }
  })
}

/** 保存商户短信配置（单个/批量/全部） */
export function saveAdminmSmsConfigApi(data: Types.AdminmSmsSaveReq) {
  return request<Types.AdminmSaveResponseData>({
    url: "merchant/adminm_config/sms",
    method: "post",
    data
  })
}

/** 保存敏感词（从txt文本解析，单个/批量/全部） */
export function saveAdminmSensitiveContentsApi(data: Types.AdminmSensitiveSaveReq) {
  return request<Types.AdminmSaveResponseData>({
    url: "merchant/adminm_config/sensitive_contents",
    method: "post",
    data
  })
}

/** 保存系统用户昵称（单个/批量/全部） */
export function saveAdminmNicknameApi(data: Types.AdminmNicknameSaveReq) {
  return request<Types.AdminmSaveResponseData>({
    url: "merchant/adminm_config/system_user_nickname",
    method: "post",
    data
  })
}

/** 获取隧道统计 */
export function getTunnelStatsApi() {
  return request<Types.TunnelStatsResponseData>({
    url: "merchant/tunnel-stats",
    method: "get"
  })
}

/** 清除商户数据 */
export function clearMerchantDataApi(merchant_no: string) {
  return request<Types.AdminmSaveResponseData>({
    url: "merchant/adminm_config/clear_data",
    method: "post",
    data: { merchant_no }
  })
}
