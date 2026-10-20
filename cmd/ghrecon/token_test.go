package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveTokenFromFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "github.token")
	if err := os.WriteFile(path, []byte("github_pat_example\r\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	token, err := resolveToken("", path)
	if err != nil {
		t.Fatal(err)
	}
	if token != "github_pat_example" {
		t.Fatal("token file was not normalized")
	}
}

func TestResolveTokenFromArgument(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "environment-token")
	token, err := resolveToken("argument-token", "")
	if err != nil {
		t.Fatal(err)
	}
	if token != "argument-token" {
		t.Fatal("explicit token did not take precedence")
	}
}

func TestResolveTokenFromEnvironment(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "environment-token")
	token, err := resolveToken("", "")
	if err != nil {
		t.Fatal(err)
	}
	if token != "environment-token" {
		t.Fatal("environment fallback failed")
	}
}

func TestResolveTokenRejectsAmbiguousSources(t *testing.T) {
	if _, err := resolveToken("argument-token", "token.txt"); err == nil {
		t.Fatal("expected mutually exclusive source error")
	}
}

func TestResolveTokenRejectsMultipleLines(t *testing.T) {
	path := filepath.Join(t.TempDir(), "github.token")
	if err := os.WriteFile(path, []byte("first\nsecond\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := resolveToken("", path); err == nil {
		t.Fatal("expected whitespace validation error")
	}
}
