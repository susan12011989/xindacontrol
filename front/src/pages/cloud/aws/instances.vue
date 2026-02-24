<script lang="ts" setup>
import type { AwsListReq, AwsModifyEc2InstanceReq, AwsOperateEc2Req } from "@@/apis/aws/type"
import { merchantQueryApi } from "@/pages/dashboard/apis"
import { listEc2Instances, listSecurityGroupOptions, listVolumes, listVolumeUsage, modifyEc2Instance, operateEc2Instance, resizeVolumeStream } from "@@/apis/aws"
import { getCloudAccountOptions } from "@@/apis/cloud_account"
import { getAwsRegions } from "@@/constants/aws-regions"
import { ArrowDown, Delete, Edit, RefreshRight, VideoPause, VideoPlay } from "@element-plus/icons-vue"
import { useRouter } from "vue-router"

defineOptions({ name: "AwsInstances" })

const loading = ref(false)
const rows = ref<any[]>([])

const router = useRouter()

// 账号类型与账号/商户选择（保持与阿里云交互一致）
const accountType = ref<"merchant" | "system">("merchant")
const cloudAccounts = ref<{ value: number, label: string }[]>([])
const selectedCloudAccount = ref<number>()
const selectedMerchant = ref<number>()
const merchantOptions = ref<{ id: number, name: string }[]>([])

// 统一调用常量（中文）
const awsRegions = getAwsRegions("cn")
const selectedRegions = ref<string[]>([])

const selection = ref<any[]>([])

// 编辑属性对话框
const editVisible = ref(false)
const editForm = reactive<AwsModifyEc2InstanceReq>({
  region_id: "",
  instance_id: "",
  name: "",
  description: "",
  tags: {},
  security_group_ids: []
})
const tagPairs = ref<Array<{ key: string, value: string }>>([])
const sgOptions = ref<Array<{ group_id: string, group_name: string }>>([])

// 实例卷缓存 { [instanceId]: AwsVolumeItem[] }
const volumesMap = ref<Record<string, Array<{ volume_id: string, device_name: string, size_gib: number, volume_type: string, state: string, used_bytes?: number, avail_bytes?: number }>>>({})
const volumesLoading = ref<Record<string, boolean>>({})

async function onShowVolumes(row: any) {
  const instanceId = row.InstanceId as string
  if (!instanceId) return
  const region = (row.Placement?.AvailabilityZone || "").slice(0, -1)
  if (!region) return
  const params: any = { region_id: region, instance_id: instanceId }
  if (accountType.value === "system") params.cloud_account_id = selectedCloudAccount.value
  else params.merchant_id = selectedMerchant.value
  volumesLoading.value[instanceId] = true
  try {
    const { data } = await listVolumes(params)
    volumesMap.value[instanceId] = (data.list || []).map((v: any) => ({
      volume_id: v.volume_id,
      device_name: v.device_name,
      size_gib: v.size_gib,
      volume_type: v.volume_type,
      state: v.state
    }))
  } finally {
    volumesLoading.value[instanceId] = false
  }
}

function getCpuCores(row: any): string {
  // 使用后端注入的 vcpu 字段（vCPU = CoreCount × ThreadsPerCore）
  const vcpu = (row.vcpu ?? row.VCpu) as number | undefined
  return typeof vcpu === "number" && vcpu > 0 ? String(vcpu) : "-"
}

function getMemoryGiB(row: any): string {
  // 使用后端注入的 memory_mib 字段
  const mib = (row.memory_mib ?? row.MemoryMiB) as number | undefined
  if (!mib || mib <= 0) return "-"
  const gib = mib / 1024
  return gib % 1 === 0 ? String(gib) : gib.toFixed(1)
}

function getPublicIp(row: any): string {
  if (row.PublicIpAddress) return row.PublicIpAddress as string
  const enis: any[] = row.NetworkInterfaces || []
  const withPub = enis.find((n: any) => n.Association && n.Association.PublicIp)
  return withPub?.Association?.PublicIp || "-"
}

function getPrivateIp(row: any): string {
  if (row.PrivateIpAddress) return row.PrivateIpAddress as string
  const enis: any[] = row.NetworkInterfaces || []
  const withPri = enis.find((n: any) => n.PrivateIpAddress)
  return withPri?.PrivateIpAddress || "-"
}

function getStateText(name: string | undefined): string {
  switch ((name || "").toLowerCase()) {
    case "running":
      return "运行中"
    case "stopped":
      return "已停止"
    case "stopping":
      return "停止中"
    case "pending":
      return "初始化中"
    case "shutting-down":
      return "释放中"
    case "terminated":
      return "已终止"
    default:
      return name || "-"
  }
}

function getStateTagType(name: string | undefined): "success" | "info" | "warning" | "danger" | undefined {
  switch ((name || "").toLowerCase()) {
    case "running":
      return "success"
    case "stopped":
      return "info"
    case "stopping":
    case "pending":
    case "shutting-down":
      return "warning"
    case "terminated":
      return "danger"
    default:
      return undefined
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
    const { data } = await listEc2Instances(params)
    rows.value = (data.list || []).flat()
  } finally {
    loading.value = false
  }
}

function onSelectionChange(sel: any[]) {
  selection.value = sel
}

async function onOperate(row: any, operation: AwsOperateEc2Req["operation"]) {
  const region = (row.Placement?.AvailabilityZone || "").slice(0, -1)
  if (!region) return
  const data: AwsOperateEc2Req = { region_id: region, instance_id: row.InstanceId, operation }
  if (accountType.value === "system") data.cloud_account_id = selectedCloudAccount.value!
  else data.merchant_id = selectedMerchant.value!
  loading.value = true
  try {
    await operateEc2Instance(data)
    ElMessage.success("操作已提交")
  } finally {
    loading.value = false
  }
}

async function onBatchOperate(operation: AwsOperateEc2Req["operation"]) {
  if (selection.value.length === 0) return
  // 按操作类型过滤可操作的实例
  const validStates: Record<string, string[]> = {
    start: ["stopped"],
    stop: ["running"],
    reboot: ["running"],
    terminate: ["running", "stopped"]
  }
  const allowed = validStates[operation] || []
  const eligible = selection.value.filter(row => {
    const state = (row.State?.Name || "").toLowerCase()
    return allowed.includes(state)
  })
  const skipped = selection.value.length - eligible.length
  if (eligible.length === 0) {
    ElMessage.warning(`所选实例均不在可${operation === "start" ? "启动" : operation === "stop" ? "停止" : operation === "reboot" ? "重启" : "销毁"}状态，已跳过`)
    return
  }
  loading.value = true
  let success = 0
  let failed = 0
  try {
    for (const row of eligible) {
      const region = (row.Placement?.AvailabilityZone || "").slice(0, -1)
      const data: AwsOperateEc2Req = { region_id: region, instance_id: row.InstanceId, operation }
      if (accountType.value === "system") data.cloud_account_id = selectedCloudAccount.value!
      else data.merchant_id = selectedMerchant.value!
      try {
        await operateEc2Instance(data)
        success++
      } catch {
        failed++
      }
    }
    const parts: string[] = []
    if (success > 0) parts.push(`${success} 台成功`)
    if (failed > 0) parts.push(`${failed} 台失败`)
    if (skipped > 0) parts.push(`${skipped} 台跳过(状态不符)`)
    ElMessage({ type: failed > 0 ? "warning" : "success", message: `批量操作完成：${parts.join("，")}` })
  } finally {
    loading.value = false
  }
}

function onGoCreate() {
  router.push({ name: "AwsInstancesCreate" })
}

function getInstanceName(row: any): string {
  const tags: Array<{ Key?: string, Value?: string }> = row.Tags || []
  const nameTag = tags.find(t => t.Key === "Name")
  return nameTag?.Value || ""
}

function onShowEdit(row: any) {
  const region = (row.Placement?.AvailabilityZone || "").slice(0, -1)
  if (!region) return
  editForm.region_id = region
  editForm.instance_id = row.InstanceId
  // 从标签预填充
  const tagsArr: Array<{ Key?: string, Value?: string }> = (row.Tags || [])
  const pairs: Array<{ key: string, value: string }> = []
  let name = ""
  let description = ""
  for (const t of tagsArr) {
    const k = (t.Key || "") as string
    const v = (t.Value || "") as string
    if (k === "Name") name = v
    else if (k === "Description") description = v
    else if (k) pairs.push({ key: k, value: v })
  }
  editForm.name = name
  editForm.description = description
  tagPairs.value = pairs.length ? pairs : [{ key: "", value: "" }]
  // 预加载安全组选项
  preloadSecurityGroups().then(() => {
    // 预选中：尝试从实例属性中提取当前安全组（可选）
    const groups: Array<{ GroupId?: string }> = row.SecurityGroups || []
    editForm.security_group_ids = groups.map(g => g.GroupId || "").filter(Boolean)
  })
  editVisible.value = true
}

function onAddTag() {
  tagPairs.value.push({ key: "", value: "" })
}

function onRemoveTag(idx: number) {
  tagPairs.value.splice(idx, 1)
  if (tagPairs.value.length === 0) tagPairs.value.push({ key: "", value: "" })
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
  // 组装 tags
  const tags: Record<string, string> = {}
  for (const p of tagPairs.value) {
    const k = (p.key || "").trim()
    if (!k) continue
    tags[k] = p.value || ""
  }
  editForm.tags = tags
  const loadingMsg = ElMessage.info({ message: "保存中...", duration: 800 })
  try {
    await modifyEc2Instance(editForm)
    ElMessage.success("已保存")
    editVisible.value = false
    onQuery()
  } finally {
    loadingMsg.close && loadingMsg.close()
  }
}

async function preloadSecurityGroups() {
  const base: any = { region_id: editForm.region_id }
  if (accountType.value === "system") base.cloud_account_id = selectedCloudAccount.value
  else base.merchant_id = selectedMerchant.value
  try {
    const { data } = await listSecurityGroupOptions(base)
    sgOptions.value = data.list || []
  } catch {
    sgOptions.value = []
  }
}

// 扩容磁盘对话框
const resizeVisible = ref(false)
const resizeForm = reactive<{ sizeGiB: number | undefined, expandFs: boolean }>({ sizeGiB: undefined, expandFs: true })
const resizeCtx = reactive<{ region: string, instanceId: string, deviceName?: string }>({ region: "", instanceId: "" })
const resizeStreaming = ref(false)
const resizeLogs = ref<string[]>([])
let cancelResizeStream: (() => void) | null = null

function onShowResize(row: any) {
  const region = (row.Placement?.AvailabilityZone || "").slice(0, -1)
  if (!region) return
  resizeCtx.region = region
  resizeCtx.instanceId = row.InstanceId
  resizeForm.sizeGiB = undefined
  resizeForm.expandFs = true
  resizeVisible.value = true
}

async function onSubmitResize() {
  if (!resizeForm.sizeGiB || resizeForm.sizeGiB <= 0) {
    return ElMessage.warning("请输入新容量(GB)")
  }
  const data: any = {
    region_id: resizeCtx.region,
    instance_id: resizeCtx.instanceId,
    new_size_gib: Number(resizeForm.sizeGiB),
    expand_fs: resizeForm.expandFs
  }
  if (accountType.value === "system") data.cloud_account_id = selectedCloudAccount.value
  else data.merchant_id = selectedMerchant.value
  // 开始流式
  resizeLogs.value = []
  resizeStreaming.value = true
  if (cancelResizeStream) cancelResizeStream()
  cancelResizeStream = resizeVolumeStream(
    data,
    (chunk: any, isComplete?: boolean) => {
      try {
        if (isComplete) {
          resizeStreaming.value = false
          if (chunk && chunk.success) {
            ElMessage.success(chunk.message || "扩容完成")
            resizeVisible.value = false
            onQuery()
          } else {
            ElMessage.error((chunk && chunk.message) || "扩容失败")
          }
          return
        }
        if (chunk && typeof chunk === "object") {
          const line = JSON.stringify(chunk)
          resizeLogs.value.push(line)
        } else if (typeof chunk === "string") {
          resizeLogs.value.push(chunk)
        }
      } catch {
        // 忽略解析错误
      }
    },
    (err: any) => {
      console.error(err)
      resizeStreaming.value = false
      ElMessage.error("扩容出错")
    }
  )
}

function formatBytes(bytes: number | undefined): string {
  if (!bytes && bytes !== 0) return "-"
  const units = ["B", "KB", "MB", "GB", "TB"]
  let v = Number(bytes)
  let u = 0
  while (v >= 1024 && u < units.length - 1) {
    v /= 1024
    u++
  }
  return `${Math.round(v * 10) / 10} ${units[u]}`
}

async function onRefreshUsage(row: any) {
  const instanceId = row.InstanceId as string
  const region = (row.Placement?.AvailabilityZone || "").slice(0, -1)
  if (!instanceId || !region) return
  const params: any = { region_id: region, instance_id: instanceId }
  if (accountType.value === "system") params.cloud_account_id = selectedCloudAccount.value
  else params.merchant_id = selectedMerchant.value
  try {
    const { data } = await listVolumeUsage(params)
    const usages: Array<{ source: string, mountpoint?: string, used_bytes: number, avail_bytes: number }> = data.list || []
    const list = volumesMap.value[instanceId] || []
    function mapXvdToNvmePrefix(dev: string): string | null {
      if (!dev) return null
      const m = dev.match(/^\/dev\/(xvd|sd)([a-z])/)
      if (!m) return null
      const letter = m[2]
      const idx = letter.charCodeAt(0) - "a".charCodeAt(0)
      if (idx < 0) return null
      return `/dev/nvme${idx}n1`
    }
    for (const u of usages) {
      let found = list.find(v => (u.source && v.device_name && (u.source === v.device_name || u.source.startsWith(v.device_name))))
      if (!found) {
        for (const v of list) {
          const nvmePrefix = mapXvdToNvmePrefix(v.device_name)
          if (nvmePrefix && u.source && u.source.startsWith(nvmePrefix)) {
            found = v
            break
          }
        }
      }
      if (!found && u.mountpoint === "/") {
        found = list.find(v => v.device_name === "/dev/xvda" || v.device_name === "/dev/sda")
      }
      if (found) {
        ;(found as any).used_bytes = (u as any).used_bytes
        ;(found as any).avail_bytes = (u as any).avail_bytes
      }
    }
    volumesMap.value[instanceId] = [...list]
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
            <el-option v-for="r in awsRegions" :key="r.id" :label="`${r.name} (${r.id})`" :value="r.id" />
          </el-select>
        </div>
        <el-button type="primary" :loading="loading" @click="onQuery">查询</el-button>
        <el-button type="success" @click="onGoCreate">创建实例</el-button>
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
        <el-table-column label="名称" min-width="160">
          <template #default="{ row }">
            {{ getInstanceName(row) || '-' }}
          </template>
        </el-table-column>
        <el-table-column label="区域" min-width="120">
          <template #default="{ row }">
            {{ (row.Placement?.AvailabilityZone || '').slice(0, -1) || '-' }}
          </template>
        </el-table-column>
        <el-table-column prop="InstanceType" label="实例规格" min-width="140" />
        <el-table-column label="磁盘" min-width="180">
          <template #default="{ row }">
            <el-popover
              placement="top"
              width="360"
              trigger="click"
              @show="onShowVolumes(row)"
            >
              <template #reference>
                <el-button text size="small">查看</el-button>
              </template>
              <div style="min-height: 60px">
                <el-skeleton v-if="volumesLoading[row.InstanceId] && !volumesMap[row.InstanceId]" :rows="2" animated />
                <template v-else>
                  <div style="display:flex; justify-content: flex-end; margin-bottom:6px">
                    <el-button size="small" text @click="onRefreshUsage(row)">刷新使用率</el-button>
                  </div>
                  <div v-if="(volumesMap[row.InstanceId] || []).length === 0" style="color:#909399">无卷信息</div>
                  <el-scrollbar v-else max-height="240">
                    <ul style="padding-left:16px; margin:0">
                      <li v-for="v in volumesMap[row.InstanceId]" :key="v.volume_id" style="margin:8px 0">
                        <div>
                          <strong>{{ v.device_name || '-' }}</strong>
                          <span style="margin-left:6px">{{ v.size_gib }} GiB</span>
                          <el-tag size="small" style="margin-left:6px">{{ v.volume_type }}</el-tag>
                          <el-tag size="small" type="info" style="margin-left:6px">{{ v.volume_id }}</el-tag>
                        </div>
                        <div style="color:#606266; font-size:12px; margin-top:2px">
                          已用/可用：{{ formatBytes(v.used_bytes) }} / {{ formatBytes(v.avail_bytes) }}
                        </div>
                        <div style="color:#909399; font-size:12px">状态：{{ v.state }}</div>
                      </li>
                    </ul>
                  </el-scrollbar>
                </template>
              </div>
            </el-popover>
          </template>
        </el-table-column>
        <el-table-column label="IP" min-width="200">
          <template #default="{ row }">
            公网: {{ getPublicIp(row) }} / 私网: {{ getPrivateIp(row) }}
          </template>
        </el-table-column>
        <el-table-column label="CPU/内存" min-width="140">
          <template #default="{ row }">
            {{ getCpuCores(row) }} / {{ getMemoryGiB(row) === '-' ? '-' : `${getMemoryGiB(row)} GB` }}
          </template>
        </el-table-column>
        <el-table-column label="状态" min-width="120">
          <template #default="{ row }">
            <el-tag :type="getStateTagType(row.State?.Name)">
              {{ getStateText(row.State?.Name) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="Placement.AvailabilityZone" label="可用区" min-width="140" />
        <el-table-column label="操作" fixed="right" min-width="140">
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
                  <el-dropdown-item @click="onShowResize(row)">
                    <el-icon><Edit /></el-icon> 扩容磁盘
                  </el-dropdown-item>
                  <el-dropdown-item @click="onOperate(row, 'start')">
                    <el-icon><VideoPlay /></el-icon> 启动
                  </el-dropdown-item>
                  <el-dropdown-item @click="onOperate(row, 'stop')">
                    <el-icon><VideoPause /></el-icon> 停止
                  </el-dropdown-item>
                  <el-dropdown-item @click="onOperate(row, 'reboot')">
                    <el-icon><RefreshRight /></el-icon> 重启
                  </el-dropdown-item>
                  <el-dropdown-item divided @click="onOperate(row, 'terminate')">
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
    <el-dialog v-model="editVisible" title="编辑实例属性" width="520px">
      <el-form :model="editForm" label-width="100px">
        <el-form-item label="名称">
          <el-input v-model="editForm.name" placeholder="实例显示名(Name)" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="editForm.description" placeholder="Description 标签" />
        </el-form-item>
        <el-form-item label="安全组">
          <el-select v-model="editForm.security_group_ids" multiple filterable placeholder="可选" style="width: 100%">
            <el-option v-for="g in sgOptions" :key="g.group_id" :label="`${g.group_name} (${g.group_id})`" :value="g.group_id" />
          </el-select>
        </el-form-item>
        <el-form-item label="标签">
          <div style="width: 100%">
            <div v-for="(p, idx) in tagPairs" :key="idx" class="tag-row">
              <el-input v-model="p.key" placeholder="Key" style="width: 40%" />
              <span class="mx-2">=</span>
              <el-input v-model="p.value" placeholder="Value" style="width: 40%" />
              <el-button link type="danger" @click="onRemoveTag(idx)">移除</el-button>
            </div>
            <el-button text type="primary" @click="onAddTag">+ 新增标签</el-button>
          </div>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="editVisible = false">取消</el-button>
        <el-button type="primary" @click="onSubmitEdit">保存</el-button>
      </template>
    </el-dialog>

    <!-- 扩容磁盘弹窗 -->
    <el-dialog v-model="resizeVisible" title="扩容磁盘（EBS）" width="520px">
      <el-form label-width="120px">
        <el-form-item label="新容量(GB)">
          <el-input-number v-model="resizeForm.sizeGiB" :min="1" :step="10" :precision="0" placeholder="例如 50" />
        </el-form-item>
        <el-form-item label="扩展文件系统">
          <el-switch v-model="resizeForm.expandFs" />
          <el-text type="info" style="margin-left: 8px">通过 SSM 在实例内执行 growpart/xfs_growfs/resize2fs</el-text>
        </el-form-item>
        <el-form-item label="执行日志">
          <el-scrollbar max-height="200">
            <pre style="white-space: pre-wrap; word-break: break-all; margin: 0">{{ resizeLogs.join('\n') || '—' }}</pre>
          </el-scrollbar>
          <el-tag v-if="resizeStreaming" size="small" type="info" style="margin-left: 8px">执行中...</el-tag>
        </el-form-item>
        <el-alert type="info" :closable="false">
          <template #default>
            <div>扩容会先修改 EBS 卷大小，然后在实例内扩展分区/文件系统。</div>
            <div>参考：
              <a href="https://docs.aws.amazon.com/zh_cn/ebs/latest/userguide/requesting-ebs-volume-modifications.html" target="_blank">申请 EBS 卷修改</a>
              、
              <a href="https://docs.aws.amazon.com/zh_cn/ebs/latest/userguide/recognize-expanded-volume-linux.html" target="_blank">扩展文件系统</a>
            </div>
          </template>
        </el-alert>
      </el-form>
      <template #footer>
        <el-button :disabled="resizeStreaming" @click="resizeVisible = false">{{ resizeStreaming ? '执行中...' : '取消' }}</el-button>
        <el-button type="primary" :loading="resizeStreaming" :disabled="resizeStreaming" @click="onSubmitResize">提交</el-button>
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

.tag-row {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-bottom: 8px;
}
</style>
