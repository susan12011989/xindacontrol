export interface AwsListReq {
  merchant_id?: number
  cloud_account_id?: number
  region_id: string[]
}

export interface AwsOperateEc2Req {
  merchant_id?: number
  cloud_account_id?: number
  region_id: string
  instance_id: string
  operation: "start" | "stop" | "reboot" | "terminate"
}

export interface AwsAuthorizeSecurityGroupReq {
  merchant_id?: number
  cloud_account_id?: number
  region_id: string
  group_id: string
  ip_protocol: string
  from_port?: number
  to_port?: number
  cidr_blocks?: string[]
}

export interface AwsOperateEipReq {
  merchant_id?: number
  cloud_account_id?: number
  region_id: string
  allocation_id: string
  operation: "associate" | "disassociate" | "release"
  instance_id?: string
  network_interface_id?: string
  private_ip_address?: string
}

export interface AwsS3ListBucketsReq {
  cloud_account_id: number
  region_id?: string
  prefix?: string
}

export interface AwsS3ListObjectsReq {
  merchant_id?: number
  cloud_account_id?: number
  region_id?: string
  bucket: string
  prefix?: string
  continuation_token?: string
  max_keys?: number
}

export interface AwsS3ObjectItem {
  key: string
  size: number
  etag: string
  last_modified: string
  storage_class: string
}

export interface AwsS3ListObjectsResponse {
  list: AwsS3ObjectItem[]
  is_truncated: boolean
  next_continuation_token: string
  total: number
}

export interface AwsS3SetBucketPublicReq {
  merchant_id?: number
  cloud_account_id?: number
  region_id?: string
  bucket: string
  public: boolean
}

export interface AwsCreateEc2InstanceReq {
  merchant_id?: number
  cloud_account_id?: number
  region_id: string
  image_id: string
  instance_type: string
  subnet_id?: string
  security_group_ids?: string[]
  key_name?: string
  volume_size_gib?: number
  instance_name?: string
}

export interface AwsListImagesReq {
  merchant_id?: number
  cloud_account_id?: number
  region_id: string
  owners?: string[]
  name?: string
  max_results?: number
}

export interface AwsImageItem {
  image_id: string
  name: string
  description: string
  owner_id: string
  creation_date: string
}

export interface AwsListInstanceTypesReq {
  merchant_id?: number
  cloud_account_id?: number
  region_id: string
  prefix?: string
  max_results?: number
}

export interface AwsInstanceTypeItem {
  instance_type: string
  vcpu: number
  memory_mib: number
}

export interface AwsListSubnetsReq {
  merchant_id?: number
  cloud_account_id?: number
  region_id: string
  vpc_id?: string
}

export interface AwsSubnetItem {
  subnet_id: string
  vpc_id: string
  cidr_block: string
  availability_zone: string
  name: string
}

export interface AwsListSecurityGroupsReq {
  merchant_id?: number
  cloud_account_id?: number
  region_id: string
  name?: string
}

export interface AwsSecurityGroupOption {
  group_id: string
  group_name: string
}

// Billing
export interface AwsBillingQueryReq {
  merchant_id?: number
  cloud_account_id?: number
  region_id?: string
  start: string // YYYY-MM-DD
  end: string // YYYY-MM-DD
  granularity?: "DAILY" | "MONTHLY"
  metrics?: string[]
  group_by_key?: string // e.g. SERVICE
}

export interface AwsBillingCostUsageResp {
  // 服务端透传的 Cost Explorer 响应或占位数据
  [key: string]: any
}

export interface AwsAllocateEipReq {
  merchant_id?: number
  cloud_account_id?: number
  region_id: string
}

export interface AwsDescribeInstanceReq {
  merchant_id?: number
  cloud_account_id?: number
  region_id: string
  instance_id: string
}

export interface AwsInstanceBrief {
  instance_id: string
  instance_type: string
  cpu: number
  memory_mib: number
  tags: Array<{ Key: string, Value: string }>
}

export interface AwsModifyEc2InstanceReq {
  merchant_id?: number
  cloud_account_id?: number
  region_id: string
  instance_id: string
  name?: string
  description?: string
  tags?: Record<string, string>
  security_group_ids?: string[]
}

export interface AwsResizeVolumeReq {
  merchant_id?: number
  cloud_account_id?: number
  region_id: string
  // 二选一：优先使用 volume_id；否则通过 instance_id(+device_name) 推导
  volume_id?: string
  instance_id?: string
  device_name?: string
  new_size_gib: number
  expand_fs?: boolean
}

export interface AwsListVolumesReq {
  merchant_id?: number
  cloud_account_id?: number
  region_id: string
  instance_id?: string
  volume_ids?: string[]
}

export interface AwsVolumeItem {
  volume_id: string
  size_gib: number
  volume_type: string
  state: string
  encrypted: boolean
  device_name: string
}

export interface AwsGetVolumeUsageReq {
  merchant_id?: number
  cloud_account_id?: number
  region_id: string
  instance_id: string
}

export interface AwsVolumeUsageItem {
  source: string
  mountpoint: string
  size_bytes: number
  used_bytes: number
  avail_bytes: number
  percent: number
}
