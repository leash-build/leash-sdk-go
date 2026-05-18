package leash

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strings"

	"github.com/leash-build/leash-sdk-go/integrations"
)

// DefaultPlatformURL is the default Leash platform base URL.
const DefaultPlatformURL = "https://leash.build"

// Client is the unified Leash SDK entry point — auth, env, and integrations
// in a single struct.
//
// Construct one from any *http.Request:
//
//	client, err := leash.New(r)
//	if err != nil { ... }
//	user, err := client.Auth().User(ctx)
//	val, err  := client.Env().Get(ctx, "OPENAI_API_KEY")
//	msgs, err := client.Integrations().Gmail().ListMessages(ctx, nil)
//
// Or for server-to-server (no inbound request):
//
//	client, err := leash.NewFromAPIKey(os.Getenv("LEASH_API_KEY"))
type Client struct {
	transport    *Transport
	auth         *AuthNamespace
	env          *EnvNamespace
	integrations *integrations.Namespace
}

// New constructs a [Client] from an inbound HTTP request.
//
// Auth precedence (highest first), matching the TS + Python surface:
//
//  1. LEASH_API_KEY env var (or [WithAPIKey] option) — server-to-server key
//  2. Authorization: Bearer <jwt> header — captured but only used for /auth
//  3. leash-auth cookie — forwarded to the platform for integration calls
//
// Returns a non-nil error only on construction-time problems (none today —
// missing auth surfaces lazily on the call that needs it, mirroring TS).
func New(r *http.Request, opts ...Option) (*Client, error) {
	cfg := resolveConfig(opts...)
	if r != nil {
		cfg.cookieValue = extractCookie(r, CookieName)
		if cfg.apiKey == "" {
			// Inbound Bearer is captured for potential future flows, but it
			// is NOT forwarded on integration POSTs — see Transport.post.
			// We still allow it to participate in env-fetch authorization
			// if no explicit key was provided.
			cfg.authToken = extractBearerToken(r)
		}
	}
	return buildClient(cfg), nil
}

// NewWithToken constructs a [Client] from a server-side JWT.
//
// Use this for CLI / agent flows that hold the user's JWT directly (no
// inbound *http.Request available). The token is forwarded as the leash-auth
// cookie value to integration calls.
func NewWithToken(token string, opts ...Option) (*Client, error) {
	cfg := resolveConfig(opts...)
	cfg.cookieValue = token
	return buildClient(cfg), nil
}

// NewFromAPIKey constructs a [Client] for server-to-server flows.
//
// The API key is sent as X-API-Key on integration calls and as
// Authorization: Bearer on env-fetch calls. No leash-auth cookie is set, so
// integration calls that require user context will fail with
// [CodeUnauthorized] — use [New] for request-scoped flows.
func NewFromAPIKey(apiKey string, opts ...Option) (*Client, error) {
	cfg := resolveConfig(opts...)
	cfg.apiKey = apiKey
	return buildClient(cfg), nil
}

// resolveConfig folds the option chain on top of env-var defaults.
//
// Precedence (highest first):
//
//	explicit option > env var > built-in default
func resolveConfig(opts ...Option) *clientConfig {
	cfg := &clientConfig{
		platformURL: DefaultPlatformURL,
		httpClient:  http.DefaultClient,
		apiKey:      os.Getenv("LEASH_API_KEY"),
	}
	if envURL := os.Getenv("LEASH_PLATFORM_URL"); envURL != "" {
		cfg.platformURL = envURL
	}
	for _, o := range opts {
		o(cfg)
	}
	cfg.platformURL = strings.TrimRight(cfg.platformURL, "/")
	if cfg.platformURL == "" {
		cfg.platformURL = DefaultPlatformURL
	}
	if cfg.httpClient == nil {
		cfg.httpClient = http.DefaultClient
	}
	return cfg
}

// buildClient finalises a [Client] from a resolved config.
func buildClient(cfg *clientConfig) *Client {
	transport := newTransport(cfg)
	envApiKey := cfg.apiKey
	if envApiKey == "" {
		// Allow inbound bearer to authorise env reads when no explicit key
		// was provided. Bearer is still NOT forwarded on integration calls.
		envApiKey = cfg.authToken
	}
	return &Client{
		transport:    transport,
		auth:         &AuthNamespace{cookieValue: cfg.cookieValue},
		env:          newEnvNamespace(cfg.platformURL, envApiKey, cfg.httpClient),
		integrations: integrations.New(&transportAdapter{t: transport}),
	}
}

// Auth returns the identity namespace.
func (c *Client) Auth() *AuthNamespace { return c.auth }

// Env returns the runtime env-var namespace.
func (c *Client) Env() *EnvNamespace { return c.env }

// Integrations returns the integrations namespace (Gmail, Calendar, Drive,
// Linear, plus the generic [integrations.Namespace.Provider] escape hatch).
func (c *Client) Integrations() *integrations.Namespace { return c.integrations }

// Transport returns the underlying [Transport]. Useful for advanced flows
// that need to dispatch a raw call (e.g. previewing a new platform endpoint).
func (c *Client) Transport() *Transport { return c.transport }

// transportAdapter implements [integrations.Caller] by forwarding to the
// root client's [Transport]. Defined here (not in the integrations package)
// so the integrations package stays free of leash imports — keeping the
// import graph one-way and avoiding cycles.
type transportAdapter struct {
	t *Transport
}

// IntegrationsCall satisfies [integrations.Caller].
func (a *transportAdapter) IntegrationsCall(ctx context.Context, provider, action string, body any) (json.RawMessage, error) {
	return a.t.integrationsCall(ctx, provider, action, body)
}
