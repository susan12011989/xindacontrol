package build

import (
	"context"
	"encoding/json"
	"fmt"
	"server/internal/server/middleware"
	"server/internal/server/model"
	"server/pkg/buildqueue"
	"server/pkg/dbs"
	"server/pkg/result"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func Routes(ge gin.IRouter) {
	build := ge.Group("build", middleware.Authorization)

	// 商户配置
	build.GET("/merchants", listMerchants)
	build.GET("/merchants/:id", getMerchant)
	build.POST("/merchants", createMerchant)
	build.PUT("/merchants/:id", updateMerchant)
	build.DELETE("/merchants/:id", deleteMerchant)
	build.POST("/merchants/:id/icon", uploadIcon)

	// 构建任务
	build.GET("/tasks", listTasks)
	build.GET("/tasks/:id", getTask)
	build.GET("/tasks/:id/progress", getTaskProgress)
	build.POST("/tasks", createTask)
	build.POST("/tasks/:id/cancel", cancelTask)
	build.POST("/tasks/:id/retry", retryTask)

	// 产物管理
	build.GET("/artifacts", listArtifacts)
	build.GET("/artifacts/:id/download", downloadArtifact)
	build.DELETE("/artifacts/expired", cleanExpiredArtifacts)

	// 统计
	build.GET("/stats", getStats)

	// 构建服务器
	build.GET("/servers", listServers)
	build.POST("/servers", createServer)
	build.PUT("/servers/:id", updateServer)
	build.DELETE("/servers/:id", deleteServer)
}

// ========== 商户配置 ==========

func listMerchants(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	name := c.Query("name")
	status := c.Query("status")

	offset := (page - 1) * size
	var list []model.BuildMerchant
	session := dbs.DBAdmin.NewSession()
	defer session.Close()

	if name != "" {
		session.Where("name LIKE ?", "%"+name+"%")
	}
	if status != "" {
		session.Where("status = ?", status)
	}

	total, err := session.Limit(size, offset).Desc("id").FindAndCount(&list)
	if err != nil {
		result.GErr(c, err)
		return
	}

	result.GOK(c, gin.H{
		"list":  list,
		"total": total,
	})
}

func getMerchant(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var merchant model.BuildMerchant
	has, err := dbs.DBAdmin.ID(id).Get(&merchant)
	if err != nil {
		result.GErr(c, err)
		return
	}
	if !has {
		result.GResult(c, 404, nil, "配置不存在")
		return
	}
	result.GOK(c, merchant)
}

func createMerchant(c *gin.Context) {
	var req model.BuildMerchantReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.GErr(c, err)
		return
	}

	merchant := model.BuildMerchant{
		MerchantID:         req.MerchantID,
		Name:               req.Name,
		AppName:            req.AppName,
		ShortName:          req.ShortName,
		Description:        req.Description,
		AndroidPackage:     req.AndroidPackage,
		AndroidVersionCode: req.AndroidVersionCode,
		AndroidVersionName: req.AndroidVersionName,
		IOSBundleID:        req.IOSBundleID,
		IOSVersion:         req.IOSVersion,
		IOSBuild:           req.IOSBuild,
		WindowsAppName:     req.WindowsAppName,
		WindowsVersion:     req.WindowsVersion,
		MacOSBundleID:      req.MacOSBundleID,
		MacOSAppName:       req.MacOSAppName,
		MacOSVersion:       req.MacOSVersion,
		ServerAPIURL:       req.ServerAPIURL,
		ServerWSHost:       req.ServerWSHost,
		ServerWSPort:       req.ServerWSPort,
		EnterpriseCode:     req.EnterpriseCode,
		PushMiAppID:        req.PushMiAppID,
		PushMiAppKey:       req.PushMiAppKey,
		PushOppoAppKey:     req.PushOppoAppKey,
		PushOppoAppSec:     req.PushOppoAppSec,
		PushVivoAppID:            req.PushVivoAppID,
		PushVivoAppKey:           req.PushVivoAppKey,
		PushHmsAppID:             req.PushHmsAppID,
		AppleTeamID:              req.AppleTeamID,
		AppleCertificateURL:      req.AppleCertificateURL,
		AppleCertificatePassword: req.AppleCertificatePassword,
		AppleProvisioningURL:     req.AppleProvisioningURL,
		AppleMacProvisioningURL:  req.AppleMacProvisioningURL,
		AppleExportMethod:        req.AppleExportMethod,
		GitRepoURL:               req.GitRepoURL,
		GitBranch:                req.GitBranch,
		GitTag:                   req.GitTag,
		GitUsername:              req.GitUsername,
		GitToken:                 req.GitToken,
		Status:                   1,
		CreatedAt:                time.Now(),
		UpdatedAt:                time.Now(),
	}

	if merchant.AndroidVersionCode == 0 {
		merchant.AndroidVersionCode = 1
	}
	if merchant.AndroidVersionName == "" {
		merchant.AndroidVersionName = "1.0.0"
	}
	if merchant.IOSVersion == "" {
		merchant.IOSVersion = "1.0.0"
	}
	if merchant.IOSBuild == "" {
		merchant.IOSBuild = "1"
	}
	if merchant.ServerWSPort == 0 {
		merchant.ServerWSPort = 5100
	}
	if merchant.GitBranch == "" && merchant.GitRepoURL != "" {
		merchant.GitBranch = "main"
	}

	_, err := dbs.DBAdmin.Insert(&merchant)
	if err != nil {
		result.GErr(c, err)
		return
	}

	result.GOK(c, merchant)
}

func updateMerchant(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var req model.BuildMerchantReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.GErr(c, err)
		return
	}

	merchant := model.BuildMerchant{
		MerchantID:         req.MerchantID,
		Name:               req.Name,
		AppName:            req.AppName,
		ShortName:          req.ShortName,
		Description:        req.Description,
		AndroidPackage:     req.AndroidPackage,
		AndroidVersionCode: req.AndroidVersionCode,
		AndroidVersionName: req.AndroidVersionName,
		IOSBundleID:        req.IOSBundleID,
		IOSVersion:         req.IOSVersion,
		IOSBuild:           req.IOSBuild,
		WindowsAppName:     req.WindowsAppName,
		WindowsVersion:     req.WindowsVersion,
		MacOSBundleID:      req.MacOSBundleID,
		MacOSAppName:       req.MacOSAppName,
		MacOSVersion:       req.MacOSVersion,
		ServerAPIURL:       req.ServerAPIURL,
		ServerWSHost:       req.ServerWSHost,
		ServerWSPort:       req.ServerWSPort,
		EnterpriseCode:     req.EnterpriseCode,
		PushMiAppID:        req.PushMiAppID,
		PushMiAppKey:       req.PushMiAppKey,
		PushOppoAppKey:     req.PushOppoAppKey,
		PushOppoAppSec:     req.PushOppoAppSec,
		PushVivoAppID:            req.PushVivoAppID,
		PushVivoAppKey:           req.PushVivoAppKey,
		PushHmsAppID:             req.PushHmsAppID,
		AppleTeamID:              req.AppleTeamID,
		AppleCertificateURL:      req.AppleCertificateURL,
		AppleCertificatePassword: req.AppleCertificatePassword,
		AppleProvisioningURL:     req.AppleProvisioningURL,
		AppleMacProvisioningURL:  req.AppleMacProvisioningURL,
		AppleExportMethod:        req.AppleExportMethod,
		GitRepoURL:               req.GitRepoURL,
		GitBranch:                req.GitBranch,
		GitTag:                   req.GitTag,
		GitUsername:              req.GitUsername,
		GitToken:                 req.GitToken,
		UpdatedAt:                time.Now(),
	}

	_, err := dbs.DBAdmin.ID(id).Update(&merchant)
	if err != nil {
		result.GErr(c, err)
		return
	}

	result.GOK(c, nil)
}

func deleteMerchant(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))

	count, _ := dbs.DBAdmin.Where("build_merchant_id = ? AND status IN (0, 1)", id).Count(&model.BuildTask{})
	if count > 0 {
		result.GResult(c, 400, nil, "存在进行中的构建任务，无法删除")
		return
	}

	_, err := dbs.DBAdmin.ID(id).Delete(&model.BuildMerchant{})
	if err != nil {
		result.GErr(c, err)
		return
	}

	result.GOK(c, nil)
}

func uploadIcon(c *gin.Context) {
	// TODO: 实现图标上传到 OSS
	result.GOK(c, gin.H{"url": ""})
}

// ========== 构建任务 ==========

func listTasks(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	status := c.Query("status")
	merchantID := c.Query("build_merchant_id")

	offset := (page - 1) * size
	var list []model.BuildTask
	session := dbs.DBAdmin.NewSession()
	defer session.Close()

	if status != "" {
		session.Where("status = ?", status)
	}
	if merchantID != "" {
		session.Where("build_merchant_id = ?", merchantID)
	}

	total, err := session.Limit(size, offset).Desc("id").FindAndCount(&list)
	if err != nil {
		result.GErr(c, err)
		return
	}

	result.GOK(c, gin.H{
		"list":  list,
		"total": total,
	})
}

func getTask(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var task model.BuildTask
	has, err := dbs.DBAdmin.ID(id).Get(&task)
	if err != nil {
		result.GErr(c, err)
		return
	}
	if !has {
		result.GResult(c, 404, nil, "任务不存在")
		return
	}

	var artifacts []model.BuildArtifact
	dbs.DBAdmin.Where("task_id = ?", id).Find(&artifacts)

	result.GOK(c, gin.H{
		"task":      task,
		"artifacts": artifacts,
	})
}

func createTask(c *gin.Context) {
	var req model.CreateBuildTaskReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.GErr(c, err)
		return
	}

	var merchant model.BuildMerchant
	has, _ := dbs.DBAdmin.ID(req.BuildMerchantID).Get(&merchant)
	if !has {
		result.GResult(c, 404, nil, "打包配置不存在")
		return
	}

	operator := ""
	if username, exists := c.Get("username"); exists {
		operator = username.(string)
	}

	task := model.BuildTask{
		BuildMerchantID:            req.BuildMerchantID,
		MerchantName:               merchant.Name,
		Platforms:                  req.Platforms,
		Status:                     model.BuildStatusQueued,
		Progress:                   0,
		CurrentStep:                "排队中",
		Operator:                   operator,
		OverrideAndroidVersionCode: req.OverrideAndroidVersionCode,
		OverrideAndroidVersionName: req.OverrideAndroidVersionName,
		OverrideIOSVersion:         req.OverrideIOSVersion,
		OverrideIOSBuild:           req.OverrideIOSBuild,
		CreatedAt:                  time.Now(),
	}

	_, err := dbs.DBAdmin.Insert(&task)
	if err != nil {
		result.GErr(c, err)
		return
	}

	// 将任务推送到构建队列
	if err := enqueueBuildTask(&task); err != nil {
		// 入队失败不影响 API 返回，Worker 会有重试机制
		// 也可以选择返回错误，取决于业务需求
	}

	result.GOK(c, task)
}

// enqueueBuildTask 将任务入队到 Redis
func enqueueBuildTask(task *model.BuildTask) error {
	msg := &buildqueue.BuildTaskMessage{
		ID:                         task.ID,
		Type:                       buildqueue.TaskTypeBuild,
		BuildMerchantID:            task.BuildMerchantID,
		Platforms:                  task.Platforms,
		OverrideAndroidVersionCode: task.OverrideAndroidVersionCode,
		OverrideAndroidVersionName: task.OverrideAndroidVersionName,
		OverrideIOSVersion:         task.OverrideIOSVersion,
		OverrideIOSBuild:           task.OverrideIOSBuild,
		CreatedAt:                  time.Now(),
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	return dbs.Rds().RPush(context.Background(), buildqueue.BuildTaskQueueKey, data).Err()
}

func cancelTask(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))

	var task model.BuildTask
	has, _ := dbs.DBAdmin.ID(id).Get(&task)
	if !has {
		result.GResult(c, 404, nil, "任务不存在")
		return
	}

	if task.Status != model.BuildStatusQueued && task.Status != model.BuildStatusBuilding {
		result.GResult(c, 400, nil, "只能取消排队中或构建中的任务")
		return
	}

	// 标记为已取消（通知 Worker）
	dbs.Rds().SAdd(context.Background(), buildqueue.BuildTaskCancelKey, id)

	now := time.Now()
	task.Status = model.BuildStatusCancelled
	task.FinishedAt = &now
	task.CurrentStep = "已取消"

	_, err := dbs.DBAdmin.ID(id).Cols("status", "finished_at", "current_step").Update(&task)
	if err != nil {
		result.GErr(c, err)
		return
	}

	result.GOK(c, nil)
}

// getTaskProgress 获取任务实时进度
func getTaskProgress(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))

	// 优先从 Redis 获取实时进度
	key := fmt.Sprintf("%d", id)
	data, err := dbs.Rds().HGet(context.Background(), buildqueue.BuildTaskProgressKey, key).Result()

	if err == nil && data != "" {
		var progress buildqueue.BuildProgress
		if json.Unmarshal([]byte(data), &progress) == nil {
			result.GOK(c, progress)
			return
		}
	}

	// Redis 没有则从数据库获取
	var task model.BuildTask
	has, _ := dbs.DBAdmin.ID(id).Get(&task)
	if !has {
		result.GResult(c, 404, nil, "任务不存在")
		return
	}

	result.GOK(c, gin.H{
		"task_id":      task.ID,
		"status":       task.Status,
		"progress":     task.Progress,
		"current_step": task.CurrentStep,
	})
}

func retryTask(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))

	var oldTask model.BuildTask
	has, _ := dbs.DBAdmin.ID(id).Get(&oldTask)
	if !has {
		result.GResult(c, 404, nil, "任务不存在")
		return
	}

	if oldTask.Status != model.BuildStatusFailed && oldTask.Status != model.BuildStatusCancelled {
		result.GResult(c, 400, nil, "只能重试失败或已取消的任务")
		return
	}

	operator := ""
	if username, exists := c.Get("username"); exists {
		operator = username.(string)
	}

	newTask := model.BuildTask{
		BuildMerchantID:            oldTask.BuildMerchantID,
		MerchantName:               oldTask.MerchantName,
		Platforms:                  oldTask.Platforms,
		Status:                     model.BuildStatusQueued,
		Progress:                   0,
		CurrentStep:                "排队中",
		Operator:                   operator,
		OverrideAndroidVersionCode: oldTask.OverrideAndroidVersionCode,
		OverrideAndroidVersionName: oldTask.OverrideAndroidVersionName,
		OverrideIOSVersion:         oldTask.OverrideIOSVersion,
		OverrideIOSBuild:           oldTask.OverrideIOSBuild,
		CreatedAt:                  time.Now(),
	}

	_, err := dbs.DBAdmin.Insert(&newTask)
	if err != nil {
		result.GErr(c, err)
		return
	}

	result.GOK(c, newTask)
}

// ========== 产物管理 ==========

func listArtifacts(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	merchantID := c.Query("build_merchant_id")
	platform := c.Query("platform")

	offset := (page - 1) * size
	var list []model.BuildArtifact
	session := dbs.DBAdmin.NewSession()
	defer session.Close()

	session.Where("is_deleted = 0")
	if merchantID != "" {
		session.Where("build_merchant_id = ?", merchantID)
	}
	if platform != "" {
		session.Where("platform = ?", platform)
	}

	total, err := session.Limit(size, offset).Desc("id").FindAndCount(&list)
	if err != nil {
		result.GErr(c, err)
		return
	}

	result.GOK(c, gin.H{
		"list":  list,
		"total": total,
	})
}

func downloadArtifact(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))

	var artifact model.BuildArtifact
	has, _ := dbs.DBAdmin.ID(id).Get(&artifact)
	if !has {
		result.GResult(c, 404, nil, "产物不存在")
		return
	}

	if artifact.IsDeleted == 1 {
		result.GResult(c, 410, nil, "产物已过期删除")
		return
	}

	if time.Now().After(artifact.ExpiresAt) {
		result.GResult(c, 410, nil, "产物已过期")
		return
	}

	dbs.DBAdmin.ID(id).Incr("download_count").Update(&model.BuildArtifact{})

	c.Redirect(302, artifact.FileURL)
}

func cleanExpiredArtifacts(c *gin.Context) {
	var artifacts []model.BuildArtifact
	dbs.DBAdmin.Where("expires_at < ? AND is_deleted = 0", time.Now()).Find(&artifacts)

	deletedCount := 0
	for _, artifact := range artifacts {
		// TODO: 删除 OSS 上的文件
		dbs.DBAdmin.ID(artifact.ID).Cols("is_deleted").Update(&model.BuildArtifact{IsDeleted: 1})
		deletedCount++
	}

	result.GOK(c, gin.H{
		"deleted_count": deletedCount,
	})
}

// ========== 统计 ==========

func getStats(c *gin.Context) {
	var resp model.BuildStatsResp

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	weekAgo := today.AddDate(0, 0, -7)

	// 今日统计
	total, _ := dbs.DBAdmin.Where("created_at >= ?", today).Count(&model.BuildTask{})
	resp.Today.Total = int(total)
	success, _ := dbs.DBAdmin.Where("created_at >= ? AND status = ?", today, model.BuildStatusSuccess).Count(&model.BuildTask{})
	resp.Today.Success = int(success)
	failed, _ := dbs.DBAdmin.Where("created_at >= ? AND status = ?", today, model.BuildStatusFailed).Count(&model.BuildTask{})
	resp.Today.Failed = int(failed)
	building, _ := dbs.DBAdmin.Where("status IN (0, 1)").Count(&model.BuildTask{})
	resp.Today.Building = int(building)

	if resp.Today.Total > 0 {
		resp.Today.Rate = float64(resp.Today.Success) / float64(resp.Today.Total) * 100
	}

	// 本周统计
	weekTotal, _ := dbs.DBAdmin.Where("created_at >= ?", weekAgo).Count(&model.BuildTask{})
	resp.Week.Total = int(weekTotal)
	weekSuccess, _ := dbs.DBAdmin.Where("created_at >= ? AND status = ?", weekAgo, model.BuildStatusSuccess).Count(&model.BuildTask{})
	resp.Week.Success = int(weekSuccess)
	weekFailed, _ := dbs.DBAdmin.Where("created_at >= ? AND status = ?", weekAgo, model.BuildStatusFailed).Count(&model.BuildTask{})
	resp.Week.Failed = int(weekFailed)

	// 平台统计
	var tasks []model.BuildTask
	dbs.DBAdmin.Where("created_at >= ?", weekAgo).Cols("platforms").Find(&tasks)
	for _, task := range tasks {
		platforms := strings.Split(task.Platforms, ",")
		for _, p := range platforms {
			switch strings.TrimSpace(p) {
			case "android":
				resp.Platforms.Android++
			case "ios":
				resp.Platforms.IOS++
			case "windows":
				resp.Platforms.Windows++
			case "macos":
				resp.Platforms.MacOS++
			}
		}
	}

	result.GOK(c, resp)
}

// ========== 构建服务器 ==========

func listServers(c *gin.Context) {
	var list []model.BuildServer
	err := dbs.DBAdmin.Find(&list)
	if err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, list)
}

func createServer(c *gin.Context) {
	var server model.BuildServer
	if err := c.ShouldBindJSON(&server); err != nil {
		result.GErr(c, err)
		return
	}

	server.Status = 1
	server.CreatedAt = time.Now()
	server.UpdatedAt = time.Now()

	_, err := dbs.DBAdmin.Insert(&server)
	if err != nil {
		result.GErr(c, err)
		return
	}

	result.GOK(c, server)
}

func updateServer(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var server model.BuildServer
	if err := c.ShouldBindJSON(&server); err != nil {
		result.GErr(c, err)
		return
	}

	server.UpdatedAt = time.Now()
	_, err := dbs.DBAdmin.ID(id).Update(&server)
	if err != nil {
		result.GErr(c, err)
		return
	}

	result.GOK(c, nil)
}

func deleteServer(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	_, err := dbs.DBAdmin.ID(id).Delete(&model.BuildServer{})
	if err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, nil)
}
