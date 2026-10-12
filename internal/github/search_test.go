package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync/atomic"
	"testing"
	"time"
)

func TestRepositoryPaginationAndAuth(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("missing auth header")
		}
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		count := 100
		if page == 2 {
			count = 1
		}
		items := make([]map[string]any, count)
		for i := range items {
			n := (page-1)*100 + i
			items[i] = map[string]any{
				"full_name":  fmt.Sprintf("owner/repo-%d", n),
				"owner":      map[string]any{"login": "owner"},
				"html_url":   "https://github.test/owner/repo",
				"clone_url":  "https://github.test/owner/repo.git",
				"visibility": "public",
				"updated_at": "2026-01-01T00:00:00Z",
			}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"total_count": 101, "items": items})
	}))
	defer server.Close()
	client := NewClient(server.URL, "2022-11-28", time.Second, TokenAuth{Token: "test-token"})
	got, err := client.SearchRepositories(context.Background(), "example", 101, true, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 101 || calls.Load() != 2 {
		t.Fatalf("findings=%d calls=%d", len(got), calls.Load())
	}
}

func TestMockedCodeSearchIntegration(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/search/code" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("X-RateLimit-Limit", "30")
		w.Header().Set("X-RateLimit-Remaining", "29")
		fmt.Fprint(w, `{"total_count":1,"items":[{"name":"config.js","path":"src/config.js","html_url":"https://github.test/acme/app/blob/main/src/config.js","repository":{"full_name":"acme/app"},"text_matches":[{"fragment":"const api = 'https://api.example.com/v1';"}]}]}`)
	}))
	defer server.Close()
	client := NewClient(server.URL, "2022-11-28", time.Second, NoAuth{})
	got, err := client.SearchCode(context.Background(), `"example.com"`, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Repository != "acme/app" || got[0].Text == "" {
		t.Fatalf("unexpected document: %#v", got)
	}
	if client.Rates.Snapshot().Remaining != 29 {
		t.Fatalf("rate state not updated: %#v", client.Rates.Snapshot())
	}
}

func TestIncompleteSearchPreservesResults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"total_count":1,"incomplete_results":true,"items":[{"name":"config.js","path":"config.js","html_url":"https://github.test/a/b/blob/main/config.js","repository":{"full_name":"a/b"},"text_matches":[{"fragment":"api.example.com"}]}]}`)
	}))
	defer server.Close()
	client := NewClient(server.URL, "2022-11-28", time.Second, NoAuth{})
	got, err := client.SearchCode(context.Background(), `"example.com"`, 10)
	if len(got) != 1 || !IsKind(err, ErrIncomplete) {
		t.Fatalf("results=%#v error=%v", got, err)
	}
}
