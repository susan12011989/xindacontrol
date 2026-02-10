<script lang="ts" setup>
import type { OssUrlResp } from "@@/apis/global/type"
import type { VxeFormInstance, VxeFormProps, VxeGridInstance, VxeGridProps, VxeModalInstance, VxeModalProps } from "vxe-table"
import { createOssUrl, deleteOssUrl, getOssUrlList, updateOssUrl } from "@@/apis/global"

defineOptions({
  name: "GlobalOssUrl"
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
        field: "url",
        itemRender: {
          name: "$input",
          props: { placeholder: "URL", clearable: true }
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
    { field: "id", title: "ID", width: 80 },
    { field: "url", title: "URL", minWidth: 300, showOverflow: true },
    { field: "updated_at", title: "更新时间", width: 180 },
    {
      title: "操作",
      width: "150px",
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
            url: form.url || "",
            size: page.pageSize,
            page: page.currentPage
          }
          getOssUrlList(params)
            .then((res) => {
              xGridOpt.loading = false
              resolve({
                total: res.data.total,
                result: res.data.list
              })
            })
            .catch(() => {
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
  beforeHideMethod: () => {
    xFormDom.value?.clearValidate()
    return Promise.resolve()
  }
})

const xFormOpt: VxeFormProps = reactive({
  span: 24,
  titleWidth: "100px",
  loading: false,
  titleColon: false,
  data: {
    url: ""
  },
  items: [
    {
      field: "url",
      title: "URL",
      itemRender: {
        name: "$input",
        props: { placeholder: "请输入URL" }
      }
    },
    {
      align: "right",
      itemRender: {
        name: "$buttons",
        children: [
          {
            props: { content: "取消" },
            events: { click: () => xModalDom.value?.close() }
          },
          {
            props: { type: "submit", content: "确定", status: "primary" },
            events: { click: () => crudStore.onSubmitForm() }
          }
        ]
      }
    }
  ],
  rules: {
    url: [{ required: true, message: "请输入URL" }]
  }
})

// ========== CRUD 操作 ==========
const crudStore = reactive({
  isUpdate: false,
  currentId: 0,

  commitQuery: () => xGridDom.value?.commitProxy("query"),

  onShowModal: (row?: OssUrlResp) => {
    if (row) {
      crudStore.isUpdate = true
      crudStore.currentId = row.id
      xModalOpt.title = "编辑 OSS URL"
      xFormOpt.data = {
        url: row.url
      }
    } else {
      crudStore.isUpdate = false
      crudStore.currentId = 0
      xModalOpt.title = "新增 OSS URL"
    }
    xModalDom.value?.open()
    nextTick(() => {
      !crudStore.isUpdate && xFormDom.value?.reset()
      xFormDom.value?.clearValidate()
    })
  },

  onSubmitForm: () => {
    if (xFormOpt.loading) return
    xFormDom.value?.validate((errMap) => {
      if (errMap) return
      xFormOpt.loading = true

      const apiCall = crudStore.isUpdate
        ? updateOssUrl(crudStore.currentId, xFormOpt.data)
        : createOssUrl(xFormOpt.data)

      apiCall
        .then(() => {
          xFormOpt.loading = false
          xModalDom.value?.close()
          ElMessage.success("操作成功")
          crudStore.commitQuery()
        })
        .catch(() => {
          xFormOpt.loading = false
        })
    })
  },

  onDelete: (row: OssUrlResp) => {
    ElMessageBox.confirm(`确定删除该 URL 吗？`, "提示", { type: "warning" }).then(() => {
      deleteOssUrl(row.id).then(() => {
        ElMessage.success("删除成功")
        crudStore.commitQuery()
      })
    })
  }
})
</script>

<template>
  <div class="app-container">
    <!-- 表格 -->
    <vxe-grid ref="xGridDom" v-bind="xGridOpt">
      <!-- 工具栏按钮 -->
      <template #toolbar-btns>
        <vxe-button status="primary" icon="vxe-icon-add" @click="crudStore.onShowModal()"> 新增 </vxe-button>
      </template>

      <!-- 操作列 -->
      <template #row-operate="{ row }">
        <el-button link type="primary" @click="crudStore.onShowModal(row)"> 编辑 </el-button>
        <el-button link type="danger" @click="crudStore.onDelete(row)"> 删除 </el-button>
      </template>
    </vxe-grid>

    <!-- 弹窗 -->
    <vxe-modal ref="xModalDom" v-bind="xModalOpt">
      <vxe-form ref="xFormDom" v-bind="xFormOpt" />
    </vxe-modal>
  </div>
</template>

<style lang="scss" scoped>
.app-container {
  padding: 20px;
}
</style>
