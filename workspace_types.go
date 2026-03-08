package honcho

import (
	"errors"
	"regexp"
	"time"
)

// CreateWorkspaceRequest represents the request body for creating/getting a workspace
type CreateWorkspaceRequest struct {
	ID            string                  `json:"id"`
	Metadata      map[string]any          `json:"metadata,omitempty"`
	Configuration *WorkspaceConfiguration `json:"configuration,omitempty"`
}

// workspaceIDPattern validates workspace ID format
var workspaceIDPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// Validate checks that mandatory fields are valid
func (req CreateWorkspaceRequest) Validate() error {
	if req.ID == "" {
		return errors.New("id is required")
	}
	if len(req.ID) > 100 {
		return errors.New("id must be 100 characters or less")
	}
	if !workspaceIDPattern.MatchString(req.ID) {
		return errors.New("id must contain only letters, numbers, underscores, or hyphens")
	}
	return nil
}

// Workspace represents a Honcho workspace
type Workspace struct {
	ID            string                  `json:"id"`
	CreatedAt     time.Time               `json:"created_at"`
	Metadata      map[string]any          `json:"metadata,omitempty"`
	Configuration *WorkspaceConfiguration `json:"configuration,omitempty"`
}

// WorkspaceConfiguration represents the configuration options for a workspace
type WorkspaceConfiguration struct {
	Reasoning *ReasoningConfig `json:"reasoning,omitempty"`
	PeerCard  *PeerCardConfig  `json:"peer_card,omitempty"`
	Summary   *SummaryConfig   `json:"summary,omitempty"`
	Dream     *DreamConfig     `json:"dream,omitempty"`
}

// ReasoningConfig represents the reasoning configuration
type ReasoningConfig struct {
	Enabled            bool   `json:"enabled,omitempty"`
	CustomInstructions string `json:"custom_instructions,omitempty"`
}

// PeerCardConfig represents the peer card configuration
type PeerCardConfig struct {
	Use    bool `json:"use,omitempty"`
	Create bool `json:"create,omitempty"`
}

// SummaryConfig represents the summary configuration
type SummaryConfig struct {
	Enabled                 bool `json:"enabled,omitempty"`
	MessagesPerShortSummary int  `json:"messages_per_short_summary,omitempty"`
	MessagesPerLongSummary  int  `json:"messages_per_long_summary,omitempty"`
}

// DreamConfig represents the dream configuration
type DreamConfig struct {
	Enabled bool `json:"enabled,omitempty"`
}

// WorkspaceGetRequest represents the request body for getting all workspaces
type WorkspaceGetRequest struct {
	Filters map[string]any `json:"filters,omitempty"`
}

// UpdateWorkspaceRequest represents the request body for updating a workspace
type UpdateWorkspaceRequest struct {
	Metadata      map[string]any          `json:"metadata,omitempty"`
	Configuration *WorkspaceConfiguration `json:"configuration,omitempty"`
}

// DreamType represents the type of dream to schedule
type DreamType string

const (
	// DreamTypeOmni represents an omni dream
	DreamTypeOmni DreamType = "omni"
)

// ScheduleDreamRequest represents the request body for scheduling a dream
type ScheduleDreamRequest struct {
	Observer  string    `json:"observer"`
	Observed  *string   `json:"observed,omitempty"`
	DreamType DreamType `json:"dream_type"`
	SessionID *string   `json:"session_id,omitempty"`
}

// Validate checks that mandatory fields are valid
func (req ScheduleDreamRequest) Validate() error {
	if req.Observer == "" {
		return errors.New("observer is required")
	}
	if req.DreamType == "" {
		return errors.New("dream_type is required")
	}
	return nil
}

// SessionQueueStatus represents the queue status for a specific session
type SessionQueueStatus struct {
	SessionID           *string `json:"session_id"`
	TotalWorkUnits      int     `json:"total_work_units"`
	CompletedWorkUnits  int     `json:"completed_work_units"`
	InProgressWorkUnits int     `json:"in_progress_work_units"`
	PendingWorkUnits    int     `json:"pending_work_units"`
}

// QueueStatus represents the processing queue status for a workspace
type QueueStatus struct {
	TotalWorkUnits      int                            `json:"total_work_units"`
	CompletedWorkUnits  int                            `json:"completed_work_units"`
	InProgressWorkUnits int                            `json:"in_progress_work_units"`
	PendingWorkUnits    int                            `json:"pending_work_units"`
	Sessions            *map[string]SessionQueueStatus `json:"sessions,omitempty"`
}

// MessageSearchOptions represents the request body for searching messages
type MessageSearchOptions struct {
	Query   string         `json:"query"`
	Filters map[string]any `json:"filters,omitempty"`
	Limit   int            `json:"limit,omitempty"`
}

// Validate checks that mandatory fields are valid
func (req MessageSearchOptions) Validate() error {
	if req.Query == "" {
		return errors.New("query is required")
	}
	if req.Limit < 1 || req.Limit > 100 {
		return errors.New("limit must be between 1 and 100")
	}
	return nil
}

// GetAllWorkspacesOptions represents optional parameters for GetAllWorkspaces
type GetAllWorkspacesOptions struct {
	Page int // Page is the page number (default: 1, minimum: 1)
	Size int // Size is the page size (default: 50, minimum: 1, maximum: 100)
}

// PageWorkspace represents a paginated response of workspaces
type PageWorkspace struct {
	Items []Workspace `json:"items"`
	Total int         `json:"total"`
	Page  int         `json:"page"`
	Size  int         `json:"size"`
	Pages int         `json:"pages"`
}
