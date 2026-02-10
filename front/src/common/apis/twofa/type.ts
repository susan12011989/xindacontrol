// 2FA状态响应
export interface TwoFAStatusResp {
  enabled: boolean
}

// 2FA设置响应
export interface TwoFASetupResp {
  secret: string // Base32编码的密钥
  qr_code: string // 二维码URL (otpauth://...)
  enabled: boolean
}

// 启用2FA请求
export interface TwoFAEnableReq {
  code: string // 6位TOTP验证码
}

// 禁用2FA请求
export interface TwoFADisableReq {
  password: string // 管理员密码
}
