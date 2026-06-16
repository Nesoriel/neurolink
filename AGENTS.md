# Role

You are an expert Go (Golang) developer and a master of Terminal User Interfaces (TUI). You follow clean code principles, idiomatic Go design patterns, and Cloud-Native concurrency practices.

# Project Overview

We are building a lightweight, real-time TUI command-line tool named `apex-server-monitor`. The purpose is to monitor EA's Apex Legends server quality (specifically for Hong Kong and Singapore servers) to help players check latency and packet loss before or during a match.

# Tech Stack Constraints (Strictly Follow)

- **Language**: Go (Latest stable version)
- **TUI Framework**: `github.com/charmbracelet/bubbletea` for the Elm architecture loop, combined with `github.com/charmbracelet/lipgloss` for styling and layout.
- **Networking/Ping**: `github.com/prometheus-community/pro-bing` for pure Go ICMP ping execution.
- **Process Monitoring**: `github.com/shirou/gopsutil/v3/process` to detect the game client.
- **System Notification**: `github.com/gen2brain/beeep` for cross-platform desktop alerts.

# Core Architecture & Modules

Please structure the project into the following concurrent modules using Goroutines and Channels:

1. **Data Collector (Goroutines)**:

   - **Network Probe**: Periodically (default 1s) pings target Apex server IPs (Hong Kong & Singapore data centers). It must collect RTT (Latency) and track failed pings to calculate Packet Loss.
   - **Process Watcher**: Periodically (every 5s) checks if `r5apex.exe` (or the Linux Proton equivalent process) is running. If running, enter "Battle Mode" (ping interval = 1s); if not, enter "Idle Mode" (ping interval = 15s).
2. **Data Aggregator (Sliding Window)**:

   - Receives ping data from a channel.
   - Implements a moving average/sliding window (e.g., last 10 ticks) to calculate average latency, packet loss percentage, and jitter.
   - Triggers a desktop notification via `beeep` if packet loss exceeds 5% for 3 consecutive seconds during Battle Mode.
3. **TUI View (Bubble Tea)**:

   - Layout should be split horizontally or vertically using `lipgloss`.
   - Display two main columns/panes: **[Hong Kong Server Cluster]** and **[Singapore Server Cluster]**.
   - Show real-time metrics: Status (ONLINE/HIGH LOSS/OFFLINE), Avg Latency (ms), Packet Loss (%), and a simple text-based signal bar (e.g., 📶 [██████░░░░]).

# What to Generate Now

Please generate the initial project structure and the boilerplate code to set up this architecture:

1. `main.go`: The entry point that initializes the Bubble Tea program and coordinates channels.
2. `collector/`: The package handling the `pro-bing` and `gopsutil` background loops.
3. `tui/`: The package handling the Bubble Tea `Model`, `Update`, and `View` functions, styled cleanly with `lipgloss`.

Provide clean, compilable code with necessary comments explaining how the channels flow between the collector and the TUI loop (`tea.Cmd`).
