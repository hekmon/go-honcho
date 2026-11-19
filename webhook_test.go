package honcho

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

// ListWebhookEndpoints sends its paging options in the query string of a GET
// request without a body, each only when set, and sends no request when
// validation refuses them. Regression test: the options went in the JSON body
// of the GET request, where Honcho does not read them, so every call got the
// first page at the default size.
func TestListWebhookEndpointsPaging(t *testing.T) {
	tests := []struct {
		name      string
		opts      *ListWebhookEndpointsOptions
		wantQuery string
		errMsg    string
	}{
		{name: "page and size", opts: &ListWebhookEndpointsOptions{Page: 2, Size: 10}, wantQuery: "page=2&size=10"},
		{name: "nil options"},
		{name: "zero-value options", opts: &ListWebhookEndpointsOptions{}},
		{name: "size over the limit", opts: &ListWebhookEndpointsOptions{Size: 101}, errMsg: "size must be between 1 and 100"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var (
				requests            int
				method, query, body string
			)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				data, err := io.ReadAll(r.Body)
				if err != nil {
					t.Errorf("failed to read the request body: %v", err)
				}
				requests++
				method, query, body = r.Method, r.URL.RawQuery, string(data)
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprint(w, `{"items":[],"total":0,"page":1,"size":50,"pages":0}`)
			}))
			u, _ := url.Parse(server.URL)
			_, err := New(&Options{BaseURL: u}).ListWebhookEndpoints(t.Context(), "workspace", tt.opts)
			// Close waits for the handler, so what it recorded can be read.
			server.Close()

			if tt.errMsg != "" {
				if err == nil || err.Error() != tt.errMsg {
					t.Errorf("ListWebhookEndpoints() error = %v, want %q", err, tt.errMsg)
				}
				if requests != 0 {
					t.Errorf("%d requests sent, want none", requests)
				}
				return
			}
			if err != nil {
				t.Fatalf("ListWebhookEndpoints() error = %v, want nil", err)
			}
			if requests != 1 {
				t.Fatalf("%d requests sent, want 1", requests)
			}
			if method != http.MethodGet {
				t.Errorf("method = %s, want %s", method, http.MethodGet)
			}
			if query != tt.wantQuery {
				t.Errorf("query = %q, want %q", query, tt.wantQuery)
			}
			if body != "" {
				t.Errorf("body = %q, want it empty", body)
			}
		})
	}
}

// ListWebhookEndpoints decodes the endpoints of a page into a slice of values, in the order the
// server sent them. Regression test: the items of the page were pointers to the endpoints.
func TestListWebhookEndpointsItems(t *testing.T) {
	const body = `{"items":[` +
		`{"id":"e1","workspace_id":"workspace","url":"https://example.test/webhook","created_at":"2026-11-18T16:09:28.620049Z"},` +
		`{"id":"e2","workspace_id":"workspace","url":"https://example.test/other","created_at":"2026-11-19T10:00:00Z"}` +
		`],"total":2,"page":1,"size":50,"pages":1}`
	client, _ := scopeServer(t, http.StatusOK, body)

	page, err := client.ListWebhookEndpoints(t.Context(), "workspace", nil)
	if err != nil {
		t.Fatalf("ListWebhookEndpoints: %v", err)
	}

	checkItems(t, page.Items, []WebhookEndpoint{
		{ID: "e1", WorkspaceID: "workspace", URL: "https://example.test/webhook", CreatedAt: time.Date(2026, time.November, 18, 16, 9, 28, 620049000, time.UTC)},
		{ID: "e2", WorkspaceID: "workspace", URL: "https://example.test/other", CreatedAt: time.Date(2026, time.November, 19, 10, 0, 0, 0, time.UTC)},
	})
}
