package leash

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestLeashError_String(t *testing.T) {
	e := &LeashError{
		Code:    CodeUpgradeRequired,
		Message: "Need Growth plan",
		Action:  "Upgrade at billing",
		SeeAlso: "https://leash.build/pricing",
	}
	s := e.Error()
	if !strings.Contains(s, "Need Growth plan") {
		t.Errorf("missing message: %q", s)
	}
	if !strings.Contains(s, "Upgrade at billing") {
		t.Errorf("missing action: %q", s)
	}
	if !strings.Contains(s, "https://leash.build/pricing") {
		t.Errorf("missing see also: %q", s)
	}
}

func TestLeashError_NilSafe(t *testing.T) {
	var e *LeashError
	if e.Error() != "<nil>" {
		t.Errorf("expected <nil>, got %q", e.Error())
	}
	if e.Unwrap() != nil {
		t.Errorf("expected nil unwrap, got %v", e.Unwrap())
	}
}

func TestLeashError_Unwrap(t *testing.T) {
	inner := fmt.Errorf("inner")
	e := &LeashError{Code: CodeNetworkError, Message: "boom", Cause: inner}
	if !errors.Is(e, inner) {
		t.Error("errors.Is should reach inner via Unwrap")
	}
}

func TestPredicates(t *testing.T) {
	cases := []struct {
		name      string
		err       error
		isPlan    bool
		isConnReq bool
		isUnauth  bool
		isMissing bool
		isNet     bool
	}{
		{"plan block", &LeashError{Code: CodeUpgradeRequired}, true, false, false, false, false},
		{"connection required", &LeashError{Code: CodeIntegrationNotEnabled}, false, true, false, false, false},
		{"unauthorized", &LeashError{Code: CodeUnauthorized}, false, false, true, false, false},
		{"key not declared", &LeashError{Code: CodeKeyNotDeclared}, false, false, false, true, false},
		{"network", &LeashError{Code: CodeNetworkError}, false, false, false, false, true},
		{"nil", nil, false, false, false, false, false},
		{"non-leash", fmt.Errorf("boom"), false, false, false, false, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if IsPlanBlock(c.err) != c.isPlan {
				t.Errorf("IsPlanBlock: got %v want %v", IsPlanBlock(c.err), c.isPlan)
			}
			if IsConnectionRequired(c.err) != c.isConnReq {
				t.Errorf("IsConnectionRequired: got %v want %v", IsConnectionRequired(c.err), c.isConnReq)
			}
			if IsUnauthorized(c.err) != c.isUnauth {
				t.Errorf("IsUnauthorized: got %v want %v", IsUnauthorized(c.err), c.isUnauth)
			}
			if IsKeyNotDeclared(c.err) != c.isMissing {
				t.Errorf("IsKeyNotDeclared: got %v want %v", IsKeyNotDeclared(c.err), c.isMissing)
			}
			if IsNetworkError(c.err) != c.isNet {
				t.Errorf("IsNetworkError: got %v want %v", IsNetworkError(c.err), c.isNet)
			}
		})
	}
}

func TestIsUpgradeRequired_Alias(t *testing.T) {
	if !IsUpgradeRequired(&LeashError{Code: CodeUpgradeRequired}) {
		t.Error("expected alias to forward to IsPlanBlock")
	}
}
