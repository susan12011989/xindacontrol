package audit

import (
	"encoding/json"
	"server/pkg/dbs"
	"server/pkg/entity"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zeromicro/go-zero/core/logx"
)

// LogEntry 审计日志条目
type LogEntry struct {
	UserId     int
	Username   string
	Action     string
	TargetType string
	TargetId   int
	TargetName string
	Detail     interface{} // 会被 JSON 序列化
	IP         string
	UserAgent  string
	Status     string // success, failed
	ErrorMsg   string
}

// LogFromContext 从 gin.Context 创建审计日志（自动填充用户信息）
func LogFromContext(c *gin.Context, action, targetType string, targetId int, targetName string, detail interface{}) {
	entry := LogEntry{
		UserId:     int(c.GetInt64("uid")),
		Username:   c.GetString("username"),
		Action:     action,
		TargetType: targetType,
		TargetId:   targetId,
		TargetName: targetName,
		Detail:     detail,
		IP:         c.ClientIP(),
		UserAgent:  c.GetHeader("User-Agent"),
		Status:     "success",
	}
	Log(entry)
}

// LogErrorFromContext 从 gin.Context 创建失败的审计日志
func LogErrorFromContext(c *gin.Context, action, targetType string, targetId int, targetName string, detail interface{}, errMsg string) {
	entry := LogEntry{
		UserId:     int(c.GetInt64("uid")),
		Username:   c.GetString("username"),
		Action:     action,
		TargetType: targetType,
		TargetId:   targetId,
		TargetName: targetName,
		Detail:     detail,
		IP:         c.ClientIP(),
		UserAgent:  c.GetHeader("User-Agent"),
		Status:     "failed",
		ErrorMsg:   errMsg,
	}
	Log(entry)
}

// Log 记录审计日志
func Log(entry LogEntry) {
	go func() {
		if err := logAsync(entry); err != nil {
			logx.Errorf("audit log error: %v", err)
		}
	}()
}

// LogSync 同步记录审计日志（用于需要确保记录成功的场景）
func LogSync(entry LogEntry) error {
	return logAsync(entry)
}

func logAsync(entry LogEntry) error {
	var detailStr string
	if entry.Detail != nil {
		if bytes, err := json.Marshal(entry.Detail); err == nil {
			detailStr = string(bytes)
		}
	}

	if entry.Status == "" {
		entry.Status = "success"
	}

	log := &entity.AuditLogs{
		UserId:     entry.UserId,
		Username:   entry.Username,
		Action:     entry.Action,
		TargetType: entry.TargetType,
		TargetId:   entry.TargetId,
		TargetName: entry.TargetName,
		Detail:     detailStr,
		IP:         entry.IP,
		UserAgent:  entry.UserAgent,
		Status:     entry.Status,
		ErrorMsg:   entry.ErrorMsg,
		CreatedAt:  time.Now(),
	}

	_, err := dbs.DBAdmin.Insert(log)
	return err
}

// QueryReq 审计日志查询请求
type QueryReq struct {
	Page       int    `form:"page" json:"page"`
	Size       int    `form:"size" json:"size"`
	UserId     int    `form:"user_id" json:"user_id"`
	Action     string `form:"action" json:"action"`
	TargetType string `form:"target_type" json:"target_type"`
	TargetId   int    `form:"target_id" json:"target_id"`
	Status     string `form:"status" json:"status"`
	StartTime  string `form:"start_time" json:"start_time"` // 格式: 2006-01-02 15:04:05
	EndTime    string `form:"end_time" json:"end_time"`
}

// QueryResp 审计日志响应
type QueryResp struct {
	Id         int64  `json:"id"`
	UserId     int    `json:"user_id"`
	Username   string `json:"username"`
	Action     string `json:"action"`
	ActionName string `json:"action_name"` // 操作名称（中文）
	TargetType string `json:"target_type"`
	TargetId   int    `json:"target_id"`
	TargetName string `json:"target_name"`
	Detail     string `json:"detail"`
	IP         string `json:"ip"`
	Status     string `json:"status"`
	ErrorMsg   string `json:"error_msg"`
	CreatedAt  string `json:"created_at"`
}

// Query 查询审计日志
func Query(req QueryReq) ([]QueryResp, int64, error) {
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.Size <= 0 {
		req.Size = 20
	}

	session := dbs.DBAdmin.Table("audit_logs")

	if req.UserId > 0 {
		session = session.Where("user_id = ?", req.UserId)
	}
	if req.Action != "" {
		session = session.Where("action = ?", req.Action)
	}
	if req.TargetType != "" {
		session = session.Where("target_type = ?", req.TargetType)
	}
	if req.TargetId > 0 {
		session = session.Where("target_id = ?", req.TargetId)
	}
	if req.Status != "" {
		session = session.Where("status = ?", req.Status)
	}
	if req.StartTime != "" {
		session = session.Where("created_at >= ?", req.StartTime)
	}
	if req.EndTime != "" {
		session = session.Where("created_at <= ?", req.EndTime)
	}

	offset := (req.Page - 1) * req.Size
	var logs []entity.AuditLogs
	total, err := session.Desc("id").Limit(req.Size, offset).FindAndCount(&logs)
	if err != nil {
		return nil, 0, err
	}

	result := make([]QueryResp, len(logs))
	for i, l := range logs {
		result[i] = QueryResp{
			Id:         l.Id,
			UserId:     l.UserId,
			Username:   l.Username,
			Action:     l.Action,
			ActionName: getActionName(l.Action),
			TargetType: l.TargetType,
			TargetId:   l.TargetId,
			TargetName: l.TargetName,
			Detail:     l.Detail,
			IP:         l.IP,
			Status:     l.Status,
			ErrorMsg:   l.ErrorMsg,
			CreatedAt:  l.CreatedAt.Format(time.DateTime),
		}
	}

	return result, total, nil
}

// getActionName 获取操作名称（中文）
func getActionName(action string) string {
	names := map[string]string{
		entity.AuditActionCreateMerchant:  "创建商户",
		entity.AuditActionUpdateMerchant:  "更新商户",
		entity.AuditActionDeleteMerchant:  "删除商户",
		entity.AuditActionChangeMerchantIP:    "更换商户IP",
		entity.AuditActionChangeGostPort:      "更换GOST端口",
		entity.AuditActionCreateServer:    "创建服务器",
		entity.AuditActionUpdateServer:    "更新服务器",
		entity.AuditActionDeleteServer:    "删除服务器",
		entity.AuditActionCreateCloudAccount: "创建云账号",
		entity.AuditActionUpdateCloudAccount: "更新云账号",
		entity.AuditActionDeleteCloudAccount: "删除云账号",
		entity.AuditActionCreateOssConfig: "创建OSS配置",
		entity.AuditActionUpdateOssConfig: "更新OSS配置",
		entity.AuditActionDeleteOssConfig: "删除OSS配置",
		entity.AuditActionCreateGostServer: "创建GOST服务器",
		entity.AuditActionUpdateGostServer: "更新GOST服务器",
		entity.AuditActionDeleteGostServer: "删除GOST服务器",
		entity.AuditActionLogin:  "登录",
		entity.AuditActionLogout: "登出",
	}
	if name, ok := names[action]; ok {
		return name
	}
	return action
}

// GetActionOptions 获取操作类型选项（用于下拉框）
func GetActionOptions() []map[string]string {
	return []map[string]string{
		{"value": entity.AuditActionCreateMerchant, "label": "创建商户"},
		{"value": entity.AuditActionUpdateMerchant, "label": "更新商户"},
		{"value": entity.AuditActionDeleteMerchant, "label": "删除商户"},
		{"value": entity.AuditActionChangeMerchantIP, "label": "更换商户IP"},
		{"value": entity.AuditActionChangeGostPort, "label": "更换GOST端口"},
		{"value": entity.AuditActionCreateServer, "label": "创建服务器"},
		{"value": entity.AuditActionUpdateServer, "label": "更新服务器"},
		{"value": entity.AuditActionDeleteServer, "label": "删除服务器"},
		{"value": entity.AuditActionCreateCloudAccount, "label": "创建云账号"},
		{"value": entity.AuditActionUpdateCloudAccount, "label": "更新云账号"},
		{"value": entity.AuditActionDeleteCloudAccount, "label": "删除云账号"},
		{"value": entity.AuditActionLogin, "label": "登录"},
		{"value": entity.AuditActionLogout, "label": "登出"},
	}
}

// GetTargetTypeOptions 获取目标类型选项（用于下拉框）
func GetTargetTypeOptions() []map[string]string {
	return []map[string]string{
		{"value": entity.AuditTargetMerchant, "label": "商户"},
		{"value": entity.AuditTargetServer, "label": "服务器"},
		{"value": entity.AuditTargetCloudAccount, "label": "云账号"},
		{"value": entity.AuditTargetOssConfig, "label": "OSS配置"},
		{"value": entity.AuditTargetGostServer, "label": "GOST服务器"},
		{"value": entity.AuditTargetUser, "label": "用户"},
	}
}
