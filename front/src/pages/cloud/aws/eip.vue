<script lang="ts" setup>
import type { AwsListReq, AwsOperateEipReq } from "@@/apis/aws/type"
import { merchantQueryApi } from "@/pages/dashboard/apis"
import { allocateEip, describeInstance, listEc2Instances, listEips, operateEip } from "@@/apis/aws"
import { getCloudAccountOptions } from "@@/apis/cloud_account"
import { getAwsRegions } from "@@/constants/aws-regions"

defineOptions({ name: "AwsEip" })

const loading = ref(false)
const rows = ref<any[]>([])

const accountType = ref<"merchant" | "system">("merchant")
const cloudAccounts = ref<{ value: number, label: string }[]>([])
const selectedCloudAccount = ref<number>()
const selectedMerchant = ref<number>()
const merchantOptions = ref<{ id: number, name: string }[]>([])

const awsRegions = getAwsRegions("cn")
const selectedRegions = ref<string[]>([])
const selection = ref<any[]>([])
const bindVisible = ref(false)
const bindForm = reactive({
  allocation_id: "",
  instance_id: "",
  private_ip: ""
})
const bindRegion = ref<string>("")
const instanceOptions = ref<Array<{ label: string, value: string }>>([])
const detailVisible = ref(false)
const detailRow = ref<any>(null)

async function fetchCloudAccounts() {
  const res = await getCloudAccountOptions("aws")
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
  const params: AwsListReq = { region_id: selectedRegions.value }
  if (accountType.value === "system") {
    if (!selectedCloudAccount.value) return ElMessage.warning("请选择系统云账号")
    params.cloud_account_id = selectedCloudAccount.value
  } else {
    if (!selectedMerchant.value) return ElMessage.warning("请选择商户")
    params.merchant_id = selectedMerchant.value
  }
  loading.value = true
  try {
    const { data } = await listEips(params)
    rows.value = (data.list || []).flat()
  } finally { loading.value = false }
}

function onSelectionChange(sel: any[]) {
  selection.value = sel
}

async function onRelease(row: any) {
  const req: AwsOperateEipReq = { region_id: row.Region || selectedRegions.value[0] || "", allocation_id: row.AllocationId, operation: "release" }
  if (accountType.value === "system") req.cloud_account_id = selectedCloudAccount.value!
  else req.merchant_id = selectedMerchant.value!
  loading.value = true
  try {
    await operateEip(req)
    ElMessage.success("已释放")
  } finally {
    loading.value = false
  }
}

async function onBatchRelease() {
  if (!selection.value.length) return
  loading.value = true
  try {
    for (const row of selection.value)
      await onRelease(row)
  } finally { loading.value = false }
}

async function onAllocate() {
  if (selectedRegions.value.length !== 1) return ElMessage.warning("请选择一个 Region")
  const data: any = { region_id: selectedRegions.value[0] }
  if (accountType.value === "system") data.cloud_account_id = selectedCloudAccount.value
  else data.merchant_id = selectedMerchant.value
  loading.value = true
  try {
    const res = await allocateEip(data)
    ElMessage.success(`已分配: ${res.data.allocation_id}`)
    onQuery()
  } finally { loading.value = false }
}

function getNameFromTags(tags: any[] | undefined): string {
  const arr = tags || []
  const t = arr.find((x: any) => x.Key === "Name")
  return t?.Value || ""
}

async function fetchInstancesForBind(query = "") {
  if (!bindRegion.value) return
  const params: AwsListReq = { region_id: [bindRegion.value] }
  if (accountType.value === "system") params.cloud_account_id = selectedCloudAccount.value
  else params.merchant_id = selectedMerchant.value
  const { data } = await listEc2Instances(params)
  const flat = (data.list || []).flat?.() || data.list || []
  const filtered = query
    ? flat.filter((ins: any) => {
        const name = getNameFromTags(ins.Tags)
        return (ins.InstanceId || "").includes(query) || name.includes(query)
      })
    : flat
  instanceOptions.value = filtered.map((ins: any) => ({
    value: ins.InstanceId,
    label: `${getNameFromTags(ins.Tags) || ins.InstanceId} (${ins.InstanceId})`
  }))
}

async function onShowBind(row: any) {
  bindForm.allocation_id = row.AllocationId
  bindForm.instance_id = ""
  bindForm.private_ip = ""
  bindRegion.value = row.Region || selectedRegions.value[0] || ""
  await fetchInstancesForBind()
  bindVisible.value = true
}

async function onBind() {
  if (!bindForm.allocation_id || !bindForm.instance_id) return
  const req: AwsOperateEipReq = { region_id: bindRegion.value || detailRow.value?.Region || selectedRegions.value[0] || "", allocation_id: bindForm.allocation_id, operation: "associate", instance_id: bindForm.instance_id, private_ip_address: bindForm.private_ip }
  if (accountType.value === "system") req.cloud_account_id = selectedCloudAccount.value!
  else req.merchant_id = selectedMerchant.value!
  loading.value = true
  try {
    await operateEip(req)
    ElMessage.success("已绑定")
    bindVisible.value = false
    onQuery()
  } finally { loading.value = false }
}

async function onUnbind(row: any) {
  const req: AwsOperateEipReq = { region_id: row.Region || selectedRegions.value[0] || "", allocation_id: row.AllocationId, operation: "disassociate" }
  if (accountType.value === "system") req.cloud_account_id = selectedCloudAccount.value!
  else req.merchant_id = selectedMerchant.value!
  loading.value = true
  try {
    await operateEip(req)
    ElMessage.success("已解绑")
    onQuery()
  } finally { loading.value = false }
}

async function onShowDetail(row: any) {
  detailRow.value = { InstanceId: row.InstanceId, Region: row.Region, Instance: null }
  detailVisible.value = true
  if (!row.InstanceId) return
  const params: any = { region_id: row.Region || selectedRegions.value[0] || "", instance_id: row.InstanceId }
  if (accountType.value === "system") params.cloud_account_id = selectedCloudAccount.value
  else params.merchant_id = selectedMerchant.value
  try {
    const { data } = await describeInstance(params)
    detailRow.value.Instance = {
      InstanceId: data.instance_id,
      InstanceType: data.instance_type,
      CpuOptions: { CoreCount: data.cpu },
      MemoryMiB: data.memory_mib,
      Tags: data.tags
    }
  } catch {}
}
</script>

<template>
  <div class="container">
    <el-card class="filter-card">
      <div class="filter-row">
        <div class="filter-item">
          <span class="label">账号类型：</span>
          <el-select v-model="accountType" style="width: 150px">
            <el-option label="系统账号" value="system" />
            <el-option label="商户账号" value="merchant" />
          </el-select>
        </div>
        <div v-if="accountType === 'system'" class="filter-item">
          <span class="label">云账号：</span>
          <el-select v-model="selectedCloudAccount" placeholder="选择云账号" filterable clearable style="width: 240px">
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
          <el-select v-model="selectedRegions" multiple filterable collapse-tags collapse-tags-tooltip placeholder="选择Region" style="min-width: 360px; max-width: 640px">
            <el-option v-for="r in awsRegions" :key="r.id" :label="`${r.name} (${r.id})`" :value="r.id" />
          </el-select>
        </div>
        <el-button type="primary" :loading="loading" @click="onQuery">查询</el-button>
        <el-button type="success" :disabled="selectedRegions.length !== 1" @click="onAllocate">分配EIP</el-button>
        <el-button type="danger" :disabled="selection.length === 0" @click="onBatchRelease">批量释放</el-button>
      </div>
    </el-card>

    <el-card class="table-card">
      <template #header>
        <div class="card-header">
          <span>弹性IP列表</span>
          <div class="operations">
            <el-button size="small" type="primary" @click="onQuery">刷新</el-button>
            <el-button size="small" type="success" :disabled="selectedRegions.length !== 1" @click="onAllocate">分配EIP</el-button>
            <el-button size="small" type="danger" :disabled="selection.length === 0" @click="onBatchRelease">批量释放</el-button>
          </div>
        </div>
      </template>

      <el-table :data="rows" v-loading="loading" height="560" border @selection-change="onSelectionChange">
        <el-table-column type="selection" width="55" />
        <el-table-column prop="PublicIp" label="公网IP" min-width="160" />
        <el-table-column prop="AllocationId" label="AllocationId" min-width="220" />
        <el-table-column prop="InstanceId" label="绑定实例" min-width="220" />
        <el-table-column prop="Region" label="区域" min-width="140" />
        <el-table-column label="操作" fixed="right" min-width="240">
          <template #default="{ row }">
            <el-button
              v-if="!row.InstanceId"
              link
              type="primary"
              @click="onShowBind(row)"
            >绑定实例</el-button>
            <el-button
              v-else
              link
              type="warning"
              @click="onUnbind(row)"
            >解绑</el-button>
            <el-button link type="primary" v-if="row.InstanceId" @click="onShowDetail(row)">实例详情</el-button>
            <el-button link type="danger" @click="onRelease(row)">释放</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- 绑定弹窗 -->
    <el-dialog v-model="bindVisible" title="绑定实例" width="520px">
      <el-form label-width="120px">
        <el-form-item label="AllocationId">
          <el-input v-model="bindForm.allocation_id" disabled />
        </el-form-item>
        <el-form-item label="InstanceId">
          <el-select
            v-model="bindForm.instance_id"
            filterable
            remote
            reserve-keyword
            :remote-method="fetchInstancesForBind"
            placeholder="选择要绑定的实例"
            style="width: 360px"
          >
            <el-option v-for="opt in instanceOptions" :key="opt.value" :label="opt.label" :value="opt.value" />
          </el-select>
        </el-form-item>
        <el-form-item label="Private IP">
          <el-input v-model="bindForm.private_ip" placeholder="可选: 10.x.x.x" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="bindVisible = false">取消</el-button>
        <el-button type="primary" :loading="loading" @click="onBind">确定</el-button>
      </template>
    </el-dialog>

    <!-- 实例详情弹窗（简版） -->
    <el-dialog v-model="detailVisible" title="实例详情" width="680px">
      <el-descriptions v-if="detailRow?.Instance" :column="2" border>
        <el-descriptions-item label="实例ID">{{ detailRow.Instance.InstanceId || detailRow.InstanceId }}</el-descriptions-item>
        <el-descriptions-item label="名称">{{ (detailRow.Instance.Tags || []).find((t:any) => t.Key === 'Name')?.Value || '-' }}</el-descriptions-item>
        <el-descriptions-item label="规格">{{ detailRow.Instance.InstanceType || '-' }}</el-descriptions-item>
        <el-descriptions-item label="CPU">{{ detailRow.Instance.CpuOptions?.CoreCount || '-' }}</el-descriptions-item>
        <el-descriptions-item label="内存">{{ detailRow.Instance.MemoryMiB ? `${(detailRow.Instance.MemoryMiB / 1024).toFixed(1)} GB` : '-' }}</el-descriptions-item>
      </el-descriptions>
      <el-empty v-else description="暂无实例信息（请后端补充 DescribeInstances 聚合）" />
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
.table-card {
  margin-top: 8px;
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
