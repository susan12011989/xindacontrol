import type * as Utils from "./type"
import { request } from "@/http/axios"

/** 端口转企业号 */
export function port2Enterprise(data: Utils.Port2EnterpriseReq) {
  return request<Utils.Port2EnterpriseResponseData>({
    url: "utils/port2enterprise",
    method: "post",
    data
  })
}

/** 企业号转端口 */
export function enterprise2Port(data: Utils.Enterprise2PortReq) {
  return request<Utils.Enterprise2PortResponseData>({
    url: "utils/enterprise2port",
    method: "post",
    data
  })
}

/** 校验端口是否可用（同时检查 port 与 port+1） */
export function checkPortAvailable(params: Utils.CheckPortReq) {
  return request<Utils.CheckPortResponseData>({
    url: "utils/check-port",
    method: "get",
    params
  })
}

/** 将IP列表嵌入到文件 */
export function embedIPs(file: File, ips: string[]) {
  const formData = new FormData()
  formData.append("file", file)
  // 后端从 FormData 中获取多个 ips 字段
  ips.forEach((ip) => {
    formData.append("ips", ip)
  })

  return request({
    url: "utils/embedips",
    method: "post",
    data: formData,
    responseType: "blob" // 返回文件
  })
}

/** 从文件提取IP列表 */
export function extractIPs(file: File) {
  const formData = new FormData()
  formData.append("file", file)

  return request<Utils.ExtractIPsResponseData>({
    url: "utils/extractips",
    method: "post",
    data: formData
  })
}

/** 批量将IP列表嵌入到zip文件中的所有文件 */
export function embedIPsBatch(file: File, ips: string[]) {
  const formData = new FormData()
  formData.append("file", file)
  ips.forEach((ip) => {
    formData.append("ips", ip)
  })

  return request({
    url: "utils/embedips-batch",
    method: "post",
    data: formData,
    responseType: "blob" // 返回zip文件
  })
}

/** 将URL列表嵌入到文件 */
export function embedURLs(file: File, urls: string[]) {
  const formData = new FormData()
  formData.append("file", file)
  urls.forEach(u => formData.append("urls", u))

  return request({
    url: "utils/embedurls",
    method: "post",
    data: formData,
    responseType: "blob"
  })
}

/** 从文件提取URL列表 */
export function extractURLs(file: File) {
  const formData = new FormData()
  formData.append("file", file)

  return request<Utils.ExtractURLsResponseData>({
    url: "utils/extracturls",
    method: "post",
    data: formData
  })
}

/** 批量将URL列表嵌入到zip文件中的所有文件 */
export function embedURLsBatch(file: File, urls: string[]) {
  const formData = new FormData()
  formData.append("file", file)
  urls.forEach((u) => {
    formData.append("urls", u)
  })

  return request({
    url: "utils/embedurls-batch",
    method: "post",
    data: formData,
    responseType: "blob" // 返回zip文件
  })
}

/** 生成版本配置文件 */
export function generateVersion(data: Utils.GenerateVersionReq) {
  return request({
    url: "utils/generateversion",
    method: "post",
    data,
    responseType: "blob" // 返回文件
  })
}

/** 解密版本配置文件 */
export function decryptVersion(file: File) {
  const formData = new FormData()
  formData.append("file", file)

  return request({
    url: "utils/decryptversion",
    method: "post",
    data: formData,
    responseType: "blob" // 返回文件
  })
}
