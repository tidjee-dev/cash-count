# Cash Count

Desktop app for POS **cash counting**: count the drawer by denomination, compare
against an expected amount, record safe drops, and keep a clean, exportable
history. Local-first, single register, no accounts, no cloud.

Built with [Wails v3](https://v3.wails.io) (Go backend) + Svelte 5 + Tailwind 4 +
shadcn-svelte.

## Features

- Denomination count grid with quantity steppers and live subtotals
- Manual expected-amount entry, plus safe drops that reconcile back into variance
- Variance (over/short/balanced) with color-coded feedback: `counted + drops − expected`
- Count types: `shift_open`, `shift_close`, `audit`, `donation_urne`
- Full count history with detail view, filters, and delete + undo
- CSV export (single count + full history), Excel-friendly (UTF-8 BOM)
- Configurable store info, currency presets (EUR/USD/GBP), and denominations
- English + French UI, dark/light mode

## Quick start

Prereqs: Go (see `go.mod`), Node + npm, and the
[Wails v3 CLI](https://v3.wails.io).

```bash
cd frontend && npm install && cd ..
wails3 dev
```

Production build:

```bash
wails3 build
```

See [DOCUMENTATION.md](DOCUMENTATION.md) for usage, architecture, and schema.

## Storage

- Database: single SQLite file at the app-data directory root, e.g.
  `~/.local/share/cashcount/cashcount.db` on Linux.
- Exports: CSV files in the `exports/` subfolder, e.g.
  `~/.local/share/cashcount/exports/`.
- Best backup: copy the app-data folder while the app is closed.

## Development

| Task                | Command                            |
| ---------------------| ------------------------------------|
| Run (hot reload)    | `wails3 dev`                       |
| Regenerate bindings | `wails3 generate bindings`         |
| Backend checks      | `go vet ./...` and `go test ./...` |
| Frontend checks     | `cd frontend && npm run check`     |
| Production build    | `wails3 build`                     |

Bindings (`frontend/bindings/`) are generated — never edit them by hand. Money
is integer cents (`int64`) end to end; see [PLAN.md](PLAN.md) for the original
project plan.

## License

MIT — see [LICENSE](LICENSE).
