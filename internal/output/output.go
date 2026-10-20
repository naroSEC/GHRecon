package output

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/naroSEC/GHRecon/internal/model"
)

type Format string

type Options struct {
	ShowSources bool
}

const (
	Text  Format = "text"
	JSON  Format = "json"
	JSONL Format = "jsonl"
)

func Write(w io.Writer, format Format, report model.Report) error {
	return WriteWithOptions(w, format, report, Options{})
}

func WriteWithOptions(w io.Writer, format Format, report model.Report, options Options) error {
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
		return writeText(w, report, options)
	default:
		return fmt.Errorf("unsupported output format %q", format)
	}
}

func writeText(w io.Writer, report model.Report, options Options) error {
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
		if options.ShowSources {
			writeSources(w, f.Sources)
		}
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

func writeSources(w io.Writer, sources []model.Provenance) {
	for i, source := range sources {
		fmt.Fprintf(w, "    source %d\n", i+1)
		writeSourceField(w, "type", source.Source)
		writeSourceField(w, "repository", source.Repository)
		writeSourceField(w, "path", source.Path)
		writeSourceField(w, "url", source.URL)
		writeSourceField(w, "query", source.Query)
	}
}

func writeSourceField(w io.Writer, label, value string) {
	if value == "" {
		return
	}
	value = strings.NewReplacer("\r", " ", "\n", " ").Replace(value)
	fmt.Fprintf(w, "      %-10s %s\n", label, value)
}
