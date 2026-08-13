package invapi

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	DefaultBaseURLCOM = "https://invapi.hostkey.com/"
	DefaultBaseURLRU  = "https://invapi.hostkey.ru/"
	defaultUserAgent  = "terraform-provider-hostkey/dev"
)

type Config struct {
	BaseURL     string
	HTTPClient  *http.Client
	MaxRetries  int
	HTTPTimeout time.Duration
	UserAgent   string
}

type Client struct {
	baseURL    string
	httpClient *http.Client
	maxRetries int
	userAgent  string
	auth       *TokenManager
}

func NewClient(cfg Config, auth *TokenManager) (*Client, error) {
	base := strings.TrimSuffix(cfg.BaseURL, "/") + "/"
	if base == "/" {
		base = DefaultBaseURLCOM
	}

	httpClient := cfg.HTTPClient
	if httpClient == nil {
		timeout := cfg.HTTPTimeout
		if timeout <= 0 {
			timeout = 60 * time.Second
		}
		httpClient = defaultHTTPClient(timeout)
	}

	retries := cfg.MaxRetries
	if retries <= 0 {
		retries = 3
	}

	ua := strings.TrimSpace(cfg.UserAgent)
	if ua == "" {
		ua = defaultUserAgent
	}

	return &Client{
		baseURL:    base,
		httpClient: httpClient,
		maxRetries: retries,
		userAgent:  ua,
		auth:       auth,
	}, nil
}

func defaultHTTPClient(timeout time.Duration) *http.Client {
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   15 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   15 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	return &http.Client{Timeout: timeout, Transport: transport}
}

func BaseURLForRegion(region string) string {
	switch strings.ToUpper(strings.TrimSpace(region)) {
	case "RU":
		return DefaultBaseURLRU
	default:
		return DefaultBaseURLCOM
	}
}

func (c *Client) SetAuth(auth *TokenManager) {
	c.auth = auth
}

func (c *Client) BaseURL() string {
	return c.baseURL
}

func (c *Client) moduleURL(module string) string {
	return c.baseURL + module + ".php"
}

func (c *Client) PostForm(ctx context.Context, module string, params url.Values) ([]byte, error) {
	return c.postForm(ctx, module, params, true)
}

func (c *Client) PostFormWithoutAuth(ctx context.Context, module string, params url.Values) ([]byte, error) {
	return c.postForm(ctx, module, params, false)
}

func (c *Client) postForm(ctx context.Context, module string, params url.Values, withAuth bool) ([]byte, error) {
	var lastErr error
	authRetried := false

	for attempt := 0; attempt < c.maxRetries; attempt++ {
		body, status, err := c.doPostOnce(ctx, module, params, withAuth)
		if err == nil {
			if apiErr := decodeAPIError(body); apiErr != nil {
				if withAuth && !authRetried && isAuthFailure(status, apiErr) {
					authRetried = true
					if c.auth != nil {
						c.auth.Invalidate()
					}
					lastErr = wrapHTTPError(module, status, apiErr)
					continue
				}
				return nil, wrapHTTPError(module, status, apiErr)
			}
			return body, nil
		}

		lastErr = wrapHTTPError(module, status, err)

		if withAuth && !authRetried && isAuthFailure(status, err) {
			authRetried = true
			if c.auth != nil {
				c.auth.Invalidate()
			}
			continue
		}

		if !retryableStatus(status) && status != 0 {
			return nil, lastErr
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(backoff(attempt)):
		}
	}

	return nil, fmt.Errorf("invapi %s: request failed after retries: %w", module, lastErr)
}

func (c *Client) doPostOnce(ctx context.Context, module string, params url.Values, withAuth bool) ([]byte, int, error) {
	reqParams := cloneValues(params)
	if withAuth && reqParams.Get("token") == "" && c.auth != nil {
		token, err := c.auth.Token(ctx)
		if err != nil {
			return nil, 0, err
		}
		reqParams.Set("token", token)
	}

	encoded := reqParams.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.moduleURL(module), strings.NewReader(encoded))
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", c.userAgent)
	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(strings.NewReader(encoded)), nil
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, resp.StatusCode, readErr
	}

	if resp.StatusCode >= 400 {
		return body, resp.StatusCode, fmt.Errorf("HTTP %d: %s", resp.StatusCode, truncate(body, 512))
	}

	return body, resp.StatusCode, nil
}

func cloneValues(in url.Values) url.Values {
	out := make(url.Values, len(in))
	for k, vs := range in {
		cp := make([]string, len(vs))
		copy(cp, vs)
		out[k] = cp
	}
	return out
}

func retryableStatus(status int) bool {
	switch status {
	case 0, 429, 502, 503, 504:
		return true
	default:
		return status >= 500
	}
}

func isAuthFailure(status int, err error) bool {
	if status == http.StatusUnauthorized {
		return true
	}
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	for _, needle := range []string{
		"unauthorized",
		"invalid token",
		"token expired",
		"token is invalid",
		"not authorized",
		"authentication failed",
	} {
		if strings.Contains(msg, needle) {
			return true
		}
	}
	return false
}

func backoff(attempt int) time.Duration {
	d := time.Duration(attempt+1) * 300 * time.Millisecond
	if d > 3*time.Second {
		d = 3 * time.Second
	}
	return d
}

func wrapHTTPError(module string, status int, err error) error {
	if err == nil {
		return nil
	}
	if status > 0 {
		return fmt.Errorf("invapi %s (HTTP %d): %w", module, status, err)
	}
	return fmt.Errorf("invapi %s: %w", module, err)
}
