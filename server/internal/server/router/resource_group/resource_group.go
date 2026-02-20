package resource_group

import (
	"server/internal/server/middleware"
	"server/internal/server/model"
	service "server/internal/server/service/resource_group"
	"server/pkg/entity"
	"server/pkg/result"
	"strconv"

	"github.com/gin-gonic/gin"
)

// Routes 注册资源分组路由
func Routes(gi gin.IRouter) {
	group := gi.Group("resource-groups")
	group.Use(middleware.Authorization)

	group.GET("", listGroups)
	group.POST("", createGroup)
	group.PUT("/:id", updateGroup)
	group.DELETE("/:id", deleteGroup)
}

func listGroups(c *gin.Context) {
	resourceType := c.DefaultQuery("type", entity.ResourceGroupTypeIpEmbedTarget)
	data, err := service.ListGroups(resourceType)
	if err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, data)
}

func createGroup(c *gin.Context) {
	var req model.ResourceGroupReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.GParamErr(c, err)
		return
	}
	resourceType := c.DefaultQuery("type", entity.ResourceGroupTypeIpEmbedTarget)
	id, err := service.CreateGroup(resourceType, req)
	if err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, gin.H{"id": id})
}

func updateGroup(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		result.GParamErr(c, err)
		return
	}
	var req model.ResourceGroupReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.GParamErr(c, err)
		return
	}
	if err := service.UpdateGroup(id, req); err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, nil)
}

func deleteGroup(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		result.GParamErr(c, err)
		return
	}
	resourceType := c.DefaultQuery("type", entity.ResourceGroupTypeIpEmbedTarget)
	if err := service.DeleteGroup(id, resourceType); err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, nil)
}
