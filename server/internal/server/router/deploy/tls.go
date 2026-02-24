package deploy

import (
	"fmt"
	"server/internal/server/model"
	deployService "server/internal/server/service/deploy"
	"server/pkg/result"
	"strconv"

	"github.com/gin-gonic/gin"
)

// ========== TLS 证书管理 ==========

// getTlsCerts 获取指定商户的有效证书
func getTlsCerts(ctx *gin.Context) {
	merchantId, err := strconv.Atoi(ctx.Query("merchant_id"))
	if err != nil || merchantId <= 0 {
		result.GErr(ctx, fmt.Errorf("merchant_id 参数无效"))
		return
	}

	data, err := deployService.GetTlsCerts(merchantId)
	if err != nil {
		result.GErr(ctx, err)
		return
	}
	result.GOK(ctx, data)
}

// generateTlsCerts 为指定商户生成 CA + 服务器证书
func generateTlsCerts(ctx *gin.Context) {
	var req model.GenerateTlsCertReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		result.GErr(ctx, err)
		return
	}

	data, err := deployService.GenerateTlsCerts(req.MerchantId, req.ValidityDays)
	if err != nil {
		result.GErr(ctx, err)
		return
	}
	result.GOK(ctx, data)
}

// disableTlsCerts 停用指定商户的证书
func disableTlsCerts(ctx *gin.Context) {
	var req model.DisableTlsCertReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		result.GErr(ctx, err)
		return
	}

	err := deployService.DisableTlsCerts(req.MerchantId)
	if err != nil {
		result.GErr(ctx, err)
		return
	}
	result.GOK(ctx, gin.H{"message": "证书已停用"})
}

// getTlsCertFingerprint 获取指定商户的证书指纹（供 App 端 Pinning）
func getTlsCertFingerprint(ctx *gin.Context) {
	merchantId, err := strconv.Atoi(ctx.Query("merchant_id"))
	if err != nil || merchantId <= 0 {
		result.GErr(ctx, fmt.Errorf("merchant_id 参数无效"))
		return
	}

	data, err := deployService.GetCertFingerprint(merchantId)
	if err != nil {
		result.GErr(ctx, err)
		return
	}
	result.GOK(ctx, data)
}

// ========== TLS 批量操作 ==========

// getTlsStatus 查看指定商户的 GOST 服务器 TLS 状态
func getTlsStatus(ctx *gin.Context) {
	merchantId, err := strconv.Atoi(ctx.Query("merchant_id"))
	if err != nil || merchantId <= 0 {
		result.GErr(ctx, fmt.Errorf("merchant_id 参数无效"))
		return
	}

	data, err := deployService.GetTlsStatus(merchantId)
	if err != nil {
		result.GErr(ctx, err)
		return
	}
	result.GOK(ctx, data)
}

// verifyTlsStatus 验证指定商户的 GOST 服务器 TLS 连接
func verifyTlsStatus(ctx *gin.Context) {
	var req model.BatchUpgradeTlsReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		result.GErr(ctx, err)
		return
	}

	data, err := deployService.VerifyTlsStatus(req.MerchantId)
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
		result.GErr(ctx, err)
		return
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
		result.GErr(ctx, err)
		return
	}

	data, err := deployService.BatchRollbackTls(req)
	if err != nil {
		result.GErr(ctx, err)
		return
	}
	result.GOK(ctx, data)
}
