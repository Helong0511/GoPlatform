package redis

import (
	"context"
)

// NewMockRedisClient 创建一个降级的 Redis 客户端（所有操作都不执行）
// 当 Redis 连接失败时使用，避免空指针错误
func NewMockRedisClient() *RedisClient {
	return &RedisClient{
		Client: nil,
		Ctx:    context.Background(),
	}
}
