<script lang="ts" setup>
import type { ClusterWizardStep } from "@@/apis/deploy/type"
import type { FormInstance, FormRules } from "element-plus"
import { getCloudAccountList } from "@@/apis/cloud_account"
import { getAwsRegions } from "@@/constants/aws-regions"
import { createStreamRequest } from "@/http/axios"

defineOptions({ name: "ClusterWizardPage" })

const router = useRouter()

// ========== 表单数据 ==========
const formRef = ref<FormInstance | null>(null)
const formData = reactive({
  merchant_name: "",
  app_name: "",
  port: 0,
  expired_at: "",
  cloud_account_id: undefined as number | undefined,
  region_id: "ap-east-1",
  key_name: "tsdd-deploy-key",
  subnet_id: "",
  db_ami_id: "",
  db_instance_type: "r5.large",
  db_volume_size_gib: 100,
  minio_ami_id: "",
  minio_instance_type: "t3.medium",
  minio_volume_size_gib: 200,
  app_ami_id: "",
  app_instance_type: "t3.large",
  app_volume_size_gib: 30
})

const formRules: FormRules = {
  merchant_name: [{ required: true, message: "请输入商户名称", trigger: "blur" }],
  port: [{ required: true, message: "请输入端口", trigger: "blur" }],
  cloud_account_id: [{ required: true, message: "请选择 AWS 账号", trigger: "change" }],
  region_id: [{ required: true, message: "请选择区域", trigger: "change" }]
}

// ========== AWS 账号 ==========
const systemAwsAccounts = ref<Array<{ id: number; name: string; access_key_id: string }>>([])
const awsRegions = getAwsRegions("cn")

async function loadSystemAwsAccounts() {
  try {
    const { data } = await getCloudAccountList({
      page: 1,
      size: 100,
      cloud_type: "aws",
      status: 1,
      account_type: "system"
    } as any)
    systemAwsAccounts.value = (data.list || []).map((acc: any) => ({
      id: acc.id,
      name: acc.name,
      access_key_id: acc.access_key_id
    }))
  } catch (error) {
    console.error("加载系统AWS账号失败", error)
  }
}

// ========== 部署进度 ==========
const deploying = ref(false)
const deployFinished = ref(false)
const deploySuccess = ref(false)
const deployError = ref("")
const progressDialogVisible = ref(false)
const steps = ref<ClusterWizardStep[]>([])
const wizardMerchantId = ref(0) // 用于重试失败节点
const resumeMerchantId = ref(0) // 手动输入恢复商户ID
let cancelStream: (() => void) | null = null

// 步骤标题预定义
const stepTitles = [
  "创建商户记录",
  "创建 DB EC2 实例",
  "创建 MinIO EC2 实例",
  "创建 App EC2 实例",
  "注册服务器",
  "部署 DB 节点",
  "部署 MinIO 节点",
  "部署 App 节点"
]

function initSteps() {
  steps.value = stepTitles.map((title, idx) => ({
    step: idx + 1,
    total: stepTitles.length,
    title,
    status: "pending",
    message: ""
  }))
}

function stepIcon(status: string) {
  if (status === "success") return "SuccessFilled"
  if (status === "failed") return "CircleCloseFilled"
  if (status === "running") return "Loading"
  return "MoreFilled"
}

function stepColor(status: string) {
  if (status === "success") return "#67c23a"
  if (status === "failed") return "#f56c6c"
  if (status === "running") return "#409eff"
  if (status === "skipped") return "#e6a23c"
  return "#dcdfe6"
}

// ========== 提交部署 ==========
async function doSubmit(retryMerchantId?: number) {
  if (!retryMerchantId) {
    // 新建模式需要验证表单
    const valid = await formRef.value?.validate()
    if (!valid) {
      ElMessage.error("请完善表单信息")
      return
    }
    if (!formData.cloud_account_id) {
      ElMessage.error("请选择 AWS 云账号")
      return
    }
    if (formData.port < 10000) {
      ElMessage.error("端口必须 >= 10000")
      return
    }
  }

  // 初始化进度
  initSteps()
  deploying.value = true
  deployFinished.value = false
  deploySuccess.value = false
  deployError.value = ""
  progressDialogVisible.value = true

  cancelStream = createStreamRequest(
    {
      url: "deploy/tsdd/cluster-wizard",
      method: "post",
      data: {
        merchant_id: retryMerchantId || undefined,
        merchant_name: formData.merchant_name,
        app_name: formData.app_name || undefined,
        port: formData.port,
        expired_at: formData.expired_at || undefined,
        cloud_account_id: formData.cloud_account_id,
        region_id: formData.region_id,
        key_name: formData.key_name || undefined,
        subnet_id: formData.subnet_id || undefined,
        db_ami_id: formData.db_ami_id || undefined,
        db_instance_type: formData.db_instance_type,
        db_volume_size_gib: formData.db_volume_size_gib,
        minio_ami_id: formData.minio_ami_id || undefined,
        minio_instance_type: formData.minio_instance_type,
        minio_volume_size_gib: formData.minio_volume_size_gib,
        app_ami_id: formData.app_ami_id || undefined,
        app_instance_type: formData.app_instance_type,
        app_volume_size_gib: formData.app_volume_size_gib
      }
    },
    (data: any, isComplete?: boolean) => {
      if (isComplete) {
        deploying.value = false
        deployFinished.value = true
        deploySuccess.value = data?.success === true
        if (!deploySuccess.value) {
          deployError.value = data?.message || "部署失败"
        }
        return
      }

      // 捕获 merchant_id（Step 1 成功时返回）
      if (data?.merchant_id) {
        wizardMerchantId.value = data.merchant_id
        resumeMerchantId.value = data.merchant_id // 同步到表单，方便刷新后使用
      }

      // 进度更新
      if (data?.step) {
        const idx = data.step - 1
        if (idx >= 0 && idx < steps.value.length) {
          steps.value[idx] = {
            step: data.step,
            total: data.total || stepTitles.length,
            title: data.title || steps.value[idx].title,
            status: data.status,
            message: data.message || ""
          }
        }
      }
    },
    (error: any) => {
      deploying.value = false
      deployFinished.value = true
      deploySuccess.value = false
      deployError.value = error?.message || "连接失败"
    }
  )
}

async function onSubmit() {
  await doSubmit()
}

function onRetry() {
  if (wizardMerchantId.value > 0) {
    doSubmit(wizardMerchantId.value)
  }
}

function onCancel() {
  router.back()
}

function onCloseProgress() {
  if (deploying.value) return
  progressDialogVisible.value = false
  if (deploySuccess.value) {
    router.push({ name: "DeployCluster" })
  }
}

onMounted(() => {
  loadSystemAwsAccounts()
  // 默认过期时间 30 天后
  const d = new Date()
  d.setDate(d.getDate() + 30)
  formData.expired_at = d.toISOString().replace("T", " ").substring(0, 19)
})

onUnmounted(() => {
  if (cancelStream) cancelStream()
})
</script>

<template>
  <div class="app-container">
    <el-card>
      <template #header>
        <div class="card-header">
          <span class="title">集群商户向导</span>
          <div class="actions">
            <el-input-number
              v-model="resumeMerchantId"
              :min="0"
              placeholder="恢复商户ID"
              style="width: 160px; margin-right: 8px;"
              :controls="false"
            />
            <el-button v-if="resumeMerchantId > 0" type="warning" @click="doSubmit(resumeMerchantId)" :loading="deploying">
              恢复部署
            </el-button>
            <el-button @click="onCancel">取消</el-button>
            <el-button v-if="!resumeMerchantId" type="primary" @click="onSubmit" :loading="deploying">开始部署</el-button>
          </div>
        </div>
      </template>

      <el-form ref="formRef" :model="formData" :rules="formRules" label-width="140px" class="wizard-form">
        <!-- 商户信息 -->
        <el-divider content-position="left">商户信息</el-divider>

        <el-row :gutter="16">
          <el-col :span="12">
            <el-form-item label="商户名称" prop="merchant_name">
              <el-input v-model="formData.merchant_name" placeholder="请输入商户名称" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="应用名称">
              <el-input v-model="formData.app_name" placeholder="可选，打包时显示" />
            </el-form-item>
          </el-col>
        </el-row>

        <el-row :gutter="16">
          <el-col :span="8">
            <el-form-item label="端口" prop="port">
              <el-input-number v-model="formData.port" :min="10000" :max="65535" style="width: 100%" />
            </el-form-item>
          </el-col>
          <el-col :span="16">
            <el-form-item label="过期时间">
              <el-date-picker
                v-model="formData.expired_at"
                type="datetime"
                placeholder="选择过期时间"
                format="YYYY-MM-DD HH:mm:ss"
                value-format="YYYY-MM-DD HH:mm:ss"
                style="width: 100%"
              />
            </el-form-item>
          </el-col>
        </el-row>

        <!-- AWS 配置 -->
        <el-divider content-position="left">AWS 配置</el-divider>

        <el-row :gutter="16">
          <el-col :span="12">
            <el-form-item label="AWS 账号" prop="cloud_account_id">
              <el-select v-model="formData.cloud_account_id" placeholder="选择系统 AWS 账号" filterable style="width: 100%">
                <el-option
                  v-for="acc in systemAwsAccounts"
                  :key="acc.id"
                  :label="`${acc.name} (${acc.access_key_id})`"
                  :value="acc.id"
                />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="部署区域" prop="region_id">
              <el-select v-model="formData.region_id" placeholder="选择 AWS 区域" filterable style="width: 100%">
                <el-option v-for="r in awsRegions" :key="r.id" :label="`${r.name} (${r.id})`" :value="r.id" />
              </el-select>
            </el-form-item>
          </el-col>
        </el-row>

        <el-row :gutter="16">
          <el-col :span="12">
            <el-form-item label="Key Name">
              <el-input v-model="formData.key_name" placeholder="可选，SSH Key Pair 名称" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="Subnet ID">
              <el-input v-model="formData.subnet_id" placeholder="可选，指定子网" />
            </el-form-item>
          </el-col>
        </el-row>

        <!-- 集群节点配置 -->
        <el-divider content-position="left">集群节点配置</el-divider>

        <el-row :gutter="16">
          <!-- DB 节点 -->
          <el-col :span="8">
            <el-card shadow="never" class="node-card node-db">
              <template #header>
                <div class="node-header">
                  <el-tag type="warning" size="small">DB</el-tag>
                  <span>数据库节点</span>
                </div>
              </template>
              <div class="node-desc">MySQL + Redis</div>
              <el-form-item label="AMI" label-width="70px">
                <el-input v-model="formData.db_ami_id" placeholder="ami-xxx（空=默认Ubuntu）" />
              </el-form-item>
              <el-form-item label="机型" label-width="70px">
                <el-input v-model="formData.db_instance_type" placeholder="r5.large" />
              </el-form-item>
              <el-form-item label="磁盘" label-width="70px">
                <el-input-number v-model="formData.db_volume_size_gib" :min="20" :max="1000" style="width: 100%" />
                <span class="unit">GB</span>
              </el-form-item>
            </el-card>
          </el-col>

          <!-- MinIO 节点 -->
          <el-col :span="8">
            <el-card shadow="never" class="node-card node-minio">
              <template #header>
                <div class="node-header">
                  <el-tag type="info" size="small">MinIO</el-tag>
                  <span>对象存储节点</span>
                </div>
              </template>
              <div class="node-desc">MinIO 文件存储</div>
              <el-form-item label="AMI" label-width="70px">
                <el-input v-model="formData.minio_ami_id" placeholder="ami-xxx（空=默认Ubuntu）" />
              </el-form-item>
              <el-form-item label="机型" label-width="70px">
                <el-input v-model="formData.minio_instance_type" placeholder="t3.medium" />
              </el-form-item>
              <el-form-item label="磁盘" label-width="70px">
                <el-input-number v-model="formData.minio_volume_size_gib" :min="20" :max="2000" style="width: 100%" />
                <span class="unit">GB</span>
              </el-form-item>
            </el-card>
          </el-col>

          <!-- App 节点 -->
          <el-col :span="8">
            <el-card shadow="never" class="node-card node-app">
              <template #header>
                <div class="node-header">
                  <el-tag type="success" size="small">App</el-tag>
                  <span>应用节点</span>
                </div>
              </template>
              <div class="node-desc">WuKongIM + Server + Web</div>
              <el-form-item label="AMI" label-width="70px">
                <el-input v-model="formData.app_ami_id" placeholder="ami-xxx（空=默认Ubuntu）" />
              </el-form-item>
              <el-form-item label="机型" label-width="70px">
                <el-input v-model="formData.app_instance_type" placeholder="t3.large" />
              </el-form-item>
              <el-form-item label="磁盘" label-width="70px">
                <el-input-number v-model="formData.app_volume_size_gib" :min="20" :max="500" style="width: 100%" />
                <span class="unit">GB</span>
              </el-form-item>
            </el-card>
          </el-col>
        </el-row>
      </el-form>
    </el-card>

    <!-- 部署进度对话框 -->
    <el-dialog
      v-model="progressDialogVisible"
      title="集群部署进度"
      width="600px"
      :close-on-click-modal="false"
      :close-on-press-escape="false"
      :show-close="!deploying"
      @close="onCloseProgress"
    >
      <el-timeline>
        <el-timeline-item
          v-for="s in steps"
          :key="s.step"
          :color="stepColor(s.status)"
          :hollow="s.status === 'pending'"
        >
          <div class="step-row">
            <div class="step-title">
              <el-icon v-if="s.status === 'running'" class="is-loading" :size="14" style="margin-right: 4px; color: #409eff;"><Loading /></el-icon>
              <strong>{{ s.step }}/{{ s.total }}</strong>
              <span style="margin-left: 6px;">{{ s.title }}</span>
              <el-tag v-if="s.status === 'success'" type="success" size="small" style="margin-left: 8px;">完成</el-tag>
              <el-tag v-if="s.status === 'failed'" type="danger" size="small" style="margin-left: 8px;">失败</el-tag>
              <el-tag v-if="s.status === 'skipped'" type="warning" size="small" style="margin-left: 8px;">跳过</el-tag>
            </div>
            <div v-if="s.message" class="step-message">{{ s.message }}</div>
          </div>
        </el-timeline-item>
      </el-timeline>

      <!-- 完成状态 -->
      <div v-if="deployFinished" style="margin-top: 16px;">
        <el-result
          v-if="deploySuccess"
          icon="success"
          title="集群部署完成"
          sub-title="所有节点已成功部署，可在集群部署管理页查看详情"
        />
        <el-result
          v-else
          icon="warning"
          title="部分完成"
          :sub-title="deployError || '部分节点未完成，可修改配置后点击重试'"
        />
      </div>

      <template #footer>
        <el-button v-if="deploying" disabled>部署中...</el-button>
        <template v-else>
          <el-button @click="progressDialogVisible = false">关闭</el-button>
          <el-button v-if="!deploySuccess && wizardMerchantId > 0" type="warning" @click="onRetry">
            重试失败节点
          </el-button>
          <el-button v-if="deploySuccess" type="primary" @click="router.push({ name: 'DeployCluster' })">
            查看集群
          </el-button>
        </template>
      </template>
    </el-dialog>
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

.wizard-form {
  max-width: 1000px;
  margin: 20px 0;
}

.node-card {
  margin-bottom: 16px;

  .node-header {
    display: flex;
    align-items: center;
    gap: 8px;
    font-weight: 600;
    font-size: 14px;
  }

  .node-desc {
    font-size: 12px;
    color: #909399;
    margin-bottom: 12px;
  }

  :deep(.el-card__header) {
    padding: 10px 16px;
  }

  :deep(.el-card__body) {
    padding: 12px 16px;
  }

  :deep(.el-form-item) {
    margin-bottom: 12px;
  }
}

.node-db {
  border-top: 2px solid #e6a23c;
}

.node-minio {
  border-top: 2px solid #909399;
}

.node-app {
  border-top: 2px solid #67c23a;
}

.unit {
  margin-left: 4px;
  color: #909399;
  font-size: 12px;
}

.step-row {
  .step-title {
    display: flex;
    align-items: center;
    font-size: 14px;
  }

  .step-message {
    font-size: 12px;
    color: #606266;
    margin-top: 4px;
  }
}
</style>
