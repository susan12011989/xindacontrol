package deploy

import (
	"fmt"
	"net/http"
	"server/internal/server/middleware"
	"server/internal/server/model"
	deployService "server/internal/server/service/deploy"
	"server/pkg/gostapi"
	"server/pkg/result"
	"server/pkg/token_manager"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/zeromicro/go-zero/core/logx"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// Routes 部署管理路由注册
func Routes(gi gin.IRouter) {
	group := gi.Group("deploy")

	// WebSSH 单独处理认证（WebSocket需要特殊处理）
	group.GET("webssh", handleWebSSH)

	// 其他接口需要认证
	group.Use(middleware.Authorization)

	// 服务器管理
	group.GET("servers", listServers)
	group.GET("servers/:id", getServer)
	group.POST("servers", createServer)
	group.PUT("servers/:id", updateServer)
	group.DELETE("servers/:id", deleteServer)
	group.POST("servers/test", testConnection)
	group.POST("servers/:id/toggle-status", toggleStatus)

	// 服务操作（systemctl 管理 server/wukongim/gost）
	group.POST("service/action", serviceAction)   // 启动/停止/重启服务
	group.GET("service/status", getServiceStatus) // 获取服务状态
	group.GET("service/logs", getServiceLogs)     // 获取服务日志

	// 服务器资源
	group.GET("server-stats", getServerStats)
	group.POST("server-stats/batch", getServerStatsBatch)

	// Docker 容器状态
	group.GET("docker/containers", getDockerContainers)

	// 文件上传（仅 server 和 wukongim）
	group.POST("upload", uploadToServer)

	// 上传到本地（用于批量分发）
	group.POST("upload-local", uploadToLocal)

	// 批量分发（从本地分发到目标服务器）
	group.POST("distribute", distributeFile)

	// 配置文件
	group.GET("config", getConfigFile)
	group.POST("config", updateConfigFile)

	// GOST API 代理
	group.GET("gost/services", listGostServices)
	group.GET("gost/services/:name", getGostService)
	group.PUT("gost/services/:name", updateGostService)
	group.POST("gost/services", createGostService)
	group.DELETE("gost/services/:name", deleteGostService)
	group.GET("gost/chains", listGostChains)

	// GOST 服务器一键部署
	group.POST("gost/deploy", deployGostServer)          // 一键部署 GOST 转发服务器（流式API）
	group.GET("gost/deploy/config", getGostDeployConfig) // 获取部署默认配置
	group.POST("gost/install", installGostToServer)      // 在已有服务器上安装 GOST（流式API）

	// 一键部署 TSDD 服务
	group.POST("tsdd/deploy", deployTSDD)              // 部署到已注册服务器 (Docker方式)
	group.POST("tsdd/deploy-by-ip", deployTSDDByIP)    // 通过IP部署（新服务器，Docker方式）
	group.POST("tsdd/deploy-ami", deployTSDDWithAMI)   // 使用 AMI 部署（推荐）
	group.GET("tsdd/status", getDeployStatus)          // 获取部署状态

	// ========== 批量运维操作 ==========
	group.POST("batch/service-action", batchServiceAction)  // 批量服务操作（start/stop/restart）
	group.POST("batch/health-check", batchHealthCheck)      // 批量健康检查
	group.POST("batch/command", batchCommand)               // 批量执行命令

	// ========== 日志管理 ==========
	group.POST("logs/query", queryLogs)   // 统一日志查询

	// ========== 版本管理 ==========
	group.GET("versions", listVersions)                    // 版本列表
	group.POST("versions/upload", uploadVersion)           // 上传新版本
	group.DELETE("versions/:id", deleteVersion)            // 删除版本
	group.POST("versions/:id/set-current", setCurrentVersion) // 设为当前版本
	group.POST("versions/deploy", deployVersion)           // 部署版本到服务器
	group.POST("versions/rollback", rollbackVersion)       // 回滚版本
	group.GET("deployment-history", getDeploymentHistory)  // 部署历史
}

// listServers 查询服务器列表
func listServers(ctx *gin.Context) {
	var req model.QueryServersReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
		result.GParamErr(ctx, err)
		return
	}

	if req.Page == 0 {
		req.Page = 1
	}
	if req.Size == 0 {
		req.Size = 10
	}

	data, err := deployService.QueryServers(req)
	if err != nil {
		result.GErr(ctx, err)
		return
	}

	result.GOK(ctx, data)
}

// getServer 获取服务器详情
func getServer(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		result.GParamErr(ctx, err)
		return
	}

	data, err := deployService.GetServerDetail(id)
	if err != nil {
		result.GErr(ctx, err)
		return
	}

	result.GOK(ctx, data)
}

// createServer 创建服务器
func createServer(ctx *gin.Context) {
	var req model.CreateServerReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		result.GParamErr(ctx, err)
		return
	}

	id, err := deployService.CreateServer(req)
	if err != nil {
		result.GErr(ctx, err)
		return
	}

	result.GOK(ctx, gin.H{"id": id})
}

// updateServer 更新服务器
func updateServer(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		result.GParamErr(ctx, err)
		return
	}

	var req model.UpdateServerReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		result.GParamErr(ctx, err)
		return
	}

	err = deployService.UpdateServer(id, req)
	if err != nil {
		result.GErr(ctx, err)
		return
	}

	result.GOK(ctx, nil)
}

// deleteServer 删除服务器
func deleteServer(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		result.GParamErr(ctx, err)
		return
	}

	err = deployService.DeleteServer(id)
	if err != nil {
		result.GErr(ctx, err)
		return
	}

	result.GOK(ctx, nil)
}

// testConnection 测试SSH连接
func testConnection(ctx *gin.Context) {
	var req model.TestConnectionReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		result.GParamErr(ctx, err)
		return
	}

	err := deployService.TestConnection(req)
	if err != nil {
		result.GErr(ctx, err)
		return
	}

	result.GOK(ctx, gin.H{"message": "连接测试成功"})
}

// toggleStatus 切换服务器启用/禁用状态
func toggleStatus(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		result.GParamErr(ctx, err)
		return
	}

	err = deployService.ToggleServerStatus(id)
	if err != nil {
		result.GErr(ctx, err)
		return
	}

	result.GOK(ctx, gin.H{"message": "状态切换成功"})
}

// serviceAction 服务操作（systemctl start/stop/restart）
func serviceAction(ctx *gin.Context) {
	var req model.ServiceActionReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		result.GParamErr(ctx, err)
		return
	}

	operator := middleware.GetUsername(ctx)
	if operator == "" {
		operator = "admin"
	}

	data, err := deployService.ServiceAction(req, operator)
	if err != nil {
		result.GErr(ctx, err)
		return
	}

	result.GOK(ctx, data)
}

// getServiceStatus 获取服务状态
func getServiceStatus(ctx *gin.Context) {
	var req model.ServiceStatusReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
		result.GParamErr(ctx, err)
		return
	}

	data, err := deployService.GetServiceStatus(req)
	if err != nil {
		result.GErr(ctx, err)
		return
	}

	result.GOK(ctx, data)
}

// getServiceLogs 获取服务日志
func getServiceLogs(ctx *gin.Context) {
	var req model.ServiceLogsReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
		result.GParamErr(ctx, err)
		return
	}

	data, err := deployService.GetServiceLogs(req)
	if err != nil {
		result.GErr(ctx, err)
		return
	}

	result.GOK(ctx, data)
}

// getServerStats 获取服务器资源
func getServerStats(ctx *gin.Context) {
	var req model.GetServerStatsReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
		result.GParamErr(ctx, err)
		return
	}

	data, err := deployService.GetServerStats(req)
	if err != nil {
		result.GErr(ctx, err)
		return
	}

	result.GOK(ctx, data)
}

// getServerStatsBatch 批量获取服务器资源
func getServerStatsBatch(ctx *gin.Context) {
	var req model.GetServerStatsBatchReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		result.GParamErr(ctx, err)
		return
	}

	data, err := deployService.GetServerStatsBatch(req)
	if err != nil {
		result.GErr(ctx, err)
		return
	}

	result.GOK(ctx, data)
}

// getDockerContainers 获取 Docker 容器状态
func getDockerContainers(ctx *gin.Context) {
	var req model.DockerContainersReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
		result.GParamErr(ctx, err)
		return
	}

	data, err := deployService.GetDockerContainers(req.ServerId)
	if err != nil {
		result.GErr(ctx, err)
		return
	}

	result.GOK(ctx, data)
}

// uploadToServer 上传文件到服务器（仅支持 server 和 wukongim）
func uploadToServer(ctx *gin.Context) {
	serverIDStr := ctx.PostForm("server_id")
	if serverIDStr == "" {
		result.GParamErr(ctx, fmt.Errorf("server_id 不能为空"))
		return
	}
	serverID, err := strconv.Atoi(serverIDStr)
	if err != nil {
		result.GParamErr(ctx, err)
		return
	}

	// 服务名：server 或 wukongim
	serviceName := ctx.PostForm("service_name")
	if serviceName != "server" && serviceName != "wukongim" {
		result.GParamErr(ctx, fmt.Errorf("仅支持上传 server 或 wukongim"))
		return
	}

	// 获取文件
	file, header, err := ctx.Request.FormFile("file")
	if err != nil {
		result.GParamErr(ctx, fmt.Errorf("读取文件失败: %v", err))
		return
	}
	defer file.Close()

	// 上传
	uploadedPath, err := deployService.UploadServiceFile(serverID, serviceName, header.Filename, file)
	if err != nil {
		result.GErr(ctx, err)
		return
	}

	result.GOK(ctx, gin.H{
		"message":      "上传成功",
		"remote_path":  uploadedPath,
		"service_name": serviceName,
	})
}

// uploadToLocal 上传文件到本地（用于批量分发）
func uploadToLocal(ctx *gin.Context) {
	// 服务名：server 或 wukongim
	serviceName := ctx.PostForm("service_name")
	if serviceName != "server" && serviceName != "wukongim" {
		result.GParamErr(ctx, fmt.Errorf("仅支持上传 server 或 wukongim"))
		return
	}

	// 获取文件
	file, header, err := ctx.Request.FormFile("file")
	if err != nil {
		result.GParamErr(ctx, fmt.Errorf("读取文件失败: %v", err))
		return
	}
	defer file.Close()

	// 上传到本地
	uploadedPath, err := deployService.UploadToLocal(serviceName, header.Filename, file)
	if err != nil {
		result.GErr(ctx, err)
		return
	}

	result.GOK(ctx, gin.H{
		"message":      "上传成功",
		"local_path":   uploadedPath,
		"service_name": serviceName,
	})
}

// distributeFile 批量分发文件
func distributeFile(ctx *gin.Context) {
	var req model.DistributeFileReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		result.GParamErr(ctx, err)
		return
	}

	// 验证服务名
	if req.ServiceName != "server" && req.ServiceName != "wukongim" {
		result.GParamErr(ctx, fmt.Errorf("仅支持分发 server 或 wukongim"))
		return
	}

	if len(req.TargetServerIds) == 0 {
		result.GParamErr(ctx, fmt.Errorf("目标服务器列表不能为空"))
		return
	}

	operator := middleware.GetUsername(ctx)
	if operator == "" {
		operator = "admin"
	}

	data, err := deployService.DistributeFile(req, operator)
	if err != nil {
		result.GErr(ctx, err)
		return
	}

	result.GOK(ctx, data)
}

// getConfigFile 获取配置文件内容
func getConfigFile(ctx *gin.Context) {
	var req model.GetConfigFileReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
		result.GParamErr(ctx, err)
		return
	}
	data, err := deployService.GetConfigFile(req.ServerId, req.ServiceName)
	if err != nil {
		result.GErr(ctx, err)
		return
	}
	result.GOK(ctx, data)
}

// updateConfigFile 更新配置文件内容
func updateConfigFile(ctx *gin.Context) {
	var req model.UpdateConfigFileReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		result.GParamErr(ctx, err)
		return
	}
	data, err := deployService.UpdateConfigFile(req.ServerId, req.ServiceName, req.Content)
	if err != nil {
		result.GErr(ctx, err)
		return
	}
	result.GOK(ctx, data)
}

// handleWebSSH WebSocket SSH连接处理
func handleWebSSH(ctx *gin.Context) {
	logx.Info("WebSSH: 开始处理连接请求")

	tokenStr := ctx.Query("token")
	if tokenStr == "" {
		authHeader := ctx.GetHeader("Authorization")
		if authHeader != "" {
			parts := strings.Split(authHeader, " ")
			if len(parts) == 2 && parts[0] == "Bearer" {
				tokenStr = parts[1]
			}
		}
	}

	if tokenStr == "" {
		logx.Error("WebSSH: token为空")
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "未授权"})
		return
	}

	tokenInfo, err := token_manager.ValidateToken(tokenStr)
	if err != nil {
		logx.Errorf("WebSSH: token验证失败: %v", err)
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "token无效"})
		return
	}

	logx.Infof("WebSSH: token验证成功, 用户: %s", tokenInfo.Username)

	ctx.Set("user_id", tokenInfo.UserID)
	ctx.Set("username", tokenInfo.Username)

	serverIdStr := ctx.Query("server_id")
	if serverIdStr == "" {
		logx.Error("WebSSH: server_id为空")
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "server_id参数不能为空"})
		return
	}

	serverId, err := strconv.Atoi(serverIdStr)
	if err != nil {
		logx.Errorf("WebSSH: server_id解析失败: %v", err)
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "server_id参数无效"})
		return
	}

	logx.Infof("WebSSH: 准备升级WebSocket, server_id=%d", serverId)

	ws, err := upgrader.Upgrade(ctx.Writer, ctx.Request, nil)
	if err != nil {
		logx.Errorf("WebSSH: WebSocket升级失败: %v", err)
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "WebSocket升级失败"})
		return
	}
	defer ws.Close()

	logx.Info("WebSSH: WebSocket升级成功，开始处理SSH会话")

	if err := deployService.HandleWebSSH(ws, serverId); err != nil {
		logx.Errorf("WebSSH: 处理会话失败: %v", err)
		ws.WriteJSON(map[string]interface{}{
			"type": "error",
			"data": err.Error(),
		})
	}

	logx.Info("WebSSH: 会话结束")
}

// ========== GOST API 代理 ==========

// listGostServices 获取 GOST 服务列表
func listGostServices(ctx *gin.Context) {
	serverID, _ := strconv.Atoi(ctx.Query("server_id"))
	if serverID == 0 {
		result.GParamErr(ctx, fmt.Errorf("server_id 不能为空"))
		return
	}
	page, _ := strconv.Atoi(ctx.Query("page"))
	size, _ := strconv.Atoi(ctx.Query("size"))
	port, _ := strconv.Atoi(ctx.Query("port"))

	data, err := deployService.ListGostServices(serverID, page, size, port)
	if err != nil {
		result.GErr(ctx, err)
		return
	}
	result.GOK(ctx, data)
}

// getGostService 获取单个 GOST 服务详情
func getGostService(ctx *gin.Context) {
	serverID, _ := strconv.Atoi(ctx.Query("server_id"))
	if serverID == 0 {
		result.GParamErr(ctx, fmt.Errorf("server_id 不能为空"))
		return
	}
	name := ctx.Param("name")
	if name == "" {
		result.GParamErr(ctx, fmt.Errorf("service_name 不能为空"))
		return
	}
	data, err := deployService.GetGostServiceDetail(serverID, name)
	if err != nil {
		result.GErr(ctx, err)
		return
	}
	result.GOK(ctx, data)
}

// updateGostService 更新 GOST 服务配置
func updateGostService(ctx *gin.Context) {
	serverID, _ := strconv.Atoi(ctx.Query("server_id"))
	if serverID == 0 {
		result.GParamErr(ctx, fmt.Errorf("server_id 不能为空"))
		return
	}
	name := ctx.Param("name")
	if name == "" {
		result.GParamErr(ctx, fmt.Errorf("service_name 不能为空"))
		return
	}

	// 直接读取 JSON body 并转换为 ServiceConfig
	var cfg gostapi.ServiceConfig
	if err := ctx.ShouldBindJSON(&cfg); err != nil {
		result.GParamErr(ctx, err)
		return
	}

	data, err := deployService.UpdateGostServiceDetail(serverID, name, &cfg)
	if err != nil {
		result.GErr(ctx, err)
		return
	}
	result.GOK(ctx, data)
}

// createGostService 创建 GOST 服务
func createGostService(ctx *gin.Context) {
	var req model.CreateGostServiceReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		result.GParamErr(ctx, err)
		return
	}

	name, err := deployService.CreateGostServiceAPI(req.ServerId, req.ListenPort, req.ForwardHost, req.ForwardPort)
	if err != nil {
		result.GErr(ctx, err)
		return
	}
	result.GOK(ctx, gin.H{"name": name, "message": "创建成功"})
}

// deleteGostService 删除 GOST 服务
func deleteGostService(ctx *gin.Context) {
	serverID, _ := strconv.Atoi(ctx.Query("server_id"))
	if serverID == 0 {
		result.GParamErr(ctx, fmt.Errorf("server_id 不能为空"))
		return
	}
	name := ctx.Param("name")
	if name == "" {
		result.GParamErr(ctx, fmt.Errorf("service_name 不能为空"))
		return
	}

	_, err := deployService.DeleteGostServiceAPI(serverID, name)
	if err != nil {
		result.GErr(ctx, err)
		return
	}
	result.GOK(ctx, gin.H{"message": "删除成功"})
}

// listGostChains 获取 GOST Chain 列表
func listGostChains(ctx *gin.Context) {
	serverID, _ := strconv.Atoi(ctx.Query("server_id"))
	if serverID == 0 {
		result.GParamErr(ctx, fmt.Errorf("server_id 不能为空"))
		return
	}

	data, err := deployService.ListGostChains(serverID)
	if err != nil {
		result.GErr(ctx, err)
		return
	}
	result.GOK(ctx, data)
}

// ========== 一键部署 TSDD 服务 ==========

// deployTSDD 部署 TSDD 到已注册服务器
func deployTSDD(ctx *gin.Context) {
	var req model.DeployTSDDReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		result.GParamErr(ctx, err)
		return
	}

	operator := middleware.GetUsername(ctx)
	if operator == "" {
		operator = "admin"
	}

	data, err := deployService.DeployTSDD(req, operator)
	if err != nil {
		result.GErr(ctx, err)
		return
	}

	result.GOK(ctx, data)
}

// deployTSDDByIP 通过IP部署 TSDD（新服务器，未注册到系统）
func deployTSDDByIP(ctx *gin.Context) {
	var req model.DeployTSDDByIPReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		result.GParamErr(ctx, err)
		return
	}

	operator := middleware.GetUsername(ctx)
	if operator == "" {
		operator = "admin"
	}

	data, err := deployService.DeployTSDDByIP(req, operator)
	if err != nil {
		result.GErr(ctx, err)
		return
	}

	result.GOK(ctx, data)
}

// getDeployStatus 获取服务器部署状态
func getDeployStatus(ctx *gin.Context) {
	var req model.GetDeployStatusReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
		result.GParamErr(ctx, err)
		return
	}

	data, err := deployService.GetDeployStatus(req)
	if err != nil {
		result.GErr(ctx, err)
		return
	}

	result.GOK(ctx, data)
}

// deployTSDDWithAMI 使用 AMI 部署 TSDD（推荐方式）
func deployTSDDWithAMI(ctx *gin.Context) {
	var req model.DeployTSDDWithAMIReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		result.GParamErr(ctx, err)
		return
	}

	operator := middleware.GetUsername(ctx)
	if operator == "" {
		operator = "admin"
	}

	data, err := deployService.DeployTSDDWithAMI(req, operator)
	if err != nil {
		result.GErr(ctx, err)
		return
	}

	result.GOK(ctx, data)
}

// ========== GOST 服务器一键部署 ==========

// deployGostServer 一键部署 GOST 转发服务器（流式API）
func deployGostServer(ctx *gin.Context) {
	var req model.DeployGostServerReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		result.GStreamEnd(ctx, true, err.Error())
		return
	}

	// 验证必填参数
	if req.CloudAccountId == 0 {
		result.GStreamEnd(ctx, true, "cloud_account_id 不能为空")
		return
	}
	if req.RegionId == "" {
		result.GStreamEnd(ctx, true, "region_id 不能为空")
		return
	}

	// 流式响应
	result.GStream(ctx)

	config := &deployService.GostDeployConfig{
		CloudAccountId: req.CloudAccountId,
		RegionId:       req.RegionId,
		InstanceType:   req.InstanceType,
		ImageId:        req.ImageId,
		ServerName:     req.ServerName,
		GroupId:        req.GroupId,
		Password:       req.Password,
		Bandwidth:      req.Bandwidth,
	}

	deployResult, err := deployService.DeployGostServer(config, func(message string) {
		result.GStreamData(ctx, gin.H{
			"message": fmt.Sprintf("%s %s", time.Now().Format(time.DateTime), message),
		})
	})

	if err != nil {
		result.GStreamEnd(ctx, true, err.Error())
		return
	}

	result.GStreamEnd(ctx, true, fmt.Sprintf("部署成功! 服务器ID: %d, IP: %s", deployResult.ServerId, deployResult.PublicIP))
}

// getGostDeployConfig 获取 GOST 部署默认配置
func getGostDeployConfig(ctx *gin.Context) {
	regionId := ctx.Query("region_id")
	if regionId == "" {
		regionId = "ap-southeast-1"
	}

	config := deployService.GetGostDefaultConfig(regionId)
	result.GOK(ctx, config)
}

// installGostToServer 在已有服务器上安装 GOST（流式API）
func installGostToServer(ctx *gin.Context) {
	var req model.InstallGostReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		result.GStreamEnd(ctx, true, err.Error())
		return
	}

	// 验证必填参数
	if req.ServerId == 0 && req.Host == "" {
		result.GStreamEnd(ctx, true, "server_id 或 host 必须提供一个")
		return
	}

	// 流式响应
	result.GStream(ctx)

	err := deployService.InstallGostToExistingServer(&req, func(message string) {
		result.GStreamData(ctx, gin.H{
			"message": message,
		})
	})

	if err != nil {
		result.GStreamEnd(ctx, true, err.Error())
		return
	}

	result.GStreamEnd(ctx, true, "GOST 安装成功!")
}

// ========== 批量运维操作 ==========

// batchServiceAction 批量服务操作
func batchServiceAction(ctx *gin.Context) {
	var req model.BatchServiceActionReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		result.GParamErr(ctx, err)
		return
	}

	operator := middleware.GetUsername(ctx)
	if operator == "" {
		operator = "admin"
	}

	data, err := deployService.BatchServiceAction(req, operator)
	if err != nil {
		result.GErr(ctx, err)
		return
	}

	result.GOK(ctx, data)
}

// batchHealthCheck 批量健康检查
func batchHealthCheck(ctx *gin.Context) {
	var req model.BatchHealthCheckReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		result.GParamErr(ctx, err)
		return
	}

	data, err := deployService.BatchHealthCheck(req)
	if err != nil {
		result.GErr(ctx, err)
		return
	}

	result.GOK(ctx, data)
}

// batchCommand 批量执行命令
func batchCommand(ctx *gin.Context) {
	var req model.BatchCommandReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		result.GParamErr(ctx, err)
		return
	}

	operator := middleware.GetUsername(ctx)
	if operator == "" {
		operator = "admin"
	}

	data, err := deployService.BatchCommand(req, operator)
	if err != nil {
		result.GErr(ctx, err)
		return
	}

	result.GOK(ctx, data)
}

// ========== 日志管理 ==========

// queryLogs 统一日志查询
func queryLogs(ctx *gin.Context) {
	var req model.LogQueryReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		result.GParamErr(ctx, err)
		return
	}

	data, err := deployService.QueryLogs(req)
	if err != nil {
		result.GErr(ctx, err)
		return
	}

	result.GOK(ctx, data)
}

// ========== 版本管理 ==========

// listVersions 获取版本列表
func listVersions(ctx *gin.Context) {
	var req model.ListVersionsReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
		result.GParamErr(ctx, err)
		return
	}

	data, err := deployService.ListVersions(req)
	if err != nil {
		result.GErr(ctx, err)
		return
	}

	result.GOK(ctx, data)
}

// uploadVersion 上传新版本
func uploadVersion(ctx *gin.Context) {
	serviceName := ctx.PostForm("service_name")
	if serviceName != "server" && serviceName != "wukongim" {
		result.GParamErr(ctx, fmt.Errorf("仅支持上传 server 或 wukongim"))
		return
	}

	version := ctx.PostForm("version")
	if version == "" {
		result.GParamErr(ctx, fmt.Errorf("version 不能为空"))
		return
	}

	changelog := ctx.PostForm("changelog")

	file, _, err := ctx.Request.FormFile("file")
	if err != nil {
		result.GParamErr(ctx, fmt.Errorf("读取文件失败: %v", err))
		return
	}
	defer file.Close()

	operator := middleware.GetUsername(ctx)
	if operator == "" {
		operator = "admin"
	}

	data, err := deployService.UploadVersion(serviceName, version, changelog, operator, file)
	if err != nil {
		result.GErr(ctx, err)
		return
	}

	result.GOK(ctx, data)
}

// deleteVersion 删除版本
func deleteVersion(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		result.GParamErr(ctx, err)
		return
	}

	err = deployService.DeleteVersion(id)
	if err != nil {
		result.GErr(ctx, err)
		return
	}

	result.GOK(ctx, gin.H{"message": "删除成功"})
}

// setCurrentVersion 设置当前版本
func setCurrentVersion(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		result.GParamErr(ctx, err)
		return
	}

	err = deployService.SetCurrentVersion(id)
	if err != nil {
		result.GErr(ctx, err)
		return
	}

	result.GOK(ctx, gin.H{"message": "设置成功"})
}

// deployVersion 部署版本到服务器
func deployVersion(ctx *gin.Context) {
	var req model.DeployVersionReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		result.GParamErr(ctx, err)
		return
	}

	operator := middleware.GetUsername(ctx)
	if operator == "" {
		operator = "admin"
	}

	data, err := deployService.DeployVersion(req, operator)
	if err != nil {
		result.GErr(ctx, err)
		return
	}

	result.GOK(ctx, data)
}

// rollbackVersion 回滚版本
func rollbackVersion(ctx *gin.Context) {
	var req model.RollbackReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		result.GParamErr(ctx, err)
		return
	}

	operator := middleware.GetUsername(ctx)
	if operator == "" {
		operator = "admin"
	}

	data, err := deployService.RollbackVersion(req, operator)
	if err != nil {
		result.GErr(ctx, err)
		return
	}

	result.GOK(ctx, data)
}

// getDeploymentHistory 获取部署历史
func getDeploymentHistory(ctx *gin.Context) {
	var req model.DeploymentHistoryReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
		result.GParamErr(ctx, err)
		return
	}

	data, err := deployService.GetDeploymentHistory(req)
	if err != nil {
		result.GErr(ctx, err)
		return
	}

	result.GOK(ctx, data)
}
