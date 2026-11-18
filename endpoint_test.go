package honcho

import (
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

// An ID reaches the server as one literal path segment, wherever it stands in the
// path: dots inside a name are kept, every character unsafe in a segment is escaped,
// an escape the ID already holds is escaped again, and a base URL keeps its path
// prefix. Regression test: IDs went into the path unescaped, so "a%b" sent the
// request to the base URL itself, and "victim%2Fpeers" reached the server as a %2F,
// which Honcho decodes before routing.
func TestEndpoint(t *testing.T) {
	tests := []struct {
		name       string
		basePrefix string
		id         string
		want       string
	}{
		{name: "letters, digits, underscore and hyphen", id: "abc_-1", want: "/v3/workspaces/abc_-1/conclusions/abc_-1"},
		{name: "dot inside a name", id: "scope.team", want: "/v3/workspaces/scope.team/conclusions/scope.team"},
		{name: "two dots inside a name", id: "a..b", want: "/v3/workspaces/a..b/conclusions/a..b"},
		{name: "space", id: "a b", want: "/v3/workspaces/a%20b/conclusions/a%20b"},
		{name: "question mark", id: "a?b", want: "/v3/workspaces/a%3Fb/conclusions/a%3Fb"},
		{name: "hash", id: "a#b", want: "/v3/workspaces/a%23b/conclusions/a%23b"},
		{name: "percent", id: "a%b", want: "/v3/workspaces/a%25b/conclusions/a%25b"},
		{name: "non-ASCII character", id: "é", want: "/v3/workspaces/%C3%A9/conclusions/%C3%A9"},
		{name: "backslash", id: `a\b`, want: "/v3/workspaces/a%5Cb/conclusions/a%5Cb"},
		{name: "escaped slash", id: "victim%2Fpeers", want: "/v3/workspaces/victim%252Fpeers/conclusions/victim%252Fpeers"},
		{name: "escaped dots", id: "%2e%2e", want: "/v3/workspaces/%252e%252e/conclusions/%252e%252e"},
		{name: "base URL with a path prefix", basePrefix: "prefix", id: "a b", want: "/prefix/v3/workspaces/a%20b/conclusions/a%20b"},
		{name: "base URL with a path prefix and a trailing slash", basePrefix: "prefix/", id: "a b", want: "/prefix/v3/workspaces/a%20b/conclusions/a%20b"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u, err := url.Parse("https://honcho.test/" + tt.basePrefix)
			if err != nil {
				t.Fatalf("failed to parse the base URL: %v", err)
			}

			got, err := New(&Options{BaseURL: u}).endpoint(workspaceBaseURI, tt.id, "conclusions", tt.id)
			if err != nil {
				t.Fatalf("endpoint() error = %v, want nil", err)
			}
			if path := got.EscapedPath(); path != tt.want {
				t.Errorf("EscapedPath() = %q, want %q", path, tt.want)
			}
		})
	}
}

// An ID that is empty, is "." or "..", or holds a "/" is refused wherever it stands
// in the path, with an error matching ErrInvalidID that quotes it. Regression test:
// JoinPath cleaned such an ID away or followed it, so DeleteConclusion with the ID
// "../sessions/victim" deleted the session victim.
func TestEndpointRefusesUnsafeIDs(t *testing.T) {
	tests := []struct {
		name string
		id   string
	}{
		{name: "empty", id: ""},
		{name: "dot", id: "."},
		{name: "two dots", id: ".."},
		{name: "slash", id: "/"},
		{name: "slash inside a name", id: "a/b"},
		{name: "parent path before a name", id: "../x"},
		{name: "parent path after a name", id: "x/.."},
	}
	client := New(nil)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, segments := range [][]string{{tt.id, "conclusions"}, {"workspace", "conclusions", tt.id}} {
				got, err := client.endpoint(workspaceBaseURI, segments...)
				if !errors.Is(err, ErrInvalidID) {
					t.Fatalf("endpoint(%q) error = %v, want %v", segments, err, ErrInvalidID)
				}
				if want := fmt.Sprintf("%q", tt.id); !strings.Contains(err.Error(), want) {
					t.Errorf("endpoint(%q) error = %q, want it to quote the ID as %s", segments, err, want)
				}
				if got != nil {
					t.Errorf("endpoint(%q) = %v, want no URL", segments, got)
				}
			}
		})
	}
}

// Every method taking an ID refuses an unsafe one in each of its ID parameters, with
// an error matching ErrInvalidID and without sending a request, while the same call
// with a safe ID is sent. Regression test: each method joined the IDs it was given to
// the path itself, so an ID could point any of them at another resource, in another
// workspace as well.
func TestMethodsRefuseUnsafeIDs(t *testing.T) {
	var requests atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	u, _ := url.Parse(server.URL)
	client := New(&Options{BaseURL: u})
	ctx := t.Context()

	var (
		conclusions = ConclusionBatchCreate{Conclusions: []ConclusionCreate{{Content: "content", ObserverID: "alice", ObservedID: "bob"}}}
		query       = ConclusionQuery{Query: "query", Filters: map[string]any{"observer": "alice", "observed": "bob"}}
		messages    = MessageBatchCreate{Messages: []MessageCreate{{Content: "content", PeerID: "alice"}}}
		upload      = MessageUpload{File: []byte("content"), Filename: "file.txt", PeerID: "alice"}
		peer        = PeerCreate{ID: "alice"}
		card        = PeerCardSet{PeerCard: []string{"card"}}
		search      = MessageSearchOptions{Query: "query"}
		wsSearch    = WorkspaceMessageSearchOptions{Query: "query"}
		dialectic   = DialecticOptions{Query: "query"}
		session     = SessionCreate{ID: "session"}
		peers       = map[string]*SessionPeerConfig{"alice": {}}
		webhook     = WebhookEndpointCreate{URL: "https://example.test/webhook"}
		dream       = ScheduleDreamRequest{Observer: "alice", DreamType: DreamTypeOmni}
		chatOptions = WorkspaceChatOptions{Query: "query"}
		scope       = ScopeCreate{ID: "team"}
		scopeAdd    = ScopeSessionsAdd{SessionIDs: []string{"session"}}
		scopeRemove = ScopeSessionsRemove{SessionIDs: []string{"session"}}
	)
	// chatStream ranges over a ChatStream to its end and returns the one error it
	// must yield before stopping.
	chatStream := func(workspaceID, peerID string) error {
		var yielded []error
		for _, err := range client.ChatStream(ctx, workspaceID, peerID, dialectic) {
			yielded = append(yielded, err)
		}
		if len(yielded) != 1 {
			return fmt.Errorf("%d values yielded, want one error: %v", len(yielded), yielded)
		}
		return yielded[0]
	}
	// workspaceChatStream ranges over a WorkspaceChatStream to its end and returns the
	// one error it must yield before stopping.
	workspaceChatStream := func(workspaceID string) error {
		var yielded []error
		for _, err := range client.WorkspaceChatStream(ctx, workspaceID, chatOptions) {
			yielded = append(yielded, err)
		}
		if len(yielded) != 1 {
			return fmt.Errorf("%d values yielded, want one error: %v", len(yielded), yielded)
		}
		return yielded[0]
	}

	tests := []struct {
		name string
		call func(id string) error
	}{
		{"CreateConclusions/workspaceID", func(id string) error { return errorOf(client.CreateConclusions(ctx, id, conclusions)) }},
		{"DeleteConclusion/workspaceID", func(id string) error { return client.DeleteConclusion(ctx, id, "conclusion") }},
		{"DeleteConclusion/conclusionID", func(id string) error { return client.DeleteConclusion(ctx, "workspace", id) }},
		{"GetConclusion/workspaceID", func(id string) error { return errorOf(client.GetConclusion(ctx, id, "conclusion")) }},
		{"GetConclusion/conclusionID", func(id string) error { return errorOf(client.GetConclusion(ctx, "workspace", id)) }},
		{"ListConclusions/workspaceID", func(id string) error { return errorOf(client.ListConclusions(ctx, id, nil, nil)) }},
		{"QueryConclusions/workspaceID", func(id string) error { return errorOf(client.QueryConclusions(ctx, id, query)) }},

		{"ListDreams/workspaceID", func(id string) error { return errorOf(client.ListDreams(ctx, id, nil, nil)) }},
		{"GetDream/workspaceID", func(id string) error { return errorOf(client.GetDream(ctx, id, "run")) }},
		{"GetDream/runID", func(id string) error { return errorOf(client.GetDream(ctx, "workspace", id)) }},

		{"GetOrCreateScope/workspaceID", func(id string) error { return errorOf(client.GetOrCreateScope(ctx, id, scope)) }},
		{"GetScopes/workspaceID", func(id string) error { return errorOf(client.GetScopes(ctx, id, nil, nil)) }},
		{"GetScope/workspaceID", func(id string) error { return errorOf(client.GetScope(ctx, id, "team")) }},
		{"GetScope/scopeID", func(id string) error { return errorOf(client.GetScope(ctx, "workspace", id)) }},
		{"GetScopeStatus/workspaceID", func(id string) error { return errorOf(client.GetScopeStatus(ctx, id, "team")) }},
		{"GetScopeStatus/scopeID", func(id string) error { return errorOf(client.GetScopeStatus(ctx, "workspace", id)) }},
		{"AddSessionsToScope/workspaceID", func(id string) error { return client.AddSessionsToScope(ctx, id, "team", scopeAdd) }},
		{"AddSessionsToScope/scopeID", func(id string) error { return client.AddSessionsToScope(ctx, "workspace", id, scopeAdd) }},
		{"GetScopeSessions/workspaceID", func(id string) error { return errorOf(client.GetScopeSessions(ctx, id, "team", nil)) }},
		{"GetScopeSessions/scopeID", func(id string) error { return errorOf(client.GetScopeSessions(ctx, "workspace", id, nil)) }},
		{"RemoveSessionsFromScope/workspaceID", func(id string) error {
			return client.RemoveSessionsFromScope(ctx, id, "team", scopeRemove)
		}},
		{"RemoveSessionsFromScope/scopeID", func(id string) error {
			return client.RemoveSessionsFromScope(ctx, "workspace", id, scopeRemove)
		}},
		{"RemoveSessionFromScope/workspaceID", func(id string) error {
			return client.RemoveSessionFromScope(ctx, id, "team", "session")
		}},
		{"RemoveSessionFromScope/scopeID", func(id string) error {
			return client.RemoveSessionFromScope(ctx, "workspace", id, "session")
		}},
		{"RemoveSessionFromScope/sessionID", func(id string) error {
			return client.RemoveSessionFromScope(ctx, "workspace", "team", id)
		}},

		{"CreateMessagesForSession/workspaceID", func(id string) error {
			return errorOf(client.CreateMessagesForSession(ctx, id, "session", messages))
		}},
		{"CreateMessagesForSession/sessionID", func(id string) error {
			return errorOf(client.CreateMessagesForSession(ctx, "workspace", id, messages))
		}},
		{"CreateMessagesWithFile/workspaceID", func(id string) error {
			return errorOf(client.CreateMessagesWithFile(ctx, id, "session", upload))
		}},
		{"CreateMessagesWithFile/sessionID", func(id string) error {
			return errorOf(client.CreateMessagesWithFile(ctx, "workspace", id, upload))
		}},
		{"GetMessage/workspaceID", func(id string) error { return errorOf(client.GetMessage(ctx, id, "session", "message")) }},
		{"GetMessage/sessionID", func(id string) error { return errorOf(client.GetMessage(ctx, "workspace", id, "message")) }},
		{"GetMessage/messageID", func(id string) error { return errorOf(client.GetMessage(ctx, "workspace", "session", id)) }},
		{"GetMessages/workspaceID", func(id string) error { return errorOf(client.GetMessages(ctx, id, "session", nil, nil)) }},
		{"GetMessages/sessionID", func(id string) error { return errorOf(client.GetMessages(ctx, "workspace", id, nil, nil)) }},
		{"UpdateMessage/workspaceID", func(id string) error {
			return errorOf(client.UpdateMessage(ctx, id, "session", "message", MessageUpdate{}))
		}},
		{"UpdateMessage/sessionID", func(id string) error {
			return errorOf(client.UpdateMessage(ctx, "workspace", id, "message", MessageUpdate{}))
		}},
		{"UpdateMessage/messageID", func(id string) error {
			return errorOf(client.UpdateMessage(ctx, "workspace", "session", id, MessageUpdate{}))
		}},

		{"GetOrCreatePeer/workspaceID", func(id string) error { return errorOf(client.GetOrCreatePeer(ctx, id, peer)) }},
		{"GetAllPeers/workspaceID", func(id string) error { return errorOf(client.GetAllPeers(ctx, id, nil, nil)) }},
		{"UpdatePeer/workspaceID", func(id string) error { return errorOf(client.UpdatePeer(ctx, id, "alice", PeerUpdate{})) }},
		{"UpdatePeer/peerID", func(id string) error { return errorOf(client.UpdatePeer(ctx, "workspace", id, PeerUpdate{})) }},
		{"GetRepresentation/workspaceID", func(id string) error {
			return errorOf(client.GetRepresentation(ctx, id, "alice", PeerRepresentationGet{}))
		}},
		{"GetRepresentation/peerID", func(id string) error {
			return errorOf(client.GetRepresentation(ctx, "workspace", id, PeerRepresentationGet{}))
		}},
		{"GetPeerCard/workspaceID", func(id string) error { return errorOf(client.GetPeerCard(ctx, id, "alice", nil)) }},
		{"GetPeerCard/peerID", func(id string) error { return errorOf(client.GetPeerCard(ctx, "workspace", id, nil)) }},
		{"SetPeerCard/workspaceID", func(id string) error { return errorOf(client.SetPeerCard(ctx, id, "alice", card, nil)) }},
		{"SetPeerCard/peerID", func(id string) error { return errorOf(client.SetPeerCard(ctx, "workspace", id, card, nil)) }},
		{"GetPeerContext/workspaceID", func(id string) error { return errorOf(client.GetPeerContext(ctx, id, "alice", nil)) }},
		{"GetPeerContext/peerID", func(id string) error { return errorOf(client.GetPeerContext(ctx, "workspace", id, nil)) }},
		{"GetSessionsForPeer/workspaceID", func(id string) error { return errorOf(client.GetSessionsForPeer(ctx, id, "alice", nil, nil)) }},
		{"GetSessionsForPeer/peerID", func(id string) error { return errorOf(client.GetSessionsForPeer(ctx, "workspace", id, nil, nil)) }},
		{"SearchPeer/workspaceID", func(id string) error { return errorOf(client.SearchPeer(ctx, id, "alice", search)) }},
		{"SearchPeer/peerID", func(id string) error { return errorOf(client.SearchPeer(ctx, "workspace", id, search)) }},
		{"Chat/workspaceID", func(id string) error { return errorOf(client.Chat(ctx, id, "alice", dialectic)) }},
		{"Chat/peerID", func(id string) error { return errorOf(client.Chat(ctx, "workspace", id, dialectic)) }},
		{"ChatStream/workspaceID", func(id string) error { return chatStream(id, "alice") }},
		{"ChatStream/peerID", func(id string) error { return chatStream("workspace", id) }},

		{"GetOrCreateSession/workspaceID", func(id string) error { return errorOf(client.GetOrCreateSession(ctx, id, session)) }},
		{"GetSessions/workspaceID", func(id string) error { return errorOf(client.GetSessions(ctx, id, nil, nil)) }},
		{"UpdateSession/workspaceID", func(id string) error { return errorOf(client.UpdateSession(ctx, id, "session", SessionUpdate{})) }},
		{"UpdateSession/sessionID", func(id string) error { return errorOf(client.UpdateSession(ctx, "workspace", id, SessionUpdate{})) }},
		{"DeleteSession/workspaceID", func(id string) error { return client.DeleteSession(ctx, id, "session") }},
		{"DeleteSession/sessionID", func(id string) error { return client.DeleteSession(ctx, "workspace", id) }},
		{"CloneSession/workspaceID", func(id string) error { return errorOf(client.CloneSession(ctx, id, "session", nil)) }},
		{"CloneSession/sessionID", func(id string) error { return errorOf(client.CloneSession(ctx, "workspace", id, nil)) }},
		{"GetSessionPeers/workspaceID", func(id string) error { return errorOf(client.GetSessionPeers(ctx, id, "session", nil)) }},
		{"GetSessionPeers/sessionID", func(id string) error { return errorOf(client.GetSessionPeers(ctx, "workspace", id, nil)) }},
		{"SetSessionPeers/workspaceID", func(id string) error { return errorOf(client.SetSessionPeers(ctx, id, "session", peers)) }},
		{"SetSessionPeers/sessionID", func(id string) error { return errorOf(client.SetSessionPeers(ctx, "workspace", id, peers)) }},
		{"AddPeersToSession/workspaceID", func(id string) error { return errorOf(client.AddPeersToSession(ctx, id, "session", peers)) }},
		{"AddPeersToSession/sessionID", func(id string) error { return errorOf(client.AddPeersToSession(ctx, "workspace", id, peers)) }},
		{"RemovePeersFromSession/workspaceID", func(id string) error {
			return errorOf(client.RemovePeersFromSession(ctx, id, "session", []string{"alice"}))
		}},
		{"RemovePeersFromSession/sessionID", func(id string) error {
			return errorOf(client.RemovePeersFromSession(ctx, "workspace", id, []string{"alice"}))
		}},
		{"GetSessionContext/workspaceID", func(id string) error { return errorOf(client.GetSessionContext(ctx, id, "session", nil)) }},
		{"GetSessionContext/sessionID", func(id string) error { return errorOf(client.GetSessionContext(ctx, "workspace", id, nil)) }},
		{"GetSessionSummaries/workspaceID", func(id string) error { return errorOf(client.GetSessionSummaries(ctx, id, "session")) }},
		{"GetSessionSummaries/sessionID", func(id string) error { return errorOf(client.GetSessionSummaries(ctx, "workspace", id)) }},
		{"SearchSession/workspaceID", func(id string) error { return errorOf(client.SearchSession(ctx, id, "session", search)) }},
		{"SearchSession/sessionID", func(id string) error { return errorOf(client.SearchSession(ctx, "workspace", id, search)) }},
		{"GetPeerConfig/workspaceID", func(id string) error { return errorOf(client.GetPeerConfig(ctx, id, "session", "alice")) }},
		{"GetPeerConfig/sessionID", func(id string) error { return errorOf(client.GetPeerConfig(ctx, "workspace", id, "alice")) }},
		{"GetPeerConfig/peerID", func(id string) error { return errorOf(client.GetPeerConfig(ctx, "workspace", "session", id)) }},
		{"SetPeerConfig/workspaceID", func(id string) error {
			return client.SetPeerConfig(ctx, id, "session", "alice", SessionPeerConfig{})
		}},
		{"SetPeerConfig/sessionID", func(id string) error {
			return client.SetPeerConfig(ctx, "workspace", id, "alice", SessionPeerConfig{})
		}},
		{"SetPeerConfig/peerID", func(id string) error {
			return client.SetPeerConfig(ctx, "workspace", "session", id, SessionPeerConfig{})
		}},

		{"GetOrCreateWebhookEndpoint/workspaceID", func(id string) error { return errorOf(client.GetOrCreateWebhookEndpoint(ctx, id, webhook)) }},
		{"ListWebhookEndpoints/workspaceID", func(id string) error { return errorOf(client.ListWebhookEndpoints(ctx, id, nil)) }},
		{"DeleteWebhookEndpoint/workspaceID", func(id string) error { return client.DeleteWebhookEndpoint(ctx, id, "endpoint") }},
		{"DeleteWebhookEndpoint/endpointID", func(id string) error { return client.DeleteWebhookEndpoint(ctx, "workspace", id) }},
		{"TestEmit/workspaceID", func(id string) error { return client.TestEmit(ctx, id) }},

		{"UpdateWorkspace/workspaceID", func(id string) error { return errorOf(client.UpdateWorkspace(ctx, id, UpdateWorkspaceRequest{})) }},
		{"DeleteWorkspace/workspaceID", func(id string) error { return client.DeleteWorkspace(ctx, id) }},
		{"GetQueueStatus/workspaceID", func(id string) error { return errorOf(client.GetQueueStatus(ctx, id, nil, nil, nil)) }},
		{"ScheduleDream/workspaceID", func(id string) error { return client.ScheduleDream(ctx, id, dream) }},
		{"SearchWorkspace/workspaceID", func(id string) error { return errorOf(client.SearchWorkspace(ctx, id, wsSearch)) }},
		{"WorkspaceChat/workspaceID", func(id string) error { return errorOf(client.WorkspaceChat(ctx, id, chatOptions)) }},
		{"WorkspaceChatStream/workspaceID", workspaceChatStream},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requests.Store(0)
			if err := tt.call("../x"); !errors.Is(err, ErrInvalidID) {
				t.Errorf("error = %v, want %v", err, ErrInvalidID)
			}
			if sent := requests.Load(); sent != 0 {
				t.Errorf("%d requests sent for the unsafe ID, want none", sent)
			}

			// The other arguments are valid: only the ID kept the request from being sent.
			requests.Store(0)
			_ = tt.call("safe")
			if sent := requests.Load(); sent != 1 {
				t.Errorf("%d requests sent for a safe ID, want 1", sent)
			}
		})
	}
}

// errorOf returns the error of a method that also returns a result.
func errorOf[T any](_ T, err error) error {
	return err
}

// JoinPath appears nowhere in the package but in endpoint, so no method can build a
// URL without its checks. Regression test: each method called JoinPath itself, with
// the IDs it was given.
func TestOnlyEndpointJoinsPaths(t *testing.T) {
	names, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("failed to list the package's files: %v", err)
	}
	var joins int
	files := token.NewFileSet()
	for _, name := range names {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(files, name, nil, 0)
		if err != nil {
			t.Fatalf("failed to parse %s: %v", name, err)
		}
		for _, decl := range file.Decls {
			function, isFunction := decl.(*ast.FuncDecl)
			inEndpoint := isFunction && function.Recv != nil && function.Name.Name == "endpoint"
			ast.Inspect(decl, func(node ast.Node) bool {
				if ident, isIdent := node.(*ast.Ident); isIdent && ident.Name == "JoinPath" {
					joins++
					if !inEndpoint {
						t.Errorf("%s: JoinPath used outside endpoint, build the URL with c.endpoint", files.Position(ident.Pos()))
					}
				}
				return true
			})
		}
	}
	if joins == 0 {
		t.Error("JoinPath not found in the package, endpoint must call it")
	}
}
