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
