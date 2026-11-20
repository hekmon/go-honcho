package honcho

import (
	"encoding/json"
	"net/http"
	"net/url"
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

// Conclusion content is accepted up to its limit in characters, as upstream counts
// them, and refused when longer, however many bytes each character takes.
// Regression test: Validate counted bytes, so text holding multi-byte characters
// was refused well below the limit and never reached the server.
func TestValidateTextLength(t *testing.T) {
	requests := []struct {
		name     string
		limit    int
		errMsg   string
		validate func(text string) error
	}{
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

// Message content and peer chat queries longer than Honcho's default limits pass
// Validate and reach the server in full, as both limits are server settings that
// a deployment can raise: MAX_MESSAGE_SIZE, 25000 characters by default, and
// DIALECTIC_MAX_QUERY_LENGTH, 10000 by default. Regression test: Validate refused
// longer text, so it never reached a server configured to accept it.
func TestValidateTextOverDefaultLimit(t *testing.T) {
	t.Run("message content", func(t *testing.T) {
		content := strings.Repeat("a", 50000)
		client, received := scopeServer(t, http.StatusCreated, `[]`)

		req := MessageBatchCreate{Messages: []MessageCreate{{Content: content, PeerID: "alice"}}}
		if _, err := client.CreateMessagesForSession(t.Context(), "workspace", "session", req); err != nil {
			t.Fatalf("CreateMessagesForSession() error = %v, want nil", err)
		}

		checkScopeRequest(t, received(), http.MethodPost, "/v3/workspaces/workspace/sessions/session/messages", url.Values{},
			`{"messages":[{"content":"`+content+`","peer_id":"alice"}]}`)
	})
	t.Run("peer chat query", func(t *testing.T) {
		query := strings.Repeat("a", 20000)
		client, received := scopeServer(t, http.StatusOK, `{"content":"ok"}`)

		if _, err := client.Chat(t.Context(), "workspace", "alice", DialecticOptions{Query: query}); err != nil {
			t.Fatalf("Chat() error = %v, want nil", err)
		}

		checkScopeRequest(t, received(), http.MethodPost, "/v3/workspaces/workspace/peers/alice/chat", url.Values{},
			`{"query":"`+query+`"}`)
	})
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

// A workspace search follows the rules of the other message searches: the query is required, and a
// limit of 0 is accepted while any other limit must be between 1 and 100. The scope and the filters
// are left to the server.
func TestValidateWorkspaceSearch(t *testing.T) {
	scope := ""
	tests := []struct {
		name    string
		request WorkspaceMessageSearchOptions
		errMsg  string
	}{
		{name: "limit 0", request: WorkspaceMessageSearchOptions{Query: "q"}},
		{name: "limit 1", request: WorkspaceMessageSearchOptions{Query: "q", Limit: 1}},
		{name: "limit 100", request: WorkspaceMessageSearchOptions{Query: "q", Limit: 100}},
		{name: "limit -1", request: WorkspaceMessageSearchOptions{Query: "q", Limit: -1}, errMsg: "limit must be between 1 and 100"},
		{name: "limit 101", request: WorkspaceMessageSearchOptions{Query: "q", Limit: 101}, errMsg: "limit must be between 1 and 100"},
		{name: "empty query", request: WorkspaceMessageSearchOptions{}, errMsg: "query is required"},
		{name: "empty scope name", request: WorkspaceMessageSearchOptions{Query: "q", Scope: &scope}},
		{name: "empty filters", request: WorkspaceMessageSearchOptions{Query: "q", Filters: map[string]any{}}},
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

// A peer card may be an empty list, which upstream takes as clearing the card, and
// is sent as an empty list. A nil card stays refused, as it would be sent as null
// and upstream refuses null. Regression test: Validate refused an empty card, so a
// request clearing a peer card never reached the server.
func TestValidatePeerCard(t *testing.T) {
	tests := []struct {
		name    string
		request PeerCardSet
		errMsg  string
	}{
		{name: "card with entries", request: PeerCardSet{PeerCard: []string{"a", "b"}}},
		{name: "empty card", request: PeerCardSet{PeerCard: []string{}}},
		{name: "nil card", request: PeerCardSet{}, errMsg: "peer_card is required"},
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
	t.Run("empty card is sent as an empty list", func(t *testing.T) {
		body, err := json.Marshal(PeerCardSet{PeerCard: []string{}})
		if err != nil {
			t.Fatalf("json.Marshal() error = %v", err)
		}
		if want := `{"peer_card":[]}`; string(body) != want {
			t.Errorf("json.Marshal() = %s, want %s", body, want)
		}
	})
}

// A conclusion query must name both the observer and the observed peer in its
// filters, as upstream requires. For each side either key may hold the value, and a
// missing, nil or empty-string value does not count. The query, top_k and distance
// checks still apply. Regression test: Validate did not check the filters, so a
// query that did not name both peers was sent and always came back as a 422.
func TestValidateConclusionQuery(t *testing.T) {
	const peersErr = "filters must name the observer (observer or observer_id) and the observed peer (observed or observed_id)"
	peers := map[string]any{"observer": "alice", "observed": "bob"}
	distance := 1.5
	tests := []struct {
		name    string
		request ConclusionQuery
		errMsg  string
	}{
		{name: "no filters", request: ConclusionQuery{Query: "q"}, errMsg: peersErr},
		{name: "empty filters", request: ConclusionQuery{Query: "q", Filters: map[string]any{}}, errMsg: peersErr},
		{name: "observer only", request: ConclusionQuery{Query: "q", Filters: map[string]any{"observer": "alice"}}, errMsg: peersErr},
		{name: "observed only", request: ConclusionQuery{Query: "q", Filters: map[string]any{"observed": "bob"}}, errMsg: peersErr},
		{name: "observer and observed", request: ConclusionQuery{Query: "q", Filters: map[string]any{"observer": "alice", "observed": "bob"}}},
		{name: "observer_id and observed_id", request: ConclusionQuery{Query: "q", Filters: map[string]any{"observer_id": "alice", "observed_id": "bob"}}},
		{name: "observer and observed_id", request: ConclusionQuery{Query: "q", Filters: map[string]any{"observer": "alice", "observed_id": "bob"}}},
		{name: "observer_id and observed", request: ConclusionQuery{Query: "q", Filters: map[string]any{"observer_id": "alice", "observed": "bob"}}},
		{name: "empty observer with observed", request: ConclusionQuery{Query: "q", Filters: map[string]any{"observer": "", "observed": "bob"}}, errMsg: peersErr},
		{name: "empty observed with observer", request: ConclusionQuery{Query: "q", Filters: map[string]any{"observer": "alice", "observed": ""}}, errMsg: peersErr},
		{name: "observer and observed plus another filter", request: ConclusionQuery{Query: "q", Filters: map[string]any{"observer": "alice", "observed": "bob", "level": "explicit"}}},
		{name: "empty observer with non-empty observer_id", request: ConclusionQuery{Query: "q", Filters: map[string]any{"observer": "", "observer_id": "alice", "observed": "bob"}}},
		{name: "observer with empty observer_id", request: ConclusionQuery{Query: "q", Filters: map[string]any{"observer": "alice", "observer_id": "", "observed": "bob"}}},
		{name: "empty observed with non-empty observed_id", request: ConclusionQuery{Query: "q", Filters: map[string]any{"observer": "alice", "observed": "", "observed_id": "bob"}}},
		{name: "nil observer", request: ConclusionQuery{Query: "q", Filters: map[string]any{"observer": nil, "observed": "bob"}}, errMsg: peersErr},
		{name: "nil observed", request: ConclusionQuery{Query: "q", Filters: map[string]any{"observer": "alice", "observed": nil}}, errMsg: peersErr},
		{name: "empty query", request: ConclusionQuery{Filters: peers}, errMsg: "query is required"},
		{name: "empty query without filters", request: ConclusionQuery{}, errMsg: "query is required"},
		{name: "top_k 101", request: ConclusionQuery{Query: "q", Filters: peers, TopK: 101}, errMsg: "top_k must be between 1 and 100"},
		{name: "distance 1.5", request: ConclusionQuery{Query: "q", Filters: peers, Distance: &distance}, errMsg: "distance must be between 0 and 1"},
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
