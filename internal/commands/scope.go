package commands

import (
	"github.com/StevenBock/docdiff/internal/git"
	"github.com/StevenBock/docdiff/internal/language"
	"github.com/StevenBock/docdiff/internal/scanner"
)

// fileHitsDoc reports whether changed file (via ann) should flag doc, honoring
// scoped annotations. An unscoped `@doc` = whole-file ownership: any change
// hits. A scoped `@doc X #s` owns [its line, next annotation) and is hit only
// when a changed hunk overlaps that region. When hunk info is unavailable — an
// untracked/new file, or --files mode — it falls back to whole-file so a real
// change is never missed.
func fileHitsDoc(ann *scanner.Annotation, doc string, hunks map[string][]git.LineRange) bool {
	if ann == nil {
		return true
	}
	var scoped []language.DocAnnotation
	for _, d := range ann.Details {
		if d.Path != doc {
			continue
		}
		if d.Scope == "" {
			return true // whole-file ownership
		}
		scoped = append(scoped, d)
	}
	if len(scoped) == 0 {
		return true
	}
	ranges, ok := hunks[ann.FilePath]
	if !ok {
		return true // no hunk info; don't narrow
	}
	for _, s := range scoped {
		start, end := ownedRegion(ann.Details, s.Line)
		for _, h := range ranges {
			if h.Start <= end && h.End >= start {
				return true
			}
		}
	}
	return false
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
