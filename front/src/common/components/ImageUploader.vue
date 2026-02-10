<script lang="ts" setup>
/**
 * 图片上传组件
 * 支持上传到本地服务器并返回URL
 */
import type { UploadFile, UploadProps } from "element-plus"
import { Plus, Delete, Loading } from "@element-plus/icons-vue"
import { request } from "@/http/axios"

defineOptions({ name: "ImageUploader" })

const props = withDefaults(defineProps<{
  modelValue?: string
  assetType?: string  // logo, icon, splash
  prefix?: string     // 兼容旧参数，不再使用
  accept?: string
  maxSize?: number // MB
  width?: number
  height?: number
  tip?: string
}>(), {
  modelValue: "",
  assetType: "logo",
  prefix: "",
  accept: "image/*",
  maxSize: 5,
  width: 120,
  height: 120,
  tip: ""
})

const emit = defineEmits<{
  (e: "update:modelValue", value: string): void
}>()

const uploading = ref(false)
const previewUrl = computed(() => props.modelValue || "")

// 上传前校验
const beforeUpload: UploadProps["beforeUpload"] = (file) => {
  // 检查文件类型
  if (!file.type.startsWith("image/")) {
    ElMessage.error("只能上传图片文件")
    return false
  }

  // 检查文件大小
  if (file.size > props.maxSize * 1024 * 1024) {
    ElMessage.error(`图片大小不能超过 ${props.maxSize}MB`)
    return false
  }

  return true
}

// 自定义上传到本地服务器
const handleUpload: UploadProps["httpRequest"] = async (options) => {
  uploading.value = true

  try {
    const file = options.file
    const form = new FormData()
    form.append("file", file)
    form.append("type", props.assetType)

    const res = await request<any>({
      url: "/merchant/upload-asset",
      method: "post",
      headers: { "Content-Type": "multipart/form-data" },
      data: form,
      timeout: 60000
    })

    // axios 拦截器返回 { code, data, message }，需要取 data.url
    if (res?.data?.url) {
      emit("update:modelValue", res.data.url)
      ElMessage.success("上传成功")
    } else {
      throw new Error(res?.message || "上传失败")
    }
  } catch (error: any) {
    console.error("上传失败:", error)
    ElMessage.error(error.message || "上传失败")
  } finally {
    uploading.value = false
  }
}

// 删除图片
function handleRemove() {
  emit("update:modelValue", "")
}
</script>

<template>
  <div class="image-uploader">
    <div v-if="previewUrl" class="preview-container" :style="{ width: `${width}px`, height: `${height}px` }">
      <el-image
        :src="previewUrl"
        fit="contain"
        class="preview-image"
      />
      <div class="preview-actions">
        <el-button
          type="danger"
          size="small"
          circle
          :icon="Delete"
          @click="handleRemove"
        />
      </div>
    </div>

    <el-upload
      v-else
      class="uploader"
      :style="{ width: `${width}px`, height: `${height}px` }"
      :show-file-list="false"
      :accept="accept"
      :before-upload="beforeUpload"
      :http-request="handleUpload"
    >
      <div v-if="uploading" class="uploading">
        <el-icon class="is-loading"><Loading /></el-icon>
        <span>上传中...</span>
      </div>
      <div v-else class="upload-placeholder">
        <el-icon><Plus /></el-icon>
        <span>点击上传</span>
      </div>
    </el-upload>

    <div v-if="tip" class="uploader-tip">{{ tip }}</div>

    <!-- URL 输入框 -->
    <el-input
      :model-value="modelValue"
      placeholder="或直接输入图片URL"
      class="url-input"
      clearable
      @update:model-value="emit('update:modelValue', $event)"
    >
      <template #prepend>URL</template>
    </el-input>
  </div>
</template>

<style lang="scss" scoped>
.image-uploader {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.preview-container {
  position: relative;
  border: 1px solid #dcdfe6;
  border-radius: 6px;
  overflow: hidden;

  .preview-image {
    width: 100%;
    height: 100%;
  }

  .preview-actions {
    position: absolute;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    display: flex;
    align-items: center;
    justify-content: center;
    background: rgba(0, 0, 0, 0.5);
    opacity: 0;
    transition: opacity 0.3s;
  }

  &:hover .preview-actions {
    opacity: 1;
  }
}

.uploader {
  :deep(.el-upload) {
    width: 100%;
    height: 100%;
    border: 1px dashed #dcdfe6;
    border-radius: 6px;
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: border-color 0.3s;

    &:hover {
      border-color: #409eff;
    }
  }
}

.upload-placeholder,
.uploading {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  color: #909399;
  font-size: 12px;
  gap: 4px;

  .el-icon {
    font-size: 24px;
  }
}

.uploading {
  color: #409eff;

  .is-loading {
    animation: rotating 2s linear infinite;
  }
}

.uploader-tip {
  font-size: 12px;
  color: #909399;
  line-height: 1.4;
}

.url-input {
  margin-top: 4px;
}

@keyframes rotating {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}
</style>
