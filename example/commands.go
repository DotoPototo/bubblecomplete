package main

import bubblecomplete "github.com/dotopototo/bubblecomplete/v2"

var demoCommands = []*bubblecomplete.Command{
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
				ShortFlag:   "-f",
				LongFlag:    "--file-name",
				Description: "Specify the name to display for a file",
				Type:        bubblecomplete.StringArgument,
			},
			{
				ShortFlag:   "-n",
				LongFlag:    "--number",
				Description: "Number all output lines",
				Type:        bubblecomplete.BoolArgument,
			},
			{
				ShortFlag:   "-p",
				LongFlag:    "--plain",
				Description: "Only show plain style, no decorations",
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
				Description: "Force overwrite of existing files",
				Type:        bubblecomplete.BoolArgument,
			},
			{
				ShortFlag:   "-t",
				Description: "Preserve modification times",
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
				Flags: []*bubblecomplete.Flag{
					{
						ShortFlag:   "-s",
						LongFlag:    "--short",
						Description: "Give output in short format",
						Type:        bubblecomplete.BoolArgument,
					},
					{
						ShortFlag:   "-b",
						LongFlag:    "--branch",
						Description: "Show branch and tracking info",
						Type:        bubblecomplete.BoolArgument,
					},
				},
			},
			{
				Command:     "stash",
				Description: "Stash the changes in a dirty working directory away",
				SubCommands: []*bubblecomplete.Command{
					{
						Command:     "pop",
						Description: "Remove a single stashed state from the stash list and apply it on top of the current working tree state",
						Flags: []*bubblecomplete.Flag{
							{
								LongFlag:    "--index",
								Description: "Try to reinstate index changes as well",
								Type:        bubblecomplete.BoolArgument,
							},
						},
					},
					{
						Command:     "apply",
						Description: "Like pop, but do not remove the state from the stash list",
						Flags: []*bubblecomplete.Flag{
							{
								LongFlag:    "--index",
								Description: "Try to reinstate index changes as well",
								Type:        bubblecomplete.BoolArgument,
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
				Flags: []*bubblecomplete.Flag{
					{
						ShortFlag:   "-f",
						LongFlag:    "--force",
						Description: "Force push even if remote has diverged",
						Type:        bubblecomplete.BoolArgument,
					},
					{
						ShortFlag:   "-u",
						LongFlag:    "--set-upstream",
						Description: "Set upstream tracking reference",
						Type:        bubblecomplete.BoolArgument,
					},
					{
						LongFlag:    "--tags",
						Description: "Push all tags",
						Type:        bubblecomplete.BoolArgument,
					},
				},
			},
			{
				Command:     "pull",
				Description: "Fetch from and integrate with another repository or a local branch",
				PositionalArguments: []*bubblecomplete.PositionalArgument{
					{
						Name:        "remote",
						Description: "Remote to pull from",
						Type:        bubblecomplete.StringArgument,
						Required:    false,
					},
					{
						Name:        "branch",
						Description: "Branch to pull",
						Type:        bubblecomplete.StringArgument,
						Required:    false,
					},
				},
				Flags: []*bubblecomplete.Flag{
					{
						LongFlag:    "--rebase",
						Description: "Rebase current branch on top of upstream",
						Type:        bubblecomplete.BoolArgument,
					},
					{
						LongFlag:    "--no-rebase",
						Description: "Merge instead of rebasing",
						Type:        bubblecomplete.BoolArgument,
					},
				},
			},
			{
				Command:     "clone",
				Description: "Clone a repository into a new directory",
				PositionalArguments: []*bubblecomplete.PositionalArgument{
					{
						Name:        "repository",
						Description: "Repository URL to clone",
						Type:        bubblecomplete.StringArgument,
						Required:    true,
					},
					{
						Name:        "directory",
						Description: "Directory to clone into",
						Type:        bubblecomplete.StringArgument,
						Required:    false,
					},
				},
				Flags: []*bubblecomplete.Flag{
					{
						LongFlag:    "--depth",
						Description: "Create a shallow clone with history truncated",
						Type:        bubblecomplete.IntArgument,
					},
					{
						LongFlag:    "--bare",
						Description: "Make a bare Git repository",
						Type:        bubblecomplete.BoolArgument,
					},
					{
						ShortFlag:   "-b",
						LongFlag:    "--branch",
						Description: "Point HEAD at a specific branch",
						Type:        bubblecomplete.StringArgument,
					},
				},
			},
			{
				Command:     "checkout",
				Description: "Switch branches or restore working tree files",
				PositionalArguments: []*bubblecomplete.PositionalArgument{
					{
						Name:        "branch",
						Description: "Branch or commit to check out",
						Type:        bubblecomplete.StringArgument,
						Required:    false,
					},
				},
				Flags: []*bubblecomplete.Flag{
					{
						ShortFlag:   "-b",
						Description: "Create and checkout a new branch",
						Type:        bubblecomplete.StringArgument,
					},
					{
						LongFlag:    "--track",
						Description: "Set up tracking for the new branch",
						Type:        bubblecomplete.BoolArgument,
					},
					{
						LongFlag:    "--detach",
						Description: "Detach HEAD at the named commit",
						Type:        bubblecomplete.BoolArgument,
					},
				},
			},
			{
				Command:     "branch",
				Description: "List, create, or delete branches",
				PositionalArguments: []*bubblecomplete.PositionalArgument{
					{
						Name:        "name",
						Description: "Branch name to create or filter",
						Type:        bubblecomplete.StringArgument,
						Required:    false,
					},
				},
				Flags: []*bubblecomplete.Flag{
					{
						ShortFlag:   "-d",
						LongFlag:    "--delete",
						Description: "Delete a branch",
						Type:        bubblecomplete.BoolArgument,
					},
					{
						ShortFlag:   "-D",
						Description: "Force delete a branch",
						Type:        bubblecomplete.BoolArgument,
					},
					{
						ShortFlag:   "-a",
						LongFlag:    "--all",
						Description: "List both local and remote branches",
						Type:        bubblecomplete.BoolArgument,
					},
					{
						ShortFlag:   "-r",
						LongFlag:    "--remotes",
						Description: "List remote-tracking branches",
						Type:        bubblecomplete.BoolArgument,
					},
				},
			},
			{
				Command:     "merge",
				Description: "Join two or more development histories together",
				PositionalArguments: []*bubblecomplete.PositionalArgument{
					{
						Name:        "branch",
						Description: "Branch to merge into current branch",
						Type:        bubblecomplete.StringArgument,
						Required:    false,
					},
				},
				Flags: []*bubblecomplete.Flag{
					{
						LongFlag:    "--no-ff",
						Description: "Create a merge commit even for fast-forward",
						Type:        bubblecomplete.BoolArgument,
					},
					{
						LongFlag:    "--squash",
						Description: "Squash commits into a single commit",
						Type:        bubblecomplete.BoolArgument,
					},
					{
						LongFlag:    "--abort",
						Description: "Abort the current merge",
						Type:        bubblecomplete.BoolArgument,
					},
				},
			},
			{
				Command:     "rebase",
				Description: "Reapply commits on top of another base tip",
				PositionalArguments: []*bubblecomplete.PositionalArgument{
					{
						Name:        "upstream",
						Description: "Upstream branch to rebase onto",
						Type:        bubblecomplete.StringArgument,
						Required:    false,
					},
				},
				Flags: []*bubblecomplete.Flag{
					{
						ShortFlag:   "-i",
						LongFlag:    "--interactive",
						Description: "Make a list of commits to be rebased and let the user edit",
						Type:        bubblecomplete.BoolArgument,
					},
					{
						LongFlag:    "--onto",
						Description: "Starting point to create new commits onto",
						Type:        bubblecomplete.StringArgument,
					},
					{
						LongFlag:    "--abort",
						Description: "Abort the rebase operation",
						Type:        bubblecomplete.BoolArgument,
					},
					{
						LongFlag:    "--continue",
						Description: "Continue the rebase after resolving conflicts",
						Type:        bubblecomplete.BoolArgument,
					},
				},
			},
			{
				Command:     "tag",
				Description: "Create, list, delete or verify a tag object signed with GPG",
				PositionalArguments: []*bubblecomplete.PositionalArgument{
					{
						Name:        "tagname",
						Description: "Tag name to create",
						Type:        bubblecomplete.StringArgument,
						Required:    false,
					},
				},
				Flags: []*bubblecomplete.Flag{
					{
						ShortFlag:   "-a",
						LongFlag:    "--annotate",
						Description: "Make an annotated tag",
						Type:        bubblecomplete.BoolArgument,
					},
					{
						ShortFlag:   "-d",
						LongFlag:    "--delete",
						Description: "Delete existing tags",
						Type:        bubblecomplete.BoolArgument,
					},
					{
						ShortFlag:   "-m",
						LongFlag:    "--message",
						Description: "Tag message",
						Type:        bubblecomplete.StringArgument,
					},
					{
						ShortFlag:   "-l",
						LongFlag:    "--list",
						Description: "List tags matching a pattern",
						Type:        bubblecomplete.BoolArgument,
					},
				},
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
		Command:     "ps",
		Description: "Example PowerShell-style command",
		PositionalArguments: []*bubblecomplete.PositionalArgument{
			{
				Name:        "Input",
				Description: "Input file to process",
				Type:        bubblecomplete.FileArgument,
				Required:    true,
			},
		},
		Flags: []*bubblecomplete.Flag{
			{
				PsFlag:      "-stringarg",
				Description: "Example flag showing PowerShell style",
				Type:        bubblecomplete.StringArgument,
			},
			{
				PsFlag:      "-boolarg",
				Description: "Example boolean argument",
				Type:        bubblecomplete.BoolArgument,
			},
			{
				PsFlag:      "-floatarg",
				Description: "Example float argument",
				Type:        bubblecomplete.FloatArgument,
			},
			{
				PsFlag:      "-intarg",
				Description: "Example int argument",
				Type:        bubblecomplete.IntArgument,
			},
			{
				PsFlag:      "-filearg",
				Description: "Example file argument",
				Type:        bubblecomplete.FileArgument,
			},
			{
				PsFlag:      "-dirarg",
				Description: "Example directory argument",
				Type:        bubblecomplete.DirArgument,
			},
			{
				PsFlag:      "-FileDirArg",
				Description: "Example file or directory argument",
				Type:        bubblecomplete.FileDirArgument,
			},
		},
	},
}
