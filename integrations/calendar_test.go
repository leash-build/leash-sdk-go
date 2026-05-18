package integrations

import (
	"context"
	"encoding/json"
	"testing"
)

func TestCalendar_ListCalendars(t *testing.T) {
	stub := newStub(map[string]stubResponse{
		"google_calendar/list-calendars": okJSON(`{"calendars":[{"id":"primary","summary":"Primary"}]}`),
	})
	got, err := New(stub).Calendar().ListCalendars(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Calendars) != 1 || got.Calendars[0].ID != "primary" {
		t.Errorf("unexpected: %+v", got)
	}
}

func TestCalendar_ListEvents(t *testing.T) {
	stub := newStub(map[string]stubResponse{
		"google_calendar/list-events": okJSON(`{"events":[{"id":"e1","summary":"meet","start":{"dateTime":"2026-01-01T00:00:00Z"},"end":{"dateTime":"2026-01-01T01:00:00Z"}}]}`),
	})
	got, err := New(stub).Calendar().ListEvents(context.Background(), &CalendarListEventsParams{MaxResults: 5})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Events) != 1 || got.Events[0].Summary != "meet" {
		t.Errorf("unexpected events: %+v", got)
	}
	body, _ := json.Marshal(stub.Calls[0].Body)
	if string(body) != `{"maxResults":5}` {
		t.Errorf("unexpected body: %s", body)
	}
}

func TestCalendar_ListEvents_NilParams(t *testing.T) {
	stub := newStub(map[string]stubResponse{"google_calendar/list-events": okJSON(`{"events":[]}`)})
	if _, err := New(stub).Calendar().ListEvents(context.Background(), nil); err != nil {
		t.Fatal(err)
	}
	if stub.Calls[0].Body != nil {
		t.Errorf("expected nil body, got %v", stub.Calls[0].Body)
	}
}

func TestCalendar_CreateEvent(t *testing.T) {
	stub := newStub(map[string]stubResponse{
		"google_calendar/create-event": okJSON(`{"id":"new","summary":"Hi","start":{"dateTime":"x"},"end":{"dateTime":"y"}}`),
	})
	got, err := New(stub).Calendar().CreateEvent(context.Background(), CalendarCreateEventParams{
		Summary: "Hi",
		Start:   CalendarEventDateTime{DateTime: "x"},
		End:     CalendarEventDateTime{DateTime: "y"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "new" {
		t.Errorf("unexpected: %+v", got)
	}
}

func TestCalendar_GetEvent(t *testing.T) {
	stub := newStub(map[string]stubResponse{
		"google_calendar/get-event": okJSON(`{"id":"e1","summary":"S","start":{"dateTime":"x"},"end":{"dateTime":"y"}}`),
	})
	got, err := New(stub).Calendar().GetEvent(context.Background(), "e1", "primary")
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "e1" {
		t.Errorf("unexpected: %+v", got)
	}
	body, _ := json.Marshal(stub.Calls[0].Body)
	if string(body) != `{"calendarId":"primary","eventId":"e1"}` {
		t.Errorf("unexpected body: %s", body)
	}
}

func TestCalendar_GoogleCalendarAlias(t *testing.T) {
	stub := newStub(map[string]stubResponse{"google_calendar/list-calendars": okJSON(`{"calendars":[]}`)})
	ns := New(stub)
	if ns.GoogleCalendar() == nil {
		t.Fatal("expected GoogleCalendar alias to return a Calendar")
	}
	// Both methods should produce identical wire calls
	_, _ = ns.GoogleCalendar().ListCalendars(context.Background())
	if stub.Calls[0].Provider != "google_calendar" || stub.Calls[0].Action != "list-calendars" {
		t.Errorf("unexpected call: %+v", stub.Calls[0])
	}
}
