package merchant

import (
	"encoding/json"
	"server/internal/server/middleware"
	merchantService "server/internal/server/service/merchant"
	"server/pkg/result"

	"github.com/gin-gonic/gin"
)

// RoutesAppLogs 应用日志路由
func RoutesAppLogs(gi gin.IRouter) {
	group := gi.Group("app_logs")
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
		keyword := c.Query("keyword")
		resp, err := merchantService.QueryAppLogs(merchantNo, page, size, keyword)
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
}
