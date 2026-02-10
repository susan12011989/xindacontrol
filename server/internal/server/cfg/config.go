package cfg

import (
	"os"
	"server/pkg/dbs"
	"strconv"

	"github.com/zeromicro/go-zero/core/logx"
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

// ApplyEnvOverrides 从环境变量覆盖配置
// 支持的环境变量:
//   - DB_HOST: MySQL 地址 (如 127.0.0.1:3306)
//   - DB_USER: MySQL 用户名
//   - DB_PASSWORD: MySQL 密码
//   - DB_NAME: MySQL 数据库名
//   - REDIS_HOST: Redis 地址 (如 127.0.0.1:6379)
//   - REDIS_PASSWORD: Redis 密码
//   - REDIS_DB: Redis 数据库编号
//   - LISTEN_ADDR: 服务监听地址 (如 0.0.0.0:58181)
//   - MERCHANT_API_USER: 商户API用户名
//   - MERCHANT_API_PASSWORD: 商户API密码
//   - ASSETS_DIR: 静态资源目录
//   - VERSIONS_DIR: 版本文件存储目录
//   - LOG_PATH: 日志文件目录
func ApplyEnvOverrides() {
	// MySQL 配置覆盖
	if C.Mysql != nil {
		if v := os.Getenv("DB_HOST"); v != "" {
			logx.Infof("Overriding MySQL addr from env: %s", v)
			C.Mysql.Addr = v
		}
		if v := os.Getenv("DB_USER"); v != "" {
			C.Mysql.UserName = v
		}
		if v := os.Getenv("DB_PASSWORD"); v != "" {
			C.Mysql.Password = v
		}
		if v := os.Getenv("DB_NAME"); v != "" {
			C.Mysql.DB = v
		}
	}

	// Redis 配置覆盖
	if C.Redis != nil {
		if v := os.Getenv("REDIS_HOST"); v != "" {
			logx.Infof("Overriding Redis addr from env: %s", v)
			C.Redis.Addr = v
		}
		if v := os.Getenv("REDIS_PASSWORD"); v != "" {
			C.Redis.Password = v
		}
		if v := os.Getenv("REDIS_DB"); v != "" {
			if db, err := strconv.Atoi(v); err == nil {
				C.Redis.DB = db
			}
		}
	}

	// 监听地址覆盖
	if v := os.Getenv("LISTEN_ADDR"); v != "" {
		logx.Infof("Overriding listen addr from env: %s", v)
		C.ListenOn = v
	}

	// 商户API配置覆盖
	if C.MerchantAPI != nil {
		if v := os.Getenv("MERCHANT_API_USER"); v != "" {
			C.MerchantAPI.Username = v
		}
		if v := os.Getenv("MERCHANT_API_PASSWORD"); v != "" {
			C.MerchantAPI.Password = v
		}
	}
}
