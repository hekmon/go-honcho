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

// ErrInvalidID is the error, wrapped, of a method given an ID that cannot be a
// segment of a request's path: an empty ID, "." or "..", or an ID holding a "/".
// No request is sent for such an ID. Test for it with errors.Is.
var ErrInvalidID = errors.New("invalid ID")

// streamHandler is a result that reads the response body as it arrives: request
// asks for an event stream and calls the handler once the status is handled,
// returning its error. As request drains the body afterwards, a handler leaving
// a stream the server may keep open must end the request's context first. Its
// connection is not reused: a server failing a stream closes it after answering,
// and the transport does not retry a POST sent on a connection being closed.
type streamHandler func(io.Reader) error

// endpoint returns the URL of base, one of this package's path constants, followed
// by segments, each becoming exactly one segment of the path. Every URL must be built
// here: a caller-supplied ID must not be able to point a request at another resource.
//
// An empty segment, "." and ".." are refused with an error wrapping ErrInvalidID, as
// joining them would clean them away and leave the request on a parent path. So is a
// segment holding a "/", which escaping would not make safe: Honcho decodes %2F before
// routing, so the session ID "victim/peers", once escaped, would still reach the peers
// route of the session victim. Any other character is escaped with url.PathEscape, and
// reaches the server as part of a literal ID.
func (c *Client) endpoint(base string, segments ...string) (*url.URL, error) {
	elements := make([]string, 0, len(segments)+1)
	elements = append(elements, base)
	for _, segment := range segments {
		if segment == "" || segment == "." || segment == ".." || strings.Contains(segment, "/") {
			return nil, fmt.Errorf("%w: %q", ErrInvalidID, segment)
		}
		elements = append(elements, url.PathEscape(segment))
	}
	return c.baseURL.JoinPath(elements...), nil
}

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
				err = fmt.Errorf("failed to encode body: %w", err)
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
		err = fmt.Errorf("failed to build request: %w", err)
		return
	}
	// Set headers
	if headers != nil {
		req.Header = headers
	}
	switch result.(type) {
	case nil, *bytes.Buffer:
		// do not set a particular accept
	case streamHandler:
		req.Header.Set("Accept", "text/event-stream")
		req.Close = true
	default:
		req.Header.Set("Accept", "application/json")
	}
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
	// Execute request
	resp, err := c.http.Do(req)
	if err != nil {
		// Returned as it is: a *url.Error names the method and the URL, and every
		// caller adds its own context.
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
		// Decode validation error: the body is read once, then decoded from
		// what was read.
		var data []byte
		if data, err = io.ReadAll(resp.Body); err != nil {
			err = fmt.Errorf("failed to read response body: %w", err)
			return
		}
		var valErr HTTPValidationError
		if err = json.Unmarshal(data, &valErr); err != nil {
			err = fmt.Errorf("failed to decode validation error %q: %w", data, err)
			return
		}
		err = &valErr
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
	// Hand a stream over to its handler
	if handler, ok := result.(streamHandler); ok {
		err = handler(resp.Body)
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
			err = fmt.Errorf("failed to decode response: %w", err)
			return
		}
		return
	}
	switch typedResult := result.(type) {
	case *bytes.Buffer:
		if _, err = io.Copy(typedResult, resp.Body); err != nil {
			err = fmt.Errorf("failed to copy response to bytes buffer: %w", err)
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

// HTTPValidationError represents a 422 validation error response. Its detail
// takes one of two forms: FastAPI's request validation errors, a list, in
// Detail; or Honcho's own validation errors, a text, in Message.
type HTTPValidationError struct {
	Detail  []ValidationError
	Message string
}

// UnmarshalJSON decodes a 422 response's body, whatever the form of its
// detail.
func (e *HTTPValidationError) UnmarshalJSON(data []byte) error {
	var body struct {
		Detail json.RawMessage `json:"detail"`
	}
	if err := json.Unmarshal(data, &body); err != nil {
		return err
	}
	if len(body.Detail) > 0 && body.Detail[0] == '"' {
		return json.Unmarshal(body.Detail, &e.Message)
	}
	return json.Unmarshal(body.Detail, &e.Detail)
}

// Error implements the error interface for HTTPValidationError
func (e *HTTPValidationError) Error() string {
	if e.Message != "" {
		return "validation error: " + e.Message
	}
	errs := make([]string, len(e.Detail))
	for i, d := range e.Detail {
		loc := make([]string, len(d.Loc))
		for j, l := range d.Loc {
			loc[j] = fmt.Sprint(l)
		}
		errs[i] = strings.Join(loc, ".") + ": " + d.Msg
	}
	return "validation error: " + strings.Join(errs, "; ")
}

// ValidationError represents a single validation error
type ValidationError struct {
	Loc   []any  `json:"loc"`
	Msg   string `json:"msg"`
	Type  string `json:"type"`
	Input any    `json:"input,omitempty"`
	Ctx   any    `json:"ctx,omitempty"`
}
