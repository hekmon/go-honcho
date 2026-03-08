package honcho

import "time"

// Key represents an API key with scoped access.
type Key struct {
	// Key is the API key value.
	Key string `json:"key"`
	// WorkspaceID is the ID of the workspace the key is scoped to (if any).
	WorkspaceID string `json:"workspace_id,omitempty"`
	// PeerID is the ID of the peer the key is scoped to (if any).
	PeerID string `json:"peer_id,omitempty"`
	// SessionID is the ID of the session the key is scoped to (if any).
	SessionID string `json:"session_id,omitempty"`
	// ExpiresAt is the expiration time of the key (if set).
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	// CreatedAt is the timestamp when the key was created.
	CreatedAt time.Time `json:"created_at"`
}

// CreateKeyRequest represents the request to create a new API key.
//
// All fields are optional query parameters that scope the key's access.
type CreateKeyRequest struct {
	// WorkspaceID is the ID of the workspace to scope the key to (optional).
	WorkspaceID string `json:"workspace_id,omitempty"`
	// PeerID is the ID of the peer to scope the key to (optional).
	PeerID string `json:"peer_id,omitempty"`
	// SessionID is the ID of the session to scope the key to (optional).
	SessionID string `json:"session_id,omitempty"`
	// ExpiresAt is the expiration time for the key (optional).
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}
