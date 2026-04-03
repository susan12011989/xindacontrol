import { pinia } from "@/pinia"
import { request } from "@/http/axios"

export type ControlMode = "local" | "cluster"

export const useControlModeStore = defineStore("control-mode", () => {
  const mode = ref<ControlMode>("cluster")
  const loaded = ref(false)

  const isLocal = computed(() => mode.value === "local")
  const isCluster = computed(() => mode.value === "cluster")

  // 从后端获取当前控制模式
  const fetchMode = async () => {
    try {
      const res = await request<{ mode: ControlMode }>({
        url: "control/mode",
        method: "get"
      })
      mode.value = (res as any).data?.mode || res.mode || "cluster"
    } catch {
      // 接口不可用时默认多机模式（兼容旧版后端）
      mode.value = "cluster"
    }
    loaded.value = true
  }

  return { mode, loaded, isLocal, isCluster, fetchMode }
})

// 在 setup 外使用
export function useControlModeStoreHook() {
  return useControlModeStore(pinia)
}
