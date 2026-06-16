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

## API Key

Apex Legends Status API requests normally require an API key. `neurolink` never hard-codes keys and does not include sample real tokens.

Use a flag:

```bash
go run . --api-key "$YOUR_APEX_STATUS_API_KEY"
```

Or use an environment variable:

```bash
NEUROLINK_APEX_API_KEY="$YOUR_APEX_STATUS_API_KEY" go run .
```

## Language

The TUI supports English and Simplified Chinese:

```bash
go run . --lang en
go run . --lang zh-Hans
```

Or:

```bash
NEUROLINK_LANG=zh-Hans go run .
```

Supported values are `en` and `zh-Hans`. The default is `en`.

## Demo Mode

If no API key is provided, `neurolink` starts in demo mode. You can also force demo mode explicitly:

```bash
go run . --demo
```

Demo mode uses deterministic sample data for UI preview and local development. The dashboard labels the source as `DEMO` and does not present sample data as live Apex service status.

## Run, Build, Test

Run with demo data:

```bash
go run .
```

Run with live status polling:

```bash
NEUROLINK_APEX_API_KEY="$YOUR_APEX_STATUS_API_KEY" go run . --poll-interval 1m
```

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
- Future work may add a config file, more service cards, status history, and more granular desktop notifications.
