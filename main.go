package main

import (
	"context"
	"embed"
	"errors"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mateusguilherme/guardarr/internal/musicbrainz"
	"github.com/mateusguilherme/guardarr/internal/server"
	"github.com/mateusguilherme/guardarr/internal/store"
)

// The compiled frontend is embedded so the whole app ships as one binary.
// Run `npm run build` in frontend/ before building the Go binary.
//
//go:embed all:frontend/dist
var frontend embed.FS

func main() {
	addr := getEnv("GUARDARR_ADDR", ":8686")
	dbPath := getEnv("GUARDARR_DB", "guardarr.db")

	st, err := store.Open(dbPath)

	if err != nil {
		log.Fatal(err)
	}

	defer st.Close()

	dist, err := fs.Sub(frontend, "frontend/dist")

	if err != nil {
		log.Fatal(err)
	}

	srv := &http.Server{
		Addr:              addr,
		Handler:           server.New(musicbrainz.NewClient(), st, dist),
		ReadHeaderTimeout: 5 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)

	defer stop()

	shutdownDone := make(chan struct{})

	go func() {
		defer close(shutdownDone)

		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Printf("shutdown: %v", err)
		}
	}()

	log.Printf("guardarr listening on %s", addr)

	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}

	// ListenAndServe returns as soon as Shutdown starts, so wait for the
	// in-flight requests to finish before the deferred store close runs.
	<-shutdownDone
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return fallback
}
