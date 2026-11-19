package honcho

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"strconv"
	"time"
)

// https://docs.honcho.dev/v3/api-reference/endpoint/messages/create-messages-for-session
func (c *Client) CreateMessagesForSession(ctx context.Context, workspaceID, sessionID string, req MessageBatchCreate) (result []*Message, err error) {
	// Validate request
	if err = req.Validate(); err != nil {
		return
	}
	// Construct URL
	requestURL, err := c.endpoint(workspaceBaseURI, workspaceID, "sessions", sessionID, "messages")
	if err != nil {
		return
	}
	// Initialize result
	result = make([]*Message, 0, len(req.Messages))
	// Make request
	if _, err = c.request(ctx, http.MethodPost, requestURL, nil, req, &result); err != nil {
		err = fmt.Errorf("failed to create messages: %w", err)
		return
	}
	return
}

// CreateMessagesWithFile creates messages from an uploaded file.
//
// Honcho converts the file to text and splits it into messages from req.PeerID. The request is a
// multipart form. Its file part declares req.ContentType, which req.Validate checks is a well-formed
// media type, or the media type http.DetectContentType finds in the content when that is empty:
// Honcho reads the file according to it, and answers 415 for a type it does not read.
//
// Metadata, configuration and created_at are form fields written only when set: metadata as the
// JSON given, configuration marshalled to JSON, and created_at formatted with time.RFC3339Nano.
// Honcho silently drops such a field when it cannot parse it, still creating the messages, so
// req.Validate refuses metadata that is not a JSON object and a created_at outside the years 1 to
// 9999 before anything is sent.
//
// https://docs.honcho.dev/v3/api-reference/endpoint/messages/create-messages-with-file
func (c *Client) CreateMessagesWithFile(ctx context.Context, workspaceID, sessionID string, req MessageUpload) (result []*Message, err error) {
	// Validate request
	if err = req.Validate(); err != nil {
		return
	}
	// Construct URL
	requestURL, err := c.endpoint(workspaceBaseURI, workspaceID, "sessions", sessionID, "messages", "upload")
	if err != nil {
		return
	}
	// Build multipart form
	bodyBuffer := &bytes.Buffer{}
	writer := multipart.NewWriter(bodyBuffer)
	// Add file field, declaring the media type Honcho reads the file according to: CreateFormFile
	// would declare application/octet-stream, which Honcho refuses
	contentType := req.ContentType
	if contentType == "" {
		contentType = http.DetectContentType(req.File)
	}
	fileHeader := make(textproto.MIMEHeader)
	fileHeader.Set("Content-Disposition", multipart.FileContentDisposition("file", req.Filename))
	fileHeader.Set("Content-Type", contentType)
	fileWriter, err := writer.CreatePart(fileHeader)
	if err != nil {
		err = fmt.Errorf("failed to create form file: %w", err)
		return
	}
	if _, err = fileWriter.Write(req.File); err != nil {
		err = fmt.Errorf("failed to write file to form: %w", err)
		return
	}
	// Add peer_id field
	if err = writer.WriteField("peer_id", req.PeerID); err != nil {
		err = fmt.Errorf("failed to write peer_id field: %w", err)
		return
	}
	// Add optional metadata field, which Validate checked is a JSON object
	if len(req.Metadata) > 0 {
		if err = writer.WriteField("metadata", string(req.Metadata)); err != nil {
			err = fmt.Errorf("failed to write metadata field: %w", err)
			return
		}
	}
	// Add optional configuration field, marshalled to JSON
	if req.Configuration != nil {
		var configuration []byte
		if configuration, err = json.Marshal(req.Configuration); err != nil {
			err = fmt.Errorf("failed to marshal configuration: %w", err)
			return
		}
		if err = writer.WriteField("configuration", string(configuration)); err != nil {
			err = fmt.Errorf("failed to write configuration field: %w", err)
			return
		}
	}
	// Add optional created_at field, keeping its nanoseconds and its zone
	if req.CreatedAt != nil {
		if err = writer.WriteField("created_at", req.CreatedAt.Format(time.RFC3339Nano)); err != nil {
			err = fmt.Errorf("failed to write created_at field: %w", err)
			return
		}
	}
	// Close multipart writer
	if err = writer.Close(); err != nil {
		err = fmt.Errorf("failed to close multipart writer: %w", err)
		return
	}
	// Initialize result
	result = make([]*Message, 0)
	// Build headers with Content-Type
	headers := make(http.Header)
	headers.Set("Content-Type", writer.FormDataContentType())
	// Make request with bytes.Buffer body (multipart data)
	if _, err = c.request(ctx, http.MethodPost, requestURL, headers, bodyBuffer, &result); err != nil {
		err = fmt.Errorf("failed to upload file and create messages: %w", err)
		return
	}
	return
}

// https://docs.honcho.dev/v3/api-reference/endpoint/messages/get-message
func (c *Client) GetMessage(ctx context.Context, workspaceID, sessionID, messageID string) (result *Message, err error) {
	// Construct URL
	requestURL, err := c.endpoint(workspaceBaseURI, workspaceID, "sessions", sessionID, "messages", messageID)
	if err != nil {
		return
	}
	// Initialize result
	result = new(Message)
	// Make request
	if _, err = c.request(ctx, http.MethodGet, requestURL, nil, nil, result); err != nil {
		err = fmt.Errorf("failed to get message: %w", err)
		return
	}
	return
}

// GetMessages gets all Messages for a Session, paginated with optional filters.
//
// The list is paginated by page number, or by cursor when opts.Cursor is set.
//
// https://docs.honcho.dev/v3/api-reference/endpoint/messages/get-messages
func (c *Client) GetMessages(ctx context.Context, workspaceID, sessionID string, req *MessageGet, opts *GetMessagesOptions) (result *PageMessage, err error) {
	// Construct URL
	requestURL, err := c.endpoint(workspaceBaseURI, workspaceID, "sessions", sessionID, "messages", "list")
	if err != nil {
		return
	}
	// Add query parameters
	queryParams := requestURL.Query()
	if opts != nil {
		if opts.Reverse {
			queryParams.Set("reverse", "true")
		}
		if opts.Page > 0 {
			queryParams.Set("page", strconv.Itoa(opts.Page))
		}
		if opts.Size > 0 {
			queryParams.Set("size", strconv.Itoa(opts.Size))
		}
		if err = setCursor(queryParams, opts.Page, opts.Cursor); err != nil {
			return
		}
	}
	requestURL.RawQuery = queryParams.Encode()
	// Initialize result
	result = new(PageMessage)
	// Make request
	if _, err = c.request(ctx, http.MethodPost, requestURL, nil, req, result); err != nil {
		err = fmt.Errorf("failed to get messages: %w", err)
		return
	}
	return
}

// https://docs.honcho.dev/v3/api-reference/endpoint/messages/update-message
func (c *Client) UpdateMessage(ctx context.Context, workspaceID, sessionID, messageID string, req MessageUpdate) (result *Message, err error) {
	// Construct URL
	requestURL, err := c.endpoint(workspaceBaseURI, workspaceID, "sessions", sessionID, "messages", messageID)
	if err != nil {
		return
	}
	// Initialize result
	result = new(Message)
	// Make request
	if _, err = c.request(ctx, http.MethodPut, requestURL, nil, req, result); err != nil {
		err = fmt.Errorf("failed to update message: %w", err)
		return
	}
	return
}
