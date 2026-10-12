package correlation

import (
	"fmt"
	"strings"
	"sync"

	"github.com/ghrecon/ghrecon/internal/model"
)

type Store struct {
	mu    sync.Mutex
	order []string
	items map[string]model.Finding
}

func New() *Store { return &Store{items: map[string]model.Finding{}} }

func (s *Store) Add(f model.Finding) {
	f.Value = strings.TrimSpace(f.Value)
	if f.Value == "" {
		return
	}
	key := fmt.Sprintf("%q|%q|%q", strings.ToLower(f.Type), strings.ToLower(f.Provider), canonicalValue(f))
	s.mu.Lock()
	defer s.mu.Unlock()
	current, ok := s.items[key]
	if !ok {
		if f.SchemaVersion == "" {
			f.SchemaVersion = model.SchemaVersion
		}
		if f.Sources == nil {
			f.Sources = []model.Provenance{}
		}
		f.Sources = dedupSources(f.Sources)
		s.items[key] = f
		s.order = append(s.order, key)
		return
	}
	if f.Confidence > current.Confidence {
		current.Confidence = f.Confidence
	}
	current.Sources = dedupSources(append(current.Sources, f.Sources...))
	if current.Repository == nil {
		current.Repository = f.Repository
	}
	if current.Reason == "" {
		current.Reason = f.Reason
	}
	if current.Attributes == nil {
		current.Attributes = f.Attributes
	}
	s.items[key] = current
}

func (s *Store) Findings() []model.Finding {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]model.Finding, 0, len(s.order))
	for _, key := range s.order {
		out = append(out, s.items[key])
	}
	return out
}

func canonicalValue(f model.Finding) string {
	switch f.Type {
	case "domain", "subdomain", "cloud_asset", "repository":
		return strings.ToLower(f.Value)
	default:
		return f.Value
	}
}

func dedupSources(in []model.Provenance) []model.Provenance {
	seen := map[string]bool{}
	out := make([]model.Provenance, 0, len(in))
	for _, p := range in {
		key := fmt.Sprintf("%q|%q|%q|%q|%q", p.Source, p.Repository, p.Path, p.URL, p.Query)
		if !seen[key] {
			seen[key] = true
			p.Fragment = ""
			out = append(out, p)
		}
	}
	return out
}
