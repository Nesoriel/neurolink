# neurolink

English | [简体中文](README.zh-Hans.md)

`neurolink` is a Go Terminal UI for monitoring Apex Legends service health before or during play, with explicit on-demand Apex player stat lookup. It presents a Crypto-style surveillance dashboard focused on Apex service availability and quick pre-game inspection, not fake ICMP pings or background player tracking.

The app monitors service availability such as:

- Crossplay Auth
- Lobby / Matchmaking
- PC / Desktop Logins
- Player Accounts
- Apex Legends Status API health
- Recent community reports when the status payload includes them

The player view can query a player by name and platform (`PC`, `PS4`, or `X1`) using Apex Legends Status `/bridge`. Lookup only runs after a visible user action such as pressing Enter in the player view. PC lookup generally uses the Origin account name, even for Steam-linked accounts. `neurolink` does not continuously track players, query match history in the background, or hide telemetry.

## Data Source

The primary data source is the Apex Legends Status public site/API ecosystem:

- Website: `https://apexlegendsstatus.com/`
- API docs/base: `https://apexlegendsapi.com/`
- API base: `https://api.apexlegendsstatus.com/`

The dashboard polls:

```text
GET https://api.apexlegendsstatus.com/servers
```

The player view queries only on demand:

```text
GET https://api.apexlegendsstatus.com/bridge?player=PLAYER_NAME&platform=PLATFORM
```

Both endpoints require an Apex Legends Status API key for live data. The app sends the key with the `Authorization` header and never hard-codes or logs sample tokens.

The `/servers` endpoint may return JSON with `Content-Type: text/plain;charset=UTF-8`, so the client sends `Accept: */*` and decodes the response body as JSON.

The normalized dashboard includes the core service cards above. Overall status is derived from playable game services; the Apex Legends Status API health card is shown separately so a status-site issue does not automatically mark Apex gameplay services as down.

The player lookup normalizes the `/bridge` response into identity, platform, UID, level, rank, selected legend, and tracker values. It intentionally avoids presenting continuous online/lobby tracking.

## Install

From this checkout:

```bash
go install .
```

Or, when installing by module path:

```bash
go install github.com/Nesoriel/neurolink@latest
```

## Run

Run with demo data:

```bash
neurolink
```

Run with live status polling as a one-off override:

```bash
neurolink --api-key "$YOUR_APEX_STATUS_API_KEY" --poll-interval 1m
```

The same flags work with `go run .` during development:

```bash
go run . --demo
go run . --lang zh-Hans
```

## TUI Controls

- `tab` / `shift+tab`: switch views
- `1`: dashboard
- `2`: player lookup
- `3`: config
- `?`: help
- `/`: open player lookup and focus the player input
- `enter`: run player lookup from the player view
- `p`: cycle `PC` / `PS4` / `X1` when the player input is not focused
- `r`: request an immediate service status refresh
- `q` / `ctrl+c`: quit

## Persistent Config

`neurolink` stores optional persistent settings in the cross-platform user config directory returned by Go's `os.UserConfigDir()`:

```text
<user-config-dir>/neurolink/config.json
```

Typical locations are `~/.config/neurolink/config.json` on Linux, `~/Library/Application Support/neurolink/config.json` on macOS, and `%AppData%\neurolink\config.json` on Windows.

Show the active config path:

```bash
neurolink config path
```

Persist settings:

```bash
neurolink config set api-key "$YOUR_APEX_STATUS_API_KEY"
neurolink config set language en
neurolink config set language zh-Hans
neurolink config set poll-interval 30s
```

You can also save multiple settings at once:

```bash
neurolink config set --api-key "$YOUR_APEX_STATUS_API_KEY" --lang zh-Hans --poll-interval 30s
```

Show saved settings. The API key is always masked:

```bash
neurolink config show
```

Unset saved settings:

```bash
neurolink config unset api-key
neurolink config unset language poll-interval
neurolink config unset all
```

Use a custom config path when needed:

```bash
neurolink --config ./local.config.json config show
neurolink --config ./local.config.json
```

Normal run precedence is:

```text
defaults < config file < environment < flags
```

## Environment Variables

- `NEUROLINK_APEX_API_KEY`: one-off API key override
- `NEUROLINK_LANG`: one-off language override, `en` or `zh-Hans`
- `NEUROLINK_POLL_INTERVAL`: one-off poll interval override, such as `30s` or `1m`
- `NEUROLINK_CONFIG`: config file path override

Flags still take final precedence:

```bash
NEUROLINK_APEX_API_KEY="$YOUR_APEX_STATUS_API_KEY" neurolink --poll-interval 1m
NEUROLINK_LANG=zh-Hans neurolink --lang en
```

## Behavior Without an API Key

If no API key is available from the config file, environment, or `--api-key`, `neurolink` starts in demo mode. You can also force demo mode explicitly:

```bash
neurolink --demo
```

Demo mode uses deterministic sample data for UI preview and local development. The dashboard and player result cards label the source as `DEMO` and do not present sample data as live Apex service or player data.

## Build and Test

Build:

```bash
go build ./...
```

Test:

```bash
go test ./...
go test -race ./...
```

Format:

```bash
gofmt -w .
```

## Community Pulse

The top-right report panel is labeled `Community Pulse`. It shows recent community reports only when the `/servers` payload includes a report feed. If the feed is absent, the panel explains that the current `/servers` response does not include recent reports and keeps attribution to Apex Legends Status.

This panel is service-health context only. Player stats are available only through the explicit player lookup view.

## Optional Ping Diagnostics

The older ICMP ping probe remains available only as an explicit diagnostics module. It is not the primary monitoring source and should not be presented as real Apex server monitoring unless concrete targets are configured.

On Linux, non-privileged ICMP diagnostics may require:

```bash
sudo sysctl -w net.ipv4.ping_group_range="0 2147483647"
```

## Current Limits and TODO

- The Apex Legends Status API payload shape may change; normalization handles common fields and core service aliases.
- Without an API key, only demo data is available.
- Recent community reports appear only when the `/servers` payload provides a report feed.
- Player lookup currently supports name + platform searches for `PC`, `PS4`, and `X1`; UID lookup is documented upstream but not exposed in this first TUI iteration.
- The player view summarizes stats from `/bridge` and avoids continuous online/lobby tracking.
- Ping diagnostics are not yet exposed as a dedicated TUI panel.
- Future work may add more service cards, richer status history, saved lookup defaults, and more granular desktop notifications.
