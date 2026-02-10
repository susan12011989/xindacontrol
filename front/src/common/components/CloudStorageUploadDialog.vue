<script lang="ts" setup>
/**
 * 云存储上传对话框组件
 * 支持批量上传和进度跟踪
 */
import type { UploadProps, UploadUserFile } from "element-plus"
import type { CloudType, UploadQueueItem, UploadStats } from "@@/apis/cloud_storage/type"
import { formatFileSize } from "@@/apis/cloud_storage"
import { CircleCheck, CircleClose, Clock, Loading, Upload } from "@element-plus/icons-vue"

defineOptions({ name: "CloudStorageUploadDialog" })

const props = defineProps<{
  modelValue: boolean
  cloudType: CloudType
  bucket: string
  prefix?: string
  uploadFn: (form: FormData, onUploadProgress?: (progressEvent: any) => void) => Promise<any>
}>()

const emit = defineEmits<{
  (e: "update:modelValue", value: boolean): void
  (e: "success"): void
}>()

const visible = computed({
  get: () => props.modelValue,
  set: (val) => emit("update:modelValue", val)
})

const uploadFileList = ref<UploadUserFile[]>([])
const uploading = ref(false)
const uploadQueue = ref<UploadQueueItem[]>([])
const uploadStats = ref<UploadStats>({ total: 0, success: 0, failed: 0 })

const handleUploadChange: UploadProps["onChange"] = (uploadFile, uploadFiles) => {
  uploadFileList.value = uploadFiles
}

async function onConfirmUpload() {
  if (!uploadFileList.value.length) {
    ElMessage.warning("请选择要上传的文件")
    return
  }

  // 检查是否有大文件
  const largeFiles = uploadFileList.value.filter(f => f.raw && f.raw.size > 100 * 1024 * 1024)
  if (largeFiles.length > 0) {
    const totalSizeMB = largeFiles.reduce((sum, f) => sum + (f.raw?.size || 0), 0) / 1024 / 1024
    const confirmed = await ElMessageBox.confirm(
      `检测到 ${largeFiles.length} 个大文件（总计 ${totalSizeMB.toFixed(2)} MB），上传可能需要较长时间，确认继续吗？`,
      "大文件上传提示",
      { type: "warning" }
    ).catch(() => false)
    if (!confirmed) return
  }

  // 初始化上传队列
  uploadQueue.value = uploadFileList.value.map(f => ({
    name: f.name,
    status: "waiting" as const,
    progress: 0,
    size: f.raw?.size || 0
  }))

  uploadStats.value = {
    total: uploadFileList.value.length,
    success: 0,
    failed: 0
  }

  uploading.value = true

  // 逐个上传文件
  for (let i = 0; i < uploadFileList.value.length; i++) {
    const file = uploadFileList.value[i].raw
    if (!file) continue

    uploadQueue.value[i].status = "uploading"

    try {
      const fd = new FormData()
      fd.append("file", file)
      fd.append("bucket", props.bucket)
      fd.append("object_key", props.prefix ? `${props.prefix}${file.name}` : file.name)

      await props.uploadFn(fd, (progressEvent: any) => {
        if (progressEvent.total) {
          uploadQueue.value[i].progress = Math.round((progressEvent.loaded * 100) / progressEvent.total)
        }
      })

      uploadQueue.value[i].status = "success"
      uploadQueue.value[i].progress = 100
      uploadStats.value.success++
    } catch {
      uploadQueue.value[i].status = "failed"
      uploadStats.value.failed++
    }
  }

  uploading.value = false

  // 显示上传结果
  const { success, failed } = uploadStats.value
  if (failed === 0) {
    ElMessage.success(`上传完成！成功上传 ${success} 个文件`)
  } else {
    ElMessage.warning(`上传完成！成功 ${success} 个，失败 ${failed} 个`)
  }

  // 触发成功事件
  emit("success")

  // 延迟关闭对话框，让用户看到结果
  setTimeout(() => {
    visible.value = false
  }, 2000)
}

function onClose() {
  uploadFileList.value = []
  uploadQueue.value = []
  uploadStats.value = { total: 0, success: 0, failed: 0 }
}
</script>

<template>
  <el-dialog
    v-model="visible"
    title="上传文件"
    width="500px"
    :close-on-click-modal="false"
    :close-on-press-escape="!uploading"
    :show-close="!uploading"
    @closed="onClose"
  >
    <div class="upload-info">
      <el-alert
        title="上传提示"
        :closable="false"
        type="info"
        show-icon
      >
        <template #default>
          <p>目标 Bucket: <strong>{{ bucket }}</strong></p>
          <p v-if="prefix">对象前缀: <strong>{{ prefix }}</strong></p>
        </template>
      </el-alert>
    </div>

    <el-upload
      v-model:file-list="uploadFileList"
      class="upload-area"
      drag
      :auto-upload="false"
      multiple
      :disabled="uploading"
      :on-change="handleUploadChange"
    >
      <el-icon class="el-icon--upload"><Upload /></el-icon>
      <div class="el-upload__text">
        拖拽文件到这里或<em>点击选择</em>
      </div>
      <template #tip>
        <div class="el-upload__tip">
          支持批量上传，建议单个文件不超过 500MB
        </div>
      </template>
    </el-upload>

    <!-- 上传队列 -->
    <div v-if="uploadQueue.length > 0" class="upload-queue">
      <div class="queue-header">
        <span class="queue-title">上传队列（{{ uploadStats.total }} 个文件）</span>
        <span class="queue-stats">
          <el-tag v-if="uploadStats.success > 0" type="success" size="small">成功: {{ uploadStats.success }}</el-tag>
          <el-tag v-if="uploadStats.failed > 0" type="danger" size="small">失败: {{ uploadStats.failed }}</el-tag>
        </span>
      </div>
      <el-scrollbar max-height="300px">
        <div
          v-for="(item, index) in uploadQueue"
          :key="index"
          class="queue-item"
          :class="`status-${item.status}`"
        >
          <div class="item-info">
            <el-icon v-if="item.status === 'waiting'" class="item-icon"><Clock /></el-icon>
            <el-icon v-else-if="item.status === 'uploading'" class="item-icon uploading"><Loading /></el-icon>
            <el-icon v-else-if="item.status === 'success'" class="item-icon success"><CircleCheck /></el-icon>
            <el-icon v-else-if="item.status === 'failed'" class="item-icon failed"><CircleClose /></el-icon>
            <span class="item-name">{{ item.name }}</span>
            <span class="item-size">{{ formatFileSize(item.size) }}</span>
          </div>
          <el-progress
            v-if="item.status === 'uploading' || item.status === 'success'"
            :percentage="item.progress"
            :status="item.status === 'success' ? 'success' : undefined"
            :show-text="false"
          />
        </div>
      </el-scrollbar>
    </div>

    <template #footer>
      <el-button :disabled="uploading" @click="visible = false">取消</el-button>
      <el-button type="primary" :loading="uploading" :disabled="!uploadFileList.length" @click="onConfirmUpload">
        {{ uploading ? '上传中...' : '确认上传' }}
      </el-button>
    </template>
  </el-dialog>
</template>

<style lang="scss" scoped>
.upload-info {
  margin-bottom: 20px;

  p {
    margin: 4px 0;
  }
}

.upload-area {
  :deep(.el-upload-dragger) {
    padding: 40px;
  }
}

.upload-queue {
  margin-top: 20px;
  padding: 16px;
  background: #f5f7fa;
  border-radius: 8px;

  .queue-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 12px;
    padding-bottom: 12px;
    border-bottom: 1px solid #e4e7ed;

    .queue-title {
      font-size: 14px;
      font-weight: 600;
      color: #303133;
    }

    .queue-stats {
      display: flex;
      gap: 8px;
    }
  }

  .queue-item {
    padding: 12px;
    margin-bottom: 8px;
    background: white;
    border-radius: 4px;
    border: 1px solid #e4e7ed;
    transition: all 0.3s;

    &.status-uploading {
      border-color: #409eff;
      background: #ecf5ff;
    }

    &.status-success {
      border-color: #67c23a;
      background: #f0f9ff;
    }

    &.status-failed {
      border-color: #f56c6c;
      background: #fef0f0;
    }

    .item-info {
      display: flex;
      align-items: center;
      gap: 8px;
      margin-bottom: 8px;

      .item-icon {
        font-size: 16px;

        &.uploading {
          color: #409eff;
          animation: rotating 2s linear infinite;
        }

        &.success {
          color: #67c23a;
        }

        &.failed {
          color: #f56c6c;
        }
      }

      .item-name {
        flex: 1;
        font-size: 13px;
        color: #303133;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
      }

      .item-size {
        font-size: 12px;
        color: #909399;
      }
    }
  }

  @keyframes rotating {
    from {
      transform: rotate(0deg);
    }
    to {
      transform: rotate(360deg);
    }
  }
}
</style>
