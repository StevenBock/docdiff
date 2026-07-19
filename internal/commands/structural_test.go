package commands

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAnnotationDeltasByFile(t *testing.T) {
	diff := strings.Join([]string{
		"diff --git a/move.go b/move.go",
		"--- a/move.go",
		"+++ b/move.go",
		"@@ -2 +0,0 @@",
		"-// @doc docs/A.md #x",
		"@@ -0,0 +9 @@",
		"+// @doc docs/A.md #x",
		"diff --git a/rescope.go b/rescope.go",
		"--- a/rescope.go",
		"+++ b/rescope.go",
		"@@ -2 +2 @@",
		"-// @doc docs/A.md #old",
		"+// @doc docs/A.md #new",
		"diff --git a/mixed.go b/mixed.go",
		"--- a/mixed.go",
		"+++ b/mixed.go",
		"@@ -1,0 +2,2 @@",
		"+// @doc docs/B.md",
		"+func New() {}",
		"diff --git a/README.md b/README.md",
		"--- a/README.md",
		"+++ b/README.md",
		"@@ -1,0 +2 @@",
		"+Example: // @doc docs/A.md",
	}, "\n")

	deltas := annotationDeltasByFile(diff, "@doc")

	// move.go: equal multisets for docs/A.md -> pure move.
	mv := deltas["move.go"]
	if mv == nil || !equalMultiset(mv.removed["docs/A.md"], mv.added["docs/A.md"]) {
		t.Errorf("move.go should have equal removed/added for docs/A.md: %+v", mv)
	}
	// rescope.go: unequal scope multisets.
	rs := deltas["rescope.go"]
	if rs == nil || equalMultiset(rs.removed["docs/A.md"], rs.added["docs/A.md"]) {
		t.Errorf("rescope.go should differ for docs/A.md: %+v", rs)
	}
	// mixed.go: annotation inside a code hunk.
	mx := deltas["mixed.go"]
	if mx == nil || !mx.mixed["docs/B.md"] {
		t.Errorf("mixed.go should mark docs/B.md as mixed: %+v", mx)
	}
	// Markdown blocks are skipped (docs quote example annotations).
	if _, ok := deltas["README.md"]; ok {
		t.Error("markdown file blocks must not contribute deltas")
	}
}

// writeStructuralConfig declares docs/API.md as gate: structural in dir.
func writeStructuralConfig(t *testing.T, dir string) {
	t.Helper()
	os.WriteFile(filepath.Join(dir, ".docdiff.yaml"), []byte(`docs:
  docs/API.md:
    gate: structural
`), 0644)
}

func runCheckForTest(t *testing.T, staged bool) (string, error) {
	t.Helper()
	checkStaged = staged
	checkJSON = false
	checkNoBacklinks = true
	checkFiles = nil
	t.Cleanup(func() {
		checkStaged = false
		checkNoBacklinks = false
		checkFiles = nil
	})
	var stdout bytes.Buffer
	checkCmd.SetOut(&stdout)
	err := checkCmd.RunE(checkCmd, nil)
	return stdout.String(), err
}

func TestCheck_StructuralGate_BodyEditDoesNotGate(t *testing.T) {
	dir := setupTestProject(t)
	writeStructuralConfig(t, dir)
	commitAll(t, dir, "Declare structural policy")

	os.WriteFile(filepath.Join(dir, "src", "handler.go"), []byte(`package main

// @doc docs/API.md
func Handler() { bodyEdit() }
`), 0644)

	initTestEnv(t, dir)
	out, err := runCheckForTest(t, false)
	if err != nil {
		t.Fatalf("body edit must not gate a structural doc, got %v:\n%s", err, out)
	}
	if strings.Contains(out, "docs/API.md") {
		t.Errorf("structural doc should not appear for a body edit, got:\n%s", out)
	}
}

func TestCheck_StructuralGate_AnnotationAddGates(t *testing.T) {
	dir := setupTestProject(t)
	writeStructuralConfig(t, dir)
	commitAll(t, dir, "Declare structural policy")

	// Add an annotation-only change: util.go now links docs/API.md.
	os.WriteFile(filepath.Join(dir, "src", "util.go"), []byte(`package main

// @doc docs/API.md
func Util() {}
`), 0644)

	initTestEnv(t, dir)
	out, err := runCheckForTest(t, false)
	if err == nil {
		t.Fatalf("annotation add must gate a structural doc, got:\n%s", out)
	}
	if !strings.Contains(out, "docs/API.md: needs update") {
		t.Errorf("expected structural doc required, got:\n%s", out)
	}
	if !strings.Contains(out, "annotation for docs/API.md added in src/util.go") {
		t.Errorf("expected structural reason line, got:\n%s", out)
	}
}

func TestCheck_StructuralGate_PureMoveDoesNotGate(t *testing.T) {
	dir := setupTestProject(t)
	writeStructuralConfig(t, dir)

	os.WriteFile(filepath.Join(dir, "src", "handler.go"), []byte(`package main

// @doc docs/API.md
func Handler() {}

func Helper() {}
`), 0644)
	commitAll(t, dir, "Baseline with annotation at top")

	// Move the annotation line further down, byte-identical otherwise.
	os.WriteFile(filepath.Join(dir, "src", "handler.go"), []byte(`package main

func Handler() {}

// @doc docs/API.md
func Helper() {}
`), 0644)

	initTestEnv(t, dir)
	out, err := runCheckForTest(t, false)
	if err != nil {
		t.Fatalf("pure annotation move must not gate, got %v:\n%s", err, out)
	}
	if strings.Contains(out, "docs/API.md: needs update") {
		t.Errorf("pure move should not require an update, got:\n%s", out)
	}
}

func TestCheck_StructuralGate_ScopeRenameGates(t *testing.T) {
	dir := setupTestProject(t)
	writeStructuralConfig(t, dir)

	os.WriteFile(filepath.Join(dir, "src", "handler.go"), []byte(`package main

// @doc docs/API.md #alpha
func Handler() {}
`), 0644)
	commitAll(t, dir, "Baseline with scoped annotation")

	os.WriteFile(filepath.Join(dir, "src", "handler.go"), []byte(`package main

// @doc docs/API.md #beta
func Handler() {}
`), 0644)

	initTestEnv(t, dir)
	out, err := runCheckForTest(t, false)
	if err == nil {
		t.Fatalf("scope rename must gate a structural doc, got:\n%s", out)
	}
	if !strings.Contains(out, "annotation for docs/API.md re-scoped in src/handler.go") {
		t.Errorf("expected re-scope reason, got:\n%s", out)
	}
}

func TestCheck_StructuralGate_RenameGates(t *testing.T) {
	dir := setupTestProject(t)
	writeStructuralConfig(t, dir)
	commitAll(t, dir, "Declare structural policy")

	runGit(t, dir, "mv", "src/handler.go", "src/renamed.go")

	initTestEnv(t, dir)
	out, err := runCheckForTest(t, true) // staged: git mv stages the rename
	if err == nil {
		t.Fatalf("linked-file rename must gate a structural doc, got:\n%s", out)
	}
	if !strings.Contains(out, "linked file renamed: src/handler.go -> src/renamed.go") {
		t.Errorf("expected rename reason, got:\n%s", out)
	}
}

func TestReport_StructuralPolicy(t *testing.T) {
	dir := setupTestProject(t)
	writeStructuralConfig(t, dir)
	commitAll(t, dir, "Declare structural policy")

	// Committed body edit: NOT stale under structural polarity.
	os.WriteFile(filepath.Join(dir, "src", "handler.go"), []byte(`package main

// @doc docs/API.md
func Handler() { bodyEdit() }
`), 0644)
	commitAll(t, dir, "Body edit only")

	initTestEnv(t, dir)
	reportStale = false
	reportOrphaned = false

	var stdout bytes.Buffer
	reportCmd.SetOut(&stdout)
	reportCmd.RunE(reportCmd, nil)
	if strings.Contains(stdout.String(), "STALE DOCS") {
		t.Fatalf("body edit should not stale a structural doc, got:\n%s", stdout.String())
	}

	// Committed annotation removal: stale.
	os.WriteFile(filepath.Join(dir, "src", "handler.go"), []byte(`package main

func Handler() { bodyEdit() }
`), 0644)
	commitAll(t, dir, "Drop the annotation")

	stdout.Reset()
	reportCmd.RunE(reportCmd, nil)
	if !strings.Contains(stdout.String(), "STALE DOCS") || !strings.Contains(stdout.String(), "docs/API.md") {
		t.Errorf("annotation removal should stale a structural doc, got:\n%s", stdout.String())
	}
}
