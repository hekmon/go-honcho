# Honcho Go SDK - Agent Guidelines

This document provides comprehensive guidelines for implementing the Honcho Go SDK. Follow these rules to ensure consistency, completeness, and maintainability.

## Table of Contents

1. [File Organization](#file-organization)
2. [Constants & Shared Resources](#constants--shared-resources)
3. [Documentation & API Discovery](#documentation--api-discovery)
4. [Implementation Pattern](#implementation-pattern)
5. [Coding Style](#coding-style)
6. [Validation](#validation)
7. [Low-Level Request Method](#low-level-request-method)
8. [Implementation Verification](#implementation-verification)
9. [Quick Reference](#quick-reference)

---

## File Organization

### Category Files

Each API category gets its own pair of files:

- `{category}.go` - Method implementations
- `{category}_types.go` - Type definitions

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

```go
// ✅ Pointer distinguishes nil vs 0
type GetPeerContextOptions struct {
    SearchTopK *int `json:"search_top_k,omitempty"`  // nil = not provided, 0 = explicitly zero
}
```

**Use value type (`int`, `bool`) with `omitempty` when:**
- Zero value (0, false) should mean "use server default"
- No need to distinguish between nil and zero

```go
// ✅ Value type with omitempty: 0 means "use default"
type ConclusionQuery struct {
    // TopK is the number of results to return (0=use default: 10, min: 1, max: 100)
    TopK int `json:"top_k,omitempty"`  // 0 omitted from JSON, server uses default (10)
}
```

```go
// ❌ Don't use pointer when 0 should mean "use default"
TopK *int `json:"top_k,omitempty"`  // forces caller to use &value instead of just value

// ❌ Don't use value type when you need to distinguish nil from 0
SearchTopK int `json:"search_top_k,omitempty"`  // can't tell if 0 means "not set" or "set to 0"
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

// ❌ Don't create constants for path extensions
const (
    peerBaseURI = "/v3/workspaces/peers"  // WRONG! This is workspaceBaseURI + "/peers"
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
- [ ] Uses `workspaceBaseURI` constant
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

### Type Declarations

```go
// ✅ Use 'any' instead of 'interface{}'
Metadata map[string]any `json:"metadata,omitempty"`

// ✅ Use pointers for nested structs that can be omitempty
Configuration *WorkspaceConfiguration `json:"configuration,omitempty"`

// ❌ Avoid value types for optional nested structs
Configuration WorkspaceConfiguration `json:"configuration,omitempty"`
```

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
      └─ Remove Validate() method entirely
```

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

**Add `Validate()` for mandatory parameters only:**

```go
// ✅ Add Validate() for mandatory parameters only
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

// ❌ Don't validate optional parameters (let server handle those)

// ❌ Don't add Validate() if ALL fields are optional - remove the method entirely
type MessageUpdate struct {
    Metadata map[string]any `json:"metadata,omitempty"`  // only field, and it's optional
}
// No Validate() method needed - caller can pass empty struct
```

**Rule:** If a request type has ONLY optional fields (all fields have `omitempty`), do NOT add a `Validate()` method. The method should be omitted entirely, not return `nil`.

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

// ✅ Document pointer defaults clearly
type ListConclusionsOptions struct {
    // Reverse is whether to reverse the order of results (default: false)
    Reverse *bool  // nil = use server default (false)
    
    // Page is the page number (default: 1, minimum: 1)
    Page int  // 0 = use server default (1)
    
    // Size is the page size (default: 50, minimum: 1, maximum: 100)
    Size int  // 0 = use server default (50)
}

// ❌ Avoid ambiguous documentation
// TopK is the number of results to return (min: 1, max: 100)  // doesn't mention 0!
```

This is critical for optional parameters where `0` means "use server default".

---

## Low-Level Request Method

### Usage

- All API methods MUST use the `request()` method from `request.go`
- This method handles HTTP request building, execution, and response parsing
- Do NOT call `http.Client.Do()` directly in API methods

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

- ✅ Organize by category (`category.go`/`category_types.go`)
- ✅ Separate types (`*_types.go`) from methods (`*.go`)
- ✅ Verify type locations after implementation
- ✅ Define types in their canonical category file
- ✅ Reuse `workspaceBaseURI` constant from `workspace.go` for endpoints starting with `/v3/workspaces`
- ✅ Create a new `*BaseURI` constant only for genuinely different base paths (e.g., `/v3/keys`)
- ✅ Use named returns and naked returns
- ✅ Validate mandatory parameters with `Validate()` methods
- ✅ Copy validation constraints exactly from the OpenAPI spec
- ✅ Allow 0 for optional int fields where 0 means "use server default"
- ✅ Use `any` instead of `interface{}`
- ✅ Use pointers for optional nested structs
- ✅ Wrap errors with context using `%w`
- ✅ Include descriptive block comments AND API doc URLs
- ✅ Use the low-level `request()` method for all API calls
- ✅ Document struct field default values and constraints
- ✅ Implement ALL schemas and HTTP status codes from the API docs
- ✅ Use `fmt.Errorf("item %d: %w", index, err)` for error messages in loops
- ✅ Pass `bytes.Buffer` directly to `request()` for multipart forms
- ✅ Omit `Validate()` method entirely if ALL fields are optional
- ✅ Omit `req.Validate()` call when `Validate()` method doesn't exist

### DON'T:

- ❌ Mix categories or leave types in the wrong category file
- ❌ Define `*BaseURI` constants for paths that start with `/v3/workspaces` (always use `workspaceBaseURI`)
- ❌ Create new `*BaseURI` constants unnecessarily (only for truly different base paths like `/v3/keys`)
- ❌ Omit descriptive block comments for methods
- ❌ **Use one-liner block comments** - always include 2+ paragraphs with API description
- ❌ **Omit the second paragraph** even for "obvious" or simple endpoints
- ❌ Validate optional parameters (server handles those)
- ❌ Approximate validation constraints
- ❌ Hardcode full URLs
- ❌ Use `interface{}` (use `any`)
- ❌ Call `http.Client.Do()` directly in API methods
- ❌ Return explicit values on naked returns
- ❌ Leave struct fields undocumented
- ❌ Ignore error schemas or HTTP status codes from the API docs
- ❌ Use string concatenation for error messages in loops
- ❌ Convert multipart `bytes.Buffer` to string with `buffer.String()`
- ❌ Add `Validate()` method when ALL fields are optional
- ❌ Call `req.Validate()` when ALL fields are optional (method doesn't exist)
- ❌ Leave `Validate()` calls after removing the `Validate()` method
