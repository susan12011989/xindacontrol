import type * as CommonTypes from "../../apis/type"
import type * as Types from "./type"
import { request } from "@/http/axios"

// 获取弹性网卡列表
export function getNetworkInterfaceList(data: CommonTypes.ListRequestData) {
  return request<Types.NetworkInterfaceList>({
    url: "/cloud/ecs/network-interface",
    method: "get",
    params: data
  })
}

// 创建弹性网卡
export function createNetworkInterface(data: Types.CreateNetworkInterfaceRequestData) {
  return request<Types.NetworkInterface>({
    url: "/cloud/ecs/network-interface",
    method: "post",
    data
  })
}

// 删除弹性网卡
export function deleteNetworkInterface(data: Types.DeleteNetworkInterfaceRequestData) {
  return request<Types.NetworkInterface>({
    url: "/cloud/ecs/network-interface",
    method: "delete",
    data
  })
}

// 绑定弹性网卡到ecs实例
export function attachNetworkInterface(data: Types.AttachNetworkInterfaceRequestData) {
  return request<Types.NetworkInterface>({
    url: "/cloud/ecs/network-interface/attach",
    method: "post",
    data
  })
}

// 解绑弹性网卡
export function detachNetworkInterface(data: Types.DetachNetworkInterfaceRequestData) {
  return request<Types.NetworkInterface>({
    url: "/cloud/ecs/network-interface/detach",
    method: "post",
    data
  })
}

// 修改弹性网卡属性
export function modifyNetworkInterfaceAttribute(data: Types.ModifyNetworkInterfaceAttributeRequestData) {
  return request<Types.NetworkInterface>({
    url: "/cloud/ecs/network-interface/modify",
    method: "post",
    data
  })
}
