package memory

import (
	"context"
	"sync"
	"time"
)

// LoginLimiter is the in-process counterpart of redis.LoginLimiter, for
// DATA_SOURCE=memory and tests: a fixed window of maxAttempts per period per
// key. Counts are lost on restart and not shared between instances.
type LoginLimiter struct {
	mu          sync.Mutex
	maxAttempts int
	period      time.Duration
	now         func() time.Time
	windows     map[string]*attemptWindow
}

type attemptWindow struct {
	count   int
	resetAt time.Time
}

func NewLoginLimiter(maxAttempts int, period time.Duration) *LoginLimiter {
	return &LoginLimiter{
		maxAttempts: maxAttempts,
		period:      period,
		now:         time.Now,
		windows:     map[string]*attemptWindow{},
	}
}

func (l *LoginLimiter) Allow(ctx context.Context, key string) (bool, time.Duration, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	for k, w := range l.windows { // drop finished windows so the map can't grow forever
		if !now.Before(w.resetAt) {
			delete(l.windows, k)
		}
	}

	w, ok := l.windows[key]
	if !ok {
		w = &attemptWindow{resetAt: now.Add(l.period)}
		l.windows[key] = w
	}
	if w.count >= l.maxAttempts {
		return false, w.resetAt.Sub(now), nil
	}
	w.count++
	return true, 0, nil
}

func (l *LoginLimiter) Reset(ctx context.Context, key string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.windows, key)
	return nil
}
