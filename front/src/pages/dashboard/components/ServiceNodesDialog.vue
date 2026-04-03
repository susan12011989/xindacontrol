<script lang="ts" setup>
defineOptions({ name: "ServiceNodesDialog" })

import type { ServiceNode, ServiceNodeReq, ServiceNodeRole } from "../apis/type"
import {
  listServiceNodesApi,
  createServiceNodeApi,
  updateServiceNodeApi,
  deleteServiceNodeApi,
  switchToClusterModeApi
} from "../apis"

const props = defineProps<{
  visible: boolean
  merchantId: number
  merchantName: string
}>()

const emit = defineEmits<{
  (e: "update:visible", value: boolean): void
  (e: "updated"): void
}>()

const dialogVisible = computed({
  get: () => props.visible,
  set: value => emit("update:visible", value)
})

const loading = ref(false)
const nodes = ref<ServiceNode[]>([])

// 角色标签映射
const roleLabels: Record<string, string> = {
  all: "All-in-One",
  im: "IM (WuKongIM)",
  api: "API (tsdd-server)",
  minio: "MinIO",
  web: "Web"
}

const roleColors: Record<string, string> = {
  all: "",
  im: "success",
  api: "warning",
  minio: "info",
  web: "danger"
}

// 当前是否为多机模式
const isCluster = computed(() => {
  return nodes.value.length > 0 && !nodes.value.some(n => n.role === "all")
})

// 加载节点列表
async function loadNodes() {
  if (!props.merchantId) return
  loading.value = true
  try {
    const { data } = await listServiceNodesApi(props.merchantId)
    nodes.value = data || []
  } finally {
    loading.value = false
  }
}

watch(() => props.visible, (val) => {
  if (val && props.merchantId) loadNodes()
})

// ========== 添加节点 ==========
const addDialogVisible = ref(false)
const addForm = ref<ServiceNodeReq>({
  role: "im" as ServiceNodeRole,
  host: "",
  is_primary: 0,
  remark: ""
})

function showAddDialog() {
  addForm.value = { role: "im" as ServiceNodeRole, host: "", is_primary: 0, remark: "" }
  addDialogVisible.value = true
}

async function handleAdd() {
  if (!addForm.value.host) {
    ElMessage.warning("请填写服务器地址")
    return
  }
  try {
    await createServiceNodeApi(props.merchantId, addForm.value)
    ElMessage.success("添加成功")
    addDialogVisible.value = false
    loadNodes()
    emit("updated")
  } catch (e: any) {
    ElMessage.error(e?.message || "添加失败")
  }
}

// ========== 编辑节点 ==========
const editDialogVisible = ref(false)
const editForm = ref<ServiceNodeReq & { id: number }>({
  id: 0,
  role: "im" as ServiceNodeRole,
  host: "",
  is_primary: 0,
  remark: ""
})

function showEditDialog(node: ServiceNode) {
  editForm.value = {
    id: node.id,
    role: node.role,
    host: node.host,
    is_primary: node.is_primary,
    remark: node.remark
  }
  editDialogVisible.value = true
}

async function handleEdit() {
  try {
    await updateServiceNodeApi(editForm.value.id, editForm.value)
    ElMessage.success("更新成功")
    editDialogVisible.value = false
    loadNodes()
    emit("updated")
  } catch (e: any) {
    ElMessage.error(e?.message || "更新失败")
  }
}

// ========== 删除节点 ==========
async function handleDelete(node: ServiceNode) {
  try {
    await ElMessageBox.confirm(
      `确认删除 ${roleLabels[node.role] || node.role} 节点（${node.host}）？`,
      "删除确认",
      { type: "warning" }
    )
    await deleteServiceNodeApi(node.id)
    ElMessage.success("删除成功")
    loadNodes()
    emit("updated")
  } catch {}
}

// ========== 切换到多机模式 ==========
const switchDialogVisible = ref(false)
const clusterNodes = ref<ServiceNodeReq[]>([
  { role: "im" as ServiceNodeRole, host: "", is_primary: 1 },
  { role: "api" as ServiceNodeRole, host: "" },
  { role: "minio" as ServiceNodeRole, host: "" }
])

function showSwitchDialog() {
  // 预填当前 all-in-one 的地址
  const currentHost = nodes.value.find(n => n.role === "all")?.host || ""
  clusterNodes.value = [
    { role: "im" as ServiceNodeRole, host: currentHost, is_primary: 1 },
    { role: "api" as ServiceNodeRole, host: currentHost },
    { role: "minio" as ServiceNodeRole, host: currentHost }
  ]
  switchDialogVisible.value = true
}

function addClusterNode() {
  clusterNodes.value.push({ role: "web" as ServiceNodeRole, host: "" })
}

function removeClusterNode(index: number) {
  clusterNodes.value.splice(index, 1)
}

async function handleSwitch() {
  const validNodes = clusterNodes.value.filter(n => n.host)
  if (validNodes.length < 2) {
    ElMessage.warning("至少需要 2 个有效节点")
    return
  }
  try {
    await ElMessageBox.confirm(
      "切换到多机模式后，nginx 配置将按角色分发到不同服务器。确认继续？",
      "模式切换",
      { type: "warning" }
    )
    await switchToClusterModeApi(props.merchantId, { nodes: validNodes })
    ElMessage.success("已切换到多机模式")
    switchDialogVisible.value = false
    loadNodes()
    emit("updated")
  } catch {}
}

const clusterRoleOptions: { value: ServiceNodeRole; label: string }[] = [
  { value: "im", label: "IM (WuKongIM)" },
  { value: "api", label: "API (tsdd-server)" },
  { value: "minio", label: "MinIO" },
  { value: "web", label: "Web" }
]
</script>

<template>
  <el-dialog v-model="dialogVisible" :title="`${merchantName} - 服务节点`" width="700px" destroy-on-close>
    <div class="mb-4 flex items-center justify-between">
      <div>
        <el-tag v-if="isCluster" type="warning">多机模式</el-tag>
        <el-tag v-else>单机模式</el-tag>
        <span class="ml-2 text-sm text-gray-500">{{ nodes.length }} 个节点</span>
      </div>
      <div>
        <el-button v-if="!isCluster && nodes.length > 0" type="warning" size="small" @click="showSwitchDialog">
          切换到多机模式
        </el-button>
        <el-button v-if="isCluster" type="primary" size="small" @click="showAddDialog">
          添加节点
        </el-button>
      </div>
    </div>

    <el-table :data="nodes" v-loading="loading" border size="small">
      <el-table-column label="角色" width="150">
        <template #default="{ row }">
          <el-tag :type="(roleColors[row.role] as any) || ''">{{ roleLabels[row.role] || row.role }}</el-tag>
          <el-tag v-if="row.is_primary" type="danger" size="small" class="ml-1">主</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="host" label="地址" />
      <el-table-column prop="remark" label="备注" width="120" />
      <el-table-column label="操作" width="130" fixed="right">
        <template #default="{ row }">
          <el-button type="primary" link size="small" @click="showEditDialog(row)">编辑</el-button>
          <el-button type="danger" link size="small" @click="handleDelete(row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>
  </el-dialog>

  <!-- 添加节点 -->
  <el-dialog v-model="addDialogVisible" title="添加服务节点" width="450px" append-to-body>
    <el-form :model="addForm" label-width="80px">
      <el-form-item label="角色">
        <el-select v-model="addForm.role">
          <el-option v-for="opt in clusterRoleOptions" :key="opt.value" :value="opt.value" :label="opt.label" />
        </el-select>
      </el-form-item>
      <el-form-item label="地址">
        <el-input v-model="addForm.host" placeholder="服务器 IP 或内网地址" />
      </el-form-item>
      <el-form-item label="主节点">
        <el-switch v-model="addForm.is_primary" :active-value="1" :inactive-value="0" />
      </el-form-item>
      <el-form-item label="备注">
        <el-input v-model="addForm.remark" />
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="addDialogVisible = false">取消</el-button>
      <el-button type="primary" @click="handleAdd">确认</el-button>
    </template>
  </el-dialog>

  <!-- 编辑节点 -->
  <el-dialog v-model="editDialogVisible" title="编辑服务节点" width="450px" append-to-body>
    <el-form :model="editForm" label-width="80px">
      <el-form-item label="角色">
        <el-tag>{{ roleLabels[editForm.role] || editForm.role }}</el-tag>
      </el-form-item>
      <el-form-item label="地址">
        <el-input v-model="editForm.host" placeholder="服务器 IP 或内网地址" />
      </el-form-item>
      <el-form-item label="主节点">
        <el-switch v-model="editForm.is_primary" :active-value="1" :inactive-value="0" />
      </el-form-item>
      <el-form-item label="备注">
        <el-input v-model="editForm.remark" />
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="editDialogVisible = false">取消</el-button>
      <el-button type="primary" @click="handleEdit">确认</el-button>
    </template>
  </el-dialog>

  <!-- 切换到多机模式 -->
  <el-dialog v-model="switchDialogVisible" title="切换到多机模式" width="600px" append-to-body>
    <el-alert type="warning" :closable="false" class="mb-4">
      切换后，各服务将按角色分发到不同服务器。nginx 会自动重新配置。
    </el-alert>
    <div v-for="(node, index) in clusterNodes" :key="index" class="mb-3 flex items-center gap-2">
      <el-select v-model="node.role" style="width: 180px">
        <el-option v-for="opt in clusterRoleOptions" :key="opt.value" :value="opt.value" :label="opt.label" />
      </el-select>
      <el-input v-model="node.host" placeholder="服务器地址" style="flex: 1" />
      <el-checkbox v-model="node.is_primary" :true-value="1" :false-value="0" label="主" />
      <el-button type="danger" link @click="removeClusterNode(index)">
        <el-icon><Delete /></el-icon>
      </el-button>
    </div>
    <el-button type="primary" link @click="addClusterNode">+ 添加节点</el-button>
    <template #footer>
      <el-button @click="switchDialogVisible = false">取消</el-button>
      <el-button type="warning" @click="handleSwitch">确认切换</el-button>
    </template>
  </el-dialog>
</template>
