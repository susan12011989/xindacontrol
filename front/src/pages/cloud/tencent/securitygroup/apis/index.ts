import type * as Types from "./type"
import { createStreamRequest, request } from "@/http/axios"

/** 获取安全组列表 */
export function getSecurityGroupList(data: Types.ListSecurityGroupsReq) {
  return request<Types.SecurityGroupList>({
    url: "/cloud/tencent/security-groups",
    method: "get",
    params: data
  })
}

/** 查询安全组规则 */
export function describeSecurityGroupPolicies(data: Types.DescribeSecurityGroupPoliciesReq) {
  return request<Types.SecurityGroupPoliciesResponse>({
    url: "/cloud/tencent/security-group/policies",
    method: "get",
    params: data
  })
}

/** 添加入站规则 */
export function authorizeSecurityGroup(data: Types.AuthorizeSecurityGroupReq) {
  return request<ApiResponseData<null>>({
    url: "/cloud/tencent/security-group/authorize",
    method: "post",
    data
  })
}

/** 删除入站规则 */
export function revokeSecurityGroup(data: Types.RevokeSecurityGroupReq) {
  return request<ApiResponseData<null>>({
    url: "/cloud/tencent/security-group/revoke",
    method: "post",
    data
  })
}

/** 创建安全组（流式 API） */
export function createSecurityGroups(data: Types.CreateSecurityGroupsReq, onData: (output: string, isComplete?: boolean) => void, onError?: (error: any) => void) {
  return createStreamRequest(
    {
      url: "/cloud/tencent/security-groups/create",
      method: "post",
      data
    },
    (chunk, isComplete) => {
      const message = chunk.message as string
      const shouldComplete = isComplete === true || chunk.success === true
      onData(message, shouldComplete)
    },
    onError
  )
}
