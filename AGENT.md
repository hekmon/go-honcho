# Honcho Go SDK - Agent Guidelines

This document provides comprehensive guidelines for implementing the Honcho Go SDK. Follow these rules to ensure consistency, completeness, and maintainability across the codebase.

## Table of Contents

1. [Quick Start](#quick-start)
2. [Introduction](#introduction)
3. [File Organization](#file-organization)
4. [Constants & Shared Resources](#constants--shared-resources)
5. [Documentation & API Discovery](#documentation--api-discovery)
6. [Implementation Pattern](#implementation-pattern)
7. [Coding Style](#coding-style)
8. [Validation](#validation)
9. [Low-Level Request Method](#low-level-request-method)
10. [Implementation Verification](#implementation-verification)
11. [Quick Reference](#quick-reference)

---

## Quick Start

**For AI Agents: Follow this workflow when implementing new endpoints**

1. **Read Core Principles** ([Introduction](#introduction)) - Understand type safety, error handling, validation rules

2. **Review File Organization** ([File Organization](#file-organization)) - Know where types vs methods go

3. **Study One Complete Example** - Read `workspace.go` + `workspace_types.go` to understand the pattern

4. **When Implementing a New Endpoint:**
   - Fetch the endpoint `.md` URL from `https://docs.honcho.dev/llms.txt`
   - Copy the pattern from a similar existing method
   - Define all types in `{category}_types.go`
   - Implement methods in `{category}.go` (Client methods only)
   - Run the [Implementation Checklist](#method-implementation-checklist)
   - Verify with [Type Location Check](#type-location-check)

5. **When Unsure → Ask Operator**

### Quick Decision Trees

**Does this struct need a `Validate()` method?**
```
Has required fields (no omitempty)?
├─ YES → Add Validate()
└─ NO → Has constrained optional fields (int/float with 0=default)?
   ├─ YES → Add Validate() (allow 0, validate non-zero only)
   └─ NO → NO Validate() method (omit entirely)
```

**Which BaseURI constant to use?**
```
Endpoint path starts with `/v3/workspaces`?
├─ YES → Use `workspaceBaseURI` (from workspace.go)
└─ NO → Create new `{category}BaseURI` constant
```

**Pointer vs Value type for optional field?**
```
Need to distinguish "not provided" from "zero value"?
├─ YES → Use pointer (*int, *bool, *string) with omitempty
└─ NO → Use value type (int, bool) with omitempty (0 = use default)
```

**Pointer vs Value type for boolean fields?**
```
Is it a boolean field?
├─ YES → Is it pagination/filter options (false = server default)?
│  ├─ YES → Use bool with omitempty (e.g., Reverse bool)
│  └─ NO → Is it configuration update (need "don't change" option)?
│     ├─ YES → Use *bool with omitempty (nil = keep existing)
│     └─ NO → Use bool with omitempty (false = default)
└─ NO → Follow pointer vs value decision above
```

## Introduction

### Purpose

This SDK provides a Go client for the Honcho API v3. All implementations must follow the patterns and conventions documented here to ensure consistency, type safety, and maintainability.

### Document Separation

This project maintains two distinct documentation files:

- **README.md** → **End-user documentation** (how to use the SDK)
  - Installation instructions
  - Quick start examples
  - API usage patterns
  - Best practices for application developers

- **AGENT.md** → **Developer guidelines** (how to implement/maintain the SDK)
  - Implementation patterns and conventions
  - Coding standards and style rules
  - File organization structure
  - API endpoint implementation guidelines
  - Validation and error handling rules

**This file (AGENT.md) is intended for SDK implementers and maintainers, not end users.**

### Core Principles

- **Type Safety**: All API requests and responses use strongly-typed structs
- **Error Handling**: All errors are wrapped with context and support error chaining via `%w`; error types implement `Error()` method
- **Validation**: Client-side validation for required fields and constraints; server handles optional field validation
- **Documentation**: Every method includes block comments with API descriptions and doc URLs
- **Consistency**: Uniform patterns across all category implementations

### Project Structure

```
go-honcho/
├── client.go              # HTTP client setup
├── request.go             # Low-level request method and shared error types
├── {category}.go          # Method implementations (e.g., workspace.go)
└── {category}_types.go    # Type definitions (e.g., workspace_types.go)
```

---

## File Organization

### Category Files

Each API category gets its own pair of files:

- `{category}.go` - Client methods only (e.g., `func (c *Client) GetWorkspace(...)`)
- `{category}_types.go` - Type definitions only (structs can have their own methods like `Validate()`)

**Example:** `workspace.go` / `workspace_types.go`, `peer.go` / `peer_types.go`

### Type Definitions

**Place all struct types in `*_types.go` files:**

```go
// ✅ In workspace_types.go
type Workspace struct {
    ID            string                  `json:"id"`
    CreatedAt     time.Time               `json:"created_at"`
    Metadata      map[string]any          `json:"metadata,omitempty"`
    Configuration *WorkspaceConfiguration `json:"configuration,omitempty"`
}
```

### Method Implementations

**Place methods in corresponding `*.go` files:**

```go
// ✅ In workspace.go
func (c *Client) GetOrCreateWorkspace(req CreateWorkspaceRequest) (result *Workspace, err error) {
    // implementation
}
```

### Type Category Verification

**After implementing types, verify they're in the correct category file:**

```bash
# ✅ Verify session types are in session_types.go
grep "type Session" session_types.go  # Should find Session*, PageSession, etc.
grep "type Session" session.go        # Should find nothing

# ✅ Verify peer types are in peer_types.go  
grep "type Peer" peer_types.go  # Should find Peer*, PagePeer, etc.
grep "type Peer" peer.go        # Should find nothing
```

**Shared types:** If a type is used by multiple categories (e.g., `MessageSearchOptions` for both workspace and session search), place it in the **primary category** where it's defined in the API docs, or create a `shared_types.go` if used across 3+ categories.

**❌ Don't leave types in the wrong category file even if they're related** - Sessions and Peers are related but their types belong in separate files.

**Canonical type location:** When implementing a new category, define all types for that category in `{category}_types.go`, even if they were previously defined elsewhere. For example, when implementing the messages category, the `Message` struct should be defined in `message_types.go`, not in `workspace_types.go` where it might have been used before. Other categories should then reuse the type from its canonical home.

### Pointer vs Value Type Decision Guide

**Use pointer (`*int`, `*bool`, `*string`) when:**
- You need to distinguish between "not provided" (nil) and "explicitly set to zero value"
- The field is truly optional with no meaningful default
- For `*bool`: when you need 3-way logic (nil = "don't change", true = "enable", false = "disable")

```go
// ✅ Pointer distinguishes nil vs 0
type GetPeerContextOptions struct {
    SearchTopK *int `json:"search_top_k,omitempty"`  // nil = not provided, 0 = explicitly zero
}

// ✅ Pointer for bool when you need 3-way logic (configuration updates)
type SessionPeerConfig struct {
    ObserveMe *bool `json:"observe_me,omitempty"`  // nil = keep existing, true/false = change
}
```

**Use value type (`int`, `bool`) with `omitempty` when:**
- Zero value (0, false) should mean "use server default"
- No need to distinguish between nil and zero
- For `bool`: when false is the server default and you never need to explicitly send false

```go
// ✅ Value type with omitempty: 0 means "use default"
type ConclusionQuery struct {
    // TopK is the number of results to return (0=use default: 10, min: 1, max: 100)
    TopK int `json:"top_k,omitempty"`  // 0 omitted from JSON, server uses default (10)
}

// ✅ bool with omitempty: false means "use default" (pagination, filters)
type GetMessagesOptions struct {
    // Reverse is whether to reverse the order of results (default: false)
    Reverse bool `json:"reverse,omitempty"`  // false omitted, true sent
}
```

```go
// ❌ Don't use pointer when 0 should mean "use default"
TopK *int `json:"top_k,omitempty"`  // forces caller to use &value instead of just value

// ❌ Don't use value type when you need to distinguish nil from 0
SearchTopK int `json:"search_top_k,omitempty"`  // can't tell if 0 means "not set" or "set to 0"

// ❌ Don't use *bool for simple boolean flags where false = default
Reverse *bool `json:"reverse,omitempty"`  // unnecessarily complex for pagination
```

---

## Constants & Shared Resources

### Base URI Constant

**Most endpoints start with `/v3/workspaces`. Define `workspaceBaseURI` once in `workspace.go` and reuse it in ALL category files that use this base path:**

**Decision rule:** Check the OpenAPI spec's endpoint path:
- If path starts with `/v3/workspaces` → **reuse `workspaceBaseURI`** (e.g., `/v3/workspaces/{id}/peers`, `/v3/workspaces/{id}/webhooks`)
- If path starts with a **different base** → **create a new `*BaseURI` constant** (e.g., `/v3/keys`, `/v3/admin`)

```go
// ✅ In workspace.go - define once
const (
    workspaceBaseURI = "/v3/workspaces"
)

// ✅ In peer.go, session.go, webhook.go, etc. - ALWAYS reuse workspaceBaseURI
requestURL := c.baseURL.JoinPath(workspaceBaseURI, workspaceID, "peers")
requestURL := c.baseURL.JoinPath(workspaceBaseURI, workspaceID, "webhooks")

// ✅ In key.go - different base path, so create keyBaseURI
const (
    keyBaseURI = "/v3/keys"  // ✅ correct - genuinely different base path
)
requestURL := c.baseURL.JoinPath(keyBaseURI)

// ❌ Don't define redundant constants for /v3/workspaces paths
const (
    peerBaseURI = "/v3/workspaces"  // WRONG! Just use workspaceBaseURI
)

// ❌ Don't create constants for path extensions (this is workspaceBaseURI + "/peers")
const (
    peersBaseURI = "/v3/workspaces/peers"  // WRONG! Use workspaceBaseURI + "peers"
)
```

### Pattern Definitions

```go
// ✅ Define regex patterns as package-level variables
var workspaceIDPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
```

### Clean Imports

```go
// ✅ Group standard library imports
import (
    "errors"
    "fmt"
    "net/http"
    "regexp"
    "time"
)

// ❌ Avoid unused imports
```

---

## Documentation & API Discovery

### API Index

**Start by discovering all available endpoints:**

- **Complete API Index**: Fetch `https://docs.honcho.dev/llms.txt` to discover ALL available endpoints
- Use the index to find all endpoints for a category and ensure complete implementation

### API References

**Include BOTH a descriptive block comment AND the Honcho docs URL (WITHOUT `.md` extension for IDE visibility):**

```go
// CreateConclusions creates one or more Conclusions.
//
// Conclusions are logical certainties derived from interactions between Peers.
// They form the basis of a Peer's Representation.
//
// https://docs.honcho.dev/v3/api-reference/endpoint/conclusions/create-conclusions
func (c *Client) CreateConclusions(workspaceID string, req ConclusionBatchCreate) (result []*Conclusion, err error) {
```

**Block Comment Guidelines:**
- ✅ Include 1-3 sentences explaining what the endpoint does
- ✅ Include the API doc URL (without `.md` extension)
- ✅ Place the comment directly above the function signature
- ✅ **Always use 2+ paragraphs** - first line summarizes, second paragraph provides additional context from the OpenAPI spec
- ✅ **Copy the description from the OpenAPI spec** - even for simple endpoints, never use just a one-liner

**Note for Agents:** When fetching documentation, append `.md` to the URL:
- Agent fetch: `https://docs.honcho.dev/v3/api-reference/endpoint/workspaces/get-or-create-workspace.md`
- IDE comment: `https://docs.honcho.dev/v3/api-reference/endpoint/workspaces/get-or-create-workspace`

**Example for simple endpoints:**
```go
// ✅ Good: Simple endpoint with proper description
// TestEmit tests publishing a webhook event.
//
// This endpoint triggers a test webhook event to verify the endpoint is configured correctly.
//
// https://docs.honcho.dev/v3/api-reference/endpoint/webhooks/test-emit
func (c *Client) TestEmit(workspaceID string) (err error) {

// ❌ Bad: Missing descriptive second paragraph
// TestEmit tests publishing a webhook event.
//
// https://docs.honcho.dev/v3/api-reference/endpoint/webhooks/test-emit
func (c *Client) TestEmit(workspaceID string) (err error) {
```

### Documentation Trick

Honcho docs support `.md` extension for direct markdown access:
- API Index: `https://docs.honcho.dev/llms.txt`
- Individual endpoint (for agents): Add `.md` to any docs URL

### Complete API Schema Implementation

**ALL information in the endpoint `.md` URL must be implemented:**

- ✅ ALL request/response schemas from the OpenAPI spec
- ✅ ALL error types and HTTP status codes (e.g., 422 Validation Error)
- ✅ Implement error types in `request.go` with proper handling per status code

```yaml
# Example from OpenAPI spec - must be implemented:
'422':
  description: Validation Error
  content:
    application/json:
      schema:
        $ref: '#/components/schemas/HTTPValidationError'
```

```go
// ✅ Implement ALL error schemas from the API docs
type HTTPValidationError struct {
    Detail []ValidationError `json:"detail"`
}

type ValidationError struct {
    Loc   []any  `json:"loc"`
    Msg   string `json:"msg"`
    Type  string `json:"type"`
    Input any    `json:"input,omitempty"`
    Ctx   any    `json:"ctx,omitempty"`
}

// ✅ Handle ALL HTTP status codes from the API docs in request()
case http.StatusUnprocessableEntity:
    var valErr HTTPValidationError
    if err = json.NewDecoder(resp.Body).Decode(&valErr); err != nil {
        err = fmt.Errorf("failed to decode validation error: %w", err)
        return
    }
    err = fmt.Errorf("validation error: %w", &valErr)
    return

// ❌ Don't ignore error schemas or HTTP status codes from the API docs
// ❌ Don't leave error responses as generic strings
```

---

## Implementation Pattern

### Method Implementation Checklist

Before finalizing a method, verify:

- [ ] Block comment with **2+ paragraphs** (1st: what it does, 2nd: additional context/description from API docs)
- [ ] API doc URL in block comment (without `.md` extension)
- [ ] Named returns: `(result *Type, err error)`
- [ ] Validation called ONLY if `Validate()` method exists on the request type
- [ ] If ALL fields are optional (no `Validate()` method), do NOT call `req.Validate()`
- [ ] Uses `workspaceBaseURI` constant (or appropriate `*BaseURI`)
- [ ] Uses `c.request()` method
- [ ] Errors wrapped with `%w`
- [ ] Naked returns (no explicit return values)
- [ ] Result initialized correctly (`new(Type)` or `make([]*Type, 0, capacity)`)

### Standard Method Structure

```go
func (c *Client) MethodName(req RequestType) (result *ResultType, err error) {
    // 1. Validate mandatory parameters (ONLY if Validate() method exists)
    if err = req.Validate(); err != nil {
        return
    }
    
    // 2. Construct URL using JoinPath
    requestURL := c.baseURL.JoinPath(endpointPath)
    
    // 3. Initialize result
    result = new(ResultType)
    
    // 4. Make request
    if _, err = c.request(http.MethodPost, requestURL, nil, req, &result); err != nil {
        err = fmt.Errorf("failed to execute request: %w", err)
        return
    }
    
    // 5. Return
    return
}
```

**Note:** If the request type has ONLY optional fields (no `Validate()` method), skip step 1 entirely. Do NOT call `req.Validate()` when the method doesn't exist.

### Method Signature Patterns

**Check existing similar methods for parameter order consistency:**

```go
// ✅ Match the pattern used in similar methods (e.g., session.go)
// When a method has both request body and options parameters:
func (c *Client) GetSessions(workspaceID string, req *SessionGet, opts *GetSessionsOptions) (result *PageSession, err error)
//                                          ^^^ request body first         ^^^^ options second

// ❌ Don't invent new parameter orderings - check existing methods first
func (c *Client) GetSessions(workspaceID string, opts *GetSessionsOptions, req *SessionGet) // wrong order!
```

**Pattern:** Request body parameters come BEFORE optional query parameter structs. This matches the API structure (body is primary, query params are modifiers).

### Path Parameters

**When the URL contains path parameters (e.g., `{workspace_id}`, `{session_id}`):**

```go
// ✅ Pass path parameters as separate string arguments
func (c *Client) DeleteWorkspace(workspaceID string) (err error) {
    if workspaceID == "" {
        err = errors.New("workspaceID is required")
        return
    }
    requestURL := c.baseURL.JoinPath(workspaceBaseURI, workspaceID)
    // ...
}

// ✅ Multiple path parameters
func (c *Client) CreateMessagesWithFile(workspaceID, sessionID string, req MessageUpload) (result []*Message, err error) {
    requestURL := c.baseURL.JoinPath(workspaceBaseURI, workspaceID, "sessions", sessionID, "messages", "upload")
    // ...
}

// ❌ Don't concatenate path strings manually
requestURL := c.baseURL.JoinPath(workspaceBaseURI + "/" + workspaceID)  // WRONG!
```

### HTTP Methods

**Use the correct HTTP method for each operation:**

| Method | Use Case | Example |
|--------|----------|---------|
| `GET` | Retrieve resources | GetPeer, GetSession |
| `POST` | Create resources or complex queries | GetOrCreateWorkspace, CreateMessages |
| `PUT` | Full resource updates | UpdateWorkspace |
| `DELETE` | Remove resources | DeleteWorkspace, DeletePeer |

```go
// ✅ Use appropriate HTTP method
http.MethodGet    // for retrieval
http.MethodPost   // for creation or search operations
http.MethodPut    // for updates
http.MethodDelete // for deletion

// ❌ Don't use wrong methods
http.MethodPost   // for simple retrieval (use GET)
http.MethodGet    // for operations with request body (use POST)
```

**Note:** Some retrieval operations use POST when they require a request body (e.g., search with filters). Follow the OpenAPI spec.

### Query Parameters

**For GET requests with query parameters:**

```go
// ✅ Build query parameters using url.Values
func (c *Client) GetSessions(workspaceID string, opts *GetSessionsOptions) (result *PageSession, err error) {
    requestURL := c.baseURL.JoinPath(workspaceBaseURI, workspaceID, "sessions")
    
    // Add query parameters
    if opts != nil {
        query := requestURL.Query()
        if opts.Page > 0 {
            query.Set("page", strconv.Itoa(opts.Page))
        }
        if opts.Size > 0 {
            query.Set("size", strconv.Itoa(opts.Size))
        }
        // ✅ For bool fields with omitempty: check value directly
        if opts.Reverse {
            query.Set("reverse", "true")
        }
        requestURL.RawQuery = query.Encode()
    }
    
    result = new(PageSession)
    if _, err = c.request(http.MethodGet, requestURL, nil, nil, &result); err != nil {
        err = fmt.Errorf("failed to get sessions: %w", err)
        return
    }
    return
}
```

**For POST requests with query parameters:**

```go
// ✅ Combine query parameters with request body
func (c *Client) SearchMessages(workspaceID string, req MessageSearchOptions, opts *SearchOptions) (result *MessageSearchResult, err error) {
    if err = req.Validate(); err != nil {
        return
    }
    
    requestURL := c.baseURL.JoinPath(workspaceBaseURI, workspaceID, "messages", "search")
    
    // Add query parameters (if any)
    if opts != nil {
        query := requestURL.Query()
        if opts.Limit > 0 {
            query.Set("limit", strconv.Itoa(opts.Limit))
        }
        requestURL.RawQuery = query.Encode()
    }
    
    result = new(MessageSearchResult)
    if _, err = c.request(http.MethodPost, requestURL, nil, req, &result); err != nil {
        err = fmt.Errorf("failed to search messages: %w", err)
        return
    }
    return
}
```

**Query Parameter Guidelines:**
- ✅ Use `requestURL.Query()` to build query strings
- ✅ Only add parameters with non-zero/non-nil values (unless zero is meaningful)
- ✅ Encode query string with `requestURL.RawQuery = query.Encode()`
- ✅ Document default values and constraints in the options struct
- ❌ Don't manually concatenate query strings (`?page=1&size=10`)

### Response Handling

**All API responses are handled by the `request()` method:**

```go
// ✅ Standard JSON response - result is populated automatically
result = new(Peer)
if _, err = c.request(http.MethodGet, requestURL, nil, nil, &result); err != nil {
    return
}
// result is now populated with JSON-decoded data

// ✅ Array response - use slice pointer
result = make([]*Message, 0)
if _, err = c.request(http.MethodPost, requestURL, nil, req, &result); err != nil {
    return
}

// ✅ No response body (204 No Content, 202 Accepted)
if _, err = c.request(http.MethodDelete, requestURL, nil, nil, nil); err != nil {
    return
}
// No result to decode

// ✅ Raw response body (e.g., for file downloads)
result = new(bytes.Buffer)
if _, err = c.request(http.MethodGet, requestURL, nil, nil, &result); err != nil {
    return
}
// result contains raw bytes
```

**Response Type Guidelines:**
- ✅ Use `new(Type)` for single object responses
- ✅ Use `make([]*Type, 0, capacity)` for array responses (ensures JSON encodes as `[]` not `null`)
- ✅ Use `nil` for no response body (DELETE, some POST operations)
- ✅ Use `*bytes.Buffer` for raw binary responses
- ❌ Don't manually decode JSON responses (request() handles this)
- ❌ Don't pass value types as result (always use pointers)

### Pagination

**Paginated responses follow a consistent pattern:**

```go
// ✅ Paginated response structure
type PageSession struct {
    Sessions   []Session `json:"sessions"`
    Page       int       `json:"page"`
    Size       int       `json:"size"`
    TotalCount int       `json:"total_count"`
    TotalPages int       `json:"total_pages"`
}

// ✅ Pagination in options struct
type GetSessionsOptions struct {
    // Page is the page number (default: 1, minimum: 1)
    Page int `json:"page,omitempty"`
    // Size is the page size (default: 50, minimum: 1, maximum: 100)
    Size int `json:"size,omitempty"`
    // Reverse is whether to reverse the order of results (default: false)
    Reverse bool `json:"reverse,omitempty"`  // false = use server default
}
```

**Pagination Guidelines:**
- ✅ Use `Page*` prefix for paginated response types (e.g., `PageSession`, `PagePeer`)
- ✅ Include pagination metadata in response (page, size, total_count, total_pages)
- ✅ Document default values and constraints for pagination parameters
- ✅ Use value types for Page/Size with `omitempty` (0 = use server default)
- ✅ Use `bool` with `omitempty` for Reverse (false = use server default)
- ✅ Simplify query parameter handling: `if opts.Reverse { query.Set("reverse", "true") }`
- ❌ Don't omit pagination metadata from response types
- ❌ Don't use pointer types for Page/Size/Reverse in pagination options

---

## Coding Style

### Function Signatures

```go
// ✅ Use named returns
func (c *Client) GetOrCreateWorkspace(req CreateWorkspaceRequest) (result *Workspace, err error)

// ❌ Avoid anonymous returns
func (c *Client) GetOrCreateWorkspace(req CreateWorkspaceRequest) (*Workspace, error)
```

### Return Statements

```go
// ✅ Use naked returns for clarity
if err != nil {
    err = fmt.Errorf("failed to execute request: %w", err)
    return
}
return

// ❌ Avoid redundant return values
return nil, err
return result, nil
```

### Error Handling

```go
// ✅ Wrap errors with context and use %w for chaining
err = fmt.Errorf("failed to parse URL: %w", err)
err = fmt.Errorf("failed to execute request: %w", err)

// ✅ Check errors immediately and return early
if err != nil {
    return
}

// ✅ Use fmt.Errorf with %w for error message formatting in loops
for i, msg := range req.Messages {
    if err := msg.Validate(); err != nil {
        return fmt.Errorf("message %d: %w", i+1, err)  // preserves error chain
    }
}

// ❌ Don't use string concatenation or string(rune()) for error messages
return errors.New("message " + string(rune(i+1)) + ": " + err.Error())  // breaks for i>=9!
```

### Error Types

**All custom error types must implement the `Error()` method:**

```go
// ✅ Error types must implement Error() method and be wrapped with %w
type HTTPValidationError struct {
    Detail []ValidationError `json:"detail"`
}

func (e *HTTPValidationError) Error() string {
    return fmt.Sprintf("validation error: %v", e.Detail)
}

// In request handling:
err = fmt.Errorf("validation error: %w", &valErr)

// ❌ Don't return error types without Error() method
// ❌ Don't forget to wrap with %w for errors.As() support
```

**Error Type Guidelines:**

1. **Implement `Error()` method**: All custom error types must satisfy the `error` interface
2. **Wrap with `%w`**: Always wrap error types with `%w` to preserve the error chain for `errors.As()` and `errors.Is()`
3. **Use pointer receivers**: Define `Error()` on pointer receivers for consistency
4. **Provide context**: Wrap error types with descriptive context messages

**Example error type implementation:**

```go
// ✅ Complete error type with Error() method
type NotFoundError struct {
    ResourceType string
    ResourceID   string
}

func (e *NotFoundError) Error() string {
    return fmt.Sprintf("%s not found: %s", e.ResourceType, e.ResourceID)
}

// Usage in method:
err = fmt.Errorf("failed to get resource: %w", &NotFoundError{ResourceType: "peer", ResourceID: peerID})
```

**Checking for specific error types:**

```go
// ✅ Check for specific error types using errors.As()
var valErr *HTTPValidationError
if errors.As(err, &valErr) {
    // Handle validation error specifically
    for _, detail := range valErr.Detail {
        // Process each validation error
    }
}

// ❌ Don't use type assertions without checking
if valErr, ok := err.(*HTTPValidationError); ok {  // breaks with wrapped errors!
```

### Type Declarations

```go
// ✅ Use 'any' instead of 'interface{}'
Metadata map[string]any `json:"metadata,omitempty"`

// ✅ Use pointers for optional nested structs that can be omitempty
Configuration *WorkspaceConfiguration `json:"configuration,omitempty"`

// ❌ Avoid value types for optional nested structs
Configuration WorkspaceConfiguration `json:"configuration,omitempty"`
```

### Generics and Advanced Type Patterns

**This SDK avoids generics for API types. Use concrete types for clarity and IDE support:**

```go
// ✅ Use concrete types for API requests and responses
func (c *Client) GetPeer(workspaceID, peerID string) (result *Peer, err error)

// ❌ Avoid generics for API types - reduces IDE autocomplete and clarity
func (c *Client) GetResource[T any](workspaceID, resourceType, resourceID string) (result *T, err error)
```

**When to use `any`:**
- ✅ Map values: `map[string]any` for flexible metadata
- ✅ Slice elements: `[]any` for heterogeneous lists
- ❌ NOT for request/response types - use concrete structs

**Pointer semantics for optional fields:**

```go
// ✅ Pointer to distinguish nil vs zero value
type GetPeerContextOptions struct {
    SearchTopK *int     `json:"search_top_k,omitempty"`  // nil = not provided, 0 = explicitly zero
    Filters    *string  `json:"filters,omitempty"`       // nil = not provided, "" = empty string
}

// ✅ Value type when 0 means "use server default"
type ConclusionQuery struct {
    TopK int `json:"top_k,omitempty"`  // 0 = use server default (10)
}

// ❌ Don't use pointer when value type suffices
TopK *int `json:"top_k,omitempty"`  // forces caller to use &value
```

**Slice initialization:**

```go
// ✅ Initialize slices that will be populated
result = make([]*Message, 0, len(req.Messages))

// ✅ Initialize empty slices for JSON responses
result = new(PageSession)  // PageSession.Sessions will be []Session or nil

// ❌ Don't leave slices as nil if they should be empty arrays in JSON
result = &PageSession{Sessions: nil}  // encodes as null, not []
```

### Time.Time Handling

**The Honcho API uses RFC 3339 timestamps. Go's `time.Time` handles this automatically with JSON:**

```go
// ✅ time.Time fields work automatically with JSON encoding/decoding
type Workspace struct {
    ID        string    `json:"id"`
    CreatedAt time.Time `json:"created_at"`  // RFC 3339 format
}

// ✅ time.Time pointers for optional timestamps
type Session struct {
    EndedAt *time.Time `json:"ended_at,omitempty"`  // nil = not ended
}
```

**Time Handling Guidelines:**
- ✅ Use `time.Time` for timestamp fields (automatically handles RFC 3339)
- ✅ Use `*time.Time` for optional timestamps (nil = not set)
- ✅ Use `omitempty` with pointer timestamps to omit nil values
- ❌ Don't use string types for timestamps (lose type safety)
- ❌ Don't manually parse/format RFC 3339 timestamps (time.Time does this)

### JSON Tag Conventions

**All struct fields must have proper JSON tags:**

```go
// ✅ Use snake_case for JSON tags (API convention)
type CreateWorkspaceRequest struct {
    WorkspaceID   string         `json:"workspace_id"`
    CreatedAt     time.Time      `json:"created_at"`
    Metadata      map[string]any `json:"metadata,omitempty"`
    Configuration *WorkspaceConfig `json:"configuration,omitempty"`
}

// ✅ Use omitempty for optional fields
type UpdateRequest struct {
    Metadata map[string]any `json:"metadata,omitempty"`  // optional
}

// ✅ Required fields omit omitempty
type CreateRequest struct {
    ID      string `json:"id"`      // required
    Content string `json:"content"` // required
}
```

**JSON Tag Guidelines:**
- ✅ Use snake_case for all JSON tags (matches API convention)
- ✅ Add `omitempty` for optional fields (prevents sending zero values)
- ✅ Omit `omitempty` for required fields (ensures they're always sent)
- ✅ Use pointers for optional nested structs (with `omitempty`)
- ❌ Don't use camelCase in JSON tags (API uses snake_case)
- ❌ Don't omit JSON tags (explicit is better than implicit)
- ❌ Don't use `omitempty` on required fields

---

## Validation

### When to Add Validate() Methods

Use this decision tree to determine if a struct needs a `Validate()` method:

```
Does the struct have ANY required fields (no omitempty)?
├─ YES → Add Validate() method
│  └─ Validate required fields and constraints
│
└─ NO → Does the struct have optional fields with constraints?
   ├─ YES, and field is int/float with 0=default → Add Validate()
   │  └─ Allow 0, validate non-zero values against constraints
   │
   └─ NO (all fields truly optional, no constraints) → NO Validate()
      └─ Do not add Validate() method - caller can pass empty struct
```

**Key Principle:** Validate only what the client can verify before making the API call. Let the server handle optional field validation.

**Examples:**

```go
// ✅ Has required field → needs Validate()
type ConclusionCreate struct {
    Content string `json:"content"`  // required
}

// ✅ Has optional field with constraints → needs Validate()
type ConclusionQuery struct {
    Query string `json:"query"`           // required
    TopK  int    `json:"top_k,omitempty"` // optional, but constrained
}

// ❌ All fields optional, no constraints → NO Validate()
type ConclusionGet struct {
    Filters map[string]any `json:"filters,omitempty"`  // only field, optional
}
// No Validate() method - caller can pass empty struct
```

### Validate() Methods

**Add `Validate()` for mandatory parameters and constrained optional parameters:**

```go
// ✅ Validate required fields with clear error messages
func (req CreateWorkspaceRequest) Validate() error {
    if req.ID == "" {
        return errors.New("id is required")
    }
    if len(req.ID) > 100 {
        return errors.New("id must be 100 characters or less")
    }
    if !workspaceIDPattern.MatchString(req.ID) {
        return errors.New("id must contain only letters, numbers, underscores, or hyphens")
    }
    return nil
}

// ✅ Validate constrained optional fields (allow 0 = use default)
func (req ConclusionQuery) Validate() error {
    if req.Query == "" {
        return errors.New("query is required")
    }
    // Allow 0 (means "use server default"), validate non-zero values
    if req.TopK != 0 && (req.TopK < 1 || req.TopK > 100) {
        return errors.New("top_k must be between 1 and 100")
    }
    return nil
}

// ❌ Don't validate optional parameters (let server handle those)

// ❌ Don't add Validate() if ALL fields are optional - omit the method entirely
type MessageUpdate struct {
    Metadata map[string]any `json:"metadata,omitempty"`  // only field, and it's optional
}
// No Validate() method needed - caller can pass empty struct{}
```

**Rules:**
1. If a request type has ONLY optional fields (all fields have `omitempty`), do NOT add a `Validate()` method
2. The method should be omitted entirely, not return `nil`
3. For optional int/float fields where 0 means "use server default", allow 0 and validate only non-zero values

### Call Validation in Methods

```go
// ✅ Call Validate() at the start of methods (ONLY if Validate() method exists)
func (c *Client) GetOrCreateWorkspace(req CreateWorkspaceRequest) (result *Workspace, err error) {
    if err = req.Validate(); err != nil {
        return
    }
    // ... rest of implementation
}

// ✅ When ALL fields are optional, do NOT call Validate() (method doesn't exist)
func (c *Client) CreateKey(req CreateKeyRequest) (result *Key, err error) {
    // No validation call - all fields are optional, Validate() method doesn't exist
    requestURL := c.baseURL.JoinPath(keyBaseURI)
    // ... rest of implementation
}

// ❌ Don't call Validate() when the method doesn't exist
func (c *Client) CreateKey(req CreateKeyRequest) (result *Key, err error) {
    if err = req.Validate(); err != nil {  // ERROR: Validate() method doesn't exist!
        return
    }
    // ... rest of implementation
}
```

**Rule:** When ALL fields are optional and you omit the `Validate()` method, ensure NO method calls `req.Validate()` on that type. Check if the `Validate()` method exists before calling it.

### Validation Constraint Precision

**Copy exact constraints from the OpenAPI spec - do not approximate:**

**For int fields with omitempty where 0 means "use default":**

```go
// ✅ Allow 0 (means "use server default"), validate non-zero values
type ConclusionQuery struct {
    // TopK is the number of results to return (0=use default: 10, min: 1, max: 100)
    TopK int `json:"top_k,omitempty"`
}

func (req ConclusionQuery) Validate() error {
    if req.Query == "" {
        return errors.New("query is required")
    }
    // ✅ Allow 0, validate only if non-zero
    if req.TopK != 0 && (req.TopK < 1 || req.TopK > 100) {
        return errors.New("top_k must be between 1 and 100")
    }
    return nil
}

// ❌ Don't reject 0 when it means "use default"
if req.TopK < 1 || req.TopK > 100 {  // rejects 0 incorrectly!
    return errors.New("top_k must be between 1 and 100")
}
```

**For float64 validation:**

```go
// ✅ Use float literals for float64 validation
if req.Distance != nil && (*req.Distance < 0.0 || *req.Distance > 1.0) {
    return errors.New("distance must be between 0 and 1")
}

// ❌ Avoid mixing int and float literals (confusing)
if req.Distance != nil && (*req.Distance < 0 || *req.Distance > 1) {
```

**Copy exact constraints from the OpenAPI spec - do not approximate:**

```yaml
# From API docs:
limit:
  type: integer
  maximum: 100
  minimum: 1  # ← Must match exactly, not 0
  default: 10
```

```go
// ✅ Match the spec exactly
func (req MessageSearchOptions) Validate() error {
    if req.Query == "" {
        return errors.New("query is required")
    }
    if req.Limit < 1 || req.Limit > 100 {  // exact min/max from spec
        return errors.New("limit must be between 1 and 100")
    }
    return nil
}

// ❌ Don't approximate or assume
func (req MessageSearchOptions) Validate() error {
    if req.Limit < 0 || req.Limit > 100 {  // wrong minimum!
        return errors.New("limit must be between 0 and 100")
    }
    return nil
}
```

**Checklist for Validate() methods:**
- ✅ Verify all numeric constraints (min/max) against OpenAPI spec
- ✅ Verify all string constraints (length, pattern) against OpenAPI spec
- ✅ Verify all required fields match the `required` array in the spec
- ✅ Don't add validation for optional fields (server handles those)
- ✅ Copy constraint values directly from the spec, don't guess

### Struct Field Documentation

**Document default values and constraints for struct fields:**

```go
// ✅ Document defaults and constraints
type GetAllWorkspacesOptions struct {
    // Page is the page number (default: 1, minimum: 1)
    Page int
    // Size is the page size (default: 50, minimum: 1, maximum: 100)
    Size int
}

// ❌ Avoid undocumented fields
type GetAllWorkspacesOptions struct {
    Page int
    Size int
}
```

**Document zero-value defaults explicitly:**

```go
// ✅ Document when 0 means "use server default"
type ConclusionQuery struct {
    // TopK is the number of results to return (0=use default: 10, min: 1, max: 100)
    TopK int `json:"top_k,omitempty"`
    
    // Distance is the maximum cosine distance threshold (optional, 0-1)
    Distance *float64 `json:"distance,omitempty"`
}

// ✅ Document value type defaults clearly
type ListConclusionsOptions struct {
    // Reverse is whether to reverse the order of results (default: false)
    Reverse bool `json:"reverse,omitempty"`  // false = use server default
    
    // Page is the page number (default: 1, minimum: 1)
    Page int `json:"page,omitempty"`  // 0 = use server default (1)
    
    // Size is the page size (default: 50, minimum: 1, maximum: 100)
    Size int `json:"size,omitempty"`  // 0 = use server default (50)
}

// ✅ Document when pointer is needed (3-way logic)
type SessionPeerConfig struct {
    // ObserveMe indicates whether to enable observation (optional)
    ObserveMe *bool `json:"observe_me,omitempty"`  // nil = keep existing, true/false = change
    
    // ObserveOthers indicates whether to observe other peers (optional)
    ObserveOthers *bool `json:"observe_others,omitempty"`  // nil = keep existing
}

// ❌ Avoid ambiguous documentation
// TopK is the number of results to return (min: 1, max: 100)  // doesn't mention 0!
```

This is critical for optional parameters where `0` means "use server default".

**Boolean Field Decision Guide:**

```go
// ✅ Use bool with omitempty for pagination/filters (false = default)
type GetMessagesOptions struct {
    Reverse bool `json:"reverse,omitempty"`  // false omitted from JSON
}

// ✅ Use *bool for configuration updates (need 3-way logic)
type SessionPeerConfig struct {
    ObserveMe *bool `json:"observe_me,omitempty"`  // nil = don't change
}

// ❌ Don't use *bool for simple flags
Reverse *bool `json:"reverse,omitempty"`  // unnecessarily complex
```

---

## Low-Level Request Method

### Usage

- All API methods MUST use the `request()` method from `request.go`
- This method handles HTTP request building, execution, and response parsing
- Do NOT call `http.Client.Do()` directly in API methods

### Method Signature

```go
func (c *Client) request(method string, requestURL *url.URL, headers http.Header, body, result any) (responseHeaders http.Header, err error)
```

**Parameters:**
- `method`: HTTP method (GET, POST, PUT, DELETE, etc.)
- `requestURL`: Full URL constructed with `JoinPath()`
- `headers`: Optional HTTP headers (nil for most cases)
- `body`: Request body (string, url.Values, struct/map, or io.Reader for multipart)
- `result`: Pointer to result struct, `*bytes.Buffer` for raw response, or nil for no response

**Returns:**
- `responseHeaders`: HTTP response headers (for advanced use cases)
- `err`: Error if request failed

### Automatic Content-Type Handling

The `request()` method automatically sets Content-Type based on body type:

| Body Type | Content-Type |
|-----------|-------------|
| `string` | `text/plain; charset=utf-8` |
| `url.Values` | `application/x-www-form-urlencoded` |
| `struct`/`map` | `application/json; charset=utf-8` |
| `io.Reader` | Must set manually via headers (for multipart forms) |

### Supported Body Types

```go
// ✅ String body (text/plain)
body := "plain text"

// ✅ URL Values (application/x-www-form-urlencoded)
body := url.Values{"key": []string{"value"}}

// ✅ Struct/Map (application/json) - default
body := MyStruct{Field: "value"}

// ✅ io.Reader for multipart/form-data or binary data
body := &bytes.Buffer{}  // *bytes.Buffer implements io.Reader
```

**Important for multipart forms:** The `request()` method detects `io.Reader` types (like `*bytes.Buffer`) and passes them directly as the request body without modification. This preserves binary data integrity for multipart forms.

### Multipart Form Handling

**When the API requires multipart/form-data (e.g., file uploads):**

```go
// ✅ Complete multipart form pattern
func (c *Client) CreateMessagesWithFile(workspaceID, sessionID string, req MessageUpload) (result []*Message, err error) {
    // Validate request
    if err = req.Validate(); err != nil {
        return
    }
    // Construct URL
    requestURL := c.baseURL.JoinPath(workspaceBaseURI, workspaceID, "sessions", sessionID, "messages", "upload")
    // Build multipart form
    bodyBuffer := &bytes.Buffer{}
    writer := multipart.NewWriter(bodyBuffer)
    // Add file field
    fileWriter, err := writer.CreateFormFile("file", req.Filename)
    if err != nil {
        err = fmt.Errorf("failed to create form file: %w", err)
        return
    }
    if _, err = fileWriter.Write(req.File); err != nil {
        err = fmt.Errorf("failed to write file to form: %w", err)
        return
    }
    // Add other form fields
    if err = writer.WriteField("peer_id", req.PeerID); err != nil {
        err = fmt.Errorf("failed to write peer_id field: %w", err)
        return
    }
    // Close multipart writer BEFORE making request
    if err = writer.Close(); err != nil {
        err = fmt.Errorf("failed to close multipart writer: %w", err)
        return
    }
    // Build headers with Content-Type (includes boundary)
    headers := make(http.Header)
    headers.Set("Content-Type", writer.FormDataContentType())
    // Make request - pass bytes.Buffer (implements io.Reader)
    // request() detects io.Reader and uses it directly without modification
    // Content-Type header with boundary is preserved
    if _, err = c.request(http.MethodPost, requestURL, headers, bodyBuffer, &result); err != nil {
        err = fmt.Errorf("failed to upload file: %w", err)
        return
    }
    return
}

// ❌ Don't convert multipart buffer to string - loses binary data integrity
if _, err = c.request(http.MethodPost, requestURL, headers, bodyBuffer.String(), &result); err != nil {

// ❌ Don't forget to close the multipart writer before making request
// ❌ Don't forget to set Content-Type header with writer.FormDataContentType()
```

**Key points for multipart forms:**
1. Use `bytes.Buffer` to collect multipart data
2. Use `multipart.NewWriter` to build the form
3. Add all fields (files and regular fields) to the writer
4. **Close the writer** before making the request (finalizes the form)
5. Set `Content-Type` header using `writer.FormDataContentType()` (includes boundary)
6. Pass `bytes.Buffer` directly to `request()` - it implements `io.Reader`
7. **Never** convert to string with `buffer.String()` - binary data will be corrupted
8. **How it works:** `request()` detects `io.Reader` types in the body switch and uses them directly, preserving the Content-Type header with boundary parameter

### Supported Result Types

```go
// ✅ No response body expected
var result any = nil

// ✅ Raw response body
result := new(bytes.Buffer)

// ✅ JSON decoding (default)
result := new(MyStruct)
```

### Extending request()

If you need to support additional input/output cases:

1. **New body type**: Add a case in the body switch statement
2. **New result type**: Add a case in the result switch statement  
3. **New content type handling**: Add decoding logic in the response section
4. **New status code handling**: Add case in the status code switch

```go
// Example: Adding XML support for result
if resp.Header.Get("Content-Type") == "application/xml" {
    if err = xml.NewDecoder(resp.Body).Decode(result); err != nil {
        err = fmt.Errorf("failed to decode XML response: %w", err)
        return
    }
    return
}
```

**Important**: When extending `request()`, ensure backward compatibility and update this documentation.

---

## Implementation Verification

**Before considering a category complete, verify all items in this checklist:**

### Testing Guidelines

**When adding new functionality:**

1. **Unit Tests**: Test validation logic and edge cases
2. **Integration Tests**: Test actual API calls (if test environment available)
3. **Error Cases**: Test all error paths and status codes

**Example test structure:**

```go
func TestCreateWorkspaceRequest_Validate(t *testing.T) {
    tests := []struct {
        name    string
        req     CreateWorkspaceRequest
        wantErr bool
        errMsg  string
    }{
        {
            name:    "valid request",
            req:     CreateWorkspaceRequest{ID: "valid-id"},
            wantErr: false,
        },
        {
            name:    "missing id",
            req:     CreateWorkspaceRequest{ID: ""},
            wantErr: true,
            errMsg:  "id is required",
        },
        {
            name:    "invalid characters",
            req:     CreateWorkspaceRequest{ID: "invalid@id"},
            wantErr: true,
            errMsg:  "must contain only letters, numbers, underscores, or hyphens",
        },
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := tt.req.Validate()
            if (err != nil) != tt.wantErr {
                t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
            }
            if err != nil && tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
                t.Errorf("Validate() error = %v, want error containing %q", err, tt.errMsg)
            }
        })
    }
}
```

### Examples

**Complete method implementation example:**

```go
// GetPeer gets a Peer by ID.
//
// Retrieves a specific peer from the workspace by its unique identifier.
// Returns the peer's current state including metadata and configuration.
//
// https://docs.honcho.dev/v3/api-reference/endpoint/peers/get-peer
func (c *Client) GetPeer(workspaceID, peerID string) (result *Peer, err error) {
    // Validate path parameters
    if workspaceID == "" {
        err = errors.New("workspaceID is required")
        return
    }
    if peerID == "" {
        err = errors.New("peerID is required")
        return
    }
    // Construct URL
    requestURL := c.baseURL.JoinPath(workspaceBaseURI, workspaceID, "peers", peerID)
    // Initialize result
    result = new(Peer)
    // Make request
    if _, err = c.request(http.MethodGet, requestURL, nil, nil, &result); err != nil {
        err = fmt.Errorf("failed to get peer: %w", err)
        return
    }
    return
}
```

---

**Before considering a category complete, verify all items in this checklist:**

### Type Location Check

```bash
# All {category} types should be in {category}_types.go
grep "type.*struct" session_types.go  # Should find Session*, PageSession, etc.
grep "type.*struct" session.go        # Should find nothing

grep "type.*struct" peer_types.go  # Should find Peer*, PagePeer, etc.
grep "type.*struct" peer.go        # Should find nothing
```

**Verify:**
- ✅ All request types are in `{category}_types.go` (e.g., `SessionCreate`, `SessionUpdate`)
- ✅ All response types are in `{category}_types.go` (e.g., `Session`, `PageSession`)
- ✅ All option types are in `{category}_types.go` (e.g., `GetSessionsOptions`)
- ✅ No struct types are defined in `{category}.go` (methods only)
- ✅ Types are not mixed between categories (Session types in session_types.go, not peer_types.go)

### API Compliance Check

**For each endpoint:**
- ✅ HTTP method matches the OpenAPI spec (GET vs POST vs PUT vs DELETE)
- ✅ URL path structure matches the spec exactly
- ✅ Request body schema matches the spec (all fields, types, omitempty)
- ✅ Response body schema matches the spec (all fields, types)
- ✅ All HTTP status codes are handled (200, 201, 202, 204, 422, etc.)
- ✅ Error response schemas are implemented (HTTPValidationError, ValidationError)

**Validation precision:**
- ✅ Numeric constraints (min/max) match the spec exactly
- ✅ String constraints (length, pattern) match the spec exactly
- ✅ Required fields match the `required` array in the spec
- ✅ Default values are documented in struct field comments

### Code Quality Check

**For each method:**
- ✅ API doc URL in block comment (without `.md` extension)
- ✅ Block comment has 2+ paragraphs with descriptive text from OpenAPI spec
- ✅ Named returns used consistently
- ✅ Naked returns used (no explicit return values)
- ✅ Errors wrapped with context using `%w`
- ✅ Validation called ONLY if `Validate()` method exists on request type
- ✅ No calls to `req.Validate()` when all fields are optional
- ✅ Uses `workspaceBaseURI` constant for `/v3/workspaces` paths (or `*BaseURI` for genuinely different base paths like `/v3/keys`)
- ✅ Uses `c.request()` method (not `http.Client.Do()`)
- ✅ Error messages in loops use `fmt.Errorf("item %d: %w", index, err)` (not string concatenation)
- ✅ Multipart forms pass `bytes.Buffer` directly (not `buffer.String()`)
- ✅ Method signature parameter order matches existing patterns (req before options)

**For each struct:**
- ✅ All fields have documentation comments
- ✅ Optional fields have default values documented
- ✅ Constraints documented (min/max, patterns, lengths)
- ✅ Uses `any` not `interface{}`
- ✅ Optional nested structs use pointers with `omitempty`
- ✅ No `Validate()` method if ALL fields are optional (remove it entirely)
- ✅ No `req.Validate()` call if `Validate()` method doesn't exist

### Completeness Check

**Using `https://docs.honcho.dev/llms.txt`:**
- ✅ All endpoints for the category are implemented
- ✅ No endpoints missing from the API index
- ✅ All related types are defined
- ✅ All error types are implemented

---

## Quick Reference

### DO:

**File Organization:**
- ✅ Organize by category (`category.go`/`category_types.go`)
- ✅ Separate types (`*_types.go`) from methods (`*.go`)
- ✅ Verify type locations after implementation
- ✅ Define types in their canonical category file

**Constants & URLs:**
- ✅ Reuse `workspaceBaseURI` constant from `workspace.go` for endpoints starting with `/v3/workspaces`
- ✅ Create a new `*BaseURI` constant only for genuinely different base paths (e.g., `/v3/keys`)
- ✅ Use `JoinPath()` for URL construction (never concatenate strings)

**Method Implementation:**
- ✅ Use named returns: `(result *Type, err error)`
- ✅ Use naked returns (no explicit return values)
- ✅ Validate mandatory parameters with `Validate()` methods
- ✅ Use the low-level `request()` method for all API calls
- ✅ Match parameter order patterns (req before options)

**Error Handling:**
- ✅ Wrap errors with context using `%w`
- ✅ Use `fmt.Errorf("item %d: %w", index, err)` for error messages in loops
- ✅ Check errors immediately and return early

**Documentation:**
- ✅ Include descriptive block comments AND API doc URLs
- ✅ Always use 2+ paragraphs (summary + context from API docs)
- ✅ Document struct field default values and constraints

**Types:**
- ✅ Use `any` instead of `interface{}`
- ✅ Use pointers for optional nested structs
- ✅ Allow 0 for optional int fields where 0 means "use server default"
- ✅ Copy validation constraints exactly from the OpenAPI spec
- ✅ Implement ALL schemas and HTTP status codes from the API docs

**Multipart Forms:**
- ✅ Pass `bytes.Buffer` directly to `request()` (never `buffer.String()`)
- ✅ Close multipart writer before making request
- ✅ Set Content-Type header with `writer.FormDataContentType()`

**Validation:**
- ✅ Omit `Validate()` method entirely if ALL fields are optional
- ✅ Omit `req.Validate()` call when `Validate()` method doesn't exist

### DON'T:

**File Organization:**
- ❌ Mix categories or leave types in the wrong category file
- ❌ Define struct types in `{category}.go` files (types belong in `*_types.go`)

**Constants & URLs:**
- ❌ Define `*BaseURI` constants for paths that start with `/v3/workspaces` (always use `workspaceBaseURI`)
- ❌ Create new `*BaseURI` constants unnecessarily (only for truly different base paths like `/v3/keys`)
- ❌ Hardcode full URLs or concatenate path strings manually

**Method Implementation:**
- ❌ Omit descriptive block comments for methods
- ❌ **Use one-liner block comments** - always include 2+ paragraphs with API description
- ❌ **Omit the second paragraph** even for "obvious" or simple endpoints
- ❌ Call `http.Client.Do()` directly in API methods
- ❌ Return explicit values on naked returns (`return nil, err`)

**Error Handling:**
- ❌ Use string concatenation for error messages in loops
- ❌ Return error types without `Error()` method implementation
- ❌ Forget to wrap errors with `%w` for error chain support

**Documentation:**
- ❌ Leave struct fields undocumented
- ❌ Omit API doc URLs from method comments

**Types:**
- ❌ Use `interface{}` (use `any`)
- ❌ Use value types for optional nested structs (use pointers)
- ❌ Approximate validation constraints (copy exactly from spec)
- ❌ Ignore error schemas or HTTP status codes from the API docs

**Multipart Forms:**
- ❌ Convert multipart `bytes.Buffer` to string with `buffer.String()`
- ❌ Forget to close the multipart writer before making request
- ❌ Forget to set Content-Type header with boundary

**Validation:**
- ❌ Validate optional parameters (server handles those)
- ❌ Add `Validate()` method when ALL fields are optional
- ❌ Call `req.Validate()` when ALL fields are optional (method doesn't exist)
- ❌ Leave `Validate()` calls after removing the `Validate()` method
- ❌ Reject 0 for optional int fields where 0 means "use server default"