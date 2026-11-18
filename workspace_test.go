package honcho

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"
)

// fullConfiguration is the full configuration a Honcho server stores and
// returns, for workspaces and sessions alike.
const fullConfiguration = `{
  "custom_instructions": "Shared instructions.",
  "reasoning": {"enabled": true, "custom_instructions": "Reasoning instructions."},
  "peer_card": {"use": true, "create": true, "custom_instructions": "Card instructions."},
  "summary": {"enabled": true, "messages_per_short_summary": 10, "messages_per_long_summary": 20, "custom_instructions": "Summary instructions."},
  "dream": {"enabled": true, "custom_instructions": "Dream instructions."},
  "dialectic": {"custom_instructions": "Dialectic instructions."}
}`

// checkFullConfiguration reports an error unless the configuration in the body
// of an update request is fullConfiguration, compared as decoded JSON.
func checkFullConfiguration(t *testing.T, body []byte) {
	t.Helper()
	var sent struct {
		Configuration any `json:"configuration"`
	}
	if err := json.Unmarshal(body, &sent); err != nil {
		t.Fatalf("failed to decode the update request's body %q: %v", body, err)
	}
	var full any
	if err := json.Unmarshal([]byte(fullConfiguration), &full); err != nil {
		t.Fatalf("failed to decode the full configuration: %v", err)
	}
	if !reflect.DeepEqual(sent.Configuration, full) {
		got, _ := json.Marshal(sent.Configuration)
		want, _ := json.Marshal(full)
		t.Errorf("configuration sent = %s, want %s", got, want)
	}
}

// A workspace's configuration, read and sent back as it is, reaches Honcho with
// every section and value Honcho returned. Regression test:
// WorkspaceConfiguration lacked the dialectic section and the custom
// instructions outside the reasoning section, so a read dropped them and writing
// the configuration back erased them, as an update replaces the whole stored
// configuration.
func TestWorkspaceConfigurationRoundTrip(t *testing.T) {
	var update []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			var err error
			if update, err = io.ReadAll(r.Body); err != nil {
				t.Errorf("failed to read the request body: %v", err)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":"workspace","created_at":"2026-10-30T00:00:00Z","configuration":`+fullConfiguration+`}`)
	}))
	defer server.Close()
	u, _ := url.Parse(server.URL)
	client := New(&Options{BaseURL: u})

	workspace, err := client.GetOrCreateWorkspace(t.Context(), CreateWorkspaceRequest{ID: "workspace"})
	if err != nil {
		t.Fatalf("GetOrCreateWorkspace: %v", err)
	}
	if _, err = client.UpdateWorkspace(t.Context(), "workspace", UpdateWorkspaceRequest{Configuration: workspace.Configuration}); err != nil {
		t.Fatalf("UpdateWorkspace: %v", err)
	}
	// Close waits for the handler, so what it recorded can be read.
	server.Close()

	checkFullConfiguration(t, update)
}

const (
	// workspaceChatQuery is the query the workspace chat tests send.
	workspaceChatQuery = "What is alice working on?"
	// workspaceChatPath is the path of the workspace chat of the workspace "workspace".
	workspaceChatPath = "/v3/workspaces/workspace/chat"
	// workspaceChatEvidence is the answer of a real Honcho server to a workspace chat
	// asking for evidence.
	workspaceChatEvidence = `{"content":"alice is designing a bridge.","evidence":{"conclusions":[{"id":"IDbKmwkeILusValV2GSbi","level":"explicit","content":"alice is designing a bridge.","created_at":"2026-11-18T15:09:06Z","session_id":"work","observer_id":"alice","observed_id":"alice","source_ids":[]}],"messages":[{"id":"Xv27Pg3BTadRElK_LnbyJ","session_id":"work","peer_id":"alice","created_at":"2026-11-18T15:09:06.775938Z"}],"tool_calls":[{"tool_name":"search","tool_input":{"query":"job"}}],"reasoning_trace_id":null}}`
)

// receivedChatRequest is what a test server recorded of a request it received.
type receivedChatRequest struct {
	method, path, accept string
	body                 []byte
}

// workspaceChatServer returns a client of a server answering every request with the
// given status, content type and body, and a function closing the server and returning
// the requests it received.
func workspaceChatServer(t *testing.T, status int, contentType, response string) (client *Client, received func() []receivedChatRequest) {
	t.Helper()
	var requests []receivedChatRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("failed to read the request body: %v", err)
		}
		requests = append(requests, receivedChatRequest{r.Method, r.URL.Path, r.Header.Get("Accept"), body})
		w.Header().Set("Content-Type", contentType)
		w.WriteHeader(status)
		fmt.Fprint(w, response)
	}))
	t.Cleanup(server.Close)
	u, _ := url.Parse(server.URL)
	return New(&Options{BaseURL: u}), func() []receivedChatRequest {
		// Close waits for the handler, so what it recorded can be read.
		server.Close()
		return requests
	}
}

// checkWorkspaceChatRequest reports an error unless the requests are a single POST to
// the workspace chat path, accepting the given content type, whose body holds exactly
// the given fields, compared as decoded JSON.
func checkWorkspaceChatRequest(t *testing.T, requests []receivedChatRequest, accept, body string) {
	t.Helper()
	if len(requests) != 1 {
		t.Fatalf("%d requests sent, want 1", len(requests))
	}
	request := requests[0]
	if request.method != http.MethodPost {
		t.Errorf("method = %s, want %s", request.method, http.MethodPost)
	}
	if request.path != workspaceChatPath {
		t.Errorf("path = %s, want %s", request.path, workspaceChatPath)
	}
	if request.accept != accept {
		t.Errorf("Accept = %q, want %q", request.accept, accept)
	}
	var got, want any
	if err := json.Unmarshal(request.body, &got); err != nil {
		t.Fatalf("failed to decode the request body %q: %v", request.body, err)
	}
	if err := json.Unmarshal([]byte(body), &want); err != nil {
		t.Fatalf("failed to decode the expected body %q: %v", body, err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("body = %s, want %s", request.body, body)
	}
}

// collectWorkspaceChatStream ranges over a WorkspaceChatStream to its end and returns
// the chunks and the errors it yielded. An error must come alone, and last.
func collectWorkspaceChatStream(t *testing.T, client *Client, req WorkspaceChatOptions) (chunks []string, errs []error) {
	t.Helper()
	for chunk, err := range client.WorkspaceChatStream(t.Context(), "workspace", req) {
		if len(errs) > 0 {
			t.Errorf("yielded (%q, %v) after the error %v", chunk, err, errs[0])
		}
		if err != nil {
			if chunk != "" {
				t.Errorf("error %v yielded with the chunk %q", err, chunk)
			}
			errs = append(errs, err)
			continue
		}
		chunks = append(chunks, chunk)
	}
	return
}

// WorkspaceChat sends a POST to the workspace's chat path, holding exactly the fields
// the caller set: a field left unset is not sent, so the server applies its default,
// and stream is never sent.
func TestWorkspaceChatRequest(t *testing.T) {
	sessionID := "work"
	tests := []struct {
		name string
		req  WorkspaceChatOptions
		want string
	}{
		{
			name: "query only",
			req:  WorkspaceChatOptions{Query: workspaceChatQuery},
			want: `{"query":"What is alice working on?"}`,
		},
		{
			name: "session",
			req:  WorkspaceChatOptions{Query: workspaceChatQuery, SessionID: &sessionID},
			want: `{"query":"What is alice working on?","session_id":"work"}`,
		},
		{
			name: "every other field",
			req: WorkspaceChatOptions{
				Query:           workspaceChatQuery,
				Scope:           []string{"team", "project"},
				ReasoningLevel:  ReasoningLevelHigh,
				ResponseFormat:  json.RawMessage(`{"type":"object","properties":{"names":{"type":"array"}}}`),
				IncludeEvidence: true,
			},
			want: `{
				"query": "What is alice working on?",
				"scope": ["team", "project"],
				"reasoning_level": "high",
				"response_format": {"type": "object", "properties": {"names": {"type": "array"}}},
				"include_evidence": true
			}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, received := workspaceChatServer(t, http.StatusOK, "application/json", `{"content":"alice is designing a bridge.","evidence":null}`)

			if _, err := client.WorkspaceChat(t.Context(), "workspace", tt.req); err != nil {
				t.Fatalf("WorkspaceChat() error = %v, want nil", err)
			}

			checkWorkspaceChatRequest(t, received(), "application/json", tt.want)
		})
	}
}

// WorkspaceChat decodes the answer and every field of its evidence.
func TestWorkspaceChatEvidence(t *testing.T) {
	client, _ := workspaceChatServer(t, http.StatusOK, "application/json", workspaceChatEvidence)

	result, err := client.WorkspaceChat(t.Context(), "workspace", WorkspaceChatOptions{Query: workspaceChatQuery, IncludeEvidence: true})
	if err != nil {
		t.Fatalf("WorkspaceChat() error = %v, want nil", err)
	}

	if want := "alice is designing a bridge."; result.Content == nil || *result.Content != want {
		t.Errorf("Content = %v, want %q", result.Content, want)
	}
	evidence := result.Evidence
	if evidence == nil {
		t.Fatal("Evidence = nil, want the evidence of the answer")
	}
	if len(evidence.Conclusions) != 1 || len(evidence.Messages) != 1 || len(evidence.ToolCalls) != 1 {
		t.Fatalf("evidence holds %d conclusions, %d messages and %d tool calls, want one of each",
			len(evidence.Conclusions), len(evidence.Messages), len(evidence.ToolCalls))
	}
	conclusion, message, toolCall := evidence.Conclusions[0], evidence.Messages[0], evidence.ToolCalls[0]
	if conclusion.SessionID == nil {
		t.Fatal("conclusion SessionID = nil, want the session of the conclusion")
	}
	for _, check := range []struct {
		name      string
		got, want any
	}{
		{"conclusion ID", conclusion.ID, "IDbKmwkeILusValV2GSbi"},
		{"conclusion Level", conclusion.Level, ConclusionLevelExplicit},
		{"conclusion Content", conclusion.Content, "alice is designing a bridge."},
		{"conclusion CreatedAt", conclusion.CreatedAt.UTC().Format(time.RFC3339Nano), "2026-11-18T15:09:06Z"},
		{"conclusion SessionID", *conclusion.SessionID, "work"},
		{"conclusion ObserverID", conclusion.ObserverID, "alice"},
		{"conclusion ObservedID", conclusion.ObservedID, "alice"},
		{"conclusion SourceIDs", conclusion.SourceIDs, []string{}},
		{"message ID", message.ID, "Xv27Pg3BTadRElK_LnbyJ"},
		{"message SessionID", message.SessionID, "work"},
		{"message PeerID", message.PeerID, "alice"},
		{"message CreatedAt", message.CreatedAt.UTC().Format(time.RFC3339Nano), "2026-11-18T15:09:06.775938Z"},
		{"tool call ToolName", toolCall.ToolName, "search"},
		{"tool call ToolInput", string(toolCall.ToolInput), `{"query":"job"}`},
	} {
		if !reflect.DeepEqual(check.got, check.want) {
			t.Errorf("%s = %#v, want %#v", check.name, check.got, check.want)
		}
	}
	if evidence.ReasoningTraceID != nil {
		t.Errorf("ReasoningTraceID = %q, want nil", *evidence.ReasoningTraceID)
	}
}

// The nullable fields of the evidence are nil when null and hold their value otherwise.
func TestWorkspaceChatEvidenceNullableFields(t *testing.T) {
	client, _ := workspaceChatServer(t, http.StatusOK, "application/json", `{"content":"alice is designing a bridge.","evidence":{
		"conclusions":[{"id":"conclusion","level":"deductive","content":"alice is an engineer.","created_at":"2026-11-18T15:09:06Z","session_id":null,"observer_id":"alice","observed_id":"alice","source_ids":["premise"]}],
		"messages":[],"tool_calls":[],"reasoning_trace_id":"trace"
	}}`)

	result, err := client.WorkspaceChat(t.Context(), "workspace", WorkspaceChatOptions{Query: workspaceChatQuery, IncludeEvidence: true})
	if err != nil {
		t.Fatalf("WorkspaceChat() error = %v, want nil", err)
	}

	if result.Evidence == nil || len(result.Evidence.Conclusions) != 1 {
		t.Fatalf("Evidence = %+v, want one conclusion", result.Evidence)
	}
	if got := result.Evidence.Conclusions[0].SessionID; got != nil {
		t.Errorf("conclusion SessionID = %q, want nil", *got)
	}
	if got := result.Evidence.ReasoningTraceID; got == nil || *got != "trace" {
		t.Errorf("ReasoningTraceID = %v, want %q", got, "trace")
	}
}

// An answer without evidence decodes to a nil Evidence, and an answer in a requested
// response format stays the JSON string the server sent.
func TestWorkspaceChatNoEvidence(t *testing.T) {
	tests := []struct{ name, response, want string }{
		{name: "plain answer", response: `{"content":"alice is designing a bridge.","evidence":null}`, want: "alice is designing a bridge."},
		{name: "response format", response: `{"content":"{\"names\":[\"alice\",\"bob\"]}","evidence":null}`, want: `{"names":["alice","bob"]}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, _ := workspaceChatServer(t, http.StatusOK, "application/json", tt.response)

			result, err := client.WorkspaceChat(t.Context(), "workspace", WorkspaceChatOptions{Query: workspaceChatQuery})
			if err != nil {
				t.Fatalf("WorkspaceChat() error = %v, want nil", err)
			}

			if result.Content == nil || *result.Content != tt.want {
				t.Errorf("Content = %v, want %q", result.Content, tt.want)
			}
			if result.Evidence != nil {
				t.Errorf("Evidence = %+v, want nil", result.Evidence)
			}
		})
	}
}

// WorkspaceChatStream sends the fields the caller set as a POST to the workspace's chat
// path, asking for a stream, and yields the non-empty chunks in order up to the final
// frame, which can carry evidence.
func TestWorkspaceChatStream(t *testing.T) {
	stream := deltaFrame("alice is ") + deltaFrame("") + deltaFrame("designing a bridge.") +
		`data: {"done": true, "evidence": {"conclusions": [], "messages": [], "tool_calls": [], "reasoning_trace_id": null}}` + "\n\n"
	client, received := workspaceChatServer(t, http.StatusOK, eventStream, stream)

	chunks, errs := collectWorkspaceChatStream(t, client, WorkspaceChatOptions{Query: workspaceChatQuery, Scope: []string{"team"}})

	if len(errs) != 0 {
		t.Errorf("errors = %v, want none", errs)
	}
	if want := []string{"alice is ", "designing a bridge."}; !slices.Equal(chunks, want) {
		t.Errorf("chunks = %q, want %q", chunks, want)
	}
	requests := received()
	checkWorkspaceChatRequest(t, requests, "text/event-stream", `{"query":"What is alice working on?","scope":["team"],"stream":true}`)
	if body := string(requests[0].body); !strings.Contains(body, `"stream":true`) {
		t.Errorf("body = %s, want it to hold \"stream\":true", body)
	}
}

// A stream that ends without its final frame is an error, never a complete answer: the
// chunks read so far are yielded, then exactly one error.
func TestWorkspaceChatStreamIncomplete(t *testing.T) {
	client := streamingClient(t, deltaFrame("alice is ")+deltaFrame("designing"))

	chunks, errs := collectWorkspaceChatStream(t, client, WorkspaceChatOptions{Query: workspaceChatQuery})

	if want := []string{"alice is ", "designing"}; !slices.Equal(chunks, want) {
		t.Errorf("chunks = %q, want %q", chunks, want)
	}
	if len(errs) != 1 {
		t.Errorf("errors = %v, want exactly one", errs)
	}
}

// WorkspaceChatStream has no way to return evidence: it refuses to be asked for it with
// a single error pointing to WorkspaceChat, and sends no request.
func TestWorkspaceChatStreamRefusesEvidence(t *testing.T) {
	client, received := workspaceChatServer(t, http.StatusOK, eventStream, deltaFrame("alice")+finalFrame)

	chunks, errs := collectWorkspaceChatStream(t, client, WorkspaceChatOptions{Query: workspaceChatQuery, IncludeEvidence: true})

	if len(chunks) != 0 {
		t.Errorf("chunks = %q, want none", chunks)
	}
	if len(errs) != 1 || !strings.Contains(errs[0].Error(), "use WorkspaceChat instead") {
		t.Errorf("errors = %v, want a single error naming WorkspaceChat", errs)
	}
	if requests := received(); len(requests) != 0 {
		t.Errorf("%d requests sent, want none", len(requests))
	}
}

// A workspace chat query is accepted up to 10000 characters, however many bytes each
// takes, and a scope list from 1 to 100 scopes. A nil scope is no restriction, while a
// non-nil empty one is refused: left out of the request, it would allow everything.
func TestWorkspaceChatOptionsValidate(t *testing.T) {
	const (
		emptyScopeErr = "scope must hold at least one scope: leave it nil for no restriction"
		manyScopesErr = "scope must hold 100 scopes or less"
		longQueryErr  = "query must be 10000 characters or less"
	)
	tests := []struct {
		name    string
		request WorkspaceChatOptions
		errMsg  string
	}{
		{name: "empty query", request: WorkspaceChatOptions{}, errMsg: "query is required"},
		{name: "10000 2-byte characters", request: WorkspaceChatOptions{Query: strings.Repeat("é", 10000)}},
		{name: "10001 2-byte characters", request: WorkspaceChatOptions{Query: strings.Repeat("é", 10001)}, errMsg: longQueryErr},
		{name: "10000 4-byte characters", request: WorkspaceChatOptions{Query: strings.Repeat("😀", 10000)}},
		{name: "10001 4-byte characters", request: WorkspaceChatOptions{Query: strings.Repeat("😀", 10001)}, errMsg: longQueryErr},
		{name: "nil scope", request: WorkspaceChatOptions{Query: workspaceChatQuery}},
		{name: "empty scope", request: WorkspaceChatOptions{Query: workspaceChatQuery, Scope: []string{}}, errMsg: emptyScopeErr},
		{name: "1 scope", request: WorkspaceChatOptions{Query: workspaceChatQuery, Scope: []string{"team"}}},
		{name: "100 scopes", request: WorkspaceChatOptions{Query: workspaceChatQuery, Scope: slices.Repeat([]string{"team"}, 100)}},
		{name: "101 scopes", request: WorkspaceChatOptions{Query: workspaceChatQuery, Scope: slices.Repeat([]string{"team"}, 101)}, errMsg: manyScopesErr},
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

// workspaceChatMethods calls each workspace chat method and returns the one error it
// must end with.
var workspaceChatMethods = []struct {
	name string
	call func(t *testing.T, client *Client, workspaceID string, req WorkspaceChatOptions) error
}{
	{"WorkspaceChat", func(t *testing.T, client *Client, workspaceID string, req WorkspaceChatOptions) error {
		return errorOf(client.WorkspaceChat(t.Context(), workspaceID, req))
	}},
	{"WorkspaceChatStream", func(t *testing.T, client *Client, workspaceID string, req WorkspaceChatOptions) error {
		var errs []error
		for chunk, err := range client.WorkspaceChatStream(t.Context(), workspaceID, req) {
			if err == nil {
				t.Errorf("chunk %q yielded, want a single error", chunk)
				continue
			}
			errs = append(errs, err)
		}
		if len(errs) != 1 {
			t.Fatalf("errors = %v, want exactly one", errs)
		}
		return errs[0]
	}},
}

// Both workspace chat methods refuse an invalid request without sending it, above all
// a non-nil empty scope, which the server would never see and so could not refuse.
func TestWorkspaceChatInvalidRequest(t *testing.T) {
	tests := []struct {
		name        string
		workspaceID string
		req         WorkspaceChatOptions
		errMsg      string
	}{
		{name: "empty workspace ID", req: WorkspaceChatOptions{Query: workspaceChatQuery}, errMsg: "workspaceID is required"},
		{name: "empty query", workspaceID: "workspace", errMsg: "query is required"},
		{name: "empty scope", workspaceID: "workspace", req: WorkspaceChatOptions{Query: workspaceChatQuery, Scope: []string{}}, errMsg: "scope must hold at least one scope"},
		{name: "101 scopes", workspaceID: "workspace", req: WorkspaceChatOptions{Query: workspaceChatQuery, Scope: slices.Repeat([]string{"team"}, 101)}, errMsg: "scope must hold 100 scopes or less"},
	}
	for _, method := range workspaceChatMethods {
		for _, tt := range tests {
			t.Run(method.name+"/"+tt.name, func(t *testing.T) {
				client, received := workspaceChatServer(t, http.StatusOK, "application/json", `{"content":"alice"}`)

				err := method.call(t, client, tt.workspaceID, tt.req)

				if err == nil || !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("error = %v, want it to hold %q", err, tt.errMsg)
				}
				if requests := received(); len(requests) != 0 {
					t.Errorf("%d requests sent, want none", len(requests))
				}
			})
		}
	}
}

// The server decides that a scope and a session are mutually exclusive: the request is
// sent, and its text-form 422 reaches the caller as an *HTTPValidationError.
func TestWorkspaceChatValidationError(t *testing.T) {
	const detail = "`scope` and `session_id` are mutually exclusive"
	sessionID := "work"
	for _, method := range workspaceChatMethods {
		t.Run(method.name, func(t *testing.T) {
			client, received := workspaceChatServer(t, http.StatusUnprocessableEntity, "application/json", `{"detail":"`+detail+`"}`)

			err := method.call(t, client, "workspace", WorkspaceChatOptions{Query: workspaceChatQuery, SessionID: &sessionID, Scope: []string{"team"}})

			var valErr *HTTPValidationError
			if !errors.As(err, &valErr) || valErr.Message != detail {
				t.Errorf("error = %v, want an *HTTPValidationError holding %q", err, detail)
			}
			if requests := received(); len(requests) != 1 {
				t.Errorf("%d requests sent, want 1", len(requests))
			}
		})
	}
}

// A 404, the answer to an unknown scope or session, reaches the caller as an error
// holding the status and the body of the response.
func TestWorkspaceChatNotFound(t *testing.T) {
	const detail = "Scope team not found in workspace workspace"
	for _, method := range workspaceChatMethods {
		t.Run(method.name, func(t *testing.T) {
			client, _ := workspaceChatServer(t, http.StatusNotFound, "application/json", `{"detail":"`+detail+`"}`)

			err := method.call(t, client, "workspace", WorkspaceChatOptions{Query: workspaceChatQuery, Scope: []string{"team"}})

			if err == nil {
				t.Fatal("error = nil, want an error")
			}
			for _, want := range []string{"404 Not Found", detail} {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("error = %v, want it to hold %q", err, want)
				}
			}
		})
	}
}

// GetOrCreateWorkspace, GetOrCreatePeer and GetOrCreateSession decode the immutable uid
// Honcho generates for the resource, from the answers of a real Honcho server.
func TestResourceUID(t *testing.T) {
	tests := []struct {
		name     string
		response string
		want     string
		get      func(t *testing.T, client *Client) (uid string, err error)
	}{
		{
			name:     "GetOrCreateWorkspace",
			response: `{"id":"w","uid":"46xqSOtyM08WIPPJQM5_6","metadata":{},"configuration":{},"created_at":"2026-11-18T16:09:28.620049Z"}`,
			want:     "46xqSOtyM08WIPPJQM5_6",
			get: func(t *testing.T, client *Client) (uid string, err error) {
				workspace, err := client.GetOrCreateWorkspace(t.Context(), CreateWorkspaceRequest{ID: "w"})
				if err != nil {
					return
				}
				uid = workspace.UID
				return
			},
		},
		{
			name:     "GetOrCreatePeer",
			response: `{"id":"alice","uid":"yTkg0JNCYKPlmpuCQAfqn","workspace_id":"w","created_at":"2026-11-18T16:09:28.620049Z","metadata":{},"configuration":{}}`,
			want:     "yTkg0JNCYKPlmpuCQAfqn",
			get: func(t *testing.T, client *Client) (uid string, err error) {
				peer, err := client.GetOrCreatePeer(t.Context(), "w", PeerCreate{ID: "alice"})
				if err != nil {
					return
				}
				uid = peer.UID
				return
			},
		},
		{
			name:     "GetOrCreateSession",
			response: `{"id":"s1","uid":"X7i6McQJTWP_F1wFzG3E2","is_active":true,"workspace_id":"w","metadata":{},"configuration":{},"created_at":"2026-11-18T16:09:28.620049Z"}`,
			want:     "X7i6McQJTWP_F1wFzG3E2",
			get: func(t *testing.T, client *Client) (uid string, err error) {
				session, err := client.GetOrCreateSession(t.Context(), "w", SessionCreate{ID: "s1"})
				if err != nil {
					return
				}
				uid = session.UID
				return
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, _ := scopeServer(t, http.StatusOK, tt.response)

			uid, err := tt.get(t, client)
			if err != nil {
				t.Fatalf("%s: %v", tt.name, err)
			}

			if uid != tt.want {
				t.Errorf("UID = %q, want %q", uid, tt.want)
			}
		})
	}
}

// ScheduleDream sends rebuild as true when Rebuild is set, and otherwise leaves it out of
// the body, so the server applies its default.
func TestScheduleDreamRebuild(t *testing.T) {
	tests := []struct {
		name string
		req  ScheduleDreamRequest
		want string
	}{
		{
			name: "rebuild set",
			req:  ScheduleDreamRequest{Observer: "alice", DreamType: DreamTypeCardRefresh, Rebuild: true},
			want: `{"observer":"alice","dream_type":"card_refresh","rebuild":true}`,
		},
		{
			name: "rebuild not set",
			req:  ScheduleDreamRequest{Observer: "alice", DreamType: DreamTypeCardRefresh},
			want: `{"observer":"alice","dream_type":"card_refresh"}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, received := scopeServer(t, http.StatusNoContent, "")

			if err := client.ScheduleDream(t.Context(), "workspace", tt.req); err != nil {
				t.Fatalf("ScheduleDream: %v", err)
			}

			checkScopeRequest(t, received(), http.MethodPost, "/v3/workspaces/workspace/schedule_dream", url.Values{}, tt.want)
		})
	}
}

// emptyPageJSON is a page holding no items, in the shape every paginated list shares.
const emptyPageJSON = `{"items":[],"total":0,"page":1,"size":50,"pages":0}`

// GetAllWorkspaces sends reverse=true when Reverse is set, and otherwise no reverse parameter
// at all, so the server keeps its default order, oldest workspace first.
func TestGetAllWorkspacesReverse(t *testing.T) {
	tests := []struct {
		name      string
		opts      *GetAllWorkspacesOptions
		wantQuery url.Values
	}{
		{name: "reverse set", opts: &GetAllWorkspacesOptions{Reverse: true}, wantQuery: url.Values{"reverse": {"true"}}},
		{name: "reverse not set", opts: &GetAllWorkspacesOptions{Page: 2}, wantQuery: url.Values{"page": {"2"}}},
		{name: "empty options", opts: &GetAllWorkspacesOptions{}, wantQuery: url.Values{}},
		{name: "nil options", wantQuery: url.Values{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, received := scopeServer(t, http.StatusOK, emptyPageJSON)

			if _, err := client.GetAllWorkspaces(t.Context(), WorkspaceGetRequest{}, tt.opts); err != nil {
				t.Fatalf("GetAllWorkspaces: %v", err)
			}

			checkScopeRequest(t, received(), http.MethodPost, "/v3/workspaces/list", tt.wantQuery, `{}`)
		})
	}
}

// SearchWorkspace sends a POST request to the workspace's search path with the fields set in the
// request. The scope is sent when it is set, an empty name included, and left out otherwise. A scope
// with a session_id filter is sent as it is: refusing it is up to the server.
func TestSearchWorkspaceRequest(t *testing.T) {
	scope, emptyScope := "team", ""
	tests := []struct {
		name     string
		req      WorkspaceMessageSearchOptions
		wantBody string
	}{
		{name: "query only", req: WorkspaceMessageSearchOptions{Query: "cats"}, wantBody: `{"query":"cats"}`},
		{name: "scope", req: WorkspaceMessageSearchOptions{Query: "cats", Scope: &scope}, wantBody: `{"query":"cats","scope":"team"}`},
		{name: "empty scope name", req: WorkspaceMessageSearchOptions{Query: "cats", Scope: &emptyScope}, wantBody: `{"query":"cats","scope":""}`},
		{
			name: "every field",
			req: WorkspaceMessageSearchOptions{
				Query:   "cats",
				Filters: map[string]any{"peer_id": "alice"},
				Limit:   5,
				Scope:   &scope,
			},
			wantBody: `{"query":"cats","filters":{"peer_id":"alice"},"limit":5,"scope":"team"}`,
		},
		{
			name:     "scope and a session_id filter",
			req:      WorkspaceMessageSearchOptions{Query: "cats", Filters: map[string]any{"session_id": "s1"}, Scope: &scope},
			wantBody: `{"query":"cats","filters":{"session_id":"s1"},"scope":"team"}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, received := scopeServer(t, http.StatusOK, `[]`)

			if _, err := client.SearchWorkspace(t.Context(), "workspace", tt.req); err != nil {
				t.Fatalf("SearchWorkspace: %v", err)
			}

			checkScopeRequest(t, received(), http.MethodPost, "/v3/workspaces/workspace/search", url.Values{}, tt.wantBody)
		})
	}
}
