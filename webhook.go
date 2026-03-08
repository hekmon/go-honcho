package honcho

import (
	"errors"
	"fmt"
	"net/http"
)

// GetOrCreateWebhookEndpoint gets or creates a webhook endpoint URL.
//
// https://docs.honcho.dev/v3/api-reference/endpoint/webhooks/get-or-create-webhook-endpoint
func (c *Client) GetOrCreateWebhookEndpoint(workspaceID string, req WebhookEndpointCreate) (result *WebhookEndpoint, err error) {
	// Validate request
	if err = req.Validate(); err != nil {
		return
	}
	// Construct URL
	requestURL := c.baseURL.JoinPath(workspaceBaseURI, workspaceID, "webhooks")
	// Initialize result
	result = new(WebhookEndpoint)
	// Make request
	if _, err = c.request(http.MethodPost, requestURL, nil, req, result); err != nil {
		err = fmt.Errorf("failed to get or create webhook endpoint: %w", err)
		return
	}
	return
}

// ListWebhookEndpoints lists all webhook endpoints, optionally filtered by workspace.
//
// Results are paginated. Use opts to control pagination.
//
// https://docs.honcho.dev/v3/api-reference/endpoint/webhooks/list-webhook-endpoints
func (c *Client) ListWebhookEndpoints(workspaceID string, opts *ListWebhookEndpointsOptions) (result *PageWebhookEndpoint, err error) {
	// Validate options
	if opts != nil {
		if err = opts.Validate(); err != nil {
			return
		}
	}
	// Construct URL
	requestURL := c.baseURL.JoinPath(workspaceBaseURI, workspaceID, "webhooks")
	// Initialize result
	result = new(PageWebhookEndpoint)
	// Make request
	if _, err = c.request(http.MethodGet, requestURL, nil, opts, result); err != nil {
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
func (c *Client) DeleteWebhookEndpoint(workspaceID, endpointID string) (err error) {
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
	requestURL := c.baseURL.JoinPath(workspaceBaseURI, workspaceID, "webhooks", endpointID)
	// Make request (204 No Content expected)
	if _, err = c.request(http.MethodDelete, requestURL, nil, nil, nil); err != nil {
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
func (c *Client) TestEmit(workspaceID string) (err error) {
	// Validate required parameters
	if workspaceID == "" {
		err = errors.New("workspace_id is required")
		return
	}
	// Construct URL
	requestURL := c.baseURL.JoinPath(workspaceBaseURI, workspaceID, "webhooks", "test")
	// Make request
	if _, err = c.request(http.MethodGet, requestURL, nil, nil, nil); err != nil {
		err = fmt.Errorf("failed to test webhook emit: %w", err)
		return
	}
	return
}
