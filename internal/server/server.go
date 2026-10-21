// Package server wires guardarr's HTTP routes to their handlers.
package server

import (
	"encoding/json"
	"io/fs"
	"log"
	"net/http"

	"github.com/mateusguilherme/guardarr/internal/musicbrainz"
)

// New builds the application's HTTP handler with every route registered.
// static holds the compiled frontend, served for everything outside /api.
func New(mb *musicbrainz.Client, static fs.FS) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /api/artists", searchArtists(mb))
	mux.Handle("GET /", http.FileServerFS(static))

	return mux
}

func searchArtists(mb *musicbrainz.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		term := r.URL.Query().Get("q")

		if term == "" {
			http.Error(w, `missing "q" parameter`, http.StatusBadRequest)
			return
		}

		artists, err := mb.SearchArtists(r.Context(), term)

		if err != nil {
			log.Printf("musicbrainz search %q: %v", term, err)
			http.Error(w, "musicbrainz search failed", http.StatusBadGateway)
			return
		}

		w.Header().Set("Content-Type", "application/json")

		if err := json.NewEncoder(w).Encode(artists); err != nil {
			log.Printf("encode response: %v", err)
		}
	}
}
