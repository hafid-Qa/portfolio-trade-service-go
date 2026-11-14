package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis_rate/v10"
	goredis "github.com/redis/go-redis/v9"
)

const loginAttemptsKeyPrefix = "login_attempts:"

// LoginLimiter caps login attempts per username with redis_rate (GCRA), so
// the count is shared by every API instance and survives restarts.
type LoginLimiter struct {
	limiter *redis_rate.Limiter
	limit   redis_rate.Limit
}

// NewLoginLimiter allows maxAttempts per period for each key.
func NewLoginLimiter(client *goredis.Client, maxAttempts int, period time.Duration) *LoginLimiter {
	return &LoginLimiter{
		limiter: redis_rate.NewLimiter(client),
		limit:   redis_rate.Limit{Rate: maxAttempts, Burst: maxAttempts, Period: period},
	}
}

// Allow records one attempt for key and reports whether it may proceed; if
// not, retryAfter is how long until the next attempt is allowed.
func (l *LoginLimiter) Allow(ctx context.Context, key string) (bool, time.Duration, error) {
	res, err := l.limiter.Allow(ctx, loginAttemptsKeyPrefix+key, l.limit)
	if err != nil {
		return false, 0, fmt.Errorf("checking login rate limit: %w", err)
	}
	if res.Allowed == 0 {
		return false, res.RetryAfter, nil
	}
	return true, 0, nil
}

// Reset clears key's attempts (called after a successful login).
func (l *LoginLimiter) Reset(ctx context.Context, key string) error {
	if err := l.limiter.Reset(ctx, loginAttemptsKeyPrefix+key); err != nil {
		return fmt.Errorf("resetting login rate limit: %w", err)
	}
	return nil
}
