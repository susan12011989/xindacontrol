<template>
  <div class="app-container">
    <!-- 顶部：商户选择 -->
    <el-card class="mb-4">
      <div class="flex justify-between items-center">
        <div class="flex items-center gap-4">
          <span class="text-base font-bold">商户:</span>
          <el-select
            v-model="selectedMerchantId"
            placeholder="请选择商户"
            filterable
            clearable
            style="width: 300px"
            @change="handleMerchantChange"
          >
            <el-option
              v-for="item in merchantList"
              :key="item.id"
              :label="`${item.name} (${item.no})`"
              :value="item.id"
            />
          </el-select>
        </div>
        <el-button @click="loadStatus" :loading="loading" :disabled="!selectedMerchantId">
          <el-icon><Refresh /></el-icon> 刷新
        </el-button>
      </div>
    </el-card>

    <!-- 限流状态 -->
    <template v-if="selectedMerchantId">
      <!-- 限流开关 -->
      <el-card class="mb-4" v-loading="loading">
        <template #header>
          <div class="flex justify-between items-center">
            <span class="font-bold">HTTP 限流开关</span>
            <el-switch
              v-model="rateLimitEnabled"
              :loading="toggleLoading"
              active-text="已开启"
              inactive-text="已关闭"
              inline-prompt
              style="--el-switch-on-color: #13ce66; --el-switch-off-color: #ff4949"
              @change="handleToggle"
            />
          </div>
        </template>
        <div class="text-sm text-gray-500">
          <p>限流规则：单 IP 每分钟最多 500 次请求，超限返回 HTTP 429。</p>
          <p>关闭限流后，所有请求将不受频率限制。白名单 IP 始终不受限流约束。</p>
        </div>
      </el-card>

      <!-- 白名单管理 -->
      <el-card v-loading="loading">
        <template #header>
          <div class="flex justify-between items-center">
            <span class="font-bold">IP 白名单</span>
            <el-button type="primary" size="small" @click="showAddDialog = true">
              <el-icon><Plus /></el-icon> 添加 IP
            </el-button>
          </div>
        </template>

        <el-table :data="whitelist" style="width: 100%" empty-text="暂无白名单 IP">
          <el-table-column prop="ip" label="IP 地址" />
          <el-table-column label="操作" width="100" fixed="right">
            <template #default="{ row }">
              <el-popconfirm
                :title="`确定移除 ${row.ip} ？`"
                confirm-button-text="移除"
                cancel-button-text="取消"
                @confirm="handleRemoveIP(row.ip)"
              >
                <template #reference>
                  <el-button link type="danger" size="small">移除</el-button>
                </template>
              </el-popconfirm>
            </template>
          </el-table-column>
        </el-table>
      </el-card>
    </template>

    <!-- 未选择商户提示 -->
    <el-card v-else>
      <el-empty description="请先选择商户" />
    </el-card>

    <!-- 添加 IP 对话框 -->
    <el-dialog v-model="showAddDialog" title="添加白名单 IP" width="400px">
      <el-form @submit.prevent="handleAddIP">
        <el-form-item label="IP 地址">
          <el-input v-model="newIP" placeholder="例如: 1.2.3.4" clearable />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showAddDialog = false">取消</el-button>
        <el-button type="primary" :loading="addLoading" @click="handleAddIP">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from "vue"
import { ElMessage } from "element-plus"
import { Refresh, Plus } from "@element-plus/icons-vue"
import { getMerchantList } from "@/common/apis/merchant"
import { getRateLimitStatus, toggleRateLimit, addWhitelistIP, removeWhitelistIP } from "@/common/apis/deploy"

interface MerchantOption {
  id: number
  no: string
  name: string
}

const selectedMerchantId = ref<number>()
const merchantList = ref<MerchantOption[]>([])
const rateLimitEnabled = ref(true)
const whitelist = ref<{ ip: string }[]>([])
const loading = ref(false)
const toggleLoading = ref(false)
const addLoading = ref(false)
const showAddDialog = ref(false)
const newIP = ref("")

// 加载商户列表
async function loadMerchants() {
  try {
    const res = await getMerchantList({ page: 1, size: 1000 }) as any
    if ((res.code === 0 || res.code === 200) && res.data?.list) {
      merchantList.value = res.data.list.map((m: any) => ({
        id: m.id,
        no: m.no,
        name: m.name
      }))
    } else if (res.list) {
      merchantList.value = res.list.map((m: any) => ({
        id: m.id,
        no: m.no,
        name: m.name
      }))
    }
  } catch (error) {
    console.error("加载商户列表失败:", error)
  }
}

// 加载限流状态
async function loadStatus() {
  if (!selectedMerchantId.value) return

  loading.value = true
  try {
    const res = await getRateLimitStatus(selectedMerchantId.value) as any
    const data = res?.data || res
    rateLimitEnabled.value = data.enabled !== false
    whitelist.value = (data.whitelist || []).map((ip: string) => ({ ip }))
  } catch (error) {
    ElMessage.error("获取限流状态失败")
    console.error(error)
  } finally {
    loading.value = false
  }
}

// 商户选择变更
function handleMerchantChange() {
  if (selectedMerchantId.value) {
    loadStatus()
  }
}

// 切换限流开关
async function handleToggle(val: boolean | string | number) {
  if (!selectedMerchantId.value) return

  toggleLoading.value = true
  try {
    await toggleRateLimit({ merchant_id: selectedMerchantId.value, enabled: !!val })
    ElMessage.success(`限流已${val ? "开启" : "关闭"}`)
  } catch (error) {
    ElMessage.error("操作失败")
    rateLimitEnabled.value = !val // 回滚
    console.error(error)
  } finally {
    toggleLoading.value = false
  }
}

// 添加白名单 IP
async function handleAddIP() {
  if (!selectedMerchantId.value || !newIP.value.trim()) {
    ElMessage.warning("请输入 IP 地址")
    return
  }

  const ip = newIP.value.trim()
  if (!/^\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}$/.test(ip)) {
    ElMessage.warning("请输入有效的 IP 地址")
    return
  }

  addLoading.value = true
  try {
    await addWhitelistIP({ merchant_id: selectedMerchantId.value, ip })
    ElMessage.success(`已添加 ${ip} 到白名单`)
    newIP.value = ""
    showAddDialog.value = false
    await loadStatus()
  } catch (error) {
    ElMessage.error("添加失败")
    console.error(error)
  } finally {
    addLoading.value = false
  }
}

// 移除白名单 IP
async function handleRemoveIP(ip: string) {
  if (!selectedMerchantId.value) return

  try {
    await removeWhitelistIP({ merchant_id: selectedMerchantId.value, ip })
    ElMessage.success(`已移除 ${ip}`)
    await loadStatus()
  } catch (error) {
    ElMessage.error("移除失败")
    console.error(error)
  }
}

onMounted(() => {
  loadMerchants()
})
</script>

<style scoped>
.app-container {
  padding: 20px;
}
.mb-4 {
  margin-bottom: 16px;
}
.flex {
  display: flex;
}
.justify-between {
  justify-content: space-between;
}
.items-center {
  align-items: center;
}
.gap-4 {
  gap: 16px;
}
.text-base {
  font-size: 16px;
}
.font-bold {
  font-weight: 600;
}
.text-sm {
  font-size: 14px;
}
.text-gray-500 {
  color: #6b7280;
}
.text-gray-500 p {
  margin: 4px 0;
}
</style>
