package honcho

import (
	"fmt"
	"net/http"
)

const (
	workspaceBaseURI = "/v3/workspaces"
)

// GetOrCreateWorkspace gets a Workspace by ID or creates a new one.
//
// Provide the desired workspace ID in the request. If the workspace exists,
// it will be returned. If it doesn't exist, it will be created with that ID.
//
// https://docs.honcho.dev/v3/api-reference/endpoint/workspaces/get-or-create-workspace.md
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
