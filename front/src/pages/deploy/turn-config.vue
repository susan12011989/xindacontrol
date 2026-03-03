<script lang="ts" setup>
import { ref, reactive, onMounted, computed } from "vue"
import type { MerchantTurnConfigItem, BatchTurnUpdateResult } from "@@/apis/merchant/type"
import { listMerchantTurnConfigs, updateMerchantTurnServer, batchUpdateTurnServer } from "@@/apis/merchant"
import { ElMessage, ElMessageBox } from "element-plus"

defineOptions({ name: "TurnConfig" })

// 商户列表
const merchants = ref<MerchantTurnConfigItem[]>([])
const loading = ref(false)
const searchName = ref("")

// 批量选择
const selectedRows = ref<MerchantTurnConfigItem[]>([])
const selectedIds = computed(() => selectedRows.value.map((r) => r.merchant_id))

// 批量操作
const batchForm = reactive({
  turnServer: "",
  turnUsername: "",
  turnCredential: ""
})
const batchLoading = ref(false)

// 操作结果
const updateResults = ref<BatchTurnUpdateResult[]>([])
const showResults = ref(false)

// 编辑弹窗
const editDialogVisible = ref(false)
const editLoading = ref(false)
const editForm = reactive({
  merchantId: 0,
  merchantName: "",
  turnServer: "",
  turnUsername: "",
  turnCredential: ""
})

// 加载数据
async function loadData() {
  loading.value = true
  try {
    const res = await listMerchantTurnConfigs(searchName.value ? { name: searchName.value } : undefined)
    merchants.value = res.data
  } catch (e: any) {
    ElMessage.error(e.message || "加载失败")
  } finally {
    loading.value = false
  }
}

// 表格选择变化
function handleSelectionChange(selection: MerchantTurnConfigItem[]) {
  selectedRows.value = selection
}

// 打开编辑弹窗
function handleEdit(row: MerchantTurnConfigItem) {
  editForm.merchantId = row.merchant_id
  editForm.merchantName = row.merchant_name
  editForm.turnServer = row.turn_server || ""
  editForm.turnUsername = row.turn_username || ""
  editForm.turnCredential = row.turn_credential || ""
  editDialogVisible.value = true
}

// 提交单个编辑
async function submitEdit() {
  if (!editForm.turnServer.trim()) {
    ElMessage.warning("请输入 TURN 服务器地址")
    return
  }
  editLoading.value = true
  try {
    await updateMerchantTurnServer(editForm.merchantId, {
      turn_server: editForm.turnServer.trim(),
      turn_username: editForm.turnUsername.trim(),
      turn_credential: editForm.turnCredential.trim()
    })
    ElMessage.success(`${editForm.merchantName} TURN 更新成功`)
    editDialogVisible.value = false
    loadData()
  } catch (e: any) {
    ElMessage.error(e.message || "更新失败")
  } finally {
    editLoading.value = false
  }
}

// 批量更新
async function handleBatchUpdate() {
  if (selectedIds.value.length === 0) {
    ElMessage.warning("请先选择商户")
    return
  }
  if (!batchForm.turnServer.trim()) {
    ElMessage.warning("请输入 TURN 服务器地址")
    return
  }

  try {
    await ElMessageBox.confirm(
      `确定要对 ${selectedIds.value.length} 个商户批量设置 TURN 配置吗？\n地址: ${batchForm.turnServer.trim()}\n用户: ${batchForm.turnUsername || "(空)"}\n密码: ${batchForm.turnCredential ? "***" : "(空)"}`,
      "批量更新确认",
      { type: "warning" }
    )
  } catch {
    return
  }

  batchLoading.value = true
  showResults.value = false
  try {
    const res = await batchUpdateTurnServer({
      merchant_ids: selectedIds.value,
      turn_server: batchForm.turnServer.trim(),
      turn_username: batchForm.turnUsername.trim(),
      turn_credential: batchForm.turnCredential.trim()
    })
    updateResults.value = res.data.results
    showResults.value = true
    ElMessage.success(`批量更新完成：成功 ${res.data.success_count}，失败 ${res.data.fail_count}`)
    loadData()
  } catch (e: any) {
    ElMessage.error(e.message || "批量更新失败")
  } finally {
    batchLoading.value = false
  }
}

// TURN 配置状态
function turnStatusText(row: MerchantTurnConfigItem) {
  if (!row.turn_server) return "未配置"
  if (row.turn_username && row.turn_credential) return "完整"
  return "缺凭据"
}
function turnStatusType(row: MerchantTurnConfigItem) {
  if (!row.turn_server) return "info"
  if (row.turn_username && row.turn_credential) return "success"
  return "warning"
}

// 状态文字
function statusText(status: number) {
  return status === 1 ? "正常" : "禁用"
}
function statusType(status: number) {
  return status === 1 ? "success" : "danger"
}

onMounted(() => loadData())
</script>

<template>
  <div class="app-container">
    <!-- 操作区 -->
    <el-card shadow="never" style="margin-bottom: 16px">
      <div style="display: flex; align-items: center; gap: 12px; flex-wrap: wrap">
        <el-input
          v-model="searchName"
          placeholder="搜索商户名称"
          clearable
          style="width: 200px"
          @clear="loadData"
          @keyup.enter="loadData"
        />
        <el-button type="primary" @click="loadData" :loading="loading">查询</el-button>
        <el-divider direction="vertical" />
        <span style="color: #909399; font-size: 13px">
          已选择 <b style="color: #409eff">{{ selectedIds.length }}</b> / {{ merchants.length }} 个商户
        </span>
      </div>
      <!-- 批量设置区 -->
      <div style="display: flex; align-items: center; gap: 10px; margin-top: 12px; flex-wrap: wrap">
        <el-input v-model="batchForm.turnServer" placeholder="TURN 地址 (ip:port)" style="width: 200px" />
        <el-input v-model="batchForm.turnUsername" placeholder="用户名" style="width: 160px" />
        <el-input
          v-model="batchForm.turnCredential"
          placeholder="密码"
          type="password"
          show-password
          style="width: 160px"
        />
        <el-button
          type="warning"
          :disabled="selectedIds.length === 0 || !batchForm.turnServer.trim()"
          :loading="batchLoading"
          @click="handleBatchUpdate"
        >
          批量更新 ({{ selectedIds.length }})
        </el-button>
      </div>
    </el-card>

    <!-- 商户列表 -->
    <el-card shadow="never">
      <el-table
        :data="merchants"
        v-loading="loading"
        stripe
        border
        @selection-change="handleSelectionChange"
        style="width: 100%"
      >
        <el-table-column type="selection" width="50" />
        <el-table-column prop="merchant_no" label="商户编号" width="110" />
        <el-table-column prop="merchant_name" label="商户名称" min-width="130" show-overflow-tooltip />
        <el-table-column prop="server_ip" label="服务器IP" width="140" />
        <el-table-column label="TURN 地址" min-width="170">
          <template #default="{ row }">
            <span v-if="row.turn_server" style="color: #67c23a; font-family: monospace">
              {{ row.turn_server }}
            </span>
            <span v-else style="color: #c0c4cc">未配置</span>
          </template>
        </el-table-column>
        <el-table-column label="TURN 凭据" width="90" align="center">
          <template #default="{ row }">
            <el-tag :type="turnStatusType(row)" size="small">{{ turnStatusText(row) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="商户状态" width="90" align="center">
          <template #default="{ row }">
            <el-tag :type="statusType(row.status)" size="small">{{ statusText(row.status) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="updated_at" label="更新时间" width="170" />
        <el-table-column label="操作" width="80" fixed="right" align="center">
          <template #default="{ row }">
            <el-button type="primary" link size="small" @click="handleEdit(row)">编辑</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- 操作结果 -->
    <el-card v-if="showResults" shadow="never" style="margin-top: 16px">
      <template #header>
        <div style="display: flex; justify-content: space-between; align-items: center">
          <span>更新结果</span>
          <el-button type="info" link @click="showResults = false">关闭</el-button>
        </div>
      </template>
      <el-table :data="updateResults" stripe border size="small">
        <el-table-column prop="merchant_name" label="商户名称" min-width="140" />
        <el-table-column prop="server_ip" label="服务器IP" width="150" />
        <el-table-column label="结果" width="80" align="center">
          <template #default="{ row }">
            <el-tag :type="row.success ? 'success' : 'danger'" size="small">
              {{ row.success ? "成功" : "失败" }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="message" label="详情" min-width="300" show-overflow-tooltip />
      </el-table>
    </el-card>

    <!-- 编辑弹窗 -->
    <el-dialog v-model="editDialogVisible" title="编辑 TURN 配置" width="480px" destroy-on-close>
      <el-form label-width="90px">
        <el-form-item label="商户">
          <span>{{ editForm.merchantName }}</span>
        </el-form-item>
        <el-form-item label="TURN 地址">
          <el-input
            v-model="editForm.turnServer"
            placeholder="格式: ip:port (例: 47.83.26.53:3478)"
          />
        </el-form-item>
        <el-form-item label="用户名">
          <el-input v-model="editForm.turnUsername" placeholder="TURN 认证用户名" />
        </el-form-item>
        <el-form-item label="密码">
          <el-input
            v-model="editForm.turnCredential"
            placeholder="TURN 认证密码"
            type="password"
            show-password
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="editDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="editLoading" @click="submitEdit">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>
