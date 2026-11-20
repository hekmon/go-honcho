package honcho

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
)

// GetOrCreateSession gets a Session by ID or creates a new Session with the given ID, creating
// the workspace too if it does not exist.
//
// On a session that already exists, each field that is sent changes it, and each field left out
// keeps what is stored:
//   - Metadata replaces the stored metadata: an empty object, {}, erases it.
//   - Configuration is merged per section, as UpdateSession does: a sent section replaces the
//     stored section of that name, so a field left nil inside it is erased. Sections not sent are
//     kept, and an empty configuration changes nothing.
//   - Peers makes each listed peer a member, creating it if missing, and members not listed stay
//     members. A peer already a member keeps its stored configuration and the sent one is ignored,
//     as with AddPeersToSession. A peer that had left rejoins with the sent configuration.
//   - Scopes adds the session to the listed scopes, as for a new session.
//
// After DeleteSession, Honcho answers 404 for the session ID until its queue has run the deletion.
// GetOrCreateSession then creates a new, empty session with that ID. With Honcho's default
// settings, the deriver backs off to 30 seconds between polls on an idle queue, so the deletion
// can take tens of seconds to run. Without a deriver, the deletion never runs. As
// GetOrCreateSession creates a missing workspace or session, its 404 means a pending deletion,
// except for a scope-scoped key whose scope does not exist. No other route shows a pending
// deletion: GetSessions leaves out inactive sessions, GetQueueStatus does not count deletions,
// and no route reads a single session.
//
// A session-scoped key reaches only its own session: any other ID gets a 401. Peer- and
// scope-scoped keys can create a new session. On an existing session, a peer-scoped key's peer
// must already be a member, even if the key lists it in Peers, and a scope-scoped key's scope
// must hold the session, or they get a 401; both get one for sending Metadata or Configuration,
// and a scope-scoped key also for Peers. A peer-scoped key may name only its own peer in Peers,
// or it gets a 401, and Scopes requires a workspace- or admin-level key, except that a
// scope-scoped key may name its own scope.
//
// https://docs.honcho.dev/v3/api-reference/endpoint/sessions/get-or-create-session
func (c *Client) GetOrCreateSession(ctx context.Context, workspaceID string, req SessionCreate) (result *Session, err error) {
	if workspaceID == "" {
		err = errors.New("workspaceID is required")
		return
	}
	if err = req.Validate(); err != nil {
		return
	}
	requestURL, err := c.endpoint(workspaceBaseURI, workspaceID, "sessions")
	if err != nil {
		return
	}
	result = new(Session)
	if _, err = c.request(
		ctx, http.MethodPost, requestURL, nil,
		req, &result,
	); err != nil {
		err = fmt.Errorf("failed to execute request: %w", err)
		return
	}
	return
}

// GetSessions gets all Sessions for a Workspace, paginated with optional filters.
//
// The list is paginated by page number, or by cursor when opts.Cursor is set.
//
// https://docs.honcho.dev/v3/api-reference/endpoint/sessions/get-sessions
func (c *Client) GetSessions(ctx context.Context, workspaceID string, req *WorkspaceSessionGet, opts *GetSessionsOptions) (result *PageSession, err error) {
	if workspaceID == "" {
		err = errors.New("workspaceID is required")
		return
	}
	requestURL, err := c.endpoint(workspaceBaseURI, workspaceID, "sessions", "list")
	if err != nil {
		return
	}
	if opts != nil {
		query := requestURL.Query()
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
		if opts.IncludeScopes {
			query.Set("include_scopes", "true")
		}
		requestURL.RawQuery = query.Encode()
	}
	result = new(PageSession)
	if _, err = c.request(
		ctx, http.MethodPost, requestURL, nil,
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
func (c *Client) UpdateSession(ctx context.Context, workspaceID, sessionID string, req SessionUpdate) (result *Session, err error) {
	if workspaceID == "" {
		err = errors.New("workspaceID is required")
		return
	}
	if sessionID == "" {
		err = errors.New("sessionID is required")
		return
	}
	requestURL, err := c.endpoint(workspaceBaseURI, workspaceID, "sessions", sessionID)
	if err != nil {
		return
	}
	result = new(Session)
	if _, err = c.request(
		ctx, http.MethodPut, requestURL, nil,
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
// Until the queued deletion has run, the session ID answers 404 to GetOrCreateSession, which
// does not create a new session before then, and to UpdateSession and DeleteSession, as for a
// session that does not exist. See GetOrCreateSession for how long that lasts.
//
// This action cannot be undone.
//
// https://docs.honcho.dev/v3/api-reference/endpoint/sessions/delete-session
func (c *Client) DeleteSession(ctx context.Context, workspaceID, sessionID string) (err error) {
	if workspaceID == "" {
		err = errors.New("workspaceID is required")
		return
	}
	if sessionID == "" {
		err = errors.New("sessionID is required")
		return
	}
	requestURL, err := c.endpoint(workspaceBaseURI, workspaceID, "sessions", sessionID)
	if err != nil {
		return
	}
	if _, err = c.request(
		ctx, http.MethodDelete, requestURL, nil,
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
func (c *Client) CloneSession(ctx context.Context, workspaceID, sessionID string, opts *CloneSessionOptions) (result *Session, err error) {
	if workspaceID == "" {
		err = errors.New("workspaceID is required")
		return
	}
	if sessionID == "" {
		err = errors.New("sessionID is required")
		return
	}
	requestURL, err := c.endpoint(workspaceBaseURI, workspaceID, "sessions", sessionID, "clone")
	if err != nil {
		return
	}
	if opts != nil && opts.MessageID != nil {
		query := requestURL.Query()
		query.Set("message_id", *opts.MessageID)
		requestURL.RawQuery = query.Encode()
	}
	result = new(Session)
	if _, err = c.request(
		ctx, http.MethodPost, requestURL, nil,
		nil, &result,
	); err != nil {
		err = fmt.Errorf("failed to execute request: %w", err)
		return
	}
	return
}

// GetSessionPeers gets all Peers in a Session.
//
// The list is paginated by page number, or by cursor when opts.Cursor is set.
//
// https://docs.honcho.dev/v3/api-reference/endpoint/sessions/get-session-peers
func (c *Client) GetSessionPeers(ctx context.Context, workspaceID, sessionID string, opts *GetSessionPeersOptions) (result *PagePeer, err error) {
	if workspaceID == "" {
		err = errors.New("workspaceID is required")
		return
	}
	if sessionID == "" {
		err = errors.New("sessionID is required")
		return
	}
	requestURL, err := c.endpoint(workspaceBaseURI, workspaceID, "sessions", sessionID, "peers")
	if err != nil {
		return
	}
	if opts != nil {
		query := requestURL.Query()
		if opts.Page > 0 {
			query.Set("page", strconv.Itoa(opts.Page))
		}
		if opts.Size > 0 {
			query.Set("size", strconv.Itoa(opts.Size))
		}
		if err = setCursor(query, opts.Page, opts.Cursor); err != nil {
			return
		}
		requestURL.RawQuery = query.Encode()
	}
	result = new(PagePeer)
	if _, err = c.request(
		ctx, http.MethodGet, requestURL, nil,
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
func (c *Client) SetSessionPeers(ctx context.Context, workspaceID, sessionID string, peers map[string]*SessionPeerConfig) (result *Session, err error) {
	if workspaceID == "" {
		err = errors.New("workspaceID is required")
		return
	}
	if sessionID == "" {
		err = errors.New("sessionID is required")
		return
	}
	requestURL, err := c.endpoint(workspaceBaseURI, workspaceID, "sessions", sessionID, "peers")
	if err != nil {
		return
	}
	result = new(Session)
	if _, err = c.request(
		ctx, http.MethodPut, requestURL, nil,
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
func (c *Client) AddPeersToSession(ctx context.Context, workspaceID, sessionID string, peers map[string]*SessionPeerConfig) (result *Session, err error) {
	if workspaceID == "" {
		err = errors.New("workspaceID is required")
		return
	}
	if sessionID == "" {
		err = errors.New("sessionID is required")
		return
	}
	requestURL, err := c.endpoint(workspaceBaseURI, workspaceID, "sessions", sessionID, "peers")
	if err != nil {
		return
	}
	result = new(Session)
	if _, err = c.request(
		ctx, http.MethodPost, requestURL, nil,
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
func (c *Client) RemovePeersFromSession(ctx context.Context, workspaceID, sessionID string, peerIDs []string) (result *Session, err error) {
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
	requestURL, err := c.endpoint(workspaceBaseURI, workspaceID, "sessions", sessionID, "peers")
	if err != nil {
		return
	}
	result = new(Session)
	if _, err = c.request(
		ctx, http.MethodDelete, requestURL, nil,
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
// Scope reads the representation and peer card of PeerTarget from a scope, and Sessions confines them to
// an allowlist of sessions. A non-nil empty Sessions is refused before anything is sent.
//
// https://docs.honcho.dev/v3/api-reference/endpoint/sessions/get-session-context
func (c *Client) GetSessionContext(ctx context.Context, workspaceID, sessionID string, opts *GetSessionContextOptions) (result *SessionContext, err error) {
	if workspaceID == "" {
		err = errors.New("workspaceID is required")
		return
	}
	if sessionID == "" {
		err = errors.New("sessionID is required")
		return
	}
	// A non-nil empty allowlist would send no sessions parameter at all, silently turning a
	// restriction the caller asked for into unrestricted recall.
	if opts != nil && opts.Sessions != nil && len(opts.Sessions) == 0 {
		err = errors.New("sessions must hold at least one session: leave it nil for no restriction")
		return
	}
	requestURL, err := c.endpoint(workspaceBaseURI, workspaceID, "sessions", sessionID, "context")
	if err != nil {
		return
	}
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
		if opts.Scope != nil {
			query.Set("scope", *opts.Scope)
		}
		for _, session := range opts.Sessions {
			query.Add("sessions", session)
		}
		requestURL.RawQuery = query.Encode()
	}
	result = new(SessionContext)
	if _, err = c.request(
		ctx, http.MethodGet, requestURL, nil,
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
func (c *Client) GetSessionSummaries(ctx context.Context, workspaceID, sessionID string) (result *SessionSummaries, err error) {
	if workspaceID == "" {
		err = errors.New("workspaceID is required")
		return
	}
	if sessionID == "" {
		err = errors.New("sessionID is required")
		return
	}
	requestURL, err := c.endpoint(workspaceBaseURI, workspaceID, "sessions", sessionID, "summaries")
	if err != nil {
		return
	}
	result = new(SessionSummaries)
	if _, err = c.request(
		ctx, http.MethodGet, requestURL, nil,
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
func (c *Client) SearchSession(ctx context.Context, workspaceID, sessionID string, req MessageSearchOptions) (result []Message, err error) {
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
	requestURL, err := c.endpoint(workspaceBaseURI, workspaceID, "sessions", sessionID, "search")
	if err != nil {
		return
	}
	if _, err = c.request(
		ctx, http.MethodPost, requestURL, nil,
		req, &result,
	); err != nil {
		result = nil // a failed decode can leave items in it
		err = fmt.Errorf("failed to execute request: %w", err)
		return
	}
	return
}

// GetPeerConfig gets the configuration for a Peer in a Session.
//
// https://docs.honcho.dev/v3/api-reference/endpoint/sessions/get-peer-config
func (c *Client) GetPeerConfig(ctx context.Context, workspaceID, sessionID, peerID string) (result *SessionPeerConfig, err error) {
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
	requestURL, err := c.endpoint(workspaceBaseURI, workspaceID, "sessions", sessionID, "peers", peerID, "config")
	if err != nil {
		return
	}
	result = new(SessionPeerConfig)
	if _, err = c.request(
		ctx, http.MethodGet, requestURL, nil,
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
func (c *Client) SetPeerConfig(ctx context.Context, workspaceID, sessionID, peerID string, config SessionPeerConfig) (err error) {
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
	requestURL, err := c.endpoint(workspaceBaseURI, workspaceID, "sessions", sessionID, "peers", peerID, "config")
	if err != nil {
		return
	}
	if _, err = c.request(
		ctx, http.MethodPut, requestURL, nil,
		config, nil,
	); err != nil {
		err = fmt.Errorf("failed to execute request: %w", err)
		return
	}
	return
}
