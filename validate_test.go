package honcho

import (
	"strings"
	"testing"
)

// Workspace, peer and session IDs are accepted up to 512 characters, as
// upstream allows, and refused when longer, empty or holding a character
// outside ^[a-zA-Z0-9_-]+$. Regression test: Validate refused IDs over 100
// characters, so valid IDs of 101 to 512 characters never reached the server.
func TestValidateID(t *testing.T) {
	requests := []struct {
		name     string
		validate func(id string) error
	}{
		{"CreateWorkspaceRequest", func(id string) error { return CreateWorkspaceRequest{ID: id}.Validate() }},
		{"PeerCreate", func(id string) error { return PeerCreate{ID: id}.Validate() }},
		{"SessionCreate", func(id string) error { return SessionCreate{ID: id}.Validate() }},
	}
	tests := []struct {
		name    string
		id      string
		wantErr bool
		errMsg  string
	}{
		{name: "512 characters", id: strings.Repeat("a", 512)},
		{name: "513 characters", id: strings.Repeat("a", 513), wantErr: true, errMsg: "id must be 512 characters or less"},
		{name: "empty", id: "", wantErr: true, errMsg: "id is required"},
		{name: "character outside the pattern", id: "invalid@id", wantErr: true, errMsg: "id must contain only letters, numbers, underscores, or hyphens"},
	}
	for _, request := range requests {
		for _, tt := range tests {
			t.Run(request.name+"/"+tt.name, func(t *testing.T) {
				err := request.validate(tt.id)
				if (err != nil) != tt.wantErr {
					t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				}
				if err != nil && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Validate() error = %v, want error containing %q", err, tt.errMsg)
				}
			})
		}
	}
}
