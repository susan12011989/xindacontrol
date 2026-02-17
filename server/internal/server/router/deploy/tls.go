package deploy

import (
	"server/internal/server/model"
	deployService "server/internal/server/service/deploy"
	"server/pkg/result"

	"github.com/gin-gonic/gin"
)

// ========== TLS 证书管理 ==========

// getTlsCerts 获取当前有效证书
func getTlsCerts(ctx *gin.Context) {
	data, err := deployService.GetTlsCerts()
	if err != nil {
		result.GErr(ctx, err)
		return
	}
	result.GOK(ctx, data)
}

// generateTlsCerts 生成 CA + 服务器证书
func generateTlsCerts(ctx *gin.Context) {
	var req model.GenerateTlsCertReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		// 允许空 body，使用默认值
		req = model.GenerateTlsCertReq{}
	}

	data, err := deployService.GenerateTlsCerts(req.ValidityDays)
	if err != nil {
		result.GErr(ctx, err)
		return
	}
	result.GOK(ctx, data)
}

// disableTlsCerts 停用当前证书
func disableTlsCerts(ctx *gin.Context) {
	err := deployService.DisableTlsCerts()
	if err != nil {
		result.GErr(ctx, err)
		return
	}
	result.GOK(ctx, gin.H{"message": "证书已停用"})
}

// getTlsCertFingerprint 获取证书指纹（供 App 端 Pinning）
func getTlsCertFingerprint(ctx *gin.Context) {
	data, err := deployService.GetCertFingerprint()
	if err != nil {
		result.GErr(ctx, err)
		return
	}
	result.GOK(ctx, data)
}

// ========== TLS 批量操作 ==========

// getTlsStatus 查看所有系统服务器 TLS 状态
func getTlsStatus(ctx *gin.Context) {
	data, err := deployService.GetTlsStatus()
	if err != nil {
		result.GErr(ctx, err)
		return
	}
	result.GOK(ctx, data)
}

// verifyTlsStatus 验证所有系统服务器 TLS 连接
func verifyTlsStatus(ctx *gin.Context) {
	data, err := deployService.VerifyTlsStatus()
	if err != nil {
		result.GErr(ctx, err)
		return
	}
	result.GOK(ctx, data)
}

// batchUpgradeTls 批量升级为 TLS
func batchUpgradeTls(ctx *gin.Context) {
	var req model.BatchUpgradeTlsReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		req = model.BatchUpgradeTlsReq{}
	}

	data, err := deployService.BatchUpgradeTls(req)
	if err != nil {
		result.GErr(ctx, err)
		return
	}
	result.GOK(ctx, data)
}

// batchRollbackTls 批量回滚为 TCP
func batchRollbackTls(ctx *gin.Context) {
	var req model.BatchRollbackTlsReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		req = model.BatchRollbackTlsReq{}
	}

	data, err := deployService.BatchRollbackTls(req)
	if err != nil {
		result.GErr(ctx, err)
		return
	}
	result.GOK(ctx, data)
}