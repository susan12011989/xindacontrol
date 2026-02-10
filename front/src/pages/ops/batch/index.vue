<script lang="ts" setup>
import type { ServerResp, BatchServiceResult, ServerHealthResult } from "@@/apis/deploy/type"
import { getServerList, batchServiceAction, batchHealthCheck } from "@@/apis/deploy"
import { ElMessage, ElMessageBox } from "element-plus"

defineOptions({
  name: "OpsBatch"
})

// 服务器列表
const servers = ref<ServerResp[]>([])
const loading = ref(false)
const selectedServerIds = ref<number[]>([])

// 操作结果
const actionResults = ref<BatchServiceResult[]>([])
const healthResults = ref<ServerHealthResult[]>([])
const showResults = ref(false)
const resultType = ref<"action" | "health">("action")

// 服务操作参数
const actionForm = reactive({
  service_name: "server" as "server" | "wukongim" | "gost",
  action: "restart" as "start" | "stop" | "restart",
  parallel: true
})

// 加载服务器列表
async function loadServers() {
  loading.value = true
  try {
    const res = await getServerList({ page: 1, size: 500, server_type: 1 })
    servers.value = res.data.list || []
  } catch (e) {
    ElMessage.error("加载服务器列表失败")
  } finally {
    loading.value = false
  }
}

// 全选/取消全选
function toggleSelectAll() {
  if (selectedServerIds.value.length === servers.value.length) {
    selectedServerIds.value = []
  } else {
    selectedServerIds.value = servers.value.map(s => s.id)
  }
}

// 执行批量服务操作
async function handleBatchAction() {
  if (selectedServerIds.value.length === 0) {
    ElMessage.warning("请先选择服务器")
    return
  }

  const actionText = { start: "启动", stop: "停止", restart: "重启" }[actionForm.action]

  try {
    await ElMessageBox.confirm(
      `确定要对 ${selectedServerIds.value.length} 台服务器执行 ${actionText} ${actionForm.service_name} 操作吗？`,
      "确认操作",
      { type: "warning" }
    )
  } catch {
    return
  }

  loading.value = true
  try {
    const res = await batchServiceAction({
      server_ids: selectedServerIds.value,
      service_name: actionForm.service_name,
      action: actionForm.action,
      parallel: actionForm.parallel
    })

    actionResults.value = res.data.results
    resultType.value = "action"
    showResults.value = true

    ElMessage.success(`操作完成: 成功 ${res.data.success_count}, 失败 ${res.data.fail_count}`)
  } catch (e: any) {
    ElMessage.error(e.message || "操作失败")
  } finally {
    loading.value = false
  }
}

// 执行批量健康检查
async function handleHealthCheck() {
  if (selectedServerIds.value.length === 0) {
    ElMessage.warning("请先选择服务器")
    return
  }

  loading.value = true
  try {
    const res = await batchHealthCheck({
      server_ids: selectedServerIds.value
    })

    healthResults.value = res.data.results
    resultType.value = "health"
    showResults.value = true

    ElMessage.success(`检查完成: 健康 ${res.data.healthy_count}, 异常 ${res.data.unhealthy_count}`)
  } catch (e: any) {
    ElMessage.error(e.message || "检查失败")
  } finally {
    loading.value = false
  }
}

// 获取状态颜色
function getStatusType(status: string) {
  switch (status) {
    case "healthy":
      return "success"
    case "unhealthy":
    case "error":
      return "danger"
    case "partial":
      return "warning"
    default:
      return "info"
  }
}

// 获取状态文本
function getStatusText(status: string) {
  switch (status) {
    case "healthy":
      return "健康"
    case "unhealthy":
      return "异常"
    case "partial":
      return "部分异常"
    case "error":
      return "错误"
    default:
      return status
  }
}

onMounted(() => {
  loadServers()
})
</script>

<template>
  <div class="app-container">
    <!-- 操作面板 -->
    <el-card class="mb-4">
      <template #header>
        <div class="flex items-center justify-between">
          <span class="font-bold">批量运维操作</span>
          <el-button type="primary" size="small" @click="loadServers">
            刷新列表
          </el-button>
        </div>
      </template>

      <el-form :inline="true" class="mb-4">
        <el-form-item label="服务">
          <el-select v-model="actionForm.service_name" style="width: 120px">
            <el-option label="server" value="server" />
            <el-option label="wukongim" value="wukongim" />
            <el-option label="gost" value="gost" />
          </el-select>
        </el-form-item>
        <el-form-item label="操作">
          <el-select v-model="actionForm.action" style="width: 100px">
            <el-option label="重启" value="restart" />
            <el-option label="启动" value="start" />
            <el-option label="停止" value="stop" />
          </el-select>
        </el-form-item>
        <el-form-item label="并行">
          <el-switch v-model="actionForm.parallel" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :loading="loading" @click="handleBatchAction">
            执行操作
          </el-button>
          <el-button type="success" :loading="loading" @click="handleHealthCheck">
            健康检查
          </el-button>
        </el-form-item>
      </el-form>

      <!-- 服务器选择 -->
      <div class="mb-2 flex items-center gap-2">
        <el-button size="small" @click="toggleSelectAll">
          {{ selectedServerIds.length === servers.length ? "取消全选" : "全选" }}
        </el-button>
        <span class="text-gray-500">
          已选择 {{ selectedServerIds.length }} / {{ servers.length }} 台服务器
        </span>
      </div>

      <el-checkbox-group v-model="selectedServerIds" class="server-grid">
        <el-checkbox
          v-for="server in servers"
          :key="server.id"
          :value="server.id"
          class="server-item"
        >
          <div class="server-info">
            <span class="server-name">{{ server.name }}</span>
            <span class="server-host">{{ server.host }}</span>
          </div>
        </el-checkbox>
      </el-checkbox-group>
    </el-card>

    <!-- 执行结果 -->
    <el-card v-if="showResults">
      <template #header>
        <div class="flex items-center justify-between">
          <span class="font-bold">
            {{ resultType === "action" ? "操作结果" : "健康检查结果" }}
          </span>
          <el-button size="small" @click="showResults = false">关闭</el-button>
        </div>
      </template>

      <!-- 服务操作结果 -->
      <el-table v-if="resultType === 'action'" :data="actionResults" stripe border>
        <el-table-column prop="server_name" label="服务器" width="160" />
        <el-table-column prop="server_host" label="IP" width="140" />
        <el-table-column label="结果" width="100">
          <template #default="{ row }">
            <el-tag :type="row.success ? 'success' : 'danger'">
              {{ row.success ? "成功" : "失败" }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="message" label="消息" show-overflow-tooltip />
        <el-table-column label="输出" width="100">
          <template #default="{ row }">
            <el-popover v-if="row.output" trigger="click" width="400">
              <template #reference>
                <el-button size="small" link>查看</el-button>
              </template>
              <pre class="text-xs whitespace-pre-wrap">{{ row.output }}</pre>
            </el-popover>
            <span v-else class="text-gray-400">-</span>
          </template>
        </el-table-column>
      </el-table>

      <!-- 健康检查结果 -->
      <el-table v-else :data="healthResults" stripe border>
        <el-table-column prop="server_name" label="服务器" width="160" />
        <el-table-column prop="server_host" label="IP" width="140" />
        <el-table-column label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="getStatusType(row.status)">
              {{ getStatusText(row.status) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="message" label="消息" show-overflow-tooltip />
        <el-table-column prop="check_time" label="检查时间" width="180" />
      </el-table>
    </el-card>
  </div>
</template>

<style scoped>
.server-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(240px, 1fr));
  gap: 8px;
}

.server-item {
  border: 1px solid #e4e7ed;
  border-radius: 4px;
  padding: 8px 12px;
  margin: 0 !important;
}

.server-item:hover {
  border-color: #409eff;
}

.server-info {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.server-name {
  font-weight: 500;
}

.server-host {
  font-size: 12px;
  color: #909399;
}
</style>
