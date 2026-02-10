import type * as CommonTypes from "../../apis/type"
import type * as Types from "./type"
import { createStreamRequest, request } from "@/http/axios"

// 获取安全组列表
export function getSecurityGroupList(data: CommonTypes.ListRequestData) {
  return request<Types.SecurityGroupList>({
    url: "/cloud/ecs/security-group",
    method: "get",
    params: data
  })
}

// 查询安全组详情
export function describeSecurityGroupAttribute(data: Types.DescribeSecurityGroupAttributeRequestData) {
  return request<Types.DescribeSecurityGroupAttributeResponseData>({
    url: "/cloud/ecs/security-group/attribute",
    method: "get",
    params: data
  })
}

// 创建安全组
export function createSecurityGroup(data: Types.CreateSecurityGroupRequestData, onData?: (output: string, isComplete?: boolean) => void, onError?: (error: any) => void) {
  if (onData) {
    // 使用流式API
    return createStreamRequest(
      {
        url: "/cloud/ecs/security-group",
        method: "post",
        data
      },
      (chunk, isComplete) => {
        const message = chunk.message as string
        const shouldComplete = isComplete === true || chunk.success === true
        console.log("处理安全组创建消息:", message, "是否完成:", shouldComplete)
        onData(message, shouldComplete)
      },
      onError
    )
  } else {
    // 兼容非流式方式
    return request<Types.SecurityGroup>({
      url: "/cloud/ecs/security-group",
      method: "post",
      data
    })
  }
}

// 删除安全组
export function deleteSecurityGroup(data: Types.DeleteSecurityGroupRequestData) {
  return request<Types.SecurityGroup>({
    url: "/cloud/ecs/security-group",
    method: "delete",
    data
  })
}

// 添加安全组入方向规则
export function authorizeSecurityGroup(data: Types.AuthorizeSecurityGroupRequestData) {
  return request<Types.SecurityGroup>({
    url: "/cloud/ecs/security-group/authorize",
    method: "post",
    data
  })
}

// 批量添加安全组入方向规则
export function authorizeSecurityGroupBatch(data: Types.AuthorizeSecurityGroupRequestBatchData) {
  return request<null>({
    url: "/cloud/ecs/security-group/authorize/batch",
    method: "post",
    data
  })
}

// 撤销全组入方向规则
export function revokeSecurityGroup(data: Types.RevokeSecurityGroupRequestData) {
  return request<Types.SecurityGroup>({
    url: "/cloud/ecs/security-group/revoke",
    method: "post",
    data
  })
}
