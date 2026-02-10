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
        <el-button-group>
          <el-button @click="loadFeatureFlags" :loading="loading" :disabled="!selectedMerchantId">
            <el-icon><Refresh /></el-icon> 刷新
          </el-button>
          <el-button type="success" :disabled="!selectedMerchantId" :loading="batchLoading" @click="enableAll">
            全部启用
          </el-button>
          <el-button type="danger" :disabled="!selectedMerchantId" :loading="batchLoading" @click="disableAll">
            全部禁用
          </el-button>
        </el-button-group>
      </div>
    </el-card>

    <!-- 功能开关列表 -->
    <el-card v-if="selectedMerchantId" v-loading="loading">
      <el-table :data="groupedFlags" style="width: 100%">
        <el-table-column prop="category" label="分类" width="100">
          <template #default="{ row }">
            <el-tag :type="getCategoryType(row.category)" size="small">{{ row.category }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="label" label="功能名称" width="140" />
        <el-table-column prop="description" label="功能说明" min-width="200" />
        <el-table-column label="状态" width="160">
          <template #default="{ row }">
            <el-tag :type="row.enabled ? 'success' : 'info'" size="small">
              {{ row.enabled ? '已启用' : '已禁用' }}
            </el-tag>
            <span v-if="row.updated_at" class="ml-2 text-xs text-gray-400">{{ row.updated_at }}</span>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="120" fixed="right">
          <template #default="{ row }">
            <el-button
              v-if="row.enabled"
              link
              type="warning"
              :loading="row.updating"
              @click="handleToggle(row, false)"
            >
              禁用
            </el-button>
            <el-button
              v-else
              link
              type="success"
              :loading="row.updating"
              @click="handleToggle(row, true)"
            >
              启用
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- 未选择商户提示 -->
    <el-card v-else>
      <el-empty description="请先选择商户" />
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed } from "vue"
import { ElMessage, ElMessageBox } from "element-plus"
import { Refresh } from "@element-plus/icons-vue"
import { getMerchantList } from "@/common/apis/merchant"
import { getFeatureFlags, updateFeatureFlag, batchUpdateFeatureFlags } from "@/common/apis/feature"
import type { FeatureFlagResp } from "@/common/apis/feature/type"

interface MerchantOption {
  id: number
  no: string
  name: string
}

interface FeatureFlagRow extends FeatureFlagResp {
  updating?: boolean
}

const selectedMerchantId = ref<number>()
const merchantList = ref<MerchantOption[]>([])
const featureFlags = ref<FeatureFlagRow[]>([])
const loading = ref(false)
const batchLoading = ref(false)

// 按分类排序的功能列表
const groupedFlags = computed(() => {
  const categoryOrder = ["支付", "通讯", "社交", "安全", "工具", "娱乐", "服务"]
  return [...featureFlags.value].sort((a, b) => {
    const orderA = categoryOrder.indexOf(a.category)
    const orderB = categoryOrder.indexOf(b.category)
    return orderA - orderB
  })
})

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

// 加载功能开关
async function loadFeatureFlags() {
  if (!selectedMerchantId.value) return

  loading.value = true
  try {
    const res = await getFeatureFlags(selectedMerchantId.value) as any
    if (res.code === 0 || res.code === 200) {
      featureFlags.value = res.data.list.map((f: FeatureFlagResp) => ({
        ...f,
        updating: false
      }))
    } else {
      ElMessage.error(res.message || "加载失败")
    }
  } catch (error) {
    ElMessage.error("加载功能开关失败")
    console.error(error)
  } finally {
    loading.value = false
  }
}

// 商户变更
function handleMerchantChange() {
  loadFeatureFlags()
}

// 切换开关
async function handleToggle(row: FeatureFlagRow, enabled: boolean) {
  if (!selectedMerchantId.value) return

  row.updating = true
  try {
    const res = await updateFeatureFlag({
      merchant_id: selectedMerchantId.value,
      feature_name: row.feature_name,
      enabled
    }) as any
    if (res.code === 0 || res.code === 200) {
      ElMessage.success(`${row.label} 已${enabled ? "启用" : "禁用"}`)
      await loadFeatureFlags()
    } else {
      ElMessage.error(res.message || "操作失败")
    }
  } catch (error) {
    ElMessage.error("操作失败")
    console.error(error)
  } finally {
    row.updating = false
  }
}

// 全部启用
async function enableAll() {
  if (!selectedMerchantId.value) return

  try {
    await ElMessageBox.confirm("确定要启用所有功能吗？", "确认", {
      type: "warning"
    })
  } catch {
    return
  }

  batchLoading.value = true
  try {
    const res = await batchUpdateFeatureFlags({
      merchant_id: selectedMerchantId.value,
      features: featureFlags.value.map(f => ({
        feature_name: f.feature_name,
        enabled: true
      }))
    }) as any
    if (res.code === 0 || res.code === 200) {
      ElMessage.success("已全部启用")
      await loadFeatureFlags()
    } else {
      ElMessage.error(res.message || "操作失败")
    }
  } catch (error) {
    ElMessage.error("操作失败")
    console.error(error)
  } finally {
    batchLoading.value = false
  }
}

// 全部禁用
async function disableAll() {
  if (!selectedMerchantId.value) return

  try {
    await ElMessageBox.confirm("确定要禁用所有功能吗？这将影响该商户的所有用户！", "警告", {
      type: "error"
    })
  } catch {
    return
  }

  batchLoading.value = true
  try {
    const res = await batchUpdateFeatureFlags({
      merchant_id: selectedMerchantId.value,
      features: featureFlags.value.map(f => ({
        feature_name: f.feature_name,
        enabled: false
      }))
    }) as any
    if (res.code === 0 || res.code === 200) {
      ElMessage.success("已全部禁用")
      await loadFeatureFlags()
    } else {
      ElMessage.error(res.message || "操作失败")
    }
  } catch (error) {
    ElMessage.error("操作失败")
    console.error(error)
  } finally {
    batchLoading.value = false
  }
}

// 获取分类标签类型
function getCategoryType(category: string): "success" | "warning" | "info" | "danger" | "primary" {
  const types: Record<string, "success" | "warning" | "info" | "danger" | "primary"> = {
    "支付": "danger",
    "通讯": "success",
    "社交": "warning",
    "安全": "danger",
    "工具": "primary",
    "娱乐": "info",
    "服务": "warning"
  }
  return types[category] || "primary"
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

.ml-2 {
  margin-left: 8px;
}

.text-xs {
  font-size: 12px;
}

.text-gray-400 {
  color: #9ca3af;
}
</style>
