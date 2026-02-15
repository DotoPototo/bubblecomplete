# Charm TUI Patterns Reference

> Distilled from studying [Crush](https://github.com/charmbracelet/crush) - Charm's production TUI app.
> All file paths are relative to the Crush codebase root.
>
> **Pinned to commit [`af86738`](https://github.com/charmbracelet/crush/tree/af86738e0d6edfeb0ae7fb239b8bdabd4c162fca)** (2025)
> Browse any file at this snapshot: `https://github.com/charmbracelet/crush/blob/af86738e0d/<path>`
> e.g. [`internal/ui/styles/styles.go`](https://github.com/charmbracelet/crush/blob/af86738e0d/internal/ui/styles/styles.go)

---

## Table of Contents

1. [Architecture & File Map](#architecture--file-map)
2. [Color System](#color-system)
3. [Style Organization](#style-organization)
4. [Markdown & Syntax Highlighting](#markdown--syntax-highlighting)
5. [Layout Patterns](#layout-patterns)
6. [Component Patterns](#component-patterns)
7. [Unicode Icons & Symbols](#unicode-icons--symbols)
8. [Gradient System](#gradient-system)
9. [Animation System](#animation-system)
10. [UX Patterns](#ux-patterns)
11. [Dialog System](#dialog-system)
12. [Completions / Autocomplete](#completions--autocomplete)
13. [Keybinding System](#keybinding-system)
14. [Responsive Design](#responsive-design)
15. [Performance Patterns](#performance-patterns)
16. [View & Rendering Pipeline](#view--rendering-pipeline)
17. [Quick Reference: Style Recipes](#quick-reference-style-recipes)
18. [Design Principles](#design-principles)

---

## Architecture & File Map

### Core Style & Theme System

| File | Purpose |
|------|---------|
| `internal/ui/styles/styles.go` | **PRIMARY** - Massive `Styles` struct (~440 fields), `DefaultStyles()` constructor, all icon constants, Chroma theme conversion |
| `internal/ui/styles/grad.go` | Gradient color blending utilities (HCL color space via `go-colorful`) |

### Common UI Elements

| File | Purpose |
|------|---------|
| `internal/ui/common/common.go` | Shared `Common` struct (App + Styles), rect helpers (`CenterRect`, `BottomLeftRect`), clipboard, file size checks |
| `internal/ui/common/elements.go` | `Section()` headers with line fills, `ModelInfo()` with token/cost display, `Status()` lines, `DialogTitle()` with gradient diagonals, `PrettyPath()` |
| `internal/ui/common/button.go` | `Button()` with underline support for keyboard shortcut hints, `ButtonGroup()` for rows of selectable buttons |
| `internal/ui/common/scrollbar.go` | `Scrollbar()` with thumb/track rendering using `┃` and `│` |
| `internal/ui/common/diff.go` | Diff formatter configuration |
| `internal/ui/common/markdown.go` | Glamour markdown renderer setup (normal + plain/thinking variants) |
| `internal/ui/common/highlight.go` | Chroma syntax highlighting wrapper |

### Main UI & Views

| File | Purpose |
|------|---------|
| `internal/ui/model/ui.go` | **MASSIVE** (~2400+ lines) - Main Bubble Tea model, `UI` struct, `Init()`, `Update()`, `View()`, `Draw()`, `generateLayout()`, focus management, key handling, message handling, completions, prompt history |
| `internal/ui/model/header.go` | Header bar: gradient logo, diagonal fills, context %, working dir, keystroke hints, LSP error counts |
| `internal/ui/model/status.go` | Bottom status bar: help hints + timed status messages (error/warn/info/success/update), auto-clearing TTL |
| `internal/ui/model/sidebar.go` | Right sidebar: logo, session title, working dir, model info, files/LSP/MCP sections with dynamic height allocation |
| `internal/ui/model/pills.go` | Pills/tabs: To-Do progress + Queue count, expandable with gradient triangles, spinner, section switching |
| `internal/ui/model/landing.go` | Landing screen: working dir, model info, two-column LSP/MCP status |
| `internal/ui/model/chat.go` | Chat view: message list, scrolling, follow mode, animation management, mouse handling |
| `internal/ui/model/keys.go` | All keybinding definitions organized by context (Editor, Chat, Initialize, Global) |
| `internal/ui/model/session.go` | Session management UI |

### Chat & Message Rendering

| File | Purpose |
|------|---------|
| `internal/ui/chat/messages.go` | `MessageItem` interface, `ExtractMessageItems()` factory, `BuildToolResultMap()` |
| `internal/ui/chat/user.go` | User message rendering (left border focus states, markdown) |
| `internal/ui/chat/assistant.go` | Assistant message rendering (thinking box with expand/collapse, streaming animation, markdown content, error display, duration footer) |
| `internal/ui/chat/tools.go` | **MASSIVE** (~1400 lines) - All tool call rendering: icons per state, parameter display, output truncation, expandable content, diffs, code blocks |
| `internal/ui/chat/bash.go` | Bash tool renderer |
| `internal/ui/chat/file.go` | File tool renderer |
| `internal/ui/chat/agent.go` | Sub-agent tool renderer with nested tool display |
| `internal/ui/chat/todos.go` | Todo tool renderer |
| `internal/ui/chat/mcp.go` | MCP tool renderer |

### Specialized Components

| File | Purpose |
|------|---------|
| `internal/ui/anim/anim.go` | Gradient-cycling animation system (20 FPS, prerendered frames, staggered entrance, ellipsis animation, caching) |
| `internal/ui/diffview/diffview.go` | Full diff viewer (~775 lines): unified + split modes, syntax highlighting, line numbers |
| `internal/ui/diffview/style.go` | Diff color schemes with insert/delete/equal/divider/missing line styles |
| `internal/ui/logo/logo.go` | Logo wordmark with block-character letterforms, gradient, random stretching, `SmallRender()` for tight spaces |
| `internal/ui/attachments/attachments.go` | File attachment chips with icon badges, delete mode with numbered selection |
| `internal/ui/completions/completions.go` | Autocomplete popup: file/MCP resource completions, fuzzy filtering, circular navigation |
| `internal/ui/dialog/dialog.go` | Dialog stack system (`Overlay` manages LIFO dialog stack), center/bottom-left positioning |
| `internal/ui/list/list.go` | Generic scrollable list: lazy rendering, scroll management, reverse mode, render callbacks, gap control |

---

## Color System

### Palette Organization (from `internal/ui/styles/styles.go`)

Uses **Charmtone** (`github.com/charmbracelet/x/exp/charmtone`) for named colors:

```
SEMANTIC NAME    CHARMTONE COLOR    ROLE
─────────────    ───────────────    ────
Primary          Charple            Brand purple/violet - borders, accents, focused borders
Secondary        Dolly              Gold/yellow - highlights, badges, cursor, logo gradient start
Tertiary         Bok                Light green - success, checks, input prompts

BgBase           Pepper             Dark base background (app background)
BgBaseLighter    BBQ                Slightly lighter dark background (thinking box, content bg)
BgSubtle         Charcoal           Subtle background distinction (buttons blur, content panel)
BgOverlay        Iron               Overlay/modal backgrounds (pill border)

FgBase           Ash                Primary text (light gray)
FgMuted          Squid              Secondary text (medium gray)
FgHalfMuted      Smoke              Between base and muted
FgSubtle         Oyster             Very subtle text (hints, descriptions)

Border           Charcoal           Default border color
BorderFocus      Charple            Focused border color (= Primary)

Error            Sriracha           Error red
Warning          Zest               Warning orange
Info             Malibu             Info blue

White            Butter             Pure white equivalent
Red              Coral              Standard red
RedDark          Sriracha           Dark red (error tags, delete indicators)
Green            Julep              Standard green (success indicators)
GreenLight       Bok                Light green
GreenDark        Guac               Dark green (pending tool icons, editor prompt)
Blue             Malibu             Standard blue (tool names, info)
BlueLight        Sardine            Light blue (agent task tags)
BlueDark         Damson             Dark blue (todo ratio, job action)
Yellow           Mustard            Standard yellow (warning indicator)
```

### Additional Charmtone Colors Used

```
CHARMTONE COLOR  USAGE
───────────────  ─────
Salt             Text selection foreground, class names (bold+underline)
Zinc             Link color
Bengal           Comment preprocessor
Pony             Keyword reserved/namespace
Guppy            Keyword type
Salmon           Operator highlighting
Cumin            String literals
Mauve            Tag names
Hazy             Attribute names
Citron           Decorators, yolo icon bg, renaming gradient, name decorator
Cheeky           Image styling, builtin names
Cherry           (Available, commented out)
```

### Key Pattern: Semantic Naming

Colors are NEVER referred to by appearance (`"lightPurple"`). Always by purpose (`Primary`, `Error`, `FgMuted`). This makes themes trivial to swap.

### Storage Pattern

Colors stored as `color.Color` in the `Styles` struct, allowing:
- Gradient blending between any two colors
- Type-safe usage across the app
- Easy theme swapping
- Direct use with `lipgloss.Style.Foreground()` / `.Background()`

---

## Style Organization

### Hierarchical Struct Pattern (from `internal/ui/styles/styles.go`)

Styles are organized in a deeply nested struct, grouped by component. The full struct has ~440 fields:

```go
type Styles struct {
    // Global window
    WindowTooSmall lipgloss.Style

    // Base reusable styles
    Base, Muted, HalfMuted, Subtle lipgloss.Style

    // Tags (colored badge backgrounds)
    TagBase  lipgloss.Style  // Padding(0,1), Foreground(white)
    TagError lipgloss.Style  // extends TagBase with Background(redDark)
    TagInfo  lipgloss.Style  // extends TagBase with Background(blueLight)

    // Semantic colors as fields
    Primary, Secondary, Tertiary   color.Color
    BgBase, BgBaseLighter          color.Color
    BgSubtle, BgOverlay            color.Color
    FgBase, FgMuted                color.Color
    FgHalfMuted, FgSubtle          color.Color
    Border, BorderColor            color.Color
    Error, Warning, Info           color.Color
    White, Blue, BlueDark          color.Color
    BlueLight, Green, GreenLight   color.Color
    GreenDark, Red, RedDark        color.Color
    Yellow                         color.Color
    Background                     color.Color  // App background

    // Logo color fields
    LogoFieldColor   color.Color  // Diagonal lines color
    LogoTitleColorA  color.Color  // Left gradient point
    LogoTitleColorB  color.Color  // Right gradient point
    LogoCharmColor   color.Color  // "Charm™" text
    LogoVersionColor color.Color  // Version text

    // Component groups (nested structs) ...
    Header struct { ... }
    CompactDetails struct { ... }
    Chat struct { Message struct { ... } }
    Tool struct { ... }
    Dialog struct { Help struct { ... }; Sessions struct { ... }; Arguments struct { ... } }
    Status struct { ... }
    Pills struct { ... }
    Section struct { ... }
    Initialize struct { ... }
    LSP struct { ... }
    Files struct { ... }
    Completions struct { ... }
    Attachments struct { ... }

    // Focus/blur border pairs
    BorderFocus, BorderBlur lipgloss.Style
    FocusedMessageBorder    lipgloss.Border

    // Button styles
    ButtonFocus lipgloss.Style  // Foreground(white), Background(secondary)
    ButtonBlur  lipgloss.Style  // Foreground(fgBase), Background(bgSubtle)

    // Panel styles
    PanelMuted lipgloss.Style  // Muted on bgBaseLighter
    PanelBase  lipgloss.Style  // Base on bgBase

    // Editor prompt styles (Normal + Yolo variants, each with Focused/Blurred)
    EditorPromptNormalFocused   lipgloss.Style  // SetString("::: ")
    EditorPromptNormalBlurred   lipgloss.Style
    EditorPromptYoloIconFocused lipgloss.Style  // SetString(" ! "), Citron bg
    EditorPromptYoloIconBlurred lipgloss.Style
    EditorPromptYoloDotsFocused lipgloss.Style  // SetString(":::"), Zest fg
    EditorPromptYoloDotsBlurred lipgloss.Style

    // Radio buttons
    RadioOn  lipgloss.Style  // SetString("◉")
    RadioOff lipgloss.Style  // SetString("○")

    // Text selection
    TextSelection lipgloss.Style  // Foreground(Salt), Background(Charple)

    // Input component styles (bubbles)
    TextInput textinput.Styles  // Focused/Blurred states + Cursor config
    TextArea  textarea.Styles   // Focused/Blurred states + Cursor config

    // Help, Diff, FilePicker, Markdown styles (from bubbles/glamour)
    Help       help.Styles
    Diff       diffview.Style
    FilePicker filepicker.Styles
    Markdown      ansi.StyleConfig  // Normal markdown
    PlainMarkdown ansi.StyleConfig  // Muted markdown for thinking content

    // Resource status indicators (LSP/MCP)
    ResourceGroupTitle     lipgloss.Style
    ResourceOfflineIcon    lipgloss.Style  // SetString("●"), Iron fg
    ResourceBusyIcon       lipgloss.Style  // SetString("●"), Citron fg
    ResourceErrorIcon      lipgloss.Style  // SetString("●"), Coral fg
    ResourceOnlineIcon     lipgloss.Style  // SetString("●"), Guac fg
    ResourceName           lipgloss.Style
    ResourceStatus         lipgloss.Style
    ResourceAdditionalText lipgloss.Style
}
```

### Key Patterns

**1. Base style composition:**
```go
base := lipgloss.NewStyle().Foreground(fgBase)
s.Muted = lipgloss.NewStyle().Foreground(fgMuted)
s.TagBase = lipgloss.NewStyle().Padding(0, 1).Foreground(white)
s.TagError = s.TagBase.Background(redDark)  // extends TagBase
```

**2. Focus/blur pairs (for every interactive element):**
```go
s.ButtonFocus = lipgloss.NewStyle().Foreground(white).Background(secondary)
s.ButtonBlur  = s.Base.Background(bgSubtle)

s.Chat.Message.UserBlurred = base.PaddingLeft(1).BorderLeft(true).
    BorderForeground(primary).BorderStyle(normalBorder)
s.Chat.Message.UserFocused = base.PaddingLeft(1).BorderLeft(true).
    BorderForeground(primary).BorderStyle(lipgloss.Border{Left: "▌"})  // thick bar
```

**3. SetString for icon styles:**
```go
s.ToolCallPending = lipgloss.NewStyle().Foreground(greenDark).SetString(ToolPending)
s.ToolCallSuccess = lipgloss.NewStyle().Foreground(green).SetString(ToolSuccess)
s.ToolCallError = lipgloss.NewStyle().Foreground(redDark).SetString(ToolError)
// Usage: just call .String() - no need to pass text
icon := s.ToolCallSuccess.String()
```

**4. Single constructor function:**
All styles initialized in one `DefaultStyles()` function. No scattered style definitions.

**5. Cursor configuration:**
```go
s.TextInput = textinput.Styles{
    Focused: textinput.StyleState{
        Text:        base,
        Placeholder: base.Foreground(fgSubtle),
        Prompt:      base.Foreground(tertiary),  // green prompt
    },
    Cursor: textinput.CursorStyle{
        Color: secondary,     // gold cursor
        Shape: tea.CursorBlock,
        Blink: true,
    },
}
```

---

## Markdown & Syntax Highlighting

### Glamour Markdown Styles (from `styles.go`)

Two markdown configurations: normal (colorful) and plain (muted, for thinking content).

**Normal Markdown key styles:**
```go
Document:  Color = Smoke (fgHalfMuted)
Heading:   Color = Malibu (blue), Bold
H1:        Color = Zest, BackgroundColor = Charple, Bold, Prefix/Suffix = " "
H2-H5:     Prefix = "## " etc.
H6:        Color = Guac (greenDark), not bold
Code:      Color = Coral (red), BackgroundColor = Charcoal, Prefix/Suffix = " "
CodeBlock: Margin = 2, full Chroma syntax highlighting
Link:      Color = Zinc, Underline
LinkText:  Color = Guac, Bold
Image:     Color = Cheeky, Underline
Item:      BlockPrefix = "• "
Task:      Ticked = "[✓] ", Unticked = "[ ] "
HorizRule: Format = "\n--------\n", Color = Charcoal
BlockQuote: IndentToken = "│ ", Indent = 1
```

**Plain Markdown (for thinking box):**
- All elements use `fgMuted` foreground + `bgBaseLighter` background
- No syntax highlighting in code blocks
- Creates a visually subdued but readable thinking content area

### Chroma Syntax Highlighting Colors

```
TOKEN TYPE       CHARMTONE COLOR
──────────       ───────────────
Text             Smoke
Error            Butter on Sriracha
Comment          Oyster
CommentPreproc   Bengal
Keyword          Malibu
KeywordReserved  Pony
KeywordNamespace Pony
KeywordType      Guppy
Operator         Salmon
Punctuation      Zest
Name             Smoke
NameBuiltin      Cheeky
NameTag          Mauve
NameAttribute    Hazy
NameClass        Salt (bold, underline)
NameDecorator    Citron
NameFunction     Guac
LiteralNumber    Julep
LiteralString    Cumin
LiteralStringEsc Bok
GenericDeleted   Coral
GenericInserted  Guac
Background       Charcoal
```

### Chroma Theme Conversion

```go
// The Styles struct has a ChromaTheme() method that converts Glamour's
// Chroma styles to a chroma.StyleEntries map for use with the Chroma library
func (s *Styles) ChromaTheme() chroma.StyleEntries { ... }
```

---

## Layout Patterns

### Ultraviolet Screen-Based Rendering

Crush uses the `charmbracelet/ultraviolet` library for screen-buffer rendering instead of string concatenation:

```go
import (
    uv "github.com/charmbracelet/ultraviolet"
    "github.com/charmbracelet/ultraviolet/layout"
    "github.com/charmbracelet/ultraviolet/screen"
)

// Create a screen buffer
canvas := uv.NewScreenBuffer(width, height)

// Draw components at specific positions
view := uv.NewStyledString(renderedContent)
view.Draw(scr, rectangle)

// Clear screen
screen.Clear(scr)
```

### Rectangle-Based Splitting (via `ultraviolet/layout`)

```go
// Vertical split (top/bottom)
headerRect, mainRect := layout.SplitVertical(area, layout.Fixed(headerHeight))

// Horizontal split (left/right)
mainRect, sideRect := layout.SplitHorizontal(appRect, layout.Fixed(appRect.Dx()-sidebarWidth))

// Chained splits for complex layouts
mainRect, editorRect := layout.SplitVertical(mainRect, layout.Fixed(mainRect.Dy()-editorH))
```

### Layout State Struct

```go
type uiLayout struct {
    area           uv.Rectangle  // Overall available area
    header         uv.Rectangle  // Header (compact mode / landing / init)
    main           uv.Rectangle  // Main content (chat, configure, landing)
    pills          uv.Rectangle  // Pills/tabs panel
    editor         uv.Rectangle  // Input editor
    sidebar        uv.Rectangle  // Right sidebar
    status         uv.Rectangle  // Bottom help/status
    sessionDetails uv.Rectangle  // Session details overlay (compact mode)
}
```

### Complete Layout Algorithm (`generateLayout`)

```go
func (m *UI) generateLayout(w, h int) uiLayout {
    area := image.Rect(0, 0, w, h)

    // Fixed dimensions
    helpHeight    := 1          // Status/help bar
    editorHeight  := 5          // Text editor
    sidebarWidth  := 30         // Right sidebar
    landingHeaderHeight := 4    // Logo header

    // Dynamic help height when expanded
    if m.status.ShowingAll() {
        for _, row := range helpKeyMap.FullHelp() {
            helpHeight = max(helpHeight, len(row))
        }
    }

    // App margins
    appRect, helpRect := layout.SplitVertical(area, layout.Fixed(area.Dy()-helpHeight))
    appRect.Min.Y += 1  // Top margin
    appRect.Max.Y -= 1  // Bottom margin
    appRect.Min.X += 1  // Left margin
    appRect.Max.X -= 1  // Right margin

    // Extra padding for landing/onboarding
    if state in [uiOnboarding, uiInitialize, uiLanding] {
        appRect.Min.X += 1
        appRect.Max.X -= 1
    }

    // State-specific layouts...
}
```

### Layout Modes

**Onboarding / Initialize:**
```
┌─────────────────┐
│     Header      │  (4 rows)
├─────────────────┤
│      Main       │
├─────────────────┤
│      Help       │  (1 row)
└─────────────────┘
```

**Landing:**
```
┌─────────────────┐
│     Header      │  (4 rows, logo)
├─────────────────┤
│   Main (info)   │  (model, LSP, MCP)
├─────────────────┤
│     Editor      │  (5 rows)
├─────────────────┤
│      Help       │  (1 row)
└─────────────────┘
```

**Chat - Full mode (width >= 120):**
```
┌──────────┬──────┐
│          │      │
│   Chat   │ Side │  (30 cols)
│          │ bar  │
├──────────┤      │
│  Pills   │      │  (optional, dynamic height)
├──────────┤      │
│  Editor  │      │  (5 rows)
├──────────┴──────┤
│      Help       │  (1 row)
└─────────────────┘
```

**Chat - Compact mode (width < 120 OR height < 30):**
```
┌─────────────────┐
│  Compact Header │  (1 row)
├─────────────────┤
│      Chat       │
├─────────────────┤
│     Pills       │  (optional)
├─────────────────┤
│     Editor      │  (5 rows)
├─────────────────┤
│      Help       │  (1 row)
└─────────────────┘
```

### lipgloss Join Patterns

```go
// Horizontal: align items along a shared axis
row := lipgloss.JoinHorizontal(lipgloss.Top, left, " ", right)
row := lipgloss.JoinHorizontal(lipgloss.Center, pillsRow, " ", helpHint)

// Vertical: stack items
col := lipgloss.JoinVertical(lipgloss.Left, header, "", content, "", footer)

// Nested: build complex grids
leftCol := lipgloss.JoinVertical(lipgloss.Left, files, "", lsp)
rightCol := lipgloss.JoinVertical(lipgloss.Left, mcp, "", details)
grid := lipgloss.JoinHorizontal(lipgloss.Left, leftCol, " ", rightCol)
```

### Width Calculation Patterns

```go
// Available content width (subtract frame sizes)
contentW := area.Dx() - style.GetHorizontalFrameSize()

// Multi-column distribution
sectionW := min(maxSectionWidth, contentW/3 - 2)

// Remaining width after fixed elements
remaining := width - lipgloss.Width(badge) - lipgloss.Width(details) - padding

// Check if content fits on one line
if lipgloss.Width(modelWithProvider) <= width {
    firstLine = modelWithProvider
} else {
    // Put provider on next line
}

// Width and height together
width, height := lipgloss.Size(view)
```

---

## Component Patterns

### Badges / Tags (from `styles.go`)

```go
// Pattern: foreground on colored background, padding 0,1
s.TagBase  = base.Padding(0, 1).Foreground(white)
s.TagError = s.TagBase.Background(redDark)
s.TagInfo  = s.TagBase.Background(blueLight)

// Status indicator badges (bold, with SetString)
s.Status.SuccessIndicator = base.Foreground(bgSubtle).Background(green).Padding(0, 1).Bold(true).SetString("OKAY!")
s.Status.ErrorIndicator   = s.Status.SuccessIndicator.Foreground(bgBase).Background(red).SetString("ERROR")
s.Status.WarnIndicator    = s.Status.SuccessIndicator.Foreground(bgOverlay).Background(yellow).SetString("WARNING")
s.Status.UpdateIndicator  = s.Status.SuccessIndicator.SetString("HEY!")

// Status messages (fill remaining width)
s.Status.SuccessMessage = base.Foreground(bgSubtle).Background(greenDark).Padding(0, 1)
s.Status.ErrorMessage   = s.Status.SuccessMessage.Foreground(white).Background(redDark)
s.Status.WarnMessage    = s.Status.SuccessMessage.Foreground(bgOverlay).Background(warning)
```

**Rendering:**
```go
errTag := sty.TagError.Render("ERROR")
okTag  := sty.Status.SuccessIndicator.String()  // Uses SetString
```

### Pills / Chips (from `pills.go`)

```go
// Pill with rounded border when focused
s.Pills.Focused = base.Padding(0, 1).
    BorderStyle(lipgloss.RoundedBorder()).
    BorderForeground(bgOverlay)

s.Pills.Blurred = base.Padding(0, 1).
    BorderStyle(lipgloss.HiddenBorder())  // Same size, no visible border

// Pill content with gradient triangles
triangles := styles.ForegroundGrad(t, "▶▶▶▶▶▶▶▶▶", false, t.RedDark, t.Secondary)
if queue < len(triangles) {
    triangles = triangles[:queue]  // Trim to match count
}
content := fmt.Sprintf("%s %s", strings.Join(triangles, ""), text)

// Todo pill with spinner and active task name
func todoPill(todos []session.Todo, spinnerView string, focused, panelFocused bool, t *styles.Styles) string {
    label := t.Base.Render("To-Do")
    progress := t.Muted.Render(fmt.Sprintf("%d/%d", completed, total))
    if currentTodo != nil {
        task := t.Subtle.Render(taskText)
        content = fmt.Sprintf("%s %s %s  %s", spinnerView, label, progress, task)
    }
}

// Pills row with help hint
pillsRow := lipgloss.JoinHorizontal(lipgloss.Top, pills...)
helpKey := t.Pills.HelpKey.Render("ctrl+t")
helpText := t.Pills.HelpText.Render("open")
pillsRow = lipgloss.JoinHorizontal(lipgloss.Center, pillsRow, " ", helpKey, " ", helpText)
```

### Attachment Chips (from `attachments.go`)

Two-part chip: icon badge + label:

```go
// Icon part (colored background) - uses SetString
s.Attachments.Image = base.Foreground(bgSubtle).Background(green).Padding(0,1).SetString(ImageIcon)
s.Attachments.Text  = base.Foreground(bgSubtle).Background(green).Padding(0,1).SetString(TextIcon)

// Label part
s.Attachments.Normal   = base.Padding(0,1).MarginRight(1).Background(fgMuted).Foreground(fgBase)
s.Attachments.Deleting = base.Padding(0,1).Bold(true).Background(red).Foreground(fgBase)

// Compose chip
chips = append(chips, icon.String(), normalStyle.Render(filename))
result := lipgloss.JoinHorizontal(lipgloss.Left, chips...)

// Filename truncation
const maxFilename = 15
if ansi.StringWidth(filename) > maxFilename {
    filename = ansi.Truncate(filename, maxFilename, "…")
}

// Delete mode: show numbered indices instead of icons
if deleting {
    chips = append(chips, r.deletingStyle.Render(fmt.Sprintf("%d", i)), r.normalStyle.Render(filename))
}

// Overflow: show "N more…" when too many attachments
maxItemWidth := lipgloss.Width(iconStyle.String() + normalStyle.Render(strings.Repeat("x", maxFilename)))
fits := int(math.Floor(float64(width)/float64(maxItemWidth))) - 1
if i == fits && len(attachments) > i {
    chips = append(chips, fmt.Sprintf("%d more…", len(attachments)-fits))
    break
}
```

### Buttons (from `button.go`)

```go
type ButtonOpts struct {
    Text           string
    UnderlineIndex int   // Which char to underline (-1 for none)
    Selected       bool
    Padding        int   // Defaults to 2
}

// Focus/blur styles
style := t.ButtonBlur
if opts.Selected { style = t.ButtonFocus }
text = style.Padding(0, opts.Padding).Render(text)

// Underline the shortcut character using lipgloss.StyleRanges
text = lipgloss.StyleRanges(text,
    lipgloss.NewRange(padding+idx, padding+idx+1, style.Underline(true)))

// Button group with configurable spacing
func ButtonGroup(t *styles.Styles, buttons []ButtonOpts, spacing string) string
// spacing: "  " for horizontal, "\n" for vertical
```

### Section Headers (from `elements.go`)

Label with line fill and optional info:

```go
func Section(t *styles.Styles, text string, width int, info ...string) string {
    char := styles.SectionSeparator  // "─"
    length := lipgloss.Width(text) + 1
    remainingWidth := width - length

    // Optional info text after the line
    if len(info) > 0 {
        infoText = " " + strings.Join(info, " ")
        remainingWidth -= lipgloss.Width(infoText)
    }

    text = t.Section.Title.Render(text)  // Subtle foreground
    if remainingWidth > 0 {
        text += " " + t.Section.Line.Render(strings.Repeat(char, remainingWidth)) + infoText
    }
    return text
}
```

Result: `Files ─────────────────────────────── 3 items`

### Dialog Title with Gradient Diagonals

```go
func DialogTitle(t *styles.Styles, title string, width int, fromColor, toColor color.Color) string {
    char := "╱"
    remainingWidth := width - lipgloss.Width(title) - 1
    if remainingWidth > 0 {
        lines := strings.Repeat(char, remainingWidth)
        lines = styles.ApplyForegroundGrad(t, lines, fromColor, toColor)
        title = title + " " + lines
    }
    return title
}
```

### Model Info Display (from `elements.go`)

```go
func ModelInfo(t *styles.Styles, modelName, providerName, reasoningInfo string,
    context *ModelContextInfo, width int) string {
    // Line 1: ◇ ModelName via ProviderName (or split to 2 lines if too wide)
    // Line 2: Reasoning Medium (indented)
    // Line 3: 45% (123K) $0.42 (indented, with warning icon if >80%)
}

// Token formatting with K/M suffixes
func formatTokensAndCost(t *styles.Styles, tokens, contextWindow int64, cost float64) string {
    // >= 1M: "1.5M", >= 1K: "123K", else raw number
    // Shows percentage, tokens in parens, and dollar cost
    // Warning icon when > 80% context used
}
```

### Status Line (from `elements.go`)

```go
type StatusOpts struct {
    Icon             string
    Title            string
    TitleColor       color.Color
    Description      string
    DescriptionColor color.Color
    ExtraContent     string
}

func Status(t *styles.Styles, opts StatusOpts, width int) string {
    // Icon + Title + Description (truncated) + ExtraContent
    // Description truncated with "…" to fit available width
}
```

### Scrollbar (from `scrollbar.go`)

```go
func Scrollbar(s *styles.Styles, height, contentSize, viewportSize, offset int) string {
    thumbSize := max(1, height*viewportSize/contentSize)
    maxOffset := contentSize - viewportSize
    trackSpace := height - thumbSize
    thumbPos := min(trackSpace, offset*trackSpace/maxOffset)

    for i := range height {
        if i >= thumbPos && i < thumbPos+thumbSize {
            sb.WriteString(s.Dialog.ScrollbarThumb.Render("┃"))  // thick
        } else {
            sb.WriteString(s.Dialog.ScrollbarTrack.Render("│"))  // thin
        }
    }
}
```

### Resource Status Indicators (LSP/MCP)

```go
// Status dot icons using SetString
s.ResourceOfflineIcon = lipgloss.NewStyle().Foreground(charmtone.Iron).SetString("●")
s.ResourceBusyIcon    = s.ResourceOfflineIcon.Foreground(charmtone.Citron)
s.ResourceErrorIcon   = s.ResourceOfflineIcon.Foreground(charmtone.Coral)
s.ResourceOnlineIcon  = s.ResourceOfflineIcon.Foreground(charmtone.Guac)

// LSP diagnostic icons
const (
    LSPErrorIcon   = "E"
    LSPWarningIcon = "W"
    LSPInfoIcon    = "I"
    LSPHintIcon    = "H"
)
```

### Header with Diagonal Fills (from `header.go`)

```go
const headerDiag = "╱"
const minHeaderDiags = 3

// Compact logo: "Charm™ CRUSH" with gradient
h.compactLogo = t.Header.Charm.Render("Charm™") + " " +
    styles.ApplyBoldForegroundGrad(t, "CRUSH", t.Secondary, t.Primary) + " "

// Fill remaining width with diagonals
remainingWidth := width - lipgloss.Width(logo) - lipgloss.Width(details) - padding
b.WriteString(t.Header.Diagonals.Render(
    strings.Repeat(headerDiag, max(minHeaderDiags, remainingWidth))))

// Header details: LSP errors, context %, keystroke hint, working dir
parts = append(parts, formattedPercentage)  // "45%"
parts = append(parts, keystroke+tip)         // "ctrl+d open"
dot := t.Header.Separator.Render(" • ")
metadata := strings.Join(parts, dot)

// Working dir with truncation
cwd = ansi.Truncate(cwd, max(0, availWidth-lipgloss.Width(metadata)), "…")
```

### Logo Wordmark (from `logo.go`)

```go
// Block character letterforms (▄ ▀ █ used for ASCII art)
type letterform func(stretch bool) string

// Random stretching: one letter gets randomly wider each render
stretchIndex := cachedRandN(len(letterforms))
crush := renderWord(spacing, stretchIndex, letterforms...)

// Apply gradient across the whole word
for r := range strings.SplitSeq(crush, "\n") {
    fmt.Fprintln(b, styles.ApplyForegroundGrad(s, r, o.TitleColorA, o.TitleColorB))
}

// Flanked by diagonal fields
leftFieldRow := fg(o.FieldColor, strings.Repeat("╱", leftWidth))
rightField: step-down pattern (each row shorter by 1)

// Compact version for sidebar/small windows
func SmallRender(t *styles.Styles, width int) string {
    title := t.Base.Foreground(t.Secondary).Render("Charm™")
    title += " " + styles.ApplyBoldForegroundGrad(t, "Crush", t.Secondary, t.Primary)
    remainingWidth := width - lipgloss.Width(title) - 1
    if remainingWidth > 0 {
        title += " " + t.Base.Foreground(t.Primary).Render(strings.Repeat("╱", remainingWidth))
    }
}
```

### Sidebar (from `sidebar.go`)

```go
func (m *UI) drawSidebar(scr uv.Screen, area uv.Rectangle) {
    // Height breakpoint for logo size
    const logoHeightBreakpoint = 30
    if height < logoHeightBreakpoint {
        sidebarLogo = logo.SmallRender(m.com.Styles, width)
    }

    // Sidebar content stack
    blocks := []string{
        sidebarLogo,
        title,          // Session title, MaxHeight(2)
        "",
        cwd,            // Working directory
        "",
        m.modelInfo(width),  // Model + provider + tokens + cost
        "",
    }
    sidebarHeader := lipgloss.JoinVertical(lipgloss.Left, blocks...)

    // Dynamic height allocation for sections
    remainingHeight := area.Dy() - lipgloss.Height(sidebarHeader) - 10
    maxFiles, maxLSPs, maxMCPs := getDynamicHeightLimits(remainingHeight)

    // Priority: files > LSP > MCP
    // Each section uses Section() header + status items
}

func getDynamicHeightLimits(availableHeight int) (maxFiles, maxLSPs, maxMCPs int) {
    const minItemsPerSection = 2
    const defaultMaxFilesShown = 10
    const defaultMaxLSPsShown = 8
    const defaultMaxMCPsShown = 8

    // Distribute height equally, then give surplus to higher-priority sections
    heightPerSection := availableHeight / 3
    // Extra space goes: files first, then LSPs, then MCPs
}
```

### Status Bar (from `status.go`)

```go
const DefaultStatusTTL = 5 * time.Second

// Status message types with distinct indicator + message styling
type InfoType: Error, Warn, Info, Update, Success

// Rendering: indicator badge + message, truncated to fit
func (s *Status) Draw(scr uv.Screen, area uv.Rectangle) {
    // 1. Render help view (from bubbles/help)
    helpView := s.com.Styles.Status.Help.Render(s.help.View(s.helpKm))

    // 2. Overlay info message if present
    ind := indStyle.String()  // e.g. " ERROR " badge
    messageWidth := area.Dx() - lipgloss.Width(ind)
    msg := ansi.Truncate(s.msg.Msg, messageWidth, "…")
    info := msgStyle.Width(messageWidth).Render(msg)
    uv.NewStyledString(ind + info).Draw(scr, area)  // Draws OVER help
}

// Auto-clearing with tea.Tick
func clearInfoMsgCmd(ttl time.Duration) tea.Cmd {
    return tea.Tick(ttl, func(time.Time) tea.Msg {
        return util.ClearStatusMsg{}
    })
}
```

### Landing Page (from `landing.go`)

```go
func (m *UI) landingView() string {
    // Single column info section
    parts := []string{cwd, "", modelInfo}
    infoSection := lipgloss.JoinVertical(lipgloss.Left, parts...)

    // Two-column LSP/MCP below
    mcpLspSectionWidth := min(30, (width-1)/2)
    lspSection := m.lspInfo(mcpLspSectionWidth, remainingHeight, false)
    mcpSection := m.mcpInfo(mcpLspSectionWidth, remainingHeight, false)
    content := lipgloss.JoinHorizontal(lipgloss.Left, lspSection, " ", mcpSection)
}
```

---

## Unicode Icons & Symbols

### Complete Icon Set (from `styles.go`)

```
ICON    CONSTANT             UNICODE    USAGE
────    ────────             ───────    ─────
✓       CheckIcon            U+2713     Success, completion
⋯       SpinnerIcon          U+22EF     Loading/thinking (fallback)
⟳       LoadingIcon          U+27F3     Loading state
◇       ModelIcon            U+25C7     Model selection/display
→       ArrowRightIcon       U+2192     Navigation, in-progress
●       ToolPending          U+25CF     Tool awaiting result
✓       ToolSuccess          U+2713     Tool completed
×       ToolError            U+00D7     Tool failed
◉       RadioOn              U+25C9     Selected radio button
○       RadioOff             U+25CB     Unselected radio button
│       BorderThin           U+2502     Thin vertical border
▌       BorderThick          U+258C     Thick vertical border (focus indicator)
─       SectionSeparator     U+2500     Horizontal section line
✓       TodoCompletedIcon    U+2713     Completed todo
•       TodoPendingIcon      U+2022     Pending todo
→       TodoInProgressIcon   U+2192     In-progress todo
■       ImageIcon            U+25A0     Image attachment badge
≡       TextIcon             U+2261     Text attachment badge
┃       ScrollbarThumb       U+2503     Scrollbar thumb (thick)
│       ScrollbarTrack       U+2502     Scrollbar track (thin)
E       LSPErrorIcon                    LSP error count
W       LSPWarningIcon                  LSP warning count
I       LSPInfoIcon                     LSP info count
H       LSPHintIcon                     LSP hint count
●       ResourceOfflineIcon  U+25CF     Offline resource (Iron color)
●       ResourceBusyIcon     U+25CF     Busy resource (Citron color)
●       ResourceErrorIcon    U+25CF     Error resource (Coral color)
●       ResourceOnlineIcon   U+25CF     Online resource (Guac color)
▶       (queue triangles)    U+25B6     Queue pill gradient decoration
```

### Box Drawing Characters Used

```
Character  Unicode   Usage
─────────  ───────   ─────
╱          U+2571    Header fill, diagonal separators, dialog title decoration
─          U+2500    Section separators (SectionSeparator)
│          U+2502    Border, tree connectors, scrollbar track, blockquote indent
┃          U+2503    Scrollbar thumb
▌          U+258C    Focused message border (thick left bar)
( )        Rounded   Panel borders, focused pills (lipgloss.RoundedBorder())
```

### Block Characters (Logo)

```
▄  U+2584  Lower half block (letter tops)
▀  U+2580  Upper half block (letter bottoms, horizontal strokes)
█  U+2588  Full block (vertical strokes)
```

---

## Gradient System

### Implementation (from `grad.go`)

```go
// Per-character foreground gradient
func ForegroundGrad(t *Styles, input string, bold bool, color1, color2 color.Color) []string {
    // Split into grapheme clusters (Unicode-aware via uniseg)
    gr := uniseg.NewGraphemes(input)
    clusters := collect(gr)

    // Blend colors in HCL space (perceptually smooth)
    ramp := blendColors(len(clusters), color1, color2)

    // Apply one color per character
    for i, c := range ramp {
        style := t.Base.Foreground(c)
        if bold { style = style.Bold(true) }
        clusters[i] = style.Render(clusters[i])
    }
    return clusters  // Returns slice for flexible joining
}

// Convenience wrappers
func ApplyForegroundGrad(t *Styles, input string, c1, c2 color.Color) string  // joins the slice
func ApplyBoldForegroundGrad(t *Styles, input string, c1, c2 color.Color) string
```

### HCL Color Blending

```go
func blendColors(size int, stops ...color.Color) []color.Color {
    // Uses colorful.BlendHcl() for perceptually uniform gradients
    // Supports multiple color stops, evenly distributed
    // Segment sizes distributed with remainder handling

    for j := range segmentSize {
        t := float64(j) / float64(segmentSize-1)
        c := c1.BlendHcl(c2, t)
        blended = append(blended, c)
    }
}
```

### Usage Examples

```go
// Logo gradient
styles.ApplyForegroundGrad(s, logoLine, colorA, colorB)

// Bold gradient text (brand name)
styles.ApplyBoldForegroundGrad(t, "CRUSH", t.Secondary, t.Primary)

// Gradient decorative triangles (for queue pills)
triangles := styles.ForegroundGrad(t, "▶▶▶▶▶▶▶▶▶", false, t.RedDark, t.Secondary)

// Dialog title gradient diagonals
lines := styles.ApplyForegroundGrad(t, "╱╱╱╱╱╱╱╱", fromColor, toColor)

// Small logo with gradient
styles.ApplyBoldForegroundGrad(t, "Crush", t.Secondary, t.Primary)
```

### Key Insight

Use **HCL color space** (not RGB) for gradients. HCL produces perceptually uniform transitions - colors look evenly spaced to the human eye. RGB gradients often have muddy midpoints. The `go-colorful` library provides `BlendHcl()`.

---

## Animation System

### Architecture (from `anim.go`)

The animation system is a sophisticated gradient-cycling spinner with these features:

```go
const (
    fps           = 20              // 50ms per frame
    maxBirthOffset = time.Second    // Staggered entrance
    prerenderedFrames = 10          // Loop length (non-cycling)
    defaultNumCyclingChars = 10     // Width of spinning area
    ellipsisAnimSpeed = 8           // Frames per ellipsis change
)

type Anim struct {
    width            int
    cyclingCharWidth int
    label            *csync.Slice[string]  // Thread-safe
    birthOffsets     []time.Duration       // Per-character entrance delay
    initialFrames    [][]string            // Pre-rendered dots
    cyclingFrames    [][]string            // Pre-rendered scrambled chars
    step             atomic.Int64          // Current frame (thread-safe)
    ellipsisStep     atomic.Int64          // Ellipsis frame counter
    id               string               // Unique animation ID
}
```

### Settings

```go
type Settings struct {
    ID          string       // Unique ID (auto-generated if empty)
    Size        int          // Width in characters (default 10)
    Label       string       // Text label after spinner ("Thinking")
    LabelColor  color.Color  // Label text color
    GradColorA  color.Color  // Gradient start
    GradColorB  color.Color  // Gradient end
    CycleColors bool         // Whether gradient shifts over time
}
```

### How It Works

1. **Pre-rendering:** All frames computed at creation time (cached by settings hash)
2. **Initial chars:** Dots (`.`) with gradient colors, per-character birth offset
3. **Cycling chars:** Random chars from `"0123456789abcdefABCDEF~!@#$£€%^&*()+=_"` with gradient
4. **Staggered entrance:** Each character has a random birth time within 1 second
5. **Ellipsis:** Label followed by animated `"." → ".." → "..." → ""` cycle
6. **Color cycling:** When enabled, gradient shifts by 1 position per frame

```go
// Render chooses between initial dots and cycling chars
func (a *Anim) Render() string {
    for i := range a.width {
        switch {
        case !initialized && time.Since(start) < birthOffsets[i]:
            b.WriteString(a.initialFrames[step][i])  // Dot
        case i < a.cyclingCharWidth:
            b.WriteString(a.cyclingFrames[step][i])   // Scrambled char
        case i == a.cyclingCharWidth:
            b.WriteString(labelGap)                    // " "
        case i > a.cyclingCharWidth:
            b.WriteString(label[i - offset])           // Label char
        }
    }
    // Append animated ellipsis after label
}
```

### Usage in Assistant Messages

```go
// Creating an animation for assistant messages
a.anim = anim.New(anim.Settings{
    ID:          a.ID(),
    Size:        15,
    GradColorA:  sty.Primary,      // Purple
    GradColorB:  sty.Secondary,    // Gold
    LabelColor:  sty.FgBase,       // Light gray
    CycleColors: true,             // Gradient shifts
})

// Dynamic label based on state
if a.message.IsThinking() {
    a.anim.SetLabel("Thinking")
} else if a.message.IsSummaryMessage {
    a.anim.SetLabel("Summarizing")
}

// Start/step via Bubble Tea commands
func (a *Anim) Start() tea.Cmd { return a.Step() }
func (a *Anim) Step() tea.Cmd {
    return tea.Tick(time.Second/time.Duration(fps), func(t time.Time) tea.Msg {
        return StepMsg{ID: a.id}
    })
}
```

### Caching

```go
// Expensive animation calculations are cached globally by settings hash
var animCacheMap = csync.NewMap[string, *animCache]()

type animCache struct {
    initialFrames  [][]string
    cyclingFrames  [][]string
    width, labelWidth int
    label          []string
    ellipsisFrames []string
}

// Cache key is an xxh3 hash of all settings
func settingsHash(opts Settings) string { ... }
```

---

## UX Patterns

### Focus State Management

```go
type uiFocusState uint8
const (
    uiFocusNone uiFocusState = iota
    uiFocusEditor    // Tab to switch
    uiFocusMain      // Tab to switch back
)

// Tab toggles focus between editor and main content
case key.Matches(msg, m.keyMap.Tab):
    if m.focus == uiFocusEditor {
        m.focus = uiFocusMain
        m.textarea.Blur()
        m.chat.Focus()
    } else {
        m.focus = uiFocusEditor
        m.textarea.Focus()
        m.chat.Blur()
    }

// Click-based focus switching
func (m *UI) handleClickFocus(msg tea.MouseClickMsg) tea.Cmd {
    switch {
    case image.Pt(msg.X, msg.Y).In(m.layout.editor):
        m.focus = uiFocusEditor
    case image.Pt(msg.X, msg.Y).In(m.layout.main):
        m.focus = uiFocusMain
    }
}
```

**Pattern:** Every interactive component has paired Focused/Blurred styles:
- User messages: thin `│` border (blurred) vs thick `▌` border (focused)
- Assistant messages: padding-left(2) (blurred) vs thick `▌` border (focused)
- Tool calls: padding-left(2) (blurred) vs thick `▌` border (focused)
- Buttons: bgSubtle background (blurred) vs secondary background (focused)
- Pills: hidden border (blurred) vs rounded border (focused)

### UI States & Transitions

```go
type uiState uint8
const (
    uiOnboarding uiState = iota  // First-time setup
    uiInitialize                  // Project initialization
    uiLanding                     // Home screen (no session)
    uiChat                        // Active chat session
)

// State transitions affect layout, focus, and compact mode
func (m *UI) setState(state uiState, focus uiFocusState) {
    if state == uiLanding {
        m.isCompact = false  // Always full layout for landing
    }
    m.state = state
    m.focus = focus
    m.updateLayoutAndSize()  // Recalculate layout
}
```

### Two-Phase Cancellation

```go
// First ESC: set warning state
cancelBinding.SetHelp("esc", "press again to cancel")
m.isCanceling = true
cmds = append(cmds, tea.Tick(2*time.Second, func(time.Time) tea.Msg {
    return cancelTimerExpiredMsg{}
}))

// Second ESC (within 2 seconds): actually cancel
if m.isCanceling {
    // Perform actual cancellation
}

// Timer expired: reset
case cancelTimerExpiredMsg:
    m.isCanceling = false
```

### Context-Aware Help

```go
func (m *UI) ShortHelp() []key.Binding {
    switch m.state {
    case uiChat:
        if m.isAgentBusy() {
            if m.isCanceling {
                cancelBinding.SetHelp("esc", "press again to cancel")
            } else if queuedPrompts > 0 {
                cancelBinding.SetHelp("esc", "clear queue")
            }
            return []key.Binding{cancelBinding, upDown}
        }
        if m.focus == uiFocusEditor {
            // Show editor shortcuts (send, newline, commands)
            if m.textarea.Value() == "" {
                commands.SetHelp("/ or ctrl+p", "commands")  // Dynamic help text
            }
        } else {
            // Show navigation shortcuts (scroll, expand, copy)
        }
    }
}

// Keyboard enhancement detection changes help text
if msg.SupportsKeyDisambiguation() {
    m.keyMap.Models.SetHelp("ctrl+m", "models")
    m.keyMap.Editor.Newline.SetHelp("shift+enter", "newline")
}
```

### Expandable Content

```go
const maxCollapsedThinkingHeight = 10

// Thinking box expansion
if !a.thinkingExpanded && totalLines > maxCollapsedThinkingHeight {
    lines = lines[totalLines-maxCollapsedThinkingHeight:]  // Show last N lines
    hint := fmt.Sprintf("… (%d lines hidden) [click or space to expand]",
        totalLines-maxCollapsedThinkingHeight)
    lines = append([]string{hint, ""}, lines...)
}

// Tool output expansion (in tools.go)
maxLines := responseContextHeight  // 10
if expanded { maxLines = len(lines) }
if !expanded && len(lines) > maxLines {
    hint := fmt.Sprintf("… (%d more lines) [space to expand]", remaining)
}

// Toggle via space key or mouse click
func (a *AssistantMessageItem) HandleMouseClick(btn, x, y int) bool {
    if y < a.thinkingBoxHeight {
        a.ToggleExpanded()
        return true
    }
}
```

### Text Truncation (uses `charmbracelet/x/ansi`)

```go
import "github.com/charmbracelet/x/ansi"

// ANSI-aware truncation (preserves color codes)
truncated := ansi.Truncate(text, maxWidth, "…")

// String width measurement (ANSI-aware)
width := ansi.StringWidth(text)

// Header working directory truncation
const dirTrimLimit = 4
cwd := fsext.DirTrim(fsext.PrettyPath(cfg.WorkingDir()), dirTrimLimit)
cwd = ansi.Truncate(cwd, max(0, availWidth-lipgloss.Width(metadata)), "…")
```

### Streaming / Thinking Indicators

```
States:
1. Thinking  -> animated gradient spinner + "Thinking" label (cycling colors)
2. Streaming -> content appears, auto-scroll, animation stops when content arrives
3. Complete  -> full content visible, thinking box with duration footer
4. Error     -> ERROR tag + message + details
5. Canceled  -> italic "Canceled" text
```

```go
// isSpinning logic
func (a *AssistantMessageItem) isSpinning() bool {
    isThinking := a.message.IsThinking()
    isFinished := a.message.IsFinished()
    hasContent := strings.TrimSpace(a.message.Content().Text) != ""
    hasToolCalls := len(a.message.ToolCalls()) > 0
    return (isThinking || !isFinished) && !hasContent && !hasToolCalls
}

// Thinking duration footer
if !a.message.IsThinking() || len(a.message.ToolCalls()) > 0 {
    duration := a.message.ThinkingDuration()
    footer = t.ThinkingFooterTitle.Render("Thought for ") +
        t.ThinkingFooterDuration.Render(duration.String())
}
```

### Auto-Scroll / Follow Mode

```go
// Auto-scroll to bottom during streaming
if m.chat.Follow() {
    if cmd := m.chat.ScrollToBottomAndAnimate(); cmd != nil {
        cmds = append(cmds, cmd)
    }
}

// Manual scroll disengages follow mode
// New content re-engages if user is near bottom

// Methods available
m.chat.ScrollToBottomAndAnimate()
m.chat.ScrollToTopAndAnimate()
m.chat.ScrollByAndAnimate(lines)
m.chat.ScrollToSelectedAndAnimate()
m.chat.AtBottom() bool
m.chat.Follow() bool
```

### Timed Status Messages

```go
const DefaultStatusTTL = 5 * time.Second

// Report helpers
util.ReportError(err)    // -> InfoMsg{Type: InfoTypeError}
util.ReportWarn(msg)     // -> InfoMsg{Type: InfoTypeWarn}
util.ReportInfo(msg)     // -> InfoMsg{Type: InfoTypeInfo}

// Auto-clearing
case util.InfoMsg:
    m.status.SetInfoMsg(msg)
    ttl := msg.TTL
    if ttl <= 0 { ttl = DefaultStatusTTL }
    cmds = append(cmds, clearInfoMsgCmd(ttl))
case util.ClearStatusMsg:
    m.status.ClearInfoMsg()
```

### Prompt History Navigation

```go
// Up/Down arrow navigation through previous messages
promptHistory struct {
    messages []string
    index    int
    draft    string   // Preserves current unsent text
}

// Up arrow: navigate to previous messages
// Down arrow: navigate forward or back to draft
// Loaded async on session start/switch
```

### Clipboard Integration

```go
// Dual clipboard: OSC 52 (terminal) + native
func CopyToClipboard(text, successMessage string) tea.Cmd {
    return tea.Sequence(
        tea.SetClipboard(text),       // OSC 52
        func() tea.Msg {
            _ = clipboard.WriteAll(text)  // Native
            return nil
        },
        util.ReportInfo(successMessage),
    )
}
```

### Tool State Machine

```
AwaitingPermission -> Running -> Success | Error | Canceled
                                    |
                                  Expandable output (click/space)
```

Each state has distinct icon (`●` → `✓` or `×`), color (greenDark → green/redDark), and animation.

### Nested Tool Rendering (Agent Tools)

```go
// Agent/sub-agent tools can contain nested tool calls
type NestedToolContainer interface {
    SetNestedTools([]ToolMessageItem)
    NestedTools() []ToolMessageItem
}

// Nested tools are marked as compact (no border, minimal styling)
type Compactable interface {
    SetCompact(bool)
}

// Recursive loading of nested tool calls
func (m *UI) loadNestedToolCalls(items []chat.MessageItem) {
    // For each agent tool, fetch child session messages
    // Extract nested tool items, mark as compact
    // Recursively load any further nesting
}
```

### Mouse Support

```go
// Mouse modes
v.MouseMode = tea.MouseModeCellMotion

// Handled events
tea.MouseClickMsg    // Click focus, item selection, thinking expand
tea.MouseMotionMsg   // Text selection drag, auto-scroll at edges
tea.MouseReleaseMsg  // End selection, auto-copy with delay
tea.MouseWheelMsg    // Scroll chat (5 lines per tick)

// Double-click detection for copy
const doubleClickThreshold = 500 * time.Millisecond
cmds = append(cmds, tea.Tick(doubleClickThreshold, func(t time.Time) tea.Msg {
    if time.Since(m.lastClickTime) >= doubleClickThreshold {
        return copyChatHighlightMsg{}
    }
    return nil
}))
```

### Editor Prompt Modes

```go
// Normal mode: green dots
s.EditorPromptNormalFocused = lipgloss.NewStyle().Foreground(greenDark).SetString("::: ")

// Yolo mode (auto-approve): warning icon + orange dots
s.EditorPromptYoloIconFocused = lipgloss.NewStyle().
    MarginRight(1).Foreground(charmtone.Oyster).Background(charmtone.Citron).Bold(true).SetString(" ! ")
s.EditorPromptYoloDotsFocused = lipgloss.NewStyle().
    MarginRight(1).Foreground(charmtone.Zest).SetString(":::")

// Placeholder changes based on state
if m.isAgentBusy() {
    m.textarea.Placeholder = m.workingPlaceholder
} else {
    m.textarea.Placeholder = m.readyPlaceholder
}
if m.com.App.Permissions.SkipRequests() {
    m.textarea.Placeholder = "Yolo mode!"
}
```

---

## Dialog System

### Overlay Architecture (from `dialog.go`)

```go
// Dialog interface
type Dialog interface {
    ID() string
    HandleMsg(msg tea.Msg) Action
    Draw(scr uv.Screen, area uv.Rectangle) *tea.Cursor
}

// LIFO stack overlay
type Overlay struct {
    dialogs []Dialog
}

// Key operations
overlay.OpenDialog(dialog)       // Push
overlay.CloseFrontDialog()       // Pop
overlay.CloseDialog(id)          // Remove by ID
overlay.BringToFront(id)         // Move to top
overlay.ContainsDialog(id)       // Check existence
overlay.HasDialogs()             // Any open?
overlay.DialogLast()             // Peek front
```

### Dialog Sizing Constants

```go
const (
    defaultDialogMaxWidth = 70
    defaultDialogHeight   = 20
    titleContentHeight    = 1
    inputContentHeight    = 1
)
```

### Positioning Helpers

```go
// Center a dialog in the screen area
func DrawCenter(scr uv.Screen, area uv.Rectangle, view string) {
    width, height := lipgloss.Size(view)
    center := common.CenterRect(area, width, height)
    uv.NewStyledString(view).Draw(scr, center)
}

// Position at bottom-left (for onboarding)
func DrawOnboarding(scr uv.Screen, area uv.Rectangle, view string) {
    bottomLeft := common.BottomLeftRect(area, width, height)
    uv.NewStyledString(view).Draw(scr, bottomLeft)
}
```

### Dialog Types (IDs)

```go
// Known dialog IDs
dialog.CommandsID     // Command palette
dialog.ModelsID       // Model selection
dialog.SessionsID     // Session list
dialog.PermissionsID  // Tool permission prompt
dialog.FilePickerID   // File browser
dialog.QuitID         // Quit confirmation
dialog.APIKeyInputID  // API key entry
dialog.OAuthID        // OAuth flow
dialog.ReasoningID    // Reasoning effort selection
dialog.ArgumentsID    // Custom command arguments
```

### Dialog Actions (Message Types)

```go
// Dialogs return Action messages that the parent handles
type ActionClose struct{}
type ActionCmd struct{ Cmd tea.Cmd }
type ActionSelectSession struct{ Session session.Session }
type ActionSelectModel struct{ ... }
type ActionToggleYoloMode struct{}
type ActionNewSession struct{}
type ActionSummarize struct{ SessionID string }
type ActionToggleHelp struct{}
type ActionToggleCompactMode struct{}
type ActionTogglePills struct{}
type ActionToggleThinking struct{}
type ActionQuit struct{}
type ActionPermissionResponse struct{ ... }
type ActionRunCustomCommand struct{ ... }
type ActionRunMCPPrompt struct{ ... }
// etc.
```

### Dialog Styles

```go
// Dialog chrome
s.Dialog.Title      = base.Padding(0, 1).Foreground(primary)
s.Dialog.TitleError = base.Foreground(red)
s.Dialog.View       = base.Border(lipgloss.RoundedBorder()).BorderForeground(borderFocus)

// Dialog items
s.Dialog.NormalItem   = base.Padding(0, 1).Foreground(fgBase)
s.Dialog.SelectedItem = base.Padding(0, 1).Background(primary).Foreground(fgBase)

// Mode-specific dialog styles (e.g., deleting sessions)
s.Dialog.Sessions.DeletingView = s.Dialog.View.BorderForeground(red)
s.Dialog.Sessions.DeletingItemFocused = s.Dialog.SelectedItem.Background(red).Foreground(Butter)
s.Dialog.Sessions.DeletingTitleGradientFromColor = red
s.Dialog.Sessions.DeletingTitleGradientToColor = primary
```

### Session Details Overlay (Compact Mode)

```go
// In compact mode, ctrl+d toggles a details overlay
const sessionDetailsMaxHeight = 20
s.CompactDetails.View = s.Base.Padding(0, 1, 1, 1).
    Border(lipgloss.RoundedBorder()).BorderForeground(borderFocus)
```

---

## Completions / Autocomplete

### Architecture (from `completions.go`)

```go
const (
    minHeight = 1
    maxHeight = 10
    minWidth  = 10
    maxWidth  = 100
)

type Completions struct {
    width, height int
    open          bool
    query         string
    keyMap        KeyMap
    list          *list.FilterableList  // Filterable, scrollable list
    normalStyle, focusedStyle, matchStyle lipgloss.Style
}
```

### Trigger and Positioning

```go
// Triggered by @ at start of prompt or after whitespace
if msg.String() == "@" && !m.completionsOpen {
    if curIdx == 0 || isWhitespace(curValue[curIdx-1]) {
        m.completionsOpen = true
        m.completionsStartIndex = curIdx
        m.completionsPositionStart = m.completionsPosition()  // x,y
        cmds = append(cmds, m.completions.Open(depth, limit))
    }
}

// Positioned above the cursor
x := m.completionsPositionStart.X
y := m.completionsPositionStart.Y - popupHeight
// Clamp to screen bounds
if x + w > screenW { x = screenW - w }
x = max(0, x)
y = max(0, y + 1)
```

### Items: Files + MCP Resources

```go
// Loaded async in parallel
func (c *Completions) Open(depth, limit int) tea.Cmd {
    return func() tea.Msg {
        var wg sync.WaitGroup
        wg.Go(func() { msg.Files = loadFiles(depth, limit) })
        wg.Go(func() { msg.Resources = loadMCPResources() })
        wg.Wait()
        return CompletionItemsLoadedMsg{...}
    }
}

// Each item has fuzzy filtering and match highlighting
type CompletionItem struct {
    text         string
    value        any  // FileCompletionValue or ResourceCompletionValue
    normalStyle, focusedStyle, matchStyle lipgloss.Style
}
```

### Navigation

```go
// Circular navigation (wraps around)
func (c *Completions) selectPrev() {
    if !c.list.SelectPrev() {
        c.list.WrapToEnd()
    }
}

// Close on: space, cursor before start index, escape
// Select on: enter/tab (close popup), shift+up/down (keep open)

// Dynamic sizing based on visible items
c.width = ordered.Clamp(maxItemWidth + 2, minWidth, maxWidth)
c.height = ordered.Clamp(len(items), minHeight, maxHeight)
```

---

## Keybinding System

### Structure (from `keys.go`)

```go
type KeyMap struct {
    Editor struct {
        AddFile, SendMessage, OpenEditor, Newline key.Binding
        AddImage, PasteImage, MentionFile, Commands key.Binding
        AttachmentDeleteMode, Escape, DeleteAllAttachments key.Binding
        HistoryPrev, HistoryNext key.Binding
    }
    Chat struct {
        NewSession, AddAttachment, Cancel key.Binding
        Tab, Details, TogglePills key.Binding
        PillLeft, PillRight key.Binding
        Down, Up, UpDown key.Binding          // Line scroll + vim keys (j/k)
        DownOneItem, UpOneItem key.Binding     // Item-level scroll (J/K, shift+arrows)
        PageDown, PageUp key.Binding           // f/b or pgdn/pgup
        HalfPageDown, HalfPageUp key.Binding   // d/u
        Home, End key.Binding                  // g/G
        Copy, ClearHighlight, Expand key.Binding
    }
    Initialize struct {
        Yes, No, Enter, Switch key.Binding
    }
    // Global
    Quit     key.Binding  // ctrl+c
    Help     key.Binding  // ctrl+g
    Commands key.Binding  // ctrl+p
    Models   key.Binding  // ctrl+m or ctrl+l
    Suspend  key.Binding  // ctrl+z
    Sessions key.Binding  // ctrl+s
    Tab      key.Binding  // tab
}
```

### Key Reference

```
KEY          CONTEXT     ACTION
───          ───────     ──────
ctrl+c       Global      Quit (opens quit dialog)
ctrl+g       Global      Toggle full help
ctrl+p       Global      Open command palette
ctrl+m/l     Global      Open model selection
ctrl+s       Global      Open sessions
ctrl+z       Global      Suspend (if not busy)
tab          Global      Switch focus (editor ↔ chat)

enter        Editor      Send message
ctrl+j       Editor      New line (also shift+enter)
ctrl+o       Editor      Open external editor
ctrl+f       Editor      Add image file
ctrl+v       Editor      Paste image from clipboard
@            Editor      Trigger file completion
/            Editor      Open commands (when empty)
ctrl+r       Editor      Attachment delete mode
up/down      Editor      Prompt history navigation

esc          Chat        Cancel (two-phase)
ctrl+n       Chat        New session
ctrl+d       Chat        Toggle details (compact mode)
ctrl+t       Chat        Toggle pills (todo/queue)
j/k/↑/↓     Chat        Scroll by line
J/K/S-↑/↓   Chat        Scroll by item
f/b/pgdn/up Chat        Page scroll
d/u          Chat        Half-page scroll
g/G          Chat        Home/End
space        Chat        Expand/collapse item
c/y          Chat        Copy content
```

### Priority: Dialogs > Cancel > State-specific > Global

```go
// Key handling priority in Update()
1. ctrl+c always opens quit dialog
2. Dialog keys if dialog is open
3. Cancel key if agent is busy (two-phase)
4. State-specific keys (uiChat, uiLanding, etc.)
5. Focus-specific keys (uiFocusEditor vs uiFocusMain)
6. Global keys (help, commands, models, sessions)
```

---

## Responsive Design

### Breakpoint System

```go
const (
    compactModeWidthBreakpoint  = 120
    compactModeHeightBreakpoint = 30
)

// Check on every resize and state change
m.isCompact = m.forceCompactMode ||
    m.width < compactModeWidthBreakpoint ||
    m.height < compactModeHeightBreakpoint

// User can also toggle compact mode manually via commands
func (m *UI) toggleCompactMode() { m.forceCompactMode = !m.forceCompactMode }
```

### Progressive Content Reduction

When terminal gets narrow, strip elements in order:
1. Remove sidebar entirely (< 120 wide) → compact header replaces it
2. Switch from full logo to `SmallRender()` (< 30 tall in sidebar)
3. Session details become an overlay instead of sidebar content
4. Reduce section heights dynamically

### Dynamic Height Distribution

```go
func getDynamicHeightLimits(availableHeight int) (maxFiles, maxLSPs, maxMCPs int) {
    const (
        minItemsPerSection   = 2
        defaultMaxFilesShown = 10
        defaultMaxLSPsShown  = 8
        defaultMaxMCPsShown  = 8
        minAvailableHeightLimit = 10
    )

    // Very little space → minimum values
    if availableHeight < minAvailableHeightLimit {
        return 2, 2, 2
    }

    // Equal distribution, then surplus to higher priority
    heightPerSection := availableHeight / 3
    // Priority: files > LSPs > MCPs
}
```

### Width-Aware Rendering

```go
// Status bar: progressively drop elements when too narrow
if lipgloss.Width(left) + lipgloss.Width(right) > barW {
    // Drop address, timer, etc.
}

// Header: truncate working directory
cwd = ansi.Truncate(cwd, max(0, availWidth-lipgloss.Width(metadata)), "…")

// Model info: put provider on second line if too wide
if lipgloss.Width(modelWithProvider) > width {
    // Split to two lines
}

// Attachments: show "N more…" when too many
if i == fits && len(attachments) > i {
    chips = append(chips, fmt.Sprintf("%d more…", len(attachments)-fits))
    break
}

// Completions: clamp popup width
c.width = ordered.Clamp(maxItemWidth + 2, minWidth, maxWidth)
```

### Component-Level Responsiveness

| Component | Full | Compact/Small |
|-----------|------|---------------|
| **Logo** | Full ASCII art with diagonal fields | `SmallRender()` one-liner: "Charm™ Crush ╱╱╱" |
| **Header** | Sidebar handles logo & info | Compact header: logo + diags + %, cwd, ctrl+d hint |
| **Sidebar** | Full sections (files, LSP, MCP) | Hidden; replaced by compact header |
| **Buttons** | With padding | Configurable padding |
| **Thinking** | Max 10 lines collapsed | Same, but truncation hint shows line count |
| **Pills** | Same rendering | Same, area clamped to available height |

---

## Performance Patterns

### Visibility-Based Animation (from `chat.go`)

Only animate items currently visible in the viewport:

```go
func (m *Chat) Animate(msg anim.StepMsg) tea.Cmd {
    startIdx, endIdx := m.list.VisibleItemIndices()
    isVisible := idx >= startIdx && idx <= endIdx

    if !isVisible {
        m.pausedAnimations[msg.ID] = struct{}{}
        return nil  // Don't propagate - pauses animation
    }

    delete(m.pausedAnimations, msg.ID)
    return animatable.Animate(msg)  // Continue animating
}
```

### Prerendered Animation Frames

```go
// All animation frames computed at creation time
a.initialFrames = make([][]string, numFrames)
a.cyclingFrames = make([][]string, numFrames)
// Each frame is a slice of pre-styled strings
// Render() just indexes into precomputed frames - no style calculations

// Global cache prevents re-computation for same settings
var animCacheMap = csync.NewMap[string, *animCache]()
```

### Width-Based Render Caching

```go
// Cache rendered content keyed by width
type cachedMessageItem struct {
    cachedWidth   int
    cachedHeight  int
    cachedContent string
}

func (c *cachedMessageItem) getCachedRender(width int) (string, int, bool) {
    if c.cachedWidth == width && c.cachedContent != "" {
        return c.cachedContent, c.cachedHeight, true
    }
    return "", 0, false
}

// Cache cleared on message update
func (a *AssistantMessageItem) SetMessage(message *message.Message) {
    a.message = message
    a.clearCache()
}
```

### Lazy List Rendering

```go
// List only renders items in the visible viewport
type List struct {
    offsetIdx  int  // First visible item index
    offsetLine int  // Lines scrolled within first item
}

// getItem renders on demand with callbacks
func (l *List) getItem(idx int) renderedItem {
    item := l.items[idx]
    for _, cb := range l.renderCallbacks {
        item = cb(idx, l.selectedIdx, item)
    }
    rendered := item.Render(l.width)
    return renderedItem{content: rendered, height: strings.Count(rendered, "\n") + 1}
}
```

### Efficient String Building

```go
// Use strings.Builder for multi-part rendering
var b strings.Builder
b.WriteString(header)
b.WriteString("\n")
b.WriteString(content)
return b.String()
```

### MaxWidth/MaxHeight for Overflow

```go
// Use lipgloss constraints instead of manual truncation
style := lipgloss.NewStyle().
    MaxWidth(width).
    MaxHeight(height).
    Render(content)
```

### Thread-Safe State for Animations

```go
// Atomic operations for animation state (concurrent access)
step     atomic.Int64
ellipsisStep atomic.Int64
initialized  atomic.Bool

// Thread-safe slices for labels
label         *csync.Slice[string]
ellipsisFrames *csync.Slice[string]
```

### Screen Buffer Rendering

```go
// Ultraviolet screen buffer avoids string concatenation for layout
canvas := uv.NewScreenBuffer(m.width, m.height)
v.Cursor = m.Draw(canvas, canvas.Bounds())

// Components draw to specific rectangles
view := uv.NewStyledString(renderedContent)
view.Draw(scr, rectangle)

// Post-processing: trim trailing spaces
content := canvas.Render()
for i, line := range lines {
    lines[i] = strings.TrimRight(line, " ")
}
```

---

## View & Rendering Pipeline

### View Function (`View()`)

```go
func (m *UI) View() tea.View {
    var v tea.View
    v.AltScreen = true
    if !m.isTransparent {
        v.BackgroundColor = m.com.Styles.Background  // BgBase (Pepper)
    }
    v.MouseMode = tea.MouseModeCellMotion
    v.WindowTitle = "crush " + home.Short(m.com.Config().WorkingDir())

    // Render to screen buffer
    canvas := uv.NewScreenBuffer(m.width, m.height)
    v.Cursor = m.Draw(canvas, canvas.Bounds())

    // Post-process: normalize newlines, trim trailing spaces
    content := strings.ReplaceAll(canvas.Render(), "\r\n", "\n")
    for i, line := range lines {
        lines[i] = strings.TrimRight(line, " ")
    }
    v.Content = content

    // Terminal progress bar (Ghostty, Windows Terminal)
    if m.progressBarEnabled && m.sendProgressBar && m.isAgentBusy() {
        v.ProgressBar = tea.NewProgressBar(tea.ProgressBarIndeterminate, rand.Intn(100))
    }
    return v
}
```

### Draw Function Layering (`Draw()`)

The `Draw()` function renders components in a specific order (back to front):

```go
func (m *UI) Draw(scr uv.Screen, area uv.Rectangle) *tea.Cursor {
    layout := m.generateLayout(area.Dx(), area.Dy())

    // 1. Clear screen
    screen.Clear(scr)

    // 2. Base layer: header OR sidebar (depending on compact mode)
    if m.isCompact {
        m.drawHeader(scr, layout.header)
    } else {
        m.drawSidebar(scr, layout.sidebar)
    }

    // 3. Main content: chat view
    m.chat.Draw(scr, layout.main)

    // 4. Pills (if visible)
    if layout.pills.Dy() > 0 && m.pillsView != "" {
        uv.NewStyledString(m.pillsView).Draw(scr, layout.pills)
    }

    // 5. Editor
    editor := uv.NewStyledString(m.renderEditorView(editorWidth))
    editor.Draw(scr, layout.editor)

    // 6. Details overlay (compact mode)
    if m.isCompact && m.detailsOpen {
        m.drawSessionDetails(scr, layout.sessionDetails)
    }

    // 7. Status bar + help (overlays info messages over help)
    m.status.Draw(scr, layout.status)

    // 8. Completions popup (positioned above cursor)
    if m.completionsOpen && m.completions.HasItems() {
        completionsView.Draw(scr, popupRect)
    }

    // 9. Debug indicator (optional, env var controlled)
    if os.Getenv("CRUSH_UI_DEBUG") == "true" { ... }

    // 10. Dialogs (always on top, full screen bounds)
    if m.dialog.HasDialogs() {
        return m.dialog.Draw(scr, scr.Bounds())
    }

    // 11. Cursor positioning
    if m.focus == uiFocusEditor && m.textarea.Focused() {
        cur := m.textarea.Cursor()
        cur.X++
        cur.Y += m.layout.editor.Min.Y + 1
        return cur
    }
    return nil
}
```

### Terminal Capabilities

```go
type Capabilities struct { ... }

// Detected features that affect rendering
tea.TerminalVersionMsg   // Terminal name (Ghostty detection for progress bar)
tea.KeyboardEnhancementsMsg  // Key disambiguation support
tea.EnvMsg               // Environment variables (WT_SESSION for Windows Terminal)
uv.KittyGraphicsEvent   // Image support
```

### Transparency Support

```go
// User-configurable transparent background
ui.isTransparent = opts.TUI.Transparent != nil && *opts.TUI.Transparent
if !m.isTransparent {
    v.BackgroundColor = m.com.Styles.Background  // Set solid background
}
```

---

## Quick Reference: Style Recipes

### Colored Badge
```go
badge := renderer.NewStyle().
    Foreground(lipgloss.Color("#000")).
    Background(accentColor).
    Bold(true).
    Padding(0, 1).
    Render("LABEL")
```

### Bordered Panel
```go
panel := renderer.NewStyle().
    Border(lipgloss.RoundedBorder()).
    BorderForeground(borderColor).
    Padding(1, 2).
    Width(w).
    Render(content)
```

### Help Chip (key + desc pair)
```go
key  := keyStyle.Render("ctrl+d")
desc := descStyle.Render("exit")
chip := key + " " + desc
```

### Section with Line Fill
```go
label := labelStyle.Render("Section")
line  := lineStyle.Render(strings.Repeat("─", width - lipgloss.Width(label) - 1))
section := label + " " + line
```

### Focus Border (left edge indicator)
```go
// Blurred: thin border
blurred := lipgloss.NewStyle().
    PaddingLeft(1).
    BorderLeft(true).
    BorderForeground(primaryColor).
    BorderStyle(lipgloss.NormalBorder())

// Focused: thick bar
focused := lipgloss.NewStyle().
    PaddingLeft(1).
    BorderLeft(true).
    BorderForeground(accentColor).
    BorderStyle(lipgloss.Border{Left: "▌"})
```

### Two-Part Status Line
```go
ind  := indicatorStyle.Render(" OK ")     // colored badge
msg  := messageStyle.Width(remaining).Render(text)  // fills remaining space
line := ind + msg
```

### Gradient Decoration
```go
diags := styles.ApplyForegroundGrad(t, strings.Repeat("╱", width), colorA, colorB)
```

### Icon with SetString
```go
// Define once in styles
s.Icon = lipgloss.NewStyle().Foreground(green).SetString("✓")
// Use everywhere - no text argument needed
rendered := s.Icon.String()
```

### Two-Part Chip (icon + label)
```go
icon  := iconStyle.String()  // uses SetString
label := labelStyle.Render(filename)
chip  := lipgloss.JoinHorizontal(lipgloss.Left, icon, label)
```

### Progressive Width Reduction
```go
full := badge + sep + user + sep + timer + sep + exitHint
if lipgloss.Width(full) > barW {
    full = badge + sep + user + sep + exitHint  // Drop timer
}
if lipgloss.Width(full) > barW {
    full = badge  // Minimal
}
```

### Diff Line Styles
```go
s.Diff = diffview.Style{
    InsertLine: diffview.LineStyle{
        LineNumber: lipgloss.NewStyle().Foreground("#629657").Background("#2b322a"),
        Symbol:     lipgloss.NewStyle().Foreground("#629657").Background("#323931"),
        Code:       lipgloss.NewStyle().Background("#323931"),
    },
    DeleteLine: diffview.LineStyle{
        LineNumber: lipgloss.NewStyle().Foreground("#a45c59").Background("#312929"),
        Symbol:     lipgloss.NewStyle().Foreground("#a45c59").Background("#383030"),
        Code:       lipgloss.NewStyle().Background("#383030"),
    },
    EqualLine: diffview.LineStyle{
        LineNumber: lipgloss.NewStyle().Foreground(fgMuted).Background(bgBase),
        Code:       lipgloss.NewStyle().Foreground(fgMuted).Background(bgBase),
    },
}
```

---

## Design Principles

1. **Semantic naming** - Colors named by purpose (`Primary`, `Error`) not appearance (`lightPurple`)
2. **Composition over duplication** - Base styles extended for variants (`TagError = TagBase.Background(red)`)
3. **Nested organization** - Related styles grouped (`Dialog.Sessions`, `Tool.Icon`, `Chat.Message`)
4. **Focus pairs everywhere** - Every interactive element has Focused/Blurred styles with clear visual distinction
5. **Progressive degradation** - Gracefully reduce content for smaller terminals (sidebar → header → minimal)
6. **Unicode-first** - Extensive use of box drawing, symbols, block characters for visual richness
7. **HCL gradients** - Perceptually smooth color transitions using `go-colorful`
8. **Visibility-gated work** - Only animate/update what's on screen
9. **Width-aware everything** - Truncate with `ansi.Truncate()`, wrap, or hide based on available space
10. **Consistent spacing** - Padding and margins follow a predictable system
11. **Pre-render expensive operations** - Animation frames, styles with `SetString`, cached renders
12. **Layered rendering** - Screen buffer approach: clear → base → content → overlays → dialogs → cursor
13. **Context-sensitive help** - Help text changes based on state, focus, and terminal capabilities
14. **Two-phase destructive actions** - Cancel requires double-press within timeout
15. **Thread-safe animation state** - Atomic operations and concurrent-safe collections for animation
16. **Dual clipboard** - Both OSC 52 (terminal) and native clipboard for maximum compatibility
17. **Screen buffer rendering** - Ultraviolet `ScreenBuffer` for positioned rendering instead of string concatenation
