import type * as CommonTypes from "../../apis/type"
import type * as Types from "./type"
import { createStreamRequest, request } from "@/http/axios"

// 获取实例列表
export function getInstanceList(data: CommonTypes.ListRequestData) {
  return request<Types.InstanceList>({
    url: "/cloud/ecs/instance",
    method: "get",
    params: data
  })
}

// 创建实例 （流式API）
export function createInstance(data: Types.CreateInstancesRequestData, onData: (output: string, isComplete?: boolean) => void, onError?: (error: any) => void) {
  return createStreamRequest(
    {
      url: "/cloud/ecs/instance",
      method: "post",
      data
    },
    (chunk, isComplete) => {
      const message = chunk.message as string
      const shouldComplete = isComplete === true || chunk.success === true
      console.log("处理消息对象:", message, "是否完成:", shouldComplete)
      onData(message, shouldComplete)
    },
    onError
  )
}

// 操作实例
export function operateInstance(data: Types.OperateInstanceRequestData) {
  return request<Types.InstanceList>({
    url: "/cloud/ecs/instance/operate",
    method: "post",
    data
  })
}

// 修改实例属性
export function modifyInstanceAttribute(data: Types.ModifyInstanceAttributeRequestData) {
  return request<Types.InstanceList>({
    url: "/cloud/ecs/instance/modify",
    method: "post",
    data
  })
}

// 获取镜像列表
export function getImageList(data: CommonTypes.ListRequestData) {
  return request<Types.ImageList>({
    url: "/cloud/ecs/image",
    method: "get",
    params: data
  })
}

// 包年包月实例 转为 按量付费实例
export function modifyInstanceChargeTypePostPaid(data: Types.ModifyInstanceChargeTypePostPaidRequestData) {
  return request<null>({
    url: "/cloud/ecs/instance/modify-charge",
    method: "post",
    data
  })
}

// 创建辅助网卡 (流式API)
export function createSecondaryNic(data: Types.CreateSecondaryNicRequestData, onData: (output: string, isComplete?: boolean) => void, onError?: (error: any) => void) {
  return createStreamRequest(
    {
      url: "/cloud/ecs/instance/create-secondary-nic",
      method: "post",
      data
    },
    (chunk, isComplete) => {
      const message = chunk.message as string
      const shouldComplete = isComplete === true || chunk.success === true
      console.log("处理创建辅助网卡消息:", message, "是否完成:", shouldComplete)
      onData(message, shouldComplete)
    },
    onError
  )
}

// ========== 镜像共享相关 API ==========

// 创建镜像
export function createImage(data: Types.CreateImageRequestData) {
  return request<Types.CreateImageResponse>({
    url: "/cloud/ecs/image",
    method: "post",
    data
  })
}

// 查询镜像共享权限
export function describeImageSharePermission(data: Types.DescribeImageShareRequestData) {
  return request<Types.DescribeImageShareResponse>({
    url: "/cloud/ecs/image/share",
    method: "get",
    params: data
  })
}

// 修改镜像共享权限
export function modifyImageSharePermission(data: Types.ModifyImageShareRequestData) {
  return request<null>({
    url: "/cloud/ecs/image/share",
    method: "post",
    data
  })
}

// 注册实例到服务器管理（自动创建SSH密钥）
export function registerInstanceWithSSHKey(data: Types.RegisterInstanceWithSSHKeyRequestData) {
  return request<Types.RegisterInstanceWithSSHKeyResponse>({
    url: "/cloud/ecs/instance/register-with-ssh-key",
    method: "post",
    data
  })
}

// ========== 实例商户绑定 ==========

// 绑定商户
export function bindInstanceMerchant(data: { instance_id: string; region_id: string; merchant_id: number }) {
  return request<null>({
    url: "/cloud/ecs/instance/bind-merchant",
    method: "post",
    data
  })
}

// 解绑商户
export function unbindInstanceMerchant(data: { instance_id: string; cloud_type?: string }) {
  return request<null>({
    url: "/cloud/ecs/instance/unbind-merchant",
    method: "post",
    data
  })
}
