package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"runtime/debug"
	"strings"
	"syscall"
	"time"

	"github.com/naroSEC/GHRecon/internal/config"
	"github.com/naroSEC/GHRecon/internal/correlation"
	"github.com/naroSEC/GHRecon/internal/extractor"
	gh "github.com/naroSEC/GHRecon/internal/github"
	"github.com/naroSEC/GHRecon/internal/model"
	"github.com/naroSEC/GHRecon/internal/output"
	"github.com/naroSEC/GHRecon/internal/query"
	"github.com/naroSEC/GHRecon/internal/scheduler"
	"github.com/naroSEC/GHRecon/internal/target"
)

var version = "dev"

type stringList []string

func (s *stringList) String() string     { return strings.Join(*s, ",") }
func (s *stringList) Set(v string) error { *s = append(*s, v); return nil }

type options struct {
	target                                                                                          model.Target
	input, configPath, outputPath, profile, token, tokenFile                                        string
	json, jsonl, reposOnly, showSources, includeForks, includeArchived, verbose, quiet, showVersion bool
	maxResults, maxQueries, maxDepth, concurrency                                                   int
	timeout                                                                                         time.Duration
}

func main() { os.Exit(run(os.Args[1:], os.Stdout, os.Stderr)) }

func run(args []string, stdout, stderr io.Writer) int {
	opts, err := parseFlags(args, stderr)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 1
	}
	if opts.showVersion {
		fmt.Fprintln(stdout, buildVersion())
		return 0
	}
	cfg, err := config.Load(opts.configPath)
	if err != nil {
		fmt.Fprintf(stderr, "ghrecon: %v\n", err)
		return 1
	}
	applyOverrides(&cfg, opts)
	if err := cfg.Validate(); err != nil {
		fmt.Fprintf(stderr, "ghrecon: %v\n", err)
		return 1
	}
	if opts.input != "" {
		if err := target.ReadFile(opts.input, &opts.target); err != nil {
			fmt.Fprintf(stderr, "ghrecon: read targets: %v\n", err)
			return 1
		}
	}
	opts.target, err = target.Normalize(opts.target)
	if err != nil {
		fmt.Fprintf(stderr, "ghrecon: %v\n", err)
		return 1
	}
	format := output.Text
	if opts.json {
		format = output.JSON
	}
	if opts.jsonl {
		format = output.JSONL
	}
	if opts.json && opts.jsonl {
		fmt.Fprintln(stderr, "ghrecon: --json and --jsonl are mutually exclusive")
		return 1
	}

	token, err := resolveToken(opts.token, opts.tokenFile)
	if err != nil {
		fmt.Fprintf(stderr, "ghrecon: authentication: %v\n", err)
		return 1
	}
	auth := gh.AuthProvider(gh.NoAuth{})
	if token != "" {
		auth = gh.TokenAuth{Token: token}
	}
	client := gh.NewClient(cfg.GitHub.APIURL, cfg.GitHub.APIVersion, cfg.Search.Timeout, auth)
	pipeline := extractor.New(cfg, opts.target)
	store := correlation.New()
	var logger *slog.Logger
	if opts.verbose && !opts.quiet {
		logger = slog.New(slog.NewJSONHandler(stderr, nil))
	}
	start := time.Now()
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	initial := query.Plan(opts.target, cfg)
	seenQueries := map[string]bool{}
	for _, q := range initial {
		seenQueries[string(q.Kind)+"\x00"+q.Text] = true
	}
	pending := initial
	var scanErrors []model.ScanError
	executed := 0
	for len(pending) > 0 && executed < cfg.Search.MaxQueries {
		if len(pending) > cfg.Search.MaxQueries-executed {
			pending = pending[:cfg.Search.MaxQueries-executed]
		}
		if logger != nil {
			logger.Info("executing search round", "queries", len(pending), "depth", pending[0].Depth)
		}
		results := scheduler.Run(ctx, pending, cfg.Search.Concurrency, client.Rates.Snapshot, func(ctx context.Context, q query.Query) (any, error) {
			switch q.Kind {
			case query.Repository:
				return client.SearchRepositories(ctx, q.Text, cfg.Search.MaxResults, cfg.Search.IncludeForks, cfg.Search.IncludeArchived)
			case query.Code:
				return client.SearchCode(ctx, q.Text, cfg.Search.MaxResults)
			case query.User:
				return client.SearchUsers(ctx, q.Text, cfg.Search.MaxResults)
			default:
				return nil, fmt.Errorf("unsupported query kind %q", q.Kind)
			}
		})
		executed += len(pending)
		maxRoundDepth := 0
		for _, result := range results {
			if result.Query.Depth > maxRoundDepth {
				maxRoundDepth = result.Query.Depth
			}
			switch value := result.Value.(type) {
			case []model.Finding:
				for _, f := range value {
					store.Add(f)
				}
			case []model.SearchDocument:
				for _, doc := range value {
					for _, f := range pipeline.Extract(doc) {
						store.Add(f)
					}
				}
			}
			if result.Err != nil {
				kind := "query"
				var apiErr *gh.APIError
				if errors.As(result.Err, &apiErr) {
					kind = string(apiErr.Kind)
				}
				scanErrors = append(scanErrors, model.ScanError{Query: result.Query.Text, Kind: kind, Message: result.Err.Error()})
				if logger != nil {
					logger.Warn("query failed", "kind", result.Query.Kind, "query", result.Query.Text, "error_kind", kind, "error", result.Err)
				}
				continue
			}
		}
		if logger != nil {
			rate := client.Rates.Snapshot()
			logger.Info("search round complete", "findings", len(store.Findings()), "rate_remaining", rate.Remaining, "rate_limit", rate.Limit, "rate_resource", rate.Resource)
		}
		if cfg.Search.Profile != "deep" || maxRoundDepth >= cfg.Search.MaxDepth {
			break
		}
		pending = nil
		for _, f := range store.Findings() {
			if f.Type != "subdomain" {
				continue
			}
			q := query.Query{Kind: query.Code, Text: "\"" + f.Value + "\"", Reason: "bounded discovery expansion", Depth: maxRoundDepth + 1}
			key := string(q.Kind) + "\x00" + q.Text
			if !seenQueries[key] {
				seenQueries[key] = true
				pending = append(pending, q)
			}
		}
	}

	findings := store.Findings()
	if opts.reposOnly {
		filtered := findings[:0]
		for _, f := range findings {
			if f.Type == "repository" {
				filtered = append(filtered, f)
			}
		}
		findings = filtered
	}
	extractor.Sort(findings)
	counts := map[string]int{}
	for _, f := range findings {
		counts[f.Type]++
	}
	status := "completed"
	partial := len(scanErrors) > 0
	if partial {
		status = "partial"
	}
	report := model.Report{
		SchemaVersion: model.SchemaVersion, Status: status, Partial: partial, Findings: findings, Errors: scanErrors,
		Summary: model.Summary{Counts: counts, Queries: executed, APIRequests: client.Requests(), DurationMS: time.Since(start).Milliseconds()},
	}
	writer := stdout
	var file *os.File
	if opts.outputPath != "" {
		file, err = os.Create(opts.outputPath)
		if err != nil {
			fmt.Fprintf(stderr, "ghrecon: create output: %v\n", err)
			return 1
		}
		defer file.Close()
		writer = file
	}
	if err := output.WriteWithOptions(writer, format, report, output.Options{ShowSources: opts.showSources}); err != nil {
		fmt.Fprintf(stderr, "ghrecon: write output: %v\n", err)
		return 1
	}
	if partial {
		return 2
	}
	return 0
}

func buildVersion() string {
	if version != "dev" {
		if strings.HasPrefix(version, "v") {
			return version
		}
		return "v" + version
	}
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return version
}

func parseFlags(args []string, stderr io.Writer) (options, error) {
	var o options
	fs := flag.NewFlagSet("ghrecon", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Var((*stringList)(&o.target.Domains), "domain", "target root domain (repeatable)")
	fs.Var((*stringList)(&o.target.Companies), "company", "target company name (repeatable)")
	fs.Var((*stringList)(&o.target.Keywords), "keyword", "target keyword (repeatable)")
	fs.Var((*stringList)(&o.target.Organizations), "org", "known GitHub organization (repeatable)")
	fs.Var((*stringList)(&o.target.Repositories), "repo", "known owner/repository (repeatable)")
	fs.StringVar(&o.input, "input", "", "target file")
	fs.StringVar(&o.configPath, "config", "", "YAML configuration file")
	fs.StringVar(&o.tokenFile, "token-file", "", "file containing a GitHub token")
	fs.StringVar(&o.token, "token", "", "GitHub token (may be visible in process lists and shell history)")
	fs.StringVar(&o.profile, "profile", "", "quick, standard, or deep")
	fs.BoolVar(&o.json, "json", false, "write a JSON report")
	fs.BoolVar(&o.jsonl, "jsonl", false, "write one finding per JSON line")
	fs.StringVar(&o.outputPath, "output", "", "write output to file")
	fs.BoolVar(&o.reposOnly, "repos-only", false, "emit repository candidates only")
	fs.BoolVar(&o.showSources, "show-sources", false, "show repository, path, URL, and query provenance in text output")
	fs.IntVar(&o.maxResults, "max-results", 0, "maximum API results per query")
	fs.IntVar(&o.maxQueries, "max-queries", 0, "maximum queries")
	fs.IntVar(&o.maxDepth, "max-depth", -1, "maximum deep expansion depth")
	fs.IntVar(&o.concurrency, "concurrency", 0, "maximum concurrent queries")
	fs.DurationVar(&o.timeout, "timeout", 0, "HTTP request timeout")
	fs.BoolVar(&o.includeForks, "include-forks", false, "include forked repositories")
	fs.BoolVar(&o.includeArchived, "include-archived", false, "include archived repositories")
	fs.BoolVar(&o.verbose, "verbose", false, "write progress to stderr")
	fs.BoolVar(&o.quiet, "quiet", false, "suppress progress logs")
	fs.BoolVar(&o.showVersion, "version", false, "print version")
	if err := fs.Parse(args); err != nil {
		return o, err
	}
	if fs.NArg() > 0 {
		return o, fmt.Errorf("unexpected arguments: %s", strings.Join(fs.Args(), " "))
	}
	return o, nil
}

func applyOverrides(cfg *config.Config, o options) {
	if o.profile != "" {
		cfg.Search.Profile = o.profile
	}
	if o.maxResults > 0 {
		cfg.Search.MaxResults = o.maxResults
	}
	if o.maxQueries > 0 {
		cfg.Search.MaxQueries = o.maxQueries
	}
	if o.maxDepth >= 0 {
		cfg.Search.MaxDepth = o.maxDepth
	}
	if o.concurrency > 0 {
		cfg.Search.Concurrency = o.concurrency
	}
	if o.timeout > 0 {
		cfg.Search.Timeout = o.timeout
	}
	if o.includeForks {
		cfg.Search.IncludeForks = true
	}
	if o.includeArchived {
		cfg.Search.IncludeArchived = true
	}
}
