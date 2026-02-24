<script lang="ts" setup>
import type { ExecuteEmbedResp, SourceFileItem, SystemIPItem, TargetItem } from "@@/apis/ip_embed/type"
import type { VersionEntry } from "@@/apis/utils/type"
import type { UploadFile } from "element-plus"
import { useUserStore } from "@/pinia/stores/user"
import { createTarget, deleteTarget, executeEmbedAndUpload, getSelectedIPs, getSourceFiles, getSystemIPs, getTargets, toggleTarget, updateTarget } from "@@/apis/ip_embed"
import type { CreateTargetReq, UpdateTargetReq } from "@@/apis/ip_embed/type"
import { getCloudAccountOptions } from "@@/apis/cloud_account"
import { getMerchantList } from "@@/apis/merchant"
import { aliyunListBuckets, awsListBuckets, tencentListBuckets } from "@@/apis/cloud_storage"
import type { CloudBucketItem } from "@@/apis/cloud_storage/type"
import { decryptVersion, embedIPs, embedIPsBatch, embedURLs, embedURLsBatch, enterprise2Port, extractIPs, extractURLs, generateVersion, port2Enterprise } from "@@/apis/utils"
import {
  ArrowRight,
  Connection,
  Delete,
  DocumentCopy,
  Download,
  Lock,
  Plus,
  Refresh,
  Right,
  Upload,
  UploadFilled,
  View
} from "@element-plus/icons-vue"
import { ElMessage, ElMessageBox, ElNotification } from "element-plus"

defineOptions({
  name: "UtilsTools"
})

// 兼容 HTTP 的剪贴板复制函数
function copyToClipboard(text: string): Promise<void> {
  // 优先使用 Clipboard API（HTTPS 环境）
  if (navigator.clipboard && window.isSecureContext) {
    return copyToClipboard(text)
  }
  // 降级方案：使用 textarea + execCommand（HTTP 环境）
  return new Promise((resolve, reject) => {
    const textarea = document.createElement("textarea")
    textarea.value = text
    textarea.style.position = "fixed"
    textarea.style.left = "-9999px"
    document.body.appendChild(textarea)
    textarea.select()
    try {
      document.execCommand("copy")
      resolve()
    } catch (err) {
      reject(err)
    } finally {
      document.body.removeChild(textarea)
    }
  })
}

// 当前激活的Tab
const activeTab = ref("port")

// 仅 admin 可见端口转换 Tab；非 admin 时自动切换到可见 Tab
const userStore = useUserStore()
const isAdmin = computed(() => userStore.roles.includes("admin"))
watchEffect(() => {
  if (!isAdmin.value && activeTab.value === "port") {
    activeTab.value = "ip"
  }
})

// 切换到IP批量上传时自动加载数据
watch(activeTab, (newVal) => {
  if (newVal === "ip-upload" && ipUploadTool.systemIPs.length === 0) {
    ipUploadTool.loadData()
  }
})

// ========== 端口转换工具 ==========
const portTool = reactive({
  loading: false,
  port: undefined as number | undefined,
  enterprise: "",
  enterpriseInput: "",
  portResult: undefined as number | undefined,

  // 端口转企业号
  convertPort: async () => {
    if (!portTool.port) {
      ElMessage.warning("请输入端口号")
      return
    }
    if (portTool.port < 1 || portTool.port > 65535) {
      ElMessage.warning("端口号范围：1-65535")
      return
    }
    portTool.loading = true
    try {
      const res = await port2Enterprise({ port: portTool.port })
      portTool.enterprise = res.data.enterprise
      ElMessage.success("转换成功")
    } finally {
      portTool.loading = false
    }
  },

  // 企业号转端口
  convertEnterprise: async () => {
    if (!portTool.enterpriseInput) {
      ElMessage.warning("请输入企业号")
      return
    }
    if (portTool.enterpriseInput.length !== 6 || !/^\d+$/.test(portTool.enterpriseInput)) {
      ElMessage.warning("企业号必须是6位数字")
      return
    }
    portTool.loading = true
    try {
      const res = await enterprise2Port({ enterprise: portTool.enterpriseInput })
      portTool.portResult = res.data.port
      ElMessage.success("转换成功")
    } catch {
      // 错误已由 axios 拦截器处理并显示
    } finally {
      portTool.loading = false
    }
  },

  reset: () => {
    portTool.port = undefined
    portTool.enterprise = ""
    portTool.enterpriseInput = ""
    portTool.portResult = undefined
  },

  // 复制企业号
  copyEnterprise: () => {
    copyToClipboard(portTool.enterprise).then(() => {
      ElMessage.success("已复制")
    })
  },

  // 复制端口号
  copyPort: () => {
    copyToClipboard(portTool.portResult!.toString()).then(() => {
      ElMessage.success("已复制")
    })
  }
})

// ========== IP工具 ==========
const ipTool = reactive({
  loading: false,
  // 嵌入工具
  embedFile: null as File | null,
  embedIpList: "",
  // 提取工具
  extractFile: null as File | null,
  extractedIPs: [] as string[],
  // 批量嵌入工具
  batchEmbedZip: null as File | null,
  batchEmbedIpList: "",

  // 嵌入IP到文件
  embedIPsToFile: async () => {
    if (!ipTool.embedFile) {
      ElMessage.warning("请选择要嵌入的文件")
      return
    }
    if (!ipTool.embedIpList.trim()) {
      ElMessage.warning("请输入IP列表")
      return
    }

    // 解析IP列表
    const ips = ipTool.embedIpList
      .split("\n")
      .map(ip => ip.trim())
      .filter(ip => ip)

    if (ips.length === 0) {
      ElMessage.warning("请输入至少一个IP地址")
      return
    }

    // 验证IP格式
    const ipRegex = /^(?:(?:25[0-5]|2[0-4]\d|[01]?\d{1,2})\.){3}(?:25[0-5]|2[0-4]\d|[01]?\d{1,2})$|^(?:[\da-f]{1,4}:){7}[\da-f]{1,4}$/i
    for (const ip of ips) {
      if (!ipRegex.test(ip)) {
        ElMessage.warning(`无效的IP地址: ${ip}`)
        return
      }
    }

    ipTool.loading = true
    try {
      // axios 拦截器已经返回了 Blob 对象本身
      const blob = (await embedIPs(ipTool.embedFile, ips)) as unknown as Blob
      // 下载文件
      const url = window.URL.createObjectURL(blob)
      const link = document.createElement("a")
      link.href = url
      link.download = ipTool.embedFile.name
      document.body.appendChild(link)
      link.click()
      document.body.removeChild(link)
      window.URL.revokeObjectURL(url)

      ElMessage.success("IP嵌入成功，文件已下载")
    } catch (error) {
      console.error(error)
    } finally {
      ipTool.loading = false
    }
  },

  // 批量嵌入IP到ZIP文件
  embedIPsToZip: async () => {
    if (!ipTool.batchEmbedZip) {
      ElMessage.warning("请选择要嵌入的zip文件")
      return
    }
    if (!ipTool.batchEmbedIpList.trim()) {
      ElMessage.warning("请输入IP列表")
      return
    }

    // 验证是否为zip文件
    if (!ipTool.batchEmbedZip.name.toLowerCase().endsWith(".zip")) {
      ElMessage.warning("请上传zip格式的文件")
      return
    }

    // 解析IP列表
    const ips = ipTool.batchEmbedIpList
      .split("\n")
      .map(ip => ip.trim())
      .filter(ip => ip)

    if (ips.length === 0) {
      ElMessage.warning("请输入至少一个IP地址")
      return
    }

    // 验证IP格式
    const ipRegex = /^(?:(?:25[0-5]|2[0-4]\d|[01]?\d{1,2})\.){3}(?:25[0-5]|2[0-4]\d|[01]?\d{1,2})$|^(?:[\da-f]{1,4}:){7}[\da-f]{1,4}$/i
    for (const ip of ips) {
      if (!ipRegex.test(ip)) {
        ElMessage.warning(`无效的IP地址: ${ip}`)
        return
      }
    }

    ipTool.loading = true
    try {
      const blob = (await embedIPsBatch(ipTool.batchEmbedZip, ips)) as unknown as Blob
      // 下载文件
      const url = window.URL.createObjectURL(blob)
      const link = document.createElement("a")
      link.href = url
      const originalName = ipTool.batchEmbedZip.name.replace(/\.zip$/i, "")
      link.download = `${originalName}_embedded.zip`
      document.body.appendChild(link)
      link.click()
      document.body.removeChild(link)
      window.URL.revokeObjectURL(url)

      ElMessage.success("批量IP嵌入成功，文件已下载")
    } catch (error) {
      console.error(error)
    } finally {
      ipTool.loading = false
    }
  },

  // 从文件提取IP
  extractIPsFromFile: async () => {
    if (!ipTool.extractFile) {
      ElMessage.warning("请选择要提取的文件")
      return
    }

    ipTool.loading = true
    try {
      const res = await extractIPs(ipTool.extractFile)
      ipTool.extractedIPs = res.data.ips || []
      if (ipTool.extractedIPs.length > 0) {
        ElNotification.success({
          title: "提取成功",
          message: `成功提取 ${ipTool.extractedIPs.length} 个IP地址`,
          duration: 3000
        })
      } else {
        ElMessage.warning("未找到嵌入的IP数据")
      }
    } catch (error) {
      console.error(error)
    } finally {
      ipTool.loading = false
    }
  },

  // 复制IP列表
  copyIPs: () => {
    const text = ipTool.extractedIPs.join("\n")
    copyToClipboard(text).then(() => {
      ElMessage.success("已复制到剪贴板")
    })
  },

  resetEmbed: () => {
    ipTool.embedFile = null
    ipTool.embedIpList = ""
  },

  resetExtract: () => {
    ipTool.extractFile = null
    ipTool.extractedIPs = []
  },

  resetBatchEmbed: () => {
    ipTool.batchEmbedZip = null
    ipTool.batchEmbedIpList = ""
  }
})

// ========== URL工具 ==========
const urlTool = reactive({
  loading: false,
  // 嵌入工具
  embedFile: null as File | null,
  embedUrlList: "",
  // 提取工具
  extractFile: null as File | null,
  extractedURLs: [] as string[],
  // 批量嵌入工具
  batchEmbedZip: null as File | null,
  batchEmbedUrlList: "",

  // 嵌入URL到文件
  embedURLsToFile: async () => {
    if (!urlTool.embedFile) {
      ElMessage.warning("请选择要嵌入的文件")
      return
    }
    if (!urlTool.embedUrlList.trim()) {
      ElMessage.warning("请输入URL列表")
      return
    }

    const urls = urlTool.embedUrlList
      .split("\n")
      .map(u => u.trim())
      .filter(u => u)

    if (urls.length === 0) {
      ElMessage.warning("请输入至少一个URL")
      return
    }

    // 基础校验
    for (const u of urls) {
      try {
        const parsed = new URL(u)
        if (!/^https?:$/.test(parsed.protocol) || !parsed.host) {
          ElMessage.warning(`无效的URL: ${u}`)
          return
        }
      } catch {
        ElMessage.warning(`无效的URL: ${u}`)
        return
      }
    }

    urlTool.loading = true
    try {
      const blob = (await embedURLs(urlTool.embedFile, urls)) as unknown as Blob
      const url = window.URL.createObjectURL(blob)
      const link = document.createElement("a")
      link.href = url
      link.download = urlTool.embedFile.name
      document.body.appendChild(link)
      link.click()
      document.body.removeChild(link)
      window.URL.revokeObjectURL(url)

      ElMessage.success("URL嵌入成功，文件已下载")
    } catch (error) {
      console.error(error)
    } finally {
      urlTool.loading = false
    }
  },

  // 批量嵌入URL到ZIP文件
  embedURLsToZip: async () => {
    if (!urlTool.batchEmbedZip) {
      ElMessage.warning("请选择要嵌入的zip文件")
      return
    }
    if (!urlTool.batchEmbedUrlList.trim()) {
      ElMessage.warning("请输入URL列表")
      return
    }

    // 验证是否为zip文件
    if (!urlTool.batchEmbedZip.name.toLowerCase().endsWith(".zip")) {
      ElMessage.warning("请上传zip格式的文件")
      return
    }

    const urls = urlTool.batchEmbedUrlList
      .split("\n")
      .map(u => u.trim())
      .filter(u => u)

    if (urls.length === 0) {
      ElMessage.warning("请输入至少一个URL")
      return
    }

    // 基础校验
    for (const u of urls) {
      try {
        const parsed = new URL(u)
        if (!/^https?:$/.test(parsed.protocol) || !parsed.host) {
          ElMessage.warning(`无效的URL: ${u}`)
          return
        }
      } catch {
        ElMessage.warning(`无效的URL: ${u}`)
        return
      }
    }

    urlTool.loading = true
    try {
      const blob = (await embedURLsBatch(urlTool.batchEmbedZip, urls)) as unknown as Blob
      const url = window.URL.createObjectURL(blob)
      const link = document.createElement("a")
      link.href = url
      const originalName = urlTool.batchEmbedZip.name.replace(/\.zip$/i, "")
      link.download = `${originalName}_embedded.zip`
      document.body.appendChild(link)
      link.click()
      document.body.removeChild(link)
      window.URL.revokeObjectURL(url)

      ElMessage.success("批量URL嵌入成功，文件已下载")
    } catch (error) {
      console.error(error)
    } finally {
      urlTool.loading = false
    }
  },

  // 从文件提取URL
  extractURLsFromFile: async () => {
    if (!urlTool.extractFile) {
      ElMessage.warning("请选择要提取的文件")
      return
    }

    urlTool.loading = true
    try {
      const res = await extractURLs(urlTool.extractFile)
      urlTool.extractedURLs = res.data.urls || []
      if (urlTool.extractedURLs.length > 0) {
        ElNotification.success({
          title: "提取成功",
          message: `成功提取 ${urlTool.extractedURLs.length} 个URL`,
          duration: 3000
        })
      } else {
        ElMessage.warning("未找到嵌入的URL数据")
      }
    } catch (error) {
      console.error(error)
    } finally {
      urlTool.loading = false
    }
  },

  // 复制URL列表
  copyURLs: () => {
    const text = urlTool.extractedURLs.join("\n")
    copyToClipboard(text).then(() => {
      ElMessage.success("已复制到剪贴板")
    })
  },

  resetEmbed: () => {
    urlTool.embedFile = null
    urlTool.embedUrlList = ""
  },

  resetExtract: () => {
    urlTool.extractFile = null
    urlTool.extractedURLs = []
  },

  resetBatchEmbed: () => {
    urlTool.batchEmbedZip = null
    urlTool.batchEmbedUrlList = ""
  }
})

// ========== 版本管理工具 ==========
const versionTool = reactive({
  loading: false,
  decryptFile: null as File | null,
  package: "mida",
  versions: [
    { channel: "oppo", version: "1.0.0" },
    { channel: "vivo", version: "1.0.0" },
    { channel: "xiaomi", version: "1.0.0" },
    { channel: "rongyao", version: "1.0.0" },
    { channel: "huawei", version: "1.0.0" },
    { channel: "local", version: "1.0.0" }
  ] as VersionEntry[],

  // 添加新版本
  addVersion: () => {
    versionTool.versions.push({ channel: "", version: "" })
  },

  // 删除版本
  removeVersion: (index: number) => {
    if (versionTool.versions.length > 1) {
      versionTool.versions.splice(index, 1)
    } else {
      ElMessage.warning("至少保留一个版本配置")
    }
  },

  // 生成版本配置文件
  generateConfig: async () => {
    // 验证versions
    for (const v of versionTool.versions) {
      if (!v.channel || !v.version) {
        ElMessage.warning("请填写完整的渠道和版本信息")
        return
      }
    }

    versionTool.loading = true
    try {
      const blob = (await generateVersion({
        package: versionTool.package,
        versions: versionTool.versions
      })) as unknown as Blob

      // 下载文件
      const url = window.URL.createObjectURL(blob)
      const link = document.createElement("a")
      link.href = url
      link.download = "content.txt"
      document.body.appendChild(link)
      link.click()
      document.body.removeChild(link)
      window.URL.revokeObjectURL(url)

      ElMessage.success("版本配置文件生成成功，已下载")
    } catch (error) {
      console.error(error)
    } finally {
      versionTool.loading = false
    }
  },

  // 解密版本配置文件
  decryptConfig: async () => {
    if (!versionTool.decryptFile) {
      ElMessage.warning("请选择要解密的文件")
      return
    }

    versionTool.loading = true
    try {
      const blob = (await decryptVersion(versionTool.decryptFile)) as unknown as Blob

      // 下载文件
      const url = window.URL.createObjectURL(blob)
      const link = document.createElement("a")
      link.href = url
      link.download = "content.json"
      document.body.appendChild(link)
      link.click()
      document.body.removeChild(link)
      window.URL.revokeObjectURL(url)

      ElMessage.success("文件解密成功，已下载")
      versionTool.decryptFile = null
    } catch (error) {
      console.error(error)
    } finally {
      versionTool.loading = false
    }
  },

  // 重置
  reset: () => {
    versionTool.package = "mida"
    versionTool.versions = [
      { channel: "oppo", version: "1.0.0" },
      { channel: "vivo", version: "1.0.0" },
      { channel: "xiaomi", version: "1.0.0" },
      { channel: "rongyao", version: "1.0.0" },
      { channel: "huawei", version: "1.0.0" },
      { channel: "local", version: "1.0.0" }
    ]
    versionTool.decryptFile = null
  }
})

// ========== IP批量上传工具 ==========

// IP项（用于扁平化展示）
interface IPItem {
  ip: string
  serverName: string
  serverId: number
  isAuxiliary: boolean
  status: number
  merchantId: number
  merchantName: string
}

// IP 商户筛选
const ipFilterMerchantId = ref<number | undefined>(undefined)

const ipUploadTool = reactive({
  loading: false,
  systemIPs: [] as SystemIPItem[],
  targets: [] as TargetItem[],
  sourceFiles: [] as SourceFileItem[],
  sourceDir: "",
  selectedIPs: [] as string[], // 改为直接选择IP
  selectedTargetIndexes: [] as number[],
  selectedFileNames: [] as string[],
  showResult: false,
  executeResult: null as ExecuteEmbedResp | null,

  // 加载数据
  loadData: async () => {
    ipUploadTool.loading = true
    try {
      const [ipsRes, targetsRes, filesRes, selectedRes] = await Promise.all([
        getSystemIPs(),
        getTargets(),
        getSourceFiles(),
        getSelectedIPs()
      ])

      ipUploadTool.systemIPs = ipsRes.data.ips || []
      ipUploadTool.targets = targetsRes.data.targets || []
      ipUploadTool.sourceFiles = filesRes.data.files || []
      ipUploadTool.sourceDir = filesRes.data.source_dir || ""

      // 直接恢复保存的IP选择
      const savedIPs = selectedRes.data.ips || []
      ipUploadTool.selectedIPs = savedIPs

      // 默认选中所有启用的目标
      ipUploadTool.selectedTargetIndexes = ipUploadTool.targets
        .filter(t => t.enabled)
        .map(t => t.index)
    } catch (e: any) {
      ElMessage.error(e.message || "加载数据失败")
    } finally {
      ipUploadTool.loading = false
    }
  },

  // 全选/取消全选IP
  toggleAllIPs: () => {
    const allIPs = flattenedIPList.value.map(item => item.ip)
    if (ipUploadTool.selectedIPs.length === allIPs.length) {
      ipUploadTool.selectedIPs = []
    } else {
      ipUploadTool.selectedIPs = [...allIPs]
    }
  },

  // 全选/取消全选目标
  toggleAllTargets: () => {
    if (ipUploadTool.selectedTargetIndexes.length === ipUploadTool.targets.length) {
      ipUploadTool.selectedTargetIndexes = []
    } else {
      ipUploadTool.selectedTargetIndexes = ipUploadTool.targets.map(t => t.index)
    }
  },

  // 全选/取消全选文件
  toggleAllFiles: () => {
    if (ipUploadTool.selectedFileNames.length === ipUploadTool.sourceFiles.length) {
      ipUploadTool.selectedFileNames = []
    } else {
      ipUploadTool.selectedFileNames = ipUploadTool.sourceFiles.map(f => f.name)
    }
  },

  // 执行嵌入上传
  handleExecute: async () => {
    if (flattenedIPList.value.length === 0) {
      return ElMessage.warning("没有可用的IP")
    }
    // 过滤掉失效的IP，只使用有效的IP
    const validIPs = new Set(flattenedIPList.value.map(item => item.ip))
    const effectiveIPs = ipUploadTool.selectedIPs.filter(ip => validIPs.has(ip))

    if (effectiveIPs.length === 0) {
      return ElMessage.warning("请至少选择一个有效的IP")
    }
    if (ipUploadTool.selectedTargetIndexes.length === 0) {
      return ElMessage.warning("请至少选择一个上传目标")
    }
    if (ipUploadTool.sourceFiles.length === 0) {
      return ElMessage.warning("源文件目录为空")
    }

    const fileCount = ipUploadTool.selectedFileNames.length || ipUploadTool.sourceFiles.length
    const invalidCount = ipUploadTool.selectedIPs.length - effectiveIPs.length
    let confirmMsg = `确定执行以下操作吗？\n` +
      `- 有效IP: ${effectiveIPs.length} 个\n` +
      `- 目标存储: ${ipUploadTool.selectedTargetIndexes.length} 个\n` +
      `- 文件数量: ${fileCount} 个`

    if (invalidCount > 0) {
      confirmMsg += `\n\n⚠️ 注意：${invalidCount} 个失效IP将被忽略`
    }

    try {
      await ElMessageBox.confirm(confirmMsg, "确认执行", { type: "warning" })
    } catch {
      return
    }

    ipUploadTool.loading = true
    try {
      const res = await executeEmbedAndUpload({
        target_indexes: ipUploadTool.selectedTargetIndexes,
        file_names: ipUploadTool.selectedFileNames.length > 0 ? ipUploadTool.selectedFileNames : undefined,
        ips: effectiveIPs
      })

      ipUploadTool.executeResult = res.data
      ipUploadTool.showResult = true

      const { summary } = res.data
      if (summary.fail_count === 0) {
        ElMessage.success(`执行成功! 共 ${summary.success_count} 个文件上传完成`)
      } else {
        ElMessage.warning(`执行完成，成功 ${summary.success_count}，失败 ${summary.fail_count}`)
      }
    } catch (e: any) {
      ElMessage.error(e.message || "执行失败")
    } finally {
      ipUploadTool.loading = false
    }
  },

  // 复制URL
  copyUrl: (url: string) => {
    copyToClipboard(url)
    ElMessage.success("已复制到剪贴板")
  }
})

// 格式化文件大小
function formatFileSize(bytes: number): string {
  if (bytes === 0) return "0 B"
  const k = 1024
  const sizes = ["B", "KB", "MB", "GB"]
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return `${(bytes / k ** i).toFixed(2)} ${sizes[i]}`
}

// 扁平化IP列表（将服务器的主IP和辅助IP展开为独立项）
const flattenedIPList = computed<IPItem[]>(() => {
  const list: IPItem[] = []
  for (const s of ipUploadTool.systemIPs) {
    // 主IP
    list.push({
      ip: s.ip,
      serverName: s.server_name,
      serverId: s.server_id,
      isAuxiliary: false,
      status: s.status,
      merchantId: s.merchant_id,
      merchantName: s.merchant_name
    })
    // 辅助IP（支持逗号分隔的多个IP）
    if (s.auxiliary_ip) {
      const auxiliaryIPs = s.auxiliary_ip.split(",").map(ip => ip.trim()).filter(ip => ip)
      for (const auxIP of auxiliaryIPs) {
        list.push({
          ip: auxIP,
          serverName: s.server_name,
          serverId: s.server_id,
          isAuxiliary: true,
          status: s.status,
          merchantId: s.merchant_id,
          merchantName: s.merchant_name
        })
      }
    }
  }
  return list
})

// IP 商户选项（从 systemIPs 中提取去重）
const ipMerchantOptions = computed(() => {
  const map = new Map<number, string>()
  for (const s of ipUploadTool.systemIPs) {
    if (s.merchant_id > 0 && s.merchant_name) {
      map.set(s.merchant_id, s.merchant_name)
    }
  }
  return Array.from(map, ([id, name]) => ({ value: id, label: name }))
})

// 按商户筛选后的 IP 列表
const filteredIPList = computed(() => {
  if (!ipFilterMerchantId.value) return flattenedIPList.value
  return flattenedIPList.value.filter(item => item.merchantId === ipFilterMerchantId.value)
})

// 检测失效的IP（选中的IP在当前系统中不存在）
const invalidSelectedIPs = computed<string[]>(() => {
  const validIPs = new Set(flattenedIPList.value.map(item => item.ip))
  return ipUploadTool.selectedIPs.filter(ip => !validIPs.has(ip))
})

// 清除失效的IP
function clearInvalidIPs() {
  const validIPs = new Set(flattenedIPList.value.map(item => item.ip))
  ipUploadTool.selectedIPs = ipUploadTool.selectedIPs.filter(ip => validIPs.has(ip))
  ElMessage.success("已清除失效IP")
}

// 云类型标签颜色
const cloudTypeColors: Record<string, "success" | "warning" | "info" | "primary" | "danger"> = {
  aws: "warning",
  aliyun: "primary",
  tencent: "success"
}

// ========== 商户选项 ==========
interface MerchantOpt {
  value: number
  label: string
}
const merchantOptions: MerchantOpt[] = reactive([])

async function loadMerchantOptions() {
  try {
    const res = await getMerchantList({ page: 1, size: 1000 })
    const list = (res.data.list || []).map((m: any) => ({
      label: `${m.name} (${m.no})`,
      value: m.id
    }))
    merchantOptions.length = 0
    merchantOptions.push(...list)
  } catch (e) {
    console.error("加载商户列表失败", e)
  }
}

// ========== 目标管理 ==========
interface CloudAccountOpt {
  value: number
  label: string
  type: string
}

const targetManagement = reactive({
  showDialog: false,
  dialogMode: "create" as "create" | "edit",
  loading: false,
  bucketsLoading: false,
  cloudAccounts: [] as CloudAccountOpt[],
  buckets: [] as CloudBucketItem[],
  selectedBuckets: [] as string[], // 批量选择的bucket（创建模式用）
  editingTarget: null as TargetItem | null,
  form: {
    name: "",
    merchant_id: 0 as number,
    cloud_account_id: undefined as number | undefined,
    region_id: "",
    bucket: "",
    object_prefix: "",
    enabled: true,
    sort_order: 0,
    group_id: 0
  },

  // 打开创建弹窗
  openCreate: async () => {
    targetManagement.dialogMode = "create"
    targetManagement.editingTarget = null
    targetManagement.buckets = []
    targetManagement.selectedBuckets = []
    targetManagement.cloudAccounts = []
    targetManagement.form = {
      name: "",
      merchant_id: 0,
      cloud_account_id: undefined,
      region_id: "",
      bucket: "",
      object_prefix: "",
      enabled: true,
      sort_order: ipUploadTool.targets.length,
      group_id: 0
    }
    await loadMerchantOptions()
    targetManagement.showDialog = true
  },

  // 打开编辑弹窗
  openEdit: async (target: TargetItem) => {
    targetManagement.dialogMode = "edit"
    targetManagement.editingTarget = target
    targetManagement.buckets = []
    targetManagement.selectedBuckets = []
    targetManagement.form = {
      name: target.name,
      merchant_id: target.merchant_id || 0,
      cloud_account_id: target.cloud_account_id,
      region_id: target.region_id,
      bucket: target.bucket,
      object_prefix: target.object_prefix,
      enabled: target.enabled,
      sort_order: target.sort_order,
      group_id: target.group_id || 0
    }
    await loadMerchantOptions()
    // 加载该商户的云账号
    if (target.merchant_id) {
      await targetManagement.loadCloudAccounts(target.merchant_id)
    } else {
      await targetManagement.loadCloudAccounts()
    }
    // 编辑时自动加载该账号的bucket列表
    if (target.cloud_account_id) {
      await targetManagement.loadBuckets(target.cloud_account_id)
    }
    targetManagement.showDialog = true
  },

  // 全选/取消全选bucket
  toggleAllBuckets: () => {
    if (targetManagement.selectedBuckets.length === targetManagement.buckets.length) {
      targetManagement.selectedBuckets = []
    } else {
      targetManagement.selectedBuckets = targetManagement.buckets.map(b => b.name)
    }
  },

  // 加载云账号选项（按商户过滤）
  loadCloudAccounts: async (merchantId?: number) => {
    try {
      const params = merchantId ? { merchant_id: merchantId } : undefined
      const res = await getCloudAccountOptions(params)
      targetManagement.cloudAccounts = res.data || []
    } catch (e: any) {
      ElMessage.error("加载云账号失败")
    }
  },

  // 加载Bucket列表
  loadBuckets: async (cloudAccountId: number) => {
    const account = targetManagement.cloudAccounts.find(a => a.value === cloudAccountId)
    if (!account) return

    targetManagement.bucketsLoading = true
    targetManagement.buckets = []
    try {
      const params = { cloud_account_id: cloudAccountId }
      let res: any
      if (account.type === "aws") {
        res = await awsListBuckets(params)
        // AWS返回的是 { list: string[] }
        targetManagement.buckets = (res.data?.list || []).map((name: string) => ({
          name,
          location: "",
          creation_date: "",
          storage_class: ""
        }))
      } else if (account.type === "tencent") {
        res = await tencentListBuckets(params)
        targetManagement.buckets = res.data?.list || []
      } else {
        res = await aliyunListBuckets(params)
        targetManagement.buckets = res.data?.list || []
      }
      // 自动填充区域（如果bucket只有一个且有location）
      if (targetManagement.buckets.length > 0 && targetManagement.buckets[0].location && !targetManagement.form.region_id) {
        targetManagement.form.region_id = targetManagement.buckets[0].location
      }
    } catch (e: any) {
      ElMessage.error("加载Bucket列表失败: " + (e.message || "未知错误"))
    } finally {
      targetManagement.bucketsLoading = false
    }
  },

  // 保存目标
  saveTarget: async () => {
    const { form, dialogMode, editingTarget, selectedBuckets, buckets, cloudAccounts } = targetManagement
    if (!form.cloud_account_id) {
      return ElMessage.warning("请选择云账号")
    }
    // 从选中的云账号获取 cloud_type
    const selectedAccount = cloudAccounts.find(a => a.value === form.cloud_account_id)
    const cloudType = selectedAccount?.type || ""

    // 创建模式：批量创建
    if (dialogMode === "create") {
      if (selectedBuckets.length === 0) {
        return ElMessage.warning("请至少选择一个Bucket")
      }

      targetManagement.loading = true
      let successCount = 0
      let failCount = 0
      try {
        for (let i = 0; i < selectedBuckets.length; i++) {
          const selected = selectedBuckets[i]
          // 确保 bucketName 是字符串（兼容 el-checkbox 可能返回对象的情况）
          let bucketName: string
          let regionId: string
          if (typeof selected === "string") {
            bucketName = selected
            const bucketInfo = buckets.find(b => b.name === bucketName)
            regionId = bucketInfo?.location || form.region_id
          } else {
            // selected 是对象，直接从中获取 name 和 location
            const selectedObj = selected as any
            bucketName = selectedObj.name || String(selected)
            regionId = selectedObj.location || form.region_id
          }
          const data: CreateTargetReq = {
            name: bucketName, // 使用bucket名称作为目标名称
            cloud_type: cloudType,
            cloud_account_id: form.cloud_account_id,
            region_id: regionId,
            bucket: bucketName,
            object_prefix: form.object_prefix,
            enabled: form.enabled,
            sort_order: ipUploadTool.targets.length + i,
            group_id: form.group_id || undefined
          }
          try {
            await createTarget(data)
            successCount++
          } catch {
            failCount++
          }
        }
        if (failCount === 0) {
          ElMessage.success(`成功添加 ${successCount} 个目标`)
        } else {
          ElMessage.warning(`添加完成：成功 ${successCount} 个，失败 ${failCount} 个`)
        }
        targetManagement.showDialog = false
        await ipUploadTool.loadData()
      } finally {
        targetManagement.loading = false
      }
      return
    }

    // 编辑模式：单个更新
    if (!form.name) {
      return ElMessage.warning("请输入目标名称")
    }
    if (!form.bucket) {
      return ElMessage.warning("请选择Bucket")
    }

    targetManagement.loading = true
    try {
      if (editingTarget) {
        const bucketInfo = buckets.find(b => b.name === form.bucket)
        const data: UpdateTargetReq = {
          name: form.name,
          cloud_type: cloudType,
          cloud_account_id: form.cloud_account_id,
          region_id: bucketInfo?.location || form.region_id,
          bucket: form.bucket,
          object_prefix: form.object_prefix,
          enabled: form.enabled,
          sort_order: form.sort_order,
          group_id: form.group_id
        }
        await updateTarget(editingTarget.id, data)
        ElMessage.success("更新成功")
      }
      targetManagement.showDialog = false
      await ipUploadTool.loadData()
    } catch (e: any) {
      ElMessage.error(e.message || "保存失败")
    } finally {
      targetManagement.loading = false
    }
  },

  // 删除目标
  handleDelete: async (target: TargetItem) => {
    try {
      await ElMessageBox.confirm(`确定要删除目标 "${target.name}" 吗？`, "确认删除", { type: "warning" })
    } catch {
      return
    }
    try {
      await deleteTarget(target.id)
      ElMessage.success("删除成功")
      await ipUploadTool.loadData()
    } catch (e: any) {
      ElMessage.error(e.message || "删除失败")
    }
  },

  // 切换启用状态
  handleToggle: async (target: TargetItem) => {
    try {
      await toggleTarget(target.id)
      ElMessage.success(target.enabled ? "已禁用" : "已启用")
      await ipUploadTool.loadData()
    } catch (e: any) {
      ElMessage.error(e.message || "操作失败")
    }
  }
})

// 商户筛选
const filterMerchantId = ref<number | undefined>(undefined)

// 展开的商户组（两级导航：点击商户展开其 OSS 目标）
const expandedGroupId = ref<number | null>(null)

function toggleExpandGroup(groupId: number) {
  expandedGroupId.value = expandedGroupId.value === groupId ? null : groupId
}

// 获取某组已选中的目标数量
function getGroupSelectedCount(group: TargetGroup): number {
  return group.allTargets.filter(t => ipUploadTool.selectedTargetIndexes.includes(t.index)).length
}

// 从已有目标中提取商户列表（去重）
const targetMerchantOptions = computed(() => {
  const map = new Map<number, string>()
  for (const t of ipUploadTool.targets) {
    const mid = t.merchant_id || 0
    if (!map.has(mid)) {
      map.set(mid, mid > 0 ? (t.merchant_name || `商户#${mid}`) : "系统")
    }
  }
  return Array.from(map.entries())
    .sort((a, b) => {
      if (a[0] === 0) return 1
      if (b[0] === 0) return -1
      return a[0] - b[0]
    })
    .map(([id, name]) => ({ value: id, label: name }))
})

// 按商户→云账号两级分组的目标列表
interface TargetSubGroup {
  accountId: number
  accountName: string
  cloudType: string
  targets: TargetItem[]
}
interface TargetGroup {
  groupId: number
  groupName: string
  subGroups: TargetSubGroup[]
  allTargets: TargetItem[] // 便于全选操作
}
const groupedTargets = computed<TargetGroup[]>(() => {
  const map = new Map<number, TargetGroup>()

  // 根据筛选条件过滤目标
  const filteredTargets = filterMerchantId.value !== undefined
    ? ipUploadTool.targets.filter(t => (t.merchant_id || 0) === filterMerchantId.value)
    : ipUploadTool.targets

  for (const t of filteredTargets) {
    const mid = t.merchant_id || 0
    if (!map.has(mid)) {
      map.set(mid, {
        groupId: mid,
        groupName: mid > 0 ? (t.merchant_name || `商户#${mid}`) : "系统",
        subGroups: [],
        allTargets: []
      })
    }
    const group = map.get(mid)!
    group.allTargets.push(t)

    // 按云账号二级分组
    const accId = t.cloud_account_id || 0
    let sub = group.subGroups.find(s => s.accountId === accId)
    if (!sub) {
      sub = {
        accountId: accId,
        accountName: t.account_name || `账号#${accId}`,
        cloudType: t.cloud_type,
        targets: []
      }
      group.subGroups.push(sub)
    }
    sub.targets.push(t)
  }
  // 按商户ID排序，系统(0)放最后
  return Array.from(map.values()).sort((a, b) => {
    if (a.groupId === 0) return 1
    if (b.groupId === 0) return -1
    return a.groupId - b.groupId
  })
})

// 按商户组全选/取消
function toggleGroupTargets(group: TargetGroup) {
  const groupIndexes = group.allTargets.map(t => t.index)
  const allSelected = groupIndexes.every(idx => ipUploadTool.selectedTargetIndexes.includes(idx))
  if (allSelected) {
    ipUploadTool.selectedTargetIndexes = ipUploadTool.selectedTargetIndexes.filter(idx => !groupIndexes.includes(idx))
  } else {
    const set = new Set([...ipUploadTool.selectedTargetIndexes, ...groupIndexes])
    ipUploadTool.selectedTargetIndexes = Array.from(set)
  }
}

// 检查商户组是否全选
function isGroupAllSelected(group: TargetGroup): boolean {
  return group.allTargets.every(t => ipUploadTool.selectedTargetIndexes.includes(t.index))
}

// 检查商户组是否部分选中
function isGroupIndeterminate(group: TargetGroup): boolean {
  const some = group.allTargets.some(t => ipUploadTool.selectedTargetIndexes.includes(t.index))
  const all = group.allTargets.every(t => ipUploadTool.selectedTargetIndexes.includes(t.index))
  return some && !all
}

// 按云账号子组全选/取消
function toggleSubGroupTargets(sub: TargetSubGroup) {
  const subIndexes = sub.targets.map(t => t.index)
  const allSelected = subIndexes.every(idx => ipUploadTool.selectedTargetIndexes.includes(idx))
  if (allSelected) {
    ipUploadTool.selectedTargetIndexes = ipUploadTool.selectedTargetIndexes.filter(idx => !subIndexes.includes(idx))
  } else {
    const set = new Set([...ipUploadTool.selectedTargetIndexes, ...subIndexes])
    ipUploadTool.selectedTargetIndexes = Array.from(set)
  }
}

// 检查子组是否全选
function isSubGroupAllSelected(sub: TargetSubGroup): boolean {
  return sub.targets.every(t => ipUploadTool.selectedTargetIndexes.includes(t.index))
}

// 检查子组是否部分选中
function isSubGroupIndeterminate(sub: TargetSubGroup): boolean {
  const some = sub.targets.some(t => ipUploadTool.selectedTargetIndexes.includes(t.index))
  const all = sub.targets.every(t => ipUploadTool.selectedTargetIndexes.includes(t.index))
  return some && !all
}

// 切换单个目标选中状态
function toggleTargetIndex(index: number) {
  const pos = ipUploadTool.selectedTargetIndexes.indexOf(index)
  if (pos >= 0) {
    ipUploadTool.selectedTargetIndexes.splice(pos, 1)
  } else {
    ipUploadTool.selectedTargetIndexes.push(index)
  }
}

// 商户变化时加载该商户的云账号
watch(() => targetManagement.form.merchant_id, async (newMerchantId) => {
  targetManagement.form.cloud_account_id = undefined
  targetManagement.form.bucket = ""
  targetManagement.form.region_id = ""
  targetManagement.buckets = []
  if (newMerchantId && newMerchantId > 0) {
    await targetManagement.loadCloudAccounts(newMerchantId)
  } else {
    targetManagement.cloudAccounts = []
  }
})

// 云账号变化时加载bucket列表
watch(() => targetManagement.form.cloud_account_id, async (newId) => {
  if (newId) {
    targetManagement.form.bucket = ""
    targetManagement.form.region_id = ""
    await targetManagement.loadBuckets(newId)
  } else {
    targetManagement.buckets = []
  }
})

// 文件上传处理
function handleEmbedFileChange(file: UploadFile) {
  ipTool.embedFile = file.raw as File
}

function handleExtractFileChange(file: UploadFile) {
  ipTool.extractFile = file.raw as File
}

function handleBatchEmbedZipChange(file: UploadFile) {
  ipTool.batchEmbedZip = file.raw as File
}

function handleUrlEmbedFileChange(file: UploadFile) {
  urlTool.embedFile = file.raw as File
}

function handleUrlExtractFileChange(file: UploadFile) {
  urlTool.extractFile = file.raw as File
}

function handleUrlBatchEmbedZipChange(file: UploadFile) {
  urlTool.batchEmbedZip = file.raw as File
}

function handleDecryptFileChange(file: UploadFile) {
  versionTool.decryptFile = file.raw as File
}
</script>

<template>
  <div class="app-container">
    <!-- 页面标题 -->
    <div class="page-header">
      <h2 class="page-title">实用工具箱</h2>
      <p class="page-desc">提供端口转换、IP隐写、版本管理等实用工具</p>
    </div>

    <!-- 工具Tab -->
    <el-tabs v-model="activeTab" type="card" class="main-tabs">
      <!-- 端口转换工具 -->
      <el-tab-pane v-if="isAdmin" label="端口转换" name="port">
        <el-card shadow="never" class="tool-card">
          <template #header>
            <div class="card-header">
              <el-icon class="header-icon" :size="20">
                <Connection />
              </el-icon>
              <span class="card-title">端口转换工具</span>
            </div>
          </template>

          <!-- 端口转企业号 -->
          <div class="tool-section">
            <div class="section-title">端口转企业号</div>
            <el-form label-width="100px">
              <el-form-item label="端口号">
                <el-input
                  v-model.number="portTool.port"
                  type="number"
                  placeholder="请输入端口号 (1-65535)"
                  min="1"
                  max="65535"
                />
              </el-form-item>
              <el-form-item label="企业号">
                <el-input v-model="portTool.enterprise" readonly placeholder="转换结果">
                  <template #append>
                    <el-button
                      v-if="portTool.enterprise"
                      :icon="DocumentCopy"
                      @click="portTool.copyEnterprise"
                    />
                  </template>
                </el-input>
              </el-form-item>
              <el-form-item>
                <el-button
                  type="primary"
                  :loading="portTool.loading"
                  :icon="Right"
                  @click="portTool.convertPort"
                >
                  转换
                </el-button>
              </el-form-item>
            </el-form>
          </div>

          <el-divider />

          <!-- 企业号转端口 -->
          <div class="tool-section">
            <div class="section-title">企业号转端口</div>
            <el-form label-width="100px">
              <el-form-item label="企业号">
                <el-input
                  v-model="portTool.enterpriseInput"
                  placeholder="请输入6位数字企业号"
                  maxlength="6"
                  show-word-limit
                />
              </el-form-item>
              <el-form-item label="端口号">
                <el-input v-model="portTool.portResult" readonly placeholder="转换结果">
                  <template #append>
                    <el-button
                      v-if="portTool.portResult"
                      :icon="DocumentCopy"
                      @click="portTool.copyPort"
                    />
                  </template>
                </el-input>
              </el-form-item>
              <el-form-item>
                <el-button
                  type="primary"
                  :loading="portTool.loading"
                  :icon="Right"
                  @click="portTool.convertEnterprise"
                >
                  转换
                </el-button>
                <el-button @click="portTool.reset">重置</el-button>
              </el-form-item>
            </el-form>
          </div>
        </el-card>
      </el-tab-pane>

      <!-- IP隐写工具 -->
      <el-tab-pane label="IP隐写" name="ip">
        <el-card shadow="never" class="tool-card">
          <template #header>
            <div class="card-header">
              <el-icon class="header-icon" :size="20">
                <Lock />
              </el-icon>
              <span class="card-title">IP隐写工具</span>
            </div>
          </template>

          <!-- IP嵌入 -->
          <div class="tool-section">
            <div class="section-title">IP嵌入到文件</div>
            <el-form label-width="100px">
              <el-form-item label="选择文件">
                <el-upload
                  :auto-upload="false"
                  :limit="1"
                  :on-change="handleEmbedFileChange"
                  :file-list="ipTool.embedFile ? [{ name: ipTool.embedFile.name, uid: 1 }] : []"
                >
                  <el-button :icon="Upload">选择文件</el-button>
                  <template #tip>
                    <div class="el-upload__tip">支持任意格式的文件</div>
                  </template>
                </el-upload>
              </el-form-item>
              <el-form-item label="IP列表">
                <el-input
                  v-model="ipTool.embedIpList"
                  type="textarea"
                  :rows="4"
                  placeholder="每行一个IP地址&#10;例如：&#10;192.168.1.1&#10;10.0.0.1"
                />
              </el-form-item>
              <el-form-item>
                <el-button
                  type="primary"
                  :loading="ipTool.loading"
                  :icon="Download"
                  @click="ipTool.embedIPsToFile"
                >
                  嵌入并下载
                </el-button>
                <el-button @click="ipTool.resetEmbed">重置</el-button>
              </el-form-item>
            </el-form>
          </div>

          <el-divider />

          <!-- 批量IP嵌入 -->
          <div class="tool-section">
            <div class="section-title">批量IP嵌入（ZIP文件）</div>
            <el-form label-width="100px">
              <el-form-item label="选择ZIP文件">
                <el-upload
                  :auto-upload="false"
                  :limit="1"
                  :on-change="handleBatchEmbedZipChange"
                  :file-list="ipTool.batchEmbedZip ? [{ name: ipTool.batchEmbedZip.name, uid: 10 }] : []"
                  accept=".zip"
                >
                  <el-button :icon="Upload">选择ZIP文件</el-button>
                  <template #tip>
                    <div class="el-upload__tip">上传包含多个文件的ZIP压缩包</div>
                  </template>
                </el-upload>
              </el-form-item>
              <el-form-item label="IP列表">
                <el-input
                  v-model="ipTool.batchEmbedIpList"
                  type="textarea"
                  :rows="4"
                  placeholder="每行一个IP地址&#10;例如：&#10;192.168.1.1&#10;10.0.0.1"
                />
              </el-form-item>
              <el-form-item>
                <el-button
                  type="primary"
                  :loading="ipTool.loading"
                  :icon="Download"
                  @click="ipTool.embedIPsToZip"
                >
                  批量嵌入并下载
                </el-button>
                <el-button @click="ipTool.resetBatchEmbed">重置</el-button>
              </el-form-item>
              <el-alert
                title="说明"
                type="info"
                :closable="false"
              >
                <template #default>
                  <div class="info-text">
                    <p>• 上传一个ZIP文件，将IP列表嵌入到压缩包内的所有文件中</p>
                    <p>• 处理完成后会自动下载包含处理结果的新ZIP文件</p>
                    <p>• 保持原ZIP文件的目录结构</p>
                  </div>
                </template>
              </el-alert>
            </el-form>
          </div>

          <el-divider />

          <!-- IP提取 -->
          <div class="tool-section">
            <div class="section-title">从文件提取IP</div>
            <el-form label-width="100px">
              <el-form-item label="选择文件">
                <el-upload
                  :auto-upload="false"
                  :limit="1"
                  :on-change="handleExtractFileChange"
                  :file-list="ipTool.extractFile ? [{ name: ipTool.extractFile.name, uid: 2 }] : []"
                >
                  <el-button :icon="Upload">选择文件</el-button>
                  <template #tip>
                    <div class="el-upload__tip">选择包含嵌入IP的文件</div>
                  </template>
                </el-upload>
              </el-form-item>
              <el-form-item>
                <el-button
                  type="success"
                  :loading="ipTool.loading"
                  :icon="View"
                  @click="ipTool.extractIPsFromFile"
                >
                  提取IP
                </el-button>
                <el-button @click="ipTool.resetExtract">重置</el-button>
              </el-form-item>
            </el-form>

            <!-- 提取结果 -->
            <div v-if="ipTool.extractedIPs.length > 0" class="result-box">
              <div class="result-header">
                <span class="result-title">提取结果 ({{ ipTool.extractedIPs.length }}个IP)</span>
                <el-button size="small" :icon="DocumentCopy" @click="ipTool.copyIPs">复制全部</el-button>
              </div>
              <div class="ip-list">
                <el-tag v-for="(ip, index) in ipTool.extractedIPs" :key="index" class="ip-tag">
                  {{ ip }}
                </el-tag>
              </div>
            </div>
          </div>
        </el-card>
      </el-tab-pane>

      <!-- URL隐写工具 -->

      <!-- IP批量上传工具 -->
      <el-tab-pane label="IP批量上传" name="ip-upload">
        <el-card shadow="never" class="tool-card">
          <template #header>
            <div class="card-header">
              <el-icon class="header-icon" :size="20">
                <UploadFilled />
              </el-icon>
              <span class="card-title">IP批量嵌入上传</span>
              <el-button type="primary" size="small" :icon="Refresh" @click="ipUploadTool.loadData" style="margin-left: auto;">
                刷新数据
              </el-button>
            </div>
          </template>

          <div v-loading="ipUploadTool.loading">
            <!-- 失效IP警告 -->
            <el-alert
              v-if="invalidSelectedIPs.length > 0"
              type="warning"
              :closable="false"
              class="invalid-ip-alert"
            >
              <template #title>
                <div class="invalid-alert-header">
                  <span>检测到 {{ invalidSelectedIPs.length }} 个失效IP（服务器IP已变更）</span>
                  <el-button type="warning" size="small" @click="clearInvalidIPs">
                    清除失效IP
                  </el-button>
                </div>
              </template>
              <div class="invalid-ip-list">
                <el-tag
                  v-for="ip in invalidSelectedIPs"
                  :key="ip"
                  type="danger"
                  size="small"
                  class="invalid-ip-tag"
                >
                  {{ ip }}
                </el-tag>
              </div>
            </el-alert>

            <!-- IP选择 -->
            <div class="tool-section">
              <div class="section-header">
                <div class="section-title" style="margin-bottom: 0;">IP选择</div>
                <div style="display: flex; align-items: center; gap: 8px;">
                  <el-select
                    v-model="ipFilterMerchantId"
                    placeholder="全部商户"
                    clearable
                    size="small"
                    style="width: 160px;"
                    @clear="ipFilterMerchantId = undefined"
                  >
                    <el-option
                      v-for="m in ipMerchantOptions"
                      :key="m.value"
                      :label="m.label"
                      :value="m.value"
                    />
                  </el-select>
                  <el-button size="small" @click="ipUploadTool.toggleAllIPs">
                    {{ ipUploadTool.selectedIPs.length === flattenedIPList.length ? '取消全选' : '全选' }}
                  </el-button>
                </div>
              </div>
              <el-checkbox-group v-model="ipUploadTool.selectedIPs" class="ip-select-list">
                <div v-for="item in filteredIPList" :key="item.ip" class="ip-select-item">
                  <el-checkbox :value="item.ip">
                    <div class="ip-select-info">
                      <el-tag :type="item.status === 1 ? 'success' : 'info'" size="small">
                        {{ item.status === 1 ? '在线' : '离线' }}
                      </el-tag>
                      <el-tag v-if="item.isAuxiliary" type="warning" size="small">辅助</el-tag>
                      <el-tag v-else type="primary" size="small">主IP</el-tag>
                      <span class="ip-address">{{ item.ip }}</span>
                      <span class="ip-server-name">{{ item.serverName }}</span>
                      <el-tag v-if="item.merchantName" size="small" type="danger" style="margin-left: 4px;">{{ item.merchantName }}</el-tag>
                    </div>
                  </el-checkbox>
                </div>
              </el-checkbox-group>
              <el-empty v-if="filteredIPList.length === 0" description="暂无可用IP" :image-size="60" />
              <div v-else class="file-tip">
                已选择 {{ ipUploadTool.selectedIPs.length }} / {{ flattenedIPList.length }} 个IP
                <span v-if="ipFilterMerchantId">（当前显示 {{ filteredIPList.length }} 个）</span>
              </div>
            </div>

            <el-divider />

            <!-- 上传目标选��� -->
            <div class="tool-section">
              <div class="section-header">
                <div class="section-title" style="margin-bottom: 0;">上传目标选择</div>
                <div class="section-actions">
                  <el-select
                    v-model="filterMerchantId"
                    placeholder="全部商户"
                    clearable
                    size="small"
                    style="width: 160px;"
                    @clear="filterMerchantId = undefined"
                  >
                    <el-option
                      v-for="m in targetMerchantOptions"
                      :key="m.value"
                      :label="m.label"
                      :value="m.value"
                    />
                  </el-select>
                  <el-button size="small" type="primary" :icon="Plus" @click="targetManagement.openCreate">
                    新增目标
                  </el-button>
                  <el-button size="small" @click="ipUploadTool.toggleAllTargets">
                    {{ ipUploadTool.selectedTargetIndexes.length === ipUploadTool.targets.length ? '取消全选' : '全选' }}
                  </el-button>
                </div>
              </div>
              <div class="target-list">
                <div v-for="group in groupedTargets" :key="group.groupId" class="target-group">
                  <!-- 商户级别（可折叠） -->
                  <div class="target-group-header target-group-header-clickable" :class="{ 'is-expanded': expandedGroupId === group.groupId }" @click="toggleExpandGroup(group.groupId)">
                    <div class="target-group-left">
                      <el-icon class="expand-arrow" :class="{ 'is-expanded': expandedGroupId === group.groupId }">
                        <ArrowRight />
                      </el-icon>
                      <el-checkbox
                        :model-value="isGroupAllSelected(group)"
                        :indeterminate="isGroupIndeterminate(group)"
                        @change="toggleGroupTargets(group)"
                        @click.stop
                      >
                        <div class="target-group-title">
                          <span>{{ group.groupName }}</span>
                          <span class="target-group-count">({{ group.allTargets.length }}个目标)</span>
                        </div>
                      </el-checkbox>
                    </div>
                    <div class="target-group-summary">
                      <el-tag v-if="getGroupSelectedCount(group) > 0" type="success" size="small">
                        已选 {{ getGroupSelectedCount(group) }}
                      </el-tag>
                    </div>
                  </div>
                  <!-- 展开后显示云账号子级和目标 -->
                  <template v-if="expandedGroupId === group.groupId">
                    <div v-for="sub in group.subGroups" :key="sub.accountId" class="target-sub-group">
                      <div class="target-sub-group-header">
                        <el-checkbox
                          :model-value="isSubGroupAllSelected(sub)"
                          :indeterminate="isSubGroupIndeterminate(sub)"
                          @change="toggleSubGroupTargets(sub)"
                        >
                          <div class="target-sub-group-title">
                            <el-tag :type="cloudTypeColors[sub.cloudType]" size="small">
                              {{ sub.cloudType.toUpperCase() }}
                            </el-tag>
                            <span>{{ sub.accountName }}</span>
                            <span class="target-group-count">({{ sub.targets.length }})</span>
                          </div>
                        </el-checkbox>
                      </div>
                      <div v-for="target in sub.targets" :key="target.index" class="target-item target-item-grouped">
                        <el-checkbox :model-value="ipUploadTool.selectedTargetIndexes.includes(target.index)"
                          @change="toggleTargetIndex(target.index)">
                          <div class="target-info">
                            <span class="target-name">{{ target.name }}</span>
                            <span class="target-detail">
                              {{ target.bucket }} / {{ target.object_prefix || '(根目录)' }}
                            </span>
                            <el-tag v-if="!target.enabled" type="info" size="small">已禁用</el-tag>
                          </div>
                        </el-checkbox>
                        <div class="target-actions" @click.stop>
                          <el-button size="small" link type="primary" @click="targetManagement.openEdit(target)">
                            编辑
                          </el-button>
                          <el-button size="small" link :type="target.enabled ? 'warning' : 'success'" @click="targetManagement.handleToggle(target)">
                            {{ target.enabled ? '禁用' : '启用' }}
                          </el-button>
                          <el-button size="small" link type="danger" @click="targetManagement.handleDelete(target)">
                            删除
                          </el-button>
                      </div>
                    </div>
                  </div>
                  </template>
                </div>
              </div>
              <el-empty v-if="ipUploadTool.targets.length === 0" description="暂无配置的上传目标，请点击上方「新增目标」按钮添加" :image-size="60" />
            </div>

            <el-divider />

            <!-- 源文件列表 -->
            <div class="tool-section">
              <div class="section-header">
                <div class="section-title" style="margin-bottom: 0;">
                  源文件列表
                  <span class="source-dir">{{ ipUploadTool.sourceDir }}</span>
                </div>
                <el-button size="small" @click="ipUploadTool.toggleAllFiles">
                  {{ ipUploadTool.selectedFileNames.length === ipUploadTool.sourceFiles.length ? '取消全选' : '全选' }}
                </el-button>
              </div>
              <el-checkbox-group v-model="ipUploadTool.selectedFileNames">
                <el-table :data="ipUploadTool.sourceFiles" max-height="300" border>
                  <el-table-column width="50">
                    <template #default="{ row }">
                      <el-checkbox :value="row.name" />
                    </template>
                  </el-table-column>
                  <el-table-column prop="name" label="文件名" />
                  <el-table-column label="大小" width="100">
                    <template #default="{ row }">{{ formatFileSize(row.size) }}</template>
                  </el-table-column>
                  <el-table-column prop="mod_time" label="修改时间" width="180" />
                </el-table>
              </el-checkbox-group>
              <div class="file-tip">
                提示：不选择则处���所有文件（已选 {{ ipUploadTool.selectedFileNames.length || ipUploadTool.sourceFiles.length }} 个）
              </div>
            </div>

            <el-divider />

            <!-- 执行按钮 -->
            <div class="tool-section">
              <el-button
                type="primary"
                size="large"
                :loading="ipUploadTool.loading"
                :disabled="ipUploadTool.selectedIPs.length === 0 || ipUploadTool.selectedTargetIndexes.length === 0"
                :icon="UploadFilled"
                @click="ipUploadTool.handleExecute"
              >
                执行IP嵌入并上传
              </el-button>
            </div>
          </div>
        </el-card>

        <!-- 结果弹窗 -->
        <el-dialog v-model="ipUploadTool.showResult" title="执行结果" width="900px">
          <template v-if="ipUploadTool.executeResult">
            <div class="result-summary">
              <el-tag type="success" size="large">成功: {{ ipUploadTool.executeResult.summary.success_count }}</el-tag>
              <el-tag type="danger" size="large" class="ml-2">失败: {{ ipUploadTool.executeResult.summary.fail_count }}</el-tag>
              <el-tag type="info" size="large" class="ml-2">耗时: {{ ipUploadTool.executeResult.summary.duration }}</el-tag>
            </div>

            <el-table :data="ipUploadTool.executeResult.results" max-height="400" border class="mt-4">
              <el-table-column prop="file_name" label="文件名" width="150" />
              <el-table-column prop="target_name" label="目标" width="120" />
              <el-table-column label="云类型" width="80">
                <template #default="{ row }">
                  <el-tag :type="cloudTypeColors[row.cloud_type]" size="small">
                    {{ row.cloud_type }}
                  </el-tag>
                </template>
              </el-table-column>
              <el-table-column label="状态" width="80">
                <template #default="{ row }">
                  <el-tag :type="row.success ? 'success' : 'danger'">
                    {{ row.success ? '成功' : '失败' }}
                  </el-tag>
                </template>
              </el-table-column>
              <el-table-column label="URL/错误" min-width="200">
                <template #default="{ row }">
                  <template v-if="row.success && row.object_url">
                    <el-button link type="primary" size="small" @click="ipUploadTool.copyUrl(row.object_url)">
                      复制URL
                    </el-button>
                  </template>
                  <span v-else class="error-text">{{ row.error }}</span>
                </template>
              </el-table-column>
            </el-table>
          </template>
        </el-dialog>

        <!-- 目标管理弹窗 -->
        <el-dialog
          v-model="targetManagement.showDialog"
          :title="targetManagement.dialogMode === 'create' ? '批量添加上传目标' : '编辑上传目标'"
          width="650px"
        >
          <el-form :model="targetManagement.form" label-width="100px">
            <!-- 编辑模式才显示目标名称 -->
            <el-form-item v-if="targetManagement.dialogMode === 'edit'" label="目标名称" required>
              <el-input v-model="targetManagement.form.name" placeholder="请输入目标名称" />
            </el-form-item>
            <el-form-item label="所属商户" required>
              <el-select v-model="targetManagement.form.merchant_id" placeholder="请选择商户" style="width: 100%;" filterable>
                <el-option
                  v-for="m in merchantOptions"
                  :key="m.value"
                  :label="m.label"
                  :value="m.value"
                />
              </el-select>
            </el-form-item>
            <el-form-item label="云账号" required>
              <el-select
                v-model="targetManagement.form.cloud_account_id"
                placeholder="请先选择商户，再选择云账号"
                style="width: 100%;"
                :disabled="!targetManagement.form.merchant_id"
              >
                <el-option
                  v-for="acc in targetManagement.cloudAccounts"
                  :key="acc.value"
                  :label="`${acc.label} (${acc.type})`"
                  :value="acc.value"
                />
              </el-select>
            </el-form-item>

            <!-- 创建模式：多选Bucket -->
            <el-form-item v-if="targetManagement.dialogMode === 'create'" label="选择Bucket" required>
              <div class="bucket-select-box" v-loading="targetManagement.bucketsLoading">
                <div v-if="!targetManagement.form.cloud_account_id" class="bucket-placeholder">
                  请先选择云账号
                </div>
                <template v-else-if="targetManagement.buckets.length > 0">
                  <div class="bucket-header">
                    <el-button size="small" @click="targetManagement.toggleAllBuckets">
                      {{ targetManagement.selectedBuckets.length === targetManagement.buckets.length ? '取消全选' : '全选' }}
                    </el-button>
                    <span class="bucket-count">已选 {{ targetManagement.selectedBuckets.length }} / {{ targetManagement.buckets.length }}</span>
                  </div>
                  <el-checkbox-group v-model="targetManagement.selectedBuckets" class="bucket-checkbox-group">
                    <el-checkbox
                      v-for="bucket in targetManagement.buckets"
                      :key="bucket.name"
                      :label="bucket.name"
                      class="bucket-checkbox-item"
                    >
                      <span class="bucket-name">{{ bucket.name }}</span>
                      <span v-if="bucket.location" class="bucket-location">({{ bucket.location }})</span>
                    </el-checkbox>
                  </el-checkbox-group>
                </template>
                <el-empty v-else description="该账号下暂无Bucket" :image-size="40" />
              </div>
            </el-form-item>

            <!-- 编辑模式：单选Bucket -->
            <el-form-item v-else label="Bucket" required>
              <el-select
                v-model="targetManagement.form.bucket"
                placeholder="请先选择云账号，再选择Bucket"
                style="width: 100%;"
                :loading="targetManagement.bucketsLoading"
                :disabled="!targetManagement.form.cloud_account_id"
                @change="(val: string) => {
                  const bucket = targetManagement.buckets.find(b => b.name === val)
                  if (bucket && bucket.location) {
                    targetManagement.form.region_id = bucket.location
                  }
                }"
              >
                <el-option
                  v-for="bucket in targetManagement.buckets"
                  :key="bucket.name"
                  :label="`${bucket.name}${bucket.location ? ' (' + bucket.location + ')' : ''}`"
                  :value="bucket.name"
                />
              </el-select>
            </el-form-item>

            <!-- 编辑模式才显示区域ID -->
            <el-form-item v-if="targetManagement.dialogMode === 'edit'" label="区域ID" required>
              <el-input v-model="targetManagement.form.region_id" placeholder="选择Bucket后自动填充，或手动输入" />
            </el-form-item>
            <el-form-item label="对象前缀">
              <el-input v-model="targetManagement.form.object_prefix" placeholder="可选，例如: images/" />
            </el-form-item>
            <!-- 编辑模式才显示排序 -->
            <el-form-item v-if="targetManagement.dialogMode === 'edit'" label="排序">
              <el-input-number v-model="targetManagement.form.sort_order" :min="0" />
            </el-form-item>
            <el-form-item label="启用状态">
              <el-switch v-model="targetManagement.form.enabled" />
            </el-form-item>
          </el-form>
          <template #footer>
            <el-button @click="targetManagement.showDialog = false">取消</el-button>
            <el-button type="primary" :loading="targetManagement.loading" @click="targetManagement.saveTarget">
              {{ targetManagement.dialogMode === 'create' ? `添加 ${targetManagement.selectedBuckets.length} 个目标` : '保存' }}
            </el-button>
          </template>
        </el-dialog>
      </el-tab-pane>

      <!-- 版本管理工具 -->
    </el-tabs>
  </div>
</template>

<style lang="scss" scoped>
.app-container {
  padding: 20px;
}

.page-header {
  margin-bottom: 24px;
  padding: 24px;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  border-radius: 12px;
  color: white;

  .page-title {
    margin: 0;
    font-size: 28px;
    font-weight: 600;
  }

  .page-desc {
    margin: 8px 0 0 0;
    font-size: 14px;
    opacity: 0.9;
  }
}

.main-tabs {
  :deep(.el-tabs__header) {
    margin-bottom: 20px;
  }

  :deep(.el-tabs__item) {
    font-size: 15px;
    font-weight: 500;
    padding: 0 24px;
    height: 44px;
    line-height: 44px;
  }

  :deep(.el-tabs__content) {
    overflow: visible;
  }
}

.tool-card {
  border: none;
  margin-bottom: 20px;

  :deep(.el-card__header) {
    background: #f5f7fa;
    padding: 16px 20px;
    border-bottom: 2px solid #e4e7ed;
  }

  :deep(.el-card__body) {
    padding: 24px;
  }
}

.card-header {
  display: flex;
  align-items: center;
  gap: 10px;

  .header-icon {
    font-size: 20px;
    color: var(--el-color-primary);
  }

  .card-title {
    font-size: 17px;
    font-weight: 600;
    color: #303133;
  }
}

.tool-section {
  .section-title {
    font-size: 16px;
    font-weight: 600;
    margin-bottom: 20px;
    padding-left: 12px;
    border-left: 4px solid var(--el-color-primary);
    color: #303133;
  }

  .form-tip {
    font-size: 12px;
    color: var(--el-text-color-secondary);
    margin-top: 4px;
  }
}

.result-box {
  margin-top: 20px;
  padding: 16px;
  background: #f5f7fa;
  border-radius: 8px;
  border: 1px solid #dcdfe6;

  .result-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 12px;

    .result-title {
      font-size: 14px;
      font-weight: 600;
      color: #606266;
    }
  }

  .ip-list {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;

    .ip-tag {
      font-family: "Courier New", monospace;
      font-size: 13px;
    }
  }
}

:deep(.el-divider) {
  margin: 24px 0;
}

:deep(.el-form-item) {
  margin-bottom: 20px;
}

:deep(.el-upload__tip) {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  margin-top: 4px;
}

.version-list {
  width: 100%;

  .version-item {
    display: flex;
    gap: 12px;
    margin-bottom: 12px;
    align-items: center;

    .version-input {
      flex: 1;
    }
  }
}

.info-text {
  font-size: 13px;
  line-height: 1.8;

  p {
    margin: 4px 0;
  }
}

.version-tabs {
  border: none;
  box-shadow: none;

  :deep(.el-tabs__header) {
    margin: 0;
    background: #f5f7fa;
  }

  :deep(.el-tabs__content) {
    padding: 20px;
  }

  .tab-label {
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: 14px;
  }
}

// IP批量上传样式
.section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

// IP选择列表样式
.ip-select-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
  width: 100%;
}

.ip-select-item {
  padding: 10px 16px;
  border: 1px solid #ebeef5;
  border-radius: 8px;
  transition: background 0.2s;

  &:hover {
    background: #f5f7fa;
  }
}

.ip-select-info {
  display: flex;
  align-items: center;
  gap: 8px;
}

.ip-address {
  font-weight: 500;
  font-family: "Courier New", monospace;
  font-size: 14px;
  color: #303133;
}

.ip-server-name {
  color: #909399;
  font-size: 13px;
  margin-left: 8px;
  padding-left: 8px;
  border-left: 1px solid #dcdfe6;
}

// 失效IP警告样式
.invalid-ip-alert {
  margin-bottom: 20px;

  .invalid-alert-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    width: 100%;
  }

  .invalid-ip-list {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
    margin-top: 12px;
  }

  .invalid-ip-tag {
    font-family: "Courier New", monospace;
  }
}

.ip-list-box {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  padding: 12px;
  background: #f5f7fa;
  border-radius: 8px;
  min-height: 60px;
}

.target-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
  width: 100%;
}

.target-group {
  border: 1px solid #e4e7ed;
  border-radius: 8px;
  overflow: hidden;
}

.target-group-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 16px;
  background: #eef1f6;

  &.is-expanded {
    border-bottom: 1px solid #e4e7ed;
  }
}

.target-group-header-clickable {
  cursor: pointer;
  user-select: none;
  transition: background 0.2s;

  &:hover {
    background: #e4e8ef;
  }
}

.target-group-left {
  display: flex;
  align-items: center;
  gap: 4px;
}

.expand-arrow {
  font-size: 14px;
  color: #606266;
  transition: transform 0.2s;
  flex-shrink: 0;

  &.is-expanded {
    transform: rotate(90deg);
  }
}

.target-group-summary {
  flex-shrink: 0;
}

.target-sub-group {
  border-top: 1px solid #ebeef5;

  &:first-child {
    border-top: none;
  }
}

.target-sub-group-header {
  display: flex;
  align-items: center;
  padding: 8px 16px 8px 32px;
  background: #f5f7fa;
  border-bottom: 1px solid #ebeef5;
}

.target-sub-group-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-weight: 500;
  font-size: 13px;
  color: #606266;
}

.target-group-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-weight: 600;
  font-size: 14px;
  color: #303133;
}

.target-group-count {
  color: #909399;
  font-weight: normal;
  font-size: 13px;
}

.target-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  border: 1px solid #ebeef5;
  border-radius: 8px;
  transition: background 0.2s;

  &:hover {
    background: #f5f7fa;
  }
}

.target-item-grouped {
  border: none;
  border-radius: 0;
  border-bottom: 1px solid #ebeef5;
  padding-left: 48px;

  &:last-child {
    border-bottom: none;
  }
}

.target-actions {
  display: flex;
  gap: 8px;
  flex-shrink: 0;
}

.section-actions {
  display: flex;
  gap: 8px;
}

.target-info {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.target-name {
  font-weight: 500;
  color: #303133;
}

.target-detail {
  color: #909399;
  font-size: 12px;
}

.source-dir {
  font-size: 12px;
  color: #909399;
  margin-left: 8px;
  font-weight: normal;
}

.file-tip {
  margin-top: 12px;
  font-size: 13px;
  color: #909399;
}

.result-summary {
  padding: 16px;
  background: #f5f7fa;
  border-radius: 8px;
}

.error-text {
  color: #f56c6c;
  font-size: 12px;
}

.ml-2 {
  margin-left: 8px;
}

.mt-4 {
  margin-top: 16px;
}

// Bucket多选样式
.bucket-select-box {
  width: 100%;
  min-height: 100px;
  max-height: 300px;
  overflow-y: auto;
  border: 1px solid #dcdfe6;
  border-radius: 8px;
  padding: 12px;
  background: #fafafa;
}

.bucket-placeholder {
  color: #909399;
  text-align: center;
  padding: 20px;
}

.bucket-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
  padding-bottom: 10px;
  border-bottom: 1px solid #ebeef5;
}

.bucket-count {
  font-size: 13px;
  color: #909399;
}

.bucket-checkbox-group {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.bucket-checkbox-item {
  display: flex;
  align-items: center;
  padding: 8px 12px;
  background: #fff;
  border: 1px solid #ebeef5;
  border-radius: 6px;
  transition: all 0.2s;

  &:hover {
    background: #f5f7fa;
    border-color: var(--el-color-primary-light-5);
  }
}

.bucket-name {
  font-weight: 500;
  color: #303133;
}

.bucket-location {
  font-size: 12px;
  color: #909399;
  margin-left: 6px;
}

// 分组管理弹窗样式
.group-create-row {
  display: flex;
  gap: 12px;
  margin-bottom: 20px;
}

.group-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.group-list-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  border: 1px solid #ebeef5;
  border-radius: 8px;
  transition: background 0.2s;

  &:hover {
    background: #f5f7fa;
  }
}

.group-list-info {
  display: flex;
  align-items: center;
  gap: 10px;
}

.group-list-name {
  font-weight: 500;
  color: #303133;
  font-size: 14px;
}

.group-list-actions {
  display: flex;
  gap: 8px;
}

// 响应式
@media (max-width: 768px) {
  .app-container {
    padding: 12px;
  }

  .tool-card {
    margin-bottom: 16px;
  }

  .version-list {
    .version-item {
      flex-direction: column;
      gap: 8px;

      .version-input {
        width: 100%;
      }
    }
  }
}
</style>
