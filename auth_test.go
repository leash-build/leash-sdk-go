package leash

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestExtractCookie(t *testing.T) {
	r := requestWithCookie(CookieName, "abc")
	if got := extractCookie(r, CookieName); got != "abc" {
		t.Errorf("expected abc, got %q", got)
	}
	if got := extractCookie(r, "other"); got != "" {
		t.Errorf("expected empty for missing cookie, got %q", got)
	}
	if got := extractCookie(nil, CookieName); got != "" {
		t.Errorf("expected empty for nil request, got %q", got)
	}
}

func TestExtractBearer(t *testing.T) {
	cases := []struct {
		name   string
		header string
		want   string
	}{
		{"valid", "Bearer abc", "abc"},
		{"lowercase scheme", "bearer abc", "abc"},
		{"trims whitespace", "Bearer  spaced ", "spaced"},
		{"basic auth", "Basic abc", ""},
		{"empty", "", ""},
		{"no scheme", "abc", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			if c.header != "" {
				r.Header.Set("Authorization", c.header)
			}
			if got := extractBearerToken(r); got != c.want {
				t.Errorf("got %q, want %q", got, c.want)
			}
		})
	}
}

func TestDecodeToken_VerifiedHappyPath(t *testing.T) {
	t.Setenv("LEASH_JWT_SECRET", testJWTSecret)
	tok := makeToken(t, nil, testJWTSecret, time.Hour)
	payload, err := decodeToken(tok)
	if err != nil {
		t.Fatal(err)
	}
	if payload.Sub != "user-123" {
		t.Errorf("expected sub=user-123, got %q", payload.Sub)
	}
}

func TestDecodeToken_ExpiredRejected(t *testing.T) {
	t.Setenv("LEASH_JWT_SECRET", testJWTSecret)
	tok := makeToken(t, nil, testJWTSecret, -time.Hour)
	_, err := decodeToken(tok)
	if err == nil {
		t.Fatal("expected error for expired token")
	}
	var lerr *LeashError
	if !errors.As(err, &lerr) || lerr.Code != CodeNoAuthContext {
		t.Errorf("expected CodeNoAuthContext, got %v", err)
	}
}

func TestDecodeToken_WrongSecretRejected(t *testing.T) {
	t.Setenv("LEASH_JWT_SECRET", testJWTSecret)
	tok := makeToken(t, nil, "other-secret", time.Hour)
	if _, err := decodeToken(tok); err == nil {
		t.Fatal("expected signature-mismatch error")
	}
}

func TestDecodeToken_MalformedRejected(t *testing.T) {
	if _, err := decodeToken("garbage"); err == nil {
		t.Fatal("expected malformed-token error")
	}
}

func TestDecodeToken_NoSecretDevFallback(t *testing.T) {
	t.Setenv("LEASH_JWT_SECRET", "")
	claims := map[string]any{
		"sub":   "user-789",
		"email": "bob@example.com",
		"name":  "Bob",
		"exp":   time.Now().Add(time.Hour).Unix(),
	}
	tok := makeUnsignedToken(t, claims)
	payload, err := decodeToken(tok)
	if err != nil {
		t.Fatalf("expected dev fallback to accept, got %v", err)
	}
	if payload.Sub != "user-789" {
		t.Errorf("unexpected sub: %q", payload.Sub)
	}
}

func TestDecodeToken_NoSecretStillChecksExpiry(t *testing.T) {
	t.Setenv("LEASH_JWT_SECRET", "")
	claims := map[string]any{
		"sub": "u",
		"exp": time.Now().Add(-time.Hour).Unix(),
	}
	tok := makeUnsignedToken(t, claims)
	if _, err := decodeToken(tok); err == nil {
		t.Fatal("expected expiry rejection even without secret")
	}
}

func TestPayloadToUser_PrefersUserId(t *testing.T) {
	user, err := payloadToUser(&LeashJWTPayload{UserID: "primary", Sub: "fallback", Email: "a@b.c", Name: "A"})
	if err != nil {
		t.Fatal(err)
	}
	if user.ID != "primary" {
		t.Errorf("expected primary, got %q", user.ID)
	}
}

func TestPayloadToUser_FallsBackToSub(t *testing.T) {
	user, err := payloadToUser(&LeashJWTPayload{Sub: "fallback", Email: "a@b.c", Name: "A"})
	if err != nil {
		t.Fatal(err)
	}
	if user.ID != "fallback" {
		t.Errorf("expected fallback, got %q", user.ID)
	}
}

func TestPayloadToUser_MissingIDFails(t *testing.T) {
	if _, err := payloadToUser(&LeashJWTPayload{Email: "a@b.c"}); err == nil {
		t.Fatal("expected error for missing id")
	}
}

func TestGetLeashUser_Happy(t *testing.T) {
	t.Setenv("LEASH_JWT_SECRET", testJWTSecret)
	tok := makeToken(t, nil, testJWTSecret, time.Hour)
	r := requestWithCookie(CookieName, tok)
	user, err := GetLeashUser(r)
	if err != nil {
		t.Fatal(err)
	}
	if user.Email != "alice@example.com" {
		t.Errorf("unexpected email: %q", user.Email)
	}
}

func TestGetLeashUser_MissingCookie(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	_, err := GetLeashUser(r)
	if err == nil {
		t.Fatal("expected error for missing cookie")
	}
	var lerr *LeashError
	if !errors.As(err, &lerr) || lerr.Code != CodeNoAuthContext {
		t.Errorf("expected CodeNoAuthContext, got %v", err)
	}
}

func TestIsAuthenticated(t *testing.T) {
	t.Setenv("LEASH_JWT_SECRET", testJWTSecret)
	tok := makeToken(t, nil, testJWTSecret, time.Hour)
	if !IsAuthenticated(requestWithCookie(CookieName, tok)) {
		t.Error("expected authenticated")
	}
	if IsAuthenticated(httptest.NewRequest(http.MethodGet, "/", nil)) {
		t.Error("expected unauthenticated without cookie")
	}
	if IsAuthenticated(requestWithCookie(CookieName, "garbage")) {
		t.Error("expected unauthenticated with garbage cookie")
	}
}

func TestAuthNamespace_UserReturnsNilOnNoCookie(t *testing.T) {
	c, _ := New(httptest.NewRequest(http.MethodGet, "/", nil))
	user, err := c.Auth().User(context.Background())
	if err != nil {
		t.Errorf("expected no error for unauthenticated, got %v", err)
	}
	if user != nil {
		t.Errorf("expected nil user, got %+v", user)
	}
	if c.Auth().IsAuthenticated() {
		t.Error("expected IsAuthenticated=false")
	}
}

func TestAuthNamespace_SwallowsDecodeErrors(t *testing.T) {
	r := requestWithCookie(CookieName, "garbage-jwt")
	c, _ := New(r)
	user, err := c.Auth().User(context.Background())
	if err != nil {
		t.Errorf("Auth.User should swallow decode errors, got %v", err)
	}
	if user != nil {
		t.Errorf("expected nil user for garbage token, got %+v", user)
	}
}
