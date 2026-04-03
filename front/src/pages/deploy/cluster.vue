<script lang="ts" setup>
import type { ClusterNodeInfo, ClusterTopologyResp, DeployNodeReq, DeployTSDDResp } from "@@/apis/deploy/type"
import type { MerchantResp } from "@@/apis/merchant/type"
import type { MerchantGostServerResp, MerchantOssConfigResp } from "@@/apis/merchant/type"
import type { ServerResp } from "@@/apis/deploy/type"
import {
  getClusterTopology, deployNodeTracked, retryDeployApi, getServerList,
  batchHealthCheck, batchServiceAction
} from "@@/apis/deploy"
import {
  getMerchantList, getMerchantGostServers, createMerchantGostServer,
  deleteMerchantGostServer, getMerchantOssConfigs, createMerchantOssConfig,
  deleteMerchantOssConfig, syncMerchantGostIP
} from "@@/apis/merchant"
import { getCloudAccountList } from "@@/apis/cloud_account"
import { getAwsRegions } from "@@/constants/aws-regions"

defineOptions({ name: "DeployCluster" })

// ========== 商户选择 ==========
const merchantList = ref<MerchantResp[]>([])
const selectedMerchantId = ref<number | undefined>(undefined)
const loading = ref(false)
const topology = ref<ClusterTopologyResp | null>(null)

async function loadMerchants() {
  const res = await getMerchantList({ page: 1, size: 2000 })
  merchantList.value = Array.isArray(res.data?.list) ? res.data.list : []
  if (merchantList.value.length > 0 && !selectedMerchantId.value) {
    selectedMerchantId.value = merchantList.value[0].id
  }
}

async function loadTopology() {
  if (!selectedMerchantId.value) return
  loading.value = true
  try {
    const res = await getClusterTopology(selectedMerchantId.value)
    topology.value = res.data
  } catch { topology.value = null }
  finally { loading.value = false }
}

// ========== 服务器列表 ==========
const allServers = ref<ServerResp[]>([])
async function loadServers() {
  const res = await getServerList({ page: 1, size: 5000 })
  allServers.value = Array.isArray(res.data?.list) ? res.data.list : []
}
const merchantServers = computed(() => allServers.value.filter(s => s.server_type === 1))
const systemGostServers = computed(() => allServers.value.filter(s => s.server_type === 2))

// ========== 节点分组 ==========
const allinoneNodes = computed(() => topology.value?.nodes.filter(n => n.role === "all") || [])
const dbNodes = computed(() => topology.value?.nodes.filter(n => n.role === "db") || [])
const minioNodes = computed(() => topology.value?.nodes.filter(n => n.role === "minio") || [])
const appNodes = computed(() => topology.value?.nodes.filter(n => n.role === "app" || n.role === "api") || [])

// ========== 部署步骤引导 ==========
const hasDB = computed(() => dbNodes.value.length > 0 || allinoneNodes.value.length > 0)
const hasMinio = computed(() => minioNodes.value.length > 0 || allinoneNodes.value.length > 0)
const hasApp = computed(() => appNodes.value.length > 0 || allinoneNodes.value.length > 0)
const currentStep = computed(() => {
  if (!hasDB.value) return 0
  if (!hasMinio.value) return 1
  if (!hasApp.value) return 2
  return 3
})
const stepMessage = computed(() => {
  if (allinoneNodes.value.length > 0) return `已部署 All-in-One 节点，可部署更多 App 节点扩容`
  if (!hasDB.value) return "第 1 步：部署 DB 节点（MySQL + Redis）"
  if (!hasMinio.value) return "第 2 步：部署 MinIO 节点（对象存储）"
  if (!hasApp.value) return "第 3 步：部署首个 App 节点"
  return `已部署 ${appNodes.value.length} 个 App 节点，可继续扩容`
})

// ========== 健康检查 ==========
const healthMap = ref<Record<number, string>>({})
const healthLoading = ref(false)

async function handleHealthCheck() {
  const ids = (topology.value?.nodes || []).filter(n => n.server_id > 0).map(n => n.server_id)
  if (!ids.length) { ElMessage.warning("当前商户无集群节点"); return }
  healthLoading.value = true
  try {
    const res = await batchHealthCheck({ server_ids: ids })
    for (const r of (res.data?.results || [])) {
      healthMap.value[r.server_id] = r.status
    }
    ElMessage.success("健康检查完成")
  } catch (e: any) { ElMessage.error(e.message || "检查失败") }
  finally { healthLoading.value = false }
}

function healthTag(serverId: number) {
  const s = healthMap.value[serverId]
  if (!s) return { type: "info" as const, text: "未检查" }
  if (s === "healthy") return { type: "success" as const, text: "健康" }
  if (s === "partial") return { type: "warning" as const, text: "部分异常" }
  return { type: "danger" as const, text: "异常" }
}

// ========== GOST 同步 ==========
const syncLoading = ref(false)
const syncResults = ref<any[]>([])
const syncDialogVisible = ref(false)

async function handleGostSync() {
  if (!selectedMerchantId.value) return
  if (appNodes.value.length === 0 && allinoneNodes.value.length === 0) {
    ElMessage.warning("当前商户无 App 节点，无需同步 GOST"); return
  }
  syncLoading.value = true
  try {
    const res = await syncMerchantGostIP(selectedMerchantId.value)
    syncResults.value = res.data?.results || []
    syncDialogVisible.value = true
    const ok = syncResults.value.filter((r: any) => r.success).length
    const fail = syncResults.value.filter((r: any) => !r.success).length
    fail === 0
      ? ElMessage.success(`GOST 同步完成：${ok} 台成功，配置已自动持久化`)
      : ElMessage.warning(`GOST 同步完成：${ok} 成功 / ${fail} 失败`)
  } catch (e: any) { ElMessage.error(e.message || "GOST 同步失败") }
  finally { syncLoading.value = false }
}

// ========== GOST 重启 ==========
const restartLoading = ref(false)

async function handleGostRestart() {
  if (gostServers.value.length === 0) {
    ElMessage.warning("当前商户无关联的 GOST 服务器"); return
  }
  try {
    await ElMessageBox.confirm("确认重启所有关联的 GOST 服务器？", "重启 GOST", { type: "warning" })
  } catch { return }
  restartLoading.value = true
  try {
    const ids = gostServers.value.map(s => s.server_id)
    const res = await batchServiceAction({ server_ids: ids, service_name: "gost", action: "restart" })
    const d = res.data as any
    d.fail_count === 0
      ? ElMessage.success(`GOST 重启完成：${d.success_count} 台成功`)
      : ElMessage.warning(`GOST 重启完成：${d.success_count} 成功 / ${d.fail_count} 失败`)
  } catch (e: any) { ElMessage.error(e.message || "GOST 重启失败") }
  finally { restartLoading.value = false }
}

// ========== GOST 服务器管理 ==========
const gostServers = ref<MerchantGostServerResp[]>([])
const gostLoading = ref(false)
const addGostDialogVisible = ref(false)
const addGostForm = ref({ server_id: undefined as number | undefined, is_primary: 0 })
const addGostLoading = ref(false)

const boundGostIds = computed(() => new Set(gostServers.value.map(s => s.server_id)))
const availableGostServers = computed(() => systemGostServers.value.filter(s => !boundGostIds.value.has(s.id)))

async function loadGostServers() {
  if (!selectedMerchantId.value) return
  gostLoading.value = true
  try {
    const res = await getMerchantGostServers(selectedMerchantId.value)
    gostServers.value = Array.isArray(res.data) ? res.data : []
  } catch { gostServers.value = [] }
  finally { gostLoading.value = false }
}

function openAddGost() {
  addGostForm.value = { server_id: undefined, is_primary: gostServers.value.length === 0 ? 1 : 0 }
  addGostDialogVisible.value = true
}

async function handleAddGost() {
  if (!selectedMerchantId.value || !addGostForm.value.server_id) return
  addGostLoading.value = true
  try {
    await createMerchantGostServer(selectedMerchantId.value, {
      server_id: addGostForm.value.server_id, is_primary: addGostForm.value.is_primary, status: 1
    })
    ElMessage.success("关联成功")
    addGostDialogVisible.value = false
    loadGostServers()
  } catch (e: any) { ElMessage.error(e.message || "关联失败") }
  finally { addGostLoading.value = false }
}

async function handleRemoveGost(row: MerchantGostServerResp) {
  try {
    await ElMessageBox.confirm(`确定解除与 ${row.server_name || row.server_host} 的关联？`, "解除关联", { type: "warning" })
    await deleteMerchantGostServer(row.id)
    ElMessage.success("已解除关联")
    loadGostServers()
  } catch {}
}

// ========== OSS 配置管理 ==========
const ossConfigs = ref<MerchantOssConfigResp[]>([])
const ossLoading = ref(false)
const addOssDialogVisible = ref(false)
const addOssForm = ref({ cloud_account_id: 0, name: "", bucket: "", region: "", is_default: 0 })
const addOssLoading = ref(false)

async function loadOssConfigs() {
  if (!selectedMerchantId.value) return
  ossLoading.value = true
  try {
    const res = await getMerchantOssConfigs(selectedMerchantId.value)
    ossConfigs.value = Array.isArray(res.data) ? res.data : []
  } catch { ossConfigs.value = [] }
  finally { ossLoading.value = false }
}

async function handleRemoveOss(row: MerchantOssConfigResp) {
  try {
    await ElMessageBox.confirm(`确定解除「${row.name}」(${row.bucket}) 绑定？`, "解除绑定", { type: "warning" })
    await deleteMerchantOssConfig(row.id)
    ElMessage.success("已解除绑定")
    loadOssConfigs()
  } catch {}
}

async function openAddOssDialog() {
  if (awsAccounts.value.length === 0) await loadAwsAccounts()
  addOssForm.value = { cloud_account_id: awsAccounts.value[0]?.id || 0, name: "", bucket: "", region: "ap-east-1", is_default: ossConfigs.value.length === 0 ? 1 : 0 }
  addOssDialogVisible.value = true
}

async function handleAddOss() {
  if (!selectedMerchantId.value) return
  const f = addOssForm.value
  if (!f.cloud_account_id || !f.name || !f.bucket) {
    ElMessage.warning("请填写云账号、名称和Bucket")
    return
  }
  addOssLoading.value = true
  try {
    await createMerchantOssConfig(selectedMerchantId.value, {
      cloud_account_id: f.cloud_account_id, name: f.name, bucket: f.bucket,
      region: f.region, is_default: f.is_default
    })
    ElMessage.success("绑定成功")
    addOssDialogVisible.value = false
    loadOssConfigs()
  } catch (e: any) { ElMessage.error(e.message || "绑定失败") }
  finally { addOssLoading.value = false }
}

// ========== 部署新节点弹窗 ==========
const deployDialogVisible = ref(false)
const deploying = ref(false)
const deployMode = ref<"existing" | "ec2">("existing")
const deployResult = ref<DeployTSDDResp | null>(null)
const deployResultDialogVisible = ref(false)

const awsAccounts = ref<Array<{ id: number; name: string; access_key_id: string }>>([])
const awsRegions = getAwsRegions("cn")

async function loadAwsAccounts() {
  try {
    const { data } = await getCloudAccountList({ page: 1, size: 100, cloud_type: "aws", status: 1 } as any)
    awsAccounts.value = (data.list || []).map((a: any) => ({ id: a.id, name: a.name, access_key_id: a.access_key_id }))
  } catch {}
}

const deployForm = ref<DeployNodeReq & { ami_id?: string; instance_type?: string; volume_size_gib?: number; cloud_account_id?: number; region_id?: string; key_name?: string; subnet_id?: string }>({
  server_id: 0,
  merchant_id: 0,
  node_role: "db",
  force_reset: false,
  db_host: "",
  minio_host: "",
  wk_node_id: 1001,
  wk_seed_node: ""
})

// 智能默认
const suggestedRole = computed<DeployNodeReq["node_role"]>(() => {
  if (!hasDB.value) return "db"
  if (!hasMinio.value) return "minio" as any
  return "app"
})

const autoDbHost = computed(() => {
  const n = dbNodes.value[0] || allinoneNodes.value[0]
  return n?.private_ip || ""
})

const autoMinioHost = computed(() => {
  const n = minioNodes.value[0]
  return n?.private_ip || autoDbHost.value // MinIO 默认与 DB 同机
})

const nextWkNodeId = computed(() => {
  const ids = (topology.value?.nodes || []).filter(n => n.wk_node_id > 0).map(n => n.wk_node_id)
  return ids.length > 0 ? Math.max(...ids) + 1 : 1001
})

const autoSeedNode = computed(() => {
  const first = appNodes.value.find(n => n.wk_node_id > 0) || allinoneNodes.value.find(n => n.wk_node_id > 0)
  return first ? `${first.wk_node_id}@${first.private_ip}:11110` : ""
})

const deployedServerIds = computed(() => new Set((topology.value?.nodes || []).map(n => n.server_id)))
const availableServers = computed(() => merchantServers.value.filter(s => !deployedServerIds.value.has(s.id)))

function openDeployDialog() {
  if (!selectedMerchantId.value) { ElMessage.warning("请先选择商户"); return }
  deployMode.value = "existing"
  const role = suggestedRole.value
  deployForm.value = {
    server_id: 0,
    merchant_id: selectedMerchantId.value!,
    node_role: role,
    force_reset: false,
    db_host: role === "app" ? autoDbHost.value : "",
    minio_host: role === "app" ? autoMinioHost.value : "",
    wk_node_id: (role === "app" || role === "allinone") ? nextWkNodeId.value : undefined as any,
    wk_seed_node: role === "app" ? autoSeedNode.value : "",
    cloud_account_id: awsAccounts.value[0]?.id,
    region_id: "ap-east-1",
    key_name: "",
    subnet_id: ""
  }
  deployDialogVisible.value = true
  loadAwsAccounts()
}

// 角色切换时更新智能默认
watch(() => deployForm.value.node_role, (role) => {
  if (role === "app") {
    deployForm.value.db_host = autoDbHost.value
    deployForm.value.minio_host = autoMinioHost.value
    deployForm.value.wk_node_id = nextWkNodeId.value
    deployForm.value.wk_seed_node = autoSeedNode.value
  } else if (role === "allinone") {
    deployForm.value.db_host = ""
    deployForm.value.minio_host = ""
    deployForm.value.wk_node_id = nextWkNodeId.value
    deployForm.value.wk_seed_node = ""
  } else {
    deployForm.value.db_host = ""
    deployForm.value.minio_host = ""
    deployForm.value.wk_node_id = undefined as any
    deployForm.value.wk_seed_node = ""
  }
})

async function handleDeploy() {
  const f = deployForm.value
  if (deployMode.value === "existing" && !f.server_id) { ElMessage.warning("请选择服务器"); return }
  if (deployMode.value === "ec2" && !f.ami_id) { ElMessage.warning("请填写 AMI ID"); return }
  if (deployMode.value === "ec2" && !f.cloud_account_id) { ElMessage.warning("请选择 AWS 账号"); return }
  if (f.node_role === "app" && !f.db_host) { ElMessage.warning("App 节点必须指定 DB 内网 IP"); return }

  deploying.value = true
  try {
    const payload: any = {
      merchant_id: f.merchant_id,
      node_role: f.node_role,
      force_reset: f.force_reset
    }
    if (deployMode.value === "ec2") {
      payload.server_id = 0
      payload.ami_id = f.ami_id
      payload.instance_type = f.instance_type
      payload.volume_size_gib = f.volume_size_gib
      payload.cloud_account_id = f.cloud_account_id
      payload.region_id = f.region_id
      payload.key_name = f.key_name
      payload.subnet_id = f.subnet_id
    } else {
      payload.server_id = f.server_id
    }
    if (f.db_host) payload.db_host = f.db_host
    if (f.minio_host) payload.minio_host = f.minio_host
    if (f.wk_node_id) payload.wk_node_id = f.wk_node_id
    if (f.wk_seed_node) payload.wk_seed_node = f.wk_seed_node

    const res = await deployNodeTracked(payload)
    deployResult.value = res.data
    deployDialogVisible.value = false
    deployResultDialogVisible.value = true
    res.data?.success ? ElMessage.success("部署成功") : ElMessage.error(res.data?.message || "部署失败")
    loadTopology()
    loadServers()
  } catch (e: any) {
    ElMessage.error(e?.message || "部署请求失败")
  } finally { deploying.value = false }
}

// ========== 重试部署 ==========
const retrying = ref(0)

async function handleRetry(node: ClusterNodeInfo) {
  try {
    await ElMessageBox.confirm(`确认重试 ${roleLabels[node.role] || node.role} 节点？`, "重试", { type: "warning" })
    retrying.value = node.node_id
    const res = await retryDeployApi({ node_id: node.node_id })
    res.data?.success ? ElMessage.success("部署成功") : ElMessage.warning(res.data?.message || "请检查详情")
    loadTopology()
  } catch {}
  finally { retrying.value = 0 }
}

// ========== 标签 ==========
const roleLabels: Record<string, string> = {
  all: "All-in-One", db: "DB（MySQL+Redis）", minio: "MinIO（对象存储）",
  app: "App（tsdd+WuKongIM+Web）", api: "API", im: "IM", web: "Web"
}
const roleColors: Record<string, string> = { all: "primary", db: "info", minio: "warning", app: "success", api: "warning" }

function deployStatusType(s: string) {
  return ({ success: "success", failed: "danger", deploying: "primary", pending: "info" } as any)[s] || "info"
}
function deployStatusLabel(s: string) {
  return ({ success: "已部署", failed: "失败", deploying: "部署中...", pending: "待部署" } as any)[s] || "未部署"
}
function statusColor(s: string) {
  return ({ success: "#67c23a", failed: "#f56c6c", running: "#409eff", skipped: "#e6a23c" } as any)[s] || "#dcdfe6"
}

// ========== 数据联动 ==========
watch(selectedMerchantId, () => {
  if (selectedMerchantId.value) {
    loadTopology()
    loadGostServers()
    loadOssConfigs()
    healthMap.value = {}
  } else {
    topology.value = null
    gostServers.value = []
    ossConfigs.value = []
  }
})

function refreshAll() {
  loadTopology()
  loadGostServers()
  loadOssConfigs()
}

onMounted(() => {
  loadMerchants()
  loadServers()
})
</script>

<template>
  <div class="app-container">
    <!-- 区块 1: 顶部操作栏 -->
    <el-card shadow="never" class="mb-4">
      <div style="display: flex; justify-content: space-between; align-items: center">
        <div style="display: flex; align-items: center; gap: 16px">
          <span class="font-bold">商户:</span>
          <el-select v-model="selectedMerchantId" placeholder="选择商户" style="width: 350px" filterable>
            <el-option v-for="m in merchantList" :key="m.id" :label="`${m.name} (${m.no})`" :value="m.id" />
          </el-select>
          <template v-if="topology">
            <el-tag :type="topology.deploy_mode === 'cluster' ? 'warning' : undefined" size="large">
              {{ topology.deploy_mode === "cluster" ? "多机" : "单机" }}
            </el-tag>
            <span class="text-gray-500 text-sm">{{ topology.nodes.length }} 节点</span>
          </template>
        </div>
        <div style="display: flex; gap: 8px">
          <el-button :loading="healthLoading" @click="handleHealthCheck" :disabled="!selectedMerchantId">健康检查</el-button>
          <el-button :loading="syncLoading" @click="handleGostSync" :disabled="!selectedMerchantId">GOST 同步</el-button>
          <el-button :loading="restartLoading" @click="handleGostRestart" :disabled="!selectedMerchantId">GOST 重启</el-button>
          <el-button type="primary" @click="openDeployDialog" :disabled="!selectedMerchantId">部署新节点</el-button>
          <el-button @click="refreshAll" :loading="loading" :disabled="!selectedMerchantId">刷新</el-button>
        </div>
      </div>
    </el-card>

    <template v-if="topology">
      <!-- 区块 2: 部署步骤引导 -->
      <el-card shadow="never" class="mb-4">
        <el-steps :active="currentStep" simple style="margin-bottom: 12px">
          <el-step title="DB 节点" description="MySQL + Redis" />
          <el-step title="MinIO 节点" description="对象存储" />
          <el-step title="App 节点" description="tsdd + WuKongIM + Web" />
        </el-steps>
        <div style="text-align: center; color: #606266; font-weight: 600">{{ stepMessage }}</div>
      </el-card>

      <!-- 区块 3: 节点表格 -->
      <el-card v-if="allinoneNodes.length" shadow="never" class="mb-4" v-loading="loading">
        <template #header><div class="section-header"><span>All-in-One 节点</span></div></template>
        <el-table :data="allinoneNodes" border size="small">
          <el-table-column prop="server_name" label="服务器" width="140" />
          <el-table-column prop="host" label="公网 IP" width="140" />
          <el-table-column prop="private_ip" label="内网 IP" width="140" />
          <el-table-column label="部署状态" width="100"><template #default="{ row }"><el-tag :type="deployStatusType(row.deploy_status)" size="small">{{ deployStatusLabel(row.deploy_status) }}</el-tag></template></el-table-column>
          <el-table-column label="健康" width="80"><template #default="{ row }"><el-tag :type="healthTag(row.server_id).type" size="small">{{ healthTag(row.server_id).text }}</el-tag></template></el-table-column>
          <el-table-column prop="last_deploy_at" label="部署时间" width="160" />
          <el-table-column label="操作" width="80"><template #default="{ row }"><el-button v-if="row.deploy_status === 'failed' || !row.deploy_status" type="warning" link size="small" :loading="retrying === row.node_id" @click="handleRetry(row)">{{ row.deploy_status === "failed" ? "重试" : "部署" }}</el-button></template></el-table-column>
        </el-table>
      </el-card>

      <el-card v-if="dbNodes.length" shadow="never" class="mb-4" v-loading="loading">
        <template #header><div class="section-header"><span>DB 节点</span><el-tag type="info" size="small">MySQL + Redis</el-tag></div></template>
        <el-table :data="dbNodes" border size="small">
          <el-table-column prop="server_name" label="服务器" width="140" />
          <el-table-column prop="host" label="公网 IP" width="140" />
          <el-table-column prop="private_ip" label="内网 IP" width="140" />
          <el-table-column label="部署状态" width="100"><template #default="{ row }"><el-tag :type="deployStatusType(row.deploy_status)" size="small">{{ deployStatusLabel(row.deploy_status) }}</el-tag></template></el-table-column>
          <el-table-column label="健康" width="80"><template #default="{ row }"><el-tag :type="healthTag(row.server_id).type" size="small">{{ healthTag(row.server_id).text }}</el-tag></template></el-table-column>
          <el-table-column prop="last_deploy_at" label="部署时间" width="160" />
          <el-table-column label="操作" width="80"><template #default="{ row }"><el-button v-if="row.deploy_status === 'failed' || !row.deploy_status" type="warning" link size="small" :loading="retrying === row.node_id" @click="handleRetry(row)">{{ row.deploy_status === "failed" ? "重试" : "部署" }}</el-button></template></el-table-column>
        </el-table>
      </el-card>

      <el-card v-if="minioNodes.length" shadow="never" class="mb-4" v-loading="loading">
        <template #header><div class="section-header"><span>MinIO 节点</span><el-tag type="warning" size="small">对象存储</el-tag></div></template>
        <el-table :data="minioNodes" border size="small">
          <el-table-column prop="server_name" label="服务器" width="140" />
          <el-table-column prop="host" label="公网 IP" width="140" />
          <el-table-column prop="private_ip" label="内网 IP" width="140" />
          <el-table-column label="部署状态" width="100"><template #default="{ row }"><el-tag :type="deployStatusType(row.deploy_status)" size="small">{{ deployStatusLabel(row.deploy_status) }}</el-tag></template></el-table-column>
          <el-table-column label="健康" width="80"><template #default="{ row }"><el-tag :type="healthTag(row.server_id).type" size="small">{{ healthTag(row.server_id).text }}</el-tag></template></el-table-column>
          <el-table-column prop="last_deploy_at" label="部署时间" width="160" />
          <el-table-column label="操作" width="80"><template #default="{ row }"><el-button v-if="row.deploy_status === 'failed' || !row.deploy_status" type="warning" link size="small" :loading="retrying === row.node_id" @click="handleRetry(row)">{{ row.deploy_status === "failed" ? "重试" : "部署" }}</el-button></template></el-table-column>
        </el-table>
      </el-card>

      <el-card v-if="appNodes.length" shadow="never" class="mb-4" v-loading="loading">
        <template #header><div class="section-header"><span>App 节点</span><el-tag type="success" size="small">tsdd + WuKongIM + Web</el-tag></div></template>
        <el-table :data="appNodes" border size="small">
          <el-table-column prop="server_name" label="服务器" width="130" />
          <el-table-column prop="host" label="公网 IP" width="130" />
          <el-table-column prop="private_ip" label="内网 IP" width="130" />
          <el-table-column prop="wk_node_id" label="WK节点ID" width="90" />
          <el-table-column prop="db_host" label="DB 地址" width="130" />
          <el-table-column prop="minio_host" label="MinIO 地址" width="130" />
          <el-table-column label="状态" width="80"><template #default="{ row }"><el-tag :type="deployStatusType(row.deploy_status)" size="small">{{ deployStatusLabel(row.deploy_status) }}</el-tag></template></el-table-column>
          <el-table-column label="健康" width="80"><template #default="{ row }"><el-tag :type="healthTag(row.server_id).type" size="small">{{ healthTag(row.server_id).text }}</el-tag></template></el-table-column>
          <el-table-column prop="last_deploy_at" label="部署时间" width="160" />
          <el-table-column label="操作" width="80" fixed="right"><template #default="{ row }"><el-button v-if="row.deploy_status === 'failed' || !row.deploy_status" type="warning" link size="small" :loading="retrying === row.node_id" @click="handleRetry(row)">{{ row.deploy_status === "failed" ? "重试" : "部署" }}</el-button></template></el-table-column>
        </el-table>
      </el-card>

      <el-empty v-if="topology.nodes.length === 0" description="暂无节点，请点击「部署新节点」" />

      <!-- 区块 4: GOST 服务器管理 -->
      <el-card shadow="never" class="mb-4" v-loading="gostLoading">
        <template #header>
          <div class="section-header">
            <div style="display: flex; align-items: center; gap: 8px">
              <span class="font-bold">GOST 转发服务器</span>
              <el-tag size="small">{{ gostServers.length }} 台</el-tag>
            </div>
            <el-button type="primary" size="small" @click="openAddGost">关联 GOST 服务器</el-button>
          </div>
        </template>
        <el-table v-if="gostServers.length" :data="gostServers" border size="small">
          <el-table-column prop="server_name" label="服务器" width="160" />
          <el-table-column prop="server_host" label="IP" width="140" />
          <el-table-column label="转发模式" width="120">
            <template #default="{ row }">
              <el-tag v-if="row.forward_type === 1 || row.tls_enabled === 1" type="success" size="small">TLS 加密</el-tag>
              <el-tag v-else type="info" size="small">TCP 直连</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="主服务器" width="90"><template #default="{ row }"><el-tag v-if="row.is_primary" type="danger" size="small">主</el-tag></template></el-table-column>
          <el-table-column label="健康" width="80"><template #default="{ row }"><el-tag :type="healthTag(row.server_id).type" size="small">{{ healthTag(row.server_id).text }}</el-tag></template></el-table-column>
          <el-table-column label="操作" width="80"><template #default="{ row }"><el-button type="danger" link size="small" @click="handleRemoveGost(row)">解除</el-button></template></el-table-column>
        </el-table>
        <el-empty v-else description="暂无关联的 GOST 服务器" :image-size="60" />
      </el-card>

      <!-- 区块 5: OSS 配置管理 -->
      <el-card shadow="never" class="mb-4" v-loading="ossLoading">
        <template #header>
          <div class="section-header">
            <div style="display: flex; align-items: center; gap: 8px">
              <span class="font-bold">OSS 上传配置</span>
              <el-tag size="small">{{ ossConfigs.length }} 个</el-tag>
            </div>
            <el-button type="primary" size="small" @click="openAddOssDialog">绑定 Bucket</el-button>
          </div>
        </template>
        <el-table v-if="ossConfigs.length" :data="ossConfigs" border size="small">
          <el-table-column prop="name" label="名称" width="140" />
          <el-table-column prop="bucket" label="Bucket" width="160" />
          <el-table-column prop="region" label="区域" width="130" />
          <el-table-column label="默认" width="70"><template #default="{ row }"><el-tag v-if="row.is_default" type="success" size="small">是</el-tag></template></el-table-column>
          <el-table-column label="操作" width="80"><template #default="{ row }"><el-button type="danger" link size="small" @click="handleRemoveOss(row)">解除</el-button></template></el-table-column>
        </el-table>
        <el-empty v-else description="暂无 OSS 配置" :image-size="60" />
      </el-card>
    </template>

    <el-empty v-else-if="!selectedMerchantId" description="请选择商户查看集群拓扑" />

    <!-- 区块 6: 部署新节点弹窗 -->
    <el-dialog v-model="deployDialogVisible" title="部署新节点" width="600px" :close-on-click-modal="false">
      <el-form :model="deployForm" label-width="130px">
        <el-form-item label="部署方式">
          <el-radio-group v-model="deployMode">
            <el-radio value="existing">选择已有服务器</el-radio>
            <el-radio value="ec2">创建新 EC2</el-radio>
          </el-radio-group>
        </el-form-item>

        <!-- 已有服务器模式 -->
        <el-form-item v-if="deployMode === 'existing'" label="目标服务器" required>
          <el-select v-model="deployForm.server_id" placeholder="选择服务器" filterable style="width: 100%">
            <el-option v-for="s in availableServers" :key="s.id" :label="`${s.name} (${s.host})`" :value="s.id" />
          </el-select>
          <div v-if="availableServers.length === 0" class="text-xs text-gray-400 mt-1">无可用服务器，请先在服务器管理添加</div>
        </el-form-item>

        <!-- EC2 模式 -->
        <template v-if="deployMode === 'ec2'">
          <el-form-item label="AMI ID" required>
            <el-input v-model="deployForm.ami_id" placeholder="ami-xxxxxxxxx" />
          </el-form-item>
          <el-row :gutter="16">
            <el-col :span="12"><el-form-item label="实例类型"><el-input v-model="deployForm.instance_type" :placeholder="deployForm.node_role === 'db' ? 'r5.large' : deployForm.node_role === 'minio' ? 't3.medium' : 'c6i.4xlarge'" /></el-form-item></el-col>
            <el-col :span="12"><el-form-item label="磁盘(GB)"><el-input-number v-model="deployForm.volume_size_gib" :min="20" :max="4000" style="width: 100%" /></el-form-item></el-col>
          </el-row>
          <el-row :gutter="16">
            <el-col :span="12"><el-form-item label="AWS 账号" required><el-select v-model="deployForm.cloud_account_id" placeholder="选择" filterable style="width: 100%"><el-option v-for="a in awsAccounts" :key="a.id" :label="`${a.name}`" :value="a.id" /></el-select></el-form-item></el-col>
            <el-col :span="12"><el-form-item label="区域"><el-select v-model="deployForm.region_id" filterable style="width: 100%"><el-option v-for="r in awsRegions" :key="r.id" :label="`${r.name} (${r.id})`" :value="r.id" /></el-select></el-form-item></el-col>
          </el-row>
          <el-row :gutter="16">
            <el-col :span="12"><el-form-item label="Key Name"><el-input v-model="deployForm.key_name" placeholder="tsdd-deploy-key" /></el-form-item></el-col>
            <el-col :span="12"><el-form-item label="Subnet ID"><el-input v-model="deployForm.subnet_id" placeholder="留空默认" /></el-form-item></el-col>
          </el-row>
        </template>

        <el-divider />

        <el-form-item label="节点角色" required>
          <el-radio-group v-model="deployForm.node_role">
            <el-radio value="db">DB</el-radio>
            <el-radio value="minio">MinIO</el-radio>
            <el-radio value="app">App</el-radio>
            <el-radio value="allinone">All-in-One</el-radio>
          </el-radio-group>
        </el-form-item>

        <template v-if="deployForm.node_role === 'app'">
          <el-form-item label="DB 内网 IP" required>
            <el-input v-model="deployForm.db_host" placeholder="DB 节点的内网 IP" />
          </el-form-item>
          <el-form-item label="MinIO 内网 IP">
            <el-input v-model="deployForm.minio_host" placeholder="MinIO 节点内网 IP（留空同 DB）" />
          </el-form-item>
        </template>

        <template v-if="deployForm.node_role === 'app' || deployForm.node_role === 'allinone'">
          <el-row :gutter="16">
            <el-col :span="12"><el-form-item label="WK 节点 ID"><el-input-number v-model="deployForm.wk_node_id" :min="1001" :max="9999" style="width: 100%" /></el-form-item></el-col>
            <el-col :span="12"><el-form-item label="种子节点"><el-input v-model="deployForm.wk_seed_node" placeholder="首个节点留空" /></el-form-item></el-col>
          </el-row>
        </template>

        <el-form-item label="强制重置">
          <el-switch v-model="deployForm.force_reset" />
          <span class="ml-2 text-xs text-gray-400">删除现有容器和数据重新部署</span>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="deployDialogVisible = false" :disabled="deploying">取消</el-button>
        <el-button type="primary" @click="handleDeploy" :loading="deploying">{{ deploying ? "部署中..." : "开始部署" }}</el-button>
      </template>
    </el-dialog>

    <!-- 区块 7: 部署结果弹窗 -->
    <el-dialog v-model="deployResultDialogVisible" title="部署结果" width="600px">
      <template v-if="deployResult">
        <div v-for="(step, idx) in deployResult.steps" :key="idx" style="display: flex; align-items: flex-start; gap: 10px; padding: 8px 0; border-bottom: 1px solid #f0f0f0">
          <div style="width: 12px; height: 12px; border-radius: 50%; margin-top: 4px; flex-shrink: 0" :style="{ background: statusColor(step.status) }" />
          <div style="flex: 1">
            <div style="font-weight: 500">{{ step.name }} <el-tag v-if="step.status === 'success'" type="success" size="small">完成</el-tag><el-tag v-else-if="step.status === 'failed'" type="danger" size="small">失败</el-tag></div>
            <div v-if="step.message" style="font-size: 12px; color: #909399; margin-top: 2px">{{ step.message }}</div>
          </div>
        </div>
        <div v-if="deployResult.success" style="margin-top: 16px; padding: 12px; background: #f0f9eb; border-radius: 4px">
          <div v-if="deployResult.api_url">API: <span style="font-family: monospace; font-weight: 600">{{ deployResult.api_url }}</span></div>
          <div v-if="deployResult.web_url">Web: <span style="font-family: monospace; font-weight: 600">{{ deployResult.web_url }}</span></div>
          <div v-if="deployResult.admin_url">Admin: <span style="font-family: monospace; font-weight: 600">{{ deployResult.admin_url }}</span></div>
        </div>
      </template>
      <template #footer><el-button @click="deployResultDialogVisible = false">关闭</el-button></template>
    </el-dialog>

    <!-- 区块 8: GOST 同步结果弹窗 -->
    <el-dialog v-model="syncDialogVisible" title="GOST 同步结果" width="600px">
      <el-table :data="syncResults" border size="small">
        <el-table-column label="状态" width="80"><template #default="{ row }"><el-tag :type="row.success ? 'success' : 'danger'" size="small">{{ row.success ? "成功" : "失败" }}</el-tag></template></el-table-column>
        <el-table-column prop="server_name" label="服务器" width="140" />
        <el-table-column prop="server_host" label="IP" width="140" />
        <el-table-column label="详情"><template #default="{ row }"><span v-if="row.error" style="color: #f56c6c">{{ row.error }}</span><span v-else class="text-gray-400">-</span></template></el-table-column>
      </el-table>
      <template #footer><el-button @click="syncDialogVisible = false">关闭</el-button></template>
    </el-dialog>

    <!-- 关联 GOST 服务器弹窗 -->
    <el-dialog v-model="addGostDialogVisible" title="关联 GOST 服务器" width="450px">
      <el-form :model="addGostForm" label-width="100px">
        <el-form-item label="GOST 服务器">
          <el-select v-model="addGostForm.server_id" placeholder="选择" filterable style="width: 100%">
            <el-option v-for="s in availableGostServers" :key="s.id" :label="`${s.name} (${s.host})`" :value="s.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="主服务器"><el-switch v-model="addGostForm.is_primary" :active-value="1" :inactive-value="0" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="addGostDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleAddGost" :loading="addGostLoading">确认</el-button>
      </template>
    </el-dialog>

    <!-- 绑定 OSS Bucket 弹窗 -->
    <el-dialog v-model="addOssDialogVisible" title="绑定 OSS Bucket" width="500px">
      <el-form :model="addOssForm" label-width="100px">
        <el-form-item label="云账号" required>
          <el-select v-model="addOssForm.cloud_account_id" placeholder="选择云账号" filterable style="width: 100%">
            <el-option v-for="a in awsAccounts" :key="a.id" :label="a.name" :value="a.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="配置名称" required>
          <el-input v-model="addOssForm.name" placeholder="如: 主OSS、备用OSS" />
        </el-form-item>
        <el-form-item label="Bucket" required>
          <el-input v-model="addOssForm.bucket" placeholder="S3 Bucket 名称" />
        </el-form-item>
        <el-form-item label="区域">
          <el-select v-model="addOssForm.region" filterable style="width: 100%">
            <el-option v-for="r in awsRegions" :key="r.id" :label="`${r.name} (${r.id})`" :value="r.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="设为默认">
          <el-switch v-model="addOssForm.is_default" :active-value="1" :inactive-value="0" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="addOssDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleAddOss" :loading="addOssLoading">确认绑定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style lang="scss" scoped>
.app-container { padding: 20px; }
.mb-4 { margin-bottom: 16px; }
.section-header { display: flex; justify-content: space-between; align-items: center; }
</style>
