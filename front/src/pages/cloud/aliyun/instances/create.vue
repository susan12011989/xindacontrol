<script setup lang="ts">
import type { Region } from "@/pages/cloud/aliyun/apis/type"
import type { CreateInstanceData } from "@/pages/cloud/aliyun/instances/apis/type"
import type { Merchant } from "@/pages/dashboard/apis/type"
import type { CloudAccountOption } from "@@/apis/cloud_account/type"
import { regionListApi } from "@/pages/cloud/aliyun/apis"
import { createInstance, getImageList } from "@/pages/cloud/aliyun/instances/apis"
import { merchantQueryApi } from "@/pages/dashboard/apis"
import { getCloudAccountOptions } from "@@/apis/cloud_account"
import {
  AddLocation,
  Calendar,
  Check,
  Cpu,
  Delete,
  DocumentCopy,
  Key,
  Location,
  Lock,
  Picture,
  Plus,
  Timer,
  Wallet
} from "@element-plus/icons-vue"
import { ElMessage } from "element-plus"
import { computed, onMounted, onUnmounted, ref, watch } from "vue"
import { useRoute, useRouter } from "vue-router"

defineOptions({
  name: "CloudInstancesCreate"
})

const router = useRouter()
const route = useRoute()

// 数据状态
const accountType = ref<string>("system") // merchant: 商户类型, system: 系统类型
const cloudAccountList = ref<CloudAccountOption[]>([])
const selectedCloudAccount = ref<number>()
const merchantList = ref<Merchant[]>([])
const regionList = ref<Region[]>([])
const selectedMerchant = ref<number>()
const imageList = ref<any[]>([])

// 实例列表
const instances = ref<CreateInstanceData[]>([])

// 创建状态
const createLoading = ref(false)
const progressDialogVisible = ref(false)
const createCompleted = ref(false)
const operationOutput = ref("")
const completionMessage = ref("")

// 获取商户列表
async function fetchMerchantList() {
  try {
    const res = await merchantQueryApi({
      page: 1,
      size: 100
    })
    merchantList.value = res.data.list
  } catch (error) {
    console.error("获取商户列表失败", error)
    ElMessage.error("获取商户列表失败")
  }
}

// 获取云账号列表
async function fetchCloudAccountList() {
  try {
    const res = await getCloudAccountOptions("aliyun")
    cloudAccountList.value = res.data || []
  } catch (error) {
    console.error("获取云账号列表失败", error)
    ElMessage.error("获取云账号列表失败")
  }
}

// 获取区域列表
async function fetchRegionList() {
  try {
    console.log("开始获取区域列表...")
    const res = await regionListApi()
    console.log("获取区域列表成功:", res.data)
    regionList.value = res.data
  } catch (error) {
    console.error("获取区域列表失败", error)
    ElMessage.error("获取区域列表失败")
  }
}

// 获取镜像列表
async function fetchImageList(region: string) {
  try {
    console.log("开始获取镜像列表...", region)
    const res = await getImageList({
      cloud_account_id: selectedCloudAccount.value,
      merchant_id: selectedMerchant.value,
      region_id: [region]
    })
    console.log("获取镜像列表成功:", res.data)
    imageList.value = res.data.list
  } catch (error) {
    console.error("获取镜像列表失败", error)
    ElMessage.error("获取镜像列表失败")
  }
}

// 商户变化处理
function handleMerchantChange(value: number | undefined) {
  console.log("商户变更为:", value)
  instances.value = []

  // 如果选中了商户，尝试获取区域列表
  if (value) {
    // 延迟执行，确保UI更新
    setTimeout(() => {
      fetchRegionList()
    }, 100)
  }
}

// 处理账号类型切换
function handleAccountTypeChange() {
  // 清空选择与实例
  selectedMerchant.value = undefined
  selectedCloudAccount.value = undefined
  instances.value = []
  imageList.value = []
  if (accountType.value === "system") {
    fetchCloudAccountList()
  } else {
    fetchMerchantList()
  }
}

// 处理云账号切换
function handleCloudAccountChange() {
  instances.value = []
}

// 处理区域变更
function handleRegionChange(region: string) {
  if (region) {
    fetchImageList(region)
  }
}

// 创建新实例配置
function createDefaultInstance(): CreateInstanceData {
  return {
    merchant_id: accountType.value === "merchant" ? (selectedMerchant.value || 0) : undefined,
    cloud_account_id: accountType.value === "system" ? (selectedCloudAccount.value || 0) : undefined,
    region: "",
    image_id: "",
    instance_type: "ecs.c9i.xlarge",
    instance_charge_type: "PrePaid", // 默认包年包月
    disk_category: "cloud_essd", // 默认ESSD云盘
    disk_size: 40, // 默认40GB
    period_unit: "Month",
    period: 1,
    use_password: false, // 默认使用SSH密钥对
    password: ""
  }
}

// 添加一个实例配置
function addInstance() {
  if (accountType.value === "merchant") {
    if (!selectedMerchant.value) {
      ElMessage.warning("请先选择商户")
      return
    }
  } else {
    if (!selectedCloudAccount.value) {
      ElMessage.warning("请先选择系统云账号")
      return
    }
  }

  // 检查区域列表是否已加载
  if (regionList.value.length === 0) {
    ElMessage.warning("正在加载区域列表，请稍候再试")
    fetchRegionList().then(() => {
      if (regionList.value.length > 0) {
        instances.value.unshift(createDefaultInstance())
      } else {
        ElMessage.error("区域列表为空，无法添加实例")
      }
    })
    return
  }

  instances.value.unshift(createDefaultInstance())
}

// 移除一个实例配置
function removeInstance(index: number) {
  instances.value.splice(index, 1)
}

// 验证是否可以提交
const canSubmit = computed(() => {
  if (
    (accountType.value === "merchant" && !selectedMerchant.value)
    || (accountType.value === "system" && !selectedCloudAccount.value)
    || instances.value.length === 0
  ) {
    return false
  }

  // 检查所有实例配置是否都已填写必填项（镜像可选，默认使用Ubuntu）
  return instances.value.every(instance =>
    instance.instance_type
    && instance.disk_size >= 40
    && instance.region // 确保区域已填写
  )
})

// 取消流式请求的函数引用
let cancelCreateInstance: (() => void) | null = null

// 提交创建实例
async function submitCreate() {
  if (!canSubmit.value) {
    // 检查是否有实例没有选择区域
    const noRegionInstances = instances.value.filter(instance => !instance.region)
    if (noRegionInstances.length > 0) {
      ElMessage.warning(`有${noRegionInstances.length}个实例未选择区域，请确保每个实例都选择了区域`)
      return
    }

    ElMessage.warning("请完善所有实例配置")
    return
  }

  // 确保所有实例都设置了最新的商户ID
  instances.value.forEach((instance) => {
    if (accountType.value === "merchant") {
      instance.merchant_id = selectedMerchant.value || 0
      delete instance.cloud_account_id
    } else {
      instance.cloud_account_id = selectedCloudAccount.value || 0
      delete instance.merchant_id
    }
  })

  try {
    createLoading.value = true
    createCompleted.value = false
    operationOutput.value = ""
    completionMessage.value = ""
    progressDialogVisible.value = true
    cancelCreateInstance = null

    console.log("开始创建实例列表:", instances.value)

    // 使用流式API
    cancelCreateInstance = createInstance(
      {
        List: [...instances.value]
      },
      (data: string, isComplete?: boolean) => {
        console.log("收到流式数据:", data, "完成标志:", isComplete)

        if (isComplete === true) {
          console.log("已收到完成信号，重置状态")
          createLoading.value = false
          createCompleted.value = true
          completionMessage.value = data
          cancelCreateInstance = null
          ElMessage.success("实例创建完成")
          return
        }

        // 添加输出并滚动到底部
        operationOutput.value += `${data}\n`
        setTimeout(() => {
          const outputEl = document.querySelector(".operation-output")
          if (outputEl) {
            outputEl.scrollTop = outputEl.scrollHeight
          }
        }, 0)
      },
      (error) => {
        console.error("创建实例失败:", error)
        createLoading.value = false
        cancelCreateInstance = null
        ElMessage.error("创建实例失败")
      }
    )
  } catch (error) {
    console.error("创建实例失败:", error)
    createLoading.value = false
    cancelCreateInstance = null
    ElMessage.error("创建实例失败")
    progressDialogVisible.value = false
  }
}

// 处理对话框关闭
function handleDialogClose() {
  if (createCompleted.value) {
    // 创建完成后返回实例列表页面
    router.push("/cloud/aliyun/instances")
  } else if (cancelCreateInstance) {
    // 如果还在创建，询问是否取消
    ElMessage.warning("创建过程被中断，实例可能未完全创建")
    cancelCreateInstance()
    progressDialogVisible.value = false
    createLoading.value = false
  } else {
    progressDialogVisible.value = false
  }
}

// 返回上一页
function goBack() {
  router.push("/cloud/instances")
}

// 组件卸载时清理
onUnmounted(() => {
  if (cancelCreateInstance) {
    cancelCreateInstance()
  }
})

// 页面加载时获取商户和区域列表，并尝试从路由参数中获取商户
onMounted(() => {
  // 优先读取URL参数
  const merchantId = route.query.merchant_id ? Number(route.query.merchant_id) : undefined
  const cloudAccountId = route.query.cloud_account_id ? Number(route.query.cloud_account_id) : undefined

  if (cloudAccountId) {
    accountType.value = "system"
    fetchCloudAccountList().then(() => {
      selectedCloudAccount.value = cloudAccountId
      fetchRegionList()
    })
  } else {
    // 默认商户类型
    fetchMerchantList().then(() => {
      if (merchantId) {
        selectedMerchant.value = merchantId
        fetchRegionList()
      }
    })
  }
})

// 监听商户变化时自动获取区域列表
watch(() => selectedMerchant.value, (newVal) => {
  if (newVal) {
    fetchRegionList()
  } else {
    // 清空区域和实例列表
    regionList.value = []
    instances.value = []
    imageList.value = []
  }
}, { immediate: false })

// 监听系统云账号变化
watch(() => selectedCloudAccount.value, (newVal) => {
  if (newVal) {
    fetchRegionList()
  } else {
    regionList.value = []
    instances.value = []
    imageList.value = []
  }
}, { immediate: false })
</script>

<template>
  <div class="container">
    <!-- 顶部导航 -->
    <div class="page-nav">
      <el-breadcrumb separator="/">
        <el-breadcrumb-item :to="{ path: '/cloud/aliyun/instances' }">
          实例列表
        </el-breadcrumb-item>
        <el-breadcrumb-item>创建实例</el-breadcrumb-item>
      </el-breadcrumb>
    </div>

    <!-- 商户选择 -->
    <el-card class="select-card" shadow="hover">
      <div class="header-row">
        <div class="merchant-select-container">
          <span class="select-label">账号类型：</span>
          <el-select v-model="accountType" style="width: 150px" @change="handleAccountTypeChange">
            <el-option label="商户类型" value="merchant" />
            <el-option label="系统类型" value="system" />
          </el-select>
        </div>
        <div v-if="accountType === 'system'" class="merchant-select-container">
          <span class="select-label">云账号：</span>
          <el-select v-model="selectedCloudAccount" placeholder="请选择云账号" clearable filterable style="width: 260px" @change="handleCloudAccountChange">
            <template #prefix>
              <el-icon><Key /></el-icon>
            </template>
            <el-option v-for="item in cloudAccountList" :key="item.value" :label="item.label" :value="item.value" />
          </el-select>
        </div>
        <div v-else class="merchant-select-container">
          <span class="select-label">选择商户：</span>
          <el-select
            v-model="selectedMerchant"
            placeholder="请选择商户"
            clearable
            filterable
            style="width: 260px"
            class="merchant-select"
            @change="handleMerchantChange"
          >
            <el-option
              v-for="item in merchantList"
              :key="item.id"
              :label="item.name"
              :value="item.id"
            />
          </el-select>
        </div>
        <el-button
          type="primary"
          size="default"
          @click="addInstance"
          :disabled="(accountType === 'merchant' && !selectedMerchant) || (accountType === 'system' && !selectedCloudAccount)"
        >
          <el-icon><Plus /></el-icon> 添加实例
        </el-button>
      </div>
    </el-card>

    <!-- 实例列表 -->
    <el-card class="instance-card" v-if="instances.length > 0" shadow="hover">
      <div v-for="(instance, index) in instances" :key="index" class="instance-item">
        <div class="instance-header">
          <el-tag size="large" type="info" effect="plain" class="instance-tag">
            实例 #{{ instances.length - index }}
          </el-tag>
          <el-button type="danger" link @click="removeInstance(index)" :disabled="createLoading">
            <el-icon><Delete /></el-icon> 删除
          </el-button>
        </div>

        <el-divider />

        <el-form
          :model="instance"
          label-position="top"
          :disabled="createLoading"
        >
          <el-row :gutter="20">
            <el-col :span="8">
              <el-form-item label="区域" required>
                <el-select v-model="instance.region" placeholder="请选择区域" filterable style="width: 100%" @change="(val) => handleRegionChange(val)">
                  <template #prefix>
                    <el-icon><Location /></el-icon>
                  </template>
                  <el-option
                    v-for="item in regionList"
                    :key="item.RegionId"
                    :label="item.LocalName"
                    :value="item.RegionId"
                  />
                </el-select>
              </el-form-item>
            </el-col>
            <el-col :span="8">
              <el-form-item label="镜像ID">
                <el-select v-model="instance.image_id" placeholder="留空使用默认Ubuntu" filterable clearable style="width: 100%">
                  <template #prefix>
                    <el-icon><Picture /></el-icon>
                  </template>
                  <el-option
                    v-for="item in imageList"
                    :key="item.ImageId"
                    :label="`${item.ImageName} (${item.OSName})`"
                    :value="item.ImageId"
                  />
                </el-select>
              </el-form-item>
            </el-col>
            <el-col :span="8">
              <el-form-item label="实例规格" required>
                <el-input v-model="instance.instance_type" placeholder="请输入实例规格">
                  <template #prefix>
                    <el-icon><Cpu /></el-icon>
                  </template>
                </el-input>
              </el-form-item>
            </el-col>
          </el-row>

          <el-row :gutter="20">
            <el-col :span="8">
              <el-form-item label="付费类型" required>
                <el-select v-model="instance.instance_charge_type" style="width: 100%">
                  <template #prefix>
                    <el-icon><Wallet /></el-icon>
                  </template>
                  <el-option label="包年包月" value="PrePaid" />
                  <el-option label="按量付费" value="PostPaid" />
                </el-select>
              </el-form-item>
            </el-col>
            <el-col :span="8">
              <el-form-item label="磁盘类别" required>
                <el-select v-model="instance.disk_category" style="width: 100%">
                  <template #prefix>
                    <el-icon><DocumentCopy /></el-icon>
                  </template>
                  <el-option label="ESSD云盘（推荐）" value="cloud_essd" />
                  <el-option label="高效云盘" value="cloud_efficiency" />
                  <el-option label="SSD云盘" value="cloud_ssd" />
                  <el-option label="普通云盘" value="cloud" />
                </el-select>
              </el-form-item>
            </el-col>
            <el-col :span="8">
              <el-form-item label="系统盘大小" required>
                <el-input-number v-model="instance.disk_size" :min="40" :step="10" controls-position="right" style="width: 100%" />
                <div class="form-tip">
                  单位：GB，最小值为40
                </div>
              </el-form-item>
            </el-col>
          </el-row>

          <!-- 包年包月选项 -->
          <el-row v-if="instance.instance_charge_type === 'PrePaid'" :gutter="20">
            <el-col :span="8">
              <el-form-item label="时长单位">
                <el-select v-model="instance.period_unit" style="width: 100%">
                  <template #prefix>
                    <el-icon><Calendar /></el-icon>
                  </template>
                  <el-option label="月" value="Month" />
                  <el-option label="周" value="Week" />
                </el-select>
              </el-form-item>
            </el-col>
            <el-col :span="8">
              <el-form-item label="时长">
                <el-select v-model="instance.period" style="width: 100%">
                  <template #prefix>
                    <el-icon><Timer /></el-icon>
                  </template>
                  <template v-if="instance.period_unit === 'Week'">
                    <el-option v-for="n in 4" :key="n" :label="`${n}周`" :value="n" />
                  </template>
                  <template v-else>
                    <el-option v-for="n in [1, 2, 3, 4, 5, 6, 7, 8, 9, 12, 24, 36, 48, 60]" :key="n" :label="`${n}个月`" :value="n" />
                  </template>
                </el-select>
              </el-form-item>
            </el-col>
          </el-row>

          <!-- SSH认证方式 -->
          <el-row :gutter="20">
            <el-col :span="8">
              <el-form-item label="认证方式">
                <el-radio-group v-model="instance.use_password">
                  <el-radio :value="false">
                    <el-icon><Key /></el-icon> SSH密钥对
                  </el-radio>
                  <el-radio :value="true">
                    <el-icon><Lock /></el-icon> 密码认证
                  </el-radio>
                </el-radio-group>
              </el-form-item>
            </el-col>
            <el-col :span="8" v-if="instance.use_password">
              <el-form-item label="登录密码">
                <el-input
                  v-model="instance.password"
                  type="password"
                  placeholder="留空则自动生成安全密码"
                  show-password
                >
                  <template #prefix>
                    <el-icon><Lock /></el-icon>
                  </template>
                </el-input>
                <div class="form-tip">
                  8-30个字符，需包含大小写字母和数字
                </div>
              </el-form-item>
            </el-col>
          </el-row>

          <!-- SSH认证说明 -->
          <el-alert
            :title="instance.use_password
              ? 'SSH认证：使用密码登录，创建完成后服务器会自动注册到服务器列表'
              : 'SSH认证：系统将自动创建SSH密钥对，创建完成后服务器会自动注册到服务器列表'"
            type="info"
            :closable="false"
            show-icon
            style="margin-top: 16px"
          />
        </el-form>
        <el-divider v-if="index < instances.length - 1" />
      </div>

      <!-- 操作按钮 -->
      <div class="action-bar">
        <el-button @click="goBack">
          取消
        </el-button>
        <el-button
          type="primary"
          :loading="createLoading"
          :disabled="!canSubmit"
          @click="submitCreate"
        >
          创建实例
        </el-button>
      </div>
    </el-card>

    <!-- 空状态提示 -->
    <el-card v-if="instances.length === 0 && selectedMerchant" class="empty-card">
      <div class="empty-container">
        <el-empty description="请点击上方添加实例按钮添加实例" :image-size="100">
          <template #image>
            <el-icon :size="60">
              <AddLocation />
            </el-icon>
          </template>
        </el-empty>
      </div>
    </el-card>

    <!-- 创建进度对话框 -->
    <el-dialog
      v-model="progressDialogVisible"
      title="创建实例进度"
      width="650px"
      :close-on-click-modal="false"
      :show-close="createCompleted"
    >
      <div class="progress-info">
        <div class="progress-status">
          <span>创建状态：</span>
          <el-tag :type="createCompleted ? 'success' : 'warning'">
            {{ createCompleted ? '创建完成' : '创建中...' }}
          </el-tag>
        </div>
        <div class="progress-count">
          <span>创建数量：{{ instances.length }} 个实例</span>
        </div>
      </div>

      <div v-if="operationOutput" class="output-container operation-output">
        <pre>{{ operationOutput }}</pre>
      </div>

      <!-- 操作状态提示 -->
      <div v-if="createCompleted" class="operation-status success">
        <el-icon class="status-icon">
          <Check />
        </el-icon>
        <div class="status-content">
          <span class="status-title">创建已完成</span>
          <span v-if="completionMessage" class="status-message">{{ completionMessage }}</span>
        </div>
      </div>

      <template #footer>
        <el-button @click="handleDialogClose">
          {{ createCompleted ? '返回实例列表' : '取消' }}
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.container {
  padding: 24px;
  max-width: 1200px;
  margin: 0 auto;
}

.page-nav {
  margin-bottom: 16px;
}

.select-card {
  margin-bottom: 20px;
  padding: 4px 0;
}

.header-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0 20px;
}

.merchant-select-container {
  display: flex;
  align-items: center;
}

.select-label {
  margin-right: 8px;
  font-weight: 500;
  color: #606266;
}

.instance-card {
  margin-bottom: 20px;
}

.instance-item {
  margin-bottom: 16px;
}

.instance-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.instance-tag {
  font-size: 14px;
}

.form-tip {
  margin-top: 5px;
  color: #909399;
  font-size: 12px;
}

.empty-card {
  min-height: 200px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.empty-container {
  padding: 40px 0;
}

.action-bar {
  display: flex;
  justify-content: center;
  margin-top: 24px;
  gap: 16px;
}

.output-container {
  margin-top: 20px;
  padding: 10px;
  background-color: #f5f7fa;
  border-radius: 4px;
  max-height: 300px;
  overflow-y: auto;
}

.output-container pre {
  margin: 0;
  white-space: pre-wrap;
  word-wrap: break-word;
  font-family: monospace;
  font-size: 14px;
  line-height: 1.5;
}

.progress-info {
  margin-bottom: 16px;
  display: flex;
  justify-content: space-between;
}

.operation-status {
  margin-top: 15px;
  padding: 12px 15px;
  border-radius: 4px;
  display: flex;
  align-items: flex-start;
}

.operation-status.success {
  background-color: #f0f9eb;
  border-left: 4px solid #67c23a;
}

.operation-status .status-icon {
  font-size: 18px;
  margin-right: 10px;
  margin-top: 2px;
  color: #67c23a;
}

.operation-status .status-content {
  display: flex;
  flex-direction: column;
}

.operation-status .status-title {
  font-size: 14px;
  font-weight: 600;
  margin-bottom: 5px;
}

.operation-status .status-message {
  font-size: 13px;
  color: #606266;
  padding: 5px 8px;
  background-color: rgba(0, 0, 0, 0.03);
  border-radius: 3px;
  margin-top: 3px;
}
</style>
