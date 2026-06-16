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
	snapshots := collector.Start(ctx, collectorCfg)

	program := tea.NewProgram(tui.NewModel(snapshots, appCfg.Language), tea.WithAltScreen())
	_, err = program.Run()
	cancel()
	if err != nil {
		fmt.Fprintf(os.Stderr, "neurolink: %v\n", err)
		os.Exit(1)
	}
}
