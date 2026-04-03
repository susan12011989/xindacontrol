import type { ApiResponseData } from "@/common/apis/type"

// ========== WuKongIM 监控 ==========

// WuKongIM 节点信息
export interface WuKongIMNode {
  server_id: number
  host: string
  merchant_no: string
  merchant_name: string
}

// 系统变量响应（/varz）
export interface WuKongIMVarzResp {
  server_id: string
  server_name: string
  version: string
  connections: number
  user_handler_count: number
  user_handler_conn_count: number
  uptime: string
  goroutine: number
  mem: number
  cpu: number
  in_msgs: number
  out_msgs: number
  in_bytes: number
  out_bytes: number
  slow_clients: number
  retry_queue: number
  tcp_addr: string
  ws_addr: string
  wss_addr: string
  manager_addr: string
  manager_on: number
  commit: string
  commit_date: string
  tree_state: string
  api_url: string
  manager_uid: string
  manager_token_on: number
}

// 连接信息
export interface WuKongIMConnInfo {
  id: number
  uid: string
  ip: string
  port: number
  last_activity: string
  uptime: string
  idle: string
  pending_bytes: number
  in_msgs: number
  out_msgs: number
  in_msg_bytes: number
  out_msg_bytes: number
  in_packets: number
  out_packets: number
  in_packet_bytes: number
  out_packet_bytes: number
  device: string
  device_id: string
  version: number
  proxy_type_format: string
  leader_id: number
  node_id: number
}

// 连接列表响应（/connz）
export interface WuKongIMConnzResp {
  connections: WuKongIMConnInfo[]
  now: string
  total: number
  offset: number
  limit: number
}

// 连接查询参数
export interface WuKongIMConnzReq {
  server_id: number
  offset?: number
  limit?: number
  uid?: string
  sort?: string
}

// 用户在线状态
export interface WuKongIMOnlineStatusItem {
  uid: string
  device_flag: number
  online: number
}

// 用户在线状态请求
export interface WuKongIMOnlineStatusReq {
  server_id: number
  uids: string[]
}

// 强制下线请求
export interface WuKongIMDeviceQuitReq {
  server_id: number
  uid: string
  device_flag: number
}

// API 响应类型
export type WuKongIMVarzResponseData = ApiResponseData<WuKongIMVarzResp>
export type WuKongIMConnzResponseData = ApiResponseData<WuKongIMConnzResp>
export type WuKongIMOnlineStatusResponseData = ApiResponseData<WuKongIMOnlineStatusItem[]>
