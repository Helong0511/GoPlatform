package testutil

import (
	"context"
	"testing"

	"ad-platform/pkg/db"
	"ad-platform/pkg/redis"
	"ad-platform/pkg/util"
)

// TestHelper 测试帮助器
type TestHelper struct {
	T            *testing.T
	Config       *util.Config
	MockDB       *db.MockDB
	MockRedis    *redis.RedisClient
	CleanupFuncs []func() error
}

// NewTestHelper 创建测试帮助器
func NewTestHelper(t *testing.T) *TestHelper {
	return &TestHelper{
		T:            t,
		Config:       &util.Config{},
		MockDB:       db.NewMockDB(),
		MockRedis:    redis.NewMockRedisClient(),
		CleanupFuncs: make([]func() error, 0),
	}
}

// Setup 设置测试环境
func (th *TestHelper) Setup(fn func() error) {
	if err := fn(); err != nil {
		th.T.Fatalf("Setup failed: %v", err)
	}
}

// Cleanup 清理测试环境
func (th *TestHelper) Cleanup() {
	for _, fn := range th.CleanupFuncs {
		if err := fn(); err != nil {
			th.T.Logf("Cleanup error: %v", err)
		}
	}

	// 关闭模拟连接
	if err := th.MockDB.Close(); err != nil {
		th.T.Logf("Close MockDB error: %v", err)
	}

	if err := th.MockRedis.Close(); err != nil {
		th.T.Logf("Close MockRedis error: %v", err)
	}
}

// RegisterCleanup 注册清理函数
func (th *TestHelper) RegisterCleanup(fn func() error) {
	th.CleanupFuncs = append(th.CleanupFuncs, fn)
}

// Assert 断言工具
type Assert struct {
	t *testing.T
}

// NewAssert 创建断言工具
func NewAssert(t *testing.T) *Assert {
	return &Assert{t: t}
}

// Equal 断言相等
func (a *Assert) Equal(expected, actual interface{}) {
	if expected != actual {
		a.t.Errorf("Expected %v, got %v", expected, actual)
	}
}

// NotEqual 断言不相等
func (a *Assert) NotEqual(expected, actual interface{}) {
	if expected == actual {
		a.t.Errorf("Expected not equal to %v, got %v", expected, actual)
	}
}

// NoError 断言无错误
func (a *Assert) NoError(err error) {
	if err != nil {
		a.t.Errorf("Expected no error, got %v", err)
	}
}

// Error 断言有错误
func (a *Assert) Error(err error) {
	if err == nil {
		a.t.Error("Expected error, got nil")
	}
}

// Nil 断言为nil
func (a *Assert) Nil(val interface{}) {
	if val != nil {
		a.t.Errorf("Expected nil, got %v", val)
	}
}

// NotNil 断言不为nil
func (a *Assert) NotNil(val interface{}) {
	if val == nil {
		a.t.Error("Expected not nil, got nil")
	}
}

// True 断言为true
func (a *Assert) True(val bool) {
	if !val {
		a.t.Error("Expected true, got false")
	}
}

// False 断言为false
func (a *Assert) False(val bool) {
	if val {
		a.t.Error("Expected false, got true")
	}
}

// TestContext 创建测试用的context
func TestContext(t *testing.T) context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	// 注意: 实际使用中应该在测试结束后调用cancel()
	_ = cancel
	return ctx
}
