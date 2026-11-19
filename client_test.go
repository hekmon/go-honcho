package honcho

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"runtime"
	"testing"
	"time"
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

// The default HTTP client lets go of an idle connection before Honcho's server
// closes it, and a client given in the options is used as it is. Regression
// test: the default client kept idle connections for 90 seconds, past the
// server's 5 second keep-alive, so a POST written as the server closed its
// connection failed with EOF.
func TestNewIdleConnTimeout(t *testing.T) {
	tests := []struct {
		name    string
		options *Options
	}{
		{name: "nil options", options: nil},
		{name: "empty options", options: &Options{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transport, ok := New(tt.options).http.Transport.(*http.Transport)
			if !ok {
				t.Fatalf("the default client's transport is not an *http.Transport")
			}
			if want := 4 * time.Second; transport.IdleConnTimeout != want {
				t.Errorf("IdleConnTimeout = %v, want %v", transport.IdleConnTimeout, want)
			}
		})
	}

	t.Run("client in the options", func(t *testing.T) {
		const idleConnTimeout = 90 * time.Second
		transport := &http.Transport{IdleConnTimeout: idleConnTimeout}
		httpClient := &http.Client{Transport: transport}

		client := New(&Options{HTTP: httpClient})

		if client.http != httpClient {
			t.Errorf("New used %p, want the options' client %p", client.http, httpClient)
		}
		if httpClient.Transport != transport {
			t.Errorf("New replaced the transport of the options' client")
		}
		if transport.IdleConnTimeout != idleConnTimeout {
			t.Errorf("IdleConnTimeout = %v, want it left at %v", transport.IdleConnTimeout, idleConnTimeout)
		}
	})
}

// NewHTTPClient returns a new client on every call, with a transport of its own
// set to go-honcho's defaults, and New uses such a client when its options hold
// none.
func TestNewHTTPClient(t *testing.T) {
	first, second := NewHTTPClient(), NewHTTPClient()
	if first == second {
		t.Errorf("two calls returned the same client %p", first)
	}
	if first.Transport == second.Transport {
		t.Errorf("two calls returned clients sharing the transport %p", first.Transport)
	}

	tests := []struct {
		name   string
		client *http.Client
	}{
		{name: "first call", client: first},
		{name: "second call", client: second},
		{name: "New with nil options", client: New(nil).http},
		{name: "New with empty options", client: New(&Options{}).http},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.client.Transport == http.DefaultTransport {
				t.Errorf("the client uses http.DefaultTransport")
			}
			transport, ok := tt.client.Transport.(*http.Transport)
			if !ok {
				t.Fatalf("the client's transport is a %T, want an *http.Transport", tt.client.Transport)
			}
			if reflect.ValueOf(transport.Proxy).Pointer() != reflect.ValueOf(http.ProxyFromEnvironment).Pointer() {
				t.Errorf("Proxy is not http.ProxyFromEnvironment")
			}
			if transport.DialContext == nil {
				t.Errorf("DialContext is not set")
			}
			if !transport.ForceAttemptHTTP2 {
				t.Errorf("ForceAttemptHTTP2 = false, want true")
			}
			if want := 100; transport.MaxIdleConns != want {
				t.Errorf("MaxIdleConns = %d, want %d", transport.MaxIdleConns, want)
			}
			if want := runtime.GOMAXPROCS(0) + 1; transport.MaxIdleConnsPerHost != want {
				t.Errorf("MaxIdleConnsPerHost = %d, want %d", transport.MaxIdleConnsPerHost, want)
			}
			if want := 4 * time.Second; transport.IdleConnTimeout != want {
				t.Errorf("IdleConnTimeout = %v, want %v", transport.IdleConnTimeout, want)
			}
			if want := 10 * time.Second; transport.TLSHandshakeTimeout != want {
				t.Errorf("TLSHandshakeTimeout = %v, want %v", transport.TLSHandshakeTimeout, want)
			}
			if want := time.Second; transport.ExpectContinueTimeout != want {
				t.Errorf("ExpectContinueTimeout = %v, want %v", transport.ExpectContinueTimeout, want)
			}
		})
	}
}

// roundTripper is an http.RoundTripper of a function.
type roundTripper func(*http.Request) (*http.Response, error)

func (f roundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}
