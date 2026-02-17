<script lang="ts" setup>
import type { DeployGostServerReq, ServerResp } from "@@/apis/deploy/type"
import type { MerchantResp } from "@@/apis/merchant/type"
import type { VxeFormInstance, VxeFormProps, VxeGridInstance, VxeGridProps, VxeModalInstance, VxeModalProps } from "vxe-table"
import { createGostServiceByAPI, deleteGostServiceByAPI, getGostServiceDetail, getServerList, listGostChains, listGostServices, updateGostServiceDetail, setupGostForward, clearGostForward, getGostForwardStatus, getProgramConfig, updateProgramConfig, getNginxCacheStatus, clearNginxCache } from "@@/apis/deploy"
import { getMerchantList } from "@@/apis/merchant"
import { getCloudAccountList } from "@@/apis/cloud_account"
import { createStreamRequest } from "@/http/axios"

defineOptions({
  name: "GostService"
})

// ========== GOST 一键部署 ==========
const deployDialogVisible = ref(false)
const deployLogs = ref<string[]>([])
const isDeploying = ref(false)
const deployLogRef = ref<HTMLDivElement>()

// 云账号列表
const cloudAccountList = ref<any[]>([])

// 部署表单
const deployForm = reactive<DeployGostServerReq>({
  cloud_account_id: 0,
  region_id: "ap-southeast-1",
  instance_type: "",
  image_id: "",
  server_name: "",
  group_id: 0,
  password: "",
  bandwidth: "5"
})

// 加载云账号列表
async function loadCloudAccounts() {
  try {
    const res = await getCloudAccountList({ page: 1, size: 100 })
    cloudAccountList.value = Array.isArray(res.data?.list) ? res.data.list : []
  } catch (e) {
    console.error("加载云账号失败:", e)
  }
}

// 打开部署弹窗
function openDeployDialog() {
  deployLogs.value = []
  deployForm.cloud_account_id = cloudAccountList.value[0]?.id || 0
  deployForm.region_id = "ap-southeast-1"
  deployForm.server_name = `gost-${Date.now()}`
  deployDialogVisible.value = true
}

// 执行一键部署
function startDeploy() {
  if (!deployForm.cloud_account_id) {
    ElMessage.warning("请选择云账号")
    return
  }
  if (!deployForm.region_id) {
    ElMessage.warning("请选择地区")
    return
  }

  isDeploying.value = true
  deployLogs.value = ["开始部署 GOST 服务器..."]

  const cancel = createStreamRequest(
    {
      url: "deploy/gost/deploy",
      method: "POST",
      data: deployForm
    },
    (data: any, isComplete?: boolean) => {
      if (data?.message) {
        deployLogs.value.push(data.message)
        scrollToBottom()
      }
      if (isComplete) {
        isDeploying.value = false
        if (data?.success) {
          ElMessage.success("部署完成!")
          loadServerList() // 刷新服务器列表
        }
      }
    },
    (error: any) => {
      isDeploying.value = false
      deployLogs.value.push(`错误: ${error?.message || error}`)
      ElMessage.error("部署失败")
    }
  )
}

// 滚动到日志底部
function scrollToBottom() {
  nextTick(() => {
    if (deployLogRef.value) {
      deployLogRef.value.scrollTop = deployLogRef.value.scrollHeight
    }
  })
}

// AWS 地区列表（用于一键部署）
const regionOptions = [
  { value: "ap-southeast-1", label: "新加坡" },
  { value: "ap-northeast-1", label: "东京" },
  { value: "ap-east-1", label: "香港" },
  { value: "us-west-2", label: "美西(俄勒冈)" },
  { value: "us-east-1", label: "美东(弗吉尼亚)" },
  { value: "eu-west-1", label: "欧洲(爱尔兰)" }
]

// ========== 服务器选择（仅系统服务器） ==========
const allServerList = ref<ServerResp[]>([])
const serverList = ref<ServerResp[]>([])
const selectedServerId = ref<number>(0)

// 仅 Service 视图

// 获取系统服务器列表
async function loadServerList() {
  try {
    const res = await getServerList({ page: 1, size: 5000 })
    const list = res.data?.list ?? []
    allServerList.value = Array.isArray(list) ? list : []
    serverList.value = (allServerList.value || []).filter(s => s.server_type === 2)
    if (serverList.value.length > 0 && !selectedServerId.value) {
      selectedServerId.value = serverList.value[0].id
    }
    if (selectedServerId.value) {
      await loadRecords()
    }
  } catch (error) {
    console.error("获取服务器列表失败:", error)
  }
}

// ========== VXE Grid 配置 ==========
const xGridDom = ref<VxeGridInstance>()
const records = ref<any[]>([])
const chainMap = ref<Record<string, any>>({})
const pagination = reactive({ currentPage: 1, pageSize: 20, total: 0 })
const merchantList = ref<MerchantResp[]>([])
const selectedMerchantId = ref<number | undefined>(undefined)

// 根据 GOST 服务名称获取关联的商户
function getMerchantByServiceName(serviceName: string): MerchantResp | undefined {
  // 服务名称格式: tcp-relay-{port}
  const match = serviceName.match(/^tcp-relay-(\d+)$/)
  if (!match) return undefined
  const port = parseInt(match[1], 10)
  return merchantList.value.find(m => (m as any).port === port)
}

function buildColumns() {
  return [
    { field: "name", title: "名称", width: 200 },
    { title: "商户", width: 180, slots: { default: "service-merchant" } },
    { field: "addr", title: "监听地址", width: 180 },
    { title: "Chain", width: 320, slots: { default: "service-chain" } },
    { title: "类型", width: 140, slots: { default: "service-types" } },
    { title: "操作", width: 140, fixed: "right", slots: { default: "row-operate" } }
  ]
}

const xGridOpt: VxeGridProps = reactive({
  loading: false,
  autoResize: true,
  data: records,
  columns: buildColumns() as any,
  pagerConfig: {
    align: "right",
    currentPage: pagination.currentPage,
    pageSize: pagination.pageSize,
    pageSizes: [10, 20, 50, 100],
    total: pagination.total
  } as any,
  toolbarConfig: {
    refresh: true,
    slots: { buttons: "toolbar-btns" }
  }
})
function onSubmitCreateForm() {
  if (xCreateFormOpt.loading) return
  xCreateFormDom.value?.validate(async (errMap) => {
    if (errMap) return
    xCreateFormOpt.loading = true
    try {
      const { listen_port, forward_host, forward_port } = xCreateFormOpt.data as any
      await createGostServiceByAPI({ server_id: selectedServerId.value, listen_port: Number(listen_port), forward_host: String(forward_host), forward_port: Number(forward_port) })
      xCreateModalDom.value?.close()
      ElMessage.success("创建成功")
      loadRecords()
    } finally {
      xCreateFormOpt.loading = false
    }
  })
}

// 仅初始/分页变化刷新

// 加载列表（service 或 chain）
async function loadRecords() {
  if (!selectedServerId.value) {
    ElMessage.warning("请选择服务器")
    return
  }
  xGridOpt.loading = true
  try {
    const [svcRes, chainRes] = await Promise.all([
      listGostServices({ server_id: selectedServerId.value, page: pagination.currentPage, size: pagination.pageSize, port: (() => {
        const mid = selectedMerchantId.value
        if (!mid) return undefined
        const m = merchantList.value.find(mm => mm.id === mid)
        return (m as any)?.port || undefined
      })() }),
      listGostChains({ server_id: selectedServerId.value })
    ])
    const map: Record<string, any> = {}
    const chainList = chainRes.data?.list
    ;(Array.isArray(chainList) ? chainList : []).forEach((c: any) => {
      map[c.name] = c
    })
    chainMap.value = map
    records.value = Array.isArray(svcRes.data?.list) ? svcRes.data.list : []
    // 若选择了商户，则过滤出名称为 tcp-relay-{port} 的 service
    if (selectedMerchantId.value) {
      const m = merchantList.value.find(m => m.id === selectedMerchantId.value)
      if (m && (m as any).port) {
        const targetName = `tcp-relay-${(m as any).port}`
        records.value = records.value.filter((s: any) => s.name === targetName)
      }
    }
    pagination.total = svcRes.data.count || 0
    if (xGridOpt.pagerConfig) {
      ;(xGridOpt.pagerConfig as any).total = pagination.total
      ;(xGridOpt.pagerConfig as any).currentPage = pagination.currentPage
      ;(xGridOpt.pagerConfig as any).pageSize = pagination.pageSize
    }
  } catch (error: any) {
    console.error("查询列表失败:", error)
    // 检查是否是连接被拒绝的错误
    const errMsg = error?.message || String(error)
    if (errMsg.includes("connection refused") || errMsg.includes("connect:")) {
      ElMessage.error("GOST 服务未运行或端口未开放，请先安装 GOST")
    } else if (errMsg.includes("timeout")) {
      ElMessage.error("连接超时，请检查服务器网络")
    } else {
      ElMessage.error("查询失败: " + errMsg)
    }
    // 清空列表
    records.value = []
    pagination.total = 0
  } finally {
    xGridOpt.loading = false
  }
}

// ========== Modal & Form 配置（JSON 编辑） ==========
const xModalDom = ref<VxeModalInstance>()
const xFormDom = ref<VxeFormInstance>()
const xCreateModalDom = ref<VxeModalInstance>()
const xCreateFormDom = ref<VxeFormInstance>()

const xModalOpt: VxeModalProps = reactive({
  title: "",
  showClose: true,
  escClosable: true,
  maskClosable: false,
  width: 800,
  beforeHideMethod: () => {
    xFormDom.value?.clearValidate()
    return Promise.resolve()
  }
})

const xCreateModalOpt: VxeModalProps = reactive({
  title: "新增 Service",
  showClose: true,
  escClosable: true,
  maskClosable: false,
  width: 600,
  beforeHideMethod: () => {
    xCreateFormDom.value?.clearValidate()
    return Promise.resolve()
  }
})

const xCreateFormOpt: VxeFormProps = reactive({
  span: 24,
  titleWidth: "120px",
  loading: false,
  titleColon: true,
  data: {
    listen_port: 0,
    forward_host: "",
    forward_port: 0
  },
  items: [
    { field: "listen_port", title: "监听端口", itemRender: { name: "$input", props: { type: "number", placeholder: "例如: 10443" } } },
    { field: "forward_host", title: "转发目标IP", itemRender: { name: "$input", props: { placeholder: "例如: 1.2.3.4" } } },
    { field: "forward_port", title: "转发目标端口", itemRender: { name: "$input", props: { type: "number", placeholder: "例如: 10444" } } },
    {
      align: "right",
      itemRender: { name: "$buttons", children: [
        { props: { content: "取消" }, events: { click: () => xCreateModalDom.value?.close() } },
        { props: { type: "submit", content: "确定", status: "primary" }, events: { click: () => onSubmitCreateForm() } }
      ] }
    }
  ],
  rules: {
    listen_port: [
      { required: true, message: "请输入监听端口" },
      { validator: ({ itemValue }) => { if (itemValue < 1 || itemValue > 65535) return new Error("端口范围: 1-65535") } }
    ],
    forward_host: [
      { required: true, message: "请输入转发目标IP" }
    ],
    forward_port: [
      { required: true, message: "请输入转发目标端口" },
      { validator: ({ itemValue }) => { if (itemValue < 1 || itemValue > 65535) return new Error("端口范围: 1-65535") } }
    ]
  }
})

// 分页变更
function handlePageChange({ currentPage, pageSize }: any) {
  pagination.currentPage = currentPage
  pagination.pageSize = pageSize
  loadRecords()
}

const xFormOpt: VxeFormProps = reactive({
  span: 24,
  titleWidth: "120px",
  loading: false,
  titleColon: true,
  data: {
    name: "",
    jsonText: ""
  },
  items: [
    {
      field: "name",
      title: "名称",
      itemRender: {
        name: "$input",
        props: { disabled: true }
      }
    },
    {
      field: "jsonText",
      title: "配置(JSON)",
      span: 24,
      itemRender: {
        name: "$textarea",
        props: { rows: 18, placeholder: "在此编辑 JSON 配置" }
      }
    },
    {
      align: "right",
      itemRender: {
        name: "$buttons",
        children: [
          { props: { content: "取消" }, events: { click: () => xModalDom.value?.close() } },
          { props: { type: "submit", content: "保存", status: "primary" }, events: { click: () => crudStore.onSubmitForm() } }
        ]
      }
    }
  ],
  rules: {
    jsonText: [
      { required: true, message: "请输入配置 JSON" },
      {
        validator: ({ itemValue }) => {
          try {
            JSON.parse(itemValue)
          } catch {
            return new Error("JSON 格式不正确")
          }
        }
      }
    ]
  }
})

// ========== CRUD ==========
const crudStore = reactive({
  currentName: "",
  onShowModal: async (row: any) => {
    if (!selectedServerId.value) {
      ElMessage.warning("请选择服务器")
      return
    }
    xModalOpt.title = "编辑 Service"
    crudStore.currentName = row.name
    try {
      const res = await getGostServiceDetail({ server_id: selectedServerId.value, service_name: row.name })
      xFormOpt.data = {
        name: row.name,
        jsonText: JSON.stringify(res.data, null, 2)
      }
      xModalDom.value?.open()
      nextTick(() => {
        xFormDom.value?.clearValidate()
      })
    } catch (error: any) {
      const errMsg = error?.message || String(error)
      // axios 拦截器已弹 ElMessage，如果是通用 "Error" 则补充提示
      if (errMsg === "Error" || !errMsg) {
        ElMessage.error(`获取 ${row.name} 详情失败，请检查 GOST 服务是否正常运行`)
      }
      console.error("获取详情失败:", row.name, error)
    }
  },
  onSubmitForm: () => {
    if (xFormOpt.loading) return
    xFormDom.value?.validate(async (errMap) => {
      if (errMap) return
      let payload: any
      try {
        payload = JSON.parse((xFormOpt.data as any).jsonText)
      } catch {
        ElMessage.error("JSON 格式不正确")
        return
      }
      xFormOpt.loading = true
      try {
        const res = await updateGostServiceDetail({ server_id: selectedServerId.value, service_name: crudStore.currentName, config: payload })
        ElMessage.success((res as any).msg || (res as any).message || "保存成功")
        xModalDom.value?.close()
        loadRecords()
      } catch {
        // ignore
      } finally {
        xFormOpt.loading = false
      }
    })
  }
})

// 新增
async function onCreate() {
  if (!selectedServerId.value) {
    ElMessage.warning("请选择服务器")
    return
  }
  xCreateFormOpt.data = { listen_port: 0, forward_host: "", forward_port: 0 }
  xCreateModalOpt.title = "新增 Service"
  xCreateModalDom.value?.open()
}

// 删除
async function onDelete(row: any) {
  if (!selectedServerId.value) return
  const ok = await ElMessageBox.confirm(`确认删除 ${row.name} 吗？`, "提示", { type: "warning" }).catch(() => false)
  if (!ok) return
  try {
    await deleteGostServiceByAPI({ server_id: selectedServerId.value, service_name: row.name })
    ElMessage.success("删除成功")
    loadRecords()
  } catch {
    // ignore
  }
}
// 监听服务器/视图变化
watch(selectedServerId, () => {
  if (selectedServerId.value) {
    loadRecords()
  }
})

// ========== GOST 转发配置（一键部署） ==========
const forwardDialogVisible = ref(false)
const forwardStatusDialogVisible = ref(false)
const forwardForm = reactive({
  target_ip: "",
  portsStr: "", // 用字符串输入，提交时解析
  mode: "tls" as "tls" | "tcp" // 连接模式：tls(加密) 或 tcp(直连)
})
const forwardStatus = ref<any>(null)
const isSettingForward = ref(false)

// 获取当前选中的服务器信息
const selectedServer = computed(() => {
  if (!selectedServerId.value) return null
  return serverList.value.find(s => s.id === selectedServerId.value) || null
})

// 打开配置转发弹窗
function openForwardDialog() {
  if (!selectedServerId.value) {
    ElMessage.warning("请先选择服务器")
    return
  }
  forwardForm.target_ip = ""
  forwardForm.portsStr = ""
  forwardDialogVisible.value = true
}

// 解析端口字符串
function parsePorts(str: string): number[] {
  if (!str.trim()) return []
  return str.split(",").map(s => parseInt(s.trim(), 10)).filter(n => !isNaN(n) && n > 0 && n <= 65535)
}

// 执行配置转发
async function doSetupForward() {
  if (!forwardForm.target_ip) {
    ElMessage.warning("请输入目标服务器IP")
    return
  }
  isSettingForward.value = true
  const ports = parsePorts(forwardForm.portsStr)
  try {
    await setupGostForward({
      server_id: selectedServerId.value,
      target_ip: forwardForm.target_ip,
      ports: ports.length > 0 ? ports : undefined,
      mode: forwardForm.mode
    })
    ElMessage.success(`转发配置成功! (${forwardForm.mode === "tls" ? "TLS加密" : "TCP直连"})`)
    forwardDialogVisible.value = false
    loadRecords()
  } catch (e: any) {
    ElMessage.error(e?.message || "配置失败")
  } finally {
    isSettingForward.value = false
  }
}

// 清除转发规则
async function doClearForward() {
  if (!selectedServerId.value) {
    ElMessage.warning("请先选择服务器")
    return
  }
  const ok = await ElMessageBox.confirm("确认清除所有转发规则吗？", "提示", { type: "warning" }).catch(() => false)
  if (!ok) return
  try {
    await clearGostForward({ server_id: selectedServerId.value })
    ElMessage.success("转发规则已清除")
    loadRecords()
  } catch (e: any) {
    ElMessage.error(e?.message || "清除失败")
  }
}

// 查看转发状态
async function openForwardStatusDialog() {
  if (!selectedServerId.value) {
    ElMessage.warning("请先选择服务器")
    return
  }
  try {
    const res = await getGostForwardStatus(selectedServerId.value)
    forwardStatus.value = res.data
    forwardStatusDialogVisible.value = true
  } catch (e: any) {
    ElMessage.error(e?.message || "获取状态失败")
  }
}

// ========== GOST 配置文件查看/编辑 ==========
const configDialogVisible = ref(false)
const configContent = ref("")
const configPath = ref("")
const isConfigLoading = ref(false)
const isConfigSaving = ref(false)

async function openConfigDialog() {
  if (!selectedServerId.value) {
    ElMessage.warning("请先选择服务器")
    return
  }
  isConfigLoading.value = true
  configDialogVisible.value = true
  try {
    const res = await getProgramConfig({ server_id: selectedServerId.value, service_name: "gost" })
    configContent.value = res.data?.content || ""
    configPath.value = res.data?.config_path || ""
  } catch (e: any) {
    ElMessage.error(e?.message || "获取 GOST 配置失败")
    configContent.value = ""
  } finally {
    isConfigLoading.value = false
  }
}

async function saveConfig() {
  if (!selectedServerId.value) return
  isConfigSaving.value = true
  try {
    await updateProgramConfig({ server_id: selectedServerId.value, service_name: "gost", content: configContent.value })
    ElMessage.success("配置文件保存成功")
    configDialogVisible.value = false
  } catch (e: any) {
    ElMessage.error(e?.message || "保存失败")
  } finally {
    isConfigSaving.value = false
  }
}

// ========== Nginx 缓存管理 ==========
const cacheDialogVisible = ref(false)
const cacheStatus = ref<{ installed: boolean; running: boolean; cache_size: string } | null>(null)
const isCacheLoading = ref(false)
const isCacheClearing = ref(false)
const isNginxInstalling = ref(false)
const installLogs = ref<string[]>([])
const installLogRef = ref<HTMLDivElement>()

async function openCacheDialog() {
  if (!selectedServerId.value) {
    ElMessage.warning("请先选择服务器")
    return
  }
  cacheDialogVisible.value = true
  await loadCacheStatus()
}

async function loadCacheStatus() {
  isCacheLoading.value = true
  try {
    const res = await getNginxCacheStatus(selectedServerId.value)
    cacheStatus.value = res.data
  } catch (e: any) {
    ElMessage.error(e?.message || "获取缓存状态失败")
    cacheStatus.value = null
  } finally {
    isCacheLoading.value = false
  }
}

async function doClearCache() {
  if (!selectedServerId.value) return
  const ok = await ElMessageBox.confirm("确认清除所有 Nginx 缓存吗？", "提示", { type: "warning" }).catch(() => false)
  if (!ok) return
  isCacheClearing.value = true
  try {
    await clearNginxCache({ server_id: selectedServerId.value })
    ElMessage.success("缓存已清除")
    await loadCacheStatus()
  } catch (e: any) {
    ElMessage.error(e?.message || "清除失败")
  } finally {
    isCacheClearing.value = false
  }
}

function startInstallNginx() {
  if (!selectedServerId.value) return
  isNginxInstalling.value = true
  installLogs.value = ["开始安装 Nginx..."]

  createStreamRequest(
    {
      url: "deploy/nginx/install",
      method: "POST",
      data: { server_id: selectedServerId.value }
    },
    (data: any, isComplete?: boolean) => {
      if (data?.message) {
        installLogs.value.push(data.message)
        nextTick(() => {
          if (installLogRef.value) {
            installLogRef.value.scrollTop = installLogRef.value.scrollHeight
          }
        })
      }
      if (isComplete) {
        isNginxInstalling.value = false
        loadCacheStatus()
      }
    },
    (error: any) => {
      isNginxInstalling.value = false
      installLogs.value.push(`错误: ${error?.message || error}`)
      ElMessage.error("安装失败")
    }
  )
}

// ========== 生命周期 ==========
onMounted(() => {
  loadServerList()
  loadCloudAccounts()
  // 加载商户下拉
  getMerchantList({ page: 1, size: 2000 }).then((res) => {
    merchantList.value = Array.isArray(res.data?.list) ? res.data.list : []
  })
})
</script>

<template>
  <div class="app-container">
    <!-- 服务器选择 + 视图切换 -->
    <el-card class="server-select-card" shadow="never">
      <div class="flex items-center gap-4">
        <span class="font-bold w-24">系统服务器:</span>
        <el-select v-model="selectedServerId" placeholder="请选择服务器" style="width: 450px" filterable>
          <el-option
            v-for="server in serverList"
            :key="server.id"
            :label="`${server.name} (${server.host})`"
            :value="server.id"
          >
            <div class="flex items-center justify-between">
              <span>{{ server.name }}</span>
              <span class="text-sm text-gray-400">{{ server.host }}</span>
            </div>
          </el-option>
        </el-select>
        <el-tag type="info">共 {{ serverList.length }} 台</el-tag>
        <div class="flex items-center gap-2 ml-6">
          <span class="font-bold">商户:</span>
          <el-select v-model="selectedMerchantId" placeholder="选择商户(可选)" style="width: 260px" filterable clearable @change="loadRecords">
            <el-option v-for="m in merchantList" :key="m.id" :label="`${m.name}${(m as any).port ? ` (端口:${(m as any).port})` : ''}`" :value="m.id" />
          </el-select>
        </div>

        <el-button type="primary" icon="Refresh" @click="loadRecords">刷新列表</el-button>
        <el-divider direction="vertical" />
        <el-button type="info" icon="Connection" @click="openForwardDialog">配置转发</el-button>
        <el-button type="info" icon="View" @click="openForwardStatusDialog">转发状态</el-button>
        <el-button type="danger" icon="Delete" @click="doClearForward">清除转发</el-button>
        <el-divider direction="vertical" />
        <el-button type="warning" icon="Document" @click="openConfigDialog">配置文件</el-button>
        <el-divider direction="vertical" />
        <el-button type="info" icon="Box" @click="openCacheDialog">缓存管理</el-button>
        <el-divider direction="vertical" />
        <el-button type="success" icon="Plus" @click="openDeployDialog">一键部署 GOST 服务器</el-button>
      </div>
    </el-card>

    <!-- 列表 -->
    <el-card class="service-list-card" shadow="never">
      <vxe-grid ref="xGridDom" v-bind="xGridOpt">
        <template #pager>
          <vxe-pager
            v-model:current-page="pagination.currentPage"
            v-model:page-size="pagination.pageSize"
            :total="pagination.total"
            :page-sizes="[10, 20, 50, 100]"
            @page-change="handlePageChange"
          />
        </template>
        <!-- 工具栏按钮 -->
        <template #toolbar-btns>
          <vxe-button icon="vxe-icon-refresh" @click="loadRecords">刷新列表</vxe-button>
          <vxe-button status="primary" icon="vxe-icon-add" @click="onCreate">新增 Service</vxe-button>
        </template>

        <!-- 商户列 -->
        <template #service-merchant="{ row }">
          <template v-if="getMerchantByServiceName(row.name)">
            <div class="merchant-info">
              <span class="merchant-name">{{ getMerchantByServiceName(row.name)?.name }}</span>
              <span class="merchant-port text-xs text-gray-400">端口: {{ (getMerchantByServiceName(row.name) as any)?.port }}</span>
            </div>
          </template>
          <span v-else class="text-gray-400">-</span>
        </template>
        <!-- 类型列 -->
        <template #service-types="{ row }">
          <div class="flex items-center gap-1">
            <el-tag size="small" type="info" v-if="row.handler?.type">H: {{ row.handler.type }}</el-tag>
            <el-tag size="small" type="info" v-if="row.listener?.type">L: {{ row.listener.type }}</el-tag>
          </div>
        </template>
        <!-- Chain 列 -->
        <template #service-chain="{ row }">
          <div class="flex flex-col">
            <div class="truncate" :title="row.handler?.chain || '-'">
              {{ row.handler?.chain || '-' }}
            </div>
            <div class="text-xs text-gray-500 truncate" :title="(chainMap[row.handler?.chain]?.hops?.[0]?.nodes?.map((n: any) => n.addr).join(', ')) || '-'">
              {{ (chainMap[row.handler?.chain]?.hops?.[0]?.nodes?.map((n: any) => n.addr).join(', ')) || '-' }}
            </div>
          </div>
        </template>
        <!-- 操作列 -->
        <template #row-operate="{ row }">
          <el-button v-permission="['admin']" link type="primary" size="small" @click="crudStore.onShowModal(row)">编辑</el-button>
          <el-button v-permission="['admin']" link type="danger" size="small" @click="onDelete(row)">删除</el-button>
        </template>
      </vxe-grid>
    </el-card>

    <!-- 编辑弹窗（JSON） -->
    <vxe-modal ref="xModalDom" v-bind="xModalOpt">
      <vxe-form ref="xFormDom" v-bind="xFormOpt" />
    </vxe-modal>

    <!-- 新增弹窗 -->
    <vxe-modal ref="xCreateModalDom" v-bind="xCreateModalOpt">
      <vxe-form ref="xCreateFormDom" v-bind="xCreateFormOpt" />
    </vxe-modal>

    <!-- 一键部署 GOST 服务器弹窗 -->
    <el-dialog v-model="deployDialogVisible" title="一键部署 GOST 服务器" width="700px" :close-on-click-modal="false">
      <el-form :model="deployForm" label-width="120px">
        <el-form-item label="云账号" required>
          <el-select v-model="deployForm.cloud_account_id" placeholder="选择云账号" style="width: 100%">
            <el-option v-for="acc in cloudAccountList" :key="acc.id" :label="`${acc.name} (${acc.cloud_type})`" :value="acc.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="地区" required>
          <el-select v-model="deployForm.region_id" placeholder="选择地区" style="width: 100%">
            <el-option v-for="r in regionOptions" :key="r.value" :label="r.label" :value="r.value" />
          </el-select>
        </el-form-item>
        <el-form-item label="服务器名称">
          <el-input v-model="deployForm.server_name" placeholder="自动生成" />
        </el-form-item>
        <el-form-item label="带宽(Mbps)">
          <el-input v-model="deployForm.bandwidth" placeholder="默认 5Mbps" />
        </el-form-item>
        <el-form-item label="SSH密码">
          <el-input v-model="deployForm.password" type="password" placeholder="留空则自动生成密钥" show-password />
        </el-form-item>
      </el-form>

      <!-- 部署日志 -->
      <div v-if="deployLogs.length > 0" class="deploy-log-container">
        <div class="deploy-log-title">部署日志:</div>
        <div ref="deployLogRef" class="deploy-log-content">
          <div v-for="(log, idx) in deployLogs" :key="idx" class="deploy-log-line">{{ log }}</div>
        </div>
      </div>

      <template #footer>
        <el-button @click="deployDialogVisible = false" :disabled="isDeploying">取消</el-button>
        <el-button type="primary" @click="startDeploy" :loading="isDeploying">
          {{ isDeploying ? '部署中...' : '开始部署' }}
        </el-button>
      </template>
    </el-dialog>

    <!-- 配置转发弹窗 -->
    <el-dialog v-model="forwardDialogVisible" title="配置 GOST 转发" width="540px" :close-on-click-modal="false">
      <el-form :model="forwardForm" label-width="120px">
        <!-- 当前服务器信息 -->
        <el-descriptions :column="1" border size="small" style="margin-bottom: 16px">
          <el-descriptions-item label="系统服务器">
            <el-tag type="primary">{{ selectedServer?.name }}</el-tag>
            <span class="ml-2 text-gray-500">{{ selectedServer?.host }}</span>
          </el-descriptions-item>
        </el-descriptions>
        <el-alert type="info" :closable="false" style="margin-bottom: 16px">
          配置后，此服务器的 GOST 将监听指定端口并转发到目标商户服务器（对称转发）。<br/>
          默认端口: 10010(TCP), 10011(WS), 10012(HTTP)
        </el-alert>
        <el-form-item label="目标商户IP" required>
          <el-input v-model="forwardForm.target_ip" placeholder="商户服务器IP，例如: 1.2.3.4" />
        </el-form-item>
        <el-form-item label="连接模式" required>
          <el-radio-group v-model="forwardForm.mode">
            <el-radio value="tls">
              <span>TLS 加密</span>
              <span class="text-xs text-gray-400 ml-1">(推荐，防运营商屏蔽)</span>
            </el-radio>
            <el-radio value="tcp">
              <span>TCP 直连</span>
              <span class="text-xs text-gray-400 ml-1">(更快，但可能被屏蔽)</span>
            </el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="自定义端口">
          <el-input v-model="forwardForm.portsStr" placeholder="留空使用默认端口(10010,10011,10012)，多个端口用逗号分隔" />
          <div class="text-xs text-gray-400 mt-1">例如: 10010,10011,10012 或 20000,20001,20002</div>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="forwardDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="doSetupForward" :loading="isSettingForward">确认配置</el-button>
      </template>
    </el-dialog>

    <!-- GOST 配置文件弹窗 -->
    <el-dialog v-model="configDialogVisible" title="GOST 配置文件" width="800px" :close-on-click-modal="false">
      <div v-if="configPath" class="text-sm text-gray-400 mb-2">路径: {{ configPath }}</div>
      <el-input
        v-model="configContent"
        type="textarea"
        :rows="22"
        :loading="isConfigLoading"
        placeholder="加载中..."
        style="font-family: 'Consolas', 'Monaco', monospace; font-size: 13px;"
      />
      <template #footer>
        <el-button @click="configDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="saveConfig" :loading="isConfigSaving">保存</el-button>
      </template>
    </el-dialog>

    <!-- 转发状态弹窗 -->
    <el-dialog v-model="forwardStatusDialogVisible" title="转发状态" width="600px">
      <template v-if="forwardStatus">
        <el-descriptions :column="2" border size="small">
          <el-descriptions-item label="服务器名称">{{ forwardStatus.server_name }}</el-descriptions-item>
          <el-descriptions-item label="服务器IP">{{ forwardStatus.server_ip }}</el-descriptions-item>
          <el-descriptions-item label="转发规则数">{{ forwardStatus.total_count }}</el-descriptions-item>
        </el-descriptions>
        <el-table :data="forwardStatus.forwards" style="margin-top: 16px" max-height="300">
          <el-table-column prop="port" label="监听端口" width="120" />
          <el-table-column prop="target_ip" label="转发目标" />
          <el-table-column prop="status" label="状态" width="100">
            <template #default="{ row }">
              <el-tag :type="row.status === 'active' ? 'success' : 'danger'" size="small">
                {{ row.status === 'active' ? '活跃' : '异常' }}
              </el-tag>
            </template>
          </el-table-column>
        </el-table>
        <el-empty v-if="forwardStatus.forwards?.length === 0" description="暂无转发规则" />
      </template>
      <template #footer>
        <el-button @click="forwardStatusDialogVisible = false">关闭</el-button>
      </template>
    </el-dialog>

    <!-- Nginx 缓存管理弹窗 -->
    <el-dialog v-model="cacheDialogVisible" title="Nginx 缓存管理" width="600px" :close-on-click-modal="false">
      <div v-loading="isCacheLoading">
        <!-- 状态信息 -->
        <template v-if="cacheStatus">
          <el-descriptions :column="2" border size="small">
            <el-descriptions-item label="Nginx 状态">
              <el-tag v-if="!cacheStatus.installed" type="info" size="small">未安装</el-tag>
              <el-tag v-else-if="cacheStatus.running" type="success" size="small">运行中</el-tag>
              <el-tag v-else type="danger" size="small">已停止</el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="缓存大小">
              {{ cacheStatus.installed ? cacheStatus.cache_size : '-' }}
            </el-descriptions-item>
          </el-descriptions>

          <!-- 未安装时显示安装按钮 -->
          <div v-if="!cacheStatus.installed" style="margin-top: 16px">
            <el-alert type="warning" :closable="false" style="margin-bottom: 12px">
              Nginx 未安装，HTTP 文件请求不会被缓存。安装后，配置转发时会自动启用缓存。
            </el-alert>
            <el-button type="primary" @click="startInstallNginx" :loading="isNginxInstalling">
              {{ isNginxInstalling ? '安装中...' : '安装 Nginx' }}
            </el-button>
          </div>

          <!-- 已安装时显示操作按钮 -->
          <div v-else style="margin-top: 16px">
            <el-alert type="info" :closable="false" style="margin-bottom: 12px">
              Nginx 会自动缓存通过 HTTP 端口传输的图片、视频、音频等媒体文件（7天有效期）。API 请求不受影响。
            </el-alert>
            <el-button type="danger" @click="doClearCache" :loading="isCacheClearing">
              {{ isCacheClearing ? '清除中...' : '清除所有缓存' }}
            </el-button>
            <el-button @click="loadCacheStatus">刷新状态</el-button>
          </div>
        </template>

        <!-- 安装日志 -->
        <div v-if="installLogs.length > 0" class="deploy-log-container" style="margin-top: 16px">
          <div class="deploy-log-title">安装日志:</div>
          <div ref="installLogRef" class="deploy-log-content">
            <div v-for="(log, idx) in installLogs" :key="idx" class="deploy-log-line">{{ log }}</div>
          </div>
        </div>
      </div>

      <template #footer>
        <el-button @click="cacheDialogVisible = false">关闭</el-button>
      </template>
    </el-dialog>

  </div>
</template>

<style lang="scss" scoped>
.app-container {
  padding: 20px;
}

.server-select-card {
  margin-bottom: 20px;
}

.service-list-card {
  :deep(.el-card__body) {
    padding: 0;
  }
}

.deploy-log-container {
  margin-top: 16px;
  border: 1px solid #e4e7ed;
  border-radius: 4px;
  background: #1e1e1e;
}

.deploy-log-title {
  padding: 8px 12px;
  background: #f5f7fa;
  border-bottom: 1px solid #e4e7ed;
  font-weight: bold;
  color: #303133;
}

.deploy-log-content {
  max-height: 300px;
  overflow-y: auto;
  padding: 12px;
  font-family: 'Consolas', 'Monaco', monospace;
  font-size: 13px;
  line-height: 1.6;
  color: #4fc3f7;
}

.deploy-log-line {
  white-space: pre-wrap;
  word-break: break-all;
}

.merchant-info {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.merchant-name {
  font-weight: 500;
  color: #303133;
}

.merchant-port {
  color: #909399;
}
</style>
