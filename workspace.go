package honcho

import (
	"fmt"
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
/* https://docs.honcho.dev/v3/api-reference/endpoint/workspaces/get-or-create-workspace.md */
func (c *Client) GetOrCreateWorkspace(req CreateWorkspaceRequest) (result *Workspace, err error) {
	if err = req.Validate(); err != nil {
		return
	}
	result = new(Workspace)
	if _, err = c.request(
		http.MethodPost, c.baseURL.JoinPath(workspaceBaseURI), nil,
		req, &result,
	); err != nil {
		err = fmt.Errorf("failed to execute request: %w", err)
		return
	}
	return
}

// GetAllWorkspaces gets all Workspaces, paginated with optional filters.
//
/* https://docs.honcho.dev/v3/api-reference/endpoint/workspaces/get-all-workspaces.md */
func (c *Client) GetAllWorkspaces(req WorkspaceGetRequest, opts *GetAllWorkspacesOptions) (result *PageWorkspace, err error) {
	requestURL := c.baseURL.JoinPath(workspaceBaseURI, "list")
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
		http.MethodPost, requestURL, nil,
		req, &result,
	); err != nil {
		err = fmt.Errorf("failed to execute request: %w", err)
		return
	}
	return
}
