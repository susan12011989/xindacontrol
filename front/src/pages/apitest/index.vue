<script lang="ts" setup>
import type { APICategory, APIEndpoint, RunAPITestResp, TestCaseResp } from "@@/apis/apitest/type"
import { createTestCase, getAPICatalog, queryTestCases, runAPITest } from "@@/apis/apitest"
import { getMerchantList } from "@@/apis/merchant"
import { Delete, DocumentAdd, Plus, Refresh, Search, VideoPlay } from "@element-plus/icons-vue"

defineOptions({
  name: "APITest"
})

// 商户选择
const merchantList = ref<any[]>([])
const currentMerchantId = ref(0)

// API目录
const apiCategories = ref<APICategory[]>([])
const selectedModule = ref("")
const searchKeyword = ref("")

// 测试配置
const testForm = reactive({
  method: "GET",
  path: "",
  headers: {} as Record<string, string>,
  query_params: {} as Record<string, string>,
  body: ""
})

// 响应数据
const responseData = ref<RunAPITestResp | null>(null)
const loading = ref(false)

// 测试用例
const testCases = ref<TestCaseResp[]>([])
const testCasesLoading = ref(false)
const showSaveDialog = ref(false)
const saveCaseForm = reactive({
  name: "",
  expected_status: 200,
  expected_contains: ""
})

// 自定义Headers
const customHeaders = ref<{ key: string; value: string }[]>([])
const customQueryParams = ref<{ key: string; value: string }[]>([])

// 加载商户列表
async function loadMerchants() {
  try {
    const res = await getMerchantList({ page: 1, size: 1000 })
    merchantList.value = res.data.list || []
    if (merchantList.value.length > 0 && !currentMerchantId.value) {
      currentMerchantId.value = merchantList.value[0].id
      loadTestCases()
    }
  } catch (error) {
    console.error("加载商户列表失败", error)
  }
}

// 加载API目录
async function loadAPICatalog() {
  try {
    const res = await getAPICatalog()
    apiCategories.value = res.data.categories || []
  } catch (error) {
    console.error("加载API目录失败", error)
  }
}

// 加载测试用例
async function loadTestCases() {
  if (!currentMerchantId.value) return

  testCasesLoading.value = true
  try {
    const res = await queryTestCases({
      page: 1,
      size: 100,
      merchant_id: currentMerchantId.value
    })
    testCases.value = res.data.list || []
  } catch (error) {
    console.error("加载测试用例失败", error)
  } finally {
    testCasesLoading.value = false
  }
}

// 选择API端点
function selectEndpoint(endpoint: APIEndpoint) {
  testForm.method = endpoint.method
  testForm.path = endpoint.path
  testForm.headers = {}
  testForm.query_params = {}
  testForm.body = ""
  customHeaders.value = []
  customQueryParams.value = []
  responseData.value = null
}

// 过滤后的API列表
const filteredCategories = computed(() => {
  let categories = apiCategories.value

  if (selectedModule.value) {
    categories = categories.filter((c: APICategory) => c.module === selectedModule.value)
  }

  if (searchKeyword.value) {
    const keyword = searchKeyword.value.toLowerCase()
    categories = categories
      .map((c: APICategory) => ({
        ...c,
        endpoints: c.endpoints.filter(
          (e: APIEndpoint) => e.path.toLowerCase().includes(keyword) || e.description.toLowerCase().includes(keyword)
        )
      }))
      .filter((c: APICategory) => c.endpoints.length > 0)
  }

  return categories
})

// 添加Header
function addHeader() {
  customHeaders.value.push({ key: "", value: "" })
}

// 移除Header
function removeHeader(index: number) {
  customHeaders.value.splice(index, 1)
}

// 添加Query参数
function addQueryParam() {
  customQueryParams.value.push({ key: "", value: "" })
}

// 移除Query参数
function removeQueryParam(index: number) {
  customQueryParams.value.splice(index, 1)
}

// 运行测试
async function runTest() {
  if (!currentMerchantId.value) {
    ElMessage.warning("请选择商户")
    return
  }

  if (!testForm.path) {
    ElMessage.warning("请选择或输入API路径")
    return
  }

  loading.value = true
  responseData.value = null

  try {
    // 构建Headers
    const headers: Record<string, string> = {}
    customHeaders.value.forEach((h) => {
      if (h.key) headers[h.key] = h.value
    })

    // 构建Query参数
    const queryParams: Record<string, string> = {}
    customQueryParams.value.forEach((p) => {
      if (p.key) queryParams[p.key] = p.value
    })

    const res = await runAPITest({
      merchant_id: currentMerchantId.value,
      method: testForm.method,
      path: testForm.path,
      headers,
      query_params: queryParams,
      body: testForm.body
    })

    responseData.value = res.data
  } catch (error: any) {
    ElMessage.error(error.message || "请求失败")
  } finally {
    loading.value = false
  }
}

// 保存为测试用例
function openSaveDialog() {
  if (!testForm.path) {
    ElMessage.warning("请先选择API")
    return
  }
  saveCaseForm.name = `${testForm.method} ${testForm.path}`
  saveCaseForm.expected_status = 200
  saveCaseForm.expected_contains = ""
  showSaveDialog.value = true
}

async function saveTestCase() {
  if (!saveCaseForm.name) {
    ElMessage.warning("请输入用例名称")
    return
  }

  try {
    const headers: Record<string, string> = {}
    customHeaders.value.forEach((h) => {
      if (h.key) headers[h.key] = h.value
    })

    const queryParams: Record<string, string> = {}
    customQueryParams.value.forEach((p) => {
      if (p.key) queryParams[p.key] = p.value
    })

    const selectedCategory = apiCategories.value.find((c: APICategory) =>
      c.endpoints.some((e: APIEndpoint) => e.path === testForm.path)
    )

    await createTestCase(currentMerchantId.value, {
      name: saveCaseForm.name,
      module: selectedCategory?.module || "unknown",
      method: testForm.method,
      path: testForm.path,
      headers,
      query_params: queryParams,
      body: testForm.body,
      expected_status: saveCaseForm.expected_status,
      expected_contains: saveCaseForm.expected_contains
    })

    ElMessage.success("保存成功")
    showSaveDialog.value = false
    loadTestCases()
  } catch (error: any) {
    ElMessage.error(error.message || "保存失败")
  }
}

// 加载测试用例到表单
function loadTestCaseToForm(testCase: TestCaseResp) {
  testForm.method = testCase.method
  testForm.path = testCase.path
  testForm.body = testCase.body || ""

  customHeaders.value = []
  if (testCase.headers) {
    Object.entries(testCase.headers).forEach(([key, value]) => {
      customHeaders.value.push({ key, value: value as string })
    })
  }

  customQueryParams.value = []
  if (testCase.query_params) {
    Object.entries(testCase.query_params).forEach(([key, value]) => {
      customQueryParams.value.push({ key, value: value as string })
    })
  }

  responseData.value = null
}

// 格式化JSON
function formatJson(str: string | undefined): string {
  if (!str) return ""
  try {
    return JSON.stringify(JSON.parse(str), null, 2)
  } catch {
    return str
  }
}

// 状态徽章类型
function getStatusType(status: number): "success" | "danger" | "info" {
  if (status === 1) return "success"
  if (status === 2) return "danger"
  return "info"
}

// 状态文本
function getStatusText(status: number): string {
  if (status === 1) return "成功"
  if (status === 2) return "失败"
  return "未运行"
}

// 方法颜色
function getMethodColor(method: string): string {
  const colors: Record<string, string> = {
    GET: "#67c23a",
    POST: "#409eff",
    PUT: "#e6a23c",
    DELETE: "#f56c6c"
  }
  return colors[method] || "#909399"
}

onMounted(() => {
  loadMerchants()
  loadAPICatalog()
})

watch(currentMerchantId, () => {
  loadTestCases()
})
</script>

<template>
  <div class="api-test-container">
    <!-- 顶部工具栏 -->
    <el-card class="toolbar-card" shadow="never">
      <div class="toolbar">
        <div class="toolbar-left">
          <el-select v-model="currentMerchantId" placeholder="选择商户" style="width: 200px" filterable>
            <el-option
              v-for="item in merchantList"
              :key="item.id"
              :label="`${item.name} (${item.server_ip}:${item.port})`"
              :value="item.id"
            />
          </el-select>

          <el-select v-model="selectedModule" placeholder="选择模块" style="width: 150px" clearable>
            <el-option v-for="cat in apiCategories" :key="cat.module" :label="cat.name" :value="cat.module" />
          </el-select>

          <el-input v-model="searchKeyword" placeholder="搜索API..." style="width: 200px" clearable>
            <template #prefix>
              <el-icon><Search /></el-icon>
            </template>
          </el-input>
        </div>
      </div>
    </el-card>

    <div class="main-content">
      <!-- 左侧API列表 -->
      <el-card class="api-list-card" shadow="never">
        <template #header>
          <div class="card-header">
            <span>API目录</span>
            <el-tag type="info" size="small"
              >{{ filteredCategories.reduce((acc: number, c: APICategory) => acc + c.endpoints.length, 0) }} 个</el-tag
            >
          </div>
        </template>

        <el-scrollbar height="calc(100vh - 280px)">
          <el-collapse accordion>
            <el-collapse-item
              v-for="category in filteredCategories"
              :key="category.module"
              :title="category.name"
              :name="category.module"
            >
              <div
                v-for="endpoint in category.endpoints"
                :key="endpoint.path"
                class="endpoint-item"
                @click="selectEndpoint(endpoint)"
              >
                <el-tag
                  :style="{ backgroundColor: getMethodColor(endpoint.method), color: '#fff', border: 'none' }"
                  size="small"
                >
                  {{ endpoint.method }}
                </el-tag>
                <span class="endpoint-path">{{ endpoint.path }}</span>
                <span class="endpoint-desc">{{ endpoint.description }}</span>
              </div>
            </el-collapse-item>
          </el-collapse>
        </el-scrollbar>
      </el-card>

      <!-- 中间测试区域 -->
      <el-card class="test-area-card" shadow="never">
        <template #header>
          <div class="card-header">
            <span>API测试</span>
            <el-button type="primary" size="small" @click="openSaveDialog">
              <el-icon><DocumentAdd /></el-icon>
              保存为用例
            </el-button>
          </div>
        </template>

        <!-- 请求配置 -->
        <div class="request-config">
          <div class="request-line">
            <el-select v-model="testForm.method" style="width: 100px">
              <el-option label="GET" value="GET" />
              <el-option label="POST" value="POST" />
              <el-option label="PUT" value="PUT" />
              <el-option label="DELETE" value="DELETE" />
            </el-select>
            <el-input v-model="testForm.path" placeholder="输入API路径，如 /v1/health" style="flex: 1" />
            <el-button type="primary" :loading="loading" @click="runTest">
              <el-icon><VideoPlay /></el-icon>
              发送
            </el-button>
          </div>

          <el-tabs type="border-card" class="config-tabs">
            <!-- Headers -->
            <el-tab-pane label="Headers">
              <div v-for="(header, index) in customHeaders" :key="index" class="param-row">
                <el-input v-model="header.key" placeholder="Key" style="width: 40%" />
                <el-input v-model="header.value" placeholder="Value" style="width: 45%" />
                <el-button type="danger" :icon="Delete" circle size="small" @click="removeHeader(index)" />
              </div>
              <el-button type="primary" text @click="addHeader">
                <el-icon><Plus /></el-icon> 添加Header
              </el-button>
            </el-tab-pane>

            <!-- Query参数 -->
            <el-tab-pane label="Query参数">
              <div v-for="(param, index) in customQueryParams" :key="index" class="param-row">
                <el-input v-model="param.key" placeholder="Key" style="width: 40%" />
                <el-input v-model="param.value" placeholder="Value" style="width: 45%" />
                <el-button type="danger" :icon="Delete" circle size="small" @click="removeQueryParam(index)" />
              </div>
              <el-button type="primary" text @click="addQueryParam">
                <el-icon><Plus /></el-icon> 添加参数
              </el-button>
            </el-tab-pane>

            <!-- Body -->
            <el-tab-pane label="Body">
              <el-input v-model="testForm.body" type="textarea" :rows="6" placeholder="JSON格式请求体" />
            </el-tab-pane>
          </el-tabs>
        </div>

        <!-- 响应结果 -->
        <div v-if="responseData" class="response-area">
          <div class="response-header">
            <span>响应结果</span>
            <div class="response-meta">
              <el-tag :type="responseData.success ? 'success' : 'danger'" size="small">
                {{ responseData.status_code }}
              </el-tag>
              <span class="response-time">{{ responseData.response_time }}ms</span>
            </div>
          </div>

          <el-tabs type="border-card">
            <el-tab-pane label="Body">
              <pre class="response-body">{{ formatJson(responseData.body) }}</pre>
            </el-tab-pane>
            <el-tab-pane label="Headers">
              <div v-for="(value, key) in responseData.headers" :key="key" class="header-item">
                <span class="header-key">{{ key }}:</span>
                <span class="header-value">{{ value }}</span>
              </div>
            </el-tab-pane>
          </el-tabs>
        </div>
      </el-card>

      <!-- 右侧测试用例 -->
      <el-card class="cases-card" shadow="never">
        <template #header>
          <div class="card-header">
            <span>测试用例</span>
            <el-button text :loading="testCasesLoading" @click="loadTestCases">
              <el-icon><Refresh /></el-icon>
            </el-button>
          </div>
        </template>

        <el-scrollbar height="calc(100vh - 280px)">
          <div v-if="testCases.length === 0" class="empty-cases">
            <el-empty description="暂无测试用例" :image-size="60" />
          </div>
          <div v-else>
            <div
              v-for="testCase in testCases"
              :key="testCase.id"
              class="case-item"
              @click="loadTestCaseToForm(testCase)"
            >
              <div class="case-header">
                <el-tag
                  :style="{ backgroundColor: getMethodColor(testCase.method), color: '#fff', border: 'none' }"
                  size="small"
                >
                  {{ testCase.method }}
                </el-tag>
                <el-tag :type="getStatusType(testCase.last_run_status)" size="small">
                  {{ getStatusText(testCase.last_run_status) }}
                </el-tag>
              </div>
              <div class="case-name">{{ testCase.name }}</div>
              <div class="case-path">{{ testCase.path }}</div>
              <div v-if="testCase.last_run_at" class="case-time">最后运行: {{ testCase.last_run_at }}</div>
            </div>
          </div>
        </el-scrollbar>
      </el-card>
    </div>

    <!-- 保存用例对话框 -->
    <el-dialog v-model="showSaveDialog" title="保存为测试用例" width="500">
      <el-form label-width="100px">
        <el-form-item label="用例名称">
          <el-input v-model="saveCaseForm.name" placeholder="输入用例名称" />
        </el-form-item>
        <el-form-item label="期望状态码">
          <el-input-number v-model="saveCaseForm.expected_status" :min="100" :max="599" />
        </el-form-item>
        <el-form-item label="期望包含">
          <el-input v-model="saveCaseForm.expected_contains" placeholder="响应体应包含的内容（可选）" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showSaveDialog = false">取消</el-button>
        <el-button type="primary" @click="saveTestCase">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.api-test-container {
  padding: 16px;
  background: #f5f7fa;
  min-height: calc(100vh - 100px);
}

.toolbar-card {
  margin-bottom: 16px;
}

.toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.toolbar-left {
  display: flex;
  gap: 12px;
}

.main-content {
  display: grid;
  grid-template-columns: 300px 1fr 280px;
  gap: 16px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.api-list-card,
.test-area-card,
.cases-card {
  height: calc(100vh - 200px);
}

.endpoint-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  cursor: pointer;
  border-radius: 4px;
  margin-bottom: 4px;
}

.endpoint-item:hover {
  background: #f5f7fa;
}

.endpoint-path {
  font-family: monospace;
  font-size: 13px;
  color: #303133;
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.endpoint-desc {
  font-size: 12px;
  color: #909399;
}

.request-config {
  margin-bottom: 16px;
}

.request-line {
  display: flex;
  gap: 8px;
  margin-bottom: 16px;
}

.config-tabs {
  margin-top: 12px;
}

.param-row {
  display: flex;
  gap: 8px;
  margin-bottom: 8px;
  align-items: center;
}

.response-area {
  border-top: 1px solid #ebeef5;
  padding-top: 16px;
}

.response-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}

.response-meta {
  display: flex;
  gap: 8px;
  align-items: center;
}

.response-time {
  font-size: 12px;
  color: #909399;
}

.response-body {
  background: #f5f7fa;
  padding: 12px;
  border-radius: 4px;
  font-family: monospace;
  font-size: 13px;
  overflow: auto;
  max-height: 300px;
  margin: 0;
  white-space: pre-wrap;
  word-break: break-all;
}

.header-item {
  padding: 4px 0;
  font-size: 13px;
}

.header-key {
  color: #606266;
  font-weight: 500;
}

.header-value {
  color: #909399;
  margin-left: 8px;
}

.empty-cases {
  padding: 40px 0;
}

.case-item {
  padding: 12px;
  border: 1px solid #ebeef5;
  border-radius: 4px;
  margin-bottom: 8px;
  cursor: pointer;
  transition: all 0.2s;
}

.case-item:hover {
  border-color: #409eff;
  background: #f5f7fa;
}

.case-header {
  display: flex;
  gap: 8px;
  margin-bottom: 8px;
}

.case-name {
  font-size: 14px;
  font-weight: 500;
  color: #303133;
  margin-bottom: 4px;
}

.case-path {
  font-family: monospace;
  font-size: 12px;
  color: #606266;
  margin-bottom: 4px;
}

.case-time {
  font-size: 11px;
  color: #909399;
}
</style>
