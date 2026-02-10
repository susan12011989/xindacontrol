<script lang="ts" setup>
import type { DockerContainerResp, HealthCheckResponse, HealthCheckItem } from "@@/apis/docker/type"
import type { FeatureFlagResp } from "@@/apis/feature/type"
import { getServerList } from "@@/apis/deploy"
import { batchOperateContainers, getContainerList, getContainerLogs, operateContainer, checkServerHealth, batchCheckServerHealth } from "@@/apis/docker"
import { getFeatureFlags, updateFeatureFlag } from "@@/apis/feature"
import { ArrowDown, ArrowUp } from "@element-plus/icons-vue"

defineOptions({
  name: "DeployDocker"
})

// 当前 Tab
const activeTab = ref("containers")

// 服务器选择
const serverList = ref<any[]>([])
const currentServerId = ref(0)
const currentServerInfo = ref<any>({})

// 容器列表
const containerList = ref<DockerContainerResp[]>([])
const loading = ref(false)
const selectedIds = ref<string[]>([])

// 筛选条件
const queryForm = reactive({
  status: "all" as "all" | "running" | "exited",
  name: ""
})

// 日志查看
const logDialogVisible = ref(false)
const currentContainer = ref<DockerContainerResp | null>(null)
const logContent = ref("")
const logLoading = ref(false)
const logOptions = reactive({
  lines: 100,
  timestamps: false
})
const logTimeRange = ref<[Date, Date]>()

// 健康检查
const healthLoading = ref(false)
const healthResult = ref<HealthCheckResponse | null>(null)
const batchHealthLoading = ref(false)
const batchHealthResults = ref<HealthCheckResponse[]>([])
const selectedServerIds = ref<number[]>([])

// 服务-容器名称映射（用于健康检查中重启容器）
const serviceContainerMap: Record<string, string> = {
  "MySQL": "mysql",
  "Redis": "redis",
  "Nginx": "nginx",
  "Docker": "",  // Docker 本身不能通过 Docker API 重启
  "商户管理后台(GET)": "manager",
  "后端API直连(GET)": "api",
  "WuKongIM": "wukongim",
  "WebSocket": "ws"
}

// 容器重启状态
const restartingServices = ref<Set<string>>(new Set())

// 功能开关
const featureLoading = ref(false)
const featureFlagsVisible = ref(false)
const featureFlags = ref<FeatureFlagResp[]>([])

// 加载服务器列表
async function loadServers() {
  try {
    const res = await getServerList({ page: 1, size: 5000 })
    serverList.value = res.data.list
    if (serverList.value.length > 0 && !currentServerId.value) {
      currentServerId.value = serverList.value[0].id
      loadContainers()
    }
  } catch (error) {
    console.error("加载服务器列表失败", error)
  }
}

// 加载容器列表
async function loadContainers() {
  if (!currentServerId.value) return

  loading.value = true
  try {
    const res = await getContainerList({
      server_id: currentServerId.value,
      status: queryForm.status,
      name: queryForm.name
    })
    containerList.value = res.data.list || []
    currentServerInfo.value = res.data.server_info
  } catch (error) {
    console.error("加载容器列表失败", error)
  } finally {
    loading.value = false
  }
}

// 刷新列表
function refreshContainers() {
  loadContainers()
}

// 筛选查询
function handleQuery() {
  loadContainers()
}

// 选择改变
function handleSelectionChange(selection: DockerContainerResp[]) {
  selectedIds.value = selection.map((item) => item.container_id)
}

// 单个操作
async function handleOperate(container: DockerContainerResp, action: string) {
  const actionText = {
    start: "启动",
    stop: "停止",
    restart: "重启",
    remove: "删除"
  }[action]

  try {
    await ElMessageBox.confirm(`确定${actionText}容器 "${container.name}" 吗？`, "提示", {
      type: "warning"
    })

    await operateContainer({
      server_id: currentServerId.value,
      container_id: container.container_id,
      action: action as any,
      force: action === "remove"
    })

    ElMessage.success(`${actionText}成功`)
    refreshContainers()
  } catch {
    // 用户取消操作
  }
}

// 批量操作
async function handleBatchOperate(action: string) {
  if (selectedIds.value.length === 0) {
    ElMessage.warning("请先选择容器")
    return
  }

  const actionText = {
    start: "启动",
    stop: "停止",
    restart: "重启"
  }[action]

  try {
    await ElMessageBox.confirm(`确定批量${actionText} ${selectedIds.value.length} 个容器吗？`, "提示", {
      type: "warning"
    })

    await batchOperateContainers({
      server_id: currentServerId.value,
      container_ids: selectedIds.value,
      action: action as any
    })

    ElMessage.success(`批量${actionText}成功`)
    refreshContainers()
  } catch {
    // 用户取消操作
  }
}

// 查看日志
function viewLogs(container: DockerContainerResp) {
  currentContainer.value = container
  logDialogVisible.value = true
  loadLogs()
}

// 加载日志
async function loadLogs() {
  if (!currentContainer.value) return

  logLoading.value = true
  try {
    const params: any = {
      server_id: currentServerId.value,
      container_id: currentContainer.value.container_id,
      lines: logOptions.lines,
      timestamps: logOptions.timestamps
    }

    // 时间范围
    if (logTimeRange.value) {
      params.since = logTimeRange.value[0].toISOString()
      params.until = logTimeRange.value[1].toISOString()
    }

    const res = await getContainerLogs(params)
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
  a.download = `${currentContainer.value?.name}_${Date.now()}.log`
  a.click()
  URL.revokeObjectURL(url)
}

// 状态类型
function getStatusType(state: string) {
  const map: Record<string, any> = {
    running: "success",
    exited: "info",
    paused: "warning",
    restarting: "warning"
  }
  return map[state] || "info"
}

function getStatusText(state: string) {
  const map: Record<string, string> = {
    running: "运行中",
    exited: "已停止",
    paused: "已暂停",
    restarting: "重启中"
  }
  return map[state] || state
}

// ========== 健康检查相关 ==========

// 单个服务器健康检查
async function runHealthCheck() {
  if (!currentServerId.value) {
    ElMessage.warning("请先选择服务器")
    return
  }

  healthLoading.value = true
  healthResult.value = null
  try {
    const res = await checkServerHealth(currentServerId.value)
    healthResult.value = res.data
  } catch (error) {
    console.error("健康检查失败", error)
    ElMessage.error("健康检查失败")
  } finally {
    healthLoading.value = false
  }
}

// 批量健康检查
async function runBatchHealthCheck() {
  if (selectedServerIds.value.length === 0) {
    ElMessage.warning("请先选择要检查的服务器")
    return
  }

  batchHealthLoading.value = true
  batchHealthResults.value = []
  try {
    const res = await batchCheckServerHealth({ server_ids: selectedServerIds.value })
    batchHealthResults.value = res.data.results
  } catch (error) {
    console.error("批量健康检查失败", error)
    ElMessage.error("批量健康检查失败")
  } finally {
    batchHealthLoading.value = false
  }
}

// 健康状态样式
function getHealthStatusType(status: string) {
  const map: Record<string, any> = {
    ok: "success",
    error: "danger",
    warning: "warning"
  }
  return map[status] || "info"
}

function getOverallStatusType(overall: string) {
  const map: Record<string, any> = {
    healthy: "success",
    unhealthy: "danger",
    partial: "warning"
  }
  return map[overall] || "info"
}

function getOverallStatusText(overall: string) {
  const map: Record<string, string> = {
    healthy: "健康",
    unhealthy: "异常",
    partial: "部分异常"
  }
  return map[overall] || overall
}

// 服务器多选改变
function handleServerSelectionChange(val: number[]) {
  selectedServerIds.value = val
}

// ========== 健康检查中操作容器 ==========

// 执行服务操作（启动/重启/部署）
async function executeServiceAction(item: HealthCheckItem) {
  const containerName = item.container_name
  const action = item.action

  if (!action || action === "none") {
    ElMessage.warning("该服务需要手动处理")
    return
  }

  if (action === "deploy") {
    ElMessage.info("部署功能即将上线，请手动部署该服务")
    return
  }

  const actionText = action === "start" ? "启动" : "重启"

  try {
    await ElMessageBox.confirm(
      `确定要${actionText} "${item.name}" 对应的容器吗？`,
      `确认${actionText}`,
      { type: "warning" }
    )
  } catch {
    return // 用户取消
  }

  restartingServices.value.add(item.name)

  try {
    // 通过容器名称查找容器
    const res = await getContainerList({
      server_id: currentServerId.value,
      name: containerName || ""
    })

    const containers = res.data.list || []
    if (containers.length === 0) {
      ElMessage.error(`未找到匹配 "${containerName}" 的容器`)
      return
    }

    // 执行操作
    const container = containers[0]
    await operateContainer({
      server_id: currentServerId.value,
      container_id: container.container_id,
      action: action as any
    })

    ElMessage.success(`${item.name} 容器${actionText}成功`)

    // 等待几秒后重新检查健康状态
    setTimeout(() => {
      runHealthCheck()
    }, 3000)
  } catch (error) {
    ElMessage.error(`${actionText} ${item.name} 失败`)
    console.error(error)
  } finally {
    restartingServices.value.delete(item.name)
  }
}

// 重启API对应的容器（用于API健康检查表格）
async function restartServiceContainer(apiName: string) {
  const containerName = serviceContainerMap[apiName]
  if (!containerName) {
    ElMessage.warning("未找到该API对应的容器")
    return
  }

  try {
    await ElMessageBox.confirm(
      `确定要重启 "${apiName}" 对应的容器 "${containerName}" 吗？`,
      "确认重启",
      { type: "warning" }
    )
  } catch {
    return // 用户取消
  }

  restartingServices.value.add(apiName)

  try {
    // 通过容器名称查找容器
    const res = await getContainerList({
      server_id: currentServerId.value,
      name: containerName
    })

    const containers = res.data.list || []
    if (containers.length === 0) {
      ElMessage.error(`未找到匹配 "${containerName}" 的容器`)
      return
    }

    // 重启容器
    const container = containers[0]
    await operateContainer({
      server_id: currentServerId.value,
      container_id: container.container_id,
      action: "restart"
    })

    ElMessage.success(`重启 ${apiName} 容器成功`)

    // 等待几秒后重新检查健康状态
    setTimeout(() => {
      runHealthCheck()
    }, 3000)
  } catch (error) {
    ElMessage.error(`重启 ${apiName} 容器失败`)
    console.error(error)
  } finally {
    restartingServices.value.delete(apiName)
  }
}

// ========== 功能开关相关 ==========

// 加载功能开关
async function loadFeatureFlags() {
  const merchantId = currentServerInfo.value?.merchant_id
  if (!merchantId) {
    ElMessage.warning("当前服务器未关联商户")
    return
  }

  featureLoading.value = true
  try {
    const res = await getFeatureFlags(merchantId) as any
    if (res.code === 0 || res.code === 200) {
      featureFlags.value = res.data.list || []
    }
  } catch (error) {
    ElMessage.error("加载功能开关失败")
    console.error(error)
  } finally {
    featureLoading.value = false
  }
}

// 切换功能开关
async function toggleFeatureFlag(flag: FeatureFlagResp) {
  const merchantId = currentServerInfo.value?.merchant_id
  if (!merchantId) return

  const newEnabled = !flag.enabled
  try {
    const res = await updateFeatureFlag({
      merchant_id: merchantId,
      feature_name: flag.feature_name,
      enabled: newEnabled
    }) as any
    if (res.code === 0 || res.code === 200) {
      ElMessage.success(`${flag.label} 已${newEnabled ? "启用" : "禁用"}`)
      await loadFeatureFlags()
    } else {
      ElMessage.error(res.message || "操作失败")
    }
  } catch (error) {
    ElMessage.error("操作失败")
    console.error(error)
  }
}

// 功能分类颜色
function getFeatureCategoryType(category: string): "success" | "warning" | "info" | "danger" | "primary" {
  const types: Record<string, "success" | "warning" | "info" | "danger" | "primary"> = {
    "支付": "danger",
    "通讯": "success",
    "社交": "warning",
    "安全": "danger",
    "工具": "primary",
    "娱乐": "info",
    "服务": "warning"
  }
  return types[category] || "primary"
}

// Tab 切换
function handleTabChange(tab: string | number) {
  if (tab === "health" && currentServerId.value) {
    // 切换到健康检查 tab 时自动执行检查
    runHealthCheck()
  }
}

// 初始化
onMounted(() => {
  loadServers()
})

// 切换服务器
watch(currentServerId, () => {
  if (currentServerId.value) {
    loadContainers()
    // 如果在健康检查 tab，自动重新检查
    if (activeTab.value === "health") {
      runHealthCheck()
    }
  }
})
</script>

<template>
  <div class="app-container">
    <!-- 顶部：服务器选择 -->
    <el-card class="mb-4">
      <div class="flex justify-between items-center">
        <div class="flex items-center gap-4">
          <span class="text-base font-bold">服务器:</span>
          <el-select v-model="currentServerId" placeholder="请选择服务器" style="width: 300px">
            <el-option
              v-for="server in serverList"
              :key="server.id"
              :label="`${server.name} (${server.host})`"
              :value="server.id"
            />
          </el-select>
          <span v-if="currentServerInfo.merchant_name" class="text-sm text-gray-500">
            商户: {{ currentServerInfo.merchant_name }}
          </span>
        </div>
      </div>
    </el-card>

    <!-- Tab 切换 -->
    <el-tabs v-model="activeTab" @tab-change="handleTabChange">
      <!-- 容器管理 Tab -->
      <el-tab-pane label="容器管理" name="containers">
        <!-- 操作按钮 -->
        <el-card class="mb-4">
          <div class="flex justify-between items-center">
            <el-form inline>
              <el-form-item label="状态">
                <el-select v-model="queryForm.status" @change="handleQuery" style="width: 120px">
                  <el-option label="全部" value="all" />
                  <el-option label="运行中" value="running" />
                  <el-option label="已停止" value="exited" />
                </el-select>
              </el-form-item>
              <el-form-item label="容器名称">
                <el-input
                  v-model="queryForm.name"
                  placeholder="输入容器名称搜索"
                  clearable
                  @clear="handleQuery"
                  @keyup.enter="handleQuery"
                  style="width: 200px"
                />
              </el-form-item>
              <el-form-item>
                <el-button type="primary" @click="handleQuery">查询</el-button>
              </el-form-item>
            </el-form>
            <el-button-group>
              <el-button @click="refreshContainers">
                <el-icon><Refresh /></el-icon> 刷新
              </el-button>
              <el-button type="success" :disabled="selectedIds.length === 0" @click="handleBatchOperate('start')">
                批量启动
              </el-button>
              <el-button type="warning" :disabled="selectedIds.length === 0" @click="handleBatchOperate('restart')">
                批量重启
              </el-button>
              <el-button type="danger" :disabled="selectedIds.length === 0" @click="handleBatchOperate('stop')">
                批量停止
              </el-button>
            </el-button-group>
          </div>
        </el-card>

        <!-- 容器列表 -->
        <el-card v-loading="loading">
          <el-table :data="containerList" @selection-change="handleSelectionChange">
            <el-table-column type="selection" width="50" />
            <el-table-column prop="container_id" label="容器ID" width="120" />
            <el-table-column prop="name" label="容器名称" width="200" />
            <el-table-column prop="image" label="镜像" show-overflow-tooltip />
            <el-table-column label="状态" width="250">
              <template #default="{ row }">
                <el-tag :type="getStatusType(row.state)">
                  {{ getStatusText(row.state) }}
                </el-tag>
                <span class="ml-2 text-xs text-gray-500">{{ row.status }}</span>
              </template>
            </el-table-column>
            <el-table-column prop="ports" label="端口" width="150" />
            <el-table-column prop="created_at" label="创建时间" width="180" />
            <el-table-column label="操作" width="250" fixed="right">
              <template #default="{ row }">
                <el-button
                  v-if="row.state !== 'running'"
                  link
                  type="success"
                  @click="handleOperate(row, 'start')"
                >
                  启动
                </el-button>
                <el-button v-if="row.state === 'running'" link type="warning" @click="handleOperate(row, 'stop')">
                  停止
                </el-button>
                <el-button link type="primary" @click="handleOperate(row, 'restart')"> 重启 </el-button>
                <el-button link type="info" @click="viewLogs(row)"> 日志 </el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-tab-pane>

      <!-- 健康检查 Tab -->
      <el-tab-pane label="健康检查" name="health">
        <!-- 单服务器健康检查 -->
        <el-card class="mb-4">
          <template #header>
            <div class="flex justify-between items-center">
              <span class="font-bold">当前服务器健康状态</span>
              <el-button type="primary" :loading="healthLoading" @click="runHealthCheck">
                <el-icon><Refresh /></el-icon> 检查
              </el-button>
            </div>
          </template>

          <div v-if="healthLoading" class="text-center py-8">
            <el-icon class="is-loading" :size="32"><Loading /></el-icon>
            <p class="mt-2 text-gray-500">正在检查服务器健康状态...</p>
          </div>

          <div v-else-if="healthResult">
            <!-- 总体状态 -->
            <div class="mb-6 p-4 bg-gray-50 rounded-lg">
              <div class="flex items-center gap-4">
                <el-tag :type="getOverallStatusType(healthResult.overall)" size="large">
                  {{ getOverallStatusText(healthResult.overall) }}
                </el-tag>
                <div class="text-sm text-gray-500">
                  <span>服务器: {{ healthResult.server_name }} ({{ healthResult.server_host }})</span>
                  <span class="ml-4">检查时间: {{ healthResult.check_time }}</span>
                </div>
              </div>
            </div>

            <!-- 基础服务状态 -->
            <div class="mb-6">
              <h4 class="text-base font-bold mb-3">基础服务</h4>
              <el-table :data="healthResult.services" border>
                <el-table-column prop="name" label="服务名称" width="150" />
                <el-table-column label="状态" width="100">
                  <template #default="{ row }">
                    <el-tag :type="getHealthStatusType(row.status)" size="small">
                      {{ row.status === 'ok' ? '正常' : row.status === 'error' ? '异常' : '警告' }}
                    </el-tag>
                  </template>
                </el-table-column>
                <el-table-column prop="message" label="说明" />
                <el-table-column prop="latency" label="响应时间" width="120">
                  <template #default="{ row }">
                    {{ row.latency }}ms
                  </template>
                </el-table-column>
                <el-table-column label="操作" width="120" fixed="right">
                  <template #default="{ row }">
                    <el-button
                      v-if="row.action && row.action !== 'none'"
                      link
                      :type="row.action === 'deploy' ? 'primary' : 'warning'"
                      :loading="restartingServices.has(row.name)"
                      @click="executeServiceAction(row)"
                    >
                      {{ row.action_label || '操作' }}
                    </el-button>
                    <span v-else class="text-gray-400">-</span>
                  </template>
                </el-table-column>
              </el-table>
            </div>

            <!-- API 健康状态 -->
            <div>
              <h4 class="text-base font-bold mb-3">API 服务</h4>
              <el-table :data="healthResult.apis" border>
                <el-table-column prop="name" label="API名称" width="180" />
                <el-table-column label="状态" width="100">
                  <template #default="{ row }">
                    <el-tag :type="getHealthStatusType(row.status)" size="small">
                      {{ row.status === 'ok' ? '正常' : row.status === 'error' ? '异常' : '警告' }}
                    </el-tag>
                  </template>
                </el-table-column>
                <el-table-column prop="message" label="说明" />
                <el-table-column prop="latency" label="响应时间" width="120">
                  <template #default="{ row }">
                    {{ row.latency }}ms
                  </template>
                </el-table-column>
                <el-table-column label="操作" width="120" fixed="right">
                  <template #default="{ row }">
                    <el-button
                      v-if="row.action && row.action !== 'none'"
                      link
                      :type="row.action === 'deploy' ? 'primary' : 'warning'"
                      :loading="restartingServices.has(row.name)"
                      @click="executeServiceAction(row)"
                    >
                      {{ row.action_label || '操作' }}
                    </el-button>
                    <span v-else class="text-gray-400">-</span>
                  </template>
                </el-table-column>
              </el-table>
            </div>

            <!-- 功能开关快捷面板 -->
            <el-card class="mt-6" v-if="currentServerInfo.merchant_id">
              <template #header>
                <div class="flex justify-between items-center">
                  <span class="font-bold">
                    功能开关 (商户: {{ currentServerInfo.merchant_name }})
                  </span>
                  <div>
                    <el-button
                      v-if="!featureFlagsVisible"
                      link
                      @click="featureFlagsVisible = true; loadFeatureFlags()"
                    >
                      <el-icon><ArrowDown /></el-icon> 展开
                    </el-button>
                    <el-button
                      v-else
                      link
                      @click="featureFlagsVisible = false"
                    >
                      <el-icon><ArrowUp /></el-icon> 收起
                    </el-button>
                  </div>
                </div>
              </template>

              <div v-if="featureFlagsVisible" v-loading="featureLoading">
                <el-table :data="featureFlags" size="small" max-height="300">
                  <el-table-column prop="category" label="分类" width="80">
                    <template #default="{ row }">
                      <el-tag :type="getFeatureCategoryType(row.category)" size="small">{{ row.category }}</el-tag>
                    </template>
                  </el-table-column>
                  <el-table-column prop="label" label="功能名称" width="140" />
                  <el-table-column prop="description" label="说明" />
                  <el-table-column label="状态" width="80">
                    <template #default="{ row }">
                      <el-switch
                        :model-value="row.enabled"
                        @change="toggleFeatureFlag(row)"
                        size="small"
                      />
                    </template>
                  </el-table-column>
                </el-table>
              </div>

              <div v-else class="text-center text-gray-400 py-2">
                点击"展开"查看和管理功能开关
              </div>
            </el-card>
          </div>

          <div v-else class="text-center py-8 text-gray-400">
            点击"检查"按钮开始健康检查
          </div>
        </el-card>

        <!-- 批量健康检查 -->
        <el-card>
          <template #header>
            <div class="flex justify-between items-center">
              <span class="font-bold">批量健康检查</span>
              <el-button type="primary" :loading="batchHealthLoading" :disabled="selectedServerIds.length === 0" @click="runBatchHealthCheck">
                <el-icon><Refresh /></el-icon> 批量检查 ({{ selectedServerIds.length }})
              </el-button>
            </div>
          </template>

          <!-- 服务器选择 -->
          <div class="mb-4">
            <el-checkbox-group v-model="selectedServerIds">
              <el-checkbox
                v-for="server in serverList"
                :key="server.id"
                :value="server.id"
                :label="server.id"
                border
                class="mr-2 mb-2"
              >
                {{ server.name }} ({{ server.host }})
              </el-checkbox>
            </el-checkbox-group>
          </div>

          <!-- 批量检查结果 -->
          <div v-if="batchHealthLoading" class="text-center py-8">
            <el-icon class="is-loading" :size="32"><Loading /></el-icon>
            <p class="mt-2 text-gray-500">正在批量检查服务器健康状态...</p>
          </div>

          <div v-else-if="batchHealthResults.length > 0">
            <el-table :data="batchHealthResults" border>
              <el-table-column prop="server_name" label="服务器" width="200">
                <template #default="{ row }">
                  <div>{{ row.server_name }}</div>
                  <div class="text-xs text-gray-400">{{ row.server_host }}</div>
                </template>
              </el-table-column>
              <el-table-column label="总体状态" width="120">
                <template #default="{ row }">
                  <el-tag :type="getOverallStatusType(row.overall)">
                    {{ getOverallStatusText(row.overall) }}
                  </el-tag>
                </template>
              </el-table-column>
              <el-table-column label="基础服务">
                <template #default="{ row }">
                  <el-tag
                    v-for="item in row.services"
                    :key="item.name"
                    :type="getHealthStatusType(item.status)"
                    size="small"
                    class="mr-1"
                  >
                    {{ item.name }}
                  </el-tag>
                </template>
              </el-table-column>
              <el-table-column label="API服务">
                <template #default="{ row }">
                  <el-tag
                    v-for="item in row.apis"
                    :key="item.name"
                    :type="getHealthStatusType(item.status)"
                    size="small"
                    class="mr-1"
                  >
                    {{ item.name }}
                  </el-tag>
                </template>
              </el-table-column>
              <el-table-column prop="check_time" label="检查时间" width="180" />
            </el-table>
          </div>

          <div v-else class="text-center py-4 text-gray-400">
            选择服务器后点击"批量检查"按钮
          </div>
        </el-card>
      </el-tab-pane>
    </el-tabs>

    <!-- 日志查看弹窗 -->
    <el-dialog
      v-model="logDialogVisible"
      :title="`容器日志 - ${currentContainer?.name}`"
      width="80%"
      destroy-on-close
    >
      <div class="log-viewer-container">
        <!-- 工具栏 -->
        <div class="log-toolbar mb-4">
          <el-form inline>
            <el-form-item label="显示行数">
              <el-select v-model="logOptions.lines" @change="loadLogs" style="width: 120px">
                <el-option label="最后100行" :value="100" />
                <el-option label="最后500行" :value="500" />
                <el-option label="最后1000行" :value="1000" />
              </el-select>
            </el-form-item>

            <el-form-item label="时间范围">
              <el-date-picker
                v-model="logTimeRange"
                type="datetimerange"
                range-separator="至"
                start-placeholder="开始时间"
                end-placeholder="结束时间"
                @change="loadLogs"
              />
            </el-form-item>

            <el-form-item>
              <el-checkbox v-model="logOptions.timestamps" @change="loadLogs"> 显示时间戳 </el-checkbox>
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

        <!-- 日志内容 -->
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
