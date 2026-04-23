package redis

import (
	"context"
	"fmt"
	"time"

	"ad-platform/pkg/util"

	"github.com/go-redis/redis/v8"
)

// RedisClient Redis客户端
type RedisClient struct {
	Client *redis.Client
	Ctx    context.Context
}

// RedisClusterClient Redis集群客户端
type RedisClusterClient struct {
	Client *redis.ClusterClient
	Ctx    context.Context
}

// NewRedisClient 创建Redis客户端
func NewRedisClient(config *util.RedisConfig) (*RedisClient, error) {
	// 解析超时配置
	dialTimeout, err := time.ParseDuration(config.DialTimeout)
	if err != nil {
		dialTimeout = 5 * time.Second
	}

	readTimeout, err := time.ParseDuration(config.ReadTimeout)
	if err != nil {
		readTimeout = 3 * time.Second
	}

	writeTimeout, err := time.ParseDuration(config.WriteTimeout)
	if err != nil {
		writeTimeout = 3 * time.Second
	}

	// 解析空闲超时时间
	idleTimeout, err := time.ParseDuration(config.IdleTimeout)
	if err != nil {
		idleTimeout = 5 * time.Minute
	}

	// 解析重试间隔
	minRetryBackoff, err := time.ParseDuration(config.MinRetryBackoff)
	if err != nil {
		minRetryBackoff = 100 * time.Millisecond
	}

	maxRetryBackoff, err := time.ParseDuration(config.MaxRetryBackoff)
	if err != nil {
		maxRetryBackoff = 1 * time.Second
	}

	// 创建Redis客户端
	client := redis.NewClient(&redis.Options{
		Addr:            fmt.Sprintf("%s:%d", config.Host, config.Port),
		Password:        config.Password,
		DB:              config.DB,
		PoolSize:        config.PoolSize,
		MinIdleConns:    config.MinIdleConns,
		DialTimeout:     dialTimeout,
		ReadTimeout:     readTimeout,
		WriteTimeout:    writeTimeout,
		IdleTimeout:     idleTimeout,
		MaxRetries:      config.MaxRetries,
		MinRetryBackoff: minRetryBackoff,
		MaxRetryBackoff: maxRetryBackoff,
	})

	// 测试连接
	ctx := context.Background()
	if _, err := client.Ping(ctx).Result(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	return &RedisClient{
		Client: client,
		Ctx:    ctx,
	}, nil
}

// NewRedisClusterClient 创建Redis集群客户端
func NewRedisClusterClient(configs []*util.RedisConfig) (*RedisClusterClient, error) {
	// 构建集群节点地址列表
	addrs := make([]string, 0, len(configs))
	for _, config := range configs {
		addrs = append(addrs, fmt.Sprintf("%s:%d", config.Host, config.Port))
	}

	// 解析超时配置
	dialTimeout, err := time.ParseDuration(configs[0].DialTimeout)
	if err != nil {
		dialTimeout = 5 * time.Second
	}
	readTimeout, err := time.ParseDuration(configs[0].ReadTimeout)
	if err != nil {
		readTimeout = 3 * time.Second
	}
	writeTimeout, err := time.ParseDuration(configs[0].WriteTimeout)
	if err != nil {
		writeTimeout = 3 * time.Second
	}

	// 解析空闲超时时间
	idleTimeout, err := time.ParseDuration(configs[0].IdleTimeout)
	if err != nil {
		idleTimeout = 5 * time.Minute
	}

	// 解析重试间隔
	minRetryBackoff, err := time.ParseDuration(configs[0].MinRetryBackoff)
	if err != nil {
		minRetryBackoff = 100 * time.Millisecond
	}

	maxRetryBackoff, err := time.ParseDuration(configs[0].MaxRetryBackoff)
	if err != nil {
		maxRetryBackoff = 1 * time.Second
	}

	// 创建Redis集群客户端
	client := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs:           addrs,
		Password:        configs[0].Password,
		PoolSize:        configs[0].PoolSize,
		MinIdleConns:    configs[0].MinIdleConns,
		DialTimeout:     dialTimeout,
		ReadTimeout:     readTimeout,
		WriteTimeout:    writeTimeout,
		IdleTimeout:     idleTimeout,
		MaxRetries:      configs[0].MaxRetries,
		MinRetryBackoff: minRetryBackoff,
		MaxRetryBackoff: maxRetryBackoff,
	})

	// 测试连接
	ctx := context.Background()
	if _, err := client.Ping(ctx).Result(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis cluster: %w", err)
	}

	return &RedisClusterClient{
		Client: client,
		Ctx:    ctx,
	}, nil
}

// Set 设置缓存
func (r *RedisClient) Set(key string, value interface{}, expiration time.Duration) error {
	if r.Client == nil {
		return nil
	}
	return r.Client.Set(r.Ctx, key, value, expiration).Err()
}

// Get 获取缓存
func (r *RedisClient) Get(key string) (string, error) {
	if r.Client == nil {
		return "", nil
	}
	return r.Client.Get(r.Ctx, key).Result()
}

// Delete 删除缓存
func (r *RedisClient) Delete(key string) error {
	if r.Client == nil {
		return nil
	}
	return r.Client.Del(r.Ctx, key).Err()
}

// Exists 检查键是否存在
func (r *RedisClient) Exists(key string) (bool, error) {
	if r.Client == nil {
		return false, nil
	}
	result, err := r.Client.Exists(r.Ctx, key).Result()
	if err != nil {
		return false, err
	}
	return result > 0, nil
}

// Set 设置缓存（集群版本）
func (r *RedisClusterClient) Set(key string, value interface{}, expiration time.Duration) error {
	return r.Client.Set(r.Ctx, key, value, expiration).Err()
}

// Get 获取缓存（集群版本）
func (r *RedisClusterClient) Get(key string) (string, error) {
	return r.Client.Get(r.Ctx, key).Result()
}

// Delete 删除缓存（集群版本）
func (r *RedisClusterClient) Delete(key string) error {
	return r.Client.Del(r.Ctx, key).Err()
}

// Exists 检查键是否存在（集群版本）
func (r *RedisClusterClient) Exists(key string) (bool, error) {
	result, err := r.Client.Exists(r.Ctx, key).Result()
	if err != nil {
		return false, err
	}
	return result > 0, nil
}

// Close 关闭Redis连接
func (r *RedisClient) Close() error {
	if r.Client != nil {
		return r.Client.Close()
	}
	return nil
}

// Close 关闭Redis集群连接
func (r *RedisClusterClient) Close() error {
	return r.Client.Close()
}
