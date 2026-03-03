/**
 * 腾讯云 CVM 实例类型定义
 */

/** 通用请求参数 */
export interface ListRequestData {
  merchant_id?: number
  cloud_account_id?: number
  region_id: string[]
}

/** 实例公网IP信息 */
export interface PublicIpAddresses {
  PublicIpAddress: string[]
}

/** 实例私网IP信息 */
export interface PrivateIpAddresses {
  PrivateIpAddress: string[]
}

/** 实例数据盘信息 */
export interface DataDisk {
  DiskSize: number // 数据盘大小，单位：GB
  DiskType: string // 数据盘类型
  DiskId: string // 数据盘ID
  DeleteWithInstance: boolean // 是否随实例释放
}

/** 实例系统盘信息 */
export interface SystemDisk {
  DiskSize: number // 系统盘大小，单位：GB
  DiskType: string // 系统盘类型
  DiskId: string // 系统盘ID
}

/** 实例安全组 */
export interface SecurityGroupIds {
  SecurityGroupId: string[]
}

/** VPC 信息 */
export interface VirtualPrivateCloud {
  VpcId: string // VPC ID
  SubnetId: string // 子网 ID
  AsVpcGateway: boolean // 是否用作公网网关
  Ipv6AddressCount: number // IPv6 地址数量
}

/** 实例网卡信息 */
export interface NetworkInterface {
  NetworkInterfaceId: string // 弹性网卡ID
  MacAddress: string // MAC地址
  PrivateIpAddress: string // 主私有IP
}

/** 实例信息 */
export interface Instance {
  CreatedTime: string // 创建时间
  InstanceId: string // 实例ID
  InstanceType: string // 实例规格
  InstanceChargeType: string // 付费类型 PREPAID：预付费 POSTPAID_BY_HOUR：按量计费
  InstanceState: string // 实例状态 PENDING/RUNNING/STOPPED/REBOOTING/TERMINATING/SHUTDOWN
  InstanceName: string // 实例名称
  CPU: number // CPU核数
  Memory: number // 内存大小，单位：GB
  OsName: string // 操作系统名称
  ImageId: string // 镜像ID
  Placement: {
    Zone: string // 可用区
    ProjectId: number // 项目ID
  }
  ExpiredTime: string // 到期时间（预付费实例）
  PublicIpAddresses: string[] // 公网IP地址列表
  PrivateIpAddresses: string[] // 私网IP地址列表
  SecurityGroupIds: string[] // 安全组ID列表
  SystemDisk: SystemDisk // 系统盘信息
  DataDisks: DataDisk[] // 数据盘信息
  VirtualPrivateCloud: VirtualPrivateCloud // VPC信息
  RestrictState: string // 实例限制状态
  RenewFlag: string // 自动续费标识
  UUID: string // 实例UUID
}

/** 实例列表响应 */
export type InstanceList = ApiResponseData<{
  list: Instance[]
  total: number
}>

/** 创建实例请求数据 */
export interface CreateInstanceData {
  merchant_id?: number // 商户ID（商户类型时必填）
  cloud_account_id?: number // 系统云账号ID（系统类型时必填）
  region_id: string // 区域
  zone: string // 可用区
  image_id?: string // 镜像ID（留空自动使用 Ubuntu 公共镜像）
  instance_type: string // 实例规格
  instance_charge_type: string // 付费类型 PREPAID/POSTPAID_BY_HOUR
  system_disk_type: string // 系统盘类型
  system_disk_size: number // 系统盘大小
  vpc_id?: string // VPC ID
  subnet_id?: string // 子网 ID
  security_group_ids?: string[] // 安全组 ID 列表
  instance_name?: string // 实例名称
  password?: string // 登录密码
  internet_max_bandwidth_out?: number // 公网出带宽（Mbps）
  // 预付费时额外参数
  period?: number // 购买时长（月）
  renew_flag?: string // 自动续费标识 NOTIFY_AND_AUTO_RENEW/NOTIFY_AND_MANUAL_RENEW/DISABLE_NOTIFY_AND_MANUAL_RENEW
}

/** 批量创建实例请求 */
export interface CreateInstancesRequestData {
  list: CreateInstanceData[]
}

/** VPC 信息 */
export interface VpcItem {
  VpcId: string
  VpcName: string
  CidrBlock: string
  IsDefault: boolean
  CreatedTime: string
}

/** VPC 列表响应 */
export type VpcListResponse = ApiResponseData<{
  list: VpcItem[]
  total: number
}>

/** 子网信息 */
export interface SubnetItem {
  SubnetId: string
  SubnetName: string
  CidrBlock: string
  VpcId: string
  Zone: string
  AvailableIpAddressCount: number
  IsDefault: boolean
  CreatedTime: string
}

/** 子网列表响应 */
export type SubnetListResponse = ApiResponseData<{
  list: SubnetItem[]
  total: number
}>

/** 实例操作请求 */
export interface OperateInstanceRequestData {
  merchant_id?: number // 商户ID
  cloud_account_id?: number // 云账号ID
  region_id: string // 区域
  instance_id: string // 实例ID
  operation: "start" | "stop" | "restart" | "delete" // 操作类型
}

/** 批量操作实例请求 */
export interface BatchOperateInstanceRequestData {
  merchant_id?: number // 商户ID
  cloud_account_id?: number // 云账号ID
  region_id: string // 区域
  instance_ids: string[] // 实例ID列表
  operation: "start" | "stop" | "restart" | "delete" // 操作类型
}

/** 修改实例属性请求 */
export interface ModifyInstanceAttributeRequestData {
  merchant_id?: number // 商户ID
  cloud_account_id?: number // 云账号ID
  region_id: string // 区域
  instance_id: string // 实例ID
  instance_name?: string // 实例名称
  security_group_ids?: string[] // 安全组ID列表
}

/** 重置实例密码请求 */
export interface ResetInstancePasswordRequestData {
  merchant_id?: number // 商户ID
  cloud_account_id?: number // 云账号ID
  region_id: string // 区域
  instance_id: string // 实例ID
  password: string // 新密码
}

/** 镜像信息 */
export interface Image {
  ImageId: string // 镜像ID
  ImageName: string // 镜像名称
  OsName: string // 操作系统名称
  ImageType: string // 镜像类型 PRIVATE_IMAGE/PUBLIC_IMAGE/SHARED_IMAGE
  CreatedTime: string // 创建时间
  ImageState: string // 镜像状态
  ImageSize: number // 镜像大小（GB）
  Platform: string // 操作系统平台
  Architecture: string // 架构
}

/** 镜像列表响应 */
export type ImageList = ApiResponseData<{
  list: Image[]
  total: number
}>

/** 实例规格信息 */
export interface InstanceTypeConfig {
  Zone: string // 可用区
  InstanceType: string // 实例规格
  InstanceFamily: string // 实例规格族
  GPU: number // GPU核数
  CPU: number // CPU核数
  Memory: number // 内存大小（GB）
  CbsSupport: string // 是否支持云硬盘
  InstanceTypeState: string // 规格状态
}

/** 实例规格列表响应 */
export type InstanceTypeList = ApiResponseData<{
  list: InstanceTypeConfig[]
  total: number
}>
