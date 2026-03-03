package merchant

import (
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"server/internal/dbhelper"
	"server/internal/server/middleware"
	"server/internal/server/service/auth"
	merchantService "server/internal/server/service/merchant"
	"server/pkg/consts"
	"server/pkg/entity"
	"server/pkg/result"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zeromicro/go-zero/core/logx"
)

// RoutesAdminmConfig adminm 配置查询路由
func RoutesAdminmConfig(gi gin.IRouter) {
	group := gi.Group("adminm_config")
	group.Use(middleware.Authorization)

	// 保存敏感词配置（从txt文本中解析，英文逗号分隔：第一列 word，第二列 tip）：支持单个、批量或全部
	type sensitiveSaveReq struct {
		MerchantNo  string                    `json:"merchant_no"`
		MerchantNos []string                  `json:"merchant_nos"`
		Broadcast   bool                      `json:"broadcast"`
		Txt         string                    `json:"txt"`                // 文本内容，每行一个：word,tip
		Contents    []*entity.SensitiveContent `json:"contents,omitempty"` // 直接传数组，允许空数组清空
	}
	group.POST("sensitive_contents", func(c *gin.Context) {
		var req sensitiveSaveReq
		if err := c.ShouldBindJSON(&req); err != nil {
			result.GParamErr(c, err)
			return
		}
		// 至少提供 txt 或 contents 之一（contents 可为空数组表示清空）
		if req.Contents == nil && strings.TrimSpace(req.Txt) == "" {
			result.GResult(c, 601, nil, "txt或contents必须至少提供一个")
			return
		}
		var contents []*entity.SensitiveContent
		if req.Contents != nil {
			// 直接使用传入的数组（允许空数组）
			contents = req.Contents
		} else {
			// 解析 txt -> []SensitiveContent（txt 可解析为空数组表示清空）
			lines := strings.Split(req.Txt, "\n")
			contents = make([]*entity.SensitiveContent, 0, len(lines))
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if line == "" {
					continue
				}
				// 允许注释行（以#开头或//开头）
				if strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") {
					continue
				}
				parts := strings.SplitN(line, ",", 2)
				if len(parts) < 2 {
					continue
				}
				word := strings.TrimSpace(parts[0])
				if word == "" {
					continue
				}
				tip := strings.TrimSpace(parts[1])
				if tip == "" {
					continue
				}
				contents = append(contents, &entity.SensitiveContent{Word: word, Tip: tip})
			}
		}
		// 目标判断：三选一
		targetCount := 0
		if req.Broadcast {
			targetCount++
		}
		if req.MerchantNo != "" {
			targetCount++
		}
		if len(req.MerchantNos) > 0 {
			targetCount++
		}
		if targetCount != 1 {
			result.GResult(c, 601, nil, "必须且只能指定一种目标：broadcast 或 merchant_no 或 merchant_nos")
			return
		}

		if req.Broadcast {
			merchants, err := dbhelper.FindAllMerchants()
			if err != nil {
				result.GErr(c, err)
				return
			}
			successCount := 0
			var failures []gin.H
			for _, m := range merchants {
				if err := merchantService.SaveAdminmSensitiveContents(m.No, contents); err != nil {
					logx.Errorf("广播敏感词配置失败: merchant=%s, err=%v", m.No, err)
					failures = append(failures, gin.H{"merchant_no": m.No, "name": m.Name, "error": err.Error()})
				} else {
					successCount++
				}
			}
			result.GOK(c, gin.H{"mode": "broadcast", "count": len(contents), "total": len(merchants), "success": successCount, "failures": failures})
			return
		}

		if req.MerchantNo != "" {
			if err := merchantService.SaveAdminmSensitiveContents(req.MerchantNo, contents); err != nil {
				result.GResult(c, 500, nil, err.Error())
				return
			}
			result.GOK(c, gin.H{"updated": 1, "count": len(contents)})
			return
		}

		successCount := 0
		var failures []gin.H
		for _, no := range req.MerchantNos {
			no = strings.TrimSpace(no)
			if no == "" {
				continue
			}
			if err := merchantService.SaveAdminmSensitiveContents(no, contents); err != nil {
				logx.Errorf("批量保存敏感词配置失败: merchant=%s, err=%v", no, err)
				failures = append(failures, gin.H{"merchant_no": no, "error": err.Error()})
			} else {
				successCount++
			}
		}
		result.GOK(c, gin.H{"updated": successCount, "count": len(contents), "total": len(req.MerchantNos), "failures": failures})
	})

	// 保存系统用户昵称（users.id=777000 的 first_name）：支持单个、批量或全部
	type nicknameSaveReq struct {
		MerchantNo  string   `json:"merchant_no"`
		MerchantNos []string `json:"merchant_nos"`
		Broadcast   bool     `json:"broadcast"`
		FirstName   string   `json:"first_name"`
	}
	group.POST("system_user_nickname", func(c *gin.Context) {
		var req nicknameSaveReq
		if err := c.ShouldBindJSON(&req); err != nil {
			result.GParamErr(c, err)
			return
		}
		if req.FirstName == "" {
			result.GResult(c, 601, nil, "first_name不能为空")
			return
		}
		// 目标判断：三选一
		targetCount := 0
		if req.Broadcast {
			targetCount++
		}
		if req.MerchantNo != "" {
			targetCount++
		}
		if len(req.MerchantNos) > 0 {
			targetCount++
		}
		if targetCount != 1 {
			result.GResult(c, 601, nil, "必须且只能指定一种目标：broadcast 或 merchant_no 或 merchant_nos")
			return
		}

		if req.Broadcast {
			merchants, err := dbhelper.FindAllMerchants()
			if err != nil {
				result.GErr(c, err)
				return
			}
			successCount := 0
			var failures []gin.H
			for _, m := range merchants {
				if err := merchantService.SaveAdminmSystemNickname(m.No, req.FirstName); err != nil {
					logx.Errorf("广播系统昵称失败: merchant=%s, err=%v", m.No, err)
					failures = append(failures, gin.H{"merchant_no": m.No, "name": m.Name, "error": err.Error()})
				} else {
					successCount++
				}
			}
			result.GOK(c, gin.H{"mode": "broadcast", "total": len(merchants), "success": successCount, "failures": failures})
			return
		}

		if req.MerchantNo != "" {
			if err := merchantService.SaveAdminmSystemNickname(req.MerchantNo, req.FirstName); err != nil {
				result.GResult(c, 500, nil, err.Error())
				return
			}
			result.GOK(c, gin.H{"updated": 1})
			return
		}

		// 批量
		successCount := 0
		var failures []gin.H
		for _, no := range req.MerchantNos {
			no = strings.TrimSpace(no)
			if no == "" {
				continue
			}
			if err := merchantService.SaveAdminmSystemNickname(no, req.FirstName); err != nil {
				logx.Errorf("批量保存系统昵称失败: merchant=%s, err=%v", no, err)
				failures = append(failures, gin.H{"merchant_no": no, "error": err.Error()})
			} else {
				successCount++
			}
		}
		result.GOK(c, gin.H{"updated": successCount, "total": len(req.MerchantNos), "failures": failures})
	})

	// 读取商户短信配置
	group.GET("sms", func(c *gin.Context) {
		merchantNo := c.Query("merchant_no")
		if merchantNo == "" {
			result.GParamErr(c, fmt.Errorf("merchant_no不能为空"))
			return
		}
		config, err := merchantService.GetAdminmSmsConfig(merchantNo)
		if err != nil {
			result.GErr(c, err)
			return
		}
		result.GOK(c, config)
	})

	// 保存短信配置：支持单个、批量或全部
	type smsSaveReq struct {
		MerchantNo  string            `json:"merchant_no"`
		MerchantNos []string          `json:"merchant_nos"`
		Broadcast   bool              `json:"broadcast"`
		Config      *entity.SmsConfig `json:"config"`
	}
	group.POST("sms", func(c *gin.Context) {
		var req smsSaveReq
		if err := c.ShouldBindJSON(&req); err != nil {
			result.GParamErr(c, err)
			return
		}
		if req.Config == nil {
			result.GResult(c, 601, nil, "config不能为空")
			return
		}
		// 目标判断：三选一
		targetCount := 0
		if req.Broadcast {
			targetCount++
		}
		if req.MerchantNo != "" {
			targetCount++
		}
		if len(req.MerchantNos) > 0 {
			targetCount++
		}
		if targetCount != 1 {
			result.GResult(c, 601, nil, "必须且只能指定一种目标：broadcast 或 merchant_no 或 merchant_nos")
			return
		}

		if req.Broadcast {
			merchants, err := dbhelper.FindAllMerchants()
			if err != nil {
				result.GErr(c, err)
				return
			}
			successCount := 0
			var failures []gin.H
			for _, m := range merchants {
				if err := merchantService.SaveAdminmSmsConfig(m.No, req.Config); err != nil {
					logx.Errorf("广播短信配置失败: merchant=%s, err=%v", m.No, err)
					failures = append(failures, gin.H{"merchant_no": m.No, "name": m.Name, "error": err.Error()})
				} else {
					successCount++
				}
			}
			result.GOK(c, gin.H{"mode": "broadcast", "total": len(merchants), "success": successCount, "failures": failures})
			return
		}

		if req.MerchantNo != "" {
			if err := merchantService.SaveAdminmSmsConfig(req.MerchantNo, req.Config); err != nil {
				result.GResult(c, 500, nil, err.Error())
				return
			}
			result.GOK(c, gin.H{"updated": 1})
			return
		}

		// 批量
		successCount := 0
		var failures []gin.H
		for _, no := range req.MerchantNos {
			no = strings.TrimSpace(no)
			if no == "" {
				continue
			}
			if err := merchantService.SaveAdminmSmsConfig(no, req.Config); err != nil {
				logx.Errorf("批量保存短信配置失败: merchant=%s, err=%v", no, err)
				failures = append(failures, gin.H{"merchant_no": no, "error": err.Error()})
			} else {
				successCount++
			}
		}
		result.GOK(c, gin.H{"updated": successCount, "total": len(req.MerchantNos), "failures": failures})
	})

	// 读取商户测试验证码
	group.GET("test_sms_code", func(c *gin.Context) {
		merchantNo := c.Query("merchant_no")
		if merchantNo == "" {
			result.GParamErr(c, fmt.Errorf("merchant_no不能为空"))
			return
		}
		code, err := merchantService.GetAdminmTestSmsCode(merchantNo)
		if err != nil {
			result.GErr(c, err)
			return
		}
		result.GOK(c, gin.H{"test_sms_code": code})
	})

	// 保存测试验证码：支持单个、批量或全部
	type testSmsCodeReq struct {
		MerchantNo  string `json:"merchant_no"`
		MerchantNos []string `json:"merchant_nos"`
		Broadcast   bool   `json:"broadcast"`
		TestSmsCode string `json:"test_sms_code"`
	}
	group.POST("test_sms_code", func(c *gin.Context) {
		var req testSmsCodeReq
		if err := c.ShouldBindJSON(&req); err != nil {
			result.GParamErr(c, err)
			return
		}
		// 目标判断：三选一
		targetCount := 0
		if req.Broadcast {
			targetCount++
		}
		if req.MerchantNo != "" {
			targetCount++
		}
		if len(req.MerchantNos) > 0 {
			targetCount++
		}
		if targetCount != 1 {
			result.GResult(c, 601, nil, "必须且只能指定一种目标：broadcast 或 merchant_no 或 merchant_nos")
			return
		}

		if req.Broadcast {
			merchants, err := dbhelper.FindAllMerchants()
			if err != nil {
				result.GErr(c, err)
				return
			}
			successCount := 0
			var failures []gin.H
			for _, m := range merchants {
				if err := merchantService.SaveAdminmTestSmsCode(m.No, req.TestSmsCode); err != nil {
					logx.Errorf("广播测试验证码失败: merchant=%s, err=%v", m.No, err)
					failures = append(failures, gin.H{"merchant_no": m.No, "name": m.Name, "error": err.Error()})
				} else {
					successCount++
				}
			}
			result.GOK(c, gin.H{"mode": "broadcast", "total": len(merchants), "success": successCount, "failures": failures})
			return
		}

		if req.MerchantNo != "" {
			if err := merchantService.SaveAdminmTestSmsCode(req.MerchantNo, req.TestSmsCode); err != nil {
				result.GResult(c, 500, nil, err.Error())
				return
			}
			result.GOK(c, gin.H{"updated": 1})
			return
		}

		// 批量
		successCount := 0
		var failures []gin.H
		for _, no := range req.MerchantNos {
			no = strings.TrimSpace(no)
			if no == "" {
				continue
			}
			if err := merchantService.SaveAdminmTestSmsCode(no, req.TestSmsCode); err != nil {
				logx.Errorf("批量保存测试验证码失败: merchant=%s, err=%v", no, err)
				failures = append(failures, gin.H{"merchant_no": no, "error": err.Error()})
			} else {
				successCount++
			}
		}
		result.GOK(c, gin.H{"updated": successCount, "total": len(req.MerchantNos), "failures": failures})
	})

	// 推送 Logo 到商户 tsdd-web 容器：支持单个、批量或全部
	type pushLogoReq struct {
		MerchantNo  string   `json:"merchant_no"`
		MerchantNos []string `json:"merchant_nos"`
		Broadcast   bool     `json:"broadcast"`
		LogoURL     string   `json:"logo_url"`     // Control 资源路径，如 /assets/logo/xxx.png
		UseOwnLogo  bool     `json:"use_own_logo"` // 使用每个商户自己的 logo_url
	}
	group.POST("push_logo", func(c *gin.Context) {
		var req pushLogoReq
		if err := c.ShouldBindJSON(&req); err != nil {
			result.GParamErr(c, err)
			return
		}
		if req.LogoURL == "" && !req.UseOwnLogo {
			result.GResult(c, 601, nil, "logo_url不能为空")
			return
		}

		// 解析 logo_url → 本地文件路径
		resolveLogoPath := func(logoURL string) string {
			rel := strings.TrimPrefix(logoURL, "/assets/")
			return filepath.Join(consts.AssetsDir, rel)
		}
		logoPath := ""
		if req.LogoURL != "" {
			logoPath = resolveLogoPath(req.LogoURL)
		}

		// 目标判断：三选一
		targetCount := 0
		if req.Broadcast {
			targetCount++
		}
		if req.MerchantNo != "" {
			targetCount++
		}
		if len(req.MerchantNos) > 0 {
			targetCount++
		}
		if targetCount != 1 {
			result.GResult(c, 601, nil, "必须且只能指定一种目标：broadcast 或 merchant_no 或 merchant_nos")
			return
		}

		// 辅助函数：获取商户 app_name，空则 fallback 到商户名称
		getAppName := func(merchantNo string) string {
			m, err := dbhelper.GetMerchantByNo(merchantNo)
			if err == nil && m != nil {
				if m.AppName != "" {
					return m.AppName
				}
				return m.Name
			}
			return ""
		}

		if req.Broadcast {
			merchants, err := dbhelper.FindAllMerchants()
			if err != nil {
				result.GErr(c, err)
				return
			}
			successCount := 0
			var skipped []gin.H
			var failures []gin.H
			for _, m := range merchants {
				appName := m.AppName
				if appName == "" {
					appName = m.Name
				}
				mLogoPath := logoPath
				if req.UseOwnLogo {
					if m.LogoUrl == "" {
						skipped = append(skipped, gin.H{"merchant_no": m.No, "name": m.Name, "reason": "未上传Logo"})
						continue
					}
					mLogoPath = resolveLogoPath(m.LogoUrl)
				}
				if err := merchantService.PushWebLogo(m.No, mLogoPath, appName); err != nil {
					logx.Errorf("广播推送Logo失败: merchant=%s, err=%v", m.No, err)
					failures = append(failures, gin.H{"merchant_no": m.No, "name": m.Name, "error": err.Error()})
				} else {
					successCount++
				}
			}
			result.GOK(c, gin.H{"mode": "broadcast", "total": len(merchants), "success": successCount, "failures": failures, "skipped": skipped})
			return
		}

		if req.MerchantNo != "" {
			appName := getAppName(req.MerchantNo)
			if err := merchantService.PushWebLogo(req.MerchantNo, logoPath, appName); err != nil {
				result.GResult(c, 500, nil, err.Error())
				return
			}
			result.GOK(c, gin.H{"pushed": 1})
			return
		}

		// 批量
		successCount := 0
		var failures []gin.H
		var skipped []gin.H
		for _, no := range req.MerchantNos {
			no = strings.TrimSpace(no)
			if no == "" {
				continue
			}
			mLogoPath := logoPath
			appName := getAppName(no)
			if req.UseOwnLogo {
				m, err := dbhelper.GetMerchantByNo(no)
				if err != nil || m == nil {
					failures = append(failures, gin.H{"merchant_no": no, "error": "商户不存在"})
					continue
				}
				if m.LogoUrl == "" {
					skipped = append(skipped, gin.H{"merchant_no": no, "name": m.Name, "reason": "未上传Logo"})
					continue
				}
				mLogoPath = resolveLogoPath(m.LogoUrl)
			}
			if err := merchantService.PushWebLogo(no, mLogoPath, appName); err != nil {
				logx.Errorf("批量推送Logo失败: merchant=%s, err=%v", no, err)
				failures = append(failures, gin.H{"merchant_no": no, "error": err.Error()})
			} else {
				successCount++
			}
		}
		result.GOK(c, gin.H{"pushed": successCount, "total": len(req.MerchantNos), "failures": failures, "skipped": skipped})
	})

	// 导出商户数据库（mysqldump 流式下载）
	group.POST("export_database", func(c *gin.Context) {
		var req struct {
			MerchantNo string `json:"merchant_no"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			result.GParamErr(c, err)
			return
		}
		if req.MerchantNo == "" {
			result.GResult(c, 601, nil, "merchant_no不能为空")
			return
		}

		// 获取商户 SSH 客户端
		sshClient, err := merchantService.GetMerchantSSHClient(req.MerchantNo)
		if err != nil {
			result.GErr(c, err)
			return
		}
		defer sshClient.Close()

		timestamp := time.Now().Unix()
		tmpFile := fmt.Sprintf("/tmp/tsdd_dump_%d.sql.gz", timestamp)

		// 1. 在商户服务器上执行 mysqldump（--single-transaction 不锁表）
		dumpCmd := fmt.Sprintf(`docker exec tsdd-mysql sh -c 'mysqldump --single-transaction --quick --routines --triggers -u root -p"$MYSQL_ROOT_PASSWORD" --databases $(mysql -u root -p"$MYSQL_ROOT_PASSWORD" -N -e "SHOW DATABASES" 2>/dev/null | grep -vE "^(mysql|information_schema|performance_schema|sys)$" | tr "\n" " ") 2>/dev/null' | gzip > %s`, tmpFile)

		logx.Infof("开始导出商户数据库: merchant=%s", req.MerchantNo)
		output, dumpErr := sshClient.ExecuteCommandWithTimeout(dumpCmd, 300*time.Second)
		if dumpErr != nil {
			// 清理临时文件
			sshClient.ExecuteCommandSilent(fmt.Sprintf("rm -f %s", tmpFile))
			logx.Errorf("mysqldump 失败: merchant=%s, err=%v, output=%s", req.MerchantNo, dumpErr, output)
			result.GResult(c, 500, nil, fmt.Sprintf("数据库导出失败: %v", dumpErr))
			return
		}

		// 2. 获取文件大小
		sizeStr := strings.TrimSpace(sshClient.ExecuteCommandSilent(fmt.Sprintf("stat -c%%s %s 2>/dev/null || stat -f%%z %s 2>/dev/null", tmpFile, tmpFile)))
		fileSize, _ := strconv.ParseInt(sizeStr, 10, 64)
		if fileSize == 0 {
			sshClient.ExecuteCommandSilent(fmt.Sprintf("rm -f %s", tmpFile))
			result.GResult(c, 500, nil, "导出文件为空")
			return
		}

		// 3. 流式读取文件
		reader, session, streamErr := sshClient.ExecuteCommandStream(fmt.Sprintf("cat %s", tmpFile))
		if streamErr != nil {
			sshClient.ExecuteCommandSilent(fmt.Sprintf("rm -f %s", tmpFile))
			result.GResult(c, 500, nil, fmt.Sprintf("读取导出文件失败: %v", streamErr))
			return
		}

		// 4. 设置响应头并流式传输
		filename := fmt.Sprintf("%s_%s.sql.gz", req.MerchantNo, time.Now().Format("20060102_150405"))
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
		c.Header("Content-Type", "application/gzip")
		if fileSize > 0 {
			c.Header("Content-Length", strconv.FormatInt(fileSize, 10))
		}

		c.Status(http.StatusOK)
		_, copyErr := io.Copy(c.Writer, reader)
		session.Wait()
		session.Close()

		// 5. 清理临时文件
		sshClient.ExecuteCommandSilent(fmt.Sprintf("rm -f %s", tmpFile))

		if copyErr != nil {
			logx.Errorf("流式传输失败: merchant=%s, err=%v", req.MerchantNo, copyErr)
		} else {
			logx.Infof("商户数据库导出完成: merchant=%s, size=%d", req.MerchantNo, fileSize)
		}
	})

	// 清除商户数据（需要密码或2FA验证）
	group.POST("clear_data", func(c *gin.Context) {
		var req struct {
			MerchantNo string `json:"merchant_no"`
			Password   string `json:"password"`   // 登录密码验证
			TOTPCode   string `json:"totp_code"`   // 2FA 验证码
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			result.GParamErr(c, err)
			return
		}
		if req.MerchantNo == "" {
			result.GResult(c, 601, nil, "merchant_no不能为空")
			return
		}

		// 身份验证：2FA 或密码
		username := middleware.GetUsername(c)
		user, err := dbhelper.GetSysUserByUsername(username)
		if err != nil {
			result.GResult(c, 401, nil, "获取用户信息失败")
			return
		}
		if user.TwoFactorEnabled == 1 {
			// 启用了2FA，验证 TOTP code
			if req.TOTPCode == "" {
				result.GResult(c, 401, nil, "需要2FA验证码")
				return
			}
			if !auth.VerifyTwoFACode(user.TwoFactorSecret, req.TOTPCode) {
				result.GResult(c, 401, nil, "2FA验证码错误")
				return
			}
		} else {
			// 未启用2FA，验证密码
			if req.Password == "" {
				result.GResult(c, 401, nil, "需要输入登录密码")
				return
			}
			if user.Password != req.Password {
				result.GResult(c, 401, nil, "密码错误")
				return
			}
		}

		if err := merchantService.ClearMerchantData(req.MerchantNo); err != nil {
			result.GErr(c, err)
			return
		}
		result.GOK(c, gin.H{"cleared": true})
	})
}
