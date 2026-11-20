package honcho

import (
	"encoding/json"
	"errors"
	"regexp"
	"slices"
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
	// differs from the stored metadata overwrites it: leave it nil to keep what is stored. Honcho
	// removes NUL characters from its keys and strings
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
	// Cursor opts in to cursor pagination when not nil (default: nil, for page numbers). An empty
	// cursor, new(string), asks for the first page: then set Cursor to each page's NextPage until
	// that is nil. A cursor page has no total and no page numbers. A cursor holds only a position
	// and is only valid for the list that returned it, so keep the other options and any filters
	// unchanged while following cursors. Page cannot be combined with Cursor: the request is
	// refused before it is sent, as the server would ignore the page
	Cursor *string `json:"cursor,omitempty"`
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
	// Cursor opts in to cursor pagination when not nil (default: nil, for page numbers). An empty
	// cursor, new(string), asks for the first page: then set Cursor to each page's NextPage until
	// that is nil. A cursor page has no total and no page numbers. A cursor holds only a position
	// and is only valid for the list that returned it, so keep the other options and any filters
	// unchanged while following cursors. Page cannot be combined with Cursor: the request is
	// refused before it is sent, as the server would ignore the page
	Cursor *string `json:"cursor,omitempty"`
	// Reverse is whether to reverse the order of results, most recently added member first (default: false)
	Reverse bool `json:"reverse,omitempty"`
}

// PageScope represents a paginated response of scopes
type PageScope struct {
	// Items is the list of scopes on this page
	Items []Scope `json:"items"`
	// Total is the number of matching scopes, exact up to the server's PAGINATION_COUNT_LIMIT
	// (default: 100000) and reported as that limit plus one when there are more. It is zero on a
	// cursor page
	Total int `json:"total"`
	// Page is the current page number (minimum: 1). It is zero on a cursor page
	Page int `json:"page"`
	// Size is the page size (minimum: 1). It is zero on a cursor page
	Size int `json:"size"`
	// Pages is the total number of pages (minimum: 0). It is zero on a cursor page
	Pages int `json:"pages"`
	// CurrentPage is the cursor to refetch the current page. It is set only on a cursor page
	CurrentPage *string `json:"current_page,omitempty"`
	// CurrentPageBackwards is the cursor to refetch the current page starting from the last item.
	// It is set only on a cursor page
	CurrentPageBackwards *string `json:"current_page_backwards,omitempty"`
	// PreviousPage is the cursor for the previous page. It is set only on a cursor page, and is nil
	// on the first page
	PreviousPage *string `json:"previous_page,omitempty"`
	// NextPage is the cursor for the next page, to set as Cursor for the next request. It is set
	// only on a cursor page, and is nil on the last page
	NextPage *string `json:"next_page,omitempty"`
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

// ScopeOption restricts what a read route can recall to one scope or to a union of scopes. Build it
// with OneScope or ScopeUnion, and leave the field holding it nil for no restriction.
//
// The two forms mean different things, even for the same single name:
//   - OneScope is sent as a JSON string. It answers from the scope's own representation of the target
//     peer: conclusion recall is confined to what the scope observed, and message recall to the
//     scope's member sessions.
//   - ScopeUnion is sent as a JSON array. It keeps the peer as the observer and restricts recall to the
//     union of the scopes' member sessions (explicit allowlist, fail-closed: scopes without any member
//     session recall nothing).
//
// The zero value is a union of no scopes: like ScopeUnion without names, it is sent as an empty array,
// which the server refuses with a 422, so it can never turn into unrestricted recall.
type ScopeOption struct {
	// one is the name of the single scope, nil for a union
	one *string
	// union is the names of the scopes of a union
	union []string
}

// OneScope returns the option answering from the named scope's own representation of the target peer.
// This is not what a union of that one name means: see ScopeOption. The server answers a scope that
// does not exist with a 404, and a scope with no member sessions recalls nothing.
func OneScope(name string) *ScopeOption {
	return &ScopeOption{one: &name}
}

// ScopeUnion returns the option restricting recall to the union of the named scopes' member sessions
// (1-100 scopes). This is not what OneScope means, even for a single name: see ScopeOption. The server
// answers a scope that does not exist with a 404 and a union of no names with a 422, and scopes with
// no member sessions recall nothing.
func ScopeUnion(names ...string) *ScopeOption {
	// Cloned so that changing the caller's slice afterwards cannot change the restriction.
	return &ScopeOption{union: slices.Clone(names)}
}

// MarshalJSON encodes one scope as a JSON string and a union as a JSON array. A union of no names is
// encoded as [] and never as null, so that the server refuses it.
func (option ScopeOption) MarshalJSON() ([]byte, error) {
	if option.one != nil {
		return json.Marshal(*option.one)
	}
	if option.union == nil {
		return []byte("[]"), nil
	}
	return json.Marshal(option.union)
}
