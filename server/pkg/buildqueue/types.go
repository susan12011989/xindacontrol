package buildqueue

import "time"

// Redis Keys
const (
	BuildTaskQueueKey    = "build:task:queue"    // 构建任务队列
	BuildTaskProgressKey = "build:task:progress" // Hash: task_id -> progress JSON
	BuildTaskCancelKey   = "build:task:cancel"   // Set: cancelled task IDs

	MaxRetryCount    = 3
	RetryIntervalSec = 30
)

// BuildTaskType 构建任务类型
type BuildTaskType string

const (
	TaskTypeBuild  BuildTaskType = "build"
	TaskTypeCancel BuildTaskType = "cancel"
)

// BuildTaskMessage Redis 队列中的任务消息
type BuildTaskMessage struct {
	ID              int           `json:"id"`                // 任务ID（数据库主键）
	Type            BuildTaskType `json:"type"`              // 任务类型
	BuildMerchantID int           `json:"build_merchant_id"` // 商户配置ID
	Platforms       string        `json:"platforms"`         // 目标平台 "android,ios"
	ServerID        int           `json:"server_id"`         // 指定构建服务器ID（0=自动选择）
	RetryCount      int           `json:"retry_count"`       // 已重试次数
	CreatedAt       time.Time     `json:"created_at"`        // 入队时间

	// 版本覆盖
	OverrideAndroidVersionCode *int    `json:"override_android_version_code,omitempty"`
	OverrideAndroidVersionName *string `json:"override_android_version_name,omitempty"`
	OverrideIOSVersion         *string `json:"override_ios_version,omitempty"`
	OverrideIOSBuild           *string `json:"override_ios_build,omitempty"`
}

// BuildProgress 构建进度信息
type BuildProgress struct {
	TaskID      int       `json:"task_id"`
	Status      int       `json:"status"`       // 0=排队 1=构建中 2=成功 3=失败 4=已取消
	Progress    int       `json:"progress"`     // 0-100
	CurrentStep string    `json:"current_step"` // 当前步骤描述
	LogLines    []string  `json:"log_lines"`    // 最近日志行
	UpdatedAt   time.Time `json:"updated_at"`
}
