package ip_embed

import (
	"fmt"

	"server/internal/server/middleware"
	"server/internal/server/model"
	"server/internal/server/service/ip_embed"
	"server/pkg/result"

	"github.com/gin-gonic/gin"
)

// Routes 注册IP嵌入上传相关路由
func Routes(gi gin.IRouter) {
	group := gi.Group("ip-embed", middleware.Authorization)

	group.GET("system-ips", getSystemIPs)        // 获取系统服务器IP列表
	group.GET("targets", getTargets)             // 获取上传目标配置
	group.GET("source-files", getSourceFiles)    // 获取源文件列表
	group.POST("execute", executeEmbedAndUpload) // 执行批量嵌入并上传
	group.GET("selected-ips", getSelectedIPs)    // 获取上次选中的IP
	group.POST("selected-ips", saveSelectedIPs)  // 保存选中的IP

	// 目标管理 CRUD
	group.POST("targets", createTarget)           // 创建上传目标
	group.PUT("targets/:id", updateTarget)        // 更新上传目标
	group.DELETE("targets/:id", deleteTarget)     // 删除上传目标
	group.PUT("targets/:id/toggle", toggleTarget) // 切换目标启用状态
}

// getSystemIPs 获取系统服务器IP列表
func getSystemIPs(c *gin.Context) {
	resp, err := ip_embed.GetSystemIPs()
	if err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, resp)
}

// getTargets 获取上传目标配置列表
func getTargets(c *gin.Context) {
	resp, err := ip_embed.GetTargets()
	if err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, resp)
}

// getSourceFiles 获取源文件列表
func getSourceFiles(c *gin.Context) {
	resp, err := ip_embed.GetSourceFiles()
	if err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, resp)
}

// executeEmbedAndUpload 执行批量嵌入并上传
func executeEmbedAndUpload(c *gin.Context) {
	var req model.ExecuteEmbedReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.GParamErr(c, err)
		return
	}

	resp, err := ip_embed.ExecuteEmbedAndUpload(req)
	if err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, resp)
}

// getSelectedIPs 获取上次选中的IP列表
func getSelectedIPs(c *gin.Context) {
	resp, err := ip_embed.GetSelectedIPs()
	if err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, resp)
}

// saveSelectedIPs 保存选中的IP列表
func saveSelectedIPs(c *gin.Context) {
	var req model.SaveSelectedIPsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.GParamErr(c, err)
		return
	}

	if err := ip_embed.SaveSelectedIPs(req.IPs); err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, nil)
}

// createTarget 创建上传目标
func createTarget(c *gin.Context) {
	var req model.CreateTargetReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.GParamErr(c, err)
		return
	}

	id, err := ip_embed.CreateTarget(req)
	if err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, map[string]int{"id": id})
}

// updateTarget 更新上传目标
func updateTarget(c *gin.Context) {
	idStr := c.Param("id")
	id := 0
	if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil || id <= 0 {
		result.GParamErr(c, fmt.Errorf("无效的目标ID"))
		return
	}

	var req model.UpdateTargetReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.GParamErr(c, err)
		return
	}

	if err := ip_embed.UpdateTarget(id, req); err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, nil)
}

// deleteTarget 删除上传目标
func deleteTarget(c *gin.Context) {
	idStr := c.Param("id")
	id := 0
	if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil || id <= 0 {
		result.GParamErr(c, fmt.Errorf("无效的目标ID"))
		return
	}

	if err := ip_embed.DeleteTarget(id); err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, nil)
}

// toggleTarget 切换目标启用状态
func toggleTarget(c *gin.Context) {
	idStr := c.Param("id")
	id := 0
	if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil || id <= 0 {
		result.GParamErr(c, fmt.Errorf("无效的目标ID"))
		return
	}

	if err := ip_embed.ToggleTarget(id); err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, nil)
}
