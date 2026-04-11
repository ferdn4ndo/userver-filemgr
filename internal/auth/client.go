package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/ferdn4ndo/userver-filemgr/lib"
)

// MeResponse is a subset of uServer-Auth /auth/me JSON.
type MeResponse struct {
	UUID       string          `json:"uuid"`
	Username   string          `json:"username"`
	SystemName string          `json:"system_name"`
	IsAdmin    bool            `json:"is_admin"`
	Message    string          `json:"message"`
	Token      json.RawMessage `json:"token"`
}

type tokenTimes struct {
	IssuedAt  string `json:"issued_at"`
	ExpiresAt string `json:"expires_at"`
}

// Client calls uServer-Auth.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient builds an HTTP client for Auth. Zero timeout defaults to 15s.
func NewClient(baseURL string, timeout time.Duration) *Client {
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	tr := lib.NewPooledTransport()
	if baseURL == "" {
		return &Client{httpClient: &http.Client{Timeout: timeout, Transport: tr}}
	}
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Timeout:   timeout,
			Transport: tr,
		},
	}
}

// ValidateBearer calls GET {base}/auth/me with Authorization: Bearer <token>.
func (c *Client) ValidateBearer(ctx context.Context, bearer string) (*MeResponse, *tokenTimes, error) {
	if c.baseURL == "" {
		return nil, nil, fmt.Errorf("USERVER_AUTH_HOST or AUTH_HOST is not set")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/auth/me", nil)
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("Authorization", "Bearer "+bearer)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("auth/me: status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var me MeResponse
	if err := json.Unmarshal(body, &me); err != nil {
		return nil, nil, err
	}
	if me.Message != "" && me.UUID == "" {
		return nil, nil, fmt.Errorf("%s", me.Message)
	}
	if me.UUID == "" || me.Username == "" {
		return nil, nil, fmt.Errorf("invalid auth/me payload")
	}
	var tt tokenTimes
	if len(me.Token) > 0 && string(me.Token) != "null" {
		_ = json.Unmarshal(me.Token, &tt)
	}
	return &me, &tt, nil
}
