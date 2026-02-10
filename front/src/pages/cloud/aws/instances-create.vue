<script lang="ts" setup>
import type { AwsCreateEc2InstanceReq, AwsSecurityGroupOption, AwsSubnetItem } from "@@/apis/aws/type"
import { createEc2Instance, listSecurityGroupOptions, listSubnets } from "@@/apis/aws"
import { getCloudAccountOptions } from "@@/apis/cloud_account"
import { getAwsRegions } from "@@/constants/aws-regions"

defineOptions({ name: "AwsInstancesCreate" })

const router = useRouter()

const loading = ref(false)
const regions = getAwsRegions()
const cloudAccounts = ref<{ value: number, label: string }[]>([])
const subnetOptions = ref<AwsSubnetItem[]>([])
const securityGroupOptions = ref<AwsSecurityGroupOption[]>([])

const form = reactive<AwsCreateEc2InstanceReq>({
  cloud_account_id: undefined,
  merchant_id: undefined,
  region_id: "",
  image_id: "",
  instance_type: "",
  subnet_id: "",
  security_group_ids: [],
  key_name: "",
  volume_size_gib: 500,
  instance_name: ""
})

onMounted(async () => {
  const { data } = await getCloudAccountOptions("aws")
  cloudAccounts.value = data || []
})

async function onSubmit() {
  if (!form.region_id) {
    ElMessage.warning("请选择区域")
    return
  }
  if (!form.cloud_account_id) {
    return ElMessage.warning("请选择系统云账号")
  }
  form.merchant_id = undefined
  loading.value = true
  try {
    const res = await createEc2Instance(form)
    ElMessage.success(`创建成功: ${res.data.instance_id}`)
    router.push({ name: "AwsInstances" })
  } finally {
    loading.value = false
  }
}

// 动态选项加载
async function reloadOptions() {
  subnetOptions.value = []
  securityGroupOptions.value = []
  if (!form.region_id || !form.cloud_account_id) return
  const base: any = {
    region_id: form.region_id,
    cloud_account_id: form.cloud_account_id
  }
  const [subnets, sgs] = await Promise.all([
    listSubnets(base),
    listSecurityGroupOptions(base)
  ])
  subnetOptions.value = subnets.data.list
  securityGroupOptions.value = sgs.data.list
}

watch([
  () => form.region_id,
  () => form.cloud_account_id
], () => {
  reloadOptions()
})

// 取消镜像与实例类型的远程搜索，改为手动输入
</script>

<template>
  <div class="app-container">
    <el-card>
      <el-form label-width="120px" :model="form">
        <el-form-item label="云账号" required>
          <el-select v-model="form.cloud_account_id" placeholder="选择云账号" filterable style="width: 320px">
            <el-option v-for="opt in cloudAccounts" :key="opt.value" :label="opt.label" :value="opt.value" />
          </el-select>
        </el-form-item>

        <el-form-item label="区域" required>
          <el-select v-model="form.region_id" placeholder="选择Region" filterable style="width: 320px">
            <el-option v-for="r in regions" :key="r.id" :label="`${r.name} (${r.id})`" :value="r.id" />
          </el-select>
        </el-form-item>

        <el-form-item label="镜像">
          <el-input v-model="form.image_id" placeholder="留空使用默认 Ubuntu 22.04" style="width: 480px" />
        </el-form-item>

        <el-form-item label="实例类型" required>
          <el-input v-model="form.instance_type" placeholder="输入实例类型（如 t3.micro）" style="width: 320px" />
        </el-form-item>

        <!-- <el-form-item label="子网">
          <el-select v-model="form.subnet_id" filterable placeholder="可选" style="width: 480px">
            <el-option v-for="s in subnetOptions" :key="s.subnet_id" :label="`${s.name || s.subnet_id} | ${s.cidr_block} | ${s.availability_zone}`" :value="s.subnet_id" />
          </el-select>
        </el-form-item> -->

        <!-- <el-form-item label="安全组">
          <el-select v-model="form.security_group_ids" multiple filterable placeholder="可选" style="width: 480px">
            <el-option v-for="g in securityGroupOptions" :key="g.group_id" :label="`${g.group_name} (${g.group_id})`" :value="g.group_id" />
          </el-select>
        </el-form-item> -->

        <!-- <el-form-item label="Key Pair">
          <el-input v-model="form.key_name" placeholder="可选" />
        </el-form-item> -->

        <el-form-item label="系统盘大小(GB)">
          <el-input-number v-model="form.volume_size_gib" :min="8" :max="10240" />
        </el-form-item>

        <el-form-item label="实例名称">
          <el-input v-model="form.instance_name" placeholder="可选" />
        </el-form-item>

        <el-form-item>
          <el-button type="primary" :loading="loading" @click="onSubmit">创建</el-button>
        </el-form-item>
      </el-form>
    </el-card>
  </div>
</template>

<style lang="scss" scoped>
.app-container {
  padding: 20px;
}
</style>
