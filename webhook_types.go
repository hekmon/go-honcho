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
	Items []WebhookEndpoint `json:"items"`
	// Total is the total number of webhook endpoints. It is zero on a cursor page.
	Total int `json:"total"`
	// Page is the current page number (minimum: 1). It is zero on a cursor page.
	Page int `json:"page"`
	// Size is the page size (minimum: 1, maximum: 100). It is zero on a cursor page.
	Size int `json:"size"`
	// Pages is the total number of pages. It is zero on a cursor page.
	Pages int `json:"pages"`
	// CurrentPage is the cursor to refetch the current page. It is set only on a cursor page.
	CurrentPage *string `json:"current_page,omitempty"`
	// CurrentPageBackwards is the cursor to refetch the current page starting from the last item.
	// It is set only on a cursor page.
	CurrentPageBackwards *string `json:"current_page_backwards,omitempty"`
	// PreviousPage is the cursor for the previous page. It is set only on a cursor page, and is nil
	// on the first page.
	PreviousPage *string `json:"previous_page,omitempty"`
	// NextPage is the cursor for the next page, to set as Cursor for the next request. It is set
	// only on a cursor page, and is nil on the last page.
	NextPage *string `json:"next_page,omitempty"`
}

// ListWebhookEndpointsOptions represents optional query parameters for listing webhook endpoints.
type ListWebhookEndpointsOptions struct {
	// Page is the page number (default: 1, minimum: 1).
	Page int `json:"page,omitempty"`
	// Size is the page size (default: 50, minimum: 1, maximum: 100).
	Size int `json:"size,omitempty"`
	// Cursor opts in to cursor pagination when not nil (default: nil, for page numbers). An empty
	// cursor, new(string), asks for the first page: then set Cursor to each page's NextPage until
	// that is nil. A cursor page has no total and no page numbers. A cursor holds only a position
	// and is only valid for the list that returned it, so keep the other options and any filters
	// unchanged while following cursors. Page cannot be combined with Cursor: the request is
	// refused before it is sent, as the server would ignore the page.
	Cursor *string `json:"cursor,omitempty"`
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
	// The same refusal, with the same error, as every list method gives through setCursor.
	if opts.Page > 0 && opts.Cursor != nil {
		return errPageWithCursor
	}
	return nil
}
