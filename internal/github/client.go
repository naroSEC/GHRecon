package github

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

type ErrorKind string

const (
	ErrAuthentication ErrorKind = "authentication"
	ErrRateLimited    ErrorKind = "rate_limited"
	ErrSecondaryLimit ErrorKind = "secondary_rate_limit"
	ErrNetwork        ErrorKind = "network"
	ErrInvalidQuery   ErrorKind = "invalid_query"
	ErrAPISchema      ErrorKind = "api_schema"
	ErrIncomplete     ErrorKind = "incomplete_results"
	ErrAPI            ErrorKind = "api"
)

type APIError struct {
	Kind       ErrorKind
	StatusCode int
	Message    string
	Retryable  bool
}

func (e *APIError) Error() string { return fmt.Sprintf("%s: %s", e.Kind, e.Message) }

type Client struct {
	BaseURL    string
	APIVersion string
	HTTP       *http.Client
	Auth       AuthProvider
	Rates      *RateManager
	MaxRetries int
	requests   atomic.Int64
	sleep      func(context.Context, time.Duration) error
}

func NewClient(baseURL, version string, timeout time.Duration, auth AuthProvider) *Client {
	return &Client{
		BaseURL: strings.TrimRight(baseURL, "/"), APIVersion: version,
		HTTP: &http.Client{Timeout: timeout}, Auth: auth, Rates: NewRateManager(), MaxRetries: 3,
		sleep: sleepContext,
	}
}

func (c *Client) Requests() int { return int(c.requests.Load()) }

func (c *Client) get(ctx context.Context, path string, query url.Values, out any) error {
	var last error
	for attempt := 0; attempt <= c.MaxRetries; attempt++ {
		if err := c.Rates.Wait(ctx); err != nil {
			return err
		}
		u := c.BaseURL + path
		if encoded := query.Encode(); encoded != "" {
			u += "?" + encoded
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		if err != nil {
			return err
		}
		req.Header.Set("Accept", "application/vnd.github.text-match+json")
		req.Header.Set("X-GitHub-Api-Version", c.APIVersion)
		req.Header.Set("User-Agent", "GHRecon/1")
		c.Auth.Apply(req)
		c.requests.Add(1)
		resp, err := c.HTTP.Do(req)
		if err != nil {
			last = &APIError{Kind: ErrNetwork, Message: err.Error(), Retryable: true}
			if attempt < c.MaxRetries {
				if err := c.sleep(ctx, backoff(attempt, 0)); err != nil {
					return err
				}
				continue
			}
			return last
		}
		c.Rates.Update(resp.Header)
		body, readErr := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
		resp.Body.Close()
		if readErr != nil {
			return &APIError{Kind: ErrNetwork, Message: readErr.Error(), Retryable: true}
		}
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			if err := json.Unmarshal(body, out); err != nil {
				return &APIError{Kind: ErrAPISchema, StatusCode: resp.StatusCode, Message: err.Error()}
			}
			return nil
		}
		apiErr := classify(resp.StatusCode, body, c.Rates.Snapshot())
		last = apiErr
		if !apiErr.Retryable || attempt == c.MaxRetries {
			return apiErr
		}
		delay := retryAfter(resp.Header.Get("Retry-After"))
		if apiErr.Kind == ErrSecondaryLimit && delay == 0 {
			delay = time.Minute
		}
		if err := c.sleep(ctx, backoff(attempt, delay)); err != nil {
			return err
		}
	}
	return last
}

func classify(status int, body []byte, rate RateState) *APIError {
	var message struct {
		Message string `json:"message"`
	}
	_ = json.Unmarshal(body, &message)
	if message.Message == "" {
		message.Message = http.StatusText(status)
	}
	lower := strings.ToLower(message.Message)
	switch {
	case status == http.StatusUnauthorized:
		return &APIError{Kind: ErrAuthentication, StatusCode: status, Message: message.Message}
	case (status == http.StatusForbidden || status == http.StatusTooManyRequests) && (strings.Contains(lower, "secondary rate") || strings.Contains(lower, "abuse")):
		return &APIError{Kind: ErrSecondaryLimit, StatusCode: status, Message: message.Message, Retryable: true}
	case status == http.StatusTooManyRequests || (status == http.StatusForbidden && rate.Remaining == 0 && !rate.Reset.IsZero()):
		return &APIError{Kind: ErrRateLimited, StatusCode: status, Message: message.Message, Retryable: true}
	case status == http.StatusUnprocessableEntity || status == http.StatusBadRequest:
		return &APIError{Kind: ErrInvalidQuery, StatusCode: status, Message: message.Message}
	case status >= 500:
		return &APIError{Kind: ErrAPI, StatusCode: status, Message: message.Message, Retryable: true}
	default:
		return &APIError{Kind: ErrAPI, StatusCode: status, Message: message.Message}
	}
}

func retryAfter(v string) time.Duration {
	if v == "" {
		return 0
	}
	if seconds, err := strconv.Atoi(v); err == nil {
		return time.Duration(seconds) * time.Second
	}
	if when, err := http.ParseTime(v); err == nil {
		if d := time.Until(when); d > 0 {
			return d
		}
	}
	return 0
}

func backoff(attempt int, minimum time.Duration) time.Duration {
	d := time.Duration(1<<attempt)*time.Second + time.Duration(rand.Intn(250))*time.Millisecond
	if minimum > d {
		return minimum
	}
	return d
}

func sleepContext(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return context.Cause(ctx)
	case <-timer.C:
		return nil
	}
}

func IsKind(err error, kind ErrorKind) bool {
	var target *APIError
	return errors.As(err, &target) && target.Kind == kind
}
