# leash-sdk-go

Go SDK for Leash-hosted integrations.

It provides typed provider clients plus generic calls through the Leash platform proxy.

## Install

```bash
go get github.com/leash-build/leash-sdk-go
```

## Quick Start

```go
package main

import (
	"fmt"
	"log"

	leash "github.com/leash-build/leash-sdk-go"
)

func main() {
	client := leash.New("your-platform-jwt")

	if !client.IsConnected("gmail") {
		fmt.Println(client.GetConnectURL("gmail", "https://myapp.example.com/settings"))
		return
	}

	data, err := client.Gmail().ListMessages(&leash.ListMessagesParams{MaxResults: 5})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(string(data))
}
```

## Default Platform URL

- `https://leash.build`

The SDK calls routes such as:

- `/api/integrations/{provider}/{action}`
- `/api/integrations/connections`
- `/api/apps/env`
- `/api/mcp/run`

## Capabilities

- Gmail
- Google Calendar
- Google Drive
- connection status lookup
- connect URL generation
- generic provider calls
- custom integration calls
- app env fetch and caching

## Example Initialization

```go
client := leash.New("your-platform-jwt")

client = leash.NewWithURL("your-platform-jwt", "https://staging.leash.build")
client.APIKey = "optional-app-api-key"
```

## Server Auth

The SDK provides helpers for authenticating users on the server side by reading
the `leash-auth` cookie set by the Leash platform.

```go
func meHandler(w http.ResponseWriter, r *http.Request) {
    user, err := leash.GetLeashUser(r)
    if err != nil { http.Error(w, "unauthorized", 401); return }
    json.NewEncoder(w).Encode(user)
}
```

## MCP Calls

Execute MCP-backed tools through the platform:

```go
result, err := client.RunMCP("@some/mcp-package", "tool-name", map[string]any{"key": "value"})
```

## Notes

- `auth_token` should be a valid Leash platform JWT.
- `APIKey` is optional, but useful for app-scoped env access.
- The SDK delegates OAuth token handling to the Leash platform.

## License

Apache-2.0
