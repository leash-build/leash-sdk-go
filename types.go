package leash

// LeashUser is the authenticated user extracted from the leash-auth cookie.
//
// Mirrors the TS [LeashUser] interface. Picture is optional.
type LeashUser struct {
	ID      string `json:"id"`
	Email   string `json:"email"`
	Name    string `json:"name"`
	Picture string `json:"picture,omitempty"`
}

// LeashJWTPayload is the raw JWT-claims shape the Leash platform sets on the
// leash-auth cookie. Either UserID or Sub will be populated.
type LeashJWTPayload struct {
	UserID   string `json:"userId,omitempty"`
	Sub      string `json:"sub,omitempty"`
	Email    string `json:"email,omitempty"`
	Name     string `json:"name,omitempty"`
	Username string `json:"username,omitempty"`
	Picture  string `json:"picture,omitempty"`
	IssuedAt int64  `json:"iat,omitempty"`
	Expires  int64  `json:"exp,omitempty"`
}
