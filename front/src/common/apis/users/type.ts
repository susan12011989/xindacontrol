import type { ApiResponseData } from "@/common/apis/type"

export type CurrentUserResponseData = ApiResponseData<{ username: string, roles: string[], ip: string, two_factor_enabled: boolean }>

export interface DashboardRegion {
  region_name: string // 地区名称
  bandwidth_rate: number // 带宽
  ip_list: string[] // IP(节点)列表
}

export interface DashboardPackage {
  package_name: string // 套餐名称
  regions: DashboardRegion[] // 地区列表
  code: string // 套餐编码,用于用户名的后缀
  account_limit: number // 账号数量限制
  balance: string // 余额($)
}

export type CurrentPackageResponseData = ApiResponseData<DashboardPackage>

// 分页基础类型
export interface Pagination {
  page: number
  size: number
}

// 用户查询请求
export interface QueryUsersReq extends Pagination {
  id?: number // 用户ID（精确）
  phone?: string // 手机号
  username?: string // 用户名
  register_ip?: string // 注册IP
  order?: string // 排序
}

// 用户更新请求
export interface UpdateUserReq {
  first_name?: string // 名
  last_name?: string // 姓
  username?: string // 用户名
  about?: string // 关于
  verified?: number // 是否认证
  premium?: number // 是否高级用户
  premium_expire_date?: number // 高级用户过期时间
}

// 封禁/解禁用户请求
export interface ToggleBanUserReq {
  phone: string // 手机号
  predefined?: boolean // true=封禁，false=解禁
  expires?: number // 过期时间戳（封禁时长，秒）
  reason?: string // 操作原因
  ips?: string[] // 选择要联动加入黑名单的IP列表
}

// 设置用户密码请求
export interface SetUserPasswordReq {
  password: string // 密码
}

// 用户响应
export interface UserResp {
  id: number // 用户ID
  first_name: string // 名
  last_name: string // 姓
  username: string // 用户名
  phone: string // 手机号
  country_code: string // 国家代码
  about: string // 关于
  photo: string // 头像URL
  verified: number // 是否认证
  premium: number // 是否高级用户
  premium_expire_date: number // 高级用户过期时间
  support: number // 是否客服
  is_bot: number // 是否机器人
  restricted: number // 是否受限
  restriction_reason: string // 限制原因
  scam: number // 是否诈骗
  fake: number // 是否虚假
  account_days_ttl: number // 账户天数TTL
  authorization_ttl_days: number // 授权天数TTL
  photo_id: number // 头像ID
  birthday: string // 生日
  deleted: number // 是否删除
  register_ip: string // 注册IP
  register_ip_region: string // 注册IP归属地
  source: string // 注册来源
  // 封禁相关字段
  banned: boolean // 是否被封禁
  banned_time: number // 封禁时间戳
  banned_expires: number // 封禁过期时间
  banned_reason: string // 封禁原因
  created_at: string // 创建时间
  updated_at: string // 更新时间
}

// 用户统计
export interface UserStats {
  total: number // 总用户数
  verified: number // 认证用户数
  premium: number // 高级用户数
  bots: number // 机器人数
  restricted: number // 受限用户数
}

// 用户列表响应
export interface QueryUsersResponse {
  list: UserResp[]
  total: number
}

// API响应类型
export type QueryUsersResponseData = ApiResponseData<QueryUsersResponse>
export type UserDetailResponseData = ApiResponseData<UserResp>
export type UserStatsResponseData = ApiResponseData<UserStats>

// 历史使用IP
export interface QueryUserUsedIpsReq extends Pagination {
  user_id: number
}
export interface UserUsedIpResp {
  ip: string
  created_at: string
  region: string
}
export type UserUsedIpListResponseData = ApiResponseData<{ list: UserUsedIpResp[], total: number }>

// IP 黑名单
export interface QueryIpBlacklistReq {
  page?: number
  size?: number
  keyword?: string
}
export interface IpBlacklistResp {
  ip: string
  region: string
  created_at: string
}
export type IpBlacklistListResponseData = ApiResponseData<{ list: IpBlacklistResp[], total: number }>

export interface AddIpBlacklistReq {
  ip: string
}
export interface AddIpBlacklistResp {
  ip: string
  affected_users: number
  banned_count: number
}
export type AddIpBlacklistResponseData = ApiResponseData<AddIpBlacklistResp>

export interface QueryIpRelatedUsersReq {
  ip: string
  page?: number
  size?: number
}
export interface IpRelatedUserResp extends UserResp {}
export type IpRelatedUsersListResponseData = ApiResponseData<{ list: IpRelatedUserResp[], total: number }>

// 批量创建用户
export interface CreateUserReqItem {
  phone: string // 区号+手机号
  password?: string // 密码，不指定则随机生成
  first_name: string // 名字
  username?: string // 用户名，不指定则随机生成
  about?: string // 个人介绍
  verified?: number // 是否认证
  premium?: number // 是否高级用户
  premium_expire_date?: number | string // 高级用户过期时间（时间戳，秒，可能是字符串需要转换）
}

export interface CreateUserReq {
  items: CreateUserReqItem[]
}

export type CreateUserResponseData = ApiResponseData<string[]> // 返回成功创建的手机号列表
