import type { ApiResponseData } from "@@/apis/type"

export interface Pagination {
  page: number
  size: number
}

export interface QueryMerchantsReq extends Pagination {
  name?: string
  order?: string
}

export interface MerchantResp {
  id: number
  no: string
  name: string
  port?: number
  server_ip?: string
  status?: number
  expired_at?: string
}

export interface QueryMerchantsResponse {
  list: MerchantResp[]
  total: number
}

export type QueryMerchantsResponseData = ApiResponseData<QueryMerchantsResponse>
