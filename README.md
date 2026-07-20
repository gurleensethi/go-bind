# go-bind

Skip the request parsing and response serialization boilerplate in Go using struct tags.

# Full example

```go
import (
    "context"
    "net/http"

    gobind "github.com/gurleensethi/go-bind"
)

type PhotoSearchRequest struct {
    Authorization   string `header:"authorization"`
    AlbumID         string `path:"albumID"`
    IncludeMetadata bool   `query:"include_metadata"`
    PageSize        int    `query:"page_size"`
    Page            int    `query:"page"`
    SessionID       string `cookie:"session_id"`
}

type PhotoSearchResponse struct {
    Result    PhotosSearchResult `body:"json"`
    RateLimit int                `header:"X-Rate-Limit"`
    Token     string             `cookie:"token"`
}

type PhotosSearchResult struct {
    Photos     []Photo `json:"photos"`
    TotalCount int     `json:"total_count"`
}

type Photo struct {
    Name string `json:"name"`
}

func PhotoSearchHandler(ctx context.Context, req *gobind.Request[PhotoSearchRequest]) (*gobind.Response[PhotoSearchResponse], error) {
    // Request fields pre-filled from incoming HTTP request:
    req.Value.Authorization   // (header:"authorization")
    req.Value.AlbumID         // (path:"albumID")
    req.Value.IncludeMetadata // (query:"include_metadata")
    req.Value.PageSize        // (query:"page_size")
    req.Value.Page            // (query:"page")
    req.Value.SessionID       // (cookie:"session_id")

    // Access raw HTTP types if needed
    req.Http.R // *http.Request
    req.Http.W // http.ResponseWriter

    // Response automatically serialized and written
    return &gobind.Response[PhotoSearchResponse]{
        StatusCode: http.StatusOK,
        Value: PhotoSearchResponse{
            RateLimit: 60,
            Token:     "abc-123",
            Result: PhotosSearchResult{
                Photos:     []Photo{{Name: "photo1.jpg"}},
                TotalCount: 10,
            },
        },
    }, nil
}

func main() {
    mux := http.NewServeMux()
    mux.Handle("/albums/{albumID}/search", gobind.Handler(PhotoSearchHandler))
    http.ListenAndServe(":9876", mux)
}
```

## Supported tags

| Tag                     | Direction        | Example                                     |
| ----------------------- | ---------------- | ------------------------------------------- |
| `header:"name"`         | Request/Response | ``Auth string `header:"authorization"` ``   |
| `query:"name"`          | Request          | ``Page int `query:"page"` ``                |
| `path:"name"`           | Request          | ``ID string `path:"id"` ``                  |
| `body:"json" \| "text"` | Request/Response | ``Data MyStruct `body:"json"` ``            |
| `cookie:"name"`         | Request/Response | ``SessionID string `cookie:"session_id"` `` |

## Error handling

Return typed errors with struct tag serialization (headers, body, cookies, status code):

```go
type ApiErrorBody struct {
    Message string         `json:"message"`
    Details map[string]any `json:"details"`
}

type ApiError struct {
    RetryAfter int          `header:"Retry-After"`
    Body       ApiErrorBody `body:"json"`
}

func PhotoSearchHandler(ctx context.Context, req *gobind.Request[PhotoSearchRequest]) (*gobind.Response[PhotoSearchResponse], error) {
    if req.Value.PageSize > 10 {
        return nil, gobind.NewError(http.StatusBadRequest, ApiError{
            RetryAfter: 10,
            Body: ApiErrorBody{
                Message: "invalid page_size",
                Details: map[string]any{"page_size": "max 10"},
            },
        })
    }

    // ...
}
```

The error body struct supports the same tags as responses (`header`, `body`, `cookie`). Status code is set via the constructor. Non-`gobind.Error` errors fall back to a generic 500 response.

## Error helpers

Predefined constructors for common HTTP error status codes:

```go
// 400 Bad Request
return nil, gobind.BadRequestError(ApiError{...})

// 401 Unauthorized
return nil, gobind.UnauthorizedError(ApiError{...})

// 403 Forbidden
return nil, gobind.ForbiddenError(ApiError{...})

// 404 Not Found
return nil, gobind.NotFoundError(ApiError{...})

// 409 Conflict
return nil, gobind.ConflictError(ApiError{...})

// 422 Unprocessable Entity
return nil, gobind.UnprocessableEntityError(ApiError{...})

// 429 Too Many Requests
return nil, gobind.TooManyRequestsError(ApiError{...})

// 500 Internal Server Error
return nil, gobind.InternalServerError(ApiError{...})

// 502 Bad Gateway
return nil, gobind.BadGatewayError(ApiError{...})

// 503 Service Unavailable
return nil, gobind.ServiceUnavailableError(ApiError{...})

// 504 Gateway Timeout
return nil, gobind.GatewayTimeoutError(ApiError{...})
```

All helpers accept a typed error body and return `*gobind.Error[T]` for structured error responses.