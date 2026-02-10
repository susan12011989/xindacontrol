<script lang="ts" setup>
import type { ServerResp, LogQueryResp } from "@@/apis/deploy/type"
import { getServerList, queryLogs } from "@@/apis/deploy"
import { ElMessage } from "element-plus"

defineOptions({
  name: "OpsLogs"
})

// 服务器列表
const servers = ref<ServerResp[]>([])
const loading = ref(false)

// 查询表单
const queryForm = reactive({
  server_id: undefined as number | undefined,
  query_type: "journalctl" as "journalctl" | "docker" | "file",
  service_name: "server",
  container_name: "",
  log_path: "",
  lines: 200,
  since: "1h",
  until: "",
  keyword: "",
  level: ""
})

// 日志结果
const logResult = ref<LogQueryResp | null>(null)

// 预设时间范围
const timePresets = [
  { label: "最近 30 分钟", value: "30m" },
  { label: "最近 1 小时", value: "1h" },
  { label: "最近 3 小时", value: "3h" },
  { label: "最近 6 小时", value: "6h" },
  { label: "最近 12 小时", value: "12h" },
  { label: "最近 24 小时", value: "24h" }
]

// 日志级别选项
const levelOptions = [
  { label: "全部", value: "" },
  { label: "Error", value: "error" },
  { label: "Warn", value: "warn" },
  { label: "Info", value: "info" },
  { label: "Debug", value: "debug" }
]

// 加载服务器列表
async function loadServers() {
  try {
    const res = await getServerList({ page: 1, size: 500, server_type: 1 })
    servers.value = res.data.list || []
    if (servers.value.length > 0 && !queryForm.server_id) {
      queryForm.server_id = servers.value[0].id
    }
  } catch (e) {
    ElMessage.error("加载服务器列表失败")
  }
}

// 查询日志
async function handleQuery() {
  if (!queryForm.server_id) {
    ElMessage.warning("请选择服务器")
    return
  }

  loading.value = true
  try {
    const res = await queryLogs({
      server_id: queryForm.server_id,
      query_type: queryForm.query_type,
      service_name: queryForm.query_type === "journalctl" ? queryForm.service_name : undefined,
      container_name: queryForm.query_type === "docker" ? queryForm.container_name : undefined,
      log_path: queryForm.query_type === "file" ? queryForm.log_path : undefined,
      lines: queryForm.lines,
      since: queryForm.since,
      until: queryForm.until || undefined,
      keyword: queryForm.keyword || undefined,
      level: queryForm.level || undefined
    })

    logResult.value = res.data
    if (res.data.line_count === 0) {
      ElMessage.info("未查询到日志")
    }
  } catch (e: any) {
    ElMessage.error(e.message || "查询失败")
  } finally {
    loading.value = false
  }
}

// 复制日志
function copyLogs() {
  if (!logResult.value?.logs) return
  navigator.clipboard.writeText(logResult.value.logs)
  ElMessage.success("已复制到剪贴板")
}

// 下载日志
function downloadLogs() {
  if (!logResult.value?.logs) return
  const blob = new Blob([logResult.value.logs], { type: "text/plain" })
  const url = URL.createObjectURL(blob)
  const a = document.createElement("a")
  a.href = url
  a.download = `logs_${queryForm.server_id}_${Date.now()}.log`
  a.click()
  URL.revokeObjectURL(url)
}

onMounted(() => {
  loadServers()
})
</script>

<template>
  <div class="app-container">
    <!-- 查询面板 -->
    <el-card class="mb-4">
      <template #header>
        <span class="font-bold">日志查询</span>
      </template>

      <el-form :inline="true" :model="queryForm" class="query-form">
        <el-form-item label="服务器" required>
          <el-select
            v-model="queryForm.server_id"
            filterable
            placeholder="选择服务器"
            style="width: 200px"
          >
            <el-option
              v-for="server in servers"
              :key="server.id"
              :label="`${server.name} (${server.host})`"
              :value="server.id"
            />
          </el-select>
        </el-form-item>

        <el-form-item label="日志类型">
          <el-select v-model="queryForm.query_type" style="width: 120px">
            <el-option label="Systemd" value="journalctl" />
            <el-option label="Docker" value="docker" />
            <el-option label="文件" value="file" />
          </el-select>
        </el-form-item>

        <!-- Systemd 服务选择 -->
        <el-form-item v-if="queryForm.query_type === 'journalctl'" label="服务">
          <el-select v-model="queryForm.service_name" style="width: 120px">
            <el-option label="server" value="server" />
            <el-option label="wukongim" value="wukongim" />
            <el-option label="gost" value="gost" />
          </el-select>
        </el-form-item>

        <!-- Docker 容器名 -->
        <el-form-item v-if="queryForm.query_type === 'docker'" label="容器">
          <el-input
            v-model="queryForm.container_name"
            placeholder="容器名称"
            style="width: 150px"
          />
        </el-form-item>

        <!-- 文件路径 -->
        <el-form-item v-if="queryForm.query_type === 'file'" label="路径">
          <el-input
            v-model="queryForm.log_path"
            placeholder="/path/to/log"
            style="width: 250px"
          />
        </el-form-item>

        <el-form-item label="时间范围">
          <el-select v-model="queryForm.since" style="width: 140px">
            <el-option
              v-for="preset in timePresets"
              :key="preset.value"
              :label="preset.label"
              :value="preset.value"
            />
          </el-select>
        </el-form-item>

        <el-form-item label="行数">
          <el-input-number v-model="queryForm.lines" :min="10" :max="5000" :step="100" />
        </el-form-item>

        <el-form-item label="关键字">
          <el-input
            v-model="queryForm.keyword"
            placeholder="过滤关键字"
            clearable
            style="width: 150px"
          />
        </el-form-item>

        <el-form-item label="级别">
          <el-select v-model="queryForm.level" style="width: 100px">
            <el-option
              v-for="level in levelOptions"
              :key="level.value"
              :label="level.label"
              :value="level.value"
            />
          </el-select>
        </el-form-item>

        <el-form-item>
          <el-button type="primary" :loading="loading" @click="handleQuery">
            查询日志
          </el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <!-- 日志展示 -->
    <el-card v-if="logResult">
      <template #header>
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-4">
            <span class="font-bold">日志内容</span>
            <el-tag size="small">{{ logResult.line_count }} 行</el-tag>
            <el-tag v-if="logResult.truncated" type="warning" size="small">已截断</el-tag>
          </div>
          <div class="flex gap-2">
            <el-button size="small" @click="copyLogs">复制</el-button>
            <el-button size="small" @click="downloadLogs">下载</el-button>
          </div>
        </div>
      </template>

      <div class="log-container">
        <pre class="log-content">{{ logResult.logs || '(空)' }}</pre>
      </div>

      <!-- 调试信息 -->
      <div v-if="logResult.command" class="mt-2 text-xs text-gray-400">
        执行命令: {{ logResult.command }}
      </div>
    </el-card>
  </div>
</template>

<style scoped>
.query-form {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.log-container {
  max-height: 600px;
  overflow: auto;
  background: #1e1e1e;
  border-radius: 4px;
  padding: 12px;
}

.log-content {
  margin: 0;
  font-family: "Fira Code", "Consolas", monospace;
  font-size: 12px;
  line-height: 1.5;
  color: #d4d4d4;
  white-space: pre-wrap;
  word-break: break-all;
}
</style>
