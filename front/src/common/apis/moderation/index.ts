import type { ApiResponseData } from "@/common/apis/type"

import { request } from "@/http/axios"

export interface QueryReportsReq {
  page: number
  size: number
  peer_type?: number // 2 用户 4 群组
  peer_id?: number
  user_id?: number
  order?: string
}

export interface ReportResp {
  id: number
  peer_type: number
  peer_id: number
  reason: number
  text: string
  photo: string
  message_id: number
  story_id: number
  user_id: number
  banned: boolean
  created_at: string
  phone?: string
}

export type ReportsListResponseData = ApiResponseData<{ list: ReportResp[], total: number }>

export function getReports(params: QueryReportsReq) {
  return request<ReportsListResponseData>({
    url: "moderation/reports",
    method: "get",
    params
  })
}
