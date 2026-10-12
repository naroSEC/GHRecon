package query

import (
	"fmt"
	"strings"

	"github.com/ghrecon/ghrecon/internal/config"
	"github.com/ghrecon/ghrecon/internal/model"
)

type Kind string

const (
	Repository Kind = "repository"
	Code       Kind = "code"
	User       Kind = "user"
)

type Query struct {
	Kind   Kind
	Text   string
	Reason string
	Depth  int
}

func Plan(t model.Target, cfg config.Config) []Query {
	var out []Query
	add := func(k Kind, text, reason string) {
		if len(out) < cfg.Search.MaxQueries && text != "" {
			out = append(out, Query{Kind: k, Text: text, Reason: reason})
		}
	}
	for _, d := range t.Domains {
		add(Repository, quote(d), "target domain")
		add(Code, quote(d), "target domain")
	}
	for _, c := range t.Companies {
		add(Repository, quote(c), "company name")
		add(Code, quote(c), "company name")
		add(User, quote(c)+" type:org", "company organization")
	}
	for _, k := range t.Keywords {
		add(Repository, quoteIfNeeded(k), "keyword")
		add(Code, quoteIfNeeded(k), "keyword")
	}
	for _, o := range t.Organizations {
		add(Repository, "org:"+o, "known organization")
		add(Code, "org:"+o, "known organization")
		add(User, o+" type:org", "known organization")
	}
	for _, r := range t.Repositories {
		add(Repository, "repo:"+r, "known repository")
		add(Code, "repo:"+r, "known repository")
	}
	if cfg.Search.Profile != "quick" {
		seeds := append(append([]string{}, t.Domains...), t.Keywords...)
		for _, seed := range seeds {
			for _, file := range cfg.InterestingFiles {
				add(Code, fmt.Sprintf("%s filename:%s", quoteIfNeeded(seed), file), "interesting file")
			}
		}
	}
	for _, custom := range cfg.CustomQueries {
		kind := Kind(custom.Type)
		if kind == Repository || kind == Code || kind == User {
			add(kind, custom.Query, custom.Reason)
		}
	}
	return dedup(out, cfg.Search.MaxQueries)
}

func quote(s string) string {
	return "\"" + strings.ReplaceAll(s, "\"", "") + "\""
}

func quoteIfNeeded(s string) string {
	if strings.ContainsAny(s, " \t") {
		return quote(s)
	}
	return s
}

func dedup(in []Query, max int) []Query {
	seen := map[string]bool{}
	out := make([]Query, 0, len(in))
	for _, q := range in {
		k := string(q.Kind) + "\x00" + q.Text
		if !seen[k] {
			seen[k] = true
			out = append(out, q)
			if len(out) >= max {
				break
			}
		}
	}
	return out
}
