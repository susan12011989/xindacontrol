package merchant

import (
	"server/internal/server/model"
	"server/internal/server/service/merchant"
	"server/pkg/result"
	"strconv"

	"github.com/gin-gonic/gin"
)

// listMerchantTurnConfigs 获取所有商户的 TURN 配置列表
func listMerchantTurnConfigs(c *gin.Context) {
	name := c.Query("name")
	list, err := merchant.ListMerchantTurnConfigs(name)
	if err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, list)
}

// updateMerchantTurnServer 更新单个商户的 TURN 服务器
func updateMerchantTurnServer(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		result.GParamErr(c, err)
		return
	}

	var req struct {
		TurnServer     string `json:"turn_server" binding:"required"`
		TurnUsername   string `json:"turn_username"`
		TurnCredential string `json:"turn_credential"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		result.GParamErr(c, err)
		return
	}

	data, err := merchant.UpdateMerchantTurnServer(id, merchant.TurnConfig{
		Server:     req.TurnServer,
		Username:   req.TurnUsername,
		Credential: req.TurnCredential,
	})
	if err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, data)
}

// batchUpdateTurnServer 批量更新商户 TURN 服务器
func batchUpdateTurnServer(c *gin.Context) {
	var req model.BatchUpdateTurnServerReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.GParamErr(c, err)
		return
	}

	data, err := merchant.BatchUpdateTurnServer(req)
	if err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, data)
}
