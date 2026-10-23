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
	"github.com/mateusguilherme/guardarr/internal/slskd"
	"github.com/mateusguilherme/guardarr/internal/store"
)

// newApp serves the application against a fake MusicBrainz handler, a fresh
// temporary database, and an in-memory frontend, all cleaned up with the test.
// slskd starts out not set up, as on a first run.
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
		"200.html":                      &fstest.MapFile{Data: []byte("<!doctype html><title>guardarr</title>")},
		"robots.txt":                    &fstest.MapFile{Data: []byte("User-agent: *")},
		"_app/version.json":             &fstest.MapFile{Data: []byte(`{"version":"1"}`)},
		"_app/immutable/entry-abc12.js": &fstest.MapFile{Data: []byte("console.log('guardarr')")},
	}

	app := httptest.NewServer(server.New(server.Deps{
		MusicBrainz: musicbrainz.NewClientWithURL(fake.URL),
		Slskd:       slskd.NewClient("", ""),
		Store:       st,
		Static:      static,
	}))
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

func put(t *testing.T, url, body string) *http.Response {
	t.Helper()

	return send(t, http.MethodPut, url, body)
}

func patch(t *testing.T, url, body string) *http.Response {
	t.Helper()

	return send(t, http.MethodPatch, url, body)
}

// send makes a method request to url with body as its JSON.
func send(t *testing.T, method, url, body string) *http.Response {
	t.Helper()

	req, err := http.NewRequest(method, url, strings.NewReader(body))

	if err != nil {
		t.Fatal(err)
	}

	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)

	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { resp.Body.Close() })

	return resp
}

func followed(t *testing.T, app *httptest.Server) []store.Artist {
	t.Helper()

	resp := get(t, app.URL+"/api/followed")

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var artists []store.Artist

	if err := json.NewDecoder(resp.Body).Decode(&artists); err != nil {
		t.Fatal(err)
	}

	return artists
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

func TestLookupArtistReturnsIt(t *testing.T) {
	app := newApp(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Path; got != "/artist/abc" {
			t.Errorf("path = %q, want %q", got, "/artist/abc")
		}

		w.Write([]byte(`{"id": "abc", "name": "Caetano Veloso", "disambiguation": "Brazilian singer"}`))
	})

	resp := get(t, app.URL+"/api/artists/abc")

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var artist musicbrainz.Artist

	if err := json.NewDecoder(resp.Body).Decode(&artist); err != nil {
		t.Fatal(err)
	}

	want := musicbrainz.Artist{ID: "abc", Name: "Caetano Veloso", Disambiguation: "Brazilian singer"}

	if artist != want {
		t.Errorf("artist = %+v, want %+v", artist, want)
	}
}

func TestLookupArtistReportsUpstreamFailure(t *testing.T) {
	app := newApp(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	resp := get(t, app.URL+"/api/artists/abc")

	if resp.StatusCode != http.StatusBadGateway {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadGateway)
	}
}

func TestArtistReleasesReturnsResults(t *testing.T) {
	app := newApp(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("query"); !strings.Contains(got, "arid:abc") {
			t.Errorf("query = %q, want it to mention arid:abc", got)
		}

		w.Write([]byte(`{"count": 1, "release-groups": [{"id": "rg1", "title": "Transa", "first-release-date": "1972-01-01"}]}`))
	})

	resp := get(t, app.URL+"/api/artists/abc/releases")

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	if got := resp.Header.Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q, want %q", got, "application/json")
	}

	var releases []musicbrainz.Release

	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		t.Fatal(err)
	}

	want := []musicbrainz.Release{{ID: "rg1", Title: "Transa", FirstReleaseDate: "1972-01-01"}}

	if !slices.Equal(releases, want) {
		t.Errorf("releases = %+v, want %+v", releases, want)
	}
}

func TestArtistReleasesReportsUpstreamFailure(t *testing.T) {
	app := newApp(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	resp := get(t, app.URL+"/api/artists/abc/releases")

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

	artists := followed(t, app)

	if artists == nil {
		t.Fatal("want an empty JSON array, got null")
	}

	if len(artists) != 0 {
		t.Errorf("want no artists, got %v", artists)
	}
}

func TestFollowArtistPersistsEveryField(t *testing.T) {
	app := newApp(t, noMusicBrainz(t))

	resp := put(t, app.URL+"/api/followed/abc",
		`{"name": "Caetano Veloso", "disambiguation": "Brazilian singer"}`)

	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusNoContent)
	}

	want := []store.Artist{{ID: "abc", Name: "Caetano Veloso", Disambiguation: "Brazilian singer"}}

	if artists := followed(t, app); !slices.Equal(artists, want) {
		t.Errorf("artists = %v, want %v", artists, want)
	}
}

func TestFollowArtistTwiceKeepsOneEntry(t *testing.T) {
	app := newApp(t, noMusicBrainz(t))

	for _, body := range []string{`{"name": "old name"}`, `{"name": "Caetano Veloso"}`} {
		if resp := put(t, app.URL+"/api/followed/abc", body); resp.StatusCode != http.StatusNoContent {
			t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusNoContent)
		}
	}

	want := []store.Artist{{ID: "abc", Name: "Caetano Veloso"}}

	if artists := followed(t, app); !slices.Equal(artists, want) {
		t.Errorf("artists = %v, want %v", artists, want)
	}
}

func TestFollowArtistRequiresAName(t *testing.T) {
	app := newApp(t, noMusicBrainz(t))

	for _, body := range []string{`{}`, `not json`} {
		if resp := put(t, app.URL+"/api/followed/abc", body); resp.StatusCode != http.StatusBadRequest {
			t.Errorf("status for %q = %d, want %d", body, resp.StatusCode, http.StatusBadRequest)
		}
	}

	if artists := followed(t, app); len(artists) != 0 {
		t.Errorf("want nothing saved, got %v", artists)
	}
}

func del(t *testing.T, url string) *http.Response {
	t.Helper()

	req, err := http.NewRequest(http.MethodDelete, url, nil)

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

func TestUnfollowArtistRemovesIt(t *testing.T) {
	app := newApp(t, noMusicBrainz(t))

	if resp := put(t, app.URL+"/api/followed/abc", `{"name": "Caetano Veloso"}`); resp.StatusCode != http.StatusNoContent {
		t.Fatalf("follow status = %d, want %d", resp.StatusCode, http.StatusNoContent)
	}

	if resp := del(t, app.URL+"/api/followed/abc"); resp.StatusCode != http.StatusNoContent {
		t.Fatalf("unfollow status = %d, want %d", resp.StatusCode, http.StatusNoContent)
	}

	if artists := followed(t, app); len(artists) != 0 {
		t.Errorf("want no artists, got %v", artists)
	}
}

func TestUnfollowUnknownArtistIsANoOp(t *testing.T) {
	app := newApp(t, noMusicBrainz(t))

	if resp := del(t, app.URL+"/api/followed/nope"); resp.StatusCode != http.StatusNoContent {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusNoContent)
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
		t.Errorf("200.html not served, got: %q", body)
	}
}

func TestServesTheFrontendOnItsOwnRoutes(t *testing.T) {
	app := newApp(t, noMusicBrainz(t))

	resp := get(t, app.URL+"/artist/abc")

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	body, err := io.ReadAll(resp.Body)

	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(string(body), "<title>guardarr</title>") {
		t.Errorf("200.html not served, got: %q", body)
	}
}

func TestServesFilesBesideThePage(t *testing.T) {
	app := newApp(t, noMusicBrainz(t))

	resp := get(t, app.URL+"/robots.txt")

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	body, err := io.ReadAll(resp.Body)

	if err != nil {
		t.Fatal(err)
	}

	if string(body) != "User-agent: *" {
		t.Errorf("body = %q, want robots.txt", body)
	}
}

func TestMissingAppFileIsNotFound(t *testing.T) {
	app := newApp(t, noMusicBrainz(t))

	resp := get(t, app.URL+"/_app/immutable/gone-abc12.js")

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}

func TestUnknownAPIPathIsNotFound(t *testing.T) {
	app := newApp(t, noMusicBrainz(t))

	for _, path := range []string{"/api/nope", "/api/artists/abc/nope"} {
		resp := get(t, app.URL+path)

		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("GET %s: status = %d, want %d", path, resp.StatusCode, http.StatusNotFound)
		}
	}
}

func TestUnknownAPIPathOpensTheAppInABrowser(t *testing.T) {
	app := newApp(t, noMusicBrainz(t))

	req, err := http.NewRequest(http.MethodGet, app.URL+"/api/nope", nil)

	if err != nil {
		t.Fatal(err)
	}

	// what a browser sends when it navigates to an address
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	resp, err := http.DefaultClient.Do(req)

	if err != nil {
		t.Fatal(err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}

	if got := resp.Header.Get("Content-Type"); !strings.HasPrefix(got, "text/html") {
		t.Errorf("Content-Type = %q, want text/html", got)
	}

	body, err := io.ReadAll(resp.Body)

	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(string(body), "<title>guardarr</title>") {
		t.Errorf("body = %q, want the app's page", body)
	}
}

func TestCachesHashedAssetsForever(t *testing.T) {
	app := newApp(t, noMusicBrainz(t))

	resp := get(t, app.URL+"/_app/immutable/entry-abc12.js")

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	want := "public, max-age=31536000, immutable"

	if got := resp.Header.Get("Cache-Control"); got != want {
		t.Errorf("Cache-Control = %q, want %q", got, want)
	}
}

func TestNeverCachesThePageItself(t *testing.T) {
	app := newApp(t, noMusicBrainz(t))

	resp := get(t, app.URL+"/")

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	if got := resp.Header.Get("Cache-Control"); got != "no-cache" {
		t.Errorf("Cache-Control = %q, want %q", got, "no-cache")
	}
}

func TestNeverCachesTheAppVersion(t *testing.T) {
	app := newApp(t, noMusicBrainz(t))

	resp := get(t, app.URL+"/_app/version.json")

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	if got := resp.Header.Get("Cache-Control"); got != "no-cache" {
		t.Errorf("Cache-Control = %q, want %q", got, "no-cache")
	}
}

// fakeSlskd serves handler as the slskd API and returns its address.
func fakeSlskd(t *testing.T, handler http.HandlerFunc) string {
	t.Helper()

	fake := httptest.NewServer(handler)
	t.Cleanup(fake.Close)

	return fake.URL
}

type slskdSettings struct {
	URL       string `json:"url"`
	HasAPIKey bool   `json:"hasApiKey"`
}

// savedSlskdSettings returns the slskd section of the saved settings.
func savedSlskdSettings(t *testing.T, app *httptest.Server) slskdSettings {
	t.Helper()

	resp := get(t, app.URL+"/api/settings")

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var settings struct {
		Slskd slskdSettings `json:"slskd"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&settings); err != nil {
		t.Fatal(err)
	}

	return settings.Slskd
}

// saveSettings saves the settings in body, failing the test unless the app
// takes them.
func saveSettings(t *testing.T, app *httptest.Server, body string) {
	t.Helper()

	if resp := patch(t, app.URL+"/api/settings", body); resp.StatusCode != http.StatusNoContent {
		t.Fatalf("saving %s: status = %d, want %d", body, resp.StatusCode, http.StatusNoContent)
	}
}

type slskdConnection struct {
	Configured bool   `json:"configured"`
	Reachable  bool   `json:"reachable"`
	LoggedIn   bool   `json:"loggedIn"`
	Username   string `json:"username"`
	Problem    string `json:"problem"`
}

func slskdStatus(t *testing.T, app *httptest.Server) slskdConnection {
	t.Helper()

	resp := get(t, app.URL+"/api/slskd/status")

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var connection slskdConnection

	if err := json.NewDecoder(resp.Body).Decode(&connection); err != nil {
		t.Fatal(err)
	}

	return connection
}

func TestSlskdSettingsStartEmpty(t *testing.T) {
	app := newApp(t, noMusicBrainz(t))

	if got := savedSlskdSettings(t, app); got != (slskdSettings{}) {
		t.Errorf("settings = %+v, want none", got)
	}

	if got := slskdStatus(t, app); got != (slskdConnection{}) {
		t.Errorf("status = %+v, want not configured", got)
	}
}

func TestSlskdSettingsNeverShowTheKey(t *testing.T) {
	app := newApp(t, noMusicBrainz(t))

	saveSettings(t, app, `{"slskd": {"url": " http://localhost:5030 ", "apiKey": "secret-key-0123456789"}}`)

	resp := get(t, app.URL+"/api/settings")
	body, err := io.ReadAll(resp.Body)

	if err != nil {
		t.Fatal(err)
	}

	if strings.Contains(string(body), "secret-key") {
		t.Errorf("the settings show the API key: %s", body)
	}

	want := slskdSettings{URL: "http://localhost:5030", HasAPIKey: true}

	if got := savedSlskdSettings(t, app); got != want {
		t.Errorf("settings = %+v, want %+v", got, want)
	}
}

func TestSavingSlskdSettingsKeepsTheKeyUnlessGivenOne(t *testing.T) {
	app := newApp(t, noMusicBrainz(t))

	saveSettings(t, app, `{"slskd": {"url": "http://localhost:5030", "apiKey": "secret-key-0123456789"}}`)
	saveSettings(t, app, `{"slskd": {"url": "http://nas.lan:5030"}}`)

	if got := savedSlskdSettings(t, app); !got.HasAPIKey {
		t.Error("changing only the address dropped the key")
	}

	saveSettings(t, app, `{"slskd": {"url": "http://nas.lan:5030", "apiKey": ""}}`)

	if got := savedSlskdSettings(t, app); got.HasAPIKey {
		t.Error("an empty key did not clear the saved one")
	}
}

func TestSavingSettingsLeavesOutSectionsAsTheyAre(t *testing.T) {
	app := newApp(t, noMusicBrainz(t))

	saveSettings(t, app, `{"slskd": {"url": "http://localhost:5030", "apiKey": "secret-key-0123456789"}}`)
	saveSettings(t, app, `{}`)

	want := slskdSettings{URL: "http://localhost:5030", HasAPIKey: true}

	if got := savedSlskdSettings(t, app); got != want {
		t.Errorf("settings = %+v, want %+v", got, want)
	}
}

func TestSavingSettingsRefusesBadOnes(t *testing.T) {
	app := newApp(t, noMusicBrainz(t))

	for _, body := range []string{
		`{"slskd": {"url": "localhost:5030"}}`,
		`{"slskd": {"url": "ftp://localhost:5030"}}`,
		`{"slskd": {"url": "http://"}}`,
		`{"slsk": {"url": "http://localhost:5030"}}`,
		`{"slskd": {"url": "http://localhost:5030", "key": "secret-key-0123456789"}}`,
		`not json`,
	} {
		if resp := patch(t, app.URL+"/api/settings", body); resp.StatusCode != http.StatusBadRequest {
			t.Errorf("saving %s: status = %d, want %d", body, resp.StatusCode, http.StatusBadRequest)
		}
	}

	if got := savedSlskdSettings(t, app); got != (slskdSettings{}) {
		t.Errorf("settings = %+v, want none after refused saves", got)
	}
}

func TestSavedSlskdSettingsTakeEffectAtOnce(t *testing.T) {
	address := fakeSlskd(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v0/server" {
			t.Errorf("path = %q, want %q", r.URL.Path, "/api/v0/server")
		}

		if got := r.Header.Get("X-API-Key"); got != "secret-key-0123456789" {
			t.Errorf("X-API-Key = %q, want the saved key", got)
		}

		w.Write([]byte(`{"state": "Connected, LoggedIn", "username": "guardarr", "isConnected": true, "isLoggedIn": true}`))
	})

	app := newApp(t, noMusicBrainz(t))

	saveSettings(t, app, `{"slskd": {"url": "`+address+`", "apiKey": "secret-key-0123456789"}}`)

	want := slskdConnection{Configured: true, Reachable: true, LoggedIn: true, Username: "guardarr"}

	if got := slskdStatus(t, app); got != want {
		t.Errorf("status = %+v, want %+v", got, want)
	}

	saveSettings(t, app, `{"slskd": {"url": ""}}`)

	if got := slskdStatus(t, app); got != (slskdConnection{}) {
		t.Errorf("status = %+v, want not configured once the address is cleared", got)
	}
}

func TestSlskdStatusSaysWhatIsWrong(t *testing.T) {
	closed := httptest.NewServer(http.NotFoundHandler())
	closed.Close()

	for _, tc := range []struct {
		name    string
		address string
		want    slskdConnection
	}{
		{
			name: "not logged in",
			address: fakeSlskd(t, func(w http.ResponseWriter, _ *http.Request) {
				w.Write([]byte(`{"state": "Disconnected", "isConnected": false, "isLoggedIn": false}`))
			}),
			want: slskdConnection{Configured: true, Reachable: true},
		},
		{
			name: "wrong key",
			address: fakeSlskd(t, func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusUnauthorized)
			}),
			want: slskdConnection{Configured: true, Problem: "slskd refused the API key."},
		},
		{
			name:    "no slskd api",
			address: fakeSlskd(t, http.NotFound),
			want:    slskdConnection{Configured: true, Problem: "There is no slskd API at this address."},
		},
		{
			name: "slskd failing",
			address: fakeSlskd(t, func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(`"The database is locked"`))
			}),
			want: slskdConnection{
				Configured: true,
				Problem:    "slskd answered 500 Internal Server Error: The database is locked",
			},
		},
		{
			name: "something else",
			address: fakeSlskd(t, func(w http.ResponseWriter, _ *http.Request) {
				w.Write([]byte(`<!doctype html><title>router login</title>`))
			}),
			want: slskdConnection{
				Configured: true,
				Problem:    "Something answered at this address, but not the way slskd does.",
			},
		},
		{
			name:    "nothing there",
			address: closed.URL,
			want:    slskdConnection{Configured: true, Problem: "Nothing answered at this address."},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			app := newApp(t, noMusicBrainz(t))

			saveSettings(t, app, `{"slskd": {"url": "`+tc.address+`", "apiKey": "secret-key-0123456789"}}`)

			if got := slskdStatus(t, app); got != tc.want {
				t.Errorf("status = %+v, want %+v", got, tc.want)
			}
		})
	}
}
