# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```sh
cd frontend && npm install && npm run build   # must run before any Go build: main.go embeds frontend/dist
go run .                                      # app on :8686 (GUARDARR_ADDR, GUARDARR_DB override port and db path)
cd frontend && npm run dev                    # Go backend on :8686 + Vite on :5173 with /api proxied

go test -race ./...                           # backend
go test ./internal/server -run TestSlskdStatusSaysWhatIsWrong   # one Go test
cd frontend && npm test                       # frontend (vitest, jsdom)
cd frontend && npx vitest run src/lib/SearchBar.test.ts -t "goes home"   # one frontend test
cd frontend && npm run check                  # svelte-check / types
cd frontend && npm run lint                   # eslint
```

- Without `frontend/dist`, `go build` fails, and so does `go test ./...` in the root package (`pattern all:frontend/dist: no matching files found`).
- `npm run dev` runs `go run ..` from `frontend/`, so the development database is `frontend/guardarr.db`, not the one at the repo root.
- `.githooks/pre-push` runs the Go tests, the lint and the frontend tests. Enable it with `git config core.hooksPath .githooks`.

## Architecture

guardarr automates downloading music from the user's favorite artists: they follow artists (found on MusicBrainz) and guardarr fetches their releases through slskd. It ships as one Go binary with the SvelteKit build embedded and data in SQLite (`modernc.org/sqlite`, no CGO).

### Backend (`main.go`, `internal/`)

**`main.go`** opens the store and reads the saved slskd settings. It then builds `server.New(server.Deps{...})` and shuts down gracefully, waiting for in-flight requests before the store closes.

**`internal/server`** registers every route on one `http.ServeMux` with Go 1.22+ method patterns.
- Everything under `/api/` is the API.
- `GET /` serves the static build. A path that names no file gets `200.html` (the SPA fallback), and `/_app/immutable/*` is cached forever.
- An unknown `/api/...` path is a 404. A browser (`Accept: text/html`) gets the app's not-found page with status 404.
- `GET /api/` must keep its method: a method-less `/api/` conflicts with `GET /` and panics at startup.

**Settings** are generic.
- `GET /api/settings` returns every section.
- `PATCH /api/settings` saves only the sections sent and rejects unknown fields.
- The slskd API key is never returned, only `hasApiKey`.
- Saving calls `slskd.Client.Configure`, so a change takes effect without a restart.

**`internal/musicbrainz`** throttles all requests to one every 1.1 s, which is the MusicBrainz rate limit. A slot freed by a cancelled request goes back to the queue. When MusicBrainz is busy it answers 503 even to clients within the limit, so a 503 is retried after 1s, 2s and 4s. Artist lookups and releases are cached for an hour, and errors are never cached. Releases come from a release-group *search*, not a browse, so they can be filtered to official albums. The artist id goes into that search query, so `Artist` and `ArtistReleases` refuse anything that isn't a lowercase UUID with `ErrNotFound`, which the routes answer as 404.

**`internal/slskd`** is a client for slskd's `/api/v0`, authenticated with `X-API-Key`.
- It is guarded by an RWMutex and reconfigurable at runtime.
- With no URL it returns `ErrNotConfigured`.
- A non-2xx answer returns `*StatusError`. slskd's error bodies are JSON-quoted strings, and `StatusError` carries them unquoted.
- Downloads use the batches endpoint, because the per-user enqueue endpoint is obsolete.

**`internal/store`** runs `migrations` in order, tracked by `PRAGMA user_version`. Only append to that slice; never edit an applied migration. The database uses the WAL journal.

### Frontend (`frontend/`)

- **Rendering:** SvelteKit in SPA mode, with adapter-static to `dist/` and `ssr = false` in `src/routes/+layout.ts`, since the Go server has no JS runtime. Svelte 5 runes throughout.
- **Imports:** use `#lib/*` (package.json `imports`) with the file extension, e.g. `#lib/api.ts`.
- **API calls:** all of them go through `request()` in `src/lib/api.ts`. It checks `response.ok`, decodes the JSON, and returns `undefined` on 204. Load functions pass SvelteKit's `fetch` in.
- **Shared state:** `AppState` (`src/lib/app-state.svelte.ts`) holds the followed artists and the search. The root layout creates it once and provides it through context, so it survives navigation; other state stays local to its component.
- **Search:** the search bar (`src/lib/SearchBar.svelte`) lives in the layout's sticky header. Results only render on the home, so a search started on another page `goto`s `/` first.
- **Error page:** `+error.svelte` covers not-found and other errors, through the `Lost` component. The artist page also renders `Lost` when the lookup answers 404.
- **Artist route:** it is `artist/[id=mbid]`. The `mbid` matcher lives in `src/params.ts`, which uses SvelteKit 3's `defineParams` (not v2's `src/params/*.ts` files), so a malformed id never reaches the page.
- **Styling:** global styles live in `src/app.css`, with `light-dark()` color tokens, and `--page-width` sets the shared column width.
- **Tests:**
  - Tests sit next to the code (`*.test.ts`) and use @testing-library/svelte.
  - Components that read `AppState` render with `{ context: appStateContext(app) }`.
  - Tests that touch `$app/state` or `$app/navigation` mock them with `vi.mock` and `vi.hoisted`.
  - `fetch` is stubbed with `vi.stubGlobal`.

## Conventions

- **Commit titles:** lowercase imperative English, with no prefix and no body, e.g. `answer unknown api paths with a 404`.
- **Braces:** in TypeScript and Svelte, every `if`/`else` body has them, even one-liners. ESLint enforces `curly: all`.
- **Go spacing:** a blank line separates an assignment from the `if err != nil` that checks it. Paired calls such as `mu.Lock()` / `defer mu.Unlock()` stay on adjacent lines.
- **Test names:** both Go and frontend tests are named as sentences describing behavior (`TestSavingSettingsRefusesBadOnes`, `it("goes home to show a search started on another page")`). Comments explain why, not what.
