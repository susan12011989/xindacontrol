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
const merchantOptions: { label: string; value: number }[] = reactive([])
const tagOptions: ResourceTagResp[] = reactive([])

const cloudTypeOptions = [
  { label: "全部", value: "" },
  { label: "AWS S3", value: "aws" },
  { label: "阿里云 OSS", value: "aliyun" },
  { label: "腾讯云 COS", value: "tencent" }
]

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
    const list = res.data.list?.map((m: any) => ({ label: `${m.name} (${m.no})`, value: m.id })) || []
    merchantOptions.length = 0
    merchantOptions.push(...list)
  } catch (e) {
    console.error("加载商户列表失败", e)
  }
}

async function loadTags() {
  try {
    const res = await getTagList()
    tagOptions.length = 0
    tagOptions.push(...(res.data || []))
  } catch (e) {
    console.error("加载标签列表失败", e)
  }
}

onMounted(() => {
  loadMerchants()
  loadTags()
})

// ========== OSS 配置列表 ==========
const ossGridDom = ref<VxeGridInstance>()
const ossGridOpt: VxeGridProps = reactive({
  loading: true,
  autoResize: true,
  pagerConfig: { align: "right" },
  formConfig: {
    items: [
      {
        field: "merchant_id",
        itemRender: {
          name: "$select",
          options: merchantOptions,
          props: { placeholder: "商户", clearable: true, filterable: true }
        }
      },
      {
        field: "cloud_type",
        itemRender: {
          name: "$select",
          options: cloudTypeOptions.slice(1), // 去掉"全部"
          props: { placeholder: "云类型", clearable: true }
        }
      },
      {
        field: "tag_id",
        itemRender: {
          name: "$select",
          options: tagOptions.map(t => ({ label: t.name, value: t.id })),
          props: { placeholder: "标签", clearable: true }
        }
      },
      {
        itemRender: {
          name: "$buttons",
          children: [
            { props: { type: "submit", content: "查询", status: "primary" } },
            { props: { type: "reset", content: "重置" } }
          ]
        }
      }
    ]
  },
  toolbarConfig: {
    refresh: true,
    custom: true,
    slots: { buttons: "oss-toolbar-btns" }
  },
  columns: [
    { type: "checkbox", width: "50px" },
    { type: "seq", width: "50px", title: "#" },
    { field: "merchant_name", title: "商户", width: 130 },
    { field: "name", title: "配置名称", width: 120 },
    { field: "cloud_type", title: "云类型", width: 100, slots: { default: "cloud-type-slot" } },
    { field: "bucket", title: "Bucket", width: 150, showOverflow: true },
    { field: "region", title: "区域", width: 120 },
    { field: "endpoint", title: "Endpoint", showOverflow: true },
    { field: "is_default", title: "默认", width: 70, slots: { default: "default-slot" } },
    { field: "status", title: "状态", width: 70, slots: { default: "status-slot" } },
    { field: "tags", title: "标签", width: 160, slots: { default: "tags-slot" } },
    { title: "操作", width: "100px", fixed: "right", slots: { default: "oss-row-operate" } }
  ],
  proxyConfig: {
    seq: true,
    form: true,
    autoLoad: true,
    props: { total: "total" },
    ajax: {
      query: ({ page, form }) => {
        ossGridOpt.loading = true
        return new Promise((resolve) => {
          getGlobalOssConfigs({
            merchant_id: form.merchant_id || undefined,
            cloud_type: form.cloud_type || undefined,
            tag_id: form.tag_id || undefined,
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

// ========== GOST 服务器列表 ==========
const gostGridDom = ref<VxeGridInstance>()
const gostGridOpt: VxeGridProps = reactive({
  loading: true,
  autoResize: true,
  pagerConfig: { align: "right" },
  formConfig: {
    items: [
      {
        field: "merchant_id",
        itemRender: {
          name: "$select",
          options: merchantOptions,
          props: { placeholder: "商户", clearable: true, filterable: true }
        }
      },
      {
        field: "tag_id",
        itemRender: {
          name: "$select",
          options: tagOptions.map(t => ({ label: t.name, value: t.id })),
          props: { placeholder: "标签", clearable: true }
        }
      },
      {
        itemRender: {
          name: "$buttons",
          children: [
            { props: { type: "submit", content: "查询", status: "primary" } },
            { props: { type: "reset", content: "重置" } }
          ]
        }
      }
    ]
  },
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
    form: true,
    autoLoad: true,
    props: { total: "total" },
    ajax: {
      query: ({ page, form }) => {
        gostGridOpt.loading = true
        return new Promise((resolve) => {
          getGlobalGostServers({
            merchant_id: form.merchant_id || undefined,
            tag_id: form.tag_id || undefined,
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
    // 刷新列表
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

// ========== OSS 健康检测 ==========
const healthCheckLoading = ref(false)
const healthCheckDialogVisible = ref(false)
const healthCheckResults = ref<OssHealthCheckResult[]>([])

const healthStepLabels: Record<string, string> = {
  sdk_connect: "SDK连接",
  upload: "上传测试",
  download_sdk: "SDK下载",
  download_url: "公网URL",
  download_cdn: "CDN域名",
  cleanup: "清理"
}

async function handleCheckOssHealth() {
  const records = ossGridDom.value?.getCheckboxRecords() || []
  if (records.length === 0) {
    ElMessage.warning("请先勾选要检测的 OSS 配置")
    return
  }
  const ids = records.map((r: any) => r.id)
  try {
    await ElMessageBox.confirm(
      `将检测 ${ids.length} 个 OSS 配置的可用性（包括SDK连接、上传、下载），是否继续？`,
      "OSS 健康检测",
      { type: "info" }
    )
    healthCheckLoading.value = true
    healthCheckResults.value = []
    healthCheckDialogVisible.value = true
    const res = await checkOssHealth({ oss_config_ids: ids })
    healthCheckResults.value = res.data || []
  } catch (e: any) {
    if (e !== "cancel") {
      ElMessage.error(e?.message || "检测失败")
    }
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
  // 提取去重的商户ID
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
        <vxe-grid ref="ossGridDom" v-bind="ossGridOpt">
          <template #oss-toolbar-btns>
            <vxe-button status="success" icon="vxe-icon-indicator" :loading="healthCheckLoading" @click="handleCheckOssHealth">
              检测可用性
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
        <!-- 创建/编辑表单 -->
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

        <!-- 标签列表 -->
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

    <!-- OSS 健康检测结果弹窗 -->
    <el-dialog v-model="healthCheckDialogVisible" title="OSS 健康检测结果" width="800px" top="5vh">
      <div v-if="healthCheckLoading" style="text-align: center; padding: 40px 0;">
        <el-icon class="is-loading" :size="32"><Loading /></el-icon>
        <p style="margin-top: 12px; color: #909399;">正在检测中，请稍候...</p>
      </div>
      <div v-else>
        <el-table :data="healthCheckResults" border stripe max-height="60vh">
          <el-table-column prop="oss_config_name" label="配置名称" width="120" />
          <el-table-column prop="merchant_name" label="商户" width="100" />
          <el-table-column prop="cloud_type" label="云类型" width="80">
            <template #default="{ row }">
              <el-tag :type="getCloudTypeTagType(row.cloud_type)" size="small">
                {{ getCloudTypeLabel(row.cloud_type) }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="bucket" label="Bucket" width="120" show-overflow-tooltip />
          <el-table-column label="检测结果" min-width="280">
            <template #default="{ row }">
              <div class="health-steps">
                <el-tooltip v-for="step in row.steps" :key="step.step" :content="`${step.message} (${step.latency})`" placement="top">
                  <el-tag :type="step.ok ? 'success' : 'danger'" size="small" style="margin: 2px 4px 2px 0;">
                    {{ healthStepLabels[step.step] || step.step }}
                    {{ step.ok ? '✓' : '✗' }}
                  </el-tag>
                </el-tooltip>
              </div>
            </template>
          </el-table-column>
          <el-table-column label="总体" width="80" align="center">
            <template #default="{ row }">
              <el-tag :type="row.healthy ? 'success' : 'danger'" size="small">
                {{ row.healthy ? "正常" : "异常" }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="duration" label="耗时" width="90" />
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

.health-steps {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
}
</style>
