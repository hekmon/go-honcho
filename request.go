package honcho

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

func (c *Client) request(ctx context.Context, method string, requestURL *url.URL, headers http.Header, body, result any) (responseHeaders http.Header, err error) {
	// Build request
	var (
		bodyReader io.Reader
		bodyBuffer bytes.Buffer
	)
	if body != nil {
		switch typed := body.(type) {
		case string:
			bodyBuffer.WriteString(typed)
			headers = setContentType(headers, "text/plain; charset=utf-8")
		case url.Values:
			bodyBuffer.WriteString(typed.Encode())
			headers = setContentType(headers, "application/x-www-form-urlencoded")
		case io.Reader:
			// io.Reader (e.g., *bytes.Buffer for multipart forms)
			// Headers must be set manually to include Content-Type with boundary
			bodyReader = typed
			if headers == nil || headers.Get("Content-Type") == "" {
				err = errors.New("headers must be set with Content-Type when body is io.Reader")
				return
			}
		default:
			if err = json.NewEncoder(&bodyBuffer).Encode(body); err != nil {
				err = fmt.Errorf("failed to encode body: %s", err)
				return
			}
			headers = setContentType(headers, "application/json; charset=utf-8")
		}
	}
	// Set body reader
	if bodyReader == nil {
		bodyReader = &bodyBuffer
	}
	req, err := http.NewRequestWithContext(ctx, method, requestURL.String(), bodyReader)
	if err != nil {
		err = fmt.Errorf("failed to build request: %s", err)
		return
	}
	// Set headers
	if headers != nil {
		req.Header = headers
	}
	switch result.(type) {
	case nil, *bytes.Buffer:
		// do not set a particular accept
	default:
		req.Header.Set("Accept", "application/json")
	}
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
	// Execute request
	resp, err := c.http.Do(req)
	if err != nil {
		err = fmt.Errorf("failed to execute request: %s", err)
		return
	}
	defer func() {
		_, _ = io.ReadAll(resp.Body)
		_ = resp.Body.Close()
	}()
	// Parse response
	responseHeaders = resp.Header
	switch resp.StatusCode {
	case http.StatusNoContent, http.StatusAccepted:
		return
	case http.StatusOK, http.StatusCreated:
		// continue
	case http.StatusUnprocessableEntity:
		// Decode validation error
		var data []byte
		data, err = io.ReadAll(resp.Body)
		if err != nil {
			err = fmt.Errorf("failed to read response body: %w", err)
			return
		}
		fmt.Println(string(data))
		var valErr HTTPValidationError
		if err = json.NewDecoder(resp.Body).Decode(&valErr); err != nil {
			err = fmt.Errorf("failed to decode validation error: %w", err)
			return
		}
		err = fmt.Errorf("validation error: %w", &valErr)
		return
	default:
		var builder strings.Builder
		// line 1
		builder.WriteString("Client request failed: ")
		builder.WriteString(resp.Status)
		builder.WriteRune('\n')
		// line 2
		builder.WriteString("URL: ")
		builder.WriteString(req.URL.String())
		builder.WriteRune('\n')
		// line 3
		data, _ := io.ReadAll(resp.Body)
		builder.Write(data)
		builder.WriteRune('\n')
		// return full error
		err = errors.New(builder.String())
		return
	}
	if result == nil {
		return
	}
	// Decode result body
	if strings.HasPrefix(resp.Header.Get("Content-Type"), "application/json") {
		var data []byte
		data, err = io.ReadAll(resp.Body)
		if err != nil {
			err = fmt.Errorf("failed to read response body: %w", err)
			return
		}
		//fmt.Println(string(data))
		if err = json.Unmarshal(data, result); err != nil {
			err = fmt.Errorf("failed to decode response: %s", err)
			return
		}
		return
	}
	switch typedResult := result.(type) {
	case *bytes.Buffer:
		if _, err = io.Copy(typedResult, resp.Body); err != nil {
			err = fmt.Errorf("failed to copy response to bytes buffer: %s", err)
			return
		}
		return
	default:
		err = fmt.Errorf("unsupported result type: %T", result)
	}
	return
}

// setContentType sets the Content-Type header, preserving existing value if present
func setContentType(headers http.Header, contentType string) http.Header {
	if headers == nil {
		headers = make(http.Header)
	}
	if headers.Get("Content-Type") == "" {
		headers.Set("Content-Type", contentType)
	}
	return headers
}

// HTTPValidationError represents a 422 validation error response
type HTTPValidationError struct {
	Detail []ValidationError `json:"detail"`
}

// Error implements the error interface for HTTPValidationError
func (e *HTTPValidationError) Error() string {
	return fmt.Sprintf("validation error: %v", e.Detail)
}

// ValidationError represents a single validation error
type ValidationError struct {
	Loc   []any  `json:"loc"`
	Msg   string `json:"msg"`
	Type  string `json:"type"`
	Input any    `json:"input,omitempty"`
	Ctx   any    `json:"ctx,omitempty"`
}
