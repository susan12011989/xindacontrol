<script setup lang="ts">
import type { Region } from "@/pages/cloud/aliyun/apis/type"
import type { Instance, InstanceBindingResp, InstanceItem } from "@/pages/cloud/aliyun/instances/apis/type"
import type { Merchant, MerchantRegions } from "@/pages/dashboard/apis/type"
import type { CloudAccountOption } from "@@/apis/cloud_account/type"
import { regionListApi } from "@/pages/cloud/aliyun/apis"
import { bindInstanceMerchant, createSecondaryNic, deployTunnelServer, getInstanceList, modifyInstanceAttribute, modifyInstanceChargeTypePostPaid, operateInstance, registerInstanceWithSSHKey, unbindInstanceMerchant } from "@/pages/cloud/aliyun/instances/apis"
import { getSecurityGroupList } from "@/pages/cloud/aliyun/securitygroup/apis"
import { merchantQueryApi } from "@/pages/dashboard/apis"
import { getCloudAccountOptions } from "@@/apis/cloud_account"
import { ArrowDown, Delete, Edit, Key, Location, RefreshRight, User, VideoPause, VideoPlay } from "@element-plus/icons-vue"
import { ElMessage, ElMessageBox } from "element-plus"
import { onMounted, reactive, ref, watch } from "vue"
import { useRouter } from "vue-router"

defineOptions({
  name: "CloudInstances"
})
const router = useRouter()

// localStorage 存储键名
const STORAGE_ACCOUNT_TYPE_KEY = "cloud_account_type"
const STORAGE_CLOUD_ACCOUNT_KEY = "cloud_selected_cloud_account"
const STORAGE_MERCHANT_KEY = "cloud_selected_merchant"
const STORAGE_REGION_KEY = "cloud_selected_region"

// 数据状态
const loading = ref(false)
const accountType = ref<string>("system") // merchant: 商户类型, system: 系统类型
const cloudAccountList = ref<CloudAccountOption[]>([])
const selectedCloudAccount = ref<number>()
const merchantList = ref<Merchant[]>([])
const regionList = ref<Region[]>([])
const instanceList = ref<Instance[]>([])
const selectedMerchant = ref<number>()
const selectedRegions = ref<string[]>([])
// 选中的实例列表用于批量操作
const selectedInstances = ref<Instance[]>([])

// 修改实例属性对话框
const modifyDialogVisible = ref(false)
const modifyLoading = ref(false)
const modifyFormRef = ref()
const modifyInstanceForm = reactive({
  instance_name: "",
  description: "",
  password: "",
  security_group_id: [] as string[]
})
const currentEditInstance = ref<Instance | null>(null)
// 安全组列表
const securityGroupList = ref<any[]>([])
const securityGroupLoading = ref(false)

// 商户绑定相关状态
const bindingMap = ref<Record<string, InstanceBindingResp>>({})
const bindMerchantDialogVisible = ref(false)
const bindMerchantLoading = ref(false)
const bindMerchantForm = reactive({
  instance_id: "",
  region_id: "",
  merchant_id: undefined as number | undefined
})
// 绑定对话框用的商户列表（复用 merchantList，它已在选择商户类型时加载）
const allMerchantList = ref<Merchant[]>([])

// 创建辅助网卡相关状态
const createNicDialogVisible = ref(false)
const createNicLoading = ref(false)
const createNicOutput = ref("")

// 一键部署隧道服务器
const tunnelDeployDialogVisible = ref(false)
const tunnelDeployFormVisible = ref(true)
const tunnelDeployLoading = ref(false)
const tunnelDeployOutput = ref("")
const tunnelDeployResults = ref<any[]>([])
const tunnelDeployForm = reactive({
  server_name: "",
  server_count: 1,
  instance_type: "ecs.t5.large",
  bandwidth: "10",
  eip_count: 4
})

// 修改表单规则
const modifyFormRules = {
  instance_name: [
    { max: 128, message: "名称最大长度为128个字符", trigger: "blur" }
  ],
  description: [
    { max: 256, message: "描述最大长度为256个字符", trigger: "blur" }
  ],
  password: [
    {
      validator: (rule: any, value: string, callback: any) => {
        if (!value) { // 如果为空，不进行验证，允许为空
          callback()
        } else if (!/^(?=.*[a-z])(?=.*[A-Z])(?=.*\d)(?=.*[^\da-zA-Z]).{8,30}$/.test(value)) {
          callback(new Error("密码必须包含大小写字母、数字和特殊字符，长度8-30"))
        } else {
          callback()
        }
      },
      trigger: "blur"
    }
  ]
}

// 从 localStorage 读取已保存的选择
function loadSelectionFromStorage() {
  try {
    const savedAccountType = localStorage.getItem(STORAGE_ACCOUNT_TYPE_KEY)
    const savedCloudAccountId = localStorage.getItem(STORAGE_CLOUD_ACCOUNT_KEY)
    const savedMerchantId = localStorage.getItem(STORAGE_MERCHANT_KEY)
    const savedRegionIds = localStorage.getItem(STORAGE_REGION_KEY)

    if (savedAccountType) {
      accountType.value = savedAccountType
      console.log("从localStorage读取账号类型:", accountType.value)
    }

    if (savedCloudAccountId) {
      selectedCloudAccount.value = Number(savedCloudAccountId)
      console.log("从localStorage读取云账号ID:", selectedCloudAccount.value)
    }

    if (savedMerchantId) {
      selectedMerchant.value = Number(savedMerchantId)
      console.log("从localStorage读取商户ID:", selectedMerchant.value)
    }

    if (savedRegionIds) {
      selectedRegions.value = JSON.parse(savedRegionIds)
      console.log("从localStorage读取区域ID:", selectedRegions.value)
    }
  } catch (error) {
    console.error("读取localStorage数据失败:", error)
  }
}

// 监听选择变化，保存到localStorage
watch(accountType, (newValue) => {
  if (newValue) {
    localStorage.setItem(STORAGE_ACCOUNT_TYPE_KEY, newValue)
  }
}, { deep: true })

watch(selectedCloudAccount, (newValue) => {
  if (newValue) {
    localStorage.setItem(STORAGE_CLOUD_ACCOUNT_KEY, newValue.toString())
  } else {
    localStorage.removeItem(STORAGE_CLOUD_ACCOUNT_KEY)
  }
}, { deep: true })

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
  // 清空区域选择
  selectedRegions.value = []
  instanceList.value = []

  if (accountType.value === "system") {
    // 系统类型：清空商户选择，获取云账号列表
    selectedMerchant.value = undefined
    fetchCloudAccountList()
  } else {
    // 商户类型：清空云账号选择，获取商户列表
    selectedCloudAccount.value = undefined
    fetchMerchantList()
  }
}

// 处理云账号切换
function handleCloudAccountChange() {
  // 清空区域选择和实例列表
  selectedRegions.value = []
  instanceList.value = []

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

// 获取实例列表
async function fetchInstanceList() {
  // 检查必要的选择条件
  const hasAccount = accountType.value === "merchant" ? selectedMerchant.value : selectedCloudAccount.value
  if (!hasAccount || !selectedRegions.value || selectedRegions.value.length === 0) {
    console.log("账号或区域未选择，不获取实例列表")
    return
  }

  console.log("开始获取实例列表, 类型:", accountType.value, "账号:", hasAccount, "区域:", selectedRegions.value)
  loading.value = true
  try {
    // 根据账号类型构建不同的请求参数
    const params: any = {
      region_id: selectedRegions.value
    }
    if (accountType.value === "merchant") {
      params.merchant_id = selectedMerchant.value
    } else {
      params.cloud_account_id = selectedCloudAccount.value
    }

    const res = await getInstanceList(params)

    console.log("获取实例列表成功:", res.data.list)
    console.log("网卡EIP映射:", res.data.nic_eip_map)

    // 兼容返回空数组的情况
    if (!res.data.list || !Array.isArray(res.data.list)) {
      console.log("实例列表为空或格式不正确，设置为空数组")
      instanceList.value = []
    } else {
      // 使用网卡EIP映射增强实例数据
      const nicEipMap = res.data.nic_eip_map || {}
      const enhancedList = res.data.list.map((instance: Instance) => {
        // 如果实例有网卡信息，填充EIP数据
        if (instance.NetworkInterfaces?.NetworkInterface) {
          instance.NetworkInterfaces.NetworkInterface = instance.NetworkInterfaces.NetworkInterface.map((nic) => {
            const nicId = nic.NetworkInterfaceId
            const eipList = nicEipMap[nicId] || []

            // 确保PrivateIpSets存在
            if (!nic.PrivateIpSets) {
              nic.PrivateIpSets = { PrivateIpSet: [] }
            }

            // 根据EIP映射填充AssociatedPublicIp
            if (eipList.length > 0 && nic.PrivateIpSets.PrivateIpSet) {
              nic.PrivateIpSets.PrivateIpSet = nic.PrivateIpSets.PrivateIpSet.map((privateIp) => {
                // 查找匹配的公网IP
                const matchedEip = eipList.find((eip: any) => eip.PrivateIpAddress === privateIp.PrivateIpAddress)
                if (matchedEip) {
                  privateIp.AssociatedPublicIp = {
                    PublicIpAddress: matchedEip.PublicIpAddress,
                    AllocationId: matchedEip.AllocationId
                  }
                }
                return privateIp
              })
            }

            return nic
          })
        }
        return instance
      })

      instanceList.value = enhancedList
    }

    // 存储商户绑定信息
    bindingMap.value = res.data.bindings || {}

    // 清空选中的实例
    selectedInstances.value = []

    // 如果列表为空，显示提示
    if (instanceList.value.length === 0) {
      ElMessage.info("当前区域暂无实例")
    }
  } catch (error) {
    console.error("获取实例列表失败", error)
    ElMessage.error("获取实例列表失败")
    instanceList.value = []
  } finally {
    loading.value = false
  }
}

// 商户变化处理
function handleMerchantChange(value: number | undefined) {
  console.log("商户变更为:", value)
  selectedRegions.value = []
  instanceList.value = []
  selectedInstances.value = []

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

// 处理实例选择变化
function handleSelectionChange(selection: Instance[]) {
  selectedInstances.value = selection
  console.log("已选择的实例:", selection)
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
  instanceList.value = []
}

// 操作实例（启动、停止、重启、删除）
async function handleOperate(instance: Instance, operation: string) {
  const hasAccount = accountType.value === "merchant" ? selectedMerchant.value : selectedCloudAccount.value
  if (!hasAccount) return

  try {
    const params: any = {
      region_id: instance.RegionId,
      instance_id: instance.InstanceId,
      operation
    }

    if (accountType.value === "merchant") {
      params.merchant_id = selectedMerchant.value
    } else {
      params.cloud_account_id = selectedCloudAccount.value
    }

    await operateInstance(params)

    ElMessage.success(`${getOperationText(operation)}成功`)
    // 延迟刷新实例列表，等待操作完成
    setTimeout(() => {
      fetchInstanceList()
    }, 2000)
  } catch (error) {
    console.error(`${getOperationText(operation)}失败`, error)
    ElMessage.error(`${getOperationText(operation)}失败`)
  }
}

// 状态文本和类型处理
function getStatusText(status: string) {
  const statusMap: Record<string, string> = {
    Running: "运行中",
    Stopped: "已停止",
    Starting: "启动中",
    Stopping: "停止中",
    Creating: "创建中",
    Deleted: "已删除"
  }
  return statusMap[status] || status
}

function getStatusType(status: string) {
  const typeMap: Record<string, "success" | "warning" | "info" | "primary" | "danger"> = {
    Running: "success",
    Stopped: "info",
    Starting: "warning",
    Stopping: "warning",
    Creating: "primary",
    Deleted: "danger"
  }
  return typeMap[status] || ""
}

function getOperationText(operation: string) {
  const operationMap: Record<string, string> = {
    start: "启动",
    stop: "停止",
    restart: "重启",
    delete: "删除"
  }
  return operationMap[operation] || operation
}

// 实例详情
const detailDialogVisible = ref(false)
const detailInstance = ref<Instance | null>(null)

// 显示实例详情
function showInstanceDetail(instance: Instance) {
  detailInstance.value = instance
  detailDialogVisible.value = true
}

// 获取网络计费类型文本
function getInternetChargeType(type: string) {
  const typeMap: Record<string, string> = {
    PayByBandwidth: "按带宽计费",
    PayByTraffic: "按流量计费"
  }
  return typeMap[type] || type
}

// 跳转到创建实例页面
function goToCreatePage() {
  if (accountType.value === "merchant") {
    if (!selectedMerchant.value) {
      ElMessage.warning("请先选择商户")
      return
    }
    router.push({
      path: "/cloud/aliyun/instances/create",
      query: {
        merchant_id: selectedMerchant.value.toString()
      }
    })
  } else {
    if (!selectedCloudAccount.value) {
      ElMessage.warning("请先选择云账号")
      return
    }
    router.push({
      path: "/cloud/aliyun/instances/create",
      query: {
        cloud_account_id: selectedCloudAccount.value.toString()
      }
    })
  }
}

// 确认删除
function confirmDelete(instance: Instance) {
  ElMessageBox.confirm(
    `确定要删除实例 ${instance.InstanceId} 吗？`,
    "删除确认",
    {
      confirmButtonText: "确定",
      cancelButtonText: "取消",
      type: "warning"
    }
  )
    .then(() => {
      handleOperate(instance, "delete")
    })
    .catch(() => {
      // 用户取消删除操作
    })
}

// 转为按量付费
function handleToPostPaid(instance: Instance) {
  const hasAccount = accountType.value === "merchant" ? selectedMerchant.value : selectedCloudAccount.value
  if (!hasAccount) return

  ElMessageBox.confirm(
    `确定要将实例 ${instance.InstanceId} 转为按量付费吗？`,
    "转换计费方式确认",
    {
      confirmButtonText: "确定",
      cancelButtonText: "取消",
      type: "warning"
    }
  )
    .then(async () => {
      try {
        const params: any = {
          region_id: instance.RegionId,
          instance_id: instance.InstanceId
        }

        if (accountType.value === "merchant") {
          params.merchant_id = selectedMerchant.value
        } else {
          params.cloud_account_id = selectedCloudAccount.value
        }

        await modifyInstanceChargeTypePostPaid(params)
        ElMessage.success("转换为按量付费成功")
        // 刷新列表
        setTimeout(() => {
          fetchInstanceList()
        }, 1000)
      } catch (error) {
        console.error("转换为按量付费失败", error)
        ElMessage.error("转换为按量付费失败")
      }
    })
    .catch(() => {
      // 用户取消操作
    })
}

// 获取安全组列表
async function fetchSecurityGroups(regionId: string) {
  const hasAccount = accountType.value === "merchant" ? selectedMerchant.value : selectedCloudAccount.value
  if (!hasAccount) return

  securityGroupLoading.value = true
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

// 打开修改实例属性对话框
function openModifyDialog(instance: Instance) {
  currentEditInstance.value = instance
  modifyInstanceForm.instance_name = instance.InstanceName || ""
  modifyInstanceForm.description = instance.Description || ""
  modifyInstanceForm.password = ""

  // 初始化安全组ID
  if (instance.SecurityGroupIds && instance.SecurityGroupIds.SecurityGroupId) {
    modifyInstanceForm.security_group_id = [...instance.SecurityGroupIds.SecurityGroupId]
  } else {
    modifyInstanceForm.security_group_id = []
  }

  // 获取安全组列表
  fetchSecurityGroups(instance.RegionId)

  modifyDialogVisible.value = true
}

// 提交修改实例属性表单
async function submitModifyForm(formEl: any) {
  const hasAccount = accountType.value === "merchant" ? selectedMerchant.value : selectedCloudAccount.value
  if (!formEl || !currentEditInstance.value || !hasAccount) return

  const instance = currentEditInstance.value
  const regionId = instance.RegionId

  await formEl.validate(async (valid: boolean) => {
    if (valid) {
      modifyLoading.value = true
      try {
        // 构建请求参数
        const params: any = {
          region_id: regionId,
          instance_id: instance.InstanceId,
          instance_name: modifyInstanceForm.instance_name,
          description: modifyInstanceForm.description,
          security_group_id: modifyInstanceForm.security_group_id
        }

        // 根据账号类型添加对应参数
        if (accountType.value === "merchant") {
          params.merchant_id = selectedMerchant.value
        } else {
          params.cloud_account_id = selectedCloudAccount.value
        }

        // 只有密码不为空时才传递密码参数
        if (modifyInstanceForm.password) {
          params.password = modifyInstanceForm.password
        }

        await modifyInstanceAttribute(params)

        ElMessage.success("修改实例属性成功")
        modifyDialogVisible.value = false

        // 刷新列表
        setTimeout(() => {
          fetchInstanceList()
        }, 1000)
      } catch (error) {
        console.error("修改实例属性失败", error)
        ElMessage.error("修改实例属性失败")
      } finally {
        modifyLoading.value = false
      }
    }
  })
}

// 打开创建辅助网卡对话框
function openCreateSecondaryNicDialog() {
  if (selectedInstances.value.length === 0) {
    ElMessage.warning("请先选择实例")
    return
  }

  // 弹出确认对话框
  ElMessageBox.confirm(
    `确定为选中的 ${selectedInstances.value.length} 个实例创建辅助网卡吗？`,
    "创建辅助网卡",
    {
      confirmButtonText: "确定",
      cancelButtonText: "取消",
      type: "warning"
    }
  )
    .then(() => {
      // 用户确认，打开进度对话框并开始创建
      createNicDialogVisible.value = true
      createNicOutput.value = ""
      handleCreateSecondaryNic()
    })
    .catch(() => {
      // 用户取消
      ElMessage.info("已取消创建辅助网卡")
    })
}

// 创建并绑定辅助网卡
function handleCreateSecondaryNic() {
  const hasAccount = accountType.value === "merchant" ? selectedMerchant.value : selectedCloudAccount.value
  if (!hasAccount || selectedInstances.value.length === 0) {
    ElMessage.warning("请先选择需要创建辅助网卡的实例")
    return
  }

  createNicLoading.value = true
  createNicOutput.value = "开始创建辅助网卡...\n"

  // 构建请求参数
  const instances: InstanceItem[] = selectedInstances.value.map(instance => ({
    instance_id: instance.InstanceId,
    region_id: instance.RegionId
  }))

  const params: any = {
    instances
  }

  if (accountType.value === "merchant") {
    params.merchant_id = selectedMerchant.value
  } else {
    params.cloud_account_id = selectedCloudAccount.value
  }

  createSecondaryNic(
    params,
    (output, isComplete) => {
      createNicOutput.value += `${output}\n`

      if (isComplete) {
        createNicLoading.value = false
        ElMessage.success("创建并绑定辅助网卡操作完成")
        // 刷新实例列表
        setTimeout(() => {
          fetchInstanceList()
        }, 1000)
      }
    },
    (error) => {
      console.error("创建辅助网卡失败", error)
      createNicOutput.value += `创建辅助网卡失败: ${error.message || JSON.stringify(error)}\n`
      createNicLoading.value = false
      ElMessage.error("创建辅助网卡失败")
    }
  )
}

// ========== 一键部署隧道服务器 ==========
function openTunnelDeployDialog() {
  if (!selectedCloudAccount.value) {
    ElMessage.warning("请先选择云账号")
    return
  }
  if (selectedRegions.value.length === 0) {
    ElMessage.warning("请先选择地域")
    return
  }
  tunnelDeployDialogVisible.value = true
  tunnelDeployFormVisible.value = true
  tunnelDeployOutput.value = ""
  tunnelDeployResults.value = []
}

function handleTunnelDeploy() {
  if (!tunnelDeployForm.server_name) {
    ElMessage.warning("请填写服务器名称")
    return
  }
  tunnelDeployFormVisible.value = false
  tunnelDeployLoading.value = true
  tunnelDeployOutput.value = "开始部署隧道服务器...\n"

  deployTunnelServer(
    {
      cloud_account_id: selectedCloudAccount.value!,
      region_id: selectedRegions.value[0],
      server_name: tunnelDeployForm.server_name,
      server_count: tunnelDeployForm.server_count,
      instance_type: tunnelDeployForm.instance_type,
      bandwidth: tunnelDeployForm.bandwidth,
      eip_count: tunnelDeployForm.eip_count
    },
    (output, isComplete, results) => {
      if (output) {
        tunnelDeployOutput.value += `${output}\n`
      }
      if (results) {
        tunnelDeployResults.value = results
      }
      if (isComplete) {
        tunnelDeployLoading.value = false
        ElMessage.success("隧道服务器部署完成")
        setTimeout(() => fetchInstanceList(), 1000)
      }
    },
    (error) => {
      tunnelDeployOutput.value += `部署失败: ${error.message || JSON.stringify(error)}\n`
      tunnelDeployLoading.value = false
      ElMessage.error("部署失败")
    }
  )
}

function downloadPrivateKey(serverName: string, privateKey: string) {
  const blob = new Blob([privateKey], { type: "text/plain" })
  const url = URL.createObjectURL(blob)
  const a = document.createElement("a")
  a.href = url
  a.download = `${serverName}.pem`
  a.click()
  URL.revokeObjectURL(url)
}

// 创建服务器
async function handleCreateServer(instance: Instance) {
  // 收集所有公网IP
  const publicIps: string[] = []

  // 从 PublicIpAddress 获取公网IP
  if (instance.PublicIpAddress && instance.PublicIpAddress.IpAddress) {
    publicIps.push(...instance.PublicIpAddress.IpAddress)
  }

  // 从 EipAddress 获取弹性公网IP
  if (instance.EipAddress && instance.EipAddress.IpAddress) {
    publicIps.push(instance.EipAddress.IpAddress)
  }

  // 从网卡获取绑定的公网IP
  if (instance.NetworkInterfaces?.NetworkInterface) {
    for (const nic of instance.NetworkInterfaces.NetworkInterface) {
      if (nic.PrivateIpSets?.PrivateIpSet) {
        for (const privateIp of nic.PrivateIpSets.PrivateIpSet) {
          if (privateIp.AssociatedPublicIp?.PublicIpAddress) {
            publicIps.push(privateIp.AssociatedPublicIp.PublicIpAddress)
          }
        }
      }
    }
  }

  // 去重
  const uniquePublicIps = Array.from(new Set(publicIps))

  // 如果没有公网IP，提示错误
  if (uniquePublicIps.length === 0) {
    ElMessage.warning("该实例没有公网IP，无法创建服务器")
    return
  }

  // 输入服务器名称
  ElMessageBox.prompt(
    `将为实例的 ${uniquePublicIps.length} 个公网IP创建服务器，请输入服务器名称：`,
    "创建服务器",
    {
      confirmButtonText: "确定",
      cancelButtonText: "取消",
      inputPattern: /\S+/,
      inputErrorMessage: "服务器名称不能为空",
      inputValue: instance.InstanceName || ""
    }
  )
    .then(async ({ value }) => {
      const serverName = value.trim()
      if (!serverName) {
        ElMessage.warning("服务器名称不能为空")
        return
      }

      let successCount = 0
      let failCount = 0
      const errors: string[] = []

      // 为每个公网IP创建服务器（自动创建SSH密钥）
      const privateKeys: { serverName: string; host: string; privateKey: string }[] = []

      for (let index = 0; index < uniquePublicIps.length; index++) {
        const ip = uniquePublicIps[index]
        // 如果有多个IP，添加数字后缀；如果只有一个IP，不添加后缀
        const finalServerName = uniquePublicIps.length > 1 ? `${serverName}-${index + 1}` : serverName

        try {
          // 使用新API：自动创建SSH密钥并绑定到实例
          const result = await registerInstanceWithSSHKey({
            cloud_account_id: selectedCloudAccount.value!,
            region_id: instance.RegionId,
            instance_id: instance.InstanceId,
            server_name: finalServerName,
            server_type: 2, // 2-系统服务器
            public_ip: ip
          })
          successCount++
          // 收集私钥信息
          if (result.data?.private_key) {
            privateKeys.push({
              serverName: result.data.server_name || finalServerName,
              host: result.data.host || ip,
              privateKey: result.data.private_key
            })
          }
        } catch (error: any) {
          console.error(`创建服务器失败 (IP: ${ip})`, error)
          errors.push(`${ip}: ${error.message || "未知错误"}`)
          failCount++
        }
      }

      // 显示结果
      if (failCount === 0) {
        ElMessage.success(`成功创建 ${successCount} 个服务器（已自动配置SSH密钥）`)
      } else if (successCount === 0) {
        ElMessage.error(`创建服务器失败: ${errors.join("; ")}`)
      } else {
        ElMessage.warning(`创建完成：成功 ${successCount} 个，失败 ${failCount} 个`)
      }

      // 如果有私钥，提示用户下载
      if (privateKeys.length > 0) {
        ElMessageBox.confirm(
          `已为 ${privateKeys.length} 个服务器创建SSH密钥，请立即下载保存私钥！私钥仅显示一次，关闭后无法再次获取。`,
          "下载SSH私钥",
          {
            confirmButtonText: "下载私钥",
            cancelButtonText: "稍后下载",
            type: "warning",
            closeOnClickModal: false
          }
        ).then(() => {
          // 下载每个私钥
          privateKeys.forEach((item, idx) => {
            const blob = new Blob([item.privateKey], { type: "text/plain" })
            const url = URL.createObjectURL(blob)
            const a = document.createElement("a")
            a.href = url
            a.download = `${item.serverName}_${item.host}.pem`
            document.body.appendChild(a)
            a.click()
            document.body.removeChild(a)
            URL.revokeObjectURL(url)
            // 多个文件时稍微延迟
            if (idx < privateKeys.length - 1) {
              setTimeout(() => {}, 500)
            }
          })
          ElMessage.success("私钥已下载，请妥善保管！")
        }).catch(() => {
          // 用户选择稍后下载，显示私钥内容供复制
          const keyContent = privateKeys.map(item =>
            `# 服务器: ${item.serverName} (${item.host})\n${item.privateKey}`
          ).join("\n\n")

          ElMessageBox.alert(
            `<div style="max-height: 400px; overflow: auto;">
              <p style="color: #E6A23C; margin-bottom: 10px;">⚠️ 请复制并保存以下私钥，关闭后将无法再次获取！</p>
              <pre style="background: #f5f5f5; padding: 10px; font-size: 12px; white-space: pre-wrap; word-break: break-all;">${keyContent}</pre>
            </div>`,
            "SSH私钥内容",
            {
              dangerouslyUseHTMLString: true,
              confirmButtonText: "我已保存",
              closeOnClickModal: false
            }
          )
        })
      }
    })
    .catch(() => {
      // 用户取消
    })
}

// ========== 商户绑定操作 ==========

// 加载全部商户列表（用于绑定对话框）
async function fetchAllMerchantList() {
  if (allMerchantList.value.length > 0) return
  try {
    const res = await merchantQueryApi({ page: 1, size: 9000 })
    allMerchantList.value = res.data.list || []
  } catch (error) {
    console.error("获取商户列表失败", error)
  }
}

// 打开绑定商户对话框
async function openBindMerchantDialog(instance: Instance) {
  bindMerchantForm.instance_id = instance.InstanceId
  bindMerchantForm.region_id = instance.RegionId
  // 如果已有绑定，预填商户
  const existing = bindingMap.value[instance.InstanceId]
  bindMerchantForm.merchant_id = existing?.merchant_id || undefined
  await fetchAllMerchantList()
  bindMerchantDialogVisible.value = true
}

// 执行绑定商户
async function handleBindMerchant() {
  if (!bindMerchantForm.merchant_id) {
    ElMessage.warning("请选择商户")
    return
  }
  bindMerchantLoading.value = true
  try {
    await bindInstanceMerchant({
      instance_id: bindMerchantForm.instance_id,
      region_id: bindMerchantForm.region_id,
      merchant_id: bindMerchantForm.merchant_id
    })
    ElMessage.success("绑定成功")
    bindMerchantDialogVisible.value = false
    // 更新本地绑定信息
    const merchant = allMerchantList.value.find(m => m.id === bindMerchantForm.merchant_id)
    if (merchant) {
      bindingMap.value[bindMerchantForm.instance_id] = {
        instance_id: bindMerchantForm.instance_id,
        merchant_id: merchant.id,
        merchant_name: merchant.name,
        merchant_no: merchant.no || ""
      }
    }
  } catch (error: any) {
    ElMessage.error(error.message || "绑定失败")
  } finally {
    bindMerchantLoading.value = false
  }
}

// 解绑商户
async function handleUnbindMerchant(instance: Instance) {
  try {
    await ElMessageBox.confirm(
      `确定解绑实例 "${instance.InstanceName || instance.InstanceId}" 的商户？`,
      "解绑商户",
      { type: "warning" }
    )
    await unbindInstanceMerchant({ instance_id: instance.InstanceId })
    ElMessage.success("解绑成功")
    delete bindingMap.value[instance.InstanceId]
  } catch (error: any) {
    if (error !== "cancel") {
      ElMessage.error(error.message || "解绑失败")
    }
  }
}

// 页面加载时获取商户和区域列表
onMounted(() => {
  // 加载保存的选择
  loadSelectionFromStorage()

  // 根据账号类型加载相应数据
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
          @click="fetchInstanceList"
        >
          查询
        </el-button>
        <el-button
          type="success"
          :disabled="accountType === 'merchant' ? !selectedMerchant : !selectedCloudAccount"
          @click="goToCreatePage"
        >
          创建实例
        </el-button>
        <el-button
          type="warning"
          :disabled="!selectedCloudAccount || selectedRegions.length === 0"
          @click="openTunnelDeployDialog"
        >
          一键部署隧道
        </el-button>
      </div>
    </el-card>

    <el-card v-if="(accountType === 'merchant' ? selectedMerchant : selectedCloudAccount) && selectedRegions.length > 0" class="table-card">
      <template #header>
        <div class="card-header">
          <span>实例列表</span>
          <div class="operations">
            <el-button type="primary" size="small" @click="fetchInstanceList">
              刷新
            </el-button>
            <el-button
              type="success"
              size="small"
              :disabled="selectedInstances.length === 0"
              @click="openCreateSecondaryNicDialog"
            >
              创建并绑定辅助网卡
            </el-button>
          </div>
        </div>
      </template>
      <el-table
        v-loading="loading"
        :data="instanceList"
        style="width: 100%"
        border
        @selection-change="handleSelectionChange"
      >
        <el-table-column type="selection" width="55" />
        <el-table-column prop="InstanceId" label="实例ID" min-width="180" />
        <el-table-column prop="InstanceName" label="实例名称" min-width="120" />
        <el-table-column label="区域" min-width="120">
          <template #default="scope">
            {{ scope.row.RegionId || "-" }}
          </template>
        </el-table-column>
        <el-table-column label="公网IP" min-width="150">
          <template #default="scope">
            <span v-if="scope.row.EipAddress?.IpAddress">
              {{ scope.row.EipAddress.IpAddress }}
            </span>
            <span v-else>-</span>
          </template>
        </el-table-column>
        <el-table-column label="配置" min-width="150">
          <template #default="scope">
            {{ scope.row.Cpu }} 核 / {{ scope.row.Memory ? Math.floor(Number(scope.row.Memory) / 1024) : 0 }} GB
          </template>
        </el-table-column>
        <el-table-column prop="Status" label="状态" min-width="100">
          <template #default="scope">
            <el-tag :type="getStatusType(scope.row.Status)">
              {{ getStatusText(scope.row.Status) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="商户" min-width="120">
          <template #default="scope">
            <el-tag v-if="bindingMap[scope.row.InstanceId]" type="success" size="small">
              {{ bindingMap[scope.row.InstanceId].merchant_name }}
            </el-tag>
            <span v-else class="text-gray-400">-</span>
          </template>
        </el-table-column>
        <el-table-column prop="OSName" label="操作系统" min-width="150" />
        <el-table-column prop="InstanceType" label="实例规格" min-width="150" />
        <el-table-column prop="CreationTime" label="创建时间" min-width="180" />
        <el-table-column label="操作" fixed="right" min-width="130">
          <template #default="scope">
            <div class="action-buttons">
              <el-button
                type="primary"
                text
                size="small"
                @click="showInstanceDetail(scope.row)"
              >
                详情
              </el-button>
              <el-dropdown trigger="click">
                <el-button type="primary" text size="small">
                  更多<el-icon class="el-icon--right">
                    <ArrowDown />
                  </el-icon>
                </el-button>
                <template #dropdown>
                  <el-dropdown-menu>
                    <el-dropdown-item v-if="scope.row.Status === 'Stopped'" @click="handleOperate(scope.row, 'start')">
                      <el-icon><VideoPlay /></el-icon> 启动
                    </el-dropdown-item>
                    <el-dropdown-item v-if="scope.row.Status === 'Running'" @click="handleOperate(scope.row, 'stop')">
                      <el-icon><VideoPause /></el-icon> 停止
                    </el-dropdown-item>
                    <el-dropdown-item v-if="scope.row.Status === 'Running'" @click="handleOperate(scope.row, 'restart')">
                      <el-icon><RefreshRight /></el-icon> 重启
                    </el-dropdown-item>
                    <el-dropdown-item @click="openModifyDialog(scope.row)">
                      <el-icon><Edit /></el-icon> 修改属性
                    </el-dropdown-item>
                    <el-dropdown-item @click="handleToPostPaid(scope.row)">
                      <el-icon><Edit /></el-icon> 转为按量付费
                    </el-dropdown-item>
                    <el-dropdown-item @click="handleCreateServer(scope.row)">
                      <el-icon><Location /></el-icon> 创建服务器
                    </el-dropdown-item>
                    <el-dropdown-item divided @click="openBindMerchantDialog(scope.row)">
                      <el-icon><User /></el-icon> {{ bindingMap[scope.row.InstanceId] ? '更换商户' : '绑定商户' }}
                    </el-dropdown-item>
                    <el-dropdown-item v-if="bindingMap[scope.row.InstanceId]" @click="handleUnbindMerchant(scope.row)">
                      <el-icon><Delete /></el-icon> 解绑商户
                    </el-dropdown-item>
                    <el-dropdown-item divided @click="confirmDelete(scope.row)">
                      <el-icon><Delete /></el-icon> 释放
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

    <!-- 实例详情对话框 -->
    <el-dialog
      v-model="detailDialogVisible"
      :title="`实例详情 - ${detailInstance?.InstanceName || ''}`"
      width="750px"
      destroy-on-close
    >
      <div v-if="detailInstance" class="instance-detail">
        <el-descriptions :column="2" border>
          <el-descriptions-item label="实例ID" :span="2">
            {{ detailInstance.InstanceId }}
          </el-descriptions-item>
          <el-descriptions-item label="实例名称">
            {{ detailInstance.InstanceName }}
          </el-descriptions-item>
          <el-descriptions-item label="状态">
            <el-tag :type="getStatusType(detailInstance.Status)">
              {{ getStatusText(detailInstance.Status) }}
            </el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="CPU">
            {{ detailInstance.Cpu }} 核
          </el-descriptions-item>
          <el-descriptions-item label="内存">
            {{ detailInstance.Memory ? Math.floor(Number(detailInstance.Memory) / 1024) : 0 }} GB
          </el-descriptions-item>
          <el-descriptions-item label="操作系统" :span="2">
            {{ detailInstance.OSName }}
          </el-descriptions-item>
          <el-descriptions-item label="创建时间">
            {{ detailInstance.CreationTime }}
          </el-descriptions-item>
          <el-descriptions-item label="到期时间">
            {{ detailInstance.ExpiredTime || '-' }}
          </el-descriptions-item>
          <el-descriptions-item label="最近启动时间">
            {{ detailInstance.StartTime || '-' }}
          </el-descriptions-item>
          <el-descriptions-item label="实例类型">
            {{ detailInstance.InstanceType }}
          </el-descriptions-item>
          <el-descriptions-item label="付费类型">
            {{ detailInstance.InstanceChargeType === 'PrePaid' ? '包年包月' : detailInstance.InstanceChargeType === 'PostPaid' ? '按量付费' : detailInstance.InstanceChargeType }}
          </el-descriptions-item>
          <el-descriptions-item label="镜像ID">
            {{ detailInstance.ImageId }}
          </el-descriptions-item>
          <el-descriptions-item label="网络类型">
            {{ detailInstance.InstanceNetworkType }}
          </el-descriptions-item>
        </el-descriptions>

        <!-- 安全组信息 -->
        <div class="detail-section">
          <h3 class="detail-title">
            安全组信息
          </h3>
          <template v-if="detailInstance.SecurityGroupIds && detailInstance.SecurityGroupIds.SecurityGroupId && detailInstance.SecurityGroupIds.SecurityGroupId.length > 0">
            <el-table :data="detailInstance.SecurityGroupIds.SecurityGroupId.map(id => ({ id }))" border style="width: 100%">
              <el-table-column prop="id" label="安全组ID" />
            </el-table>
          </template>
          <el-empty v-else description="暂无安全组信息" :image-size="60" />
        </div>

        <!-- 网络信息 -->
        <div class="detail-section">
          <h3 class="detail-title">
            网络信息
          </h3>

          <!-- EIP信息 -->
          <template v-if="detailInstance.EipAddress?.IpAddress">
            <el-descriptions :column="2" border>
              <el-descriptions-item label="公网IP" :span="2">
                {{ detailInstance.EipAddress.IpAddress }}
              </el-descriptions-item>
              <el-descriptions-item label="带宽">
                {{ detailInstance.EipAddress.Bandwidth ? Number(detailInstance.EipAddress.Bandwidth) : 0 }} Mbps
              </el-descriptions-item>
              <el-descriptions-item label="计费方式">
                {{ getInternetChargeType(detailInstance.EipAddress.InternetChargeType) }}
              </el-descriptions-item>
            </el-descriptions>
          </template>
          <el-empty v-else description="暂无公网IP" :image-size="60" />

          <!-- VPC信息 -->
          <h4 class="detail-subtitle">
            VPC信息
          </h4>
          <template v-if="detailInstance.VpcAttributes?.VpcId">
            <el-descriptions :column="2" border>
              <el-descriptions-item label="VPC ID">
                {{ detailInstance.VpcAttributes.VpcId }}
              </el-descriptions-item>
              <el-descriptions-item label="交换机ID">
                {{ detailInstance.VpcAttributes.VSwitchId }}
              </el-descriptions-item>
              <el-descriptions-item label="NAT IP" :span="2">
                {{ detailInstance.VpcAttributes.NatIpAddress || '-' }}
              </el-descriptions-item>
              <el-descriptions-item label="内网IP" :span="2" v-if="detailInstance.VpcAttributes.PrivateIpAddress?.IpAddress?.length">
                {{ detailInstance.VpcAttributes.PrivateIpAddress.IpAddress.join(', ') || '-' }}
              </el-descriptions-item>
            </el-descriptions>
          </template>
          <el-empty v-else description="暂无VPC信息" :image-size="60" />

          <!-- 网卡信息 -->
          <h4 class="detail-subtitle">
            网卡信息
          </h4>
          <template v-if="detailInstance.NetworkInterfaces?.NetworkInterface?.length">
            <el-table :data="detailInstance.NetworkInterfaces.NetworkInterface" style="width: 100%" border>
              <el-table-column prop="NetworkInterfaceId" label="网卡ID" width="220" />
              <el-table-column prop="Type" label="类型" width="100">
                <template #default="scope">
                  <el-tag :type="scope.row.Type === 'Primary' ? 'success' : 'info'">
                    {{ scope.row.Type === 'Primary' ? '主网卡' : '辅助网卡' }}
                  </el-tag>
                </template>
              </el-table-column>
              <el-table-column prop="MacAddress" label="MAC地址" width="150" />
              <el-table-column prop="PrimaryIpAddress" label="主私有IP" width="150" />
              <el-table-column label="私有IP详情">
                <template #default="scope">
                  <template v-if="scope.row.PrivateIpSets?.PrivateIpSet?.length">
                    <div v-for="ip in scope.row.PrivateIpSets.PrivateIpSet" :key="ip.PrivateIpAddress" style="margin-bottom: 8px;">
                      <div>
                        <span>私有IP: {{ ip.PrivateIpAddress }}</span>
                        <el-tag v-if="ip.Primary" size="small" type="success" style="margin-left: 5px">主IP</el-tag>
                      </div>
                      <div v-if="ip.AssociatedPublicIp?.PublicIpAddress" style="margin-top: 4px;">
                        <span style="color: #67c23a;">公网IP: {{ ip.AssociatedPublicIp.PublicIpAddress }}</span>
                        <el-tag size="small" type="warning" style="margin-left: 5px">已绑定EIP</el-tag>
                      </div>
                    </div>
                  </template>
                  <span v-else>-</span>
                </template>
              </el-table-column>
            </el-table>
          </template>
          <el-empty v-else description="暂无网卡信息" :image-size="60" />
        </div>
      </div>
      <el-empty v-else description="未找到实例详情" />
    </el-dialog>

    <!-- 修改实例属性对话框 -->
    <el-dialog
      v-model="modifyDialogVisible"
      :title="`修改实例属性 - ${currentEditInstance?.InstanceName || ''}`"
      width="550px"
      destroy-on-close
    >
      <el-form
        ref="modifyFormRef"
        :model="modifyInstanceForm"
        :rules="modifyFormRules"
        label-width="100px"
      >
        <el-form-item label="实例名称" prop="instance_name">
          <el-input v-model="modifyInstanceForm.instance_name" placeholder="请输入实例名称" />
        </el-form-item>
        <el-form-item label="描述" prop="description">
          <el-input
            v-model="modifyInstanceForm.description"
            type="textarea"
            :rows="2"
            placeholder="请输入实例描述"
          />
        </el-form-item>
        <el-form-item label="密码" prop="password">
          <el-input
            v-model="modifyInstanceForm.password"
            type="password"
            placeholder="留空则不修改密码，修改需要包含大小写字母、数字和特殊字符"
            show-password
          />
        </el-form-item>
        <el-form-item label="安全组" prop="security_group_id">
          <el-select
            v-model="modifyInstanceForm.security_group_id"
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

    <!-- 绑定商户对话框 -->
    <el-dialog
      v-model="bindMerchantDialogVisible"
      title="绑定商户"
      width="450px"
      destroy-on-close
    >
      <el-form label-width="80px">
        <el-form-item label="实例ID">
          <span>{{ bindMerchantForm.instance_id }}</span>
        </el-form-item>
        <el-form-item label="商户">
          <el-select
            v-model="bindMerchantForm.merchant_id"
            placeholder="请选择商户"
            filterable
            clearable
            style="width: 100%"
          >
            <el-option
              v-for="m in allMerchantList"
              :key="m.id"
              :label="m.name"
              :value="m.id"
            >
              <div class="custom-option">
                <span>{{ m.name }}</span>
                <small v-if="m.status === 1" class="status active">正常</small>
                <small v-else class="status inactive">禁用</small>
              </div>
            </el-option>
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="bindMerchantDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="bindMerchantLoading" @click="handleBindMerchant">
          确定
        </el-button>
      </template>
    </el-dialog>

    <!-- 一键部署隧道服务器对话框 -->
    <el-dialog
      v-model="tunnelDeployDialogVisible"
      title="一键部署隧道服务器"
      width="650px"
      destroy-on-close
      :close-on-click-modal="false"
      :close-on-press-escape="false"
    >
      <!-- 表单 -->
      <el-form v-if="tunnelDeployFormVisible" label-width="120px">
        <el-form-item label="服务器名称">
          <el-input v-model="tunnelDeployForm.server_name" placeholder="如: gost-hk" />
        </el-form-item>
        <el-form-item label="服务器数量">
          <el-input-number v-model="tunnelDeployForm.server_count" :min="1" :max="10" />
        </el-form-item>
        <el-form-item label="实例规格">
          <el-input v-model="tunnelDeployForm.instance_type" placeholder="ecs.t5.large" />
        </el-form-item>
        <el-form-item label="EIP 带宽(Mbps)">
          <el-input v-model="tunnelDeployForm.bandwidth" placeholder="10" />
        </el-form-item>
        <el-form-item label="每台 EIP 数量">
          <el-input-number v-model="tunnelDeployForm.eip_count" :min="1" :max="10" />
        </el-form-item>
        <el-form-item label="地域">
          <span>{{ selectedRegions[0] }}</span>
        </el-form-item>
        <el-form-item label="计费方式">
          <span>按量付费</span>
        </el-form-item>
      </el-form>

      <!-- 进度输出 -->
      <div v-if="!tunnelDeployFormVisible" v-loading="tunnelDeployLoading" class="output-container">
        <pre>{{ tunnelDeployOutput }}</pre>
      </div>

      <!-- 部署结果：SSH 私钥下载 -->
      <div v-if="tunnelDeployResults.length > 0" style="margin-top: 12px;">
        <el-alert type="warning" :closable="false" style="margin-bottom: 8px;">
          <template #title>请立即下载 SSH 私钥，关闭后无法再次获取</template>
        </el-alert>
        <div v-for="r in tunnelDeployResults" :key="r.instance_id" style="margin-bottom: 8px;">
          <el-button type="primary" size="small" @click="downloadPrivateKey(r.server_name, r.private_key)">
            下载 {{ r.server_name }}.pem
          </el-button>
          <span style="margin-left: 8px; color: #999;">IP: {{ r.public_ip }}, EIPs: {{ r.eips?.join(', ') }}</span>
        </div>
      </div>

      <template #footer>
        <el-button v-if="tunnelDeployFormVisible" @click="tunnelDeployDialogVisible = false">取消</el-button>
        <el-button v-if="tunnelDeployFormVisible" type="primary" @click="handleTunnelDeploy">开始部署</el-button>
        <el-button v-if="!tunnelDeployFormVisible" :disabled="tunnelDeployLoading" @click="tunnelDeployDialogVisible = false">关闭</el-button>
      </template>
    </el-dialog>

    <!-- 创建辅助网卡对话框 -->
    <el-dialog
      v-model="createNicDialogVisible"
      title="创建辅助网卡"
      width="550px"
      destroy-on-close
      :close-on-click-modal="false"
      :close-on-press-escape="false"
    >
      <div v-loading="createNicLoading" class="output-container">
        <pre>{{ createNicOutput }}</pre>
      </div>
      <template #footer>
        <el-button :disabled="createNicLoading" @click="createNicDialogVisible = false">
          关闭
        </el-button>
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

.output-container {
  margin-top: 20px;
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

.form-tip {
  margin-top: 5px;
  color: #909399;
  font-size: 12px;
}

.operation-status {
  margin-top: 15px;
  padding: 12px 15px;
  border-radius: 4px;
  display: flex;
  align-items: flex-start;
}

.operation-status.success {
  background-color: #f0f9eb;
  border-left: 4px solid #67c23a;
}

.operation-status .status-icon {
  font-size: 18px;
  margin-right: 10px;
  margin-top: 2px;
  color: #67c23a;
}

.operation-status .status-content {
  display: flex;
  flex-direction: column;
}

.operation-status .status-title {
  font-size: 14px;
  font-weight: 600;
  margin-bottom: 5px;
}

.operation-status .status-message {
  font-size: 13px;
  color: #606266;
  padding: 5px 8px;
  background-color: rgba(0, 0, 0, 0.03);
  border-radius: 3px;
  margin-top: 3px;
}

.status-debug {
  font-size: 12px;
  color: #909399;
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

.instance-detail {
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

.detail-subtitle {
  margin: 16px 0 8px;
  font-size: 14px;
  font-weight: 500;
  color: #606266;
}

.action-buttons {
  display: flex;
  align-items: center;
  gap: 8px;
}
</style>
