package integrations

import (
	"context"
	"encoding/json"
)

// GmailMessage is a minimal message reference returned by Gmail list calls.
type GmailMessage struct {
	ID       string `json:"id"`
	ThreadID string `json:"threadId"`
}

// GmailMessageList is the paginated list shape returned by ListMessages /
// SearchMessages.
type GmailMessageList struct {
	Messages           []GmailMessage `json:"messages,omitempty"`
	NextPageToken      string         `json:"nextPageToken,omitempty"`
	ResultSizeEstimate int            `json:"resultSizeEstimate,omitempty"`
}

// GmailLabel is a single Gmail label.
type GmailLabel struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

// GmailLabelList is the shape returned by [Gmail.ListLabels].
type GmailLabelList struct {
	Labels []GmailLabel `json:"labels"`
}

// GmailListParams configures a [Gmail.ListMessages] call.
//
// All fields are optional — leave the struct zero-valued (or pass nil) to use
// platform defaults.
type GmailListParams struct {
	Query      string   `json:"query,omitempty"`
	MaxResults int      `json:"maxResults,omitempty"`
	LabelIDs   []string `json:"labelIds,omitempty"`
	PageToken  string   `json:"pageToken,omitempty"`
}

// GmailSendMessageParams configures a [Gmail.SendMessage] call.
//
// To, Subject, and Body are required.
type GmailSendMessageParams struct {
	To      string `json:"to"`
	Subject string `json:"subject"`
	Body    string `json:"body"`
	CC      string `json:"cc,omitempty"`
	BCC     string `json:"bcc,omitempty"`
}

// GmailMessageFormat is the response-detail level for [Gmail.GetMessage].
//
// Mirrors the Gmail REST API enum: "full" | "metadata" | "minimal" | "raw".
// Empty string uses the platform default ("full").
type GmailMessageFormat string

const (
	GmailFormatFull     GmailMessageFormat = "full"
	GmailFormatMetadata GmailMessageFormat = "metadata"
	GmailFormatMinimal  GmailMessageFormat = "minimal"
	GmailFormatRaw      GmailMessageFormat = "raw"
)

// Gmail is the typed Gmail provider client. Obtain via Namespace.Gmail().
type Gmail struct {
	caller Caller
}

const gmailProvider = "gmail"

// ListMessages returns messages from the user's mailbox.
//
// Pass nil for params to use platform defaults.
func (g *Gmail) ListMessages(ctx context.Context, params *GmailListParams) (*GmailMessageList, error) {
	raw, err := g.caller.IntegrationsCall(ctx, gmailProvider, "list-messages", paramsOrEmpty(params))
	if err != nil {
		return nil, err
	}
	var out GmailMessageList
	if err := unmarshalIfPresent(raw, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetMessage retrieves a single message by ID.
//
// Pass an empty [GmailMessageFormat] to use the platform default ("full").
func (g *Gmail) GetMessage(ctx context.Context, messageID string, format GmailMessageFormat) (json.RawMessage, error) {
	body := map[string]any{"messageId": messageID}
	if format != "" {
		body["format"] = string(format)
	}
	return g.caller.IntegrationsCall(ctx, gmailProvider, "get-message", body)
}

// SendMessage sends an email message and returns the platform's raw response.
func (g *Gmail) SendMessage(ctx context.Context, params GmailSendMessageParams) (json.RawMessage, error) {
	return g.caller.IntegrationsCall(ctx, gmailProvider, "send-message", params)
}

// SearchMessages runs a Gmail-syntax search query. Pass 0 for maxResults to
// use the platform default.
func (g *Gmail) SearchMessages(ctx context.Context, query string, maxResults int) (*GmailMessageList, error) {
	body := map[string]any{"query": query}
	if maxResults > 0 {
		body["maxResults"] = maxResults
	}
	raw, err := g.caller.IntegrationsCall(ctx, gmailProvider, "search-messages", body)
	if err != nil {
		return nil, err
	}
	var out GmailMessageList
	if err := unmarshalIfPresent(raw, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ListLabels returns all labels in the user's mailbox.
func (g *Gmail) ListLabels(ctx context.Context) (*GmailLabelList, error) {
	raw, err := g.caller.IntegrationsCall(ctx, gmailProvider, "list-labels", nil)
	if err != nil {
		return nil, err
	}
	var out GmailLabelList
	if err := unmarshalIfPresent(raw, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetProfile returns the authenticated user's Gmail profile (emailAddress,
// messagesTotal, threadsTotal, historyId) as a raw envelope.
func (g *Gmail) GetProfile(ctx context.Context) (json.RawMessage, error) {
	return g.caller.IntegrationsCall(ctx, gmailProvider, "get-profile", nil)
}
