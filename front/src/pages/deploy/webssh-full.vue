<script lang="ts" setup>
import { getToken } from "@@/utils/cache/cookies"
import { Terminal } from "xterm"
import { FitAddon } from "xterm-addon-fit"
import { WebLinksAddon } from "xterm-addon-web-links"
import "xterm/css/xterm.css"

defineOptions({
  name: "DeployWebSSHFull"
})

// 路由参数
const route = useRoute()

// 终端和连接
let terminal: Terminal | null = null
let fitAddon: FitAddon | null = null
let socket: WebSocket | null = null
const terminalRef = ref<HTMLElement>()
const connected = ref(false)
const connecting = ref(false)

// 心跳和重连
let heartbeatTimer: number | null = null
let reconnectTimer: number | null = null
let reconnectAttempts = 0
const maxReconnectAttempts = 999 // 几乎无限重连
const reconnectInterval = 3000 // 3秒重连间隔
const heartbeatInterval = 20000 // 20秒心跳间隔（比后端60秒超时要短）
let lastPongTime = 0 // 最后收到 pong 的时间

// 页面可见性
const isPageVisible = ref(true)

// 初始化终端
function initTerminal() {
  if (!terminalRef.value) return

  terminal = new Terminal({
    cursorBlink: true,
    cursorStyle: "block",
    fontSize: 14,
    fontFamily: "Consolas, Monaco, monospace",
    theme: {
      background: "#1e1e1e",
      foreground: "#d4d4d4",
      cursor: "#ffffff"
    }
  })

  fitAddon = new FitAddon()
  const webLinksAddon = new WebLinksAddon()

  terminal.loadAddon(fitAddon)
  terminal.loadAddon(webLinksAddon)
  terminal.open(terminalRef.value)
  fitAddon.fit()

  terminal.onData((data) => {
    if (socket && socket.readyState === WebSocket.OPEN) {
      const normalized = data
        .replace(/\x7F/g, "\b")
        .replace(/\r?\n/g, "\r")
      socket.send(
        JSON.stringify({
          type: "input",
          data: normalized
        })
      )
    }
  })

  window.addEventListener("resize", handleResize)

  // 监听页面可见性变化
  document.addEventListener("visibilitychange", handleVisibilityChange)
}

// 启动心跳
function startHeartbeat() {
  stopHeartbeat()
  heartbeatTimer = window.setInterval(() => {
    if (socket && socket.readyState === WebSocket.OPEN) {
      // 检查上次心跳响应时间，如果超过2个心跳周期没收到响应，认为连接有问题
      const now = Date.now()
      const timeSinceLastPong = now - lastPongTime
      if (timeSinceLastPong > heartbeatInterval * 2.5) {
        console.warn(`心跳超时: ${timeSinceLastPong}ms 未收到响应，断开重连`)
        if (terminal) {
          terminal.writeln("\r\n\x1B[1;31m=== 心跳超时，连接异常，正在重连... ===\x1B[0m\r\n")
        }
        // 主动关闭并重连
        connected.value = false
        stopHeartbeat()
        if (socket) {
          socket.close()
          socket = null
        }
        if (isPageVisible.value) {
          reconnectAttempts = 0
          connectSSH()
        }
        return
      }

      // 即使页面不可见也发送心跳，保持连接
      // 这样可以避免后端超时断开连接
      try {
        socket.send(JSON.stringify({ type: "ping" }))
      } catch (err) {
        console.error("心跳发送失败:", err)
        // 心跳发送失败，连接可能已断开
        connected.value = false
        stopHeartbeat()
      }
    }
  }, heartbeatInterval)
}

// 停止心跳
function stopHeartbeat() {
  if (heartbeatTimer) {
    clearInterval(heartbeatTimer)
    heartbeatTimer = null
  }
}

// 页面可见性变化处理
function handleVisibilityChange() {
  isPageVisible.value = !document.hidden

  if (isPageVisible.value) {
    // 页面重新可见时，检查连接状态
    checkConnectionAndReconnect()
  } else {
    // 页面不可见时，取消待处理的重连定时器
    if (reconnectTimer) {
      clearTimeout(reconnectTimer)
      reconnectTimer = null
    }
  }
}

// 检查连接状态并在需要时重连
function checkConnectionAndReconnect() {
  // 检查 WebSocket 实际连接状态
  if (socket && socket.readyState !== WebSocket.OPEN) {
    // WebSocket 已断开但状态未更新，强制重置
    console.log("检测到连接已断开，准备重连...")
    connected.value = false
    connecting.value = false
    socket = null
    stopHeartbeat()
  }

  // 如果未连接，则尝试重连
  if (!connected.value && !connecting.value) {
    if (terminal) {
      terminal.writeln("\r\n\x1B[1;33m=== 检测到页面重新激活，正在重新连接... ===\x1B[0m\r\n")
    }
    // 重置重连计数，允许立即重连
    reconnectAttempts = 0
    connectSSH()
  } else if (connected.value && socket && socket.readyState === WebSocket.OPEN) {
    // 连接正常，发送一个心跳测试
    try {
      socket.send(JSON.stringify({ type: "ping" }))
    } catch (err) {
      // 发送失败，说明连接有问题
      console.error("心跳发送失败:", err)
      connected.value = false
      checkConnectionAndReconnect()
    }
  }
}

// 尝试重连
function tryReconnect() {
  if (reconnectAttempts >= maxReconnectAttempts) {
    ElMessage.error("重连次数过多，请刷新页面")
    return
  }

  if (reconnectTimer) {
    clearTimeout(reconnectTimer)
  }

  reconnectAttempts++
  const delay = Math.min(reconnectInterval * reconnectAttempts, 30000) // 最多30秒

  if (terminal) {
    terminal.writeln(`\r\n\x1B[1;33m=== 连接断开，${delay / 1000}秒后尝试第${reconnectAttempts}次重连... ===\x1B[0m\r\n`)
  }

  reconnectTimer = window.setTimeout(() => {
    if (!connected.value && isPageVisible.value) {
      connectSSH()
    }
  }, delay)
}

// 连接 SSH（根据 query server_id 自动连接）
function connectSSH() {
  const serverId = Number(route.query.server_id || 0)
  if (!serverId) {
    ElMessage.error("缺少 server_id 参数")
    return
  }

  if (connecting.value || connected.value) return

  // 确保旧连接已经关闭（防止重连时的资源泄漏）
  if (socket) {
    try {
      socket.close()
    } catch {
      // 忽略关闭错误
    }
    socket = null
  }

  connecting.value = true

  const token = getToken()
  if (!token) {
    ElMessage.error("请先登录")
    connecting.value = false
    return
  }

  const baseURL = import.meta.env.VITE_BASE_URL || "/server/v1"
  const protocol = window.location.protocol === "https:" ? "wss:" : "ws:"

  let wsHost = window.location.host
  let wsPath = baseURL
  if (baseURL.startsWith("http://") || baseURL.startsWith("https://")) {
    try {
      const url = new URL(baseURL)
      wsHost = url.host
      wsPath = url.pathname
    } catch {}
  }

  const wsUrl = `${protocol}//${wsHost}${wsPath}/deploy/webssh?server_id=${serverId}&token=${encodeURIComponent(token)}`

  try {
    socket = new WebSocket(wsUrl)
    socket.onopen = () => {
      connecting.value = false
      connected.value = true
      reconnectAttempts = 0 // 重置重连计数
      lastPongTime = Date.now() // 初始化心跳时间
      if (terminal) {
        terminal.clear()
        terminal.writeln("\x1B[1;32m=== 已连接到服务器（全屏） ===\x1B[0m\r\n")
      }
      if (terminal && fitAddon) {
        socket?.send(
          JSON.stringify({ type: "resize", rows: terminal.rows, cols: terminal.cols })
        )
      }
      // 启动心跳
      startHeartbeat()
    }

    socket.onmessage = (event) => {
      try {
        const msg = JSON.parse(event.data)
        if (msg.type === "output" && terminal) {
          terminal.write(msg.data)
        } else if (msg.type === "error" && terminal) {
          terminal.writeln(`\r\n\x1B[1;31m错误: ${msg.data}\x1B[0m\r\n`)
        } else if (msg.type === "pong") {
          // 收到心跳响应，连接正常
          lastPongTime = Date.now()
        }
      } catch {
        if (terminal) terminal.write(event.data)
      }
    }

    socket.onerror = () => {
      connecting.value = false
      connected.value = false
      stopHeartbeat()
      if (terminal && reconnectAttempts === 0) {
        terminal.writeln("\r\n\x1B[1;31m=== 连接错误 ===\x1B[0m\r\n")
      }
    }

    socket.onclose = () => {
      const wasConnected = connected.value
      connecting.value = false
      connected.value = false
      stopHeartbeat()

      if (terminal && reconnectAttempts === 0) {
        terminal.writeln("\r\n\x1B[1;33m=== 连接已断开 ===\x1B[0m\r\n")
      }

      // 自动重连
      if (wasConnected) {
        if (isPageVisible.value) {
          // 页面可见时立即重连
          tryReconnect()
        } else {
          // 页面不可见时，等待页面可见时再重连（通过 handleVisibilityChange）
          if (terminal) {
            terminal.writeln("\r\n\x1B[1;33m=== 页面在后台，将在重新激活时自动连接 ===\x1B[0m\r\n")
          }
        }
      }
    }
  } catch {
    connecting.value = false
    connected.value = false
    ElMessage.error("创建WebSocket连接失败")
  }
}

// 窗口大小调整
function handleResize() {
  if (fitAddon && terminal) {
    fitAddon.fit()
    if (socket && socket.readyState === WebSocket.OPEN) {
      socket.send(
        JSON.stringify({ type: "resize", rows: terminal.rows, cols: terminal.cols })
      )
    }
  }
}

// 断开连接
function disconnect() {
  stopHeartbeat()
  if (reconnectTimer) {
    clearTimeout(reconnectTimer)
    reconnectTimer = null
  }
  if (socket) {
    socket.close()
    socket = null
  }
  connected.value = false
  connecting.value = false
  if (terminal) terminal.clear()
}

onMounted(() => {
  initTerminal()
  connectSSH()
})

onBeforeUnmount(() => {
  disconnect()
  window.removeEventListener("resize", handleResize)
  document.removeEventListener("visibilitychange", handleVisibilityChange)
  if (terminal) terminal.dispose()
})
</script>

<template>
  <div class="fullscreen-app">
    <div ref="terminalRef" class="terminal-container"></div>
  </div>
</template>

<style lang="scss" scoped>
.fullscreen-app {
  padding: 0;
  margin: 0;
  height: 100vh;
  width: 100vw;
  background: #1e1e1e;

  .terminal-container {
    height: 100vh;
    width: 100vw;
    background: #1e1e1e;
  }
}
</style>
