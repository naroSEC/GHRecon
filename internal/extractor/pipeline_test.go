package extractor

import (
	"os"
	"strings"
	"testing"

	"github.com/ghrecon/ghrecon/internal/config"
	"github.com/ghrecon/ghrecon/internal/model"
)

func TestPipelineFixture(t *testing.T) {
	cfg, err := config.Load("")
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile("../../tests/fixtures/mixed.txt")
	if err != nil {
		t.Fatal(err)
	}
	p := New(cfg, model.Target{Domains: []string{"example.com"}})
	got := p.Extract(model.SearchDocument{Source: "github_code_search", Repository: "example/app", Path: ".env", Text: string(data)})
	values := map[string]bool{}
	for _, f := range got {
		values[f.Type+":"+f.Provider+":"+f.Value] = true
	}
	for _, want := range []string{
		"subdomain::api.example.com",
		"url::https://api.example.com/v1/users?active=true",
		"endpoint::/admin/debug",
		"cloud_asset:aws:assets.s3.us-east-1.amazonaws.com",
		"cloud_asset:azure:examplestore.blob.core.windows.net",
		"cloud_asset:gcp:example-service.run.app",
		"secret_candidate::example/app:.env",
	} {
		if !values[want] {
			t.Errorf("missing %q in %#v", want, values)
		}
	}
}

func TestURLIsStructured(t *testing.T) {
	cfg, _ := config.Load("")
	got := New(cfg, model.Target{Domains: []string{"example.com"}}).Extract(model.SearchDocument{Text: "see https://api.example.com:8443/v1?q=x"})
	for _, f := range got {
		if f.Type == "url" {
			if f.Attributes["host"] != "api.example.com" || f.Attributes["port"] != "8443" {
				t.Fatalf("bad URL attributes: %#v", f.Attributes)
			}
			return
		}
	}
	t.Fatal("URL not extracted")
}

func TestUnrelatedDomainIsNotTargetDomain(t *testing.T) {
	cfg, _ := config.Load("")
	got := New(cfg, model.Target{Domains: []string{"example.com"}}).Extract(model.SearchDocument{Text: "api.example.com attackerexample.com unrelated.test"})
	for _, f := range got {
		if (f.Type == "domain" || f.Type == "subdomain") && !strings.HasSuffix(f.Value, "example.com") {
			t.Fatalf("unrelated domain: %#v", f)
		}
	}
}
