package musicbrainz

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"slices"
	"sync"
	"testing"
	"time"
)

// testClient returns a client with a short throttle interval so tests
// do not sleep for the real MusicBrainz spacing.
func testClient(apiURL string) *Client {
	c := NewClientWithURL(apiURL)
	c.interval = 10 * time.Millisecond

	return c
}

func TestSearchArtists(t *testing.T) {
	fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("query"); got != "caetano veloso" {
			t.Errorf("query = %q, want %q", got, "caetano veloso")
		}

		if got := r.URL.Query().Get("fmt"); got != "json" {
			t.Errorf("fmt = %q, want %q", got, "json")
		}

		if r.Header.Get("User-Agent") != userAgent {
			t.Errorf("User-Agent = %q, want %q", r.Header.Get("User-Agent"), userAgent)
		}

		w.Write([]byte(`{"artists": [{"id": "abc", "name": "Caetano Veloso", "score": 100}]}`))
	}))

	defer fake.Close()

	artists, err := testClient(fake.URL).SearchArtists(context.Background(), "caetano veloso")

	if err != nil {
		t.Fatal(err)
	}

	if len(artists) != 1 || artists[0].Name != "Caetano Veloso" {
		t.Errorf("unexpected artists: %+v", artists)
	}
}

func TestSearchArtistsUnexpectedStatus(t *testing.T) {
	fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))

	defer fake.Close()

	if _, err := testClient(fake.URL).SearchArtists(context.Background(), "x"); err == nil {
		t.Fatal("want an error on a 503 response, got nil")
	}
}

func TestSearchArtistsNormalizesNullToEmpty(t *testing.T) {
	fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(`{"artists": null}`))
	}))

	defer fake.Close()

	artists, err := testClient(fake.URL).SearchArtists(context.Background(), "x")

	if err != nil {
		t.Fatal(err)
	}

	if artists == nil {
		t.Fatal("want an empty slice, got nil")
	}

	if len(artists) != 0 {
		t.Errorf("want no artists, got %+v", artists)
	}
}

func TestArtist(t *testing.T) {
	fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Path; got != "/artist/abc" {
			t.Errorf("path = %q, want %q", got, "/artist/abc")
		}

		if got := r.URL.Query().Get("fmt"); got != "json" {
			t.Errorf("fmt = %q, want %q", got, "json")
		}

		w.Write([]byte(`{"id": "abc", "name": "Caetano Veloso", "disambiguation": "Brazilian singer"}`))
	}))

	defer fake.Close()

	artist, err := testClient(fake.URL).Artist(context.Background(), "abc")

	if err != nil {
		t.Fatal(err)
	}

	want := Artist{ID: "abc", Name: "Caetano Veloso", Disambiguation: "Brazilian singer"}

	if artist != want {
		t.Errorf("artist = %+v, want %+v", artist, want)
	}
}

func TestArtistUnexpectedStatus(t *testing.T) {
	fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))

	defer fake.Close()

	if _, err := testClient(fake.URL).Artist(context.Background(), "nope"); err == nil {
		t.Fatal("want an error on a 404 response, got nil")
	}
}

func TestArtistReleasesKeepsOfficialAlbumsNewestFirst(t *testing.T) {
	fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Path; got != "/release-group" {
			t.Errorf("path = %q, want %q", got, "/release-group")
		}

		q := r.URL.Query()
		want := "arid:abc AND primarytype:album AND status:official AND -secondarytype:*"

		if got := q.Get("query"); got != want {
			t.Errorf("query = %q, want %q", got, want)
		}

		if got := q.Get("fmt"); got != "json" {
			t.Errorf("fmt = %q, want %q", got, "json")
		}

		w.Write([]byte(`{
			"count": 4,
			"release-groups": [
				{"id": "rg1", "title": "Transa", "first-release-date": "1972-01-01", "secondary-types": []},
				{"id": "rg2", "title": "Greatest Hits", "first-release-date": "1990", "secondary-types": ["Compilation"]},
				{"id": "rg3", "title": "Araçá Azul", "first-release-date": "1973"},
				{"id": "rg4", "title": "Lost Tapes", "secondary-types": []}
			]
		}`))
	}))

	defer fake.Close()

	releases, err := testClient(fake.URL).ArtistReleases(context.Background(), "abc")

	if err != nil {
		t.Fatal(err)
	}

	// The compilation is dropped, the rest come newest first with the
	// undated album last.
	want := []Release{
		{ID: "rg3", Title: "Araçá Azul", FirstReleaseDate: "1973"},
		{ID: "rg1", Title: "Transa", FirstReleaseDate: "1972-01-01"},
		{ID: "rg4", Title: "Lost Tapes"},
	}

	if !slices.Equal(releases, want) {
		t.Errorf("releases = %+v, want %+v", releases, want)
	}
}

func TestArtistReleasesPaginates(t *testing.T) {
	var offsets []string

	fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		offset := r.URL.Query().Get("offset")
		offsets = append(offsets, offset)

		if offset == "0" {
			w.Write([]byte(`{"count": 2, "release-groups": [{"id": "rg1", "title": "One", "first-release-date": "2001"}]}`))
			return
		}

		w.Write([]byte(`{"count": 2, "release-groups": [{"id": "rg2", "title": "Two", "first-release-date": "2002"}]}`))
	}))

	defer fake.Close()

	releases, err := testClient(fake.URL).ArtistReleases(context.Background(), "abc")

	if err != nil {
		t.Fatal(err)
	}

	if want := []string{"0", "1"}; !slices.Equal(offsets, want) {
		t.Errorf("offsets = %v, want %v", offsets, want)
	}

	want := []Release{
		{ID: "rg2", Title: "Two", FirstReleaseDate: "2002"},
		{ID: "rg1", Title: "One", FirstReleaseDate: "2001"},
	}

	if !slices.Equal(releases, want) {
		t.Errorf("releases = %+v, want %+v", releases, want)
	}
}

func TestArtistReleasesUnexpectedStatus(t *testing.T) {
	fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))

	defer fake.Close()

	if _, err := testClient(fake.URL).ArtistReleases(context.Background(), "abc"); err == nil {
		t.Fatal("want an error on a 503 response, got nil")
	}
}

func TestArtistReleasesNormalizesNullToEmpty(t *testing.T) {
	fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(`{"count": 0, "release-groups": null}`))
	}))

	defer fake.Close()

	releases, err := testClient(fake.URL).ArtistReleases(context.Background(), "abc")

	if err != nil {
		t.Fatal(err)
	}

	if releases == nil {
		t.Fatal("want an empty slice, got nil")
	}

	if len(releases) != 0 {
		t.Errorf("want no releases, got %+v", releases)
	}
}

func TestArtistServesRepeatsFromCache(t *testing.T) {
	requests := 0

	fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests++

		w.Write([]byte(`{"id": "abc", "name": "Caetano Veloso"}`))
	}))

	defer fake.Close()

	c := testClient(fake.URL)

	for range 2 {
		artist, err := c.Artist(context.Background(), "abc")

		if err != nil {
			t.Fatal(err)
		}

		if artist.Name != "Caetano Veloso" {
			t.Errorf("artist = %+v, want name %q", artist, "Caetano Veloso")
		}
	}

	if requests != 1 {
		t.Errorf("server saw %d requests, want 1", requests)
	}
}

func TestArtistReleasesServesRepeatsFromCache(t *testing.T) {
	requests := 0

	fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests++

		w.Write([]byte(`{"count": 1, "release-groups": [{"id": "rg1", "title": "Transa", "first-release-date": "1972"}]}`))
	}))

	defer fake.Close()

	c := testClient(fake.URL)
	want := []Release{{ID: "rg1", Title: "Transa", FirstReleaseDate: "1972"}}

	for range 2 {
		releases, err := c.ArtistReleases(context.Background(), "abc")

		if err != nil {
			t.Fatal(err)
		}

		if !slices.Equal(releases, want) {
			t.Errorf("releases = %+v, want %+v", releases, want)
		}
	}

	if requests != 1 {
		t.Errorf("server saw %d requests, want 1", requests)
	}
}

func TestArtistReleasesCacheExpires(t *testing.T) {
	requests := 0

	fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests++

		w.Write([]byte(`{"count": 0, "release-groups": []}`))
	}))

	defer fake.Close()

	c := testClient(fake.URL)
	c.releases = newTTLCache[[]Release](10 * time.Millisecond)

	for range 2 {
		if _, err := c.ArtistReleases(context.Background(), "abc"); err != nil {
			t.Fatal(err)
		}

		time.Sleep(20 * time.Millisecond)
	}

	if requests != 2 {
		t.Errorf("server saw %d requests, want 2 once the entry expired", requests)
	}
}

func TestThrottleSpacesRequests(t *testing.T) {
	c := testClient("http://unused")
	start := time.Now()

	for range 3 {
		if err := c.throttle(context.Background()); err != nil {
			t.Fatal(err)
		}
	}

	// The first call is free; the next two must each wait one interval.
	if elapsed := time.Since(start); elapsed < 2*c.interval {
		t.Errorf("3 calls took %v, want at least %v", elapsed, 2*c.interval)
	}
}

func TestThrottleSpacesConcurrentRequests(t *testing.T) {
	c := testClient("http://unused")

	var wg sync.WaitGroup

	start := time.Now()

	for range 3 {
		wg.Go(func() {
			if err := c.throttle(context.Background()); err != nil {
				t.Error(err)
			}
		})
	}

	wg.Wait()

	if elapsed := time.Since(start); elapsed < 2*c.interval {
		t.Errorf("3 concurrent calls took %v, want at least %v", elapsed, 2*c.interval)
	}
}

func TestThrottleHonorsCancellation(t *testing.T) {
	c := testClient("http://unused")
	c.interval = time.Hour

	// The first call reserves the slot one hour ahead...
	if err := c.throttle(context.Background()); err != nil {
		t.Fatal(err)
	}

	// ...so the second would sleep until then, but the deadline fires first.
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)

	defer cancel()

	start := time.Now()
	err := c.throttle(ctx)

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err = %v, want %v", err, context.DeadlineExceeded)
	}

	if elapsed := time.Since(start); elapsed > time.Second {
		t.Errorf("cancellation took %v, want a prompt return", elapsed)
	}
}
