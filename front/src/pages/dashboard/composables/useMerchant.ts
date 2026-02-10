import type { CreateOrEditMerchantRequestData, Merchant, MerchantQueryRequestData } from "../apis/type"
import { useFullscreenLoading } from "@@/composables/useFullscreenLoading"
import { cloneDeep } from "lodash-es"
import { createMerchantApi, deleteMerchantApi, merchantQueryApi, updateMerchantApi } from "../apis"

export const DEFAULT_FORM_DATA: CreateOrEditMerchantRequestData = {
  name: "",
  no: "",
  server_ip: "",
  status: 1,
  expired_at: "",
  package_configuration: {
    dau_limit: 0,
    register_limit: 0,
    group_member_limit: 0
  }
}

export function useMerchant() {
  const loading = ref<boolean>(false)
  const tableData = ref<Merchant[]>([])
  const dialogVisible = ref<boolean>(false)
  const detailDialogVisible = ref<boolean>(false)
  const formData = ref<CreateOrEditMerchantRequestData>(cloneDeep(DEFAULT_FORM_DATA))
  const editingMerchantId = ref<number | null>(null)
  const currentMerchant = ref<Merchant | null>(null)
  // 保存最后一次查询参数
  const lastQueryParams = ref<MerchantQueryRequestData>({ page: 1, size: 10 })
  // 新增总数变量
  const total = ref<number>(0)

  // 查询商户列表
  function getMerchantList(params: MerchantQueryRequestData) {
    loading.value = true
    // 保存最后一次查询参数
    lastQueryParams.value = { ...params }
    console.log("查询参数:", params)
    return merchantQueryApi(params)
      .then(({ data }) => {
        tableData.value = data.list
        // 更新总数
        total.value = data.total
        return data
      })
      .catch(() => {
        tableData.value = []
        total.value = 0
        return { total: 0, list: [] }
      })
      .finally(() => {
        loading.value = false
      })
  }

  // 显示新增商户对话框
  function showAddDialog() {
    dialogVisible.value = true
    editingMerchantId.value = null
    formData.value = cloneDeep(DEFAULT_FORM_DATA)
  }

  // 显示编辑商户对话框
  function showEditDialog(row: Merchant) {
    dialogVisible.value = true
    editingMerchantId.value = row.id

    formData.value = {
      id: row.id,
      name: row.name,
      no: row.no,
      port: row.port,
      server_ip: row.server_ip,
      status: row.status,
      expired_at: row.expired_at,
      package_configuration: row.package_configuration ? cloneDeep(row.package_configuration) : undefined
    }
  }

  // 显示详情对话框
  function showDetailDialog(row: Merchant) {
    currentMerchant.value = row
    detailDialogVisible.value = true
  }

  // 提交表单
  function normalizePayload(data: CreateOrEditMerchantRequestData): CreateOrEditMerchantRequestData {
    const payload = cloneDeep(data)
    return payload
  }

  function submitForm() {
    loading.value = true
    const apiData = normalizePayload(formData.value)

    const runWithLoading = useFullscreenLoading(async () => {
      if (editingMerchantId.value) {
        // 编辑模式，确保ID存在
        apiData.id = editingMerchantId.value
        // 编辑模式不允许修改端口：移除 port 字段
        if ("port" in apiData) {
          delete (apiData as any).port
        }

        return updateMerchantApi(apiData)
          .then(() => {
            ElMessage.success("修改成功")
            dialogVisible.value = false
            // 刷新商户列表
            return refreshList().then(() => true)
          })
          .catch(() => {
            ElMessage.error("修改失败")
            return false
          })
          .finally(() => {
            loading.value = false
          })
      } else {
        // 新增模式，确保无ID
        delete apiData.id

        return createMerchantApi(apiData)
          .then(() => {
            ElMessage.success("添加成功")
            dialogVisible.value = false
            // 刷新商户列表
            return refreshList().then(() => true)
          })
          .catch(() => {
            ElMessage.error("添加失败")
            return false
          })
          .finally(() => {
            loading.value = false
          })
      }
    }, { text: "创建/保存中，请稍候..." })
    return runWithLoading()
  }

  // 删除商户
  function handleDelete(row: Merchant) {
    return ElMessageBox.confirm(`确定要删除商户 "${row.name}" 吗？`, "提示", {
      confirmButtonText: "确定",
      cancelButtonText: "取消",
      type: "warning"
    }).then(() => {
      loading.value = true
      return deleteMerchantApi(row.id)
        .then(() => {
          ElMessage.success("删除成功")
          // 刷新商户列表
          refreshList()
          return true
        })
        .catch(() => {
          ElMessage.error("删除失败")
          return false
        })
        .finally(() => {
          loading.value = false
        })
    }).catch(() => false)
  }

  // 使用上次的参数刷新列表
  function refreshList() {
    return getMerchantList(lastQueryParams.value)
  }

  return {
    loading,
    tableData,
    dialogVisible,
    detailDialogVisible,
    formData,
    editingMerchantId,
    currentMerchant,
    total, // 导出总数
    getMerchantList,
    refreshList,
    showAddDialog,
    showEditDialog,
    showDetailDialog,
    submitForm,
    handleDelete
  }
}
