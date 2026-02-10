import type { QueryDocumentFilesReq, QueryDocumentFilesResponseData } from "./type"
import { request } from "@/http/axios"

/** 获取文档文件列表 */
export function getDocumentFilesApi(params: QueryDocumentFilesReq) {
  return request<QueryDocumentFilesResponseData>({
    url: "/document/files",
    method: "GET",
    params
  })
}
