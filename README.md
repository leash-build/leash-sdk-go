# leash-sdk-go

Go SDK for the [Leash](https://leash.build) platform. One unified `Client`
that handles auth, runtime env-vars, and integrations through the Leash
platform proxy.

Stdlib-only — no third-party dependencies.

## Install

```bash
go get github.com/leash-build/leash-sdk-go@v0.4.0
```

Requires Go 1.21 or newer.

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"
    "net/http"

    leash "github.com/leash-build/leash-sdk-go"
)

func handler(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()

    client, err := leash.New(r)
    if err != nil { log.Fatal(err) }

    user, _ := client.Auth().User(ctx)
    if user == nil {
        http.Error(w, "Not signed in", http.StatusUnauthorized)
        return
    }

    msgs, err := client.Integrations().Gmail().ListMessages(ctx,
        &leash.GmailListParams{MaxResults: 5})
    if err != nil { log.Fatal(err) }

    fmt.Fprintf(w, "Hi %s — %d messages\n", user.Name, len(msgs.Messages))
}
```

(The package imports its provider types directly — `leash.GmailListParams` is
a re-import alias for `integrations.GmailListParams`. Both spellings work.)

## Concepts

A `Client` carries three resolved bits of state, all derived at construction
time from your *http.Request:

- **Cookie** (`leash-auth`) — forwarded to integration calls so the platform
  resolves the right user.
- **API key** (`LEASH_API_KEY` env var, `WithAPIKey` option, or
  `NewFromAPIKey`) — sent as `X-API-Key` to integration calls and as
  `Authorization: Bearer` to env-var calls.
- **Platform URL** — defaults to `https://leash.build`; override with
  `LEASH_PLATFORM_URL` env var or `WithPlatformURL` option.

`Authorization: Bearer` headers on the inbound request are extracted but
**never forwarded** on integration calls — that path is reserved for the
platform's user-JWT verifier and would short-circuit the API-key check.

## Constructors

```go
// From any *http.Request — net/http, chi, gin (c.Request), echo (c.Request()),
// fiber (c.Context().Request, after a small adapter)
client, err := leash.New(r)

// Explicit JWT (CLI / agent flow)
client, err := leash.NewWithToken("eyJhbGciOi...")

// Server-to-server with an API key
client, err := leash.NewFromAPIKey(os.Getenv("LEASH_API_KEY"))
```

All three accept `leash.Option` variadic args:

```go
client, err := leash.New(r,
    leash.WithPlatformURL("https://staging.leash.build"),
    leash.WithHTTPClient(myHTTPClient),
    leash.WithAPIKey("lsk_live_…"),
)
```

## Auth

```go
user, err := client.Auth().User(ctx)
// err == nil && user == nil → not signed in (no/invalid cookie)
// err == nil && user != nil → signed in
// err != nil               → infrastructure error
```

The free function `leash.GetLeashUser(r)` is also available for handlers that
just want the user without constructing a full client.

## Runtime env-vars

```go
val, err := client.Env().Get(ctx, "OPENAI_API_KEY")
if err != nil { return err }
if val == nil { /* key not declared / not found */ }

// Bypass the 60s in-memory cache
fresh, err := client.Env().Get(ctx, "STRIPE_SECRET_KEY", leash.EnvFresh())

// Bulk
m, err := client.Env().GetMany(ctx, []string{"A", "B", "C"})
```

`Get` returns `(*string, error)`. A nil value with a nil error is the
"not declared / not found" signal — match the Python `Optional[str]` pattern
to branch with `if val == nil`.

## Integrations

Typed providers cover Gmail, Google Calendar, Google Drive, and Linear. Every
verb that exists on the TypeScript SDK is available here in PascalCase:

```go
gm, err := client.Integrations().Gmail().ListMessages(ctx,
    &leash.GmailListParams{MaxResults: 5})

events, err := client.Integrations().Calendar().ListEvents(ctx,
    &leash.CalendarListEventsParams{
        TimeMin: "2026-01-01T00:00:00Z",
    })

files, err := client.Integrations().Drive().ListFiles(ctx,
    &leash.DriveListFilesParams{Query: "name contains 'invoice'"})

issues, err := client.Integrations().Linear().ListIssues(ctx,
    &leash.LinearListIssuesFilter{StateType: leash.LinearStateStarted})
```

Aliases match the TS surface — `GoogleCalendar()` and `GoogleDrive()` return
the same provider as `Calendar()` / `Drive()`.

### Generic provider escape hatch

For providers without typed wrappers yet (Slack, GitHub, HubSpot, Jira, Gong,
Slite, BigQuery, etc.):

```go
slack := client.Integrations().Provider("slack")
raw, err := slack.Call(ctx, "post-message", map[string]any{
    "channel": "#general",
    "text":    "Deploy succeeded",
})
```

`raw` is a `json.RawMessage` — unmarshal into whichever shape the provider
returns.

## Errors

Every call site returns `*leash.LeashError`. Recover the concrete value with
`errors.As`:

```go
_, err := client.Integrations().Gmail().ListMessages(ctx, nil)
var lerr *leash.LeashError
if errors.As(err, &lerr) {
    fmt.Println(lerr.Code, lerr.Message, lerr.Status)
}
```

Predicates cover the common branches without parsing strings:

```go
leash.IsPlanBlock(err)           // 402 — surface upgrade CTA
leash.IsConnectionRequired(err)  // 403 — surface "Connect Gmail" button
leash.IsUnauthorized(err)        // 401 — re-auth flow
leash.IsNetworkError(err)        // sub-HTTP failure
```

## Framework examples

### `net/http`

```go
http.HandleFunc("/issues", func(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    client, _ := leash.New(r)
    res, err := client.Integrations().Linear().ListIssues(ctx,
        &leash.LinearListIssuesFilter{Limit: 25})
    if err != nil { http.Error(w, err.Error(), 500); return }
    json.NewEncoder(w).Encode(res)
})
```

### `chi`

```go
r := chi.NewRouter()
r.Get("/me", func(w http.ResponseWriter, r *http.Request) {
    user, _ := leash.New(r).Auth().User(r.Context())
    json.NewEncoder(w).Encode(user)
})
```

### `gin`

```go
r := gin.Default()
r.GET("/me", func(c *gin.Context) {
    client, _ := leash.New(c.Request)
    user, _ := client.Auth().User(c.Request.Context())
    c.JSON(200, user)
})
```

### `echo`

```go
e := echo.New()
e.GET("/me", func(c echo.Context) error {
    client, _ := leash.New(c.Request())
    user, _ := client.Auth().User(c.Request().Context())
    return c.JSON(200, user)
})
```

## What's not in 0.4 yet

Mirrors the parity gap called out in the Python SDK's 0.4 release:

- **Connection-status listing** (`GetConnections`, `IsConnected`). The 0.3
  surface had `client.IsConnected("gmail")`. In 0.4 you discover this
  implicitly through `IsConnectionRequired(err)` after attempting a call. A
  dedicated `Integrations().Status()` namespace is planned but not yet shipped.
- **`Integrations().GetAccessToken(...)`** and
  **`Integrations().GetCustomMcpConfig(...)`** — these existed in the 0.3
  surface but haven't been ported to the unified `Client` yet. Drop down to
  the [`Transport`](#advanced-transport-access) escape hatch if you need them
  today.
- **Local-dev auth handler** (`CreateDevAuthHandler` in TS, `attachLocalDevHandler`
  in Python). Local-dev cookie issuance currently goes through the dashboard;
  exposing a Go handler is tracked but not yet started.

If you need any of these on the 0.4 surface, open an issue.

## Advanced: transport access

`client.Transport()` exposes the raw HTTP carrier for unusual flows — e.g.
calling a new platform endpoint before a typed wrapper exists:

```go
transport := client.Transport()
// transport.HTTPClient, transport.PlatformURL, transport.APIKey, transport.CookieValue
```

## Testing

```bash
go test ./...        # ~120 cases, all hermetic (httptest, no network)
go vet ./...
staticcheck ./...    # if installed
```

## License

Apache-2.0
