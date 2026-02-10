import type * as CommonTypes from "../../apis/type"
import type * as Types from "./type"
import { createStreamRequest, request } from "@/http/axios"

// 获取弹性公网IP列表
export function getEipList(data: CommonTypes.ListRequestData) {
  return request<Types.EipList>({
    url: "/cloud/vpc/eip",
    method: "get",
    params: data
  })
}

// 操作弹性公网IP
export function operateEip(data: Types.OperateEipReq) {
  return request<Types.EipList>({
    url: "/cloud/vpc/eip/operate",
    method: "post",
    data
  })
}

// 批量创建弹性公网IP
export function batchCreateEip(
  data: Types.BatchCreateEipRequestData,
  onData?: (data: string, isComplete?: boolean) => void,
  onError?: (error: any) => void
) {
  if (onData) {
    // 使用流式API
    return createStreamRequest(
      {
        url: "/cloud/vpc/eip",
        method: "post",
        data
      },
      (chunk, isComplete) => {
        const message = chunk.message as string
        const shouldComplete = isComplete === true || chunk.success === true
        console.log("处理EIP创建消息:", message, "是否完成:", shouldComplete)
        onData(message, shouldComplete)
      },
      onError
    )
  } else {
    // 兼容非流式方式
    return request<Types.EipList>({
      url: "/cloud/vpc/eip/batch",
      method: "post",
      data
    })
  }
}

// 批量智能绑定弹性公网IP（流式API）
export function batchAssociateEip(
  data: Types.BatchAssociateEipRequestData,
  onData: (output: string, isComplete?: boolean) => void,
  onError?: (error: any) => void
) {
  return createStreamRequest(
    {
      url: "/cloud/vpc/eip/batch-associate",
      method: "post",
      data
    },
    (chunk, isComplete) => {
      const message = chunk.message as string
      const shouldComplete = isComplete === true || chunk.success === true
      console.log("处理批量绑定EIP消息:", message, "是否完成:", shouldComplete)
      onData(message, shouldComplete)
    },
    onError
  )
}

// 更换弹性公网IP（流式API）
export function replaceEip(
  data: Types.ReplaceEipReq,
  onData: (output: string, isComplete?: boolean) => void,
  onError?: (error: any) => void
) {
  return createStreamRequest(
    {
      url: "/cloud/vpc/eip/replace",
      method: "post",
      data
    },
    (chunk, isComplete) => {
      const message = chunk.message as string
      const shouldComplete = isComplete === true || chunk.success === true
      console.log("处理更换EIP消息:", message, "是否完成:", shouldComplete)
      onData(message, shouldComplete)
    },
    onError
  )
}

// 批量更换弹性公网IP（流式API）
export function batchReplaceEip(
  data: Types.BatchReplaceEipReq,
  onData: (output: string, isComplete?: boolean) => void,
  onError?: (error: any) => void
) {
  return createStreamRequest(
    {
      url: "/cloud/vpc/eip/batch-replace",
      method: "post",
      data
    },
    (chunk, isComplete) => {
      const message = chunk.message as string
      const shouldComplete = isComplete === true || chunk.success === true
      console.log("处理批量更换EIP消息:", message, "是否完成:", shouldComplete)
      onData(message, shouldComplete)
    },
    onError
  )
}
