package cfg

import (
	"server/pkg/dbs"

	"github.com/zeromicro/go-zero/core/service"
)

var Release string

// MerchantAPICfg 商户API认证配置
type MerchantAPICfg struct {
	Username string
	Password string
}

// IpEmbedTarget 单个上传目标配置
type IpEmbedTarget struct {
	Name           string // 目标名称
	CloudType      string // 云类型: aws, aliyun, tencent
	CloudAccountId int64  // 云账号ID
	RegionId       string // 区域ID
	Bucket         string // Bucket名称
	ObjectPrefix   string // 对象前缀
	Enabled        bool   // 是否启用
}

// IpEmbedConfig IP隐写上传配置
type IpEmbedConfig struct {
	SourceDir string          // 源文件目录
	Targets   []IpEmbedTarget // 上传目标列表
}

var C struct {
	service.ServiceConf
	ListenOn    string
	Mysql       *dbs.MysqlCfg
	Redis       *dbs.RedisCfg
	MerchantAPI *MerchantAPICfg // 商户API认证配置
	IpEmbed     *IpEmbedConfig  // IP隐写上传配置
}
