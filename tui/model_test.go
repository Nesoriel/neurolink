package tui

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Nesoriel/neurolink/collector"
	"github.com/Nesoriel/neurolink/playerapi"
	"github.com/Nesoriel/neurolink/statusapi"
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
		"Refresh age",
		"Trend",
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

func TestPlayerLookupViewRunsAsyncLookup(t *testing.T) {
	now := time.Unix(100, 0)
	snapshot := collector.Snapshot{
		Mode:      collector.ModeIdle,
		Status:    demoSnapshotForView(now),
		UpdatedAt: now,
	}
	provider := &fakePlayerProvider{snapshot: playerapi.PlayerSnapshot{
		Source:         playerapi.SourceLive,
		Attribution:    playerapi.Attribution,
		LookupAt:       now,
		PlayerName:     "Ace Origin",
		Platform:       playerapi.PlatformPC,
		UID:            "uid-1",
		Level:          99,
		HasLevel:       true,
		RankName:       "Gold",
		RankDivision:   "1",
		RankScore:      7600,
		HasRank:        true,
		SelectedLegend: "Crypto",
		Trackers:       []playerapi.Tracker{{Name: "Kills", Value: "123"}},
	}}

	model := NewModelWithPlayerProvider(make(chan collector.Snapshot), provider, nil)
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 110, Height: 36})
	model = updated.(Model)
	updated, _ = model.Update(snapshotMsg(snapshot))
	model = updated.(Model)
	updated, _ = model.Update(keyMsg("/"))
	model = updated.(Model)
	for _, r := range []rune("Ace Origin") {
		updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		model = updated.(Model)
	}

	updated, cmd := model.Update(keyMsg("enter"))
	model = updated.(Model)
	if cmd == nil {
		t.Fatal("enter should return async lookup command")
	}
	msg := cmd()
	updated, _ = model.Update(msg)
	model = updated.(Model)

	if provider.last.Player != "Ace Origin" || provider.last.Platform != playerapi.PlatformPC {
		t.Fatalf("lookup request = %#v, want typed player on PC", provider.last)
	}
	view := model.View()
	for _, want := range []string{
		"PLAYER LOOKUP",
		"Ace Origin",
		"uid-1",
		"Gold 1 / 7600",
		"Crypto",
		"Kills",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("view does not contain %q\n%s", want, view)
		}
	}
}

func TestPlayerLookupChineseErrorRendering(t *testing.T) {
	now := time.Unix(100, 0)
	snapshot := collector.Snapshot{
		Mode:      collector.ModeIdle,
		Status:    demoSnapshotForView(now),
		UpdatedAt: now,
	}
	provider := &fakePlayerProvider{err: playerapi.ErrInvalidAPIKey}

	model := NewModelWithPlayerProvider(make(chan collector.Snapshot), provider, nil, LanguageSimplifiedChinese)
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 140, Height: 34})
	model = updated.(Model)
	updated, _ = model.Update(snapshotMsg(snapshot))
	model = updated.(Model)
	updated, _ = model.Update(keyMsg("/"))
	model = updated.(Model)
	for _, r := range []rune("玩家") {
		updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		model = updated.(Model)
	}
	updated, cmd := model.Update(keyMsg("enter"))
	model = updated.(Model)
	updated, _ = model.Update(cmd())
	model = updated.(Model)

	view := model.View()
	for _, want := range []string{
		"玩家查询",
		"查询错误",
		"玩家查询 API 拒绝了 API key",
		"不会后台跟踪玩家",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("view does not contain %q\n%s", want, view)
		}
	}
}

func TestNavigationHelpConfigAndRefreshKey(t *testing.T) {
	now := time.Unix(100, 0)
	snapshot := collector.Snapshot{
		Mode:      collector.ModeIdle,
		Status:    demoSnapshotForView(now),
		UpdatedAt: now,
	}
	refresh := make(chan struct{}, 1)
	model := NewModelWithPlayerProvider(make(chan collector.Snapshot), &fakePlayerProvider{}, refresh)
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 140, Height: 34})
	model = updated.(Model)
	updated, _ = model.Update(snapshotMsg(snapshot))
	model = updated.(Model)

	updated, cmd := model.Update(keyMsg("r"))
	model = updated.(Model)
	if cmd == nil {
		t.Fatal("r should return refresh command")
	}
	updated, _ = model.Update(cmd())
	model = updated.(Model)
	select {
	case <-refresh:
	default:
		t.Fatal("refresh channel did not receive request")
	}
	if !strings.Contains(model.View(), "Refresh requested") {
		t.Fatalf("dashboard did not show refresh request\n%s", model.View())
	}

	updated, _ = model.Update(keyMsg("3"))
	model = updated.(Model)
	if view := model.View(); !strings.Contains(view, "CONFIG") || !strings.Contains(view, "NEUROLINK_APEX_API_KEY") {
		t.Fatalf("config view missing expected content\n%s", view)
	}
	updated, _ = model.Update(keyMsg("/"))
	model = updated.(Model)
	updated, _ = model.Update(keyMsg("?"))
	model = updated.(Model)
	if view := model.View(); !strings.Contains(view, "HELP") || !strings.Contains(view, "does not continuously track players") {
		t.Fatalf("help view missing expected content\n%s", view)
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

type fakePlayerProvider struct {
	snapshot playerapi.PlayerSnapshot
	err      error
	last     playerapi.LookupRequest
}

func (p *fakePlayerProvider) Lookup(ctx context.Context, request playerapi.LookupRequest) (playerapi.PlayerSnapshot, error) {
	p.last = request
	if p.err != nil {
		return playerapi.PlayerSnapshot{}, p.err
	}
	if p.snapshot.PlayerName == "" {
		return playerapi.PlayerSnapshot{}, errors.New("missing test player snapshot")
	}
	return p.snapshot, nil
}

func keyMsg(value string) tea.KeyMsg {
	switch value {
	case "/":
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}}
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "r":
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}}
	case "3":
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}}
	case "?":
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}}
	default:
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(value)}
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
