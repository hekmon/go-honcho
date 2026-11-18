package honcho

import (
	"encoding/json"
	"errors"
	"regexp"
	"time"
	"unicode/utf8"
)

// CreateWorkspaceRequest represents the request body for creating/getting a workspace
type CreateWorkspaceRequest struct {
	ID            string                  `json:"id"`
	Metadata      json.RawMessage         `json:"metadata,omitempty"`
	Configuration *WorkspaceConfiguration `json:"configuration,omitempty"`
}

// workspaceIDPattern validates workspace ID format
var workspaceIDPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// Validate checks that mandatory fields are valid
func (req CreateWorkspaceRequest) Validate() error {
	if req.ID == "" {
		return errors.New("id is required")
	}
	if len(req.ID) > 512 {
		return errors.New("id must be 512 characters or less")
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
	Metadata      json.RawMessage         `json:"metadata,omitempty"`
	Configuration *WorkspaceConfiguration `json:"configuration,omitempty"`
}

// WorkspaceConfiguration represents the configuration options for a workspace
type WorkspaceConfiguration struct {
	// CustomInstructions is custom instructions shared by the deriver, summarizer, dialectic and dreamer (a module's own custom_instructions at the same level takes precedence)
	CustomInstructions *string          `json:"custom_instructions,omitempty"`
	Reasoning          *ReasoningConfig `json:"reasoning,omitempty"`
	PeerCard           *PeerCardConfig  `json:"peer_card,omitempty"`
	Summary            *SummaryConfig   `json:"summary,omitempty"`
	Dream              *DreamConfig     `json:"dream,omitempty"`
	// Dialectic is configuration for the dialectic chat endpoints
	Dialectic *DialecticConfig `json:"dialectic,omitempty"`
}

// ReasoningConfig represents the reasoning configuration
type ReasoningConfig struct {
	Enabled            *bool   `json:"enabled"`
	CustomInstructions *string `json:"custom_instructions,omitempty"`
}

// PeerCardConfig represents the peer card configuration
type PeerCardConfig struct {
	Use    *bool `json:"use"`
	Create *bool `json:"create"`
	// CustomInstructions is custom instructions for peer card updates, added on top of the dream instructions (does not fall back to the top-level custom_instructions)
	CustomInstructions *string `json:"custom_instructions,omitempty"`
}

// SummaryConfig represents the summary configuration
type SummaryConfig struct {
	Enabled                 *bool `json:"enabled"`
	MessagesPerShortSummary *int  `json:"messages_per_short_summary,omitempty"`
	MessagesPerLongSummary  *int  `json:"messages_per_long_summary,omitempty"`
	// CustomInstructions is custom instructions for session summaries (overrides the top-level custom_instructions at the same level, empty string to use none)
	CustomInstructions *string `json:"custom_instructions,omitempty"`
}

// DreamConfig represents the dream configuration
type DreamConfig struct {
	Enabled *bool `json:"enabled"`
	// CustomInstructions is custom instructions for dreams (overrides the top-level custom_instructions at the same level, empty string to use none)
	CustomInstructions *string `json:"custom_instructions,omitempty"`
}

// DialecticConfig represents the dialectic configuration
type DialecticConfig struct {
	// CustomInstructions is custom instructions for the dialectic chat endpoints (overrides the top-level custom_instructions at the same level, empty string to use none)
	CustomInstructions *string `json:"custom_instructions,omitempty"`
}

// WorkspaceGetRequest represents the request body for getting all workspaces
type WorkspaceGetRequest struct {
	Filters map[string]any `json:"filters,omitempty"`
}

// UpdateWorkspaceRequest represents the request body for updating a workspace
type UpdateWorkspaceRequest struct {
	Metadata json.RawMessage `json:"metadata,omitempty"`
	// Configuration replaces the entire stored configuration. An empty non-nil value clears it.
	// A nil value leaves it unchanged.
	Configuration *WorkspaceConfiguration `json:"configuration,omitempty"`
}

// DreamType represents the type of dream to schedule
type DreamType string

const (
	// DreamTypeOmni represents an omni dream
	DreamTypeOmni DreamType = "omni"
	// DreamTypeCardRefresh represents a lightweight card-only refresh, which never creates or removes conclusions
	DreamTypeCardRefresh DreamType = "card_refresh"
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
	// Limit is the maximum number of results to return (0=use default: 10, min: 1, max: 100)
	Limit int `json:"limit,omitempty"`
}

// Validate checks that mandatory fields are valid
func (req MessageSearchOptions) Validate() error {
	if req.Query == "" {
		return errors.New("query is required")
	}
	if req.Limit != 0 && (req.Limit < 1 || req.Limit > 100) {
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

// WorkspaceChatOptions represents the request body for chatting with a whole workspace
type WorkspaceChatOptions struct {
	// Query is the workspace chat prompt (required, 1-10000 characters)
	Query string `json:"query"`
	// SessionID is an optional session to scope message tools to. Mutually exclusive with Scope
	SessionID *string `json:"session_id,omitempty"`
	// Scope is optional (unprefixed) scope names restricting recall to the union of the scopes'
	// member sessions (explicit allowlist, fail-closed: an empty union recalls nothing). Nil leaves
	// recall unrestricted and an empty non-nil list is refused (1-100 scopes). Mutually exclusive
	// with SessionID. Requires a workspace- or admin-level key
	Scope []string `json:"scope,omitempty"`
	// ReasoningLevel is the level of reasoning to apply: minimal, low, medium, high, or max (default: low)
	ReasoningLevel ReasoningLevel `json:"reasoning_level,omitempty"`
	// ResponseFormat is an optional JSON Schema (root type "object") the response must conform to.
	// When provided, the response's content is a JSON string matching this schema
	ResponseFormat json.RawMessage `json:"response_format,omitempty"`
	// IncludeEvidence is whether the response includes an evidence object listing the conclusions and
	// messages the agent read while answering, plus the tool calls it made (default: false). Evidence
	// is collated from what the agent accessed; the model is never asked to cite anything, so evidence
	// may over-report (accessed is not the same as used). WorkspaceChatStream refuses it
	IncludeEvidence bool `json:"include_evidence,omitempty"`
}

// Validate checks that mandatory fields are valid
func (req WorkspaceChatOptions) Validate() error {
	if req.Query == "" {
		return errors.New("query is required")
	}
	if utf8.RuneCountInString(req.Query) > 10000 {
		return errors.New("query must be 10000 characters or less")
	}
	// A non-nil empty scope is an allowlist of nothing. With omitempty it would silently be left
	// out of the request, turning it into unrestricted recall across the whole workspace.
	if req.Scope != nil && len(req.Scope) == 0 {
		return errors.New("scope must hold at least one scope: leave it nil for no restriction")
	}
	if len(req.Scope) > 100 {
		return errors.New("scope must hold 100 scopes or less")
	}
	return nil
}

// workspaceChatStreamRequest represents the request body WorkspaceChatStream sends
type workspaceChatStreamRequest struct {
	WorkspaceChatOptions
	// Stream asks for the answer as an event stream
	Stream bool `json:"stream"`
}
