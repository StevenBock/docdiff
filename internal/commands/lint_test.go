package commands

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/StevenBock/docdiff/internal/language"
	"github.com/StevenBock/docdiff/internal/scanner"
)

func TestLintFindings(t *testing.T) {
	res := scanner.NewResult()

	// scope-to-eof: only annotation, scoped.
	res.AddAnnotation("src/eof.go", []language.DocAnnotation{
		{Path: "docs/A.md", Scope: "top", Line: 2},
	}, "go")

	// NOT scope-to-eof: region closed by a terminator.
	res.AddAnnotation("src/closed.go", []language.DocAnnotation{
		{Path: "docs/A.md", Scope: "top", Line: 2},
		{Path: language.Terminator, Line: 8},
	}, "go")

	// mixed-ownership: docs/B.md linked whole-file AND scoped.
	res.AddAnnotation("src/mixed.go", []language.DocAnnotation{
		{Path: "docs/B.md", Line: 2},
		{Path: "docs/B.md", Scope: "part", Line: 6},
	}, "go")

	// fan-out: three distinct whole-file links.
	res.AddAnnotation("src/fan.go", []language.DocAnnotation{
		{Path: "docs/A.md", Line: 2},
		{Path: "docs/B.md", Line: 3},
		{Path: "docs/C.md", Line: 4},
	}, "go")

	// vestigial-link: docs/A.md was read and mentions closed.go but not eof.go.
	res.DocRefs["docs/A.md"] = map[string]bool{"src/closed.go": true}

	findings := lintFindings(res, "@doc", 3)

	byRule := make(map[string][]lintFinding)
	for _, f := range findings {
		byRule[f.Rule] = append(byRule[f.Rule], f)
	}

	if got := byRule["scope-to-eof"]; len(got) != 1 || got[0].File != "src/eof.go" {
		t.Errorf("scope-to-eof = %+v, want exactly src/eof.go", got)
	}
	if got := byRule["mixed-ownership"]; len(got) != 1 || got[0].File != "src/mixed.go" || got[0].Doc != "docs/B.md" {
		t.Errorf("mixed-ownership = %+v, want src/mixed.go for docs/B.md", got)
	}
	if got := byRule["fan-out"]; len(got) != 1 || got[0].File != "src/fan.go" {
		t.Errorf("fan-out = %+v, want exactly src/fan.go", got)
	}
	for _, f := range byRule["vestigial-link"] {
		if !f.Advisory {
			t.Errorf("vestigial-link must be advisory: %+v", f)
		}
	}
	// eof.go and fan.go link docs/A.md without being mentioned by it;
	// closed.go is mentioned. mixed.go links docs/B.md which was never read.
	vest := make(map[string]bool)
	for _, f := range byRule["vestigial-link"] {
		vest[f.File] = true
	}
	if !vest["src/eof.go"] || !vest["src/fan.go"] || vest["src/closed.go"] || vest["src/mixed.go"] {
		t.Errorf("vestigial-link files = %v, want eof.go and fan.go only", vest)
	}

	// Higher threshold silences fan-out.
	for _, f := range lintFindings(res, "@doc", 4) {
		if f.Rule == "fan-out" {
			t.Errorf("threshold 4 should silence a 3-doc fan-out: %+v", f)
		}
	}
}

func TestLintFindings_ScopeDrift(t *testing.T) {
	res := scanner.NewResult()

	// Code scopes for docs/A.md: #kept and #renamed.
	res.AddAnnotation("src/a.go", []language.DocAnnotation{
		{Path: "docs/A.md", Scope: "kept", Line: 2},
		{Path: "docs/A.md", Scope: "renamed", Line: 8},
	}, "go")
	// Doc marks #kept (match) and #gone (no code scope); #renamed is unmarked.
	res.DocScopeMarkers["docs/A.md"] = []string{"kept", "gone"}

	// A doc with code scopes but NO markers: never checked (opt-in).
	res.AddAnnotation("src/b.go", []language.DocAnnotation{
		{Path: "docs/B.md", Scope: "whatever", Line: 2},
	}, "go")
	res.DocScopeMarkers["docs/B.md"] = []string{}

	var drift []lintFinding
	for _, f := range lintFindings(res, "@doc", 3) {
		if f.Rule == "scope-drift" {
			drift = append(drift, f)
		}
	}
	if len(drift) != 2 {
		t.Fatalf("scope-drift = %+v, want exactly 2 findings (marker #gone, code #renamed)", drift)
	}
	msgs := drift[0].Message + "\n" + drift[1].Message
	if !strings.Contains(msgs, "#gone") || !strings.Contains(msgs, "#renamed") {
		t.Errorf("expected findings for #gone and #renamed, got:\n%s", msgs)
	}
	for _, f := range drift {
		if f.Advisory {
			t.Errorf("scope-drift should be a problem, not advisory: %+v", f)
		}
		if strings.Contains(f.Message, "#kept") {
			t.Errorf("matched scope #kept must not be flagged: %+v", f)
		}
	}
}

func TestScopeDrift_Integration(t *testing.T) {
	dir := setupTestProject(t)

	os.WriteFile(filepath.Join(dir, "docs", "API.md"), []byte(`# API Docs

<!-- @scope alpha -->
Alpha section.

<!-- @scope ghost -->
Ghost section.
`), 0644)
	os.WriteFile(filepath.Join(dir, "src", "handler.go"), []byte(`package main

// @doc docs/API.md #alpha
func Handler() {}

// @doc -
func Helper() {}
`), 0644)
	commitAll(t, dir, "Add scoped doc and code")

	initTestEnv(t, dir)
	lintJSON = false

	var stdout bytes.Buffer
	lintCmd.SetOut(&stdout)
	err := lintCmd.RunE(lintCmd, nil)
	if err == nil {
		t.Fatalf("ghost marker should be a lint problem, got:\n%s", stdout.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "[scope-drift]") || !strings.Contains(out, "#ghost") {
		t.Errorf("expected scope-drift for #ghost, got:\n%s", out)
	}
	if strings.Contains(out, "#alpha") {
		t.Errorf("matched scope #alpha must not drift, got:\n%s", out)
	}
}

func TestLint_Integration(t *testing.T) {
	dir := setupTestProject(t)

	// A linked doc OUTSIDE the docs directory that mentions its file: must be
	// read as content, so the link is not reported vestigial.
	os.WriteFile(filepath.Join(dir, "NOTES.md"), []byte("# Notes\n\nSee src/noted.go for details.\n"), 0644)
	os.WriteFile(filepath.Join(dir, "src", "noted.go"), []byte(`package main

// @doc NOTES.md
func Noted() {}
`), 0644)

	// A problem for the exit code: only-annotation-is-scoped.
	os.WriteFile(filepath.Join(dir, "src", "eof.go"), []byte(`package main

// @doc docs/API.md #eof
func Eof() {}
`), 0644)
	commitAll(t, dir, "Add lint fixtures")

	initTestEnv(t, dir)
	lintJSON = false

	var stdout bytes.Buffer
	lintCmd.SetOut(&stdout)
	err := lintCmd.RunE(lintCmd, nil)
	if err == nil {
		t.Fatalf("lint should exit non-zero on problems, got:\n%s", stdout.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "[scope-to-eof]") || !strings.Contains(out, "src/eof.go:3") {
		t.Errorf("expected scope-to-eof problem for src/eof.go, got:\n%s", out)
	}
	if !strings.Contains(out, "'@doc -'") {
		t.Errorf("scope-to-eof fix hint should name the terminator, got:\n%s", out)
	}
	if strings.Contains(out, "NOTES.md never mentions src/noted.go") {
		t.Errorf("doc outside docs dir was not read as content:\n%s", out)
	}
	// docs/API.md exists but never mentions its linked files: advisory only.
	if !strings.Contains(out, "[vestigial-link]") {
		t.Errorf("expected vestigial-link advisories, got:\n%s", out)
	}
}
