package honcho

import "time"

// Key is an API key as Honcho returns it on creation: the key value alone,
// without its scopes, expiration or creation time.
type Key struct {
	// Key is the API key value.
	Key string `json:"key"`
}

// CreateKeyRequest represents the request to create a new API key.
//
// Its fields are query parameters. At least one scope is required among
// WorkspaceID, PeerID, SessionID and ScopeID. A peer or session scope also needs
// WorkspaceID, and a named scope needs WorkspaceID and excludes PeerID and
// SessionID. Honcho enforces these rules, answering 422 otherwise.
type CreateKeyRequest struct {
	// WorkspaceID is the ID of the workspace to scope the key to (optional).
	WorkspaceID string `json:"workspace_id,omitempty"`
	// PeerID is the ID of the peer to scope the key to (optional).
	PeerID string `json:"peer_id,omitempty"`
	// SessionID is the ID of the session to scope the key to (optional).
	SessionID string `json:"session_id,omitempty"`
	// ScopeID is the name of the scope for recall and ingestion (optional). It requires WorkspaceID.
	// Nil means the key is not tied to a named scope. A non-nil empty value is refused by CreateKey,
	// as leaving it out would mint a workspace-wide key instead of the restricted one asked for.
	ScopeID *string `json:"scope_id,omitempty"`
	// ExpiresAt is the expiration time for the key (optional).
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}
