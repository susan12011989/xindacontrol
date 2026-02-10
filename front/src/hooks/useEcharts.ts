import type { EChartsOption } from "echarts"
import * as echarts from "echarts"
import { onUnmounted } from "vue"

export function useEcharts(el: HTMLElement) {
  // 创建 echarts 实例
  const chartInstance = echarts.init(el)

  // 设置图表配置项
  const setOptions = (options: EChartsOption) => {
    chartInstance.setOption(options)
  }

  // 调整图表大小
  const resize = () => {
    chartInstance.resize()
  }

  // 监听窗口大小变化
  window.addEventListener("resize", resize)

  // 组件卸载时释放资源
  onUnmounted(() => {
    window.removeEventListener("resize", resize)
    chartInstance.dispose()
  })

  return {
    chartInstance,
    setOptions,
    resize
  }
}
