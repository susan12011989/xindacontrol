package model

import "time"

// ========== 商户打包配置 ==========

// BuildMerchant 商户打包配置
type BuildMerchant struct {
	ID          int       `json:"id" xorm:"pk autoincr 'id'"`
	MerchantID  *int      `json:"merchant_id" xorm:"'merchant_id'"`
	Name        string    `json:"name" xorm:"'name'"`
	AppName     string    `json:"app_name" xorm:"'app_name'"`
	ShortName   string    `json:"short_name" xorm:"'short_name'"`
	Description string    `json:"description" xorm:"'description'"`
	Status      int       `json:"status" xorm:"'status'"`
	CreatedAt   time.Time `json:"created_at" xorm:"'created_at'"`
	UpdatedAt   time.Time `json:"updated_at" xorm:"'updated_at'"`

	// Android
	AndroidPackage     string `json:"android_package" xorm:"'android_package'"`
	AndroidVersionCode int    `json:"android_version_code" xorm:"'android_version_code'"`
	AndroidVersionName string `json:"android_version_name" xorm:"'android_version_name'"`

	// iOS
	IOSBundleID string `json:"ios_bundle_id" xorm:"'ios_bundle_id'"`
	IOSVersion  string `json:"ios_version" xorm:"'ios_version'"`
	IOSBuild    string `json:"ios_build" xorm:"'ios_build'"`

	// Windows
	WindowsAppName string `json:"windows_app_name" xorm:"'windows_app_name'"`
	WindowsVersion string `json:"windows_version" xorm:"'windows_version'"`

	// macOS
	MacOSBundleID string `json:"macos_bundle_id" xorm:"'macos_bundle_id'"`
	MacOSAppName  string `json:"macos_app_name" xorm:"'macos_app_name'"`
	MacOSVersion  string `json:"macos_version" xorm:"'macos_version'"`

	// 服务器配置
	ServerAPIURL   string `json:"server_api_url" xorm:"'server_api_url'"`
	ServerWSHost   string `json:"server_ws_host" xorm:"'server_ws_host'"`
	ServerWSPort   int    `json:"server_ws_port" xorm:"'server_ws_port'"`
	EnterpriseCode string `json:"enterprise_code" xorm:"'enterprise_code'"`

	// 资源
	IconURL   string `json:"icon_url" xorm:"'icon_url'"`
	LogoURL   string `json:"logo_url" xorm:"'logo_url'"`
	SplashURL string `json:"splash_url" xorm:"'splash_url'"`

	// Android 签名
	AndroidKeystoreURL      string `json:"android_keystore_url" xorm:"'android_keystore_url'"`
	AndroidKeystorePassword string `json:"-" xorm:"'android_keystore_password'"`
	AndroidKeyAlias         string `json:"android_key_alias" xorm:"'android_key_alias'"`
	AndroidKeyPassword      string `json:"-" xorm:"'android_key_password'"`

	// 推送配置
	PushMiAppID    string `json:"push_mi_app_id" xorm:"'push_mi_app_id'"`
	PushMiAppKey   string `json:"push_mi_app_key" xorm:"'push_mi_app_key'"`
	PushOppoAppKey string `json:"push_oppo_app_key" xorm:"'push_oppo_app_key'"`
	PushOppoAppSec string `json:"push_oppo_app_secret" xorm:"'push_oppo_app_secret'"`
	PushVivoAppID  string `json:"push_vivo_app_id" xorm:"'push_vivo_app_id'"`
	PushVivoAppKey string `json:"push_vivo_app_key" xorm:"'push_vivo_app_key'"`
	PushHmsAppID   string `json:"push_hms_app_id" xorm:"'push_hms_app_id'"`

	// Apple 开发者配置 (iOS/macOS)
	AppleTeamID              string `json:"apple_team_id" xorm:"'apple_team_id'"`                           // Apple 开发者团队 ID
	AppleCertificateURL      string `json:"apple_certificate_url" xorm:"'apple_certificate_url'"`           // P12 证书文件 URL
	AppleCertificatePassword string `json:"-" xorm:"'apple_certificate_password'"`                          // P12 证书密码
	AppleProvisioningURL     string `json:"apple_provisioning_url" xorm:"'apple_provisioning_url'"`         // iOS Provisioning Profile URL
	AppleMacProvisioningURL  string `json:"apple_mac_provisioning_url" xorm:"'apple_mac_provisioning_url'"` // macOS Provisioning Profile URL
	AppleExportMethod        string `json:"apple_export_method" xorm:"'apple_export_method'"`               // 导出方式: app-store, ad-hoc, enterprise, development

	// Git 源码配置
	GitRepoURL  string `json:"git_repo_url" xorm:"'git_repo_url'"`   // Git 仓库地址
	GitBranch   string `json:"git_branch" xorm:"'git_branch'"`       // 分支名（默认 main）
	GitTag      string `json:"git_tag" xorm:"'git_tag'"`             // 指定 tag（优先于分支）
	GitUsername string `json:"git_username" xorm:"'git_username'"`   // Git 用户名（私有仓库）
	GitToken    string `json:"-" xorm:"'git_token'"`                 // Git Token/密码（私有仓库）
}

func (BuildMerchant) TableName() string {
	return "build_merchants"
}

// BuildMerchantReq 创建/编辑打包配置请求
type BuildMerchantReq struct {
	MerchantID  *int   `json:"merchant_id"`
	Name        string `json:"name" binding:"required"`
	AppName     string `json:"app_name" binding:"required"`
	ShortName   string `json:"short_name" binding:"required"`
	Description string `json:"description"`

	AndroidPackage     string `json:"android_package" binding:"required"`
	AndroidVersionCode int    `json:"android_version_code"`
	AndroidVersionName string `json:"android_version_name"`
	IOSBundleID        string `json:"ios_bundle_id" binding:"required"`
	IOSVersion         string `json:"ios_version"`
	IOSBuild           string `json:"ios_build"`
	WindowsAppName     string `json:"windows_app_name"`
	WindowsVersion     string `json:"windows_version"`
	MacOSBundleID      string `json:"macos_bundle_id"`
	MacOSAppName       string `json:"macos_app_name"`
	MacOSVersion       string `json:"macos_version"`

	ServerAPIURL   string `json:"server_api_url"`
	ServerWSHost   string `json:"server_ws_host"`
	ServerWSPort   int    `json:"server_ws_port"`
	EnterpriseCode string `json:"enterprise_code"`

	PushMiAppID    string `json:"push_mi_app_id"`
	PushMiAppKey   string `json:"push_mi_app_key"`
	PushOppoAppKey string `json:"push_oppo_app_key"`
	PushOppoAppSec string `json:"push_oppo_app_secret"`
	PushVivoAppID  string `json:"push_vivo_app_id"`
	PushVivoAppKey string `json:"push_vivo_app_key"`
	PushHmsAppID   string `json:"push_hms_app_id"`

	// Apple 开发者配置
	AppleTeamID              string `json:"apple_team_id"`
	AppleCertificateURL      string `json:"apple_certificate_url"`
	AppleCertificatePassword string `json:"apple_certificate_password"`
	AppleProvisioningURL     string `json:"apple_provisioning_url"`
	AppleMacProvisioningURL  string `json:"apple_mac_provisioning_url"`
	AppleExportMethod        string `json:"apple_export_method"`

	// Git 源码配置
	GitRepoURL  string `json:"git_repo_url"`
	GitBranch   string `json:"git_branch"`
	GitTag      string `json:"git_tag"`
	GitUsername string `json:"git_username"`
	GitToken    string `json:"git_token"`
}

// ========== 构建任务 ==========

// BuildTask 构建任务
type BuildTask struct {
	ID              int        `json:"id" xorm:"pk autoincr 'id'"`
	BuildMerchantID int        `json:"build_merchant_id" xorm:"'build_merchant_id'"`
	MerchantName    string     `json:"merchant_name" xorm:"'merchant_name'"`
	Platforms       string     `json:"platforms" xorm:"'platforms'"`
	Status          int        `json:"status" xorm:"'status'"`
	Progress        int        `json:"progress" xorm:"'progress'"`
	CurrentStep     string     `json:"current_step" xorm:"'current_step'"`
	Operator        string     `json:"operator" xorm:"'operator'"`
	StartedAt       *time.Time `json:"started_at" xorm:"'started_at'"`
	FinishedAt      *time.Time `json:"finished_at" xorm:"'finished_at'"`
	Duration        int        `json:"duration" xorm:"'duration'"`
	ErrorMsg        string     `json:"error_msg" xorm:"'error_msg'"`
	LogURL          string     `json:"log_url" xorm:"'log_url'"`
	CreatedAt       time.Time  `json:"created_at" xorm:"'created_at'"`

	OverrideAndroidVersionCode *int    `json:"override_android_version_code" xorm:"'override_android_version_code'"`
	OverrideAndroidVersionName *string `json:"override_android_version_name" xorm:"'override_android_version_name'"`
	OverrideIOSVersion         *string `json:"override_ios_version" xorm:"'override_ios_version'"`
	OverrideIOSBuild           *string `json:"override_ios_build" xorm:"'override_ios_build'"`
}

func (BuildTask) TableName() string {
	return "build_tasks"
}

const (
	BuildStatusQueued    = 0
	BuildStatusBuilding  = 1
	BuildStatusSuccess   = 2
	BuildStatusFailed    = 3
	BuildStatusCancelled = 4
)

type CreateBuildTaskReq struct {
	BuildMerchantID            int     `json:"build_merchant_id" binding:"required"`
	Platforms                  string  `json:"platforms" binding:"required"`
	OverrideAndroidVersionCode *int    `json:"override_android_version_code"`
	OverrideAndroidVersionName *string `json:"override_android_version_name"`
	OverrideIOSVersion         *string `json:"override_ios_version"`
	OverrideIOSBuild           *string `json:"override_ios_build"`
}

// ========== 构建产物 ==========

type BuildArtifact struct {
	ID              int       `json:"id" xorm:"pk autoincr 'id'"`
	TaskID          int       `json:"task_id" xorm:"'task_id'"`
	BuildMerchantID int       `json:"build_merchant_id" xorm:"'build_merchant_id'"`
	MerchantName    string    `json:"merchant_name" xorm:"'merchant_name'"`
	Platform        string    `json:"platform" xorm:"'platform'"`
	FileName        string    `json:"file_name" xorm:"'file_name'"`
	FileSize        int64     `json:"file_size" xorm:"'file_size'"`
	FileURL         string    `json:"file_url" xorm:"'file_url'"`
	Version         string    `json:"version" xorm:"'version'"`
	ExpiresAt       time.Time `json:"expires_at" xorm:"'expires_at'"`
	DownloadCount   int       `json:"download_count" xorm:"'download_count'"`
	IsDeleted       int       `json:"is_deleted" xorm:"'is_deleted'"`
	CreatedAt       time.Time `json:"created_at" xorm:"'created_at'"`
}

func (BuildArtifact) TableName() string {
	return "build_artifacts"
}

// ========== 构建统计 ==========

type BuildStatsDaily struct {
	ID              int       `json:"id" xorm:"pk autoincr 'id'"`
	Date            time.Time `json:"date" xorm:"'date'"`
	TotalBuilds     int       `json:"total_builds" xorm:"'total_builds'"`
	SuccessBuilds   int       `json:"success_builds" xorm:"'success_builds'"`
	FailedBuilds    int       `json:"failed_builds" xorm:"'failed_builds'"`
	CancelledBuilds int       `json:"cancelled_builds" xorm:"'cancelled_builds'"`
	AndroidBuilds   int       `json:"android_builds" xorm:"'android_builds'"`
	IOSBuilds       int       `json:"ios_builds" xorm:"'ios_builds'"`
	WindowsBuilds   int       `json:"windows_builds" xorm:"'windows_builds'"`
	MacOSBuilds     int       `json:"macos_builds" xorm:"'macos_builds'"`
	TotalDuration   int       `json:"total_duration" xorm:"'total_duration'"`
	AvgDuration     int       `json:"avg_duration" xorm:"'avg_duration'"`
}

func (BuildStatsDaily) TableName() string {
	return "build_stats_daily"
}

// ========== 构建服务器 ==========

type BuildServer struct {
	ID            int        `json:"id" xorm:"pk autoincr 'id'"`
	Name          string     `json:"name" xorm:"'name'"`
	Host          string     `json:"host" xorm:"'host'"`
	Port          int        `json:"port" xorm:"'port'"`
	Username      string     `json:"username" xorm:"'username'"`
	AuthType      int        `json:"auth_type" xorm:"'auth_type'"`
	Password      string     `json:"-" xorm:"'password'"`
	PrivateKey    string     `json:"-" xorm:"'private_key'"`
	WorkDir       string     `json:"work_dir" xorm:"'work_dir'"`
	Platforms     string     `json:"platforms" xorm:"'platforms'"`
	MaxConcurrent int        `json:"max_concurrent" xorm:"'max_concurrent'"`
	CurrentTasks  int        `json:"current_tasks" xorm:"'current_tasks'"`
	Status        int        `json:"status" xorm:"'status'"`
	LastHeartbeat *time.Time `json:"last_heartbeat" xorm:"'last_heartbeat'"`
	Description   string     `json:"description" xorm:"'description'"`
	CreatedAt     time.Time  `json:"created_at" xorm:"'created_at'"`
	UpdatedAt     time.Time  `json:"updated_at" xorm:"'updated_at'"`
}

func (BuildServer) TableName() string {
	return "build_servers"
}

// ========== 统计响应 ==========

type BuildStatsResp struct {
	Today struct {
		Total     int     `json:"total"`
		Success   int     `json:"success"`
		Failed    int     `json:"failed"`
		Rate      float64 `json:"rate"`
		Building  int     `json:"building"`
		AvgSecond int     `json:"avg_second"`
	} `json:"today"`
	Week struct {
		Total   int `json:"total"`
		Success int `json:"success"`
		Failed  int `json:"failed"`
	} `json:"week"`
	Platforms struct {
		Android int `json:"android"`
		IOS     int `json:"ios"`
		Windows int `json:"windows"`
		MacOS   int `json:"macos"`
	} `json:"platforms"`
}
