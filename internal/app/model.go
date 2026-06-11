package app

import (
	"fmt"
	"math"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"typing-test-tui/internal/history"
	"typing-test-tui/internal/prompts"
	"typing-test-tui/internal/typing"
)

const (
	tabTest = iota
	tabHistory
)

type tickMsg time.Time

type Model struct {
	historyPath string
	history     []history.Run
	table       table.Model
	session     typing.Session

	tab             int
	languageIndex   int
	durationIndex   int
	width           int
	height          int
	resultSaved     bool
	lastMessage     string
	historyLoadErr  error
	historyWriteErr error
}

var durations = []int{15, 30, 60, 120}

var (
	baseStyle      = lipgloss.NewStyle().Padding(1, 2)
	mutedStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	activeTabStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15")).Background(lipgloss.Color("4")).Padding(0, 1)
	tabStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Padding(0, 1)
	okStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	badStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	pendingStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("7"))
	cursorStyle    = lipgloss.NewStyle().Reverse(true).Underline(true)
	titleStyle     = lipgloss.NewStyle().Bold(true)
)

func New(historyPath string) Model {
	m := Model{
		historyPath:   historyPath,
		durationIndex: 2,
		width:         80,
		height:        24,
	}
	m.resetSession()
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
		m.finishIfExpired(now)
		return m, tick()
	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m Model) View() string {
	var body string
	if m.tab == tabHistory {
		body = m.historyView()
	} else {
		body = m.testView()
	}

	return baseStyle.Width(max(20, m.width-4)).Render(strings.Join([]string{
		m.renderTabs(),
		body,
		m.helpView(),
	}, "\n"))
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	now := time.Now()
	switch msg.Type {
	case tea.KeyCtrlC:
		return m, tea.Quit
	case tea.KeyTab, tea.KeyRight:
		if !m.session.Started || m.session.Done {
			m.tab = (m.tab + 1) % 2
		}
		return m, nil
	case tea.KeyLeft:
		if !m.session.Started || m.session.Done {
			m.tab = (m.tab + 1) % 2
		}
		return m, nil
	case tea.KeyBackspace, tea.KeyCtrlH:
		if m.tab == tabTest {
			m.session.Backspace()
		}
		return m, nil
	case tea.KeySpace:
		if m.tab == tabTest {
			m.session.TypeRune(' ', now)
			m.finishIfExpired(now)
		}
		return m, nil
	}

	switch msg.String() {
	case "q":
		return m, tea.Quit
	case "r":
		m.resetSession()
		return m, nil
	case "h":
		if !m.session.Started || m.session.Done {
			m.tab = tabHistory
		}
		return m, nil
	case "t":
		if !m.session.Started || m.session.Done {
			m.tab = tabTest
		}
		return m, nil
	case "[":
		if !m.session.Started {
			m.durationIndex = (m.durationIndex + len(durations) - 1) % len(durations)
			m.resetSession()
		}
		return m, nil
	case "]":
		if !m.session.Started {
			m.durationIndex = (m.durationIndex + 1) % len(durations)
			m.resetSession()
		}
		return m, nil
	case ",":
		if !m.session.Started {
			m.languageIndex = (m.languageIndex + len(prompts.Languages) - 1) % len(prompts.Languages)
			m.resetSession()
		}
		return m, nil
	case ".":
		if !m.session.Started {
			m.languageIndex = (m.languageIndex + 1) % len(prompts.Languages)
			m.resetSession()
		}
		return m, nil
	}

	if m.tab == tabTest && msg.Type == tea.KeyRunes {
		for _, r := range msg.Runes {
			m.session.TypeRune(r, now)
		}
		m.finishIfExpired(now)
	}

	return m, nil
}

func (m *Model) resetSession() {
	minRunes := durations[m.durationIndex] * 12
	target := prompts.Random(prompts.Languages[m.languageIndex], minRunes)
	m.session = typing.NewSession(target, time.Duration(durations[m.durationIndex])*time.Second)
	m.resultSaved = false
	m.historyWriteErr = nil
	m.lastMessage = ""
}

func (m *Model) finishIfExpired(now time.Time) {
	if !m.session.Started || m.session.Done || m.session.Remaining(now) > 0 {
		return
	}
	m.session.Complete(now)
	m.saveResult(now)
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
	m.loadHistory()
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
		{Title: "When", Width: 16},
		{Title: "Lang", Width: 10},
		{Title: "Sec", Width: 4},
		{Title: "Raw", Width: 6},
		{Title: "Acc", Width: 6},
		{Title: "Net", Width: 6},
	}

	rows := make([]table.Row, 0, min(len(m.history), 12))
	for i := len(m.history) - 1; i >= 0 && len(rows) < 12; i-- {
		run := m.history[i]
		rows = append(rows, table.Row{
			run.Timestamp.Format("01-02 15:04"),
			run.Language,
			fmt.Sprintf("%d", run.DurationSeconds),
			fmt.Sprintf("%.1f", run.RawWPM),
			fmt.Sprintf("%.0f%%", run.Accuracy*100),
			fmt.Sprintf("%.1f", run.NetWPM),
		})
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(false),
		table.WithHeight(min(12, max(4, m.height-10))),
	)
	styles := table.DefaultStyles()
	styles.Header = styles.Header.Bold(true).BorderStyle(lipgloss.NormalBorder()).BorderBottom(true)
	styles.Selected = styles.Selected.Foreground(lipgloss.Color("15")).Background(lipgloss.Color("4"))
	t.SetStyles(styles)
	m.table = t
}

func (m Model) testView() string {
	now := time.Now()
	result := m.session.Metrics(now)
	remaining := m.session.Remaining(now).Round(time.Second)
	if !m.session.Started {
		remaining = m.session.Duration
	}

	status := fmt.Sprintf(
		"%s  %ds  remaining %s  raw %.1f  acc %.0f%%  net %.1f",
		prompts.Languages[m.languageIndex],
		durations[m.durationIndex],
		remaining,
		result.RawWPM,
		result.Accuracy*100,
		result.NetWPM,
	)

	lines := []string{
		titleStyle.Render("Typing Test"),
		mutedStyle.Render(status),
		"",
		m.renderPrompt(max(30, m.width-8), max(5, m.height-11)),
	}

	if m.historyWriteErr != nil {
		lines = append(lines, "", badStyle.Render("could not save result: "+m.historyWriteErr.Error()))
	} else if m.session.Done {
		lines = append(lines, "", okStyle.Render("complete - "+m.lastMessage))
	} else if !m.session.Started {
		lines = append(lines, "", mutedStyle.Render("start typing when ready"))
	}

	return strings.Join(lines, "\n")
}

func (m Model) historyView() string {
	lines := []string{
		titleStyle.Render("History"),
		mutedStyle.Render("net speed trend"),
		sparkline(m.history, max(20, min(60, m.width-8))),
		"",
	}

	if m.historyLoadErr != nil {
		lines = append(lines, badStyle.Render("could not load history: "+m.historyLoadErr.Error()))
	} else if len(m.history) == 0 {
		lines = append(lines, mutedStyle.Render("no completed runs yet"))
	} else {
		lines = append(lines, m.table.View())
	}
	return strings.Join(lines, "\n")
}

func (m Model) renderTabs() string {
	tabs := []string{"Test", "History"}
	rendered := make([]string, len(tabs))
	for i, label := range tabs {
		if i == m.tab {
			rendered[i] = activeTabStyle.Render(label)
		} else {
			rendered[i] = tabStyle.Render(label)
		}
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, rendered...)
}

func (m Model) renderPrompt(width int, maxLines int) string {
	var builder strings.Builder
	lineWidth := 0
	lines := 1

	for i, target := range m.session.Target {
		if lines > maxLines {
			break
		}

		text := string(target)
		if target == '\n' {
			builder.WriteRune('\n')
			lineWidth = 0
			lines++
			continue
		}

		if lineWidth >= width && target == ' ' {
			builder.WriteRune('\n')
			lineWidth = 0
			lines++
			continue
		}

		style := pendingStyle
		if i < len(m.session.Typed) {
			if m.session.Typed[i] == target {
				style = okStyle
			} else {
				style = badStyle
			}
		}
		if i == len(m.session.Typed) && !m.session.Done {
			style = style.Inherit(cursorStyle)
			if target == ' ' {
				text = " "
			}
		}
		builder.WriteString(style.Render(text))
		lineWidth += max(1, utf8.RuneLen(target))
	}

	if len(m.session.Typed) >= len(m.session.Target) && !m.session.Done {
		builder.WriteString(cursorStyle.Render(" "))
	}

	return builder.String()
}

func (m Model) helpView() string {
	if m.session.Started && !m.session.Done {
		return mutedStyle.Render("backspace edit  r restart  q quit")
	}
	return mutedStyle.Render("tab switch  ,/. language  [/] duration  r restart  q quit")
}

func sparkline(runs []history.Run, width int) string {
	if len(runs) == 0 {
		return mutedStyle.Render(strings.Repeat("─", max(1, width)))
	}

	values := make([]float64, 0, min(len(runs), width))
	start := max(0, len(runs)-width)
	for _, run := range runs[start:] {
		values = append(values, run.NetWPM)
	}

	minValue, maxValue := values[0], values[0]
	for _, value := range values {
		minValue = math.Min(minValue, value)
		maxValue = math.Max(maxValue, value)
	}

	blocks := []rune("▁▂▃▄▅▆▇█")
	var builder strings.Builder
	for _, value := range values {
		index := 0
		if maxValue > minValue {
			index = int(math.Round((value - minValue) / (maxValue - minValue) * float64(len(blocks)-1)))
		}
		builder.WriteRune(blocks[index])
	}
	return okStyle.Render(builder.String())
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
