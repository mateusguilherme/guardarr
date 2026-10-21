package musicbrainz

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
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
