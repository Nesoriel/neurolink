package tui

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Nesoriel/neurolink/collector"
	"github.com/Nesoriel/neurolink/playerapi"
	"github.com/Nesoriel/neurolink/statusapi"
)

type Model struct {
	snapshots     <-chan collector.Snapshot
	statusRefresh chan<- struct{}
	player        playerapi.Provider
	snapshot      collector.Snapshot
	copy          lexicon
	ready         bool
	width         int
	height        int
	stopped       bool
	view          appView
	now           time.Time
	tick          int

	history []statusHistorySample

	playerInput       string
	playerFocused     bool
	playerPlatform    playerapi.Platform
	playerLoading     bool
	playerLookupSeq   int
	playerLastRequest playerapi.LookupRequest
	playerResult      playerapi.PlayerSnapshot
	playerErr         error
	playerSearched    bool
	refreshRequested  bool
}

type snapshotMsg collector.Snapshot
type snapshotsClosedMsg struct{}
type tickMsg time.Time
type statusRefreshRequestedMsg struct{}
type statusRefreshUnavailableMsg struct{}

type playerLookupMsg struct {
	seq      int
	request  playerapi.LookupRequest
	snapshot playerapi.PlayerSnapshot
	err      error
}

type appView int

const (
	viewDashboard appView = iota
	viewPlayer
	viewConfig
	viewHelp
)

type statusHistorySample struct {
	At       time.Time
	Overall  statusapi.Status
	Services map[statusapi.ServiceID]statusapi.Status
}

const maxHistorySamples = 24

func NewModel(snapshots <-chan collector.Snapshot, languages ...Language) Model {
	return NewModelWithPlayerProvider(snapshots, playerapi.NewDemoProvider(), nil, languages...)
}

func NewModelWithPlayerProvider(snapshots <-chan collector.Snapshot, player playerapi.Provider, statusRefresh chan<- struct{}, languages ...Language) Model {
	language := LanguageEnglish
	if len(languages) > 0 {
		language = languages[0]
	}
	if player == nil {
		player = playerapi.NewDemoProvider()
	}
	return Model{
		snapshots:      snapshots,
		statusRefresh:  statusRefresh,
		player:         player,
		copy:           lexiconFor(language),
		view:           viewDashboard,
		now:            time.Now(),
		playerPlatform: playerapi.PlatformPC,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(waitForSnapshot(m.snapshots), tickEverySecond())
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(msg)
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case snapshotMsg:
		m.snapshot = collector.Snapshot(msg)
		m.ready = true
		m.now = time.Now()
		m.history = appendStatusHistory(m.history, m.snapshot)
		m.refreshRequested = false
		return m, waitForSnapshot(m.snapshots)
	case snapshotsClosedMsg:
		m.stopped = true
		return m, tea.Quit
	case tickMsg:
		m.now = time.Time(msg)
		m.tick++
		return m, tickEverySecond()
	case statusRefreshRequestedMsg:
		m.refreshRequested = true
	case statusRefreshUnavailableMsg:
		m.refreshRequested = false
	case playerLookupMsg:
		if msg.seq != m.playerLookupSeq {
			return m, nil
		}
		m.playerLoading = false
		m.playerSearched = true
		m.playerLastRequest = msg.request
		m.playerResult = msg.snapshot
		m.playerErr = msg.err
	}

	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	if m.view == viewPlayer && m.playerFocused {
		switch key {
		case "ctrl+c":
			return m, tea.Quit
		case "tab":
			m.view = nextView(m.view)
			m.playerFocused = false
			return m, nil
		case "shift+tab":
			m.view = previousView(m.view)
			m.playerFocused = false
			return m, nil
		case "?":
			m.view = viewHelp
			m.playerFocused = false
			return m, nil
		}
		return m.handlePlayerKey(msg)
	}

	switch key {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "tab":
		m.view = nextView(m.view)
		m.playerFocused = m.view == viewPlayer && m.playerFocused
		return m, nil
	case "shift+tab":
		m.view = previousView(m.view)
		m.playerFocused = m.view == viewPlayer && m.playerFocused
		return m, nil
	case "1":
		m.view = viewDashboard
		m.playerFocused = false
		return m, nil
	case "2":
		m.view = viewPlayer
		return m, nil
	case "3":
		m.view = viewConfig
		m.playerFocused = false
		return m, nil
	case "?":
		m.view = viewHelp
		m.playerFocused = false
		return m, nil
	case "/":
		m.view = viewPlayer
		m.playerFocused = true
		return m, nil
	case "r":
		return m, requestStatusRefresh(m.statusRefresh)
	}

	if m.view == viewPlayer {
		return m.handlePlayerKey(msg)
	}
	return m, nil
}

func (m Model) handlePlayerKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	switch key {
	case "esc":
		m.playerFocused = false
		return m, nil
	case "enter":
		return m.startPlayerLookup()
	case "p":
		if !m.playerFocused {
			m.playerPlatform = nextPlatform(m.playerPlatform)
			m.playerErr = nil
			return m, nil
		}
	case "backspace", "ctrl+h":
		if m.playerFocused && len(m.playerInput) > 0 {
			runes := []rune(m.playerInput)
			m.playerInput = string(runes[:len(runes)-1])
			m.playerErr = nil
		}
		return m, nil
	case "delete", "ctrl+u":
		if m.playerFocused {
			m.playerInput = ""
			m.playerErr = nil
		}
		return m, nil
	}

	if m.playerFocused && msg.Type == tea.KeyRunes {
		m.playerInput += string(msg.Runes)
		m.playerErr = nil
		return m, nil
	}
	return m, nil
}

func (m Model) startPlayerLookup() (tea.Model, tea.Cmd) {
	request := playerapi.LookupRequest{Player: strings.TrimSpace(m.playerInput), Platform: m.playerPlatform}
	if request.Player == "" {
		m.playerErr = playerapi.ErrPlayerRequired
		m.playerSearched = true
		m.playerResult = playerapi.PlayerSnapshot{}
		return m, nil
	}
	m.playerLookupSeq++
	m.playerLoading = true
	m.playerSearched = true
	m.playerErr = nil
	m.playerResult = playerapi.PlayerSnapshot{}
	m.playerLastRequest = request
	return m, lookupPlayer(m.player, request, m.playerLookupSeq)
}

func (m Model) View() string {
	if m.stopped {
		return m.copy.stopped
	}

	width := contentWidth(m.width)
	if !m.ready {
		return shellStyle.Width(width).Render(loadingStyle.Render(m.copy.loading))
	}

	parts := []string{
		renderHeader(m.snapshot, width, m.copy, m.view),
		renderTabs(width, m.copy, m.view),
		m.renderActiveView(width),
		renderFooter(m.snapshot, width, m.copy, m.view),
	}

	return shellStyle.Width(width).Render(lipgloss.JoinVertical(lipgloss.Left, parts...))
}

func (m Model) renderActiveView(width int) string {
	switch m.view {
	case viewPlayer:
		return renderPlayerLookup(m, width)
	case viewConfig:
		return renderConfigView(m.snapshot, width, m.copy)
	case viewHelp:
		return renderHelpView(width, m.copy)
	default:
		return renderDashboard(m.snapshot, width, m.copy, m.history, m.now, m.refreshRequested)
	}
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

func tickEverySecond() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func lookupPlayer(provider playerapi.Provider, request playerapi.LookupRequest, seq int) tea.Cmd {
	return func() tea.Msg {
		snapshot, err := provider.Lookup(context.Background(), request)
		return playerLookupMsg{seq: seq, request: request, snapshot: snapshot, err: err}
	}
}

func requestStatusRefresh(refresh chan<- struct{}) tea.Cmd {
	return func() tea.Msg {
		if refresh == nil {
			return statusRefreshUnavailableMsg{}
		}
		select {
		case refresh <- struct{}{}:
			return statusRefreshRequestedMsg{}
		default:
			return statusRefreshRequestedMsg{}
		}
	}
}

func renderHeader(snapshot collector.Snapshot, width int, copy lexicon, view appView) string {
	mode := copy.modeLabel(snapshot.BattleMode)
	if snapshot.ProcessName != "" {
		mode += " " + snapshot.ProcessName
	}

	title := titleStyle.Render("NEUROLINK")
	subtitle := mutedStyle.Render(copy.headerSubtitle)
	viewChip := selectedTabStyle.Render(fit(copy.viewLabel(view), 18))
	modeChip := modeStyle(snapshot.BattleMode).Render(fit(mode, 26))
	sourceChip := sourceStyle(snapshot.Status.Source).Render(string(snapshot.Status.Source))
	update := mutedStyle.Render(copy.lastUpdatePrefix + formatClock(snapshot.UpdatedAt))

	meta := lipgloss.JoinHorizontal(lipgloss.Center, viewChip, " ", modeChip, " ", sourceChip, " ", update)
	line := title
	if width >= 78 {
		space := max(1, width-lipgloss.Width(title)-lipgloss.Width(meta)-4)
		line = title + strings.Repeat(" ", space) + meta
	}

	return headerBox.Width(width).Render(lipgloss.JoinVertical(lipgloss.Left, line, subtitle))
}

func renderTabs(width int, copy lexicon, selected appView) string {
	views := []appView{viewDashboard, viewPlayer, viewConfig, viewHelp}
	parts := make([]string, 0, len(views))
	for _, view := range views {
		label := copy.viewKeyLabel(view)
		if view == selected {
			parts = append(parts, selectedTabStyle.Render(label))
			continue
		}
		parts = append(parts, tabStyle.Render(label))
	}
	return fit(lipgloss.JoinHorizontal(lipgloss.Center, parts...), width)
}

func renderDashboard(snapshot collector.Snapshot, width int, copy lexicon, history []statusHistorySample, now time.Time, refreshRequested bool) string {
	summary := renderSummary(snapshot, cardContentWidth(width), copy, history, now, refreshRequested)
	reports := renderReports(snapshot, cardContentWidth(width), copy)

	top := lipgloss.JoinVertical(lipgloss.Left, summary, reports)
	if width >= 96 {
		col := twoColumnContentWidth(width)
		top = lipgloss.JoinHorizontal(lipgloss.Top, renderSummary(snapshot, col, copy, history, now, refreshRequested), "  ", renderReports(snapshot, col, copy))
	}

	services := renderServiceGrid(snapshot.Status, width, copy, history)
	return lipgloss.JoinVertical(lipgloss.Left, top, services)
}

func renderSummary(snapshot collector.Snapshot, width int, copy lexicon, history []statusHistorySample, now time.Time, refreshRequested bool) string {
	status := snapshot.Status.Overall
	down, degraded, unknown := serviceCounts(snapshot.Status.Services)
	refreshAge := now.Sub(snapshot.UpdatedAt)
	if now.IsZero() || snapshot.UpdatedAt.IsZero() || refreshAge < 0 {
		refreshAge = 0
	}

	lines := []string{
		sectionTitleStyle.Render(copy.statusSummaryTitle),
		statusChip(status, copy) + "  " + statusBar(status),
		summaryLine(copy.refreshAge, formatDuration(refreshAge)),
		summaryLine(copy.historyTrend, trendForOverall(history)),
		summaryLine(copy.impactedServices, countStyle.Render(fmt.Sprintf("%d", down+degraded))),
		summaryLine(copy.downDegraded, fmt.Sprintf("%d / %d", down, degraded)),
		summaryLine(copy.unknownChecks, fmt.Sprintf("%d", unknown)),
		summaryLine(copy.apiMode, string(snapshot.Status.Source)),
	}
	if refreshRequested {
		lines = append(lines, warningStyle.Render(copy.refreshQueued))
	}
	if snapshot.Status.Source == statusapi.SourceDemo {
		lines = append(lines, "", warningStyle.Render(copy.demoDataNotLive))
	}

	return cardStyleFor(status).Width(width).Render(strings.Join(lines, "\n"))
}

func renderReports(snapshot collector.Snapshot, width int, copy lexicon) string {
	lines := []string{sectionTitleStyle.Render(copy.communityPulseTitle)}
	if snapshot.Status.Notice != "" {
		lines = append(lines, warningStyle.Render(fit(copy.notice(snapshot.Status.Notice), width-6)))
	}
	if snapshot.LastError != "" {
		lines = append(lines, errorStyle.Render(copy.refreshIssuePrefix+fit(copy.summarizeError(snapshot.LastError), width-lipgloss.Width(copy.refreshIssuePrefix)-6)))
	}

	reports := snapshot.Status.RecentReports
	if len(reports) == 0 {
		lines = append(lines,
			mutedStyle.Render(fit(copy.noReportsLine1, width-6)),
			mutedStyle.Render(fit(copy.noReportsLine2, width-6)),
		)
	} else {
		for i, report := range reports {
			if i >= 3 {
				break
			}
			lines = append(lines, renderReport(report, width-6, copy))
		}
	}

	if attribution := copy.attributionLine(snapshot.Status.Attribution); attribution != "" {
		lines = append(lines, "", mutedStyle.Render(fit(attribution, width-6)))
	}
	return cardStyle.Width(width).Render(strings.Join(lines, "\n"))
}

func renderServiceGrid(snapshot statusapi.Snapshot, width int, copy lexicon, history []statusHistorySample) string {
	services := orderedServices(snapshot)
	if width < 86 {
		cards := make([]string, 0, len(services))
		for _, service := range services {
			cards = append(cards, renderServiceCard(service, cardContentWidth(width), copy, history))
		}
		return lipgloss.JoinVertical(lipgloss.Left, cards...)
	}

	colWidth := twoColumnContentWidth(width)
	rows := make([]string, 0, (len(services)+1)/2)
	for i := 0; i < len(services); i += 2 {
		left := renderServiceCard(services[i], colWidth, copy, history)
		if i+1 >= len(services) {
			rows = append(rows, left)
			continue
		}
		right := renderServiceCard(services[i+1], colWidth, copy, history)
		rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, left, "  ", right))
	}
	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

func renderServiceCard(service statusapi.ServiceStatus, width int, copy lexicon, history []statusHistorySample) string {
	regions := renderRegions(service.Regions, width-6)
	if regions == "" {
		regions = mutedStyle.Render(copy.noRegionalDetail)
	}

	lines := []string{
		serviceTitleStyle.Render(fit(service.Name, width-6)),
		statusChip(service.Status, copy) + "  " + statusBar(service.Status),
		fit(service.Summary, width-6),
		mutedStyle.Render(copy.historyTrend + " " + trendForService(history, service.ID)),
		mutedStyle.Render(copy.updatedPrefix + formatClock(service.UpdatedAt)),
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

func renderReport(report statusapi.RecentReport, width int, copy lexicon) string {
	parts := []string{}
	for _, value := range []string{report.Country, report.Platform, report.Issue, report.ErrorCode, report.At} {
		if value != "" {
			parts = append(parts, value)
		}
	}
	if len(parts) == 0 {
		return mutedStyle.Render(copy.reportReceived)
	}
	return fit("• "+strings.Join(parts, " · "), width)
}

func renderFooter(snapshot collector.Snapshot, width int, copy lexicon, view appView) string {
	hint := copy.footerHint(view, snapshot.Status.Source)
	return footerStyle.Width(width).Render(fit(hint, width))
}

func renderPlayerLookup(m Model, width int) string {
	cardWidth := cardContentWidth(width)
	form := renderPlayerForm(m, cardWidth)
	side := renderPlayerState(m, cardWidth)
	if width >= 100 {
		col := twoColumnContentWidth(width)
		form = renderPlayerForm(m, col)
		side = renderPlayerState(m, col)
		return lipgloss.JoinHorizontal(lipgloss.Top, form, "  ", side)
	}
	return lipgloss.JoinVertical(lipgloss.Left, form, side)
}

func renderPlayerForm(m Model, cardWidth int) string {
	input := renderPlayerInput(m, max(16, cardWidth-10), m.copy)
	platforms := renderPlatformSelector(m.playerPlatform)

	lines := []string{
		sectionTitleStyle.Render(m.copy.playerLookupTitle),
		mutedStyle.Render(fit(m.copy.playerLookupSubtitle, cardWidth-6)),
		"",
		m.copy.playerInputLabel,
		input,
		m.copy.playerPlatformLabel + "  " + platforms,
		mutedStyle.Render(fit(m.copy.pcOriginNote, cardWidth-6)),
	}
	if m.playerLoading {
		lines = append(lines, "", loadingStyle.Render(spinnerFrame(m.tick)+" "+m.copy.playerLoading))
	}
	return cardStyle.Width(cardWidth).Render(strings.Join(lines, "\n"))
}

func renderPlayerInput(m Model, width int, copy lexicon) string {
	value := m.playerInput
	if value == "" {
		value = mutedStyle.Render(copy.playerPlaceholder)
	}
	if m.playerFocused {
		value += focusedCursorStyle.Render("█")
		return focusedInputStyle.Width(width).Render(fit(value, width))
	}
	return inputStyle.Width(width).Render(fit(value, width))
}

func renderPlatformSelector(selected playerapi.Platform) string {
	parts := make([]string, 0, len(playerapi.SupportedPlatforms()))
	for _, platform := range playerapi.SupportedPlatforms() {
		label := string(platform)
		if platform == selected {
			parts = append(parts, selectedPlatformStyle.Render(label))
			continue
		}
		parts = append(parts, platformStyle.Render(label))
	}
	return lipgloss.JoinHorizontal(lipgloss.Center, parts...)
}

func renderPlayerState(m Model, width int) string {
	lines := []string{sectionTitleStyle.Render(m.copy.playerResultTitle)}
	if !m.playerSearched {
		lines = append(lines,
			mutedStyle.Render(fit(m.copy.playerIdleLine1, width-6)),
			mutedStyle.Render(fit(m.copy.playerIdleLine2, width-6)),
		)
		return cardStyle.Width(width).Render(strings.Join(lines, "\n"))
	}
	if m.playerErr != nil {
		lines[0] = sectionTitleStyle.Render(m.copy.playerErrorTitle)
		lines = append(lines, errorStyle.Render(fit(m.copy.playerError(m.playerErr), width-6)))
		if errors.Is(m.playerErr, playerapi.ErrPlayerNotFound) {
			lines = append(lines, mutedStyle.Render(fit(m.copy.pcOriginNote, width-6)))
		}
		return cardStyle.BorderForeground(lipgloss.Color("203")).Width(width).Render(strings.Join(lines, "\n"))
	}
	if m.playerLoading {
		lines = append(lines, loadingStyle.Render(spinnerFrame(m.tick)+" "+m.copy.playerLoading))
		return cardStyle.Width(width).Render(strings.Join(lines, "\n"))
	}

	snapshot := m.playerResult
	lines = append(lines,
		serviceTitleStyle.Render(fit(snapshot.PlayerName, width-6)),
		m.copy.playerSource+"  "+playerSourceStyle(snapshot.Source).Render(string(snapshot.Source)),
	)
	if snapshot.UID != "" {
		lines = append(lines, summaryLine(m.copy.playerUID, snapshot.UID))
	}
	lines = append(lines, summaryLine(m.copy.playerPlatform, string(snapshot.Platform)))
	if snapshot.HasLevel {
		lines = append(lines, summaryLine(m.copy.playerLevel, fmt.Sprintf("%d", snapshot.Level)))
	}
	if snapshot.HasRank {
		rank := strings.TrimSpace(snapshot.RankName + " " + snapshot.RankDivision)
		if snapshot.RankScore > 0 {
			rank += fmt.Sprintf(" / %d", snapshot.RankScore)
		}
		lines = append(lines, summaryLine(m.copy.playerRank, rank))
	}
	if snapshot.SelectedLegend != "" {
		lines = append(lines, summaryLine(m.copy.playerLegend, snapshot.SelectedLegend))
	}
	if len(snapshot.Trackers) > 0 {
		lines = append(lines, "", m.copy.playerTrackers)
		for i, tracker := range snapshot.Trackers {
			if i >= 5 {
				break
			}
			lines = append(lines, fit("• "+tracker.Name+"  "+tracker.Value, width-6))
		}
	}
	if snapshot.Notice != "" {
		lines = append(lines, "", warningStyle.Render(fit(m.copy.notice(snapshot.Notice), width-6)))
	}
	if attribution := m.copy.attributionLine(snapshot.Attribution); attribution != "" {
		lines = append(lines, mutedStyle.Render(fit(attribution, width-6)))
	}
	return cardStyle.Width(width).Render(strings.Join(lines, "\n"))
}

func renderConfigView(snapshot collector.Snapshot, width int, copy lexicon) string {
	lines := []string{
		sectionTitleStyle.Render(copy.configTitle),
		summaryLine(copy.configSource, string(snapshot.Status.Source)),
		summaryLine(copy.configLanguage, string(copy.language)),
		"",
		fit(copy.configAPIKey, width-6),
		fit(copy.configEnv, width-6),
		fit(copy.configDemo, width-6),
	}
	if snapshot.Status.Source == statusapi.SourceDemo {
		lines = append(lines, "", warningStyle.Render(copy.demoDataNotLive))
	}
	return cardStyle.Width(cardContentWidth(width)).Render(strings.Join(lines, "\n"))
}

func renderHelpView(width int, copy lexicon) string {
	lines := []string{
		sectionTitleStyle.Render(copy.helpTitle),
		copy.helpDashboard,
		copy.helpPlayer,
		copy.helpConfig,
		copy.helpHelp,
		copy.helpSearch,
		copy.helpPlatform,
		copy.helpRefresh,
		copy.helpQuit,
		"",
		mutedStyle.Render(fit(copy.helpPrivacy, width-6)),
	}
	return cardStyle.Width(cardContentWidth(width)).Render(strings.Join(lines, "\n"))
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

func nextView(view appView) appView {
	switch view {
	case viewDashboard:
		return viewPlayer
	case viewPlayer:
		return viewConfig
	case viewConfig:
		return viewHelp
	default:
		return viewDashboard
	}
}

func previousView(view appView) appView {
	switch view {
	case viewHelp:
		return viewConfig
	case viewConfig:
		return viewPlayer
	case viewPlayer:
		return viewDashboard
	default:
		return viewHelp
	}
}

func nextPlatform(platform playerapi.Platform) playerapi.Platform {
	platforms := playerapi.SupportedPlatforms()
	for i, candidate := range platforms {
		if candidate == platform {
			return platforms[(i+1)%len(platforms)]
		}
	}
	return playerapi.PlatformPC
}

func appendStatusHistory(history []statusHistorySample, snapshot collector.Snapshot) []statusHistorySample {
	sample := statusHistorySample{
		At:       snapshot.UpdatedAt,
		Overall:  snapshot.Status.Overall,
		Services: map[statusapi.ServiceID]statusapi.Status{},
	}
	for _, service := range snapshot.Status.Services {
		sample.Services[service.ID] = service.Status
	}
	history = append(history, sample)
	if len(history) > maxHistorySamples {
		history = history[len(history)-maxHistorySamples:]
	}
	return history
}

func trendForOverall(history []statusHistorySample) string {
	if len(history) == 0 {
		return mutedStyle.Render("▱")
	}
	statuses := make([]statusapi.Status, 0, len(history))
	for _, sample := range lastHistory(history, 12) {
		statuses = append(statuses, sample.Overall)
	}
	return renderTrend(statuses)
}

func trendForService(history []statusHistorySample, id statusapi.ServiceID) string {
	if len(history) == 0 {
		return mutedStyle.Render("▱")
	}
	statuses := make([]statusapi.Status, 0, len(history))
	for _, sample := range lastHistory(history, 12) {
		status, ok := sample.Services[id]
		if !ok {
			status = statusapi.StatusUnknown
		}
		statuses = append(statuses, status)
	}
	return renderTrend(statuses)
}

func lastHistory(history []statusHistorySample, limit int) []statusHistorySample {
	if len(history) <= limit {
		return history
	}
	return history[len(history)-limit:]
}

func renderTrend(statuses []statusapi.Status) string {
	parts := make([]string, 0, len(statuses))
	for _, status := range statuses {
		glyph := "▱"
		switch status {
		case statusapi.StatusHealthy:
			glyph = "▰"
		case statusapi.StatusDegraded:
			glyph = "▅"
		case statusapi.StatusDown:
			glyph = "▁"
		}
		parts = append(parts, barStyle(status).Render(glyph))
	}
	return strings.Join(parts, "")
}

func formatDuration(duration time.Duration) string {
	if duration < time.Second {
		return "0s"
	}
	if duration < time.Minute {
		return fmt.Sprintf("%ds", int(duration.Seconds()))
	}
	if duration < time.Hour {
		return fmt.Sprintf("%dm%02ds", int(duration.Minutes()), int(duration.Seconds())%60)
	}
	return fmt.Sprintf("%dh%02dm", int(duration.Hours()), int(duration.Minutes())%60)
}

func spinnerFrame(tick int) string {
	frames := []string{"◐", "◓", "◑", "◒"}
	return frames[tick%len(frames)]
}

func summaryLine(label string, value string) string {
	return label + "  " + value
}

func statusChip(status statusapi.Status, copy lexicon) string {
	return chipStyle(status).Render(copy.statusLabel(status))
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

func playerSourceStyle(source playerapi.SourceMode) lipgloss.Style {
	if source == playerapi.SourceLive {
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
	if lipgloss.Width(value) <= width {
		return value
	}
	if width == 1 {
		return "…"
	}

	var builder strings.Builder
	for _, r := range value {
		next := string(r)
		if lipgloss.Width(builder.String())+lipgloss.Width(next) > width-1 {
			break
		}
		builder.WriteRune(r)
	}
	return builder.String() + "…"
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

	tabStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("244")).
			Padding(0, 1)

	selectedTabStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("16")).
				Background(lipgloss.Color("86")).
				Padding(0, 1)

	inputStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("238")).
			Padding(0, 1)

	focusedInputStyle = inputStyle.Copy().
				BorderForeground(lipgloss.Color("86"))

	focusedCursorStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("86"))

	platformStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("244")).
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("238")).
			Padding(0, 1)

	selectedPlatformStyle = platformStyle.Copy().
				Bold(true).
				Foreground(lipgloss.Color("16")).
				Background(lipgloss.Color("86")).
				BorderForeground(lipgloss.Color("86"))
)
