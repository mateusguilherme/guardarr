// Package server wires guardarr's HTTP routes to their handlers.
package server

import (
	"encoding/json"
	"io/fs"
	"log"
	"net/http"

	"github.com/mateusguilherme/guardarr/internal/musicbrainz"
	"github.com/mateusguilherme/guardarr/internal/store"
)

// New builds the application's HTTP handler with every route registered.
// static holds the compiled frontend, served for everything outside /api.
func New(mb *musicbrainz.Client, st *store.Store, static fs.FS) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /api/artists", searchArtists(mb))
	mux.Handle("GET /api/followed", listFollowed(st))
	mux.Handle("PUT /api/followed/{id}", followArtist(st))
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

func listFollowed(st *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		artists, err := st.Artists(r.Context())

		if err != nil {
			log.Printf("list followed artists: %v", err)
			http.Error(w, "listing followed artists failed", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")

		if err := json.NewEncoder(w).Encode(artists); err != nil {
			log.Printf("encode response: %v", err)
		}
	}
}

func followArtist(st *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Name           string `json:"name"`
			Disambiguation string `json:"disambiguation"`
		}

		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "invalid JSON body", http.StatusBadRequest)
			return
		}

		if body.Name == "" {
			http.Error(w, `missing "name" field`, http.StatusBadRequest)
			return
		}

		artist := store.Artist{
			ID:             r.PathValue("id"),
			Name:           body.Name,
			Disambiguation: body.Disambiguation,
		}

		if err := st.SaveArtist(r.Context(), artist); err != nil {
			log.Printf("follow artist: %v", err)
			http.Error(w, "following the artist failed", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
