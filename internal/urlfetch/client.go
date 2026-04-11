package urlfetch

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/ferdn4ndo/userver-filemgr/lib"
)

// Client is an SSRF-hardened HTTP client for outbound URL fetches (e.g. upload-from-url).
type Client struct {
	env  lib.Env
	http *http.Client
}

// NewClient builds a client from env (timeouts, redirect cap, TLS, DNS pinning to public IPs in production).
func NewClient(env lib.Env) *Client {
	allowHTTP := env.URLFetchAllowHTTP
	allowLoopback := !env.IsProduction()
	maxRedir := env.URLFetchMaxRedirects
	if maxRedir <= 0 {
		maxRedir = 8
	}
	return &Client{
		env: env,
		http: &http.Client{
			Timeout:   env.URLFetchTimeout,
			Transport: newSecureTransport(env),
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= maxRedir {
					return fmt.Errorf("urlfetch: too many redirects")
				}
				return validateFetchURL(req.URL, allowHTTP, allowLoopback)
			},
		},
	}
}

// Get performs a GET after validating the URL. The caller must close resp.Body.
func (c *Client) Get(ctx context.Context, rawURL string) (*http.Response, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("urlfetch: parse url: %w", err)
	}
	if err := validateFetchURL(u, c.env.URLFetchAllowHTTP, !c.env.IsProduction()); err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "userver-filemgr/1.0")
	return c.http.Do(req)
}

// MaxBytes returns the configured upper bound for response bodies.
func (c *Client) MaxBytes() int64 { return c.env.URLFetchMaxBytes }
