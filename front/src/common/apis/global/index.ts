import type * as Global from "./type"
import { request } from "@/http/axios"

/** 获取OSS URL列表 */
export function getOssUrlList(params: Global.QueryOssUrlReq) {
  return request<Global.QueryOssUrlResponseData>({
    url: "global/oss-url",
    method: "get",
    params
  })
}

/** 创建OSS URL */
export function createOssUrl(data: Global.CreateOssUrlReq) {
  return request({
    url: "global/oss-url",
    method: "post",
    data
  })
}

/** 更新OSS URL */
export function updateOssUrl(id: number, data: Global.UpdateOssUrlReq) {
  return request({
    url: `global/oss-url/${id}`,
    method: "put",
    data
  })
}

/** 删除OSS URL */
export function deleteOssUrl(id: number) {
  return request({
    url: `global/oss-url/${id}`,
    method: "delete"
  })
}
