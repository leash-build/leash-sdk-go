# leash-sdk-go

Go SDK for the [Leash](https://leash.build) platform integrations API.

Provides typed clients for Gmail, Google Calendar, and Google Drive through the Leash platform proxy.

## Install

```bash
go get github.com/leash-build/leash-sdk-go
```

## Quick Start

```go
package main

import (
    "encoding/json"
    "fmt"
    "log"

    leash "github.com/leash-build/leash-sdk-go"
)

func main() {
    client := leash.New("your-auth-token")

    // Check if Gmail is connected
    if !client.IsConnected("gmail") {
        url := client.GetConnectURL("gmail", "https://myapp.com/callback")
        fmt.Println("Connect Gmail:", url)
        return
    }

    // List recent emails
    data, err := client.Gmail().ListMessages(&leash.ListMessagesParams{
        MaxResults: 5,
    })
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(string(data))
}
```

## Usage

### Initialize

```go
// Default platform URL (https://leash.build)
client := leash.New("your-auth-token")

// Custom platform URL
client := leash.NewWithURL("your-auth-token", "https://your-instance.example.com")

// Custom HTTP client
client.HTTPClient = &http.Client{Timeout: 30 * time.Second}
```

### Gmail

```go
gmail := client.Gmail()

// List messages
messages, err := gmail.ListMessages(&leash.ListMessagesParams{
    Query:      "from:boss@company.com",
    MaxResults: 10,
    LabelIDs:   []string{"INBOX"},
})

// Get a single message
msg, err := gmail.GetMessage("message-id-123", "full")

// Send a message
sent, err := gmail.SendMessage(leash.SendMessageParams{
    To:      "user@example.com",
    Subject: "Hello from Go",
    Body:    "This email was sent via the Leash SDK.",
    CC:      "cc@example.com",
})

// Search messages
results, err := gmail.SearchMessages("subject:important", 20)

// List labels
labels, err := gmail.ListLabels()
```

### Google Calendar

```go
cal := client.Calendar()

// List calendars
calendars, err := cal.ListCalendars()

// List events
events, err := cal.ListEvents(&leash.ListEventsParams{
    CalendarID: "primary",
    TimeMin:    "2026-04-01T00:00:00Z",
    TimeMax:    "2026-04-30T23:59:59Z",
    MaxResults: 25,
})

// Create an event
event, err := cal.CreateEvent(leash.CreateEventParams{
    Summary:     "Team Standup",
    Description: "Daily sync",
    Start:       leash.EventDateTime{DateTime: "2026-04-10T09:00:00-04:00"},
    End:         leash.EventDateTime{DateTime: "2026-04-10T09:30:00-04:00"},
    Attendees:   []leash.Attendee{{Email: "team@example.com"}},
})

// Get an event
ev, err := cal.GetEvent("event-id-123", "primary")
```

### Google Drive

```go
drive := client.Drive()

// List files
files, err := drive.ListFiles(&leash.ListFilesParams{
    MaxResults: 10,
    FolderID:   "folder-id-123",
})

// Get file metadata
file, err := drive.GetFile("file-id-123")

// Search files
results, err := drive.SearchFiles("name contains 'report'", 20)
```

### Connections

```go
// Check if a provider is connected
connected := client.IsConnected("gmail")

// Get all connections
connections, err := client.GetConnections()
for _, c := range connections {
    fmt.Printf("%s: %s\n", c.ProviderID, c.Status)
}

// Get OAuth connect URL for UI
url := client.GetConnectURL("gmail", "https://myapp.com/settings")
```

### Generic Call

For providers or actions not yet covered by typed methods:

```go
data, err := client.Call("slack", "send-message", map[string]any{
    "channel": "#general",
    "text":    "Hello from Leash!",
})
```

### Error Handling

```go
data, err := client.Gmail().ListMessages(nil)
if err != nil {
    var leashErr *leash.Error
    if errors.As(err, &leashErr) {
        if leashErr.Code == "not_connected" {
            fmt.Println("Connect at:", leashErr.ConnectURL)
        }
    }
    log.Fatal(err)
}
```

## Response Format

All methods return `(json.RawMessage, error)`. The raw JSON lets you unmarshal
into your own types or work with dynamic data:

```go
data, err := client.Gmail().ListMessages(nil)
if err != nil {
    log.Fatal(err)
}

var result struct {
    Messages []struct {
        ID      string `json:"id"`
        Snippet string `json:"snippet"`
    } `json:"messages"`
    NextPageToken string `json:"nextPageToken"`
}
json.Unmarshal(data, &result)
```

## Requirements

- Go 1.21+
- No external dependencies (standard library only)

## License

MIT
