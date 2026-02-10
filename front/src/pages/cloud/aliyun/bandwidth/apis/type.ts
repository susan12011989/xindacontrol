export interface Bandwidth {
  BandwidthPackageId: string // 共享带宽ID
  Bandwidth: string // 带宽峰值
  ReservationActiveTime: string // 续费生效时间
  Status: string // 状态
  /**
   * 共享带宽实例的计费类型。
   * PostPaid：按量计费。
   * PrePaid：包年包月。
   */
  InstanceChargeType: string
  /**
   * 公网IP的计费方式。
   * PayByBandwidth：按带宽计费。
   * PayByTraffic：按流量计费。
   */
  InternetChargeType: string
  RegionId: string // 区域ID
  Name: string // 名称
  Description: string // 描述
  ExpiredTime: string // 共享带宽实例的过期时间
  /**
   * 续费变配方式。
   * RENEWCHANGE：续费变配。
   * TEMP_UPGRADE：短时升配。
   * UPGRADE ：升级。
   */
  ReservationOrderType: string
  /**
   * 续费付费类型。
   * PayByBandwidth：按固定带宽计费。
   * PayByTraffic：按使用流量计费。
   */
  ReservationInternetChargeType: string
  ReservationBandwidth: string // 变配之后的带宽值， 单位：Mbps
  HasReservationData: string
  PublicIpAddresses: { // 共享带宽实例中的公网 IP 地址。
    PublicIpAddresse: {
      IpAddress: string // 公网 IP 地址。
      AllocationId: string // 弹性公网IP的ID。
      /**
       * 弹性公网IP与共享带宽实例的关联状态。
       * BINDED: EIP 与共享带宽关联完成
       * BINDING: 关联中
       */
      BandwidthPackageIpRelationStatus: string
    }[]
  }
  Zone: string // 可用区
}

// 查询共享带宽列表
export type BandwidthList = ApiResponseData<{ list: Bandwidth[], total: number }>

// 创建共享带宽
export interface CreateBandwidthRequestData {
  merchant_id?: number // 商户id
  cloud_account_id?: number // 系统云账号id
  region_id: string // 区域
  bandwidth: number // 带宽
}

// 操作共享带宽
export interface OperateBandwidthPackageRequestData {
  /**
   * 操作类型
   * modify: 修改属性
   * spec: 修改规格
   * addEip: 添加EIP
   * removeEip: 移除EIP
   * delete: 删除
   */
  operation: string
  merchant_id: number // 商户id
  region_id: string // 区域
  bandwidth_package_id: string // 共享带宽ID
  name?: string // modify 修改属性用
  description?: string // modify 修改属性用
  bandwidth?: string // spec 修改规格用
  ip_instance_ids?: string[] // addEip 添加EIP用
  ip_instance_id?: string // removeEip 移除EIP用
}
