package project

import (
	"strconv"
	"sync"
	"time"

	"server/internal/server/middleware"
	"server/internal/server/service/project"
	"server/pkg/result"

	"github.com/gin-gonic/gin"
)

func Routes(ge gin.IRouter) {
	projectGroup := ge.Group("project", middleware.Authorization)

	// 项目 CRUD
	projectGroup.GET("", listProjects)
	projectGroup.POST("", createProject)
	projectGroup.GET("/options", getProjectOptions) // 下拉选项（放在 :id 前面避免路由冲突）
	projectGroup.GET("/:id", getProject)
	projectGroup.PUT("/:id", updateProject)
	projectGroup.DELETE("/:id", deleteProject)

	// 项目 GOST 服务器管理
	projectGroup.GET("/:id/gost-servers", listProjectGostServers)
	projectGroup.POST("/:id/gost-servers", createProjectGostServer)
	projectGroup.PUT("/gost-servers/:relation_id", updateProjectGostServer)
	projectGroup.DELETE("/gost-servers/:relation_id", deleteProjectGostServer)

	// 项目商户管理
	projectGroup.GET("/:id/merchants", listProjectMerchants)
	projectGroup.POST("/:id/merchants", addMerchantToProject)
	projectGroup.POST("/:id/merchants/batch", batchAddMerchantsToProject)
	projectGroup.DELETE("/:id/merchants/:merchant_id", removeMerchantFromProject)

	// 项目 GOST IP 同步
	projectGroup.GET("/:id/sync-status", getProjectSyncStatus)
	projectGroup.POST("/:id/sync-gost-ip", syncProjectGostIP)
}

// ===== 简易防抖 =====
type opRecord struct {
	last time.Time
	mu   sync.Mutex
}

var debounceMap sync.Map

func allowAndMark(key string, window time.Duration) bool {
	now := time.Now()
	val, _ := debounceMap.LoadOrStore(key, &opRecord{})
	rec := val.(*opRecord)
	rec.mu.Lock()
	defer rec.mu.Unlock()
	if !rec.last.IsZero() && now.Sub(rec.last) < window {
		return false
	}
	rec.last = now
	return true
}

// ========== 项目 CRUD ==========

// 获取项目列表
func listProjects(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "10"))
	name := c.Query("name")

	list, total, err := project.ListProjects(page, size, name)
	if err != nil {
		result.GErr(c, err)
		return
	}

	result.GOK(c, gin.H{
		"list":  list,
		"total": total,
	})
}

// 创建项目
func createProject(c *gin.Context) {
	var req project.ProjectReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.GParamErr(c, err)
		return
	}

	id, err := project.CreateProject(req)
	if err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, gin.H{"id": id})
}

// 获取项目详情
func getProject(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		result.GParamErr(c, err)
		return
	}

	data, err := project.GetProject(id)
	if err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, data)
}

// 更新项目
func updateProject(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		result.GParamErr(c, err)
		return
	}

	var req project.ProjectReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.GParamErr(c, err)
		return
	}

	err = project.UpdateProject(id, req)
	if err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, nil)
}

// 删除项目
func deleteProject(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		result.GParamErr(c, err)
		return
	}

	err = project.DeleteProject(id)
	if err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, nil)
}

// 获取项目选项列表（下拉框）
func getProjectOptions(c *gin.Context) {
	options, err := project.GetProjectOptions()
	if err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, options)
}

// ========== 项目 GOST 服务器管理 ==========

// 获取项目 GOST 服务器列表
func listProjectGostServers(c *gin.Context) {
	projectId, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		result.GParamErr(c, err)
		return
	}

	list, err := project.ListProjectGostServers(projectId)
	if err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, list)
}

// 为项目添加 GOST 服务器
func createProjectGostServer(c *gin.Context) {
	projectId, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		result.GParamErr(c, err)
		return
	}

	var req project.ProjectGostServerReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.GParamErr(c, err)
		return
	}

	id, err := project.CreateProjectGostServer(projectId, req)
	if err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, gin.H{"id": id})
}

// 更新项目 GOST 服务器
func updateProjectGostServer(c *gin.Context) {
	relationId, err := strconv.Atoi(c.Param("relation_id"))
	if err != nil {
		result.GParamErr(c, err)
		return
	}

	var req project.ProjectGostServerReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.GParamErr(c, err)
		return
	}

	err = project.UpdateProjectGostServer(relationId, req)
	if err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, nil)
}

// 删除项目 GOST 服务器
func deleteProjectGostServer(c *gin.Context) {
	relationId, err := strconv.Atoi(c.Param("relation_id"))
	if err != nil {
		result.GParamErr(c, err)
		return
	}

	err = project.DeleteProjectGostServer(relationId)
	if err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, nil)
}

// ========== 项目商户管理 ==========

// 获取项目商户列表
func listProjectMerchants(c *gin.Context) {
	projectId, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		result.GParamErr(c, err)
		return
	}

	list, err := project.ListProjectMerchants(projectId)
	if err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, list)
}

// 添加商户到项目
func addMerchantToProject(c *gin.Context) {
	projectId, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		result.GParamErr(c, err)
		return
	}

	var req struct {
		MerchantId int `json:"merchant_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		result.GParamErr(c, err)
		return
	}

	err = project.AddMerchantToProject(projectId, req.MerchantId)
	if err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, nil)
}

// 批量添加商户到项目
func batchAddMerchantsToProject(c *gin.Context) {
	projectId, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		result.GParamErr(c, err)
		return
	}

	var req struct {
		MerchantIds []int `json:"merchant_ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		result.GParamErr(c, err)
		return
	}

	err = project.BatchAddMerchantsToProject(projectId, req.MerchantIds)
	if err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, nil)
}

// 从项目移除商户
func removeMerchantFromProject(c *gin.Context) {
	merchantId, err := strconv.Atoi(c.Param("merchant_id"))
	if err != nil {
		result.GParamErr(c, err)
		return
	}

	err = project.RemoveMerchantFromProject(merchantId)
	if err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, nil)
}

// ========== 项目 GOST IP 同步 ==========

// 获取项目同步状态
func getProjectSyncStatus(c *gin.Context) {
	projectId, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		result.GParamErr(c, err)
		return
	}

	data, err := project.GetProjectSyncStatus(projectId)
	if err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, data)
}

// 同步项目 GOST IP 到所有商户 OSS
func syncProjectGostIP(c *gin.Context) {
	projectId, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		result.GParamErr(c, err)
		return
	}

	var req project.SyncProjectGostIPReq
	if err := c.ShouldBindJSON(&req); err != nil {
		// 允许空 body，使用默认参数
		req = project.SyncProjectGostIPReq{}
	}
	req.ProjectId = projectId

	// 防抖：同一项目 10 秒内只允许一次
	key := "syncprojectgostip:" + strconv.Itoa(projectId)
	if !allowAndMark(key, 10*time.Second) {
		result.GResult(c, 429, nil, "操作过于频繁，请稍后再试")
		return
	}

	data, err := project.SyncProjectGostIP(req)
	if err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, data)
}
