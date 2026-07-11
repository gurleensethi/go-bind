package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	gobind "github.com/gurleensethi/go-bind"
)

func main() {
	mux := http.NewServeMux()

	mux.Handle("/photos/{albumID}/search", gobind.Handler(PhotoAlbumSearch))

	slog.Info("running server on 9876")
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

type Album struct {
	Name      string `json:"name"`
	NumPhotos int    `json:"numPhotos"`
}

type PhotoAlbumSearchResponse struct {
	CacheHit int          `header:"X-Cache-Hits"`
	Albums   []Album      `body:"json"`
	Session  *http.Cookie `cookie:"session"`
	Token    string       `cookie:"token"`
}

func PhotoAlbumSearch(ctx context.Context, req *gobind.Request[PhotoAlbumSearchRequest]) (*gobind.Response[PhotoAlbumSearchResponse], error) {
	fmt.Printf("%+v\n", req.Request)

	return &gobind.Response[PhotoAlbumSearchResponse]{
		Response: PhotoAlbumSearchResponse{
			CacheHit: 1,
			Albums: []Album{
				{
					Name:      "demo album",
					NumPhotos: 10,
				},
			},
			Session: &http.Cookie{
				Name:  "session_new",
				Value: "123",
				Path:  "/",
			},
			Token: "token-123",
		},
	}, nil
}
