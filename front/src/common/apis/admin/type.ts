export interface UpdatePasswordRequestData {
  old_password: string // 旧密码
  new_password: string // 新密码
}
export type UpdatePasswordResponseData = ApiResponseData<string>

export type Pong = ApiResponseData<{ timestamp: number }>
