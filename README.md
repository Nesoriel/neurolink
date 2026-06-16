# neurolink

## apex-server-monitor

Initial Go TUI for monitoring Apex Legends server quality for Hong Kong and
Singapore clusters.

### Build

```bash
go build ./...
```

For dependency maintenance after changing imports, run:

```bash
go mod tidy
```

### Run

```bash
go run .
```

The initial server addresses are RFC 5737 TEST-NET placeholders (`192.0.2.10`
and `198.51.100.10`). They are safe defaults that should not route on the public
internet, so the UI will show offline until you provide real targets.

Override targets without editing source:

```bash
go run . -hk <hong-kong-target> -sg <singapore-target>
```

Or via environment variables:

```bash
NEUROLINK_HK_TARGET=<hong-kong-target> \
NEUROLINK_SG_TARGET=<singapore-target> \
go run .
```

On Linux, ICMP may require unprivileged ping support:

```bash
sudo sysctl -w net.ipv4.ping_group_range="0 2147483647"
```

The process watcher checks for `r5apex.exe`/`r5apex` every 5 seconds. When the
game is detected, probes run every second; otherwise they run every 15 seconds.
