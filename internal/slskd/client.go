// Package slskd is a client for the REST API of slskd, the Soulseek daemon
// that guardarr hands its searches and downloads to.
package slskd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// ErrNotConfigured is what every call returns while the client has no slskd
// to talk to.
var ErrNotConfigured = errors.New("slskd: not configured")

// StatusError is a request slskd answered with an error status, with the
// reason it gave, if any.
type StatusError struct {
	Code   int
	Status string // e.g. "401 Unauthorized"
	Reason string
}

func (e *StatusError) Error() string {
	if e.Reason == "" {
		return "slskd: unexpected status " + e.Status
	}

	return "slskd: unexpected status " + e.Status + ": " + e.Reason
}

// ServerState is slskd's connection to the Soulseek network: it can only
// search and download while logged in.
type ServerState struct {
	Username    string `json:"username"`
	IsConnected bool   `json:"isConnected"`
	IsLoggedIn  bool   `json:"isLoggedIn"`
}

// Search is a search slskd runs across the network in the background,
// collecting responses until it completes.
type Search struct {
	ID            string `json:"id"`
	SearchText    string `json:"searchText"`
	State         string `json:"state"` // e.g. "InProgress" or "Completed, TimedOut"
	IsComplete    bool   `json:"isComplete"`
	ResponseCount int    `json:"responseCount"`
	FileCount     int    `json:"fileCount"`
}

// Response is one user's answer to a search: the matching files they share
// and how soon they would send them.
type Response struct {
	Username          string `json:"username"`
	Files             []File `json:"files"`
	HasFreeUploadSlot bool   `json:"hasFreeUploadSlot"`
	QueueLength       int64  `json:"queueLength"`
	UploadSpeed       int    `json:"uploadSpeed"` // bytes per second
}

// File is a shared file. Filename is the full path on the sharer's side,
// as Soulseek needs it to request the file. The audio fields are zero when
// the sharer's client did not report them.
type File struct {
	Filename   string `json:"filename"`
	Size       int64  `json:"size"`
	Extension  string `json:"extension,omitempty"`
	BitRate    int    `json:"bitRate,omitempty"`
	BitDepth   int    `json:"bitDepth,omitempty"`
	SampleRate int    `json:"sampleRate,omitempty"`
	Length     int    `json:"length,omitempty"` // seconds
}

// Batch is a set of downloads from one user that slskd tracks together.
// Failures lists the files it refused to queue, each with the reason why.
type Batch struct {
	ID       string
	Failures []Failure
}

type Failure struct {
	Filename string `json:"filename"`
	Message  string `json:"message"`
}

type Client struct {
	http *http.Client

	mu     sync.RWMutex
	apiURL string // empty while there is no slskd to talk to
	apiKey string
}

// NewClient returns a client configured as Configure describes.
func NewClient(baseURL, apiKey string) *Client {
	c := &Client{http: &http.Client{Timeout: 15 * time.Second}}
	c.Configure(baseURL, apiKey)

	return c
}

// Configure points the client at the slskd at baseURL, its web address
// including any URL base it is set up with; an empty baseURL leaves the
// client with no slskd. apiKey goes in every request, unless it is empty
// for an slskd that runs without authentication. Requests already on their
// way finish against the previous slskd.
func (c *Client) Configure(baseURL, apiKey string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.apiURL = ""

	if baseURL != "" {
		c.apiURL = strings.TrimSuffix(baseURL, "/") + "/api/v0"
	}

	c.apiKey = apiKey
}

func (c *Client) settings() (apiURL, apiKey string) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.apiURL, c.apiKey
}

// do sends one request to the API, with in as its JSON body unless nil, and
// decodes the JSON response into out unless nil.
func (c *Client) do(ctx context.Context, method, path string, in, out any) error {
	apiURL, apiKey := c.settings()

	if apiURL == "" {
		return ErrNotConfigured
	}

	var body io.Reader

	if in != nil {
		data, err := json.Marshal(in)

		if err != nil {
			return err
		}

		body = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, apiURL+path, body)

	if err != nil {
		return err
	}

	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	if apiKey != "" {
		req.Header.Set("X-API-Key", apiKey)
	}

	resp, err := c.http.Do(req)

	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return &StatusError{Code: resp.StatusCode, Status: resp.Status, Reason: refusal(resp.Body)}
	}

	if out == nil {
		return nil
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("slskd: decode response: %w", err)
	}

	return nil
}

// refusal reads why slskd refused a request. It says so in a short JSON
// string, or in plain text from a proxy in front of it; either may be empty.
func refusal(body io.Reader) string {
	data, _ := io.ReadAll(io.LimitReader(body, 512))

	var reason string

	if err := json.Unmarshal(data, &reason); err != nil {
		reason = string(data)
	}

	return strings.TrimSpace(reason)
}

func (c *Client) Server(ctx context.Context) (ServerState, error) {
	var state ServerState

	if err := c.do(ctx, http.MethodGet, "/server", nil, &state); err != nil {
		return ServerState{}, err
	}

	return state, nil
}

// StartSearch asks slskd to search the network for text and returns as soon
// as the search starts; Search tells when it is complete.
func (c *Client) StartSearch(ctx context.Context, text string) (Search, error) {
	in := struct {
		SearchText string `json:"searchText"`
	}{text}

	var search Search

	if err := c.do(ctx, http.MethodPost, "/searches", in, &search); err != nil {
		return Search{}, err
	}

	return search, nil
}

func (c *Client) Search(ctx context.Context, id string) (Search, error) {
	var search Search

	if err := c.do(ctx, http.MethodGet, "/searches/"+url.PathEscape(id), nil, &search); err != nil {
		return Search{}, err
	}

	return search, nil
}

// SearchResponses returns the responses a search has collected so far.
func (c *Client) SearchResponses(ctx context.Context, id string) ([]Response, error) {
	path := "/searches/" + url.PathEscape(id) + "/responses"

	var responses []Response

	if err := c.do(ctx, http.MethodGet, path, nil, &responses); err != nil {
		return nil, err
	}

	return responses, nil
}

// Download queues files from username's share as one batch, tied to the
// search that found them. slskd queues what it can, so a nil error does
// not mean every file made it: the refused ones are in the batch Failures.
func (c *Client) Download(ctx context.Context, searchID, username string, files []File) (Batch, error) {
	type item struct {
		Filename string `json:"filename"`
		Size     int64  `json:"size"`
	}

	in := struct {
		SearchID string `json:"searchId,omitempty"`
		Username string `json:"username"`
		Files    []item `json:"files"`
	}{SearchID: searchID, Username: username}

	for _, file := range files {
		in.Files = append(in.Files, item{file.Filename, file.Size})
	}

	var out struct {
		Batch struct {
			ID string `json:"id"`
		} `json:"batch"`
		Failures []Failure `json:"failures"`
	}

	if err := c.do(ctx, http.MethodPost, "/transfers/downloads/batches", in, &out); err != nil {
		return Batch{}, err
	}

	return Batch{ID: out.Batch.ID, Failures: out.Failures}, nil
}
