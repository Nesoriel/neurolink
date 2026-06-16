package collector

import (
	"context"
	"path/filepath"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v3/process"
)

var apexProcessNames = map[string]struct{}{
	"r5apex.exe": {},
	"r5apex":     {},
}

func WatchProcess(ctx context.Context, interval time.Duration, out chan<- ModeState) {
	if interval <= 0 {
		interval = 5 * time.Second
	}

	var last ModeState
	var initialized bool
	checkAndSend := func() bool {
		running, name := IsApexRunning()
		mode := ModeIdle
		if running {
			mode = ModeBattle
		}

		state := ModeState{
			Mode:        mode,
			BattleMode:  running,
			ProcessName: name,
			CheckedAt:   time.Now(),
		}
		if initialized && state.Mode == last.Mode && state.BattleMode == last.BattleMode && state.ProcessName == last.ProcessName {
			return true
		}

		initialized = true
		last = state
		return sendMode(ctx, out, state)
	}

	if !checkAndSend() {
		return
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !checkAndSend() {
				return
			}
		}
	}
}

func IsApexRunning() (bool, string) {
	processes, err := process.Processes()
	if err != nil {
		return false, ""
	}

	for _, proc := range processes {
		name, err := proc.Name()
		if err == nil && matchesApexExecutable(name) {
			return true, name
		}

		cmdline, err := proc.Cmdline()
		if err == nil && matchesProtonApexCommand(cmdline) {
			if name != "" {
				return true, name
			}
			return true, cmdline
		}
	}

	return false, ""
}

func matchesApexExecutable(value string) bool {
	base := strings.ToLower(filepath.Base(strings.TrimSpace(value)))
	_, ok := apexProcessNames[base]
	return ok
}

func matchesProtonApexCommand(value string) bool {
	lower := strings.ToLower(value)
	if !strings.Contains(lower, "r5apex") {
		return false
	}
	return strings.Contains(lower, "proton") || strings.Contains(lower, "wine") || strings.Contains(lower, "steamapps")
}

func sendMode(ctx context.Context, out chan<- ModeState, state ModeState) bool {
	select {
	case <-ctx.Done():
		return false
	case out <- state:
		return true
	}
}
