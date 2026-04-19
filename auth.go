package leash

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// LeashUser represents the authenticated user extracted from the leash-auth cookie.
type LeashUser struct {
	ID      string `json:"sub"`
	Email   string `json:"email"`
	Name    string `json:"name"`
	Picture string `json:"picture"`
}

// ErrNoCookie is returned when the leash-auth cookie is not present on the request.
var ErrNoCookie = errors.New("leash: leash-auth cookie not found")

// ErrInvalidToken is returned when the JWT token cannot be parsed or is invalid.
var ErrInvalidToken = errors.New("leash: invalid token")

// ErrExpiredToken is returned when the JWT token has expired.
var ErrExpiredToken = errors.New("leash: token has expired")

// GetLeashUser reads the leash-auth cookie from the request, decodes the JWT,
// and returns the authenticated user.
//
// If LEASH_JWT_SECRET is set, the token signature is verified using HMAC.
// If LEASH_JWT_SECRET is not set, the token is decoded without signature verification.
//
// This function works with any Go HTTP framework that uses or wraps *http.Request
// (stdlib net/http, Gin, Echo, Chi, Fiber, etc.).
func GetLeashUser(r *http.Request) (*LeashUser, error) {
	cookie, err := r.Cookie("leash-auth")
	if err != nil {
		return nil, ErrNoCookie
	}

	tokenStr := cookie.Value
	secret := os.Getenv("LEASH_JWT_SECRET")

	if secret != "" {
		return parseWithVerification(tokenStr, secret)
	}
	return parseWithoutVerification(tokenStr)
}

// IsAuthenticated returns true if the request contains a valid leash-auth cookie
// with a decodable JWT token.
func IsAuthenticated(r *http.Request) bool {
	user, err := GetLeashUser(r)
	return err == nil && user != nil
}

// parseWithVerification parses and verifies the JWT using the given HMAC secret.
func parseWithVerification(tokenStr, secret string) (*LeashUser, error) {
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("leash: unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}

	if !token.Valid {
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, ErrInvalidToken
	}

	return mapClaimsToUser(claims)
}

// parseWithoutVerification decodes the JWT payload without verifying the signature.
func parseWithoutVerification(tokenStr string) (*LeashUser, error) {
	parts := strings.Split(tokenStr, ".")
	if len(parts) != 3 {
		return nil, ErrInvalidToken
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, ErrInvalidToken
	}

	var claims map[string]any
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, ErrInvalidToken
	}

	// Check expiration even without signature verification
	if exp, ok := claims["exp"]; ok {
		var expFloat float64
		switch v := exp.(type) {
		case float64:
			expFloat = v
		case json.Number:
			expFloat, _ = v.Float64()
		}
		if expFloat > 0 && time.Now().Unix() > int64(expFloat) {
			return nil, ErrExpiredToken
		}
	}

	return mapClaimsToUser(claims)
}

// mapClaimsToUser extracts user fields from JWT claims.
func mapClaimsToUser(claims map[string]any) (*LeashUser, error) {
	user := &LeashUser{}

	if sub, ok := claims["sub"].(string); ok {
		user.ID = sub
	}
	if email, ok := claims["email"].(string); ok {
		user.Email = email
	}
	if name, ok := claims["name"].(string); ok {
		user.Name = name
	}
	if picture, ok := claims["picture"].(string); ok {
		user.Picture = picture
	}

	return user, nil
}

// mapClaimsToUser for jwt.MapClaims (same interface, different type assertion handled by Go).
// The above function works for both map[string]any and jwt.MapClaims since
// jwt.MapClaims is defined as map[string]interface{}.
