package honcho

import (
	"errors"
	"regexp"
)

// sessionIDPattern validates session ID format
var sessionIDPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// SessionCreate represents the request body for creating/getting a session
type SessionCreate struct {
	// ID is the session identifier (required, 1-100 characters, pattern: ^[a-zA-Z0-9_-]+$)
	ID string `json:"id"`
	// Metadata is optional metadata for the session
	Metadata map[string]any `json:"metadata,omitempty"`
	// Peers is optional map of peer IDs to their session-level configuration
	Peers map[string]*SessionPeerConfig `json:"peers,omitempty"`
	// Configuration is optional session-level configuration
	Configuration *SessionConfiguration `json:"configuration,omitempty"`
}

// Validate checks that mandatory fields are valid
func (req SessionCreate) Validate() error {
	if req.ID == "" {
		return errors.New("id is required")
	}
	if len(req.ID) > 100 {
		return errors.New("id must be 100 characters or less")
	}
	if !sessionIDPattern.MatchString(req.ID) {
		return errors.New("id must contain only letters, numbers, underscores, or hyphens")
	}
	return nil
}

// SessionUpdate represents the request body for updating a session
type SessionUpdate struct {
	// Metadata is optional metadata to update
	Metadata map[string]any `json:"metadata,omitempty"`
	// Configuration is optional configuration to update
	Configuration *SessionConfiguration `json:"configuration,omitempty"`
}

// SessionConfiguration represents the configuration options for a session
// All fields are optional. Session-level configuration overrides workspace-level configuration.
type SessionConfiguration struct {
	// Reasoning is configuration for reasoning functionality
	Reasoning *ReasoningConfiguration `json:"reasoning,omitempty"`
	// PeerCard is configuration for peer card functionality
	PeerCard *PeerCardConfiguration `json:"peer_card,omitempty"`
	// Summary is configuration for summary functionality
	Summary *SummaryConfiguration `json:"summary,omitempty"`
	// Dream is configuration for dream functionality
	Dream *DreamConfiguration `json:"dream,omitempty"`
}

// ReasoningConfiguration represents the reasoning configuration
type ReasoningConfiguration struct {
	// Enabled indicates whether to enable reasoning functionality
	Enabled *bool `json:"enabled"`
	// CustomInstructions is TODO: currently unused. Custom instructions for the reasoning system
	CustomInstructions *string `json:"custom_instructions,omitempty"`
}

// PeerCardConfiguration represents the peer card configuration
type PeerCardConfiguration struct {
	// Use indicates whether to use peer card during reasoning process
	Use *bool `json:"use"`
	// Create indicates whether to generate peer card based on content
	Create *bool `json:"create"`
}

// SummaryConfiguration represents the summary configuration
type SummaryConfiguration struct {
	// Enabled indicates whether to enable summary functionality
	Enabled *bool `json:"enabled"`
	// MessagesPerShortSummary is number of messages per short summary (minimum: 10)
	MessagesPerShortSummary *int `json:"messages_per_short_summary,omitempty"`
	// MessagesPerLongSummary is number of messages per long summary (minimum: 20, must be > messages_per_short_summary)
	MessagesPerLongSummary *int `json:"messages_per_long_summary,omitempty"`
}

// DreamConfiguration represents the dream configuration
type DreamConfiguration struct {
	// Enabled indicates whether to enable dream functionality
	Enabled *bool `json:"enabled"`
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
}

// GetSessionContextOptions represents optional parameters for GetSessionContext
type GetSessionContextOptions struct {
	// Tokens is number of tokens to use for the context (maximum: 100000)
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
}

// GetSessionPeersOptions represents optional parameters for GetSessionPeers
type GetSessionPeersOptions struct {
	// Page is the page number (default: 1, minimum: 1)
	Page int
	// Size is the page size (default: 50, minimum: 1, maximum: 100)
	Size int
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

// PageSession represents a paginated response of sessions
type PageSession struct {
	Items []Session `json:"items"`
	Total int       `json:"total"`
	Page  int       `json:"page"`
	Size  int       `json:"size"`
	Pages int       `json:"pages"`
}
