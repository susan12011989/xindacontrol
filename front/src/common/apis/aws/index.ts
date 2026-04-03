import type * as Aws from "./type"
import { createStreamRequest, request } from "@/http/axios"

export function listEc2Instances(params: Aws.AwsListReq) {
  return request<any>({
    url: "aws/ec2/instance",
    method: "get",
    params
  })
}

export function operateEc2Instance(data: Aws.AwsOperateEc2Req) {
  return request<any>({
    url: "aws/ec2/instance/operate",
    method: "post",
    data
  })
}

export function listSecurityGroups(params: Aws.AwsListReq) {
  return request<any>({
    url: "aws/ec2/security-group",
    method: "get",
    params
  })
}

export function describeSecurityGroup(params: { merchant_id?: number, cloud_account_id?: number, region_id: string, group_id: string }) {
  return request<any>({
    url: "aws/ec2/security-group/attribute",
    method: "get",
    params
  })
}

export function authorizeSecurityGroup(data: Aws.AwsAuthorizeSecurityGroupReq) {
  return request<any>({
    url: "aws/ec2/security-group/authorize",
    method: "post",
    data
  })
}

export function revokeSecurityGroup(data: Aws.AwsAuthorizeSecurityGroupReq) {
  return request<any>({
    url: "aws/ec2/security-group/revoke",
    method: "post",
    data
  })
}

export function listEips(params: Aws.AwsListReq) {
  return request<any>({
    url: "aws/ec2/eip",
    method: "get",
    params
  })
}

export function operateEip(data: Aws.AwsOperateEipReq) {
  return request<any>({
    url: "aws/ec2/eip/operate",
    method: "post",
    data
  })
}

export function listBuckets(params: Aws.AwsS3ListBucketsReq) {
  return request<any>({
    url: "aws/s3/buckets",
    method: "get",
    params
  })
}

export function listObjects(params: Aws.AwsS3ListObjectsReq) {
  return request<{ code: number, data: Aws.AwsS3ListObjectsResponse, message: string }>({
    url: "aws/s3/objects",
    method: "get",
    params
  })
}

export function uploadObject(form: FormData, onUploadProgress?: (progressEvent: any) => void) {
  return request<any>({
    url: "aws/s3/object/upload",
    method: "post",
    headers: { "Content-Type": "multipart/form-data" },
    data: form,
    timeout: 300000, // 5分钟超时（大文件上传）
    onUploadProgress
  })
}

export function downloadObject(params: { merchant_id?: number, cloud_account_id?: number, region_id?: string, bucket: string, object_key: string, filename?: string, attachment?: number }) {
  return request<Blob>({
    url: "aws/s3/object/download",
    method: "get",
    responseType: "blob",
    params,
    timeout: 300000 // 5分钟超时（大文件下载）
  })
}

export function setBucketPublic(data: Aws.AwsS3SetBucketPublicReq) {
  return request<any>({
    url: "aws/s3/bucket/set-public",
    method: "post",
    data
  })
}

export function createEc2Instance(data: Aws.AwsCreateEc2InstanceReq) {
  return request<{ code: number, data: { instance_id: string }, message: string }>({
    url: "aws/ec2/instance/create",
    method: "post",
    data
  })
}

export function listImages(params: Aws.AwsListImagesReq) {
  return request<{ code: number, data: { list: Aws.AwsImageItem[] }, message: string }>({
    url: "aws/ec2/images",
    method: "get",
    params
  })
}

export function listInstanceTypes(params: Aws.AwsListInstanceTypesReq) {
  return request<{ code: number, data: { list: Aws.AwsInstanceTypeItem[] }, message: string }>({
    url: "aws/ec2/instance-types",
    method: "get",
    params
  })
}

export function listSubnets(params: Aws.AwsListSubnetsReq) {
  return request<{ code: number, data: { list: Aws.AwsSubnetItem[] }, message: string }>({
    url: "aws/ec2/subnets",
    method: "get",
    params
  })
}

export function listSecurityGroupOptions(params: Aws.AwsListSecurityGroupsReq) {
  return request<{ code: number, data: { list: Aws.AwsSecurityGroupOption[] }, message: string }>({
    url: "aws/ec2/security-groups/options",
    method: "get",
    params
  })
}

export function modifyEc2Instance(data: Aws.AwsModifyEc2InstanceReq) {
  return request({
    url: "aws/ec2/instance/modify",
    method: "post",
    data
  })
}

// removed non-stream resize API; use resizeVolumeStream instead

export function resizeInstanceTypeStream(data: Aws.AwsResizeEc2InstanceReq, onData: (chunk: any, isComplete?: boolean) => void, onError?: (err: any) => void) {
  return createStreamRequest({
    url: "aws/ec2/instance/resize/stream",
    method: "post",
    data,
    timeout: 600000
  }, onData, onError)
}

export function resizeVolumeStream(data: Aws.AwsResizeVolumeReq, onData: (chunk: any, isComplete?: boolean) => void, onError?: (err: any) => void) {
  return createStreamRequest({
    url: "aws/ec2/volume/resize/stream",
    method: "post",
    data,
    timeout: 600000
  }, onData, onError)
}

export function listVolumes(params: Aws.AwsListVolumesReq) {
  return request<{ code: number, data: { list: Aws.AwsVolumeItem[], total: number }, message: string }>({
    url: "aws/ec2/volumes",
    method: "get",
    params
  })
}

export function listVolumeUsage(params: Aws.AwsGetVolumeUsageReq) {
  return request<{ code: number, data: { list: Aws.AwsVolumeUsageItem[], total: number }, message: string }>({
    url: "aws/ec2/volumes/usage",
    method: "get",
    params
  })
}

export function allocateEip(data: Aws.AwsAllocateEipReq) {
  return request<{ code: number, data: { allocation_id: string }, message: string }>({
    url: "aws/ec2/eip/allocate",
    method: "post",
    data
  })
}

export function describeInstance(params: Aws.AwsDescribeInstanceReq) {
  return request<{ code: number, data: Aws.AwsInstanceBrief, message: string }>({
    url: "aws/ec2/instance/describe",
    method: "get",
    params
  })
}

// Billing
export function getBillingCostUsage(params: Aws.AwsBillingQueryReq) {
  return request<{ code: number, data: Aws.AwsBillingCostUsageResp, message: string }>({
    url: "aws/billing/cost-usage",
    method: "get",
    params
  })
}

// CloudWatch Monitoring
export function getCloudWatchMetrics(params: Aws.AwsCloudWatchMetricsReq) {
  return request<{ code: number, data: Aws.AwsCloudWatchMetricsResp, message: string }>({
    url: "aws/cloudwatch/metrics",
    method: "get",
    params,
    timeout: 30000
  })
}
