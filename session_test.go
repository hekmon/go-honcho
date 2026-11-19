package honcho

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"mime"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"slices"
	"testing"
	"time"
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

// GetSessions sends a POST request to the workspace's sessions list path, with the options set
// in the query and the restrictions set in the body. What is left unset is not sent, so the
// server applies its defaults. An empty scope name is sent as it is, and so are a scope and
// InNoScope together: refusing them is up to the server.
func TestGetSessionsRequest(t *testing.T) {
	scope, emptyScope := "team", ""
	filters := map[string]any{"metadata": map[string]any{"topic": "support"}}
	tests := []struct {
		name      string
		req       *WorkspaceSessionGet
		opts      *GetSessionsOptions
		wantQuery url.Values
		wantBody  string
	}{
		{name: "nil request and nil options", wantQuery: url.Values{}, wantBody: `null`},
		{
			name:      "empty request and empty options",
			req:       &WorkspaceSessionGet{},
			opts:      &GetSessionsOptions{},
			wantQuery: url.Values{},
			wantBody:  `{}`,
		},
		{name: "page only", opts: &GetSessionsOptions{Page: 2}, wantQuery: url.Values{"page": {"2"}}, wantBody: `null`},
		{name: "reverse only", opts: &GetSessionsOptions{Reverse: true}, wantQuery: url.Values{"reverse": {"true"}}, wantBody: `null`},
		{
			name:      "include scopes only",
			opts:      &GetSessionsOptions{IncludeScopes: true},
			wantQuery: url.Values{"include_scopes": {"true"}},
			wantBody:  `null`,
		},
		{
			name:      "filters only",
			req:       &WorkspaceSessionGet{Filters: filters},
			wantQuery: url.Values{},
			wantBody:  `{"filters":{"metadata":{"topic":"support"}}}`,
		},
		{name: "scope only", req: &WorkspaceSessionGet{Scope: &scope}, wantQuery: url.Values{}, wantBody: `{"scope":"team"}`},
		{name: "empty scope name", req: &WorkspaceSessionGet{Scope: &emptyScope}, wantQuery: url.Values{}, wantBody: `{"scope":""}`},
		{name: "in no scope only", req: &WorkspaceSessionGet{InNoScope: true}, wantQuery: url.Values{}, wantBody: `{"in_no_scope":true}`},
		{
			name:      "scope and in no scope",
			req:       &WorkspaceSessionGet{Scope: &scope, InNoScope: true},
			wantQuery: url.Values{},
			wantBody:  `{"scope":"team","in_no_scope":true}`,
		},
		{
			name: "every option with a scope",
			req:  &WorkspaceSessionGet{Filters: filters, Scope: &scope},
			opts: &GetSessionsOptions{Page: 3, Size: 10, Reverse: true, IncludeScopes: true},
			wantQuery: url.Values{
				"page":           {"3"},
				"size":           {"10"},
				"reverse":        {"true"},
				"include_scopes": {"true"},
			},
			wantBody: `{"filters":{"metadata":{"topic":"support"}},"scope":"team"}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, received := scopeServer(t, http.StatusOK, scopeSessionPageJSON)

			if _, err := client.GetSessions(t.Context(), "workspace", tt.req, tt.opts); err != nil {
				t.Fatalf("GetSessions: %v", err)
			}

			checkScopeRequest(t, received(), http.MethodPost, "/v3/workspaces/workspace/sessions/list", tt.wantQuery, tt.wantBody)
		})
	}
}

// Session.Scopes tells apart what the sessions list can return: nil when the field is absent,
// as it is without IncludeScopes, an empty non-nil slice for a session in no scope, and the
// names of the scopes otherwise.
func TestSessionScopes(t *testing.T) {
	const fields = `"id":"s1","uid":"Xv27Pg3BTadRElK_LnbyJ","is_active":true,"workspace_id":"workspace","metadata":{},"configuration":{},"created_at":"2026-11-18T16:09:34.673599Z"`
	tests := []struct {
		name    string
		session string
		want    []string
	}{
		{name: "absent", session: `{` + fields + `}`},
		{name: "in no scope", session: `{` + fields + `,"scopes":[]}`, want: []string{}},
		{name: "in two scopes", session: `{` + fields + `,"scopes":["team","project"]}`, want: []string{"team", "project"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, _ := scopeServer(t, http.StatusOK, `{"items":[`+tt.session+`],"total":1,"page":1,"size":50,"pages":1}`)

			page, err := client.GetSessions(t.Context(), "workspace", nil, &GetSessionsOptions{IncludeScopes: true})
			if err != nil {
				t.Fatalf("GetSessions: %v", err)
			}

			if len(page.Items) != 1 {
				t.Fatalf("%d sessions decoded, want 1", len(page.Items))
			}
			// DeepEqual tells a nil slice from an empty one.
			if got := page.Items[0].Scopes; !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Scopes = %#v, want %#v", got, tt.want)
			}
		})
	}
}

// sessionContextJSON is a session context in the shape of upstream's SessionContext schema.
const sessionContextJSON = `{"id":"session","messages":[]}`

// GetSessionContext sends the scope, and one sessions parameter per entry, in the query of a GET
// request without a body. What is left unset is not sent. A scope and sessions together, or without
// a target, are sent as they are: refusing them is up to the server.
func TestGetSessionContextRestrictions(t *testing.T) {
	scope, emptyScope, target := "team", "", "alice"
	tests := []struct {
		name      string
		opts      *GetSessionContextOptions
		wantQuery url.Values
	}{
		{name: "nil options", wantQuery: url.Values{}},
		{name: "empty options", opts: &GetSessionContextOptions{}, wantQuery: url.Values{}},
		{
			name:      "scope",
			opts:      &GetSessionContextOptions{PeerTarget: &target, Scope: &scope},
			wantQuery: url.Values{"peer_target": {"alice"}, "scope": {"team"}},
		},
		{
			name:      "empty scope name",
			opts:      &GetSessionContextOptions{PeerTarget: &target, Scope: &emptyScope},
			wantQuery: url.Values{"peer_target": {"alice"}, "scope": {""}},
		},
		{
			name:      "one session",
			opts:      &GetSessionContextOptions{PeerTarget: &target, Sessions: []string{"session"}},
			wantQuery: url.Values{"peer_target": {"alice"}, "sessions": {"session"}},
		},
		{
			name:      "three sessions",
			opts:      &GetSessionContextOptions{PeerTarget: &target, Sessions: []string{"session", "s2", "s3"}},
			wantQuery: url.Values{"peer_target": {"alice"}, "sessions": {"session", "s2", "s3"}},
		},
		{
			name:      "scope and sessions without a target",
			opts:      &GetSessionContextOptions{Scope: &scope, Sessions: []string{"session"}},
			wantQuery: url.Values{"scope": {"team"}, "sessions": {"session"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, received := scopeServer(t, http.StatusOK, sessionContextJSON)

			if _, err := client.GetSessionContext(t.Context(), "workspace", "session", tt.opts); err != nil {
				t.Fatalf("GetSessionContext: %v", err)
			}

			checkScopeRequest(t, received(), http.MethodGet, "/v3/workspaces/workspace/sessions/session/context", tt.wantQuery, "")
		})
	}
}

// GetSessionContext refuses a non-nil empty Sessions before sending anything: no sessions parameter
// would be sent, turning the allowlist the caller asked for into unrestricted recall. Nil Sessions
// ask for no restriction and are accepted.
func TestGetSessionContextRefusesEmptySessions(t *testing.T) {
	t.Run("empty sessions", func(t *testing.T) {
		client, received := scopeServer(t, http.StatusOK, sessionContextJSON)

		result, err := client.GetSessionContext(t.Context(), "workspace", "session", &GetSessionContextOptions{Sessions: []string{}})

		if want := "sessions must hold at least one session: leave it nil for no restriction"; err == nil || err.Error() != want {
			t.Errorf("error = %v, want %q", err, want)
		}
		if result != nil {
			t.Errorf("result = %+v, want nil", result)
		}
		if requests := received(); len(requests) != 0 {
			t.Errorf("%d requests sent, want none", len(requests))
		}
	})
	t.Run("nil sessions", func(t *testing.T) {
		client, received := scopeServer(t, http.StatusOK, sessionContextJSON)

		if _, err := client.GetSessionContext(t.Context(), "workspace", "session", &GetSessionContextOptions{}); err != nil {
			t.Errorf("error = %v, want nil", err)
		}
		if requests := received(); len(requests) != 1 {
			t.Errorf("%d requests sent, want 1", len(requests))
		}
	})
}

// sessionJSON is a session in the shape of upstream's Session schema.
const sessionJSON = `{"id":"session","is_active":true,"workspace_id":"workspace","created_at":"2026-10-30T00:00:00Z"}`

// GetOrCreateSession sends the names of the scopes to add the session to, and no scopes field when
// Scopes is not set.
func TestGetOrCreateSessionScopes(t *testing.T) {
	tests := []struct {
		name     string
		scopes   []string
		wantBody string
	}{
		{name: "no scopes", wantBody: `{"id":"session"}`},
		{name: "one scope", scopes: []string{"team"}, wantBody: `{"id":"session","scopes":["team"]}`},
		{name: "two scopes", scopes: []string{"team", "music"}, wantBody: `{"id":"session","scopes":["team","music"]}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, received := scopeServer(t, http.StatusOK, sessionJSON)

			if _, err := client.GetOrCreateSession(t.Context(), "workspace", SessionCreate{ID: "session", Scopes: tt.scopes}); err != nil {
				t.Fatalf("GetOrCreateSession: %v", err)
			}

			checkScopeRequest(t, received(), http.MethodPost, "/v3/workspaces/workspace/sessions", url.Values{}, tt.wantBody)
		})
	}
}

// A session is created with up to 100 scopes, as upstream allows, and more are refused before
// anything is sent.
func TestGetOrCreateSessionScopesLimit(t *testing.T) {
	scopes := make([]string, 101)
	for i := range scopes {
		scopes[i] = fmt.Sprintf("scope-%d", i)
	}
	t.Run("100 scopes", func(t *testing.T) {
		client, received := scopeServer(t, http.StatusOK, sessionJSON)

		if _, err := client.GetOrCreateSession(t.Context(), "workspace", SessionCreate{ID: "session", Scopes: scopes[:100]}); err != nil {
			t.Fatalf("GetOrCreateSession: %v", err)
		}

		want, err := json.Marshal(map[string]any{"id": "session", "scopes": scopes[:100]})
		if err != nil {
			t.Fatalf("failed to marshal the expected body: %v", err)
		}
		checkScopeRequest(t, received(), http.MethodPost, "/v3/workspaces/workspace/sessions", url.Values{}, string(want))
	})
	t.Run("101 scopes", func(t *testing.T) {
		client, received := scopeServer(t, http.StatusOK, sessionJSON)

		_, err := client.GetOrCreateSession(t.Context(), "workspace", SessionCreate{ID: "session", Scopes: scopes})

		if want := "scopes must hold 100 scopes or less"; err == nil || err.Error() != want {
			t.Errorf("error = %v, want %q", err, want)
		}
		if requests := received(); len(requests) != 0 {
			t.Errorf("%d requests sent, want none", len(requests))
		}
	})
}

// uploadedMessagesJSON is a list of one message in the shape of upstream's Message schema, as the
// upload route answers it.
const uploadedMessagesJSON = `[{"id":"m1","content":"line one\nline two","peer_id":"alice","session_id":"session","metadata":{},"created_at":"2026-11-19T10:00:00Z","workspace_id":"workspace","token_count":4}]`

// receivedUpload is what a test server recorded of a multipart upload it received.
type receivedUpload struct {
	method, path, mediaType string
	// values holds the fields of the form and files its files, both by field name.
	values map[string][]string
	files  map[string][]uploadedFile
}

// uploadedFile is a file of a multipart form: the Content-Disposition and Content-Type headers of
// its part, as sent, and its content.
type uploadedFile struct {
	contentDisposition, contentType, content string
}

// uploadServer returns a client of a server parsing every request as a multipart form and answering
// 201 with uploadedMessagesJSON, and a function closing the server and returning the uploads it
// received.
func uploadServer(t *testing.T) (client *Client, received func() []receivedUpload) {
	t.Helper()
	var uploads []receivedUpload
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upload := receivedUpload{method: r.Method, path: r.URL.Path, files: make(map[string][]uploadedFile)}
		var err error
		if upload.mediaType, _, err = mime.ParseMediaType(r.Header.Get("Content-Type")); err != nil {
			t.Errorf("failed to parse the Content-Type: %v", err)
		}
		if err = r.ParseMultipartForm(1 << 20); err != nil {
			t.Errorf("failed to parse the multipart form: %v", err)
		} else {
			upload.values = r.MultipartForm.Value
			for field, headers := range r.MultipartForm.File {
				for _, header := range headers {
					file, err := header.Open()
					if err != nil {
						t.Errorf("failed to open the file %q: %v", header.Filename, err)
						continue
					}
					content, err := io.ReadAll(file)
					_ = file.Close()
					if err != nil {
						t.Errorf("failed to read the file %q: %v", header.Filename, err)
					}
					upload.files[field] = append(upload.files[field], uploadedFile{
						contentDisposition: header.Header.Get("Content-Disposition"),
						contentType:        header.Header.Get("Content-Type"),
						content:            string(content),
					})
				}
			}
		}
		uploads = append(uploads, upload)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, uploadedMessagesJSON)
	}))
	t.Cleanup(server.Close)
	u, _ := url.Parse(server.URL)
	return New(&Options{BaseURL: u}), func() []receivedUpload {
		// Close waits for the handler, so what it recorded can be read.
		server.Close()
		return uploads
	}
}

// checkUploadForm reports an error unless the uploads are a single POST multipart form, sent to the
// upload path of the session, holding notes.txt as its file, of the type detected for its text, and
// exactly the given fields, each sent once. The configuration is compared as decoded JSON, any other
// field as it is.
func checkUploadForm(t *testing.T, uploads []receivedUpload, want map[string]string) {
	t.Helper()
	if len(uploads) != 1 {
		t.Fatalf("%d requests sent, want 1", len(uploads))
	}
	upload := uploads[0]
	if upload.method != http.MethodPost {
		t.Errorf("method = %s, want %s", upload.method, http.MethodPost)
	}
	if path := "/v3/workspaces/workspace/sessions/session/messages/upload"; upload.path != path {
		t.Errorf("path = %q, want %q", upload.path, path)
	}
	if mediaType := "multipart/form-data"; upload.mediaType != mediaType {
		t.Errorf("media type = %q, want %q", upload.mediaType, mediaType)
	}
	if files := map[string][]uploadedFile{"file": {{
		contentDisposition: `form-data; name="file"; filename="notes.txt"`,
		contentType:        "text/plain; charset=utf-8",
		content:            "line one\nline two",
	}}}; !reflect.DeepEqual(upload.files, files) {
		t.Errorf("files = %+v, want %+v", upload.files, files)
	}
	if got, wanted := slices.Sorted(maps.Keys(upload.values)), slices.Sorted(maps.Keys(want)); !slices.Equal(got, wanted) {
		t.Errorf("fields = %q, want %q", got, wanted)
	}
	for name, value := range want {
		values := upload.values[name]
		if len(values) != 1 {
			t.Errorf("%s sent %d times, want once", name, len(values))
			continue
		}
		if name != "configuration" {
			if values[0] != value {
				t.Errorf("%s = %q, want %q", name, values[0], value)
			}
			continue
		}
		var got, wanted any
		if err := json.Unmarshal([]byte(values[0]), &got); err != nil {
			t.Errorf("failed to decode the configuration %q: %v", values[0], err)
			continue
		}
		if err := json.Unmarshal([]byte(value), &wanted); err != nil {
			t.Fatalf("failed to decode the expected configuration %q: %v", value, err)
		}
		if !reflect.DeepEqual(got, wanted) {
			t.Errorf("configuration = %s, want %s", values[0], value)
		}
	}
}

// CreateMessagesWithFile sends a POST multipart form holding the file and peer_id, and each optional
// field only when set: metadata as the JSON given, configuration marshalled to JSON, and created_at
// formatted with time.RFC3339Nano, keeping its nanoseconds and its zone. Empty metadata is not sent.
// Regression test: the three optional fields were strings written to the form as given, and Honcho
// silently drops a value it cannot parse, so a value written wrongly was lost without any error.
func TestCreateMessagesWithFileForm(t *testing.T) {
	createdAt := time.Date(2020, time.January, 2, 3, 4, 5, 123456789, time.FixedZone("", 2*60*60))
	tests := []struct {
		name string
		req  MessageUpload
		want map[string]string
	}{
		{
			name: "every field",
			req: MessageUpload{
				Metadata:      json.RawMessage(`{"source": "upload"}`),
				Configuration: &MessageConfiguration{Reasoning: &ReasoningConfiguration{Enabled: pointerTo(false)}},
				CreatedAt:     &createdAt,
			},
			want: map[string]string{
				"peer_id":       "alice",
				"metadata":      `{"source": "upload"}`,
				"configuration": `{"reasoning": {"enabled": false}}`,
				"created_at":    "2020-01-02T03:04:05.123456789+02:00",
			},
		},
		{name: "no optional field", want: map[string]string{"peer_id": "alice"}},
		{name: "empty metadata", req: MessageUpload{Metadata: json.RawMessage{}}, want: map[string]string{"peer_id": "alice"}},
		{
			name: "empty configuration",
			req:  MessageUpload{Configuration: &MessageConfiguration{}},
			want: map[string]string{"peer_id": "alice", "configuration": `{}`},
		},
		{
			name: "created_at in UTC",
			req:  MessageUpload{CreatedAt: pointerTo(time.Date(2020, time.January, 2, 3, 4, 5, 0, time.UTC))},
			want: map[string]string{"peer_id": "alice", "created_at": "2020-01-02T03:04:05Z"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, received := uploadServer(t)
			req := tt.req
			req.File, req.Filename, req.PeerID = []byte("line one\nline two"), "notes.txt", "alice"

			result, err := client.CreateMessagesWithFile(t.Context(), "workspace", "session", req)
			if err != nil {
				t.Fatalf("CreateMessagesWithFile: %v", err)
			}

			checkUploadForm(t, received(), tt.want)
			if len(result) != 1 || result[0].ID != "m1" {
				t.Errorf("result = %+v, want the message m1", result)
			}
		})
	}
}

// checkUploadValidation reports an error unless MessageUpload.Validate and CreateMessagesWithFile
// both refuse req with the error errMsg and nothing is sent, or, when errMsg is empty, both accept
// req and it is sent.
func checkUploadValidation(t *testing.T, req MessageUpload, errMsg string) {
	t.Helper()
	client, received := uploadServer(t)

	validateErr := req.Validate()
	result, err := client.CreateMessagesWithFile(t.Context(), "workspace", "session", req)
	requests := received()

	if errMsg == "" {
		if validateErr != nil {
			t.Errorf("Validate() error = %v, want nil", validateErr)
		}
		if err != nil {
			t.Errorf("CreateMessagesWithFile error = %v, want nil", err)
		}
		if len(requests) != 1 {
			t.Errorf("%d requests sent, want 1", len(requests))
		}
		return
	}
	if validateErr == nil || validateErr.Error() != errMsg {
		t.Errorf("Validate() error = %v, want %q", validateErr, errMsg)
	}
	if err == nil || err.Error() != errMsg {
		t.Errorf("CreateMessagesWithFile error = %v, want %q", err, errMsg)
	}
	if result != nil {
		t.Errorf("result = %+v, want nil", result)
	}
	if len(requests) != 0 {
		t.Errorf("%d requests sent, want none", len(requests))
	}
}

// MessageUpload.Validate refuses metadata that is not a JSON object, so CreateMessagesWithFile sends
// nothing: Honcho silently drops metadata that is not valid JSON, creating the messages without it,
// and stores null as no metadata. Empty metadata is accepted, and so is an object, whatever
// whitespace surrounds it. Regression test: metadata was a string written to the form as given, so
// metadata Honcho could not parse was lost while the upload succeeded.
func TestCreateMessagesWithFileRefusesInvalidMetadata(t *testing.T) {
	const errMsg = "metadata must be a JSON object: leave it empty for no metadata"
	tests := []struct {
		name     string
		metadata json.RawMessage
		errMsg   string
	}{
		{name: "invalid JSON", metadata: json.RawMessage(`optional metadata`), errMsg: errMsg},
		{name: "truncated object", metadata: json.RawMessage(`{`), errMsg: errMsg},
		{name: "object followed by text", metadata: json.RawMessage(`{"source": "upload"} x`), errMsg: errMsg},
		{name: "blank", metadata: json.RawMessage(" "), errMsg: errMsg},
		{name: "array", metadata: json.RawMessage(`[]`), errMsg: errMsg},
		{name: "string", metadata: json.RawMessage(`"text"`), errMsg: errMsg},
		{name: "number", metadata: json.RawMessage(`5`), errMsg: errMsg},
		{name: "null", metadata: json.RawMessage(`null`), errMsg: errMsg},
		{name: "nil"},
		{name: "empty", metadata: json.RawMessage{}},
		{name: "empty object", metadata: json.RawMessage(`{}`)},
		{name: "object surrounded by whitespace", metadata: json.RawMessage("\n {\"source\": \"upload\"}\t\r\n")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := MessageUpload{File: []byte("line one\nline two"), Filename: "notes.txt", PeerID: "alice", Metadata: tt.metadata}
			checkUploadValidation(t, req, tt.errMsg)
		})
	}
}

// MessageUpload.Validate refuses a created_at whose year, in its own location, is outside 1 to 9999,
// so CreateMessagesWithFile sends nothing: Honcho cannot parse such a timestamp, and silently replaces
// it with the time of the upload. Milliseconds read as seconds give such a year.
func TestCreateMessagesWithFileRefusesCreatedAtOutOfRange(t *testing.T) {
	const errMsg = "created_at year must be between 1 and 9999"
	tests := []struct {
		name      string
		createdAt time.Time
		errMsg    string
	}{
		{name: "year 1", createdAt: time.Date(1, time.January, 1, 0, 0, 0, 0, time.UTC)},
		{name: "year 9999", createdAt: time.Date(9999, time.December, 31, 23, 59, 59, 999999999, time.UTC)},
		{name: "year 0", createdAt: time.Date(0, time.December, 31, 23, 59, 59, 0, time.UTC), errMsg: errMsg},
		{name: "year 10000", createdAt: time.Date(10000, time.January, 1, 0, 0, 0, 0, time.UTC), errMsg: errMsg},
		{name: "milliseconds read as seconds", createdAt: time.Unix(1577934245000, 0).UTC(), errMsg: errMsg},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := MessageUpload{File: []byte("line one\nline two"), Filename: "notes.txt", PeerID: "alice", CreatedAt: &tt.createdAt}
			checkUploadValidation(t, req, tt.errMsg)
		})
	}
}

// uploadedFileOf returns the file of the single upload in uploads, failing the test unless there is
// exactly one upload, holding exactly one file, sent as the file field.
func uploadedFileOf(t *testing.T, uploads []receivedUpload) uploadedFile {
	t.Helper()
	if len(uploads) != 1 {
		t.Fatalf("%d requests sent, want 1", len(uploads))
	}
	files := uploads[0].files
	if len(files) != 1 || len(files["file"]) != 1 {
		t.Fatalf("files = %+v, want one file sent as the file field", files)
	}
	return files["file"][0]
}

// CreateMessagesWithFile declares the media type of the file in the Content-Type of its part, as
// Honcho reads the file according to it: ContentType as given when set, or else the type
// http.DetectContentType finds, text/plain for text, JSON included, and application/pdf for a PDF.
// Regression test: the part was written with CreateFormFile, which declares
// application/octet-stream, so Honcho refused every upload with a 415.
func TestCreateMessagesWithFileContentType(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		content     string
		want        string
	}{
		{name: "given text type", contentType: "text/markdown; charset=utf-8", content: "# Notes\n", want: "text/markdown; charset=utf-8"},
		{name: "given JSON type", contentType: "application/json", content: `{"note": "text"}`, want: "application/json"},
		{name: "text detected", content: "line one\nline two", want: "text/plain; charset=utf-8"},
		{name: "JSON detected as text", content: `{"note": "text"}`, want: "text/plain; charset=utf-8"},
		{name: "PDF detected", content: "%PDF-1.7\n%\xe2\xe3\xcf\xd3\n", want: "application/pdf"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, received := uploadServer(t)
			req := MessageUpload{File: []byte(tt.content), Filename: "notes", ContentType: tt.contentType, PeerID: "alice"}

			if _, err := client.CreateMessagesWithFile(t.Context(), "workspace", "session", req); err != nil {
				t.Fatalf("CreateMessagesWithFile: %v", err)
			}

			file := uploadedFileOf(t, received())
			if file.contentType != tt.want {
				t.Errorf("Content-Type = %q, want %q", file.contentType, tt.want)
			}
			if file.content != tt.content {
				t.Errorf("content = %q, want %q", file.content, tt.content)
			}
		})
	}
}

// CreateMessagesWithFile names the file part file in its Content-Disposition, with the file name
// escaped as CreateFormFile escapes it: each backslash and quote is preceded by a backslash, so the
// name is read back as given.
func TestCreateMessagesWithFileFilename(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     string
	}{
		{name: "plain name", filename: "notes.txt", want: `form-data; name="file"; filename="notes.txt"`},
		{name: "name with quotes", filename: `my "notes".txt`, want: `form-data; name="file"; filename="my \"notes\".txt"`},
		{name: "name with a backslash", filename: `notes\v2.txt`, want: `form-data; name="file"; filename="notes\\v2.txt"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, received := uploadServer(t)
			req := MessageUpload{File: []byte("line one\nline two"), Filename: tt.filename, PeerID: "alice"}

			if _, err := client.CreateMessagesWithFile(t.Context(), "workspace", "session", req); err != nil {
				t.Fatalf("CreateMessagesWithFile: %v", err)
			}

			file := uploadedFileOf(t, received())
			if file.contentDisposition != tt.want {
				t.Errorf("Content-Disposition = %s, want %s", file.contentDisposition, tt.want)
			}
			// Parsed here rather than read from the server's FileHeader, which keeps only the base
			// name, the separators of which depend on the platform.
			_, params, err := mime.ParseMediaType(file.contentDisposition)
			if err != nil {
				t.Fatalf("failed to parse the Content-Disposition %q: %v", file.contentDisposition, err)
			}
			if params["filename"] != tt.filename {
				t.Errorf("filename read back = %q, want %q", params["filename"], tt.filename)
			}
		})
	}
}

// MessageUpload.Validate refuses a ContentType that is not a media type, so CreateMessagesWithFile
// sends nothing: a typo would only get a 415, and a line break would break the header of the file
// part, letting the value add headers of its own. The error wraps the one of mime.ParseMediaType,
// which skips a line break around the type or between parameters, so such a line break is refused
// as well. Any media type is accepted, not only those Honcho reads, and so is an empty one.
func TestCreateMessagesWithFileRefusesInvalidContentType(t *testing.T) {
	const lineBreakErr = "content_type must not hold a line break"
	tests := []struct {
		name        string
		contentType string
		// unparsable is whether mime.ParseMediaType fails on the content type, and errMsg the error
		// expected otherwise, empty when the content type is accepted
		unparsable bool
		errMsg     string
	}{
		{name: "header injected", contentType: "text/plain\r\nX-Injected: 1", unparsable: true},
		{name: "space for a slash", contentType: "text plain", unparsable: true},
		{name: "semicolon alone", contentType: ";", unparsable: true},
		{name: "line break after the type", contentType: "text/plain\r\n", errMsg: lineBreakErr},
		{name: "line break between parameters", contentType: "text/plain; charset=utf-8\r\n; format=flowed", errMsg: lineBreakErr},
		{name: "text type with a parameter", contentType: "text/markdown; charset=utf-8"},
		{name: "JSON type", contentType: "application/json"},
		{name: "type Honcho does not read", contentType: "image/png"},
		{name: "empty"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := MessageUpload{File: []byte("line one\nline two"), Filename: "notes.txt", ContentType: tt.contentType, PeerID: "alice"}
			errMsg := tt.errMsg
			if tt.unparsable {
				_, _, parseErr := mime.ParseMediaType(tt.contentType)
				if parseErr == nil {
					t.Fatalf("mime.ParseMediaType(%q) error = nil, want one", tt.contentType)
				}
				errMsg = "content_type must be a media type such as text/plain: " + parseErr.Error()
				if wrapped := errors.Unwrap(req.Validate()); wrapped == nil || wrapped.Error() != parseErr.Error() {
					t.Errorf("Validate() wraps %v, want %v", wrapped, parseErr)
				}
			}

			checkUploadValidation(t, req, errMsg)
		})
	}
}
