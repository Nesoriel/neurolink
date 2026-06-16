package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"

	"neurolink/apex-server-monitor/collector"
	"neurolink/apex-server-monitor/tui"
)

func main() {
	cfg := collector.DefaultConfig()

	hongKongTarget := flag.String("hk", envOrDefault("NEUROLINK_HK_TARGET", collector.HongKongTargetIP), "Hong Kong Apex server target IP or hostname")
	singaporeTarget := flag.String("sg", envOrDefault("NEUROLINK_SG_TARGET", collector.SingaporeTargetIP), "Singapore Apex server target IP or hostname")
	flag.Parse()

	cfg.Targets = []collector.Target{
		{Cluster: collector.ClusterHongKong, Name: "Hong Kong Server Cluster", Address: *hongKongTarget},
		{Cluster: collector.ClusterSingapore, Name: "Singapore Server Cluster", Address: *singaporeTarget},
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	metricsCh := collector.Start(ctx, cfg)

	program := tea.NewProgram(tui.NewModel(metricsCh), tea.WithAltScreen())
	_, err := program.Run()
	cancel()
	if err != nil {
		fmt.Fprintf(os.Stderr, "apex-server-monitor: %v\n", err)
		os.Exit(1)
	}
}

func envOrDefault(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
