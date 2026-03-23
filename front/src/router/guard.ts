import type { Router } from "vue-router"
import { ping } from "@/common/apis/admin"
import { useControlModeStore } from "@/pinia/stores/control-mode"
import { usePermissionStore } from "@/pinia/stores/permission"
import { useUserStore } from "@/pinia/stores/user"
import { routerConfig } from "@/router/config"
import { isWhiteList } from "@/router/whitelist"
import { setRouteChange } from "@@/composables/useRouteListener"
import { useTitle } from "@@/composables/useTitle"
import { getToken } from "@@/utils/cache/cookies"
import { ElMessage } from "element-plus"
import NProgress from "nprogress"

NProgress.configure({ showSpinner: false })

const { setTitle } = useTitle()

const LOGIN_PATH = "/login"
const IP_CHECKED_KEY = "ip_whitelist_checked"

export function registerNavigationGuard(router: Router) {
  // 全局前置守卫
  router.beforeEach(async (to, _from) => {
    NProgress.start()
    const userStore = useUserStore()
    const permissionStore = usePermissionStore()

    // 调用 ping 检查 IP 是否允许访问（只检查一次）
    if (!localStorage.getItem(IP_CHECKED_KEY)) {
      try {
        const res = await ping()
        if (res.code !== 200) {
          ElMessage.error(res.message || "网络错误，无法访问")
          NProgress.done()
          return false
        }
        // 记录已经检查过IP白名单
        localStorage.setItem(IP_CHECKED_KEY, "true")
      } catch {
        ElMessage.error("网络错误，无法访问")
        NProgress.done()
        return false
      }
    }

    // 如果没有登录
    if (!getToken()) {
      // 如果在免登录的白名单中，则直接进入
      if (isWhiteList(to)) return true
      // 其他没有访问权限的页面将被重定向到登录页面
      return LOGIN_PATH
    }
    // 如果已经登录，并准备进入 Login 页面，则重定向到主页
    if (to.path === LOGIN_PATH) return "/"
    // 如果访问2FA设置页面，直接放行（已登录用户可访问）
    if (to.path === "/2fa-setup") return true
    // 如果用户已经获得其权限角色，仍需检查当前路由是否需要特定角色
    if (userStore.roles.length !== 0) {
      const routeRoles = to.meta?.roles as string[] | undefined
      if (!routeRoles || routeRoles.length === 0) return true
      const has = userStore.roles.some(r => routeRoles.includes(r))
      if (has) return true
      return "/403"
    }
    // 否则要重新获取权限角色
    try {
      await userStore.getInfo()
      // 获取控制模式（单机/多机），用于前端菜单适配
      const controlModeStore = useControlModeStore()
      if (!controlModeStore.loaded) {
        await controlModeStore.fetchMode()
      }
      // 注意：角色必须是一个数组！ 例如: ["admin"] 或 ["developer", "editor"]
      const roles = userStore.roles
      // 生成可访问的 Routes
      routerConfig.dynamic ? permissionStore.setRoutes(roles) : permissionStore.setAllRoutes()
      // 将 "有访问权限的动态路由" 添加到 Router 中
      permissionStore.addRoutes.forEach(route => router.addRoute(route))
      // 单机模式：将默认页重定向到系统概览
      if (controlModeStore.isLocal && (to.path === "/" || to.path === "/dashboard")) {
        return { path: "/overview", replace: true }
      }
      // 设置 replace: true, 因此导航将不会留下历史记录
      return { ...to, replace: true }
    } catch (error) {
      // 过程中发生任何错误，都直接重置 Token，并重定向到登录页面
      userStore.resetToken()
      ElMessage.error((error as Error).message || "路由守卫发生错误")
      return LOGIN_PATH
    }
  })

  // 全局后置钩子
  router.afterEach((to) => {
    setRouteChange(to)
    setTitle(to.meta.title)
    NProgress.done()
  })
}
