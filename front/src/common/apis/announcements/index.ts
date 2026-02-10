import type { CreateAnnouncementReq, CreateAnnouncementResp, QueryAnnouncementLogsReq, QueryAnnouncementLogsResp } from "./type"
import { request } from "@/http/axios"

export function createAnnouncementApi(data: CreateAnnouncementReq) {
  return request<CreateAnnouncementResp>({
    url: "announcements",
    method: "post",
    data
  })
}

export function getAnnouncementLogsApi(params: QueryAnnouncementLogsReq) {
  return request<QueryAnnouncementLogsResp>({
    url: "announcements/logs",
    method: "get",
    params
  })
}
