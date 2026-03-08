package honcho

import (
	"errors"
	"regexp"
	"time"
)

// peerIDPattern validates peer ID format
var peerIDPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// Peer represents a Honcho peer
type Peer struct {
	ID            string         `json:"id"`
	WorkspaceID   string         `json:"workspace_id"`
	CreatedAt     time.Time      `json:"created_at"`
	Metadata      map[string]any `json:"metadata,omitempty"`
	Configuration map[string]any `json:"configuration,omitempty"`
}

// PeerCreate represents the request body for creating/getting a peer
type PeerCreate struct {
	ID            string         `json:"id"`
	Metadata      map[string]any `json:"metadata,omitempty"`
	Configuration map[string]any `json:"configuration,omitempty"`
}

// Validate checks that mandatory fields are valid
func (req PeerCreate) Validate() error {
	if req.ID == "" {
		return errors.New("id is required")
	}
	if len(req.ID) > 100 {
		return errors.New("id must be 100 characters or less")
	}
	if !peerIDPattern.MatchString(req.ID) {
		return errors.New("id must contain only letters, numbers, underscores, or hyphens")
	}
	return nil
}

// PeerGet represents the request body for getting peers with filters
// Use nil for no filters
type PeerGet struct {
	Filters map[string]any `json:"filters,omitempty"`
}

// PagePeer represents a paginated response of peers
type PagePeer struct {
	Items []Peer `json:"items"`
	Total int    `json:"total"`
	Page  int    `json:"page"`
	Size  int    `json:"size"`
	Pages int    `json:"pages"`
}

// PeerUpdate represents the request body for updating a peer
type PeerUpdate struct {
	Metadata      map[string]any `json:"metadata,omitempty"`
	Configuration map[string]any `json:"configuration,omitempty"`
}

// PeerRepresentationGet represents the request body for getting a peer's representation
type PeerRepresentationGet struct {
	// SessionID is optional session ID within which to scope the representation
	SessionID *string `json:"session_id,omitempty"`
	// Target is optional peer ID to get the representation for, from the perspective of this peer
	Target *string `json:"target,omitempty"`
	// SearchQuery is optional input to curate the representation around semantic search results
	SearchQuery *string `json:"search_query,omitempty"`
	// SearchTopK is only used if SearchQuery is provided. Number of semantic-search-retrieved conclusions to include (1-100)
	SearchTopK *int `json:"search_top_k,omitempty"`
	// SearchMaxDistance is only used if SearchQuery is provided. Maximum distance to search for semantically relevant conclusions (0-1)
	SearchMaxDistance *float64 `json:"search_max_distance,omitempty"`
	// IncludeMostFrequent is only used if SearchQuery is provided. Whether to include the most frequent conclusions
	IncludeMostFrequent *bool `json:"include_most_frequent,omitempty"`
	// MaxConclusions is only used if SearchQuery is provided. Maximum number of conclusions to include (1-100, default: 25)
	MaxConclusions *int `json:"max_conclusions,omitempty"`
}

// RepresentationResponse represents the response for getting a peer's representation
type RepresentationResponse struct {
	Representation string `json:"representation"`
}

// PeerCardResponse represents the response for getting/setting a peer card
type PeerCardResponse struct {
	PeerCard []string `json:"peer_card,omitempty"`
}

// PeerCardSet represents the request body for setting a peer card
type PeerCardSet struct {
	PeerCard []string `json:"peer_card"`
}

// Validate checks that mandatory fields are valid
func (req PeerCardSet) Validate() error {
	if len(req.PeerCard) == 0 {
		return errors.New("peer_card is required")
	}
	return nil
}

// PeerContext represents the response for getting peer context
type PeerContext struct {
	PeerID         string   `json:"peer_id"`
	TargetID       string   `json:"target_id"`
	Representation *string  `json:"representation,omitempty"`
	PeerCard       []string `json:"peer_card,omitempty"`
}

// Session represents a Honcho session
type Session struct {
	ID            string         `json:"id"`
	IsActive      bool           `json:"is_active"`
	WorkspaceID   string         `json:"workspace_id"`
	Metadata      map[string]any `json:"metadata,omitempty"`
	Configuration map[string]any `json:"configuration,omitempty"`
	CreatedAt     time.Time      `json:"created_at"`
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

// GetAllPeersOptions represents optional parameters for GetAllPeers
type GetAllPeersOptions struct {
	Page int // Page is the page number (default: 1, minimum: 1)
	Size int // Size is the page size (default: 50, minimum: 1, maximum: 100)
}

// GetSessionsForPeerOptions represents optional parameters for GetSessionsForPeer
type GetSessionsForPeerOptions struct {
	Page int // Page is the page number (default: 1, minimum: 1)
	Size int // Size is the page size (default: 50, minimum: 1, maximum: 100)
}

// GetPeerContextOptions represents optional parameters for GetPeerContext
type GetPeerContextOptions struct {
	// Target is optional target peer to get context for, from the observer's perspective
	Target *string
	// SearchQuery is optional query to curate the representation around semantic search results
	SearchQuery *string
	// SearchTopK is number of semantic-search-retrieved conclusions to include (1-100)
	SearchTopK *int
	// SearchMaxDistance is maximum distance for semantically relevant conclusions (0-1)
	SearchMaxDistance *float64
	// IncludeMostFrequent is whether to include the most frequent conclusions (default: true)
	IncludeMostFrequent *bool
	// MaxConclusions is maximum number of conclusions to include (1-100)
	MaxConclusions *int
}

// ReasoningLevel represents the level of reasoning for chat
type ReasoningLevel string

const (
	ReasoningLevelMinimal ReasoningLevel = "minimal"
	ReasoningLevelLow     ReasoningLevel = "low"
	ReasoningLevelMedium  ReasoningLevel = "medium"
	ReasoningLevelHigh    ReasoningLevel = "high"
	ReasoningLevelMax     ReasoningLevel = "max"
)

// DialecticOptions represents the request body for chatting with a peer's representation
type DialecticOptions struct {
	// SessionID is ID of the session to scope the representation to
	SessionID *string `json:"session_id,omitempty"`
	// Target is optional peer to get the representation for, from the perspective of this peer
	Target *string `json:"target,omitempty"`
	// Query is the dialectic API prompt (1-10000 characters)
	Query string `json:"query"`
	// Stream is whether to stream the response (default: false)
	Stream bool `json:"stream,omitempty"`
	// ReasoningLevel is the level of reasoning to apply (default: low)
	ReasoningLevel ReasoningLevel `json:"reasoning_level,omitempty"`
}

// Validate checks that mandatory fields are valid
func (req DialecticOptions) Validate() error {
	if req.Query == "" {
		return errors.New("query is required")
	}
	if len(req.Query) > 10000 {
		return errors.New("query must be 10000 characters or less")
	}
	return nil
}

// DialecticResponse represents the response for chatting with a peer's representation
type DialecticResponse struct {
	Content *string `json:"content"`
}
