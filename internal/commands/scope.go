package commands

// @doc CLAUDE.md

import (
	"regexp"
	"strings"

	"github.com/StevenBock/docdiff/internal/git"
	"github.com/StevenBock/docdiff/internal/language"
	"github.com/StevenBock/docdiff/internal/scanner"
)

// scopeHit records a changed hunk overlapping a scoped annotation's owned
// region: provenance for why a scoped link flagged its doc.
type scopeHit struct {
	Scope   string // scope name of the annotation that was hit
	AnnLine int    // line the annotation sits on (matches DocAnnotation.Line)
	Start   int    // overlap of the hunk with the owned region, inclusive
	End     int
}

// fileHitsDoc reports whether changed file (via ann) should flag doc, honoring
// scoped annotations. An unscoped annotation = whole-file ownership: any change
// hits. A scoped one (`#name` suffix) owns [its line, next annotation) and is hit
// only when a changed hunk overlaps that region — those overlaps are returned as
// scope hits so check can show WHERE the region was touched. When hunk info is
// unavailable — an untracked/new file, or --files mode — it falls back to
// whole-file so a real change is never missed.
func fileHitsDoc(ann *scanner.Annotation, doc string, hunks map[string][]git.LineRange) (bool, []scopeHit) {
	if ann == nil {
		return true, nil
	}
	var scoped []language.DocAnnotation
	for _, d := range ann.Details {
		if d.Path != doc {
			continue
		}
		if d.Scope == "" {
			return true, nil // whole-file ownership
		}
		scoped = append(scoped, d)
	}
	if len(scoped) == 0 {
		return true, nil
	}
	ranges, ok := hunks[ann.FilePath]
	if !ok {
		return true, nil // no hunk info; don't narrow
	}
	var hits []scopeHit
	for _, s := range scoped {
		start, end := ownedRegion(ann.Details, s.Line)
		for _, h := range ranges {
			if h.Start <= end && h.End >= start {
				hits = append(hits, scopeHit{
					Scope:   s.Scope,
					AnnLine: s.Line,
					Start:   max(h.Start, start),
					End:     min(h.End, end),
				})
			}
		}
	}
	return len(hits) > 0, hits
}

const eof = 1 << 30

// ownedRegion is the inclusive [start,end] line region an annotation at `line`
// owns: from its line to just before the next annotation in the file (any doc),
// or EOF for the last annotation.
func ownedRegion(details []language.DocAnnotation, line int) (int, int) {
	end := eof
	for _, d := range details {
		if d.Line > line && d.Line-1 < end {
			end = d.Line - 1
		}
	}
	return line, end
}

// regionOwnsCodeAbove reports whether any annotation preceding annLine owns
// substantive content — a line that is non-blank and not wholly comment.
//
// This separates two structurally identical shapes that mean different things:
// a scope deliberately covering its whole file, and a scope written at the top
// and never bounded. It cannot tell them apart on its own — only reading what
// the scope names against what the code does can — so it is used to pick which
// finding to raise, never to suppress one.
//
// Content above the file's FIRST annotation belongs to no region, so a `<?php`,
// `<script setup>`, shebang or import preamble is excluded without naming any
// of them.
func regionOwnsCodeAbove(content []byte, patterns []*regexp.Regexp, details []language.DocAnnotation, annLine int) bool {
	mask := make([]bool, len(content))
	for _, p := range patterns {
		for _, loc := range p.FindAllIndex(content, -1) {
			for i := loc[0]; i < loc[1] && i < len(mask); i++ {
				mask[i] = true
			}
		}
	}

	substantive := make(map[int]bool)
	off, lineNo := 0, 1
	for _, line := range strings.SplitAfter(string(content), "\n") {
		for i := 0; i < len(line); i++ {
			if b := line[i]; !mask[off+i] && b != ' ' && b != '\t' && b != '\n' && b != '\r' {
				substantive[lineNo] = true
				break
			}
		}
		off += len(line)
		lineNo++
	}

	for _, d := range details {
		if d.Line >= annLine {
			continue
		}
		start, end := ownedRegion(details, d.Line)
		if end > annLine-1 {
			end = annLine - 1
		}
		for n := start; n <= end; n++ {
			if substantive[n] {
				return true
			}
		}
	}
	return false
}

// scopedChangedFiles narrows a doc's changed-file list to the files whose
// changed hunks actually land in a region that doc owns, over a COMMIT RANGE
// rather than the working tree. Without it, scoped annotations protect against
// staleness in `check` but not in `report`/`explain`/`graph` — so a central file
// carrying 30 well-placed scopes flags every doc that touches it, which is the
// opposite of what scoping is for.
//
// Conservative in every failure direction: a git error, a file with no hunk
// data (added, deleted, renamed), or a whole-file annotation all keep the file.
func scopedChangedFiles(g *git.Git, doc, from, to string, files []string, anns map[string]*scanner.Annotation, tag string) []string {
	if len(files) == 0 {
		return files
	}
	hunks, err := g.ChangedHunksBetween(from, to, files, tag)
	if err != nil {
		return files
	}
	kept := make([]string, 0, len(files))
	for _, f := range files {
		if hit, _ := fileHitsDoc(anns[f], doc, hunks); hit {
			kept = append(kept, f)
		}
	}
	return kept
}
