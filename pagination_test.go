package honcho

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

// One valid item of each list, in the shape of upstream's schemas.
const (
	workspaceItemJSON  = `{"id":"w","uid":"46xqSOtyM08WIPPJQM5_6","metadata":{},"configuration":{},"created_at":"2026-11-18T16:09:28.620049Z"}`
	peerItemJSON       = `{"id":"alice","uid":"yTkg0JNCYKPlmpuCQAfqn","workspace_id":"workspace","created_at":"2026-11-18T16:09:28.620049Z","metadata":{},"configuration":{}}`
	sessionItemJSON    = `{"id":"s1","uid":"X7i6McQJTWP_F1wFzG3E2","is_active":true,"workspace_id":"workspace","metadata":{},"configuration":{},"created_at":"2026-11-18T16:09:28.620049Z"}`
	messageItemJSON    = `{"id":"m1","content":"Hello.","peer_id":"alice","session_id":"session","metadata":{},"created_at":"2026-11-18T16:09:28.620049Z","workspace_id":"workspace","token_count":2}`
	conclusionItemJSON = `{"id":"vnzseRU4JQrIGkJg4FQoj","content":"alice has a sister named Joke.","observer_id":"alice","observed_id":"alice","session_id":"s","level":"explicit","source_ids":[],"source_message_ids":["cgGcyBR7s6VdYYv40PKvN"],"times_derived":1,"created_at":"2026-11-18T14:33:17.975162Z"}`
	webhookItemJSON    = `{"id":"e1","workspace_id":"workspace","url":"https://example.test/webhook","created_at":"2026-11-18T16:09:28.620049Z"}`
	dreamRunItemJSON   = `{` + dreamRunJSON + `}`
	scopeItemJSON      = `{"id":"team","metadata":{},"created_at":"2026-11-18T16:09:34.673599Z"}`
)

// listRoutes are the list methods, each of which can paginate by page number or by cursor.
var listRoutes = []struct {
	name, method, path string
	// item is one valid item of the list, as the server returns it
	item string
	// list calls the method with the given paging options and returns the page it decoded
	list func(ctx context.Context, client *Client, page, size int, cursor *string) (any, error)
}{
	{
		name: "GetAllWorkspaces", method: http.MethodPost, path: "/v3/workspaces/list", item: workspaceItemJSON,
		list: func(ctx context.Context, client *Client, page, size int, cursor *string) (any, error) {
			return client.GetAllWorkspaces(ctx, WorkspaceGetRequest{}, &GetAllWorkspacesOptions{Page: page, Size: size, Cursor: cursor})
		},
	},
	{
		name: "GetAllPeers", method: http.MethodPost, path: "/v3/workspaces/workspace/peers/list", item: peerItemJSON,
		list: func(ctx context.Context, client *Client, page, size int, cursor *string) (any, error) {
			return client.GetAllPeers(ctx, "workspace", nil, &GetAllPeersOptions{Page: page, Size: size, Cursor: cursor})
		},
	},
	{
		name: "GetSessionsForPeer", method: http.MethodPost, path: "/v3/workspaces/workspace/peers/alice/sessions", item: sessionItemJSON,
		list: func(ctx context.Context, client *Client, page, size int, cursor *string) (any, error) {
			return client.GetSessionsForPeer(ctx, "workspace", "alice", nil, &GetSessionsForPeerOptions{Page: page, Size: size, Cursor: cursor})
		},
	},
	{
		name: "GetSessions", method: http.MethodPost, path: "/v3/workspaces/workspace/sessions/list", item: sessionItemJSON,
		list: func(ctx context.Context, client *Client, page, size int, cursor *string) (any, error) {
			return client.GetSessions(ctx, "workspace", nil, &GetSessionsOptions{Page: page, Size: size, Cursor: cursor})
		},
	},
	{
		name: "GetSessionPeers", method: http.MethodGet, path: "/v3/workspaces/workspace/sessions/session/peers", item: peerItemJSON,
		list: func(ctx context.Context, client *Client, page, size int, cursor *string) (any, error) {
			return client.GetSessionPeers(ctx, "workspace", "session", &GetSessionPeersOptions{Page: page, Size: size, Cursor: cursor})
		},
	},
	{
		name: "GetMessages", method: http.MethodPost, path: "/v3/workspaces/workspace/sessions/session/messages/list", item: messageItemJSON,
		list: func(ctx context.Context, client *Client, page, size int, cursor *string) (any, error) {
			return client.GetMessages(ctx, "workspace", "session", nil, &GetMessagesOptions{Page: page, Size: size, Cursor: cursor})
		},
	},
	{
		name: "ListConclusions", method: http.MethodPost, path: "/v3/workspaces/workspace/conclusions/list", item: conclusionItemJSON,
		list: func(ctx context.Context, client *Client, page, size int, cursor *string) (any, error) {
			return client.ListConclusions(ctx, "workspace", nil, &ListConclusionsOptions{Page: page, Size: size, Cursor: cursor})
		},
	},
	{
		name: "ListWebhookEndpoints", method: http.MethodGet, path: "/v3/workspaces/workspace/webhooks", item: webhookItemJSON,
		list: func(ctx context.Context, client *Client, page, size int, cursor *string) (any, error) {
			return client.ListWebhookEndpoints(ctx, "workspace", &ListWebhookEndpointsOptions{Page: page, Size: size, Cursor: cursor})
		},
	},
	{
		name: "ListDreams", method: http.MethodPost, path: "/v3/workspaces/workspace/dreams/list", item: dreamRunItemJSON,
		list: func(ctx context.Context, client *Client, page, size int, cursor *string) (any, error) {
			return client.ListDreams(ctx, "workspace", nil, &ListDreamsOptions{Page: page, Size: size, Cursor: cursor})
		},
	},
	{
		name: "GetScopes", method: http.MethodPost, path: "/v3/workspaces/workspace/scopes/list", item: scopeItemJSON,
		list: func(ctx context.Context, client *Client, page, size int, cursor *string) (any, error) {
			return client.GetScopes(ctx, "workspace", nil, &GetScopesOptions{Page: page, Size: size, Cursor: cursor})
		},
	},
	{
		name: "GetScopeSessions", method: http.MethodPost, path: "/v3/workspaces/workspace/scopes/team/sessions/list", item: sessionItemJSON,
		list: func(ctx context.Context, client *Client, page, size int, cursor *string) (any, error) {
			return client.GetScopeSessions(ctx, "workspace", "team", &GetScopeSessionsOptions{Page: page, Size: size, Cursor: cursor})
		},
	},
}

// pageFields is what the tests read of a page, whatever the type of its items.
type pageFields struct {
	items                                                     int
	total, page, size, pages                                  int
	currentPage, currentPageBackwards, previousPage, nextPage *string
}

// fieldsOf returns the fields every page type shares, read from page.
func fieldsOf(t *testing.T, page any) pageFields {
	t.Helper()
	switch p := page.(type) {
	case *PageWorkspace:
		return pageFields{len(p.Items), p.Total, p.Page, p.Size, p.Pages, p.CurrentPage, p.CurrentPageBackwards, p.PreviousPage, p.NextPage}
	case *PagePeer:
		return pageFields{len(p.Items), p.Total, p.Page, p.Size, p.Pages, p.CurrentPage, p.CurrentPageBackwards, p.PreviousPage, p.NextPage}
	case *PageSession:
		return pageFields{len(p.Items), p.Total, p.Page, p.Size, p.Pages, p.CurrentPage, p.CurrentPageBackwards, p.PreviousPage, p.NextPage}
	case *PageMessage:
		return pageFields{len(p.Items), p.Total, p.Page, p.Size, p.Pages, p.CurrentPage, p.CurrentPageBackwards, p.PreviousPage, p.NextPage}
	case *PageConclusion:
		return pageFields{len(p.Items), p.Total, p.Page, p.Size, p.Pages, p.CurrentPage, p.CurrentPageBackwards, p.PreviousPage, p.NextPage}
	case *PageWebhookEndpoint:
		return pageFields{len(p.Items), p.Total, p.Page, p.Size, p.Pages, p.CurrentPage, p.CurrentPageBackwards, p.PreviousPage, p.NextPage}
	case *PageDreamRun:
		return pageFields{len(p.Items), p.Total, p.Page, p.Size, p.Pages, p.CurrentPage, p.CurrentPageBackwards, p.PreviousPage, p.NextPage}
	case *PageScope:
		return pageFields{len(p.Items), p.Total, p.Page, p.Size, p.Pages, p.CurrentPage, p.CurrentPageBackwards, p.PreviousPage, p.NextPage}
	}
	t.Fatalf("%T is not a page type", page)
	return pageFields{}
}

// String formats the fields, the cursors by value rather than by address.
func (f pageFields) String() string {
	cursor := func(value *string) string {
		if value == nil {
			return "nil"
		}
		return strconv.Quote(*value)
	}
	return fmt.Sprintf("{items: %d, total: %d, page: %d, size: %d, pages: %d, current_page: %s, current_page_backwards: %s, previous_page: %s, next_page: %s}",
		f.items, f.total, f.page, f.size, f.pages,
		cursor(f.currentPage), cursor(f.currentPageBackwards), cursor(f.previousPage), cursor(f.nextPage))
}

// Every list method sends a cursor that is set in the cursor query parameter, an empty cursor
// included, and alongside the size: the server reads exactly the cursor it was given, whether it
// holds characters to escape or escapes of its own, as the cursors the server returns can. Left
// nil, the cursor is not sent at all, so a request by page number is the same as before cursors.
func TestListsSendCursor(t *testing.T) {
	tests := []struct {
		name      string
		page      int
		cursor    *string
		wantQuery url.Values
	}{
		{name: "no cursor", page: 2, wantQuery: url.Values{"page": {"2"}, "size": {"10"}}},
		{name: "empty cursor", cursor: new(string), wantQuery: url.Values{"cursor": {""}, "size": {"10"}}},
		{name: "cursor holding escapes", cursor: pointerTo("Pg%3D%3D"), wantQuery: url.Values{"cursor": {"Pg%3D%3D"}, "size": {"10"}}},
		{name: "cursor to escape", cursor: pointerTo("ab+/="), wantQuery: url.Values{"cursor": {"ab+/="}, "size": {"10"}}},
	}
	for _, route := range listRoutes {
		for _, tt := range tests {
			t.Run(route.name+"/"+tt.name, func(t *testing.T) {
				client, received := scopeServer(t, http.StatusOK, emptyPageJSON)

				if _, err := route.list(t.Context(), client, tt.page, 10, tt.cursor); err != nil {
					t.Fatalf("%s: %v", route.name, err)
				}

				requests := received()
				if len(requests) != 1 {
					t.Fatalf("%d requests sent, want 1", len(requests))
				}
				request := requests[0]
				if request.method != route.method || request.path != route.path {
					t.Errorf("request = %s %s, want %s %s", request.method, request.path, route.method, route.path)
				}
				if !reflect.DeepEqual(request.query, tt.wantQuery) {
					t.Errorf("query = %v, want %v", request.query, tt.wantQuery)
				}
			})
		}
	}
}

// Every list method refuses a page given together with a cursor, an empty cursor included, with
// the same error and without sending a request: the server would silently ignore the page.
func TestListsRefusePageWithCursor(t *testing.T) {
	cursors := []struct {
		name   string
		cursor *string
	}{
		{name: "empty cursor", cursor: new(string)},
		{name: "cursor", cursor: pointerTo("Pg%3D%3D")},
	}
	for _, route := range listRoutes {
		for _, tt := range cursors {
			t.Run(route.name+"/"+tt.name, func(t *testing.T) {
				client, received := scopeServer(t, http.StatusOK, emptyPageJSON)

				_, err := route.list(t.Context(), client, 2, 10, tt.cursor)

				if !errors.Is(err, errPageWithCursor) || err.Error() != errPageWithCursor.Error() {
					t.Errorf("error = %v, want %v", err, errPageWithCursor)
				}
				if requests := received(); len(requests) != 0 {
					t.Errorf("%d requests sent, want none", len(requests))
				}
			})
		}
	}
}

// ListWebhookEndpointsOptions.Validate refuses a page given together with a cursor, with the
// error every list method gives, and accepts a page or a cursor alone.
func TestListWebhookEndpointsOptionsValidateCursor(t *testing.T) {
	tests := []struct {
		name    string
		opts    ListWebhookEndpointsOptions
		wantErr error
	}{
		{name: "page only", opts: ListWebhookEndpointsOptions{Page: 2}},
		{name: "empty cursor only", opts: ListWebhookEndpointsOptions{Cursor: new(string)}},
		{name: "cursor only", opts: ListWebhookEndpointsOptions{Cursor: pointerTo("Pg%3D%3D")}},
		{name: "page and empty cursor", opts: ListWebhookEndpointsOptions{Page: 2, Cursor: new(string)}, wantErr: errPageWithCursor},
		{name: "page and cursor", opts: ListWebhookEndpointsOptions{Page: 2, Cursor: pointerTo("Pg%3D%3D")}, wantErr: errPageWithCursor},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.opts.Validate(); !errors.Is(err, tt.wantErr) {
				t.Errorf("Validate() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

// Every list method decodes a cursor page, whose total is null and which has no page numbers,
// into its items and cursors, the total and page numbers left at zero and the previous page nil
// as on a first page. A page by number still decodes as before, with every cursor nil.
func TestListsDecodePages(t *testing.T) {
	tests := []struct {
		name   string
		cursor *string
		// body is the page the server answers with, holding the given item
		body func(item string) string
		want pageFields
	}{
		{
			name:   "cursor page",
			cursor: new(string),
			body: func(item string) string {
				return `{"items":[` + item + `],"total":null,"current_page":"a","current_page_backwards":"b","previous_page":null,"next_page":"c"}`
			},
			want: pageFields{items: 1, currentPage: pointerTo("a"), currentPageBackwards: pointerTo("b"), nextPage: pointerTo("c")},
		},
		{
			name: "page by number",
			body: func(item string) string {
				return `{"items":[` + item + `],"total":1,"page":1,"size":50,"pages":1}`
			},
			want: pageFields{items: 1, total: 1, page: 1, size: 50, pages: 1},
		},
	}
	for _, route := range listRoutes {
		for _, tt := range tests {
			t.Run(route.name+"/"+tt.name, func(t *testing.T) {
				client, _ := scopeServer(t, http.StatusOK, tt.body(route.item))

				page, err := route.list(t.Context(), client, 0, 0, tt.cursor)
				if err != nil {
					t.Fatalf("%s: %v", route.name, err)
				}

				if got := fieldsOf(t, page); !reflect.DeepEqual(got, tt.want) {
					t.Errorf("page = %v, want %v", got, tt.want)
				}
			})
		}
	}
}

// A cursor the server cannot read comes back as an error holding the detail of the 400 the
// server answers it with.
func TestListInvalidCursor(t *testing.T) {
	const detail = "Invalid cursor"
	client, received := scopeServer(t, http.StatusBadRequest, `{"detail":"`+detail+`"}`)

	_, err := client.GetSessions(t.Context(), "workspace", nil, &GetSessionsOptions{Cursor: pointerTo("garbage")})

	if err == nil || !strings.Contains(err.Error(), detail) {
		t.Errorf("error = %v, want an error holding %q", err, detail)
	}
	if requests := received(); len(requests) != 1 {
		t.Errorf("%d requests sent, want 1", len(requests))
	}
}
