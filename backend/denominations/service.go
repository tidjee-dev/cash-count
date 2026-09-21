package denominations

import (
	"errors"

	"github.com/tidjee-dev/cash-count/backend/store"
)

// Service manages the configurable denomination list.
type Service struct {
	Store *store.Store
}

func New(s *store.Store) *Service { return &Service{Store: s} }

type DenominationInput struct {
	ID         string `json:"id"`
	Label      string `json:"label"`
	ValueCents int64  `json:"value_cents"`
	Kind       string `json:"kind"`
	SortOrder  int64  `json:"sort_order"`
	Active     bool   `json:"active"`
}

func scanDenom(scanner interface {
	Scan(dest ...any) error
}, d *store.Denomination) error {
	var active int
	err := scanner.Scan(&d.ID, &d.Label, &d.ValueCents, &d.Kind, &d.SortOrder, &active)
	d.Active = active != 0
	return err
}

// List returns denominations ordered bills → coins. Pass activeOnly to hide
// disabled rows (count screen); settings passes false to edit everything.
func (s *Service) List(activeOnly bool) ([]store.Denomination, error) {
	q := `SELECT id, label, value_cents, kind, sort_order, active FROM denominations ORDER BY sort_order, id`
	if activeOnly {
		q = `SELECT id, label, value_cents, kind, sort_order, active FROM denominations WHERE active = 1 ORDER BY sort_order, id`
	}
	rows, err := s.Store.DB.Query(q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []store.Denomination{}
	for rows.Next() {
		var d store.Denomination
		if err := scanDenom(rows, &d); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// SaveAll replaces the denomination list. Rows still referenced by history are
// deactivated instead of deleted so past counts stay intact.
func (s *Service) SaveAll(in []DenominationInput) ([]store.Denomination, error) {
	for _, d := range in {
		if d.Label == "" {
			return nil, errors.New("denomination label is required")
		}
		if d.ValueCents <= 0 {
			return nil, errors.New("denomination value must be positive")
		}
		if !store.ValidKind(d.Kind) {
			return nil, errors.New("denomination kind must be bill or coin")
		}
	}
	tx, err := s.Store.DB.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	keep := map[string]bool{}
	for i, d := range in {
		if d.ID != "" {
			if _, err := tx.Exec(
				`UPDATE denominations SET label = ?, value_cents = ?, kind = ?, sort_order = ?, active = ? WHERE id = ?`,
				d.Label, d.ValueCents, d.Kind, i, boolToInt(d.Active), d.ID,
			); err != nil {
				return nil, err
			}
			keep[d.ID] = true
		} else {
			id, err := store.NewID("den")
			if err != nil {
				return nil, err
			}
			if _, err := tx.Exec(
				`INSERT INTO denominations (id, label, value_cents, kind, sort_order, active) VALUES (?, ?, ?, ?, ?, ?)`,
				id, d.Label, d.ValueCents, d.Kind, i, boolToInt(d.Active),
			); err != nil {
				return nil, err
			}
			keep[id] = true
		}
	}

	// Remove rows absent from the payload: hard-delete only when unreferenced,
	// otherwise deactivate to preserve history snapshots.
	rows, err := tx.Query(`SELECT id FROM denominations`)
	if err != nil {
		return nil, err
	}
	var stale []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return nil, err
		}
		if !keep[id] {
			stale = append(stale, id)
		}
	}
	rows.Close()
	for _, id := range stale {
		var refs int
		if err := tx.QueryRow(`SELECT COUNT(*) FROM count_items WHERE denomination_id = ?`, id).Scan(&refs); err != nil {
			return nil, err
		}
		if refs > 0 {
			if _, err := tx.Exec(`UPDATE denominations SET active = 0 WHERE id = ?`, id); err != nil {
				return nil, err
			}
		} else if _, err := tx.Exec(`DELETE FROM denominations WHERE id = ?`, id); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return s.List(false)
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
