<script lang="ts" setup>
import type { DistributeResult, ServerResp } from "@@/apis/deploy/type"
import type { MerchantResp } from "@@/apis/merchant/type"
import type { VxeFormInstance, VxeFormProps, VxeGridInstance, VxeGridProps, VxeModalInstance, VxeModalProps } from "vxe-table"
import { createServer, deleteServer, distributeFile, getServerList, getServerStatsBatch, testConnection, toggleServerStatus, updateServer, uploadToLocal, batchUpgradeTls, batchRollbackTls, getTlsStatus, verifyTlsStatus } from "@@/apis/deploy"
import type { TlsServerResult, TlsStatusResp } from "@@/apis/deploy/type"
import { getMerchantList } from "@@/apis/merchant"
import { getResourceGroups, createResourceGroup, updateResourceGroup, deleteResourceGroup } from "@@/apis/ip_embed"
import type { ResourceGroupItem } from "@@/apis/ip_embed/type"

defineOptions({
  name: "DeployServers"
})

// 服务器类型
const serverType = ref(1) // 1-商户服务器 2-系统服务器

// 商户选项（VXE表单筛选用）
const merchantOptions: { label: string; value: number }[] = reactive([])
// 商户完整列表（编辑表单用）
const merchantList = ref<MerchantResp[]>([])

// 分组选项（VXE表单筛选用）
const groupOptions: { label: string; value: number }[] = reactive([])
// 分组完整列表（编辑表单+管理弹窗用）
const groupList = ref<ResourceGroupItem[]>([])

// 分组管理弹窗
const groupDialogVisible = ref(false)
const groupDialogLoading = ref(false)
const newGroupName = ref("")
const editingGroupId = ref<number | null>(null)
const editingGroupName = ref("")

// 加载商户列表
async function loadMerchantList() {
  try {
    const res = await getMerchantList({ page: 1, size: 2000 })
    const list = Array.isArray(res.data?.list) ? res.data.list : []
    merchantList.value = list
    merchantOptions.length = 0
    merchantOptions.push(...list.map((m: MerchantResp) => ({
      label: `${m.name} (${m.no})`,
      value: m.id
    })))
  } catch (e) {
    console.error("加载商户列表失败:", e)
  }
}

// 加载服务器分组列表
async function loadGroupList() {
  try {
    const res = await getResourceGroups("server")
    const list = Array.isArray(res.data) ? res.data : []
    groupList.value = list
    groupOptions.length = 0
    groupOptions.push(...list.map((g: ResourceGroupItem) => ({
      label: `${g.name} (${g.count})`,
      value: g.id
    })))
  } catch (e) {
    console.error("加载分组列表失败:", e)
  }
}

// 分组管理操作
async function handleCreateGroup() {
  if (!newGroupName.value.trim()) return
  groupDialogLoading.value = true
  try {
    await createResourceGroup({ name: newGroupName.value.trim(), sort_order: 0 }, "server")
    newGroupName.value = ""
    ElMessage.success("分组创建成功")
    await loadGroupList()
  } catch (e: any) {
    ElMessage.error(e.message || "创建分组失败")
  } finally {
    groupDialogLoading.value = false
  }
}

function startEditGroup(group: ResourceGroupItem) {
  editingGroupId.value = group.id
  editingGroupName.value = group.name
}

function cancelEditGroup() {
  editingGroupId.value = null
  editingGroupName.value = ""
}

async function handleUpdateGroup(id: number) {
  if (!editingGroupName.value.trim()) return
  groupDialogLoading.value = true
  try {
    await updateResourceGroup(id, { name: editingGroupName.value.trim(), sort_order: 0 })
    editingGroupId.value = null
    editingGroupName.value = ""
    ElMessage.success("分组更新成功")
    await loadGroupList()
    crudStore.commitQuery()
  } catch (e: any) {
    ElMessage.error(e.message || "更新分组失败")
  } finally {
    groupDialogLoading.value = false
  }
}

async function handleDeleteGroup(group: ResourceGroupItem) {
  try {
    await ElMessageBox.confirm(
      `确定删除分组"${group.name}"吗？该分组下的 ${group.count} 个服务器将变为未分组。`,
      "删除分组",
      { type: "warning" }
    )
    groupDialogLoading.value = true
    await deleteResourceGroup(group.id, "server")
    ElMessage.success("分组删除成功")
    await loadGroupList()
    crudStore.commitQuery()
  } catch (e: any) {
    if (e !== "cancel") ElMessage.error(e.message || "删除分组失败")
  } finally {
    groupDialogLoading.value = false
  }
}

// 根据服务器类型生成表格列
function getColumns() {
  const baseColumns: Record<string, any>[] = [
    { field: "id", title: "ID", width: "70px" },
    { field: "name", title: "服务器名称", width: "160px", showOverflow: true },
    { field: "server_type", title: "类型", width: "100px", slots: { default: "type-slot" } },
    { field: "host", title: "主机地址", width: "140px" }
  ]

  // 两种类型都显示商户名称和分组
  baseColumns.push({ title: "商户", width: "140px", slots: { default: "merchant-slot" } })
  baseColumns.push({ title: "分组", width: "120px", slots: { default: "group-slot" } })

  // 系统服务器显示辅助IP和TLS状态，商户服务器显示SSH端口、用户名
  if (serverType.value === 2) {
    baseColumns.push({ field: "auxiliary_ip", title: "辅助IP", width: "140px" })
    baseColumns.push({ title: "TLS", width: "100px", slots: { default: "tls-slot" } })
  } else {
    baseColumns.push({ field: "port", title: "SSH端口", width: "80px" })
    baseColumns.push({ field: "username", title: "用户名", width: "110px", showOverflow: true })
  }

  baseColumns.push({ field: "description", title: "描述", width: "150px", showOverflow: true })

  // 商户服务器显示CPU、内存
  if (serverType.value === 1) {
    baseColumns.push({ title: "CPU", width: "80px", slots: { default: "cpu-slot" } })
    baseColumns.push({ title: "内存", width: "130px", slots: { default: "mem-slot" } })
  }

  baseColumns.push({ field: "status", title: "状态", width: "90px", slots: { default: "status-slot" } })
  baseColumns.push({ field: "created_at", title: "创建时间", width: "160px" })
  baseColumns.push({ title: "操作", width: "200px", fixed: "right", slots: { default: "row-operate" } })

  return baseColumns
}

// 监听服务器类型切换
watch(serverType, () => {
  xGridOpt.columns = getColumns() as any
  crudStore.commitQuery()
})

// ========== VXE Grid 配置 ==========
const xGridDom = ref<VxeGridInstance>()
const xGridOpt: VxeGridProps = reactive({
  loading: true,
  autoResize: true,
  pagerConfig: {
    align: "right"
  },
  formConfig: {
    items: [
      {
        field: "name",
        itemRender: {
          name: "$input",
          props: { placeholder: "服务器名称", clearable: true }
        }
      },
      {
        field: "host",
        itemRender: {
          name: "$input",
          props: { placeholder: "IP地址", clearable: true }
        }
      },
      {
        field: "merchant_id",
        itemRender: {
          name: "$select",
          options: merchantOptions,
          props: { placeholder: "选择商户", clearable: true, filterable: true }
        }
      },
      {
        field: "group_id",
        itemRender: {
          name: "$select",
          options: groupOptions,
          props: { placeholder: "选择分组", clearable: true }
        }
      },
      {
        itemRender: {
          name: "$buttons",
          children: [
            { props: { type: "submit", content: "查询", status: "primary" } },
            { props: { type: "reset", content: "重置" } }
          ]
        }
      }
    ]
  },
  columns: getColumns() as any,
  proxyConfig: {
    seq: true,
    form: true,
    autoLoad: true,
    props: { total: "total" },
    ajax: {
      query: ({ page, form }) => {
        xGridOpt.loading = true
        return new Promise((resolve) => {
          const params: Record<string, unknown> = {
            server_type: serverType.value,
            name: form.name || "",
            host: form.host || "",
            merchant_id: form.merchant_id || undefined,
            group_id: form.group_id || undefined,
            size: page.pageSize,
            page: page.currentPage
          }

          getServerList(params as any)
            .then((res) => {
              xGridOpt.loading = false
              lastPageRows.value = res.data.list || []
              Object.keys(statMap).forEach(k => delete (statMap as Record<string, unknown>)[k])
              resolve({
                total: res.data.total,
                result: res.data.list
              })
            })
            .catch(() => {
              xGridOpt.loading = false
            })
        })
      }
    }
  }
})

// 当前页数据与统计缓存
const lastPageRows = ref<ServerResp[]>([])
const statMap = reactive<Record<number, { cpu_usage: string; memory_usage: string; memory_total: string; error?: string }>>({})

// ========== Modal & Form 配置 ==========
const xModalDom = ref<VxeModalInstance>()
const xFormDom = ref<VxeFormInstance>()

const xModalOpt: VxeModalProps = reactive({
  title: "",
  showClose: true,
  escClosable: true,
  maskClosable: true,
  beforeHideMethod: () => {
    xFormDom.value?.clearValidate()
    return Promise.resolve()
  }
})

const xFormOpt: VxeFormProps = reactive({
  span: 24,
  titleWidth: "120px",
  loading: false,
  titleColon: false,
  data: {
    server_type: 1,
    forward_type: 1,
    merchant_id: 0,
    group_id: 0,
    name: "",
    host: "",
    auxiliary_ip: "",
    port: 22,
    username: "root",
    auth_type: 1,
    password: "",
    private_key: "",
    description: ""
  },
  items: [
    {
      field: "server_type",
      title: "服务器类型",
      itemRender: {
        name: "$radio",
        options: [
          { label: "商户服务器", value: 1 },
          { label: "系统服务器", value: 2 }
        ]
      }
    },
    {
      field: "forward_type",
      title: "转发类型",
      visibleMethod: () => xFormOpt.data.server_type === 2,
      itemRender: {
        name: "$radio",
        options: [
          { label: "加密 (relay+tls)", value: 1 },
          { label: "直连 (tcp)", value: 2 }
        ]
      }
    },
    {
      field: "merchant_id",
      title: "所属商户",
      slots: { default: "merchant-form-slot" }
    },
    {
      field: "group_id",
      title: "分组",
      slots: { default: "group-form-slot" }
    },
    {
      field: "name",
      title: "服务器名称",
      itemRender: {
        name: "$input",
        props: { placeholder: "请输入服务器名称" }
      }
    },
    {
      field: "host",
      title: "主机地址",
      itemRender: {
        name: "$input",
        props: { placeholder: "请输入IP地址" }
      }
    },
    {
      field: "auxiliary_ip",
      title: "辅助IP",
      visibleMethod: () => xFormOpt.data.server_type === 2,
      itemRender: {
        name: "$input",
        props: { placeholder: "请输入辅助IP（用于IP内嵌）" }
      }
    },
    {
      field: "port",
      title: "SSH端口",
      itemRender: {
        name: "$input",
        props: { type: "number", placeholder: "默认22" }
      }
    },
    {
      field: "username",
      title: "SSH用户名",
      itemRender: {
        name: "$input",
        props: { placeholder: "默认root" }
      }
    },
    {
      field: "auth_type",
      title: "认证方式",
      itemRender: {
        name: "$radio",
        options: [
          { label: "密码", value: 1 },
          { label: "密钥", value: 2 }
        ]
      }
    },
    {
      field: "password",
      title: "SSH密码",
      visibleMethod: () => xFormOpt.data.auth_type === 1,
      itemRender: {
        name: "$input",
        props: { type: "password", placeholder: "请输入密码" }
      }
    },
    {
      field: "private_key",
      title: "SSH私钥",
      visibleMethod: () => xFormOpt.data.auth_type === 2,
      itemRender: {
        name: "$textarea",
        props: { placeholder: "请粘贴私钥内容", rows: 8 }
      }
    },
    {
      field: "description",
      title: "描述",
      itemRender: {
        name: "$textarea",
        props: { placeholder: "请输入描述", rows: 3 }
      }
    },
    {
      align: "right",
      itemRender: {
        name: "$buttons",
        children: [
          {
            props: { content: "取消" },
            events: { click: () => xModalDom.value?.close() }
          },
          {
            props: { content: "测试连接", status: "warning" },
            events: { click: () => crudStore.onTestConnection() }
          },
          {
            props: { type: "submit", content: "确定", status: "primary" },
            events: { click: () => crudStore.onSubmitForm() }
          }
        ]
      }
    }
  ],
  rules: {
    name: [{ required: true, message: "请输入服务器名称" }],
    host: [{ required: true, message: "请输入主机地址" }],
    port: [{ required: true, message: "请输入SSH端口" }],
    username: [{ required: true, message: "请输入SSH用户名" }],
    password: [
      {
        validator: ({ itemValue }) => {
          if (xFormOpt.data.auth_type === 1 && !itemValue && !crudStore.isUpdate) {
            return new Error("请输入SSH密码")
          }
        }
      }
    ],
    private_key: [
      {
        validator: ({ itemValue }) => {
          if (xFormOpt.data.auth_type === 2 && !itemValue && !crudStore.isUpdate) {
            return new Error("请粘贴SSH私钥")
          }
        }
      }
    ]
  }
})

// ========== 批量更新程序 ==========
const batchUpdateVisible = ref(false)
const batchUpdateLoading = ref(false)
const batchUpdateForm = reactive({
  serviceName: "server" as "server" | "wukongim",
  targetServerIds: [] as number[],
  restartAfter: true
})
const uploadPercent = ref(0)
const uploadFile = ref<File | null>(null)
const merchantServers = ref<ServerResp[]>([]) // 商户服务器列表（作为目标）
const distributeResults = ref<DistributeResult[]>([])
const distributeStep = ref<"config" | "result">("config")

// 加载商户服务器列表
async function loadServerLists() {
  try {
    // 加载商户服务器（server_type=1）
    const merchantRes = await getServerList({ server_type: 1, page: 1, size: 1000 })
    merchantServers.value = merchantRes.data.list || []
  } catch {
    ElMessage.error("加载服务器列表失败")
  }
}

// 打开批量更新弹窗
async function openBatchUpdate() {
  batchUpdateForm.serviceName = "server"
  batchUpdateForm.targetServerIds = []
  batchUpdateForm.restartAfter = true
  uploadFile.value = null
  uploadPercent.value = 0
  distributeResults.value = []
  distributeStep.value = "config"

  await loadServerLists()
  batchUpdateVisible.value = true
}

// 处理文件选择
function handleFileChange(file: { raw?: File }) {
  if (file.raw) {
    uploadFile.value = file.raw
  }
}

// 全选/取消全选商户服务器
function toggleSelectAll() {
  if (batchUpdateForm.targetServerIds.length === merchantServers.value.length) {
    batchUpdateForm.targetServerIds = []
  } else {
    batchUpdateForm.targetServerIds = merchantServers.value.map(s => s.id)
  }
}

// 执行批量更新
async function executeBatchUpdate() {
  if (!uploadFile.value) {
    return ElMessage.warning("请选择要上传的程序文件")
  }
  if (batchUpdateForm.targetServerIds.length === 0) {
    return ElMessage.warning("请选择至少一个目标服务器")
  }

  batchUpdateLoading.value = true
  uploadPercent.value = 0

  try {
    // Step 1: 上传文件到本地
    ElMessage.info("正在上传文件...")
    const formData = new FormData()
    formData.append("service_name", batchUpdateForm.serviceName)
    formData.append("file", uploadFile.value, batchUpdateForm.serviceName) // 文件名使用服务名

    await uploadToLocal(formData, (percent) => {
      uploadPercent.value = percent
    })
    ElMessage.success("文件上传成功")

    // Step 2: 分发到目标服务器
    ElMessage.info("正在分发到目标服务器...")
    const res = await distributeFile({
      service_name: batchUpdateForm.serviceName,
      target_server_ids: batchUpdateForm.targetServerIds,
      restart_after: batchUpdateForm.restartAfter
    })

    distributeResults.value = res.data.results || []
    distributeStep.value = "result"

    const { success_count, fail_count } = res.data
    if (fail_count === 0) {
      ElMessage.success(`分发完成，全部成功 (${success_count}/${res.data.total_count})`)
    } else {
      ElMessage.warning(`分发完成，成功 ${success_count}，失败 ${fail_count}`)
    }
  } catch (e: any) {
    ElMessage.error(e.message || "批量更新失败")
  } finally {
    batchUpdateLoading.value = false
  }
}

// ========== TLS 管理 ==========
const tlsDialogVisible = ref(false)
const tlsLoading = ref(false)
const tlsStatus = ref<TlsStatusResp | null>(null)
const tlsResults = ref<TlsServerResult[]>([])
const tlsStep = ref<"status" | "result">("status")

async function openTlsDialog() {
  tlsStep.value = "status"
  tlsResults.value = []
  tlsDialogVisible.value = true
  await loadTlsStatus()
}

async function loadTlsStatus() {
  tlsLoading.value = true
  try {
    const res = await getTlsStatus()
    tlsStatus.value = res.data
  } catch (e: any) {
    ElMessage.error(e.message || "获取 TLS 状态失败")
  } finally {
    tlsLoading.value = false
  }
}

async function handleTlsUpgrade() {
  ElMessageBox.confirm("确定将所有系统服务器升级为 TLS 模式？", "批量升级 TLS", { type: "warning" }).then(async () => {
    tlsLoading.value = true
    try {
      const res = await batchUpgradeTls({})
      tlsResults.value = res.data.results || []
      tlsStep.value = "result"
      const { success, failed } = res.data
      if (failed === 0) {
        ElMessage.success(`升级完成，全部成功 (${success}/${res.data.total})`)
      } else {
        ElMessage.warning(`升级完成，成功 ${success}，失败 ${failed}`)
      }
      crudStore.commitQuery()
    } catch (e: any) {
      ElMessage.error(e.message || "TLS 升级失败")
    } finally {
      tlsLoading.value = false
    }
  })
}

async function handleTlsRollback() {
  ElMessageBox.confirm("确定将所有系统服务器回滚为 TCP 模式？", "批量回滚 TLS", { type: "warning" }).then(async () => {
    tlsLoading.value = true
    try {
      const res = await batchRollbackTls({})
      tlsResults.value = res.data.results || []
      tlsStep.value = "result"
      const { success, failed } = res.data
      if (failed === 0) {
        ElMessage.success(`回滚完成，全部成功 (${success}/${res.data.total})`)
      } else {
        ElMessage.warning(`回滚完成，成功 ${success}，失败 ${failed}`)
      }
      crudStore.commitQuery()
    } catch (e: any) {
      ElMessage.error(e.message || "TLS 回滚失败")
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
    ElMessage.error(e.message || "TLS 验证失败")
  } finally {
    tlsLoading.value = false
  }
}

// ========== CRUD 操作 ==========
const crudStore = reactive({
  isUpdate: false,
  currentId: 0,

  commitQuery: () => xGridDom.value?.commitProxy("query"),

  onShowModal: (row?: ServerResp) => {
    if (row) {
      crudStore.isUpdate = true
      crudStore.currentId = row.id
      xModalOpt.title = "编辑服务器"
      xFormOpt.data = {
        server_type: row.server_type,
        forward_type: row.forward_type || 1,
        merchant_id: row.merchant_id || 0,
        group_id: row.group_id || 0,
        name: row.name,
        host: row.host,
        auxiliary_ip: row.auxiliary_ip || "",
        port: row.port,
        username: row.username,
        auth_type: row.auth_type,
        password: "",
        private_key: "",
        description: row.description
      }
    } else {
      crudStore.isUpdate = false
      crudStore.currentId = 0
      xModalOpt.title = "新增服务器"
    }
    xModalDom.value?.open()
    nextTick(() => {
      if (!crudStore.isUpdate) {
        xFormDom.value?.reset()
        xFormOpt.data.server_type = serverType.value
      }
      xFormDom.value?.clearValidate()
    })
  },

  onTestConnection: () => {
    xFormDom.value?.validate((errMap) => {
      if (errMap) return
      ElMessageBox.confirm("确定测试SSH连接?", "提示", { type: "warning" }).then(() => {
        const authType = Number(xFormOpt.data.auth_type)
        const data = {
          host: xFormOpt.data.host,
          port: Number(xFormOpt.data.port),
          username: xFormOpt.data.username,
          auth_type: authType,
          password: authType === 1 ? xFormOpt.data.password : "",
          private_key: authType === 2 ? xFormOpt.data.private_key : ""
        }
        testConnection(data)
          .then(() => {
            ElMessage.success("连接测试成功!")
          })
          .catch(() => {
            ElMessage.error("连接测试失败，请检查配置")
          })
      })
    })
  },

  onSubmitForm: () => {
    if (xFormOpt.loading) return
    xFormDom.value?.validate((errMap) => {
      if (errMap) return
      xFormOpt.loading = true

      const submitData: Record<string, unknown> = {
        ...xFormOpt.data,
        server_type: Number(xFormOpt.data.server_type),
        forward_type: Number(xFormOpt.data.forward_type),
        merchant_id: Number(xFormOpt.data.merchant_id) || 0,
        group_id: Number(xFormOpt.data.group_id) || 0,
        port: Number(xFormOpt.data.port),
        auth_type: Number(xFormOpt.data.auth_type)
      }

      // 根据认证方式清空不需要的字段
      if (submitData.auth_type === 1) {
        submitData.private_key = ""
      } else if (submitData.auth_type === 2) {
        submitData.password = ""
      }

      const apiCall = crudStore.isUpdate
        ? updateServer(crudStore.currentId, submitData as any)
        : createServer(submitData as any)

      apiCall
        .then(() => {
          xFormOpt.loading = false
          xModalDom.value?.close()
          ElMessage.success("操作成功")
          crudStore.commitQuery()
        })
        .catch(() => {
          xFormOpt.loading = false
        })
    })
  },

  onDelete: (row: ServerResp) => {
    ElMessageBox.confirm(`确定删除服务器 "${row.name}" 吗？`, "提示", { type: "warning" }).then(() => {
      deleteServer(row.id).then(() => {
        ElMessage.success("删除成功")
        crudStore.commitQuery()
      })
    })
  },

  onToggleStatus: (row: ServerResp) => {
    const action = row.status === 1 ? "禁用" : "启用"
    ElMessageBox.confirm(`确定${action}服务器 "${row.name}" 吗？`, "提示", { type: "warning" }).then(() => {
      toggleServerStatus(row.id).then(() => {
        ElMessage.success(`${action}成功`)
        crudStore.commitQuery()
      })
    })
  },

  onBatchFetchStats: async () => {
    const rows = lastPageRows.value || []
    if (!rows.length) {
      ElMessage.warning("当前页没有数据")
      return
    }
    const ids = rows.map(r => r.id)
    xGridOpt.loading = true
    try {
      const res = await getServerStatsBatch({ server_ids: ids })
      const list = res.data.stats || []
      Object.keys(statMap).forEach(k => delete (statMap as Record<string, unknown>)[k])
      for (const s of list) {
        statMap[s.server_id] = { cpu_usage: s.cpu_usage, memory_usage: s.memory_usage, memory_total: s.memory_total, error: s.error }
      }
      ElMessage.success("查询完成")
    } catch {
      // 忽略
    } finally {
      xGridOpt.loading = false
    }
  }
})

// 初始化加载商户列表和分组列表
onMounted(() => {
  loadMerchantList()
  loadGroupList()
})
</script>

<template>
  <div class="app-container">
    <!-- 搜索区域 -->
    <el-card shadow="never" class="search-wrapper">
      <div class="flex items-center gap-4 flex-wrap">
        <span class="text-base font-bold">服务器类型:</span>
        <el-radio-group v-model="serverType">
          <el-radio :value="1">商户服务器</el-radio>
          <el-radio :value="2">系统服务器</el-radio>
        </el-radio-group>
      </div>
    </el-card>

    <!-- 表格区域 -->
    <el-card v-loading="!!xGridOpt.loading" shadow="never">
      <div class="toolbar-wrapper">
        <div>
          <el-button type="primary" @click="crudStore.onShowModal()">
            新增服务器
          </el-button>
          <el-button v-if="serverType === 1" type="success" :disabled="!!xGridOpt.loading" @click="crudStore.onBatchFetchStats()">
            查询本页CPU/内存
          </el-button>
          <el-button v-if="serverType === 1" type="warning" @click="openBatchUpdate()">
            批量更新程序
          </el-button>
          <el-button v-if="serverType === 2" type="success" @click="openTlsDialog()">
            TLS 管理
          </el-button>
          <el-button type="info" @click="groupDialogVisible = true">
            管理分组
          </el-button>
        </div>
      </div>

      <div class="table-wrapper">
        <vxe-grid ref="xGridDom" v-bind="xGridOpt">
          <!-- 类型列 -->
          <template #type-slot="{ row }">
            <el-tag v-if="row.server_type === 1" type="primary" size="small">商户</el-tag>
            <template v-else>
              <el-tag type="warning" size="small">系统</el-tag>
              <el-tag v-if="row.forward_type === 2" type="danger" size="small" style="margin-left: 4px">直连</el-tag>
              <el-tag v-else type="success" size="small" style="margin-left: 4px">加密</el-tag>
            </template>
          </template>

          <!-- 商户列 -->
          <template #merchant-slot="{ row }">
            <span v-if="row.merchant_name" class="text-primary">{{ row.merchant_name }}</span>
            <span v-else class="text-gray-400">-</span>
          </template>

          <!-- 分组列 -->
          <template #group-slot="{ row }">
            <el-tag v-if="row.group_name" size="small">{{ row.group_name }}</el-tag>
            <span v-else class="text-gray-400">-</span>
          </template>

          <!-- 状态列 -->
          <template #status-slot="{ row }">
            <el-tag :type="row.status === 1 ? 'success' : 'info'">
              {{ row.status === 1 ? "启用" : "禁用" }}
            </el-tag>
          </template>

          <!-- CPU列 -->
          <template #cpu-slot="{ row }">
            <span>{{ statMap[row.id]?.cpu_usage || '-' }}</span>
          </template>

          <!-- 内存列 -->
          <template #mem-slot="{ row }">
            <span>
              {{ statMap[row.id] ? `${statMap[row.id].memory_usage} / ${statMap[row.id].memory_total}` : '-' }}
            </span>
          </template>

          <!-- TLS状态列 -->
          <template #tls-slot="{ row }">
            <el-tag v-if="row.tls_enabled === 1" type="success" size="small">TLS</el-tag>
            <el-tag v-else type="info" size="small">TCP</el-tag>
          </template>

          <!-- 操作列 -->
          <template #row-operate="{ row }">
            <el-button link type="primary" size="small" @click="crudStore.onShowModal(row)">
              编辑
            </el-button>
            <el-button link type="primary" size="small" @click="$router.push(`/deploy/control?server_id=${row.id}`)">
              控制
            </el-button>
            <el-button
              link
              size="small"
              :type="row.status === 1 ? 'warning' : 'success'"
              @click="crudStore.onToggleStatus(row)"
            >
              {{ row.status === 1 ? '禁用' : '启用' }}
            </el-button>
            <el-button link type="danger" size="small" @click="crudStore.onDelete(row)">
              删除
            </el-button>
          </template>
        </vxe-grid>
      </div>
    </el-card>

    <!-- 服务器编辑弹窗 -->
    <vxe-modal ref="xModalDom" v-bind="xModalOpt">
      <vxe-form ref="xFormDom" v-bind="xFormOpt">
        <template #merchant-form-slot>
          <el-select
            v-model="xFormOpt.data.merchant_id"
            placeholder="不绑定商户"
            style="width: 100%"
            filterable
            clearable
            @clear="xFormOpt.data.merchant_id = 0"
          >
            <el-option
              v-for="m in merchantList"
              :key="m.id"
              :label="`${m.name} (${m.no})`"
              :value="m.id"
            />
          </el-select>
        </template>
        <template #group-form-slot>
          <el-select
            v-model="xFormOpt.data.group_id"
            placeholder="不选择分组"
            style="width: 100%"
            clearable
            @clear="xFormOpt.data.group_id = 0"
          >
            <el-option
              v-for="g in groupList"
              :key="g.id"
              :label="g.name"
              :value="g.id"
            />
          </el-select>
        </template>
      </vxe-form>
    </vxe-modal>

    <!-- 批量更新弹窗 -->
    <el-dialog
      v-model="batchUpdateVisible"
      title="批量更新程序"
      width="700px"
      :close-on-click-modal="false"
    >
      <div v-if="distributeStep === 'config'" v-loading="batchUpdateLoading">
        <el-form label-width="120px">
          <!-- 选择服务类型 -->
          <el-form-item label="程序类型">
            <el-radio-group v-model="batchUpdateForm.serviceName">
              <el-radio value="server">server</el-radio>
              <el-radio value="wukongim">wukongim</el-radio>
            </el-radio-group>
          </el-form-item>

          <!-- 上传程序文件 -->
          <el-form-item label="程序文件">
            <el-upload
              :auto-upload="false"
              :show-file-list="true"
              :limit="1"
              :on-change="handleFileChange"
              accept=""
            >
              <el-button type="primary">选择文件</el-button>
            </el-upload>
            <div v-if="uploadPercent > 0 && uploadPercent < 100" class="upload-progress">
              <el-progress :percentage="uploadPercent" />
            </div>
          </el-form-item>

          <!-- 选择目标服务器 -->
          <el-form-item label="目标服务器">
            <div class="target-servers">
              <div class="target-header">
                <el-button size="small" @click="toggleSelectAll">
                  {{ batchUpdateForm.targetServerIds.length === merchantServers.length ? '取消全选' : '全选' }}
                </el-button>
                <span class="selected-count">已选择 {{ batchUpdateForm.targetServerIds.length }} 个</span>
              </div>
              <el-checkbox-group v-model="batchUpdateForm.targetServerIds" class="target-list">
                <el-checkbox
                  v-for="s in merchantServers"
                  :key="s.id"
                  :value="s.id"
                  :label="`${s.name} (${s.host})`"
                />
              </el-checkbox-group>
            </div>
          </el-form-item>

          <!-- 是否重启 -->
          <el-form-item label="分发后重启">
            <el-switch v-model="batchUpdateForm.restartAfter" />
            <span class="form-tip-inline">分发完成后自动执行 systemctl restart</span>
          </el-form-item>
        </el-form>
      </div>

      <!-- 分发结果 -->
      <div v-else class="distribute-results">
        <el-table :data="distributeResults" max-height="400">
          <el-table-column prop="server_name" label="服务器" width="180" />
          <el-table-column prop="success" label="状态" width="80">
            <template #default="{ row }">
              <el-tag :type="row.success ? 'success' : 'danger'">
                {{ row.success ? '成功' : '失败' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="message" label="消息" show-overflow-tooltip />
        </el-table>
      </div>

      <template #footer>
        <div class="dialog-footer">
          <el-button @click="batchUpdateVisible = false">
            {{ distributeStep === 'result' ? '关闭' : '取消' }}
          </el-button>
          <el-button
            v-if="distributeStep === 'config'"
            type="primary"
            :loading="batchUpdateLoading"
            @click="executeBatchUpdate"
          >
            开始更新
          </el-button>
          <el-button
            v-if="distributeStep === 'result'"
            type="primary"
            @click="distributeStep = 'config'"
          >
            返回配置
          </el-button>
        </div>
      </template>
    </el-dialog>

    <!-- TLS 管理弹窗 -->
    <el-dialog
      v-model="tlsDialogVisible"
      title="TLS 证书管理"
      width="750px"
      :close-on-click-modal="false"
    >
      <div v-loading="tlsLoading">
        <!-- 状态视图 -->
        <template v-if="tlsStep === 'status'">
          <div v-if="tlsStatus" class="tls-summary">
            <el-descriptions :column="3" border size="small">
              <el-descriptions-item label="系统服务器总数">{{ tlsStatus.total }}</el-descriptions-item>
              <el-descriptions-item label="已启用 TLS">
                <el-tag type="success" size="small">{{ tlsStatus.tls_count }}</el-tag>
              </el-descriptions-item>
              <el-descriptions-item label="未启用 (TCP)">
                <el-tag type="info" size="small">{{ tlsStatus.tcp_count }}</el-tag>
              </el-descriptions-item>
            </el-descriptions>

            <el-table :data="tlsStatus.servers" max-height="350" style="margin-top: 16px">
              <el-table-column prop="server_name" label="服务器" width="160" />
              <el-table-column prop="host" label="IP" width="140" />
              <el-table-column label="TLS状态" width="100">
                <template #default="{ row }">
                  <el-tag :type="row.tls_enabled === 1 ? 'success' : 'info'" size="small">
                    {{ row.tls_enabled === 1 ? 'TLS' : 'TCP' }}
                  </el-tag>
                </template>
              </el-table-column>
              <el-table-column label="验证" width="100">
                <template #default="{ row }">
                  <el-tag v-if="row.tls_verified" type="success" size="small">通过</el-tag>
                  <el-tag v-else-if="row.tls_enabled === 1" type="danger" size="small">
                    {{ row.verify_error || '未验证' }}
                  </el-tag>
                  <span v-else class="text-gray-400">-</span>
                </template>
              </el-table-column>
              <el-table-column prop="tls_deployed_at" label="部署时间" width="160" />
            </el-table>
          </div>
          <el-empty v-else description="暂无数据" />
        </template>

        <!-- 操作结果视图 -->
        <template v-else>
          <el-table :data="tlsResults" max-height="400">
            <el-table-column prop="server_name" label="服务器" width="160" />
            <el-table-column prop="host" label="IP" width="140" />
            <el-table-column label="状态" width="80">
              <template #default="{ row }">
                <el-tag :type="row.success ? 'success' : 'danger'" size="small">
                  {{ row.success ? '成功' : '失败' }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="error" label="错误信息" show-overflow-tooltip />
          </el-table>
        </template>
      </div>

      <template #footer>
        <div class="dialog-footer">
          <el-button @click="tlsDialogVisible = false">关闭</el-button>
          <template v-if="tlsStep === 'status'">
            <el-button type="info" :loading="tlsLoading" @click="handleTlsVerify">
              验证连接
            </el-button>
            <el-button type="warning" :loading="tlsLoading" @click="handleTlsRollback">
              批量回滚 TCP
            </el-button>
            <el-button type="success" :loading="tlsLoading" @click="handleTlsUpgrade">
              批量升级 TLS
            </el-button>
          </template>
          <el-button v-else type="primary" @click="tlsStep = 'status'; loadTlsStatus()">
            返回状态
          </el-button>
        </div>
      </template>
    </el-dialog>

    <!-- 分组管理弹窗 -->
    <el-dialog
      v-model="groupDialogVisible"
      title="服务器分组管理"
      width="500px"
      :close-on-click-modal="false"
    >
      <div v-loading="groupDialogLoading">
        <!-- 新建分组 -->
        <div class="group-add-row">
          <el-input
            v-model="newGroupName"
            placeholder="输入新分组名称"
            style="flex: 1"
            @keyup.enter="handleCreateGroup"
          />
          <el-button type="primary" :disabled="!newGroupName.trim()" @click="handleCreateGroup">
            新建
          </el-button>
        </div>

        <!-- 分组列表 -->
        <el-table :data="groupList" max-height="400" style="margin-top: 16px">
          <el-table-column label="分组名称" min-width="150">
            <template #default="{ row }">
              <template v-if="editingGroupId === row.id">
                <el-input
                  v-model="editingGroupName"
                  size="small"
                  @keyup.enter="handleUpdateGroup(row.id)"
                />
              </template>
              <span v-else>{{ row.name }}</span>
            </template>
          </el-table-column>
          <el-table-column prop="count" label="服务器数" width="90" align="center" />
          <el-table-column label="操作" width="160" align="center">
            <template #default="{ row }">
              <template v-if="editingGroupId === row.id">
                <el-button link type="primary" size="small" @click="handleUpdateGroup(row.id)">
                  保存
                </el-button>
                <el-button link size="small" @click="cancelEditGroup">
                  取消
                </el-button>
              </template>
              <template v-else>
                <el-button link type="primary" size="small" @click="startEditGroup(row)">
                  重命名
                </el-button>
                <el-button link type="danger" size="small" @click="handleDeleteGroup(row)">
                  删除
                </el-button>
              </template>
            </template>
          </el-table-column>
        </el-table>

        <el-empty v-if="groupList.length === 0" description="暂无分组" :image-size="60" />
      </div>

      <template #footer>
        <el-button @click="groupDialogVisible = false">关闭</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style lang="scss" scoped>
.app-container {
  padding: 20px;
}

.toolbar-wrapper {
  display: flex;
  justify-content: space-between;
  margin-bottom: 20px;
}

.table-wrapper {
  margin-bottom: 20px;
}

.search-wrapper {
  margin-bottom: 20px;
}

.form-tip {
  font-size: 12px;
  color: #909399;
  margin-top: 4px;
}

.form-tip-inline {
  font-size: 12px;
  color: #909399;
  margin-left: 12px;
}

.upload-progress {
  width: 100%;
  margin-top: 8px;
}

.target-servers {
  width: 100%;
  border: 1px solid #dcdfe6;
  border-radius: 4px;
  padding: 12px;

  .target-header {
    display: flex;
    align-items: center;
    gap: 12px;
    margin-bottom: 12px;
    padding-bottom: 12px;
    border-bottom: 1px solid #ebeef5;

    .selected-count {
      font-size: 12px;
      color: #909399;
    }
  }

  .target-list {
    display: flex;
    flex-direction: column;
    gap: 8px;
    max-height: 200px;
    overflow-y: auto;
  }
}

.distribute-results {
  padding: 0;
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

.tls-summary {
  :deep(.el-descriptions) {
    margin-bottom: 0;
  }
}

.group-add-row {
  display: flex;
  gap: 12px;
  align-items: center;
}
</style>
