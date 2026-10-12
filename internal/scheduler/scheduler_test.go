package scheduler

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	gh "github.com/ghrecon/ghrecon/internal/github"
	"github.com/ghrecon/ghrecon/internal/query"
)

func TestRunBoundsConcurrency(t *testing.T) {
	jobs := make([]query.Query, 12)
	var active atomic.Int32
	var maximum atomic.Int32
	results := Run(context.Background(), jobs, 3, func() gh.RateState {
		return gh.RateState{Limit: 30, Remaining: 30}
	}, func(context.Context, query.Query) (any, error) {
		now := active.Add(1)
		for {
			old := maximum.Load()
			if now <= old || maximum.CompareAndSwap(old, now) {
				break
			}
		}
		time.Sleep(5 * time.Millisecond)
		active.Add(-1)
		return nil, nil
	})
	if len(results) != len(jobs) {
		t.Fatalf("got %d results", len(results))
	}
	if maximum.Load() > 3 {
		t.Fatalf("concurrency exceeded: %d", maximum.Load())
	}
}
