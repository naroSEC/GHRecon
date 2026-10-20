package scheduler

import (
	"context"
	"errors"

	gh "github.com/naroSEC/GHRecon/internal/github"
	"github.com/naroSEC/GHRecon/internal/query"
)

type Result struct {
	Query query.Query
	Value any
	Err   error
}

type Executor func(context.Context, query.Query) (any, error)
type RateSnapshot func() gh.RateState

// Run uses a controller instead of a permanently saturated worker pool, so the
// active request count can shrink immediately after throttling or API failures.
func Run(ctx context.Context, jobs []query.Query, maxConcurrency int, rateSnapshot RateSnapshot, execute Executor) []Result {
	if maxConcurrency < 1 {
		maxConcurrency = 1
	}
	results := make([]Result, 0, len(jobs))
	completed := make(chan Result, maxConcurrency)
	next, inflight, limit := 0, 0, maxConcurrency
	for next < len(jobs) || inflight > 0 {
		for next < len(jobs) && inflight < limit {
			job := jobs[next]
			next++
			inflight++
			go func() {
				value, err := execute(ctx, job)
				completed <- Result{Query: job, Value: value, Err: err}
			}()
		}
		if inflight == 0 {
			break
		}
		select {
		case <-ctx.Done():
			for inflight > 0 {
				result := <-completed
				results = append(results, result)
				inflight--
			}
			return results
		case result := <-completed:
			inflight--
			results = append(results, result)
			var apiErr *gh.APIError
			if errors.As(result.Err, &apiErr) && (apiErr.Retryable || apiErr.Kind == gh.ErrRateLimited || apiErr.Kind == gh.ErrSecondaryLimit) {
				if limit > 1 {
					limit = (limit + 1) / 2
				}
			} else if result.Err == nil {
				rate := gh.RateState{}
				if rateSnapshot != nil {
					rate = rateSnapshot()
				}
				switch {
				case rate.Limit > 0 && rate.Remaining*10 <= rate.Limit:
					limit = 1
				case rate.Limit > 0 && rate.Remaining*4 <= rate.Limit && limit > 2:
					limit = 2
				case limit < maxConcurrency:
					limit++
				}
			}
		}
	}
	return results
}
