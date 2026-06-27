package commands

import "strings"

// maybeHideAnnotations strips annotation-only hunks from a unified diff when the
// --hide-annotations flag is set, so behaviorally relevant changes aren't buried
// under broad @doc annotation noise.
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

	lines := strings.Split(diff, "\n")
	var preamble [][]string // lines before the first block (rare)
	var blocks [][]string
	var cur []string
	inBlock := false
	for _, ln := range lines {
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
			preamble = append(preamble, []string{ln})
		}
	}
	if inBlock {
		blocks = append(blocks, cur)
	}

	var out []string
	for _, p := range preamble {
		out = append(out, p...)
	}
	for _, block := range blocks {
		if kept := filterBlock(block, tag); kept != nil {
			out = append(out, kept...)
		}
	}
	return strings.Join(out, "\n")
}

// filterBlock returns the block with annotation-only hunks removed, or nil if
// every hunk was annotation-only (drop the whole file block).
func filterBlock(block []string, tag string) []string {
	firstHunk := -1
	for i, ln := range block {
		if strings.HasPrefix(ln, "@@") {
			firstHunk = i
			break
		}
	}
	if firstHunk == -1 {
		return block // no hunks (rename/mode change only) — keep
	}

	header := block[:firstHunk]
	var hunks [][]string
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
