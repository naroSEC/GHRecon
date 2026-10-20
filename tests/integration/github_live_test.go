//go:build integration

package integration

import (
	"context"
	"os"
	"testing"
	"time"

	gh "github.com/naroSEC/GHRecon/internal/github"
)

func TestGitHubRepositorySearchLive(t *testing.T) {
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		t.Skip("GITHUB_TOKEN is required for live integration tests")
	}
	client := gh.NewClient("https://api.github.com", "2022-11-28", 20*time.Second, gh.TokenAuth{Token: token})
	got, err := client.SearchRepositories(context.Background(), "Hello-World in:name user:octocat", 5, true, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) == 0 {
		t.Fatal("expected at least one public repository")
	}
}
