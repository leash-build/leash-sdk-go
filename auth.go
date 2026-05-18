package leash

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"
)

// CookieName is the name of the leash-auth cookie the platform sets.
const CookieName = "leash-auth"

// ---------------------------------------------------------------------------
// Cookie + Bearer extraction (HTTP request → string)
// ---------------------------------------------------------------------------

// extractCookie returns the named cookie value off an *http.Request, or "" when
// the cookie is missing.
func extractCookie(r *http.Request, name string) string {
	if r == nil {
		return ""
	}
	c, err := r.Cookie(name)
	if err != nil {
		return ""
	}
	return c.Value
}

// extractBearerToken returns the token portion of an Authorization: Bearer …
// header, or "" when absent/malformed.
func extractBearerToken(r *http.Request) string {
	if r == nil {
		return ""
	}
	raw := r.Header.Get("Authorization")
	if raw == "" {
		return ""
	}
	parts := strings.SplitN(raw, " ", 2)
	if len(parts) != 2 {
		return ""
	}
	if !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	tok := strings.TrimSpace(parts[1])
	if tok == "" {
		return ""
	}
	return tok
}

// ---------------------------------------------------------------------------
// JWT decoding (stdlib-only mini parser)
// ---------------------------------------------------------------------------

// decodeToken parses a leash-auth JWT and returns the user payload.
//
// When LEASH_JWT_SECRET is set, the HS256 signature is verified. Without it,
// the SDK falls back to verify-disabled decoding so local development works
// without provisioning a secret — the platform still controls issuance, so a
// malformed token still gets rejected here.
//
// Mirrors the dev-fallback in leash-sdk-ts/src/server/auth.ts and
// leash-sdk-python/leash/auth.py.
func decodeToken(token string) (*LeashJWTPayload, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, &LeashError{
			Code:    CodeNoAuthContext,
			Message: "Invalid leash-auth cookie: malformed JWT.",
			Action:  "Re-open the app in the Leash dashboard to mint a fresh cookie.",
			SeeAlso: "https://leash.build/docs/sdk",
		}
	}

	secret := os.Getenv("LEASH_JWT_SECRET")
	if secret != "" {
		if err := verifyHS256(parts[0]+"."+parts[1], parts[2], secret); err != nil {
			return nil, err
		}
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, &LeashError{
			Code:    CodeNoAuthContext,
			Message: "Invalid leash-auth cookie: payload not valid base64url.",
			Action:  "Re-open the app in the Leash dashboard to mint a fresh cookie.",
			SeeAlso: "https://leash.build/docs/sdk",
			Cause:   err,
		}
	}

	var raw map[string]any
	if err := json.Unmarshal(payloadBytes, &raw); err != nil {
		return nil, &LeashError{
			Code:    CodeNoAuthContext,
			Message: "Invalid leash-auth cookie: payload not valid JSON.",
			Action:  "Re-open the app in the Leash dashboard to mint a fresh cookie.",
			SeeAlso: "https://leash.build/docs/sdk",
			Cause:   err,
		}
	}

	// Expiry — always checked, even when signature verification is disabled.
	if expVal, ok := raw["exp"]; ok {
		var exp int64
		switch v := expVal.(type) {
		case float64:
			exp = int64(v)
		case int64:
			exp = v
		case int:
			exp = int64(v)
		}
		if exp > 0 && time.Now().Unix() > exp {
			return nil, &LeashError{
				Code:    CodeNoAuthContext,
				Message: "leash-auth cookie has expired.",
				Action:  "Re-open the app from the Leash dashboard to refresh the cookie.",
				SeeAlso: "https://leash.build/docs/sdk",
			}
		}
	}

	p := &LeashJWTPayload{}
	if v, ok := raw["userId"].(string); ok {
		p.UserID = v
	}
	if v, ok := raw["sub"].(string); ok {
		p.Sub = v
	}
	if v, ok := raw["email"].(string); ok {
		p.Email = v
	}
	if v, ok := raw["name"].(string); ok {
		p.Name = v
	}
	if v, ok := raw["username"].(string); ok {
		p.Username = v
	}
	if v, ok := raw["picture"].(string); ok {
		p.Picture = v
	}
	if v, ok := raw["iat"].(float64); ok {
		p.IssuedAt = int64(v)
	}
	if v, ok := raw["exp"].(float64); ok {
		p.Expires = int64(v)
	}
	return p, nil
}

// verifyHS256 verifies an HS256 JWT signature using the shared secret.
// signed = header + "." + payload (base64url-encoded segments joined by a dot).
func verifyHS256(signed, sigB64, secret string) error {
	expectedSig, err := base64.RawURLEncoding.DecodeString(sigB64)
	if err != nil {
		return &LeashError{
			Code:    CodeNoAuthContext,
			Message: "Invalid leash-auth cookie: signature not valid base64url.",
			Action:  "Re-open the app in the Leash dashboard to mint a fresh cookie.",
			SeeAlso: "https://leash.build/docs/sdk",
			Cause:   err,
		}
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signed))
	got := mac.Sum(nil)
	if !hmac.Equal(got, expectedSig) {
		return &LeashError{
			Code:    CodeNoAuthContext,
			Message: "Invalid leash-auth cookie: signature mismatch.",
			Action:  "Re-open the app in the Leash dashboard to mint a fresh cookie.",
			SeeAlso: "https://leash.build/docs/sdk",
		}
	}
	return nil
}

// payloadToUser converts a JWT payload into a [LeashUser].
//
// Prefers `userId` when present (legacy platforms) and falls back to `sub`.
func payloadToUser(p *LeashJWTPayload) (*LeashUser, error) {
	if p == nil {
		return nil, &LeashError{
			Code:    CodeNoAuthContext,
			Message: "leash-auth cookie missing payload.",
			Action:  "Re-open the app in the Leash dashboard to mint a fresh cookie.",
			SeeAlso: "https://leash.build/docs/sdk",
		}
	}
	id := p.UserID
	if id == "" {
		id = p.Sub
	}
	if id == "" {
		return nil, &LeashError{
			Code:    CodeNoAuthContext,
			Message: "leash-auth cookie missing user identifier.",
			Action:  "Re-open the app in the Leash dashboard to mint a fresh cookie.",
			SeeAlso: "https://leash.build/docs/sdk",
		}
	}
	return &LeashUser{
		ID:      id,
		Email:   p.Email,
		Name:    p.Name,
		Picture: p.Picture,
	}, nil
}

// ---------------------------------------------------------------------------
// Public free functions (framework-agnostic server helpers)
// ---------------------------------------------------------------------------

// GetLeashUser decodes the request's leash-auth cookie into a [LeashUser].
//
// Returns a *LeashError with Code [CodeNoAuthContext] when the cookie is
// missing or invalid. For a non-throwing variant, use [Client.Auth].User
// — it swallows decode errors and returns (nil, nil) for unauthenticated
// requests so handlers can branch with a clean `if user == nil`.
func GetLeashUser(r *http.Request) (*LeashUser, error) {
	tok := extractCookie(r, CookieName)
	if tok == "" {
		return nil, &LeashError{
			Code:    CodeNoAuthContext,
			Message: "No leash-auth cookie on the request.",
			Action:  "Open the app from the Leash dashboard to mint a fresh cookie.",
			SeeAlso: "https://leash.build/docs/sdk",
		}
	}
	p, err := decodeToken(tok)
	if err != nil {
		return nil, err
	}
	return payloadToUser(p)
}

// IsAuthenticated reports whether the request has a valid leash-auth cookie.
func IsAuthenticated(r *http.Request) bool {
	_, err := GetLeashUser(r)
	return err == nil
}

// ---------------------------------------------------------------------------
// AuthNamespace (instance method on Client.Auth())
// ---------------------------------------------------------------------------

// AuthNamespace is returned by [Client.Auth] and exposes identity helpers
// scoped to the request the [Client] was built from.
type AuthNamespace struct {
	cookieValue string
}

// User returns the authenticated user, or (nil, nil) when not authenticated.
//
// Decode errors are swallowed — this method never returns a non-nil error
// for a missing/invalid cookie so handlers can branch cleanly:
//
//	user, err := client.Auth().User(ctx)
//	if err != nil { /* handle infra error */ }
//	if user == nil { /* not signed in */ }
//
// The ctx argument is accepted for forward compatibility (so the surface
// matches the other namespaces). Decode is purely in-memory today.
func (a *AuthNamespace) User(ctx context.Context) (*LeashUser, error) {
	_ = ctx
	if a == nil || a.cookieValue == "" {
		return nil, nil
	}
	p, err := decodeToken(a.cookieValue)
	if err != nil {
		return nil, nil //nolint:nilerr // intentional — see docstring
	}
	user, err := payloadToUser(p)
	if err != nil {
		return nil, nil //nolint:nilerr // intentional — see docstring
	}
	return user, nil
}

// IsAuthenticated reports whether [AuthNamespace.User] would return a user.
func (a *AuthNamespace) IsAuthenticated() bool {
	if a == nil {
		return false
	}
	user, _ := a.User(context.Background())
	return user != nil
}
