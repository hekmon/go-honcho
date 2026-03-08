# Honcho Go SDK - Agent Guidelines

## File Organization

### Category Files
- Each API category gets its own pair of files: `category.go` and `category_types.go`
- Examples: `workspace.go`/`workspace_types.go`, `peer.go`/`peer_types.go`

### Type Definitions
- Place all struct types in `*_types.go` files (e.g., `workspace_types.go`)
- Keep data structures separate from method implementations

### Method Implementations
- Place methods in corresponding `*.go` files (e.g., `workspace.go`)
- One file per resource category (workspace, peer, session, etc.)

### Base URI Constant
- Each category file must define a constant for the base URI
- This constant is shared by all methods in that category
```go
const (
    workspaceBaseURI = "/v3/workspaces"
)
```

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

### Type Declarations
```go
// ✅ Use 'any' instead of 'interface{}'
Metadata map[string]any `json:"metadata,omitempty"`

// ✅ Use pointers for nested structs that can be omitempty
Configuration *WorkspaceConfiguration `json:"configuration,omitempty"`

// ❌ Avoid value types for optional nested structs
Configuration WorkspaceConfiguration `json:"configuration,omitempty"`
```

## URL Construction

### Endpoint Paths
```go
// ✅ Use constants for endpoint paths
const (
    workspaceBaseURI = "/v3/workspaces"
)

// ✅ Use baseURL.JoinPath() for clean URL construction
requestURL := c.baseURL.JoinPath(workspaceBaseURI)

// ❌ Avoid hardcoding full URLs
requestURL, err := url.Parse("https://api.honcho.dev/v3/workspaces")
```

## Validation

### Validate() Methods
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

## Documentation

### API References
```go
// ✅ Include Honcho docs URL with .md extension in block comment
/*
    https://docs.honcho.dev/v3/api-reference/endpoint/workspaces/get-or-create-workspace.md
*/

// GetOrCreateWorkspace gets a Workspace by ID or creates a new one.
func (c *Client) GetOrCreateWorkspace(req CreateWorkspaceRequest) (result *Workspace, err error) {
```

### Documentation Trick
- Honcho docs support `.md` extension for direct markdown access
- Example: `https://docs.honcho.dev/v3/api-reference/endpoint/workspaces/get-or-create-workspace.md`
- Use this for clean API documentation scraping

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

## Constants

### Pattern Definitions
```go
// ✅ Define regex patterns as package-level variables
var workspaceIDPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// ✅ Use constants for endpoint paths
const (
    workspaceBaseURI = "/v3/workspaces"
)
```

## Imports

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

## Summary

**DO:**
- ✅ Organize by category (`category.go`/`category_types.go`)
- ✅ Define base URI constant per category (e.g., `workspaceBaseURI`)
- ✅ Separate types (`*_types.go`) from methods (`*.go`)
- ✅ Use named returns and naked returns
- ✅ Validate mandatory parameters with `Validate()` methods
- ✅ Use `any` instead of `interface{}`
- ✅ Use pointers for optional nested structs
- ✅ Use `baseURL.JoinPath()` for URL construction
- ✅ Wrap errors with context using `%w`
- ✅ Include `.md` API doc URLs in block comments
- ✅ Use the low-level `request()` method for all API calls
- ✅ Extend `request()` when new body/result types are needed

**DON'T:**
- ❌ Mix categories in the same file
- ❌ Omit base URI constant per category
- ❌ Omit `.md` documentation links for methods
- ❌ Mix types and methods in the same file
- ❌ Validate optional parameters (server handles those)
- ❌ Hardcode full URLs
- ❌ Use `interface{}` (use `any`)
- ❌ Use value types for optional nested structs
- ❌ Call `http.Client.Do()` directly in API methods
- ❌ Duplicate request/response handling logic
- ❌ Return explicit values on naked returns