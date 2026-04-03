// IP嵌入上传类型定义

/** 服务器关联的商户信息 */
export interface SystemIPMerchantItem {
  merchant_id: number
  merchant_name: string
}

/** 系统IP项 */
export interface SystemIPItem {
  server_id: number
  server_name: string
  ip: string
  auxiliary_ip: string // 辅助IP
  status: number
  merchant_id: number   // 主商户ID（兼容旧逻辑）
  merchant_name: string // 主商户名称（兼容旧逻辑）
  merchants: SystemIPMerchantItem[] // 关联的所有商户
}

/** 获取系统IP响应 */
export interface GetSystemIPsResp {
  ips: SystemIPItem[]
  total: number
}

/** 上传目标项 */
export interface TargetItem {
  id: number
  index: number
  name: string
  cloud_type: "aws" | "aliyun" | "tencent"
  cloud_account_id: number
  account_name: string
  region_id: string
  bucket: string
  object_prefix: string
  enabled: boolean
  sort_order: number
  group_id: number
  group_name: string
  merchant_id: number
  merchant_name: string
}

/** 获取目标响应 */
export interface GetTargetsResp {
  targets: TargetItem[]
}

/** 源文件项 */
export interface SourceFileItem {
  name: string
  size: number
  mod_time: string
}

/** 获取源文件响应 */
export interface GetSourceFilesResp {
  files: SourceFileItem[]
  total: number
  source_dir: string
}

/** 执行嵌入请求 */
export interface ExecuteEmbedReq {
  target_indexes: number[]
  file_names?: string[]
  ips: string[] // 选择的IP列表
}

/** 上传结果项 */
export interface UploadResultItem {
  file_name: string
  target_name: string
  cloud_type: string
  bucket: string
  object_key: string
  success: boolean
  error?: string
  object_url?: string
}

/** 执行摘要 */
export interface ExecutionSummary {
  success_count: number
  fail_count: number
  total_count: number
  duration: string
}

/** 执行结果响应 */
export interface ExecuteEmbedResp {
  execution_id: string
  total_files: number
  total_targets: number
  results: UploadResultItem[]
  summary: ExecutionSummary
}

/** 保存选中IP请求 */
export interface SaveSelectedIPsReq {
  ips: string[]
}

/** 获取选中IP响应 */
export interface GetSelectedIPsResp {
  ips: string[]
}

/** 创建目标请求 */
export interface CreateTargetReq {
  name: string
  cloud_type: string
  cloud_account_id: number
  region_id: string
  bucket: string
  object_prefix?: string
  enabled?: boolean
  sort_order?: number
  group_id?: number
}

/** 更新目标请求 */
export interface UpdateTargetReq {
  name?: string
  cloud_type?: string
  cloud_account_id?: number
  region_id?: string
  bucket?: string
  object_prefix?: string
  enabled?: boolean
  sort_order?: number
  group_id?: number
}

/** 资源分组项 */
export interface ResourceGroupItem {
  id: number
  name: string
  resource_type: string
  sort_order: number
  count: number
  created_at: string
}

/** 资源分组请求 */
export interface ResourceGroupReq {
  name: string
  sort_order?: number
}
