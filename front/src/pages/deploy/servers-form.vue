<script lang="ts" setup>
import type { ServerResp } from "@@/apis/deploy/type"
import { createServer, getServerDetail, testConnection, updateServer } from "@@/apis/deploy"

defineOptions({ name: "DeployServersForm" })

const router = useRouter()
const route = useRoute()

const isEdit = computed(() => !!route.params.id)
const pageTitle = computed(() => isEdit.value ? "编辑服务器" : "新增服务器")

const loading = ref(false)

// 表单数据
const formData = reactive({
  server_type: 1,
  forward_type: 1, // 转发类型：1-加密(relay+tls) 2-直连(tcp)，仅系统服务器有效
  name: "",
  host: "",
  auxiliary_ip: "", // 辅助IP，仅系统服务器使用
  port: 22,
  username: "root",
  auth_type: 1, // 1-密码 2-密钥
  password: "",
  private_key: "",
  description: ""
})

// 加载服务器详情（编辑模式）
async function loadServerDetail() {
  if (!isEdit.value)
    return
  loading.value = true
  try {
    const id = Number(route.params.id)
    const res = await getServerDetail(id)
    const data: ServerResp = res.data
    formData.server_type = data.server_type
    formData.forward_type = data.forward_type || 1
    formData.name = data.name
    formData.host = data.host
    formData.auxiliary_ip = data.auxiliary_ip || ""
    formData.port = data.port
    formData.username = data.username
    formData.auth_type = data.auth_type
    formData.description = data.description
  }
  catch {
    ElMessage.error("加载服务器信息失败")
    router.back()
  }
  finally {
    loading.value = false
  }
}

// 测试连接
async function onTestConnection() {
  if (!formData.host || !formData.port || !formData.username) {
    return ElMessage.warning("请先填写主机地址、端口和用户名")
  }

  if (formData.auth_type === 1 && !formData.password && !isEdit.value) {
    return ElMessage.warning("请填写密码")
  }

  if (formData.auth_type === 2 && !formData.private_key && !isEdit.value) {
    return ElMessage.warning("请填写私钥")
  }

  loading.value = true
  try {
    await testConnection({
      host: formData.host,
      port: formData.port,
      username: formData.username,
      auth_type: formData.auth_type,
      password: formData.auth_type === 1 ? formData.password : "",
      private_key: formData.auth_type === 2 ? formData.private_key : ""
    })
    ElMessage.success("连接测试成功!")
  }
  catch {
    ElMessage.error("连接测试失败，请检查配置")
  }
  finally {
    loading.value = false
  }
}

// 提交表单
async function onSubmit() {
  if (!formData.name || !formData.host || !formData.port || !formData.username) {
    return ElMessage.warning("请填写必填项")
  }

  // 新增时必须提供认证信息
  if (!isEdit.value) {
    if (formData.auth_type === 1 && !formData.password) {
      return ElMessage.warning("请填写SSH密码")
    }
    if (formData.auth_type === 2 && !formData.private_key) {
      return ElMessage.warning("请填写SSH私钥")
    }
  }

  loading.value = true
  try {
    const submitData: Record<string, unknown> = {
      ...formData,
      port: Number(formData.port),
      auth_type: Number(formData.auth_type),
      server_type: Number(formData.server_type),
      forward_type: Number(formData.forward_type)
    }

    // 根据认证方式清空不需要的字段
    if (submitData.auth_type === 1) {
      submitData.private_key = ""
    }
    else if (submitData.auth_type === 2) {
      submitData.password = ""
    }

    if (isEdit.value) {
      await updateServer(Number(route.params.id), submitData as any)
      ElMessage.success("更新成功")
    }
    else {
      await createServer(submitData as any)
      ElMessage.success("创建成功")
    }

    router.push({ name: "DeployServers" })
  }
  catch {
    ElMessage.error(isEdit.value ? "更新失败" : "创建失败")
  }
  finally {
    loading.value = false
  }
}

// 取消返回
function onCancel() {
  router.back()
}

onMounted(() => {
  loadServerDetail()
})
</script>

<template>
  <div class="app-container">
    <el-card v-loading="loading">
      <template #header>
        <div class="card-header">
          <span class="title">{{ pageTitle }}</span>
          <div class="actions">
            <el-button @click="onCancel">
              取消
            </el-button>
            <el-button type="warning" @click="onTestConnection">
              测试连接
            </el-button>
            <el-button type="primary" @click="onSubmit">
              保存
            </el-button>
          </div>
        </div>
      </template>

      <el-form :model="formData" label-width="120px" class="server-form">
        <!-- 服务器类型 -->
        <el-form-item label="服务器类型" required>
          <el-radio-group v-model="formData.server_type">
            <el-radio :value="1">
              商户服务器
            </el-radio>
            <el-radio :value="2">
              系统服务器
            </el-radio>
          </el-radio-group>
        </el-form-item>

        <!-- 转发类型（仅系统服务器显示） -->
        <el-form-item v-if="formData.server_type === 2" label="转发类型" required>
          <el-radio-group v-model="formData.forward_type">
            <el-radio :value="1">
              加密 (relay+tls)
            </el-radio>
            <el-radio :value="2">
              直连 (tcp)
            </el-radio>
          </el-radio-group>
          <div class="form-item-tip">
            V2架构：统一端口 443(TLS) → 商户:10443(nginx路径分发) + TCP:10010<br>
            加密：relay+tls 全链路加密（推荐）<br>
            直连：TCP 直连，无加密
          </div>
        </el-form-item>

        <!-- 服务器名称 -->
        <el-form-item label="服务器名称" required>
          <el-input v-model="formData.name" placeholder="请输入服务器名称" />
        </el-form-item>

        <!-- 主机地址 -->
        <el-form-item label="主机地址" required>
          <el-input v-model="formData.host" placeholder="请输入IP地址" />
        </el-form-item>

        <!-- 辅助IP（仅系统服务器显示） -->
        <el-form-item v-if="formData.server_type === 2" label="辅助IP">
          <el-input v-model="formData.auxiliary_ip" placeholder="请输入辅助IP（用于IP内嵌）" />
          <div class="form-item-tip">
            可选，用于IP批量内嵌上传功能
          </div>
        </el-form-item>

        <!-- SSH端口 -->
        <el-form-item label="SSH端口" required>
          <el-input-number v-model="formData.port" :min="1" :max="65535" placeholder="默认22" style="width: 100%" />
        </el-form-item>

        <!-- SSH用户名 -->
        <el-form-item label="SSH用户名" required>
          <el-input v-model="formData.username" placeholder="默认root" />
        </el-form-item>

        <!-- 认证方式 -->
        <el-form-item label="认证方式" required>
          <el-radio-group v-model="formData.auth_type">
            <el-radio :value="1">
              密码
            </el-radio>
            <el-radio :value="2">
              密钥
            </el-radio>
          </el-radio-group>
        </el-form-item>

        <!-- SSH密码 -->
        <el-form-item v-if="formData.auth_type === 1" label="SSH密码" :required="!isEdit">
          <el-input v-model="formData.password" type="password" placeholder="请输入密码" show-password />
          <div v-if="isEdit" class="form-item-tip">
            编辑时留空表示不修改密码
          </div>
        </el-form-item>

        <!-- SSH私钥 -->
        <el-form-item v-if="formData.auth_type === 2" label="SSH私钥" :required="!isEdit">
          <el-input v-model="formData.private_key" type="textarea" :rows="8" placeholder="请粘贴私钥内容" />
          <div v-if="isEdit" class="form-item-tip">
            编辑时留空表示不修改私钥
          </div>
        </el-form-item>

        <!-- 描述 -->
        <el-form-item label="描述">
          <el-input v-model="formData.description" type="textarea" :rows="3" placeholder="请输入描述" />
        </el-form-item>
      </el-form>
    </el-card>
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

  .title {
    font-size: 18px;
    font-weight: 600;
  }

  .actions {
    display: flex;
    gap: 12px;
  }
}

.server-form {
  max-width: 800px;
  margin: 20px 0;
}

.form-item-tip {
  font-size: 12px;
  color: #909399;
  margin-top: 4px;
}
</style>
