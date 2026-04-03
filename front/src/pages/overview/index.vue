<script lang="ts" setup>
defineOptions({ name: "Overview" })

import { controlHealthCheckApi, controlServerStatsApi, controlGostOneClickDeployApi } from "@@/apis/control"
import type { ServiceStatus, ServerStatsResp } from "@@/apis/control/type"
import { ElMessage, ElMessageBox } from "element-plus"

// 服务状态
const services = ref<Record<string, ServiceStatus>>({})
const serverStats = ref<ServerStatsResp | null>(null)
const loading = ref(false)
const deployLogs = ref<string[]>([])
const deploying = ref(false)

const statusColor = (status: string) => {
  if (status === "running") return "#67c23a"
  if (status === "stopped") return "#f56c6c"
  return "#909399"
}

const statusText = (status: string) => {
  if (status === "running") return "运行中"
  if (status === "stopped") return "已停止"
  return "未知"
}

// 加载数据
const loadData = async () => {
  loading.value = true
  try {
    const [healthRes, statsRes] = await Promise.all([
      controlHealthCheckApi(),
      controlServerStatsApi()
    ])
    services.value = (healthRes as any).data || {}
    serverStats.value = (statsRes as any).data || null
  } catch (e: any) {
    console.error("加载失败", e)
  } finally {
    loading.value = false
  }
}

// GOST 一键部署
const handleGostDeploy = async () => {
  try {
    await ElMessageBox.confirm("将在本机安装并配置 GOST 服务，是否继续？", "GOST 一键部署", {
      confirmButtonText: "开始部署",
      cancelButtonText: "取消",
      type: "info"
    })
  } catch {
    return
  }

  deploying.value = true
  deployLogs.value = []

  try {
    const stream = await (controlGostOneClickDeployApi as any)()
    const reader = (stream as any).getReader()
    const decoder = new TextDecoder()

    while (true) {
      const { done, value } = await reader.read()
      if (done) break
      const text = decoder.decode(value)
      // 解析 SSE 格式
      for (const line of text.split("\n")) {
        if (line.startsWith("data:")) {
          try {
            const data = JSON.parse(line.replace(/^data:\s*/, ""))
            if (data.message) {
              deployLogs.value.push(data.message)
            }
          } catch { /* ignore */ }
        }
        if (line.startsWith("event: end")) {
          deploying.value = false
          ElMessage.success("GOST 部署完成")
          loadData()
        }
      }
    }
  } catch (e: any) {
    ElMessage.error("部署失败: " + (e.message || e))
  } finally {
    deploying.value = false
  }
}

onMounted(loadData)
</script>

<template>
  <div class="app-container">
    <el-row :gutter="20">
      <!-- 服务器资源 -->
      <el-col :span="24">
        <el-card shadow="never" style="margin-bottom: 20px">
          <template #header>
            <div class="card-header">
              <span>系统概览 (单机模式)</span>
              <el-button type="primary" :loading="loading" @click="loadData" size="small">
                刷新
              </el-button>
            </div>
          </template>
          <el-row :gutter="20" v-if="serverStats">
            <el-col :span="6">
              <el-statistic title="CPU 使用率" :value="(serverStats.cpu_usage || 'N/A') as any" />
            </el-col>
            <el-col :span="6">
              <el-statistic title="内存" :value="(`${serverStats.memory_usage || '?'} / ${serverStats.memory_total || '?'}`) as any" />
            </el-col>
            <el-col :span="6">
              <el-statistic title="磁盘" :value="(`${serverStats.disk_usage || '?'} / ${serverStats.disk_total || '?'}`) as any" />
            </el-col>
            <el-col :span="6">
              <el-statistic title="负载" :value="(serverStats.load_avg || 'N/A') as any" />
            </el-col>
          </el-row>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="20">
      <!-- 服务状态 -->
      <el-col :span="12">
        <el-card shadow="never" style="margin-bottom: 20px">
          <template #header>
            <span>服务状态</span>
          </template>
          <el-table :data="Object.values(services)" v-loading="loading" stripe>
            <el-table-column prop="service_name" label="服务" width="120" />
            <el-table-column label="状态" width="100">
              <template #default="{ row }">
                <el-tag :color="statusColor(row.status)" effect="dark" size="small" style="border: none; color: white">
                  {{ statusText(row.status) }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="cpu" label="CPU" width="80" />
            <el-table-column prop="memory" label="内存" width="100" />
            <el-table-column prop="uptime" label="运行时间" />
          </el-table>
        </el-card>
      </el-col>

      <!-- 快捷操作 -->
      <el-col :span="12">
        <el-card shadow="never" style="margin-bottom: 20px">
          <template #header>
            <span>快捷操作</span>
          </template>
          <div style="display: flex; flex-direction: column; gap: 12px">
            <el-button type="primary" @click="handleGostDeploy" :loading="deploying" size="large" style="width: 100%">
              GOST 一键部署
            </el-button>
            <el-alert type="info" :closable="false">
              V2 架构：自动安装 GOST + nginx 路径分发<br/>
              统一入口 :10443 (/ws, /api/, /s3/) + TCP :10010
            </el-alert>
          </div>
        </el-card>

        <!-- 部署日志 -->
        <el-card v-if="deployLogs.length > 0" shadow="never">
          <template #header>
            <span>部署日志</span>
          </template>
          <div style="max-height: 400px; overflow-y: auto; font-family: monospace; font-size: 12px; line-height: 1.6; background: #1e1e1e; color: #d4d4d4; padding: 12px; border-radius: 4px">
            <div v-for="(log, i) in deployLogs" :key="i">{{ log }}</div>
          </div>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<style scoped>
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
</style>
