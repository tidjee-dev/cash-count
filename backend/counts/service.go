package counts

import (
	"bytes"
	"database/sql"
	"encoding/csv"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"time"

	"github.com/tidjee-dev/cash-count/backend/store"
)

// Service records counts, history and CSV exports.
type Service struct {
	Store *store.Store
}

func New(s *store.Store) *Service { return &Service{Store: s} }

type CountItemInput struct {
	DenominationID string `json:"denomination_id"`
	Quantity       int64  `json:"quantity"`
}

type CountDropInput struct {
	AmountCents int64  `json:"amount_cents"`
	Note        string `json:"note"`
}

type CreateCountInput struct {
	Type          string           `json:"type"`
	ExpectedCents int64            `json:"expected_cents"`
	Note          string           `json:"note"`
	Items         []CountItemInput `json:"items"`
	Drops         []CountDropInput `json:"drops"`
}

type CountFilter struct {
	Type   string `json:"type"`
	From   string `json:"from"`
	To     string `json:"to"`
	Limit  int    `json:"limit"`
	Offset int    `json:"offset"`
}

func parseTime(v string) time.Time {
	for _, layout := range []string{time.RFC3339, "2006-01-02 15:04:05", "2006-01-02"} {
		if t, err := time.Parse(layout, v); err == nil {
			return t
		}
	}
	return time.Now().UTC()
}

func scanCount(row interface {
	Scan(dest ...any) error
}, c *store.Count) error {
	var created string
	err := row.Scan(&c.ID, &created, &c.Type, &c.ExpectedCents, &c.CountedCents, &c.DropsCents, &c.VarianceCents, &c.Note)
	if err != nil {
		return err
	}
	c.CreatedAt = parseTime(created)
	return nil
}

const countCols = `id, created_at, type, expected_cents, counted_cents, drops_cents, variance_cents, note`

// CreateCount validates the input, computes totals and persists count + items
// + drops in a single transaction. Variance = counted + drops - expected.
func (s *Service) CreateCount(in CreateCountInput) (store.CountDetail, error) {
	if !store.ValidCountType(in.Type) {
		return store.CountDetail{}, errors.New("type must be shift_open, shift_close, audit or donation_urne")
	}
	if in.ExpectedCents < 0 {
		return store.CountDetail{}, errors.New("expected amount cannot be negative")
	}
	if len(in.Items) == 0 && len(in.Drops) == 0 {
		return store.CountDetail{}, errors.New("add at least one denomination quantity or drop")
	}

	tx, err := s.Store.DB.Begin()
	if err != nil {
		return store.CountDetail{}, err
	}
	defer tx.Rollback()

	type resolved struct {
		denomID string
		label   string
		value   int64
		qty     int64
	}
	var lines []resolved
	var counted int64
	seen := map[string]bool{}
	for _, it := range in.Items {
		if it.Quantity < 0 {
			return store.CountDetail{}, errors.New("quantities cannot be negative")
		}
		if it.Quantity == 0 || seen[it.DenominationID] {
			continue
		}
		seen[it.DenominationID] = true
		var label, kind string
		var value int64
		var active int
		err := tx.QueryRow(
			`SELECT label, value_cents, kind, active FROM denominations WHERE id = ?`, it.DenominationID,
		).Scan(&label, &value, &kind, &active)
		if err == sql.ErrNoRows {
			return store.CountDetail{}, fmt.Errorf("unknown denomination id %s", it.DenominationID)
		}
		if err != nil {
			return store.CountDetail{}, err
		}
		if active == 0 {
			return store.CountDetail{}, fmt.Errorf("denomination %q is disabled", label)
		}
		lines = append(lines, resolved{it.DenominationID, label, value, it.Quantity})
		counted += value * it.Quantity
	}
	if len(lines) == 0 {
		return store.CountDetail{}, errors.New("add at least one denomination quantity")
	}

	var dropsTotal int64
	for _, d := range in.Drops {
		if d.AmountCents <= 0 {
			return store.CountDetail{}, errors.New("drop amounts must be positive")
		}
		dropsTotal += d.AmountCents
	}

	variance := counted + dropsTotal - in.ExpectedCents
	now := time.Now().UTC().Format(time.RFC3339)

	countID, err := store.NewID("cnt")
	if err != nil {
		return store.CountDetail{}, err
	}
	if _, err := tx.Exec(
		`INSERT INTO counts (id, created_at, type, expected_cents, counted_cents, drops_cents, variance_cents, note) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		countID, now, in.Type, in.ExpectedCents, counted, dropsTotal, variance, in.Note,
	); err != nil {
		return store.CountDetail{}, err
	}

	detail := store.CountDetail{
		Count: store.Count{
			ID: countID, CreatedAt: parseTime(now), Type: in.Type,
			ExpectedCents: in.ExpectedCents, CountedCents: counted,
			DropsCents: dropsTotal, VarianceCents: variance, Note: in.Note,
		},
		Items: []store.CountItem{},
		Drops: []store.CountDrop{},
	}
	for _, l := range lines {
		sub := l.value * l.qty
		itemID, err := store.NewID("itm")
		if err != nil {
			return store.CountDetail{}, err
		}
		if _, err := tx.Exec(
			`INSERT INTO count_items (id, count_id, denomination_id, quantity, subtotal_cents, denomination_label, value_cents) VALUES (?, ?, ?, ?, ?, ?, ?)`,
			itemID, countID, l.denomID, l.qty, sub, l.label, l.value,
		); err != nil {
			return store.CountDetail{}, err
		}
		detail.Items = append(detail.Items, store.CountItem{
			ID: itemID, CountID: countID, DenominationID: l.denomID,
			DenominationLabel: l.label, ValueCents: l.value, Quantity: l.qty, SubtotalCents: sub,
		})
	}
	for _, d := range in.Drops {
		dropID, err := store.NewID("drp")
		if err != nil {
			return store.CountDetail{}, err
		}
		if _, err := tx.Exec(
			`INSERT INTO count_drops (id, count_id, amount_cents, note, created_at) VALUES (?, ?, ?, ?, ?)`,
			dropID, countID, d.AmountCents, d.Note, now,
		); err != nil {
			return store.CountDetail{}, err
		}
		detail.Drops = append(detail.Drops, store.CountDrop{
			ID: dropID, CountID: countID, AmountCents: d.AmountCents, Note: d.Note, CreatedAt: parseTime(now),
		})
	}
	if err := tx.Commit(); err != nil {
		return store.CountDetail{}, err
	}
	// Return items largest-first so the detail view matches the count screen
	// (bills → coins), regardless of input order.
	sort.SliceStable(detail.Items, func(i, j int) bool {
		if detail.Items[i].ValueCents != detail.Items[j].ValueCents {
			return detail.Items[i].ValueCents > detail.Items[j].ValueCents
		}
		return detail.Items[i].DenominationLabel < detail.Items[j].DenominationLabel
	})
	return detail, nil
}

// ListCounts returns count summaries, newest first.
func (s *Service) ListCounts(f CountFilter) ([]store.Count, error) {
	q := `SELECT ` + countCols + ` FROM counts WHERE 1 = 1`
	var args []any
	if store.ValidCountType(f.Type) {
		q += ` AND type = ?`
		args = append(args, f.Type)
	}
	if f.From != "" {
		q += ` AND created_at >= ?`
		args = append(args, f.From)
	}
	if f.To != "" {
		q += ` AND created_at <= ?`
		args = append(args, f.To)
	}
	q += ` ORDER BY created_at DESC, id DESC`
	if f.Limit > 0 {
		q += ` LIMIT ?`
		args = append(args, f.Limit)
		if f.Offset > 0 {
			q += ` OFFSET ?`
			args = append(args, f.Offset)
		}
	}
	rows, err := s.Store.DB.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []store.Count{}
	for rows.Next() {
		var c store.Count
		if err := scanCount(rows, &c); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// GetCount returns the full immutable detail of one count.
func (s *Service) GetCount(id string) (store.CountDetail, error) {
	var d store.CountDetail
	if err := scanCount(
		s.Store.DB.QueryRow(`SELECT `+countCols+` FROM counts WHERE id = ?`, id), &d.Count,
	); err != nil {
		if err == sql.ErrNoRows {
			return d, errors.New("count not found")
		}
		return d, err
	}
	d.Items = []store.CountItem{}
	rows, err := s.Store.DB.Query(
		`SELECT id, count_id, denomination_id, quantity, subtotal_cents, denomination_label, value_cents FROM count_items WHERE count_id = ? ORDER BY value_cents DESC, denomination_label, id`,
		id,
	)
	if err != nil {
		return d, err
	}
	for rows.Next() {
		var it store.CountItem
		if err := rows.Scan(&it.ID, &it.CountID, &it.DenominationID, &it.Quantity, &it.SubtotalCents, &it.DenominationLabel, &it.ValueCents); err != nil {
			rows.Close()
			return d, err
		}
		d.Items = append(d.Items, it)
	}
	rows.Close()
	d.Drops = []store.CountDrop{}
	drows, err := s.Store.DB.Query(
		`SELECT id, count_id, amount_cents, note, created_at FROM count_drops WHERE count_id = ? ORDER BY created_at, id`,
		id,
	)
	if err != nil {
		return d, err
	}
	defer drows.Close()
	for drows.Next() {
		var dr store.CountDrop
		var created string
		if err := drows.Scan(&dr.ID, &dr.CountID, &dr.AmountCents, &dr.Note, &created); err != nil {
			return d, err
		}
		dr.CreatedAt = parseTime(created)
		d.Drops = append(d.Drops, dr)
	}
	return d, drows.Err()
}

// DeleteCount removes a count and its items/drops (CASCADE).
func (s *Service) DeleteCount(id string) error {
	res, err := s.Store.DB.Exec(`DELETE FROM counts WHERE id = ?`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return errors.New("count not found")
	}
	return nil
}

// RestoreCount re-inserts a previously deleted count with its original IDs,
// timestamps and totals, so delete-undo is a true restore rather than a new
// count. Denomination snapshots travel with the items, so current settings
// are left untouched.
func (s *Service) RestoreCount(d store.CountDetail) (store.CountDetail, error) {
	if d.ID == "" {
		return store.CountDetail{}, errors.New("count id is required")
	}
	if !store.ValidCountType(d.Type) {
		return store.CountDetail{}, errors.New("type must be shift_open, shift_close, audit or donation_urne")
	}
	var exists int
	if err := s.Store.DB.QueryRow(`SELECT COUNT(*) FROM counts WHERE id = ?`, d.ID).Scan(&exists); err != nil {
		return store.CountDetail{}, err
	}
	if exists > 0 {
		return store.CountDetail{}, errors.New("count already exists")
	}

	tx, err := s.Store.DB.Begin()
	if err != nil {
		return store.CountDetail{}, err
	}
	defer tx.Rollback()

	for _, it := range d.Items {
		var refs int
		if err := tx.QueryRow(`SELECT COUNT(*) FROM denominations WHERE id = ?`, it.DenominationID).Scan(&refs); err != nil {
			return store.CountDetail{}, err
		}
		if refs == 0 {
			return store.CountDetail{}, fmt.Errorf("cannot restore: denomination %q was removed", it.DenominationLabel)
		}
	}

	if _, err := tx.Exec(
		`INSERT INTO counts (id, created_at, type, expected_cents, counted_cents, drops_cents, variance_cents, note) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		d.ID, d.CreatedAt.Format(time.RFC3339), d.Type, d.ExpectedCents, d.CountedCents, d.DropsCents, d.VarianceCents, d.Note,
	); err != nil {
		return store.CountDetail{}, err
	}
	for _, it := range d.Items {
		if _, err := tx.Exec(
			`INSERT INTO count_items (id, count_id, denomination_id, quantity, subtotal_cents, denomination_label, value_cents) VALUES (?, ?, ?, ?, ?, ?, ?)`,
			it.ID, d.ID, it.DenominationID, it.Quantity, it.SubtotalCents, it.DenominationLabel, it.ValueCents,
		); err != nil {
			return store.CountDetail{}, err
		}
	}
	for _, dr := range d.Drops {
		if _, err := tx.Exec(
			`INSERT INTO count_drops (id, count_id, amount_cents, note, created_at) VALUES (?, ?, ?, ?, ?)`,
			dr.ID, d.ID, dr.AmountCents, dr.Note, dr.CreatedAt.Format(time.RFC3339),
		); err != nil {
			return store.CountDetail{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return store.CountDetail{}, err
	}
	return s.GetCount(d.ID)
}

func centsStr(c int64) string {
	neg := c < 0
	if neg {
		c = -c
	}
	s := strconv.FormatInt(c/100, 10) + "." + fmt.Sprintf("%02d", c%100)
	if neg {
		return "-" + s
	}
	return s
}

func writeCSV(records [][]string) (string, error) {
	var buf bytes.Buffer
	buf.WriteString("\xef\xbb\xbf") // UTF-8 BOM for Excel
	w := csv.NewWriter(&buf)
	if err := w.WriteAll(records); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// ExportCountCSV writes a per-count CSV into the exports subdir of the
// app-data dir and returns the path.
func (s *Service) ExportCountCSV(id string) (string, error) {
	d, err := s.GetCount(id)
	if err != nil {
		return "", err
	}
	records := [][]string{
		{"count_id", "created_at", "type", "note"},
		{d.ID, d.CreatedAt.Format(time.RFC3339), d.Type, d.Note},
		{},
		{"denomination", "value", "quantity", "subtotal"},
	}
	for _, it := range d.Items {
		records = append(records, []string{
			it.DenominationLabel, centsStr(it.ValueCents),
			strconv.FormatInt(it.Quantity, 10), centsStr(it.SubtotalCents),
		})
	}
	records = append(records, []string{"drop_note", "amount"})
	for _, dr := range d.Drops {
		records = append(records, []string{dr.Note, centsStr(dr.AmountCents)})
	}
	records = append(records, []string{},
		[]string{"counted", "drops", "expected", "variance"},
		[]string{centsStr(d.CountedCents), centsStr(d.DropsCents), centsStr(d.ExpectedCents), centsStr(d.VarianceCents)},
	)
	content, err := writeCSV(records)
	if err != nil {
		return "", err
	}
	return s.writeExport(fmt.Sprintf("count-%s.csv", d.ID), content)
}

// ExportHistoryCSV writes the full count history into the exports subdir of
// the app-data dir and returns the path.
func (s *Service) ExportHistoryCSV() (string, error) {
	list, err := s.ListCounts(CountFilter{})
	if err != nil {
		return "", err
	}
	records := [][]string{
		{"id", "created_at", "type", "counted", "drops", "expected", "variance", "note"},
	}
	for _, c := range list {
		records = append(records, []string{
			c.ID, c.CreatedAt.Format(time.RFC3339), c.Type,
			centsStr(c.CountedCents), centsStr(c.DropsCents),
			centsStr(c.ExpectedCents), centsStr(c.VarianceCents), c.Note,
		})
	}
	content, err := writeCSV(records)
	if err != nil {
		return "", err
	}
	return s.writeExport(fmt.Sprintf("history-%s.csv", time.Now().Format("20060102-150405")), content)
}

func (s *Service) writeExport(name, content string) (string, error) {
	dir := filepath.Join(s.Store.Dir, "exports")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(dir, name)
	if err := atomicWrite(path, content); err != nil {
		return "", err
	}
	return path, nil
}
