package leash

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
)

// successResponse returns a JSON-encoded apiResponse with success=true and the given data.
func successResponse(data any) string {
	raw, _ := json.Marshal(data)
	resp := apiResponse{Success: true, Data: raw}
	b, _ := json.Marshal(resp)
	return string(b)
}

// errorResponse returns a JSON-encoded apiResponse with success=false.
func errorResponse(msg, code, connectURL string) string {
	resp := struct {
		Success    bool   `json:"success"`
		Error      string `json:"error"`
		Code       string `json:"code"`
		ConnectURL string `json:"connectUrl,omitempty"`
	}{
		Success:    false,
		Error:      msg,
		Code:       code,
		ConnectURL: connectURL,
	}
	b, _ := json.Marshal(resp)
	return string(b)
}

// newTestClient creates a LeashIntegrations pointed at the given test server.
func newTestClient(serverURL string) *LeashIntegrations {
	return &LeashIntegrations{
		PlatformURL: serverURL,
		AuthToken:   "test-jwt-token",
		HTTPClient:  http.DefaultClient,
		APIKey:      "test-api-key",
	}
}

// --- Client Initialization ---

func TestNew_DefaultPlatformURL(t *testing.T) {
	// Clear env to avoid interference
	orig := os.Getenv("LEASH_API_KEY")
	os.Setenv("LEASH_API_KEY", "env-key-123")
	defer os.Setenv("LEASH_API_KEY", orig)

	client := New("my-jwt")
	if client.PlatformURL != DefaultPlatformURL {
		t.Errorf("expected PlatformURL=%q, got %q", DefaultPlatformURL, client.PlatformURL)
	}
	if client.AuthToken != "my-jwt" {
		t.Errorf("expected AuthToken=%q, got %q", "my-jwt", client.AuthToken)
	}
	if client.APIKey != "env-key-123" {
		t.Errorf("expected APIKey=%q from env, got %q", "env-key-123", client.APIKey)
	}
	if client.HTTPClient == nil {
		t.Error("expected HTTPClient to be set")
	}
}

func TestNewWithURL_CustomURL(t *testing.T) {
	client := NewWithURL("jwt", "https://custom.example.com/")
	if client.PlatformURL != "https://custom.example.com" {
		t.Errorf("expected trailing slash stripped, got %q", client.PlatformURL)
	}
	if client.AuthToken != "jwt" {
		t.Errorf("expected AuthToken=%q, got %q", "jwt", client.AuthToken)
	}
	// NewWithURL does not read LEASH_API_KEY from env
	if client.APIKey != "" {
		t.Errorf("expected empty APIKey from NewWithURL, got %q", client.APIKey)
	}
}

func TestNew_APIKeyFromEnv(t *testing.T) {
	orig := os.Getenv("LEASH_API_KEY")
	defer os.Setenv("LEASH_API_KEY", orig)

	os.Setenv("LEASH_API_KEY", "")
	client := New("jwt")
	if client.APIKey != "" {
		t.Errorf("expected empty APIKey when env is unset, got %q", client.APIKey)
	}

	os.Setenv("LEASH_API_KEY", "secret-key")
	client = New("jwt")
	if client.APIKey != "secret-key" {
		t.Errorf("expected APIKey=%q, got %q", "secret-key", client.APIKey)
	}
}

// --- Auth Headers ---

func TestAuthHeaders_JWTAndAPIKey(t *testing.T) {
	var gotAuth, gotAPIKey string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotAPIKey = r.Header.Get("X-API-Key")
		fmt.Fprint(w, successResponse(map[string]string{"ok": "true"}))
	}))
	defer ts.Close()

	client := newTestClient(ts.URL)
	_, _ = client.Call("test", "action", nil)

	if gotAuth != "Bearer test-jwt-token" {
		t.Errorf("expected Authorization=%q, got %q", "Bearer test-jwt-token", gotAuth)
	}
	if gotAPIKey != "test-api-key" {
		t.Errorf("expected X-API-Key=%q, got %q", "test-api-key", gotAPIKey)
	}
}

func TestAuthHeaders_NoAPIKey(t *testing.T) {
	var gotAPIKey string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAPIKey = r.Header.Get("X-API-Key")
		fmt.Fprint(w, successResponse("ok"))
	}))
	defer ts.Close()

	client := &LeashIntegrations{
		PlatformURL: ts.URL,
		AuthToken:   "jwt",
		HTTPClient:  http.DefaultClient,
	}
	_, _ = client.Call("test", "action", nil)

	if gotAPIKey != "" {
		t.Errorf("expected no X-API-Key header, got %q", gotAPIKey)
	}
}

func TestAuthHeaders_NoJWT(t *testing.T) {
	var gotAuth string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		fmt.Fprint(w, successResponse("ok"))
	}))
	defer ts.Close()

	client := &LeashIntegrations{
		PlatformURL: ts.URL,
		HTTPClient:  http.DefaultClient,
	}
	_, _ = client.Call("test", "action", nil)

	if gotAuth != "" {
		t.Errorf("expected no Authorization header, got %q", gotAuth)
	}
}

// --- URL Construction ---

func TestURLConstruction_ProviderCall(t *testing.T) {
	var gotPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		fmt.Fprint(w, successResponse("ok"))
	}))
	defer ts.Close()

	client := newTestClient(ts.URL)
	_, _ = client.Call("gmail", "send-message", nil)

	expected := "/api/integrations/gmail/send-message"
	if gotPath != expected {
		t.Errorf("expected path=%q, got %q", expected, gotPath)
	}
}

func TestURLConstruction_ConnectionsEndpoint(t *testing.T) {
	var gotPath, gotMethod string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		fmt.Fprint(w, successResponse([]ConnectionStatus{}))
	}))
	defer ts.Close()

	client := newTestClient(ts.URL)
	_, _ = client.GetConnections()

	if gotPath != "/api/integrations/connections" {
		t.Errorf("expected path=%q, got %q", "/api/integrations/connections", gotPath)
	}
	if gotMethod != http.MethodGet {
		t.Errorf("expected method=%q, got %q", http.MethodGet, gotMethod)
	}
}

func TestGetConnectURL(t *testing.T) {
	client := &LeashIntegrations{PlatformURL: "https://leash.build"}

	t.Run("without return URL", func(t *testing.T) {
		u := client.GetConnectURL("google_calendar", "")
		expected := "https://leash.build/api/integrations/connect/google_calendar"
		if u != expected {
			t.Errorf("expected %q, got %q", expected, u)
		}
	})

	t.Run("with return URL", func(t *testing.T) {
		u := client.GetConnectURL("gmail", "https://app.example.com/callback")
		if !strings.Contains(u, "/api/integrations/connect/gmail?return_url=") {
			t.Errorf("expected connect URL with return_url param, got %q", u)
		}
		if !strings.Contains(u, "app.example.com") {
			t.Errorf("expected return URL to be encoded in query, got %q", u)
		}
	})
}

// --- Provider Clients ---

func TestGmailClient(t *testing.T) {
	var paths []string
	var bodies []string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		b, _ := io.ReadAll(r.Body)
		bodies = append(bodies, string(b))
		fmt.Fprint(w, successResponse(map[string]string{"id": "msg-1"}))
	}))
	defer ts.Close()

	client := newTestClient(ts.URL)
	gmail := client.Gmail()

	t.Run("ListMessages", func(t *testing.T) {
		paths = nil
		_, err := gmail.ListMessages(&ListMessagesParams{Query: "from:test@example.com", MaxResults: 5})
		if err != nil {
			t.Fatal(err)
		}
		if paths[0] != "/api/integrations/gmail/list-messages" {
			t.Errorf("expected path gmail/list-messages, got %q", paths[0])
		}
	})

	t.Run("GetMessage", func(t *testing.T) {
		paths = nil
		bodies = nil
		_, err := gmail.GetMessage("msg-123", "full")
		if err != nil {
			t.Fatal(err)
		}
		if paths[0] != "/api/integrations/gmail/get-message" {
			t.Errorf("unexpected path: %q", paths[0])
		}
		if !strings.Contains(bodies[0], `"messageId":"msg-123"`) {
			t.Errorf("expected messageId in body, got %q", bodies[0])
		}
	})

	t.Run("SendMessage", func(t *testing.T) {
		paths = nil
		_, err := gmail.SendMessage(SendMessageParams{
			To:      "recipient@example.com",
			Subject: "Hello",
			Body:    "World",
		})
		if err != nil {
			t.Fatal(err)
		}
		if paths[0] != "/api/integrations/gmail/send-message" {
			t.Errorf("unexpected path: %q", paths[0])
		}
	})

	t.Run("SearchMessages", func(t *testing.T) {
		paths = nil
		_, err := gmail.SearchMessages("is:unread", 10)
		if err != nil {
			t.Fatal(err)
		}
		if paths[0] != "/api/integrations/gmail/search-messages" {
			t.Errorf("unexpected path: %q", paths[0])
		}
	})

	t.Run("ListLabels", func(t *testing.T) {
		paths = nil
		_, err := gmail.ListLabels()
		if err != nil {
			t.Fatal(err)
		}
		if paths[0] != "/api/integrations/gmail/list-labels" {
			t.Errorf("unexpected path: %q", paths[0])
		}
	})
}

func TestCalendarClient(t *testing.T) {
	var lastPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lastPath = r.URL.Path
		fmt.Fprint(w, successResponse(map[string]string{"id": "evt-1"}))
	}))
	defer ts.Close()

	client := newTestClient(ts.URL)
	cal := client.Calendar()

	t.Run("ListCalendars", func(t *testing.T) {
		_, err := cal.ListCalendars()
		if err != nil {
			t.Fatal(err)
		}
		if lastPath != "/api/integrations/google_calendar/list-calendars" {
			t.Errorf("unexpected path: %q", lastPath)
		}
	})

	t.Run("ListEvents", func(t *testing.T) {
		_, err := cal.ListEvents(&ListEventsParams{MaxResults: 10})
		if err != nil {
			t.Fatal(err)
		}
		if lastPath != "/api/integrations/google_calendar/list-events" {
			t.Errorf("unexpected path: %q", lastPath)
		}
	})

	t.Run("CreateEvent", func(t *testing.T) {
		_, err := cal.CreateEvent(CreateEventParams{
			Summary: "Meeting",
			Start:   EventDateTime{DateTime: "2026-01-01T10:00:00Z"},
			End:     EventDateTime{DateTime: "2026-01-01T11:00:00Z"},
		})
		if err != nil {
			t.Fatal(err)
		}
		if lastPath != "/api/integrations/google_calendar/create-event" {
			t.Errorf("unexpected path: %q", lastPath)
		}
	})

	t.Run("GetEvent", func(t *testing.T) {
		_, err := cal.GetEvent("evt-456", "primary")
		if err != nil {
			t.Fatal(err)
		}
		if lastPath != "/api/integrations/google_calendar/get-event" {
			t.Errorf("unexpected path: %q", lastPath)
		}
	})
}

func TestDriveClient(t *testing.T) {
	var lastPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lastPath = r.URL.Path
		fmt.Fprint(w, successResponse(map[string]string{"id": "file-1"}))
	}))
	defer ts.Close()

	client := newTestClient(ts.URL)
	drive := client.Drive()

	t.Run("ListFiles", func(t *testing.T) {
		_, err := drive.ListFiles(&ListFilesParams{MaxResults: 5})
		if err != nil {
			t.Fatal(err)
		}
		if lastPath != "/api/integrations/google_drive/list-files" {
			t.Errorf("unexpected path: %q", lastPath)
		}
	})

	t.Run("GetFile", func(t *testing.T) {
		_, err := drive.GetFile("file-abc")
		if err != nil {
			t.Fatal(err)
		}
		if lastPath != "/api/integrations/google_drive/get-file" {
			t.Errorf("unexpected path: %q", lastPath)
		}
	})

	t.Run("SearchFiles", func(t *testing.T) {
		_, err := drive.SearchFiles("budget 2025", 20)
		if err != nil {
			t.Fatal(err)
		}
		if lastPath != "/api/integrations/google_drive/search-files" {
			t.Errorf("unexpected path: %q", lastPath)
		}
	})
}

// --- Error Handling ---

func TestLeashError(t *testing.T) {
	t.Run("error with code", func(t *testing.T) {
		e := &Error{Message: "not connected", Code: "not_connected", ConnectURL: "https://leash.build/connect"}
		s := e.Error()
		if !strings.Contains(s, "not connected") || !strings.Contains(s, "not_connected") {
			t.Errorf("unexpected error string: %q", s)
		}
	})

	t.Run("error without code", func(t *testing.T) {
		e := &Error{Message: "something failed"}
		s := e.Error()
		if s != "leash: something failed" {
			t.Errorf("unexpected error string: %q", s)
		}
	})
}

func TestCallReturnsLeashError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, errorResponse("Gmail not connected", "not_connected", "https://leash.build/connect/gmail"))
	}))
	defer ts.Close()

	client := newTestClient(ts.URL)
	_, err := client.Call("gmail", "list-messages", nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	leashErr, ok := err.(*Error)
	if !ok {
		t.Fatalf("expected *Error, got %T", err)
	}
	if leashErr.Message != "Gmail not connected" {
		t.Errorf("unexpected message: %q", leashErr.Message)
	}
	if leashErr.Code != "not_connected" {
		t.Errorf("unexpected code: %q", leashErr.Code)
	}
	if leashErr.ConnectURL != "https://leash.build/connect/gmail" {
		t.Errorf("unexpected connectURL: %q", leashErr.ConnectURL)
	}
}

// --- Env Caching ---

func TestGetEnv_Caching(t *testing.T) {
	var callCount int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&callCount, 1)
		fmt.Fprint(w, successResponse(map[string]string{
			"DATABASE_URL": "postgres://localhost/db",
			"SECRET_KEY":   "s3cr3t",
		}))
	}))
	defer ts.Close()

	client := newTestClient(ts.URL)

	// First call should hit the server
	env1, err := client.GetEnv()
	if err != nil {
		t.Fatal(err)
	}
	if env1["DATABASE_URL"] != "postgres://localhost/db" {
		t.Errorf("unexpected DATABASE_URL: %q", env1["DATABASE_URL"])
	}

	// Second call should use cache
	env2, err := client.GetEnv()
	if err != nil {
		t.Fatal(err)
	}
	if env2["SECRET_KEY"] != "s3cr3t" {
		t.Errorf("unexpected SECRET_KEY: %q", env2["SECRET_KEY"])
	}

	if atomic.LoadInt32(&callCount) != 1 {
		t.Errorf("expected 1 HTTP call (cached), got %d", callCount)
	}
}

func TestGetEnvKey(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, successResponse(map[string]string{
			"MY_VAR": "hello",
		}))
	}))
	defer ts.Close()

	client := newTestClient(ts.URL)

	val, err := client.GetEnvKey("MY_VAR")
	if err != nil {
		t.Fatal(err)
	}
	if val != "hello" {
		t.Errorf("expected %q, got %q", "hello", val)
	}

	val, err = client.GetEnvKey("MISSING")
	if err != nil {
		t.Fatal(err)
	}
	if val != "" {
		t.Errorf("expected empty string for missing key, got %q", val)
	}
}

func TestGetEnv_EndpointAndMethod(t *testing.T) {
	var gotPath, gotMethod string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		fmt.Fprint(w, successResponse(map[string]string{}))
	}))
	defer ts.Close()

	client := newTestClient(ts.URL)
	_, _ = client.GetEnv()

	if gotPath != "/api/apps/env" {
		t.Errorf("expected path=%q, got %q", "/api/apps/env", gotPath)
	}
	if gotMethod != http.MethodGet {
		t.Errorf("expected method=%q, got %q", http.MethodGet, gotMethod)
	}
}

// --- Connection Status ---

func TestIsConnected(t *testing.T) {
	connections := []ConnectionStatus{
		{ProviderID: "gmail", Status: "active", Email: "user@gmail.com"},
		{ProviderID: "google_calendar", Status: "expired"},
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, successResponse(connections))
	}))
	defer ts.Close()

	client := newTestClient(ts.URL)

	t.Run("connected provider", func(t *testing.T) {
		if !client.IsConnected("gmail") {
			t.Error("expected gmail to be connected")
		}
	})

	t.Run("expired provider", func(t *testing.T) {
		if client.IsConnected("google_calendar") {
			t.Error("expected google_calendar to NOT be connected (expired)")
		}
	})

	t.Run("unknown provider", func(t *testing.T) {
		if client.IsConnected("slack") {
			t.Error("expected slack to NOT be connected (not present)")
		}
	})
}

func TestIsConnected_ServerError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, errorResponse("server error", "internal", ""))
	}))
	defer ts.Close()

	client := newTestClient(ts.URL)
	if client.IsConnected("gmail") {
		t.Error("expected IsConnected to return false on server error")
	}
}

func TestGetConnections(t *testing.T) {
	connections := []ConnectionStatus{
		{ProviderID: "gmail", Status: "active", Email: "user@gmail.com"},
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, successResponse(connections))
	}))
	defer ts.Close()

	client := newTestClient(ts.URL)
	conns, err := client.GetConnections()
	if err != nil {
		t.Fatal(err)
	}
	if len(conns) != 1 {
		t.Fatalf("expected 1 connection, got %d", len(conns))
	}
	if conns[0].ProviderID != "gmail" {
		t.Errorf("expected providerId=%q, got %q", "gmail", conns[0].ProviderID)
	}
	if conns[0].Email != "user@gmail.com" {
		t.Errorf("expected email=%q, got %q", "user@gmail.com", conns[0].Email)
	}
}

// --- MCP Calls ---

func TestMCP(t *testing.T) {
	var gotPath, gotMethod string
	var gotBody map[string]any
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		b, _ := io.ReadAll(r.Body)
		json.Unmarshal(b, &gotBody)
		fmt.Fprint(w, successResponse(map[string]string{"result": "ok"}))
	}))
	defer ts.Close()

	client := newTestClient(ts.URL)

	t.Run("with args", func(t *testing.T) {
		result, err := client.MCP("@modelcontextprotocol/server-github", "list_issues", map[string]string{"repo": "my-repo"})
		if err != nil {
			t.Fatal(err)
		}
		if gotPath != "/api/mcp/run" {
			t.Errorf("expected path=%q, got %q", "/api/mcp/run", gotPath)
		}
		if gotMethod != http.MethodPost {
			t.Errorf("expected POST, got %s", gotMethod)
		}
		if gotBody["package"] != "@modelcontextprotocol/server-github" {
			t.Errorf("unexpected package: %v", gotBody["package"])
		}
		if gotBody["tool"] != "list_issues" {
			t.Errorf("unexpected tool: %v", gotBody["tool"])
		}
		if gotBody["args"] == nil {
			t.Error("expected args to be present")
		}
		if result == nil {
			t.Error("expected non-nil result")
		}
	})

	t.Run("without args", func(t *testing.T) {
		gotBody = nil
		_, err := client.MCP("@some/server", "ping", nil)
		if err != nil {
			t.Fatal(err)
		}
		if gotBody["args"] != nil {
			t.Errorf("expected args to be absent when nil, got %v", gotBody["args"])
		}
	})

	t.Run("auth headers sent", func(t *testing.T) {
		var gotAuth, gotAPIKey string
		ts2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotAuth = r.Header.Get("Authorization")
			gotAPIKey = r.Header.Get("X-API-Key")
			fmt.Fprint(w, successResponse("ok"))
		}))
		defer ts2.Close()

		c := newTestClient(ts2.URL)
		_, _ = c.MCP("pkg", "tool", nil)
		if gotAuth != "Bearer test-jwt-token" {
			t.Errorf("expected auth header on MCP call, got %q", gotAuth)
		}
		if gotAPIKey != "test-api-key" {
			t.Errorf("expected api key header on MCP call, got %q", gotAPIKey)
		}
	})

	t.Run("error response", func(t *testing.T) {
		ts3 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, errorResponse("tool not found", "not_found", ""))
		}))
		defer ts3.Close()

		c := newTestClient(ts3.URL)
		_, err := c.MCP("pkg", "missing-tool", nil)
		if err == nil {
			t.Fatal("expected error")
		}
		leashErr, ok := err.(*Error)
		if !ok {
			t.Fatalf("expected *Error, got %T", err)
		}
		if leashErr.Code != "not_found" {
			t.Errorf("expected code=%q, got %q", "not_found", leashErr.Code)
		}
	})
}

// --- Custom Integration ---

func TestCustomIntegration(t *testing.T) {
	var gotPath, gotMethod string
	var gotBody map[string]any
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		b, _ := io.ReadAll(r.Body)
		json.Unmarshal(b, &gotBody)
		fmt.Fprint(w, successResponse(map[string]string{"status": "ok"}))
	}))
	defer ts.Close()

	client := newTestClient(ts.URL)
	integration := client.Integration("slack")

	t.Run("Call sends to correct endpoint", func(t *testing.T) {
		_, err := integration.Call("/chat.postMessage", "POST", map[string]string{"channel": "#general"})
		if err != nil {
			t.Fatal(err)
		}
		if gotPath != "/api/integrations/custom/slack" {
			t.Errorf("expected path=%q, got %q", "/api/integrations/custom/slack", gotPath)
		}
		if gotMethod != http.MethodPost {
			t.Errorf("expected POST, got %s", gotMethod)
		}
		if gotBody["path"] != "/chat.postMessage" {
			t.Errorf("expected path in body, got %v", gotBody["path"])
		}
		if gotBody["method"] != "POST" {
			t.Errorf("expected method in body, got %v", gotBody["method"])
		}
	})

	t.Run("CallWithHeaders includes headers", func(t *testing.T) {
		gotBody = nil
		headers := map[string]string{"X-Custom": "value"}
		_, err := integration.CallWithHeaders("/api/resource", "GET", nil, headers)
		if err != nil {
			t.Fatal(err)
		}
		if gotBody["headers"] == nil {
			t.Error("expected headers in body")
		}
	})

	t.Run("error response", func(t *testing.T) {
		ts2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, errorResponse("integration not found", "not_found", ""))
		}))
		defer ts2.Close()

		c := newTestClient(ts2.URL)
		integ := c.Integration("nonexistent")
		_, err := integ.Call("/test", "GET", nil)
		if err == nil {
			t.Fatal("expected error")
		}
		leashErr, ok := err.(*Error)
		if !ok {
			t.Fatalf("expected *Error, got %T", err)
		}
		if leashErr.Code != "not_found" {
			t.Errorf("expected code=%q, got %q", "not_found", leashErr.Code)
		}
	})
}

// --- Request Body ---

func TestCall_SendsRequestBody(t *testing.T) {
	var gotBody string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		fmt.Fprint(w, successResponse("ok"))
	}))
	defer ts.Close()

	client := newTestClient(ts.URL)
	_, _ = client.Call("gmail", "send-message", map[string]string{
		"to":      "test@example.com",
		"subject": "Hi",
	})

	if !strings.Contains(gotBody, `"to":"test@example.com"`) {
		t.Errorf("expected body to contain request data, got %q", gotBody)
	}
}

func TestCall_NilBody(t *testing.T) {
	var gotContentLength int64
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotContentLength = r.ContentLength
		fmt.Fprint(w, successResponse("ok"))
	}))
	defer ts.Close()

	client := newTestClient(ts.URL)
	_, err := client.Call("gmail", "list-labels", nil)
	if err != nil {
		t.Fatal(err)
	}
	// With nil body, content length should be 0 or -1 (no body)
	if gotContentLength > 0 {
		t.Errorf("expected no body for nil input, got content-length=%d", gotContentLength)
	}
}

// --- Content-Type ---

func TestCall_ContentType(t *testing.T) {
	var gotContentType string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotContentType = r.Header.Get("Content-Type")
		fmt.Fprint(w, successResponse("ok"))
	}))
	defer ts.Close()

	client := newTestClient(ts.URL)
	_, _ = client.Call("test", "action", map[string]string{"key": "val"})

	if gotContentType != "application/json" {
		t.Errorf("expected Content-Type=%q, got %q", "application/json", gotContentType)
	}
}

// --- Invalid JSON response ---

func TestCall_InvalidJSONResponse(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "this is not json")
	}))
	defer ts.Close()

	client := newTestClient(ts.URL)
	_, err := client.Call("test", "action", nil)
	if err == nil {
		t.Fatal("expected error for invalid JSON response")
	}
	if !strings.Contains(err.Error(), "failed to parse response") {
		t.Errorf("expected parse error, got %q", err.Error())
	}
}

// --- Method verification for Call ---

func TestCall_UsesPOST(t *testing.T) {
	var gotMethod string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		fmt.Fprint(w, successResponse("ok"))
	}))
	defer ts.Close()

	client := newTestClient(ts.URL)
	_, _ = client.Call("provider", "action", nil)

	if gotMethod != http.MethodPost {
		t.Errorf("expected POST, got %s", gotMethod)
	}
}
