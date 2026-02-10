<script lang="ts" setup>
import type { Merchant } from "../apis/type"
import { getCloudAccountList } from "@@/apis/cloud_account"

const props = defineProps<{
  visible: boolean
  merchant: Merchant | null
}>()

const emit = defineEmits<{
  (e: "update:visible", value: boolean): void
}>()

// 计算属性 - 可见性双向绑定
const visible = computed({
  get: () => props.visible,
  set: value => emit("update:visible", value)
})

// 详情页按需加载AWS账号（从 CloudAccounts 查询 merchant 最新启用的 aws 账号）
const awsAccount = ref<null | {
  id: number
  access_key_id: string
  access_key_secret: string
}>(null)

async function loadAwsAccount() {
  if (!props.merchant) return
  const { data } = await getCloudAccountList({
    page: 1,
    size: 1,
    name: undefined,
    cloud_type: "aws",
    status: 1,
    account_type: "merchant",
    merchant_id: props.merchant.id
  } as any)
  const acc = (data.list || [])[0]
  if (acc) {
    awsAccount.value = {
      id: acc.id,
      access_key_id: acc.access_key_id,
      access_key_secret: acc.access_key_secret
    }
  } else {
    awsAccount.value = null
  }
}
</script>

<template>
  <el-dialog
    v-model="visible"
    :title="`商户详情: ${merchant?.name || ''}`"
    width="75%"
    destroy-on-close
  >
    <div v-if="merchant">
      <el-tabs>
        <el-tab-pane label="基本信息">
          <el-descriptions :column="2" border>
            <el-descriptions-item label="ID">
              {{ merchant.id }}
            </el-descriptions-item>
            <el-descriptions-item label="企业号">
              {{ merchant.no }}
            </el-descriptions-item>
            <el-descriptions-item label="服务器IP">
              {{ merchant.server_ip }}
            </el-descriptions-item>
            <el-descriptions-item label="商户名称">
              {{ merchant.name }}
            </el-descriptions-item>
            <el-descriptions-item label="状态">
              <el-tag :type="merchant.status === 1 ? 'success' : 'danger'">
                {{ merchant.status === 1 ? '正常' : '禁用' }}
              </el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="服务到期时间">
              {{ merchant.expired_at }}
            </el-descriptions-item>
            <el-descriptions-item label="到期状态">
              <el-tag :type="merchant.expiring_soon === 0 ? 'success' : merchant.expiring_soon === 1 ? 'warning' : 'danger'">
                {{ merchant.expiring_soon === 0 ? '正常' : merchant.expiring_soon === 1 ? '即将到期' : '已到期' }}
              </el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="创建时间">
              {{ merchant.created_at }}
            </el-descriptions-item>
            <el-descriptions-item label="更新时间">
              {{ merchant.updated_at }}
            </el-descriptions-item>
          </el-descriptions>
        </el-tab-pane>

        <el-tab-pane label="套餐配置">
          <el-descriptions :column="2" border v-if="merchant.package_configuration">
            <el-descriptions-item label="日活限制">
              {{ merchant.package_configuration.dau_limit }}
            </el-descriptions-item>
            <el-descriptions-item label="注册人数限制">
              {{ merchant.package_configuration.register_limit }}
            </el-descriptions-item>
            <el-descriptions-item label="群人数限制">
              {{ merchant.package_configuration.group_member_limit }}
            </el-descriptions-item>
            <el-descriptions-item label="套餐可用应用" :span="2">
              <div v-if="merchant.package_configuration.app_packages && merchant.package_configuration.app_packages.length > 0">
                <el-tag
                  v-for="pkg in merchant.package_configuration.app_packages"
                  :key="pkg"
                  type="info"
                  size="default"
                  style="margin-right: 8px; margin-bottom: 8px;"
                >
                  {{ pkg }}
                </el-tag>
              </div>
              <span v-else class="text-gray-400">不限制应用</span>
            </el-descriptions-item>
          </el-descriptions>
          <div v-else class="text-gray-400">
            无套餐配置
          </div>
        </el-tab-pane>
        <el-tab-pane label="云账号">
          <el-descriptions :column="2" border>
            <el-descriptions-item label="AWS Access Key">
              <template #default>
                <span v-if="awsAccount">{{ awsAccount.access_key_id || '-' }}</span>
                <el-button v-else type="primary" link @click="loadAwsAccount">点击加载</el-button>
              </template>
            </el-descriptions-item>
            <el-descriptions-item label="AWS Secret (掩码)">
              <template #default>
                <span v-if="awsAccount">{{ awsAccount.access_key_secret ? '********' : '-' }}</span>
                <el-button v-else type="primary" link @click="loadAwsAccount">点击加载</el-button>
              </template>
            </el-descriptions-item>
          </el-descriptions>
          <div class="text-gray-400 mt-2" v-if="!awsAccount">
            提示：从 CloudAccounts 加载该商户下最新的 AWS 账号
          </div>
        </el-tab-pane>
      </el-tabs>
    </div>
  </el-dialog>
</template>

<style scoped>
.ml-2 {
  margin-left: 8px;
}
.mr-2 {
  margin-right: 8px;
}
.mb-2 {
  margin-bottom: 8px;
}
.mb-4 {
  margin-bottom: 16px;
}
.mt-0 {
  margin-top: 0;
}
.mt-4 {
  margin-top: 16px;
}
.text-gray-400 {
  color: #9ca3af;
}
.text-green-500 {
  color: #10b981;
}
.text-red-500 {
  color: #ef4444;
}
.flex {
  display: flex;
}
.justify-between {
  justify-content: space-between;
}
.justify-end {
  justify-content: flex-end;
}
.items-center {
  align-items: center;
}
</style>
