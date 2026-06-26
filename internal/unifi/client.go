// Package unifi is the client for Ubiquiti's official UniFi APIs: the local Network
// Integration API (https://{host}/proxy/network/integration/v1) and, with a different base,
// the Site Manager cloud API (https://api.ui.com/v1). Auth is a single X-API-KEY header.
// Responses are decoded and snake_cased (see transform.go) so callers emit a stable contract.
package unifi

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/rnwolfe/ufi/internal/errs"
)

// IntegrationBase is the path prefix for the local Network Integration API.
const IntegrationBase = "/proxy/network/integration/v1"

// CloudBase is the Site Manager cloud API root.
const CloudBase = "https://api.ui.com/v1"

// Client talks to one UniFi API surface (local console or cloud) with an X-API-KEY.
type Client struct {
	base *url.URL
	key  string
	http *http.Client
}

// Options configure a Client.
type Options struct {
	Insecure bool          // skip TLS verification (consoles ship a self-signed cert)
	Timeout  time.Duration // per-request timeout (default 30s)
}

// NewLocal builds a client for the local Integration API at host (e.g. https://192.168.1.1).
func NewLocal(host, key string, opt Options) (*Client, error) {
	if strings.TrimSpace(host) == "" {
		return nil, errs.New(errs.ExitConfig, "CONFIG", "no UniFi host configured",
			"set --host or UNIFI_HOST (e.g. https://192.168.1.1)")
	}
	u, err := normalizeHost(host)
	if err != nil {
		return nil, errs.New(errs.ExitConfig, "CONFIG", "invalid host: "+err.Error(),
			"use a URL or IP like https://192.168.1.1")
	}
	u.Path = strings.TrimRight(u.Path, "/") + IntegrationBase
	return newClient(u, key, opt), nil
}

// NewCloud builds a client for the Site Manager cloud API.
func NewCloud(key string, opt Options) (*Client, error) {
	u, _ := url.Parse(CloudBase)
	return newClient(u, key, opt), nil
}

func newClient(base *url.URL, key string, opt Options) *Client {
	if opt.Timeout == 0 {
		opt.Timeout = 30 * time.Second
	}
	tr := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: opt.Insecure}}
	return &Client{base: base, key: key, http: &http.Client{Transport: tr, Timeout: opt.Timeout}}
}

// errorMessage mirrors the UniFi error envelope (Error Message schema).
type errorMessage struct {
	StatusCode int    `json:"statusCode"`
	StatusName string `json:"statusName"`
	Code       string `json:"code"`
	Message    string `json:"message"`
	RequestID  string `json:"requestId"`
}

// do performs one HTTP request with bounded retries on transient failures (GET only, to avoid
// re-issuing mutations). It returns the raw body and status, or a classified *errs.CLIError.
func (c *Client) do(ctx context.Context, method, path string, query url.Values, body []byte) ([]byte, error) {
	u := *c.base
	u.Path = strings.TrimRight(u.Path, "/") + "/" + strings.TrimLeft(path, "/")
	if query != nil {
		u.RawQuery = query.Encode()
	}

	const maxAttempts = 3
	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		var rdr io.Reader
		if body != nil {
			rdr = bytes.NewReader(body)
		}
		req, err := http.NewRequestWithContext(ctx, method, u.String(), rdr)
		if err != nil {
			return nil, errs.New(errs.ExitGeneric, "INTERNAL", err.Error(), "")
		}
		req.Header.Set("X-API-KEY", c.key)
		req.Header.Set("Accept", "application/json")
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}

		resp, err := c.http.Do(req)
		if err != nil {
			lastErr = errs.New(errs.ExitRetry, "TRANSIENT", netErrMessage(err),
				"the console was unreachable; check --host/network and retry")
			if method == http.MethodGet && attempt < maxAttempts {
				time.Sleep(backoff(attempt))
				continue
			}
			return nil, lastErr
		}
		raw, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return raw, nil
		}
		cerr := classify(resp, raw)
		// Retry 5xx on idempotent GETs only.
		if method == http.MethodGet && resp.StatusCode >= 500 && attempt < maxAttempts {
			lastErr = cerr
			time.Sleep(backoff(attempt))
			continue
		}
		return nil, cerr
	}
	return nil, lastErr
}

func backoff(attempt int) time.Duration {
	return time.Duration(attempt*attempt) * 150 * time.Millisecond
}

func netErrMessage(err error) string {
	msg := err.Error()
	if i := strings.LastIndex(msg, ": "); i >= 0 && len(msg)-i < 60 {
		return msg
	}
	return msg
}

// classify maps an upstream non-2xx response to a structured, exit-coded error.
func classify(resp *http.Response, raw []byte) error {
	var em errorMessage
	_ = json.Unmarshal(raw, &em)
	msg := em.Message
	if msg == "" {
		msg = strings.TrimSpace(string(raw))
	}
	if msg == "" {
		msg = http.StatusText(resp.StatusCode)
	}
	switch resp.StatusCode {
	case http.StatusUnauthorized:
		return errs.New(errs.ExitAuth, "AUTH_REQUIRED", msg, "run `ufi auth login` or set UNIFI_API_KEY")
	case http.StatusForbidden:
		return errs.New(errs.ExitPerm, "PERMISSION_DENIED", msg, "the API key lacks the required scope")
	case http.StatusNotFound:
		return errs.New(errs.ExitNotFound, "NOT_FOUND", msg, "list the resource to find a valid id")
	case http.StatusTooManyRequests:
		ra := resp.Header.Get("Retry-After")
		rem := "rate limited; retry later"
		if ra != "" {
			rem = "rate limited; retry after " + ra + "s"
		}
		ce := errs.New(errs.ExitRate, "RATE_LIMITED", msg, rem)
		return ce
	case http.StatusBadRequest:
		if em.Code == "api.firewall.zone-based-firewall-not-configured" {
			return errs.New(errs.ExitUnsupported, "UNSUPPORTED", msg,
				"enable Zone-Based Firewall on the console (Settings → Security) to use firewall commands")
		}
		rem := "check the request body/arguments"
		if em.Code != "" {
			rem = "upstream code: " + em.Code
		}
		return errs.New(errs.ExitUsage, "BAD_REQUEST", msg, rem)
	default:
		if resp.StatusCode >= 500 {
			return errs.New(errs.ExitRetry, "TRANSIENT", msg, "transient server error; retry")
		}
		return errs.New(errs.ExitGeneric, "API_ERROR", msg, em.Code)
	}
}

// GetObject fetches a single resource and returns it snake_cased.
func (c *Client) GetObject(ctx context.Context, path string) (any, error) {
	raw, err := c.do(ctx, http.MethodGet, path, nil, nil)
	if err != nil {
		return nil, err
	}
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil, errs.New(errs.ExitGeneric, "SCHEMA_DRIFT", "could not parse response: "+err.Error(), "")
	}
	return snakeKeys(v), nil
}

// page is the UniFi list envelope: { offset, limit, count, totalCount, data }.
type page struct {
	Offset     int               `json:"offset"`
	Limit      int               `json:"limit"`
	Count      int               `json:"count"`
	TotalCount int               `json:"totalCount"`
	Data       []json.RawMessage `json:"data"`
}

// ListResult is a normalized, snake_cased page for the CLI list envelope.
type ListResult struct {
	Items      []any
	Count      int
	TotalCount int
	NextCursor any // opaque base64(offset) string, or nil at end-of-results
}

// ListOpts controls listing.
type ListOpts struct {
	Limit  int    // max items to return across pages (CLI --limit); <=0 means one page
	Cursor string // opaque cursor (CLI --cursor); decodes to a start offset
	Filter string // server-side RSQL filter (optional)
}

const pageSize = 200 // request size per upstream call

// List pages the Integration API until it has gathered up to opts.Limit items (or exhausts
// results), returning a snake_cased item slice plus an opaque nextCursor when more remain.
func (c *Client) List(ctx context.Context, path string, opts ListOpts) (*ListResult, error) {
	start, err := decodeCursor(opts.Cursor)
	if err != nil {
		return nil, errs.New(errs.ExitUsage, "USAGE", "invalid --cursor", "use the nextCursor from a prior response")
	}
	want := opts.Limit
	res := &ListResult{Items: []any{}}
	offset := start
	for {
		take := pageSize
		if want > 0 {
			if remaining := want - len(res.Items); remaining < take {
				take = remaining
			}
		}
		if take <= 0 {
			break
		}
		q := url.Values{}
		q.Set("offset", strconv.Itoa(offset))
		q.Set("limit", strconv.Itoa(take))
		if opts.Filter != "" {
			q.Set("filter", opts.Filter)
		}
		raw, err := c.do(ctx, http.MethodGet, path, q, nil)
		if err != nil {
			return nil, err
		}
		var p page
		if err := json.Unmarshal(raw, &p); err != nil {
			return nil, errs.New(errs.ExitGeneric, "SCHEMA_DRIFT", "could not parse list response: "+err.Error(), "")
		}
		res.TotalCount = p.TotalCount
		for _, rm := range p.Data {
			var v any
			if err := json.Unmarshal(rm, &v); err != nil {
				return nil, errs.New(errs.ExitGeneric, "SCHEMA_DRIFT", "could not parse list item: "+err.Error(), "")
			}
			res.Items = append(res.Items, snakeKeys(v))
		}
		offset = p.Offset + p.Count
		exhausted := p.Count == 0 || offset >= p.TotalCount
		if exhausted {
			res.NextCursor = nil
			break
		}
		if want > 0 && len(res.Items) >= want {
			res.NextCursor = encodeCursor(offset)
			break
		}
	}
	res.Count = len(res.Items)
	return res, nil
}

// Send issues a write (POST/PUT/PATCH/DELETE) with an optional JSON body and returns the
// snake_cased response (or nil for empty 2xx bodies).
func (c *Client) Send(ctx context.Context, method, path string, body []byte) (any, error) {
	raw, err := c.do(ctx, method, path, nil, body)
	if err != nil {
		return nil, err
	}
	if len(bytes.TrimSpace(raw)) == 0 {
		return nil, nil
	}
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil, errs.New(errs.ExitGeneric, "SCHEMA_DRIFT", "could not parse response: "+err.Error(), "")
	}
	return snakeKeys(v), nil
}

// Validate performs a cheap authenticated GET /info, returning the application version.
func (c *Client) Validate(ctx context.Context) (string, error) {
	v, err := c.GetObject(ctx, "/info")
	if err != nil {
		return "", err
	}
	if m, ok := v.(map[string]any); ok {
		if s, ok := m["application_version"].(string); ok {
			return s, nil
		}
	}
	return "", nil
}

func normalizeHost(host string) (*url.URL, error) {
	host = strings.TrimSpace(host)
	if !strings.Contains(host, "://") {
		host = "https://" + host
	}
	u, err := url.Parse(host)
	if err != nil {
		return nil, err
	}
	if u.Host == "" {
		return nil, fmt.Errorf("missing host")
	}
	u.Path = strings.TrimRight(u.Path, "/")
	u.RawQuery = ""
	return u, nil
}
