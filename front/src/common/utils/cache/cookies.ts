// 统一处理 Cookie

import { CacheKey } from "@@/constants/cache-key"
import Cookies from "js-cookie"

export function getToken() {
  return Cookies.get(CacheKey.TOKEN)
}

export function setToken(token: string, days = 1) {
  Cookies.set(CacheKey.TOKEN, token, {
    expires: days,
    sameSite: "Lax"
    // secure: true // 部署在 https 时建议开启
  })
}

export function removeToken() {
  Cookies.remove(CacheKey.TOKEN)
}
