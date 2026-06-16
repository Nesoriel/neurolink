package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Nesoriel/neurolink/collector"
	"github.com/Nesoriel/neurolink/tui"
)

func main() {
	if isConfigCommand(os.Args[1:]) {
		if err := runConfigCommand(os.Args[1:], os.Getenv, os.Stdout); err != nil {
			fmt.Fprintf(os.Stderr, "neurolink: %v\n", err)
			os.Exit(2)
		}
		return
	}

	appCfg, err := parseAppConfig(os.Args[1:], os.Getenv)
	if err != nil {
		fmt.Fprintf(os.Stderr, "neurolink: %v\n", err)
		os.Exit(2)
	}

	collectorCfg := collector.DefaultConfig()
	collectorCfg.Provider = appCfg.provider()
	collectorCfg.PollInterval = appCfg.PollInterval

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	refresh := make(chan struct{}, 1)
	snapshots := collector.StartWithRefresh(ctx, collectorCfg, refresh)

	program := tea.NewProgram(tui.NewModelWithPlayerProvider(snapshots, appCfg.playerProvider(), refresh, appCfg.Language), tea.WithAltScreen())
	_, err = program.Run()
	cancel()
	if err != nil {
		fmt.Fprintf(os.Stderr, "neurolink: %v\n", err)
		os.Exit(1)
	}
}
