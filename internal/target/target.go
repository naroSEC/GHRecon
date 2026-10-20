package target

import (
	"bufio"
	"fmt"
	"net"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/naroSEC/GHRecon/internal/model"
)

var domainRE = regexp.MustCompile("(?i)^(?:[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\\.)+[a-z]{2,63}$")
var repoRE = regexp.MustCompile("^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$")

func Normalize(t model.Target) (model.Target, error) {
	t.Domains = normalize(t.Domains, func(s string) string {
		s = strings.TrimSpace(strings.ToLower(s))
		s = strings.TrimPrefix(s, "*.")
		if u, err := url.Parse(s); err == nil && u.Hostname() != "" {
			s = u.Hostname()
		}
		if host, _, err := net.SplitHostPort(s); err == nil {
			s = host
		}
		return strings.TrimSuffix(s, ".")
	})
	for _, d := range t.Domains {
		if !domainRE.MatchString(d) {
			return t, fmt.Errorf("invalid domain %q", d)
		}
	}
	t.Companies = normalize(t.Companies, strings.TrimSpace)
	t.Keywords = normalize(t.Keywords, strings.TrimSpace)
	t.Organizations = normalize(t.Organizations, strings.TrimSpace)
	t.Repositories = normalize(t.Repositories, strings.TrimSpace)
	for _, r := range t.Repositories {
		if !repoRE.MatchString(r) {
			return t, fmt.Errorf("invalid repository %q (expected owner/name)", r)
		}
	}
	if len(t.Domains)+len(t.Companies)+len(t.Keywords)+len(t.Organizations)+len(t.Repositories) == 0 {
		return t, fmt.Errorf("at least one target identifier is required")
	}
	return t, nil
}

func ReadFile(path string, t *model.Target) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	for n := 1; s.Scan(); n++ {
		line := strings.TrimSpace(s.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		kind, value, found := strings.Cut(line, ":")
		if !found {
			if domainRE.MatchString(strings.ToLower(line)) {
				t.Domains = append(t.Domains, line)
			} else {
				t.Keywords = append(t.Keywords, line)
			}
			continue
		}
		value = strings.TrimSpace(value)
		switch strings.ToLower(strings.TrimSpace(kind)) {
		case "domain":
			t.Domains = append(t.Domains, value)
		case "company":
			t.Companies = append(t.Companies, value)
		case "keyword":
			t.Keywords = append(t.Keywords, value)
		case "org", "organization":
			t.Organizations = append(t.Organizations, value)
		case "repo", "repository":
			t.Repositories = append(t.Repositories, value)
		default:
			return fmt.Errorf("%s:%d: unknown target type %q", path, n, kind)
		}
	}
	return s.Err()
}

func normalize(in []string, fn func(string) string) []string {
	seen := map[string]string{}
	for _, raw := range in {
		v := fn(raw)
		if v != "" {
			k := strings.ToLower(v)
			if _, ok := seen[k]; !ok {
				seen[k] = v
			}
		}
	}
	out := make([]string, 0, len(seen))
	for _, v := range seen {
		out = append(out, v)
	}
	sort.Strings(out)
	return out
}
