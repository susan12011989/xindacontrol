export interface Eip {
  AllocationId: string // ID
  /**
   * VpcId
   * 开通了 IPv4 网关功能且与 EIP 同地域的 VPC ID。
   * EIP 绑定 IP 地址时，系统可以根据该 VPC 的路由配置，使绑定的 IP 地址具备公网访问能力
   */
  VpcId: string
  RegionId: string // 区域ID
  IpAddress: string // 公网IP地址
  Bandwidth: string // 带宽峰值
  AllocationTime: string // EIP 的创建时间
  ChargeType: string // PostPaid：按量计费 PrePaid：包年包月。
  InternetChargeType: string // 公网IP的计费方式 PayByBandwidth：按带宽计费。PayByTraffic：按流量计费。
  Description: string // 弹性公网IP的描述信息
  BandwidthPackageId: string // 加入的共享带宽 ID。
  BandwidthPackageBandwidth: string // 共享带宽的带宽
  ExpiredTime: string // 到期时间
  BusinessStatus: string // 业务状态 Normal：正常 FinancialLocked：被锁定。

  /**
   * 当前绑定的实例类型。
   * EcsInstance：VPC 类型的 ECS 实例
   * SlbInstance：VPC 类型的 CLB 实例
   * Nat：NAT 网关
   * HaVip：高可用虚拟 IP
   * NetworkInterface：辅助弹性网卡
   * IpAddress：IP 地址
   */
  InstanceType: string
  InstanceId: string // 当前绑定的实例的 ID
  PrivateIpAddress: string // EIP 所绑定的辅助弹性网卡实例的私网 IP 地址。

  /**
   * 绑定模式
   * NAT：NAT 模式（普通模式）
   * MULTI_BINDED：多 EIP 网卡可见模式。
   * BINDED：EIP 网卡可见模式。
   */
  Mode: string
  ReservationActiveTime: string // 续费生效时间
  /**
   * 状态
   * Associating：绑定中。
   * Unassociating：解绑中。
   * InUse：已分配。
   * Available：可用。
   * Releasing：释放中。
   */
  Status: string
  /**
   * 续费订单类型
   * RENEWCHANGE：续费变配。
   * TEMP_UPGRADE：短时升配。
   * UPGRADE ：升级。
   */
  ReservationOrderType: string
  /**
   * 续费付费类型
   * PayByBandwidth：按固定带宽计费
   * PayByTraffic：按使用流量计费。
   */
  ReservationInternetChargeType: string
  ReservationBandwidth: string // 续费带宽
  Name: string // 名称
}
export type EipList = ApiResponseData<{ list: Eip[], total: number }>

export interface CreateEipRequestData {
  merchant_id: number // 商户id
  region_id: string // 区域
  instance_charge_type: string // 付费类型 PrePaid：包年包月 PostPaid：按量付费
  internet_charge_type: string // 公网IP的计费方式 PayByBandwidth：按带宽计费。PayByTraffic：按流量计费。
  bandwidth: string // 带宽
}

export interface CreateEipItemData {
  merchant_id?: number // 商户id（商户类型）
  cloud_account_id?: number // 系统云账号ID（系统类型）
  region_id: string // 区域
  instance_charge_type: string // 付费类型 PrePaid：包年包月 PostPaid：按量付费
  internet_charge_type: string // 公网IP的计费方式 PayByBandwidth：按带宽计费。PayByTraffic：按流量计费。
  bandwidth: string // 带宽
  num: number // 数量
}

export interface BatchCreateEipRequestData {
  list: CreateEipItemData[] // EIP列表
}

export interface OperateEipReq {
  merchant_id: number // 商户id
  region_id: string // 区域
  operation: string | "modify" | "associate" | "unassociate" | "delete" // 操作类型 modify：修改带宽 associate：绑定实例或网卡 unassociate：解绑 delete：删除
  allocation_id: string // 弹性公网IP的ID
  instance_type: string | "EcsInstance" | "NetworkInterface"// 实例类型
  instance_id: string // 实例/网卡ID
  // name 与 description 只用在 modify 操作中
  name: string // 弹性公网IP名称
  description: string // 弹性公网IP描述
}

// 批量绑定EIP接口参数
export interface EipItem {
  allocation_id: string // 弹性IP的ID
  region_id: string // 区域ID
}

export interface BatchAssociateEipRequestData {
  merchant_id: number // 商户ID
  eip_list: EipItem[] // EIP列表
}

// 更换弹性IP请求参数
export interface ReplaceEipReq {
  merchant_id?: number // 商户id（商户类型）
  cloud_account_id?: number // 系统云账号ID（系统类型）
  region_id: string // 区域
  allocation_id: string // 要更换的弹性IP的ID
  old_ip_address?: string // 旧EIP的IP地址（可选，传入可跳过查询）
  instance_id?: string // 绑定的实例ID（网卡ID）
  instance_type?: string // 实例类型：EcsInstance, NetworkInterface
  bandwidth_package_id?: string // 所属共享带宽ID
  bandwidth?: string // 新EIP带宽（默认继承原带宽）
  internet_charge_type?: string // 计费方式：PayByBandwidth, PayByTraffic
  isp?: string // 线路类型：BGP, BGP_PRO
  name?: string // 新EIP名称（默认继承原名称）
}

// 批量更换弹性IP配置项
export interface BatchReplaceEipConfig {
  region_id: string // 区域ID
  allocation_id: string // 要更换的弹性IP的ID
  old_ip_address?: string // 旧EIP的IP地址（可选，传入可跳过查询）
  instance_id?: string // 绑定的实例ID（网卡ID）
  instance_type?: string // 实例类型：EcsInstance, NetworkInterface
  bandwidth_package_id?: string // 所属共享带宽ID
  bandwidth?: string // 新EIP带宽（默认继承原带宽）
  internet_charge_type?: string // 计费方式：PayByBandwidth, PayByTraffic
}

// 批量更换弹性IP请求参数
export interface BatchReplaceEipReq {
  merchant_id?: number // 商户id（商户类型）
  cloud_account_id?: number // 系统云账号ID（系统类型）
  eip_list: BatchReplaceEipConfig[] // EIP配置列表
}
