package github

import (
	"context"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/ghrecon/ghrecon/internal/model"
)

type repoOwner struct {
	Login string `json:"login"`
}
type repoItem struct {
	FullName      string    `json:"full_name"`
	Owner         repoOwner `json:"owner"`
	HTMLURL       string    `json:"html_url"`
	CloneURL      string    `json:"clone_url"`
	Description   string    `json:"description"`
	DefaultBranch string    `json:"default_branch"`
	Visibility    string    `json:"visibility"`
	Private       bool      `json:"private"`
	Archived      bool      `json:"archived"`
	Fork          bool      `json:"fork"`
	Stars         int       `json:"stargazers_count"`
	Language      string    `json:"language"`
	UpdatedAt     time.Time `json:"updated_at"`
}
type repoResponse struct {
	Total      int        `json:"total_count"`
	Incomplete bool       `json:"incomplete_results"`
	Items      []repoItem `json:"items"`
}

type codeRepository struct {
	FullName string `json:"full_name"`
}
type textMatch struct {
	Fragment string `json:"fragment"`
}
type codeItem struct {
	Name        string         `json:"name"`
	Path        string         `json:"path"`
	HTMLURL     string         `json:"html_url"`
	Repository  codeRepository `json:"repository"`
	TextMatches []textMatch    `json:"text_matches"`
}
type codeResponse struct {
	Total      int        `json:"total_count"`
	Incomplete bool       `json:"incomplete_results"`
	Items      []codeItem `json:"items"`
}
type userItem struct {
	Login   string `json:"login"`
	HTMLURL string `json:"html_url"`
	Type    string `json:"type"`
}
type userResponse struct {
	Total      int        `json:"total_count"`
	Incomplete bool       `json:"incomplete_results"`
	Items      []userItem `json:"items"`
}

func (c *Client) SearchRepositories(ctx context.Context, query string, maxResults int, includeForks, includeArchived bool) ([]model.Finding, error) {
	var out []model.Finding
	for page := 1; len(out) < maxResults; page++ {
		perPage := min(100, maxResults-len(out))
		var response repoResponse
		values := url.Values{"q": {query}, "page": {strconv.Itoa(page)}, "per_page": {strconv.Itoa(perPage)}}
		if err := c.get(ctx, "/search/repositories", values, &response); err != nil {
			return out, err
		}
		for _, item := range response.Items {
			if (!includeForks && item.Fork) || (!includeArchived && item.Archived) {
				continue
			}
			visibility := item.Visibility
			if visibility == "" {
				if item.Private {
					visibility = "private"
				} else {
					visibility = "public"
				}
			}
			repo := model.Repository{
				FullName: item.FullName, Owner: item.Owner.Login, HTMLURL: item.HTMLURL, CloneURL: item.CloneURL,
				Description: item.Description, DefaultBranch: item.DefaultBranch, Visibility: visibility,
				Archived: item.Archived, Fork: item.Fork, Stars: item.Stars, Language: item.Language, LastUpdated: item.UpdatedAt,
			}
			out = append(out, model.Finding{
				SchemaVersion: model.SchemaVersion, Type: "repository", Value: item.FullName, Confidence: repositoryConfidence(query, item),
				Repository: &repo, Sources: []model.Provenance{{Source: "github_repository_search", URL: item.HTMLURL, Query: query}},
			})
			if len(out) == maxResults {
				break
			}
		}
		if response.Incomplete {
			return out, &APIError{Kind: ErrIncomplete, Message: "GitHub reported incomplete repository search results"}
		}
		if len(response.Items) < perPage || page*perPage >= response.Total || page >= 10 {
			break
		}
	}
	return out, nil
}

func (c *Client) SearchCode(ctx context.Context, query string, maxResults int) ([]model.SearchDocument, error) {
	var out []model.SearchDocument
	for page := 1; len(out) < maxResults; page++ {
		perPage := min(100, maxResults-len(out))
		var response codeResponse
		values := url.Values{"q": {query}, "page": {strconv.Itoa(page)}, "per_page": {strconv.Itoa(perPage)}}
		if err := c.get(ctx, "/search/code", values, &response); err != nil {
			return out, err
		}
		for _, item := range response.Items {
			if len(item.TextMatches) == 0 {
				out = append(out, model.SearchDocument{Source: "github_code_search", Repository: item.Repository.FullName, Path: item.Path, URL: item.HTMLURL, Query: query})
			} else {
				for _, match := range item.TextMatches {
					out = append(out, model.SearchDocument{Source: "github_code_search", Repository: item.Repository.FullName, Path: item.Path, URL: item.HTMLURL, Query: query, Text: match.Fragment})
				}
			}
			if len(out) >= maxResults {
				break
			}
		}
		if response.Incomplete {
			return out, &APIError{Kind: ErrIncomplete, Message: "GitHub reported incomplete code search results"}
		}
		if len(response.Items) < perPage || page*perPage >= response.Total || page >= 10 {
			break
		}
	}
	return out, nil
}

func (c *Client) SearchUsers(ctx context.Context, query string, maxResults int) ([]model.Finding, error) {
	var out []model.Finding
	for page := 1; len(out) < maxResults; page++ {
		perPage := min(100, maxResults-len(out))
		var response userResponse
		values := url.Values{"q": {query}, "page": {strconv.Itoa(page)}, "per_page": {strconv.Itoa(perPage)}}
		if err := c.get(ctx, "/search/users", values, &response); err != nil {
			return out, err
		}
		for _, item := range response.Items {
			kind := "user"
			if item.Type == "Organization" {
				kind = "organization"
			}
			out = append(out, model.Finding{
				SchemaVersion: model.SchemaVersion, Type: kind, Value: item.Login, Confidence: .75,
				Sources: []model.Provenance{{Source: "github_user_search", URL: item.HTMLURL, Query: query}},
			})
			if len(out) >= maxResults {
				break
			}
		}
		if response.Incomplete {
			return out, &APIError{Kind: ErrIncomplete, Message: "GitHub reported incomplete user search results"}
		}
		if len(response.Items) < perPage || page*perPage >= response.Total || page >= 10 {
			break
		}
	}
	return out, nil
}

func repositoryConfidence(query string, item repoItem) float64 {
	score := 0.55
	if item.Owner.Login != "" && containsFold(query, item.Owner.Login) {
		score += .15
	}
	if containsFold(item.FullName, trimQuery(query)) {
		score += .12
	}
	if containsFold(item.Description, trimQuery(query)) {
		score += .08
	}
	if score > .95 {
		return .95
	}
	return score
}

func containsFold(s, sub string) bool {
	if sub == "" {
		return false
	}
	return strings.Contains(strings.ToLower(s), strings.ToLower(sub))
}
func trimQuery(s string) string {
	for len(s) > 0 && (s[0] == '"' || s[0] == ' ') {
		s = s[1:]
	}
	for len(s) > 0 && (s[len(s)-1] == '"' || s[len(s)-1] == ' ') {
		s = s[:len(s)-1]
	}
	return s
}
