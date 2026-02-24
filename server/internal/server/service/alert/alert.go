package alert

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"server/pkg/dbs"
	"server/pkg/entity"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

// ========== 告警规则管理 ==========

// RuleReq 告警规则请求
type RuleReq struct {
	Id              int     `json:"id"`
	Name            string  `json:"name" binding:"required"`
	Type            string  `json:"type" binding:"required"`
	Threshold       float64 `json:"threshold"`
	MerchantId      int     `json:"merchant_id"`
	NotifyType      string  `json:"notify_type" binding:"required"`
	NotifyUrl       string  `json:"notify_url"`
	NotifyEmail     string  `json:"notify_email"`
	NotifyPhone     string  `json:"notify_phone"`
	IntervalMinutes int     `json:"interval_minutes"`
	Status          int     `json:"status"`
	Description     string  `json:"description"`
}

// RuleResp 告警规则响应
type RuleResp struct {
	Id              int     `json:"id"`
	Name            string  `json:"name"`
	Type            string  `json:"type"`
	TypeName        string  `json:"type_name"`
	Threshold       float64 `json:"threshold"`
	MerchantId      int     `json:"merchant_id"`
	NotifyType      string  `json:"notify_type"`
	NotifyUrl       string  `json:"notify_url"`
	NotifyEmail     string  `json:"notify_email"`
	NotifyPhone     string  `json:"notify_phone"`
	IntervalMinutes int     `json:"interval_minutes"`
	Status          int     `json:"status"`
	Description     string  `json:"description"`
	CreatedAt       string  `json:"created_at"`
	UpdatedAt       string  `json:"updated_at"`
}

// QueryRulesReq 查询规则请求
type QueryRulesReq struct {
	Page       int    `form:"page" json:"page"`
	Size       int    `form:"size" json:"size"`
	Type       string `form:"type" json:"type"`
	MerchantId int    `form:"merchant_id" json:"merchant_id"`
	Status     *int   `form:"status" json:"status"`
}

// ListRules 查询告警规则列表
func ListRules(req QueryRulesReq) ([]RuleResp, int64, error) {
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.Size <= 0 {
		req.Size = 20
	}

	session := dbs.DBAdmin.Table("alert_rules")

	if req.Type != "" {
		session = session.Where("type = ?", req.Type)
	}
	if req.MerchantId > 0 {
		session = session.Where("merchant_id = ?", req.MerchantId)
	}
	if req.Status != nil {
		session = session.Where("status = ?", *req.Status)
	}

	offset := (req.Page - 1) * req.Size
	var rules []entity.AlertRules
	total, err := session.Desc("id").Limit(req.Size, offset).FindAndCount(&rules)
	if err != nil {
		return nil, 0, err
	}

	result := make([]RuleResp, len(rules))
	for i, r := range rules {
		result[i] = RuleResp{
			Id:              r.Id,
			Name:            r.Name,
			Type:            r.Type,
			TypeName:        getAlertTypeName(r.Type),
			Threshold:       r.Threshold,
			MerchantId:      r.MerchantId,
			NotifyType:      r.NotifyType,
			NotifyUrl:       r.NotifyUrl,
			NotifyEmail:     r.NotifyEmail,
			NotifyPhone:     r.NotifyPhone,
			IntervalMinutes: r.IntervalMinutes,
			Status:          r.Status,
			Description:     r.Description,
			CreatedAt:       r.CreatedAt.Format(time.DateTime),
			UpdatedAt:       r.UpdatedAt.Format(time.DateTime),
		}
	}

	return result, total, nil
}

// CreateRule 创建告警规则
func CreateRule(req RuleReq) (int, error) {
	if req.IntervalMinutes <= 0 {
		req.IntervalMinutes = 60
	}

	rule := &entity.AlertRules{
		Name:            req.Name,
		Type:            req.Type,
		Threshold:       req.Threshold,
		MerchantId:      req.MerchantId,
		NotifyType:      req.NotifyType,
		NotifyUrl:       req.NotifyUrl,
		NotifyEmail:     req.NotifyEmail,
		NotifyPhone:     req.NotifyPhone,
		IntervalMinutes: req.IntervalMinutes,
		Status:          1,
		Description:     req.Description,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	_, err := dbs.DBAdmin.Insert(rule)
	if err != nil {
		return 0, err
	}
	return rule.Id, nil
}

// UpdateRule 更新告警规则
func UpdateRule(id int, req RuleReq) error {
	updates := map[string]interface{}{
		"name":             req.Name,
		"type":             req.Type,
		"threshold":        req.Threshold,
		"merchant_id":      req.MerchantId,
		"notify_type":      req.NotifyType,
		"notify_url":       req.NotifyUrl,
		"notify_email":     req.NotifyEmail,
		"notify_phone":     req.NotifyPhone,
		"interval_minutes": req.IntervalMinutes,
		"description":      req.Description,
		"updated_at":       time.Now(),
	}

	if req.Status != 0 {
		updates["status"] = req.Status
	}

	_, err := dbs.DBAdmin.Table("alert_rules").Where("id = ?", id).Update(updates)
	return err
}

// DeleteRule 删除告警规则
func DeleteRule(id int) error {
	_, err := dbs.DBAdmin.ID(id).Delete(&entity.AlertRules{})
	return err
}

// ToggleRuleStatus 切换规则状态
func ToggleRuleStatus(id int) error {
	var rule entity.AlertRules
	has, err := dbs.DBAdmin.ID(id).Get(&rule)
	if err != nil {
		return err
	}
	if !has {
		return fmt.Errorf("规则不存在")
	}

	newStatus := 1
	if rule.Status == 1 {
		newStatus = 0
	}

	_, err = dbs.DBAdmin.Table("alert_rules").Where("id = ?", id).Update(map[string]interface{}{
		"status":     newStatus,
		"updated_at": time.Now(),
	})
	return err
}

// ========== 告警日志管理 ==========

// LogResp 告警日志响应
type LogResp struct {
	Id           int64  `json:"id"`
	RuleId       int    `json:"rule_id"`
	RuleName     string `json:"rule_name"`
	Type         string `json:"type"`
	TypeName     string `json:"type_name"`
	Level        string `json:"level"`
	TargetType   string `json:"target_type"`
	TargetId     int    `json:"target_id"`
	TargetName   string `json:"target_name"`
	Message      string `json:"message"`
	Detail       string `json:"detail"`
	NotifyStatus string `json:"notify_status"`
	NotifyResult string `json:"notify_result"`
	CreatedAt    string `json:"created_at"`
}

// QueryLogsReq 查询日志请求
type QueryLogsReq struct {
	Page         int    `form:"page" json:"page"`
	Size         int    `form:"size" json:"size"`
	Type         string `form:"type" json:"type"`
	Level        string `form:"level" json:"level"`
	TargetType   string `form:"target_type" json:"target_type"`
	TargetId     int    `form:"target_id" json:"target_id"`
	NotifyStatus string `form:"notify_status" json:"notify_status"`
	StartTime    string `form:"start_time" json:"start_time"`
	EndTime      string `form:"end_time" json:"end_time"`
}

// ListLogs 查询告警日志列表
func ListLogs(req QueryLogsReq) ([]LogResp, int64, error) {
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.Size <= 0 {
		req.Size = 20
	}

	session := dbs.DBAdmin.Table("alert_logs")

	if req.Type != "" {
		session = session.Where("type = ?", req.Type)
	}
	if req.Level != "" {
		session = session.Where("level = ?", req.Level)
	}
	if req.TargetType != "" {
		session = session.Where("target_type = ?", req.TargetType)
	}
	if req.TargetId > 0 {
		session = session.Where("target_id = ?", req.TargetId)
	}
	if req.NotifyStatus != "" {
		session = session.Where("notify_status = ?", req.NotifyStatus)
	}
	if req.StartTime != "" {
		session = session.Where("created_at >= ?", req.StartTime)
	}
	if req.EndTime != "" {
		session = session.Where("created_at <= ?", req.EndTime)
	}

	offset := (req.Page - 1) * req.Size
	var logs []entity.AlertLogs
	total, err := session.Desc("id").Limit(req.Size, offset).FindAndCount(&logs)
	if err != nil {
		return nil, 0, err
	}

	result := make([]LogResp, len(logs))
	for i, l := range logs {
		result[i] = LogResp{
			Id:           l.Id,
			RuleId:       l.RuleId,
			RuleName:     l.RuleName,
			Type:         l.Type,
			TypeName:     getAlertTypeName(l.Type),
			Level:        l.Level,
			TargetType:   l.TargetType,
			TargetId:     l.TargetId,
			TargetName:   l.TargetName,
			Message:      l.Message,
			Detail:       l.Detail,
			NotifyStatus: l.NotifyStatus,
			NotifyResult: l.NotifyResult,
			CreatedAt:    l.CreatedAt.Format(time.DateTime),
		}
	}

	return result, total, nil
}

// ========== 告警触发 ==========

// TriggerAlert 触发告警
func TriggerAlert(rule *entity.AlertRules, level, targetType string, targetId int, targetName, message string, detail interface{}) {
	go func() {
		// 检查是否在告警间隔内
		if !shouldAlert(rule.Id, rule.IntervalMinutes) {
			logx.Infof("alert rule %d is in cooldown, skip", rule.Id)
			return
		}

		var detailStr string
		if detail != nil {
			if bytes, err := json.Marshal(detail); err == nil {
				detailStr = string(bytes)
			}
		}

		// 创建告警日志
		log := &entity.AlertLogs{
			RuleId:       rule.Id,
			RuleName:     rule.Name,
			Type:         rule.Type,
			Level:        level,
			TargetType:   targetType,
			TargetId:     targetId,
			TargetName:   targetName,
			Message:      message,
			Detail:       detailStr,
			NotifyStatus: entity.NotifyStatusPending,
			CreatedAt:    time.Now(),
		}

		_, err := dbs.DBAdmin.Insert(log)
		if err != nil {
			logx.Errorf("insert alert log error: %v", err)
			return
		}

		// 发送通知
		notifyResult, notifyErr := sendNotification(rule, log)
		notifyStatus := entity.NotifyStatusSent
		if notifyErr != nil {
			notifyStatus = entity.NotifyStatusFailed
			notifyResult = notifyErr.Error()
		}

		// 更新通知状态
		_, _ = dbs.DBAdmin.Table("alert_logs").Where("id = ?", log.Id).Update(map[string]interface{}{
			"notify_status": notifyStatus,
			"notify_result": notifyResult,
		})
	}()
}

// shouldAlert 检查是否应该发送告警（基于间隔时间）
func shouldAlert(ruleId, intervalMinutes int) bool {
	if intervalMinutes <= 0 {
		return true
	}

	var lastLog entity.AlertLogs
	has, err := dbs.DBAdmin.Where("rule_id = ?", ruleId).Desc("id").Get(&lastLog)
	if err != nil || !has {
		return true
	}

	// 检查最后一次告警时间
	intervalDuration := time.Duration(intervalMinutes) * time.Minute
	return time.Since(lastLog.CreatedAt) >= intervalDuration
}

// sendNotification 发送通知
func sendNotification(rule *entity.AlertRules, log *entity.AlertLogs) (string, error) {
	switch rule.NotifyType {
	case entity.NotifyTypeWebhook:
		return sendWebhook(rule.NotifyUrl, log)
	case entity.NotifyTypeEmail:
		return sendEmail(rule.NotifyEmail, log)
	case entity.NotifyTypeSms:
		return sendSms(rule.NotifyPhone, log)
	default:
		return "", fmt.Errorf("unsupported notify type: %s", rule.NotifyType)
	}
}

// sendWebhook 发送 Webhook 通知
func sendWebhook(url string, log *entity.AlertLogs) (string, error) {
	if url == "" {
		return "", fmt.Errorf("webhook url is empty")
	}

	payload := map[string]interface{}{
		"alert_id":    log.Id,
		"rule_id":     log.RuleId,
		"rule_name":   log.RuleName,
		"type":        log.Type,
		"level":       log.Level,
		"target_type": log.TargetType,
		"target_id":   log.TargetId,
		"target_name": log.TargetName,
		"message":     log.Message,
		"detail":      log.Detail,
		"timestamp":   log.CreatedAt.Format(time.RFC3339),
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return string(respBody), fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	return fmt.Sprintf("status=%d, body=%s", resp.StatusCode, string(respBody)), nil
}

// sendEmail 发送邮件通知（待实现）
func sendEmail(email string, log *entity.AlertLogs) (string, error) {
	if email == "" {
		return "", fmt.Errorf("email is empty")
	}
	// TODO: 实现邮件发送
	return "email sending not implemented", nil
}

// sendSms 发送短信通知（待实现）
func sendSms(phone string, log *entity.AlertLogs) (string, error) {
	if phone == "" {
		return "", fmt.Errorf("phone is empty")
	}
	// TODO: 实现短信发送
	return "sms sending not implemented", nil
}

// ========== 辅助函数 ==========

// getAlertTypeName 获取告警类型名称
func getAlertTypeName(t string) string {
	names := map[string]string{
		entity.AlertTypeMerchantExpired: "商户即将过期",
		entity.AlertTypeServerDown:      "服务器宕机",
		entity.AlertTypeCpuHigh:         "CPU使用率过高",
		entity.AlertTypeMemoryHigh:      "内存使用率过高",
		entity.AlertTypeDiskHigh:        "磁盘使用率过高",
		entity.AlertTypeServiceDown:     "服务异常",
		entity.AlertTypeGostDown:           "GOST服务不可达",
		entity.AlertTypeGostForwardMissing: "GOST转发规则不完整",
		entity.AlertTypeGostHighErrors:     "GOST错误率过高",
	}
	if name, ok := names[t]; ok {
		return name
	}
	return t
}

// GetAlertTypeOptions 获取告警类型选项
func GetAlertTypeOptions() []map[string]string {
	return []map[string]string{
		{"value": entity.AlertTypeMerchantExpired, "label": "商户即将过期"},
		{"value": entity.AlertTypeServerDown, "label": "服务器宕机"},
		{"value": entity.AlertTypeCpuHigh, "label": "CPU使用率过高"},
		{"value": entity.AlertTypeMemoryHigh, "label": "内存使用率过高"},
		{"value": entity.AlertTypeDiskHigh, "label": "磁盘使用率过高"},
		{"value": entity.AlertTypeServiceDown, "label": "服务异常"},
		{"value": entity.AlertTypeGostDown, "label": "GOST服务不可达"},
		{"value": entity.AlertTypeGostForwardMissing, "label": "GOST转发规则不完整"},
		{"value": entity.AlertTypeGostHighErrors, "label": "GOST错误率过高"},
	}
}

// GetNotifyTypeOptions 获取通知类型选项
func GetNotifyTypeOptions() []map[string]string {
	return []map[string]string{
		{"value": entity.NotifyTypeWebhook, "label": "Webhook"},
		{"value": entity.NotifyTypeEmail, "label": "邮件"},
		{"value": entity.NotifyTypeSms, "label": "短信"},
	}
}

// GetAlertLevelOptions 获取告警级别选项
func GetAlertLevelOptions() []map[string]string {
	return []map[string]string{
		{"value": entity.AlertLevelInfo, "label": "信息"},
		{"value": entity.AlertLevelWarning, "label": "警告"},
		{"value": entity.AlertLevelError, "label": "错误"},
		{"value": entity.AlertLevelCritical, "label": "严重"},
	}
}
