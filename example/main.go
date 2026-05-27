package main

import (
	"fmt"
	"log"
	"os"
	"runtime/pprof"

	bubblecomplete "github.com/dotopototo/bubblecomplete"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

var (
	validCommandStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("#00c300")).Bold(true)
	unknownCommandStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("#ff6e67")).Bold(true)
	unknownCommandErrorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#ff6e67")).Faint(true)
)

type model struct {
	bubblecomplete bubblecomplete.Model
	command        string
	err            error
}

func initialModel() tea.Model {
	home, _ := os.UserHomeDir()
	historyFilePath := home + "/.bubblecomplete_history.json"

	bc, err := bubblecomplete.New(
		bubblecomplete.TestCommands,
		100,
		bubblecomplete.WithCompletionsPosition(bubblecomplete.PositionBelow),
		bubblecomplete.WithBorderScroll(true),
		bubblecomplete.WithScrollbar(true),
		bubblecomplete.WithIcons(true),
		bubblecomplete.WithCompletionRows(7),
		bubblecomplete.WithHistoryFilePath(historyFilePath),
		bubblecomplete.WithHistoryLimit(50),
	)
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	m := model{
		bubblecomplete: bc,
	}

	return m
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.bubblecomplete, cmd = m.bubblecomplete.Update(msg)

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "esc":
			if m.bubblecomplete.ShowingCompletions() {
				m.bubblecomplete.CloseCompletions()
			}
		}
	case tea.WindowSizeMsg:
		m.bubblecomplete.SetWidth(msg.Width)
	case bubblecomplete.SelectedCommandMsg:
		m.command = msg.Command
		m.err = msg.Err
	}

	return m, cmd
}

func (m model) View() tea.View {
	if m.bubblecomplete.Err != nil {
		return tea.NewView(m.bubblecomplete.Err.Error())
	}

	var text string
	if m.command != "" {
		text = "Entered Command: "
		if m.err != nil {
			text += unknownCommandStyle.Render(
				m.command,
			) + unknownCommandErrorStyle.Render(
				" ["+m.err.Error()+"]",
			)
		} else {
			text += validCommandStyle.Render(m.command)
		}
		text += "\n"
	}
	return tea.NewView(text + m.bubblecomplete.Render())
}

func main() {
	f, e := os.Create("cpu.prof")
	if e != nil {
		log.Fatal(e)
	}
	pprof.StartCPUProfile(f)
	defer pprof.StopCPUProfile()

	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}

	fMem, er := os.Create("mem.prof")
	if er != nil {
		log.Fatal(er)
	}
	pprof.WriteHeapProfile(fMem)
	fMem.Close()

	fGoroutine, err := os.Create("goroutine.prof")
	if err != nil {
		log.Fatal(err)
	}
	pprof.Lookup("goroutine").WriteTo(fGoroutine, 0)
	fGoroutine.Close()
}
