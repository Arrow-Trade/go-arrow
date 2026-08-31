// client.go
package arrow

import (
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/valyala/fasthttp"
)

// DefaultHTTPTimeout is applied to every REST call when the caller does not set
// a timeout. fasthttp.Client.Do has no deadline unless ReadTimeout/WriteTimeout
// are set; without this a stalled connection can block forever.
const DefaultHTTPTimeout = 10 * time.Second

// Config holds the SDK configuration settings.
type Config struct {
	AppID     string        // Application ID for API authentication.
	AppSecret string        // Application secret key for API authentication.
	Token     string        // Authentication token for API requests.
	BaseURL   string        // Base URL of the Arrow API.
	Debug     bool          // Enables verbose SDK debug logs when true.
	Timeout   time.Duration // Per-request REST timeout. Zero means DefaultHTTPTimeout.
}

// Client is the main struct for interacting with the Arrow API.
//
// It contains the configuration settings and an HTTP client for making API requests.
// Configure request bounds with SetHTTPTimeout before the first call, or pass a
// timeout to NewClientWithTimeout. Prefer those APIs over mutating HTTPClient
// fields directly.
type Client struct {
	Config     Config           // Configuration settings for the API client.
	HTTPClient *fasthttp.Client // HTTP client for executing requests.
	mu         sync.RWMutex
}

// NewClient initializes a new SDK client with DefaultHTTPTimeout (10s, same as py-arrow) on every REST call.
func NewClient(appID, appSecret string) *Client {
	return NewClientWithTimeout(appID, appSecret, DefaultHTTPTimeout)
}

// NewClientWithTimeout is NewClient with an explicit per-request timeout.
// A non-positive timeout falls back to DefaultHTTPTimeout.
func NewClientWithTimeout(appID, appSecret string, timeout time.Duration) *Client {
	if timeout <= 0 {
		timeout = DefaultHTTPTimeout
	}
	return &Client{
		Config: Config{
			AppID:     appID,
			AppSecret: appSecret,
			BaseURL:   "https://edge.arrow.trade",
			Timeout:   timeout,
		},
		HTTPClient: newHTTPClient(timeout),
	}
}

func newHTTPClient(timeout time.Duration) *fasthttp.Client {
	return &fasthttp.Client{
		ReadTimeout:  timeout,
		WriteTimeout: timeout,
	}
}

// SetHTTPTimeout sets the per-request REST deadline used by DoTimeout.
// Call this before the first request so fasthttp also copies ReadTimeout/
// WriteTimeout onto the per-host client. A non-positive value is ignored.
func (c *Client) SetHTTPTimeout(timeout time.Duration) {
	if timeout <= 0 {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Config.Timeout = timeout
	if c.HTTPClient != nil {
		c.HTTPClient.ReadTimeout = timeout
		c.HTTPClient.WriteTimeout = timeout
	}
}

// HTTPTimeout returns the configured per-request REST timeout.
func (c *Client) HTTPTimeout() time.Duration {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.Config.Timeout > 0 {
		return c.Config.Timeout
	}
	return DefaultHTTPTimeout
}

func (c *Client) do(req *fasthttp.Request, resp *fasthttp.Response) error {
	return c.HTTPClient.DoTimeout(req, resp, c.HTTPTimeout())
}

// request sends an HTTP API request to the Arrow server and retrieves the response.
//
// This function constructs an HTTP request with the required authentication headers
// and executes it using the `fasthttp` client.
//
// Parameters:
//   - endpoint: The API endpoint (relative to BaseURL) to send the request to.
//   - method: The HTTP method ("GET" or "POST").
//   - payload: The request body (for POST requests).
//
// Returns:
//   - A byte slice containing the response body if successful.
//   - An error if the request fails.
func (c *Client) request(endpoint string, method string, payload []byte) ([]byte, error) {
	url := c.Config.BaseURL + endpoint
	c.debugf("Making request", func(e *zerolog.Event) {
		e.Str("url", url).Str("method", method)
	})

	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)
	req.SetRequestURI(url)
	req.Header.Set("appId", c.Config.AppID)
	req.Header.Set("token", c.Config.Token)
	req.Header.SetMethod(method)
	if len(payload) > 0 {
		req.Header.SetContentType("application/json")
		req.SetBody(payload)
	}

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	err := c.do(req, resp)
	if err != nil {
		log.Error().Err(err).Msg("API request failed")
		return nil, err
	}
	if resp.StatusCode() >= fasthttp.StatusBadRequest {
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode(), string(resp.Body()))
	}

	return resp.Body(), nil
}

// rawRequest sends an HTTP request to a fully specified URL and retrieves the response.
//
// Unlike `request()`, this function allows specifying an absolute URL rather than an endpoint.
//
// Parameters:
//   - url: The full API URL to send the request to.
//   - method: The HTTP method ("GET" or "POST").
//   - payload: The request body (for POST requests).
//
// Returns:
//   - A byte slice containing the response body if successful.
//   - An error if the request fails.
func (c *Client) rawRequest(url string, method string, payload []byte) ([]byte, error) {
	c.debugf("Making raw request", func(e *zerolog.Event) {
		e.Str("url", url).Str("method", method)
	})

	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)
	req.SetRequestURI(url)
	req.Header.SetMethod(method)
	if len(payload) > 0 {
		req.Header.SetContentType("application/json")
		req.SetBody(payload)
	}

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	err := c.do(req, resp)
	if err != nil {
		log.Error().Err(err).Msg("API request failed")
		return nil, err
	}
	if resp.StatusCode() >= fasthttp.StatusBadRequest {
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode(), string(resp.Body()))
	}

	return resp.Body(), nil
}

// rawRequestAuth is like rawRequest but sets appId and token headers (required by historical-api.arrow.trade).
func (c *Client) rawRequestAuth(fullURL string, method string, payload []byte) ([]byte, error) {
	c.debugf("Making raw request (auth)", func(e *zerolog.Event) {
		e.Str("url", fullURL).Str("method", method)
	})

	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)
	req.SetRequestURI(fullURL)
	req.Header.Set("appId", c.Config.AppID)
	req.Header.Set("appID", c.Config.AppID)
	req.Header.Set("token", c.Config.Token)
	req.Header.SetMethod(method)
	if len(payload) > 0 {
		req.Header.SetContentType("application/json")
		req.SetBody(payload)
	}

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	err := c.do(req, resp)
	if err != nil {
		log.Error().Err(err).Msg("API request failed")
		return nil, err
	}
	if resp.StatusCode() >= fasthttp.StatusBadRequest {
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode(), string(resp.Body()))
	}

	return resp.Body(), nil
}

// SetToken updates the authentication token dynamically.
//
// This function allows updating the API token at runtime without needing to recreate the client.
//
// Parameters:
//   - token: The new authentication token.
func (c *Client) SetToken(token string) {
	c.Config.Token = token
}

// SetDebug enables or disables verbose SDK logging.
func (c *Client) SetDebug(enabled bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Config.Debug = enabled
}

// IsDebug returns whether verbose SDK logging is enabled.
func (c *Client) IsDebug() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Config.Debug
}

func (c *Client) debugf(msg string, addFields func(*zerolog.Event)) {
	if !c.IsDebug() {
		return
	}
	e := log.Debug()
	if addFields != nil {
		addFields(e)
	}
	e.Msg(msg)
}

// GetToken retrieves the current authentication token.
//
// This function returns the current API token used for authentication.
//
// Returns:
//   - The current authentication token.
func (c *Client) GetToken() string {
	return c.Config.Token
}
