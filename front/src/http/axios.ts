import type { AxiosInstance, AxiosRequestConfig } from "axios"
import { useUserStore } from "@/pinia/stores/user"
import { getToken } from "@@/utils/cache/cookies"
import axios from "axios"
import { get, merge } from "lodash-es"

/** 退出登录并强制刷新页面（会重定向到登录页） */
function logout() {
  useUserStore().logout()
  location.reload()
}

// 避免重复弹窗
let isAuthDialogShown = false
function showAuthDialog(message?: string) {
  if (isAuthDialogShown) return
  isAuthDialogShown = true
  const text = message || "登录状态已失效，请重新登录"
  // 单按钮确认弹窗，用户确认后再退出登录
  ElMessageBox.alert(text, "登录过期", {
    confirmButtonText: "确定",
    type: "warning",
    showClose: false
  })
    .then(() => {
      isAuthDialogShown = false
      logout()
    })
    .catch(() => {
      isAuthDialogShown = false
      logout()
    })
}

/** 创建请求实例 */
function createInstance() {
  // 创建一个 axios 实例命名为 instance
  const instance = axios.create()
  // 请求拦截器
  instance.interceptors.request.use(
    // 发送之前
    (config) => {
      // 如果是 FormData，删除 Content-Type 让浏览器自动设置
      if (config.data instanceof FormData) {
        delete config.headers["Content-Type"]
      }
      return config
    },
    // 发送失败
    error => Promise.reject(error)
  )
  // 响应拦截器（可根据具体业务作出相应的调整）
  instance.interceptors.response.use(
    (response) => {
      // apiData 是 api 返回的数据
      const apiData = response.data
      // 二进制数据则直接返回
      const responseType = response.request?.responseType
      if (responseType === "blob" || responseType === "arraybuffer") return apiData
      // 这个 code 是和后端约定的业务 code
      const code = apiData.code
      // 如果没有 code, 代表这不是项目后端开发的 api
      if (code === undefined) {
        ElMessage.error("非本系统的接口")
        return Promise.reject(new Error("非本系统的接口"))
      }
      switch (code) {
        case 0:
          return apiData
        case 200:
          return apiData
        case 401:
          // 统一弹窗展示原因，确认后再退出
          showAuthDialog(apiData.reason || apiData.message)
          return Promise.reject(new Error("Unauthorized"))
        default:
          // 不是正确的 code
          console.error("[API Error]", response.config?.method?.toUpperCase(), response.config?.url, "→ code:", code, "message:", apiData.message, "data:", apiData)
          ElMessage.error(apiData.message || "Error")
          return Promise.reject(new Error(apiData.message || "Error"))
      }
    },
    (error) => {
      // status 是 HTTP 状态码
      const status = get(error, "response.status")
      const message = get(error, "response.data.message")
      const reason = get(error, "response.data.reason")
      switch (status) {
        case 400:
          error.message = "请求错误"
          break
        case 401:
          // Token 失效：展示后端返回原因（reason优先）
          error.message = reason || message || "未授权"
          showAuthDialog(error.message)
          break
        case 403:
          error.message = message || "拒绝访问"
          break
        case 404:
          error.message = "请求地址出错"
          break
        case 408:
          error.message = "请求超时"
          break
        case 500:
          error.message = "服务器内部错误"
          break
        case 501:
          error.message = "服务未实现"
          break
        case 502:
          error.message = "网关错误"
          break
        case 503:
          error.message = "服务不可用"
          break
        case 504:
          error.message = "网关超时"
          break
        case 505:
          error.message = "HTTP 版本不受支持"
          break
      }
      ElMessage.error(error.message)
      return Promise.reject(error)
    }
  )
  return instance
}

/** 创建请求方法 */
function createRequest(instance: AxiosInstance) {
  return <T>(config: AxiosRequestConfig): Promise<T> => {
    const token = getToken()
    // 默认配置
    const defaultConfig: AxiosRequestConfig = {
      // 接口地址
      baseURL: import.meta.env.VITE_BASE_URL,
      // 请求头
      headers: {
        // 携带 Token
        "Authorization": token ? `Bearer ${token}` : undefined,
        "Content-Type": "application/json"
      },
      // 请求体
      data: {},
      // 请求超时 60秒
      timeout: 60000,
      // 跨域请求时是否携带 Cookies
      withCredentials: false
    }
    // 将默认配置 defaultConfig 和传入的自定义配置 config 进行合并成为 mergeConfig
    const mergeConfig = merge(defaultConfig, config)
    return instance(mergeConfig)
  }
}

/** 用于请求的实例 */
const instance = createInstance()

/** 用于请求的方法 */
export const request = createRequest(instance)

/**
 * 创建流式请求方法
 * 使用fetch API处理流式响应，适用于浏览器环境
 */
export function createStreamRequest(config: AxiosRequestConfig, onData: (data: any, isComplete?: boolean) => void, onError?: (error: any) => void) {
  const token = getToken()
  // 添加标记，避免重复触发完成事件
  let hasCompletedSignal = false

  // 构建URL（安全拼接 baseURL 与相对路径）
  let url = config.url || ""
  let base = (config.baseURL || import.meta.env.VITE_BASE_URL) as string | undefined

  // 如果 base 为空或未定义，使用当前页面的 origin（适用于生产环境）
  if (!base || base.trim() === "") {
    base = window.location.origin
  }

  // 如果 base 是相对路径（如 /server/v1），拼接 origin 使其成为完整 URL
  if (base && !base.startsWith("http")) {
    base = window.location.origin + base
  }

  // 确保 base 是有效的 URL
  if (base) {
    const baseURL = base.endsWith("/") ? base : `${base}/`
    const path = url.startsWith("/") ? url.slice(1) : url
    url = new URL(path, baseURL).toString()
  }

  // 添加查询参数
  if (config.params) {
    const queryParams = new URLSearchParams()
    for (const key in config.params) {
      queryParams.append(key, String(config.params[key]))
    }
    url += `?${queryParams.toString()}`
  }

  // 构建fetch请求选项
  const controller = new AbortController()
  const headers: Record<string, string> = {
    "Content-Type": "application/json"
  }

  // 添加认证头
  if (token) {
    headers.Authorization = `Bearer ${token}`
  }

  // 合并自定义头
  if (config.headers) {
    Object.entries(config.headers).forEach(([key, value]) => {
      if (value !== undefined && value !== null) {
        headers[key] = String(value)
      }
    })
  }

  const fetchOptions: RequestInit = {
    method: config.method?.toUpperCase() || "GET",
    headers,
    signal: controller.signal,
    credentials: config.withCredentials ? "include" : "same-origin"
  }

  // 如果有请求体
  if (config.data) {
    fetchOptions.body = JSON.stringify(config.data)
  }

  console.log("创建流式请求:", url, fetchOptions)

  // 包装onData回调，避免重复触发完成事件
  const safeOnData = (data: any, isComplete?: boolean) => {
    if (isComplete === true) {
      if (!hasCompletedSignal) {
        console.log("触发完成信号:", data)
        hasCompletedSignal = true
        onData(data, true)
      } else {
        console.log("已经触发过完成信号，忽略:", data)
      }
    } else {
      onData(data, false)
    }
  }

  // 使用fetch API发送请求
  fetch(url, fetchOptions)
    .then((response) => {
      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`)
      }

      // 获取响应体的reader
      const reader = response.body?.getReader()
      if (!reader) {
        throw new Error("Stream reading not supported")
      }

      // 处理流数据
      const processStream = async () => {
        let buffer = "" // 用于存储未完整的JSON
        // let isEventEnd = false; // 标记是否收到了end事件

        try {
          while (true) {
            const { done, value } = await reader.read()

            if (done) {
              // 处理最后可能残留的数据
              if (buffer.trim()) {
                try {
                  // 检查是否是SSE格式
                  if (buffer.includes("event:") || buffer.includes("data:")) {
                    // 处理SSE格式
                    handleSSEData(buffer)
                  } else {
                    // 尝试解析JSON
                    const parsedData = JSON.parse(buffer)
                    // 检查是否是结束信号
                    if (parsedData.success === true) {
                      safeOnData(parsedData, true)
                    } else {
                      safeOnData(parsedData)
                    }
                  }
                } catch {
                  console.log("无法解析数据:", buffer)
                  safeOnData(buffer)
                }
              }
              break
            }

            // 将Uint8Array转换为字符串
            const chunk = new TextDecoder().decode(value)
            buffer += chunk

            // 检查是否包含SSE格式的数据
            if (buffer.includes("event:") || buffer.includes("data:")) {
              // 处理SSE格式的数据
              const result = handleSSEData(buffer)
              buffer = result.remainingBuffer

              // 如果是end事件，可以提前结束
              if (result.isEnd) {
                break
              }
            } else {
              // 常规处理：按行分割并解析
              const lines = buffer.split("\n")

              // 保留最后一行，它可能是不完整的
              buffer = lines.pop() || ""

              // 处理完整的行
              for (const line of lines) {
                if (line.trim()) {
                  try {
                    const parsedData = JSON.parse(line)
                    // 检查是否是结束信号
                    if (parsedData.success === true) {
                      safeOnData(parsedData, true)
                    } else {
                      safeOnData(parsedData)
                    }
                  } catch {
                    console.log("无法解析行:", line)
                    safeOnData(line)
                  }
                }
              }
            }
          }
        } catch (error) {
          console.error("处理流数据时出错:", error)
          if (onError) onError(error)
        }
      }

      // 处理SSE格式的数据
      const handleSSEData = (sseData: string) => {
        console.log("处理SSE格式数据:", sseData)
        let remainingBuffer = ""
        let isEnd = false

        // 按行分割
        const lines = sseData.split("\n")
        let currentEvent = ""
        let currentData = ""

        for (let i = 0; i < lines.length; i++) {
          const line = lines[i].trim()

          if (line.startsWith("event:")) {
            currentEvent = line.substring(6).trim()
            console.log("检测到事件:", currentEvent)
            if (currentEvent === "end") {
              isEnd = true
            }
          } else if (line.startsWith("data:")) {
            currentData = line.substring(5).trim()
            console.log("检测到数据:", currentData)

            // 尝试解析数据为JSON
            try {
              const parsedData = JSON.parse(currentData)
              console.log("解析的JSON数据:", parsedData)

              // 如果是end事件或success为true，标记为完成
              if (isEnd || parsedData.success === true) {
                console.log("检测到完成信号:", isEnd ? "(event:end)" : "(success:true)", parsedData)
                safeOnData(parsedData, true)
              } else {
                safeOnData(parsedData)
              }
            } catch {
              console.log("SSE数据不是有效JSON:", currentData)
              // 即使不是JSON，如果是end事件，也标记为完成
              if (isEnd) {
                console.log("非JSON的end事件:", currentData)
                safeOnData(currentData, true)
              } else {
                safeOnData(currentData)
              }
            }

            // 不要在这里重置事件，因为需要将event:end与后续的data:关联起来
            if (!isEnd) {
              currentEvent = ""
              currentData = ""
            }
          } else if (line === "") {
            // 事件块的结束
            if (currentEvent && currentData) {
              try {
                const parsedData = JSON.parse(currentData)
                if (currentEvent === "end" || parsedData.success === true) {
                  console.log("事件块结束，检测到完成信号:", parsedData)
                  safeOnData(parsedData, true)
                  isEnd = true
                } else {
                  safeOnData(parsedData)
                }
              } catch {
                if (currentEvent === "end") {
                  console.log("事件块结束，非JSON的end事件:", currentData)
                  safeOnData(currentData, true)
                  isEnd = true
                } else {
                  safeOnData(currentData)
                }
              }
              currentEvent = ""
              currentData = ""
            }
          } else {
            // 不是SSE格式的行，添加到剩余缓冲区
            if (i === lines.length - 1) {
              remainingBuffer = line
            }
          }
        }

        // 如果数据中包含了event:end和data:，但没有空行分隔，手动处理
        if (isEnd && currentData && !currentEvent) {
          try {
            const parsedData = JSON.parse(currentData)
            console.log("手动处理End事件数据:", parsedData)
            safeOnData(parsedData, true)
          } catch {
            console.log("手动处理非JSON的End事件数据:", currentData)
            safeOnData(currentData, true)
          }
        }

        return { remainingBuffer, isEnd }
      }

      // 开始处理流
      processStream()
        .catch((error) => {
          console.error("处理流时出错:", error)
          if (onError) onError(error)
        })
    })
    .catch((error) => {
      console.error("请求错误:", error)
      if (onError) onError(error)
    })

  // 返回取消函数
  return () => {
    controller.abort()
  }
}
