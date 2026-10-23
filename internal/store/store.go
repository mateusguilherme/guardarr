// Package store persists guardarr's data in a SQLite database.
package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	_ "modernc.org/sqlite" // registers the "sqlite" database/sql driver
)

// migrations run in order on Open; the database's user_version pragma
// records how many have been applied, so each one runs exactly once.
var migrations = []string{
	`CREATE TABLE IF NOT EXISTS artists (id TEXT PRIMARY KEY) STRICT`,
	`ALTER TABLE artists ADD COLUMN name TEXT NOT NULL DEFAULT ''`,
	`ALTER TABLE artists ADD COLUMN disambiguation TEXT NOT NULL DEFAULT ''`,
	// One row at most: there is one slskd to reach.
	`CREATE TABLE slskd_settings (
		id INTEGER PRIMARY KEY CHECK (id = 1),
		url TEXT NOT NULL,
		api_key TEXT NOT NULL
	) STRICT`,
}

// Artist is a followed artist as stored locally. It mirrors the fields
// guardarr keeps from MusicBrainz.
type Artist struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Disambiguation string `json:"disambiguation,omitempty"`
}

// SlskdSettings is how guardarr reaches slskd. An empty URL means slskd is
// not set up.
type SlskdSettings struct {
	URL    string
	APIKey string
}

// Store is the SQLite-backed persistence layer.
type Store struct {
	db *sql.DB
}

// Open opens the database at path, creating the file and migrating the
// schema as needed. The caller owns the returned Store and must Close it.
func Open(path string) (*Store, error) {
	// busy_timeout makes concurrent writers wait instead of failing
	// immediately with SQLITE_BUSY. WAL lets reads proceed while a write
	// commits; under the default rollback journal a steady run of writes
	// can starve readers until busy_timeout gives up. WAL does mean extra
	// -wal and -shm files beside the database, and no network filesystems.
	db, err := sql.Open("sqlite", "file:"+path+"?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)")

	if err != nil {
		return nil, err
	}

	if err := migrate(db); err != nil {
		db.Close()
		return nil, err
	}

	return &Store{db: db}, nil
}

func migrate(db *sql.DB) error {
	var version int

	if err := db.QueryRow(`PRAGMA user_version`).Scan(&version); err != nil {
		return fmt.Errorf("read schema version: %w", err)
	}

	if version > len(migrations) {
		return fmt.Errorf("database schema version %d is newer than this build supports", version)
	}

	for i, m := range migrations[version:] {
		if _, err := db.Exec(m); err != nil {
			return fmt.Errorf("migration %d: %w", version+i+1, err)
		}

		if _, err := db.Exec(fmt.Sprintf(`PRAGMA user_version = %d`, version+i+1)); err != nil {
			return fmt.Errorf("record schema version %d: %w", version+i+1, err)
		}
	}

	return nil
}

// Close releases the database.
func (s *Store) Close() error {
	return s.db.Close()
}

// SaveArtist records an artist. Saving an id again overwrites its data,
// so a follow always stores the freshest name and disambiguation.
func (s *Store) SaveArtist(ctx context.Context, a Artist) error {
	_, err := s.db.ExecContext(
		ctx,
		`INSERT INTO artists (id, name, disambiguation) VALUES (?, ?, ?)
		 ON CONFLICT (id) DO UPDATE SET
		   name = excluded.name,
		   disambiguation = excluded.disambiguation`,
		a.ID, a.Name, a.Disambiguation,
	)

	return err
}

// DeleteArtist removes an artist. Deleting an unknown id is a no-op.
func (s *Store) DeleteArtist(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM artists WHERE id = ?`, id)

	return err
}

// Artists returns every saved artist, ordered by name.
func (s *Store) Artists(ctx context.Context) ([]Artist, error) {
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT id, name, disambiguation FROM artists ORDER BY name, id`,
	)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	artists := []Artist{}

	for rows.Next() {
		var a Artist

		if err := rows.Scan(&a.ID, &a.Name, &a.Disambiguation); err != nil {
			return nil, err
		}

		artists = append(artists, a)
	}

	return artists, rows.Err()
}

// SlskdSettings returns the saved slskd settings, empty until first saved.
func (s *Store) SlskdSettings(ctx context.Context) (SlskdSettings, error) {
	var settings SlskdSettings

	err := s.db.QueryRowContext(ctx, `SELECT url, api_key FROM slskd_settings`).
		Scan(&settings.URL, &settings.APIKey)

	if errors.Is(err, sql.ErrNoRows) {
		return SlskdSettings{}, nil
	}

	return settings, err
}

// SaveSlskdSettings replaces the saved slskd settings.
func (s *Store) SaveSlskdSettings(ctx context.Context, settings SlskdSettings) error {
	_, err := s.db.ExecContext(
		ctx,
		`INSERT INTO slskd_settings (id, url, api_key) VALUES (1, ?, ?)
		 ON CONFLICT (id) DO UPDATE SET
		   url = excluded.url,
		   api_key = excluded.api_key`,
		settings.URL, settings.APIKey,
	)

	return err
}
