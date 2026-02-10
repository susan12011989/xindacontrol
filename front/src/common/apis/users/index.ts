import type * as Users from "./type"
import { request } from "@/http/axios"

/** 获取当前登录用户详情 */
export function getCurrentUserApi() {
  return request<Users.CurrentUserResponseData>({
    url: "auth/me",
    method: "get"
  })
}

/** 获取用户列表 */
export function getUserList(params: Users.QueryUsersReq) {
  return request<Users.QueryUsersResponseData>({
    url: "users",
    method: "get",
    params
  })
}

/** 获取用户详情 */
export function getUserDetail(id: number) {
  return request<Users.UserDetailResponseData>({
    url: `users/${id}`,
    method: "get"
  })
}

/** 更新用户 */
export function updateUser(id: number, data: Users.UpdateUserReq) {
  return request({
    url: `users/${id}`,
    method: "put",
    data
  })
}

/** 获取用户统计 */
export function getUserStats() {
  return request<Users.UserStatsResponseData>({
    url: "users/stats",
    method: "get"
  })
}

/** 封禁/解禁用户 */
export function toggleBanUser(data: Users.ToggleBanUserReq) {
  return request({
    url: "users/toggleban",
    method: "post",
    data
  })
}

/** 设置用户密码 */
export function setUserPassword(id: number, data: Users.SetUserPasswordReq) {
  return request({
    url: `users/${id}/password`,
    method: "post",
    data
  })
}

/** 获取用户历史使用IP */
export function getUserUsedIps(params: Users.QueryUserUsedIpsReq) {
  const { user_id, ...rest } = params
  return request<Users.UserUsedIpListResponseData>({
    url: `users/${user_id}/used_ips`,
    method: "get",
    params: rest
  })
}

/** IP 黑名单：列表 */
export function getIpBlacklist(params: Users.QueryIpBlacklistReq) {
  return request<Users.IpBlacklistListResponseData>({
    url: "ip/blacklist",
    method: "get",
    params
  })
}

/** IP 黑名单：添加并封禁关联用户 */
export function addIpBlacklist(data: Users.AddIpBlacklistReq) {
  return request<Users.AddIpBlacklistResponseData>({
    url: "ip/blacklist",
    method: "post",
    data
  })
}

/** IP 黑名单：移除 */
export function removeIpBlacklist(ip: string) {
  return request({
    url: `ip/blacklist/${ip}`,
    method: "delete"
  })
}

/** IP 黑名单：查看关联用户（分页） */
export function getIpRelatedUsers(params: Users.QueryIpRelatedUsersReq) {
  const { ip, ...rest } = params
  return request<Users.IpRelatedUsersListResponseData>({
    url: `ip/blacklist/${ip}/users`,
    method: "get",
    params: rest
  })
}

/** 批量创建用户 */
export function createUsers(data: Users.CreateUserReq) {
  return request<Users.CreateUserResponseData>({
    url: "users/creates",
    method: "post",
    data
  })
}
