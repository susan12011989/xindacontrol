/**
 * 腾讯云 CVM 实例 API
 */
import type * as Types from "./type"
import { createStreamRequest, request } from "@/http/axios"

/** 获取实例列表 */
export function getInstanceList(data: Types.ListRequestData) {
  return request<Types.InstanceList>({
    url: "/cloud/tencent/cvm/instances",
    method: "get",
    params: data
  })
}

/** 操作实例（启动/停止/重启/释放） */
export function operateInstance(data: Types.OperateInstanceRequestData) {
  return request<ApiResponseData<null>>({
    url: "/cloud/tencent/cvm/instance/operate",
    method: "post",
    data
  })
}

/** 批量操作实例 */
export function batchOperateInstance(data: Types.BatchOperateInstanceRequestData) {
  return request<ApiResponseData<null>>({
    url: "/cloud/tencent/cvm/instances/operate",
    method: "post",
    data
  })
}

/** 修改实例属性 */
export function modifyInstanceAttribute(data: Types.ModifyInstanceAttributeRequestData) {
  return request<ApiResponseData<null>>({
    url: "/cloud/tencent/cvm/instance/modify",
    method: "post",
    data
  })
}

/** 重置实例密码 */
export function resetInstancePassword(data: Types.ResetInstancePasswordRequestData) {
  return request<ApiResponseData<null>>({
    url: "/cloud/tencent/cvm/instance/reset-password",
    method: "post",
    data
  })
}

/** 获取镜像列表 */
export function getImageList(data: Types.ListRequestData) {
  return request<Types.ImageList>({
    url: "/cloud/tencent/cvm/images",
    method: "get",
    params: data
  })
}

/** 获取实例规格列表 */
export function getInstanceTypeList(data: Types.ListRequestData) {
  return request<Types.InstanceTypeList>({
    url: "/cloud/tencent/cvm/instance-types",
    method: "get",
    params: data
  })
}

/** 创建实例（流式 API） */
export function createInstances(data: Types.CreateInstancesRequestData, onData: (output: string, isComplete?: boolean) => void, onError?: (error: any) => void) {
  return createStreamRequest(
    {
      url: "/cloud/tencent/cvm/instances/create",
      method: "post",
      data
    },
    (chunk, isComplete) => {
      const message = chunk.message as string
      const shouldComplete = isComplete === true || chunk.success === true
      onData(message, shouldComplete)
    },
    onError
  )
}

/** 获取 VPC 列表 */
export function getVpcList(params: { merchant_id?: number; cloud_account_id?: number; region_id: string }) {
  return request<Types.VpcListResponse>({
    url: "/cloud/tencent/vpcs",
    method: "get",
    params
  })
}

/** 获取子网列表 */
export function getSubnetList(params: { merchant_id?: number; cloud_account_id?: number; region_id: string; vpc_id?: string }) {
  return request<Types.SubnetListResponse>({
    url: "/cloud/tencent/subnets",
    method: "get",
    params
  })
}
