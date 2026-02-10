import type * as T from "./type"
import { request } from "@/http/axios"

// 列表查询
export function queryAdminmUsers(params: T.AdminmUsersQueryReq) {
  return request<T.AdminmUsersQueryResp>({
    url: "merchant/adminm_users",
    method: "get",
    params
  })
}

// 创建
export function createAdminmUser(data: T.CreateAdminmUserReq) {
  return request<{ code: number, data: null, message: string }>({
    url: "merchant/adminm_users",
    method: "post",
    data
  })
}

// 更新
export function updateAdminmUser(data: T.UpdateAdminmUserReq) {
  return request<{ code: number, data: null, message: string }>({
    url: "merchant/adminm_users",
    method: "put",
    data
  })
}

// 删除
export function deleteAdminmUser(data: T.DeleteAdminmUserReq) {
  return request<{ code: number, data: null, message: string }>({
    url: "merchant/adminm_users",
    method: "delete",
    data
  })
}

// 活跃数据
export function queryAdminmActive(params: T.AdminmActiveQueryReq) {
  return request<T.AdminmActiveRespData>({
    url: "merchant/adminm_users/active",
    method: "get",
    params
  })
}
