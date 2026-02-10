<script lang="ts" setup>
import type { DeployGostServerReq, InstallGostReq, ServerResp } from "@@/apis/deploy/type"
import type { MerchantResp } from "@@/apis/merchant/type"
import type { Instance } from "@/pages/cloud/aliyun/instances/apis/type"
import type { VxeFormInstance, VxeFormProps, VxeGridInstance, VxeGridProps, VxeModalInstance, VxeModalProps } from "vxe-table"
import { createGostServiceByAPI, deleteGostServiceByAPI, getGostServiceDetail, getServerList, listGostChains, listGostServices, updateGostServiceDetail } from "@@/apis/deploy"
import { getMerchantList } from "@@/apis/merchant"
import { getCloudAccountList } from "@@/apis/cloud_account"
import { getInstanceList } from "@/pages/cloud/aliyun/instances/apis"
import { createStreamRequest } from "@/http/axios"

defineOptions({
  name: "GostService"
})

// ========== GOST 一键部署 ==========
const deployDialogVisible = ref(false)
const installDialogVisible = ref(false)
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

// 安装表单（在已有服务器上安装）
const installForm = reactive<InstallGostReq>({
  server_id: 0,
  host: "",
  port: 22,
  username: "root",
  password: "",
  private_key: ""
})

// 安装模式: server=从已有服务器选择, cloud=从云实例选择, manual=手动填写
const installMode = ref<"server" | "cloud" | "manual">("server")

// 云实例相关
const cloudInstanceList = ref<Instance[]>([])
const cloudInstanceLoading = ref(false)
const selectedCloudAccountId = ref<number>(0)
const selectedRegionId = ref<string>("cn-hongkong")

// 获取云实例公网IP
function getInstancePublicIP(instance: Instance): string {
  // 优先使用 EIP
  if (instance.EipAddress?.IpAddress) {
    return instance.EipAddress.IpAddress
  }
  // 其次使用 PublicIpAddress
  if (instance.PublicIpAddress?.IpAddress?.length > 0) {
    return instance.PublicIpAddress.IpAddress[0]
  }
  // 最后检查网卡的公网IP
  const nics = instance.NetworkInterfaces?.NetworkInterface || []
  for (const nic of nics) {
    const ipSets = nic.PrivateIpSets?.PrivateIpSet || []
    for (const ipSet of ipSets) {
      if (ipSet.AssociatedPublicIp?.PublicIpAddress) {
        return ipSet.AssociatedPublicIp.PublicIpAddress
      }
    }
  }
  return ""
}

// 加载云实例列表
async function loadCloudInstances() {
  if (!selectedCloudAccountId.value) {
    ElMessage.warning("请先选择云账号")
    return
  }
  cloudInstanceLoading.value = true
  try {
    const res = await getInstanceList({
      page: 1,
      size: 100,
      cloud_account_id: selectedCloudAccountId.value,
      region_id: selectedRegionId.value
    } as any)
    cloudInstanceList.value = Array.isArray(res.data?.list) ? res.data.list : []
  } catch (e) {
    console.error("加载云实例失败:", e)
  } finally {
    cloudInstanceLoading.value = false
  }
}

// 选择云实例时自动填充IP
function onSelectCloudInstance(instanceId: string) {
  const instance = cloudInstanceList.value.find(i => i.InstanceId === instanceId)
  if (instance) {
    const publicIP = getInstancePublicIP(instance)
    if (publicIP) {
      installForm.host = publicIP
      installForm.server_id = 0 // 清除服务器选择
    } else {
      ElMessage.warning("该实例没有公网IP")
    }
  }
}

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

// 打开安装弹窗
function openInstallDialog() {
  deployLogs.value = []
  installForm.server_id = selectedServerId.value || 0
  installDialogVisible.value = true
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

// 执行安装 GOST
function startInstall() {
  if (!installForm.server_id && !installForm.host) {
    ElMessage.warning("请选择服务器或填写IP地址")
    return
  }

  isDeploying.value = true
  deployLogs.value = ["开始安装 GOST..."]

  const cancel = createStreamRequest(
    {
      url: "deploy/gost/install",
      method: "POST",
      data: installForm
    },
    (data: any, isComplete?: boolean) => {
      if (data?.message) {
        deployLogs.value.push(data.message)
        scrollToBottom()
      }
      if (isComplete) {
        isDeploying.value = false
        if (data?.success) {
          ElMessage.success("安装完成!")
          loadServerList()
        }
      }
    },
    (error: any) => {
      isDeploying.value = false
      deployLogs.value.push(`错误: ${error?.message || error}`)
      ElMessage.error("安装失败")
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

// 阿里云地区列表（用于云实例选择）
const aliyunRegionOptions = [
  { value: "cn-hongkong", label: "香港" },
  { value: "cn-shanghai", label: "上海" },
  { value: "cn-hangzhou", label: "杭州" },
  { value: "cn-shenzhen", label: "深圳" },
  { value: "cn-beijing", label: "北京" },
  { value: "ap-southeast-1", label: "新加坡" },
  { value: "ap-northeast-1", label: "东京" },
  { value: "ap-southeast-5", label: "雅加达" },
  { value: "us-west-1", label: "美西(硅谷)" },
  { value: "us-east-1", label: "美东(弗吉尼亚)" },
  { value: "eu-central-1", label: "德国(法兰克福)" }
]

// 腾讯云地区列表
const tencentRegionOptions = [
  { value: "ap-hongkong", label: "香港" },
  { value: "ap-singapore", label: "新加坡" },
  { value: "ap-tokyo", label: "东京" },
  { value: "ap-seoul", label: "首尔" },
  { value: "ap-bangkok", label: "曼谷" },
  { value: "na-siliconvalley", label: "美西(硅谷)" },
  { value: "na-ashburn", label: "美东(弗吉尼亚)" },
  { value: "eu-frankfurt", label: "德国(法兰克福)" }
]

// 云供应商选项
const cloudTypeOptions = [
  { value: "aliyun", label: "阿里云" },
  { value: "aws", label: "AWS" },
  { value: "tencent", label: "腾讯云" }
]
const selectedCloudType = ref<string>("aliyun")

// 根据选择的云供应商筛选云账号
const filteredCloudAccountList = computed(() => {
  return (cloudAccountList.value || []).filter(acc => acc.cloud_type === selectedCloudType.value)
})

// 根据选择的云供应商获取对应的区域选项
const currentRegionOptions = computed(() => {
  switch (selectedCloudType.value) {
    case "aws":
      return regionOptions
    case "tencent":
      return tencentRegionOptions
    default:
      return aliyunRegionOptions
  }
})

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
  } catch (error) {
    console.error("查询列表失败:", error)
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
    } catch (error) {
      console.error("获取详情失败:", error)
      xFormOpt.data = { name: row.name, jsonText: "" }
    }
    xModalDom.value?.open()
    nextTick(() => {
      xFormDom.value?.clearValidate()
    })
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
        <el-button type="success" icon="Plus" @click="openDeployDialog">一键部署 GOST 服务器</el-button>
        <el-button type="warning" icon="Download" @click="openInstallDialog">在已有服务器安装</el-button>
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

    <!-- 在已有服务器安装 GOST 弹窗 -->
    <el-dialog v-model="installDialogVisible" title="在已有服务器安装 GOST" width="750px" :close-on-click-modal="false">
      <el-form :model="installForm" label-width="120px">
        <!-- 安装模式选择 -->
        <el-form-item label="服务器来源">
          <el-radio-group v-model="installMode" @change="() => { installForm.server_id = 0; installForm.host = '' }">
            <el-radio value="server">已有服务器</el-radio>
            <el-radio value="cloud">云实例</el-radio>
            <el-radio value="manual">手动填写</el-radio>
          </el-radio-group>
        </el-form-item>

        <!-- 模式1: 从已有服务器选择 -->
        <template v-if="installMode === 'server'">
          <el-form-item label="选择服务器">
            <el-select v-model="installForm.server_id" placeholder="选择已有服务器" style="width: 100%" filterable>
              <el-option v-for="s in allServerList" :key="s.id" :label="`${s.name} (${s.host})`" :value="s.id" />
            </el-select>
          </el-form-item>
        </template>

        <!-- 模式2: 从云实例选择 -->
        <template v-if="installMode === 'cloud'">
          <el-form-item label="云供应商">
            <el-radio-group v-model="selectedCloudType" @change="() => { selectedCloudAccountId = 0; cloudInstanceList = [] }">
              <el-radio-button v-for="opt in cloudTypeOptions" :key="opt.value" :value="opt.value">{{ opt.label }}</el-radio-button>
            </el-radio-group>
          </el-form-item>
          <el-form-item label="云账号">
            <el-select v-model="selectedCloudAccountId" placeholder="选择云账号" style="width: 100%" @change="cloudInstanceList = []">
              <el-option v-for="acc in filteredCloudAccountList" :key="acc.id" :label="acc.name" :value="acc.id" />
            </el-select>
          </el-form-item>
          <el-form-item label="区域">
            <div class="flex items-center gap-2" style="width: 100%">
              <el-select v-model="selectedRegionId" placeholder="选择区域" style="flex: 1">
                <el-option v-for="r in currentRegionOptions" :key="r.value" :label="r.label" :value="r.value" />
              </el-select>
              <el-button type="primary" @click="loadCloudInstances" :loading="cloudInstanceLoading">
                加载实例
              </el-button>
            </div>
          </el-form-item>
          <el-form-item label="选择实例">
            <el-select
              v-model="installForm.host"
              placeholder="选择云实例"
              style="width: 100%"
              filterable
              :loading="cloudInstanceLoading"
              @change="(val: string) => onSelectCloudInstance(cloudInstanceList.find(i => getInstancePublicIP(i) === val)?.InstanceId || '')"
            >
              <el-option
                v-for="inst in cloudInstanceList"
                :key="inst.InstanceId"
                :label="`${inst.InstanceName} (${getInstancePublicIP(inst) || '无公网IP'})`"
                :value="getInstancePublicIP(inst)"
                :disabled="!getInstancePublicIP(inst)"
              >
                <div class="flex items-center justify-between">
                  <span>{{ inst.InstanceName }}</span>
                  <span class="text-sm text-gray-400">{{ getInstancePublicIP(inst) || '无公网IP' }}</span>
                </div>
              </el-option>
            </el-select>
          </el-form-item>
          <el-form-item label="SSH密码">
            <el-input v-model="installForm.password" type="password" placeholder="实例的SSH密码" show-password />
          </el-form-item>
        </template>

        <!-- 模式3: 手动填写 -->
        <template v-if="installMode === 'manual'">
          <el-form-item label="服务器IP">
            <el-input v-model="installForm.host" placeholder="例如: 1.2.3.4" />
          </el-form-item>
          <el-form-item label="SSH端口">
            <el-input-number v-model="installForm.port" :min="1" :max="65535" />
          </el-form-item>
          <el-form-item label="用户名">
            <el-input v-model="installForm.username" placeholder="默认 root" />
          </el-form-item>
          <el-form-item label="SSH密码">
            <el-input v-model="installForm.password" type="password" placeholder="SSH密码" show-password />
          </el-form-item>
        </template>
      </el-form>

      <!-- 安装日志 -->
      <div v-if="deployLogs.length > 0" class="deploy-log-container">
        <div class="deploy-log-title">安装日志:</div>
        <div ref="deployLogRef" class="deploy-log-content">
          <div v-for="(log, idx) in deployLogs" :key="idx" class="deploy-log-line">{{ log }}</div>
        </div>
      </div>

      <template #footer>
        <el-button @click="installDialogVisible = false" :disabled="isDeploying">取消</el-button>
        <el-button type="primary" @click="startInstall" :loading="isDeploying">
          {{ isDeploying ? '安装中...' : '开始安装' }}
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
