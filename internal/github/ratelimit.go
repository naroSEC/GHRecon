package github

import (
	"context"
	"strconv"
	"sync"
	"time"
)

type RateState struct {
	Limit     int
	Remaining int
	Used      int
	Reset     time.Time
	Resource  string
}

type RateManager struct {
	mu    sync.RWMutex
	state RateState
	now   func() time.Time
}

func NewRateManager() *RateManager { return &RateManager{now: time.Now} }

func (m *RateManager) Update(header map[string][]string) RateState {
	m.mu.Lock()
	defer m.mu.Unlock()
	parse := func(name string, old int) int {
		v := first(header[name])
		if v == "" {
			return old
		}
		n, err := strconv.Atoi(v)
		if err != nil {
			return old
		}
		return n
	}
	m.state.Limit = parse("X-Ratelimit-Limit", m.state.Limit)
	m.state.Remaining = parse("X-Ratelimit-Remaining", m.state.Remaining)
	m.state.Used = parse("X-Ratelimit-Used", m.state.Used)
	if v := first(header["X-Ratelimit-Reset"]); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			m.state.Reset = time.Unix(n, 0)
		}
	}
	if v := first(header["X-Ratelimit-Resource"]); v != "" {
		m.state.Resource = v
	}
	return m.state
}

func (m *RateManager) Snapshot() RateState { m.mu.RLock(); defer m.mu.RUnlock(); return m.state }

func (m *RateManager) Wait(ctx context.Context) error {
	s := m.Snapshot()
	if s.Remaining != 0 || s.Reset.IsZero() || !s.Reset.After(m.now()) {
		return nil
	}
	d := s.Reset.Sub(m.now()) + 250*time.Millisecond
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func first(v []string) string {
	if len(v) == 0 {
		return ""
	}
	return v[0]
}
