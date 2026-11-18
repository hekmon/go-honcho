package honcho

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"
)

const (
	keyBaseURI = "/v3/keys"
)

// CreateKey creates a new scoped API key.
//
// Keys can be scoped to a workspace, peer, session, or named scope for fine-grained access control.
// An optional expiration time can also be set.
//
// It requires an admin key, and returns only the key value: Honcho does not send
// back the key's scopes, expiration or creation time.
//
// A non-nil empty ScopeID is refused before anything is sent.
//
// https://docs.honcho.dev/v3/api-reference/endpoint/keys/create-key
func (c *Client) CreateKey(ctx context.Context, req CreateKeyRequest) (result *Key, err error) {
	// Leaving an empty scope name out of the request would mint a workspace-wide key instead of the
	// restricted one the caller asked for.
	if req.ScopeID != nil && *req.ScopeID == "" {
		err = errors.New("scope_id must not be empty: leave it nil for a key not tied to a named scope")
		return
	}
	requestURL, err := c.endpoint(keyBaseURI)
	if err != nil {
		return
	}
	query := requestURL.Query()
	if req.WorkspaceID != "" {
		query.Set("workspace_id", req.WorkspaceID)
	}
	if req.PeerID != "" {
		query.Set("peer_id", req.PeerID)
	}
	if req.SessionID != "" {
		query.Set("session_id", req.SessionID)
	}
	if req.ScopeID != nil {
		query.Set("scope_id", *req.ScopeID)
	}
	if req.ExpiresAt != nil {
		query.Set("expires_at", req.ExpiresAt.Format(time.RFC3339))
	}
	requestURL.RawQuery = query.Encode()
	result = new(Key)
	if _, err = c.request(ctx, http.MethodPost, requestURL, nil, nil, result); err != nil {
		err = fmt.Errorf("failed to create key: %w", err)
		return
	}
	return
}
