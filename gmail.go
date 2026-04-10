package leash

import "encoding/json"

// GmailClient provides methods for interacting with Gmail via the Leash
// platform proxy. Obtain one by calling LeashIntegrations.Gmail().
type GmailClient struct {
	client *LeashIntegrations
}

// ListMessages returns messages from the user's mailbox.
//
// Pass nil for params to use server defaults.
func (g *GmailClient) ListMessages(params *ListMessagesParams) (json.RawMessage, error) {
	return g.client.call("gmail", "list-messages", params)
}

// GetMessage retrieves a single message by ID.
//
// Format controls the response format: "full", "metadata", "minimal", or "raw".
// Pass an empty string to use the server default ("full").
func (g *GmailClient) GetMessage(messageID string, format string) (json.RawMessage, error) {
	body := GetMessageParams{MessageID: messageID, Format: format}
	return g.client.call("gmail", "get-message", body)
}

// SendMessage sends an email message.
func (g *GmailClient) SendMessage(params SendMessageParams) (json.RawMessage, error) {
	return g.client.call("gmail", "send-message", params)
}

// SearchMessages searches messages using a Gmail query string.
func (g *GmailClient) SearchMessages(query string, maxResults int) (json.RawMessage, error) {
	body := SearchMessagesParams{Query: query, MaxResults: maxResults}
	return g.client.call("gmail", "search-messages", body)
}

// ListLabels returns all labels in the user's mailbox.
func (g *GmailClient) ListLabels() (json.RawMessage, error) {
	return g.client.call("gmail", "list-labels", nil)
}
