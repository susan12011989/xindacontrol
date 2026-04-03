import type { ApiResponseData } from "@/common/apis/type"

// ========== 服务器管理 ==========

// 分页
export interface Pagination {
  page: number
  size: number
}

// 查询服务器请求
export interface QueryServersReq extends Pagination {
  name?: string
  host?: string
  status?: number
  server_type?: number // 1-商户服务器 2-系统服务器
  merchant_id?: number // 按商户ID筛选
  group_id?: number    // 按分组ID筛选
}

// 创建服务器请求
export interface CreateServerReq {
  name: string
  host: string
  auxiliary_ip?: string // 辅助IP，仅系统服务器使用
  port: number
  username: string
  auth_type: number // 1-密码 2-密钥
  password?: string
  private_key?: string
  server_type?: number // 1-商户服务器 2-系统服务器
  forward_type?: number // 转发类型：1-加密(relay+tls) 2-直连(tcp)，仅系统服务器有效
  group_id?: number     // 分组ID
  description?: string
}

// 更新服务器请求
export interface UpdateServerReq {
  name?: string
  host?: string
  auxiliary_ip?: string // 辅助IP，仅系统服务器使用
  port?: number
  username?: string
  auth_type?: number
  password?: string
  private_key?: string
  server_type?: number // 1-商户服务器 2-系统服务器
  forward_type?: number // 转发类型：1-加密(relay+tls) 2-直连(tcp)，仅系统服务器有效
  group_id?: number     // 分组ID
  status?: number
  description?: string
}

// 服务器响应
export interface ServerResp {
  id: number
  name: string
  host: string
  auxiliary_ip: string // 辅助IP，仅系统服务器使用
  port: number
  username: string
  auth_type: number
  server_type: number // 1-商户服务器 2-系统服务器
  forward_type: number // 转发类型：1-加密(relay+tls) 2-直连(tcp)
  status: number
  tls_enabled: number // 客户端TLS：0-未启用 1-已启用
  tls_deployed_at: string // TLS证书部署时间
  description: string
  merchant_id: number // 关联的商户ID
  merchant_name: string // 关联的商户名称
  merchant_no: string // 商户号
  group_id: number // 分组ID
  group_name: string // 分组名称
  created_at: string
  updated_at: string
}

// 服务器列表响应
export interface QueryServersResponse {
  list: ServerResp[]
  total: number
}

// 测试连接请求
export interface TestConnectionReq {
  host: string
  port: number
  username: string
  auth_type: number
  password?: string
  private_key?: string
}

// ========== 服务操作（systemctl） ==========

// 支持的服务：server, wukongim, gost
export type ServiceName = "server" | "wukongim" | "gost"

// 服务操作请求
export interface ServiceActionReq {
  server_id: number
  service_name: ServiceName
  action: "start" | "stop" | "restart"
}

// 服务操作响应
export interface ServiceActionResp {
  success: boolean
  message: string
  output: string
  error_msg: string
}

// 服务状态请求
export interface ServiceStatusReq {
  server_id: number
  service_name?: ServiceName
}

// 服务状态响应
export interface ServiceStatusResp {
  service_name: string
  status: string // running/stopped/unknown
  pid: number
  uptime: string
  cpu: string
  memory: string
}

// 服务状态列表响应
export interface ServiceStatusListResp {
  services: ServiceStatusResp[]
}

// 服务日志请求
export interface ServiceLogsReq {
  server_id: number
  service_name: ServiceName
  lines?: number
}

// 服务日志响应
export interface ServiceLogsResp {
  logs: string
  total_lines: number
  service_name: string
}

// ========== 服务器资源 ==========

// 服务器资源响应
export interface ServerStatsResp {
  cpu_usage: string
  memory_usage: string
  memory_total: string
  disk_usage: string
  disk_total: string
  load_avg: string
}

// 批量获取服务器资源请求
export interface GetServerStatsBatchReq {
  server_ids: number[]
}

// 批量服务器基础资源
export interface ServerBasicStat {
  server_id: number
  cpu_usage: string
  memory_usage: string
  memory_total: string
  error?: string
}

// 批量获取服务器资源响应
export interface GetServerStatsBatchResp {
  stats: ServerBasicStat[]
}

// ========== 文件上传 ==========

export interface UploadToServerResp {
  message: string
  remote_path: string
  service_name: string
}

// ========== 配置文件 ==========

export interface GetConfigFileReq {
  server_id: number
  service_name: ServiceName
}

export interface ConfigFileResp {
  service_name: string
  config_path: string
  content: string
}

export interface UpdateConfigFileReq {
  server_id: number
  service_name: ServiceName
  content: string
}

// ========== API 响应类型 ==========

export type QueryServersResponseData = ApiResponseData<QueryServersResponse>
export type ServerDetailResponseData = ApiResponseData<ServerResp>
export type ServiceActionResponseData = ApiResponseData<ServiceActionResp>
export type ServiceStatusResponseData = ApiResponseData<ServiceStatusListResp>
export type ServiceLogsResponseData = ApiResponseData<ServiceLogsResp>
export type ServerStatsResponseData = ApiResponseData<ServerStatsResp>
export type GetServerStatsBatchResponseData = ApiResponseData<GetServerStatsBatchResp>
export type UploadToServerResponseData = ApiResponseData<UploadToServerResp>
export type ConfigFileResponseData = ApiResponseData<ConfigFileResp>

// ========== GOST API 代理 ==========

// GOST 服务列表请求
export interface ListGostServicesReq {
  server_id: number
  page?: number
  size?: number
  port?: number
}

// GOST 服务配置
export interface GostServiceConfig {
  name: string
  addr?: string
  handler?: Record<string, unknown>
  listener?: Record<string, unknown>
  forwarder?: Record<string, unknown>
  [key: string]: unknown
}

// GOST 服务列表响应
export interface GostServiceListResp {
  count: number
  list: GostServiceConfig[]
}

// GOST Chain 配置
export interface GostChainConfig {
  name: string
  hops?: Array<{
    name?: string
    nodes?: Array<{
      name?: string
      addr?: string
      [key: string]: unknown
    }>
    [key: string]: unknown
  }>
  [key: string]: unknown
}

// GOST Chain 列表响应
export interface GostChainListResp {
  count: number
  list: GostChainConfig[]
}

// 创建 GOST 服务请求
export interface CreateGostServiceReq {
  server_id: number
  listen_port: number
  forward_host: string
  forward_port: number
}

// 删除 GOST 服务请求
export interface DeleteGostServiceReq {
  server_id: number
  service_name: string
}

// GOST API 响应类型
export type GostServiceListResponseData = ApiResponseData<GostServiceListResp>
export type GostServiceDetailResponseData = ApiResponseData<GostServiceConfig>
export type GostChainListResponseData = ApiResponseData<GostChainListResp>

// ========== 批量分发 ==========

// 批量分发请求（从本地服务器分发到目标服务器）
export interface DistributeFileReq {
  service_name: ServiceName // 服务名：server 或 wukongim
  target_server_ids: number[] // 目标服务器ID列表（商户服务器）
  restart_after?: boolean // 分发后是否重启服务
}

// 上传到本地响应
export interface UploadToLocalResp {
  message: string
  local_path: string
  service_name: string
}

export type UploadToLocalResponseData = ApiResponseData<UploadToLocalResp>

// ========== Docker 容器状态 ==========

// Docker 容器状态
export interface DockerContainerStatus {
  container_id: string // 容器ID（短）
  name: string // 容器名称
  image: string // 镜像名
  status: string // 状态（Up/Exited等）
  ports: string // 端口映射
  created: string // 创建时间
  running_for: string // 运行时长
  cpu_percent: string // CPU 使用率
  mem_usage: string // 内存使用
  mem_percent: string // 内存使用率
}

// Docker 容器状态响应
export interface DockerContainersResp {
  containers: DockerContainerStatus[]
}

export type DockerContainersResponseData = ApiResponseData<DockerContainersResp>

// 单个服务器分发结果
export interface DistributeResult {
  server_id: number
  server_name: string
  success: boolean
  message: string
}

// 批量分发响应
export interface DistributeFileResp {
  total_count: number
  success_count: number
  fail_count: number
  results: DistributeResult[]
}

export type DistributeFileResponseData = ApiResponseData<DistributeFileResp>

// ========== GOST 一键部署 ==========

// 部署 GOST 服务器请求
export interface DeployGostServerReq {
  cloud_account_id: number // 云账号ID
  region_id: string // 地区ID
  instance_type?: string // 实例类型，为空使用默认
  image_id?: string // 镜像ID，为空使用默认Ubuntu
  server_name?: string // 服务器名称
  group_id?: number // 服务器分组ID
  password?: string // SSH 密码（可选，不填则自动生成密钥）
  bandwidth?: string // EIP 带宽，默认 5Mbps
}

// GOST 一键部署（安装+配置转发）
export interface SetupGostDeployReq {
  server_id: number // 系统服务器ID
  merchant_ids: number[] // 商户ID列表
  forward_type?: number // 转发类型: 1-加密(默认) 2-直连
}

// 在已有服务器上安装 GOST 请求
export interface InstallGostReq {
  server_id?: number // 服务器ID（二选一）
  host?: string // 服务器IP（二选一）
  port?: number // SSH端口，默认22
  username?: string // SSH用户名，默认root
  password?: string // SSH密码
  private_key?: string // SSH私钥（二选一）
}

// GOST 部署默认配置
export interface GostDeployConfig {
  default_instance_type: string
  default_image_id: string
  default_bandwidth: string
  available_regions: Array<{ id: string; name: string }>
}

// ========== 批量运维操作 ==========

// 批量服务操作请求
export interface BatchServiceActionReq {
  server_ids: number[]
  service_name: ServiceName
  action: "start" | "stop" | "restart"
  parallel?: boolean // 是否并行执行
}

// 批量服务操作结果
export interface BatchServiceResult {
  server_id: number
  server_name: string
  server_host: string
  success: boolean
  message: string
  output?: string
}

// 批量服务操作响应
export interface BatchServiceActionResp {
  total_count: number
  success_count: number
  fail_count: number
  results: BatchServiceResult[]
}

// 批量健康检查请求
export interface BatchHealthCheckReq {
  server_ids: number[]
}

// 服务器健康检查结果
export interface ServerHealthResult {
  server_id: number
  server_name: string
  server_host: string
  status: "healthy" | "unhealthy" | "partial" | "error"
  message?: string
  check_time: string
}

// 批量健康检查响应
export interface BatchHealthCheckResp {
  total_count: number
  healthy_count: number
  unhealthy_count: number
  partial_count: number
  results: ServerHealthResult[]
}

// 批量命令执行请求
export interface BatchCommandReq {
  server_ids: number[]
  command: string
  timeout?: number // 超时时间（秒）
  parallel?: boolean
}

// 批量命令执行结果
export interface BatchCommandResult {
  server_id: number
  server_name: string
  server_host: string
  success: boolean
  output: string
  error?: string
  duration_ms: number
}

// 批量命令执行响应
export interface BatchCommandResp {
  total_count: number
  success_count: number
  fail_count: number
  results: BatchCommandResult[]
}

// ========== 日志查询 ==========

// 日志查询请求
export interface LogQueryReq {
  server_id: number
  query_type?: "journalctl" | "docker" | "file" // 默认 journalctl
  service_name?: string // server/wukongim/gost
  container_name?: string // Docker 容器名
  log_path?: string // 文件日志路径
  lines?: number // 行数，默认 100
  since?: string // 起始时间，如 "1h", "30m", "2024-01-01 00:00:00"
  until?: string // 结束时间
  keyword?: string // 关键字过滤
  level?: string // 日志级别过滤 error/warn/info
}

// 日志查询响应
export interface LogQueryResp {
  logs: string
  line_count: number
  truncated: boolean
  command?: string
}

// 批量运维响应类型
export type BatchServiceActionResponseData = ApiResponseData<BatchServiceActionResp>
export type BatchHealthCheckResponseData = ApiResponseData<BatchHealthCheckResp>
export type BatchCommandResponseData = ApiResponseData<BatchCommandResp>
export type LogQueryResponseData = ApiResponseData<LogQueryResp>

// ========== TSDD AMI 部署 ==========

// 使用 AMI 部署 TSDD 请求
export interface DeployTSDDWithAMIReq {
  merchant_id: number
  cloud_account_id: number
  region_id: string
  ami_id?: string // 可选，不填则使用默认 TSDD AMI
  instance_type?: string // 可选，默认 t3.medium
  subnet_id?: string // 可选
  key_name?: string // 可选
  volume_size_gib?: number // 可选，默认 30
  server_name?: string // 服务器名称
  source_server_id?: number // 可选，从某服务器克隆（会先创建 AMI）
  // EBS 额外数据卷
  enable_extra_ebs?: boolean // 是否创建额外 EBS 数据卷
  db_volume_size_gib?: number // DB 数据卷大小(GB)，默认 20
  db_volume_iops?: number // DB 数据卷 IOPS，默认 3000
  minio_volume_size_gib?: number // MinIO 数据卷大小(GB)，默认 50
  minio_volume_iops?: number // MinIO 数据卷 IOPS，默认 3000
}

// 使用 AMI 部署响应
export interface DeployTSDDWithAMIResp {
  success: boolean
  message: string
  instance_id: string
  public_ip: string
  private_ip: string
  server_id: number // 注册到系统的服务器ID
  api_url: string
  web_url: string
  admin_url: string
  // EBS 卷信息
  db_volume_id?: string
  minio_volume_id?: string
}

export type DeployTSDDWithAMIResponseData = ApiResponseData<DeployTSDDWithAMIResp>

// ========== TSDD Docker 部署 ==========

// 部署 TSDD 到已注册服务器
export interface DeployTSDDReq {
  server_id: number // 目标服务器ID
  merchant_id: number // 商户ID（用于获取配置）
  use_images?: boolean // true=使用官方镜像, false=使用自定义二进制
  force_reset?: boolean // 强制重置（删除现有容器和数据）
}

// 通过IP直接部署 TSDD（新服务器，未注册到系统）
export interface DeployTSDDByIPReq {
  host: string // 服务器IP
  port?: number // SSH端口，默认22
  username?: string // SSH用户名，默认root
  password: string // SSH密码
  merchant_id: number // 商户ID
  server_name?: string // 服务器名称（用于注册）
  use_images?: boolean // true=使用官方镜像
  force_reset?: boolean // 强制重置
}

// 部署步骤
export interface DeployStep {
  name: string // 步骤名称
  status: string // pending/running/success/failed
  message: string // 详细信息
  output: string // 命令输出
}

// 部署 TSDD 响应
export interface DeployTSDDResp {
  success: boolean
  message: string
  steps: DeployStep[] // 各步骤执行结果
  server_id: number // 服务器ID（新注册时返回）
  api_url?: string // 部署后的API地址
  web_url?: string // 部署后的Web地址
  admin_url?: string // 管理后台地址
}

// 获取部署状态请求
export interface GetDeployStatusReq {
  server_id: number
}

// 部署状态响应
export interface GetDeployStatusResp {
  deployed: boolean // 是否已部署
  services: string[] // 已部署的服务列表
  healthy: boolean // 服务是否健康
  last_deploy: string // 最后部署时间
  version: string // 部署版本
}

export type DeployTSDDResponseData = ApiResponseData<DeployTSDDResp>
export type GetDeployStatusResponseData = ApiResponseData<GetDeployStatusResp>

// ========== 集群节点部署 ==========

export interface DeployNodeReq {
  server_id: number
  merchant_id: number
  node_role: "allinone" | "db" | "app" | "minio"
  force_reset?: boolean
  db_host?: string       // app 节点必填，DB 节点内网 IP
  minio_host?: string    // app 节点可选，MinIO 节点内网 IP
  wk_node_id?: number    // WuKongIM 集群节点 ID
  wk_seed_node?: string  // 种子节点
  // EC2 创建模式（server_id=0 时使用）
  ami_id?: string
  instance_type?: string
  volume_size_gib?: number
  cloud_account_id?: number
  region_id?: string
  key_name?: string
  subnet_id?: string
}

export type DeployNodeResponseData = ApiResponseData<DeployTSDDResp>

export interface ClusterNodeInfo {
  node_id: number
  merchant_id: number
  role: string
  host: string
  private_ip: string
  server_id: number
  server_name: string
  is_primary: number
  status: number
  wk_node_id: number
  db_host: string
  minio_host: string
  deploy_status: string
  deploy_error: string
  last_deploy_at: string
}

export interface ClusterTopologyResp {
  merchant_id: number
  merchant_name: string
  deploy_mode: string
  nodes: ClusterNodeInfo[]
}

export type ClusterTopologyResponseData = ApiResponseData<ClusterTopologyResp>

export interface RetryDeployReq {
  node_id: number
}

// ========== GOST 转发配置（一键部署） ==========

// 配置 GOST 转发请求
export interface SetupGostForwardReq {
  server_id: number // GOST 服务器ID
  target_ip: string // 转发目标IP
  ports?: number[] // 转发端口列表（可选，为空使用默认）
  mode?: "tls" | "tcp" // 连接模式：tls(加密，默认) 或 tcp(直连)
}

// 清除 GOST 转发请求
export interface ClearGostForwardReq {
  server_id: number // GOST 服务器ID
  ports?: number[] // 要清除的端口列表（可选，为空清除所有）
}

// 单个转发项
export interface GostForwardItem {
  port: number // 监听端口
  target_ip: string // 目标IP
  status: string // 状态：active/inactive
}

// GOST 转发状态响应
export interface GostForwardStatusResp {
  server_id: number
  server_name: string
  server_ip: string
  forwards: GostForwardItem[]
  total_count: number
}

export type GostForwardStatusResponseData = ApiResponseData<GostForwardStatusResp>

// ========== GOST 配置持久化 ==========

// GOST 配置同步状态响应
export interface GostConfigSyncStatusResp {
  synced: boolean
  running_service_count: number
  running_chain_count: number
  file_service_count: number
  file_chain_count: number
  message: string
}

export type GostConfigSyncStatusResponseData = ApiResponseData<GostConfigSyncStatusResp>

// ========== Nginx 缓存管理 ==========

// 安装 Nginx 请求
export interface InstallNginxReq {
  server_id: number
}

// 清除 Nginx 缓存请求
export interface ClearNginxCacheReq {
  server_id: number
}

// Nginx 缓存状态响应
export interface NginxCacheStatusResp {
  installed: boolean
  running: boolean
  cache_size: string
}

export type NginxCacheStatusResponseData = ApiResponseData<NginxCacheStatusResp>

// ========== TLS 证书管理 ==========

// 证书详情
export interface TlsCertificateResp {
  id: number
  name: string
  cert_type: number // 1-CA根证书 2-服务器证书
  fingerprint: string // SHA-256 指纹
  expires_at: string
  status: number
  created_at: string
  updated_at: string
}

// 生成证书请求
export interface GenerateTlsCertReq {
  validity_days?: number // 有效期天数，默认 3650(10年)
}

// 生成证书响应
export interface GenerateTlsCertResp {
  ca: TlsCertificateResp
  server: TlsCertificateResp
}

// 批量升级/回滚 TLS 请求
export interface BatchTlsReq {
  server_ids?: number[] // 为空则操作所有系统服务器
}

// 单台服务器 TLS 操作结果
export interface TlsServerResult {
  server_id: number
  server_name: string
  host: string
  success: boolean
  error?: string
}

// 批量 TLS 操作响应
export interface BatchTlsResp {
  total: number
  success: number
  failed: number
  results: TlsServerResult[]
}

// 单台服务器 TLS 状态
export interface TlsServerStatus {
  server_id: number
  server_name: string
  host: string
  tls_enabled: number // 0-未启用 1-已启用
  tls_deployed_at: string
  tls_verified: boolean
  verify_error?: string
}

// TLS 状态查询响应
export interface TlsStatusResp {
  total: number
  tls_count: number
  tcp_count: number
  servers: TlsServerStatus[]
}

// 证书指纹响应
export interface CertFingerprintResp {
  fingerprint: string
  expires_at: string
}

// TLS API 响应类型
export type TlsCertsResponseData = ApiResponseData<TlsCertificateResp[]>
export type GenerateTlsCertResponseData = ApiResponseData<GenerateTlsCertResp>
export type BatchTlsResponseData = ApiResponseData<BatchTlsResp>
export type TlsStatusResponseData = ApiResponseData<TlsStatusResp>
export type CertFingerprintResponseData = ApiResponseData<CertFingerprintResp>

// ========== 流量监控与应急响应 ==========

// 网卡流量数据
export interface InterfaceTraffic {
  name: string
  rx_rate: number  // 入站 KB/s
  tx_rate: number  // 出站 KB/s
  rx_total: string
  tx_total: string
}

// IP 连接统计
export interface IPConnection {
  ip: string
  count: number
}

// 流量监控响应
export interface TrafficStatsResp {
  interfaces: InterfaceTraffic[]
  total_rx_rate: number
  total_tx_rate: number
  total_rx: string
  total_tx: string
  connections: number
  syn_count: number
  estab_count: number
  top_ips: IPConnection[]
  alert_level: "normal" | "warning" | "danger"
  alert_msg: string
}

// 批量流量监控项
export interface ServerTrafficStat {
  server_id: number
  total_rx_rate: number
  total_tx_rate: number
  connections: number
  syn_count: number
  alert_level: "normal" | "warning" | "danger"
  alert_msg: string
  error?: string
}

// 封禁 IP 请求
export interface BlockIPReq {
  server_id: number
  ip: string
  duration?: string  // "1h", "24h", "permanent"
}

// 紧急限流请求
export interface EmergencyRateLimitReq {
  server_id: number
  max_conn_per_ip: number  // 0=取消限制
  max_syn_rate: number     // 0=取消限制
}

// 流量监控 API 响应类型
export type TrafficStatsResponseData = ApiResponseData<TrafficStatsResp>
export type TrafficStatsBatchResponseData = ApiResponseData<{ stats: ServerTrafficStat[] }>
export type BlockedIPsResponseData = ApiResponseData<{ ips: string[] }>
