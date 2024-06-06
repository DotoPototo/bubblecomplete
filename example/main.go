package main

import (
	"fmt"
	"os"

	bubblecomplete "github.com/mikecbone/bubblecomplete"

	tea "github.com/charmbracelet/bubbletea"
)

type model struct {
	autoComplete bubblecomplete.Model
}

var commands = []*bubblecomplete.Command{
	{
		Command: "git",
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
				Arguments: []*bubblecomplete.Argument{
					{
						Argument:    "-m",
						Description: "Use the given message as the commit message",
						Type:        bubblecomplete.StringArgument,
					},
					{
						Argument:    "--index",
						Description: "A test argument for int type",
						Type:        bubblecomplete.IntArgument,
					},
					{
						Argument:    "--floaty",
						Description: "A test argument for float type",
						Type:        bubblecomplete.FloatArgument,
					},
					{
						Argument:    "-a",
						Description: "Tell the command to automatically stage files that have been modified and deleted, but new files you have not told Git about are not affected",
						Type:        bubblecomplete.BoolArgument,
					},
					{
						Argument:    "--amend",
						Description: "Replace the tip of the current branch by creating a new commit",
						Type:        bubblecomplete.BoolArgument,
					},
					{
						Argument:    "--file",
						Description: "A test argument for file type",
						Type:        bubblecomplete.FileArgument,
					},
				},
			},
			{
				Command:     "push",
				Description: "Update remote refs along with associated objects",
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
	},
	{
		Command: "go",
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

	ac := bubblecomplete.New(commands)

	// TODO: Make some changes to the model as example

	m := model{
		autoComplete: ac,
	}

	return m
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.autoComplete, cmd = m.autoComplete.Update(msg)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			return m, tea.Quit
		}
	case bubblecomplete.SelectedCommandMsg:
		// TODO: Do something with the selected command or error
	}
	return m, cmd
}

func (m model) View() string {
	return m.autoComplete.View()
}

func main() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}
}
