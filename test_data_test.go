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
				Description: "Force overwrite of existing files",
				Type:        BoolArgument,
			},
			{
				ShortFlag:   "-t",
				Description: "Preserve modification times",
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
				Flags: []*Flag{
					{
						ShortFlag:   "-s",
						LongFlag:    "--short",
						Description: "Give output in short format",
						Type:        BoolArgument,
					},
					{
						ShortFlag:   "-b",
						LongFlag:    "--branch",
						Description: "Show branch and tracking info",
						Type:        BoolArgument,
					},
				},
			},
			{
				Command:     "stash",
				Description: "Stash the changes in a dirty working directory away",
				SubCommands: []*Command{
					{
						Command:     "pop",
						Description: "Remove a single stashed state from the stash list and apply it on top of the current working tree state",
						Flags: []*Flag{
							{
								LongFlag:    "--index",
								Description: "Try to reinstate index changes as well",
								Type:        BoolArgument,
							},
						},
					},
					{
						Command:     "apply",
						Description: "Like pop, but do not remove the state from the stash list",
						Flags: []*Flag{
							{
								LongFlag:    "--index",
								Description: "Try to reinstate index changes as well",
								Type:        BoolArgument,
							},
						},
					},
					{
						Command:     "drop",
						Description: "Remove a single stashed state from the stash list",
					},
					{
						Command:     "list",
						Description: "List the stash entries",
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
				Flags: []*Flag{
					{
						ShortFlag:   "-f",
						LongFlag:    "--force",
						Description: "Force push even if remote has diverged",
						Type:        BoolArgument,
					},
					{
						ShortFlag:   "-u",
						LongFlag:    "--set-upstream",
						Description: "Set upstream tracking reference",
						Type:        BoolArgument,
					},
					{
						LongFlag:    "--tags",
						Description: "Push all tags",
						Type:        BoolArgument,
					},
				},
			},
			{
				Command:     "pull",
				Description: "Fetch from and integrate with another repository or a local branch",
				PositionalArguments: []*PositionalArgument{
					{
						Name:        "remote",
						Description: "Remote to pull from",
						Type:        StringArgument,
						Required:    false,
					},
					{
						Name:        "branch",
						Description: "Branch to pull",
						Type:        StringArgument,
						Required:    false,
					},
				},
				Flags: []*Flag{
					{
						LongFlag:    "--rebase",
						Description: "Rebase current branch on top of upstream",
						Type:        BoolArgument,
					},
					{
						LongFlag:    "--no-rebase",
						Description: "Merge instead of rebasing",
						Type:        BoolArgument,
					},
				},
			},
			{
				Command:     "clone",
				Description: "Clone a repository into a new directory",
				PositionalArguments: []*PositionalArgument{
					{
						Name:        "repository",
						Description: "Repository URL to clone",
						Type:        StringArgument,
						Required:    true,
					},
					{
						Name:        "directory",
						Description: "Directory to clone into",
						Type:        StringArgument,
						Required:    false,
					},
				},
				Flags: []*Flag{
					{
						LongFlag:    "--depth",
						Description: "Create a shallow clone with history truncated",
						Type:        IntArgument,
					},
					{
						LongFlag:    "--bare",
						Description: "Make a bare Git repository",
						Type:        BoolArgument,
					},
					{
						ShortFlag:   "-b",
						LongFlag:    "--branch",
						Description: "Point HEAD at a specific branch",
						Type:        StringArgument,
					},
				},
			},
			{
				Command:     "checkout",
				Description: "Switch branches or restore working tree files",
				PositionalArguments: []*PositionalArgument{
					{
						Name:        "branch",
						Description: "Branch or commit to check out",
						Type:        StringArgument,
						Required:    false,
					},
				},
				Flags: []*Flag{
					{
						ShortFlag:   "-b",
						Description: "Create and checkout a new branch",
						Type:        StringArgument,
					},
					{
						LongFlag:    "--track",
						Description: "Set up tracking for the new branch",
						Type:        BoolArgument,
					},
					{
						LongFlag:    "--detach",
						Description: "Detach HEAD at the named commit",
						Type:        BoolArgument,
					},
				},
			},
			{
				Command:     "branch",
				Description: "List, create, or delete branches",
				PositionalArguments: []*PositionalArgument{
					{
						Name:        "name",
						Description: "Branch name to create or filter",
						Type:        StringArgument,
						Required:    false,
					},
				},
				Flags: []*Flag{
					{
						ShortFlag:   "-d",
						LongFlag:    "--delete",
						Description: "Delete a branch",
						Type:        BoolArgument,
					},
					{
						ShortFlag:   "-D",
						Description: "Force delete a branch",
						Type:        BoolArgument,
					},
					{
						ShortFlag:   "-a",
						LongFlag:    "--all",
						Description: "List both local and remote branches",
						Type:        BoolArgument,
					},
					{
						ShortFlag:   "-r",
						LongFlag:    "--remotes",
						Description: "List remote-tracking branches",
						Type:        BoolArgument,
					},
				},
			},
			{
				Command:     "merge",
				Description: "Join two or more development histories together",
				PositionalArguments: []*PositionalArgument{
					{
						Name:        "branch",
						Description: "Branch to merge into current branch",
						Type:        StringArgument,
						Required:    false,
					},
				},
				Flags: []*Flag{
					{
						LongFlag:    "--no-ff",
						Description: "Create a merge commit even for fast-forward",
						Type:        BoolArgument,
					},
					{
						LongFlag:    "--squash",
						Description: "Squash commits into a single commit",
						Type:        BoolArgument,
					},
					{
						LongFlag:    "--abort",
						Description: "Abort the current merge",
						Type:        BoolArgument,
					},
				},
			},
			{
				Command:     "rebase",
				Description: "Reapply commits on top of another base tip",
				PositionalArguments: []*PositionalArgument{
					{
						Name:        "upstream",
						Description: "Upstream branch to rebase onto",
						Type:        StringArgument,
						Required:    false,
					},
				},
				Flags: []*Flag{
					{
						ShortFlag:   "-i",
						LongFlag:    "--interactive",
						Description: "Make a list of commits to be rebased and let the user edit",
						Type:        BoolArgument,
					},
					{
						LongFlag:    "--onto",
						Description: "Starting point to create new commits onto",
						Type:        StringArgument,
					},
					{
						LongFlag:    "--abort",
						Description: "Abort the rebase operation",
						Type:        BoolArgument,
					},
					{
						LongFlag:    "--continue",
						Description: "Continue the rebase after resolving conflicts",
						Type:        BoolArgument,
					},
				},
			},
			{
				Command:     "tag",
				Description: "Create, list, delete or verify a tag object signed with GPG",
				PositionalArguments: []*PositionalArgument{
					{
						Name:        "tagname",
						Description: "Tag name to create",
						Type:        StringArgument,
						Required:    false,
					},
				},
				Flags: []*Flag{
					{
						ShortFlag:   "-a",
						LongFlag:    "--annotate",
						Description: "Make an annotated tag",
						Type:        BoolArgument,
					},
					{
						ShortFlag:   "-d",
						LongFlag:    "--delete",
						Description: "Delete existing tags",
						Type:        BoolArgument,
					},
					{
						ShortFlag:   "-m",
						LongFlag:    "--message",
						Description: "Tag message",
						Type:        StringArgument,
					},
					{
						ShortFlag:   "-l",
						LongFlag:    "--list",
						Description: "List tags matching a pattern",
						Type:        BoolArgument,
					},
				},
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
