package main

import (
	"fmt"
	"log"
	"os"
	"runtime/pprof"

	bubblecomplete "github.com/dotopototo/bubblecomplete"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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
	bc, err := bubblecomplete.New(bubblecomplete.TestCommands, 100)

	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	bc.CompletionsPosition = bubblecomplete.PositionBelow
	bc.ShowBorderScroll = true
	home, _ := os.UserHomeDir()
	historyFilePath := home + "/.bubblecomplete_history.json"
	bc.SetHistoryFilePath(historyFilePath)
	bc.SetPlaceholder("Enter Command...")

	bc.HistoryLimit = 50

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
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			return m, tea.Quit
		case tea.KeyEsc:
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

func (m model) View() string {
	if m.bubblecomplete.Err != nil {
		return m.bubblecomplete.Err.Error()
	}

	text := "Enter Command:"
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
	}
	return text + "\n" + m.bubblecomplete.View()
}

func main() {
	// Profiling CPU before the main workload is started
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

	// Profiling memory after the main workload is completed
	fMem, er := os.Create("mem.prof")
	if er != nil {
		log.Fatal(er)
	}
	pprof.WriteHeapProfile(fMem)
	fMem.Close()

	// Profiling goroutines after the main workload is completed
	fGoroutine, err := os.Create("goroutine.prof")
	if err != nil {
		log.Fatal(err)
	}
	pprof.Lookup("goroutine").WriteTo(fGoroutine, 0)
	fGoroutine.Close()
}
