package merchant

import (
	"server/internal/dbhelper"
	"server/internal/server/middleware"
	merchantService "server/internal/server/service/merchant"
	"server/pkg/entity"
	"server/pkg/result"
	"strings"

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
			for _, m := range merchants {
				if err := merchantService.SaveAdminmSensitiveContents(m.No, contents); err != nil {
					logx.Errorf("广播敏感词配置失败: merchant=%s, err=%v", m.No, err)
				} else {
					successCount++
				}
			}
			result.GOK(c, gin.H{"mode": "broadcast", "count": len(contents), "total": len(merchants), "success": successCount})
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
		for _, no := range req.MerchantNos {
			no = strings.TrimSpace(no)
			if no == "" {
				continue
			}
			if err := merchantService.SaveAdminmSensitiveContents(no, contents); err != nil {
				logx.Errorf("批量保存敏感词配置失败: merchant=%s, err=%v", no, err)
			} else {
				successCount++
			}
		}
		result.GOK(c, gin.H{"updated": successCount, "count": len(contents), "total": len(req.MerchantNos)})
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
			for _, m := range merchants {
				if err := merchantService.SaveAdminmSystemNickname(m.No, req.FirstName); err != nil {
					logx.Errorf("广播系统昵称失败: merchant=%s, err=%v", m.No, err)
				} else {
					successCount++
				}
			}
			result.GOK(c, gin.H{"mode": "broadcast", "total": len(merchants), "success": successCount})
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
		for _, no := range req.MerchantNos {
			no = strings.TrimSpace(no)
			if no == "" {
				continue
			}
			if err := merchantService.SaveAdminmSystemNickname(no, req.FirstName); err != nil {
				logx.Errorf("批量保存系统昵称失败: merchant=%s, err=%v", no, err)
			} else {
				successCount++
			}
		}
		result.GOK(c, gin.H{"updated": successCount, "total": len(req.MerchantNos)})
	})

	// 保存短信配置：支持单个、批量或全部
	type smsSaveReq struct {
		MerchantNo  string            `json:"merchant_no"`
		MerchantNos []string          `json:"merchant_nos"`
		Broadcast   bool              `json:"broadcast"`
		Config      *entity.SmsConfig `json:"config"`
	}
	group.POST("sms_config", func(c *gin.Context) {
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
			for _, m := range merchants {
				if err := merchantService.SaveAdminmSmsConfig(m.No, req.Config); err != nil {
					logx.Errorf("广播短信配置失败: merchant=%s, err=%v", m.No, err)
				} else {
					successCount++
				}
			}
			result.GOK(c, gin.H{"mode": "broadcast", "total": len(merchants), "success": successCount})
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
		for _, no := range req.MerchantNos {
			no = strings.TrimSpace(no)
			if no == "" {
				continue
			}
			if err := merchantService.SaveAdminmSmsConfig(no, req.Config); err != nil {
				logx.Errorf("批量保存短信配置失败: merchant=%s, err=%v", no, err)
			} else {
				successCount++
			}
		}
		result.GOK(c, gin.H{"updated": successCount, "total": len(req.MerchantNos)})
	})
}
