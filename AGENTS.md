# Role

You are an expert Go developer and a master of modern Terminal User Interfaces. This project must feel like a polished Charmbracelet-style product, not a plain debug table.

# Product

`neurolink` is a Go TUI tool for monitoring Apex Legends service health. The theme is a Crypto-style background “surveillance drone” for checking whether Apex services are healthy before or during play.

# Core Direction

The primary data source is service-status polling, not fake server pings.

Use the public site/API ecosystem around Apex Legends Status:

- Website reference: `https://apexlegendsstatus.com/`
- API docs/base: `https://apexlegendsapi.com/` and `https://api.apexlegendsstatus.com/`
- Server status endpoint: `GET https://api.apexlegendsstatus.com/servers`

Important: the API normally requires an API key. Never hard-code keys. Read keys from flags or environment variables only. The `/servers` endpoint may return JSON as `text/plain;charset=UTF-8`; clients should avoid strict `Accept: application/json` and decode the body as JSON.

This project is about Apex service/server availability only. Do not implement player profile, player stats, UID lookup, match history, or other user-status features unless the user explicitly changes scope.

The app should monitor service availability such as:

- Crossplay Auth
- Lobby / Matchmaking Servers
- PC / Desktop Logins
- Player Accounts
- Apex Legends Status website/API health
- Recent user reports if available

Keep the old ICMP ping probe only as an optional diagnostics module. Do not present placeholder TEST-NET pings as real Apex server monitoring.

# Required Tech Stack

- Language: Go
- TUI: `github.com/charmbracelet/bubbletea`
- Styling/layout: `github.com/charmbracelet/lipgloss`
- Optional ping diagnostics: `github.com/prometheus-community/pro-bing`
- Process detection: `github.com/shirou/gopsutil/v3/process`
- Desktop alerts: `github.com/gen2brain/beeep`

# Architecture

Use goroutines and channels, but keep responsibilities clean:

1. `statusapi/`
   - API client and response normalization.
   - Supports real API mode when an API key is configured.
   - Supports mock/demo mode with honest labels when no key is present.
   - Must not fabricate live data as if it were real.

2. `collector/`
   - Polls status API at an interval.
   - Watches Apex process and exposes Battle/Idle mode.
   - Emits normalized snapshots through channels.
   - Optional ping diagnostics can be a separate secondary feed.

3. `tui/`
   - Bubble Tea model, update loop, and view rendering.
   - Must render a polished dashboard with clear sections, colors, status chips, progress bars, and help text.
   - Should be useful in narrow terminals and attractive in wide terminals.

# TUI Quality Bar

The interface should look closer to Charmbracelet examples than to a raw table.

Minimum visual requirements:

- Strong header with app name, mode, source, and last update time.
- Status summary card: overall status, degraded services count, API mode.
- Service cards for Lobby/Matchmaking, Crossplay Auth, PC Login, Player Accounts, and API health.
- Clear color coding: healthy, degraded, down, unknown.
- Use glyphs and bars tastefully, e.g. `● RUNNING`, `▲ DEGRADED`, `✕ DOWN`, `▰▰▰▱▱`.
- Include a footer showing keys and configuration hints.
- Do not dump raw JSON or full network errors into the main pane.

# UX / Configuration

Provide flags and environment variables:

- API key: `--api-key` or `NEUROLINK_APEX_API_KEY`
- Poll interval: `--poll-interval`
- Demo mode: `--demo`
- Language: `--lang` or `NEUROLINK_LANG`, supporting `en` and `zh-Hans` with default `en`
- Optional ping diagnostics targets: explicit flags only

If no API key is provided, start in demo mode or show a clear setup screen. The user must understand whether data is real or demo.

# Documentation

README.md must be English-first. Keep a full Simplified Chinese document at README.zh-Hans.md for domestic users. Cross-link both files near the top, and update both when user-facing behavior, configuration, limits, or data-source explanations change.

Both English and Chinese docs must explain:

- What the tool does
- Current data sources
- API key configuration
- Language configuration
- Behavior without an API key
- How to run, build, and test
- Current limitations and TODOs

# Engineering Rules

- Keep code idiomatic and testable.
- Add tests for normalization, status mapping, config parsing, and collector behavior.
- Do not commit secrets or sample real tokens.
- Keep `go test ./...` and `go build ./...` passing.
- Prefer small focused packages over one giant file.
