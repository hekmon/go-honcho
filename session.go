package honcho

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
)

// GetOrCreateSession gets a Session by ID or creates a new Session with the given ID.
//
// If Session ID is provided as a parameter, it verifies the Session is in the Workspace.
// Otherwise, it uses the session_id from the JWT for verification.
//
// https://docs.honcho.dev/v3/api-reference/endpoint/sessions/get-or-create-session
func (c *Client) GetOrCreateSession(workspaceID string, req SessionCreate) (result *Session, err error) {
	if workspaceID == "" {
		err = errors.New("workspaceID is required")
		return
	}
	if err = req.Validate(); err != nil {
		return
	}
	result = new(Session)
	if _, err = c.request(
		http.MethodPost, c.baseURL.JoinPath(workspaceBaseURI, workspaceID, "sessions"), nil,
		req, &result,
	); err != nil {
		err = fmt.Errorf("failed to execute request: %w", err)
		return
	}
	return
}

// GetSessions gets all Sessions for a Workspace, paginated with optional filters.
//
// https://docs.honcho.dev/v3/api-reference/endpoint/sessions/get-sessions
func (c *Client) GetSessions(workspaceID string, req *SessionGet, opts *GetSessionsOptions) (result *PageSession, err error) {
	if workspaceID == "" {
		err = errors.New("workspaceID is required")
		return
	}
	requestURL := c.baseURL.JoinPath(workspaceBaseURI, workspaceID, "sessions", "list")
	if opts != nil {
		query := requestURL.Query()
		if opts.Page > 0 {
			query.Set("page", strconv.Itoa(opts.Page))
		}
		if opts.Size > 0 {
			query.Set("size", strconv.Itoa(opts.Size))
		}
		requestURL.RawQuery = query.Encode()
	}
	result = new(PageSession)
	if _, err = c.request(
		http.MethodPost, requestURL, nil,
		req, &result,
	); err != nil {
		err = fmt.Errorf("failed to execute request: %w", err)
		return
	}
	return
}

// UpdateSession updates a Session's metadata and/or configuration.
//
// https://docs.honcho.dev/v3/api-reference/endpoint/sessions/update-session
func (c *Client) UpdateSession(workspaceID, sessionID string, req SessionUpdate) (result *Session, err error) {
	if workspaceID == "" {
		err = errors.New("workspaceID is required")
		return
	}
	if sessionID == "" {
		err = errors.New("sessionID is required")
		return
	}
	result = new(Session)
	if _, err = c.request(
		http.MethodPut, c.baseURL.JoinPath(workspaceBaseURI, workspaceID, "sessions", sessionID), nil,
		req, &result,
	); err != nil {
		err = fmt.Errorf("failed to execute request: %w", err)
		return
	}
	return
}

// DeleteSession deletes a Session and all associated messages.
//
// The Session is marked as inactive immediately and returns 202 Accepted. The actual
// deletion of all related data happens asynchronously via the queue with retry support.
//
// This action cannot be undone.
//
// https://docs.honcho.dev/v3/api-reference/endpoint/sessions/delete-session
func (c *Client) DeleteSession(workspaceID, sessionID string) (err error) {
	if workspaceID == "" {
		err = errors.New("workspaceID is required")
		return
	}
	if sessionID == "" {
		err = errors.New("sessionID is required")
		return
	}
	if _, err = c.request(
		http.MethodDelete, c.baseURL.JoinPath(workspaceBaseURI, workspaceID, "sessions", sessionID), nil,
		nil, nil,
	); err != nil {
		err = fmt.Errorf("failed to execute request: %w", err)
		return
	}
	return
}

// CloneSession clones a Session, optionally up to a specific message ID.
//
// https://docs.honcho.dev/v3/api-reference/endpoint/sessions/clone-session
func (c *Client) CloneSession(workspaceID, sessionID string, opts *CloneSessionOptions) (result *Session, err error) {
	if workspaceID == "" {
		err = errors.New("workspaceID is required")
		return
	}
	if sessionID == "" {
		err = errors.New("sessionID is required")
		return
	}
	requestURL := c.baseURL.JoinPath(workspaceBaseURI, workspaceID, "sessions", sessionID, "clone")
	if opts != nil && opts.MessageID != nil {
		query := requestURL.Query()
		query.Set("message_id", *opts.MessageID)
		requestURL.RawQuery = query.Encode()
	}
	result = new(Session)
	if _, err = c.request(
		http.MethodPost, requestURL, nil,
		nil, &result,
	); err != nil {
		err = fmt.Errorf("failed to execute request: %w", err)
		return
	}
	return
}

// GetSessionPeers gets all Peers in a Session. Results are paginated.
//
// https://docs.honcho.dev/v3/api-reference/endpoint/sessions/get-session-peers
func (c *Client) GetSessionPeers(workspaceID, sessionID string, opts *GetSessionPeersOptions) (result *PagePeer, err error) {
	if workspaceID == "" {
		err = errors.New("workspaceID is required")
		return
	}
	if sessionID == "" {
		err = errors.New("sessionID is required")
		return
	}
	requestURL := c.baseURL.JoinPath(workspaceBaseURI, workspaceID, "sessions", sessionID, "peers")
	if opts != nil {
		query := requestURL.Query()
		if opts.Page > 0 {
			query.Set("page", strconv.Itoa(opts.Page))
		}
		if opts.Size > 0 {
			query.Set("size", strconv.Itoa(opts.Size))
		}
		requestURL.RawQuery = query.Encode()
	}
	result = new(PagePeer)
	if _, err = c.request(
		http.MethodGet, requestURL, nil,
		nil, &result,
	); err != nil {
		err = fmt.Errorf("failed to execute request: %w", err)
		return
	}
	return
}

// SetSessionPeers sets the Peers in a Session. If a Peer does not yet exist, it will be created automatically.
//
// This will fully replace the current set of Peers in the Session.
//
// https://docs.honcho.dev/v3/api-reference/endpoint/sessions/set-session-peers
func (c *Client) SetSessionPeers(workspaceID, sessionID string, peers map[string]*SessionPeerConfig) (result *Session, err error) {
	if workspaceID == "" {
		err = errors.New("workspaceID is required")
		return
	}
	if sessionID == "" {
		err = errors.New("sessionID is required")
		return
	}
	result = new(Session)
	if _, err = c.request(
		http.MethodPut, c.baseURL.JoinPath(workspaceBaseURI, workspaceID, "sessions", sessionID, "peers"), nil,
		peers, &result,
	); err != nil {
		err = fmt.Errorf("failed to execute request: %w", err)
		return
	}
	return
}

// AddPeersToSession adds Peers to a Session. If a Peer does not yet exist, it will be created automatically.
//
// https://docs.honcho.dev/v3/api-reference/endpoint/sessions/add-peers-to-session
func (c *Client) AddPeersToSession(workspaceID, sessionID string, peers map[string]*SessionPeerConfig) (result *Session, err error) {
	if workspaceID == "" {
		err = errors.New("workspaceID is required")
		return
	}
	if sessionID == "" {
		err = errors.New("sessionID is required")
		return
	}
	result = new(Session)
	if _, err = c.request(
		http.MethodPost, c.baseURL.JoinPath(workspaceBaseURI, workspaceID, "sessions", sessionID, "peers"), nil,
		peers, &result,
	); err != nil {
		err = fmt.Errorf("failed to execute request: %w", err)
		return
	}
	return
}

// RemovePeersFromSession removes Peers by ID from a Session.
//
// https://docs.honcho.dev/v3/api-reference/endpoint/sessions/remove-peers-from-session
func (c *Client) RemovePeersFromSession(workspaceID, sessionID string, peerIDs []string) (result *Session, err error) {
	if workspaceID == "" {
		err = errors.New("workspaceID is required")
		return
	}
	if sessionID == "" {
		err = errors.New("sessionID is required")
		return
	}
	if len(peerIDs) == 0 {
		err = errors.New("peerIDs is required")
		return
	}
	result = new(Session)
	if _, err = c.request(
		http.MethodDelete, c.baseURL.JoinPath(workspaceBaseURI, workspaceID, "sessions", sessionID, "peers"), nil,
		peerIDs, &result,
	); err != nil {
		err = fmt.Errorf("failed to execute request: %w", err)
		return
	}
	return
}

// GetSessionContext produces a context object from the Session. The caller provides an optional token limit
// which the entire context must fit into. If not provided, the context will be exhaustive (within configured max tokens).
//
// To do this, we allocate 40% of the token limit to the summary, and 60% to recent messages -- as many as can fit.
// Note that the summary will usually take up less space than this. If the caller does not want a summary,
// we allocate all the tokens to recent messages.
//
// https://docs.honcho.dev/v3/api-reference/endpoint/sessions/get-session-context
func (c *Client) GetSessionContext(workspaceID, sessionID string, opts *GetSessionContextOptions) (result *SessionContext, err error) {
	if workspaceID == "" {
		err = errors.New("workspaceID is required")
		return
	}
	if sessionID == "" {
		err = errors.New("sessionID is required")
		return
	}
	requestURL := c.baseURL.JoinPath(workspaceBaseURI, workspaceID, "sessions", sessionID, "context")
	if opts != nil {
		query := requestURL.Query()
		if opts.Tokens != nil {
			query.Set("tokens", strconv.Itoa(*opts.Tokens))
		}
		if opts.SearchQuery != nil {
			query.Set("search_query", *opts.SearchQuery)
		}
		if opts.Summary != nil {
			query.Set("summary", strconv.FormatBool(*opts.Summary))
		}
		if opts.PeerTarget != nil {
			query.Set("peer_target", *opts.PeerTarget)
		}
		if opts.PeerPerspective != nil {
			query.Set("peer_perspective", *opts.PeerPerspective)
		}
		if opts.LimitToSession != nil {
			query.Set("limit_to_session", strconv.FormatBool(*opts.LimitToSession))
		}
		if opts.SearchTopK != nil {
			query.Set("search_top_k", strconv.Itoa(*opts.SearchTopK))
		}
		if opts.SearchMaxDistance != nil {
			query.Set("search_max_distance", strconv.FormatFloat(*opts.SearchMaxDistance, 'f', -1, 64))
		}
		if opts.IncludeMostFrequent != nil {
			query.Set("include_most_frequent", strconv.FormatBool(*opts.IncludeMostFrequent))
		}
		if opts.MaxConclusions != nil {
			query.Set("max_conclusions", strconv.Itoa(*opts.MaxConclusions))
		}
		requestURL.RawQuery = query.Encode()
	}
	result = new(SessionContext)
	if _, err = c.request(
		http.MethodGet, requestURL, nil,
		nil, &result,
	); err != nil {
		err = fmt.Errorf("failed to execute request: %w", err)
		return
	}
	return
}

// GetSessionSummaries gets available summaries for a Session.
//
// Returns both short and long summaries if available, including metadata like
// the message ID they cover up to, creation timestamp, and token count.
//
// https://docs.honcho.dev/v3/api-reference/endpoint/sessions/get-session-summaries
func (c *Client) GetSessionSummaries(workspaceID, sessionID string) (result *SessionSummaries, err error) {
	if workspaceID == "" {
		err = errors.New("workspaceID is required")
		return
	}
	if sessionID == "" {
		err = errors.New("sessionID is required")
		return
	}
	result = new(SessionSummaries)
	if _, err = c.request(
		http.MethodGet, c.baseURL.JoinPath(workspaceBaseURI, workspaceID, "sessions", sessionID, "summaries"), nil,
		nil, &result,
	); err != nil {
		err = fmt.Errorf("failed to execute request: %w", err)
		return
	}
	return
}

// SearchSession searches a Session with optional filters.
//
// Use limit to control the number of results returned.
//
// https://docs.honcho.dev/v3/api-reference/endpoint/sessions/search-session
func (c *Client) SearchSession(workspaceID, sessionID string, req MessageSearchOptions) (result *[]Message, err error) {
	if workspaceID == "" {
		err = errors.New("workspaceID is required")
		return
	}
	if sessionID == "" {
		err = errors.New("sessionID is required")
		return
	}
	if err = req.Validate(); err != nil {
		return
	}
	result = new([]Message)
	if _, err = c.request(
		http.MethodPost, c.baseURL.JoinPath(workspaceBaseURI, workspaceID, "sessions", sessionID, "search"), nil,
		req, &result,
	); err != nil {
		err = fmt.Errorf("failed to execute request: %w", err)
		return
	}
	return
}

// GetPeerConfig gets the configuration for a Peer in a Session.
//
// https://docs.honcho.dev/v3/api-reference/endpoint/sessions/get-peer-config
func (c *Client) GetPeerConfig(workspaceID, sessionID, peerID string) (result *SessionPeerConfig, err error) {
	if workspaceID == "" {
		err = errors.New("workspaceID is required")
		return
	}
	if sessionID == "" {
		err = errors.New("sessionID is required")
		return
	}
	if peerID == "" {
		err = errors.New("peerID is required")
		return
	}
	result = new(SessionPeerConfig)
	if _, err = c.request(
		http.MethodGet, c.baseURL.JoinPath(workspaceBaseURI, workspaceID, "sessions", sessionID, "peers", peerID, "config"), nil,
		nil, &result,
	); err != nil {
		err = fmt.Errorf("failed to execute request: %w", err)
		return
	}
	return
}

// SetPeerConfig sets the configuration for a Peer in a Session.
//
// https://docs.honcho.dev/v3/api-reference/endpoint/sessions/set-peer-config
func (c *Client) SetPeerConfig(workspaceID, sessionID, peerID string, config SessionPeerConfig) (err error) {
	if workspaceID == "" {
		err = errors.New("workspaceID is required")
		return
	}
	if sessionID == "" {
		err = errors.New("sessionID is required")
		return
	}
	if peerID == "" {
		err = errors.New("peerID is required")
		return
	}
	if _, err = c.request(
		http.MethodPut, c.baseURL.JoinPath(workspaceBaseURI, workspaceID, "sessions", sessionID, "peers", peerID, "config"), nil,
		config, nil,
	); err != nil {
		err = fmt.Errorf("failed to execute request: %w", err)
		return
	}
	return
}
