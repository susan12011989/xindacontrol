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
