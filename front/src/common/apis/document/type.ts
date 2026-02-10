import type { ApiResponseData } from "../type"

export interface QueryDocumentFilesReq {
  type?: string
  page?: number
  page_size?: number
  mime_like?: string
  ext?: string
  name_like?: string
  min_size?: number
  max_size?: number
  from?: number
  to?: number
  order?: "asc" | "desc"
}

export interface DocumentFileItem {
  id: string
  type: string
  dc_id: number
  name: string
  ext: string
  mime_type: string
  size: number
  bucket: string
  object_path: string
  url: string
  thumbnail_url: string
  kind: string
  playable: boolean
  previewable: boolean
  created_at: number
  attributes: Record<string, any>
}

export interface QueryDocumentFilesResp {
  page: number
  page_size: number
  total: number
  items: DocumentFileItem[]
}

export type QueryDocumentFilesResponseData = ApiResponseData<QueryDocumentFilesResp>
