package apitest

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"server/internal/server/model"
	"server/pkg/dbs"
	"server/pkg/entity"
	"strings"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

// ========== API 目录 ==========

// GetAPICatalog 获取API目录（商户服务器所有API）
func GetAPICatalog() model.GetAPICatalogResp {
	categories := []model.APICategory{
		{
			Name:   "用户管理",
			Module: "user",
			Endpoints: []model.APIEndpoint{
				{Method: "POST", Path: "/v1/manager/user/login", Description: "管理员登录", RequireAuth: false},
				{Method: "POST", Path: "/v1/user/device_token", Description: "注册设备Token", RequireAuth: true},
				{Method: "PUT", Path: "/v1/user/current", Description: "更新当前用户信息", RequireAuth: true},
				{Method: "GET", Path: "/v1/user/qrcode", Description: "获取我的二维码", RequireAuth: true},
				{Method: "PUT", Path: "/v1/user/my/setting", Description: "更新用户设置", RequireAuth: true},
				{Method: "POST", Path: "/v1/user/chatpwd", Description: "设置聊天密码", RequireAuth: true},
				{Method: "GET", Path: "/v1/user/devices", Description: "设备列表", RequireAuth: true},
				{Method: "GET", Path: "/v1/user/online", Description: "在线用户列表", RequireAuth: true},
			},
		},
		{
			Name:   "好友管理",
			Module: "friend",
			Endpoints: []model.APIEndpoint{
				{Method: "POST", Path: "/v1/friend/apply", Description: "申请添加好友", RequireAuth: true},
				{Method: "GET", Path: "/v1/friend/apply", Description: "获取好友申请列表", RequireAuth: true},
				{Method: "POST", Path: "/v1/friend/sure", Description: "确认好友申请", RequireAuth: true},
				{Method: "PUT", Path: "/v1/friend/refuse/:to_uid", Description: "拒绝好友申请", RequireAuth: true},
				{Method: "GET", Path: "/v1/friend/sync", Description: "同步好友列表", RequireAuth: true},
				{Method: "GET", Path: "/v1/friend/search", Description: "搜索好友", RequireAuth: true},
				{Method: "PUT", Path: "/v1/friend/remark", Description: "设置好友备注", RequireAuth: true},
				{Method: "DELETE", Path: "/v1/friends/:uid", Description: "删除好友", RequireAuth: true},
			},
		},
		{
			Name:   "群组管理",
			Module: "group",
			Endpoints: []model.APIEndpoint{
				{Method: "POST", Path: "/v1/group/create", Description: "创建群组", RequireAuth: true},
				{Method: "GET", Path: "/v1/group/my", Description: "我的群组列表", RequireAuth: true},
				{Method: "GET", Path: "/v1/group/search", Description: "搜索群组", RequireAuth: true},
				{Method: "POST", Path: "/v1/groups/:group_no/members", Description: "添加群成员", RequireAuth: true},
				{Method: "DELETE", Path: "/v1/groups/:group_no/members", Description: "移除群成员", RequireAuth: true},
				{Method: "GET", Path: "/v1/groups/:group_no/members", Description: "获取群成员", RequireAuth: true},
				{Method: "GET", Path: "/v1/groups/:group_no", Description: "获取群详情", RequireAuth: true},
				{Method: "PUT", Path: "/v1/groups/:group_no/setting", Description: "更新群设置", RequireAuth: true},
				{Method: "POST", Path: "/v1/groups/:group_no/exit", Description: "退出群组", RequireAuth: true},
				{Method: "DELETE", Path: "/v1/groups/:group_no/disband", Description: "解散群组", RequireAuth: true},
			},
		},
		{
			Name:   "消息管理",
			Module: "message",
			Endpoints: []model.APIEndpoint{
				{Method: "POST", Path: "/v1/message/sync", Description: "同步消息", RequireAuth: true},
				{Method: "DELETE", Path: "/v1/message", Description: "删除消息", RequireAuth: true},
				{Method: "POST", Path: "/v1/message/revoke", Description: "撤回消息", RequireAuth: true},
				{Method: "POST", Path: "/v1/message/search", Description: "搜索消息", RequireAuth: true},
				{Method: "POST", Path: "/v1/msg/send", Description: "发送消息", RequireAuth: true},
				{Method: "POST", Path: "/v1/manager/message/send", Description: "管理员发送消息", RequireAuth: true},
				{Method: "GET", Path: "/v1/manager/message", Description: "消息列表", RequireAuth: true},
			},
		},
		{
			Name:   "会话管理",
			Module: "conversation",
			Endpoints: []model.APIEndpoint{
				{Method: "GET", Path: "/v1/conversations", Description: "获取会话列表", RequireAuth: true},
				{Method: "PUT", Path: "/v1/coversations/clearUnread", Description: "清除未读", RequireAuth: true},
				{Method: "POST", Path: "/v1/conversation/sync", Description: "同步会话", RequireAuth: true},
				{Method: "DELETE", Path: "/v1/conversations/:channel_id/:channel_type", Description: "删除会话", RequireAuth: true},
			},
		},
		{
			Name:   "红包功能",
			Module: "redpacket",
			Endpoints: []model.APIEndpoint{
				{Method: "POST", Path: "/v1/redpacket/send", Description: "发送红包", RequireAuth: true},
				{Method: "POST", Path: "/v1/redpacket/grab/:packet_no", Description: "抢红包", RequireAuth: true},
				{Method: "GET", Path: "/v1/redpacket/:packet_no", Description: "红包详情", RequireAuth: true},
				{Method: "GET", Path: "/v1/redpacket/config", Description: "红包配置", RequireAuth: true},
				{Method: "GET", Path: "/v1/manager/redpacket/statistics", Description: "红包统计", RequireAuth: true},
				{Method: "GET", Path: "/v1/manager/redpacket/list", Description: "红包列表", RequireAuth: true},
			},
		},
		{
			Name:   "钱包功能",
			Module: "wallet",
			Endpoints: []model.APIEndpoint{
				{Method: "GET", Path: "/v1/wallet/info", Description: "钱包信息", RequireAuth: true},
				{Method: "GET", Path: "/v1/wallet/transactions", Description: "交易记录", RequireAuth: true},
				{Method: "POST", Path: "/v1/wallet/transfer", Description: "转账", RequireAuth: true},
				{Method: "GET", Path: "/v1/manager/wallet/statistics", Description: "钱包统计", RequireAuth: true},
				{Method: "GET", Path: "/v1/manager/wallet/list", Description: "钱包列表", RequireAuth: true},
				{Method: "POST", Path: "/v1/manager/wallet/recharge", Description: "充值", RequireAuth: true},
			},
		},
		{
			Name:   "签到功能",
			Module: "checkin",
			Endpoints: []model.APIEndpoint{
				{Method: "POST", Path: "/v1/checkin", Description: "签到", RequireAuth: true},
				{Method: "GET", Path: "/v1/checkin/status", Description: "签到状态", RequireAuth: true},
				{Method: "GET", Path: "/v1/checkin/records", Description: "签到记录", RequireAuth: true},
				{Method: "GET", Path: "/v1/checkin/month", Description: "月签到数据", RequireAuth: true},
				{Method: "GET", Path: "/v1/manager/checkin/statistics", Description: "签到统计", RequireAuth: true},
				{Method: "GET", Path: "/v1/manager/checkin/config", Description: "签到配置", RequireAuth: true},
				{Method: "POST", Path: "/v1/manager/checkin/config", Description: "更新签到配置", RequireAuth: true},
			},
		},
		{
			Name:   "朋友圈",
			Module: "moments",
			Endpoints: []model.APIEndpoint{
				{Method: "POST", Path: "/v1/moments", Description: "发布动态", RequireAuth: true},
				{Method: "GET", Path: "/v1/moments", Description: "动态列表", RequireAuth: true},
				{Method: "GET", Path: "/v1/moments/:moment_no", Description: "动态详情", RequireAuth: true},
				{Method: "DELETE", Path: "/v1/moments/:moment_no", Description: "删除动态", RequireAuth: true},
				{Method: "PUT", Path: "/v1/moments/:moment_no/like", Description: "点赞", RequireAuth: true},
				{Method: "POST", Path: "/v1/moments/:moment_no/comments", Description: "评论", RequireAuth: true},
			},
		},
		{
			Name:   "文件上传",
			Module: "file",
			Endpoints: []model.APIEndpoint{
				{Method: "GET", Path: "/v1/upload", Description: "获取上传路径", RequireAuth: true},
				{Method: "POST", Path: "/v1/upload", Description: "上传文件", RequireAuth: true},
				{Method: "GET", Path: "/v1/preview/*path", Description: "预览文件", RequireAuth: false},
			},
		},
		{
			Name:   "系统健康",
			Module: "common",
			Endpoints: []model.APIEndpoint{
				{Method: "GET", Path: "/v1/health", Description: "健康检查", RequireAuth: false},
				{Method: "GET", Path: "/v1/manager/common/appconfig", Description: "应用配置", RequireAuth: true},
				{Method: "POST", Path: "/v1/manager/common/appconfig", Description: "更新应用配置", RequireAuth: true},
			},
		},
		{
			Name:   "安全管理",
			Module: "security",
			Endpoints: []model.APIEndpoint{
				{Method: "GET", Path: "/v1/manager/ip-rules", Description: "IP规则列表", RequireAuth: true},
				{Method: "POST", Path: "/v1/manager/ip-rules", Description: "添加IP规则", RequireAuth: true},
				{Method: "DELETE", Path: "/v1/manager/ip-rules/:id", Description: "删除IP规则", RequireAuth: true},
			},
		},
		{
			Name:   "举报管理",
			Module: "report",
			Endpoints: []model.APIEndpoint{
				{Method: "GET", Path: "/v1/report/categories", Description: "举报分类", RequireAuth: false},
				{Method: "POST", Path: "/v1/report", Description: "提交举报", RequireAuth: true},
				{Method: "GET", Path: "/v1/manager/report/list", Description: "举报列表", RequireAuth: true},
			},
		},
		{
			Name:   "邀请功能",
			Module: "invite",
			Endpoints: []model.APIEndpoint{
				{Method: "GET", Path: "/v1/manager/invite/config", Description: "邀请配置", RequireAuth: true},
				{Method: "POST", Path: "/v1/manager/invite/config", Description: "更新邀请配置", RequireAuth: true},
				{Method: "GET", Path: "/v1/manager/invite/stats", Description: "邀请统计", RequireAuth: true},
				{Method: "GET", Path: "/v1/manager/invite/codes", Description: "邀请码列表", RequireAuth: true},
				{Method: "POST", Path: "/v1/manager/invite/codes", Description: "创建邀请码", RequireAuth: true},
			},
		},
		{
			Name:   "机器人",
			Module: "robot",
			Endpoints: []model.APIEndpoint{
				{Method: "POST", Path: "/v1/robot/sync", Description: "同步机器人", RequireAuth: true},
				{Method: "POST", Path: "/v1/robot/inline_query", Description: "内联查询", RequireAuth: true},
				{Method: "GET", Path: "/v1/manager/robot/menus", Description: "机器人菜单列表", RequireAuth: true},
			},
		},
		{
			Name:   "收藏功能",
			Module: "favorite",
			Endpoints: []model.APIEndpoint{
				{Method: "POST", Path: "/v1/favorites", Description: "添加收藏", RequireAuth: true},
				{Method: "DELETE", Path: "/v1/favorites/:id", Description: "删除收藏", RequireAuth: true},
				{Method: "GET", Path: "/v1/favorite/my", Description: "我的收藏", RequireAuth: true},
			},
		},
		{
			Name:   "标签管理",
			Module: "label",
			Endpoints: []model.APIEndpoint{
				{Method: "POST", Path: "/v1/labels", Description: "添加标签", RequireAuth: true},
				{Method: "DELETE", Path: "/v1/labels/:id", Description: "删除标签", RequireAuth: true},
				{Method: "PUT", Path: "/v1/labels/:id", Description: "更新标签", RequireAuth: true},
				{Method: "GET", Path: "/v1/labels", Description: "标签列表", RequireAuth: true},
			},
		},
		{
			Name:   "表情包",
			Module: "sticker",
			Endpoints: []model.APIEndpoint{
				{Method: "GET", Path: "/v1/sticker", Description: "搜索表情", RequireAuth: false},
				{Method: "POST", Path: "/v1/sticker/user", Description: "添加用户表情", RequireAuth: true},
				{Method: "DELETE", Path: "/v1/sticker/user", Description: "删除用户表情", RequireAuth: true},
				{Method: "GET", Path: "/v1/sticker/user", Description: "用户表情列表", RequireAuth: true},
				{Method: "GET", Path: "/v1/sticker/store", Description: "表情商店", RequireAuth: false},
			},
		},
	}

	total := 0
	for _, cat := range categories {
		total += len(cat.Endpoints)
	}

	return model.GetAPICatalogResp{
		Categories: categories,
		Total:      total,
	}
}

// ========== 测试用例管理 ==========

// CreateTestCase 创建测试用例
func CreateTestCase(merchantId int, req model.TestCaseReq) error {
	headersJSON, _ := json.Marshal(req.Headers)
	queryParamsJSON, _ := json.Marshal(req.QueryParams)

	testCase := &entity.APITestCases{
		MerchantId:       merchantId,
		Name:             req.Name,
		Module:           req.Module,
		Method:           req.Method,
		Path:             req.Path,
		Headers:          string(headersJSON),
		QueryParams:      string(queryParamsJSON),
		Body:             req.Body,
		ExpectedStatus:   req.ExpectedStatus,
		ExpectedContains: req.ExpectedContains,
	}

	_, err := dbs.DBAdmin.Insert(testCase)
	return err
}

// UpdateTestCase 更新测试用例
func UpdateTestCase(id int64, req model.TestCaseReq) error {
	headersJSON, _ := json.Marshal(req.Headers)
	queryParamsJSON, _ := json.Marshal(req.QueryParams)

	testCase := &entity.APITestCases{
		Name:             req.Name,
		Module:           req.Module,
		Method:           req.Method,
		Path:             req.Path,
		Headers:          string(headersJSON),
		QueryParams:      string(queryParamsJSON),
		Body:             req.Body,
		ExpectedStatus:   req.ExpectedStatus,
		ExpectedContains: req.ExpectedContains,
		UpdatedAt:        time.Now(),
	}

	_, err := dbs.DBAdmin.Where("id = ?", id).Update(testCase)
	return err
}

// DeleteTestCase 删除测试用例
func DeleteTestCase(id int64) error {
	_, err := dbs.DBAdmin.Where("id = ?", id).Delete(&entity.APITestCases{})
	return err
}

// QueryTestCases 查询测试用例
func QueryTestCases(req model.QueryTestCaseReq) (model.QueryTestCaseResp, error) {
	var resp model.QueryTestCaseResp
	var list []entity.APITestCases

	session := dbs.DBAdmin.NewSession()
	defer session.Close()

	if req.MerchantId > 0 {
		session = session.Where("merchant_id = ?", req.MerchantId)
	}
	if req.Module != "" {
		session = session.Where("module = ?", req.Module)
	}
	if req.Name != "" {
		session = session.Where("name LIKE ?", "%"+req.Name+"%")
	}

	offset := (req.Page - 1) * req.Size
	total, err := session.Desc("id").Limit(req.Size, offset).FindAndCount(&list)
	if err != nil {
		return resp, err
	}

	resp.Total = int(total)
	for _, item := range list {
		var headers map[string]string
		var queryParams map[string]string
		json.Unmarshal([]byte(item.Headers), &headers)
		json.Unmarshal([]byte(item.QueryParams), &queryParams)

		lastRunAt := ""
		if !item.LastRunAt.IsZero() {
			lastRunAt = item.LastRunAt.Format(time.DateTime)
		}

		resp.List = append(resp.List, model.TestCaseResp{
			Id:               item.Id,
			Name:             item.Name,
			Module:           item.Module,
			Method:           item.Method,
			Path:             item.Path,
			Headers:          headers,
			QueryParams:      queryParams,
			Body:             item.Body,
			ExpectedStatus:   item.ExpectedStatus,
			ExpectedContains: item.ExpectedContains,
			LastRunAt:        lastRunAt,
			LastRunStatus:    item.LastRunStatus,
			CreatedAt:        item.CreatedAt.Format(time.DateTime),
			UpdatedAt:        item.UpdatedAt.Format(time.DateTime),
		})
	}

	return resp, nil
}

// ========== API 测试执行 ==========

// RunAPITest 运行单个API测试
func RunAPITest(req model.RunAPITestReq) (model.RunAPITestResp, error) {
	var resp model.RunAPITestResp

	// 获取商户信息
	var merchant entity.Merchants
	has, err := dbs.DBAdmin.Where("id = ?", req.MerchantId).Get(&merchant)
	if err != nil {
		return resp, err
	}
	if !has {
		return resp, errors.New("商户不存在")
	}

	// 构建完整URL
	baseURL := fmt.Sprintf("http://%s:%d", merchant.ServerIP, merchant.Port)
	fullURL := baseURL + req.Path

	// 添加查询参数
	if len(req.QueryParams) > 0 {
		params := make([]string, 0)
		for k, v := range req.QueryParams {
			params = append(params, fmt.Sprintf("%s=%s", k, v))
		}
		fullURL += "?" + strings.Join(params, "&")
	}

	// 创建HTTP请求
	var httpReq *http.Request
	if req.Body != "" {
		httpReq, err = http.NewRequest(req.Method, fullURL, strings.NewReader(req.Body))
	} else {
		httpReq, err = http.NewRequest(req.Method, fullURL, nil)
	}
	if err != nil {
		resp.Error = err.Error()
		return resp, nil
	}

	// 设置请求头
	httpReq.Header.Set("Content-Type", "application/json")
	for k, v := range req.Headers {
		httpReq.Header.Set(k, v)
	}

	// 发送请求并计时
	client := &http.Client{Timeout: 30 * time.Second}
	startTime := time.Now()
	httpResp, err := client.Do(httpReq)
	elapsed := time.Since(startTime).Milliseconds()

	if err != nil {
		resp.Success = false
		resp.Error = err.Error()
		resp.ResponseTime = elapsed
		return resp, nil
	}
	defer httpResp.Body.Close()

	// 读取响应
	body, _ := io.ReadAll(httpResp.Body)

	// 收集响应头
	respHeaders := make(map[string]string)
	for k, v := range httpResp.Header {
		if len(v) > 0 {
			respHeaders[k] = v[0]
		}
	}

	resp.Success = true
	resp.StatusCode = httpResp.StatusCode
	resp.ResponseTime = elapsed
	resp.Headers = respHeaders
	resp.Body = string(body)

	return resp, nil
}

// RunTestCase 运行测试用例
func RunTestCase(merchantId int, testCaseId int64) (model.RunAPITestResp, error) {
	var testCase entity.APITestCases
	has, err := dbs.DBAdmin.Where("id = ?", testCaseId).Get(&testCase)
	if err != nil {
		return model.RunAPITestResp{}, err
	}
	if !has {
		return model.RunAPITestResp{}, errors.New("测试用例不存在")
	}

	var headers map[string]string
	var queryParams map[string]string
	json.Unmarshal([]byte(testCase.Headers), &headers)
	json.Unmarshal([]byte(testCase.QueryParams), &queryParams)

	req := model.RunAPITestReq{
		MerchantId:  merchantId,
		Method:      testCase.Method,
		Path:        testCase.Path,
		Headers:     headers,
		QueryParams: queryParams,
		Body:        testCase.Body,
	}

	resp, err := RunAPITest(req)
	if err != nil {
		return resp, err
	}

	// 判断测试结果
	runStatus := 1 // 成功
	if !resp.Success || resp.StatusCode != testCase.ExpectedStatus {
		runStatus = 2 // 失败
	}
	if testCase.ExpectedContains != "" && !strings.Contains(resp.Body, testCase.ExpectedContains) {
		runStatus = 2
	}

	// 更新测试用例状态
	dbs.DBAdmin.Where("id = ?", testCaseId).Cols("last_run_at", "last_run_status").Update(&entity.APITestCases{
		LastRunAt:     time.Now(),
		LastRunStatus: runStatus,
	})

	return resp, nil
}

// ========== 批量测试 ==========

// BatchTest 批量测试
func BatchTest(req model.BatchTestReq) (model.BatchTestResp, error) {
	var resp model.BatchTestResp
	startTime := time.Now()

	for _, testCaseId := range req.TestCaseIds {
		var testCase entity.APITestCases
		has, _ := dbs.DBAdmin.Where("id = ?", testCaseId).Get(&testCase)
		if !has {
			continue
		}

		testResp, err := RunTestCase(req.MerchantId, testCaseId)

		result := model.BatchTestResult{
			TestCaseId:   testCaseId,
			TestCaseName: testCase.Name,
			StatusCode:   testResp.StatusCode,
			ResponseTime: testResp.ResponseTime,
		}

		if err != nil {
			result.Success = false
			result.Error = err.Error()
			resp.Failed++
		} else if !testResp.Success {
			result.Success = false
			result.Error = testResp.Error
			resp.Failed++
		} else if testResp.StatusCode != testCase.ExpectedStatus {
			result.Success = false
			result.Error = fmt.Sprintf("状态码不匹配: 期望 %d, 实际 %d", testCase.ExpectedStatus, testResp.StatusCode)
			resp.Failed++
		} else if testCase.ExpectedContains != "" && !strings.Contains(testResp.Body, testCase.ExpectedContains) {
			result.Success = false
			result.Error = "响应内容不包含期望内容"
			resp.Failed++
		} else {
			result.Success = true
			resp.Success++
		}

		resp.Results = append(resp.Results, result)
		resp.Total++
	}

	resp.TotalTime = time.Since(startTime).Milliseconds()
	return resp, nil
}

// ========== 监控配置管理 ==========

// CreateMonitorConfig 创建监控配置
func CreateMonitorConfig(req model.MonitorConfigReq) error {
	testCaseIdsJSON, _ := json.Marshal(req.TestCaseIds)

	enabled := 0
	if req.Enabled {
		enabled = 1
	}

	config := &entity.APIMonitorConfigs{
		MerchantId:   req.MerchantId,
		Name:         req.Name,
		TestCaseIds:  string(testCaseIdsJSON),
		Interval:     req.Interval,
		Enabled:      enabled,
		AlertEmail:   req.AlertEmail,
		AlertWebhook: req.AlertWebhook,
	}

	_, err := dbs.DBAdmin.Insert(config)
	return err
}

// UpdateMonitorConfig 更新监控配置
func UpdateMonitorConfig(id int64, req model.MonitorConfigReq) error {
	testCaseIdsJSON, _ := json.Marshal(req.TestCaseIds)

	enabled := 0
	if req.Enabled {
		enabled = 1
	}

	config := &entity.APIMonitorConfigs{
		Name:         req.Name,
		TestCaseIds:  string(testCaseIdsJSON),
		Interval:     req.Interval,
		Enabled:      enabled,
		AlertEmail:   req.AlertEmail,
		AlertWebhook: req.AlertWebhook,
		UpdatedAt:    time.Now(),
	}

	_, err := dbs.DBAdmin.Where("id = ?", id).Update(config)
	return err
}

// DeleteMonitorConfig 删除监控配置
func DeleteMonitorConfig(id int64) error {
	_, err := dbs.DBAdmin.Where("id = ?", id).Delete(&entity.APIMonitorConfigs{})
	return err
}

// QueryMonitorConfigs 查询监控配置
func QueryMonitorConfigs(req model.QueryMonitorReq) (model.QueryMonitorResp, error) {
	var resp model.QueryMonitorResp

	type ConfigWithMerchant struct {
		entity.APIMonitorConfigs `xorm:"extends"`
		MerchantName             string `xorm:"merchants.name"`
	}

	var list []ConfigWithMerchant

	session := dbs.DBAdmin.Table("api_monitor_configs").
		Join("LEFT", "merchants", "api_monitor_configs.merchant_id = merchants.id")

	if req.MerchantId > 0 {
		session = session.Where("api_monitor_configs.merchant_id = ?", req.MerchantId)
	}
	if req.Enabled != nil {
		enabled := 0
		if *req.Enabled {
			enabled = 1
		}
		session = session.Where("api_monitor_configs.enabled = ?", enabled)
	}

	offset := (req.Page - 1) * req.Size
	total, err := session.Desc("api_monitor_configs.id").Limit(req.Size, offset).FindAndCount(&list)
	if err != nil {
		logx.Errorf("query monitor configs err: %+v", err)
		return resp, err
	}

	resp.Total = int(total)
	for _, item := range list {
		var testCaseIds []int64
		json.Unmarshal([]byte(item.TestCaseIds), &testCaseIds)

		lastRunAt := ""
		if !item.LastRunAt.IsZero() {
			lastRunAt = item.LastRunAt.Format(time.DateTime)
		}

		resp.List = append(resp.List, model.MonitorConfigResp{
			Id:           item.Id,
			MerchantId:   item.MerchantId,
			MerchantName: item.MerchantName,
			Name:         item.Name,
			TestCaseIds:  testCaseIds,
			Interval:     item.Interval,
			Enabled:      item.Enabled == 1,
			AlertEmail:   item.AlertEmail,
			AlertWebhook: item.AlertWebhook,
			LastRunAt:    lastRunAt,
			LastStatus:   item.LastStatus,
			CreatedAt:    item.CreatedAt.Format(time.DateTime),
		})
	}

	return resp, nil
}

// QueryMonitorHistory 查询监控历史
func QueryMonitorHistory(req model.QueryMonitorHistoryReq) (model.QueryMonitorHistoryResp, error) {
	var resp model.QueryMonitorHistoryResp
	var list []entity.APIMonitorHistory

	session := dbs.DBAdmin.Where("monitor_id = ?", req.MonitorId)

	offset := (req.Page - 1) * req.Size
	total, err := session.Desc("id").Limit(req.Size, offset).FindAndCount(&list)
	if err != nil {
		return resp, err
	}

	resp.Total = int(total)
	for _, item := range list {
		resp.List = append(resp.List, model.MonitorHistoryResp{
			Id:        item.Id,
			MonitorId: item.MonitorId,
			RunAt:     item.CreatedAt.Format(time.DateTime),
			Total:     item.Total,
			Success:   item.Success,
			Failed:    item.Failed,
			TotalTime: item.TotalTime,
			Results:   item.Results,
		})
	}

	return resp, nil
}
