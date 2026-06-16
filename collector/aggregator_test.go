package collector

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestClusterWindowAppendKeepsLatestSamples(t *testing.T) {
	window := &clusterWindow{
		target: Target{Cluster: ClusterHongKong, Name: "Hong Kong", Address: "192.0.2.10"},
	}

	for i := 0; i < 12; i++ {
		window.append(PingSample{
			Cluster: ClusterHongKong,
			Target:  "Hong Kong",
			Address: "192.0.2.10",
			At:      time.Unix(int64(i), 0),
			RTT:     time.Duration(i) * time.Millisecond,
			Success: true,
		}, 10)
	}

	if len(window.samples) != 10 {
		t.Fatalf("sample count = %d, want 10", len(window.samples))
	}
	if got := window.samples[0].RTT; got != 2*time.Millisecond {
		t.Fatalf("oldest retained RTT = %s, want 2ms", got)
	}
	if got := window.samples[9].RTT; got != 11*time.Millisecond {
		t.Fatalf("newest retained RTT = %s, want 11ms", got)
	}
}

func TestCalculateMetricsAveragesLossJitterAndSignal(t *testing.T) {
	base := time.Unix(100, 0)
	window := &clusterWindow{
		target: Target{Cluster: ClusterHongKong, Name: "Hong Kong", Address: "192.0.2.10"},
		samples: []PingSample{
			{Cluster: ClusterHongKong, At: base, RTT: 40 * time.Millisecond, Success: true},
			{Cluster: ClusterHongKong, At: base.Add(time.Second), RTT: 70 * time.Millisecond, Success: true},
			{Cluster: ClusterHongKong, At: base.Add(2 * time.Second), Success: false, Error: "timeout"},
			{Cluster: ClusterHongKong, At: base.Add(3 * time.Second), RTT: 100 * time.Millisecond, Success: true},
		},
	}

	metrics := calculateMetrics(window, 5, 10)

	if metrics.Status != StatusHighLoss {
		t.Fatalf("status = %s, want %s", metrics.Status, StatusHighLoss)
	}
	if metrics.AvgLatency != 70*time.Millisecond {
		t.Fatalf("average latency = %s, want 70ms", metrics.AvgLatency)
	}
	if metrics.PacketLoss != 25 {
		t.Fatalf("packet loss = %.1f, want 25.0", metrics.PacketLoss)
	}
	if metrics.Jitter != 30*time.Millisecond {
		t.Fatalf("jitter = %s, want 30ms", metrics.Jitter)
	}
	if metrics.SignalBars != 5 {
		t.Fatalf("signal bars = %d, want 5", metrics.SignalBars)
	}
	if !metrics.LastSucceeded || metrics.LastRTT != 100*time.Millisecond {
		t.Fatalf("last sample = success %t RTT %s, want success true RTT 100ms", metrics.LastSucceeded, metrics.LastRTT)
	}
}

func TestStatusForThresholds(t *testing.T) {
	tests := []struct {
		name       string
		packetLoss float64
		successes  int
		threshold  float64
		want       Status
	}{
		{name: "offline without successes", packetLoss: 100, successes: 0, threshold: 5, want: StatusOffline},
		{name: "online at threshold", packetLoss: 5, successes: 10, threshold: 5, want: StatusOnline},
		{name: "high loss above threshold", packetLoss: 5.1, successes: 10, threshold: 5, want: StatusHighLoss},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := statusFor(tt.packetLoss, tt.successes, tt.threshold); got != tt.want {
				t.Fatalf("statusFor() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestSignalBars(t *testing.T) {
	tests := []struct {
		name       string
		status     Status
		latency    time.Duration
		packetLoss float64
		want       int
	}{
		{name: "offline", status: StatusOffline, latency: 0, packetLoss: 100, want: 0},
		{name: "excellent", status: StatusOnline, latency: 20 * time.Millisecond, packetLoss: 0, want: 10},
		{name: "latency penalty", status: StatusOnline, latency: 130 * time.Millisecond, packetLoss: 0, want: 7},
		{name: "high loss cap", status: StatusHighLoss, latency: 20 * time.Millisecond, packetLoss: 6, want: 5},
		{name: "minimum online bar", status: StatusHighLoss, latency: 250 * time.Millisecond, packetLoss: 95, want: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := signalBars(tt.status, tt.latency, tt.packetLoss); got != tt.want {
				t.Fatalf("signalBars() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestAggregateAppliesDefaultsAndEmitsMetrics(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	modeCh := make(chan ModeState, 1)
	sampleCh := make(chan PingSample, 3)
	out := make(chan MetricsSnapshot, 8)
	cfg := Config{
		Targets: []Target{
			{Cluster: ClusterSingapore, Name: "Singapore", Address: "198.51.100.10"},
		},
	}

	go Aggregate(ctx, cfg, modeCh, sampleCh, out)

	initial := receiveSnapshot(t, out)
	if initial.Mode != ModeIdle {
		t.Fatalf("initial mode = %s, want %s", initial.Mode, ModeIdle)
	}

	modeCh <- ModeState{Mode: ModeBattle, BattleMode: true, ProcessName: "r5apex.exe", CheckedAt: time.Now()}
	receiveSnapshotWithMode(t, out, ModeBattle)

	sampleCh <- PingSample{Cluster: ClusterSingapore, Target: "Singapore", Address: "198.51.100.10", At: time.Now(), RTT: 42 * time.Millisecond, Success: true}

	snapshot := receiveSnapshotWithCluster(t, out, ClusterSingapore, 1)
	metrics := snapshot.Clusters[ClusterSingapore]
	if snapshot.Mode != ModeBattle || !snapshot.BattleMode {
		t.Fatalf("mode = %s battle=%t, want battle mode", snapshot.Mode, snapshot.BattleMode)
	}
	if metrics.Status != StatusOnline {
		t.Fatalf("status = %s, want %s", metrics.Status, StatusOnline)
	}
	if metrics.AvgLatency != 42*time.Millisecond {
		t.Fatalf("avg latency = %s, want 42ms", metrics.AvgLatency)
	}
	if metrics.SampleCount != 1 {
		t.Fatalf("sample count = %d, want 1", metrics.SampleCount)
	}
}

func TestLossNotificationRequiresCurrentBattleModeStreak(t *testing.T) {
	originalNotify := notifyDesktop
	notifications := make(chan string, 1)
	notifyDesktop = func(title string, message string, appIcon any) error {
		notifications <- title + ": " + message
		return nil
	}
	defer func() {
		notifyDesktop = originalNotify
	}()

	cfg := Config{LossThresholdPercent: 5, LossNotificationSamples: 3, BattlePingInterval: time.Second}
	sample := PingSample{Cluster: ClusterHongKong, Mode: ModeBattle}
	metrics := ClusterMetrics{Target: "Hong Kong", PacketLoss: 50}
	highLossSince := make(map[ClusterID]time.Time)

	for i := 0; i < 3; i++ {
		checkLossNotification(cfg, false, sample, metrics, highLossSince)
	}
	if _, ok := highLossSince[ClusterHongKong]; ok {
		t.Fatal("idle high-loss timer should not start")
	}
	assertNoNotification(t, notifications)

	base := time.Unix(100, 0)
	sample.At = base
	checkLossNotification(cfg, true, sample, metrics, highLossSince)
	sample.At = base.Add(time.Second)
	checkLossNotification(cfg, true, sample, metrics, highLossSince)
	assertNoNotification(t, notifications)

	sample.At = base.Add(2 * time.Second)
	checkLossNotification(cfg, true, sample, metrics, highLossSince)
	got := receiveNotification(t, notifications)
	if !strings.Contains(got, "Hong Kong packet loss is 50.0%") {
		t.Fatalf("notification = %q, want packet loss message", got)
	}
}

func receiveSnapshot(t *testing.T, ch <-chan MetricsSnapshot) MetricsSnapshot {
	t.Helper()

	select {
	case snapshot := <-ch:
		return snapshot
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for metrics snapshot")
		return MetricsSnapshot{}
	}
}

func receiveSnapshotWithCluster(t *testing.T, ch <-chan MetricsSnapshot, cluster ClusterID, samples int) MetricsSnapshot {
	t.Helper()

	deadline := time.After(time.Second)
	for {
		select {
		case snapshot := <-ch:
			if metrics, ok := snapshot.Clusters[cluster]; ok && metrics.SampleCount == samples {
				return snapshot
			}
		case <-deadline:
			t.Fatalf("timed out waiting for %s sample count %d", cluster, samples)
			return MetricsSnapshot{}
		}
	}
}

func receiveSnapshotWithMode(t *testing.T, ch <-chan MetricsSnapshot, mode Mode) MetricsSnapshot {
	t.Helper()

	deadline := time.After(time.Second)
	for {
		select {
		case snapshot := <-ch:
			if snapshot.Mode == mode {
				return snapshot
			}
		case <-deadline:
			t.Fatalf("timed out waiting for mode %s", mode)
			return MetricsSnapshot{}
		}
	}
}

func assertNoNotification(t *testing.T, ch <-chan string) {
	t.Helper()

	select {
	case got := <-ch:
		t.Fatalf("unexpected notification %q", got)
	default:
	}
}

func receiveNotification(t *testing.T, ch <-chan string) string {
	t.Helper()

	select {
	case got := <-ch:
		return got
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for notification")
		return ""
	}
}
