package honcho

import "time"

// ConclusionLevel represents the reasoning level of a conclusion
type ConclusionLevel string

const (
	ConclusionLevelExplicit      ConclusionLevel = "explicit"
	ConclusionLevelDeductive     ConclusionLevel = "deductive"
	ConclusionLevelInductive     ConclusionLevel = "inductive"
	ConclusionLevelContradiction ConclusionLevel = "contradiction"
)

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
	// Level is the reasoning level of the conclusion (default: explicit): explicit is directly
	// extracted from messages, while deductive, inductive and contradiction are derived during dreaming
	Level ConclusionLevel `json:"level"`
	// SourceIDs are the IDs of the conclusions this one was derived from: premises for deductive,
	// supporting sources for inductive, conflicting conclusions for contradiction. Empty for
	// explicit conclusions, which derive from messages rather than from other conclusions. Never null.
	SourceIDs []string `json:"source_ids"`
	// SourceMessageIDs are the IDs of the messages the deriver cited as evidence for an explicit
	// conclusion, in citation order. Evidence may include messages from peers other than the
	// observed one. Nil for derived levels and for conclusions written before citations were recorded.
	SourceMessageIDs []string `json:"source_message_ids"`
	// TimesDerived is the number of times this conclusion has been independently derived (default: 1)
	TimesDerived int `json:"times_derived"`
	// CreatedAt is the timestamp when the conclusion was created
	CreatedAt time.Time `json:"created_at"`
}

// ConclusionCreate is the schema for creating a single conclusion.
type ConclusionCreate struct {
	// Content is the text content of the conclusion (required, 1-65535 characters). Honcho counts
	// NUL characters toward this limit, then removes them
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
	// Filters are the filters to apply (required). They must name the observer, under
	// "observer" or "observer_id", and the observed peer, under "observed" or "observed_id".
	Filters map[string]any `json:"filters,omitempty"`
}

// PageConclusion is the paginated response for listing conclusions.
type PageConclusion struct {
	// Items is the list of conclusions on this page
	Items []Conclusion `json:"items"`
	// Total is the total number of conclusions matching the query. It is zero on a cursor page
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

// ListConclusionsOptions are optional query parameters for listing conclusions.
type ListConclusionsOptions struct {
	// Reverse is whether to reverse the order of results (default: false)
	Reverse bool `json:"reverse,omitempty"`
	// Page is the page number (default: 1, minimum: 1)
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
}
