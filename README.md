# go-bind

Skip the request paring and response serilaization dance in Golang using struct tags.

```go
import "github.com/gurleensethi/go-bind"

type PhotoSearchRequest struct {
    Authorization   string `header:"authorization"`    
    AlbumID         string `path:"albumID"`
    IncludeMetadata bool   `query:"include_metadata"`
    PageSize        int    `query:"page_size"`
    Page            int    `query:"page"`
}

type PhotoSearchResponse struct {
    Result    PhotosSearchResult `body:"json"`
    RateLimit string             `header:"X-Rate-Limit"`
}

type PhotosSearchResult struct {
    Photos     []Photo `json:"photos"`
    TotalCount int     `json:"total_count"`
}

type Photo struct {
    Name string `json:"name"`
}

func PhotoSearchHandler(ctx context.Context, req *gobind.Request[PhotoSearchRequest]) *gobind.Response[PhotoSearchResponse] {
    // Access request fields pre-filled with information from incomding request
    req.Request.Authorization
    req.Request.AlbumID
    req.Request.IncludeMetadata
    req.Request.PageSize
    req.Request.Page

    // Access to http request and response
    req.Http.R  // *http.Request
    req.Http.W  // http.ResponseWriter

    // Response is autoamtically serialized and sent back.
    return &gobind.Response[PhotoSearchResponse]{
        Response: PhotoSearchResponse{
            RateLimit: 60,
            Result: PhotoSearchResult{
                Photos: []Photo{},
                TotalCount: 10,
            }
        }
    }, nil
}

func main() {
    mux := http.NewServeMux()

    // Wire up using gobind.Handler
	mux.Handle("/albums/{albumID}/search", gobind.Handler(PhotoSearchHandler))

	http.ListenAndServe(":9876", mux)
}
```