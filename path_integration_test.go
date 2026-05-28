package bubblecomplete

import (
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// pathTestCommands constructs a minimal set of test commands tailored to
// exercising the Phase 3 Update integration without depending on the
// project-wide TestCommands fixture.
func pathTestCommands() []*Command {
	return []*Command{
		{
			Command: "cat",
			PositionalArguments: []*PositionalArgument{
				{Name: "File", Type: FileArgument, Required: true},
			},
		},
		{
			Command: "cd",
			PositionalArguments: []*PositionalArgument{
				{Name: "Dir", Type: DirArgument, Required: true},
			},
		},
		{
			Command: "find",
			PositionalArguments: []*PositionalArgument{
				{Name: "Dir", Type: DirArgument, Required: true},
			},
			Flags: []*Flag{
				{LongFlag: "--path", Type: FileArgument},
			},
		},
		{
			Command: "echo",
			PositionalArguments: []*PositionalArgument{
				{Name: "Msg", Type: StringArgument, Required: true},
			},
		},
	}
}

func newPathTestModel(t *testing.T, enable bool) Model {
	t.Helper()
	opts := []Option{}
	if enable {
		opts = append(opts, WithFilesystemCompletions(true))
	}
	m, err := New(pathTestCommands(), 100, opts...)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func TestRecomputePathState_DisabledIsInactive(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, "foo.txt"))
	m := newPathTestModel(t, false)
	m = simulateTyping(t, m, "cat "+filepath.Join(dir, "fo"))
	if m.pathState.active {
		t.Errorf("pathState.active = true when feature disabled")
	}
}

func TestRecomputePathState_EnabledPopulatesState(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, "foo.txt"))
	m := newPathTestModel(t, true)
	m = simulateTyping(t, m, "cat "+filepath.Join(dir, "fo"))

	if !m.pathState.active {
		t.Fatal("pathState.active = false, want true")
	}
	if m.pathState.kind != FileArgument {
		t.Errorf("kind = %v, want FileArgument", m.pathState.kind)
	}
	if m.pathState.validity != pathPartial {
		t.Errorf("validity = %d, want pathPartial", m.pathState.validity)
	}
	if !pathCandidateNames(m.pathState.candidates).has("foo.txt") {
		t.Errorf("candidates missing foo.txt: %v", pathCandidateNames(m.pathState.candidates))
	}
}

func TestRecomputePathState_ExactFileIsValid(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, "foo.txt"))
	m := newPathTestModel(t, true)
	m = simulateTyping(t, m, "cat "+filepath.Join(dir, "foo.txt"))

	if !m.pathState.active {
		t.Fatal("expected active")
	}
	if m.pathState.validity != pathValid {
		t.Errorf("validity = %d, want pathValid", m.pathState.validity)
	}
}

func TestRecomputePathState_StringArgIsInactive(t *testing.T) {
	m := newPathTestModel(t, true)
	m = simulateTyping(t, m, "echo hello")
	if m.pathState.active {
		t.Errorf("string arg should not trigger active path state")
	}
}

func TestGetCompletions_WholesaleReplaceWhenActive(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, "alpha.txt"))
	mustWriteFile(t, filepath.Join(dir, "beta.txt"))
	m := newPathTestModel(t, true)
	m = simulateTyping(t, m, "cat "+filepath.Join(dir, "a"))

	if len(m.completions) == 0 {
		t.Fatal("expected at least one completion")
	}
	for _, c := range m.completions {
		if _, ok := c.(pathCompletion); !ok {
			t.Errorf("expected pathCompletion, got %T (%q)", c, c.getName())
		}
	}
}

func TestEmptyInputClearsPathStateWithoutValidationErr(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, "foo.txt"))
	m := newPathTestModel(t, true)
	full := filepath.Join(dir, "foo.txt")
	m = simulateTyping(t, m, "cat "+full)

	if !m.pathState.active || m.pathState.validity != pathValid {
		t.Fatal("setup: expected active pathValid state")
	}
	if m.validationErr != nil {
		t.Fatalf("setup: expected nil validationErr (valid file), got %v", m.validationErr)
	}

	// Backspace through the entire input. Each backspace runs the input-
	// changed branch; the final transition to empty should reset pathState
	// even though there's no validationErr to clear.
	for range len("cat " + full) {
		m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})
	}

	if m.input.Value() != "" {
		t.Fatalf("setup: expected empty value after backspace loop, got %q", m.input.Value())
	}
	if m.pathState.active {
		t.Errorf("pathState should be cleared after input emptied")
	}
}

func TestResetModelClearsPathState(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, "foo.txt"))
	m := newPathTestModel(t, true)
	m = simulateTyping(t, m, "cat "+filepath.Join(dir, "foo.txt"))
	if !m.pathState.active {
		t.Fatal("setup: expected active state")
	}
	m = m.resetModel()
	if m.pathState.active {
		t.Errorf("resetModel did not clear pathState")
	}
}

// TestTabAccept_EqualsFormRegressionGuard locks in the v4 regression: the
// Phase 3 wiring must not drop the "--path=" prefix when Tab-accepting a
// path completion. This exercises the full Update flow with simulateTyping.
func TestTabAccept_EqualsFormRegressionGuard(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, "report.txt"))

	m := newPathTestModel(t, true)
	// Type "find . --path=<dir>/rep"
	m = simulateTyping(t, m, "find . --path="+filepath.Join(dir, "rep"))

	if !m.pathState.active {
		t.Fatal("expected active path state")
	}
	if !pathCandidateNames(m.pathState.candidates).has("report.txt") {
		t.Fatalf("expected report.txt candidate: %v", pathCandidateNames(m.pathState.candidates))
	}

	// Tab to accept the first candidate.
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyTab})

	want := "find . --path=" + filepath.Join(dir, "report.txt")
	if got := m.input.Value(); got != want {
		t.Errorf("after Tab: %q, want %q", got, want)
	}
}

func TestTabAccept_PositionalFile(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, "alpha.txt"))

	m := newPathTestModel(t, true)
	m = simulateTyping(t, m, "cat "+filepath.Join(dir, "alph"))

	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyTab})

	want := "cat " + filepath.Join(dir, "alpha.txt")
	if got := m.input.Value(); got != want {
		t.Errorf("Tab insertion = %q, want %q", got, want)
	}
}

func TestTabAccept_PositionalDirAddsTrailingSlash(t *testing.T) {
	dir := t.TempDir()
	mustMkdir(t, filepath.Join(dir, "subdir"))

	m := newPathTestModel(t, true)
	m = simulateTyping(t, m, "cd "+filepath.Join(dir, "sub"))

	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyTab})

	want := "cd " + filepath.Join(dir, "subdir") + "/"
	if got := m.input.Value(); got != want {
		t.Errorf("Tab insertion = %q, want %q", got, want)
	}
}

func TestTabAccept_QuotedPositional(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, "alpha.txt"))

	m := newPathTestModel(t, true)
	m = simulateTyping(t, m, `cat "`+filepath.Join(dir, "alph"))

	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyTab})

	want := `cat "` + filepath.Join(dir, "alpha.txt") + `"`
	if got := m.input.Value(); got != want {
		t.Errorf("Tab insertion = %q, want %q", got, want)
	}
}

func TestTabAccept_TildeExpansion(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	mustWriteFile(t, filepath.Join(home, "report.txt"))

	m := newPathTestModel(t, true)
	m = simulateTyping(t, m, "cat ~/rep")

	if !pathCandidateNames(m.pathState.candidates).has("report.txt") {
		t.Fatalf("expected report.txt under ~/: %v", pathCandidateNames(m.pathState.candidates))
	}

	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyTab})

	// userPrefix is preserved verbatim, so the result keeps "~/", not the
	// expanded $HOME path.
	want := "cat ~/report.txt"
	if got := m.input.Value(); got != want {
		t.Errorf("Tab insertion = %q, want %q", got, want)
	}
}

// renderWithFeature is a small helper that builds a model with the feature
// toggled to `on`, types `input`, and returns the rendered output.
func renderWithFeature(t *testing.T, on bool, input string) string {
	t.Helper()
	m := newPathTestModel(t, on)
	m = simulateTyping(t, m, input)
	return m.Render()
}

// TestRender_OverlayChangesOutputByValidity asserts that the rendered output
// differs between valid/partial/invalid validity classes when the feature
// is enabled — i.e. the overlay is applied and the chosen style differs by
// classification. The exact ANSI bytes are not asserted (lipgloss merges
// styles when overlaying via StyleRanges, so the precise output isn't
// trivially reconstructable). Asserting pairwise inequality is the
// strongest observation we can make without coupling to internal ANSI
// codes.
func TestRender_OverlayChangesOutputByValidity(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, "foo.txt"))

	valid := renderWithFeature(t, true, "cat "+filepath.Join(dir, "foo.txt"))
	partial := renderWithFeature(t, true, "cat "+filepath.Join(dir, "fo"))
	invalid := renderWithFeature(t, true, "cat "+filepath.Join(dir, "zzz_nope"))

	if valid == partial {
		t.Errorf("valid and partial renders should differ (overlay style not changing)")
	}
	if valid == invalid {
		t.Errorf("valid and invalid renders should differ")
	}
	if partial == invalid {
		t.Errorf("partial and invalid renders should differ")
	}
}

// TestRender_OverlayPresentWhenFeatureEnabled asserts that enabling the
// feature produces a different rendered output than leaving it disabled,
// holding all other state equal. The byte-level difference proves that
// styleInputPathRange wrote something to the output.
func TestRender_OverlayPresentWhenFeatureEnabled(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, "foo.txt"))
	full := filepath.Join(dir, "foo.txt")

	on := renderWithFeature(t, true, "cat "+full)
	off := renderWithFeature(t, false, "cat "+full)
	if on == off {
		t.Errorf("feature-on render should differ from feature-off render")
	}
}

// TestRender_OverlaySkippedWhenOverflow asserts that an input wider than
// m.width bypasses the overlay rather than painting garbage. The test
// compares renderedInput against the raw textinput.View directly — the
// only thing renderedInput should add when active is the StyleRanges
// overlay, so when the overflow guard fires, those outputs must match.
func TestRender_OverlaySkippedWhenOverflow(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, "foo.txt"))
	input := "cat " + filepath.Join(dir, "foo.txt")

	m, err := New(pathTestCommands(), 8, WithFilesystemCompletions(true))
	if err != nil {
		t.Fatal(err)
	}
	m = simulateTyping(t, m, input)
	if !m.pathState.active {
		t.Fatal("setup: expected pathState.active=true")
	}
	if m.renderedInput() != m.input.View() {
		t.Errorf("overflow case should bypass overlay:\n  got %q\n want %q",
			m.renderedInput(), m.input.View())
	}
}

// TestRender_EqualsFormOverlayDiffers ensures the equals-form rendering
// changes when the feature is enabled — the value portion gets the
// overlay even though tokenPrefix ("--path=") is in the same token.
func TestRender_EqualsFormOverlayDiffers(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, "report.txt"))
	input := "find . --path=" + filepath.Join(dir, "report.txt")
	if renderWithFeature(t, true, input) == renderWithFeature(t, false, input) {
		t.Errorf("equals-form render should differ between feature on/off")
	}
}

// TestSuppression_DoesNotHideErrorFromOtherArg verifies that when one
// path argument is invalid (PathNotFound from a committed token) and a
// LATER positional is a partial path, the whole-input red is NOT
// suppressed — the validation error belongs to a different token.
func TestSuppression_DoesNotHideErrorFromOtherArg(t *testing.T) {
	cmd := &Command{
		Command: "twopath",
		PositionalArguments: []*PositionalArgument{
			{Name: "first", Type: FileArgument, Required: true},
			{Name: "second", Type: DirArgument, Required: true},
		},
	}
	dir := t.TempDir()
	mustMkdir(t, filepath.Join(dir, "subdir"))

	m, err := New([]*Command{cmd}, 200, WithFilesystemCompletions(true))
	if err != nil {
		t.Fatal(err)
	}

	// First path: definitely-missing. Second path: prefix of "subdir/".
	input := "twopath " + filepath.Join(dir, "missing") + " " + filepath.Join(dir, "sub")
	m = simulateTyping(t, m, input)

	if m.validationErr == nil {
		t.Fatal("setup: expected validationErr from first path")
	}
	if !m.pathState.active || m.pathState.validity != pathPartial {
		t.Fatalf("setup: expected active+partial for second path; got active=%v validity=%d",
			m.pathState.active, m.pathState.validity)
	}
	if m.isPartialPathMidType() {
		t.Errorf("suppression must NOT fire: validation error belongs to a different argument")
	}
}

// TestSuppression_FiresWhenErrorMatchesActiveArg confirms the positive case
// — single path argument, partial typing, error matches → suppression fires.
func TestSuppression_FiresWhenErrorMatchesActiveArg(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, "foo.txt"))
	m := newPathTestModel(t, true)
	m = simulateTyping(t, m, "cat "+filepath.Join(dir, "fo"))

	if !m.isPartialPathMidType() {
		t.Errorf("expected suppression to fire for single-arg partial path")
	}
}

// TestTabAutoAcceptDrillsIntoUniqueDir locks in the bash-like single-match
// auto-accept behaviour: typing a prefix that uniquely matches a directory
// and pressing Tab should accept it (input now ends in "uniquedir/") AND
// exit cycling state so the next Tab lists the directory's children. Two
// Tabs in a row drill one level deep.
//
// Uses cat (FileArgument) because FileArgument candidates include files
// inside the drilled-into dir — exercising the auto-accept-then-recompute
// path end-to-end. DirArgument would filter the file child out (dirs only)
// which would make the drill-down land on empty candidates.
func TestTabAutoAcceptDrillsIntoUniqueDir(t *testing.T) {
	dir := t.TempDir()
	mustMkdir(t, filepath.Join(dir, "uniquedir"))
	mustWriteFile(t, filepath.Join(dir, "uniquedir", "child.txt"))

	m, err := New(pathTestCommands(), 500, WithFilesystemCompletions(true))
	if err != nil {
		t.Fatal(err)
	}
	m = simulateTyping(t, m, "cat "+filepath.Join(dir, "uniq"))

	// First Tab: single match → auto-accept, no cycling state, recompute
	// runs the same Update tick.
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	wantAfterFirst := "cat " + filepath.Join(dir, "uniquedir") + "/"
	if got := m.input.Value(); got != wantAfterFirst {
		t.Fatalf("after first Tab: got %q, want %q", got, wantAfterFirst)
	}
	if m.completionHolder != "" || m.completionIndex != -1 {
		t.Errorf("single-match Tab should clear cycling state; holder=%q index=%d",
			m.completionHolder, m.completionIndex)
	}
	if !pathCandidateNames(m.pathState.candidates).has("child.txt") {
		t.Fatalf("expected child.txt candidate after auto-accept drill-down: %v",
			pathCandidateNames(m.pathState.candidates))
	}

	// Second Tab: single child auto-accepts again, drilling to the file.
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	wantAfterSecond := "cat " + filepath.Join(dir, "uniquedir", "child.txt")
	if got := m.input.Value(); got != wantAfterSecond {
		t.Errorf("after second Tab: got %q, want %q", got, wantAfterSecond)
	}
}

// TestRender_OverlayBypassedDuringCompletionCycling locks in that the
// overlay is skipped while the user is cycling Tab completions. Without
// the bypass, the frozen pathState offsets would mis-style the cycled
// preview value.
func TestRender_OverlayBypassedDuringCompletionCycling(t *testing.T) {
	dir := t.TempDir()
	// Two files with the same prefix → Tab enters cycling state. A single
	// match would auto-accept and skip cycling entirely.
	mustWriteFile(t, filepath.Join(dir, "foo.txt"))
	mustWriteFile(t, filepath.Join(dir, "fox.txt"))

	// Width must comfortably exceed the temp-dir path or the overflow
	// guard, not the cycling bypass, would be what produces the
	// no-overlay state — defeating the test.
	m, err := New(pathTestCommands(), 500, WithFilesystemCompletions(true))
	if err != nil {
		t.Fatal(err)
	}
	m = simulateTyping(t, m, "cat "+filepath.Join(dir, "fo"))

	beforeTab := m.renderedInput()
	rawBeforeTab := m.input.View()
	if beforeTab == rawBeforeTab {
		t.Fatal("setup: expected overlay to be applied before Tab (width may be too small)")
	}

	// Tab to begin cycling — input.Value updates to the candidate, but
	// pathState stays frozen. renderedInput should bypass the overlay.
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	if m.completionHolder == "" {
		t.Fatal("setup: expected Tab to enter cycling state")
	}
	if m.renderedInput() != m.input.View() {
		t.Errorf("overlay should be bypassed during cycling:\n  got %q\n want %q",
			m.renderedInput(), m.input.View())
	}
}

func TestRender_WholeInputNotInvalidForPartialPathNotFound(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, "foo.txt"))
	m := newPathTestModel(t, true)
	partial := filepath.Join(dir, "fo")
	m = simulateTyping(t, m, "cat "+partial)

	// Setup invariant: partial path resolves to PathNotFound at submit time,
	// but classifier reports pathPartial — suppression should fire.
	if m.pathState.validity != pathPartial {
		t.Fatalf("setup: expected pathPartial, got %d", m.pathState.validity)
	}
	if !m.isPartialPathMidType() {
		t.Fatalf("setup: expected isPartialPathMidType=true (validationErr=%v)", m.validationErr)
	}

	// The textinput's Focused.Text style should be the Valid style, not
	// Invalid, because the suppression rule fired.
	got := m.input.Styles().Focused.Text.Render("sample")
	want := m.Styles().Input.Valid.Render("sample")
	if got != want {
		t.Errorf("expected suppression to keep Valid style; got %q want %q", got, want)
	}
}

// pathCandidateNames is a small helper for set-style assertions.
type pathCandidateNames []pathCompletion

func (p pathCandidateNames) has(name string) bool {
	for _, c := range p {
		if c.displayName == name {
			return true
		}
	}
	return false
}

func (p pathCandidateNames) String() string {
	names := make([]string, len(p))
	for i, c := range p {
		names[i] = c.displayName
	}
	return "[" + filepath.Join(names...) + "]"
}
