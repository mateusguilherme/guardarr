// Package store persists guardarr's data in a SQLite database.
package store

import (
	"context"
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite" // registers the "sqlite" database/sql driver
)

const schema = `CREATE TABLE IF NOT EXISTS artists (id TEXT PRIMARY KEY) STRICT`

// Store is the SQLite-backed persistence layer.
type Store struct {
	db *sql.DB
}

// Open opens the database at path, creating the file and the schema as
// needed. The caller owns the returned Store and must Close it.
func Open(path string) (*Store, error) {
	// busy_timeout makes concurrent writers wait instead of failing
	// immediately with SQLITE_BUSY.
	db, err := sql.Open("sqlite", "file:"+path+"?_pragma=busy_timeout(5000)")

	if err != nil {
		return nil, err
	}

	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("create schema: %w", err)
	}

	return &Store{db: db}, nil
}

// Close releases the database.
func (s *Store) Close() error {
	return s.db.Close()
}

// SaveArtist records an artist id. Saving an id twice is a no-op.
func (s *Store) SaveArtist(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(
		ctx,
		`INSERT INTO artists (id) VALUES (?) ON CONFLICT DO NOTHING`,
		id,
	)

	return err
}

// ArtistIDs returns every saved artist id.
func (s *Store) ArtistIDs(ctx context.Context) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id FROM artists ORDER BY id`)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	ids := []string{}

	for rows.Next() {
		var id string

		if err := rows.Scan(&id); err != nil {
			return nil, err
		}

		ids = append(ids, id)
	}

	return ids, rows.Err()
}
