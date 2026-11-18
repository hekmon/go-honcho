package honcho

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"
)

// GetConclusion sends a GET request without a body to the conclusion's path and
// decodes every field a real Honcho server returns: an explicit conclusion citing
// its messages, and a derived one whose null session and null cited messages read
// as nil.
func TestGetConclusion(t *testing.T) {
	sessionID := "s"
	tests := []struct {
		name string
		body string
		want Conclusion
	}{
		{
			name: "explicit conclusion",
			body: `{"id": "vnzseRU4JQrIGkJg4FQoj", "content": "alice has a sister named Joke.", "observer_id": "alice", "observed_id": "alice", "session_id": "s", "level": "explicit", "source_ids": [], "source_message_ids": ["cgGcyBR7s6VdYYv40PKvN"], "times_derived": 1, "created_at": "2026-11-18T14:33:17.975162Z"}`,
			want: Conclusion{
				ID:               "vnzseRU4JQrIGkJg4FQoj",
				Content:          "alice has a sister named Joke.",
				ObserverID:       "alice",
				ObservedID:       "alice",
				SessionID:        &sessionID,
				Level:            ConclusionLevelExplicit,
				SourceIDs:        []string{},
				SourceMessageIDs: []string{"cgGcyBR7s6VdYYv40PKvN"},
				TimesDerived:     1,
				CreatedAt:        time.Date(2026, time.November, 18, 14, 33, 17, 975162000, time.UTC),
			},
		},
		{
			name: "deductive conclusion",
			body: `{"id": "ZKRVWepoHZUN4BV8pGK1E", "content": "alice's bridge design work falls within her profession.", "observer_id": "alice", "observed_id": "alice", "session_id": null, "level": "deductive", "source_ids": ["DB7KJZFicoM2Qo3hOv5li", "2iCnyxwp4m3f2HhvFhESq"], "source_message_ids": null, "times_derived": 1, "created_at": "2026-11-18T14:34:33.840982Z"}`,
			want: Conclusion{
				ID:               "ZKRVWepoHZUN4BV8pGK1E",
				Content:          "alice's bridge design work falls within her profession.",
				ObserverID:       "alice",
				ObservedID:       "alice",
				SessionID:        nil,
				Level:            ConclusionLevelDeductive,
				SourceIDs:        []string{"DB7KJZFicoM2Qo3hOv5li", "2iCnyxwp4m3f2HhvFhESq"},
				SourceMessageIDs: nil,
				TimesDerived:     1,
				CreatedAt:        time.Date(2026, time.November, 18, 14, 34, 33, 840982000, time.UTC),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var method, path, body string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				data, err := io.ReadAll(r.Body)
				if err != nil {
					t.Errorf("failed to read the request body: %v", err)
				}
				method, path, body = r.Method, r.URL.Path, string(data)
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprint(w, tt.body)
			}))
			u, _ := url.Parse(server.URL)

			conclusion, err := New(&Options{BaseURL: u}).GetConclusion(t.Context(), "workspace", tt.want.ID)
			// Close waits for the handler, so what it recorded can be read.
			server.Close()
			if err != nil {
				t.Fatalf("GetConclusion: %v", err)
			}

			if method != http.MethodGet {
				t.Errorf("method = %s, want %s", method, http.MethodGet)
			}
			if want := "/v3/workspaces/workspace/conclusions/" + tt.want.ID; path != want {
				t.Errorf("path = %q, want %q", path, want)
			}
			if body != "" {
				t.Errorf("body = %q, want it empty", body)
			}
			// DeepEqual tells a nil list from an empty one.
			if !reflect.DeepEqual(conclusion, &tt.want) {
				got, _ := json.Marshal(conclusion)
				want, _ := json.Marshal(tt.want)
				t.Errorf("conclusion = %s, want %s", got, want)
			}
		})
	}
}

// GetConclusion refuses an empty workspace or conclusion ID without sending a
// request.
func TestGetConclusionRequiredArguments(t *testing.T) {
	tests := []struct {
		name         string
		workspaceID  string
		conclusionID string
		errMsg       string
	}{
		{name: "empty workspaceID", conclusionID: "conclusion", errMsg: "workspaceID is required"},
		{name: "empty conclusionID", workspaceID: "workspace", errMsg: "conclusionID is required"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var requests int
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests++
			}))
			u, _ := url.Parse(server.URL)

			_, err := New(&Options{BaseURL: u}).GetConclusion(t.Context(), tt.workspaceID, tt.conclusionID)
			// Close waits for the handler, so what it recorded can be read.
			server.Close()

			if err == nil || err.Error() != tt.errMsg {
				t.Errorf("GetConclusion() error = %v, want %q", err, tt.errMsg)
			}
			if requests != 0 {
				t.Errorf("%d requests sent, want none", requests)
			}
		})
	}
}

// A conclusion Honcho does not find comes back as an error holding the detail of
// the 404 response.
func TestGetConclusionNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `{"detail":"Conclusion not found"}`)
	}))
	defer server.Close()
	u, _ := url.Parse(server.URL)

	_, err := New(&Options{BaseURL: u}).GetConclusion(t.Context(), "workspace", "unknown")
	if err == nil {
		t.Fatal("GetConclusion() error = nil, want an error")
	}
	if !strings.HasPrefix(err.Error(), "failed to get conclusion: ") {
		t.Errorf("error = %q, want it prefixed with %q", err, "failed to get conclusion: ")
	}
	if !strings.Contains(err.Error(), "Conclusion not found") {
		t.Errorf("error = %q, want it to hold %q", err, "Conclusion not found")
	}
}
