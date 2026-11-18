package honcho

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
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
