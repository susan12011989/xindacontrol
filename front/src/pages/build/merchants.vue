<script setup lang="ts">
import { ref, onMounted } from "vue"
import { ElMessage, ElMessageBox } from "element-plus"
import { getBuildMerchants, createBuildMerchant, updateBuildMerchant, deleteBuildMerchant } from "@/common/apis/build"
import type { BuildMerchant, BuildMerchantReq } from "@/common/apis/build/type"

const merchants = ref<BuildMerchant[]>([])
const total = ref(0)
const loading = ref(false)
const page = ref(1)
const size = ref(20)
const nameFilter = ref("")

// 编辑弹窗
const showDialog = ref(false)
const isEdit = ref(false)
const editId = ref(0)
const form = ref<BuildMerchantReq>({
  name: "",
  app_name: "",
  short_name: "",
  android_package: "",
  ios_bundle_id: "",
})

// 加载列表
async function loadData() {
  loading.value = true
  try {
    const res = await getBuildMerchants({
      page: page.value,
      size: size.value,
      name: nameFilter.value,
    })
    merchants.value = res.data.list || []
    total.value = res.data.total
  } finally {
    loading.value = false
  }
}

// 打开新建弹窗
function openCreate() {
  isEdit.value = false
  editId.value = 0
  form.value = {
    name: "",
    app_name: "",
    short_name: "",
    android_package: "",
    android_version_code: 1,
    android_version_name: "1.0.0",
    ios_bundle_id: "",
    ios_version: "1.0.0",
    ios_build: "1",
    server_ws_port: 5100,
  }
  showDialog.value = true
}

// 打开编辑弹窗
function openEdit(row: BuildMerchant) {
  isEdit.value = true
  editId.value = row.id
  form.value = {
    merchant_id: row.merchant_id,
    name: row.name,
    app_name: row.app_name,
    short_name: row.short_name,
    description: row.description,
    android_package: row.android_package,
    android_version_code: row.android_version_code,
    android_version_name: row.android_version_name,
    ios_bundle_id: row.ios_bundle_id,
    ios_version: row.ios_version,
    ios_build: row.ios_build,
    windows_app_name: row.windows_app_name,
    windows_version: row.windows_version,
    macos_bundle_id: row.macos_bundle_id,
    macos_app_name: row.macos_app_name,
    macos_version: row.macos_version,
    server_api_url: row.server_api_url,
    server_ws_host: row.server_ws_host,
    server_ws_port: row.server_ws_port,
    enterprise_code: row.enterprise_code,
    push_mi_app_id: row.push_mi_app_id,
    push_mi_app_key: row.push_mi_app_key,
    push_oppo_app_key: row.push_oppo_app_key,
    push_oppo_app_secret: row.push_oppo_app_secret,
    push_vivo_app_id: row.push_vivo_app_id,
    push_vivo_app_key: row.push_vivo_app_key,
    push_hms_app_id: row.push_hms_app_id,
    // Apple 开发者配置
    apple_team_id: row.apple_team_id,
    apple_certificate_url: row.apple_certificate_url,
    apple_provisioning_url: row.apple_provisioning_url,
    apple_mac_provisioning_url: row.apple_mac_provisioning_url,
    apple_export_method: row.apple_export_method,
    // Git 源码配置
    git_repo_url: row.git_repo_url,
    git_branch: row.git_branch,
    git_tag: row.git_tag,
    git_username: row.git_username,
  }
  showDialog.value = true
}

// 提交表单
async function submitForm() {
  if (!form.value.name || !form.value.app_name || !form.value.android_package || !form.value.ios_bundle_id) {
    ElMessage.warning("请填写必填项")
    return
  }

  try {
    if (isEdit.value) {
      await updateBuildMerchant(editId.value, form.value)
      ElMessage.success("更新成功")
    } else {
      await createBuildMerchant(form.value)
      ElMessage.success("创建成功")
    }
    showDialog.value = false
    loadData()
  } catch (e: any) {
    ElMessage.error(e.message || "操作失败")
  }
}

// 删除
async function handleDelete(row: BuildMerchant) {
  await ElMessageBox.confirm(`确定要删除配置"${row.name}"吗？`, "提示", { type: "warning" })
  await deleteBuildMerchant(row.id)
  ElMessage.success("删除成功")
  loadData()
}

// 分页
function handlePageChange(p: number) {
  page.value = p
  loadData()
}

onMounted(() => {
  loadData()
})
</script>

<template>
  <div class="app-container">
    <!-- 搜索栏 -->
    <el-card class="filter-card">
      <el-row :gutter="20">
        <el-col :span="6">
          <el-input v-model="nameFilter" placeholder="搜索配置名称" clearable @keyup.enter="loadData" />
        </el-col>
        <el-col :span="6">
          <el-button type="primary" @click="loadData">搜索</el-button>
          <el-button @click="nameFilter = ''; loadData()">重置</el-button>
        </el-col>
        <el-col :span="12" style="text-align: right">
          <el-button type="primary" @click="openCreate">
            <el-icon><Plus /></el-icon>
            新建配置
          </el-button>
        </el-col>
      </el-row>
    </el-card>

    <!-- 列表 -->
    <el-card>
      <el-table :data="merchants" v-loading="loading" stripe>
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column label="图标" width="80">
          <template #default="{ row }">
            <el-avatar v-if="row.icon_url" :src="row.icon_url" :size="40" />
            <el-avatar v-else :size="40">{{ row.short_name?.charAt(0) }}</el-avatar>
          </template>
        </el-table-column>
        <el-table-column prop="name" label="配置名称" min-width="120" />
        <el-table-column prop="app_name" label="应用名称" min-width="120" />
        <el-table-column prop="android_package" label="Android包名" min-width="180" />
        <el-table-column prop="ios_bundle_id" label="iOS Bundle ID" min-width="180" />
        <el-table-column label="状态" width="80">
          <template #default="{ row }">
            <el-tag :type="row.status === 1 ? 'success' : 'info'" size="small">
              {{ row.status === 1 ? '启用' : '禁用' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="created_at" label="创建时间" width="180" />
        <el-table-column label="操作" width="150" fixed="right">
          <template #default="{ row }">
            <el-button type="primary" size="small" link @click="openEdit(row)">编辑</el-button>
            <el-button type="danger" size="small" link @click="handleDelete(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>

      <el-pagination
        v-if="total > size"
        class="pagination"
        layout="total, prev, pager, next"
        :total="total"
        :page-size="size"
        :current-page="page"
        @current-change="handlePageChange"
      />
    </el-card>

    <!-- 编辑弹窗 -->
    <el-dialog v-model="showDialog" :title="isEdit ? '编辑配置' : '新建配置'" width="700px">
      <el-form :model="form" label-width="120px">
        <el-divider content-position="left">基本信息</el-divider>
        <el-row :gutter="20">
          <el-col :span="12">
            <el-form-item label="配置名称" required>
              <el-input v-model="form.name" placeholder="用于识别，如：商户A" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="应用名称" required>
              <el-input v-model="form.app_name" placeholder="应用显示名称" />
            </el-form-item>
          </el-col>
        </el-row>
        <el-row :gutter="20">
          <el-col :span="12">
            <el-form-item label="应用短名称">
              <el-input v-model="form.short_name" placeholder="桌面图标名称" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="企业号">
              <el-input v-model="form.enterprise_code" placeholder="6位数字企业号" />
            </el-form-item>
          </el-col>
        </el-row>

        <el-divider content-position="left">Android 配置</el-divider>
        <el-row :gutter="20">
          <el-col :span="12">
            <el-form-item label="包名" required>
              <el-input v-model="form.android_package" placeholder="com.example.app" />
            </el-form-item>
          </el-col>
          <el-col :span="6">
            <el-form-item label="版本号">
              <el-input-number v-model="form.android_version_code" :min="1" style="width: 100%" />
            </el-form-item>
          </el-col>
          <el-col :span="6">
            <el-form-item label="版本名">
              <el-input v-model="form.android_version_name" placeholder="1.0.0" />
            </el-form-item>
          </el-col>
        </el-row>

        <el-divider content-position="left">iOS 配置</el-divider>
        <el-row :gutter="20">
          <el-col :span="12">
            <el-form-item label="Bundle ID" required>
              <el-input v-model="form.ios_bundle_id" placeholder="com.example.app" />
            </el-form-item>
          </el-col>
          <el-col :span="6">
            <el-form-item label="版本">
              <el-input v-model="form.ios_version" placeholder="1.0.0" />
            </el-form-item>
          </el-col>
          <el-col :span="6">
            <el-form-item label="Build">
              <el-input v-model="form.ios_build" placeholder="1" />
            </el-form-item>
          </el-col>
        </el-row>

        <el-divider content-position="left">Apple 开发者配置</el-divider>
        <el-row :gutter="20">
          <el-col :span="12">
            <el-form-item label="Team ID">
              <el-input v-model="form.apple_team_id" placeholder="Apple 开发者团队 ID" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="导出方式">
              <el-select v-model="form.apple_export_method" placeholder="选择导出方式" style="width: 100%">
                <el-option label="App Store" value="app-store" />
                <el-option label="Ad Hoc" value="ad-hoc" />
                <el-option label="Enterprise" value="enterprise" />
                <el-option label="Development" value="development" />
              </el-select>
            </el-form-item>
          </el-col>
        </el-row>
        <el-row :gutter="20">
          <el-col :span="12">
            <el-form-item label="P12证书URL">
              <el-input v-model="form.apple_certificate_url" placeholder="签名证书P12文件URL" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="证书密码">
              <el-input v-model="form.apple_certificate_password" type="password" show-password placeholder="P12证书密码" />
            </el-form-item>
          </el-col>
        </el-row>
        <el-row :gutter="20">
          <el-col :span="12">
            <el-form-item label="iOS描述文件">
              <el-input v-model="form.apple_provisioning_url" placeholder="iOS Provisioning Profile URL" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="macOS描述文件">
              <el-input v-model="form.apple_mac_provisioning_url" placeholder="macOS Provisioning Profile URL" />
            </el-form-item>
          </el-col>
        </el-row>

        <el-divider content-position="left">Git 源码配置</el-divider>
        <el-row :gutter="20">
          <el-col :span="24">
            <el-form-item label="Git仓库地址">
              <el-input v-model="form.git_repo_url" placeholder="https://github.com/your-org/your-repo.git" />
            </el-form-item>
          </el-col>
        </el-row>
        <el-row :gutter="20">
          <el-col :span="12">
            <el-form-item label="分支名">
              <el-input v-model="form.git_branch" placeholder="main（默认）" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="Tag版本">
              <el-input v-model="form.git_tag" placeholder="指定tag（优先于分支）" />
            </el-form-item>
          </el-col>
        </el-row>
        <el-row :gutter="20">
          <el-col :span="12">
            <el-form-item label="Git用户名">
              <el-input v-model="form.git_username" placeholder="私有仓库用户名（可选）" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="Git Token">
              <el-input v-model="form.git_token" type="password" show-password placeholder="私有仓库Token/密码（可选）" />
            </el-form-item>
          </el-col>
        </el-row>

        <el-divider content-position="left">服务器配置</el-divider>
        <el-row :gutter="20">
          <el-col :span="12">
            <el-form-item label="API地址">
              <el-input v-model="form.server_api_url" placeholder="https://api.example.com" />
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="WS主机">
              <el-input v-model="form.server_ws_host" placeholder="ws.example.com" />
            </el-form-item>
          </el-col>
          <el-col :span="4">
            <el-form-item label="WS端口">
              <el-input-number v-model="form.server_ws_port" :min="1" :max="65535" style="width: 100%" />
            </el-form-item>
          </el-col>
        </el-row>

        <el-form-item label="备注">
          <el-input v-model="form.description" type="textarea" :rows="2" placeholder="可选备注" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showDialog = false">取消</el-button>
        <el-button type="primary" @click="submitForm">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped lang="scss">
.filter-card {
  margin-bottom: 20px;
}

.pagination {
  margin-top: 20px;
  justify-content: flex-end;
}
</style>
