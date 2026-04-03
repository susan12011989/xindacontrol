package model

// ========== 系统IP ==========

// GetSystemIPsResp 系统服务器IP响应
type GetSystemIPsResp struct {
	IPs   []SystemIPItem `json:"ips"`
	Total int            `json:"total"`
}

// SystemIPItem 系统服务器IP项
type SystemIPItem struct {
	ServerId     int                      `json:"server_id"`
	ServerName   string                   `json:"server_name"`
	IP           string                   `json:"ip"`
	AuxiliaryIP  string                   `json:"auxiliary_ip"`  // 辅助IP
	Status       int                      `json:"status"`
	MerchantId   int                      `json:"merchant_id"`   // 主商户ID（兼容旧逻辑）
	MerchantName string                   `json:"merchant_name"` // 主商户名称（兼容旧逻辑）
	Merchants    []SystemIPMerchantItem   `json:"merchants"`     // 关联的所有商户
}

// SystemIPMerchantItem 服务器关联的商户信息
type SystemIPMerchantItem struct {
	MerchantId   int    `json:"merchant_id"`
	MerchantName string `json:"merchant_name"`
}

// ========== 上传目标 ==========

// GetTargetsResp 上传目标列表响应
type GetTargetsResp struct {
	Targets []TargetItem `json:"targets"`
}

// TargetItem 上传目标项
type TargetItem struct {
	Id             int    `json:"id"`               // 目标ID
	Index          int    `json:"index"`            // 配置索引（兼容旧逻辑）
	Name           string `json:"name"`             // 目标名称
	CloudType      string `json:"cloud_type"`       // 云类型
	CloudAccountId int64  `json:"cloud_account_id"` // 云账号ID
	AccountName    string `json:"account_name"`     // 云账号名称
	RegionId       string `json:"region_id"`        // 区域ID
	Bucket         string `json:"bucket"`           // Bucket名称
	ObjectPrefix   string `json:"object_prefix"`    // 对象前缀
	Enabled        bool   `json:"enabled"`          // 是否启用
	SortOrder      int    `json:"sort_order"`       // 排序顺序
	GroupId        int    `json:"group_id"`         // 分组ID
	GroupName      string `json:"group_name"`       // 分组名称
	MerchantId     int    `json:"merchant_id"`      // 商户ID（来自云账号）
	MerchantName   string `json:"merchant_name"`    // 商户名称
}

// CreateTargetReq 创建上传目标请求
type CreateTargetReq struct {
	Name           string `json:"name" binding:"required"`             // 目标名称
	CloudType      string `json:"cloud_type" binding:"required"`       // 云类型: aliyun, aws, tencent
	CloudAccountId int64  `json:"cloud_account_id" binding:"required"` // 云账号ID
	RegionId       string `json:"region_id" binding:"required"`        // 区域ID
	Bucket         string `json:"bucket" binding:"required"`           // Bucket名称
	ObjectPrefix   string `json:"object_prefix"`                       // 对象前缀
	Enabled        bool   `json:"enabled"`                             // 是否启用
	SortOrder      int    `json:"sort_order"`                          // 排序顺序
	GroupId        int    `json:"group_id"`                            // 分组ID
}

// UpdateTargetReq 更新上传目标请求
type UpdateTargetReq struct {
	Name           *string `json:"name"`             // 目标名称
	CloudType      *string `json:"cloud_type"`       // 云类型
	CloudAccountId *int64  `json:"cloud_account_id"` // 云账号ID
	RegionId       *string `json:"region_id"`        // 区域ID
	Bucket         *string `json:"bucket"`           // Bucket名称
	ObjectPrefix   *string `json:"object_prefix"`    // 对象前缀
	Enabled        *bool   `json:"enabled"`          // 是否启用
	SortOrder      *int    `json:"sort_order"`       // 排序顺序
	GroupId        *int    `json:"group_id"`         // 分组ID
}

// ========== 源文件 ==========

// GetSourceFilesResp 源文件列表响应
type GetSourceFilesResp struct {
	Files     []SourceFileItem `json:"files"`
	Total     int              `json:"total"`
	SourceDir string           `json:"source_dir"`
}

// SourceFileItem 源文件项
type SourceFileItem struct {
	Name    string `json:"name"`
	Size    int64  `json:"size"`
	ModTime string `json:"mod_time"`
}

// ========== 执行操作 ==========

// ExecuteEmbedReq 执行嵌入上传请求
type ExecuteEmbedReq struct {
	TargetIndexes []int    `json:"target_indexes" binding:"required,min=1"` // 选择的目标索引
	FileNames     []string `json:"file_names"`                              // 可选，为空则处理所有文件
	IPs           []string `json:"ips" binding:"required,min=1"`            // 选择的IP列表
}

// ExecuteEmbedResp 执行结果响应
type ExecuteEmbedResp struct {
	ExecutionId  string             `json:"execution_id"`
	TotalFiles   int                `json:"total_files"`
	TotalTargets int                `json:"total_targets"`
	Results      []UploadResultItem `json:"results"`
	Summary      ExecutionSummary   `json:"summary"`
}

// UploadResultItem 上传结果项
type UploadResultItem struct {
	FileName   string `json:"file_name"`
	TargetName string `json:"target_name"`
	CloudType  string `json:"cloud_type"`
	Bucket     string `json:"bucket"`
	ObjectKey  string `json:"object_key"`
	Success    bool   `json:"success"`
	Error      string `json:"error,omitempty"`
	ObjectUrl  string `json:"object_url,omitempty"`
}

// ExecutionSummary 执行摘要
type ExecutionSummary struct {
	SuccessCount int    `json:"success_count"`
	FailCount    int    `json:"fail_count"`
	TotalCount   int    `json:"total_count"`
	Duration     string `json:"duration"`
}

// ========== IP选择记录 ==========

// SaveSelectedIPsReq 保存选中IP请求
type SaveSelectedIPsReq struct {
	IPs []string `json:"ips" binding:"required"` // 选中的IP列表
}

// GetSelectedIPsResp 获取选中IP响应
type GetSelectedIPsResp struct {
	IPs []string `json:"ips"`
}
