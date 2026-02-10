import type * as CloudAccount from "./type"
import { request } from "@/http/axios"

/** 获取云账号列表 */
export function getCloudAccountList(params: CloudAccount.QueryCloudAccountsReq) {
  return request<CloudAccount.QueryCloudAccountsResponseData>({
    url: "cloud_account",
    method: "get",
    params
  })
}

/** 获取云账号详情 */
export function getCloudAccountDetail(id: number) {
  return request<CloudAccount.CloudAccountDetailResponseData>({
    url: `cloud_account/${id}`,
    method: "get"
  })
}

/** 创建云账号 */
export function createCloudAccount(data: CloudAccount.CreateCloudAccountReq) {
  return request({
    url: "cloud_account",
    method: "post",
    data
  })
}

/** 更新云账号 */
export function updateCloudAccount(id: number, data: CloudAccount.UpdateCloudAccountReq) {
  return request({
    url: `cloud_account/${id}`,
    method: "put",
    data
  })
}

/** 删除云账号 */
export function deleteCloudAccount(id: number) {
  return request({
    url: `cloud_account/${id}`,
    method: "delete"
  })
}

/** 获取云账号选项（用于下拉框） */
export function getCloudAccountOptions(cloud_type?: string) {
  return request<CloudAccount.CloudAccountOptionsResponseData>({
    url: "cloud_account/options",
    method: "get",
    params: { cloud_type }
  })
}

/** 查询阿里云账户余额（支持 merchant_id 或 cloud_account_id） */
export function getAliyunBalance(params: { merchant_id?: number, cloud_account_id?: number }) {
  return request<{ code: number, data: CloudAccount.AliyunBalanceResp, message: string }>({
    url: "cloud/account/balance",
    method: "get",
    params
  })
}
