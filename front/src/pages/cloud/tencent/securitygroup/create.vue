<script lang="ts" setup>
import type * as Types from "./apis/type"
import { createSecurityGroups } from "./apis"
import { getCloudAccountOptions } from "@@/apis/cloud_account"
import { getTencentRegions } from "@@/constants/tencent-regions"
import { Delete, Key, Location, Plus } from "@element-plus/icons-vue"
import { computed, onMounted, onUnmounted, ref } from "vue"
import { useRouter } from "vue-router"

defineOptions({ name: "TencentSecurityGroupCreate" })

const router = useRouter()

// 账号类型
const accountType = ref<"merchant" | "system">("system")
const cloudAccounts = ref<{ value: number; label: string }[]>([])
const selectedCloudAccount = ref<number>()
const selectedMerchant = ref<number>()
const merchantOptions = ref<{ id: number; name: string }[]>([])

// 区域
const tencentRegions = getTencentRegions("cn")

// 安全组列表
const securityGroups = ref<Types.CreateSecurityGroupData[]>([])

// 创建状态
const createLoading = ref(false)
const progressDialogVisible = ref(false)
const createCompleted = ref(false)
const operationOutput = ref("")
const completionMessage = ref("")

function buildAccountParams(): { merchant_id?: number; cloud_account_id?: number } {
  if (accountType.value === "system") return { cloud_account_id: selectedCloudAccount.value }
  return { merchant_id: selectedMerchant.value }
}

async function fetchCloudAccounts() {
  const res = await getCloudAccountOptions("tencent")
  cloudAccounts.value = res.data || []
}

async function searchMerchant(query: string) {
  const { merchantQueryApi } = await import("@/pages/dashboard/apis")
  const { data } = await merchantQueryApi({ page: 1, size: 20, name: query })
  merchantOptions.value = data.list || []
}

function handleAccountTypeChange() {
  selectedCloudAccount.value = undefined
  selectedMerchant.value = undefined
  securityGroups.value = []
  if (accountType.value === "system") fetchCloudAccounts()
}

function createDefaultSG(): Types.CreateSecurityGroupData {
  return {
    region_id: "",
    name: "",
    description: ""
  }
}

function addSecurityGroup() {
  const acct = buildAccountParams()
  if (!acct.merchant_id && !acct.cloud_account_id) {
    ElMessage.warning("请先选择账号")
    return
  }
  securityGroups.value.unshift(createDefaultSG())
}

function removeSecurityGroup(index: number) {
  securityGroups.value.splice(index, 1)
}

const canSubmit = computed(() => {
  const acct = buildAccountParams()
  if (!acct.merchant_id && !acct.cloud_account_id) return false
  if (securityGroups.value.length === 0) return false
  return securityGroups.value.every(sg => sg.region_id)
})

let cancelCreate: (() => void) | null = null

function submitCreate() {
  if (!canSubmit.value) {
    ElMessage.warning("请完善所有安全组配置")
    return
  }

  const acct = buildAccountParams()
  const list = securityGroups.value.map(sg => ({ ...sg, ...acct }))

  createLoading.value = true
  createCompleted.value = false
  operationOutput.value = ""
  completionMessage.value = ""
  progressDialogVisible.value = true
  cancelCreate = null

  cancelCreate = createSecurityGroups(
    { list },
    (data: string, isComplete?: boolean) => {
      if (isComplete === true) {
        createLoading.value = false
        createCompleted.value = true
        completionMessage.value = data
        cancelCreate = null
        ElMessage.success("安全组创建完成")
        return
      }
      operationOutput.value += `${data}\n`
      setTimeout(() => {
        const outputEl = document.querySelector(".operation-output")
        if (outputEl) outputEl.scrollTop = outputEl.scrollHeight
      }, 0)
    },
    () => {
      createLoading.value = false
      cancelCreate = null
      ElMessage.error("创建安全组失败")
    }
  ) as (() => void) | null
}

function handleDialogClose() {
  if (createCompleted.value) {
    router.push("/cloud/tencent/securitygroup")
  } else if (cancelCreate) {
    ElMessage.warning("创建过程被中断")
    cancelCreate()
    cancelCreate = null
    progressDialogVisible.value = false
    createLoading.value = false
  } else {
    progressDialogVisible.value = false
  }
}

function goBack() {
  router.push("/cloud/tencent/securitygroup")
}

onMounted(() => {
  fetchCloudAccounts()
})

onUnmounted(() => {
  if (cancelCreate) {
    cancelCreate()
    cancelCreate = null
  }
})
</script>

<template>
  <div class="container">
    <!-- 顶部导航 -->
    <div class="page-nav">
      <el-breadcrumb separator="/">
        <el-breadcrumb-item :to="{ path: '/cloud/tencent/securitygroup' }">安全组列表</el-breadcrumb-item>
        <el-breadcrumb-item>创建安全组</el-breadcrumb-item>
      </el-breadcrumb>
    </div>

    <!-- 账号选择 -->
    <el-card class="select-card" shadow="hover">
      <div class="header-row">
        <div class="select-container">
          <div class="select-item">
            <span class="select-label">账号类型：</span>
            <el-select v-model="accountType" style="width: 150px" @change="handleAccountTypeChange">
              <el-option label="系统类型" value="system" />
              <el-option label="商户类型" value="merchant" />
            </el-select>
          </div>
          <div v-if="accountType === 'system'" class="select-item">
            <span class="select-label">云账号：</span>
            <el-select v-model="selectedCloudAccount" placeholder="请选择云账号" filterable clearable style="width: 260px">
              <template #prefix><el-icon><Key /></el-icon></template>
              <el-option v-for="opt in cloudAccounts" :key="opt.value" :label="opt.label" :value="opt.value" />
            </el-select>
          </div>
          <div v-else class="select-item">
            <span class="select-label">商户：</span>
            <el-select v-model="selectedMerchant" placeholder="搜索商户" filterable remote clearable :remote-method="searchMerchant" style="width: 260px">
              <el-option v-for="m in merchantOptions" :key="m.id" :label="m.name" :value="m.id" />
            </el-select>
          </div>
        </div>
        <el-button
          type="primary"
          :disabled="!buildAccountParams().merchant_id && !buildAccountParams().cloud_account_id"
          @click="addSecurityGroup"
        >
          <el-icon><Plus /></el-icon> 添加安全组
        </el-button>
      </div>
    </el-card>

    <!-- 安全组配置列表 -->
    <el-card v-if="securityGroups.length > 0" class="sg-card" shadow="hover">
      <div v-for="(sg, index) in securityGroups" :key="index" class="sg-item">
        <div class="sg-header">
          <el-tag size="large" type="info" effect="plain">安全组 #{{ securityGroups.length - index }}</el-tag>
          <el-button type="danger" link @click="removeSecurityGroup(index)" :disabled="createLoading">
            <el-icon><Delete /></el-icon> 删除
          </el-button>
        </div>
        <el-divider />
        <el-form :model="sg" label-position="top" :disabled="createLoading">
          <el-row :gutter="20">
            <el-col :span="24">
              <el-form-item label="区域" required>
                <el-select v-model="sg.region_id" placeholder="请选择区域" filterable style="width: 100%">
                  <template #prefix><el-icon><Location /></el-icon></template>
                  <el-option v-for="r in tencentRegions" :key="r.id" :label="`${r.name} (${r.id})`" :value="r.id" />
                </el-select>
              </el-form-item>
            </el-col>
          </el-row>
          <el-row :gutter="20">
            <el-col :span="12">
              <el-form-item label="安全组名称">
                <el-input v-model="sg.name" placeholder="请输入安全组名称" maxlength="60" show-word-limit />
              </el-form-item>
            </el-col>
            <el-col :span="12">
              <el-form-item label="描述">
                <el-input v-model="sg.description" placeholder="可选" maxlength="100" show-word-limit />
              </el-form-item>
            </el-col>
          </el-row>
        </el-form>
        <el-divider v-if="index < securityGroups.length - 1" />
      </div>

      <el-alert
        title="创建后将自动添加默认入站规则（SSH 22、HTTP 80、HTTPS 443、Control 58182、GOST 等端口）"
        type="info"
        :closable="false"
        show-icon
        style="margin-top: 10px"
      />

      <div class="action-bar">
        <el-button @click="goBack">取消</el-button>
        <el-button type="primary" :loading="createLoading" :disabled="!canSubmit" @click="submitCreate">
          创建安全组
        </el-button>
      </div>
    </el-card>

    <el-empty v-if="securityGroups.length === 0 && (selectedCloudAccount || selectedMerchant)" description="请点击上方添加安全组按钮" />

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
          <el-alert title="创建完成" type="success" :description="completionMessage || '已成功创建所有安全组'" show-icon />
        </div>
      </div>
      <template #footer>
        <el-button v-if="createCompleted" @click="handleDialogClose">返回安全组列表</el-button>
        <el-button v-else type="danger" @click="handleDialogClose">取消创建</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.container {
  padding: 20px;
  max-width: 1000px;
  margin: 0 auto;
}
.page-nav {
  margin-bottom: 16px;
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
.sg-card {
  margin-bottom: 20px;
}
.sg-item {
  margin-bottom: 16px;
}
.sg-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}
.action-bar {
  display: flex;
  justify-content: center;
  margin-top: 24px;
  gap: 16px;
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
</style>
