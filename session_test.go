package honcho

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"
)

// A session's configuration decodes with every section and value Honcho
// returns, an empty configuration holds no section, and a response without a
// configuration leaves it nil. Regression test: Session declared a peer's
// configuration, whose fields Honcho does not return for a session, so a
// session's configuration always read as empty.
func TestSessionConfiguration(t *testing.T) {
	const fields = `"id":"session","is_active":true,"workspace_id":"workspace","created_at":"2026-10-30T00:00:00Z"`
	yes, no := true, false
	short, long := 10, 20
	tests := []struct {
		name string
		body string
		want *SessionConfiguration
	}{
		{
			name: "full configuration",
			body: `{` + fields + `,"configuration":{"reasoning": {"enabled": false}, "peer_card": {"use": true, "create": false}, "summary": {"enabled": true, "messages_per_short_summary": 10, "messages_per_long_summary": 20}, "dream": {"enabled": false}}}`,
			want: &SessionConfiguration{
				Reasoning: &ReasoningConfiguration{Enabled: &no},
				PeerCard:  &PeerCardConfiguration{Use: &yes, Create: &no},
				Summary:   &SummaryConfiguration{Enabled: &yes, MessagesPerShortSummary: &short, MessagesPerLongSummary: &long},
				Dream:     &DreamConfiguration{Enabled: &no},
			},
		},
		{name: "empty configuration", body: `{` + fields + `,"configuration":{}}`, want: &SessionConfiguration{}},
		{name: "no configuration", body: `{` + fields + `}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprint(w, tt.body)
			}))
			defer server.Close()
			u, _ := url.Parse(server.URL)

			session, err := New(&Options{BaseURL: u}).GetOrCreateSession(t.Context(), "workspace", SessionCreate{ID: "session"})
			if err != nil {
				t.Fatalf("GetOrCreateSession: %v", err)
			}
			if session.ID != "session" {
				t.Errorf("ID = %q, want %q", session.ID, "session")
			}
			if !reflect.DeepEqual(session.Configuration, tt.want) {
				got, _ := json.Marshal(session.Configuration)
				want, _ := json.Marshal(tt.want)
				t.Errorf("Configuration = %s, want %s", got, want)
			}
		})
	}
}

// A session's configuration, read and sent back as it is, reaches Honcho with
// every section and value Honcho returned. Regression test: SessionConfiguration
// lacked the dialectic section and the custom instructions outside the reasoning
// section, so a read dropped them and writing the configuration back erased those
// of the peer card, summary and dream sections, as a section that is sent
// replaces the stored section of that name.
func TestSessionConfigurationRoundTrip(t *testing.T) {
	var update []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			var err error
			if update, err = io.ReadAll(r.Body); err != nil {
				t.Errorf("failed to read the request body: %v", err)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":"session","is_active":true,"workspace_id":"workspace","created_at":"2026-10-30T00:00:00Z","configuration":`+fullConfiguration+`}`)
	}))
	defer server.Close()
	u, _ := url.Parse(server.URL)
	client := New(&Options{BaseURL: u})

	session, err := client.GetOrCreateSession(t.Context(), "workspace", SessionCreate{ID: "session"})
	if err != nil {
		t.Fatalf("GetOrCreateSession: %v", err)
	}
	if _, err = client.UpdateSession(t.Context(), "workspace", "session", SessionUpdate{Configuration: session.Configuration}); err != nil {
		t.Fatalf("UpdateSession: %v", err)
	}
	// Close waits for the handler, so what it recorded can be read.
	server.Close()

	checkFullConfiguration(t, update)
}

// GetSessions sends a POST request to the workspace's sessions list path, with the options set
// in the query and the restrictions set in the body. What is left unset is not sent, so the
// server applies its defaults. An empty scope name is sent as it is, and so are a scope and
// InNoScope together: refusing them is up to the server.
func TestGetSessionsRequest(t *testing.T) {
	scope, emptyScope := "team", ""
	filters := map[string]any{"metadata": map[string]any{"topic": "support"}}
	tests := []struct {
		name      string
		req       *WorkspaceSessionGet
		opts      *GetSessionsOptions
		wantQuery url.Values
		wantBody  string
	}{
		{name: "nil request and nil options", wantQuery: url.Values{}, wantBody: `null`},
		{
			name:      "empty request and empty options",
			req:       &WorkspaceSessionGet{},
			opts:      &GetSessionsOptions{},
			wantQuery: url.Values{},
			wantBody:  `{}`,
		},
		{name: "page only", opts: &GetSessionsOptions{Page: 2}, wantQuery: url.Values{"page": {"2"}}, wantBody: `null`},
		{name: "reverse only", opts: &GetSessionsOptions{Reverse: true}, wantQuery: url.Values{"reverse": {"true"}}, wantBody: `null`},
		{
			name:      "include scopes only",
			opts:      &GetSessionsOptions{IncludeScopes: true},
			wantQuery: url.Values{"include_scopes": {"true"}},
			wantBody:  `null`,
		},
		{
			name:      "filters only",
			req:       &WorkspaceSessionGet{Filters: filters},
			wantQuery: url.Values{},
			wantBody:  `{"filters":{"metadata":{"topic":"support"}}}`,
		},
		{name: "scope only", req: &WorkspaceSessionGet{Scope: &scope}, wantQuery: url.Values{}, wantBody: `{"scope":"team"}`},
		{name: "empty scope name", req: &WorkspaceSessionGet{Scope: &emptyScope}, wantQuery: url.Values{}, wantBody: `{"scope":""}`},
		{name: "in no scope only", req: &WorkspaceSessionGet{InNoScope: true}, wantQuery: url.Values{}, wantBody: `{"in_no_scope":true}`},
		{
			name:      "scope and in no scope",
			req:       &WorkspaceSessionGet{Scope: &scope, InNoScope: true},
			wantQuery: url.Values{},
			wantBody:  `{"scope":"team","in_no_scope":true}`,
		},
		{
			name: "every option with a scope",
			req:  &WorkspaceSessionGet{Filters: filters, Scope: &scope},
			opts: &GetSessionsOptions{Page: 3, Size: 10, Reverse: true, IncludeScopes: true},
			wantQuery: url.Values{
				"page":           {"3"},
				"size":           {"10"},
				"reverse":        {"true"},
				"include_scopes": {"true"},
			},
			wantBody: `{"filters":{"metadata":{"topic":"support"}},"scope":"team"}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, received := scopeServer(t, http.StatusOK, scopeSessionPageJSON)

			if _, err := client.GetSessions(t.Context(), "workspace", tt.req, tt.opts); err != nil {
				t.Fatalf("GetSessions: %v", err)
			}

			checkScopeRequest(t, received(), http.MethodPost, "/v3/workspaces/workspace/sessions/list", tt.wantQuery, tt.wantBody)
		})
	}
}

// Session.Scopes tells apart what the sessions list can return: nil when the field is absent,
// as it is without IncludeScopes, an empty non-nil slice for a session in no scope, and the
// names of the scopes otherwise.
func TestSessionScopes(t *testing.T) {
	const fields = `"id":"s1","uid":"Xv27Pg3BTadRElK_LnbyJ","is_active":true,"workspace_id":"workspace","metadata":{},"configuration":{},"created_at":"2026-11-18T16:09:34.673599Z"`
	tests := []struct {
		name    string
		session string
		want    []string
	}{
		{name: "absent", session: `{` + fields + `}`},
		{name: "in no scope", session: `{` + fields + `,"scopes":[]}`, want: []string{}},
		{name: "in two scopes", session: `{` + fields + `,"scopes":["team","project"]}`, want: []string{"team", "project"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, _ := scopeServer(t, http.StatusOK, `{"items":[`+tt.session+`],"total":1,"page":1,"size":50,"pages":1}`)

			page, err := client.GetSessions(t.Context(), "workspace", nil, &GetSessionsOptions{IncludeScopes: true})
			if err != nil {
				t.Fatalf("GetSessions: %v", err)
			}

			if len(page.Items) != 1 {
				t.Fatalf("%d sessions decoded, want 1", len(page.Items))
			}
			// DeepEqual tells a nil slice from an empty one.
			if got := page.Items[0].Scopes; !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Scopes = %#v, want %#v", got, tt.want)
			}
		})
	}
}
