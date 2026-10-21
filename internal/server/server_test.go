package server_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/mateusguilherme/guardarr/internal/musicbrainz"
	"github.com/mateusguilherme/guardarr/internal/server"
)

// newApp serves the application against a fake MusicBrainz handler and an
// in-memory frontend, both cleaned up with the test.
func newApp(t *testing.T, fakeMusicBrainz http.HandlerFunc) *httptest.Server {
	t.Helper()

	fake := httptest.NewServer(fakeMusicBrainz)
	t.Cleanup(fake.Close)

	static := fstest.MapFS{
		"index.html": &fstest.MapFile{Data: []byte("<!doctype html><title>guardarr</title>")},
	}

	app := httptest.NewServer(server.New(musicbrainz.NewClientWithURL(fake.URL), static))
	t.Cleanup(app.Close)

	return app
}

func get(t *testing.T, url string) *http.Response {
	t.Helper()

	resp, err := http.Get(url)

	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { resp.Body.Close() })

	return resp
}

func TestSearchArtistsReturnsResults(t *testing.T) {
	app := newApp(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(`{"artists": [{"id": "abc", "name": "Caetano Veloso"}]}`))
	})

	resp := get(t, app.URL+"/api/artists?q=caetano")

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	if got := resp.Header.Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q, want %q", got, "application/json")
	}

	var artists []musicbrainz.Artist

	if err := json.NewDecoder(resp.Body).Decode(&artists); err != nil {
		t.Fatal(err)
	}

	if len(artists) != 1 || artists[0].Name != "Caetano Veloso" {
		t.Errorf("unexpected artists: %+v", artists)
	}
}

func TestSearchArtistsRequiresQuery(t *testing.T) {
	app := newApp(t, func(_ http.ResponseWriter, _ *http.Request) {
		t.Error("MusicBrainz should not be called without a q parameter")
	})

	resp := get(t, app.URL+"/api/artists")

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestSearchArtistsReportsUpstreamFailure(t *testing.T) {
	app := newApp(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	resp := get(t, app.URL+"/api/artists?q=caetano")

	if resp.StatusCode != http.StatusBadGateway {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadGateway)
	}
}

func TestServesTheFrontend(t *testing.T) {
	app := newApp(t, func(_ http.ResponseWriter, _ *http.Request) {
		t.Error("MusicBrainz should not be called for a page load")
	})

	resp := get(t, app.URL+"/")

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	body, err := io.ReadAll(resp.Body)

	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(string(body), "guardarr") {
		t.Errorf("index.html not served, got: %q", body)
	}
}
