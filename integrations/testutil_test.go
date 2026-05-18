package integrations

import (
	"context"
	"encoding/json"
)

// recordedCall captures a single integration call dispatched through the
// stub caller — useful for asserting wire shapes in tests.
type recordedCall struct {
	Provider string
	Action   string
	Body     any
}

// stubCaller is an in-memory [Caller] that returns canned responses keyed by
// (provider, action). It records every call so tests can assert the exact
// body sent.
type stubCaller struct {
	responses map[string]stubResponse
	Calls     []recordedCall
}

type stubResponse struct {
	body []byte
	err  error
}

func newStub(responses map[string]stubResponse) *stubCaller {
	return &stubCaller{responses: responses}
}

func (s *stubCaller) IntegrationsCall(_ context.Context, provider, action string, body any) (json.RawMessage, error) {
	s.Calls = append(s.Calls, recordedCall{Provider: provider, Action: action, Body: body})
	key := provider + "/" + action
	r, ok := s.responses[key]
	if !ok {
		return nil, &stubError{msg: "no response stubbed for " + key}
	}
	if r.err != nil {
		return nil, r.err
	}
	return r.body, nil
}

type stubError struct{ msg string }

func (e *stubError) Error() string { return e.msg }

// helper to wrap bytes for stubResponse
func okJSON(s string) stubResponse {
	return stubResponse{body: []byte(s)}
}
