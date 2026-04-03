package cloud_account

import (
	"server/internal/server/middleware"
	"server/internal/server/model"
	cloudAccountService "server/internal/server/service/cloud_account"
	"server/pkg/result"
	"strconv"

	"github.com/gin-gonic/gin"
)

// Routes 注册云账号相关路由
func Routes(gi gin.IRouter) {
	group := gi.Group("cloud_account")
	group.Use(middleware.Authorization) // 需要认证

	group.GET("", queryCloudAccounts)            // GET /cloud_account - 查询云账号列表
	group.GET(":id", getCloudAccountDetail)      // GET /cloud_account/:id - 获取云账号详情
	group.POST("", createCloudAccount)           // POST /cloud_account - 创建云账号
	group.PUT(":id", updateCloudAccount)         // PUT /cloud_account/:id - 更新云账号
	group.DELETE(":id", deleteCloudAccount)      // DELETE /cloud_account/:id - 删除云账号
	group.GET("options", getCloudAccountOptions)       // GET /cloud_account/options - 获取云账号选项（下拉框）
	group.GET("batch_balance", batchQueryBalance)       // GET /cloud_account/batch_balance - 批量查询余额
}

// 查询云账号列表
func queryCloudAccounts(ctx *gin.Context) {
	var req model.QueryCloudAccountsReq
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

	data, err := cloudAccountService.QueryCloudAccounts(req)
	if err != nil {
		result.GErr(ctx, err)
		return
	}

	result.GOK(ctx, data)
}

// 获取云账号详情
func getCloudAccountDetail(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		result.GParamErr(ctx, err)
		return
	}

	data, err := cloudAccountService.GetCloudAccountDetail(id)
	if err != nil {
		result.GErr(ctx, err)
		return
	}

	result.GOK(ctx, data)
}

// 创建云账号
func createCloudAccount(ctx *gin.Context) {
	var req model.CreateCloudAccountReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		result.GParamErr(ctx, err)
		return
	}

	id, err := cloudAccountService.CreateCloudAccount(req)
	if err != nil {
		result.GErr(ctx, err)
		return
	}

	result.GOK(ctx, gin.H{"id": id})
}

// 更新云账号
func updateCloudAccount(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		result.GParamErr(ctx, err)
		return
	}

	var req model.UpdateCloudAccountReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		result.GParamErr(ctx, err)
		return
	}

	err = cloudAccountService.UpdateCloudAccount(id, req)
	if err != nil {
		result.GErr(ctx, err)
		return
	}

	result.GOK(ctx, nil)
}

// 删除云账号
func deleteCloudAccount(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		result.GParamErr(ctx, err)
		return
	}

	err = cloudAccountService.DeleteCloudAccount(id)
	if err != nil {
		result.GErr(ctx, err)
		return
	}

	result.GOK(ctx, nil)
}

// 批量查询云账号余额
func batchQueryBalance(ctx *gin.Context) {
	cloudType := ctx.Query("cloud_type")

	data, err := cloudAccountService.BatchQueryBalance(cloudType)
	if err != nil {
		result.GErr(ctx, err)
		return
	}

	result.GOK(ctx, data)
}

// 获取云账号选项（用于下拉框）
func getCloudAccountOptions(ctx *gin.Context) {
	cloudType := ctx.Query("cloud_type")
	merchantId, _ := strconv.Atoi(ctx.Query("merchant_id"))

	options, err := cloudAccountService.GetCloudAccountOptions(cloudType, merchantId)
	if err != nil {
		result.GErr(ctx, err)
		return
	}

	result.GOK(ctx, options)
}
