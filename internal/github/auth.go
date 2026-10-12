package github

import "net/http"

type AuthProvider interface {
	Apply(*http.Request)
}

type TokenAuth struct{ Token string }

func (a TokenAuth) Apply(r *http.Request) {
	if a.Token != "" {
		r.Header.Set("Authorization", "Bearer "+a.Token)
	}
}

type NoAuth struct{}

func (NoAuth) Apply(*http.Request) {}
