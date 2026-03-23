export interface ModeResp {
  mode: "local" | "cluster"
}

export interface ServiceStatus {
  service_name: string
  status: string // running / stopped / unknown
  pid: number
  uptime: string
  cpu: string
  memory: string
}

export interface HealthCheckResp {
  [serviceName: string]: ServiceStatus
}

export interface ServiceActionReq {
  server_id?: number
  service_name: string
  action: "start" | "stop" | "restart"
}

export interface ServiceStatusResp {
  services: ServiceStatus[]
}

export interface ServiceLogsResp {
  logs: string
  total_lines: number
  service_name: string
}

export interface ServerStatsResp {
  cpu_usage: string
  memory_usage: string
  memory_total: string
  disk_usage: string
  disk_total: string
  load_avg: string
}

export interface Endpoint {
  host: string
  port: number
}

export interface EndpointsResp {
  endpoints: Endpoint[]
}
