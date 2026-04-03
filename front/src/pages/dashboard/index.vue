<script lang="ts" setup>
import type { FormInstance } from "element-plus"
import type { MerchantQueryRequestData, SmsConfig, TunnelCheckItem, TunnelStats } from "./apis/type"
import type { MerchantStorageResp } from "@@/apis/merchant_storage/type"
import { getClientList } from "@/common/apis/clients"
import { usePagination } from "@/common/composables/usePagination"
import { queryAdminmActive } from "@@/apis/adminm_users"
import { getMerchantStorageList, createMerchantStorage, updateMerchantStorage, deleteMerchantStorage, pushMerchantStorage } from "@@/apis/merchant_storage"
import {
  CirclePlus,
  Connection,
  Monitor,
  Refresh,
  Search,
  Shop
} from "@element-plus/icons-vue"
import { changeMerchantGostPortApi, changeMerchantIPApi, clearMerchantDataApi, exportMerchantDatabaseApi, getAdminmSmsConfigApi, getTunnelStatsApi, merchantQueryApi, pushWebLogoApi, saveAdminmNicknameApi, saveAdminmSensitiveContentsApi, saveAdminmSmsConfigApi, tunnelCheckApi, updateMerchantApi } from "./apis/index"
import { getTwoFAStatusApi } from "@/common/apis/twofa"
import ImageUploader from "@/common/components/ImageUploader.vue"
import AdminmUsersDialog from "./components/AdminmUsersDialog.vue"
import MerchantDetail from "./components/MerchantDetail.vue"
import MerchantForm from "./components/MerchantForm.vue"
import ServiceNodesDialog from "./components/ServiceNodesDialog.vue"
import { useMerchant } from "./composables/useMerchant"

const router = useRouter()

defineOptions({
  name: "MerchantManagement"
})

// 隧道统计
const tunnelStats = reactive<TunnelStats>({
  total_merchants: 0,
  total_gost_servers: 0,
  total_merchant_servers: 0
})
const statsLoading = ref(false)

async function fetchTunnelStats() {
  statsLoading.value = true
  try {
    const res = await getTunnelStatsApi()
    if (res.data) {
      tunnelStats.total_merchants = res.data.total_merchants
      tunnelStats.total_gost_servers = res.data.total_gost_servers
      tunnelStats.total_merchant_servers = res.data.total_merchant_servers
    }
  } catch (err) {
    console.error("获取隧道统计失败", err)
  } finally {
    statsLoading.value = false
  }
}

onMounted(() => {
  fetchTunnelStats()
})

// 分页相关
const { paginationData, handleCurrentChange, handleSizeChange } = usePagination()

// 查询相关
const searchFormRef = ref<FormInstance | null>(null)
const searchData = reactive<{
  name: string
  merchant_no: string
  expiring_soon: number | ""
  order?: string
}>({
  name: "",
  merchant_no: "",
  expiring_soon: "",
  order: undefined
})

// 商户相关
const {
  loading,
  tableData,
  dialogVisible,
  detailDialogVisible,
  formData,
  editingMerchantId,
  currentMerchant,
  total,
  getMerchantList,
  refreshList,
  showDetailDialog,
  submitForm,
  handleDelete
} = useMerchant()

// 服务节点管理弹窗
const serviceNodesVisible = ref(false)
const serviceNodesMerchantId = ref(0)
const serviceNodesMerchantName = ref("")
function openServiceNodes(row: any) {
  serviceNodesMerchantId.value = row.id
  serviceNodesMerchantName.value = row.name
  serviceNodesVisible.value = true
}

// adminm 用户管理弹窗
const adminmUsersVisible = ref(false)
const adminmMerchantNo = ref("")
function openAdminmUsers(row: any) {
  adminmMerchantNo.value = row.no
  adminmUsersVisible.value = true
}

// 活跃数据弹窗
const activeDialogVisible = ref(false)
const activeData = reactive({
  merchant_no: "",
  total_users: 0,
  online_users: 0,
  dau: 0
})
function openActiveDialog(row: any) {
  if (!row?.no) return
  activeData.merchant_no = row.no
  queryAdminmActive({ merchant_no: row.no })
    .then((res) => {
      activeData.total_users = res.data.total_users
      activeData.online_users = res.data.online_users
      activeData.dau = res.data.dau
      activeDialogVisible.value = true
    })
    .catch((err) => {
      console.error("获取活跃数据失败", err)
    })
}

// 隧道检测弹窗
const tunnelDialogVisible = ref(false)
const tunnelLoading = ref(false)
const tunnelResults = ref<TunnelCheckItem[]>([])
const tunnelTarget = reactive({
  server_ip: "",
  title: ""
})
function openTunnelDialog(row: any) {
  if (!row?.id) return
  tunnelResults.value = []
  tunnelTarget.server_ip = row.server_ip
  // 标题中不再写死端口，后端会根据当前 GOST 端口进行探测
  tunnelTarget.title = `隧道连接检测 - ${row.name} (${row.server_ip})`
  tunnelDialogVisible.value = true
  tunnelLoading.value = true
  tunnelCheckApi({ merchant_id: row.id })
    .then((res) => {
      tunnelResults.value = res.data || []
    })
    .catch((err) => {
      console.error("隧道检测失败", err)
    })
    .finally(() => {
      tunnelLoading.value = false
    })
}

// 更换IP
const changeIpLoading = ref(false)
function handleChangeIP(row: any) {
  if (changeIpLoading.value) return
  ElMessageBox.confirm(
    `确定为商户 “${row.name}” 更换公网IP 吗？`,
    "确认",
    { type: "warning", confirmButtonText: "确定", cancelButtonText: "取消" }
  )
    .then(() => {
      changeIpLoading.value = true
      return changeMerchantIPApi(row.id)
    })
    .then((res: any) => {
      if (!res) return
      const newIp = res.data?.new_ip
      ElMessage.success(`更换成功，新IP：${newIp || "(未知)"}`)
      refreshList()
    })
    .catch(() => {})
    .finally(() => {
      changeIpLoading.value = false
    })
}

// 更换隧道端口（GOST 转发端口）
const changeGostPortLoading = ref(false)
function handleChangeGostPort(row: any) {
  if (changeGostPortLoading.value) return
  const defaultPort = 10544
  ElMessageBox.prompt(
    `为商户 “${row.name}” 设置新的隧道端口（GOST 监听端口）。\n建议使用 10000–65535 的未占用端口，修改后需客户端/接入配置保持一致。`,
    "更换隧道端口",
    {
      confirmButtonText: "确定",
      cancelButtonText: "取消",
      inputPlaceholder: `例如：${defaultPort}`,
      inputType: "number",
      inputValidator: (value: string) => {
        const v = Number(value)
        if (!value || Number.isNaN(v)) return "请输入数字端口"
        if (v < 1 || v > 65535) return "端口范围必须在 1 ~ 65535 之间"
        return true
      }
    }
  )
    .then(({ value }) => {
      const port = Number(value)
      changeGostPortLoading.value = true
      return changeMerchantGostPortApi(row.id, port)
    })
    .then((res: any) => {
      if (!res) return
      const oldPort = res.data?.old_port
      const newPort = res.data?.new_port
      if (oldPort) {
        ElMessage.success(`隧道端口更换成功：${oldPort} → ${newPort}`)
      } else {
        ElMessage.success(`隧道端口更换成功，新端口：${newPort}`)
      }
    })
    .catch(() => {})
    .finally(() => {
      changeGostPortLoading.value = false
    })
}

// 一键打包
function handleQuickBuild(row: any) {
  // 检查是否有必要的打包信息
  if (!row.app_name && !row.name) {
    ElMessage.warning("请先在商户编辑中填写应用名称")
    return
  }
  if (!row.icon_url && !row.logo_url) {
    ElMessageBox.confirm(
      `商户 "${row.name}" 尚未配置 Logo 和图标，是否继续进入打包页面？`,
      "提示",
      { confirmButtonText: "继续", cancelButtonText: "去编辑", type: "warning" }
    )
      .then(() => {
        // 跳转到打包页面，带上商户信息
        router.push({
          name: "BuildMerchants",
          query: {
            merchant_id: row.id,
            merchant_name: row.name,
            app_name: row.app_name || row.name,
            logo_url: row.logo_url || "",
            icon_url: row.icon_url || "",
            enterprise_code: row.no
          }
        })
      })
      .catch(() => {
        // 跳转到编辑页面
        router.push({ name: "MerchantEdit", params: { id: row.id }, query: { data: JSON.stringify(row) } })
      })
    return
  }
  // 直接跳转到打包页面
  router.push({
    name: "BuildMerchants",
    query: {
      merchant_id: row.id,
      merchant_name: row.name,
      app_name: row.app_name || row.name,
      logo_url: row.logo_url || "",
      icon_url: row.icon_url || "",
      enterprise_code: row.no
    }
  })
}

// 导出商户数据库
const exportDbLoading = ref(false)
async function handleExportDatabase(row: any) {
  if (!row?.no) return

  try {
    await ElMessageBox.confirm(
      `确定导出商户 "${row.name}" 的数据库吗？\n\n导出使用 --single-transaction 快照读取，不会影响线上服务。`,
      "导出数据库",
      { confirmButtonText: "开始导出", cancelButtonText: "取消", type: "info" }
    )
  } catch {
    return
  }

  exportDbLoading.value = true
  ElMessage.info("正在导出数据库，请稍候...")
  try {
    const blob = (await exportMerchantDatabaseApi(row.no)) as unknown as Blob
    const url = URL.createObjectURL(blob)
    const a = document.createElement("a")
    a.href = url
    const dateStr = new Date().toISOString().slice(0, 10).replace(/-/g, "")
    a.download = `${row.name || row.no}_${dateStr}.sql.gz`
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    URL.revokeObjectURL(url)
    ElMessage.success("数据库导出完成")
  } catch (err: any) {
    if (err?.response?.data instanceof Blob) {
      const text = await err.response.data.text()
      try {
        const json = JSON.parse(text)
        ElMessage.error(json.message || "导出失败")
      } catch {
        ElMessage.error("导出失败")
      }
    } else {
      ElMessage.error(err?.message || "导出失败")
    }
  } finally {
    exportDbLoading.value = false
  }
}

// 推送 Logo 到 Web
const pushLogoLoading = ref(false)
async function handlePushLogo(row: any) {
  if (!row?.no) return
  const name = row.app_name || row.name || row.no
  try {
    await ElMessageBox.confirm(
      `确定推送 "${name}" 的 Logo 和应用名称到 Web 端吗？`,
      "推送Logo到Web",
      { confirmButtonText: "确定推送", cancelButtonText: "取消", type: "info" }
    )
  } catch {
    return
  }
  pushLogoLoading.value = true
  try {
    await pushWebLogoApi({ merchant_no: row.no })
    ElMessage.success("Logo推送成功")
  } catch (err: any) {
    ElMessage.error(err?.message || "推送失败")
  } finally {
    pushLogoLoading.value = false
  }
}

async function handleBatchPushLogo() {
  try {
    await ElMessageBox.confirm(
      "确定推送所有商户的 Logo 和应用名称到 Web 端吗？\n\n每个商户将使用自己的 Logo 和应用名称。",
      "批量推送Logo",
      { confirmButtonText: "全部推送", cancelButtonText: "取消", type: "warning" }
    )
  } catch {
    return
  }
  pushLogoLoading.value = true
  ElMessage.info("正在批量推送Logo，请稍候...")
  try {
    const res = await pushWebLogoApi({ broadcast: true, use_own_logo: true })
    ElMessage.success(`推送完成：成功 ${res.data.success}，失败 ${res.data.failed}，共 ${res.data.total}`)
  } catch (err: any) {
    ElMessage.error(err?.message || "批量推送失败")
  } finally {
    pushLogoLoading.value = false
  }
}

// 清除商户数据
const clearDataLoading = ref(false)
async function handleClearData(row: any) {
  if (!row?.no) return

  // 检查当前用户 2FA 状态
  let has2FA = false
  try {
    const statusRes = await getTwoFAStatusApi()
    has2FA = !!(statusRes as any)?.data?.enabled
  } catch { /* ignore */ }

  const promptMsg = has2FA
    ? `此操作将清除商户 "${row.name}" 的所有数据（用户、消息、文件等），但保留系统账号和管理账号。\n\n请输入 2FA 验证码确认：`
    : `此操作将清除商户 "${row.name}" 的所有数据（用户、消息、文件等），但保留系统账号和管理账号。\n\n请输入登录密码确认：`

  const inputType = has2FA ? "text" : "password"
  const placeholder = has2FA ? "请输入6位验证码" : "请输入登录密码"

  ElMessageBox.prompt(promptMsg, "危险操作 - 清除数据", {
    confirmButtonText: "确认清除",
    cancelButtonText: "取消",
    type: "error",
    inputType,
    inputPlaceholder: placeholder,
    inputValidator: (value: string) => {
      if (!value) return "不能为空"
      if (has2FA && !/^\d{6}$/.test(value)) return "请输入6位数字验证码"
      return true
    }
  })
    .then(({ value }) => {
      clearDataLoading.value = true
      const password = has2FA ? undefined : value
      const totp_code = has2FA ? value : undefined
      return clearMerchantDataApi(row.no, password, totp_code)
    })
    .then((res: any) => {
      if (!res) return
      ElMessage.success(`商户 "${row.name}" 的数据已清除（MySQL + Redis + MinIO + 消息缓存）`)
    })
    .catch(() => {})
    .finally(() => {
      clearDataLoading.value = false
    })
}

// 商户配置（短信）
const configDialogVisible = ref(false)
const configLoading = ref(false)
const smsConfigForm = reactive<SmsConfig>({
  provider: "aliyun",
  region_id: "",
  access_key: "",
  secret_key: "",
  sign_name: "",
  template_code: "",
  unisms_access_key_id: "",
  unisms_access_key_secret: "",
  unisms_signature: "",
  unisms_template_id: "",
  smsbao_account: "",
  smsbao_api_key: "",
  smsbao_template: ""
})
const configMerchantNo = ref("")
const isBatchConfig = ref(false)
const batchTarget = reactive<{ mode: "broadcast" | "merchant_nos", selectedNos: string[] }>({ mode: "broadcast", selectedNos: [] })
const batchMerchantLoading = ref(false)
const batchMerchantOptions = ref<{ label: string, value: string }[]>([])

// Client 选择用于预填配置
const clientLoading = ref(false)
const clientOptions = ref<{ label: string, value: number, raw: any }[]>([])
const selectedSmsClientId = ref<number | undefined>(undefined)

// 系统昵称/头像批量更新
const nicknameDialogVisible = ref(false)
const nicknameValue = ref("")
const avatarUrlValue = ref("")
const nicknameTarget = reactive<{ mode: "broadcast" | "merchant_nos", selectedNos: string[] }>({ mode: "broadcast", selectedNos: [] })
const isNicknameSingle = ref(false)
const nicknameSingleMerchantLabel = ref("")

// 敏感词批量更新
const sensitiveDialogVisible = ref(false)
const sensitiveTxt = ref("")
const sensitiveTarget = reactive<{ mode: "broadcast" | "merchant_nos", selectedNos: string[] }>({ mode: "broadcast", selectedNos: [] })
const sensitiveFileInputRef = ref<HTMLInputElement | null>(null)

function triggerSensitiveFileSelect() {
  sensitiveFileInputRef.value?.click()
}

function onSensitiveFileChange(e: Event) {
  const input = e.target as HTMLInputElement
  const file = input.files && input.files[0]
  if (!file) return
  // 简单的体积限制（可按需调整）
  const maxSize = 2 * 1024 * 1024 // 2MB
  if (file.size > maxSize) {
    ElMessage.error("文件过大，请选择不超过2MB的TXT文件")
    input.value = ""
    return
  }
  // 仅允许 .txt
  const name = file.name.toLowerCase()
  if (!name.endsWith(".txt")) {
    ElMessage.error("仅支持TXT文件")
    input.value = ""
    return
  }
  const reader = new FileReader()
  reader.onload = () => {
    const text = (reader.result as string) || ""
    // 统一换行符为 \n
    sensitiveTxt.value = text.replace(/\r\n?/g, "\n")
    // 读完后重置 input，便于再次选择同一文件
    input.value = ""
  }
  reader.onerror = () => {
    ElMessage.error("读取文件失败")
    input.value = ""
  }
  reader.readAsText(file, "utf-8")
}

function resetSmsForm() {
  smsConfigForm.provider = "aliyun"
  smsConfigForm.region_id = ""
  smsConfigForm.access_key = ""
  smsConfigForm.secret_key = ""
  smsConfigForm.sign_name = ""
  smsConfigForm.template_code = ""
  smsConfigForm.unisms_access_key_id = ""
  smsConfigForm.unisms_access_key_secret = ""
  smsConfigForm.unisms_signature = ""
  smsConfigForm.unisms_template_id = ""
  smsConfigForm.smsbao_account = ""
  smsConfigForm.smsbao_api_key = ""
  smsConfigForm.smsbao_template = ""
}

async function fetchSmsConfig(merchantNo: string) {
  configLoading.value = true
  try {
    const smsRes = await getAdminmSmsConfigApi(merchantNo)
    const data = smsRes.data
    if (data) {
      smsConfigForm.provider = data.provider || "aliyun"
      smsConfigForm.region_id = data.region_id || ""
      smsConfigForm.access_key = data.access_key || ""
      smsConfigForm.secret_key = data.secret_key || ""
      smsConfigForm.sign_name = data.sign_name || ""
      smsConfigForm.template_code = data.template_code || ""
      smsConfigForm.unisms_access_key_id = data.unisms_access_key_id || ""
      smsConfigForm.unisms_access_key_secret = data.unisms_access_key_secret || ""
      smsConfigForm.unisms_signature = data.unisms_signature || ""
      smsConfigForm.unisms_template_id = data.unisms_template_id || ""
      smsConfigForm.smsbao_account = data.smsbao_account || ""
      smsConfigForm.smsbao_api_key = data.smsbao_api_key || ""
      smsConfigForm.smsbao_template = data.smsbao_template || ""
    } else {
      resetSmsForm()
    }
  } catch {
    ElMessage.error("获取短信配置失败")
  } finally {
    configLoading.value = false
  }
}

function openSmsConfig(row: any) {
  if (!row?.no) return
  configMerchantNo.value = row.no
  resetSmsForm()
  selectedSmsClientId.value = undefined
  configDialogVisible.value = true
  isBatchConfig.value = false
  fetchSmsConfig(row.no)
  fetchClients("")
}

// 批量修改：打开短信配置
function openBatchSmsConfig() {
  configMerchantNo.value = ""
  isBatchConfig.value = true
  batchTarget.mode = "broadcast"
  batchTarget.selectedNos = []
  resetSmsForm()
  selectedSmsClientId.value = undefined
  configDialogVisible.value = true
  fetchClients("")
}

function openBatchNicknameDialog() {
  nicknameDialogVisible.value = true
  nicknameValue.value = ""
  avatarUrlValue.value = ""
  nicknameTarget.mode = "broadcast"
  nicknameTarget.selectedNos = []
  isNicknameSingle.value = false
  nicknameSingleMerchantLabel.value = ""
}

function openNicknameForRow(row: any) {
  if (!row?.no) return
  nicknameDialogVisible.value = true
  nicknameValue.value = ""
  avatarUrlValue.value = ""
  nicknameTarget.mode = "merchant_nos"
  nicknameTarget.selectedNos = [row.no]
  isNicknameSingle.value = true
  nicknameSingleMerchantLabel.value = `${row.name} (${row.no})`
}

function openBatchSensitiveDialog() {
  sensitiveDialogVisible.value = true
  sensitiveTxt.value = ""
  sensitiveTarget.mode = "broadcast"
  sensitiveTarget.selectedNos = []
}

// 查询商户列表
function fetchMerchantList() {
  const params: MerchantQueryRequestData = {
    page: paginationData.currentPage,
    size: paginationData.pageSize,
    name: searchData.name || undefined,
    merchant_no: searchData.merchant_no || undefined,
    order: searchData.order,
    expiring_soon: searchData.expiring_soon === "" ? undefined : searchData.expiring_soon
  }
  getMerchantList(params).then(() => {
    // 确保total值被更新到paginationData
    paginationData.total = total.value
  })
}

// 搜索
function handleSearch() {
  paginationData.currentPage === 1 ? fetchMerchantList() : (paginationData.currentPage = 1)
}

// 重置搜索
function resetSearch() {
  searchFormRef.value?.resetFields()
  searchData.name = ""
  searchData.expiring_soon = ""
  searchData.order = undefined
  handleSearch()
}

// 监听分页变化
watch(
  [() => paginationData.currentPage, () => paginationData.pageSize],
  fetchMerchantList,
  { immediate: true }
)

//

// 保存短信配置：单个或批量/全部
async function handleSaveSms() {
  try {
    if (isBatchConfig.value) {
      const smsPayload: any = { config: { ...smsConfigForm } }
      if (batchTarget.mode === "broadcast") {
        smsPayload.broadcast = true
      } else {
        const nos = batchTarget.selectedNos
        if (!nos || nos.length === 0) {
          ElMessage.error("请填写至少一个企业号")
          return
        }
        smsPayload.merchant_nos = nos
      }
      await saveAdminmSmsConfigApi(smsPayload)
      ElMessage.success("短信配置已推送更新")
      return
    }
    if (!configMerchantNo.value) {
      ElMessage.error("未指定商户")
      return
    }
    await saveAdminmSmsConfigApi({ merchant_no: configMerchantNo.value, config: { ...smsConfigForm } })
    ElMessage.success("短信配置已保存")
  } catch {
    // 错误提示由拦截器处理
  }
}

// 保存系统昵称/头像（单个/批量/全部）
async function handleSaveNickname() {
  const firstName = (nicknameValue.value || "").trim()
  const avatarUrl = (avatarUrlValue.value || "").trim()
  if (!firstName && !avatarUrl) {
    ElMessage.error("请输入系统昵称或头像URL")
    return
  }
  try {
    const payload: any = {}
    if (firstName) payload.first_name = firstName
    if (avatarUrl) payload.avatar_url = avatarUrl
    if (nicknameTarget.mode === "broadcast") {
      payload.broadcast = true
    } else {
      const nos = nicknameTarget.selectedNos
      if (!nos || nos.length === 0) {
        ElMessage.error("请至少选择一个商户")
        return
      }
      payload.merchant_nos = nos
    }
    await saveAdminmNicknameApi(payload)
    ElMessage.success("系统账号资料已下发更新")
  } catch {
  }
}

// 保存敏感词（从txt文本解析）
async function handleSaveSensitive() {
  const txt = (sensitiveTxt.value || "").trim()
  try {
    const payload: any = {}
    if (txt) {
      payload.txt = txt
    } else {
      // 传空数组用于清空敏感词
      payload.contents = []
    }
    if (sensitiveTarget.mode === "broadcast") {
      payload.broadcast = true
    } else {
      const nos = sensitiveTarget.selectedNos
      if (!nos || nos.length === 0) {
        ElMessage.error("请至少选择一个商户")
        return
      }
      payload.merchant_nos = nos
    }
    await saveAdminmSensitiveContentsApi(payload)
    ElMessage.success("敏感词已下发更新")
  } catch {}
}

// 监听查询条件变化
watch(
  [() => searchData.expiring_soon, () => searchData.order],
  () => {
    handleSearch()
  }
)

// 当敏感词切换到“指定企业号”时预加载
watch(
  () => sensitiveTarget.mode,
  (mode) => {
    if (mode === "merchant_nos") {
      fetchBatchMerchants("")
    }
  }
)

// 批量商户选择：远程搜索
async function fetchBatchMerchants(query: string) {
  batchMerchantLoading.value = true
  try {
    const res = await merchantQueryApi({
      page: 1,
      size: 50,
      name: query || undefined,
      merchant_no: query || undefined,
      order: undefined,
      expiring_soon: undefined
    } as unknown as MerchantQueryRequestData)
    const list = res.data?.list || []
    batchMerchantOptions.value = list.map((m: any) => ({ label: `${m.name} (${m.no})`, value: m.no }))
  } finally {
    batchMerchantLoading.value = false
  }
}

// 当切换到“指定企业号”时预加载
watch(
  () => batchTarget.mode,
  (mode) => {
    if (isBatchConfig.value && mode === "merchant_nos") {
      fetchBatchMerchants("")
    }
  }
)

// 远程搜索 Client
async function fetchClients(query: string) {
  clientLoading.value = true
  try {
    const res = await getClientList({
      page: 1,
      size: 50,
      app_name: query || undefined,
      app_package_name: query || undefined
    })
    const list = (res as any).data?.list || []
    clientOptions.value = list.map((c: any) => ({ label: `${c.app_name} (${c.app_package_name})`, value: c.id, raw: c }))
  } finally {
    clientLoading.value = false
  }
}

function onSelectSmsClient(id: number | undefined) {
  if (id == null) return
  const found = clientOptions.value.find(o => o.value === id)
  if (!found) return
  const cfg = found.raw?.sms_config
  if (!cfg) return
  smsConfigForm.provider = cfg.provider || "aliyun"
  smsConfigForm.region_id = cfg.region_id || ""
  smsConfigForm.access_key = cfg.access_key || ""
  smsConfigForm.secret_key = cfg.secret_key || ""
  smsConfigForm.sign_name = cfg.sign_name || ""
  smsConfigForm.template_code = cfg.template_code || ""
  smsConfigForm.unisms_access_key_id = cfg.unisms_access_key_id || ""
  smsConfigForm.unisms_access_key_secret = cfg.unisms_access_key_secret || ""
  smsConfigForm.unisms_signature = cfg.unisms_signature || ""
  smsConfigForm.unisms_template_id = cfg.unisms_template_id || ""
  smsConfigForm.smsbao_account = cfg.smsbao_account || ""
  smsConfigForm.smsbao_api_key = cfg.smsbao_api_key || ""
  smsConfigForm.smsbao_template = cfg.smsbao_template || ""
}

// Logo 上传相关
const logoUploadVisible = ref<Record<number, boolean>>({})
const uploadingLogoId = ref<number | null>(null)

function openLogoUpload(row: any) {
  logoUploadVisible.value[row.id] = true
}

async function handleLogoUpdate(row: any, url: string) {
  if (uploadingLogoId.value === row.id) return
  uploadingLogoId.value = row.id
  try {
    await updateMerchantApi({
      id: row.id,
      name: row.name,
      expired_at: row.expired_at,
      logo_url: url
    })
    // 更新本地数据
    row.logo_url = url
    ElMessage.success("Logo 更新成功")
    logoUploadVisible.value[row.id] = false
  } catch {
    ElMessage.error("Logo 更新失败")
  } finally {
    uploadingLogoId.value = null
  }
}

// ========== 存储配置弹窗 ==========
const storageDialogVisible = ref(false)
const storageLoading = ref(false)
const storageList = ref<MerchantStorageResp[]>([])
const storageMerchant = reactive({ id: 0, name: "", no: "" })

// 存储类型选项
const storageTypeOptions = [
  { label: "MinIO", value: "minio" },
  { label: "阿里云 OSS", value: "aliyunOSS" },
  { label: "AWS S3", value: "aws_s3" },
  { label: "腾讯云 COS", value: "tencent_cos" }
]

function getStorageTypeLabel(type: string) {
  return storageTypeOptions.find(o => o.value === type)?.label || type
}
function getStorageTypeTagType(type: string): "success" | "warning" | "info" | "primary" | "danger" {
  const m: Record<string, "success" | "warning" | "info" | "primary" | "danger"> = { minio: "primary", aliyunOSS: "warning", aws_s3: "success", tencent_cos: "info" }
  return m[type] || "info"
}

async function openStorageDialog(row: any) {
  storageMerchant.id = row.id
  storageMerchant.name = row.name
  storageMerchant.no = row.no
  storageDialogVisible.value = true
  await loadStorageList()
}

async function loadStorageList() {
  storageLoading.value = true
  try {
    const res = await getMerchantStorageList({ merchant_id: storageMerchant.id, page: 1, size: 100 })
    storageList.value = res.data.list || []
  } catch {
    storageList.value = []
  } finally {
    storageLoading.value = false
  }
}

// 存储配置编辑
const storageFormVisible = ref(false)
const storageFormLoading = ref(false)
const storageIsEdit = ref(false)
const storageEditId = ref(0)
const storageForm = reactive({
  storage_type: "minio",
  name: "",
  endpoint: "",
  bucket: "",
  region: "",
  access_key_id: "",
  access_key_secret: "",
  upload_url: "",
  download_url: "",
  file_base_url: "",
  bucket_url: "",
  custom_domain: "",
  is_default: 0,
  status: 1
})

function resetStorageForm() {
  Object.assign(storageForm, {
    storage_type: "minio", name: "", endpoint: "", bucket: "", region: "",
    access_key_id: "", access_key_secret: "", upload_url: "", download_url: "",
    file_base_url: "", bucket_url: "", custom_domain: "", is_default: 0, status: 1
  })
}

function showAddStorage() {
  storageIsEdit.value = false
  storageEditId.value = 0
  resetStorageForm()
  storageFormVisible.value = true
}

function showEditStorage(row: MerchantStorageResp) {
  storageIsEdit.value = true
  storageEditId.value = row.id
  Object.assign(storageForm, {
    storage_type: row.storage_type, name: row.name, endpoint: row.endpoint,
    bucket: row.bucket, region: row.region, access_key_id: row.access_key_id,
    access_key_secret: "", upload_url: row.upload_url, download_url: row.download_url,
    file_base_url: row.file_base_url, bucket_url: row.bucket_url,
    custom_domain: row.custom_domain, is_default: row.is_default, status: row.status
  })
  storageFormVisible.value = true
}

async function handleStorageSubmit() {
  if (!storageForm.name || !storageForm.bucket || !storageForm.access_key_id) {
    ElMessage.warning("请填写必填项：配置名称、Bucket、AccessKeyId")
    return
  }
  storageFormLoading.value = true
  try {
    const data = { ...storageForm, merchant_id: storageMerchant.id }
    if (storageIsEdit.value) {
      await updateMerchantStorage(storageEditId.value, data)
      ElMessage.success("更新成功")
    } else {
      await createMerchantStorage(data)
      ElMessage.success("创建成功")
    }
    storageFormVisible.value = false
    await loadStorageList()
  } catch (e: any) {
    ElMessage.error(e?.message || "操作失败")
  } finally {
    storageFormLoading.value = false
  }
}

function handleStorageDelete(row: MerchantStorageResp) {
  ElMessageBox.confirm(`确定删除配置「${row.name}」吗？`, "删除确认", { type: "warning" })
    .then(() => deleteMerchantStorage(row.id))
    .then(() => { ElMessage.success("删除成功"); loadStorageList() })
    .catch(() => {})
}

// 推送
const pushDialogVisible = ref(false)
const pushLoading = ref(false)
const pushTarget = reactive({ config_id: 0, config_name: "", twofa_code: "" })

function showPushDialog(row: MerchantStorageResp) {
  pushTarget.config_id = row.id
  pushTarget.config_name = row.name
  pushTarget.twofa_code = ""
  pushDialogVisible.value = true
}

async function handlePush() {
  if (!pushTarget.twofa_code || pushTarget.twofa_code.length !== 6) {
    ElMessage.warning("请输入6位2FA验证码")
    return
  }
  pushLoading.value = true
  try {
    const res = await pushMerchantStorage({
      merchant_id: storageMerchant.id,
      config_id: pushTarget.config_id,
      twofa_code: pushTarget.twofa_code
    })
    if (res.data.success) {
      ElMessage.success("推送成功")
      pushDialogVisible.value = false
      await loadStorageList()
    } else {
      ElMessage.error(res.data.message || "推送失败")
    }
  } catch (e: any) {
    ElMessage.error(e?.message || "推送失败")
  } finally {
    pushLoading.value = false
  }
}
</script>

<template>
  <div class="app-container">
    <!-- 隧道统计卡片 -->
    <el-row :gutter="20" class="stats-row">
      <el-col :xs="24" :sm="8" :md="8" :lg="8">
        <el-card v-loading="statsLoading" shadow="hover" class="stats-card">
          <div class="stats-content">
            <div class="stats-icon merchants">
              <el-icon :size="28"><Shop /></el-icon>
            </div>
            <div class="stats-info">
              <div class="stats-value">{{ tunnelStats.total_merchants }}</div>
              <div class="stats-label">有效商户</div>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :xs="24" :sm="8" :md="8" :lg="8">
        <el-card v-loading="statsLoading" shadow="hover" class="stats-card">
          <div class="stats-content">
            <div class="stats-icon gost-servers">
              <el-icon :size="28"><Connection /></el-icon>
            </div>
            <div class="stats-info">
              <div class="stats-value">{{ tunnelStats.total_gost_servers }}</div>
              <div class="stats-label">系统服务器(GOST)</div>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :xs="24" :sm="8" :md="8" :lg="8">
        <el-card v-loading="statsLoading" shadow="hover" class="stats-card">
          <div class="stats-content">
            <div class="stats-icon merchant-servers">
              <el-icon :size="28"><Monitor /></el-icon>
            </div>
            <div class="stats-info">
              <div class="stats-value">{{ tunnelStats.total_merchant_servers }}</div>
              <div class="stats-label">商户服务器</div>
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- 搜索区域 -->
    <el-card v-loading="loading" shadow="never" class="search-wrapper">
      <el-form ref="searchFormRef" :inline="true" :model="searchData">
        <el-form-item prop="merchant_no" label="企业号">
          <el-input v-model="searchData.merchant_no" placeholder="请输入企业号" clearable style="width: 200px;" />
        </el-form-item>
        <el-form-item prop="name" label="商户名称">
          <el-input v-model="searchData.name" placeholder="请输入商户名称" clearable style="width: 200px;" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :icon="Search" @click="handleSearch">
            查询
          </el-button>
          <el-button :icon="Refresh" @click="resetSearch">
            重置
          </el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <!-- 表格区域 -->
    <el-card v-loading="loading" shadow="never">
      <div class="toolbar-wrapper">
        <div>
          <el-button type="primary" :icon="CirclePlus" @click="$router.push({ name: 'MerchantCreate' })">
            新增商户（单机）
          </el-button>
          <el-button type="warning" :icon="Monitor" @click="$router.push({ name: 'MerchantCreateCluster' })">
            新增商户（集群）
          </el-button>
          <el-button type="primary" plain style="margin-left: 8px;" @click="openBatchSmsConfig">
            批量修改短信配置
          </el-button>
          <el-button type="primary" plain style="margin-left: 8px;" @click="openBatchNicknameDialog">
            批量修改系统账号
          </el-button>
          <el-button type="primary" plain style="margin-left: 8px;" @click="openBatchSensitiveDialog">
            批量修改敏感词
          </el-button>
          <el-button type="primary" plain style="margin-left: 8px;" :loading="pushLogoLoading" @click="handleBatchPushLogo">
            批量推送Logo
          </el-button>
        </div>
        <div>
          <el-button type="primary" :icon="Refresh" circle @click="refreshList" />
        </div>
      </div>

      <div class="table-wrapper">
        <el-table :data="tableData" border style="width: 100%">
          <el-table-column prop="id" label="ID" width="80" align="center" />
          <el-table-column label="应用信息" min-width="200" align="center">
            <template #default="{ row }">
              <div class="app-info-cell">
                <el-popover
                  :visible="logoUploadVisible[row.id]"
                  placement="right"
                  :width="280"
                  trigger="click"
                >
                  <template #reference>
                    <div class="logo-wrapper" @click="openLogoUpload(row)">
                      <el-image
                        v-if="row.logo_url || row.icon_url"
                        :src="row.logo_url || row.icon_url"
                        fit="contain"
                        class="app-logo clickable"
                      />
                      <el-avatar v-else :size="40" shape="square" class="clickable">
                        <span>{{ (row.app_name || row.name || '').charAt(0) }}</span>
                      </el-avatar>
                    </div>
                  </template>
                  <div class="logo-upload-popover">
                    <div class="popover-header">
                      <span>上传 Logo</span>
                      <el-button link type="primary" size="small" @click="logoUploadVisible[row.id] = false">关闭</el-button>
                    </div>
                    <ImageUploader
                      :model-value="row.logo_url || ''"
                      :width="100"
                      :height="100"
                      asset-type="logo"
                      @update:model-value="handleLogoUpdate(row, $event)"
                    />
                  </div>
                </el-popover>
                <span class="app-name">{{ row.app_name || row.name || '-' }}</span>
              </div>
            </template>
          </el-table-column>
          <el-table-column prop="name" label="商户名称" min-width="150" align="center" />
          <el-table-column label="服务器IP" min-width="180" align="center">
            <template #default="{ row }">
              <div>{{ row.server_ip }}</div>
              <el-tag v-if="row.deploy_mode === 'cluster'" type="warning" size="small" class="mt-1 cursor-pointer" @click="openServiceNodes(row)">
                多机
              </el-tag>
              <el-tag v-else size="small" class="mt-1 cursor-pointer" @click="openServiceNodes(row)">
                单机
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="port" label="端口" width="120" align="center" />
          <el-table-column prop="no" label="企业号" min-width="200" align="center" />
          <!-- <el-table-column label="商户状态" width="100" align="center">
            <template #default="{ row }">
              <el-tag :type="row.status === 1 ? 'success' : 'danger'" effect="plain">
                {{ row.status === 1 ? '正常' : '禁用' }}
              </el-tag>
            </template>
          </el-table-column> -->
          <el-table-column label="过期时间" min-width="180" align="center">
            <template #default="{ row }">
              <el-tag :type="row.expiring_soon === 0 ? 'success' : row.expiring_soon === 1 ? 'warning' : 'danger'" effect="plain">
                {{ row.expired_at }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="套餐配置" min-width="250" align="center">
            <template #default="{ row }">
              <div v-if="row.package_configuration">
                <div>日活: {{ row.package_configuration.dau_limit }}</div>
                <div>注册: {{ row.package_configuration.register_limit }}</div>
                <div>群人数: {{ row.package_configuration.group_member_limit }}</div>
                <div v-if="row.package_configuration.app_packages && row.package_configuration.app_packages.length > 0">
                  <el-tag
                    v-for="pkg in row.package_configuration.app_packages.slice(0, 2)"
                    :key="pkg"
                    size="small"
                    type="info"
                    style="margin: 2px;"
                  >
                    {{ pkg.split('.').pop() }}
                  </el-tag>
                  <el-tag v-if="row.package_configuration.app_packages.length > 2" size="small" type="info">
                    +{{ row.package_configuration.app_packages.length - 2 }}
                  </el-tag>
                </div>
              </div>
              <span v-else>-</span>
            </template>
          </el-table-column>
          <el-table-column label="创建时间" width="180" align="center">
            <template #default="{ row }">
              {{ row.created_at }}
            </template>
          </el-table-column>
          <el-table-column fixed="right" label="操作" width="260" align="center">
            <template #default="{ row }">
              <div class="operation-wrapper">
                <el-button link type="primary" size="small" @click="openAdminmUsers(row)">后台账号</el-button>
                <el-button link type="primary" size="small" @click="openActiveDialog(row)">活跃数据</el-button>
                <el-dropdown trigger="click">
                  <el-button link type="primary" size="small">更多</el-button>
                  <template #dropdown>
                    <el-dropdown-menu>
                      <el-dropdown-item @click="showDetailDialog(row)">详情</el-dropdown-item>
                      <el-dropdown-item @click="openSmsConfig(row)">短信配置</el-dropdown-item>
                      <el-dropdown-item @click="openStorageDialog(row)">存储配置</el-dropdown-item>
                      <el-dropdown-item @click="openServiceNodes(row)">服务节点</el-dropdown-item>
                      <el-dropdown-item @click="openNicknameForRow(row)">系统账号资料</el-dropdown-item>
                      <el-dropdown-item divided @click="openTunnelDialog(row)">隧道连接检测</el-dropdown-item>
                      <el-dropdown-item @click="handleChangeGostPort(row)" :disabled="changeGostPortLoading">更换隧道端口</el-dropdown-item>
                      <el-dropdown-item @click="$router.push({ name: 'MerchantEdit', params: { id: row.id }, query: { data: JSON.stringify(row) } })">编辑</el-dropdown-item>
                      <el-dropdown-item @click="handleQuickBuild(row)">一键打包</el-dropdown-item>
                      <el-dropdown-item :disabled="pushLogoLoading" @click="handlePushLogo(row)">推送Logo到Web</el-dropdown-item>
                      <el-dropdown-item :disabled="exportDbLoading" @click="handleExportDatabase(row)">导出数据库</el-dropdown-item>
                      <el-dropdown-item divided class="danger-item" :disabled="clearDataLoading" @click="handleClearData(row)">清除数据</el-dropdown-item>
                      <el-dropdown-item class="danger-item" :disabled="changeIpLoading" @click="handleChangeIP(row)">更换IP</el-dropdown-item>
                      <el-dropdown-item class="danger-item" @click="handleDelete(row)">删除</el-dropdown-item>
                    </el-dropdown-menu>
                  </template>
                </el-dropdown>
              </div>
            </template>
          </el-table-column>
        </el-table>
      </div>

      <!-- 分页 -->
      <div class="pager-wrapper">
        <el-pagination
          background
          :layout="paginationData.layout"
          :page-sizes="paginationData.pageSizes"
          :total="paginationData.total"
          :page-size="paginationData.pageSize"
          :current-page="paginationData.currentPage"
          @size-change="handleSizeChange"
          @current-change="handleCurrentChange"
        />
      </div>
    </el-card>

    <!-- 商户表单对话框 -->
    <MerchantForm
      v-model:visible="dialogVisible"
      v-model:form-data="formData"
      :merchant="editingMerchantId ? tableData.find(item => item.id === editingMerchantId) || null : null"
      @submit="submitForm"
    />

    <!-- 商户详情对话框 -->
    <MerchantDetail
      v-model:visible="detailDialogVisible"
      :merchant="currentMerchant"
    />

    <!-- 服务节点管理弹窗 -->
    <ServiceNodesDialog
      v-model:visible="serviceNodesVisible"
      :merchant-id="serviceNodesMerchantId"
      :merchant-name="serviceNodesMerchantName"
      @updated="refreshList"
    />

    <!-- Adminm 用户管理弹窗 -->
    <AdminmUsersDialog v-model:visible="adminmUsersVisible" :merchant-no="adminmMerchantNo" />

    <!-- 活跃数据弹窗 -->
    <el-dialog v-model="activeDialogVisible" :title="`活跃数据 - ${activeData.merchant_no}`" width="400px">
      <el-descriptions :column="1" border>
        <el-descriptions-item label="总用户数">
          {{ activeData.total_users }}
        </el-descriptions-item>
        <el-descriptions-item label="在线用户数">
          {{ activeData.online_users }}
        </el-descriptions-item>
        <el-descriptions-item label="DAU（日活）">
          {{ activeData.dau }}
        </el-descriptions-item>
      </el-descriptions>
      <template #footer>
        <el-button @click="activeDialogVisible = false">关闭</el-button>
      </template>
    </el-dialog>

    <!-- 隧道连接检测弹窗 -->
    <el-dialog v-model="tunnelDialogVisible" :title="tunnelTarget.title" width="960px">
      <el-table :data="tunnelResults" v-loading="tunnelLoading" border style="width: 100%">
        <el-table-column prop="server_name" label="系统服务器" min-width="130" />
        <el-table-column prop="server_ip" label="IP" min-width="110" />
        <el-table-column label="端口探测" width="85" align="center">
          <template #default="{ row }">
            <el-tag :type="row.success ? 'success' : 'danger'" size="small" effect="dark">
              {{ row.success ? '连通' : '失败' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="HTTP隧道" width="85" align="center">
          <template #default="{ row }">
            <el-tag :type="row.e2e_success ? 'success' : 'danger'" size="small" effect="dark">
              {{ row.e2e_success ? '正常' : '异常' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="MinIO隧道" width="85" align="center">
          <template #default="{ row }">
            <el-tag :type="row.minio_e2e_success ? 'success' : 'danger'" size="small" effect="dark">
              {{ row.minio_e2e_success ? '正常' : '异常' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="详情" min-width="200" show-overflow-tooltip>
          <template #default="{ row }">
            <div>
              <span :style="{ color: row.e2e_success ? '' : 'var(--el-color-danger)' }">HTTP: {{ row.e2e_message }}</span>
            </div>
            <div>
              <span :style="{ color: row.minio_e2e_success ? '' : 'var(--el-color-danger)' }">MinIO: {{ row.minio_e2e_message }}</span>
            </div>
          </template>
        </el-table-column>
      </el-table>
      <div v-if="tunnelResults.length > 0 && tunnelResults.some(r => r.success && (!r.e2e_success || !r.minio_e2e_success))" class="tunnel-tip">
        <el-alert type="warning" :closable="false" show-icon>
          <template #title>端口连通但隧道握手异常，可能原因：TLS证书不匹配、GOST未启动、relay配置错误、或MinIO节点不可达</template>
        </el-alert>
      </div>
      <template #footer>
        <el-button @click="tunnelDialogVisible = false">关闭</el-button>
      </template>
    </el-dialog>

    <!-- 商户配置（短信） -->
    <el-dialog v-model="configDialogVisible" :title="isBatchConfig ? '批量修改短信配置' : `短信配置 - ${configMerchantNo}`" width="720px">
      <template v-if="isBatchConfig">
        <el-alert type="info" show-icon class="mb-2" title="请选择更新范围" :closable="false" />
        <el-form label-width="120px" class="mb-3">
          <el-form-item label="更新范围">
            <el-radio-group v-model="batchTarget.mode">
              <el-radio label="broadcast">全部商户</el-radio>
              <el-radio label="merchant_nos">指定企业号</el-radio>
            </el-radio-group>
          </el-form-item>
          <el-form-item v-if="batchTarget.mode === 'merchant_nos'" label="选择商户">
            <el-select
              v-model="batchTarget.selectedNos"
              multiple
              filterable
              remote
              reserve-keyword
              placeholder="搜索商户名称或企业号"
              :remote-method="fetchBatchMerchants"
              :loading="batchMerchantLoading"
              style="width: 100%;"
            >
              <el-option
                v-for="opt in batchMerchantOptions"
                :key="opt.value"
                :label="opt.label"
                :value="opt.value"
              />
            </el-select>
          </el-form-item>
        </el-form>
      </template>
      <el-form label-width="120px" v-loading="configLoading">
        <el-form-item>
          <template #label>
            <span>从模板导入</span>
            <el-text type="info" size="small" style="margin-left: 4px;">（可选）</el-text>
          </template>
          <el-select
            v-model="selectedSmsClientId"
            filterable
            clearable
            placeholder="选择已有客户端配置快速填充"
            :loading="clientLoading"
            style="width: 100%;"
            @change="onSelectSmsClient"
          >
            <el-option v-for="opt in clientOptions" :key="opt.value" :label="opt.label" :value="opt.value" />
          </el-select>
        </el-form-item>
        <el-form-item label="短信通道">
          <el-select v-model="smsConfigForm.provider" placeholder="选择短信通道" style="width: 100%;">
            <el-option label="阿里云 (Aliyun)" value="aliyun" />
            <el-option label="联合短信 (UniSMS)" value="unisms" />
            <el-option label="短信宝 (SmsBao)" value="smsbao" />
          </el-select>
        </el-form-item>
        <!-- 阿里云配置 -->
        <template v-if="smsConfigForm.provider === 'aliyun' || !smsConfigForm.provider">
          <el-form-item label="RegionId">
            <el-input v-model="smsConfigForm.region_id" placeholder="如：cn-hangzhou" clearable />
          </el-form-item>
          <el-form-item label="AccessKey">
            <el-input v-model="smsConfigForm.access_key" placeholder="阿里云 AccessKeyID" clearable />
          </el-form-item>
          <el-form-item label="SecretKey">
            <el-input v-model="smsConfigForm.secret_key" placeholder="阿里云 AccessSecret" clearable show-password />
          </el-form-item>
          <el-form-item label="签名">
            <el-input v-model="smsConfigForm.sign_name" placeholder="短信签名" clearable />
          </el-form-item>
          <el-form-item label="模板Code">
            <el-input v-model="smsConfigForm.template_code" placeholder="短信模板代码" clearable />
          </el-form-item>
        </template>
        <!-- UniSMS 配置 -->
        <template v-if="smsConfigForm.provider === 'unisms'">
          <el-form-item label="AccessKeyID">
            <el-input v-model="smsConfigForm.unisms_access_key_id" placeholder="UniSMS AccessKeyID" clearable />
          </el-form-item>
          <el-form-item label="AccessKeySecret">
            <el-input v-model="smsConfigForm.unisms_access_key_secret" placeholder="UniSMS AccessKeySecret（可选）" clearable show-password />
          </el-form-item>
          <el-form-item label="签名">
            <el-input v-model="smsConfigForm.unisms_signature" placeholder="UniSMS 签名" clearable />
          </el-form-item>
          <el-form-item label="模板ID">
            <el-input v-model="smsConfigForm.unisms_template_id" placeholder="UniSMS 模板ID" clearable />
          </el-form-item>
        </template>
        <!-- 短信宝配置 -->
        <template v-if="smsConfigForm.provider === 'smsbao'">
          <el-form-item label="账号">
            <el-input v-model="smsConfigForm.smsbao_account" placeholder="短信宝账号" clearable />
          </el-form-item>
          <el-form-item label="密码">
            <el-input v-model="smsConfigForm.smsbao_api_key" placeholder="短信宝登录密码（原文）" clearable show-password />
          </el-form-item>
          <el-form-item label="模板">
            <el-input v-model="smsConfigForm.smsbao_template" type="textarea" :rows="2" placeholder="模板内容，用 {code} 作为验证码占位符" clearable />
          </el-form-item>
        </template>
        <el-form-item>
          <el-button type="primary" @click="handleSaveSms">保存</el-button>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="configDialogVisible = false">关闭</el-button>
      </template>
    </el-dialog>

    <!-- 修改系统账号资料（昵称+头像） -->
    <el-dialog v-model="nicknameDialogVisible" :title="isNicknameSingle ? '修改系统账号资料' : '批量修改系统账号资料'" width="520px">
      <el-form label-width="120px">
        <el-form-item v-if="!isNicknameSingle" label="更新范围">
          <el-radio-group v-model="nicknameTarget.mode">
            <el-radio label="broadcast">全部商户</el-radio>
            <el-radio label="merchant_nos">指定商户</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item v-if="isNicknameSingle" label="目标商户">
          <span>{{ nicknameSingleMerchantLabel }}</span>
        </el-form-item>
        <el-form-item v-else-if="nicknameTarget.mode === 'merchant_nos'" label="选择商户">
          <el-select
            v-model="nicknameTarget.selectedNos"
            multiple filterable remote reserve-keyword
            placeholder="搜索商户名称或企业号"
            :remote-method="fetchBatchMerchants"
            :loading="batchMerchantLoading"
            style="width: 100%;"
          >
            <el-option v-for="opt in batchMerchantOptions" :key="opt.value" :label="opt.label" :value="opt.value" />
          </el-select>
        </el-form-item>
        <el-form-item label="系统昵称">
          <el-input v-model="nicknameValue" placeholder="请输入新的系统昵称（留空则不修改）" maxlength="64" show-word-limit />
        </el-form-item>
        <el-form-item label="系统头像URL">
          <el-input v-model="avatarUrlValue" placeholder="请输入头像图片URL（留空则不修改）" maxlength="500" />
          <div v-if="avatarUrlValue" style="margin-top: 8px;">
            <el-image :src="avatarUrlValue" style="width: 64px; height: 64px; border-radius: 50%;" fit="cover">
              <template #error>
                <div style="width: 64px; height: 64px; border-radius: 50%; background: #f5f7fa; display: flex; align-items: center; justify-content: center; color: #909399; font-size: 12px;">
                  预览失败
                </div>
              </template>
            </el-image>
          </div>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="handleSaveNickname">保存</el-button>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="nicknameDialogVisible = false">关闭</el-button>
      </template>
    </el-dialog>

    <!-- 批量修改敏感词 -->
    <el-dialog v-model="sensitiveDialogVisible" title="批量修改敏感词" width="720px">
      <el-form label-width="120px">
        <el-form-item label="更新范围">
          <el-radio-group v-model="sensitiveTarget.mode">
            <el-radio label="broadcast">全部商户</el-radio>
            <el-radio label="merchant_nos">指定商户</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item v-if="sensitiveTarget.mode === 'merchant_nos'" label="选择商户">
          <el-select
            v-model="sensitiveTarget.selectedNos"
            multiple filterable remote reserve-keyword
            placeholder="搜索商户名称或企业号"
            :remote-method="fetchBatchMerchants"
            :loading="batchMerchantLoading"
            style="width: 100%;"
          >
            <el-option v-for="opt in batchMerchantOptions" :key="opt.value" :label="opt.label" :value="opt.value" />
          </el-select>
        </el-form-item>
        <el-form-item label="敏感词TXT">
          <el-input
            v-model="sensitiveTxt"
            type="textarea"
            :autosize="{ minRows: 8 }"
            placeholder="每行一条，英文逗号分隔：word,tip ；支持以 # 或 // 开头的注释行"
          />
        </el-form-item>
        <el-form-item label="从文件读取">
          <el-button @click="triggerSensitiveFileSelect">选择TXT文件</el-button>
          <input ref="sensitiveFileInputRef" type="file" accept=".txt,text/plain" style="display:none" @change="onSensitiveFileChange" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="handleSaveSensitive">保存</el-button>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="sensitiveDialogVisible = false">关闭</el-button>
      </template>
    </el-dialog>

    <!-- 存储配置弹窗 -->
    <el-dialog v-model="storageDialogVisible" :title="`存储配置 - ${storageMerchant.name}`" width="860px">
      <div style="margin-bottom: 12px; display: flex; justify-content: flex-end;">
        <el-button type="primary" size="small" @click="showAddStorage">新增配置</el-button>
      </div>
      <el-table :data="storageList" v-loading="storageLoading" border size="small">
        <el-table-column prop="name" label="配置名称" width="120" />
        <el-table-column label="存储类型" width="110">
          <template #default="{ row }">
            <el-tag :type="getStorageTypeTagType(row.storage_type)" size="small">
              {{ getStorageTypeLabel(row.storage_type) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="bucket" label="Bucket" width="130" show-overflow-tooltip />
        <el-table-column prop="endpoint" label="Endpoint" show-overflow-tooltip />
        <el-table-column label="默认" width="60" align="center">
          <template #default="{ row }">
            <el-tag v-if="row.is_default === 1" type="success" size="small">是</el-tag>
            <span v-else>-</span>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="60" align="center">
          <template #default="{ row }">
            <el-tag :type="row.status === 1 ? 'success' : 'danger'" size="small">
              {{ row.status === 1 ? '启用' : '禁用' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="推送" width="80" align="center">
          <template #default="{ row }">
            <el-tag v-if="row.last_push_result === '成功'" type="success" size="small">成功</el-tag>
            <el-tooltip v-else-if="row.last_push_result" :content="row.last_push_result" placement="top">
              <el-tag type="danger" size="small">失败</el-tag>
            </el-tooltip>
            <span v-else class="text-gray">-</span>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="160" align="center" fixed="right">
          <template #default="{ row }">
            <el-button type="primary" link size="small" @click="showEditStorage(row)">编辑</el-button>
            <el-button type="success" link size="small" @click="showPushDialog(row)">推送</el-button>
            <el-button type="danger" link size="small" @click="handleStorageDelete(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
      <el-empty v-if="!storageLoading && storageList.length === 0" description="暂无存储配置，点击「新增配置」添加" />
      <template #footer>
        <el-button @click="storageDialogVisible = false">关闭</el-button>
      </template>
    </el-dialog>

    <!-- 存储配置编辑弹窗 -->
    <el-dialog v-model="storageFormVisible" :title="storageIsEdit ? '编辑存储配置' : '新增存储配置'" width="600px" append-to-body>
      <el-form :model="storageForm" label-width="120px" v-loading="storageFormLoading">
        <el-form-item label="存储类型" required>
          <el-select v-model="storageForm.storage_type" style="width: 100%;">
            <el-option v-for="opt in storageTypeOptions" :key="opt.value" :label="opt.label" :value="opt.value" />
          </el-select>
        </el-form-item>
        <el-form-item label="配置名称" required>
          <el-input v-model="storageForm.name" placeholder="请输入配置名称" />
        </el-form-item>
        <el-form-item label="服务端点">
          <el-input v-model="storageForm.endpoint" placeholder="如: http://minio:9000" />
        </el-form-item>
        <el-form-item label="Bucket" required>
          <el-input v-model="storageForm.bucket" placeholder="请输入 Bucket 名称" />
        </el-form-item>
        <el-form-item v-if="['aws_s3', 'tencent_cos'].includes(storageForm.storage_type)" label="区域">
          <el-input v-model="storageForm.region" placeholder="如: us-east-1, ap-guangzhou" />
        </el-form-item>
        <el-form-item label="AccessKeyId" required>
          <el-input v-model="storageForm.access_key_id" placeholder="请输入 AccessKeyId" />
        </el-form-item>
        <el-form-item label="AccessKeySecret">
          <el-input v-model="storageForm.access_key_secret" type="password" show-password :placeholder="storageIsEdit ? '留空表示不修改' : '请输入 AccessKeySecret'" />
        </el-form-item>
        <template v-if="storageForm.storage_type === 'minio'">
          <el-form-item label="上传URL">
            <el-input v-model="storageForm.upload_url" placeholder="可选，留空使用 Endpoint" />
          </el-form-item>
          <el-form-item label="下载URL">
            <el-input v-model="storageForm.download_url" placeholder="可选，留空使用 Endpoint" />
          </el-form-item>
        </template>
        <el-form-item v-if="['minio', 'aws_s3', 'tencent_cos'].includes(storageForm.storage_type)" label="文件基础URL">
          <el-input v-model="storageForm.file_base_url" placeholder="文件访问的基础URL" />
        </el-form-item>
        <el-form-item v-if="storageForm.storage_type === 'aliyunOSS'" label="Bucket URL">
          <el-input v-model="storageForm.bucket_url" placeholder="如: https://bucket.oss-cn-hangzhou.aliyuncs.com" />
        </el-form-item>
        <el-form-item label="自定义域名">
          <el-input v-model="storageForm.custom_domain" placeholder="CDN 自定义域名（可选）" />
        </el-form-item>
        <el-form-item label="设为默认">
          <el-switch v-model="storageForm.is_default" :active-value="1" :inactive-value="0" />
        </el-form-item>
        <el-form-item label="状态">
          <el-switch v-model="storageForm.status" :active-value="1" :inactive-value="0" active-text="启用" inactive-text="禁用" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="storageFormVisible = false">取消</el-button>
        <el-button type="primary" :loading="storageFormLoading" @click="handleStorageSubmit">确定</el-button>
      </template>
    </el-dialog>

    <!-- 推送存储配置确认弹窗 -->
    <el-dialog v-model="pushDialogVisible" title="推送存储配置" width="400px" append-to-body>
      <p>即将推送配置「{{ pushTarget.config_name }}」到商户 {{ storageMerchant.name }} 的服务器</p>
      <p style="color: #909399; font-size: 13px; margin: 10px 0;">推送后将立即生效，请确保配置正确</p>
      <el-input v-model="pushTarget.twofa_code" placeholder="请输入6位2FA验证码" maxlength="6" style="margin-top: 15px;" />
      <template #footer>
        <el-button @click="pushDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="pushLoading" @click="handlePush">确认推送</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style lang="scss" scoped>
.tunnel-tip {
  margin-top: 12px;
}

.stats-row {
  margin-bottom: 20px;
}

.stats-card {
  :deep(.el-card__body) {
    padding: 20px;
  }
}

.stats-content {
  display: flex;
  align-items: center;
  gap: 16px;
}

.stats-icon {
  width: 56px;
  height: 56px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;

  &.merchants {
    background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  }

  &.gost-servers {
    background: linear-gradient(135deg, #11998e 0%, #38ef7d 100%);
  }

  &.merchant-servers {
    background: linear-gradient(135deg, #f093fb 0%, #f5576c 100%);
  }
}

.stats-info {
  flex: 1;
}

.stats-value {
  font-size: 28px;
  font-weight: 600;
  color: #303133;
  line-height: 1.2;
}

.stats-label {
  font-size: 14px;
  color: #909399;
  margin-top: 4px;
}

.toolbar-wrapper {
  display: flex;
  justify-content: space-between;
  margin-bottom: 20px;
}

.table-wrapper {
  margin-bottom: 20px;
}

.pager-wrapper {
  display: flex;
  justify-content: flex-end;
}

.search-wrapper {
  margin-bottom: 20px;
}

.search-wrapper :deep(.el-form-item) {
  margin-bottom: 10px;
  margin-right: 15px;
}

.search-wrapper :deep(.el-select) {
  width: 100%;
}

.region-tag,
.ip-tag {
  margin-right: 5px;
  margin-bottom: 5px;
}

.node-count-wrapper {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 5px;
}

.online-dot {
  display: inline-block;
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background-color: #67c23a;
  box-shadow: 0 0 4px #67c23a;
}

/* 操作按钮包装器 */
.operation-wrapper {
  display: flex;
  justify-content: center;
  align-items: center;
  gap: 8px; /* 按钮之间的间距 */
}

/* 删除菜单项样式 */
.danger-item {
  color: #f56c6c;
}

.balance-low {
  color: #f56c6c;
  font-weight: bold;
}

.awaiting-amount {
  color: #e6a23c;
  font-size: 12px;
}

.balance-container {
  display: flex;
  align-items: center;
  justify-content: center;
  white-space: nowrap;
}

.app-info-cell {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
}

.app-logo {
  width: 40px;
  height: 40px;
  border-radius: 8px;
  flex-shrink: 0;

  &.clickable {
    cursor: pointer;
    transition: all 0.2s;

    &:hover {
      transform: scale(1.05);
      box-shadow: 0 2px 8px rgba(0, 0, 0, 0.15);
    }
  }
}

.logo-wrapper {
  cursor: pointer;
}

.clickable {
  cursor: pointer;
  transition: all 0.2s;

  &:hover {
    opacity: 0.8;
  }
}

.logo-upload-popover {
  .popover-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 12px;
    font-weight: 500;
  }
}

.app-name {
  font-weight: 500;
  color: #303133;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 100px;
}
</style>
