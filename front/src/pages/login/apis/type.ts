export interface LoginRequestData {
  /** admin 或 editor */
  username: string
  /** 密码 */
  password: string
  /** 验证码 */
  code: string
  /** 2FA验证码（可选） */
  two_fa_code?: string
}

export type CaptchaResponseData = ApiResponseData<string>

export type LoginResponseData = ApiResponseData<{
  token: string
  two_factor_enabled: boolean
}>

export interface ChallengeResponseData {
  nonce: string
  pub_pem: string
  expires: number
}

export interface EncryptedLoginRequestData {
  cipher: string
}
