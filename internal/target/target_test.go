package target

import (
	"testing"

	"github.com/ghrecon/ghrecon/internal/model"
)

func TestNormalize(t *testing.T) {
	got, err := Normalize(model.Target{Domains: []string{"HTTPS://API.Example.COM/path", "*.example.com", "example.com"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Domains) != 2 || got.Domains[0] != "api.example.com" || got.Domains[1] != "example.com" {
		t.Fatalf("unexpected domains: %#v", got.Domains)
	}
}

func TestNormalizeRejectsInvalid(t *testing.T) {
	if _, err := Normalize(model.Target{Domains: []string{"not a domain"}}); err == nil {
		t.Fatal("expected invalid domain error")
	}
}
