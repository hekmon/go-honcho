package honcho

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"net/http"
	"strconv"
)

const (
	workspaceBaseURI = "/v3/workspaces"
)

// GetOrCreateWorkspace gets a Workspace by ID or creates a new one.
//
// Provide the desired workspace ID in the request. If the workspace exists,
// it will be returned. If it doesn't exist, it will be created with that ID.
//
// https://docs.honcho.dev/v3/api-reference/endpoint/workspaces/get-or-create-workspace
func (c *Client) GetOrCreateWorkspace(ctx context.Context, req CreateWorkspaceRequest) (result *Workspace, err error) {
	if err = req.Validate(); err != nil {
		return
	}
	requestURL, err := c.endpoint(workspaceBaseURI)
	if err != nil {
		return
	}
	result = new(Workspace)
	if _, err = c.request(
		ctx, http.MethodPost, requestURL, nil,
		req, &result,
	); err != nil {
		err = fmt.Errorf("failed to execute request: %w", err)
		return
	}
	return
}

// GetAllWorkspaces gets all Workspaces, paginated with optional filters.
//
// https://docs.honcho.dev/v3/api-reference/endpoint/workspaces/get-all-workspaces
func (c *Client) GetAllWorkspaces(ctx context.Context, req WorkspaceGetRequest, opts *GetAllWorkspacesOptions) (result *PageWorkspace, err error) {
	requestURL, err := c.endpoint(workspaceBaseURI, "list")
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
		requestURL.RawQuery = query.Encode()
	}
	result = new(PageWorkspace)
	if _, err = c.request(
		ctx, http.MethodPost, requestURL, nil,
		req, &result,
	); err != nil {
		err = fmt.Errorf("failed to execute request: %w", err)
		return
	}
	return
}

// UpdateWorkspace updates a Workspace's metadata and/or configuration.
//
// https://docs.honcho.dev/v3/api-reference/endpoint/workspaces/update-workspace
func (c *Client) UpdateWorkspace(ctx context.Context, workspaceID string, req UpdateWorkspaceRequest) (result *Workspace, err error) {
	if workspaceID == "" {
		err = errors.New("workspaceID is required")
		return
	}
	requestURL, err := c.endpoint(workspaceBaseURI, workspaceID)
	if err != nil {
		return
	}
	result = new(Workspace)
	if _, err = c.request(
		ctx, http.MethodPut, requestURL, nil,
		req, &result,
	); err != nil {
		err = fmt.Errorf("failed to execute request: %w", err)
		return
	}
	return
}

// DeleteWorkspace deletes a Workspace. This accepts the deletion request and processes it in the background,
// permanently deleting all peers, messages, conclusions, and other resources associated with the workspace.
//
// Returns 409 Conflict if the workspace contains active sessions. Delete all sessions first, then delete the workspace.
// This action cannot be undone.
//
// https://docs.honcho.dev/v3/api-reference/endpoint/workspaces/delete-workspace
func (c *Client) DeleteWorkspace(ctx context.Context, workspaceID string) (err error) {
	if workspaceID == "" {
		err = errors.New("workspaceID is required")
		return
	}
	requestURL, err := c.endpoint(workspaceBaseURI, workspaceID)
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

// GetQueueStatus gets the processing queue status for a Workspace, optionally scoped to an observer, sender, and/or session.
//
// Only tracks user-facing task types (representation, summary, dream). Internal infrastructure tasks
// (reconciler, webhook, deletion) are excluded. Note: completed counts reflect items since the last
// periodic queue cleanup, not lifetime totals.
//
// https://docs.honcho.dev/v3/api-reference/endpoint/workspaces/get-queue-status
func (c *Client) GetQueueStatus(ctx context.Context, workspaceID string, observerID, senderID, sessionID *string) (result *QueueStatus, err error) {
	if workspaceID == "" {
		err = errors.New("workspaceID is required")
		return
	}
	requestURL, err := c.endpoint(workspaceBaseURI, workspaceID, "queue", "status")
	if err != nil {
		return
	}
	query := requestURL.Query()
	if observerID != nil {
		query.Set("observer_id", *observerID)
	}
	if senderID != nil {
		query.Set("sender_id", *senderID)
	}
	if sessionID != nil {
		query.Set("session_id", *sessionID)
	}
	requestURL.RawQuery = query.Encode()
	result = new(QueueStatus)
	if _, err = c.request(
		ctx, http.MethodGet, requestURL, nil,
		nil, &result,
	); err != nil {
		err = fmt.Errorf("failed to execute request: %w", err)
		return
	}
	return
}

// ScheduleDream manually schedules a dream task for a specific collection.
//
// This endpoint bypasses all automatic dream conditions (document threshold, minimum hours between dreams)
// and schedules the dream task for a future execution. Currently this endpoint only supports scheduling
// immediate dreams.
//
// https://docs.honcho.dev/v3/api-reference/endpoint/workspaces/schedule-dream
func (c *Client) ScheduleDream(ctx context.Context, workspaceID string, req ScheduleDreamRequest) (err error) {
	if workspaceID == "" {
		err = errors.New("workspaceID is required")
		return
	}
	if err = req.Validate(); err != nil {
		return
	}
	requestURL, err := c.endpoint(workspaceBaseURI, workspaceID, "schedule_dream")
	if err != nil {
		return
	}
	if _, err = c.request(
		ctx, http.MethodPost, requestURL, nil,
		req, nil,
	); err != nil {
		err = fmt.Errorf("failed to execute request: %w", err)
		return
	}
	return
}

// SearchWorkspace searches messages in a Workspace using optional filters.
//
// https://docs.honcho.dev/v3/api-reference/endpoint/workspaces/search-workspace
func (c *Client) SearchWorkspace(ctx context.Context, workspaceID string, req MessageSearchOptions) (result *[]Message, err error) {
	if workspaceID == "" {
		err = errors.New("workspaceID is required")
		return
	}
	if err = req.Validate(); err != nil {
		return
	}
	requestURL, err := c.endpoint(workspaceBaseURI, workspaceID, "search")
	if err != nil {
		return
	}
	result = new([]Message)
	if _, err = c.request(
		ctx, http.MethodPost, requestURL, nil,
		req, &result,
	); err != nil {
		err = fmt.Errorf("failed to execute request: %w", err)
		return
	}
	return
}

// WorkspaceChat queries a whole Workspace using natural language.
//
// Runs the Dialectic agent over every peer in the workspace rather than a single
// (observer, observed) pair: there is no anchor peer and no target. Set SessionID to
// narrow message tools to one session, or Scope to restrict recall to the union of the
// named scopes' member sessions (always an allowlist, even for a single name: a scope
// with no member sessions recalls nothing). SessionID and Scope are mutually exclusive.
// Requires a workspace- or admin-level key.
//
// WorkspaceChat reads the whole answer at once: use WorkspaceChatStream to read it as
// an event stream.
//
// https://docs.honcho.dev/v3/api-reference/endpoint/workspaces/chat-workspace
func (c *Client) WorkspaceChat(ctx context.Context, workspaceID string, req WorkspaceChatOptions) (result *DialecticResponse, err error) {
	if workspaceID == "" {
		err = errors.New("workspaceID is required")
		return
	}
	if err = req.Validate(); err != nil {
		return
	}
	requestURL, err := c.endpoint(workspaceBaseURI, workspaceID, "chat")
	if err != nil {
		return
	}
	result = new(DialecticResponse)
	if _, err = c.request(
		ctx, http.MethodPost, requestURL, nil,
		req, &result,
	); err != nil {
		err = fmt.Errorf("failed to execute request: %w", err)
		return
	}
	return
}

// WorkspaceChatStream queries a whole Workspace using natural language, as WorkspaceChat
// does, and returns the answer chunk by chunk.
//
// The answer arrives when the agent finishes gathering context, usually as a single
// chunk: only an answer synthesized at the tool-call limit streams as it is generated.
//
// The request is sent when the returned iterator is ranged over. Any failure yields a
// single error and ends the iteration, and that includes a stream the server ends
// before finishing the answer: an iteration that ends without an error has read the
// whole answer. Breaking out of the loop cancels the request.
//
// An iterator has no way to return evidence, so a request with IncludeEvidence set is
// refused before anything is sent: use WorkspaceChat to read the evidence.
//
//	for chunk, err := range client.WorkspaceChatStream(ctx, workspaceID, req) {
//		if err != nil {
//			return err
//		}
//		fmt.Print(chunk)
//	}
//
// https://docs.honcho.dev/v3/api-reference/endpoint/workspaces/chat-workspace
func (c *Client) WorkspaceChatStream(ctx context.Context, workspaceID string, req WorkspaceChatOptions) iter.Seq2[string, error] {
	return func(yield func(string, error) bool) {
		if workspaceID == "" {
			yield("", errors.New("workspaceID is required"))
			return
		}
		if req.IncludeEvidence {
			yield("", errors.New("include_evidence is not supported by WorkspaceChatStream: use WorkspaceChat instead"))
			return
		}
		if err := req.Validate(); err != nil {
			yield("", err)
			return
		}
		requestURL, err := c.endpoint(workspaceBaseURI, workspaceID, "chat")
		if err != nil {
			yield("", err)
			return
		}
		c.streamDialectic(ctx, requestURL, workspaceChatStreamRequest{WorkspaceChatOptions: req, Stream: true}, yield)
	}
}
