<script lang="ts" setup>
import type { AwsAuthorizeSecurityGroupReq, AwsListReq } from "@@/apis/aws/type"
import { merchantQueryApi } from "@/pages/dashboard/apis"
import { authorizeSecurityGroup, listSecurityGroups, revokeSecurityGroup } from "@@/apis/aws"
import { getCloudAccountOptions } from "@@/apis/cloud_account"
import { getAwsRegions } from "@@/constants/aws-regions"

defineOptions({ name: "AwsSecurityGroup" })

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

// 可编辑的规则（协议、端口范围、CIDR）
const rule = reactive({
  ip_protocol: "tcp" as "tcp" | "udp" | "icmp" | "-1",
  from_port: 22,
  to_port: 22,
  cidr: "0.0.0.0/0"
})

// 规则编辑弹窗
const ruleDialogVisible = ref(false)
const ruleDialogMode = ref<"set" | "edit">("set")
const editingOriginalRule = ref<{ ip_protocol: string, from_port?: number, to_port?: number, cidrs: string[] } | null>(null)

async function confirmRuleDialog() {
  const from = Number(rule.from_port || 0)
  const to = Number(rule.to_port || 0)
  if (!validatePortRange(from, to)) return
  if (ruleDialogMode.value === "edit") {
    await performEditRule()
  } else {
    await performAddRule()
  }
}

async function performAddRule() {
  if (!currentGroupRow.value) {
    ruleDialogVisible.value = false
    return
  }
  const region = currentGroupRow.value.RegionId || currentGroupRow.value.Region || selectedRegions.value[0] || ""
  const groupId = currentGroupRow.value.GroupId
  if (!groupId || !region) {
    ElMessage.error("缺少必要参数")
    return
  }
  const authorizeReq: AwsAuthorizeSecurityGroupReq = {
    region_id: region,
    group_id: groupId,
    ip_protocol: rule.ip_protocol,
    from_port: Number(rule.from_port),
    to_port: Number(rule.to_port),
    cidr_blocks: [rule.cidr]
  }
  if (!validatePortRange(authorizeReq.from_port || 0, authorizeReq.to_port || 0)) return
  if (accountType.value === "system") authorizeReq.cloud_account_id = selectedCloudAccount.value!
  else authorizeReq.merchant_id = selectedMerchant.value!

  try {
    await authorizeSecurityGroup(authorizeReq)
    ElMessage.success("规则已新增")
    ruleDialogVisible.value = false
    // 刷新当前组规则
    await onQuery()
    const updated = rows.value.find((g: any) => g.GroupId === groupId)
    if (updated) showRules(updated)
  } catch (error) {
    console.error("新增规则失败", error)
  }
}

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
    const { data } = await listSecurityGroups(params)
    const list = data.list || []
    // 保留每条记录的 RegionId，避免后续授权缺少 region_id
    if (Array.isArray(list) && Array.isArray(list[0])) {
      const out: any[] = []
      ;(list as any[]).forEach((arr: any[], idx: number) => {
        ;(arr || []).forEach((item: any) => {
          const region = item.RegionId || item.Region || selectedRegions.value[idx] || ""
          out.push({ ...item, RegionId: region })
        })
      })
      rows.value = out
    } else {
      rows.value = (list as any[]).map((item: any) => ({
        ...item,
        RegionId: item.RegionId || item.Region || selectedRegions.value[0] || ""
      }))
    }
  } finally { loading.value = false }
}

function onSelectionChange(sel: any[]) {
  selection.value = sel
}

async function doAuthorize(row: any) {
  const req: AwsAuthorizeSecurityGroupReq = {
    region_id: row.RegionId || row.Region || selectedRegions.value[0] || "",
    group_id: row.GroupId,
    ip_protocol: rule.ip_protocol,
    from_port: Number(rule.from_port),
    to_port: Number(rule.to_port),
    cidr_blocks: [rule.cidr]
  }
  if (!validatePortRange(req.from_port || 0, req.to_port || 0)) return
  if (accountType.value === "system") req.cloud_account_id = selectedCloudAccount.value!
  else req.merchant_id = selectedMerchant.value!
  await authorizeSecurityGroup(req)
}

async function doRevoke(row: any) {
  const req: AwsAuthorizeSecurityGroupReq = {
    region_id: row.RegionId || row.Region || selectedRegions.value[0] || "",
    group_id: row.GroupId,
    ip_protocol: rule.ip_protocol,
    from_port: Number(rule.from_port),
    to_port: Number(rule.to_port),
    cidr_blocks: [rule.cidr]
  }
  if (!validatePortRange(req.from_port || 0, req.to_port || 0)) return
  if (accountType.value === "system") req.cloud_account_id = selectedCloudAccount.value!
  else req.merchant_id = selectedMerchant.value!
  await revokeSecurityGroup(req)
}

async function onAuthorize(row: any) {
  loading.value = true
  try {
    await doAuthorize(row)
    ElMessage.success("规则已授权")
  } finally { loading.value = false }
}

async function onRevoke(row: any) {
  loading.value = true
  try {
    await doRevoke(row)
    ElMessage.success("规则已撤销")
  } finally { loading.value = false }
}

async function onBatchAuthorize() {
  if (!selection.value.length) return
  loading.value = true
  try {
    for (const row of selection.value) await doAuthorize(row)
    ElMessage.success("批量授权完成")
  } finally { loading.value = false }
}

async function onBatchRevoke() {
  if (!selection.value.length) return
  loading.value = true
  try {
    for (const row of selection.value) await doRevoke(row)
    ElMessage.success("批量撤销完成")
  } finally { loading.value = false }
}

function validatePortRange(from: number, to: number) {
  if (rule.ip_protocol === "-1" || rule.ip_protocol === "icmp") return true
  if (Number.isNaN(from) || Number.isNaN(to)) {
    ElMessage.error("端口必须为数字")
    return false
  }
  if (from < 1 || to > 65535 || from > to) {
    ElMessage.error("端口范围需在 1-65535 且起始不大于结束")
    return false
  }
  return true
}

// 规则展示
const rulesDialogVisible = ref(false)
const currentRules = ref<any[]>([])
const currentGroupRow = ref<any | null>(null)
function showRules(row: any) {
  // 兼容 AWS 返回结构：IpPermissions
  const perms = row.IpPermissions || []
  currentRules.value = perms.map((p: any) => ({
    IpProtocol: p.IpProtocol,
    FromPort: p.FromPort,
    ToPort: p.ToPort,
    IpRanges: p.IpRanges || []
  }))
  currentGroupRow.value = row
  rulesDialogVisible.value = true
}

function onBeginEditRule(r: any) {
  // 预填充编辑
  rule.ip_protocol = (r.IpProtocol || "tcp") as any
  rule.from_port = Number(r.FromPort ?? 1)
  rule.to_port = Number(r.ToPort ?? 65535)
  const cidr = (r.IpRanges && r.IpRanges.length) ? (r.IpRanges[0].CidrIp || "0.0.0.0/0") : "0.0.0.0/0"
  rule.cidr = cidr
  editingOriginalRule.value = {
    ip_protocol: r.IpProtocol,
    from_port: r.FromPort,
    to_port: r.ToPort,
    cidrs: (r.IpRanges || []).map((x: any) => x.CidrIp).filter((x: any) => !!x)
  }
  ruleDialogMode.value = "edit"
  ruleDialogVisible.value = true
}

function onBeginAddRule() {
  // 重置为默认值
  rule.ip_protocol = "tcp"
  rule.from_port = 22
  rule.to_port = 22
  rule.cidr = "0.0.0.0/0"
  editingOriginalRule.value = null
  ruleDialogMode.value = "set"
  ruleDialogVisible.value = true
}

async function performEditRule() {
  if (!currentGroupRow.value) {
    ruleDialogVisible.value = false
    return
  }
  const region = currentGroupRow.value.RegionId || currentGroupRow.value.Region || selectedRegions.value[0] || ""
  const groupId = currentGroupRow.value.GroupId
  const original = editingOriginalRule.value
  if (!groupId || !region || !original) {
    return
  }
  // 1) 先撤销旧规则（全部 CIDR）
  const revokeReq: AwsAuthorizeSecurityGroupReq = {
    region_id: region,
    group_id: groupId,
    ip_protocol: original.ip_protocol as any,
    from_port: Number(original.from_port ?? 0),
    to_port: Number(original.to_port ?? 0),
    cidr_blocks: original.cidrs.length ? original.cidrs : [rule.cidr]
  }
  if (accountType.value === "system") revokeReq.cloud_account_id = selectedCloudAccount.value!
  else revokeReq.merchant_id = selectedMerchant.value!
  await revokeSecurityGroup(revokeReq)
  // 2) 授权新规则（当前 rule）
  const authorizeReq: AwsAuthorizeSecurityGroupReq = {
    region_id: region,
    group_id: groupId,
    ip_protocol: rule.ip_protocol,
    from_port: Number(rule.from_port),
    to_port: Number(rule.to_port),
    cidr_blocks: [rule.cidr]
  }
  if (!validatePortRange(authorizeReq.from_port || 0, authorizeReq.to_port || 0)) return
  if (accountType.value === "system") authorizeReq.cloud_account_id = selectedCloudAccount.value!
  else authorizeReq.merchant_id = selectedMerchant.value!
  await authorizeSecurityGroup(authorizeReq)
  ElMessage.success("规则已更新")
  ruleDialogVisible.value = false
  // 刷新当前组规则
  await onQuery()
  const updated = rows.value.find((g: any) => g.GroupId === groupId)
  if (updated) showRules(updated)
}

async function onDeleteRule(r: any) {
  if (!currentGroupRow.value) return
  const region = currentGroupRow.value.RegionId || currentGroupRow.value.Region || selectedRegions.value[0] || ""
  const groupId = currentGroupRow.value.GroupId
  const cidrs: string[] = (r.IpRanges || []).map((x: any) => x.CidrIp).filter((x: any) => !!x)
  const req: AwsAuthorizeSecurityGroupReq = {
    region_id: region,
    group_id: groupId,
    ip_protocol: r.IpProtocol,
    from_port: Number(r.FromPort ?? 0),
    to_port: Number(r.ToPort ?? 0),
    cidr_blocks: cidrs.length ? cidrs : ["0.0.0.0/0"]
  }
  if (accountType.value === "system") req.cloud_account_id = selectedCloudAccount.value!
  else req.merchant_id = selectedMerchant.value!
  await revokeSecurityGroup(req)
  ElMessage.success("规则已删除")
  // 刷新当前组规则
  await onQuery()
  const updated = rows.value.find((g: any) => g.GroupId === groupId)
  if (updated) showRules(updated)
}
</script>

<template>
  <div class="container">
    <div class="mb-4 flex gap-2 items-end flex-wrap">
      <el-select v-model="accountType" style="width: 140px">
        <el-option label="系统账号" value="system" />
        <el-option label="商户账号" value="merchant" />
      </el-select>
      <el-select v-if="accountType === 'system'" v-model="selectedCloudAccount" placeholder="选择云账号" filterable style="width: 240px">
        <el-option v-for="opt in cloudAccounts" :key="opt.value" :label="opt.label" :value="opt.value" />
      </el-select>
      <el-select
        v-else
        v-model="selectedMerchant"
        placeholder="搜索商户"
        filterable
        remote
        :remote-method="searchMerchant"
        style="width: 260px"
      >
        <el-option v-for="m in merchantOptions" :key="m.id" :label="m.name" :value="m.id" />
      </el-select>
      <el-select v-model="selectedRegions" multiple filterable collapse-tags collapse-tags-tooltip placeholder="选择Region" style="min-width: 360px; max-width: 640px">
        <el-option v-for="r in awsRegions" :key="r.id" :label="`${r.name} (${r.id})`" :value="r.id" />
      </el-select>
      <el-button type="primary" :loading="loading" @click="onQuery">查询</el-button>
      <el-button type="primary" :disabled="selection.length === 0" @click="onBatchAuthorize">批量授权</el-button>
      <el-button type="warning" :disabled="selection.length === 0" @click="onBatchRevoke">批量撤销</el-button>
    </div>

    <el-card class="table-card">
      <template #header>
        <div class="card-header">
          <span>安全组列表</span>
          <div class="operations">
            <el-button size="small" type="primary" @click="onQuery">刷新</el-button>
            <el-button size="small" type="primary" :disabled="selection.length === 0" @click="onBatchAuthorize">批量授权</el-button>
            <el-button size="small" type="warning" :disabled="selection.length === 0" @click="onBatchRevoke">批量撤销</el-button>
          </div>
        </div>
      </template>
      <el-table :data="rows" v-loading="loading" height="560" border @selection-change="onSelectionChange">
        <el-table-column type="selection" width="55" />
        <el-table-column prop="GroupId" label="安全组ID" min-width="220" />
        <el-table-column prop="GroupName" label="名称" min-width="180" />
        <el-table-column prop="Description" label="描述" min-width="240" />
        <el-table-column label="规则" min-width="140">
          <template #default="{ row }">
            <el-button link type="primary" @click="showRules(row)">规则管理</el-button>
          </template>
        </el-table-column>
        <el-table-column label="操作" fixed="right" min-width="160">
          <template #default="{ row }">
            <el-button link type="primary" @click="onAuthorize(row)">授权</el-button>
            <el-button link type="danger" @click="onRevoke(row)">撤销</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- 规则展示弹窗 -->
    <el-dialog v-model="rulesDialogVisible" title="安全组规则" width="780px">
      <div class="mb-3">
        <el-button type="primary" size="small" @click="onBeginAddRule">新增规则</el-button>
      </div>
      <el-table v-if="currentRules.length" :data="currentRules" border height="420">
        <el-table-column prop="IpProtocol" label="协议" width="120" />
        <el-table-column label="端口" width="140">
          <template #default="{ row }">
            <span v-if="row.IpProtocol === 'tcp' || row.IpProtocol === 'udp'">{{ row.FromPort }} - {{ row.ToPort }}</span>
            <span v-else>-</span>
          </template>
        </el-table-column>
        <el-table-column label="来源(CIDR)" min-width="300">
          <template #default="{ row }">
            <span v-if="row.IpRanges && row.IpRanges.length">{{ row.IpRanges.map((r: any) => r.CidrIp).join(', ') }}</span>
            <span v-else>-</span>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="180" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="onBeginEditRule(row)">编辑</el-button>
            <el-button link type="danger" @click="onDeleteRule(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
      <el-empty v-else description="暂无规则" />
    </el-dialog>

    <!-- 规则编辑弹窗 -->
    <el-dialog v-model="ruleDialogVisible" :title="ruleDialogMode === 'edit' ? '编辑安全组规则' : '新增安全组规则'" width="560px">
      <div class="rule-form">
        <el-form label-width="100px">
          <el-form-item label="协议">
            <el-select v-model="rule.ip_protocol" style="width: 180px">
              <el-option label="TCP" value="tcp" />
              <el-option label="UDP" value="udp" />
              <el-option label="ICMP" value="icmp" />
              <el-option label="全部(-1)" value="-1" />
            </el-select>
          </el-form-item>
          <el-form-item label="端口范围" v-if="rule.ip_protocol !== 'icmp' && rule.ip_protocol !== '-1'">
            <el-input-number v-model.number="rule.from_port" :min="1" :max="65535" placeholder="起始端口" />
            <span class="mx-2">-</span>
            <el-input-number v-model.number="rule.to_port" :min="1" :max="65535" placeholder="结束端口" />
          </el-form-item>
          <el-form-item label="CIDR">
            <el-input v-model="rule.cidr" placeholder="例如: 0.0.0.0/0" />
          </el-form-item>
        </el-form>
      </div>
      <template #footer>
        <el-button @click="ruleDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="confirmRuleDialog">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style lang="scss" scoped>
.container {
  padding: 16px;
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
.rule-form {
  padding: 8px 4px;
}
</style>
