package leash

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

const testJWTSecret = "test-secret-key"

// makeToken mints a signed HS256 JWT for tests. expOffset is added to time.Now()
// — pass a positive value for a valid token, negative for expired, or 0 to omit.
func makeToken(t *testing.T, claims map[string]any, secret string, expOffset time.Duration) string {
	t.Helper()
	if claims == nil {
		claims = map[string]any{}
	}
	if _, ok := claims["sub"]; !ok {
		claims["sub"] = "user-123"
	}
	if _, ok := claims["email"]; !ok {
		claims["email"] = "alice@example.com"
	}
	if _, ok := claims["name"]; !ok {
		claims["name"] = "Alice"
	}
	if expOffset != 0 {
		claims["exp"] = time.Now().Add(expOffset).Unix()
	}

	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	payloadBytes, err := json.Marshal(claims)
	if err != nil {
		t.Fatalf("marshal claims: %v", err)
	}
	payload := base64.RawURLEncoding.EncodeToString(payloadBytes)
	signing := header + "." + payload
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signing))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return signing + "." + sig
}

// makeUnsignedToken mints a JWT with a fake signature — used to verify the
// dev-fallback path that skips signature verification.
func makeUnsignedToken(t *testing.T, claims map[string]any) string {
	t.Helper()
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none","typ":"JWT"}`))
	payloadBytes, err := json.Marshal(claims)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	payload := base64.RawURLEncoding.EncodeToString(payloadBytes)
	return header + "." + payload + ".fakesignature"
}

// requestWithCookie builds an *http.Request with the named cookie attached.
func requestWithCookie(name, value string) *http.Request {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.AddCookie(&http.Cookie{Name: name, Value: value})
	return r
}

// requestWithBearer builds an *http.Request with an Authorization: Bearer header.
func requestWithBearer(token string) *http.Request {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Authorization", "Bearer "+token)
	return r
}

// platformResponse describes a single mocked HTTP response keyed by URL path.
type platformResponse struct {
	Status int
	Body   any // raw JSON-able body
}

// newPlatformServer builds an httptest server that returns mapped responses for
// the given (method,path) pairs. captured holds every request body received in
// order, so tests can assert the exact wire shape.
type captured struct {
	Path    string
	Method  string
	Headers http.Header
	Body    []byte
}

type fakePlatform struct {
	URL      string
	Captured []captured
	server   *httptest.Server
}

func (f *fakePlatform) Close() { f.server.Close() }

func newFakePlatform(t *testing.T, routes map[string]platformResponse) *fakePlatform {
	t.Helper()
	f := &fakePlatform{}
	f.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := readAll(r)
		f.Captured = append(f.Captured, captured{
			Path:    r.URL.Path,
			Method:  r.Method,
			Headers: r.Header.Clone(),
			Body:    body,
		})
		key := r.Method + " " + r.URL.Path
		resp, ok := routes[key]
		if !ok {
			http.Error(w, fmt.Sprintf("unexpected route: %s", key), http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(resp.Status)
		if resp.Body == nil {
			return
		}
		switch b := resp.Body.(type) {
		case []byte:
			_, _ = w.Write(b)
		case string:
			_, _ = w.Write([]byte(b))
		default:
			_ = json.NewEncoder(w).Encode(b)
		}
	}))
	f.URL = f.server.URL
	return f
}

// readAll drains the body and resets it (so the handler can still see it).
func readAll(r *http.Request) ([]byte, error) {
	if r.Body == nil {
		return nil, nil
	}
	defer r.Body.Close()
	buf := make([]byte, 0, 256)
	chunk := make([]byte, 256)
	for {
		n, err := r.Body.Read(chunk)
		if n > 0 {
			buf = append(buf, chunk[:n]...)
		}
		if err != nil {
			break
		}
	}
	return buf, nil
}
