package store

import (
	"context"
	"path/filepath"
	"slices"
	"testing"
)

func testStore(t *testing.T) *Store {
	t.Helper()

	s, err := Open(filepath.Join(t.TempDir(), "test.db"))

	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { s.Close() })

	return s
}

func TestArtistIDsStartsEmpty(t *testing.T) {
	ids, err := testStore(t).ArtistIDs(context.Background())

	if err != nil {
		t.Fatal(err)
	}

	if len(ids) != 0 {
		t.Errorf("want no ids, got %v", ids)
	}
}

func TestSaveArtistPersistsIDs(t *testing.T) {
	s := testStore(t)

	for _, id := range []string{"abc", "def"} {
		if err := s.SaveArtist(context.Background(), id); err != nil {
			t.Fatal(err)
		}
	}

	ids, err := s.ArtistIDs(context.Background())

	if err != nil {
		t.Fatal(err)
	}

	if want := []string{"abc", "def"}; !slices.Equal(ids, want) {
		t.Errorf("ids = %v, want %v", ids, want)
	}
}

func TestSaveArtistIgnoresDuplicates(t *testing.T) {
	s := testStore(t)

	for range 2 {
		if err := s.SaveArtist(context.Background(), "abc"); err != nil {
			t.Fatal(err)
		}
	}

	ids, err := s.ArtistIDs(context.Background())

	if err != nil {
		t.Fatal(err)
	}

	if want := []string{"abc"}; !slices.Equal(ids, want) {
		t.Errorf("ids = %v, want %v", ids, want)
	}
}

func TestOpenReusesAnExistingDatabase(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")
	first, err := Open(path)

	if err != nil {
		t.Fatal(err)
	}

	if err := first.SaveArtist(context.Background(), "abc"); err != nil {
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

	ids, err := second.ArtistIDs(context.Background())

	if err != nil {
		t.Fatal(err)
	}

	if want := []string{"abc"}; !slices.Equal(ids, want) {
		t.Errorf("ids = %v, want %v", ids, want)
	}
}
