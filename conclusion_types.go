package honcho

import "time"

// Conclusion represents a logical certainty derived from interactions between Peers.
// Conclusions form the basis of a Peer's Representation.
type Conclusion struct {
	// ID is the unique identifier for the conclusion
	ID string `json:"id"`
	// Content is the text content of the conclusion
	Content string `json:"content"`
	// ObserverID is the peer who made the conclusion
	ObserverID string `json:"observer_id"`
	// ObservedID is the peer the conclusion is about
	ObservedID string `json:"observed_id"`
	// SessionID is the session ID where the conclusion was created (optional)
	SessionID *string `json:"session_id,omitempty"`
	// CreatedAt is the timestamp when the conclusion was created
	CreatedAt time.Time `json:"created_at"`
}

// ConclusionCreate is the schema for creating a single conclusion.
type ConclusionCreate struct {
	// Content is the text content of the conclusion (required, 1-65535 characters)
	Content string `json:"content"`
	// ObserverID is the peer making the conclusion (required)
	ObserverID string `json:"observer_id"`
	// ObservedID is the peer the conclusion is about (required)
	ObservedID string `json:"observed_id"`
	// SessionID is a session ID to store the conclusion in, if specified (optional)
	SessionID *string `json:"session_id,omitempty"`
}

// ConclusionBatchCreate is the schema for batch conclusion creation with a max of 100 conclusions.
type ConclusionBatchCreate struct {
	// Conclusions is the batch of conclusions to create (required, 1-100 items)
	Conclusions []ConclusionCreate `json:"conclusions"`
}

// ConclusionGet is the schema for listing conclusions with optional filters.
type ConclusionGet struct {
	// Filters are optional filters for the conclusions list (optional)
	Filters map[string]any `json:"filters,omitempty"`
}

// ConclusionQuery is the query parameters for semantic search of conclusions.
type ConclusionQuery struct {
	// Query is the semantic search query (required)
	Query string `json:"query"`
	// TopK is the number of results to return (0=use default: 10, min: 1, max: 100)
	TopK int `json:"top_k,omitempty"`
	// Distance is the maximum cosine distance threshold for results (optional, 0-1)
	Distance *float64 `json:"distance,omitempty"`
	// Filters are additional filters to apply (optional)
	Filters map[string]any `json:"filters,omitempty"`
}

// PageConclusion is the paginated response for listing conclusions.
type PageConclusion struct {
	// Items is the list of conclusions on this page
	Items []Conclusion `json:"items"`
	// Total is the total number of conclusions matching the query
	Total int `json:"total"`
	// Page is the current page number (minimum: 1)
	Page int `json:"page"`
	// Size is the page size (minimum: 1)
	Size int `json:"size"`
	// Pages is the total number of pages (minimum: 0)
	Pages int `json:"pages"`
}

// ListConclusionsOptions are optional query parameters for listing conclusions.
type ListConclusionsOptions struct {
	// Reverse is whether to reverse the order of results (default: false)
	Reverse bool `json:"reverse,omitempty"`
	// Page is the page number (default: 1, minimum: 1)
	Page int `json:"page,omitempty"`
	// Size is the page size (default: 50, minimum: 1, maximum: 100)
	Size int `json:"size,omitempty"`
}
