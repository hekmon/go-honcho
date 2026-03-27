package honcho

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
)

// GetOrCreatePeer gets a Peer by ID or creates a new Peer with the given ID.
//
// If peer_id is provided as a query parameter, it uses that (must match JWT workspace_id).
// Otherwise, it uses the peer_id from the JWT.
//
// https://docs.honcho.dev/v3/api-reference/endpoint/peers/get-or-create-peer
func (c *Client) GetOrCreatePeer(ctx context.Context, workspaceID string, req PeerCreate) (result *Peer, err error) {
	if workspaceID == "" {
		err = errors.New("workspaceID is required")
		return
	}
	if err = req.Validate(); err != nil {
		return
	}
	result = new(Peer)
	if _, err = c.request(
		ctx, http.MethodPost, c.baseURL.JoinPath(workspaceBaseURI, workspaceID, "peers"), nil,
		req, &result,
	); err != nil {
		err = fmt.Errorf("failed to execute request: %w", err)
		return
	}
	return
}

// GetAllPeers gets all Peers for a Workspace, paginated with optional filters.
//
// https://docs.honcho.dev/v3/api-reference/endpoint/peers/get-peers
func (c *Client) GetAllPeers(ctx context.Context, workspaceID string, req *PeerGet, opts *GetAllPeersOptions) (result *PagePeer, err error) {
	if workspaceID == "" {
		err = errors.New("workspaceID is required")
		return
	}
	requestURL := c.baseURL.JoinPath(workspaceBaseURI, workspaceID, "peers", "list")
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
		ctx, http.MethodPost, requestURL, nil,
		req, &result,
	); err != nil {
		err = fmt.Errorf("failed to execute request: %w", err)
		return
	}
	return
}

// UpdatePeer updates a Peer's metadata and/or configuration.
//
// https://docs.honcho.dev/v3/api-reference/endpoint/peers/update-peer
func (c *Client) UpdatePeer(ctx context.Context, workspaceID, peerID string, req PeerUpdate) (result *Peer, err error) {
	if workspaceID == "" {
		err = errors.New("workspaceID is required")
		return
	}
	if peerID == "" {
		err = errors.New("peerID is required")
		return
	}
	result = new(Peer)
	if _, err = c.request(
		ctx, http.MethodPut, c.baseURL.JoinPath(workspaceBaseURI, workspaceID, "peers", peerID), nil,
		req, &result,
	); err != nil {
		err = fmt.Errorf("failed to execute request: %w", err)
		return
	}
	return
}

// GetRepresentation gets a curated subset of a Peer's Representation.
//
// A Representation is always a subset of the total knowledge about the Peer.
// The subset can be scoped and filtered in various ways.
//
// If a session_id is provided in the body, we get the Representation of the Peer scoped to that Session.
// If a target is provided, we get the Representation of the target from the perspective of the Peer.
// If no target is provided, we get the omniscient Honcho Representation of the Peer.
//
// https://docs.honcho.dev/v3/api-reference/endpoint/peers/get-representation
func (c *Client) GetRepresentation(ctx context.Context, workspaceID, peerID string, req PeerRepresentationGet) (result *RepresentationResponse, err error) {
	if workspaceID == "" {
		err = errors.New("workspaceID is required")
		return
	}
	if peerID == "" {
		err = errors.New("peerID is required")
		return
	}
	result = new(RepresentationResponse)
	if _, err = c.request(
		ctx, http.MethodPost, c.baseURL.JoinPath(workspaceBaseURI, workspaceID, "peers", peerID, "representation"), nil,
		req, &result,
	); err != nil {
		err = fmt.Errorf("failed to execute request: %w", err)
		return
	}
	return
}

// GetPeerCard gets a peer card for a specific peer relationship.
//
// Returns the peer card that the observer peer has for the target peer if it exists.
// If no target is specified, returns the observer's own peer card.
//
// https://docs.honcho.dev/v3/api-reference/endpoint/peers/get-peer-card
func (c *Client) GetPeerCard(ctx context.Context, workspaceID, peerID string, target *string) (result *PeerCardResponse, err error) {
	if workspaceID == "" {
		err = errors.New("workspaceID is required")
		return
	}
	if peerID == "" {
		err = errors.New("peerID is required")
		return
	}
	requestURL := c.baseURL.JoinPath(workspaceBaseURI, workspaceID, "peers", peerID, "card")
	if target != nil {
		query := requestURL.Query()
		query.Set("target", *target)
		requestURL.RawQuery = query.Encode()
	}
	result = new(PeerCardResponse)
	if _, err = c.request(
		ctx, http.MethodGet, requestURL, nil,
		nil, &result,
	); err != nil {
		err = fmt.Errorf("failed to execute request: %w", err)
		return
	}
	return
}

// SetPeerCard sets a peer card for a specific peer relationship.
//
// Sets the peer card that the observer peer has for the target peer.
// If no target is specified, sets the observer's own peer card.
//
// https://docs.honcho.dev/v3/api-reference/endpoint/peers/set-peer-card
func (c *Client) SetPeerCard(ctx context.Context, workspaceID, peerID string, req PeerCardSet, target *string) (result *PeerCardResponse, err error) {
	if workspaceID == "" {
		err = errors.New("workspaceID is required")
		return
	}
	if peerID == "" {
		err = errors.New("peerID is required")
		return
	}
	if err = req.Validate(); err != nil {
		return
	}
	requestURL := c.baseURL.JoinPath(workspaceBaseURI, workspaceID, "peers", peerID, "card")
	if target != nil {
		query := requestURL.Query()
		query.Set("target", *target)
		requestURL.RawQuery = query.Encode()
	}
	result = new(PeerCardResponse)
	if _, err = c.request(
		ctx, http.MethodPut, requestURL, nil,
		req, &result,
	); err != nil {
		err = fmt.Errorf("failed to execute request: %w", err)
		return
	}
	return
}

// GetPeerContext gets context for a peer, including their representation and peer card.
//
// This endpoint returns a curated subset of the representation and peer card for a peer.
// If a target is specified, returns the context for the target from the observer peer's perspective.
// If no target is specified, returns the peer's own context (self-observation).
//
// This is useful for getting all the context needed about a peer without making multiple API calls.
//
// https://docs.honcho.dev/v3/api-reference/endpoint/peers/get-peer-context
func (c *Client) GetPeerContext(ctx context.Context, workspaceID, peerID string, opts *GetPeerContextOptions) (result *PeerContext, err error) {
	if workspaceID == "" {
		err = errors.New("workspaceID is required")
		return
	}
	if peerID == "" {
		err = errors.New("peerID is required")
		return
	}
	requestURL := c.baseURL.JoinPath(workspaceBaseURI, workspaceID, "peers", peerID, "context")
	query := requestURL.Query()
	if opts != nil {
		if opts.Target != nil {
			query.Set("target", *opts.Target)
		}
		if opts.SearchQuery != nil {
			query.Set("search_query", *opts.SearchQuery)
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
	}
	requestURL.RawQuery = query.Encode()
	result = new(PeerContext)
	if _, err = c.request(
		ctx, http.MethodGet, requestURL, nil,
		nil, &result,
	); err != nil {
		err = fmt.Errorf("failed to execute request: %w", err)
		return
	}
	return
}

// GetSessionsForPeer gets all Sessions for a Peer, paginated with optional filters.
//
// https://docs.honcho.dev/v3/api-reference/endpoint/peers/get-sessions-for-peer
func (c *Client) GetSessionsForPeer(ctx context.Context, workspaceID, peerID string, req *SessionGet, opts *GetSessionsForPeerOptions) (result *PageSession, err error) {
	if workspaceID == "" {
		err = errors.New("workspaceID is required")
		return
	}
	if peerID == "" {
		err = errors.New("peerID is required")
		return
	}
	requestURL := c.baseURL.JoinPath(workspaceBaseURI, workspaceID, "peers", peerID, "sessions")
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
		ctx, http.MethodPost, requestURL, nil,
		req, &result,
	); err != nil {
		err = fmt.Errorf("failed to execute request: %w", err)
		return
	}
	return
}

// SearchPeer searches a Peer's messages, optionally filtered by various criteria.
//
// https://docs.honcho.dev/v3/api-reference/endpoint/peers/search-peer
func (c *Client) SearchPeer(ctx context.Context, workspaceID, peerID string, req MessageSearchOptions) (result *[]Message, err error) {
	if workspaceID == "" {
		err = errors.New("workspaceID is required")
		return
	}
	if peerID == "" {
		err = errors.New("peerID is required")
		return
	}
	if err = req.Validate(); err != nil {
		return
	}
	result = new([]Message)
	if _, err = c.request(
		ctx, http.MethodPost, c.baseURL.JoinPath(workspaceBaseURI, workspaceID, "peers", peerID, "search"), nil,
		req, &result,
	); err != nil {
		err = fmt.Errorf("failed to execute request: %w", err)
		return
	}
	return
}

// Chat queries a Peer's representation using natural language.
//
// Performs agentic search and reasoning to comprehensively answer the query based on
// all latent knowledge gathered about the peer from their messages and conclusions.
//
// https://docs.honcho.dev/v3/api-reference/endpoint/peers/chat
func (c *Client) Chat(ctx context.Context, workspaceID, peerID string, req DialecticOptions) (result *DialecticResponse, err error) {
	if workspaceID == "" {
		err = errors.New("workspaceID is required")
		return
	}
	if peerID == "" {
		err = errors.New("peerID is required")
		return
	}
	if err = req.Validate(); err != nil {
		return
	}
	result = new(DialecticResponse)
	if _, err = c.request(
		ctx, http.MethodPost, c.baseURL.JoinPath(workspaceBaseURI, workspaceID, "peers", peerID, "chat"), nil,
		req, &result,
	); err != nil {
		err = fmt.Errorf("failed to execute request: %w", err)
		return
	}
	return
}
