package bubblecomplete

import (
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// pathTestCommands constructs a minimal set of test commands tailored to
// exercising the path-completion Update integration without depending on
// the project-wide TestCommands fixture.
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

// TestTabAccept_EqualsFormRegressionGuard locks in the regression where
// the recompute → completion wiring previously dropped the "--path="
// prefix when Tab-accepting a path completion. Drives the full Update
// flow with simulateTyping.
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

	// Tab to accept the first candidate. File completions append a
	// trailing space (bash-style "this token is done").
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyTab})

	want := "find . --path=" + filepath.Join(dir, "report.txt") + " "
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

	// File completions get a trailing space.
	want := "cat " + filepath.Join(dir, "alpha.txt") + " "
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

	// File completions get a trailing space AFTER the closing quote.
	want := `cat "` + filepath.Join(dir, "alpha.txt") + `" `
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
	// expanded $HOME path. File completion appends a trailing space.
	want := "cat ~/report.txt "
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
	// File completion appends a trailing space.
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	wantAfterSecond := "cat " + filepath.Join(dir, "uniquedir", "child.txt") + " "
	if got := m.input.Value(); got != wantAfterSecond {
		t.Errorf("after second Tab: got %q, want %q", got, wantAfterSecond)
	}
}

// TestRender_OverlayInactiveDuringFileCycling locks in the natural
// interaction between the bash-style trailing-space-after-files rule and
// the per-cycle pathState refresh: cycled FILE candidates have a trailing
// space, which makes activeFileArgument inactive, which skips the
// overlay. Users cycling between file candidates see no validity colour
// — but the cycled values are all real files by construction (they came
// from generateCandidates' kind filter), so the missing colour carries
// no information.
func TestRender_OverlayInactiveDuringFileCycling(t *testing.T) {
	dir := t.TempDir()
	// Two files with the same prefix → Tab enters cycling state.
	mustWriteFile(t, filepath.Join(dir, "foo.txt"))
	mustWriteFile(t, filepath.Join(dir, "fox.txt"))

	// Width must comfortably exceed the temp-dir path or the overflow
	// guard (not the trailing-space-makes-inactive interaction) would be
	// what produces the no-overlay state — defeating the test.
	m, err := New(pathTestCommands(), 500, WithFilesystemCompletions(true))
	if err != nil {
		t.Fatal(err)
	}
	m = simulateTyping(t, m, "cat "+filepath.Join(dir, "fo"))

	if m.renderedInput() == m.input.View() {
		t.Fatal("setup: expected overlay to be applied before Tab")
	}

	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	if m.completionHolder == "" {
		t.Fatal("setup: expected Tab to enter cycling state")
	}
	// Cycled file value ends in trailing space → pathState refresh sees
	// HasSuffix(input, " ") and returns inactive. Overlay skipped.
	if m.pathState.active {
		t.Errorf("expected pathState inactive after cycling to a file (trailing space); active=true")
	}
	if m.renderedInput() != m.input.View() {
		t.Errorf("overlay should be skipped for cycled file (trailing space):\n  got %q\n want %q",
			m.renderedInput(), m.input.View())
	}
}

// TestRender_OverlayAppliesDuringDirCycling is the dir-specific counterpart
// to the file-cycling test: cycled directory candidates end with "/" (no
// trailing space), so pathState refresh keeps the overlay active with the
// correct cycled-value offsets. The overlay therefore reflects the
// classification of the cycled directory candidate (partial for FileArg
// drill-down, valid for DirArg target).
func TestRender_OverlayAppliesDuringDirCycling(t *testing.T) {
	dir := t.TempDir()
	mustMkdir(t, filepath.Join(dir, "doc1"))
	mustMkdir(t, filepath.Join(dir, "doc2"))

	m, err := New(pathTestCommands(), 500, WithFilesystemCompletions(true))
	if err != nil {
		t.Fatal(err)
	}
	m = simulateTyping(t, m, "cd "+filepath.Join(dir, "doc"))
	if !m.pathState.active {
		t.Fatal("setup: expected active pathState for partial dir match")
	}

	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	if m.completionHolder == "" {
		t.Fatal("setup: expected Tab to enter cycling state (multiple dir candidates)")
	}
	// pathState should be refreshed to reflect the cycled value
	// "cd <dir>/doc1/" (or doc2). base is empty (trailing /), DirArgument
	// classifies as pathValid.
	if !m.pathState.active {
		t.Errorf("expected pathState ACTIVE during dir cycling; active=false")
	}
	if m.pathState.validity != pathValid {
		t.Errorf("cycled dir under DirArgument should classify pathValid; got %d", m.pathState.validity)
	}
	if m.renderedInput() == m.input.View() {
		t.Errorf("overlay should apply during dir cycling, but renderedInput matches raw input.View")
	}

	// Forward-Tab to the next dir candidate. Validity stays pathValid (both
	// are real dirs under DirArgument) — but the cycled value differs, so
	// the rendered output must differ between the two cycle states.
	firstCycleRender := m.renderedInput()
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	if m.renderedInput() == firstCycleRender {
		t.Errorf("cycling to second dir candidate should change the rendered overlay coverage")
	}
}

// TestRender_OverlayRefreshesOnCycleRevert exercises the wrap-to-original
// path: with two candidates, three forward Tabs cycle index 0 → 1 → -1
// (revert to completionHolder). After revert, pathState must reflect the
// ORIGINAL input — not the last-cycled candidate — because the
// input-changed branch later in Update won't fire (lastInput == restored
// value).
func TestRender_OverlayRefreshesOnCycleRevert(t *testing.T) {
	dir := t.TempDir()
	mustMkdir(t, filepath.Join(dir, "doc1"))
	mustMkdir(t, filepath.Join(dir, "doc2"))

	m, err := New(pathTestCommands(), 500, WithFilesystemCompletions(true))
	if err != nil {
		t.Fatal(err)
	}
	original := "cd " + filepath.Join(dir, "doc")
	m = simulateTyping(t, m, original)
	if !m.pathState.active {
		t.Fatal("setup: expected active pathState for partial dir match")
	}
	// Capture the *original* partial-state offsets so we can compare them
	// after the revert. The original token is "doc" (3 chars), base "doc".
	origValueStart := m.pathState.valueStart
	origValueEnd := m.pathState.valueEnd
	origBase := m.pathState.base
	if m.pathState.validity != pathPartial {
		t.Fatalf("setup: expected pathPartial for original, got %d", m.pathState.validity)
	}

	// Three Tabs: 0, 1, revert.
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyTab}) // index 0
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyTab}) // index 1
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyTab}) // wraps to -1, revert to original

	if m.input.Value() != original {
		t.Fatalf("after wrap-to-revert: input %q, want original %q", m.input.Value(), original)
	}
	if m.completionHolder != "" {
		t.Errorf("after revert: completionHolder should be cleared, got %q", m.completionHolder)
	}
	if !m.pathState.active {
		t.Fatal("pathState should be active after revert (matches original partial state)")
	}
	if m.pathState.validity != pathPartial {
		t.Errorf("after revert: validity should reflect original partial state, got %d", m.pathState.validity)
	}
	if m.pathState.valueStart != origValueStart || m.pathState.valueEnd != origValueEnd {
		t.Errorf("after revert: offsets %d..%d, want original %d..%d (offsets should track restored value)",
			m.pathState.valueStart, m.pathState.valueEnd, origValueStart, origValueEnd)
	}
	if m.pathState.base != origBase {
		t.Errorf("after revert: base %q, want original %q", m.pathState.base, origBase)
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

// TestHistoryNav_RefreshesPathState verifies that pulling a historic value
// via Up arrow triggers the input-changed branch and refreshes pathState
// against the historic path. Without this, the overlay would still
// reflect whatever was on screen before Up — typically nothing for a
// fresh model.
func TestHistoryNav_RefreshesPathState(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, "foo.txt"))
	historic := "cat " + filepath.Join(dir, "foo.txt")

	m, err := New(pathTestCommands(), 500, WithFilesystemCompletions(true))
	if err != nil {
		t.Fatal(err)
	}
	// Seed history manually; the integration here is the Up → SetValue →
	// input-changed branch chain, not the submit-saves-history path.
	m.History = []string{historic}

	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyUp})

	if m.input.Value() != historic {
		t.Fatalf("after Up: input %q, want historic %q", m.input.Value(), historic)
	}
	if !m.pathState.active {
		t.Errorf("pathState should be active after history nav loaded a path")
	}
	if m.pathState.validity != pathValid {
		t.Errorf("historic path resolves to a real file → expected pathValid, got %d", m.pathState.validity)
	}
}

// TestPsFlagWithFileArgument exercises the PowerShell-style flag form
// (`-Path value` and `-Path=value`) carrying a FileArgument. Both code
// paths in activeFileArgument (Case 1 equals-form, Case 2 space-separated)
// branch on LongFlag OR PsFlag — these tests confirm the PsFlag branch
// works end-to-end, not just the LongFlag side that the rest of the suite
// already exercises.
func TestPsFlagWithFileArgument(t *testing.T) {
	psCmd := &Command{
		Command: "ps",
		PositionalArguments: []*PositionalArgument{
			{Name: "Cmd", Type: StringArgument, Required: true},
		},
		Flags: []*Flag{
			{PsFlag: "-Path", Type: FileArgument},
		},
	}
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, "alpha.txt"))

	t.Run("equals-form", func(t *testing.T) {
		m, err := New([]*Command{psCmd}, 500, WithFilesystemCompletions(true))
		if err != nil {
			t.Fatal(err)
		}
		m = simulateTyping(t, m, "ps run -Path="+filepath.Join(dir, "alph"))
		if !m.pathState.active {
			t.Fatal("expected pathState active for -Path=<partial>")
		}
		if m.pathState.kind != FileArgument {
			t.Errorf("kind = %v, want FileArgument", m.pathState.kind)
		}
		if !pathCandidateNames(m.pathState.candidates).has("alpha.txt") {
			t.Errorf("expected alpha.txt candidate, got %v", pathCandidateNames(m.pathState.candidates))
		}
	})

	t.Run("space-separated", func(t *testing.T) {
		m, err := New([]*Command{psCmd}, 500, WithFilesystemCompletions(true))
		if err != nil {
			t.Fatal(err)
		}
		m = simulateTyping(t, m, "ps run -Path "+filepath.Join(dir, "alph"))
		if !m.pathState.active {
			t.Fatal("expected pathState active for -Path <partial>")
		}
		if m.pathState.kind != FileArgument {
			t.Errorf("kind = %v, want FileArgument", m.pathState.kind)
		}
		if !pathCandidateNames(m.pathState.candidates).has("alpha.txt") {
			t.Errorf("expected alpha.txt candidate, got %v", pathCandidateNames(m.pathState.candidates))
		}
	})
}

// TestKeyRight_DrillDownAfterCyclingDir is the explicit accept-and-recompute
// flow for a path candidate. With two matching dirs, Tab enters cycling,
// Right accepts the cycled candidate AND triggers a recompute, then the
// next Tab cycles among the accepted dir's children. The Tab → Right →
// Tab pattern is the natural drill-down for a host that maps the accept
// key to Right.
func TestKeyRight_DrillDownAfterCyclingDir(t *testing.T) {
	dir := t.TempDir()
	mustMkdir(t, filepath.Join(dir, "doc1"))
	mustWriteFile(t, filepath.Join(dir, "doc1", "child.txt"))
	mustMkdir(t, filepath.Join(dir, "doc2"))

	m, err := New(pathTestCommands(), 500, WithFilesystemCompletions(true))
	if err != nil {
		t.Fatal(err)
	}
	m = simulateTyping(t, m, "cat "+filepath.Join(dir, "doc"))

	// Tab enters cycling on the first dir candidate.
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	if m.completionHolder == "" {
		t.Fatal("setup: expected Tab to enter cycling state (2 dir candidates)")
	}
	cycledValue := m.input.Value()

	// Right accepts the cycled candidate and exits cycling — completionHolder
	// clears, input-changed branch fires, recompute populates child candidates.
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	if m.completionHolder != "" {
		t.Errorf("Right should have cleared completionHolder, got %q", m.completionHolder)
	}
	if m.input.Value() != cycledValue {
		t.Errorf("Right should not change input value: was %q, now %q", cycledValue, m.input.Value())
	}
	if !pathCandidateNames(m.pathState.candidates).has("child.txt") {
		t.Fatalf("expected child.txt in candidates after Right-accept of doc1/; got %v",
			pathCandidateNames(m.pathState.candidates))
	}

	// Next Tab drills into the single child (auto-accept on single match).
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	want := "cat " + filepath.Join(dir, "doc1", "child.txt") + " "
	if got := m.input.Value(); got != want {
		t.Errorf("after drill-down Tab: %q, want %q", got, want)
	}
}

// TestRuntimeToggle_FilesystemCompletions verifies the documented behaviour
// for direct field mutation: changes to FilesystemCompletions take effect
// on the NEXT input change, not in place. The field is public for parity
// with other Model knobs, but pathState is not refreshed eagerly when a
// host flips the flag — the host must produce an input event (typing,
// backspace, etc.) to trigger a recompute.
func TestRuntimeToggle_FilesystemCompletions(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, "foo.txt"))

	m := newPathTestModel(t, false)
	m = simulateTyping(t, m, "cat "+filepath.Join(dir, "fo"))
	if m.pathState.active {
		t.Fatal("setup: feature disabled, pathState should be inactive")
	}

	// Toggle on directly — no input event yet, so pathState stays stale.
	m.FilesystemCompletions = true
	if m.pathState.active {
		t.Errorf("toggling FilesystemCompletions should NOT refresh pathState in place")
	}

	// Type one more character → input-changed branch fires → recompute
	// runs with the toggle now on → pathState becomes active.
	m = simulateTyping(t, m, "o")
	if !m.pathState.active {
		t.Errorf("after next input event, pathState should be active (feature now on)")
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
