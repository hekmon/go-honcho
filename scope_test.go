package honcho

import (
	"context"
	"encoding/json"
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

const (
	// scopeJSON is the answer of a real Honcho server to getting or creating a scope.
	scopeJSON = `{"id":"meta","metadata":{"owner":"alice"},"created_at":"2026-11-18T16:09:34.691032Z"}`
	// scopePageJSON is the answer of a real Honcho server to listing scopes with their
	// session count.
	scopePageJSON = `{"items":[{"id":"team","metadata":{},"created_at":"2026-11-18T16:09:34.673599Z","session_count":2}],"total":3,"page":1,"size":50,"pages":1}`
	// scopeStatusJSON is the answer of a real Honcho server for a scope with a completed
	// backfill.
	scopeStatusJSON = `{"backfill_status":{"s1":{"state":"completed","updated_at":"2026-11-18T16:09:34.808826+00:00","docs_copied":2}}}`
	// scopeStatusEmptyJSON is the answer of a real Honcho server for a scope without a
	// backfill.
	scopeStatusEmptyJSON = `{"backfill_status":{}}`
	// scopeSessionPageJSON is a page of sessions in the shape of upstream's Session schema.
	scopeSessionPageJSON = `{"items":[{"id":"s1","uid":"Xv27Pg3BTadRElK_LnbyJ","is_active":true,"workspace_id":"workspace","metadata":{"topic":"support"},"configuration":{},"created_at":"2026-11-18T16:09:34.673599Z"}],"total":1,"page":1,"size":50,"pages":1}`
)

// capturedScope returns what scopeJSON decodes to.
func capturedScope() *Scope {
	return &Scope{
		ID:        "meta",
		Metadata:  json.RawMessage(`{"owner":"alice"}`),
		CreatedAt: time.Date(2026, time.November, 18, 16, 9, 34, 691032000, time.UTC),
	}
}

// receivedScopeRequest is what a test server recorded of a request it received.
type receivedScopeRequest struct {
	method, path, contentType string
	query                     url.Values
	body                      []byte
}

// scopeServer returns a client of a server answering every request with the given status
// and JSON body, and a function closing the server and returning the requests it received.
func scopeServer(t *testing.T, status int, response string) (client *Client, received func() []receivedScopeRequest) {
	t.Helper()
	var requests []receivedScopeRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("failed to read the request body: %v", err)
		}
		requests = append(requests, receivedScopeRequest{r.Method, r.URL.Path, r.Header.Get("Content-Type"), r.URL.Query(), body})
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		fmt.Fprint(w, response)
	}))
	t.Cleanup(server.Close)
	u, _ := url.Parse(server.URL)
	return New(&Options{BaseURL: u}), func() []receivedScopeRequest {
		// Close waits for the handler, so what it recorded can be read.
		server.Close()
		return requests
	}
}

// checkScopeRequest reports an error unless the requests are a single one with the given
// method, path and query, whose body holds exactly the given JSON, compared as decoded
// JSON. An empty body stands for a request sent without any body.
func checkScopeRequest(t *testing.T, requests []receivedScopeRequest, method, path string, query url.Values, body string) {
	t.Helper()
	if len(requests) != 1 {
		t.Fatalf("%d requests sent, want 1", len(requests))
	}
	request := requests[0]
	if request.method != method {
		t.Errorf("method = %s, want %s", request.method, method)
	}
	if request.path != path {
		t.Errorf("path = %q, want %q", request.path, path)
	}
	if !reflect.DeepEqual(request.query, query) {
		t.Errorf("query = %v, want %v", request.query, query)
	}
	if body == "" {
		if len(request.body) != 0 {
			t.Errorf("body = %q, want it empty", request.body)
		}
		if request.contentType != "" {
			t.Errorf("Content-Type = %q, want none", request.contentType)
		}
		return
	}
	if want := "application/json; charset=utf-8"; request.contentType != want {
		t.Errorf("Content-Type = %q, want %q", request.contentType, want)
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

// GetOrCreateScope sends a POST request to the workspace's scopes path with exactly the
// fields set in the request, and decodes the scope whether Honcho created it, answering
// 201, or already held it, answering 200. Metadata left nil is not sent, so an existing
// scope keeps what it stores, while an empty object is sent, as it overwrites that.
func TestGetOrCreateScope(t *testing.T) {
	metadata := json.RawMessage(`{"owner":"alice"}`)
	tests := []struct {
		name     string
		status   int
		req      ScopeCreate
		wantBody string
	}{
		{
			name:     "created",
			status:   http.StatusCreated,
			req:      ScopeCreate{ID: "meta", Metadata: metadata},
			wantBody: `{"id":"meta","metadata":{"owner":"alice"}}`,
		},
		{
			name:     "already existing",
			status:   http.StatusOK,
			req:      ScopeCreate{ID: "meta", Metadata: metadata},
			wantBody: `{"id":"meta","metadata":{"owner":"alice"}}`,
		},
		{
			name:     "nil metadata",
			status:   http.StatusOK,
			req:      ScopeCreate{ID: "meta"},
			wantBody: `{"id":"meta"}`,
		},
		{
			name:     "empty metadata",
			status:   http.StatusOK,
			req:      ScopeCreate{ID: "meta", Metadata: json.RawMessage(`{}`)},
			wantBody: `{"id":"meta","metadata":{}}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, received := scopeServer(t, tt.status, scopeJSON)

			scope, err := client.GetOrCreateScope(t.Context(), "workspace", tt.req)
			if err != nil {
				t.Fatalf("GetOrCreateScope: %v", err)
			}

			checkScopeRequest(t, received(), http.MethodPost, "/v3/workspaces/workspace/scopes", url.Values{}, tt.wantBody)
			if want := capturedScope(); !reflect.DeepEqual(scope, want) {
				t.Errorf("scope = %+v, want %+v", scope, want)
			}
		})
	}
}

// GetScopes sends a POST request to the workspace's scopes list path, with the options set
// in the query and the filters in the body. A nil request is sent as a JSON null, which the
// route reads as no options, and nil options leave the query empty.
func TestGetScopesRequest(t *testing.T) {
	filters := map[string]any{"id": map[string]any{"icontains": "sup"}}
	const filtersBody = `{"filters":{"id":{"icontains":"sup"}}}`
	tests := []struct {
		name      string
		req       *ScopeGet
		opts      *GetScopesOptions
		wantQuery url.Values
		wantBody  string
	}{
		{name: "nil request and nil options", wantQuery: url.Values{}, wantBody: `null`},
		{
			name:      "empty request and empty options",
			req:       &ScopeGet{},
			opts:      &GetScopesOptions{},
			wantQuery: url.Values{},
			wantBody:  `{}`,
		},
		{name: "page only", opts: &GetScopesOptions{Page: 2}, wantQuery: url.Values{"page": {"2"}}, wantBody: `null`},
		{name: "size only", opts: &GetScopesOptions{Size: 20}, wantQuery: url.Values{"size": {"20"}}, wantBody: `null`},
		{name: "reverse only", opts: &GetScopesOptions{Reverse: true}, wantQuery: url.Values{"reverse": {"true"}}, wantBody: `null`},
		{
			name:      "session count only",
			opts:      &GetScopesOptions{IncludeSessionCount: true},
			wantQuery: url.Values{"include_session_count": {"true"}},
			wantBody:  `null`,
		},
		{name: "filters only", req: &ScopeGet{Filters: filters}, wantQuery: url.Values{}, wantBody: filtersBody},
		{
			name: "every field",
			req:  &ScopeGet{Filters: filters},
			opts: &GetScopesOptions{Page: 3, Size: 10, Reverse: true, IncludeSessionCount: true},
			wantQuery: url.Values{
				"page":                  {"3"},
				"size":                  {"10"},
				"reverse":               {"true"},
				"include_session_count": {"true"},
			},
			wantBody: filtersBody,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, received := scopeServer(t, http.StatusOK, scopePageJSON)

			if _, err := client.GetScopes(t.Context(), "workspace", tt.req, tt.opts); err != nil {
				t.Fatalf("GetScopes: %v", err)
			}

			checkScopeRequest(t, received(), http.MethodPost, "/v3/workspaces/workspace/scopes/list", tt.wantQuery, tt.wantBody)
		})
	}
}

// GetScopes decodes a page of scopes as a real Honcho server returns it. The session count
// reads as nil when the server leaves it out, and is told apart from a count of zero.
func TestGetScopesDecodesPage(t *testing.T) {
	createdAt := time.Date(2026, time.November, 18, 16, 9, 34, 673599000, time.UTC)
	tests := []struct {
		name         string
		body         string
		sessionCount *int
	}{
		{name: "captured page", body: scopePageJSON, sessionCount: pointerTo(2)},
		{
			name: "without session count",
			body: `{"items":[{"id":"team","metadata":{},"created_at":"2026-11-18T16:09:34.673599Z"}],"total":3,"page":1,"size":50,"pages":1}`,
		},
		{
			name:         "session count of zero",
			body:         `{"items":[{"id":"team","metadata":{},"created_at":"2026-11-18T16:09:34.673599Z","session_count":0}],"total":3,"page":1,"size":50,"pages":1}`,
			sessionCount: pointerTo(0),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, _ := scopeServer(t, http.StatusOK, tt.body)

			page, err := client.GetScopes(t.Context(), "workspace", nil, nil)
			if err != nil {
				t.Fatalf("GetScopes: %v", err)
			}

			want := &PageScope{
				Items: []Scope{{
					ID:           "team",
					Metadata:     json.RawMessage(`{}`),
					CreatedAt:    createdAt,
					SessionCount: tt.sessionCount,
				}},
				Total: 3,
				Page:  1,
				Size:  50,
				Pages: 1,
			}
			if !reflect.DeepEqual(page, want) {
				got, _ := json.Marshal(page)
				wanted, _ := json.Marshal(want)
				t.Errorf("page = %s, want %s", got, wanted)
			}
		})
	}
}

// GetScope sends a GET request without a body to the scope's path and decodes the scope as
// a real Honcho server returns it.
func TestGetScope(t *testing.T) {
	client, received := scopeServer(t, http.StatusOK, scopeJSON)

	scope, err := client.GetScope(t.Context(), "workspace", "meta")
	if err != nil {
		t.Fatalf("GetScope: %v", err)
	}

	checkScopeRequest(t, received(), http.MethodGet, "/v3/workspaces/workspace/scopes/meta", url.Values{}, "")
	if want := capturedScope(); !reflect.DeepEqual(scope, want) {
		t.Errorf("scope = %+v, want %+v", scope, want)
	}
}

// GetScopeStatus sends a GET request without a body to the scope's status path and decodes
// the state of each session's backfill. The number of documents copied reads as nil until
// a backfill completes, and is told apart from a backfill that copied nothing.
func TestGetScopeStatus(t *testing.T) {
	updatedAt := time.Date(2026, time.November, 18, 16, 9, 34, 808826000, time.UTC)
	tests := []struct {
		name string
		body string
		want map[string]ScopeBackfillStatus
	}{
		{
			name: "captured completed backfill",
			body: scopeStatusJSON,
			want: map[string]ScopeBackfillStatus{
				"s1": {State: ScopeBackfillStateCompleted, UpdatedAt: updatedAt, DocsCopied: pointerTo(2)},
			},
		},
		{name: "captured status without a backfill", body: scopeStatusEmptyJSON, want: map[string]ScopeBackfillStatus{}},
		{
			name: "completed backfill that copied nothing",
			body: `{"backfill_status":{"s1":{"state":"completed","updated_at":"2026-11-18T16:09:34.808826+00:00","docs_copied":0}}}`,
			want: map[string]ScopeBackfillStatus{
				"s1": {State: ScopeBackfillStateCompleted, UpdatedAt: updatedAt, DocsCopied: pointerTo(0)},
			},
		},
		{
			name: "pending and failed backfills",
			body: `{"backfill_status":{"s1":{"state":"pending","updated_at":"2026-11-18T16:09:34.808826+00:00"},"s2":{"state":"failed","updated_at":"2026-11-18T16:09:34.808826+00:00"}}}`,
			want: map[string]ScopeBackfillStatus{
				"s1": {State: ScopeBackfillStatePending, UpdatedAt: updatedAt},
				"s2": {State: ScopeBackfillStateFailed, UpdatedAt: updatedAt},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, received := scopeServer(t, http.StatusOK, tt.body)

			status, err := client.GetScopeStatus(t.Context(), "workspace", "team")
			if err != nil {
				t.Fatalf("GetScopeStatus: %v", err)
			}

			checkScopeRequest(t, received(), http.MethodGet, "/v3/workspaces/workspace/scopes/team/status", url.Values{}, "")
			if status.BackfillStatus == nil {
				t.Fatal("BackfillStatus = nil, want a map")
			}
			if len(status.BackfillStatus) != len(tt.want) {
				t.Errorf("BackfillStatus = %+v, want %+v", status.BackfillStatus, tt.want)
			}
			for sessionID, want := range tt.want {
				got, found := status.BackfillStatus[sessionID]
				if !found {
					t.Errorf("no status for the session %q", sessionID)
					continue
				}
				if got.State != want.State {
					t.Errorf("%s: State = %q, want %q", sessionID, got.State, want.State)
				}
				// Equal, as an offset of +00:00 does not decode to the UTC location.
				if !got.UpdatedAt.Equal(want.UpdatedAt) {
					t.Errorf("%s: UpdatedAt = %v, want %v", sessionID, got.UpdatedAt, want.UpdatedAt)
				}
				if !reflect.DeepEqual(got.DocsCopied, want.DocsCopied) {
					t.Errorf("%s: DocsCopied = %v, want %v", sessionID, got.DocsCopied, want.DocsCopied)
				}
			}
		})
	}
}

// Each backfill state constant holds the value Honcho sends for it.
func TestScopeBackfillStateValues(t *testing.T) {
	tests := []struct {
		got  ScopeBackfillState
		want string
	}{
		{ScopeBackfillStatePending, "pending"},
		{ScopeBackfillStateCompleted, "completed"},
		{ScopeBackfillStateFailed, "failed"},
	}
	for _, tt := range tests {
		if string(tt.got) != tt.want {
			t.Errorf("state = %q, want %q", tt.got, tt.want)
		}
	}
}

// AddSessionsToScope and RemoveSessionsFromScope send the session IDs in a POST request to
// their paths, RemoveSessionFromScope sends a DELETE request without a body to the member's
// path, and all three return nil on the 204 Honcho answers.
func TestScopeMembershipRequests(t *testing.T) {
	tests := []struct {
		name   string
		call   func(ctx context.Context, client *Client) error
		method string
		path   string
		body   string
	}{
		{
			name: "AddSessionsToScope",
			call: func(ctx context.Context, client *Client) error {
				return client.AddSessionsToScope(ctx, "workspace", "team", ScopeSessionsAdd{SessionIDs: []string{"s1", "s2"}})
			},
			method: http.MethodPost,
			path:   "/v3/workspaces/workspace/scopes/team/sessions",
			body:   `{"session_ids":["s1","s2"]}`,
		},
		{
			name: "RemoveSessionsFromScope",
			call: func(ctx context.Context, client *Client) error {
				return client.RemoveSessionsFromScope(ctx, "workspace", "team", ScopeSessionsRemove{SessionIDs: []string{"s1", "s2"}})
			},
			method: http.MethodPost,
			path:   "/v3/workspaces/workspace/scopes/team/sessions/remove",
			body:   `{"session_ids":["s1","s2"]}`,
		},
		{
			name: "RemoveSessionFromScope",
			call: func(ctx context.Context, client *Client) error {
				return client.RemoveSessionFromScope(ctx, "workspace", "team", "s1")
			},
			method: http.MethodDelete,
			path:   "/v3/workspaces/workspace/scopes/team/sessions/s1",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, received := scopeServer(t, http.StatusNoContent, "")

			if err := tt.call(t.Context(), client); err != nil {
				t.Errorf("error = %v, want nil", err)
			}

			checkScopeRequest(t, received(), tt.method, tt.path, url.Values{}, tt.body)
		})
	}
}

// GetScopeSessions sends a POST request without a body to the scope's sessions list path,
// with the options set in the query, and nil options leave the query empty.
func TestGetScopeSessionsRequest(t *testing.T) {
	tests := []struct {
		name      string
		opts      *GetScopeSessionsOptions
		wantQuery url.Values
	}{
		{name: "nil options", wantQuery: url.Values{}},
		{name: "empty options", opts: &GetScopeSessionsOptions{}, wantQuery: url.Values{}},
		{name: "page only", opts: &GetScopeSessionsOptions{Page: 2}, wantQuery: url.Values{"page": {"2"}}},
		{name: "size only", opts: &GetScopeSessionsOptions{Size: 20}, wantQuery: url.Values{"size": {"20"}}},
		{name: "reverse only", opts: &GetScopeSessionsOptions{Reverse: true}, wantQuery: url.Values{"reverse": {"true"}}},
		{
			name:      "every option",
			opts:      &GetScopeSessionsOptions{Page: 3, Size: 10, Reverse: true},
			wantQuery: url.Values{"page": {"3"}, "size": {"10"}, "reverse": {"true"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, received := scopeServer(t, http.StatusOK, scopeSessionPageJSON)

			if _, err := client.GetScopeSessions(t.Context(), "workspace", "team", tt.opts); err != nil {
				t.Fatalf("GetScopeSessions: %v", err)
			}

			checkScopeRequest(t, received(), http.MethodPost, "/v3/workspaces/workspace/scopes/team/sessions/list", tt.wantQuery, "")
		})
	}
}

// GetScopeSessions decodes the member sessions of a scope into a page of sessions.
func TestGetScopeSessionsDecodesPage(t *testing.T) {
	client, _ := scopeServer(t, http.StatusOK, scopeSessionPageJSON)

	page, err := client.GetScopeSessions(t.Context(), "workspace", "team", nil)
	if err != nil {
		t.Fatalf("GetScopeSessions: %v", err)
	}

	want := &PageSession{
		Items: []Session{{
			ID:            "s1",
			UID:           "Xv27Pg3BTadRElK_LnbyJ",
			IsActive:      true,
			WorkspaceID:   "workspace",
			Metadata:      json.RawMessage(`{"topic":"support"}`),
			Configuration: &SessionConfiguration{},
			CreatedAt:     time.Date(2026, time.November, 18, 16, 9, 34, 673599000, time.UTC),
		}},
		Total: 1,
		Page:  1,
		Size:  50,
		Pages: 1,
	}
	if !reflect.DeepEqual(page, want) {
		got, _ := json.Marshal(page)
		wanted, _ := json.Marshal(want)
		t.Errorf("page = %s, want %s", got, wanted)
	}
}

// A scope ID is accepted from 1 to 506 characters matching ^[a-zA-Z0-9_-]+$, as upstream
// measures it in characters, and GetOrCreateScope sends it as it is. Any other ID is refused
// without sending a request.
func TestScopeCreateValidate(t *testing.T) {
	const (
		requiredErr = "id is required"
		lengthErr   = "id must be 506 characters or less"
		patternErr  = "id must contain only letters, numbers, underscores, or hyphens"
	)
	tests := []struct {
		name   string
		id     string
		errMsg string
	}{
		{name: "1 character", id: "a"},
		{name: "every kind of allowed character", id: "aZ09_-"},
		{name: "506 characters", id: strings.Repeat("a", 506)},
		{name: "507 characters", id: strings.Repeat("a", 507), errMsg: lengthErr},
		{name: "empty", id: "", errMsg: requiredErr},
		{name: "space", id: "a b", errMsg: patternErr},
		{name: "dot", id: "scope.team", errMsg: patternErr},
		{name: "slash", id: "a/b", errMsg: patternErr},
		{name: "at sign", id: "a@b", errMsg: patternErr},
		{name: "trailing newline", id: "team\n", errMsg: patternErr},
		{name: "non-ASCII character", id: "équipe", errMsg: patternErr},
		{name: "506 multi-byte characters", id: strings.Repeat("é", 506), errMsg: patternErr},
		{name: "507 multi-byte characters", id: strings.Repeat("é", 507), errMsg: lengthErr},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := ScopeCreate{ID: tt.id}
			client, received := scopeServer(t, http.StatusOK, scopeJSON)

			validateErr := req.Validate()
			_, err := client.GetOrCreateScope(t.Context(), "workspace", req)
			requests := received()

			if tt.errMsg == "" {
				if validateErr != nil {
					t.Errorf("Validate() error = %v, want nil", validateErr)
				}
				if err != nil {
					t.Fatalf("GetOrCreateScope: %v", err)
				}
				body, _ := json.Marshal(req)
				checkScopeRequest(t, requests, http.MethodPost, "/v3/workspaces/workspace/scopes", url.Values{}, string(body))
				return
			}
			if validateErr == nil || validateErr.Error() != tt.errMsg {
				t.Errorf("Validate() error = %v, want %q", validateErr, tt.errMsg)
			}
			if err == nil || err.Error() != tt.errMsg {
				t.Errorf("GetOrCreateScope error = %v, want %q", err, tt.errMsg)
			}
			if len(requests) != 0 {
				t.Errorf("%d requests sent, want none", len(requests))
			}
		})
	}
}

// A membership request is accepted from 1 to 100 session IDs, all of them sent, and refused
// without sending a request when it holds none or more than 100.
func TestScopeSessionsValidate(t *testing.T) {
	const (
		emptyErr = "session_ids must hold at least one session"
		manyErr  = "session_ids must hold 100 sessions or less"
	)
	requests := []struct {
		name     string
		path     string
		validate func(sessionIDs []string) error
		send     func(ctx context.Context, client *Client, sessionIDs []string) error
	}{
		{
			name:     "ScopeSessionsAdd",
			path:     "/v3/workspaces/workspace/scopes/team/sessions",
			validate: func(sessionIDs []string) error { return ScopeSessionsAdd{SessionIDs: sessionIDs}.Validate() },
			send: func(ctx context.Context, client *Client, sessionIDs []string) error {
				return client.AddSessionsToScope(ctx, "workspace", "team", ScopeSessionsAdd{SessionIDs: sessionIDs})
			},
		},
		{
			name:     "ScopeSessionsRemove",
			path:     "/v3/workspaces/workspace/scopes/team/sessions/remove",
			validate: func(sessionIDs []string) error { return ScopeSessionsRemove{SessionIDs: sessionIDs}.Validate() },
			send: func(ctx context.Context, client *Client, sessionIDs []string) error {
				return client.RemoveSessionsFromScope(ctx, "workspace", "team", ScopeSessionsRemove{SessionIDs: sessionIDs})
			},
		},
	}
	tests := []struct {
		name       string
		sessionIDs []string
		errMsg     string
	}{
		{name: "nil", errMsg: emptyErr},
		{name: "empty", sessionIDs: []string{}, errMsg: emptyErr},
		{name: "1 item", sessionIDs: []string{"session"}},
		{name: "100 items", sessionIDs: slices.Repeat([]string{"session"}, 100)},
		{name: "101 items", sessionIDs: slices.Repeat([]string{"session"}, 101), errMsg: manyErr},
	}
	for _, request := range requests {
		for _, tt := range tests {
			t.Run(request.name+"/"+tt.name, func(t *testing.T) {
				client, received := scopeServer(t, http.StatusNoContent, "")

				validateErr := request.validate(tt.sessionIDs)
				err := request.send(t.Context(), client, tt.sessionIDs)
				sent := received()

				if tt.errMsg == "" {
					if validateErr != nil {
						t.Errorf("Validate() error = %v, want nil", validateErr)
					}
					if err != nil {
						t.Fatalf("send error = %v, want nil", err)
					}
					body, _ := json.Marshal(map[string][]string{"session_ids": tt.sessionIDs})
					checkScopeRequest(t, sent, http.MethodPost, request.path, url.Values{}, string(body))
					return
				}
				if validateErr == nil || validateErr.Error() != tt.errMsg {
					t.Errorf("Validate() error = %v, want %q", validateErr, tt.errMsg)
				}
				if err == nil || err.Error() != tt.errMsg {
					t.Errorf("send error = %v, want %q", err, tt.errMsg)
				}
				if len(sent) != 0 {
					t.Errorf("%d requests sent, want none", len(sent))
				}
			})
		}
	}
}

// Every scope method refuses an empty required argument without sending a request.
func TestScopesRequiredArguments(t *testing.T) {
	const (
		workspaceErr = "workspaceID is required"
		scopeErr     = "scopeID is required"
		sessionErr   = "sessionID is required"
	)
	var (
		create = ScopeCreate{ID: "team"}
		add    = ScopeSessionsAdd{SessionIDs: []string{"session"}}
		remove = ScopeSessionsRemove{SessionIDs: []string{"session"}}
	)
	tests := []struct {
		name   string
		call   func(ctx context.Context, client *Client) error
		errMsg string
	}{
		{
			name: "GetOrCreateScope with an empty workspaceID",
			call: func(ctx context.Context, client *Client) error {
				return errorOf(client.GetOrCreateScope(ctx, "", create))
			},
			errMsg: workspaceErr,
		},
		{
			name: "GetScopes with an empty workspaceID",
			call: func(ctx context.Context, client *Client) error {
				return errorOf(client.GetScopes(ctx, "", nil, nil))
			},
			errMsg: workspaceErr,
		},
		{
			name: "GetScope with an empty workspaceID",
			call: func(ctx context.Context, client *Client) error {
				return errorOf(client.GetScope(ctx, "", "team"))
			},
			errMsg: workspaceErr,
		},
		{
			name: "GetScope with an empty scopeID",
			call: func(ctx context.Context, client *Client) error {
				return errorOf(client.GetScope(ctx, "workspace", ""))
			},
			errMsg: scopeErr,
		},
		{
			name: "GetScopeStatus with an empty workspaceID",
			call: func(ctx context.Context, client *Client) error {
				return errorOf(client.GetScopeStatus(ctx, "", "team"))
			},
			errMsg: workspaceErr,
		},
		{
			name: "GetScopeStatus with an empty scopeID",
			call: func(ctx context.Context, client *Client) error {
				return errorOf(client.GetScopeStatus(ctx, "workspace", ""))
			},
			errMsg: scopeErr,
		},
		{
			name: "AddSessionsToScope with an empty workspaceID",
			call: func(ctx context.Context, client *Client) error {
				return client.AddSessionsToScope(ctx, "", "team", add)
			},
			errMsg: workspaceErr,
		},
		{
			name: "AddSessionsToScope with an empty scopeID",
			call: func(ctx context.Context, client *Client) error {
				return client.AddSessionsToScope(ctx, "workspace", "", add)
			},
			errMsg: scopeErr,
		},
		{
			name: "GetScopeSessions with an empty workspaceID",
			call: func(ctx context.Context, client *Client) error {
				return errorOf(client.GetScopeSessions(ctx, "", "team", nil))
			},
			errMsg: workspaceErr,
		},
		{
			name: "GetScopeSessions with an empty scopeID",
			call: func(ctx context.Context, client *Client) error {
				return errorOf(client.GetScopeSessions(ctx, "workspace", "", nil))
			},
			errMsg: scopeErr,
		},
		{
			name: "RemoveSessionsFromScope with an empty workspaceID",
			call: func(ctx context.Context, client *Client) error {
				return client.RemoveSessionsFromScope(ctx, "", "team", remove)
			},
			errMsg: workspaceErr,
		},
		{
			name: "RemoveSessionsFromScope with an empty scopeID",
			call: func(ctx context.Context, client *Client) error {
				return client.RemoveSessionsFromScope(ctx, "workspace", "", remove)
			},
			errMsg: scopeErr,
		},
		{
			name: "RemoveSessionFromScope with an empty workspaceID",
			call: func(ctx context.Context, client *Client) error {
				return client.RemoveSessionFromScope(ctx, "", "team", "session")
			},
			errMsg: workspaceErr,
		},
		{
			name: "RemoveSessionFromScope with an empty scopeID",
			call: func(ctx context.Context, client *Client) error {
				return client.RemoveSessionFromScope(ctx, "workspace", "", "session")
			},
			errMsg: scopeErr,
		},
		{
			name: "RemoveSessionFromScope with an empty sessionID",
			call: func(ctx context.Context, client *Client) error {
				return client.RemoveSessionFromScope(ctx, "workspace", "team", "")
			},
			errMsg: sessionErr,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, received := scopeServer(t, http.StatusNoContent, "")

			err := tt.call(t.Context(), client)

			if err == nil || err.Error() != tt.errMsg {
				t.Errorf("error = %v, want %q", err, tt.errMsg)
			}
			if requests := received(); len(requests) != 0 {
				t.Errorf("%d requests sent, want none", len(requests))
			}
		})
	}
}

// A 404 comes back from every scope method as an error naming what failed and holding the
// detail of the response, such as the scope or the sessions Honcho did not find.
func TestScopesNotFound(t *testing.T) {
	const (
		unknownWorkspace = "Workspace w not found"
		unknownScope     = "Scope nope not found in workspace w"
		unknownSessions  = "Session(s) ['nope'] not found in workspace w"
		unknownSession   = "Session nope not found in workspace w"
	)
	var (
		add    = ScopeSessionsAdd{SessionIDs: []string{"nope"}}
		remove = ScopeSessionsRemove{SessionIDs: []string{"nope"}}
	)
	tests := []struct {
		name   string
		call   func(ctx context.Context, client *Client) error
		prefix string
		detail string
	}{
		{
			name: "GetOrCreateScope",
			call: func(ctx context.Context, client *Client) error {
				return errorOf(client.GetOrCreateScope(ctx, "w", ScopeCreate{ID: "team"}))
			},
			prefix: "failed to get or create scope: ",
			detail: unknownWorkspace,
		},
		{
			name: "GetScopes",
			call: func(ctx context.Context, client *Client) error {
				return errorOf(client.GetScopes(ctx, "w", nil, nil))
			},
			prefix: "failed to get scopes: ",
			detail: unknownWorkspace,
		},
		{
			name: "GetScope",
			call: func(ctx context.Context, client *Client) error {
				return errorOf(client.GetScope(ctx, "w", "nope"))
			},
			prefix: "failed to get scope: ",
			detail: unknownScope,
		},
		{
			name: "GetScopeStatus",
			call: func(ctx context.Context, client *Client) error {
				return errorOf(client.GetScopeStatus(ctx, "w", "nope"))
			},
			prefix: "failed to get scope status: ",
			detail: unknownScope,
		},
		{
			name: "GetScopeSessions",
			call: func(ctx context.Context, client *Client) error {
				return errorOf(client.GetScopeSessions(ctx, "w", "nope", nil))
			},
			prefix: "failed to get scope sessions: ",
			detail: unknownScope,
		},
		{
			name: "AddSessionsToScope with an unknown scope",
			call: func(ctx context.Context, client *Client) error {
				return client.AddSessionsToScope(ctx, "w", "nope", add)
			},
			prefix: "failed to add sessions to scope: ",
			detail: unknownScope,
		},
		{
			name: "AddSessionsToScope with an unknown session",
			call: func(ctx context.Context, client *Client) error {
				return client.AddSessionsToScope(ctx, "w", "team", add)
			},
			prefix: "failed to add sessions to scope: ",
			detail: unknownSessions,
		},
		{
			name: "RemoveSessionsFromScope with an unknown scope",
			call: func(ctx context.Context, client *Client) error {
				return client.RemoveSessionsFromScope(ctx, "w", "nope", remove)
			},
			prefix: "failed to remove sessions from scope: ",
			detail: unknownScope,
		},
		{
			name: "RemoveSessionsFromScope with an unknown session",
			call: func(ctx context.Context, client *Client) error {
				return client.RemoveSessionsFromScope(ctx, "w", "team", remove)
			},
			prefix: "failed to remove sessions from scope: ",
			detail: unknownSessions,
		},
		{
			name: "RemoveSessionFromScope with an unknown scope",
			call: func(ctx context.Context, client *Client) error {
				return client.RemoveSessionFromScope(ctx, "w", "nope", "session")
			},
			prefix: "failed to remove session from scope: ",
			detail: unknownScope,
		},
		{
			name: "RemoveSessionFromScope with an unknown session",
			call: func(ctx context.Context, client *Client) error {
				return client.RemoveSessionFromScope(ctx, "w", "team", "nope")
			},
			prefix: "failed to remove session from scope: ",
			detail: unknownSession,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, _ := scopeServer(t, http.StatusNotFound, fmt.Sprintf(`{"detail":%q}`, tt.detail))

			err := tt.call(t.Context(), client)

			if err == nil {
				t.Fatal("error = nil, want an error")
			}
			if !strings.HasPrefix(err.Error(), tt.prefix) {
				t.Errorf("error = %q, want it prefixed with %q", err, tt.prefix)
			}
			if !strings.Contains(err.Error(), tt.detail) {
				t.Errorf("error = %q, want it to hold %q", err, tt.detail)
			}
		})
	}
}
