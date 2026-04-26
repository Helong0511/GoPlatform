package middleware

import (
	"fmt"
	"sync"
	"time"

	"ad-platform/pkg/response"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// RateLimiter 限流器
type RateLimiter struct {
	visitors map[string]*visitor
	mu       sync.Mutex
	rate     rate.Limit
	burst    int
}

type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// NewRateLimiter 创建限流器
func NewRateLimiter(r rate.Limit, b int) *RateLimiter {
	rl := &RateLimiter{
		visitors: make(map[string]*visitor),
		rate:     r,
		burst:    b,
	}

	// 定期清理过期的访问者
	go rl.cleanupVisitors()

	return rl
}

// getLimiter 获取访问者的限流器
func (rl *RateLimiter) getLimiter(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	v, exists := rl.visitors[ip]
	if !exists {
		limiter := rate.NewLimiter(rl.rate, rl.burst)
		rl.visitors[ip] = &visitor{limiter, time.Now()}
		return limiter
	}

	v.lastSeen = time.Now()
	return v.limiter
}

// cleanupVisitors 清理过期的访问者
func (rl *RateLimiter) cleanupVisitors() {
	for {
		time.Sleep(time.Minute)
		rl.mu.Lock()
		for ip, v := range rl.visitors {
			if time.Since(v.lastSeen) > 3*time.Minute {
				delete(rl.visitors, ip)
			}
		}
		rl.mu.Unlock()
	}
}

// RateLimitMiddleware 限流中间件
func (rl *RateLimiter) RateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		limiter := rl.getLimiter(ip)

		if !limiter.Allow() {
			response.SetErrorReason(c, response.ReasonRateLimited)
			response.TooManyRequests(c, response.MsgTooManyRequestsRetryLater)
			c.Abort()
			return
		}

		c.Next()
	}
}

// CircuitBreaker 熔断器
type CircuitBreaker struct {
	failureThreshold int
	successThreshold int
	timeout          time.Duration
	state            string
	failures         int
	successes        int
	lastFailure      time.Time
	mu               sync.Mutex
}

const (
	StateClosed   = "closed"
	StateOpen     = "open"
	StateHalfOpen = "half-open"
)

// NewCircuitBreaker 创建熔断器
func NewCircuitBreaker(failureThreshold, successThreshold int, timeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		failureThreshold: failureThreshold,
		successThreshold: successThreshold,
		timeout:          timeout,
		state:            StateClosed,
	}
}

// Execute 执行函数，带有熔断保护
func (cb *CircuitBreaker) Execute(fn func() error) error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case StateOpen:
		if time.Since(cb.lastFailure) < cb.timeout {
			return fmt.Errorf("circuit breaker is open")
		}
		cb.state = StateHalfOpen
	case StateHalfOpen:
		// 在半开状态下，只允许一个请求通过
		// 这里简化处理，允许请求通过
	}

	err := fn()
	if err != nil {
		cb.onFailure()
		return err
	}
	cb.onSuccess()
	return nil
}

func (cb *CircuitBreaker) onFailure() {
	cb.failures++
	cb.successes = 0
	cb.lastFailure = time.Now()

	if cb.state == StateHalfOpen {
		cb.state = StateOpen
	} else if cb.failures >= cb.failureThreshold {
		cb.state = StateOpen
	}
}

func (cb *CircuitBreaker) onSuccess() {
	if cb.state == StateHalfOpen {
		cb.successes++
		if cb.successes >= cb.successThreshold {
			cb.state = StateClosed
			cb.failures = 0
			cb.successes = 0
		}
	} else if cb.state == StateClosed {
		cb.failures = 0
	}
}

// CircuitBreakerMiddleware 熔断器中间件
func CircuitBreakerMiddleware(cb *CircuitBreaker) gin.HandlerFunc {
	return func(c *gin.Context) {
		err := cb.Execute(func() error {
			c.Next()
			return nil
		})

		if err != nil {
			response.SetErrorReason(c, response.ReasonCircuitOpen)
			response.Error(c, response.CodeServiceUnavailable, response.MsgServiceUnavailableRetry)
			c.Abort()
		}
	}
}
