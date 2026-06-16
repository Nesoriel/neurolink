# neurolink

English | [简体中文](README.zh-Hans.md)

`neurolink` is a Go Terminal UI for monitoring Apex Legends service health before or during play. It presents a Crypto-style surveillance dashboard focused on Apex service availability, not fake ICMP pings or player-status tracking.

The app monitors service availability such as:

- Crossplay Auth
- Lobby / Matchmaking
- PC / Desktop Logins
- Player Accounts
- Apex Legends Status API health
- Recent community reports when the status payload includes them

`neurolink` does not query player profiles, player stats, UID lookup, match history, or player online status.

## Data Source

The primary data source is the Apex Legends Status public site/API ecosystem:

- Website: `https://apexlegendsstatus.com/`
- API docs/base: `https://apexlegendsapi.com/`
- API base: `https://api.apexlegendsstatus.com/`

The current implementation polls:

```text
GET https://api.apexlegendsstatus.com/servers
```

The `/servers` endpoint may return JSON with `Content-Type: text/plain;charset=UTF-8`, so the client sends `Accept: */*` and decodes the response body as JSON.

The normalized dashboard includes the core service cards above. Overall status is derived from playable game services; the Apex Legends Status API health card is shown separately so a status-site issue does not automatically mark Apex gameplay services as down.

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

Demo mode uses deterministic sample data for UI preview and local development. The dashboard labels the source as `DEMO` and does not present sample data as live Apex service status.

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

This panel is service-health context only. It is not player profile or player-status tracking.

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
- Ping diagnostics are not yet exposed as a dedicated TUI panel.
- Future work may add more service cards, status history, and more granular desktop notifications.
