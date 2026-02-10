<script lang="ts" setup>
import type { DockerContainerStatus, ServiceName, ServiceStatusResp } from "@@/apis/deploy/type"
import { getDockerContainers, getProgramConfig, getServerDetail, getServerList, getServerStats, getServiceLogs, getServiceStatus, serviceAction, updateProgramConfig, uploadServerFile } from "@@/apis/deploy"

defineOptions({
  name: "DeployControl"
})

// 支持的服务列表
const SUPPORTED_SERVICES: ServiceName[] = ["server", "wukongim", "gost"]
// 支持上传的服务
const UPLOADABLE_SERVICES: ServiceName[] = ["server", "wukongim"]

// 从路由获取服务器ID
const route = useRoute()
const router = useRouter()
const serverId = ref(Number(route.query.server_id) || 0)
const serverInfo = ref<any>({})

// 服务列表
const serviceList = ref<ServiceStatusResp[]>([])
const loading = ref(false)
const executing = ref(false)

// 服务器整体资源
const serverStats = ref<any>(null)
const statsLoading = ref(false)

// Docker 容器状态
const dockerContainers = ref<DockerContainerStatus[]>([])
const dockerLoading = ref(false)

// 日志查看
const logDialogVisible = ref(false)
const currentService = ref<ServiceName>("server")
const logContent = ref("")
const logLoading = ref(false)
const logLines = ref(100)

// 顶部：服务器切换（远程搜索）
const serverSelectLoading = ref(false)
const serverOptions = ref<{ label: string, value: number }[]>([])
const selectedServerId = ref<number | undefined>(serverId.value || undefined)
let serverSearchTimer: any = null

async function fetchServerOptions(keyword = "") {
  serverSelectLoading.value = true
  try {
    const params: any = { page: 1, size: 20 }
    const key = keyword.trim()
    if (key) {
      if (/[0-9.:]/.test(key)) {
        params.host = key
      } else {
        params.name = key
      }
    }
    const res = await getServerList(params)
    serverOptions.value = (res.data.list || []).map((s: any) => ({
      value: s.id,
      label: `${s.name} (${s.host}:${s.port})`
    }))
  } catch {
    // ignore
  } finally {
    serverSelectLoading.value = false
  }
}

function remoteSearchServers(query: string) {
  if (serverSearchTimer) clearTimeout(serverSearchTimer)
  serverSearchTimer = setTimeout(() => fetchServerOptions(query), 200)
}

function onServerChange(id: number) {
  if (!id || id === serverId.value) return
  router.push({ name: "DeployControl", query: { server_id: String(id) } })
}

// 行内"更新"上传逻辑
const fileInputRef = ref<HTMLInputElement | null>(null)
const uploadingService = ref<string>("")
const uploadPercent = ref<number>(0)
const pendingServiceName = ref<ServiceName | "">("")

function canUpload(serviceName: string): boolean {
  return UPLOADABLE_SERVICES.includes(serviceName as ServiceName)
}

function onClickUpdate(row: ServiceStatusResp) {
  if (!serverId.value) {
    ElMessage.warning("请先选择服务器")
    return
  }
  if (!canUpload(row.service_name)) {
    ElMessage.warning("该服务不支持上传")
    return
  }
  pendingServiceName.value = row.service_name as ServiceName
  nextTick(() => fileInputRef.value?.click())
}

async function onFileChosen(e: Event) {
  const input = e.target as HTMLInputElement
  const file = input.files && input.files[0]
  if (!file || !serverId.value || !pendingServiceName.value) {
    if (input) input.value = ""
    return
  }

  const fd = new FormData()
  fd.append("server_id", String(serverId.value))
  fd.append("file", file)
  fd.append("service_name", pendingServiceName.value)

  uploadingService.value = pendingServiceName.value
  try {
    uploadPercent.value = 0
    const res = await uploadServerFile(fd, (percent) => {
      uploadPercent.value = percent
    })
    ElMessage.success(res.data.message || "上传成功")
    setTimeout(() => loadServiceStatus(), 2000)
  } catch {
    // handled by axios interceptor
  } finally {
    uploadingService.value = ""
    uploadPercent.value = 0
    pendingServiceName.value = ""
    if (input) input.value = ""
  }
}

// 配置查看/编辑
const configDialogVisible = ref(false)
const configLoading = ref(false)
const configSaving = ref(false)
const configServiceName = ref<ServiceName>("server")
const configPath = ref("")
const configContent = ref("")

async function onClickConfig(row: ServiceStatusResp) {
  if (!serverId.value) {
    ElMessage.warning("请先选择服务器")
    return
  }
  configDialogVisible.value = true
  configServiceName.value = row.service_name as ServiceName
  configLoading.value = true
  try {
    const { data } = await getProgramConfig({ server_id: serverId.value, service_name: row.service_name as ServiceName })
    configPath.value = data.config_path
    configContent.value = data.content
  } catch {
    ElMessage.error("读取配置失败")
  } finally {
    configLoading.value = false
  }
}

async function onSaveConfig() {
  if (!serverId.value || !configServiceName.value) return
  configSaving.value = true
  try {
    await updateProgramConfig({ server_id: serverId.value, service_name: configServiceName.value, content: configContent.value })
    ElMessage.success("保存成功")
    configDialogVisible.value = false
  } catch {
    // handled globally
  } finally {
    configSaving.value = false
  }
}

// 加载服务器信息
async function loadServerInfo() {
  if (!serverId.value) return
  try {
    const res = await getServerDetail(serverId.value)
    serverInfo.value = res.data
  } catch {
    console.error("加载服务器信息失败")
  }
}

// 加载服务状态
async function loadServiceStatus() {
  if (!serverId.value) return

  loading.value = true
  try {
    const res = await getServiceStatus({
      server_id: serverId.value
    })
    serviceList.value = res.data.services || []
  } catch {
    ElMessage.error("加载服务状态失败")
  } finally {
    loading.value = false
  }
}

// 加载服务器资源
async function loadServerStats() {
  if (!serverId.value) return

  statsLoading.value = true
  try {
    const res = await getServerStats(serverId.value)
    serverStats.value = res.data
  } catch {
    // silent fail
  } finally {
    statsLoading.value = false
  }
}

// 加载 Docker 容器状态
async function loadDockerContainers() {
  if (!serverId.value) return

  dockerLoading.value = true
  try {
    const res = await getDockerContainers(serverId.value)
    dockerContainers.value = res.data.containers || []
  } catch {
    // silent fail - docker may not be installed
    dockerContainers.value = []
  } finally {
    dockerLoading.value = false
  }
}

// 获取 Docker 容器状态类型
function getDockerStatusType(status: string) {
  if (status.toLowerCase().startsWith("up")) {
    return "success"
  } else if (status.toLowerCase().includes("exited")) {
    return "danger"
  }
  return "info"
}

// 刷新所有数据
function refreshAll() {
  loadServiceStatus()
  loadServerStats()
  loadDockerContainers()
}

// 执行操作
async function handleExecute(action: "start" | "stop" | "restart", serviceName: ServiceName) {
  const actionText: Record<string, string> = {
    start: "启动",
    stop: "停止",
    restart: "重启"
  }

  try {
    const message = `确定${actionText[action]}服务 "${serviceName}" 吗？`
    await ElMessageBox.confirm(message, "提示", { type: "warning" })

    executing.value = true
    const res = await serviceAction({
      server_id: serverId.value,
      action,
      service_name: serviceName
    })

    if (res.data.success) {
      ElMessage.success(res.data.message)
    } else {
      ElMessage.warning(res.data.message)
    }

    // 刷新状态
    setTimeout(() => {
      loadServiceStatus()
    }, 2000)
  } catch {
    // 用户取消操作
  } finally {
    executing.value = false
  }
}

// 查看日志
async function viewLogs(serviceName: ServiceName) {
  currentService.value = serviceName
  logDialogVisible.value = true
  loadLogs()
}

// 加载日志
async function loadLogs() {
  logLoading.value = true
  try {
    const res = await getServiceLogs({
      server_id: serverId.value,
      service_name: currentService.value,
      lines: logLines.value
    })
    logContent.value = res.data.logs
  } catch {
    ElMessage.error("读取日志失败")
  } finally {
    logLoading.value = false
  }
}

// 下载日志
function downloadLogs() {
  if (!logContent.value) return

  const blob = new Blob([logContent.value], { type: "text/plain" })
  const url = URL.createObjectURL(blob)
  const a = document.createElement("a")
  a.href = url
  a.download = `${currentService.value}_${Date.now()}.log`
  a.click()
  URL.revokeObjectURL(url)
}

// 状态标签类型
function getStatusType(status: string) {
  return status === "running" ? "success" : "info"
}

// 服务显示名称
function getServiceDisplayName(name: string) {
  const names: Record<string, string> = {
    server: "Server (业务服务)",
    wukongim: "WuKongIM (通讯层)",
    gost: "GOST (隧道服务)"
  }
  return names[name] || name
}

// 初始化
onMounted(() => {
  if (serverId.value) {
    loadServerInfo()
    refreshAll()
  } else {
    ElMessage.warning("请先选择服务器")
  }
  fetchServerOptions("")
})

// 路由 server_id 变化时，自动刷新
watch(() => route.query.server_id, (val) => {
  const nextId = Number(Array.isArray(val) ? val?.[0] : val) || 0
  if (nextId && nextId !== serverId.value) {
    serverId.value = nextId
    selectedServerId.value = nextId
    loadServerInfo()
    refreshAll()
  }
})
</script>

<template>
  <div class="app-container">
    <!-- 顶部：服务器信息 + 全局操作 -->
    <el-card class="mb-4">
      <div class="flex justify-between items-center">
        <div>
          <span class="text-lg font-bold mr-4">{{ serverInfo.name || '未选择服务器' }}</span>
          <span v-if="serverInfo.host" class="text-gray-600">{{ serverInfo.host }}:{{ serverInfo.port }}</span>
        </div>
        <div class="flex items-center gap-3">
          <el-select
            v-model="selectedServerId"
            filterable
            remote
            reserve-keyword
            clearable
            placeholder="切换服务器"
            :remote-method="remoteSearchServers"
            :loading="serverSelectLoading"
            style="width: 300px"
            @change="onServerChange as any"
          >
            <el-option
              v-for="opt in serverOptions"
              :key="opt.value"
              :label="opt.label"
              :value="opt.value"
            />
          </el-select>

          <el-button @click="refreshAll">
            <el-icon><Refresh /></el-icon> 刷新
          </el-button>
        </div>
      </div>
    </el-card>

    <!-- 隐藏文件选择器 -->
    <input ref="fileInputRef" type="file" style="display:none" @change="onFileChosen" />

    <!-- 服务器资源监控 -->
    <el-card v-if="serverStats" v-loading="statsLoading" class="mb-4">
      <template #header>
        <span class="text-base font-bold">服务器资源</span>
      </template>
      <div class="grid grid-cols-4 gap-4">
        <div class="stat-item">
          <div class="stat-label">CPU 使用率</div>
          <div class="stat-value">{{ serverStats.cpu_usage || "-" }}</div>
        </div>
        <div class="stat-item">
          <div class="stat-label">内存使用</div>
          <div class="stat-value">{{ serverStats.memory_usage }} / {{ serverStats.memory_total }}</div>
        </div>
        <div class="stat-item">
          <div class="stat-label">磁盘使用</div>
          <div class="stat-value">{{ serverStats.disk_usage }} ({{ serverStats.disk_total }})</div>
        </div>
        <div class="stat-item">
          <div class="stat-label">系统负载</div>
          <div class="stat-value text-sm">{{ serverStats.load_avg || "-" }}</div>
        </div>
      </div>
    </el-card>

    <!-- 服务列表 -->
    <el-card v-loading="loading">
      <template #header>
        <div class="flex justify-between items-center">
          <span class="text-base font-bold">服务列表 (systemctl)</span>
          <el-button size="small" @click="loadServiceStatus">
            <el-icon><Refresh /></el-icon> 刷新列表
          </el-button>
        </div>
      </template>

      <el-table :data="serviceList" stripe>
        <el-table-column type="index" label="#" width="60" />
        <el-table-column label="服务名称" width="280">
          <template #default="{ row }">
            <span class="font-medium">{{ getServiceDisplayName(row.service_name) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="getStatusType(row.status)" size="small">
              {{ row.status === "running" ? "运行中" : "已停止" }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="pid" label="进程ID" width="100" />
        <el-table-column prop="cpu" label="CPU" width="80" />
        <el-table-column prop="memory" label="内存" width="100" />
        <el-table-column prop="uptime" label="运行时长" width="120" />
        <el-table-column label="操作" width="400" fixed="right">
          <template #default="{ row }">
            <el-button
              v-if="row.status !== 'running'"
              link
              type="success"
              size="small"
              :loading="executing"
              @click="handleExecute('start', row.service_name)"
            >
              启动
            </el-button>
            <el-button
              v-if="row.status === 'running'"
              link
              type="warning"
              size="small"
              :loading="executing"
              @click="handleExecute('stop', row.service_name)"
            >
              停止
            </el-button>
            <el-button
              link
              type="primary"
              size="small"
              :loading="executing"
              @click="handleExecute('restart', row.service_name)"
            >
              重启
            </el-button>
            <el-button
              v-if="canUpload(row.service_name)"
              v-permission="['admin']"
              link
              type="success"
              size="small"
              :loading="uploadingService === row.service_name"
              @click="onClickUpdate(row)"
            >
              上传更新
            </el-button>
            <el-progress
              v-if="uploadingService === row.service_name && uploadPercent > 0"
              :percentage="uploadPercent"
              :stroke-width="6"
              style="width: 100px; margin-left: 8px;"
            />
            <el-button
              v-permission="['admin']"
              link
              type="warning"
              size="small"
              @click="onClickConfig(row)"
            >
              配置
            </el-button>
            <el-button
              v-permission="['admin']"
              link
              type="info"
              size="small"
              @click="viewLogs(row.service_name)"
            >
              日志
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- Docker 容器列表 -->
    <el-card v-if="dockerContainers.length > 0" v-loading="dockerLoading" class="mt-4">
      <template #header>
        <div class="flex justify-between items-center">
          <span class="text-base font-bold">Docker 容器</span>
          <el-button size="small" @click="loadDockerContainers">
            <el-icon><Refresh /></el-icon> 刷新
          </el-button>
        </div>
      </template>

      <el-table :data="dockerContainers" stripe>
        <el-table-column prop="name" label="容器名称" width="150" />
        <el-table-column prop="image" label="镜像" width="180" show-overflow-tooltip />
        <el-table-column label="状态" width="120">
          <template #default="{ row }">
            <el-tag :type="getDockerStatusType(row.status)" size="small">
              {{ row.status }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="cpu_percent" label="CPU" width="80" />
        <el-table-column prop="mem_usage" label="内存" width="140" show-overflow-tooltip />
        <el-table-column prop="mem_percent" label="内存%" width="80" />
        <el-table-column prop="ports" label="端口映射" min-width="180" show-overflow-tooltip />
        <el-table-column prop="running_for" label="运行时长" width="100" />
        <el-table-column prop="container_id" label="容器ID" width="100" />
      </el-table>
    </el-card>

    <!-- 配置查看/编辑 -->
    <el-dialog v-model="configDialogVisible" :title="`配置 - ${configServiceName}`" width="70%" destroy-on-close>
      <div v-loading="configLoading">
        <div class="mb-2 text-gray-500">路径：{{ configPath || '-' }}</div>
        <el-input v-model="configContent" type="textarea" :rows="20" placeholder="配置内容" />
      </div>
      <template #footer>
        <el-button @click="configDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="configSaving" @click="onSaveConfig">保存</el-button>
      </template>
    </el-dialog>

    <!-- 日志查看弹窗 -->
    <el-dialog v-model="logDialogVisible" :title="`服务日志 - ${currentService}`" width="80%" destroy-on-close>
      <div class="log-viewer-container">
        <div class="log-toolbar mb-4">
          <el-form inline>
            <el-form-item label="显示行数">
              <el-select v-model="logLines" @change="loadLogs" style="width: 120px">
                <el-option label="最后100行" :value="100" />
                <el-option label="最后500行" :value="500" />
                <el-option label="最后1000行" :value="1000" />
              </el-select>
            </el-form-item>
            <el-form-item>
              <el-button @click="loadLogs" :loading="logLoading">
                <el-icon><Refresh /></el-icon> 刷新
              </el-button>
              <el-button @click="downloadLogs">
                <el-icon><Download /></el-icon> 下载
              </el-button>
            </el-form-item>
          </el-form>
        </div>
        <div v-loading="logLoading" class="log-content">
          <pre><code>{{ logContent || "暂无日志" }}</code></pre>
        </div>
      </div>
    </el-dialog>
  </div>
</template>

<style lang="scss" scoped>
.app-container {
  padding: 20px;
}

.stat-item {
  text-align: center;
  padding: 16px;
  background: #f5f7fa;
  border-radius: 8px;

  .stat-label {
    font-size: 14px;
    color: #909399;
    margin-bottom: 8px;
  }

  .stat-value {
    font-size: 20px;
    font-weight: bold;
    color: #303133;
  }
}

.log-viewer-container {
  .log-content {
    background: #1e1e1e;
    color: #d4d4d4;
    padding: 16px;
    border-radius: 4px;
    max-height: 600px;
    overflow-y: auto;
    font-family: "Consolas", "Monaco", monospace;
    font-size: 13px;
    line-height: 1.5;

    pre {
      margin: 0;
      white-space: pre-wrap;
      word-wrap: break-word;
    }
  }
}
</style>
