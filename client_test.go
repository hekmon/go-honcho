package honcho

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

// A client sends its requests through the HTTP client its options hold, and
// New leaves the options as given. Regression test: New built every client on
// http.DefaultClient, whatever the options held, and wrote its defaults into
// them.
func TestNew(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":"workspace","created_at":"2026-10-30T00:00:00Z"}`)
	}))
	defer server.Close()
	u, _ := url.Parse(server.URL)
	var sent int
	options := &Options{BaseURL: u, HTTP: &http.Client{Transport: roundTripper(func(r *http.Request) (*http.Response, error) {
		sent++
		return server.Client().Transport.RoundTrip(r)
	})}}

	if _, err := New(options).GetOrCreateWorkspace(t.Context(), CreateWorkspaceRequest{ID: "workspace"}); err != nil {
		t.Fatalf("GetOrCreateWorkspace: %v", err)
	}
	if sent != 1 {
		t.Errorf("%d requests went through the options' client, want 1", sent)
	}
	var empty Options
	New(&empty)
	if empty.BaseURL != nil || empty.HTTP != nil {
		t.Errorf("New wrote its defaults into the options: %+v", empty)
	}
}

// roundTripper is an http.RoundTripper of a function.
type roundTripper func(*http.Request) (*http.Response, error)

func (f roundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}
