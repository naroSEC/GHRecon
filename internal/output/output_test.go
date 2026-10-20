package output

import (
	"bufio"
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/naroSEC/GHRecon/internal/model"
)

func TestJSONSchema(t *testing.T) {
	report := model.Report{
		SchemaVersion: model.SchemaVersion,
		Status:        "completed",
		Findings: []model.Finding{{
			SchemaVersion: model.SchemaVersion,
			Type:          "domain",
			Value:         "example.com",
			Confidence:    .9,
			Sources:       []model.Provenance{},
		}},
		Summary: model.Summary{Counts: map[string]int{"domain": 1}},
	}
	var buf bytes.Buffer
	if err := Write(&buf, JSON, report); err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded["schema_version"] != "1" || decoded["status"] != "completed" {
		t.Fatalf("bad envelope: %s", buf.String())
	}
}

func TestTextShowSources(t *testing.T) {
	report := model.Report{
		Status: "completed",
		Findings: []model.Finding{{
			Type:       "subdomain",
			Value:      "api.example.com",
			Confidence: .9,
			Sources: []model.Provenance{{
				Source:     "github_code_search",
				Repository: "acme/app",
				Path:       "config/app.yml",
				URL:        "https://github.test/acme/app/blob/main/config/app.yml",
				Query:      `"example.com"`,
			}},
		}},
		Summary: model.Summary{Counts: map[string]int{"subdomain": 1}},
	}

	var plain bytes.Buffer
	if err := Write(&plain, Text, report); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(plain.String(), "acme/app") {
		t.Fatalf("default text output unexpectedly showed provenance: %s", plain.String())
	}

	var detailed bytes.Buffer
	if err := WriteWithOptions(&detailed, Text, report, Options{ShowSources: true}); err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"github_code_search", "acme/app", "config/app.yml", "https://github.test/acme/app/blob/main/config/app.yml", `"example.com"`} {
		if !strings.Contains(detailed.String(), expected) {
			t.Fatalf("detailed text output missing %q: %s", expected, detailed.String())
		}
	}
}

func TestJSONLOneFindingPerLine(t *testing.T) {
	report := model.Report{Findings: []model.Finding{
		{SchemaVersion: "1", Type: "domain", Value: "a.test", Sources: []model.Provenance{}},
		{SchemaVersion: "1", Type: "domain", Value: "b.test", Sources: []model.Provenance{}},
	}}
	var buf bytes.Buffer
	if err := Write(&buf, JSONL, report); err != nil {
		t.Fatal(err)
	}
	scanner := bufio.NewScanner(&buf)
	count := 0
	for scanner.Scan() {
		count++
		var value model.Finding
		if err := json.Unmarshal(scanner.Bytes(), &value); err != nil {
			t.Fatal(err)
		}
		if value.SchemaVersion != "1" {
			t.Fatal("missing schema version")
		}
	}
	if count != 2 {
		t.Fatalf("got %d lines", count)
	}
}
