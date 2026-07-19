package commands

import (
	"strconv"
	"strings"
)

// maybeHideAnnotations strips annotation-only hunks from a unified diff when the
// --hide-annotations flag is set, so behaviorally relevant changes aren't buried
// under broad annotation-tag noise.
func maybeHideAnnotations(diff string) string {
	if !changesHideAnnot {
		return diff
	}
	return filterAnnotationDiff(diff, cfg.AnnotationTag)
}

// filterAnnotationDiff drops file blocks/hunks from a `git diff`-style unified
// diff when every changed (+/-) line in the hunk contains the annotation tag.
// A hunk that mixes annotation and real code changes is kept untouched.
//
// ponytail: line-level heuristic — a single line that edits both code and an
// annotation counts as a real change and is kept. Good enough; upgrade to a
// token-level diff only if mixed-line noise actually shows up.
func filterAnnotationDiff(diff, tag string) string {
	if tag == "" || !strings.Contains(diff, "diff --git ") {
		return diff
	}

	preamble, blocks := splitDiffBlocks(diff)

	var out []string
	out = append(out, preamble...)
	for _, block := range blocks {
		if kept := filterBlock(block, tag); kept != nil {
			out = append(out, kept...)
		}
	}
	return strings.Join(out, "\n")
}

// splitDiffBlocks splits a unified diff into any preamble lines and one block
// of lines per "diff --git" file section.
func splitDiffBlocks(diff string) (preamble []string, blocks [][]string) {
	var cur []string
	inBlock := false
	for _, ln := range strings.Split(diff, "\n") {
		if strings.HasPrefix(ln, "diff --git ") {
			if inBlock {
				blocks = append(blocks, cur)
			}
			cur = []string{ln}
			inBlock = true
			continue
		}
		if inBlock {
			cur = append(cur, ln)
		} else {
			preamble = append(preamble, ln)
		}
	}
	if inBlock {
		blocks = append(blocks, cur)
	}
	return preamble, blocks
}

// annotationOnlyDiffFiles returns the files in a unified diff whose every hunk
// is annotation-only. Files with any real hunk, or with no hunks at all
// (rename/mode-only blocks), are not included — they stay conservative.
func annotationOnlyDiffFiles(diff, tag string) map[string]bool {
	out := make(map[string]bool)
	if tag == "" || !strings.Contains(diff, "diff --git ") {
		return out
	}
	_, blocks := splitDiffBlocks(diff)
	for _, block := range blocks {
		if !blockHasHunks(block) {
			continue
		}
		if filterBlock(block, tag) == nil {
			if f := diffBlockFile(block); f != "" {
				out[f] = true
			}
		}
	}
	return out
}

func blockHasHunks(block []string) bool {
	for _, ln := range block {
		if strings.HasPrefix(ln, "@@") {
			return true
		}
	}
	return false
}

// diffBlockFile extracts the file path a block describes: the new-side path,
// or the old-side path for deletions. Handles git's quoted-path form.
func diffBlockFile(block []string) string {
	name := func(ln, marker, side string) string {
		p := strings.TrimPrefix(ln, marker)
		if strings.HasPrefix(p, `"`) {
			if uq, err := strconv.Unquote(p); err == nil {
				p = uq
			}
		}
		return strings.TrimPrefix(p, side)
	}
	old := ""
	for _, ln := range block {
		switch {
		case strings.HasPrefix(ln, "+++ "):
			if p := name(ln, "+++ ", "b/"); p != "/dev/null" {
				return p
			}
		case strings.HasPrefix(ln, "--- ") && old == "":
			if p := name(ln, "--- ", "a/"); p != "/dev/null" {
				old = p
			}
		case strings.HasPrefix(ln, "@@"):
			return old
		}
	}
	return old
}

// splitBlockHunks splits one file block into its header lines and one slice of
// lines per "@@" hunk. hunks is nil for hunk-less blocks (rename/mode change).
func splitBlockHunks(block []string) (header []string, hunks [][]string) {
	firstHunk := -1
	for i, ln := range block {
		if strings.HasPrefix(ln, "@@") {
			firstHunk = i
			break
		}
	}
	if firstHunk == -1 {
		return block, nil
	}

	header = block[:firstHunk]
	var cur []string
	for _, ln := range block[firstHunk:] {
		if strings.HasPrefix(ln, "@@") {
			if len(cur) > 0 {
				hunks = append(hunks, cur)
			}
			cur = []string{ln}
		} else {
			cur = append(cur, ln)
		}
	}
	if len(cur) > 0 {
		hunks = append(hunks, cur)
	}
	return header, hunks
}

// filterBlock returns the block with annotation-only hunks removed, or nil if
// every hunk was annotation-only (drop the whole file block).
func filterBlock(block []string, tag string) []string {
	header, hunks := splitBlockHunks(block)
	if hunks == nil {
		return block // no hunks (rename/mode change only) — keep
	}

	out := append([]string{}, header...)
	keptAny := false
	for _, h := range hunks {
		if !annotationOnlyHunk(h, tag) {
			out = append(out, h...)
			keptAny = true
		}
	}
	if !keptAny {
		return nil
	}
	return out
}

// annotationOnlyHunk reports whether a hunk has at least one changed line and
// every changed line contains the annotation tag.
func annotationOnlyHunk(hunk []string, tag string) bool {
	changed := 0
	for _, ln := range hunk {
		if strings.HasPrefix(ln, "+++") || strings.HasPrefix(ln, "---") {
			continue
		}
		if strings.HasPrefix(ln, "+") || strings.HasPrefix(ln, "-") {
			changed++
			if !strings.Contains(ln, tag) {
				return false
			}
		}
	}
	return changed > 0
}
