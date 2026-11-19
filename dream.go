package honcho

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
)

// ListDreams lists dream runs, newest first.
//
// Each run records what one dream changed in a representation.
// Filter by `observed_id` for a peer's history, and add `observer_id` for a single
// representation. Runs are not pruned unless the server sets a retention window.
//
// The runs can be narrowed only by observed and observer peer, as Honcho v3.6.1 has no
// filter object on this route.
//
// The list is paginated by page number, or by cursor when opts.Cursor is set.
//
// https://docs.honcho.dev/v3/api-reference/endpoint/dreams/list-dreams
func (c *Client) ListDreams(ctx context.Context, workspaceID string, req *DreamRunList, opts *ListDreamsOptions) (result *PageDreamRun, err error) {
	if workspaceID == "" {
		err = errors.New("workspaceID is required")
		return
	}
	requestURL, err := c.endpoint(workspaceBaseURI, workspaceID, "dreams", "list")
	if err != nil {
		return
	}
	query := requestURL.Query()
	if opts != nil {
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
	result = new(PageDreamRun)
	if _, err = c.request(ctx, http.MethodPost, requestURL, nil, req, result); err != nil {
		err = fmt.Errorf("failed to list dreams: %w", err)
		return
	}
	return
}

// GetDream gets one dream run with every change it committed.
//
// The response lists the conclusions the dream created with the conclusions each was
// derived from, the conclusions it removed as they read when removed, and the peer card
// before and after.
//
// Honcho answers 404 for an unknown run or a run in another workspace.
//
// https://docs.honcho.dev/v3/api-reference/endpoint/dreams/get-dream
func (c *Client) GetDream(ctx context.Context, workspaceID, runID string) (result *DreamRunDetail, err error) {
	if workspaceID == "" {
		err = errors.New("workspaceID is required")
		return
	}
	if runID == "" {
		err = errors.New("runID is required")
		return
	}
	requestURL, err := c.endpoint(workspaceBaseURI, workspaceID, "dreams", runID)
	if err != nil {
		return
	}
	result = new(DreamRunDetail)
	if _, err = c.request(ctx, http.MethodGet, requestURL, nil, nil, &result); err != nil {
		err = fmt.Errorf("failed to get dream: %w", err)
		return
	}
	return
}
