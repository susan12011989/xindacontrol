<script lang="ts" setup>
import type { AdminmUserListItem } from "@@/apis/adminm_users/type"
import type { FormInstance, FormRules } from "element-plus"
import { createAdminmUser, deleteAdminmUser, queryAdminmUsers, updateAdminmUser } from "@@/apis/adminm_users"
import { usePagination } from "@@/composables/usePagination"

const props = defineProps<{
  visible: boolean
  merchantNo: string
}>()

const emit = defineEmits<{ (e: "update:visible", v: boolean): void }>()

const visible = computed({
  get: () => props.visible,
  set: v => emit("update:visible", v)
})

const { paginationData, handleCurrentChange, handleSizeChange } = usePagination()
const loading = ref(false)
const usernameKeyword = ref("")
const list = ref<AdminmUserListItem[]>([])
const total = ref(0)

function fetchList() {
  if (!props.merchantNo) return
  loading.value = true
  return queryAdminmUsers({
    merchant_no: props.merchantNo,
    page: paginationData.currentPage,
    size: paginationData.pageSize,
    username: usernameKeyword.value || undefined
  })
    .then(({ data }) => {
      list.value = data.list
      total.value = data.total
      paginationData.total = data.total
    })
    .catch(() => {
      // 已由 axios 拦截器统一弹出错误提示
    })
    .finally(() => (loading.value = false))
}

watch([() => paginationData.currentPage, () => paginationData.pageSize], () => {
  if (props.visible) fetchList()
})

watch(() => props.visible, (v) => {
  if (v) {
    paginationData.currentPage = 1
    fetchList()
  }
})

function onSearch() {
  paginationData.currentPage = 1
  fetchList()
}

// ====== 创建/编辑 ======
const editDialogVisible = ref(false)
const isEdit = ref(false)
const formRef = ref<FormInstance | null>(null)
const form = ref<{ username: string, phone: string, password: string, target_username?: string, allowed_ips?: string }>({
  username: "",
  phone: "",
  password: "",
  allowed_ips: ""
})

const rules = computed<FormRules>(() => ({
  username: [{ required: !isEdit.value, message: "请输入用户名", trigger: "blur" }],
  phone: [{ required: !isEdit.value, message: "请输入手机号", trigger: "blur" }],
  password: [{ required: !isEdit.value, message: "请输入密码", trigger: "blur" }]
}))

function openCreate() {
  isEdit.value = false
  form.value = { username: "", phone: "", password: "", allowed_ips: "" }
  editDialogVisible.value = true
}

const originalAllowedIPs = ref("")

function openEdit(row: AdminmUserListItem) {
  isEdit.value = true
  originalAllowedIPs.value = row.allowed_ips || ""
  form.value = { username: row.username, phone: "", password: "", target_username: row.username, allowed_ips: row.allowed_ips || "" }
  editDialogVisible.value = true
}

function submitForm() {
  formRef.value?.validate(async (valid) => {
    if (!valid) return
    const loadingMsg = ElMessage({ type: "info", message: isEdit.value ? "更新中..." : "创建中...", duration: 0 })
    try {
      if (isEdit.value) {
        // 只有当IP白名单变化时才传递，避免不必要的踢出登录
        const ipChanged = form.value.allowed_ips !== originalAllowedIPs.value
        await updateAdminmUser({
          merchant_no: props.merchantNo,
          target_username: form.value.target_username || "",
          password: form.value.password || undefined,
          allowed_ips: ipChanged ? form.value.allowed_ips : undefined
        })
        ElMessage.success("更新成功")
      } else {
        await createAdminmUser({
          merchant_no: props.merchantNo,
          username: form.value.username,
          phone: form.value.phone,
          password: form.value.password
        })
        ElMessage.success("创建成功")
      }
      editDialogVisible.value = false
      fetchList()
    } catch {
      // 已统一弹出
    } finally {
      loadingMsg.close()
    }
  })
}

function handleDelete(row: AdminmUserListItem) {
  ElMessageBox.confirm(`确认删除用户 ${row.username} ?`, "提示", { type: "warning" })
    .then(async () => {
      await deleteAdminmUser({ merchant_no: props.merchantNo, username: row.username })
      ElMessage.success("删除成功")
      fetchList()
    })
    .catch(() => {})
}
</script>

<template>
  <el-dialog v-model="visible" title="后台账号管理" width="720px" destroy-on-close>
    <div class="toolbar">
      <el-input v-model="usernameKeyword" placeholder="按用户名搜索" clearable style="width: 220px;" @keyup.enter="onSearch" />
      <el-button type="primary" @click="onSearch">查询</el-button>
      <el-button type="primary" @click="openCreate">新增账号</el-button>
    </div>

    <el-table :data="list" v-loading="loading" border>
      <el-table-column prop="username" label="用户名" min-width="120" align="center" />
      <el-table-column prop="name" label="显示名称" min-width="120" align="center" />
      <el-table-column prop="allowed_ips" label="IP白名单" min-width="160" align="center">
        <template #default="{ row }">
          <span v-if="row.allowed_ips" class="ip-list">{{ row.allowed_ips }}</span>
          <span v-else class="no-limit">不限制</span>
        </template>
      </el-table-column>
      <el-table-column prop="register_time" label="创建时间" width="160" align="center" />
      <el-table-column label="操作" width="140" align="center" fixed="right">
        <template #default="{ row }">
          <el-button link type="primary" size="small" @click="openEdit(row)">编辑</el-button>
          <el-button link type="danger" size="small" @click="handleDelete(row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>

    <div class="pager">
      <el-pagination
        background
        :layout="paginationData.layout"
        :page-sizes="paginationData.pageSizes"
        :total="paginationData.total"
        :page-size="paginationData.pageSize"
        :current-page="paginationData.currentPage"
        @size-change="handleSizeChange"
        @current-change="handleCurrentChange"
      />
    </div>

    <template #footer>
      <el-button @click="visible = false">关闭</el-button>
    </template>
  </el-dialog>

  <!-- 创建/编辑 -->
  <el-dialog v-model="editDialogVisible" :title="isEdit ? '编辑账号' : '新增账号'" width="520px" destroy-on-close>
    <el-form ref="formRef" :model="form" :rules="rules" label-width="120px">
      <el-form-item v-if="!isEdit" label="用户名" prop="username">
        <el-input v-model="form.username" placeholder="请输入用户名" />
      </el-form-item>
      <el-form-item v-if="!isEdit" label="手机号" prop="phone">
        <el-input v-model="form.phone" placeholder="请输入手机号" />
      </el-form-item>
      <el-form-item v-if="isEdit" label="账号">
        <span>{{ form.target_username }}</span>
      </el-form-item>
      <el-form-item label="密码" prop="password">
        <el-input v-model="form.password" type="password" show-password :placeholder="isEdit ? '留空不修改密码' : '请输入密码'" />
      </el-form-item>
      <el-form-item v-if="isEdit" label="IP白名单">
        <el-input v-model="form.allowed_ips" type="textarea" :rows="2" placeholder="多个IP用逗号分隔，为空表示不限制" />
        <div class="form-tip">设置后，该账号只能从指定IP登录后台</div>
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="editDialogVisible = false">取消</el-button>
      <el-button type="primary" @click="submitForm">确定</el-button>
    </template>
  </el-dialog>
</template>

<style scoped>
.toolbar {
  display: flex;
  gap: 10px;
  margin-bottom: 12px;
}
.pager {
  margin-top: 12px;
  display: flex;
  justify-content: flex-end;
}
.form-tip {
  color: #909399;
  font-size: 12px;
  line-height: 1.5;
  margin-top: 4px;
}
.ip-list {
  color: #409eff;
  font-size: 12px;
  word-break: break-all;
}
.no-limit {
  color: #909399;
  font-size: 12px;
}
</style>
