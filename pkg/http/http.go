// Package http provides a fluent, retry-aware HTTP client for Kashvi.
//
// Usage:
//
//	resp, err := http.Get("https://api.example.com/users").
//	    Header("Authorization", "Bearer "+token).
//	    Timeout(5 * time.Second).
//	    Retry(3, time.Second).
//	    Send()
//
//	var users []User
//	err = resp.JSON(&users)
//
//	// POST JSON body
//	resp, err := http.Post("https://api.example.com/users").
//	    Body(map[string]any{"name": "Shashi"}).
//	    Send()
package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	gohttp "net/http"
	"time"

	"github.com/shashiranjanraj/kashvi/pkg/logger"
)

// defaultTransport is the high-performance connection-pooled transport used in
// production.  Tests can replace DefaultClient.Transport to inject mocks.
var defaultTransport = &gohttp.Transport{
	MaxIdleConns:        200,
	MaxIdleConnsPerHost: 100,
	IdleConnTimeout:     90 * time.Second,
	DisableCompression:  false,
}

// DefaultClient is the shared HTTP client used by all Kashvi outgoing requests.
// Tests can swap DefaultClient.Transport to intercept calls:
//
//	http.DefaultClient.Transport = myMockTransport
//	defer http.ResetTransport()
var DefaultClient = &gohttp.Client{
	Transport: defaultTransport,
}

// ResetTransport restores the production transport on DefaultClient.
// Call via defer after injecting a test transport.
func ResetTransport() {
	DefaultClient.Transport = defaultTransport
}

// ------------------- Request -------------------

// Request is a fluent HTTP request builder.
type Request struct {
	method    string
	url       string
	headers   map[string]string
	body      interface{}
	timeout   time.Duration
	retries   int
	retryWait time.Duration
	ctx       context.Context
}

// Get starts a GET request.
func Get(url string) *Request { return newRequest(gohttp.MethodGet, url) }

// Post starts a POST request.
func Post(url string) *Request { return newRequest(gohttp.MethodPost, url) }

// Put starts a PUT request.
func Put(url string) *Request { return newRequest(gohttp.MethodPut, url) }

// Patch starts a PATCH request.
func Patch(url string) *Request { return newRequest(gohttp.MethodPatch, url) }

// Delete starts a DELETE request.
func Delete(url string) *Request { return newRequest(gohttp.MethodDelete, url) }

func newRequest(method, url string) *Request {
	return &Request{
		method:    method,
		url:       url,
		headers:   map[string]string{"Content-Type": "application/json", "Accept": "application/json"},
		timeout:   30 * time.Second,
		retries:   1,
		retryWait: 500 * time.Millisecond,
		ctx:       context.Background(),
	}
}

// Header adds a single header to the request.
func (r *Request) Header(key, value string) *Request {
	r.headers[key] = value
	return r
}

// Headers merges a map of headers.
func (r *Request) Headers(h map[string]string) *Request {
	for k, v := range h {
		r.headers[k] = v
	}
	return r
}

// Bearer sets the Authorization: Bearer <token> header.
func (r *Request) Bearer(token string) *Request {
	return r.Header("Authorization", "Bearer "+token)
}

// Body sets the request body. v is marshalled to JSON automatically.
// Pass a string or []byte to send raw bodies.
func (r *Request) Body(v interface{}) *Request {
	r.body = v
	return r
}

// Timeout sets the per-attempt timeout.
func (r *Request) Timeout(d time.Duration) *Request {
	r.timeout = d
	return r
}

// Retry configures automatic retries on failure.
// n is total attempts (1 = no retry), wait is the initial backoff (doubles each attempt).
func (r *Request) Retry(n int, wait time.Duration) *Request {
	r.retries = n
	r.retryWait = wait
	return r
}

// WithContext sets a custom context.
func (r *Request) WithContext(ctx context.Context) *Request {
	r.ctx = ctx
	return r
}

// ------------------- Send -------------------

// Send executes the request and returns a Response.
func (r *Request) Send() (*Response, error) {
	var lastErr error

	for attempt := 1; attempt <= r.retries; attempt++ {
		resp, err := r.do()
		if err == nil {
			return resp, nil
		}
		lastErr = err
		if attempt < r.retries {
			// Exponential backoff: wait * 2^(attempt-1)
			backoff := time.Duration(float64(r.retryWait) * math.Pow(2, float64(attempt-1)))
			logger.Warn("http: request failed, retrying",
				"url", r.url, "attempt", attempt, "backoff", backoff, "error", err)
			time.Sleep(backoff)
		}
	}

	return nil, fmt.Errorf("http: all %d attempts failed for %s %s: %w", r.retries, r.method, r.url, lastErr)
}

func (r *Request) do() (*Response, error) {
	body, ct, err := r.buildBody()
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(r.ctx, r.timeout)
	defer cancel()

	req, err := gohttp.NewRequestWithContext(ctx, r.method, r.url, body)
	if err != nil {
		return nil, fmt.Errorf("http: build request: %w", err)
	}

	for k, v := range r.headers {
		req.Header.Set(k, v)
	}
	if ct != "" {
		req.Header.Set("Content-Type", ct)
	}

	resp, err := DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http: send: %w", err)
	}

	raw, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return nil, fmt.Errorf("http: read body: %w", err)
	}

	return &Response{
		StatusCode: resp.StatusCode,
		Headers:    resp.Header,
		Raw:        raw,
		native:     resp,
	}, nil
}

func (r *Request) buildBody() (io.Reader, string, error) {
	if r.body == nil {
		return nil, "", nil
	}
	switch v := r.body.(type) {
	case string:
		return bytes.NewBufferString(v), "text/plain", nil
	case []byte:
		return bytes.NewReader(v), "application/octet-stream", nil
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return nil, "", fmt.Errorf("http: marshal body: %w", err)
		}
		return bytes.NewReader(b), "application/json", nil
	}
}

// ------------------- Response -------------------

// Response wraps the HTTP response with convenience methods.
type Response struct {
	StatusCode int
	Headers    gohttp.Header
	Raw        []byte
	native     *gohttp.Response
}

// OK reports whether the status code is 2xx.
func (r *Response) OK() bool {
	return r.StatusCode >= 200 && r.StatusCode < 300
}

// JSON unmarshals the response body into dest.
func (r *Response) JSON(dest interface{}) error {
	if err := json.Unmarshal(r.Raw, dest); err != nil {
		return fmt.Errorf("http: decode JSON: %w", err)
	}
	return nil
}

// Text returns the response body as a string.
func (r *Response) Text() string {
	return string(r.Raw)
}

// Header returns a single response header value.
func (r *Response) Header(key string) string {
	return r.Headers.Get(key)
}

// Throw returns an error if the response status is not 2xx.
func (r *Response) Throw() error {
	if !r.OK() {
		return fmt.Errorf("http: request failed with status %d: %s", r.StatusCode, string(r.Raw))
	}
	return nil
}
