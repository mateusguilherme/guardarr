# guardarr

Artist search on MusicBrainz. Go backend with a SvelteKit + TypeScript
frontend, built as a single-page app and shipped inside a single binary. Data
lives in a SQLite database through a pure-Go driver, so building still needs
no CGO.

## Run

The frontend build is embedded into the Go binary, so build it first:

```sh
cd frontend && npm install && npm run build && cd ..
go run .
```

Then open <http://localhost:8686>, or hit the API directly:

```sh
curl 'localhost:8686/api/artists?q=caetano+veloso'
```

The default port is `:8686`; override it with `GUARDARR_ADDR`. The database
is created as `guardarr.db` in the working directory; override the path with
`GUARDARR_DB`.

Downloads go through [slskd](https://github.com/slskd/slskd), which is
optional. To connect it, set `GUARDARR_SLSKD_URL` to its web address,
including any URL base (e.g. `http://localhost:5030`), and
`GUARDARR_SLSKD_API_KEY` to one of its API keys. `GET /api/slskd/status`
tells whether guardarr reaches it and whether it is logged in to Soulseek.

## Test

```sh
go test -race ./...             # backend
cd frontend && npm test         # frontend
cd frontend && npm run lint     # frontend lint
```

A pre-push hook runs the suites and the linter; enable it once per clone with:

```sh
git config core.hooksPath .githooks
```

## Develop

One command runs backend and frontend together, with hot reload on the UI:

```sh
cd frontend && npm run dev      # API on :8686, UI on :5173 with /api proxied
```
