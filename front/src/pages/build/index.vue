<script setup lang="ts">
import { ref, onMounted, computed } from "vue"
import { ElMessage, ElMessageBox } from "element-plus"
import { getBuildTasks, getBuildStats, getBuildMerchants, createBuildTask, cancelBuildTask, retryBuildTask, downloadBuildArtifact } from "@/common/apis/build"
import type { BuildTask, BuildMerchant, BuildStats, CreateTaskReq } from "@/common/apis/build/type"
import { TaskStatusLabel, TaskStatusColor, PlatformIcon } from "@/common/apis/build/type"

// 统计数据
const stats = ref<BuildStats | null>(null)
const loadingStats = ref(false)

// 任务列表
const tasks = ref<BuildTask[]>([])
const total = ref(0)
const loading = ref(false)
const page = ref(1)
const size = ref(20)
const statusFilter = ref("")

// 商户配置列表（用于下拉选择）
const merchants = ref<BuildMerchant[]>([])

// 新建任务弹窗
const showCreateDialog = ref(false)
const createForm = ref<CreateTaskReq>({
  build_merchant_id: 0,
  platforms: "",
})
const selectedPlatforms = ref<string[]>([])

// 加载统计
async function loadStats() {
  loadingStats.value = true
  try {
    const res = await getBuildStats()
    stats.value = res.data
  } finally {
    loadingStats.value = false
  }
}

// 加载任务列表
async function loadTasks() {
  loading.value = true
  try {
    const res = await getBuildTasks({
      page: page.value,
      size: size.value,
      status: statusFilter.value,
    })
    tasks.value = res.data.list || []
    total.value = res.data.total
  } finally {
    loading.value = false
  }
}

// 加载商户配置
async function loadMerchants() {
  const res = await getBuildMerchants({ page: 1, size: 100, status: "1" })
  merchants.value = res.data.list || []
}

// 打开新建弹窗
function openCreateDialog() {
  createForm.value = { build_merchant_id: 0, platforms: "" }
  selectedPlatforms.value = []
  showCreateDialog.value = true
}

// 提交新建任务
async function submitCreate() {
  if (!createForm.value.build_merchant_id) {
    ElMessage.warning("请选择商户配置")
    return
  }
  if (selectedPlatforms.value.length === 0) {
    ElMessage.warning("请选择至少一个平台")
    return
  }

  createForm.value.platforms = selectedPlatforms.value.join(",")

  try {
    await createBuildTask(createForm.value)
    ElMessage.success("任务创建成功")
    showCreateDialog.value = false
    loadTasks()
    loadStats()
  } catch (e: any) {
    ElMessage.error(e.message || "创建失败")
  }
}

// 取消任务
async function handleCancel(task: BuildTask) {
  await ElMessageBox.confirm("确定要取消该构建任务吗？", "提示", { type: "warning" })
  await cancelBuildTask(task.id)
  ElMessage.success("已取消")
  loadTasks()
  loadStats()
}

// 重试任务
async function handleRetry(task: BuildTask) {
  await retryBuildTask(task.id)
  ElMessage.success("已创建重试任务")
  loadTasks()
  loadStats()
}

// 下载产物
function handleDownload(artifactId: number) {
  window.open(downloadBuildArtifact(artifactId), "_blank")
}

// 格式化平台显示
function formatPlatforms(platforms: string) {
  return platforms.split(",").map(p => PlatformIcon[p.trim()] || p).join(" ")
}

// 格式化文件大小
function formatFileSize(bytes: number) {
  if (bytes < 1024) return bytes + " B"
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + " KB"
  return (bytes / 1024 / 1024).toFixed(1) + " MB"
}

// 格式化耗时
function formatDuration(seconds: number) {
  if (seconds < 60) return seconds + "秒"
  if (seconds < 3600) return Math.floor(seconds / 60) + "分" + (seconds % 60) + "秒"
  return Math.floor(seconds / 3600) + "时" + Math.floor((seconds % 3600) / 60) + "分"
}

// 分页变化
function handlePageChange(p: number) {
  page.value = p
  loadTasks()
}

onMounted(() => {
  loadStats()
  loadTasks()
  loadMerchants()
})
</script>

<template>
  <div class="app-container">
    <!-- 统计卡片 -->
    <el-row :gutter="20" class="stats-row">
      <el-col :span="6">
        <el-card shadow="hover">
          <template #header>今日构建</template>
          <div class="stat-value">{{ stats?.today.total || 0 }}</div>
          <div class="stat-sub">
            成功 {{ stats?.today.success || 0 }} / 失败 {{ stats?.today.failed || 0 }}
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover">
          <template #header>成功率</template>
          <div class="stat-value">{{ (stats?.today.rate || 0).toFixed(1) }}%</div>
          <div class="stat-sub">进行中 {{ stats?.today.building || 0 }}</div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover">
          <template #header>本周构建</template>
          <div class="stat-value">{{ stats?.week.total || 0 }}</div>
          <div class="stat-sub">
            成功 {{ stats?.week.success || 0 }} / 失败 {{ stats?.week.failed || 0 }}
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover">
          <template #header>平台分布</template>
          <div class="platform-stats">
            <span>🤖 {{ stats?.platforms.android || 0 }}</span>
            <span>🍎 {{ stats?.platforms.ios || 0 }}</span>
            <span>🪟 {{ stats?.platforms.windows || 0 }}</span>
            <span>💻 {{ stats?.platforms.macos || 0 }}</span>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- 操作栏 -->
    <el-card class="filter-card">
      <el-row :gutter="20">
        <el-col :span="6">
          <el-select v-model="statusFilter" placeholder="状态筛选" clearable style="width: 100%" @change="loadTasks">
            <el-option label="全部" value="" />
            <el-option label="排队中" value="0" />
            <el-option label="构建中" value="1" />
            <el-option label="成功" value="2" />
            <el-option label="失败" value="3" />
            <el-option label="已取消" value="4" />
          </el-select>
        </el-col>
        <el-col :span="6">
          <el-button type="primary" @click="openCreateDialog">
            <el-icon><Plus /></el-icon>
            发起构建
          </el-button>
          <el-button @click="loadTasks">
            <el-icon><Refresh /></el-icon>
            刷新
          </el-button>
        </el-col>
      </el-row>
    </el-card>

    <!-- 任务列表 -->
    <el-card>
      <el-table :data="tasks" v-loading="loading" stripe>
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column prop="merchant_name" label="商户配置" min-width="120" />
        <el-table-column label="平台" width="120">
          <template #default="{ row }">
            <span class="platforms">{{ formatPlatforms(row.platforms) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="TaskStatusColor[row.status]" size="small">
              {{ TaskStatusLabel[row.status] }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="进度" width="150">
          <template #default="{ row }">
            <el-progress
              v-if="row.status === 1"
              :percentage="row.progress"
              :stroke-width="10"
              style="width: 100px"
            />
            <span v-else>{{ row.current_step }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="operator" label="操作人" width="100" />
        <el-table-column label="耗时" width="100">
          <template #default="{ row }">
            {{ row.duration ? formatDuration(row.duration) : "-" }}
          </template>
        </el-table-column>
        <el-table-column prop="created_at" label="创建时间" width="180" />
        <el-table-column label="操作" width="150" fixed="right">
          <template #default="{ row }">
            <el-button
              v-if="row.status === 0 || row.status === 1"
              type="danger"
              size="small"
              link
              @click="handleCancel(row)"
            >
              取消
            </el-button>
            <el-button
              v-if="row.status === 3 || row.status === 4"
              type="primary"
              size="small"
              link
              @click="handleRetry(row)"
            >
              重试
            </el-button>
            <el-button
              v-if="row.status === 2"
              type="success"
              size="small"
              link
              @click="$router.push(`/build/tasks/${row.id}`)"
            >
              下载
            </el-button>
          </template>
        </el-table-column>
      </el-table>

      <el-pagination
        v-if="total > size"
        class="pagination"
        layout="total, prev, pager, next"
        :total="total"
        :page-size="size"
        :current-page="page"
        @current-change="handlePageChange"
      />
    </el-card>

    <!-- 新建任务弹窗 -->
    <el-dialog v-model="showCreateDialog" title="发起构建" width="500px">
      <el-form label-width="100px">
        <el-form-item label="商户配置" required>
          <el-select v-model="createForm.build_merchant_id" placeholder="选择商户配置" style="width: 100%">
            <el-option
              v-for="m in merchants"
              :key="m.id"
              :label="m.name"
              :value="m.id"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="目标平台" required>
          <el-checkbox-group v-model="selectedPlatforms">
            <el-checkbox label="android">🤖 Android</el-checkbox>
            <el-checkbox label="ios">🍎 iOS</el-checkbox>
            <el-checkbox label="windows">🪟 Windows</el-checkbox>
            <el-checkbox label="macos">💻 macOS</el-checkbox>
          </el-checkbox-group>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showCreateDialog = false">取消</el-button>
        <el-button type="primary" @click="submitCreate">开始构建</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped lang="scss">
.stats-row {
  margin-bottom: 20px;
}

.stat-value {
  font-size: 32px;
  font-weight: bold;
  color: #409eff;
}

.stat-sub {
  font-size: 12px;
  color: #909399;
  margin-top: 8px;
}

.platform-stats {
  display: flex;
  gap: 16px;
  font-size: 16px;
}

.filter-card {
  margin-bottom: 20px;
}

.platforms {
  font-size: 18px;
}

.pagination {
  margin-top: 20px;
  justify-content: flex-end;
}
</style>
