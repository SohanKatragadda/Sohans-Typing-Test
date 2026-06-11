package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"typing-test-tui/internal/app"
)

func main() {
	program := tea.NewProgram(app.New("typing_history.csv"), tea.WithAltScreen())
	if _, err := program.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "typing-test-tui: %v\n", err)
		os.Exit(1)
	}
}
