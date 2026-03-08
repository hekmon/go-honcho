package honcho

import (
	"errors"
	"time"
)

// Message represents a message in a session
type Message struct {
	// ID is the unique identifier for the message
	ID string `json:"id"`
	// Content is the message content
	Content string `json:"content"`
	// PeerID is the ID of the peer who sent the message
	PeerID string `json:"peer_id"`
	// SessionID is the ID of the session the message belongs to
	SessionID string `json:"session_id"`
	// Metadata is optional metadata for the message
	Metadata map[string]any `json:"metadata,omitempty"`
	// CreatedAt is the timestamp when the message was created
	CreatedAt time.Time `json:"created_at"`
	// WorkspaceID is the ID of the workspace the message belongs to
	WorkspaceID string `json:"workspace_id"`
	// TokenCount is the number of tokens in the message
	TokenCount int `json:"token_count"`
}

// MessageCreate represents the request body for creating a single message
type MessageCreate struct {
	// Content is the message content (required, 0-25000 characters)
	Content string `json:"content"`
	// PeerID is the ID of the peer sending the message (required)
	PeerID string `json:"peer_id"`
	// Metadata is optional metadata for the message
	Metadata map[string]any `json:"metadata,omitempty"`
	// Configuration is optional message-level configuration
	Configuration *MessageConfiguration `json:"configuration,omitempty"`
	// CreatedAt is optional timestamp for the message
	CreatedAt *time.Time `json:"created_at,omitempty"`
}

// Validate checks that mandatory fields are valid
func (req MessageCreate) Validate() error {
	if req.Content == "" {
		return errors.New("content is required")
	}
	if len(req.Content) > 25000 {
		return errors.New("content must be 25000 characters or less")
	}
	if req.PeerID == "" {
		return errors.New("peer_id is required")
	}
	return nil
}

// MessageBatchCreate represents the request body for creating multiple messages
type MessageBatchCreate struct {
	// Messages is the array of messages to create (required, 1-100 messages)
	Messages []MessageCreate `json:"messages"`
}

// Validate checks that mandatory fields are valid
func (req MessageBatchCreate) Validate() error {
	if len(req.Messages) == 0 {
		return errors.New("at least one message is required")
	}
	if len(req.Messages) > 100 {
		return errors.New("maximum 100 messages allowed per batch")
	}
	// Validate each message
	for i, msg := range req.Messages {
		if err := msg.Validate(); err != nil {
			return errors.New("message " + string(rune(i+1)) + ": " + err.Error())
		}
	}
	return nil
}

// MessageGet represents the request body for getting messages with filters
type MessageGet struct {
	// Filters is optional filters for the message list
	Filters map[string]any `json:"filters,omitempty"`
}

// MessageUpdate represents the request body for updating a message
type MessageUpdate struct {
	// Metadata is the metadata to update (required, will overwrite existing metadata)
	Metadata map[string]any `json:"metadata,omitempty"`
}

// Validate checks that mandatory fields are valid
func (req MessageUpdate) Validate() error {
	// Metadata is the only field and it's optional, so no validation needed
	return nil
}

// MessageConfiguration represents the configuration options for a message
// All fields are optional. Message-level configuration overrides all other configurations.
// ReasoningConfiguration is defined in session_types.go
type MessageConfiguration struct {
	// Reasoning is configuration for reasoning functionality
	Reasoning *ReasoningConfiguration `json:"reasoning,omitempty"`
}

// PageMessage represents a paginated response of messages
type PageMessage struct {
	// Items is the array of messages in the current page
	Items []Message `json:"items"`
	// Total is the total number of messages
	Total int `json:"total"`
	// Page is the current page number
	Page int `json:"page"`
	// Size is the page size
	Size int `json:"size"`
	// Pages is the total number of pages
	Pages int `json:"pages"`
}

// GetMessagesOptions represents optional parameters for GetMessages
type GetMessagesOptions struct {
	// Reverse indicates whether to reverse the order of results (default: false)
	Reverse *bool
	// Page is the page number (default: 1, minimum: 1)
	Page int
	// Size is the page size (default: 50, minimum: 1, maximum: 100)
	Size int
}

// MessageUpload represents the request body for uploading a file to create messages
type MessageUpload struct {
	// File is the file content to upload (will be converted to text and split into messages)
	File []byte `json:"-"`
	// Filename is the name of the file being uploaded
	Filename string `json:"-"`
	// PeerID is the ID of the peer sending the message (required)
	PeerID string `json:"peer_id"`
	// Metadata is optional metadata for the message (as JSON string in multipart form)
	Metadata *string `json:"metadata,omitempty"`
	// Configuration is optional message-level configuration (as JSON string in multipart form)
	Configuration *string `json:"configuration,omitempty"`
	// CreatedAt is optional timestamp for the message (as RFC3339 string in multipart form)
	CreatedAt *string `json:"created_at,omitempty"`
}

// Validate checks that mandatory fields are valid
func (req MessageUpload) Validate() error {
	if len(req.File) == 0 {
		return errors.New("file is required")
	}
	if req.PeerID == "" {
		return errors.New("peer_id is required")
	}
	return nil
}
