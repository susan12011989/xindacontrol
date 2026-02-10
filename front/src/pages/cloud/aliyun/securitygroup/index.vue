<script setup lang="ts">
import type { Region } from "@/pages/cloud/aliyun/apis/type"
import type { DescribeSecurityGroupAttributeResponse, SecurityGroupWrap } from "@/pages/cloud/aliyun/securitygroup/apis/type"
import type { Merchant, MerchantRegions } from "@/pages/dashboard/apis/type"
import type { CloudAccountOption } from "@@/apis/cloud_account/type"
import { regionListApi } from "@/pages/cloud/aliyun/apis"
import { authorizeSecurityGroup, authorizeSecurityGroupBatch, deleteSecurityGroup, describeSecurityGroupAttribute, getSecurityGroupList, revokeSecurityGroup } from "@/pages/cloud/aliyun/securitygroup/apis"
import { merchantQueryApi } from "@/pages/dashboard/apis"
import { getCloudAccountOptions } from "@@/apis/cloud_account"
import { ArrowDown, Delete, Edit, Key, Location, User } from "@element-plus/icons-vue"
import { ElMessage, ElMessageBox } from "element-plus"
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from "vue"
import { useRoute, useRouter } from "vue-router"

defineOptions({
  name: "CloudSecurityGroup"
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
const securityGroupList = ref<SecurityGroupWrap[]>([])
const selectedMerchant = ref<number>()
const selectedRegions = ref<string[]>([])

// 响应式检测屏幕尺寸
const isSmallScreen = computed(() => {
  return window.innerWidth < 768
})

// 监听窗口大小变化
function handleResize() {
  // 使用无操作函数调用来避免ESLint错误
  // 这只是为了触发计算属性的依赖收集
  void isSmallScreen.value
}

// 安全组详情
const detailDialogVisible = ref(false)
const detailLoading = ref(false)
const detailSecurityGroup = ref<SecurityGroupWrap | null>(null)
const securityGroupDetail = ref<DescribeSecurityGroupAttributeResponse | null>(null)

// 批量添加规则对话框
const batchRulesDialogVisible = ref(false)
const batchRuleForm = reactive({
  IpProtocol: "tcp",
  PortRange: "",
  SourceCidrIp: "",
  Policy: "Accept"
})
const batchRuleFormRef = ref()
const batchAddRuleLoading = ref(false)
const batchRuleFormRules = {
  PortRange: [
    { required: true, message: "请输入端口范围", trigger: "blur" }
  ],
  SourceCidrIp: [
    { required: true, message: "请输入源IP地址", trigger: "blur" }
  ]
}

// 获取路由实例和查询参数
const route = useRoute()
const router = useRouter()

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
  securityGroupList.value = []

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
  securityGroupList.value = []

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

// 获取安全组列表
async function fetchSecurityGroupList() {
  const hasAccount = accountType.value === "merchant" ? selectedMerchant.value : selectedCloudAccount.value
  if (!hasAccount || !selectedRegions.value || selectedRegions.value.length === 0) {
    console.log("账号或区域未选择，不获取安全组列表")
    return
  }

  console.log("开始获取安全组列表, 类型:", accountType.value, "账号:", hasAccount, "区域:", selectedRegions.value)
  loading.value = true
  try {
    const params: any = { region_id: selectedRegions.value }
    if (accountType.value === "merchant") {
      params.merchant_id = selectedMerchant.value
    } else {
      params.cloud_account_id = selectedCloudAccount.value
    }

    const res = await getSecurityGroupList(params)

    console.log("获取安全组列表成功:", res.data.list)
    // 兼容返回空数组的情况
    if (!res.data.list || !Array.isArray(res.data.list)) {
      console.log("安全组列表为空或格式不正确，设置为空数组")
      securityGroupList.value = []
    } else {
      securityGroupList.value = res.data.list
    }

    // 如果列表为空，显示提示
    if (securityGroupList.value.length === 0) {
      ElMessage.info("当前区域暂无安全组")
    }
  } catch (error) {
    console.error("获取安全组列表失败", error)
    ElMessage.error("获取安全组列表失败")
    securityGroupList.value = []
  } finally {
    loading.value = false
  }
}

// 商户变化处理
function handleMerchantChange(value: number | undefined) {
  console.log("商户变更为:", value)
  selectedRegions.value = []
  securityGroupList.value = []

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
  securityGroupList.value = []
}

// 确认删除安全组
function confirmDelete(securityGroup: SecurityGroupWrap) {
  ElMessageBox.confirm(
    `确定要删除安全组 ${securityGroup.SecurityGroup.SecurityGroupName || securityGroup.SecurityGroup.SecurityGroupId} 吗？`,
    "删除确认",
    {
      confirmButtonText: "确定",
      cancelButtonText: "取消",
      type: "warning"
    }
  )
    .then(() => {
      handleDelete(securityGroup)
    })
    .catch(() => {
      // 用户取消删除操作
    })
}

// 删除安全组
async function handleDelete(securityGroup: SecurityGroupWrap) {
  const hasAccount = accountType.value === "merchant" ? selectedMerchant.value : selectedCloudAccount.value
  if (!hasAccount) return

  try {
    const params: any = {
      region_id: securityGroup.RegionId,
      security_group_id: securityGroup.SecurityGroup.SecurityGroupId
    }

    if (accountType.value === "merchant") {
      params.merchant_id = selectedMerchant.value
    } else {
      params.cloud_account_id = selectedCloudAccount.value
    }

    await deleteSecurityGroup(params)

    ElMessage.success("删除成功")
    // 延迟刷新列表，等待操作完成
    setTimeout(() => {
      fetchSecurityGroupList()
    }, 2000)
  } catch (error) {
    console.error("删除安全组失败", error)
    ElMessage.error("删除安全组失败")
  }
}

// 安全组规则管理对话框
const rulesDialogVisible = ref(false)
const currentRulesSecurityGroup = ref<SecurityGroupWrap | null>(null)
const ruleForm = reactive({
  IpProtocol: "tcp",
  PortRange: "",
  SourceCidrIp: "",
  Policy: "Accept"
})
const ruleFormRef = ref()
const ruleFormRules = {
  PortRange: [
    { required: true, message: "请输入端口范围", trigger: "blur" }
  ],
  SourceCidrIp: [
    { required: true, message: "请输入源IP地址", trigger: "blur" }
  ]
}
const addRuleLoading = ref(false)
const removeRuleLoading = ref(false)
const ruleDetailVisible = ref(false)
const selectedRule = ref<any>(null)

// 打开规则管理对话框
function openRulesDialog(securityGroup: SecurityGroupWrap) {
  currentRulesSecurityGroup.value = securityGroup
  showSecurityGroupDetail(securityGroup)
  rulesDialogVisible.value = true
  resetRuleForm()
}

// 重置规则表单
function resetRuleForm() {
  ruleForm.IpProtocol = "tcp"
  ruleForm.PortRange = ""
  ruleForm.SourceCidrIp = ""
  ruleForm.Policy = "Accept"
  if (ruleFormRef.value) {
    ruleFormRef.value.resetFields()
  }
}

// 添加安全组规则
async function addSecurityGroupRule(formEl: any) {
  const hasAccount = accountType.value === "merchant" ? selectedMerchant.value : selectedCloudAccount.value
  if (!formEl || !currentRulesSecurityGroup.value || !hasAccount) return

  await formEl.validate(async (valid: boolean) => {
    if (valid) {
      addRuleLoading.value = true
      try {
        if (currentRulesSecurityGroup.value) {
          const params: any = {
            region_id: currentRulesSecurityGroup.value.RegionId,
            security_group_id: currentRulesSecurityGroup.value.SecurityGroup.SecurityGroupId,
            permissions: [{
              IpProtocol: ruleForm.IpProtocol,
              PortRange: ruleForm.PortRange,
              SourceCidrIp: ruleForm.SourceCidrIp,
              Policy: ruleForm.Policy
            }]
          }

          if (accountType.value === "merchant") {
            params.merchant_id = selectedMerchant.value
          } else {
            params.cloud_account_id = selectedCloudAccount.value
          }

          await authorizeSecurityGroup(params)

          ElMessage.success("添加安全组规则成功")
          resetRuleForm()

          // 刷新详情
          if (currentRulesSecurityGroup.value) {
            showSecurityGroupDetail(currentRulesSecurityGroup.value)
          }
        }
      } catch (error) {
        console.error("添加安全组规则失败", error)
        ElMessage.error("添加安全组规则失败")
      } finally {
        addRuleLoading.value = false
      }
    }
  })
}

// 查看规则详情
function viewRuleDetail(rule: any) {
  selectedRule.value = rule
  ruleDetailVisible.value = true
}

// 删除安全组规则
async function removeSecurityGroupRule(rule: any) {
  const hasAccount = accountType.value === "merchant" ? selectedMerchant.value : selectedCloudAccount.value
  if (!currentRulesSecurityGroup.value || !hasAccount) return

  ElMessageBox.confirm(
    `确定要删除该安全组规则吗？`,
    "删除确认",
    {
      confirmButtonText: "确定",
      cancelButtonText: "取消",
      type: "warning"
    }
  )
    .then(async () => {
      removeRuleLoading.value = true
      try {
        if (currentRulesSecurityGroup.value) {
          const params: any = {
            region_id: currentRulesSecurityGroup.value.RegionId,
            security_group_id: currentRulesSecurityGroup.value.SecurityGroup.SecurityGroupId,
            permissions: [{
              IpProtocol: rule.IpProtocol,
              PortRange: rule.PortRange,
              SourceCidrIp: rule.SourceCidrIp,
              Policy: rule.Policy
            }]
          }

          if (accountType.value === "merchant") {
            params.merchant_id = selectedMerchant.value
          } else {
            params.cloud_account_id = selectedCloudAccount.value
          }

          await revokeSecurityGroup(params)

          ElMessage.success("删除安全组规则成功")

          // 刷新详情
          if (currentRulesSecurityGroup.value) {
            showSecurityGroupDetail(currentRulesSecurityGroup.value)
          }

          // 关闭规则详情对话框
          if (ruleDetailVisible.value) {
            ruleDetailVisible.value = false
          }
        }
      } catch (error) {
        console.error("删除安全组规则失败", error)
        ElMessage.error("删除安全组规则失败")
      } finally {
        removeRuleLoading.value = false
      }
    })
    .catch(() => {
      // 用户取消操作
    })
}

// 获取协议文本
function getProtocolText(protocol: string) {
  const protocolMap: Record<string, string> = {
    tcp: "TCP",
    udp: "UDP",
    icmp: "ICMP",
    gre: "GRE",
    all: "全部"
  }
  return protocolMap[protocol.toLowerCase()] || protocol
}

// 跳转到创建安全组页面
function goToCreatePage() {
  if (accountType.value === "merchant") {
    if (!selectedMerchant.value) {
      ElMessage.warning("请先选择商户")
      return
    }
    router.push({
      path: "/cloud/aliyun/securitygroup/create",
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
      path: "/cloud/aliyun/securitygroup/create",
      query: {
        cloud_account_id: selectedCloudAccount.value.toString()
      }
    })
  }
}

// 显示安全组详情
async function showSecurityGroupDetail(securityGroup: SecurityGroupWrap) {
  const hasAccount = accountType.value === "merchant" ? selectedMerchant.value : selectedCloudAccount.value
  if (!hasAccount) return

  detailSecurityGroup.value = securityGroup
  detailDialogVisible.value = true
  detailLoading.value = true
  securityGroupDetail.value = null

  try {
    const params: any = {
      region_id: securityGroup.RegionId,
      security_group_id: securityGroup.SecurityGroup.SecurityGroupId
    }

    if (accountType.value === "merchant") {
      params.merchant_id = selectedMerchant.value
    } else {
      params.cloud_account_id = selectedCloudAccount.value
    }

    const res = await describeSecurityGroupAttribute(params)

    securityGroupDetail.value = res.data
  } catch (error) {
    console.error("获取安全组详情失败", error)
    ElMessage.error("获取安全组详情失败")
  } finally {
    detailLoading.value = false
  }
}

// 打开批量添加规则对话框
function openBatchRulesDialog() {
  batchRulesDialogVisible.value = true
  resetBatchRuleForm()
}

// 重置批量规则表单
function resetBatchRuleForm() {
  batchRuleForm.IpProtocol = "tcp"
  batchRuleForm.PortRange = ""
  batchRuleForm.SourceCidrIp = ""
  batchRuleForm.Policy = "Accept"
  if (batchRuleFormRef.value) {
    batchRuleFormRef.value.resetFields()
  }
}

// 批量添加安全组规则
async function addBatchSecurityGroupRule(formEl: any) {
  const hasAccount = accountType.value === "merchant" ? selectedMerchant.value : selectedCloudAccount.value
  if (!formEl || !hasAccount || !selectedRegions.value || selectedRegions.value.length === 0) return

  await formEl.validate(async (valid: boolean) => {
    if (valid) {
      batchAddRuleLoading.value = true
      try {
        const params: any = {
          region_ids: selectedRegions.value,
          permissions: [{
            IpProtocol: batchRuleForm.IpProtocol,
            PortRange: batchRuleForm.PortRange,
            SourceCidrIp: batchRuleForm.SourceCidrIp,
            Policy: batchRuleForm.Policy
          }]
        }

        if (accountType.value === "merchant") {
          params.merchant_id = Number(selectedMerchant.value)
        } else {
          params.cloud_account_id = selectedCloudAccount.value
        }

        await authorizeSecurityGroupBatch(params)

        ElMessage.success("批量添加安全组规则成功")
        resetBatchRuleForm()
        batchRulesDialogVisible.value = false

        // 刷新列表
        setTimeout(() => {
          fetchSecurityGroupList()
        }, 2000)
      } catch (error) {
        console.error("批量添加安全组规则失败", error)
        ElMessage.error("批量添加安全组规则失败")
      } finally {
        batchAddRuleLoading.value = false
      }
    }
  })
}

// 页面加载时获取商户和区域列表
onMounted(() => {
  // 添加窗口大小变化监听
  window.addEventListener("resize", handleResize)

  loadSelectionFromStorage()

  const fetchInitialData = () => {
    // 处理从其他页面跳转的情况
    const queryMerchantId = route.query.merchant_id
    const querySecurityGroupId = route.query.security_group_id
    const queryRegionId = route.query.region_id

    // 如果URL中有指定的商户ID和安全组ID，优先使用
    if (queryMerchantId && typeof queryMerchantId === "string") {
      const merchantId = Number.parseInt(queryMerchantId)
      if (!Number.isNaN(merchantId)) {
        accountType.value = "merchant"
        selectedMerchant.value = merchantId

        // 获取区域列表后查询安全组列表
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

            // 获取安全组列表
            fetchSecurityGroupList().then(() => {
              // 如果有指定的安全组ID，找到并显示详情
              if (querySecurityGroupId && typeof querySecurityGroupId === "string") {
                const foundSecurityGroup = securityGroupList.value.find(
                  sg => sg.SecurityGroup.SecurityGroupId === querySecurityGroupId
                )

                if (foundSecurityGroup) {
                  // 找到安全组后显示详情
                  showSecurityGroupDetail(foundSecurityGroup)
                } else {
                  ElMessage.warning(`未找到ID为 ${querySecurityGroupId} 的安全组，可能不在当前区域`)
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

// 组件卸载时移除事件监听
onBeforeUnmount(() => {
  window.removeEventListener("resize", handleResize)
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
        <div class="action-buttons-container">
          <el-button
            type="primary"
            :disabled="(accountType === 'merchant' ? !selectedMerchant : !selectedCloudAccount) || !selectedRegions.length"
            @click="fetchSecurityGroupList"
          >
            查询
          </el-button>
          <el-button
            type="success"
            :disabled="(accountType === 'merchant' ? !selectedMerchant : !selectedCloudAccount) || !selectedRegions.length"
            @click="goToCreatePage"
          >
            创建安全组
          </el-button>
          <el-button
            type="primary"
            :disabled="(accountType === 'merchant' ? !selectedMerchant : !selectedCloudAccount) || !selectedRegions.length"
            @click="openBatchRulesDialog"
          >
            批量添加规则
          </el-button>
        </div>
      </div>
    </el-card>

    <el-card v-if="(accountType === 'merchant' ? selectedMerchant : selectedCloudAccount) && selectedRegions.length > 0" class="table-card">
      <template #header>
        <div class="card-header">
          <span>安全组列表</span>
          <div class="operations">
            <el-button type="primary" size="small" @click="fetchSecurityGroupList">
              刷新
            </el-button>
          </div>
        </div>
      </template>
      <el-table
        v-loading="loading"
        :data="securityGroupList"
        style="width: 100%"
        border
      >
        <el-table-column prop="SecurityGroup.SecurityGroupId" label="安全组ID" min-width="180" />
        <el-table-column prop="SecurityGroup.SecurityGroupName" label="安全组名称" min-width="150" />
        <el-table-column label="区域" min-width="120">
          <template #default="scope">
            {{ scope.row.RegionId || "-" }}
          </template>
        </el-table-column>
        <el-table-column prop="SecurityGroup.VpcId" label="VPC ID" min-width="180" />
        <el-table-column prop="SecurityGroup.EcsCount" label="实例数量" min-width="100" />
        <el-table-column prop="SecurityGroup.Description" label="描述" min-width="150" show-overflow-tooltip />
        <el-table-column prop="SecurityGroup.CreationTime" label="创建时间" min-width="180" />
        <el-table-column label="操作" fixed="right" min-width="120">
          <template #default="scope">
            <div class="action-buttons">
              <el-button
                type="primary"
                text
                size="small"
                @click="showSecurityGroupDetail(scope.row)"
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
                    <el-dropdown-item @click="openRulesDialog(scope.row)">
                      <el-icon><Edit /></el-icon> 规则管理
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

    <!-- 安全组详情对话框 -->
    <el-dialog
      v-model="detailDialogVisible"
      :title="`安全组详情 - ${detailSecurityGroup?.SecurityGroup.SecurityGroupName || detailSecurityGroup?.SecurityGroup.SecurityGroupId || ''}`"
      width="750px"
      :fullscreen="isSmallScreen"
      destroy-on-close
    >
      <div v-loading="detailLoading">
        <div v-if="securityGroupDetail" class="security-group-detail">
          <el-descriptions :column="2" border>
            <el-descriptions-item label="安全组ID" :span="2">
              {{ securityGroupDetail.SecurityGroupId }}
            </el-descriptions-item>
            <el-descriptions-item label="安全组名称">
              {{ securityGroupDetail.SecurityGroupName }}
            </el-descriptions-item>
            <el-descriptions-item label="描述">
              {{ securityGroupDetail.Description || '-' }}
            </el-descriptions-item>
            <el-descriptions-item label="VPC ID">
              {{ securityGroupDetail.VpcId || '-' }}
            </el-descriptions-item>
            <el-descriptions-item label="区域">
              {{ securityGroupDetail.RegionId || '-' }}
            </el-descriptions-item>
          </el-descriptions>

          <!-- 安全组规则信息 -->
          <div class="detail-section">
            <h3 class="detail-title">
              安全组规则
            </h3>
            <div v-if="securityGroupDetail.Permissions && securityGroupDetail.Permissions.Permission && securityGroupDetail.Permissions.Permission.length">
              <el-table :data="securityGroupDetail.Permissions.Permission" border style="width: 100%">
                <el-table-column prop="Direction" label="方向" width="80">
                  <template #default="scope">
                    {{ scope.row.Direction === 'ingress' ? '入方向' : '出方向' }}
                  </template>
                </el-table-column>
                <el-table-column prop="IpProtocol" label="协议类型" width="100">
                  <template #default="scope">
                    {{ getProtocolText(scope.row.IpProtocol) }}
                  </template>
                </el-table-column>
                <el-table-column prop="PortRange" label="端口范围" width="120" />
                <el-table-column prop="SourceCidrIp" label="源IP地址" min-width="150" />
                <el-table-column prop="Policy" label="授权策略" width="100">
                  <template #default="scope">
                    {{ scope.row.Policy === 'Accept' ? '允许' : '拒绝' }}
                  </template>
                </el-table-column>
                <el-table-column prop="CreateTime" label="创建时间" width="180" />
              </el-table>
            </div>
            <el-empty v-else description="暂无安全组规则" :image-size="60" />
          </div>
        </div>
        <el-empty v-else description="未找到安全组详情" />
      </div>
    </el-dialog>

    <!-- 安全组规则管理对话框 -->
    <el-dialog
      v-model="rulesDialogVisible"
      :title="`安全组规则管理 - ${currentRulesSecurityGroup?.SecurityGroup.SecurityGroupName || currentRulesSecurityGroup?.SecurityGroup.SecurityGroupId || ''}`"
      width="900px"
      :fullscreen="isSmallScreen"
      destroy-on-close
    >
      <div class="rules-management">
        <!-- 添加规则表单 -->
        <div class="add-rule-section">
          <h3>添加安全组规则</h3>
          <el-form
            ref="ruleFormRef"
            :model="ruleForm"
            :rules="ruleFormRules"
            label-width="120px"
            label-position="right"
          >
            <el-row :gutter="20">
              <el-col :span="12">
                <el-form-item label="协议类型">
                  <el-select v-model="ruleForm.IpProtocol" style="width: 100%">
                    <el-option label="TCP协议" value="tcp" />
                    <el-option label="UDP协议" value="udp" />
                    <el-option label="ICMP协议" value="icmp" />
                    <el-option label="GRE协议" value="gre" />
                    <el-option label="全部协议" value="all" />
                  </el-select>
                </el-form-item>
              </el-col>
              <el-col :span="12">
                <el-form-item label="端口范围" prop="PortRange">
                  <el-input v-model="ruleForm.PortRange" placeholder="例如: 22/22, 1/65535" />
                  <div class="form-tip">
                    端口范围格式: 起始端口/结束端口
                  </div>
                </el-form-item>
              </el-col>
            </el-row>
            <el-row :gutter="20">
              <el-col :span="12">
                <el-form-item label="授权策略">
                  <el-select v-model="ruleForm.Policy" style="width: 100%">
                    <el-option label="允许" value="Accept" />
                    <el-option label="拒绝" value="Drop" />
                  </el-select>
                </el-form-item>
              </el-col>
              <el-col :span="12">
                <el-form-item label="源IP地址" prop="SourceCidrIp">
                  <el-input v-model="ruleForm.SourceCidrIp" placeholder="例如: 10.0.0.0/8, 0.0.0.0/0" />
                  <div class="form-tip">
                    CIDR格式，0.0.0.0/0表示允许所有IP
                  </div>
                </el-form-item>
              </el-col>
            </el-row>
            <el-form-item>
              <el-button type="primary" :loading="addRuleLoading" @click="addSecurityGroupRule(ruleFormRef)">
                添加规则
              </el-button>
              <el-button @click="resetRuleForm">
                重置
              </el-button>
            </el-form-item>
          </el-form>
        </div>

        <!-- 规则列表 -->
        <div class="rule-list-section">
          <h3>现有安全组规则</h3>
          <div v-if="securityGroupDetail?.Permissions && securityGroupDetail?.Permissions.Permission && securityGroupDetail?.Permissions.Permission.length">
            <el-table :data="securityGroupDetail.Permissions.Permission" border style="width: 100%">
              <el-table-column prop="Direction" label="方向" width="80">
                <template #default="scope">
                  {{ scope.row.Direction === 'ingress' ? '入方向' : '出方向' }}
                </template>
              </el-table-column>
              <el-table-column prop="IpProtocol" label="协议类型" width="100">
                <template #default="scope">
                  {{ getProtocolText(scope.row.IpProtocol) }}
                </template>
              </el-table-column>
              <el-table-column prop="PortRange" label="端口范围" width="120" />
              <el-table-column prop="SourceCidrIp" label="源IP地址" min-width="150" />
              <el-table-column prop="Policy" label="授权策略" width="100">
                <template #default="scope">
                  {{ scope.row.Policy === 'Accept' ? '允许' : '拒绝' }}
                </template>
              </el-table-column>
              <el-table-column label="操作" width="150">
                <template #default="scope">
                  <el-button type="primary" text size="small" @click="viewRuleDetail(scope.row)">
                    详情
                  </el-button>
                  <el-button type="danger" text size="small" @click="removeSecurityGroupRule(scope.row)">
                    删除
                  </el-button>
                </template>
              </el-table-column>
            </el-table>
          </div>
          <el-empty v-else description="暂无安全组规则" />
        </div>
      </div>
    </el-dialog>

    <!-- 规则详情对话框 -->
    <el-dialog
      v-model="ruleDetailVisible"
      title="安全组规则详情"
      width="600px"
      :fullscreen="isSmallScreen"
      destroy-on-close
    >
      <div v-if="selectedRule">
        <el-descriptions :column="1" border>
          <el-descriptions-item label="方向">
            {{ selectedRule.Direction === 'ingress' ? '入方向' : '出方向' }}
          </el-descriptions-item>
          <el-descriptions-item label="协议类型">
            {{ getProtocolText(selectedRule.IpProtocol) }}
          </el-descriptions-item>
          <el-descriptions-item label="端口范围">
            {{ selectedRule.PortRange }}
          </el-descriptions-item>
          <el-descriptions-item label="源IP地址">
            {{ selectedRule.SourceCidrIp }}
          </el-descriptions-item>
          <el-descriptions-item label="授权策略">
            {{ selectedRule.Policy === 'Accept' ? '允许' : '拒绝' }}
          </el-descriptions-item>
          <el-descriptions-item label="创建时间">
            {{ selectedRule.CreateTime }}
          </el-descriptions-item>
        </el-descriptions>
        <div class="dialog-footer" style="margin-top: 20px; text-align: right;">
          <el-button @click="ruleDetailVisible = false">
            关闭
          </el-button>
          <el-button type="danger" :loading="removeRuleLoading" @click="removeSecurityGroupRule(selectedRule)">
            删除规则
          </el-button>
        </div>
      </div>
      <el-empty v-else description="无规则详情" />
    </el-dialog>

    <!-- 批量添加规则对话框 -->
    <el-dialog
      v-model="batchRulesDialogVisible"
      title="批量添加安全组规则"
      width="750px"
      :fullscreen="isSmallScreen"
      destroy-on-close
    >
      <div class="batch-rules-management">
        <p class="batch-tip-text">
          此操作将为选中的所有区域内的所有安全组添加相同的规则
        </p>
        <el-form
          ref="batchRuleFormRef"
          :model="batchRuleForm"
          :rules="batchRuleFormRules"
          :label-width="isSmallScreen ? '90px' : '120px'"
          label-position="right"
          class="batch-rule-form"
        >
          <el-row :gutter="isSmallScreen ? 15 : 30">
            <el-col :xs="24" :sm="24" :md="12" :lg="12" :xl="12">
              <el-form-item label="协议类型">
                <el-select v-model="batchRuleForm.IpProtocol" style="width: 100%">
                  <el-option label="TCP协议" value="tcp" />
                  <el-option label="UDP协议" value="udp" />
                  <el-option label="ICMP协议" value="icmp" />
                  <el-option label="GRE协议" value="gre" />
                  <el-option label="全部协议" value="all" />
                </el-select>
              </el-form-item>
            </el-col>
            <el-col :xs="24" :sm="24" :md="12" :lg="12" :xl="12">
              <el-form-item label="端口范围" prop="PortRange">
                <el-input v-model="batchRuleForm.PortRange" placeholder="例如: 22/22, 1/65535" />
                <div class="form-tip">
                  端口范围格式: 起始端口/结束端口
                </div>
              </el-form-item>
            </el-col>
          </el-row>
          <el-row :gutter="isSmallScreen ? 15 : 30">
            <el-col :xs="24" :sm="24" :md="12" :lg="12" :xl="12">
              <el-form-item label="授权策略">
                <el-select v-model="batchRuleForm.Policy" style="width: 100%">
                  <el-option label="允许" value="Accept" />
                  <el-option label="拒绝" value="Drop" />
                </el-select>
              </el-form-item>
            </el-col>
            <el-col :xs="24" :sm="24" :md="12" :lg="12" :xl="12">
              <el-form-item label="源IP地址" prop="SourceCidrIp">
                <el-input v-model="batchRuleForm.SourceCidrIp" placeholder="例如: 10.0.0.0/8, 0.0.0.0/0" />
                <div class="form-tip">
                  CIDR格式，0.0.0.0/0表示允许所有IP
                </div>
              </el-form-item>
            </el-col>
          </el-row>
          <el-form-item class="warning-section">
            <el-alert
              type="warning"
              :closable="false"
              show-icon
              class="batch-warning-alert"
            >
              <template #default>
                <div class="warning-content">
                  <p class="warning-title">
                    操作提示
                  </p>
                  <p>选中区域: <strong>{{ selectedRegions.join(', ') }}</strong></p>
                  <p>此操作将为所选区域中的所有安全组添加相同的入方向规则，请谨慎操作！</p>
                </div>
              </template>
            </el-alert>
          </el-form-item>
          <el-form-item class="batch-form-footer">
            <el-button type="primary" :loading="batchAddRuleLoading" @click="addBatchSecurityGroupRule(batchRuleFormRef)">
              确认添加
            </el-button>
            <el-button @click="batchRulesDialogVisible = false">
              取消
            </el-button>
          </el-form-item>
        </el-form>
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

.security-group-detail {
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

.rules-management {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.add-rule-section {
  border: 1px solid #ebeef5;
  border-radius: 4px;
  padding: 20px;
  margin-bottom: 20px;
  background-color: #f8f8f8;
}

.add-rule-section h3 {
  margin-top: 0;
  margin-bottom: 20px;
  font-size: 16px;
  font-weight: 500;
  color: #303133;
}

.rule-list-section h3 {
  margin-top: 0;
  margin-bottom: 20px;
  font-size: 16px;
  font-weight: 500;
  color: #303133;
}

.security-group-creation {
  padding: 20px;
}

.header-actions {
  display: flex;
  justify-content: flex-end;
  margin-bottom: 20px;
}

.empty-state {
  display: flex;
  justify-content: center;
  align-items: center;
  height: 200px;
}

.security-group-list {
  display: flex;
  flex-wrap: wrap;
  gap: 16px;
}

.security-group-item {
  width: 100%;
  margin-bottom: 16px;
}

.progress-dialog-content {
  padding: 20px;
}

.operation-output-container {
  margin-bottom: 20px;
}

.operation-output {
  height: 300px;
  overflow-y: auto;
  background-color: #1e1e1e;
  color: #f8f8f8;
  font-family: monospace;
  padding: 10px;
  border-radius: 4px;
  white-space: pre-wrap;
  word-break: break-all;
}

.completion-message {
  margin-top: 20px;
}

.tip-text {
  margin-bottom: 16px;
  color: #606266;
  font-size: 14px;
}

.batch-rules-management {
  padding: 20px;
}

.batch-rule-form .el-form-item {
  margin-bottom: 25px;
}

.batch-tip-text {
  margin-bottom: 20px;
  color: #606266;
  font-size: 14px;
  font-weight: 500;
}

.warning-section {
  margin-top: 10px;
}

.batch-warning-alert {
  margin: 15px 0;
}

.warning-content {
  line-height: 1.6;
}

.warning-title {
  font-weight: bold;
  margin-bottom: 8px;
  font-size: 15px;
}

.batch-form-footer {
  margin-top: 25px;
  padding-top: 20px;
  border-top: 1px dashed #ebeef5;
}

/* 移动端适配样式 */
@media (max-width: 767px) {
  .batch-rules-management {
    padding: 15px 10px;
  }

  .batch-rule-form .el-form-item {
    margin-bottom: 15px;
  }

  .batch-tip-text {
    margin-bottom: 15px;
  }

  .warning-content {
    font-size: 13px;
  }

  .warning-title {
    font-size: 14px;
    margin-bottom: 5px;
  }

  .form-tip {
    font-size: 11px;
  }

  .batch-form-footer {
    margin-top: 15px;
    padding-top: 15px;
  }

  .filter-row {
    flex-direction: column;
    align-items: flex-start;
  }

  .filter-item {
    width: 100%;
    margin-right: 0;
    margin-bottom: 10px;
  }

  .filter-item .el-select {
    width: 100% !important;
  }

  .action-buttons-container {
    display: flex;
    flex-direction: column;
    width: 100%;
    gap: 10px;
    margin-top: 10px;
  }

  .action-buttons-container .el-button {
    width: 100%;
    margin-left: 0 !important;
  }
}
</style>
