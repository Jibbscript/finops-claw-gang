package ratelimit

import (
	"fmt"
	"sync"
	"time"
)

// ActivityBudget tracks per-tenant activity call counts within time windows.
type ActivityBudget struct {
	mu     sync.Mutex
	counts map[string]*windowCounter

	maxPerWindow int
	windowSize   time.Duration
	now          func() time.Time
}

type windowCounter struct {
	count     int
	windowEnd time.Time
}

// NewActivityBudget creates a budget limiter.
// maxPerWindow limits calls per (tenantID, activity) within windowSize.
func NewActivityBudget(maxPerWindow int, windowSize time.Duration) *ActivityBudget {
	return &ActivityBudget{
		counts:       make(map[string]*windowCounter),
		maxPerWindow: maxPerWindow,
		windowSize:   windowSize,
		now:          time.Now,
	}
}

func budgetKey(tenantID, activity string) string {
	return tenantID + "|" + activity
}

// Check returns an error if the tenant has exceeded the budget for the activity.
func (b *ActivityBudget) Check(tenantID, activity string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	key := budgetKey(tenantID, activity)
	wc, ok := b.counts[key]
	if !ok || b.now().After(wc.windowEnd) {
		return nil // no window or expired window
	}
	if wc.count >= b.maxPerWindow {
		return fmt.Errorf("activity budget exceeded: tenant %s activity %s (%d/%d in window)",
			tenantID, activity, wc.count, b.maxPerWindow)
	}
	return nil
}

// Record records an activity call for the tenant.
func (b *ActivityBudget) Record(tenantID, activity string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	key := budgetKey(tenantID, activity)
	wc, ok := b.counts[key]
	if !ok || b.now().After(wc.windowEnd) {
		b.counts[key] = &windowCounter{
			count:     1,
			windowEnd: b.now().Add(b.windowSize),
		}
		return
	}
	wc.count++
}
