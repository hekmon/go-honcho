package honcho

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
)

const (
	// eventStream is the content type a real Honcho server streams an answer with.
	eventStream = "text/event-stream; charset=utf-8"
	// finalFrame is the frame a real Honcho server ends a streamed answer with.
	finalFrame = `data: {"done": true}` + "\n\n"
	// streamTimeout is how long a test waits for what must happen at once.
	streamTimeout = 5 * time.Second
)

// deltaFrame is the frame a real Honcho server sends a chunk of an answer in.
func deltaFrame(chunk string) string {
	content, _ := json.Marshal(chunk)
	return `data: {"delta": {"content": ` + string(content) + `}, "done": false}` + "\n\n"
}

// streamingClient returns a client of a server answering every request with the
// given event stream.
func streamingClient(t *testing.T, stream string) *Client {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", eventStream)
		fmt.Fprint(w, stream)
	}))
	t.Cleanup(server.Close)
	u, _ := url.Parse(server.URL)
	return New(&Options{BaseURL: u})
}

// collectChatStream ranges over a ChatStream to its end and returns the chunks
// and the errors it yielded. An error must come alone, and last.
func collectChatStream(t *testing.T, client *Client) (chunks []string, errs []error) {
	t.Helper()
	for chunk, err := range client.ChatStream(t.Context(), "workspace", "alice", DialecticOptions{Query: "What is my cat's name?"}) {
		if len(errs) > 0 {
			t.Errorf("yielded (%q, %v) after the error %v", chunk, err, errs[0])
		}
		if err != nil {
			if chunk != "" {
				t.Errorf("error %v yielded with the chunk %q", err, chunk)
			}
			errs = append(errs, err)
			continue
		}
		chunks = append(chunks, chunk)
	}
	return
}

// ChatStream sends the query as a POST to the chat path, asking for a stream
// whatever the caller set, and yields the non-empty chunks in order up to the
// final frame, which can carry extra keys. Regression test: a streamed answer
// could not be read, Chat returning "unsupported result type:
// **honcho.DialecticResponse" after the server had done the work.
func TestChatStream(t *testing.T) {
	var method, path, accept, body string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("failed to read the request body: %v", err)
		}
		method, path, accept, body = r.Method, r.URL.Path, r.Header.Get("Accept"), string(data)
		w.Header().Set("Content-Type", eventStream)
		fmt.Fprint(w, deltaFrame("Your cat's name "), deltaFrame(""), deltaFrame("is **Marzipan**."))
		fmt.Fprint(w, `data: {"done": true, "evidence": [{"id": "conclusion"}]}`+"\n\n")
	}))
	defer server.Close()
	u, _ := url.Parse(server.URL)

	chunks, errs := collectChatStream(t, New(&Options{BaseURL: u}))
	// Close waits for the handler, so what it recorded can be read.
	server.Close()

	if len(errs) != 0 {
		t.Errorf("errors = %v, want none", errs)
	}
	if want := []string{"Your cat's name ", "is **Marzipan**."}; !slices.Equal(chunks, want) {
		t.Errorf("chunks = %q, want %q", chunks, want)
	}
	if method != http.MethodPost {
		t.Errorf("method = %s, want %s", method, http.MethodPost)
	}
	if want := "/v3/workspaces/workspace/peers/alice/chat"; path != want {
		t.Errorf("path = %s, want %s", path, want)
	}
	if want := "text/event-stream"; accept != want {
		t.Errorf("Accept = %q, want %q", accept, want)
	}
	if !strings.Contains(body, `"stream":true`) {
		t.Errorf("body = %s, want it to hold \"stream\":true", body)
	}
}

// ChatStream reads the stream line by line: the last line needs no newline, and
// blank lines and lines that are not data fields are skipped. Reading stops at
// the final frame, after yielding the chunk it may hold.
func TestChatStreamLines(t *testing.T) {
	const delta = `data: {"delta": {"content": "Marzipan"}, "done": false}`
	tests := []struct{ name, stream string }{
		{name: "last line without a newline", stream: delta + "\n\n" + `data: {"done": true}`},
		{name: "carriage returns", stream: delta + "\r\n\r\n" + `data: {"done": true}` + "\r\n\r\n"},
		{name: "no space after the colon", stream: `data:{"delta":{"content":"Marzipan"},"done":false}` + "\n\n" + `data:{"done":true}` + "\n\n"},
		{name: "other lines", stream: ": keep-alive\n\nevent: message\nid: 1\nretry: 1000\ndatabase: none\ndata\ndata:\n\n" + delta + "\n\n" + finalFrame},
		{name: "chunk in the final frame", stream: `data: {"delta": {"content": "Marzipan"}, "done": true}` + "\n\n"},
		{name: "frames after the final frame", stream: delta + "\n\n" + finalFrame + deltaFrame("ignored") + "data: not read\n\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chunks, errs := collectChatStream(t, streamingClient(t, tt.stream))
			if len(errs) != 0 {
				t.Errorf("errors = %v, want none", errs)
			}
			if want := []string{"Marzipan"}; !slices.Equal(chunks, want) {
				t.Errorf("chunks = %q, want %q", chunks, want)
			}
		})
	}
}

// A frame larger than the 64 KB a default bufio.Scanner holds arrives whole.
func TestChatStreamLargeFrame(t *testing.T) {
	answer := strings.Repeat("x", 256*1024)

	chunks, errs := collectChatStream(t, streamingClient(t, deltaFrame(answer)+finalFrame))

	if len(errs) != 0 {
		t.Errorf("errors = %v, want none", errs)
	}
	if got := strings.Join(chunks, ""); len(chunks) != 1 || got != answer {
		t.Errorf("got %d chunks holding %d bytes, want 1 chunk holding the %d bytes of the answer", len(chunks), len(got), len(answer))
	}
}

// A stream that ends without its final frame is an error, never a complete
// answer: the chunks read so far are yielded, then exactly one error. A server
// that fails once it has started the response ends it this way, with a 200.
func TestChatStreamIncomplete(t *testing.T) {
	tests := []struct {
		name   string
		status int
		stream string
		abort  bool
		want   []string
	}{
		{name: "after some chunks", status: http.StatusOK, stream: deltaFrame("Your cat's ") + deltaFrame("name"), want: []string{"Your cat's ", "name"}},
		{name: "empty body", status: http.StatusOK},
		{name: "last frame not final", status: http.StatusOK, stream: `data: {"done": false}`},
		{name: "connection lost", status: http.StatusOK, stream: deltaFrame("Your cat's "), abort: true, want: []string{"Your cat's "}},
		// Client.request does not read the body of these statuses.
		{name: "no content", status: http.StatusNoContent},
		{name: "accepted", status: http.StatusAccepted},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", eventStream)
				w.WriteHeader(tt.status)
				if tt.stream != "" {
					fmt.Fprint(w, tt.stream)
				}
				if tt.abort {
					w.(http.Flusher).Flush()
					panic(http.ErrAbortHandler)
				}
			}))
			defer server.Close()
			u, _ := url.Parse(server.URL)

			chunks, errs := collectChatStream(t, New(&Options{BaseURL: u}))

			if !slices.Equal(chunks, tt.want) {
				t.Errorf("chunks = %q, want %q", chunks, tt.want)
			}
			if len(errs) != 1 {
				t.Errorf("errors = %v, want exactly one", errs)
			}
		})
	}
}

// A malformed frame is an error that ends the stream: the chunks before it are
// yielded, and nothing after it is read.
func TestChatStreamMalformedFrame(t *testing.T) {
	for _, frame := range []string{
		`{"delta": {"content": "cut sho`,
		`not json`,
		`null`,
		`["Marzipan"]`,
		`{"delta": "Marzipan"}`,
		`{"done": "yes"}`,
	} {
		t.Run(frame, func(t *testing.T) {
			stream := deltaFrame("Your cat's ") + "data: " + frame + "\n\n" + deltaFrame("name") + finalFrame

			chunks, errs := collectChatStream(t, streamingClient(t, stream))

			if want := []string{"Your cat's "}; !slices.Equal(chunks, want) {
				t.Errorf("chunks = %q, want %q", chunks, want)
			}
			if len(errs) != 1 {
				t.Errorf("errors = %v, want exactly one", errs)
			}
		})
	}
}

// A 422 comes back as a single error matching *HTTPValidationError.
func TestChatStreamValidationError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnprocessableEntity)
		fmt.Fprint(w, `{"detail":"Peer alice cannot be queried"}`)
	}))
	defer server.Close()
	u, _ := url.Parse(server.URL)

	chunks, errs := collectChatStream(t, New(&Options{BaseURL: u}))

	if len(chunks) != 0 {
		t.Errorf("chunks = %q, want none", chunks)
	}
	if len(errs) != 1 {
		t.Fatalf("errors = %v, want exactly one", errs)
	}
	var valErr *HTTPValidationError
	if !errors.As(errs[0], &valErr) || valErr.Message != "Peer alice cannot be queried" {
		t.Errorf("error = %v, want an *HTTPValidationError holding its detail", errs[0])
	}
}

// Another error status comes back as a single error holding the status and the
// body of the response.
func TestChatStreamErrorStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "the dialectic is unavailable", http.StatusServiceUnavailable)
	}))
	defer server.Close()
	u, _ := url.Parse(server.URL)

	chunks, errs := collectChatStream(t, New(&Options{BaseURL: u}))

	if len(chunks) != 0 {
		t.Errorf("chunks = %q, want none", chunks)
	}
	if len(errs) != 1 {
		t.Fatalf("errors = %v, want exactly one", errs)
	}
	for _, want := range []string{"503 Service Unavailable", "the dialectic is unavailable"} {
		if !strings.Contains(errs[0].Error(), want) {
			t.Errorf("error = %v, want it to hold %q", errs[0], want)
		}
	}
}

// ChatStream refuses what Chat refuses with a single error, and sends no request.
func TestChatStreamInvalidRequest(t *testing.T) {
	tests := []struct{ name, workspaceID, peerID, query, errMsg string }{
		{name: "empty query", workspaceID: "workspace", peerID: "alice", errMsg: "query is required"},
		{name: "empty workspace ID", peerID: "alice", query: "What is my cat's name?", errMsg: "workspaceID is required"},
		{name: "empty peer ID", workspaceID: "workspace", query: "What is my cat's name?", errMsg: "peerID is required"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var requests int
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests++
			}))
			defer server.Close()
			u, _ := url.Parse(server.URL)

			var yields int
			var got error
			for _, err := range New(&Options{BaseURL: u}).ChatStream(t.Context(), tt.workspaceID, tt.peerID, DialecticOptions{Query: tt.query}) {
				yields++
				got = err
			}
			// Close waits for the handler, so what it recorded can be read.
			server.Close()

			if yields != 1 || got == nil || got.Error() != tt.errMsg {
				t.Errorf("yielded %d times, last error = %v, want the single error %q", yields, got, tt.errMsg)
			}
			if requests != 0 {
				t.Errorf("%d requests sent, want none", requests)
			}
		})
	}
}

// ChatStream returns at once when the caller breaks out of the loop, and when
// the final frame arrives, while the server keeps the stream open, and the
// server sees the request cancelled. Client.request drains the body after its
// handler, which would otherwise wait for as long as the server holds on.
func TestChatStreamHeldOpen(t *testing.T) {
	tests := []struct {
		name      string
		stream    string
		breakLoop bool
	}{
		{name: "break out of the loop", stream: deltaFrame("Marzipan") + deltaFrame("never read"), breakLoop: true},
		{name: "final frame", stream: deltaFrame("Marzipan") + finalFrame},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cancelled, release := make(chan struct{}), make(chan struct{})
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// The server only watches the connection once the request body is read.
				if _, err := io.Copy(io.Discard, r.Body); err != nil {
					t.Errorf("failed to read the request body: %v", err)
				}
				w.Header().Set("Content-Type", eventStream)
				fmt.Fprint(w, tt.stream)
				w.(http.Flusher).Flush()
				select {
				case <-r.Context().Done():
					close(cancelled)
				case <-release:
				}
			}))
			defer server.Close()
			// Runs before Close, which waits for the handler.
			defer close(release)
			u, _ := url.Parse(server.URL)
			client := New(&Options{BaseURL: u})

			var (
				chunks []string
				errs   []error
			)
			finished := make(chan struct{})
			go func() {
				defer close(finished)
				for chunk, err := range client.ChatStream(t.Context(), "workspace", "alice", DialecticOptions{Query: "What is my cat's name?"}) {
					if err != nil {
						errs = append(errs, err)
						continue
					}
					chunks = append(chunks, chunk)
					if tt.breakLoop {
						break
					}
				}
			}()

			select {
			case <-finished:
			case <-time.After(streamTimeout):
				t.Fatal("ChatStream did not return while the server kept the stream open")
			}
			select {
			case <-cancelled:
			case <-time.After(streamTimeout):
				t.Fatal("the server did not see the request cancelled")
			}
			if len(errs) != 0 {
				t.Errorf("errors = %v, want none", errs)
			}
			if want := []string{"Marzipan"}; !slices.Equal(chunks, want) {
				t.Errorf("chunks = %q, want %q", chunks, want)
			}
		})
	}
}

// Cancelling the caller's context ends a stream the server keeps open, with a
// single error matching context.Canceled.
func TestChatStreamContextCancelled(t *testing.T) {
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", eventStream)
		fmt.Fprint(w, deltaFrame("Marzipan"))
		w.(http.Flusher).Flush()
		<-release
	}))
	defer server.Close()
	// Runs before Close, which waits for the handler.
	defer close(release)
	u, _ := url.Parse(server.URL)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	var errs []error
	finished := make(chan struct{})
	go func() {
		defer close(finished)
		for _, err := range New(&Options{BaseURL: u}).ChatStream(ctx, "workspace", "alice", DialecticOptions{Query: "What is my cat's name?"}) {
			if err != nil {
				errs = append(errs, err)
			}
			cancel()
		}
	}()

	select {
	case <-finished:
	case <-time.After(streamTimeout):
		t.Fatal("ChatStream did not return once its context was cancelled")
	}
	if len(errs) != 1 || !errors.Is(errs[0], context.Canceled) {
		t.Errorf("errors = %v, want a single error matching context.Canceled", errs)
	}
}

// A streaming request does not leave its connection to be reused, and the
// requests after it share another one. Regression test: the connection of a
// stream went back to the pool, so after a server failed a stream and closed the
// connection, the next POST failed with "Post ...: EOF", as Go's transport does
// not retry a POST.
func TestChatStreamConnectionNotReused(t *testing.T) {
	var (
		mu          sync.Mutex
		closes      []bool
		remotes     []string
		connections int
	)
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := io.Copy(io.Discard, r.Body); err != nil {
			t.Errorf("failed to read the request body: %v", err)
		}
		mu.Lock()
		closes, remotes = append(closes, r.Close), append(remotes, r.RemoteAddr)
		mu.Unlock()
		if r.Header.Get("Accept") != "text/event-stream" {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"content":"Your cat's name is **Marzipan**."}`)
			return
		}
		// Written in one go, the end of the response reaches the client with the
		// final frame, which leaves the connection fit for reuse.
		w.Header().Set("Content-Type", eventStream)
		fmt.Fprint(w, deltaFrame("Marzipan"), finalFrame)
	}))
	server.Config.ConnState = func(_ net.Conn, state http.ConnState) {
		if state == http.StateNew {
			mu.Lock()
			connections++
			mu.Unlock()
		}
	}
	server.Start()
	defer server.Close()
	u, _ := url.Parse(server.URL)
	client := New(&Options{BaseURL: u})

	chunks, errs := collectChatStream(t, client)
	if len(errs) != 0 {
		t.Fatalf("errors = %v, want none", errs)
	}
	if want := []string{"Marzipan"}; !slices.Equal(chunks, want) {
		t.Errorf("chunks = %q, want %q", chunks, want)
	}
	for range 2 {
		if _, err := client.Chat(t.Context(), "workspace", "alice", DialecticOptions{Query: "What is my cat's name?"}); err != nil {
			t.Fatalf("Chat() error = %v, want nil", err)
		}
	}
	// Close waits for the handlers, so what they recorded can be read.
	server.Close()
	mu.Lock()
	defer mu.Unlock()

	if want := []bool{true, false, false}; !slices.Equal(closes, want) {
		t.Errorf("r.Close of the stream and of the two requests after it = %v, want %v", closes, want)
	}
	if len(remotes) != 3 || remotes[1] != remotes[2] {
		t.Errorf("requests came from %v, want the last two to share a connection", remotes)
	}
	if connections != 2 {
		t.Errorf("%d connections opened, want 2: one for the stream, one shared by the two requests after it", connections)
	}
}

// Chat refuses to stream before sending anything, and points to ChatStream.
// Regression test: the request went out, the server did the work, and the call
// returned "unsupported result type: **honcho.DialecticResponse".
func TestChatRefusesStream(t *testing.T) {
	var requests int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		w.Header().Set("Content-Type", eventStream)
		fmt.Fprint(w, deltaFrame("Marzipan"), finalFrame)
	}))
	defer server.Close()
	u, _ := url.Parse(server.URL)

	result, err := New(&Options{BaseURL: u}).Chat(t.Context(), "workspace", "alice", DialecticOptions{Query: "What is my cat's name?", Stream: true})
	// Close waits for the handler, so what it recorded can be read.
	server.Close()

	if err == nil || !strings.Contains(err.Error(), "ChatStream") {
		t.Errorf("Chat() error = %v, want an error naming ChatStream", err)
	}
	if result != nil {
		t.Errorf("Chat() result = %+v, want nil", result)
	}
	if requests != 0 {
		t.Errorf("%d requests sent, want none", requests)
	}
}

// Chat without Stream still asks for a JSON answer and decodes it.
func TestChatNotStreamed(t *testing.T) {
	var accept, body string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("failed to read the request body: %v", err)
		}
		accept, body = r.Header.Get("Accept"), string(data)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"content":"Your cat's name is **Marzipan**."}`)
	}))
	defer server.Close()
	u, _ := url.Parse(server.URL)

	result, err := New(&Options{BaseURL: u}).Chat(t.Context(), "workspace", "alice", DialecticOptions{Query: "What is my cat's name?"})
	// Close waits for the handler, so what it recorded can be read.
	server.Close()

	if err != nil {
		t.Fatalf("Chat() error = %v, want nil", err)
	}
	if want := "Your cat's name is **Marzipan**."; result.Content == nil || *result.Content != want {
		t.Errorf("Content = %v, want %q", result.Content, want)
	}
	if want := "application/json"; accept != want {
		t.Errorf("Accept = %q, want %q", accept, want)
	}
	if strings.Contains(body, "stream") {
		t.Errorf("body = %s, want it without the stream field", body)
	}
}

// recallQuery is the query the tests of the recall restrictions send.
const recallQuery = "What is my cat's name?"

// recallRoutes are the routes restricting what they recall with a scope and filters.
var recallRoutes = []struct {
	name, path string
	// response is what the server answers the route with
	response string
	// unrestricted is the body the route sends without any restriction, less its closing brace
	unrestricted string
	// call sends a request with the given restrictions and returns its error
	call func(ctx context.Context, client *Client, scope *ScopeOption, filters map[string]any) error
}{
	{
		name:         "Chat",
		path:         "/v3/workspaces/workspace/peers/alice/chat",
		response:     `{"content":"Your cat's name is **Marzipan**."}`,
		unrestricted: `{"query":"What is my cat's name?"`,
		call: func(ctx context.Context, client *Client, scope *ScopeOption, filters map[string]any) error {
			_, err := client.Chat(ctx, "workspace", "alice", DialecticOptions{Query: recallQuery, Scope: scope, Filters: filters})
			return err
		},
	},
	{
		name:         "ChatStream",
		path:         "/v3/workspaces/workspace/peers/alice/chat",
		response:     deltaFrame("Marzipan") + finalFrame,
		unrestricted: `{"query":"What is my cat's name?","stream":true`,
		call: func(ctx context.Context, client *Client, scope *ScopeOption, filters map[string]any) error {
			for _, err := range client.ChatStream(ctx, "workspace", "alice", DialecticOptions{Query: recallQuery, Scope: scope, Filters: filters}) {
				if err != nil {
					return err
				}
			}
			return nil
		},
	},
	{
		name:         "GetRepresentation",
		path:         "/v3/workspaces/workspace/peers/alice/representation",
		response:     `{"representation":"alice has a cat named Marzipan."}`,
		unrestricted: `{"include_most_frequent":null`,
		call: func(ctx context.Context, client *Client, scope *ScopeOption, filters map[string]any) error {
			_, err := client.GetRepresentation(ctx, "workspace", "alice", PeerRepresentationGet{Scope: scope, Filters: filters})
			return err
		},
	},
}

// Chat, ChatStream and GetRepresentation send the scope and the filters they are given in the request
// body, and leave them out when they are not set. One scope is sent as a string and a union as an
// array, even of a single name, as the server reads the two differently. A filter allowing no session
// and a union of no names are sent too, and so are a scope and filters together: what they recall, or
// refusing them, is up to the server.
func TestRecallRestrictionsRequest(t *testing.T) {
	sessions := map[string]any{"session_id": []string{"s1", "s2"}}
	tests := []struct {
		name    string
		scope   *ScopeOption
		filters map[string]any
		// wantFields is what the restriction adds to the request body
		wantFields string
	}{
		{name: "no restriction"},
		{name: "one scope", scope: OneScope("team"), wantFields: `,"scope":"team"`},
		{name: "union of two scopes", scope: ScopeUnion("team", "music"), wantFields: `,"scope":["team","music"]`},
		{name: "union of one scope", scope: ScopeUnion("team"), wantFields: `,"scope":["team"]`},
		{name: "union of no names", scope: ScopeUnion(), wantFields: `,"scope":[]`},
		{name: "zero value scope", scope: &ScopeOption{}, wantFields: `,"scope":[]`},
		{name: "filter on one session", filters: map[string]any{"session_id": "s2"}, wantFields: `,"filters":{"session_id":"s2"}`},
		{name: "filter on a list of sessions", filters: sessions, wantFields: `,"filters":{"session_id":["s1","s2"]}`},
		{
			name:       "filter with the in operator",
			filters:    map[string]any{"session_id": map[string]any{"in": []string{"s1"}}},
			wantFields: `,"filters":{"session_id":{"in":["s1"]}}`,
		},
		{name: "filter allowing no session", filters: map[string]any{"session_id": []string{}}, wantFields: `,"filters":{"session_id":[]}`},
		{name: "scope and filters", scope: OneScope("team"), filters: sessions, wantFields: `,"scope":"team","filters":{"session_id":["s1","s2"]}`},
	}
	for _, route := range recallRoutes {
		for _, tt := range tests {
			t.Run(route.name+"/"+tt.name, func(t *testing.T) {
				client, received := scopeServer(t, http.StatusOK, route.response)

				if err := route.call(t.Context(), client, tt.scope, tt.filters); err != nil {
					t.Fatalf("%s: %v", route.name, err)
				}

				checkScopeRequest(t, received(), http.MethodPost, route.path, url.Values{}, route.unrestricted+tt.wantFields+`}`)
			})
		}
	}
}

// Chat, ChatStream and GetRepresentation refuse a non-nil empty Filters before sending anything: it
// would be left out of the request, turning a restriction the caller asked for into unrestricted
// recall. Nil Filters ask for no restriction and are accepted.
func TestRecallRefusesEmptyFilters(t *testing.T) {
	for _, route := range recallRoutes {
		t.Run(route.name+"/empty filters", func(t *testing.T) {
			client, received := scopeServer(t, http.StatusOK, route.response)

			err := route.call(t.Context(), client, nil, map[string]any{})

			if want := "filters must hold at least one filter: leave it nil for no restriction"; err == nil || err.Error() != want {
				t.Errorf("error = %v, want %q", err, want)
			}
			if requests := received(); len(requests) != 0 {
				t.Errorf("%d requests sent, want none", len(requests))
			}
		})
		t.Run(route.name+"/nil filters", func(t *testing.T) {
			client, received := scopeServer(t, http.StatusOK, route.response)

			if err := route.call(t.Context(), client, nil, nil); err != nil {
				t.Errorf("error = %v, want nil", err)
			}
			if requests := received(); len(requests) != 1 {
				t.Errorf("%d requests sent, want 1", len(requests))
			}
		})
	}
}

// GetAllPeers sends reverse=true when Reverse is set and the kind in the body when Kind is
// set. Left unset, neither is sent, so the server applies its defaults: oldest peer first,
// and regular peers only.
func TestGetAllPeersRequest(t *testing.T) {
	filters := map[string]any{"metadata": map[string]any{"role": "agent"}}
	tests := []struct {
		name      string
		req       *PeerGet
		opts      *GetAllPeersOptions
		wantQuery url.Values
		wantBody  string
	}{
		{name: "nil request and nil options", wantQuery: url.Values{}, wantBody: `null`},
		{
			name:      "empty request and empty options",
			req:       &PeerGet{},
			opts:      &GetAllPeersOptions{},
			wantQuery: url.Values{},
			wantBody:  `{}`,
		},
		{name: "reverse set", opts: &GetAllPeersOptions{Reverse: true}, wantQuery: url.Values{"reverse": {"true"}}, wantBody: `null`},
		{name: "reverse not set", opts: &GetAllPeersOptions{Page: 2}, wantQuery: url.Values{"page": {"2"}}, wantBody: `null`},
		{name: "scope peers", req: &PeerGet{Kind: PeerKindScope}, wantQuery: url.Values{}, wantBody: `{"kind":"scope"}`},
		{name: "every peer", req: &PeerGet{Kind: PeerKindAll}, wantQuery: url.Values{}, wantBody: `{"kind":"all"}`},
		{
			name:      "kind not set",
			req:       &PeerGet{Filters: filters},
			wantQuery: url.Values{},
			wantBody:  `{"filters":{"metadata":{"role":"agent"}}}`,
		},
		{
			name:      "kind with filters and reverse",
			req:       &PeerGet{Filters: filters, Kind: PeerKindAll},
			opts:      &GetAllPeersOptions{Reverse: true},
			wantQuery: url.Values{"reverse": {"true"}},
			wantBody:  `{"filters":{"metadata":{"role":"agent"}},"kind":"all"}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, received := scopeServer(t, http.StatusOK, emptyPageJSON)

			if _, err := client.GetAllPeers(t.Context(), "workspace", tt.req, tt.opts); err != nil {
				t.Fatalf("GetAllPeers: %v", err)
			}

			checkScopeRequest(t, received(), http.MethodPost, "/v3/workspaces/workspace/peers/list", tt.wantQuery, tt.wantBody)
		})
	}
}

// GetSessionsForPeer sends reverse=true when Reverse is set, and otherwise no reverse
// parameter at all, so the server keeps its default order, oldest session first.
func TestGetSessionsForPeerReverse(t *testing.T) {
	tests := []struct {
		name      string
		opts      *GetSessionsForPeerOptions
		wantQuery url.Values
	}{
		{name: "reverse set", opts: &GetSessionsForPeerOptions{Reverse: true}, wantQuery: url.Values{"reverse": {"true"}}},
		{name: "reverse not set", opts: &GetSessionsForPeerOptions{Page: 2}, wantQuery: url.Values{"page": {"2"}}},
		{name: "empty options", opts: &GetSessionsForPeerOptions{}, wantQuery: url.Values{}},
		{name: "nil options", wantQuery: url.Values{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, received := scopeServer(t, http.StatusOK, scopeSessionPageJSON)

			if _, err := client.GetSessionsForPeer(t.Context(), "workspace", "alice", &SessionGet{}, tt.opts); err != nil {
				t.Fatalf("GetSessionsForPeer: %v", err)
			}

			checkScopeRequest(t, received(), http.MethodPost, "/v3/workspaces/workspace/peers/alice/sessions", tt.wantQuery, `{}`)
		})
	}
}
