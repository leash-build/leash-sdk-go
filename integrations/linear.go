package integrations

import (
	"context"
	"encoding/json"
)

// LinearStateType is the workflow-state classification on a Linear issue.
type LinearStateType string

const (
	LinearStateBacklog   LinearStateType = "backlog"
	LinearStateUnstarted LinearStateType = "unstarted"
	LinearStateStarted   LinearStateType = "started"
	LinearStateCompleted LinearStateType = "completed"
	LinearStateCanceled  LinearStateType = "canceled"
	LinearStateTriage    LinearStateType = "triage"
)

// LinearPriority is the priority bucket on a Linear issue.
//
//	0 = no priority
//	1 = urgent
//	2 = high
//	3 = normal / medium
//	4 = low
type LinearPriority int

// LinearProjectState is the lifecycle state of a Linear project.
type LinearProjectState string

const (
	LinearProjectPlanned   LinearProjectState = "planned"
	LinearProjectStarted   LinearProjectState = "started"
	LinearProjectPaused    LinearProjectState = "paused"
	LinearProjectCompleted LinearProjectState = "completed"
	LinearProjectCanceled  LinearProjectState = "canceled"
	LinearProjectBacklog   LinearProjectState = "backlog"
)

// LinearUserRef is a minimal user reference Linear returns on issues + comments.
type LinearUserRef struct {
	ID          string `json:"id"`
	Name        string `json:"name,omitempty"`
	Email       string `json:"email,omitempty"`
	DisplayName string `json:"displayName,omitempty"`
}

// LinearStateRef is a minimal workflow-state reference returned on issues.
type LinearStateRef struct {
	ID    string          `json:"id"`
	Name  string          `json:"name"`
	Type  LinearStateType `json:"type"`
	Color string          `json:"color,omitempty"`
}

// LinearTeamRef is a minimal team reference returned on issues.
type LinearTeamRef struct {
	ID   string `json:"id"`
	Key  string `json:"key"`
	Name string `json:"name"`
}

// LinearIssue is a single Linear issue.
type LinearIssue struct {
	ID          string          `json:"id"`
	Identifier  string          `json:"identifier,omitempty"`
	Title       string          `json:"title"`
	Description string          `json:"description,omitempty"`
	Priority    *LinearPriority `json:"priority,omitempty"`
	CreatedAt   string          `json:"createdAt,omitempty"`
	UpdatedAt   string          `json:"updatedAt,omitempty"`
	URL         string          `json:"url,omitempty"`
	Assignee    *LinearUserRef  `json:"assignee,omitempty"`
	State       *LinearStateRef `json:"state,omitempty"`
	Team        *LinearTeamRef  `json:"team,omitempty"`
	LabelIDs    []string        `json:"labelIds,omitempty"`
	ProjectID   string          `json:"projectId,omitempty"`
}

// LinearComment is a single comment on a Linear issue.
type LinearComment struct {
	ID        string         `json:"id"`
	Body      string         `json:"body"`
	IssueID   string         `json:"issueId"`
	User      *LinearUserRef `json:"user,omitempty"`
	CreatedAt string         `json:"createdAt,omitempty"`
	UpdatedAt string         `json:"updatedAt,omitempty"`
	URL       string         `json:"url,omitempty"`
}

// LinearTeam is a single Linear team.
type LinearTeam struct {
	ID          string `json:"id"`
	Key         string `json:"key"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Private     bool   `json:"private,omitempty"`
	Icon        string `json:"icon,omitempty"`
	Color       string `json:"color,omitempty"`
}

// LinearProject is a single Linear project.
type LinearProject struct {
	ID          string             `json:"id"`
	Name        string             `json:"name"`
	Description string             `json:"description,omitempty"`
	State       LinearProjectState `json:"state,omitempty"`
	TargetDate  string             `json:"targetDate,omitempty"`
	StartDate   string             `json:"startDate,omitempty"`
	URL         string             `json:"url,omitempty"`
	TeamIDs     []string           `json:"teamIds,omitempty"`
	Progress    float64            `json:"progress,omitempty"`
}

// LinearListIssuesFilter narrows a [Linear.ListIssues] call.
type LinearListIssuesFilter struct {
	TeamID     string          `json:"teamId,omitempty"`
	AssigneeID string          `json:"assigneeId,omitempty"`
	StateType  LinearStateType `json:"stateType,omitempty"`
	Limit      int             `json:"limit,omitempty"`
	Cursor     string          `json:"cursor,omitempty"`
}

// LinearListIssuesResult is the response shape from [Linear.ListIssues].
type LinearListIssuesResult struct {
	Issues []LinearIssue `json:"issues"`
	Cursor string        `json:"cursor,omitempty"`
}

// LinearCreateIssueInput is the body of a [Linear.CreateIssue] call.
//
// TeamID and Title are required.
type LinearCreateIssueInput struct {
	TeamID      string          `json:"teamId"`
	Title       string          `json:"title"`
	Description string          `json:"description,omitempty"`
	AssigneeID  string          `json:"assigneeId,omitempty"`
	Priority    *LinearPriority `json:"priority,omitempty"`
	LabelIDs    []string        `json:"labelIds,omitempty"`
}

// LinearUpdateIssuePatch is the partial-update body of [Linear.UpdateIssue].
//
// All fields are optional — only set fields are sent to the platform.
type LinearUpdateIssuePatch struct {
	TeamID      string          `json:"teamId,omitempty"`
	Title       string          `json:"title,omitempty"`
	Description string          `json:"description,omitempty"`
	AssigneeID  string          `json:"assigneeId,omitempty"`
	Priority    *LinearPriority `json:"priority,omitempty"`
	LabelIDs    []string        `json:"labelIds,omitempty"`
}

// LinearListProjectsFilter narrows a [Linear.ListProjects] call.
type LinearListProjectsFilter struct {
	TeamID string `json:"teamId,omitempty"`
}

// Linear is the typed Linear provider client. The platform uses underscored
// action names on the wire (list_issues, etc.) — preserved here. Tolerant of
// envelope vs. bare-array response shapes from the upstream MCP server.
type Linear struct {
	caller Caller
}

const linearProvider = "linear"

// ListIssues returns issues, optionally filtered. Pass nil for filter to fetch
// the user's default issue view.
func (l *Linear) ListIssues(ctx context.Context, filter *LinearListIssuesFilter) (*LinearListIssuesResult, error) {
	raw, err := l.caller.IntegrationsCall(ctx, linearProvider, "list_issues", paramsOrEmpty(filter))
	if err != nil {
		return nil, err
	}
	if len(raw) == 0 || string(raw) == "null" {
		return &LinearListIssuesResult{Issues: []LinearIssue{}}, nil
	}
	// Tolerate bare-array shape.
	var arr []LinearIssue
	if err := json.Unmarshal(raw, &arr); err == nil {
		return &LinearListIssuesResult{Issues: arr}, nil
	}
	var out LinearListIssuesResult
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	if out.Issues == nil {
		out.Issues = []LinearIssue{}
	}
	return &out, nil
}

// GetIssue returns a single issue by Linear UUID.
func (l *Linear) GetIssue(ctx context.Context, id string) (*LinearIssue, error) {
	raw, err := l.caller.IntegrationsCall(ctx, linearProvider, "get_issue", map[string]any{"id": id})
	if err != nil {
		return nil, err
	}
	var out LinearIssue
	if err := unmarshalIfPresent(raw, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// CreateIssue creates a new Linear issue.
func (l *Linear) CreateIssue(ctx context.Context, input LinearCreateIssueInput) (*LinearIssue, error) {
	raw, err := l.caller.IntegrationsCall(ctx, linearProvider, "create_issue", input)
	if err != nil {
		return nil, err
	}
	var out LinearIssue
	if err := unmarshalIfPresent(raw, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// UpdateIssue patches an issue. Only fields set on the patch are forwarded.
//
// The Linear platform expects the issue id alongside the patch fields in a
// single body — this method merges them for you.
func (l *Linear) UpdateIssue(ctx context.Context, id string, patch LinearUpdateIssuePatch) (*LinearIssue, error) {
	body := map[string]any{"id": id}
	if patch.TeamID != "" {
		body["teamId"] = patch.TeamID
	}
	if patch.Title != "" {
		body["title"] = patch.Title
	}
	if patch.Description != "" {
		body["description"] = patch.Description
	}
	if patch.AssigneeID != "" {
		body["assigneeId"] = patch.AssigneeID
	}
	if patch.Priority != nil {
		body["priority"] = *patch.Priority
	}
	if patch.LabelIDs != nil {
		body["labelIds"] = patch.LabelIDs
	}
	raw, err := l.caller.IntegrationsCall(ctx, linearProvider, "update_issue", body)
	if err != nil {
		return nil, err
	}
	var out LinearIssue
	if err := unmarshalIfPresent(raw, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// AddComment posts a comment to an existing issue.
func (l *Linear) AddComment(ctx context.Context, issueID, body string) (*LinearComment, error) {
	raw, err := l.caller.IntegrationsCall(ctx, linearProvider, "add_comment", map[string]any{"issueId": issueID, "body": body})
	if err != nil {
		return nil, err
	}
	var out LinearComment
	if err := unmarshalIfPresent(raw, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ListTeams returns all teams visible to the authenticated user. Tolerates
// envelope `{ teams: [...] }` and bare `[...]` responses.
func (l *Linear) ListTeams(ctx context.Context) ([]LinearTeam, error) {
	raw, err := l.caller.IntegrationsCall(ctx, linearProvider, "list_teams", nil)
	if err != nil {
		return nil, err
	}
	if len(raw) == 0 || string(raw) == "null" {
		return []LinearTeam{}, nil
	}
	var arr []LinearTeam
	if err := json.Unmarshal(raw, &arr); err == nil {
		return arr, nil
	}
	var envelope struct {
		Teams []LinearTeam `json:"teams"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, err
	}
	if envelope.Teams == nil {
		return []LinearTeam{}, nil
	}
	return envelope.Teams, nil
}

// ListProjects returns Linear projects, optionally filtered. Tolerates
// envelope `{ projects: [...] }` and bare `[...]` responses.
func (l *Linear) ListProjects(ctx context.Context, filter *LinearListProjectsFilter) ([]LinearProject, error) {
	raw, err := l.caller.IntegrationsCall(ctx, linearProvider, "list_projects", paramsOrEmpty(filter))
	if err != nil {
		return nil, err
	}
	if len(raw) == 0 || string(raw) == "null" {
		return []LinearProject{}, nil
	}
	var arr []LinearProject
	if err := json.Unmarshal(raw, &arr); err == nil {
		return arr, nil
	}
	var envelope struct {
		Projects []LinearProject `json:"projects"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, err
	}
	if envelope.Projects == nil {
		return []LinearProject{}, nil
	}
	return envelope.Projects, nil
}
