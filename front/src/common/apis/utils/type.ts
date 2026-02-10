import type { ApiResponseData } from "@@/apis/type"

// ========== 端口转换 ==========

/** 端口转企业号请求 */
export interface Port2EnterpriseReq {
  port: number
}

/** 端口转企业号响应 */
export interface Port2EnterpriseResp {
  enterprise: string
}

/** 企业号转端口请求 */
export interface Enterprise2PortReq {
  enterprise: string
}

/** 企业号转端口响应 */
export interface Enterprise2PortResp {
  port: number
}

// ========== IP工具 ==========

/** IP嵌入请求 */
export interface EmbedIPsReq {
  ips: string[]
}

/** IP提取响应 */
export interface ExtractIPsResp {
  ips: string[]
}

// ========== URL工具 ==========

/** URL提取响应 */
export interface ExtractURLsResp {
  urls: string[]
}

// ========== 版本管理 ==========

/** 版本条目 */
export interface VersionEntry {
  channel: string
  version: string
}

/** 生成版本配置请求 */
export interface GenerateVersionReq {
  package: string
  versions: VersionEntry[]
}

// API 响应类型
export type Port2EnterpriseResponseData = ApiResponseData<Port2EnterpriseResp>
export type Enterprise2PortResponseData = ApiResponseData<Enterprise2PortResp>
export type ExtractIPsResponseData = ApiResponseData<ExtractIPsResp>

// ========== 端口校验 ==========
export interface CheckPortReq {
  port: number
}
export interface CheckPortResp {
  ok: boolean
}
export type CheckPortResponseData = ApiResponseData<CheckPortResp>

// URL
export type ExtractURLsResponseData = ApiResponseData<ExtractURLsResp>
