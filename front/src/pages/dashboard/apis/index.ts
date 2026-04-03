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

/** 导出商户数据库（SQL dump） */
export function exportMerchantDatabaseApi(merchant_no: string) {
  return request<Blob>({
    url: "merchant/adminm_config/export_database",
    method: "post",
    data: { merchant_no },
    responseType: "blob",
    timeout: 300000 // 5分钟（大数据库需要时间）
  })
}

/** 推送 Logo + 应用名称到商户 Web */
export function pushWebLogoApi(data: Types.PushLogoReq) {
  return request<Types.PushLogoResponseData>({
    url: "merchant/adminm_config/push_logo",
    method: "post",
    data,
    timeout: 120000
  })
}

/** 清除商户数据（需要密码或2FA验证） */
export function clearMerchantDataApi(merchant_no: string, password?: string, totp_code?: string) {
  return request<Types.AdminmSaveResponseData>({
    url: "merchant/adminm_config/clear_data",
    method: "post",
    data: { merchant_no, password, totp_code },
    timeout: 120000 // 2分钟超时（包含SSH清理）
  })
}

// ========== 商户服务节点（单机/多机部署） ==========

/** 获取商户服务节点列表 */
export function listServiceNodesApi(merchantId: number) {
  return request<Types.ServiceNodeListResponseData>({
    url: `merchant/${merchantId}/service-nodes`,
    method: "get"
  })
}

/** 创建商户服务节点 */
export function createServiceNodeApi(merchantId: number, data: Types.ServiceNodeReq) {
  return request<Types.ServiceNodeCreateResponseData>({
    url: `merchant/${merchantId}/service-nodes`,
    method: "post",
    data
  })
}

/** 更新商户服务节点 */
export function updateServiceNodeApi(nodeId: number, data: Types.ServiceNodeReq) {
  return request({
    url: `merchant/service-nodes/${nodeId}`,
    method: "put",
    data
  })
}

/** 删除商户服务节点 */
export function deleteServiceNodeApi(nodeId: number) {
  return request({
    url: `merchant/service-nodes/${nodeId}`,
    method: "delete"
  })
}

/** 切换到多机模式 */
export function switchToClusterModeApi(merchantId: number, data: Types.SwitchClusterReq) {
  return request({
    url: `merchant/${merchantId}/switch-cluster`,
    method: "post",
    data
  })
}
