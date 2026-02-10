import type { ApiResponseData } from "../type"
import type { ChannelMessageResp, MessageResp, QueryAllMessagesReq, QueryMessagesReq, SearchUsersReq, UserSearchResp } from "./type"
import { request } from "@/http/axios"

/** 获取用户消息列表 */
export function getUserMessagesApi(params: QueryMessagesReq) {
  return request<ApiResponseData<{ list: MessageResp[], total: number }>>({
    url: `/messages/user/${params.user_id}`,
    method: "GET",
    params: {
      peer_user_id: params.peer_user_id,
      page: params.page,
      size: params.size
    }
  })
}

/** 搜索用户 */
export function searchUsersApi(params: SearchUsersReq) {
  return request<ApiResponseData<UserSearchResp[]>>({
    url: "/messages/search-users",
    method: "GET",
    params
  })
}

/** 查询所有消息（私聊或群聊） */
export function getAllMessagesApi(params: QueryAllMessagesReq) {
  return request<ApiResponseData<{ list: MessageResp[] | ChannelMessageResp[], total: number }>>({
    url: "/messages/all",
    method: "GET",
    params
  })
}
