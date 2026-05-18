package leash

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Transport is the shared HTTP carrier the integrations namespace and the
// env namespace dispatch through.
//
// Exported so consumers can inject one into the [integrations.Namespace] —
// useful for vendoring the wire shape into another tool. Most callers
// shouldn't need to touch it.
type Transport struct {
	PlatformURL string
	APIKey      string
	CookieValue string
	HTTPClient  *http.Client
}

// newTransport snapshots the resolved auth + platform URL into a Transport.
func newTransport(cfg *clientConfig) *Transport {
	return &Transport{
		PlatformURL: strings.TrimRight(cfg.platformURL, "/"),
		APIKey:      cfg.apiKey,
		CookieValue: cfg.cookieValue,
		HTTPClient:  cfg.httpClient,
	}
}

// integrationsCall POSTs to /api/integrations/{provider}/{action} and returns
// the unwrapped response data. Mirrors the TS `_call` / Python `_Transport.call`.
//
// Critical: only X-API-Key + Cookie are forwarded. Authorization: Bearer is
// intentionally not sent — the platform's verifyToken can reject a user JWT
// before the API-key check runs (LEA-262 era review).
func (t *Transport) integrationsCall(ctx context.Context, provider, action string, body any) (json.RawMessage, error) {
	url := fmt.Sprintf("%s/api/integrations/%s/%s", t.PlatformURL, provider, action)
	docsURL := fmt.Sprintf("https://leash.build/docs/integrations/%s", provider)
	return t.post(ctx, url, body, docsURL)
}

// post is the shared POST helper — JSON in, JSON out, structured errors.
func (t *Transport) post(ctx context.Context, url string, body any, docsURL string) (json.RawMessage, error) {
	var reader io.Reader
	if body == nil {
		reader = bytes.NewReader([]byte("{}"))
	} else {
		buf, err := json.Marshal(body)
		if err != nil {
			return nil, &LeashError{
				Code:    CodeIntegrationError,
				Message: "Failed to marshal request body.",
				Cause:   err,
			}
		}
		reader = bytes.NewReader(buf)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, reader)
	if err != nil {
		return nil, &LeashError{
			Code:    CodeNetworkError,
			Message: "Failed to build platform request.",
			Cause:   err,
		}
	}
	req.Header.Set("Content-Type", "application/json")
	if t.APIKey != "" {
		req.Header.Set("X-API-Key", t.APIKey)
	}
	if t.CookieValue != "" {
		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", CookieName, t.CookieValue))
	}

	resp, err := t.client().Do(req)
	if err != nil {
		return nil, &LeashError{
			Code:    CodeNetworkError,
			Message: err.Error(),
			Action:  "Check your network connection and that the Leash platform is reachable.",
			SeeAlso: "https://leash.build/docs/sdk",
			Cause:   err,
		}
	}
	defer resp.Body.Close()

	rawBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, &LeashError{
			Code:    CodeIntegrationError,
			Message: "Failed to read platform response.",
			Status:  resp.StatusCode,
			Cause:   err,
		}
	}

	if resp.StatusCode >= 400 {
		return nil, mapIntegrationStatus(resp.StatusCode, rawBody, docsURL)
	}

	// Success envelope: { success, data } | { data } | raw shape.
	var envelope struct {
		Success *bool           `json:"success"`
		Data    json.RawMessage `json:"data"`
		Error   string          `json:"error"`
		Code    string          `json:"code"`
	}
	if len(rawBody) > 0 {
		if err := json.Unmarshal(rawBody, &envelope); err != nil {
			// Body wasn't an object — return it raw so callers that expect a
			// bare array (e.g. Linear list_teams) can still parse it.
			return rawBody, nil
		}
	}
	if envelope.Success != nil && !*envelope.Success {
		code := ErrorCode(envelope.Code)
		if code == "" {
			code = CodeIntegrationError
		}
		msg := envelope.Error
		if msg == "" {
			msg = "Integration error"
		}
		return nil, &LeashError{
			Code:    code,
			Message: msg,
			Action:  "Check your integration configuration and try again.",
			SeeAlso: docsURL,
			Status:  resp.StatusCode,
		}
	}
	if envelope.Data != nil {
		return envelope.Data, nil
	}
	return rawBody, nil
}

// mapIntegrationStatus translates a non-2xx HTTP status from an integration
// call into the matching [LeashError]. Mirrors the TS `_post` error block and
// the Python `_raise_for_status`.
func mapIntegrationStatus(status int, body []byte, docsURL string) error {
	message := fmt.Sprintf("HTTP %d", status)
	var parsed map[string]any
	if len(body) > 0 {
		_ = json.Unmarshal(body, &parsed) // best-effort
		if errMsg, ok := parsed["error"].(string); ok && errMsg != "" {
			message = errMsg
		}
	}

	switch status {
	case http.StatusUnauthorized:
		return &LeashError{
			Code:    CodeUnauthorized,
			Message: message,
			Action:  "Ensure the leash-auth cookie is present, or open your app from the Leash dashboard to get a valid session.",
			SeeAlso: "https://leash.build/docs/sdk",
			Status:  status,
		}
	case http.StatusPaymentRequired:
		msg := "This feature requires a higher plan."
		if m, ok := parsed["message"].(string); ok && m != "" {
			msg = m
		}
		return &LeashError{
			Code:    CodeUpgradeRequired,
			Message: msg,
			Action:  "Upgrade your plan at https://leash.build/dashboard/billing.",
			SeeAlso: "https://leash.build/pricing",
			Status:  status,
		}
	case http.StatusForbidden:
		return &LeashError{
			Code:    CodeIntegrationNotEnabled,
			Message: message,
			Action:  "Connect the integration at /dashboard/integrations and make sure this app is on the allow-list.",
			SeeAlso: "https://leash.build/dashboard/integrations",
			Status:  status,
		}
	}
	return &LeashError{
		Code:    CodeIntegrationError,
		Message: message,
		Action:  "Check your integration configuration and try again — the upstream provider returned an error.",
		SeeAlso: docsURL,
		Status:  status,
	}
}

// client returns the resolved *http.Client (defaulting to http.DefaultClient).
func (t *Transport) client() *http.Client {
	if t.HTTPClient != nil {
		return t.HTTPClient
	}
	return http.DefaultClient
}
