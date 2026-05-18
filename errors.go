// Package leash provides the Go SDK for the Leash platform — auth, env, and
// integrations through a single unified [Client].
//
// See [New] for the canonical entry point. The package is framework-agnostic:
// any code path that produces an *http.Request (net/http, chi, gin, echo,
// fiber, lambdaurl, etc.) can construct a client.
package leash

import (
	"errors"
	"fmt"
)

// ErrorCode is a stable, machine-readable identifier for a [LeashError].
//
// Codes mirror leash-sdk-ts/src/errors.ts so consumers can switch on the
// same strings regardless of language.
type ErrorCode string

// Known error codes — mirror leash-sdk-ts and leash-sdk-python exactly.
const (
	CodeNoAPIKey                 = ErrorCode("NO_API_KEY")
	CodeNoRequestServerConstruct = ErrorCode("NO_REQUEST_SERVER_CONSTRUCT")
	CodeBrowserModeUnsupported   = ErrorCode("BROWSER_MODE_UNSUPPORTED")
	CodeUnauthorized             = ErrorCode("UNAUTHORIZED")
	CodeNoAuthContext            = ErrorCode("NO_AUTH_CONTEXT")
	CodeIntegrationNotEnabled    = ErrorCode("INTEGRATION_NOT_ENABLED")
	CodeIntegrationError         = ErrorCode("INTEGRATION_ERROR")
	CodeUpgradeRequired          = ErrorCode("UPGRADE_REQUIRED")
	CodeNetworkError             = ErrorCode("NETWORK_ERROR")
	CodeKeyNotDeclared           = ErrorCode("KEY_NOT_DECLARED")
	CodeInvalidKey               = ErrorCode("INVALID_KEY")
	CodeSourceResyncFailed       = ErrorCode("SOURCE_RESYNC_FAILED")
	CodeEnvFetchError            = ErrorCode("ENV_FETCH_ERROR")
)

// LeashError is the structured error type returned by every SDK call site.
//
// Use [errors.As] to recover the concrete value:
//
//	var lerr *leash.LeashError
//	if errors.As(err, &lerr) {
//	    switch lerr.Code {
//	    case leash.CodeUpgradeRequired: ...
//	    }
//	}
//
// Convenience predicates ([IsPlanBlock], [IsUnauthorized], etc.) cover the
// common branches.
type LeashError struct {
	// Code is the stable machine-readable identifier — switch on this.
	Code ErrorCode
	// Message is the human-readable summary.
	Message string
	// Action is an optional remediation hint shown after Message.
	Action string
	// SeeAlso is an optional URL for further reading.
	SeeAlso string
	// Status is the originating HTTP status code (0 if not from a response).
	Status int
	// Cause is the underlying error this was wrapped from, when applicable.
	Cause error
}

// Error implements the error interface.
func (e *LeashError) Error() string {
	if e == nil {
		return "<nil>"
	}
	out := fmt.Sprintf("leash: %s", e.Message)
	if e.Action != "" {
		out += "\n  Fix: " + e.Action
	}
	if e.SeeAlso != "" {
		out += "\n  See: " + e.SeeAlso
	}
	return out
}

// Unwrap returns the underlying cause, if any, for [errors.Is] / [errors.As].
func (e *LeashError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

// ---------------------------------------------------------------------------
// Predicates
// ---------------------------------------------------------------------------

// codeIs reports whether err is (or wraps) a [LeashError] with the given code.
func codeIs(err error, code ErrorCode) bool {
	var lerr *LeashError
	if !errors.As(err, &lerr) {
		return false
	}
	return lerr.Code == code
}

// IsPlanBlock reports whether err is a plan/billing block (HTTP 402).
//
// Use this to surface an upgrade CTA in your UI without parsing message text.
func IsPlanBlock(err error) bool {
	return codeIs(err, CodeUpgradeRequired)
}

// IsUpgradeRequired is an alias of [IsPlanBlock] matching the TS naming.
func IsUpgradeRequired(err error) bool {
	return IsPlanBlock(err)
}

// IsConnectionRequired reports whether err indicates the user has not yet
// connected the integration (the platform returns 403 + INTEGRATION_NOT_ENABLED).
func IsConnectionRequired(err error) bool {
	return codeIs(err, CodeIntegrationNotEnabled)
}

// IsUnauthorized reports whether err is an auth failure (HTTP 401).
func IsUnauthorized(err error) bool {
	return codeIs(err, CodeUnauthorized)
}

// IsKeyNotDeclared reports whether err is a "missing env-var" signal.
//
// Note: [EnvNamespace.Get] returns (nil, nil) for missing keys, so this
// predicate is rare — it fires only when callers reach out to the lower-level
// surface. Kept for parity with TS.
func IsKeyNotDeclared(err error) bool {
	return codeIs(err, CodeKeyNotDeclared)
}

// IsNetworkError reports whether err originated below the HTTP layer
// (DNS failure, refused connection, TLS error, etc.).
func IsNetworkError(err error) bool {
	return codeIs(err, CodeNetworkError)
}
