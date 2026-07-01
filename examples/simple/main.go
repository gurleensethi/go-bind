package main

import (
	"context"
	"fmt"
	"net/http"

	gobind "github.com/gurleensethi/go-bind"
)

func main() {
	mux := http.NewServeMux()

	mux.Handle("/photos/{albumID}/search", gobind.Handler(PhotoAlbumSearch))

	http.ListenAndServe(":9876", mux)
}

type PhotoFilters struct {
	Message string `json:"message"`
}

type PhotoAlbumSearchRequest struct {
	Authorization        string        `header:"authorization"`
	SearchQuery          *string       `query:"q"`
	SearchQueryNotExists *string       `query:"not_exists"`
	AlbumID              string        `path:"albumID"`
	PhotoFilters1        string        `body:"text"`
	PhotoFilters2        *PhotoFilters `body:"json"`
	PageSize             int8          `query:"page_size"`
	PageSizeFloat        float32       `query:"page_size"`
	IncludeMetadata      *bool         `query:"include_metadata"`
}

type PhotoAlbumSearchResponse struct {
}

func PhotoAlbumSearch(ctx context.Context, req *gobind.Request[PhotoAlbumSearchRequest]) (*gobind.Response[PhotoAlbumSearchResponse], error) {
	fmt.Printf("%+v\n", req.Request)

	return &gobind.Response[PhotoAlbumSearchResponse]{}, nil
}
