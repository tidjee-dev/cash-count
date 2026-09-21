package counts

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	denomsvc "github.com/tidjee-dev/cash-count/backend/denominations"
	"github.com/tidjee-dev/cash-count/backend/settings"
	"github.com/tidjee-dev/cash-count/backend/store"
)

func openTest(t *testing.T) (*store.Store, *Service) {
	t.Helper()
	st, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	return st, New(st)
}

func TestEndToEnd(t *testing.T) {
	st, svc := openTest(t)

	ss := settings.New(st)
	got, err := ss.GetSettings()
	if err != nil || got.CurrencySymbol != "€" || got.CurrencyLabel != "EUR" {
		t.Fatalf("seeded settings: %+v err=%v", got, err)
	}

	ds := denomsvc.New(st)
	list, err := ds.List(true)
	if err != nil || len(list) != 15 {
		t.Fatalf("seeded denoms: n=%d err=%v", len(list), err)
	}
	var twenty, fiftyc string
	for _, d := range list {
		if d.Label == "€20" {
			twenty = d.ID
		}
		if d.Label == "50c" {
			fiftyc = d.ID
		}
	}
	if twenty == "" || fiftyc == "" {
		t.Fatal("seed rows missing")
	}

	d, err := svc.CreateCount(CreateCountInput{
		Type:          store.CountTypeShiftClose,
		ExpectedCents: 2000,
		Note:          "test",
		Items: []CountItemInput{
			{DenominationID: twenty, Quantity: 1},
			{DenominationID: fiftyc, Quantity: 2},
		},
		Drops: []CountDropInput{{AmountCents: 500, Note: "safe"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if d.CountedCents != 2100 || d.DropsCents != 500 || d.VarianceCents != 600 {
		t.Fatalf("bad totals: %+v", d.Count)
	}
	if len(d.Items) != 2 || len(d.Drops) != 1 {
		t.Fatalf("bad detail lengths: %+v", d)
	}

	got2, err := svc.GetCount(d.ID)
	if err != nil || got2.VarianceCents != 600 {
		t.Fatalf("get: %+v err=%v", got2, err)
	}
	all, err := svc.ListCounts(CountFilter{})
	if err != nil || len(all) != 1 {
		t.Fatalf("list: n=%d err=%v", len(all), err)
	}
	filtered, err := svc.ListCounts(CountFilter{Type: store.CountTypeAudit})
	if err != nil || len(filtered) != 0 {
		t.Fatalf("filter: n=%d err=%v", len(filtered), err)
	}

	p1, err := svc.ExportCountCSV(d.ID)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(p1)
	if !strings.HasPrefix(string(raw), "\xef\xbb\xbf") || !strings.Contains(string(raw), "variance") {
		t.Fatal("count csv malformed")
	}
	p2, err := svc.ExportHistoryCSV()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(p2); err != nil {
		t.Fatal(err)
	}
	if filepath.Dir(p1) != filepath.Join(st.Dir, "exports") {
		t.Fatalf("count export outside exports dir: %q", p1)
	}
	if filepath.Dir(p2) != filepath.Join(st.Dir, "exports") {
		t.Fatalf("history export outside exports dir: %q", p2)
	}

	// Removing a referenced denomination deactivates instead of deleting.
	rest := []denomsvc.DenominationInput{}
	for _, x := range list {
		if x.ID == twenty {
			continue
		}
		rest = append(rest, denomsvc.DenominationInput{
			ID: x.ID, Label: x.Label, ValueCents: x.ValueCents,
			Kind: x.Kind, SortOrder: x.SortOrder, Active: x.Active,
		})
	}
	after, err := ds.SaveAll(rest)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, x := range after {
		if x.Label == "€20" {
			found = true
			if x.Active {
				t.Fatal("€20 should be deactivated, not kept active")
			}
		}
	}
	if !found {
		t.Fatal("€20 row must survive (referenced by history)")
	}

	if _, err := svc.CreateCount(CreateCountInput{
		Type: store.CountTypeAudit,
		Items: []CountItemInput{
			{DenominationID: twenty, Quantity: 1},
		},
	}); err == nil {
		t.Fatal("counting a disabled denomination must fail")
	}

	if err := svc.DeleteCount(d.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.GetCount(d.ID); err == nil {
		t.Fatal("deleted count must be gone")
	}
}

func TestDeleteRestoreRoundTrip(t *testing.T) {
	st, svc := openTest(t)

	list, err := denomsvc.New(st).List(true)
	if err != nil {
		t.Fatal(err)
	}
	byLabel := map[string]string{}
	for _, d := range list {
		byLabel[d.Label] = d.ID
	}
	created, err := svc.CreateCount(CreateCountInput{
		Type:          store.CountTypeShiftClose,
		ExpectedCents: 1000,
		Note:          "roundtrip",
		Items: []CountItemInput{
			{DenominationID: byLabel["€20"], Quantity: 1},
			{DenominationID: byLabel["50c"], Quantity: 4},
		},
		Drops: []CountDropInput{{AmountCents: 100, Note: "safe"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := svc.GetCount(created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.DeleteCount(created.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.GetCount(created.ID); err == nil {
		t.Fatal("deleted count must be gone")
	}
	restored, err := svc.RestoreCount(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if restored.ID != snapshot.ID {
		t.Fatalf("restore id: got %q want %q", restored.ID, snapshot.ID)
	}
	if restored.CreatedAt.Unix() != snapshot.CreatedAt.Unix() ||
		restored.CountedCents != snapshot.CountedCents ||
		restored.VarianceCents != snapshot.VarianceCents ||
		restored.Note != snapshot.Note {
		t.Fatalf("restore mismatch: %+v vs %+v", restored.Count, snapshot.Count)
	}
	if len(restored.Items) != len(snapshot.Items) || len(restored.Drops) != len(snapshot.Drops) {
		t.Fatalf("restore lengths: %+v", restored)
	}
	for i := range restored.Items {
		a, b := restored.Items[i], snapshot.Items[i]
		if a.ID != b.ID || a.DenominationID != b.DenominationID || a.Quantity != b.Quantity || a.SubtotalCents != b.SubtotalCents {
			t.Fatalf("restore item %d mismatch: %+v vs %+v", i, a, b)
		}
	}
	if _, err := svc.RestoreCount(snapshot); err == nil {
		t.Fatal("restoring twice must fail")
	}
}

func TestItemsOrderedLargestFirst(t *testing.T) {
	st, svc := openTest(t)

	ds := denomsvc.New(st)
	list, err := ds.List(true)
	if err != nil {
		t.Fatal(err)
	}
	byLabel := map[string]string{}
	for _, d := range list {
		byLabel[d.Label] = d.ID
	}
	// Intentionally ascending input: 50c, €10, €20.
	d, err := svc.CreateCount(CreateCountInput{
		Type:          store.CountTypeAudit,
		ExpectedCents: 0,
		Items: []CountItemInput{
			{DenominationID: byLabel["50c"], Quantity: 50},
			{DenominationID: byLabel["€10"], Quantity: 10},
			{DenominationID: byLabel["€20"], Quantity: 10},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"€20", "€10", "50c"}
	for i, label := range want {
		if d.Items[i].DenominationLabel != label {
			t.Fatalf("create order: got %q want %q (full: %+v)", d.Items[i].DenominationLabel, label, d.Items)
		}
	}

	got, err := svc.GetCount(d.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Items) != 3 {
		t.Fatalf("get items: n=%d", len(got.Items))
	}
	for i, label := range want {
		if got.Items[i].DenominationLabel != label {
			t.Fatalf("get order: got %q want %q (full: %+v)", got.Items[i].DenominationLabel, label, got.Items)
		}
	}
}
