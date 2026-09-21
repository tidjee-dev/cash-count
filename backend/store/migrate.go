package store

import (
	"database/sql"
	"strings"
)

// migrateTextIDs converts pre-text-ID databases (integer AUTOINCREMENT ids,
// and the v1 counts CHECK without donation_urne) to the current schema.
// Tables are rebuilt from the shared column definitions with remapped ids;
// history is preserved. Fresh or current databases are untouched.
func migrateTextIDs(db *sql.DB) error {
	var idType string
	err := db.QueryRow(`SELECT type FROM pragma_table_info('denominations') WHERE name = 'id'`).Scan(&idType)
	if err == sql.ErrNoRows {
		return nil // fresh database; schema exec creates everything
	}
	if err != nil {
		return err
	}
	if strings.EqualFold(idType, "TEXT") {
		return nil // already migrated
	}

	type denomRow struct {
		id     int64
		label  string
		value  int64
		kind   string
		sort   int64
		active int
	}
	type countRow struct {
		id       int64
		created  string
		typ      string
		expected int64
		counted  int64
		drops    int64
		variance int64
		note     string
	}
	type itemRow struct {
		id      int64
		countID int64
		denomID int64
		qty     int64
		sub     int64
		label   string
		value   int64
	}
	type dropRow struct {
		id      int64
		countID int64
		amount  int64
		note    string
		created string
	}

	var denoms []denomRow
	rows, err := db.Query(`SELECT id, label, value_cents, kind, sort_order, active FROM denominations`)
	if err != nil {
		return err
	}
	for rows.Next() {
		var r denomRow
		if err := rows.Scan(&r.id, &r.label, &r.value, &r.kind, &r.sort, &r.active); err != nil {
			rows.Close()
			return err
		}
		denoms = append(denoms, r)
	}
	rows.Close()

	var counts []countRow
	crows, err := db.Query(`SELECT id, created_at, type, expected_cents, counted_cents, drops_cents, variance_cents, note FROM counts`)
	if err != nil {
		return err
	}
	for crows.Next() {
		var r countRow
		if err := crows.Scan(&r.id, &r.created, &r.typ, &r.expected, &r.counted, &r.drops, &r.variance, &r.note); err != nil {
			crows.Close()
			return err
		}
		counts = append(counts, r)
	}
	crows.Close()

	var items []itemRow
	irows, err := db.Query(`SELECT id, count_id, denomination_id, quantity, subtotal_cents, denomination_label, value_cents FROM count_items`)
	if err != nil {
		return err
	}
	for irows.Next() {
		var r itemRow
		if err := irows.Scan(&r.id, &r.countID, &r.denomID, &r.qty, &r.sub, &r.label, &r.value); err != nil {
			irows.Close()
			return err
		}
		items = append(items, r)
	}
	irows.Close()

	var drops []dropRow
	drows, err := db.Query(`SELECT id, count_id, amount_cents, note, created_at FROM count_drops`)
	if err != nil {
		return err
	}
	for drows.Next() {
		var r dropRow
		if err := drows.Scan(&r.id, &r.countID, &r.amount, &r.note, &r.created); err != nil {
			drows.Close()
			return err
		}
		drops = append(drops, r)
	}
	drows.Close()

	denomMap := map[int64]string{}
	for _, r := range denoms {
		id, err := NewID("den")
		if err != nil {
			return err
		}
		denomMap[r.id] = id
	}
	countMap := map[int64]string{}
	for _, r := range counts {
		id, err := NewID("cnt")
		if err != nil {
			return err
		}
		countMap[r.id] = id
	}

	// FKs must be off during the rebuild: dropping parents with enforcement
	// on would cascade-delete children. The pragma is a no-op inside a
	// transaction, so it is toggled outside of it (single-conn pool).
	if _, err := db.Exec(`PRAGMA foreign_keys = OFF`); err != nil {
		return err
	}
	defer db.Exec(`PRAGMA foreign_keys = ON`)
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, ddl := range []string{
		`CREATE TABLE denominations_new ` + colsDenominations,
		`CREATE TABLE counts_new ` + colsCounts,
		`CREATE TABLE count_items_new ` + colsCountItems,
		`CREATE TABLE count_drops_new ` + colsCountDrops,
	} {
		if _, err := tx.Exec(ddl); err != nil {
			return err
		}
	}
	for _, r := range denoms {
		if _, err := tx.Exec(`INSERT INTO denominations_new (id, label, value_cents, kind, sort_order, active) VALUES (?, ?, ?, ?, ?, ?)`,
			denomMap[r.id], r.label, r.value, r.kind, r.sort, r.active); err != nil {
			return err
		}
	}
	for _, r := range counts {
		if _, err := tx.Exec(`INSERT INTO counts_new (id, created_at, type, expected_cents, counted_cents, drops_cents, variance_cents, note) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			countMap[r.id], r.created, r.typ, r.expected, r.counted, r.drops, r.variance, r.note); err != nil {
			return err
		}
	}
	for _, r := range items {
		itemID, err := NewID("itm")
		if err != nil {
			return err
		}
		countID, ok1 := countMap[r.countID]
		denomID, ok2 := denomMap[r.denomID]
		if !ok1 || !ok2 {
			continue // orphaned row from an inconsistent DB; drop it
		}
		if _, err := tx.Exec(`INSERT INTO count_items_new (id, count_id, denomination_id, quantity, subtotal_cents, denomination_label, value_cents) VALUES (?, ?, ?, ?, ?, ?, ?)`,
			itemID, countID, denomID, r.qty, r.sub, r.label, r.value); err != nil {
			return err
		}
	}
	for _, r := range drops {
		dropID, err := NewID("drp")
		if err != nil {
			return err
		}
		countID, ok := countMap[r.countID]
		if !ok {
			continue
		}
		if _, err := tx.Exec(`INSERT INTO count_drops_new (id, count_id, amount_cents, note, created_at) VALUES (?, ?, ?, ?, ?)`,
			dropID, countID, r.amount, r.note, r.created); err != nil {
			return err
		}
	}
	for _, name := range []string{"count_items", "count_drops", "denominations", "counts"} {
		if _, err := tx.Exec(`DROP TABLE ` + name); err != nil {
			return err
		}
		if _, err := tx.Exec(`ALTER TABLE ` + name + `_new RENAME TO ` + name); err != nil {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}
