package integrations

import (
	"encoding/json"
)

// paramsOrEmpty returns nil-safe wire body for typed params pointers.
// When the caller passes a nil *T, this returns nil so the transport sends `{}`.
func paramsOrEmpty(v any) any {
	if v == nil {
		return nil
	}
	// Handle typed nil (*T)(nil) — reflection would be heavy here; check the
	// concrete pointer types we use directly.
	switch t := v.(type) {
	case *GmailListParams:
		if t == nil {
			return nil
		}
	case *CalendarListEventsParams:
		if t == nil {
			return nil
		}
	case *DriveListFilesParams:
		if t == nil {
			return nil
		}
	case *LinearListIssuesFilter:
		if t == nil {
			return nil
		}
	case *LinearListProjectsFilter:
		if t == nil {
			return nil
		}
	}
	return v
}

// unmarshalIfPresent unmarshals raw into out, tolerating an empty or null body.
// Returns the unmarshal error wrapped as nil-safe — callers that get an empty
// body get back the zero value of out.
func unmarshalIfPresent(raw json.RawMessage, out any) error {
	if len(raw) == 0 {
		return nil
	}
	trimmed := string(raw)
	if trimmed == "null" {
		return nil
	}
	return json.Unmarshal(raw, out)
}
