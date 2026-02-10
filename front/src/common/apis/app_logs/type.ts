import type { ApiResponseData } from "@@/apis/type"

export interface AppLogListItem {
  id: number
  uid: string
  log_type: string
  log_date: string
  file_name: string
  minio_path: string
  file_size: number
  device_info: string
  app_version: string
  created_at: string
  name: string
  phone: string
  short_no: string
}

export interface AppLogsQueryReq {
  merchant_no: string
  page: number
  size: number
  keyword?: string
}

export type AppLogsQueryResp = ApiResponseData<{
  list: AppLogListItem[]
  total: number
}>
