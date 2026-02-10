<script lang="ts" setup>
import type { AppLogListItem } from "@@/apis/app_logs/type"
import type { MerchantResp } from "@/common/apis/merchant/type"
import { queryAppLogs } from "@@/apis/app_logs"
import { getMerchantList } from "@/common/apis/merchant"
import { ElMessage } from "element-plus"
import { onMounted, ref } from "vue"

defineOptions({
  name: "AppLogs"
})

// 商户列表
const merchants = ref<MerchantResp[]>([])
const selectedMerchantNo = ref("")
const loadingMerchants = ref(false)

// 日志列表
const logs = ref<AppLogListItem[]>([])
const total = ref(0)
const page = ref(1)
const size = ref(20)
const keyword = ref("")
const loadingLogs = ref(false)

onMounted(async () => {
  await loadMerchants()
})

async function loadMerchants() {
  loadingMerchants.value = true
  try {
    const { data } = await getMerchantList({ page: 1, size: 500 })
    merchants.value = data.list || []
  } catch {
    ElMessage.error("获取商户列表失败")
  } finally {
    loadingMerchants.value = false
  }
}

async function loadLogs() {
  if (!selectedMerchantNo.value) {
    ElMessage.warning("请先选择商户")
    return
  }
  loadingLogs.value = true
  try {
    const { data } = await queryAppLogs({
      merchant_no: selectedMerchantNo.value,
      page: page.value,
      size: size.value,
      keyword: keyword.value || undefined
    })
    logs.value = data.list || []
    total.value = data.total || 0
  } catch (e: any) {
    ElMessage.error(e?.message || "获取日志列表失败")
  } finally {
    loadingLogs.value = false
  }
}

function onMerchantChange() {
  page.value = 1
  loadLogs()
}

function onSearch() {
  page.value = 1
  loadLogs()
}

function onPageChange(newPage: number) {
  page.value = newPage
  loadLogs()
}

function onSizeChange(newSize: number) {
  size.value = newSize
  page.value = 1
  loadLogs()
}

function formatFileSize(bytes: number): string {
  if (bytes === 0) return "0 B"
  const k = 1024
  const sizes = ["B", "KB", "MB", "GB"]
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + " " + sizes[i]
}

function getLogTypeLabel(logType: string): string {
  switch (logType) {
    case "log":
      return "普通日志"
    case "crash":
      return "崩溃日志"
    default:
      return logType
  }
}

function getLogTypeTagType(logType: string): "success" | "danger" | "info" {
  switch (logType) {
    case "log":
      return "success"
    case "crash":
      return "danger"
    default:
      return "info"
  }
}

function downloadLog(url: string) {
  if (url) {
    window.open(url, "_blank")
  }
}
</script>

<template>
  <div class="app-container">
    <el-card>
      <template #header>
        <div class="flex items-center justify-between">
          <div class="font-600">应用日志</div>
        </div>
      </template>

      <!-- 筛选区域 -->
      <div class="filter-area mb-4">
        <el-form :inline="true">
          <el-form-item label="商户">
            <el-select
              v-model="selectedMerchantNo"
              filterable
              placeholder="请选择商户"
              :loading="loadingMerchants"
              style="width: 280px;"
              @change="onMerchantChange"
            >
              <el-option
                v-for="m in merchants"
                :key="m.id"
                :label="`${m.name || '未命名'} (${m.no})`"
                :value="m.no"
              />
            </el-select>
          </el-form-item>
          <el-form-item label="搜索">
            <el-input
              v-model="keyword"
              placeholder="用户名/手机号/短号"
              clearable
              style="width: 200px;"
              @keyup.enter="onSearch"
            />
          </el-form-item>
          <el-form-item>
            <el-button type="primary" :disabled="!selectedMerchantNo" @click="onSearch">查询</el-button>
          </el-form-item>
        </el-form>
      </div>

      <!-- 日志列表 -->
      <el-table :data="logs" v-loading="loadingLogs" stripe border>
        <el-table-column label="用户信息" min-width="200">
          <template #default="{ row }">
            <div class="user-info">
              <div class="name">{{ row.name || "-" }}</div>
              <div class="meta">
                <span v-if="row.phone">手机: {{ row.phone }}</span>
                <span v-if="row.short_no" class="ml-2">短号: {{ row.short_no }}</span>
              </div>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="日志类型" width="120" align="center">
          <template #default="{ row }">
            <el-tag :type="getLogTypeTagType(row.log_type)" size="small">
              {{ getLogTypeLabel(row.log_type) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="日志日期" prop="log_date" width="120" align="center" />
        <el-table-column label="文件名" prop="file_name" min-width="200" show-overflow-tooltip />
        <el-table-column label="文件大小" width="100" align="center">
          <template #default="{ row }">
            {{ formatFileSize(row.file_size) }}
          </template>
        </el-table-column>
        <el-table-column label="设备信息" prop="device_info" min-width="180" show-overflow-tooltip />
        <el-table-column label="App版本" prop="app_version" width="100" align="center" />
        <el-table-column label="上传时间" prop="created_at" width="180" />
        <el-table-column label="操作" width="100" fixed="right" align="center">
          <template #default="{ row }">
            <el-button
              v-if="row.minio_path"
              type="primary"
              link
              size="small"
              @click="downloadLog(row.minio_path)"
            >
              下载
            </el-button>
            <span v-else>-</span>
          </template>
        </el-table-column>
      </el-table>

      <!-- 分页 -->
      <div class="mt-4 flex justify-end">
        <el-pagination
          v-model:current-page="page"
          v-model:page-size="size"
          :page-sizes="[10, 20, 50, 100]"
          layout="total, sizes, prev, pager, next, jumper"
          :total="total"
          @current-change="onPageChange"
          @size-change="onSizeChange"
        />
      </div>
    </el-card>
  </div>
</template>

<style lang="scss" scoped>
.app-container {
  padding: 20px;
}

.filter-area {
  background: #f5f7fa;
  padding: 16px;
  border-radius: 4px;
}

.user-info {
  .name {
    font-weight: 500;
    color: #303133;
  }
  .meta {
    font-size: 12px;
    color: #909399;
    margin-top: 4px;
  }
}
</style>
