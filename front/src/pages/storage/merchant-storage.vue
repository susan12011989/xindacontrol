<script lang="ts" setup>
import type { MerchantStorageResp } from "@@/apis/merchant_storage/type"
import type { VxeFormInstance, VxeFormProps, VxeGridInstance, VxeGridProps, VxeModalInstance, VxeModalProps } from "vxe-table"
import { createMerchantStorage, deleteMerchantStorage, getMerchantStorageList, pushMerchantStorage, updateMerchantStorage } from "@@/apis/merchant_storage"
import { getMerchantList } from "@@/apis/merchant"
import { ElMessage, ElMessageBox } from "element-plus"

defineOptions({
  name: "MerchantStorageManagement"
})

// 存储类型选项
const storageTypeOptions = [
  { label: "MinIO", value: "minio" },
  { label: "阿里云 OSS", value: "aliyunOSS" },
  { label: "AWS S3", value: "aws_s3" },
  { label: "腾讯云 COS", value: "tencent_cos" }
]

// 状态选项
const statusOptions = [
  { label: "启用", value: 1 },
  { label: "禁用", value: 0 }
]

// 商户选项 (使用 reactive 数组以便在 VXE Grid options 中使用)
const merchantOptions: { label: string; value: number }[] = reactive([])

// 加载商户列表
async function loadMerchants() {
  try {
    const res = await getMerchantList({ page: 1, size: 1000 })
    const list = res.data.list?.map((m: any) => ({
      label: `${m.name} (${m.no})`,
      value: m.id
    })) || []
    merchantOptions.length = 0
    merchantOptions.push(...list)
  } catch (e) {
    console.error("加载商户列表失败", e)
  }
}

onMounted(() => {
  loadMerchants()
})

// ========== VXE Grid 配置 ==========
const xGridDom = ref<VxeGridInstance>()
const xGridOpt: VxeGridProps = reactive({
  loading: true,
  autoResize: true,
  pagerConfig: {
    align: "right"
  },
  formConfig: {
    items: [
      {
        field: "merchant_id",
        itemRender: {
          name: "$select",
          options: merchantOptions,
          props: { placeholder: "选择商户", clearable: true, filterable: true }
        }
      },
      {
        field: "storage_type",
        itemRender: {
          name: "$select",
          options: storageTypeOptions,
          props: { placeholder: "存储类型", clearable: true }
        }
      },
      {
        field: "status",
        itemRender: {
          name: "$select",
          options: statusOptions,
          props: { placeholder: "状态", clearable: true }
        }
      },
      {
        itemRender: {
          name: "$buttons",
          children: [
            { props: { type: "submit", content: "查询", status: "primary" } },
            { props: { type: "reset", content: "重置" } }
          ]
        }
      }
    ]
  },
  toolbarConfig: {
    refresh: true,
    custom: true,
    slots: { buttons: "toolbar-btns" }
  },
  columns: [
    { type: "seq", width: "60px", title: "序号" },
    { field: "merchant_name", title: "商户", width: 150 },
    { field: "name", title: "配置名称", width: 120 },
    {
      field: "storage_type",
      title: "存储类型",
      width: 120,
      slots: { default: "storage-type-slot" }
    },
    { field: "bucket", title: "Bucket", width: 150, showOverflow: true },
    { field: "endpoint", title: "Endpoint", showOverflow: true },
    {
      field: "is_default",
      title: "默认",
      width: 80,
      slots: { default: "default-slot" }
    },
    {
      field: "status",
      title: "状态",
      width: 80,
      slots: { default: "status-slot" }
    },
    { field: "last_push_at", title: "最后推送", width: 160 },
    {
      field: "last_push_result",
      title: "推送结果",
      width: 100,
      slots: { default: "push-result-slot" }
    },
    {
      title: "操作",
      width: "200px",
      fixed: "right",
      slots: { default: "row-operate" }
    }
  ],
  proxyConfig: {
    seq: true,
    form: true,
    autoLoad: true,
    props: { total: "total" },
    ajax: {
      query: ({ page, form }) => {
        xGridOpt.loading = true
        return new Promise((resolve) => {
          const params = {
            merchant_id: form.merchant_id || undefined,
            storage_type: form.storage_type || undefined,
            status: form.status,
            size: page.pageSize,
            page: page.currentPage
          }
          getMerchantStorageList(params).then((res) => {
            xGridOpt.loading = false
            resolve({
              total: res.data.total,
              result: res.data.list || []
            })
          }).catch(() => {
            xGridOpt.loading = false
          })
        })
      }
    }
  }
})

// ========== Modal & Form 配置 ==========
const xModalDom = ref<VxeModalInstance>()
const xFormDom = ref<VxeFormInstance>()

const xModalOpt: VxeModalProps = reactive({
  title: "",
  showClose: true,
  escClosable: true,
  maskClosable: true,
  width: 700,
  beforeHideMethod: () => {
    xFormDom.value?.clearValidate()
    return Promise.resolve()
  }
})

const xFormOpt: VxeFormProps = reactive({
  span: 24,
  titleWidth: "120px",
  loading: false,
  titleColon: false,
  data: {
    merchant_id: undefined as number | undefined,
    storage_type: "minio",
    name: "",
    endpoint: "",
    bucket: "",
    region: "",
    access_key_id: "",
    access_key_secret: "",
    upload_url: "",
    download_url: "",
    file_base_url: "",
    bucket_url: "",
    custom_domain: "",
    is_default: 0,
    status: 1
  },
  items: [
    {
      field: "merchant_id",
      title: "商户",
      itemRender: {
        name: "$select",
        options: merchantOptions,
        props: { placeholder: "请选择商户", filterable: true }
      }
    },
    {
      field: "storage_type",
      title: "存储类型",
      itemRender: {
        name: "$select",
        options: storageTypeOptions,
        props: { placeholder: "请选择存储类型" }
      }
    },
    {
      field: "name",
      title: "配置名称",
      itemRender: {
        name: "$input",
        props: { placeholder: "请输入配置名称" }
      }
    },
    {
      field: "endpoint",
      title: "服务端点",
      itemRender: {
        name: "$input",
        props: { placeholder: "如: http://minio:9000 或 oss-cn-hangzhou.aliyuncs.com" }
      }
    },
    {
      field: "bucket",
      title: "Bucket",
      itemRender: {
        name: "$input",
        props: { placeholder: "请输入 Bucket 名称" }
      }
    },
    {
      field: "region",
      title: "区域",
      visibleMethod: () => ["aws_s3", "tencent_cos"].includes(xFormOpt.data.storage_type),
      itemRender: {
        name: "$input",
        props: { placeholder: "如: us-east-1, ap-guangzhou" }
      }
    },
    {
      field: "access_key_id",
      title: "AccessKeyId",
      itemRender: {
        name: "$input",
        props: { placeholder: "请输入 AccessKeyId" }
      }
    },
    {
      field: "access_key_secret",
      title: "AccessKeySecret",
      itemRender: {
        name: "$input",
        props: { placeholder: "编辑时留空表示不修改", type: "password", showPassword: true }
      }
    },
    {
      field: "upload_url",
      title: "上传URL",
      visibleMethod: () => xFormOpt.data.storage_type === "minio",
      itemRender: {
        name: "$input",
        props: { placeholder: "可选，留空使用 Endpoint" }
      }
    },
    {
      field: "download_url",
      title: "下载URL",
      visibleMethod: () => xFormOpt.data.storage_type === "minio",
      itemRender: {
        name: "$input",
        props: { placeholder: "可选，留空使用 Endpoint" }
      }
    },
    {
      field: "file_base_url",
      title: "文件基础URL",
      visibleMethod: () => ["minio", "aws_s3", "tencent_cos"].includes(xFormOpt.data.storage_type),
      itemRender: {
        name: "$input",
        props: { placeholder: "文件访问的基础URL" }
      }
    },
    {
      field: "bucket_url",
      title: "Bucket URL",
      visibleMethod: () => xFormOpt.data.storage_type === "aliyunOSS",
      itemRender: {
        name: "$input",
        props: { placeholder: "如: https://bucket.oss-cn-hangzhou.aliyuncs.com" }
      }
    },
    {
      field: "custom_domain",
      title: "自定义域名",
      itemRender: {
        name: "$input",
        props: { placeholder: "CDN 自定义域名（可选）" }
      }
    },
    {
      field: "is_default",
      title: "设为默认",
      itemRender: {
        name: "$radio",
        options: [
          { label: "否", value: 0 },
          { label: "是", value: 1 }
        ]
      }
    },
    {
      field: "status",
      title: "状态",
      itemRender: {
        name: "$radio",
        options: statusOptions
      }
    }
  ],
  rules: {
    merchant_id: [{ required: true, message: "请选择商户" }],
    storage_type: [{ required: true, message: "请选择存储类型" }],
    name: [{ required: true, message: "请输入配置名称" }],
    bucket: [{ required: true, message: "请输入 Bucket" }],
    access_key_id: [{ required: true, message: "请输入 AccessKeyId" }]
  }
})

// ========== CRUD 操作 ==========
const crudStore = reactive({
  isUpdate: false,
  currentId: 0,
  commitQuery: () => xGridDom.value?.commitProxy("query"),
  onShowModal: (row?: MerchantStorageResp) => {
    if (row) {
      crudStore.isUpdate = true
      crudStore.currentId = row.id
      xModalOpt.title = "编辑存储配置"
      Object.assign(xFormOpt.data, {
        merchant_id: row.merchant_id,
        storage_type: row.storage_type,
        name: row.name,
        endpoint: row.endpoint,
        bucket: row.bucket,
        region: row.region,
        access_key_id: row.access_key_id,
        access_key_secret: "", // 编辑时不显示密钥
        upload_url: row.upload_url,
        download_url: row.download_url,
        file_base_url: row.file_base_url,
        bucket_url: row.bucket_url,
        custom_domain: row.custom_domain,
        is_default: row.is_default,
        status: row.status
      })
    } else {
      crudStore.isUpdate = false
      crudStore.currentId = 0
      xModalOpt.title = "新增存储配置"
      Object.assign(xFormOpt.data, {
        merchant_id: undefined,
        storage_type: "minio",
        name: "",
        endpoint: "",
        bucket: "",
        region: "",
        access_key_id: "",
        access_key_secret: "",
        upload_url: "",
        download_url: "",
        file_base_url: "",
        bucket_url: "",
        custom_domain: "",
        is_default: 0,
        status: 1
      })
    }
    xModalDom.value?.open()
  },
  onSubmitForm: () => {
    xFormDom.value?.validate((errMap) => {
      if (errMap) return
      xFormOpt.loading = true
      const apiCall = crudStore.isUpdate
        ? updateMerchantStorage(crudStore.currentId, xFormOpt.data as any)
        : createMerchantStorage(xFormOpt.data as any)
      apiCall.then(() => {
        ElMessage.success(crudStore.isUpdate ? "更新成功" : "创建成功")
        xModalDom.value?.close()
        crudStore.commitQuery()
      }).catch((err) => {
        ElMessage.error(err?.message || "操作失败")
      }).finally(() => {
        xFormOpt.loading = false
      })
    })
  },
  onDelete: (row: MerchantStorageResp) => {
    ElMessageBox.confirm(`确定删除配置「${row.name}」吗？`, "删除确认", {
      type: "warning"
    }).then(() => {
      deleteMerchantStorage(row.id).then(() => {
        ElMessage.success("删除成功")
        crudStore.commitQuery()
      }).catch((err) => {
        ElMessage.error(err?.message || "删除失败")
      })
    }).catch(() => {})
  }
})

// ========== 推送配置 ==========
const pushModalVisible = ref(false)
const pushForm = reactive({
  merchant_id: 0,
  config_id: 0,
  config_name: "",
  twofa_code: ""
})
const pushLoading = ref(false)

function onShowPushModal(row: MerchantStorageResp) {
  pushForm.merchant_id = row.merchant_id
  pushForm.config_id = row.id
  pushForm.config_name = row.name
  pushForm.twofa_code = ""
  pushModalVisible.value = true
}

function onPushSubmit() {
  if (!pushForm.twofa_code || pushForm.twofa_code.length !== 6) {
    ElMessage.warning("请输入6位2FA验证码")
    return
  }
  pushLoading.value = true
  pushMerchantStorage({
    merchant_id: pushForm.merchant_id,
    config_id: pushForm.config_id,
    twofa_code: pushForm.twofa_code
  }).then((res) => {
    if (res.data.success) {
      ElMessage.success("推送成功")
      pushModalVisible.value = false
      crudStore.commitQuery()
    } else {
      ElMessage.error(res.data.message || "推送失败")
    }
  }).catch((err) => {
    ElMessage.error(err?.message || "推送失败")
  }).finally(() => {
    pushLoading.value = false
  })
}

// 获取存储类型标签
function getStorageTypeLabel(type: string) {
  const option = storageTypeOptions.find((o) => o.value === type)
  return option?.label || type
}

// 获取存储类型标签颜色
function getStorageTypeTagType(type: string): "success" | "warning" | "info" | "primary" | "danger" {
  const typeMap: Record<string, "success" | "warning" | "info" | "primary" | "danger"> = {
    minio: "primary",
    aliyunOSS: "warning",
    aws_s3: "success",
    tencent_cos: "info"
  }
  return typeMap[type] || "info"
}
</script>

<template>
  <div class="curd-container">
    <!-- VXE Table -->
    <vxe-grid ref="xGridDom" v-bind="xGridOpt">
      <!-- 工具栏按钮 -->
      <template #toolbar-btns>
        <vxe-button status="primary" icon="vxe-icon-add" @click="crudStore.onShowModal()">
          新增配置
        </vxe-button>
      </template>

      <!-- 存储类型 -->
      <template #storage-type-slot="{ row }">
        <el-tag :type="getStorageTypeTagType(row.storage_type)" size="small">
          {{ getStorageTypeLabel(row.storage_type) }}
        </el-tag>
      </template>

      <!-- 默认标记 -->
      <template #default-slot="{ row }">
        <el-tag v-if="row.is_default === 1" type="success" size="small">
          默认
        </el-tag>
        <span v-else>-</span>
      </template>

      <!-- 状态 -->
      <template #status-slot="{ row }">
        <el-tag :type="row.status === 1 ? 'success' : 'danger'" size="small">
          {{ row.status === 1 ? "启用" : "禁用" }}
        </el-tag>
      </template>

      <!-- 推送结果 -->
      <template #push-result-slot="{ row }">
        <el-tag v-if="row.last_push_result === '成功'" type="success" size="small">
          成功
        </el-tag>
        <el-tooltip v-else-if="row.last_push_result" :content="row.last_push_result" placement="top">
          <el-tag type="danger" size="small">
            失败
          </el-tag>
        </el-tooltip>
        <span v-else>-</span>
      </template>

      <!-- 操作按钮 -->
      <template #row-operate="{ row }">
        <el-button type="primary" link size="small" @click="crudStore.onShowModal(row)">
          编辑
        </el-button>
        <el-button type="success" link size="small" @click="onShowPushModal(row)">
          推送
        </el-button>
        <el-button type="danger" link size="small" @click="crudStore.onDelete(row)">
          删除
        </el-button>
      </template>
    </vxe-grid>

    <!-- 编辑弹窗 -->
    <vxe-modal ref="xModalDom" v-bind="xModalOpt">
      <template #default>
        <vxe-form ref="xFormDom" v-bind="xFormOpt" />
      </template>
      <template #footer>
        <vxe-button @click="xModalDom?.close()">
          取消
        </vxe-button>
        <vxe-button status="primary" :loading="xFormOpt.loading" @click="crudStore.onSubmitForm">
          确定
        </vxe-button>
      </template>
    </vxe-modal>

    <!-- 推送确认弹窗 -->
    <el-dialog v-model="pushModalVisible" title="推送存储配置" width="400px">
      <div class="push-form">
        <p>即将推送配置「{{ pushForm.config_name }}」到商户服务器</p>
        <p style="color: #909399; font-size: 13px; margin: 10px 0;">
          推送后将立即生效，请确保配置正确
        </p>
        <el-input
          v-model="pushForm.twofa_code"
          placeholder="请输入6位2FA验证码"
          maxlength="6"
          style="margin-top: 15px;"
        />
      </div>
      <template #footer>
        <el-button @click="pushModalVisible = false">
          取消
        </el-button>
        <el-button type="primary" :loading="pushLoading" @click="onPushSubmit">
          确认推送
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.curd-container {
  padding: 15px;
  background-color: var(--el-bg-color);
}

.push-form {
  padding: 10px 0;
}
</style>
