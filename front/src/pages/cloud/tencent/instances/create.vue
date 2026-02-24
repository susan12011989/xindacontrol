<script lang="ts" setup>
import type * as Types from "./apis/type"
import { createInstances, getImageList, getInstanceTypeList, getVpcList, getSubnetList } from "./apis"
import { getSecurityGroupList } from "../securitygroup/apis"
import { getCloudAccountOptions } from "@@/apis/cloud_account"
import { getTencentRegions } from "@@/constants/tencent-regions"
import {
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
import { computed, onMounted, onUnmounted, ref } from "vue"
import { useRouter } from "vue-router"

defineOptions({ name: "TencentInstancesCreate" })

const router = useRouter()

// 账号类型与选择
const accountType = ref<"merchant" | "system">("system")
const cloudAccounts = ref<{ value: number; label: string }[]>([])
const selectedCloudAccount = ref<number>()
const selectedMerchant = ref<number>()
const merchantOptions = ref<{ id: number; name: string }[]>([])

// 区域
const tencentRegions = getTencentRegions("cn")

// 缓存: region -> data
const imageCache = ref<Record<string, Types.Image[]>>({})
const instanceTypeCache = ref<Record<string, Types.InstanceTypeConfig[]>>({})
const vpcCache = ref<Record<string, Types.VpcItem[]>>({})
const subnetCache = ref<Record<string, Types.SubnetItem[]>>({})
const sgCache = ref<Record<string, any[]>>({})

// 实例配置列表
const instances = ref<Types.CreateInstanceData[]>([])

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
  instances.value = []
  imageCache.value = {}
  instanceTypeCache.value = {}
  vpcCache.value = {}
  subnetCache.value = {}
  sgCache.value = {}
  if (accountType.value === "system") fetchCloudAccounts()
}

// ========== 按区域加载数据 ==========

async function loadRegionData(regionId: string) {
  const acct = buildAccountParams()
  if (!acct.merchant_id && !acct.cloud_account_id) return

  // 并发加载镜像、规格、VPC、安全组
  const promises: Promise<void>[] = []

  if (!imageCache.value[regionId]) {
    promises.push(
      getImageList({ ...acct, region_id: [regionId] }).then(res => {
        imageCache.value[regionId] = res.data.list || []
      }).catch(() => { imageCache.value[regionId] = [] })
    )
  }

  if (!instanceTypeCache.value[regionId]) {
    promises.push(
      getInstanceTypeList({ ...acct, region_id: [regionId] }).then(res => {
        instanceTypeCache.value[regionId] = res.data.list || []
      }).catch(() => { instanceTypeCache.value[regionId] = [] })
    )
  }

  if (!vpcCache.value[regionId]) {
    promises.push(
      getVpcList({ ...acct, region_id: regionId }).then(res => {
        vpcCache.value[regionId] = res.data.list || []
      }).catch(() => { vpcCache.value[regionId] = [] })
    )
  }

  if (!sgCache.value[regionId]) {
    promises.push(
      getSecurityGroupList({ ...acct, region_id: regionId }).then(res => {
        sgCache.value[regionId] = res.data.list || []
      }).catch(() => { sgCache.value[regionId] = [] })
    )
  }

  await Promise.all(promises)
}

async function loadSubnets(regionId: string, vpcId: string) {
  const key = `${regionId}__${vpcId}`
  if (subnetCache.value[key]) return
  const acct = buildAccountParams()
  try {
    const res = await getSubnetList({ ...acct, region_id: regionId, vpc_id: vpcId })
    subnetCache.value[key] = res.data.list || []
  } catch {
    subnetCache.value[key] = []
  }
}

// 获取区域的可用区列表（从规格或子网中提取）
function getZonesForRegion(regionId: string): string[] {
  const types = instanceTypeCache.value[regionId] || []
  const zones = new Set<string>()
  types.forEach(t => { if (t.Zone) zones.add(t.Zone) })
  return Array.from(zones).sort()
}

function getImagesForRegion(regionId: string): Types.Image[] {
  return imageCache.value[regionId] || []
}

function getInstanceTypesForZone(regionId: string, zone: string): Types.InstanceTypeConfig[] {
  const types = instanceTypeCache.value[regionId] || []
  if (!zone) return types
  return types.filter(t => t.Zone === zone)
}

function getVpcsForRegion(regionId: string): Types.VpcItem[] {
  return vpcCache.value[regionId] || []
}

function getSubnetsForVpc(regionId: string, vpcId: string): Types.SubnetItem[] {
  return subnetCache.value[`${regionId}__${vpcId}`] || []
}

function getSgsForRegion(regionId: string): any[] {
  return sgCache.value[regionId] || []
}

// ========== 实例配置管理 ==========

function createDefaultInstance(): Types.CreateInstanceData {
  return {
    region_id: "",
    zone: "",
    image_id: "",
    instance_type: "",
    instance_charge_type: "POSTPAID_BY_HOUR",
    system_disk_type: "CLOUD_PREMIUM",
    system_disk_size: 50,
    vpc_id: "",
    subnet_id: "",
    security_group_ids: [],
    instance_name: "",
    password: "",
    internet_max_bandwidth_out: 1,
    period: 1,
    renew_flag: "NOTIFY_AND_AUTO_RENEW"
  }
}

function addInstance() {
  const acct = buildAccountParams()
  if (!acct.merchant_id && !acct.cloud_account_id) {
    ElMessage.warning("请先选择账号")
    return
  }
  instances.value.unshift(createDefaultInstance())
}

function removeInstance(index: number) {
  instances.value.splice(index, 1)
}

async function handleRegionChange(instance: Types.CreateInstanceData) {
  // 清空依赖字段
  instance.zone = ""
  instance.image_id = ""
  instance.instance_type = ""
  instance.vpc_id = ""
  instance.subnet_id = ""
  instance.security_group_ids = []
  if (instance.region_id) {
    await loadRegionData(instance.region_id)
  }
}

async function handleVpcChange(instance: Types.CreateInstanceData) {
  instance.subnet_id = ""
  if (instance.vpc_id && instance.region_id) {
    await loadSubnets(instance.region_id, instance.vpc_id)
  }
}

function handleZoneChange(instance: Types.CreateInstanceData) {
  instance.instance_type = ""
  instance.subnet_id = ""
}

// ========== 提交创建 ==========

const canSubmit = computed(() => {
  const acct = buildAccountParams()
  if (!acct.merchant_id && !acct.cloud_account_id) return false
  if (instances.value.length === 0) return false
  return instances.value.every(i =>
    i.region_id && i.zone && i.instance_type && i.system_disk_size >= 20
  )
})

let cancelCreate: (() => void) | null = null

function submitCreate() {
  if (!canSubmit.value) {
    ElMessage.warning("请完善所有实例配置")
    return
  }

  const acct = buildAccountParams()
  const list = instances.value.map(i => ({ ...i, ...acct }))

  createLoading.value = true
  createCompleted.value = false
  operationOutput.value = ""
  completionMessage.value = ""
  progressDialogVisible.value = true
  cancelCreate = null

  cancelCreate = createInstances(
    { list },
    (data: string, isComplete?: boolean) => {
      if (isComplete === true) {
        createLoading.value = false
        createCompleted.value = true
        completionMessage.value = data
        cancelCreate = null
        ElMessage.success("实例创建完成")
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
      ElMessage.error("创建实例失败")
    }
  ) as (() => void) | null
}

function handleDialogClose() {
  if (createCompleted.value) {
    router.push("/cloud/tencent/instances")
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
  router.push("/cloud/tencent/instances")
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
        <el-breadcrumb-item :to="{ path: '/cloud/tencent/instances' }">实例列表</el-breadcrumb-item>
        <el-breadcrumb-item>创建实例</el-breadcrumb-item>
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
          @click="addInstance"
        >
          <el-icon><Plus /></el-icon> 添加实例
        </el-button>
      </div>
    </el-card>

    <!-- 实例配置列表 -->
    <el-card v-if="instances.length > 0" class="instance-card" shadow="hover">
      <div v-for="(instance, index) in instances" :key="index" class="instance-item">
        <div class="instance-header">
          <el-tag size="large" type="info" effect="plain">实例 #{{ instances.length - index }}</el-tag>
          <el-button type="danger" link @click="removeInstance(index)" :disabled="createLoading">
            <el-icon><Delete /></el-icon> 删除
          </el-button>
        </div>

        <el-divider />

        <el-form :model="instance" label-position="top" :disabled="createLoading">
          <!-- 区域 + 可用区 + 镜像 -->
          <el-row :gutter="20">
            <el-col :span="8">
              <el-form-item label="区域" required>
                <el-select v-model="instance.region_id" placeholder="请选择区域" filterable style="width: 100%" @change="() => handleRegionChange(instance)">
                  <template #prefix><el-icon><Location /></el-icon></template>
                  <el-option v-for="r in tencentRegions" :key="r.id" :label="`${r.name} (${r.id})`" :value="r.id" />
                </el-select>
              </el-form-item>
            </el-col>
            <el-col :span="8">
              <el-form-item label="可用区" required>
                <el-select v-model="instance.zone" placeholder="请选择可用区" filterable style="width: 100%" @change="() => handleZoneChange(instance)">
                  <template #prefix><el-icon><Location /></el-icon></template>
                  <el-option v-for="z in getZonesForRegion(instance.region_id)" :key="z" :label="z" :value="z" />
                </el-select>
                <div v-if="instance.region_id && getZonesForRegion(instance.region_id).length === 0" class="form-tip">加载中或暂无可用区</div>
              </el-form-item>
            </el-col>
            <el-col :span="8">
              <el-form-item label="镜像">
                <el-select v-model="instance.image_id" placeholder="留空使用默认 Ubuntu" filterable clearable style="width: 100%">
                  <template #prefix><el-icon><Picture /></el-icon></template>
                  <el-option
                    v-for="img in getImagesForRegion(instance.region_id)"
                    :key="img.ImageId"
                    :label="`${img.ImageName} (${img.OsName})`"
                    :value="img.ImageId"
                  />
                </el-select>
              </el-form-item>
            </el-col>
          </el-row>

          <!-- 实例规格 + 实例名称 + 密码 -->
          <el-row :gutter="20">
            <el-col :span="8">
              <el-form-item label="实例规格" required>
                <el-select v-model="instance.instance_type" placeholder="请选择实例规格" filterable style="width: 100%">
                  <template #prefix><el-icon><Cpu /></el-icon></template>
                  <el-option
                    v-for="t in getInstanceTypesForZone(instance.region_id, instance.zone)"
                    :key="t.InstanceType"
                    :label="`${t.InstanceType} (${t.CPU}核/${t.Memory}GB)`"
                    :value="t.InstanceType"
                  />
                </el-select>
              </el-form-item>
            </el-col>
            <el-col :span="8">
              <el-form-item label="实例名称">
                <el-input v-model="instance.instance_name" placeholder="可选" />
              </el-form-item>
            </el-col>
            <el-col :span="8">
              <el-form-item label="登录密码">
                <el-input v-model="instance.password" type="password" show-password placeholder="8-30位，含大小写字母+数字">
                  <template #prefix><el-icon><Lock /></el-icon></template>
                </el-input>
              </el-form-item>
            </el-col>
          </el-row>

          <!-- VPC + 子网 + 安全组 -->
          <el-row :gutter="20">
            <el-col :span="8">
              <el-form-item label="VPC">
                <el-select v-model="instance.vpc_id" placeholder="请选择 VPC" filterable clearable style="width: 100%" @change="() => handleVpcChange(instance)">
                  <el-option
                    v-for="v in getVpcsForRegion(instance.region_id)"
                    :key="v.VpcId"
                    :label="`${v.VpcName || v.VpcId} (${v.CidrBlock})`"
                    :value="v.VpcId"
                  />
                </el-select>
              </el-form-item>
            </el-col>
            <el-col :span="8">
              <el-form-item label="子网">
                <el-select v-model="instance.subnet_id" placeholder="请选择子网" filterable clearable style="width: 100%">
                  <el-option
                    v-for="s in getSubnetsForVpc(instance.region_id, instance.vpc_id || '')"
                    :key="s.SubnetId"
                    :label="`${s.SubnetName || s.SubnetId} (${s.CidrBlock}) [${s.Zone}]`"
                    :value="s.SubnetId"
                  />
                </el-select>
              </el-form-item>
            </el-col>
            <el-col :span="8">
              <el-form-item label="安全组">
                <el-select v-model="instance.security_group_ids" multiple placeholder="请选择安全组" filterable clearable style="width: 100%">
                  <el-option
                    v-for="sg in getSgsForRegion(instance.region_id)"
                    :key="sg.SecurityGroupId"
                    :label="`${sg.SecurityGroupName || sg.SecurityGroupId}`"
                    :value="sg.SecurityGroupId"
                  />
                </el-select>
              </el-form-item>
            </el-col>
          </el-row>

          <!-- 系统盘 + 付费类型 + 公网带宽 -->
          <el-row :gutter="20">
            <el-col :span="8">
              <el-form-item label="系统盘类型">
                <el-select v-model="instance.system_disk_type" style="width: 100%">
                  <template #prefix><el-icon><DocumentCopy /></el-icon></template>
                  <el-option label="高性能云硬盘" value="CLOUD_PREMIUM" />
                  <el-option label="SSD 云硬盘" value="CLOUD_SSD" />
                  <el-option label="增强型 SSD" value="CLOUD_HSSD" />
                </el-select>
              </el-form-item>
            </el-col>
            <el-col :span="8">
              <el-form-item label="系统盘大小 (GB)">
                <el-input-number v-model="instance.system_disk_size" :min="20" :step="10" controls-position="right" style="width: 100%" />
              </el-form-item>
            </el-col>
            <el-col :span="8">
              <el-form-item label="公网带宽 (Mbps)">
                <el-input-number v-model="instance.internet_max_bandwidth_out" :min="0" :max="200" :step="1" controls-position="right" style="width: 100%" />
                <div class="form-tip">0 表示不分配公网 IP</div>
              </el-form-item>
            </el-col>
          </el-row>

          <!-- 付费类型 -->
          <el-row :gutter="20">
            <el-col :span="8">
              <el-form-item label="付费类型">
                <el-select v-model="instance.instance_charge_type" style="width: 100%">
                  <template #prefix><el-icon><Wallet /></el-icon></template>
                  <el-option label="按量计费" value="POSTPAID_BY_HOUR" />
                  <el-option label="包年包月" value="PREPAID" />
                </el-select>
              </el-form-item>
            </el-col>
            <template v-if="instance.instance_charge_type === 'PREPAID'">
              <el-col :span="8">
                <el-form-item label="购买时长 (月)">
                  <el-select v-model="instance.period" style="width: 100%">
                    <template #prefix><el-icon><Timer /></el-icon></template>
                    <el-option v-for="n in [1,2,3,4,5,6,7,8,9,12,24,36,48,60]" :key="n" :label="`${n}个月`" :value="n" />
                  </el-select>
                </el-form-item>
              </el-col>
              <el-col :span="8">
                <el-form-item label="自动续费">
                  <el-select v-model="instance.renew_flag" style="width: 100%">
                    <el-option label="通知且自动续费" value="NOTIFY_AND_AUTO_RENEW" />
                    <el-option label="通知不续费" value="NOTIFY_AND_MANUAL_RENEW" />
                    <el-option label="不通知不续费" value="DISABLE_NOTIFY_AND_MANUAL_RENEW" />
                  </el-select>
                </el-form-item>
              </el-col>
            </template>
          </el-row>
        </el-form>

        <el-divider v-if="index < instances.length - 1" />
      </div>

      <!-- 操作按钮 -->
      <div class="action-bar">
        <el-button @click="goBack">取消</el-button>
        <el-button type="primary" :loading="createLoading" :disabled="!canSubmit" @click="submitCreate">
          创建实例
        </el-button>
      </div>
    </el-card>

    <!-- 空状态 -->
    <el-empty v-if="instances.length === 0 && (selectedCloudAccount || selectedMerchant)" description="请点击上方添加实例按钮" />

    <!-- 创建进度对话框 -->
    <el-dialog
      v-model="progressDialogVisible"
      :title="createCompleted ? '创建完成' : '创建实例进度'"
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
          <el-alert title="创建完成" type="success" :description="completionMessage || '已成功创建所有实例'" show-icon />
        </div>
      </div>
      <template #footer>
        <el-button v-if="createCompleted" @click="handleDialogClose">返回实例列表</el-button>
        <el-button v-else type="danger" @click="handleDialogClose">取消创建</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.container {
  padding: 20px;
  max-width: 1200px;
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
.form-tip {
  font-size: 12px;
  color: #909399;
  margin-top: 4px;
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
