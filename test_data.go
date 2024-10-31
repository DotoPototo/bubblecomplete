package bubblecomplete

var TestCommands = []*Command{
	{
		Command:     "cat",
		Description: "Concatenate and display the content of files",
		PositionalArguments: []*PositionalArgument{
			{
				Name:        "File",
				Description: "File to display",
				Type:        FileArgument,
				Required:    true,
			},
		},
		Flags: []*Flag{
			{
				LongFlag:    "--show-ends",
				Description: "Display $ at end of each line",
				Type:        BoolArgument,
			},
			{
				ShortFlag:   "-f",
				LongFlag:    "--file-name",
				Description: "Specify the name to display for a file",
				Type:        StringArgument,
			},
			{
				ShortFlag:   "-n",
				LongFlag:    "--number",
				Description: "Number all output lines",
				Type:        BoolArgument,
			},
			{
				ShortFlag:   "-p",
				LongFlag:    "--plain",
				Description: "Only show plain style, no decorations",
				Type:        BoolArgument,
			},
		},
	},
	{
		Command:     "cp",
		Description: "Copy files and directories",
		PositionalArguments: []*PositionalArgument{
			{
				Name:        "file",
				Description: "File to copy",
				Type:        FileDirArgument,
				Required:    true,
			},
			{
				Name:        "destination",
				Description: "Destination to copy the file to",
				Type:        DirArgument,
				Required:    true,
			},
		},
		Flags: []*Flag{
			{
				ShortFlag:   "-r",
				Description: "Copy directories recursively",
				Type:        BoolArgument,
			},
			{
				ShortFlag:   "-f",
				Description: "Copy directories recursively",
				Type:        BoolArgument,
			},
			{
				ShortFlag:   "-t",
				Description: "Copy directories recursively",
				Type:        BoolArgument,
			},
		},
	},
	{
		Command:     "git",
		Description: "Git is a distributed version control system",
		SubCommands: []*Command{
			{
				Command:     "status",
				Description: "Show the working tree status",
			},
			{
				Command:     "stash",
				Description: "Stash the changes in a dirty working directory away",
				SubCommands: []*Command{
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
				Flags: []*Flag{
					{
						ShortFlag:   "-m",
						LongFlag:    "--message",
						Description: "Use the given message as the commit message",
						Type:        StringArgument,
					},
					{
						ShortFlag:   "-a",
						LongFlag:    "--all",
						Description: "Tell the command to automatically stage files that have been modified and deleted, but new files you have not told Git about are not affected",
						Type:        BoolArgument,
					},
					{
						LongFlag:    "--amend",
						Description: "Replace the tip of the current branch by creating a new commit",
						Type:        BoolArgument,
					},
				},
			},
			{
				Command:     "push",
				Description: "Update remote refs along with associated objects",
				PositionalArguments: []*PositionalArgument{
					{
						Name:        "remote",
						Description: "Remote repository to push to",
						Type:        StringArgument,
						Required:    false,
					},
					{
						Name:        "branch",
						Description: "Branch to push",
						Type:        StringArgument,
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
		Flags: []*Flag{
			{
				LongFlag:    "--version",
				Description: "Print the Git version",
				Type:        BoolArgument,
			},
			{
				LongFlag:    "--help",
				Description: "Show the help message",
				Type:        BoolArgument,
				Persistent:  true,
			},
		},
	},
	{
		Command:     "ps",
		Description: "Example PowerShell-style command",
		PositionalArguments: []*PositionalArgument{
			{
				Name:        "Input",
				Description: "Input file to process",
				Type:        FileArgument,
				Required:    true,
			},
		},
		Flags: []*Flag{
			{
				PsFlag:      "-stringarg",
				Description: "Example flag showing PowerShell style",
				Type:        StringArgument,
			},
			{
				PsFlag:      "-boolarg",
				Description: "Example boolean argument",
				Type:        BoolArgument,
			},
			{
				PsFlag:      "-floatarg",
				Description: "Example float argument",
				Type:        FloatArgument,
			},
			{
				PsFlag:      "-intarg",
				Description: "Example int argument",
				Type:        IntArgument,
			},
			{
				PsFlag:      "-filearg",
				Description: "Example file argument",
				Type:        FileArgument,
			},
			{
				PsFlag:      "-dirarg",
				Description: "Example directory argument",
				Type:        DirArgument,
			},
			{
				PsFlag:      "-FileDirArg",
				Description: "Example file or directory argument",
				Type:        FileDirArgument,
			},
		},
	},
}
