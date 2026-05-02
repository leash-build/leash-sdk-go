package leash

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
)

// LeashIntegrations is the main client for accessing Leash platform integrations.
//
// Create one with New or NewWithURL, then access provider clients via
// Gmail(), Calendar(), and Drive().
type LeashIntegrations struct {
	// PlatformURL is the base URL of the Leash platform (no trailing slash).
	PlatformURL string
	// AuthToken is the JWT token used for authentication.
	AuthToken string
	// HTTPClient is the HTTP client used for requests. Defaults to http.DefaultClient.
	HTTPClient *http.Client
	// APIKey is an optional API key for service-to-service authentication.
	// When set, it is sent as the X-API-Key header on every request.
	APIKey string

	envCache map[string]string
}

// New creates a LeashIntegrations client with the default platform URL.
func New(authToken string) *LeashIntegrations {
	return &LeashIntegrations{
		PlatformURL: DefaultPlatformURL,
		AuthToken:   authToken,
		HTTPClient:  http.DefaultClient,
		APIKey:      os.Getenv("LEASH_API_KEY"),
	}
}

// NewWithURL creates a LeashIntegrations client with a custom platform URL.
func NewWithURL(authToken, platformURL string) *LeashIntegrations {
	return &LeashIntegrations{
		PlatformURL: strings.TrimRight(platformURL, "/"),
		AuthToken:   authToken,
		HTTPClient:  http.DefaultClient,
	}
}

// Gmail returns a GmailClient for interacting with the Gmail integration.
func (l *LeashIntegrations) Gmail() *GmailClient {
	return &GmailClient{client: l}
}

// Calendar returns a CalendarClient for interacting with the Google Calendar integration.
func (l *LeashIntegrations) Calendar() *CalendarClient {
	return &CalendarClient{client: l}
}

// Drive returns a DriveClient for interacting with the Google Drive integration.
func (l *LeashIntegrations) Drive() *DriveClient {
	return &DriveClient{client: l}
}

// Integration returns a CustomIntegration for the given integration name.
// This is the escape hatch for custom/untyped integrations that don't have
// dedicated provider clients.
func (l *LeashIntegrations) Integration(name string) *CustomIntegration {
	return &CustomIntegration{name: name, client: l}
}

// CustomIntegration provides an untyped client for a custom integration.
// It proxies requests through the Leash platform at
// /api/integrations/custom/{name}.
type CustomIntegration struct {
	name   string
	client *LeashIntegrations
}

// CustomCallRequest is the request body sent to the custom integration proxy.
type CustomCallRequest struct {
	Path    string            `json:"path"`
	Method  string            `json:"method"`
	Body    any               `json:"body,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
}

// Call invokes the custom integration proxy endpoint.
//
// It sends a POST request to /api/integrations/custom/{name} with the given
// path, method, and optional body/headers forwarded to the upstream service.
func (c *CustomIntegration) Call(path string, method string, body any) (json.RawMessage, error) {
	return c.CallWithHeaders(path, method, body, nil)
}

// CallWithHeaders is like Call but also forwards custom headers.
func (c *CustomIntegration) CallWithHeaders(path string, method string, body any, headers map[string]string) (json.RawMessage, error) {
	endpoint := fmt.Sprintf("%s/api/integrations/custom/%s", c.client.PlatformURL, c.name)

	payload := CustomCallRequest{
		Path:    path,
		Method:  method,
		Body:    body,
		Headers: headers,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("leash: failed to marshal request body: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("leash: failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.client.AuthToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.client.AuthToken)
	}
	if c.client.APIKey != "" {
		req.Header.Set("X-API-Key", c.client.APIKey)
	}

	httpClient := c.client.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("leash: request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("leash: failed to read response: %w", err)
	}

	var apiResp apiResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("leash: failed to parse response: %w", err)
	}

	if !apiResp.Success {
		return nil, &Error{
			Message:    apiResp.ErrorMsg,
			Code:       apiResp.Code,
			ConnectURL: apiResp.ConnectURL,
		}
	}

	return apiResp.Data, nil
}

// Call performs a generic integration API call. It sends a POST request to
// the platform proxy at /api/integrations/{provider}/{action} and returns
// the raw JSON data from the response.
//
// This is useful for calling custom or new provider actions that don't yet
// have dedicated methods.
func (l *LeashIntegrations) Call(provider, action string, body any) (json.RawMessage, error) {
	return l.call(provider, action, body)
}

// call is the internal HTTP call method used by all provider clients.
func (l *LeashIntegrations) call(provider, action string, body any) (json.RawMessage, error) {
	endpoint := fmt.Sprintf("%s/api/integrations/%s/%s", l.PlatformURL, provider, action)

	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("leash: failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequest(http.MethodPost, endpoint, reqBody)
	if err != nil {
		return nil, fmt.Errorf("leash: failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if l.AuthToken != "" {
		req.Header.Set("Authorization", "Bearer "+l.AuthToken)
	}
	if l.APIKey != "" {
		req.Header.Set("X-API-Key", l.APIKey)
	}

	httpClient := l.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("leash: request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("leash: failed to read response: %w", err)
	}

	var apiResp apiResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("leash: failed to parse response: %w", err)
	}

	if !apiResp.Success {
		return nil, &Error{
			Message:    apiResp.ErrorMsg,
			Code:       apiResp.Code,
			ConnectURL: apiResp.ConnectURL,
		}
	}

	return apiResp.Data, nil
}

// IsConnected checks whether a given provider is actively connected
// for the current user.
func (l *LeashIntegrations) IsConnected(providerID string) bool {
	connections, err := l.GetConnections()
	if err != nil {
		return false
	}
	for _, c := range connections {
		if c.ProviderID == providerID && c.Status == "active" {
			return true
		}
	}
	return false
}

// GetConnections retrieves the connection status for all providers.
func (l *LeashIntegrations) GetConnections() ([]ConnectionStatus, error) {
	endpoint := fmt.Sprintf("%s/api/integrations/connections", l.PlatformURL)

	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("leash: failed to create request: %w", err)
	}

	if l.AuthToken != "" {
		req.Header.Set("Authorization", "Bearer "+l.AuthToken)
	}
	if l.APIKey != "" {
		req.Header.Set("X-API-Key", l.APIKey)
	}

	httpClient := l.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("leash: request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("leash: failed to read response: %w", err)
	}

	var apiResp apiResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("leash: failed to parse response: %w", err)
	}

	if !apiResp.Success {
		return nil, &Error{
			Message:    apiResp.ErrorMsg,
			Code:       apiResp.Code,
			ConnectURL: apiResp.ConnectURL,
		}
	}

	var connections []ConnectionStatus
	if err := json.Unmarshal(apiResp.Data, &connections); err != nil {
		return nil, fmt.Errorf("leash: failed to parse connections: %w", err)
	}

	return connections, nil
}

// MCP calls any MCP server tool directly via the Leash platform.
//
// It sends a POST request to /api/mcp/run with the given npm package name,
// tool name, and optional arguments, then returns the raw JSON data.
func (l *LeashIntegrations) MCP(npmPackage, tool string, args any) (json.RawMessage, error) {
	endpoint := fmt.Sprintf("%s/api/mcp/run", l.PlatformURL)

	payload := map[string]any{
		"package": npmPackage,
		"tool":    tool,
	}
	if args != nil {
		payload["args"] = args
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("leash: failed to marshal request body: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("leash: failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if l.AuthToken != "" {
		req.Header.Set("Authorization", "Bearer "+l.AuthToken)
	}
	if l.APIKey != "" {
		req.Header.Set("X-API-Key", l.APIKey)
	}

	httpClient := l.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("leash: request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("leash: failed to read response: %w", err)
	}

	var apiResp apiResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("leash: failed to parse response: %w", err)
	}

	if !apiResp.Success {
		return nil, &Error{
			Message:    apiResp.ErrorMsg,
			Code:       apiResp.Code,
			ConnectURL: apiResp.ConnectURL,
		}
	}

	return apiResp.Data, nil
}

// GetEnv fetches all environment variables from the Leash platform.
// The result is cached after the first call.
func (l *LeashIntegrations) GetEnv() (map[string]string, error) {
	if l.envCache != nil {
		return l.envCache, nil
	}

	endpoint := fmt.Sprintf("%s/api/apps/env", l.PlatformURL)

	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("leash: failed to create request: %w", err)
	}

	if l.AuthToken != "" {
		req.Header.Set("Authorization", "Bearer "+l.AuthToken)
	}
	if l.APIKey != "" {
		req.Header.Set("X-API-Key", l.APIKey)
	}

	httpClient := l.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("leash: request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("leash: failed to read response: %w", err)
	}

	var apiResp apiResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("leash: failed to parse response: %w", err)
	}

	if !apiResp.Success {
		return nil, &Error{
			Message:    apiResp.ErrorMsg,
			Code:       apiResp.Code,
			ConnectURL: apiResp.ConnectURL,
		}
	}

	var envMap map[string]string
	if err := json.Unmarshal(apiResp.Data, &envMap); err != nil {
		return nil, fmt.Errorf("leash: failed to parse env data: %w", err)
	}

	l.envCache = envMap
	return l.envCache, nil
}

// GetEnvKey fetches a single environment variable by key.
// Returns the value and an error. If the key is not found, the value is empty.
func (l *LeashIntegrations) GetEnvKey(key string) (string, error) {
	envMap, err := l.GetEnv()
	if err != nil {
		return "", err
	}
	return envMap[key], nil
}

// GetConnectURL returns the URL to initiate an OAuth connection flow for
// the given provider. This URL can be used in UI buttons or redirects.
func (l *LeashIntegrations) GetConnectURL(providerID string, returnURL string) string {
	u := fmt.Sprintf("%s/api/integrations/connect/%s", l.PlatformURL, providerID)
	if returnURL != "" {
		u += "?return_url=" + url.QueryEscape(returnURL)
	}
	return u
}

// tokenResponse is the shape of the data field in a successful response from
// POST /api/integrations/token.
type tokenResponse struct {
	AccessToken string `json:"accessToken"`
	Provider    string `json:"provider"`
}

// GetAccessToken returns the user's current access token for a provider —
// built-in or org-registered (LEA-142). Use this to call third-party APIs
// directly without proxying every request through Leash. Refresh-on-expiry
// happens transparently on the platform side.
//
// Returns a *Error with Code="not_connected" when the user hasn't completed
// the OAuth flow for this provider.
func (l *LeashIntegrations) GetAccessToken(provider string) (string, error) {
	endpoint := fmt.Sprintf("%s/api/integrations/token", l.PlatformURL)

	payload := map[string]string{"provider": provider}
	data, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("leash: failed to marshal request body: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("leash: failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if l.AuthToken != "" {
		req.Header.Set("Authorization", "Bearer "+l.AuthToken)
	}
	if l.APIKey != "" {
		req.Header.Set("X-API-Key", l.APIKey)
	}

	httpClient := l.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("leash: request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("leash: failed to read response: %w", err)
	}

	var apiResp apiResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return "", fmt.Errorf("leash: failed to parse response: %w", err)
	}

	if !apiResp.Success {
		return "", &Error{
			Message:    apiResp.ErrorMsg,
			Code:       apiResp.Code,
			ConnectURL: apiResp.ConnectURL,
		}
	}

	var token tokenResponse
	if err := json.Unmarshal(apiResp.Data, &token); err != nil {
		return "", fmt.Errorf("leash: failed to parse token data: %w", err)
	}

	return token.AccessToken, nil
}

// GetCustomMcpConfig returns the resolved config for a customer-registered
// MCP server (LEA-143). The returned config contains the customer's MCP URL
// plus auth headers (e.g. "Authorization: Bearer …" for bearer-auth servers)
// — feed it directly into your MCP client. Leash isn't on the MCP request
// path.
//
// Returns a *Error with Code="unknown_mcp_server" when the slug isn't
// registered for the caller's org.
func (l *LeashIntegrations) GetCustomMcpConfig(slug string) (*CustomMcpServerConfig, error) {
	endpoint := fmt.Sprintf("%s/api/integrations/mcp-config/%s", l.PlatformURL, url.PathEscape(slug))

	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("leash: failed to create request: %w", err)
	}

	if l.AuthToken != "" {
		req.Header.Set("Authorization", "Bearer "+l.AuthToken)
	}
	if l.APIKey != "" {
		req.Header.Set("X-API-Key", l.APIKey)
	}

	httpClient := l.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("leash: request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("leash: failed to read response: %w", err)
	}

	var apiResp apiResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("leash: failed to parse response: %w", err)
	}

	if !apiResp.Success {
		return nil, &Error{
			Message:    apiResp.ErrorMsg,
			Code:       apiResp.Code,
			ConnectURL: apiResp.ConnectURL,
		}
	}

	var config CustomMcpServerConfig
	if err := json.Unmarshal(apiResp.Data, &config); err != nil {
		return nil, fmt.Errorf("leash: failed to parse mcp config: %w", err)
	}

	return &config, nil
}
