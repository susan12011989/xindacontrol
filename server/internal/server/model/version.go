package model

import "time"

// ========== 版本管理请求/响应 ==========

// ListVersionsReq 版本列表请求
type ListVersionsReq struct {
	ServiceName string `form:"service_name"` // 可选过滤
	Page        int    `form:"page"`
	PageSize    int    `form:"page_size"`
}

// VersionInfo 版本信息
type VersionInfo struct {
	Id          int       `json:"id"`
	ServiceName string    `json:"service_name"`
	Version     string    `json:"version"`
	FileHash    string    `json:"file_hash"`
	FileSize    int64     `json:"file_size"`
	FilePath    string    `json:"file_path"`
	Changelog   string    `json:"changelog"`
	IsCurrent   bool      `json:"is_current"`
	UploadedBy  string    `json:"uploaded_by"`
	CreatedAt   time.Time `json:"created_at"`
}

// ListVersionsResp 版本列表响应
type ListVersionsResp struct {
	Total int64          `json:"total"`
	List  []*VersionInfo `json:"list"`
}

// UploadVersionReq 上传版本请求 (form-data)
type UploadVersionReq struct {
	ServiceName string `form:"service_name" binding:"required,oneof=server wukongim"`
	Version     string `form:"version" binding:"required"`
	Changelog   string `form:"changelog"`
	// file 字段由 multipart 处理
}

// SetCurrentVersionReq 设置当前版本请求
type SetCurrentVersionReq struct {
	VersionId int `json:"version_id" binding:"required"`
}

// DeployVersionReq 部署版本请求
type DeployVersionReq struct {
	VersionId int   `json:"version_id" binding:"required"`
	ServerIds []int `json:"server_ids" binding:"required,min=1"`
	Parallel  bool  `json:"parallel"`
}

// DeployResult 单台服务器部署结果
type DeployResult struct {
	ServerId   int    `json:"server_id"`
	ServerName string `json:"server_name"`
	Success    bool   `json:"success"`
	Message    string `json:"message"`
	Duration   int64  `json:"duration"` // 毫秒
}

// DeployVersionResp 部署版本响应
type DeployVersionResp struct {
	Total     int             `json:"total"`
	Success   int             `json:"success"`
	Failed    int             `json:"failed"`
	Results   []*DeployResult `json:"results"`
	StartedAt time.Time       `json:"started_at"`
	EndedAt   time.Time       `json:"ended_at"`
}

// RollbackReq 回滚请求
type RollbackReq struct {
	ServerId    int `json:"server_id" binding:"required"`
	ServiceName string `json:"service_name" binding:"required,oneof=server wukongim"`
}

// RollbackResp 回滚响应
type RollbackResp struct {
	Success          bool   `json:"success"`
	Message          string `json:"message"`
	RolledBackTo     string `json:"rolled_back_to"` // 回滚到的版本号
	PreviousVersion  string `json:"previous_version"`
}

// DeploymentHistoryReq 部署历史请求
type DeploymentHistoryReq struct {
	ServerId    int    `form:"server_id"`
	ServiceName string `form:"service_name"`
	Page        int    `form:"page"`
	PageSize    int    `form:"page_size"`
}

// DeploymentRecord 部署记录
type DeploymentRecord struct {
	Id                int64     `json:"id"`
	ServerId          int       `json:"server_id"`
	ServerName        string    `json:"server_name"`
	ServiceName       string    `json:"service_name"`
	VersionId         int       `json:"version_id"`
	Version           string    `json:"version"`
	PreviousVersionId int       `json:"previous_version_id"`
	PreviousVersion   string    `json:"previous_version"`
	Action            string    `json:"action"`
	Status            int       `json:"status"`
	StatusText        string    `json:"status_text"`
	Operator          string    `json:"operator"`
	BackupPath        string    `json:"backup_path"`
	Output            string    `json:"output"`
	StartedAt         time.Time `json:"started_at"`
	CompletedAt       time.Time `json:"completed_at"`
	CreatedAt         time.Time `json:"created_at"`
}

// DeploymentHistoryResp 部署历史响应
type DeploymentHistoryResp struct {
	Total int64               `json:"total"`
	List  []*DeploymentRecord `json:"list"`
}
