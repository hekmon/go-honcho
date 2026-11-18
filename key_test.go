package honcho

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

// CreateKey sends the scopes and the expiry in the query string of a POST
// request without a body, and its result holds the key Honcho returns and
// nothing else. Regression test: Key also declared the scopes, the expiry and a
// creation time, which Honcho does not return, so they always read as empty and
// a marshalled key carried a zero created_at that looked like real data.
func TestCreateKey(t *testing.T) {
	var method, path, query, body string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("failed to read the request body: %v", err)
		}
		method, path, query, body = r.Method, r.URL.Path, r.URL.RawQuery, string(data)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"key":"token"}`)
	}))
	u, _ := url.Parse(server.URL)
	expiresAt := time.Date(2026, time.December, 31, 23, 59, 59, 0, time.UTC)

	key, err := New(&Options{BaseURL: u}).CreateKey(t.Context(), CreateKeyRequest{
		WorkspaceID: "workspace",
		PeerID:      "peer",
		SessionID:   "session",
		ExpiresAt:   &expiresAt,
	})
	// Close waits for the handler, so what it recorded can be read.
	server.Close()
	if err != nil {
		t.Fatalf("CreateKey: %v", err)
	}

	if method != http.MethodPost {
		t.Errorf("method = %s, want %s", method, http.MethodPost)
	}
	if path != "/v3/keys" {
		t.Errorf("path = %q, want %q", path, "/v3/keys")
	}
	const wantQuery = "expires_at=2026-12-31T23%3A59%3A59Z&peer_id=peer&session_id=session&workspace_id=workspace"
	if query != wantQuery {
		t.Errorf("query = %q, want %q", query, wantQuery)
	}
	if body != "" {
		t.Errorf("body = %q, want it empty", body)
	}
	if key.Key != "token" {
		t.Errorf("Key = %q, want %q", key.Key, "token")
	}
	const wantJSON = `{"key":"token"}`
	data, err := json.Marshal(key)
	if err != nil {
		t.Fatalf("failed to marshal the key: %v", err)
	}
	if string(data) != wantJSON {
		t.Errorf("marshalled key = %s, want %s", data, wantJSON)
	}
}

// CreateKey sends the name of a named scope as scope_id, and no scope_id at all when ScopeID is nil.
func TestCreateKeyScopeID(t *testing.T) {
	scope := "team"
	tests := []struct {
		name      string
		req       CreateKeyRequest
		wantQuery url.Values
	}{
		{
			name:      "named scope",
			req:       CreateKeyRequest{WorkspaceID: "workspace", ScopeID: &scope},
			wantQuery: url.Values{"workspace_id": {"workspace"}, "scope_id": {"team"}},
		},
		{
			name:      "nil scope ID",
			req:       CreateKeyRequest{WorkspaceID: "workspace"},
			wantQuery: url.Values{"workspace_id": {"workspace"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, received := scopeServer(t, http.StatusOK, `{"key":"token"}`)

			if _, err := client.CreateKey(t.Context(), tt.req); err != nil {
				t.Fatalf("CreateKey: %v", err)
			}

			checkScopeRequest(t, received(), http.MethodPost, "/v3/keys", tt.wantQuery, "")
		})
	}
}

// CreateKey refuses a non-nil empty ScopeID before sending anything: leaving it out of the request
// would mint a workspace-wide key instead of the restricted one the caller asked for.
func TestCreateKeyRefusesEmptyScopeID(t *testing.T) {
	client, received := scopeServer(t, http.StatusOK, `{"key":"token"}`)
	scope := ""

	key, err := client.CreateKey(t.Context(), CreateKeyRequest{WorkspaceID: "workspace", ScopeID: &scope})

	if want := "scope_id must not be empty: leave it nil for a key not tied to a named scope"; err == nil || err.Error() != want {
		t.Errorf("error = %v, want %q", err, want)
	}
	if key != nil {
		t.Errorf("key = %+v, want nil", key)
	}
	if requests := received(); len(requests) != 0 {
		t.Errorf("%d requests sent, want none", len(requests))
	}
}
