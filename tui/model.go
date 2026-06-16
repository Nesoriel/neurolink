package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"neurolink/apex-server-monitor/collector"
	"neurolink/apex-server-monitor/statusapi"
)

type Model struct {
	snapshots <-chan collector.Snapshot
	snapshot  collector.Snapshot
	ready     bool
	width     int
	height    int
	stopped   bool
}

type snapshotMsg collector.Snapshot
type snapshotsClosedMsg struct{}

func NewModel(snapshots <-chan collector.Snapshot) Model {
	return Model{snapshots: snapshots}
}

func (m Model) Init() tea.Cmd {
	return waitForSnapshot(m.snapshots)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case snapshotMsg:
		m.snapshot = collector.Snapshot(msg)
		m.ready = true
		return m, waitForSnapshot(m.snapshots)
	case snapshotsClosedMsg:
		m.stopped = true
		return m, tea.Quit
	}

	return m, nil
}

func (m Model) View() string {
	if m.stopped {
		return "neurolink stopped\n"
	}

	width := contentWidth(m.width)
	if !m.ready {
		return shellStyle.Width(width).Render(loadingStyle.Render("NEUROLINK // acquiring service feed..."))
	}

	parts := []string{
		renderHeader(m.snapshot, width),
		renderDashboard(m.snapshot, width),
		renderFooter(m.snapshot, width),
	}

	return shellStyle.Width(width).Render(lipgloss.JoinVertical(lipgloss.Left, parts...))
}

func waitForSnapshot(ch <-chan collector.Snapshot) tea.Cmd {
	return func() tea.Msg {
		snapshot, ok := <-ch
		if !ok {
			return snapshotsClosedMsg{}
		}
		return snapshotMsg(snapshot)
	}
}

func renderHeader(snapshot collector.Snapshot, width int) string {
	mode := "● IDLE"
	if snapshot.BattleMode {
		mode = "● BATTLE"
	}
	if snapshot.ProcessName != "" {
		mode += " " + snapshot.ProcessName
	}

	title := titleStyle.Render("NEUROLINK")
	subtitle := mutedStyle.Render("Crypto surveillance drone / Apex service health")
	modeChip := modeStyle(snapshot.BattleMode).Render(fit(mode, 26))
	sourceChip := sourceStyle(snapshot.Status.Source).Render(string(snapshot.Status.Source))
	update := mutedStyle.Render("Last update " + formatClock(snapshot.UpdatedAt))

	meta := lipgloss.JoinHorizontal(lipgloss.Center, modeChip, " ", sourceChip, " ", update)
	line := title
	if width >= 78 {
		space := max(1, width-lipgloss.Width(title)-lipgloss.Width(meta)-4)
		line = title + strings.Repeat(" ", space) + meta
	}

	return headerBox.Width(width).Render(lipgloss.JoinVertical(lipgloss.Left, line, subtitle))
}

func renderDashboard(snapshot collector.Snapshot, width int) string {
	summary := renderSummary(snapshot, cardContentWidth(width))
	reports := renderReports(snapshot, cardContentWidth(width))

	top := lipgloss.JoinVertical(lipgloss.Left, summary, reports)
	if width >= 96 {
		col := twoColumnContentWidth(width)
		top = lipgloss.JoinHorizontal(lipgloss.Top, renderSummary(snapshot, col), "  ", renderReports(snapshot, col))
	}

	services := renderServiceGrid(snapshot.Status, width)
	return lipgloss.JoinVertical(lipgloss.Left, top, services)
}

func renderSummary(snapshot collector.Snapshot, width int) string {
	status := snapshot.Status.Overall
	down, degraded, unknown := serviceCounts(snapshot.Status.Services)

	lines := []string{
		sectionTitleStyle.Render("STATUS SUMMARY"),
		statusChip(status) + "  " + statusBar(status),
		fmt.Sprintf("Impacted services  %s", countStyle.Render(fmt.Sprintf("%d", down+degraded))),
		fmt.Sprintf("Down / degraded    %d / %d", down, degraded),
		fmt.Sprintf("Unknown checks     %d", unknown),
		fmt.Sprintf("API mode           %s", string(snapshot.Status.Source)),
	}
	if snapshot.Status.Source == statusapi.SourceDemo {
		lines = append(lines, "", warningStyle.Render("Demo data is not live Apex service status."))
	}

	return cardStyleFor(status).Width(width).Render(strings.Join(lines, "\n"))
}

func renderReports(snapshot collector.Snapshot, width int) string {
	lines := []string{sectionTitleStyle.Render("REPORTS / SOURCE")}
	if snapshot.Status.Notice != "" {
		lines = append(lines, warningStyle.Render(fit(snapshot.Status.Notice, width-6)))
	}
	if snapshot.LastError != "" {
		lines = append(lines, errorStyle.Render("Refresh issue: "+fit(summarizeError(snapshot.LastError), width-21)))
	}

	reports := snapshot.Status.RecentReports
	if len(reports) == 0 {
		lines = append(lines, mutedStyle.Render("No recent user-report feed in this payload."))
	} else {
		for i, report := range reports {
			if i >= 3 {
				break
			}
			lines = append(lines, renderReport(report, width-6))
		}
	}

	lines = append(lines, "", mutedStyle.Render(fit(snapshot.Status.Attribution, width-6)))
	return cardStyle.Width(width).Render(strings.Join(lines, "\n"))
}

func renderServiceGrid(snapshot statusapi.Snapshot, width int) string {
	services := orderedServices(snapshot)
	if width < 86 {
		cards := make([]string, 0, len(services))
		for _, service := range services {
			cards = append(cards, renderServiceCard(service, cardContentWidth(width)))
		}
		return lipgloss.JoinVertical(lipgloss.Left, cards...)
	}

	colWidth := twoColumnContentWidth(width)
	rows := make([]string, 0, (len(services)+1)/2)
	for i := 0; i < len(services); i += 2 {
		left := renderServiceCard(services[i], colWidth)
		if i+1 >= len(services) {
			rows = append(rows, left)
			continue
		}
		right := renderServiceCard(services[i+1], colWidth)
		rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, left, "  ", right))
	}
	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

func renderServiceCard(service statusapi.ServiceStatus, width int) string {
	regions := renderRegions(service.Regions, width-6)
	if regions == "" {
		regions = mutedStyle.Render("No regional detail")
	}

	lines := []string{
		serviceTitleStyle.Render(fit(service.Name, width-6)),
		statusChip(service.Status) + "  " + statusBar(service.Status),
		fit(service.Summary, width-6),
		mutedStyle.Render("Updated " + formatClock(service.UpdatedAt)),
		"",
		regions,
	}

	return cardStyleFor(service.Status).Width(width).Height(9).Render(strings.Join(lines, "\n"))
}

func renderRegions(regions []statusapi.RegionStatus, width int) string {
	if len(regions) == 0 {
		return ""
	}

	parts := make([]string, 0, min(4, len(regions)))
	for i, region := range regions {
		if i >= 4 {
			break
		}
		label := statusGlyph(region.Status) + " " + region.Name
		if region.HasLatency {
			label += fmt.Sprintf(" %dms", region.Latency.Milliseconds())
		}
		parts = append(parts, label)
	}
	if len(regions) > 4 {
		parts = append(parts, fmt.Sprintf("+%d", len(regions)-4))
	}
	return mutedStyle.Render(fit(strings.Join(parts, "  "), width))
}

func renderReport(report statusapi.RecentReport, width int) string {
	parts := []string{}
	for _, value := range []string{report.Country, report.Platform, report.Issue, report.ErrorCode, report.At} {
		if value != "" {
			parts = append(parts, value)
		}
	}
	if len(parts) == 0 {
		return mutedStyle.Render("Report received")
	}
	return fit("• "+strings.Join(parts, " · "), width)
}

func renderFooter(snapshot collector.Snapshot, width int) string {
	hint := "q quit  ctrl+c quit  --api-key or NEUROLINK_APEX_API_KEY for live status  --demo for sample data"
	if snapshot.Status.Source == statusapi.SourceLive {
		hint = "q quit  ctrl+c quit  --poll-interval controls refresh cadence"
	}
	return footerStyle.Width(width).Render(fit(hint, width))
}

func orderedServices(snapshot statusapi.Snapshot) []statusapi.ServiceStatus {
	ordered := statusapi.CoreServices()
	for i, service := range ordered {
		if actual, ok := statusapi.ServiceByID(snapshot, service.ID); ok {
			ordered[i] = actual
		}
	}
	return ordered
}

func serviceCounts(services []statusapi.ServiceStatus) (down int, degraded int, unknown int) {
	for _, service := range services {
		switch service.Status {
		case statusapi.StatusDown:
			down++
		case statusapi.StatusDegraded:
			degraded++
		case statusapi.StatusUnknown:
			unknown++
		}
	}
	return down, degraded, unknown
}

func statusChip(status statusapi.Status) string {
	return chipStyle(status).Render(statusLabel(status))
}

func statusLabel(status statusapi.Status) string {
	switch status {
	case statusapi.StatusHealthy:
		return "● RUNNING"
	case statusapi.StatusDegraded:
		return "▲ DEGRADED"
	case statusapi.StatusDown:
		return "✕ DOWN"
	default:
		return "? UNKNOWN"
	}
}

func statusGlyph(status statusapi.Status) string {
	switch status {
	case statusapi.StatusHealthy:
		return "●"
	case statusapi.StatusDegraded:
		return "▲"
	case statusapi.StatusDown:
		return "✕"
	default:
		return "?"
	}
}

func statusBar(status statusapi.Status) string {
	filled := 1
	switch status {
	case statusapi.StatusHealthy:
		filled = 5
	case statusapi.StatusDegraded:
		filled = 3
	case statusapi.StatusDown:
		filled = 0
	}
	return barStyle(status).Render(strings.Repeat("▰", filled) + strings.Repeat("▱", 5-filled))
}

func cardStyleFor(status statusapi.Status) lipgloss.Style {
	switch status {
	case statusapi.StatusHealthy:
		return cardStyle.BorderForeground(lipgloss.Color("42"))
	case statusapi.StatusDegraded:
		return cardStyle.BorderForeground(lipgloss.Color("214"))
	case statusapi.StatusDown:
		return cardStyle.BorderForeground(lipgloss.Color("203"))
	default:
		return cardStyle.BorderForeground(lipgloss.Color("244"))
	}
}

func chipStyle(status statusapi.Status) lipgloss.Style {
	style := lipgloss.NewStyle().Bold(true).Padding(0, 1)
	switch status {
	case statusapi.StatusHealthy:
		return style.Foreground(lipgloss.Color("16")).Background(lipgloss.Color("42"))
	case statusapi.StatusDegraded:
		return style.Foreground(lipgloss.Color("16")).Background(lipgloss.Color("214"))
	case statusapi.StatusDown:
		return style.Foreground(lipgloss.Color("230")).Background(lipgloss.Color("160"))
	default:
		return style.Foreground(lipgloss.Color("230")).Background(lipgloss.Color("238"))
	}
}

func barStyle(status statusapi.Status) lipgloss.Style {
	switch status {
	case statusapi.StatusHealthy:
		return healthyStyle
	case statusapi.StatusDegraded:
		return warningStyle
	case statusapi.StatusDown:
		return errorStyle
	default:
		return mutedStyle
	}
}

func sourceStyle(source statusapi.SourceMode) lipgloss.Style {
	if source == statusapi.SourceLive {
		return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("16")).Background(lipgloss.Color("81")).Padding(0, 1)
	}
	return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("16")).Background(lipgloss.Color("220")).Padding(0, 1)
}

func modeStyle(battle bool) lipgloss.Style {
	if battle {
		return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("16")).Background(lipgloss.Color("203")).Padding(0, 1)
	}
	return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("16")).Background(lipgloss.Color("42")).Padding(0, 1)
}

func summarizeError(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if strings.Contains(value, "HTTP 401") || strings.Contains(value, "HTTP 403") {
		return "API key rejected by status API"
	}
	if strings.Contains(value, "context canceled") {
		return "refresh canceled"
	}
	return fit(value, 72)
}

func formatClock(value time.Time) string {
	if value.IsZero() {
		return "--:--:--"
	}
	return value.Format("15:04:05")
}

func contentWidth(width int) int {
	if width <= 0 {
		return 100
	}
	return max(40, width-4)
}

func cardContentWidth(width int) int {
	return max(32, width-6)
}

func twoColumnContentWidth(width int) int {
	return max(34, (width-14)/2)
}

func fit(value string, width int) string {
	value = strings.TrimSpace(value)
	if width <= 0 {
		return ""
	}
	runes := []rune(value)
	if len(runes) <= width {
		return value
	}
	if width == 1 {
		return "…"
	}
	return string(runes[:width-1]) + "…"
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

var (
	shellStyle = lipgloss.NewStyle().
			Padding(1, 2)

	headerBox = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), false, false, true, false).
			BorderForeground(lipgloss.Color("30")).
			PaddingBottom(1).
			MarginBottom(1)

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("86"))

	sectionTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("81"))

	serviceTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("230"))

	mutedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("244"))

	healthyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("42"))

	warningStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("214"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("203"))

	countStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("230"))

	cardStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("238")).
			Padding(1, 2).
			MarginBottom(1)

	footerStyle = lipgloss.NewStyle().
			MarginTop(1).
			Foreground(lipgloss.Color("244"))

	loadingStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("86")).
			Bold(true)
)
