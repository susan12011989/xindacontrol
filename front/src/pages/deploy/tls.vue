<script lang="ts" setup>
import type { MerchantResp } from "@@/apis/merchant/type"
import type { TlsCertificateResp, TlsStatusResp } from "@@/apis/deploy/type"
import { getMerchantList } from "@@/apis/merchant"
import { generateTlsCerts, getTlsCerts, disableTlsCerts, getTlsCertFingerprint, getTlsStatus, verifyTlsStatus, batchUpgradeTls, batchRollbackTls } from "@@/apis/deploy"

defineOptions({
  name: "DeployTls"
})

// ========== 商户选择 ==========
const merchantList = ref<MerchantResp[]>([])
const selectedMerchantId = ref<number | undefined>(undefined)

const selectedMerchantName = computed(() => {
  const m = merchantList.value.find(m => m.id === selectedMerchantId.value)
  return m?.name || ""
})

// ========== 证书状态 ==========
const loading = ref(false)
const certs = ref<TlsCertificateResp[]>([])
const fingerprint = ref("")
const fingerprintExpires = ref("")
const tlsStatus = ref<TlsStatusResp | null>(null)

// 切换商户
async function onMerchantChange() {
  certs.value = []
  fingerprint.value = ""
  fingerprintExpires.value = ""
  tlsStatus.value = null

  if (!selectedMerchantId.value) return
  await Promise.all([loadCerts(), loadFingerprint(), loadTlsStatus()])
}

// 加载证书
async function loadCerts() {
  if (!selectedMerchantId.value) return
  loading.value = true
  try {
    const res = await getTlsCerts(selectedMerchantId.value)
    // 后端返回 { ca: TlsCertificateResp, server: TlsCertificateResp }
    const { ca, server } = res.data || {}
    certs.value = [ca, server].filter(Boolean)
  } catch {
    certs.value = []
  } finally {
    loading.value = false
  }
}

// 加载指纹
async function loadFingerprint() {
  if (!selectedMerchantId.value) return
  try {
    const res = await getTlsCertFingerprint(selectedMerchantId.value)
    fingerprint.value = res.data.fingerprint || ""
    fingerprintExpires.value = res.data.expires_at || ""
  } catch {
    fingerprint.value = ""
  }
}

// 加载 TLS 状态
async function loadTlsStatus() {
  if (!selectedMerchantId.value) return
  try {
    const res = await getTlsStatus(selectedMerchantId.value)
    tlsStatus.value = res.data
  } catch {
    tlsStatus.value = null
  }
}

// 生成证书
async function handleGenerate() {
  if (!selectedMerchantId.value) return
  ElMessageBox.confirm(
    `将为商户「${selectedMerchantName.value}」生成新的 CA 根证书和服务器证书。已部署的服务器需要重新升级 TLS。确定继续？`,
    "生成证书",
    { type: "warning" }
  ).then(async () => {
    loading.value = true
    try {
      await generateTlsCerts({ merchant_id: selectedMerchantId.value! })
      ElMessage.success("证书生成成功")
      await loadCerts()
      await loadFingerprint()
    } catch (e: any) {
      ElMessage.error(e.message || "证书生成失败")
    } finally {
      loading.value = false
    }
  })
}

// 停用证书
async function handleDisable() {
  if (!selectedMerchantId.value) return
  ElMessageBox.confirm(
    `停用商户「${selectedMerchantName.value}」的证书后，新部署的服务器将不会自动启用 TLS。确定停用？`,
    "停用证书",
    { type: "warning" }
  ).then(async () => {
    loading.value = true
    try {
      await disableTlsCerts({ merchant_id: selectedMerchantId.value! })
      ElMessage.success("证书已停用")
      await loadCerts()
      fingerprint.value = ""
    } catch (e: any) {
      ElMessage.error(e.message || "停用失败")
    } finally {
      loading.value = false
    }
  })
}

// 复制指纹
function copyFingerprint() {
  if (!fingerprint.value) return
  navigator.clipboard.writeText(fingerprint.value).then(() => {
    ElMessage.success("指纹已复制到剪贴板")
  })
}

// 批量升级 TLS
async function handleUpgrade() {
  if (!selectedMerchantId.value) return
  ElMessageBox.confirm(
    `将商户「${selectedMerchantName.value}」的所有 GOST 服务器升级为 TLS 模式？`,
    "批量升级",
    { type: "warning" }
  ).then(async () => {
    loading.value = true
    try {
      const res = await batchUpgradeTls({ merchant_id: selectedMerchantId.value! })
      const { success, failed, total } = res.data
      if (failed === 0) {
        ElMessage.success(`升级完成，全部成功 (${success}/${total})`)
      } else {
        ElMessage.warning(`升级完成，成功 ${success}，失败 ${failed}`)
      }
      await loadTlsStatus()
    } catch (e: any) {
      ElMessage.error(e.message || "升级失败")
    } finally {
      loading.value = false
    }
  })
}

// 批量回滚
async function handleRollback() {
  if (!selectedMerchantId.value) return
  ElMessageBox.confirm(
    `将商户「${selectedMerchantName.value}」的所有 GOST 服务器回滚为 TCP 模式？`,
    "批量回滚",
    { type: "warning" }
  ).then(async () => {
    loading.value = true
    try {
      const res = await batchRollbackTls({ merchant_id: selectedMerchantId.value! })
      const { success, failed, total } = res.data
      if (failed === 0) {
        ElMessage.success(`回滚完成，全部成功 (${success}/${total})`)
      } else {
        ElMessage.warning(`回滚完成，成功 ${success}，失败 ${failed}`)
      }
      await loadTlsStatus()
    } catch (e: any) {
      ElMessage.error(e.message || "回滚失败")
    } finally {
      loading.value = false
    }
  })
}

// 验证 TLS 连接
async function handleVerify() {
  if (!selectedMerchantId.value) return
  loading.value = true
  try {
    const res = await verifyTlsStatus({ merchant_id: selectedMerchantId.value! })
    tlsStatus.value = res.data
    ElMessage.success("验证完成")
  } catch (e: any) {
    ElMessage.error(e.message || "验证失败")
  } finally {
    loading.value = false
  }
}

// 初始化
onMounted(async () => {
  getMerchantList({ page: 1, size: 2000 }).then((res) => {
    merchantList.value = Array.isArray(res.data?.list) ? res.data.list : []
  })
})
</script>

<template>
  <div class="app-container">
    <!-- 商户选择 -->
    <el-card shadow="never" class="mb-4">
      <div class="flex items-center gap-3">
        <span class="font-bold text-base">选择商户</span>
        <el-select
          v-model="selectedMerchantId"
          placeholder="请选择商户"
          style="width: 300px"
          filterable
          @change="onMerchantChange"
        >
          <el-option
            v-for="m in merchantList"
            :key="m.id"
            :label="`${m.name} (${m.no})`"
            :value="m.id"
          />
        </el-select>
      </div>
    </el-card>

    <template v-if="selectedMerchantId">
      <!-- 证书管理 -->
      <el-card v-loading="loading" shadow="never">
        <template #header>
          <div class="card-header">
            <span class="font-bold text-base">TLS 证书 — {{ selectedMerchantName }}</span>
            <div>
              <el-button type="primary" @click="handleGenerate">生成证书</el-button>
              <el-button type="danger" :disabled="certs.length === 0" @click="handleDisable">停用证书</el-button>
            </div>
          </div>
        </template>

        <el-table v-if="certs.length > 0" :data="certs" border size="small">
          <el-table-column prop="name" label="名称" width="160" />
          <el-table-column label="类型" width="120">
            <template #default="{ row }">
              <el-tag :type="row.cert_type === 1 ? 'warning' : 'primary'" size="small">
                {{ row.cert_type === 1 ? 'CA 根证书' : '服务器证书' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="状态" width="80">
            <template #default="{ row }">
              <el-tag :type="row.status === 1 ? 'success' : 'info'" size="small">
                {{ row.status === 1 ? '启用' : '停用' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="fingerprint" label="指纹 (SHA-256)" show-overflow-tooltip />
          <el-table-column prop="expires_at" label="过期时间" width="160" />
          <el-table-column prop="created_at" label="创建时间" width="160" />
        </el-table>
        <el-empty v-else description="暂未生成证书，请点击「生成证书」" />
      </el-card>

      <!-- 证书指纹（App 端 Pinning） -->
      <el-card v-if="fingerprint" shadow="never" class="mt-4">
        <template #header>
          <span class="font-bold text-base">App 证书指纹 (Certificate Pinning)</span>
        </template>
        <el-descriptions :column="1" border size="small">
          <el-descriptions-item label="SHA-256 指纹">
            <code class="fingerprint-text">{{ fingerprint }}</code>
            <el-button type="primary" link size="small" style="margin-left: 8px" @click="copyFingerprint">复制</el-button>
          </el-descriptions-item>
          <el-descriptions-item label="证书过期时间">{{ fingerprintExpires }}</el-descriptions-item>
        </el-descriptions>
      </el-card>

      <!-- 系统服务器 TLS 状态 -->
      <el-card v-loading="loading" shadow="never" class="mt-4">
        <template #header>
          <div class="card-header">
            <span class="font-bold text-base">GOST 服务器 TLS 状态 — {{ selectedMerchantName }}</span>
            <div>
              <el-button @click="handleVerify">验证连接</el-button>
              <el-button type="warning" @click="handleRollback">批量回滚 TCP</el-button>
              <el-button type="success" :disabled="certs.length === 0" @click="handleUpgrade">批量升级 TLS</el-button>
            </div>
          </div>
        </template>

        <div v-if="tlsStatus">
          <el-descriptions :column="3" border size="small" class="mb-4">
            <el-descriptions-item label="总数">{{ tlsStatus.total }}</el-descriptions-item>
            <el-descriptions-item label="TLS">
              <el-tag type="success" size="small">{{ tlsStatus.tls_count }}</el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="TCP">
              <el-tag type="info" size="small">{{ tlsStatus.tcp_count }}</el-tag>
            </el-descriptions-item>
          </el-descriptions>

          <el-table :data="tlsStatus.servers" border size="small">
            <el-table-column prop="server_name" label="服务器" width="160" />
            <el-table-column prop="host" label="IP" width="140" />
            <el-table-column label="TLS" width="80">
              <template #default="{ row }">
                <el-tag :type="row.tls_enabled === 1 ? 'success' : 'info'" size="small">
                  {{ row.tls_enabled === 1 ? 'TLS' : 'TCP' }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column label="验证" width="120">
              <template #default="{ row }">
                <el-tag v-if="row.tls_verified" type="success" size="small">通过</el-tag>
                <el-tag v-else-if="row.verify_error" type="danger" size="small">失败</el-tag>
                <span v-else class="text-gray-400">-</span>
              </template>
            </el-table-column>
            <el-table-column prop="verify_error" label="错误详情" show-overflow-tooltip />
            <el-table-column prop="tls_deployed_at" label="部署时间" width="160" />
          </el-table>
        </div>
        <el-empty v-else description="该商户暂无关联的 GOST 服务器" />
      </el-card>
    </template>

    <!-- 未选商户提示 -->
    <el-empty v-else description="请先选择商户" />
  </div>
</template>

<style lang="scss" scoped>
.app-container {
  padding: 20px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.mt-4 {
  margin-top: 16px;
}

.mb-4 {
  margin-bottom: 16px;
}

.flex {
  display: flex;
}

.items-center {
  align-items: center;
}

.gap-3 {
  gap: 12px;
}

.fingerprint-text {
  font-family: monospace;
  font-size: 12px;
  color: #303133;
  word-break: break-all;
}
</style>
