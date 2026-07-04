package commands

import (
	"testing"

	"github.com/StevenBock/docdiff/internal/git"
	"github.com/StevenBock/docdiff/internal/language"
	"github.com/StevenBock/docdiff/internal/scanner"
)

func TestFileHitsDoc(t *testing.T) {
	// central.go: #general owns [2,4], #mobile owns [5,EOF].
	ann := &scanner.Annotation{
		FilePath: "central.go",
		Details: []language.DocAnnotation{
			{Path: "docs/GENERAL.md", Scope: "general", Line: 2},
			{Path: "docs/MOBILE.md", Scope: "mobile", Line: 5},
		},
	}
	hunks := map[string][]git.LineRange{"central.go": {{Start: 6, End: 6}}} // change in mobile region

	if fileHitsDoc(ann, "docs/GENERAL.md", hunks) {
		t.Error("GENERAL should NOT be hit by a change at line 6")
	}
	if !fileHitsDoc(ann, "docs/MOBILE.md", hunks) {
		t.Error("MOBILE should be hit by a change at line 6")
	}

	// No hunk info for the file (untracked/new or --files): fall back to hit.
	if !fileHitsDoc(ann, "docs/GENERAL.md", map[string][]git.LineRange{}) {
		t.Error("missing hunk info must fall back to whole-file (hit)")
	}

	// An unscoped annotation owns the whole file regardless of hunks.
	bare := &scanner.Annotation{
		FilePath: "plain.go",
		Details:  []language.DocAnnotation{{Path: "docs/PLAIN.md", Line: 3}},
	}
	if !fileHitsDoc(bare, "docs/PLAIN.md", map[string][]git.LineRange{"plain.go": {{Start: 99, End: 99}}}) {
		t.Error("unscoped annotation should be hit by any change")
	}
}

func TestOwnedRegion(t *testing.T) {
	details := []language.DocAnnotation{{Line: 2}, {Line: 5}, {Line: 5}}
	if s, e := ownedRegion(details, 2); s != 2 || e != 4 {
		t.Errorf("region for line 2 = [%d,%d], want [2,4]", s, e)
	}
	if s, e := ownedRegion(details, 5); s != 5 || e != eof {
		t.Errorf("region for last line 5 = [%d,%d], want [5,eof]", s, e)
	}
}
