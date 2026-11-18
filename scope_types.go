package honcho

import (
	"encoding/json"
	"errors"
	"regexp"
	"time"
	"unicode/utf8"
)

// scopeIDPattern validates scope ID format
var scopeIDPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// ScopeBackfillState represents the state of a session's backfill job in a scope
type ScopeBackfillState string

const (
	// ScopeBackfillStatePending represents a backfill waiting to run or still running
	ScopeBackfillStatePending ScopeBackfillState = "pending"
	// ScopeBackfillStateCompleted represents a backfill that copied the session's documents into the scope
	ScopeBackfillStateCompleted ScopeBackfillState = "completed"
	// ScopeBackfillStateFailed represents a backfill that failed
	ScopeBackfillStateFailed ScopeBackfillState = "failed"
)

// ScopeCreate represents the request body for getting or creating a scope
type ScopeCreate struct {
	// ID is the unprefixed scope name (required, 1-506 characters, pattern: ^[a-zA-Z0-9_-]+$)
	ID string `json:"id"`
	// Metadata is optional metadata for the scope. On a scope that already exists, metadata that
	// differs from the stored metadata overwrites it: leave it nil to keep what is stored
	Metadata json.RawMessage `json:"metadata,omitempty"`
}

// Validate checks that mandatory fields are valid
func (req ScopeCreate) Validate() error {
	if req.ID == "" {
		return errors.New("id is required")
	}
	if utf8.RuneCountInString(req.ID) > 506 {
		return errors.New("id must be 506 characters or less")
	}
	if !scopeIDPattern.MatchString(req.ID) {
		return errors.New("id must contain only letters, numbers, underscores, or hyphens")
	}
	return nil
}

// Scope represents a named group of sessions that acts as a visibility boundary on recall
type Scope struct {
	// ID is the unprefixed scope name
	ID string `json:"id"`
	// Metadata is the metadata of the scope
	Metadata json.RawMessage `json:"metadata,omitempty"`
	// CreatedAt is the timestamp when the scope was created
	CreatedAt time.Time `json:"created_at"`
	// SessionCount is the number of sessions that are active members of this scope. Only returned
	// by GetScopes when IncludeSessionCount is true, nil otherwise
	SessionCount *int `json:"session_count,omitempty"`
}

// ScopeGet represents the request body for listing scopes with filters
// Use nil for no filters
type ScopeGet struct {
	// Filters is the filter object on the scope list (optional). id matches the unprefixed scope
	// name, e.g. {"id": {"icontains": "sup"}}; created_at and metadata work as on other lists
	Filters map[string]any `json:"filters,omitempty"`
}

// GetScopesOptions represents optional parameters for GetScopes
type GetScopesOptions struct {
	// Page is the page number (default: 1, minimum: 1, maximum: 2147483647)
	Page int `json:"page,omitempty"`
	// Size is the page size (default: 50, minimum: 1, maximum: 100)
	Size int `json:"size,omitempty"`
	// Reverse is whether to reverse the order of results, newest scope first (default: false)
	Reverse bool `json:"reverse,omitempty"`
	// IncludeSessionCount is whether to include each scope's member session count as SessionCount (default: false)
	IncludeSessionCount bool `json:"include_session_count,omitempty"`
}

// GetScopeSessionsOptions represents optional parameters for GetScopeSessions
type GetScopeSessionsOptions struct {
	// Page is the page number (default: 1, minimum: 1, maximum: 2147483647)
	Page int `json:"page,omitempty"`
	// Size is the page size (default: 50, minimum: 1, maximum: 100)
	Size int `json:"size,omitempty"`
	// Reverse is whether to reverse the order of results, most recently added member first (default: false)
	Reverse bool `json:"reverse,omitempty"`
}

// PageScope represents a paginated response of scopes
type PageScope struct {
	// Items is the list of scopes on this page
	Items []Scope `json:"items"`
	// Total is the number of matching scopes, exact up to the server's PAGINATION_COUNT_LIMIT
	// (default: 100000) and reported as that limit plus one when there are more
	Total int `json:"total"`
	// Page is the current page number (minimum: 1)
	Page int `json:"page"`
	// Size is the page size (minimum: 1)
	Size int `json:"size"`
	// Pages is the total number of pages (minimum: 0)
	Pages int `json:"pages"`
}

// ScopeSessionsAdd represents the request body for adding sessions to a scope
type ScopeSessionsAdd struct {
	// SessionIDs is the IDs of existing sessions to add to the scope (required, 1-100 items)
	SessionIDs []string `json:"session_ids"`
}

// Validate checks that mandatory fields are valid
func (req ScopeSessionsAdd) Validate() error {
	if len(req.SessionIDs) == 0 {
		return errors.New("session_ids must hold at least one session")
	}
	if len(req.SessionIDs) > 100 {
		return errors.New("session_ids must hold 100 sessions or less")
	}
	return nil
}

// ScopeSessionsRemove represents the request body for removing sessions from a scope
type ScopeSessionsRemove struct {
	// SessionIDs is the IDs of existing sessions to remove from the scope (required, 1-100 items).
	// Sessions that are not members are ignored
	SessionIDs []string `json:"session_ids"`
}

// Validate checks that mandatory fields are valid
func (req ScopeSessionsRemove) Validate() error {
	if len(req.SessionIDs) == 0 {
		return errors.New("session_ids must hold at least one session")
	}
	if len(req.SessionIDs) > 100 {
		return errors.New("session_ids must hold 100 sessions or less")
	}
	return nil
}

// ScopeBackfillStatus represents the backfill job of one session in a scope
type ScopeBackfillStatus struct {
	// State is the current state of the job
	State ScopeBackfillState `json:"state"`
	// UpdatedAt is the timestamp of the job's last state change
	UpdatedAt time.Time `json:"updated_at"`
	// DocsCopied is the number of documents copied into the scope, nil until the backfill completes
	DocsCopied *int `json:"docs_copied,omitempty"`
}

// ScopeStatus represents the per-session backfill job status of a scope
type ScopeStatus struct {
	// BackfillStatus maps each session that has had a backfill enqueued to the state of its job.
	// Empty when no backfill has ever been enqueued for the scope, whatever its member sessions
	BackfillStatus map[string]ScopeBackfillStatus `json:"backfill_status"`
}
