<script lang="ts" setup>
import type { ClusterNodeInfo, DeployStep, DeployTSDDResp, GostSyncResult, NodeRole, ServerResp } from "@@/apis/deploy/type"
import type { MerchantGostServerResp, MerchantOssConfigResp } from "@@/apis/merchant/type"
import type { TargetItem } from "@@/apis/ip_embed/type"
import { deployNode, getServerList, batchHealthCheck, getClusterNodes, syncClusterGost, batchServiceAction } from "@@/apis/deploy"
import { getMerchantList, listMerchantGostServers, createMerchantGostServer, deleteMerchantGostServer, listMerchantOssConfigs, createMerchantOssConfig, deleteMerchantOssConfig } from "@@/apis/merchant"
import { getTargets } from "@@/apis/ip_embed"
import { getCloudAccountList } from "@@/apis/cloud_account"
import { getAwsRegions } from "@@/constants/aws-regions"

defineOptions({
  name: "DeployCluster"
})

// ========== 商户列表 ==========
const merchantList = ref<{ id: number; name: string }[]>([])
const selectedMerchantId = ref<number | undefined>(undefined)

async function fetchMerchants() {
  const res = await getMerchantList({ page: 1, size: 1000 })
  merchantList.value = (res.data.list || []).map((m: any) => ({ id: m.id, name: m.name }))
  // 自动选择第一个商户
  if (merchantList.value.length > 0 && !selectedMerchantId.value) {
    selectedMerchantId.value = merchantList.value[0].id
  }
}

// ========== 服务器列表 ==========
const serverList = ref<ServerResp[]>([])
const allServerList = ref<ServerResp[]>([])
const serverLoading = ref(false)

async function fetchServers() {
  serverLoading.value = true
  try {
    const params: any = { page: 1, size: 200 }
    if (selectedMerchantId.value) {
      params.merchant_id = selectedMerchantId.value
    }
    const res = await getServerList(params)
    serverList.value = res.data.list || []
  } finally {
    serverLoading.value = false
  }
}

async function fetchAllServers() {
  try {
    const res = await getServerList({ page: 1, size: 5000 })
    allServerList.value = res.data?.list ?? []
  } catch {
    allServerList.value = []
  }
}

// ========== 集群拓扑 ==========
const clusterNodes = ref<ClusterNodeInfo[]>([])
const topologyLoading = ref(false)

// 按角色分类
const dbNodes = computed(() => clusterNodes.value.filter(n => n.node_role === "db"))
const minioNodes = computed(() => clusterNodes.value.filter(n => n.node_role === "minio"))
const appNodes = computed(() => clusterNodes.value.filter(n => n.node_role === "app"))
const allinoneNodes = computed(() => clusterNodes.value.filter(n => n.node_role === "allinone"))

// 部署进度
const deployProgress = computed(() => {
  const hasDb = dbNodes.value.length > 0 || allinoneNodes.value.length > 0
  const hasMinio = minioNodes.value.length > 0 || allinoneNodes.value.length > 0
  const hasApp = appNodes.value.length > 0 || allinoneNodes.value.length > 0
  return { hasDb, hasMinio, hasApp }
})

// 推荐下一步
const nextStep = computed(() => {
  if (allinoneNodes.value.length > 0) return "已部署全量节点，可部署更多 App 节点扩容"
  if (!deployProgress.value.hasDb) return "第 1 步：部署 DB 节点（MySQL + Redis）"
  if (!deployProgress.value.hasMinio) return "第 2 步：部署 MinIO 节点（对象存储）"
  if (!deployProgress.value.hasApp) return "第 3 步：部署首个 App 节点"
  return `已部署 ${appNodes.value.length} 个 App 节点，可继续扩容`
})

async function fetchTopology() {
  if (!selectedMerchantId.value) return
  topologyLoading.value = true
  try {
    const res = await getClusterNodes(selectedMerchantId.value)
    clusterNodes.value = res.data || []
  } catch (e: any) {
    clusterNodes.value = []
  } finally {
    topologyLoading.value = false
  }
}

// 切换商户时自动刷新拓扑 + 服务器列表 + GOST 关联 + OSS 配置
watch(selectedMerchantId, () => {
  fetchTopology()
  fetchServers()
  fetchGostRelations()
  fetchOssConfigs()
})

// ========== 健康检查 ==========
const healthLoading = ref(false)
const healthResults = ref<Record<number, string>>({}) // server_id -> status

async function checkAllHealth() {
  const nodeServerIds = clusterNodes.value.map(n => n.server_id)
  if (nodeServerIds.length === 0) {
    ElMessage.warning("当前商户无集群节点")
    return
  }
  healthLoading.value = true
  try {
    const res = await batchHealthCheck({ server_ids: nodeServerIds })
    const results: Record<number, string> = {}
    for (const r of res.data.results || []) {
      results[r.server_id] = r.status
    }
    healthResults.value = results
    ElMessage.success("健康检查完成")
  } catch (e: any) {
    ElMessage.error(e.message || "健康检查失败")
  } finally {
    healthLoading.value = false
  }
}

function healthTag(serverId: number) {
  const s = healthResults.value[serverId]
  if (!s) return { type: "info" as const, text: "未检查" }
  if (s === "healthy") return { type: "success" as const, text: "健康" }
  if (s === "partial") return { type: "warning" as const, text: "部分异常" }
  return { type: "danger" as const, text: "异常" }
}

// ========== GOST 服务器关联 ==========
const gostRelations = ref<MerchantGostServerResp[]>([])
const gostRelationLoading = ref(false)
const addGostDialogVisible = ref(false)
const addGostForm = ref({ server_id: undefined as number | undefined, is_primary: 0 })
const addGostLoading = ref(false)

// 系统服务器列表（server_type=2，用于 GOST 关联）
const systemServers = computed(() =>
  allServerList.value.filter(s => s.server_type === 2)
)

// 已关联的 server_id 集合
const relatedServerIds = computed(() => new Set(gostRelations.value.map(r => r.server_id)))

// 可选的系统服务器（排除已关联的）
const availableGostServers = computed(() =>
  systemServers.value.filter(s => !relatedServerIds.value.has(s.id))
)

async function fetchGostRelations() {
  if (!selectedMerchantId.value) return
  gostRelationLoading.value = true
  try {
    const res = await listMerchantGostServers(selectedMerchantId.value)
    gostRelations.value = Array.isArray(res.data) ? res.data : []
  } catch {
    gostRelations.value = []
  } finally {
    gostRelationLoading.value = false
  }
}

function openAddGostDialog() {
  addGostForm.value = { server_id: undefined, is_primary: gostRelations.value.length === 0 ? 1 : 0 }
  addGostDialogVisible.value = true
}

async function submitAddGost() {
  if (!selectedMerchantId.value || !addGostForm.value.server_id) {
    ElMessage.warning("请选择 GOST 服务器")
    return
  }
  addGostLoading.value = true
  try {
    await createMerchantGostServer(selectedMerchantId.value, {
      server_id: addGostForm.value.server_id,
      is_primary: addGostForm.value.is_primary,
      status: 1
    })
    ElMessage.success("关联成功")
    addGostDialogVisible.value = false
    fetchGostRelations()
  } catch (e: any) {
    ElMessage.error(e.message || "关联失败")
  } finally {
    addGostLoading.value = false
  }
}

async function removeGostRelation(relation: MerchantGostServerResp) {
  try {
    await ElMessageBox.confirm(
      `确定解除与 ${relation.server_name} (${relation.server_host}) 的关联吗？`,
      "解除关联",
      { confirmButtonText: "确定", cancelButtonText: "取消", type: "warning" }
    )
    await deleteMerchantGostServer(relation.id)
    ElMessage.success("已解除关联")
    fetchGostRelations()
  } catch {
    // 用户取消
  }
}

// ========== GOST 同步 ==========
const gostSyncing = ref(false)
const gostResults = ref<GostSyncResult[]>([])
const gostResultDialogVisible = ref(false)

async function doSyncGost() {
  if (!selectedMerchantId.value) {
    ElMessage.warning("请先选择商户")
    return
  }
  if (appNodes.value.length === 0) {
    ElMessage.warning("当前商户无 App 节点，无需同步 GOST")
    return
  }
  gostSyncing.value = true
  try {
    const res = await syncClusterGost(selectedMerchantId.value)
    gostResults.value = res.data || []
    gostResultDialogVisible.value = true
    const ok = gostResults.value.filter(r => r.success).length
    const fail = gostResults.value.filter(r => !r.success).length
    if (fail === 0) {
      ElMessage.success(`GOST 同步完成：${ok} 台成功，配置已自动持久化`)
    } else {
      ElMessage.warning(`GOST 同步完成：${ok} 成功 / ${fail} 失败`)
    }
  } catch (e: any) {
    ElMessage.error(e.message || "GOST 同步失败")
  } finally {
    gostSyncing.value = false
  }
}

// ========== 重启 GOST ==========
const gostRestarting = ref(false)

async function doRestartGost() {
  if (gostRelations.value.length === 0) {
    ElMessage.warning("当前商户无关联的 GOST 服务器")
    return
  }
  try {
    await ElMessageBox.confirm("确认重启所有关联的 GOST 服务器？", "重启 GOST", { type: "warning" })
  } catch {
    return
  }
  gostRestarting.value = true
  try {
    const serverIds = gostRelations.value.map(r => r.server_id)
    const res = await batchServiceAction({ server_ids: serverIds, service_name: "gost", action: "restart" })
    const data = res.data
    if (data.fail_count === 0) {
      ElMessage.success(`GOST 重启完成：${data.success_count} 台成功`)
    } else {
      ElMessage.warning(`GOST 重启完成：${data.success_count} 成功 / ${data.fail_count} 失败`)
    }
  } catch (e: any) {
    ElMessage.error(e.message || "GOST 重启失败")
  } finally {
    gostRestarting.value = false
  }
}

// ========== OSS 配置管理 ==========
const ossConfigs = ref<MerchantOssConfigResp[]>([])
const ossConfigLoading = ref(false)
const addOssDialogVisible = ref(false)
const addOssLoading = ref(false)

// 已有的上传目标列表（ip_embed_targets）
const allTargets = ref<TargetItem[]>([])
const selectedTargetId = ref<number | undefined>(undefined)
const addOssIsDefault = ref(0)

// 已绑定的 cloud_account_id+bucket 组合，排除已绑定的目标
const boundKeys = computed(() => new Set(
  ossConfigs.value.map(c => `${c.cloud_account_id}:${c.bucket}`)
))
const availableTargets = computed(() =>
  allTargets.value.filter(t => !boundKeys.value.has(`${t.cloud_account_id}:${t.bucket}`))
)
const selectedTarget = computed(() =>
  allTargets.value.find(t => t.id === selectedTargetId.value)
)

async function fetchOssConfigs() {
  if (!selectedMerchantId.value) return
  ossConfigLoading.value = true
  try {
    const res = await listMerchantOssConfigs(selectedMerchantId.value)
    ossConfigs.value = Array.isArray(res.data) ? res.data : []
  } catch {
    ossConfigs.value = []
  } finally {
    ossConfigLoading.value = false
  }
}

async function loadAllTargets() {
  try {
    const res = await getTargets()
    allTargets.value = res.data?.targets || []
  } catch {
    allTargets.value = []
  }
}

function openAddOssDialog() {
  selectedTargetId.value = undefined
  addOssIsDefault.value = ossConfigs.value.length === 0 ? 1 : 0
  addOssDialogVisible.value = true
  loadAllTargets()
}

async function submitAddOss() {
  const target = selectedTarget.value
  if (!selectedMerchantId.value || !target) {
    ElMessage.warning("请选择上传目标")
    return
  }
  addOssLoading.value = true
  try {
    await createMerchantOssConfig(selectedMerchantId.value, {
      cloud_account_id: target.cloud_account_id,
      name: target.name,
      bucket: target.bucket,
      region: target.region_id,
      is_default: addOssIsDefault.value
    })
    ElMessage.success("绑定成功")
    addOssDialogVisible.value = false
    fetchOssConfigs()
  } catch (e: any) {
    ElMessage.error(e.message || "绑定失败")
  } finally {
    addOssLoading.value = false
  }
}

async function removeOssConfig(config: MerchantOssConfigResp) {
  try {
    await ElMessageBox.confirm(
      `确定解除「${config.name}」(${config.bucket}) 与商户的绑定吗？上传目标不受影响。`,
      "解除绑定",
      { confirmButtonText: "确定", cancelButtonText: "取消", type: "warning" }
    )
    await deleteMerchantOssConfig(config.id)
    ElMessage.success("已解除绑定")
    fetchOssConfigs()
  } catch {
    // 用户取消
  }
}

// ========== AWS 账号（创建 EC2 用） ==========
const merchantAwsAccounts = ref<Array<{ id: number; name: string; access_key_id: string }>>([])
const systemAwsAccounts = ref<Array<{ id: number; name: string; access_key_id: string }>>([])
const awsRegions = getAwsRegions("cn")

async function loadAwsAccounts() {
  const mapAccount = (a: any) => ({ id: a.id, name: a.name, access_key_id: a.access_key_id })
  try {
    // 加载商户 AWS 账号
    if (selectedMerchantId.value) {
      const { data } = await getCloudAccountList({
        page: 1, size: 100, cloud_type: "aws", status: 1, merchant_id: selectedMerchantId.value
      })
      merchantAwsAccounts.value = (data.list || []).map(mapAccount)
    }
    // 加载系统 AWS 账号
    const { data: sysData } = await getCloudAccountList({
      page: 1, size: 100, cloud_type: "aws", status: 1, account_type: "system"
    })
    systemAwsAccounts.value = (sysData.list || []).map(mapAccount)
  } catch (e) {
    console.error("加载 AWS 账号失败", e)
  }
}

// ========== 部署对话框 ==========
const deployDialogVisible = ref(false)
const deploying = ref(false)
const createNewServer = ref(false) // 是否创建新 EC2

const deployForm = ref({
  server_id: undefined as number | undefined,
  merchant_id: undefined as number | undefined,
  node_role: "app" as NodeRole,
  force_reset: false,
  db_host: "",
  minio_host: "",
  wk_node_id: undefined as number | undefined,
  wk_seed_node: "",
  // EC2 创建字段
  ami_id: "",
  instance_type: "",
  volume_size_gib: undefined as number | undefined,
  cloud_account_id: undefined as number | undefined,
  region_id: "ap-east-1",
  key_name: "",
  subnet_id: ""
})

// 部署结果
const deployResult = ref<DeployTSDDResp | null>(null)
const resultDialogVisible = ref(false)

// 已分配的服务器ID集合（避免重复部署）
const deployedServerIds = computed(() => new Set(clusterNodes.value.map(n => n.server_id)))

// 可选服务器列表（排除已部署的）
const availableServers = computed(() =>
  serverList.value.filter(s => !deployedServerIds.value.has(s.id))
)

function openDeployDialog() {
  // 自动推断角色
  let defaultRole = "app" as NodeRole
  if (!deployProgress.value.hasDb) defaultRole = "db"
  else if (!deployProgress.value.hasMinio) defaultRole = "minio"
  else if (allinoneNodes.value.length > 0) defaultRole = "app" // 已有全量节点，默认扩容 App

  // 自动填充 DB/MinIO 内网 IP
  const dbNode = dbNodes.value[0]
  const minioNode = minioNodes.value[0]

  // 自动计算下一个 WK 节点 ID
  const existingIds = appNodes.value.map(n => n.wk_node_id).filter(id => id > 0)
  const nextWkId = existingIds.length > 0 ? Math.max(...existingIds) + 1 : 1001

  // 自动发现种子节点
  const firstApp = appNodes.value[0]
  const seedNode = firstApp ? `${firstApp.wk_node_id}@${firstApp.private_ip}:11110` : ""

  createNewServer.value = false
  deployForm.value = {
    server_id: undefined,
    merchant_id: selectedMerchantId.value,
    node_role: defaultRole,
    force_reset: false,
    db_host: dbNode?.private_ip || "",
    minio_host: minioNode?.private_ip || "",
    wk_node_id: defaultRole === "app" ? nextWkId : undefined,
    wk_seed_node: defaultRole === "app" && firstApp ? seedNode : "",
    ami_id: "",
    instance_type: "",
    volume_size_gib: undefined,
    cloud_account_id: undefined,
    region_id: "ap-east-1",
    key_name: "",
    subnet_id: ""
  }
  deployResult.value = null
  deployDialogVisible.value = true
  // 加载 AWS 账号
  loadAwsAccounts().then(() => {
    // 自动选择商户账号（优先）或系统账号
    if (merchantAwsAccounts.value.length > 0) {
      deployForm.value.cloud_account_id = merchantAwsAccounts.value[0].id
    } else if (systemAwsAccounts.value.length > 0) {
      deployForm.value.cloud_account_id = systemAwsAccounts.value[0].id
    }
  })
}

// 切换角色时自动更新表单
watch(() => deployForm.value.node_role, (role) => {
  const dbNode = dbNodes.value[0]
  const minioNode = minioNodes.value[0]
  if (role === "app") {
    deployForm.value.db_host = dbNode?.private_ip || ""
    deployForm.value.minio_host = minioNode?.private_ip || ""
    const existingIds = appNodes.value.map(n => n.wk_node_id).filter(id => id > 0)
    deployForm.value.wk_node_id = existingIds.length > 0 ? Math.max(...existingIds) + 1 : 1001
    const firstApp = appNodes.value[0]
    deployForm.value.wk_seed_node = firstApp ? `${firstApp.wk_node_id}@${firstApp.private_ip}:11110` : ""
  } else if (role === "allinone") {
    deployForm.value.db_host = ""
    deployForm.value.minio_host = ""
    const existingIds = [...appNodes.value, ...allinoneNodes.value].map(n => n.wk_node_id).filter(id => id > 0)
    deployForm.value.wk_node_id = existingIds.length > 0 ? Math.max(...existingIds) + 1 : 1001
    deployForm.value.wk_seed_node = ""
  } else {
    deployForm.value.db_host = ""
    deployForm.value.minio_host = ""
    deployForm.value.wk_node_id = undefined
    deployForm.value.wk_seed_node = ""
  }
})

async function submitDeploy() {
  const form = deployForm.value
  if (!createNewServer.value && !form.server_id) {
    ElMessage.warning("请选择目标服务器")
    return
  }
  if (createNewServer.value && !form.ami_id) {
    ElMessage.warning("请填写 AMI ID")
    return
  }
  if (createNewServer.value && !form.cloud_account_id) {
    ElMessage.warning("请选择 AWS 云账号")
    return
  }
  if (!form.merchant_id) {
    ElMessage.warning("请选择商户")
    return
  }
  if (form.node_role === "app" && !form.db_host) {
    ElMessage.warning("App 节点必须指定 DB 节点内网 IP")
    return
  }

  deploying.value = true
  try {
    const data: any = {
      merchant_id: form.merchant_id,
      node_role: form.node_role,
      force_reset: form.force_reset
    }

    if (createNewServer.value) {
      // 创建新 EC2 模式
      data.server_id = 0
      data.ami_id = form.ami_id
      data.instance_type = form.instance_type || undefined
      data.volume_size_gib = form.volume_size_gib || undefined
      data.cloud_account_id = form.cloud_account_id
      data.region_id = form.region_id
      data.key_name = form.key_name || undefined
      data.subnet_id = form.subnet_id || undefined
    } else {
      data.server_id = form.server_id
    }

    if (form.db_host) data.db_host = form.db_host
    if (form.minio_host) data.minio_host = form.minio_host
    if (form.wk_node_id) data.wk_node_id = form.wk_node_id
    if (form.wk_seed_node) data.wk_seed_node = form.wk_seed_node

    const res = await deployNode(data)
    deployResult.value = res.data
    deployDialogVisible.value = false
    resultDialogVisible.value = true

    if (res.data.success) {
      ElMessage.success("部署成功")
      fetchTopology() // 刷新拓扑
      fetchServers()
    } else {
      ElMessage.error(res.data.message || "部署失败")
    }
  } catch (e: any) {
    ElMessage.error(e.message || "部署请求失败")
  } finally {
    deploying.value = false
  }
}

// 步骤状态
function stepColor(status: string) {
  if (status === "success") return "#67c23a"
  if (status === "failed") return "#f56c6c"
  if (status === "warning") return "#e6a23c"
  if (status === "running") return "#409eff"
  return "#909399"
}

// 角色标签
function roleLabel(role: string) {
  if (role === "db") return "DB 节点"
  if (role === "minio") return "MinIO 节点"
  if (role === "app") return "App 节点"
  return "全量节点"
}

function roleTagType(role: string) {
  if (role === "db") return "warning"
  if (role === "minio") return "info"
  if (role === "app") return "success"
  return "primary"
}

function formatTime(t: string) {
  if (!t) return "-"
  return t.replace("T", " ").substring(0, 19)
}

// ========== 初始化 ==========
onMounted(() => {
  fetchServers()
  fetchAllServers()
  fetchMerchants()
})
</script>

<template>
  <div class="app-container">
    <!-- 顶部操作栏 -->
    <el-card shadow="never" class="mb-4">
      <div style="display: flex; justify-content: space-between; align-items: center;">
        <div style="display: flex; align-items: center; gap: 16px;">
          <h3 style="margin: 0;">集群部署管理</h3>
          <el-select
            v-model="selectedMerchantId"
            placeholder="选择商户"
            filterable
            style="width: 200px;"
          >
            <el-option
              v-for="m in merchantList"
              :key="m.id"
              :label="m.name"
              :value="m.id"
            />
          </el-select>
        </div>
        <div style="display: flex; gap: 8px;">
          <el-button :loading="healthLoading" @click="checkAllHealth" :disabled="clusterNodes.length === 0">
            健康检查
          </el-button>
          <el-button type="warning" :loading="gostSyncing" @click="doSyncGost" :disabled="appNodes.length === 0">
            同步 GOST
          </el-button>
          <el-button type="danger" :loading="gostRestarting" @click="doRestartGost" :disabled="gostRelations.length === 0">
            重启 GOST
          </el-button>
          <el-button type="primary" @click="openDeployDialog" :disabled="!selectedMerchantId">
            部署节点
          </el-button>
        </div>
      </div>
    </el-card>

    <!-- 部署指引 -->
    <el-alert
      v-if="selectedMerchantId"
      :title="nextStep"
      :type="clusterNodes.length === 0 ? 'info' : (deployProgress.hasDb && deployProgress.hasMinio && deployProgress.hasApp ? 'success' : 'warning')"
      :closable="false"
      show-icon
      class="mb-4"
    />

    <!-- 部署进度步骤 -->
    <el-card shadow="never" class="mb-4" v-if="selectedMerchantId">
      <div class="deploy-steps">
        <div class="step-item" :class="{ done: deployProgress.hasDb }">
          <div class="step-circle">{{ deployProgress.hasDb ? '&#10003;' : '1' }}</div>
          <div class="step-label">DB 节点</div>
          <div class="step-desc">MySQL + Redis</div>
        </div>
        <div class="step-line" :class="{ done: deployProgress.hasDb }" />
        <div class="step-item" :class="{ done: deployProgress.hasMinio }">
          <div class="step-circle">{{ deployProgress.hasMinio ? '&#10003;' : '2' }}</div>
          <div class="step-label">MinIO 节点</div>
          <div class="step-desc">对象存储</div>
        </div>
        <div class="step-line" :class="{ done: deployProgress.hasMinio }" />
        <div class="step-item" :class="{ done: deployProgress.hasApp }">
          <div class="step-circle">{{ deployProgress.hasApp ? '&#10003;' : '3' }}</div>
          <div class="step-label">App 节点</div>
          <div class="step-desc">WuKongIM + Server</div>
        </div>
      </div>
    </el-card>

    <!-- 集群拓扑 -->
    <div v-loading="topologyLoading">
      <!-- All-in-One 节点 -->
      <el-card shadow="never" class="mb-4" v-if="allinoneNodes.length > 0">
        <template #header>
          <div style="display: flex; align-items: center; gap: 8px;">
            <el-tag type="primary" size="small">All-in-One</el-tag>
            <span>全量节点 ({{ allinoneNodes.length }})</span>
          </div>
        </template>
        <el-table :data="allinoneNodes" stripe border size="small">
          <el-table-column prop="server_name" label="服务器" min-width="120" />
          <el-table-column prop="server_host" label="公网 IP" width="140" />
          <el-table-column prop="private_ip" label="内网 IP" width="140" />
          <el-table-column prop="wk_node_id" label="WK 节点 ID" width="110" align="center" />
          <el-table-column label="健康" width="80" align="center">
            <template #default="{ row }">
              <el-tag :type="healthTag(row.server_id).type" size="small">{{ healthTag(row.server_id).text }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="部署时间" width="170">
            <template #default="{ row }">{{ formatTime(row.deployed_at) }}</template>
          </el-table-column>
        </el-table>
      </el-card>

      <!-- DB 节点 -->
      <el-card shadow="never" class="mb-4" v-if="dbNodes.length > 0">
        <template #header>
          <div style="display: flex; align-items: center; gap: 8px;">
            <el-tag type="warning" size="small">DB</el-tag>
            <span>数据库节点 ({{ dbNodes.length }})</span>
          </div>
        </template>
        <el-table :data="dbNodes" stripe border size="small">
          <el-table-column prop="server_name" label="服务器" min-width="120" />
          <el-table-column prop="server_host" label="公网 IP" width="140" />
          <el-table-column prop="private_ip" label="内网 IP" width="140">
            <template #default="{ row }">
              <span style="font-weight: 600; color: #e6a23c;">{{ row.private_ip }}</span>
            </template>
          </el-table-column>
          <el-table-column label="健康" width="80" align="center">
            <template #default="{ row }">
              <el-tag :type="healthTag(row.server_id).type" size="small">{{ healthTag(row.server_id).text }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="部署时间" width="170">
            <template #default="{ row }">{{ formatTime(row.deployed_at) }}</template>
          </el-table-column>
        </el-table>
      </el-card>

      <!-- MinIO 节点 -->
      <el-card shadow="never" class="mb-4" v-if="minioNodes.length > 0">
        <template #header>
          <div style="display: flex; align-items: center; gap: 8px;">
            <el-tag type="info" size="small">MinIO</el-tag>
            <span>对象存储节点 ({{ minioNodes.length }})</span>
          </div>
        </template>
        <el-table :data="minioNodes" stripe border size="small">
          <el-table-column prop="server_name" label="服务器" min-width="120" />
          <el-table-column prop="server_host" label="公网 IP" width="140" />
          <el-table-column prop="private_ip" label="内网 IP" width="140">
            <template #default="{ row }">
              <span style="font-weight: 600; color: #909399;">{{ row.private_ip }}</span>
            </template>
          </el-table-column>
          <el-table-column label="健康" width="80" align="center">
            <template #default="{ row }">
              <el-tag :type="healthTag(row.server_id).type" size="small">{{ healthTag(row.server_id).text }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="部署时间" width="170">
            <template #default="{ row }">{{ formatTime(row.deployed_at) }}</template>
          </el-table-column>
        </el-table>
      </el-card>

      <!-- App 节点 -->
      <el-card shadow="never" class="mb-4" v-if="appNodes.length > 0">
        <template #header>
          <div style="display: flex; align-items: center; gap: 8px;">
            <el-tag type="success" size="small">App</el-tag>
            <span>应用节点 ({{ appNodes.length }})</span>
          </div>
        </template>
        <el-table :data="appNodes" stripe border size="small">
          <el-table-column prop="server_name" label="服务器" min-width="120" />
          <el-table-column prop="server_host" label="公网 IP" width="130" />
          <el-table-column prop="private_ip" label="内网 IP" width="130" />
          <el-table-column prop="wk_node_id" label="WK 节点 ID" width="110" align="center" />
          <el-table-column prop="db_host" label="DB 内网" width="130" />
          <el-table-column prop="minio_host" label="MinIO 内网" width="130" />
          <el-table-column label="健康" width="80" align="center">
            <template #default="{ row }">
              <el-tag :type="healthTag(row.server_id).type" size="small">{{ healthTag(row.server_id).text }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="部署时间" width="170">
            <template #default="{ row }">{{ formatTime(row.deployed_at) }}</template>
          </el-table-column>
        </el-table>
      </el-card>

      <!-- GOST 服务器关联 -->
      <el-card shadow="never" class="mb-4" v-if="selectedMerchantId" v-loading="gostRelationLoading">
        <template #header>
          <div style="display: flex; align-items: center; justify-content: space-between;">
            <div style="display: flex; align-items: center; gap: 8px;">
              <el-tag type="danger" size="small">GOST</el-tag>
              <span>GOST 服务器关联 ({{ gostRelations.length }})</span>
            </div>
            <el-button type="primary" size="small" @click="openAddGostDialog">关联服务器</el-button>
          </div>
        </template>
        <el-table v-if="gostRelations.length > 0" :data="gostRelations" stripe border size="small">
          <el-table-column prop="server_name" label="服务器名称" min-width="120" />
          <el-table-column prop="server_host" label="IP 地址" width="140" />
          <el-table-column label="主服务器" width="90" align="center">
            <template #default="{ row }">
              <el-tag v-if="row.is_primary === 1" type="success" size="small">主</el-tag>
              <span v-else style="color: #909399;">备</span>
            </template>
          </el-table-column>
          <el-table-column label="状态" width="80" align="center">
            <template #default="{ row }">
              <el-tag :type="row.status === 1 ? 'success' : 'info'" size="small">
                {{ row.status === 1 ? '启用' : '禁用' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="80" align="center">
            <template #default="{ row }">
              <el-button type="danger" text size="small" @click="removeGostRelation(row)">解除</el-button>
            </template>
          </el-table-column>
        </el-table>
        <el-empty v-else description="暂无关联的 GOST 服务器" :image-size="60">
          <el-button type="primary" size="small" @click="openAddGostDialog">关联服务器</el-button>
        </el-empty>
      </el-card>

      <!-- OSS 配置管理 -->
      <el-card shadow="never" class="mb-4" v-if="selectedMerchantId" v-loading="ossConfigLoading">
        <template #header>
          <div style="display: flex; align-items: center; justify-content: space-between;">
            <div style="display: flex; align-items: center; gap: 8px;">
              <el-tag type="primary" size="small">OSS</el-tag>
              <span>OSS 配置 ({{ ossConfigs.length }})</span>
            </div>
            <el-button type="primary" size="small" @click="openAddOssDialog">绑定 Bucket</el-button>
          </div>
        </template>
        <el-table v-if="ossConfigs.length > 0" :data="ossConfigs" stripe border size="small">
          <el-table-column prop="name" label="名称" min-width="100" />
          <el-table-column prop="cloud_account_name" label="云账号" min-width="100" />
          <el-table-column prop="cloud_type" label="云类型" width="80" align="center">
            <template #default="{ row }">
              <el-tag size="small" :type="row.cloud_type === 'aws' ? 'warning' : row.cloud_type === 'aliyun' ? 'primary' : 'info'">
                {{ row.cloud_type }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="bucket" label="Bucket" min-width="120" />
          <el-table-column prop="region" label="区域" width="120" />
          <el-table-column label="默认" width="60" align="center">
            <template #default="{ row }">
              <el-tag v-if="row.is_default === 1" type="success" size="small">是</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="状态" width="70" align="center">
            <template #default="{ row }">
              <el-tag :type="row.status === 1 ? 'success' : 'info'" size="small">
                {{ row.status === 1 ? '启用' : '禁用' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="80" align="center">
            <template #default="{ row }">
              <el-button type="danger" text size="small" @click="removeOssConfig(row)">解除</el-button>
            </template>
          </el-table-column>
        </el-table>
        <el-empty v-else description="暂无 OSS 绑定，请从已有上传目标中选择绑定" :image-size="60">
          <el-button type="primary" size="small" @click="openAddOssDialog">绑定 Bucket</el-button>
        </el-empty>
      </el-card>

      <!-- 空状态 -->
      <el-card shadow="never" v-if="selectedMerchantId && clusterNodes.length === 0 && !topologyLoading">
        <el-empty description="暂无集群节点，请点击「部署节点」开始">
          <el-button type="primary" @click="openDeployDialog">部署节点</el-button>
        </el-empty>
      </el-card>

      <el-card shadow="never" v-if="!selectedMerchantId">
        <el-empty description="请先在顶部选择商户" />
      </el-card>
    </div>

    <!-- 部署对话框 -->
    <el-dialog v-model="deployDialogVisible" title="部署集群节点" width="620px" :close-on-click-modal="false">
      <el-form :model="deployForm" label-width="130px" label-position="right">
        <el-form-item label="服务器来源">
          <el-radio-group v-model="createNewServer">
            <el-radio-button :value="false">选择已有服务器</el-radio-button>
            <el-radio-button :value="true">创建新 EC2</el-radio-button>
          </el-radio-group>
        </el-form-item>

        <!-- 选择已有服务器 -->
        <el-form-item v-if="!createNewServer" label="目标服务器" required>
          <el-select v-model="deployForm.server_id" placeholder="选择服务器" filterable style="width: 100%">
            <el-option-group label="可用服务器">
              <el-option
                v-for="s in availableServers"
                :key="s.id"
                :label="`${s.name} (${s.host})`"
                :value="s.id"
              />
            </el-option-group>
            <el-option-group label="已部署（重新部署）" v-if="serverList.filter(s => deployedServerIds.has(s.id)).length > 0">
              <el-option
                v-for="s in serverList.filter(s => deployedServerIds.has(s.id))"
                :key="s.id"
                :label="`${s.name} (${s.host}) - 已部署`"
                :value="s.id"
                style="color: #909399;"
              />
            </el-option-group>
          </el-select>
        </el-form-item>

        <!-- 创建新 EC2 -->
        <template v-if="createNewServer">
          <el-form-item label="AMI ID" required>
            <el-input v-model="deployForm.ami_id" placeholder="ami-xxxxxxxxxxxxxxxxx" />
          </el-form-item>
          <el-row :gutter="12">
            <el-col :span="12">
              <el-form-item label="实例类型">
                <el-input v-model="deployForm.instance_type" :placeholder="deployForm.node_role === 'db' ? 'r5.large' : deployForm.node_role === 'minio' ? 't3.medium' : 't3.large'" />
              </el-form-item>
            </el-col>
            <el-col :span="12">
              <el-form-item label="磁盘(GB)">
                <el-input-number v-model="deployForm.volume_size_gib" :min="20" :max="2000" :placeholder="deployForm.node_role === 'db' ? '100' : deployForm.node_role === 'minio' ? '200' : '30'" style="width: 100%" />
              </el-form-item>
            </el-col>
          </el-row>
          <el-row :gutter="12">
            <el-col :span="12">
              <el-form-item label="AWS 账号" required>
                <el-select v-model="deployForm.cloud_account_id" placeholder="选择 AWS 账号" filterable style="width: 100%">
                  <el-option-group v-if="merchantAwsAccounts.length > 0" label="商户账号">
                    <el-option v-for="acc in merchantAwsAccounts" :key="acc.id" :label="`${acc.name} (${acc.access_key_id})`" :value="acc.id" />
                  </el-option-group>
                  <el-option-group label="系统账号">
                    <el-option v-for="acc in systemAwsAccounts" :key="acc.id" :label="`${acc.name} (${acc.access_key_id})`" :value="acc.id" />
                  </el-option-group>
                </el-select>
              </el-form-item>
            </el-col>
            <el-col :span="12">
              <el-form-item label="区域">
                <el-select v-model="deployForm.region_id" style="width: 100%">
                  <el-option v-for="r in awsRegions" :key="r.id" :label="r.name" :value="r.id" />
                </el-select>
              </el-form-item>
            </el-col>
          </el-row>
          <el-row :gutter="12">
            <el-col :span="12">
              <el-form-item label="Key Name">
                <el-input v-model="deployForm.key_name" placeholder="SSH Key 名称" />
              </el-form-item>
            </el-col>
            <el-col :span="12">
              <el-form-item label="Subnet ID">
                <el-input v-model="deployForm.subnet_id" placeholder="子网 ID" />
              </el-form-item>
            </el-col>
          </el-row>
        </template>

        <el-form-item label="商户" required>
          <el-select v-model="deployForm.merchant_id" placeholder="选择商户" filterable style="width: 100%" disabled>
            <el-option
              v-for="m in merchantList"
              :key="m.id"
              :label="m.name"
              :value="m.id"
            />
          </el-select>
        </el-form-item>

        <el-form-item label="节点角色" required>
          <el-radio-group v-model="deployForm.node_role">
            <el-radio-button value="db">DB 节点</el-radio-button>
            <el-radio-button value="minio">MinIO 节点</el-radio-button>
            <el-radio-button value="app">App 节点</el-radio-button>
            <el-radio-button value="allinone">全量</el-radio-button>
          </el-radio-group>
          <div style="color: #909399; font-size: 12px; margin-top: 4px;">
            <template v-if="deployForm.node_role === 'db'">MySQL + Redis 数据层</template>
            <template v-else-if="deployForm.node_role === 'minio'">MinIO 对象存储</template>
            <template v-else-if="deployForm.node_role === 'app'">WuKongIM + tsdd-server + web + manager</template>
            <template v-else>所有服务部署在同一台机器（测试用）</template>
          </div>
        </el-form-item>

        <!-- App 节点额外配置 -->
        <template v-if="deployForm.node_role === 'app'">
          <el-form-item label="DB 内网 IP" required>
            <el-input v-model="deployForm.db_host" placeholder="如 172.31.9.143">
              <template #append v-if="dbNodes.length > 0">
                <el-tooltip content="自动填充自 DB 节点">
                  <el-tag size="small" type="warning">自动</el-tag>
                </el-tooltip>
              </template>
            </el-input>
          </el-form-item>

          <el-form-item label="MinIO 内网 IP">
            <el-input v-model="deployForm.minio_host" placeholder="留空则与 DB 节点相同">
              <template #append v-if="minioNodes.length > 0">
                <el-tooltip content="自动填充自 MinIO 节点">
                  <el-tag size="small" type="info">自动</el-tag>
                </el-tooltip>
              </template>
            </el-input>
          </el-form-item>
        </template>

        <!-- WuKongIM 集群配置（App 和 Allinone） -->
        <template v-if="deployForm.node_role === 'app' || deployForm.node_role === 'allinone'">
          <el-form-item label="WK 节点 ID">
            <el-input-number v-model="deployForm.wk_node_id" :min="1001" :max="9999" style="width: 100%" />
            <div style="color: #909399; font-size: 12px; margin-top: 2px;">
              WuKongIM 集群节点 ID，已自动递增
            </div>
          </el-form-item>

          <el-form-item label="种子节点">
            <el-input v-model="deployForm.wk_seed_node" placeholder="首个节点留空">
              <template #append v-if="deployForm.wk_seed_node">
                <el-tooltip content="自动发现自已部署的 App 节点">
                  <el-tag size="small" type="success">自动</el-tag>
                </el-tooltip>
              </template>
            </el-input>
            <div style="color: #909399; font-size: 12px; margin-top: 2px;">
              <template v-if="appNodes.length === 0">首个 App 节点，无需填写种子节点</template>
              <template v-else>自动发现：加入 {{ appNodes[0].server_name }} 的集群</template>
            </div>
          </el-form-item>
        </template>

        <el-form-item label="强制重置">
          <el-switch v-model="deployForm.force_reset" />
          <span style="color: #f56c6c; font-size: 12px; margin-left: 8px;" v-if="deployForm.force_reset">
            将删除现有容器和数据!
          </span>
        </el-form-item>
      </el-form>

      <template #footer>
        <el-button @click="deployDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="deploying" @click="submitDeploy">
          {{ deploying ? (createNewServer ? '创建 EC2 + 部署中...' : '部署中...') : (createNewServer ? '创建 EC2 + 部署' : '开始部署') }}
        </el-button>
      </template>
    </el-dialog>

    <!-- 部署结果对话框 -->
    <el-dialog v-model="resultDialogVisible" title="部署结果" width="700px">
      <template v-if="deployResult">
        <el-result
          :icon="deployResult.success ? 'success' : 'error'"
          :title="deployResult.success ? '部署成功' : '部署失败'"
          :sub-title="deployResult.message"
        />

        <div v-if="deployResult.steps && deployResult.steps.length" style="margin-top: 16px;">
          <h4>部署步骤</h4>
          <el-timeline>
            <el-timeline-item
              v-for="(step, idx) in deployResult.steps"
              :key="idx"
              :color="stepColor(step.status)"
            >
              <div style="display: flex; align-items: center; gap: 8px;">
                <strong>{{ step.name }}</strong>
                <el-tag :type="step.status === 'success' ? 'success' : step.status === 'failed' ? 'danger' : step.status === 'warning' ? 'warning' : 'info'" size="small">
                  {{ step.status }}
                </el-tag>
              </div>
              <div v-if="step.message" style="color: #606266; margin-top: 4px;">{{ step.message }}</div>
              <el-collapse v-if="step.output" style="margin-top: 4px;">
                <el-collapse-item title="查看输出">
                  <pre style="background: #f5f7fa; padding: 8px; border-radius: 4px; font-size: 12px; max-height: 200px; overflow: auto; white-space: pre-wrap;">{{ step.output }}</pre>
                </el-collapse-item>
              </el-collapse>
            </el-timeline-item>
          </el-timeline>
        </div>

        <div v-if="deployResult.api_url" style="margin-top: 16px; padding: 12px; background: #f0f9eb; border-radius: 4px;">
          <div><strong>API:</strong> {{ deployResult.api_url }}</div>
          <div v-if="deployResult.web_url"><strong>Web:</strong> {{ deployResult.web_url }}</div>
          <div v-if="deployResult.admin_url"><strong>Admin:</strong> {{ deployResult.admin_url }}</div>
        </div>
      </template>

      <template #footer>
        <el-button @click="resultDialogVisible = false">关闭</el-button>
      </template>
    </el-dialog>

    <!-- GOST 同步结果对话框 -->
    <el-dialog v-model="gostResultDialogVisible" title="GOST 同步结果" width="750px">
      <div v-for="(result, idx) in gostResults" :key="idx" style="margin-bottom: 16px;">
        <div style="display: flex; align-items: center; gap: 8px; margin-bottom: 8px;">
          <el-tag :type="result.success ? 'success' : 'danger'" size="small">
            {{ result.success ? '成功' : '失败' }}
          </el-tag>
          <strong>{{ result.server_name }}</strong>
          <span style="color: #909399;">({{ result.server_host }})</span>
          <el-tag type="info" size="small">
            {{ result.forward_type === 'encrypted' ? 'relay+tls' : 'TCP 直连' }}
          </el-tag>
          <span style="color: #909399; font-size: 12px;">→ {{ result.target_ip }}</span>
        </div>
        <el-table :data="result.ports || []" stripe border size="small" style="margin-left: 16px;">
          <el-table-column label="协议" width="90" align="center">
            <template #default="{ row }">
              <el-tag
                :type="row.name === 'tcp' ? 'primary' : row.name === 'ws' ? 'success' : row.name === 'http' ? 'warning' : 'info'"
                size="small"
              >{{ row.name.toUpperCase() }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="监听端口" width="120" align="center">
            <template #default="{ row }">
              <span style="font-family: monospace; font-weight: 600;">:{{ row.listen_port }}</span>
            </template>
          </el-table-column>
          <el-table-column label="" width="40" align="center">
            <template #default>→</template>
          </el-table-column>
          <el-table-column label="目标端口" width="120" align="center">
            <template #default="{ row }">
              <span style="font-family: monospace; font-weight: 600;">:{{ row.target_port }}</span>
            </template>
          </el-table-column>
          <el-table-column label="说明" min-width="160">
            <template #default="{ row }">
              <span style="color: #909399; font-size: 12px;">
                {{ row.name === 'tcp' ? 'WuKongIM TCP 长连接' : row.name === 'ws' ? 'WuKongIM WebSocket' : row.name === 'http' ? 'tsdd-server API' : 'MinIO S3 对象存储' }}
              </span>
            </template>
          </el-table-column>
        </el-table>
        <div v-if="result.error" style="color: #f56c6c; font-size: 12px; margin-top: 4px; margin-left: 16px;">
          {{ result.error }}
        </div>
      </div>
      <template #footer>
        <el-button @click="gostResultDialogVisible = false">关闭</el-button>
      </template>
    </el-dialog>

    <!-- 关联 GOST 服务器对话框 -->
    <el-dialog v-model="addGostDialogVisible" title="关联 GOST 服务器" width="500px">
      <el-form label-width="120px">
        <el-form-item label="GOST 服务器" required>
          <el-select v-model="addGostForm.server_id" placeholder="选择系统服务器" filterable style="width: 100%">
            <el-option
              v-for="s in availableGostServers"
              :key="s.id"
              :label="`${s.name} (${s.host})`"
              :value="s.id"
            />
          </el-select>
          <div v-if="availableGostServers.length === 0" style="color: #e6a23c; font-size: 12px; margin-top: 4px;">
            没有可关联的系统服务器（已全部关联或无系统服务器）
          </div>
        </el-form-item>
        <el-form-item label="设为主服务器">
          <el-switch v-model="addGostForm.is_primary" :active-value="1" :inactive-value="0" />
          <span style="color: #909399; font-size: 12px; margin-left: 8px;">
            主服务器优先用于客户端连接
          </span>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="addGostDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="addGostLoading" :disabled="!addGostForm.server_id" @click="submitAddGost">
          关联
        </el-button>
      </template>
    </el-dialog>

    <!-- 绑定 OSS Bucket 对话框 -->
    <el-dialog v-model="addOssDialogVisible" title="绑定 Bucket 到商户" width="560px" :close-on-click-modal="false">
      <el-form label-width="110px">
        <el-form-item label="选择目标" required>
          <el-select v-model="selectedTargetId" placeholder="从已有上传目标中选择" filterable style="width: 100%">
            <el-option
              v-for="t in availableTargets"
              :key="t.id"
              :label="`${t.name} (${t.cloud_type} / ${t.bucket})`"
              :value="t.id"
            />
          </el-select>
          <div v-if="availableTargets.length === 0 && allTargets.length > 0" style="color: #e6a23c; font-size: 12px; margin-top: 4px;">
            所有上传目标已绑定到此商户
          </div>
          <div v-if="allTargets.length === 0" style="color: #e6a23c; font-size: 12px; margin-top: 4px;">
            暂无上传目标，请先在「工具」页面添加
          </div>
        </el-form-item>
        <template v-if="selectedTarget">
          <el-form-item label="云账号">
            <span>{{ selectedTarget.account_name }} ({{ selectedTarget.cloud_type }})</span>
          </el-form-item>
          <el-form-item label="Bucket">
            <span style="font-weight: 600;">{{ selectedTarget.bucket }}</span>
          </el-form-item>
          <el-form-item label="区域">
            <span>{{ selectedTarget.region_id || '-' }}</span>
          </el-form-item>
        </template>
        <el-form-item label="设为默认">
          <el-switch v-model="addOssIsDefault" :active-value="1" :inactive-value="0" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="addOssDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="addOssLoading" :disabled="!selectedTargetId" @click="submitAddOss">
          绑定
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.mb-4 {
  margin-bottom: 16px;
}

.deploy-steps {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 8px 0;
}

.step-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  min-width: 100px;
}

.step-circle {
  width: 36px;
  height: 36px;
  border-radius: 50%;
  background: #dcdfe6;
  color: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 14px;
  font-weight: 600;
  margin-bottom: 6px;
  transition: background 0.3s;
}

.step-item.done .step-circle {
  background: #67c23a;
}

.step-label {
  font-size: 13px;
  font-weight: 600;
  color: #303133;
}

.step-desc {
  font-size: 11px;
  color: #909399;
  margin-top: 2px;
}

.step-line {
  flex: 1;
  height: 2px;
  background: #dcdfe6;
  margin: 0 12px;
  margin-bottom: 30px;
  transition: background 0.3s;
}

.step-line.done {
  background: #67c23a;
}
</style>
