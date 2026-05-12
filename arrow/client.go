// client.go
package arrow

import (
	"fmt"
	"sync"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/valyala/fasthttp"
)

// Config holds the SDK configuration settings.
type Config struct {
	AppID        string // Application ID for API authentication.
	AppSecret    string // Application secret key for API authentication.
	Token        string // Authentication token for API requests.
	BaseURL      string // Base URL of the Arrow API.
	RefreshToken string // Token used to refresh authentication when expired.
	Debug        bool   // Enables verbose SDK debug logs when true.
}

// Client is the main struct for interacting with the Arrow API.
//
// It contains the configuration settings and an HTTP client for making API requests.
type Client struct {
	Config     Config           // Configuration settings for the API client.
	HTTPClient *fasthttp.Client // HTTP client for executing requests.
	mu         sync.RWMutex
}

// NewClient initializes a new SDK client with the provided application credentials.
//
// Parameters:
//   - appID: The application ID used for authentication.
//   - appSecret: The application secret key used for authentication.
//
// Returns:
//   - A pointer to a newly created Client instance.
func NewClient(appID, appSecret string) *Client {
	return &Client{
		Config: Config{
			AppID:     appID,
			AppSecret: appSecret,
			BaseURL:   "https://edge.arrow.trade",
		},
		HTTPClient: &fasthttp.Client{},
	}
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

	// Execute the request using the fasthttp client.
	err := c.HTTPClient.Do(req, resp)
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

	// Execute the request using the fasthttp client.
	err := c.HTTPClient.Do(req, resp)
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
	req.Header.Set("token", c.Config.Token)
	req.Header.SetMethod(method)
	if len(payload) > 0 {
		req.Header.SetContentType("application/json")
		req.SetBody(payload)
	}

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	err := c.HTTPClient.Do(req, resp)
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

// GetRefreshToken gets the refresh token of the user.
//
// This function allows to get the refresh token at runtime which can be used to create new Token.
//
// Returns:
//   - refreshToken: The refresh token.
func (c *Client) GetRefreshToken() string {
	return c.Config.RefreshToken
}
