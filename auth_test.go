package leash

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const testSecret = "test-jwt-secret-key"

// makeToken creates a signed JWT with the given claims and secret.
func makeToken(claims jwt.MapClaims, secret string) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	s, _ := token.SignedString([]byte(secret))
	return s
}

// makeUnsignedToken creates a JWT-shaped string (header.payload.signature)
// without a valid signature, for testing unverified decoding.
func makeUnsignedToken(claims map[string]any) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none","typ":"JWT"}`))
	payload, _ := json.Marshal(claims)
	payloadEnc := base64.RawURLEncoding.EncodeToString(payload)
	return header + "." + payloadEnc + ".fakesignature"
}

func validClaims() jwt.MapClaims {
	return jwt.MapClaims{
		"sub":     "user-123",
		"email":   "alice@example.com",
		"name":    "Alice Smith",
		"picture": "https://example.com/alice.jpg",
		"exp":     float64(time.Now().Add(1 * time.Hour).Unix()),
	}
}

func requestWithCookie(cookieValue string) *http.Request {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.AddCookie(&http.Cookie{Name: "leash-auth", Value: cookieValue})
	return r
}

func TestGetLeashUser_ValidToken(t *testing.T) {
	os.Setenv("LEASH_JWT_SECRET", testSecret)
	defer os.Unsetenv("LEASH_JWT_SECRET")

	token := makeToken(validClaims(), testSecret)
	r := requestWithCookie(token)

	user, err := GetLeashUser(r)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if user.ID != "user-123" {
		t.Errorf("expected ID=%q, got %q", "user-123", user.ID)
	}
	if user.Email != "alice@example.com" {
		t.Errorf("expected Email=%q, got %q", "alice@example.com", user.Email)
	}
	if user.Name != "Alice Smith" {
		t.Errorf("expected Name=%q, got %q", "Alice Smith", user.Name)
	}
	if user.Picture != "https://example.com/alice.jpg" {
		t.Errorf("expected Picture=%q, got %q", "https://example.com/alice.jpg", user.Picture)
	}
}

func TestGetLeashUser_MissingCookie(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)

	_, err := GetLeashUser(r)
	if err != ErrNoCookie {
		t.Errorf("expected ErrNoCookie, got %v", err)
	}
}

func TestGetLeashUser_InvalidToken(t *testing.T) {
	os.Setenv("LEASH_JWT_SECRET", testSecret)
	defer os.Unsetenv("LEASH_JWT_SECRET")

	r := requestWithCookie("not-a-valid-jwt")

	_, err := GetLeashUser(r)
	if err != ErrInvalidToken {
		t.Errorf("expected ErrInvalidToken, got %v", err)
	}
}

func TestGetLeashUser_ExpiredToken(t *testing.T) {
	os.Setenv("LEASH_JWT_SECRET", testSecret)
	defer os.Unsetenv("LEASH_JWT_SECRET")

	claims := validClaims()
	claims["exp"] = float64(time.Now().Add(-1 * time.Hour).Unix())
	token := makeToken(claims, testSecret)
	r := requestWithCookie(token)

	_, err := GetLeashUser(r)
	if err != ErrExpiredToken {
		t.Errorf("expected ErrExpiredToken, got %v", err)
	}
}

func TestGetLeashUser_WrongSecret(t *testing.T) {
	os.Setenv("LEASH_JWT_SECRET", testSecret)
	defer os.Unsetenv("LEASH_JWT_SECRET")

	token := makeToken(validClaims(), "wrong-secret")
	r := requestWithCookie(token)

	_, err := GetLeashUser(r)
	if err != ErrInvalidToken {
		t.Errorf("expected ErrInvalidToken, got %v", err)
	}
}

func TestGetLeashUser_NoSecret_DecodesWithoutVerification(t *testing.T) {
	os.Unsetenv("LEASH_JWT_SECRET")

	claims := map[string]any{
		"sub":     "user-456",
		"email":   "bob@example.com",
		"name":    "Bob Jones",
		"picture": "https://example.com/bob.jpg",
		"exp":     float64(time.Now().Add(1 * time.Hour).Unix()),
	}
	token := makeUnsignedToken(claims)
	r := requestWithCookie(token)

	user, err := GetLeashUser(r)
	if err != nil {
		t.Fatalf("expected no error without secret, got %v", err)
	}
	if user.ID != "user-456" {
		t.Errorf("expected ID=%q, got %q", "user-456", user.ID)
	}
	if user.Email != "bob@example.com" {
		t.Errorf("expected Email=%q, got %q", "bob@example.com", user.Email)
	}
	if user.Name != "Bob Jones" {
		t.Errorf("expected Name=%q, got %q", "Bob Jones", user.Name)
	}
}

func TestGetLeashUser_NoSecret_ExpiredToken(t *testing.T) {
	os.Unsetenv("LEASH_JWT_SECRET")

	claims := map[string]any{
		"sub":   "user-789",
		"email": "expired@example.com",
		"exp":   float64(time.Now().Add(-1 * time.Hour).Unix()),
	}
	token := makeUnsignedToken(claims)
	r := requestWithCookie(token)

	_, err := GetLeashUser(r)
	if err != ErrExpiredToken {
		t.Errorf("expected ErrExpiredToken, got %v", err)
	}
}

func TestIsAuthenticated_True(t *testing.T) {
	os.Setenv("LEASH_JWT_SECRET", testSecret)
	defer os.Unsetenv("LEASH_JWT_SECRET")

	token := makeToken(validClaims(), testSecret)
	r := requestWithCookie(token)

	if !IsAuthenticated(r) {
		t.Error("expected IsAuthenticated to return true for valid token")
	}
}

func TestIsAuthenticated_False_NoCookie(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)

	if IsAuthenticated(r) {
		t.Error("expected IsAuthenticated to return false with no cookie")
	}
}

func TestIsAuthenticated_False_InvalidToken(t *testing.T) {
	os.Setenv("LEASH_JWT_SECRET", testSecret)
	defer os.Unsetenv("LEASH_JWT_SECRET")

	r := requestWithCookie("garbage")

	if IsAuthenticated(r) {
		t.Error("expected IsAuthenticated to return false for invalid token")
	}
}

func TestIsAuthenticated_False_ExpiredToken(t *testing.T) {
	os.Setenv("LEASH_JWT_SECRET", testSecret)
	defer os.Unsetenv("LEASH_JWT_SECRET")

	claims := validClaims()
	claims["exp"] = float64(time.Now().Add(-1 * time.Hour).Unix())
	token := makeToken(claims, testSecret)
	r := requestWithCookie(token)

	if IsAuthenticated(r) {
		t.Error("expected IsAuthenticated to return false for expired token")
	}
}

func TestGetLeashUser_WorksWithStandardHTTPRequest(t *testing.T) {
	os.Setenv("LEASH_JWT_SECRET", testSecret)
	defer os.Unsetenv("LEASH_JWT_SECRET")

	// Use httptest.NewRequest which returns a standard *http.Request
	token := makeToken(validClaims(), testSecret)
	r := httptest.NewRequest(http.MethodGet, "https://myapp.example.com/dashboard", nil)
	r.AddCookie(&http.Cookie{Name: "leash-auth", Value: token})

	user, err := GetLeashUser(r)
	if err != nil {
		t.Fatalf("expected no error with standard http.Request, got %v", err)
	}
	if user.Email != "alice@example.com" {
		t.Errorf("expected Email=%q, got %q", "alice@example.com", user.Email)
	}
}
