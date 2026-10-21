// Package musicbrainz is a minimal, allocation-conscious client for the
// MusicBrainz web service.
package musicbrainz

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	apiURL    = "https://musicbrainz.org/ws/2"
	userAgent = "guardarr/0.1.0 ( https://github.com/mateusguilherme/guardarr )"
	// MusicBrainz limits clients to ~1 request per second per IP and replies
	// 503 above that, so requests are spaced out instead of failed and retried.
	minInterval = 1100 * time.Millisecond
	// An artist's data and discography change rarely, while every page view
	// asks for them again; repeats served from memory skip the throttle wait
	// entirely. One hour bounds how long a brand-new release can stay hidden.
	cacheTTL = time.Hour
)

type Artist struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Disambiguation string `json:"disambiguation,omitempty"`
}

// Release is one entry of an artist's discography, backed by a MusicBrainz
// release group. For now only official albums are kept: primary type Album,
// no secondary type (those mark compilations, live albums, soundtracks and
// the like) and at least one release with official status (demos and other
// bootlegs never have one).
type Release struct {
	ID               string `json:"id"`
	Title            string `json:"title"`
	FirstReleaseDate string `json:"firstReleaseDate,omitempty"`
}

type Client struct {
	apiURL   string
	interval time.Duration // minimum spacing between requests
	http     *http.Client
	mu       sync.Mutex
	next     time.Time // earliest moment the next request may leave

	artists  *ttlCache[Artist]
	releases *ttlCache[[]Release]
}

func NewClient() *Client {
	return &Client{
		apiURL:   apiURL,
		interval: minInterval,
		artists:  newTTLCache[Artist](cacheTTL),
		releases: newTTLCache[[]Release](cacheTTL),
		http: &http.Client{
			Timeout: 15 * time.Second,
			// Keeping the TLS connection alive between requests saves a full
			// handshake round-trip to musicbrainz.org on every search.
			Transport: &http.Transport{
				MaxIdleConnsPerHost: 2,
				IdleConnTimeout:     90 * time.Second,
				ForceAttemptHTTP2:   true,
			},
		},
	}
}

// NewClientWithURL is a client against another base URL, for tests.
func NewClientWithURL(apiURL string) *Client {
	c := NewClient()
	c.apiURL = apiURL

	return c
}

// throttle reserves the next request slot and sleeps until it opens,
// honoring ctx cancellation while waiting.
func (c *Client) throttle(ctx context.Context) error {
	c.mu.Lock()

	wait := max(time.Until(c.next), 0)
	c.next = time.Now().Add(wait + c.interval)

	c.mu.Unlock()

	if wait == 0 {
		return nil
	}

	timer := time.NewTimer(wait)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// getJSON performs one throttled request against path and decodes the JSON
// response into out, straight off the socket; the body is never buffered whole.
func (c *Client) getJSON(ctx context.Context, path string, query url.Values, out any) error {
	if err := c.throttle(ctx); err != nil {
		return err
	}

	query.Set("fmt", "json")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.apiURL+path+"?"+query.Encode(), nil)

	if err != nil {
		return err
	}

	req.Header.Set("User-Agent", userAgent)
	resp, err := c.http.Do(req)

	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("musicbrainz: unexpected status %s", resp.Status)
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("musicbrainz: decode response: %w", err)
	}

	return nil
}

// Artist looks up one artist by its MBID, served from cache when fresh.
func (c *Client) Artist(ctx context.Context, id string) (Artist, error) {
	if artist, ok := c.artists.get(id); ok {
		return artist, nil
	}

	var artist Artist

	if err := c.getJSON(ctx, "/artist/"+url.PathEscape(id), url.Values{}, &artist); err != nil {
		return Artist{}, err
	}

	c.artists.put(id, artist)

	return artist, nil
}

func (c *Client) SearchArtists(ctx context.Context, term string) ([]Artist, error) {
	var body struct {
		Artists []Artist `json:"artists"`
	}

	if err := c.getJSON(ctx, "/artist", url.Values{"query": {term}}, &body); err != nil {
		return nil, err
	}

	// A missing or null artists field decodes to a nil slice, which would
	// re-encode as JSON null; normalize so callers always get an array.
	if body.Artists == nil {
		body.Artists = []Artist{}
	}

	return body.Artists, nil
}

// searchLimit is the most results MusicBrainz returns per search request.
const searchLimit = 100

// ArtistReleases returns every release of an artist, newest first, served
// from cache when fresh; the slice is shared, so callers must not mutate it.
// It searches release groups instead of browsing them: "official" is a
// status on individual releases, which a browse cannot filter by, so
// bootleg-only groups — garage demos, say — would slip in.
func (c *Client) ArtistReleases(ctx context.Context, artistID string) ([]Release, error) {
	if releases, ok := c.releases.get(artistID); ok {
		return releases, nil
	}

	releases := []Release{}

	// The search pages through the matches; the reported total says when
	// every page has been seen.
	for offset := 0; ; {
		var body struct {
			Count         int `json:"count"`
			ReleaseGroups []struct {
				ID               string   `json:"id"`
				Title            string   `json:"title"`
				FirstReleaseDate string   `json:"first-release-date"`
				SecondaryTypes   []string `json:"secondary-types"`
			} `json:"release-groups"`
		}

		query := url.Values{
			"query": {fmt.Sprintf(
				"arid:%s AND primarytype:album AND status:official AND -secondarytype:*",
				artistID,
			)},
			"limit":  {strconv.Itoa(searchLimit)},
			"offset": {strconv.Itoa(offset)},
		}

		if err := c.getJSON(ctx, "/release-group", query, &body); err != nil {
			return nil, err
		}

		for _, rg := range body.ReleaseGroups {
			// The query already excludes secondary types; this only guards
			// against the search index serving something else.
			if len(rg.SecondaryTypes) == 0 {
				releases = append(releases, Release{
					ID:               rg.ID,
					Title:            rg.Title,
					FirstReleaseDate: rg.FirstReleaseDate,
				})
			}
		}

		offset += len(body.ReleaseGroups)

		if len(body.ReleaseGroups) == 0 || offset >= body.Count {
			break
		}
	}

	// MusicBrainz dates are YYYY, YYYY-MM or YYYY-MM-DD, so they order
	// lexicographically; undated releases sink to the end.
	slices.SortFunc(releases, func(a, b Release) int {
		switch {
		case a.FirstReleaseDate == b.FirstReleaseDate:
			return strings.Compare(a.Title, b.Title)
		case a.FirstReleaseDate == "":
			return 1
		case b.FirstReleaseDate == "":
			return -1
		}

		return strings.Compare(b.FirstReleaseDate, a.FirstReleaseDate)
	})

	c.releases.put(artistID, releases)

	return releases, nil
}

// ttlCache is a mutex-guarded map whose entries expire after a fixed
// duration. Errors are never stored, so a failed lookup retries right away.
type ttlCache[V any] struct {
	mu      sync.Mutex
	ttl     time.Duration
	entries map[string]ttlEntry[V]
}

type ttlEntry[V any] struct {
	value   V
	expires time.Time
}

func newTTLCache[V any](ttl time.Duration) *ttlCache[V] {
	return &ttlCache[V]{ttl: ttl, entries: map[string]ttlEntry[V]{}}
}

func (c *ttlCache[V]) get(key string) (V, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, ok := c.entries[key]

	if !ok || time.Now().After(entry.expires) {
		var zero V

		return zero, false
	}

	return entry.value, true
}

// put stores value under key and sweeps expired entries out, keeping the
// cache no larger than the set of keys asked about within one ttl.
func (c *ttlCache[V]) put(key string, value V) {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()

	for k, entry := range c.entries {
		if now.After(entry.expires) {
			delete(c.entries, k)
		}
	}

	c.entries[key] = ttlEntry[V]{value: value, expires: now.Add(c.ttl)}
}
