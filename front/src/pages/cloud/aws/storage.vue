<script lang="ts" setup>
import type { AwsS3ObjectItem } from "@@/apis/aws/type"
import { downloadObject, getBillingCostUsage, listBuckets, listObjects, setBucketPublic, uploadObject } from "@@/apis/aws"
import { awsDeleteObject, awsCreateBucket, awsDeleteBucket } from "@@/apis/cloud_storage"
import { getCloudAccountOptions } from "@@/apis/cloud_account"
import type { CloudAccountOption } from "@@/apis/cloud_account/type"
import { getMerchantList } from "@@/apis/merchant"
import { formatFileSize, generateObjectUrl } from "@@/apis/cloud_storage"
import CloudStorageUploadDialog from "@@/components/CloudStorageUploadDialog.vue"
import { getAwsRegions } from "@@/constants/aws-regions"
import { ArrowDown, CopyDocument, Delete, Document, Download, FolderOpened, Lock, Plus, Refresh, Search, Select, TrendCharts, Unlock, Upload } from "@element-plus/icons-vue"

defineOptions({ name: "CloudAwsStorage" })

// ========== 状态管理 ==========
interface BucketInfo { name: string; location: string }
const loading = ref(false)
const accountType = ref<string>("system")
const buckets = ref<BucketInfo[]>([])
const objects = ref<AwsS3ObjectItem[]>([])
const cloudAccounts = ref<CloudAccountOption[]>([])
const selectedCloudAccount = ref<number>()
const selectedMerchant = ref<number>()
const merchantList = ref<{ value: number, label: string }[]>([])
const awsRegions = getAwsRegions("cn")
const uploadDialogVisible = ref(false)

const form = reactive({
  cloud_account_id: 0,
  merchant_id: 0,
  region_id: "",
  bucket: "",
  prefix: ""
})

// ========== 创建 Bucket ==========
const createBucketDialog = ref(false)
const createBucketForm = reactive({
  bucket: ""
})

// ========== 账单相关 ==========
const billing = ref<any>(null)
const billingDialog = ref(false)
const billingLoading = ref(false)
const billingForm = reactive({
  start: "",
  end: "",
  granularity: "DAILY",
  group_by_key: "SERVICE"
})

// ========== 初始化 ==========
onMounted(async () => {
  await Promise.all([fetchCloudAccounts(), fetchMerchantList()])
})

async function fetchCloudAccounts() {
  try {
    const res = await getCloudAccountOptions("aws")
    cloudAccounts.value = res.data || []
  } catch {
    console.error("获取云账号失败")
  }
}

async function fetchMerchantList() {
  try {
    const res = await getMerchantList({ page: 1, size: 1000 })
    merchantList.value = (res.data.list || []).map((m: any) => ({
      label: `${m.name} (${m.no})`,
      value: m.id
    }))
  } catch {
    console.error("获取商户列表失败")
  }
}

// ========== 账号类型切换 ==========
function handleAccountTypeChange() {
  selectedCloudAccount.value = undefined
  selectedMerchant.value = undefined
  form.region_id = ""
  form.bucket = ""
  buckets.value = []
  objects.value = []
  if (accountType.value === "system") {
    fetchCloudAccounts()
  }
}

// ========== 操作步骤提示 ==========
const currentStep = computed(() => {
  if (accountType.value === "system" && !selectedCloudAccount.value) return 0
  if (accountType.value === "merchant" && !selectedMerchant.value) return 0
  if (!form.bucket) return 1
  return 2
})

// ========== Bucket 操作 ==========
async function onLoadBuckets() {
  if (accountType.value === "system" && !selectedCloudAccount.value) {
    ElMessage.warning("请选择云账号")
    return
  }
  if (accountType.value === "merchant" && !selectedMerchant.value) {
    ElMessage.warning("请选择商户")
    return
  }

  form.cloud_account_id = selectedCloudAccount.value || 0
  form.merchant_id = selectedMerchant.value || 0

  loading.value = true
  try {
    const params: any = { region_id: form.region_id }
    if (accountType.value === "merchant") params.merchant_id = selectedMerchant.value
    else params.cloud_account_id = selectedCloudAccount.value
    const { data } = await listBuckets(params)
    buckets.value = data.list || []
    ElMessage.success(`加载成功，共 ${buckets.value.length} 个 Bucket`)
  } catch {
    buckets.value = []
  } finally {
    loading.value = false
  }
}

// 当选择 Bucket 时自动加载对象
watch(() => form.bucket, (newBucket) => {
  if (newBucket) {
    onListObjects()
  } else {
    objects.value = []
  }
})

// ========== 对象操作 ==========
async function onListObjects() {
  if (!form.bucket) {
    ElMessage.warning("请选择 Bucket")
    return
  }
  loading.value = true
  try {
    const params: any = {
      region_id: form.region_id,
      bucket: form.bucket,
      prefix: form.prefix || undefined
    }
    if (accountType.value === "merchant") {
      params.merchant_id = selectedMerchant.value
    } else {
      params.cloud_account_id = selectedCloudAccount.value
    }
    const { data } = await listObjects(params)
    objects.value = data.list || []
  } finally {
    loading.value = false
  }
}

async function onRefresh() {
  if (form.bucket) {
    await onListObjects()
    ElMessage.success("刷新成功")
  } else {
    ElMessage.warning("请先选择 Bucket")
  }
}

// ========== 上传操作 ==========
function onShowUpload() {
  if (!form.bucket) {
    ElMessage.warning("请先选择 Bucket")
    return
  }
  uploadDialogVisible.value = true
}

function handleUpload(formData: FormData, onUploadProgress?: (progressEvent: any) => void) {
  formData.append("region_id", form.region_id || "")
  if (accountType.value === "merchant") {
    formData.append("merchant_id", String(selectedMerchant.value || ""))
  } else {
    formData.append("cloud_account_id", String(selectedCloudAccount.value || ""))
  }
  return uploadObject(formData, onUploadProgress)
}

function onUploadSuccess() {
  onListObjects()
}

// ========== 下载操作 ==========
async function onDownload(row: AwsS3ObjectItem) {
  loading.value = true
  try {
    const params: any = {
      region_id: form.region_id,
      bucket: form.bucket,
      object_key: row.key,
      filename: row.key,
      attachment: 1
    }
    if (accountType.value === "merchant") {
      params.merchant_id = selectedMerchant.value
    } else {
      params.cloud_account_id = selectedCloudAccount.value
    }
    const blob = await downloadObject(params)
    const url = URL.createObjectURL(blob as any)
    const a = document.createElement("a")
    a.href = url
    a.download = row.key.split("/").pop() || row.key
    a.click()
    URL.revokeObjectURL(url)
    ElMessage.success("下载成功")
  } finally {
    loading.value = false
  }
}

// ========== 删除操作 ==========
async function onDelete(row: AwsS3ObjectItem) {
  await ElMessageBox.confirm(`确定要删除 "${row.key}" 吗？此操作不可恢复！`, "警告", {
    confirmButtonText: "确定",
    cancelButtonText: "取消",
    type: "warning"
  })
  loading.value = true
  try {
    const params: any = {
      region_id: form.region_id || "",
      bucket: form.bucket,
      object_key: row.key
    }
    if (accountType.value === "merchant") {
      params.merchant_id = selectedMerchant.value
    } else {
      params.cloud_account_id = selectedCloudAccount.value
    }
    await awsDeleteObject(params)
    ElMessage.success("删除成功")
    onListObjects()
  } finally {
    loading.value = false
  }
}

// ========== 复制链接 ==========
async function onCopyUrl(row: AwsS3ObjectItem) {
  try {
    const url = generateObjectUrl("aws", form.bucket, row.key, form.region_id)
    await navigator.clipboard.writeText(url)
    ElMessage.success({ message: "链接已复制到剪贴板", duration: 2000 })
  } catch {
    try {
      const textarea = document.createElement("textarea")
      const url = generateObjectUrl("aws", form.bucket, row.key, form.region_id)
      textarea.value = url
      textarea.style.position = "fixed"
      textarea.style.opacity = "0"
      document.body.appendChild(textarea)
      textarea.select()
      document.execCommand("copy")
      document.body.removeChild(textarea)
      ElMessage.success({ message: "链接已复制到剪贴板", duration: 2000 })
    } catch {
      ElMessage.error("复制失败，请手动复制")
    }
  }
}

// ========== 设置 Bucket 公开访问 ==========
async function onSetBucketPublic(command: string) {
  if (!form.bucket) {
    ElMessage.warning("请先选择 Bucket")
    return
  }

  const isPublic = command === "public"
  const action = isPublic ? "设置为公开" : "设置为私有"
  const message = isPublic
    ? "设置为公开后，所有人都可以通过链接访问 Bucket 中的对象，确认继续吗？"
    : "设置为私有后，需要通过 AWS 凭证才能访问对象，确认继续吗？"

  try {
    await ElMessageBox.confirm(message, `${action}访问`, {
      confirmButtonText: "确认",
      cancelButtonText: "取消",
      type: "warning"
    })

    loading.value = true
    const params: any = {
      region_id: form.region_id,
      bucket: form.bucket,
      public: isPublic
    }
    if (accountType.value === "merchant") {
      params.merchant_id = selectedMerchant.value
    } else {
      params.cloud_account_id = selectedCloudAccount.value
    }
    await setBucketPublic(params)

    ElMessage.success(`已${action}成功`)
  } catch (error: any) {
    if (error !== "cancel") {
      ElMessage.error(`${action}失败`)
    }
  } finally {
    loading.value = false
  }
}

// ========== 创建 Bucket ==========
function onShowCreateBucket() {
  if (accountType.value === "system" && !selectedCloudAccount.value) {
    ElMessage.warning("请先选择云账号")
    return
  }
  if (accountType.value === "merchant" && !selectedMerchant.value) {
    ElMessage.warning("请先选择商户")
    return
  }
  if (!form.region_id) {
    ElMessage.warning("请先选择 Region")
    return
  }
  createBucketForm.bucket = ""
  createBucketDialog.value = true
}

async function onCreateBucket() {
  if (!createBucketForm.bucket.trim()) {
    ElMessage.warning("请输入 Bucket 名称")
    return
  }
  loading.value = true
  try {
    const params: any = {
      region_id: form.region_id || "",
      bucket: createBucketForm.bucket.trim()
    }
    if (accountType.value === "merchant") {
      params.merchant_id = selectedMerchant.value
    } else {
      params.cloud_account_id = selectedCloudAccount.value
    }
    await awsCreateBucket(params)
    ElMessage.success("创建成功")
    createBucketDialog.value = false
    onLoadBuckets()
  } finally {
    loading.value = false
  }
}

// ========== 删除 Bucket ==========
async function onDeleteBucket(bucket: string) {
  await ElMessageBox.confirm(
    `确定要删除 Bucket "${bucket}" 吗？此操作不可恢复！\n注意：Bucket 必须为空才能删除。`,
    "警告",
    {
      confirmButtonText: "确定删除",
      cancelButtonText: "取消",
      type: "warning"
    }
  )
  loading.value = true
  try {
    const params: any = {
      region_id: form.region_id || "",
      bucket
    }
    if (accountType.value === "merchant") {
      params.merchant_id = selectedMerchant.value
    } else {
      params.cloud_account_id = selectedCloudAccount.value
    }
    await awsDeleteBucket(params)
    ElMessage.success("删除成功")
    if (form.bucket === bucket) {
      form.bucket = ""
      objects.value = []
    }
    onLoadBuckets()
  } finally {
    loading.value = false
  }
}

// ========== 账单查询 ==========
async function onQueryBilling() {
  if (!selectedCloudAccount.value) {
    ElMessage.warning("请选择云账号")
    return
  }
  if (!form.region_id) {
    ElMessage.warning("请选择 Region")
    return
  }
  if (!billingForm.start || !billingForm.end) {
    ElMessage.warning("请选择起止日期")
    return
  }
  billingDialog.value = true
  billingLoading.value = true
  try {
    const { data } = await getBillingCostUsage({
      cloud_account_id: selectedCloudAccount.value,
      region_id: form.region_id,
      start: billingForm.start,
      end: billingForm.end,
      granularity: billingForm.granularity as any,
      group_by_key: billingForm.group_by_key
    })
    billing.value = data
  } finally {
    billingLoading.value = false
  }
}

</script>

<template>
  <div class="app-container">
    <!-- 页面标题 -->
    <div class="page-header">
      <div class="header-left">
        <h2 class="title">AWS S3 对象存储</h2>
        <p class="subtitle">管理您的 S3 Buckets 和对象文件</p>
      </div>
    </div>

    <!-- 步骤指示器 -->
    <el-steps :active="currentStep" align-center class="steps-indicator">
      <el-step title="选择账号" description="选择云账号或商户" />
      <el-step title="选择 Bucket" description="加载并选择存储桶" />
      <el-step title="管理对象" description="上传、下载、浏览文件" />
    </el-steps>

    <!-- 账号和区域选择 -->
    <el-card shadow="hover" class="mb-4">
      <template #header>
        <div class="card-header">
          <span class="card-title">
            <el-icon><FolderOpened /></el-icon>
            基础配置
          </span>
        </div>
      </template>

      <div class="config-grid">
        <el-form label-position="top">
          <el-row :gutter="20">
            <el-col :xs="24" :sm="12" :md="6">
              <el-form-item label="账号类型">
                <el-select v-model="accountType" style="width: 100%" @change="handleAccountTypeChange">
                  <el-option label="系统类型" value="system" />
                  <el-option label="商户类型" value="merchant" />
                </el-select>
              </el-form-item>
            </el-col>

            <el-col v-if="accountType === 'system'" :xs="24" :sm="12" :md="6">
              <el-form-item label="云账号">
                <el-select
                  v-model="selectedCloudAccount"
                  placeholder="请选择云账号"
                  filterable
                  clearable
                  style="width: 100%"
                  @change="() => { form.bucket = ''; buckets = []; objects = [] }"
                >
                  <el-option
                    v-for="opt in cloudAccounts"
                    :key="opt.value"
                    :label="opt.label"
                    :value="opt.value"
                  />
                </el-select>
              </el-form-item>
            </el-col>

            <el-col v-else :xs="24" :sm="12" :md="6">
              <el-form-item label="商户">
                <el-select
                  v-model="selectedMerchant"
                  placeholder="请选择商户"
                  filterable
                  clearable
                  style="width: 100%"
                  @change="() => { form.bucket = ''; buckets = []; objects = [] }"
                >
                  <el-option
                    v-for="m in merchantList"
                    :key="m.value"
                    :label="m.label"
                    :value="m.value"
                  />
                </el-select>
              </el-form-item>
            </el-col>

            <el-col :xs="24" :sm="12" :md="6">
              <el-form-item label="Region（可选）">
                <el-select
                  v-model="form.region_id"
                  placeholder="选择 Region"
                  filterable
                  clearable
                  style="width: 100%"
                >
                  <el-option
                    v-for="r in awsRegions"
                    :key="r.id"
                    :label="`${r.name} (${r.id})`"
                    :value="r.id"
                  />
                </el-select>
              </el-form-item>
            </el-col>

            <el-col :xs="24" :sm="12" :md="6">
              <el-form-item label="操作">
                <el-button
                  type="primary"
                  :icon="Refresh"
                  :loading="loading"
                  :disabled="accountType === 'system' ? !selectedCloudAccount : !selectedMerchant"
                  @click="onLoadBuckets"
                  style="width: 100%"
                >
                  加载 Buckets
                </el-button>
              </el-form-item>
            </el-col>
          </el-row>
        </el-form>
      </div>
    </el-card>

    <!-- Bucket 列表展示 -->
    <el-card v-if="buckets.length || (accountType === 'system' ? selectedCloudAccount : selectedMerchant)" shadow="hover" class="mb-4 bucket-list-card">
      <template #header>
        <div class="card-header">
          <span class="card-title">
            <el-icon><FolderOpened /></el-icon>
            存储桶列表
            <el-tag type="info" size="small" style="margin-left: 8px">{{ buckets.length }}</el-tag>
          </span>
          <div class="card-actions">
            <el-button type="primary" :icon="Plus" size="small" @click="onShowCreateBucket">
              创建 Bucket
            </el-button>
          </div>
        </div>
      </template>

      <div v-if="buckets.length" class="bucket-grid">
        <div
          v-for="bucket in buckets"
          :key="bucket.name"
          class="bucket-item"
          :class="{ active: form.bucket === bucket.name }"
          @click="form.bucket = bucket.name"
        >
          <div class="bucket-icon">
            <el-icon :size="24"><FolderOpened /></el-icon>
          </div>
          <div class="bucket-name">{{ bucket.name }}</div>
          <div class="bucket-location">{{ bucket.location }}</div>
          <div v-if="form.bucket === bucket.name" class="bucket-selected">
            <el-icon color="#67c23a"><Select /></el-icon>
          </div>
          <el-button
            class="bucket-delete"
            type="danger"
            :icon="Delete"
            size="small"
            circle
            @click.stop="onDeleteBucket(bucket.name)"
          />
        </div>
      </div>
      <el-empty v-else description="暂无存储桶，请点击上方按钮创建" :image-size="80" />
    </el-card>

    <!-- Bucket 和对象管理 -->
    <el-card shadow="hover">
      <template #header>
        <div class="card-header">
          <span class="card-title">
            <el-icon><Document /></el-icon>
            对象管理
            <el-tag v-if="form.bucket" type="success" size="small" style="margin-left: 8px">{{ form.bucket }}</el-tag>
          </span>
          <div class="card-actions">
            <el-dropdown v-if="form.bucket" trigger="click" @command="onSetBucketPublic">
              <el-button size="small">
                Bucket 权限设置
                <el-icon class="el-icon--right"><ArrowDown /></el-icon>
              </el-button>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item command="public">
                    <el-icon><Unlock /></el-icon>
                    设置为公开
                  </el-dropdown-item>
                  <el-dropdown-item command="private">
                    <el-icon><Lock /></el-icon>
                    设置为私有
                  </el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
            <el-tooltip content="刷新列表">
              <el-button :icon="Refresh" circle size="small" @click="onRefresh" :disabled="!form.bucket" />
            </el-tooltip>
            <el-button type="primary" :icon="Upload" size="small" @click="onShowUpload" :disabled="!form.bucket">
              上传文件
            </el-button>
          </div>
        </div>
      </template>

      <!-- 搜索栏（简化为只有前缀和搜索按钮） -->
      <div v-if="form.bucket" class="filter-bar">
        <el-row :gutter="12">
          <el-col :xs="24" :sm="16" :md="18">
            <el-input
              v-model="form.prefix"
              placeholder="对象前缀（可选，例如: folder/subfolder/）"
              clearable
              @keyup.enter="onListObjects"
            >
              <template #prefix>
                <el-icon><Search /></el-icon>
              </template>
            </el-input>
          </el-col>
          <el-col :xs="24" :sm="8" :md="6">
            <el-button type="primary" :icon="Search" @click="onListObjects" style="width: 100%">
              搜索对象
            </el-button>
          </el-col>
        </el-row>
      </div>

      <!-- 未选择 Bucket 的提示 -->
      <el-empty v-if="!form.bucket && buckets.length" description="请在上方选择一个存储桶" :image-size="120" />

      <!-- 对象列表 -->
      <div class="table-container">
        <el-table
          :data="objects"
          v-loading="loading"
          height="480"
          stripe
          :empty-text="form.bucket ? '暂无数据' : '请先选择 Bucket'"
        >
          <el-table-column type="index" label="#" width="60" />
          <el-table-column prop="key" label="对象键" min-width="300" show-overflow-tooltip>
            <template #default="{ row }">
              <div class="object-key">
                <el-icon color="#409eff"><Document /></el-icon>
                <span>{{ row.key }}</span>
              </div>
            </template>
          </el-table-column>
          <el-table-column prop="size" label="大小" width="120" align="right">
            <template #default="{ row }">
              <el-tag size="small" type="info">{{ formatFileSize(row.size) }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="storage_class" label="存储类型" width="140" align="center">
            <template #default="{ row }">
              <el-tag size="small">{{ row.storage_class }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="last_modified" label="最后修改时间" width="180" />
          <el-table-column label="操作" width="180" fixed="right" align="center">
            <template #default="{ row }">
              <el-button link type="primary" :icon="CopyDocument" @click="onCopyUrl(row)">
                复制链接
              </el-button>
              <el-button link type="success" :icon="Download" @click="onDownload(row)">
                下载
              </el-button>
              <el-button link type="danger" :icon="Delete" @click="onDelete(row)">
                删除
              </el-button>
            </template>
          </el-table-column>
        </el-table>
      </div>

      <!-- 统计信息 -->
      <div v-if="objects.length" class="stats-footer">
        <el-text type="info">
          共 {{ objects.length }} 个对象
        </el-text>
      </div>
    </el-card>

    <!-- 账单查询卡片 -->
    <el-card shadow="hover" class="mt-4 billing-card">
      <template #header>
        <div class="card-header">
          <span class="card-title">
            <el-icon><TrendCharts /></el-icon>
            成本分析（Cost Explorer）
          </span>
        </div>
      </template>

      <el-form label-position="top">
        <el-row :gutter="12">
          <el-col :xs="24" :sm="12" :md="6">
            <el-form-item label="开始日期">
              <el-date-picker
                v-model="billingForm.start"
                type="date"
                placeholder="选择日期"
                value-format="YYYY-MM-DD"
                style="width: 100%"
              />
            </el-form-item>
          </el-col>
          <el-col :xs="24" :sm="12" :md="6">
            <el-form-item label="结束日期">
              <el-date-picker
                v-model="billingForm.end"
                type="date"
                placeholder="选择日期"
                value-format="YYYY-MM-DD"
                style="width: 100%"
              />
            </el-form-item>
          </el-col>
          <el-col :xs="24" :sm="12" :md="5">
            <el-form-item label="时间粒度">
              <el-select v-model="billingForm.granularity" style="width: 100%">
                <el-option label="按天" value="DAILY" />
                <el-option label="按月" value="MONTHLY" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :xs="24" :sm="12" :md="5">
            <el-form-item label="分组方式">
              <el-select v-model="billingForm.group_by_key" style="width: 100%">
                <el-option label="按服务" value="SERVICE" />
                <el-option label="按账号" value="LINKED_ACCOUNT" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :xs="24" :sm="12" :md="2">
            <el-form-item label=" ">
              <el-button
                type="primary"
                :icon="Search"
                :loading="billingLoading"
                @click="onQueryBilling"
                style="width: 100%"
              >
                查询
              </el-button>
            </el-form-item>
          </el-col>
        </el-row>
      </el-form>
    </el-card>

    <!-- 上传对话框 -->
    <CloudStorageUploadDialog
      v-model="uploadDialogVisible"
      cloud-type="aws"
      :bucket="form.bucket"
      :prefix="form.prefix"
      :upload-fn="handleUpload"
      @success="onUploadSuccess"
    />

    <!-- 账单详情对话框 -->
    <el-dialog v-model="billingDialog" title="AWS 成本详情" width="800px">
      <div class="billing-content">
        <el-skeleton v-if="billingLoading" :rows="8" animated />
        <el-scrollbar v-else height="500px">
          <pre class="billing-json">{{ JSON.stringify(billing, null, 2) }}</pre>
        </el-scrollbar>
      </div>
      <template #footer>
        <el-button type="primary" @click="billingDialog = false">关闭</el-button>
      </template>
    </el-dialog>

    <!-- 创建 Bucket 对话框 -->
    <el-dialog v-model="createBucketDialog" title="创建 Bucket" width="500px">
      <el-form label-position="top">
        <el-form-item label="Bucket 名称" required>
          <el-input
            v-model="createBucketForm.bucket"
            placeholder="请输入 Bucket 名称（全局唯一）"
            @keyup.enter="onCreateBucket"
          />
        </el-form-item>
        <el-alert type="info" :closable="false" style="margin-top: -8px">
          <template #default>
            <div style="font-size: 12px; line-height: 1.6">
              <p style="margin: 0">Bucket 名称要求：</p>
              <ul style="margin: 4px 0 0 16px; padding: 0">
                <li>必须在 AWS 全局唯一</li>
                <li>长度 3-63 个字符</li>
                <li>只能包含小写字母、数字和连字符</li>
                <li>必须以字母或数字开头和结尾</li>
              </ul>
            </div>
          </template>
        </el-alert>
      </el-form>
      <template #footer>
        <el-button @click="createBucketDialog = false">取消</el-button>
        <el-button type="primary" :loading="loading" @click="onCreateBucket">创建</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style lang="scss" scoped>
.app-container {
  padding: 20px;
  background-color: #f5f7fa;
  min-height: 100vh;
}

// 页面标题
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24px;

  .header-left {
    .title {
      font-size: 24px;
      font-weight: 600;
      color: #303133;
      margin: 0 0 8px 0;
    }

    .subtitle {
      font-size: 14px;
      color: #909399;
      margin: 0;
    }
  }
}

// 步骤指示器
.steps-indicator {
  margin-bottom: 24px;
  padding: 20px;
  background: white;
  border-radius: 8px;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.05);
}

// 卡片头部
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;

  .card-title {
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 16px;
    font-weight: 600;
    color: #303133;
  }

  .card-actions {
    display: flex;
    gap: 8px;
  }
}

// 配置网格
.config-grid {
  :deep(.el-form-item__label) {
    font-weight: 500;
  }
}

// 过滤栏
.filter-bar {
  margin-bottom: 16px;
}

// 表格容器
.table-container {
  .object-key {
    display: flex;
    align-items: center;
    gap: 8px;
  }
}

// 统计信息底部
.stats-footer {
  margin-top: 12px;
  padding-top: 12px;
  border-top: 1px solid #ebeef5;
  text-align: right;
}

// Bucket 列表卡片
.bucket-list-card {
  .bucket-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
    gap: 16px;
  }

  .bucket-item {
    position: relative;
    padding: 20px;
    background: #f5f7fa;
    border: 2px solid #e4e7ed;
    border-radius: 8px;
    cursor: pointer;
    transition: all 0.3s;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 12px;

    &:hover {
      background: #ecf5ff;
      border-color: #409eff;
      transform: translateY(-2px);
      box-shadow: 0 4px 8px rgba(64, 158, 255, 0.2);
    }

    &.active {
      background: #ecf5ff;
      border-color: #409eff;
      box-shadow: 0 4px 12px rgba(64, 158, 255, 0.3);

      .bucket-icon {
        color: #409eff;
      }

      .bucket-name {
        color: #409eff;
        font-weight: 600;
      }
    }

    .bucket-icon {
      color: #909399;
      transition: color 0.3s;
    }

    .bucket-name {
      font-size: 14px;
      color: #303133;
      text-align: center;
      word-break: break-all;
      line-height: 1.5;
    }

    .bucket-location {
      font-size: 12px;
      color: #909399;
      text-align: center;
    }

    .bucket-selected {
      position: absolute;
      top: 8px;
      right: 8px;
    }

    .bucket-delete {
      position: absolute;
      top: 8px;
      left: 8px;
      opacity: 0;
      transition: opacity 0.2s;
    }

    &:hover .bucket-delete {
      opacity: 1;
    }
  }
}

// 账单卡片
.billing-card {
  :deep(.el-card__body) {
    padding-top: 12px;
  }
}

// 账单内容
.billing-content {
  .billing-json {
    font-family: "Consolas", "Monaco", "Courier New", monospace;
    font-size: 13px;
    line-height: 1.6;
    background-color: #f5f7fa;
    padding: 16px;
    border-radius: 4px;
    margin: 0;
  }
}

// 响应式
@media (max-width: 768px) {
  .app-container {
    padding: 12px;
  }

  .page-header {
    flex-direction: column;
    align-items: flex-start;
    gap: 12px;
  }

  .card-header {
    flex-direction: column;
    align-items: flex-start;
    gap: 12px;

    .card-actions {
      width: 100%;

      .el-button {
        flex: 1;
      }
    }
  }
}
</style>
