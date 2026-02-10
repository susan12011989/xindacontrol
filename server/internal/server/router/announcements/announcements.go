package announcements

import (
	"encoding/json"
	"fmt"
	"server/internal/server/middleware"
	"server/internal/server/service/merchant"
	"server/pkg/dbs"
	"server/pkg/entity"
	"server/pkg/result"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zeromicro/go-zero/core/logx"
)

// Routes 注册公告相关路由（仅发送公告，不含上传）
func Routes(gi gin.IRouter) {
	group := gi.Group("announcements")
	group.Use(middleware.Authorization)
	group.POST("", createAnnouncement)
	group.GET("logs", listAnnouncementLogs)
}

// createAnnouncement 接收公告内容并广播给所有已连接的 adminm 客户端
type createAnnouncementReq struct {
	Text        string                      `json:"text"`
	Entities    []entity.AnnouncementEntity `json:"entities"`
	Silent      bool                        `json:"silent"`
	NoForwards  bool                        `json:"noforwards"`
	MerchantIds []int64                     `json:"merchant_ids"`
	MerchantNos []string                    `json:"merchant_nos"`
}

func createAnnouncement(c *gin.Context) {
	var req createAnnouncementReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.GParamErr(c, err)
		return
	}

	// 默认静音、禁转发
	if !req.Silent {
		req.Silent = true
	}
	if !req.NoForwards {
		req.NoForwards = true
	}

	if strings.TrimSpace(req.Text) == "" {
		result.GResult(c, 400, nil, "text不能为空")
		return
	}

	// 指定商户发送（优先 merchant_nos，其次 merchant_ids）；都为空则广播
	selectedNos := make([]string, 0)
	if len(req.MerchantNos) > 0 {
		// 去重并逐个发送
		seen := map[string]struct{}{}
		for _, no := range req.MerchantNos {
			no = strings.TrimSpace(no)
			if no == "" {
				continue
			}
			if _, ok := seen[no]; ok {
				continue
			}
			seen[no] = struct{}{}
			selectedNos = append(selectedNos, no)
		}
	} else if len(req.MerchantIds) > 0 {
		// 查询对应 no 并发送
		var items []entity.Merchants
		if err := dbs.DBAdmin.In("id", req.MerchantIds).Cols("no").Find(&items); err != nil {
			result.GErr(c, err)
			return
		}
		if len(items) == 0 {
			result.GResult(c, 400, nil, "未找到指定商户")
			return
		}
		seen := map[string]struct{}{}
		for i := range items {
			no := strings.TrimSpace(items[i].No)
			if no == "" {
				continue
			}
			if _, ok := seen[no]; ok {
				continue
			}
			seen[no] = struct{}{}
			selectedNos = append(selectedNos, no)
		}
	} else {
		// 未指定则广播
	}

	// 记录日志
	entitiesJSON, _ := json.Marshal(req.Entities)
	nosJSON, _ := json.Marshal(selectedNos)
	logItem := &entity.AnnouncementLogs{
		Text:        req.Text,
		Entities:    string(entitiesJSON),
		Silent:      boolToInt(req.Silent),
		NoForwards:  boolToInt(req.NoForwards),
		MerchantNos: string(nosJSON),
		Broadcast:   boolToInt(len(selectedNos) == 0),
		CreatedAt:   time.Now(),
	}
	_, _ = dbs.DBAdmin.Insert(logItem)

	// 向商户发送公告
	announcementReq := &entity.AnnouncementReq{
		Text:       req.Text,
		Entities:   req.Entities,
		Silent:     req.Silent,
		NoForwards: req.NoForwards,
	}

	if len(selectedNos) == 0 {
		// 广播：查询所有商户
		var merchants []entity.Merchants
		if err := dbs.DBAdmin.Where("status = ?", 1).Cols("no").Find(&merchants); err != nil {
			logx.Errorf("查询商户列表失败: %v", err)
		} else {
			// 异步发送给所有商户
			go func() {
				for _, m := range merchants {
					if err := merchant.SendAnnouncement(m.No, announcementReq); err != nil {
						logx.Errorf("发送公告到商户失败: merchant=%s, err=%v", m.No, err)
					}
				}
				logx.Infof("公告广播完成: 商户数=%d", len(merchants))
			}()
		}
	} else {
		// 指定商户发送
		go func() {
			for _, no := range selectedNos {
				if err := merchant.SendAnnouncement(no, announcementReq); err != nil {
					logx.Errorf("发送公告到商户失败: merchant=%s, err=%v", no, err)
				}
			}
			logx.Infof("公告发送完成: 商户数=%d", len(selectedNos))
		}()
	}

	result.GOK(c, gin.H{"accepted": true})
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// 查询公告发送日志
func listAnnouncementLogs(c *gin.Context) {
	pageStr := c.DefaultQuery("page", "1")
	sizeStr := c.DefaultQuery("size", "10")
	page := atoiSafe(pageStr, 1)
	size := atoiSafe(sizeStr, 10)
	if page <= 0 {
		page = 1
	}
	if size <= 0 || size > 100 {
		size = 10
	}

	session := dbs.DBAdmin.Where("1=1").OrderBy("id desc")
	var rows []entity.AnnouncementLogs
	total, err := session.Limit(size, (page-1)*size).FindAndCount(&rows)
	if err != nil {
		result.GErr(c, err)
		return
	}

	// 映射为前端友好结构
	type LogView struct {
		Id          int64    `json:"id"`
		Text        string   `json:"text"`
		MerchantNos []string `json:"merchant_nos"`
		Broadcast   bool     `json:"broadcast"`
		CreatedAt   string   `json:"created_at"`
	}
	list := make([]LogView, 0, len(rows))
	for i := range rows {
		var nos []string
		_ = json.Unmarshal([]byte(rows[i].MerchantNos), &nos)
		list = append(list, LogView{
			Id:          rows[i].Id,
			Text:        rows[i].Text,
			MerchantNos: nos,
			Broadcast:   rows[i].Broadcast == 1,
			CreatedAt:   rows[i].CreatedAt.Format(time.DateTime),
		})
	}

	result.GOK(c, gin.H{"list": list, "total": total})
}

func atoiSafe(s string, def int) int {
	var n int
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c < '0' || c > '9' {
			return def
		}
	}
	_, _ = fmt.Sscanf(s, "%d", &n)
	if n == 0 {
		return def
	}
	return n
}
