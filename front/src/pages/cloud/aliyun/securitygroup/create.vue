<script setup lang="ts">
import type { Region } from "@/pages/cloud/aliyun/apis/type"
import type { CreateSecurityData } from "@/pages/cloud/aliyun/securitygroup/apis/type"
import type { Merchant } from "@/pages/dashboard/apis/type"
import type { CloudAccountOption } from "@@/apis/cloud_account/type"
import { regionListApi } from "@/pages/cloud/aliyun/apis"
import { createSecurityGroup } from "@/pages/cloud/aliyun/securitygroup/apis"
import { merchantQueryApi } from "@/pages/dashboard/apis"
import { getCloudAccountOptions } from "@@/apis/cloud_account"
import {
  Delete,
  Key,
  Location,
  Plus,
  User
} from "@element-plus/icons-vue"
import { ElMessage, ElMessageBox } from "element-plus"
import { computed, onMounted, onUnmounted, ref, watch } from "vue"
import { useRoute, useRouter } from "vue-router"

defineOptions({
  name: "CloudSecurityGroupCreate"
})

const router = useRouter()
const route = useRoute()

// 数据状态
const accountType = ref<string>("merchant")
const cloudAccountList = ref<CloudAccountOption[]>([])
const selectedCloudAccount = ref<number>()
const merchantList = ref<Merchant[]>([])
const regionList = ref<Region[]>([])
const selectedMerchant = ref<number>()

// 安全组列表
const securityGroups = ref<CreateSecurityData[]>([])

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

async function fetchCloudAccountList() {
  try {
    const res = await getCloudAccountOptions("aliyun")
    cloudAccountList.value = res.data || []
  } catch (error) {
    console.error("获取云账号列表失败", error)
    ElMessage.error("获取云账号列表失败")
  }
}

// 商户变化处理
function handleMerchantChange(value: number | undefined) {
  console.log("商户变更为:", value)
  securityGroups.value = []

  // 如果选中了商户，尝试获取区域列表
  if (value) {
    // 延迟执行，确保UI更新
    setTimeout(() => {
      fetchRegionList()
    }, 100)
  }
}

// 创建新安全组配置
function createDefaultSecurityGroup(): CreateSecurityData {
  return {
    merchant_id: accountType.value === "merchant" ? (selectedMerchant.value || 0) : undefined,
    cloud_account_id: accountType.value === "system" ? (selectedCloudAccount.value || 0) : undefined,
    region_id: "",
    name: "",
    description: ""
  }
}

// 添加一个安全组配置
function addSecurityGroup() {
  if (accountType.value === "merchant") {
    if (!selectedMerchant.value) {
      ElMessage.warning("请先选择商户")
      return
    }
  } else {
    if (!selectedCloudAccount.value) {
      ElMessage.warning("请先选择云账号")
      return
    }
  }

  if (regionList.value.length === 0) {
    ElMessage.warning("正在加载区域列表，请稍候再试")
    fetchRegionList().then(() => {
      if (regionList.value.length > 0) {
        securityGroups.value.unshift(createDefaultSecurityGroup())
      } else {
        ElMessage.error("区域列表为空，无法添加安全组")
      }
    })
    return
  }

  securityGroups.value.unshift(createDefaultSecurityGroup())
}

// 移除一个安全组配置
function removeSecurityGroup(index: number) {
  securityGroups.value.splice(index, 1)
}

// 验证是否可以提交
const canSubmit = computed(() => {
  if ((accountType.value === "merchant" && !selectedMerchant.value)
    || (accountType.value === "system" && !selectedCloudAccount.value)
    || securityGroups.value.length === 0) {
    return false
  }

  // 检查所有安全组配置是否都已填写必填项
  return securityGroups.value.every(securityGroup =>
    securityGroup.region_id // 只要有区域ID即可
  )
})

// 取消流式请求的函数引用
let cancelCreateSecurityGroup: (() => void) | null = null

// 提交创建安全组
async function submitCreate() {
  if (!canSubmit.value) {
    ElMessage.warning("请完善所有安全组配置")
    return
  }

  // 确保所有安全组都设置了最新的账号ID
  securityGroups.value.forEach((sg) => {
    if (accountType.value === "merchant") {
      sg.merchant_id = selectedMerchant.value || 0
      delete sg.cloud_account_id
    } else {
      sg.cloud_account_id = selectedCloudAccount.value || 0
      delete sg.merchant_id
    }
  })

  try {
    createLoading.value = true
    createCompleted.value = false
    operationOutput.value = ""
    completionMessage.value = ""
    progressDialogVisible.value = true
    cancelCreateSecurityGroup = null

    console.log("开始创建安全组列表:", securityGroups.value)

    // 使用流式API
    cancelCreateSecurityGroup = createSecurityGroup(
      {
        list: [...securityGroups.value]
      },
      (data: string, isComplete?: boolean) => {
        console.log("收到流式数据:", data, "完成标志:", isComplete)

        if (isComplete === true) {
          console.log("已收到完成信号，重置状态")
          createLoading.value = false
          createCompleted.value = true
          completionMessage.value = data
          cancelCreateSecurityGroup = null
          ElMessage.success("安全组创建完成")
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
        console.error("创建安全组失败:", error)
        createLoading.value = false
        cancelCreateSecurityGroup = null
        ElMessage.error("创建安全组失败")
      }
    ) as (() => void) | null
  } catch (error) {
    console.error("创建安全组失败:", error)
    createLoading.value = false
    cancelCreateSecurityGroup = null
    ElMessage.error("创建安全组失败")
    progressDialogVisible.value = false
  }
}

// 处理对话框关闭
function handleDialogClose() {
  if (createCompleted.value) {
    // 创建完成后返回安全组列表页面
    router.push("/cloud/aliyun/securitygroup")
  } else if (cancelCreateSecurityGroup) {
    // 如果还在创建，询问是否取消
    ElMessageBox.confirm(
      "取消创建将停止处理，已创建的安全组不会被删除。确定取消吗？",
      "取消确认",
      {
        confirmButtonText: "确定",
        cancelButtonText: "继续创建",
        type: "warning"
      }
    )
      .then(() => {
        if (cancelCreateSecurityGroup) {
          cancelCreateSecurityGroup()
          cancelCreateSecurityGroup = null
        }
        progressDialogVisible.value = false
        createLoading.value = false
      })
      .catch(() => {
        // 用户取消操作，继续创建
      })
    return false
  } else {
    progressDialogVisible.value = false
  }
}

// 返回上一页
function goBack() {
  router.push("/cloud/securitygroup")
}

// 组件卸载时清理
onUnmounted(() => {
  if (cancelCreateSecurityGroup) {
    cancelCreateSecurityGroup()
    cancelCreateSecurityGroup = null
  }
})

// 页面加载时获取商户和区域列表，并尝试从路由参数中获取商户
onMounted(() => {
  const merchantId = route.query.merchant_id ? Number(route.query.merchant_id) : undefined
  const cloudAccountId = route.query.cloud_account_id ? Number(route.query.cloud_account_id) : undefined

  if (cloudAccountId) {
    accountType.value = "system"
    fetchCloudAccountList().then(() => {
      selectedCloudAccount.value = cloudAccountId
      fetchRegionList()
    })
  } else {
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
    // 清空区域和安全组列表
    regionList.value = []
    securityGroups.value = []
  }
}, { immediate: false })
</script>

<template>
  <div class="container">
    <!-- 顶部导航 -->
    <div class="page-nav">
      <el-breadcrumb separator="/">
        <el-breadcrumb-item :to="{ path: '/cloud/aliyun/securitygroup' }">
          安全组列表
        </el-breadcrumb-item>
        <el-breadcrumb-item>创建安全组</el-breadcrumb-item>
      </el-breadcrumb>
    </div>

    <!-- 商户和区域选择 -->
    <el-card class="select-card" shadow="hover">
      <div class="header-row">
        <div class="select-container">
          <div class="select-item">
            <span class="select-label">账号类型：</span>
            <el-select v-model="accountType" style="width: 150px" @change="() => { selectedMerchant = undefined as any; selectedCloudAccount = undefined as any; securityGroups = [] as any }">
              <el-option label="商户类型" value="merchant" />
              <el-option label="系统类型" value="system" />
            </el-select>
          </div>
          <div v-if="accountType === 'system'" class="select-item">
            <span class="select-label">云账号：</span>
            <el-select v-model="selectedCloudAccount" placeholder="请选择云账号" clearable filterable style="width: 260px">
              <template #prefix>
                <el-icon><Key /></el-icon>
              </template>
              <el-option v-for="item in cloudAccountList" :key="item.value" :label="item.label" :value="item.value" />
            </el-select>
          </div>
          <div v-else class="select-item">
            <span class="select-label">商户：</span>
            <el-select
              v-model="selectedMerchant"
              placeholder="请选择商户"
              clearable
              filterable
              style="width: 260px"
              class="merchant-select"
              @change="handleMerchantChange"
            >
              <template #prefix>
                <el-icon><User /></el-icon>
              </template>
              <el-option
                v-for="item in merchantList"
                :key="item.id"
                :label="item.name"
                :value="item.id"
              />
            </el-select>
          </div>
        </div>

        <el-button
          type="primary"
          size="default"
          @click="addSecurityGroup"
          :disabled="(accountType === 'merchant' && !selectedMerchant) || (accountType === 'system' && !selectedCloudAccount)"
        >
          <el-icon><Plus /></el-icon> 添加安全组
        </el-button>
      </div>
    </el-card>

    <!-- 安全组列表 -->
    <el-card class="security-group-card" v-if="securityGroups.length > 0" shadow="hover">
      <div v-for="(securityGroup, index) in securityGroups" :key="index" class="security-group-item">
        <div class="security-group-header">
          <el-tag size="large" type="info" effect="plain" class="security-group-tag">
            安全组 #{{ securityGroups.length - index }}
          </el-tag>
          <el-button type="danger" link @click="removeSecurityGroup(index)" :disabled="createLoading">
            <el-icon><Delete /></el-icon> 删除
          </el-button>
        </div>

        <el-divider />

        <el-form
          :model="securityGroup"
          label-position="top"
          :disabled="createLoading"
        >
          <el-row :gutter="20">
            <el-col :span="24">
              <el-form-item label="区域" required>
                <el-select v-model="securityGroup.region_id" placeholder="请选择区域" filterable style="width: 100%">
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
          </el-row>
          <el-row :gutter="20">
            <el-col :span="12">
              <el-form-item label="安全组名称">
                <el-input
                  v-model="securityGroup.name"
                  placeholder="请输入安全组名称"
                  maxlength="128"
                  show-word-limit
                />
                <div class="form-tip">
                  长度为2-128个字符，必须以字母开头
                </div>
              </el-form-item>
            </el-col>
            <el-col :span="12">
              <el-form-item label="描述">
                <el-input
                  v-model="securityGroup.description"
                  placeholder="请输入安全组描述"
                  type="textarea"
                  :rows="1"
                  maxlength="128"
                  show-word-limit
                />
                <div class="form-tip">
                  最多支持128个字符
                </div>
              </el-form-item>
            </el-col>
          </el-row>
        </el-form>
        <el-divider v-if="index < securityGroups.length - 1" />
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
          创建安全组
        </el-button>
      </div>
    </el-card>

    <el-empty v-else-if="selectedMerchant" description="请添加安全组配置" />

    <!-- 创建进度对话框 -->
    <el-dialog
      v-model="progressDialogVisible"
      :title="createCompleted ? '创建完成' : '安全组创建进度'"
      width="700px"
      :close-on-click-modal="false"
      :close-on-press-escape="false"
      :before-close="handleDialogClose"
    >
      <div class="progress-dialog-content">
        <div class="operation-output-container">
          <pre class="operation-output">{{ operationOutput }}</pre>
        </div>

        <div v-if="createCompleted" class="completion-message">
          <el-alert
            title="创建完成"
            type="success"
            :description="completionMessage || '已成功创建所有安全组'"
            show-icon
          />
        </div>
      </div>

      <template #footer>
        <div class="dialog-footer">
          <el-button v-if="createCompleted" @click="handleDialogClose">
            返回安全组列表
          </el-button>
          <el-button
            v-else
            type="danger"
            @click="handleDialogClose"
          >
            取消创建
          </el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.container {
  padding: 20px;
}

.page-nav {
  margin-bottom: 20px;
}

.select-card {
  margin-bottom: 20px;
}

.header-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.select-container {
  display: flex;
  gap: 20px;
}

.select-item {
  display: flex;
  align-items: center;
}

.select-label {
  margin-right: 8px;
  white-space: nowrap;
  font-weight: 500;
  color: #606266;
}

.security-group-card {
  margin-bottom: 20px;
}

.security-group-item {
  margin-bottom: 16px;
}

.security-group-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.security-group-tag {
  font-size: 14px;
  padding: 4px 8px;
}

.form-tip {
  font-size: 12px;
  color: #909399;
  margin-top: 4px;
}

.action-bar {
  display: flex;
  justify-content: center;
  margin-top: 20px;
  gap: 10px;
}

.progress-dialog-content {
  padding: 20px;
}

.operation-output-container {
  margin-bottom: 20px;
}

.operation-output {
  height: 300px;
  overflow-y: auto;
  background-color: #1e1e1e;
  color: #f8f8f8;
  font-family: monospace;
  padding: 10px;
  border-radius: 4px;
  white-space: pre-wrap;
  word-break: break-all;
}

.completion-message {
  margin-top: 20px;
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
}
</style>
