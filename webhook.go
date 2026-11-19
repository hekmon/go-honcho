package honcho

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
)

// GetOrCreateWebhookEndpoint gets or creates a webhook endpoint URL.
//
// https://docs.honcho.dev/v3/api-reference/endpoint/webhooks/get-or-create-webhook-endpoint
func (c *Client) GetOrCreateWebhookEndpoint(ctx context.Context, workspaceID string, req WebhookEndpointCreate) (result *WebhookEndpoint, err error) {
	// Validate request
	if err = req.Validate(); err != nil {
		return
	}
	// Construct URL
	requestURL, err := c.endpoint(workspaceBaseURI, workspaceID, "webhooks")
	if err != nil {
		return
	}
	// Initialize result
	result = new(WebhookEndpoint)
	// Make request
	if _, err = c.request(ctx, http.MethodPost, requestURL, nil, req, result); err != nil {
		err = fmt.Errorf("failed to get or create webhook endpoint: %w", err)
		return
	}
	return
}

// ListWebhookEndpoints lists all webhook endpoints, optionally filtered by workspace.
//
// The list is paginated by page number, or by cursor when opts.Cursor is set.
//
// https://docs.honcho.dev/v3/api-reference/endpoint/webhooks/list-webhook-endpoints
func (c *Client) ListWebhookEndpoints(ctx context.Context, workspaceID string, opts *ListWebhookEndpointsOptions) (result *PageWebhookEndpoint, err error) {
	// Validate options
	if opts != nil {
		if err = opts.Validate(); err != nil {
			return
		}
	}
	// Construct URL
	requestURL, err := c.endpoint(workspaceBaseURI, workspaceID, "webhooks")
	if err != nil {
		return
	}
	// Add query parameters
	if opts != nil {
		query := requestURL.Query()
		if opts.Page > 0 {
			query.Set("page", strconv.Itoa(opts.Page))
		}
		if opts.Size > 0 {
			query.Set("size", strconv.Itoa(opts.Size))
		}
		if err = setCursor(query, opts.Page, opts.Cursor); err != nil {
			return
		}
		requestURL.RawQuery = query.Encode()
	}
	// Initialize result
	result = new(PageWebhookEndpoint)
	// Make request
	if _, err = c.request(ctx, http.MethodGet, requestURL, nil, nil, result); err != nil {
		err = fmt.Errorf("failed to list webhook endpoints: %w", err)
		return
	}
	return
}

// DeleteWebhookEndpoint deletes a specific webhook endpoint.
//
// This action cannot be undone.
//
// https://docs.honcho.dev/v3/api-reference/endpoint/webhooks/delete-webhook-endpoint
func (c *Client) DeleteWebhookEndpoint(ctx context.Context, workspaceID, endpointID string) (err error) {
	// Validate required parameters
	if workspaceID == "" {
		err = errors.New("workspace_id is required")
		return
	}
	if endpointID == "" {
		err = errors.New("endpoint_id is required")
		return
	}
	// Construct URL
	requestURL, err := c.endpoint(workspaceBaseURI, workspaceID, "webhooks", endpointID)
	if err != nil {
		return
	}
	// Make request (204 No Content expected)
	if _, err = c.request(ctx, http.MethodDelete, requestURL, nil, nil, nil); err != nil {
		err = fmt.Errorf("failed to delete webhook endpoint: %w", err)
		return
	}
	return
}

// TestEmit tests publishing a webhook event.
//
// This endpoint triggers a test webhook event to verify the endpoint is configured correctly.
//
// https://docs.honcho.dev/v3/api-reference/endpoint/webhooks/test-emit
func (c *Client) TestEmit(ctx context.Context, workspaceID string) (err error) {
	// Validate required parameters
	if workspaceID == "" {
		err = errors.New("workspace_id is required")
		return
	}
	// Construct URL
	requestURL, err := c.endpoint(workspaceBaseURI, workspaceID, "webhooks", "test")
	if err != nil {
		return
	}
	// Make request
	if _, err = c.request(ctx, http.MethodGet, requestURL, nil, nil, nil); err != nil {
		err = fmt.Errorf("failed to test webhook emit: %w", err)
		return
	}
	return
}
