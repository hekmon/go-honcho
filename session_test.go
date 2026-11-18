package honcho

import (
	"encoding/json"
	"fmt"
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
