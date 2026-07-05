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
}

type PhotoSearchResponse struct {
    Result    PhotosSearchResult `body:"json"`
    RateLimit int                `header:"X-Rate-Limit"`
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
    req.Request.Authorization   // (header:"authorization")
    req.Request.AlbumID         // (path:"albumID")
    req.Request.IncludeMetadata // (query:"include_metadata")
    req.Request.PageSize        // (query:"page_size")
    req.Request.Page            // (query:"page")

    // Access raw HTTP types if needed
    req.Http.R  // *http.Request
    req.Http.W  // http.ResponseWriter

    // Response automatically serialized and written
    return &gobind.Response[PhotoSearchResponse]{
        StatusCode: http.StatusOK,
        Response: PhotoSearchResponse{
            RateLimit: 60,
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