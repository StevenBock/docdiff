package commands

// @doc CLAUDE.md

import (
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
