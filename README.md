# go-honcho

Unofficial Go SDK for the [Honcho API v3](https://docs.honcho.dev/v3).

Honcho is a memory layer for AI applications that enables persistent, contextual memory across conversations. This SDK provides a type-safe, idiomatic Go client for interacting with the Honcho API.

> **Note:** This is a community-maintained SDK and is not officially affiliated with Honcho.

> **Agentic Engineering:** This SDK was developed using AI-assisted development following the guidelines in [AGENT.md](AGENT.md). The implementation adheres to consistent patterns and conventions established for AI agent collaboration.

## Features

- **Type-safe API client** - Strongly-typed structs for all requests and responses
- **Comprehensive error handling** - Wrapped errors with context and error chain support
- **Client-side validation** - Validate required fields and constraints before API calls
- **Thread-safe** - Client instances are safe for concurrent use
- **Minimal dependencies** - Only requires `go-cleanhttp` for HTTP client pooling

## Installation

```bash
go get github.com/hekmon/go-honcho
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/hekmon/go-honcho"
)

func main() {
    // Initialize client with API key
    // For self-hosted instances, API key is optional (can be empty string)
    client := honcho.New(&honcho.Options{
        APIKey: "your-api-key",
    })

    // Get or create a workspace
    workspace, err := client.GetOrCreateWorkspace(context.Background(), honcho.CreateWorkspaceRequest{
        ID: "my-workspace",
    })
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Workspace ID: %s\n", workspace.ID)
}
```

## Client Initialization

### Basic Setup

```go
// Initialize with default settings (uses https://api.honcho.dev)
client := honcho.New(&honcho.Options{
    APIKey: "your-api-key",
})
```

### Self-Hosted / No API Key

```go
// For self-hosted instances, API key is optional
import "net/url"

baseURL, _ := url.Parse("https://your-self-hosted-url.com")
client := honcho.New(&honcho.Options{
    BaseURL: baseURL,
    // APIKey omitted - no authorization header will be sent
})
```

### Custom Base URL

```go
// Use a custom base URL (e.g., for self-hosted or different environments)
import "net/url"

baseURL, _ := url.Parse("https://api.honcho.dev")
client := honcho.New(&honcho.Options{
    BaseURL: baseURL,
    APIKey:  "your-api-key",
})
```

### Custom HTTP Client

```go
// Customize HTTP client settings (timeouts, transport, etc.)
import (
    "net/http"
    "time"

    "github.com/hashicorp/go-cleanhttp"
)

// Start with cleanhttp.DefaultPooledClient() and modify as needed
httpClient := cleanhttp.DefaultPooledClient()
httpClient.Timeout = 30 * time.Second
// Stay below the server's 5 second keep-alive: a POST sent on a connection the server is closing fails with EOF
httpClient.Transport.(*http.Transport).IdleConnTimeout = 4 * time.Second

client := honcho.New(&honcho.Options{
    APIKey: "your-api-key",
    HTTP:   httpClient,
})
```

## API Categories

The SDK is organized by API categories. Each category has its own methods and types:

| Category | Description | Files |
|----------|-------------|-------|
| **Workspace** | Manage workspaces (top-level containers) | `workspace.go`, `workspace_types.go` |
| **Peer** | Manage peers (users/entities in workspaces) | `peer.go`, `peer_types.go` |
| **Session** | Manage sessions (conversation threads) | `session.go`, `session_types.go` |
| **Message** | Send and retrieve messages | `message.go`, `message_types.go` |
| **Conclusion** | Manage conclusions (derived insights) | `conclusion.go`, `conclusion_types.go` |
| **Dream** | Review what each dream changed in a representation | `dream.go`, `dream_types.go` |
| **Scope** | Group sessions into visibility boundaries on recall | `scope.go`, `scope_types.go` |
| **Webhook** | Configure webhook endpoints | `webhook.go`, `webhook_types.go` |
| **Key** | Manage API keys | `key.go`, `key_types.go` |

### Workspace Operations

```go
// Get or create a workspace
workspace, err := client.GetOrCreateWorkspace(honcho.CreateWorkspaceRequest{
    ID: "my-workspace",
})

// List all workspaces (paginated)
workspaces, err := client.GetAllWorkspaces(honcho.WorkspaceGetRequest{}, &honcho.GetAllWorkspacesOptions{
    Page: 1,
    Size: 50,
})

// Update workspace metadata
workspace, err = client.UpdateWorkspace("my-workspace", honcho.UpdateWorkspaceRequest{
    Metadata: map[string]any{
        "environment": "production",
    },
})

// Delete a workspace (must delete all sessions first)
err = client.DeleteWorkspace("my-workspace")

// Search messages in a workspace (returns pointer to slice: *[]Message)
result, err := client.SearchWorkspace(ctx, "my-workspace", honcho.WorkspaceMessageSearchOptions{
    Query: "important topic",
})
// result is *[]Message - dereference to access: messages := *result

// Get queue status (optional: observerID, senderID, sessionID)
status, err := client.GetQueueStatus("my-workspace", nil, nil, nil)

// Get queue status scoped to specific observer and session
observerID := "user-123"
sessionID := "session-456"
status, err = client.GetQueueStatus("my-workspace", &observerID, nil, &sessionID)

// Schedule a dream task
err = client.ScheduleDream("my-workspace", honcho.ScheduleDreamRequest{
    Observer:  "user-123",
    DreamType: honcho.DreamTypeOmni,
})

// Query the whole workspace in natural language, here within two scopes
response, err := client.WorkspaceChat(ctx, "my-workspace", honcho.WorkspaceChatOptions{
    Query:           "What are the peers working on?",
    Scope:           []string{"team-a", "team-b"},
    IncludeEvidence: true, // response.Evidence lists what the agent read
})

// Stream the answer (evidence is only available through WorkspaceChat)
for chunk, err := range client.WorkspaceChatStream(ctx, "my-workspace", honcho.WorkspaceChatOptions{
    Query: "What are the peers working on?",
}) {
    if err != nil {
        log.Fatal(err)
    }
    fmt.Print(chunk)
}
```

### Peer Operations

```go
// Create or get a peer
peer, err := client.GetOrCreatePeer("workspace-id", honcho.PeerCreate{
    ID: "user-123",
    Metadata: map[string]any{
        "name": "Alice",
    },
})

// Get peer representation
rep, err := client.GetRepresentation("workspace-id", "user-123", honcho.PeerRepresentationGet{
    // Optional fields for scoping
})

// Get peer card
card, err := client.GetPeerCard("workspace-id", "user-123", nil)

// Set peer card
card, err = client.SetPeerCard("workspace-id", "user-123", honcho.PeerCardSet{
    Card: "peer card content",
}, nil)

// Get peer context (for RAG/retrieval)
searchTopK := 5
context, err := client.GetPeerContext("workspace-id", "user-123", &honcho.GetPeerContextOptions{
    SearchTopK: &searchTopK,
})

// Get all peers (paginated)
peers, err := client.GetAllPeers("workspace-id", nil, &honcho.GetAllPeersOptions{
    Page: 1,
    Size: 20,
})

// Update peer metadata
peer, err = client.UpdatePeer("workspace-id", "user-123", honcho.PeerUpdate{
    Metadata: map[string]any{
        "name": "Alice Smith",
    },
})

// Get sessions for a peer
sessions, err := client.GetSessionsForPeer("workspace-id", "user-123", nil, &honcho.GetSessionsForPeerOptions{
    Page: 1,
    Size: 20,
})

// Search peer's messages
messages, err := client.SearchPeer("workspace-id", "user-123", honcho.MessageSearchOptions{
    Query: "search query",
})

// Chat with peer's representation
response, err := client.Chat("workspace-id", "user-123", honcho.DialecticOptions{
    Query: "What does this peer prefer?",
})

// Answer from one scope's own representation of the peer
response, err = client.Chat(ctx, "workspace-id", "user-123", honcho.DialecticOptions{
    Query: "What does this peer prefer?",
    Scope: honcho.OneScope("team-a"),
})

// Restrict recall to the member sessions of these scopes. A union is an allowlist of
// sessions, even with a single name: ScopeUnion("team-a") is not OneScope("team-a")
response, err = client.Chat(ctx, "workspace-id", "user-123", honcho.DialecticOptions{
    Query: "What does this peer prefer?",
    Scope: honcho.ScopeUnion("team-a", "team-b"),
})

// Ask for an answer matching a JSON Schema: *response.Content is then a JSON string
response, err = client.Chat(ctx, "workspace-id", "user-123", honcho.DialecticOptions{
    Query:           "What are this peer's hobbies?",
    ResponseFormat:  json.RawMessage(`{"type":"object","properties":{"hobbies":{"type":"array"}}}`),
    IncludeEvidence: true, // response.Evidence lists what the agent read
})

// Stream the answer as it is generated (evidence is only available through Chat)
for chunk, err := range client.ChatStream(ctx, "workspace-id", "user-123", honcho.DialecticOptions{
    Query: "What does this peer prefer?",
}) {
    if err != nil {
        log.Fatal(err)
    }
    fmt.Print(chunk)
}
```

### Session Operations

```go
// Create or get a session
session, err := client.GetOrCreateSession("workspace-id", honcho.SessionCreate{
    ID: "my-session",
    Metadata: map[string]any{
        "topic": "Customer Support",
    },
})

// Get sessions (paginated) - pass nil for req when no filters needed
sessions, err := client.GetSessions("workspace-id", nil, &honcho.GetSessionsOptions{
    Page: 1,
    Size: 20,
})

// Get sessions with filters
sessions, err = client.GetSessions("workspace-id", &honcho.WorkspaceSessionGet{
    Filters: map[string]any{"status": "active"},
}, &honcho.GetSessionsOptions{
    Page: 1,
    Size: 20,
})

// Update session metadata
session, err = client.UpdateSession("workspace-id", "session-id", honcho.SessionUpdate{
    Metadata: map[string]any{
        "topic": "Updated Topic",
    },
})

// Delete a session
err = client.DeleteSession("workspace-id", "session-id")

// Clone a session
cloned, err := client.CloneSession("workspace-id", "session-id", nil)

// Clone up to a specific message
messageID := "msg-123"
cloned, err = client.CloneSession("workspace-id", "session-id", &honcho.CloneSessionOptions{
    MessageID: &messageID,
})

// Get session peers
pagePeer, err := client.GetSessionPeers("workspace-id", "session-id", &honcho.GetSessionPeersOptions{
    Page: 1,
    Size: 10,
})

// Set session peers
observeMe := true
peers := map[string]*honcho.SessionPeerConfig{
    "user-123": {
        ObserveMe: &observeMe,
    },
}
session, err = client.SetSessionPeers("workspace-id", "session-id", peers)

// Add peers to session
session, err = client.AddPeersToSession("workspace-id", "session-id", peers)

// Remove peers from session
err = client.RemovePeersFromSession("workspace-id", "session-id", []string{"user-123"})

// Get session context
tokens := 1000
context, err := client.GetSessionContext("workspace-id", "session-id", &honcho.GetSessionContextOptions{
    Tokens: &tokens,
})

// Get session summaries
summaries, err := client.GetSessionSummaries("workspace-id", "session-id")

// Search session messages
messages, err := client.SearchSession("workspace-id", "session-id", honcho.MessageSearchOptions{
    Query: "search query",
})

// Get peer config in session
config, err := client.GetPeerConfig("workspace-id", "session-id", "user-123")

// Set peer config in session
observeMe := true
err = client.SetPeerConfig("workspace-id", "session-id", "user-123", honcho.SessionPeerConfig{
    ObserveMe: &observeMe,
})

// Get session summaries (short and long)
summaries, err := client.GetSessionSummaries("workspace-id", "session-id")
```

### Message Operations

```go
// Create messages for a session
messages, err := client.CreateMessagesForSession("workspace-id", "session-id", honcho.MessageBatchCreate{
    Messages: []honcho.MessageCreate{
        {
            PeerID:  "user-123",
            Content: "Hello, how can I help?",
            IsUser:  true,
        },
        {
            PeerID:  "assistant-1",
            Content: "I need help with my account.",
            IsUser:  false,
        },
    },
})

// Create messages with file upload
fileContent := []byte("file content here")
metadata := "optional metadata"
messages, err = client.CreateMessagesWithFile("workspace-id", "session-id", honcho.MessageUpload{
    File:     fileContent,
    Filename: "document.txt",
    PeerID:   "user-123",
    Metadata: &metadata,
})

// Get a single message
message, err := client.GetMessage("workspace-id", "session-id", "message-id")

// Get messages (paginated) - pass nil for req when no filters needed
page, err := client.GetMessages("workspace-id", "session-id", nil, &honcho.GetMessagesOptions{
    Page: 1,
    Size: 50,
})

// Get messages with filters
page, err = client.GetMessages("workspace-id", "session-id", &honcho.MessageGet{
    Filters: map[string]any{"peer_id": "user-123"},
}, &honcho.GetMessagesOptions{
    Page: 1,
    Size: 50,
})

// Update message metadata
message, err = client.UpdateMessage("workspace-id", "session-id", "message-id", honcho.MessageUpdate{
    Metadata: map[string]any{
        "flagged": true,
    },
})
```

### Conclusion Operations

```go
// Create conclusions
conclusions, err := client.CreateConclusions("workspace-id", honcho.ConclusionBatchCreate{
    Conclusions: []honcho.ConclusionCreate{
        {
            ObserverID: "user-123",
            ObservedID: "user-456",
            Content:    "User prefers email communication",
        },
    },
})

// List conclusions (paginated) - pass nil for req when no filters needed
page, err := client.ListConclusions("workspace-id", nil, &honcho.ListConclusionsOptions{
    Page:    1,
    Size:    20,
    Reverse: false, // Order direction (default: false)
})

// List conclusions with filters
page, err = client.ListConclusions("workspace-id", &honcho.ConclusionGet{
    Filters: map[string]any{"observer_id": "user-123"},
}, &honcho.ListConclusionsOptions{
    Page: 1,
    Size: 20,
})

// Query conclusions (semantic search)
conclusions, err = client.QueryConclusions("workspace-id", honcho.ConclusionQuery{
    Query: "communication preferences",
    TopK:  10,
    // Required: name the observer (observer or observer_id) and the observed peer (observed or observed_id)
    Filters: map[string]any{"observer_id": "user-123", "observed_id": "user-456"},
})

// Get a conclusion by ID
conclusion, err := client.GetConclusion(ctx, "workspace-id", "conclusion-id")

// Delete a conclusion
err = client.DeleteConclusion("workspace-id", "conclusion-id")
```

### Dream Operations

```go
// List dream runs, newest first (paginated) - pass nil for req when no filters needed
page, err := client.ListDreams(ctx, "workspace-id", nil, &honcho.ListDreamsOptions{
    Page: 1,
    Size: 20,
})

// List the runs of one representation
observedID, observerID := "user-123", "user-456"
page, err = client.ListDreams(ctx, "workspace-id", &honcho.DreamRunList{
    ObservedID: &observedID,
    ObserverID: &observerID,
}, nil)

// Get one run with every change it committed
dream, err := client.GetDream(ctx, "workspace-id", "run-id")
```

### Scope Operations

```go
// Get or create a scope (ID: 1 to 506 characters)
// On an existing scope, metadata that differs overwrites what is stored: leave it nil to keep it
scope, err := client.GetOrCreateScope(ctx, "workspace-id", honcho.ScopeCreate{
    ID:       "team",
    Metadata: json.RawMessage(`{"owner":"alice"}`),
})

// List scopes with their member session counts (paginated) - pass nil for req when no filters needed
page, err := client.GetScopes(ctx, "workspace-id", nil, &honcho.GetScopesOptions{
    Page:                1,
    Size:                20,
    IncludeSessionCount: true,
})

// List the scopes whose name contains a text
page, err = client.GetScopes(ctx, "workspace-id", &honcho.ScopeGet{
    Filters: map[string]any{"id": map[string]any{"icontains": "sup"}},
}, nil)

// Get a scope without creating it
scope, err = client.GetScope(ctx, "workspace-id", "team")

// Add 1 to 100 existing sessions: those with messages start an asynchronous backfill
err = client.AddSessionsToScope(ctx, "workspace-id", "team", honcho.ScopeSessionsAdd{
    SessionIDs: []string{"session-1", "session-2"},
})

// Track the backfills, by session
status, err := client.GetScopeStatus(ctx, "workspace-id", "team")
for sessionID, backfill := range status.BackfillStatus {
    fmt.Println(sessionID, backfill.State)
}

// List the member sessions (paginated)
sessions, err := client.GetScopeSessions(ctx, "workspace-id", "team", &honcho.GetScopeSessionsOptions{
    Page:    1,
    Size:    20,
    Reverse: true, // Most recently added first
})

// Remove 1 to 100 sessions: if any does not exist, none is removed
err = client.RemoveSessionsFromScope(ctx, "workspace-id", "team", honcho.ScopeSessionsRemove{
    SessionIDs: []string{"session-1", "session-2"},
})

// Remove one session
err = client.RemoveSessionFromScope(ctx, "workspace-id", "team", "session-1")
```

### Webhook Operations

```go
// Get or create a webhook endpoint
webhook, err := client.GetOrCreateWebhookEndpoint("workspace-id", honcho.WebhookEndpointCreate{
    URL: "https://example.com/webhook",
})

// List webhook endpoints
page, err := client.ListWebhookEndpoints("workspace-id", &honcho.ListWebhookEndpointsOptions{
    Page: 1,
    Size: 10,
})

// Delete a webhook endpoint
err = client.DeleteWebhookEndpoint("workspace-id", "endpoint-id")

// Test webhook emission
err = client.TestEmit("workspace-id")
```

### API Key Operations

```go
// Create an API key scoped to a workspace
key, err := client.CreateKey(honcho.CreateKeyRequest{
    WorkspaceID: "workspace-id",
})

// Create an API key scoped to a peer
key, err = client.CreateKey(honcho.CreateKeyRequest{
    WorkspaceID: "workspace-id",
    PeerID:      "user-123",
})

// Create an API key with expiration
expiresAt := time.Now().Add(24 * time.Hour)
key, err = client.CreateKey(honcho.CreateKeyRequest{
    WorkspaceID: "workspace-id",
    ExpiresAt:   &expiresAt,
})
```

## Error Handling

The SDK provides comprehensive error handling with context and error chain support:

**Note on 202/204 Responses:** Some endpoints return `202 Accepted` (e.g., `DeleteSession`) or `204 No Content` (e.g., `DeleteWorkspace`, `DeleteWebhookEndpoint`). These methods return only an error - no result value is provided since the response body is empty.

```go
workspace, err := client.GetOrCreateWorkspace(honcho.CreateWorkspaceRequest{
    ID: "my-workspace",
})
if err != nil {
    // Check for validation errors
    var valErr *honcho.HTTPValidationError
    if errors.As(err, &valErr) {
        // Honcho's own validation errors are a text
        if valErr.Message != "" {
            log.Printf("Validation error: %s", valErr.Message)
        }
        // FastAPI's request validation errors are a list
        for _, detail := range valErr.Detail {
            log.Printf("Validation error at %v: %s", detail.Loc, detail.Msg)
        }
        return
    }
    
    // Handle other errors
    log.Printf("API error: %v", err)
    return
}
```

### Error Types

| Error Type | HTTP Status | Description |
|------------|-------------|-------------|
| `HTTPValidationError` | 422 | Request validation failed: Honcho's own errors in `Message`, FastAPI's request validation errors in `Detail` |
| Generic error | 4xx/5xx | Other API errors with status code |

## Validation

The SDK performs client-side validation for required fields and constraints:

```go
// Validation happens automatically when calling API methods
workspace, err := client.GetOrCreateWorkspace(honcho.CreateWorkspaceRequest{
    ID: "", // Will fail validation: "id is required"
})

// Check validation errors
if err != nil {
    log.Printf("Validation failed: %v", err)
}
```

### Validation Rules

- **Required fields** - Must be provided (e.g., `ID`, `PeerID`)
- **ID format** - Must contain only letters, numbers, underscores, or hyphens
- **Length constraints** - IDs must be 512 characters or less
- **Numeric constraints** - Pagination and limits have min/max values

## Pagination

List methods paginate by page number by default, or by cursor when the `Cursor` of their options is set. Every page type has the same fields as `PageSession`, with its own type of items:

```go
type PageSession struct {
    Items []Session `json:"items"`
    // Set on a numbered page, zero on a cursor page
    Total int `json:"total"`
    Page  int `json:"page"`
    Size  int `json:"size"`
    Pages int `json:"pages"`
    // Set only on a cursor page
    CurrentPage          *string `json:"current_page,omitempty"`
    CurrentPageBackwards *string `json:"current_page_backwards,omitempty"`
    PreviousPage         *string `json:"previous_page,omitempty"` // nil on the first page
    NextPage             *string `json:"next_page,omitempty"`     // nil on the last page
}
```

### By Page Number

```go
// Walk every page - pass nil for req when no filters needed
for number := 1; ; number++ {
    page, err := client.GetSessions(ctx, "workspace-id", nil, &honcho.GetSessionsOptions{
        Page:    number, // Page number (default: 1)
        Size:    50,     // Items per page (default: 50, max: 100)
        Reverse: false,  // Newest first when true (default: false)
    })
    if err != nil {
        log.Fatal(err)
    }
    for _, session := range page.Items {
        // ...
    }
    if number >= page.Pages {
        break
    }
}
```

### By Cursor

Set `Cursor` to an empty cursor, `new(string)`, for the first page, then to each page's `NextPage` until that is nil:

```go
opts := &honcho.GetSessionsOptions{Cursor: new(string), Size: 100}
for opts.Cursor != nil {
    page, err := client.GetSessions(ctx, "workspace-id", nil, opts)
    if err != nil {
        log.Fatal(err)
    }
    for _, session := range page.Items {
        // ...
    }
    opts.Cursor = page.NextPage // nil after the last page
}
```

- A cursor page has no total and no page numbers: `Total`, `Page`, `Size` and `Pages` are zero.
- A cursor holds only a position and is only valid for the list that returned it: keep the other options and the filters unchanged while following cursors, as a cursor used otherwise can return a wrong page without any error. A cursor the server cannot read comes back as an error holding `Invalid cursor`.
- `Page` cannot be combined with `Cursor`: the request is refused before it is sent, as the server would ignore the page.

## Pointer Semantics

The SDK uses Go pointers strategically to distinguish between "not set" and "set to zero value":

### Return Types

Most API methods return pointers to structs or slices:

```go
// Single object returns *Type
workspace, err := client.GetOrCreateWorkspace(req)  // *Workspace
peer, err := client.GetOrCreatePeer(workspaceID, req)  // *Peer

// Multiple object returns *[]Type (pointer to slice)
messages, err := client.SearchWorkspace(workspaceID, req)  // *[]Message
conclusions, err := client.QueryConclusions(workspaceID, req)  // *[]Conclusion

// Paginated returns *PageType
sessions, err := client.GetSessions(workspaceID, req, opts)  // *PageSession
```

**Working with pointer results:**

```go
// ✅ Dereference slice pointers to access elements
result, err := client.SearchWorkspace(ctx, "workspace-id", honcho.WorkspaceMessageSearchOptions{
    Query: "search query",
})
if err != nil {
    log.Fatal(err)
}
// result is *[]Message - dereference to access
for _, msg := range *result {
    fmt.Println(msg.Content)
}

// ✅ Access struct fields directly (no dereference needed)
workspace, err := client.GetOrCreateWorkspace(honcho.CreateWorkspaceRequest{
    ID: "my-workspace",
})
fmt.Println(workspace.ID)  // workspace is *Workspace, field access auto-dereferences
```

### Request Struct Fields

**Pointer fields** (`*int`, `*bool`, `*string`): Used when you need to distinguish "not provided" from "set to zero value":

```go
// ✅ Pointer distinguishes nil vs 0
searchTopK := 5
context, err := client.GetPeerContext("workspace-id", "peer-id", &honcho.GetPeerContextOptions{
    SearchTopK: &searchTopK,  // nil = not provided, &value = explicitly set
})

// ✅ Pointer for bool when you need 3-way logic (configuration updates)
observeMe := true
err = client.SetPeerConfig("workspace-id", "session-id", "peer-id", honcho.SessionPeerConfig{
    ObserveMe: &observeMe,  // nil = keep existing, true/false = change
})
```

**Value fields** (`int`, `bool`) with `omitempty`: Used when zero value means "use server default":

```go
// ✅ Value type with omitempty: 0 or false means "use server default"
sessions, err := client.GetSessions("workspace-id", nil, &honcho.GetSessionsOptions{
    Page:    1,     // 0 = use server default (1)
    Size:    50,    // 0 = use server default (50)
    Reverse: false, // false = use server default (false)
})
```

### Quick Reference

| Field Type | Use Case | Example |
|------------|----------|---------|
| `*int` | Optional numeric where 0 is meaningful | `SearchTopK *int` |
| `*bool` | Configuration updates (3-way logic) | `ObserveMe *bool` |
| `bool` | Pagination/filters (false = default) | `Reverse bool` |
| `*string` | Optional text where empty string is meaningful | `Target *string` |
| `*time.Time` | Optional timestamps | `ExpiresAt *time.Time` |


## Best Practices

### Client Initialization

- ✅ Initialize client once at application startup
- ✅ Store API key securely (environment variable, secrets manager, etc.)
- ✅ Reuse client instance across your application (it's thread-safe)
- ✅ Set appropriate timeouts based on operation type
- ✅ For self-hosted instances, API key is optional

```go
// ✅ Recommended pattern
var client *honcho.Client

func init() {
    apiKey := os.Getenv("HONCHO_API_KEY") // or load from secrets manager
    client = honcho.New(&honcho.Options{
        APIKey: apiKey,
    })
}

// Use the same client instance everywhere
func handler(w http.ResponseWriter, r *http.Request) {
    workspace, _ := client.GetOrCreateWorkspace(...)
}
```

### Error Handling

- ✅ Always check errors from API calls
- ✅ Use `errors.As()` to check for specific error types
- ✅ Log errors with context for debugging

```go
// ✅ Proper error handling
workspace, err := client.GetOrCreateWorkspace(req)
if err != nil {
    var valErr *honcho.HTTPValidationError
    if errors.As(err, &valErr) {
        // Handle validation error
        return
    }
    log.Printf("API error: %v", err)
    return
}
```

### Validation

- ✅ Let the SDK validate required fields automatically
- ✅ Document optional field defaults in your code
- ✅ Handle validation errors gracefully

```go
// ✅ Validation happens automatically
workspace, err := client.GetOrCreateWorkspace(honcho.CreateWorkspaceRequest{
    ID: "valid-id-123",
})
```

### Concurrency

- ✅ Client instances are thread-safe
- ✅ Reuse the same client across goroutines
- ✅ No need for locking around client calls

```go
// ✅ Safe for concurrent use
go func() {
    client.GetPeerContext(workspaceID, peerID, nil)
}()

go func() {
    client.CreateMessagesForSession(workspaceID, sessionID, req)
}()
```

### Request Parameters

- ✅ Pass `nil` for request body structs when all fields are optional and no filters needed
- ✅ Pass `nil` for options structs when using default pagination/settings

```go
// ✅ Pass nil when no filters needed
sessions, err := client.GetSessions("workspace-id", nil, nil)

// ✅ Pass struct when filters are needed
sessions, err := client.GetSessions("workspace-id", &honcho.WorkspaceSessionGet{
    Filters: map[string]any{"status": "active"},
}, nil)
```

### Pointer Helpers

Some optional fields use pointers to distinguish "not set" from "set to zero value". For these, create variables first:

```go
// ✅ Create pointer variables for *int, *string, *float64 fields
searchTopK := 5
context, err := client.GetPeerContext("workspace-id", "user-123", &honcho.GetPeerContextOptions{
    SearchTopK: &searchTopK,
})

// ✅ Reuse pointer variables
observeMe := true
peers := map[string]*honcho.SessionPeerConfig{
    "user-123": {
        ObserveMe: &observeMe,
    },
}
```

**Note:** Some boolean fields use `bool` with `omitempty` for simplicity (e.g., `Reverse` in pagination options), so you can pass `true` or `false` directly without pointers. However, configuration booleans like `ObserveMe` use `*bool` to tell unset (nil) apart from an explicit `true` or `false`. What nil does depends on the endpoint: `SetPeerConfig` keeps the stored value, while `UpdatePeer`, `UpdateWorkspace` and a section sent to `UpdateSession` replace what is stored.

## API Reference

For complete API documentation, visit:

- **Workspaces**: https://docs.honcho.dev/v3/api-reference/endpoint/workspaces
- **Peers**: https://docs.honcho.dev/v3/api-reference/endpoint/peers
- **Sessions**: https://docs.honcho.dev/v3/api-reference/endpoint/sessions
- **Messages**: https://docs.honcho.dev/v3/api-reference/endpoint/messages
- **Conclusions**: https://docs.honcho.dev/v3/api-reference/endpoint/conclusions
- **Dreams**: https://docs.honcho.dev/v3/api-reference/endpoint/dreams
- **Scopes**: https://docs.honcho.dev/v3/api-reference/endpoint/scopes
- **Webhooks**: https://docs.honcho.dev/v3/api-reference/endpoint/webhooks
- **API Keys**: https://docs.honcho.dev/v3/api-reference/endpoint/keys

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
