package integrations

import "context"

// CalendarListEntry is a calendar visible in the user's calendar list.
type CalendarListEntry struct {
	ID              string `json:"id"`
	Summary         string `json:"summary,omitempty"`
	Description     string `json:"description,omitempty"`
	TimeZone        string `json:"timeZone,omitempty"`
	Primary         bool   `json:"primary,omitempty"`
	BackgroundColor string `json:"backgroundColor,omitempty"`
	ForegroundColor string `json:"foregroundColor,omitempty"`
}

// CalendarList is the shape returned by [Calendar.ListCalendars].
type CalendarList struct {
	Calendars []CalendarListEntry `json:"calendars"`
}

// CalendarEventDateTime is a Google-style date/time descriptor.
//
// Exactly one of DateTime (timed event) or Date (all-day event) is populated.
type CalendarEventDateTime struct {
	DateTime string `json:"dateTime,omitempty"`
	Date     string `json:"date,omitempty"`
	TimeZone string `json:"timeZone,omitempty"`
}

// CalendarAttendee is a single attendee on a calendar event.
type CalendarAttendee struct {
	Email          string `json:"email"`
	ResponseStatus string `json:"responseStatus,omitempty"`
}

// CalendarEvent is a single event on a calendar.
type CalendarEvent struct {
	ID          string                `json:"id,omitempty"`
	Summary     string                `json:"summary,omitempty"`
	Description string                `json:"description,omitempty"`
	Location    string                `json:"location,omitempty"`
	Start       CalendarEventDateTime `json:"start"`
	End         CalendarEventDateTime `json:"end"`
	Attendees   []CalendarAttendee    `json:"attendees,omitempty"`
	Status      string                `json:"status,omitempty"`
	HTMLLink    string                `json:"htmlLink,omitempty"`
	Created     string                `json:"created,omitempty"`
	Updated     string                `json:"updated,omitempty"`
}

// CalendarEventList is the shape returned by [Calendar.ListEvents].
type CalendarEventList struct {
	Events        []CalendarEvent `json:"events"`
	NextPageToken string          `json:"nextPageToken,omitempty"`
}

// CalendarListEventsParams configures a [Calendar.ListEvents] call.
//
// All fields are optional. SingleEvents is a *bool so the zero value
// distinguishes "not set" from "explicitly false".
type CalendarListEventsParams struct {
	CalendarID   string `json:"calendarId,omitempty"`
	TimeMin      string `json:"timeMin,omitempty"`
	TimeMax      string `json:"timeMax,omitempty"`
	MaxResults   int    `json:"maxResults,omitempty"`
	Query        string `json:"query,omitempty"`
	SingleEvents *bool  `json:"singleEvents,omitempty"`
	OrderBy      string `json:"orderBy,omitempty"`
}

// CalendarCreateEventParams configures a [Calendar.CreateEvent] call.
//
// Summary, Start, and End are required.
type CalendarCreateEventParams struct {
	CalendarID  string                `json:"calendarId,omitempty"`
	Summary     string                `json:"summary"`
	Description string                `json:"description,omitempty"`
	Location    string                `json:"location,omitempty"`
	Start       CalendarEventDateTime `json:"start"`
	End         CalendarEventDateTime `json:"end"`
	Attendees   []CalendarAttendee    `json:"attendees,omitempty"`
}

// Calendar is the typed Google Calendar provider client.
type Calendar struct {
	caller Caller
}

const calendarProvider = "google_calendar"

// ListCalendars returns the user's accessible calendars.
func (c *Calendar) ListCalendars(ctx context.Context) (*CalendarList, error) {
	raw, err := c.caller.IntegrationsCall(ctx, calendarProvider, "list-calendars", nil)
	if err != nil {
		return nil, err
	}
	var out CalendarList
	if err := unmarshalIfPresent(raw, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ListEvents returns events on a calendar. Pass nil for params to use
// platform defaults (primary calendar, recent window).
func (c *Calendar) ListEvents(ctx context.Context, params *CalendarListEventsParams) (*CalendarEventList, error) {
	raw, err := c.caller.IntegrationsCall(ctx, calendarProvider, "list-events", paramsOrEmpty(params))
	if err != nil {
		return nil, err
	}
	var out CalendarEventList
	if err := unmarshalIfPresent(raw, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// CreateEvent creates a new calendar event.
func (c *Calendar) CreateEvent(ctx context.Context, params CalendarCreateEventParams) (*CalendarEvent, error) {
	raw, err := c.caller.IntegrationsCall(ctx, calendarProvider, "create-event", params)
	if err != nil {
		return nil, err
	}
	var out CalendarEvent
	if err := unmarshalIfPresent(raw, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetEvent retrieves a single event. Pass an empty calendarID to use the
// primary calendar.
func (c *Calendar) GetEvent(ctx context.Context, eventID, calendarID string) (*CalendarEvent, error) {
	body := map[string]any{"eventId": eventID}
	if calendarID != "" {
		body["calendarId"] = calendarID
	}
	raw, err := c.caller.IntegrationsCall(ctx, calendarProvider, "get-event", body)
	if err != nil {
		return nil, err
	}
	var out CalendarEvent
	if err := unmarshalIfPresent(raw, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
