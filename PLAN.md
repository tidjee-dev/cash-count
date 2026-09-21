# Cash Count — Project Plan

> Status: Implemented · Last updated: 2026-09-21

## Overview

A desktop app for a POS **cash count** workflow: the cashier counts the physical
money in the drawer by denomination, the app totals it, compares it against an
expected amount, and records the result with any safe drops. Single-register,
single-user, fully local.

## Goals

- Fast, keyboard-friendly denomination counting with live totals.
- Clear, unambiguous variance (over/short) feedback.
- Reliable local persistence with a full audit history.
- Configurable denominations and currency without code changes.
- CSV export for records and reporting.

## Non-goals (v1)

- No multi-register, multi-user, or login.
- No sales/POS data integration — expected amount is entered manually.
- No cloud sync, no multi-device.
- No invoicing, inventory, or payments.

## Decisions

| Area         | Choice                                            | Rationale                                        |
| ------------ | ------------------------------------------------- | ------------------------------------------------ |
| Backend      | Wails v3 (Go services, embedded webview)          | Single binary, no server to deploy                       |
| Frontend     | Svelte 5 SPA (Vite + `svelte-spa-router`)         | Lightweight hash routing, no SSR needed for embedding    |
| UI           | Tailwind 4 + shadcn-svelte                        | Consistent, accessible component system                  |
| Persistence  | SQLite via `modernc.org/sqlite` (pure Go, no CGO) | Single-file DB, no CGO toolchain needed                  |
| Money        | Integer cents everywhere (Go `int64`)             | Eliminates float rounding errors                         |
| Bindings     | `wails3 generate bindings` (auto-generated TS)    | Type-safe, zero boilerplate front-end calls              |
| Export       | CSV (Go `encoding/csv`, UTF-8 BOM) into `exports/`| Excel-friendly, dependency-free, DB dir stays clean      |

## Scope

Core features, all confirmed:

- [x] Denomination count grid with quantity steppers and live subtotals.
- [x] Manual expected-amount entry.
- [x] Variance tracking — computed, color-coded over/short.
- [x] Safe drops recorded against a count.
- [x] Count history with detail view and filters.
- [x] CSV export (single count + full history) into `exports/` under the app-data dir.
- [x] Configurable currency and denominations.

> **Note:** all items below shipped. PDF export and expected-from-sales
> remain explicitly deferred (see Open questions).

## Data model

Money is stored as integer cents. Totals are denormalized onto `counts` so a
count row is immutable history even if denominations change later.

Entity tables use opaque text IDs (`den_…`, `cnt_…`, `itm_…`, `drp_…`); only
the settings singleton keeps a fixed integer id.

| Table            | Columns                                                                                                            |
| ---------------- | ------------------------------------------------------------------------------------------------------------------ |
| `settings`       | id, store_name, currency_label, currency_symbol                                                                   |
| `denominations`  | id, label, value_cents, kind (`bill`/`coin`), sort_order, active                                                    |
| `counts`         | id, created_at, type (`shift_open`/`shift_close`/`audit`/`donation_urne`), expected_cents, counted_cents, drops_cents, variance_cents, note |
| `count_items`    | id, count_id (FK), denomination_id (FK), quantity, subtotal_cents, denomination_label (snapshot), value_cents (snapshot) |
| `count_drops`    | id, count_id (FK), amount_cents, note, created_at                                                                  |

**Variance formula**

```
variance = counted_cents + drops_cents - expected_cents
```

Safe drops are money pulled from the drawer to the safe, so they are added back
to reconcile. Zero = balanced; positive = over; negative = short.

## Service API

Three Wails v3 Go services, registered in `main.go`.

### SettingsService

```
GetSettings()                           Settings
SaveSettings(input SettingsInput)       Settings
ListCurrencyPresets()                   []CurrencyPreset
```

### DenominationService

```
List(activeOnly bool)                     []Denomination
SaveAll(input []DenominationInput)      []Denomination
```

### CountService

```
CreateCount(input CreateCountInput)     CountDetail   // validates + computes totals
ListCounts(filter CountFilter)          []Count       // newest first, optional type/date filter + limit/offset
GetCount(id string)                     CountDetail
DeleteCount(id string)                  void          // items/drops removed via CASCADE
RestoreCount(detail CountDetail)        CountDetail   // delete-undo with original IDs
ExportCountCSV(id string)               string        // saved file path under exports/
ExportHistoryCSV()                      string
```

## Frontend

| Route          | Screen                                                              |
| -------------- | ------------------------------------------------------------------- |
| `/`            | **Count** — the primary screen                                      |
| `/history`     | **History** — filterable table of counts                            |
| `/history/:id` | **Count detail** — full breakdown incl. items and drops             |
| `/settings`    | **Settings** — store info, currency preset, denomination editor     |

Hash routing via `svelte-spa-router`; the app shell is a sidebar + header
(theme and language toggles), with native File/Go menus wired to the same
routes.

### Count screen layout

- **Left:** denominations grouped `bills` → `coins`, one row per denomination
  with `+`/`−` quantity steppers; live per-row subtotal; grand total footer.
- **Right:** count type select (`shift_open` / `shift_close` / `audit` /
  `donation_urne`; urns carry no expected amount), expected amount input
  (with history shortcut), safe drops list (+ amount & note), note field.
- **Variance panel:** large, color-coded result (green `+` / red `−` / neutral
  balanced), computed on a reactive `variance = counted + drops − expected`.
- **Save** → `CreateCount` → success toast with a "view in history" action;
  the form resets for the next count.

## Milestones (all complete)

### M1 — Scaffold
- `wails3 init -n cashcount -t sveltekit-ts`, git init.
- Verify `wails3 dev` shows the template window.

**Acceptance:** blank app runs via `wails3 dev`.

### M2 — Backend
- SQLite layer: schema, migrations, currency/denomination seeds.
- Services + `main.go` registration; `wails3 generate bindings`.
- `CreateCount` computes and persists totals in one transaction.

**Acceptance:** services callable from frontend bindings; counts persist across
restarts; DB file created in app data dir.

### M3 — Frontend shell
- `shadcn-svelte` components (button card table dialog input label select badge
  separator sonner, …).
- App shell with sidebar + route scaffolding.

**Acceptance:** all routes render; nav works.

### M4 — Count screen
- Denomination grid, variance math, safe drops, save flow, success toast.

**Acceptance:** manual count against expected produces correct variance; a safe
drop shifts the balance by its amount; data appears in history.

### M5 — History
- `/history` table + `/history/[id]` detail; CSV export buttons.

**Acceptance:** counts list, filter, open detail; CSVs open cleanly in Excel.

### M6 — Settings
- Store name, currency preset picker, denomination editor (add/remove/disable,
  live values in cents).

**Acceptance:** edits reflect in count screen; counters never miscount.

### M7 — Polish & verify
- Formatting helpers (cents → `$1,234.56`), empty states, confirm dialogs on
  delete, keyboard flow (Tab / arrows / Enter).
- `go vet ./...`, `npm run check` (svelte-check), `wails3 build` smoke test.

**Acceptance:** all checks pass; production build launches.

## Open questions / Future

- PDF export — defer; candidate: HTML report rendered in the webview + print,
  or a Go PDF lib (e.g. `maroto`) if a hard copy is needed.
- Expected-from-sales — natural next step if a POS data source ever appears.
- Multi-register / cashier login — depends on store needs; schema is ready for
  extra FK columns.
- Currency switching mid-history: item rows snapshot their denomination
  label/value, but the currency symbol always renders from current settings —
  there is no per-count currency snapshot, so old counts relabel when the
  currency changes.