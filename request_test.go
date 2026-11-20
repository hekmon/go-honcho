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

// Any status but 200, 201, 202, 204 and 422, such as a 404 or a 502, comes back
// as an *HTTPError holding the status, the request's method and URL, and the
// body as received. Its text gives them on one line, the body trimmed and left
// out when empty, and the wrapped message says once that the request failed. A
// password in the base URL is masked in the URL and in the text. Regression
// test: such a status came back as a text of three lines, "Client request
// failed: 404 Not Found", the URL, password included, and the body, which
// callers searched for "404 Not Found", and the wrapped message said twice that
// the request failed.
func TestHTTPError(t *testing.T) {
	const (
		detail = `{"detail":"Session s not found in workspace w"}`
		secret = "secret" // the password of a base URL
	)
	deleteSession := func(ctx context.Context, client *Client) error { return client.DeleteSession(ctx, "w", "s") }
	tests := []struct {
		name       string
		user       *url.Userinfo // the user of the base URL, if any
		call       func(ctx context.Context, client *Client) error
		statusCode int    // the status the server answers with
		status     string // its text, as the client reads it
		body       string // the body the server sends
		method     string // the method of the request
		url        string // the URL of the error, %s standing for the server's host
		text       string // the text of the error, %s standing for that URL
	}{
		{
			name:       "404 with a detail",
			call:       deleteSession,
			statusCode: http.StatusNotFound,
			status:     "404 Not Found",
			body:       detail + "\n",
			method:     http.MethodDelete,
			url:        "http://%s/v3/workspaces/w/sessions/s",
			text:       "failed to execute request: DELETE %s: 404 Not Found: " + detail,
		},
		{
			name: "502 without a body",
			call: func(ctx context.Context, client *Client) error {
				return errorOf(client.GetOrCreateWorkspace(ctx, CreateWorkspaceRequest{ID: "w"}))
			},
			statusCode: http.StatusBadGateway,
			status:     "502 Bad Gateway",
			method:     http.MethodPost,
			url:        "http://%s/v3/workspaces",
			text:       "failed to execute request: POST %s: 502 Bad Gateway",
		},
		{
			name:       "404 to a base URL holding a password",
			user:       url.UserPassword("user", secret),
			call:       deleteSession,
			statusCode: http.StatusNotFound,
			status:     "404 Not Found",
			method:     http.MethodDelete,
			url:        "http://user:xxxxx@%s/v3/workspaces/w/sessions/s",
			text:       "failed to execute request: DELETE %s: 404 Not Found",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				fmt.Fprint(w, tt.body)
			}))
			defer server.Close()
			u, _ := url.Parse(server.URL)
			u.User = tt.user

			err := tt.call(t.Context(), New(&Options{BaseURL: u}))

			var httpErr *HTTPError
			if !errors.As(err, &httpErr) {
				t.Fatalf("error = %v, want an *HTTPError", err)
			}
			wantURL := fmt.Sprintf(tt.url, u.Host)
			if httpErr.StatusCode != tt.statusCode || httpErr.Status != tt.status {
				t.Errorf("StatusCode, Status = %d, %q, want %d, %q", httpErr.StatusCode, httpErr.Status, tt.statusCode, tt.status)
			}
			if httpErr.Method != tt.method || httpErr.URL != wantURL {
				t.Errorf("Method, URL = %s, %s, want %s, %s", httpErr.Method, httpErr.URL, tt.method, wantURL)
			}
			if string(httpErr.Body) != tt.body {
				t.Errorf("Body = %q, want %q", httpErr.Body, tt.body)
			}
			if want := fmt.Sprintf(tt.text, wantURL); err.Error() != want {
				t.Errorf("error = %q, want %q", err, want)
			}
			if strings.Contains(httpErr.URL, secret) || strings.Contains(err.Error(), secret) {
				t.Errorf("URL = %s, error = %v, want neither to hold the password", httpErr.URL, err)
			}
		})
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
