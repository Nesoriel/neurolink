package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"

	"neurolink/apex-server-monitor/collector"
	"neurolink/apex-server-monitor/tui"
)

func main() {
	appCfg, err := parseAppConfig(os.Args[1:], os.Getenv)
	if err != nil {
		fmt.Fprintf(os.Stderr, "neurolink: %v\n", err)
		os.Exit(2)
	}

	collectorCfg := collector.DefaultConfig()
	collectorCfg.Provider = appCfg.provider()
	collectorCfg.PollInterval = appCfg.PollInterval

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	snapshots := collector.Start(ctx, collectorCfg)

	program := tea.NewProgram(tui.NewModel(snapshots), tea.WithAltScreen())
	_, err = program.Run()
	cancel()
	if err != nil {
		fmt.Fprintf(os.Stderr, "neurolink: %v\n", err)
		os.Exit(1)
	}
}
