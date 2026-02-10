package dbs

import (
	"context"

	"github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/logx"
)

const (
	ErrNil = redis.Nil
)

var (
	rds, streamRds *redis.Client
)

type RedisCfg struct {
	Addr     string
	UserName string
	Password string
	DB       int
	PoolSize int
}

func Rds() *redis.Client {
	return rds
}

func StreamRds() *redis.Client {
	return streamRds
}

func InitRedis(conCfg *RedisCfg) {
	logx.Infof("redis config: %+v", conCfg)

	rds = redis.NewClient(&redis.Options{
		Addr:        conCfg.Addr,
		Password:    conCfg.Password,
		DB:          conCfg.DB,
		PoolSize:    conCfg.PoolSize,
		ReadTimeout: -1,
	})

	if _, err := rds.Ping(context.Background()).Result(); err != nil {
		panic(err)
	}

	// 简单的使用db 1
	streamRds = redis.NewClient(&redis.Options{
		Addr:        conCfg.Addr,
		Password:    conCfg.Password,
		DB:          1,
		PoolSize:    2,
		ReadTimeout: -1,
	})
	if _, err := streamRds.Ping(context.Background()).Result(); err != nil {
		panic(err)
	}
}
