package cloud_tencent

import (
	"fmt"
	"net/http"
	"server/internal/server/cloud/tencent"
	"server/internal/server/middleware"
	"server/internal/server/model"
	"server/pkg/result"
	"strconv"

	"github.com/gin-gonic/gin"
)

// Routes 腾讯云服务API路由
func Routes(ge gin.IRouter) {
	cloudGroup := ge.Group("/cloud/tencent", middleware.Authorization)

	// 账户余额
	cloudGroup.GET("/balance", getAccountBalance)

	// COS 对象存储
	cloudGroup.GET("/cos/buckets", listCosBuckets)
	cloudGroup.GET("/cos/objects", listCosObjects)
	cloudGroup.POST("/cos/object", uploadCosObject)
	cloudGroup.GET("/cos/object", downloadCosObject)
	cloudGroup.DELETE("/cos/object", deleteCosObject)
	cloudGroup.POST("/cos/bucket", createCosBucket)
	cloudGroup.DELETE("/cos/bucket", deleteCosBucket)
	cloudGroup.POST("/cos/bucket/set-public", setCosBucketPublic)
}

// ========== COS Handlers ==========

// listCosBuckets 列举 Bucket
func listCosBuckets(c *gin.Context) {
	var req model.CosListBucketsReq
	if err := c.ShouldBindQuery(&req); err != nil {
		result.GErr(c, err)
		return
	}
	if req.MerchantId == 0 && req.CloudAccountId == 0 {
		result.GErr(c, fmt.Errorf("merchant_id或cloud_account_id必须提供一个"))
		return
	}
	data, err := tencent.ListBuckets(req)
	if err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, data)
}

// listCosObjects 列举对象
func listCosObjects(c *gin.Context) {
	var req model.CosListObjectsReq
	if err := c.ShouldBindQuery(&req); err != nil {
		result.GErr(c, err)
		return
	}
	if req.MerchantId == 0 && req.CloudAccountId == 0 {
		result.GErr(c, fmt.Errorf("merchant_id或cloud_account_id必须提供一个"))
		return
	}
	data, err := tencent.ListObjects(req)
	if err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, data)
}

// uploadCosObject 上传对象（multipart/form-data）
func uploadCosObject(c *gin.Context) {
	var form model.CosUploadForm
	form.MerchantId, _ = strconv.Atoi(c.PostForm("merchant_id"))
	if v := c.PostForm("cloud_account_id"); v != "" {
		if id, err := strconv.ParseInt(v, 10, 64); err == nil {
			form.CloudAccountId = id
		}
	}
	form.RegionId = c.PostForm("region_id")
	form.Bucket = c.PostForm("bucket")
	form.ObjectKey = c.PostForm("object_key")

	if form.RegionId == "" || form.Bucket == "" || form.ObjectKey == "" {
		result.GParamErr(c, fmt.Errorf("region_id/bucket/object_key 必填"))
		return
	}
	if form.MerchantId == 0 && form.CloudAccountId == 0 {
		result.GErr(c, fmt.Errorf("merchant_id或cloud_account_id必须提供一个"))
		return
	}

	file, _, err := c.Request.FormFile("file")
	if err != nil {
		result.GParamErr(c, err)
		return
	}
	defer file.Close()

	if err := tencent.UploadObject(form.MerchantId, form.CloudAccountId, form.RegionId, form.Bucket, form.ObjectKey, file); err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, nil)
}

// downloadCosObject 下载对象
func downloadCosObject(c *gin.Context) {
	var req model.CosDownloadReq
	if err := c.ShouldBindQuery(&req); err != nil {
		result.GErr(c, err)
		return
	}
	if req.MerchantId == 0 && req.CloudAccountId == 0 {
		result.GErr(c, fmt.Errorf("merchant_id或cloud_account_id必须提供一个"))
		return
	}
	if req.RegionId == "" || req.Bucket == "" || req.ObjectKey == "" {
		result.GParamErr(c, fmt.Errorf("region_id/bucket/object_key 必填"))
		return
	}

	data, contentType, filename, err := tencent.DownloadObject(req.MerchantId, req.CloudAccountId, req.RegionId, req.Bucket, req.ObjectKey)
	if err != nil {
		result.GErr(c, err)
		return
	}

	if req.Filename != "" {
		filename = req.Filename
	}
	if req.Attachment == 1 {
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	} else {
		c.Header("Content-Disposition", fmt.Sprintf("inline; filename=%q", filename))
	}
	c.Data(http.StatusOK, contentType, data)
}

// deleteCosObject 删除对象
func deleteCosObject(c *gin.Context) {
	var req model.CosDownloadReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.GErr(c, err)
		return
	}
	if req.MerchantId == 0 && req.CloudAccountId == 0 {
		result.GErr(c, fmt.Errorf("merchant_id或cloud_account_id必须提供一个"))
		return
	}
	if req.RegionId == "" || req.Bucket == "" || req.ObjectKey == "" {
		result.GParamErr(c, fmt.Errorf("region_id/bucket/object_key 必填"))
		return
	}

	if err := tencent.DeleteObject(req.MerchantId, req.CloudAccountId, req.RegionId, req.Bucket, req.ObjectKey); err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, nil)
}

// createCosBucket 创建 Bucket
func createCosBucket(c *gin.Context) {
	var req model.CosCreateBucketReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.GErr(c, err)
		return
	}
	if req.MerchantId == 0 && req.CloudAccountId == 0 {
		result.GErr(c, fmt.Errorf("merchant_id或cloud_account_id必须提供一个"))
		return
	}
	if err := tencent.CreateBucket(req.MerchantId, req.CloudAccountId, req.RegionId, req.Bucket); err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, nil)
}

// deleteCosBucket 删除 Bucket
func deleteCosBucket(c *gin.Context) {
	var req model.CosDeleteBucketReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.GErr(c, err)
		return
	}
	if req.MerchantId == 0 && req.CloudAccountId == 0 {
		result.GErr(c, fmt.Errorf("merchant_id或cloud_account_id必须提供一个"))
		return
	}
	if err := tencent.DeleteBucket(req.MerchantId, req.CloudAccountId, req.RegionId, req.Bucket); err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, nil)
}

// setCosBucketPublic 设置 Bucket 公开访问
func setCosBucketPublic(c *gin.Context) {
	var req model.CosSetBucketPublicReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.GErr(c, err)
		return
	}
	if req.MerchantId == 0 && req.CloudAccountId == 0 {
		result.GErr(c, fmt.Errorf("merchant_id或cloud_account_id必须提供一个"))
		return
	}
	if err := tencent.SetBucketPublicAccess(req.MerchantId, req.CloudAccountId, req.RegionId, req.Bucket, req.Public); err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, nil)
}

// ========== Billing Handlers ==========

// getAccountBalance 查询腾讯云账户余额
func getAccountBalance(c *gin.Context) {
	merchantIdStr := c.Query("merchant_id")
	cloudAccountIdStr := c.Query("cloud_account_id")

	var merchantId int
	var cloudAccountId int64
	if merchantIdStr != "" {
		merchantId, _ = strconv.Atoi(merchantIdStr)
	}
	if cloudAccountIdStr != "" {
		cloudAccountId, _ = strconv.ParseInt(cloudAccountIdStr, 10, 64)
	}

	if merchantId == 0 && cloudAccountId == 0 {
		result.GErr(c, fmt.Errorf("merchant_id或cloud_account_id必须提供一个"))
		return
	}

	data, err := tencent.GetAccountBalance(merchantId, cloudAccountId)
	if err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, data)
}
