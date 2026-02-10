package model

// ========== 商户 API 测试工具 ==========

// APICategory API分类
type APICategory struct {
	Name     string        `json:"name"`     // 分类名称
	Module   string        `json:"module"`   // 模块名
	Endpoints []APIEndpoint `json:"endpoints"` // 端点列表
}

// APIEndpoint API端点
type APIEndpoint struct {
	Method      string            `json:"method"`       // HTTP方法
	Path        string            `json:"path"`         // 路径
	Description string            `json:"description"`  // 描述
	Module      string            `json:"module"`       // 所属模块
	Headers     map[string]string `json:"headers"`      // 请求头
	Params      []APIParam        `json:"params"`       // 参数列表
	RequireAuth bool              `json:"require_auth"` // 是否需要认证
}

// APIParam API参数
type APIParam struct {
	Name        string `json:"name"`        // 参数名
	Type        string `json:"type"`        // 参数类型
	Required    bool   `json:"required"`    // 是否必填
	Description string `json:"description"` // 描述
	In          string `json:"in"`          // 位置: path, query, body
	Default     string `json:"default"`     // 默认值
}

// ========== 测试用例管理 ==========

// TestCaseReq 测试用例请求
type TestCaseReq struct {
	Id          int64             `json:"id"`           // 用例ID（编辑时使用）
	Name        string            `json:"name"`         // 用例名称
	Module      string            `json:"module"`       // 所属模块
	Method      string            `json:"method"`       // HTTP方法
	Path        string            `json:"path"`         // 请求路径
	Headers     map[string]string `json:"headers"`      // 请求头
	QueryParams map[string]string `json:"query_params"` // 查询参数
	Body        string            `json:"body"`         // 请求体JSON
	ExpectedStatus int            `json:"expected_status"` // 期望状态码
	ExpectedContains string       `json:"expected_contains"` // 期望包含内容
}

// TestCaseResp 测试用例响应
type TestCaseResp struct {
	Id             int64             `json:"id"`
	Name           string            `json:"name"`
	Module         string            `json:"module"`
	Method         string            `json:"method"`
	Path           string            `json:"path"`
	Headers        map[string]string `json:"headers"`
	QueryParams    map[string]string `json:"query_params"`
	Body           string            `json:"body"`
	ExpectedStatus int               `json:"expected_status"`
	ExpectedContains string          `json:"expected_contains"`
	LastRunAt      string            `json:"last_run_at"`
	LastRunStatus  int               `json:"last_run_status"` // 0:未运行 1:成功 2:失败
	CreatedAt      string            `json:"created_at"`
	UpdatedAt      string            `json:"updated_at"`
}

// QueryTestCaseReq 查询测试用例请求
type QueryTestCaseReq struct {
	Pagination
	MerchantId int    `json:"merchant_id" form:"merchant_id"`
	Module     string `json:"module" form:"module"`
	Name       string `json:"name" form:"name"`
}

// QueryTestCaseResp 查询测试用例响应
type QueryTestCaseResp struct {
	List  []TestCaseResp `json:"list"`
	Total int            `json:"total"`
}

// ========== API 测试执行 ==========

// RunAPITestReq 运行API测试请求
type RunAPITestReq struct {
	MerchantId  int               `json:"merchant_id" binding:"required"` // 商户ID
	Method      string            `json:"method" binding:"required"`      // HTTP方法
	Path        string            `json:"path" binding:"required"`        // 请求路径
	Headers     map[string]string `json:"headers"`                        // 请求头
	QueryParams map[string]string `json:"query_params"`                   // 查询参数
	Body        string            `json:"body"`                           // 请求体JSON
}

// RunAPITestResp 运行API测试响应
type RunAPITestResp struct {
	Success      bool              `json:"success"`       // 是否成功
	StatusCode   int               `json:"status_code"`   // 响应状态码
	ResponseTime int64             `json:"response_time"` // 响应时间(ms)
	Headers      map[string]string `json:"headers"`       // 响应头
	Body         string            `json:"body"`          // 响应体
	Error        string            `json:"error"`         // 错误信息
}

// ========== 批量测试 ==========

// BatchTestReq 批量测试请求
type BatchTestReq struct {
	MerchantId int     `json:"merchant_id" binding:"required"`
	TestCaseIds []int64 `json:"test_case_ids" binding:"required"`
}

// BatchTestResult 单个测试结果
type BatchTestResult struct {
	TestCaseId   int64  `json:"test_case_id"`
	TestCaseName string `json:"test_case_name"`
	Success      bool   `json:"success"`
	StatusCode   int    `json:"status_code"`
	ResponseTime int64  `json:"response_time"`
	Error        string `json:"error"`
}

// BatchTestResp 批量测试响应
type BatchTestResp struct {
	Total     int               `json:"total"`
	Success   int               `json:"success"`
	Failed    int               `json:"failed"`
	Results   []BatchTestResult `json:"results"`
	TotalTime int64             `json:"total_time"` // 总耗时(ms)
}

// ========== 自动化监控 ==========

// MonitorConfigReq 监控配置请求
type MonitorConfigReq struct {
	Id          int64   `json:"id"`
	MerchantId  int     `json:"merchant_id" binding:"required"`
	Name        string  `json:"name" binding:"required"`
	TestCaseIds []int64 `json:"test_case_ids" binding:"required"`
	Interval    int     `json:"interval"`  // 检测间隔(分钟)
	Enabled     bool    `json:"enabled"`   // 是否启用
	AlertEmail  string  `json:"alert_email"` // 告警邮箱
	AlertWebhook string `json:"alert_webhook"` // 告警Webhook
}

// MonitorConfigResp 监控配置响应
type MonitorConfigResp struct {
	Id           int64   `json:"id"`
	MerchantId   int     `json:"merchant_id"`
	MerchantName string  `json:"merchant_name"`
	Name         string  `json:"name"`
	TestCaseIds  []int64 `json:"test_case_ids"`
	Interval     int     `json:"interval"`
	Enabled      bool    `json:"enabled"`
	AlertEmail   string  `json:"alert_email"`
	AlertWebhook string  `json:"alert_webhook"`
	LastRunAt    string  `json:"last_run_at"`
	LastStatus   int     `json:"last_status"` // 0:未运行 1:成功 2:部分失败 3:全部失败
	CreatedAt    string  `json:"created_at"`
}

// QueryMonitorReq 查询监控请求
type QueryMonitorReq struct {
	Pagination
	MerchantId int  `json:"merchant_id" form:"merchant_id"`
	Enabled    *bool `json:"enabled" form:"enabled"`
}

// QueryMonitorResp 查询监控响应
type QueryMonitorResp struct {
	List  []MonitorConfigResp `json:"list"`
	Total int                 `json:"total"`
}

// MonitorHistoryResp 监控历史响应
type MonitorHistoryResp struct {
	Id         int64  `json:"id"`
	MonitorId  int64  `json:"monitor_id"`
	RunAt      string `json:"run_at"`
	Total      int    `json:"total"`
	Success    int    `json:"success"`
	Failed     int    `json:"failed"`
	TotalTime  int64  `json:"total_time"`
	Results    string `json:"results"` // JSON格式的详细结果
}

// QueryMonitorHistoryReq 查询监控历史请求
type QueryMonitorHistoryReq struct {
	Pagination
	MonitorId int64 `json:"monitor_id" form:"monitor_id" binding:"required"`
}

// QueryMonitorHistoryResp 查询监控历史响应
type QueryMonitorHistoryResp struct {
	List  []MonitorHistoryResp `json:"list"`
	Total int                  `json:"total"`
}

// ========== API 目录 ==========

// GetAPICatalogResp 获取API目录响应
type GetAPICatalogResp struct {
	Categories []APICategory `json:"categories"`
	Total      int           `json:"total"`
}
