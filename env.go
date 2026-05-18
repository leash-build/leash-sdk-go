package leash

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"
)

// DefaultEnvTTL is the in-memory cache window for env-var lookups.
//
// Matches the TS / Python default of 60 seconds.
const DefaultEnvTTL = 60 * time.Second

// EnvNamespace is the [Client.Env] surface — runtime env-var fetcher with a
// per-instance TTL cache.
type EnvNamespace struct {
	platformURL string
	apiKey      string
	httpClient  *http.Client

	mu    sync.Mutex
	cache map[string]envCacheEntry
}

type envCacheEntry struct {
	// value is the resolved value, or nil for "not declared / not found"
	// (HTTP 404). Caching the nil avoids re-querying for repeatedly-missing
	// keys within the TTL window.
	value     *string
	expiresAt time.Time
}

func newEnvNamespace(platformURL, apiKey string, httpClient *http.Client) *EnvNamespace {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &EnvNamespace{
		platformURL: platformURL,
		apiKey:      apiKey,
		httpClient:  httpClient,
		cache:       make(map[string]envCacheEntry),
	}
}

// Get resolves a single env-var by name.
//
// Returns (nil, nil) when the platform reports the key as not declared /
// not found (HTTP 404) — so callers can branch with `if val == nil` rather
// than catching a specific error. All other failures return a [LeashError].
func (e *EnvNamespace) Get(ctx context.Context, key string, opts ...EnvOption) (*string, error) {
	cfg := envCallConfig{ttl: DefaultEnvTTL}
	for _, o := range opts {
		o(&cfg)
	}

	now := time.Now()
	if !cfg.fresh {
		e.mu.Lock()
		entry, ok := e.cache[key]
		e.mu.Unlock()
		if ok && entry.expiresAt.After(now) {
			return entry.value, nil
		}
	}

	val, err := e.fetch(ctx, key)
	if err != nil {
		return nil, err
	}

	ttl := cfg.ttl
	if ttl <= 0 {
		ttl = DefaultEnvTTL
	}
	e.mu.Lock()
	e.cache[key] = envCacheEntry{value: val, expiresAt: now.Add(ttl)}
	e.mu.Unlock()
	return val, nil
}

// GetMany resolves several env-vars in one call, sharing the per-instance
// cache. Each entry is `*string` so callers can distinguish "missing" (nil)
// from "empty string".
//
// If any key fails (auth, plan, network), the whole call returns that error
// — partial results are not surfaced.
func (e *EnvNamespace) GetMany(ctx context.Context, keys []string) (map[string]*string, error) {
	out := make(map[string]*string, len(keys))
	for _, k := range keys {
		v, err := e.Get(ctx, k)
		if err != nil {
			return nil, err
		}
		out[k] = v
	}
	return out, nil
}

// fetch issues a single GET /api/apps/me/secrets/{key} call.
func (e *EnvNamespace) fetch(ctx context.Context, key string) (*string, error) {
	if e.apiKey == "" {
		return nil, &LeashError{
			Code:    CodeNoAPIKey,
			Message: "LEASH_API_KEY is required to call Env.Get.",
			Action:  "Set LEASH_API_KEY in your environment or pass leash.WithAPIKey(...) when constructing the client.",
			SeeAlso: "https://leash.build/dashboard/organization",
		}
	}

	endpoint := fmt.Sprintf("%s/api/apps/me/secrets/%s", e.platformURL, url.PathEscape(key))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, &LeashError{
			Code:    CodeNetworkError,
			Message: "Failed to build env-fetch request.",
			Cause:   err,
		}
	}
	req.Header.Set("Authorization", "Bearer "+e.apiKey)

	resp, err := e.httpClient.Do(req)
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

	body, _ := io.ReadAll(resp.Body)

	switch resp.StatusCode {
	case http.StatusBadRequest:
		return nil, &LeashError{
			Code:    CodeInvalidKey,
			Message: fmt.Sprintf("Invalid env-var key: %q.", key),
			Action:  "Env-var names must match /^[A-Za-z_][A-Za-z0-9_]*$/ and be no longer than 100 characters.",
			SeeAlso: "https://leash.build/docs/sdk",
			Status:  resp.StatusCode,
		}
	case http.StatusUnauthorized:
		return nil, &LeashError{
			Code:    CodeUnauthorized,
			Message: "Missing or invalid LEASH_API_KEY.",
			Action:  "Mint a fresh API key at /dashboard/organization.",
			SeeAlso: "https://leash.build/dashboard/organization",
			Status:  resp.StatusCode,
		}
	case http.StatusPaymentRequired:
		var parsed map[string]any
		_ = json.Unmarshal(body, &parsed)
		required, _ := parsed["requiredPlan"].(string)
		suffix := ""
		if required != "" {
			suffix = " (requiredPlan: " + required + ")"
		}
		return nil, &LeashError{
			Code:    CodeUpgradeRequired,
			Message: fmt.Sprintf("Env.Get requires the Growth plan or above%s.", suffix),
			Action:  "Upgrade at https://leash.build/dashboard/billing.",
			SeeAlso: "https://leash.build/dashboard/billing",
			Status:  resp.StatusCode,
		}
	case http.StatusNotFound:
		// Adapted behaviour: return (nil, nil) so Go callers can branch on
		// missing keys naturally — matches the Python `Optional[str]` pattern.
		return nil, nil
	case http.StatusBadGateway:
		var parsed map[string]any
		_ = json.Unmarshal(body, &parsed)
		msg, _ := parsed["error"].(string)
		if msg == "" {
			msg = "Secret source resync failed on the platform side."
		}
		return nil, &LeashError{
			Code:    CodeSourceResyncFailed,
			Message: msg,
			Action:  "Check your secret source configuration in the Leash dashboard.",
			SeeAlso: "https://leash.build/dashboard",
			Status:  resp.StatusCode,
		}
	}

	if resp.StatusCode >= 400 {
		return nil, &LeashError{
			Code:    CodeEnvFetchError,
			Message: fmt.Sprintf("Unexpected response from platform: HTTP %d", resp.StatusCode),
			Action:  "Check the Leash platform status and your configuration.",
			SeeAlso: "https://leash.build/docs/sdk",
			Status:  resp.StatusCode,
		}
	}

	var parsed map[string]any
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, &LeashError{
			Code:    CodeEnvFetchError,
			Message: fmt.Sprintf("Platform returned an unparseable response for key %q.", key),
			Action:  "Check the Leash platform status and your configuration.",
			SeeAlso: "https://leash.build/docs/sdk",
			Status:  resp.StatusCode,
			Cause:   err,
		}
	}
	v, ok := parsed["value"].(string)
	if !ok {
		return nil, &LeashError{
			Code:    CodeEnvFetchError,
			Message: fmt.Sprintf("Platform returned an unexpected response shape for key %q.", key),
			Action:  "Check the Leash platform status and your configuration.",
			SeeAlso: "https://leash.build/docs/sdk",
			Status:  resp.StatusCode,
		}
	}
	return &v, nil
}
