package slskd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

const testKey = "a-key-of-at-least-16-chars"

// fakeSlskd serves handler as the slskd API and returns a client for it
// that sends testKey.
func fakeSlskd(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()

	fake := httptest.NewServer(handler)
	t.Cleanup(fake.Close)

	return NewClient(fake.URL, testKey)
}

// expectRequest fails the test unless r is a method request for path.
func expectRequest(t *testing.T, r *http.Request, method, path string) {
	t.Helper()

	if r.Method != method || r.URL.Path != path {
		t.Errorf("request = %s %s, want %s %s", r.Method, r.URL.Path, method, path)
	}

	if got := r.Header.Get("X-API-Key"); got != testKey {
		t.Errorf("X-API-Key = %q, want %q", got, testKey)
	}
}

// expectBody fails the test unless r carries want as its JSON body.
func expectBody(t *testing.T, r *http.Request, want string) {
	t.Helper()

	if got := r.Header.Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q, want %q", got, "application/json")
	}

	body, err := io.ReadAll(r.Body)

	if err != nil {
		t.Error(err)
	}

	if string(body) != want {
		t.Errorf("body = %s, want %s", body, want)
	}
}

func TestState(t *testing.T) {
	sk := fakeSlskd(t, func(w http.ResponseWriter, r *http.Request) {
		expectRequest(t, r, http.MethodGet, "/api/v0/application")
		w.Write([]byte(`{
			"pendingRestart": false,
			"server": {
				"address": "vps.slsknet.org:2271",
				"state": "Connected, LoggedIn",
				"isConnected": true,
				"isLoggedIn": true,
				"isTransitioning": false
			},
			"user": {
				"username": "guardarr",
				"privileges": {"isPrivileged": false, "privilegesRemaining": 0}
			}
		}`))
	})

	state, err := sk.State(context.Background())

	if err != nil {
		t.Fatal(err)
	}

	want := State{
		Server: ServerState{IsConnected: true, IsLoggedIn: true},
		User:   UserState{Username: "guardarr"},
	}

	if state != want {
		t.Errorf("state = %+v, want %+v", state, want)
	}
}

func TestKeepsTheURLBase(t *testing.T) {
	fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/slskd/api/v0/application" {
			t.Errorf("path = %q, want %q", r.URL.Path, "/slskd/api/v0/application")
		}

		w.Write([]byte(`{}`))
	}))
	t.Cleanup(fake.Close)

	if _, err := NewClient(fake.URL+"/slskd/", testKey).State(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestSendsNoKeyWithoutOne(t *testing.T) {
	fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := r.Header["X-Api-Key"]; ok {
			t.Error("want no X-API-Key header")
		}

		w.Write([]byte(`{}`))
	}))
	t.Cleanup(fake.Close)

	if _, err := NewClient(fake.URL, "").State(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestStartSearch(t *testing.T) {
	sk := fakeSlskd(t, func(w http.ResponseWriter, r *http.Request) {
		expectRequest(t, r, http.MethodPost, "/api/v0/searches")
		expectBody(t, r, `{"searchText":"caetano veloso transa"}`)
		w.Write([]byte(`{
			"id": "7f3c1d2e-0000-4000-8000-000000000001",
			"searchText": "caetano veloso transa",
			"state": "InProgress",
			"isComplete": false,
			"responseCount": 0,
			"fileCount": 0,
			"responses": []
		}`))
	})

	search, err := sk.StartSearch(context.Background(), "caetano veloso transa")

	if err != nil {
		t.Fatal(err)
	}

	want := Search{
		ID:         "7f3c1d2e-0000-4000-8000-000000000001",
		SearchText: "caetano veloso transa",
		State:      "InProgress",
	}

	if search != want {
		t.Errorf("search = %+v, want %+v", search, want)
	}
}

func TestSearch(t *testing.T) {
	sk := fakeSlskd(t, func(w http.ResponseWriter, r *http.Request) {
		expectRequest(t, r, http.MethodGet, "/api/v0/searches/s1")
		w.Write([]byte(`{
			"id": "s1",
			"searchText": "caetano veloso transa",
			"state": "Completed, TimedOut",
			"isComplete": true,
			"responseCount": 2,
			"fileCount": 14
		}`))
	})

	search, err := sk.Search(context.Background(), "s1")

	if err != nil {
		t.Fatal(err)
	}

	want := Search{
		ID:            "s1",
		SearchText:    "caetano veloso transa",
		State:         "Completed, TimedOut",
		IsComplete:    true,
		ResponseCount: 2,
		FileCount:     14,
	}

	if search != want {
		t.Errorf("search = %+v, want %+v", search, want)
	}
}

func TestSearchResponses(t *testing.T) {
	sk := fakeSlskd(t, func(w http.ResponseWriter, r *http.Request) {
		expectRequest(t, r, http.MethodGet, "/api/v0/searches/s1/responses")
		w.Write([]byte(`[{
			"username": "some user",
			"fileCount": 1,
			"files": [{
				"filename": "@@music\\Caetano Veloso\\Transa\\01 You Don't Know Me.flac",
				"size": 31457280,
				"extension": "flac",
				"bitDepth": 16,
				"sampleRate": 44100,
				"length": 230,
				"code": 1,
				"isLocked": false
			}],
			"hasFreeUploadSlot": true,
			"lockedFileCount": 0,
			"lockedFiles": [],
			"queueLength": 3,
			"token": 42,
			"uploadSpeed": 1048576
		}]`))
	})

	responses, err := sk.SearchResponses(context.Background(), "s1")

	if err != nil {
		t.Fatal(err)
	}

	want := []Response{{
		Username: "some user",
		Files: []File{{
			Filename:   `@@music\Caetano Veloso\Transa\01 You Don't Know Me.flac`,
			Size:       31457280,
			Extension:  "flac",
			BitDepth:   16,
			SampleRate: 44100,
			Length:     230,
		}},
		HasFreeUploadSlot: true,
		QueueLength:       3,
		UploadSpeed:       1048576,
	}}

	if !reflect.DeepEqual(responses, want) {
		t.Errorf("responses = %+v, want %+v", responses, want)
	}
}

var transa = []File{
	{Filename: `@@music\Caetano Veloso\Transa\01 You Don't Know Me.flac`, Size: 31457280, Extension: "flac"},
	{Filename: `@@music\Caetano Veloso\Transa\02 Nine Out of Ten.flac`, Size: 29360128, Extension: "flac"},
}

func TestDownload(t *testing.T) {
	sk := fakeSlskd(t, func(w http.ResponseWriter, r *http.Request) {
		expectRequest(t, r, http.MethodPost, "/api/v0/transfers/downloads/batches")
		expectBody(t, r, `{"searchId":"s1","username":"some user","files":[`+
			`{"filename":"@@music\\Caetano Veloso\\Transa\\01 You Don't Know Me.flac","size":31457280},`+
			`{"filename":"@@music\\Caetano Veloso\\Transa\\02 Nine Out of Ten.flac","size":29360128}]}`)
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"batch": {"id": "b1", "username": "some user"}, "failures": []}`))
	})

	batch, err := sk.Download(context.Background(), "s1", "some user", transa)

	if err != nil {
		t.Fatal(err)
	}

	if batch.ID != "b1" || len(batch.Failures) != 0 {
		t.Errorf("batch = %+v, want b1 with no failures", batch)
	}
}

func TestDownloadReportsTheRefusedFiles(t *testing.T) {
	sk := fakeSlskd(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusMultiStatus)
		w.Write([]byte(`{
			"batch": {"id": "b1"},
			"failures": [{
				"filename": "@@music\\Caetano Veloso\\Transa\\02 Nine Out of Ten.flac",
				"message": "already queued"
			}]
		}`))
	})

	batch, err := sk.Download(context.Background(), "s1", "some user", transa)

	if err != nil {
		t.Fatal(err)
	}

	want := []Failure{{Filename: transa[1].Filename, Message: "already queued"}}

	if !reflect.DeepEqual(batch.Failures, want) {
		t.Errorf("failures = %+v, want %+v", batch.Failures, want)
	}
}

func TestUnexpectedStatusSaysWhy(t *testing.T) {
	sk := fakeSlskd(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		w.Write([]byte(`"The server connection must be connected and logged in to perform a search (currently: Disconnected)"`))
	})

	_, err := sk.StartSearch(context.Background(), "x")
	want := "slskd: unexpected status 409 Conflict: The server connection must be connected" +
		" and logged in to perform a search (currently: Disconnected)"

	if err == nil || err.Error() != want {
		t.Errorf("err = %v, want %q", err, want)
	}
}

func TestUnexpectedStatusSaysWhyInPlainText(t *testing.T) {
	sk := fakeSlskd(t, func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
	})

	_, err := sk.State(context.Background())
	want := "slskd: unexpected status 502 Bad Gateway: Bad Gateway"

	if err == nil || err.Error() != want {
		t.Errorf("err = %v, want %q", err, want)
	}
}

func TestUnexpectedStatusWithoutReason(t *testing.T) {
	sk := fakeSlskd(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})

	_, err := sk.State(context.Background())

	if err == nil || err.Error() != "slskd: unexpected status 401 Unauthorized" {
		t.Errorf("err = %v, want the bare status", err)
	}
}

func TestStatusErrorCarriesTheCode(t *testing.T) {
	sk := fakeSlskd(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})

	_, err := sk.State(context.Background())

	var refused *StatusError

	if !errors.As(err, &refused) || refused.Code != http.StatusUnauthorized {
		t.Errorf("err = %v, want a StatusError with code 401", err)
	}
}

func TestUnconfiguredClientCallsNothing(t *testing.T) {
	sk := NewClient("", testKey)

	if _, err := sk.State(context.Background()); !errors.Is(err, ErrNotConfigured) {
		t.Errorf("err = %v, want ErrNotConfigured", err)
	}
}

func TestConfigureSwitchesTheSlskd(t *testing.T) {
	var hits [2]int

	fakes := [2]*httptest.Server{}

	for i := range fakes {
		fakes[i] = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			hits[i]++

			if got := r.Header.Get("X-API-Key"); got != fmt.Sprint("key-", i) {
				t.Errorf("slskd %d got X-API-Key %q", i, got)
			}

			w.Write([]byte(`{}`))
		}))
		t.Cleanup(fakes[i].Close)
	}

	sk := NewClient(fakes[0].URL, "key-0")

	if _, err := sk.State(context.Background()); err != nil {
		t.Fatal(err)
	}

	sk.Configure(fakes[1].URL, "key-1")

	if _, err := sk.State(context.Background()); err != nil {
		t.Fatal(err)
	}

	sk.Configure("", "")

	if _, err := sk.State(context.Background()); !errors.Is(err, ErrNotConfigured) {
		t.Errorf("err = %v, want ErrNotConfigured once cleared", err)
	}

	if hits != [2]int{1, 1} {
		t.Errorf("hits = %v, want one request to each slskd", hits)
	}
}
