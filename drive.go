package leash

import "encoding/json"

// DriveClient provides methods for interacting with Google Drive via the
// Leash platform proxy. Obtain one by calling LeashIntegrations.Drive().
type DriveClient struct {
	client *LeashIntegrations
}

// ListFiles returns files from the user's Drive.
//
// Pass nil for params to use server defaults.
func (d *DriveClient) ListFiles(params *ListFilesParams) (json.RawMessage, error) {
	return d.client.call("google_drive", "list-files", params)
}

// GetFile retrieves file metadata by ID.
func (d *DriveClient) GetFile(fileID string) (json.RawMessage, error) {
	body := map[string]string{"fileId": fileID}
	return d.client.call("google_drive", "get-file", body)
}

// SearchFiles searches for files using a query string.
func (d *DriveClient) SearchFiles(query string, maxResults int) (json.RawMessage, error) {
	body := SearchFilesParams{Query: query, MaxResults: maxResults}
	return d.client.call("google_drive", "search-files", body)
}
