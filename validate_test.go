package honcho

import (
	"encoding/json"
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

// Message content, chat queries and conclusion content are accepted up to their
// limit in characters, as upstream counts them, and refused when longer, however
// many bytes each character takes. Regression test: Validate counted bytes, so
// text holding multi-byte characters was refused well below the limit and never
// reached the server.
func TestValidateTextLength(t *testing.T) {
	requests := []struct {
		name     string
		limit    int
		errMsg   string
		validate func(text string) error
	}{
		{"MessageCreate", 25000, "content must be 25000 characters or less", func(text string) error {
			return MessageCreate{Content: text, PeerID: "peer"}.Validate()
		}},
		{"DialecticOptions", 10000, "query must be 10000 characters or less", func(text string) error {
			return DialecticOptions{Query: text}.Validate()
		}},
		{"ConclusionCreate", 65535, "content must be 65535 characters or less", func(text string) error {
			return ConclusionCreate{Content: text, ObserverID: "observer", ObservedID: "observed"}.Validate()
		}},
	}
	characters := []struct {
		name      string
		character string
	}{
		{"1-byte character", "a"},
		{"2-byte character", "é"},
		{"4-byte character", "😀"},
	}
	for _, request := range requests {
		for _, tt := range characters {
			t.Run(request.name+"/"+tt.name+"/at the limit", func(t *testing.T) {
				if err := request.validate(strings.Repeat(tt.character, request.limit)); err != nil {
					t.Errorf("Validate() error = %v, want nil", err)
				}
			})
			t.Run(request.name+"/"+tt.name+"/one over the limit", func(t *testing.T) {
				err := request.validate(strings.Repeat(tt.character, request.limit+1))
				if err == nil || err.Error() != request.errMsg {
					t.Errorf("Validate() error = %v, want %q", err, request.errMsg)
				}
			})
		}
	}
}

// Message content may be empty, as upstream declares it with min_length=0, while
// the peer ID stays required. Conclusion content and chat queries still need at
// least one character, as upstream requires. Regression test: Validate refused
// empty message content, so a message or batch holding one never reached the
// server.
func TestValidateEmptyContent(t *testing.T) {
	tests := []struct {
		name    string
		request interface{ Validate() error }
		errMsg  string
	}{
		{name: "MessageCreate with empty content", request: MessageCreate{PeerID: "peer"}},
		{name: "MessageBatchCreate holding an empty message", request: MessageBatchCreate{Messages: []MessageCreate{
			{Content: "hello", PeerID: "peer"},
			{PeerID: "peer"},
		}}},
		{name: "MessageCreate without peer ID", request: MessageCreate{Content: "hello"}, errMsg: "peer_id is required"},
		{name: "ConclusionCreate with empty content", request: ConclusionCreate{ObserverID: "observer", ObservedID: "observed"}, errMsg: "content is required"},
		{name: "DialecticOptions with empty query", request: DialecticOptions{}, errMsg: "query is required"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.request.Validate()
			if tt.errMsg == "" {
				if err != nil {
					t.Errorf("Validate() error = %v, want nil", err)
				}
				return
			}
			if err == nil || err.Error() != tt.errMsg {
				t.Errorf("Validate() error = %v, want %q", err, tt.errMsg)
			}
		})
	}
}

// A search limit of 0 is accepted and left out of the request, so the server
// applies its default of 10, since upstream refuses an explicit 0. Any other limit
// must be between 1 and 100, and the query stays required. Regression test:
// Validate refused a limit of 0, so a search that did not set a limit never
// reached the server.
func TestValidateSearchLimit(t *testing.T) {
	tests := []struct {
		name    string
		request MessageSearchOptions
		errMsg  string
	}{
		{name: "limit 0", request: MessageSearchOptions{Query: "q"}},
		{name: "limit 1", request: MessageSearchOptions{Query: "q", Limit: 1}},
		{name: "limit 100", request: MessageSearchOptions{Query: "q", Limit: 100}},
		{name: "limit -1", request: MessageSearchOptions{Query: "q", Limit: -1}, errMsg: "limit must be between 1 and 100"},
		{name: "limit 101", request: MessageSearchOptions{Query: "q", Limit: 101}, errMsg: "limit must be between 1 and 100"},
		{name: "empty query", request: MessageSearchOptions{}, errMsg: "query is required"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.request.Validate()
			if tt.errMsg == "" {
				if err != nil {
					t.Errorf("Validate() error = %v, want nil", err)
				}
				return
			}
			if err == nil || err.Error() != tt.errMsg {
				t.Errorf("Validate() error = %v, want %q", err, tt.errMsg)
			}
		})
	}
	t.Run("limit 0 is left out of the request", func(t *testing.T) {
		body, err := json.Marshal(MessageSearchOptions{Query: "q"})
		if err != nil {
			t.Fatalf("json.Marshal() error = %v", err)
		}
		if strings.Contains(string(body), `"limit"`) {
			t.Errorf("json.Marshal() = %s, want no limit field", body)
		}
	})
}
