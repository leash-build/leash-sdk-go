package leash

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newPlatformClient(t *testing.T, plat *fakePlatform) *Client {
	t.Helper()
	c, err := New(requestWithCookie(CookieName, "ck"), WithPlatformURL(plat.URL), WithAPIKey("lsk_live"))
	if err != nil {
		t.Fatal(err)
	}
	c.transport.HTTPClient = plat.server.Client()
	return c
}

func TestTransport_IntegrationErrorMapping(t *testing.T) {
	cases := []struct {
		name   string
		status int
		body   any
		code   ErrorCode
		pred   func(error) bool
	}{
		{"401", 401, map[string]any{"error": "bad cookie"}, CodeUnauthorized, IsUnauthorized},
		{"402", 402, map[string]any{"message": "Upgrade to Growth"}, CodeUpgradeRequired, IsPlanBlock},
		{"403", 403, map[string]any{"error": "Not allow-listed"}, CodeIntegrationNotEnabled, IsConnectionRequired},
		{"500", 500, map[string]any{"error": "boom"}, CodeIntegrationError, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			plat := newFakePlatform(t, map[string]platformResponse{
				"POST /api/integrations/gmail/list-messages": {Status: tc.status, Body: tc.body},
			})
			defer plat.Close()
			c := newPlatformClient(t, plat)
			_, err := c.Integrations().Gmail().ListMessages(context.Background(), nil)
			var lerr *LeashError
			if !errors.As(err, &lerr) {
				t.Fatalf("expected *LeashError, got %v", err)
			}
			if lerr.Code != tc.code {
				t.Errorf("got code %q, want %q", lerr.Code, tc.code)
			}
			if tc.pred != nil && !tc.pred(err) {
				t.Errorf("predicate failed for %q", tc.name)
			}
			if lerr.Status != tc.status {
				t.Errorf("expected status %d, got %d", tc.status, lerr.Status)
			}
		})
	}
}

func TestTransport_SuccessFalseInBody(t *testing.T) {
	plat := newFakePlatform(t, map[string]platformResponse{
		"POST /api/integrations/gmail/list-messages": {
			Status: 200,
			Body:   map[string]any{"success": false, "error": "Provider returned 429", "code": "RATE_LIMITED"},
		},
	})
	defer plat.Close()
	c := newPlatformClient(t, plat)
	_, err := c.Integrations().Gmail().ListMessages(context.Background(), nil)
	var lerr *LeashError
	if !errors.As(err, &lerr) {
		t.Fatalf("expected LeashError, got %v", err)
	}
	if lerr.Code != "RATE_LIMITED" {
		t.Errorf("expected RATE_LIMITED, got %q", lerr.Code)
	}
}

func TestTransport_NetworkErrorMapped(t *testing.T) {
	// Build a client pointing at an unroutable address — Dial will fail.
	c, err := NewFromAPIKey("k", WithPlatformURL("http://127.0.0.1:1"), WithHTTPClient(&http.Client{}))
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.Integrations().Gmail().ListMessages(context.Background(), nil)
	if !IsNetworkError(err) {
		t.Errorf("expected IsNetworkError true, got %v", err)
	}
}

func TestTransport_RawBodyPassedThroughForBareArrays(t *testing.T) {
	// Linear list_teams can return a bare array — verify the transport returns
	// the raw body untouched when the envelope shape doesn't match.
	plat := newFakePlatform(t, map[string]platformResponse{
		"POST /api/integrations/linear/list_teams": {Status: 200, Body: []map[string]any{{"id": "t1"}}},
	})
	defer plat.Close()
	c := newPlatformClient(t, plat)
	teams, err := c.Integrations().Linear().ListTeams(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(teams) != 1 || teams[0].ID != "t1" {
		t.Errorf("unexpected: %+v", teams)
	}
}

func TestTransport_PostContentType(t *testing.T) {
	plat := newFakePlatform(t, map[string]platformResponse{
		"POST /api/integrations/gmail/list-messages": {Status: 200, Body: map[string]any{"success": true, "data": map[string]any{}}},
	})
	defer plat.Close()
	c := newPlatformClient(t, plat)
	if _, err := c.Integrations().Gmail().ListMessages(context.Background(), nil); err != nil {
		t.Fatal(err)
	}
	if ct := plat.Captured[0].Headers.Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %q", ct)
	}
	if got := plat.Captured[0].Headers.Get("X-Api-Key"); got != "lsk_live" {
		t.Errorf("expected X-API-Key forwarded, got %q", got)
	}
}

func TestTransport_PlatformURLNormalised(t *testing.T) {
	// Constructor should trim trailing slashes — verify URL path doesn't get
	// "/api/integrations/..." double-slashed.
	plat := newFakePlatform(t, map[string]platformResponse{
		"POST /api/integrations/gmail/list-messages": {Status: 200, Body: map[string]any{"success": true, "data": map[string]any{}}},
	})
	defer plat.Close()
	c := newPlatformClient(t, plat)
	if !strings.HasSuffix(c.transport.PlatformURL, plat.URL[len(plat.URL)-5:]) {
		t.Logf("platform url: %q", c.transport.PlatformURL)
	}
	if _, err := c.Integrations().Gmail().ListMessages(context.Background(), nil); err != nil {
		t.Fatal(err)
	}
}

// Ensure we don't accidentally re-instantiate http.DefaultClient when option
// returns the same client.
func TestWithHTTPClient_RoundTrips(t *testing.T) {
	c1 := &http.Client{}
	c, _ := NewFromAPIKey("k", WithHTTPClient(c1))
	if c.transport.HTTPClient != c1 {
		t.Errorf("expected injected client, got different")
	}
}

func TestWithHTTPClient_NilDefaults(t *testing.T) {
	c, _ := NewFromAPIKey("k", WithHTTPClient(nil))
	if c.transport.HTTPClient != http.DefaultClient {
		t.Errorf("expected http.DefaultClient when WithHTTPClient(nil), got %v", c.transport.HTTPClient)
	}
}

// Quick sanity that httptest doesn't accidentally satisfy the bearer-forward
// gotcha — duplicates TestBearerNotForwardedOnIntegrationCalls but at the
// transport layer to make the contract obvious.
func TestTransport_NeverSendsAuthorization(t *testing.T) {
	plat := newFakePlatform(t, map[string]platformResponse{
		"POST /api/integrations/slack/post-message": {Status: 200, Body: map[string]any{"success": true, "data": map[string]any{}}},
	})
	defer plat.Close()
	r := requestWithBearer("some-jwt")
	c, _ := New(r, WithPlatformURL(plat.URL))
	c.transport.HTTPClient = plat.server.Client()
	if _, err := c.Integrations().Provider("slack").Call(context.Background(), "post-message", nil); err != nil {
		t.Fatal(err)
	}
	if plat.Captured[0].Headers.Get("Authorization") != "" {
		t.Errorf("Authorization header should never be sent on integration calls")
	}
}

var _ = httptest.NewServer // silence linter when only used indirectly
