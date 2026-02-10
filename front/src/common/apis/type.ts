/** API响应数据通用类型 */
export interface ApiResponseData<T> {
  code: number
  data: T
  message: string
}
