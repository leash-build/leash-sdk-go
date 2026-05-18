// Package integrations carries the typed provider wrappers (Gmail, Calendar,
// Drive, Linear) plus a generic [IntegrationCaller] escape hatch for providers
// the SDK doesn't model yet.
//
// Construct one through the root [github.com/leash-build/leash-sdk-go.Client]:
//
//	gm, err := client.Integrations().Gmail().ListMessages(ctx, &leash.GmailListParams{...})
//
// The integrations namespace shares the same auth context as the rest of the
// client — cookie + LEASH_API_KEY — and routes every call through the
// Leash platform proxy.
package integrations

import (
	"context"
	"encoding/json"
)

// Caller is the minimal contract the namespace needs from the parent client's
// transport. Defined as an interface so callers can mock it without pulling
// in the full HTTP stack — most users will never reference it directly.
type Caller interface {
	IntegrationsCall(ctx context.Context, provider, action string, body any) (json.RawMessage, error)
}

// Namespace is the typed integrations surface returned by Client.Integrations().
//
// Each method returns a typed provider client; the generic [Provider] method
// is the escape hatch for providers without typed wrappers yet (Slack, GitHub,
// HubSpot, Jira, Gong, Slite, BigQuery, …).
type Namespace struct {
	caller Caller
}

// New returns a [Namespace] bound to the given transport.
//
// Most callers should use Client.Integrations() instead — this constructor is
// exposed for advanced use (e.g. composing a sub-client around a stubbed
// transport in tests).
func New(caller Caller) *Namespace {
	return &Namespace{caller: caller}
}

// Gmail returns the typed Gmail provider client.
func (n *Namespace) Gmail() *Gmail { return &Gmail{caller: n.caller} }

// Calendar returns the typed Google Calendar provider client.
func (n *Namespace) Calendar() *Calendar { return &Calendar{caller: n.caller} }

// GoogleCalendar is an alias of [Namespace.Calendar] matching the long-form
// provider id the TS SDK exposes.
func (n *Namespace) GoogleCalendar() *Calendar { return n.Calendar() }

// Drive returns the typed Google Drive provider client.
func (n *Namespace) Drive() *Drive { return &Drive{caller: n.caller} }

// GoogleDrive is an alias of [Namespace.Drive] matching the long-form
// provider id the TS SDK exposes.
func (n *Namespace) GoogleDrive() *Drive { return n.Drive() }

// Linear returns the typed Linear provider client.
func (n *Namespace) Linear() *Linear { return &Linear{caller: n.caller} }

// Provider returns an [IntegrationCaller] bound to the given provider name.
//
// Use this for providers without a typed wrapper yet — slack, github,
// hubspot, jira, gong, slite, bigquery, custom MCP servers, etc.
//
//	slack := client.Integrations().Provider("slack")
//	res, err := slack.Call(ctx, "post-message", map[string]any{"channel":"#general", "text":"hi"})
func (n *Namespace) Provider(name string) *IntegrationCaller {
	return &IntegrationCaller{caller: n.caller, name: name}
}

// IntegrationCaller is a generic provider invoker — no typed shape, just a
// pass-through of action + params. Returned by [Namespace.Provider].
type IntegrationCaller struct {
	caller Caller
	name   string
}

// Name returns the provider id this caller is bound to.
func (c *IntegrationCaller) Name() string { return c.name }

// Call POSTs to /api/integrations/{name}/{action} and returns the raw
// response data as a [json.RawMessage].
func (c *IntegrationCaller) Call(ctx context.Context, action string, body any) (json.RawMessage, error) {
	return c.caller.IntegrationsCall(ctx, c.name, action, body)
}
