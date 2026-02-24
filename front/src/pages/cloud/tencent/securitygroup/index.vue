<script lang="ts" setup>
import type * as Types from "./apis/type"
import { authorizeSecurityGroup, describeSecurityGroupPolicies, getSecurityGroupList, revokeSecurityGroup } from "./apis"
import { getCloudAccountOptions } from "@@/apis/cloud_account"
import { getTencentRegions } from "@@/constants/tencent-regions"
import { ArrowDown, Delete, Edit, Plus } from "@element-plus/icons-vue"
import { useRouter } from "vue-router"

defineOptions({ name: "TencentSecurityGroup" })

const router = useRouter()

const loading = ref(false)
const rows = ref<Types.SecurityGroup[]>([])

// 账号类型与账号选择
const accountType = ref<"merchant" | "system">("system")
const cloudAccounts = ref<{ value: number; label: string }[]>([])
const selectedCloudAccount = ref<number>()
const selectedMerchant = ref<number>()
const merchantOptions = ref<{ id: number; name: string }[]>([])

// 区域选择（单选）
const tencentRegions = getTencentRegions("cn")
const selectedRegion = ref<string>("")

// 安全组规则详情 / 管理
const detailVisible = ref(false)
const detailLoading = ref(false)
const currentGroup = ref<Types.SecurityGroup | null>(null)
const policySet = ref<Types.SecurityGroupPolicySet | null>(null)

// 添加规则表单
const ruleForm = reactive<Types.AuthorizeSecurityGroupPolicy>({
  protocol: "TCP",
  port: "",
  cidr_block: "",
  action: "ACCEPT",
  description: ""
})
const ruleFormRef = ref()
const ruleFormRules = {
  port: [{ required: true, message: "请输入端口", trigger: "blur" }],
  cidr_block: [{ required: true, message: "请输入源IP地址", trigger: "blur" }]
}
const addRuleLoading = ref(false)
const removeRuleLoading = ref(false)

function buildAccountParams(): { merchant_id?: number; cloud_account_id?: number } {
  if (accountType.value === "system") return { cloud_account_id: selectedCloudAccount.value }
  return { merchant_id: selectedMerchant.value }
}

async function fetchCloudAccounts() {
  const res = await getCloudAccountOptions("tencent")
  cloudAccounts.value = res.data || []
}

async function searchMerchant(query: string) {
  const { merchantQueryApi } = await import("@/pages/dashboard/apis")
  const { data } = await merchantQueryApi({ page: 1, size: 20, name: query })
  merchantOptions.value = data.list || []
}

watch(accountType, () => {
  selectedCloudAccount.value = undefined
  selectedMerchant.value = undefined
  selectedRegion.value = ""
  rows.value = []
  if (accountType.value === "system") fetchCloudAccounts()
})

onMounted(() => {
  fetchCloudAccounts()
})

async function onQuery() {
  if (!selectedRegion.value) return ElMessage.warning("请选择区域")
  const acct = buildAccountParams()
  if (!acct.merchant_id && !acct.cloud_account_id) return ElMessage.warning("请选择账号")

  loading.value = true
  try {
    const { data } = await getSecurityGroupList({ ...acct, region_id: selectedRegion.value })
    rows.value = data.list || []
    if (rows.value.length === 0) ElMessage.info("当前区域暂无安全组")
  } finally {
    loading.value = false
  }
}

// ========== 规则管理 ==========

async function openRulesDialog(group: Types.SecurityGroup) {
  currentGroup.value = group
  detailVisible.value = true
  resetRuleForm()
  await fetchPolicies()
}

async function fetchPolicies() {
  if (!currentGroup.value) return
  detailLoading.value = true
  policySet.value = null
  try {
    const acct = buildAccountParams()
    const { data } = await describeSecurityGroupPolicies({
      ...acct,
      region_id: selectedRegion.value,
      security_group_id: currentGroup.value.SecurityGroupId
    })
    policySet.value = data
  } catch (e) {
    ElMessage.error("获取安全组规则失败")
  } finally {
    detailLoading.value = false
  }
}

function resetRuleForm() {
  ruleForm.protocol = "TCP"
  ruleForm.port = ""
  ruleForm.cidr_block = ""
  ruleForm.action = "ACCEPT"
  ruleForm.description = ""
  ruleFormRef.value?.resetFields()
}

async function addRule(formEl: any) {
  if (!formEl || !currentGroup.value) return
  await formEl.validate(async (valid: boolean) => {
    if (!valid) return
    addRuleLoading.value = true
    try {
      const acct = buildAccountParams()
      await authorizeSecurityGroup({
        ...acct,
        region_id: selectedRegion.value,
        security_group_id: currentGroup.value!.SecurityGroupId,
        policies: [{ ...ruleForm }]
      })
      ElMessage.success("添加规则成功")
      resetRuleForm()
      fetchPolicies()
    } catch {
      ElMessage.error("添加规则失败")
    } finally {
      addRuleLoading.value = false
    }
  })
}

async function removeRule(policy: Types.SecurityGroupPolicy) {
  if (!currentGroup.value) return
  await ElMessageBox.confirm("确定要删除该安全组规则吗？", "删除确认", {
    confirmButtonText: "确定",
    cancelButtonText: "取消",
    type: "warning"
  })
  removeRuleLoading.value = true
  try {
    const acct = buildAccountParams()
    await revokeSecurityGroup({
      ...acct,
      region_id: selectedRegion.value,
      security_group_id: currentGroup.value.SecurityGroupId,
      policy_indexes: [policy.PolicyIndex]
    })
    ElMessage.success("删除规则成功")
    fetchPolicies()
  } catch {
    ElMessage.error("删除规则失败")
  } finally {
    removeRuleLoading.value = false
  }
}

function getProtocolText(protocol: string) {
  const map: Record<string, string> = { TCP: "TCP", UDP: "UDP", ICMP: "ICMP", ALL: "全部" }
  return map[protocol?.toUpperCase()] || protocol
}

function getActionText(action: string) {
  return action?.toUpperCase() === "ACCEPT" ? "允许" : "拒绝"
}

function getActionTagType(action: string): "success" | "danger" {
  return action?.toUpperCase() === "ACCEPT" ? "success" : "danger"
}
</script>

<template>
  <div class="container">
    <el-card class="filter-card">
      <div class="filter-row">
        <div class="filter-item">
          <span class="label">账号类型：</span>
          <el-select v-model="accountType" style="width: 150px">
            <el-option label="系统类型" value="system" />
            <el-option label="商户类型" value="merchant" />
          </el-select>
        </div>
        <div v-if="accountType === 'system'" class="filter-item">
          <span class="label">云账号：</span>
          <el-select v-model="selectedCloudAccount" placeholder="请选择云账号" filterable clearable style="width: 240px">
            <el-option v-for="opt in cloudAccounts" :key="opt.value" :label="opt.label" :value="opt.value" />
          </el-select>
        </div>
        <div v-else class="filter-item">
          <span class="label">商户：</span>
          <el-select v-model="selectedMerchant" placeholder="搜索商户" filterable remote clearable :remote-method="searchMerchant" style="width: 260px">
            <el-option v-for="m in merchantOptions" :key="m.id" :label="m.name" :value="m.id" />
          </el-select>
        </div>
        <div class="filter-item">
          <span class="label">区域：</span>
          <el-select v-model="selectedRegion" filterable clearable placeholder="请选择区域" style="width: 300px">
            <el-option v-for="r in tencentRegions" :key="r.id" :label="`${r.name} (${r.id})`" :value="r.id" />
          </el-select>
        </div>
        <el-button type="primary" :loading="loading" @click="onQuery">查询</el-button>
      </div>
    </el-card>

    <el-card class="table-card">
      <template #header>
        <div class="card-header">
          <span>安全组列表</span>
          <div style="display: flex; gap: 8px">
            <el-button type="success" size="small" @click="router.push('/cloud/tencent/securitygroup/create')">
              <el-icon><Plus /></el-icon> 创建安全组
            </el-button>
            <el-button type="primary" size="small" @click="onQuery">刷新</el-button>
          </div>
        </div>
      </template>

      <el-table :data="rows" v-loading="loading" border style="width: 100%">
        <el-table-column prop="SecurityGroupId" label="安全组ID" min-width="200" />
        <el-table-column prop="SecurityGroupName" label="名称" min-width="160" />
        <el-table-column prop="SecurityGroupDesc" label="描述" min-width="180" show-overflow-tooltip />
        <el-table-column prop="CreatedTime" label="创建时间" min-width="180" />
        <el-table-column label="操作" fixed="right" min-width="120">
          <template #default="{ row }">
            <el-dropdown trigger="click">
              <el-button type="primary" text size="small">
                更多<el-icon class="el-icon--right"><ArrowDown /></el-icon>
              </el-button>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item @click="openRulesDialog(row)">
                    <el-icon><Edit /></el-icon> 规则管理
                  </el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- 安全组规则管理对话框 -->
    <el-dialog
      v-model="detailVisible"
      :title="`安全组规则管理 - ${currentGroup?.SecurityGroupName || currentGroup?.SecurityGroupId || ''}`"
      width="950px"
      destroy-on-close
    >
      <div v-loading="detailLoading" class="rules-management">
        <!-- 添加规则表单 -->
        <div class="add-rule-section">
          <h3>添加入站规则</h3>
          <el-form
            ref="ruleFormRef"
            :model="ruleForm"
            :rules="ruleFormRules"
            label-width="100px"
          >
            <el-row :gutter="20">
              <el-col :span="8">
                <el-form-item label="协议">
                  <el-select v-model="ruleForm.protocol" style="width: 100%">
                    <el-option label="TCP" value="TCP" />
                    <el-option label="UDP" value="UDP" />
                    <el-option label="ICMP" value="ICMP" />
                    <el-option label="ALL" value="ALL" />
                  </el-select>
                </el-form-item>
              </el-col>
              <el-col :span="8">
                <el-form-item label="端口" prop="port">
                  <el-input v-model="ruleForm.port" placeholder="如: 80 或 8000-9000" />
                </el-form-item>
              </el-col>
              <el-col :span="8">
                <el-form-item label="源IP" prop="cidr_block">
                  <el-input v-model="ruleForm.cidr_block" placeholder="如: 0.0.0.0/0" />
                </el-form-item>
              </el-col>
            </el-row>
            <el-row :gutter="20">
              <el-col :span="8">
                <el-form-item label="策略">
                  <el-select v-model="ruleForm.action" style="width: 100%">
                    <el-option label="允许 (ACCEPT)" value="ACCEPT" />
                    <el-option label="拒绝 (DROP)" value="DROP" />
                  </el-select>
                </el-form-item>
              </el-col>
              <el-col :span="8">
                <el-form-item label="描述">
                  <el-input v-model="ruleForm.description" placeholder="可选" />
                </el-form-item>
              </el-col>
              <el-col :span="8">
                <el-form-item>
                  <el-button type="primary" :loading="addRuleLoading" @click="addRule(ruleFormRef)">
                    添加规则
                  </el-button>
                  <el-button @click="resetRuleForm">重置</el-button>
                </el-form-item>
              </el-col>
            </el-row>
          </el-form>
        </div>

        <!-- 入站规则列表 -->
        <div class="rule-list-section">
          <h3>入站规则 (Ingress)</h3>
          <div v-if="policySet?.Ingress?.length">
            <el-table :data="policySet.Ingress" border style="width: 100%">
              <el-table-column prop="PolicyIndex" label="#" width="60" />
              <el-table-column label="协议" width="80">
                <template #default="{ row }">{{ getProtocolText(row.Protocol) }}</template>
              </el-table-column>
              <el-table-column prop="Port" label="端口" width="120" />
              <el-table-column prop="CidrBlock" label="源IP" min-width="150" />
              <el-table-column label="策略" width="80">
                <template #default="{ row }">
                  <el-tag size="small" :type="getActionTagType(row.Action)">{{ getActionText(row.Action) }}</el-tag>
                </template>
              </el-table-column>
              <el-table-column prop="PolicyDescription" label="描述" min-width="120" show-overflow-tooltip />
              <el-table-column prop="ModifyTime" label="修改时间" width="180" />
              <el-table-column label="操作" width="80">
                <template #default="{ row }">
                  <el-button type="danger" text size="small" :loading="removeRuleLoading" @click="removeRule(row)">
                    <el-icon><Delete /></el-icon>
                  </el-button>
                </template>
              </el-table-column>
            </el-table>
          </div>
          <el-empty v-else description="暂无入站规则" :image-size="60" />
        </div>

        <!-- 出站规则列表 -->
        <div class="rule-list-section">
          <h3>出站规则 (Egress)</h3>
          <div v-if="policySet?.Egress?.length">
            <el-table :data="policySet.Egress" border style="width: 100%">
              <el-table-column prop="PolicyIndex" label="#" width="60" />
              <el-table-column label="协议" width="80">
                <template #default="{ row }">{{ getProtocolText(row.Protocol) }}</template>
              </el-table-column>
              <el-table-column prop="Port" label="端口" width="120" />
              <el-table-column prop="CidrBlock" label="目标IP" min-width="150" />
              <el-table-column label="策略" width="80">
                <template #default="{ row }">
                  <el-tag size="small" :type="getActionTagType(row.Action)">{{ getActionText(row.Action) }}</el-tag>
                </template>
              </el-table-column>
              <el-table-column prop="PolicyDescription" label="描述" min-width="120" show-overflow-tooltip />
              <el-table-column prop="ModifyTime" label="修改时间" width="180" />
            </el-table>
          </div>
          <el-empty v-else description="暂无出站规则" :image-size="60" />
        </div>
      </div>
    </el-dialog>
  </div>
</template>

<style lang="scss" scoped>
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

.rules-management {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.add-rule-section {
  border: 1px solid #ebeef5;
  border-radius: 4px;
  padding: 20px;
  background-color: #f8f8f8;
}

.add-rule-section h3,
.rule-list-section h3 {
  margin-top: 0;
  margin-bottom: 16px;
  font-size: 15px;
  font-weight: 500;
  color: #303133;
  border-left: 3px solid #409eff;
  padding-left: 10px;
}
</style>
