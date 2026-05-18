package leash

import "github.com/leash-build/leash-sdk-go/integrations"

// Re-exports of the most-used integration types so callers can write
// `leash.GmailListParams{...}` instead of importing the subpackage
// separately. The subpackage remains the source of truth; the aliases here
// stay 1:1 with whatever it exports.

// Gmail aliases
type (
	GmailMessage           = integrations.GmailMessage
	GmailMessageList       = integrations.GmailMessageList
	GmailLabel             = integrations.GmailLabel
	GmailLabelList         = integrations.GmailLabelList
	GmailListParams        = integrations.GmailListParams
	GmailSendMessageParams = integrations.GmailSendMessageParams
	GmailMessageFormat     = integrations.GmailMessageFormat
)

const (
	GmailFormatFull     = integrations.GmailFormatFull
	GmailFormatMetadata = integrations.GmailFormatMetadata
	GmailFormatMinimal  = integrations.GmailFormatMinimal
	GmailFormatRaw      = integrations.GmailFormatRaw
)

// Calendar aliases
type (
	CalendarListEntry         = integrations.CalendarListEntry
	CalendarList              = integrations.CalendarList
	CalendarEvent             = integrations.CalendarEvent
	CalendarEventList         = integrations.CalendarEventList
	CalendarEventDateTime     = integrations.CalendarEventDateTime
	CalendarAttendee          = integrations.CalendarAttendee
	CalendarListEventsParams  = integrations.CalendarListEventsParams
	CalendarCreateEventParams = integrations.CalendarCreateEventParams
)

// Drive aliases
type (
	DriveFile             = integrations.DriveFile
	DriveFileList         = integrations.DriveFileList
	DriveListFilesParams  = integrations.DriveListFilesParams
	DriveUploadFileParams = integrations.DriveUploadFileParams
)

// Linear aliases
type (
	LinearStateType          = integrations.LinearStateType
	LinearPriority           = integrations.LinearPriority
	LinearProjectState       = integrations.LinearProjectState
	LinearUserRef            = integrations.LinearUserRef
	LinearStateRef           = integrations.LinearStateRef
	LinearTeamRef            = integrations.LinearTeamRef
	LinearIssue              = integrations.LinearIssue
	LinearComment            = integrations.LinearComment
	LinearTeam               = integrations.LinearTeam
	LinearProject            = integrations.LinearProject
	LinearListIssuesFilter   = integrations.LinearListIssuesFilter
	LinearListIssuesResult   = integrations.LinearListIssuesResult
	LinearCreateIssueInput   = integrations.LinearCreateIssueInput
	LinearUpdateIssuePatch   = integrations.LinearUpdateIssuePatch
	LinearListProjectsFilter = integrations.LinearListProjectsFilter
)

const (
	LinearStateBacklog   = integrations.LinearStateBacklog
	LinearStateUnstarted = integrations.LinearStateUnstarted
	LinearStateStarted   = integrations.LinearStateStarted
	LinearStateCompleted = integrations.LinearStateCompleted
	LinearStateCanceled  = integrations.LinearStateCanceled
	LinearStateTriage    = integrations.LinearStateTriage

	LinearProjectPlanned   = integrations.LinearProjectPlanned
	LinearProjectStarted   = integrations.LinearProjectStarted
	LinearProjectPaused    = integrations.LinearProjectPaused
	LinearProjectCompleted = integrations.LinearProjectCompleted
	LinearProjectCanceled  = integrations.LinearProjectCanceled
	LinearProjectBacklog   = integrations.LinearProjectBacklog
)

// IntegrationCaller is an alias of the generic provider escape hatch.
type IntegrationCaller = integrations.IntegrationCaller
