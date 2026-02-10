function initStarNotification() {
  // 定义获取通知的函数
  const fetchNotifications = () => {
    /*
    getNotifyListApi().then((res) => {
      if (res.data.notify.length == 0) {
        return
      }

      // 顺序显示通知，每条通知之间有延迟
      res.data.notify.forEach((notify, index) => {
        // 设置延迟，每条通知间隔1秒显示
        setTimeout(() => {
          ElNotification({
            title: notify.title,
            type: notify.typ as "success" | "warning" | "info" | "error",
            message: h(
              "div",
              null,
              [
                h("div", null, notify.message),
                notify.href ? h("a", { style: "color: teal", target: "_blank", href: notify.href }, "查看详情") : null
              ]
            ),
            duration: notify.duration*1000 || 0,
            position: "bottom-right"
          })
        }, index * 1000) // 每条通知延迟显示，第一条立即显示，第二条延迟1秒，第三条延迟2秒...
      })
    }).catch((error) => {
      console.error("获取通知失败:", error)
    })
    */
  }

  // 立即执行一次
  fetchNotifications()
  return

  // 设置每5秒执行一次
  const timer = setInterval(fetchNotifications, 5000)

  // 返回清除定时器的函数，以便在需要时停止
  return () => {
    clearInterval(timer)
  }
}

export function usePany() {
  return { initStarNotification }
}
