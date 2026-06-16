package collector

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"neurolink/apex-server-monitor/statusapi"
)

type fakeProvider struct {
	snapshots []statusapi.Snapshot
	err       error
	calls     int
}

func (p *fakeProvider) Fetch(ctx context.Context) (statusapi.Snapshot, error) {
	if p.err != nil {
		return statusapi.UnavailableSnapshot(statusapi.SourceLive, p.err, time.Now()), p.err
	}
	if len(p.snapshots) == 0 {
		return statusapi.NewDemoProvider().Fetch(ctx)
	}
	index := p.calls
	if index >= len(p.snapshots) {
		index = len(p.snapshots) - 1
	}
	p.calls++
	return p.snapshots[index], nil
}

func TestPollStatusFetchesImmediately(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	out := make(chan PollResult, 2)
	provider := &fakeProvider{snapshots: []statusapi.Snapshot{testStatusSnapshot(statusapi.StatusHealthy)}}

	go PollStatus(ctx, provider, time.Hour, out)

	select {
	case result := <-out:
		if result.Err != nil {
			t.Fatalf("unexpected error = %v", result.Err)
		}
		if result.Snapshot.Overall != statusapi.StatusHealthy {
			t.Fatalf("overall = %s, want healthy", result.Snapshot.Overall)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for immediate poll")
	}
}

func TestCombinePublishesStatusWithBattleMode(t *testing.T) {
	originalNotify := notifyDesktop
	notifyDesktop = func(title string, message string, appIcon any) error { return nil }
	defer func() {
		notifyDesktop = originalNotify
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	modeCh := make(chan ModeState, 1)
	statusCh := make(chan PollResult, 1)
	out := make(chan Snapshot, 4)

	go Combine(ctx, modeCh, statusCh, out)

	statusCh <- PollResult{Snapshot: testStatusSnapshot(statusapi.StatusDegraded), At: time.Now()}
	modeCh <- ModeState{Mode: ModeBattle, BattleMode: true, ProcessName: "r5apex.exe", CheckedAt: time.Now()}

	snapshot := receiveCollectorSnapshotWithMode(t, out, ModeBattle)
	if snapshot.Mode != ModeBattle || !snapshot.BattleMode {
		t.Fatalf("mode = %s battle=%t, want battle mode", snapshot.Mode, snapshot.BattleMode)
	}
	if snapshot.ProcessName != "r5apex.exe" {
		t.Fatalf("process name = %q, want r5apex.exe", snapshot.ProcessName)
	}
	if snapshot.Status.Overall != statusapi.StatusDegraded {
		t.Fatalf("overall = %s, want degraded", snapshot.Status.Overall)
	}
}

func TestCombinePreservesProviderErrors(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	statusCh := make(chan PollResult, 1)
	out := make(chan Snapshot, 2)
	err := errors.New("status API returned HTTP 403")

	go Combine(ctx, nil, statusCh, out)
	statusCh <- PollResult{Err: err, At: time.Now()}

	snapshot := receiveCollectorSnapshot(t, out)
	if !strings.Contains(snapshot.LastError, "HTTP 403") {
		t.Fatalf("last error = %q, want HTTP 403", snapshot.LastError)
	}
	apiHealth, ok := statusapi.ServiceByID(snapshot.Status, statusapi.ServiceAPIHealth)
	if !ok || apiHealth.Status != statusapi.StatusDown {
		t.Fatalf("api health = %#v, want down service", apiHealth)
	}
}

func testStatusSnapshot(overall statusapi.Status) statusapi.Snapshot {
	now := time.Now()
	services := statusapi.CoreServices()
	for i := range services {
		services[i].Status = statusapi.StatusHealthy
		services[i].Summary = "test"
		services[i].UpdatedAt = now
	}
	if overall == statusapi.StatusDegraded {
		services[1].Status = statusapi.StatusDegraded
	}
	if overall == statusapi.StatusDown {
		services[1].Status = statusapi.StatusDown
	}
	return statusapi.Snapshot{
		Source:      statusapi.SourceDemo,
		Attribution: statusapi.Attribution,
		GeneratedAt: now,
		Overall:     overall,
		Services:    services,
	}
}

func receiveCollectorSnapshot(t *testing.T, ch <-chan Snapshot) Snapshot {
	t.Helper()

	select {
	case snapshot := <-ch:
		return snapshot
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for collector snapshot")
		return Snapshot{}
	}
}

func receiveCollectorSnapshotWithMode(t *testing.T, ch <-chan Snapshot, mode Mode) Snapshot {
	t.Helper()

	deadline := time.After(time.Second)
	for {
		select {
		case snapshot := <-ch:
			if snapshot.Mode == mode {
				return snapshot
			}
		case <-deadline:
			t.Fatalf("timed out waiting for collector snapshot with mode %s", mode)
			return Snapshot{}
		}
	}
}
