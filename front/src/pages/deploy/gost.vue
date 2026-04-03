<script lang="ts" setup>
import type { ServerResp, TlsCertificateResp, TlsStatusResp } from "@@/apis/deploy/type"
import type { MerchantResp } from "@@/apis/merchant/type"
import type { VxeFormInstance, VxeFormProps, VxeGridInstance, VxeGridProps, VxeModalInstance, VxeModalProps } from "vxe-table"
import { createGostServiceByAPI, deleteGostServiceByAPI, getGostServiceDetail, getServerList, listGostChains, listGostServices, updateGostServiceDetail, setupGostForward, clearGostForward, getGostForwardStatus, getProgramConfig, updateProgramConfig, getNginxCacheStatus, clearNginxCache, persistGostConfig, getGostConfigSyncStatus, setupGostDeploy, rebuildAllMerchantGost, connectionCheck, connectionCheckByMerchant, connectionCheckByServer, repairGostServer, enableNginxCache, generateTlsCerts, getTlsCerts, disableTlsCerts, getTlsCertFingerprint, getTlsStatus, verifyTlsStatus, batchUpgradeTls, batchRollbackTls } from "@@/apis/deploy"
import type { GostConfigSyncStatusResp } from "@@/apis/deploy/type"
import { getMerchantList, reorderMerchantGostServers, importOssFromTargets, syncMerchantGostIP } from "@@/apis/merchant"
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

// ========== 批量重建 GOST 规则 ==========
const rebuildDialogVisible = ref(false)
const rebuildLogs = ref<string[]>([])
const isRebuilding = ref(false)
const rebuildLogRef = ref<HTMLDivElement>()

function openRebuildDialog() {
  rebuildLogs.value = []
  rebuildDialogVisible.value = true
}

function startRebuild() {
  isRebuilding.value = true
  rebuildLogs.value = ["开始重建所有商户 GOST 规则..."]

  rebuildAllMerchantGost(
    (data: any, isComplete?: boolean) => {
      if (data?.message) {
        rebuildLogs.value.push(data.message)
        nextTick(() => {
          if (rebuildLogRef.value) {
            rebuildLogRef.value.scrollTop = rebuildLogRef.value.scrollHeight
          }
        })
      }
      if (isComplete) {
        isRebuilding.value = false
        if (data?.success) {
          ElMessage.success("重建完成!")
          loadRecords()
        } else {
          ElMessage.error(data?.message || "重建失败")
        }
      }
    },
    (error: any) => {
      isRebuilding.value = false
      rebuildLogs.value.push(`错误: ${error?.message || error}`)
      ElMessage.error("重建失败")
    }
  )
}

// ========== 连接状态检测 ==========
const checkDialogVisible = ref(false)
const checkLoading = ref(false)
const checkResult = ref<any>(null)

async function runConnectionCheck() {
  checkDialogVisible.value = true
  checkLoading.value = true
  checkResult.value = null
  try {
    const res = await connectionCheck()
    checkResult.value = res.data
  } catch (e: any) {
    ElMessage.error("检测失败: " + (e?.message || e))
  } finally {
    checkLoading.value = false
  }
}

// ========== 单服务器检测 ==========
const serverCheckLoading = ref(false)
const serverCheckResult = ref<any>(null)
const serverCheckDialogVisible = ref(false)

async function runServerCheck() {
  if (!selectedServerId.value) {
    ElMessage.warning("请先选择系统服务器")
    return
  }
  serverCheckDialogVisible.value = true
  serverCheckLoading.value = true
  serverCheckResult.value = null
  try {
    const res = await connectionCheckByServer(selectedServerId.value)
    serverCheckResult.value = res.data
  } catch (e: any) {
    ElMessage.error("检测失败: " + (e?.message || e))
  } finally {
    serverCheckLoading.value = false
  }
}

// ========== 隧道修复 ==========
const repairDialogVisible = ref(false)
const isRepairing = ref(false)
const repairLogs = ref<string[]>([])
const repairLogRef = ref<HTMLElement>()

function startRepair() {
  if (!selectedServerId.value) {
    ElMessage.warning("请先选择系统服务器")
    return
  }
  repairDialogVisible.value = true
  repairLogs.value = []
  isRepairing.value = true

  repairGostServer(selectedServerId.value, (chunk: any, isComplete?: boolean) => {
    if (chunk?.message) {
      repairLogs.value.push(chunk.message)
      nextTick(() => { repairLogRef.value?.scrollTo(0, repairLogRef.value.scrollHeight) })
    }
    if (isComplete) {
      isRepairing.value = false
      if (chunk?.success) {
        ElMessage.success("修复完成")
        loadRecords()
        loadSyncStatus()
      } else {
        ElMessage.error("修复失败: " + (chunk?.message || "未知错误"))
      }
    }
  }, (err: any) => {
    isRepairing.value = false
    repairLogs.value.push("错误: " + (err?.message || err))
  })
}

function statusTag(s: string) {
  if (s === "ok") return "success"
  if (s === "protected") return "warning"
  return "danger"
}

async function doImportOss(merchantId: number, merchantName: string) {
  try {
    const res = await importOssFromTargets(merchantId)
    const count = res.data?.imported || 0
    if (count > 0) {
      ElMessage.success(`${merchantName}: 成功导入 ${count} 个 OSS 配置`)
    } else {
      ElMessage.info(`${merchantName}: 无新增（已全部导入或工具页无目标）`)
    }
  } catch (e: any) {
    ElMessage.error(`导入失败: ${e?.message || e}`)
  }
}

async function moveGostServer(merchantId: number, servers: any[], index: number, direction: "up" | "down") {
  const newIndex = direction === "up" ? index - 1 : index + 1
  if (newIndex < 0 || newIndex >= servers.length) return
  // swap
  const tmp = servers[index]
  servers[index] = servers[newIndex]
  servers[newIndex] = tmp
  // 保存新顺序
  const ids = servers.map((s: any) => s.relation_id || s.id)
  try {
    await reorderMerchantGostServers(merchantId, ids)
    ElMessage.success("排序已保存，同步 IP 后生效")
  } catch (e: any) {
    ElMessage.error("排序保存失败: " + (e?.message || e))
  }
}

// 单个商户检测中状态
const merchantChecking = ref<Record<number, boolean>>({})
const merchantSyncing = ref<Record<number, boolean>>({})

async function checkSingleMerchant(merchantId: number) {
  merchantChecking.value[merchantId] = true
  try {
    const res = await connectionCheckByMerchant(merchantId)
    // 更新 checkResult 中对应商户的数据
    if (checkResult.value) {
      const idx = checkResult.value.merchants.findIndex((m: any) => m.merchant_id === merchantId)
      if (idx >= 0) {
        checkResult.value.merchants[idx] = res.data
      }
    }
    ElMessage.success("检测完成")
  } catch (e: any) {
    ElMessage.error("检测失败: " + (e?.message || e))
  } finally {
    merchantChecking.value[merchantId] = false
  }
}

async function syncSingleMerchantIP(merchantId: number, merchantName: string) {
  merchantSyncing.value[merchantId] = true
  try {
    const res = await syncMerchantGostIP(merchantId)
    const results = res.data?.results || []
    const successCount = results.filter((r: any) => r.success).length
    ElMessage.success(`${merchantName}: 同步完成 (${successCount}/${results.length} 个 OSS)`)
  } catch (e: any) {
    ElMessage.error(`${merchantName} 同步失败: ${e?.message || e}`)
  } finally {
    merchantSyncing.value[merchantId] = false
  }
}

// ========== Nginx 文件缓存 ==========
const cacheEnableDialogVisible = ref(false)
const cacheEnableLogs = ref<string[]>([])
const isCacheEnabling = ref(false)
const cacheEnableLogRef = ref<HTMLDivElement>()

function openCacheEnableDialog() {
  cacheEnableLogs.value = []
  cacheEnableDialogVisible.value = true
}

function startCacheEnable() {
  const sysServerIds = serverList.value.map(s => s.id)
  if (sysServerIds.length === 0) {
    ElMessage.warning("无系统服务器")
    return
  }
  isCacheEnabling.value = true
  cacheEnableLogs.value = ["开始启用文件缓存..."]

  enableNginxCache(
    { server_ids: sysServerIds },
    (data: any, isComplete?: boolean) => {
      if (data?.message) {
        cacheEnableLogs.value.push(data.message)
        nextTick(() => {
          if (cacheEnableLogRef.value) cacheEnableLogRef.value.scrollTop = cacheEnableLogRef.value.scrollHeight
        })
      }
      if (isComplete) {
        isCacheEnabling.value = false
        if (data?.success) ElMessage.success("缓存启用完成!")
      }
    },
    (error: any) => {
      isCacheEnabling.value = false
      cacheEnableLogs.value.push(`错误: ${error?.message || error}`)
    }
  )
}

// 全选/取消全选商户
function toggleAllMerchants(checked: any) {
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
      loadSyncStatus()
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

// ========== TLS 证书管理（合并自 tls.vue） ==========
const activeTab = ref("services")
const tlsLoading = ref(false)
const tlsCerts = ref<TlsCertificateResp[]>([])
const tlsFingerprint = ref("")
const tlsFingerprintExpires = ref("")
const tlsStatus = ref<TlsStatusResp | null>(null)

async function loadTlsCerts() {
  tlsLoading.value = true
  try {
    const res = await getTlsCerts()
    tlsCerts.value = Array.isArray(res.data) ? res.data : []
  } catch {
    tlsCerts.value = []
  } finally {
    tlsLoading.value = false
  }
}

async function loadTlsFingerprint() {
  try {
    const res = await getTlsCertFingerprint()
    tlsFingerprint.value = res.data.fingerprint || ""
    tlsFingerprintExpires.value = res.data.expires_at || ""
  } catch {
    tlsFingerprint.value = ""
  }
}

async function loadTlsStatus() {
  try {
    const res = await getTlsStatus()
    tlsStatus.value = res.data
  } catch {
    tlsStatus.value = null
  }
}

async function handleTlsGenerate() {
  ElMessageBox.confirm(
    "将生成新的 CA 根证书和服务器证书，旧证书将被停用。已部署的服务器需要重新升级 TLS。确定继续？",
    "生成证书",
    { type: "warning" }
  ).then(async () => {
    tlsLoading.value = true
    try {
      await generateTlsCerts({})
      ElMessage.success("证书生成成功")
      await loadTlsCerts()
      await loadTlsFingerprint()
    } catch (e: any) {
      ElMessage.error(e.message || "证书生成失败")
    } finally {
      tlsLoading.value = false
    }
  })
}

async function handleTlsDisable() {
  ElMessageBox.confirm("停用证书后，新部署的服务器将不会自动启用 TLS。确定停用？", "停用证书", { type: "warning" }).then(async () => {
    tlsLoading.value = true
    try {
      await disableTlsCerts()
      ElMessage.success("证书已停用")
      await loadTlsCerts()
      tlsFingerprint.value = ""
    } catch (e: any) {
      ElMessage.error(e.message || "停用失败")
    } finally {
      tlsLoading.value = false
    }
  })
}

function copyTlsFingerprint() {
  if (!tlsFingerprint.value) return
  navigator.clipboard.writeText(tlsFingerprint.value).then(() => {
    ElMessage.success("指纹已复制到剪贴板")
  })
}

async function handleTlsUpgrade() {
  ElMessageBox.confirm("将所有系统服务器升级为 TLS 模式？", "批量升级", { type: "warning" }).then(async () => {
    tlsLoading.value = true
    try {
      const res = await batchUpgradeTls({})
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
  ElMessageBox.confirm("将所有系统服务器回滚为 TCP 模式？", "批量回滚", { type: "warning" }).then(async () => {
    tlsLoading.value = true
    try {
      const res = await batchRollbackTls({})
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
  tlsLoading.value = true
  try {
    const res = await verifyTlsStatus()
    tlsStatus.value = res.data
    ElMessage.success("验证完成")
  } catch (e: any) {
    ElMessage.error(e.message || "验证失败")
  } finally {
    tlsLoading.value = false
  }
}

// 切换到 TLS Tab 时懒加载
watch(activeTab, (val) => {
  if (val === "tls" && tlsCerts.value.length === 0) {
    Promise.all([loadTlsCerts(), loadTlsFingerprint(), loadTlsStatus()])
  }
})

// ========== 生命周期 ==========
onMounted(() => {
  loadServerList()
  getMerchantList({ page: 1, size: 2000 }).then((res) => {
    merchantList.value = Array.isArray(res.data?.list) ? res.data.list : []
  })
})
</script>

<template>
  <div class="app-container">
    <el-tabs v-model="activeTab" type="border-card">
      <el-tab-pane label="GOST 服务管理" name="services">

    <!-- 服务器选择 + 视图切换 -->
    <el-card class="server-select-card" shadow="never">
      <div class="flex items-center gap-4 flex-wrap">
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
        <!-- 配置同步状态 -->
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
        <el-divider direction="vertical" />
        <el-button type="success" icon="Plus" @click="openSetupDialog">一键部署</el-button>
        <el-button type="warning" icon="Refresh" @click="openRebuildDialog">批量重建</el-button>
        <el-button type="danger" icon="MagicStick" @click="startRepair" :loading="isRepairing">隧道修复</el-button>
        <el-divider direction="vertical" />
        <el-button type="primary" icon="Monitor" @click="runServerCheck" :loading="serverCheckLoading">检测当前</el-button>
        <el-button icon="Monitor" @click="runConnectionCheck">全部检测</el-button>
        <el-button icon="Cpu" @click="openCacheEnableDialog">文件缓存</el-button>
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
          <template v-if="row.merchant_name">
            <div class="merchant-info">
              <span class="merchant-name">{{ row.merchant_name }}</span>
            </div>
          </template>
          <template v-else-if="getMerchantByServiceName(row.name)">
            <div class="merchant-info">
              <span class="merchant-name">{{ getMerchantByServiceName(row.name)?.name }}</span>
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
            <div class="text-xs text-gray-500 truncate" :title="row.chain_target || (chainMap[row.handler?.chain]?.hops?.[0]?.nodes?.map((n: any) => n.addr).join(', ')) || '-'">
              {{ row.chain_target || (chainMap[row.handler?.chain]?.hops?.[0]?.nodes?.map((n: any) => n.addr).join(', ')) || '-' }}
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

      </el-tab-pane>

      <el-tab-pane label="TLS 证书" name="tls">
        <!-- 证书管理 -->
        <el-card v-loading="tlsLoading" shadow="never">
          <template #header>
            <div class="card-header">
              <span class="font-bold text-base">TLS 证书</span>
              <div>
                <el-button type="primary" @click="handleTlsGenerate">生成证书</el-button>
                <el-button type="danger" :disabled="tlsCerts.length === 0" @click="handleTlsDisable">停用证书</el-button>
              </div>
            </div>
          </template>

          <el-table v-if="tlsCerts.length > 0" :data="tlsCerts" border size="small">
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
        <el-card v-if="tlsFingerprint" shadow="never" style="margin-top: 16px">
          <template #header>
            <span class="font-bold text-base">App 证书指纹 (Certificate Pinning)</span>
          </template>
          <el-descriptions :column="1" border size="small">
            <el-descriptions-item label="SHA-256 指纹">
              <code class="fingerprint-text">{{ tlsFingerprint }}</code>
              <el-button type="primary" link size="small" style="margin-left: 8px" @click="copyTlsFingerprint">复制</el-button>
            </el-descriptions-item>
            <el-descriptions-item label="证书过期时间">{{ tlsFingerprintExpires }}</el-descriptions-item>
          </el-descriptions>
        </el-card>

        <!-- 系统服务器 TLS 状态 -->
        <el-card v-loading="tlsLoading" shadow="never" style="margin-top: 16px">
          <template #header>
            <div class="card-header">
              <span class="font-bold text-base">系统服务器 TLS 状态</span>
              <div>
                <el-button @click="handleTlsVerify">验证连接</el-button>
                <el-button type="warning" @click="handleTlsRollback">批量回滚 TCP</el-button>
                <el-button type="success" :disabled="tlsCerts.length === 0" @click="handleTlsUpgrade">批量升级 TLS</el-button>
              </div>
            </div>
          </template>

          <div v-if="tlsStatus">
            <el-descriptions :column="3" border size="small" style="margin-bottom: 16px">
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
          <el-empty v-else description="暂无系统服务器" />
        </el-card>
      </el-tab-pane>
    </el-tabs>

    <!-- 编辑弹窗（JSON） -->
    <vxe-modal ref="xModalDom" v-bind="xModalOpt">
      <vxe-form ref="xFormDom" v-bind="xFormOpt" />
    </vxe-modal>

    <!-- 新增弹窗 -->
    <vxe-modal ref="xCreateModalDom" v-bind="xCreateModalOpt">
      <vxe-form ref="xCreateFormDom" v-bind="xCreateFormOpt" />
    </vxe-modal>

    <!-- 一键部署弹窗 -->
    <el-dialog v-model="setupDialogVisible" title="一键部署 GOST" width="700px" :close-on-click-modal="false">
      <el-form :model="setupForm" label-width="120px">
        <el-form-item label="系统服务器" required>
          <el-select v-model="setupForm.server_id" placeholder="选择系统服务器" style="width: 100%">
            <el-option v-for="s in allServerList.filter(s => s.server_type === 2)" :key="s.id" :label="`${s.name} (${s.host})`" :value="s.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="商户" required>
          <div style="margin-bottom: 8px">
            <el-checkbox :model-value="setupForm.merchant_ids.length === merchantList.length && merchantList.length > 0" :indeterminate="setupForm.merchant_ids.length > 0 && setupForm.merchant_ids.length < merchantList.length" @change="toggleAllMerchants">全选</el-checkbox>
          </div>
          <el-checkbox-group v-model="setupForm.merchant_ids">
            <el-checkbox v-for="m in merchantList" :key="m.id" :value="m.id">{{ m.name }} (端口:{{ m.port }}, IP:{{ m.server_ip || '未配置' }})</el-checkbox>
          </el-checkbox-group>
        </el-form-item>
        <el-form-item label="转发模式">
          <el-radio-group v-model="setupForm.forward_type">
            <el-radio :value="1">加密 (relay+tls)</el-radio>
            <el-radio :value="2">直连 (tcp)</el-radio>
          </el-radio-group>
        </el-form-item>
      </el-form>

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

    <!-- 批量重建弹窗 -->
    <el-dialog v-model="rebuildDialogVisible" title="批量重建 GOST 规则" width="700px" :close-on-click-modal="false">
      <el-alert type="warning" :closable="false" style="margin-bottom: 12px">
        将清除所有系统服务器上的 GOST 转发规则，然后按商户关联关系重新创建。确保所有 GOST 服务器在线。
      </el-alert>
      <div ref="rebuildLogRef" style="height: 400px; overflow-y: auto; background: #1e1e1e; color: #d4d4d4; padding: 12px; border-radius: 4px; font-family: monospace; font-size: 13px; line-height: 1.6">
        <div v-for="(log, i) in rebuildLogs" :key="i">{{ log }}</div>
      </div>
      <template #footer>
        <el-button @click="rebuildDialogVisible = false" :disabled="isRebuilding">关闭</el-button>
        <el-button type="warning" @click="startRebuild" :loading="isRebuilding" :disabled="isRebuilding">
          {{ isRebuilding ? "重建中..." : "开始重建" }}
        </el-button>
      </template>
    </el-dialog>

    <!-- 文件缓存弹窗 -->
    <el-dialog v-model="cacheEnableDialogVisible" title="启用文件缓存 (Nginx)" width="700px" :close-on-click-modal="false">
      <el-alert type="info" :closable="false" style="margin-bottom: 12px">
        在所有系统服务器上安装 Nginx 并配置 8080 端口文件缓存，加速图片/文件加载。
      </el-alert>
      <div ref="cacheEnableLogRef" style="height: 350px; overflow-y: auto; background: #1e1e1e; color: #d4d4d4; padding: 12px; border-radius: 4px; font-family: monospace; font-size: 13px; line-height: 1.6">
        <div v-for="(log, i) in cacheEnableLogs" :key="i">{{ log }}</div>
      </div>
      <template #footer>
        <el-button @click="cacheEnableDialogVisible = false" :disabled="isCacheEnabling">关闭</el-button>
        <el-button type="primary" @click="startCacheEnable" :loading="isCacheEnabling">
          {{ isCacheEnabling ? "启用中..." : "开始启用" }}
        </el-button>
      </template>
    </el-dialog>

    <!-- 单服务器检测弹窗 -->
    <el-dialog v-model="serverCheckDialogVisible" title="服务器状态检测" width="500px" :close-on-click-modal="false">
      <div v-loading="serverCheckLoading">
        <div v-if="serverCheckResult">
          <el-descriptions :column="2" border size="small">
            <el-descriptions-item label="名称">{{ serverCheckResult.server_name }}</el-descriptions-item>
            <el-descriptions-item label="IP">{{ serverCheckResult.host }}</el-descriptions-item>
            <el-descriptions-item label="GOST API">
              <el-tag :type="statusTag(serverCheckResult.gost_api)" size="small">{{ serverCheckResult.gost_api }}</el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="服务/链">{{ serverCheckResult.service_count }}/{{ serverCheckResult.chain_count }}</el-descriptions-item>
            <el-descriptions-item label="443">
              <el-tag :type="statusTag(serverCheckResult.port_443)" size="small">{{ serverCheckResult.port_443 }}</el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="80">
              <el-tag :type="statusTag(serverCheckResult.port_80)" size="small">{{ serverCheckResult.port_80 }}</el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="8080">
              <el-tag :type="statusTag(serverCheckResult.port_8080)" size="small">{{ serverCheckResult.port_8080 }}</el-tag>
            </el-descriptions-item>
          </el-descriptions>
          <div v-if="serverCheckResult.service_count === 0 || serverCheckResult.port_443 === 'fail'" style="margin-top: 12px">
            <el-alert type="warning" :closable="false">检测到异常，建议使用「隧道修复」功能修复</el-alert>
          </div>
        </div>
      </div>
      <template #footer>
        <el-button @click="serverCheckDialogVisible = false">关闭</el-button>
        <el-button type="primary" @click="runServerCheck" :loading="serverCheckLoading">刷新</el-button>
        <el-button type="danger" icon="MagicStick" @click="serverCheckDialogVisible = false; startRepair()" v-if="serverCheckResult && (serverCheckResult.service_count === 0 || serverCheckResult.port_443 === 'fail')">修复</el-button>
      </template>
    </el-dialog>

    <!-- 隧道修复弹窗 -->
    <el-dialog v-model="repairDialogVisible" title="隧道修复" width="700px" :close-on-click-modal="false">
      <div ref="repairLogRef" style="height: 400px; overflow-y: auto; background: #1e1e1e; color: #d4d4d4; padding: 12px; border-radius: 4px; font-family: monospace; font-size: 13px; line-height: 1.6">
        <div v-for="(log, i) in repairLogs" :key="i">{{ log }}</div>
        <div v-if="repairLogs.length === 0 && !isRepairing" style="color: #666">点击「开始修复」自动诊断并修复当前服务器的 GOST 隧道问题</div>
      </div>
      <template #footer>
        <el-button @click="repairDialogVisible = false" :disabled="isRepairing">关闭</el-button>
        <el-button type="danger" icon="MagicStick" @click="startRepair" :loading="isRepairing">
          {{ isRepairing ? "修复中..." : "开始修复" }}
        </el-button>
      </template>
    </el-dialog>

    <!-- 连接检测弹窗 -->
    <el-dialog v-model="checkDialogVisible" title="连接状态检测" width="900px" :close-on-click-modal="false">
      <div v-loading="checkLoading">
        <div v-if="checkResult" v-for="m in checkResult.merchants" :key="m.merchant_id" style="margin-bottom: 20px">
          <el-divider content-position="left">
            {{ m.merchant_name }} ({{ m.server_ip }})
            <el-button size="small" type="primary" link @click="checkSingleMerchant(m.merchant_id)" :loading="merchantChecking[m.merchant_id]" style="margin-left: 12px">单独检测</el-button>
            <el-button size="small" type="success" link @click="syncSingleMerchantIP(m.merchant_id, m.merchant_name)" :loading="merchantSyncing[m.merchant_id]">同步 IP 到 OSS</el-button>
            <el-button size="small" type="primary" link @click="doImportOss(m.merchant_id, m.merchant_name)">导入 OSS</el-button>
          </el-divider>

          <div style="margin-bottom: 8px; font-weight: bold; font-size: 13px">商户服务端口</div>
          <el-space wrap>
            <el-tag v-for="svc in m.services" :key="svc.port" :type="statusTag(svc.status)" size="default">
              {{ svc.name }}:{{ svc.port }} {{ svc.status === "ok" ? "✓" : "✗" }}
            </el-tag>
          </el-space>

          <div v-if="m.gost_servers.length > 0" style="margin-top: 12px">
            <div style="margin-bottom: 8px; font-weight: bold; font-size: 13px">GOST 系统服务器</div>
            <el-table :data="m.gost_servers" size="small" border stripe>
              <el-table-column label="#" width="40">
                <template #default="{ $index }">{{ $index + 1 }}</template>
              </el-table-column>
              <el-table-column prop="server_name" label="名称" width="140" />
              <el-table-column prop="host" label="IP" width="120" />
              <el-table-column label="API" width="70">
                <template #default="{ row }">
                  <el-tag :type="statusTag(row.gost_api)" size="small">{{ row.gost_api }}</el-tag>
                </template>
              </el-table-column>
              <el-table-column label="服务/链" width="70">
                <template #default="{ row }">{{ row.service_count }}/{{ row.chain_count }}</template>
              </el-table-column>
              <el-table-column label="443" width="55">
                <template #default="{ row }"><el-tag :type="statusTag(row.port_443)" size="small">{{ row.port_443 }}</el-tag></template>
              </el-table-column>
              <el-table-column label="80" width="55">
                <template #default="{ row }"><el-tag :type="statusTag(row.port_80)" size="small">{{ row.port_80 }}</el-tag></template>
              </el-table-column>
              <el-table-column label="8080" width="55">
                <template #default="{ row }"><el-tag :type="statusTag(row.port_8080)" size="small">{{ row.port_8080 }}</el-tag></template>
              </el-table-column>
              <el-table-column label="排序" width="90" v-if="m.gost_servers.length > 1">
                <template #default="{ $index }">
                  <el-button-group size="small">
                    <el-button :disabled="$index === 0" @click="moveGostServer(m.merchant_id, m.gost_servers, $index, 'up')">↑</el-button>
                    <el-button :disabled="$index === m.gost_servers.length - 1" @click="moveGostServer(m.merchant_id, m.gost_servers, $index, 'down')">↓</el-button>
                  </el-button-group>
                </template>
              </el-table-column>
            </el-table>
          </div>
          <el-alert v-else type="warning" :closable="false" style="margin-top: 8px">无关联系统服务器</el-alert>
        </div>
        <el-empty v-if="checkResult && checkResult.merchants.length === 0" description="无有效商户" />
      </div>
      <template #footer>
        <el-button @click="checkDialogVisible = false">关闭</el-button>
        <el-button type="primary" @click="runConnectionCheck" :loading="checkLoading">刷新</el-button>
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
          V2架构：统一入口 443(TLS) + TCP 10010，nginx 路径分发。<br/>
          商户端口: 10443(统一入口→nginx路径分发WS/HTTP/S3), 10010(TCP长连接)
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
          <el-input v-model="forwardForm.portsStr" placeholder="留空使用默认端口(10443,10010)，多个端口用逗号分隔" />
          <div class="text-xs text-gray-400 mt-1">V2默认: 10443(统一入口), 10010(TCP)</div>
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

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.fingerprint-text {
  font-family: monospace;
  font-size: 12px;
  color: #303133;
  word-break: break-all;
}
</style>
