package output

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/ghrecon/ghrecon/internal/model"
)

type Format string

const (
	Text  Format = "text"
	JSON  Format = "json"
	JSONL Format = "jsonl"
)

func Write(w io.Writer, format Format, report model.Report) error {
	switch format {
	case JSON:
		encoder := json.NewEncoder(w)
		encoder.SetEscapeHTML(false)
		encoder.SetIndent("", "  ")
		return encoder.Encode(report)
	case JSONL:
		bw := bufio.NewWriter(w)
		encoder := json.NewEncoder(bw)
		encoder.SetEscapeHTML(false)
		for _, finding := range report.Findings {
			if err := encoder.Encode(finding); err != nil {
				return err
			}
		}
		return bw.Flush()
	case Text:
		return writeText(w, report)
	default:
		return fmt.Errorf("unsupported output format %q", format)
	}
}

func writeText(w io.Writer, report model.Report) error {
	if _, err := fmt.Fprintf(w, "GHRecon %s\n\n", report.Status); err != nil {
		return err
	}
	last := ""
	for _, f := range report.Findings {
		if f.Type != last {
			if last != "" {
				fmt.Fprintln(w)
			}
			fmt.Fprintf(w, "%s\n%s\n", strings.ToUpper(f.Type), strings.Repeat("-", len(f.Type)))
			last = f.Type
		}
		fmt.Fprintf(w, "%-60s %.2f  (%d sources)\n", f.Value, f.Confidence, len(f.Sources))
	}
	keys := make([]string, 0, len(report.Summary.Counts))
	for k := range report.Summary.Counts {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	fmt.Fprintln(w, "\nSummary")
	for _, k := range keys {
		fmt.Fprintf(w, "  %-20s %d\n", k, report.Summary.Counts[k])
	}
	fmt.Fprintf(w, "  %-20s %d\n  %-20s %d\n  %-20s %d ms\n", "queries", report.Summary.Queries, "api_requests", report.Summary.APIRequests, "duration", report.Summary.DurationMS)
	if report.Partial {
		fmt.Fprintf(w, "\nPartial completion: %d queries failed.\n", len(report.Errors))
	}
	return nil
}
