package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"neurolink/apex-server-monitor/collector"
	"neurolink/apex-server-monitor/statusapi"
)

func TestViewRendersPolishedDemoDashboard(t *testing.T) {
	now := time.Unix(100, 0)
	snapshot := collector.Snapshot{
		Mode:      collector.ModeIdle,
		Status:    demoSnapshotForView(now),
		UpdatedAt: now,
		LastError: "",
	}

	model := NewModel(make(chan collector.Snapshot))
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 110, Height: 36})
	model = updated.(Model)
	updated, _ = model.Update(snapshotMsg(snapshot))
	model = updated.(Model)

	view := model.View()
	for _, want := range []string{
		"NEUROLINK",
		"STATUS SUMMARY",
		"Lobby / Matchmaking",
		"Crossplay Auth",
		"DEMO",
		"Demo data is not live Apex service status.",
		"▰",
		"q quit",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("view does not contain %q\n%s", want, view)
		}
	}
}

func demoSnapshotForView(now time.Time) statusapi.Snapshot {
	services := statusapi.CoreServices()
	for i := range services {
		services[i].Status = statusapi.StatusHealthy
		services[i].Summary = "running"
		services[i].UpdatedAt = now
		services[i].Regions = []statusapi.RegionStatus{{Name: "Asia", Status: statusapi.StatusHealthy, Label: "RUNNING", Latency: 15 * time.Millisecond, HasLatency: true, CheckedAt: now}}
	}
	services[1].Status = statusapi.StatusDegraded
	services[1].Summary = "one region degraded"
	return statusapi.Snapshot{
		Source:        statusapi.SourceDemo,
		Attribution:   statusapi.Attribution,
		GeneratedAt:   now,
		Overall:       statusapi.StatusDegraded,
		Services:      services,
		Notice:        "Demo data is not live Apex service status.",
		RecentReports: []statusapi.RecentReport{{Country: "JP", Issue: "Match making", Platform: "PC"}},
	}
}
