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
