<script lang="ts" setup>
import type { FormInstance, FormRules } from "element-plus"
import type { CreateOrEditMerchantRequestData, Merchant } from "../apis/type"
import { request } from "@/http/axios"
import { getCloudAccountList } from "@@/apis/cloud_account"
import { cloneDeep } from "lodash-es"

const props = defineProps<{
  visible: boolean
  merchant: Merchant | null
  formData: CreateOrEditMerchantRequestData
}>()

const emit = defineEmits<{
  (e: "update:visible", value: boolean): void
  (e: "update:formData", value: CreateOrEditMerchantRequestData): void
  (e: "submit", data: CreateOrEditMerchantRequestData): void
}>()

const formRef = ref<FormInstance | null>(null)
// const ipInput = ref("")
// const showIpInput = ref(false)
const localFormData = ref<CreateOrEditMerchantRequestData>(cloneDeep(props.formData))
const awsLoading = ref(false)
const systemAwsAccounts = ref<Array<{ id: number, name: string, access_key_id: string }>>([])
const selectedSystemAccountId = ref<number | undefined>(undefined)
const removeFromSystem = ref(false)

// 系统服务器列表
const systemServers = ref<Array<{ id: number, name: string, host: string }>>([])
const selectedServers = ref<number[]>([])

// 分页配置
const pagination = reactive({
  current: 1,
  pageSize: 10,
  total: 0
})

// 监听props的变化，更新本地数据
watch(() => props.formData, (newVal) => {
  localFormData.value = cloneDeep(newVal)
}, { deep: true })

// 加载系统类型的AWS账号列表（创建模式）
async function loadSystemAwsAccounts() {
  if (props.merchant) return // 仅创建模式需要
  try {
    const { data } = await getCloudAccountList({
      page: 1,
      size: 100,
      cloud_type: "aws",
      status: 1,
      account_type: "system"
    } as any)
    systemAwsAccounts.value = (data.list || []).map(acc => ({
      id: acc.id,
      name: acc.name,
      access_key_id: acc.access_key_id
    }))
  } catch (error) {
    console.error("加载系统AWS账号失败", error)
  }
}

// 加载系统服务器列表（创建模式）
async function loadSystemServers() {
  if (props.merchant) return // 仅创建模式需要
  try {
    const res = await request<any>({
      url: "/deploy/servers",
      method: "get",
      params: { server_type: 2, page: 1, size: 1000 }
    })
    systemServers.value = (res.data.list || []).map((s: any) => ({
      id: s.id,
      name: s.name,
      host: s.host
    }))
    pagination.total = systemServers.value.length
  } catch (error) {
    console.error("加载系统服务器失败", error)
  }
}

// 当前页的服务器列表
const pagedServers = computed(() => {
  const start = (pagination.current - 1) * pagination.pageSize
  const end = start + pagination.pageSize
  return systemServers.value.slice(start, end)
})

// 全选服务器
function selectAllServers() {
  selectedServers.value = systemServers.value.map(s => s.id)
}

// 清空选择
function clearAllServers() {
  selectedServers.value = []
}

// 编辑模式：加载当前商户的 AWS 云账号（仅填充 AccessKeyId，Secret 默认留空表示不修改）
async function loadAwsAccountForEdit() {
  if (!props.merchant || !props.visible) return
  awsLoading.value = true
  try {
    const { data } = await getCloudAccountList({
      page: 1,
      size: 1,
      cloud_type: "aws",
      status: 1,
      account_type: "merchant",
      merchant_id: props.merchant.id
    } as any)
    const acc = (data.list || [])[0]
    if (acc) {
      // 预填 AccessKeyId，Secret 留空（提示：留空不更新）
      localFormData.value.aws_access_key_id = acc.access_key_id || ""
      localFormData.value.aws_access_key_secret = ""
      emit("update:formData", localFormData.value)
    }
  } finally {
    awsLoading.value = false
  }
}

watch([
  () => props.visible,
  () => props.merchant && props.merchant.id
], () => {
  if (props.visible) {
    if (props.merchant) {
      loadAwsAccountForEdit()
    } else {
      // 创建模式：加载系统AWS账号列表和系统服务器列表
      loadSystemAwsAccounts()
      loadSystemServers()
      selectedSystemAccountId.value = undefined
      removeFromSystem.value = false
      selectedServers.value = []
      pagination.current = 1
    }
  }
})

// 表单校验规则
const formRules: FormRules = {
  name: [{ required: true, message: "请输入商户名称", trigger: "blur" }],
  server_ip: [{ required: !props.merchant, message: "请输入服务器IP", trigger: "blur" }],
  expired_at: [{ required: true, message: "请选择服务过期时间", trigger: "blur" }]
}

// 当选择系统AWS账号时，自动填充AccessKeyId
function onSelectSystemAccount(accountId: number | undefined) {
  if (!accountId) {
    localFormData.value.aws_access_key_id = ""
    localFormData.value.aws_access_key_secret = ""
    localFormData.value.selected_aws_account_id = undefined
    emit("update:formData", localFormData.value)
    return
  }
  const account = systemAwsAccounts.value.find(acc => acc.id === accountId)
  if (account) {
    localFormData.value.aws_access_key_id = account.access_key_id
    localFormData.value.aws_access_key_secret = "" // 系统账号的密钥已存在，不需要手动填写
    localFormData.value.selected_aws_account_id = accountId
    emit("update:formData", localFormData.value)
  }
}

// 提交表单
function submitForm() {
  formRef.value?.validate((valid) => {
    if (!valid) {
      ElMessage.error("请完善表单信息")
      return
    }
    // 如果选择了系统账号，添加相关字段
    if (selectedSystemAccountId.value) {
      localFormData.value.selected_aws_account_id = selectedSystemAccountId.value
      localFormData.value.remove_from_system = removeFromSystem.value
    } else {
      localFormData.value.selected_aws_account_id = undefined
      localFormData.value.remove_from_system = false
    }
    // 添加选中的系统服务器ID列表（仅创建模式）
    if (!props.merchant) {
      (localFormData.value as any).sync_gost_servers = selectedServers.value
    }
    emit("submit", localFormData.value)
  })
}

// 更新表单字段的工具函数
function updateName(val: string) {
  if (localFormData.value) {
    localFormData.value.name = val
    emit("update:formData", localFormData.value)
  }
}

// 更新套餐配置 - 日活限制
function updateDauLimit(val: number | undefined) {
  if (localFormData.value && typeof val === "number") {
    if (!localFormData.value.package_configuration) {
      localFormData.value.package_configuration = {
        dau_limit: val,
        register_limit: 0,
        group_member_limit: 0
      }
    } else {
      localFormData.value.package_configuration.dau_limit = val
    }
    emit("update:formData", localFormData.value)
  }
}

// 更新套餐配置 - 注册限制
function updateRegisterLimit(val: number | undefined) {
  if (localFormData.value && typeof val === "number") {
    if (!localFormData.value.package_configuration) {
      localFormData.value.package_configuration = {
        dau_limit: 0,
        register_limit: val,
        group_member_limit: 0
      }
    } else {
      localFormData.value.package_configuration.register_limit = val
    }
    emit("update:formData", localFormData.value)
  }
}

// 更新套餐配置 - 群人数限制
function updateGroupMemberLimit(val: number | undefined) {
  if (localFormData.value && typeof val === "number") {
    if (!localFormData.value.package_configuration) {
      localFormData.value.package_configuration = {
        dau_limit: 0,
        register_limit: 0,
        group_member_limit: val
      }
    } else {
      localFormData.value.package_configuration.group_member_limit = val
    }
    emit("update:formData", localFormData.value)
  }
}

// 更新套餐配置 - TURN服务器
function updateTurnServer(val: string) {
  if (localFormData.value) {
    if (!localFormData.value.package_configuration) {
      localFormData.value.package_configuration = {
        dau_limit: 0,
        register_limit: 0,
        group_member_limit: 0,
        turn_server: val
      }
    } else {
      localFormData.value.package_configuration.turn_server = val
    }
    emit("update:formData", localFormData.value)
  }
}

// 更新套餐配置 - TURN用户名
function updateTurnUsername(val: string) {
  if (localFormData.value) {
    if (!localFormData.value.package_configuration) {
      localFormData.value.package_configuration = {
        dau_limit: 0,
        register_limit: 0,
        group_member_limit: 0,
        turn_username: val
      }
    } else {
      localFormData.value.package_configuration.turn_username = val
    }
    emit("update:formData", localFormData.value)
  }
}

// 更新套餐配置 - TURN密码
function updateTurnCredential(val: string) {
  if (localFormData.value) {
    if (!localFormData.value.package_configuration) {
      localFormData.value.package_configuration = {
        dau_limit: 0,
        register_limit: 0,
        group_member_limit: 0,
        turn_credential: val
      }
    } else {
      localFormData.value.package_configuration.turn_credential = val
    }
    emit("update:formData", localFormData.value)
  }
}

// eslint-disable-next-line unused-imports/no-unused-vars
function updateStatus(val: string | number | boolean | undefined) {
  if (localFormData.value && typeof val === "number") {
    localFormData.value.status = val
    emit("update:formData", localFormData.value)
  }
}

function updateExpiredAt(val: string) {
  if (localFormData.value && val) {
    localFormData.value.expired_at = val
    emit("update:formData", localFormData.value)
  }
}

// 更新服务器IP
function updateServerIp(val: string) {
  if (!localFormData.value) return
  localFormData.value.server_ip = val
  emit("update:formData", localFormData.value)
}

// 更新 AWS Access Key / Secret（编辑模式留空表示不修改）
function updateAwsAccessKeyId(val: string) {
  if (!localFormData.value) return
  localFormData.value.aws_access_key_id = val
  emit("update:formData", localFormData.value)
}
function updateAwsAccessKeySecret(val: string) {
  if (!localFormData.value) return
  localFormData.value.aws_access_key_secret = val
  emit("update:formData", localFormData.value)
}

</script>

<template>
  <el-dialog
    :model-value="visible"
    :title="merchant ? '编辑商户' : '新增商户'"
    width="650px"
    destroy-on-close
    @update:model-value="$emit('update:visible', $event)"
  >
    <el-form
      ref="formRef"
      :model="localFormData"
      :rules="formRules"
      label-width="100px"
      label-position="right"
    >
      <el-form-item label="商户名称" prop="name">
        <el-input v-model="localFormData.name" placeholder="请输入商户名称" @update:model-value="updateName" />
      </el-form-item>

      <el-form-item label="服务器IP" prop="server_ip">
        <el-input v-model="localFormData.server_ip" placeholder="请输入服务器IP" @update:model-value="updateServerIp" />
      </el-form-item>

      <el-form-item label="企业号" prop="no">
        <el-input v-model="localFormData.no" placeholder="创建时自动计算，编辑不可修改" disabled />
      </el-form-item>

      <!-- <el-form-item label="商户状态" prop="status">
        <el-radio-group
          v-model="localFormData.status"
          :disabled="!merchant"
          @update:model-value="updateStatus"
        >
          <el-radio :value="1">
            正常
          </el-radio>
          <el-radio :value="-1">
            禁用
          </el-radio>
        </el-radio-group>
      </el-form-item> -->

      <el-form-item label="过期时间" prop="expired_at">
        <el-date-picker
          v-model="localFormData.expired_at"
          type="datetime"
          placeholder="选择过期时间"
          format="YYYY-MM-DD HH:mm:ss"
          value-format="YYYY-MM-DD HH:mm:ss"
          style="width: 100%"
          @update:model-value="updateExpiredAt"
        />
      </el-form-item>

      <el-divider content-position="left">
        套餐配置
      </el-divider>

      <el-form-item label="日活限制" prop="package_configuration.dau_limit">
        <el-input-number
          :model-value="localFormData.package_configuration?.dau_limit || 100"
          :min="0"
          :max="1000000"
          placeholder="日活限制"
          @update:model-value="updateDauLimit"
        />
      </el-form-item>

      <el-form-item label="注册人数限制" prop="package_configuration.register_limit">
        <el-input-number
          :model-value="localFormData.package_configuration?.register_limit || 100"
          :min="0"
          :max="1000000"
          placeholder="注册人数限制"
          @update:model-value="updateRegisterLimit"
        />
      </el-form-item>

      <el-form-item label="群人数限制" prop="package_configuration.group_member_limit">
        <el-input-number
          :model-value="localFormData.package_configuration?.group_member_limit || 100"
          :min="0"
          :max="100000"
          placeholder="群人数限制"
          @update:model-value="updateGroupMemberLimit"
        />
      </el-form-item>

      <el-form-item label="TURN服务器" prop="package_configuration.turn_server">
        <el-input
          :model-value="localFormData.package_configuration?.turn_server || ''"
          placeholder="音视频TURN服务器地址 (格式: ip:port)"
          @update:model-value="updateTurnServer"
        />
        <div class="text-gray-500" style="font-size: 12px; margin-top: 4px;">
          用于音视频通话的TURN服务器，格式如：192.168.1.100:3478
        </div>
      </el-form-item>

      <el-form-item label="TURN用户名" prop="package_configuration.turn_username">
        <el-input
          :model-value="localFormData.package_configuration?.turn_username || ''"
          placeholder="TURN服务器用户名"
          @update:model-value="updateTurnUsername"
        />
      </el-form-item>

      <el-form-item label="TURN密码" prop="package_configuration.turn_credential">
        <el-input
          :model-value="localFormData.package_configuration?.turn_credential || ''"
          placeholder="TURN服务器密码"
          show-password
          @update:model-value="updateTurnCredential"
        />
      </el-form-item>

      <el-divider content-position="left">
        AWS 云账号
      </el-divider>

      <!-- 创建模式：选择系统AWS账号 -->
      <template v-if="!merchant">
        <el-form-item label="选择系统账号">
          <el-select
            v-model="selectedSystemAccountId"
            placeholder="可选择现有系统AWS账号"
            clearable
            filterable
            style="width: 100%"
            @change="onSelectSystemAccount"
          >
            <el-option
              v-for="acc in systemAwsAccounts"
              :key="acc.id"
              :label="`${acc.name} (${acc.access_key_id})`"
              :value="acc.id"
            />
          </el-select>
        </el-form-item>

        <el-form-item v-if="selectedSystemAccountId" label=" ">
          <el-checkbox v-model="removeFromSystem">
            从系统账号中移除（转为商户专用账号）
          </el-checkbox>
          <div class="text-gray-500" style="font-size: 12px; margin-top: 4px;">
            勾选后，该账号将从系统账号列表中移除，并转换为此商户的专用账号；<br>
            不勾选则复制一份新的账号给商户，系统账号保持不变
          </div>
        </el-form-item>

        <el-divider v-if="selectedSystemAccountId" content-position="center">
          或手动填写
        </el-divider>
      </template>

      <el-row :gutter="12">
        <el-col :span="12">
          <el-form-item label="AccessKey" prop="aws_access_key_id">
            <el-input
              v-model="localFormData.aws_access_key_id"
              :disabled="!!selectedSystemAccountId && !merchant"
              :placeholder="merchant ? '留空表示不修改' : (selectedSystemAccountId ? '已自动填充' : '请输入 AWS Access Key')"
              @update:model-value="updateAwsAccessKeyId"
            />
          </el-form-item>
        </el-col>
        <el-col :span="12">
          <el-form-item label="SecretKey" prop="aws_access_key_secret">
            <el-input
              v-model="localFormData.aws_access_key_secret"
              type="password"
              show-password
              :disabled="!!selectedSystemAccountId && !merchant"
              :placeholder="merchant ? '留空表示不修改' : (selectedSystemAccountId ? '已自动使用系统账号密钥' : '请输入 AWS Access Secret')"
              @update:model-value="updateAwsAccessKeySecret"
            />
          </el-form-item>
        </el-col>
      </el-row>
      <div v-if="merchant" class="text-gray-500" style="font-size: 12px; margin-top: 4px;">
        留空表示不修改
      </div>

      <!-- <div class="cloud-account-section">
        <h4 class="cloud-provider-title">
          华为云
        </h4>
        <el-form-item label="AccessKey" prop="cloud_account.huawei.access_key">
          <el-input
            :model-value="localFormData.cloud_account?.huawei?.access_key || ''"
            placeholder="请输入华为云Access Key"
            @update:model-value="updateHuaweiAccessKey"
          />
        </el-form-item>

        <el-form-item label="SecretKey" prop="cloud_account.huawei.access_secret">
          <el-input
            :model-value="localFormData.cloud_account?.huawei?.access_secret || ''"
            placeholder="请输入华为云Access Secret"
            type="password"
            show-password
            @update:model-value="updateHuaweiAccessSecret"
          />
        </el-form-item>
      </div> -->

      <!-- <div class="cloud-account-section">
        <h4 class="cloud-provider-title">
          腾讯云
        </h4>
        <el-form-item label="AccessKey" prop="cloud_account.tencent.access_key">
          <el-input
            :model-value="localFormData.cloud_account?.tencent?.access_key || ''"
            placeholder="请输入腾讯云Access Key"
            @update:model-value="updateTencentAccessKey"
          />
        </el-form-item>

        <el-form-item label="SecretKey" prop="cloud_account.tencent.access_secret">
          <el-input
            :model-value="localFormData.cloud_account?.tencent?.access_secret || ''"
            placeholder="请输入腾讯云Access Secret"
            type="password"
            show-password
            @update:model-value="updateTencentAccessSecret"
          />
        </el-form-item>
      </div> -->

      <!-- 系统服务器选择（仅创建模式） -->
      <template v-if="!merchant">
        <el-divider content-position="left">
          同步Gost服务
        </el-divider>

        <el-form-item label="选择系统服务器">
          <div class="server-select-container">
            <div v-if="systemServers.length === 0" class="empty-tip">
              暂无可用的系统服务器
            </div>
            <template v-else>
              <div class="server-list-header">
                <span class="info-text">
                  为选中的系统服务器创建Gost转发服务（共 {{ systemServers.length }} 个，已选 {{ selectedServers.length }} 个）
                </span>
                <div class="header-actions">
                  <el-button link type="primary" size="small" @click="selectAllServers">
                    全选
                  </el-button>
                  <el-button link type="info" size="small" @click="clearAllServers">
                    清空
                  </el-button>
                </div>
              </div>
              <el-checkbox-group v-model="selectedServers" class="server-checkbox-list">
                <el-checkbox
                  v-for="s in pagedServers"
                  :key="s.id"
                  :label="s.id"
                  class="server-checkbox-item"
                >
                  <span class="server-name">{{ s.name }}</span>
                  <span class="server-info">主机: {{ s.host }}</span>
                </el-checkbox>
              </el-checkbox-group>
              <el-pagination
                v-if="pagination.total > pagination.pageSize"
                v-model:current-page="pagination.current"
                v-model:page-size="pagination.pageSize"
                :total="pagination.total"
                :page-sizes="[10, 20, 50]"
                layout="total, sizes, prev, pager, next, jumper"
                small
                background
                class="server-pagination"
              />
            </template>
          </div>
        </el-form-item>
      </template>
    </el-form>

    <template #footer>
      <el-button @click="$emit('update:visible', false)">
        取消
      </el-button>
      <el-button type="primary" @click="submitForm">
        确定
      </el-button>
    </template>
  </el-dialog>
</template>

<style lang="scss" scoped>
.region-container {
  display: flex;
  flex-direction: column;
}

.region-header {
  margin-bottom: 15px;
}

.region-content {
  width: 100%;
}

.region-list {
  border: 1px dashed #e0e0e0;
  border-radius: 4px;
  padding: 15px;
  margin-bottom: 10px;
}

.region-item {
  display: flex;
  align-items: center;
  margin-bottom: 10px;
  gap: 10px;
}

.region-name {
  flex: 3;
}

.region-bandwidth {
  flex: 2;
  margin-right: 10px;
}

.no-data-tip {
  color: #909399;
  font-size: 14px;
  padding: 5px;
  text-align: center;
  width: 100%;
}

.ip-text {
  display: flex;
  align-items: center;
  padding: 0 15px;
  height: 32px;
  background-color: #f5f7fa;
  border-radius: 4px;
  border: 1px solid #e4e7ed;
}

.cloud-account-section {
  margin-bottom: 20px;
  padding: 15px;
  border: 1px solid #e8e8e8;
  border-radius: 4px;
  background-color: #fafafa;
}

.cloud-provider-title {
  margin-bottom: 15px;
  font-size: 16px;
  color: #409eff;
  padding-bottom: 8px;
  border-bottom: 1px dashed #e0e0e0;
}

.ip-list {
  display: flex;
  flex-wrap: wrap;
  gap: 5px;
}

.ip-tag {
  background-color: #f0f0f0;
  padding: 5px 10px;
  border-radius: 4px;
  margin-right: 5px;
  margin-bottom: 5px;
}

.ip-input-wrapper {
  display: flex;
  align-items: center;
  gap: 10px;
}

.ip-input-buttons {
  display: flex;
  gap: 5px;
}

.region-input-group {
  display: flex;
  align-items: center;
  gap: 10px;
}

.kv-list .kv-item {
  display: flex;
  gap: 10px;
  margin-bottom: 10px;
}

.kv-add {
  display: flex;
  gap: 10px;
}

.server-select-container {
  width: 100%;

  .empty-tip {
    color: #999;
    font-size: 12px;
    padding: 12px;
    background: #f5f7fa;
    border-radius: 4px;
    text-align: center;
  }

  .server-list-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 12px;
    padding-bottom: 8px;
    border-bottom: 1px solid #ebeef5;

    .info-text {
      font-size: 13px;
      color: #606266;
    }

    .header-actions {
      display: flex;
      gap: 8px;
    }
  }

  .server-checkbox-list {
    display: flex;
    flex-direction: column;
    border: 1px solid #dcdfe6;
    border-radius: 4px;
    padding: 12px;
    background: #fafafa;
    min-height: 60px;

    .server-checkbox-item {
      display: flex;
      align-items: center;
      padding: 12px;
      margin: 0 0 8px 0;
      background: white;
      border-radius: 4px;
      border: 1px solid #e4e7ed;
      transition: all 0.3s;

      &:hover {
        border-color: #409eff;
        box-shadow: 0 2px 4px rgba(64, 158, 255, 0.1);
      }

      &:last-child {
        margin-bottom: 0;
      }

      :deep(.el-checkbox__label) {
        flex: 1;
        display: flex;
        flex-direction: column;
        gap: 4px;
      }

      .server-name {
        font-size: 14px;
        font-weight: 500;
        color: #303133;
      }

      .server-info {
        font-size: 12px;
        color: #909399;
      }
    }
  }

  .server-pagination {
    margin-top: 16px;
    display: flex;
    justify-content: center;
  }
}
</style>
