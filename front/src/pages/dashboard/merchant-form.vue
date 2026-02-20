<script lang="ts" setup>
import type { FormInstance, FormRules } from "element-plus"
import type { CreateOrEditMerchantRequestData, Merchant } from "./apis/type"
import { request } from "@/http/axios"
import { getCloudAccountList } from "@@/apis/cloud_account"
import { deployTSDDWithAMI } from "@@/apis/deploy"
import { checkPortAvailable, port2Enterprise } from "@@/apis/utils"
import { getAwsRegions } from "@@/constants/aws-regions"
import { useFullscreenLoading } from "@@/composables/useFullscreenLoading"
import { cloneDeep } from "lodash-es"
import { createMerchantApi, updateMerchantApi } from "./apis"
import ImageUploader from "@@/components/ImageUploader.vue"

defineOptions({ name: "MerchantFormPage" })

const router = useRouter()
const route = useRoute()

const isEdit = computed(() => !!route.params.id)
const pageTitle = computed(() => isEdit.value ? "编辑商户" : "新增商户")

// 从路由获取商户数据（编辑模式）
const routeMerchant = computed(() => route.query.data ? JSON.parse(route.query.data as string) as Merchant : null)

const formRef = ref<FormInstance | null>(null)
const loading = ref(false)

// 表单数据
const formData = ref<CreateOrEditMerchantRequestData>({
  name: "",
  app_name: "",
  logo_url: "",
  icon_url: "",
  no: "",
  port: 0,
  server_ip: "",
  status: 1,
  expired_at: "",
  package_configuration: {
    dau_limit: 100,
    register_limit: 100,
    group_member_limit: 100,
    turn_server: "8.218.153.228:3478"
  },
  aws_access_key_id: "",
  aws_access_key_secret: ""
})

// AWS 账号相关
const systemAwsAccounts = ref<Array<{ id: number, name: string, access_key_id: string }>>([])
const selectedSystemAccountId = ref<number | undefined>(undefined)
const removeFromSystem = ref(false)

// 系统服务器相关
const systemServers = ref<Array<{ id: number, name: string, host: string }>>([])
const selectedServers = ref<number[]>([])

// 服务器来源模式: manual=手动填写IP, ami=从AWS AMI部署
const serverSourceMode = ref<"manual" | "ami">("manual")

// AMI 部署配置
const awsRegions = getAwsRegions("cn")
const amiDeployConfig = reactive({
  cloud_account_id: undefined as number | undefined,
  region_id: "ap-east-1", // 默认香港
  ami_id: "", // 可选，留空使用默认 TSDD AMI
  source_server_id: undefined as number | undefined, // 从已有服务器克隆
  instance_type: "t3.medium",
  volume_size_gib: 30,
  // EBS 数据卷配置
  enable_extra_ebs: false,
  db_volume_size_gib: 20,
  db_volume_iops: 3000,
  minio_volume_size_gib: 50,
  minio_volume_iops: 3000,
})
const amiDeploying = ref(false)
// AMI 来源模式: fresh=全新部署, clone=克隆已有服务器
const amiSourceMode = ref<"fresh" | "clone">("fresh")
// 可克隆的商户服务器列表
const cloneableServers = ref<Array<{ id: number, name: string, host: string }>>([])
async function loadCloneableServers() {
  try {
    const res = await request<any>({
      url: "/deploy/servers",
      method: "get",
      params: { server_type: 1, page: 1, size: 1000 }
    })
    cloneableServers.value = (res.data.list || []).filter((s: any) => s.host && s.status === 1).map((s: any) => ({
      id: s.id,
      name: s.name || `Server #${s.id}`,
      host: s.host
    }))
  } catch (error) {
    console.error("加载可克隆服务器失败", error)
  }
}


// 分页配置
const pagination = reactive({
  current: 1,
  pageSize: 10,
  total: 0
})

// 加载系统AWS账号列表
async function loadSystemAwsAccounts() {
  try {
    const { data } = await getCloudAccountList({
      page: 1,
      size: 100,
      cloud_type: "aws",
      status: 1,
      account_type: "system"
    } as any)
    systemAwsAccounts.value = (data.list || []).map(acc => ({
      id: acc.id,
      name: acc.name,
      access_key_id: acc.access_key_id
    }))
  }
  catch (error) {
    console.error("加载系统AWS账号失败", error)
  }
}

// 加载系统服务器列表
async function loadSystemServers() {
  try {
    const res = await request<any>({
      url: "/deploy/servers",
      method: "get",
      params: { server_type: 2, page: 1, size: 1000 }
    })
    systemServers.value = (res.data.list || []).map((s: any) => ({
      id: s.id,
      name: s.name,
      host: s.host
    }))
    pagination.total = systemServers.value.length
  }
  catch (error) {
    console.error("加载系统服务器失败", error)
  }
}


// 加载商户详情（编辑模式）
async function loadMerchantDetail() {
  if (!isEdit.value || !routeMerchant.value)
    return
  loading.value = true
  try {
    const data = routeMerchant.value
    formData.value = {
      id: data.id,
      name: data.name,
      app_name: data.app_name || "",
      logo_url: data.logo_url || "",
      icon_url: data.icon_url || "",
      no: data.no,
      port: data.port,
      server_ip: data.server_ip,
      status: data.status,
      expired_at: data.expired_at,
      package_configuration: data.package_configuration
        ? cloneDeep(data.package_configuration)
        : {
            dau_limit: 100,
            register_limit: 100,
            group_member_limit: 100,
            turn_server: ""
          },
      aws_access_key_id: "",
      aws_access_key_secret: ""
    }

    // 编辑模式：加载当前商户的 AWS 云账号
    const awsRes = await getCloudAccountList({
      page: 1,
      size: 1,
      cloud_type: "aws",
      status: 1,
      account_type: "merchant",
      merchant_id: data.id
    } as any)
    const acc = (awsRes.data.list || [])[0]
    if (acc) {
      formData.value.aws_access_key_id = acc.access_key_id || ""
    }
  }
  catch {
    ElMessage.error("加载商户信息失败")
    router.back()
  }
  finally {
    loading.value = false
  }
}

// 表单校验规则
const formRules = computed<FormRules>(() => ({
  name: [{ required: true, message: "请输入商户名称", trigger: "blur" }],
  port: [{ required: !isEdit.value, message: "请输入端口", trigger: "blur" }],
  // 手动模式时需要填写 IP，AMI 模式时不需要
  server_ip: [{ required: !isEdit.value && serverSourceMode.value === "manual", message: "请输入服务器IP", trigger: "blur" }],
  expired_at: [{ required: true, message: "请选择服务过期时间", trigger: "blur" }]
}))

// 当选择系统AWS账号时，自动填充
function onSelectSystemAccount(accountId: number | undefined) {
  if (!accountId) {
    formData.value.aws_access_key_id = ""
    formData.value.aws_access_key_secret = ""
    formData.value.selected_aws_account_id = undefined
    removeFromSystem.value = false
    return
  }
  const account = systemAwsAccounts.value.find(acc => acc.id === accountId)
  if (account) {
    formData.value.aws_access_key_id = account.access_key_id
    formData.value.aws_access_key_secret = ""
    formData.value.selected_aws_account_id = accountId
    removeFromSystem.value = true
  }
}

// 端口变化时计算企业号
function onPortChange(val: number | undefined) {
  if (!val || isEdit.value) return
  formData.value.port = val
  port2Enterprise({ port: val })
    .then(({ data }) => {
      formData.value.no = data.enterprise
    })
    .catch((err) => {
      console.error("获取企业号失败", err)
    })
}

// 端口失焦（端口唯一性检查已移除，每个商户配独立隧道）
function onPortBlur() {
  // no-op
}

// 提交表单
async function onSubmit() {
  const valid = await formRef.value?.validate()
  if (!valid) {
    ElMessage.error("请完善表单信息")
    return
  }

  // AMI 模式额外校验
  if (!isEdit.value && serverSourceMode.value === "ami") {
    if (!amiDeployConfig.cloud_account_id) {
      ElMessage.error("请选择 AWS 云账号")
      return
    }
    if (!amiDeployConfig.region_id) {
      ElMessage.error("请选择部署区域")
      return
    }
    if (amiSourceMode.value === "clone" && !amiDeployConfig.source_server_id) {
      ElMessage.error("请选择要克隆的源服务器")
      return
    }
  }

  // 规范化数据
  const payload = cloneDeep(formData.value)
  // AMI 模式：自动使用 AMI 配置的 AWS 账号作为商户云账号
  if (!isEdit.value && serverSourceMode.value === "ami" && amiDeployConfig.cloud_account_id) {
    payload.selected_aws_account_id = amiDeployConfig.cloud_account_id
    payload.remove_from_system = false // 复制一份，不移除系统账号
  } else if (selectedSystemAccountId.value) {
    // 手动模式：使用手动选择的系统账号
    payload.selected_aws_account_id = selectedSystemAccountId.value
    payload.remove_from_system = removeFromSystem.value
  }

  // 添加选中的系统服务器ID列表（仅创建模式）
  if (!isEdit.value) {
    (payload as any).sync_gost_servers = selectedServers.value
  }

  const runWithLoading = useFullscreenLoading(async () => {
    try {
      if (isEdit.value) {
        payload.id = Number(route.params.id)
        // 编辑模式不允许修改端口
        delete (payload as any).port
        await updateMerchantApi(payload)
        ElMessage.success("修改成功")
        router.push({ name: "Dashboard" })
        return true
      }
      else {
        delete payload.id

        // AMI 部署模式
        if (serverSourceMode.value === "ami") {
          // 先创建商户（IP 暂时为空或占位）
          payload.server_ip = "pending-ami-deploy"
          const createRes = await createMerchantApi(payload)
          const merchantId = (createRes as any).data?.id

          if (!merchantId) {
            ElMessage.error("创建商户失败，无法获取商户ID")
            return false
          }

          // 调用 AMI 部署
          amiDeploying.value = true
          ElMessage.info("正在部署 AWS EC2 实例，请稍候...")

          try {
            const deployRes = await deployTSDDWithAMI({
              merchant_id: merchantId,
              cloud_account_id: amiDeployConfig.cloud_account_id!,
              region_id: amiDeployConfig.region_id,
              ami_id: amiDeployConfig.ami_id || undefined,
              source_server_id: amiSourceMode.value === "clone" ? amiDeployConfig.source_server_id : undefined,
              instance_type: amiDeployConfig.instance_type,
              volume_size_gib: amiDeployConfig.volume_size_gib,
              server_name: payload.name,
              enable_extra_ebs: amiDeployConfig.enable_extra_ebs,
              db_volume_size_gib: amiDeployConfig.enable_extra_ebs ? amiDeployConfig.db_volume_size_gib : undefined,
              db_volume_iops: amiDeployConfig.enable_extra_ebs ? amiDeployConfig.db_volume_iops : undefined,
              minio_volume_size_gib: amiDeployConfig.enable_extra_ebs ? amiDeployConfig.minio_volume_size_gib : undefined,
              minio_volume_iops: amiDeployConfig.enable_extra_ebs ? amiDeployConfig.minio_volume_iops : undefined,
            })

            const publicIp = deployRes.data?.public_ip
            if (publicIp) {
              // 更新商户的服务器 IP
              await updateMerchantApi({
                id: merchantId,
                name: payload.name,
                server_ip: publicIp,
                expired_at: payload.expired_at
              })
              let successMsg = `部署成功！服务器 IP: ${publicIp}`
              if (deployRes.data?.db_volume_id) {
                successMsg += ` | DB磁盘: ${deployRes.data.db_volume_id}`
              }
              if (deployRes.data?.minio_volume_id) {
                successMsg += ` | MinIO磁盘: ${deployRes.data.minio_volume_id}`
              }
              ElMessage.success(successMsg)
            } else {
              ElMessage.warning("部署完成，但未获取到公网 IP，请手动更新")
            }
          } catch (deployError: any) {
            ElMessage.error(`AMI 部署失败: ${deployError.message || "未知错误"}，商户已创建，请手动配置服务器`)
          } finally {
            amiDeploying.value = false
          }
        } else {
          // 手动模式，直接创建
          await createMerchantApi(payload)
          ElMessage.success("添加成功")
        }

        router.push({ name: "Dashboard" })
        return true
      }
    }
    catch {
      ElMessage.error(isEdit.value ? "修改失败" : "添加失败")
      return false
    }
  }, { text: serverSourceMode.value === "ami" ? "创建商户并部署服务器中..." : "保存中，请稍候..." })

  await runWithLoading()
}

// 取消
function onCancel() {
  router.back()
}

// 监听系统服务器列表变化，更新总数
watch(() => systemServers.value, (list) => {
  pagination.total = list.length
  // 如果当前页超出范围，回到第一页
  const maxPage = Math.ceil(list.length / pagination.pageSize) || 1
  if (pagination.current > maxPage) {
    pagination.current = 1
  }
}, { immediate: true })

onMounted(() => {
  if (!isEdit.value) {
    loadSystemAwsAccounts()
    loadSystemServers()
    loadCloneableServers()
  }
  else {
    loadMerchantDetail()
  }
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
            <el-button type="primary" @click="onSubmit">
              保存
            </el-button>
          </div>
        </div>
      </template>

      <el-form ref="formRef" :model="formData" :rules="formRules" label-width="140px" class="merchant-form">
        <!-- 商户名称 -->
        <el-form-item label="商户名称" prop="name">
          <el-input v-model="formData.name" placeholder="请输入商户名称" />
        </el-form-item>

        <!-- 应用名称 -->
        <el-form-item label="应用名称">
          <el-input v-model="formData.app_name" placeholder="打包时显示的应用名称（可选）" />
          <div class="form-tip">用于打包时显示的应用名称，如不填则使用商户名称</div>
        </el-form-item>

        <el-divider content-position="left">
          打包资源（可选）
        </el-divider>

        <!-- 应用 Logo（同时用于启动页和桌面图标） -->
        <el-form-item label="应用 Logo">
          <div class="image-upload-row">
            <ImageUploader
              v-model="formData.logo_url"
              :width="200"
              :height="200"
              asset-type="logo"
              tip="用于启动页和手机桌面图标，必须方形，推荐尺寸 1024x1024"
              @update:model-value="(url: string) => { formData.logo_url = url; formData.icon_url = url }"
            />
          </div>
        </el-form-item>

        <el-divider content-position="left">
          服务器配置
        </el-divider>

        <!-- 端口 -->
        <el-form-item label="端口" prop="port">
          <el-input-number
            v-model="formData.port"
            :min="1"
            :max="65535"
            :disabled="isEdit"
            style="width: 200px"
            @change="onPortChange"
            @blur="onPortBlur"
          />
          <span class="ml-2 text-gray-500">{{ isEdit ? "编辑时不可修改端口" : "用于自动化部署和GOST配置" }}</span>
        </el-form-item>

        <!-- 服务器来源（仅新增模式） -->
        <el-form-item v-if="!isEdit" label="服务器来源">
          <el-radio-group v-model="serverSourceMode">
            <el-radio value="manual">手动填写 IP</el-radio>
            <el-radio value="ami">从 AWS AMI 部署</el-radio>
          </el-radio-group>
        </el-form-item>

        <!-- 手动模式：服务器IP -->
        <el-form-item v-if="isEdit || serverSourceMode === 'manual'" label="服务器IP" prop="server_ip">
          <el-input v-model="formData.server_ip" placeholder="请输入服务器IP" />
        </el-form-item>

        <!-- AMI 部署模式 -->
        <template v-if="!isEdit && serverSourceMode === 'ami'">
          <el-card class="ami-deploy-card" shadow="never">
            <template #header>
              <span class="ami-card-title">AWS AMI 部署配置</span>
            </template>

            <el-form-item label="AWS 账号" required>
              <el-select
                v-model="amiDeployConfig.cloud_account_id"
                placeholder="选择 AWS 云账号"
                filterable
                style="width: 100%"
              >
                <el-option
                  v-for="acc in systemAwsAccounts"
                  :key="acc.id"
                  :label="`${acc.name} (${acc.access_key_id})`"
                  :value="acc.id"
                />
              </el-select>
            </el-form-item>

            <el-form-item label="部署区域" required>
              <el-select
                v-model="amiDeployConfig.region_id"
                placeholder="选择 AWS 区域"
                filterable
                style="width: 100%"
              >
                <el-option
                  v-for="r in awsRegions"
                  :key="r.id"
                  :label="`${r.name} (${r.id})`"
                  :value="r.id"
                />
              </el-select>
            </el-form-item>

            <el-row :gutter="16">
              <el-col :span="12">
                <el-form-item label="实例类型">
                  <el-input v-model="amiDeployConfig.instance_type" placeholder="如 t3.medium" />
                </el-form-item>
              </el-col>
              <el-col :span="12">
                <el-form-item label="磁盘大小(GB)">
                  <el-input-number
                    v-model="amiDeployConfig.volume_size_gib"
                    :min="20"
                    :max="1000"
                    style="width: 100%"
                  />
                </el-form-item>
              </el-col>
            </el-row>

            <el-form-item label="部署来源">
              <el-radio-group v-model="amiSourceMode">
                <el-radio-button value="clone">克隆已有服务器</el-radio-button>
                <el-radio-button value="fresh">全新部署</el-radio-button>
              </el-radio-group>
              <div class="form-tip" v-if="amiSourceMode === 'clone'">从已有商户服务器创建 AMI 并部署（保留所有配置、自定义 Web、端口等）</div>
              <div class="form-tip" v-else>使用默认 TSDD 镜像全新安装</div>
            </el-form-item>

            <el-form-item v-if="amiSourceMode === 'clone'" label="源服务器" required>
              <el-select
                v-model="amiDeployConfig.source_server_id"
                placeholder="选择要克隆的服务器"
                filterable
                style="width: 100%"
              >
                <el-option
                  v-for="s in cloneableServers"
                  :key="s.id"
                  :label="`${s.name} (${s.host})`"
                  :value="s.id"
                />
              </el-select>
            </el-form-item>

            <el-form-item v-if="amiSourceMode === 'fresh'" label="AMI ID">
              <el-input v-model="amiDeployConfig.ami_id" placeholder="留空使用默认 TSDD 镜像" />
              <div class="form-tip">可选，留空将使用系统预设的 TSDD AMI 镜像</div>
            </el-form-item>

            <!-- EBS 独立数据磁盘 -->
            <el-form-item label="独立数据磁盘">
              <el-switch
                v-model="amiDeployConfig.enable_extra_ebs"
                active-text="启用"
                inactive-text="关闭"
              />
              <div class="form-tip" style="margin-top: 4px;">创建独立 EBS 数据盘，DB 和 MinIO 数据与系统盘隔离</div>
            </el-form-item>

            <template v-if="amiDeployConfig.enable_extra_ebs">
              <el-divider content-position="left" style="margin: 8px 0 16px;">数据库磁盘 (MySQL + Redis + WuKongIM)</el-divider>
              <el-row :gutter="16">
                <el-col :span="12">
                  <el-form-item label="磁盘大小(GB)">
                    <el-input-number
                      v-model="amiDeployConfig.db_volume_size_gib"
                      :min="10"
                      :max="1000"
                      style="width: 100%"
                    />
                  </el-form-item>
                </el-col>
                <el-col :span="12">
                  <el-form-item label="IOPS">
                    <el-input-number
                      v-model="amiDeployConfig.db_volume_iops"
                      :min="3000"
                      :max="16000"
                      :step="1000"
                      style="width: 100%"
                    />
                  </el-form-item>
                </el-col>
              </el-row>

              <el-divider content-position="left" style="margin: 8px 0 16px;">文件存储磁盘 (MinIO)</el-divider>
              <el-row :gutter="16">
                <el-col :span="12">
                  <el-form-item label="磁盘大小(GB)">
                    <el-input-number
                      v-model="amiDeployConfig.minio_volume_size_gib"
                      :min="10"
                      :max="2000"
                      style="width: 100%"
                    />
                  </el-form-item>
                </el-col>
                <el-col :span="12">
                  <el-form-item label="IOPS">
                    <el-input-number
                      v-model="amiDeployConfig.minio_volume_iops"
                      :min="3000"
                      :max="16000"
                      :step="1000"
                      style="width: 100%"
                    />
                  </el-form-item>
                </el-col>
              </el-row>
            </template>
          </el-card>
        </template>

        <!-- 过期时间 -->
        <el-form-item label="过期时间" prop="expired_at">
          <el-date-picker
            v-model="formData.expired_at"
            type="datetime"
            placeholder="选择过期时间"
            format="YYYY-MM-DD HH:mm:ss"
            value-format="YYYY-MM-DD HH:mm:ss"
            style="width: 100%"
          />
        </el-form-item>

        <el-divider content-position="left">
          套餐配置
        </el-divider>

        <!-- 日活限制 -->
        <el-form-item label="日活限制">
          <el-input-number
            v-model="formData.package_configuration!.dau_limit"
            :min="0"
            :max="1000000"
            placeholder="日活限制"
          />
        </el-form-item>

        <!-- 注册人数限制 -->
        <el-form-item label="注册人数限制">
          <el-input-number
            v-model="formData.package_configuration!.register_limit"
            :min="0"
            :max="1000000"
            placeholder="注册人数限制"
          />
        </el-form-item>

        <!-- 群人数限制 -->
        <el-form-item label="群人数限制">
          <el-input-number
            v-model="formData.package_configuration!.group_member_limit"
            :min="0"
            :max="100000"
            placeholder="群人数限制"
          />
        </el-form-item>

        <!-- TURN服务器 -->
        <el-form-item label="TURN服务器">
          <el-input
            v-model="formData.package_configuration!.turn_server"
            placeholder="音视频TURN服务器地址 (格式: ip:port)"
          />
          <div class="form-tip" style="margin-top: 4px;">
            用于音视频通话的TURN服务器，格式如：192.168.1.100:3478
          </div>
        </el-form-item>

        <el-divider v-if="isEdit || serverSourceMode !== 'ami'" content-position="left">
          AWS 云账号
        </el-divider>

        <!-- 创建模式：选择系统AWS账号（AMI模式下自动使用AMI配置的账号，无需手动选择） -->
        <template v-if="!isEdit && serverSourceMode !== 'ami'">
          <el-form-item label="选择系统账号">
            <el-select
              v-model="selectedSystemAccountId"
              placeholder="可选择现有系统AWS账号"
              clearable
              filterable
              style="width: 100%"
              @change="onSelectSystemAccount"
            >
              <el-option
                v-for="acc in systemAwsAccounts"
                :key="acc.id"
                :label="`${acc.name} (${acc.access_key_id})`"
                :value="acc.id"
              />
            </el-select>
          </el-form-item>

          <el-form-item v-if="selectedSystemAccountId" label=" ">
            <el-checkbox v-model="removeFromSystem">
              从系统账号中移除（转为商户专用账号）
            </el-checkbox>
            <div class="text-gray-500" style="font-size: 12px; margin-top: 4px;">
              勾选后，该账号将从系统账号列表中移除，并转换为此商户的专用账号；<br>
              不勾选则复制一份新的账号给商户，系统账号保持不变
            </div>
          </el-form-item>

          <el-divider v-if="selectedSystemAccountId" content-position="center">
            或手动填写
          </el-divider>
        </template>

        <template v-if="isEdit || serverSourceMode !== 'ami'">
          <el-row :gutter="12">
            <el-col :span="12">
              <el-form-item label="AccessKey">
                <el-input
                  v-model="formData.aws_access_key_id"
                  :disabled="!!selectedSystemAccountId && !isEdit"
                  :placeholder="isEdit ? '留空表示不修改' : (selectedSystemAccountId ? '已自动填充' : '请输入 AWS Access Key')"
                />
              </el-form-item>
            </el-col>
            <el-col :span="12">
              <el-form-item label="SecretKey">
                <el-input
                  v-model="formData.aws_access_key_secret"
                  type="password"
                  show-password
                  :disabled="!!selectedSystemAccountId && !isEdit"
                  :placeholder="isEdit ? '留空表示不修改' : (selectedSystemAccountId ? '已自动使用系统账号密钥' : '请输入 AWS Access Secret')"
                />
              </el-form-item>
            </el-col>
          </el-row>
          <div v-if="isEdit" class="form-tip">
            留空表示不修改
          </div>
        </template>
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

.merchant-form {
  max-width: 900px;
  margin: 20px 0;
}

.form-tip {
  font-size: 12px;
  color: #909399;
  margin-top: -12px;
  margin-bottom: 18px;
}

.ami-deploy-card {
  margin-bottom: 20px;
  background: #f8fafc;
  border: 1px dashed #409eff;

  .ami-card-title {
    font-weight: 600;
    color: #409eff;
  }

  :deep(.el-card__header) {
    padding: 12px 16px;
    background: #ecf5ff;
    border-bottom: 1px dashed #409eff;
  }

  :deep(.el-card__body) {
    padding: 16px;
  }

  :deep(.el-form-item) {
    margin-bottom: 16px;
  }
}

.image-upload-row {
  display: flex;
  gap: 24px;
  align-items: flex-start;
}

.kv-list {
  width: 100%;

  .kv-item {
    display: flex;
    gap: 10px;
    margin-bottom: 10px;
  }

  .kv-add {
    display: flex;
    gap: 10px;
  }
}

.server-select-container {
  width: 100%;

  .empty-tip {
    color: #999;
    font-size: 12px;
    padding: 12px;
    background: #f5f7fa;
    border-radius: 4px;
    text-align: center;
  }

  .server-list-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 12px;
    padding-bottom: 8px;
    border-bottom: 1px solid #ebeef5;

    .info-text {
      font-size: 13px;
      color: #606266;
    }

    .header-actions {
      display: flex;
      gap: 8px;
    }
  }

  .server-checkbox-list {
    display: flex;
    flex-direction: column;
    border: 1px solid #dcdfe6;
    border-radius: 4px;
    padding: 12px;
    background: #fafafa;
    min-height: 60px;

    .server-checkbox-item {
      display: flex;
      align-items: center;
      padding: 12px;
      margin: 0 0 8px 0;
      background: white;
      border-radius: 4px;
      border: 1px solid #e4e7ed;
      transition: all 0.3s;

      &:hover {
        border-color: #409eff;
        box-shadow: 0 2px 4px rgba(64, 158, 255, 0.1);
      }

      &:last-child {
        margin-bottom: 0;
      }

      :deep(.el-checkbox__label) {
        flex: 1;
        display: flex;
        flex-direction: column;
        gap: 4px;
      }

      .server-name {
        font-size: 14px;
        font-weight: 500;
        color: #303133;
      }

      .server-info {
        font-size: 12px;
        color: #909399;
      }
    }
  }

  .server-pagination {
    margin-top: 16px;
    display: flex;
    justify-content: center;
  }
}
</style>
