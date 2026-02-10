import type * as Clients from "./type"
import { request } from "@/http/axios"

/** 获取客户端列表 */
export function getClientList(params: Clients.QueryClientsReq) {
  return request<Clients.QueryClientsResponseData>({
    url: "clients",
    method: "get",
    params
  })
}

/** 获取客户端详情 */
export function getClientDetail(id: number) {
  return request<Clients.ClientDetailResponseData>({
    url: `clients/${id}`,
    method: "get"
  })
}

/** 创建客户端 */
export function createClient(data: Clients.CreateClientReq) {
  return request({
    url: "clients",
    method: "post",
    data
  })
}

/** 更新客户端 */
export function updateClient(id: number, data: Clients.UpdateClientReq) {
  return request({
    url: `clients/${id}`,
    method: "put",
    data
  })
}

/** 删除客户端 */
export function deleteClient(id: number) {
  return request({
    url: `clients/${id}`,
    method: "delete"
  })
}
