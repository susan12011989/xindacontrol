<script lang="ts" setup>
import type { CloudAccountResp } from "@@/apis/cloud_account/type"
import type { VxeFormInstance, VxeFormProps, VxeGridInstance, VxeGridProps, VxeModalInstance, VxeModalProps } from "vxe-table"
import { getBillingCostUsage } from "@@/apis/aws"
import { createCloudAccount, deleteCloudAccount, getAliyunBalance, getCloudAccountList, getTencentBalance, updateCloudAccount } from "@@/apis/cloud_account"
import { getMerchantList } from "@@/apis/merchant"
import { getAwsRegions } from "@@/constants/aws-regions"

defineOptions({
  name: "CloudAccountManagement"
})

// 云类型选项
const cloudTypeOptions = [
  { label: "阿里云", value: "aliyun" },
  { label: "AWS", value: "aws" },
  { label: "腾讯云", value: "tencent" }
]

// 站点类型选项（仅阿里云需要区分）
const siteTypeOptions = [
  { label: "国内站", value: "cn" },
  { label: "国际站", value: "intl" }
]

// 状态选项
const statusOptions = [
  { label: "启用", value: 1 },
  { label: "禁用", value: 0 }
]

// 商户选项
const merchantOptions: { label: string; value: number }[] = reactive([])

async function loadMerchants() {
  try {
    const res = await getMerchantList({ page: 1, size: 1000 })
    const list = res.data.list?.map((m: any) => ({
      label: `${m.name} (${m.no})`,
      value: m.id
    })) || []
    merchantOptions.length = 0
    merchantOptions.push(...list)
  } catch (e) {
    console.error("加载商户列表失败", e)
  }
}

onMounted(() => {
  loadMerchants()
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
          props: { placeholder: "账号名称", clearable: true }
        }
      },
      {
        field: "cloud_type",
        itemRender: {
          name: "$select",
          options: cloudTypeOptions,
          props: { placeholder: "云类型", clearable: true }
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
        field: "status",
        itemRender: {
          name: "$select",
          options: statusOptions,
          props: { placeholder: "状态", clearable: true }
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
  toolbarConfig: {
    refresh: true,
    custom: true,
    slots: { buttons: "toolbar-btns" }
  },
  columns: [
    { type: "seq", width: "60px", title: "序号" },
    { field: "name", title: "账号名称", width: 180 },
    {
      field: "cloud_type",
      title: "云类型",
      width: 120,
      slots: { default: "cloud-type-slot" }
    },
    {
      field: "site_type",
      title: "站点",
      width: 100,
      slots: { default: "site-type-slot" }
    },
    {
      field: "merchant_name",
      title: "商户",
      width: 180,
      slots: { default: "merchant-slot" }
    },
    { field: "access_key_id", title: "AccessKeyId", width: 200, showOverflow: true },
    { field: "description", title: "描述", showOverflow: true },
    { title: "余额", width: 120, slots: { default: "balance-slot" } },
    {
      field: "status",
      title: "状态",
      width: 100,
      slots: { default: "status-slot" }
    },
    { field: "created_at", title: "创建时间", width: 180 },
    {
      title: "操作",
      width: "150px",
      fixed: "right",
      slots: { default: "row-operate" }
    }
  ],
  proxyConfig: {
    seq: true,
    form: true,
    autoLoad: true,
    props: { total: "total" },
    ajax: {
      query: ({ page, form }) => {
        xGridOpt.loading = true
        return new Promise((resolve) => {
          const params = {
            name: form.name || "",
            cloud_type: form.cloud_type || "",
            status: form.status,
            merchant_id: form.merchant_id || "",
            size: page.pageSize,
            page: page.currentPage
          }
          getCloudAccountList(params).then((res) => {
            xGridOpt.loading = false
            resolve({
              total: res.data.total,
              result: res.data.list
            })
          }).catch(() => {
            xGridOpt.loading = false
          })
        })
      }
    }
  }
})

// ========== Modal & Form 配置 ==========
const xModalDom = ref<VxeModalInstance>()
const xFormDom = ref<VxeFormInstance>()

const xModalOpt: VxeModalProps = reactive({
  title: "",
  showClose: true,
  escClosable: true,
  maskClosable: true,
  width: 600,
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
    name: "",
    cloud_type: "",
    site_type: "cn",
    merchant_id: 0,
    access_key_id: "",
    access_key_secret: "",
    description: ""
  },
  items: [
    {
      field: "name",
      title: "账号名称",
      itemRender: {
        name: "$input",
        props: { placeholder: "请输入账号名称" }
      }
    },
    {
      field: "cloud_type",
      title: "云类型",
      itemRender: {
        name: "$select",
        options: cloudTypeOptions,
        props: { placeholder: "请选择云类型" }
      }
    },
    {
      field: "merchant_id",
      title: "所属商户",
      itemRender: {
        name: "$select",
        options: merchantOptions,
        props: { placeholder: "不选则为系统账号", clearable: true, filterable: true }
      }
    },
    {
      field: "site_type",
      title: "站点类型",
      itemRender: {
        name: "$select",
        options: siteTypeOptions,
        props: { placeholder: "请选择站点类型" }
      }
    },
    {
      field: "access_key_id",
      title: "AccessKeyId",
      itemRender: {
        name: "$input",
        props: { placeholder: "请输入AccessKeyId" }
      }
    },
    {
      field: "access_key_secret",
      title: "AccessKeySecret",
      itemRender: {
        name: "$input",
        props: {
          type: "password",
          placeholder: "请输入AccessKeySecret",
          showPassword: true
        }
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
            props: { type: "submit", content: "确定", status: "primary" },
            events: { click: () => crudStore.onSubmitForm() }
          }
        ]
      }
    }
  ],
  rules: {
    name: [{ required: true, message: "请输入账号名称" }],
    cloud_type: [{ required: true, message: "请选择云类型" }],
    access_key_id: [{ required: true, message: "请输入AccessKeyId" }],
    access_key_secret: [{ required: true, message: "请输入AccessKeySecret" }]
  }
})

// ========== CRUD 操作 ==========
const crudStore = reactive({
  isUpdate: false,
  currentId: 0,

  commitQuery: () => xGridDom.value?.commitProxy("query"),

  onShowModal: (row?: CloudAccountResp) => {
    if (row) {
      crudStore.isUpdate = true
      crudStore.currentId = row.id
      xModalOpt.title = "编辑云账号"
      xFormOpt.data = {
        name: row.name,
        cloud_type: row.cloud_type,
        site_type: row.site_type || "cn",
        merchant_id: row.merchant_id || 0,
        access_key_id: row.access_key_id,
        access_key_secret: row.access_key_secret,
        description: row.description
      }
      // 编辑时云类型不可修改
      if (xFormOpt.items?.[1]?.itemRender) {
        xFormOpt.items[1].itemRender.props = {
          ...xFormOpt.items[1].itemRender.props,
          disabled: true
        }
      }
    } else {
      crudStore.isUpdate = false
      crudStore.currentId = 0
      xModalOpt.title = "新增云账号"
      // 新增时云类型可选
      if (xFormOpt.items?.[1]?.itemRender) {
        xFormOpt.items[1].itemRender.props = {
          placeholder: "请选择云类型"
        }
      }
    }
    xModalDom.value?.open()
    nextTick(() => {
      !crudStore.isUpdate && xFormDom.value?.reset()
      xFormDom.value?.clearValidate()
    })
  },

  onSubmitForm: () => {
    if (xFormOpt.loading) return
    xFormDom.value?.validate((errMap) => {
      if (errMap) return
      xFormOpt.loading = true

      const apiCall = crudStore.isUpdate
        ? updateCloudAccount(crudStore.currentId, xFormOpt.data)
        : createCloudAccount(xFormOpt.data)

      apiCall.then(() => {
        xFormOpt.loading = false
        xModalDom.value?.close()
        ElMessage.success("操作成功")
        crudStore.commitQuery()
      }).catch(() => {
        xFormOpt.loading = false
      })
    })
  },

  onDelete: (row: CloudAccountResp) => {
    ElMessageBox.confirm(
      `确定删除云账号 "${row.name}" 吗？`,
      "提示",
      { type: "warning" }
    ).then(() => {
      deleteCloudAccount(row.id).then(() => {
        ElMessage.success("删除成功")
        crudStore.commitQuery()
      })
    })
  },

  onToggleStatus: (row: CloudAccountResp) => {
    const newStatus = row.status === 1 ? 0 : 1
    const statusText = newStatus === 1 ? "启用" : "禁用"

    ElMessageBox.confirm(
      `确定${statusText}云账号 "${row.name}" 吗？`,
      "提示",
      { type: "warning" }
    ).then(() => {
      updateCloudAccount(row.id, { status: newStatus }).then(() => {
        ElMessage.success(`${statusText}成功`)
        crudStore.commitQuery()
      })
    })
  }
})

// 余额弹窗状态（阿里云）
const balanceDialogVisible = ref(false)
const balanceDialogLoading = ref(false)
const balanceDialogValue = ref<string>("")

// 腾讯云余额弹窗
const tencentBalanceDialog = reactive({
  visible: false,
  loading: false,
  data: null as any
})

function buildBalanceParams(row: CloudAccountResp) {
  return row.account_type === "merchant" && row.merchant_id
    ? { merchant_id: row.merchant_id }
    : { cloud_account_id: row.id }
}

function onShowBalance(row: CloudAccountResp) {
  if (row.cloud_type === "aliyun") {
    balanceDialogVisible.value = true
    balanceDialogLoading.value = true
    getAliyunBalance(buildBalanceParams(row)).then((res) => {
      balanceDialogValue.value = res.data.balance
    }).finally(() => {
      balanceDialogLoading.value = false
    })
  } else if (row.cloud_type === "tencent") {
    tencentBalanceDialog.visible = true
    tencentBalanceDialog.loading = true
    tencentBalanceDialog.data = null
    getTencentBalance(buildBalanceParams(row)).then((res) => {
      tencentBalanceDialog.data = res.data
    }).finally(() => {
      tencentBalanceDialog.loading = false
    })
  }
}

// AWS 账单弹窗逻辑（从云账号列表直达）
const awsBillingDialog = reactive({
  visible: false,
  loading: false,
  cloudAccountId: 0,
  regions: getAwsRegions("cn"),
  form: {
    region_id: "",
    start: "",
    end: "",
    granularity: "DAILY",
    group_by_key: "SERVICE"
  },
  result: null as any,
  tableRows: [] as Array<{ service: string, amount: number, unit: string }>,
  currency: "",
  totalAmount: 0,
  periodStart: "",
  periodEnd: "",
  estimated: false,
  // 按日期明细（每日/每月）
  detailPeriods: [] as Array<{
    date: string
    end: string
    estimated: boolean
    subtotal: number
    unit: string
    items: Array<{ group: string, amount: number }>
  }>,
  tab: "summary",
  excludeEstimated: false
})

function onOpenAwsBilling(row: CloudAccountResp) {
  awsBillingDialog.cloudAccountId = row.id
  // 默认时间：本月1号 ~ 今天（CE为开区间End，传今天即可代表“到目前”）
  const now = new Date()
  const y = now.getFullYear()
  const m = String(now.getMonth() + 1).padStart(2, "0")
  const d = String(now.getDate()).padStart(2, "0")
  awsBillingDialog.form.start = `${y}-${m}-01`
  awsBillingDialog.form.end = `${y}-${m}-${d}`
  awsBillingDialog.visible = true
}

async function onFetchAwsBilling() {
  if (!awsBillingDialog.form.region_id) {
    ElMessage.warning("请选择Region")
    return
  }
  if (!awsBillingDialog.form.start || !awsBillingDialog.form.end) {
    ElMessage.warning("请选择起止日期")
    return
  }
  awsBillingDialog.loading = true
  try {
    const { data } = await getBillingCostUsage({
      cloud_account_id: awsBillingDialog.cloudAccountId,
      region_id: awsBillingDialog.form.region_id,
      start: awsBillingDialog.form.start,
      end: awsBillingDialog.form.end,
      granularity: awsBillingDialog.form.granularity as any,
      group_by_key: awsBillingDialog.form.group_by_key
    })
    awsBillingDialog.result = data
    parseAwsCostUsage()
  } finally {
    awsBillingDialog.loading = false
  }
}

function parseAwsCostUsage() {
  const data: any = awsBillingDialog.result
  const rowsMap: Record<string, { service: string, amount: number, unit: string }> = {}
  let unit = ""
  let total = 0
  let start = ""
  let end = ""
  let estimated = false
  const list = data?.ResultsByTime || []
  if (list.length > 0) {
    start = list[0]?.TimePeriod?.Start || ""
    end = list[list.length - 1]?.TimePeriod?.End || ""
  }
  // 清空按日期明细
  awsBillingDialog.detailPeriods = []
  for (const r of list) {
    if (awsBillingDialog.excludeEstimated && r?.Estimated) {
      continue
    }
    if (r?.Estimated) estimated = true
    const groups = r?.Groups || []
    // 汇总每个时间段小计
    let subtotal = 0
    const periodItems: Array<{ group: string, amount: number }> = []
    for (const g of groups) {
      const service = (g?.Keys && g.Keys[0]) || "-"
      const m = g?.Metrics?.UnblendedCost
      if (!m) continue
      const amt = Number.parseFloat(m.Amount || "0")
      unit = m.Unit || unit
      total += Number.isFinite(amt) ? amt : 0
      if (!rowsMap[service]) {
        rowsMap[service] = { service, amount: 0, unit: unit || "USD" }
      }
      rowsMap[service].amount += Number.isFinite(amt) ? amt : 0
      subtotal += Number.isFinite(amt) ? amt : 0
      periodItems.push({ group: service, amount: Number.isFinite(amt) ? Number(amt.toFixed(4)) : 0 })
    }
    // 写入单日/单月
    awsBillingDialog.detailPeriods.push({
      date: r?.TimePeriod?.Start || "",
      end: r?.TimePeriod?.End || "",
      estimated: !!r?.Estimated,
      subtotal: Number(subtotal.toFixed(4)),
      unit: unit || "USD",
      items: periodItems.sort((a, b) => b.amount - a.amount)
    })
  }
  const rows = Object.values(rowsMap).sort((a, b) => b.amount - a.amount)
  awsBillingDialog.tableRows = rows
  awsBillingDialog.currency = unit || "USD"
  awsBillingDialog.totalAmount = Number(total.toFixed(4))
  awsBillingDialog.periodStart = start
  awsBillingDialog.periodEnd = end
  awsBillingDialog.estimated = !!estimated
}

function groupKeyNameCN(key: string) {
  const map: Record<string, string> = {
    SERVICE: "服务",
    LINKED_ACCOUNT: "关联账号",
    OPERATION: "操作",
    USAGE_TYPE: "用量类型"
  }
  return map[key] || key
}

// 获取云类型标签类型
function getCloudTypeTag(type: string): "primary" | "warning" | "info" | "success" | "danger" {
  const map: Record<string, "primary" | "warning" | "info" | "success" | "danger"> = {
    aliyun: "primary",
    aws: "warning",
    tencent: "success"
  }
  return map[type] || "info"
}

// 获取云类型标签文本
function getCloudTypeText(type: string) {
  const map: Record<string, string> = {
    aliyun: "阿里云",
    aws: "AWS",
    tencent: "腾讯云"
  }
  return map[type] || type
}
</script>

<template>
  <div class="app-container">
    <!-- 表格 -->
    <vxe-grid ref="xGridDom" v-bind="xGridOpt">
      <!-- 工具栏按钮 -->
      <template #toolbar-btns>
        <vxe-button status="primary" icon="vxe-icon-add" @click="crudStore.onShowModal()">
          新增云账号
        </vxe-button>
      </template>

      <!-- 云类型列 -->
      <template #cloud-type-slot="{ row }">
        <el-tag :type="getCloudTypeTag(row.cloud_type)">
          {{ getCloudTypeText(row.cloud_type) }}
        </el-tag>
      </template>

      <!-- 站点类型列 -->
      <template #site-type-slot="{ row }">
        <el-tag v-if="row.cloud_type === 'aliyun'" :type="row.site_type === 'intl' ? 'warning' : 'success'" size="small">
          {{ row.site_type === 'intl' ? '国际站' : '国内站' }}
        </el-tag>
        <span v-else>-</span>
      </template>

      <!-- 商户列 -->
      <template #merchant-slot="{ row }">
        <el-tag v-if="row.merchant_name" type="success">
          {{ row.merchant_name }}
        </el-tag>
        <el-tag v-else type="info">系统</el-tag>
      </template>
      <!-- 余额列 -->
      <template #balance-slot="{ row }">
        <el-button v-if="row.cloud_type === 'aliyun'" link type="primary" @click="onShowBalance(row)">查余额</el-button>
        <el-button v-else-if="row.cloud_type === 'aws'" link type="primary" @click="onOpenAwsBilling(row)">账单</el-button>
        <el-button v-else-if="row.cloud_type === 'tencent'" link type="primary" @click="onShowBalance(row)">查余额</el-button>
        <span v-else>-</span>
      </template>

      <!-- 状态列 -->
      <template #status-slot="{ row }">
        <el-tag :type="row.status === 1 ? 'success' : 'info'">
          {{ row.status === 1 ? '启用' : '禁用' }}
        </el-tag>
      </template>

      <!-- 操作列 -->
      <template #row-operate="{ row }">
        <el-button link type="primary" @click="crudStore.onShowModal(row)">
          编辑
        </el-button>
        <el-button
          link
          :type="row.status === 1 ? 'warning' : 'success'"
          @click="crudStore.onToggleStatus(row)"
        >
          {{ row.status === 1 ? '禁用' : '启用' }}
        </el-button>
        <el-button link type="danger" @click="crudStore.onDelete(row)">
          删除
        </el-button>
      </template>
    </vxe-grid>

    <!-- 弹窗 -->
    <vxe-modal ref="xModalDom" v-bind="xModalOpt">
      <vxe-form ref="xFormDom" v-bind="xFormOpt" />
    </vxe-modal>

    <!-- 阿里云余额弹窗 -->
    <el-dialog v-model="balanceDialogVisible" title="阿里云账户余额" width="360px">
      <div style="min-height: 60px; display: flex; align-items: center;">
        <el-skeleton v-if="balanceDialogLoading" :rows="1" animated />
        <div v-else>余额：<b>{{ balanceDialogValue || '-' }}</b></div>
      </div>
      <template #footer>
        <el-button @click="balanceDialogVisible = false">关闭</el-button>
      </template>
    </el-dialog>

    <!-- 腾讯云余额弹窗 -->
    <el-dialog v-model="tencentBalanceDialog.visible" title="腾讯云账户余额" width="450px">
      <div v-if="tencentBalanceDialog.loading" style="min-height: 120px">
        <el-skeleton :rows="4" animated />
      </div>
      <div v-else-if="tencentBalanceDialog.data" class="tencent-balance">
        <el-descriptions :column="1" border>
          <el-descriptions-item label="可用余额">
            <span style="font-size: 18px; font-weight: bold; color: #409eff">{{ tencentBalanceDialog.data.balance_yuan }} 元</span>
          </el-descriptions-item>
          <el-descriptions-item label="现金余额">{{ tencentBalanceDialog.data.cash_balance_yuan }} 元</el-descriptions-item>
          <el-descriptions-item label="赠送余额">{{ (tencentBalanceDialog.data.present_balance / 100).toFixed(2) }} 元</el-descriptions-item>
          <el-descriptions-item label="收入余额">{{ (tencentBalanceDialog.data.income_balance / 100).toFixed(2) }} 元</el-descriptions-item>
          <el-descriptions-item label="冻结金额">{{ (tencentBalanceDialog.data.freeze_balance / 100).toFixed(2) }} 元</el-descriptions-item>
          <el-descriptions-item label="欠费金额">
            <span :style="{ color: tencentBalanceDialog.data.owe_balance > 0 ? '#f56c6c' : '' }">
              {{ (tencentBalanceDialog.data.owe_balance / 100).toFixed(2) }} 元
            </span>
          </el-descriptions-item>
          <el-descriptions-item label="欠费状态">
            <el-tag :type="tencentBalanceDialog.data.is_overdue ? 'danger' : 'success'" size="small">
              {{ tencentBalanceDialog.data.is_overdue ? '欠费' : '正常' }}
            </el-tag>
          </el-descriptions-item>
        </el-descriptions>
      </div>
      <div v-else>查询失败</div>
      <template #footer>
        <el-button @click="tencentBalanceDialog.visible = false">关闭</el-button>
      </template>
    </el-dialog>

    <!-- AWS 账单弹窗 -->
    <el-dialog v-model="awsBillingDialog.visible" :title="`AWS 账单查询（按 ${groupKeyNameCN(awsBillingDialog.form.group_by_key)}）`" width="980px">
      <div class="mb-3 flex gap-2 items-end flex-wrap">
        <el-select v-model="awsBillingDialog.form.region_id" placeholder="Region" filterable style="min-width: 260px">
          <el-option v-for="r in awsBillingDialog.regions" :key="r.id" :label="`${r.name} (${r.id})`" :value="r.id" />
        </el-select>
        <el-date-picker v-model="awsBillingDialog.form.start" type="date" placeholder="开始日期" value-format="YYYY-MM-DD" />
        <el-date-picker v-model="awsBillingDialog.form.end" type="date" placeholder="结束日期" value-format="YYYY-MM-DD" />
        <el-select v-model="awsBillingDialog.form.granularity" style="width: 160px">
          <el-option label="每日" value="DAILY" />
          <el-option label="每月" value="MONTHLY" />
        </el-select>
        <el-select v-model="awsBillingDialog.form.group_by_key" style="width: 200px">
          <el-option label="服务" value="SERVICE" />
          <el-option label="关联账号" value="LINKED_ACCOUNT" />
        </el-select>
        <el-checkbox v-model="awsBillingDialog.excludeEstimated">仅显示已出账</el-checkbox>
        <el-button type="primary" :loading="awsBillingDialog.loading" @click="onFetchAwsBilling">查询</el-button>
      </div>
      <div v-if="awsBillingDialog.loading" style="min-height: 220px">
        <el-skeleton :rows="6" animated />
      </div>
      <div v-else>
        <div class="mb-2 text-sm text-gray-500">
          <span>周期：{{ awsBillingDialog.periodStart }} ~ {{ awsBillingDialog.periodEnd }}</span>
          <span class="ml-4">合计：{{ awsBillingDialog.totalAmount }} {{ awsBillingDialog.currency }}</span>
          <span v-if="awsBillingDialog.estimated" class="ml-4">（含预估）</span>
        </div>
        <el-tabs v-model="awsBillingDialog.tab">
          <el-tab-pane name="summary" label="汇总（按分组）">
            <el-table :data="awsBillingDialog.tableRows" height="520" border>
              <el-table-column :label="groupKeyNameCN(awsBillingDialog.form.group_by_key)" prop="service" min-width="260" />
              <el-table-column prop="amount" label="金额" width="160">
                <template #default="{ row }">
                  {{ Number(row.amount).toFixed(4) }}
                </template>
              </el-table-column>
              <el-table-column prop="unit" label="币种" width="120" />
            </el-table>
          </el-tab-pane>
          <el-tab-pane name="bydate" :label="awsBillingDialog.form.granularity === 'MONTHLY' ? '按月' : '按日'">
            <el-table :data="awsBillingDialog.detailPeriods" height="520" border>
              <el-table-column type="expand">
                <template #default="{ row }">
                  <el-table :data="row.items" size="small">
                    <el-table-column :label="groupKeyNameCN(awsBillingDialog.form.group_by_key)" prop="group" min-width="220" />
                    <el-table-column prop="amount" label="金额" width="140">
                      <template #default="{ row: r }">
                        {{ Number(r.amount).toFixed(4) }}
                      </template>
                    </el-table-column>
                    <el-table-column label="币种" width="120">
                      <template #default>
                        {{ awsBillingDialog.currency }}
                      </template>
                    </el-table-column>
                  </el-table>
                </template>
              </el-table-column>
              <el-table-column :label="awsBillingDialog.form.granularity === 'MONTHLY' ? '月份' : '日期'" width="180">
                <template #default="{ row }">
                  {{ row.date }}<span v-if="row.end && row.end !== row.date"> ~ {{ row.end }}</span>
                </template>
              </el-table-column>
              <el-table-column prop="subtotal" label="合计金额" width="160">
                <template #default="{ row }">
                  {{ Number(row.subtotal).toFixed(4) }}
                </template>
              </el-table-column>
              <el-table-column label="币种" width="120">
                <template #default="{ row }">{{ row.unit }}</template>
              </el-table-column>
              <el-table-column label="预估" width="100">
                <template #default="{ row }">
                  <el-tag size="small" :type="row.estimated ? 'warning' : 'success'">{{ row.estimated ? '是' : '否' }}</el-tag>
                </template>
              </el-table-column>
            </el-table>
          </el-tab-pane>
        </el-tabs>
      </div>
      <template #footer>
        <el-button @click="awsBillingDialog.visible = false">关闭</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style lang="scss" scoped>
.app-container {
  padding: 20px;
}
</style>
