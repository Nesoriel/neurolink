package collector

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/gen2brain/beeep"
)

var notifyDesktop = beeep.Notify

type clusterWindow struct {
	target  Target
	samples []PingSample
}

func Aggregate(ctx context.Context, cfg Config, modeCh <-chan ModeState, samples <-chan PingSample, out chan MetricsSnapshot) {
	cfg = cfg.withDefaults()

	windows := make(map[ClusterID]*clusterWindow, len(cfg.Targets))
	for _, target := range cfg.Targets {
		windows[target.Cluster] = &clusterWindow{target: target}
	}

	modeState := ModeState{Mode: ModeIdle, CheckedAt: time.Now()}
	highLossSince := make(map[ClusterID]time.Time, len(cfg.Targets))
	publishSnapshot(ctx, out, buildSnapshot(modeState, windows, cfg.LossThresholdPercent, cfg.WindowSize))
	defer close(out)

	for modeCh != nil || samples != nil {
		select {
		case <-ctx.Done():
			return
		case state, ok := <-modeCh:
			if !ok {
				modeCh = nil
				continue
			}
			enteringBattle := state.BattleMode && !modeState.BattleMode
			modeState = state
			if enteringBattle {
				resetWindows(windows)
				clear(highLossSince)
			}
			if !state.BattleMode {
				clear(highLossSince)
			}
			publishSnapshot(ctx, out, buildSnapshot(modeState, windows, cfg.LossThresholdPercent, cfg.WindowSize))
		case sample, ok := <-samples:
			if !ok {
				samples = nil
				continue
			}
			window, ok := windows[sample.Cluster]
			if !ok {
				window = &clusterWindow{
					target: Target{
						Cluster: sample.Cluster,
						Name:    sample.Target,
						Address: sample.Address,
					},
				}
				windows[sample.Cluster] = window
			}
			window.append(sample, cfg.WindowSize)

			snapshot := buildSnapshot(modeState, windows, cfg.LossThresholdPercent, cfg.WindowSize)
			checkLossNotification(cfg, modeState.BattleMode, sample, snapshot.Clusters[sample.Cluster], highLossSince)
			publishSnapshot(ctx, out, snapshot)
		}
	}
}

func (w *clusterWindow) append(sample PingSample, max int) {
	w.samples = append(w.samples, sample)
	if len(w.samples) > max {
		copy(w.samples, w.samples[len(w.samples)-max:])
		w.samples = w.samples[:max]
	}
}

func resetWindows(windows map[ClusterID]*clusterWindow) {
	for _, window := range windows {
		window.samples = nil
	}
}

func buildSnapshot(modeState ModeState, windows map[ClusterID]*clusterWindow, lossThreshold float64, maxWindowSize int) MetricsSnapshot {
	snapshot := MetricsSnapshot{
		Mode:        modeState.Mode,
		BattleMode:  modeState.BattleMode,
		ProcessName: modeState.ProcessName,
		UpdatedAt:   time.Now(),
		Clusters:    make(map[ClusterID]ClusterMetrics, len(windows)),
	}

	for cluster, window := range windows {
		snapshot.Clusters[cluster] = calculateMetrics(window, lossThreshold, maxWindowSize)
	}

	return snapshot
}

func calculateMetrics(window *clusterWindow, lossThreshold float64, maxWindowSize int) ClusterMetrics {
	metrics := ClusterMetrics{
		Cluster:    window.target.Cluster,
		Target:     window.target.Name,
		Address:    window.target.Address,
		Status:     StatusOffline,
		SignalBars: 0,
		WindowSize: maxWindowSize,
	}

	if len(window.samples) == 0 {
		return metrics
	}

	var (
		successCount int
		failureCount int
		totalRTT     time.Duration
		successRTTs  []time.Duration
	)

	for _, sample := range window.samples {
		if sample.At.After(metrics.LastUpdated) {
			metrics.LastUpdated = sample.At
			metrics.LastError = sample.Error
			metrics.LastRTT = sample.RTT
			metrics.LastSucceeded = sample.Success
		}
		if sample.Success {
			successCount++
			totalRTT += sample.RTT
			successRTTs = append(successRTTs, sample.RTT)
		} else {
			failureCount++
		}
	}

	metrics.SampleCount = len(window.samples)
	if successCount > 0 {
		metrics.AvgLatency = totalRTT / time.Duration(successCount)
	}
	metrics.PacketLoss = float64(failureCount) / float64(len(window.samples)) * 100
	metrics.Jitter = averageJitter(successRTTs)
	metrics.Status = statusFor(metrics.PacketLoss, successCount, lossThreshold)
	metrics.SignalBars = signalBars(metrics.Status, metrics.AvgLatency, metrics.PacketLoss)

	return metrics
}

func averageJitter(rtts []time.Duration) time.Duration {
	if len(rtts) < 2 {
		return 0
	}

	var total time.Duration
	for i := 1; i < len(rtts); i++ {
		diff := rtts[i] - rtts[i-1]
		if diff < 0 {
			diff = -diff
		}
		total += diff
	}
	return total / time.Duration(len(rtts)-1)
}

func statusFor(packetLoss float64, successCount int, lossThreshold float64) Status {
	if successCount == 0 {
		return StatusOffline
	}
	if packetLoss > lossThreshold {
		return StatusHighLoss
	}
	return StatusOnline
}

func signalBars(status Status, latency time.Duration, packetLoss float64) int {
	if status == StatusOffline {
		return 0
	}

	bars := 10
	bars -= int(math.Ceil(packetLoss / 10))

	switch {
	case latency > 180*time.Millisecond:
		bars -= 4
	case latency > 120*time.Millisecond:
		bars -= 3
	case latency > 80*time.Millisecond:
		bars -= 2
	case latency > 50*time.Millisecond:
		bars--
	}

	if status == StatusHighLoss && bars > 5 {
		bars = 5
	}
	if bars < 1 {
		return 1
	}
	if bars > 10 {
		return 10
	}
	return bars
}

func checkLossNotification(cfg Config, battleMode bool, sample PingSample, metrics ClusterMetrics, highLossSince map[ClusterID]time.Time) {
	if !battleMode || metrics.PacketLoss <= cfg.LossThresholdPercent {
		delete(highLossSince, sample.Cluster)
		return
	}

	startedAt, ok := highLossSince[sample.Cluster]
	if !ok {
		highLossSince[sample.Cluster] = sample.At
		return
	}

	requiredDuration := time.Duration(cfg.LossNotificationSamples-1) * cfg.BattlePingInterval
	if requiredDuration < 0 {
		requiredDuration = 0
	}
	if sample.At.Sub(startedAt) < requiredDuration {
		return
	}

	delete(highLossSince, sample.Cluster)
	title := "Apex server packet loss"
	message := fmt.Sprintf("%s packet loss is %.1f%%", metrics.Target, metrics.PacketLoss)
	go func() {
		_ = notifyDesktop(title, message, "")
	}()
}

func publishSnapshot(ctx context.Context, out chan MetricsSnapshot, snapshot MetricsSnapshot) {
	select {
	case <-ctx.Done():
		return
	case out <- snapshot:
		return
	default:
		// Drop this snapshot when the UI is behind. The next probe/mode tick will
		// publish a fresher value without the producer reading from its own output.
		return
	}
}
