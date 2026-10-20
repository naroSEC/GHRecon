package main

import (
	"fmt"
	"os"
	"strings"
	"unicode"
)

const maxTokenFileSize = 64 * 1024

// resolveToken applies the explicit CLI, file, environment precedence without
// ever returning token contents in an error.
func resolveToken(direct, path string) (string, error) {
	if direct != "" && path != "" {
		return "", fmt.Errorf("--token and --token-file are mutually exclusive")
	}
	if direct != "" {
		return normalizeToken(direct)
	}
	if path != "" {
		info, err := os.Stat(path)
		if err != nil {
			return "", fmt.Errorf("inspect token file: %w", err)
		}
		if !info.Mode().IsRegular() {
			return "", fmt.Errorf("token file must be a regular file")
		}
		if info.Size() > maxTokenFileSize {
			return "", fmt.Errorf("token file is unexpectedly large")
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("read token file: %w", err)
		}
		return normalizeToken(string(data))
	}
	return normalizeOptionalToken(os.Getenv("GITHUB_TOKEN"))
}

func normalizeOptionalToken(raw string) (string, error) {
	if strings.TrimSpace(raw) == "" {
		return "", nil
	}
	return normalizeToken(raw)
}

func normalizeToken(raw string) (string, error) {
	token := strings.TrimSpace(strings.TrimPrefix(raw, "\uFEFF"))
	if token == "" {
		return "", fmt.Errorf("token is empty")
	}
	if strings.IndexFunc(token, unicode.IsSpace) >= 0 {
		return "", fmt.Errorf("token contains whitespace; the file must contain only one token")
	}
	return token, nil
}
