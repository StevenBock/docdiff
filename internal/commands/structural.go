package commands

// @doc CLAUDE.md

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/StevenBock/docdiff/internal/git"
	"github.com/StevenBock/docdiff/internal/language"
)

// fileAnnDelta captures how one changed file's annotation lines moved, per
// target doc: scope multisets removed/added in annotation-only hunks, plus a
// flag for annotation lines that changed inside hunks that also change code.
type fileAnnDelta struct {
	removed map[string][]string // doc -> scopes of removed annotation lines
	added   map[string][]string // doc -> scopes of added annotation lines
	mixed   map[string]bool     // doc -> its annotation changed alongside code
}

// annotationDeltasByFile parses a unified diff and extracts, per changed file,
// the annotation lines added/removed for each target doc. Deltas from
// annotation-only hunks feed the pure-move exemption (equal multisets of
// (path, scope) = a move); an annotation line inside a mixed hunk marks the
// doc as conservatively changed.
func annotationDeltasByFile(diff, tag string) map[string]*fileAnnDelta {
	out := make(map[string]*fileAnnDelta)
	if tag == "" || !strings.Contains(diff, "diff --git ") {
		return out
	}
	tagPattern := regexp.MustCompile(regexp.QuoteMeta(tag) + `\s+(\S+)(?:\s+#(\S+))?`)

	_, blocks := splitDiffBlocks(diff)
	for _, block := range blocks {
		file := diffBlockFile(block)
		if file == "" {
			continue
		}
		// ponytail: docs quote example annotation lines; annotations only live
		// in source comments, so markdown blocks are skipped wholesale. Files
		// excluded from scanning can still contribute deltas — narrow only if
		// that false-gates a structural doc in practice.
		if ext := strings.ToLower(file); strings.HasSuffix(ext, ".md") || strings.HasSuffix(ext, ".markdown") {
			continue
		}
		_, hunks := splitBlockHunks(block)
		for _, h := range hunks {
			annOnly := annotationOnlyHunk(h, tag)
			for _, ln := range h {
				if strings.HasPrefix(ln, "+++") || strings.HasPrefix(ln, "---") {
					continue
				}
				var side byte
				switch {
				case strings.HasPrefix(ln, "+"):
					side = '+'
				case strings.HasPrefix(ln, "-"):
					side = '-'
				default:
					continue
				}
				for _, m := range tagPattern.FindAllStringSubmatch(ln, -1) {
					doc := m[1]
					if doc == language.Terminator {
						continue
					}
					d := out[file]
					if d == nil {
						d = &fileAnnDelta{
							removed: make(map[string][]string),
							added:   make(map[string][]string),
							mixed:   make(map[string]bool),
						}
						out[file] = d
					}
					switch {
					case !annOnly:
						d.mixed[doc] = true
					case side == '+':
						d.added[doc] = append(d.added[doc], m[2])
					default:
						d.removed[doc] = append(d.removed[doc], m[2])
					}
				}
			}
		}
	}
	return out
}

func equalMultiset(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	as := append([]string{}, a...)
	bs := append([]string{}, b...)
	sort.Strings(as)
	sort.Strings(bs)
	for i := range as {
		if as[i] != bs[i] {
			return false
		}
	}
	return true
}

// structuralTriggers evaluates a `gate: structural` doc against a change set:
// annotation-map changes (except pure moves) and linked-file add/rename gate;
// body edits never do. linkedNow reports whether a file currently links doc.
// Deleted linked files gate via their removed annotation lines in the diff.
func structuralTriggers(doc string, deltas map[string]*fileAnnDelta, nameStatus []git.NameStatus, untracked []string, linkedNow func(string) bool) (files, reasons []string) {
	seen := make(map[string]bool)
	add := func(file, reason string) {
		if !seen[file+"\x00"+reason] {
			seen[file+"\x00"+reason] = true
			files = append(files, file)
			reasons = append(reasons, reason)
		}
	}

	deltaFiles := make([]string, 0, len(deltas))
	for f := range deltas {
		deltaFiles = append(deltaFiles, f)
	}
	sort.Strings(deltaFiles)
	for _, f := range deltaFiles {
		d := deltas[f]
		if d.mixed[doc] {
			add(f, fmt.Sprintf("annotation for %s changed alongside code in %s", doc, f))
			continue
		}
		removed, added := d.removed[doc], d.added[doc]
		if len(removed) == 0 && len(added) == 0 {
			continue
		}
		if equalMultiset(removed, added) {
			continue // pure move within the file: the map is unchanged
		}
		switch {
		case len(removed) == 0:
			add(f, fmt.Sprintf("annotation for %s added in %s", doc, f))
		case len(added) == 0:
			add(f, fmt.Sprintf("annotation for %s removed from %s", doc, f))
		default:
			add(f, fmt.Sprintf("annotation for %s re-scoped in %s", doc, f))
		}
	}

	for _, ns := range nameStatus {
		switch ns.Status {
		case 'A', 'C':
			if linkedNow(ns.Path) {
				add(ns.Path, fmt.Sprintf("linked file added: %s", ns.Path))
			}
		case 'R':
			if linkedNow(ns.Path) {
				add(ns.Path, fmt.Sprintf("linked file renamed: %s -> %s", ns.OldPath, ns.Path))
			}
		}
	}

	for _, f := range untracked {
		if linkedNow(f) {
			add(f, fmt.Sprintf("linked file added: %s", f))
		}
	}

	return files, reasons
}

// uniqueSorted de-duplicates and sorts a string slice.
func uniqueSorted(in []string) []string {
	seen := make(map[string]bool, len(in))
	out := make([]string, 0, len(in))
	for _, s := range in {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	sort.Strings(out)
	return out
}
