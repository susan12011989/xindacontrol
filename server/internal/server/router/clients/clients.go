package clients

import (
	"server/internal/dbhelper"
	"server/internal/server/middleware"
	"server/internal/server/model"
	"server/pkg/dbs"
	"server/pkg/entity"
	"server/pkg/result"
	"strconv"

	"github.com/gin-gonic/gin"
)

// Routes 注册客户端管理路由
func Routes(gi gin.IRouter) {
	group := gi.Group("clients")
	group.Use(middleware.Authorization)
	group.GET("", listClients)
	group.GET(":id", getClient)
	group.POST("", createClient)
	group.PUT(":id", updateClient)
	group.DELETE(":id", deleteClient)
}

// listClients 获取客户端列表
func listClients(c *gin.Context) {
	var req model.QueryClientsReq
	if err := c.ShouldBindQuery(&req); err != nil {
		result.GParamErr(c, err)
		return
	}

	// 默认分页参数
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.Size <= 0 {
		req.Size = 10
	}

	// 构建查询条件
	condMap := make(map[string]interface{})
	likeFields := make([]string, 0)

	if req.AppPackageName != "" {
		condMap["app_package_name"] = req.AppPackageName
		likeFields = append(likeFields, "app_package_name")
	}
	if req.AppName != "" {
		condMap["app_name"] = req.AppName
		likeFields = append(likeFields, "app_name")
	}

	var clients []entity.Clients
	total, err := dbhelper.FindWithPaginationAndMultiConditions(
		dbs.DBAdmin,
		"clients",
		req.Page,
		req.Size,
		&clients,
		condMap,
		likeFields,
		"id DESC",
	)
	if err != nil {
		result.GErr(c, err)
		return
	}

	// 转换为响应格式
	list := make([]model.ClientResp, 0, len(clients))
	for _, client := range clients {
		list = append(list, model.ClientResp{
			Id:             client.Id,
			AppPackageName: client.AppPackageName,
			AppName:        client.AppName,
			SmsConfig:      client.SmsConfig,
			PushConfig:     client.PushConfig,
			TrtcConfig:     client.TrtcConfig,
		})
	}

	result.GOK(c, model.QueryClientsResponse{
		List:  list,
		Total: int(total),
	})
}

// getClient 获取单个客户端详情
func getClient(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		result.GResult(c, 400, nil, "无效的ID")
		return
	}

	var client entity.Clients
	ok, err := dbs.DBAdmin.ID(id).Get(&client)
	if err != nil {
		result.GErr(c, err)
		return
	}
	if !ok {
		result.GResult(c, 404, nil, "客户端不存在")
		return
	}

	result.GOK(c, model.ClientResp{
		Id:             client.Id,
		AppPackageName: client.AppPackageName,
		AppName:        client.AppName,
		SmsConfig:      client.SmsConfig,
		PushConfig:     client.PushConfig,
		TrtcConfig:     client.TrtcConfig,
	})
}

// createClient 创建客户端
func createClient(c *gin.Context) {
	var req model.CreateClientReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.GParamErr(c, err)
		return
	}

	// 检查包名是否已存在
	exist, err := dbs.DBAdmin.Where("app_package_name = ?", req.AppPackageName).Exist(&entity.Clients{})
	if err != nil {
		result.GErr(c, err)
		return
	}
	if exist {
		result.GResult(c, 400, nil, "包名已存在")
		return
	}

	client := &entity.Clients{
		AppPackageName: req.AppPackageName,
		AppName:        req.AppName,
		SmsConfig:      req.SmsConfig,
		PushConfig:     req.PushConfig,
		TrtcConfig:     req.TrtcConfig,
	}

	_, err = dbs.DBAdmin.Insert(client)
	if err != nil {
		result.GErr(c, err)
		return
	}

	result.GOK(c, model.ClientResp{
		Id:             client.Id,
		AppPackageName: client.AppPackageName,
		AppName:        client.AppName,
		SmsConfig:      client.SmsConfig,
		PushConfig:     client.PushConfig,
		TrtcConfig:     client.TrtcConfig,
	})
}

// updateClient 更新客户端
func updateClient(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		result.GResult(c, 400, nil, "无效的ID")
		return
	}

	var req model.UpdateClientReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.GParamErr(c, err)
		return
	}

	// 检查客户端是否存在
	var client entity.Clients
	ok, err := dbs.DBAdmin.ID(id).Get(&client)
	if err != nil {
		result.GErr(c, err)
		return
	}
	if !ok {
		result.GResult(c, 404, nil, "客户端不存在")
		return
	}

	// 如果更新包名，检查是否与其他记录冲突
	if req.AppPackageName != "" && req.AppPackageName != client.AppPackageName {
		exist, err := dbs.DBAdmin.Where("app_package_name = ? AND id != ?", req.AppPackageName, id).Exist(&entity.Clients{})
		if err != nil {
			result.GErr(c, err)
			return
		}
		if exist {
			result.GResult(c, 400, nil, "包名已存在")
			return
		}
		client.AppPackageName = req.AppPackageName
	}

	// 更新字段
	if req.AppName != "" {
		client.AppName = req.AppName
	}
	if req.SmsConfig != nil {
		client.SmsConfig = req.SmsConfig
	}
	if req.PushConfig != nil {
		client.PushConfig = req.PushConfig
	}
	if req.TrtcConfig != nil {
		client.TrtcConfig = req.TrtcConfig
	}

	_, err = dbs.DBAdmin.ID(id).AllCols().Update(&client)
	if err != nil {
		result.GErr(c, err)
		return
	}

	result.GOK(c, model.ClientResp{
		Id:             client.Id,
		AppPackageName: client.AppPackageName,
		AppName:        client.AppName,
		SmsConfig:      client.SmsConfig,
		PushConfig:     client.PushConfig,
		TrtcConfig:     client.TrtcConfig,
	})
}

// deleteClient 删除客户端
func deleteClient(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		result.GResult(c, 400, nil, "无效的ID")
		return
	}

	// 检查客户端是否存在
	exist, err := dbs.DBAdmin.ID(id).Exist(&entity.Clients{})
	if err != nil {
		result.GErr(c, err)
		return
	}
	if !exist {
		result.GResult(c, 404, nil, "客户端不存在")
		return
	}

	_, err = dbs.DBAdmin.ID(id).Delete(&entity.Clients{})
	if err != nil {
		result.GErr(c, err)
		return
	}

	result.GOK(c, nil)
}
