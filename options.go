package leash

import (
	"net/http"
	"time"
)

// Option configures a [Client] at construction time.
//
// Pass any combination to [New], [NewWithToken], or [NewFromAPIKey]:
//
//	client, err := leash.New(r,
//	    leash.WithPlatformURL("https://staging.leash.build"),
//	    leash.WithHTTPClient(myHTTPClient),
//	)
type Option func(*clientConfig)

// clientConfig holds the resolved constructor knobs before the [Client]
// snapshots them. Internal — consumers only see [Option].
type clientConfig struct {
	platformURL string
	httpClient  *http.Client
	apiKey      string
	authToken   string
	cookieValue string
}

// WithPlatformURL overrides the platform base URL.
//
// Precedence: explicit Option > LEASH_PLATFORM_URL env var > the default
// https://leash.build.
func WithPlatformURL(url string) Option {
	return func(c *clientConfig) {
		c.platformURL = url
	}
}

// WithHTTPClient injects a custom [*http.Client] for outbound requests.
//
// Useful for tests (with httptest.NewServer) or for applying timeouts /
// transports your application needs.
func WithHTTPClient(client *http.Client) Option {
	return func(c *clientConfig) {
		c.httpClient = client
	}
}

// WithAPIKey provides an explicit LEASH_API_KEY, overriding the env var.
//
// Equivalent to constructing with [NewFromAPIKey], but composable when you
// already have an *http.Request.
func WithAPIKey(apiKey string) Option {
	return func(c *clientConfig) {
		c.apiKey = apiKey
	}
}

// ---------------------------------------------------------------------------
// Env.Get options (variadic)
// ---------------------------------------------------------------------------

// EnvOption configures a single [EnvNamespace.Get] call.
//
// Used as variadic args:
//
//	val, err := client.Env().Get(ctx, "OPENAI_API_KEY", leash.EnvFresh())
type EnvOption func(*envCallConfig)

// envCallConfig holds the resolved per-call knobs for env.Get.
type envCallConfig struct {
	fresh bool
	// ttl is reserved for a future per-call override; the platform contract
	// is currently fixed at 60s. Kept as a public option so callers can opt
	// into a longer-lived snapshot once support lands.
	ttl time.Duration
}

// EnvFresh bypasses the in-memory TTL cache for this call. The freshly
// fetched value is still written back to the cache for subsequent reads.
//
// Mirrors the TS `{ fresh: true }` option.
func EnvFresh() EnvOption {
	return func(c *envCallConfig) {
		c.fresh = true
	}
}

// EnvTTL overrides the cache TTL applied to the freshly-fetched value.
//
// The default cache window is 60s — bump it when a key changes infrequently
// and you want to amortise the lookup across more requests.
func EnvTTL(d time.Duration) EnvOption {
	return func(c *envCallConfig) {
		c.ttl = d
	}
}
