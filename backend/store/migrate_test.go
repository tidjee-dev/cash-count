package store

import (
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

// oldIntSchema mirrors the pre-text-ID database: integer AUTOINCREMENT ids
// and a counts CHECK without donation_urne.
const oldIntSchema = `
CREATE TABLE settings (
	id INTEGER PRIMARY KEY CHECK (id = 1),
	store_name TEXT NOT NULL DEFAULT '',
	currency_label TEXT NOT NULL DEFAULT 'USD',
	currency_symbol TEXT NOT NULL DEFAULT '$'
);
CREATE TABLE denominations (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	label TEXT NOT NULL,
	value_cents INTEGER NOT NULL CHECK (value_cents > 0),
	kind TEXT NOT NULL CHECK (kind IN ('bill','coin')),
	sort_order INTEGER NOT NULL DEFAULT 0,
	active INTEGER NOT NULL DEFAULT 1
);
CREATE TABLE counts (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	created_at TEXT NOT NULL DEFAULT (datetime('now')),
	type TEXT NOT NULL CHECK (type IN ('shift_open','shift_close','audit')),
	expected_cents INTEGER NOT NULL DEFAULT 0,
	counted_cents INTEGER NOT NULL DEFAULT 0,
	drops_cents INTEGER NOT NULL DEFAULT 0,
	variance_cents INTEGER NOT NULL DEFAULT 0,
	note TEXT NOT NULL DEFAULT ''
);
CREATE TABLE count_items (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	count_id INTEGER NOT NULL REFERENCES counts(id) ON DELETE CASCADE,
	denomination_id INTEGER NOT NULL REFERENCES denominations(id),
	quantity INTEGER NOT NULL CHECK (quantity >= 0),
	subtotal_cents INTEGER NOT NULL,
	denomination_label TEXT NOT NULL DEFAULT '',
	value_cents INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX idx_count_items_count ON count_items(count_id);
CREATE TABLE count_drops (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	count_id INTEGER NOT NULL REFERENCES counts(id) ON DELETE CASCADE,
	amount_cents INTEGER NOT NULL CHECK (amount_cents > 0),
	note TEXT NOT NULL DEFAULT '',
	created_at TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX idx_count_drops_count ON count_drops(count_id);
`

// TestMigrateTextIDs builds an integer-ID database, opens it through
// Store.Open and verifies history survives with remapped text IDs.
func TestMigrateTextIDs(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cashcount.db")

	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(oldIntSchema); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO settings (id, store_name, currency_label, currency_symbol) VALUES (1, 'Test', 'EUR', '€')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO denominations (label, value_cents, kind, sort_order, active) VALUES ('€20', 2000, 'bill', 0, 1)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO counts (created_at, type, expected_cents, counted_cents, drops_cents, variance_cents, note)
		VALUES ('2026-01-01T10:00:00Z', 'shift_close', 2000, 2000, 0, 0, 'old')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO count_items (count_id, denomination_id, quantity, subtotal_cents, denomination_label, value_cents)
		VALUES (1, 1, 1, 2000, '€20', 2000)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO count_drops (count_id, amount_cents, note, created_at)
		VALUES (1, 500, 'safe', '2026-01-01T10:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	db.Close()

	st, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	var id, typ, note string
	if err := st.DB.QueryRow(`SELECT id, type, note FROM counts`).Scan(&id, &typ, &note); err != nil {
		t.Fatalf("old row lost: %v", err)
	}
	if typ != "shift_close" || note != "old" {
		t.Fatalf("old row changed: %s %s", typ, note)
	}
	if len(id) < 5 || id[:4] != "cnt_" {
		t.Fatalf("count id not remapped to text: %q", id)
	}
	var items, drops int
	if err := st.DB.QueryRow(`SELECT COUNT(*) FROM count_items WHERE count_id = ?`, id).Scan(&items); err != nil || items != 1 {
		t.Fatalf("old items lost: n=%d err=%v", items, err)
	}
	if err := st.DB.QueryRow(`SELECT COUNT(*) FROM count_drops WHERE count_id = ?`, id).Scan(&drops); err != nil || drops != 1 {
		t.Fatalf("old drops lost: n=%d err=%v", drops, err)
	}
	var denomID string
	if err := st.DB.QueryRow(`SELECT denomination_id FROM count_items WHERE count_id = ?`, id).Scan(&denomID); err != nil {
		t.Fatal(err)
	}
	var denomLabel string
	if err := st.DB.QueryRow(`SELECT label FROM denominations WHERE id = ?`, denomID).Scan(&denomLabel); err != nil || denomLabel != "€20" {
		t.Fatalf("denom remap broken: %q err=%v", denomLabel, err)
	}
	var idx int
	if err := st.DB.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type = 'index' AND name IN ('idx_count_items_count','idx_count_drops_count')`).Scan(&idx); err != nil || idx != 2 {
		t.Fatalf("indexes not recreated: n=%d err=%v", idx, err)
	}
	if _, err := st.DB.Exec(`INSERT INTO counts (id, created_at, type, expected_cents, counted_cents, drops_cents, variance_cents, note)
		VALUES (?, '2026-01-02T10:00:00Z', 'donation_urne', 0, 150, 0, 150, 'urne')`, MustID("cnt")); err != nil {
		t.Fatalf("donation_urne rejected: %v", err)
	}

	// Reopen: migration must be a no-op the second time.
	st.Close()
	st2, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer st2.Close()
	var n int
	if err := st2.DB.QueryRow(`SELECT COUNT(*) FROM counts`).Scan(&n); err != nil || n != 2 {
		t.Fatalf("rows after reopen: n=%d err=%v", n, err)
	}
}
