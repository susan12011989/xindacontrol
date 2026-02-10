<template>
  <div class="app-container">
    <!-- 顶部操作栏 -->
    <el-card shadow="never" class="mb-4">
      <div class="flex items-center justify-between">
        <div class="flex items-center gap-4">
          <el-select v-model="filterService" placeholder="服务类型" clearable style="width: 150px" @change="loadVersions">
            <el-option label="全部" value="" />
            <el-option label="server" value="server" />
            <el-option label="wukongim" value="wukongim" />
          </el-select>
          <el-button type="primary" @click="showUploadDialog = true">
            <el-icon class="mr-1"><Upload /></el-icon>
            上传版本
          </el-button>
        </div>
        <el-button @click="loadVersions">
          <el-icon class="mr-1"><Refresh /></el-icon>
          刷新
        </el-button>
      </div>
    </el-card>

    <!-- 版本列表 -->
    <el-card shadow="never" class="mb-4">
      <template #header>
        <span class="font-bold">版本列表</span>
      </template>
      <el-table :data="versions" v-loading="loading" stripe border>
        <el-table-column prop="service_name" label="服务" width="100">
          <template #default="{ row }">
            <el-tag :type="row.service_name === 'server' ? 'primary' : 'success'" size="small">
              {{ row.service_name }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="version" label="版本号" width="150">
          <template #default="{ row }">
            <div class="flex items-center gap-2">
              <span>{{ row.version }}</span>
              <el-tag v-if="row.is_current" type="warning" size="small">当前</el-tag>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="file_size" label="文件大小" width="120">
          <template #default="{ row }">
            {{ formatFileSize(row.file_size) }}
          </template>
        </el-table-column>
        <el-table-column prop="file_hash" label="SHA256" width="180">
          <template #default="{ row }">
            <el-tooltip :content="row.file_hash" placement="top">
              <span class="text-gray-500 text-xs font-mono">{{ row.file_hash.substring(0, 16) }}...</span>
            </el-tooltip>
          </template>
        </el-table-column>
        <el-table-column prop="changelog" label="更新日志" min-width="200">
          <template #default="{ row }">
            <span class="text-gray-600">{{ row.changelog || '-' }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="uploaded_by" label="上传者" width="100" />
        <el-table-column prop="created_at" label="上传时间" width="170">
          <template #default="{ row }">
            {{ formatDateTime(row.created_at) }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="240" fixed="right">
          <template #default="{ row }">
            <el-button size="small" type="primary" @click="openDeployDialog(row)">
              部署
            </el-button>
            <el-button size="small" type="warning" :disabled="row.is_current" @click="handleSetCurrent(row)">
              设为当前
            </el-button>
            <el-popconfirm title="确定删除此版本?" @confirm="handleDelete(row)">
              <template #reference>
                <el-button size="small" type="danger" :disabled="row.is_current">删除</el-button>
              </template>
            </el-popconfirm>
          </template>
        </el-table-column>
      </el-table>

      <div class="flex justify-end mt-4">
        <el-pagination
          v-model:current-page="pagination.page"
          v-model:page-size="pagination.pageSize"
          :total="pagination.total"
          :page-sizes="[10, 20, 50]"
          layout="total, sizes, prev, pager, next"
          @size-change="loadVersions"
          @current-change="loadVersions"
        />
      </div>
    </el-card>

    <!-- 部署历史 -->
    <el-card shadow="never">
      <template #header>
        <div class="flex items-center justify-between">
          <span class="font-bold">部署历史</span>
          <el-button size="small" @click="loadHistory">刷新</el-button>
        </div>
      </template>
      <el-table :data="history" v-loading="historyLoading" stripe border max-height="400">
        <el-table-column prop="server_name" label="服务器" width="150" />
        <el-table-column prop="service_name" label="服务" width="100">
          <template #default="{ row }">
            <el-tag :type="row.service_name === 'server' ? 'primary' : 'success'" size="small">
              {{ row.service_name }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="version" label="版本" width="120" />
        <el-table-column prop="action" label="操作" width="100">
          <template #default="{ row }">
            <el-tag :type="row.action === 'deploy' ? 'primary' : 'warning'" size="small">
              {{ row.action === 'deploy' ? '部署' : '回滚' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="status_text" label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="row.status === 1 ? 'success' : row.status === 2 ? 'danger' : 'info'" size="small">
              {{ row.status_text }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="operator" label="操作人" width="100" />
        <el-table-column prop="output" label="输出" min-width="200">
          <template #default="{ row }">
            <el-tooltip :content="row.output" placement="top" :disabled="!row.output">
              <span class="text-gray-600 text-sm">{{ truncate(row.output, 50) || '-' }}</span>
            </el-tooltip>
          </template>
        </el-table-column>
        <el-table-column prop="created_at" label="时间" width="170">
          <template #default="{ row }">
            {{ formatDateTime(row.created_at) }}
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- 上传版本对话框 -->
    <el-dialog v-model="showUploadDialog" title="上传新版本" width="500px" :close-on-click-modal="false">
      <el-form :model="uploadForm" label-width="100px">
        <el-form-item label="服务类型" required>
          <el-select v-model="uploadForm.service_name" placeholder="选择服务" style="width: 100%">
            <el-option label="server" value="server" />
            <el-option label="wukongim" value="wukongim" />
          </el-select>
        </el-form-item>
        <el-form-item label="版本号" required>
          <el-input v-model="uploadForm.version" placeholder="如: v1.0.0" />
        </el-form-item>
        <el-form-item label="更新日志">
          <el-input v-model="uploadForm.changelog" type="textarea" :rows="3" placeholder="描述此版本的更新内容" />
        </el-form-item>
        <el-form-item label="程序文件" required>
          <el-upload
            ref="uploadRef"
            :auto-upload="false"
            :limit="1"
            :on-change="handleFileChange"
            :on-remove="() => uploadForm.file = null"
          >
            <template #trigger>
              <el-button type="primary">选择文件</el-button>
            </template>
            <template #tip>
              <div class="el-upload__tip">请选择编译好的二进制文件</div>
            </template>
          </el-upload>
        </el-form-item>
        <el-form-item v-if="uploadProgress > 0">
          <el-progress :percentage="uploadProgress" :status="uploadProgress === 100 ? 'success' : undefined" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showUploadDialog = false">取消</el-button>
        <el-button type="primary" :loading="uploading" @click="handleUpload">上传</el-button>
      </template>
    </el-dialog>

    <!-- 部署对话框 -->
    <el-dialog v-model="showDeployDialog" title="部署版本" width="700px" :close-on-click-modal="false">
      <div class="mb-4">
        <el-alert type="info" :closable="false">
          <template #title>
            将 <strong>{{ deployVersion?.service_name }}</strong> 版本
            <strong>{{ deployVersion?.version }}</strong> 部署到选中的服务器
          </template>
        </el-alert>
      </div>

      <el-form :model="deployForm" label-width="100px">
        <el-form-item label="目标服务器" required>
          <el-select
            v-model="deployForm.server_ids"
            multiple
            filterable
            placeholder="选择目标服务器"
            style="width: 100%"
          >
            <el-option
              v-for="server in servers"
              :key="server.id"
              :label="`${server.name} (${server.host})`"
              :value="server.id"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="执行方式">
          <el-radio-group v-model="deployForm.parallel">
            <el-radio :label="true">并行执行</el-radio>
            <el-radio :label="false">顺序执行</el-radio>
          </el-radio-group>
        </el-form-item>
      </el-form>

      <!-- 部署结果 -->
      <div v-if="deployResults.length > 0" class="mt-4">
        <el-divider>部署结果</el-divider>
        <el-table :data="deployResults" size="small" stripe border max-height="300">
          <el-table-column prop="server_name" label="服务器" />
          <el-table-column prop="success" label="状态" width="100">
            <template #default="{ row }">
              <el-tag :type="row.success ? 'success' : 'danger'" size="small">
                {{ row.success ? '成功' : '失败' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="message" label="消息" />
          <el-table-column prop="duration" label="耗时" width="100">
            <template #default="{ row }">
              {{ row.duration }}ms
            </template>
          </el-table-column>
        </el-table>
      </div>

      <template #footer>
        <el-button @click="showDeployDialog = false">关闭</el-button>
        <el-button type="primary" :loading="deploying" :disabled="deployResults.length > 0" @click="handleDeploy">
          开始部署
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from "vue"
import { ElMessage } from "element-plus"
import { Upload, Refresh } from "@element-plus/icons-vue"
import * as deployApi from "@/common/apis/deploy"
import type * as Deploy from "@/common/apis/deploy/type"

// 状态
const loading = ref(false)
const historyLoading = ref(false)
const uploading = ref(false)
const deploying = ref(false)
const uploadProgress = ref(0)

// 数据
const versions = ref<Deploy.VersionInfo[]>([])
const history = ref<Deploy.DeploymentRecord[]>([])
const servers = ref<Deploy.ServerResp[]>([])

// 过滤
const filterService = ref("")

// 分页
const pagination = reactive({
  page: 1,
  pageSize: 10,
  total: 0
})

// 上传表单
const showUploadDialog = ref(false)
const uploadRef = ref()
const uploadForm = reactive({
  service_name: "server",
  version: "",
  changelog: "",
  file: null as File | null
})

// 部署表单
const showDeployDialog = ref(false)
const deployVersion = ref<Deploy.VersionInfo | null>(null)
const deployForm = reactive({
  server_ids: [] as number[],
  parallel: true
})
const deployResults = ref<Deploy.DeployResult[]>([])

// 加载版本列表
async function loadVersions() {
  loading.value = true
  try {
    const res = await deployApi.listVersions({
      service_name: filterService.value || undefined,
      page: pagination.page,
      page_size: pagination.pageSize
    }) as any
    versions.value = res.data?.list || []
    pagination.total = res.data?.total || 0
  } finally {
    loading.value = false
  }
}

// 加载部署历史
async function loadHistory() {
  historyLoading.value = true
  try {
    const res = await deployApi.getDeploymentHistory({
      page: 1,
      page_size: 20
    }) as any
    history.value = res.data?.list || []
  } finally {
    historyLoading.value = false
  }
}

// 加载服务器列表
async function loadServers() {
  try {
    const res = await deployApi.getServerList({ page: 1, size: 100, server_type: 1 }) as any
    servers.value = res.data?.list || []
  } catch (e) {
    console.error("加载服务器列表失败", e)
  }
}

// 文件选择
function handleFileChange(file: any) {
  uploadForm.file = file.raw
}

// 上传版本
async function handleUpload() {
  if (!uploadForm.service_name || !uploadForm.version || !uploadForm.file) {
    ElMessage.warning("请填写完整信息")
    return
  }

  const form = new FormData()
  form.append("service_name", uploadForm.service_name)
  form.append("version", uploadForm.version)
  form.append("changelog", uploadForm.changelog)
  form.append("file", uploadForm.file)

  uploading.value = true
  uploadProgress.value = 0

  try {
    await deployApi.uploadVersion(form, (percent) => {
      uploadProgress.value = percent
    })
    ElMessage.success("上传成功")
    showUploadDialog.value = false
    uploadForm.version = ""
    uploadForm.changelog = ""
    uploadForm.file = null
    uploadProgress.value = 0
    loadVersions()
  } catch (e: any) {
    ElMessage.error(e.message || "上传失败")
  } finally {
    uploading.value = false
  }
}

// 设为当前版本
async function handleSetCurrent(row: Deploy.VersionInfo) {
  try {
    await deployApi.setCurrentVersion(row.id)
    ElMessage.success("设置成功")
    loadVersions()
  } catch (e: any) {
    ElMessage.error(e.message || "设置失败")
  }
}

// 删除版本
async function handleDelete(row: Deploy.VersionInfo) {
  try {
    await deployApi.deleteVersion(row.id)
    ElMessage.success("删除成功")
    loadVersions()
  } catch (e: any) {
    ElMessage.error(e.message || "删除失败")
  }
}

// 打开部署对话框
function openDeployDialog(row: Deploy.VersionInfo) {
  deployVersion.value = row
  deployForm.server_ids = []
  deployForm.parallel = true
  deployResults.value = []
  showDeployDialog.value = true
}

// 执行部署
async function handleDeploy() {
  if (deployForm.server_ids.length === 0) {
    ElMessage.warning("请选择目标服务器")
    return
  }

  deploying.value = true
  try {
    const res = await deployApi.deployVersion({
      version_id: deployVersion.value!.id,
      server_ids: deployForm.server_ids,
      parallel: deployForm.parallel
    }) as any
    deployResults.value = res.data?.results || []
    ElMessage.success(`部署完成: ${res.data?.success || 0} 成功, ${res.data?.failed || 0} 失败`)
    loadHistory()
  } catch (e: any) {
    ElMessage.error(e.message || "部署失败")
  } finally {
    deploying.value = false
  }
}

// 格式化文件大小
function formatFileSize(bytes: number): string {
  if (bytes < 1024) return bytes + " B"
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + " KB"
  return (bytes / 1024 / 1024).toFixed(2) + " MB"
}

// 格式化日期时间
function formatDateTime(dateStr: string): string {
  if (!dateStr) return "-"
  const d = new Date(dateStr)
  return d.toLocaleString("zh-CN", { hour12: false })
}

// 截断文本
function truncate(str: string, len: number): string {
  if (!str) return ""
  return str.length > len ? str.substring(0, len) + "..." : str
}

onMounted(() => {
  loadVersions()
  loadHistory()
  loadServers()
})
</script>

<style scoped>
.app-container {
  padding: 20px;
}
.mb-4 {
  margin-bottom: 16px;
}
.mt-4 {
  margin-top: 16px;
}
.mr-1 {
  margin-right: 4px;
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
  justify-end: flex-end;
}
.gap-2 {
  gap: 8px;
}
.gap-4 {
  gap: 16px;
}
.font-bold {
  font-weight: bold;
}
.font-mono {
  font-family: monospace;
}
.text-xs {
  font-size: 12px;
}
.text-sm {
  font-size: 14px;
}
.text-gray-500 {
  color: #909399;
}
.text-gray-600 {
  color: #606266;
}
</style>
