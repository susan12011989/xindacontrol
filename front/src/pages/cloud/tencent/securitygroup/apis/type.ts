/** 安全组列表请求 */
export interface ListSecurityGroupsReq {
  merchant_id?: number
  cloud_account_id?: number
  region_id: string
}

/** 腾讯云安全组 (SDK 返回结构) */
export interface SecurityGroup {
  SecurityGroupId: string
  SecurityGroupName: string
  SecurityGroupDesc: string
  CreatedTime: string
  ProjectId: string
  IsDefault: boolean
}

/** 安全组列表响应 */
export type SecurityGroupList = ApiResponseData<{
  list: SecurityGroup[]
  total: number
}>

/** 查询安全组规则请求 */
export interface DescribeSecurityGroupPoliciesReq {
  merchant_id?: number
  cloud_account_id?: number
  region_id: string
  security_group_id: string
}

/** 安全组规则 */
export interface SecurityGroupPolicy {
  PolicyIndex: number
  Protocol: string // TCP, UDP, ICMP, ALL
  Port: string // "80", "8000-9000", "ALL"
  CidrBlock: string // "0.0.0.0/0"
  Ipv6CidrBlock: string
  SecurityGroupId: string
  Action: string // ACCEPT, DROP
  PolicyDescription: string
  ModifyTime: string
}

/** 安全组规则集 */
export interface SecurityGroupPolicySet {
  Ingress: SecurityGroupPolicy[]
  Egress: SecurityGroupPolicy[]
}

/** 安全组规则集响应 */
export type SecurityGroupPoliciesResponse = ApiResponseData<SecurityGroupPolicySet>

/** 添加入站规则请求 */
export interface AuthorizeSecurityGroupReq {
  merchant_id?: number
  cloud_account_id?: number
  region_id: string
  security_group_id: string
  policies: AuthorizeSecurityGroupPolicy[]
}

/** 添加入站规则 */
export interface AuthorizeSecurityGroupPolicy {
  protocol: string // TCP, UDP, ICMP, ALL
  port: string // "80", "8000-9000"
  cidr_block: string // "0.0.0.0/0"
  action: string // ACCEPT, DROP
  description?: string
}

/** 删除入站规则请求 */
export interface RevokeSecurityGroupReq {
  merchant_id?: number
  cloud_account_id?: number
  region_id: string
  security_group_id: string
  policy_indexes: number[]
}

/** 创建安全组数据 */
export interface CreateSecurityGroupData {
  merchant_id?: number
  cloud_account_id?: number
  region_id: string
  name: string
  description?: string
}

/** 批量创建安全组请求 */
export interface CreateSecurityGroupsReq {
  list: CreateSecurityGroupData[]
}
