/** 通用列表响应 */
export interface ListResp<T> {
  list: T[]
  total: number
}

/** 商户打包配置 */
export interface BuildMerchant {
  id: number
  merchant_id?: number
  name: string
  app_name: string
  short_name: string
  description: string
  status: number

  // Android
  android_package: string
  android_version_code: number
  android_version_name: string

  // iOS
  ios_bundle_id: string
  ios_version: string
  ios_build: string

  // Windows
  windows_app_name: string
  windows_version: string

  // macOS
  macos_bundle_id: string
  macos_app_name: string
  macos_version: string

  // 服务器配置
  server_api_url: string
  server_ws_host: string
  server_ws_port: number
  enterprise_code: string

  // 资源
  icon_url: string
  logo_url: string
  splash_url: string

  // 推送配置
  push_mi_app_id: string
  push_mi_app_key: string
  push_oppo_app_key: string
  push_oppo_app_secret: string
  push_vivo_app_id: string
  push_vivo_app_key: string
  push_hms_app_id: string

  // Apple 开发者配置
  apple_team_id: string
  apple_certificate_url: string
  apple_provisioning_url: string
  apple_mac_provisioning_url: string
  apple_export_method: string

  // Git 源码配置
  git_repo_url: string
  git_branch: string
  git_tag: string
  git_username: string

  created_at: string
  updated_at: string
}

/** 商户配置列表请求 */
export interface MerchantListReq {
  page?: number
  size?: number
  name?: string
  status?: string
}

/** 创建/编辑商户配置请求 */
export interface BuildMerchantReq {
  merchant_id?: number
  name: string
  app_name: string
  short_name: string
  description?: string

  android_package: string
  android_version_code?: number
  android_version_name?: string
  ios_bundle_id: string
  ios_version?: string
  ios_build?: string
  windows_app_name?: string
  windows_version?: string
  macos_bundle_id?: string
  macos_app_name?: string
  macos_version?: string

  server_api_url?: string
  server_ws_host?: string
  server_ws_port?: number
  enterprise_code?: string

  push_mi_app_id?: string
  push_mi_app_key?: string
  push_oppo_app_key?: string
  push_oppo_app_secret?: string
  push_vivo_app_id?: string
  push_vivo_app_key?: string
  push_hms_app_id?: string

  // Apple 开发者配置
  apple_team_id?: string
  apple_certificate_url?: string
  apple_certificate_password?: string
  apple_provisioning_url?: string
  apple_mac_provisioning_url?: string
  apple_export_method?: string

  // Git 源码配置
  git_repo_url?: string
  git_branch?: string
  git_tag?: string
  git_username?: string
  git_token?: string
}

/** 构建任务 */
export interface BuildTask {
  id: number
  build_merchant_id: number
  merchant_name: string
  platforms: string
  status: number // 0-排队 1-构建中 2-成功 3-失败 4-取消
  progress: number
  current_step: string
  operator: string
  started_at?: string
  finished_at?: string
  duration: number
  error_msg: string
  log_url: string
  created_at: string

  override_android_version_code?: number
  override_android_version_name?: string
  override_ios_version?: string
  override_ios_build?: string
}

/** 任务列表请求 */
export interface TaskListReq {
  page?: number
  size?: number
  status?: string
  build_merchant_id?: string
}

/** 创建任务请求 */
export interface CreateTaskReq {
  build_merchant_id: number
  platforms: string
  override_android_version_code?: number
  override_android_version_name?: string
  override_ios_version?: string
  override_ios_build?: string
}

/** 构建产物 */
export interface BuildArtifact {
  id: number
  task_id: number
  build_merchant_id: number
  merchant_name: string
  platform: string
  file_name: string
  file_size: number
  file_url: string
  version: string
  expires_at: string
  download_count: number
  is_deleted: number
  created_at: string
}

/** 产物列表请求 */
export interface ArtifactListReq {
  page?: number
  size?: number
  build_merchant_id?: string
  platform?: string
}

/** 构建统计 */
export interface BuildStats {
  today: {
    total: number
    success: number
    failed: number
    rate: number
    building: number
    avg_second: number
  }
  week: {
    total: number
    success: number
    failed: number
  }
  platforms: {
    android: number
    ios: number
    windows: number
    macos: number
  }
}

/** 构建服务器 */
export interface BuildServer {
  id: number
  name: string
  host: string
  port: number
  username: string
  auth_type: number
  work_dir: string
  platforms: string
  max_concurrent: number
  current_tasks: number
  status: number // 0-离线 1-在线 2-忙碌
  last_heartbeat?: string
  description: string
  created_at: string
  updated_at: string
}

/** 构建服务器请求 */
export interface BuildServerReq {
  name: string
  host: string
  port?: number
  username: string
  auth_type: number
  password?: string
  private_key?: string
  work_dir?: string
  platforms: string
  max_concurrent?: number
  description?: string
}

/** 任务状态枚举 */
export const TaskStatus = {
  QUEUED: 0,
  BUILDING: 1,
  SUCCESS: 2,
  FAILED: 3,
  CANCELLED: 4,
} as const

/** 任务状态标签 */
export const TaskStatusLabel: Record<number, string> = {
  0: "排队中",
  1: "构建中",
  2: "成功",
  3: "失败",
  4: "已取消",
}

/** 任务状态颜色 */
export const TaskStatusColor: Record<number, "success" | "warning" | "info" | "primary" | "danger"> = {
  0: "info",
  1: "warning",
  2: "success",
  3: "danger",
  4: "info",
}

/** 平台图标 */
export const PlatformIcon: Record<string, string> = {
  android: "🤖",
  ios: "🍎",
  windows: "🪟",
  macos: "💻",
}
