package honcho

import (
	"fmt"
	"net/http"
	"time"
)

const (
	keyBaseURI = "/v3/keys"
)

// CreateKey creates a new API key with optional scoping.
//
// Keys can be scoped to a workspace, peer, or session for fine-grained access control.
// An optional expiration time can also be set.
//
// https://docs.honcho.dev/v3/api-reference/endpoint/keys/create-key
func (c *Client) CreateKey(req CreateKeyRequest) (result *Key, err error) {
	requestURL := c.baseURL.JoinPath(keyBaseURI)
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
	if _, err = c.request(http.MethodPost, requestURL, nil, nil, result); err != nil {
		err = fmt.Errorf("failed to create key: %w", err)
		return
	}
	return
}
