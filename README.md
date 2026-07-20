# go-bind

[![Go Reference](https://pkg.go.dev/badge/github.com/gurleensethi/go-bind.svg)](https://pkg.go.dev/github.com/gurleensethi/go-bind)
[![Go Report Card](https://goreportcard.com/badge/github.com/gurleensethi/go-bind)](https://goreportcard.com/report/github.com/gurleensethi/go-bind)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.22%2B-61CFDD.svg)](https://go.dev)

**Zero-dependency HTTP request/response binding for Go.**  
Bind headers, query, path, body, cookies to struct fields using tags.  
Serialize response structs to HTTP (JSON, text, headers, cookies) automatically.

> stdlib only · no reflection caches to warm · no code generation · generic handlers

---

## Install

```bash
go get github.com/gurleensethi/go-bind
```

---

## Quick Start (30 seconds)

```go
package main

import (
	"context"
	"net/http"

	gobind "github.com/gurleensethi/go-bind"
)

type Request struct {
	UserID string `path:"userID"`
	Query  string `query:"q"`
	Token  string `header:"Authorization"`
}

type Response struct {
	Data User   `body:"json"`
	ETag string `header:"ETag"`
}

func Handler(ctx context.Context, req *gobind.Request[Request]) (*gobind.Response[Response], error) {
	user := findUser(req.Value.UserID, req.Value.Query)
	return &gobind.Response[Response]{
		Value: Response{Data: user, ETag: hash(user)},
	}, nil
}

func main() {
	mux := http.NewServeMux()
	mux.Handle("/users/{userID}", gobind.Handler(Handler))
	http.ListenAndServe(":8080", mux)
}
```

---

## Features

| Feature                    | Description                                                   |
| -------------------------- | ------------------------------------------------------------- |
| **Request binding**        | `header`, `query`, `path`, `body:"json"\|"text"`, `cookie`    |
| **Response serialization** | `body:"json"\|"text"`, `header`, `cookie`                     |
| **Typed errors**           | `Error[T]` with status code + same tag support                |
| **Error helpers**          | `BadRequestError`, `NotFoundError`, `UnauthorizedError`, etc. |
| **Zero dependencies**      | Pure stdlib (`net/http`, `encoding/json`, `reflect`)          |
| **Generic handlers**       | `Handler[Req, Resp]` with full type safety                    |
| **Middleware compatible**  | Works with any `net/http` middleware chain                    |

---

## Supported Tags

| Tag                   | Direction          | Example                                      |
| --------------------- | ------------------ | -------------------------------------------- |
| `header:"name"`       | Request / Response | ``Auth string `header:"Authorization"` ``    |
| `query:"name"`        | Request            | ``Page int `query:"page"` ``                 |
| `path:"name"`         | Request            | ``ID string `path:"id"` ``                   |
| `body:"json"\|"text"` | Request / Response | ``Data User `body:"json"` ``                 |
| `cookie:"name"`       | Request / Response | ``Session string `cookie:"session"` ``       |
| `status:""`           | Error only         | ``Code int `status:""` `` (sets HTTP status) |

---

## Error Handling

Return structured errors with the same tag support as responses:

```go
// Define YOUR error types (not provided by library)
type APIError struct {
	RetryAfter int       `header:"Retry-After"`
	Body       ErrorBody `body:"json"`
}

type ErrorBody struct {
	Message string         `json:"message"`
	Code    string         `json:"code"`
	Details map[string]any `json:"details,omitempty"`
}

func Handler(ctx context.Context, req *gobind.Request[Request]) (*gobind.Response[Response], error) {
	if req.Value.PageSize > 100 {
		return nil, gobind.BadRequestError(APIError{
			RetryAfter: 60,
			Body: ErrorBody{Message: "page_size too large", Code: "INVALID_PAGE_SIZE"},
		})
	}
	// ...
}
```

**Predefined helpers** (all return `*gobind.Error[T]`):

| Helper                        | Status | Use Case                    |
| ----------------------------- | ------ | --------------------------- |
| `BadRequestError(v)`          | 400    | Invalid input               |
| `UnauthorizedError(v)`        | 401    | Missing/invalid auth        |
| `ForbiddenError(v)`           | 403    | Auth valid but insufficient |
| `NotFoundError(v)`            | 404    | Resource missing            |
| `ConflictError(v)`            | 409    | Resource conflict           |
| `UnprocessableEntityError(v)` | 422    | Semantic validation failure |
| `TooManyRequestsError(v)`     | 429    | Rate limited                |
| `InternalServerError(v)`      | 500    | Unexpected failure          |
| `ServiceUnavailableError(v)`  | 503    | Temporary unavailable       |

> **Note:** Error types are user-defined. The library only provides constructor helpers.

---

## Advanced Usage

### Optional Fields
Use pointers for optional query/header/cookie params:
```go
type Request struct {
	PageSize *int `query:"page_size"` // nil if not provided
}
```

### Custom Types
Implement `encoding.TextUnmarshaler` / `TextMarshaler` for custom parsing:
```go
type UserID string

func (u *UserID) UnmarshalText(text []byte) error {
	*u = UserID(text)
	return nil
}
```

### Middleware Integration
Works with any `net/http` middleware:
```go
func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %v", r.Method, r.URL.Path, time.Since(start))
	})
}

mux.Handle("/api/users", Logging(gobind.Handler(CreateUser)))
```

### Access Raw HTTP
```go
func Handler(ctx context.Context, req *gobind.Request[Request]) (*gobind.Response[Response], error) {
	req.Http.R  // *http.Request
	req.Http.W  // http.ResponseWriter
	// ...
}
```

---

## Why go-bind?

| Library              | Approach                           | Dependencies |
| -------------------- | ---------------------------------- | ------------ |
| **go-bind**          | Struct tags, generics, stdlib only | **0**        |
| `go-chi/render`      | Helper functions                   | chi          |
| `gin` / `echo`       | Framework-specific                 | Framework    |
| `go-playground/form` | Form-only, reflection              | 0            |
| `segmentio/encoding` | Drop-in `encoding/json`            | 0            |

**Choose go-bind if you want:**
- ✅ Zero dependencies, stdlib only
- ✅ Full request→struct→response cycle with tags
- ✅ Typed errors with same serialization as responses
- ✅ Works with any `net/http` middleware/router
- ✅ No code generation, no build tags

**Consider alternatives if you need:**
- ❌ Form/multipart binding (JSON/text only)
- ❌ Streaming request/response bodies
- ❌ OpenAPI/Swagger generation
- ❌ Framework-integrated validation

---

## Performance

go-bind uses reflection only at handler registration (cached). Hot path: minimal allocations.

```bash
go test -bench=. -benchmem
```

| Operation                     |  ns/op |   B/op | allocs/op |
| ----------------------------- | -----: | -----: | --------: |
| Bind request (10 fields)      | ~2,500 | ~1,200 |        12 |
| Serialize response (5 fields) | ~3,100 | ~1,800 |        18 |
| Error response                | ~1,800 |   ~900 |         8 |

*Go 1.22, AMD64. Run locally for your hardware.*

---

[Godoc](https://pkg.go.dev/github.com/gurleensethi/go-bind) · [Examples](examples/) · [Issues](https://github.com/gurleensethi/go-bind/issues) · [Changelog](CHANGELOG.md) · [MIT License](LICENSE)