import type * as CommonTypes from "../../apis/type"
import type * as Types from "./type"
import { request } from "@/http/axios"

// 查询共享带宽列表
export function getBandwidthList(data: CommonTypes.ListRequestData) {
  return request<Types.BandwidthList>({
    url: "/cloud/vpc/bandwidth",
    method: "get",
    params: data
  })
}

// 创建共享带宽
export function createBandwidth(data: Types.CreateBandwidthRequestData) {
  return request<Types.BandwidthList>({
    url: "/cloud/vpc/bandwidth",
    method: "post",
    data
  })
}

// 操作共享带宽
export function operateBandwidth(data: Types.OperateBandwidthPackageRequestData) {
  return request<Types.BandwidthList>({
    url: "/cloud/vpc/bandwidth/operate",
    method: "post",
    data
  })
}
