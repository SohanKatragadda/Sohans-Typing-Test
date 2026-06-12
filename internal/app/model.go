package app

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"typing-test-tui/internal/history"
	"typing-test-tui/internal/prompts"
	"typing-test-tui/internal/typing"
)

type screenState int

const (
	screenSplash screenState = iota
	screenHome
	screenCountdown
	screenRunning
)

type homeTab int

const (
	tabTests homeTab = iota
	tabPastResults
)

type selectionFocus int

const (
	focusDuration selectionFocus = iota
	focusLanguage
	focusStart
)

type tickMsg time.Time

type Model struct {
	historyPath string
	history     []history.Run
	table       table.Model
	session     typing.Session

	screen            screenState
	tab               homeTab
	focus             selectionFocus
	durationIndex     int
	languageIndex     int
	durationConfirmed bool
	languageConfirmed bool
	width             int
	height            int
	splashStarted     time.Time
	countdownStarted  time.Time
	resultSaved       bool
	lastMessage       string
	historyLoadErr    error
	historyWriteErr   error
}

var durations = []int{15, 30, 60, 120}

var (
	panelStyle     = lipgloss.NewStyle().Padding(1, 2)
	mutedStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	orangeStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("208"))
	cyanStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("14"))
	okStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	badStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	pendingStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("7"))
	cursorStyle    = lipgloss.NewStyle().Reverse(true).Underline(true)
	titleStyle     = lipgloss.NewStyle().Bold(true)
	activeTabStyle = orangeStyle.Copy().Bold(true).Border(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("208")).Padding(0, 2)
	tabStyle       = mutedStyle.Copy().Border(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("8")).Padding(0, 2)
	focusStyle     = cyanStyle.Copy().Reverse(true).Blink(true)
)

func New(historyPath string) Model {
	now := time.Now()
	m := Model{
		historyPath:       historyPath,
		screen:            screenSplash,
		tab:               tabTests,
		focus:             focusDuration,
		durationIndex:     2,
		width:             80,
		height:            24,
		splashStarted:     now,
		countdownStarted:  now,
		durationConfirmed: false,
		languageConfirmed: false,
	}
	m.loadHistory()
	return m
}

func (m Model) Init() tea.Cmd {
	return tick()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.updateTable()
		return m, nil
	case tickMsg:
		now := time.Time(msg)
		m.handleTick(now)
		return m, tick()
	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m *Model) handleTick(now time.Time) {
	switch m.screen {
	case screenSplash:
		if now.Sub(m.splashStarted) >= time.Second {
			m.screen = screenHome
		}
	case screenCountdown:
		if now.Sub(m.countdownStarted) >= 4*time.Second {
			m.screen = screenRunning
			m.session.Start(now)
		}
	case screenRunning:
		m.finishIfExpired(now)
	}
}

func (m Model) View() string {
	switch m.screen {
	case screenSplash:
		return m.center(titleStyle.Render("Sohan's Typing Test"))
	case screenCountdown:
		return m.center(titleStyle.Render(m.countdownLabel(time.Now())))
	case screenRunning:
		return m.center(m.runningView())
	default:
		return m.center(m.homeView())
	}
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.screen {
	case screenHome:
		return m.handleHomeKey(msg)
	case screenCountdown:
		return m.handleCountdownKey(msg)
	case screenRunning:
		return m.handleRunningKey(msg)
	default:
		if msg.Type == tea.KeyCtrlQ {
			return m, tea.Quit
		}
		return m, nil
	}
}

func (m Model) handleHomeKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlQ:
		return m, tea.Quit
	case tea.KeyCtrlR:
		return m, nil
	}

	switch msg.String() {
	case "t":
		m.tab = tabTests
		return m, nil
	case "p":
		m.tab = tabPastResults
		return m, nil
	}

	if m.tab == tabTests {
		return m.handleTestsKey(msg)
	}
	return m, nil
}

func (m Model) handleTestsKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.focus == focusStart {
		switch msg.Type {
		case tea.KeyEnter:
			m.prepareCountdown(time.Now())
			return m, nil
		case tea.KeyEsc, tea.KeyLeft:
			m.languageConfirmed = false
			m.focus = focusLanguage
			return m, nil
		}
		if msg.String() == "a" {
			m.languageConfirmed = false
			m.focus = focusLanguage
		}
		return m, nil
	}

	switch msg.Type {
	case tea.KeyEnter:
		if m.focus == focusDuration {
			m.durationConfirmed = true
			m.focus = focusLanguage
		} else {
			m.languageConfirmed = true
			m.focus = focusStart
		}
		return m, nil
	case tea.KeyLeft, tea.KeyUp:
		m.previousOption()
		return m, nil
	case tea.KeyRight, tea.KeyDown:
		m.nextOption()
		return m, nil
	case tea.KeyEsc:
		m.focus = focusDuration
		m.durationConfirmed = false
		m.languageConfirmed = false
		return m, nil
	}

	switch msg.String() {
	case "a", "w":
		m.previousOption()
	case "d", "s":
		m.nextOption()
	}
	return m, nil
}

func (m Model) handleCountdownKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlQ:
		m.abortToHome()
	case tea.KeyCtrlR:
		m.prepareCountdown(time.Now())
	}
	return m, nil
}

func (m Model) handleRunningKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	now := time.Now()
	switch msg.Type {
	case tea.KeyCtrlQ:
		m.abortToHome()
		return m, nil
	case tea.KeyCtrlR:
		m.prepareCountdown(now)
		return m, nil
	case tea.KeyBackspace, tea.KeyCtrlH:
		m.session.Backspace()
		return m, nil
	case tea.KeyEnter:
		m.typeRune('\n', now)
		return m, nil
	case tea.KeySpace:
		m.typeRune(' ', now)
		return m, nil
	case tea.KeyTab:
		if m.isCodingLanguage() {
			m.typeText("    ", now)
		}
		return m, nil
	case tea.KeyUp, tea.KeyDown, tea.KeyLeft, tea.KeyRight:
		return m, nil
	}

	if msg.Type == tea.KeyRunes {
		for _, r := range msg.Runes {
			m.typeRune(r, now)
		}
	}
	return m, nil
}

func (m *Model) previousOption() {
	if m.focus == focusDuration {
		m.durationIndex = (m.durationIndex + len(durations) - 1) % len(durations)
		m.durationConfirmed = false
		m.languageConfirmed = false
		return
	}
	m.languageIndex = (m.languageIndex + len(prompts.Languages) - 1) % len(prompts.Languages)
	m.languageConfirmed = false
}

func (m *Model) nextOption() {
	if m.focus == focusDuration {
		m.durationIndex = (m.durationIndex + 1) % len(durations)
		m.durationConfirmed = false
		m.languageConfirmed = false
		return
	}
	m.languageIndex = (m.languageIndex + 1) % len(prompts.Languages)
	m.languageConfirmed = false
}

func (m *Model) prepareCountdown(now time.Time) {
	duration := durations[m.durationIndex]
	language := prompts.Languages[m.languageIndex]
	m.session = typing.NewSession(prompts.Generate(language, duration), time.Duration(duration)*time.Second)
	m.ensureUpcomingSegments()
	m.screen = screenCountdown
	m.countdownStarted = now
	m.resultSaved = false
	m.historyWriteErr = nil
	m.lastMessage = ""
}

func (m *Model) abortToHome() {
	m.screen = screenHome
	m.tab = tabTests
	m.resultSaved = false
	m.historyWriteErr = nil
}

func (m *Model) typeRune(r rune, now time.Time) {
	advanced := m.session.TypeRune(r, now)
	if advanced || m.session.NeedsMoreSegments() {
		m.ensureUpcomingSegments()
	}
	m.finishIfExpired(now)
}

func (m *Model) typeText(text string, now time.Time) {
	for _, r := range text {
		m.typeRune(r, now)
	}
}

func (m *Model) ensureUpcomingSegments() {
	if !m.session.NeedsMoreSegments() {
		return
	}
	m.session.AppendSegments(prompts.MoreExcluding(prompts.Languages[m.languageIndex], 8, m.session.SegmentStrings()))
}

func (m *Model) finishIfExpired(now time.Time) {
	if m.screen != screenRunning || !m.session.Started || m.session.Done || m.session.Remaining(now) > 0 {
		return
	}
	m.session.Complete(now)
	m.saveResult(now)
	m.screen = screenHome
	m.tab = tabPastResults
	m.loadHistory()
}

func (m *Model) saveResult(now time.Time) {
	if m.resultSaved {
		return
	}
	result := m.session.Metrics(now)
	run := history.Run{
		Timestamp:       now,
		Language:        string(prompts.Languages[m.languageIndex]),
		DurationSeconds: durations[m.durationIndex],
		RawWPM:          result.RawWPM,
		Accuracy:        result.Accuracy,
		NetWPM:          result.NetWPM,
		CorrectChars:    m.session.CorrectChars(),
		TypedChars:      m.session.TypedChars(),
	}
	if err := history.Append(m.historyPath, run); err != nil {
		m.historyWriteErr = err
		return
	}
	m.resultSaved = true
	m.lastMessage = "saved to " + m.historyPath
}

func (m *Model) loadHistory() {
	runs, err := history.Load(m.historyPath)
	m.historyLoadErr = err
	if err == nil {
		m.history = runs
	}
	m.updateTable()
}

func (m *Model) updateTable() {
	columns := []table.Column{
		{Title: "Date", Width: 16},
		{Title: "Format", Width: 10},
		{Title: "Time", Width: 6},
		{Title: "WPM", Width: 7},
		{Title: "Raw", Width: 7},
		{Title: "Acc", Width: 6},
	}

	rows := make([]table.Row, 0, min(len(m.history), 10))
	for i := len(m.history) - 1; i >= 0 && len(rows) < 10; i-- {
		run := m.history[i]
		rows = append(rows, table.Row{
			run.Timestamp.Format("2006-01-02 15:04"),
			run.Language,
			fmt.Sprintf("%ds", run.DurationSeconds),
			fmt.Sprintf("%.1f", run.NetWPM),
			fmt.Sprintf("%.1f", run.RawWPM),
			fmt.Sprintf("%.0f%%", run.Accuracy*100),
		})
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(false),
		table.WithHeight(min(10, max(4, m.height-14))),
	)
	styles := table.DefaultStyles()
	styles.Header = styles.Header.Bold(true).Foreground(lipgloss.Color("14")).BorderStyle(lipgloss.NormalBorder()).BorderBottom(true)
	styles.Cell = styles.Cell.Foreground(lipgloss.Color("7"))
	t.SetStyles(styles)
	m.table = t
}

func (m Model) homeView() string {
	content := []string{
		m.centerLine(titleStyle.Render("Sohan's Typing Test")),
		m.renderTabs(),
		"",
	}
	if m.tab == tabTests {
		content = append(content, m.testsView())
	} else {
		content = append(content, m.pastResultsView())
	}
	content = append(content, "", m.homeHelp())
	return panelStyle.Width(m.contentWidth()).Align(lipgloss.Center).Render(strings.Join(content, "\n"))
}

func (m Model) testsView() string {
	lines := []string{
		m.centerLine(mutedStyle.Render("choose a time limit, then a language format")),
		"",
		m.centerLine("Time  " + m.renderOptions(durationLabels(), m.durationIndex, m.focus == focusDuration, m.durationConfirmed)),
		m.centerLine("Lang  " + m.renderOptions(languageLabels(), m.languageIndex, m.focus == focusLanguage, m.languageConfirmed)),
	}
	if m.focus == focusStart {
		lines = append(lines, "", m.centerLine(focusStyle.Copy().Bold(true).Render("Start  press enter")))
	} else if m.durationConfirmed && !m.languageConfirmed {
		lines = append(lines, "", m.centerLine(mutedStyle.Render("press enter to confirm language")))
	} else {
		lines = append(lines, "", m.centerLine(mutedStyle.Render("press enter to confirm time")))
	}
	if m.lastMessage != "" {
		lines = append(lines, "", m.centerLine(okStyle.Render(m.lastMessage)))
	}
	if m.historyWriteErr != nil {
		lines = append(lines, "", m.centerLine(badStyle.Render("could not save result: "+m.historyWriteErr.Error())))
	}
	return strings.Join(lines, "\n")
}

func (m Model) pastResultsView() string {
	lines := []string{
		m.centerLine(mutedStyle.Render("previous completed attempts")),
		"",
	}
	if m.historyLoadErr != nil {
		lines = append(lines, m.centerLine(badStyle.Render("could not load history: "+m.historyLoadErr.Error())))
	} else if m.historyWriteErr != nil {
		lines = append(lines, m.centerLine(badStyle.Render("could not save result: "+m.historyWriteErr.Error())))
	} else if len(m.history) == 0 {
		lines = append(lines, m.centerLine(mutedStyle.Render("no completed runs yet")))
		lines = append(lines, "", m.centerLine(mutedStyle.Render("WPM trend")), m.centerLine(asciiSparkline(nil, 40)))
	} else {
		lines = append(lines, m.centerBlock(m.table.View()), "", m.centerLine(mutedStyle.Render("WPM trend")), m.centerLine(asciiSparkline(m.history, min(56, m.contentWidth()-4))))
	}
	return strings.Join(lines, "\n")
}

func (m Model) runningView() string {
	now := time.Now()
	result := m.session.Metrics(now)
	remaining := m.session.Remaining(now).Round(time.Second)
	status := fmt.Sprintf(
		"%s  %ds  remaining %s  raw %.1f  acc %.0f%%  wpm %.1f",
		prompts.Languages[m.languageIndex],
		durations[m.durationIndex],
		remaining,
		result.RawWPM,
		result.Accuracy*100,
		result.NetWPM,
	)

	lines := []string{
		m.centerLine(mutedStyle.Render(status)),
		"",
		m.centerBlock(m.renderSegment(max(32, m.contentWidth()-4), max(5, m.height-10))),
		"",
		m.centerLine(mutedStyle.Render("ctrl+q home  ctrl+r restart")),
	}
	align := lipgloss.Center
	if m.isCodingLanguage() {
		align = lipgloss.Left
	}
	return panelStyle.Width(m.contentWidth()).Align(align).Render(strings.Join(lines, "\n"))
}

func (m Model) renderTabs() string {
	tests := tabStyle.Render("Tests (t)")
	results := tabStyle.Render("Past Results (p)")
	if m.tab == tabTests {
		tests = activeTabStyle.Render("Tests (t)")
	} else {
		results = activeTabStyle.Render("Past Results (p)")
	}
	return m.centerBlock(lipgloss.JoinHorizontal(lipgloss.Top, tests, "  ", results))
}

func (m Model) renderOptions(labels []string, selected int, focused bool, confirmed bool) string {
	rendered := make([]string, 0, len(labels))
	for i, label := range labels {
		text := " " + label + " "
		switch {
		case i == selected && focused:
			rendered = append(rendered, focusStyle.Render(text))
		case i == selected && confirmed:
			rendered = append(rendered, okStyle.Render("["+label+"]"))
		case i == selected:
			rendered = append(rendered, cyanStyle.Render("["+label+"]"))
		default:
			rendered = append(rendered, mutedStyle.Render(text))
		}
	}
	return strings.Join(rendered, " ")
}

func (m Model) renderSegment(width int, maxLines int) string {
	target := m.session.CurrentSegment()
	typed := m.session.CurrentTyped()
	wordStates := typing.WordStates(target, typed)

	var builder strings.Builder
	lineWidth := 0
	lines := 1
	for i, targetRune := range target {
		if lines > maxLines {
			break
		}
		if targetRune == '\n' {
			if i == len(typed) {
				builder.WriteString(cursorStyle.Render(" "))
			}
			builder.WriteRune('\n')
			lineWidth = 0
			lines++
			continue
		}
		if lineWidth >= width && targetRune == ' ' {
			builder.WriteRune('\n')
			lineWidth = 0
			lines++
			continue
		}

		style := pendingStyle
		if i < len(typed) {
			if typed[i] == targetRune {
				style = okStyle
			} else {
				style = badStyle
			}
		}

		word := wordStateAt(wordStates, i)
		if word != nil {
			if word.CompleteCorrect {
				style = style.Italic(true)
			} else if word.Current {
				style = style.Bold(true)
			}
		}
		if i == len(typed) {
			style = style.Inherit(cursorStyle)
		}

		text, displayWidth := visibleRune(targetRune)
		builder.WriteString(style.Render(text))
		lineWidth += displayWidth
	}
	if len(typed) >= len(target) {
		builder.WriteString(cursorStyle.Render(" "))
	}
	return builder.String()
}

func wordStateAt(states []typing.WordState, index int) *typing.WordState {
	for i := range states {
		if index >= states[i].Start && index < states[i].End {
			return &states[i]
		}
	}
	return nil
}

func (m Model) homeHelp() string {
	if m.tab == tabTests {
		return m.centerLine(mutedStyle.Render("t/p tabs  wasd/arrows select  enter confirm  ctrl+q quit"))
	}
	return m.centerLine(mutedStyle.Render("t tests  p past results  ctrl+q quit"))
}

func (m Model) countdownLabel(now time.Time) string {
	elapsed := now.Sub(m.countdownStarted)
	switch {
	case elapsed < time.Second:
		return "3"
	case elapsed < 2*time.Second:
		return "2"
	case elapsed < 3*time.Second:
		return "1"
	default:
		return "GO"
	}
}

func (m Model) center(content string) string {
	return lipgloss.Place(
		max(1, m.width),
		max(1, m.height),
		lipgloss.Center,
		lipgloss.Center,
		content,
	)
}

func (m Model) centerLine(content string) string {
	return content
}

func (m Model) centerBlock(content string) string {
	return content
}

func (m Model) contentWidth() int {
	return min(88, max(1, m.width-6))
}

func visibleRune(r rune) (string, int) {
	if r == '\t' {
		return "    ", 4
	}
	return string(r), 1
}

func (m Model) isCodingLanguage() bool {
	return prompts.Languages[m.languageIndex] != prompts.English
}

func asciiSparkline(runs []history.Run, width int) string {
	if width < 1 {
		width = 1
	}
	if len(runs) == 0 {
		return mutedStyle.Render(strings.Repeat("-", width))
	}

	start := max(0, len(runs)-width)
	values := make([]float64, 0, len(runs)-start)
	for _, run := range runs[start:] {
		values = append(values, run.NetWPM)
	}

	minValue, maxValue := values[0], values[0]
	for _, value := range values {
		minValue = math.Min(minValue, value)
		maxValue = math.Max(maxValue, value)
	}

	levels := []rune("._-=+*#")
	var builder strings.Builder
	for _, value := range values {
		index := 0
		if maxValue > minValue {
			index = int(math.Round((value - minValue) / (maxValue - minValue) * float64(len(levels)-1)))
		}
		builder.WriteRune(levels[index])
	}
	return cyanStyle.Render(builder.String())
}

func durationLabels() []string {
	labels := make([]string, len(durations))
	for i, duration := range durations {
		labels[i] = fmt.Sprintf("%ds", duration)
	}
	return labels
}

func languageLabels() []string {
	labels := make([]string, len(prompts.Languages))
	for i, language := range prompts.Languages {
		labels[i] = string(language)
	}
	return labels
}

func tick() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
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
