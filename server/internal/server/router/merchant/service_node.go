package merchant

import (
	"server/internal/server/service/audit"
	svcMerchant "server/internal/server/service/merchant"
	"server/pkg/entity"
	"server/pkg/result"
	"strconv"

	"github.com/gin-gonic/gin"
)

// 获取商户服务节点列表
func listServiceNodes(c *gin.Context) {
	merchantId, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		result.GParamErr(c, err)
		return
	}

	list, err := svcMerchant.ListServiceNodes(merchantId)
	if err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, list)
}

// 创建商户服务节点
func createServiceNode(c *gin.Context) {
	merchantId, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		result.GParamErr(c, err)
		return
	}

	var req svcMerchant.ServiceNodeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.GParamErr(c, err)
		return
	}
	req.MerchantId = merchantId

	id, err := svcMerchant.CreateServiceNode(req)
	if err != nil {
		audit.LogErrorFromContext(c, entity.AuditActionCreateServiceNode, entity.AuditTargetServiceNode,
			0, req.Role, req, err.Error())
		result.GErr(c, err)
		return
	}

	audit.LogFromContext(c, entity.AuditActionCreateServiceNode, entity.AuditTargetServiceNode,
		id, req.Role, map[string]interface{}{"merchant_id": merchantId, "host": req.Host, "role": req.Role})
	result.GOK(c, gin.H{"id": id})
}

// 更新商户服务节点
func updateServiceNode(c *gin.Context) {
	nodeId, err := strconv.Atoi(c.Param("node_id"))
	if err != nil {
		result.GParamErr(c, err)
		return
	}

	var req svcMerchant.ServiceNodeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.GParamErr(c, err)
		return
	}

	if err := svcMerchant.UpdateServiceNode(nodeId, req); err != nil {
		audit.LogErrorFromContext(c, entity.AuditActionUpdateServiceNode, entity.AuditTargetServiceNode,
			nodeId, req.Role, req, err.Error())
		result.GErr(c, err)
		return
	}

	audit.LogFromContext(c, entity.AuditActionUpdateServiceNode, entity.AuditTargetServiceNode,
		nodeId, req.Role, req)
	result.GOK(c, nil)
}

// 删除商户服务节点
func deleteServiceNode(c *gin.Context) {
	nodeId, err := strconv.Atoi(c.Param("node_id"))
	if err != nil {
		result.GParamErr(c, err)
		return
	}

	if err := svcMerchant.DeleteServiceNode(nodeId); err != nil {
		audit.LogErrorFromContext(c, entity.AuditActionDeleteServiceNode, entity.AuditTargetServiceNode,
			nodeId, "", nil, err.Error())
		result.GErr(c, err)
		return
	}

	audit.LogFromContext(c, entity.AuditActionDeleteServiceNode, entity.AuditTargetServiceNode,
		nodeId, "", nil)
	result.GOK(c, nil)
}

// 切换到多机模式
func switchToClusterMode(c *gin.Context) {
	merchantId, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		result.GParamErr(c, err)
		return
	}

	var req struct {
		Nodes []svcMerchant.ServiceNodeReq `json:"nodes" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		result.GParamErr(c, err)
		return
	}

	if err := svcMerchant.SwitchToClusterMode(merchantId, req.Nodes); err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, nil)
}
