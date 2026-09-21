package store

// Entity tables use custom text IDs (see NewID) instead of autoincrement
// integers, so IDs are opaque and portable. The settings singleton keeps its
// fixed integer id; it is never exposed as an entity reference.
//
// Column definitions are shared between the fresh-install schema and the
// integer-to-text migration so both always agree.
const colsDenominations = `(
	id TEXT PRIMARY KEY,
	label TEXT NOT NULL,
	value_cents INTEGER NOT NULL CHECK (value_cents > 0),
	kind TEXT NOT NULL CHECK (kind IN ('bill','coin')),
	sort_order INTEGER NOT NULL DEFAULT 0,
	active INTEGER NOT NULL DEFAULT 1
)`

const colsCounts = `(
	id TEXT PRIMARY KEY,
	created_at TEXT NOT NULL DEFAULT (datetime('now')),
	type TEXT NOT NULL CHECK (type IN ('shift_open','shift_close','audit','donation_urne')),
	expected_cents INTEGER NOT NULL DEFAULT 0,
	counted_cents INTEGER NOT NULL DEFAULT 0,
	drops_cents INTEGER NOT NULL DEFAULT 0,
	variance_cents INTEGER NOT NULL DEFAULT 0,
	note TEXT NOT NULL DEFAULT ''
)`

const colsCountItems = `(
	id TEXT PRIMARY KEY,
	count_id TEXT NOT NULL REFERENCES counts(id) ON DELETE CASCADE,
	denomination_id TEXT NOT NULL REFERENCES denominations(id),
	quantity INTEGER NOT NULL CHECK (quantity >= 0),
	subtotal_cents INTEGER NOT NULL,
	denomination_label TEXT NOT NULL DEFAULT '',
	value_cents INTEGER NOT NULL DEFAULT 0
)`

const colsCountDrops = `(
	id TEXT PRIMARY KEY,
	count_id TEXT NOT NULL REFERENCES counts(id) ON DELETE CASCADE,
	amount_cents INTEGER NOT NULL CHECK (amount_cents > 0),
	note TEXT NOT NULL DEFAULT '',
	created_at TEXT NOT NULL DEFAULT (datetime('now'))
)`

// schemaV1 creates all tables. Totals are denormalized onto counts so a count
// row stays an immutable snapshot even if denominations change later.
const schemaV1 = `
CREATE TABLE IF NOT EXISTS settings (
	id INTEGER PRIMARY KEY CHECK (id = 1),
	store_name TEXT NOT NULL DEFAULT '',
	currency_label TEXT NOT NULL DEFAULT 'USD',
	currency_symbol TEXT NOT NULL DEFAULT '$'
);

CREATE TABLE IF NOT EXISTS denominations ` + colsDenominations + `;

CREATE TABLE IF NOT EXISTS counts ` + colsCounts + `;

CREATE TABLE IF NOT EXISTS count_items ` + colsCountItems + `;
CREATE INDEX IF NOT EXISTS idx_count_items_count ON count_items(count_id);

CREATE TABLE IF NOT EXISTS count_drops ` + colsCountDrops + `;
CREATE INDEX IF NOT EXISTS idx_count_drops_count ON count_drops(count_id);
`
