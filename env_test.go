package leash

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func newEnvClient(t *testing.T, plat *fakePlatform, apiKey string) *Client {
	t.Helper()
	c, err := NewFromAPIKey(apiKey, WithPlatformURL(plat.URL), WithHTTPClient(plat.server.Client()))
	if err != nil {
		t.Fatalf("construct client: %v", err)
	}
	return c
}

func TestEnvGet_ReturnsValueAndForwardsBearer(t *testing.T) {
	plat := newFakePlatform(t, map[string]platformResponse{
		"GET /api/apps/me/secrets/OPENAI_API_KEY": {Status: 200, Body: map[string]any{"value": "sk-test"}},
	})
	defer plat.Close()
	c := newEnvClient(t, plat, "lsk_live_abc")

	val, err := c.Env().Get(context.Background(), "OPENAI_API_KEY")
	if err != nil {
		t.Fatal(err)
	}
	if val == nil || *val != "sk-test" {
		t.Errorf("expected sk-test, got %v", val)
	}
	if got := plat.Captured[0].Headers.Get("Authorization"); got != "Bearer lsk_live_abc" {
		t.Errorf("expected Bearer auth, got %q", got)
	}
}

func TestEnvGet_404ReturnsNilNil(t *testing.T) {
	plat := newFakePlatform(t, map[string]platformResponse{
		"GET /api/apps/me/secrets/MISSING": {Status: 404, Body: map[string]any{}},
	})
	defer plat.Close()
	c := newEnvClient(t, plat, "lsk_live_abc")

	val, err := c.Env().Get(context.Background(), "MISSING")
	if err != nil {
		t.Fatalf("expected nil err for 404, got %v", err)
	}
	if val != nil {
		t.Errorf("expected nil value for 404, got %v", *val)
	}
}

func TestEnvGet_RequiresAPIKey(t *testing.T) {
	t.Setenv("LEASH_API_KEY", "")
	c, _ := New(nil)
	_, err := c.Env().Get(context.Background(), "ANY")
	var lerr *LeashError
	if !errors.As(err, &lerr) || lerr.Code != CodeNoAPIKey {
		t.Errorf("expected CodeNoAPIKey, got %v", err)
	}
}

func TestEnvGet_Errors(t *testing.T) {
	cases := []struct {
		name   string
		status int
		body   any
		code   ErrorCode
	}{
		{"400 invalid key", 400, map[string]any{"error": "invalid"}, CodeInvalidKey},
		{"401 unauthorized", 401, map[string]any{"error": "bad"}, CodeUnauthorized},
		{"402 upgrade required", 402, map[string]any{"requiredPlan": "growth"}, CodeUpgradeRequired},
		{"502 source resync", 502, map[string]any{"error": "aws timeout"}, CodeSourceResyncFailed},
		{"503 unexpected", 503, map[string]any{}, CodeEnvFetchError},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			plat := newFakePlatform(t, map[string]platformResponse{
				"GET /api/apps/me/secrets/X": {Status: c.status, Body: c.body},
			})
			defer plat.Close()
			client := newEnvClient(t, plat, "lsk_live_abc")
			_, err := client.Env().Get(context.Background(), "X")
			var lerr *LeashError
			if !errors.As(err, &lerr) || lerr.Code != c.code {
				t.Errorf("expected %q, got %v", c.code, err)
			}
		})
	}
}

func TestEnvGet_402SurfacesRequiredPlan(t *testing.T) {
	plat := newFakePlatform(t, map[string]platformResponse{
		"GET /api/apps/me/secrets/X": {Status: 402, Body: map[string]any{"requiredPlan": "growth"}},
	})
	defer plat.Close()
	c := newEnvClient(t, plat, "lsk_live_abc")
	_, err := c.Env().Get(context.Background(), "X")
	if !IsPlanBlock(err) {
		t.Fatalf("expected IsPlanBlock true, got %v", err)
	}
	if !strings.Contains(strings.ToLower(err.Error()), "growth") {
		t.Errorf("expected growth in message, got %v", err)
	}
}

func TestEnvGet_BodyMissingValue(t *testing.T) {
	plat := newFakePlatform(t, map[string]platformResponse{
		"GET /api/apps/me/secrets/X": {Status: 200, Body: map[string]any{}},
	})
	defer plat.Close()
	c := newEnvClient(t, plat, "lsk_live_abc")
	_, err := c.Env().Get(context.Background(), "X")
	var lerr *LeashError
	if !errors.As(err, &lerr) || lerr.Code != CodeEnvFetchError {
		t.Errorf("expected CodeEnvFetchError, got %v", err)
	}
}

func TestEnvGetMany_ResolvesAll(t *testing.T) {
	plat := newFakePlatform(t, map[string]platformResponse{
		"GET /api/apps/me/secrets/A":       {Status: 200, Body: map[string]any{"value": "1"}},
		"GET /api/apps/me/secrets/B":       {Status: 200, Body: map[string]any{"value": "2"}},
		"GET /api/apps/me/secrets/MISSING": {Status: 404, Body: map[string]any{}},
	})
	defer plat.Close()
	c := newEnvClient(t, plat, "lsk_live_abc")
	got, err := c.Env().GetMany(context.Background(), []string{"A", "B", "MISSING"})
	if err != nil {
		t.Fatal(err)
	}
	if got["A"] == nil || *got["A"] != "1" {
		t.Errorf("A: expected 1, got %v", got["A"])
	}
	if got["B"] == nil || *got["B"] != "2" {
		t.Errorf("B: expected 2, got %v", got["B"])
	}
	if got["MISSING"] != nil {
		t.Errorf("MISSING: expected nil, got %v", got["MISSING"])
	}
}

func TestEnvGet_CacheHit(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"value":"v1"}`))
	}))
	defer srv.Close()

	c, _ := NewFromAPIKey("k", WithPlatformURL(srv.URL), WithHTTPClient(srv.Client()))
	for i := 0; i < 3; i++ {
		_, err := c.Env().Get(context.Background(), "X")
		if err != nil {
			t.Fatal(err)
		}
	}
	if got := atomic.LoadInt32(&hits); got != 1 {
		t.Errorf("expected 1 network call (rest cached), got %d", got)
	}
}

func TestEnvGet_FreshBypassesCache(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&hits, 1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"value":"v` + itoa(int(n)) + `"}`))
	}))
	defer srv.Close()

	c, _ := NewFromAPIKey("k", WithPlatformURL(srv.URL), WithHTTPClient(srv.Client()))

	v1, _ := c.Env().Get(context.Background(), "X")
	v2, _ := c.Env().Get(context.Background(), "X")             // cache
	v3, _ := c.Env().Get(context.Background(), "X", EnvFresh()) // bypass
	v4, _ := c.Env().Get(context.Background(), "X")             // cache (post-fresh)
	if *v1 != "v1" || *v2 != "v1" || *v3 != "v2" || *v4 != "v2" {
		t.Errorf("unexpected sequence: %q %q %q %q", *v1, *v2, *v3, *v4)
	}
	if got := atomic.LoadInt32(&hits); got != 2 {
		t.Errorf("expected 2 network calls, got %d", got)
	}
}

func TestEnvGet_TTLOptionShortensCache(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"value":"v"}`))
	}))
	defer srv.Close()

	c, _ := NewFromAPIKey("k", WithPlatformURL(srv.URL), WithHTTPClient(srv.Client()))
	_, _ = c.Env().Get(context.Background(), "X", EnvTTL(1*time.Millisecond))
	time.Sleep(10 * time.Millisecond)
	_, _ = c.Env().Get(context.Background(), "X")
	if got := atomic.LoadInt32(&hits); got != 2 {
		t.Errorf("expected ttl expiry to force refetch, got %d hits", got)
	}
}

// itoa avoids strconv import in this test file for tighter coverage of stdlib.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
