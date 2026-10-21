# guardarr

Artist search on MusicBrainz. Go backend (pure stdlib, zero dependencies) with
a Svelte 5 + TypeScript frontend, shipped as a single binary.

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

The default port is `:8686`; override it with `GUARDARR_ADDR`.

## Develop

One command runs backend and frontend together, with hot reload on the UI:

```sh
cd frontend && npm run dev      # API on :8686, UI on :5173 with /api proxied
```
