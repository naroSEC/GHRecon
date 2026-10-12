package main

import (
	"bufio"
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestRunJSONLEndToEndWithMockGitHub(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-RateLimit-Limit", "30")
		w.Header().Set("X-RateLimit-Remaining", "29")
		switch r.URL.Path {
		case "/search/repositories":
			fmt.Fprint(w, `{"total_count":1,"items":[{"full_name":"acme/app","owner":{"login":"acme"},"html_url":"https://github.test/acme/app","clone_url":"https://github.test/acme/app.git","visibility":"public","updated_at":"2026-01-01T00:00:00Z"}]}`)
		case "/search/code":
			fmt.Fprint(w, `{"total_count":1,"items":[{"name":"config.js","path":"config.js","html_url":"https://github.test/acme/app/blob/main/config.js","repository":{"full_name":"acme/app"},"text_matches":[{"fragment":"https://api.example.com/v1"}]}]}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	configPath := filepath.Join(t.TempDir(), "config.yaml")
	configBody := []byte("github:\n  api_url: " + server.URL + "\nsearch:\n  profile: quick\n  max_queries: 2\n  max_results: 1\n  concurrency: 1\n")
	if err := os.WriteFile(configPath, configBody, 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := run([]string{"--config", configPath, "--domain", "example.com", "--jsonl"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit=%d stderr=%s", code, stderr.String())
	}
	scanner := bufio.NewScanner(&stdout)
	lines := 0
	for scanner.Scan() {
		lines++
	}
	if lines < 3 {
		t.Fatalf("expected repository/domain/url findings, got %d lines: %s", lines, stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("machine output emitted diagnostics: %s", stderr.String())
	}
}
