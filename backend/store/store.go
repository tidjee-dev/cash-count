package store

import (
	"database/sql"
	"os"
	"path/filepath"
	"runtime"

	_ "modernc.org/sqlite"
)

// Store wraps the SQLite handle.
type Store struct {
	DB  *sql.DB
	Dir string
}

// AppDataDir returns the OS-appropriate data directory for the app.
func AppDataDir() (string, error) {
	if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
		return filepath.Join(xdg, "cashcount"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(home, "Library", "Application Support", "cashcount"), nil
	case "windows":
		if appdata := os.Getenv("APPDATA"); appdata != "" {
			return filepath.Join(appdata, "cashcount"), nil
		}
		return filepath.Join(home, "AppData", "Roaming", "cashcount"), nil
	default:
		return filepath.Join(home, ".local", "share", "cashcount"), nil
	}
}

// Open creates the data dir (plus the exports subdir), opens (or creates)
// the SQLite file at the data dir root, applies the schema and seeds
// defaults on first launch.
func Open(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Join(dir, "exports"), 0o755); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", filepath.Join(dir, "cashcount.db"))
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(`PRAGMA foreign_keys = ON`); err != nil {
		db.Close()
		return nil, err
	}
	if _, err := db.Exec(`PRAGMA journal_mode = WAL`); err != nil {
		db.Close()
		return nil, err
	}
	// Migration first: it rebuilds old tables (dropping their indexes), and
	// the schema exec below recreates anything missing.
	if err := migrateTextIDs(db); err != nil {
		db.Close()
		return nil, err
	}
	if _, err := db.Exec(schemaV1); err != nil {
		db.Close()
		return nil, err
	}
	if err := seed(db); err != nil {
		db.Close()
		return nil, err
	}
	return &Store{DB: db, Dir: dir}, nil
}

func (s *Store) Close() error {
	return s.DB.Close()
}
