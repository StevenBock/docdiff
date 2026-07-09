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
