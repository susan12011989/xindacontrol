export interface PrivateIpSet {
  PrivateIpAddress: string // 实例的私网 IP 地址
  AssociatedPublicIp: AssociatedPublicIp // 弹性网卡辅助私有 IP 地址关联的弹性公网 IP
}
export interface PrivateIpSets {
  PrivateIpSet: PrivateIpSet[]
}
export interface AssociatedPublicIp {
  PublicIpAddress: string // 弹性公网 IP 地址
  AllocationId: string // eip实例id?
}

export interface NetworkInterface {
  NetworkInterfaceName: string // 网卡名称
  Description: string // 弹性网卡描述
  NetworkInterfaceId: string // 网卡ID
  InstanceId: string // 弹性网卡附加的ecs实例 ID。
  VSwitchId: string // VPC 的交换机 ID。
  CreationTime: string // 创建时间
  VpcId: string // 所属的VPC ID
  Type: string // 网卡类型 Primary：主网卡 Secondary：辅助网卡
  /**
   * Available：可用。
   * Attaching：附加中。
   * InUse：已附加。
   * Detaching：分离中。
   * Deleting：删除中。
   */
  Status: string
  MacAddress: string // MAC地址
  PrivateIpAddress: string // 弹性网卡的私网 IP 地址
  AssociatedPublicIp: AssociatedPublicIp // 弹性网卡辅助私有 IP 地址关联的弹性公网 IP
  PrivateIpSets: PrivateIpSets
  SecurityGroupIds: {
    SecurityGroupId: string[] // 安全组ID
  }
}

export interface NetworkInterfaceWrap {
  NetworkInterface: NetworkInterface // 详情
  RegionId: string // 区域ID
}

export type NetworkInterfaceList = ApiResponseData<{ list: NetworkInterfaceWrap[], total: number }>

// 创建弹性网卡
export interface CreateNetworkInterfaceRequestData {
  merchant_id: number // 商户id
  cloud_account_id: number // 云账号id
  region_id: string // 区域
  vswitch_id: string // VPC 的交换机 ID。
  security_group_id: string // 安全组ID
  network_interface_name: string // 弹性网卡名称
}

// 删除弹性网卡
export interface DeleteNetworkInterfaceRequestData {
  merchant_id: number // 商户id
  region_id: string // 区域
  network_interface_id: string // 网卡ID
}

// 绑定弹性网卡到ecs实例
export interface AttachNetworkInterfaceRequestData {
  merchant_id: number // 商户id
  region_id: string // 区域
  network_interface_id: string // 网卡ID
  instance_id: string // ECS实例ID
}

// 解绑弹性网卡
export interface DetachNetworkInterfaceRequestData {
  merchant_id: number // 商户id
  region_id: string // 区域
  network_interface_id: string // 网卡ID
  instance_id: string // ECS实例ID
}

// 修改弹性网卡属性
export interface ModifyNetworkInterfaceAttributeRequestData {
  merchant_id: number // 商户id
  region_id: string // 区域
  network_interface_id: string // 网卡ID
  network_interface_name: string // 弹性网卡名称
  description: string // 弹性网卡描述
  security_group_id: string[] // 最终加入的安全组，并会移出已有的安全组
}
