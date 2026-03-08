# Honcho Go SDK - Agent Guidelines

This document provides comprehensive guidelines for implementing the Honcho Go SDK. Follow these rules to ensure consistency, completeness, and maintainability.

## Table of Contents

1. [File Organization](#file-organization)
2. [Documentation & API Discovery](#documentation--api-discovery)
3. [Implementation Pattern](#implementation-pattern)
4. [Coding Style](#coding-style)
5. [Validation](#validation)
6. [URL Construction](#url-construction)
7. [Low-Level Request Method](#low-level-request-method)
8. [Constants & Imports](#constants--imports)
9. [Implementation Verification](#implementation-verification)
10. [Summary](#summary)

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

### Base URI Constant

**All API endpoints share the same base URI `/v3/workspaces`. Use the `workspaceBaseURI` constant from `workspace.go` in all category files:**

**Note:** If you encounter an endpoint that does NOT start with `/v3/workspaces`, create a new constant for that specific base path.

```go
// ✅ In peer.go, session.go, etc. - reuse workspaceBaseURI
requestURL := c.baseURL.JoinPath(workspaceBaseURI, workspaceID, "peers")

// ❌ Don't define a new constant per category
const (
    peerBaseURI = "/v3/workspaces"  // redundant!
)
```

---

## Documentation & API Discovery

### API Index

**Start by discovering all available endpoints:**

- **Complete API Index**: Fetch `https://docs.honcho.dev/llms.txt` to discover ALL available endpoints
- Use the index to find all endpoints for a category and ensure complete implementation

### API References

**Include Honcho docs URL in block comments (WITHOUT `.md` extension for IDE visibility):**

```go
// https://docs.honcho.dev/v3/api-reference/endpoint/workspaces/get-or-create-workspace
func (c *Client) GetOrCreateWorkspace(req CreateWorkspaceRequest) (result *Workspace, err error) {
```

**Note for Agents:** When fetching documentation, append `.md` to the URL:
- Agent fetch: `https://docs.honcho.dev/v3/api-reference/endpoint/workspaces/get-or-create-workspace.md`
- IDE comment: `https://docs.honcho.dev/v3/api-reference/endpoint/workspaces/get-or-create-workspace`

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

### Standard Method Structure

```go
func (c *Client) MethodName(req RequestType) (result *ResultType, err error) {
    // 1. Validate mandatory parameters
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
err = fmt.Errorf("failed to parse URL: %s", err)
err = fmt.Errorf("failed to execute request: %w", err)

// ✅ Check errors immediately and return early
if err != nil {
    return
}
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
```

### Call Validation in Methods

```go
// ✅ Call Validate() at the start of methods
func (c *Client) GetOrCreateWorkspace(req CreateWorkspaceRequest) (result *Workspace, err error) {
    if err = req.Validate(); err != nil {
        return
    }
    // ... rest of implementation
}
```

### Validation Constraint Precision

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

This is critical for optional parameters where `0` means "use server default".

---

## URL Construction

### Endpoint Paths

```go
// ✅ Use workspaceBaseURI constant (defined in workspace.go) for all categories
// All endpoints follow the pattern: /v3/workspaces/{workspace_id}/{category}/...
requestURL := c.baseURL.JoinPath(workspaceBaseURI, workspaceID, "peers")

// ✅ If an endpoint doesn't start with /v3/workspaces, define a new constant
const specialBaseURI = "/v3/special-endpoint"

// ✅ Build category-specific paths by appending to workspaceBaseURI
const (
    workspaceBaseURI = "/v3/workspaces"  // defined once in workspace.go
)

// ❌ Avoid hardcoding full URLs
requestURL, err := url.Parse("https://api.honcho.dev/v3/workspaces")

// ❌ Don't define redundant base URI constants per category
const (
    peerBaseURI = "/v3/workspaces"  // wrong! reuse workspaceBaseURI
)

// ❌ Don't create new constants unnecessarily - only for truly different base paths
const (
    peerBaseURI = "/v3/workspaces/peers"  // wrong! this is just a path extension
)
```

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
```

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

## Constants & Imports

### Pattern Definitions

```go
// ✅ Define regex patterns as package-level variables
var workspaceIDPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// ✅ Use constants for endpoint paths
const (
    workspaceBaseURI = "/v3/workspaces"
)
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
- ✅ Named returns used consistently
- ✅ Naked returns used (no explicit return values)
- ✅ Errors wrapped with context using `%w`
- ✅ Validation called at method start (if request has Validate())
- ✅ Uses `workspaceBaseURI` constant (not hardcoded URLs)
- ✅ Uses `c.request()` method (not `http.Client.Do()`)

**For each struct:**
- ✅ All fields have documentation comments
- ✅ Optional fields have default values documented
- ✅ Constraints documented (min/max, patterns, lengths)
- ✅ Uses `any` not `interface{}`
- ✅ Optional nested structs use pointers with `omitempty`

### Completeness Check

**Using `https://docs.honcho.dev/llms.txt`:**
- ✅ All endpoints for the category are implemented
- ✅ No endpoints missing from the API index
- ✅ All related types are defined
- ✅ All error types are implemented

---

## Summary

### DO:

- ✅ Organize by category (`category.go`/`category_types.go`)
- ✅ Reuse `workspaceBaseURI` constant from `workspace.go` (all endpoints start with `/v3/workspaces`)
- ✅ Create a new base URI constant only for endpoints that don't start with `/v3/workspaces`
- ✅ Separate types (`*_types.go`) from methods (`*.go`)
- ✅ Verify type locations after implementation (session types in session_types.go, etc.)
- ✅ Use named returns and naked returns
- ✅ Validate mandatory parameters with `Validate()` methods
- ✅ Copy validation constraints exactly from the OpenAPI spec (don't approximate)
- ✅ Use `any` instead of `interface{}`
- ✅ Use pointers for optional nested structs
- ✅ Use `baseURL.JoinPath()` for URL construction
- ✅ Wrap errors with context using `%w`
- ✅ Include API doc URLs in block comments (without `.md` - for IDE visibility)
- ✅ Agents should append `.md` when fetching documentation
- ✅ Use the low-level `request()` method for all API calls
- ✅ Extend `request()` when new body/result types are needed
- ✅ Document struct field default values and constraints
- ✅ Implement ALL schemas from the API docs (request, response, errors)
- ✅ Implement ALL HTTP status codes from the API docs
- ✅ Give error types an `Error()` method and wrap with `%w`
- ✅ Use `https://docs.honcho.dev/llms.txt` to discover all endpoints
- ✅ Run the implementation verification checklist before considering work complete

### DON'T:

- ❌ Mix categories in the same file
- ❌ Leave types in the wrong category file (even if related)
- ❌ Define redundant base URI constants (reuse `workspaceBaseURI` from `workspace.go`)
- ❌ Omit documentation links for methods
- ❌ Mix types and methods in the same file
- ❌ Validate optional parameters (server handles those)
- ❌ Approximate validation constraints (copy exact min/max from spec)
- ❌ Hardcode full URLs
- ❌ Use `interface{}` (use `any`)
- ❌ Use value types for optional nested structs
- ❌ Call `http.Client.Do()` directly in API methods
- ❌ Duplicate request/response handling logic
- ❌ Return explicit values on naked returns
- ❌ Leave struct fields undocumented (especially optional params)
- ❌ Ignore error schemas or HTTP status codes from the API docs
- ❌ Return error types without `Error()` method
- ❌ Consider implementation complete without running the verification checklist