package output

import (
	"bufio"
	"bytes"
	"encoding/json"
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
