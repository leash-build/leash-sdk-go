package leash

import "encoding/json"

// CalendarClient provides methods for interacting with Google Calendar via
// the Leash platform proxy. Obtain one by calling LeashIntegrations.Calendar().
type CalendarClient struct {
	client *LeashIntegrations
}

// ListCalendars returns all calendars accessible to the user.
func (c *CalendarClient) ListCalendars() (json.RawMessage, error) {
	return c.client.call("google_calendar", "list-calendars", nil)
}

// ListEvents returns events from a calendar.
//
// Pass nil for params to use server defaults.
func (c *CalendarClient) ListEvents(params *ListEventsParams) (json.RawMessage, error) {
	return c.client.call("google_calendar", "list-events", params)
}

// CreateEvent creates a new calendar event.
func (c *CalendarClient) CreateEvent(params CreateEventParams) (json.RawMessage, error) {
	return c.client.call("google_calendar", "create-event", params)
}

// GetEvent retrieves a single event by ID.
//
// CalendarID is optional; if empty, the server uses "primary".
func (c *CalendarClient) GetEvent(eventID string, calendarID string) (json.RawMessage, error) {
	body := GetEventParams{EventID: eventID, CalendarID: calendarID}
	return c.client.call("google_calendar", "get-event", body)
}
