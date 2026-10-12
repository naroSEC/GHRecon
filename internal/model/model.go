package model

import "time"

const SchemaVersion = "1"

type Target struct {
	Domains       []string `json:"domains" yaml:"domains"`
	Companies     []string `json:"companies" yaml:"companies"`
	Keywords      []string `json:"keywords" yaml:"keywords"`
	Organizations []string `json:"organizations" yaml:"organizations"`
	Repositories  []string `json:"repositories" yaml:"repositories"`
}

type Provenance struct {
	Source     string `json:"source"`
	Repository string `json:"repository,omitempty"`
	Path       string `json:"path,omitempty"`
	URL        string `json:"url,omitempty"`
	Query      string `json:"query,omitempty"`
	Fragment   string `json:"fragment,omitempty"`
}

type Repository struct {
	FullName      string    `json:"full_name"`
	Owner         string    `json:"owner"`
	HTMLURL       string    `json:"html_url"`
	CloneURL      string    `json:"clone_url,omitempty"`
	Description   string    `json:"description,omitempty"`
	DefaultBranch string    `json:"default_branch,omitempty"`
	Visibility    string    `json:"visibility"`
	Archived      bool      `json:"archived"`
	Fork          bool      `json:"fork"`
	Stars         int       `json:"stars"`
	Language      string    `json:"language,omitempty"`
	LastUpdated   time.Time `json:"last_updated,omitempty"`
}

// Finding is the stable, deduplicated interchange unit. Sources contains every
// observation that contributed to the asset; it is never intentionally omitted.
type Finding struct {
	SchemaVersion string         `json:"schema_version"`
	Type          string         `json:"type"`
	Value         string         `json:"value"`
	Provider      string         `json:"provider,omitempty"`
	Confidence    float64        `json:"confidence"`
	Repository    *Repository    `json:"repository_details,omitempty"`
	Reason        string         `json:"reason,omitempty"`
	Sources       []Provenance   `json:"sources"`
	Attributes    map[string]any `json:"attributes,omitempty"`
}

type ScanError struct {
	Query   string `json:"query,omitempty"`
	Kind    string `json:"kind"`
	Message string `json:"message"`
}

type Summary struct {
	Counts      map[string]int `json:"counts"`
	Queries     int            `json:"queries"`
	APIRequests int            `json:"api_requests"`
	DurationMS  int64          `json:"duration_ms"`
}

type Report struct {
	SchemaVersion string      `json:"schema_version"`
	Status        string      `json:"status"`
	Partial       bool        `json:"partial"`
	Findings      []Finding   `json:"findings"`
	Errors        []ScanError `json:"errors"`
	Summary       Summary     `json:"summary"`
}

type SearchDocument struct {
	Source     string
	Repository string
	Path       string
	URL        string
	Query      string
	Text       string
}
