package honcho

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"unicode/utf8"
)

// CreateConclusions creates one or more Conclusions.
//
// Conclusions are logical certainties derived from interactions between Peers.
// They form the basis of a Peer's Representation.
//
// https://docs.honcho.dev/v3/api-reference/endpoint/conclusions/create-conclusions
func (c *Client) CreateConclusions(ctx context.Context, workspaceID string, req ConclusionBatchCreate) (result []Conclusion, err error) {
	if workspaceID == "" {
		err = errors.New("workspaceID is required")
		return
	}
	if err = req.Validate(); err != nil {
		return
	}
	requestURL, err := c.endpoint(workspaceBaseURI, workspaceID, "conclusions")
	if err != nil {
		return
	}
	if _, err = c.request(ctx, http.MethodPost, requestURL, nil, req, &result); err != nil {
		result = nil // a failed decode can leave items in it
		err = fmt.Errorf("failed to create conclusions: %w", err)
		return
	}
	return
}

// DeleteConclusion deletes a single Conclusion by ID.
//
// This action cannot be undone.
//
// https://docs.honcho.dev/v3/api-reference/endpoint/conclusions/delete-conclusion
func (c *Client) DeleteConclusion(ctx context.Context, workspaceID, conclusionID string) (err error) {
	if workspaceID == "" {
		err = errors.New("workspaceID is required")
		return
	}
	if conclusionID == "" {
		err = errors.New("conclusionID is required")
		return
	}
	requestURL, err := c.endpoint(workspaceBaseURI, workspaceID, "conclusions", conclusionID)
	if err != nil {
		return
	}
	if _, err = c.request(ctx, http.MethodDelete, requestURL, nil, nil, nil); err != nil {
		err = fmt.Errorf("failed to delete conclusion: %w", err)
		return
	}
	return
}

// GetConclusion gets a single Conclusion by ID.
//
// Honcho answers 404 for an unknown ID, an unknown workspace or a deleted conclusion.
//
// https://docs.honcho.dev/v3/api-reference/endpoint/conclusions/get-conclusion
func (c *Client) GetConclusion(ctx context.Context, workspaceID, conclusionID string) (result *Conclusion, err error) {
	if workspaceID == "" {
		err = errors.New("workspaceID is required")
		return
	}
	if conclusionID == "" {
		err = errors.New("conclusionID is required")
		return
	}
	requestURL, err := c.endpoint(workspaceBaseURI, workspaceID, "conclusions", conclusionID)
	if err != nil {
		return
	}
	result = new(Conclusion)
	if _, err = c.request(ctx, http.MethodGet, requestURL, nil, nil, &result); err != nil {
		err = fmt.Errorf("failed to get conclusion: %w", err)
		return
	}
	return
}

// ListConclusions lists Conclusions using optional filters, ordered by recency unless `reverse` is true.
//
// The list is paginated by page number, or by cursor when opts.Cursor is set.
//
// https://docs.honcho.dev/v3/api-reference/endpoint/conclusions/list-conclusions
func (c *Client) ListConclusions(ctx context.Context, workspaceID string, req *ConclusionGet, opts *ListConclusionsOptions) (result *PageConclusion, err error) {
	if workspaceID == "" {
		err = errors.New("workspaceID is required")
		return
	}
	requestURL, err := c.endpoint(workspaceBaseURI, workspaceID, "conclusions", "list")
	if err != nil {
		return
	}
	query := requestURL.Query()
	if opts != nil {
		if opts.Reverse {
			query.Set("reverse", "true")
		}
		if opts.Page > 0 {
			query.Set("page", strconv.Itoa(opts.Page))
		}
		if opts.Size > 0 {
			query.Set("size", strconv.Itoa(opts.Size))
		}
		if err = setCursor(query, opts.Page, opts.Cursor); err != nil {
			return
		}
	}
	requestURL.RawQuery = query.Encode()
	result = new(PageConclusion)
	if _, err = c.request(ctx, http.MethodPost, requestURL, nil, req, result); err != nil {
		err = fmt.Errorf("failed to list conclusions: %w", err)
		return
	}
	return
}

// QueryConclusions queries Conclusions using semantic search.
//
// Use `top_k` to control the number of results returned.
//
// `filters` is required and must name the observer, under `observer` or `observer_id`,
// and the observed peer, under `observed` or `observed_id`.
//
// https://docs.honcho.dev/v3/api-reference/endpoint/conclusions/query-conclusions
func (c *Client) QueryConclusions(ctx context.Context, workspaceID string, req ConclusionQuery) (result []Conclusion, err error) {
	if workspaceID == "" {
		err = errors.New("workspaceID is required")
		return
	}
	if err = req.Validate(); err != nil {
		return
	}
	requestURL, err := c.endpoint(workspaceBaseURI, workspaceID, "conclusions", "query")
	if err != nil {
		return
	}
	if _, err = c.request(ctx, http.MethodPost, requestURL, nil, req, &result); err != nil {
		result = nil // a failed decode can leave items in it
		err = fmt.Errorf("failed to query conclusions: %w", err)
		return
	}
	return
}

// Validate validates the ConclusionBatchCreate request.
func (req ConclusionBatchCreate) Validate() error {
	if len(req.Conclusions) == 0 {
		return errors.New("conclusions is required")
	}
	if len(req.Conclusions) > 100 {
		return errors.New("conclusions must be 100 items or less")
	}
	for i, conclusion := range req.Conclusions {
		if err := conclusion.Validate(); err != nil {
			return fmt.Errorf("conclusion %d: %w", i+1, err)
		}
	}
	return nil
}

// Validate validates the ConclusionCreate request.
func (req ConclusionCreate) Validate() error {
	if req.Content == "" {
		return errors.New("content is required")
	}
	if utf8.RuneCountInString(req.Content) > 65535 {
		return errors.New("content must be 65535 characters or less")
	}
	if req.ObserverID == "" {
		return errors.New("observer_id is required")
	}
	if req.ObservedID == "" {
		return errors.New("observed_id is required")
	}
	return nil
}

// Validate validates the ConclusionQuery request.
func (req ConclusionQuery) Validate() error {
	if req.Query == "" {
		return errors.New("query is required")
	}
	if !namesPeer(req.Filters, "observer", "observer_id") || !namesPeer(req.Filters, "observed", "observed_id") {
		return errors.New("filters must name the observer (observer or observer_id) and the observed peer (observed or observed_id)")
	}
	if req.TopK != 0 && (req.TopK < 1 || req.TopK > 100) {
		return errors.New("top_k must be between 1 and 100")
	}
	if req.Distance != nil && (*req.Distance < 0.0 || *req.Distance > 1.0) {
		return errors.New("distance must be between 0 and 1")
	}
	return nil
}

// namesPeer reports whether filters holds a value under one of keys. A missing, nil
// or empty-string value does not count.
func namesPeer(filters map[string]any, keys ...string) bool {
	for _, key := range keys {
		if value := filters[key]; value != nil && value != "" {
			return true
		}
	}
	return false
}
