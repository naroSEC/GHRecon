package correlation

import (
	"testing"

	"github.com/naroSEC/GHRecon/internal/model"
)

func TestDedupPreservesProvenance(t *testing.T) {
	s := New()
	s.Add(model.Finding{Type: "subdomain", Value: "API.EXAMPLE.COM", Confidence: .5, Sources: []model.Provenance{{Source: "github_code_search", Path: "a"}}})
	s.Add(model.Finding{Type: "subdomain", Value: "api.example.com", Confidence: .9, Sources: []model.Provenance{{Source: "github_code_search", Path: "b"}}})
	got := s.Findings()
	if len(got) != 1 {
		t.Fatalf("got %d findings", len(got))
	}
	if got[0].Confidence != .9 || len(got[0].Sources) != 2 {
		t.Fatalf("merge failed: %#v", got[0])
	}
	if got[0].Sources[0].Fragment != "" {
		t.Fatal("fragment should be discarded")
	}
}
