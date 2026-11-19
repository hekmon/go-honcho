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
	"strings"
	"testing"
	"time"
)

// dreamRunJSON holds the fields of a dream run captured from a real Honcho server,
// shared by the run of a list and the run of a detail.
const dreamRunJSON = `"id":"AM77WLfN_fQA29JSVbOl0","observer_id":"bob","observed_id":"alice","dream_type":"omni","trigger_reason":"manual","status":"completed","created_count":7,"removed_count":1,"peer_card_updated":true,"started_at":"2026-11-18T15:31:39.540638Z","finished_at":"2026-11-18T15:32:50.496594Z","pruned_at":null`

// pointerTo returns a pointer to value.
func pointerTo[T any](value T) *T {
	return &value
}

// capturedDreamRun returns what dreamRunJSON decodes to.
func capturedDreamRun() DreamRun {
	return DreamRun{
		ID:              "AM77WLfN_fQA29JSVbOl0",
		ObserverID:      "bob",
		ObservedID:      "alice",
		DreamType:       pointerTo(DreamTypeOmni),
		TriggerReason:   pointerTo("manual"),
		Status:          DreamRunStatusCompleted,
		CreatedCount:    7,
		RemovedCount:    1,
		PeerCardUpdated: true,
		StartedAt:       time.Date(2026, time.November, 18, 15, 31, 39, 540638000, time.UTC),
		FinishedAt:      pointerTo(time.Date(2026, time.November, 18, 15, 32, 50, 496594000, time.UTC)),
		PrunedAt:        nil,
	}
}

// ListDreams sends a POST request to the workspace's dreams list path, with the page
// and size set in the query and exactly the fields set in the body. A nil request is
// sent as a JSON null, which the route reads as no options, and nil options leave the
// query empty.
func TestListDreamsRequest(t *testing.T) {
	tests := []struct {
		name      string
		req       *DreamRunList
		opts      *ListDreamsOptions
		wantQuery url.Values
		wantBody  map[string]any
	}{
		{name: "nil request and nil options", wantQuery: url.Values{}},
		{
			name:      "empty request and empty options",
			req:       &DreamRunList{},
			opts:      &ListDreamsOptions{},
			wantQuery: url.Values{},
			wantBody:  map[string]any{},
		},
		{name: "page only", opts: &ListDreamsOptions{Page: 2}, wantQuery: url.Values{"page": {"2"}}},
		{name: "size only", opts: &ListDreamsOptions{Size: 20}, wantQuery: url.Values{"size": {"20"}}},
		{
			name:      "observed peer only",
			req:       &DreamRunList{ObservedID: pointerTo("alice")},
			wantQuery: url.Values{},
			wantBody:  map[string]any{"observed_id": "alice"},
		},
		{
			name:      "observer only",
			req:       &DreamRunList{ObserverID: pointerTo("bob")},
			wantQuery: url.Values{},
			wantBody:  map[string]any{"observer_id": "bob"},
		},
		{
			name:      "every field",
			req:       &DreamRunList{ObservedID: pointerTo("alice"), ObserverID: pointerTo("bob")},
			opts:      &ListDreamsOptions{Page: 3, Size: 10},
			wantQuery: url.Values{"page": {"3"}, "size": {"10"}},
			wantBody:  map[string]any{"observed_id": "alice", "observer_id": "bob"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var (
				method, path string
				query        url.Values
				body         []byte
			)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				data, err := io.ReadAll(r.Body)
				if err != nil {
					t.Errorf("failed to read the request body: %v", err)
				}
				method, path, query, body = r.Method, r.URL.Path, r.URL.Query(), data
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprint(w, `{"items":[],"total":0,"page":1,"size":50,"pages":0}`)
			}))
			u, _ := url.Parse(server.URL)

			_, err := New(&Options{BaseURL: u}).ListDreams(t.Context(), "workspace", tt.req, tt.opts)
			// Close waits for the handler, so what it recorded can be read.
			server.Close()
			if err != nil {
				t.Fatalf("ListDreams: %v", err)
			}

			if method != http.MethodPost {
				t.Errorf("method = %s, want %s", method, http.MethodPost)
			}
			if want := "/v3/workspaces/workspace/dreams/list"; path != want {
				t.Errorf("path = %q, want %q", path, want)
			}
			if !reflect.DeepEqual(query, tt.wantQuery) {
				t.Errorf("query = %v, want %v", query, tt.wantQuery)
			}
			var fields map[string]any
			if err := json.Unmarshal(body, &fields); err != nil {
				t.Fatalf("failed to decode the request body %q: %v", body, err)
			}
			if !reflect.DeepEqual(fields, tt.wantBody) {
				t.Errorf("body = %s, want the fields %v", body, tt.wantBody)
			}
		})
	}
}

// ListDreams decodes a page of dream runs as a real Honcho server returns it.
func TestListDreamsDecodesPage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"items":[{`+dreamRunJSON+`}],"total":3,"page":1,"size":50,"pages":1}`)
	}))
	defer server.Close()
	u, _ := url.Parse(server.URL)

	page, err := New(&Options{BaseURL: u}).ListDreams(t.Context(), "workspace", nil, nil)
	if err != nil {
		t.Fatalf("ListDreams: %v", err)
	}

	want := &PageDreamRun{Items: []DreamRun{capturedDreamRun()}, Total: 3, Page: 1, Size: 50, Pages: 1}
	if !reflect.DeepEqual(page, want) {
		got, _ := json.Marshal(page)
		wanted, _ := json.Marshal(want)
		t.Errorf("page = %s, want %s", got, wanted)
	}
}

// Each status constant holds the value Honcho sends for it.
func TestDreamStatusValues(t *testing.T) {
	tests := []struct {
		got  string
		want string
	}{
		{string(DreamRunStatusRunning), "running"},
		{string(DreamRunStatusCompleted), "completed"},
		{string(DreamRunStatusFailed), "failed"},
		{string(DreamRunStatusInterrupted), "interrupted"},
		{string(DreamConclusionStatusCurrent), "current"},
		{string(DreamConclusionStatusRemoved), "removed"},
		{string(DreamConclusionStatusUnavailable), "unavailable"},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("status = %q, want %q", tt.got, tt.want)
		}
	}
}

// GetDream sends a GET request without a body to the run's path and decodes every
// field of the run and of its changes: those a real Honcho server returned, the null
// fields of a run still in progress, which read as nil, and the fields only a pruned
// run or a card that already existed sets.
func TestGetDream(t *testing.T) {
	tests := []struct {
		name string
		body string
		want DreamRunDetail
	}{
		{
			name: "captured run",
			body: `{` + dreamRunJSON + `,"created":[{"conclusion":{"id":"YTj0gii5wZ-w-oHzY4f-l","content":"alice is a pet owner, with a cat named Marzipan.","level":"deductive","status":"current","removed_in_run":null},"sources":[{"id":"5krQE8z2HwfRzCM5GvLig","content":"alice owns a cat named Marzipan.","level":"explicit","status":"current","removed_in_run":null}],"specialist":"deduction"}],"removed":[{"id":"r1","content":"alice lives in Rotterdam.","level":"explicit","replaced_by":"n1","specialist":"deduction"}],"peer_card":{"before":null,"after":["IDENTITY: Name: alice"]}}`,
			want: DreamRunDetail{
				DreamRun: capturedDreamRun(),
				Created: []DreamCreatedConclusion{{
					Conclusion: DreamConclusionRef{
						ID:      "YTj0gii5wZ-w-oHzY4f-l",
						Content: pointerTo("alice is a pet owner, with a cat named Marzipan."),
						Level:   pointerTo(ConclusionLevelDeductive),
						Status:  DreamConclusionStatusCurrent,
					},
					Sources: []DreamConclusionRef{{
						ID:      "5krQE8z2HwfRzCM5GvLig",
						Content: pointerTo("alice owns a cat named Marzipan."),
						Level:   pointerTo(ConclusionLevelExplicit),
						Status:  DreamConclusionStatusCurrent,
					}},
					Specialist: pointerTo("deduction"),
				}},
				Removed: []DreamRemovedConclusion{{
					ID:         "r1",
					Content:    "alice lives in Rotterdam.",
					Level:      pointerTo(ConclusionLevelExplicit),
					ReplacedBy: pointerTo("n1"),
					Specialist: pointerTo("deduction"),
				}},
				PeerCard: &DreamPeerCardChange{Before: nil, After: []string{"IDENTITY: Name: alice"}},
			},
		},
		{
			name: "null fields",
			body: `{"id":"run","observer_id":"bob","observed_id":"alice","dream_type":null,"trigger_reason":null,"status":"running","created_count":1,"removed_count":1,"peer_card_updated":false,"started_at":"2026-11-18T15:31:39.540638Z","finished_at":null,"pruned_at":null,"created":[{"conclusion":{"id":"gone","content":null,"level":null,"status":"unavailable","removed_in_run":null},"sources":[],"specialist":null}],"removed":[{"id":"r1","content":"alice lives in Rotterdam.","level":null,"replaced_by":null,"specialist":null}],"peer_card":null}`,
			want: DreamRunDetail{
				DreamRun: DreamRun{
					ID:           "run",
					ObserverID:   "bob",
					ObservedID:   "alice",
					Status:       DreamRunStatusRunning,
					CreatedCount: 1,
					RemovedCount: 1,
					StartedAt:    time.Date(2026, time.November, 18, 15, 31, 39, 540638000, time.UTC),
				},
				Created: []DreamCreatedConclusion{{
					Conclusion: DreamConclusionRef{ID: "gone", Status: DreamConclusionStatusUnavailable},
					Sources:    []DreamConclusionRef{},
				}},
				Removed: []DreamRemovedConclusion{{ID: "r1", Content: "alice lives in Rotterdam."}},
			},
		},
		{
			name: "pruned run over an existing card",
			body: `{"id":"run","observer_id":"bob","observed_id":"alice","dream_type":"card_refresh","trigger_reason":"manual","status":"interrupted","created_count":1,"removed_count":0,"peer_card_updated":true,"started_at":"2026-11-18T15:31:39.540638Z","finished_at":"2026-11-18T15:32:50.496594Z","pruned_at":"2026-11-19T08:00:00Z","created":[{"conclusion":{"id":"c1","content":"alice owns a cat.","level":"inductive","status":"removed","removed_in_run":"later"},"sources":[],"specialist":null}],"removed":[],"peer_card":{"before":["NAME: alice"],"after":[]}}`,
			want: DreamRunDetail{
				DreamRun: DreamRun{
					ID:              "run",
					ObserverID:      "bob",
					ObservedID:      "alice",
					DreamType:       pointerTo(DreamTypeCardRefresh),
					TriggerReason:   pointerTo("manual"),
					Status:          DreamRunStatusInterrupted,
					CreatedCount:    1,
					PeerCardUpdated: true,
					StartedAt:       time.Date(2026, time.November, 18, 15, 31, 39, 540638000, time.UTC),
					FinishedAt:      pointerTo(time.Date(2026, time.November, 18, 15, 32, 50, 496594000, time.UTC)),
					PrunedAt:        pointerTo(time.Date(2026, time.November, 19, 8, 0, 0, 0, time.UTC)),
				},
				Created: []DreamCreatedConclusion{{
					Conclusion: DreamConclusionRef{
						ID:           "c1",
						Content:      pointerTo("alice owns a cat."),
						Level:        pointerTo(ConclusionLevelInductive),
						Status:       DreamConclusionStatusRemoved,
						RemovedInRun: pointerTo("later"),
					},
					Sources: []DreamConclusionRef{},
				}},
				Removed:  []DreamRemovedConclusion{},
				PeerCard: &DreamPeerCardChange{Before: []string{"NAME: alice"}, After: []string{}},
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

			dream, err := New(&Options{BaseURL: u}).GetDream(t.Context(), "workspace", tt.want.ID)
			// Close waits for the handler, so what it recorded can be read.
			server.Close()
			if err != nil {
				t.Fatalf("GetDream: %v", err)
			}

			if method != http.MethodGet {
				t.Errorf("method = %s, want %s", method, http.MethodGet)
			}
			if want := "/v3/workspaces/workspace/dreams/" + tt.want.ID; path != want {
				t.Errorf("path = %q, want %q", path, want)
			}
			if body != "" {
				t.Errorf("body = %q, want it empty", body)
			}
			// DeepEqual tells a nil list from an empty one.
			if !reflect.DeepEqual(dream, &tt.want) {
				got, _ := json.Marshal(dream)
				want, _ := json.Marshal(tt.want)
				t.Errorf("dream = %s, want %s", got, want)
			}
		})
	}
}

// ListDreams and GetDream refuse an empty required argument without sending a request.
func TestDreamsRequiredArguments(t *testing.T) {
	tests := []struct {
		name   string
		call   func(ctx context.Context, client *Client) error
		errMsg string
	}{
		{
			name: "ListDreams with an empty workspaceID",
			call: func(ctx context.Context, client *Client) error {
				return errorOf(client.ListDreams(ctx, "", nil, nil))
			},
			errMsg: "workspaceID is required",
		},
		{
			name: "GetDream with an empty workspaceID",
			call: func(ctx context.Context, client *Client) error {
				return errorOf(client.GetDream(ctx, "", "run"))
			},
			errMsg: "workspaceID is required",
		},
		{
			name: "GetDream with an empty runID",
			call: func(ctx context.Context, client *Client) error {
				return errorOf(client.GetDream(ctx, "workspace", ""))
			},
			errMsg: "runID is required",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var requests int
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests++
			}))
			u, _ := url.Parse(server.URL)

			err := tt.call(t.Context(), New(&Options{BaseURL: u}))
			// Close waits for the handler, so what it recorded can be read.
			server.Close()

			if err == nil || err.Error() != tt.errMsg {
				t.Errorf("error = %v, want %q", err, tt.errMsg)
			}
			if requests != 0 {
				t.Errorf("%d requests sent, want none", requests)
			}
		})
	}
}

// A run Honcho does not find, unknown or in another workspace, comes back as an error
// holding the detail of the 404 response.
func TestGetDreamNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `{"detail":"Dream run unknown-run not found"}`)
	}))
	defer server.Close()
	u, _ := url.Parse(server.URL)

	_, err := New(&Options{BaseURL: u}).GetDream(t.Context(), "workspace", "unknown-run")
	if err == nil {
		t.Fatal("GetDream() error = nil, want an error")
	}
	if !strings.HasPrefix(err.Error(), "failed to get dream: ") {
		t.Errorf("error = %q, want it prefixed with %q", err, "failed to get dream: ")
	}
	if !strings.Contains(err.Error(), "Dream run unknown-run not found") {
		t.Errorf("error = %q, want it to hold %q", err, "Dream run unknown-run not found")
	}
}
