<script lang="ts" setup>
import type { ClientResp } from "@@/apis/clients/type"
import type { VxeFormInstance, VxeFormProps, VxeGridInstance, VxeGridProps, VxeModalInstance, VxeModalProps } from "vxe-table"
import { createClient, deleteClient, getClientList, updateClient } from "@@/apis/clients"

defineOptions({
  name: "ClientsList"
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
        field: "app_package_name",
        itemRender: {
          name: "$input",
          props: { placeholder: "安卓包名", clearable: true }
        }
      },
      {
        field: "app_name",
        itemRender: {
          name: "$input",
          props: { placeholder: "APP名称", clearable: true }
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
    { field: "app_package_name", title: "安卓包名", minWidth: 200, showOverflow: true },
    { field: "app_name", title: "APP名称", minWidth: 150, showOverflow: true },
    {
      title: "配置状态",
      width: "120px",
      slots: { default: "config-status" }
    },
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
            app_package_name: form.app_package_name || "",
            app_name: form.app_name || "",
            size: page.pageSize,
            page: page.currentPage
          }
          getClientList(params)
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
    app_package_name: "",
    app_name: "",
    sms_config: {
      region_id: "",
      access_key: "",
      secret_key: "",
      sign_name: "",
      template_code: ""
    },
    push_config: {
      push_hms: { app_id: "", app_secret: "" },
      push_xiaomi: { app_secret: "", package: "", channel_id: "", time_to_live: 3600 },
      push_oppo: { app_key: "", master_secret: "", channel_id: "", time_to_live: 3600 },
      push_vivo: { app_id: 0, app_key: "", app_secret: "", time_to_live: 3600 },
      push_honor: { app_id: "", client_id: "", client_secret: "", time_to_live: 3600 }
    },
    trtc_config: { app_id: 0, app_key: "" }
  },
  items: [
    {
      field: "app_package_name",
      title: "安卓包名",
      itemRender: {
        name: "$input",
        props: { placeholder: "请输入安卓包名，如：com.example.app" }
      }
    },
    {
      field: "app_name",
      title: "APP名称",
      itemRender: {
        name: "$input",
        props: { placeholder: "请输入APP名称" }
      }
    },
    {
      title: "配置信息",
      span: 24,
      slots: { default: "form-config" }
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
    app_package_name: [
      {
        required: true,
        validator: ({ itemValue }) => {
          if (!itemValue) return new Error("请输入安卓包名")
          if (!itemValue.trim()) return new Error("包名不能为空格")
          // 验证包名格式（简单验证）
          const packageNamePattern = /^[a-z]\w*(\.[a-z]\w*)+$/i
          if (!packageNamePattern.test(itemValue)) {
            return new Error("包名格式不正确，如：com.example.app")
          }
        }
      }
    ],
    app_name: [
      {
        required: true,
        validator: ({ itemValue }) => {
          if (!itemValue) return new Error("请输入APP名称")
          if (!itemValue.trim()) return new Error("APP名称不能为空格")
        }
      }
    ]
  }
})

// ========== CRUD 操作 ==========
const crudStore = reactive({
  isUpdate: false,
  currentId: 0,

  commitQuery: () => xGridDom.value?.commitProxy("query"),

  onShowModal: (row?: ClientResp) => {
    if (row) {
      crudStore.isUpdate = true
      crudStore.currentId = row.id
      xModalOpt.title = "编辑客户端"
      xFormOpt.data = {
        app_package_name: row.app_package_name,
        app_name: row.app_name,
        sms_config: row.sms_config || {
          region_id: "",
          access_key: "",
          secret_key: "",
          sign_name: "",
          template_code: ""
        },
        push_config: row.push_config
          ? {
              push_hms: row.push_config.push_hms || { app_id: "", app_secret: "" },
              push_xiaomi: { ...(row.push_config.push_xiaomi || { app_secret: "", package: "", channel_id: "" }), time_to_live: 3600 },
              push_oppo: { ...(row.push_config.push_oppo || { app_key: "", master_secret: "", channel_id: "" }), time_to_live: 3600 },
              push_vivo: { ...(row.push_config.push_vivo || { app_id: 0, app_key: "", app_secret: "" }), time_to_live: 3600 },
              push_honor: { ...(row.push_config.push_honor || { app_id: "", client_id: "", client_secret: "" }), time_to_live: 3600 }
            }
          : {
              push_hms: { app_id: "", app_secret: "" },
              push_xiaomi: { app_secret: "", package: "", channel_id: "", time_to_live: 3600 },
              push_oppo: { app_key: "", master_secret: "", channel_id: "", time_to_live: 3600 },
              push_vivo: { app_id: 0, app_key: "", app_secret: "", time_to_live: 3600 },
              push_honor: { app_id: "", client_id: "", client_secret: "", time_to_live: 3600 }
            },
        trtc_config: row.trtc_config || { app_id: 0, app_key: "" }
      }
    } else {
      crudStore.isUpdate = false
      crudStore.currentId = 0
      xModalOpt.title = "新增客户端"
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
        ? updateClient(crudStore.currentId, xFormOpt.data)
        : createClient(xFormOpt.data)

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

  onDelete: (row: ClientResp) => {
    ElMessageBox.confirm(`确定删除客户端 "${row.app_name}" (${row.app_package_name}) 吗？`, "提示", {
      type: "warning"
    })
      .then(() => {
        deleteClient(row.id).then(() => {
          ElMessage.success("删除成功")
          crudStore.commitQuery()
        })
      })
      .catch(() => {
        // 用户取消删除
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
        <vxe-button status="primary" icon="vxe-icon-add" @click="crudStore.onShowModal()"> 新增客户端 </vxe-button>
      </template>

      <!-- 配置状态列 -->
      <template #config-status="{ row }">
        <div style="display: flex; flex-direction: column; gap: 4px;">
          <el-tag v-if="row.sms_config" type="success" size="small"> SMS已配 </el-tag>
          <el-tag v-else type="info" size="small"> SMS未配 </el-tag>
          <el-tag v-if="row.push_config" type="success" size="small"> 推送已配 </el-tag>
          <el-tag v-else type="info" size="small"> 推送未配 </el-tag>
          <el-tag v-if="row.trtc_config" type="success" size="small"> TRTC已配 </el-tag>
          <el-tag v-else type="info" size="small"> TRTC未配 </el-tag>
        </div>
      </template>

      <!-- 操作列 -->
      <template #row-operate="{ row }">
        <el-button link type="primary" @click="crudStore.onShowModal(row)"> 编辑 </el-button>
        <el-button link type="danger" @click="crudStore.onDelete(row)"> 删除 </el-button>
      </template>
    </vxe-grid>

    <!-- 弹窗 -->
    <vxe-modal ref="xModalDom" v-bind="xModalOpt" width="800px">
      <vxe-form ref="xFormDom" v-bind="xFormOpt">
        <!-- 配置信息插槽 -->
        <template #form-config>
          <el-collapse accordion style="margin-top: 10px;">
            <!-- SMS配置 -->
            <el-collapse-item title="短信配置 (SMS Config)" name="sms">
              <el-form label-width="120px" size="default">
                <el-form-item label="区域ID">
                  <el-input v-model="xFormOpt.data.sms_config.region_id" placeholder="如：cn-hangzhou" clearable />
                </el-form-item>
                <el-form-item label="AccessKey">
                  <el-input v-model="xFormOpt.data.sms_config.access_key" placeholder="请输入AccessKey" clearable />
                </el-form-item>
                <el-form-item label="SecretKey">
                  <el-input v-model="xFormOpt.data.sms_config.secret_key" type="password" show-password placeholder="请输入SecretKey" clearable />
                </el-form-item>
                <el-form-item label="签名">
                  <el-input v-model="xFormOpt.data.sms_config.sign_name" placeholder="短信签名" clearable />
                </el-form-item>
                <el-form-item label="模板代码">
                  <el-input v-model="xFormOpt.data.sms_config.template_code" placeholder="短信模板代码" clearable />
                </el-form-item>
              </el-form>
            </el-collapse-item>

            <!-- TRTC 配置 -->
            <el-collapse-item title="TRTC 配置" name="trtc">
              <el-form label-width="120px" size="default">
                <el-form-item label="App ID">
                  <el-input-number v-model="xFormOpt.data.trtc_config.app_id" :min="0" placeholder="TRTC App ID" style="width: 100%;" />
                </el-form-item>
                <el-form-item label="App Key">
                  <el-input v-model="xFormOpt.data.trtc_config.app_key" type="password" show-password placeholder="TRTC App Key" clearable />
                </el-form-item>
              </el-form>
            </el-collapse-item>

            <!-- 推送配置 -->
            <el-collapse-item title="推送配置 (Push Config)" name="push">
              <el-tabs type="border-card">
                <!-- 华为推送 -->
                <el-tab-pane label="华为 (HMS)">
                  <el-form label-width="120px" size="default">
                    <el-form-item label="App ID">
                      <el-input v-model="xFormOpt.data.push_config.push_hms.app_id" placeholder="华为 App ID" clearable />
                    </el-form-item>
                    <el-form-item label="App Secret">
                      <el-input v-model="xFormOpt.data.push_config.push_hms.app_secret" type="password" show-password placeholder="华为 App Secret" clearable />
                    </el-form-item>
                  </el-form>
                </el-tab-pane>

                <!-- 小米推送 -->
                <el-tab-pane label="小米 (Xiaomi)">
                  <el-form label-width="120px" size="default">
                    <el-form-item label="App Secret">
                      <el-input v-model="xFormOpt.data.push_config.push_xiaomi.app_secret" type="password" show-password placeholder="小米 App Secret" clearable />
                    </el-form-item>
                    <el-form-item label="Package">
                      <el-input v-model="xFormOpt.data.push_config.push_xiaomi.package" placeholder="应用包名" clearable />
                    </el-form-item>
                    <el-form-item label="Channel ID">
                      <el-input v-model="xFormOpt.data.push_config.push_xiaomi.channel_id" placeholder="渠道ID" clearable />
                    </el-form-item>
                  </el-form>
                </el-tab-pane>

                <!-- OPPO推送 -->
                <el-tab-pane label="OPPO">
                  <el-form label-width="120px" size="default">
                    <el-form-item label="App Key">
                      <el-input v-model="xFormOpt.data.push_config.push_oppo.app_key" placeholder="OPPO App Key" clearable />
                    </el-form-item>
                    <el-form-item label="Master Secret">
                      <el-input v-model="xFormOpt.data.push_config.push_oppo.master_secret" type="password" show-password placeholder="OPPO Master Secret" clearable />
                    </el-form-item>
                    <el-form-item label="Channel ID">
                      <el-input v-model="xFormOpt.data.push_config.push_oppo.channel_id" placeholder="渠道ID" clearable />
                    </el-form-item>
                  </el-form>
                </el-tab-pane>

                <!-- Vivo推送 -->
                <el-tab-pane label="Vivo">
                  <el-form label-width="120px" size="default">
                    <el-form-item label="App ID">
                      <el-input-number v-model="xFormOpt.data.push_config.push_vivo.app_id" :min="0" placeholder="Vivo App ID" style="width: 100%;" />
                    </el-form-item>
                    <el-form-item label="App Key">
                      <el-input v-model="xFormOpt.data.push_config.push_vivo.app_key" placeholder="Vivo App Key" clearable />
                    </el-form-item>
                    <el-form-item label="App Secret">
                      <el-input v-model="xFormOpt.data.push_config.push_vivo.app_secret" type="password" show-password placeholder="Vivo App Secret" clearable />
                    </el-form-item>
                  </el-form>
                </el-tab-pane>

                <!-- 荣耀推送 -->
                <el-tab-pane label="荣耀 (Honor)">
                  <el-form label-width="120px" size="default">
                    <el-form-item label="App ID">
                      <el-input v-model="xFormOpt.data.push_config.push_honor.app_id" placeholder="荣耀 App ID" clearable />
                    </el-form-item>
                    <el-form-item label="Client ID">
                      <el-input v-model="xFormOpt.data.push_config.push_honor.client_id" placeholder="荣耀 Client ID" clearable />
                    </el-form-item>
                    <el-form-item label="Client Secret">
                      <el-input v-model="xFormOpt.data.push_config.push_honor.client_secret" type="password" show-password placeholder="荣耀 Client Secret" clearable />
                    </el-form-item>
                  </el-form>
                </el-tab-pane>
              </el-tabs>
            </el-collapse-item>
          </el-collapse>
        </template>
      </vxe-form>
    </vxe-modal>
  </div>
</template>

<style lang="scss" scoped>
.app-container {
  padding: 20px;
}
</style>
