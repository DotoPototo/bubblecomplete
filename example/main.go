package main

import (
	"fmt"
	"log"
	"os"
	"runtime/pprof"

	bubblecomplete "github.com/mikecbone/bubblecomplete"

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

var commands = []*bubblecomplete.Command{
	{
		Command:     "cat",
		Description: "Concatenate and display the content of files",
		PositionalArguments: []*bubblecomplete.PositionalArgument{
			{
				Name:        "File",
				Description: "File to display",
				Type:        bubblecomplete.FileArgument,
				Required:    true,
			},
		},
		Flags: []*bubblecomplete.Flag{
			{
				LongFlag:    "--show-ends",
				Description: "Display $ at end of each line",
				Type:        bubblecomplete.BoolArgument,
			},
			{
				ShortFlag:   "-n",
				LongFlag:    "--number",
				Description: "Number all output lines",
				Type:        bubblecomplete.BoolArgument,
			},
		},
	},
	{
		Command:     "cp",
		Description: "Copy files and directories",
		PositionalArguments: []*bubblecomplete.PositionalArgument{
			{
				Name:        "file",
				Description: "File to copy",
				Type:        bubblecomplete.FileDirArgument,
				Required:    true,
			},
			{
				Name:        "destination",
				Description: "Destination to copy the file to",
				Type:        bubblecomplete.DirArgument,
				Required:    true,
			},
		},
		Flags: []*bubblecomplete.Flag{
			{
				ShortFlag:   "-r",
				Description: "Copy directories recursively",
				Type:        bubblecomplete.BoolArgument,
			},
			{
				ShortFlag:   "-f",
				Description: "Copy directories recursively",
				Type:        bubblecomplete.BoolArgument,
			},
			{
				ShortFlag:   "-t",
				Description: "Copy directories recursively",
				Type:        bubblecomplete.BoolArgument,
			},
		},
	},
	{
		Command:     "git",
		Description: "Git is a distributed version control system",
		SubCommands: []*bubblecomplete.Command{
			{
				Command:     "status",
				Description: "Show the working tree status",
			},
			{
				Command:     "stash",
				Description: "Stash the changes in a dirty working directory away",
				SubCommands: []*bubblecomplete.Command{
					{
						Command:     "pop",
						Description: "Remove a single stashed state from the stash list and apply it on top of the current working tree state",
					},
					{
						Command:     "apply",
						Description: "Like pop, but do not remove the state from the stash list",
					},
				},
			},
			{
				Command:     "commit",
				Description: "Record changes to the repository",
				Flags: []*bubblecomplete.Flag{
					{
						ShortFlag:   "-m",
						LongFlag:    "--message",
						Description: "Use the given message as the commit message",
						Type:        bubblecomplete.StringArgument,
					},
					{
						ShortFlag:   "-a",
						LongFlag:    "--all",
						Description: "Tell the command to automatically stage files that have been modified and deleted, but new files you have not told Git about are not affected",
						Type:        bubblecomplete.BoolArgument,
					},
					{
						LongFlag:    "--amend",
						Description: "Replace the tip of the current branch by creating a new commit",
						Type:        bubblecomplete.BoolArgument,
					},
				},
			},
			{
				Command:     "push",
				Description: "Update remote refs along with associated objects",
				PositionalArguments: []*bubblecomplete.PositionalArgument{
					{
						Name:        "remote",
						Description: "Remote repository to push to",
						Type:        bubblecomplete.StringArgument,
						Required:    false,
					},
					{
						Name:        "branch",
						Description: "Branch to push",
						Type:        bubblecomplete.StringArgument,
						Required:    false,
					},
				},
			},
			{
				Command:     "pull",
				Description: "Fetch from and integrate with another repository or a local branch",
			},
			{
				Command:     "clone",
				Description: "Clone a repository into a new directory",
			},
			{
				Command:     "checkout",
				Description: "Switch branches or restore working tree files",
			},
			{
				Command:     "branch",
				Description: "List, create, or delete branches",
			},
			{
				Command:     "merge",
				Description: "Join two or more development histories together",
			},
			{
				Command:     "rebase",
				Description: "Reapply commits on top of another base tip",
			},
			{
				Command:     "tag",
				Description: "Create, list, delete or verify a tag object signed with GPG",
			},
		},
		Flags: []*bubblecomplete.Flag{
			{
				LongFlag:    "--version",
				Description: "Print the Git version",
				Type:        bubblecomplete.BoolArgument,
			},
			{
				LongFlag:    "--help",
				Description: "Show the help message",
				Type:        bubblecomplete.BoolArgument,
				Persistent:  true,
			},
		},
	},
	{
		Command:     "go",
		Description: "Go is a statically typed, compiled programming language designed at Google",
		SubCommands: []*bubblecomplete.Command{
			{
				Command:     "run",
				Description: "Compile and run Go program",
			},
			{
				Command:     "build",
				Description: "Compile packages and dependencies",
				SubCommands: []*bubblecomplete.Command{
					{
						Command: "test",
					},
					{
						Command: "all",
					},
				},
			},
			{
				Command:     "test",
				Description: "Test packages",
			},
		},
	},
}

func initialModel() tea.Model {
	bc, err := bubblecomplete.New(commands, 100)

	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	bc.CompletionsPosition = bubblecomplete.PositionBelow
	bc.ShowBorderScroll = true
	home, _ := os.UserHomeDir()
	historyFilePath := home + "/.bubblecomplete_history.json"
	bc.SetHistoryFilePath(historyFilePath)
	bc.SetPlaceholder("Enter Falcon RTR Command...")

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
