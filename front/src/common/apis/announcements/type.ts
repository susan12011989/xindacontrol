import type { ApiResponseData } from "../type"

export interface MessageEntityDTO {
  type: "bold" | "italic" | "underline" | "strike" | "code" | "text_url"
  offset: number
  length: number
  url?: string
}

export interface CreateAnnouncementReq {
  text: string
  entities?: MessageEntityDTO[]
  silent?: boolean
  noforwards?: boolean
  merchant_ids?: number[]
}

export type CreateAnnouncementResp = ApiResponseData<{ accepted: boolean }>

export interface QueryAnnouncementLogsReq {
  page: number
  size: number
}

export interface AnnouncementLogItem {
  id: number
  text: string
  merchant_nos: string[]
  broadcast: boolean
  created_at: string
}

export interface QueryAnnouncementLogsData {
  list: AnnouncementLogItem[]
  total: number
}

export type QueryAnnouncementLogsResp = ApiResponseData<QueryAnnouncementLogsData>
