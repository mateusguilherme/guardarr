// Package musicbrainz is a minimal, allocation-conscious client for the
// MusicBrainz web service.
package musicbrainz

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"
)

const (
	apiURL    = "https://musicbrainz.org/ws/2"
	userAgent = "guardarr/0.1.0 ( https://github.com/mateusguilherme/guardarr )"
	// MusicBrainz limits clients to ~1 request per second per IP and replies
	// 503 above that, so requests are spaced out instead of failed and retried.
	minInterval = 1100 * time.Millisecond
)

type Artist struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Disambiguation string `json:"disambiguation,omitempty"`
}

type Client struct {
	apiURL   string
	interval time.Duration // minimum spacing between requests
	http     *http.Client
	mu       sync.Mutex
	next     time.Time // earliest moment the next request may leave
}

func NewClient() *Client {
	return &Client{
		apiURL:   apiURL,
		interval: minInterval,
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

func (c *Client) SearchArtists(ctx context.Context, term string) ([]Artist, error) {
	if err := c.throttle(ctx); err != nil {
		return nil, err
	}

	query := url.Values{"query": {term}, "fmt": {"json"}}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.apiURL+"/artist?"+query.Encode(), nil)

	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", userAgent)
	resp, err := c.http.Do(req)

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("musicbrainz: unexpected status %s", resp.Status)
	}

	// Decode straight off the socket; the response body is never buffered whole.
	var body struct {
		Artists []Artist `json:"artists"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("musicbrainz: decode response: %w", err)
	}

	// A missing or null artists field decodes to a nil slice, which would
	// re-encode as JSON null; normalize so callers always get an array.
	if body.Artists == nil {
		body.Artists = []Artist{}
	}

	return body.Artists, nil
}
