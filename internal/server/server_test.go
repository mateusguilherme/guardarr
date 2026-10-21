package server_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/mateusguilherme/guardarr/internal/musicbrainz"
	"github.com/mateusguilherme/guardarr/internal/server"
	"github.com/mateusguilherme/guardarr/internal/store"
)

// newApp serves the application against a fake MusicBrainz handler, a fresh
// temporary database, and an in-memory frontend, all cleaned up with the test.
func newApp(t *testing.T, fakeMusicBrainz http.HandlerFunc) *httptest.Server {
	t.Helper()

	fake := httptest.NewServer(fakeMusicBrainz)
	t.Cleanup(fake.Close)

	st, err := store.Open(filepath.Join(t.TempDir(), "test.db"))

	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { st.Close() })

	static := fstest.MapFS{
		"index.html": &fstest.MapFile{Data: []byte("<!doctype html><title>guardarr</title>")},
	}

	app := httptest.NewServer(server.New(musicbrainz.NewClientWithURL(fake.URL), st, static))
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

func put(t *testing.T, url string) *http.Response {
	t.Helper()

	req, err := http.NewRequest(http.MethodPut, url, nil)

	if err != nil {
		t.Fatal(err)
	}

	resp, err := http.DefaultClient.Do(req)

	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { resp.Body.Close() })

	return resp
}

func followedIDs(t *testing.T, app *httptest.Server) []string {
	t.Helper()

	resp := get(t, app.URL+"/api/followed")

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var ids []string

	if err := json.NewDecoder(resp.Body).Decode(&ids); err != nil {
		t.Fatal(err)
	}

	return ids
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

// noMusicBrainz fails the test if the MusicBrainz API is reached.
func noMusicBrainz(t *testing.T) http.HandlerFunc {
	return func(_ http.ResponseWriter, _ *http.Request) {
		t.Error("MusicBrainz should not be called")
	}
}

func TestFollowedStartsEmpty(t *testing.T) {
	app := newApp(t, noMusicBrainz(t))

	ids := followedIDs(t, app)

	if ids == nil {
		t.Fatal("want an empty JSON array, got null")
	}

	if len(ids) != 0 {
		t.Errorf("want no ids, got %v", ids)
	}
}

func TestFollowArtistPersists(t *testing.T) {
	app := newApp(t, noMusicBrainz(t))

	resp := put(t, app.URL+"/api/followed/abc")

	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusNoContent)
	}

	if ids := followedIDs(t, app); !slices.Equal(ids, []string{"abc"}) {
		t.Errorf("ids = %v, want %v", ids, []string{"abc"})
	}
}

func TestFollowArtistTwiceKeepsOneID(t *testing.T) {
	app := newApp(t, noMusicBrainz(t))

	for range 2 {
		if resp := put(t, app.URL+"/api/followed/abc"); resp.StatusCode != http.StatusNoContent {
			t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusNoContent)
		}
	}

	if ids := followedIDs(t, app); !slices.Equal(ids, []string{"abc"}) {
		t.Errorf("ids = %v, want %v", ids, []string{"abc"})
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
