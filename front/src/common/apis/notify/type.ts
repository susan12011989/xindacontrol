export interface NotifyMessage {
  typ: string // success,warning,error
  title: string // 标题
  message: string // 内容
  href: string // 链接
  duration: number // 持续时间
}

export type NotifyResponseData = ApiResponseData<{ notify: NotifyMessage[] }>
