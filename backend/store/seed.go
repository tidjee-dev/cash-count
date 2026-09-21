package store

import "database/sql"

// eurSeed is inserted on first launch when the denominations table is empty.
var eurSeed = []Denomination{
	{Label: "€500", ValueCents: 50000, Kind: KindBill, SortOrder: 0, Active: true},
	{Label: "€200", ValueCents: 20000, Kind: KindBill, SortOrder: 1, Active: true},
	{Label: "€100", ValueCents: 10000, Kind: KindBill, SortOrder: 2, Active: true},
	{Label: "€50", ValueCents: 5000, Kind: KindBill, SortOrder: 3, Active: true},
	{Label: "€20", ValueCents: 2000, Kind: KindBill, SortOrder: 4, Active: true},
	{Label: "€10", ValueCents: 1000, Kind: KindBill, SortOrder: 5, Active: true},
	{Label: "€5", ValueCents: 500, Kind: KindBill, SortOrder: 6, Active: true},
	{Label: "€2", ValueCents: 200, Kind: KindCoin, SortOrder: 7, Active: true},
	{Label: "€1", ValueCents: 100, Kind: KindCoin, SortOrder: 8, Active: true},
	{Label: "50c", ValueCents: 50, Kind: KindCoin, SortOrder: 9, Active: true},
	{Label: "20c", ValueCents: 20, Kind: KindCoin, SortOrder: 10, Active: true},
	{Label: "10c", ValueCents: 10, Kind: KindCoin, SortOrder: 11, Active: true},
	{Label: "5c", ValueCents: 5, Kind: KindCoin, SortOrder: 12, Active: true},
	{Label: "2c", ValueCents: 2, Kind: KindCoin, SortOrder: 13, Active: true},
	{Label: "1c", ValueCents: 1, Kind: KindCoin, SortOrder: 14, Active: true},
}

func seed(db *sql.DB) error {
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM denominations`).Scan(&n); err != nil {
		return err
	}
	if n == 0 {
		for _, d := range eurSeed {
			id, err := NewID("den")
			if err != nil {
				return err
			}
			if _, err := db.Exec(
				`INSERT INTO denominations (id, label, value_cents, kind, sort_order, active) VALUES (?, ?, ?, ?, ?, 1)`,
				id, d.Label, d.ValueCents, d.Kind, d.SortOrder,
			); err != nil {
				return err
			}
		}
	}
	if _, err := db.Exec(
		`INSERT INTO settings (id, store_name, currency_label, currency_symbol) VALUES (1, '', 'EUR', '€') ON CONFLICT(id) DO NOTHING`,
	); err != nil {
		return err
	}
	return nil
}
