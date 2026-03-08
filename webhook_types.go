package honcho

import (
	"errors"
	"time"
)

// WebhookEndpoint represents a webhook endpoint configuration.
type WebhookEndpoint struct {
	// ID is the unique identifier of the webhook endpoint.
	ID string `json:"id"`
	// WorkspaceID is the ID of the workspace the endpoint belongs to.
	WorkspaceID string `json:"workspace_id"`
	// URL is the webhook endpoint URL.
	URL string `json:"url"`
	// CreatedAt is the timestamp when the endpoint was created.
	CreatedAt time.Time `json:"created_at"`
}

// WebhookEndpointCreate represents the request to create a webhook endpoint.
type WebhookEndpointCreate struct {
	// URL is the webhook endpoint URL (required).
	URL string `json:"url"`
}

// Validate validates the required fields for creating a webhook endpoint.
func (req WebhookEndpointCreate) Validate() error {
	if req.URL == "" {
		return errors.New("url is required")
	}
	return nil
}

// PageWebhookEndpoint represents a paginated list of webhook endpoints.
type PageWebhookEndpoint struct {
	// Items is the list of webhook endpoints on this page.
	Items []*WebhookEndpoint `json:"items"`
	// Total is the total number of webhook endpoints.
	Total int `json:"total"`
	// Page is the current page number (minimum: 1).
	Page int `json:"page"`
	// Size is the page size (minimum: 1, maximum: 100).
	Size int `json:"size"`
	// Pages is the total number of pages.
	Pages int `json:"pages"`
}

// ListWebhookEndpointsOptions represents optional query parameters for listing webhook endpoints.
type ListWebhookEndpointsOptions struct {
	// Page is the page number (default: 1, minimum: 1).
	Page int `json:"page,omitempty"`
	// Size is the page size (default: 50, minimum: 1, maximum: 100).
	Size int `json:"size,omitempty"`
}

// Validate validates the list options parameters.
func (opts ListWebhookEndpointsOptions) Validate() error {
	// Allow 0 to mean "use server default" (page: 1, size: 50)
	if opts.Page != 0 && opts.Page < 1 {
		return errors.New("page must be at least 1")
	}
	if opts.Size != 0 && (opts.Size < 1 || opts.Size > 100) {
		return errors.New("size must be between 1 and 100")
	}
	return nil
}
