package leash

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNew_NilRequestStillReturnsClient(t *testing.T) {
	c, err := New(nil)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if c == nil {
		t.Fatal("expected non-nil client")
	}
	if c.Auth() == nil || c.Env() == nil || c.Integrations() == nil {
		t.Fatal("expected all namespaces wired")
	}
}

func TestNew_ExtractsCookieAndBearer(t *testing.T) {
	t.Setenv("LEASH_JWT_SECRET", testJWTSecret)
	token := makeToken(t, nil, testJWTSecret, time.Hour)
	r := requestWithCookie(CookieName, token)
	r.Header.Set("Authorization", "Bearer some-other-jwt")

	c, err := New(r)
	if err != nil {
		t.Fatal(err)
	}
	user, err := c.Auth().User(context.Background())
	if err != nil || user == nil {
		t.Fatalf("expected user, got err=%v user=%v", err, user)
	}
	if user.Email != "alice@example.com" {
		t.Errorf("unexpected email: %q", user.Email)
	}
}

func TestNew_AuthPrecedence_APIKey(t *testing.T) {
	t.Setenv("LEASH_API_KEY", "env-key")
	c, err := New(nil, WithAPIKey("explicit-key"))
	if err != nil {
		t.Fatal(err)
	}
	if c.transport.APIKey != "explicit-key" {
		t.Errorf("explicit option should win, got %q", c.transport.APIKey)
	}
}

func TestNew_AuthPrecedence_EnvVar(t *testing.T) {
	t.Setenv("LEASH_API_KEY", "env-key")
	c, err := New(nil)
	if err != nil {
		t.Fatal(err)
	}
	if c.transport.APIKey != "env-key" {
		t.Errorf("env var should win when no option, got %q", c.transport.APIKey)
	}
}

func TestNewWithToken_SetsCookieValue(t *testing.T) {
	c, err := NewWithToken("jwt-from-cli", WithAPIKey("k"))
	if err != nil {
		t.Fatal(err)
	}
	if c.transport.CookieValue != "jwt-from-cli" {
		t.Errorf("expected cookie value, got %q", c.transport.CookieValue)
	}
	if c.transport.APIKey != "k" {
		t.Errorf("expected api key, got %q", c.transport.APIKey)
	}
}

func TestNewFromAPIKey_SetsAPIKey(t *testing.T) {
	c, err := NewFromAPIKey("lsk_live_abc")
	if err != nil {
		t.Fatal(err)
	}
	if c.transport.APIKey != "lsk_live_abc" {
		t.Errorf("expected explicit api key, got %q", c.transport.APIKey)
	}
}

func TestPlatformURL_EnvOverride(t *testing.T) {
	t.Setenv("LEASH_PLATFORM_URL", "https://staging.leash.build/")
	c, err := New(nil)
	if err != nil {
		t.Fatal(err)
	}
	if c.transport.PlatformURL != "https://staging.leash.build" {
		t.Errorf("expected staging URL trimmed, got %q", c.transport.PlatformURL)
	}
}

func TestPlatformURL_ExplicitOptionWins(t *testing.T) {
	t.Setenv("LEASH_PLATFORM_URL", "https://env.leash.build")
	c, err := New(nil, WithPlatformURL("https://explicit.leash.build/"))
	if err != nil {
		t.Fatal(err)
	}
	if c.transport.PlatformURL != "https://explicit.leash.build" {
		t.Errorf("expected explicit URL, got %q", c.transport.PlatformURL)
	}
}

func TestBearerNotForwardedOnIntegrationCalls(t *testing.T) {
	// Critical platform contract: inbound Bearer header MUST NOT be sent on
	// integration calls — only X-API-Key + Cookie are forwarded.
	t.Setenv("LEASH_API_KEY", "")

	plat := newFakePlatform(t, map[string]platformResponse{
		"POST /api/integrations/gmail/list-messages": {Status: 200, Body: map[string]any{"success": true, "data": map[string]any{}}},
	})
	defer plat.Close()

	r := requestWithBearer("some-jwt")
	c, err := New(r, WithPlatformURL(plat.URL), WithHTTPClient(httptest.NewServer(nil).Client()))
	if err != nil {
		t.Fatal(err)
	}
	// Replace HTTPClient with the plat's default client (which talks to plat.URL)
	c.transport.HTTPClient = plat.server.Client()

	_, err = c.Integrations().Gmail().ListMessages(context.Background(), nil)
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if len(plat.Captured) != 1 {
		t.Fatalf("expected 1 request, got %d", len(plat.Captured))
	}
	got := plat.Captured[0].Headers
	if got.Get("Authorization") != "" {
		t.Errorf("Authorization header should not be forwarded, got %q", got.Get("Authorization"))
	}
	if got.Get("X-Api-Key") != "" {
		t.Errorf("no API key was set; X-API-Key should be empty, got %q", got.Get("X-Api-Key"))
	}
}

func TestCookieForwardedOnIntegrationCalls(t *testing.T) {
	plat := newFakePlatform(t, map[string]platformResponse{
		"POST /api/integrations/gmail/list-messages": {Status: 200, Body: map[string]any{"success": true, "data": map[string]any{}}},
	})
	defer plat.Close()

	r := requestWithCookie(CookieName, "leash-cookie-jwt")
	c, err := New(r, WithPlatformURL(plat.URL))
	if err != nil {
		t.Fatal(err)
	}
	c.transport.HTTPClient = plat.server.Client()

	if _, err := c.Integrations().Gmail().ListMessages(context.Background(), nil); err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	cookieHeader := plat.Captured[0].Headers.Get("Cookie")
	if !strings.Contains(cookieHeader, CookieName+"=leash-cookie-jwt") {
		t.Errorf("expected cookie forwarded, got %q", cookieHeader)
	}
}
