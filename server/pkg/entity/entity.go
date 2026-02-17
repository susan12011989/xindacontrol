package entity

import (
	"time"
)

type AdminUsers struct {
	Id               int       `xorm:"not null pk autoincr INT"`
	Role             string    `xorm:"not null comment('角色') VARCHAR(32)"` // user, admin
	Username         string    `xorm:"not null unique VARCHAR(16)"`
	Password         string    `xorm:"not null VARCHAR(32)"`
	TwoFactorSecret  string    `xorm:"default '' comment('2FA密钥(Base32编码)') VARCHAR(32)"`
	TwoFactorEnabled int       `xorm:"default 0 comment('是否启用2FA') index TINYINT(1)"`
	CreatedAt        time.Time `xorm:"default CURRENT_TIMESTAMP DATETIME"`
	UpdatedAt        time.Time `xorm:"default CURRENT_TIMESTAMP DATETIME"`
}

// 套餐配置
type PackageConfiguration struct {
	DirectIP         string   `json:"direct_ip"`          // 直连ip 也就是aws自己的ip
	DirectPort       int32    `json:"direct_port"`        // 直连端口
	Port             int      `json:"port"`               // 端口
	DauLimit         int      `json:"dau_limit"`          // 日活限制
	RegisterLimit    int      `json:"register_limit"`     // 注册人数限制
	GroupMemberLimit int      `json:"group_member_limit"` // 人数限制
	ExpiredAt        int64    `json:"expired_at"`         // 套餐到期时间(秒级时间戳)
	AppPackages      []string `json:"app_packages"`       // 套餐内可用应用列表
	TurnServer       string   `json:"turn_server"`        // TURN服务器地址 (格式: ip:port)
}

type CloudAccountDetail struct {
	AccessKey    string `json:"access_key,omitempty"`
	AccessSecret string `json:"access_secret,omitempty"`
}

// 云账号配置
type CloudAccount struct {
	Aliyun  *CloudAccountDetail `json:"aliyun,omitempty"`
	Huawei  *CloudAccountDetail `json:"huawei,omitempty"`
	Tencent *CloudAccountDetail `json:"tencent,omitempty"`
	Aws     *CloudAccountDetail `json:"aws,omitempty"`
}
type DcOption struct {
	Ip   string `json:"ip"`
	Port int32  `json:"port"`
}
type Configs struct {
	DcOptions []DcOption `json:"dc_options,omitempty"` // 数据中心,客户端首选服务器地址列表
}
type AppConfigs struct {
	OssUrl []string `json:"oss_url,omitempty"` // ip文件 oss地址列表
}

type Merchants struct {
	Id                   int                   `xorm:"not null pk autoincr INT"`
	No                   string                `xorm:"not null unique VARCHAR(16)"`
	ServerIP             string                `xorm:"server_ip not null comment('服务器IP') VARCHAR(128)"`
	Port                 int                   `xorm:"not null default 0 comment('商户端口') INT"` // 商户端口
	Name                 string                `xorm:"not null comment('商户名') VARCHAR(64)"`
	AppName              string                `xorm:"comment('应用名称') VARCHAR(64)"`  // 应用显示名称，用于打包
	LogoUrl              string                `xorm:"comment('Logo地址') VARCHAR(512)"` // Logo URL，用于打包
	IconUrl              string                `xorm:"comment('图标地址') VARCHAR(512)"` // 应用图标 URL，用于打包
	Status               int                   `xorm:"not null default 1 comment('状态') TINYINT"`
	PackageConfiguration *PackageConfiguration `xorm:"comment('套餐限制') JSON"`
	Configs              *Configs              `xorm:"comment('全局配置') JSON"`
	AppConfigs           *AppConfigs           `xorm:"comment('应用配置') JSON"`
	CreatedAt            time.Time             `xorm:"default CURRENT_TIMESTAMP DATETIME"`
	UpdatedAt            time.Time             `xorm:"default CURRENT_TIMESTAMP DATETIME"`
	ExpiredAt            time.Time             `xorm:"not null default CURRENT_TIMESTAMP DATETIME"`
	ProjectId            int                   `xorm:"default 0 comment('所属项目ID') INT"`
}

// ========== 服务器运维管理 ==========

// 服务器配置
type Servers struct {
	Id          int       `xorm:"not null pk autoincr INT"`
	ServerType  int       `xorm:"not null default 1 comment('服务器类型:1-商户服务器 2-系统服务器') TINYINT"`
	UsageType   int       `xorm:"not null default 0 comment('用途:0-通用 1-商户专属GOST 2-系统共享GOST') TINYINT"`
	ForwardType int       `xorm:"not null default 1 comment('转发类型:1-加密(relay+tls) 2-直连(tcp)') TINYINT"`
	MerchantId  int       `xorm:"comment('商户ID') INT"`
	Name        string    `xorm:"not null comment('服务器名称') VARCHAR(64)"`
	Host        string    `xorm:"not null comment('服务器地址') VARCHAR(128)"`
	AuxiliaryIP string    `xorm:"auxiliary_ip default '' comment('辅助IP,仅系统服务器使用') VARCHAR(128)"`
	Port        int       `xorm:"not null default 22 comment('SSH端口') INT"`
	Username    string    `xorm:"not null comment('SSH用户名') VARCHAR(32)"`
	AuthType    int       `xorm:"not null default 1 comment('认证方式:1-密码 2-密钥') TINYINT"`
	Password    string    `xorm:"default '' comment('SSH密码') VARCHAR(128)"`
	PrivateKey  string    `xorm:"comment('SSH私钥') TEXT"`
	DeployPath  string    `xorm:"default '/opt/teamgram/bin' comment('部署目录') VARCHAR(255)"`
	Status        int        `xorm:"not null default 1 comment('状态:0-停用 1-启用') TINYINT"`
	TlsEnabled    int        `xorm:"not null default 0 comment('客户端TLS:0-未启用 1-已启用') TINYINT"`
	TlsDeployedAt *time.Time `xorm:"comment('TLS证书部署时间') DATETIME"`
	Description   string     `xorm:"default '' comment('描述') VARCHAR(255)"`
	Tags          string     `xorm:"default '' comment('标签') VARCHAR(255)"`
	CreatedAt     time.Time  `xorm:"default CURRENT_TIMESTAMP DATETIME"`
	UpdatedAt     time.Time  `xorm:"default CURRENT_TIMESTAMP DATETIME"`
}

// 转发类型常量
const (
	ForwardTypeEncrypted = 1 // 加密转发 (relay+tls) -> 商户GOST 10010/11/12 -> 业务程序 10000/01/02
	ForwardTypeDirect    = 2 // 直连转发 (tcp) -> 商户业务程序 10000/01/02
)

// TLS 证书
type TlsCertificates struct {
	Id          int       `xorm:"not null pk autoincr INT"`
	Name        string    `xorm:"not null unique comment('证书名称') VARCHAR(64)"`
	CertType    int       `xorm:"not null default 1 comment('证书类型:1-CA根证书 2-服务器证书') TINYINT"`
	CertPem     string    `xorm:"not null comment('证书内容PEM') TEXT"`
	KeyPem      string    `xorm:"not null comment('私钥内容PEM') TEXT"`
	Fingerprint string    `xorm:"not null default '' comment('SHA-256指纹') VARCHAR(128)"`
	ExpiresAt   time.Time `xorm:"not null comment('过期时间') DATETIME"`
	Status      int       `xorm:"not null default 1 comment('状态:0-停用 1-启用') TINYINT"`
	CreatedAt   time.Time `xorm:"default CURRENT_TIMESTAMP DATETIME"`
	UpdatedAt   time.Time `xorm:"default CURRENT_TIMESTAMP DATETIME"`
}

// 部署配置
type DeployConfigs struct {
	Id               int       `xorm:"not null pk autoincr INT"`
	ServerId         int       `xorm:"not null comment('服务器ID') INT"`
	Name             string    `xorm:"not null comment('配置名称') VARCHAR(64)"`
	ServiceName      string    `xorm:"not null comment('服务名称') VARCHAR(64)"`
	DeployPath       string    `xorm:"not null comment('部署目录') VARCHAR(255)"`
	StartCommand     string    `xorm:"not null comment('启动命令') TEXT"`
	StopCommand      string    `xorm:"not null comment('停止命令') TEXT"`
	RestartCommand   string    `xorm:"comment('重启命令') TEXT"`
	StatusCommand    string    `xorm:"comment('状态查询命令') TEXT"`
	LogPath          string    `xorm:"comment('日志路径') VARCHAR(255)"`
	PreDeployScript  string    `xorm:"comment('部署前脚本') TEXT"`
	PostDeployScript string    `xorm:"comment('部署后脚本') TEXT"`
	EnvVars          string    `xorm:"comment('环境变量') TEXT"`
	StartOrder       int       `xorm:"not null default 0 comment('启动顺序') INT"`
	SleepAfter       int       `xorm:"not null default 1 comment('启动后等待秒数') INT"`
	ServiceGroup     string    `xorm:"default '' comment('服务分组') VARCHAR(32)"`
	CreatedAt        time.Time `xorm:"default CURRENT_TIMESTAMP DATETIME"`
	UpdatedAt        time.Time `xorm:"default CURRENT_TIMESTAMP DATETIME"`
}

// 部署历史
type DeployHistory struct {
	Id          int       `xorm:"not null pk autoincr INT"`
	ServerId    int       `xorm:"not null comment('服务器ID') INT"`
	ConfigId    int       `xorm:"comment('配置ID') INT"`
	Action      string    `xorm:"not null comment('操作类型') VARCHAR(32)"`
	ServiceName string    `xorm:"default '' comment('服务名称') VARCHAR(64)"`
	Operator    string    `xorm:"not null comment('操作人') VARCHAR(32)"`
	Status      int       `xorm:"not null default 0 comment('状态') TINYINT"`
	Output      string    `xorm:"comment('执行输出') TEXT"`
	ErrorMsg    string    `xorm:"comment('错误信息') TEXT"`
	Duration    int       `xorm:"default 0 comment('执行时长') INT"`
	CreatedAt   time.Time `xorm:"default CURRENT_TIMESTAMP DATETIME"`
}

// Docker操作历史
type DockerOperationHistory struct {
	Id            int       `xorm:"not null pk autoincr INT"`
	ServerId      int       `xorm:"not null comment('服务器ID') INT"`
	MerchantId    int       `xorm:"not null comment('商户ID') INT"`
	ContainerId   string    `xorm:"not null comment('容器ID') VARCHAR(64)"`
	ContainerName string    `xorm:"default '' comment('容器名称') VARCHAR(128)"`
	Action        string    `xorm:"not null comment('操作') VARCHAR(32)"`
	Operator      string    `xorm:"not null comment('操作人') VARCHAR(32)"`
	Params        string    `xorm:"comment('操作参数') TEXT"`
	Status        int       `xorm:"not null default 1 comment('状态') TINYINT"`
	Output        string    `xorm:"comment('执行输出') TEXT"`
	ErrorMsg      string    `xorm:"comment('错误信息') TEXT"`
	CreatedAt     time.Time `xorm:"default CURRENT_TIMESTAMP DATETIME"`
}

// 公告发送日志
type AnnouncementLogs struct {
	Id          int64     `xorm:"not null pk autoincr BIGINT"`
	Text        string    `xorm:"comment('发送内容') TEXT"`
	Entities    string    `xorm:"comment('消息实体JSON') TEXT"`
	Silent      int       `xorm:"default 1 comment('静音') TINYINT(1)"`
	NoForwards  int       `xorm:"default 1 comment('禁转发') TINYINT(1)"`
	MerchantNos string    `xorm:"comment('选择的商户no(数组JSON)') TEXT"`
	Broadcast   int       `xorm:"default 0 comment('是否全部广播') TINYINT(1)"`
	CreatedAt   time.Time `xorm:"default CURRENT_TIMESTAMP DATETIME"`
}

// 系统云账号
type CloudAccounts struct {
	Id              int64     `xorm:"not null pk autoincr BIGINT"`
	AccountType     string    `xorm:"not null comment('账号类型: system, merchant') VARCHAR(32)"`
	MerchantId      int       `xorm:"comment('商户ID') INT"`
	Name            string    `xorm:"not null comment('账号名称') VARCHAR(100)"`
	CloudType       string    `xorm:"not null comment('云类型: aliyun, aws, tencent') index VARCHAR(20)"`
	SiteType        string    `xorm:"default 'cn' comment('站点类型: cn-国内站, intl-国际站') VARCHAR(10)"`
	AccessKeyId     string    `xorm:"not null comment('AccessKeyId') VARCHAR(255)"`
	AccessKeySecret string    `xorm:"not null comment('AccessKeySecret') VARCHAR(255)"`
	Description     string    `xorm:"default '' comment('描述') VARCHAR(500)"`
	Status          int       `xorm:"default 1 comment('状态: 0-禁用 1-启用') index TINYINT"`
	CreatedAt       time.Time `xorm:"default CURRENT_TIMESTAMP DATETIME"`
	UpdatedAt       time.Time `xorm:"default CURRENT_TIMESTAMP DATETIME"`
}

type GlobalOssUrl struct {
	Id        int       `xorm:"not null pk autoincr INT"`
	Url       string    `xorm:"url"`
	UpdatedAt time.Time `xorm:"default CURRENT_TIMESTAMP DATETIME"`
}

type SmsConfig struct {
	Provider string `json:"provider"` // "aliyun"(默认), "unisms", "smsbao"
	// 阿里云
	RegionId     string `json:"region_id"`
	AccessKey    string `json:"access_key"`
	SecretKey    string `json:"secret_key"`
	SignName     string `json:"sign_name"`
	TemplateCode string `json:"template_code"`
	// UniSMS (联合短信)
	UnismsAccessKeyID     string `json:"unisms_access_key_id"`
	UnismsAccessKeySecret string `json:"unisms_access_key_secret"`
	UnismsSignature       string `json:"unisms_signature"`
	UnismsTemplateId      string `json:"unisms_template_id"`
	// 短信宝
	SmsbaoAccount  string `json:"smsbao_account"`
	SmsbaoApiKey   string `json:"smsbao_api_key"`
	SmsbaoTemplate string `json:"smsbao_template"`
}
type PushHMS struct {
	AppId     string `json:"app_id"`
	AppSecret string `json:"app_secret"`
}

type PushXiaomi struct {
	AppSecret  string `json:"app_secret"`
	Package    string `json:"package"`
	ChannelId  string `json:"channel_id"`
	TimeToLive int    `json:"time_to_live"`
}

type PushOppo struct {
	AppKey       string `json:"app_key"`
	MasterSecret string `json:"master_secret"`
	ChannelId    string `json:"channel_id"`
	TimeToLive   int    `json:"time_to_live"`
}

type PushVivo struct {
	AppId      int64  `json:"app_id"`
	AppKey     string `json:"app_key"`
	AppSecret  string `json:"app_secret"`
	TimeToLive int    `json:"time_to_live"`
}

type PushHonor struct {
	AppID        string `json:"app_id"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	TimeToLive   int    `json:"time_to_live"`
}

type AllPushConfig struct {
	PushHMS    *PushHMS    `json:"push_hms"`
	PushXiaomi *PushXiaomi `json:"push_xiaomi"`
	PushOppo   *PushOppo   `json:"push_oppo"`
	PushVivo   *PushVivo   `json:"push_vivo"`
	PushHonor  *PushHonor  `json:"push_honor"`
}

type TrtcConfig struct {
	AppId  int    `json:"app_id"`
	AppKey string `json:"app_key"`
}

// 客户端包列表
type Clients struct {
	Id             int            `xorm:"not null pk autoincr INT"`
	AppPackageName string         `xorm:"not null comment('安卓包名') VARCHAR(100)"`
	AppName        string         `xorm:"not null comment('app名称') VARCHAR(100)"`
	SmsConfig      *SmsConfig     `xorm:"comment('短信配置') JSON"`   // 短信配置
	PushConfig     *AllPushConfig `xorm:"comment('推送配置') JSON"`   // 推送配置
	TrtcConfig     *TrtcConfig    `xorm:"comment('TRTC配置') JSON"` // TRTC配置
}

// IP嵌入选择记录
type IpEmbedSelections struct {
	Id          int       `xorm:"not null pk autoincr INT"`
	KeyName     string    `xorm:"not null unique comment('配置键名') VARCHAR(64)"`
	SelectedIPs string    `xorm:"selected_ips comment('选中的IP列表(JSON数组)') TEXT"`
	UpdatedAt   time.Time `xorm:"default CURRENT_TIMESTAMP DATETIME"`
}

// IP嵌入上传目标配置
type IpEmbedTargets struct {
	Id             int       `xorm:"not null pk autoincr INT"`
	Name           string    `xorm:"not null comment('目标名称') VARCHAR(64)"`
	CloudType      string    `xorm:"not null comment('云类型: aliyun, aws, tencent') VARCHAR(20)"`
	CloudAccountId int64     `xorm:"not null comment('云账号ID') BIGINT"`
	RegionId       string    `xorm:"not null comment('区域ID') VARCHAR(64)"`
	Bucket         string    `xorm:"not null comment('Bucket名称') VARCHAR(128)"`
	ObjectPrefix   string    `xorm:"default '' comment('对象前缀') VARCHAR(128)"`
	Enabled        int       `xorm:"not null default 1 comment('是否启用:0-禁用 1-启用') TINYINT"`
	SortOrder      int       `xorm:"not null default 0 comment('排序顺序') INT"`
	CreatedAt      time.Time `xorm:"default CURRENT_TIMESTAMP DATETIME"`
	UpdatedAt      time.Time `xorm:"default CURRENT_TIMESTAMP DATETIME"`
}

// ========== 功能开关管理 ==========

// 功能开关配置
type FeatureFlags struct {
	Id          int       `xorm:"not null pk autoincr INT"`
	MerchantId  int       `xorm:"not null comment('商户ID') unique(merchant_feature) INT"`
	FeatureName string    `xorm:"not null comment('功能名称') unique(merchant_feature) VARCHAR(64)"`
	Enabled     int       `xorm:"not null default 1 comment('是否启用:0-禁用 1-启用') TINYINT"`
	Description string    `xorm:"default '' comment('功能描述') VARCHAR(255)"`
	CreatedAt   time.Time `xorm:"default CURRENT_TIMESTAMP DATETIME"`
	UpdatedAt   time.Time `xorm:"default CURRENT_TIMESTAMP DATETIME"`
}

// 预定义的功能模块
const (
	FeatureRedPacket = "redpacket" // 红包功能
	FeatureMoments   = "moments"   // 朋友圈功能
	FeatureRTC       = "rtc"       // 实时通话功能
	FeatureWallet    = "wallet"    // 钱包功能
	FeatureGroup     = "group"     // 群组功能
	FeatureTransfer  = "transfer"  // 转账功能
	FeatureSticker   = "sticker"   // 表情包功能
	FeatureLocation  = "location"  // 位置分享功能
)

// ========== 商户独立 OSS 和 GOST 配置 ==========

// 商户 OSS 配置（每商户可配置多个 OSS，引用 cloud_accounts 避免重复存储凭证）
type MerchantOssConfigs struct {
	Id             int       `xorm:"not null pk autoincr INT"`
	MerchantId     int       `xorm:"not null comment('商户ID') index INT"`
	CloudAccountId int64     `xorm:"not null comment('云账号ID') index BIGINT"`
	Name           string    `xorm:"not null comment('配置名称') VARCHAR(64)"`
	Bucket         string    `xorm:"not null comment('Bucket名称') VARCHAR(128)"`
	Region         string    `xorm:"default '' comment('区域') VARCHAR(64)"`
	Endpoint       string    `xorm:"default '' comment('OSS Endpoint,留空自动生成') VARCHAR(255)"`
	CustomDomain   string    `xorm:"default '' comment('自定义域名CDN') VARCHAR(255)"`
	IsDefault      int       `xorm:"not null default 0 comment('是否默认OSS') TINYINT"`
	Status         int       `xorm:"not null default 1 comment('状态:0-禁用 1-启用') TINYINT"`
	CreatedAt      time.Time `xorm:"default CURRENT_TIMESTAMP DATETIME"`
	UpdatedAt      time.Time `xorm:"default CURRENT_TIMESTAMP DATETIME"`
}

// 商户 GOST 服务器关联（每商户可配置多个 GOST 转发服务器）
type MerchantGostServers struct {
	Id         int       `xorm:"not null pk autoincr INT"`
	MerchantId int       `xorm:"not null comment('商户ID') index INT"`
	ServerId   int       `xorm:"not null comment('服务器ID') index INT"`
	CloudType  string    `xorm:"default '' comment('云类型: aliyun, tencent, aws') VARCHAR(20)"`
	Region     string    `xorm:"default '' comment('区域/地区') VARCHAR(64)"`
	ListenPort int       `xorm:"not null default 0 comment('监听端口,0使用商户默认端口') INT"`
	IsPrimary  int       `xorm:"not null default 0 comment('是否主转发服务器') TINYINT"`
	Priority   int       `xorm:"not null default 0 comment('优先级,数字越小越高') INT"`
	Status     int       `xorm:"not null default 1 comment('状态:0-禁用 1-启用') TINYINT"`
	Remark     string    `xorm:"default '' comment('备注') VARCHAR(255)"`
	CreatedAt  time.Time `xorm:"default CURRENT_TIMESTAMP DATETIME"`
	UpdatedAt  time.Time `xorm:"default CURRENT_TIMESTAMP DATETIME"`
}

// 服务器用途类型常量
const (
	ServerUsageGeneral         = 0 // 通用
	ServerUsageMerchantGost    = 1 // 商户专属 GOST
	ServerUsageSystemSharedGost = 2 // 系统共享 GOST
)

// OSS 云类型常量
const (
	OssCloudTypeAliyun  = "aliyun"
	OssCloudTypeTencent = "tencent"
	OssCloudTypeAws     = "aws"
	OssCloudTypeMinio   = "minio"
)

// ========== 操作审计日志 ==========

// AuditLogs 操作审计日志
type AuditLogs struct {
	Id         int64     `xorm:"not null pk autoincr BIGINT"`
	UserId     int       `xorm:"not null comment('操作用户ID') index INT"`
	Username   string    `xorm:"not null comment('操作用户名') VARCHAR(32)"`
	Action     string    `xorm:"not null comment('操作类型') index VARCHAR(64)"`
	TargetType string    `xorm:"not null comment('目标类型') index VARCHAR(32)"`
	TargetId   int       `xorm:"not null default 0 comment('目标ID') INT"`
	TargetName string    `xorm:"default '' comment('目标名称') VARCHAR(128)"`
	Detail     string    `xorm:"comment('操作详情JSON') TEXT"`
	IP         string    `xorm:"not null default '' comment('操作IP') VARCHAR(64)"`
	UserAgent  string    `xorm:"default '' comment('浏览器UA') VARCHAR(512)"`
	Status     string    `xorm:"not null default 'success' comment('操作状态') VARCHAR(16)"`
	ErrorMsg   string    `xorm:"default '' comment('错误信息') VARCHAR(512)"`
	CreatedAt  time.Time `xorm:"default CURRENT_TIMESTAMP index DATETIME"`
}

// 审计日志操作类型常量
const (
	// 商户操作
	AuditActionCreateMerchant = "create_merchant"
	AuditActionUpdateMerchant = "update_merchant"
	AuditActionDeleteMerchant = "delete_merchant"
	AuditActionChangeMerchantIP   = "change_merchant_ip"
	AuditActionChangeGostPort     = "change_gost_port"

	// 服务器操作
	AuditActionCreateServer = "create_server"
	AuditActionUpdateServer = "update_server"
	AuditActionDeleteServer = "delete_server"

	// 云账号操作
	AuditActionCreateCloudAccount = "create_cloud_account"
	AuditActionUpdateCloudAccount = "update_cloud_account"
	AuditActionDeleteCloudAccount = "delete_cloud_account"

	// OSS 配置操作
	AuditActionCreateOssConfig = "create_oss_config"
	AuditActionUpdateOssConfig = "update_oss_config"
	AuditActionDeleteOssConfig = "delete_oss_config"

	// GOST 服务器操作
	AuditActionCreateGostServer = "create_gost_server"
	AuditActionUpdateGostServer = "update_gost_server"
	AuditActionDeleteGostServer = "delete_gost_server"

	// 用户操作
	AuditActionLogin  = "login"
	AuditActionLogout = "logout"
)

// 审计日志目标类型常量
const (
	AuditTargetMerchant     = "merchant"
	AuditTargetServer       = "server"
	AuditTargetCloudAccount = "cloud_account"
	AuditTargetOssConfig    = "oss_config"
	AuditTargetGostServer   = "gost_server"
	AuditTargetUser         = "user"
)

// ========== 告警通知系统 ==========

// AlertRules 告警规则
type AlertRules struct {
	Id              int       `xorm:"not null pk autoincr INT"`
	Name            string    `xorm:"not null comment('规则名称') VARCHAR(64)"`
	Type            string    `xorm:"not null comment('告警类型') index VARCHAR(32)"`
	Threshold       float64   `xorm:"default 0 comment('阈值') DECIMAL(10,2)"`
	MerchantId      int       `xorm:"default 0 comment('商户ID') index INT"`
	NotifyType      string    `xorm:"not null default 'webhook' comment('通知方式') VARCHAR(32)"`
	NotifyUrl       string    `xorm:"default '' comment('Webhook URL') VARCHAR(512)"`
	NotifyEmail     string    `xorm:"default '' comment('通知邮箱') VARCHAR(128)"`
	NotifyPhone     string    `xorm:"default '' comment('通知手机号') VARCHAR(32)"`
	IntervalMinutes int       `xorm:"not null default 60 comment('告警间隔分钟') INT"`
	Status          int       `xorm:"not null default 1 comment('状态') index TINYINT"`
	Description     string    `xorm:"default '' comment('规则描述') VARCHAR(255)"`
	CreatedAt       time.Time `xorm:"default CURRENT_TIMESTAMP DATETIME"`
	UpdatedAt       time.Time `xorm:"default CURRENT_TIMESTAMP DATETIME"`
}

// AlertLogs 告警日志
type AlertLogs struct {
	Id           int64     `xorm:"not null pk autoincr BIGINT"`
	RuleId       int       `xorm:"not null comment('规则ID') index INT"`
	RuleName     string    `xorm:"not null comment('规则名称') VARCHAR(64)"`
	Type         string    `xorm:"not null comment('告警类型') index VARCHAR(32)"`
	Level        string    `xorm:"not null default 'warning' comment('告警级别') VARCHAR(16)"`
	TargetType   string    `xorm:"not null comment('目标类型') VARCHAR(32)"`
	TargetId     int       `xorm:"not null default 0 comment('目标ID') INT"`
	TargetName   string    `xorm:"default '' comment('目标名称') VARCHAR(128)"`
	Message      string    `xorm:"not null comment('告警消息') VARCHAR(512)"`
	Detail       string    `xorm:"comment('告警详情JSON') TEXT"`
	NotifyStatus string    `xorm:"not null default 'pending' comment('通知状态') index VARCHAR(16)"`
	NotifyResult string    `xorm:"default '' comment('通知结果') VARCHAR(512)"`
	CreatedAt    time.Time `xorm:"default CURRENT_TIMESTAMP index DATETIME"`
}

// 告警类型常量
const (
	AlertTypeMerchantExpired = "merchant_expired" // 商户即将过期
	AlertTypeServerDown      = "server_down"      // 服务器宕机
	AlertTypeCpuHigh         = "cpu_high"         // CPU 使用率过高
	AlertTypeMemoryHigh      = "memory_high"      // 内存使用率过高
	AlertTypeDiskHigh        = "disk_high"        // 磁盘使用率过高
	AlertTypeServiceDown     = "service_down"     // 服务异常
)

// 告警级别常量
const (
	AlertLevelInfo     = "info"
	AlertLevelWarning  = "warning"
	AlertLevelError    = "error"
	AlertLevelCritical = "critical"
)

// 通知类型常量
const (
	NotifyTypeWebhook = "webhook"
	NotifyTypeEmail   = "email"
	NotifyTypeSms     = "sms"
)

// 通知状态常量
const (
	NotifyStatusPending = "pending"
	NotifyStatusSent    = "sent"
	NotifyStatusFailed  = "failed"
)

// ========== API 测试系统 ==========

// APITestCases API 测试用例
type APITestCases struct {
	Id               int64     `xorm:"not null pk autoincr BIGINT"`
	MerchantId       int       `xorm:"not null comment('商户ID') index INT"`
	Name             string    `xorm:"not null comment('用例名称') VARCHAR(128)"`
	Module           string    `xorm:"not null default '' comment('模块分类') index VARCHAR(32)"`
	Method           string    `xorm:"not null comment('HTTP方法') VARCHAR(10)"`
	Path             string    `xorm:"not null comment('请求路径') VARCHAR(255)"`
	Headers          string    `xorm:"comment('请求头JSON') TEXT"`
	QueryParams      string    `xorm:"comment('查询参数JSON') TEXT"`
	Body             string    `xorm:"comment('请求体') TEXT"`
	ExpectedStatus   int       `xorm:"not null default 200 comment('期望状态码') INT"`
	ExpectedContains string    `xorm:"default '' comment('期望包含内容') VARCHAR(512)"`
	LastRunAt        time.Time `xorm:"comment('最后运行时间') DATETIME"`
	LastRunStatus    int       `xorm:"not null default 0 comment('最后运行状态:0-未运行 1-成功 2-失败') TINYINT"`
	CreatedAt        time.Time `xorm:"default CURRENT_TIMESTAMP DATETIME"`
	UpdatedAt        time.Time `xorm:"default CURRENT_TIMESTAMP DATETIME"`
}

// APIMonitorConfigs API 监控配置
type APIMonitorConfigs struct {
	Id           int64     `xorm:"not null pk autoincr BIGINT"`
	MerchantId   int       `xorm:"not null comment('商户ID') index INT"`
	Name         string    `xorm:"not null comment('监控名称') VARCHAR(128)"`
	TestCaseIds  string    `xorm:"comment('测试用例ID列表JSON') TEXT"`
	Interval     int       `xorm:"not null default 60 comment('检测间隔秒') INT"`
	Enabled      int       `xorm:"not null default 1 comment('是否启用') TINYINT"`
	AlertEmail   string    `xorm:"default '' comment('告警邮箱') VARCHAR(128)"`
	AlertWebhook string    `xorm:"default '' comment('告警Webhook') VARCHAR(512)"`
	LastRunAt    time.Time `xorm:"comment('最后运行时间') DATETIME"`
	LastStatus   int       `xorm:"not null default 0 comment('最后运行状态:0-未运行 1-正常 2-异常') TINYINT"`
	CreatedAt    time.Time `xorm:"default CURRENT_TIMESTAMP DATETIME"`
	UpdatedAt    time.Time `xorm:"default CURRENT_TIMESTAMP DATETIME"`
}

// APIMonitorHistory API 监控历史
type APIMonitorHistory struct {
	Id        int64     `xorm:"not null pk autoincr BIGINT"`
	MonitorId int64     `xorm:"not null comment('监控配置ID') index BIGINT"`
	Total     int       `xorm:"not null default 0 comment('总测试数') INT"`
	Success   int       `xorm:"not null default 0 comment('成功数') INT"`
	Failed    int       `xorm:"not null default 0 comment('失败数') INT"`
	TotalTime int64     `xorm:"not null default 0 comment('总耗时毫秒') BIGINT"`
	Results   string    `xorm:"comment('详细结果JSON') TEXT"`
	CreatedAt time.Time `xorm:"default CURRENT_TIMESTAMP index DATETIME"`
}

// ========== 版本管理系统 ==========

// ServiceVersions 服务版本注册表
type ServiceVersions struct {
	Id          int       `xorm:"not null pk autoincr INT"`
	ServiceName string    `xorm:"not null comment('服务名称:server/wukongim') index VARCHAR(32)"`
	Version     string    `xorm:"not null comment('版本号') VARCHAR(64)"`
	FileHash    string    `xorm:"not null comment('文件SHA256') VARCHAR(64)"`
	FileSize    int64     `xorm:"not null default 0 comment('文件大小') BIGINT"`
	FilePath    string    `xorm:"not null comment('存储路径') VARCHAR(255)"`
	Changelog   string    `xorm:"comment('更新日志') TEXT"`
	IsCurrent   int       `xorm:"not null default 0 comment('是否当前版本') TINYINT"`
	UploadedBy  string    `xorm:"default '' comment('上传者') VARCHAR(32)"`
	CreatedAt   time.Time `xorm:"default CURRENT_TIMESTAMP DATETIME"`
}

// DeploymentRecords 部署记录
type DeploymentRecords struct {
	Id                int64     `xorm:"not null pk autoincr BIGINT"`
	ServerId          int       `xorm:"not null comment('服务器ID') index INT"`
	ServiceName       string    `xorm:"not null comment('服务名称') VARCHAR(32)"`
	VersionId         int       `xorm:"not null comment('版本ID') INT"`
	PreviousVersionId int       `xorm:"default 0 comment('上一版本ID') INT"`
	Action            string    `xorm:"not null comment('操作:deploy/rollback') VARCHAR(16)"`
	Status            int       `xorm:"not null default 0 comment('状态:0-进行中 1-成功 2-失败') TINYINT"`
	Operator          string    `xorm:"default '' comment('操作人') VARCHAR(32)"`
	BackupPath        string    `xorm:"default '' comment('备份路径') VARCHAR(255)"`
	Output            string    `xorm:"comment('执行输出') TEXT"`
	StartedAt         time.Time `xorm:"comment('开始时间') DATETIME"`
	CompletedAt       time.Time `xorm:"comment('完成时间') DATETIME"`
	CreatedAt         time.Time `xorm:"default CURRENT_TIMESTAMP DATETIME"`
}

// 部署操作类型常量
const (
	DeployActionDeploy   = "deploy"
	DeployActionRollback = "rollback"
)

// 部署状态常量
const (
	DeployStatusPending = 0
	DeployStatusSuccess = 1
	DeployStatusFailed  = 2
)

// ========== 项目管理 ==========

// Projects 项目表
type Projects struct {
	Id          int       `xorm:"not null pk autoincr INT"`
	Name        string    `xorm:"not null unique comment('项目名称') VARCHAR(64)"`
	Description string    `xorm:"default '' comment('项目描述') VARCHAR(255)"`
	Status      int       `xorm:"not null default 1 comment('状态:0-禁用 1-启用') TINYINT"`
	CreatedAt   time.Time `xorm:"default CURRENT_TIMESTAMP DATETIME"`
	UpdatedAt   time.Time `xorm:"default CURRENT_TIMESTAMP DATETIME"`
}

// ProjectGostServers 项目 GOST 服务器关联表
type ProjectGostServers struct {
	Id        int       `xorm:"not null pk autoincr INT"`
	ProjectId int       `xorm:"not null comment('项目ID') index INT"`
	ServerId  int       `xorm:"not null comment('服务器ID') index INT"`
	IsPrimary int       `xorm:"not null default 0 comment('是否主服务器') TINYINT"`
	Priority  int       `xorm:"not null default 0 comment('优先级,数字越小越高') INT"`
	Status    int       `xorm:"not null default 1 comment('状态:0-禁用 1-启用') TINYINT"`
	Remark    string    `xorm:"default '' comment('备注') VARCHAR(255)"`
	CreatedAt time.Time `xorm:"default CURRENT_TIMESTAMP DATETIME"`
	UpdatedAt time.Time `xorm:"default CURRENT_TIMESTAMP DATETIME"`
}

// ========== 商户存储配置管理 ==========

// MerchantStorageConfigs 商户存储配置（用于推送到商户服务器）
type MerchantStorageConfigs struct {
	Id              int       `xorm:"not null pk autoincr INT"`
	MerchantId      int       `xorm:"not null comment('商户ID') index INT"`
	StorageType     string    `xorm:"not null comment('存储类型') VARCHAR(20)"`
	Name            string    `xorm:"not null comment('配置名称') VARCHAR(64)"`
	Endpoint        string    `xorm:"default '' comment('服务端点') VARCHAR(255)"`
	Bucket          string    `xorm:"not null comment('Bucket名称') VARCHAR(128)"`
	Region          string    `xorm:"default '' comment('区域') VARCHAR(64)"`
	AccessKeyId     string    `xorm:"not null comment('AccessKeyId') VARCHAR(255)"`
	AccessKeySecret string    `xorm:"not null comment('AccessKeySecret') VARCHAR(255)"`
	UploadUrl       string    `xorm:"default '' comment('上传URL') VARCHAR(255)"`
	DownloadUrl     string    `xorm:"default '' comment('下载URL') VARCHAR(255)"`
	FileBaseUrl     string    `xorm:"default '' comment('文件基础URL') VARCHAR(255)"`
	BucketUrl       string    `xorm:"default '' comment('Bucket URL') VARCHAR(255)"`
	CustomDomain    string    `xorm:"default '' comment('自定义域名') VARCHAR(255)"`
	IsDefault       int       `xorm:"not null default 0 comment('是否默认') TINYINT"`
	Status          int       `xorm:"not null default 1 comment('状态') TINYINT"`
	LastPushAt      time.Time `xorm:"comment('最后推送时间') DATETIME"`
	LastPushResult  string    `xorm:"default '' comment('推送结果') VARCHAR(255)"`
	CreatedAt       time.Time `xorm:"default CURRENT_TIMESTAMP DATETIME"`
	UpdatedAt       time.Time `xorm:"default CURRENT_TIMESTAMP DATETIME"`
}

// 存储类型常量
const (
	StorageTypeMinio      = "minio"
	StorageTypeAliyunOSS  = "aliyunOSS"
	StorageTypeAwsS3      = "aws_s3"
	StorageTypeTencentCOS = "tencent_cos"
)

// 审计日志操作类型 - 存储配置
const (
	AuditActionCreateStorageConfig = "create_storage_config"
	AuditActionUpdateStorageConfig = "update_storage_config"
	AuditActionDeleteStorageConfig = "delete_storage_config"
	AuditActionPushStorageConfig   = "push_storage_config"
)

// 审计日志目标类型 - 存储配置
const (
	AuditTargetStorageConfig = "storage_config"
)
