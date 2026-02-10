<script lang="ts" setup>
import type { TwoFASetupResp } from "@/common/apis/twofa/type"
import { disableTwoFAApi, enableTwoFAApi, getTwoFASetupApi, getTwoFAStatusApi } from "@/common/apis/twofa"
import { Key } from "@element-plus/icons-vue"
import { ElMessage, ElMessageBox } from "element-plus"

const router = useRouter()

const loading = ref(false)
const statusLoading = ref(false)
const setupInfo = ref<TwoFASetupResp | null>(null)
const is2FAEnabled = ref(false)

// 表单数据
const verificationCode = ref("")

// 加载2FA状态（未使用但保留以备将来使用）
function _loadStatus() {
  statusLoading.value = true
  getTwoFAStatusApi()
    .then(({ data }: { data: { enabled: boolean } }) => {
      is2FAEnabled.value = data.enabled
      if (data.enabled) {
        ElMessage.success("2FA已启用")
      }
    })
    .catch((error: Error) => {
      console.error("获取2FA状态失败:", error)
    })
    .finally(() => {
      statusLoading.value = false
    })
}

// 加载设置信息（二维码等）
function loadSetupInfo() {
  loading.value = true
  getTwoFASetupApi()
    .then(({ data }) => {
      setupInfo.value = data
      is2FAEnabled.value = data.enabled
    })
    .catch((error) => {
      console.error("获取2FA设置信息失败:", error)
      ElMessage.error("获取2FA设置信息失败")
    })
    .finally(() => {
      loading.value = false
    })
}

// 启用2FA
function handleEnable() {
  if (!verificationCode.value) {
    ElMessage.warning("请输入验证码")
    return
  }
  if (!/^\d{6}$/.test(verificationCode.value)) {
    ElMessage.warning("验证码必须是6位数字")
    return
  }

  loading.value = true
  enableTwoFAApi({ code: verificationCode.value })
    .then(() => {
      ElMessage.success("2FA已成功启用")
      is2FAEnabled.value = true
      verificationCode.value = ""
      // 跳转到首页
      setTimeout(() => {
        router.push("/")
      }, 1500)
    })
    .catch((error) => {
      console.error("启用2FA失败:", error)
      ElMessage.error("验证码错误或启用失败")
    })
    .finally(() => {
      loading.value = false
    })
}

// 禁用2FA
function handleDisable() {
  ElMessageBox.prompt("请输入管理员密码以禁用2FA", "确认禁用", {
    confirmButtonText: "确定",
    cancelButtonText: "取消",
    inputType: "password",
    inputPlaceholder: "请输入密码",
    inputValidator: (value) => {
      if (!value) {
        return "密码不能为空"
      }
      return true
    }
  })
    .then(({ value }) => {
      loading.value = true
      return disableTwoFAApi({ password: value })
    })
    .then(() => {
      ElMessage.success("2FA已禁用")
      is2FAEnabled.value = false
      setupInfo.value = null
      // 重新加载设置信息
      loadSetupInfo()
    })
    .catch((error) => {
      if (error !== "cancel") {
        console.error("禁用2FA失败:", error)
        ElMessage.error("密码错误或禁用失败")
      }
    })
    .finally(() => {
      loading.value = false
    })
}

// 跳过设置
function handleSkip() {
  router.push("/")
}

// 复制密钥到剪贴板
async function copySecret() {
  if (setupInfo.value?.secret) {
    try {
      await window.navigator.clipboard.writeText(setupInfo.value.secret)
      ElMessage.success("已复制到剪贴板")
    }
    catch {
      ElMessage.error("复制失败")
    }
  }
}

// 页面加载时获取设置信息
onMounted(() => {
  loadSetupInfo()
})
</script>

<template>
  <div class="twofa-container">
    <el-card v-loading="loading || statusLoading" class="twofa-card">
      <template #header>
        <div class="card-header">
          <span class="title">双因素认证 (2FA)</span>
        </div>
      </template>

      <div v-if="!is2FAEnabled" class="setup-section">
        <el-alert
          title="建议启用2FA"
          type="warning"
          description="双因素认证可以大大提高您的账户安全性，建议启用。"
          :closable="false"
          show-icon
          class="alert-box"
        />

        <div class="qrcode-section">
          <h3>步骤 1：使用认证应用扫描二维码</h3>
          <p class="tip">请使用 Google Authenticator、Microsoft Authenticator 或其他TOTP认证应用扫描下方二维码</p>

          <div v-if="setupInfo" class="qrcode-wrapper">
            <img
              :src="`https://api.qrserver.com/v1/create-qr-code/?size=200x200&data=${encodeURIComponent(setupInfo.qr_code)}`"
              alt="2FA QR Code"
              class="qrcode-image"
            >
            <div class="secret-info">
              <p>或手动输入密钥：</p>
              <el-input
                :model-value="setupInfo.secret"
                readonly
                class="secret-input"
              >
                <template #append>
                  <el-button @click="copySecret">
                    复制
                  </el-button>
                </template>
              </el-input>
            </div>
          </div>
        </div>

        <div class="verify-section">
          <h3>步骤 2：输入验证码</h3>
          <p class="tip">请输入认证应用中显示的6位验证码</p>
          <el-input
            v-model="verificationCode"
            placeholder="请输入6位验证码"
            :prefix-icon="Key"
            maxlength="6"
            size="large"
            class="code-input"
            @keyup.enter="handleEnable"
          />
        </div>

        <div class="action-buttons">
          <el-button type="primary" size="large" :loading="loading" @click="handleEnable">
            启用2FA
          </el-button>
          <el-button size="large" @click="handleSkip">
            暂时跳过
          </el-button>
        </div>
      </div>

      <div v-else class="enabled-section">
        <el-result icon="success" title="2FA已启用" sub-title="您的账户已受到双因素认证保护">
          <template #extra>
            <el-space direction="vertical" :size="20">
              <el-alert
                title="注意"
                type="info"
                description="禁用2FA将降低账户安全性，请谨慎操作。"
                :closable="false"
                show-icon
              />
              <el-button type="danger" :loading="loading" @click="handleDisable">
                解绑2FA
              </el-button>
              <el-button type="primary" @click="handleSkip">
                返回首页
              </el-button>
            </el-space>
          </template>
        </el-result>
      </div>
    </el-card>
  </div>
</template>

<style lang="scss" scoped>
.twofa-container {
  display: flex;
  justify-content: center;
  align-items: center;
  min-height: 100vh;
  padding: 20px;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
}

.twofa-card {
  width: 100%;
  max-width: 600px;
  border-radius: 12px;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.1);
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;

  .title {
    font-size: 20px;
    font-weight: 600;
  }
}

.setup-section {
  .alert-box {
    margin-bottom: 30px;
  }

  .qrcode-section {
    margin-bottom: 30px;

    h3 {
      margin-bottom: 10px;
      font-size: 16px;
      font-weight: 600;
    }

    .tip {
      margin-bottom: 20px;
      color: #909399;
      font-size: 14px;
    }

    .qrcode-wrapper {
      display: flex;
      flex-direction: column;
      align-items: center;

      .qrcode-image {
        width: 200px;
        height: 200px;
        margin-bottom: 20px;
        border: 1px solid #dcdfe6;
        border-radius: 8px;
      }

      .secret-info {
        width: 100%;

        p {
          margin-bottom: 10px;
          font-size: 14px;
          color: #606266;
        }

        .secret-input {
          max-width: 400px;
        }
      }
    }
  }

  .verify-section {
    margin-bottom: 30px;

    h3 {
      margin-bottom: 10px;
      font-size: 16px;
      font-weight: 600;
    }

    .tip {
      margin-bottom: 15px;
      color: #909399;
      font-size: 14px;
    }

    .code-input {
      max-width: 300px;
    }
  }

  .action-buttons {
    display: flex;
    gap: 15px;
    justify-content: center;
  }
}

.enabled-section {
  padding: 20px 0;
}
</style>
