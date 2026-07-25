package commands

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/StevenBock/docdiff/internal/config"
	"github.com/StevenBock/docdiff/internal/git"
	"github.com/StevenBock/docdiff/internal/language"
)

func setupTestProject(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()

	cmds := [][]string{
		{"git", "init"},
		{"git", "config", "user.email", "test@example.com"},
		{"git", "config", "user.name", "Test User"},
	}

	for _, args := range cmds {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = tmpDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("Failed to run %v: %v", args, err)
		}
	}

	os.MkdirAll(filepath.Join(tmpDir, "docs"), 0755)
	os.MkdirAll(filepath.Join(tmpDir, "src"), 0755)

	os.WriteFile(filepath.Join(tmpDir, "docs", "API.md"), []byte("# API Docs\n"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "docs", "GUIDE.md"), []byte("# Guide\n"), 0644)

	os.WriteFile(filepath.Join(tmpDir, "src", "handler.go"), []byte(`package main

// @doc docs/API.md
func Handler() {}
`), 0644)

	os.WriteFile(filepath.Join(tmpDir, "src", "util.go"), []byte(`package main

func Util() {}
`), 0644)

	cmd := exec.Command("git", "add", ".")
	cmd.Dir = tmpDir
	cmd.Run()
	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = tmpDir
	cmd.Run()

	return tmpDir
}

func runDocdiff(t *testing.T, dir string, args ...string) (string, string, error) {
	t.Helper()

	allArgs := append([]string{"--dir", dir}, args...)
	rootCmd.SetArgs(allArgs)

	var stdout, stderr bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stderr)

	err := rootCmd.Execute()
	return stdout.String(), stderr.String(), err
}

func initTestEnv(t *testing.T, dir string) {
	t.Helper()
	rootDir = dir
	var err error
	cfg, err = config.Load(dir)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}
	registry = language.DefaultRegistry()
}

// commitAll stages everything and commits it in dir.
func commitAll(t *testing.T, dir, message string) {
	t.Helper()
	for _, args := range [][]string{
		{"git", "add", "."},
		{"git", "commit", "-m", message},
	} {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		if err := cmd.Run(); err != nil {
			t.Fatalf("Failed to run %v: %v", args, err)
		}
	}
}

func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to run git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

func TestReport_Integration(t *testing.T) {
	dir := setupTestProject(t)
	initTestEnv(t, dir)

	reportStale = false
	reportOrphaned = false
	reportJSON = false
	reportSARIF = false
	reportCI = false

	var stdout bytes.Buffer
	reportCmd.SetOut(&stdout)

	if err := reportCmd.RunE(reportCmd, nil); err != nil {
		t.Fatalf("report failed: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "Documentation Coverage Report") {
		t.Error("Report should contain title")
	}
	if !strings.Contains(output, "docs/API.md") {
		t.Error("Report should contain API.md")
	}
}

func TestReport_Stale(t *testing.T) {
	dir := setupTestProject(t)

	// Change the source linked to docs/API.md in a later commit than the doc.
	os.WriteFile(filepath.Join(dir, "src", "handler.go"), []byte(`package main

// @doc docs/API.md
func Handler() {
    // Modified
}
`), 0644)
	commitAll(t, dir, "Modify handler")

	initTestEnv(t, dir)
	reportStale = false
	reportOrphaned = false

	var stdout bytes.Buffer
	reportCmd.SetOut(&stdout)

	reportCmd.RunE(reportCmd, nil)

	output := stdout.String()
	if !strings.Contains(output, "STALE DOCS") {
		t.Error("Report should show stale docs after code change")
	}
	if !strings.Contains(output, "docs/API.md") {
		t.Error("docs/API.md should be marked as stale")
	}
}

func TestReport_Fresh_SameCommit(t *testing.T) {
	dir := setupTestProject(t)

	// Edit code AND its linked doc together in one commit — the shared commit
	// is the doc's review anchor, so nothing should be stale.
	os.WriteFile(filepath.Join(dir, "src", "handler.go"), []byte(`package main

// @doc docs/API.md
func Handler() { /* changed */ }
`), 0644)
	os.WriteFile(filepath.Join(dir, "docs", "API.md"), []byte("# API Docs\n\nupdated\n"), 0644)
	commitAll(t, dir, "Change handler and doc together")

	initTestEnv(t, dir)
	reportStale = false
	reportOrphaned = false

	var stdout bytes.Buffer
	reportCmd.SetOut(&stdout)

	reportCmd.RunE(reportCmd, nil)

	if strings.Contains(stdout.String(), "STALE DOCS") {
		t.Errorf("doc committed with its code should not be stale, got:\n%s", stdout.String())
	}
}

func TestAck_SuppressesStale(t *testing.T) {
	dir := setupTestProject(t)

	// Change only the code linked to docs/API.md, in its own commit → stale.
	os.WriteFile(filepath.Join(dir, "src", "handler.go"), []byte(`package main

// @doc docs/API.md
func Handler() { /* changed, but the doc is still accurate */ }
`), 0644)
	commitAll(t, dir, "Change handler only")

	initTestEnv(t, dir)
	reportStale = false
	reportOrphaned = false

	var stdout bytes.Buffer
	reportCmd.SetOut(&stdout)
	reportCmd.RunE(reportCmd, nil)
	if !strings.Contains(stdout.String(), "STALE DOCS") {
		t.Fatalf("precondition: API.md should be stale before ack, got:\n%s", stdout.String())
	}

	// Ack it at HEAD: reviewed, no edit needed.
	ackTo = ""
	var ackOut bytes.Buffer
	ackCmd.SetOut(&ackOut)
	if err := ackCmd.RunE(ackCmd, []string{"docs/API.md"}); err != nil {
		t.Fatalf("ack failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".docdiff-acks.json")); err != nil {
		t.Errorf("ack should write .docdiff-acks.json: %v", err)
	}

	// Report is now clean without touching the doc.
	stdout.Reset()
	reportCmd.RunE(reportCmd, nil)
	if strings.Contains(stdout.String(), "STALE DOCS") {
		t.Errorf("acked doc should not be stale, got:\n%s", stdout.String())
	}

	// A later code change re-stales it — the ack only covers up to its floor.
	os.WriteFile(filepath.Join(dir, "src", "handler.go"), []byte(`package main

// @doc docs/API.md
func Handler() { /* changed again, now the doc really is behind */ }
`), 0644)
	commitAll(t, dir, "Change handler again")

	stdout.Reset()
	reportCmd.RunE(reportCmd, nil)
	if !strings.Contains(stdout.String(), "STALE DOCS") {
		t.Errorf("ack should not suppress changes after its floor, got:\n%s", stdout.String())
	}
}

func TestAckAmend_ReanchorsAfterRewrite(t *testing.T) {
	dir := setupTestProject(t)

	os.WriteFile(filepath.Join(dir, "src", "handler.go"), []byte(`package main

// @doc docs/API.md
func Handler() { /* reviewed via amend ack */ }
`), 0644)
	commitAll(t, dir, "Change handler only")

	initTestEnv(t, dir)
	ackTo = ""
	ackAmend = true
	defer func() { ackAmend = false }()

	var ackOut bytes.Buffer
	ackCmd.SetOut(&ackOut)
	if err := ackCmd.RunE(ackCmd, []string{"docs/API.md"}); err != nil {
		t.Fatalf("ack --amend failed: %v", err)
	}

	runGit(t, dir, "reflog", "expire", "--expire=now", "--all")
	runGit(t, dir, "gc", "--prune=now")

	reportStale = false
	reportOrphaned = false
	reportUndocumented = false
	reportJSON = false
	reportSARIF = false
	reportCI = false
	reportNoBacklinks = true
	defer func() { reportNoBacklinks = false }()

	var reportOut bytes.Buffer
	reportCmd.SetOut(&reportOut)
	if err := reportCmd.RunE(reportCmd, nil); err != nil {
		t.Fatalf("report failed after ack --amend: %v", err)
	}
	if strings.Contains(reportOut.String(), "STALE DOCS") {
		t.Fatalf("amended ack should remain clean after dangling commit prune, got:\n%s", reportOut.String())
	}

	checkStaged = false
	checkJSON = false
	checkNoBacklinks = true
	checkFiles = []string{"src/handler.go"}
	defer func() {
		checkFiles = nil
		checkNoBacklinks = false
	}()

	var checkOut bytes.Buffer
	checkCmd.SetOut(&checkOut)
	if err := checkCmd.RunE(checkCmd, nil); err != nil {
		t.Fatalf("check --files should honor amended ack baseline, got %v:\n%s", err, checkOut.String())
	}
	if !strings.Contains(checkOut.String(), "No docs are linked to your files changes.") {
		t.Fatalf("check --files should be clean after amended ack, got:\n%s", checkOut.String())
	}
}

func TestAckRequired(t *testing.T) {
	dir := setupTestProject(t)

	// Code-only commit: docs/API.md becomes stale and check would have
	// required it for this diff.
	os.WriteFile(filepath.Join(dir, "src", "handler.go"), []byte(`package main

// @doc docs/API.md
func Handler() { /* reviewed, doc fine */ }
`), 0644)
	commitAll(t, dir, "Change handler only")

	initTestEnv(t, dir)
	ackTo = ""
	ackAmend = false
	ackRequired = true
	defer func() { ackRequired = false }()

	var ackOut bytes.Buffer
	ackCmd.SetOut(&ackOut)
	if err := ackCmd.RunE(ackCmd, nil); err != nil {
		t.Fatalf("ack --required failed: %v", err)
	}
	if !strings.Contains(ackOut.String(), "Acked docs/API.md") {
		t.Fatalf("ack --required should ack docs/API.md, got:\n%s", ackOut.String())
	}

	reportStale = false
	reportOrphaned = false
	var stdout bytes.Buffer
	reportCmd.SetOut(&stdout)
	reportCmd.RunE(reportCmd, nil)
	if strings.Contains(stdout.String(), "STALE DOCS") {
		t.Errorf("doc acked via --required should not be stale, got:\n%s", stdout.String())
	}

	// Re-running finds nothing left to ack.
	ackOut.Reset()
	if err := ackCmd.RunE(ackCmd, nil); err != nil {
		t.Fatalf("second ack --required failed: %v", err)
	}
	if !strings.Contains(ackOut.String(), "Nothing to ack") {
		t.Errorf("second run should have nothing to ack, got:\n%s", ackOut.String())
	}
}

func TestAckRequired_SkipsUpdatedDoc(t *testing.T) {
	dir := setupTestProject(t)

	// Code and doc committed together: nothing was required for that commit.
	os.WriteFile(filepath.Join(dir, "src", "handler.go"), []byte(`package main

// @doc docs/API.md
func Handler() { /* changed */ }
`), 0644)
	os.WriteFile(filepath.Join(dir, "docs", "API.md"), []byte("# API Docs\n\nupdated\n"), 0644)
	commitAll(t, dir, "Change handler and doc together")

	initTestEnv(t, dir)
	ackTo = ""
	ackAmend = false
	ackRequired = true
	defer func() { ackRequired = false }()

	var ackOut bytes.Buffer
	ackCmd.SetOut(&ackOut)
	if err := ackCmd.RunE(ackCmd, nil); err != nil {
		t.Fatalf("ack --required failed: %v", err)
	}
	if !strings.Contains(ackOut.String(), "Nothing to ack") {
		t.Errorf("doc updated with its code should not be acked, got:\n%s", ackOut.String())
	}
}

func TestAckRequired_RejectsDocArgs(t *testing.T) {
	dir := setupTestProject(t)
	initTestEnv(t, dir)

	ackRequired = true
	defer func() { ackRequired = false }()
	if err := ackCmd.RunE(ackCmd, []string{"docs/API.md"}); err == nil {
		t.Error("ack --required with doc args should fail")
	}
}

func TestCheck_ScopeHitProvenance(t *testing.T) {
	dir := setupTestProject(t)

	os.WriteFile(filepath.Join(dir, "src", "central.go"), []byte(`package main

// @doc docs/API.md #alpha
func Alpha() {}

// @doc docs/GUIDE.md #beta
func Beta() {}
`), 0644)
	commitAll(t, dir, "Add central file with scoped annotations")

	// Touch only the #beta region.
	os.WriteFile(filepath.Join(dir, "src", "central.go"), []byte(`package main

// @doc docs/API.md #alpha
func Alpha() {}

// @doc docs/GUIDE.md #beta
func Beta() { betaChanged() }
`), 0644)

	initTestEnv(t, dir)
	checkStaged = false
	checkJSON = false
	checkNoBacklinks = true
	checkFiles = nil
	defer func() {
		checkNoBacklinks = false
		checkFiles = nil
	}()

	var stdout bytes.Buffer
	checkCmd.SetOut(&stdout)
	err := checkCmd.RunE(checkCmd, nil)
	if err == nil {
		t.Fatal("check should require docs/GUIDE.md")
	}
	out := stdout.String()
	if !strings.Contains(out, "docs/GUIDE.md: needs update") {
		t.Fatalf("GUIDE should be required, got:\n%s", out)
	}
	if strings.Contains(out, "docs/API.md: needs update") {
		t.Fatalf("API (untouched scope) should not be required, got:\n%s", out)
	}
	if !strings.Contains(out, "lines 7–7 hit #beta") {
		t.Errorf("expected scope-hit line provenance, got:\n%s", out)
	}
	if !strings.Contains(out, "> func Beta() { betaChanged() }") {
		t.Errorf("expected changed-line excerpt, got:\n%s", out)
	}
}

func TestCheck_TerminatorEndsScopedRegion(t *testing.T) {
	dir := setupTestProject(t)

	os.WriteFile(filepath.Join(dir, "src", "central.go"), []byte(`package main

// @doc docs/API.md #alpha
func Alpha() {}

// @doc -
func Unowned() {}
`), 0644)
	commitAll(t, dir, "Add central file with terminated region")

	// Change only code BELOW the terminator: no doc owns it.
	os.WriteFile(filepath.Join(dir, "src", "central.go"), []byte(`package main

// @doc docs/API.md #alpha
func Alpha() {}

// @doc -
func Unowned() { changed() }
`), 0644)

	initTestEnv(t, dir)
	checkStaged = false
	checkJSON = false
	checkNoBacklinks = true
	checkFiles = nil
	defer func() {
		checkNoBacklinks = false
		checkFiles = nil
	}()

	var stdout bytes.Buffer
	checkCmd.SetOut(&stdout)
	if err := checkCmd.RunE(checkCmd, nil); err != nil {
		t.Fatalf("change below '@doc -' must not flag any doc, got %v:\n%s", err, stdout.String())
	}
	// Scoped to the Required section: "Outstanding from HEAD" legitimately names
	// API.md here (the setup commit added its annotation), which says nothing
	// about whether this working-tree change flagged it.
	if strings.Contains(stdout.String(), "docs/API.md: needs update") {
		t.Errorf("API.md should not be affected, got:\n%s", stdout.String())
	}

	// Sanity: a change INSIDE the scoped region still flags the doc.
	os.WriteFile(filepath.Join(dir, "src", "central.go"), []byte(`package main

// @doc docs/API.md #alpha
func Alpha() { changedInScope() }

// @doc -
func Unowned() {}
`), 0644)
	stdout.Reset()
	if err := checkCmd.RunE(checkCmd, nil); err == nil {
		t.Fatalf("change inside the scoped region should flag the doc, got:\n%s", stdout.String())
	}
}

func TestCheck_ScopeNotes(t *testing.T) {
	dir := setupTestProject(t)

	// Stage one change, leave another outside the index.
	os.WriteFile(filepath.Join(dir, "src", "handler.go"), []byte(`package main

// @doc docs/API.md
func Handler() { staged() }
`), 0644)
	runGit(t, dir, "add", "src/handler.go")
	os.WriteFile(filepath.Join(dir, "src", "util.go"), []byte(`package main

func Util() { unstaged() }
`), 0644)

	initTestEnv(t, dir)
	checkJSON = false
	checkNoBacklinks = true
	checkFiles = nil
	defer func() {
		checkNoBacklinks = false
		checkFiles = nil
	}()

	// Staged mode: the unstaged util.go change is ignored — say so.
	checkStaged = true
	var stdout bytes.Buffer
	checkCmd.SetOut(&stdout)
	checkCmd.RunE(checkCmd, nil)
	if !strings.Contains(stdout.String(), "1 changed file(s) outside the index ignored") {
		t.Errorf("staged mode should note ignored working-tree files, got:\n%s", stdout.String())
	}

	// Working-tree mode with staged changes: the mirror note.
	checkStaged = false
	stdout.Reset()
	checkCmd.RunE(checkCmd, nil)
	if !strings.Contains(stdout.String(), "1 file(s) are staged; working-tree scope checks them together") {
		t.Errorf("working-tree mode should note staged files, got:\n%s", stdout.String())
	}
}

func TestCheck_ConfigScopeStaged(t *testing.T) {
	dir := setupTestProject(t)
	os.WriteFile(filepath.Join(dir, ".docdiff.yaml"), []byte("check:\n  scope: staged\n"), 0644)

	// An unstaged change only: with scope pinned to staged, check sees nothing.
	os.WriteFile(filepath.Join(dir, "src", "handler.go"), []byte(`package main

// @doc docs/API.md
func Handler() { unstaged() }
`), 0644)

	initTestEnv(t, dir)
	checkStaged = false
	checkJSON = false
	checkNoBacklinks = true
	checkFiles = nil
	defer func() {
		checkNoBacklinks = false
		checkFiles = nil
	}()

	var stdout bytes.Buffer
	checkCmd.SetOut(&stdout)
	if err := checkCmd.RunE(checkCmd, nil); err != nil {
		t.Fatalf("config-scoped staged check should pass with nothing staged, got %v:\n%s", err, stdout.String())
	}
	if !strings.Contains(stdout.String(), "staged") {
		t.Errorf("output should reflect the staged scope, got:\n%s", stdout.String())
	}
}

func TestCheck_ConfigScopeInvalid(t *testing.T) {
	dir := setupTestProject(t)
	os.WriteFile(filepath.Join(dir, ".docdiff.yaml"), []byte("check:\n  scope: bogus\n"), 0644)

	initTestEnv(t, dir)
	checkStaged = false
	checkJSON = false
	checkFiles = nil
	defer func() { checkFiles = nil }()

	if err := checkCmd.RunE(checkCmd, nil); err == nil || !strings.Contains(err.Error(), "check.scope") {
		t.Errorf("invalid check.scope should error, got: %v", err)
	}
}

func TestAck_UnknownDoc(t *testing.T) {
	dir := setupTestProject(t)
	initTestEnv(t, dir)

	ackTo = ""
	if err := ackCmd.RunE(ackCmd, []string{"docs/NOPE.md"}); err == nil {
		t.Error("ack should fail for a doc that does not exist")
	}
}

func TestReport_JSON(t *testing.T) {
	dir := setupTestProject(t)
	initTestEnv(t, dir)

	reportJSON = true
	reportStale = false
	reportOrphaned = false
	reportSARIF = false
	defer func() { reportJSON = false }()

	var stdout bytes.Buffer
	reportCmd.SetOut(&stdout)

	reportCmd.RunE(reportCmd, nil)

	output := stdout.String()
	if !strings.HasPrefix(output, "{") {
		t.Error("JSON output should start with {")
	}
	if !strings.Contains(output, "\"files_by_doc\"") {
		t.Error("JSON output should contain files_by_doc field")
	}
}

func TestReportJSON_RootCommandWritesStdout(t *testing.T) {
	dir := setupTestProject(t)

	reportCmd.SetOut(nil)
	reportCmd.SetErr(nil)
	reportJSON = false
	reportStale = false
	reportOrphaned = false
	reportUndocumented = false
	reportSARIF = false
	reportCI = false
	reportNoBacklinks = false

	stdout, stderr, err := runDocdiff(t, dir, "report", "--json")
	if err != nil {
		t.Fatalf("report --json failed: %v\nstderr:\n%s", err, stderr)
	}
	if !strings.HasPrefix(stdout, "{") {
		t.Fatalf("report --json should write JSON to stdout, got stdout=%q stderr prefix=%q", stdout, stderr[:min(len(stderr), 80)])
	}
	if stderr != "" {
		t.Fatalf("report --json should not write JSON to stderr, got:\n%s", stderr)
	}
}

func TestChanges_Integration(t *testing.T) {
	dir := setupTestProject(t)

	os.WriteFile(filepath.Join(dir, "src", "handler.go"), []byte(`package main

// @doc docs/API.md
func Handler() {
    fmt.Println("new code")
}
`), 0644)
	commitAll(t, dir, "Add new code")

	initTestEnv(t, dir)
	changesCommits = false
	changesSummary = false
	changesAI = false

	var stdout bytes.Buffer
	changesCmd.SetOut(&stdout)

	if err := changesCmd.RunE(changesCmd, []string{"docs/API.md"}); err != nil {
		t.Fatalf("changes failed: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "Changes to docs/API.md") {
		t.Error("Changes should show doc name")
	}
	if !strings.Contains(output, "Commits:") {
		t.Error("Changes should show commit count")
	}
}

func TestChanges_UsesAckFloor(t *testing.T) {
	dir := setupTestProject(t)

	os.WriteFile(filepath.Join(dir, "src", "handler.go"), []byte(`package main

// @doc docs/API.md
func Handler() { /* code changed, doc reviewed without edit */ }
`), 0644)
	commitAll(t, dir, "Modify handler")

	initTestEnv(t, dir)
	ackTo = ""
	ackAmend = false
	var ackOut bytes.Buffer
	ackCmd.SetOut(&ackOut)
	if err := ackCmd.RunE(ackCmd, []string{"docs/API.md"}); err != nil {
		t.Fatalf("ack failed: %v", err)
	}

	changesCommits = false
	changesSummary = false
	changesAI = false
	changesWorkTree = false
	changesStaged = false
	var stdout bytes.Buffer
	changesCmd.SetOut(&stdout)

	if err := changesCmd.RunE(changesCmd, []string{"docs/API.md"}); err != nil {
		t.Fatalf("changes failed: %v", err)
	}
	if !strings.Contains(stdout.String(), "No changes since last documentation update.") {
		t.Fatalf("changes should diff from ack floor, got:\n%s", stdout.String())
	}
}

func TestChanges_AI(t *testing.T) {
	dir := setupTestProject(t)

	os.WriteFile(filepath.Join(dir, "src", "handler.go"), []byte(`package main

// @doc docs/API.md
func Handler() { /* modified */ }
`), 0644)
	commitAll(t, dir, "Modify handler")

	initTestEnv(t, dir)
	changesAI = true
	changesCommits = false
	changesSummary = false
	defer func() { changesAI = false }()

	if err := changesCmd.RunE(changesCmd, []string{"docs/API.md"}); err != nil {
		t.Fatalf("changes --ai failed: %v", err)
	}
}

func TestOnboard_Output(t *testing.T) {
	var stdout bytes.Buffer
	onboardCmd.SetOut(&stdout)

	err := onboardCmd.RunE(onboardCmd, nil)
	if err != nil {
		t.Fatalf("onboard failed: %v", err)
	}

	output := stdout.String()

	// Verify key commands are mentioned
	for _, want := range []string{
		"docdiff check",
		"docdiff report",
		"docdiff changes",
		"docdiff graph",
	} {
		if !strings.Contains(output, want) {
			t.Errorf("onboard output should mention %q", want)
		}
	}

	// Verify agent instruction file names are mentioned
	for _, want := range []string{
		"CLAUDE.md",
		".github/copilot-instructions.md",
		".cursorrules",
		".windsurfrules",
		"AGENTS.md",
	} {
		if !strings.Contains(output, want) {
			t.Errorf("onboard output should mention agent file %q", want)
		}
	}

	// Verify delimiters are present
	if !strings.Contains(output, "START DOCDIFF INSTRUCTIONS") {
		t.Error("onboard output should contain START marker")
	}
	if !strings.Contains(output, "END DOCDIFF INSTRUCTIONS") {
		t.Error("onboard output should contain END marker")
	}
}

func TestOnboard_NoPersistentPreRun(t *testing.T) {
	// Clear global state to verify the command works without project setup
	origDir := rootDir
	origCfg := cfg
	origRegistry := registry
	rootDir = ""
	cfg = nil
	registry = nil
	defer func() {
		rootDir = origDir
		cfg = origCfg
		registry = origRegistry
	}()

	var stdout bytes.Buffer
	onboardCmd.SetOut(&stdout)

	err := onboardCmd.RunE(onboardCmd, nil)
	if err != nil {
		t.Fatalf("onboard should work without project setup: %v", err)
	}

	if stdout.Len() == 0 {
		t.Error("onboard should produce output even without project setup")
	}
}

func TestOnboardIncludesSkillInstallPromptForAgents(t *testing.T) {
	dir := t.TempDir()
	onboardCmd.SetOut(nil)
	onboardCmd.SetErr(nil)
	onboardCmd.SetIn(nil)
	t.Cleanup(func() {
		onboardCmd.SetOut(nil)
		onboardCmd.SetErr(nil)
		onboardCmd.SetIn(nil)
	})

	stdout, stderr, err := runDocdiff(t, dir, "onboard")
	if err != nil {
		t.Fatalf("onboard failed: %v", err)
	}
	if !strings.Contains(stdout, "# docdiff") {
		t.Fatalf("onboard should print instructions, got:\n%s", stdout)
	}
	if stderr != "" {
		t.Fatalf("onboard should not write prompts to stderr, got:\n%s", stderr)
	}

	for _, want := range []string{
		"Optional Coding-Agent Skill",
		"ask the user before installing",
		"Do not write any skill files unless the user says yes.",
		".claude/skills/docdiff/SKILL.md",
		".agents/skills/docdiff/SKILL.md",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("onboard output should include %q, got:\n%s", want, stdout)
		}
	}

	if !strings.Contains(stdout, docdiffSkillContent) {
		t.Fatal("onboard output should include the bundled docdiff skill content")
	}

	if _, err := os.Stat(filepath.Join(dir, ".claude")); !os.IsNotExist(err) {
		t.Fatalf("onboard should not install skill files directly, stat err=%v", err)
	}
}

func TestCheck_WorkingTree(t *testing.T) {
	dir := setupTestProject(t)
	initTestEnv(t, dir)

	// Modify a source file linked to docs/API.md, but leave it uncommitted.
	os.WriteFile(filepath.Join(dir, "src", "handler.go"), []byte(`package main

// @doc docs/API.md
func Handler() {
    // uncommitted change
}
`), 0644)

	initTestEnv(t, dir)
	checkStaged = false
	checkJSON = false
	checkFiles = nil
	defer func() { checkFiles = nil }()

	var stdout bytes.Buffer
	checkCmd.SetOut(&stdout)

	err := checkCmd.RunE(checkCmd, nil)
	if err == nil {
		t.Error("check should fail when a linked doc needs updating")
	}
	out := stdout.String()
	if !strings.Contains(out, "docs/API.md: needs update") {
		t.Errorf("expected API.md to need update, got:\n%s", out)
	}

	// Now also edit the doc itself: it should count as "updated" and pass.
	os.WriteFile(filepath.Join(dir, "docs", "API.md"), []byte("# API Docs\n\nupdated\n"), 0644)

	stdout.Reset()
	err = checkCmd.RunE(checkCmd, nil)
	if err != nil {
		t.Errorf("check should pass when the doc was also edited, got: %v", err)
	}
	if !strings.Contains(stdout.String(), "Already updated") || !strings.Contains(stdout.String(), "docs/API.md") {
		t.Errorf("expected API.md in the already-updated section, got:\n%s", stdout.String())
	}
}

// Committing makes the working tree clean, so check's changed set is empty and
// nothing is "linked to your changes" — but the docs that commit put on review
// are still outstanding. They used to be counted as "unrelated stale", which
// told the agent to ignore exactly the ones that mattered.
func TestCheck_OutstandingFromHead(t *testing.T) {
	dir := setupTestProject(t)
	initTestEnv(t, dir)

	os.WriteFile(filepath.Join(dir, "src", "handler.go"), []byte(`package main

// @doc docs/API.md
func Handler() {
    committed()
}
`), 0644)
	commitAll(t, dir, "Change handler without touching its doc")

	initTestEnv(t, dir)
	checkStaged, checkJSON, checkFiles, checkNoBacklinks = false, false, nil, true
	defer func() { checkNoBacklinks, checkFiles = false, nil }()

	var stdout bytes.Buffer
	checkCmd.SetOut(&stdout)
	if err := checkCmd.RunE(checkCmd, nil); err != nil {
		t.Errorf("outstanding-from-HEAD reports on an already-made commit; it must not gate, got: %v", err)
	}
	out := stdout.String()
	if !strings.Contains(out, "Outstanding from HEAD (1)") || !strings.Contains(out, "docs/API.md") {
		t.Errorf("expected API.md listed as outstanding from HEAD, got:\n%s", out)
	}
	// The wording is the fix: a bare count is what got ignored before.
	if !strings.Contains(out, "docdiff ack --required --amend") {
		t.Errorf("outstanding section must name the command that clears it, got:\n%s", out)
	}
	if strings.Contains(out, "unrelated stale") {
		t.Errorf("HEAD-caused staleness must not be counted as unrelated, got:\n%s", out)
	}

	// Acking it resolves the section.
	ackRequired = true
	defer func() { ackRequired = false }()
	var ackOut bytes.Buffer
	ackCmd.SetOut(&ackOut)
	if err := ackCmd.RunE(ackCmd, nil); err != nil {
		t.Fatalf("ack --required failed: %v", err)
	}
	if !strings.Contains(ackOut.String(), "docs/API.md") {
		t.Fatalf("ack --required should have found the same doc, got:\n%s", ackOut.String())
	}

	stdout.Reset()
	if err := checkCmd.RunE(checkCmd, nil); err != nil {
		t.Errorf("check should still pass after ack, got: %v", err)
	}
	if strings.Contains(stdout.String(), "Outstanding from HEAD") {
		t.Errorf("acked doc should clear the outstanding section, got:\n%s", stdout.String())
	}
}

func TestCheck_PrintsProvenanceAndBroadHint(t *testing.T) {
	dir := setupTestProject(t)

	for i := 0; i < 19; i++ {
		path := filepath.Join(dir, "src", "extra"+strconv.Itoa(i)+".go")
		os.WriteFile(path, []byte(`package main
// @doc docs/API.md
func Extra() {}
`), 0644)
	}
	commitAll(t, dir, "Add broad API coverage")

	os.WriteFile(filepath.Join(dir, "src", "extra0.go"), []byte(`package main
// @doc docs/API.md
func Extra() { /* changed */ }
`), 0644)

	initTestEnv(t, dir)
	checkStaged = false
	checkJSON = false
	checkNoBacklinks = true
	checkFiles = nil
	defer func() {
		checkNoBacklinks = false
		checkFiles = nil
	}()

	var stdout bytes.Buffer
	checkCmd.SetOut(&stdout)
	err := checkCmd.RunE(checkCmd, nil)
	if err == nil {
		t.Fatal("check should fail when broad linked doc needs update")
	}
	out := stdout.String()
	if !strings.Contains(out, "docs/API.md: needs update (broad: 20 linked files)") {
		t.Fatalf("check should flag broad docs inline, got:\n%s", out)
	}
	if !strings.Contains(out, "via src/extra0.go:2 whole-file @doc") {
		t.Fatalf("check should print annotation provenance, got:\n%s", out)
	}
	if !strings.Contains(out, "docdiff onboard") {
		t.Fatalf("check failure should suggest onboard, got:\n%s", out)
	}
}

func TestCheck_AnnotationOnlyChange(t *testing.T) {
	dir := setupTestProject(t)

	// Only the @doc line changes (bare -> scoped, same doc): ownership churn,
	// not a code change. Must not gate, must land in the ownership section.
	os.WriteFile(filepath.Join(dir, "src", "handler.go"), []byte(`package main

// @doc docs/API.md #handler
func Handler() {}
`), 0644)

	initTestEnv(t, dir)
	checkStaged = false
	checkJSON = false
	checkNoBacklinks = true
	checkFiles = nil
	defer func() {
		checkNoBacklinks = false
		checkFiles = nil
	}()

	var stdout bytes.Buffer
	checkCmd.SetOut(&stdout)
	if err := checkCmd.RunE(checkCmd, nil); err != nil {
		t.Fatalf("annotation-only change must not gate, got: %v\n%s", err, stdout.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "Ownership map changed (1):") {
		t.Errorf("expected ownership section, got:\n%s", out)
	}
	if !strings.Contains(out, "docs/API.md: annotation-only change") {
		t.Errorf("expected docs/API.md in ownership section, got:\n%s", out)
	}
	if strings.Contains(out, "needs update") {
		t.Errorf("annotation-only change must not require an update, got:\n%s", out)
	}

	// JSON: additive ownership_changed array, needs_update stays 0.
	checkJSON = true
	defer func() { checkJSON = false }()
	stdout.Reset()
	if err := checkCmd.RunE(checkCmd, nil); err != nil {
		t.Fatalf("check --json failed: %v", err)
	}
	if !strings.Contains(stdout.String(), "\"ownership_changed\"") {
		t.Errorf("JSON should carry ownership_changed, got:\n%s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "\"needs_update\": 0") {
		t.Errorf("needs_update should be 0, got:\n%s", stdout.String())
	}
}

func TestCheck_AnnotationOnlyChange_Staged(t *testing.T) {
	dir := setupTestProject(t)

	os.WriteFile(filepath.Join(dir, "src", "handler.go"), []byte(`package main

// @doc docs/API.md #handler
func Handler() {}
`), 0644)
	runGit(t, dir, "add", ".")

	initTestEnv(t, dir)
	checkStaged = true
	checkJSON = false
	checkNoBacklinks = true
	checkFiles = nil
	defer func() {
		checkStaged = false
		checkNoBacklinks = false
		checkFiles = nil
	}()

	var stdout bytes.Buffer
	checkCmd.SetOut(&stdout)
	if err := checkCmd.RunE(checkCmd, nil); err != nil {
		t.Fatalf("staged annotation-only change must not gate, got: %v\n%s", err, stdout.String())
	}
	if !strings.Contains(stdout.String(), "Ownership map changed (1):") {
		t.Errorf("expected ownership section in staged mode, got:\n%s", stdout.String())
	}
}

func TestCheck_MixedChangeStillGates(t *testing.T) {
	dir := setupTestProject(t)

	// Annotation churn AND a real code change in the same file: still required.
	os.WriteFile(filepath.Join(dir, "src", "handler.go"), []byte(`package main

// @doc docs/API.md #handler
func Handler() { realChange() }
`), 0644)

	initTestEnv(t, dir)
	checkStaged = false
	checkJSON = false
	checkNoBacklinks = true
	checkFiles = nil
	defer func() {
		checkNoBacklinks = false
		checkFiles = nil
	}()

	var stdout bytes.Buffer
	checkCmd.SetOut(&stdout)
	if err := checkCmd.RunE(checkCmd, nil); err == nil {
		t.Fatalf("mixed change should still gate, got:\n%s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "docs/API.md: needs update") {
		t.Errorf("expected docs/API.md required, got:\n%s", stdout.String())
	}
}

func TestReport_AnnotationOnlyCommitNotStale(t *testing.T) {
	dir := setupTestProject(t)

	// Commit ONLY an annotation change to the linked file: the doc must not be
	// reported stale (report stays consistent with check).
	os.WriteFile(filepath.Join(dir, "src", "handler.go"), []byte(`package main

// @doc docs/API.md #handler
func Handler() {}
`), 0644)
	commitAll(t, dir, "Retarget annotation only")

	initTestEnv(t, dir)
	reportStale = false
	reportOrphaned = false

	var stdout bytes.Buffer
	reportCmd.SetOut(&stdout)
	reportCmd.RunE(reportCmd, nil)
	if strings.Contains(stdout.String(), "STALE DOCS") {
		t.Errorf("annotation-only commit should not stale the doc, got:\n%s", stdout.String())
	}
}

func TestReport_NoBacklinksFlag(t *testing.T) {
	dir := setupTestProject(t)
	initTestEnv(t, dir)

	reportStale = false
	reportOrphaned = false
	reportUndocumented = false
	reportJSON = false
	reportSARIF = false
	reportCI = false
	reportNoBacklinks = true
	defer func() { reportNoBacklinks = false }()

	var stdout bytes.Buffer
	reportCmd.SetOut(&stdout)
	if err := reportCmd.RunE(reportCmd, nil); err != nil {
		t.Fatalf("report --no-backlinks behavior should not fail: %v", err)
	}
}

func TestIsCI(t *testing.T) {
	originalCI, ciSet := os.LookupEnv("CI")
	originalGHA, ghaSet := os.LookupEnv("GITHUB_ACTIONS")
	originalGitlab, gitlabSet := os.LookupEnv("GITLAB_CI")

	defer func() {
		if ciSet {
			os.Setenv("CI", originalCI)
		} else {
			os.Unsetenv("CI")
		}
		if ghaSet {
			os.Setenv("GITHUB_ACTIONS", originalGHA)
		} else {
			os.Unsetenv("GITHUB_ACTIONS")
		}
		if gitlabSet {
			os.Setenv("GITLAB_CI", originalGitlab)
		} else {
			os.Unsetenv("GITLAB_CI")
		}
	}()

	os.Unsetenv("CI")
	os.Unsetenv("GITHUB_ACTIONS")
	os.Unsetenv("GITLAB_CI")

	if isCI() {
		t.Error("isCI() should return false when no CI env vars set")
	}

	os.Setenv("CI", "true")
	if !isCI() {
		t.Error("isCI() should return true when CI=true")
	}

	os.Unsetenv("CI")
	os.Setenv("GITHUB_ACTIONS", "true")
	if !isCI() {
		t.Error("isCI() should return true when GITHUB_ACTIONS=true")
	}
}

// A central file carrying several scoped annotations must only stale the doc
// whose region actually changed. check honoured this for the working tree while
// report/explain/graph matched at file level, so a 1700-line file with 30 well
// placed scopes flagged every doc touching it — the opposite of what scoping is
// for. In Hangar this accounted for 18 of 19 stale docs, all reviewed as
// still-true.
func TestReport_ScopedRegionNotTouched(t *testing.T) {
	dir := setupTestProject(t)

	os.MkdirAll(filepath.Join(dir, "docs"), 0755)
	os.WriteFile(filepath.Join(dir, "docs", "ALPHA.md"), []byte("# Alpha\n\nsrc/central.go\n"), 0644)
	os.WriteFile(filepath.Join(dir, "docs", "BETA.md"), []byte("# Beta\n\nsrc/central.go\n"), 0644)
	os.WriteFile(filepath.Join(dir, "src", "central.go"), []byte(`package main

// @doc docs/ALPHA.md #alpha
func Alpha() {}

// @doc docs/BETA.md #beta
func Beta() {}
`), 0644)
	commitAll(t, dir, "Add central file with two scoped regions")

	// Change ONLY the #beta region, in a later commit than both docs.
	os.WriteFile(filepath.Join(dir, "src", "central.go"), []byte(`package main

// @doc docs/ALPHA.md #alpha
func Alpha() {}

// @doc docs/BETA.md #beta
func Beta() { changed() }
`), 0644)
	commitAll(t, dir, "Change only the beta region")

	initTestEnv(t, dir)
	reportStale, reportOrphaned = false, false

	var stdout bytes.Buffer
	reportCmd.SetOut(&stdout)
	reportCmd.RunE(reportCmd, nil)
	// Scoped to the STALE DOCS section: the coverage listing below it names
	// every doc in the repo, so grepping the whole report proves nothing.
	out := stdout.String()
	staleSection := out
	if i := strings.Index(out, "STALE DOCS"); i >= 0 {
		staleSection = out[i:]
		if j := strings.Index(staleSection, "\nBy Documentation File"); j >= 0 {
			staleSection = staleSection[:j]
		}
	} else {
		staleSection = ""
	}

	if strings.Contains(staleSection, "docs/ALPHA.md") {
		t.Errorf("ALPHA owns lines the commit never touched; it must not be stale:\n%s", out)
	}
	if !strings.Contains(staleSection, "docs/BETA.md") {
		t.Errorf("BETA owns the changed region and must still be stale:\n%s", out)
	}
}

// The narrowing must not weaken whole-file links: without a scope, any change
// to a linked file still stales the doc.
func TestReport_WholeFileLinkStillStales(t *testing.T) {
	dir := setupTestProject(t)

	os.WriteFile(filepath.Join(dir, "src", "handler.go"), []byte(`package main

// @doc docs/API.md
func Handler() { changed() }
`), 0644)
	commitAll(t, dir, "Change a whole-file-linked file")

	initTestEnv(t, dir)
	reportStale, reportOrphaned = false, false

	var stdout bytes.Buffer
	reportCmd.SetOut(&stdout)
	reportCmd.RunE(reportCmd, nil)
	if !strings.Contains(stdout.String(), "docs/API.md") {
		t.Errorf("a whole-file link must still stale on any change, got:\n%s", stdout.String())
	}
}

// `changes` is the command an author reads to decide what to edit, so showing a
// hunk from another doc's region invites an edit for code this doc doesn't
// describe. It narrows the same way report and explain do.
func TestChanges_NarrowsToOwnedRegions(t *testing.T) {
	dir := setupTestProject(t)

	os.WriteFile(filepath.Join(dir, "docs", "ALPHA.md"), []byte("# Alpha\n\nsrc/central.go\n"), 0644)
	os.WriteFile(filepath.Join(dir, "docs", "BETA.md"), []byte("# Beta\n\nsrc/central.go\n"), 0644)
	os.WriteFile(filepath.Join(dir, "src", "central.go"), []byte(`package main

// @doc docs/ALPHA.md #alpha
func Alpha() {}

// @doc docs/BETA.md #beta
func Beta() {}
`), 0644)
	commitAll(t, dir, "Add central file")

	os.WriteFile(filepath.Join(dir, "src", "central.go"), []byte(`package main

// @doc docs/ALPHA.md #alpha
func Alpha() {}

// @doc docs/BETA.md #beta
func Beta() { betaOnlyChange() }
`), 0644)
	commitAll(t, dir, "Change only the beta region")

	initTestEnv(t, dir)
	changesWorkTree, changesStaged, changesAI, changesSummary, changesCommits = false, false, false, false, false

	var stdout bytes.Buffer
	changesCmd.SetOut(&stdout)
	if err := changesCmd.RunE(changesCmd, []string{"docs/ALPHA.md"}); err != nil {
		t.Fatalf("changes failed: %v", err)
	}
	if strings.Contains(stdout.String(), "betaOnlyChange") {
		t.Errorf("ALPHA must not be shown BETA's hunk:\n%s", stdout.String())
	}

	stdout.Reset()
	if err := changesCmd.RunE(changesCmd, []string{"docs/BETA.md"}); err != nil {
		t.Fatalf("changes failed: %v", err)
	}
	if !strings.Contains(stdout.String(), "betaOnlyChange") {
		t.Errorf("BETA owns the changed region and must still see its diff:\n%s", stdout.String())
	}
}

// Narrowing in `changes` is file-granular, not hunk-granular, and that is
// deliberate: once a file has an in-scope change the whole file's diff prints,
// because a hunk-filtered diff shows fragments without their surroundings and
// this is the command an author reads to decide what to edit. It also errs
// toward showing too much, which is the safe direction for review. Pinned so
// the choice can't drift into hunk-filtering by accident.
func TestChanges_NarrowsByFileNotByHunk(t *testing.T) {
	dir := setupTestProject(t)

	os.WriteFile(filepath.Join(dir, "docs", "ALPHA.md"), []byte("# Alpha\n\nsrc/central.go\n"), 0644)
	os.WriteFile(filepath.Join(dir, "docs", "BETA.md"), []byte("# Beta\n\nsrc/central.go\n"), 0644)
	os.WriteFile(filepath.Join(dir, "src", "central.go"), []byte(`package main

// @doc docs/ALPHA.md #alpha
func Alpha() {}

// @doc docs/BETA.md #beta
func Beta() {}
`), 0644)
	commitAll(t, dir, "Add central file")

	// One commit touching BOTH regions.
	os.WriteFile(filepath.Join(dir, "src", "central.go"), []byte(`package main

// @doc docs/ALPHA.md #alpha
func Alpha() { alphaProbe() }

// @doc docs/BETA.md #beta
func Beta() { betaProbe() }
`), 0644)
	commitAll(t, dir, "Change both regions")

	initTestEnv(t, dir)
	changesWorkTree, changesStaged, changesAI, changesSummary, changesCommits = false, false, false, false, false

	var stdout bytes.Buffer
	changesCmd.SetOut(&stdout)
	if err := changesCmd.RunE(changesCmd, []string{"docs/ALPHA.md"}); err != nil {
		t.Fatalf("changes failed: %v", err)
	}
	out := stdout.String()
	if !strings.Contains(out, "alphaProbe") {
		t.Errorf("ALPHA owns this hunk and must see it:\n%s", out)
	}
	if !strings.Contains(out, "betaProbe") {
		t.Errorf("file-granular narrowing: once the file is kept its whole diff prints, "+
			"including hunks ALPHA does not own — if this now fails, narrowing became "+
			"hunk-granular and the docs need updating:\n%s", out)
	}
}

// Regression: with a --reason the ack entry is an object, so the doc's KEY line
// stays unchanged when only the nested "commit" value moves. Re-anchoring used
// `git log -G` over that key line, so it walked past the commit that actually
// carried the ack and landed on whenever the key was introduced — resolving the
// floor BACKWARD, which can only under-report freshness. `ack --amend` then
// never converged for any doc carrying a reason.
func TestAckAmend_ReanchorsWithReason(t *testing.T) {
	dir := setupTestProject(t)

	// First ack, with a reason, so the entry is written in object form.
	os.WriteFile(filepath.Join(dir, "src", "handler.go"), []byte(`package main

// @doc docs/API.md
func Handler() { first() }
`), 0644)
	commitAll(t, dir, "Change handler once")

	initTestEnv(t, dir)
	ackTo, ackAmend, ackRequired = "", true, false
	ackReason = "reviewed: rename only"
	defer func() { ackAmend, ackReason, ackRequired = false, "", false }()

	var ackOut bytes.Buffer
	ackCmd.SetOut(&ackOut)
	if err := ackCmd.RunE(ackCmd, []string{"docs/API.md"}); err != nil {
		t.Fatalf("first ack failed: %v", err)
	}

	// Second round: the key line is now already present, so only the nested
	// commit value changes — the exact shape that defeated the regex.
	os.WriteFile(filepath.Join(dir, "src", "handler.go"), []byte(`package main

// @doc docs/API.md
func Handler() { second() }
`), 0644)
	commitAll(t, dir, "Change handler again")

	ackOut.Reset()
	if err := ackCmd.RunE(ackCmd, []string{"docs/API.md"}); err != nil {
		t.Fatalf("second ack failed: %v", err)
	}

	head := runGit(t, dir, "rev-parse", "--short", "HEAD")

	acks, err := loadAcks(dir)
	if err != nil {
		t.Fatalf("loadAcks: %v", err)
	}
	floor, reanchored := resolveAckFloor(git.New(dir), "docs/API.md", acks["docs/API.md"])
	if !reanchored {
		t.Fatalf("amend rewrote the recorded sha, so the floor must re-anchor; got floor=%s", floor)
	}
	if floor != head {
		t.Errorf("floor re-anchored to %s, want HEAD %s — resolving anywhere else moves the floor backward", floor, head)
	}

	// And the doc must actually be fresh: a floor that lands behind the change
	// it reviewed leaves the doc stale and the loop never converges.
	reportStale, reportOrphaned, reportNoBacklinks = false, false, true
	var stdout bytes.Buffer
	reportCmd.SetOut(&stdout)
	reportCmd.RunE(reportCmd, nil)
	if strings.Contains(stdout.String(), "STALE DOCS") {
		t.Errorf("acked-with-reason doc should be fresh after --amend, got:\n%s", stdout.String())
	}
}
