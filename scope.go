package honcho

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
)

// GetOrCreateScope gets a Scope by ID or creates a new Scope with the given ID.
//
// A scope is a named group of sessions that acts as a visibility boundary on recall.
// Honcho answers 201 when the scope is created and 200 when it already exists.
//
// On a scope that already exists, metadata that differs from the stored metadata
// overwrites it. Leave Metadata nil to keep what is stored.
//
// A peer already occupying the scope's reserved internal name is never adopted: Honcho
// answers 409 for that conflict.
//
// https://docs.honcho.dev/v3/api-reference/endpoint/scopes/get-or-create-scope
func (c *Client) GetOrCreateScope(ctx context.Context, workspaceID string, req ScopeCreate) (result *Scope, err error) {
	if workspaceID == "" {
		err = errors.New("workspaceID is required")
		return
	}
	if err = req.Validate(); err != nil {
		return
	}
	requestURL, err := c.endpoint(workspaceBaseURI, workspaceID, "scopes")
	if err != nil {
		return
	}
	result = new(Scope)
	if _, err = c.request(ctx, http.MethodPost, requestURL, nil, req, result); err != nil {
		err = fmt.Errorf("failed to get or create scope: %w", err)
		return
	}
	return
}

// GetScopes gets all Scopes for a Workspace.
//
// Results are paginated, oldest scope first unless Reverse is set. Filter by name with
// `{"filters": {"id": {"icontains": "sup"}}}`: `id` is the unprefixed scope name. Set
// IncludeSessionCount to get each scope's member session count in SessionCount.
//
// The list is paginated by page number, or by cursor when opts.Cursor is set.
//
// https://docs.honcho.dev/v3/api-reference/endpoint/scopes/get-scopes
func (c *Client) GetScopes(ctx context.Context, workspaceID string, req *ScopeGet, opts *GetScopesOptions) (result *PageScope, err error) {
	if workspaceID == "" {
		err = errors.New("workspaceID is required")
		return
	}
	requestURL, err := c.endpoint(workspaceBaseURI, workspaceID, "scopes", "list")
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
		if opts.Reverse {
			query.Set("reverse", "true")
		}
		if opts.IncludeSessionCount {
			query.Set("include_session_count", "true")
		}
	}
	requestURL.RawQuery = query.Encode()
	result = new(PageScope)
	if _, err = c.request(ctx, http.MethodPost, requestURL, nil, req, result); err != nil {
		err = fmt.Errorf("failed to get scopes: %w", err)
		return
	}
	return
}

// GetScope gets a single Scope by ID.
//
// Unlike GetOrCreateScope, it never creates the scope: Honcho answers 404 for an unknown
// scope.
//
// https://docs.honcho.dev/v3/api-reference/endpoint/scopes/get-scope
func (c *Client) GetScope(ctx context.Context, workspaceID, scopeID string) (result *Scope, err error) {
	if workspaceID == "" {
		err = errors.New("workspaceID is required")
		return
	}
	if scopeID == "" {
		err = errors.New("scopeID is required")
		return
	}
	requestURL, err := c.endpoint(workspaceBaseURI, workspaceID, "scopes", scopeID)
	if err != nil {
		return
	}
	result = new(Scope)
	if _, err = c.request(ctx, http.MethodGet, requestURL, nil, nil, result); err != nil {
		err = fmt.Errorf("failed to get scope: %w", err)
		return
	}
	return
}

// GetScopeStatus gets the backfill job status of a Scope.
//
// Returns a per-session map of the backfill job state (pending, completed or failed) with
// the number of documents copied once complete. The map is empty when no backfill has ever
// been enqueued for the scope.
//
// Honcho answers 404 for an unknown scope.
//
// https://docs.honcho.dev/v3/api-reference/endpoint/scopes/get-scope-status
func (c *Client) GetScopeStatus(ctx context.Context, workspaceID, scopeID string) (result *ScopeStatus, err error) {
	if workspaceID == "" {
		err = errors.New("workspaceID is required")
		return
	}
	if scopeID == "" {
		err = errors.New("scopeID is required")
		return
	}
	requestURL, err := c.endpoint(workspaceBaseURI, workspaceID, "scopes", scopeID, "status")
	if err != nil {
		return
	}
	result = new(ScopeStatus)
	if _, err = c.request(ctx, http.MethodGet, requestURL, nil, nil, result); err != nil {
		err = fmt.Errorf("failed to get scope status: %w", err)
		return
	}
	return
}

// AddSessionsToScope adds Sessions to a Scope.
//
// All named sessions must already exist: Honcho answers 404 otherwise, as it does for an
// unknown scope. Adding a session that is already a member is a no-op.
//
// Any added session that already has messages starts an asynchronous backfill-by-copy of
// its existing documents into the scope: track its progress with GetScopeStatus.
//
// https://docs.honcho.dev/v3/api-reference/endpoint/scopes/add-sessions-to-scope
func (c *Client) AddSessionsToScope(ctx context.Context, workspaceID, scopeID string, req ScopeSessionsAdd) (err error) {
	if workspaceID == "" {
		err = errors.New("workspaceID is required")
		return
	}
	if scopeID == "" {
		err = errors.New("scopeID is required")
		return
	}
	if err = req.Validate(); err != nil {
		return
	}
	requestURL, err := c.endpoint(workspaceBaseURI, workspaceID, "scopes", scopeID, "sessions")
	if err != nil {
		return
	}
	if _, err = c.request(ctx, http.MethodPost, requestURL, nil, req, nil); err != nil {
		err = fmt.Errorf("failed to add sessions to scope: %w", err)
		return
	}
	return
}

// GetScopeSessions gets the Sessions that are members of a Scope.
//
// Results are paginated and ordered by how long each session has been a member: the
// longest-standing member first, or the most recently added first when Reverse is set.
//
// Honcho answers 404 for an unknown scope.
//
// The list is paginated by page number, or by cursor when opts.Cursor is set.
//
// https://docs.honcho.dev/v3/api-reference/endpoint/scopes/get-scope-sessions
func (c *Client) GetScopeSessions(ctx context.Context, workspaceID, scopeID string, opts *GetScopeSessionsOptions) (result *PageSession, err error) {
	if workspaceID == "" {
		err = errors.New("workspaceID is required")
		return
	}
	if scopeID == "" {
		err = errors.New("scopeID is required")
		return
	}
	requestURL, err := c.endpoint(workspaceBaseURI, workspaceID, "scopes", scopeID, "sessions", "list")
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
		if opts.Reverse {
			query.Set("reverse", "true")
		}
	}
	requestURL.RawQuery = query.Encode()
	result = new(PageSession)
	if _, err = c.request(ctx, http.MethodPost, requestURL, nil, nil, result); err != nil {
		err = fmt.Errorf("failed to get scope sessions: %w", err)
		return
	}
	return
}

// RemoveSessionsFromScope removes up to 100 Sessions from a Scope in one request.
//
// The batch is all-or-nothing: if the scope or any named session does not exist, Honcho
// answers 404 and no membership changes. Sessions that are not members of the scope are
// ignored. Each removed session is reconciled asynchronously, exactly as
// RemoveSessionFromScope does.
func (c *Client) RemoveSessionsFromScope(ctx context.Context, workspaceID, scopeID string, req ScopeSessionsRemove) (err error) {
	if workspaceID == "" {
		err = errors.New("workspaceID is required")
		return
	}
	if scopeID == "" {
		err = errors.New("scopeID is required")
		return
	}
	if err = req.Validate(); err != nil {
		return
	}
	requestURL, err := c.endpoint(workspaceBaseURI, workspaceID, "scopes", scopeID, "sessions", "remove")
	if err != nil {
		return
	}
	if _, err = c.request(ctx, http.MethodPost, requestURL, nil, req, nil); err != nil {
		err = fmt.Errorf("failed to remove sessions from scope: %w", err)
		return
	}
	return
}

// RemoveSessionFromScope removes a Session from a Scope.
//
// Documents copied or derived while the session was a member are reconciled asynchronously:
// the session's explicit copies are soft-deleted from the scope, dependent derived documents
// follow (fail-closed), and the scope's card is rebuilt from the remaining evidence.
//
// Removing a session that is not a member is a no-op. Honcho answers 404 for an unknown
// scope or session.
//
// https://docs.honcho.dev/v3/api-reference/endpoint/scopes/remove-session-from-scope
func (c *Client) RemoveSessionFromScope(ctx context.Context, workspaceID, scopeID, sessionID string) (err error) {
	if workspaceID == "" {
		err = errors.New("workspaceID is required")
		return
	}
	if scopeID == "" {
		err = errors.New("scopeID is required")
		return
	}
	if sessionID == "" {
		err = errors.New("sessionID is required")
		return
	}
	requestURL, err := c.endpoint(workspaceBaseURI, workspaceID, "scopes", scopeID, "sessions", sessionID)
	if err != nil {
		return
	}
	if _, err = c.request(ctx, http.MethodDelete, requestURL, nil, nil, nil); err != nil {
		err = fmt.Errorf("failed to remove session from scope: %w", err)
		return
	}
	return
}
