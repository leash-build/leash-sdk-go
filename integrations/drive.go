package integrations

import (
	"context"
	"encoding/json"
)

// DriveFile is the metadata view of a Google Drive file.
type DriveFile struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	MimeType       string   `json:"mimeType"`
	Size           string   `json:"size,omitempty"`
	CreatedTime    string   `json:"createdTime,omitempty"`
	ModifiedTime   string   `json:"modifiedTime,omitempty"`
	Parents        []string `json:"parents,omitempty"`
	WebViewLink    string   `json:"webViewLink,omitempty"`
	WebContentLink string   `json:"webContentLink,omitempty"`
}

// DriveFileList is the shape returned by [Drive.ListFiles] / [Drive.SearchFiles].
type DriveFileList struct {
	Files         []DriveFile `json:"files"`
	NextPageToken string      `json:"nextPageToken,omitempty"`
}

// DriveListFilesParams configures a [Drive.ListFiles] call.
type DriveListFilesParams struct {
	Query      string `json:"query,omitempty"`
	MaxResults int    `json:"maxResults,omitempty"`
	FolderID   string `json:"folderId,omitempty"`
}

// DriveUploadFileParams configures a [Drive.UploadFile] call.
//
// Name, Content, and MimeType are required.
type DriveUploadFileParams struct {
	Name     string `json:"name"`
	Content  string `json:"content"`
	MimeType string `json:"mimeType"`
	ParentID string `json:"parentId,omitempty"`
}

// Drive is the typed Google Drive provider client.
type Drive struct {
	caller Caller
}

const driveProvider = "google_drive"

// ListFiles returns files from the user's Drive. Pass nil for params to use
// platform defaults.
func (d *Drive) ListFiles(ctx context.Context, params *DriveListFilesParams) (*DriveFileList, error) {
	raw, err := d.caller.IntegrationsCall(ctx, driveProvider, "list-files", paramsOrEmpty(params))
	if err != nil {
		return nil, err
	}
	var out DriveFileList
	if err := unmarshalIfPresent(raw, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetFile returns metadata for a single file by ID.
func (d *Drive) GetFile(ctx context.Context, fileID string) (*DriveFile, error) {
	raw, err := d.caller.IntegrationsCall(ctx, driveProvider, "get-file", map[string]any{"fileId": fileID})
	if err != nil {
		return nil, err
	}
	var out DriveFile
	if err := unmarshalIfPresent(raw, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// DownloadFile returns the file's raw content envelope from the platform.
//
// The platform's response shape depends on the file type — we surface it as
// [json.RawMessage] so callers can parse whichever envelope the platform
// returns (base64-encoded bytes for binaries, plain text for text files).
func (d *Drive) DownloadFile(ctx context.Context, fileID string) (json.RawMessage, error) {
	return d.caller.IntegrationsCall(ctx, driveProvider, "download-file", map[string]any{"fileId": fileID})
}

// CreateFolder creates a new folder. Pass an empty parentID to create at the
// root of My Drive.
func (d *Drive) CreateFolder(ctx context.Context, name, parentID string) (*DriveFile, error) {
	body := map[string]any{"name": name}
	if parentID != "" {
		body["parentId"] = parentID
	}
	raw, err := d.caller.IntegrationsCall(ctx, driveProvider, "create-folder", body)
	if err != nil {
		return nil, err
	}
	var out DriveFile
	if err := unmarshalIfPresent(raw, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// UploadFile uploads a new file with the given content.
func (d *Drive) UploadFile(ctx context.Context, params DriveUploadFileParams) (*DriveFile, error) {
	raw, err := d.caller.IntegrationsCall(ctx, driveProvider, "upload-file", params)
	if err != nil {
		return nil, err
	}
	var out DriveFile
	if err := unmarshalIfPresent(raw, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// DeleteFile permanently deletes a file by ID.
func (d *Drive) DeleteFile(ctx context.Context, fileID string) (json.RawMessage, error) {
	return d.caller.IntegrationsCall(ctx, driveProvider, "delete-file", map[string]any{"fileId": fileID})
}

// SearchFiles runs a Drive-syntax search query. Pass 0 for maxResults to use
// the platform default.
func (d *Drive) SearchFiles(ctx context.Context, query string, maxResults int) (*DriveFileList, error) {
	body := map[string]any{"query": query}
	if maxResults > 0 {
		body["maxResults"] = maxResults
	}
	raw, err := d.caller.IntegrationsCall(ctx, driveProvider, "search-files", body)
	if err != nil {
		return nil, err
	}
	var out DriveFileList
	if err := unmarshalIfPresent(raw, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
