// 实例弹性公网IP
export interface EipAddress {
  AllocationId: string // 弹性公网IP的ID
  IsSupportUnassociate: boolean // 是否支持解绑
  InternetChargeType: string // 公网IP的计费方式 PayByBandwidth：按带宽计费。PayByTraffic：按流量计费。
  IpAddress: string // 公网IP地址
  Bandwidth: number // 公网IP的带宽
}
// 实例vpc属性信息
export interface VpcAttributes {
  VpcId: string // 实例所属的VPC ID
  NatIpAddress: string // 云产品的 IP，用于 VPC 云产品之间的网络互通
  VSwitchId: string // 虚拟交换机 ID
  PrivateIpAddress: {
    IpAddress: string[] // 私有IP地址
  }
}
// 实例公网ip信息
export interface PublicIpAddress {
  IpAddress: string[] // 公网IP列表
}
// 单个网卡信息
export interface NetworkInterface {
  NetworkInterfaceId: string // 弹性网卡 ID
  Type: string // 网卡类型
  MacAddress: string // 网卡MAC地址
  PrimaryIpAddress: string // 弹性网卡主私有 IP 地址
  PrivateIpSets?: {
    PrivateIpSet: Array<{
      Primary: boolean
      PrivateIpAddress: string
      AssociatedPublicIp?: {
        PublicIpAddress: string // 绑定的公网IP
        AllocationId?: string // EIP分配ID
      }
    }>
  }
}
// 实例网卡信息
export interface NetworkInterfaces {
  NetworkInterface: NetworkInterface[] // 网卡列表
}
// 实例信息
export interface Instance {
  CreationTime: string // 实例创建时间
  InstanceId: string // 实例ID
  InstanceNetworkType: string // 实例网络类型 classic：经典网络 vpc：专有网络 VPC
  InstanceType: string // 实例规格
  InstanceChargeType: string // 实例付费类型 PrePaid：包年包月 PostPaid：按量付费
  Status: string // 实例状态
  InstanceName: string // 实例名称
  Description: string // 实例描述
  Cpu: number // 实例CPU核数
  Memory: number // 内存大小，单位为 MiB。
  OSName: string // 操作系统名称
  ImageId: string // 镜像ID
  RegionId: string // 实例所属的区域ID
  AutoReleaseTime: string // 按量付费实例的自动释放时间。
  StartTime: string // 实例最近一次的启动时间
  ExpiredTime: string // 实例到期时间
  NetworkInterfaces: NetworkInterfaces // 实例网卡信息
  PublicIpAddress: PublicIpAddress // 实例公网ip信息
  VpcAttributes: VpcAttributes // 实例vpc属性信息
  EipAddress: EipAddress // 实例弹性公网IP
  SecurityGroupIds: { // 实例所属安全组
    SecurityGroupId: string[]
  }
}

// 实例商户绑定信息
export interface InstanceBindingResp {
  instance_id: string
  merchant_id: number
  merchant_name: string
  merchant_no: string
}

export type InstanceList = ApiResponseData<{
  list: Instance[]
  total: number
  nic_eip_map?: Record<string, Array<{
    PrivateIpAddress: string
    PublicIpAddress: string
    AllocationId?: string
    Primary: boolean
  }>>
  bindings?: Record<string, InstanceBindingResp>
}>

export interface CreateInstanceData {
  merchant_id?: number // 商户id（商户类型时必填）
  cloud_account_id?: number // 系统云账号ID（系统类型时必填）
  region: string // 区域
  image_id: string // 镜像id
  instance_type: string // 规格
  instance_charge_type: string // 付费类型 PrePaid：包年包月 PostPaid：按量付费
  disk_category: string // 系统盘的云盘种类。取值范围：
  disk_size: number // 系统盘大小 40G
  // 付费类型为包年包月时，需要设置的参数
  period_unit?: string // 时长单位 Month：月 Week：周
  // PeriodUnit=Week 时，Period 取值：1、2、3、4
  // PeriodUnit=Month 时，Period 取值：1、2、3、4、5、6、7、8、9、12、24、36、48、60。
  period?: number // 时长
  // 自动续费（仅包年包月生效）
  auto_renew?: boolean // 是否自动续费
  auto_renew_period?: number // 自动续费周期（月），默认与购买周期一致
  // SSH认证信息（用于自动注册服务器）
  use_password?: boolean // 是否使用密码认证（true=密码，false=自动创建密钥对）
  password?: string // SSH登录密码，8-30个字符，必须包含大小写字母、数字
  key_pair_name?: string // 阿里云SSH密钥对名称
  ssh_private_key?: string // SSH私钥内容（PEM格式）
}

export interface CreateInstancesRequestData {
  List: CreateInstanceData[]
}

export interface OperateInstanceRequestData {
  merchant_id: number // 商户id
  region_id: string // 区域
  instance_id: string // 实例id
  operation: string | "start" | "stop" | "restart" | "delete" // 操作
}

export interface ModifyInstanceAttributeRequestData {
  merchant_id: number // 商户id
  region_id: string // 区域
  instance_id: string // 实例id
  instance_name: string // 实例名称
  description: string // 实例描述
  password: string // 密码
  security_group_id: string[] // 安全组
}

export interface Image {
  ImageId: string // 镜像ID
  ImageName: string // 镜像名称
  OSName: string // 操作系统名称
  CreationTime: string // 镜像创建时间
  Status: string // 镜像状态
  Progress: string // 镜像进度
  RegionId?: string // 区域ID
}

export type ImageList = ApiResponseData<{ list: Image[], total: number }>

// 包年包月实例 转为 按量付费实例
export interface ModifyInstanceChargeTypePostPaidRequestData {
  merchant_id: number // 商户id
  region_id: string // 区域
  instance_id: string // 实例id
}

// 创建辅助网卡接口参数
export interface InstanceItem {
  instance_id: string // 实例ID
  region_id: string // 区域ID
}

export interface CreateSecondaryNicRequestData {
  merchant_id: number // 商户ID
  instances: InstanceItem[] // 实例列表
}

// ========== 镜像共享相关类型 ==========

// 创建镜像请求
export interface CreateImageRequestData {
  merchant_id?: number // 商户ID（商户类型时必填）
  cloud_account_id?: number // 系统云账号ID（系统类型时必填）
  region_id: string // 区域ID
  instance_id: string // 实例ID
  image_name: string // 镜像名称
  description?: string // 镜像描述
}

// 创建镜像响应
export type CreateImageResponse = ApiResponseData<{
  image_id: string
}>

// 查询镜像共享权限请求
export interface DescribeImageShareRequestData {
  merchant_id?: number
  cloud_account_id?: number
  region_id: string
  image_id: string
}

// 镜像共享账号信息
export interface ShareAccount {
  aliyun_id: string // 阿里云账号ID
}

// 镜像共享权限响应
export type DescribeImageShareResponse = ApiResponseData<{
  image_id: string
  region_id: string
  share_accounts: ShareAccount[]
  total_count: number
}>

// 修改镜像共享权限请求
export interface ModifyImageShareRequestData {
  merchant_id?: number
  cloud_account_id?: number
  region_id: string
  image_id: string
  add_accounts?: string[] // 要添加共享的阿里云账号ID列表
  remove_accounts?: string[] // 要取消共享的阿里云账号ID列表
}

// ========== 一键部署隧道服务器 ==========

export interface DeployTunnelServerRequestData {
  cloud_account_id: number
  region_id: string
  server_name: string
  server_count: number
  instance_type: string
  bandwidth: string
  eip_count: number
}

// ========== 注册实例到服务器管理（自动创建SSH密钥） ==========

// 注册实例请求
export interface RegisterInstanceWithSSHKeyRequestData {
  cloud_account_id: number // 云账号ID
  region_id: string // 区域ID
  instance_id: string // 实例ID
  server_name: string // 服务器名称
  server_type: number // 服务器类型：1-商户服务器 2-系统服务器
  public_ip: string // 公网IP
}

// 注册实例响应
export type RegisterInstanceWithSSHKeyResponse = ApiResponseData<{
  server_id: number
  server_name: string
  host: string
  key_pair_name: string
  private_key: string // SSH私钥，用户需要下载保存
}>
