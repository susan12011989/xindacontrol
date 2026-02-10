/** 消息查询请求 */
export interface QueryMessagesReq {
  user_id: number
  peer_user_id?: number // 对方用户ID，可选
  page?: number
  size?: number
}

/** 用户搜索请求 */
export interface SearchUsersReq {
  keyword: string
  search_type: "id" | "phone" | "username" | "name"
}

export interface MessageMedia {
  kind: string
  mime_type: string
  size: number
  bucket: string
  object_path: string
  url: string
  thumbnail_url: string
}

/** 消息响应 */
export interface MessageResp {
  id: number
  user_message_box_id: number
  user_id: number
  dialog_message_id: number
  message: string
  sender: UserSearchResp | null // 发送者信息
  peer: UserSearchResp | null // 接收者信息
  deleted: number
  created_at: string
  media?: MessageMedia | null
}

/** 用户搜索响应 */
export interface UserSearchResp {
  id: number
  first_name: string
  last_name: string
  username: string
  phone: string
}

/** 查询所有消息请求 */
export interface QueryAllMessagesReq {
  message_type: "private" | "channel" // 消息类型：private（私聊）或 channel（群聊）
  start_time?: string // 开始时间，格式：2024-01-01 00:00:00
  end_time?: string // 结束时间，格式：2024-01-01 23:59:59
  page?: number
  size?: number
}

/** 群聊消息响应 */
export interface ChannelMessageResp {
  id: number
  channel_id: number
  channel_message_id: number
  message: string
  sender: UserSearchResp | null // 发送者信息
  deleted: number
  views: number
  pinned: number
  created_at: string
  media?: MessageMedia | null
}
