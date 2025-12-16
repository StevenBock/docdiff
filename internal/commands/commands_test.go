package commands

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
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

func TestInit_Integration(t *testing.T) {
	dir := setupTestProject(t)

	metaPath := filepath.Join(dir, "docs", ".doc-versions.json")
	if _, err := os.Stat(metaPath); !os.IsNotExist(err) {
		t.Fatal("Metadata should not exist before init")
	}

	initTestEnv(t, dir)

	err := initCmd.RunE(initCmd, nil)
	if err != nil {
		t.Fatalf("init failed: %v", err)
	}

	if _, err := os.Stat(metaPath); os.IsNotExist(err) {
		t.Error("Metadata file should exist after init")
	}

	content, _ := os.ReadFile(metaPath)
	if !strings.Contains(string(content), "docs/API.md") {
		t.Error("Metadata should contain docs/API.md")
	}
	if !strings.Contains(string(content), "docs/GUIDE.md") {
		t.Error("Metadata should contain docs/GUIDE.md")
	}
}

func TestInit_Force(t *testing.T) {
	dir := setupTestProject(t)

	metaPath := filepath.Join(dir, "docs", ".doc-versions.json")
	os.WriteFile(metaPath, []byte(`{"docs/OLD.md": "old123"}`), 0644)

	initTestEnv(t, dir)
	initForce = true
	defer func() { initForce = false }()

	err := initCmd.RunE(initCmd, nil)
	if err != nil {
		t.Fatalf("init --force failed: %v", err)
	}

	content, _ := os.ReadFile(metaPath)
	if strings.Contains(string(content), "docs/OLD.md") {
		t.Error("Old metadata should be replaced")
	}
	if !strings.Contains(string(content), "docs/API.md") {
		t.Error("New metadata should be written")
	}
}

func TestInit_AlreadyExists(t *testing.T) {
	dir := setupTestProject(t)

	metaPath := filepath.Join(dir, "docs", ".doc-versions.json")
	os.WriteFile(metaPath, []byte(`{}`), 0644)

	initTestEnv(t, dir)
	initForce = false

	err := initCmd.RunE(initCmd, nil)
	if err == nil {
		t.Error("init should fail when metadata exists without --force")
	}
}

func TestReport_Integration(t *testing.T) {
	dir := setupTestProject(t)

	initTestEnv(t, dir)
	initForce = false
	err := initCmd.RunE(initCmd, nil)
	if err != nil {
		t.Fatalf("init failed: %v", err)
	}

	initTestEnv(t, dir)
	reportStale = false
	reportOrphaned = false
	reportJSON = false
	reportSARIF = false
	reportCI = false

	var stdout bytes.Buffer
	reportCmd.SetOut(&stdout)

	err = reportCmd.RunE(reportCmd, nil)
	if err != nil {
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

	initTestEnv(t, dir)
	initForce = false
	initCmd.RunE(initCmd, nil)

	os.WriteFile(filepath.Join(dir, "src", "handler.go"), []byte(`package main

// @doc docs/API.md
func Handler() {
    // Modified
}
`), 0644)

	cmd := exec.Command("git", "add", ".")
	cmd.Dir = dir
	cmd.Run()
	cmd = exec.Command("git", "commit", "-m", "Modify handler")
	cmd.Dir = dir
	cmd.Run()

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

func TestReport_JSON(t *testing.T) {
	dir := setupTestProject(t)

	initTestEnv(t, dir)
	initCmd.RunE(initCmd, nil)

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
	if !strings.Contains(output, "\"metadata\"") {
		t.Error("JSON output should contain metadata field")
	}
}

func TestSync_Integration(t *testing.T) {
	dir := setupTestProject(t)

	initTestEnv(t, dir)
	initCmd.RunE(initCmd, nil)

	os.WriteFile(filepath.Join(dir, "src", "handler.go"), []byte(`package main

// @doc docs/API.md
func Handler() { /* modified */ }
`), 0644)

	cmd := exec.Command("git", "add", ".")
	cmd.Dir = dir
	cmd.Run()
	cmd = exec.Command("git", "commit", "-m", "Modify")
	cmd.Dir = dir
	cmd.Run()

	initTestEnv(t, dir)

	var stdout bytes.Buffer
	syncCmd.SetOut(&stdout)

	err := syncCmd.RunE(syncCmd, []string{"docs/API.md"})
	if err != nil {
		t.Fatalf("sync failed: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "Updated docs/API.md") {
		t.Error("Sync should report updated doc")
	}
}

func TestChanges_Integration(t *testing.T) {
	dir := setupTestProject(t)

	initTestEnv(t, dir)
	initCmd.RunE(initCmd, nil)

	os.WriteFile(filepath.Join(dir, "src", "handler.go"), []byte(`package main

// @doc docs/API.md
func Handler() {
    fmt.Println("new code")
}
`), 0644)

	cmd := exec.Command("git", "add", ".")
	cmd.Dir = dir
	cmd.Run()
	cmd = exec.Command("git", "commit", "-m", "Add new code")
	cmd.Dir = dir
	cmd.Run()

	initTestEnv(t, dir)
	changesCommits = false
	changesSummary = false
	changesAI = false

	var stdout bytes.Buffer
	changesCmd.SetOut(&stdout)

	err := changesCmd.RunE(changesCmd, []string{"docs/API.md"})
	if err != nil {
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

func TestChanges_AI(t *testing.T) {
	dir := setupTestProject(t)

	initTestEnv(t, dir)
	initCmd.RunE(initCmd, nil)

	os.WriteFile(filepath.Join(dir, "src", "handler.go"), []byte(`package main

// @doc docs/API.md
func Handler() { /* modified */ }
`), 0644)

	cmd := exec.Command("git", "add", ".")
	cmd.Dir = dir
	cmd.Run()
	cmd = exec.Command("git", "commit", "-m", "Modify handler")
	cmd.Dir = dir
	cmd.Run()

	initTestEnv(t, dir)
	changesAI = true
	changesCommits = false
	changesSummary = false
	defer func() { changesAI = false }()

	err := changesCmd.RunE(changesCmd, []string{"docs/API.md"})
	if err != nil {
		t.Fatalf("changes --ai failed: %v", err)
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
