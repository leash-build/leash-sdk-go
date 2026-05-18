package integrations

import (
	"context"
	"encoding/json"
	"testing"
)

func TestDrive_ListFiles(t *testing.T) {
	stub := newStub(map[string]stubResponse{
		"google_drive/list-files": okJSON(`{"files":[{"id":"f1","name":"hi.txt","mimeType":"text/plain"}]}`),
	})
	got, err := New(stub).Drive().ListFiles(context.Background(), &DriveListFilesParams{Query: "q"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Files) != 1 || got.Files[0].ID != "f1" {
		t.Errorf("unexpected: %+v", got)
	}
}

func TestDrive_GetFile(t *testing.T) {
	stub := newStub(map[string]stubResponse{
		"google_drive/get-file": okJSON(`{"id":"f1","name":"a","mimeType":"text/plain"}`),
	})
	got, err := New(stub).Drive().GetFile(context.Background(), "f1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "a" {
		t.Errorf("unexpected: %+v", got)
	}
}

func TestDrive_DownloadFile(t *testing.T) {
	stub := newStub(map[string]stubResponse{"google_drive/download-file": okJSON(`{"content":"aGk="}`)})
	raw, err := New(stub).Drive().DownloadFile(context.Background(), "f1")
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != `{"content":"aGk="}` {
		t.Errorf("unexpected raw: %s", raw)
	}
}

func TestDrive_CreateFolder(t *testing.T) {
	stub := newStub(map[string]stubResponse{
		"google_drive/create-folder": okJSON(`{"id":"folder1","name":"new","mimeType":"application/vnd.google-apps.folder"}`),
	})
	got, err := New(stub).Drive().CreateFolder(context.Background(), "new", "parent")
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "folder1" {
		t.Errorf("unexpected: %+v", got)
	}
	body, _ := json.Marshal(stub.Calls[0].Body)
	if string(body) != `{"name":"new","parentId":"parent"}` {
		t.Errorf("unexpected body: %s", body)
	}
}

func TestDrive_UploadFile(t *testing.T) {
	stub := newStub(map[string]stubResponse{
		"google_drive/upload-file": okJSON(`{"id":"u1","name":"x","mimeType":"text/plain"}`),
	})
	got, err := New(stub).Drive().UploadFile(context.Background(), DriveUploadFileParams{Name: "x", Content: "data", MimeType: "text/plain"})
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "u1" {
		t.Errorf("unexpected: %+v", got)
	}
}

func TestDrive_DeleteFile(t *testing.T) {
	stub := newStub(map[string]stubResponse{"google_drive/delete-file": okJSON(`{"deleted":true}`)})
	if _, err := New(stub).Drive().DeleteFile(context.Background(), "f1"); err != nil {
		t.Fatal(err)
	}
}

func TestDrive_SearchFiles(t *testing.T) {
	stub := newStub(map[string]stubResponse{"google_drive/search-files": okJSON(`{"files":[]}`)})
	if _, err := New(stub).Drive().SearchFiles(context.Background(), "q", 5); err != nil {
		t.Fatal(err)
	}
	body, _ := json.Marshal(stub.Calls[0].Body)
	if string(body) != `{"maxResults":5,"query":"q"}` {
		t.Errorf("unexpected body: %s", body)
	}
}

func TestDrive_GoogleDriveAlias(t *testing.T) {
	stub := newStub(map[string]stubResponse{"google_drive/list-files": okJSON(`{"files":[]}`)})
	ns := New(stub)
	if ns.GoogleDrive() == nil {
		t.Fatal("expected GoogleDrive alias to return a Drive")
	}
	_, _ = ns.GoogleDrive().ListFiles(context.Background(), nil)
	if stub.Calls[0].Provider != "google_drive" {
		t.Errorf("unexpected call: %+v", stub.Calls[0])
	}
}
