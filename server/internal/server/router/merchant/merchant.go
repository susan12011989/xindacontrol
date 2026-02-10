package merchant

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"server/internal/server/cloud/aliyun"
	"server/internal/server/middleware"
	"server/internal/server/service/audit"
	"server/internal/server/service/merchant"
	"server/pkg/consts"
	"server/pkg/entity"
	"server/pkg/result"
	"strconv"
	"strings"
	"sync"
	"time"

	"server/internal/server/model"

	"github.com/gin-gonic/gin"
)

func Routes(ge gin.IRouter) {
	merchantGroup := ge.Group("merchant", middleware.Authorization)
	merchantGroup.GET("", listMerchant)
	merchantGroup.POST("", createMerchant)
	merchantGroup.PUT("/:id", updateMerchant)
	merchantGroup.DELETE("/:id", deleteMerchant)
	merchantGroup.GET("balance-cloud", getBalance)              // 获取云账号的余额
	merchantGroup.GET("tunnel-check", tunnelCheck)              // 隧道连接检测
	merchantGroup.GET("tunnel-stats", getTunnelStats)            // 隧道统计
	merchantGroup.POST("/:id/change-ip", changeMerchantIP)      // 更换IP（AWS）
	merchantGroup.POST("/:id/change-gost-port", changeGostPort) // 更换 GOST 转发端口

	// 商户 OSS 配置管理
	merchantGroup.GET("/:id/oss-configs", listMerchantOssConfigs)
	merchantGroup.POST("/:id/oss-configs", createMerchantOssConfig)
	merchantGroup.PUT("/oss-configs/:config_id", updateMerchantOssConfig)
	merchantGroup.DELETE("/oss-configs/:config_id", deleteMerchantOssConfig)

	// 商户 GOST 服务器管理
	merchantGroup.GET("/:id/gost-servers", listMerchantGostServers)
	merchantGroup.POST("/:id/gost-servers", createMerchantGostServer)
	merchantGroup.PUT("/gost-servers/:relation_id", updateMerchantGostServer)
	merchantGroup.DELETE("/gost-servers/:relation_id", deleteMerchantGostServer)

	// 商户 GOST IP 同步到 OSS
	merchantGroup.GET("/:id/gost-sync-status", getMerchantGostSyncStatus)
	merchantGroup.POST("/:id/sync-gost-ip", syncMerchantGostIP)
	merchantGroup.POST("/batch-sync-gost-ip", batchSyncMerchantGostIP)

	// adminm 登录账号管理
	RoutesAdminmUsers(merchantGroup)
	// adminm 配置查询
	RoutesAdminmConfig(merchantGroup)
	// 应用日志
	RoutesAppLogs(merchantGroup)

	// 资源上传（Logo等）
	merchantGroup.POST("upload-asset", uploadAsset)
}

func getBalance(c *gin.Context) {
	var req model.BalanceReq
	if err := c.ShouldBindQuery(&req); err != nil {
		result.GErr(c, err)
		return
	}

	balances := make([]*model.BalanceData, 0, len(req.MerchantId))
	var wg sync.WaitGroup
	balanceCh := make(chan *model.BalanceData, len(req.MerchantId))

	for _, mid := range req.MerchantId {
		wg.Add(1)
		go func(merchantId int) {
			defer wg.Done()
			balance, err := aliyun.Balance(merchantId)
			if err != nil {
				return
			}
			balanceCh <- &model.BalanceData{
				MerchantId: merchantId,
				Balance:    balance,
			}
		}(mid)
	}

	go func() {
		wg.Wait()
		close(balanceCh)
	}()

	for balance := range balanceCh {
		balances = append(balances, balance)
	}

	result.GOK(c, balances)
}

// 获取商户列表
func listMerchant(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "10"))
	name := c.Query("name")
	orderBy := c.Query("order")
	merchantNo := c.Query("merchant_no")
	if orderBy == "" {
		orderBy = "id desc"
	}
	expiringSoon, _ := strconv.Atoi(c.DefaultQuery("expiring_soon", "0"))

	merchantList, total, err := merchant.ListMerchant(page, size, name, orderBy, expiringSoon, merchantNo)
	if err != nil {
		result.GErr(c, err)
		return
	}

	result.GOK(c, gin.H{
		"list":  merchantList,
		"total": total,
	})
}

func createMerchant(c *gin.Context) {
	var req model.CreateOrEditMerchantReq

	if err := c.ShouldBindJSON(&req); err != nil {
		result.GErr(c, err)
		return
	}

	if err := merchant.CreateMerchant(&req); err != nil {
		audit.LogErrorFromContext(c, entity.AuditActionCreateMerchant, entity.AuditTargetMerchant,
			0, req.Name, req, err.Error())
		result.GErr(c, err)
		return
	}

	audit.LogFromContext(c, entity.AuditActionCreateMerchant, entity.AuditTargetMerchant,
		0, req.Name, map[string]interface{}{"port": req.Port, "server_ip": req.ServerIP})
	result.GOK(c, nil)
}

// 更新商户
func updateMerchant(c *gin.Context) {
	idStr := c.Param("id")
	id, _ := strconv.Atoi(idStr)
	var req model.CreateOrEditMerchantReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.GErr(c, err)
		return
	}
	if err := merchant.UpdateMerchant(idStr, &req); err != nil {
		audit.LogErrorFromContext(c, entity.AuditActionUpdateMerchant, entity.AuditTargetMerchant,
			id, req.Name, req, err.Error())
		result.GErr(c, err)
		return
	}
	audit.LogFromContext(c, entity.AuditActionUpdateMerchant, entity.AuditTargetMerchant,
		id, req.Name, req)
	result.GOK(c, nil)
}

func deleteMerchant(c *gin.Context) {
	idStr := c.Param("id")
	id, _ := strconv.Atoi(idStr)
	merchantName, err := merchant.DeleteMerchant(idStr)
	if err != nil {
		audit.LogErrorFromContext(c, entity.AuditActionDeleteMerchant, entity.AuditTargetMerchant,
			id, "", nil, err.Error())
		result.GErr(c, err)
		return
	}
	audit.LogFromContext(c, entity.AuditActionDeleteMerchant, entity.AuditTargetMerchant,
		id, merchantName, nil)
	result.GOK(c, nil)
}

// ===== 简易防抖 =====
type opRecord struct {
	last time.Time
	mu   sync.Mutex
}

var debounceMap sync.Map

func allowAndMark(key string, window time.Duration) bool {
	now := time.Now()
	val, _ := debounceMap.LoadOrStore(key, &opRecord{})
	rec := val.(*opRecord)
	rec.mu.Lock()
	defer rec.mu.Unlock()
	if !rec.last.IsZero() && now.Sub(rec.last) < window {
		return false
	}
	rec.last = now
	return true
}

// 隧道统计：获取商户、系统服务器、商户服务器的统计数据
func getTunnelStats(c *gin.Context) {
	data, err := merchant.GetTunnelStats()
	if err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, data)
}

// 隧道连接检测：遍历所有系统服务器，通过SSH从远端探测当前商户IP:10544
func tunnelCheck(c *gin.Context) {
	var req model.TunnelCheckReq
	if err := c.ShouldBindQuery(&req); err != nil {
		result.GParamErr(c, err)
		return
	}

	// 防抖：同一商户/目标IP 20 秒内只允许一次
	key := "tunnel:"
	if req.MerchantId > 0 {
		key += strconv.Itoa(req.MerchantId)
	} else {
		key += req.ServerIP
	}
	if !allowAndMark(key, 8*time.Second) {
		result.GResult(c, 429, nil, "操作过于频繁，请稍后再试")
		return
	}

	// 调用 service
	data, err := merchant.TunnelCheck(req)
	if err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, data)
}

// 更换商户公网IP（AWS）
func changeMerchantIP(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		result.GParamErr(c, err)
		return
	}

	// 防抖：同一商户 60 秒内只允许一次
	key := "changeip:" + idStr
	if !allowAndMark(key, 60*time.Second) {
		result.GResult(c, 429, nil, "操作过于频繁，请稍后再试")
		return
	}
	data, err := merchant.ChangeMerchantIP(id)
	if err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, data)
}

// 更换商户 GOST 转发端口（仅更换商户服务器上的 GOST 监听端口及系统服务器转发配置，商户公网 IP 不变）
func changeGostPort(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		result.GParamErr(c, err)
		return
	}

	var req model.ChangeGostPortReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.GErr(c, err)
		return
	}

	data, err := merchant.ChangeMerchantGostPort(id, req.GostPort)
	if err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, data)
}

// ========== 商户 OSS 配置管理 ==========

// 获取商户 OSS 配置列表
func listMerchantOssConfigs(c *gin.Context) {
	idStr := c.Param("id")
	merchantId, err := strconv.Atoi(idStr)
	if err != nil {
		result.GParamErr(c, err)
		return
	}

	list, err := merchant.ListMerchantOssConfigs(merchantId)
	if err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, list)
}

// 创建商户 OSS 配置
func createMerchantOssConfig(c *gin.Context) {
	idStr := c.Param("id")
	merchantId, err := strconv.Atoi(idStr)
	if err != nil {
		result.GParamErr(c, err)
		return
	}

	var req merchant.MerchantOssConfigReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.GParamErr(c, err)
		return
	}
	req.MerchantId = merchantId

	id, err := merchant.CreateMerchantOssConfig(req)
	if err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, gin.H{"id": id})
}

// 更新商户 OSS 配置
func updateMerchantOssConfig(c *gin.Context) {
	configIdStr := c.Param("config_id")
	configId, err := strconv.Atoi(configIdStr)
	if err != nil {
		result.GParamErr(c, err)
		return
	}

	var req merchant.MerchantOssConfigReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.GParamErr(c, err)
		return
	}

	err = merchant.UpdateMerchantOssConfig(configId, req)
	if err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, nil)
}

// 删除商户 OSS 配置
func deleteMerchantOssConfig(c *gin.Context) {
	configIdStr := c.Param("config_id")
	configId, err := strconv.Atoi(configIdStr)
	if err != nil {
		result.GParamErr(c, err)
		return
	}

	err = merchant.DeleteMerchantOssConfig(configId)
	if err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, nil)
}

// ========== 商户 GOST 服务器管理 ==========

// 获取商户 GOST 服务器列表
func listMerchantGostServers(c *gin.Context) {
	idStr := c.Param("id")
	merchantId, err := strconv.Atoi(idStr)
	if err != nil {
		result.GParamErr(c, err)
		return
	}

	list, err := merchant.ListMerchantGostServers(merchantId)
	if err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, list)
}

// 创建商户 GOST 服务器关联
func createMerchantGostServer(c *gin.Context) {
	idStr := c.Param("id")
	merchantId, err := strconv.Atoi(idStr)
	if err != nil {
		result.GParamErr(c, err)
		return
	}

	var req merchant.MerchantGostServerReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.GParamErr(c, err)
		return
	}
	req.MerchantId = merchantId

	id, err := merchant.CreateMerchantGostServer(req)
	if err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, gin.H{"id": id})
}

// 更新商户 GOST 服务器关联
func updateMerchantGostServer(c *gin.Context) {
	relationIdStr := c.Param("relation_id")
	relationId, err := strconv.Atoi(relationIdStr)
	if err != nil {
		result.GParamErr(c, err)
		return
	}

	var req merchant.MerchantGostServerReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.GParamErr(c, err)
		return
	}

	err = merchant.UpdateMerchantGostServer(relationId, req)
	if err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, nil)
}

// 删除商户 GOST 服务器关联
func deleteMerchantGostServer(c *gin.Context) {
	relationIdStr := c.Param("relation_id")
	relationId, err := strconv.Atoi(relationIdStr)
	if err != nil {
		result.GParamErr(c, err)
		return
	}

	err = merchant.DeleteMerchantGostServer(relationId)
	if err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, nil)
}

// ========== 商户 GOST IP 同步 ==========

// 获取商户 GOST IP 同步状态
func getMerchantGostSyncStatus(c *gin.Context) {
	idStr := c.Param("id")
	merchantId, err := strconv.Atoi(idStr)
	if err != nil {
		result.GParamErr(c, err)
		return
	}

	data, err := merchant.GetMerchantGostIPSyncStatus(merchantId)
	if err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, data)
}

// 同步商户 GOST IP 到 OSS
func syncMerchantGostIP(c *gin.Context) {
	idStr := c.Param("id")
	merchantId, err := strconv.Atoi(idStr)
	if err != nil {
		result.GParamErr(c, err)
		return
	}

	var req merchant.SyncGostIPReq
	if err := c.ShouldBindJSON(&req); err != nil {
		// 允许空 body，使用默认参数
		req = merchant.SyncGostIPReq{}
	}
	req.MerchantId = merchantId

	// 防抖：同一商户 10 秒内只允许一次
	key := "syncgostip:" + idStr
	if !allowAndMark(key, 10*time.Second) {
		result.GResult(c, 429, nil, "操作过于频繁，请稍后再试")
		return
	}

	data, err := merchant.SyncMerchantGostIP(req)
	if err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, data)
}

// 批量同步商户 GOST IP 到 OSS
func batchSyncMerchantGostIP(c *gin.Context) {
	var req struct {
		MerchantIds []int `json:"merchant_ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		result.GParamErr(c, err)
		return
	}

	if len(req.MerchantIds) == 0 {
		result.GParamErr(c, fmt.Errorf("merchant_ids 不能为空"))
		return
	}

	data, err := merchant.BatchSyncMerchantGostIP(req.MerchantIds)
	if err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, data)
}

// ========== 资源上传 ==========

// uploadAsset 上传资源文件（Logo等）到本地
func uploadAsset(c *gin.Context) {
	// 获取上传类型（logo, icon 等）
	assetType := c.PostForm("type")
	if assetType == "" {
		assetType = "logo"
	}

	// 验证类型
	allowedTypes := map[string]bool{"logo": true, "icon": true, "splash": true}
	if !allowedTypes[assetType] {
		result.GParamErr(c, fmt.Errorf("不支持的资源类型: %s", assetType))
		return
	}

	// 获取文件
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		result.GParamErr(c, fmt.Errorf("读取文件失败: %v", err))
		return
	}
	defer file.Close()

	// 验证文件类型
	contentType := header.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "image/") {
		result.GParamErr(c, fmt.Errorf("只支持图片文件"))
		return
	}

	// 获取文件扩展名
	ext := filepath.Ext(header.Filename)
	if ext == "" {
		ext = ".png"
	}

	// 生成唯一文件名
	filename := fmt.Sprintf("%s_%d_%s%s", assetType, time.Now().UnixNano(), randString(6), ext)

	// 确保目录存在
	uploadDir := filepath.Join(consts.AssetsDir, assetType)
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		result.GErr(c, fmt.Errorf("创建目录失败: %v", err))
		return
	}

	// 保存文件
	filePath := filepath.Join(uploadDir, filename)
	dst, err := os.Create(filePath)
	if err != nil {
		result.GErr(c, fmt.Errorf("创建文件失败: %v", err))
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		result.GErr(c, fmt.Errorf("保存文件失败: %v", err))
		return
	}

	// 返回访问 URL
	url := fmt.Sprintf("/assets/%s/%s", assetType, filename)
	result.GOK(c, gin.H{
		"url":      url,
		"filename": filename,
		"type":     assetType,
	})
}

// randString 生成随机字符串
func randString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[time.Now().UnixNano()%int64(len(letters))]
		time.Sleep(time.Nanosecond)
	}
	return string(b)
}
