package merchant

import (
	"encoding/json"
	"server/internal/server/middleware"
	"server/internal/server/model"
	merchantService "server/internal/server/service/merchant"
	"server/pkg/result"

	"github.com/gin-gonic/gin"
)

// RoutesAdminmUsers adminm 登录账号管理路由
func RoutesAdminmUsers(gi gin.IRouter) {
	group := gi.Group("adminm_users")
	group.Use(middleware.Authorization)

	group.GET("", func(c *gin.Context) { // 列表查询
		merchantNo := c.Query("merchant_no")
		if merchantNo == "" {
			result.GResult(c, 601, nil, "merchant_no不能为空")
			return
		}
		page := 1
		size := 20
		if v := c.Query("page"); v != "" {
			_ = json.Unmarshal([]byte(v), &page)
		}
		if v := c.Query("size"); v != "" {
			_ = json.Unmarshal([]byte(v), &size)
		}
		username := c.Query("username")
		resp, err := merchantService.QueryAdminmUsers(merchantNo, page, size, username)
		if err != nil {
			result.GErr(c, err)
			return
		}
		if resp.Err != "" {
			result.GResult(c, 500, nil, resp.Err)
			return
		}
		result.GOK(c, gin.H{"list": resp.List, "total": resp.Total})
	})

	group.GET("active", func(c *gin.Context) { // 活跃数据
		merchantNo := c.Query("merchant_no")
		if merchantNo == "" {
			result.GResult(c, 601, nil, "merchant_no不能为空")
			return
		}
		resp, err := merchantService.QueryAdminmActive(merchantNo)
		if err != nil {
			result.GErr(c, err)
			return
		}
		if resp.Err != "" {
			result.GResult(c, 500, nil, resp.Err)
			return
		}
		result.GOK(c, gin.H{
			"total_users":  resp.TotalUsers,
			"online_users": resp.OnlineUsers,
			"dau":          resp.Dau,
		})
	})

	group.POST("", func(c *gin.Context) { // 创建
		var req model.CreateAdminmUserReq
		if err := c.ShouldBindJSON(&req); err != nil {
			result.GParamErr(c, err)
			return
		}
		merchantService.CreateAdminmUser(&req)
		result.GOK(c, nil)
	})

	group.PUT("", func(c *gin.Context) { // 更新
		var req model.UpdateAdminmUserReq
		if err := c.ShouldBindJSON(&req); err != nil {
			result.GParamErr(c, err)
			return
		}
		merchantService.UpdateAdminmUser(&req)
		result.GOK(c, nil)
	})

	group.DELETE("", func(c *gin.Context) { // 删除
		var req model.DeleteAdminmUserReq
		if err := c.ShouldBindJSON(&req); err != nil {
			result.GParamErr(c, err)
			return
		}
		merchantService.DeleteAdminmUser(&req)
		result.GOK(c, nil)
	})
}
