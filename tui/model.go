package tui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"neurolink/apex-server-monitor/collector"
)

type Model struct {
	metricsCh <-chan collector.MetricsSnapshot
	snapshot  collector.MetricsSnapshot
	ready     bool
	width     int
	height    int
	stopped   bool
}

type metricsMsg collector.MetricsSnapshot
type metricsClosedMsg struct{}

func NewModel(metricsCh <-chan collector.MetricsSnapshot) Model {
	return Model{metricsCh: metricsCh}
}

func (m Model) Init() tea.Cmd {
	return waitForMetrics(m.metricsCh)
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
	case metricsMsg:
		m.snapshot = collector.MetricsSnapshot(msg)
		m.ready = true
		return m, waitForMetrics(m.metricsCh)
	case metricsClosedMsg:
		m.stopped = true
		return m, tea.Quit
	}

	return m, nil
}

func (m Model) View() string {
	if m.stopped {
		return "apex-server-monitor stopped\n"
	}
	if !m.ready {
		return shellStyle.Render("Waiting for probe data...")
	}

	header := renderHeader(m.snapshot)
	body := m.renderClusters()
	footer := footerStyle.Render("q: quit")

	return shellStyle.Width(max(0, m.width-2)).Render(
		lipgloss.JoinVertical(lipgloss.Left, header, body, footer),
	)
}

// waitForMetrics is the channel bridge into Bubble Tea. The command blocks on
// the collector metrics channel outside Update, then returns a message. Update
// stores the snapshot and schedules another waitForMetrics command.
func waitForMetrics(ch <-chan collector.MetricsSnapshot) tea.Cmd {
	return func() tea.Msg {
		snapshot, ok := <-ch
		if !ok {
			return metricsClosedMsg{}
		}
		return metricsMsg(snapshot)
	}
}

func (m Model) renderClusters() string {
	keys := make([]collector.ClusterID, 0, len(m.snapshot.Clusters))
	for key := range m.snapshot.Clusters {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })

	panes := make([]string, 0, len(keys))
	for _, key := range keys {
		panes = append(panes, renderCluster(m.snapshot.Clusters[key], m.paneWidth()))
	}

	if len(panes) == 0 {
		return mutedStyle.Render("No targets configured.")
	}
	if m.width > 0 && m.width < 90 {
		return lipgloss.JoinVertical(lipgloss.Left, panes...)
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, panes...)
}

func (m Model) paneWidth() int {
	if m.width <= 0 {
		return 42
	}
	if m.width < 90 {
		return max(30, m.width-4)
	}
	return max(38, (m.width-6)/2)
}

func renderHeader(snapshot collector.MetricsSnapshot) string {
	mode := string(snapshot.Mode)
	if snapshot.BattleMode && snapshot.ProcessName != "" {
		mode = fmt.Sprintf("%s (%s)", mode, snapshot.ProcessName)
	}

	return headerStyle.Render("apex-server-monitor") + "\n" +
		mutedStyle.Render(fmt.Sprintf("Mode: %s  Updated: %s", mode, snapshot.UpdatedAt.Format("15:04:05")))
}

func renderCluster(metrics collector.ClusterMetrics, width int) string {
	status := statusStyle(metrics.Status).Render(string(metrics.Status))
	lines := []string{
		titleStyle.Render(metrics.Target),
		mutedStyle.Render(metrics.Address),
		"",
		fmt.Sprintf("Status      %s", status),
		fmt.Sprintf("Avg Latency %s", formatDuration(metrics.AvgLatency)),
		fmt.Sprintf("Packet Loss %.1f%%", metrics.PacketLoss),
		fmt.Sprintf("Jitter      %s", formatDuration(metrics.Jitter)),
		fmt.Sprintf("Signal      %s", signalBar(metrics.SignalBars)),
	}

	if metrics.LastError != "" && metrics.Status != collector.StatusOnline {
		lines = append(lines, "", errorStyle.Render(summarizeError(metrics.LastError, 72)))
	}

	return paneStyle.Width(width).Render(strings.Join(lines, "\n"))
}

func signalBar(bars int) string {
	if bars < 0 {
		bars = 0
	}
	if bars > 10 {
		bars = 10
	}
	return "📶 [" + strings.Repeat("█", bars) + strings.Repeat("░", 10-bars) + "]"
}

func summarizeError(value string, maxRunes int) string {
	value = strings.TrimSpace(value)
	runes := []rune(value)
	if len(runes) <= maxRunes {
		return value
	}
	return string(runes[:maxRunes-1]) + "…"
}

func formatDuration(d time.Duration) string {
	if d <= 0 {
		return "--"
	}
	return fmt.Sprintf("%.1f ms", float64(d.Microseconds())/1000)
}

func statusStyle(status collector.Status) lipgloss.Style {
	switch status {
	case collector.StatusOnline:
		return onlineStyle
	case collector.StatusHighLoss:
		return lossStyle
	default:
		return offlineStyle
	}
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

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("86"))

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("230"))

	mutedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("244"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("203"))

	onlineStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("42"))

	lossStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("214"))

	offlineStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("203"))

	paneStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("238")).
			Padding(1, 2).
			MarginRight(2).
			MarginTop(1)

	footerStyle = lipgloss.NewStyle().
			MarginTop(1).
			Foreground(lipgloss.Color("244"))
)
