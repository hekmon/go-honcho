package honcho

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

// A 422 comes back as an *HTTPValidationError holding its detail, whatever its
// form: Honcho's own errors, a text, and FastAPI's request validation errors,
// a list. Regression test: every 422 read "failed to decode validation error:
// EOF", its body printed to the standard output.
func TestValidationError(t *testing.T) {
	for _, tt := range []struct{ body, want string }{
		{`{"detail":"Peer name(s) ['alice@example.com'] must match pattern ^[a-zA-Z0-9_-]+$"}`, "must match pattern"},
		{`{"detail":[{"loc":["query","tokens"],"msg":"Input should be less than or equal to 100000","type":"less_than_equal"}]}`,
			"query.tokens: Input should be less than or equal to 100000"},
	} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnprocessableEntity)
			fmt.Fprint(w, tt.body)
		}))
		u, _ := url.Parse(server.URL)
		client := New(&Options{BaseURL: u})
		_, err := client.CreateMessagesForSession(t.Context(), "workspace", "session", MessageBatchCreate{
			Messages: []MessageCreate{{Content: "Hello.", PeerID: "alice"}},
		})
		server.Close()
		var valErr *HTTPValidationError
		if !errors.As(err, &valErr) || !strings.Contains(err.Error(), tt.want) || strings.Count(err.Error(), "validation error") != 1 {
			t.Errorf("error = %v, want an *HTTPValidationError holding %q, prefixed once", err, tt.want)
		}
	}
}

// The error of a failed request keeps its cause in its chain, for errors.Is and
// errors.As to find through the wrapping of any method, and its message says once
// that the request failed. Regression test: request flattened its causes into
// text, so a cancelled call did not match context.Canceled, and read "failed to
// execute request: failed to execute request: Post ...: context canceled".
func TestRequestErrorChain(t *testing.T) {
	const timeout = 50 * time.Millisecond
	// A server that does not answer: its handlers wait for the end of the test.
	release := make(chan struct{})
	silent := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		<-release
	}))
	defer silent.Close()
	// Runs before Close, which waits for the handlers.
	defer close(release)
	silentURL, _ := url.Parse(silent.URL)
	client := New(&Options{BaseURL: silentURL})

	// A server answering with a body that is not the JSON it announces.
	invalid := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, "not JSON")
	}))
	defer invalid.Close()
	invalidURL, _ := url.Parse(invalid.URL)

	cancelled, cancel := context.WithCancel(t.Context())
	cancel()
	isCancelled := func(err error) bool { return errors.Is(err, context.Canceled) }
	getOrCreateWorkspace := func(ctx context.Context, client *Client) error {
		return errorOf(client.GetOrCreateWorkspace(ctx, CreateWorkspaceRequest{ID: "workspace"}))
	}

	tests := []struct {
		name     string
		call     func() error
		want     string           // what the error must match
		matches  func(error) bool // whether it does
		prefixes int              // how many times the message says "failed to execute request"
	}{
		{
			name:     "cancelled GetOrCreateWorkspace",
			call:     func() error { return getOrCreateWorkspace(cancelled, client) },
			want:     "context.Canceled",
			matches:  isCancelled,
			prefixes: 1,
		},
		{
			name:     "cancelled GetMessage",
			call:     func() error { return errorOf(client.GetMessage(cancelled, "workspace", "session", "message")) },
			want:     "context.Canceled",
			matches:  isCancelled,
			prefixes: 0,
		},
		{
			name: "cancelled ChatStream",
			call: func() error {
				for _, err := range client.ChatStream(cancelled, "workspace", "alice", DialecticOptions{Query: "What is my cat's name?"}) {
					if err != nil {
						return err
					}
				}
				return nil
			},
			want:     "context.Canceled",
			matches:  isCancelled,
			prefixes: 1,
		},
		{
			name: "context deadline",
			call: func() error {
				ctx, cancel := context.WithTimeout(t.Context(), timeout)
				defer cancel()
				return getOrCreateWorkspace(ctx, client)
			},
			want:     "context.DeadlineExceeded",
			matches:  func(err error) bool { return errors.Is(err, context.DeadlineExceeded) },
			prefixes: 1,
		},
		{
			name: "http.Client.Timeout",
			call: func() error {
				return getOrCreateWorkspace(t.Context(), New(&Options{BaseURL: silentURL, HTTP: &http.Client{Timeout: timeout}}))
			},
			want: "a net.Error whose Timeout() is true",
			matches: func(err error) bool {
				var netErr net.Error
				return errors.As(err, &netErr) && netErr.Timeout()
			},
			prefixes: 1,
		},
		{
			name: "invalid JSON response",
			call: func() error {
				return getOrCreateWorkspace(t.Context(), New(&Options{BaseURL: invalidURL}))
			},
			want: "*json.SyntaxError",
			matches: func(err error) bool {
				var syntaxErr *json.SyntaxError
				return errors.As(err, &syntaxErr)
			},
			prefixes: 1,
		},
		{
			name: "body that cannot be encoded",
			call: func() error {
				return errorOf(client.request(t.Context(), http.MethodPost, silentURL, nil, make(chan int), nil))
			},
			want: "*json.UnsupportedTypeError",
			matches: func(err error) bool {
				var typeErr *json.UnsupportedTypeError
				return errors.As(err, &typeErr)
			},
			prefixes: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.call()
			if err == nil {
				t.Fatalf("error = nil, want an error matching %s", tt.want)
			}
			if !tt.matches(err) {
				t.Errorf("error = %v, want an error matching %s", err, tt.want)
			}
			if got := strings.Count(err.Error(), "failed to execute request"); got != tt.prefixes {
				t.Errorf("error = %v, says \"failed to execute request\" %d times, want %d", err, got, tt.prefixes)
			}
		})
	}
}
