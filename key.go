package honcho

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

const (
	keyBaseURI = "/v3/keys"
)

// CreateKey creates a new scoped API key.
//
// Keys can be scoped to a workspace, peer, or session for fine-grained access control.
// An optional expiration time can also be set.
//
// It requires an admin key, and returns only the key value: Honcho does not send
// back the key's scopes, expiration or creation time.
//
// https://docs.honcho.dev/v3/api-reference/endpoint/keys/create-key
func (c *Client) CreateKey(ctx context.Context, req CreateKeyRequest) (result *Key, err error) {
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
