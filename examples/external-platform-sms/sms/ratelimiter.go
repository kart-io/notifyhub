// Rate limiter for SMS platform
package sms

import (
	"sync"
	"time"
)

// RateLimiter implements SMS rate limiting
type RateLimiter struct {
	config    RateLimitConfig
	counters  map[string]*Counter
	mu        sync.RWMutex
	cleanupCh chan struct{}
}

// Counter tracks sending count for a phone number
type Counter struct {
	hourlyCount int
	dailyCount  int
	hourlyReset time.Time
	dailyReset  time.Time
	lastAccess  time.Time
	mu          sync.Mutex
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(config RateLimitConfig) *RateLimiter {
	limiter := &RateLimiter{
		config:    config,
		counters:  make(map[string]*Counter),
		cleanupCh: make(chan struct{}),
	}

	// 启动清理协程
	go limiter.cleanup()

	return limiter
}

// Allow checks if the phone number is allowed to send SMS
func (rl *RateLimiter) Allow(phone string) bool {
	if !rl.config.Enabled {
		return true
	}

	rl.mu.Lock()
	counter, exists := rl.counters[phone]
	if !exists {
		counter = &Counter{
			hourlyReset: time.Now().Add(time.Hour),
			dailyReset:  time.Now().Add(24 * time.Hour),
			lastAccess:  time.Now(),
		}
		rl.counters[phone] = counter
	}
	rl.mu.Unlock()

	counter.mu.Lock()
	defer counter.mu.Unlock()

	now := time.Now()
	counter.lastAccess = now

	// 重置计数器
	if now.After(counter.hourlyReset) {
		counter.hourlyCount = 0
		counter.hourlyReset = now.Add(time.Hour)
	}

	if now.After(counter.dailyReset) {
		counter.dailyCount = 0
		counter.dailyReset = now.Add(24 * time.Hour)
	}

	// 检查限制
	if rl.config.MaxPerHour > 0 && counter.hourlyCount >= rl.config.MaxPerHour {
		return false
	}

	if rl.config.MaxPerDay > 0 && counter.dailyCount >= rl.config.MaxPerDay {
		return false
	}

	// 增加计数
	counter.hourlyCount++
	counter.dailyCount++

	return true
}

// GetStats returns rate limiting statistics for a phone number
func (rl *RateLimiter) GetStats(phone string) *RateLimitStats {
	rl.mu.RLock()
	counter, exists := rl.counters[phone]
	rl.mu.RUnlock()

	if !exists {
		return &RateLimitStats{
			Phone:           phone,
			HourlyRemaining: rl.config.MaxPerHour,
			DailyRemaining:  rl.config.MaxPerDay,
		}
	}

	counter.mu.Lock()
	defer counter.mu.Unlock()

	// 计算剩余次数
	hourlyRemaining := rl.config.MaxPerHour - counter.hourlyCount
	if hourlyRemaining < 0 {
		hourlyRemaining = 0
	}

	dailyRemaining := rl.config.MaxPerDay - counter.dailyCount
	if dailyRemaining < 0 {
		dailyRemaining = 0
	}

	return &RateLimitStats{
		Phone:           phone,
		HourlyCount:     counter.hourlyCount,
		DailyCount:      counter.dailyCount,
		HourlyRemaining: hourlyRemaining,
		DailyRemaining:  dailyRemaining,
		HourlyReset:     counter.hourlyReset,
		DailyReset:      counter.dailyReset,
	}
}

// GetAllStats returns statistics for all phone numbers
func (rl *RateLimiter) GetAllStats() map[string]*RateLimitStats {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	stats := make(map[string]*RateLimitStats)
	for phone := range rl.counters {
		stats[phone] = rl.GetStats(phone)
	}

	return stats
}

// Reset resets counters for a phone number
func (rl *RateLimiter) Reset(phone string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	if counter, exists := rl.counters[phone]; exists {
		counter.mu.Lock()
		counter.hourlyCount = 0
		counter.dailyCount = 0
		counter.hourlyReset = time.Now().Add(time.Hour)
		counter.dailyReset = time.Now().Add(24 * time.Hour)
		counter.mu.Unlock()
	}
}

// Close stops the rate limiter
func (rl *RateLimiter) Close() {
	close(rl.cleanupCh)
}

// cleanup removes old counters
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rl.cleanupOldCounters()
		case <-rl.cleanupCh:
			return
		}
	}
}

// cleanupOldCounters removes counters that haven't been accessed recently
func (rl *RateLimiter) cleanupOldCounters() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	cutoff := time.Now().Add(-24 * time.Hour)

	for phone, counter := range rl.counters {
		counter.mu.Lock()
		if counter.lastAccess.Before(cutoff) {
			delete(rl.counters, phone)
		}
		counter.mu.Unlock()
	}
}

// RateLimitStats represents rate limiting statistics
type RateLimitStats struct {
	Phone           string    `json:"phone"`
	HourlyCount     int       `json:"hourly_count"`
	DailyCount      int       `json:"daily_count"`
	HourlyRemaining int       `json:"hourly_remaining"`
	DailyRemaining  int       `json:"daily_remaining"`
	HourlyReset     time.Time `json:"hourly_reset"`
	DailyReset      time.Time `json:"daily_reset"`
}
