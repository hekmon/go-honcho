package honcho

import (
	"bytes"
	"fmt"
	"mime/multipart"
	"net/http"
	"strconv"
)

// https://docs.honcho.dev/v3/api-reference/endpoint/messages/create-messages-for-session
func (c *Client) CreateMessagesForSession(workspaceID, sessionID string, req MessageBatchCreate) (result []*Message, err error) {
	// Validate request
	if err = req.Validate(); err != nil {
		return
	}
	// Construct URL
	requestURL := c.baseURL.JoinPath(workspaceBaseURI, workspaceID, "sessions", sessionID, "messages")
	// Initialize result
	result = make([]*Message, 0, len(req.Messages))
	// Make request
	if _, err = c.request(http.MethodPost, requestURL, nil, req, &result); err != nil {
		err = fmt.Errorf("failed to create messages: %w", err)
		return
	}
	return
}

// https://docs.honcho.dev/v3/api-reference/endpoint/messages/create-messages-with-file
func (c *Client) CreateMessagesWithFile(workspaceID, sessionID string, req MessageUpload) (result []*Message, err error) {
	// Validate request
	if err = req.Validate(); err != nil {
		return
	}
	// Construct URL
	requestURL := c.baseURL.JoinPath(workspaceBaseURI, workspaceID, "sessions", sessionID, "messages", "upload")
	// Build multipart form
	bodyBuffer := &bytes.Buffer{}
	writer := multipart.NewWriter(bodyBuffer)
	// Add file field
	fileWriter, err := writer.CreateFormFile("file", req.Filename)
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
	// Add optional metadata field
	if req.Metadata != nil {
		if err = writer.WriteField("metadata", *req.Metadata); err != nil {
			err = fmt.Errorf("failed to write metadata field: %w", err)
			return
		}
	}
	// Add optional configuration field
	if req.Configuration != nil {
		if err = writer.WriteField("configuration", *req.Configuration); err != nil {
			err = fmt.Errorf("failed to write configuration field: %w", err)
			return
		}
	}
	// Add optional created_at field
	if req.CreatedAt != nil {
		if err = writer.WriteField("created_at", *req.CreatedAt); err != nil {
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
	// Make request with string body (multipart data)
	if _, err = c.request(http.MethodPost, requestURL, headers, bodyBuffer.String(), &result); err != nil {
		err = fmt.Errorf("failed to upload file and create messages: %w", err)
		return
	}
	return
}

// https://docs.honcho.dev/v3/api-reference/endpoint/messages/get-message
func (c *Client) GetMessage(workspaceID, sessionID, messageID string) (result *Message, err error) {
	// Construct URL
	requestURL := c.baseURL.JoinPath(workspaceBaseURI, workspaceID, "sessions", sessionID, "messages", messageID)
	// Initialize result
	result = new(Message)
	// Make request
	if _, err = c.request(http.MethodGet, requestURL, nil, nil, result); err != nil {
		err = fmt.Errorf("failed to get message: %w", err)
		return
	}
	return
}

// https://docs.honcho.dev/v3/api-reference/endpoint/messages/get-messages
func (c *Client) GetMessages(workspaceID, sessionID string, options *GetMessagesOptions, req *MessageGet) (result *PageMessage, err error) {
	// Construct URL
	requestURL := c.baseURL.JoinPath(workspaceBaseURI, workspaceID, "sessions", sessionID, "messages", "list")
	// Add query parameters
	queryParams := requestURL.Query()
	if options != nil {
		if options.Reverse != nil {
			queryParams.Set("reverse", strconv.FormatBool(*options.Reverse))
		}
		if options.Page > 0 {
			queryParams.Set("page", strconv.Itoa(options.Page))
		}
		if options.Size > 0 {
			queryParams.Set("size", strconv.Itoa(options.Size))
		}
	}
	requestURL.RawQuery = queryParams.Encode()
	// Initialize result
	result = new(PageMessage)
	// Make request
	if _, err = c.request(http.MethodPost, requestURL, nil, req, result); err != nil {
		err = fmt.Errorf("failed to get messages: %w", err)
		return
	}
	return
}

// https://docs.honcho.dev/v3/api-reference/endpoint/messages/update-message
func (c *Client) UpdateMessage(workspaceID, sessionID, messageID string, req MessageUpdate) (result *Message, err error) {
	// Validate request
	if err = req.Validate(); err != nil {
		return
	}
	// Construct URL
	requestURL := c.baseURL.JoinPath(workspaceBaseURI, workspaceID, "sessions", sessionID, "messages", messageID)
	// Initialize result
	result = new(Message)
	// Make request
	if _, err = c.request(http.MethodPut, requestURL, nil, req, result); err != nil {
		err = fmt.Errorf("failed to update message: %w", err)
		return
	}
	return
}
