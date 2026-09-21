# Cash Count

Desktop app for POS **cash counting**: count the drawer by denomination, compare
against an expected amount, record safe drops, and keep a clean, exportable
history. Local-first, single register, Wails v3 + SvelteKit.

## Features

- [x] Denomination count grid with quantity steppers and live subtotals
- [x] Manual expected-amount entry
- [x] Variance (over/short) with color-coded feedback
- [x] Safe drops recorded against a count
- [x] Full count history with detail view
- [x] CSV export (single count + full history)
- [x] Configurable currency and denominations

## Tech stack

| Layer     | Technology                                          |
| --------- | --------------------------------------------------- |
| Runtime   | Wails v3 (Go backend, WebView frontend)             |
| Backend   | Go services + SQLite (`modernc.org/sqlite`, no CGO) |
| Frontend  | SvelteKit 5 + `@sveltejs/adapter-static` (SSR off)  |
| UI        | Tailwind 4 + [shadcn-svelte](https://shadcn-svelte.com) |
| Bindings  | `wails3 generate bindings` (auto-generated TS)      |

## Quick start

Prereqs: Go, Node + `npm`, and the [Wails v3 CLI](https://v3.wails.io).

```bash
wails3 init -n cashcount -t sveltekit-ts
cd cashcount && git init

cd frontend
pnpm install
pnpm dlx shadcn-svelte@latest init
pnpm dlx shadcn-svelte@latest add button card dialog input label select badge table tabs separator sonner
cd ..

wails3 dev
```

SvelteKit's static adapter + SSR-off are already configured by the template; a
migration run seeds the database on first launch.

## Usage

**Open / close / audit a count** — Go to `Count`. Stepper `+`/`−` per
denomination, select the count type, enter the expected amount, add any safe
drops, then **Save**. The variance panel updates live:

```
variance = counted + safe drops − expected
```

Zero = balanced · positive = over · negative = short.

**Record a safe drop** — In the Count panel, add an amount and optional note.
Drops factor back into the variance, so money pulled to the safe doesn't look
like a shortage.

**Review history** — `History` lists every count (date, type, counted, expected,
variance). Open a row for the full breakdown of denominations and drops.

**Export** — From history or a count detail: download a CSV for that count or
the whole history. Files are written to the `exports/` folder inside the app
data directory; the app shows the saved path with a copy button.

**Configure** — `Settings` sets the store name, picks a currency preset, and
lets you add, remove, or disable denominations. Changes apply live to the count
screen.

## Architecture

Three Go services back the whole app. The webview calls them through
auto-generated, type-safe bindings — no HTTP layer in between.

```mermaid
flowchart LR
  FE[SvelteKit UI<br/>/count · /history · /settings]
  BEO[Bindings<br/>frontend/bindings/]
  SVC[Go Services<br/>Settings · Denomination · Count]
  DB[(SQLite<br/>app-data dir)]
  FE --> BEO --> SVC --> DB
  SVC -->|CSV files| FS[(OS filesystem)]
```

### Schema

```mermaid
erDiagram
  DENOMINATIONS ||--o{ COUNT_ITEMS : counted
  COUNTS ||--o{ COUNT_ITEMS : holds
  COUNTS ||--o{ COUNT_DROPS : pulls
  SETTINGS ||--o| DENOMINATIONS : defines
  SETTINGS {
    string store_name
    string currency_label
    string currency_symbol
  }
  DENOMINATIONS {
    int id
    string label
    int value_cents
    string kind "bill | coin"
    int sort_order
    bool active
  }
  COUNTS {
    int id
    timestamp created_at
    string type "shift_open | shift_close | audit"
    int expected_cents
    int counted_cents
    int drops_cents
    int variance_cents
    string note
  }
  COUNT_ITEMS {
    int id
    int count_id
    int denomination_id
    int quantity
    int subtotal_cents
  }
  COUNT_DROPS {
    int id
    int count_id
    int amount_cents
    string note
    timestamp created_at
  }
```

Money is stored as **integer cents** throughout. Totals are denormalized onto
`counts`, so history stays accurate even if denominations are edited later.

### Data flow

```mermaid
sequenceDiagram
  participant U as Cashier
  participant F as SvelteKit
  participant G as Go Service
  participant D as SQLite
  U->>F: steppers, expected, drops
  F->>G: CreateCount(input)
  G->>G: validate + compute variance
  G->>D: insert count + items + drops (tx)
  D-->>G: row
  G-->>F: Count detail
  F-->>U: success dialog + breakdown
```

## Storage

- Database: single SQLite file at the app-data directory root (e.g.
  `~/.local/share/cashcount/cashcount.db` on Linux).
- Exports: CSV files in the `exports/` subfolder (e.g.
  `~/.local/share/cashcount/exports/`). Older installs may still have loose
  `*.csv` files at the root; they are left untouched.
- Best backup: the app-data folder while the app is closed.

## Development

| Task                | Command                                  |
| ------------------- | ---------------------------------------- |
| Run (hot reload)    | `wails3 dev`                             |
| Regenerate bindings | `wails3 generate bindings`               |
| Backend checks      | `go vet ./...`                           |
| Frontend checks     | `cd frontend && npx svelte-check`        |
| Production build    | `wails3 build`                           |

Bindings are generated automatically on `dev`/`build` — never edit
`frontend/bindings/` by hand.

## Roadmap

- PDF export (HTML report printed from the webview, or a Go PDF lib).
- Expected cash computed from recorded POS sales.
- Multi-register support and cashier attribution.
- Per-count currency snapshot/remapping.
