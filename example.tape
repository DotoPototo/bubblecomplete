# VHS documentation: https://github.com/charmbracelet/vhs

Output ./.github/example.gif

Set Shell "zsh"
Set FontSize 22
Set Width 1600
Set Height 500
Set WindowBar Colorful
Set TypingSpeed 100ms

Hide
Type "go run ./example"
Enter
Show
Sleep 1s

# Browse subcommands: type the prefix, cycle through with Tab to see options,
# then pick the one you want and continue typing the rest.
Type "git c"
Sleep 650ms
Tab
Sleep 500ms
Tab
Sleep 500ms
Tab
Sleep 800ms
Type " --am"
Sleep 600ms
Tab
Sleep 800ms
Enter
Sleep 800ms

# Filesystem multi-match: pick a source by Tab-cycling through real files,
# then jump into the second positional and Tab the destination directory.
# Directory completions don't add a trailing space, so the dir goes green.
Type "cp p"
Sleep 800ms
Tab
Sleep 650ms
Tab
Sleep 650ms
Tab
Sleep 800ms
Type "exa"
Sleep 600ms
Tab
Sleep 1200ms
Enter
Sleep 800ms

# Missing required argument — a different validation error kind
# from the "unknown command" case.
Type "cp parse.go"
Sleep 600ms
Enter
Sleep 2500ms
