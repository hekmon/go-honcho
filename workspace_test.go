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
