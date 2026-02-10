package cache

import (
	"context"
	"encoding/json"
	"server/pkg/dbs"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

const (
	// 默认缓存过期时间
	DefaultExpiration = 5 * time.Minute
	// 短期缓存
	ShortExpiration = 1 * time.Minute
	// 长期缓存
	LongExpiration = 30 * time.Minute
)

// Cache 缓存操作接口
type Cache struct {
	prefix string
}

// New 创建缓存实例
func New(prefix string) *Cache {
	return &Cache{prefix: prefix}
}

// key 生成完整的缓存 key
func (c *Cache) key(k string) string {
	return "cache:" + c.prefix + ":" + k
}

// Get 获取缓存
func (c *Cache) Get(ctx context.Context, key string, dest interface{}) bool {
	data, err := dbs.Rds().Get(ctx, c.key(key)).Bytes()
	if err != nil {
		return false
	}

	if err := json.Unmarshal(data, dest); err != nil {
		logx.Errorf("cache unmarshal error: %v", err)
		return false
	}

	return true
}

// Set 设置缓存
func (c *Cache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	return dbs.Rds().Set(ctx, c.key(key), data, expiration).Err()
}

// Delete 删除缓存
func (c *Cache) Delete(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}

	fullKeys := make([]string, len(keys))
	for i, k := range keys {
		fullKeys[i] = c.key(k)
	}

	return dbs.Rds().Del(ctx, fullKeys...).Err()
}

// DeleteByPattern 按模式删除缓存
func (c *Cache) DeleteByPattern(ctx context.Context, pattern string) error {
	fullPattern := c.key(pattern)
	iter := dbs.Rds().Scan(ctx, 0, fullPattern, 100).Iterator()

	var keys []string
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}

	if err := iter.Err(); err != nil {
		return err
	}

	if len(keys) > 0 {
		return dbs.Rds().Del(ctx, keys...).Err()
	}

	return nil
}

// GetOrSet 获取缓存，如果不存在则调用 loader 加载并缓存
func (c *Cache) GetOrSet(ctx context.Context, key string, dest interface{}, expiration time.Duration, loader func() (interface{}, error)) error {
	// 尝试从缓存获取
	if c.Get(ctx, key, dest) {
		return nil
	}

	// 缓存未命中，调用 loader
	value, err := loader()
	if err != nil {
		return err
	}

	// 将结果存入缓存
	if err := c.Set(ctx, key, value, expiration); err != nil {
		logx.Errorf("cache set error: %v", err)
	}

	// 将值复制到 dest
	data, _ := json.Marshal(value)
	return json.Unmarshal(data, dest)
}

// 预定义的缓存实例
var (
	// MerchantCache 商户相关缓存
	MerchantCache = New("merchant")
	// ServerCache 服务器相关缓存
	ServerCache = New("server")
	// ConfigCache 配置相关缓存
	ConfigCache = New("config")
)

// InvalidateMerchant 清除商户相关缓存
func InvalidateMerchant(ctx context.Context, merchantId int) {
	_ = MerchantCache.DeleteByPattern(ctx, "*")
}

// InvalidateServer 清除服务器相关缓存
func InvalidateServer(ctx context.Context, serverId int) {
	_ = ServerCache.DeleteByPattern(ctx, "*")
}
