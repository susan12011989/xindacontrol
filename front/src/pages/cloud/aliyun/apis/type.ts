export interface ListRequestData {
  merchant_id?: number
  cloud_account_id?: number
  region_id: string[]
}

export interface Region {
  Status: string // 状态
  RegionId: string // 区域ID
  LocalName: string // 区域名称
  RegionEndpoint: string // 区域端点
}

export type RegionList = ApiResponseData<Region[]>
