package honcho

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

func (c *Client) request(method string, requestURL *url.URL, headers http.Header, body, result any) (responseHeaders http.Header, err error) {
	// Build request
	var (
		contentType string
		bodyBuffer  bytes.Buffer
	)
	if body != nil {
		switch typed := body.(type) {
		case string:
			bodyBuffer.WriteString(typed)
			contentType = "text/plain; charset=utf-8"
		case url.Values:
			bodyBuffer.WriteString(typed.Encode())
			contentType = "application/x-www-form-urlencoded"
		default:
			if err = json.NewEncoder(&bodyBuffer).Encode(body); err != nil {
				err = fmt.Errorf("failed to encode body: %s", err)
				return
			}
			contentType = "application/json; charset=utf-8"
		}
	}
	req, err := http.NewRequest(method, requestURL.String(), &bodyBuffer)
	if err != nil {
		err = fmt.Errorf("failed to build request: %s", err)
		return
	}
	// Set headers
	if headers != nil {
		req.Header = headers
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
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
	case http.StatusNoContent:
		return
	case http.StatusOK:
		// continue
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
		if err = json.NewDecoder(resp.Body).Decode(result); err != nil {
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
