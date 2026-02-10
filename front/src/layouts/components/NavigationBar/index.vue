<script lang="ts" setup>
import { updatePassword } from "@/common/apis/admin"
import { useAppStore } from "@/pinia/stores/app"
import { useSettingsStore } from "@/pinia/stores/settings"
import { useUserStore } from "@/pinia/stores/user"
// import Notify from "@@/components/Notify/index.vue"
import Screenfull from "@@/components/Screenfull/index.vue"
import SearchMenu from "@@/components/SearchMenu/index.vue"
import ThemeSwitch from "@@/components/ThemeSwitch/index.vue"
import { useDevice } from "@@/composables/useDevice"
import { useLayoutMode } from "@@/composables/useLayoutMode"
import { UserFilled } from "@element-plus/icons-vue"
import { Breadcrumb, Hamburger, Sidebar } from "../index"

const { isMobile } = useDevice()
const { isTop } = useLayoutMode()
const router = useRouter()
const appStore = useAppStore()
const userStore = useUserStore()
const settingsStore = useSettingsStore()
const { /* showNotify, */ showThemeSwitch, showScreenfull, showSearchMenu } = storeToRefs(settingsStore)

/** 切换侧边栏 */
function toggleSidebar() {
  appStore.toggleSidebar(false)
}

/** 登出 */
function logout() {
  userStore.logout()
  router.push("/login")
}

/** 跳转到2FA设置页面 */
function goto2FASettings() {
  router.push("/2fa-setup")
}

// 密码表单
const passwordForm = ref({
  old_password: "",
  new_password: "",
  confirm_password: ""
})

// 密码表单规则
const passwordRules = {
  old_password: [{ required: true, message: "请输入旧密码", trigger: "blur" }],
  new_password: [
    { required: true, message: "请输入新密码", trigger: "blur" },
    { min: 8, message: "密码长度不得少于8位", trigger: "blur" }
  ],
  confirm_password: [
    { required: true, message: "请确认新密码", trigger: "blur" },
    {
      validator: (rule: any, value: string, callback: (error?: Error) => void) => {
        if (value !== passwordForm.value.new_password) {
          callback(new Error("两次输入的密码不一致"))
        } else {
          callback()
        }
      },
      trigger: "blur"
    }
  ]
}

// 密码对话框可见性
const passwordDialogVisible = ref(false)
// 表单ref
const passwordFormRef = ref()

// 打开修改密码对话框
function openPasswordDialog() {
  passwordDialogVisible.value = true
  passwordForm.value = {
    old_password: "",
    new_password: "",
    confirm_password: ""
  }
}

// 提交修改密码
async function submitPasswordForm() {
  if (!passwordFormRef.value) return

  await passwordFormRef.value.validate(async (valid: boolean) => {
    if (valid) {
      try {
        await updatePassword({
          old_password: passwordForm.value.old_password,
          new_password: passwordForm.value.new_password
        })
        ElMessage.success("密码修改成功，请重新登录")
        passwordDialogVisible.value = false
        // 修改密码成功后自动退出登录
        setTimeout(() => {
          logout()
        }, 1500)
      } catch (error) {
        console.error(error)
      }
    }
  })
}
</script>

<template>
  <div class="navigation-bar">
    <Hamburger
      v-if="!isTop || isMobile"
      :is-active="appStore.sidebar.opened"
      class="hamburger"
      @toggle-click="toggleSidebar"
    />
    <Breadcrumb v-if="!isTop || isMobile" class="breadcrumb" />
    <Sidebar v-if="isTop && !isMobile" class="sidebar" />
    <div class="right-menu">
      <SearchMenu v-if="showSearchMenu" class="right-menu-item" />
      <Screenfull v-if="showScreenfull" class="right-menu-item" />
      <ThemeSwitch v-if="showThemeSwitch" class="right-menu-item" />
      <!-- <Notify v-if="showNotify" class="right-menu-item" /> -->
      <el-dropdown>
        <div class="right-menu-item user">
          <el-avatar :icon="UserFilled" :size="30" />
          <span>{{ userStore.username }}</span>
        </div>
        <template #dropdown>
          <el-dropdown-menu>
            <el-dropdown-item disabled>
              当前IP：{{ userStore.ip || '-' }}
            </el-dropdown-item>
            <el-dropdown-item divided @click="openPasswordDialog">
              修改密码
            </el-dropdown-item>
            <el-dropdown-item @click="goto2FASettings">
              编辑2FA
            </el-dropdown-item>
            <el-dropdown-item divided @click="logout">
              退出登录
            </el-dropdown-item>
          </el-dropdown-menu>
        </template>
      </el-dropdown>
    </div>
  </div>

  <!-- 修改密码对话框 -->
  <el-dialog
    v-model="passwordDialogVisible"
    title="修改密码"
    width="400px"
  >
    <el-form
      ref="passwordFormRef"
      :model="passwordForm"
      :rules="passwordRules"
      label-width="100px"
      label-position="right"
    >
      <el-form-item label="旧密码" prop="old_password">
        <el-input v-model="passwordForm.old_password" type="password" show-password placeholder="请输入旧密码" />
      </el-form-item>
      <el-form-item label="新密码" prop="new_password">
        <el-input v-model="passwordForm.new_password" type="password" show-password placeholder="请输入新密码" />
      </el-form-item>
      <el-form-item label="确认新密码" prop="confirm_password">
        <el-input v-model="passwordForm.confirm_password" type="password" show-password placeholder="请再次输入新密码" />
      </el-form-item>
    </el-form>
    <template #footer>
      <span class="dialog-footer">
        <el-button @click="passwordDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="submitPasswordForm">确认</el-button>
      </span>
    </template>
  </el-dialog>
</template>

<style lang="scss" scoped>
.navigation-bar {
  height: var(--v3-navigationbar-height);
  overflow: hidden;
  color: var(--v3-navigationbar-text-color);
  display: flex;
  justify-content: space-between;
  .hamburger {
    display: flex;
    align-items: center;
    height: 100%;
    padding: 0 15px;
    cursor: pointer;
  }
  .breadcrumb {
    flex: 1;
    // 参考 Bootstrap 的响应式设计将宽度设置为 576
    @media screen and (max-width: 576px) {
      display: none;
    }
  }
  .sidebar {
    flex: 1;
    // 设置 min-width 是为了让 Sidebar 里的 el-menu 宽度自适应
    min-width: 0px;
    :deep(.el-menu) {
      background-color: transparent;
    }
    :deep(.el-sub-menu) {
      &.is-active {
        .el-sub-menu__title {
          color: var(--el-color-primary);
        }
      }
    }
  }
  .right-menu {
    margin-right: 10px;
    height: 100%;
    display: flex;
    align-items: center;
    &-item {
      margin: 0 10px;
      cursor: pointer;
      &:last-child {
        margin-left: 20px;
      }
    }
    .user {
      display: flex;
      align-items: center;
      .el-avatar {
        margin-right: 10px;
      }
      span {
        font-size: 16px;
      }
    }
  }
}
</style>
