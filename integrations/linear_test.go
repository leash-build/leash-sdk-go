package integrations

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"
)

func TestLinear_ListIssues_Envelope(t *testing.T) {
	stub := newStub(map[string]stubResponse{
		"linear/list_issues": okJSON(`{"issues":[{"id":"i1","title":"Bug"}],"cursor":"next"}`),
	})
	got, err := New(stub).Linear().ListIssues(context.Background(), &LinearListIssuesFilter{StateType: LinearStateStarted})
	if err != nil {
		t.Fatal(err)
	}
	want := &LinearListIssuesResult{
		Issues: []LinearIssue{{ID: "i1", Title: "Bug"}},
		Cursor: "next",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
	body, _ := json.Marshal(stub.Calls[0].Body)
	if string(body) != `{"stateType":"started"}` {
		t.Errorf("unexpected body: %s", body)
	}
}

func TestLinear_ListIssues_BareArray(t *testing.T) {
	stub := newStub(map[string]stubResponse{
		"linear/list_issues": okJSON(`[{"id":"i1","title":"Bug"}]`),
	})
	got, err := New(stub).Linear().ListIssues(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Issues) != 1 || got.Issues[0].ID != "i1" {
		t.Errorf("unexpected: %+v", got)
	}
	if got.Cursor != "" {
		t.Errorf("expected empty cursor, got %q", got.Cursor)
	}
}

func TestLinear_ListIssues_EmptyEnvelope(t *testing.T) {
	stub := newStub(map[string]stubResponse{"linear/list_issues": okJSON(`{"issues":[]}`)})
	got, err := New(stub).Linear().ListIssues(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Issues) != 0 {
		t.Errorf("expected empty, got %+v", got.Issues)
	}
}

func TestLinear_GetIssue(t *testing.T) {
	stub := newStub(map[string]stubResponse{"linear/get_issue": okJSON(`{"id":"i1","title":"T"}`)})
	got, err := New(stub).Linear().GetIssue(context.Background(), "i1")
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "i1" {
		t.Errorf("unexpected: %+v", got)
	}
}

func TestLinear_CreateIssue(t *testing.T) {
	stub := newStub(map[string]stubResponse{"linear/create_issue": okJSON(`{"id":"new","title":"Hi"}`)})
	priority := LinearPriority(2)
	got, err := New(stub).Linear().CreateIssue(context.Background(), LinearCreateIssueInput{
		TeamID:   "t1",
		Title:    "Hi",
		Priority: &priority,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "new" {
		t.Errorf("unexpected: %+v", got)
	}
	body, _ := json.Marshal(stub.Calls[0].Body)
	if string(body) != `{"teamId":"t1","title":"Hi","priority":2}` {
		t.Errorf("unexpected body: %s", body)
	}
}

func TestLinear_UpdateIssue(t *testing.T) {
	stub := newStub(map[string]stubResponse{"linear/update_issue": okJSON(`{"id":"i1","title":"updated"}`)})
	got, err := New(stub).Linear().UpdateIssue(context.Background(), "i1", LinearUpdateIssuePatch{Title: "updated"})
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != "updated" {
		t.Errorf("unexpected: %+v", got)
	}
	body, _ := json.Marshal(stub.Calls[0].Body)
	if string(body) != `{"id":"i1","title":"updated"}` {
		t.Errorf("unexpected body: %s", body)
	}
}

func TestLinear_AddComment(t *testing.T) {
	stub := newStub(map[string]stubResponse{"linear/add_comment": okJSON(`{"id":"c1","body":"hi","issueId":"i1"}`)})
	got, err := New(stub).Linear().AddComment(context.Background(), "i1", "hi")
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "c1" {
		t.Errorf("unexpected: %+v", got)
	}
}

func TestLinear_ListTeams_Envelope(t *testing.T) {
	stub := newStub(map[string]stubResponse{
		"linear/list_teams": okJSON(`{"teams":[{"id":"t1","key":"LEA","name":"Leash"}]}`),
	})
	got, err := New(stub).Linear().ListTeams(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != "t1" {
		t.Errorf("unexpected: %+v", got)
	}
}

func TestLinear_ListTeams_BareArray(t *testing.T) {
	stub := newStub(map[string]stubResponse{"linear/list_teams": okJSON(`[{"id":"t1","key":"LEA","name":"Leash"}]`)})
	got, err := New(stub).Linear().ListTeams(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != "t1" {
		t.Errorf("unexpected: %+v", got)
	}
}

func TestLinear_ListProjects_Envelope(t *testing.T) {
	stub := newStub(map[string]stubResponse{"linear/list_projects": okJSON(`{"projects":[{"id":"p1","name":"P"}]}`)})
	got, err := New(stub).Linear().ListProjects(context.Background(), &LinearListProjectsFilter{TeamID: "t1"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != "p1" {
		t.Errorf("unexpected: %+v", got)
	}
	body, _ := json.Marshal(stub.Calls[0].Body)
	if string(body) != `{"teamId":"t1"}` {
		t.Errorf("unexpected body: %s", body)
	}
}

func TestLinear_ListProjects_BareArray(t *testing.T) {
	stub := newStub(map[string]stubResponse{"linear/list_projects": okJSON(`[{"id":"p1","name":"P"}]`)})
	got, err := New(stub).Linear().ListProjects(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Name != "P" {
		t.Errorf("unexpected: %+v", got)
	}
}

func TestProvider_GenericCaller(t *testing.T) {
	stub := newStub(map[string]stubResponse{"slack/post-message": okJSON(`{"ok":true}`)})
	c := New(stub).Provider("slack")
	if c.Name() != "slack" {
		t.Errorf("name: %q", c.Name())
	}
	raw, err := c.Call(context.Background(), "post-message", map[string]any{"channel": "#general", "text": "hi"})
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != `{"ok":true}` {
		t.Errorf("unexpected raw: %s", raw)
	}
	if stub.Calls[0].Provider != "slack" || stub.Calls[0].Action != "post-message" {
		t.Errorf("unexpected: %+v", stub.Calls[0])
	}
}
