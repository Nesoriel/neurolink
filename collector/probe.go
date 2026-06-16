package collector

import (
	"context"
	"time"

	probing "github.com/prometheus-community/pro-bing"
)

func NetworkProbe(ctx context.Context, target Target, cfg Config, modeCh <-chan ModeState, out chan<- PingSample) {
	mode := ModeIdle
	probed := false
	timer := time.NewTimer(0)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case state, ok := <-modeCh:
			if !ok {
				return
			}
			mode = state.Mode
			if probed {
				resetTimer(timer, intervalForMode(mode, cfg))
			} else {
				resetTimer(timer, 0)
			}
		case <-timer.C:
			sample := pingOnce(target, mode, cfg.PingTimeout)
			probed = true
			if !sendSample(ctx, out, sample) {
				return
			}
			resetTimer(timer, intervalForMode(mode, cfg))
		}
	}
}

func pingOnce(target Target, mode Mode, timeout time.Duration) PingSample {
	sample := PingSample{
		Cluster: target.Cluster,
		Target:  target.Name,
		Address: target.Address,
		Mode:    mode,
		At:      time.Now(),
	}

	pinger, err := probing.NewPinger(target.Address)
	if err != nil {
		sample.Error = err.Error()
		return sample
	}

	pinger.Count = 1
	pinger.Timeout = timeout

	if err := pinger.Run(); err != nil {
		sample.Error = err.Error()
		return sample
	}

	stats := pinger.Statistics()
	if stats == nil || stats.PacketsRecv == 0 {
		sample.Error = "no echo reply"
		return sample
	}

	sample.Success = true
	sample.RTT = stats.AvgRtt
	return sample
}

func intervalForMode(mode Mode, cfg Config) time.Duration {
	if mode == ModeBattle {
		return cfg.BattlePingInterval
	}
	return cfg.IdlePingInterval
}

func sendSample(ctx context.Context, out chan<- PingSample, sample PingSample) bool {
	select {
	case <-ctx.Done():
		return false
	case out <- sample:
		return true
	}
}

func resetTimer(timer *time.Timer, d time.Duration) {
	if !timer.Stop() {
		select {
		case <-timer.C:
		default:
		}
	}
	timer.Reset(d)
}
