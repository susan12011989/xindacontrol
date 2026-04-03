<script lang="ts" setup>
import type { FormInstance } from "element-plus"
import { getCloudAccountList } from "@@/apis/cloud_account"
import { getAwsRegions } from "@@/constants/aws-regions"
import { createStreamRequest } from "@/http/axios"

defineOptions({ name: "ClusterWizardPage" })

const router = useRouter()
const formRef = ref<FormInstance | null>(null)

// 表单数据
const form = reactive({
  merchant_name: "",
  app_name: "",
  port: 0,
  expired_at: "",
  cloud_account_id: undefined as number | undefined,
  region_id: "ap-east-1",
  key_name: "tsdd-deploy-key",
  subnet_id: "",
  db_ami_id: "ami-05cf7289318975e63",
  db_instance_type: "r5.large",
  db_volume_size_gib: 100,
  minio_ami_id: "ami-0486d55a4b5362224",
  minio_instance_type: "t3.medium",
  minio_volume_size_gib: 200,
  app_ami_id: "ami-0cd8818611fe5048c",
  app_instance_type: "c6i.4xlarge",
  app_volume_size_gib: 30
})

const formRules = {
  merchant_name: [{ required: true, message: "请输入商户名称", trigger: "blur" }],
  port: [{ required: true, message: "请输入端口", trigger: "blur" }],
  cloud_account_id: [{ required: true, message: "请选择 AWS 账号", trigger: "change" }],
  region_id: [{ required: true, message: "请选择区域", trigger: "change" }]
}

// AWS 账号
const awsAccounts = ref<Array<{ id: number; name: string; access_key_id: string }>>([])
const awsRegions = getAwsRegions("cn")

async function loadAccounts() {
  try {
    const { data } = await getCloudAccountList({ page: 1, size: 100, cloud_type: "aws", status: 1, account_type: "system" } as any)
    awsAccounts.value = (data.list || []).map((acc: any) => ({ id: acc.id, name: acc.name, access_key_id: acc.access_key_id }))
  } catch (e) {
    console.error("加载AWS账号失败", e)
  }
}

// 部署状态
const deploying = ref(false)
const completed = ref(false)
const success = ref(false)
const errorMsg = ref("")
const showSteps = ref(false)
const merchantId = ref(0)
const resumeMerchantId = ref(0)

interface StepInfo {
  step: number
  total: number
  title: string
  status: string
  message: string
}

const stepNames = [
  "创建商户记录",
  "创建 DB EC2 实例",
  "创建 MinIO EC2 实例",
  "创建 App EC2 实例",
  "注册服务器",
  "部署 DB 节点",
  "部署 MinIO 节点",
  "部署 App 节点"
]

const steps = ref<StepInfo[]>([])

function initSteps() {
  steps.value = stepNames.map((title, i) => ({
    step: i + 1,
    total: stepNames.length,
    title,
    status: "pending",
    message: ""
  }))
}

function statusColor(status: string) {
  switch (status) {
    case "success": return "#67c23a"
    case "failed": return "#f56c6c"
    case "running": return "#409eff"
    case "skipped": return "#e6a23c"
    default: return "#dcdfe6"
  }
}

let cancelStream: (() => void) | null = null

async function startDeploy(resumeId?: number) {
  if (!resumeId) {
    const valid = await formRef.value?.validate()
    if (!valid) {
      ElMessage.error("请完善表单信息")
      return
    }
    if (!form.cloud_account_id) {
      ElMessage.error("请选择 AWS 云账号")
      return
    }
    if (form.port < 10000) {
      ElMessage.error("端口必须 >= 10000")
      return
    }
  }

  initSteps()
  deploying.value = true
  completed.value = false
  success.value = false
  errorMsg.value = ""
  showSteps.value = true

  cancelStream = createStreamRequest(
    {
      url: "deploy/tsdd/cluster-wizard",
      method: "post",
      data: {
        merchant_id: resumeId || undefined,
        merchant_name: form.merchant_name,
        app_name: form.app_name || undefined,
        port: form.port,
        expired_at: form.expired_at || undefined,
        cloud_account_id: form.cloud_account_id,
        region_id: form.region_id,
        key_name: form.key_name || undefined,
        subnet_id: form.subnet_id || undefined,
        db_ami_id: form.db_ami_id || undefined,
        db_instance_type: form.db_instance_type,
        db_volume_size_gib: form.db_volume_size_gib,
        minio_ami_id: form.minio_ami_id || undefined,
        minio_instance_type: form.minio_instance_type,
        minio_volume_size_gib: form.minio_volume_size_gib,
        app_ami_id: form.app_ami_id || undefined,
        app_instance_type: form.app_instance_type,
        app_volume_size_gib: form.app_volume_size_gib
      }
    },
    (data: any, isComplete?: boolean) => {
      if (isComplete) {
        deploying.value = false
        completed.value = true
        success.value = data?.success === true
        if (!success.value) errorMsg.value = data?.message || "部署失败"
        return
      }
      if (data?.merchant_id) {
        merchantId.value = data.merchant_id
        resumeMerchantId.value = data.merchant_id
      }
      if (data?.step) {
        const idx = data.step - 1
        if (idx >= 0 && idx < steps.value.length) {
          steps.value[idx] = {
            step: data.step,
            total: data.total || stepNames.length,
            title: data.title || steps.value[idx].title,
            status: data.status,
            message: data.message || ""
          }
        }
      }
    },
    (error: any) => {
      deploying.value = false
      completed.value = true
      success.value = false
      errorMsg.value = error?.message || "连接失败"
    }
  )
}

function resumeDeploy() {
  const mid = resumeMerchantId.value || merchantId.value
  if (mid > 0) startDeploy(mid)
  else ElMessage.warning("请输入商户 ID")
}

function goBack() {
  router.back()
}

function finish() {
  if (!deploying.value) {
    showSteps.value = false
    if (success.value) router.push({ name: "DeployCluster" })
  }
}

onMounted(() => {
  loadAccounts()
  const d = new Date()
  d.setDate(d.getDate() + 30)
  form.expired_at = d.toISOString().replace("T", " ").substring(0, 19)
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
            <el-input-number v-model="resumeMerchantId" :min="0" placeholder="恢复商户ID" style="width: 160px; margin-right: 8px" :controls="false" />
            <el-button v-if="resumeMerchantId > 0" type="warning" @click="resumeDeploy" :loading="deploying">恢复部署</el-button>
            <el-button @click="goBack">取消</el-button>
            <el-button v-if="!resumeMerchantId" type="primary" @click="startDeploy()" :loading="deploying">开始部署</el-button>
          </div>
        </div>
      </template>

      <el-form ref="formRef" :model="form" :rules="formRules" label-width="140px" class="wizard-form">
        <el-divider content-position="left">商户信息</el-divider>

        <el-row :gutter="16">
          <el-col :span="12">
            <el-form-item label="商户名称" prop="merchant_name">
              <el-input v-model="form.merchant_name" placeholder="请输入商户名称" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="应用名称">
              <el-input v-model="form.app_name" placeholder="可选，打包时显示" />
            </el-form-item>
          </el-col>
        </el-row>

        <el-row :gutter="16">
          <el-col :span="8">
            <el-form-item label="端口" prop="port">
              <el-input-number v-model="form.port" :min="10000" :max="65535" style="width: 100%" />
            </el-form-item>
          </el-col>
          <el-col :span="16">
            <el-form-item label="过期时间">
              <el-date-picker v-model="form.expired_at" type="datetime" placeholder="选择过期时间" format="YYYY-MM-DD HH:mm:ss" value-format="YYYY-MM-DD HH:mm:ss" style="width: 100%" />
            </el-form-item>
          </el-col>
        </el-row>

        <el-divider content-position="left">AWS 配置</el-divider>

        <el-row :gutter="16">
          <el-col :span="12">
            <el-form-item label="AWS 账号" prop="cloud_account_id">
              <el-select v-model="form.cloud_account_id" placeholder="选择 AWS 账号" filterable style="width: 100%">
                <el-option v-for="acc in awsAccounts" :key="acc.id" :label="`${acc.name} (${acc.access_key_id})`" :value="acc.id" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="区域" prop="region_id">
              <el-select v-model="form.region_id" placeholder="选择区域" filterable style="width: 100%">
                <el-option v-for="r in awsRegions" :key="r.id" :label="`${r.name} (${r.id})`" :value="r.id" />
              </el-select>
            </el-form-item>
          </el-col>
        </el-row>

        <el-row :gutter="16">
          <el-col :span="12">
            <el-form-item label="SSH Key Name">
              <el-input v-model="form.key_name" placeholder="tsdd-deploy-key" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="Subnet ID">
              <el-input v-model="form.subnet_id" placeholder="留空使用默认子网" />
            </el-form-item>
          </el-col>
        </el-row>

        <el-divider content-position="left">
          <span class="node-header">DB 节点</span>
          <el-tag type="info" size="small" class="ml-2">MySQL + Redis</el-tag>
        </el-divider>
        <el-row :gutter="16">
          <el-col :span="8">
            <el-form-item label="实例类型"><el-input v-model="form.db_instance_type" /></el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="磁盘(GB)"><el-input-number v-model="form.db_volume_size_gib" :min="20" :max="2000" style="width: 100%" /></el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="AMI ID"><el-input v-model="form.db_ami_id" placeholder="留空自动查找" /></el-form-item>
          </el-col>
        </el-row>

        <el-divider content-position="left">
          <span class="node-header">MinIO 节点</span>
          <el-tag type="info" size="small" class="ml-2">文件存储</el-tag>
        </el-divider>
        <el-row :gutter="16">
          <el-col :span="8">
            <el-form-item label="实例类型"><el-input v-model="form.minio_instance_type" /></el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="磁盘(GB)"><el-input-number v-model="form.minio_volume_size_gib" :min="20" :max="4000" style="width: 100%" /></el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="AMI ID"><el-input v-model="form.minio_ami_id" placeholder="留空自动查找" /></el-form-item>
          </el-col>
        </el-row>

        <el-divider content-position="left">
          <span class="node-header">App 节点</span>
          <el-tag type="success" size="small" class="ml-2">tsdd-server + WuKongIM + Web</el-tag>
        </el-divider>
        <el-row :gutter="16">
          <el-col :span="8">
            <el-form-item label="实例类型"><el-input v-model="form.app_instance_type" /></el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="磁盘(GB)"><el-input-number v-model="form.app_volume_size_gib" :min="20" :max="1000" style="width: 100%" /></el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="AMI ID"><el-input v-model="form.app_ami_id" placeholder="留空自动查找" /></el-form-item>
          </el-col>
        </el-row>
      </el-form>
    </el-card>

    <!-- 部署进度弹窗 -->
    <el-dialog v-model="showSteps" title="集群部署进度" width="650px" :close-on-click-modal="false" :close-on-press-escape="false" @close="finish">
      <div v-for="s in steps" :key="s.step" class="step-row">
        <div class="step-indicator" :style="{ backgroundColor: statusColor(s.status) }" />
        <div class="step-content">
          <div class="step-title">
            {{ s.step }}. {{ s.title }}
            <span style="margin-left: 6px">
              <el-tag v-if="s.status === 'success'" type="success" size="small">完成</el-tag>
              <el-tag v-else-if="s.status === 'failed'" type="danger" size="small">失败</el-tag>
              <el-tag v-else-if="s.status === 'running'" type="primary" size="small">进行中...</el-tag>
              <el-tag v-else-if="s.status === 'skipped'" type="warning" size="small">跳过</el-tag>
            </span>
          </div>
          <div v-if="s.message" class="step-message">{{ s.message }}</div>
        </div>
      </div>

      <div v-if="completed" style="margin-top: 16px">
        <el-result v-if="success" icon="success" title="集群部署完成" sub-title="所有节点已成功部署">
          <template #extra>
            <el-button type="primary" @click="finish">前往集群管理</el-button>
          </template>
        </el-result>
        <el-result v-else icon="error" :title="errorMsg || '部分步骤失败'">
          <template #extra>
            <el-button type="warning" @click="resumeDeploy" :disabled="!merchantId">恢复部署</el-button>
            <el-button @click="finish">关闭</el-button>
          </template>
        </el-result>
      </div>

      <template #footer>
        <el-button @click="finish" :disabled="deploying">{{ deploying ? "部署中..." : "关闭" }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style lang="scss" scoped>
.app-container { padding: 20px; }

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  .title { font-size: 18px; font-weight: 600; }
  .actions { display: flex; gap: 8px; align-items: center; }
}

.wizard-form { max-width: 1000px; margin: 16px 0; }
.node-header { font-weight: 600; }

.step-row {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  padding: 10px 0;
  border-bottom: 1px solid #f0f0f0;
  &:last-child { border-bottom: none; }
}

.step-indicator {
  width: 12px;
  height: 12px;
  border-radius: 50%;
  margin-top: 4px;
  flex-shrink: 0;
}

.step-content { flex: 1; }
.step-title { font-weight: 500; font-size: 14px; }
.step-message { font-size: 12px; color: #909399; margin-top: 4px; word-break: break-all; }
</style>
