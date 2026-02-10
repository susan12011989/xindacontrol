<script lang="ts" setup>
import { getServerList } from "@@/apis/deploy"
import { getToken } from "@@/utils/cache/cookies"
import { Terminal } from "xterm"
import { FitAddon } from "xterm-addon-fit"
import { WebLinksAddon } from "xterm-addon-web-links"
import "xterm/css/xterm.css"

defineOptions({
  name: "DeployWebSSH"
})

// 服务器选择
const serverType = ref<number>(2) // 0-全部, 1-商户服务器, 2-系统服务器
const allServerList = ref<any[]>([])
const serverList = ref<any[]>([])
const currentServerId = ref(0)
const currentServerInfo = ref<any>({})

// 终端和连接
let terminal: Terminal | null = null
let fitAddon: FitAddon | null = null
let socket: WebSocket | null = null
const terminalRef = ref<HTMLElement>()
const connected = ref(false)
const connecting = ref(false)
// 路由实例（避免在事件中调用 useRouter 导致实例丢失）
const router = useRouter()

// 打开全屏标签页
function openFullscreenTab() {
  if (!currentServerId.value) {
    ElMessage.warning("请先选择服务器")
    return
  }
  const { href } = router.resolve({ name: "DeployWebSSHFull", query: { server_id: currentServerId.value } })
  window.open(href, "_blank")
}

// 加载服务器列表
async function loadServers() {
  try {
    const res = await getServerList({ page: 1, size: 5000 })
    allServerList.value = res.data.list
    filterServers()
  } catch {
    ElMessage.error("加载服务器列表失败")
  }
}

// 根据服务器类型过滤服务器列表
function filterServers() {
  if (serverType.value === 0) {
    serverList.value = allServerList.value
  } else {
    serverList.value = allServerList.value.filter(s => s.server_type === serverType.value)
  }

  // 如果当前选中的服务器不在过滤后的列表中，清空选择
  if (currentServerId.value && !serverList.value.find(s => s.id === currentServerId.value)) {
    currentServerId.value = 0
    currentServerInfo.value = {}
  }
}

// 监听服务器类型变化
watch(serverType, () => {
  filterServers()
})

// 初始化终端
function initTerminal() {
  if (!terminalRef.value) return

  // 创建终端实例
  terminal = new Terminal({
    cursorBlink: true,
    cursorStyle: "block",
    fontSize: 14,
    fontFamily: "Consolas, Monaco, monospace",
    theme: {
      background: "#1e1e1e",
      foreground: "#d4d4d4",
      cursor: "#ffffff"
    },
    rows: 30,
    cols: 100
  })

  // 创建插件
  fitAddon = new FitAddon()
  const webLinksAddon = new WebLinksAddon()

  // 加载插件
  terminal.loadAddon(fitAddon)
  terminal.loadAddon(webLinksAddon)

  // 挂载到DOM
  terminal.open(terminalRef.value)

  // 自适应大小
  fitAddon.fit()

  // 监听用户输入
  terminal.onData((data) => {
    if (socket && socket.readyState === WebSocket.OPEN) {
      // 规范化输入：
      // - 将 DEL(0x7f) 转换为 Backspace(^H, 0x08)，解决 Windows 本地退格无效
      // - 将换行统一为 CR(\r)，兼容 Windows Shell
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

  // 监听窗口大小变化
  window.addEventListener("resize", handleResize)
}

// 连接SSH
function connectSSH() {
  if (!currentServerId.value) {
    ElMessage.warning("请先选择服务器")
    return
  }

  if (connecting.value || connected.value) {
    return
  }

  connecting.value = true

  // 获取 token
  const token = getToken()
  if (!token) {
    ElMessage.error("请先登录")
    connecting.value = false
    return
  }

  // 构建WebSocket URL
  // 获取后端地址（与axios的baseURL保持一致）
  const baseURL = import.meta.env.VITE_BASE_URL || "/server/v1"
  const protocol = window.location.protocol === "https:" ? "wss:" : "ws:"

  // 解析baseURL，提取host和path
  let wsHost = window.location.host
  let wsPath = baseURL

  // 如果baseURL是完整URL（如 http://localhost:58181/server/v1）
  if (baseURL.startsWith("http://") || baseURL.startsWith("https://")) {
    try {
      const url = new URL(baseURL)
      wsHost = url.host
      wsPath = url.pathname
    } catch {
      // 解析失败，使用默认值
    }
  }

  const wsUrl = `${protocol}//${wsHost}${wsPath}/deploy/webssh?server_id=${currentServerId.value}&token=${encodeURIComponent(token)}`

  console.log("WebSSH: 连接配置", { baseURL, wsHost, wsPath, wsUrl: wsUrl.replace(token, "***") })

  try {
    socket = new WebSocket(wsUrl)

    socket.onopen = () => {
      connecting.value = false
      connected.value = true
      ElMessage.success("SSH连接成功")

      if (terminal) {
        terminal.clear()
        terminal.writeln("\x1B[1;32m=== 已连接到服务器 ===\x1B[0m")
        terminal.writeln("")
      }

      // 发送终端尺寸
      if (terminal && fitAddon) {
        socket?.send(
          JSON.stringify({
            type: "resize",
            rows: terminal.rows,
            cols: terminal.cols
          })
        )
      }
    }

    socket.onmessage = (event) => {
      try {
        const msg = JSON.parse(event.data)
        console.log("WebSSH: 收到消息", msg)
        if (msg.type === "output" && terminal) {
          terminal.write(msg.data)
        } else if (msg.type === "error" && terminal) {
          terminal.writeln(`\r\n\x1B[1;31m错误: ${msg.data}\x1B[0m\r\n`)
          console.error("WebSSH错误:", msg.data)
        }
      } catch {
        // 如果不是JSON，直接输出
        if (terminal) {
          terminal.write(event.data)
        }
      }
    }

    socket.onerror = (error) => {
      console.error("WebSSH: WebSocket错误", error)
      connecting.value = false
      connected.value = false
      ElMessage.error("WebSocket连接错误")
      if (terminal) {
        terminal.writeln("\r\n\x1B[1;31m=== 连接错误 ===\x1B[0m\r\n")
      }
    }

    socket.onclose = (event) => {
      console.log("WebSSH: 连接关闭", event.code, event.reason)
      connecting.value = false
      connected.value = false
      if (terminal) {
        terminal.writeln("\r\n\x1B[1;33m=== 连接已断开 ===\x1B[0m\r\n")
      }
    }
  } catch (error) {
    console.error("WebSSH: 创建连接失败", error)
    connecting.value = false
    connected.value = false
    ElMessage.error("创建WebSocket连接失败")
  }
}

// 断开连接
function disconnect() {
  if (socket) {
    socket.close()
    socket = null
  }
  connected.value = false
  if (terminal) {
    terminal.clear()
  }
}

// 清屏
function clearTerminal() {
  if (terminal) {
    terminal.clear()
  }
}

// 窗口大小调整
function handleResize() {
  if (fitAddon && terminal) {
    fitAddon.fit()
    // 通知服务器终端尺寸变化
    if (socket && socket.readyState === WebSocket.OPEN) {
      socket.send(
        JSON.stringify({
          type: "resize",
          rows: terminal.rows,
          cols: terminal.cols
        })
      )
    }
  }
}

// 切换服务器
watch(currentServerId, () => {
  if (currentServerId.value && serverList.value.length > 0) {
    const server = serverList.value.find(s => s.id === currentServerId.value)
    if (server) {
      currentServerInfo.value = server
    }
    // 断开之前的连接
    if (connected.value) {
      disconnect()
    }
  }
})

// 初始化
onMounted(() => {
  loadServers()
  initTerminal()
})

// 页面激活时（从其他页面切回来）
onActivated(() => {
  // 重新适配终端大小
  if (fitAddon && terminal) {
    nextTick(() => {
      fitAddon?.fit()
    })
  }
})

// 页面失活时（切换到其他页面）
onDeactivated(() => {
  // WebSocket 连接保持，不断开
  // 终端状态保持
})

// 清理
onBeforeUnmount(() => {
  disconnect()
  window.removeEventListener("resize", handleResize)
  if (terminal) {
    terminal.dispose()
  }
})
</script>

<template>
  <div class="app-container">
    <!-- 顶部控制栏 -->
    <el-card class="mb-4">
      <div class="flex flex-col gap-3">
        <!-- 第一行：服务器类型选择 -->
        <div class="flex items-center gap-4">
          <span class="text-base font-bold w-20">类型:</span>
          <el-radio-group v-model="serverType" :disabled="connected">
            <el-radio-button :value="2">系统服务器</el-radio-button>
            <el-radio-button :value="1">商户服务器</el-radio-button>
          </el-radio-group>
          <el-tag type="info">共 {{ serverList.length }} 台服务器</el-tag>
        </div>

        <!-- 第二行：服务器选择和连接控制 -->
        <div class="flex justify-between items-center">
          <div class="flex items-center gap-4">
            <span class="text-base font-bold w-20">选择:</span>
            <el-select
              v-model="currentServerId"
              placeholder="请选择服务器"
              style="width: 450px"
              filterable
              :disabled="connected"
            >
              <el-option
                v-for="server in serverList"
                :key="server.id"
                :label="`${server.name} (${server.host}) ${server.merchant_name ? `- ${server.merchant_name}` : ''}`"
                :value="server.id"
              >
                <div class="flex items-center justify-between">
                  <span>{{ server.name }}</span>
                  <span class="text-sm text-gray-400">{{ server.host }}</span>
                </div>
              </el-option>
            </el-select>
            <el-tag v-if="connected" type="success">
              <el-icon><Connection /></el-icon> 已连接
            </el-tag>
            <el-tag v-else type="info">
              <el-icon><Close /></el-icon> 未连接
            </el-tag>
          </div>
          <el-button-group>
            <el-button v-if="!connected" type="primary" :loading="connecting" @click="connectSSH">
              <el-icon><Link /></el-icon> 连接
            </el-button>
            <el-button v-else type="danger" @click="disconnect">
              <el-icon><Close /></el-icon> 断开
            </el-button>
            <el-button @click="clearTerminal">
              <el-icon><Delete /></el-icon> 清屏
            </el-button>
            <el-button :disabled="!currentServerId" @click="openFullscreenTab">
              <el-icon><FullScreen /></el-icon> 新标签全屏
            </el-button>
          </el-button-group>
        </div>
      </div>
    </el-card>

    <!-- 终端区域 -->
    <el-card>
      <template #header>
        <div class="flex justify-between items-center">
          <span class="text-base font-bold">SSH 终端</span>
          <span v-if="currentServerInfo.host" class="text-sm text-gray-500">
            {{ currentServerInfo.username }}@{{ currentServerInfo.host }}:{{ currentServerInfo.port }}
          </span>
        </div>
      </template>
      <div ref="terminalRef" class="terminal-container"></div>
    </el-card>
  </div>
</template>

<style lang="scss" scoped>
.app-container {
  padding: 20px;
  height: calc(100vh - 100px);
  display: flex;
  flex-direction: column;

  .terminal-container {
    height: calc(100vh - 300px);
    min-height: 400px;
    background: #1e1e1e;
    padding: 10px;
    border-radius: 4px;
  }
}
</style>
