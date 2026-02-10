package global

import (
	"server/internal/server/middleware"
	"server/internal/server/model"
	globalService "server/internal/server/service/global" // 使用别名避免包名冲突
	"server/pkg/result"
	"strconv"

	"github.com/gin-gonic/gin"
)

// Routes 注册全局管理相关路由
func Routes(gi gin.IRouter) {
	group := gi.Group("global")
	group.Use(middleware.Authorization) // 需要认证

	// OSS URL管理
	group.GET("oss-url", queryOssUrl)         // GET /global/oss-url - 查询列表
	group.POST("oss-url", createOssUrl)       // POST /global/oss-url - 创建
	group.PUT("oss-url/:id", updateOssUrl)    // PUT /global/oss-url/:id - 更新
	group.DELETE("oss-url/:id", deleteOssUrl) // DELETE /global/oss-url/:id - 删除
}

// 查询OSS URL列表
func queryOssUrl(ctx *gin.Context) {
	var req model.QueryOssUrlReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
		result.GParamErr(ctx, err)
		return
	}

	// 设置默认分页参数
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.Size <= 0 {
		req.Size = 10
	}

	data, err := globalService.QueryOssUrl(req)
	if err != nil {
		result.GErr(ctx, err)
		return
	}

	result.GOK(ctx, data)
}

// 创建OSS URL
func createOssUrl(ctx *gin.Context) {
	var req model.CreateOssUrlReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		result.GParamErr(ctx, err)
		return
	}

	id, err := globalService.CreateOssUrl(req)
	if err != nil {
		result.GErr(ctx, err)
		return
	}

	result.GOK(ctx, gin.H{"id": id})
}

// 更新OSS URL
func updateOssUrl(ctx *gin.Context) {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		result.GParamErr(ctx, err)
		return
	}

	var req model.UpdateOssUrlReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		result.GParamErr(ctx, err)
		return
	}

	err = globalService.UpdateOssUrl(id, req)
	if err != nil {
		result.GErr(ctx, err)
		return
	}

	result.GOK(ctx, nil)
}

// 删除OSS URL
func deleteOssUrl(ctx *gin.Context) {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		result.GParamErr(ctx, err)
		return
	}

	err = globalService.DeleteOssUrl(id)
	if err != nil {
		result.GErr(ctx, err)
		return
	}

	result.GOK(ctx, nil)
}
