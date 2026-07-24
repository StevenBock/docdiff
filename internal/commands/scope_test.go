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

	if hit, _ := fileHitsDoc(ann, "docs/GENERAL.md", hunks); hit {
		t.Error("GENERAL should NOT be hit by a change at line 6")
	}
	hit, hits := fileHitsDoc(ann, "docs/MOBILE.md", hunks)
	if !hit {
		t.Error("MOBILE should be hit by a change at line 6")
	}
	// The overlapping hunk is returned as provenance.
	if len(hits) != 1 || hits[0].Scope != "mobile" || hits[0].AnnLine != 5 || hits[0].Start != 6 || hits[0].End != 6 {
		t.Errorf("scope hits = %+v, want one #mobile hit at lines 6-6", hits)
	}

	// No hunk info for the file (untracked/new or --files): fall back to hit.
	if hit, _ := fileHitsDoc(ann, "docs/GENERAL.md", map[string][]git.LineRange{}); !hit {
		t.Error("missing hunk info must fall back to whole-file (hit)")
	}

	// An unscoped annotation owns the whole file regardless of hunks.
	bare := &scanner.Annotation{
		FilePath: "plain.go",
		Details:  []language.DocAnnotation{{Path: "docs/PLAIN.md", Line: 3}},
	}
	if hit, _ := fileHitsDoc(bare, "docs/PLAIN.md", map[string][]git.LineRange{"plain.go": {{Start: 99, End: 99}}}); !hit {
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

// The Vue mixed-comment-style bug delivered Details out of line order. Lint's
// scope-to-eof read the wrong annotation as last; region computation did not,
// because ownedRegion scans for the nearest following line rather than trusting
// position. extractDetailed now sorts, but this pins the property so region
// ownership can never acquire an ordering dependency by accident.
func TestOwnedRegion_IndependentOfDetailOrder(t *testing.T) {
	inOrder := []language.DocAnnotation{
		{Path: "docs/A.md", Scope: "script", Line: 2},
		{Path: "docs/B.md", Scope: "mid", Line: 606},
		{Path: "docs/C.md", Scope: "tail", Line: 908},
	}
	// Grouped by comment style, as the parser used to emit for a .vue file.
	shuffled := []language.DocAnnotation{inOrder[1], inOrder[2], inOrder[0]}

	for _, line := range []int{2, 606, 908} {
		wantStart, wantEnd := ownedRegion(inOrder, line)
		gotStart, gotEnd := ownedRegion(shuffled, line)
		if wantStart != gotStart || wantEnd != gotEnd {
			t.Errorf("line %d: ordered [%d,%d] vs shuffled [%d,%d] — region must not depend on slice order",
				line, wantStart, wantEnd, gotStart, gotEnd)
		}
	}
	if _, end := ownedRegion(inOrder, 2); end != 605 {
		t.Errorf("the line-2 region must stop at the next annotation, got end=%d want 605", end)
	}
	if _, end := ownedRegion(inOrder, 908); end != eof {
		t.Errorf("the last annotation must own to EOF, got end=%d", end)
	}
}
