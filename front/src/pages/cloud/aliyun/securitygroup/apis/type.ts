export interface SecurityGroup {
  SecurityGroupId: string // 安全组ID
  SecurityGroupName: string // 安全组名称
  Description: string // 安全组描述
  VpcId: string // 所属VPC ID
  CreationTime: string // 创建时间
  EcsCount: number // 安全组中已经容纳的私网 IP 数量
}
export interface SecurityGroupWrap {
  RegionId: string
  SecurityGroup: SecurityGroup
}
// 安全组列表
export type SecurityGroupList = ApiResponseData<{ list: SecurityGroupWrap[], total: number }>

// 创建安全组
export interface CreateSecurityData {
  merchant_id?: number
  cloud_account_id?: number
  region_id: string
  name?: string
  description?: string
}
export interface CreateSecurityGroupRequestData {
  list: CreateSecurityData[]
}

// 删除安全组
export interface DeleteSecurityGroupRequestData {
  merchant_id: number
  region_id: string
  security_group_id: string
}

// 添加安全组入方向规则
export interface AuthorizeSecurityGroupRequestData {
  merchant_id: number
  region_id: string
  security_group_id: string
  permissions: AuthorizeSecurityGroupRequestPermissions[]
}

// 批量添加安全组入方向规则
export interface AuthorizeSecurityGroupRequestBatchData {
  merchant_id: number
  region_ids: string[]
  permissions: AuthorizeSecurityGroupRequestPermissions[]
}

export interface AuthorizeSecurityGroupRequestPermissions {
  IpProtocol: string // 协议类型
  PortRange: string // 端口范围
  SourceCidrIp: string // 设置源端 IPv4 地址
  Policy: string // 授权策略 Accept
}

// 撤销全组入方向规则
export interface RevokeSecurityGroupRequestData {
  merchant_id: number
  region_id: string
  security_group_id: string
  permissions: RevokeSecurityGroupRequestPermissions[]
}

export interface RevokeSecurityGroupRequestPermissions {
  IpProtocol: string // 协议类型
  PortRange: string // 端口范围
  SourceCidrIp: string // 设置源端 IPv4 地址
  Policy: string // 授权策略 Accept
}

// 查询安全组属性
export interface DescribeSecurityGroupAttributeRequestData {
  merchant_id: number
  region_id: string
  security_group_id: string
}

// 查询安全组规则
export interface DescribeSecurityGroupAttributeResponse {
  SecurityGroupId: string // 安全组ID
  SecurityGroupName: string // 安全组名称
  Description: string // 安全组描述
  VpcId: string // 所属VPC ID
  RegionId: string // 地域ID
  Permissions: { // 安全组规则
    Permission: DescribeSecurityGroupAttributeResponsePermissions[]
  }
}

export type DescribeSecurityGroupAttributeResponseData = ApiResponseData<DescribeSecurityGroupAttributeResponse>

export interface DescribeSecurityGroupAttributeResponsePermissions {
  Direction: string // 方向
  SourceGroupId: string // 源安全组ID
  IpProtocol: string // 协议类型
  SourceCidrIp: string // 设置源端 IPv4 地址
  PortRange: string // 端口范围
  Policy: string // 授权策略
  CreateTime: string // 创建时间
}
