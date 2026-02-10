package cfg

import (
	"server/pkg/dbs"

	"github.com/zeromicro/go-zero/core/service"
)

var C struct {
	service.ServiceConf

	// 数据库和 Redis（与主服务共享）
	Mysql *dbs.MysqlCfg
	Redis *dbs.RedisCfg

	// Worker 配置
	Worker WorkerConfig

	// 云存储配置（用于上传产物）
	Storage StorageConfig
}

type WorkerConfig struct {
	// 并发配置
	MaxConcurrent int `json:",default=2"` // 最大并发构建数
	PollInterval  int `json:",default=5"` // 队列轮询间隔（秒）

	// 构建脚本路径
	ScriptsDir   string `json:",default=/opt/build/scripts"`
	OutputDir    string `json:",default=/opt/build/outputs"`
	MerchantsDir string `json:",default=/opt/build/merchants"`

	// 超时配置
	BuildTimeout int `json:",default=1800"` // 构建超时（秒），默认30分钟
}

type StorageConfig struct {
	Type           string `json:",default=aliyun"` // aliyun, tencent, aws
	CloudAccountID int64  // 系统云账号ID
	RegionID       string
	Bucket         string
	ObjectPrefix   string `json:",default=builds/"` // 对象前缀
}
