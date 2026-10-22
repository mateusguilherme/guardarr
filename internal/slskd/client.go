// Package slskd is a client for the REST API of slskd, the Soulseek daemon
// that guardarr hands its searches and downloads to.
package slskd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

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
	apiURL string
	apiKey string
	http   *http.Client
}

// NewClient talks to the slskd at baseURL, its web address including any
// URL base it is set up with. apiKey goes in every request, unless it is
// empty for an slskd that runs without authentication.
func NewClient(baseURL, apiKey string) *Client {
	return &Client{
		apiURL: strings.TrimSuffix(baseURL, "/") + "/api/v0",
		apiKey: apiKey,
		http:   &http.Client{Timeout: 15 * time.Second},
	}
}

// do sends one request to the API, with in as its JSON body unless nil, and
// decodes the JSON response into out unless nil.
func (c *Client) do(ctx context.Context, method, path string, in, out any) error {
	var body io.Reader

	if in != nil {
		data, err := json.Marshal(in)

		if err != nil {
			return err
		}

		body = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.apiURL+path, body)

	if err != nil {
		return err
	}

	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	if c.apiKey != "" {
		req.Header.Set("X-API-Key", c.apiKey)
	}

	resp, err := c.http.Do(req)

	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		if reason := refusal(resp.Body); reason != "" {
			return fmt.Errorf("slskd: unexpected status %s: %s", resp.Status, reason)
		}

		return fmt.Errorf("slskd: unexpected status %s", resp.Status)
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
