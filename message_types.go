package honcho

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"mime"
	"strings"
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
	Metadata json.RawMessage `json:"metadata,omitempty"`
	// CreatedAt is the timestamp when the message was created
	CreatedAt time.Time `json:"created_at"`
	// WorkspaceID is the ID of the workspace the message belongs to
	WorkspaceID string `json:"workspace_id"`
	// TokenCount is the number of tokens in the message
	TokenCount int `json:"token_count"`
}

// MessageCreate represents the request body for creating a single message
type MessageCreate struct {
	// Content is the message content, which may be empty. Honcho refuses with a 422 a content
	// longer than its MAX_MESSAGE_SIZE setting, 25000 characters by default, counting NUL
	// characters, which it then removes
	Content string `json:"content"`
	// PeerID is the ID of the peer sending the message (required)
	PeerID string `json:"peer_id"`
	// Metadata is optional metadata for the message. Honcho removes NUL characters from its keys
	// and strings
	Metadata json.RawMessage `json:"metadata,omitempty"`
	// Configuration is optional message-level configuration
	Configuration *MessageConfiguration `json:"configuration,omitempty"`
	// CreatedAt is optional timestamp for the message
	CreatedAt *time.Time `json:"created_at,omitempty"`
}

// Validate checks that mandatory fields are valid
func (req MessageCreate) Validate() error {
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
			return fmt.Errorf("message %d: %w", i+1, err)
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
	// Metadata is optional metadata to update (will overwrite existing metadata). Honcho removes NUL
	// characters from its keys and strings
	Metadata json.RawMessage `json:"metadata,omitempty"`
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
	// Total is the total number of messages. It is zero on a cursor page
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

// GetMessagesOptions represents optional parameters for GetMessages
type GetMessagesOptions struct {
	// Reverse indicates whether to reverse the order of results (default: false)
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

// MessageUpload represents the request for uploading a file to create messages. It is sent as a
// multipart form: the file as a part declaring its media type, and each other optional field as a
// form field written only when set
type MessageUpload struct {
	// File is the file content to upload (will be converted to text and split into messages)
	File []byte `json:"-"`
	// Filename is the name of the file being uploaded
	Filename string `json:"-"`
	// ContentType is the media type of the file, which Honcho reads the file according to. Honcho
	// accepts application/pdf, application/json (exactly, without parameters) and any text/* type,
	// such as text/plain or text/markdown, and answers 415 for any other type. Leave it empty to
	// detect it from the content with http.DetectContentType, which tells PDFs and text apart: a
	// JSON file is then read as text, so set application/json to have it read as JSON. Validate
	// refuses a value mime.ParseMediaType cannot parse, such as text plain, or holding a line break,
	// which would break the header of the file part
	ContentType string `json:"-"`
	// PeerID is the ID of the peer sending the message (required)
	PeerID string `json:"peer_id"`
	// Metadata is optional metadata for the messages, a JSON object, sent as given as the metadata
	// form field when not empty. Leave it empty for no metadata: Validate refuses a non-empty value
	// that is not a JSON object, as the server would otherwise silently drop invalid JSON, creating
	// the messages without metadata, and store null as no metadata. Honcho removes NUL characters
	// from its keys and strings
	Metadata json.RawMessage `json:"metadata,omitempty"`
	// Configuration is optional message-level configuration, sent marshalled to JSON as the
	// configuration form field when not nil
	Configuration *MessageConfiguration `json:"configuration,omitempty"`
	// CreatedAt is optional timestamp for the messages, sent formatted with time.RFC3339Nano as the
	// created_at form field when not nil. Its year, in its own location, must be between 1 and 9999:
	// the server cannot parse any other, and would silently use the time of the upload instead
	CreatedAt *time.Time `json:"created_at,omitempty"`
}

// Validate checks that mandatory fields are valid, and that the optional fields hold values the
// server parses rather than silently drops
func (req MessageUpload) Validate() error {
	if len(req.File) == 0 {
		return errors.New("file is required")
	}
	if req.PeerID == "" {
		return errors.New("peer_id is required")
	}
	if req.ContentType != "" {
		if _, _, err := mime.ParseMediaType(req.ContentType); err != nil {
			return fmt.Errorf("content_type must be a media type such as text/plain: %w", err)
		}
		// ParseMediaType skips a line break around the type or between parameters, which would
		// still break the header of the file part
		if strings.ContainsAny(req.ContentType, "\r\n") {
			return errors.New("content_type must not hold a line break")
		}
	}
	// Valid JSON whose first character, after any whitespace, is "{" holds an object
	if len(req.Metadata) > 0 && (!json.Valid(req.Metadata) || bytes.TrimLeft(req.Metadata, " \t\r\n")[0] != '{') {
		return errors.New("metadata must be a JSON object: leave it empty for no metadata")
	}
	if req.CreatedAt != nil {
		if year := req.CreatedAt.Year(); year < 1 || year > 9999 {
			return errors.New("created_at year must be between 1 and 9999")
		}
	}
	return nil
}
