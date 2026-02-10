<script setup lang="ts">
import type { Region } from "@/pages/cloud/aliyun/apis/type"
import type { Bandwidth } from "@/pages/cloud/aliyun/bandwidth/apis/type"
import type { BatchReplaceEipConfig, Eip, EipItem } from "@/pages/cloud/aliyun/eip/apis/type"
import type { Instance } from "@/pages/cloud/aliyun/instances/apis/type"
import type { NetworkInterfaceWrap } from "@/pages/cloud/aliyun/network/apis/type"
import type { Merchant, MerchantRegions } from "@/pages/dashboard/apis/type"
import type { CloudAccountOption } from "@@/apis/cloud_account/type"
import { regionListApi } from "@/pages/cloud/aliyun/apis"
import { getBandwidthList, operateBandwidth } from "@/pages/cloud/aliyun/bandwidth/apis"
import { batchAssociateEip, batchReplaceEip, getEipList, operateEip, replaceEip } from "@/pages/cloud/aliyun/eip/apis"
import { getInstanceList } from "@/pages/cloud/aliyun/instances/apis"
import { getNetworkInterfaceList } from "@/pages/cloud/aliyun/network/apis"
import { merchantQueryApi } from "@/pages/dashboard/apis"
import { getCloudAccountOptions } from "@@/apis/cloud_account"
import { ArrowDown, Delete, Edit, Key, Link, Location, Monitor, User } from "@element-plus/icons-vue"
import { ElMessage, ElMessageBox } from "element-plus"
import { onMounted, reactive, ref, watch } from "vue"
import { useRouter } from "vue-router"

defineOptions({
  name: "CloudEIP"
})

// 初始化路由
const router = useRouter()

// localStorage 存储键名
const STORAGE_ACCOUNT_TYPE_KEY = "cloud_account_type"
const STORAGE_CLOUD_ACCOUNT_KEY = "cloud_selected_cloud_account"
const STORAGE_MERCHANT_KEY = "cloud_selected_merchant"
const STORAGE_REGION_KEY = "cloud_selected_region"

// 数据状态
const loading = ref(false)
const accountType = ref<string>("merchant") // merchant: 商户类型, system: 系统类型
const cloudAccountList = ref<CloudAccountOption[]>([])
const selectedCloudAccount = ref<number>()
const merchantList = ref<Merchant[]>([])
const regionList = ref<Region[]>([])
const eipList = ref<Eip[]>([])
const selectedMerchant = ref<number>()
const selectedRegions = ref<string[]>([])
// 选中的EIP列表用于批量操作
const selectedEips = ref<Eip[]>([])

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
  eipList.value = []

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
  eipList.value = []

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

// 获取弹性IP列表
async function fetchEipList() {
  const hasAccount = accountType.value === "merchant" ? selectedMerchant.value : selectedCloudAccount.value
  if (!hasAccount || !selectedRegions.value || selectedRegions.value.length === 0) {
    console.log("账号或区域未选择，不获取弹性IP列表")
    return
  }

  console.log("开始获取弹性IP列表, 类型:", accountType.value, "账号:", hasAccount, "区域:", selectedRegions.value)
  loading.value = true
  try {
    const params: any = { region_id: selectedRegions.value }
    if (accountType.value === "merchant") {
      params.merchant_id = selectedMerchant.value
    } else {
      params.cloud_account_id = selectedCloudAccount.value
    }

    const res = await getEipList(params)

    console.log("获取弹性IP列表成功:", res.data.list)
    // 兼容返回空数组的情况
    if (!res.data.list || !Array.isArray(res.data.list)) {
      console.log("弹性IP列表为空或格式不正确，设置为空数组")
      eipList.value = []
    } else {
      eipList.value = res.data.list
    }

    // 清空选中的EIP
    selectedEips.value = []

    // 如果列表为空，显示提示
    if (eipList.value.length === 0) {
      ElMessage.info("当前区域暂无弹性IP")
    }
  } catch (error) {
    console.error("获取弹性IP列表失败", error)
    ElMessage.error("获取弹性IP列表失败")
    eipList.value = []
  } finally {
    loading.value = false
  }
}

// 商户变化处理
function handleMerchantChange(value: number | undefined) {
  console.log("商户变更为:", value)
  selectedRegions.value = []
  eipList.value = []
  selectedEips.value = []

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
  eipList.value = []
}

// 获取状态文本和类型处理
function getStatusText(status: string) {
  const statusMap: Record<string, string> = {
    Associating: "绑定中",
    Unassociating: "解绑中",
    InUse: "已分配",
    Available: "可用",
    Releasing: "释放中"
  }
  return statusMap[status] || status
}

function getStatusType(status: string) {
  const typeMap: Record<string, "success" | "warning" | "info" | "primary" | "danger"> = {
    Associating: "warning",
    Unassociating: "warning",
    InUse: "success",
    Available: "info",
    Releasing: "danger"
  }
  return typeMap[status] || ""
}

function getInstanceTypeText(type: string) {
  const typeMap: Record<string, string> = {
    EcsInstance: "ECS实例",
    SlbInstance: "CLB实例",
    Nat: "NAT网关",
    HaVip: "高可用虚拟IP",
    NetworkInterface: "辅助弹性网卡",
    IpAddress: "IP地址"
  }
  return typeMap[type] || type
}

// 获取网络计费类型文本
function getInternetChargeType(type: string) {
  const typeMap: Record<string, string> = {
    PayByBandwidth: "按带宽计费",
    PayByTraffic: "按流量计费"
  }
  return typeMap[type] || type
}

// 确认删除
function confirmDelete(eip: Eip) {
  ElMessageBox.confirm(
    `确定要删除弹性IP ${eip.IpAddress} 吗？`,
    "删除确认",
    {
      confirmButtonText: "确定",
      cancelButtonText: "取消",
      type: "warning"
    }
  )
    .then(() => {
      handleOperate(eip, "delete")
    })
    .catch(() => {
      // 用户取消删除操作
    })
}

// 确认解绑实例
function confirmUnassociate(eip: Eip) {
  ElMessageBox.confirm(
    `确定要解绑弹性IP ${eip.IpAddress} 吗？`,
    "解绑确认",
    {
      confirmButtonText: "确定",
      cancelButtonText: "取消",
      type: "warning"
    }
  )
    .then(() => {
      handleOperate(eip, "unassociate")
    })
    .catch(() => {
      // 用户取消解绑操作
    })
}

// 操作弹性IP（修改、绑定、解绑、删除）
async function handleOperate(eip: Eip, operation: string) {
  const hasAccount = accountType.value === "merchant" ? selectedMerchant.value : selectedCloudAccount.value
  if (!hasAccount) return

  try {
    const params: any = {
      region_id: eip.RegionId,
      allocation_id: eip.AllocationId,
      operation,
      instance_type: eip.InstanceType || "",
      instance_id: eip.InstanceId || "",
      name: eip.Name || "",
      description: eip.Description || ""
    }

    if (accountType.value === "merchant") {
      params.merchant_id = selectedMerchant.value
    } else {
      params.cloud_account_id = selectedCloudAccount.value
    }

    await operateEip(params)

    const operationText = {
      delete: "删除",
      unassociate: "解绑",
      associate: "绑定",
      modify: "修改"
    }[operation] || operation

    ElMessage.success(`${operationText}成功`)
    // 延迟刷新弹性IP列表，等待操作完成
    setTimeout(() => {
      fetchEipList()
    }, 2000)
  } catch (error) {
    console.error(`操作失败`, error)
    ElMessage.error(`操作失败`)
  }
}

// 弹性IP详情
const detailDialogVisible = ref(false)
const detailEip = ref<Eip | null>(null)

// 显示弹性IP详情
function showEipDetail(eip: Eip) {
  detailEip.value = eip
  detailDialogVisible.value = true
}

// 打开创建弹性IP对话框
function openCreateDialog() {
  const hasAccount = accountType.value === "merchant" ? selectedMerchant.value : selectedCloudAccount.value
  if (!hasAccount) {
    ElMessage.warning("请先选择账号")
    return
  }

  const query: any = {}
  if (accountType.value === "merchant") {
    query.merchant_id = selectedMerchant.value?.toString()
  } else {
    query.cloud_account_id = selectedCloudAccount.value?.toString()
  }

  router.push({
    path: "/cloud/aliyun/eip/create",
    query
  })
}

// 实例详情
const instanceDialogVisible = ref(false)
const instanceLoading = ref(false)
const instanceDetail = ref<Instance | null>(null)

// 查看绑定的ECS实例详情
async function viewEcsInstanceDetail(eip: Eip) {
  const hasAccount = accountType.value === "merchant" ? selectedMerchant.value : selectedCloudAccount.value
  if (!hasAccount || !selectedRegions.value || selectedRegions.value.length === 0) {
    ElMessage.warning("账号或区域信息缺失")
    return
  }

  if (!eip.InstanceId || eip.InstanceType !== "EcsInstance") {
    ElMessage.warning("该弹性IP未绑定ECS实例")
    return
  }

  instanceLoading.value = true
  instanceDetail.value = null
  instanceDialogVisible.value = true

  try {
    const params: any = {
      region_id: selectedRegions.value
    }

    if (accountType.value === "merchant") {
      params.merchant_id = selectedMerchant.value
    } else {
      params.cloud_account_id = selectedCloudAccount.value
    }

    const res = await getInstanceList(params)

    if (res.data.list && Array.isArray(res.data.list)) {
      const instance = res.data.list.find(item => item.InstanceId === eip.InstanceId)
      if (instance) {
        instanceDetail.value = instance
      } else {
        ElMessage.warning(`未找到ID为 ${eip.InstanceId} 的实例信息`)
      }
    }
  } catch (error) {
    console.error("获取实例详情失败", error)
    ElMessage.error("获取实例详情失败")
  } finally {
    instanceLoading.value = false
  }
}

// 查看绑定的辅助网卡详情
function viewNetworkInterfaceDetail(eip: Eip) {
  if (!selectedMerchant.value) {
    ElMessage.warning("商户或区域信息缺失")
    return
  }

  if (!eip.InstanceId || eip.InstanceType !== "NetworkInterface") {
    ElMessage.warning("该弹性IP未绑定辅助网卡")
    return
  }

  // 使用路由导航到网卡管理页面
  router.push({
    path: "/cloud/network",
    query: {
      merchant_id: selectedMerchant.value.toString(),
      network_interface_id: eip.InstanceId,
      region_id: [eip.RegionId]
    }
  })
}

// 修改弹性IP对话框
const modifyDialogVisible = ref(false)
const modifyLoading = ref(false)
const modifyFormRef = ref()
const modifyEipForm = reactive({
  name: "",
  description: ""
})
const currentEditEip = ref<Eip | null>(null)

// 修改表单规则
const modifyFormRules = {
  name: [
    { max: 128, message: "名称最大长度为128个字符", trigger: "blur" }
  ],
  description: [
    { max: 256, message: "描述最大长度为256个字符", trigger: "blur" }
  ]
}

// 打开修改弹性IP对话框
function openModifyDialog(eip: Eip) {
  currentEditEip.value = eip
  modifyEipForm.name = eip.Name || ""
  modifyEipForm.description = eip.Description || ""
  modifyDialogVisible.value = true
}

// 提交修改弹性IP表单
async function submitModifyForm(formEl: any) {
  const hasAccount = accountType.value === "merchant" ? selectedMerchant.value : selectedCloudAccount.value
  if (!formEl || !currentEditEip.value || !hasAccount || !selectedRegions.value || selectedRegions.value.length === 0) return

  const eip = currentEditEip.value
  const regionId = selectedRegions.value[0]

  await formEl.validate(async (valid: boolean) => {
    if (valid) {
      modifyLoading.value = true
      try {
        const params: any = {
          region_id: regionId,
          allocation_id: eip.AllocationId,
          operation: "modify",
          instance_type: eip.InstanceType || "",
          instance_id: eip.InstanceId || "",
          name: modifyEipForm.name,
          description: modifyEipForm.description
        }

        if (accountType.value === "merchant") {
          params.merchant_id = selectedMerchant.value
        } else {
          params.cloud_account_id = selectedCloudAccount.value
        }

        await operateEip(params)

        ElMessage.success("修改弹性IP成功")
        modifyDialogVisible.value = false

        // 刷新列表
        setTimeout(() => {
          fetchEipList()
        }, 1000)
      } catch (error) {
        console.error("修改弹性IP失败", error)
        ElMessage.error("修改弹性IP失败")
      } finally {
        modifyLoading.value = false
      }
    }
  })
}

// 绑定对话框
const bindDialogVisible = ref(false)
const bindLoading = ref(false)
const bindInstanceType = ref<"EcsInstance" | "NetworkInterface">("EcsInstance")
const bindInstanceId = ref("")
const currentBindEip = ref<Eip | null>(null)
const instanceListForBind = ref<Instance[]>([])
const networkListForBind = ref<NetworkInterfaceWrap[]>([])
const loadingInstancesForBind = ref(false)
const loadingNetworksForBind = ref(false)

// 打开绑定对话框
async function openBindDialog(eip: Eip) {
  const hasAccount = accountType.value === "merchant" ? selectedMerchant.value : selectedCloudAccount.value
  if (!hasAccount || !selectedRegions.value || selectedRegions.value.length === 0) {
    ElMessage.warning("账号或区域信息缺失")
    return
  }

  if (eip.Status !== "Available") {
    ElMessage.warning("只能对可用状态的弹性IP进行绑定")
    return
  }

  currentBindEip.value = eip
  bindInstanceType.value = "EcsInstance"
  bindInstanceId.value = ""
  bindDialogVisible.value = true

  // 默认加载ECS实例列表
  await fetchInstancesForBind(eip.RegionId)
}

// 获取用于绑定的ECS实例列表
async function fetchInstancesForBind(regionId: string) {
  const hasAccount = accountType.value === "merchant" ? selectedMerchant.value : selectedCloudAccount.value
  if (!hasAccount) return

  loadingInstancesForBind.value = true
  instanceListForBind.value = []

  try {
    const params: any = {
      region_id: [regionId]
    }

    if (accountType.value === "merchant") {
      params.merchant_id = selectedMerchant.value
    } else {
      params.cloud_account_id = selectedCloudAccount.value
    }

    const res = await getInstanceList(params)

    instanceListForBind.value = res.data.list

    if (instanceListForBind.value.length === 0) {
      ElMessage.info("当前区域没有可用的ECS实例")
    }
  } catch (error) {
    console.error("获取ECS实例列表失败", error)
    ElMessage.error("获取ECS实例列表失败")
  } finally {
    loadingInstancesForBind.value = false
  }
}

// 获取用于绑定的辅助弹性网卡列表
async function fetchNetworksForBind(regionId: string) {
  const hasAccount = accountType.value === "merchant" ? selectedMerchant.value : selectedCloudAccount.value
  if (!hasAccount || !selectedRegions.value || selectedRegions.value.length === 0) return

  loadingNetworksForBind.value = true
  networkListForBind.value = []

  try {
    const params: any = {
      region_id: [regionId]
    }

    if (accountType.value === "merchant") {
      params.merchant_id = selectedMerchant.value
    } else {
      params.cloud_account_id = selectedCloudAccount.value
    }

    const res = await getNetworkInterfaceList(params)

    if (res.data.list && Array.isArray(res.data.list)) {
      // 只保留辅助网卡（Type为Secondary）
      networkListForBind.value = res.data.list.filter(item =>
        item.NetworkInterface && item.NetworkInterface.Type === "Secondary"
      )
    }

    if (networkListForBind.value.length === 0) {
      ElMessage.info("当前区域没有可用的弹性网卡")
    }
  } catch (error) {
    console.error("获取弹性网卡列表失败", error)
    ElMessage.error("获取弹性网卡列表失败")
  } finally {
    loadingNetworksForBind.value = false
  }
}

// 实例类型变化处理
async function handleInstanceTypeChange(val: string | number | boolean | undefined) {
  bindInstanceId.value = ""

  const type = val as "EcsInstance" | "NetworkInterface"

  if (type === "EcsInstance") {
    await fetchInstancesForBind(currentBindEip.value?.RegionId || "")
  } else {
    await fetchNetworksForBind(currentBindEip.value?.RegionId || "")
  }
}

// 提交绑定表单
async function submitBindForm() {
  const hasAccount = accountType.value === "merchant" ? selectedMerchant.value : selectedCloudAccount.value
  if (!currentBindEip.value || !hasAccount) {
    ElMessage.warning("绑定信息不完整")
    return
  }

  if (!bindInstanceId.value) {
    ElMessage.warning("请选择要绑定的实例")
    return
  }

  bindLoading.value = true
  try {
    const params: any = {
      region_id: currentBindEip.value.RegionId,
      allocation_id: currentBindEip.value.AllocationId,
      operation: "associate",
      instance_type: bindInstanceType.value,
      instance_id: bindInstanceId.value,
      name: currentBindEip.value.Name || "",
      description: currentBindEip.value.Description || ""
    }

    if (accountType.value === "merchant") {
      params.merchant_id = selectedMerchant.value
    } else {
      params.cloud_account_id = selectedCloudAccount.value
    }

    await operateEip(params)

    ElMessage.success("绑定成功")
    bindDialogVisible.value = false

    // 延迟刷新弹性IP列表，等待操作完成
    setTimeout(() => {
      fetchEipList()
    }, 2000)
  } catch (error) {
    console.error("绑定失败", error)
    ElMessage.error("绑定失败")
  } finally {
    bindLoading.value = false
  }
}

// 加入共享带宽对话框
const joinBandwidthDialogVisible = ref(false)
const joinBandwidthLoading = ref(false)
const bandwidthListForJoin = ref<Bandwidth[]>([])
const loadingBandwidthForJoin = ref(false)
const selectedBandwidthId = ref("")
const currentJoinEip = ref<Eip | null>(null)

// 获取可用的共享带宽列表
async function fetchBandwidthListForJoin() {
  const hasAccount = accountType.value === "merchant" ? selectedMerchant.value : selectedCloudAccount.value
  if (!hasAccount) return

  loadingBandwidthForJoin.value = true
  bandwidthListForJoin.value = []

  try {
    const params: any = {
      region_id: [currentJoinEip.value?.RegionId || ""]
    }

    if (accountType.value === "merchant") {
      params.merchant_id = selectedMerchant.value
    } else {
      params.cloud_account_id = selectedCloudAccount.value
    }

    const res = await getBandwidthList(params)

    if (res.data.list && Array.isArray(res.data.list)) {
      // 只要状态正常的共享带宽
      bandwidthListForJoin.value = res.data.list.filter(item =>
        item.Status === "Available" || item.Status === "InUse"
      )
    }

    if (bandwidthListForJoin.value.length === 0) {
      ElMessage.info("当前区域没有可用的共享带宽")
    }
  } catch (error) {
    console.error("获取共享带宽列表失败", error)
    ElMessage.error("获取共享带宽列表失败")
  } finally {
    loadingBandwidthForJoin.value = false
  }
}

// 打开加入共享带宽对话框
async function openJoinBandwidthDialog(eip: Eip) {
  const hasAccount = accountType.value === "merchant" ? selectedMerchant.value : selectedCloudAccount.value
  if (!hasAccount) {
    ElMessage.warning("账号或区域信息缺失")
    return
  }

  // 已经在共享带宽中的EIP不能再加入
  if (eip.BandwidthPackageId) {
    ElMessage.warning("该弹性IP已经加入了共享带宽")
    return
  }

  currentJoinEip.value = eip
  selectedBandwidthId.value = ""
  joinBandwidthDialogVisible.value = true

  // 加载可用的共享带宽列表
  await fetchBandwidthListForJoin()
}

// 提交加入共享带宽表单
async function submitJoinBandwidthForm() {
  const hasAccount = accountType.value === "merchant" ? selectedMerchant.value : selectedCloudAccount.value
  if (!currentJoinEip.value || !selectedBandwidthId.value
    || !hasAccount || !selectedRegions.value || selectedRegions.value.length === 0) {
    ElMessage.warning("请选择要加入的共享带宽")
    return
  }

  joinBandwidthLoading.value = true
  try {
    const params: any = {
      region_id: selectedRegions.value[0],
      bandwidth_package_id: selectedBandwidthId.value,
      operation: "addEip",
      ip_instance_ids: [currentJoinEip.value.AllocationId]
    }

    if (accountType.value === "merchant") {
      params.merchant_id = selectedMerchant.value
    } else {
      params.cloud_account_id = selectedCloudAccount.value
    }

    await operateBandwidth(params)

    ElMessage.success("加入共享带宽成功")
    joinBandwidthDialogVisible.value = false

    // 刷新列表
    setTimeout(() => {
      fetchEipList()
    }, 2000)
  } catch (error) {
    console.error("加入共享带宽失败", error)
    ElMessage.error("加入共享带宽失败")
  } finally {
    joinBandwidthLoading.value = false
  }
}

// 确认离开共享带宽
function confirmLeaveBandwidth(eip: Eip) {
  if (!eip.BandwidthPackageId) {
    ElMessage.warning("该弹性IP未加入共享带宽")
    return
  }

  ElMessageBox.confirm(
    `确定要将弹性IP ${eip.IpAddress} 从共享带宽中移除吗？`,
    "移除确认",
    {
      confirmButtonText: "确定",
      cancelButtonText: "取消",
      type: "warning"
    }
  )
    .then(() => {
      handleLeaveBandwidth(eip)
    })
    .catch(() => {
      // 用户取消操作
    })
}

// 处理离开共享带宽
async function handleLeaveBandwidth(eip: Eip) {
  const hasAccount = accountType.value === "merchant" ? selectedMerchant.value : selectedCloudAccount.value
  if (!eip.BandwidthPackageId || !hasAccount || !selectedRegions.value || selectedRegions.value.length === 0) return

  try {
    const params: any = {
      region_id: selectedRegions.value[0],
      bandwidth_package_id: eip.BandwidthPackageId,
      operation: "removeEip",
      ip_instance_id: eip.AllocationId
    }

    if (accountType.value === "merchant") {
      params.merchant_id = selectedMerchant.value
    } else {
      params.cloud_account_id = selectedCloudAccount.value
    }

    await operateBandwidth(params)

    ElMessage.success("已从共享带宽中移除")
    // 延迟刷新列表
    setTimeout(() => {
      fetchEipList()
    }, 2000)
  } catch (error) {
    console.error("从共享带宽中移除失败", error)
    ElMessage.error("从共享带宽中移除失败")
  }
}

// 处理EIP选择变化
function handleSelectionChange(selection: Eip[]) {
  selectedEips.value = selection
  console.log("已选择的EIP:", selection)
}

// 批量智能绑定相关状态
const batchBindDialogVisible = ref(false)
const batchBindLoading = ref(false)
const batchBindOutput = ref("")

// 更换弹性IP相关状态
const replaceDialogVisible = ref(false)
const replaceLoading = ref(false)
const replaceOutput = ref("")
const currentReplaceEip = ref<Eip | null>(null)

// 批量更换弹性IP相关状态
const batchReplaceDialogVisible = ref(false)
const batchReplaceLoading = ref(false)
const batchReplaceOutput = ref("")

// 打开批量智能绑定对话框
function openBatchBindDialog() {
  if (selectedEips.value.length === 0) {
    ElMessage.warning("请先选择需要绑定的弹性IP")
    return
  }

  // 检查是否有已绑定的EIP
  const boundEips = selectedEips.value.filter(eip => eip.Status !== "Available")
  if (boundEips.length > 0) {
    ElMessage.warning("只能对可用状态的弹性IP进行绑定，请重新选择")
    return
  }

  // 弹出确认对话框
  ElMessageBox.confirm(
    `确定为选中的 ${selectedEips.value.length} 个弹性IP智能绑定到对应区域的实例或网卡吗？`,
    "批量智能绑定",
    {
      confirmButtonText: "确定",
      cancelButtonText: "取消",
      type: "warning"
    }
  )
    .then(() => {
      // 用户确认，打开进度对话框并开始绑定
      batchBindDialogVisible.value = true
      batchBindOutput.value = ""
      handleBatchBind()
    })
    .catch(() => {
      // 用户取消
      ElMessage.info("已取消批量绑定")
    })
}

// 执行批量智能绑定
function handleBatchBind() {
  const hasAccount = accountType.value === "merchant" ? selectedMerchant.value : selectedCloudAccount.value
  if (!hasAccount || selectedEips.value.length === 0) {
    ElMessage.warning("请先选择需要绑定的弹性IP")
    return
  }

  batchBindLoading.value = true
  batchBindOutput.value = "开始批量智能绑定弹性IP...\n"

  // 构建请求参数
  const eip_list: EipItem[] = selectedEips.value.map(eip => ({
    allocation_id: eip.AllocationId,
    region_id: eip.RegionId
  }))

  const params: any = {
    eip_list
  }

  if (accountType.value === "merchant") {
    params.merchant_id = selectedMerchant.value
  } else {
    params.cloud_account_id = selectedCloudAccount.value
  }

  batchAssociateEip(
    params,
    (output, isComplete) => {
      batchBindOutput.value += `${output}\n`

      if (isComplete) {
        batchBindLoading.value = false
        ElMessage.success("批量智能绑定操作完成")
        // 刷新弹性IP列表
        setTimeout(() => {
          fetchEipList()
        }, 1000)
      }
    },
    (error) => {
      console.error("批量智能绑定失败", error)
      batchBindOutput.value += `批量智能绑定失败: ${error.message || JSON.stringify(error)}\n`
      batchBindLoading.value = false
      ElMessage.error("批量智能绑定失败")
    }
  )
}

// 打开更换弹性IP对话框
function openReplaceDialog(eip: Eip) {
  const hasAccount = accountType.value === "merchant" ? selectedMerchant.value : selectedCloudAccount.value
  if (!hasAccount) {
    ElMessage.warning("账号信息缺失")
    return
  }

  // 检查EIP是否绑定了实例或网卡
  if (!eip.InstanceId || (eip.InstanceType !== "NetworkInterface" && eip.InstanceType !== "EcsInstance")) {
    ElMessage.warning("更换IP功能仅支持绑定了ECS实例或辅助网卡的弹性IP")
    return
  }

  ElMessageBox.confirm(
    `确定要更换弹性IP ${eip.IpAddress} 吗？\n\n操作流程：\n1. 解绑当前IP与实例/网卡\n2. 从共享带宽中移除（如有）\n3. 释放旧IP\n4. 创建新IP\n5. 加入共享带宽（如有）\n6. 绑定新IP到原实例/网卡`,
    "更换弹性IP",
    {
      confirmButtonText: "确定",
      cancelButtonText: "取消",
      type: "warning"
    }
  )
    .then(() => {
      currentReplaceEip.value = eip
      replaceDialogVisible.value = true
      replaceOutput.value = ""
      handleReplaceEip(eip)
    })
    .catch(() => {
      // 用户取消
    })
}

// 执行更换弹性IP
function handleReplaceEip(eip: Eip) {
  const hasAccount = accountType.value === "merchant" ? selectedMerchant.value : selectedCloudAccount.value
  if (!hasAccount || !eip) {
    ElMessage.warning("请先选择账号")
    return
  }

  replaceLoading.value = true
  replaceOutput.value = "开始更换弹性IP...\n"

  const params: any = {
    region_id: eip.RegionId,
    allocation_id: eip.AllocationId,
    old_ip_address: eip.IpAddress || "",
    instance_id: eip.InstanceId,
    instance_type: eip.InstanceType,
    bandwidth_package_id: eip.BandwidthPackageId || "",
    bandwidth: eip.Bandwidth || "",
    internet_charge_type: eip.InternetChargeType || "",
    name: eip.Name || ""
  }

  if (accountType.value === "merchant") {
    params.merchant_id = selectedMerchant.value
  } else {
    params.cloud_account_id = selectedCloudAccount.value
  }

  replaceEip(
    params,
    (output, isComplete) => {
      replaceOutput.value += `${output}\n`

      if (isComplete) {
        replaceLoading.value = false
        ElMessage.success("更换弹性IP操作完成")
        // 刷新弹性IP列表
        setTimeout(() => {
          fetchEipList()
        }, 1000)
      }
    },
    (error) => {
      console.error("更换弹性IP失败", error)
      replaceOutput.value += `更换弹性IP失败: ${error.message || JSON.stringify(error)}\n`
      replaceLoading.value = false
      ElMessage.error("更换弹性IP失败")
    }
  )
}

// 打开批量更换弹性IP对话框
function openBatchReplaceDialog() {
  if (selectedEips.value.length === 0) {
    ElMessage.warning("请先选择需要更换的弹性IP")
    return
  }

  // 检查是否全部都绑定了实例或辅助网卡
  const invalidEips = selectedEips.value.filter(
    eip => (eip.InstanceType !== "NetworkInterface" && eip.InstanceType !== "EcsInstance") || !eip.InstanceId
  )
  if (invalidEips.length > 0) {
    ElMessage.warning("只能对绑定了ECS实例或辅助网卡的弹性IP进行更换，请重新选择")
    return
  }

  // 弹出确认对话框
  ElMessageBox.confirm(
    `确定要更换选中的 ${selectedEips.value.length} 个弹性IP吗？\n\n操作将依次对每个EIP执行：\n1. 创建新EIP\n2. 解绑旧EIP\n3. 移除旧EIP共享带宽（如有）\n4. 加入新EIP共享带宽（如有）\n5. 绑定新EIP\n6. 释放旧EIP`,
    "批量更换弹性IP",
    {
      confirmButtonText: "确定",
      cancelButtonText: "取消",
      type: "warning"
    }
  )
    .then(() => {
      // 用户确认，打开进度对话框并开始更换
      batchReplaceDialogVisible.value = true
      batchReplaceOutput.value = ""
      handleBatchReplace()
    })
    .catch(() => {
      // 用户取消
      ElMessage.info("已取消批量更换")
    })
}

// 执行批量更换弹性IP
function handleBatchReplace() {
  const hasAccount = accountType.value === "merchant" ? selectedMerchant.value : selectedCloudAccount.value
  if (!hasAccount || selectedEips.value.length === 0) {
    ElMessage.warning("请先选择需要更换的弹性IP")
    return
  }

  batchReplaceLoading.value = true
  batchReplaceOutput.value = "开始批量更换弹性IP...\n"

  // 构建请求参数
  const eip_list: BatchReplaceEipConfig[] = selectedEips.value.map(eip => ({
    region_id: eip.RegionId,
    allocation_id: eip.AllocationId,
    old_ip_address: eip.IpAddress || "",
    instance_id: eip.InstanceId,
    instance_type: eip.InstanceType,
    bandwidth_package_id: eip.BandwidthPackageId || "",
    bandwidth: eip.Bandwidth || "",
    internet_charge_type: eip.InternetChargeType || ""
  }))

  const params: any = {
    eip_list
  }

  if (accountType.value === "merchant") {
    params.merchant_id = selectedMerchant.value
  } else {
    params.cloud_account_id = selectedCloudAccount.value
  }

  batchReplaceEip(
    params,
    (output, isComplete) => {
      batchReplaceOutput.value += `${output}\n`

      if (isComplete) {
        batchReplaceLoading.value = false
        ElMessage.success("批量更换弹性IP操作完成")
        // 刷新弹性IP列表
        setTimeout(() => {
          fetchEipList()
        }, 1000)
      }
    },
    (error) => {
      console.error("批量更换弹性IP失败", error)
      batchReplaceOutput.value += `批量更换弹性IP失败: ${error.message || JSON.stringify(error)}\n`
      batchReplaceLoading.value = false
      ElMessage.error("批量更换弹性IP失败")
    }
  )
}

// 页面加载时获取商户和区域列表
onMounted(() => {
  loadSelectionFromStorage()

  if (accountType.value === "system") {
    fetchCloudAccountList().then(() => {
      if (selectedCloudAccount.value) {
        fetchRegionList()
      }
    })
  } else {
    fetchMerchantList().then(() => {
      if (selectedMerchant.value) {
        fetchRegionList()
      }
    })
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
          @click="fetchEipList"
        >
          查询
        </el-button>
        <el-button
          type="success"
          :disabled="!selectedMerchant && !selectedCloudAccount"
          @click="openCreateDialog"
        >
          创建弹性IP
        </el-button>
      </div>
    </el-card>

    <el-card v-if="(accountType === 'merchant' ? selectedMerchant : selectedCloudAccount) && selectedRegions.length > 0" class="table-card">
      <template #header>
        <div class="card-header">
          <span>弹性IP列表</span>
          <div class="operations">
            <el-button type="primary" size="small" @click="fetchEipList">
              刷新
            </el-button>
            <el-button
              type="success"
              size="small"
              :disabled="selectedEips.length === 0"
              @click="openBatchBindDialog"
            >
              批量智能绑定
            </el-button>
            <el-button
              type="warning"
              size="small"
              :disabled="selectedEips.length === 0"
              @click="openBatchReplaceDialog"
            >
              批量更换IP
            </el-button>
          </div>
        </div>
      </template>
      <el-table
        v-loading="loading"
        :data="eipList"
        style="width: 100%"
        border
        @selection-change="handleSelectionChange"
      >
        <el-table-column type="selection" width="55" />
        <el-table-column prop="AllocationId" label="弹性IP ID" min-width="180" />
        <el-table-column prop="IpAddress" label="公网IP" min-width="150" />
        <el-table-column label="区域" min-width="120">
          <template #default="scope">
            {{ scope.row.RegionName || scope.row.RegionId || "-" }}
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
        <el-table-column label="绑定实例" min-width="180">
          <template #default="scope">
            <div class="flex-row-center">
              <span v-if="scope.row.InstanceId">
                {{ getInstanceTypeText(scope.row.InstanceType) }}: {{ scope.row.InstanceId }}
                <el-button
                  v-if="scope.row.InstanceType === 'EcsInstance'"
                  type="success"
                  text
                  size="small"
                  @click.stop="viewEcsInstanceDetail(scope.row)"
                >
                  <el-icon><Monitor /></el-icon>
                  查看
                </el-button>
                <el-button
                  v-if="scope.row.InstanceType === 'NetworkInterface'"
                  type="success"
                  text
                  size="small"
                  @click.stop="viewNetworkInterfaceDetail(scope.row)"
                >
                  <el-icon><Monitor /></el-icon>
                  查看
                </el-button>
              </span>
              <span v-else>-</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="ChargeType" label="计费方式" min-width="120">
          <template #default="scope">
            {{ getInternetChargeType(scope.row.InternetChargeType) }}
          </template>
        </el-table-column>
        <el-table-column prop="AllocationTime" label="创建时间" min-width="180" />
        <el-table-column label="操作" fixed="right" min-width="120">
          <template #default="scope">
            <div class="action-buttons">
              <el-button
                type="primary"
                text
                size="small"
                @click="showEipDetail(scope.row)"
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
                    <el-dropdown-item v-if="scope.row.Status === 'Available'" @click="openBindDialog(scope.row)">
                      <el-icon><Link /></el-icon> 绑定
                    </el-dropdown-item>
                    <el-dropdown-item v-if="scope.row.Status === 'InUse'" @click="confirmUnassociate(scope.row)">
                      <el-icon><Link /></el-icon> 解绑
                    </el-dropdown-item>
                    <el-dropdown-item v-if="!scope.row.BandwidthPackageId" @click="openJoinBandwidthDialog(scope.row)">
                      <el-icon><Link /></el-icon> 加入共享带宽
                    </el-dropdown-item>
                    <el-dropdown-item v-if="scope.row.BandwidthPackageId" @click="confirmLeaveBandwidth(scope.row)">
                      <el-icon><Link /></el-icon> 离开共享带宽
                    </el-dropdown-item>
                    <el-dropdown-item v-if="(scope.row.InstanceType === 'NetworkInterface' || scope.row.InstanceType === 'EcsInstance') && scope.row.InstanceId" @click="openReplaceDialog(scope.row)">
                      <el-icon><Edit /></el-icon> 更换IP
                    </el-dropdown-item>
                    <el-dropdown-item @click="openModifyDialog(scope.row)">
                      <el-icon><Edit /></el-icon> 修改
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

    <!-- 弹性IP详情对话框 -->
    <el-dialog
      v-model="detailDialogVisible"
      :title="`弹性IP详情 - ${detailEip?.IpAddress || ''}`"
      width="750px"
      destroy-on-close
    >
      <div v-if="detailEip" class="eip-detail">
        <el-descriptions :column="2" border>
          <el-descriptions-item label="弹性IP ID" :span="2">
            {{ detailEip.AllocationId }}
          </el-descriptions-item>
          <el-descriptions-item label="名称">
            {{ detailEip.Name || '-' }}
          </el-descriptions-item>
          <el-descriptions-item label="状态">
            <el-tag :type="getStatusType(detailEip.Status)">
              {{ getStatusText(detailEip.Status) }}
            </el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="IP地址" :span="2">
            {{ detailEip.IpAddress }}
          </el-descriptions-item>
          <el-descriptions-item label="带宽">
            {{ detailEip.Bandwidth ? `${detailEip.Bandwidth} Mbps` : '-' }}
          </el-descriptions-item>
          <el-descriptions-item label="计费方式">
            {{ getInternetChargeType(detailEip.InternetChargeType) }}
          </el-descriptions-item>
          <el-descriptions-item label="共享带宽包ID" v-if="detailEip.BandwidthPackageId">
            {{ detailEip.BandwidthPackageId }}
          </el-descriptions-item>
          <el-descriptions-item label="共享带宽包带宽" v-if="detailEip.BandwidthPackageBandwidth">
            {{ detailEip.BandwidthPackageBandwidth }} Mbps
          </el-descriptions-item>
          <el-descriptions-item label="创建时间">
            {{ detailEip.AllocationTime }}
          </el-descriptions-item>
          <el-descriptions-item label="到期时间">
            {{ detailEip.ExpiredTime || '-' }}
          </el-descriptions-item>
          <el-descriptions-item label="业务状态">
            {{ detailEip.BusinessStatus === 'Normal' ? '正常' : detailEip.BusinessStatus }}
          </el-descriptions-item>
          <el-descriptions-item label="计费类型">
            {{ detailEip.ChargeType === 'PostPaid' ? '按量计费' : '包年包月' }}
          </el-descriptions-item>
        </el-descriptions>

        <!-- 绑定信息 -->
        <div v-if="detailEip.InstanceId" class="detail-section">
          <h3 class="detail-title">
            绑定信息
          </h3>
          <el-descriptions :column="2" border>
            <el-descriptions-item label="实例类型">
              {{ getInstanceTypeText(detailEip.InstanceType) }}
            </el-descriptions-item>
            <el-descriptions-item label="实例ID">
              {{ detailEip.InstanceId }}
            </el-descriptions-item>
            <el-descriptions-item label="绑定模式">
              {{ {
                NAT: 'NAT模式',
                MULTI_BINDED: '多EIP网卡可见模式',
                BINDED: 'EIP网卡可见模式',
              }[detailEip.Mode] || detailEip.Mode }}
            </el-descriptions-item>
            <el-descriptions-item label="私网IP" v-if="detailEip.PrivateIpAddress">
              {{ detailEip.PrivateIpAddress }}
            </el-descriptions-item>
          </el-descriptions>
        </div>

        <!-- 续费信息 -->
        <div v-if="detailEip.ReservationOrderType" class="detail-section">
          <h3 class="detail-title">
            续费信息
          </h3>
          <el-descriptions :column="2" border>
            <el-descriptions-item label="续费订单类型">
              {{ {
                RENEWCHANGE: '续费变配',
                TEMP_UPGRADE: '短时升配',
                UPGRADE: '升级',
              }[detailEip.ReservationOrderType] || detailEip.ReservationOrderType }}
            </el-descriptions-item>
            <el-descriptions-item label="续费带宽">
              {{ detailEip.ReservationBandwidth ? `${detailEip.ReservationBandwidth} Mbps` : '-' }}
            </el-descriptions-item>
            <el-descriptions-item label="续费生效时间">
              {{ detailEip.ReservationActiveTime || '-' }}
            </el-descriptions-item>
            <el-descriptions-item label="续费计费方式">
              {{ getInternetChargeType(detailEip.ReservationInternetChargeType || '') }}
            </el-descriptions-item>
          </el-descriptions>
        </div>

        <!-- 其他信息 -->
        <div class="detail-section">
          <h3 class="detail-title">
            其他信息
          </h3>
          <el-descriptions :column="2" border>
            <el-descriptions-item label="描述" :span="2">
              {{ detailEip.Description || '-' }}
            </el-descriptions-item>
            <el-descriptions-item label="VPC ID" v-if="detailEip.VpcId">
              {{ detailEip.VpcId }}
            </el-descriptions-item>
          </el-descriptions>
        </div>

        <!-- 共享带宽包信息 -->
        <div class="detail-section">
          <h3 class="detail-title">
            共享带宽包信息
          </h3>
          <div v-if="detailEip.BandwidthPackageId">
            <el-descriptions :column="2" border>
              <el-descriptions-item label="共享带宽包ID">
                {{ detailEip.BandwidthPackageId }}
              </el-descriptions-item>
              <el-descriptions-item label="共享带宽">
                {{ detailEip.BandwidthPackageBandwidth ? `${detailEip.BandwidthPackageBandwidth} Mbps` : '-' }}
              </el-descriptions-item>
            </el-descriptions>
          </div>
          <el-empty v-else description="未加入共享带宽包" :image-size="60" />
        </div>
      </div>
      <el-empty v-else description="未找到弹性IP详情" />
    </el-dialog>

    <!-- 实例详情对话框 -->
    <el-dialog
      v-model="instanceDialogVisible"
      title="实例详情"
      width="750px"
      destroy-on-close
    >
      <div v-loading="instanceLoading" class="instance-detail">
        <div v-if="instanceDetail" class="instance-detail">
          <el-descriptions :column="2" border>
            <el-descriptions-item label="实例ID">
              {{ instanceDetail.InstanceId }}
            </el-descriptions-item>
            <el-descriptions-item label="实例名称">
              {{ instanceDetail.InstanceName }}
            </el-descriptions-item>
            <el-descriptions-item label="CPU">
              {{ instanceDetail.Cpu }} 核
            </el-descriptions-item>
            <el-descriptions-item label="内存">
              {{ instanceDetail.Memory ? Math.floor(Number(instanceDetail.Memory) / 1024) : 0 }} GB
            </el-descriptions-item>
            <el-descriptions-item label="实例状态">
              <el-tag :type="getStatusType(instanceDetail.Status)">
                {{ getStatusText(instanceDetail.Status) }}
              </el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="操作系统">
              {{ instanceDetail.OSName }}
            </el-descriptions-item>
            <el-descriptions-item label="实例类型">
              {{ instanceDetail.InstanceType }}
            </el-descriptions-item>
            <el-descriptions-item label="网络类型">
              {{ instanceDetail.InstanceNetworkType }}
            </el-descriptions-item>
            <el-descriptions-item label="公网IP" v-if="instanceDetail.EipAddress?.IpAddress">
              {{ instanceDetail.EipAddress.IpAddress }}
            </el-descriptions-item>
            <el-descriptions-item label="创建时间">
              {{ instanceDetail.CreationTime }}
            </el-descriptions-item>
            <el-descriptions-item label="到期时间">
              {{ instanceDetail.ExpiredTime || '-' }}
            </el-descriptions-item>
          </el-descriptions>
        </div>
        <el-empty v-else description="未找到实例详情" />
      </div>
    </el-dialog>

    <!-- 修改弹性IP对话框 -->
    <el-dialog
      v-model="modifyDialogVisible"
      title="修改弹性IP"
      width="600px"
      destroy-on-close
    >
      <el-form
        ref="modifyFormRef"
        :model="modifyEipForm"
        :rules="modifyFormRules"
        label-width="120px"
        label-position="right"
      >
        <el-form-item label="名称">
          <el-input
            v-model="modifyEipForm.name"
            placeholder="请输入名称"
            style="width: 200px"
          />
        </el-form-item>

        <el-form-item label="描述">
          <el-input
            v-model="modifyEipForm.description"
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

    <!-- 绑定对话框 -->
    <el-dialog
      v-model="bindDialogVisible"
      title="绑定弹性IP"
      width="700px"
      destroy-on-close
    >
      <div class="bind-form">
        <el-form label-width="120px" label-position="right" style="width: 100%;">
          <el-form-item label="实例类型">
            <el-radio-group v-model="bindInstanceType" @change="handleInstanceTypeChange">
              <el-radio label="EcsInstance">
                ECS实例
              </el-radio>
              <el-radio label="NetworkInterface">
                辅助弹性网卡
              </el-radio>
            </el-radio-group>
          </el-form-item>

          <el-form-item v-if="bindInstanceType === 'EcsInstance'" label="实例ID">
            <div v-loading="loadingInstancesForBind">
              <el-select
                v-model="bindInstanceId"
                placeholder="请选择实例"
                style="width: 100%; min-width: 400px;"
                :disabled="loadingInstancesForBind || !instanceListForBind.length"
              >
                <el-option
                  v-for="item in instanceListForBind"
                  :key="item.InstanceId"
                  :label="`${item.InstanceName || ''} (${item.InstanceId})`"
                  :value="item.InstanceId"
                />
              </el-select>
              <div class="form-tip" v-if="instanceListForBind.length === 0 && !loadingInstancesForBind">
                当前区域没有可用的ECS实例
              </div>
            </div>
          </el-form-item>

          <el-form-item v-else label="弹性网卡ID">
            <div v-loading="loadingNetworksForBind">
              <el-select
                v-model="bindInstanceId"
                placeholder="请选择弹性网卡"
                style="width: 100%; min-width: 400px;"
                :disabled="loadingNetworksForBind || !networkListForBind.length"
              >
                <el-option
                  v-for="item in networkListForBind"
                  :key="item.NetworkInterface.NetworkInterfaceId"
                  :label="item.NetworkInterface.NetworkInterfaceId"
                  :value="item.NetworkInterface.NetworkInterfaceId"
                />
              </el-select>
              <div class="form-tip" v-if="networkListForBind.length === 0 && !loadingNetworksForBind">
                当前区域没有可用的弹性网卡
              </div>
            </div>
          </el-form-item>
        </el-form>
      </div>

      <template #footer>
        <div class="dialog-footer">
          <el-button @click="bindDialogVisible = false">
            取消
          </el-button>
          <el-button
            type="primary"
            :loading="bindLoading"
            @click="submitBindForm"
          >
            绑定
          </el-button>
        </div>
      </template>
    </el-dialog>

    <!-- 加入共享带宽对话框 -->
    <el-dialog
      v-model="joinBandwidthDialogVisible"
      title="加入共享带宽"
      width="600px"
      destroy-on-close
    >
      <div class="bind-form">
        <el-form label-width="120px" label-position="right">
          <el-form-item label="共享带宽ID">
            <div v-loading="loadingBandwidthForJoin">
              <el-select
                v-model="selectedBandwidthId"
                placeholder="请选择共享带宽"
                style="width: 100%; min-width: 400px;"
                :disabled="loadingBandwidthForJoin || !bandwidthListForJoin.length"
              >
                <el-option
                  v-for="item in bandwidthListForJoin"
                  :key="item.BandwidthPackageId"
                  :label="`${item.Name || item.BandwidthPackageId} (${item.Bandwidth}Mbps)`"
                  :value="item.BandwidthPackageId"
                >
                  <div class="bandwidth-option">
                    <div>
                      <div>{{ item.Name || item.BandwidthPackageId }}</div>
                      <div class="bandwidth-option-meta">
                        <span>带宽: {{ item.Bandwidth }}Mbps</span>
                        <span>状态: {{ getStatusText(item.Status) }}</span>
                      </div>
                    </div>
                    <el-tag size="small" :type="getStatusType(item.Status)">
                      {{ getStatusText(item.Status) }}
                    </el-tag>
                  </div>
                </el-option>
              </el-select>
              <div class="form-tip" v-if="bandwidthListForJoin.length === 0 && !loadingBandwidthForJoin">
                当前区域没有可用的共享带宽
              </div>
            </div>
          </el-form-item>
        </el-form>
      </div>

      <template #footer>
        <div class="dialog-footer">
          <el-button @click="joinBandwidthDialogVisible = false">
            取消
          </el-button>
          <el-button
            type="primary"
            :loading="joinBandwidthLoading"
            @click="submitJoinBandwidthForm"
          >
            加入
          </el-button>
        </div>
      </template>
    </el-dialog>

    <!-- 批量智能绑定对话框 -->
    <el-dialog
      v-model="batchBindDialogVisible"
      title="批量智能绑定"
      width="600px"
      destroy-on-close
      :close-on-click-modal="false"
      :close-on-press-escape="false"
    >
      <div v-loading="batchBindLoading" class="output-container">
        <pre>{{ batchBindOutput }}</pre>
      </div>
      <template #footer>
        <div class="dialog-footer">
          <el-button :disabled="batchBindLoading" @click="batchBindDialogVisible = false">
            关闭
          </el-button>
        </div>
      </template>
    </el-dialog>

    <!-- 更换弹性IP对话框 -->
    <el-dialog
      v-model="replaceDialogVisible"
      :title="`更换弹性IP - ${currentReplaceEip?.IpAddress || ''}`"
      width="650px"
      destroy-on-close
      :close-on-click-modal="false"
      :close-on-press-escape="false"
    >
      <div v-loading="replaceLoading" class="output-container">
        <pre>{{ replaceOutput }}</pre>
      </div>
      <template #footer>
        <div class="dialog-footer">
          <el-button :disabled="replaceLoading" @click="replaceDialogVisible = false">
            关闭
          </el-button>
        </div>
      </template>
    </el-dialog>

    <!-- 批量更换弹性IP对话框 -->
    <el-dialog
      v-model="batchReplaceDialogVisible"
      title="批量更换弹性IP"
      width="700px"
      destroy-on-close
      :close-on-click-modal="false"
      :close-on-press-escape="false"
    >
      <div v-loading="batchReplaceLoading" class="output-container">
        <pre>{{ batchReplaceOutput }}</pre>
      </div>
      <template #footer>
        <div class="dialog-footer">
          <el-button :disabled="batchReplaceLoading" @click="batchReplaceDialogVisible = false">
            关闭
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

.eip-detail {
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

.unit {
  margin-left: 8px;
  color: #606266;
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

.instance-detail {
  padding: 10px;
}

.el-dropdown-link {
  cursor: pointer;
  color: #409eff;
  display: flex;
  align-items: center;
}

.flex-row-center {
  display: flex;
  align-items: center;
  gap: 8px;
}

.bandwidth-option {
  display: flex;
  justify-content: space-between;
  align-items: center;
  width: 100%;
}

.bandwidth-option-meta {
  font-size: 12px;
  color: #909399;
  margin-top: 2px;
  display: flex;
  gap: 10px;
}

.output-container {
  margin-top: 10px;
  padding: 10px;
  background-color: #1e1e1e;
  border-radius: 4px;
  max-height: 300px;
  overflow-y: auto;
}

.output-container pre {
  margin: 0;
  white-space: pre-wrap;
  word-wrap: break-word;
  font-family: monospace;
  font-size: 14px;
  line-height: 1.5;
  color: #e6e6e6;
}
</style>
