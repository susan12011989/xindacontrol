import type * as APITest from "./type"
import { request } from "@/http/axios"

// 获取API目录
export function getAPICatalog() {
  return request<APITest.APICatalogRespData>({
    url: "/api-test/catalog",
    method: "get"
  })
}

// 创建测试用例
export function createTestCase(merchantId: number, data: APITest.TestCaseReq) {
  return request({
    url: "/api-test/cases",
    method: "post",
    params: { merchant_id: merchantId },
    data
  })
}

// 更新测试用例
export function updateTestCase(id: number, data: APITest.TestCaseReq) {
  return request({
    url: `/api-test/cases/${id}`,
    method: "put",
    data
  })
}

// 删除测试用例
export function deleteTestCase(id: number) {
  return request({
    url: `/api-test/cases/${id}`,
    method: "delete"
  })
}

// 查询测试用例
export function queryTestCases(params: APITest.QueryTestCaseReq) {
  return request<APITest.QueryTestCaseRespData>({
    url: "/api-test/cases",
    method: "get",
    params
  })
}

// 运行API测试
export function runAPITest(data: APITest.RunAPITestReq) {
  return request<APITest.RunAPITestRespData>({
    url: "/api-test/run",
    method: "post",
    data
  })
}

// 运行单个测试用例
export function runTestCase(caseId: number, merchantId: number) {
  return request<APITest.RunAPITestRespData>({
    url: `/api-test/run/case/${caseId}`,
    method: "post",
    params: { merchant_id: merchantId }
  })
}

// 批量测试
export function batchTest(data: APITest.BatchTestReq) {
  return request<APITest.BatchTestRespData>({
    url: "/api-test/run/batch",
    method: "post",
    data
  })
}

// 创建监控配置
export function createMonitor(data: APITest.MonitorConfigReq) {
  return request({
    url: "/api-test/monitors",
    method: "post",
    data
  })
}

// 更新监控配置
export function updateMonitor(id: number, data: APITest.MonitorConfigReq) {
  return request({
    url: `/api-test/monitors/${id}`,
    method: "put",
    data
  })
}

// 删除监控配置
export function deleteMonitor(id: number) {
  return request({
    url: `/api-test/monitors/${id}`,
    method: "delete"
  })
}

// 查询监控配置
export function queryMonitors(params: APITest.QueryMonitorReq) {
  return request<APITest.QueryMonitorRespData>({
    url: "/api-test/monitors",
    method: "get",
    params
  })
}

// 查询监控历史
export function queryMonitorHistory(monitorId: number, params: APITest.QueryMonitorHistoryReq) {
  return request<APITest.QueryMonitorHistoryRespData>({
    url: `/api-test/monitors/${monitorId}/history`,
    method: "get",
    params
  })
}
