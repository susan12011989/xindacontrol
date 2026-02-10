// API端点
export interface APIEndpoint {
  method: string
  path: string
  description: string
  module: string
  headers?: Record<string, string>
  params?: APIParam[]
  require_auth: boolean
}

// API参数
export interface APIParam {
  name: string
  type: string
  required: boolean
  description: string
  in: string
  default?: string
}

// API分类
export interface APICategory {
  name: string
  module: string
  endpoints: APIEndpoint[]
}

// API目录响应
export interface APICatalogResp {
  categories: APICategory[]
  total: number
}

// 测试用例请求
export interface TestCaseReq {
  id?: number
  name: string
  module: string
  method: string
  path: string
  headers?: Record<string, string>
  query_params?: Record<string, string>
  body?: string
  expected_status: number
  expected_contains?: string
}

// 测试用例响应
export interface TestCaseResp {
  id: number
  name: string
  module: string
  method: string
  path: string
  headers?: Record<string, string>
  query_params?: Record<string, string>
  body?: string
  expected_status: number
  expected_contains?: string
  last_run_at?: string
  last_run_status: number
  created_at: string
  updated_at: string
}

// 查询测试用例请求
export interface QueryTestCaseReq {
  page: number
  size: number
  merchant_id?: number
  module?: string
  name?: string
}

// 查询测试用例响应
export interface QueryTestCaseResp {
  list: TestCaseResp[]
  total: number
}

// 运行API测试请求
export interface RunAPITestReq {
  merchant_id: number
  method: string
  path: string
  headers?: Record<string, string>
  query_params?: Record<string, string>
  body?: string
}

// 运行API测试响应
export interface RunAPITestResp {
  success: boolean
  status_code: number
  response_time: number
  headers?: Record<string, string>
  body?: string
  error?: string
}

// 批量测试请求
export interface BatchTestReq {
  merchant_id: number
  test_case_ids: number[]
}

// 批量测试结果
export interface BatchTestResult {
  test_case_id: number
  test_case_name: string
  success: boolean
  status_code: number
  response_time: number
  error?: string
}

// 批量测试响应
export interface BatchTestResp {
  total: number
  success: number
  failed: number
  results: BatchTestResult[]
  total_time: number
}

// 监控配置请求
export interface MonitorConfigReq {
  id?: number
  merchant_id: number
  name: string
  test_case_ids: number[]
  interval: number
  enabled: boolean
  alert_email?: string
  alert_webhook?: string
}

// 监控配置响应
export interface MonitorConfigResp {
  id: number
  merchant_id: number
  merchant_name: string
  name: string
  test_case_ids: number[]
  interval: number
  enabled: boolean
  alert_email?: string
  alert_webhook?: string
  last_run_at?: string
  last_status: number
  created_at: string
}

// 查询监控请求
export interface QueryMonitorReq {
  page: number
  size: number
  merchant_id?: number
  enabled?: boolean
}

// 查询监控响应
export interface QueryMonitorResp {
  list: MonitorConfigResp[]
  total: number
}

// 监控历史响应
export interface MonitorHistoryResp {
  id: number
  monitor_id: number
  run_at: string
  total: number
  success: number
  failed: number
  total_time: number
  results: string
}

// 查询监控历史请求
export interface QueryMonitorHistoryReq {
  page: number
  size: number
}

// 查询监控历史响应
export interface QueryMonitorHistoryResp {
  list: MonitorHistoryResp[]
  total: number
}

// ========== API Response Data 类型 (包装 ApiResponseData) ==========
import type { ApiResponseData } from "@/common/apis/type"

export type APICatalogRespData = ApiResponseData<APICatalogResp>
export type QueryTestCaseRespData = ApiResponseData<QueryTestCaseResp>
export type RunAPITestRespData = ApiResponseData<RunAPITestResp>
export type BatchTestRespData = ApiResponseData<BatchTestResp>
export type QueryMonitorRespData = ApiResponseData<QueryMonitorResp>
export type QueryMonitorHistoryRespData = ApiResponseData<QueryMonitorHistoryResp>
