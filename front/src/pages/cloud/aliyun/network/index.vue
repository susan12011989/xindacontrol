<script setup lang="ts">
import type { Region } from "@/pages/cloud/aliyun/apis/type"
import type { Instance } from "@/pages/cloud/aliyun/instances/apis/type"
import type { CreateNetworkInterfaceRequestData, NetworkInterfaceWrap } from "@/pages/cloud/aliyun/network/apis/type"
import type { SecurityGroupWrap } from "@/pages/cloud/aliyun/securitygroup/apis/type"
import type { Merchant, MerchantRegions } from "@/pages/dashboard/apis/type"
import type { CloudAccountOption } from "@@/apis/cloud_account/type"
import { regionListApi } from "@/pages/cloud/aliyun/apis"
import { getInstanceList } from "@/pages/cloud/aliyun/instances/apis"
import { attachNetworkInterface, createNetworkInterface, deleteNetworkInterface, detachNetworkInterface, getNetworkInterfaceList, modifyNetworkInterfaceAttribute } from "@/pages/cloud/aliyun/network/apis"
import { getSecurityGroupList } from "@/pages/cloud/aliyun/securitygroup/apis"
import { merchantQueryApi } from "@/pages/dashboard/apis"
import { getCloudAccountOptions } from "@@/apis/cloud_account"
import { ArrowDown, CopyDocument, Delete, Edit, Key, Link, Location, Monitor, User } from "@element-plus/icons-vue"
import { ElMessage, ElMessageBox } from "element-plus"
import { onMounted, reactive, ref, watch } from "vue"
import { useRoute } from "vue-router"

defineOptions({
  name: "CloudNetwork"
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
const networkList = ref<NetworkInterfaceWrap[]>([])
const selectedMerchant = ref<number>()
const selectedRegions = ref<string[]>([])

// 创建弹性网卡对话框
const createDialogVisible = ref(false)
const createLoading = ref(false)
const createFormRef = ref()
const createNetworkForm = reactive<CreateNetworkInterfaceRequestData>({
  merchant_id: 0,
  cloud_account_id: 0,
  region_id: "", // 用于API请求
  vswitch_id: "",
  security_group_id: "",
  network_interface_name: "" // 弹性网卡名称
})

// 创建表单相关数据
const instanceListForCreate = ref<Instance[]>([])
const instancesLoading = ref(false)
const securityGroupListForCreate = ref<SecurityGroupWrap[]>([])
const securityGroupsLoading = ref(false)
const selectedInstanceId = ref<string>("")

// 创建表单规则
const createFormRules = {
  vswitch_id: [
    { required: true, message: "请输入交换机ID", trigger: "blur" }
  ],
  security_group_id: [
    { required: true, message: "请输入安全组ID", trigger: "blur" }
  ],
  network_interface_name: [
    { max: 128, message: "名称最大长度为128个字符", trigger: "blur" }
  ]
}

// 获取路由实例和查询参数
const route = useRoute()

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
  networkList.value = []

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
  networkList.value = []

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

// 获取弹性网卡列表
async function fetchNetworkList() {
  const hasAccount = accountType.value === "merchant" ? selectedMerchant.value : selectedCloudAccount.value
  if (!hasAccount || !selectedRegions.value || selectedRegions.value.length === 0) {
    console.log("账号或区域未选择，不获取弹性网卡列表")
    return
  }

  console.log("开始获取弹性网卡列表, 类型:", accountType.value, "账号:", hasAccount, "区域:", selectedRegions.value)
  loading.value = true
  try {
    const params: any = { region_id: selectedRegions.value }
    if (accountType.value === "merchant") {
      params.merchant_id = selectedMerchant.value
    } else {
      params.cloud_account_id = selectedCloudAccount.value
    }

    const res = await getNetworkInterfaceList(params)

    console.log("获取弹性网卡列表成功:", res.data.list)
    // 兼容返回空数组的情况
    if (!res.data.list || !Array.isArray(res.data.list)) {
      console.log("弹性网卡列表为空或格式不正确，设置为空数组")
      networkList.value = []
    } else {
      networkList.value = res.data.list
    }

    // 如果列表为空，显示提示
    if (networkList.value.length === 0) {
      ElMessage.info("当前区域暂无弹性网卡")
    }
  } catch (error) {
    console.error("获取弹性网卡列表失败", error)
    ElMessage.error("获取弹性网卡列表失败")
    networkList.value = []
  } finally {
    loading.value = false
  }
}

// 商户变化处理
function handleMerchantChange(value: number | undefined) {
  console.log("商户变更为:", value)
  selectedRegions.value = []
  networkList.value = []

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
  networkList.value = []
}

// 获取状态文本和类型处理
function getStatusText(status: string) {
  const statusMap: Record<string, string> = {
    Available: "可用",
    Attaching: "附加中",
    InUse: "已附加",
    Detaching: "分离中",
    Deleting: "删除中"
  }
  return statusMap[status] || status
}

function getStatusType(status: string) {
  const typeMap: Record<string, "success" | "warning" | "info" | "primary" | "danger"> = {
    Available: "info",
    Attaching: "warning",
    InUse: "success",
    Detaching: "warning",
    Deleting: "danger"
  }
  return typeMap[status] || ""
}

// 确认删除网卡
function confirmDelete(network: NetworkInterfaceWrap) {
  ElMessageBox.confirm(
    `确定要删除弹性网卡 ${network.NetworkInterface.NetworkInterfaceId} 吗？`,
    "删除确认",
    {
      confirmButtonText: "确定",
      cancelButtonText: "取消",
      type: "warning"
    }
  )
    .then(() => {
      handleDelete(network)
    })
    .catch(() => {
      // 用户取消删除操作
    })
}

// 确认解绑实例
function confirmDetach(network: NetworkInterfaceWrap) {
  ElMessageBox.confirm(
    `确定要解绑弹性网卡 ${network.NetworkInterface.NetworkInterfaceId} 吗？`,
    "解绑确认",
    {
      confirmButtonText: "确定",
      cancelButtonText: "取消",
      type: "warning"
    }
  )
    .then(() => {
      handleDetach(network)
    })
    .catch(() => {
      // 用户取消解绑操作
    })
}

// 删除网卡
async function handleDelete(network: NetworkInterfaceWrap) {
  const hasAccount = accountType.value === "merchant" ? selectedMerchant.value : selectedCloudAccount.value
  if (!hasAccount) return

  try {
    const params: any = {
      region_id: network.RegionId,
      network_interface_id: network.NetworkInterface.NetworkInterfaceId
    }

    if (accountType.value === "merchant") {
      params.merchant_id = selectedMerchant.value
    } else {
      params.cloud_account_id = selectedCloudAccount.value
    }

    await deleteNetworkInterface(params)

    ElMessage.success("删除成功")
    // 延迟刷新列表，等待操作完成
    setTimeout(() => {
      fetchNetworkList()
    }, 2000)
  } catch (error) {
    console.error("删除弹性网卡失败", error)
    ElMessage.error("删除弹性网卡失败")
  }
}

// 解绑网卡
async function handleDetach(network: NetworkInterfaceWrap) {
  const hasAccount = accountType.value === "merchant" ? selectedMerchant.value : selectedCloudAccount.value
  if (!hasAccount || !network.NetworkInterface.InstanceId) return

  try {
    const params: any = {
      region_id: network.RegionId,
      network_interface_id: network.NetworkInterface.NetworkInterfaceId,
      instance_id: network.NetworkInterface.InstanceId
    }

    if (accountType.value === "merchant") {
      params.merchant_id = selectedMerchant.value
    } else {
      params.cloud_account_id = selectedCloudAccount.value
    }

    await detachNetworkInterface(params)

    ElMessage.success("解绑成功")
    // 延迟刷新列表，等待操作完成
    setTimeout(() => {
      fetchNetworkList()
    }, 2000)
  } catch (error) {
    console.error("解绑弹性网卡失败", error)
    ElMessage.error("解绑弹性网卡失败")
  }
}

// 弹性网卡详情
const detailDialogVisible = ref(false)
const detailNetwork = ref<NetworkInterfaceWrap | null>(null)

// 显示弹性网卡详情
function showNetworkDetail(network: NetworkInterfaceWrap) {
  detailNetwork.value = network
  detailDialogVisible.value = true
}

// 打开创建弹性网卡对话框
function openCreateDialog() {
  const hasAccount = accountType.value === "merchant" ? selectedMerchant.value : selectedCloudAccount.value
  if (!hasAccount) {
    ElMessage.warning("请先选择账号")
    return
  }

  if (accountType.value === "merchant") {
    createNetworkForm.merchant_id = selectedMerchant.value!
  } else {
    createNetworkForm.cloud_account_id = selectedCloudAccount.value!
  }
  createNetworkForm.region_id = ""
  createNetworkForm.vswitch_id = ""
  createNetworkForm.security_group_id = ""
  createNetworkForm.network_interface_name = ""
  selectedInstanceId.value = ""
  createDialogVisible.value = true
}

// 当创建弹窗中的region_id变化时，获取实例列表和安全组列表
watch(() => createNetworkForm.region_id, (newValue) => {
  // 重置选中的实例ID
  selectedInstanceId.value = ""

  const hasAccount = accountType.value === "merchant" ? selectedMerchant.value : selectedCloudAccount.value
  if (newValue && hasAccount) {
    // 获取实例列表
    fetchInstancesForCreate(newValue)
    // 获取安全组列表
    fetchSecurityGroupsForCreate(newValue)
  } else {
    instanceListForCreate.value = []
    securityGroupListForCreate.value = []
  }
})

// 获取创建对话框中的实例列表
async function fetchInstancesForCreate(regionId: string) {
  const hasAccount = accountType.value === "merchant" ? selectedMerchant.value : selectedCloudAccount.value
  if (!hasAccount) return

  instancesLoading.value = true
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

    if (res.data && res.data.list) {
      instanceListForCreate.value = res.data.list
    } else {
      instanceListForCreate.value = []
    }
  } catch (error) {
    console.error("获取实例列表失败", error)
    ElMessage.error("获取实例列表失败")
    instanceListForCreate.value = []
  } finally {
    instancesLoading.value = false
  }
}

// 获取创建对话框中的安全组列表
async function fetchSecurityGroupsForCreate(regionId: string) {
  const hasAccount = accountType.value === "merchant" ? selectedMerchant.value : selectedCloudAccount.value
  if (!hasAccount) return

  securityGroupsLoading.value = true
  try {
    const params: any = {
      region_id: [regionId]
    }

    if (accountType.value === "merchant") {
      params.merchant_id = selectedMerchant.value
    } else {
      params.cloud_account_id = selectedCloudAccount.value
    }

    const res = await getSecurityGroupList(params)

    if (res.data && res.data.list) {
      securityGroupListForCreate.value = res.data.list
    } else {
      securityGroupListForCreate.value = []
    }
  } catch (error) {
    console.error("获取安全组列表失败", error)
    ElMessage.error("获取安全组列表失败")
    securityGroupListForCreate.value = []
  } finally {
    securityGroupsLoading.value = false
  }
}

// 从实例中选择VSwitchId
function selectVSwitchFromInstance(instanceId: string) {
  selectedInstanceId.value = instanceId
  const selectedInstance = instanceListForCreate.value.find(instance => instance.InstanceId === instanceId)
  if (selectedInstance && selectedInstance.VpcAttributes && selectedInstance.VpcAttributes.VSwitchId) {
    createNetworkForm.vswitch_id = selectedInstance.VpcAttributes.VSwitchId
  }
}

// 提交创建弹性网卡表单
async function submitCreateForm(formEl: any) {
  if (!formEl) return

  await formEl.validate(async (valid: boolean) => {
    if (valid) {
      // 更新region_id为选中的区域
      if (!createNetworkForm.region_id) {
        ElMessage.warning("请选择区域")
        return
      }

      createLoading.value = true
      try {
        await createNetworkInterface(createNetworkForm)
        ElMessage.success("创建弹性网卡成功")
        createDialogVisible.value = false

        // 刷新列表
        setTimeout(() => {
          fetchNetworkList()
        }, 1000)
      } catch (error) {
        console.error("创建弹性网卡失败", error)
        ElMessage.error("创建弹性网卡失败")
      } finally {
        createLoading.value = false
      }
    }
  })
}

// 重置创建表单
function resetCreateForm(formEl: any) {
  if (!formEl) return
  formEl.resetFields()
  createNetworkForm.region_id = ""
  createNetworkForm.vswitch_id = ""
  createNetworkForm.security_group_id = ""
  createNetworkForm.network_interface_name = ""
  selectedInstanceId.value = ""
  instanceListForCreate.value = []
  securityGroupListForCreate.value = []
}

// 修改弹性网卡对话框
const modifyDialogVisible = ref(false)
const modifyLoading = ref(false)
const modifyFormRef = ref()
const modifyNetworkForm = reactive({
  network_interface_name: "",
  description: "",
  security_group_id: [] as string[]
})
const currentEditNetwork = ref<NetworkInterfaceWrap | null>(null)
// 安全组列表
const securityGroupList = ref<any[]>([])
const securityGroupLoading = ref(false)

// 修改表单规则
const modifyFormRules = {
  network_interface_name: [
    { max: 128, message: "名称最大长度为128个字符", trigger: "blur" }
  ],
  description: [
    { max: 256, message: "描述最大长度为256个字符", trigger: "blur" }
  ]
}

// 获取安全组列表
async function fetchSecurityGroups(regionId: string) {
  if (!selectedMerchant.value) return

  securityGroupLoading.value = true
  try {
    const res = await getSecurityGroupList({
      merchant_id: selectedMerchant.value,
      region_id: [regionId]
    })

    if (res.data && res.data.list) {
      securityGroupList.value = res.data.list
    } else {
      securityGroupList.value = []
    }
  } catch (error) {
    console.error("获取安全组列表失败", error)
    ElMessage.error("获取安全组列表失败")
    securityGroupList.value = []
  } finally {
    securityGroupLoading.value = false
  }
}

// 打开修改弹性网卡对话框
function openModifyDialog(network: NetworkInterfaceWrap) {
  currentEditNetwork.value = network
  modifyNetworkForm.network_interface_name = network.NetworkInterface.NetworkInterfaceId || ""
  modifyNetworkForm.description = ""

  // 设置当前安全组ID
  if (network.NetworkInterface.SecurityGroupIds
    && network.NetworkInterface.SecurityGroupIds.SecurityGroupId) {
    modifyNetworkForm.security_group_id = [...network.NetworkInterface.SecurityGroupIds.SecurityGroupId]
  } else {
    modifyNetworkForm.security_group_id = []
  }

  // 获取安全组列表
  fetchSecurityGroups(network.RegionId)

  modifyDialogVisible.value = true
}

// 提交修改弹性网卡表单
async function submitModifyForm(formEl: any) {
  const hasAccount = accountType.value === "merchant" ? selectedMerchant.value : selectedCloudAccount.value
  if (!formEl || !currentEditNetwork.value || !hasAccount) return

  const network = currentEditNetwork.value
  const regionId = network.RegionId

  await formEl.validate(async (valid: boolean) => {
    if (valid) {
      modifyLoading.value = true
      try {
        const params: any = {
          region_id: regionId,
          network_interface_id: network.NetworkInterface.NetworkInterfaceId,
          network_interface_name: modifyNetworkForm.network_interface_name,
          description: modifyNetworkForm.description,
          security_group_id: modifyNetworkForm.security_group_id
        }

        if (accountType.value === "merchant") {
          params.merchant_id = selectedMerchant.value
        } else {
          params.cloud_account_id = selectedCloudAccount.value
        }

        await modifyNetworkInterfaceAttribute(params)

        ElMessage.success("修改弹性网卡成功")
        modifyDialogVisible.value = false

        // 刷新列表
        setTimeout(() => {
          fetchNetworkList()
        }, 1000)
      } catch (error) {
        console.error("修改弹性网卡失败", error)
        ElMessage.error("修改弹性网卡失败")
      } finally {
        modifyLoading.value = false
      }
    }
  })
}

// 实例绑定对话框
const attachDialogVisible = ref(false)
const attachLoading = ref(false)
const instanceList = ref<Instance[]>([])
const instanceLoadingForAttach = ref(false)
const selectedInstance = ref("")
const currentAttachNetwork = ref<NetworkInterfaceWrap | null>(null)

// 打开实例绑定对话框
async function openAttachDialog(network: NetworkInterfaceWrap) {
  const hasAccount = accountType.value === "merchant" ? selectedMerchant.value : selectedCloudAccount.value
  if (!hasAccount) return

  currentAttachNetwork.value = network
  selectedInstance.value = ""
  attachDialogVisible.value = true
  instanceLoadingForAttach.value = true

  try {
    const params: any = {
      region_id: [network.RegionId]
    }

    if (accountType.value === "merchant") {
      params.merchant_id = selectedMerchant.value
    } else {
      params.cloud_account_id = selectedCloudAccount.value
    }

    const res = await getInstanceList(params)

    if (res.data.list && Array.isArray(res.data.list)) {
      instanceList.value = res.data.list
    } else {
      instanceList.value = []
    }

    if (instanceList.value.length === 0) {
      ElMessage.warning("当前区域没有可用的实例")
    }
  } catch (error) {
    console.error("获取实例列表失败", error)
    ElMessage.error("获取实例列表失败")
    instanceList.value = []
  } finally {
    instanceLoadingForAttach.value = false
  }
}

// 提交绑定实例表单
async function submitAttachForm() {
  const hasAccount = accountType.value === "merchant" ? selectedMerchant.value : selectedCloudAccount.value
  if (!currentAttachNetwork.value || !hasAccount || !selectedInstance.value) {
    ElMessage.warning("请先选择实例")
    return
  }

  attachLoading.value = true
  try {
    const params: any = {
      region_id: currentAttachNetwork.value.RegionId,
      network_interface_id: currentAttachNetwork.value.NetworkInterface.NetworkInterfaceId,
      instance_id: selectedInstance.value
    }

    if (accountType.value === "merchant") {
      params.merchant_id = selectedMerchant.value
    } else {
      params.cloud_account_id = selectedCloudAccount.value
    }

    await attachNetworkInterface(params)

    ElMessage.success("绑定成功")
    attachDialogVisible.value = false

    // 刷新列表
    setTimeout(() => {
      fetchNetworkList()
    }, 2000)
  } catch (error) {
    console.error("绑定弹性网卡失败", error)
    ElMessage.error("绑定弹性网卡失败")
  } finally {
    attachLoading.value = false
  }
}

// 实例详情对话框
const instanceDialogVisible = ref(false)
const instanceDetailLoading = ref(false)
const instanceDetail = ref<Instance | null>(null)

// 复制实例ID到剪贴板
async function copyInstanceId(instanceId: string) {
  if (!instanceId) return

  try {
    // 优先使用现代 Clipboard API（仅在 HTTPS 或 localhost 下可用）
    if (navigator.clipboard && window.isSecureContext) {
      await navigator.clipboard.writeText(instanceId)
      ElMessage.success("实例ID已复制到剪贴板")
    } else {
      // 降级方案：使用传统的 execCommand 方法（兼容 HTTP 环境）
      const textarea = document.createElement("textarea")
      textarea.value = instanceId
      textarea.style.position = "fixed"
      textarea.style.opacity = "0"
      document.body.appendChild(textarea)
      textarea.select()
      const successful = document.execCommand("copy")
      document.body.removeChild(textarea)

      if (successful) {
        ElMessage.success("实例ID已复制到剪贴板")
      } else {
        throw new Error("execCommand 复制失败")
      }
    }
  } catch (error) {
    console.error("复制失败", error)
    ElMessage.error("复制失败，请手动复制")
  }
}

// 查看绑定的ECS实例详情
async function viewEcsInstanceDetail(network: NetworkInterfaceWrap) {
  if (!selectedMerchant.value || !selectedRegions.value || selectedRegions.value.length === 0) {
    ElMessage.warning("商户或区域信息缺失")
    return
  }

  if (!network.NetworkInterface.InstanceId) {
    ElMessage.warning("该弹性网卡未绑定实例")
    return
  }

  instanceDetailLoading.value = true
  instanceDetail.value = null
  instanceDialogVisible.value = true

  try {
    const res = await getInstanceList({
      merchant_id: selectedMerchant.value,
      region_id: selectedRegions.value
    })

    if (res.data.list && Array.isArray(res.data.list)) {
      const instance = res.data.list.find(item => item.InstanceId === network.NetworkInterface.InstanceId)
      if (instance) {
        instanceDetail.value = instance
      } else {
        ElMessage.warning(`未找到ID为 ${network.NetworkInterface.InstanceId} 的实例信息`)
      }
    }
  } catch (error) {
    console.error("获取实例详情失败", error)
    ElMessage.error("获取实例详情失败")
  } finally {
    instanceDetailLoading.value = false
  }
}

// 页面加载时获取商户和区域列表
onMounted(() => {
  loadSelectionFromStorage()

  const fetchInitialData = () => {
    // 处理从EIP页面跳转的情况
    const queryMerchantId = route.query.merchant_id
    const queryNetworkId = route.query.network_interface_id
    const queryRegionId = route.query.region_id

    // 如果URL中有指定的商户ID和网卡ID，优先使用
    if (queryMerchantId && typeof queryMerchantId === "string") {
      const merchantId = Number.parseInt(queryMerchantId)
      if (!Number.isNaN(merchantId)) {
        accountType.value = "merchant"
        selectedMerchant.value = merchantId

        // 获取区域列表后查询网卡列表
        fetchRegionList().then(() => {
          // 等待区域列表加载
          if (regionList.value.length > 0) {
            // 如果提供了区域ID，优先使用，否则使用第一个区域
            if (queryRegionId && typeof queryRegionId === "string") {
              // 验证区域ID是否有效
              const validRegion = regionList.value.find(region => region.RegionId === queryRegionId)
              if (validRegion) {
                selectedRegions.value = [queryRegionId]
              } else {
                // 如果区域ID无效，使用第一个区域
                selectedRegions.value = [regionList.value[0].RegionId]
                console.warn(`区域ID ${queryRegionId} 无效，使用第一个可用区域`)
              }
            } else {
              // 没有提供区域ID，使用第一个区域
              selectedRegions.value = [regionList.value[0].RegionId]
            }

            // 获取网卡列表
            fetchNetworkList().then(() => {
              // 如果有指定的网卡ID，找到并显示详情
              if (queryNetworkId && typeof queryNetworkId === "string") {
                const foundNetwork = networkList.value.find(
                  network => network.NetworkInterface.NetworkInterfaceId === queryNetworkId
                )

                if (foundNetwork) {
                  // 找到网卡后显示详情
                  showNetworkDetail(foundNetwork)
                } else {
                  ElMessage.warning(`未找到ID为 ${queryNetworkId} 的网卡，可能不在当前区域`)
                }
              }
            })
          }
        })
      }
    } else {
      // 如果没有URL参数，根据账号类型获取相应数据
      if (accountType.value === "merchant" && selectedMerchant.value) {
        fetchRegionList()
      } else if (accountType.value === "system" && selectedCloudAccount.value) {
        fetchRegionList()
      }
    }
  }

  // 根据账号类型初始化数据
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
          @click="fetchNetworkList"
        >
          查询
        </el-button>
        <el-button
          type="success"
          :disabled="accountType === 'merchant' ? !selectedMerchant : !selectedCloudAccount"
          @click="openCreateDialog"
        >
          创建弹性网卡
        </el-button>
      </div>
    </el-card>

    <el-card v-if="(accountType === 'merchant' ? selectedMerchant : selectedCloudAccount) && selectedRegions.length > 0" class="table-card">
      <template #header>
        <div class="card-header">
          <span>弹性网卡列表</span>
          <div class="operations">
            <el-button type="primary" size="small" @click="fetchNetworkList">
              刷新
            </el-button>
          </div>
        </div>
      </template>
      <el-table
        v-loading="loading"
        :data="networkList.sort((a, b) => {
          // 优先按照RegionId排序
          const regionCompare = a.RegionId.localeCompare(b.RegionId);
          if (regionCompare !== 0) return regionCompare;

          // 其次按照Type排序（Primary排在前面）
          if (a.NetworkInterface.Type === 'Primary' && b.NetworkInterface.Type !== 'Primary') return -1;
          if (a.NetworkInterface.Type !== 'Primary' && b.NetworkInterface.Type === 'Primary') return 1;

          // 最后按照InstanceId排序
          if (!a.NetworkInterface.InstanceId && b.NetworkInterface.InstanceId) return 1;
          if (a.NetworkInterface.InstanceId && !b.NetworkInterface.InstanceId) return -1;
          return a.NetworkInterface.InstanceId?.localeCompare(b.NetworkInterface.InstanceId || '') || 0;
        })"
        style="width: 100%"
        border
      >
        <el-table-column prop="NetworkInterface.NetworkInterfaceName" label="网卡名称" min-width="180">
          <template #default="scope">
            {{ scope.row.NetworkInterface.NetworkInterfaceName || '-' }}
          </template>
        </el-table-column>
        <el-table-column prop="NetworkInterface.MacAddress" label="MAC地址" min-width="180" />
        <el-table-column label="区域" min-width="120">
          <template #default="scope">
            {{ scope.row.RegionId || "-" }}
          </template>
        </el-table-column>
        <el-table-column prop="NetworkInterface.Type" label="网卡类型" min-width="100">
          <template #default="scope">
            {{ scope.row.NetworkInterface.Type === 'Primary' ? '主网卡' : '辅助网卡' }}
          </template>
        </el-table-column>
        <el-table-column prop="NetworkInterface.Status" label="状态" min-width="100">
          <template #default="scope">
            <el-tag :type="getStatusType(scope.row.NetworkInterface.Status)">
              {{ getStatusText(scope.row.NetworkInterface.Status) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="私网IP" min-width="150">
          <template #default="scope">
            {{ scope.row.NetworkInterface.PrivateIpAddress || "-" }}
          </template>
        </el-table-column>
        <el-table-column label="实例ID" min-width="180">
          <template #default="scope">
            <span v-if="scope.row.NetworkInterface.InstanceId">
              {{ scope.row.NetworkInterface.InstanceId }}
              <el-button
                type="success"
                text
                size="small"
                @click.stop="viewEcsInstanceDetail(scope.row)"
              >
                <el-icon><Monitor /></el-icon>
                查看
              </el-button>
              <el-button
                type="primary"
                text
                size="small"
                :icon="CopyDocument"
                @click.stop="copyInstanceId(scope.row.NetworkInterface.InstanceId)"
              />
            </span>
            <span v-else>-</span>
          </template>
        </el-table-column>
        <el-table-column prop="NetworkInterface.CreationTime" label="创建时间" min-width="180" />
        <el-table-column label="操作" fixed="right" min-width="120">
          <template #default="scope">
            <div class="action-buttons">
              <el-button
                type="primary"
                text
                size="small"
                @click="showNetworkDetail(scope.row)"
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
                    <el-dropdown-item v-if="scope.row.NetworkInterface.Status === 'Available'" @click="openAttachDialog(scope.row)">
                      <el-icon><Link /></el-icon> 绑定实例
                    </el-dropdown-item>
                    <el-dropdown-item v-if="scope.row.NetworkInterface.Status === 'InUse'" @click="confirmDetach(scope.row)">
                      <el-icon><Link /></el-icon> 解绑实例
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

    <!-- 弹性网卡详情对话框 -->
    <el-dialog
      v-model="detailDialogVisible"
      :title="`弹性网卡详情 - ${detailNetwork?.NetworkInterface.NetworkInterfaceName || detailNetwork?.NetworkInterface.NetworkInterfaceId || ''}`"
      width="750px"
      destroy-on-close
    >
      <div v-if="detailNetwork" class="network-detail">
        <el-descriptions :column="2" border>
          <el-descriptions-item label="网卡ID" :span="1">
            {{ detailNetwork.NetworkInterface.NetworkInterfaceId }}
          </el-descriptions-item>
          <el-descriptions-item label="网卡名称" :span="1">
            {{ detailNetwork.NetworkInterface.NetworkInterfaceName || '-' }}
          </el-descriptions-item>
          <el-descriptions-item label="描述" :span="2">
            {{ detailNetwork.NetworkInterface.Description || '-' }}
          </el-descriptions-item>
          <el-descriptions-item label="MAC地址">
            {{ detailNetwork.NetworkInterface.MacAddress || '-' }}
          </el-descriptions-item>
          <el-descriptions-item label="状态">
            <el-tag :type="getStatusType(detailNetwork.NetworkInterface.Status)">
              {{ getStatusText(detailNetwork.NetworkInterface.Status) }}
            </el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="实例ID" v-if="detailNetwork.NetworkInterface.InstanceId">
            {{ detailNetwork.NetworkInterface.InstanceId }}
            <el-button
              type="primary"
              text
              size="small"
              :icon="CopyDocument"
              @click="copyInstanceId(detailNetwork.NetworkInterface.InstanceId)"
            />
          </el-descriptions-item>
          <el-descriptions-item label="网卡类型">
            {{ detailNetwork.NetworkInterface.Type === 'Primary' ? '主网卡' : '辅助网卡' }}
          </el-descriptions-item>
          <el-descriptions-item label="私网IP">
            {{ detailNetwork.NetworkInterface.PrivateIpAddress || '-' }}
          </el-descriptions-item>
          <el-descriptions-item label="创建时间">
            {{ detailNetwork.NetworkInterface.CreationTime }}
          </el-descriptions-item>
        </el-descriptions>

        <!-- VPC信息 -->
        <div class="detail-section">
          <h3 class="detail-title">
            VPC信息
          </h3>
          <el-descriptions :column="2" border>
            <el-descriptions-item label="VPC ID">
              {{ detailNetwork.NetworkInterface.VpcId || '-' }}
            </el-descriptions-item>
            <el-descriptions-item label="交换机 ID">
              {{ detailNetwork.NetworkInterface.VSwitchId || '-' }}
            </el-descriptions-item>
          </el-descriptions>
        </div>

        <!-- 安全组信息 -->
        <div class="detail-section">
          <h3 class="detail-title">
            安全组信息
          </h3>
          <div v-if="detailNetwork.NetworkInterface.SecurityGroupIds && detailNetwork.NetworkInterface.SecurityGroupIds.SecurityGroupId.length">
            <el-descriptions :column="1" border>
              <el-descriptions-item v-for="(sgId, index) in detailNetwork.NetworkInterface.SecurityGroupIds.SecurityGroupId" :key="index" label="安全组ID">
                {{ sgId }}
              </el-descriptions-item>
            </el-descriptions>
          </div>
          <el-empty v-else description="无安全组信息" :image-size="60" />
        </div>

        <!-- IP地址信息 -->
        <div class="detail-section">
          <h3 class="detail-title">
            IP地址信息
          </h3>
          <div v-if="detailNetwork.NetworkInterface.PrivateIpSets && detailNetwork.NetworkInterface.PrivateIpSets.PrivateIpSet.length">
            <el-descriptions v-for="(ip, index) in detailNetwork.NetworkInterface.PrivateIpSets.PrivateIpSet" :key="index" :column="2" border>
              <el-descriptions-item label="私网IP地址">
                {{ ip.PrivateIpAddress }}
              </el-descriptions-item>
              <el-descriptions-item label="公网IP地址" v-if="ip.AssociatedPublicIp && ip.AssociatedPublicIp.PublicIpAddress">
                {{ ip.AssociatedPublicIp.PublicIpAddress }}
              </el-descriptions-item>
              <el-descriptions-item label="EIP实例ID" v-if="ip.AssociatedPublicIp && ip.AssociatedPublicIp.AllocationId">
                {{ ip.AssociatedPublicIp.AllocationId }}
              </el-descriptions-item>
            </el-descriptions>
          </div>
          <el-empty v-else description="无IP地址信息" :image-size="60" />
        </div>
      </div>
      <el-empty v-else description="未找到弹性网卡详情" />
    </el-dialog>

    <!-- 创建弹性网卡对话框 -->
    <el-dialog
      v-model="createDialogVisible"
      title="创建弹性网卡"
      width="600px"
      destroy-on-close
    >
      <el-form
        ref="createFormRef"
        :model="createNetworkForm"
        :rules="createFormRules"
        label-width="120px"
        label-position="right"
      >
        <el-form-item label="区域">
          <el-select
            v-model="createNetworkForm.region_id"
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

        <el-form-item label="选择实例" v-if="createNetworkForm.region_id">
          <el-select
            placeholder="选择实例以获取VSwitch信息"
            style="width: 100%"
            :loading="instancesLoading"
            @change="selectVSwitchFromInstance"
            v-model="selectedInstanceId"
            filterable
            clearable
          >
            <el-option
              v-for="item in instanceListForCreate"
              :key="item.InstanceId"
              :label="`${item.InstanceName || item.InstanceId}(${item.InstanceId})`"
              :value="item.InstanceId"
            >
              <div style="display: flex; justify-content: space-between; align-items: center">
                <span>{{ item.InstanceName || item.InstanceId }}</span>
                <el-tag size="small" :type="item.Status === 'Running' ? 'success' : 'warning'">
                  {{ item.Status }}
                </el-tag>
              </div>
              <div style="font-size: 12px; color: #909399">
                VSwitchId: {{ item.VpcAttributes?.VSwitchId || '无' }}
              </div>
            </el-option>
            <template #empty>
              <el-empty description="暂无实例" :image-size="60" />
            </template>
          </el-select>
          <div class="form-tip">
            选择实例后会自动填充对应的交换机ID
          </div>
          <div v-if="selectedInstanceId" class="selected-instance-info">
            <el-tag type="success">
              已选中实例ID: {{ selectedInstanceId }}
              <el-button
                type="primary"
                text
                size="small"
                :icon="CopyDocument"
                @click.stop="copyInstanceId(selectedInstanceId)"
              />
            </el-tag>
          </div>
        </el-form-item>

        <el-form-item label="交换机ID" prop="vswitch_id">
          <el-input
            v-model="createNetworkForm.vswitch_id"
            placeholder="请输入交换机ID"
            style="width: 100%"
          />
          <div class="form-tip">
            例如：vsw-bp1s5fnvk4gn2tw***
          </div>
        </el-form-item>

        <el-form-item label="弹性网卡名称" prop="network_interface_name">
          <el-input
            v-model="createNetworkForm.network_interface_name"
            placeholder="请输入弹性网卡名称"
            style="width: 100%"
          />
          <div class="form-tip">
            弹性网卡名称，长度为2~128个字符
          </div>
        </el-form-item>

        <el-form-item label="安全组" prop="security_group_id" v-if="createNetworkForm.region_id">
          <el-select
            v-model="createNetworkForm.security_group_id"
            placeholder="请选择安全组"
            style="width: 100%"
            :loading="securityGroupsLoading"
            filterable
          >
            <el-option
              v-for="item in securityGroupListForCreate"
              :key="item.SecurityGroup.SecurityGroupId"
              :label="item.SecurityGroup.SecurityGroupName || item.SecurityGroup.SecurityGroupId"
              :value="item.SecurityGroup.SecurityGroupId"
            >
              <div style="display: flex; justify-content: space-between; align-items: center">
                <span>{{ item.SecurityGroup.SecurityGroupName || item.SecurityGroup.SecurityGroupId }}</span>
              </div>
              <div style="font-size: 12px; color: #909399">
                ID: {{ item.SecurityGroup.SecurityGroupId }}
              </div>
              <div style="font-size: 12px; color: #909399" v-if="item.SecurityGroup.VpcId">
                VPC: {{ item.SecurityGroup.VpcId }}
              </div>
            </el-option>
            <template #empty>
              <el-empty description="暂无安全组" :image-size="60" />
            </template>
          </el-select>
          <div class="form-tip">
            选择安全组或手动输入安全组ID
          </div>
        </el-form-item>

        <el-form-item label="安全组ID" prop="security_group_id" v-else>
          <el-input
            v-model="createNetworkForm.security_group_id"
            placeholder="请输入安全组ID"
            style="width: 100%"
          />
          <div class="form-tip">
            例如：sg-bp1fg655nh68xyz***
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

    <!-- 修改弹性网卡对话框 -->
    <el-dialog
      v-model="modifyDialogVisible"
      title="修改弹性网卡"
      width="500px"
      destroy-on-close
    >
      <el-form
        ref="modifyFormRef"
        :model="modifyNetworkForm"
        :rules="modifyFormRules"
        label-width="100px"
      >
        <el-form-item label="名称" prop="network_interface_name">
          <el-input v-model="modifyNetworkForm.network_interface_name" placeholder="请输入弹性网卡名称" />
        </el-form-item>
        <el-form-item label="描述" prop="description">
          <el-input v-model="modifyNetworkForm.description" placeholder="请输入弹性网卡描述" />
        </el-form-item>
        <el-form-item label="安全组" prop="security_group_id">
          <el-select
            v-model="modifyNetworkForm.security_group_id"
            multiple
            placeholder="请选择安全组"
            style="width: 100%"
            :loading="securityGroupLoading"
          >
            <el-option
              v-for="item in securityGroupList"
              :key="item.SecurityGroup.SecurityGroupId"
              :label="item.SecurityGroup.SecurityGroupName || item.SecurityGroup.SecurityGroupId"
              :value="item.SecurityGroup.SecurityGroupId"
            >
              <span style="float: left">{{ item.SecurityGroup.SecurityGroupName || item.SecurityGroup.SecurityGroupId }}</span>
              <span style="float: right; color: #8492a6; font-size: 13px">{{ item.SecurityGroup.SecurityGroupId }}</span>
            </el-option>
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="modifyDialogVisible = false">
          取消
        </el-button>
        <el-button
          type="primary"
          @click="submitModifyForm(modifyFormRef)"
          :loading="modifyLoading"
        >
          确定
        </el-button>
      </template>
    </el-dialog>

    <!-- 绑定实例对话框 -->
    <el-dialog
      v-model="attachDialogVisible"
      title="绑定ECS实例"
      width="600px"
      destroy-on-close
    >
      <div v-loading="instanceLoadingForAttach">
        <el-form label-width="120px" label-position="right">
          <el-form-item label="选择实例">
            <el-select
              v-model="selectedInstance"
              placeholder="请选择ECS实例"
              style="width: 100%"
              :disabled="instanceLoadingForAttach || instanceList.length === 0"
            >
              <el-option
                v-for="item in instanceList"
                :key="item.InstanceId"
                :label="`${item.InstanceName || item.InstanceId}(${item.InstanceId})`"
                :value="item.InstanceId"
              />
            </el-select>
            <div class="form-tip" v-if="instanceList.length === 0">
              当前区域没有可用的ECS实例，请先创建实例
            </div>
          </el-form-item>
        </el-form>
      </div>

      <template #footer>
        <div class="dialog-footer">
          <el-button @click="attachDialogVisible = false">
            取消
          </el-button>
          <el-button
            type="primary"
            :loading="attachLoading"
            :disabled="!selectedInstance"
            @click="submitAttachForm"
          >
            绑定
          </el-button>
        </div>
      </template>
    </el-dialog>

    <!-- 实例详情对话框 -->
    <el-dialog
      v-model="instanceDialogVisible"
      title="ECS实例详情"
      width="600px"
      destroy-on-close
    >
      <div v-loading="instanceDetailLoading">
        <el-descriptions :column="2" border>
          <el-descriptions-item label="实例ID">
            {{ instanceDetail?.InstanceId }}
            <el-button
              v-if="instanceDetail?.InstanceId"
              type="primary"
              text
              size="small"
              :icon="CopyDocument"
              @click="copyInstanceId(instanceDetail!.InstanceId)"
            />
          </el-descriptions-item>
          <el-descriptions-item label="实例名称">
            {{ instanceDetail?.InstanceName }}
          </el-descriptions-item>
          <el-descriptions-item label="状态">
            {{ instanceDetail?.Status }}
          </el-descriptions-item>
          <el-descriptions-item label="创建时间">
            {{ instanceDetail?.CreationTime }}
          </el-descriptions-item>
          <el-descriptions-item label="实例类型">
            {{ instanceDetail?.InstanceType }}
          </el-descriptions-item>
          <el-descriptions-item label="CPU">
            {{ instanceDetail?.Cpu }} 核
          </el-descriptions-item>
          <el-descriptions-item label="内存">
            {{ instanceDetail?.Memory ? Math.floor(Number(instanceDetail.Memory) / 1024) : 0 }} GB
          </el-descriptions-item>
          <el-descriptions-item label="操作系统">
            {{ instanceDetail?.OSName }}
          </el-descriptions-item>
          <el-descriptions-item label="网络类型">
            {{ instanceDetail?.InstanceNetworkType }}
          </el-descriptions-item>
          <el-descriptions-item label="公网IP" v-if="instanceDetail?.EipAddress?.IpAddress">
            {{ instanceDetail.EipAddress.IpAddress }}
          </el-descriptions-item>
        </el-descriptions>
      </div>
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

.network-detail {
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

.selected-instance-info {
  margin-top: 10px;
}

.el-button[size="small"] {
  padding: 5px 8px;
}

.copy-button {
  margin-left: 5px;
}
</style>
