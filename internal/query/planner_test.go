package query

import (
	"testing"

	"github.com/ghrecon/ghrecon/internal/config"
	"github.com/ghrecon/ghrecon/internal/model"
)

func TestPlanDeterministicAndBounded(t *testing.T) {
	cfg, err := config.Load("")
	if err != nil {
		t.Fatal(err)
	}
	cfg.Search.MaxQueries = 4
	got := Plan(model.Target{Domains: []string{"example.com"}, Companies: []string{"Example Corp"}}, cfg)
	if len(got) != 4 {
		t.Fatalf("got %d queries", len(got))
	}
	if got[0].Kind != Repository || got[0].Text != `"example.com"` {
		t.Fatalf("unexpected first query: %#v", got[0])
	}
	if got[2].Text != `"Example Corp"` {
		t.Fatalf("company was not quoted: %#v", got[2])
	}
}
