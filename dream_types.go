package honcho

import "time"

// DreamRunStatus represents the state of a dream run. Failed and interrupted runs can
// still have changes: each change is committed as it is made.
type DreamRunStatus string

const (
	// DreamRunStatusRunning represents a dream still in progress
	DreamRunStatusRunning DreamRunStatus = "running"
	// DreamRunStatusCompleted represents a dream that ran to its end
	DreamRunStatusCompleted DreamRunStatus = "completed"
	// DreamRunStatusFailed represents a dream that failed
	DreamRunStatusFailed DreamRunStatus = "failed"
	// DreamRunStatusInterrupted represents a dream that was interrupted
	DreamRunStatusInterrupted DreamRunStatus = "interrupted"
)

// DreamConclusionStatus represents what became of a conclusion a dream references,
// resolved when the run is read.
type DreamConclusionStatus string

const (
	// DreamConclusionStatusCurrent represents a conclusion still in the representation
	DreamConclusionStatusCurrent DreamConclusionStatus = "current"
	// DreamConclusionStatusRemoved represents a conclusion a later dream removed, named by RemovedInRun
	DreamConclusionStatusRemoved DreamConclusionStatus = "removed"
	// DreamConclusionStatusUnavailable represents a conclusion that was deleted (through the API, with its
	// session, or by the deriver), or whose removing dream is past the server's retention window
	DreamConclusionStatusUnavailable DreamConclusionStatus = "unavailable"
)

// DreamRunList represents the request body for listing dream runs with filters
// Use nil for no filters
type DreamRunList struct {
	// ObservedID keeps only the runs about this peer (optional)
	ObservedID *string `json:"observed_id,omitempty"`
	// ObserverID keeps only the runs from this observer's perspective (optional). Combine it with
	// ObservedID for one representation
	ObserverID *string `json:"observer_id,omitempty"`
	// Filters is the filter object, e.g. {"started_at": {"gte": "2026-01-01"}} (optional). Filterable
	// fields: started_at, finished_at, observer_id, observed_id. Combined with ObservedID and ObserverID by AND
	Filters map[string]any `json:"filters,omitempty"`
}

// ListDreamsOptions represents optional parameters for ListDreams
type ListDreamsOptions struct {
	Page int // Page is the page number (default: 1, minimum: 1, maximum: 2147483647)
	Size int // Size is the page size (default: 50, minimum: 1, maximum: 100)
	// Cursor opts in to cursor pagination when not nil (default: nil, for page numbers). An empty
	// cursor, new(string), asks for the first page: then set Cursor to each page's NextPage until
	// that is nil. A cursor page has no total and no page numbers. A cursor holds only a position
	// and is only valid for the list that returned it, so keep the other options and any filters
	// unchanged while following cursors. Page cannot be combined with Cursor: the request is
	// refused before it is sent, as the server would ignore the page
	Cursor *string
}

// PageDreamRun represents a paginated response of dream runs
type PageDreamRun struct {
	// Items is the list of dream runs on this page, newest first
	Items []DreamRun `json:"items"`
	// Total is the number of matching runs, exact up to the server's PAGINATION_COUNT_LIMIT
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

// DreamRun represents one dream over a representation: the observer's model of the observed peer
type DreamRun struct {
	// ID is the dream's run ID
	ID string `json:"id"`
	// ObserverID is the peer whose representation the dream changed
	ObserverID string `json:"observer_id"`
	// ObservedID is the peer the representation is about
	ObservedID string `json:"observed_id"`
	// DreamType is the kind of dream, e.g. omni (deduction and induction) or card_refresh (optional)
	DreamType *DreamType `json:"dream_type,omitempty"`
	// TriggerReason is what scheduled the dream, e.g. document_threshold or manual (optional)
	TriggerReason *string `json:"trigger_reason,omitempty"`
	// Status is the state of the run. Failed and interrupted runs can still have changes: each
	// change is committed as it is made
	Status DreamRunStatus `json:"status"`
	// CreatedCount is the number of conclusions the dream created
	CreatedCount int `json:"created_count"`
	// RemovedCount is the number of conclusions the dream removed
	RemovedCount int `json:"removed_count"`
	// PeerCardUpdated is whether the dream changed the peer card
	PeerCardUpdated bool `json:"peer_card_updated"`
	// StartedAt is the timestamp when the dream started
	StartedAt time.Time `json:"started_at"`
	// FinishedAt is the timestamp when the dream finished, nil while it has not
	FinishedAt *time.Time `json:"finished_at,omitempty"`
	// PrunedAt is when retention first deleted some of this run's changes. Nil unless a retention
	// window (DREAM_HISTORY_RETENTION_DAYS, unset by default) pruned the run, which can predate the
	// current setting. Past the window a run keeps only the created conclusions that still exist:
	// Removed and PeerCard are gone, while the counts stay as the dream recorded them
	PrunedAt *time.Time `json:"pruned_at,omitempty"`
}

// DreamRunDetail represents a dream run with every change it committed
type DreamRunDetail struct {
	// DreamRun is the run the changes belong to
	DreamRun
	// Created is the conclusions the dream created, with the conclusions each was derived from
	Created []DreamCreatedConclusion `json:"created"`
	// Removed is the conclusions the dream removed, as they read when removed
	Removed []DreamRemovedConclusion `json:"removed"`
	// PeerCard is the peer card before and after the dream, nil when the dream did not change it
	PeerCard *DreamPeerCardChange `json:"peer_card,omitempty"`
}

// DreamConclusionRef represents a conclusion referenced by a dream, resolved when the run is read
type DreamConclusionRef struct {
	// ID is the unique identifier for the conclusion
	ID string `json:"id"`
	// Content is the text content of the conclusion, nil when the conclusion is unavailable
	Content *string `json:"content,omitempty"`
	// Level is the reasoning level of the conclusion, nil when the conclusion is unavailable
	Level *ConclusionLevel `json:"level,omitempty"`
	// Status is what became of the conclusion since the dream
	Status DreamConclusionStatus `json:"status"`
	// RemovedInRun is the run ID of the later dream that removed the conclusion, nil unless Status is removed
	RemovedInRun *string `json:"removed_in_run,omitempty"`
}

// DreamCreatedConclusion represents a conclusion a dream created, with the conclusions it was derived from
type DreamCreatedConclusion struct {
	// Conclusion is the conclusion the dream created
	Conclusion DreamConclusionRef `json:"conclusion"`
	// Sources is the premises for deductive conclusions, the supporting evidence for inductive ones
	Sources []DreamConclusionRef `json:"sources"`
	// Specialist is the dream specialist that made the change (optional)
	Specialist *string `json:"specialist,omitempty"`
}

// DreamRemovedConclusion represents a conclusion a dream removed, as it read when it was removed
type DreamRemovedConclusion struct {
	// ID is the unique identifier for the conclusion
	ID string `json:"id"`
	// Content is the text content of the conclusion when it was removed
	Content string `json:"content"`
	// Level is the reasoning level of the conclusion (optional)
	Level *ConclusionLevel `json:"level,omitempty"`
	// ReplacedBy is the new conclusion from the same dream that superseded this one as a near-duplicate (optional)
	ReplacedBy *string `json:"replaced_by,omitempty"`
	// Specialist is the dream specialist that made the change (optional)
	Specialist *string `json:"specialist,omitempty"`
}

// DreamPeerCardChange represents the peer card before the dream's first update and after its last
type DreamPeerCardChange struct {
	// Before is the peer card before the dream, nil when there was no card
	Before []string `json:"before"`
	// After is the peer card after the dream
	After []string `json:"after"`
}
