package extractor

import (
	"net"
	"net/url"
	"path"
	"regexp"
	"sort"
	"strings"

	"github.com/ghrecon/ghrecon/internal/config"
	"github.com/ghrecon/ghrecon/internal/model"
)

var (
	hostCandidate     = regexp.MustCompile("(?i)(?:[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\\.)+[a-z]{2,63}")
	urlCandidate      = regexp.MustCompile("(?i)https?://[a-z0-9._~%:-]+(?:/[^\\s\\\"'<>]*)?")
	endpointCandidate = regexp.MustCompile("(?:^|[\\s\\\"'=(,:])(/(?:api|admin|oauth|auth|graphql|v[0-9]+|internal|debug|health|metrics|webhook)(?:/[A-Za-z0-9._~%!$&'()*+,;=:@{}-]+)*)")
	awsAsset          = regexp.MustCompile("(?i)(?:[a-z0-9][a-z0-9.-]{1,61}\\.s3(?:[.-][a-z0-9-]+)?\\.amazonaws\\.com|[a-z0-9.-]+\\.execute-api\\.[a-z0-9-]+\\.amazonaws\\.com)")
	azureAsset        = regexp.MustCompile("(?i)[a-z0-9][a-z0-9-]{1,62}\\.(?:blob|file|queue|table)\\.core\\.windows\\.net")
	gcpAsset          = regexp.MustCompile("(?i)(?:storage\\.googleapis\\.com/[a-z0-9._-]+|[a-z0-9][a-z0-9._-]+\\.storage\\.googleapis\\.com|[a-z0-9-]+\\.run\\.app)")
)

type Pipeline struct {
	cfg         config.Config
	rootDomains []string
}

func New(cfg config.Config, target model.Target) *Pipeline {
	return &Pipeline{cfg: cfg, rootDomains: append([]string(nil), target.Domains...)}
}

func (p *Pipeline) Extract(doc model.SearchDocument) []model.Finding {
	var out []model.Finding
	provenance := model.Provenance{Source: doc.Source, Repository: doc.Repository, Path: doc.Path, URL: doc.URL, Query: doc.Query}
	if p.cfg.Extractors.Domains {
		seen := map[string]bool{}
		for _, candidate := range hostCandidate.FindAllString(doc.Text, -1) {
			host := strings.ToLower(strings.TrimSuffix(candidate, "."))
			for _, root := range p.rootDomains {
				if host == root || strings.HasSuffix(host, "."+root) {
					if !seen[host] {
						seen[host] = true
						kind, confidence := "subdomain", .95
						if host == root {
							kind, confidence = "domain", .90
						}
						out = append(out, finding(kind, host, confidence, provenance))
					}
					break
				}
			}
		}
	}
	if p.cfg.Extractors.URLs {
		for _, raw := range urlCandidate.FindAllString(doc.Text, -1) {
			raw = strings.TrimRight(raw, ".,;:)]}")
			u, err := url.Parse(raw)
			if err != nil || (u.Scheme != "http" && u.Scheme != "https") || (net.ParseIP(u.Hostname()) == nil && u.Hostname() == "") {
				continue
			}
			f := finding("url", u.String(), urlConfidence(u.Hostname(), p.rootDomains), provenance)
			f.Attributes = map[string]any{"scheme": u.Scheme, "host": u.Hostname(), "path": u.EscapedPath()}
			if u.Port() != "" {
				f.Attributes["port"] = u.Port()
			}
			if u.RawQuery != "" {
				f.Attributes["has_query"] = true
			}
			out = append(out, f)
		}
	}
	if p.cfg.Search.Profile != "quick" && p.cfg.Extractors.Endpoints {
		for _, match := range endpointCandidate.FindAllStringSubmatch(doc.Text, -1) {
			if len(match) > 1 && validEndpoint(match[1]) {
				out = append(out, finding("endpoint", match[1], .72, provenance))
			}
		}
	}
	if p.cfg.Search.Profile != "quick" && p.cfg.Extractors.Cloud {
		for _, rule := range []struct {
			provider string
			re       *regexp.Regexp
		}{{"aws", awsAsset}, {"azure", azureAsset}, {"gcp", gcpAsset}} {
			for _, value := range rule.re.FindAllString(doc.Text, -1) {
				f := finding("cloud_asset", strings.ToLower(value), .86, provenance)
				f.Provider = rule.provider
				out = append(out, f)
			}
		}
	}
	base := strings.ToLower(path.Base(strings.ReplaceAll(doc.Path, "\\", "/")))
	if p.cfg.Search.Profile != "quick" {
		for _, pattern := range p.cfg.SecretCandidateFiles {
			if base == strings.ToLower(pattern) || strings.Contains(base, strings.ToLower(pattern)) {
				f := finding("secret_candidate", doc.Repository+":"+doc.Path, .60, provenance)
				f.Reason = "potential credential file"
				out = append(out, f)
				break
			}
		}
	}
	return out
}

func finding(kind, value string, confidence float64, p model.Provenance) model.Finding {
	return model.Finding{SchemaVersion: model.SchemaVersion, Type: kind, Value: value, Confidence: confidence, Sources: []model.Provenance{p}}
}

func validEndpoint(v string) bool {
	if len(v) < 4 || len(v) > 300 || strings.Contains(v, "//") {
		return false
	}
	switch strings.ToLower(path.Ext(v)) {
	case ".png", ".jpg", ".jpeg", ".gif", ".svg", ".css", ".js", ".map":
		return false
	}
	return true
}

func urlConfidence(host string, roots []string) float64 {
	host = strings.ToLower(host)
	for _, root := range roots {
		if host == root || strings.HasSuffix(host, "."+root) {
			return .93
		}
	}
	return .65
}

func Sort(findings []model.Finding) {
	sort.Slice(findings, func(i, j int) bool {
		if findings[i].Type == findings[j].Type {
			if findings[i].Provider == findings[j].Provider {
				return findings[i].Value < findings[j].Value
			}
			return findings[i].Provider < findings[j].Provider
		}
		return findings[i].Type < findings[j].Type
	})
}
