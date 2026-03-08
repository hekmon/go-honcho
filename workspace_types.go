package honcho

import (
	"errors"
	"regexp"
	"time"
)

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
