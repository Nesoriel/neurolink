package collector

import (
	"context"
	"fmt"
	"time"

	"github.com/Nesoriel/neurolink/statusapi"
)

// Start wires the application's background pipeline. The primary feed is the
// Apex Legends Status API provider; process watching only changes the displayed
// play mode and alert policy.
func Start(ctx context.Context, cfg Config) <-chan Snapshot {
	return StartWithRefresh(ctx, cfg, nil)
}

func StartWithRefresh(ctx context.Context, cfg Config, refresh <-chan struct{}) <-chan Snapshot {
	cfg = cfg.withDefaults()

	rawModeCh := make(chan ModeState, 1)
	statusCh := make(chan PollResult, 2)
	snapshotCh := make(chan Snapshot, 4)

	go WatchProcess(ctx, cfg.ProcessCheckInterval, rawModeCh)
	go PollStatusWithRefresh(ctx, cfg.Provider, cfg.PollInterval, refresh, statusCh)
	go Combine(ctx, rawModeCh, statusCh, snapshotCh)

	return snapshotCh
}

func PollStatus(ctx context.Context, provider statusapi.Provider, interval time.Duration, out chan<- PollResult) {
	PollStatusWithRefresh(ctx, provider, interval, nil, out)
}

func PollStatusWithRefresh(ctx context.Context, provider statusapi.Provider, interval time.Duration, refresh <-chan struct{}, out chan<- PollResult) {
	if interval <= 0 {
		interval = time.Minute
	}
	defer close(out)

	fetchAndSend := func() bool {
		snapshot, err := provider.Fetch(ctx)
		result := PollResult{Snapshot: snapshot, Err: err, At: time.Now()}
		select {
		case <-ctx.Done():
			return false
		case out <- result:
			return true
		}
	}

	if !fetchAndSend() {
		return
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case _, ok := <-refresh:
			if !ok {
				refresh = nil
				continue
			}
			if !fetchAndSend() {
				return
			}
		case <-ticker.C:
			if !fetchAndSend() {
				return
			}
		}
	}
}

func Combine(ctx context.Context, modeCh <-chan ModeState, statusCh <-chan PollResult, out chan<- Snapshot) {
	modeState := ModeState{Mode: ModeIdle, CheckedAt: time.Now()}
	var latest statusapi.Snapshot
	var hasStatus bool
	var lastOverall statusapi.Status
	defer close(out)

	for modeCh != nil || statusCh != nil {
		select {
		case <-ctx.Done():
			return
		case state, ok := <-modeCh:
			if !ok {
				modeCh = nil
				continue
			}
			modeState = state
			if hasStatus {
				publish(ctx, out, buildSnapshot(modeState, latest, ""))
			}
		case result, ok := <-statusCh:
			if !ok {
				statusCh = nil
				continue
			}
			snapshot := result.Snapshot
			if snapshot.GeneratedAt.IsZero() {
				snapshot = statusapi.UnavailableSnapshot(statusapi.SourceLive, result.Err, result.At)
			}
			if result.Err != nil && snapshot.LastError == "" {
				snapshot.LastError = result.Err.Error()
			}
			hasStatus = true
			latest = snapshot
			if shouldNotify(modeState.BattleMode, lastOverall, snapshot.Overall) {
				notifyServiceIncident(snapshot)
			}
			lastOverall = snapshot.Overall
			publish(ctx, out, buildSnapshot(modeState, snapshot, snapshot.LastError))
		}
	}
}

func buildSnapshot(modeState ModeState, status statusapi.Snapshot, lastError string) Snapshot {
	updatedAt := status.GeneratedAt
	if updatedAt.IsZero() {
		updatedAt = time.Now()
	}
	return Snapshot{
		Mode:        modeState.Mode,
		BattleMode:  modeState.BattleMode,
		ProcessName: modeState.ProcessName,
		ModeChecked: modeState.CheckedAt,
		Status:      status,
		UpdatedAt:   updatedAt,
		LastError:   lastError,
	}
}

func shouldNotify(battleMode bool, previous statusapi.Status, current statusapi.Status) bool {
	if !battleMode || !current.IsIncident() {
		return false
	}
	return previous == "" || !previous.IsIncident()
}

func notifyServiceIncident(snapshot statusapi.Snapshot) {
	title := "Apex service status"
	message := fmt.Sprintf("%s: %s", snapshot.Overall, incidentSummary(snapshot))
	_ = notifyDesktop(title, message, "")
}

func incidentSummary(snapshot statusapi.Snapshot) string {
	var impacted int
	for _, service := range snapshot.Services {
		if service.Status.IsIncident() {
			impacted++
		}
	}
	if impacted == 0 {
		return "service incident detected"
	}
	return fmt.Sprintf("%d impacted services", impacted)
}

func publish(ctx context.Context, out chan<- Snapshot, snapshot Snapshot) {
	select {
	case <-ctx.Done():
		return
	case out <- snapshot:
		return
	default:
		// Drop stale frames if the Bubble Tea loop is behind; the next poll or
		// mode change will carry the latest state.
		return
	}
}
