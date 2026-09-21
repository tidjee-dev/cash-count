package settings

import (
	"errors"

	"github.com/tidjee-dev/cash-count/backend/store"
)

// Service backs store info and currency presets.
type Service struct {
	Store *store.Store
}

func New(s *store.Store) *Service { return &Service{Store: s} }

type SettingsInput struct {
	StoreName      string `json:"store_name"`
	CurrencyLabel  string `json:"currency_label"`
	CurrencySymbol string `json:"currency_symbol"`
}

type CurrencyPreset struct {
	Label         string               `json:"label"`
	Symbol        string               `json:"symbol"`
	Denominations []PresetDenomination `json:"denominations"`
}

type PresetDenomination struct {
	Label      string `json:"label"`
	ValueCents int64  `json:"value_cents"`
	Kind       string `json:"kind"`
}

func (s *Service) GetSettings() (store.Settings, error) {
	var out store.Settings
	err := s.Store.DB.QueryRow(
		`SELECT id, store_name, currency_label, currency_symbol FROM settings WHERE id = 1`,
	).Scan(&out.ID, &out.StoreName, &out.CurrencyLabel, &out.CurrencySymbol)
	return out, err
}

func (s *Service) SaveSettings(in SettingsInput) (store.Settings, error) {
	if in.CurrencySymbol == "" {
		return store.Settings{}, errors.New("currency symbol is required")
	}
	if in.CurrencyLabel == "" {
		return store.Settings{}, errors.New("currency label is required")
	}
	_, err := s.Store.DB.Exec(
		`UPDATE settings SET store_name = ?, currency_label = ?, currency_symbol = ? WHERE id = 1`,
		in.StoreName, in.CurrencyLabel, in.CurrencySymbol,
	)
	if err != nil {
		return store.Settings{}, err
	}
	return s.GetSettings()
}

func (s *Service) ListCurrencyPresets() ([]CurrencyPreset, error) {
	return []CurrencyPreset{
		{
			Label: "EUR", Symbol: "€",
			Denominations: []PresetDenomination{
				{"€500", 50000, "bill"}, {"€200", 20000, "bill"}, {"€100", 10000, "bill"},
				{"€50", 5000, "bill"}, {"€20", 2000, "bill"}, {"€10", 1000, "bill"},
				{"€5", 500, "bill"}, {"€2", 200, "coin"}, {"€1", 100, "coin"},
				{"50c", 50, "coin"}, {"20c", 20, "coin"}, {"10c", 10, "coin"},
				{"5c", 5, "coin"}, {"2c", 2, "coin"}, {"1c", 1, "coin"},
			},
		},
		{
			Label: "USD", Symbol: "$",
			Denominations: []PresetDenomination{
				{"$100", 10000, "bill"}, {"$50", 5000, "bill"}, {"$20", 2000, "bill"},
				{"$10", 1000, "bill"}, {"$5", 500, "bill"}, {"$1", 100, "bill"},
				{"25¢", 25, "coin"}, {"10¢", 10, "coin"}, {"5¢", 5, "coin"}, {"1¢", 1, "coin"},
			},
		},
		{
			Label: "GBP", Symbol: "£",
			Denominations: []PresetDenomination{
				{"£50", 5000, "bill"}, {"£20", 2000, "bill"}, {"£10", 1000, "bill"},
				{"£5", 500, "bill"}, {"£2", 200, "coin"}, {"£1", 100, "coin"},
				{"50p", 50, "coin"}, {"20p", 20, "coin"}, {"10p", 10, "coin"},
				{"5p", 5, "coin"}, {"2p", 2, "coin"}, {"1p", 1, "coin"},
			},
		},
	}, nil
}
