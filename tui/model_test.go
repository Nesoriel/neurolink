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
		"COMMUNITY PULSE",
		"Lobby / Matchmaking",
		"Crossplay Auth",
		"DEMO",
		"Demo data is not live Apex service status.",
		"JP",
		"▰",
		"q quit",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("view does not contain %q\n%s", want, view)
		}
	}
}

func TestViewRendersChineseDashboardAndMissingReportsExplanation(t *testing.T) {
	now := time.Unix(100, 0)
	status := demoSnapshotForView(now)
	status.RecentReports = nil
	status.Notice = "Demo mode: deterministic sample data, not live Apex service status"
	snapshot := collector.Snapshot{
		Mode:      collector.ModeIdle,
		Status:    status,
		UpdatedAt: now,
		LastError: "status API returned HTTP 403",
	}

	model := NewModel(make(chan collector.Snapshot), LanguageSimplifiedChinese)
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 132, Height: 36})
	model = updated.(Model)
	updated, _ = model.Update(snapshotMsg(snapshot))
	model = updated.(Model)

	view := model.View()
	for _, want := range []string{
		"NEUROLINK",
		"状态总览",
		"用户反馈",
		"演示数据不是实时 Apex 服务状态",
		"/servers 可能不包含用户反馈 feed",
		"不查询玩家资料或玩家状态",
		"状态 API 拒绝了 API key",
		"来源：apexlegendsstatus.com",
		"q 退出",
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
