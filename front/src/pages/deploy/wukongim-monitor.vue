<script lang="ts" setup>
import type { WuKongIMConnInfo, WuKongIMConnzReq, WuKongIMVarzResp } from "@@/apis/wukongim/type"
import { getWuKongIMConnz, getWuKongIMNodes, getWuKongIMOnlineStatus, getWuKongIMVarz, wukongimDeviceQuit } from "@@/apis/wukongim"
import { Refresh, Search } from "@element-plus/icons-vue"

defineOptions({
  name: "WuKongIMMonitor"
})

// 从路由获取服务器ID
const route = useRoute()
const router = useRouter()
const serverId = ref(Number(route.query.server_id) || 0)

// ========== 服务器选择 ==========
const serverSelectLoading = ref(false)
const serverOptions = ref<{ label: string, value: number }[]>([])
const selectedServerId = ref<number | undefined>(serverId.value || undefined)

async function fetchServerOptions() {
  serverSelectLoading.value = true
  try {
    const res = await getWuKongIMNodes()
    serverOptions.value = (res.data || []).map((n: any) => ({
      value: n.server_id,
      label: `${n.merchant_name || n.merchant_no} (${n.host})`
    }))
    // 如果只有一个节点且未选择，自动选中
    if (!serverId.value && serverOptions.value.length === 1) {
      onServerChange(serverOptions.value[0].value)
    }
  } catch {
    // ignore
  } finally {
    serverSelectLoading.value = false
  }
}

function onServerChange(id: number) {
  if (!id || id === serverId.value) return
  router.push({ name: "WuKongIMMonitor", query: { server_id: String(id) } })
}

// ========== Varz 系统变量 ==========
const varzData = ref<WuKongIMVarzResp | null>(null)
const varzLoading = ref(false)
let varzTimer: any = null

async function loadVarz() {
  if (!serverId.value) return
  varzLoading.value = true
  try {
    const res = await getWuKongIMVarz(serverId.value)
    varzData.value = res.data
  } catch {
    varzData.value = null
  } finally {
    varzLoading.value = false
  }
}

// ========== Connz 连接管理 ==========
const connzData = ref<WuKongIMConnInfo[]>([])
const connzTotal = ref(0)
const connzLoading = ref(false)
const connzParams = ref<WuKongIMConnzReq>({
  server_id: serverId.value,
  offset: 0,
  limit: 20,
  uid: "",
  sort: ""
})

async function loadConnz() {
  if (!serverId.value) return
  connzLoading.value = true
  try {
    connzParams.value.server_id = serverId.value
    const res = await getWuKongIMConnz(connzParams.value)
    connzData.value = res.data.connections || []
    connzTotal.value = res.data.total || 0
  } catch {
    connzData.value = []
    connzTotal.value = 0
  } finally {
    connzLoading.value = false
  }
}

function onConnzSearch() {
  connzParams.value.offset = 0
  loadConnz()
}

function onConnzPageChange(page: number) {
  connzParams.value.offset = (page - 1) * connzParams.value.limit!
  loadConnz()
}

// ========== 用户在线查询 ==========
const onlineQueryUids = ref("")
const onlineResults = ref<{ uid: string, device_flag: number, online: number }[]>([])
const onlineLoading = ref(false)

async function queryOnlineStatus() {
  if (!serverId.value || !onlineQueryUids.value.trim()) return
  onlineLoading.value = true
  try {
    const uids = onlineQueryUids.value.split(",").map(s => s.trim()).filter(Boolean)
    const res = await getWuKongIMOnlineStatus({ server_id: serverId.value, uids })
    onlineResults.value = res.data || []
  } catch {
    onlineResults.value = []
  } finally {
    onlineLoading.value = false
  }
}

// ========== 踢下线 ==========
async function handleKick(uid: string, deviceFlag: number) {
  try {
    await ElMessageBox.confirm(`确定要将用户 ${uid} 强制下线吗？`, "确认操作", { type: "warning" })
    await wukongimDeviceQuit({ server_id: serverId.value, uid, device_flag: deviceFlag })
    ElMessage.success("已强制下线")
    loadConnz()
  } catch {
    // cancelled
  }
}

// ========== 格式化工具 ==========
function formatBytes(bytes: number): string {
  if (bytes === 0) return "0 B"
  const units = ["B", "KB", "MB", "GB", "TB"]
  const i = Math.floor(Math.log(bytes) / Math.log(1024))
  return `${(bytes / Math.pow(1024, i)).toFixed(1)} ${units[i]}`
}

function formatDevice(flag: number): string {
  const map: Record<number, string> = { 0: "APP", 1: "WEB", 2: "PC" }
  return map[flag] ?? `未知(${flag})`
}

// ========== 自动刷新 ==========
const autoRefresh = ref(false)

function startAutoRefresh() {
  stopAutoRefresh()
  if (autoRefresh.value) {
    varzTimer = setInterval(() => {
      loadVarz()
    }, 5000)
  }
}

function stopAutoRefresh() {
  if (varzTimer) {
    clearInterval(varzTimer)
    varzTimer = null
  }
}

watch(autoRefresh, () => {
  startAutoRefresh()
})

// ========== 生命周期 ==========
function refreshAll() {
  if (!serverId.value) return
  loadVarz()
  loadConnz()
}

watch(() => route.query.server_id, (val) => {
  serverId.value = Number(val) || 0
  selectedServerId.value = serverId.value || undefined
  connzParams.value.offset = 0
  connzParams.value.uid = ""
  refreshAll()
})

onMounted(() => {
  fetchServerOptions()
  refreshAll()
})

onUnmounted(() => {
  stopAutoRefresh()
})
</script>

<template>
  <div class="app-container">
    <!-- 顶部：服务器选择 -->
    <el-card shadow="never" class="mb-4">
      <div class="flex items-center gap-4">
        <span class="font-bold text-sm whitespace-nowrap">WuKongIM 监控</span>
        <el-select
          v-model="selectedServerId"
          filterable
          placeholder="选择 WuKongIM 节点..."
          :loading="serverSelectLoading"
          style="width: 320px"
          @change="onServerChange"
        >
          <el-option v-for="opt in serverOptions" :key="opt.value" :label="opt.label" :value="opt.value" />
        </el-select>
        <el-button :icon="Refresh" @click="refreshAll" :disabled="!serverId">
          刷新
        </el-button>
        <el-switch v-model="autoRefresh" active-text="自动刷新(5s)" inactive-text="" :disabled="!serverId" />
      </div>
    </el-card>

    <template v-if="serverId">
      <!-- 系统概览卡片 -->
      <el-card shadow="never" class="mb-4" v-loading="varzLoading">
        <template #header>
          <span class="font-bold">系统概览</span>
        </template>
        <template v-if="varzData">
          <el-row :gutter="20">
            <el-col :span="4">
              <el-statistic title="当前连接数" :value="varzData.connections" />
            </el-col>
            <el-col :span="4">
              <el-statistic title="收到消息" :value="varzData.in_msgs" />
            </el-col>
            <el-col :span="4">
              <el-statistic title="发出消息" :value="varzData.out_msgs" />
            </el-col>
            <el-col :span="4">
              <el-statistic title="Goroutine" :value="varzData.goroutine" />
            </el-col>
            <el-col :span="4">
              <el-statistic title="内存" :value="(formatBytes(varzData.mem) as any)" />
            </el-col>
            <el-col :span="4">
              <el-statistic title="运行时间" :value="(varzData.uptime as any)" />
            </el-col>
          </el-row>
          <el-divider />
          <el-descriptions :column="3" border size="small">
            <el-descriptions-item label="版本">{{ varzData.version }}</el-descriptions-item>
            <el-descriptions-item label="Server ID">{{ varzData.server_id }}</el-descriptions-item>
            <el-descriptions-item label="CPU">{{ varzData.cpu }}%</el-descriptions-item>
            <el-descriptions-item label="收到字节">{{ formatBytes(varzData.in_bytes) }}</el-descriptions-item>
            <el-descriptions-item label="发出字节">{{ formatBytes(varzData.out_bytes) }}</el-descriptions-item>
            <el-descriptions-item label="慢客户端">{{ varzData.slow_clients }}</el-descriptions-item>
            <el-descriptions-item label="重试队列">{{ varzData.retry_queue }}</el-descriptions-item>
            <el-descriptions-item label="TCP 地址">{{ varzData.tcp_addr }}</el-descriptions-item>
            <el-descriptions-item label="WebSocket 地址">{{ varzData.ws_addr }}</el-descriptions-item>
            <el-descriptions-item label="Commit">{{ varzData.commit }}</el-descriptions-item>
            <el-descriptions-item label="Commit Date">{{ varzData.commit_date }}</el-descriptions-item>
            <el-descriptions-item label="Manager Token">
              <el-tag :type="varzData.manager_token_on ? 'success' : 'info'" size="small">
                {{ varzData.manager_token_on ? '已开启' : '未开启' }}
              </el-tag>
            </el-descriptions-item>
          </el-descriptions>
        </template>
        <el-empty v-else description="暂无数据，请选择服务器" />
      </el-card>

      <!-- 连接管理 -->
      <el-card shadow="never" class="mb-4">
        <template #header>
          <div class="flex items-center justify-between">
            <span class="font-bold">连接管理</span>
            <div class="flex items-center gap-2">
              <el-input
                v-model="connzParams.uid"
                placeholder="按 UID 过滤"
                clearable
                style="width: 200px"
                @keyup.enter="onConnzSearch"
                @clear="onConnzSearch"
              />
              <el-select v-model="connzParams.sort" placeholder="排序" clearable style="width: 140px" @change="onConnzSearch">
                <el-option label="ID ↑" value="id" />
                <el-option label="ID ↓" value="idDesc" />
                <el-option label="收消息 ↓" value="inMsgDesc" />
                <el-option label="发消息 ↓" value="outMsgDesc" />
                <el-option label="连接时间 ↓" value="uptimeDesc" />
                <el-option label="空闲 ↓" value="idleDesc" />
              </el-select>
              <el-button :icon="Search" @click="onConnzSearch">查询</el-button>
            </div>
          </div>
        </template>

        <el-table :data="connzData" v-loading="connzLoading" stripe border size="small" style="width: 100%">
          <el-table-column prop="uid" label="UID" min-width="120" show-overflow-tooltip />
          <el-table-column prop="device" label="设备" width="80" />
          <el-table-column prop="ip" label="IP" width="130" />
          <el-table-column prop="uptime" label="连接时间" width="100" />
          <el-table-column prop="idle" label="空闲" width="80" />
          <el-table-column label="消息 (入/出)" width="120">
            <template #default="{ row }">
              {{ row.in_msgs }} / {{ row.out_msgs }}
            </template>
          </el-table-column>
          <el-table-column label="字节 (入/出)" width="140">
            <template #default="{ row }">
              {{ formatBytes(row.in_msg_bytes) }} / {{ formatBytes(row.out_msg_bytes) }}
            </template>
          </el-table-column>
          <el-table-column prop="version" label="协议版本" width="80" align="center" />
          <el-table-column label="操作" width="80" align="center" fixed="right">
            <template #default="{ row }">
              <el-button type="danger" link size="small" @click="handleKick(row.uid, -1)">踢下线</el-button>
            </template>
          </el-table-column>
        </el-table>

        <div class="flex justify-end mt-4" v-if="connzTotal > connzParams.limit!">
          <el-pagination
            background
            layout="total, prev, pager, next"
            :total="connzTotal"
            :page-size="connzParams.limit"
            :current-page="Math.floor(connzParams.offset! / connzParams.limit!) + 1"
            @current-change="onConnzPageChange"
          />
        </div>
      </el-card>

      <!-- 用户在线查询 -->
      <el-card shadow="never">
        <template #header>
          <span class="font-bold">用户在线查询</span>
        </template>
        <div class="flex items-center gap-2 mb-4">
          <el-input
            v-model="onlineQueryUids"
            placeholder="输入 UID，多个用逗号分隔"
            clearable
            style="width: 400px"
            @keyup.enter="queryOnlineStatus"
          />
          <el-button type="primary" :loading="onlineLoading" @click="queryOnlineStatus" :disabled="!onlineQueryUids.trim()">
            查询
          </el-button>
        </div>

        <el-table v-if="onlineResults.length" :data="onlineResults" stripe border size="small" style="width: 100%">
          <el-table-column prop="uid" label="UID" min-width="150" />
          <el-table-column label="设备类型" width="120">
            <template #default="{ row }">{{ formatDevice(row.device_flag) }}</template>
          </el-table-column>
          <el-table-column label="状态" width="100" align="center">
            <template #default="{ row }">
              <el-tag :type="row.online ? 'success' : 'info'" size="small">
                {{ row.online ? '在线' : '离线' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="80" align="center">
            <template #default="{ row }">
              <el-button
                v-if="row.online"
                type="danger"
                link
                size="small"
                @click="handleKick(row.uid, row.device_flag)"
              >
                踢下线
              </el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-card>
    </template>

    <!-- 未选择服务器 -->
    <el-card v-else shadow="never">
      <el-empty description="请先选择一个服务器" />
    </el-card>
  </div>
</template>

<style lang="scss" scoped>
.mb-4 {
  margin-bottom: 16px;
}

.flex {
  display: flex;
}

.items-center {
  align-items: center;
}

.justify-between {
  justify-content: space-between;
}

.justify-end {
  justify-content: flex-end;
}

.gap-2 {
  gap: 8px;
}

.gap-4 {
  gap: 16px;
}

.mt-4 {
  margin-top: 16px;
}

.font-bold {
  font-weight: 600;
}

.text-sm {
  font-size: 14px;
}

.whitespace-nowrap {
  white-space: nowrap;
}
</style>
