// Package store owns SQLite persistence: schema, migrations, seeds and the
// shared row models. Money is integer cents (int64) throughout.
package store

import "time"

// Count types.
const (
	CountTypeShiftOpen    = "shift_open"
	CountTypeShiftClose   = "shift_close"
	CountTypeAudit        = "audit"
	CountTypeDonationUrne = "donation_urne"
)

// Denomination kinds.
const (
	KindBill = "bill"
	KindCoin = "coin"
)

func ValidCountType(t string) bool {
	switch t {
	case CountTypeShiftOpen, CountTypeShiftClose, CountTypeAudit, CountTypeDonationUrne:
		return true
	}
	return false
}

func ValidKind(k string) bool {
	return k == KindBill || k == KindCoin
}

type Settings struct {
	ID             int64  `json:"id"`
	StoreName      string `json:"store_name"`
	CurrencyLabel  string `json:"currency_label"`
	CurrencySymbol string `json:"currency_symbol"`
}

type Denomination struct {
	ID         string `json:"id"`
	Label      string `json:"label"`
	ValueCents int64  `json:"value_cents"`
	Kind       string `json:"kind"`
	SortOrder  int64  `json:"sort_order"`
	Active     bool   `json:"active"`
}

type Count struct {
	ID            string    `json:"id"`
	CreatedAt     time.Time `json:"created_at"`
	Type          string    `json:"type"`
	ExpectedCents int64     `json:"expected_cents"`
	CountedCents  int64     `json:"counted_cents"`
	DropsCents    int64     `json:"drops_cents"`
	VarianceCents int64     `json:"variance_cents"`
	Note          string    `json:"note"`
}

type CountItem struct {
	ID                string `json:"id"`
	CountID           string `json:"count_id"`
	DenominationID    string `json:"denomination_id"`
	DenominationLabel string `json:"denomination_label"`
	ValueCents        int64  `json:"value_cents"`
	Quantity          int64  `json:"quantity"`
	SubtotalCents     int64  `json:"subtotal_cents"`
}

type CountDrop struct {
	ID          string    `json:"id"`
	CountID     string    `json:"count_id"`
	AmountCents int64     `json:"amount_cents"`
	Note        string    `json:"note"`
	CreatedAt   time.Time `json:"created_at"`
}

// CountDetail is the full immutable snapshot returned for detail views.
type CountDetail struct {
	Count
	Items []CountItem `json:"items"`
	Drops []CountDrop `json:"drops"`
}
