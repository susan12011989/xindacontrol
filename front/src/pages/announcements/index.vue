<script lang="ts" setup>
import type { MessageEntityDTO } from "@/common/apis/announcements/type"
import type { MerchantResp } from "@/common/apis/merchant/type"
import { createAnnouncementApi, getAnnouncementLogsApi } from "@/common/apis/announcements"
import { getMerchantList } from "@/common/apis/merchant"
import { ElMessage, ElMessageBox } from "element-plus"
import { computed, onMounted, ref } from "vue"

type EntityType = MessageEntityDTO["type"]

const text = ref("")
const inputRef = ref<any>(null)
const entities = ref<MessageEntityDTO[]>([])
const selectStart = ref(0)
const selectEnd = ref(0)

const form = ref({ silent: true, noforwards: true })
const sending = ref(false)
const merchants = ref<MerchantResp[]>([])
const selectedMerchantIds = ref<number[]>([])
const broadcastAll = ref(true)

// 日志列表
interface LogItem {
  id: number
  text: string
  merchant_nos: string[]
  broadcast: boolean
  created_at: string
}
const logs = ref<LogItem[]>([])
const total = ref(0)
const page = ref(1)
const size = ref(10)
const loadingLogs = ref(false)

onMounted(async () => {
  try {
    const { data } = await getMerchantList({ page: 1, size: 200 })
    merchants.value = data.list || []
  } catch {
    // ignore
  }
  await loadLogs()
})

async function loadLogs() {
  loadingLogs.value = true
  try {
    const { data } = await getAnnouncementLogsApi({ page: page.value, size: size.value })
    logs.value = data.list || []
    total.value = data.total || 0
  } finally {
    loadingLogs.value = false
  }
}

// 右键菜单状态
const showMenu = ref(false)
const menuX = ref(0)
const menuY = ref(0)

function updateSelection() {
  const el = (inputRef.value?.textarea || inputRef.value?.input) as HTMLTextAreaElement | HTMLInputElement | undefined
  if (!el) return
  selectStart.value = el.selectionStart || 0
  selectEnd.value = el.selectionEnd || 0
}

function onContextMenu(e: MouseEvent) {
  // 更新选择区
  updateSelection()
  const len = Math.max(0, selectEnd.value - selectStart.value)
  if (len === 0) {
    showMenu.value = false
    return
  }
  menuX.value = e.clientX
  menuY.value = e.clientY
  showMenu.value = true
}

function overlap(aStart: number, aLen: number, bStart: number, bLen: number) {
  const aEnd = aStart + aLen
  const bEnd = bStart + bLen
  return Math.max(aStart, bStart) < Math.min(aEnd, bEnd)
}

function removeEntitiesInRange(start: number, len: number, type?: EntityType) {
  entities.value = entities.value.filter((e) => {
    if (type && e.type !== type) return true
    return !overlap(e.offset, e.length, start, len)
  })
}

function pushEntity(type: EntityType, offset: number, length: number, url?: string) {
  if (length <= 0) return
  // 合并/去重简单处理：移除同范围同类型后再插入
  removeEntitiesInRange(offset, length, type)
  const ent: MessageEntityDTO = { type, offset, length }
  if (type === "text_url" && url) ent.url = url
  entities.value.push(ent)
  // 按 offset 排序，保持稳定
  entities.value.sort((a, b) => a.offset - b.offset || a.length - b.length)
}

function applyEntity(type: EntityType) {
  const start = selectStart.value
  const end = selectEnd.value
  const len = Math.max(0, end - start)
  if (len === 0) return
  pushEntity(type, start, len)
  showMenu.value = false
}

async function applyLink() {
  const start = selectStart.value
  const end = selectEnd.value
  const len = Math.max(0, end - start)
  if (len === 0) return
  try {
    const { value } = await ElMessageBox.prompt("请输入链接地址", "插入链接", { inputPlaceholder: "https://" })
    if (!value) return
    pushEntity("text_url", start, len, value)
  } catch {}
  showMenu.value = false
}

const renderedHtml = computed(() => renderPreview(text.value, entities.value))

function renderPreview(t: string, es: MessageEntityDTO[]) {
  if (!t) return ""
  if (!es || es.length === 0) return escapeHtml(t).replace(/\n/g, "<br>")
  // 将实体切片为非重叠片段，按 offset 排序后逐个渲染（简单叠加样式）
  const sorted = [...es].sort((a, b) => a.offset - b.offset || b.length - a.length)
  let html = ""
  let i = 0
  for (const e of sorted) {
    if (e.offset > i) html += escapeHtml(t.slice(i, e.offset))
    const seg = t.slice(e.offset, e.offset + e.length)
    html += wrapEntityHtml(escapeHtml(seg), e)
    i = e.offset + e.length
  }
  if (i < t.length) html += escapeHtml(t.slice(i))
  return html.replace(/\n/g, "<br>")
}

function wrapEntityHtml(seg: string, e: MessageEntityDTO) {
  switch (e.type) {
    case "bold":
      return `<b>${seg}</b>`
    case "italic":
      return `<i>${seg}</i>`
    case "underline":
      return `<u>${seg}</u>`
    case "strike":
      return `<s>${seg}</s>`
    case "code":
      return `<code style="background:#f5f5f5;padding:2px 4px;border-radius:4px">${seg}</code>`
    case "text_url":
      return `<a href="${escapeAttr(e.url || "#")}" target="_blank" rel="noreferrer">${seg}</a>`
    default:
      return seg
  }
}

function escapeHtml(s: string) {
  return s
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;")
    .replace(/'/g, "&#39;")
}

function escapeAttr(s: string) {
  return escapeHtml(s).replace(/"/g, "&quot;")
}

async function onSend() {
  if (!text.value) {
    ElMessage.warning("请填写文本")
    return
  }
  if (!broadcastAll.value && selectedMerchantIds.value.length === 0) {
    ElMessage.warning("请选择商户或勾选全部广播")
    return
  }
  sending.value = true
  try {
    await createAnnouncementApi({
      text: text.value,
      entities: entities.value,
      silent: form.value.silent,
      noforwards: form.value.noforwards,
      merchant_ids: broadcastAll.value ? undefined : selectedMerchantIds.value
    })
    ElMessage.success("已提交发送")
    // 刷新日志
    page.value = 1
    await loadLogs()
  } catch (e: any) {
    ElMessage.error(e?.message || "发送失败")
  } finally {
    sending.value = false
  }
}
</script>

<template>
  <div class="announcement-page p-4" @click="showMenu = false">
    <el-card>
      <template #header>
        <div class="flex items-center justify-between">
          <div class="font-600">系统公告</div>
          <div class="space-x-2">
            <el-switch v-model="form.silent" active-text="静音" />
            <el-switch v-model="form.noforwards" active-text="禁转发" />
          </div>
        </div>
      </template>

      <!-- 右键菜单：监听 wrapper 的 contextmenu -->
      <div class="mt-2 flex items-center gap-3">
        <el-checkbox v-model="broadcastAll">全部广播</el-checkbox>
        <el-select
          v-model="selectedMerchantIds"
          multiple
          filterable
          placeholder="选择商户（可多选）"
          :disabled="broadcastAll"
          style="min-width: 360px;"
        >
          <el-option v-for="m in merchants" :key="m.id" :label="m.name || `商户${m.id}`" :value="m.id" />
        </el-select>
      </div>

      <!-- 右键菜单：监听 wrapper 的 contextmenu -->
      <div @contextmenu.prevent="onContextMenu">
        <el-input
          ref="inputRef"
          v-model="text"
          type="textarea"
          :rows="6"
          placeholder="输入公告文本..."
          @mouseup="updateSelection"
          @keyup="updateSelection"
        />
      </div>

      <div class="mt-4">
        <el-button type="primary" :loading="sending" @click="onSend">发送公告</el-button>
      </div>
    </el-card>

    <el-card class="mt-4">
      <template #header>
        <div class="font-600">预览</div>
      </template>
      <div class="preview" v-html="renderedHtml"></div>
    </el-card>

    <el-card class="mt-4">
      <template #header>
        <div class="flex items-center justify-between">
          <div class="font-600">发送记录</div>
        </div>
      </template>
      <el-table :data="logs" v-loading="loadingLogs" size="small">
        <el-table-column label="发送内容" min-width="360">
          <template #default="{ row }">
            <div class="truncate" :title="row.text">{{ row.text }}</div>
          </template>
        </el-table-column>
        <el-table-column label="发送目标" min-width="200">
          <template #default="{ row }">
            <el-tag v-if="row.broadcast" type="success">全部广播</el-tag>
            <template v-else>
              <el-tooltip placement="top" :content="row.merchant_nos.join(', ')">
                <span>已选 {{ row.merchant_nos.length }} 个</span>
              </el-tooltip>
            </template>
          </template>
        </el-table-column>
        <el-table-column label="时间" prop="created_at" min-width="180" />
      </el-table>
      <div class="mt-2 flex justify-end">
        <el-pagination
          v-model:current-page="page"
          v-model:page-size="size"
          layout="total, prev, pager, next, jumper"
          :total="total"
          @current-change="loadLogs"
          @size-change="() => { page = 1; loadLogs() }"
        />
      </div>
    </el-card>

    <!-- 自定义右键菜单 -->
    <div v-if="showMenu" class="ctx-menu" @click.stop :style="{ left: `${menuX}px`, top: `${menuY}px` }">
      <div class="ctx-item" @click="applyEntity('bold')">加粗</div>
      <div class="ctx-item" @click="applyEntity('italic')">斜体</div>
      <div class="ctx-item" @click="applyEntity('underline')">下划线</div>
      <div class="ctx-item" @click="applyEntity('strike')">删除线</div>
      <div class="ctx-item" @click="applyEntity('code')">等宽</div>
      <div class="ctx-item" @click="applyLink()">链接</div>
    </div>
  </div>
</template>

<style scoped>
.preview {
  white-space: pre-wrap;
  word-break: break-word;
  line-height: 1.6;
}
.ctx-menu {
  position: fixed;
  z-index: 9999;
  background: #fff;
  border: 1px solid #e5e5e5;
  box-shadow: 0 6px 18px rgba(0, 0, 0, 0.08);
  border-radius: 6px;
  padding: 4px 0;
  min-width: 120px;
}
.ctx-item {
  padding: 6px 12px;
  cursor: pointer;
  font-size: 13px;
}
.ctx-item:hover {
  background: #f5f7fa;
}
</style>
