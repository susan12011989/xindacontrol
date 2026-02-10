<script setup lang="ts">
import type { Region } from "@/pages/cloud/aliyun/apis/type"
import type { Bandwidth, CreateBandwidthRequestData } from "@/pages/cloud/aliyun/bandwidth/apis/type"
import type { Eip } from "@/pages/cloud/aliyun/eip/apis/type"
import type { Merchant, MerchantRegions } from "@/pages/dashboard/apis/type"
import type { CloudAccountOption } from "@@/apis/cloud_account/type"
import { regionListApi } from "@/pages/cloud/aliyun/apis"
import { createBandwidth, getBandwidthList, operateBandwidth } from "@/pages/cloud/aliyun/bandwidth/apis"
import { getEipList } from "@/pages/cloud/aliyun/eip/apis"
import { merchantQueryApi } from "@/pages/dashboard/apis"
import { getCloudAccountOptions } from "@@/apis/cloud_account"
import { ArrowDown, Delete, Edit, Key, Location, Plus, User } from "@element-plus/icons-vue"
import { ElMessage, ElMessageBox } from "element-plus"
import { debounce } from "lodash-es"
import { onMounted, reactive, ref, watch } from "vue"

defineOptions({
  name: "CloudBandwidth"
})

// localStorage 存储键名
const STORAGE_ACCOUNT_TYPE_KEY = "cloud_account_type"
const STORAGE_CLOUD_ACCOUNT_KEY = "cloud_selected_cloud_account"
const STORAGE_MERCHANT_KEY = "cloud_selected_merchant"
const STORAGE_REGION_KEY = "cloud_selected_region"

// 数据状态
const loading = ref(false)
const accountType = ref<string>("merchant")
const cloudAccountList = ref<CloudAccountOption[]>([])
const selectedCloudAccount = ref<number>()
const merchantList = ref<Merchant[]>([])
const regionList = ref<Region[]>([])
const bandwidthList = ref<Bandwidth[]>([])
const selectedMerchant = ref<number>()
const selectedRegions = ref<string[]>([])

// 创建共享带宽对话框
const createDialogVisible = ref(false)
const createLoading = ref(false)
const createFormRef = ref()
const createBandwidthForm = reactive<CreateBandwidthRequestData>({
  region_id: "",
  bandwidth: 5 // 默认带宽5Mbps
})

// 创建表单规则
const createFormRules = {
  bandwidth: [
    { required: true, message: "请输入带宽值", trigger: "blur" },
    { pattern: /^[1-9]\d*$/, message: "带宽必须为正整数", trigger: "blur" }
  ]
}

// 修改共享带宽对话框
const modifyDialogVisible = ref(false)
const modifyLoading = ref(false)
const modifyFormRef = ref()
const modifyBandwidthForm = reactive({
  name: "",
  description: ""
})
const currentEditBandwidth = ref<Bandwidth | null>(null)

// 修改表单规则
const modifyFormRules = {
  name: [
    { max: 128, message: "名称最大长度为128个字符", trigger: "blur" }
  ],
  description: [
    { max: 256, message: "描述最大长度为256个字符", trigger: "blur" }
  ]
}

// 添加弹性IP对话框
const addEipDialogVisible = ref(false)
const addEipLoading = ref(false)
const availableEipList = ref<Eip[]>([])
const loadingEips = ref(false)
const selectedEipIds = ref<string[]>([])
const currentBandwidthForAddEip = ref<Bandwidth | null>(null)

// 从 localStorage 读取已保存的选择
function loadSelectionFromStorage() {
  try {
    const savedAccountType = localStorage.getItem(STORAGE_ACCOUNT_TYPE_KEY)
    const savedCloudAccountId = localStorage.getItem(STORAGE_CLOUD_ACCOUNT_KEY)
    const savedMerchantId = localStorage.getItem(STORAGE_MERCHANT_KEY)
    const savedRegionIds = localStorage.getItem(STORAGE_REGION_KEY)

    if (savedAccountType) accountType.value = savedAccountType
    if (savedCloudAccountId) selectedCloudAccount.value = Number(savedCloudAccountId)
    if (savedMerchantId) selectedMerchant.value = Number(savedMerchantId)
    if (savedRegionIds) selectedRegions.value = JSON.parse(savedRegionIds)
  } catch (error) {
    console.error("读取localStorage数据失败:", error)
  }
}

// 监听选择变化，保存到localStorage
watch(accountType, (newValue) => {
  if (newValue) localStorage.setItem(STORAGE_ACCOUNT_TYPE_KEY, newValue)
})

watch(selectedCloudAccount, (newValue) => {
  if (newValue) {
    localStorage.setItem(STORAGE_CLOUD_ACCOUNT_KEY, newValue.toString())
  } else {
    localStorage.removeItem(STORAGE_CLOUD_ACCOUNT_KEY)
  }
})

watch(selectedMerchant, (newValue) => {
  if (newValue) {
    localStorage.setItem(STORAGE_MERCHANT_KEY, newValue.toString())
  } else {
    localStorage.removeItem(STORAGE_MERCHANT_KEY)
  }
}, { deep: true })

watch(selectedRegions, (newValue) => {
  if (newValue && newValue.length > 0) {
    localStorage.setItem(STORAGE_REGION_KEY, JSON.stringify(newValue))
  } else {
    localStorage.removeItem(STORAGE_REGION_KEY)
  }
}, { deep: true })

// 获取云账号列表
async function fetchCloudAccountList() {
  try {
    const res = await getCloudAccountOptions("aliyun")
    cloudAccountList.value = res.data || []
  } catch (error) {
    console.error("获取云账号列表失败", error)
    ElMessage.error("获取云账号列表失败")
  }
}

// 获取商户列表
async function fetchMerchantList() {
  try {
    const res = await merchantQueryApi({
      page: 1,
      size: 9000
    })
    merchantList.value = res.data.list
  } catch (error) {
    console.error("获取商户列表失败", error)
    ElMessage.error("获取商户列表失败")
  }
}

// 处理账号类型切换
function handleAccountTypeChange() {
  selectedRegions.value = []
  bandwidthList.value = []

  if (accountType.value === "system") {
    selectedMerchant.value = undefined
    fetchCloudAccountList()
  } else {
    selectedCloudAccount.value = undefined
    fetchMerchantList()
  }
}

// 处理云账号切换
function handleCloudAccountChange() {
  selectedRegions.value = []
  bandwidthList.value = []

  if (selectedCloudAccount.value) {
    fetchRegionList()
  }
}

// 获取区域列表
async function fetchRegionList() {
  try {
    const res = await regionListApi()
    regionList.value = res.data
  } catch (error) {
    console.error("获取区域列表失败", error)
    ElMessage.error("获取区域列表失败")
  }
}

// 获取共享带宽列表
async function fetchBandwidthList() {
  const hasAccount = accountType.value === "merchant" ? selectedMerchant.value : selectedCloudAccount.value
  if (!hasAccount || !selectedRegions.value || selectedRegions.value.length === 0) {
    console.log("账号或区域未选择，不获取共享带宽列表")
    return
  }

  console.log("开始获取共享带宽列表, 类型:", accountType.value, "账号:", hasAccount, "区域:", selectedRegions.value)
  loading.value = true
  try {
    const params: any = { region_id: selectedRegions.value }
    if (accountType.value === "merchant") {
      params.merchant_id = selectedMerchant.value
    } else {
      params.cloud_account_id = selectedCloudAccount.value
    }

    const res = await getBandwidthList(params)

    console.log("获取共享带宽列表成功:", res.data.list)
    // 兼容返回空数组的情况
    if (!res.data.list || !Array.isArray(res.data.list)) {
      console.log("共享带宽列表为空或格式不正确，设置为空数组")
      bandwidthList.value = []
    } else {
      bandwidthList.value = res.data.list
    }

    // 如果列表为空，显示提示
    if (bandwidthList.value.length === 0) {
      ElMessage.info("当前区域暂无共享带宽")
    }
  } catch (error) {
    console.error("获取共享带宽列表失败", error)
    ElMessage.error("获取共享带宽列表失败")
    bandwidthList.value = []
  } finally {
    loading.value = false
  }
}

// 商户变化处理
function handleMerchantChange(value: number | undefined) {
  console.log("商户变更为:", value)
  selectedRegions.value = []
  bandwidthList.value = []

  // 如果选中了商户，尝试获取区域列表
  if (value) {
    // 延迟执行，确保UI更新
    setTimeout(() => {
      fetchRegionList().then(() => {
        // 获取区域列表后，尝试根据商户区域自动匹配
        const selectedMerchantInfo = merchantList.value.find(m => m.id === value)
        if (selectedMerchantInfo?.regions && selectedMerchantInfo.regions.length > 0 && regionList.value.length > 0) {
          autoMatchRegions(selectedMerchantInfo.regions)
        }
      })
    }, 100)
  }
}

// 根据商户区域信息自动匹配区域ID
function autoMatchRegions(merchantRegions: MerchantRegions[]) {
  if (!merchantRegions.length || !regionList.value.length) return

  console.log("开始自动匹配区域", merchantRegions)
  const matchedRegionIds: string[] = []

  merchantRegions.forEach((merchantRegion) => {
    // 移除区域名称后面的数字
    const regionName = merchantRegion.region_name.replace(/\d+$/, "")

    // 先尝试匹配括号中的内容
    const bracketMatch = regionName.match(/（(.*?)）|\((.*?)\)/)
    const bracketContent = bracketMatch ? (bracketMatch[1] || bracketMatch[2]) : ""

    let matched = false

    if (bracketContent) {
      // 如果有括号内容，先用括号内容匹配
      regionList.value.forEach((region) => {
        if (region.LocalName.includes(bracketContent) && !matchedRegionIds.includes(region.RegionId)) {
          matchedRegionIds.push(region.RegionId)
          matched = true
        }
      })
    }

    // 如果没有通过括号内容匹配到，则使用整个区域名称进行匹配
    if (!matched) {
      regionList.value.forEach((region) => {
        if (region.LocalName.includes(regionName) && !matchedRegionIds.includes(region.RegionId)) {
          matchedRegionIds.push(region.RegionId)
        }
      })
    }
  })

  if (matchedRegionIds.length > 0) {
    console.log("自动匹配到区域:", matchedRegionIds)
    selectedRegions.value = matchedRegionIds
  }
}

// 区域变化处理
function handleRegionChange(value: string[] | undefined) {
  console.log("区域变更为:", value)
  bandwidthList.value = []
}

// 获取状态文本和类型处理
function getStatusText(status: string) {
  const statusMap: Record<string, string> = {
    Available: "可用",
    InUse: "已分配",
    Pending: "配置中",
    Associating: "绑定中",
    Unassociating: "解绑中"
  }
  return statusMap[status] || status
}

function getStatusType(status: string) {
  const typeMap: Record<string, "success" | "warning" | "info" | "primary" | "danger"> = {
    Available: "success",
    InUse: "primary",
    Pending: "warning",
    Associating: "warning",
    Unassociating: "warning"
  }
  return typeMap[status] || ""
}

// 获取计费方式文本
function getInternetChargeType(type: string) {
  const typeMap: Record<string, string> = {
    PayByBandwidth: "按带宽计费",
    PayByTraffic: "按流量计费"
  }
  return typeMap[type] || type
}

// 获取实例计费类型文本
function getInstanceChargeType(type: string) {
  const typeMap: Record<string, string> = {
    PostPaid: "按量计费",
    PrePaid: "包年包月"
  }
  return typeMap[type] || type
}

// 确认删除
function confirmDelete(bandwidth: Bandwidth) {
  ElMessageBox.confirm(
    `确定要删除共享带宽 ${bandwidth.Name || bandwidth.BandwidthPackageId} 吗？`,
    "删除确认",
    {
      confirmButtonText: "确定",
      cancelButtonText: "取消",
      type: "warning"
    }
  )
    .then(() => {
      handleOperate(bandwidth, "delete")
    })
    .catch(() => {
      // 用户取消删除操作
    })
}

// 操作共享带宽（修改、添加EIP、移除EIP、删除等）
async function handleOperate(bandwidth: Bandwidth, operation: string, additionalData: any = {}) {
  const hasAccount = accountType.value === "merchant" ? selectedMerchant.value : selectedCloudAccount.value
  if (!hasAccount) return

  try {
    const operateData: any = {
      region_id: bandwidth.RegionId,
      bandwidth_package_id: bandwidth.BandwidthPackageId,
      operation,
      name: bandwidth.Name || "",
      description: bandwidth.Description || "",
      bandwidth: bandwidth.Bandwidth || "",
      ...additionalData
    }

    if (accountType.value === "merchant") {
      operateData.merchant_id = selectedMerchant.value
    } else {
      operateData.cloud_account_id = selectedCloudAccount.value
    }

    await operateBandwidth(operateData)

    const operationText = {
      delete: "删除",
      modify: "修改",
      spec: "修改规格",
      addEip: "添加EIP",
      removeEip: "移除EIP"
    }[operation] || operation

    ElMessage.success(`${operationText}成功`)
    // 延迟刷新共享带宽列表，等待操作完成
    setTimeout(() => {
      fetchBandwidthList()
    }, 2000)
  } catch (error) {
    console.error(`操作失败`, error)
    ElMessage.error(`操作失败`)
  }
}

// 共享带宽详情
const detailDialogVisible = ref(false)
const detailBandwidth = ref<Bandwidth | null>(null)

// 显示共享带宽详情
function showBandwidthDetail(bandwidth: Bandwidth) {
  detailBandwidth.value = bandwidth
  detailDialogVisible.value = true
}

// 打开创建共享带宽对话框
function openCreateDialog() {
  const hasAccount = accountType.value === "merchant" ? selectedMerchant.value : selectedCloudAccount.value
  if (!hasAccount) {
    ElMessage.warning("请先选择账号")
    return
  }

  if (accountType.value === "merchant") {
    createBandwidthForm.merchant_id = selectedMerchant.value!
  } else {
    createBandwidthForm.cloud_account_id = selectedCloudAccount.value!
  }
  createBandwidthForm.region_id = ""
  createDialogVisible.value = true
}

// 提交创建共享带宽表单
const submitCreateForm = debounce(async (formEl: any) => {
  if (!formEl) return

  try {
    const valid = await formEl.validate().catch(() => false)
    if (!valid) return

    // 检查区域是否已选择
    if (!createBandwidthForm.region_id) {
      ElMessage.warning("请选择区域")
      return
    }

    createLoading.value = true
    await createBandwidth(createBandwidthForm)
    ElMessage.success("创建共享带宽成功")
    createDialogVisible.value = false

    // 刷新列表
    setTimeout(() => {
      fetchBandwidthList()
    }, 1000)
  } catch (error) {
    console.error("创建共享带宽失败", error)
    ElMessage.error("创建共享带宽失败")
  } finally {
    createLoading.value = false
  }
}, 500) // 500ms的防抖时间

// 重置创建表单
function resetCreateForm(formEl: any) {
  if (!formEl) return
  formEl.resetFields()
  createBandwidthForm.region_id = ""
  createBandwidthForm.bandwidth = 5
}

// 打开修改共享带宽对话框
function openModifyDialog(bandwidth: Bandwidth) {
  currentEditBandwidth.value = bandwidth
  modifyBandwidthForm.name = bandwidth.Name || ""
  modifyBandwidthForm.description = bandwidth.Description || ""
  modifyDialogVisible.value = true
}

// 提交修改共享带宽表单
const submitModifyForm = debounce(async (formEl: any) => {
  const hasAccount = accountType.value === "merchant" ? selectedMerchant.value : selectedCloudAccount.value
  if (!formEl || !currentEditBandwidth.value || !hasAccount || !selectedRegions.value || selectedRegions.value.length === 0) return

  try {
    const valid = await formEl.validate().catch(() => false)
    if (!valid) return

    const bandwidth = currentEditBandwidth.value
    const regionId = selectedRegions.value[0]

    modifyLoading.value = true

    const params: any = {
      region_id: regionId,
      bandwidth_package_id: bandwidth.BandwidthPackageId,
      operation: "modify",
      name: modifyBandwidthForm.name,
      description: modifyBandwidthForm.description,
      bandwidth: bandwidth.Bandwidth
    }

    if (accountType.value === "merchant") {
      params.merchant_id = selectedMerchant.value
    } else {
      params.cloud_account_id = selectedCloudAccount.value
    }

    await operateBandwidth(params)

    ElMessage.success("修改共享带宽成功")
    modifyDialogVisible.value = false

    // 刷新列表
    setTimeout(() => {
      fetchBandwidthList()
    }, 1000)
  } catch (error) {
    console.error("修改共享带宽失败", error)
    ElMessage.error("修改共享带宽失败")
  } finally {
    modifyLoading.value = false
  }
}, 500) // 500ms的防抖时间

// 修改带宽规格对话框
const modifySpecDialogVisible = ref(false)
const modifySpecLoading = ref(false)
const modifySpecFormRef = ref()
const modifySpecForm = reactive({
  bandwidth: ""
})
const currentEditSpecBandwidth = ref<Bandwidth | null>(null)

// 修改规格表单规则
const modifySpecFormRules = {
  bandwidth: [
    { required: true, message: "请输入带宽值", trigger: "blur" },
    { pattern: /^[1-9]\d*$/, message: "带宽必须为正整数", trigger: "blur" }
  ]
}

// 打开修改带宽规格对话框
function openModifySpecDialog(bandwidth: Bandwidth) {
  currentEditSpecBandwidth.value = bandwidth
  modifySpecForm.bandwidth = bandwidth.Bandwidth
  modifySpecDialogVisible.value = true
}

// 提交修改带宽规格表单
const submitModifySpecForm = debounce(async (formEl: any) => {
  const hasAccount = accountType.value === "merchant" ? selectedMerchant.value : selectedCloudAccount.value
  if (!formEl || !currentEditSpecBandwidth.value || !hasAccount || !selectedRegions.value || selectedRegions.value.length === 0) return

  try {
    const valid = await formEl.validate().catch(() => false)
    if (!valid) return

    const bandwidth = currentEditSpecBandwidth.value

    modifySpecLoading.value = true

    const params: any = {
      region_id: bandwidth.RegionId,
      bandwidth_package_id: bandwidth.BandwidthPackageId,
      operation: "spec",
      name: bandwidth.Name || "",
      description: bandwidth.Description || "",
      bandwidth: modifySpecForm.bandwidth
    }

    if (accountType.value === "merchant") {
      params.merchant_id = selectedMerchant.value
    } else {
      params.cloud_account_id = selectedCloudAccount.value
    }

    await operateBandwidth(params)

    ElMessage.success("修改共享带宽规格成功")
    modifySpecDialogVisible.value = false

    // 刷新列表
    setTimeout(() => {
      fetchBandwidthList()
    }, 1000)
  } catch (error) {
    console.error("修改共享带宽规格失败", error)
    ElMessage.error("修改共享带宽规格失败")
  } finally {
    modifySpecLoading.value = false
  }
}, 500) // 500ms的防抖时间

// 获取可用的弹性IP列表
async function fetchAvailableEips(regionId: string) {
  const hasAccount = accountType.value === "merchant" ? selectedMerchant.value : selectedCloudAccount.value
  if (!hasAccount) {
    ElMessage.warning("账号或区域信息缺失")
    return
  }

  loadingEips.value = true
  availableEipList.value = []

  try {
    const params: any = {
      region_id: [regionId]
    }

    if (accountType.value === "merchant") {
      params.merchant_id = selectedMerchant.value
    } else {
      params.cloud_account_id = selectedCloudAccount.value
    }

    const res = await getEipList(params)

    if (res.data.list && Array.isArray(res.data.list)) {
      // 过滤出可用的并且未加入共享带宽的EIP
      availableEipList.value = res.data.list.filter(eip =>
        (eip.Status === "Available" || eip.Status === "InUse") && !eip.BandwidthPackageId
      )
    }

    if (availableEipList.value.length === 0) {
      ElMessage.info("没有找到可添加的弹性IP")
    }
  } catch (error) {
    console.error("获取弹性IP列表失败", error)
    ElMessage.error("获取弹性IP列表失败")
  } finally {
    loadingEips.value = false
  }
}

// 打开添加弹性IP对话框
async function openAddEipDialog(bandwidth: Bandwidth) {
  const hasAccount = accountType.value === "merchant" ? selectedMerchant.value : selectedCloudAccount.value
  if (!hasAccount) {
    ElMessage.warning("账号或区域信息缺失")
    return
  }

  currentBandwidthForAddEip.value = bandwidth
  selectedEipIds.value = []
  addEipDialogVisible.value = true

  // 加载可用的弹性IP
  await fetchAvailableEips(bandwidth.RegionId)
}

// 提交添加弹性IP到共享带宽
const submitAddEipForm = debounce(async () => {
  const hasAccount = accountType.value === "merchant" ? selectedMerchant.value : selectedCloudAccount.value
  if (!currentBandwidthForAddEip.value || !selectedEipIds.value.length || !hasAccount) {
    ElMessage.warning("请选择要添加的弹性IP")
    return
  }

  addEipLoading.value = true
  try {
    const params: any = {
      region_id: currentBandwidthForAddEip.value.RegionId,
      bandwidth_package_id: currentBandwidthForAddEip.value.BandwidthPackageId,
      operation: "addEip",
      ip_instance_ids: selectedEipIds.value
    }

    if (accountType.value === "merchant") {
      params.merchant_id = selectedMerchant.value
    } else {
      params.cloud_account_id = selectedCloudAccount.value
    }

    await operateBandwidth(params)

    ElMessage.success("添加弹性IP成功")
    addEipDialogVisible.value = false

    // 刷新列表
    setTimeout(() => {
      fetchBandwidthList()
    }, 2000)
  } catch (error) {
    console.error("添加弹性IP失败", error)
    ElMessage.error("添加弹性IP失败")
  } finally {
    addEipLoading.value = false
  }
}, 500) // 500ms的防抖时间

// 移除弹性IP
function confirmRemoveEip(bandwidth: Bandwidth, eip: any) {
  ElMessageBox.confirm(
    `确定要从共享带宽中移除弹性IP ${eip.IpAddress} 吗？`,
    "移除确认",
    {
      confirmButtonText: "确定",
      cancelButtonText: "取消",
      type: "warning"
    }
  )
    .then(() => {
      handleOperate(bandwidth, "removeEip", { ip_instance_id: eip.AllocationId })
    })
    .catch(() => {
      // 用户取消操作
    })
}

// 页面加载时获取商户和区域列表
onMounted(() => {
  loadSelectionFromStorage()

  const fetchInitialData = () => {
    if (accountType.value === "merchant" && selectedMerchant.value) {
      fetchRegionList()
    } else if (accountType.value === "system" && selectedCloudAccount.value) {
      fetchRegionList()
    }
  }

  if (accountType.value === "system") {
    fetchCloudAccountList().then(fetchInitialData)
  } else {
    fetchMerchantList().then(fetchInitialData)
  }
})
</script>

<template>
  <div class="container">
    <el-card class="filter-card">
      <div class="filter-row">
        <div class="filter-item">
          <span class="label">账号类型：</span>
          <el-select
            v-model="accountType"
            style="width: 150px"
            @change="handleAccountTypeChange"
          >
            <el-option label="商户类型" value="merchant" />
            <el-option label="系统类型" value="system" />
          </el-select>
        </div>
        <div v-if="accountType === 'system'" class="filter-item">
          <span class="label">云账号：</span>
          <el-select
            v-model="selectedCloudAccount"
            placeholder="请选择云账号"
            clearable
            filterable
            style="width: 220px"
            @change="handleCloudAccountChange"
          >
            <template #prefix>
              <el-icon><Key /></el-icon>
            </template>
            <el-option
              v-for="item in cloudAccountList"
              :key="item.value"
              :label="item.label"
              :value="item.value"
            />
          </el-select>
        </div>
        <div v-if="accountType === 'merchant'" class="filter-item">
          <span class="label">商户：</span>
          <el-select
            v-model="selectedMerchant"
            placeholder="请选择商户"
            clearable
            filterable
            style="width: 220px"
            popper-class="merchant-select-dropdown"
            value-key="id"
            @change="handleMerchantChange"
          >
            <template #prefix>
              <el-icon><User /></el-icon>
            </template>
            <el-option
              v-for="item in merchantList"
              :key="item.id"
              :label="item.name"
              :value="item.id"
            >
              <div class="custom-option">
                <span>{{ item.name }}</span>
                <small v-if="item.status === 1" class="status active">正常</small>
                <small v-else class="status inactive">禁用</small>
              </div>
            </el-option>
          </el-select>
        </div>
        <div class="filter-item">
          <span class="label">区域：</span>
          <el-select
            v-model="selectedRegions"
            placeholder="请选择区域（可多选）"
            clearable
            filterable
            multiple
            collapse-tags
            collapse-tags-tooltip
            style="width: 300px"
            popper-class="region-select-dropdown"
            :disabled="accountType === 'merchant' ? !selectedMerchant : !selectedCloudAccount"
            @change="handleRegionChange"
          >
            <template #prefix>
              <el-icon><Location /></el-icon>
            </template>
            <el-option
              v-for="item in regionList"
              :key="item.RegionId"
              :label="item.LocalName"
              :value="item.RegionId"
            />
          </el-select>
        </div>
        <el-button
          type="primary"
          :disabled="(accountType === 'merchant' ? !selectedMerchant : !selectedCloudAccount) || !selectedRegions.length"
          @click="fetchBandwidthList"
        >
          查询
        </el-button>
        <el-button
          type="success"
          :disabled="accountType === 'merchant' ? !selectedMerchant : !selectedCloudAccount"
          @click="openCreateDialog"
        >
          创建共享带宽
        </el-button>
      </div>
    </el-card>

    <el-card v-if="(accountType === 'merchant' ? selectedMerchant : selectedCloudAccount) && selectedRegions.length > 0" class="table-card">
      <template #header>
        <div class="card-header">
          <span>共享带宽列表</span>
          <div class="operations">
            <el-button type="primary" size="small" @click="fetchBandwidthList">
              刷新
            </el-button>
          </div>
        </div>
      </template>
      <el-table
        v-loading="loading"
        :data="bandwidthList"
        style="width: 100%"
        border
      >
        <el-table-column prop="BandwidthPackageId" label="共享带宽 ID" min-width="180" />
        <el-table-column prop="Name" label="名称" min-width="150">
          <template #default="scope">
            {{ scope.row.Name || '-' }}
          </template>
        </el-table-column>
        <el-table-column label="区域" min-width="120">
          <template #default="scope">
            {{ scope.row.RegionId || "-" }}
          </template>
        </el-table-column>
        <el-table-column prop="Bandwidth" label="带宽(Mbps)" min-width="100" />
        <el-table-column prop="Status" label="状态" min-width="100">
          <template #default="scope">
            <el-tag :type="getStatusType(scope.row.Status)">
              {{ getStatusText(scope.row.Status) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="计费方式" min-width="120">
          <template #default="scope">
            {{ getInternetChargeType(scope.row.InternetChargeType) }}
          </template>
        </el-table-column>
        <el-table-column label="实例计费类型" min-width="120">
          <template #default="scope">
            {{ getInstanceChargeType(scope.row.InstanceChargeType) }}
          </template>
        </el-table-column>
        <el-table-column prop="PublicIpAddresses.PublicIpAddresse" label="包含IP数量" min-width="120">
          <template #default="scope">
            {{ scope.row.PublicIpAddresses?.PublicIpAddresse?.length || 0 }}
          </template>
        </el-table-column>
        <el-table-column prop="ExpiredTime" label="到期时间" min-width="180" />
        <el-table-column label="操作" fixed="right" min-width="120">
          <template #default="scope">
            <div class="action-buttons">
              <el-button
                type="primary"
                text
                size="small"
                @click="showBandwidthDetail(scope.row)"
              >
                详情
              </el-button>
              <el-dropdown>
                <span class="el-dropdown-link">
                  <el-button type="primary" text size="small">
                    更多<el-icon class="el-icon--right"><ArrowDown /></el-icon>
                  </el-button>
                </span>
                <template #dropdown>
                  <el-dropdown-menu>
                    <el-dropdown-item @click="openModifyDialog(scope.row)">
                      <el-icon><Edit /></el-icon> 修改
                    </el-dropdown-item>
                    <el-dropdown-item @click="openModifySpecDialog(scope.row)">
                      <el-icon><Edit /></el-icon> 修改规格
                    </el-dropdown-item>
                    <el-dropdown-item @click="openAddEipDialog(scope.row)">
                      <el-icon><Plus /></el-icon> 添加弹性IP
                    </el-dropdown-item>
                    <el-dropdown-item divided @click="confirmDelete(scope.row)">
                      <el-icon><Delete /></el-icon> 删除
                    </el-dropdown-item>
                  </el-dropdown-menu>
                </template>
              </el-dropdown>
            </div>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-empty v-else description="请先选择商户和区域" />

    <!-- 共享带宽详情对话框 -->
    <el-dialog
      v-model="detailDialogVisible"
      :title="`共享带宽详情 - ${detailBandwidth?.Name || detailBandwidth?.BandwidthPackageId || ''}`"
      width="750px"
      destroy-on-close
    >
      <div v-if="detailBandwidth" class="bandwidth-detail">
        <el-descriptions :column="2" border>
          <el-descriptions-item label="共享带宽 ID" :span="2">
            {{ detailBandwidth.BandwidthPackageId }}
          </el-descriptions-item>
          <el-descriptions-item label="名称">
            {{ detailBandwidth.Name || '-' }}
          </el-descriptions-item>
          <el-descriptions-item label="状态">
            <el-tag :type="getStatusType(detailBandwidth.Status)">
              {{ getStatusText(detailBandwidth.Status) }}
            </el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="带宽">
            {{ detailBandwidth.Bandwidth ? `${detailBandwidth.Bandwidth} Mbps` : '-' }}
          </el-descriptions-item>
          <el-descriptions-item label="计费方式">
            {{ getInternetChargeType(detailBandwidth.InternetChargeType) }}
          </el-descriptions-item>
          <el-descriptions-item label="实例计费类型">
            {{ getInstanceChargeType(detailBandwidth.InstanceChargeType) }}
          </el-descriptions-item>
          <el-descriptions-item label="到期时间">
            {{ detailBandwidth.ExpiredTime || '-' }}
          </el-descriptions-item>
          <el-descriptions-item label="区域">
            {{ detailBandwidth.RegionId }}
          </el-descriptions-item>
        </el-descriptions>

        <!-- 续费信息 -->
        <div v-if="detailBandwidth.ReservationOrderType" class="detail-section">
          <h3 class="detail-title">
            续费信息
          </h3>
          <el-descriptions :column="2" border>
            <el-descriptions-item label="续费订单类型">
              {{ {
                RENEWCHANGE: '续费变配',
                TEMP_UPGRADE: '短时升配',
                UPGRADE: '升级',
              }[detailBandwidth.ReservationOrderType] || detailBandwidth.ReservationOrderType }}
            </el-descriptions-item>
            <el-descriptions-item label="续费带宽">
              {{ detailBandwidth.ReservationBandwidth ? `${detailBandwidth.ReservationBandwidth} Mbps` : '-' }}
            </el-descriptions-item>
            <el-descriptions-item label="续费生效时间">
              {{ detailBandwidth.ReservationActiveTime || '-' }}
            </el-descriptions-item>
            <el-descriptions-item label="续费计费方式">
              {{ getInternetChargeType(detailBandwidth.ReservationInternetChargeType || '') }}
            </el-descriptions-item>
          </el-descriptions>
        </div>

        <!-- 包含的EIP列表 -->
        <div class="detail-section">
          <h3 class="detail-title">
            绑定的弹性公网IP
          </h3>
          <div v-if="detailBandwidth.PublicIpAddresses?.PublicIpAddresse?.length">
            <el-table :data="detailBandwidth.PublicIpAddresses.PublicIpAddresse" style="width: 100%" border>
              <el-table-column prop="IpAddress" label="IP地址" />
              <el-table-column prop="AllocationId" label="弹性公网IP ID" />
              <el-table-column prop="BandwidthPackageIpRelationStatus" label="关联状态">
                <template #default="scope">
                  <el-tag :type="scope.row.BandwidthPackageIpRelationStatus === 'BINDED' ? 'success' : 'warning'">
                    {{ scope.row.BandwidthPackageIpRelationStatus === 'BINDED' ? '已关联' : '关联中' }}
                  </el-tag>
                </template>
              </el-table-column>
              <el-table-column label="操作" width="100">
                <template #default="scope">
                  <el-button
                    type="danger"
                    text
                    size="small"
                    @click="confirmRemoveEip(detailBandwidth, scope.row)"
                  >
                    移除
                  </el-button>
                </template>
              </el-table-column>
            </el-table>
          </div>
          <el-empty v-else description="暂无绑定的弹性公网IP" :image-size="60" />
        </div>

        <!-- 其他信息 -->
        <div class="detail-section">
          <h3 class="detail-title">
            其他信息
          </h3>
          <el-descriptions :column="2" border>
            <el-descriptions-item label="描述" :span="2">
              {{ detailBandwidth.Description || '-' }}
            </el-descriptions-item>
            <el-descriptions-item label="可用区" v-if="detailBandwidth.Zone">
              {{ detailBandwidth.Zone }}
            </el-descriptions-item>
          </el-descriptions>
        </div>
      </div>
      <el-empty v-else description="未找到共享带宽详情" />
    </el-dialog>

    <!-- 创建共享带宽对话框 -->
    <el-dialog
      v-model="createDialogVisible"
      title="创建共享带宽"
      width="600px"
      destroy-on-close
    >
      <el-form
        ref="createFormRef"
        :model="createBandwidthForm"
        :rules="createFormRules"
        label-width="120px"
        label-position="right"
      >
        <el-form-item label="区域">
          <el-select
            v-model="createBandwidthForm.region_id"
            placeholder="请选择区域"
            style="width: 100%"
            :disabled="!regionList.length"
          >
            <el-option
              v-for="item in regionList"
              :key="item.RegionId"
              :label="item.LocalName"
              :value="item.RegionId"
            />
          </el-select>
        </el-form-item>

        <el-form-item label="带宽" prop="bandwidth">
          <el-input
            v-model.number="createBandwidthForm.bandwidth"
            placeholder="请输入带宽值"
            style="width: 200px"
            type="number"
            min="5"
          >
            <template #append>
              Mbps
            </template>
          </el-input>
          <div class="form-tip">
            带宽范围：5-1000Mbps
          </div>
        </el-form-item>
      </el-form>

      <template #footer>
        <div class="dialog-footer">
          <el-button @click="createDialogVisible = false">
            取消
          </el-button>
          <el-button @click="resetCreateForm(createFormRef)">
            重置
          </el-button>
          <el-button
            type="primary"
            :loading="createLoading"
            @click="submitCreateForm(createFormRef)"
          >
            创建
          </el-button>
        </div>
      </template>
    </el-dialog>

    <!-- 修改共享带宽对话框 -->
    <el-dialog
      v-model="modifyDialogVisible"
      title="修改共享带宽"
      width="600px"
      destroy-on-close
    >
      <el-form
        ref="modifyFormRef"
        :model="modifyBandwidthForm"
        :rules="modifyFormRules"
        label-width="120px"
        label-position="right"
      >
        <el-form-item label="名称">
          <el-input
            v-model="modifyBandwidthForm.name"
            placeholder="请输入名称"
            style="width: 200px"
          />
        </el-form-item>

        <el-form-item label="描述">
          <el-input
            v-model="modifyBandwidthForm.description"
            placeholder="请输入描述"
            style="width: 200px"
          />
        </el-form-item>
      </el-form>

      <template #footer>
        <div class="dialog-footer">
          <el-button @click="modifyDialogVisible = false">
            取消
          </el-button>
          <el-button
            type="primary"
            :loading="modifyLoading"
            @click="submitModifyForm(modifyFormRef)"
          >
            修改
          </el-button>
        </div>
      </template>
    </el-dialog>

    <!-- 修改带宽规格对话框 -->
    <el-dialog
      v-model="modifySpecDialogVisible"
      title="修改带宽规格"
      width="600px"
      destroy-on-close
    >
      <el-form
        ref="modifySpecFormRef"
        :model="modifySpecForm"
        :rules="modifySpecFormRules"
        label-width="120px"
        label-position="right"
      >
        <el-form-item label="带宽" prop="bandwidth">
          <el-input
            v-model.number="modifySpecForm.bandwidth"
            placeholder="请输入带宽值"
            style="width: 200px"
            type="number"
            min="5"
          >
            <template #append>
              Mbps
            </template>
          </el-input>
          <div class="form-tip">
            带宽范围：5-1000Mbps
          </div>
        </el-form-item>
      </el-form>

      <template #footer>
        <div class="dialog-footer">
          <el-button @click="modifySpecDialogVisible = false">
            取消
          </el-button>
          <el-button
            type="primary"
            :loading="modifySpecLoading"
            @click="submitModifySpecForm(modifySpecFormRef)"
          >
            修改
          </el-button>
        </div>
      </template>
    </el-dialog>

    <!-- 添加弹性IP对话框 -->
    <el-dialog
      v-model="addEipDialogVisible"
      title="添加弹性IP到共享带宽"
      width="700px"
      destroy-on-close
    >
      <div v-loading="loadingEips" class="eip-select-container">
        <div v-if="availableEipList.length > 0">
          <p class="dialog-subtitle">
            选择要添加到共享带宽的弹性IP
          </p>
          <el-table
            :data="availableEipList"
            style="width: 100%"
            border
            @selection-change="(val: Eip[]) => selectedEipIds = val.map(item => item.AllocationId)"
          >
            <el-table-column type="selection" width="55" />
            <el-table-column prop="IpAddress" label="IP地址" />
            <el-table-column prop="AllocationId" label="弹性公网IP ID" />
            <el-table-column prop="Status" label="状态">
              <template #default="scope">
                <el-tag :type="getStatusType(scope.row.Status)">
                  {{ getStatusText(scope.row.Status) }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="Bandwidth" label="带宽(Mbps)" />
            <el-table-column prop="InstanceId" label="绑定实例">
              <template #default="scope">
                {{ scope.row.InstanceId || '-' }}
              </template>
            </el-table-column>
          </el-table>
        </div>
        <el-empty v-else description="没有找到可用的弹性IP" :image-size="80" />
      </div>

      <template #footer>
        <div class="dialog-footer">
          <el-button @click="addEipDialogVisible = false">
            取消
          </el-button>
          <el-button
            type="primary"
            :loading="addEipLoading"
            :disabled="!selectedEipIds.length"
            @click="submitAddEipForm"
          >
            添加
          </el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
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
  margin-right: 16px;
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

.custom-option {
  display: flex;
  justify-content: space-between;
  align-items: center;
  width: 100%;
}

.status {
  font-size: 12px;
  padding: 2px 6px;
  border-radius: 4px;
}

.status.active {
  color: #67c23a;
  background-color: #f0f9eb;
}

.status.inactive {
  color: #f56c6c;
  background-color: #fef0f0;
}

.bandwidth-detail {
  padding: 10px;
}

.detail-section {
  margin-top: 24px;
}

.detail-title {
  margin-bottom: 16px;
  font-size: 16px;
  font-weight: 500;
  color: #303133;
  border-left: 3px solid #409eff;
  padding-left: 10px;
}

.action-buttons {
  display: flex;
  align-items: center;
  gap: 8px;
}

.form-tip {
  font-size: 12px;
  color: #909399;
  margin-top: 4px;
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}

.el-dropdown-link {
  cursor: pointer;
  color: #409eff;
  display: flex;
  align-items: center;
}

.dialog-subtitle {
  margin-bottom: 16px;
  color: #606266;
  font-size: 14px;
}

.eip-select-container {
  min-height: 250px;
}
</style>
