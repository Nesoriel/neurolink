package collector

import "context"

// Start wires the application's background pipeline:
//   - WatchProcess emits ModeState values when Apex starts or exits.
//   - broadcastMode fans those mode changes out to every network probe and the
//     aggregator. A single Go channel is not a broadcast primitive, so this
//     small fan-out goroutine keeps each consumer independent.
//   - Each NetworkProbe goroutine pings one target and sends PingSample values.
//   - Aggregate consumes samples plus mode changes, maintains sliding windows,
//     and emits MetricsSnapshot values for the Bubble Tea model.
//
// The TUI receives only the final metrics channel. Its tea.Cmd blocks on this
// channel, returns a message to Update, then schedules another wait command.
func Start(ctx context.Context, cfg Config) <-chan MetricsSnapshot {
	cfg = cfg.withDefaults()

	rawModeCh := make(chan ModeState, 1)
	sampleCh := make(chan PingSample, len(cfg.Targets)*2)
	metricsCh := make(chan MetricsSnapshot, 4)
	aggregatorModeCh := make(chan ModeState, 1)

	subscribers := []chan ModeState{aggregatorModeCh}
	for _, target := range cfg.Targets {
		modeCh := make(chan ModeState, 1)
		subscribers = append(subscribers, modeCh)
		go NetworkProbe(ctx, target, cfg, modeCh, sampleCh)
	}

	go WatchProcess(ctx, cfg.ProcessCheckInterval, rawModeCh)
	go broadcastMode(ctx, rawModeCh, subscribers...)
	go Aggregate(ctx, cfg, aggregatorModeCh, sampleCh, metricsCh)

	return metricsCh
}

func broadcastMode(ctx context.Context, in <-chan ModeState, subscribers ...chan ModeState) {
	for {
		select {
		case <-ctx.Done():
			return
		case state, ok := <-in:
			if !ok {
				return
			}
			for _, subscriber := range subscribers {
				sendLatestMode(ctx, subscriber, state)
			}
		}
	}
}

func sendLatestMode(ctx context.Context, ch chan ModeState, state ModeState) {
	select {
	case ch <- state:
		return
	default:
	}

	select {
	case <-ch:
	default:
	}

	select {
	case <-ctx.Done():
		return
	case ch <- state:
		return
	default:
		return
	}
}
