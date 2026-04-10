// Package leash provides a Go client for the Leash platform integrations API.
//
// The SDK communicates with the Leash platform proxy which handles OAuth tokens
// and routes requests to Google Gmail, Calendar, and Drive APIs.
package leash

import (
	"encoding/json"
	"fmt"
)

// DefaultPlatformURL is the default Leash platform base URL.
const DefaultPlatformURL = "https://leash.build"

// Error represents an error returned by the Leash platform API.
type Error struct {
	// Message is the human-readable error description.
	Message string
	// Code is an optional machine-readable error code (e.g. "not_connected").
	Code string
	// ConnectURL is provided when the user needs to connect an integration.
	ConnectURL string
}

func (e *Error) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("leash: %s (code: %s)", e.Message, e.Code)
	}
	return fmt.Sprintf("leash: %s", e.Message)
}

// apiResponse is the standard envelope returned by all platform API endpoints.
type apiResponse struct {
	Success    bool            `json:"success"`
	Data       json.RawMessage `json:"data"`
	ErrorMsg   string          `json:"error"`
	Code       string          `json:"code"`
	ConnectURL string          `json:"connectUrl"`
}

// ConnectionStatus represents the status of a provider connection.
type ConnectionStatus struct {
	ProviderID string `json:"providerId"`
	Status     string `json:"status"`
	Email      string `json:"email,omitempty"`
	ExpiresAt  string `json:"expiresAt,omitempty"`
}

// --- Gmail types ---

// ListMessagesParams configures a Gmail ListMessages request.
type ListMessagesParams struct {
	// Query is a Gmail search query (e.g. "from:user@example.com").
	Query string `json:"query,omitempty"`
	// MaxResults is the maximum number of messages to return.
	MaxResults int `json:"maxResults,omitempty"`
	// LabelIDs filters messages by label (e.g. []string{"INBOX"}).
	LabelIDs []string `json:"labelIds,omitempty"`
	// PageToken is the token for fetching the next page of results.
	PageToken string `json:"pageToken,omitempty"`
}

// GetMessageParams configures a Gmail GetMessage request.
type GetMessageParams struct {
	// MessageID is the ID of the message to retrieve. Required.
	MessageID string `json:"messageId"`
	// Format controls the response format: "full", "metadata", "minimal", or "raw".
	Format string `json:"format,omitempty"`
}

// SendMessageParams configures a Gmail SendMessage request.
type SendMessageParams struct {
	// To is the recipient email address. Required.
	To string `json:"to"`
	// Subject is the email subject line. Required.
	Subject string `json:"subject"`
	// Body is the email body text. Required.
	Body string `json:"body"`
	// CC is an optional CC recipient.
	CC string `json:"cc,omitempty"`
	// BCC is an optional BCC recipient.
	BCC string `json:"bcc,omitempty"`
}

// SearchMessagesParams configures a Gmail SearchMessages request.
type SearchMessagesParams struct {
	// Query is the Gmail search query. Required.
	Query string `json:"query"`
	// MaxResults is the maximum number of results to return.
	MaxResults int `json:"maxResults,omitempty"`
}

// --- Calendar types ---

// ListEventsParams configures a Calendar ListEvents request.
type ListEventsParams struct {
	// CalendarID is the calendar identifier (defaults to "primary" on the server).
	CalendarID string `json:"calendarId,omitempty"`
	// TimeMin is the lower bound for event start time (RFC 3339).
	TimeMin string `json:"timeMin,omitempty"`
	// TimeMax is the upper bound for event start time (RFC 3339).
	TimeMax string `json:"timeMax,omitempty"`
	// MaxResults is the maximum number of events to return.
	MaxResults int `json:"maxResults,omitempty"`
	// SingleEvents expands recurring events into individual instances.
	SingleEvents *bool `json:"singleEvents,omitempty"`
	// OrderBy specifies the order of results (e.g. "startTime", "updated").
	OrderBy string `json:"orderBy,omitempty"`
}

// EventDateTime represents a start or end time for a calendar event.
type EventDateTime struct {
	// DateTime is the RFC 3339 timestamp (for timed events).
	DateTime string `json:"dateTime,omitempty"`
	// Date is the date string (for all-day events, format "2006-01-02").
	Date string `json:"date,omitempty"`
	// TimeZone is the IANA time zone (e.g. "America/New_York").
	TimeZone string `json:"timeZone,omitempty"`
}

// Attendee represents a calendar event attendee.
type Attendee struct {
	Email string `json:"email"`
}

// CreateEventParams configures a Calendar CreateEvent request.
type CreateEventParams struct {
	// CalendarID is the calendar identifier (defaults to "primary" on the server).
	CalendarID string `json:"calendarId,omitempty"`
	// Summary is the event title. Required.
	Summary string `json:"summary"`
	// Description is an optional event description.
	Description string `json:"description,omitempty"`
	// Location is an optional event location.
	Location string `json:"location,omitempty"`
	// Start is the event start time. Required.
	Start EventDateTime `json:"start"`
	// End is the event end time. Required.
	End EventDateTime `json:"end"`
	// Attendees is an optional list of attendees.
	Attendees []Attendee `json:"attendees,omitempty"`
}

// GetEventParams configures a Calendar GetEvent request.
type GetEventParams struct {
	// EventID is the event identifier. Required.
	EventID string `json:"eventId"`
	// CalendarID is the calendar identifier.
	CalendarID string `json:"calendarId,omitempty"`
}

// --- Drive types ---

// ListFilesParams configures a Drive ListFiles request.
type ListFilesParams struct {
	// Query is a Google Drive search query.
	Query string `json:"query,omitempty"`
	// MaxResults is the maximum number of files to return.
	MaxResults int `json:"maxResults,omitempty"`
	// FolderID restricts results to files within a specific folder.
	FolderID string `json:"folderId,omitempty"`
}

// SearchFilesParams configures a Drive SearchFiles request.
type SearchFilesParams struct {
	// Query is the search query. Required.
	Query string `json:"query"`
	// MaxResults is the maximum number of results to return.
	MaxResults int `json:"maxResults,omitempty"`
}
