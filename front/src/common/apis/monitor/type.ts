/** GOST 健康检查结果 */
export interface GostCheckResult {
  server_id: number
  server_name: string
  server_host: string
  status: "up" | "down" | "degraded" | "unknown"
  api_reachable: number
  expected_ports: number
  actual_ports: number
  missing_ports?: number[]
  total_conns: number
  current_conns: number
  input_bytes: number
  output_bytes: number
  total_errors: number
  error_rate: number
  error_message?: string
  check_duration: number
}

/** GOST 监控日志 */
export interface GostMonitorLog {
  id: number
  server_id: number
  server_name: string
  server_host: string
  status: string
  api_reachable: number
  expected_ports: number
  actual_ports: number
  missing_ports: string
  total_conns: number
  current_conns: number
  input_bytes: number
  output_bytes: number
  total_errors: number
  error_rate: number
  error_message: string
  check_duration: number
  created_at: string
}

/** 查询监控日志请求 */
export interface QueryMonitorLogsReq {
  page: number
  size: number
  server_id?: number
  status?: string
}

/** 带宽测速结果 */
export interface BandwidthTestResult {
  server_id: number
  server_name: string
  server_host: string
  internal_speeds: { target: string; speed_mbs: number }[]
  gost_upload_speeds: { target: string; speed_mbs: number }[]
  latencies: { target: string; avg_ms: number }[]
  public_upload_kbps: number
  public_download_kbps: number
}

/** 服务器带宽信息 */
export interface BandwidthInfoResp {
  server_id: number
  server_name: string
  internet_max_bandwidth_in: number
  internet_max_bandwidth_out: number
  internet_charge_type: string
  has_eip: boolean
  eip_allocation_id?: string
  eip_bandwidth?: number
  eip_charge_type?: string
}
