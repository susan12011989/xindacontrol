<script setup lang="ts">
import type { Region } from "@/pages/cloud/aliyun/apis/type"
import type { Image, ShareAccount } from "@/pages/cloud/aliyun/instances/apis/type"
import type { CloudAccountOption } from "@@/apis/cloud_account/type"
import { regionListApi } from "@/pages/cloud/aliyun/apis"
import { describeImageSharePermission, getImageList, modifyImageSharePermission } from "@/pages/cloud/aliyun/instances/apis"
import { getCloudAccountOptions } from "@@/apis/cloud_account"
import { Key, Location, Share, Delete, Plus } from "@element-plus/icons-vue"
import { ElMessage, ElMessageBox } from "element-plus"
import { onMounted, ref, watch } from "vue"

defineOptions({
  name: "CloudAliyunImages"
})

// localStorage 存储键名
const STORAGE_CLOUD_ACCOUNT_KEY = "cloud_images_selected_cloud_account"
const STORAGE_REGION_KEY = "cloud_images_selected_region"

// 数据状态
const loading = ref(false)
const cloudAccountList = ref<CloudAccountOption[]>([])
const selectedCloudAccount = ref<number>()
const regionList = ref<Region[]>([])
const imageList = ref<Image[]>([])
const selectedRegions = ref<string[]>([])

// 镜像共享对话框
const shareDialogVisible = ref(false)
const shareLoading = ref(false)
const currentShareImage = ref<Image | null>(null)
const shareAccounts = ref<ShareAccount[]>([])
const newShareAccount = ref("")

// 从 localStorage 读取已保存的选择
function loadSelectionFromStorage() {
  try {
    const savedCloudAccountId = localStorage.getItem(STORAGE_CLOUD_ACCOUNT_KEY)
    const savedRegionIds = localStorage.getItem(STORAGE_REGION_KEY)

    if (savedCloudAccountId) {
      selectedCloudAccount.value = Number(savedCloudAccountId)
    }

    if (savedRegionIds) {
      selectedRegions.value = JSON.parse(savedRegionIds)
    }
  } catch (error) {
    console.error("读取localStorage数据失败:", error)
  }
}

// 监听选择变化，保存到localStorage
watch(selectedCloudAccount, (newValue) => {
  if (newValue) {
    localStorage.setItem(STORAGE_CLOUD_ACCOUNT_KEY, newValue.toString())
  } else {
    localStorage.removeItem(STORAGE_CLOUD_ACCOUNT_KEY)
  }
}, { deep: true })

watch(selectedRegions, (newValue) => {
  if (newValue && newValue.length > 0) {
    localStorage.setItem(STORAGE_REGION_KEY, JSON.stringify(newValue))
  } else {
    localStorage.removeItem(STORAGE_REGION_KEY)
  }
}, { deep: true })

// 获取云账号列表
async function fetchCloudAccountList() {
  try {
    const res = await getCloudAccountOptions("aliyun")
    cloudAccountList.value = res.data || []
  } catch (error) {
    console.error("获取云账号列表失败", error)
    ElMessage.error("获取云账号列表失败")
  }
}

// 处理云账号切换
function handleCloudAccountChange() {
  selectedRegions.value = []
  imageList.value = []

  if (selectedCloudAccount.value) {
    fetchRegionList()
  }
}

// 获取区域列表
async function fetchRegionList() {
  try {
    const res = await regionListApi()
    regionList.value = res.data
  } catch (error) {
    console.error("获取区域列表失败", error)
    ElMessage.error("获取区域列表失败")
  }
}

// 获取镜像列表
async function fetchImageList() {
  if (!selectedCloudAccount.value || !selectedRegions.value || selectedRegions.value.length === 0) {
    return
  }

  loading.value = true
  try {
    const res = await getImageList({
      cloud_account_id: selectedCloudAccount.value,
      region_id: selectedRegions.value
    })

    imageList.value = res.data?.list || []

    if (imageList.value.length === 0) {
      ElMessage.info("当前区域暂无自定义镜像")
    }
  } catch (error) {
    console.error("获取镜像列表失败", error)
    ElMessage.error("获取镜像列表失败")
    imageList.value = []
  } finally {
    loading.value = false
  }
}

// 区域变化处理
function handleRegionChange() {
  imageList.value = []
}

// 获取镜像状态文本
function getStatusText(status: string) {
  const statusMap: Record<string, string> = {
    Available: "可用",
    Creating: "创建中",
    Waiting: "等待中",
    CreateFailed: "创建失败",
    Deprecated: "已弃用"
  }
  return statusMap[status] || status
}

// 获取镜像状态类型
function getStatusType(status: string) {
  const typeMap: Record<string, "success" | "warning" | "info" | "danger"> = {
    Available: "success",
    Creating: "warning",
    Waiting: "info",
    CreateFailed: "danger",
    Deprecated: "info"
  }
  return typeMap[status] || "info"
}

// 打开共享对话框
async function openShareDialog(image: Image) {
  currentShareImage.value = image
  shareDialogVisible.value = true
  shareLoading.value = true
  newShareAccount.value = ""

  try {
    const res = await describeImageSharePermission({
      cloud_account_id: selectedCloudAccount.value,
      region_id: image.RegionId || selectedRegions.value[0],
      image_id: image.ImageId
    })

    shareAccounts.value = res.data?.share_accounts || []
  } catch (error) {
    console.error("获取共享权限失败", error)
    ElMessage.error("获取共享权限失败")
    shareAccounts.value = []
  } finally {
    shareLoading.value = false
  }
}

// 添加共享账号
async function addShareAccount() {
  if (!newShareAccount.value.trim()) {
    ElMessage.warning("请输入阿里云账号ID")
    return
  }

  if (!currentShareImage.value) return

  // 验证账号ID格式（阿里云账号ID通常是数字）
  if (!/^\d+$/.test(newShareAccount.value.trim())) {
    ElMessage.warning("请输入有效的阿里云账号ID（纯数字）")
    return
  }

  shareLoading.value = true
  try {
    await modifyImageSharePermission({
      cloud_account_id: selectedCloudAccount.value,
      region_id: currentShareImage.value.RegionId || selectedRegions.value[0],
      image_id: currentShareImage.value.ImageId,
      add_accounts: [newShareAccount.value.trim()]
    })

    ElMessage.success("添加共享账号成功")
    newShareAccount.value = ""

    // 刷新共享列表
    const res = await describeImageSharePermission({
      cloud_account_id: selectedCloudAccount.value,
      region_id: currentShareImage.value.RegionId || selectedRegions.value[0],
      image_id: currentShareImage.value.ImageId
    })
    shareAccounts.value = res.data?.share_accounts || []
  } catch (error) {
    console.error("添加共享账号失败", error)
    ElMessage.error("添加共享账号失败")
  } finally {
    shareLoading.value = false
  }
}

// 移除共享账号
async function removeShareAccount(account: ShareAccount) {
  if (!currentShareImage.value) return

  ElMessageBox.confirm(
    `确定要取消与账号 ${account.aliyun_id} 的共享吗？`,
    "取消共享确认",
    {
      confirmButtonText: "确定",
      cancelButtonText: "取消",
      type: "warning"
    }
  ).then(async () => {
    shareLoading.value = true
    try {
      await modifyImageSharePermission({
        cloud_account_id: selectedCloudAccount.value,
        region_id: currentShareImage.value!.RegionId || selectedRegions.value[0],
        image_id: currentShareImage.value!.ImageId,
        remove_accounts: [account.aliyun_id]
      })

      ElMessage.success("取消共享成功")

      // 刷新共享列表
      const res = await describeImageSharePermission({
        cloud_account_id: selectedCloudAccount.value,
        region_id: currentShareImage.value!.RegionId || selectedRegions.value[0],
        image_id: currentShareImage.value!.ImageId
      })
      shareAccounts.value = res.data?.share_accounts || []
    } catch (error) {
      console.error("取消共享失败", error)
      ElMessage.error("取消共享失败")
    } finally {
      shareLoading.value = false
    }
  }).catch(() => {})
}

// 页面加载
onMounted(() => {
  loadSelectionFromStorage()
  fetchCloudAccountList().then(() => {
    if (selectedCloudAccount.value) {
      fetchRegionList()
    }
  })
})
</script>

<template>
  <div class="container">
    <el-card class="filter-card">
      <div class="filter-row">
        <div class="filter-item">
          <span class="label">云账号：</span>
          <el-select
            v-model="selectedCloudAccount"
            placeholder="请选择云账号"
            clearable
            filterable
            style="width: 220px"
            @change="handleCloudAccountChange"
          >
            <template #prefix>
              <el-icon><Key /></el-icon>
            </template>
            <el-option
              v-for="item in cloudAccountList"
              :key="item.value"
              :label="item.label"
              :value="item.value"
            />
          </el-select>
        </div>
        <div class="filter-item">
          <span class="label">区域：</span>
          <el-select
            v-model="selectedRegions"
            placeholder="请选择区域（可多选）"
            clearable
            filterable
            multiple
            collapse-tags
            collapse-tags-tooltip
            style="width: 300px"
            :disabled="!selectedCloudAccount"
            @change="handleRegionChange"
          >
            <template #prefix>
              <el-icon><Location /></el-icon>
            </template>
            <el-option
              v-for="item in regionList"
              :key="item.RegionId"
              :label="item.LocalName"
              :value="item.RegionId"
            />
          </el-select>
        </div>
        <el-button
          type="primary"
          :disabled="!selectedCloudAccount || !selectedRegions.length"
          @click="fetchImageList"
        >
          查询
        </el-button>
      </div>
    </el-card>

    <el-card v-if="selectedCloudAccount && selectedRegions.length > 0" class="table-card">
      <template #header>
        <div class="card-header">
          <span>镜像列表（自定义镜像）</span>
          <el-button type="primary" size="small" @click="fetchImageList">
            刷新
          </el-button>
        </div>
      </template>
      <el-table
        v-loading="loading"
        :data="imageList"
        style="width: 100%"
        border
      >
        <el-table-column prop="ImageId" label="镜像ID" min-width="220" />
        <el-table-column prop="ImageName" label="镜像名称" min-width="180" />
        <el-table-column prop="OSName" label="操作系统" min-width="200" />
        <el-table-column prop="Status" label="状态" min-width="100">
          <template #default="scope">
            <el-tag :type="getStatusType(scope.row.Status)">
              {{ getStatusText(scope.row.Status) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="Progress" label="进度" min-width="100">
          <template #default="scope">
            {{ scope.row.Progress || "100%" }}
          </template>
        </el-table-column>
        <el-table-column prop="CreationTime" label="创建时间" min-width="180" />
        <el-table-column label="操作" fixed="right" min-width="120">
          <template #default="scope">
            <el-button
              type="primary"
              text
              size="small"
              :icon="Share"
              @click="openShareDialog(scope.row)"
            >
              共享
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-empty v-else description="请先选择云账号和区域" />

    <!-- 镜像共享对话框 -->
    <el-dialog
      v-model="shareDialogVisible"
      :title="`镜像共享 - ${currentShareImage?.ImageName || ''}`"
      width="600px"
      destroy-on-close
    >
      <div v-loading="shareLoading" class="share-dialog-content">
        <div class="share-info">
          <el-descriptions :column="1" border size="small">
            <el-descriptions-item label="镜像ID">
              {{ currentShareImage?.ImageId }}
            </el-descriptions-item>
            <el-descriptions-item label="镜像名称">
              {{ currentShareImage?.ImageName }}
            </el-descriptions-item>
          </el-descriptions>
        </div>

        <div class="share-add-section">
          <h4>添加共享账号</h4>
          <div class="add-account-row">
            <el-input
              v-model="newShareAccount"
              placeholder="请输入阿里云账号ID"
              style="flex: 1"
              @keyup.enter="addShareAccount"
            />
            <el-button
              type="primary"
              :icon="Plus"
              @click="addShareAccount"
              :loading="shareLoading"
            >
              添加
            </el-button>
          </div>
          <div class="tip">
            提示：阿里云账号ID可在目标账号的「账号管理」页面查看
          </div>
        </div>

        <div class="share-list-section">
          <h4>已共享账号 ({{ shareAccounts.length }})</h4>
          <el-table :data="shareAccounts" border size="small" v-if="shareAccounts.length > 0">
            <el-table-column prop="aliyun_id" label="阿里云账号ID" />
            <el-table-column label="操作" width="100" fixed="right">
              <template #default="scope">
                <el-button
                  type="danger"
                  text
                  size="small"
                  :icon="Delete"
                  @click="removeShareAccount(scope.row)"
                >
                  取消
                </el-button>
              </template>
            </el-table-column>
          </el-table>
          <el-empty v-else description="暂无共享账号" :image-size="60" />
        </div>
      </div>

      <template #footer>
        <el-button @click="shareDialogVisible = false">
          关闭
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.container {
  padding: 16px;
}

.filter-card {
  margin-bottom: 16px;
}

.filter-row {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 16px;
}

.filter-item {
  display: flex;
  align-items: center;
  margin-right: 16px;
}

.label {
  margin-right: 8px;
  white-space: nowrap;
  font-weight: 500;
  color: #606266;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.share-dialog-content {
  min-height: 200px;
}

.share-info {
  margin-bottom: 20px;
}

.share-add-section {
  margin-bottom: 20px;
}

.share-add-section h4 {
  margin-bottom: 10px;
  font-size: 14px;
  font-weight: 500;
  color: #303133;
}

.add-account-row {
  display: flex;
  gap: 10px;
}

.tip {
  margin-top: 8px;
  font-size: 12px;
  color: #909399;
}

.share-list-section h4 {
  margin-bottom: 10px;
  font-size: 14px;
  font-weight: 500;
  color: #303133;
}
</style>
