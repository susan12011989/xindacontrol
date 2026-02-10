import type * as Merchant from "./type"
import { request } from "@/http/axios"

export function getMerchantList(params: Merchant.QueryMerchantsReq) {
  return request<Merchant.QueryMerchantsResponseData>({
    url: "merchant",
    method: "get",
    params
  })
}
