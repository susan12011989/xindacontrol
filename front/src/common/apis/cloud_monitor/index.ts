import { request } from "@/http/axios"

// ========== 统一云监控 API ==========

export interface CloudMonitorMetricsReq {
  server_id: number
  period?: "1h" | "6h" | "24h" | "7d"
}

export interface MetricDataPoint {
  timestamp: number
  value: number
}

export interface MetricSeries {
  metric_name: string
  label: string
  unit: string
  data_points: MetricDataPoint[]
}

export interface CloudMonitorMetricsResp {
  cloud_type: string
  instance_id: string
  region_id: string
  volume_ids?: string[]
  period: number
  start_time: number
  end_time: number
  metrics: MetricSeries[]
}

export function getCloudMonitorMetrics(params: CloudMonitorMetricsReq) {
  return request<{ code: number; data: CloudMonitorMetricsResp; message: string }>({
    url: "cloud/monitor/metrics",
    method: "get",
    params,
    timeout: 30000,
  })
}
