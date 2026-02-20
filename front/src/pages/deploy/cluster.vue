<script lang="ts" setup>
import type { DeployStep, DeployTSDDResp, NodeRole, ServerResp } from "@@/apis/deploy/type"
import { deployNode, getServerList, batchHealthCheck } from "@@/apis/deploy"
import { getMerchantList } from "@@/apis/merchant"

defineOptions({
  name: "DeployCluster"
})

// ========== 服务器列表 ==========
const serverList = ref<ServerResp[]>([])
const serverLoading = ref(false)

async function fetchServers() {
  serverLoading.value = true
  try {
    const res = await getServerList({ page: 1, size: 200 })
    serverList.value = res.data.list || []
  } finally {
    serverLoading.value = false
  }
}

// ========== 商户列表 ==========
const merchantList = ref<{ id: number; name: string }[]>([])
async function fetchMerchants() {
  const res = await getMerchantList({ page: 1, size: 1000 })
  merchantList.value = (res.data.list || []).map((m: any) => ({ id: m.id, name: m.name }))
}

// ========== 健康检查 ==========
const healthLoading = ref(false)
const healthResults = ref<Record<number, string>>({}) // server_id -> status

async function checkAllHealth() {
  if (serverList.value.length === 0) return
  healthLoading.value = true
  try {
    const ids = serverList.value.map(s => s.id)
    const res = await batchHealthCheck({ server_ids: ids })
    const results: Record<number, string> = {}
    for (const r of res.data.results || []) {
      results[r.server_id] = r.status
    }
    healthResults.value = results
    ElMessage.success("健康检查完成")
  } catch (e: any) {
    ElMessage.error(e.message || "健康检查失败")
  } finally {
    healthLoading.value = false
  }
}

function healthTag(serverId: number) {
  const s = healthResults.value[serverId]
  if (!s) return { type: "info" as const, text: "未检查" }
  if (s === "healthy") return { type: "success" as const, text: "健康" }
  if (s === "partial") return { type: "warning" as const, text: "部分异常" }
  return { type: "danger" as const, text: "异常" }
}

// ========== 部署对话框 ==========
const deployDialogVisible = ref(false)
const deploying = ref(false)
const deployForm = ref({
  server_id: undefined as number | undefined,
  merchant_id: undefined as number | undefined,
  node_role: "allinone" as NodeRole,
  force_reset: false,
  db_host: "",
  wk_node_id: undefined as number | undefined,
  wk_seed_node: ""
})

// 部署结果
const deployResult = ref<DeployTSDDResp | null>(null)
const resultDialogVisible = ref(false)

function openDeployDialog() {
  deployForm.value = {
    server_id: undefined,
    merchant_id: undefined,
    node_role: "allinone",
    force_reset: false,
    db_host: "",
    wk_node_id: undefined,
    wk_seed_node: ""
  }
  deployResult.value = null
  deployDialogVisible.value = true
}

async function submitDeploy() {
  const form = deployForm.value
  if (!form.server_id) {
    ElMessage.warning("请选择目标服务器")
    return
  }
  if (!form.merchant_id) {
    ElMessage.warning("请选择商户")
    return
  }
  if (form.node_role === "app" && !form.db_host) {
    ElMessage.warning("App 节点必须指定 DB 节点内网 IP")
    return
  }

  deploying.value = true
  try {
    const data: any = {
      server_id: form.server_id,
      merchant_id: form.merchant_id,
      node_role: form.node_role,
      force_reset: form.force_reset
    }
    if (form.db_host) data.db_host = form.db_host
    if (form.wk_node_id) data.wk_node_id = form.wk_node_id
    if (form.wk_seed_node) data.wk_seed_node = form.wk_seed_node

    const res = await deployNode(data)
    deployResult.value = res.data
    deployDialogVisible.value = false
    resultDialogVisible.value = true

    if (res.data.success) {
      ElMessage.success("部署成功")
      fetchServers()
    } else {
      ElMessage.error(res.data.message || "部署失败")
    }
  } catch (e: any) {
    ElMessage.error(e.message || "部署请求失败")
  } finally {
    deploying.value = false
  }
}

// 步骤状态图标
function stepIcon(status: string) {
  if (status === "success") return "SuccessFilled"
  if (status === "failed") return "CircleCloseFilled"
  if (status === "warning") return "WarningFilled"
  if (status === "running") return "Loading"
  return "Clock"
}

function stepColor(status: string) {
  if (status === "success") return "#67c23a"
  if (status === "failed") return "#f56c6c"
  if (status === "warning") return "#e6a23c"
  if (status === "running") return "#409eff"
  return "#909399"
}

// 角色标签
function roleLabel(role: string) {
  if (role === "db") return "DB 节点"
  if (role === "app") return "App 节点"
  return "全量节点"
}

function roleTagType(role: string) {
  if (role === "db") return "warning"
  if (role === "app") return "success"
  return "primary"
}

// ========== 初始化 ==========
onMounted(() => {
  fetchServers()
  fetchMerchants()
})
</script>

<template>
  <div class="app-container">
    <!-- 顶部操作栏 -->
    <el-card shadow="never" class="mb-4">
      <div style="display: flex; justify-content: space-between; align-items: center;">
        <div>
          <h3 style="margin: 0;">集群部署管理</h3>
          <p style="color: #909399; margin: 4px 0 0; font-size: 13px;">
            管理集群节点的部署、健康检查和扩容操作
          </p>
        </div>
        <div>
          <el-button type="info" :loading="healthLoading" @click="checkAllHealth">
            <el-icon><Monitor /></el-icon>&nbsp;健康检查
          </el-button>
          <el-button type="primary" @click="openDeployDialog">
            <el-icon><Plus /></el-icon>&nbsp;部署节点
          </el-button>
        </div>
      </div>
    </el-card>

    <!-- 服务器列表 -->
    <el-card shadow="never">
      <el-table :data="serverList" v-loading="serverLoading" stripe border style="width: 100%">
        <el-table-column prop="id" label="ID" width="60" />
        <el-table-column prop="name" label="名称" min-width="140" />
        <el-table-column prop="host" label="IP" width="140" />
        <el-table-column prop="port" label="端口" width="70" />
        <el-table-column label="商户" width="120">
          <template #default="{ row }">
            {{ row.merchant_name || '-' }}
          </template>
        </el-table-column>
        <el-table-column label="健康状态" width="100" align="center">
          <template #default="{ row }">
            <el-tag :type="healthTag(row.id).type" size="small">
              {{ healthTag(row.id).text }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="180" align="center">
          <template #default="{ row }">
            <el-button size="small" type="primary" link @click="() => { deployForm.server_id = row.id; deployForm.merchant_id = row.merchant_id; openDeployDialog() }">
              部署
            </el-button>
            <el-button size="small" type="info" link @click="$router.push({ name: 'DeployControl', query: { server_id: row.id } })">
              管理
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- 部署对话框 -->
    <el-dialog v-model="deployDialogVisible" title="部署集群节点" width="560px" :close-on-click-modal="false">
      <el-form :model="deployForm" label-width="120px" label-position="right">
        <el-form-item label="目标服务器" required>
          <el-select v-model="deployForm.server_id" placeholder="选择服务器" filterable style="width: 100%">
            <el-option
              v-for="s in serverList"
              :key="s.id"
              :label="`${s.name} (${s.host})`"
              :value="s.id"
            />
          </el-select>
        </el-form-item>

        <el-form-item label="商户" required>
          <el-select v-model="deployForm.merchant_id" placeholder="选择商户" filterable style="width: 100%">
            <el-option
              v-for="m in merchantList"
              :key="m.id"
              :label="m.name"
              :value="m.id"
            />
          </el-select>
        </el-form-item>

        <el-form-item label="节点角色" required>
          <el-radio-group v-model="deployForm.node_role">
            <el-radio-button value="allinone">全量 (All-in-One)</el-radio-button>
            <el-radio-button value="db">DB 节点</el-radio-button>
            <el-radio-button value="app">App 节点</el-radio-button>
          </el-radio-group>
          <div style="color: #909399; font-size: 12px; margin-top: 4px;">
            <template v-if="deployForm.node_role === 'allinone'">所有服务部署在同一台机器</template>
            <template v-else-if="deployForm.node_role === 'db'">仅部署 MySQL + Redis + MinIO</template>
            <template v-else>部署 WuKongIM + tsdd-server，连接远程 DB</template>
          </div>
        </el-form-item>

        <!-- App 节点额外配置 -->
        <template v-if="deployForm.node_role === 'app'">
          <el-form-item label="DB 内网 IP" required>
            <el-input v-model="deployForm.db_host" placeholder="如 172.31.9.143" />
            <div style="color: #909399; font-size: 12px; margin-top: 2px;">DB 节点的内网 IP 地址</div>
          </el-form-item>
        </template>

        <!-- WuKongIM 集群配置（App 和 Allinone） -->
        <template v-if="deployForm.node_role !== 'db'">
          <el-form-item label="WK 节点 ID">
            <el-input-number v-model="deployForm.wk_node_id" :min="1001" :max="9999" placeholder="如 1001" style="width: 100%" />
            <div style="color: #909399; font-size: 12px; margin-top: 2px;">WuKongIM 集群节点 ID（留空则不启用集群）</div>
          </el-form-item>

          <el-form-item label="种子节点">
            <el-input v-model="deployForm.wk_seed_node" placeholder="如 1001@172.31.0.1:11110" />
            <div style="color: #909399; font-size: 12px; margin-top: 2px;">加入已有集群时填写，首个节点留空</div>
          </el-form-item>
        </template>

        <el-form-item label="强制重置">
          <el-switch v-model="deployForm.force_reset" />
          <span style="color: #f56c6c; font-size: 12px; margin-left: 8px;" v-if="deployForm.force_reset">
            将删除现有容器和数据!
          </span>
        </el-form-item>
      </el-form>

      <template #footer>
        <el-button @click="deployDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="deploying" @click="submitDeploy">
          {{ deploying ? '部署中...' : '开始部署' }}
        </el-button>
      </template>
    </el-dialog>

    <!-- 部署结果对话框 -->
    <el-dialog v-model="resultDialogVisible" title="部署结果" width="700px">
      <template v-if="deployResult">
        <el-result
          :icon="deployResult.success ? 'success' : 'error'"
          :title="deployResult.success ? '部署成功' : '部署失败'"
          :sub-title="deployResult.message"
        />

        <!-- 部署步骤详情 -->
        <div v-if="deployResult.steps && deployResult.steps.length" style="margin-top: 16px;">
          <h4>部署步骤</h4>
          <el-timeline>
            <el-timeline-item
              v-for="(step, idx) in deployResult.steps"
              :key="idx"
              :color="stepColor(step.status)"
              :icon="stepIcon(step.status) as any"
            >
              <div style="display: flex; align-items: center; gap: 8px;">
                <strong>{{ step.name }}</strong>
                <el-tag :type="step.status === 'success' ? 'success' : step.status === 'failed' ? 'danger' : step.status === 'warning' ? 'warning' : 'info'" size="small">
                  {{ step.status }}
                </el-tag>
              </div>
              <div v-if="step.message" style="color: #606266; margin-top: 4px;">{{ step.message }}</div>
              <el-collapse v-if="step.output" style="margin-top: 4px;">
                <el-collapse-item title="查看输出">
                  <pre style="background: #f5f7fa; padding: 8px; border-radius: 4px; font-size: 12px; max-height: 200px; overflow: auto; white-space: pre-wrap;">{{ step.output }}</pre>
                </el-collapse-item>
              </el-collapse>
            </el-timeline-item>
          </el-timeline>
        </div>

        <!-- 部署后的 URL -->
        <div v-if="deployResult.api_url" style="margin-top: 16px; padding: 12px; background: #f0f9eb; border-radius: 4px;">
          <div><strong>API:</strong> {{ deployResult.api_url }}</div>
          <div v-if="deployResult.web_url"><strong>Web:</strong> {{ deployResult.web_url }}</div>
          <div v-if="deployResult.admin_url"><strong>Admin:</strong> {{ deployResult.admin_url }}</div>
        </div>
      </template>

      <template #footer>
        <el-button @click="resultDialogVisible = false">关闭</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.mb-4 {
  margin-bottom: 16px;
}
</style>
