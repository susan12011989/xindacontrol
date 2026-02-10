<script lang="ts" setup>
import type * as Types from "./apis/type"
import { merchantQueryApi } from "@/pages/dashboard/apis"
import { getInstanceList, operateInstance, modifyInstanceAttribute, resetInstancePassword } from "./apis"
import { getCloudAccountOptions } from "@@/apis/cloud_account"
import { getTencentRegions } from "@@/constants/tencent-regions"
import { ArrowDown, Delete, Edit, RefreshRight, VideoPause, VideoPlay } from "@element-plus/icons-vue"

defineOptions({ name: "TencentInstances" })

const loading = ref(false)
const rows = ref<Types.Instance[]>([])

// 账号类型与账号/商户选择
const accountType = ref<"merchant" | "system">("merchant")
const cloudAccounts = ref<{ value: number, label: string }[]>([])
const selectedCloudAccount = ref<number>()
const selectedMerchant = ref<number>()
const merchantOptions = ref<{ id: number, name: string }[]>([])

// 区域选择（使用腾讯云区域常量）
const tencentRegions = getTencentRegions("cn")
const selectedRegions = ref<string[]>([])

const selection = ref<Types.Instance[]>([])

// 编辑属性对话框
const editVisible = ref(false)
const editForm = reactive<Types.ModifyInstanceAttributeRequestData>({
  region_id: "",
  instance_id: "",
  instance_name: "",
  security_group_ids: []
})

// 重置密码对话框
const resetPwdVisible = ref(false)
const resetPwdForm = reactive<Types.ResetInstancePasswordRequestData>({
  region_id: "",
  instance_id: "",
  password: ""
})

function getPublicIp(row: Types.Instance): string {
  return row.PublicIpAddresses?.length ? row.PublicIpAddresses.join(", ") : "-"
}

function getPrivateIp(row: Types.Instance): string {
  return row.PrivateIpAddresses?.length ? row.PrivateIpAddresses.join(", ") : "-"
}

function getStateText(state: string | undefined): string {
  switch ((state || "").toUpperCase()) {
    case "RUNNING":
      return "运行中"
    case "STOPPED":
      return "已停止"
    case "STOPPING":
      return "关机中"
    case "STARTING":
      return "开机中"
    case "PENDING":
      return "创建中"
    case "REBOOTING":
      return "重启中"
    case "TERMINATING":
      return "销毁中"
    case "SHUTDOWN":
      return "已停止"
    case "LAUNCH_FAILED":
      return "创建失败"
    default:
      return state || "-"
  }
}

function getStateTagType(state: string | undefined): "success" | "info" | "warning" | "danger" | undefined {
  switch ((state || "").toUpperCase()) {
    case "RUNNING":
      return "success"
    case "STOPPED":
    case "SHUTDOWN":
      return "info"
    case "STOPPING":
    case "STARTING":
    case "PENDING":
    case "REBOOTING":
    case "TERMINATING":
      return "warning"
    case "LAUNCH_FAILED":
      return "danger"
    default:
      return undefined
  }
}

function getChargeTypeText(chargeType: string | undefined): string {
  switch (chargeType) {
    case "PREPAID":
      return "包年包月"
    case "POSTPAID_BY_HOUR":
      return "按量计费"
    case "SPOTPAID":
      return "竞价实例"
    default:
      return chargeType || "-"
  }
}

function getRegionFromZone(zone: string | undefined): string {
  if (!zone) return ""
  // 腾讯云可用区格式: ap-guangzhou-1 -> ap-guangzhou
  const parts = zone.split("-")
  if (parts.length >= 2) {
    return parts.slice(0, -1).join("-")
  }
  return zone
}

async function fetchCloudAccounts() {
  const res = await getCloudAccountOptions("tencent")
  cloudAccounts.value = res.data || []
}

async function searchMerchant(query: string) {
  const { data } = await merchantQueryApi({ page: 1, size: 20, name: query })
  merchantOptions.value = data.list || []
}

watch(accountType, () => {
  selectedCloudAccount.value = undefined
  selectedMerchant.value = undefined
  selectedRegions.value = []
  rows.value = []
  if (accountType.value === "system") fetchCloudAccounts()
})

onMounted(() => {
  fetchCloudAccounts()
})

async function onQuery() {
  const params: Types.ListRequestData = { region_id: selectedRegions.value }
  if (accountType.value === "system") {
    if (!selectedCloudAccount.value) return ElMessage.warning("请选择系统云账号")
    params.cloud_account_id = selectedCloudAccount.value
  } else {
    if (!selectedMerchant.value) return ElMessage.warning("请选择商户")
    params.merchant_id = selectedMerchant.value
  }
  loading.value = true
  try {
    const { data } = await getInstanceList(params)
    rows.value = data.list || []
  } finally {
    loading.value = false
  }
}

function onSelectionChange(sel: Types.Instance[]) {
  selection.value = sel
}

async function onOperate(row: Types.Instance, operation: Types.OperateInstanceRequestData["operation"]) {
  const region = getRegionFromZone(row.Placement?.Zone)
  if (!region) return ElMessage.warning("无法获取区域信息")

  // 销毁操作需要二次确认
  if (operation === "delete") {
    await ElMessageBox.confirm("确定要销毁此实例吗？此操作不可恢复！", "警告", {
      confirmButtonText: "确定",
      cancelButtonText: "取消",
      type: "warning"
    })
  }

  const data: Types.OperateInstanceRequestData = {
    region_id: region,
    instance_id: row.InstanceId,
    operation
  }
  if (accountType.value === "system") data.cloud_account_id = selectedCloudAccount.value
  else data.merchant_id = selectedMerchant.value
  loading.value = true
  try {
    await operateInstance(data)
    ElMessage.success("操作已提交")
    // 延迟刷新
    setTimeout(() => onQuery(), 2000)
  } finally {
    loading.value = false
  }
}

async function onBatchOperate(operation: Types.OperateInstanceRequestData["operation"]) {
  if (selection.value.length === 0) return

  // 销毁操作需要二次确认
  if (operation === "delete") {
    await ElMessageBox.confirm(`确定要批量销毁 ${selection.value.length} 个实例吗？此操作不可恢复！`, "警告", {
      confirmButtonText: "确定",
      cancelButtonText: "取消",
      type: "warning"
    })
  }

  loading.value = true
  try {
    for (const row of selection.value) {
      const region = getRegionFromZone(row.Placement?.Zone)
      if (!region) continue
      const data: Types.OperateInstanceRequestData = {
        region_id: region,
        instance_id: row.InstanceId,
        operation
      }
      if (accountType.value === "system") data.cloud_account_id = selectedCloudAccount.value
      else data.merchant_id = selectedMerchant.value
      await operateInstance(data)
    }
    ElMessage.success("批量操作完成")
    setTimeout(() => onQuery(), 2000)
  } finally {
    loading.value = false
  }
}

function onShowEdit(row: Types.Instance) {
  const region = getRegionFromZone(row.Placement?.Zone)
  if (!region) return ElMessage.warning("无法获取区域信息")
  editForm.region_id = region
  editForm.instance_id = row.InstanceId
  editForm.instance_name = row.InstanceName || ""
  editForm.security_group_ids = row.SecurityGroupIds || []
  editVisible.value = true
}

async function onSubmitEdit() {
  if (accountType.value === "system") {
    if (!selectedCloudAccount.value) return ElMessage.warning("请选择系统云账号")
    editForm.cloud_account_id = selectedCloudAccount.value
    editForm.merchant_id = undefined
  } else {
    if (!selectedMerchant.value) return ElMessage.warning("请选择商户")
    editForm.merchant_id = selectedMerchant.value
    editForm.cloud_account_id = undefined
  }
  loading.value = true
  try {
    await modifyInstanceAttribute(editForm)
    ElMessage.success("修改成功")
    editVisible.value = false
    onQuery()
  } finally {
    loading.value = false
  }
}

function onShowResetPwd(row: Types.Instance) {
  const region = getRegionFromZone(row.Placement?.Zone)
  if (!region) return ElMessage.warning("无法获取区域信息")
  resetPwdForm.region_id = region
  resetPwdForm.instance_id = row.InstanceId
  resetPwdForm.password = ""
  resetPwdVisible.value = true
}

async function onSubmitResetPwd() {
  if (!resetPwdForm.password) return ElMessage.warning("请输入新密码")
  if (resetPwdForm.password.length < 8) return ElMessage.warning("密码长度不能少于8位")

  if (accountType.value === "system") {
    if (!selectedCloudAccount.value) return ElMessage.warning("请选择系统云账号")
    resetPwdForm.cloud_account_id = selectedCloudAccount.value
    resetPwdForm.merchant_id = undefined
  } else {
    if (!selectedMerchant.value) return ElMessage.warning("请选择商户")
    resetPwdForm.merchant_id = selectedMerchant.value
    resetPwdForm.cloud_account_id = undefined
  }
  loading.value = true
  try {
    await resetInstancePassword(resetPwdForm)
    ElMessage.success("密码重置成功，请重启实例使新密码生效")
    resetPwdVisible.value = false
  } finally {
    loading.value = false
  }
}

function formatDiskSize(disk: Types.SystemDisk | undefined): string {
  if (!disk) return "-"
  return `${disk.DiskSize}GB`
}
</script>

<template>
  <div class="container">
    <el-card class="filter-card">
      <div class="filter-row">
        <div class="filter-item">
          <span class="label">账号类型：</span>
          <el-select v-model="accountType" style="width: 150px">
            <el-option label="系统类型" value="system" />
            <el-option label="商户类型" value="merchant" />
          </el-select>
        </div>
        <div v-if="accountType === 'system'" class="filter-item">
          <span class="label">云账号：</span>
          <el-select v-model="selectedCloudAccount" placeholder="请选择云账号" filterable clearable style="width: 240px">
            <el-option v-for="opt in cloudAccounts" :key="opt.value" :label="opt.label" :value="opt.value" />
          </el-select>
        </div>
        <div v-else class="filter-item">
          <span class="label">商户：</span>
          <el-select v-model="selectedMerchant" placeholder="搜索商户" filterable remote clearable :remote-method="searchMerchant" style="width: 260px">
            <el-option v-for="m in merchantOptions" :key="m.id" :label="m.name" :value="m.id" />
          </el-select>
        </div>
        <div class="filter-item">
          <span class="label">区域：</span>
          <el-select v-model="selectedRegions" multiple filterable collapse-tags collapse-tags-tooltip placeholder="请选择区域" style="min-width: 360px; max-width: 640px">
            <el-option v-for="r in tencentRegions" :key="r.id" :label="`${r.name} (${r.id})`" :value="r.id" />
          </el-select>
        </div>
        <el-button type="primary" :loading="loading" @click="onQuery">查询</el-button>
      </div>
    </el-card>

    <el-card class="table-card">
      <template #header>
        <div class="card-header">
          <span>实例列表</span>
          <div class="operations">
            <el-button size="small" type="primary" @click="onQuery">刷新</el-button>
            <el-button size="small" type="primary" :disabled="selection.length === 0" @click="onBatchOperate('start')">批量启动</el-button>
            <el-button size="small" type="warning" :disabled="selection.length === 0" @click="onBatchOperate('stop')">批量停止</el-button>
          </div>
        </div>
      </template>

      <el-table :data="rows" v-loading="loading" border height="560" @selection-change="onSelectionChange">
        <el-table-column type="selection" width="55" />
        <el-table-column prop="InstanceId" label="实例ID" min-width="180" />
        <el-table-column prop="InstanceName" label="名称" min-width="140" />
        <el-table-column label="区域/可用区" min-width="160">
          <template #default="{ row }">
            {{ row.Placement?.Zone || '-' }}
          </template>
        </el-table-column>
        <el-table-column prop="InstanceType" label="实例规格" min-width="120" />
        <el-table-column label="配置" min-width="120">
          <template #default="{ row }">
            {{ row.CPU }}核 / {{ row.Memory }}GB
          </template>
        </el-table-column>
        <el-table-column label="系统盘" min-width="100">
          <template #default="{ row }">
            {{ formatDiskSize(row.SystemDisk) }}
          </template>
        </el-table-column>
        <el-table-column label="公网IP" min-width="140">
          <template #default="{ row }">
            {{ getPublicIp(row) }}
          </template>
        </el-table-column>
        <el-table-column label="私网IP" min-width="140">
          <template #default="{ row }">
            {{ getPrivateIp(row) }}
          </template>
        </el-table-column>
        <el-table-column label="付费类型" min-width="100">
          <template #default="{ row }">
            <el-tag size="small" :type="row.InstanceChargeType === 'PREPAID' ? 'warning' : undefined">
              {{ getChargeTypeText(row.InstanceChargeType) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="状态" min-width="100">
          <template #default="{ row }">
            <el-tag :type="getStateTagType(row.InstanceState)">
              {{ getStateText(row.InstanceState) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="OsName" label="操作系统" min-width="160" show-overflow-tooltip />
        <el-table-column label="操作" fixed="right" min-width="120">
          <template #default="{ row }">
            <el-dropdown trigger="click">
              <el-button type="primary" text size="small">
                更多<el-icon class="el-icon--right"><ArrowDown /></el-icon>
              </el-button>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item @click="onShowEdit(row)">
                    <el-icon><Edit /></el-icon> 编辑属性
                  </el-dropdown-item>
                  <el-dropdown-item @click="onShowResetPwd(row)">
                    <el-icon><Edit /></el-icon> 重置密码
                  </el-dropdown-item>
                  <el-dropdown-item divided @click="onOperate(row, 'start')">
                    <el-icon><VideoPlay /></el-icon> 启动
                  </el-dropdown-item>
                  <el-dropdown-item @click="onOperate(row, 'stop')">
                    <el-icon><VideoPause /></el-icon> 停止
                  </el-dropdown-item>
                  <el-dropdown-item @click="onOperate(row, 'restart')">
                    <el-icon><RefreshRight /></el-icon> 重启
                  </el-dropdown-item>
                  <el-dropdown-item divided @click="onOperate(row, 'delete')">
                    <el-icon><Delete /></el-icon> 销毁
                  </el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- 编辑属性弹窗 -->
    <el-dialog v-model="editVisible" title="编辑实例属性" width="500px">
      <el-form :model="editForm" label-width="100px">
        <el-form-item label="实例名称">
          <el-input v-model="editForm.instance_name" placeholder="请输入实例名称" />
        </el-form-item>
        <el-form-item label="安全组">
          <el-select v-model="editForm.security_group_ids" multiple filterable placeholder="请选择安全组" style="width: 100%">
            <!-- 安全组选项需要从API获取，这里暂时只允许编辑已有的 -->
          </el-select>
          <div class="text-gray-400 text-xs mt-1">注：如需修改安全组，请确保输入正确的安全组ID</div>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="editVisible = false">取消</el-button>
        <el-button type="primary" :loading="loading" @click="onSubmitEdit">保存</el-button>
      </template>
    </el-dialog>

    <!-- 重置密码弹窗 -->
    <el-dialog v-model="resetPwdVisible" title="重置实例密码" width="450px">
      <el-form :model="resetPwdForm" label-width="100px">
        <el-form-item label="新密码">
          <el-input v-model="resetPwdForm.password" type="password" show-password placeholder="请输入新密码（至少8位）" />
        </el-form-item>
        <el-alert type="warning" :closable="false">
          <template #default>
            <div>密码要求：8-30个字符，必须包含大写字母、小写字母、数字和特殊符号中的至少三种。</div>
            <div class="mt-1">重置密码后需要重启实例才能生效。</div>
          </template>
        </el-alert>
      </el-form>
      <template #footer>
        <el-button @click="resetPwdVisible = false">取消</el-button>
        <el-button type="primary" :loading="loading" @click="onSubmitResetPwd">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style lang="scss" scoped>
.container {
  padding: 16px;
}

.filter-card {
  margin-bottom: 16px;
}

.filter-row {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 16px;
}

.filter-item {
  display: flex;
  align-items: center;
}

.label {
  margin-right: 8px;
  white-space: nowrap;
  font-weight: 500;
  color: #606266;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.operations {
  display: flex;
  gap: 8px;
}
</style>
