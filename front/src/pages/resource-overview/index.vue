<script lang="ts" setup>
import type { GlobalOssConfigResp, GlobalGostServerResp, ResourceTagResp, OssHealthCheckResult } from "@@/apis/resource_overview/type"
import type { VxeGridInstance, VxeGridProps } from "vxe-table"
import {
  getGlobalOssConfigs,
  getGlobalGostServers,
  getTagList,
  createTag,
  updateTag,
  deleteTag,
  assignTags,
  removeTags,
  batchSyncGostIP,
  checkOssHealth
} from "@@/apis/resource_overview"
import { getMerchantList } from "@@/apis/merchant"
import { ElMessage, ElMessageBox } from "element-plus"
import { Loading } from "@element-plus/icons-vue"

defineOptions({ name: "ResourceOverview" })

// ========== 基础数据 ==========
const activeTab = ref("oss")
const merchantOptions = ref<{ label: string; value: number }[]>([])
const tagOptions = ref<ResourceTagResp[]>([])

function getCloudTypeLabel(type: string) {
  const map: Record<string, string> = { aws: "AWS S3", aliyun: "阿里云", tencent: "腾讯云" }
  return map[type] || type || "-"
}

function getCloudTypeTagType(type: string): "success" | "warning" | "info" | "primary" {
  const map: Record<string, "success" | "warning" | "info" | "primary"> = {
    aws: "success",
    aliyun: "warning",
    tencent: "info"
  }
  return map[type] || "info"
}

async function loadMerchants() {
  try {
    const res = await getMerchantList({ page: 1, size: 1000 })
    merchantOptions.value = res.data.list?.map((m: any) => ({ label: `${m.name} (${m.no})`, value: m.id })) || []
  } catch (e) {
    console.error("加载商户列表失败", e)
  }
}

async function loadTags() {
  try {
    const res = await getTagList()
    tagOptions.value = res.data || []
  } catch (e) {
    console.error("加载标签列表失败", e)
  }
}

onMounted(() => {
  loadMerchants()
  loadTags()
})

// ========== OSS 筛选 ==========
const ossFilter = reactive({
  merchant_id: undefined as number | undefined,
  cloud_type: "" as string,
  tag_id: undefined as number | undefined
})

function refreshOssGrid() {
  ossGridDom.value?.commitProxy("query")
}

function resetOssFilter() {
  ossFilter.merchant_id = undefined
  ossFilter.cloud_type = ""
  ossFilter.tag_id = undefined
  refreshOssGrid()
}

// ========== OSS 配置列表 ==========
const ossGridDom = ref<VxeGridInstance>()
const ossGridOpt: VxeGridProps = reactive({
  loading: true,
  autoResize: true,
  pagerConfig: { align: "right" },
  toolbarConfig: {
    refresh: true,
    custom: true,
    slots: { buttons: "oss-toolbar-btns" }
  },
  columns: [
    { type: "checkbox", width: "50px" },
    { type: "seq", width: "50px", title: "#" },
    { field: "merchant_name", title: "商户", width: 120 },
    { field: "name", title: "配置名称", width: 100 },
    { field: "cloud_type", title: "云类型", width: 90, slots: { default: "cloud-type-slot" } },
    { field: "bucket", title: "Bucket", width: 140, showOverflow: true },
    { field: "region", title: "区域", width: 100 },
    { field: "download_url", title: "下载地址", minWidth: 240, showOverflow: true, slots: { default: "download-url-slot" } },
    { field: "is_default", title: "默认", width: 60, slots: { default: "default-slot" } },
    { field: "status", title: "状态", width: 60, slots: { default: "status-slot" } },
    { field: "tags", title: "标签", width: 140, slots: { default: "tags-slot" } },
    { title: "操作", width: "80px", fixed: "right", slots: { default: "oss-row-operate" } }
  ],
  proxyConfig: {
    seq: true,
    autoLoad: true,
    props: { total: "total" },
    ajax: {
      query: ({ page }) => {
        ossGridOpt.loading = true
        return new Promise((resolve) => {
          getGlobalOssConfigs({
            merchant_id: ossFilter.merchant_id || undefined,
            cloud_type: ossFilter.cloud_type || undefined,
            tag_id: ossFilter.tag_id || undefined,
            size: page.pageSize,
            page: page.currentPage
          }).then((res) => {
            ossGridOpt.loading = false
            resolve({ total: res.data.total, result: res.data.list || [] })
          }).catch(() => { ossGridOpt.loading = false })
        })
      }
    }
  }
})

// ========== GOST 筛选 ==========
const gostFilter = reactive({
  merchant_id: undefined as number | undefined,
  tag_id: undefined as number | undefined
})

function refreshGostGrid() {
  gostGridDom.value?.commitProxy("query")
}

function resetGostFilter() {
  gostFilter.merchant_id = undefined
  gostFilter.tag_id = undefined
  refreshGostGrid()
}

// ========== GOST 服务器列表 ==========
const gostGridDom = ref<VxeGridInstance>()
const gostGridOpt: VxeGridProps = reactive({
  loading: true,
  autoResize: true,
  pagerConfig: { align: "right" },
  toolbarConfig: {
    refresh: true,
    custom: true,
    slots: { buttons: "gost-toolbar-btns" }
  },
  columns: [
    { type: "checkbox", width: "50px" },
    { type: "seq", width: "50px", title: "#" },
    { field: "merchant_name", title: "商户", width: 130 },
    { field: "server_name", title: "服务器名", width: 120 },
    { field: "server_host", title: "服务器IP", width: 140 },
    { field: "region", title: "区域", width: 120 },
    { field: "listen_port", title: "监听端口", width: 90 },
    { field: "is_primary", title: "主服务器", width: 80, slots: { default: "primary-slot" } },
    { field: "status", title: "状态", width: 70, slots: { default: "status-slot" } },
    { field: "tags", title: "标签", width: 160, slots: { default: "tags-slot" } },
    { title: "操作", width: "100px", fixed: "right", slots: { default: "gost-row-operate" } }
  ],
  proxyConfig: {
    seq: true,
    autoLoad: true,
    props: { total: "total" },
    ajax: {
      query: ({ page }) => {
        gostGridOpt.loading = true
        return new Promise((resolve) => {
          getGlobalGostServers({
            merchant_id: gostFilter.merchant_id || undefined,
            tag_id: gostFilter.tag_id || undefined,
            size: page.pageSize,
            page: page.currentPage
          }).then((res) => {
            gostGridOpt.loading = false
            resolve({ total: res.data.total, result: res.data.list || [] })
          }).catch(() => { gostGridOpt.loading = false })
        })
      }
    }
  }
})

// ========== 标签管理 ==========
const tagDialogVisible = ref(false)
const tagForm = reactive({ name: "", color: "#409EFF", description: "" })
const tagEditId = ref(0)

function showTagDialog() {
  tagDialogVisible.value = true
}

function resetTagForm() {
  tagForm.name = ""
  tagForm.color = "#409EFF"
  tagForm.description = ""
  tagEditId.value = 0
}

function editTag(tag: ResourceTagResp) {
  tagEditId.value = tag.id
  tagForm.name = tag.name
  tagForm.color = tag.color || "#409EFF"
  tagForm.description = tag.description
}

async function saveTag() {
  if (!tagForm.name.trim()) {
    ElMessage.warning("请输入标签名称")
    return
  }
  try {
    if (tagEditId.value) {
      await updateTag(tagEditId.value, tagForm)
      ElMessage.success("标签更新成功")
    } else {
      await createTag(tagForm)
      ElMessage.success("标签创建成功")
    }
    resetTagForm()
    await loadTags()
  } catch (e: any) {
    ElMessage.error(e?.message || "操作失败")
  }
}

async function removeTag(tag: ResourceTagResp) {
  try {
    await ElMessageBox.confirm(`确定删除标签「${tag.name}」吗？关联的资源标签也会被清除。`, "删除确认", { type: "warning" })
    await deleteTag(tag.id)
    ElMessage.success("标签已删除")
    await loadTags()
  } catch {}
}

// ========== 批量打标签 ==========
const batchTagDialogVisible = ref(false)
const batchTagResourceType = ref("")
const batchTagResourceIds = ref<number[]>([])
const batchTagSelectedIds = ref<number[]>([])

function showBatchTagDialog(type: "oss" | "gost") {
  const grid = type === "oss" ? ossGridDom.value : gostGridDom.value
  const records = grid?.getCheckboxRecords() || []
  if (records.length === 0) {
    ElMessage.warning("请先勾选要打标签的资源")
    return
  }
  batchTagResourceType.value = type === "oss" ? "oss_config" : "gost_server"
  batchTagResourceIds.value = records.map((r: any) => r.id)
  batchTagSelectedIds.value = []
  batchTagDialogVisible.value = true
}

async function submitBatchTag() {
  if (batchTagSelectedIds.value.length === 0) {
    ElMessage.warning("请选择至少一个标签")
    return
  }
  try {
    await assignTags({
      resource_type: batchTagResourceType.value,
      resource_ids: batchTagResourceIds.value,
      tag_ids: batchTagSelectedIds.value
    })
    ElMessage.success("标签分配成功")
    batchTagDialogVisible.value = false
    if (batchTagResourceType.value === "oss_config") {
      ossGridDom.value?.commitProxy("query")
    } else {
      gostGridDom.value?.commitProxy("query")
    }
  } catch (e: any) {
    ElMessage.error(e?.message || "操作失败")
  }
}

// ========== 移除标签 ==========
async function removeResourceTag(resourceType: string, resourceId: number, tagId: number) {
  try {
    await removeTags({
      resource_type: resourceType,
      resource_ids: [resourceId],
      tag_ids: [tagId]
    })
    ElMessage.success("标签已移除")
    if (resourceType === "oss_config") {
      ossGridDom.value?.commitProxy("query")
    } else {
      gostGridDom.value?.commitProxy("query")
    }
  } catch (e: any) {
    ElMessage.error(e?.message || "移除失败")
  }
}

// ========== OSS 健康检测（URL 可达性） ==========
const healthCheckLoading = ref(false)
const healthCheckDialogVisible = ref(false)
const healthCheckResults = ref<OssHealthCheckResult[]>([])

async function handleCheckOssHealth() {
  const records = ossGridDom.value?.getCheckboxRecords() || []
  if (records.length === 0) {
    ElMessage.warning("请先勾选要检测的 OSS 配置")
    return
  }
  const ids = records.map((r: any) => r.id)
  healthCheckLoading.value = true
  healthCheckResults.value = []
  healthCheckDialogVisible.value = true
  try {
    const res = await checkOssHealth({ oss_config_ids: ids })
    healthCheckResults.value = res.data || []
  } catch (e: any) {
    ElMessage.error(e?.message || "检测失败")
  } finally {
    healthCheckLoading.value = false
  }
}

// ========== 批量同步 IP ==========
const syncLoading = ref(false)

async function handleBatchSyncIP() {
  const records = gostGridDom.value?.getCheckboxRecords() || []
  if (records.length === 0) {
    ElMessage.warning("请先勾选要同步的 GOST 服务器")
    return
  }
  const merchantIds = [...new Set(records.map((r: any) => r.merchant_id))]
  try {
    await ElMessageBox.confirm(
      `将为 ${merchantIds.length} 个商户的 GOST IP 同步到所有 OSS，是否继续？`,
      "批量同步确认",
      { type: "warning" }
    )
    syncLoading.value = true
    const res = await batchSyncGostIP({ merchant_ids: merchantIds })
    const results = res.data || []
    const successCount = results.filter((r: any) => r.summary?.fail_count === 0).length
    ElMessage.success(`同步完成：${successCount}/${results.length} 个商户全部成功`)
  } catch (e: any) {
    if (e !== "cancel") {
      ElMessage.error(e?.message || "同步失败")
    }
  } finally {
    syncLoading.value = false
  }
}
</script>

<template>
  <div class="resource-overview">
    <!-- 页头 -->
    <div class="page-header">
      <h3>资源总览</h3>
      <el-button @click="showTagDialog">
        管理标签
      </el-button>
    </div>

    <!-- Tabs -->
    <el-tabs v-model="activeTab" type="border-card">
      <!-- Tab 1: OSS 配置 -->
      <el-tab-pane label="OSS 配置" name="oss">
        <!-- 自定义筛选栏 -->
        <div class="filter-bar">
          <el-select v-model="ossFilter.merchant_id" placeholder="商户" clearable filterable style="width: 200px;">
            <el-option v-for="m in merchantOptions" :key="m.value" :label="m.label" :value="m.value" />
          </el-select>
          <el-select v-model="ossFilter.cloud_type" placeholder="云类型" clearable style="width: 130px;">
            <el-option label="AWS S3" value="aws" />
            <el-option label="阿里云 OSS" value="aliyun" />
            <el-option label="腾讯云 COS" value="tencent" />
          </el-select>
          <el-select v-model="ossFilter.tag_id" placeholder="标签" clearable style="width: 140px;">
            <el-option v-for="tag in tagOptions" :key="tag.id" :label="tag.name" :value="tag.id" />
          </el-select>
          <el-button type="primary" @click="refreshOssGrid">查询</el-button>
          <el-button @click="resetOssFilter">重置</el-button>
        </div>

        <vxe-grid ref="ossGridDom" v-bind="ossGridOpt">
          <template #oss-toolbar-btns>
            <vxe-button status="success" icon="vxe-icon-indicator" :loading="healthCheckLoading" @click="handleCheckOssHealth">
              URL 检测
            </vxe-button>
            <vxe-button status="primary" icon="vxe-icon-tag" @click="showBatchTagDialog('oss')">
              批量打标签
            </vxe-button>
          </template>

          <template #cloud-type-slot="{ row }">
            <el-tag :type="getCloudTypeTagType(row.cloud_type)" size="small">
              {{ getCloudTypeLabel(row.cloud_type) }}
            </el-tag>
          </template>

          <template #download-url-slot="{ row }">
            <a v-if="row.download_url" :href="row.download_url" target="_blank" class="download-url-link">
              {{ row.download_url }}
            </a>
            <span v-else>-</span>
          </template>

          <template #default-slot="{ row }">
            <el-tag v-if="row.is_default === 1" type="success" size="small">默认</el-tag>
            <span v-else>-</span>
          </template>

          <template #status-slot="{ row }">
            <el-tag :type="row.status === 1 ? 'success' : 'danger'" size="small">
              {{ row.status === 1 ? "启用" : "禁用" }}
            </el-tag>
          </template>

          <template #tags-slot="{ row }">
            <el-tag
              v-for="tag in row.tags"
              :key="tag.id"
              :color="tag.color"
              size="small"
              closable
              style="margin-right: 4px; color: #fff;"
              @close="removeResourceTag('oss_config', row.id, tag.id)"
            >
              {{ tag.name }}
            </el-tag>
            <span v-if="!row.tags?.length">-</span>
          </template>

          <template #oss-row-operate="{ row }">
            <el-button type="primary" link size="small" @click="showBatchTagDialog('oss')">
              打标签
            </el-button>
          </template>
        </vxe-grid>
      </el-tab-pane>

      <!-- Tab 2: 隧道服务器 -->
      <el-tab-pane label="隧道服务器" name="gost">
        <!-- 自定义筛选栏 -->
        <div class="filter-bar">
          <el-select v-model="gostFilter.merchant_id" placeholder="商户" clearable filterable style="width: 200px;">
            <el-option v-for="m in merchantOptions" :key="m.value" :label="m.label" :value="m.value" />
          </el-select>
          <el-select v-model="gostFilter.tag_id" placeholder="标签" clearable style="width: 140px;">
            <el-option v-for="tag in tagOptions" :key="tag.id" :label="tag.name" :value="tag.id" />
          </el-select>
          <el-button type="primary" @click="refreshGostGrid">查询</el-button>
          <el-button @click="resetGostFilter">重置</el-button>
        </div>

        <vxe-grid ref="gostGridDom" v-bind="gostGridOpt">
          <template #gost-toolbar-btns>
            <vxe-button status="warning" icon="vxe-icon-refresh" :loading="syncLoading" @click="handleBatchSyncIP">
              批量同步IP
            </vxe-button>
            <vxe-button status="primary" icon="vxe-icon-tag" @click="showBatchTagDialog('gost')">
              批量打标签
            </vxe-button>
          </template>

          <template #primary-slot="{ row }">
            <el-tag v-if="row.is_primary === 1" type="warning" size="small">主</el-tag>
            <span v-else>-</span>
          </template>

          <template #status-slot="{ row }">
            <el-tag :type="row.status === 1 ? 'success' : 'danger'" size="small">
              {{ row.status === 1 ? "启用" : "禁用" }}
            </el-tag>
          </template>

          <template #tags-slot="{ row }">
            <el-tag
              v-for="tag in row.tags"
              :key="tag.id"
              :color="tag.color"
              size="small"
              closable
              style="margin-right: 4px; color: #fff;"
              @close="removeResourceTag('gost_server', row.id, tag.id)"
            >
              {{ tag.name }}
            </el-tag>
            <span v-if="!row.tags?.length">-</span>
          </template>

          <template #gost-row-operate="{ row }">
            <el-button type="primary" link size="small" @click="showBatchTagDialog('gost')">
              打标签
            </el-button>
          </template>
        </vxe-grid>
      </el-tab-pane>
    </el-tabs>

    <!-- 标签管理弹窗 -->
    <el-dialog v-model="tagDialogVisible" title="管理标签" width="500px">
      <div class="tag-manager">
        <div class="tag-form">
          <el-input v-model="tagForm.name" placeholder="标签名称" style="width: 140px;" />
          <el-color-picker v-model="tagForm.color" size="default" />
          <el-input v-model="tagForm.description" placeholder="描述(可选)" style="width: 160px;" />
          <el-button type="primary" @click="saveTag">
            {{ tagEditId ? "更新" : "添加" }}
          </el-button>
          <el-button v-if="tagEditId" @click="resetTagForm">
            取消
          </el-button>
        </div>

        <div class="tag-list">
          <div v-for="tag in tagOptions" :key="tag.id" class="tag-item">
            <el-tag :color="tag.color" style="color: #fff;">
              {{ tag.name }}
            </el-tag>
            <span class="tag-desc">{{ tag.description }}</span>
            <div class="tag-actions">
              <el-button type="primary" link size="small" @click="editTag(tag)">编辑</el-button>
              <el-button type="danger" link size="small" @click="removeTag(tag)">删除</el-button>
            </div>
          </div>
          <el-empty v-if="tagOptions.length === 0" description="暂无标签" :image-size="60" />
        </div>
      </div>
    </el-dialog>

    <!-- OSS URL 检测结果弹窗 -->
    <el-dialog v-model="healthCheckDialogVisible" title="OSS 下载地址检测" width="900px" top="5vh">
      <div v-if="healthCheckLoading" style="text-align: center; padding: 40px 0;">
        <el-icon class="is-loading" :size="32"><Loading /></el-icon>
        <p style="margin-top: 12px; color: #909399;">正在检测中，请稍候...</p>
      </div>
      <div v-else>
        <el-table :data="healthCheckResults" border stripe max-height="60vh">
          <el-table-column prop="oss_config_name" label="配置" width="100" />
          <el-table-column prop="merchant_name" label="商户" width="100" />
          <el-table-column prop="cloud_type" label="云类型" width="80">
            <template #default="{ row }">
              <el-tag :type="getCloudTypeTagType(row.cloud_type)" size="small">
                {{ getCloudTypeLabel(row.cloud_type) }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="download_url" label="下载地址" min-width="220" show-overflow-tooltip />
          <el-table-column label="状态" width="80" align="center">
            <template #default="{ row }">
              <el-tooltip :content="`HTTP ${row.status_code} - ${row.message}`" placement="top">
                <el-tag :type="row.healthy ? 'success' : 'danger'" size="small">
                  {{ row.healthy ? "可达" : "不可达" }}
                </el-tag>
              </el-tooltip>
            </template>
          </el-table-column>
          <el-table-column label="CDN" width="80" align="center">
            <template #default="{ row }">
              <el-tooltip v-if="row.cdn_url" :content="row.cdn_url" placement="top">
                <el-tag :type="row.cdn_healthy ? 'success' : 'danger'" size="small">
                  {{ row.cdn_healthy ? "可达" : "不可达" }}
                </el-tag>
              </el-tooltip>
              <span v-else>-</span>
            </template>
          </el-table-column>
          <el-table-column prop="latency" label="耗时" width="90" />
        </el-table>
      </div>
    </el-dialog>

    <!-- 批量打标签弹窗 -->
    <el-dialog v-model="batchTagDialogVisible" title="批量打标签" width="400px">
      <p style="margin-bottom: 10px;">已选择 {{ batchTagResourceIds.length }} 个资源</p>
      <el-checkbox-group v-model="batchTagSelectedIds">
        <el-checkbox v-for="tag in tagOptions" :key="tag.id" :value="tag.id" style="margin-bottom: 8px;">
          <el-tag :color="tag.color" size="small" style="color: #fff;">{{ tag.name }}</el-tag>
          <span v-if="tag.description" style="color: #909399; margin-left: 6px; font-size: 12px;">{{ tag.description }}</span>
        </el-checkbox>
      </el-checkbox-group>
      <el-empty v-if="tagOptions.length === 0" description="请先创建标签" :image-size="60" />
      <template #footer>
        <el-button @click="batchTagDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="submitBatchTag">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.resource-overview {
  padding: 15px;
  background-color: var(--el-bg-color);
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 15px;
}

.page-header h3 {
  margin: 0;
}

.filter-bar {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 12px;
  flex-wrap: wrap;
}

.download-url-link {
  color: var(--el-color-primary);
  text-decoration: none;
  font-size: 12px;
  word-break: break-all;
}

.download-url-link:hover {
  text-decoration: underline;
}

.tag-manager .tag-form {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 15px;
  padding-bottom: 15px;
  border-bottom: 1px solid var(--el-border-color-lighter);
}

.tag-list .tag-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 0;
  border-bottom: 1px solid var(--el-border-color-extra-light);
}

.tag-list .tag-item:last-child {
  border-bottom: none;
}

.tag-desc {
  flex: 1;
  color: #909399;
  font-size: 13px;
}

.tag-actions {
  white-space: nowrap;
}
</style>
