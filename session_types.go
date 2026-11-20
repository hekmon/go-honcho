package honcho

import (
	"encoding/json"
	"errors"
	"regexp"
	"time"
)

// sessionIDPattern validates session ID format
var sessionIDPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// SessionCreate represents the request body for creating/getting a session
type SessionCreate struct {
	// ID is the session identifier (required, 1-512 characters, pattern: ^[a-zA-Z0-9_-]+$)
	ID string `json:"id"`
	// Metadata is optional metadata for the session. If the session already exists, it replaces the
	// stored metadata: an empty object, {}, erases it. Honcho removes NUL characters from its keys
	// and strings
	Metadata json.RawMessage `json:"metadata,omitempty"`
	// Peers is optional map of peer IDs to their session-level configuration. If the session already
	// exists, members not listed stay. A peer already a member keeps its stored configuration, as
	// with AddPeersToSession, and a peer that had left rejoins with the sent configuration
	Peers map[string]*SessionPeerConfig `json:"peers,omitempty"`
	// Configuration is optional session-level configuration. If the session already exists, it is
	// merged as by UpdateSession: a section that is sent replaces the stored section of that name,
	// so a field left nil inside it is erased, and other sections are kept
	Configuration *SessionConfiguration `json:"configuration,omitempty"`
	// Scopes is the optional names of the scopes to add this session to (maximum: 100), even if the
	// session already exists. A scope that does not exist is created. If the session already has
	// messages, its existing documents are backfilled into the scope asynchronously
	Scopes []string `json:"scopes,omitempty"`
}

// Validate checks that mandatory fields are valid
func (req SessionCreate) Validate() error {
	if req.ID == "" {
		return errors.New("id is required")
	}
	if len(req.ID) > 512 {
		return errors.New("id must be 512 characters or less")
	}
	if !sessionIDPattern.MatchString(req.ID) {
		return errors.New("id must contain only letters, numbers, underscores, or hyphens")
	}
	if len(req.Scopes) > 100 {
		return errors.New("scopes must hold 100 scopes or less")
	}
	return nil
}

// SessionUpdate represents the request body for updating a session
type SessionUpdate struct {
	// Metadata is optional metadata to update. Honcho removes NUL characters from its keys and
	// strings
	Metadata json.RawMessage `json:"metadata,omitempty"`
	// Configuration is optional configuration to update: a section that is sent replaces the stored section of that name, other sections are kept
	Configuration *SessionConfiguration `json:"configuration,omitempty"`
}

// SessionConfiguration represents the configuration options for a session
// All fields are optional. Session-level configuration overrides workspace-level configuration.
type SessionConfiguration struct {
	// CustomInstructions is custom instructions shared by the deriver, summarizer, dialectic and dreamer (a module's own custom_instructions at the same level takes precedence)
	CustomInstructions *string `json:"custom_instructions,omitempty"`
	// Reasoning is configuration for reasoning functionality
	Reasoning *ReasoningConfiguration `json:"reasoning,omitempty"`
	// PeerCard is configuration for peer card functionality
	PeerCard *PeerCardConfiguration `json:"peer_card,omitempty"`
	// Summary is configuration for summary functionality
	Summary *SummaryConfiguration `json:"summary,omitempty"`
	// Dream is configuration for dream functionality
	Dream *DreamConfiguration `json:"dream,omitempty"`
	// Dialectic is configuration for the dialectic chat endpoints
	Dialectic *DialecticConfiguration `json:"dialectic,omitempty"`
}

// ReasoningConfiguration represents the reasoning configuration
type ReasoningConfiguration struct {
	// Enabled indicates whether to enable reasoning functionality
	Enabled *bool `json:"enabled"`
	// CustomInstructions is custom instructions for the deriver (overrides the top-level custom_instructions at the same level, empty string to use none)
	CustomInstructions *string `json:"custom_instructions,omitempty"`
}

// PeerCardConfiguration represents the peer card configuration
type PeerCardConfiguration struct {
	// Use indicates whether to use peer card during reasoning process
	Use *bool `json:"use"`
	// Create indicates whether to generate peer card based on content
	Create *bool `json:"create"`
	// CustomInstructions is custom instructions for peer card updates, added on top of the dream instructions (does not fall back to the top-level custom_instructions)
	CustomInstructions *string `json:"custom_instructions,omitempty"`
}

// SummaryConfiguration represents the summary configuration
type SummaryConfiguration struct {
	// Enabled indicates whether to enable summary functionality
	Enabled *bool `json:"enabled"`
	// MessagesPerShortSummary is number of messages per short summary (minimum: 10)
	MessagesPerShortSummary *int `json:"messages_per_short_summary,omitempty"`
	// MessagesPerLongSummary is number of messages per long summary (minimum: 20, must be > messages_per_short_summary)
	MessagesPerLongSummary *int `json:"messages_per_long_summary,omitempty"`
	// CustomInstructions is custom instructions for session summaries (overrides the top-level custom_instructions at the same level, empty string to use none)
	CustomInstructions *string `json:"custom_instructions,omitempty"`
}

// DreamConfiguration represents the dream configuration
type DreamConfiguration struct {
	// Enabled indicates whether to enable dream functionality
	Enabled *bool `json:"enabled"`
	// CustomInstructions is custom instructions for dreams (overrides the top-level custom_instructions at the same level, empty string to use none)
	CustomInstructions *string `json:"custom_instructions,omitempty"`
}

// DialecticConfiguration represents the dialectic configuration
type DialecticConfiguration struct {
	// CustomInstructions is custom instructions for the dialectic chat endpoints (overrides the top-level custom_instructions at the same level, empty string to use none)
	CustomInstructions *string `json:"custom_instructions,omitempty"`
}

// SessionPeerConfig represents the session-level configuration for a peer
type SessionPeerConfig struct {
	// ObserveMe indicates whether Honcho will use reasoning to form a representation of this peer
	ObserveMe *bool `json:"observe_me"`
	// ObserveOthers indicates whether this peer should form a session-level theory-of-mind representation of other peers
	ObserveOthers *bool `json:"observe_others"`
}

// SessionContext represents the response for getting session context
type SessionContext struct {
	ID                 string    `json:"id"`
	Messages           []Message `json:"messages"`
	Summary            *Summary  `json:"summary,omitempty"`
	PeerRepresentation *string   `json:"peer_representation,omitempty"`
	PeerCard           []string  `json:"peer_card,omitempty"`
}

// Summary represents a session summary
type Summary struct {
	Content     string `json:"content"`
	MessageID   string `json:"message_id"`
	SummaryType string `json:"summary_type"`
	CreatedAt   string `json:"created_at"`
	TokenCount  int    `json:"token_count"`
}

// SessionSummaries represents the response for getting session summaries
type SessionSummaries struct {
	ID           string   `json:"id"`
	ShortSummary *Summary `json:"short_summary,omitempty"`
	LongSummary  *Summary `json:"long_summary,omitempty"`
}

// GetSessionsOptions represents optional parameters for GetSessions
type GetSessionsOptions struct {
	// Page is the page number (default: 1, minimum: 1)
	Page int
	// Size is the page size (default: 50, minimum: 1, maximum: 100)
	Size int
	// Cursor opts in to cursor pagination when not nil (default: nil, for page numbers). An empty
	// cursor, new(string), asks for the first page: then set Cursor to each page's NextPage until
	// that is nil. A cursor page has no total and no page numbers. A cursor holds only a position
	// and is only valid for the list that returned it, so keep the other options and any filters
	// unchanged while following cursors. Page cannot be combined with Cursor: the request is
	// refused before it is sent, as the server would ignore the page
	Cursor *string `json:"cursor,omitempty"`
	// Reverse is whether to reverse the order of results, newest session first (default: false)
	Reverse bool `json:"reverse,omitempty"`
	// IncludeScopes is whether to return the names of each session's scopes in Session.Scopes (default: false)
	IncludeScopes bool `json:"include_scopes,omitempty"`
}

// GetSessionContextOptions represents optional parameters for GetSessionContext
type GetSessionContextOptions struct {
	// Tokens is number of tokens to use for the context (maximum: the server's
	// GET_CONTEXT_MAX_TOKENS setting, 100000 by default)
	Tokens *int
	// SearchQuery is a query string used to fetch semantically relevant conclusions
	SearchQuery *string
	// Summary indicates whether to include a summary if available (default: true)
	Summary *bool
	// PeerTarget is the target of the perspective
	PeerTarget *string
	// PeerPerspective is a peer to get context for (must be provided with PeerTarget)
	PeerPerspective *string
	// LimitToSession is only used if SearchQuery is provided (default: false)
	LimitToSession *bool
	// SearchTopK is only used if SearchQuery is provided (1-100)
	SearchTopK *int
	// SearchMaxDistance is only used if SearchQuery is provided (0-1)
	SearchMaxDistance *float64
	// IncludeMostFrequent is only used if SearchQuery is provided (default: false)
	IncludeMostFrequent *bool
	// MaxConclusions is only used if SearchQuery is provided (1-100)
	MaxConclusions *int
	// Scope is the name of a scope to use as the perspective source (optional, nil for none): the
	// representation and peer card of PeerTarget are read from the scope's observations instead of the
	// global (or PeerPerspective) view. It requires PeerTarget and excludes PeerPerspective and Sessions,
	// which the server reports with a 422. Requires a workspace- or admin-level key
	Scope *string
	// Sessions is an optional allowlist of session IDs confining the representation of PeerTarget to
	// those sessions (nil for no restriction). Recall is restricted to conclusions stated directly in
	// the allowed sessions, and the peer card is omitted. It requires PeerTarget, must include this
	// session and excludes Scope and LimitToSession, which the server reports with a 422. An empty
	// non-nil list is refused by GetSessionContext, as no parameter would be sent and the restriction
	// would be lost. Each entry is sent as its own parameter, so a list of a few hundred sessions can
	// exceed what the server accepts in a request line: use Scope for large or reusable session sets
	Sessions []string
}

// GetSessionPeersOptions represents optional parameters for GetSessionPeers
type GetSessionPeersOptions struct {
	// Page is the page number (default: 1, minimum: 1)
	Page int
	// Size is the page size (default: 50, minimum: 1, maximum: 100)
	Size int
	// Cursor opts in to cursor pagination when not nil (default: nil, for page numbers). An empty
	// cursor, new(string), asks for the first page: then set Cursor to each page's NextPage until
	// that is nil. A cursor page has no total and no page numbers. A cursor holds only a position
	// and is only valid for the list that returned it, so keep the other options and any filters
	// unchanged while following cursors. Page cannot be combined with Cursor: the request is
	// refused before it is sent, as the server would ignore the page
	Cursor *string
}

// CloneSessionOptions represents optional parameters for CloneSession
type CloneSessionOptions struct {
	// MessageID is optional message ID to cut off the clone at
	MessageID *string
}

// SessionGet represents the request body for getting sessions with filters
// Use nil for no filters
type SessionGet struct {
	Filters map[string]any `json:"filters,omitempty"`
}

// WorkspaceSessionGet represents the request body for listing the sessions of a workspace with GetSessions
// Use nil for no filters
//
// It is separate from SessionGet, the request body of GetSessionsForPeer, mirroring upstream's two types.
// The peer route silently ignores scope and in_no_scope and returns every session, so with a shared type
// a GetSessionsForPeer caller could believe the results were restricted when they are not
type WorkspaceSessionGet struct {
	// Filters is the filter object on the session list (optional)
	Filters map[string]any `json:"filters,omitempty"`
	// Scope is the name of the scope to list the active members of (optional, nil for no restriction).
	// It combines with Filters. The server answers an unknown scope with a 404, an empty name with a
	// 422, and Scope set together with InNoScope with a 422
	Scope *string `json:"scope,omitempty"`
	// InNoScope is whether to list only the sessions in no scope (default: false). The server answers
	// InNoScope set together with Scope with a 422
	InNoScope bool `json:"in_no_scope,omitempty"`
}

// Session represents a Honcho session
type Session struct {
	ID string `json:"id"`
	// UID is the immutable identifier Honcho assigns to this session. ID is the name the client chose, UID is generated by the server
	UID         string          `json:"uid"`
	IsActive    bool            `json:"is_active"`
	WorkspaceID string          `json:"workspace_id"`
	Metadata    json.RawMessage `json:"metadata,omitempty"`
	// Configuration is the session-level configuration, nil when the response holds none
	Configuration *SessionConfiguration `json:"configuration,omitempty"`
	CreatedAt     time.Time             `json:"created_at"`
	// Scopes is the names of the scopes the session is in. It is nil unless GetSessionsOptions.IncludeScopes
	// is set, and empty for a session in no scope
	Scopes []string `json:"scopes,omitempty"`
}

// PageSession represents a paginated response of sessions
type PageSession struct {
	// Items is the list of sessions on this page
	Items []Session `json:"items"`
	// Total is the total number of matching sessions. It is zero on a cursor page
	Total int `json:"total"`
	// Page is the current page number. It is zero on a cursor page
	Page int `json:"page"`
	// Size is the page size. It is zero on a cursor page
	Size int `json:"size"`
	// Pages is the total number of pages. It is zero on a cursor page
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
