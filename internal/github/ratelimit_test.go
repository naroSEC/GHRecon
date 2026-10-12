package github

import (
	"net/http"
	"strconv"
	"testing"
	"time"
)

func TestRateHeaders(t *testing.T) {
	m := NewRateManager()
	reset := time.Now().Add(time.Minute).Unix()
	got := m.Update(http.Header{
		"X-Ratelimit-Limit":     []string{"30"},
		"X-Ratelimit-Remaining": []string{"12"},
		"X-Ratelimit-Used":      []string{"18"},
		"X-Ratelimit-Reset":     []string{strconv.FormatInt(reset, 10)},
		"X-Ratelimit-Resource":  []string{"search"},
	})
	if got.Limit != 30 || got.Remaining != 12 || got.Used != 18 || got.Resource != "search" {
		t.Fatalf("unexpected state: %#v", got)
	}
}

func TestClassifySecondaryLimit(t *testing.T) {
	err := classify(http.StatusForbidden, []byte(`{"message":"You have exceeded a secondary rate limit"}`), RateState{Remaining: 10})
	if err.Kind != ErrSecondaryLimit || !err.Retryable {
		t.Fatalf("unexpected classification: %#v", err)
	}
}

func TestOrdinaryForbiddenIsNotRateLimit(t *testing.T) {
	err := classify(http.StatusForbidden, []byte(`{"message":"Resource not accessible"}`), RateState{})
	if err.Kind != ErrAPI || err.Retryable {
		t.Fatalf("unexpected classification: %#v", err)
	}
}
