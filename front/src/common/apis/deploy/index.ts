import type { AxiosProgressEvent } from "axios"
import type * as Deploy from "./type"
import { createStreamRequest, request } from "@/http/axios"

// ========== 服务器管理 ==========

/** 获取服务器列表 */
export function getServerList(params: Deploy.QueryServersReq) {
  return request<Deploy.QueryServersResponseData>({
    url: "deploy/servers",
    method: "get",
    params
  })
}

/** 获取服务器详情 */
export function getServerDetail(id: number) {
  return request<Deploy.ServerDetailResponseData>({
    url: `deploy/servers/${id}`,
    method: "get"
  })
}

/** 创建服务器 */
export function createServer(data: Deploy.CreateServerReq) {
  return request({
    url: "deploy/servers",
    method: "post",
    data
  })
}

/** 更新服务器 */
export function updateServer(id: number, data: Deploy.UpdateServerReq) {
  return request({
    url: `deploy/servers/${id}`,
    method: "put",
    data
  })
}

/** 删除服务器 */
export function deleteServer(id: number) {
  return request({
    url: `deploy/servers/${id}`,
    method: "delete"
  })
}

/** 切换服务器启用/禁用状态 */
export function toggleServerStatus(id: number) {
  return request({
    url: `deploy/servers/${id}/toggle-status`,
    method: "post"
  })
}

/** 测试SSH连接 */
export function testConnection(data: Deploy.TestConnectionReq) {
  return request({
    url: "deploy/servers/test",
    method: "post",
    data
  })
}

// ========== 服务操作（systemctl） ==========

/** 服务操作（start/stop/restart） */
export function serviceAction(data: Deploy.ServiceActionReq) {
  return request<Deploy.ServiceActionResponseData>({
    url: "deploy/service/action",
    method: "post",
    data
  })
}

/** 获取服务状态 */
export function getServiceStatus(params: Deploy.ServiceStatusReq) {
  return request<Deploy.ServiceStatusResponseData>({
    url: "deploy/service/status",
    method: "get",
    params
  })
}

/** 获取服务日志 */
export function getServiceLogs(params: Deploy.ServiceLogsReq) {
  return request<Deploy.ServiceLogsResponseData>({
    url: "deploy/service/logs",
    method: "get",
    params
  })
}

// ========== 服务器资源 ==========

/** 获取服务器资源使用情况 */
export function getServerStats(server_id: number) {
  return request<Deploy.ServerStatsResponseData>({
    url: "deploy/server-stats",
    method: "get",
    params: { server_id }
  })
}

/** 批量获取服务器资源使用情况 */
export function getServerStatsBatch(data: Deploy.GetServerStatsBatchReq) {
  return request<Deploy.GetServerStatsBatchResponseData>({
    url: "deploy/server-stats/batch",
    method: "post",
    data
  })
}

// ========== 文件上传（仅 server 和 wukongim） ==========

/** 上传文件到服务器 */
export function uploadServerFile(form: FormData, onProgress?: (percent: number, evt: AxiosProgressEvent) => void) {
  return request<Deploy.UploadToServerResponseData>({
    url: "deploy/upload",
    method: "post",
    data: form,
    timeout: 180000,
    onUploadProgress: (evt: AxiosProgressEvent) => {
      const total = evt.total || 0
      if (total > 0) {
        const percent = Math.round((evt.loaded * 100) / total)
        onProgress?.(percent, evt)
      } else {
        onProgress?.(0, evt)
      }
    }
  })
}

// ========== 配置文件 ==========

/** 获取程序配置文件 */
export function getProgramConfig(params: Deploy.GetConfigFileReq) {
  return request<Deploy.ConfigFileResponseData>({
    url: "deploy/config",
    method: "get",
    params
  })
}

/** 更新程序配置文件 */
export function updateProgramConfig(data: Deploy.UpdateConfigFileReq) {
  return request<Deploy.ConfigFileResponseData>({
    url: "deploy/config",
    method: "post",
    data
  })
}

// ========== GOST API 代理 ==========

/** 获取 GOST 服务列表 */
export function listGostServices(params: Deploy.ListGostServicesReq) {
  return request<Deploy.GostServiceListResponseData>({
    url: "deploy/gost/services",
    method: "get",
    params
  })
}

/** 获取 GOST 服务详情 */
export function getGostServiceDetail(params: { server_id: number; service_name: string }) {
  return request<Deploy.GostServiceDetailResponseData>({
    url: `deploy/gost/services/${encodeURIComponent(params.service_name)}`,
    method: "get",
    params: { server_id: params.server_id }
  })
}

/** 更新 GOST 服务配置 */
export function updateGostServiceDetail(params: { server_id: number; service_name: string; config: Deploy.GostServiceConfig }) {
  return request({
    url: `deploy/gost/services/${encodeURIComponent(params.service_name)}`,
    method: "put",
    params: { server_id: params.server_id },
    data: params.config
  })
}

/** 创建 GOST 服务 */
export function createGostServiceByAPI(data: Deploy.CreateGostServiceReq) {
  return request({
    url: "deploy/gost/services",
    method: "post",
    data
  })
}

/** 删除 GOST 服务 */
export function deleteGostServiceByAPI(params: Deploy.DeleteGostServiceReq) {
  return request({
    url: `deploy/gost/services/${encodeURIComponent(params.service_name)}`,
    method: "delete",
    params: { server_id: params.server_id }
  })
}

/** 获取 GOST Chain 列表 */
export function listGostChains(params: { server_id: number }) {
  return request<Deploy.GostChainListResponseData>({
    url: "deploy/gost/chains",
    method: "get",
    params
  })
}

// ========== Docker 容器状态 ==========

/** 获取 Docker 容器状态 */
export function getDockerContainers(server_id: number) {
  return request<Deploy.DockerContainersResponseData>({
    url: "deploy/docker/containers",
    method: "get",
    params: { server_id }
  })
}

// ========== 批量分发 ==========

/** 上传文件到本地（用于批量分发） */
export function uploadToLocal(form: FormData, onProgress?: (percent: number, evt: AxiosProgressEvent) => void) {
  return request<Deploy.UploadToLocalResponseData>({
    url: "deploy/upload-local",
    method: "post",
    data: form,
    timeout: 180000,
    onUploadProgress: (evt: AxiosProgressEvent) => {
      const total = evt.total || 0
      if (total > 0) {
        const percent = Math.round((evt.loaded * 100) / total)
        onProgress?.(percent, evt)
      } else {
        onProgress?.(0, evt)
      }
    }
  })
}

/** 批量分发文件（从本地分发到目标服务器） */
export function distributeFile(data: Deploy.DistributeFileReq) {
  return request<Deploy.DistributeFileResponseData>({
    url: "deploy/distribute",
    method: "post",
    data,
    timeout: 600000 // 10分钟超时
  })
}

// ========== GOST 一键部署 ==========

/** 获取 GOST 部署默认配置 */
export function getGostDeployConfig(region_id?: string) {
  return request<{ code: number; data: Deploy.GostDeployConfig }>({
    url: "deploy/gost/deploy/config",
    method: "get",
    params: { region_id }
  })
}

// ========== 批量运维操作 ==========

/** 批量服务操作（start/stop/restart） */
export function batchServiceAction(data: Deploy.BatchServiceActionReq) {
  return request<Deploy.BatchServiceActionResponseData>({
    url: "deploy/batch/service-action",
    method: "post",
    data,
    timeout: 300000 // 5分钟超时
  })
}

/** 批量健康检查 */
export function batchHealthCheck(data: Deploy.BatchHealthCheckReq) {
  return request<Deploy.BatchHealthCheckResponseData>({
    url: "deploy/batch/health-check",
    method: "post",
    data,
    timeout: 120000 // 2分钟超时
  })
}

/** 批量执行命令 */
export function batchCommand(data: Deploy.BatchCommandReq) {
  return request<Deploy.BatchCommandResponseData>({
    url: "deploy/batch/command",
    method: "post",
    data,
    timeout: 300000 // 5分钟超时
  })
}

// ========== 日志查询 ==========

/** 统一日志查询 */
export function queryLogs(data: Deploy.LogQueryReq) {
  return request<Deploy.LogQueryResponseData>({
    url: "deploy/logs/query",
    method: "post",
    data,
    timeout: 60000 // 1分钟超时
  })
}

// ========== TSDD AMI 部署 ==========

/** 使用 AMI 部署 TSDD 服务器 */
export function deployTSDDWithAMI(data: Deploy.DeployTSDDWithAMIReq) {
  return request<Deploy.DeployTSDDWithAMIResponseData>({
    url: "deploy/tsdd/deploy-ami",
    method: "post",
    data,
    timeout: 600000 // 10分钟超时，AMI 部署需要时间
  })
}

// ========== TSDD Docker 部署 ==========

/** 部署 TSDD 到已注册服务器 */
export function deployTSDD(data: Deploy.DeployTSDDReq) {
  return request<Deploy.DeployTSDDResponseData>({
    url: "deploy/tsdd/deploy",
    method: "post",
    data,
    timeout: 600000 // 10分钟超时
  })
}

/** 通过IP部署 TSDD（新服务器） */
export function deployTSDDByIP(data: Deploy.DeployTSDDByIPReq) {
  return request<Deploy.DeployTSDDResponseData>({
    url: "deploy/tsdd/deploy-by-ip",
    method: "post",
    data,
    timeout: 600000
  })
}

/** 获取服务器部署状态 */
export function getDeployStatus(server_id: number) {
  return request<Deploy.GetDeployStatusResponseData>({
    url: "deploy/tsdd/status",
    method: "get",
    params: { server_id }
  })
}

// ========== 集群节点部署 ==========

/** 部署集群节点（支持水平扩容） */
export function deployNode(data: Deploy.DeployNodeReq) {
  return request<Deploy.DeployNodeResponseData>({
    url: "deploy/tsdd/deploy-node",
    method: "post",
    data,
    timeout: 600000 // 10分钟超时
  })
}

// ========== GOST 一键部署（流式API） ==========

/** 一键部署 GOST 转发服务器（流式） */
export function deployGostServer(data: Deploy.DeployGostServerReq, onData: (chunk: any, isComplete?: boolean) => void, onError?: (err: any) => void) {
  return createStreamRequest({
    url: "deploy/gost/deploy",
    method: "post",
    data,
    timeout: 600000
  }, onData, onError)
}

/** 在已有服务器上安装 GOST（流式） */
export function installGostToServer(data: Deploy.InstallGostReq, onData: (chunk: any, isComplete?: boolean) => void, onError?: (err: any) => void) {
  return createStreamRequest({
    url: "deploy/gost/install",
    method: "post",
    data,
    timeout: 600000
  }, onData, onError)
}

// ========== Nginx 安装（流式API） ==========

/** 安装 Nginx 到系统服务器（流式） */
export function installNginx(data: Deploy.InstallNginxReq, onData: (chunk: any, isComplete?: boolean) => void, onError?: (err: any) => void) {
  return createStreamRequest({
    url: "deploy/nginx/install",
    method: "post",
    data,
    timeout: 600000
  }, onData, onError)
}

// ========== GOST 转发配置（一键部署） ==========

/** 配置 GOST 转发目标 */
export function setupGostForward(data: Deploy.SetupGostForwardReq) {
  return request<{ code: number; data: { message: string }; message: string }>({
    url: "deploy/gost/forward/setup",
    method: "post",
    data
  })
}

/** 清除 GOST 转发规则 */
export function clearGostForward(data: Deploy.ClearGostForwardReq) {
  return request<{ code: number; data: { message: string }; message: string }>({
    url: "deploy/gost/forward/clear",
    method: "delete",
    data
  })
}

/** 获取 GOST 转发状态 */
export function getGostForwardStatus(server_id: number) {
  return request<Deploy.GostForwardStatusResponseData>({
    url: "deploy/gost/forward/status",
    method: "get",
    params: { server_id }
  })
}

// ========== GOST 配置持久化 ==========

/** 持久化 GOST 运行配置到文件 */
export function persistGostConfig(data: { server_id: number }) {
  return request<{ code: number; data: { message: string }; message: string }>({
    url: "deploy/gost/config/persist",
    method: "post",
    data,
    timeout: 30000
  })
}

/** 获取 GOST 配置同步状态 */
export function getGostConfigSyncStatus(server_id: number) {
  return request<Deploy.GostConfigSyncStatusResponseData>({
    url: "deploy/gost/config/sync-status",
    method: "get",
    params: { server_id }
  })
}

// ========== Nginx 缓存管理 ==========

/** 获取 Nginx 缓存状态 */
export function getNginxCacheStatus(server_id: number) {
  return request<Deploy.NginxCacheStatusResponseData>({
    url: "deploy/nginx/cache/status",
    method: "get",
    params: { server_id }
  })
}

/** 清除 Nginx 缓存 */
export function clearNginxCache(data: Deploy.ClearNginxCacheReq) {
  return request<{ code: number; data: { message: string }; message: string }>({
    url: "deploy/nginx/cache/clear",
    method: "post",
    data
  })
}

// ========== TLS 证书管理 ==========

/** 获取当前有效证书 */
export function getTlsCerts() {
  return request<Deploy.TlsCertsResponseData>({
    url: "deploy/tls/certs",
    method: "get"
  })
}

/** 生成 CA + 服务器证书 */
export function generateTlsCerts(data?: Deploy.GenerateTlsCertReq) {
  return request<Deploy.GenerateTlsCertResponseData>({
    url: "deploy/tls/certs/generate",
    method: "post",
    data
  })
}

/** 停用当前证书 */
export function disableTlsCerts() {
  return request({
    url: "deploy/tls/certs/disable",
    method: "post"
  })
}

/** 获取证书指纹（供 App 端 Pinning） */
export function getTlsCertFingerprint() {
  return request<Deploy.CertFingerprintResponseData>({
    url: "deploy/tls/certs/fingerprint",
    method: "get"
  })
}

/** 查看所有系统服务器 TLS 状态 */
export function getTlsStatus() {
  return request<Deploy.TlsStatusResponseData>({
    url: "deploy/tls/status",
    method: "get"
  })
}

/** 验证所有系统服务器 TLS 连接 */
export function verifyTlsStatus() {
  return request<Deploy.TlsStatusResponseData>({
    url: "deploy/tls/verify",
    method: "post",
    timeout: 120000 // 2分钟超时
  })
}

/** 批量升级为 TLS */
export function batchUpgradeTls(data?: Deploy.BatchTlsReq) {
  return request<Deploy.BatchTlsResponseData>({
    url: "deploy/tls/upgrade",
    method: "post",
    data,
    timeout: 300000 // 5分钟超时
  })
}

/** 批量回滚为 TCP */
export function batchRollbackTls(data?: Deploy.BatchTlsReq) {
  return request<Deploy.BatchTlsResponseData>({
    url: "deploy/tls/rollback",
    method: "post",
    data,
    timeout: 300000
  })
}
