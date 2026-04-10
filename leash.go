package leash

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
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
}

// New creates a LeashIntegrations client with the default platform URL.
func New(authToken string) *LeashIntegrations {
	return &LeashIntegrations{
		PlatformURL: DefaultPlatformURL,
		AuthToken:   authToken,
		HTTPClient:  http.DefaultClient,
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

// GetConnectURL returns the URL to initiate an OAuth connection flow for
// the given provider. This URL can be used in UI buttons or redirects.
func (l *LeashIntegrations) GetConnectURL(providerID string, returnURL string) string {
	u := fmt.Sprintf("%s/api/integrations/connect/%s", l.PlatformURL, providerID)
	if returnURL != "" {
		u += "?return_url=" + url.QueryEscape(returnURL)
	}
	return u
}
