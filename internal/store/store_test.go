package store

import (
	"context"
	"database/sql"
	"path/filepath"
	"slices"
	"testing"
	"time"
)

var caetano = Artist{ID: "abc", Name: "Caetano Veloso"}

func testStore(t *testing.T) *Store {
	t.Helper()

	s, err := Open(filepath.Join(t.TempDir(), "test.db"))

	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { s.Close() })

	return s
}

func TestArtistsStartsEmpty(t *testing.T) {
	artists, err := testStore(t).Artists(context.Background())

	if err != nil {
		t.Fatal(err)
	}

	if len(artists) != 0 {
		t.Errorf("want no artists, got %v", artists)
	}
}

func TestSaveArtistPersistsEveryField(t *testing.T) {
	s := testStore(t)
	trio := Artist{ID: "def", Name: "Caetaneando", Disambiguation: "tribute trio"}

	for _, a := range []Artist{trio, caetano} {
		if err := s.SaveArtist(context.Background(), a); err != nil {
			t.Fatal(err)
		}
	}

	artists, err := s.Artists(context.Background())

	if err != nil {
		t.Fatal(err)
	}

	// Artists lists by name: "Caetaneando" sorts before "Caetano Veloso".
	if want := []Artist{trio, caetano}; !slices.Equal(artists, want) {
		t.Errorf("artists = %v, want %v", artists, want)
	}
}

func TestSaveArtistOverwritesDuplicates(t *testing.T) {
	s := testStore(t)

	if err := s.SaveArtist(context.Background(), Artist{ID: "abc", Name: "old name"}); err != nil {
		t.Fatal(err)
	}

	if err := s.SaveArtist(context.Background(), caetano); err != nil {
		t.Fatal(err)
	}

	artists, err := s.Artists(context.Background())

	if err != nil {
		t.Fatal(err)
	}

	if want := []Artist{caetano}; !slices.Equal(artists, want) {
		t.Errorf("artists = %v, want %v", artists, want)
	}
}

func TestDeleteArtistRemovesIt(t *testing.T) {
	s := testStore(t)

	if err := s.SaveArtist(context.Background(), caetano); err != nil {
		t.Fatal(err)
	}

	if err := s.DeleteArtist(context.Background(), caetano.ID); err != nil {
		t.Fatal(err)
	}

	artists, err := s.Artists(context.Background())

	if err != nil {
		t.Fatal(err)
	}

	if len(artists) != 0 {
		t.Errorf("want no artists, got %v", artists)
	}
}

func TestDeleteArtistIgnoresUnknownIDs(t *testing.T) {
	if err := testStore(t).DeleteArtist(context.Background(), "nope"); err != nil {
		t.Errorf("want a no-op, got %v", err)
	}
}

func TestReadsProceedDuringAWrite(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	conn, err := s.db.Conn(ctx)

	if err != nil {
		t.Fatal(err)
	}

	defer conn.Close()

	// Hold the write lock the way a commit in progress does.
	if _, err := conn.ExecContext(ctx, `BEGIN EXCLUSIVE`); err != nil {
		t.Fatal(err)
	}

	defer conn.ExecContext(ctx, `ROLLBACK`)

	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	if _, err := s.Artists(ctx); err != nil {
		t.Fatalf("listing while a write holds the lock: %v", err)
	}
}

func TestOpenReusesAnExistingDatabase(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")
	first, err := Open(path)

	if err != nil {
		t.Fatal(err)
	}

	if err := first.SaveArtist(context.Background(), caetano); err != nil {
		t.Fatal(err)
	}

	if err := first.Close(); err != nil {
		t.Fatal(err)
	}

	second, err := Open(path)

	if err != nil {
		t.Fatal(err)
	}

	defer second.Close()

	artists, err := second.Artists(context.Background())

	if err != nil {
		t.Fatal(err)
	}

	if want := []Artist{caetano}; !slices.Equal(artists, want) {
		t.Errorf("artists = %v, want %v", artists, want)
	}
}

func TestOpenMigratesTheFirstSchema(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")

	// Write a database the way the id-only schema did, before name and
	// disambiguation existed.
	db, err := sql.Open("sqlite", "file:"+path)

	if err != nil {
		t.Fatal(err)
	}

	if _, err := db.Exec(`CREATE TABLE artists (id TEXT PRIMARY KEY) STRICT`); err != nil {
		t.Fatal(err)
	}

	if _, err := db.Exec(`INSERT INTO artists (id) VALUES ('abc')`); err != nil {
		t.Fatal(err)
	}

	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	s, err := Open(path)

	if err != nil {
		t.Fatal(err)
	}

	defer s.Close()

	artists, err := s.Artists(context.Background())

	if err != nil {
		t.Fatal(err)
	}

	if want := []Artist{{ID: "abc"}}; !slices.Equal(artists, want) {
		t.Errorf("artists = %v, want %v", artists, want)
	}
}

func TestOpenRefusesANewerSchema(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")
	db, err := sql.Open("sqlite", "file:"+path)

	if err != nil {
		t.Fatal(err)
	}

	if _, err := db.Exec(`PRAGMA user_version = 99`); err != nil {
		t.Fatal(err)
	}

	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	if _, err := Open(path); err == nil {
		t.Fatal("want an error opening a database from a newer build, got nil")
	}
}
