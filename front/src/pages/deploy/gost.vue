<script lang="ts" setup>
import type { ServerResp, GostConfigSyncStatusResp, TlsCertificateResp, TlsStatusResp } from "@@/apis/deploy/type"
import type { MerchantResp } from "@@/apis/merchant/type"
import type { VxeFormInstance, VxeFormProps, VxeGridInstance, VxeGridProps, VxeModalInstance, VxeModalProps } from "vxe-table"
import { createGostServiceByAPI, deleteGostServiceByAPI, getGostServiceDetail, getServerList, listGostChains, listGostServices, updateGostServiceDetail, setupGostForward, clearGostForward, getGostForwardStatus, getProgramConfig, updateProgramConfig, getNginxCacheStatus, clearNginxCache, persistGostConfig, getGostConfigSyncStatus, setupGostDeploy, diagnoseGost, generateTlsCerts, getTlsCerts, disableTlsCerts, getTlsCertFingerprint, getTlsStatus, verifyTlsStatus, batchUpgradeTls, batchRollbackTls } from "@@/apis/deploy"
import type { GostCheckResult } from "@@/apis/monitor/type"
import { checkGostServers, runBandwidthTest, getServerBandwidth, modifyServerBandwidth } from "@@/apis/monitor"
import type { BandwidthTestResult, BandwidthInfoResp } from "@@/apis/monitor/type"
import { getMerchantList, listMerchantGostServers } from "@@/apis/merchant"
import type { MerchantGostServerResp } from "@@/apis/merchant/type"
import { createStreamRequest } from "@/http/axios"

defineOptions({
  name: "GostService"
})

// ========== GOST 一键部署 ==========
const setupDialogVisible = ref(false)
const setupLogs = ref<string[]>([])
const isSettingUp = ref(false)
const setupLogRef = ref<HTMLDivElement>()

// 部署表单
const setupForm = reactive({
  server_id: 0,
  merchant_ids: [] as number[],
  forward_type: 1
})

// 打开部署弹窗
function openSetupDialog() {
  setupLogs.value = []
  setupForm.server_id = 0
  setupForm.merchant_ids = []
  setupForm.forward_type = 1
  setupDialogVisible.value = true
}

// 执行一键部署
function startSetup() {
  if (!setupForm.server_id) {
    ElMessage.warning("请选择服务器")
    return
  }
  if (setupForm.merchant_ids.length === 0) {
    ElMessage.warning("请选择至少一个商户")
    return
  }

  isSettingUp.value = true
  setupLogs.value = ["开始部署..."]

  setupGostDeploy(
    setupForm,
    (data: any, isComplete?: boolean) => {
      if (data?.message) {
        setupLogs.value.push(data.message)
        scrollToBottom()
      }
      if (isComplete) {
        isSettingUp.value = false
        if (data?.success) {
          ElMessage.success("部署完成!")
          loadRecords()
          loadSyncStatus()
        }
      }
    },
    (error: any) => {
      isSettingUp.value = false
      setupLogs.value.push(`错误: ${error?.message || error}`)
      ElMessage.error("部署失败")
    }
  )
}

// 全选/取消全选商户
function toggleAllMerchants(checked: boolean) {
  if (checked) {
    setupForm.merchant_ids = merchantList.value.map(m => m.id)
  } else {
    setupForm.merchant_ids = []
  }
}

// 滚动到日志底部
function scrollToBottom() {
  nextTick(() => {
    if (setupLogRef.value) {
      setupLogRef.value.scrollTop = setupLogRef.value.scrollHeight
    }
  })
}

// ========== 服务器选择（仅系统服务器） ==========
const allServerList = ref<ServerResp[]>([])
const serverList = ref<ServerResp[]>([])
const selectedServerId = ref<number>(0)
const merchantGostServers = ref<MerchantGostServerResp[]>([])

// 获取所有系统服务器（用于一键部署弹窗等）
async function loadServerList() {
  try {
    const res = await getServerList({ page: 1, size: 5000 })
    const list = res.data?.list ?? []
    allServerList.value = Array.isArray(list) ? list : []
  } catch (error) {
    console.error("获取服务器列表失败:", error)
  }
}

// 选择商户后，加载该商户关联的 GOST 服务器
async function onMerchantChange() {
  // 重置服务器选择
  selectedServerId.value = 0
  serverList.value = []
  merchantGostServers.value = []
  records.value = []
  gostUnreachable.value = false

  // 重置 TLS 数据
  certs.value = []
  tlsFingerprint.value = ""
  fingerprintExpires.value = ""
  tlsStatus.value = null

  if (!selectedMerchantId.value) return

  try {
    const res = await listMerchantGostServers(selectedMerchantId.value)
    merchantGostServers.value = Array.isArray(res.data) ? res.data : []

    // 构建服务器下拉列表（从关联表中提取）
    serverList.value = merchantGostServers.value.map(mg => ({
      id: mg.server_id,
      name: mg.server_name,
      host: mg.server_host,
      server_type: 2,
    } as ServerResp))

    // 只有一台服务器时自动选中
    if (serverList.value.length === 1) {
      selectedServerId.value = serverList.value[0].id
    } else if (serverList.value.length > 1) {
      // 优先选主服务器
      const primary = merchantGostServers.value.find(mg => mg.is_primary === 1)
      if (primary) {
        selectedServerId.value = primary.server_id
      } else {
        selectedServerId.value = serverList.value[0].id
      }
    }
  } catch (error) {
    console.error("获取商户 GOST 服务器失败:", error)
  }

  // 加载当前 tab 的数据
  if (selectedServerId.value) {
    loadRecords()
    loadSyncStatus()
  }
  if (activeTab.value === "tls") {
    loadTlsData()
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
  // 服务名称格式: tcp-relay-{port}, ws-relay-{port+1}, http-relay-{port+2}, minio-relay-{port+3}, wss-relay-443
  const match = serviceName.match(/^(?:tcp|ws|http|minio)-relay-(\d+)$/)
  if (!match) return undefined
  const port = parseInt(match[1], 10)
  // tcp 用 basePort 直接匹配，ws/http/minio 需要反算 basePort
  const offsets: Record<string, number> = { "tcp": 0, "ws": 1, "http": 2, "minio": 3 }
  const prefix = serviceName.split("-relay-")[0]
  const basePort = port - (offsets[prefix] ?? 0)
  return merchantList.value.find(m => (m as any).port === basePort)
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
      loadSyncStatus()
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
    // 若选择了商户，过滤出该商户相关的所有转发规则（base port 段 + wss-443）
    if (selectedMerchantId.value) {
      const m = merchantList.value.find(m => m.id === selectedMerchantId.value)
      if (m && (m as any).port) {
        const basePort = (m as any).port as number
        const relatedNames = new Set([
          `tcp-relay-${basePort}`,
          `ws-relay-${basePort + 1}`,
          `http-relay-${basePort + 2}`,
          `minio-relay-${basePort + 3}`,
          `wss-relay-443`
        ])
        records.value = records.value.filter((s: any) => relatedNames.has(s.name))
      }
    }
    pagination.total = svcRes.data.count || 0
    if (xGridOpt.pagerConfig) {
      ;(xGridOpt.pagerConfig as any).total = pagination.total
      ;(xGridOpt.pagerConfig as any).currentPage = pagination.currentPage
      ;(xGridOpt.pagerConfig as any).pageSize = pagination.pageSize
    }
    gostUnreachable.value = false
  } catch (error: any) {
    console.error("查询列表失败:", error)
    // 检查是否是连接被拒绝的错误
    const errMsg = error?.message || String(error)
    if (errMsg.includes("connection refused") || errMsg.includes("connect:")) {
      gostUnreachable.value = true
      ElMessage.error("GOST 服务未运行，请点击「诊断修复」按钮")
    } else if (errMsg.includes("timeout")) {
      gostUnreachable.value = true
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
        loadSyncStatus()
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
    loadSyncStatus()
  } catch {
    // ignore
  }
}
// 监听服务器/视图变化
watch(selectedServerId, () => {
  if (selectedServerId.value) {
    loadRecords()
    loadSyncStatus()
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
    loadSyncStatus()
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
    loadSyncStatus()
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

// ========== GOST 配置同步状态 ==========
const configSyncStatus = ref<GostConfigSyncStatusResp | null>(null)
const isSyncLoading = ref(false)
const isPersisting = ref(false)

async function loadSyncStatus() {
  if (!selectedServerId.value) return
  isSyncLoading.value = true
  try {
    const res = await getGostConfigSyncStatus(selectedServerId.value)
    configSyncStatus.value = res.data
  } catch {
    configSyncStatus.value = null
  } finally {
    isSyncLoading.value = false
  }
}

async function doPersistConfig() {
  if (!selectedServerId.value) return
  isPersisting.value = true
  try {
    await persistGostConfig({ server_id: selectedServerId.value })
    ElMessage.success("配置已保存到文件")
    await loadSyncStatus()
  } catch (e: any) {
    ElMessage.error(e?.message || "保存失败")
  } finally {
    isPersisting.value = false
  }
}

// ========== 健康检查 ==========
const healthCheckDialogVisible = ref(false)
const isChecking = ref(false)
const healthCheckResults = ref<GostCheckResult[]>([])

async function doHealthCheck() {
  isChecking.value = true
  healthCheckDialogVisible.value = true
  healthCheckResults.value = []
  try {
    const res = await checkGostServers()
    healthCheckResults.value = (res as any).data?.list || res?.list || []
    const downCount = healthCheckResults.value.filter(r => r.status === "down").length
    if (downCount > 0) {
      ElMessage.warning(`检查完成，${downCount} 台服务器异常`)
    } else {
      ElMessage.success("所有服务器正常")
    }
  } catch (e: any) {
    ElMessage.error(e?.message || "健康检查失败")
  } finally {
    isChecking.value = false
  }
}

function formatBytes(bytes: number): string {
  if (bytes === 0) return "0 B"
  const k = 1024
  const sizes = ["B", "KB", "MB", "GB", "TB"]
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return `${(bytes / k ** i).toFixed(1)} ${sizes[i]}`
}

function getStatusType(status: string): "success" | "danger" | "warning" | "info" {
  if (status === "up") return "success"
  if (status === "down") return "danger"
  if (status === "degraded") return "warning"
  return "info"
}

function getStatusLabel(status: string): string {
  if (status === "up") return "正常"
  if (status === "down") return "不可达"
  if (status === "degraded") return "异常"
  return "未知"
}

// ========== GOST 诊断修复 ==========
const diagnoseDialogVisible = ref(false)
const isDiagnosing = ref(false)
const diagnoseLogs = ref<string[]>([])
const diagnoseLogRef = ref<HTMLDivElement>()
const gostUnreachable = ref(false)

function startDiagnose() {
  if (!selectedServerId.value) {
    ElMessage.warning("请先选择服务器")
    return
  }
  isDiagnosing.value = true
  diagnoseDialogVisible.value = true
  diagnoseLogs.value = ["开始诊断..."]

  diagnoseGost(
    { server_id: selectedServerId.value },
    (data: any, isComplete?: boolean) => {
      if (data?.message) {
        diagnoseLogs.value.push(data.message)
        nextTick(() => {
          if (diagnoseLogRef.value) {
            diagnoseLogRef.value.scrollTop = diagnoseLogRef.value.scrollHeight
          }
        })
      }
      if (isComplete) {
        isDiagnosing.value = false
        if (data?.success) {
          ElMessage.success("诊断修复完成!")
          gostUnreachable.value = false
          loadRecords()
          loadSyncStatus()
        }
      }
    },
    (error: any) => {
      isDiagnosing.value = false
      diagnoseLogs.value.push(`错误: ${error?.message || error}`)
      ElMessage.error("诊断修复失败")
    }
  )
}

// ========== 带宽测速 ==========
const speedTestDialogVisible = ref(false)
const isSpeedTesting = ref(false)
const speedTestLogs = ref<string[]>([])
const speedTestLogRef = ref<HTMLDivElement>()
const speedTestResult = ref<BandwidthTestResult | null>(null)

function startSpeedTest() {
  if (!selectedServerId.value) {
    ElMessage.warning("请先选择服务器")
    return
  }
  isSpeedTesting.value = true
  speedTestDialogVisible.value = true
  speedTestLogs.value = ["开始测速..."]
  speedTestResult.value = null

  runBandwidthTest(
    { server_id: selectedServerId.value },
    (data: any, isComplete?: boolean) => {
      if (data?.message) {
        speedTestLogs.value.push(data.message)
        nextTick(() => {
          if (speedTestLogRef.value) {
            speedTestLogRef.value.scrollTop = speedTestLogRef.value.scrollHeight
          }
        })
      }
      if (data?.result) {
        speedTestResult.value = data.result
      }
      if (isComplete) {
        isSpeedTesting.value = false
        if (data?.success) {
          ElMessage.success("测速完成!")
        }
      }
    },
    (error: any) => {
      isSpeedTesting.value = false
      speedTestLogs.value.push(`错误: ${error?.message || error}`)
      ElMessage.error("测速失败")
    }
  )
}

// ========== 带宽管理 ==========
const bandwidthDialogVisible = ref(false)
const isBandwidthLoading = ref(false)
const isBandwidthSaving = ref(false)
const bandwidthInfo = ref<BandwidthInfoResp | null>(null)
const newBandwidthOut = ref(1)

async function openBandwidthDialog() {
  if (!selectedServerId.value) {
    ElMessage.warning("请先选择服务器")
    return
  }
  // 检查服务器是否绑定了云账号
  const server = allServerList.value.find(s => s.id === selectedServerId.value)
  if (!server?.cloud_account_id || !server?.cloud_instance_id) {
    ElMessage.warning("该服务器未绑定云账号/实例，请先在服务器编辑页面绑定 cloud_account_id / cloud_instance_id / cloud_region_id")
    return
  }

  bandwidthDialogVisible.value = true
  isBandwidthLoading.value = true
  bandwidthInfo.value = null
  try {
    const res = await getServerBandwidth({ server_id: selectedServerId.value })
    bandwidthInfo.value = res.data
    newBandwidthOut.value = res.data.internet_max_bandwidth_out
  } catch (e: any) {
    ElMessage.error(e?.message || "查询带宽失败")
  } finally {
    isBandwidthLoading.value = false
  }
}

async function saveBandwidth() {
  if (!selectedServerId.value || !newBandwidthOut.value) return
  isBandwidthSaving.value = true
  try {
    await modifyServerBandwidth({
      server_id: selectedServerId.value,
      internet_max_bandwidth_out: newBandwidthOut.value
    })
    ElMessage.success(`带宽已修改为 ${newBandwidthOut.value} Mbps`)
    // 刷新显示
    const res = await getServerBandwidth({ server_id: selectedServerId.value })
    bandwidthInfo.value = res.data
  } catch (e: any) {
    ElMessage.error(e?.message || "修改带宽失败")
  } finally {
    isBandwidthSaving.value = false
  }
}

// ========== Tab 切换 ==========
const activeTab = ref("tunnel")

// ========== TLS 证书管理 ==========
const tlsLoading = ref(false)
const certs = ref<TlsCertificateResp[]>([])
const tlsFingerprint = ref("")
const fingerprintExpires = ref("")
const tlsStatus = ref<TlsStatusResp | null>(null)

const selectedMerchantName = computed(() => {
  const m = merchantList.value.find(m => m.id === selectedMerchantId.value)
  return m?.name || ""
})

async function loadTlsData() {
  if (!selectedMerchantId.value) return
  await Promise.all([loadCerts(), loadFingerprint(), loadTlsStatus()])
}

async function loadCerts() {
  if (!selectedMerchantId.value) return
  tlsLoading.value = true
  try {
    const res = await getTlsCerts(selectedMerchantId.value)
    const { ca, server } = res.data || {}
    certs.value = [ca, server].filter(Boolean)
  } catch {
    certs.value = []
  } finally {
    tlsLoading.value = false
  }
}

async function loadFingerprint() {
  if (!selectedMerchantId.value) return
  try {
    const res = await getTlsCertFingerprint(selectedMerchantId.value)
    tlsFingerprint.value = res.data.fingerprint || ""
    fingerprintExpires.value = res.data.expires_at || ""
  } catch {
    tlsFingerprint.value = ""
  }
}

async function loadTlsStatus() {
  if (!selectedMerchantId.value) return
  try {
    const res = await getTlsStatus(selectedMerchantId.value)
    tlsStatus.value = res.data
  } catch {
    tlsStatus.value = null
  }
}

async function handleTlsGenerate() {
  if (!selectedMerchantId.value) return
  ElMessageBox.confirm(
    `将为商户「${selectedMerchantName.value}」生成新的 CA 根证书和服务器证书。已部署的服务器需要重新升级 TLS。确定继续？`,
    "生成证书",
    { type: "warning" }
  ).then(async () => {
    tlsLoading.value = true
    try {
      await generateTlsCerts({ merchant_id: selectedMerchantId.value! })
      ElMessage.success("证书生成成功")
      await loadCerts()
      await loadFingerprint()
    } catch (e: any) {
      ElMessage.error(e.message || "证书生成失败")
    } finally {
      tlsLoading.value = false
    }
  })
}

async function handleTlsDisable() {
  if (!selectedMerchantId.value) return
  ElMessageBox.confirm(
    `停用商户「${selectedMerchantName.value}」的证书后，新部署的服务器将不会自动启用 TLS。确定停用？`,
    "停用证书",
    { type: "warning" }
  ).then(async () => {
    tlsLoading.value = true
    try {
      await disableTlsCerts({ merchant_id: selectedMerchantId.value! })
      ElMessage.success("证书已停用")
      await loadCerts()
      tlsFingerprint.value = ""
    } catch (e: any) {
      ElMessage.error(e.message || "停用失败")
    } finally {
      tlsLoading.value = false
    }
  })
}

function copyFingerprint() {
  if (!tlsFingerprint.value) return
  navigator.clipboard.writeText(tlsFingerprint.value).then(() => {
    ElMessage.success("指纹已复制到剪贴板")
  })
}

async function handleTlsUpgrade() {
  if (!selectedMerchantId.value) return
  ElMessageBox.confirm(
    `将商户「${selectedMerchantName.value}」的所有 GOST 服务器升级为 TLS 模式？`,
    "批量升级",
    { type: "warning" }
  ).then(async () => {
    tlsLoading.value = true
    try {
      const res = await batchUpgradeTls({ merchant_id: selectedMerchantId.value! })
      const { success, failed, total } = res.data
      if (failed === 0) {
        ElMessage.success(`升级完成，全部成功 (${success}/${total})`)
      } else {
        ElMessage.warning(`升级完成，成功 ${success}，失败 ${failed}`)
      }
      await loadTlsStatus()
    } catch (e: any) {
      ElMessage.error(e.message || "升级失败")
    } finally {
      tlsLoading.value = false
    }
  })
}

async function handleTlsRollback() {
  if (!selectedMerchantId.value) return
  ElMessageBox.confirm(
    `将商户「${selectedMerchantName.value}」的所有 GOST 服务器回滚为 TCP 模式？`,
    "批量回滚",
    { type: "warning" }
  ).then(async () => {
    tlsLoading.value = true
    try {
      const res = await batchRollbackTls({ merchant_id: selectedMerchantId.value! })
      const { success, failed, total } = res.data
      if (failed === 0) {
        ElMessage.success(`回滚完成，全部成功 (${success}/${total})`)
      } else {
        ElMessage.warning(`回滚完成，成功 ${success}，失败 ${failed}`)
      }
      await loadTlsStatus()
    } catch (e: any) {
      ElMessage.error(e.message || "回滚失败")
    } finally {
      tlsLoading.value = false
    }
  })
}

async function handleTlsVerify() {
  if (!selectedMerchantId.value) return
  tlsLoading.value = true
  try {
    const res = await verifyTlsStatus({ merchant_id: selectedMerchantId.value! })
    tlsStatus.value = res.data
    ElMessage.success("验证完成")
  } catch (e: any) {
    ElMessage.error(e.message || "验证失败")
  } finally {
    tlsLoading.value = false
  }
}

// Tab 切换时加载数据
watch(activeTab, (tab) => {
  if (tab === "tls" && selectedMerchantId.value && !certs.value.length) {
    loadTlsData()
  }
})

// ========== 生命周期 ==========
onMounted(() => {
  // 加载所有服务器（用于一键部署弹窗等场景）
  loadServerList()
  // 加载商户下拉
  getMerchantList({ page: 1, size: 2000 }).then((res) => {
    merchantList.value = Array.isArray(res.data?.list) ? res.data.list : []
    // 只有一个商户时自动选中
    if (merchantList.value.length === 1) {
      selectedMerchantId.value = merchantList.value[0].id
      onMerchantChange()
    }
  })
})
</script>

<template>
  <div class="app-container">
    <!-- 顶部：商户 → 服务器 → 操作 -->
    <el-card class="server-select-card" shadow="never">
      <div class="flex items-center gap-4 flex-wrap">
        <span class="font-bold">商户:</span>
        <el-select v-model="selectedMerchantId" placeholder="请选择商户" style="width: 280px" filterable @change="onMerchantChange">
          <el-option v-for="m in merchantList" :key="m.id" :label="`${m.name} (${m.no})`" :value="m.id" />
        </el-select>

        <el-divider direction="vertical" />
        <span class="font-bold">GOST 服务器:</span>
        <el-select v-model="selectedServerId" placeholder="请先选择商户" style="width: 360px" filterable :disabled="!selectedMerchantId">
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
        <el-tag v-if="serverList.length > 0" type="info">{{ serverList.length }} 台</el-tag>

        <el-divider direction="vertical" />
        <el-button type="primary" icon="Refresh" @click="loadRecords" :disabled="!selectedServerId">刷新</el-button>
        <el-button type="success" icon="Plus" @click="openSetupDialog">一键部署</el-button>
        <el-button type="primary" icon="Monitor" :loading="isChecking" @click="doHealthCheck" :disabled="!selectedServerId">健康检查</el-button>
        <el-button type="warning" icon="SetUp" :loading="isDiagnosing" @click="startDiagnose" :disabled="!selectedServerId">诊断修复</el-button>
        <el-button type="info" icon="Odometer" :loading="isSpeedTesting" @click="startSpeedTest" :disabled="!selectedServerId">测速</el-button>
        <el-button type="info" icon="Connection" @click="openBandwidthDialog" :disabled="!selectedServerId">带宽管理</el-button>
      </div>
    </el-card>

    <template v-if="selectedMerchantId">
    <!-- GOST 不可达提示 -->
    <el-alert
      v-if="gostUnreachable && selectedServerId"
      type="error"
      show-icon
      :closable="false"
      style="margin-bottom: 16px"
    >
      <template #title>
        <span>GOST 服务不可达 (connection refused)</span>
      </template>
      <template #default>
        <span>当前服务器的 GOST 服务未运行或端口未开放，无法查询转发规则。</span>
        <el-button type="warning" size="small" style="margin-left: 12px" :loading="isDiagnosing" @click="startDiagnose">
          诊断修复
        </el-button>
      </template>
    </el-alert>

    <!-- 无关联服务器提示 -->
    <el-alert
      v-if="serverList.length === 0"
      type="warning"
      show-icon
      :closable="false"
      style="margin-bottom: 16px"
    >
      该商户暂无关联的 GOST 服务器，请先在「一键部署」中为该商户配置服务器。
    </el-alert>

    <!-- Tab 切换 -->
    <el-tabs v-model="activeTab" type="border-card">
      <!-- Tab 1: 隧道服务 -->
      <el-tab-pane label="隧道服务" name="tunnel">
        <!-- 隧道服务工具栏 -->
        <div class="tunnel-toolbar flex items-center gap-2 flex-wrap" style="margin-bottom: 12px">
          <el-button type="info" icon="Connection" @click="openForwardDialog">配置转发</el-button>
          <el-button type="info" icon="View" @click="openForwardStatusDialog">转发状态</el-button>
          <el-button type="danger" icon="Delete" @click="doClearForward">清除转发</el-button>
          <el-divider direction="vertical" />
          <el-button type="warning" icon="Document" @click="openConfigDialog">配置文件</el-button>
          <el-tooltip v-if="configSyncStatus" :content="configSyncStatus.message" placement="bottom">
            <el-tag
              :type="configSyncStatus.synced ? 'success' : 'warning'"
              style="cursor: pointer"
              @click="loadSyncStatus"
            >
              {{ configSyncStatus.synced ? '已同步' : '未同步' }}
              <span v-if="!configSyncStatus.synced" class="text-xs">
                (运行:{{ configSyncStatus.running_service_count }} / 文件:{{ configSyncStatus.file_service_count }})
              </span>
            </el-tag>
          </el-tooltip>
          <el-button
            type="success"
            icon="Download"
            :loading="isPersisting"
            :disabled="configSyncStatus?.synced === true"
            @click="doPersistConfig"
          >保存到文件</el-button>
          <el-divider direction="vertical" />
          <el-button type="info" icon="Box" @click="openCacheDialog">缓存管理</el-button>
        </div>

        <!-- 服务列表 -->
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
          <template #toolbar-btns>
            <vxe-button icon="vxe-icon-refresh" @click="loadRecords">刷新列表</vxe-button>
            <vxe-button status="primary" icon="vxe-icon-add" @click="onCreate">新增 Service</vxe-button>
          </template>
          <template #service-merchant="{ row }">
            <template v-if="getMerchantByServiceName(row.name)">
              <div class="merchant-info">
                <span class="merchant-name">{{ getMerchantByServiceName(row.name)?.name }}</span>
                <span class="merchant-port text-xs text-gray-400">端口: {{ (getMerchantByServiceName(row.name) as any)?.port }}</span>
              </div>
            </template>
            <span v-else class="text-gray-400">-</span>
          </template>
          <template #service-types="{ row }">
            <div class="flex items-center gap-1">
              <el-tag size="small" type="info" v-if="row.handler?.type">H: {{ row.handler.type }}</el-tag>
              <el-tag size="small" type="info" v-if="row.listener?.type">L: {{ row.listener.type }}</el-tag>
            </div>
          </template>
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
          <template #row-operate="{ row }">
            <el-button v-permission="['admin']" link type="primary" size="small" @click="crudStore.onShowModal(row)">编辑</el-button>
            <el-button v-permission="['admin']" link type="danger" size="small" @click="onDelete(row)">删除</el-button>
          </template>
        </vxe-grid>
      </el-tab-pane>

      <!-- Tab 2: TLS 证书 -->
      <el-tab-pane label="TLS 证书" name="tls">
          <!-- 证书管理 -->
          <el-card v-loading="tlsLoading" shadow="never" style="margin-bottom: 0">
            <template #header>
              <div class="card-header">
                <span class="font-bold text-base">TLS 证书 — {{ selectedMerchantName }}</span>
                <div>
                  <el-button type="primary" @click="handleTlsGenerate">生成证书</el-button>
                  <el-button type="danger" :disabled="certs.length === 0" @click="handleTlsDisable">停用证书</el-button>
                </div>
              </div>
            </template>
            <el-table v-if="certs.length > 0" :data="certs" border size="small">
              <el-table-column prop="name" label="名称" width="160" />
              <el-table-column label="类型" width="120">
                <template #default="{ row }">
                  <el-tag :type="row.cert_type === 1 ? 'warning' : 'primary'" size="small">
                    {{ row.cert_type === 1 ? 'CA 根证书' : '服务器证书' }}
                  </el-tag>
                </template>
              </el-table-column>
              <el-table-column label="状态" width="80">
                <template #default="{ row }">
                  <el-tag :type="row.status === 1 ? 'success' : 'info'" size="small">
                    {{ row.status === 1 ? '启用' : '停用' }}
                  </el-tag>
                </template>
              </el-table-column>
              <el-table-column prop="fingerprint" label="指纹 (SHA-256)" show-overflow-tooltip />
              <el-table-column prop="expires_at" label="过期时间" width="160" />
              <el-table-column prop="created_at" label="创建时间" width="160" />
            </el-table>
            <el-empty v-else description="暂未生成证书，请点击「生成证书」" />
          </el-card>

          <!-- 证书指纹（App 端 Pinning） -->
          <el-card v-if="tlsFingerprint" shadow="never" class="mt-4">
            <template #header>
              <span class="font-bold text-base">App 证书指纹 (Certificate Pinning)</span>
            </template>
            <el-descriptions :column="1" border size="small">
              <el-descriptions-item label="SHA-256 指纹">
                <code class="fingerprint-text">{{ tlsFingerprint }}</code>
                <el-button type="primary" link size="small" style="margin-left: 8px" @click="copyFingerprint">复制</el-button>
              </el-descriptions-item>
              <el-descriptions-item label="证书过期时间">{{ fingerprintExpires }}</el-descriptions-item>
            </el-descriptions>
          </el-card>

          <!-- 系统服务器 TLS 状态 -->
          <el-card v-loading="tlsLoading" shadow="never" class="mt-4">
            <template #header>
              <div class="card-header">
                <span class="font-bold text-base">GOST 服务器 TLS 状态 — {{ selectedMerchantName }}</span>
                <div>
                  <el-button @click="handleTlsVerify">验证连接</el-button>
                  <el-button type="warning" @click="handleTlsRollback">批量回滚 TCP</el-button>
                  <el-button type="success" :disabled="certs.length === 0" @click="handleTlsUpgrade">批量升级 TLS</el-button>
                </div>
              </div>
            </template>
            <div v-if="tlsStatus">
              <el-descriptions :column="3" border size="small" class="mb-4">
                <el-descriptions-item label="总数">{{ tlsStatus.total }}</el-descriptions-item>
                <el-descriptions-item label="TLS">
                  <el-tag type="success" size="small">{{ tlsStatus.tls_count }}</el-tag>
                </el-descriptions-item>
                <el-descriptions-item label="TCP">
                  <el-tag type="info" size="small">{{ tlsStatus.tcp_count }}</el-tag>
                </el-descriptions-item>
              </el-descriptions>
              <el-table :data="tlsStatus.servers" border size="small">
                <el-table-column prop="server_name" label="服务器" width="160" />
                <el-table-column prop="host" label="IP" width="140" />
                <el-table-column label="TLS" width="80">
                  <template #default="{ row }">
                    <el-tag :type="row.tls_enabled === 1 ? 'success' : 'info'" size="small">
                      {{ row.tls_enabled === 1 ? 'TLS' : 'TCP' }}
                    </el-tag>
                  </template>
                </el-table-column>
                <el-table-column label="验证" width="120">
                  <template #default="{ row }">
                    <el-tag v-if="row.tls_verified" type="success" size="small">通过</el-tag>
                    <el-tag v-else-if="row.verify_error" type="danger" size="small">失败</el-tag>
                    <span v-else class="text-gray-400">-</span>
                  </template>
                </el-table-column>
                <el-table-column prop="verify_error" label="错误详情" show-overflow-tooltip />
                <el-table-column prop="tls_deployed_at" label="部署时间" width="160" />
              </el-table>
            </div>
            <el-empty v-else description="该商户暂无关联的 GOST 服务器" />
          </el-card>
      </el-tab-pane>
    </el-tabs>
    </template>
    <el-empty v-else description="请先选择商户" style="margin-top: 40px" />

    <!-- 编辑弹窗（JSON） -->
    <vxe-modal ref="xModalDom" v-bind="xModalOpt">
      <vxe-form ref="xFormDom" v-bind="xFormOpt" />
    </vxe-modal>

    <!-- 新增弹窗 -->
    <vxe-modal ref="xCreateModalDom" v-bind="xCreateModalOpt">
      <vxe-form ref="xCreateFormDom" v-bind="xCreateFormOpt" />
    </vxe-modal>

    <!-- 一键部署 GOST 弹窗 -->
    <el-dialog v-model="setupDialogVisible" title="一键部署 GOST" width="700px" :close-on-click-modal="false">
      <el-form :model="setupForm" label-width="120px">
        <el-form-item label="系统服务器" required>
          <el-select v-model="setupForm.server_id" placeholder="选择服务器" style="width: 100%" filterable>
            <el-option
              v-for="s in allServerList.filter(s => s.server_type === 2)"
              :key="s.id"
              :label="`${s.name} (${s.host})`"
              :value="s.id"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="转发类型" required>
          <el-radio-group v-model="setupForm.forward_type">
            <el-radio :value="1">
              <span>加密 (relay+tls)</span>
              <span class="text-xs text-gray-400 ml-1">(推荐)</span>
            </el-radio>
            <el-radio :value="2">
              <span>直连 (tcp)</span>
            </el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="选择商户" required>
          <div style="margin-bottom: 8px">
            <el-checkbox
              :model-value="setupForm.merchant_ids.length === merchantList.length && merchantList.length > 0"
              :indeterminate="setupForm.merchant_ids.length > 0 && setupForm.merchant_ids.length < merchantList.length"
              @change="(val: any) => toggleAllMerchants(!!val)"
            >全选 ({{ setupForm.merchant_ids.length }}/{{ merchantList.length }})</el-checkbox>
          </div>
          <el-checkbox-group v-model="setupForm.merchant_ids">
            <el-checkbox
              v-for="m in merchantList"
              :key="m.id"
              :value="m.id"
              style="display: block; margin-left: 0; margin-bottom: 4px"
            >
              {{ m.name }}
              <span class="text-xs text-gray-400 ml-1">端口: {{ (m as any).port || '-' }} | IP: {{ (m as any).server_ip || '-' }}</span>
            </el-checkbox>
          </el-checkbox-group>
        </el-form-item>
      </el-form>

      <el-alert type="info" :closable="false" style="margin-bottom: 12px">
        系统自动检测 GOST 安装状态，未安装则自动安装。TLS 证书如已生成也会自动部署。
      </el-alert>

      <!-- 部署日志 -->
      <div v-if="setupLogs.length > 0" class="deploy-log-container">
        <div class="deploy-log-title">部署日志:</div>
        <div ref="setupLogRef" class="deploy-log-content">
          <div v-for="(log, idx) in setupLogs" :key="idx" class="deploy-log-line">{{ log }}</div>
        </div>
      </div>

      <template #footer>
        <el-button @click="setupDialogVisible = false" :disabled="isSettingUp">取消</el-button>
        <el-button type="primary" @click="startSetup" :loading="isSettingUp">
          {{ isSettingUp ? '部署中...' : '开始部署' }}
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

    <!-- 健康检查结果弹窗 -->
    <el-dialog v-model="healthCheckDialogVisible" title="GOST 健康检查" width="1100px" :close-on-click-modal="false">
      <div v-loading="isChecking" element-loading-text="正在检查所有 GOST 服务器...">
        <!-- 概览 -->
        <div v-if="healthCheckResults.length > 0" class="health-summary">
          <el-tag type="success" size="large">
            正常: {{ healthCheckResults.filter(r => r.status === 'up').length }}
          </el-tag>
          <el-tag type="danger" size="large" style="margin-left: 8px">
            不可达: {{ healthCheckResults.filter(r => r.status === 'down').length }}
          </el-tag>
          <el-tag type="warning" size="large" style="margin-left: 8px">
            异常: {{ healthCheckResults.filter(r => r.status === 'degraded').length }}
          </el-tag>
        </div>

        <!-- 结果表格 -->
        <el-table :data="healthCheckResults" border stripe style="margin-top: 16px" max-height="500">
          <el-table-column prop="server_name" label="服务器" width="140" />
          <el-table-column prop="server_host" label="IP" width="140" />
          <el-table-column label="状态" width="90" align="center">
            <template #default="{ row }">
              <el-tag :type="getStatusType(row.status)" effect="dark">{{ getStatusLabel(row.status) }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="API" width="70" align="center">
            <template #default="{ row }">
              <el-tag :type="row.api_reachable ? 'success' : 'danger'" size="small">
                {{ row.api_reachable ? 'OK' : 'FAIL' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="转发端口" width="110" align="center">
            <template #default="{ row }">
              <span :class="{ 'text-red-500': row.actual_ports < row.expected_ports }">
                {{ row.actual_ports }} / {{ row.expected_ports }}
              </span>
            </template>
          </el-table-column>
          <el-table-column label="活跃连接" width="90" align="right">
            <template #default="{ row }">{{ row.current_conns }}</template>
          </el-table-column>
          <el-table-column label="总连接" width="90" align="right">
            <template #default="{ row }">{{ row.total_conns }}</template>
          </el-table-column>
          <el-table-column label="流入" width="90" align="right">
            <template #default="{ row }">{{ formatBytes(row.input_bytes) }}</template>
          </el-table-column>
          <el-table-column label="流出" width="90" align="right">
            <template #default="{ row }">{{ formatBytes(row.output_bytes) }}</template>
          </el-table-column>
          <el-table-column label="错误率" width="90" align="right">
            <template #default="{ row }">
              <span :class="{ 'text-red-500 font-bold': row.error_rate >= 0.05 }">
                {{ (row.error_rate * 100).toFixed(2) }}%
              </span>
            </template>
          </el-table-column>
          <el-table-column label="耗时" width="70" align="right">
            <template #default="{ row }">{{ row.check_duration }}ms</template>
          </el-table-column>
        </el-table>

        <!-- 错误详情 -->
        <div v-for="r in healthCheckResults.filter(r => r.error_message)" :key="r.server_id" class="health-error-item">
          <el-alert :title="`${r.server_name}: ${r.error_message}`" type="error" :closable="false" show-icon />
        </div>
      </div>

      <template #footer>
        <el-button :loading="isChecking" type="primary" @click="doHealthCheck">重新检查</el-button>
        <el-button @click="healthCheckDialogVisible = false">关闭</el-button>
      </template>
    </el-dialog>

    <!-- GOST 诊断修复弹窗 -->
    <el-dialog v-model="diagnoseDialogVisible" title="GOST 诊断修复" width="700px" :close-on-click-modal="false">
      <el-alert type="info" :closable="false" style="margin-bottom: 12px">
        自动检测 GOST 二进制、服务状态、配置文件、端口监听，并尝试修复问题。
      </el-alert>
      <div class="deploy-log-container">
        <div class="deploy-log-title">诊断日志:</div>
        <div ref="diagnoseLogRef" class="deploy-log-content">
          <div v-for="(log, idx) in diagnoseLogs" :key="idx" class="deploy-log-line">{{ log }}</div>
        </div>
      </div>
      <template #footer>
        <el-button @click="diagnoseDialogVisible = false" :disabled="isDiagnosing">关闭</el-button>
        <el-button type="warning" @click="startDiagnose" :loading="isDiagnosing">
          {{ isDiagnosing ? '诊断中...' : '重新诊断' }}
        </el-button>
      </template>
    </el-dialog>

    <!-- 带宽测速弹窗 -->
    <el-dialog v-model="speedTestDialogVisible" title="GOST 隧道测速" width="700px" :close-on-click-modal="false">
      <el-alert type="info" :closable="false" style="margin-bottom: 12px">
        通过 SSH 在中继服务器上测试公网带宽和到 App 节点的网络质量。
      </el-alert>
      <div class="deploy-log-container">
        <div class="deploy-log-title">测速日志:</div>
        <div ref="speedTestLogRef" class="deploy-log-content">
          <div v-for="(log, idx) in speedTestLogs" :key="idx" class="deploy-log-line">{{ log }}</div>
        </div>
      </div>
      <!-- 结果摘要 -->
      <template v-if="speedTestResult && !isSpeedTesting">
        <el-divider content-position="left">测速结果</el-divider>
        <el-descriptions :column="2" border size="small">
          <el-descriptions-item label="公网下载">
            {{ speedTestResult.public_download_kbps > 0 ? `${(speedTestResult.public_download_kbps / 1024 * 8).toFixed(1)} Mbps` : '-' }}
          </el-descriptions-item>
          <el-descriptions-item label="公网上传">
            {{ speedTestResult.public_upload_kbps > 0 ? `${(speedTestResult.public_upload_kbps / 1024 * 8).toFixed(1)} Mbps` : '-' }}
          </el-descriptions-item>
        </el-descriptions>
        <el-table v-if="speedTestResult.latencies?.length" :data="speedTestResult.latencies" size="small" style="margin-top: 12px">
          <el-table-column prop="target" label="目标节点" />
          <el-table-column prop="avg_ms" label="平均延迟 (ms)">
            <template #default="{ row }">{{ row.avg_ms.toFixed(1) }} ms</template>
          </el-table-column>
        </el-table>
        <el-table v-if="speedTestResult.internal_speeds?.length" :data="speedTestResult.internal_speeds" size="small" style="margin-top: 12px">
          <el-table-column prop="target" label="目标节点" />
          <el-table-column prop="speed_mbs" label="下载吞吐量">
            <template #default="{ row }">
              {{ row.speed_mbs >= 1 ? row.speed_mbs.toFixed(2) + ' MB/s' : row.speed_mbs > 0 ? (row.speed_mbs * 1024).toFixed(0) + ' KB/s' : '-' }}
              <span v-if="row.speed_mbs > 0" style="color: #909399; margin-left: 4px">({{ (row.speed_mbs * 8).toFixed(1) }} Mbps)</span>
            </template>
          </el-table-column>
        </el-table>
        <el-table v-if="speedTestResult.gost_upload_speeds?.length" :data="speedTestResult.gost_upload_speeds" size="small" style="margin-top: 12px">
          <el-table-column prop="target" label="目标节点" />
          <el-table-column prop="speed_mbs" label="上传吞吐量">
            <template #default="{ row }">
              {{ row.speed_mbs >= 1 ? row.speed_mbs.toFixed(2) + ' MB/s' : row.speed_mbs > 0 ? (row.speed_mbs * 1024).toFixed(0) + ' KB/s' : '-' }}
              <span v-if="row.speed_mbs > 0" style="color: #909399; margin-left: 4px">({{ (row.speed_mbs * 8).toFixed(1) }} Mbps)</span>
            </template>
          </el-table-column>
        </el-table>
      </template>
      <template #footer>
        <el-button @click="speedTestDialogVisible = false" :disabled="isSpeedTesting">关闭</el-button>
        <el-button type="primary" @click="startSpeedTest" :loading="isSpeedTesting">
          {{ isSpeedTesting ? '测速中...' : '重新测速' }}
        </el-button>
      </template>
    </el-dialog>

    <!-- 带宽管理弹窗 -->
    <el-dialog v-model="bandwidthDialogVisible" title="ECS 实例带宽管理" width="500px" :close-on-click-modal="false">
      <div v-loading="isBandwidthLoading">
        <template v-if="bandwidthInfo">
          <el-descriptions :column="1" border size="small">
            <el-descriptions-item label="服务器">{{ bandwidthInfo.server_name }}</el-descriptions-item>
            <el-descriptions-item label="网络类型">
              <el-tag v-if="bandwidthInfo.has_eip" type="warning" size="small">EIP 弹性公网IP</el-tag>
              <el-tag v-else type="success" size="small">实例公网IP</el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="计费方式">{{ bandwidthInfo.internet_charge_type === 'PayByTraffic' ? '按流量' : bandwidthInfo.eip_charge_type === 'PayByTraffic' ? '按流量' : '按带宽' }}</el-descriptions-item>
            <el-descriptions-item label="入带宽">{{ bandwidthInfo.internet_max_bandwidth_in }} Mbps</el-descriptions-item>
            <el-descriptions-item label="出带宽(当前)">{{ bandwidthInfo.internet_max_bandwidth_out }} Mbps</el-descriptions-item>
          </el-descriptions>
          <el-divider content-position="left">修改出带宽</el-divider>
          <el-form label-width="120px">
            <el-form-item label="出带宽 (Mbps)">
              <el-input-number v-model="newBandwidthOut" :min="1" :max="200" :step="1" />
            </el-form-item>
          </el-form>
        </template>
        <el-empty v-else-if="!isBandwidthLoading" description="未能获取带宽信息" />
      </div>
      <template #footer>
        <el-button @click="bandwidthDialogVisible = false">关闭</el-button>
        <el-button type="primary" @click="saveBandwidth" :loading="isBandwidthSaving" :disabled="!bandwidthInfo">
          保存修改
        </el-button>
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

.health-summary {
  padding: 12px 16px;
  background: #f5f7fa;
  border-radius: 8px;
}

.health-error-item {
  margin-top: 12px;
}

// TLS tab styles
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.mt-4 {
  margin-top: 16px;
}

.mb-4 {
  margin-bottom: 16px;
}

.fingerprint-text {
  font-family: monospace;
  font-size: 12px;
  color: #303133;
  word-break: break-all;
}

.tunnel-toolbar {
  padding: 4px 0;
}
</style>
